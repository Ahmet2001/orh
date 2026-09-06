package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pertevniyalai/orh/internal/providers"
)

type chatToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type chatToolSpec struct {
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type chatToolCall struct {
	Function chatToolCallFunction `json:"function"`
}

type chatMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content,omitempty"`
	ToolCalls []chatToolCall `json:"tool_calls,omitempty"`
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []chatMessage  `json:"messages"`
	Tools    []chatToolSpec `json:"tools,omitempty"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
	Error   string      `json:"error"`
}

// Chat implements providers.ToolCallingProvider against Ollama's native
// /api/chat endpoint, which supports the "tools" field for capable models.
func (p *Provider) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	body, err := json.Marshal(chatRequest{
		Model:    p.Model,
		Messages: toChatMessages(req.Messages),
		Tools:    toChatTools(req.Tools),
		Stream:   false,
		Options:  p.Options,
	})
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("encoding ollama chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("building ollama chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("calling ollama at %s: %w", p.BaseURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("reading ollama chat response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return providers.ChatResponse{}, fmt.Errorf("ollama returned %s: %s", resp.Status, string(data))
	}

	var out chatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return providers.ChatResponse{}, fmt.Errorf("decoding ollama chat response: %w", err)
	}
	if out.Error != "" {
		return providers.ChatResponse{}, fmt.Errorf("ollama error: %s", out.Error)
	}

	return providers.ChatResponse{Message: fromChatMessage(out.Message)}, nil
}

func toChatMessages(msgs []providers.Message) []chatMessage {
	out := make([]chatMessage, len(msgs))
	for i, m := range msgs {
		calls := make([]chatToolCall, len(m.ToolCalls))
		for j, c := range m.ToolCalls {
			calls[j] = chatToolCall{Function: chatToolCallFunction{Name: c.Name, Arguments: c.Arguments}}
		}
		out[i] = chatMessage{Role: m.Role, Content: m.Content, ToolCalls: calls}
	}
	return out
}

func toChatTools(specs []providers.ToolSpec) []chatToolSpec {
	if len(specs) == 0 {
		return nil
	}
	out := make([]chatToolSpec, len(specs))
	for i, s := range specs {
		out[i] = chatToolSpec{
			Type: "function",
			Function: chatToolFunction{
				Name:        s.Name,
				Description: s.Description,
				Parameters:  s.Parameters,
			},
		}
	}
	return out
}

func fromChatMessage(m chatMessage) providers.Message {
	calls := make([]providers.ToolCall, len(m.ToolCalls))
	for i, c := range m.ToolCalls {
		calls[i] = providers.ToolCall{Name: c.Function.Name, Arguments: c.Function.Arguments}
	}
	return providers.Message{Role: m.Role, Content: m.Content, ToolCalls: calls}
}
