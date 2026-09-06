package ollama

import (
	"os"
	"strconv"
)

// optionsFromEnv provides experiment-level control without coupling the ORH
// architecture graph to Ollama-specific request fields. Unset values preserve
// Ollama defaults.
func optionsFromEnv() map[string]any {
	options := map[string]any{}
	if raw := os.Getenv("ORH_OLLAMA_NUM_PREDICT"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			options["num_predict"] = value
		}
	}
	if raw := os.Getenv("ORH_OLLAMA_TEMPERATURE"); raw != "" {
		if value, err := strconv.ParseFloat(raw, 64); err == nil && value >= 0 {
			options["temperature"] = value
		}
	}
	if raw := os.Getenv("ORH_OLLAMA_SEED"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			options["seed"] = value
		}
	}
	if len(options) == 0 {
		return nil
	}
	return options
}
