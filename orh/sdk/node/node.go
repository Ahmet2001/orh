//go:build wasm

// Package node is the guest-side SDK for writing an ORH custom node in
// Go, compiled to WASM with TinyGo. It lives outside internal/ (unlike the
// rest of ORH) because a node is authored in a *separate* repository and
// module — Go's internal/ visibility would make it unimportable there.
//
// A node's main.go looks like:
//
//	package main
//
//	import "github.com/pertevniyalai/orh/sdk/node"
//
//	func main() {}
//
//	func init() { node.Handle(route) }
//
//	func route(event node.Event) []node.Command {
//	    if event.Payload == "urgent" {
//	        return []node.Command{{Port: "agent_b", Payload: event.Payload}}
//	    }
//	    return []node.Command{{Port: "agent_a", Payload: event.Payload}}
//	}
//
// Build it with:
//
//	tinygo build -o mynode.wasm -target=wasip1 -gc=leaking -no-debug main.go
//
// -gc=leaking is required: buffers this package hands to the host via
// pointer/length pairs must never be moved or collected by TinyGo's
// garbage collector while the host is still reading them.
package node

import (
	"encoding/json"
	"errors"
	"unsafe"
)

//go:wasmimport env orh_log
func hostLog(ptr, length uint32)

//go:wasmimport env orh_state_get
func hostStateGet(keyPtr, keyLen uint32) uint64

//go:wasmimport env orh_state_set
func hostStateSet(keyPtr, keyLen, valPtr, valLen uint32)

//go:wasmimport env orh_call_model
func hostCallModel(slotPtr, slotLen, promptPtr, promptLen uint32) uint64

//go:wasmimport env orh_call_tool
func hostCallTool(namePtr, nameLen, argsPtr, argsLen uint32) uint64

// Event mirrors the event a node receives — see internal/runtime/events.
type Event struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Port    string `json:"port"`
	Payload string `json:"payload"`
}

// Command is what a node emits: route Payload onward from the named
// output Port.
type Command struct {
	Port    string `json:"port"`
	Payload string `json:"payload"`
}

// HandlerFunc is a node's event-handling logic.
type HandlerFunc func(Event) []Command

var handler HandlerFunc

// Handle registers fn as the node's event handler. Call it once, e.g. from
// an init() function, before the host ever calls into the module.
func Handle(fn HandlerFunc) { handler = fn }

// Log sends a message to the ORH host's log, prefixed with this node's
// name.
func Log(msg string) {
	b := []byte(msg)
	hostLog(bufPtr(b), uint32(len(b)))
}

// StateGet reads a value previously stored with StateSet. Reports false if
// the key has never been set.
func StateGet(key string) (string, bool) {
	kb := []byte(key)
	packed := hostStateGet(bufPtr(kb), uint32(len(kb)))
	if packed == 0 {
		return "", false
	}
	ptr, length := unpack(packed)
	return string(readMemory(ptr, length)), true
}

// StateSet stores value under key for later StateGet calls. State is
// scoped to this node instance and does not survive past the run.
func StateSet(key, value string) {
	kb, vb := []byte(key), []byte(value)
	hostStateSet(bufPtr(kb), uint32(len(kb)), bufPtr(vb), uint32(len(vb)))
}

// callResult mirrors the host's internal/wasm.callResult envelope.
type callResult struct {
	OK     bool   `json:"ok"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// CallModel asks the host to run prompt against the architecture's
// modelSlot model (a key in the owning .orh file's `models:` block) and
// returns its text response. The host rejects the call unless modelSlot is
// listed under this node's own permissions.models.allow in its .node
// manifest — in which case CallModel returns that rejection as an error.
func CallModel(modelSlot, prompt string) (string, error) {
	sb, pb := []byte(modelSlot), []byte(prompt)
	packed := hostCallModel(bufPtr(sb), uint32(len(sb)), bufPtr(pb), uint32(len(pb)))
	return decodeCallResult(packed)
}

// CallTool asks the host to invoke the fully qualified tool named name
// (e.g. "web.search") with a JSON-encoded arguments object, returning its
// text result. The host rejects the call unless name is listed under this
// node's own permissions.tools.allow in its .node manifest.
func CallTool(name, argumentsJSON string) (string, error) {
	nb, ab := []byte(name), []byte(argumentsJSON)
	packed := hostCallTool(bufPtr(nb), uint32(len(nb)), bufPtr(ab), uint32(len(ab)))
	return decodeCallResult(packed)
}

func decodeCallResult(packed uint64) (string, error) {
	if packed == 0 {
		return "", errors.New("host call failed")
	}
	ptr, length := unpack(packed)
	var result callResult
	if err := json.Unmarshal(readMemory(ptr, length), &result); err != nil {
		return "", errors.New("decoding host call result")
	}
	if !result.OK {
		if result.Error != "" {
			return "", errors.New(result.Error)
		}
		return "", errors.New("host call failed")
	}
	return result.Result, nil
}

//export alloc
func alloc(size uint32) uint32 {
	if size == 0 {
		return 0
	}
	buf := make([]byte, size)
	return bufPtr(buf)
}

//export handle
func handleExport(ptr, length uint32) uint64 {
	if handler == nil {
		return 0
	}

	var event Event
	if err := json.Unmarshal(readMemory(ptr, length), &event); err != nil {
		return 0
	}

	cmds := handler(event)
	if cmds == nil {
		cmds = []Command{}
	}

	out, err := json.Marshal(cmds)
	if err != nil {
		return 0
	}

	outPtr := bufPtr(out)
	return pack(outPtr, uint32(len(out)))
}

func pack(ptr, length uint32) uint64 {
	return (uint64(ptr) << 32) | uint64(length)
}

func unpack(v uint64) (ptr, length uint32) {
	return uint32(v >> 32), uint32(v)
}

// bufPtr returns buf's address as a WASM linear-memory offset. The caller
// must keep a reference to buf alive (or build with -gc=leaking) for as
// long as the host may still read it.
func bufPtr(buf []byte) uint32 {
	if len(buf) == 0 {
		return 0
	}
	return uint32(uintptr(unsafe.Pointer(&buf[0])))
}

func readMemory(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}
