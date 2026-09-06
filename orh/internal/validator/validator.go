// Package validator checks a parsed .orh architecture for structural and
// semantic correctness before it is turned into an execution graph.
package validator

import (
	"fmt"
	"strings"

	"github.com/pertevniyalai/orh/internal/spec"
)

// Result carries the outcome of validating an Architecture.
type Result struct {
	Errors []string
}

// Valid reports whether the architecture had no validation errors.
func (r Result) Valid() bool {
	return len(r.Errors) == 0
}

// Validate runs syntax and semantic checks over an Architecture.
func Validate(arch *spec.Architecture) Result {
	var errs []string

	errs = append(errs, checkSyntax(arch)...)
	if len(errs) == 0 {
		errs = append(errs, checkComponents(arch)...)
		errs = append(errs, checkConnections(arch)...)
	}

	return Result{Errors: errs}
}

func checkSyntax(arch *spec.Architecture) []string {
	var errs []string

	if arch.APIVersion == "" {
		errs = append(errs, "apiVersion is required")
	}
	if arch.Name == "" {
		errs = append(errs, "name is required")
	}
	if len(arch.Components) == 0 {
		errs = append(errs, "at least one component is required")
	}
	if len(arch.Connections) == 0 {
		errs = append(errs, "at least one connection is required")
	}

	return errs
}

func checkComponents(arch *spec.Architecture) []string {
	var errs []string

	for name, c := range arch.Components {
		switch c.Type {
		case "":
			errs = append(errs, fmt.Sprintf("component %q: type is required", name))
		case "agent":
			if c.Model == "" {
				errs = append(errs, fmt.Sprintf("component %q: model is required", name))
				continue
			}
			if _, ok := arch.Models[c.Model]; !ok {
				errs = append(errs, fmt.Sprintf("component %q: references undefined model %q", name, c.Model))
			}
		case "orchestration", "node":
			if c.Source == "" {
				errs = append(errs, fmt.Sprintf("component %q: source is required for %s components", name, c.Type))
			}
		default:
			errs = append(errs, fmt.Sprintf("component %q: unknown type %q", name, c.Type))
		}
	}

	return errs
}

func checkConnections(arch *spec.Architecture) []string {
	var errs []string

	for i, conn := range arch.Connections {
		if conn.From == "" || conn.To == "" {
			errs = append(errs, fmt.Sprintf("connection[%d]: from and to are required", i))
			continue
		}
		if endpointComponent(conn.From) == spec.NodeOutput {
			errs = append(errs, fmt.Sprintf("connection[%d]: %q cannot be a connection source", i, spec.NodeOutput))
		} else if err := checkEndpoint(arch, conn.From); err != nil {
			errs = append(errs, fmt.Sprintf("connection[%d] from %q: %s", i, conn.From, err))
		}

		if endpointComponent(conn.To) == spec.NodeInput {
			errs = append(errs, fmt.Sprintf("connection[%d]: %q cannot be a connection target", i, spec.NodeInput))
		} else if err := checkEndpoint(arch, conn.To); err != nil {
			errs = append(errs, fmt.Sprintf("connection[%d] to %q: %s", i, conn.To, err))
		}
	}

	return errs
}

// checkEndpoint validates a connection endpoint of the form "input",
// "input.port", "output", "output.port", "component", or "component.port".
func checkEndpoint(arch *spec.Architecture, endpoint string) error {
	name := endpointComponent(endpoint)
	if name == spec.NodeInput || name == spec.NodeOutput {
		return nil
	}

	if _, ok := arch.Components[name]; !ok {
		return fmt.Errorf("references undefined component %q", name)
	}
	return nil
}

// endpointComponent returns the component (or boundary node) part of a
// connection endpoint, e.g. "input.question" -> "input". It splits on the
// *last* dot since composition (see internal/composition) produces dotted
// component names, e.g. "search.crawler.input" -> component "search.crawler".
func endpointComponent(endpoint string) string {
	if idx := strings.LastIndex(endpoint, "."); idx != -1 {
		return endpoint[:idx]
	}
	return endpoint
}
