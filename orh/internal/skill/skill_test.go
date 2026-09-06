package skill

import "testing"

func TestParse_WithFrontmatter(t *testing.T) {
	data := []byte(`---
description: Applies a code review checklist
---

Review the code for:
- Correctness
- Security
`)

	s, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if s.Description != "Applies a code review checklist" {
		t.Errorf("Description = %q, want %q", s.Description, "Applies a code review checklist")
	}
	want := "Review the code for:\n- Correctness\n- Security"
	if s.Instructions != want {
		t.Errorf("Instructions = %q, want %q", s.Instructions, want)
	}
}

func TestParse_WithoutFrontmatter(t *testing.T) {
	data := []byte("Just do the thing carefully.\n")

	s, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if s.Description != "" {
		t.Errorf("Description = %q, want empty", s.Description)
	}
	if s.Instructions != "Just do the thing carefully." {
		t.Errorf("Instructions = %q, want %q", s.Instructions, "Just do the thing carefully.")
	}
}

func TestParse_UnclosedFrontmatter(t *testing.T) {
	data := []byte("---\ndescription: broken\nno closing delimiter\n")

	if _, err := Parse(data); err == nil {
		t.Fatal("expected an error for unclosed frontmatter, got nil")
	}
}

func TestValidate_RequiresInstructions(t *testing.T) {
	s, err := Parse([]byte("---\ndescription: nothing here\n---\n\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if errs := Validate(s); len(errs) == 0 {
		t.Fatal("expected a validation error for empty instructions, got none")
	}
}
