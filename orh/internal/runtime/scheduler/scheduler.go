// Package scheduler runs an execution graph: it owns the event queue,
// dispatches events to the right component, and turns the Commands a
// component returns into new queued events.
package scheduler

import (
	"context"
	"fmt"

	"github.com/pertevniyalai/orh/internal/commands"
	"github.com/pertevniyalai/orh/internal/components"
	"github.com/pertevniyalai/orh/internal/graph"
	"github.com/pertevniyalai/orh/internal/runtime/events"
	"github.com/pertevniyalai/orh/internal/spec"
)

// defaultMaxEvents caps how many events a single Run will process. The
// graph format does not require a DAG (cycles are allowed by design), so
// this cap is what keeps a runaway cycle from hanging the runtime forever
// rather than a validation-time rejection.
const defaultMaxEvents = 10000

// Trace, when set on a Scheduler, is called for each notable step during
// Run — used to power `orh run --debug`. kind is a short label ("Event",
// "Executing", "Output", "Completed"); value is empty for kind-only steps.
type Trace func(kind, value string)

// Scheduler executes a Graph by feeding events through it, starting at the
// "input" boundary node and collecting whatever reaches "output".
type Scheduler struct {
	Graph      *graph.Graph
	Components map[string]components.Component
	Trace      Trace
	// MaxEvents overrides defaultMaxEvents when non-zero.
	MaxEvents int
}

type queueItem struct {
	endpoint string
	event    events.Event
}

// Run initializes every component that needs it, feeds the input event
// into the graph, and returns the event(s) collected at the "output"
// boundary. Components are shut down again before Run returns, whether it
// succeeded or not.
func (s *Scheduler) Run(ctx context.Context, input events.Event) ([]events.Event, error) {
	if err := s.initComponents(ctx); err != nil {
		return nil, err
	}
	defer s.shutdownComponents(ctx)

	maxEvents := s.MaxEvents
	if maxEvents == 0 {
		maxEvents = defaultMaxEvents
	}

	var outputs []events.Event
	var queue []queueItem

	s.trace("Event", spec.NodeInput)
	for _, target := range s.Graph.Successors(spec.NodeInput) {
		queue = append(queue, queueItem{endpoint: target, event: input})
	}

	processed := 0
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		processed++
		if processed > maxEvents {
			return nil, fmt.Errorf("exceeded maximum of %d events; possible unbounded cycle", maxEvents)
		}

		if item.endpoint == spec.NodeOutput {
			s.trace("Output", spec.NodeOutput)
			outputs = append(outputs, item.event)
			continue
		}

		name, port := graph.SplitEndpoint(item.endpoint)
		component, ok := s.Components[name]
		if !ok {
			return nil, fmt.Errorf("no component registered for %q", name)
		}

		event := item.event
		event.Target = item.endpoint
		if event.Port == "" {
			event.Port = port
		}

		s.trace("Executing", name)
		cmds, err := s.safeHandle(ctx, name, component, event)
		if err != nil {
			return nil, err
		}

		if err := s.applyCommands(name, cmds, &queue); err != nil {
			return nil, err
		}
	}

	s.trace("Completed", "")
	return outputs, nil
}

func (s *Scheduler) applyCommands(componentName string, cmds []commands.Command, queue *[]queueItem) error {
	for _, cmd := range cmds {
		switch c := cmd.(type) {
		case commands.EmitEvent:
			out := c.Event
			if out.Source == "" {
				out.Source = componentName
			}
			srcEndpoint := out.Source + "." + out.Port
			s.trace("Output", srcEndpoint)
			for _, target := range s.Graph.Successors(srcEndpoint) {
				*queue = append(*queue, queueItem{endpoint: target, event: out})
			}
		case commands.SetOutput:
			*queue = append(*queue, queueItem{endpoint: spec.NodeOutput, event: c.Event})
		default:
			return fmt.Errorf("component %q: unsupported command %T", componentName, cmd)
		}
	}
	return nil
}

// initComponents calls Init on every component that implements
// components.Initializer (e.g. a NodeComponent compiling its WASM module).
func (s *Scheduler) initComponents(ctx context.Context) error {
	for name, c := range s.Components {
		if init, ok := c.(components.Initializer); ok {
			if err := init.Init(ctx); err != nil {
				return fmt.Errorf("component %q: init: %w", name, err)
			}
		}
	}
	return nil
}

// shutdownComponents calls Shutdown on every component that implements
// components.Shutdowner. Errors are traced, not returned — a cleanup
// failure shouldn't hide the run's actual result.
func (s *Scheduler) shutdownComponents(ctx context.Context) {
	for name, c := range s.Components {
		if sd, ok := c.(components.Shutdowner); ok {
			if err := sd.Shutdown(ctx); err != nil {
				s.trace("ShutdownError", name+": "+err.Error())
			}
		}
	}
}

// safeHandle calls component.Handle, converting a panic (e.g. an
// unexpected WASM trap a node's sandbox didn't already turn into an error)
// into a normal error instead of taking down the whole ORH process — a
// broken node fails its run, it doesn't crash the runtime.
func (s *Scheduler) safeHandle(ctx context.Context, name string, component components.Component, event events.Event) (cmds []commands.Command, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("component %q panicked: %v", name, r)
		}
	}()
	return component.Handle(ctx, event)
}

func (s *Scheduler) trace(kind, value string) {
	if s.Trace != nil {
		s.Trace(kind, value)
	}
}
