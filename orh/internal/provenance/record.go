// Package provenance records enough information about an ORH execution to
// inspect and reconstruct the declared computational process.
package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pertevniyalai/orh/internal/runtime/events"
	"github.com/pertevniyalai/orh/internal/spec"
)

const SchemaVersion = "orh.run/v1"

type Record struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Runtime       Runtime                 `json:"runtime"`
	Architecture  Architecture            `json:"architecture"`
	Models        map[string]ModelBinding `json:"models,omitempty"`
	Dependencies  []Dependency            `json:"dependencies,omitempty"`
	Input         Value                   `json:"input"`
	Outputs       []Value                 `json:"outputs,omitempty"`
	Trace         []Step                  `json:"trace,omitempty"`
	StartedAt     time.Time               `json:"startedAt"`
	CompletedAt   time.Time               `json:"completedAt"`
	Status        string                  `json:"status"`
	Error         string                  `json:"error,omitempty"`
}

type Runtime struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Architecture struct {
	Name       string `json:"name,omitempty"`
	Source     string `json:"source"`
	SourceHash string `json:"sourceHash,omitempty"`
	GraphHash  string `json:"graphHash,omitempty"`
}

type ModelBinding struct {
	Provider string `json:"provider"`
	Name     string `json:"name,omitempty"`
}

type Dependency struct {
	Alias  string `json:"alias"`
	Source string `json:"source"`
	Commit string `json:"commit,omitempty"`
}

type Value struct {
	Source string `json:"source,omitempty"`
	Port   string `json:"port,omitempty"`
	Text   string `json:"text"`
	Hash   string `json:"hash"`
}

type Step struct {
	Sequence int       `json:"sequence"`
	Kind     string    `json:"kind"`
	Value    string    `json:"value,omitempty"`
	At       time.Time `json:"at"`
}

func New(sourcePath, input string) *Record {
	r := &Record{
		SchemaVersion: SchemaVersion,
		Runtime:       Runtime{Name: "orh", Version: "0.1.0-dev"},
		Architecture:  Architecture{Source: sourcePath},
		Models:        map[string]ModelBinding{},
		Input:         value("", "", input),
		StartedAt:     time.Now().UTC(),
		Status:        "running",
	}
	if data, err := os.ReadFile(sourcePath); err == nil {
		r.Architecture.SourceHash = hash(data)
	}
	return r
}

func (r *Record) SetArchitecture(arch *spec.Architecture, modelOverride string, deps spec.Dependencies) error {
	r.Architecture.Name = arch.Name
	encoded, err := json.Marshal(arch)
	if err != nil {
		return fmt.Errorf("encoding resolved architecture for provenance: %w", err)
	}
	r.Architecture.GraphHash = hash(encoded)

	for slot, model := range arch.Models {
		binding := ModelBinding{Provider: model.Provider, Name: model.Name}
		if modelOverride != "" {
			parts := strings.SplitN(modelOverride, ":", 2)
			binding.Provider = parts[0]
			if len(parts) == 2 {
				binding.Name = parts[1]
			}
		}
		r.Models[slot] = binding
	}

	aliases := make([]string, 0, len(deps))
	for alias := range deps {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		dep := deps[alias]
		r.Dependencies = append(r.Dependencies, Dependency{Alias: alias, Source: dep.Source, Commit: dep.Version})
	}
	return nil
}

func (r *Record) AddStep(kind, stepValue string) {
	r.Trace = append(r.Trace, Step{Sequence: len(r.Trace) + 1, Kind: kind, Value: stepValue, At: time.Now().UTC()})
}

func (r *Record) Finish(outputs []events.Event, runErr error) {
	r.CompletedAt = time.Now().UTC()
	if runErr != nil {
		r.Status = "failed"
		r.Error = runErr.Error()
		return
	}
	r.Status = "completed"
	for _, output := range outputs {
		r.Outputs = append(r.Outputs, value(output.Source, output.Port, output.Text()))
	}
}

func (r *Record) Save(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding run record: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("creating run record directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing run record: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("committing run record: %w", err)
	}
	return nil
}

func value(source, port, text string) Value {
	return Value{Source: source, Port: port, Text: text, Hash: hash([]byte(text))}
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
