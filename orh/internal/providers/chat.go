package providers

import (
	"context"
	"encoding/json"
)

// ToolSpec describes one tool available to the model during a Chat call.
type ToolSpec struct {
	Name        string
	Description string
	// Parameters is a JSON Schema object describing the tool's arguments.
	Parameters json.RawMessage
}

// ToolCall is a request from the model to invoke one tool.
type ToolCall struct {
	// ID identifies this call so its result can be matched back to it via
	// Message.ToolCallID. Providers that don't have a native call ID (e.g.
	// Ollama's native chat API) may leave this empty.
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Message is one turn in a Chat conversation.
type Message struct {
	// Role is "system", "user", "assistant", or "tool".
	Role string
	// Content is the message text (the assistant's answer, or a tool's
	// result when Role is "tool").
	Content string
	// ToolCalls is set on an assistant message that wants to invoke tools
	// instead of (or before) giving a final answer.
	ToolCalls []ToolCall
	// ToolCallID identifies which ToolCall a "tool" role message is the
	// result of.
	ToolCallID string
}

// ChatRequest is a multi-turn conversation, optionally with tools the model
// may call.
type ChatRequest struct {
	Messages []Message
	Tools    []ToolSpec
}

// ChatResponse is the model's next turn.
type ChatResponse struct {
	Message Message
}

// ToolCallingProvider is implemented by providers that support multi-turn
// chat with tool/function calling, in addition to the simpler ModelProvider
// (single prompt in, text out) interface. A component only needs this when
// it actually has tools configured.
type ToolCallingProvider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
