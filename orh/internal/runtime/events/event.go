// Package events defines Event, the data unit that flows along connections
// between components in the ORH execution graph.
package events

// Event carries data between components. Source, Target, and Port identify
// where the event came from and where it is headed; Payload carries the
// actual data and is deliberately untyped so future component kinds (tools,
// custom nodes, ...) can pass structured data without changing this type.
type Event struct {
	// Source is the component name that produced this event, if any.
	Source string
	// Target is the fully qualified endpoint this event is addressed to,
	// e.g. "writer.input". Set by the runtime while routing, not by the
	// component that emits the event.
	Target string
	// Port is the named port on Source (when emitting) or Target (when
	// delivering) this event flows through, e.g. "input" or "output".
	Port string
	// Payload is the event's data.
	Payload any
}

// Text returns the event's payload as a string, or "" if the payload is not
// a string. Most components in the current phase deal only in text.
func (e Event) Text() string {
	s, _ := e.Payload.(string)
	return s
}
