// Package lock reads and writes orh.lock files, which pin the exact
// commit a package's dependencies were resolved to so re-pulling reproduces
// the same result.
package lock

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FileName is the standard lock file name.
const FileName = "orh.lock"

// Lock is the parsed content of an orh.lock file.
type Lock struct {
	LockVersion int                    `yaml:"lockVersion"`
	Packages    map[string]PackageLock `yaml:"packages"`
}

// PackageLock pins one dependency to the exact ref and commit it resolved
// to.
type PackageLock struct {
	Ref    string `yaml:"ref"`
	Commit string `yaml:"commit"`
}

// New returns an empty, ready-to-populate Lock.
func New() *Lock {
	return &Lock{LockVersion: 1, Packages: map[string]PackageLock{}}
}

// Load reads a lock file from path. A missing file is not an error — it
// returns a fresh empty Lock, since a package may not have one yet.
func Load(path string) (*Lock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var l Lock
	if err := yaml.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if l.Packages == nil {
		l.Packages = map[string]PackageLock{}
	}
	return &l, nil
}

// Save writes the lock file to path.
func (l *Lock) Save(path string) error {
	data, err := yaml.Marshal(l)
	if err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
