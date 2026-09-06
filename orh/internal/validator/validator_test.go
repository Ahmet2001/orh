package validator

import (
	"testing"

	"github.com/pertevniyalai/orh/internal/spec"
)

func validArch() *spec.Architecture {
	return &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "hello-agent",
		Models: map[string]spec.Model{
			"main": {Provider: "ollama", Name: "qwen3"},
		},
		Components: map[string]spec.Component{
			"assistant": {Type: "agent", Model: "main", Prompt: "hi"},
		},
		Connections: []spec.Connection{
			{From: "input", To: "assistant.input"},
			{From: "assistant.output", To: "output"},
		},
	}
}

func TestValidate_Valid(t *testing.T) {
	result := Validate(validArch())
	if !result.Valid() {
		t.Fatalf("expected valid architecture, got errors: %v", result.Errors)
	}
}

func TestValidate_MissingName(t *testing.T) {
	arch := validArch()
	arch.Name = ""

	result := Validate(arch)
	if result.Valid() {
		t.Fatal("expected invalid architecture")
	}
}

func TestValidate_UndefinedModel(t *testing.T) {
	arch := validArch()
	arch.Components["assistant"] = spec.Component{Type: "agent", Model: "missing", Prompt: "hi"}

	result := Validate(arch)
	if result.Valid() {
		t.Fatal("expected invalid architecture for undefined model reference")
	}
}

func TestValidate_UndefinedComponentInConnection(t *testing.T) {
	arch := validArch()
	arch.Connections = append(arch.Connections, spec.Connection{From: "ghost.output", To: "output"})

	result := Validate(arch)
	if result.Valid() {
		t.Fatal("expected invalid architecture for undefined component reference")
	}
}

func TestValidate_UnknownComponentType(t *testing.T) {
	arch := validArch()
	arch.Components["assistant"] = spec.Component{Type: "debate", Model: "main"}

	result := Validate(arch)
	if result.Valid() {
		t.Fatal("expected invalid architecture for unknown component type")
	}
}

func TestValidate_OutputCannotBeSource(t *testing.T) {
	arch := validArch()
	arch.Connections = append(arch.Connections, spec.Connection{From: "output", To: "assistant.input"})

	result := Validate(arch)
	if result.Valid() {
		t.Fatal("expected invalid architecture: output used as a connection source")
	}
}

func TestValidate_InputCannotBeTarget(t *testing.T) {
	arch := validArch()
	arch.Connections = append(arch.Connections, spec.Connection{From: "assistant.output", To: "input"})

	result := Validate(arch)
	if result.Valid() {
		t.Fatal("expected invalid architecture: input used as a connection target")
	}
}

func TestValidate_CyclesAreAllowed(t *testing.T) {
	// ORH does not require a DAG: a component may feed back into an
	// earlier one. The runtime (scheduler) is responsible for bounding
	// runaway cycles, not the validator.
	arch := validArch()
	arch.Components["reviewer"] = spec.Component{Type: "agent", Model: "main", Prompt: "review"}
	arch.Connections = append(arch.Connections,
		spec.Connection{From: "assistant.output", To: "reviewer.input"},
		spec.Connection{From: "reviewer.output", To: "assistant.input"},
	)

	result := Validate(arch)
	if !result.Valid() {
		t.Fatalf("expected a cycle to pass validation, got errors: %v", result.Errors)
	}
}
