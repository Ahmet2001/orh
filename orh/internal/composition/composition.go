// Package composition implements ORH's architecture composition: expanding
// `type: orchestration` components into the architecture they reference.
//
// This is deliberately a pure, build-time graph rewrite and not a new
// runtime concept. A "sub-architecture" is flattened — its components
// renamed with a prefix and merged in, its connections rewritten — into a
// single, ordinary architecture before validation and execution ever see
// it. The scheduler and every existing Component implementation stay
// completely unaware that composition happened at all: there is still only
// one Component contract in ORH, not a second "orchestration" runtime type.
package composition

import (
	"context"
	"fmt"

	"github.com/pertevniyalai/orh/internal/graph"
	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/spec"
)

// PackageResolver resolves an orchestration component's source (and
// optional version) into a loaded, validated package. It is an interface
// so tests can substitute an in-memory fake instead of hitting GitHub.
type PackageResolver interface {
	Resolve(ctx context.Context, source, version string) (*pkg.Package, error)
}

// Flatten expands every `type: orchestration` component in arch,
// recursively, into the plain components and connections of the
// architecture it references. The result contains no orchestration-typed
// components — only whatever native types (currently just "agent") the
// referenced architectures used.
func Flatten(ctx context.Context, arch *spec.Architecture, resolver PackageResolver) (*spec.Architecture, error) {
	return flatten(ctx, arch, resolver, map[string]bool{})
}

// HasOrchestrationComponents reports whether arch has any `type:
// orchestration` components — callers can use this to skip composition
// (and the network access it implies) entirely for ordinary architectures.
func HasOrchestrationComponents(arch *spec.Architecture) bool {
	for _, c := range arch.Components {
		if c.Type == "orchestration" {
			return true
		}
	}
	return false
}

func flatten(ctx context.Context, arch *spec.Architecture, resolver PackageResolver, seen map[string]bool) (*spec.Architecture, error) {
	out := &spec.Architecture{
		APIVersion: arch.APIVersion,
		Name:       arch.Name,
		Inputs:     arch.Inputs,
		Outputs:    arch.Outputs,
		Models:     cloneModels(arch.Models),
		Components: map[string]spec.Component{},
	}

	// inputTargets/outputSources record how a reference to an orchestration
	// component's boundary — bare "name" or ported "name.port" — should be
	// rewritten once that component's internals are inlined under the
	// "name." prefix.
	inputTargets := map[string][]string{}
	outputSources := map[string][]string{}

	for name, c := range arch.Components {
		if c.Type != "orchestration" {
			out.Components[name] = c
			continue
		}

		key := c.Source
		if c.Version != "" {
			key += "@" + c.Version
		}
		if seen[key] {
			return nil, fmt.Errorf("circular dependency detected on %s", key)
		}

		p, err := resolver.Resolve(ctx, c.Source, c.Version)
		if err != nil {
			return nil, fmt.Errorf("component %q: resolving %s: %w", name, key, err)
		}
		if p.Entrypoint == nil {
			return nil, fmt.Errorf("component %q: package %s has no validated entrypoint", name, key)
		}

		child, err := flatten(ctx, p.Entrypoint, resolver, withSeen(seen, key))
		if err != nil {
			return nil, fmt.Errorf("component %q: %w", name, err)
		}

		mergeChild(out, name, child, inputTargets, outputSources)
	}

	for _, conn := range arch.Connections {
		out.Connections = append(out.Connections, rewriteConnection(conn, inputTargets, outputSources)...)
	}

	return out, nil
}

// mergeChild inlines an already-flattened child architecture into out under
// the given name prefix: its models and components are renamed and merged
// in, its purely-internal connections are renamed and copied in, and its
// boundary connections (the ones touching its own "input"/"output") are
// recorded into inputTargets/outputSources instead of being copied,
// because they describe how the *parent's* references to this component
// should be rewritten.
func mergeChild(out *spec.Architecture, prefix string, child *spec.Architecture, inputTargets, outputSources map[string][]string) {
	for mname, m := range child.Models {
		out.Models[prefix+"."+mname] = m
	}

	for cname, c := range child.Components {
		if c.Model != "" {
			c.Model = prefix + "." + c.Model
		}
		out.Components[prefix+"."+cname] = c
	}

	for _, conn := range child.Connections {
		fromComp, fromPort := graph.SplitEndpoint(conn.From)
		toComp, toPort := graph.SplitEndpoint(conn.To)

		switch {
		case fromComp == spec.NodeInput && toComp == spec.NodeOutput:
			// A pass-through boundary (input wired straight to output)
			// isn't needed by any real package so far; skip it rather
			// than half-support it.
			continue
		case fromComp == spec.NodeInput:
			key := boundaryKey(prefix, fromPort)
			inputTargets[key] = append(inputTargets[key], prefixEndpoint(prefix, toComp, toPort))
		case toComp == spec.NodeOutput:
			key := boundaryKey(prefix, toPort)
			outputSources[key] = append(outputSources[key], prefixEndpoint(prefix, fromComp, fromPort))
		default:
			out.Connections = append(out.Connections, spec.Connection{
				From: prefixEndpoint(prefix, fromComp, fromPort),
				To:   prefixEndpoint(prefix, toComp, toPort),
			})
		}
	}
}

// rewriteConnection rewrites one connection from the *importing*
// architecture: an endpoint that refers to an orchestration component's
// boundary (bare "name" or ported "name.port") is replaced by the internal
// endpoint(s) recorded for it, fanning out to multiple connections if the
// referenced port maps to more than one internal target. An endpoint that
// isn't an orchestration component reference passes through unchanged.
func rewriteConnection(conn spec.Connection, inputTargets, outputSources map[string][]string) []spec.Connection {
	froms, ok := outputSources[conn.From]
	if !ok {
		froms = []string{conn.From}
	}

	tos, ok := inputTargets[conn.To]
	if !ok {
		tos = []string{conn.To}
	}

	out := make([]spec.Connection, 0, len(froms)*len(tos))
	for _, f := range froms {
		for _, t := range tos {
			out = append(out, spec.Connection{From: f, To: t})
		}
	}
	return out
}

func boundaryKey(prefix, port string) string {
	if port == "" {
		return prefix
	}
	return prefix + "." + port
}

func prefixEndpoint(prefix, comp, port string) string {
	if port == "" {
		return prefix + "." + comp
	}
	return prefix + "." + comp + "." + port
}

func cloneModels(m map[string]spec.Model) map[string]spec.Model {
	out := make(map[string]spec.Model, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func withSeen(seen map[string]bool, key string) map[string]bool {
	out := make(map[string]bool, len(seen)+1)
	for k, v := range seen {
		out[k] = v
	}
	out[key] = true
	return out
}
