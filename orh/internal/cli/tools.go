package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/resolver"
	"github.com/pertevniyalai/orh/internal/runtime/executor"
)

func newToolsCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tools",
		Short: "Work with locally cached toolbox packages",
	}
	root.AddCommand(newToolsListCmd())
	return root
}

func newToolsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List every tool across locally cached toolbox packages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			refs, err := resolver.Installed()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Installed tools:")
			fmt.Fprintln(out)

			for _, ref := range refs {
				info, err := resolver.GetInfo(ref)
				if err != nil || info.Kind != pkg.KindToolbox {
					continue
				}
				for _, name := range info.ToolNames {
					fmt.Fprintf(out, "%s.%s\n", ref, name)
				}
			}
			return nil
		},
	}
}

func newToolCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tool",
		Short: "Work with an individual tool",
	}
	root.AddCommand(newToolTestCmd())
	return root
}

func newToolTestCmd() *cobra.Command {
	var dir string
	var input string

	cmd := &cobra.Command{
		Use:   "test <alias.tool>",
		Short: "Call one tool from the current package's dependencies and print its result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := pkg.Load(dir)
			if err != nil {
				return err
			}
			if result := p.Validate(); !result.Valid() {
				return printValidationOrRun(cmd, &pkg.ValidationError{Errors: result})
			}

			registry, closers, err := executor.BuildToolRegistry(cmd.Context(), p.Manifest.Dependencies)
			defer executor.CloseAll(closers)
			if err != nil {
				return err
			}

			t, ok := registry.Get(args[0])
			if !ok {
				return fmt.Errorf("no tool named %q among %s's dependencies", args[0], dir)
			}

			result, err := t.Execute(cmd.Context(), json.RawMessage(input))
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), result)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "package directory to resolve the tool's dependencies from")
	cmd.Flags().StringVar(&input, "input", "{}", "JSON arguments to call the tool with")

	return cmd
}
