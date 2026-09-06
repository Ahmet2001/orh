package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pertevniyalai/orh/internal/mcp"
)

// MCPTool adapts one tool exposed by an MCP server to the Tool interface.
type MCPTool struct {
	client      *mcp.Client
	name        string
	description string
	inputSchema json.RawMessage
}

// NewMCPTool wraps one MCP-server-exposed tool as a Tool.
func NewMCPTool(client *mcp.Client, name, description string, inputSchema json.RawMessage) *MCPTool {
	return &MCPTool{client: client, name: name, description: description, inputSchema: inputSchema}
}

func (t *MCPTool) Name() string                 { return t.name }
func (t *MCPTool) Description() string          { return t.description }
func (t *MCPTool) InputSchema() json.RawMessage { return t.inputSchema }

// Execute implements Tool by calling the tool on the underlying MCP server.
func (t *MCPTool) Execute(ctx context.Context, arguments json.RawMessage) (string, error) {
	var args any
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &args); err != nil {
			return "", fmt.Errorf("tool %q: decoding arguments: %w", t.name, err)
		}
	}

	result, err := t.client.CallTool(ctx, t.name, args)
	if err != nil {
		return "", fmt.Errorf("tool %q: %w", t.name, err)
	}

	var text strings.Builder
	for _, c := range result.Content {
		text.WriteString(c.Text)
	}

	if result.IsError {
		return "", fmt.Errorf("tool %q returned an error: %s", t.name, text.String())
	}
	return text.String(), nil
}
