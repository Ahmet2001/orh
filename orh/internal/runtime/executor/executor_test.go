package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pertevniyalai/orh/internal/components"
	"github.com/pertevniyalai/orh/internal/memory"
	"github.com/pertevniyalai/orh/internal/spec"
	"github.com/pertevniyalai/orh/internal/tools"
)

const chainArch = `
apiVersion: orh/v1
name: simple-chain
models:
  main:
    provider: ollama
    name: qwen3
components:
  researcher:
    type: agent
    model: main
    prompt: research it
  writer:
    type: agent
    model: main
    prompt: write it up
connections:
  - from: input
    to: researcher.input
  - from: researcher.output
    to: writer.input
  - from: writer.output
    to: output
`

func writeArch(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.orh")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing test .orh file: %v", err)
	}
	return path
}

func TestRun_MultiComponentChain(t *testing.T) {
	// resolveProvider only knows the "ollama" provider, which needs a
	// live server, so this exercises everything up to (but not
	// including) the network call and confirms multi-component graphs
	// build and route correctly.
	path := writeArch(t, chainArch)

	_, err := Run(context.Background(), path, "topic", Options{})
	if err == nil {
		t.Fatal("expected an error calling the (unavailable) ollama server, got nil")
	}
}

func TestRun_ValidationError(t *testing.T) {
	path := writeArch(t, "apiVersion: orh/v1\nname: bad\n")

	_, err := Run(context.Background(), path, "hi", Options{})
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Fatalf("err = %T, want *ValidationError", err)
	}
}

func TestLoadAndValidate(t *testing.T) {
	path := writeArch(t, chainArch)

	arch, err := LoadAndValidate(context.Background(), path)
	if err != nil {
		t.Fatalf("LoadAndValidate() error = %v", err)
	}
	if len(arch.Components) != 2 {
		t.Errorf("len(arch.Components) = %d, want 2", len(arch.Components))
	}
}

func TestBuildComponents_AppendsSkillInstructionsToPrompt(t *testing.T) {
	arch := &spec.Architecture{
		Models: map[string]spec.Model{"main": {Provider: "ollama", Name: "qwen3"}},
		Components: map[string]spec.Component{
			"assistant": {
				Type:   "agent",
				Model:  "main",
				Prompt: "You are helpful.",
				Skills: []spec.SkillRef{{Alias: "review"}},
			},
		},
	}
	skillPrompts := map[string]string{"review": "Always check for security issues."}

	comps, err := buildComponents(context.Background(), arch, "", tools.NewRegistry(), skillPrompts, nil, "", nil)
	if err != nil {
		t.Fatalf("buildComponents() error = %v", err)
	}

	agent, ok := comps["assistant"].(*components.AgentComponent)
	if !ok {
		t.Fatalf("comps[assistant] = %T, want *components.AgentComponent", comps["assistant"])
	}

	want := "You are helpful.\n\nAlways check for security issues."
	if agent.Prompt != want {
		t.Errorf("agent.Prompt = %q, want %q", agent.Prompt, want)
	}
}

func TestBuildComponents_UndefinedSkillIsAnError(t *testing.T) {
	arch := &spec.Architecture{
		Models: map[string]spec.Model{"main": {Provider: "ollama", Name: "qwen3"}},
		Components: map[string]spec.Component{
			"assistant": {
				Type:   "agent",
				Model:  "main",
				Prompt: "You are helpful.",
				Skills: []spec.SkillRef{{Alias: "missing"}},
			},
		},
	}

	_, err := buildComponents(context.Background(), arch, "", tools.NewRegistry(), map[string]string{}, nil, "", nil)
	if err == nil {
		t.Fatal("expected an error for an undefined skill alias, got nil")
	}
}

func TestBuildComponents_InlineSkillNeedsNoPackage(t *testing.T) {
	arch := &spec.Architecture{
		Models: map[string]spec.Model{"main": {Provider: "ollama", Name: "qwen3"}},
		Components: map[string]spec.Component{
			"assistant": {
				Type:   "agent",
				Model:  "main",
				Prompt: "You are helpful.",
				Skills: []spec.SkillRef{{Inline: "Always answer in one word."}},
			},
		},
	}

	// No skillPrompts entries at all — an inline skill must not need one.
	comps, err := buildComponents(context.Background(), arch, "", tools.NewRegistry(), map[string]string{}, nil, "", nil)
	if err != nil {
		t.Fatalf("buildComponents() error = %v", err)
	}

	agent := comps["assistant"].(*components.AgentComponent)
	want := "You are helpful.\n\nAlways answer in one word."
	if agent.Prompt != want {
		t.Errorf("agent.Prompt = %q, want %q", agent.Prompt, want)
	}
}

func TestBuildComponents_MixedAliasAndInlineSkills(t *testing.T) {
	arch := &spec.Architecture{
		Models: map[string]spec.Model{"main": {Provider: "ollama", Name: "qwen3"}},
		Components: map[string]spec.Component{
			"assistant": {
				Type:   "agent",
				Model:  "main",
				Prompt: "You are helpful.",
				Skills: []spec.SkillRef{
					{Alias: "review"},
					{Inline: "Always answer in one word."},
				},
			},
		},
	}
	skillPrompts := map[string]string{"review": "Always check for security issues."}

	comps, err := buildComponents(context.Background(), arch, "", tools.NewRegistry(), skillPrompts, nil, "", nil)
	if err != nil {
		t.Fatalf("buildComponents() error = %v", err)
	}

	agent := comps["assistant"].(*components.AgentComponent)
	want := "You are helpful.\n\nAlways check for security issues.\n\nAlways answer in one word."
	if agent.Prompt != want {
		t.Errorf("agent.Prompt = %q, want %q", agent.Prompt, want)
	}
}

func TestBuildComponents_ConfiguresOptInMemory(t *testing.T) {
	arch := &spec.Architecture{
		Name:   "research-loop",
		Models: map[string]spec.Model{"main": {Provider: "ollama", Name: "qwen3"}},
		Components: map[string]spec.Component{
			"remembering": {Type: "agent", Model: "main", Prompt: "remember", Memory: true},
			"stateless":   {Type: "agent", Model: "main", Prompt: "forget"},
		},
	}
	store := memory.NewInMemoryStore()

	comps, err := buildComponents(context.Background(), arch, "", tools.NewRegistry(), nil, store, "experiment-7", nil)
	if err != nil {
		t.Fatalf("buildComponents() error = %v", err)
	}

	remembering := comps["remembering"].(*components.AgentComponent)
	if remembering.Memory != store {
		t.Fatal("memory-enabled component did not receive the run's memory store")
	}
	if remembering.SessionKey != "experiment-7:research-loop:remembering" {
		t.Errorf("SessionKey = %q", remembering.SessionKey)
	}

	stateless := comps["stateless"].(*components.AgentComponent)
	if stateless.Memory != nil {
		t.Fatal("stateless component unexpectedly received memory")
	}
}
