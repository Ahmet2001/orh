// Package cache manages ORH's local package cache under ~/.orh (overridable
// via the ORH_HOME environment variable, mainly for tests).
package cache

import (
	"os"
	"path/filepath"
)

// Root returns the ORH home directory.
func Root() (string, error) {
	if v := os.Getenv("ORH_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".orh"), nil
}

// GitHubDir returns the root of the GitHub package cache,
// ~/.orh/cache/github.
func GitHubDir() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "cache", "github"), nil
}

// PackageDir returns the cache directory for one resolved GitHub package
// version: ~/.orh/cache/github/{owner}/{repo}/{commit}.
func PackageDir(owner, repo, commit string) (string, error) {
	dir, err := GitHubDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, owner, repo, commit), nil
}

// SessionsDir returns the root of ORH's persisted agent-memory sessions,
// ~/.orh/sessions — one file per (session, component) key, written by
// internal/memory.FileStore.
func SessionsDir() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "sessions"), nil
}
