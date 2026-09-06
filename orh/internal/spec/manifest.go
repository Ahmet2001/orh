package spec

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Manifest is the root document parsed from a package's orh.yaml.
type Manifest struct {
	APIVersion   string       `yaml:"apiVersion"`
	Kind         string       `yaml:"kind"`
	Name         string       `yaml:"name"`
	Version      string       `yaml:"version"`
	Entrypoint   string       `yaml:"entrypoint"`
	Author       Author       `yaml:"author"`
	Dependencies Dependencies `yaml:"dependencies"`
	// Permissions is a declarative allow-list a package can publish for
	// the tools it exposes or depends on. ORH checks it as a pre-flight
	// gate before a tool call; it is not a sandbox, since the tool itself
	// usually runs in another process ORH doesn't control.
	Permissions Permissions `yaml:"permissions,omitempty"`
}

// Permissions is a package's declared allow-list. An empty Permissions
// (the common case) means "no restrictions declared" — ORH does not
// default to denying anything.
type Permissions struct {
	Network    NetworkPermission    `yaml:"network,omitempty"`
	Filesystem FilesystemPermission `yaml:"filesystem,omitempty"`
}

// NetworkPermission lists hosts a package's tools are allowed to reach.
// An empty Allow list means "no restriction declared".
type NetworkPermission struct {
	Allow []string `yaml:"allow,omitempty"`
}

// FilesystemPermission declares filesystem access a package's tools need.
type FilesystemPermission struct {
	Read bool `yaml:"read,omitempty"`
}

// Author identifies who published a package.
type Author struct {
	GitHub string `yaml:"github"`
}

// Dependency references another ORH package a package depends on.
type Dependency struct {
	Source  string `yaml:"source"`
	Version string `yaml:"version"`
}

// Dependencies accepts either a mapping of name -> Dependency, or an empty
// sequence ("dependencies: []") to mean "no dependencies" — both forms show
// up in example manifests.
type Dependencies map[string]Dependency

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Dependencies) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.SequenceNode {
		if len(value.Content) != 0 {
			return fmt.Errorf("dependencies: a non-empty list is not valid; use a mapping of name to {source, version}")
		}
		*d = Dependencies{}
		return nil
	}

	var m map[string]Dependency
	if err := value.Decode(&m); err != nil {
		return err
	}
	*d = Dependencies(m)
	return nil
}
