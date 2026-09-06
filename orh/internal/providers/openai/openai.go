// Package openai implements ORH providers against the OpenAI-compatible
// /v1/chat/completions protocol. The endpoint and credential are configured
// with OPENAI_BASE_URL and OPENAI_API_KEY.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pertevniyalai/orh/internal/providers"
)

const defaultBaseURL = "https://api.openai.com/v1"

type Provider struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func New(model string) *Provider {
	baseURL := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Provider{
		BaseURL: baseURL,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   model,
		Client:  http.DefaultClient,
	}
}

type functionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type toolSpec struct {
	Type     string       `json:"type"`
	Function functionSpec `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type completionRequest struct {
	Model    string     `json:"model"`
	Messages []message  `json:"messages"`
	Tools    []toolSpec `json:"tools,omitempty"`
}

type completionResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *Provider) Generate(ctx context.Context, req providers.Request) (providers.Response, error) {
	resp, err := p.Chat(ctx, providers.ChatRequest{Messages: []providers.Message{{Role: "user", Content: req.Prompt}}})
	if err != nil {
		return providers.Response{}, err
	}
	return providers.Response{Text: resp.Message.Content}, nil
}

func (p *Provider) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	body, err := json.Marshal(completionRequest{
		Model:    p.Model,
		Messages: toMessages(req.Messages),
		Tools:    toTools(req.Tools),
	})
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("encoding OpenAI-compatible request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("building OpenAI-compatible request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("calling OpenAI-compatible endpoint at %s: %w", p.BaseURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("reading OpenAI-compatible response: %w", err)
	}

	var out completionResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return providers.ChatResponse{}, fmt.Errorf("decoding OpenAI-compatible response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(data))
		if out.Error != nil && out.Error.Message != "" {
			message = out.Error.Message
		}
		return providers.ChatResponse{}, fmt.Errorf("OpenAI-compatible endpoint returned %s: %s", resp.Status, message)
	}
	if out.Error != nil && out.Error.Message != "" {
		return providers.ChatResponse{}, fmt.Errorf("OpenAI-compatible error: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return providers.ChatResponse{}, fmt.Errorf("OpenAI-compatible response contained no choices")
	}

	return providers.ChatResponse{Message: fromMessage(out.Choices[0].Message)}, nil
}

func toMessages(input []providers.Message) []message {
	output := make([]message, len(input))
	for i, item := range input {
		calls := make([]toolCall, len(item.ToolCalls))
		for j, call := range item.ToolCalls {
			calls[j] = toolCall{
				ID:   call.ID,
				Type: "function",
				Function: functionCall{
					Name:      call.Name,
					Arguments: string(call.Arguments),
				},
			}
		}
		output[i] = message{Role: item.Role, Content: item.Content, ToolCalls: calls, ToolCallID: item.ToolCallID}
	}
	return output
}

func toTools(input []providers.ToolSpec) []toolSpec {
	if len(input) == 0 {
		return nil
	}
	output := make([]toolSpec, len(input))
	for i, item := range input {
		output[i] = toolSpec{
			Type: "function",
			Function: functionSpec{
				Name:        item.Name,
				Description: item.Description,
				Parameters:  item.Parameters,
			},
		}
	}
	return output
}

func fromMessage(input message) providers.Message {
	calls := make([]providers.ToolCall, len(input.ToolCalls))
	for i, call := range input.ToolCalls {
		arguments := json.RawMessage(call.Function.Arguments)
		if len(arguments) == 0 {
			arguments = json.RawMessage(`{}`)
		}
		calls[i] = providers.ToolCall{ID: call.ID, Name: call.Function.Name, Arguments: arguments}
	}
	return providers.Message{Role: input.Role, Content: input.Content, ToolCalls: calls, ToolCallID: input.ToolCallID}
}
