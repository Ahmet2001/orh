package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pertevniyalai/orh/internal/resolver"
)

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <owner/repo>[@version]",
		Short: "Download an ORH architecture package from GitHub",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Downloading package...")

			if _, err := resolver.Default().Pull(cmd.Context(), args[0]); err != nil {
				return printValidationOrRun(cmd, err)
			}

			fmt.Fprintln(out, "✓ manifest valid")
			fmt.Fprintln(out, "✓ architecture cached")
			return nil
		},
	}
}
