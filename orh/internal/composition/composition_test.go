package composition

import (
	"context"
	"testing"

	"github.com/pertevniyalai/orh/internal/pkg"
	"github.com/pertevniyalai/orh/internal/spec"
	"github.com/pertevniyalai/orh/internal/validator"
)

// fakeResolver serves a fixed set of architectures in memory, keyed by
// "source@version" (or bare "source" if version is empty) — no network.
type fakeResolver struct {
	packages map[string]*spec.Architecture
}

func (f *fakeResolver) Resolve(ctx context.Context, source, version string) (*pkg.Package, error) {
	key := source
	if version != "" {
		key += "@" + version
	}
	arch, ok := f.packages[key]
	if !ok {
		return nil, errNotFound(key)
	}
	return &pkg.Package{Entrypoint: arch}, nil
}

type errNotFound string

func (e errNotFound) Error() string { return "package not found: " + string(e) }

// legacyAgentArch is a Faz1/2/3-style package with no declared inputs or
// outputs contract — just the default single "input"/"output" boundary.
func legacyAgentArch(name string) *spec.Architecture {
	return &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       name,
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

func TestFlatten_NoOrchestrationComponentsIsNoOp(t *testing.T) {
	arch := legacyAgentArch("solo")
	r := &fakeResolver{}

	out, err := Flatten(context.Background(), arch, r)
	if err != nil {
		t.Fatalf("Flatten() error = %v", err)
	}
	if len(out.Components) != 1 || len(out.Connections) != 2 {
		t.Fatalf("expected the architecture to pass through unchanged, got %+v", out)
	}
}

func TestFlatten_LegacyPackageAsOrchestrationComponent(t *testing.T) {
	// A parent imports a legacy (no ports declared) single-agent package as
	// a bare orchestration component — its default input/output boundary
	// should be addressable as "search" with no port.
	r := &fakeResolver{packages: map[string]*spec.Architecture{
		"github:test/search-agent@v1.0.0": legacyAgentArch("search-agent"),
	}}

	parent := &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "wrapper",
		Components: map[string]spec.Component{
			"search": {Type: "orchestration", Source: "github:test/search-agent", Version: "v1.0.0"},
		},
		Connections: []spec.Connection{
			{From: "input", To: "search"},
			{From: "search", To: "output"},
		},
	}

	out, err := Flatten(context.Background(), parent, r)
	if err != nil {
		t.Fatalf("Flatten() error = %v", err)
	}

	if _, ok := out.Components["search.assistant"]; !ok {
		t.Fatalf("expected prefixed component %q, got components %+v", "search.assistant", out.Components)
	}
	if _, ok := out.Models["search.main"]; !ok {
		t.Fatalf("expected prefixed model %q, got models %+v", "search.main", out.Models)
	}

	if result := validator.Validate(out); !result.Valid() {
		t.Fatalf("flattened architecture failed validation: %v", result.Errors)
	}

	want := map[spec.Connection]bool{
		{From: "input", To: "search.assistant.input"}:   true,
		{From: "search.assistant.output", To: "output"}: true,
	}
	if len(out.Connections) != len(want) {
		t.Fatalf("Connections = %+v, want %d entries matching %+v", out.Connections, len(want), want)
	}
	for _, c := range out.Connections {
		if !want[c] {
			t.Errorf("unexpected connection %+v", c)
		}
	}
}

func TestFlatten_NamedPortsChain(t *testing.T) {
	searchAgent := &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "search-agent",
		Inputs:     map[string]spec.Port{"query": {Type: "string"}},
		Outputs:    map[string]spec.Port{"documents": {Type: "string"}},
		Models:     map[string]spec.Model{"main": {Provider: "ollama", Name: "qwen3"}},
		Components: map[string]spec.Component{
			"crawler": {Type: "agent", Model: "main", Prompt: "search"},
		},
		Connections: []spec.Connection{
			{From: "input.query", To: "crawler.input"},
			{From: "crawler.output", To: "output.documents"},
		},
	}

	summarizer := &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "summarizer",
		Inputs:     map[string]spec.Port{"documents": {Type: "string"}},
		Outputs:    map[string]spec.Port{"summary": {Type: "string"}},
		Models:     map[string]spec.Model{"main": {Provider: "ollama", Name: "qwen3"}},
		Components: map[string]spec.Component{
			"writer": {Type: "agent", Model: "main", Prompt: "summarize"},
		},
		Connections: []spec.Connection{
			{From: "input.documents", To: "writer.input"},
			{From: "writer.output", To: "output.summary"},
		},
	}

	r := &fakeResolver{packages: map[string]*spec.Architecture{
		"github:test/search-agent@v1.0.0": searchAgent,
		"github:test/summarizer@v1.0.0":   summarizer,
	}}

	researchSystem := &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "research-system",
		Components: map[string]spec.Component{
			"search": {Type: "orchestration", Source: "github:test/search-agent", Version: "v1.0.0"},
			"digest": {Type: "orchestration", Source: "github:test/summarizer", Version: "v1.0.0"},
		},
		Connections: []spec.Connection{
			{From: "input", To: "search.query"},
			{From: "search.documents", To: "digest.documents"},
			{From: "digest.summary", To: "output"},
		},
	}

	out, err := Flatten(context.Background(), researchSystem, r)
	if err != nil {
		t.Fatalf("Flatten() error = %v", err)
	}

	if result := validator.Validate(out); !result.Valid() {
		t.Fatalf("flattened architecture failed validation: %v", result.Errors)
	}

	want := map[spec.Connection]bool{
		{From: "input", To: "search.crawler.input"}:                true,
		{From: "search.crawler.output", To: "digest.writer.input"}: true,
		{From: "digest.writer.output", To: "output"}:               true,
	}
	if len(out.Connections) != len(want) {
		t.Fatalf("Connections = %+v, want %d entries matching %+v", out.Connections, len(want), want)
	}
	for _, c := range out.Connections {
		if !want[c] {
			t.Errorf("unexpected connection %+v", c)
		}
	}
}

func TestFlatten_NestedOrchestration(t *testing.T) {
	inner := legacyAgentArch("inner")
	middle := &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "middle",
		Components: map[string]spec.Component{
			"leaf": {Type: "orchestration", Source: "github:test/inner", Version: "v1.0.0"},
		},
		Connections: []spec.Connection{
			{From: "input", To: "leaf"},
			{From: "leaf", To: "output"},
		},
	}

	r := &fakeResolver{packages: map[string]*spec.Architecture{
		"github:test/inner@v1.0.0":  inner,
		"github:test/middle@v1.0.0": middle,
	}}

	top := &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "top",
		Components: map[string]spec.Component{
			"mid": {Type: "orchestration", Source: "github:test/middle", Version: "v1.0.0"},
		},
		Connections: []spec.Connection{
			{From: "input", To: "mid"},
			{From: "mid", To: "output"},
		},
	}

	out, err := Flatten(context.Background(), top, r)
	if err != nil {
		t.Fatalf("Flatten() error = %v", err)
	}
	if _, ok := out.Components["mid.leaf.assistant"]; !ok {
		t.Fatalf("expected double-prefixed component %q, got %+v", "mid.leaf.assistant", out.Components)
	}
	if result := validator.Validate(out); !result.Valid() {
		t.Fatalf("flattened architecture failed validation: %v", result.Errors)
	}
}

func TestFlatten_CircularDependencyDetected(t *testing.T) {
	// top (the local entrypoint, no source identity of its own) imports a,
	// which imports b, which imports a again — a real cycle among fetched
	// packages, which is what the "seen" chain is built to catch.
	a := &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "a",
		Components: map[string]spec.Component{
			"b": {Type: "orchestration", Source: "github:test/b", Version: "v1.0.0"},
		},
		Connections: []spec.Connection{{From: "input", To: "b"}, {From: "b", To: "output"}},
	}
	b := &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "b",
		Components: map[string]spec.Component{
			"a": {Type: "orchestration", Source: "github:test/a", Version: "v1.0.0"},
		},
		Connections: []spec.Connection{{From: "input", To: "a"}, {From: "a", To: "output"}},
	}
	top := &spec.Architecture{
		APIVersion: "orh/v1",
		Name:       "top",
		Components: map[string]spec.Component{
			"a": {Type: "orchestration", Source: "github:test/a", Version: "v1.0.0"},
		},
		Connections: []spec.Connection{{From: "input", To: "a"}, {From: "a", To: "output"}},
	}

	r := &fakeResolver{packages: map[string]*spec.Architecture{
		"github:test/a@v1.0.0": a,
		"github:test/b@v1.0.0": b,
	}}

	if _, err := Flatten(context.Background(), top, r); err == nil {
		t.Fatal("expected a circular dependency error, got nil")
	}
}
