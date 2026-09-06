// Package cli wires ORH's cobra commands.
package cli

import "github.com/spf13/cobra"

// NewRootCmd builds the root `orh` command with its subcommands attached.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "orh",
		Short:         "ORH — Open Runtime for AI Orchestration",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newInitCmd())
	root.AddCommand(newValidateCmd())
	root.AddCommand(newInspectCmd())
	root.AddCommand(newRunCmd())
	root.AddCommand(newPullCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newInfoCmd())
	root.AddCommand(newDepsCmd())
	root.AddCommand(newToolsCmd())
	root.AddCommand(newToolCmd())
	root.AddCommand(newNodesCmd())
	root.AddCommand(newNodeCmd())

	return root
}
