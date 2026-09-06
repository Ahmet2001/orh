// Package toolbox parses and validates .tb files: the toolbox package
// format Faz5 introduces alongside .orh architectures.
package toolbox

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/pertevniyalai/orh/internal/spec"
)

// ParseFile reads and unmarshals a .tb file from disk.
func ParseFile(path string) (*spec.Toolbox, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return Parse(data)
}

// Parse unmarshals raw YAML bytes into a Toolbox.
func Parse(data []byte) (*spec.Toolbox, error) {
	var tb spec.Toolbox
	if err := yaml.Unmarshal(data, &tb); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}
	return &tb, nil
}

// Validate checks a Toolbox for structural correctness.
func Validate(tb *spec.Toolbox) []string {
	var errs []string

	if tb.Version != 1 {
		errs = append(errs, fmt.Sprintf("unsupported toolbox version %d (expected 1)", tb.Version))
	}
	if len(tb.Tools) == 0 {
		errs = append(errs, "toolbox must declare at least one tool")
		return errs
	}

	for name, t := range tb.Tools {
		if t.Runtime.Type != "mcp" {
			errs = append(errs, fmt.Sprintf("tool %q: unsupported runtime type %q (only \"mcp\" is supported)", name, t.Runtime.Type))
			continue
		}
		if t.Runtime.Server == "" {
			errs = append(errs, fmt.Sprintf("tool %q: runtime.server is required", name))
		}
	}

	return errs
}
