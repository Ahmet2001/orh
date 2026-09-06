package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pertevniyalai/orh/internal/resolver"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List locally cached ORH architecture packages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			refs, err := resolver.Installed()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Installed architectures:")
			fmt.Fprintln(out)
			for _, ref := range refs {
				fmt.Fprintln(out, ref)
			}
			return nil
		},
	}
}
