package resolver

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
	"testing"

	"github.com/pertevniyalai/orh/internal/lock"
	"github.com/pertevniyalai/orh/internal/resolver/github"
)

const testManifest = `
apiVersion: orh/v1
kind: orchestration
name: debate-pro
version: 1.0.0
entrypoint: main.orh
author:
  github: rifat
dependencies: []
`

const testArch = `
apiVersion: orh/v1
name: debate-pro
models:
  main:
    provider: ollama
    name: qwen3
components:
  assistant:
    type: agent
    model: main
    prompt: hi
connections:
  - from: input
    to: assistant.input
  - from: assistant.output
    to: output
`

func tarGz(t *testing.T, wrapper string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: wrapper + "/", Typeflag: tar.TypeDir, Mode: 0o755})
	for name, content := range files {
		tw.WriteHeader(&tar.Header{Name: wrapper + "/" + name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(content))})
		tw.Write([]byte(content))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

// fakeGitHub spins up an httptest server pair that serves a single fixed
// package version for owner/repo, plus a default-branch/commit resolution.
func fakeGitHub(t *testing.T, owner, repo, branch, commit string, files map[string]string) *github.Client {
	t.Helper()

	tarball := tarGz(t, repo+"-"+branch, files)

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/repos/"+owner+"/"+repo, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"default_branch": branch})
	})
	apiMux.HandleFunc("/repos/"+owner+"/"+repo+"/commits/"+branch, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"sha": commit})
	})
	apiServer := httptest.NewServer(apiMux)
	t.Cleanup(apiServer.Close)

	codeloadMux := http.NewServeMux()
	codeloadMux.HandleFunc("/"+owner+"/"+repo+"/tar.gz/"+branch, func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarball)
	})
	codeloadServer := httptest.NewServer(codeloadMux)
	t.Cleanup(codeloadServer.Close)

	return &github.Client{
		APIBaseURL:      apiServer.URL,
		CodeloadBaseURL: codeloadServer.URL,
		HTTPClient:      http.DefaultClient,
	}
}

func TestPull_DownloadsValidatesAndCaches(t *testing.T) {
	t.Setenv("ORH_HOME", t.TempDir())

	client := fakeGitHub(t, "rifat", "debate-pro", "main", "a83bc91", map[string]string{
		"orh.yaml": testManifest,
		"main.orh": testArch,
	})

	r := New(client)
	result, err := r.Pull(context.Background(), "rifat/debate-pro")
	if err != nil {
		t.Fatalf("Pull() error = %v", err)
	}
	if result.Cached {
		t.Error("expected first Pull() to not be served from cache")
	}
	if result.Commit != "a83bc91" {
		t.Errorf("Commit = %q, want %q", result.Commit, "a83bc91")
	}
	if result.Package.Manifest.Name != "debate-pro" {
		t.Errorf("Manifest.Name = %q, want %q", result.Package.Manifest.Name, "debate-pro")
	}

	result2, err := r.Pull(context.Background(), "rifat/debate-pro")
	if err != nil {
		t.Fatalf("second Pull() error = %v", err)
	}
	if !result2.Cached {
		t.Error("expected second Pull() to be served from cache")
	}
}

func TestPull_InvalidManifestFails(t *testing.T) {
	t.Setenv("ORH_HOME", t.TempDir())

	client := fakeGitHub(t, "rifat", "broken", "main", "deadbeef", map[string]string{
		"orh.yaml": "apiVersion: orh/v2\nkind: orchestration\nname: broken\nentrypoint: main.orh\n",
		"main.orh": testArch,
	})

	r := New(client)
	if _, err := r.Pull(context.Background(), "rifat/broken"); err == nil {
		t.Fatal("expected Pull() to fail for an unsupported apiVersion")
	}
}

func TestPull_ResolvesDependenciesAndWritesLock(t *testing.T) {
	t.Setenv("ORH_HOME", t.TempDir())

	depManifest := `
apiVersion: orh/v1
kind: orchestration
name: critic
version: 1.0.0
entrypoint: main.orh
dependencies: []
`

	mainManifest := `
apiVersion: orh/v1
kind: orchestration
name: debate-pro
version: 1.0.0
entrypoint: main.orh
dependencies:
  critic:
    source: github:rifat/critic
    version: v1.0.0
`

	// Both packages are served off the same fake GitHub host so a single
	// Resolver.Client can resolve either one.
	tarballMain := tarGz(t, "debate-pro-main", map[string]string{"orh.yaml": mainManifest, "main.orh": testArch})
	tarballDep := tarGz(t, "critic-v1.0.0", map[string]string{"orh.yaml": depManifest, "main.orh": testArch})

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/repos/rifat/debate-pro", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"default_branch": "main"})
	})
	apiMux.HandleFunc("/repos/rifat/debate-pro/commits/main", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"sha": "mainsha1"})
	})
	apiMux.HandleFunc("/repos/rifat/critic/commits/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"sha": "criticsha1"})
	})
	apiServer := httptest.NewServer(apiMux)
	defer apiServer.Close()

	codeloadMux := http.NewServeMux()
	codeloadMux.HandleFunc("/rifat/debate-pro/tar.gz/main", func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarballMain)
	})
	codeloadMux.HandleFunc("/rifat/critic/tar.gz/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarballDep)
	})
	codeloadServer := httptest.NewServer(codeloadMux)
	defer codeloadServer.Close()

	client := &github.Client{APIBaseURL: apiServer.URL, CodeloadBaseURL: codeloadServer.URL, HTTPClient: http.DefaultClient}
	r := New(client)

	result, err := r.Pull(context.Background(), "rifat/debate-pro")
	if err != nil {
		t.Fatalf("Pull() error = %v", err)
	}

	l, err := lock.Load(filepath.Join(result.Dir, lock.FileName))
	if err != nil {
		t.Fatalf("lock.Load() error = %v", err)
	}
	entry, ok := l.Packages["github:rifat/critic"]
	if !ok {
		t.Fatalf("expected orh.lock to record github:rifat/critic, got %+v", l.Packages)
	}
	if entry.Commit != "criticsha1" {
		t.Errorf("entry.Commit = %q, want %q", entry.Commit, "criticsha1")
	}

	// The dependency itself must also have been pulled into the cache.
	depDir, err := latestCachedDir("rifat", "critic")
	if err != nil {
		t.Fatalf("latestCachedDir() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(depDir, "orh.yaml")); err != nil {
		t.Errorf("expected dependency package to be cached: %v", err)
	}
}

func TestInstalled_ListsCachedPackages(t *testing.T) {
	t.Setenv("ORH_HOME", t.TempDir())

	client := fakeGitHub(t, "rifat", "debate-pro", "main", "a83bc91", map[string]string{
		"orh.yaml": testManifest,
		"main.orh": testArch,
	})
	r := New(client)
	if _, err := r.Pull(context.Background(), "rifat/debate-pro"); err != nil {
		t.Fatalf("Pull() error = %v", err)
	}

	refs, err := Installed()
	if err != nil {
		t.Fatalf("Installed() error = %v", err)
	}
	if len(refs) != 1 || refs[0] != "rifat/debate-pro" {
		t.Errorf("Installed() = %v, want [rifat/debate-pro]", refs)
	}
}

func TestGetInfo(t *testing.T) {
	t.Setenv("ORH_HOME", t.TempDir())

	client := fakeGitHub(t, "rifat", "debate-pro", "main", "a83bc91", map[string]string{
		"orh.yaml": testManifest,
		"main.orh": testArch,
	})
	r := New(client)
	if _, err := r.Pull(context.Background(), "rifat/debate-pro"); err != nil {
		t.Fatalf("Pull() error = %v", err)
	}

	info, err := GetInfo("rifat/debate-pro")
	if err != nil {
		t.Fatalf("GetInfo() error = %v", err)
	}
	if info.Name != "debate-pro" || info.ComponentCount != 1 {
		t.Errorf("GetInfo() = %+v, want Name=debate-pro ComponentCount=1", info)
	}
}
