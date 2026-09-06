package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/pertevniyalai/orh/internal/mcp"
)

// newFakeMCPClient wires an mcp.Client to a goroutine that answers exactly
// one "tools/call" request with the given content/isError, so MCPTool can
// be tested without a real MCP server subprocess.
func newFakeMCPClient(t *testing.T, resultText string, isError bool) *mcp.Client {
	t.Helper()

	clientToServerR, clientToServerW := io.Pipe()
	serverToClientR, serverToClientW := io.Pipe()

	client := mcp.New(clientToServerW, serverToClientR, func() error {
		clientToServerW.Close()
		return nil
	})

	go func() {
		reader := bufio.NewReader(clientToServerR)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}
		var req map[string]any
		json.Unmarshal(line, &req)

		reply, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result": map[string]any{
				"content": []map[string]any{{"type": "text", "text": resultText}},
				"isError": isError,
			},
		})
		reply = append(reply, '\n')
		serverToClientW.Write(reply)
	}()

	return client
}

func TestMCPTool_Execute(t *testing.T) {
	client := newFakeMCPClient(t, "42 results", false)
	tool := NewMCPTool(client, "search", "Search the web", json.RawMessage(`{"type":"object"}`))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, json.RawMessage(`{"query":"AI news"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "42 results" {
		t.Errorf("Execute() = %q, want %q", result, "42 results")
	}
}

func TestMCPTool_Execute_ServerReportsError(t *testing.T) {
	client := newFakeMCPClient(t, "boom", true)
	tool := NewMCPTool(client, "search", "Search the web", json.RawMessage(`{"type":"object"}`))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := tool.Execute(ctx, json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected an error when the MCP server reports isError, got nil")
	}
}
