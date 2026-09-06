// Package components hosts the Component primitive and its
// implementations. ORH itself knows nothing about orchestration patterns
// (sequential, debate, supervisor, swarm, ...) — those emerge from how
// components are wired together in a .orh graph, not from special-cased
// runtime code.
package components

import (
	"context"

	"github.com/pertevniyalai/orh/internal/commands"
	"github.com/pertevniyalai/orh/internal/runtime/events"
)

// Component is the single execution primitive the ORH runtime understands.
type Component interface {
	Handle(ctx context.Context, event events.Event) ([]commands.Command, error)
}

// Initializer is an optional extension a Component can implement when it
// has real setup work to do before its first Handle call (e.g. a
// NodeComponent compiling and instantiating a WASM sandbox). The scheduler
// calls Init on every component that implements this, once, before running
// — plain components like AgentComponent don't need to implement it.
type Initializer interface {
	Init(ctx context.Context) error
}

// Shutdowner is the Initializer counterpart: an optional extension for a
// Component that holds resources (like a WASM instance) needing explicit
// cleanup after a run finishes.
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}
