package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/pertevniyalai/orh/internal/nodes"
	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/resolver"
	"github.com/pertevniyalai/orh/internal/runtime/events"
)

func newNodesCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "nodes",
		Short: "Work with locally cached node packages",
	}
	root.AddCommand(newNodesListCmd())
	return root
}

func newNodesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List locally cached node packages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			refs, err := resolver.Installed()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Installed nodes:")
			fmt.Fprintln(out)

			for _, ref := range refs {
				info, err := resolver.GetInfo(ref)
				if err != nil || info.Kind != pkg.KindNode {
					continue
				}
				fmt.Fprintln(out, info.Name)
			}
			return nil
		},
	}
}

func newNodeCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "node",
		Short: "Work with an individual node",
	}
	root.AddCommand(newNodeTestCmd())
	root.AddCommand(newNodeBuildCmd())
	return root
}

func newNodeTestCmd() *cobra.Command {
	var payload string
	var port string

	cmd := &cobra.Command{
		Use:   "test <package-dir | owner/repo[@version]>",
		Short: "Load a node, send it one test event, and print what it emits",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if info, err := os.Stat(dir); err != nil || !info.IsDir() {
				result, err := resolver.Default().Pull(cmd.Context(), args[0])
				if err != nil {
					return printValidationOrRun(cmd, err)
				}
				dir = result.Dir
			}

			p, err := pkg.Load(dir)
			if err != nil {
				return err
			}
			if result := p.Validate(); !result.Valid() {
				return printValidationOrRun(cmd, &pkg.ValidationError{Errors: result})
			}
			if p.Manifest.Kind != pkg.KindNode {
				return fmt.Errorf("%s is a %q package, not a node", args[0], p.Manifest.Kind)
			}

			node, err := nodes.Load(p.Dir, p.Node, p.Manifest.Name)
			if err != nil {
				return err
			}
			if err := node.Init(cmd.Context()); err != nil {
				return err
			}
			defer node.Shutdown(cmd.Context())

			cmds, err := node.Handle(cmd.Context(), testEvent(port, payload))
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, c := range cmds {
				fmt.Fprintf(out, "%+v\n", c)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&payload, "payload", "", "payload to send the node in the test event")
	cmd.Flags().StringVar(&port, "port", "input", "port the test event arrives on")

	return cmd
}

func newNodeBuildCmd() *cobra.Command {
	var output string
	var srcDir string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Compile a node written with sdk/node to WASM using TinyGo",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("tinygo"); err != nil {
				return fmt.Errorf("tinygo not found in PATH — install it from https://tinygo.org to build nodes")
			}

			tinygoCmd := exec.CommandContext(cmd.Context(), "tinygo", "build",
				"-o", output,
				"-target=wasip1",
				"-gc=leaking",
				"-no-debug",
				srcDir,
			)
			tinygoCmd.Stdout = cmd.OutOrStdout()
			tinygoCmd.Stderr = cmd.ErrOrStderr()

			if err := tinygoCmd.Run(); err != nil {
				return fmt.Errorf("tinygo build: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "built %s\n", output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "node.wasm", "output WASM file path")
	cmd.Flags().StringVar(&srcDir, "src", ".", "node source directory")

	return cmd
}

func testEvent(port, payload string) events.Event {
	return events.Event{Port: port, Payload: payload}
}
