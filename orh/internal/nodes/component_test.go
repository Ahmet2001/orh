package nodes

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pertevniyalai/orh/internal/spec"
	"github.com/pertevniyalai/orh/internal/tools"
)

// fakeTool is a minimal tools.Tool for testing NodeComponent.CallTool.
type fakeTool struct {
	name   string
	result string
	err    error
	called bool
	gotArg json.RawMessage
}

func (f *fakeTool) Name() string                 { return f.name }
func (f *fakeTool) Description() string          { return "fake" }
func (f *fakeTool) InputSchema() json.RawMessage { return json.RawMessage(`{}`) }
func (f *fakeTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	f.called = true
	f.gotArg = args
	return f.result, f.err
}

func TestNodeComponent_CallModel_DeniedWhenNotInAllowList(t *testing.T) {
	n := &NodeComponent{
		Name:        "router",
		Models:      map[string]spec.Model{"reasoning": {Provider: "ollama", Name: "qwen3"}},
		Permissions: spec.NodePermissions{Models: spec.NodeModelsPermission{Allow: []string{"fast"}}},
	}

	_, err := n.CallModel(context.Background(), "reasoning", "hi")
	if err == nil {
		t.Fatal("expected an error for a model slot not in permissions.models.allow, got nil")
	}
}

func TestNodeComponent_CallModel_DeniedWhenAllowListEmpty(t *testing.T) {
	n := &NodeComponent{
		Name:   "router",
		Models: map[string]spec.Model{"reasoning": {Provider: "ollama", Name: "qwen3"}},
	}

	_, err := n.CallModel(context.Background(), "reasoning", "hi")
	if err == nil {
		t.Fatal("expected an error when permissions.models.allow is empty, got nil")
	}
}

func TestNodeComponent_CallModel_UndefinedSlotEvenIfPermitted(t *testing.T) {
	n := &NodeComponent{
		Name:        "router",
		Models:      map[string]spec.Model{},
		Permissions: spec.NodePermissions{Models: spec.NodeModelsPermission{Allow: []string{"reasoning"}}},
	}

	_, err := n.CallModel(context.Background(), "reasoning", "hi")
	if err == nil {
		t.Fatal("expected an error for a permitted but undefined model slot, got nil")
	}
}

func TestNodeComponent_CallTool_DeniedWhenNotInAllowList(t *testing.T) {
	reg := tools.NewRegistry()
	ft := &fakeTool{name: "search", result: "ok"}
	reg.Register("web.search", ft)

	n := &NodeComponent{
		Name:        "router",
		Tools:       reg,
		Permissions: spec.NodePermissions{Tools: spec.NodeToolsPermission{Allow: []string{"web.other"}}},
	}

	_, err := n.CallTool(context.Background(), "web.search", `{}`)
	if err == nil {
		t.Fatal("expected an error for a tool not in permissions.tools.allow, got nil")
	}
	if ft.called {
		t.Error("tool.Execute should never be invoked when permission is denied")
	}
}

func TestNodeComponent_CallTool_AllowedRunsTheTool(t *testing.T) {
	reg := tools.NewRegistry()
	ft := &fakeTool{name: "search", result: "42 results"}
	reg.Register("web.search", ft)

	n := &NodeComponent{
		Name:        "router",
		Tools:       reg,
		Permissions: spec.NodePermissions{Tools: spec.NodeToolsPermission{Allow: []string{"web.search"}}},
	}

	out, err := n.CallTool(context.Background(), "web.search", `{"q":"orh"}`)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if out != "42 results" {
		t.Errorf("CallTool() = %q, want %q", out, "42 results")
	}
	if !ft.called {
		t.Error("expected the tool to be executed")
	}
	if string(ft.gotArg) != `{"q":"orh"}` {
		t.Errorf("tool received arguments %s, want %s", ft.gotArg, `{"q":"orh"}`)
	}
}

func TestNodeComponent_CallTool_UndefinedToolEvenIfPermitted(t *testing.T) {
	n := &NodeComponent{
		Name:        "router",
		Tools:       tools.NewRegistry(),
		Permissions: spec.NodePermissions{Tools: spec.NodeToolsPermission{Allow: []string{"web.search"}}},
	}

	_, err := n.CallTool(context.Background(), "web.search", `{}`)
	if err == nil {
		t.Fatal("expected an error for a permitted but unregistered tool, got nil")
	}
}
