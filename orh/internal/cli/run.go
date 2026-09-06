package cli

import (
	"bufio"
	"fmt"
	"io"
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
	var sessionID string
	var recordPath string

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
			userInput, err := readRunInput(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
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
				SessionID:     sessionID,
				RecordPath:    recordPath,
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
	cmd.Flags().StringVar(&sessionID, "session", "", "persist memory-enabled agents under this session id")
	cmd.Flags().StringVar(&recordPath, "record", "", "write a machine-readable JSON execution record")

	return cmd
}

// readRunInput keeps the one-line interactive CLI behavior while preserving
// all lines when input is piped from an evaluation script or a file.
func readRunInput(r io.Reader) (string, error) {
	if file, ok := r.(*os.File); ok {
		if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			scanner := bufio.NewScanner(file)
			if scanner.Scan() {
				return scanner.Text(), nil
			}
			return "", scanner.Err()
		}
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\r\n"), nil
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
