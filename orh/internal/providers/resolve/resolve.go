// Package resolve maps a "provider:model" pair from a .orh model slot to a
// concrete providers implementation. It exists as its own package (rather
// than living in internal/providers itself) because it must import
// providers/ollama, and internal/providers/ollama already imports
// internal/providers — putting this here avoids that cycle while still
// letting both the executor and WASM nodes share one place that knows about
// supported provider names.
package resolve

import (
	"fmt"

	"github.com/pertevniyalai/orh/internal/providers"
	"github.com/pertevniyalai/orh/internal/providers/ollama"
	"github.com/pertevniyalai/orh/internal/providers/openai"
)

// Provider resolves providerName/modelName to a ModelProvider.
func Provider(providerName, modelName string) (providers.ModelProvider, error) {
	switch providerName {
	case "ollama":
		if modelName == "" {
			return nil, fmt.Errorf("ollama provider requires a model name")
		}
		return ollama.New(modelName), nil
	case "openai", "openai-compatible":
		if modelName == "" {
			return nil, fmt.Errorf("%s provider requires a model name", providerName)
		}
		return openai.New(modelName), nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", providerName)
	}
}

// ChatProvider resolves providerName/modelName to a ToolCallingProvider.
func ChatProvider(providerName, modelName string) (providers.ToolCallingProvider, error) {
	switch providerName {
	case "ollama":
		if modelName == "" {
			return nil, fmt.Errorf("ollama provider requires a model name")
		}
		return ollama.New(modelName), nil
	case "openai", "openai-compatible":
		if modelName == "" {
			return nil, fmt.Errorf("%s provider requires a model name", providerName)
		}
		return openai.New(modelName), nil
	default:
		return nil, fmt.Errorf("unsupported provider %q for tool-calling", providerName)
	}
}
