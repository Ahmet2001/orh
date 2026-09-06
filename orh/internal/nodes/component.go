package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/pertevniyalai/orh/internal/commands"
	"github.com/pertevniyalai/orh/internal/providers"
	"github.com/pertevniyalai/orh/internal/providers/resolve"
	"github.com/pertevniyalai/orh/internal/runtime/events"
	"github.com/pertevniyalai/orh/internal/spec"
	"github.com/pertevniyalai/orh/internal/tools"
	"github.com/pertevniyalai/orh/internal/wasm"
)

// NodeComponent runs a custom, community-built component inside a WASM
// sandbox. It implements components.Component (Handle) plus the optional
// components.Initializer/Shutdowner interfaces: a WASM module has real
// setup and teardown work (compiling and instantiating the sandbox), unlike
// a plain AgentComponent, so it uses those hooks rather than doing that
// work lazily inside Handle.
type NodeComponent struct {
	Name      string
	WASMBytes []byte

	// Models is the owning architecture's `models:` block (slot name ->
	// provider/model), and Tools is its resolved tool registry (keyed by
	// "alias.toolName") — both wired in by executor.buildNodeComponent so
	// CallModel/CallTool have something to call. Permissions is this
	// node's own .node manifest allow-lists, which gate every call.
	Models      map[string]spec.Model
	Tools       *tools.Registry
	Permissions spec.NodePermissions

	mu     sync.Mutex
	module *wasm.Module
	state  map[string]string
}

// Init implements components.Initializer.
func (n *NodeComponent) Init(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.state = map[string]string{}
	module, err := wasm.Load(ctx, n.WASMBytes, n)
	if err != nil {
		return fmt.Errorf("node %q: %w", n.Name, err)
	}
	n.module = module
	return nil
}

// Shutdown implements components.Shutdowner.
func (n *NodeComponent) Shutdown(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.module == nil {
		return nil
	}
	err := n.module.Close(ctx)
	n.module = nil
	return err
}

// wireEvent/wireCommand are the JSON shapes exchanged with the guest —
// mirrored by sdk/node's Event/Command types.
type wireEvent struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Port    string `json:"port"`
	Payload string `json:"payload"`
}

type wireCommand struct {
	Port    string `json:"port"`
	Payload string `json:"payload"`
}

// Handle implements components.Component by delegating to the WASM
// module's "handle" export.
func (n *NodeComponent) Handle(ctx context.Context, event events.Event) ([]commands.Command, error) {
	n.mu.Lock()
	module := n.module
	n.mu.Unlock()

	if module == nil {
		return nil, fmt.Errorf("node %q: not initialized", n.Name)
	}

	input, err := json.Marshal(wireEvent{
		Source:  event.Source,
		Target:  event.Target,
		Port:    event.Port,
		Payload: event.Text(),
	})
	if err != nil {
		return nil, fmt.Errorf("node %q: encoding input: %w", n.Name, err)
	}

	out, err := module.Handle(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("node %q: %w", n.Name, err)
	}
	if len(out) == 0 {
		return nil, nil
	}

	var wireCmds []wireCommand
	if err := json.Unmarshal(out, &wireCmds); err != nil {
		return nil, fmt.Errorf("node %q: decoding output: %w", n.Name, err)
	}

	cmds := make([]commands.Command, len(wireCmds))
	for i, c := range wireCmds {
		cmds[i] = commands.EmitEvent{Event: events.Event{Source: n.Name, Port: c.Port, Payload: c.Payload}}
	}
	return cmds, nil
}

// Log implements wasm.HostAPI (backs the guest's orh_log import).
func (n *NodeComponent) Log(message string) {
	log.Printf("[node %s] %s", n.Name, message)
}

// StateGet implements wasm.HostAPI.
func (n *NodeComponent) StateGet(key string) (string, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	v, ok := n.state[key]
	return v, ok
}

// StateSet implements wasm.HostAPI.
func (n *NodeComponent) StateSet(key, value string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.state[key] = value
}

// CallModel implements wasm.HostAPI (backs the guest's orh_call_model
// import). modelSlot must both name a slot in the owning architecture's
// `models:` block and appear in this node's .node manifest under
// permissions.models.allow — the latter is the actual enforcement point,
// checked before any model is ever reached.
func (n *NodeComponent) CallModel(ctx context.Context, modelSlot, prompt string) (string, error) {
	if !contains(n.Permissions.Models.Allow, modelSlot) {
		return "", fmt.Errorf("node %q: model slot %q is not permitted (add it to permissions.models.allow in the node's .node manifest)", n.Name, modelSlot)
	}

	modelSpec, ok := n.Models[modelSlot]
	if !ok {
		return "", fmt.Errorf("node %q: architecture has no model slot %q", n.Name, modelSlot)
	}

	provider, err := resolve.Provider(modelSpec.Provider, modelSpec.Name)
	if err != nil {
		return "", fmt.Errorf("node %q: resolving model slot %q: %w", n.Name, modelSlot, err)
	}

	resp, err := provider.Generate(ctx, providers.Request{Prompt: prompt})
	if err != nil {
		return "", fmt.Errorf("node %q: calling model slot %q: %w", n.Name, modelSlot, err)
	}
	return resp.Text, nil
}

// CallTool implements wasm.HostAPI (backs the guest's orh_call_tool
// import). toolName is a fully qualified "alias.toolName" reference, the
// same form an agent's `tools:` list uses. As with CallModel, the node's
// own permissions.tools.allow list is the enforcement point, checked
// before the tool registry is even consulted.
func (n *NodeComponent) CallTool(ctx context.Context, toolName, argumentsJSON string) (string, error) {
	if !contains(n.Permissions.Tools.Allow, toolName) {
		return "", fmt.Errorf("node %q: tool %q is not permitted (add it to permissions.tools.allow in the node's .node manifest)", n.Name, toolName)
	}

	if n.Tools == nil {
		return "", fmt.Errorf("node %q: no tools are available to this architecture", n.Name)
	}
	t, ok := n.Tools.Get(toolName)
	if !ok {
		return "", fmt.Errorf("node %q: undefined tool %q", n.Name, toolName)
	}

	return t.Execute(ctx, json.RawMessage(argumentsJSON))
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
