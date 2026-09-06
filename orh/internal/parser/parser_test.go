package parser

import "testing"

func TestParse(t *testing.T) {
	data := []byte(`
apiVersion: orh/v1
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
`)

	arch, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if arch.Name != "hello-agent" {
		t.Errorf("Name = %q, want %q", arch.Name, "hello-agent")
	}
	if len(arch.Components) != 1 {
		t.Errorf("len(Components) = %d, want 1", len(arch.Components))
	}
	if len(arch.Connections) != 2 {
		t.Errorf("len(Connections) = %d, want 2", len(arch.Connections))
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	if _, err := Parse([]byte("not: valid: yaml: [")); err == nil {
		t.Fatal("Parse() expected error for invalid yaml, got nil")
	}
}

func TestParse_SkillsAliasAndInline(t *testing.T) {
	data := []byte(`
apiVersion: orh/v1
name: skill-agent
models:
  main:
    provider: ollama
    name: qwen3
components:
  assistant:
    type: agent
    model: main
    prompt: hi
    skills:
      - review
      - text: |
          Always answer in one word.
        description: brief
connections:
  - from: input
    to: assistant.input
  - from: assistant.output
    to: output
`)

	arch, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	skills := arch.Components["assistant"].Skills
	if len(skills) != 2 {
		t.Fatalf("len(skills) = %d, want 2", len(skills))
	}
	if skills[0].Alias != "review" || skills[0].Inline != "" {
		t.Errorf("skills[0] = %+v, want an alias-only ref to %q", skills[0], "review")
	}
	if skills[1].Alias != "" || skills[1].Inline != "Always answer in one word.\n" {
		t.Errorf("skills[1] = %+v, want an inline ref", skills[1])
	}
	if skills[1].Description != "brief" {
		t.Errorf("skills[1].Description = %q, want %q", skills[1].Description, "brief")
	}
}

func TestParse_InlineSkillRequiresText(t *testing.T) {
	data := []byte(`
apiVersion: orh/v1
name: skill-agent
components:
  assistant:
    type: agent
    model: main
    prompt: hi
    skills:
      - description: missing text field
`)

	if _, err := Parse(data); err == nil {
		t.Fatal("Parse() expected error for an inline skill with no text field, got nil")
	}
}
