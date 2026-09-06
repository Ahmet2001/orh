package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// fakeHostAPI is an in-memory HostAPI for tests.
type fakeHostAPI struct {
	logs  []string
	state map[string]string
}

func newFakeHostAPI() *fakeHostAPI {
	return &fakeHostAPI{state: map[string]string{}}
}

func (f *fakeHostAPI) Log(message string) { f.logs = append(f.logs, message) }
func (f *fakeHostAPI) StateGet(key string) (string, bool) {
	v, ok := f.state[key]
	return v, ok
}
func (f *fakeHostAPI) StateSet(key, value string) { f.state[key] = value }
func (f *fakeHostAPI) CallModel(ctx context.Context, modelSlot, prompt string) (string, error) {
	return "", fmt.Errorf("fakeHostAPI: CallModel not implemented")
}
func (f *fakeHostAPI) CallTool(ctx context.Context, toolName, argumentsJSON string) (string, error) {
	return "", fmt.Errorf("fakeHostAPI: CallTool not implemented")
}

// loadRouterFixture loads testdata/router.wasm, compiled from
// testdata/router via `tinygo build -target=wasip1 -gc=leaking`. It skips
// the test if the fixture hasn't been built (e.g. TinyGo isn't installed
// in this environment).
func loadRouterFixture(t *testing.T, hostAPI HostAPI) *Module {
	t.Helper()

	data, err := os.ReadFile("testdata/router.wasm")
	if err != nil {
		t.Skipf("testdata/router.wasm not built (run tinygo build in internal/wasm/testdata/router): %v", err)
	}

	mod, err := Load(context.Background(), data, hostAPI)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	t.Cleanup(func() { mod.Close(context.Background()) })
	return mod
}

type event struct {
	Port    string `json:"port"`
	Payload string `json:"payload"`
}

type command struct {
	Port    string `json:"port"`
	Payload string `json:"payload"`
}

func TestModule_Handle_RoutesByPayload(t *testing.T) {
	host := newFakeHostAPI()
	mod := loadRouterFixture(t, host)

	input, _ := json.Marshal(event{Payload: "urgent"})
	out, err := mod.Handle(context.Background(), input)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	var cmds []command
	if err := json.Unmarshal(out, &cmds); err != nil {
		t.Fatalf("decoding result: %v (raw: %s)", err, out)
	}
	if len(cmds) != 1 || cmds[0].Port != "agent_b" {
		t.Fatalf("cmds = %+v, want one command routed to agent_b", cmds)
	}
}

func TestModule_Handle_DefaultRoute(t *testing.T) {
	host := newFakeHostAPI()
	mod := loadRouterFixture(t, host)

	input, _ := json.Marshal(event{Payload: "hello"})
	out, err := mod.Handle(context.Background(), input)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	var cmds []command
	if err := json.Unmarshal(out, &cmds); err != nil {
		t.Fatalf("decoding result: %v", err)
	}
	if len(cmds) != 1 || cmds[0].Port != "agent_a" {
		t.Fatalf("cmds = %+v, want one command routed to agent_a", cmds)
	}
}

func TestModule_Handle_LogsAndState(t *testing.T) {
	host := newFakeHostAPI()
	mod := loadRouterFixture(t, host)

	input, _ := json.Marshal(event{Payload: "hello"})
	if _, err := mod.Handle(context.Background(), input); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if len(host.logs) != 1 || host.logs[0] != "routing payload: hello" {
		t.Errorf("logs = %v, want one log about routing hello", host.logs)
	}
	if host.state["count"] != "01" {
		t.Errorf("state[count] = %q, want %q", host.state["count"], "01")
	}

	// A second call should see the state the first call wrote.
	if _, err := mod.Handle(context.Background(), input); err != nil {
		t.Fatalf("second Handle() error = %v", err)
	}
	if host.state["count"] != "011" {
		t.Errorf("state[count] after second call = %q, want %q", host.state["count"], "011")
	}
}
