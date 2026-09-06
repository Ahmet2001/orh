package wasm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// callResult is the envelope orh_call_model/orh_call_tool write back to the
// guest — a single return value can't distinguish "empty success" from
// "denied/failed" any other way, so both outcomes go through this shape and
// the guest SDK (sdk/node) decides whether to surface Error.
type callResult struct {
	OK     bool   `json:"ok"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// registerWASI wires up the standard WASI preview1 imports a
// TinyGo-compiled module needs (memory/runtime bootstrap, mainly), with
// one deliberate override: proc_exit. TinyGo's wasip1 target always
// compiles a "command" style module whose _start calls proc_exit(0) once
// its (empty) main() returns; the stock WASI implementation treats that as
// "the program is done" and permanently closes the module. ORH instead
// wants to keep calling a node's "handle" export many times after that one
// startup pass, so proc_exit here is a no-op instead of a shutdown.
func registerWASI(ctx context.Context, rt wazero.Runtime) error {
	builder := rt.NewHostModuleBuilder(wasi_snapshot_preview1.ModuleName)
	wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(builder)

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, exitCode uint32) {}).
		Export("proc_exit")

	if _, err := builder.Instantiate(ctx); err != nil {
		return fmt.Errorf("registering WASI host module: %w", err)
	}
	return nil
}

// registerHostAPI registers the "env" host module a node module imports
// from: orh_log, orh_state_get, orh_state_set, orh_call_model,
// orh_call_tool. This is the *entire* capability surface a WASM node has —
// there is no network, filesystem, or process access unless a later phase
// deliberately adds a host function for it. orh_call_model/orh_call_tool
// are gated by the calling node's own .node manifest permissions (enforced
// inside the HostAPI implementation, e.g. nodes.NodeComponent), not here —
// this layer only marshals bytes across the sandbox boundary.
func registerHostAPI(ctx context.Context, rt wazero.Runtime, hostAPI HostAPI) error {
	builder := rt.NewHostModuleBuilder("env")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, ptr, length uint32) {
			if msg, ok := mod.Memory().Read(ptr, length); ok {
				hostAPI.Log(string(msg))
			}
		}).
		Export("orh_log")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, keyPtr, keyLen uint32) uint64 {
			key, ok := mod.Memory().Read(keyPtr, keyLen)
			if !ok {
				return 0
			}
			val, found := hostAPI.StateGet(string(key))
			if !found {
				return 0
			}

			alloc := mod.ExportedFunction("alloc")
			if alloc == nil {
				return 0
			}
			results, err := alloc.Call(ctx, uint64(len(val)))
			if err != nil || len(results) == 0 {
				return 0
			}
			valPtr := uint32(results[0])
			if !mod.Memory().Write(valPtr, []byte(val)) {
				return 0
			}
			return pack(valPtr, uint32(len(val)))
		}).
		Export("orh_state_get")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, keyPtr, keyLen, valPtr, valLen uint32) {
			key, kOK := mod.Memory().Read(keyPtr, keyLen)
			val, vOK := mod.Memory().Read(valPtr, valLen)
			if kOK && vOK {
				hostAPI.StateSet(string(key), string(val))
			}
		}).
		Export("orh_state_set")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, slotPtr, slotLen, promptPtr, promptLen uint32) uint64 {
			slot, ok1 := mod.Memory().Read(slotPtr, slotLen)
			prompt, ok2 := mod.Memory().Read(promptPtr, promptLen)
			if !ok1 || !ok2 {
				return writeCallResult(ctx, mod, callResult{Error: "reading arguments from wasm memory"})
			}

			text, err := hostAPI.CallModel(ctx, string(slot), string(prompt))
			if err != nil {
				return writeCallResult(ctx, mod, callResult{Error: err.Error()})
			}
			return writeCallResult(ctx, mod, callResult{OK: true, Result: text})
		}).
		Export("orh_call_model")

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, namePtr, nameLen, argsPtr, argsLen uint32) uint64 {
			name, ok1 := mod.Memory().Read(namePtr, nameLen)
			args, ok2 := mod.Memory().Read(argsPtr, argsLen)
			if !ok1 || !ok2 {
				return writeCallResult(ctx, mod, callResult{Error: "reading arguments from wasm memory"})
			}

			text, err := hostAPI.CallTool(ctx, string(name), string(args))
			if err != nil {
				return writeCallResult(ctx, mod, callResult{Error: err.Error()})
			}
			return writeCallResult(ctx, mod, callResult{OK: true, Result: text})
		}).
		Export("orh_call_tool")

	if _, err := builder.Instantiate(ctx); err != nil {
		return fmt.Errorf("registering host module: %w", err)
	}
	return nil
}

// writeCallResult JSON-encodes result, writes it into the guest's memory
// via its "alloc" export, and returns the packed ptr/len the guest reads it
// back from. Returns 0 (empty) if any step fails — the guest treats that as
// an ok:false result with no detail, same as encoding/allocation failing
// for any other host function.
func writeCallResult(ctx context.Context, mod api.Module, result callResult) uint64 {
	data, err := json.Marshal(result)
	if err != nil {
		return 0
	}

	alloc := mod.ExportedFunction("alloc")
	if alloc == nil {
		return 0
	}
	results, err := alloc.Call(ctx, uint64(len(data)))
	if err != nil || len(results) == 0 {
		return 0
	}
	ptr := uint32(results[0])
	if !mod.Memory().Write(ptr, data) {
		return 0
	}
	return pack(ptr, uint32(len(data)))
}
