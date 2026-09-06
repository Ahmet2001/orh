// Package tools defines the Tool primitive agents call, and a Registry
// components look up tools by name in.
package tools

import (
	"context"
	"encoding/json"
)

// Tool is a single callable capability an agent can invoke mid-reasoning.
type Tool interface {
	Name() string
	Description() string
	// InputSchema is a JSON Schema describing the tool's arguments, handed
	// to the model so it knows how to call the tool.
	InputSchema() json.RawMessage
	// Execute runs the tool with the given arguments (typically a JSON
	// object) and returns its result as text.
	Execute(ctx context.Context, arguments json.RawMessage) (string, error)
}
