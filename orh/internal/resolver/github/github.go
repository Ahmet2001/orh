// Package github resolves ORH package references ("owner/repo",
// "github:owner/repo@version") against real GitHub repositories: figuring
// out which commit a ref points at, and downloading + extracting the
// repository's source tree.
package github

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultAPIBaseURL      = "https://api.github.com"
	defaultCodeloadBaseURL = "https://codeload.github.com"
)

// Ref is a parsed ORH package reference.
type Ref struct {
	Owner string
	Repo  string
	// Version is a tag or commit from an explicit "@version" suffix, or ""
	// to mean "the repository's default branch".
	Version string
}

// String renders the ref back in "owner/repo[@version]" form.
func (r Ref) String() string {
	if r.Version == "" {
		return r.Owner + "/" + r.Repo
	}
	return r.Owner + "/" + r.Repo + "@" + r.Version
}

// ParseRef parses a package reference of the form "owner/repo",
// "owner/repo@version", "github:owner/repo", or "github:owner/repo@version".
func ParseRef(s string) (Ref, error) {
	rest := strings.TrimPrefix(s, "github:")

	var version string
	if idx := strings.LastIndex(rest, "@"); idx != -1 {
		version = rest[idx+1:]
		rest = rest[:idx]
	}

	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Ref{}, fmt.Errorf("invalid package reference %q (want owner/repo or owner/repo@version)", s)
	}

	return Ref{Owner: parts[0], Repo: parts[1], Version: version}, nil
}

// Resolved is a Ref pinned to an exact git ref and commit.
type Resolved struct {
	GitRef string // the branch or tag name actually used
	Commit string // the exact commit sha it points at
}

// Client resolves and downloads packages from GitHub. Its base URLs are
// fields (rather than constants) so tests can point it at a local fake
// server instead of the real GitHub.
type Client struct {
	APIBaseURL      string
	CodeloadBaseURL string
	HTTPClient      *http.Client
}

// NewClient returns a Client configured against the real github.com.
func NewClient() *Client {
	return &Client{
		APIBaseURL:      defaultAPIBaseURL,
		CodeloadBaseURL: defaultCodeloadBaseURL,
		HTTPClient:      http.DefaultClient,
	}
}

// Resolve determines the exact git ref and commit a Ref points at: its
// explicit Version if given, otherwise the repository's default branch.
func (c *Client) Resolve(ctx context.Context, ref Ref) (Resolved, error) {
	gitRef := ref.Version
	if gitRef == "" {
		branch, err := c.defaultBranch(ctx, ref.Owner, ref.Repo)
		if err != nil {
			return Resolved{}, fmt.Errorf("looking up default branch: %w", err)
		}
		gitRef = branch
	}

	commit, err := c.commitSHA(ctx, ref.Owner, ref.Repo, gitRef)
	if err != nil {
		return Resolved{}, fmt.Errorf("looking up commit for %s: %w", gitRef, err)
	}

	return Resolved{GitRef: gitRef, Commit: commit}, nil
}

type repoInfo struct {
	DefaultBranch string `json:"default_branch"`
}

func (c *Client) defaultBranch(ctx context.Context, owner, repo string) (string, error) {
	var info repoInfo
	if err := c.getJSON(ctx, fmt.Sprintf("%s/repos/%s/%s", c.APIBaseURL, owner, repo), &info); err != nil {
		return "", err
	}
	if info.DefaultBranch == "" {
		return "", fmt.Errorf("repository %s/%s has no default branch (does it exist?)", owner, repo)
	}
	return info.DefaultBranch, nil
}

type commitInfo struct {
	SHA string `json:"sha"`
}

func (c *Client) commitSHA(ctx context.Context, owner, repo, gitRef string) (string, error) {
	var info commitInfo
	if err := c.getJSON(ctx, fmt.Sprintf("%s/repos/%s/%s/commits/%s", c.APIBaseURL, owner, repo, gitRef), &info); err != nil {
		return "", err
	}
	if info.SHA == "" {
		return "", fmt.Errorf("ref %q not found", gitRef)
	}
	return info.SHA, nil
}

func (c *Client) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s returned %s: %s", url, resp.Status, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// Download fetches the tarball for owner/repo at gitRef and extracts it
// into destDir, stripping the tarball's single top-level directory.
func (c *Client) Download(ctx context.Context, owner, repo, gitRef, destDir string) error {
	url := fmt.Sprintf("%s/%s/%s/tar.gz/%s", c.CodeloadBaseURL, owner, repo, gitRef)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s returned %s: %s", url, resp.Status, string(body))
	}

	return extractTarGz(resp.Body, destDir)
}

// extractTarGz extracts a gzip-compressed tarball into destDir, stripping
// each entry's first path component (the "{repo}-{ref}/" wrapper directory
// GitHub's codeload tarballs always have). It rejects any entry that would
// extract outside destDir (a "zip slip" path traversal attack).
func extractTarGz(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("opening tarball: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	destRoot := filepath.Clean(destDir)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading tarball: %w", err)
		}

		name := hdr.Name
		if idx := strings.Index(name, "/"); idx != -1 {
			name = name[idx+1:]
		} else {
			continue // the wrapper directory entry itself
		}
		if name == "" {
			continue
		}

		target := filepath.Join(destRoot, name)
		if target != destRoot && !strings.HasPrefix(target, destRoot+string(os.PathSeparator)) {
			return fmt.Errorf("tar entry %q escapes destination directory", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := writeFile(target, tr, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		}
	}
}

func writeFile(path string, r io.Reader, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}
