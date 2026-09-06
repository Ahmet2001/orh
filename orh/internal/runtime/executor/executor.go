// Package executor is the high-level entry point that turns a .orh file on
// disk into a running graph: parse, validate, build components, run the
// scheduler.
package executor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pertevniyalai/orh/internal/cache"
	"github.com/pertevniyalai/orh/internal/components"
	"github.com/pertevniyalai/orh/internal/composition"
	"github.com/pertevniyalai/orh/internal/graph"
	"github.com/pertevniyalai/orh/internal/memory"
	"github.com/pertevniyalai/orh/internal/nodes"
	"github.com/pertevniyalai/orh/internal/parser"
	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/providers/resolve"
	"github.com/pertevniyalai/orh/internal/resolver"
	"github.com/pertevniyalai/orh/internal/runtime/events"
	"github.com/pertevniyalai/orh/internal/runtime/scheduler"
	"github.com/pertevniyalai/orh/internal/spec"
	"github.com/pertevniyalai/orh/internal/tools"
	"github.com/pertevniyalai/orh/internal/validator"
)

// ValidationError is returned when the architecture fails validation.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed with %d error(s)", len(e.Errors))
}

// Options configures a Run.
type Options struct {
	// ModelOverride, if set, is a "provider:name" string applied to every
	// agent component regardless of what its .orh model slot declares.
	ModelOverride string
	// Trace, if set, receives scheduler execution steps (see
	// scheduler.Trace) — used to power `orh run --debug`.
	Trace scheduler.Trace
	// Dependencies is the owning package's orh.yaml dependency map, used to
	// resolve `tools:` aliases on agent components (e.g. "web.search" ->
	// the "web" dependency's toolbox). Empty for a bare .orh file run
	// outside of any package.
	Dependencies spec.Dependencies
	// SessionID, if set, makes every `memory: true` agent in this run
	// persist its history to disk (~/.orh/sessions) under this id, so it
	// survives across separate `orh run` invocations. Empty means memory
	// is still active within this one run (useful for a component
	// revisited in a cyclic graph) but never touches disk.
	SessionID string
}

// LoadAndValidate parses, composes (flattens any `type: orchestration`
// components), and validates a .orh file without running it. It is shared
// by `orh validate`, `orh inspect`, and `orh run`. Composition only touches
// the network (resolving and pulling imported packages) when the
// architecture actually has orchestration components; an ordinary .orh
// file is validated fully offline, same as before Faz 4.
func LoadAndValidate(ctx context.Context, path string) (*spec.Architecture, error) {
	arch, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}

	if composition.HasOrchestrationComponents(arch) {
		arch, err = composition.Flatten(ctx, arch, composition.DefaultResolver())
		if err != nil {
			return nil, err
		}
	}

	result := validator.Validate(arch)
	if !result.Valid() {
		return nil, &ValidationError{Errors: result.Errors}
	}

	return arch, nil
}

// LoadRaw parses and structurally validates a .orh file *without* expanding
// any `type: orchestration` components — used by `orh inspect` to show a
// package's own declared structure (including which components are
// imported from elsewhere) rather than the fully composed graph. Unlike
// LoadAndValidate, this never touches the network.
func LoadRaw(path string) (*spec.Architecture, error) {
	arch, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}

	result := validator.Validate(arch)
	if !result.Valid() {
		return nil, &ValidationError{Errors: result.Errors}
	}

	return arch, nil
}

// Run loads, composes, validates, and executes a .orh file, feeding input
// in on the graph's "input" boundary and returning whatever reaches
// "output".
func Run(ctx context.Context, path string, input string, opts Options) ([]events.Event, error) {
	arch, err := LoadAndValidate(ctx, path)
	if err != nil {
		return nil, err
	}

	registry := tools.NewRegistry()
	var closers []io.Closer
	if needsTools(arch) {
		registry, closers, err = BuildToolRegistry(ctx, opts.Dependencies)
		if err != nil {
			CloseAll(closers)
			return nil, err
		}
	}
	defer CloseAll(closers)

	skillPrompts := map[string]string{}
	if needsSkills(arch) {
		skillPrompts, err = BuildSkillPrompts(ctx, opts.Dependencies)
		if err != nil {
			return nil, err
		}
	}

	comps, err := buildComponents(ctx, arch, opts.ModelOverride, registry, skillPrompts)
	if err != nil {
		return nil, err
	}

	sched := &scheduler.Scheduler{
		Graph:      graph.Build(arch),
		Components: comps,
		Trace:      opts.Trace,
	}

	return sched.Run(ctx, events.Event{Payload: input})
}

func buildComponents(ctx context.Context, arch *spec.Architecture, modelOverride string, registry *tools.Registry, skillPrompts map[string]string) (map[string]components.Component, error) {
	built := make(map[string]components.Component)

	for name, c := range arch.Components {
		if c.Type == "node" {
			node, err := buildNodeComponent(ctx, name, c, arch.Models, registry)
			if err != nil {
				return nil, err
			}
			built[name] = node
			continue
		}

		if c.Type != "agent" {
			return nil, fmt.Errorf("component %q: unsupported type %q", name, c.Type)
		}

		modelSpec := arch.Models[c.Model]
		providerName := modelSpec.Provider
		modelName := modelSpec.Name

		if modelOverride != "" {
			parts := strings.SplitN(modelOverride, ":", 2)
			providerName = parts[0]
			if len(parts) == 2 {
				modelName = parts[1]
			}
		}

		prompt := c.Prompt
		for _, ref := range c.Skills {
			text := ref.Inline
			if ref.Alias != "" {
				var ok bool
				text, ok = skillPrompts[ref.Alias]
				if !ok {
					return nil, fmt.Errorf("component %q: references undefined skill %q", name, ref.Alias)
				}
			}
			prompt = strings.TrimRight(prompt, "\n") + "\n\n" + text
		}

		agent := &components.AgentComponent{Name: name, Prompt: prompt}

		if len(c.Tools) == 0 {
			provider, err := resolve.Provider(providerName, modelName)
			if err != nil {
				return nil, fmt.Errorf("component %q: %w", name, err)
			}
			agent.Provider = provider
		} else {
			chatProvider, err := resolve.ChatProvider(providerName, modelName)
			if err != nil {
				return nil, fmt.Errorf("component %q: %w", name, err)
			}
			agent.ChatProvider = chatProvider

			for _, toolRef := range c.Tools {
				t, ok := registry.Get(toolRef)
				if !ok {
					return nil, fmt.Errorf("component %q: references undefined tool %q", name, toolRef)
				}
				agent.Tools = append(agent.Tools, t)
			}
		}

		built[name] = agent
	}

	return built, nil
}

// buildNodeComponent resolves a `type: node` component's source package and
// loads its WASM module into a nodes.NodeComponent. models and registry are
// the architecture's model slots and resolved tool registry, threaded
// through so the node's orh_call_model/orh_call_tool host functions have
// something to call — gated by the node's own .node permissions block.
func buildNodeComponent(ctx context.Context, name string, c spec.Component, models map[string]spec.Model, registry *tools.Registry) (*nodes.NodeComponent, error) {
	ref := c.Source
	if c.Version != "" {
		ref += "@" + c.Version
	}

	result, err := resolver.Default().Pull(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("component %q: resolving %s: %w", name, ref, err)
	}
	if result.Package.Manifest.Kind != pkg.KindNode {
		return nil, fmt.Errorf("component %q: %s is a %q package, not a node", name, ref, result.Package.Manifest.Kind)
	}

	node, err := nodes.Load(result.Dir, result.Package.Node, name)
	if err != nil {
		return nil, fmt.Errorf("component %q: %w", name, err)
	}
	node.Models = models
	node.Tools = registry
	return node, nil
}
