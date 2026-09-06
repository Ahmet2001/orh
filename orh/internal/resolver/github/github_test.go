package github

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRef(t *testing.T) {
	tests := []struct {
		in   string
		want Ref
	}{
		{"rifat/debate-pro", Ref{Owner: "rifat", Repo: "debate-pro"}},
		{"rifat/debate-pro@v1.2.0", Ref{Owner: "rifat", Repo: "debate-pro", Version: "v1.2.0"}},
		{"github:rifat/debate-pro", Ref{Owner: "rifat", Repo: "debate-pro"}},
		{"github:rifat/debate-pro@a83bc91", Ref{Owner: "rifat", Repo: "debate-pro", Version: "a83bc91"}},
	}

	for _, tt := range tests {
		got, err := ParseRef(tt.in)
		if err != nil {
			t.Errorf("ParseRef(%q) error = %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRef(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestParseRef_Invalid(t *testing.T) {
	for _, in := range []string{"", "no-slash", "/missing-owner", "owner/"} {
		if _, err := ParseRef(in); err == nil {
			t.Errorf("ParseRef(%q) expected error, got nil", in)
		}
	}
}

// buildTarGz constructs an in-memory gzip-compressed tarball with a single
// top-level wrapper directory, mimicking what GitHub's codeload serves.
func buildTarGz(t *testing.T, wrapper string, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	if err := tw.WriteHeader(&tar.Header{Name: wrapper + "/", Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
		t.Fatalf("writing wrapper dir header: %v", err)
	}

	for name, content := range files {
		hdr := &tar.Header{
			Name:     wrapper + "/" + name,
			Typeflag: tar.TypeReg,
			Mode:     0o644,
			Size:     int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("writing header for %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("writing content for %s: %v", name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("closing tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("closing gzip writer: %v", err)
	}
	return buf.Bytes()
}

func TestExtractTarGz(t *testing.T) {
	data := buildTarGz(t, "debate-pro-main", map[string]string{
		"orh.yaml":          "apiVersion: orh/v1\n",
		"main.orh":          "apiVersion: orh/v1\n",
		"prompts/critic.md": "be critical",
	})

	dest := t.TempDir()
	if err := extractTarGz(bytes.NewReader(data), dest); err != nil {
		t.Fatalf("extractTarGz() error = %v", err)
	}

	for _, name := range []string{"orh.yaml", "main.orh", "prompts/critic.md"} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Errorf("expected extracted file %s: %v", name, err)
		}
	}
}

func TestExtractTarGz_RejectsPathTraversal(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	malicious := "evil-main/../../etc/passwd"
	content := "pwned"
	if err := tw.WriteHeader(&tar.Header{Name: malicious, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(content))}); err != nil {
		t.Fatalf("writing malicious header: %v", err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatalf("writing malicious content: %v", err)
	}
	tw.Close()
	gz.Close()

	dest := t.TempDir()
	err := extractTarGz(bytes.NewReader(buf.Bytes()), dest)
	if err == nil {
		t.Fatal("expected extractTarGz to reject a path-traversal entry, got nil error")
	}
	if !strings.Contains(err.Error(), "escapes destination") {
		t.Errorf("error = %v, want a path-traversal error", err)
	}
}

func TestClient_ResolveAndDownload(t *testing.T) {
	tarball := buildTarGz(t, "debate-pro-main", map[string]string{
		"orh.yaml": "apiVersion: orh/v1\n",
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/rifat/debate-pro", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"default_branch": "main"})
	})
	mux.HandleFunc("/repos/rifat/debate-pro/commits/main", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"sha": "a83bc91deadbeef"})
	})
	apiServer := httptest.NewServer(mux)
	defer apiServer.Close()

	codeloadMux := http.NewServeMux()
	codeloadMux.HandleFunc("/rifat/debate-pro/tar.gz/main", func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarball)
	})
	codeloadServer := httptest.NewServer(codeloadMux)
	defer codeloadServer.Close()

	client := &Client{
		APIBaseURL:      apiServer.URL,
		CodeloadBaseURL: codeloadServer.URL,
		HTTPClient:      http.DefaultClient,
	}

	ref := Ref{Owner: "rifat", Repo: "debate-pro"}
	resolved, err := client.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.GitRef != "main" || resolved.Commit != "a83bc91deadbeef" {
		t.Fatalf("Resolve() = %+v, want {GitRef: main, Commit: a83bc91deadbeef}", resolved)
	}

	dest := t.TempDir()
	if err := client.Download(context.Background(), ref.Owner, ref.Repo, resolved.GitRef, dest); err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "orh.yaml")); err != nil {
		t.Errorf("expected orh.yaml to be extracted: %v", err)
	}
}
