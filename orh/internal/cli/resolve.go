package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/resolver"
	"github.com/pertevniyalai/orh/internal/spec"
)

// resolvedEntrypoint is a concrete local .orh file path to run or validate,
// plus the dependency map (if any) of the package it came from — needed to
// resolve `tools:` aliases on its components.
type resolvedEntrypoint struct {
	Path         string
	Dependencies spec.Dependencies
}

// resolveEntrypoint turns a CLI argument into a resolvedEntrypoint: a
// direct .orh file path has no dependencies, a local package directory is
// loaded and validated, and anything else is treated as a remote
// "owner/repo[@version]" package reference and pulled first.
func resolveEntrypoint(ctx context.Context, arg string) (resolvedEntrypoint, error) {
	if info, err := os.Stat(arg); err == nil {
		if info.IsDir() {
			p, err := pkg.Load(arg)
			if err != nil {
				return resolvedEntrypoint{}, err
			}
			if result := p.Validate(); !result.Valid() {
				return resolvedEntrypoint{}, &pkg.ValidationError{Errors: result}
			}
			return resolvedEntrypoint{
				Path:         filepath.Join(arg, p.Manifest.Entrypoint),
				Dependencies: p.Manifest.Dependencies,
			}, nil
		}
		return resolvedEntrypoint{Path: arg}, nil
	}

	result, err := resolver.Default().Pull(ctx, arg)
	if err != nil {
		return resolvedEntrypoint{}, err
	}
	return resolvedEntrypoint{
		Path:         filepath.Join(result.Dir, result.Package.Manifest.Entrypoint),
		Dependencies: result.Package.Manifest.Dependencies,
	}, nil
}
