// Package parser reads .orh files into the spec data model.
package parser

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/pertevniyalai/orh/internal/spec"
)

// ParseFile reads and unmarshals a .orh file from disk.
func ParseFile(path string) (*spec.Architecture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return Parse(data)
}

// Parse unmarshals raw YAML bytes into an Architecture.
func Parse(data []byte) (*spec.Architecture, error) {
	var arch spec.Architecture
	if err := yaml.Unmarshal(data, &arch); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}
	return &arch, nil
}
