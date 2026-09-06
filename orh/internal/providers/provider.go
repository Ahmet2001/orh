// Package providers defines the model provider abstraction that components
// use to call a model, independent of which backend serves it.
package providers

import "context"

// Request is a model generation request.
type Request struct {
	Prompt string
}

// Response is a model generation result.
type Response struct {
	Text string
}

// ModelProvider generates a response for a request. Implementations adapt
// this to a specific backend (Ollama, vLLM, an OpenAI-compatible API, ...).
type ModelProvider interface {
	Generate(ctx context.Context, request Request) (Response, error)
}
