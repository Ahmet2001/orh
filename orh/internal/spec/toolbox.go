package spec

// Toolbox is the root document parsed from a .tb file: a named set of
// tools a toolbox package exposes.
type Toolbox struct {
	Version int                `yaml:"version"`
	Tools   map[string]ToolDef `yaml:"tools"`
}

// ToolDef declares one tool in a toolbox.
type ToolDef struct {
	Description string          `yaml:"description"`
	Input       map[string]Port `yaml:"input,omitempty"`
	Output      map[string]Port `yaml:"output,omitempty"`
	Runtime     ToolRuntime     `yaml:"runtime"`
}

// ToolRuntime says how a tool is actually executed. Currently only "mcp" is
// supported: Server is a shell command that launches an MCP server ORH
// talks to over stdio.
type ToolRuntime struct {
	Type   string `yaml:"type"`
	Server string `yaml:"server"`
}
