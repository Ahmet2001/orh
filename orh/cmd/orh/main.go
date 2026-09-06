// Command orh is the ORH CLI entry point.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pertevniyalai/orh/internal/cli"
)

func main() {
	root := cli.NewRootCmd()
	if err := root.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
