//go:build wasm

// A minimal test node exercising node.CallModel and node.CallTool — used to
// prove the orh_call_model/orh_call_tool host ABI round-trips real values
// (and surfaces host-side permission denials) through an actual compiled
// WASM module, not just Go-level fakes.
package main

import "github.com/pertevniyalai/orh/sdk/node"

func main() {}

func init() { node.Handle(route) }

func route(event node.Event) []node.Command {
	switch event.Port {
	case "model":
		result, err := node.CallModel("reasoning", event.Payload)
		if err != nil {
			return []node.Command{{Port: "error", Payload: err.Error()}}
		}
		return []node.Command{{Port: "model_out", Payload: result}}
	case "tool":
		result, err := node.CallTool("web.search", event.Payload)
		if err != nil {
			return []node.Command{{Port: "error", Payload: err.Error()}}
		}
		return []node.Command{{Port: "tool_out", Payload: result}}
	default:
		return []node.Command{{Port: "error", Payload: "unknown port"}}
	}
}
