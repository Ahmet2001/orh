package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/runtime/executor"
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <file.orh | package-dir>",
		Short: "Validate an .orh architecture file or an ORH package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if info, err := os.Stat(args[0]); err == nil && info.IsDir() {
				return validatePackage(cmd, args[0])
			}

			_, err := executor.LoadAndValidate(cmd.Context(), args[0])
			if err != nil {
				return printValidationOrRun(cmd, err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "✓ syntax valid")
			fmt.Fprintln(cmd.OutOrStdout(), "✓ components valid")
			fmt.Fprintln(cmd.OutOrStdout(), "✓ connections valid")
			return nil
		},
	}
}

func validatePackage(cmd *cobra.Command, dir string) error {
	p, err := pkg.Load(dir)
	if err != nil {
		return err
	}

	if result := p.Validate(); !result.Valid() {
		return printValidationOrRun(cmd, &pkg.ValidationError{Errors: result})
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "✓ manifest valid")
	fmt.Fprintln(out, "✓ entrypoint found")
	fmt.Fprintln(out, "✓ architecture valid")
	return nil
}
