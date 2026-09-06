// Package spec defines the data model for the .orh architecture format.
package spec

// Architecture is the root document parsed from a .orh file.
type Architecture struct {
	APIVersion  string               `yaml:"apiVersion"`
	Name        string               `yaml:"name"`
	Inputs      map[string]Port      `yaml:"inputs,omitempty"`
	Outputs     map[string]Port      `yaml:"outputs,omitempty"`
	Models      map[string]Model     `yaml:"models"`
	Components  map[string]Component `yaml:"components"`
	Connections []Connection         `yaml:"connections"`
}

// Port declares one named port in an architecture's input or output
// contract — what another architecture sees when it imports this one as an
// orchestration component.
type Port struct {
	Type string `yaml:"type"`
}

// Model declares a named model slot, resolved to a provider at run time.
type Model struct {
	Provider string `yaml:"provider"`
	Name     string `yaml:"name,omitempty"`
}

// Component declares a node in the execution graph. For type "agent",
// Model and Prompt apply; for type "orchestration", Source (and optionally
// Version) name another ORH package to inline as this component.
type Component struct {
	Type    string `yaml:"type"`
	Model   string `yaml:"model,omitempty"`
	Prompt  string `yaml:"prompt,omitempty"`
	Source  string `yaml:"source,omitempty"`
	Version string `yaml:"version,omitempty"`
	// Tools lists the tools (as "alias.toolName", where alias is a key in
	// the owning package's orh.yaml dependencies) this agent may call.
	Tools []string `yaml:"tools,omitempty"`
	// Skills lists the skills whose instructions are appended to this
	// agent's prompt — each entry either references a `kind: skill`
	// package (a dependency alias) or is written directly inline. See
	// SkillRef.
	Skills []SkillRef `yaml:"skills,omitempty"`
	// Memory opts this agent into loading/saving conversation history
	// across Handle calls (see internal/memory). Scoped in-process by
	// default; persisted across separate `orh run` invocations when the
	// CLI is given --session.
	Memory bool `yaml:"memory,omitempty"`
}

// Connection wires the output of one component (or the graph input) to the
// input of another (or the graph output).
type Connection struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

const (
	// NodeInput and NodeOutput are the reserved graph boundary node names
	// usable on either side of a Connection.
	NodeInput  = "input"
	NodeOutput = "output"
)
