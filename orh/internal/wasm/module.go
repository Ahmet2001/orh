// Package wasm is ORH's WASM host: it loads a compiled node module,
// exposes a small, fixed set of host functions to it (log, get/set state —
// see host.go), and calls its "handle" export. A node can only do what
// these host functions let it do; there is no ambient access to the
// network, filesystem, or process beyond what this package deliberately
// wires up (currently: nothing dangerous at all).
package wasm

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// HostAPI is what a Module's host functions call back into. A node
// component implements this to provide its own logging and per-instance
// state.
type HostAPI interface {
	Log(message string)
	StateGet(key string) (string, bool)
	StateSet(key, value string)
	// CallModel runs prompt against the architecture's modelSlot model
	// (a key in the .orh file's `models:` block) and returns its text
	// response. Implementations must reject slots the node's own .node
	// manifest hasn't granted under permissions.models.allow.
	CallModel(ctx context.Context, modelSlot, prompt string) (string, error)
	// CallTool invokes the fully-qualified tool named toolName (e.g.
	// "web.search") with a JSON-encoded arguments object and returns its
	// text result. Implementations must reject tools the node's own
	// .node manifest hasn't granted under permissions.tools.allow.
	CallTool(ctx context.Context, toolName, argumentsJSON string) (string, error)
}

// Module is one loaded, instantiated WASM node. Each Module owns a private
// wazero runtime, so state and host bindings never leak between nodes.
type Module struct {
	runtime wazero.Runtime
	guest   api.Module
	alloc   api.Function
	handle  api.Function
}

// Load compiles wasmBytes and instantiates it, wiring hostAPI as its host
// functions. The module must export "alloc" (uint32 size -> uint32 ptr)
// and "handle" (uint32 ptr, uint32 len -> uint64 packed ptr/len) — see
// sdk/node for the reference implementation of this contract.
func Load(ctx context.Context, wasmBytes []byte, hostAPI HostAPI) (*Module, error) {
	rt := wazero.NewRuntime(ctx)

	if err := registerWASI(ctx, rt); err != nil {
		rt.Close(ctx)
		return nil, err
	}

	if err := registerHostAPI(ctx, rt, hostAPI); err != nil {
		rt.Close(ctx)
		return nil, err
	}

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("compiling wasm module: %w", err)
	}

	guest, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("instantiating wasm module: %w", err)
	}

	alloc := guest.ExportedFunction("alloc")
	if alloc == nil {
		rt.Close(ctx)
		return nil, fmt.Errorf(`wasm module does not export "alloc"`)
	}
	handle := guest.ExportedFunction("handle")
	if handle == nil {
		rt.Close(ctx)
		return nil, fmt.Errorf(`wasm module does not export "handle"`)
	}

	return &Module{runtime: rt, guest: guest, alloc: alloc, handle: handle}, nil
}

// Handle writes input into the guest's memory (via its "alloc" export) and
// calls its "handle" export, returning whatever bytes the guest wrote back.
func (m *Module) Handle(ctx context.Context, input []byte) ([]byte, error) {
	ptr := uint32(0)
	if len(input) > 0 {
		results, err := m.alloc.Call(ctx, uint64(len(input)))
		if err != nil {
			return nil, fmt.Errorf("calling alloc: %w", err)
		}
		ptr = uint32(results[0])
		if !m.guest.Memory().Write(ptr, input) {
			return nil, fmt.Errorf("writing input into wasm memory")
		}
	}

	results, err := m.handle.Call(ctx, uint64(ptr), uint64(len(input)))
	if err != nil {
		return nil, fmt.Errorf("calling handle: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("handle returned no result")
	}

	resPtr, resLen := unpack(results[0])
	if resLen == 0 {
		return nil, nil
	}
	out, ok := m.guest.Memory().Read(resPtr, resLen)
	if !ok {
		return nil, fmt.Errorf("reading handle result from wasm memory")
	}

	cp := make([]byte, len(out))
	copy(cp, out)
	return cp, nil
}

// Close releases the module's WASM runtime.
func (m *Module) Close(ctx context.Context) error {
	return m.runtime.Close(ctx)
}

func pack(ptr, length uint32) uint64 {
	return (uint64(ptr) << 32) | uint64(length)
}

func unpack(v uint64) (ptr, length uint32) {
	return uint32(v >> 32), uint32(v)
}
