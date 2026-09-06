package scheduler

import (
	"context"
	"fmt"
	"testing"

	"github.com/pertevniyalai/orh/internal/commands"
	"github.com/pertevniyalai/orh/internal/components"
	"github.com/pertevniyalai/orh/internal/graph"
	"github.com/pertevniyalai/orh/internal/runtime/events"
	"github.com/pertevniyalai/orh/internal/spec"
)

// echoComponent appends its name to the incoming payload and emits it on
// "output" — enough to trace a value through a graph in tests.
type echoComponent struct{ name string }

func (c echoComponent) Handle(ctx context.Context, event events.Event) ([]commands.Command, error) {
	return []commands.Command{
		commands.EmitEvent{Event: events.Event{
			Source:  c.name,
			Port:    "output",
			Payload: event.Text() + ">" + c.name,
		}},
	}, nil
}

func TestScheduler_Sequential(t *testing.T) {
	arch := &spec.Architecture{
		Connections: []spec.Connection{
			{From: "input", To: "a.input"},
			{From: "a.output", To: "b.input"},
			{From: "b.output", To: "output"},
		},
	}

	sched := &Scheduler{
		Graph: graph.Build(arch),
		Components: map[string]components.Component{
			"a": echoComponent{name: "a"},
			"b": echoComponent{name: "b"},
		},
	}

	outputs, err := sched.Run(context.Background(), events.Event{Payload: "start"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("len(outputs) = %d, want 1", len(outputs))
	}
	if got, want := outputs[0].Text(), "start>a>b"; got != want {
		t.Errorf("outputs[0].Text() = %q, want %q", got, want)
	}
}

func TestScheduler_ChainOfThree(t *testing.T) {
	arch := &spec.Architecture{
		Connections: []spec.Connection{
			{From: "input", To: "a.input"},
			{From: "a.output", To: "b.input"},
			{From: "b.output", To: "c.input"},
			{From: "c.output", To: "output"},
		},
	}

	sched := &Scheduler{
		Graph: graph.Build(arch),
		Components: map[string]components.Component{
			"a": echoComponent{name: "a"},
			"b": echoComponent{name: "b"},
			"c": echoComponent{name: "c"},
		},
	}

	outputs, err := sched.Run(context.Background(), events.Event{Payload: "start"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(outputs) != 1 || outputs[0].Text() != "start>a>b>c" {
		t.Fatalf("outputs = %+v, want a single event with payload %q", outputs, "start>a>b>c")
	}
}

func TestScheduler_FanOut(t *testing.T) {
	// Input feeds both A and B directly; neither result needs to be
	// merged in this phase, so both should independently reach "output".
	arch := &spec.Architecture{
		Connections: []spec.Connection{
			{From: "input", To: "a.input"},
			{From: "input", To: "b.input"},
			{From: "a.output", To: "output"},
			{From: "b.output", To: "output"},
		},
	}

	sched := &Scheduler{
		Graph: graph.Build(arch),
		Components: map[string]components.Component{
			"a": echoComponent{name: "a"},
			"b": echoComponent{name: "b"},
		},
	}

	outputs, err := sched.Run(context.Background(), events.Event{Payload: "start"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(outputs) != 2 {
		t.Fatalf("len(outputs) = %d, want 2", len(outputs))
	}

	got := map[string]bool{outputs[0].Text(): true, outputs[1].Text(): true}
	if !got["start>a"] || !got["start>b"] {
		t.Errorf("outputs = %+v, want one event each from a and b", outputs)
	}
}

func TestScheduler_UnknownComponent(t *testing.T) {
	arch := &spec.Architecture{
		Connections: []spec.Connection{
			{From: "input", To: "ghost.input"},
		},
	}

	sched := &Scheduler{Graph: graph.Build(arch), Components: map[string]components.Component{}}

	if _, err := sched.Run(context.Background(), events.Event{Payload: "hi"}); err == nil {
		t.Fatal("expected error for unregistered component, got nil")
	}
}

func TestScheduler_CycleIsBoundedNotRejected(t *testing.T) {
	// A feeds back into itself forever. The graph format allows cycles
	// (no DAG requirement), so this must fail at run time via the event
	// cap rather than being rejected up front.
	arch := &spec.Architecture{
		Connections: []spec.Connection{
			{From: "input", To: "a.input"},
			{From: "a.output", To: "a.input"},
		},
	}

	sched := &Scheduler{
		Graph:      graph.Build(arch),
		Components: map[string]components.Component{"a": echoComponent{name: "a"}},
		MaxEvents:  5,
	}

	_, err := sched.Run(context.Background(), events.Event{Payload: "start"})
	if err == nil {
		t.Fatal("expected an unbounded-cycle error, got nil")
	}
}

func TestScheduler_Trace(t *testing.T) {
	arch := &spec.Architecture{
		Connections: []spec.Connection{
			{From: "input", To: "a.input"},
			{From: "a.output", To: "output"},
		},
	}

	var steps []string
	sched := &Scheduler{
		Graph:      graph.Build(arch),
		Components: map[string]components.Component{"a": echoComponent{name: "a"}},
		Trace: func(kind, value string) {
			steps = append(steps, kind+":"+value)
		},
	}

	if _, err := sched.Run(context.Background(), events.Event{Payload: "start"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := []string{"Event:input", "Executing:a", "Output:a.output", "Output:output", "Completed:"}
	if len(steps) != len(want) {
		t.Fatalf("steps = %v, want %v", steps, want)
	}
	for i := range want {
		if steps[i] != want[i] {
			t.Errorf("steps[%d] = %q, want %q", i, steps[i], want[i])
		}
	}
}

// lifecycleComponent tracks Init/Shutdown calls and echoes its input.
type lifecycleComponent struct {
	name           string
	initCalled     *bool
	shutdownCalled *bool
	initErr        error
}

func (c *lifecycleComponent) Init(ctx context.Context) error {
	*c.initCalled = true
	return c.initErr
}

func (c *lifecycleComponent) Shutdown(ctx context.Context) error {
	*c.shutdownCalled = true
	return nil
}

func (c *lifecycleComponent) Handle(ctx context.Context, event events.Event) ([]commands.Command, error) {
	return []commands.Command{
		commands.EmitEvent{Event: events.Event{Source: c.name, Port: "output", Payload: event.Text()}},
	}, nil
}

func TestScheduler_InitAndShutdownAreCalled(t *testing.T) {
	arch := &spec.Architecture{
		Connections: []spec.Connection{
			{From: "input", To: "a.input"},
			{From: "a.output", To: "output"},
		},
	}

	initCalled, shutdownCalled := false, false
	comp := &lifecycleComponent{name: "a", initCalled: &initCalled, shutdownCalled: &shutdownCalled}

	sched := &Scheduler{Graph: graph.Build(arch), Components: map[string]components.Component{"a": comp}}

	if _, err := sched.Run(context.Background(), events.Event{Payload: "hi"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !initCalled {
		t.Error("expected Init to be called")
	}
	if !shutdownCalled {
		t.Error("expected Shutdown to be called")
	}
}

func TestScheduler_ShutdownRunsEvenWhenInitFails(t *testing.T) {
	arch := &spec.Architecture{
		Connections: []spec.Connection{{From: "input", To: "a.input"}, {From: "a.output", To: "output"}},
	}

	initCalled, shutdownCalled := false, false
	comp := &lifecycleComponent{name: "a", initCalled: &initCalled, shutdownCalled: &shutdownCalled, initErr: fmt.Errorf("boom")}

	sched := &Scheduler{Graph: graph.Build(arch), Components: map[string]components.Component{"a": comp}}

	if _, err := sched.Run(context.Background(), events.Event{Payload: "hi"}); err == nil {
		t.Fatal("expected Run() to fail when Init fails")
	}
}

// panicComponent panics on Handle, simulating a crashed node.
type panicComponent struct{}

func (panicComponent) Handle(ctx context.Context, event events.Event) ([]commands.Command, error) {
	panic("simulated node crash")
}

func TestScheduler_HandlePanicDoesNotCrashRuntime(t *testing.T) {
	arch := &spec.Architecture{
		Connections: []spec.Connection{{From: "input", To: "a.input"}, {From: "a.output", To: "output"}},
	}

	sched := &Scheduler{Graph: graph.Build(arch), Components: map[string]components.Component{"a": panicComponent{}}}

	_, err := sched.Run(context.Background(), events.Event{Payload: "hi"})
	if err == nil {
		t.Fatal("expected Run() to return an error for a panicking component, not crash the test process")
	}
}
