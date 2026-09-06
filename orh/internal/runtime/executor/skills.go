package executor

import (
	"context"
	"fmt"

	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/resolver"
	"github.com/pertevniyalai/orh/internal/spec"
)

// needsSkills reports whether any component references a skill *by
// alias* (an inline skill needs no package resolution at all), so callers
// can skip resolving skill dependencies for architectures that don't need
// it.
func needsSkills(arch *spec.Architecture) bool {
	for _, c := range arch.Components {
		for _, ref := range c.Skills {
			if ref.Alias != "" {
				return true
			}
		}
	}
	return false
}

// BuildSkillPrompts pulls every skill-kind dependency in deps and returns
// its instructions text keyed by dependency alias — the same alias an
// agent's `skills:` list references. Unlike a toolbox, a skill has no
// runtime process to start; it's purely text to append to a prompt.
func BuildSkillPrompts(ctx context.Context, deps spec.Dependencies) (map[string]string, error) {
	prompts := map[string]string{}

	for alias, dep := range deps {
		ref := dep.Source
		if dep.Version != "" {
			ref += "@" + dep.Version
		}

		result, err := resolver.Default().Pull(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("dependency %q: %w", alias, err)
		}
		if result.Package.Manifest.Kind != pkg.KindSkill {
			continue
		}

		prompts[alias] = result.Package.Skill.Instructions
	}

	return prompts, nil
}
