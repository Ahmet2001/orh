package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pertevniyalai/orh/internal/providers"
)

func TestGenerateUsesOpenAICompatibleChatCompletions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
		}
		var request completionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "test-model" || len(request.Messages) != 1 || request.Messages[0].Content != "hello" {
			t.Errorf("request = %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"world"}}]}`))
	}))
	defer server.Close()

	p := &Provider{BaseURL: server.URL + "/v1", APIKey: "secret", Model: "test-model", Client: server.Client()}
	got, err := p.Generate(context.Background(), providers.Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "world" {
		t.Errorf("Text = %q", got.Text)
	}
}

func TestChatConvertsToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request completionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Tools) != 1 || request.Tools[0].Function.Name != "lookup" {
			t.Errorf("tools = %+v", request.Tools)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call-1","type":"function","function":{"name":"lookup","arguments":"{\"id\":7}"}}]}}]}`))
	}))
	defer server.Close()

	p := &Provider{BaseURL: server.URL, Model: "test-model", Client: server.Client()}
	got, err := p.Chat(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "find it"}},
		Tools:    []providers.ToolSpec{{Name: "lookup", Parameters: json.RawMessage(`{"type":"object"}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Message.ToolCalls) != 1 || got.Message.ToolCalls[0].ID != "call-1" || got.Message.ToolCalls[0].Name != "lookup" {
		t.Fatalf("tool calls = %+v", got.Message.ToolCalls)
	}
	if string(got.Message.ToolCalls[0].Arguments) != `{"id":7}` {
		t.Errorf("arguments = %s", got.Message.ToolCalls[0].Arguments)
	}
}
