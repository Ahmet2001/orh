package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/pertevniyalai/orh/internal/graph"
	"github.com/pertevniyalai/orh/internal/runtime/executor"
	"github.com/pertevniyalai/orh/internal/spec"
)

func newInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <file.orh | package-dir | owner/repo[@version]>",
		Short: "Show an architecture's components, connections, and imported dependencies",
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
			fmt.Fprintf(out, "Architecture: %s\n\n", arch.Name)

			fmt.Fprintln(out, "Components:")
			var deps []string
			names := make([]string, 0, len(arch.Components))
			for name, c := range arch.Components {
				names = append(names, name)
				if c.Type == "orchestration" {
					ref := c.Source
					if c.Version != "" {
						ref += "@" + c.Version
					}
					deps = append(deps, ref)
				}
			}
			sort.Strings(names)
			for _, name := range names {
				fmt.Fprintf(out, "✓ %s\n", name)
			}
			fmt.Fprintln(out)

			fmt.Fprintln(out, "Connections:")
			printConnections(out, arch)

			if len(deps) > 0 {
				sort.Strings(deps)
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Dependencies:")
				for _, d := range deps {
					fmt.Fprintf(out, "✓ %s\n", d)
				}
			}

			return nil
		},
	}
}

// printConnections renders the graph as a simple top-to-bottom chain when
// the architecture has no branching, and falls back to a flat "from -> to"
// listing otherwise (fan-out, fan-in, or cycles).
func printConnections(out io.Writer, arch *spec.Architecture) {
	g := graph.Build(arch)

	if chain, ok := linearChain(g); ok {
		fmt.Fprintln(out, chain[0])
		for _, node := range chain[1:] {
			fmt.Fprintln(out, " |")
			fmt.Fprintln(out, node)
		}
		return
	}

	for _, c := range arch.Connections {
		fmt.Fprintf(out, "%s -> %s\n", c.From, c.To)
	}
}

// linearChain reports whether the architecture is a single, unbranched
// input -> ... -> output chain, returning the ordered node names if so.
func linearChain(g *graph.Graph) ([]string, bool) {
	chain := []string{spec.NodeInput}
	current := spec.NodeInput
	visited := map[string]bool{}

	for {
		succs := g.Successors(current)
		if len(succs) != 1 {
			return nil, false
		}

		next := succs[0]
		if next == spec.NodeOutput {
			chain = append(chain, spec.NodeOutput)
			return chain, true
		}

		name, _ := graph.SplitEndpoint(next)
		if visited[name] {
			return nil, false
		}
		visited[name] = true
		chain = append(chain, name)
		current = name + ".output"
	}
}
