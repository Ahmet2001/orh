package cli

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/pertevniyalai/orh/internal/composition"
	"github.com/pertevniyalai/orh/internal/runtime/executor"
	"github.com/pertevniyalai/orh/internal/spec"
)

func newDepsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deps <file.orh | package-dir | owner/repo[@version]>",
		Short: "Show an architecture's composition dependency tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := resolveEntrypoint(cmd.Context(), args[0])
			if err != nil {
				return printValidationOrRun(cmd, err)
			}

			arch, err := executor.LoadRaw(entry.Path)
			if err != nil {
				return printValidationOrRun(cmd, err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, args[0])
			fmt.Fprintln(out)

			return printDepsTree(cmd.Context(), out, arch, "")
		},
	}
}

type orchestrationDep struct {
	componentName string
	ref           string
	source        string
	version       string
}

func orchestrationDeps(arch *spec.Architecture) []orchestrationDep {
	var deps []orchestrationDep
	for name, c := range arch.Components {
		if c.Type != "orchestration" {
			continue
		}
		ref := c.Source
		if c.Version != "" {
			ref += "@" + c.Version
		}
		deps = append(deps, orchestrationDep{componentName: name, ref: ref, source: c.Source, version: c.Version})
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].componentName < deps[j].componentName })
	return deps
}

// printDepsTree prints a recursive tree of an architecture's `type:
// orchestration` components, resolving each one to show its own
// dependencies in turn.
func printDepsTree(ctx context.Context, out io.Writer, arch *spec.Architecture, prefix string) error {
	deps := orchestrationDeps(arch)
	resolver := composition.DefaultResolver()

	for i, dep := range deps {
		last := i == len(deps)-1
		branch, childPrefix := "├── ", prefix+"│   "
		if last {
			branch, childPrefix = "└── ", prefix+"    "
		}
		fmt.Fprintf(out, "%s%s%s\n", prefix, branch, dep.ref)

		p, err := resolver.Resolve(ctx, dep.source, dep.version)
		if err != nil {
			return fmt.Errorf("resolving %s: %w", dep.ref, err)
		}
		if err := printDepsTree(ctx, out, p.Entrypoint, childPrefix); err != nil {
			return err
		}
	}

	return nil
}
