// Package pkg loads and validates an ORH package: an orh.yaml manifest plus
// the .orh architecture it points at as its entrypoint.
package pkg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/pertevniyalai/orh/internal/nodes"
	"github.com/pertevniyalai/orh/internal/parser"
	"github.com/pertevniyalai/orh/internal/skill"
	"github.com/pertevniyalai/orh/internal/spec"
	"github.com/pertevniyalai/orh/internal/toolbox"
	"github.com/pertevniyalai/orh/internal/validator"
)

// SupportedAPIVersion is the only orh.yaml apiVersion this build understands.
const SupportedAPIVersion = "orh/v1"

// KindOrchestration identifies a package whose entrypoint is a .orh
// architecture. KindToolbox identifies a package whose entrypoint is a .tb
// toolbox. KindNode identifies a package whose entrypoint is a .node
// manifest describing a WASM-sandboxed custom component.
// KindSkill identifies a package whose entrypoint is a SKILL.md file: a
// reusable instruction module an agent can attach via its `skills:` list.
const (
	KindOrchestration = "orchestration"
	KindToolbox       = "toolbox"
	KindNode          = "node"
	KindSkill         = "skill"
)

var supportedKinds = map[string]bool{KindOrchestration: true, KindToolbox: true, KindNode: true, KindSkill: true}

// wasmMagic is the 4-byte header every WASM binary starts with.
var wasmMagic = []byte{0x00, 0x61, 0x73, 0x6d}

// ManifestFileName is the required manifest filename at a package's root.
const ManifestFileName = "orh.yaml"

// Package is a loaded ORH package: its manifest, and — once Validate has
// succeeded — its parsed entrypoint. For an orchestration package that's
// Entrypoint; for a toolbox package that's Toolbox; for a node package
// that's Node.
type Package struct {
	Dir        string
	Manifest   spec.Manifest
	Entrypoint *spec.Architecture
	Toolbox    *spec.Toolbox
	Node       *spec.Node
	Skill      *spec.Skill
}

// ValidationError is returned when a package's manifest or entrypoint fails
// validation.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("package validation failed with %d error(s)", len(e.Errors))
}

// Load reads and parses a package's orh.yaml manifest from dir. It does not
// validate the manifest or load the entrypoint — call Validate for that.
func Load(dir string) (*Package, error) {
	path := filepath.Join(dir, ManifestFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var manifest spec.Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &Package{Dir: dir, Manifest: manifest}, nil
}

// Validate checks the manifest fields and the entrypoint .orh file,
// populating Package.Entrypoint on success.
func (p *Package) Validate() ValidationErrors {
	var errs []string

	if p.Manifest.APIVersion != SupportedAPIVersion {
		errs = append(errs, fmt.Sprintf("unsupported apiVersion %q (expected %q)", p.Manifest.APIVersion, SupportedAPIVersion))
	}
	if !supportedKinds[p.Manifest.Kind] {
		errs = append(errs, fmt.Sprintf("unsupported kind %q", p.Manifest.Kind))
	}
	if p.Manifest.Name == "" {
		errs = append(errs, "name is required")
	}
	if p.Manifest.Entrypoint == "" {
		errs = append(errs, "entrypoint is required")
	}
	for name, dep := range p.Manifest.Dependencies {
		if dep.Source == "" {
			errs = append(errs, fmt.Sprintf("dependency %q: source is required", name))
		}
		if dep.Version == "" {
			errs = append(errs, fmt.Sprintf("dependency %q: version is required", name))
		}
	}

	if len(errs) > 0 {
		return ValidationErrors(errs)
	}

	entrypointPath := filepath.Join(p.Dir, p.Manifest.Entrypoint)

	switch p.Manifest.Kind {
	case KindToolbox:
		tb, err := toolbox.ParseFile(entrypointPath)
		if err != nil {
			return ValidationErrors{fmt.Sprintf("entrypoint %q: %s", p.Manifest.Entrypoint, err)}
		}
		if tbErrs := toolbox.Validate(tb); len(tbErrs) > 0 {
			for _, e := range tbErrs {
				errs = append(errs, fmt.Sprintf("entrypoint: %s", e))
			}
			return ValidationErrors(errs)
		}
		p.Toolbox = tb

	case KindNode:
		n, err := nodes.ParseFile(entrypointPath)
		if err != nil {
			return ValidationErrors{fmt.Sprintf("entrypoint %q: %s", p.Manifest.Entrypoint, err)}
		}
		if nErrs := nodes.Validate(n); len(nErrs) > 0 {
			for _, e := range nErrs {
				errs = append(errs, fmt.Sprintf("entrypoint: %s", e))
			}
			return ValidationErrors(errs)
		}

		modulePath := filepath.Join(p.Dir, n.Entrypoint.Module)
		header := make([]byte, 4)
		f, err := os.Open(modulePath)
		if err != nil {
			return ValidationErrors{fmt.Sprintf("module %q: %s", n.Entrypoint.Module, err)}
		}
		_, readErr := f.Read(header)
		f.Close()
		if readErr != nil || !bytes.Equal(header, wasmMagic) {
			return ValidationErrors{fmt.Sprintf("module %q: not a valid WASM binary", n.Entrypoint.Module)}
		}

		p.Node = n

	case KindSkill:
		s, err := skill.ParseFile(entrypointPath)
		if err != nil {
			return ValidationErrors{fmt.Sprintf("entrypoint %q: %s", p.Manifest.Entrypoint, err)}
		}
		if sErrs := skill.Validate(s); len(sErrs) > 0 {
			for _, e := range sErrs {
				errs = append(errs, fmt.Sprintf("entrypoint: %s", e))
			}
			return ValidationErrors(errs)
		}
		p.Skill = s

	default: // KindOrchestration
		arch, err := parser.ParseFile(entrypointPath)
		if err != nil {
			return ValidationErrors{fmt.Sprintf("entrypoint %q: %s", p.Manifest.Entrypoint, err)}
		}
		archResult := validator.Validate(arch)
		if !archResult.Valid() {
			for _, e := range archResult.Errors {
				errs = append(errs, fmt.Sprintf("entrypoint: %s", e))
			}
			return ValidationErrors(errs)
		}
		p.Entrypoint = arch
	}

	return nil
}

// ValidationErrors is a possibly-empty list of validation error messages.
type ValidationErrors []string

// Valid reports whether there were no validation errors.
func (v ValidationErrors) Valid() bool { return len(v) == 0 }
