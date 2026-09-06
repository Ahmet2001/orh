package lock

import (
	"path/filepath"
	"testing"
)

func TestLoad_MissingFileReturnsEmptyLock(t *testing.T) {
	l, err := Load(filepath.Join(t.TempDir(), "orh.lock"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if l.LockVersion != 1 || len(l.Packages) != 0 {
		t.Errorf("Load() = %+v, want a fresh empty lock", l)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orh.lock")

	l := New()
	l.Packages["github:rifat/debate-pro"] = PackageLock{Ref: "v1.2.0", Commit: "a83bc91"}

	if err := l.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got, ok := loaded.Packages["github:rifat/debate-pro"]
	if !ok {
		t.Fatal("expected package entry to round-trip")
	}
	if got.Ref != "v1.2.0" || got.Commit != "a83bc91" {
		t.Errorf("got = %+v, want {Ref: v1.2.0, Commit: a83bc91}", got)
	}
}
