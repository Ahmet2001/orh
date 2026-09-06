package nodes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pertevniyalai/orh/internal/spec"
)

// Load reads a node's compiled WASM module (named by manifest.Entrypoint.Module,
// resolved relative to dir, the node package's directory) and returns a
// NodeComponent ready for Init.
func Load(dir string, manifest *spec.Node, componentName string) (*NodeComponent, error) {
	if manifest.Runtime.Type != "wasm" {
		return nil, fmt.Errorf("node %q: unsupported runtime type %q", manifest.Name, manifest.Runtime.Type)
	}

	wasmPath := filepath.Join(dir, manifest.Entrypoint.Module)
	data, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", wasmPath, err)
	}

	return &NodeComponent{Name: componentName, WASMBytes: data, Permissions: manifest.Permissions}, nil
}
