package spec

// Node is the root document parsed from a .node file: a custom
// WASM-sandboxed component's contract.
type Node struct {
	APIVersion  string          `yaml:"apiVersion"`
	Kind        string          `yaml:"kind"`
	Name        string          `yaml:"name"`
	Version     string          `yaml:"version"`
	Runtime     NodeRuntime     `yaml:"runtime"`
	Entrypoint  NodeEntrypoint  `yaml:"entrypoint"`
	Inputs      map[string]Port `yaml:"inputs,omitempty"`
	Outputs     map[string]Port `yaml:"outputs,omitempty"`
	Permissions NodePermissions `yaml:"permissions,omitempty"`
}

// NodeRuntime says how the node is executed. Currently only "wasm" is
// supported.
type NodeRuntime struct {
	Type string `yaml:"type"`
}

// NodeEntrypoint names the compiled module a "wasm" runtime loads.
type NodeEntrypoint struct {
	Module string `yaml:"module"`
}

// NodePermissions is a node's declared allow-list, checked before a node
// is loaded. models/tools/network/filesystem are currently only enforced
// implicitly, by which host functions the WASM sandbox exposes at all
// (nothing dangerous is exposed yet regardless of what a node declares
// here); once the host API grows model/tool/network/filesystem calls, this
// becomes the gate for them.
type NodePermissions struct {
	Models     NodeModelsPermission     `yaml:"models,omitempty"`
	Tools      NodeToolsPermission      `yaml:"tools,omitempty"`
	Network    NodeNetworkPermission    `yaml:"network,omitempty"`
	Filesystem NodeFilesystemPermission `yaml:"filesystem,omitempty"`
}

type NodeModelsPermission struct {
	Allow []string `yaml:"allow,omitempty"`
}

type NodeToolsPermission struct {
	Allow []string `yaml:"allow,omitempty"`
}

type NodeNetworkPermission struct {
	Allow []string `yaml:"allow,omitempty"`
}

type NodeFilesystemPermission struct {
	Read []string `yaml:"read,omitempty"`
}
