// Package ollama implements the providers.ModelProvider interface against a
// local Ollama server.
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

const defaultBaseURL = "http://localhost:11434"

// Provider calls a model served by Ollama's HTTP API.
type Provider struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

// New builds an Ollama provider for the given model name, targeting the
// default local Ollama server.
func New(model string) *Provider {
	return &Provider{
		BaseURL: defaultBaseURL,
		Model:   model,
		Client:  http.DefaultClient,
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// Generate implements providers.ModelProvider.
func (p *Provider) Generate(ctx context.Context, req providers.Request) (providers.Response, error) {
	body, err := json.Marshal(generateRequest{
		Model:  p.Model,
		Prompt: req.Prompt,
		Stream: false,
	})
	if err != nil {
		return providers.Response{}, fmt.Errorf("encoding ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return providers.Response{}, fmt.Errorf("building ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return providers.Response{}, fmt.Errorf("calling ollama at %s: %w", p.BaseURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return providers.Response{}, fmt.Errorf("reading ollama response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return providers.Response{}, fmt.Errorf("ollama returned %s: %s", resp.Status, string(data))
	}

	var out generateResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return providers.Response{}, fmt.Errorf("decoding ollama response: %w", err)
	}
	if out.Error != "" {
		return providers.Response{}, fmt.Errorf("ollama error: %s", out.Error)
	}

	return providers.Response{Text: out.Response}, nil
}
