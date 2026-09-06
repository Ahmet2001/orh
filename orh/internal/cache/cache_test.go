package cache

import (
	"path/filepath"
	"testing"
)

func TestRoot_UsesORHHomeOverride(t *testing.T) {
	t.Setenv("ORH_HOME", "/tmp/orh-test-home")

	root, err := Root()
	if err != nil {
		t.Fatalf("Root() error = %v", err)
	}
	if root != "/tmp/orh-test-home" {
		t.Errorf("Root() = %q, want %q", root, "/tmp/orh-test-home")
	}
}

func TestPackageDir(t *testing.T) {
	t.Setenv("ORH_HOME", "/tmp/orh-test-home")

	dir, err := PackageDir("rifat", "debate-pro", "a83bc91")
	if err != nil {
		t.Fatalf("PackageDir() error = %v", err)
	}

	want := filepath.Join("/tmp/orh-test-home", "cache", "github", "rifat", "debate-pro", "a83bc91")
	if dir != want {
		t.Errorf("PackageDir() = %q, want %q", dir, want)
	}
}
