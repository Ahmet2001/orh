// Package commands defines the results a Component can hand back to the
// runtime. A Component never mutates runtime state directly — it only
// returns Commands describing what it wants to happen next.
package commands

import "github.com/pertevniyalai/orh/internal/runtime/events"

// Command is something a Component asks the runtime to do after handling an
// event. It is deliberately an open interface (rather than a closed set of
// structs) so new command kinds — SpawnComponent, CallTool, UpdateState,
// ... — can be introduced later without changing the Component contract or
// breaking existing components.
type Command interface{}

// EmitEvent asks the runtime to route Event along whatever graph
// connections leave Event.Source's Event.Port. This is how most components
// hand data to their downstream neighbors.
type EmitEvent struct {
	Event events.Event
}

// SetOutput asks the runtime to record Event directly as a final result,
// bypassing graph connection resolution. Reserved for components that need
// to finalize a result outside the normal routing path.
type SetOutput struct {
	Event events.Event
}
