package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/runtime/executor"
	"github.com/pertevniyalai/orh/internal/runtime/scheduler"
)

func newRunCmd() *cobra.Command {
	var modelOverride string
	var debug bool

	cmd := &cobra.Command{
		Use:   "run <file.orh | package-dir | owner/repo[@version]>",
		Short: "Run a local .orh file, a local package, or a remote ORH package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := resolveEntrypoint(cmd.Context(), args[0])
			if err != nil {
				return printValidationOrRun(cmd, err)
			}

			fmt.Fprint(cmd.OutOrStdout(), "> ")
			scanner := bufio.NewScanner(os.Stdin)
			var userInput string
			if scanner.Scan() {
				userInput = scanner.Text()
			}

			var trace scheduler.Trace
			if debug {
				trace = func(kind, value string) {
					if value == "" {
						fmt.Fprintf(cmd.OutOrStdout(), "%s\n", kind)
						return
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s:\n%s\n\n", kind, value)
				}
			}

			outputs, err := executor.Run(cmd.Context(), entry.Path, userInput, executor.Options{
				ModelOverride: modelOverride,
				Trace:         trace,
				Dependencies:  entry.Dependencies,
			})
			if err != nil {
				return printValidationOrRun(cmd, err)
			}

			for _, out := range outputs {
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(out.Text()))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&modelOverride, "model", "", "override model as provider:name, e.g. ollama:qwen3")
	cmd.Flags().BoolVar(&debug, "debug", false, "print a step-by-step execution trace")

	return cmd
}

// printValidationOrRun prints per-error lines for a validation error
// (either an .orh file's or a package's) and returns a short summary error;
// any other error is returned unchanged.
func printValidationOrRun(cmd *cobra.Command, err error) error {
	switch verr := err.(type) {
	case *executor.ValidationError:
		for _, e := range verr.Errors {
			fmt.Fprintf(cmd.OutOrStdout(), "✗ %s\n", e)
		}
	case *pkg.ValidationError:
		for _, e := range verr.Errors {
			fmt.Fprintf(cmd.OutOrStdout(), "✗ %s\n", e)
		}
	}
	return err
}
