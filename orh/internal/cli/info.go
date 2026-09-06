package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/resolver"
)

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <owner/repo>",
		Short: "Show details about a cached ORH package (architecture or toolbox)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := resolver.GetInfo(args[0])
			if err != nil {
				return printValidationOrRun(cmd, err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Name:\n%s\n\n", info.Name)
			fmt.Fprintf(out, "Version:\n%s\n\n", info.Version)
			fmt.Fprintf(out, "Author:\n%s\n\n", info.Author)

			if info.Kind == pkg.KindToolbox {
				fmt.Fprintln(out, "Type:")
				fmt.Fprintln(out, "toolbox")
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Tools:")
				for _, name := range info.ToolNames {
					fmt.Fprintf(out, "✓ %s\n", name)
				}
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Runtime:")
				fmt.Fprintln(out, "MCP")
				return nil
			}

			if info.Kind == pkg.KindSkill {
				fmt.Fprintln(out, "Type:")
				fmt.Fprintln(out, "skill")
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Description:")
				fmt.Fprintln(out, info.SkillDescription)
				return nil
			}

			if info.Kind == pkg.KindNode {
				fmt.Fprintln(out, "Type:")
				fmt.Fprintln(out, "node")
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Runtime:")
				fmt.Fprintln(out, info.NodeRuntime)
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Module:")
				fmt.Fprintln(out, info.NodeModule)
				return nil
			}

			fmt.Fprintf(out, "Components:\n%d\n\n", info.ComponentCount)
			fmt.Fprintf(out, "Dependencies:\n%d\n", info.DependencyCount)
			return nil
		},
	}
}
