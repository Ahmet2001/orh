package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// protocolVersion is the MCP protocol date-version ORH speaks.
const protocolVersion = "2024-11-05"

// InitializeResult is the server's response to the initialize handshake.
type InitializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// Initialize performs the MCP handshake: sends "initialize", then the
// "notifications/initialized" notification the spec requires afterward.
func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
	params := map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "orh", "version": "0.1.0"},
	}

	raw, err := c.call(ctx, "initialize", params)
	if err != nil {
		return nil, fmt.Errorf("mcp initialize: %w", err)
	}

	var result InitializeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decoding initialize result: %w", err)
	}

	if err := c.notify("notifications/initialized", map[string]any{}); err != nil {
		return nil, fmt.Errorf("sending initialized notification: %w", err)
	}

	return &result, nil
}

// ToolInfo describes one tool an MCP server exposes.
type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ListTools calls "tools/list" and returns the server's tool catalog.
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	raw, err := c.call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("mcp tools/list: %w", err)
	}

	var result struct {
		Tools []ToolInfo `json:"tools"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decoding tools/list result: %w", err)
	}
	return result.Tools, nil
}

// ContentBlock is one piece of a tool call's result.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// CallToolResult is the server's response to a tool invocation.
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError"`
}

// CallTool calls "tools/call" for the named tool with the given arguments
// (typically a map[string]any or json.RawMessage of a JSON object).
func (c *Client) CallTool(ctx context.Context, name string, arguments any) (*CallToolResult, error) {
	params := map[string]any{"name": name, "arguments": arguments}

	raw, err := c.call(ctx, "tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("mcp tools/call %s: %w", name, err)
	}

	var result CallToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decoding tools/call result: %w", err)
	}
	return &result, nil
}
