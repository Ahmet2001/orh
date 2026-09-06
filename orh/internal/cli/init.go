package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const scaffold = `apiVersion: orh/v1
name: hello-agent

models:
  main:
    provider: ollama
    name: qwen3

components:
  assistant:
    type: agent
    model: main
    prompt: |
      Answer the user question.

connections:
  - from: input
    to: assistant.input

  - from: assistant.output
    to: output
`

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [directory]",
		Short: "Scaffold a new ORH project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "my-agent"
			if len(args) == 1 {
				dir = args[0]
			}

			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("creating %s: %w", dir, err)
			}

			path := filepath.Join(dir, "main.orh")
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists", path)
			}

			if err := os.WriteFile(path, []byte(scaffold), 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", path, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", path)
			return nil
		},
	}
}
