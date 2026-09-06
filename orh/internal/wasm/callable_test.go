package wasm

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

// callFakeHostAPI is a fakeHostAPI whose CallModel/CallTool are scriptable,
// used to prove the orh_call_model/orh_call_tool ABI round-trips both
// successful results and host-side permission denials through an actual
// compiled WASM module (internal/wasm/testdata/caller).
type callFakeHostAPI struct {
	*fakeHostAPI
	modelResult string
	modelErr    error
	toolResult  string
	toolErr     error
}

func (f *callFakeHostAPI) CallModel(ctx context.Context, modelSlot, prompt string) (string, error) {
	if modelSlot != "reasoning" {
		return "", errors.New("unexpected model slot: " + modelSlot)
	}
	return f.modelResult, f.modelErr
}

func (f *callFakeHostAPI) CallTool(ctx context.Context, toolName, argumentsJSON string) (string, error) {
	if toolName != "web.search" {
		return "", errors.New("unexpected tool name: " + toolName)
	}
	return f.toolResult, f.toolErr
}

func loadCallerFixture(t *testing.T, hostAPI HostAPI) *Module {
	t.Helper()

	data, err := os.ReadFile("testdata/caller/caller.wasm")
	if err != nil {
		t.Skipf("testdata/caller/caller.wasm not built (run tinygo build in internal/wasm/testdata/caller): %v", err)
	}

	mod, err := Load(context.Background(), data, hostAPI)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	t.Cleanup(func() { mod.Close(context.Background()) })
	return mod
}

func TestModule_CallModel_SuccessRoundTrips(t *testing.T) {
	host := &callFakeHostAPI{fakeHostAPI: newFakeHostAPI(), modelResult: "Paris"}
	mod := loadCallerFixture(t, host)

	input, _ := json.Marshal(event{Port: "model", Payload: "capital of France?"})
	out, err := mod.Handle(context.Background(), input)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	var cmds []command
	if err := json.Unmarshal(out, &cmds); err != nil {
		t.Fatalf("decoding result: %v (raw: %s)", err, out)
	}
	if len(cmds) != 1 || cmds[0].Port != "model_out" || cmds[0].Payload != "Paris" {
		t.Fatalf("cmds = %+v, want one model_out command with payload Paris", cmds)
	}
}

func TestModule_CallModel_DenialSurfacesAsError(t *testing.T) {
	host := &callFakeHostAPI{fakeHostAPI: newFakeHostAPI(), modelErr: errors.New("model slot \"reasoning\" is not permitted")}
	mod := loadCallerFixture(t, host)

	input, _ := json.Marshal(event{Port: "model", Payload: "hi"})
	out, err := mod.Handle(context.Background(), input)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	var cmds []command
	if err := json.Unmarshal(out, &cmds); err != nil {
		t.Fatalf("decoding result: %v (raw: %s)", err, out)
	}
	if len(cmds) != 1 || cmds[0].Port != "error" {
		t.Fatalf("cmds = %+v, want one error command", cmds)
	}
	if cmds[0].Payload == "" {
		t.Error("expected the denial reason to reach the guest")
	}
}

func TestModule_CallTool_SuccessAndDenial(t *testing.T) {
	host := &callFakeHostAPI{fakeHostAPI: newFakeHostAPI(), toolResult: "3 results found"}
	mod := loadCallerFixture(t, host)

	input, _ := json.Marshal(event{Port: "tool", Payload: `{"q":"orh"}`})
	out, err := mod.Handle(context.Background(), input)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	var cmds []command
	if err := json.Unmarshal(out, &cmds); err != nil {
		t.Fatalf("decoding result: %v", err)
	}
	if len(cmds) != 1 || cmds[0].Port != "tool_out" || cmds[0].Payload != "3 results found" {
		t.Fatalf("cmds = %+v, want one tool_out command", cmds)
	}

	host2 := &callFakeHostAPI{fakeHostAPI: newFakeHostAPI(), toolErr: errors.New("tool \"web.search\" is not permitted")}
	mod2 := loadCallerFixture(t, host2)
	out2, err := mod2.Handle(context.Background(), input)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	var cmds2 []command
	if err := json.Unmarshal(out2, &cmds2); err != nil {
		t.Fatalf("decoding result: %v", err)
	}
	if len(cmds2) != 1 || cmds2[0].Port != "error" {
		t.Fatalf("cmds = %+v, want one error command for a denied tool call", cmds2)
	}
}
