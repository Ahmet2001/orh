package executor

import (
	"context"
	"fmt"
	"io"

	"github.com/pertevniyalai/orh/internal/mcp"
	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/resolver"
	"github.com/pertevniyalai/orh/internal/spec"
	"github.com/pertevniyalai/orh/internal/tools"
)

// needsTools reports whether any component in arch might call a tool, so
// callers can skip resolving toolbox dependencies (and the network access
// and subprocesses that implies) entirely for ordinary architectures. An
// agent only ever calls the tools listed in its own `tools:`, so that's
// known statically; a node's tool use is gated by its .node manifest's
// permissions.tools.allow, which isn't known until the node package is
// pulled — so any `type: node` component conservatively counts as needing
// the registry built.
func needsTools(arch *spec.Architecture) bool {
	for _, c := range arch.Components {
		if len(c.Tools) > 0 {
			return true
		}
		if c.Type == "node" {
			return true
		}
	}
	return false
}

// BuildToolRegistry pulls every toolbox-kind dependency in deps, launches
// the MCP server each of its tools runs on (one server process per unique
// runtime.server command, reused across tools that share it), discovers
// its live tool catalog, and registers each tool as "alias.toolName". The
// returned closers must be closed by the caller once the run is done.
func BuildToolRegistry(ctx context.Context, deps spec.Dependencies) (*tools.Registry, []io.Closer, error) {
	registry := tools.NewRegistry()
	var closers []io.Closer

	for alias, dep := range deps {
		ref := dep.Source
		if dep.Version != "" {
			ref += "@" + dep.Version
		}

		result, err := resolver.Default().Pull(ctx, ref)
		if err != nil {
			return registry, closers, fmt.Errorf("dependency %q: %w", alias, err)
		}
		if result.Package.Manifest.Kind != pkg.KindToolbox {
			continue
		}

		servers := map[string]*mcp.Client{}
		for toolName, def := range result.Package.Toolbox.Tools {
			if def.Runtime.Type != "mcp" {
				return registry, closers, fmt.Errorf("dependency %q: tool %q: unsupported runtime %q", alias, toolName, def.Runtime.Type)
			}
			if _, ok := servers[def.Runtime.Server]; ok {
				continue
			}

			client, err := mcp.Start(ctx, def.Runtime.Server, result.Dir)
			if err != nil {
				return registry, closers, fmt.Errorf("dependency %q: starting MCP server for tool %q: %w", alias, toolName, err)
			}
			closers = append(closers, client)

			if _, err := client.Initialize(ctx); err != nil {
				return registry, closers, fmt.Errorf("dependency %q: %w", alias, err)
			}
			servers[def.Runtime.Server] = client
		}

		for _, client := range servers {
			discovered, err := client.ListTools(ctx)
			if err != nil {
				return registry, closers, fmt.Errorf("dependency %q: %w", alias, err)
			}
			for _, t := range discovered {
				registry.Register(alias+"."+t.Name, tools.NewMCPTool(client, t.Name, t.Description, t.InputSchema))
			}
		}
	}

	return registry, closers, nil
}

func CloseAll(closers []io.Closer) {
	for _, c := range closers {
		c.Close()
	}
}
