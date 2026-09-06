//go:build wasm

// A minimal test node: routes to "agent_b" if the payload contains
// "urgent", otherwise "agent_a". Also exercises Log and State to prove the
// host API round-trip works.
package main

import "github.com/pertevniyalai/orh/sdk/node"

func main() {}

func init() { node.Handle(route) }

func route(event node.Event) []node.Command {
	node.Log("routing payload: " + event.Payload)

	count := "0"
	if v, ok := node.StateGet("count"); ok {
		count = v
	}
	node.StateSet("count", count+"1")

	if event.Payload == "urgent" {
		return []node.Command{{Port: "agent_b", Payload: event.Payload}}
	}
	return []node.Command{{Port: "agent_a", Payload: event.Payload}}
}
