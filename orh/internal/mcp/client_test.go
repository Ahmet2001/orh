package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

// fakeServer is a minimal in-process MCP server driving a Client through an
// io.Pipe pair, so tests don't need a real subprocess.
type fakeServer struct {
	toClient   *io.PipeWriter
	fromClient *bufio.Reader
}

func newFakeServer(t *testing.T) (*Client, *fakeServer) {
	t.Helper()

	clientToServerR, clientToServerW := io.Pipe()
	serverToClientR, serverToClientW := io.Pipe()

	client := New(clientToServerW, serverToClientR, func() error {
		clientToServerW.Close()
		return nil
	})

	server := &fakeServer{
		toClient:   serverToClientW,
		fromClient: bufio.NewReader(clientToServerR),
	}

	return client, server
}

func (s *fakeServer) readRequest(t *testing.T) map[string]any {
	t.Helper()
	line, err := s.fromClient.ReadBytes('\n')
	if err != nil {
		t.Fatalf("reading request: %v", err)
	}
	var req map[string]any
	if err := json.Unmarshal(line, &req); err != nil {
		t.Fatalf("decoding request: %v", err)
	}
	return req
}

func (s *fakeServer) reply(t *testing.T, id float64, result any) {
	t.Helper()
	data, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	if err != nil {
		t.Fatalf("encoding reply: %v", err)
	}
	data = append(data, '\n')
	if _, err := s.toClient.Write(data); err != nil {
		t.Fatalf("writing reply: %v", err)
	}
}

func TestClient_InitializeListCall(t *testing.T) {
	client, server := newFakeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)

		req := server.readRequest(t)
		if req["method"] != "initialize" {
			t.Errorf("first request method = %v, want initialize", req["method"])
		}
		server.reply(t, req["id"].(float64), map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]any{"name": "fake", "version": "1.0"},
		})

		// notifications/initialized has no id and expects no reply.
		notif := server.readRequest(t)
		if notif["method"] != "notifications/initialized" {
			t.Errorf("second message method = %v, want notifications/initialized", notif["method"])
		}

		req = server.readRequest(t)
		if req["method"] != "tools/list" {
			t.Errorf("third request method = %v, want tools/list", req["method"])
		}
		server.reply(t, req["id"].(float64), map[string]any{
			"tools": []map[string]any{
				{"name": "search", "description": "Search the web", "inputSchema": map[string]any{"type": "object"}},
			},
		})

		req = server.readRequest(t)
		if req["method"] != "tools/call" {
			t.Errorf("fourth request method = %v, want tools/call", req["method"])
		}
		server.reply(t, req["id"].(float64), map[string]any{
			"content": []map[string]any{{"type": "text", "text": "42 results"}},
			"isError": false,
		})
	}()

	info, err := client.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if info.ServerInfo.Name != "fake" {
		t.Errorf("ServerInfo.Name = %q, want %q", info.ServerInfo.Name, "fake")
	}

	toolList, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(toolList) != 1 || toolList[0].Name != "search" {
		t.Fatalf("ListTools() = %+v, want one tool named search", toolList)
	}

	result, err := client.CallTool(ctx, "search", map[string]any{"query": "AI news"})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Text != "42 results" {
		t.Fatalf("CallTool() = %+v, want content [42 results]", result)
	}

	<-done
}

func TestClient_ServerError(t *testing.T) {
	client, server := newFakeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		req := server.readRequest(t)
		data, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"error":   map[string]any{"code": -32601, "message": "method not found"},
		})
		data = append(data, '\n')
		server.toClient.Write(data)
	}()

	if _, err := client.ListTools(ctx); err == nil {
		t.Fatal("expected an error from a JSON-RPC error response, got nil")
	}
}
