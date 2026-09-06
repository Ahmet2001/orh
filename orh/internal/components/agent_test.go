package components

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pertevniyalai/orh/internal/commands"
	"github.com/pertevniyalai/orh/internal/providers"
	"github.com/pertevniyalai/orh/internal/runtime/events"
	"github.com/pertevniyalai/orh/internal/tools"
)

type fakeProvider struct{}

func (fakeProvider) Generate(ctx context.Context, req providers.Request) (providers.Response, error) {
	return providers.Response{Text: "echo: " + req.Prompt}, nil
}

func TestAgentComponent_Simple(t *testing.T) {
	a := &AgentComponent{Name: "a", Prompt: "hi", Provider: fakeProvider{}}

	cmds, err := a.Handle(context.Background(), events.Event{Payload: "hello"})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("len(cmds) = %d, want 1", len(cmds))
	}
}

// fakeChatProvider answers a scripted sequence of responses, one per call.
type fakeChatProvider struct {
	responses []providers.ChatResponse
	calls     int
}

func (f *fakeChatProvider) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	resp := f.responses[f.calls]
	f.calls++
	return resp, nil
}

type fakeTool struct{ calls int }

func (f *fakeTool) Name() string                 { return "search" }
func (f *fakeTool) Description() string          { return "search the web" }
func (f *fakeTool) InputSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (f *fakeTool) Execute(ctx context.Context, arguments json.RawMessage) (string, error) {
	f.calls++
	return "42 results", nil
}

func TestAgentComponent_ToolCallLoop(t *testing.T) {
	tool := &fakeTool{}
	chat := &fakeChatProvider{responses: []providers.ChatResponse{
		{Message: providers.Message{
			Role:      "assistant",
			ToolCalls: []providers.ToolCall{{Name: "search", Arguments: json.RawMessage(`{"query":"AI news"}`)}},
		}},
		{Message: providers.Message{Role: "assistant", Content: "Here is the answer."}},
	}}

	a := &AgentComponent{
		Name:         "researcher",
		Prompt:       "you are a researcher",
		ChatProvider: chat,
		Tools:        []tools.Tool{tool},
	}

	cmds, err := a.Handle(context.Background(), events.Event{Payload: "what's new in AI?"})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if tool.calls != 1 {
		t.Errorf("tool.calls = %d, want 1", tool.calls)
	}
	if len(cmds) != 1 {
		t.Fatalf("len(cmds) = %d, want 1", len(cmds))
	}
	emit, ok := cmds[0].(commands.EmitEvent)
	if !ok {
		t.Fatalf("cmds[0] = %T, want commands.EmitEvent", cmds[0])
	}
	if got := emit.Event.Text(); got != "Here is the answer." {
		t.Errorf("output = %q, want %q", got, "Here is the answer.")
	}
}

func TestAgentComponent_ToolCallLoop_UnknownTool(t *testing.T) {
	chat := &fakeChatProvider{responses: []providers.ChatResponse{
		{Message: providers.Message{
			Role:      "assistant",
			ToolCalls: []providers.ToolCall{{Name: "ghost", Arguments: json.RawMessage(`{}`)}},
		}},
		{Message: providers.Message{Role: "assistant", Content: "I couldn't find that tool."}},
	}}

	a := &AgentComponent{Name: "researcher", Prompt: "hi", ChatProvider: chat, Tools: []tools.Tool{&fakeTool{}}}

	cmds, err := a.Handle(context.Background(), events.Event{Payload: "hi"})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("len(cmds) = %d, want 1", len(cmds))
	}
}

func TestAgentComponent_ToolCallLoop_ExceedsMaxIterations(t *testing.T) {
	loopForever := providers.ChatResponse{Message: providers.Message{
		Role:      "assistant",
		ToolCalls: []providers.ToolCall{{Name: "search", Arguments: json.RawMessage(`{}`)}},
	}}
	chat := &fakeChatProvider{responses: []providers.ChatResponse{loopForever, loopForever, loopForever}}

	a := &AgentComponent{
		Name:              "researcher",
		Prompt:            "hi",
		ChatProvider:      chat,
		Tools:             []tools.Tool{&fakeTool{}},
		MaxToolIterations: 3,
	}

	if _, err := a.Handle(context.Background(), events.Event{Payload: "hi"}); err == nil {
		t.Fatal("expected an error when the tool-calling loop never terminates, got nil")
	}
}
