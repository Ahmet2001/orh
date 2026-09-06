package tools

import "sort"

// Registry looks tools up by their fully qualified name ("alias.toolName").
type Registry struct {
	tools map[string]Tool
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Register adds a tool under the given fully qualified name.
func (r *Registry) Register(fullName string, t Tool) {
	r.tools[fullName] = t
}

// Get looks up a tool by its fully qualified name.
func (r *Registry) Get(fullName string) (Tool, bool) {
	t, ok := r.tools[fullName]
	return t, ok
}

// Names returns every registered tool's fully qualified name, sorted.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
