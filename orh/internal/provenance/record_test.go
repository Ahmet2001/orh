package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pertevniyalai/orh/internal/runtime/events"
	"github.com/pertevniyalai/orh/internal/spec"
)

func TestRecordCapturesResolvedProcess(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "main.orh")
	if err := os.WriteFile(source, []byte("architecture source"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New(source, "research question")
	arch := &spec.Architecture{
		Name: "scientist",
		Models: map[string]spec.Model{
			"reasoning": {Provider: "ollama", Name: "qwen3"},
		},
		Components:  map[string]spec.Component{"researcher": {Type: "agent", Model: "reasoning"}},
		Connections: []spec.Connection{{From: "input", To: "researcher.input"}},
	}
	deps := spec.Dependencies{"evidence": {Source: "github:org/evidence", Version: "abc123"}}
	if err := r.SetArchitecture(arch, "openai-compatible:test-model", deps); err != nil {
		t.Fatal(err)
	}
	r.AddStep("Executing", "researcher")
	r.Finish([]events.Event{{Source: "researcher", Port: "output", Payload: "answer"}}, nil)

	path := filepath.Join(dir, "records", "run.json")
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Record
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "completed" || got.Architecture.GraphHash == "" || got.Architecture.SourceHash == "" {
		t.Fatalf("incomplete record: %+v", got)
	}
	if got.Models["reasoning"].Provider != "openai-compatible" {
		t.Errorf("model override was not recorded: %+v", got.Models)
	}
	if len(got.Dependencies) != 1 || got.Dependencies[0].Commit != "abc123" {
		t.Errorf("pinned dependency missing: %+v", got.Dependencies)
	}
	if len(got.Trace) != 1 || len(got.Outputs) != 1 || got.Outputs[0].Hash == "" {
		t.Errorf("trace or output missing: %+v", got)
	}
}
