package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeTool struct{ name string }

func (f fakeTool) Name() string                 { return f.name }
func (f fakeTool) Description() string          { return "a fake tool" }
func (f fakeTool) InputSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (f fakeTool) Execute(ctx context.Context, arguments json.RawMessage) (string, error) {
	return "ok", nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register("web.search", fakeTool{name: "search"})

	tool, ok := r.Get("web.search")
	if !ok {
		t.Fatal("expected web.search to be registered")
	}
	if tool.Name() != "search" {
		t.Errorf("tool.Name() = %q, want %q", tool.Name(), "search")
	}

	if _, ok := r.Get("web.missing"); ok {
		t.Error("expected web.missing to be absent")
	}
}

func TestRegistry_Names(t *testing.T) {
	r := NewRegistry()
	r.Register("web.search", fakeTool{name: "search"})
	r.Register("github.read", fakeTool{name: "read"})

	names := r.Names()
	if len(names) != 2 || names[0] != "github.read" || names[1] != "web.search" {
		t.Errorf("Names() = %v, want sorted [github.read web.search]", names)
	}
}
