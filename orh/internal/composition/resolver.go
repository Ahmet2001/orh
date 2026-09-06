package composition

import (
	"context"

	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/resolver"
)

// GitHubResolver resolves orchestration component sources through ORH's
// GitHub package resolver: pull (or reuse from cache), validate, and hand
// back the loaded package.
type GitHubResolver struct {
	Resolver *resolver.Resolver
}

// DefaultResolver returns a GitHubResolver configured against the real
// github.com.
func DefaultResolver() *GitHubResolver {
	return &GitHubResolver{Resolver: resolver.Default()}
}

// Resolve implements PackageResolver.
func (g *GitHubResolver) Resolve(ctx context.Context, source, version string) (*pkg.Package, error) {
	ref := source
	if version != "" {
		ref += "@" + version
	}

	result, err := g.Resolver.Pull(ctx, ref)
	if err != nil {
		return nil, err
	}
	return result.Package, nil
}
