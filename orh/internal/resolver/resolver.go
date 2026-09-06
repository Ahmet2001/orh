// Package resolver ties together the GitHub client, local cache, lock
// files, and package validation into the high-level operations the CLI
// needs: pull a package (and its dependencies), list what's installed, and
// inspect a package's manifest.
package resolver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pertevniyalai/orh/internal/cache"
	"github.com/pertevniyalai/orh/internal/lock"
	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/resolver/github"
)

// Resolver performs package resolution against a configurable GitHub
// client, so it can be pointed at a fake server in tests.
type Resolver struct {
	Client *github.Client
}

// New builds a Resolver around the given GitHub client.
func New(client *github.Client) *Resolver {
	return &Resolver{Client: client}
}

// Default returns a Resolver configured against the real github.com.
func Default() *Resolver {
	return New(github.NewClient())
}

// PullResult reports the outcome of resolving and downloading one package.
type PullResult struct {
	Dir     string
	Package *pkg.Package
	Ref     string
	Commit  string
	Cached  bool
}

// Pull ensures the package identified by refStr ("owner/repo",
// "owner/repo@version", or "github:..." forms) is present in the local
// cache, downloading it if necessary, validating its manifest and
// entrypoint, and — if it declares dependencies — recursively pulling those
// too and recording them in the package's own orh.lock.
func (r *Resolver) Pull(ctx context.Context, refStr string) (*PullResult, error) {
	return r.pull(ctx, refStr, "", map[string]bool{})
}

func (r *Resolver) pull(ctx context.Context, refStr, pinnedCommit string, seen map[string]bool) (*PullResult, error) {
	ref, err := github.ParseRef(refStr)
	if err != nil {
		return nil, err
	}

	key := ref.Owner + "/" + ref.Repo
	if seen[key] {
		return nil, fmt.Errorf("circular dependency detected on %s", key)
	}
	seen[key] = true
	defer delete(seen, key)

	var resolved github.Resolved
	if pinnedCommit != "" {
		// A lock entry already records the immutable commit. Do not resolve
		// the original branch or tag again: tags can move, and cached locked
		// packages should remain usable without an API lookup.
		resolved = github.Resolved{GitRef: pinnedCommit, Commit: pinnedCommit}
	} else {
		resolved, err = r.Client.Resolve(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("resolving %s: %w", refStr, err)
		}
	}

	dir, err := cache.PackageDir(ref.Owner, ref.Repo, resolved.Commit)
	if err != nil {
		return nil, err
	}

	cached := true
	if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
		cached = false
		if err := downloadInto(ctx, r.Client, ref.Owner, ref.Repo, resolved.GitRef, dir); err != nil {
			return nil, fmt.Errorf("downloading %s: %w", refStr, err)
		}
	}

	p, err := pkg.Load(dir)
	if err != nil {
		return nil, err
	}
	if result := p.Validate(); !result.Valid() {
		return nil, &pkg.ValidationError{Errors: result}
	}

	lockPath := filepath.Join(dir, lock.FileName)
	l, err := lock.Load(lockPath)
	if err != nil {
		return nil, err
	}
	lockChanged := false

	if len(p.Manifest.Dependencies) > 0 {
		for name, dep := range p.Manifest.Dependencies {
			depRefStr := dep.Source
			if dep.Version != "" {
				depRefStr += "@" + dep.Version
			}

			depRef, err := github.ParseRef(dep.Source)
			if err != nil {
				return nil, fmt.Errorf("dependency %q: %w", name, err)
			}
			lockKey := "github:" + depRef.Owner + "/" + depRef.Repo
			locked := l.Packages[lockKey]

			depResult, err := r.pull(ctx, depRefStr, locked.Commit, seen)
			if err != nil {
				return nil, fmt.Errorf("dependency %q: %w", name, err)
			}
			if locked.Commit != "" && depResult.Commit != locked.Commit {
				return nil, fmt.Errorf("dependency %q: locked commit %s resolved as %s", name, locked.Commit, depResult.Commit)
			}

			if locked.Commit == "" {
				l.Packages[lockKey] = lock.PackageLock{
					Ref:    depResult.Ref,
					Commit: depResult.Commit,
				}
				lockChanged = true
			}
		}
	}

	if lockChanged {
		if err := l.Save(lockPath); err != nil {
			return nil, err
		}
	}
	if err := ApplyLock(p, l); err != nil {
		return nil, err
	}

	return &PullResult{Dir: dir, Package: p, Ref: resolved.GitRef, Commit: resolved.Commit, Cached: cached}, nil
}

// ApplyLock rewrites a loaded package's dependency and component versions to
// the immutable commits recorded in l. This keeps all later consumers
// (composition, toolboxes, skills, and custom nodes) on the same resolved
// package graph instead of resolving a mutable tag a second time.
func ApplyLock(p *pkg.Package, l *lock.Lock) error {
	for name, dep := range p.Manifest.Dependencies {
		key, err := lockKey(dep.Source)
		if err != nil {
			return fmt.Errorf("dependency %q: %w", name, err)
		}
		if pinned, ok := l.Packages[key]; ok && pinned.Commit != "" {
			dep.Version = pinned.Commit
			p.Manifest.Dependencies[name] = dep
		}
	}

	if p.Entrypoint != nil {
		for name, component := range p.Entrypoint.Components {
			if component.Source == "" {
				continue
			}
			key, err := lockKey(component.Source)
			if err != nil {
				return fmt.Errorf("component %q: %w", name, err)
			}
			if pinned, ok := l.Packages[key]; ok && pinned.Commit != "" {
				component.Version = pinned.Commit
				p.Entrypoint.Components[name] = component
			}
		}
	}
	return nil
}

func lockKey(source string) (string, error) {
	ref, err := github.ParseRef(source)
	if err != nil {
		return "", err
	}
	return "github:" + ref.Owner + "/" + ref.Repo, nil
}

// downloadInto downloads a package into a temporary directory and, only on
// success, atomically renames it into place at dir — so a failed or
// interrupted download never leaves a corrupt cache entry behind.
func downloadInto(ctx context.Context, client *github.Client, owner, repo, gitRef, dir string) error {
	tmp := dir + ".download"
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}

	if err := client.Download(ctx, owner, repo, gitRef, tmp); err != nil {
		os.RemoveAll(tmp)
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	os.RemoveAll(dir)
	return os.Rename(tmp, dir)
}

// Installed lists "owner/repo" for every package version currently in the
// local cache.
func Installed() ([]string, error) {
	githubDir, err := cache.GitHubDir()
	if err != nil {
		return nil, err
	}

	owners, err := os.ReadDir(githubDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var refs []string
	for _, owner := range owners {
		if !owner.IsDir() {
			continue
		}
		repos, err := os.ReadDir(filepath.Join(githubDir, owner.Name()))
		if err != nil {
			continue
		}
		for _, repo := range repos {
			if repo.IsDir() {
				refs = append(refs, owner.Name()+"/"+repo.Name())
			}
		}
	}

	sort.Strings(refs)
	return refs, nil
}

// Info summarizes a cached package's manifest.
type Info struct {
	Name             string
	Kind             string
	Version          string
	Author           string
	ComponentCount   int // set for kind: orchestration
	ToolNames        []string
	DependencyCount  int
	NodeRuntime      string // set for kind: node, e.g. "wasm"
	NodeModule       string // set for kind: node
	SkillDescription string // set for kind: skill
}

// GetInfo loads manifest details for the most recently cached version of
// owner/repo.
func GetInfo(refStr string) (*Info, error) {
	ref, err := github.ParseRef(refStr)
	if err != nil {
		return nil, err
	}

	dir, err := latestCachedDir(ref.Owner, ref.Repo)
	if err != nil {
		return nil, err
	}

	p, err := pkg.Load(dir)
	if err != nil {
		return nil, err
	}
	if result := p.Validate(); !result.Valid() {
		return nil, &pkg.ValidationError{Errors: result}
	}

	info := &Info{
		Name:            p.Manifest.Name,
		Kind:            p.Manifest.Kind,
		Version:         p.Manifest.Version,
		Author:          p.Manifest.Author.GitHub,
		DependencyCount: len(p.Manifest.Dependencies),
	}

	if p.Entrypoint != nil {
		info.ComponentCount = len(p.Entrypoint.Components)
	}
	if p.Toolbox != nil {
		for name := range p.Toolbox.Tools {
			info.ToolNames = append(info.ToolNames, name)
		}
		sort.Strings(info.ToolNames)
	}
	if p.Node != nil {
		info.NodeRuntime = p.Node.Runtime.Type
		info.NodeModule = p.Node.Entrypoint.Module
	}
	if p.Skill != nil {
		info.SkillDescription = p.Skill.Description
	}

	return info, nil
}

func latestCachedDir(owner, repo string) (string, error) {
	githubDir, err := cache.GitHubDir()
	if err != nil {
		return "", err
	}

	notInstalled := fmt.Errorf("%s/%s is not installed (run `orh pull %s/%s` first)", owner, repo, owner, repo)

	repoDir := filepath.Join(githubDir, owner, repo)
	commits, err := os.ReadDir(repoDir)
	if err != nil {
		return "", notInstalled
	}

	var best string
	var bestTime time.Time
	for _, c := range commits {
		if !c.IsDir() {
			continue
		}
		info, err := c.Info()
		if err != nil {
			continue
		}
		if best == "" || info.ModTime().After(bestTime) {
			best = c.Name()
			bestTime = info.ModTime()
		}
	}
	if best == "" {
		return "", notInstalled
	}

	return filepath.Join(repoDir, best), nil
}
