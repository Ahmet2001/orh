// Package nodes loads .node manifests and runs the WASM modules they
// describe as ORH Component implementations.
package nodes

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/pertevniyalai/orh/internal/spec"
)

// ParseFile reads and unmarshals a .node file from disk.
func ParseFile(path string) (*spec.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return Parse(data)
}

// Parse unmarshals raw YAML bytes into a Node.
func Parse(data []byte) (*spec.Node, error) {
	var n spec.Node
	if err := yaml.Unmarshal(data, &n); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}
	return &n, nil
}

// Validate checks a Node manifest for structural correctness. It does not
// touch the referenced WASM module — see Load for that.
func Validate(n *spec.Node) []string {
	var errs []string

	if n.Runtime.Type != "wasm" {
		errs = append(errs, fmt.Sprintf("unsupported node runtime type %q (only \"wasm\" is supported)", n.Runtime.Type))
	}
	if n.Entrypoint.Module == "" {
		errs = append(errs, "entrypoint.module is required")
	}

	return errs
}
