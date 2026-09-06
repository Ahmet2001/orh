package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

const validManifest = `
apiVersion: orh/v1
kind: orchestration
name: debate-pro
version: 1.0.0
entrypoint: main.orh
author:
  github: rifat
dependencies: []
`

const validArch = `
apiVersion: orh/v1
name: debate-pro
models:
  main:
    provider: ollama
    name: qwen3
components:
  assistant:
    type: agent
    model: main
    prompt: hi
connections:
  - from: input
    to: assistant.input
  - from: assistant.output
    to: output
`

func writePackage(t *testing.T, manifest, arch string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	if arch != "" {
		if err := os.WriteFile(filepath.Join(dir, "main.orh"), []byte(arch), 0o644); err != nil {
			t.Fatalf("writing entrypoint: %v", err)
		}
	}
	return dir
}

func TestLoadAndValidate_Valid(t *testing.T) {
	dir := writePackage(t, validManifest, validArch)

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	result := p.Validate()
	if !result.Valid() {
		t.Fatalf("expected valid package, got errors: %v", result)
	}
	if p.Entrypoint == nil {
		t.Fatal("expected Entrypoint to be populated after Validate")
	}
}

func TestLoad_MissingManifest(t *testing.T) {
	dir := t.TempDir()

	if _, err := Load(dir); err == nil {
		t.Fatal("expected error for missing orh.yaml, got nil")
	}
}

func TestValidate_UnsupportedAPIVersion(t *testing.T) {
	manifest := `
apiVersion: orh/v2
kind: orchestration
name: debate-pro
entrypoint: main.orh
`
	dir := writePackage(t, manifest, validArch)

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if result := p.Validate(); result.Valid() {
		t.Fatal("expected invalid package for unsupported apiVersion")
	}
}

func TestValidate_MissingEntrypointFile(t *testing.T) {
	dir := writePackage(t, validManifest, "")

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if result := p.Validate(); result.Valid() {
		t.Fatal("expected invalid package for missing entrypoint file")
	}
}

const validToolboxManifest = `
apiVersion: orh/v1
kind: toolbox
name: web-tools
version: 1.0.0
entrypoint: web.tb
author:
  github: rifat
dependencies: []
`

const validTB = `
version: 1
tools:
  search:
    description: Search the web
    runtime:
      type: mcp
      server: "web-search-server"
`

func writeToolboxPackage(t *testing.T, manifest, tb string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	if tb != "" {
		if err := os.WriteFile(filepath.Join(dir, "web.tb"), []byte(tb), 0o644); err != nil {
			t.Fatalf("writing toolbox: %v", err)
		}
	}
	return dir
}

func TestLoadAndValidate_ToolboxPackage(t *testing.T) {
	dir := writeToolboxPackage(t, validToolboxManifest, validTB)

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	result := p.Validate()
	if !result.Valid() {
		t.Fatalf("expected a valid toolbox package, got errors: %v", result)
	}
	if p.Toolbox == nil {
		t.Fatal("expected Toolbox to be populated after Validate")
	}
	if p.Entrypoint != nil {
		t.Error("expected Entrypoint to stay nil for a toolbox package")
	}
	if _, ok := p.Toolbox.Tools["search"]; !ok {
		t.Errorf("expected tool %q in parsed toolbox", "search")
	}
}

const validNodeManifest = `
apiVersion: orh/v1
kind: node
name: smart-router
version: 1.0.0
entrypoint: router.node
author:
  github: rifat
dependencies: []
`

const validNode = `
apiVersion: orh/v1
kind: node
name: smart-router
version: 1.0.0
runtime:
  type: wasm
entrypoint:
  module: router.wasm
`

func writeNodePackage(t *testing.T, manifest, node string, wasmBytes []byte) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	if node != "" {
		if err := os.WriteFile(filepath.Join(dir, "router.node"), []byte(node), 0o644); err != nil {
			t.Fatalf("writing node manifest: %v", err)
		}
	}
	if wasmBytes != nil {
		if err := os.WriteFile(filepath.Join(dir, "router.wasm"), wasmBytes, 0o644); err != nil {
			t.Fatalf("writing wasm module: %v", err)
		}
	}
	return dir
}

func TestLoadAndValidate_NodePackage(t *testing.T) {
	fakeWASM := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	dir := writeNodePackage(t, validNodeManifest, validNode, fakeWASM)

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	result := p.Validate()
	if !result.Valid() {
		t.Fatalf("expected a valid node package, got errors: %v", result)
	}
	if p.Node == nil {
		t.Fatal("expected Node to be populated after Validate")
	}
	if p.Entrypoint != nil {
		t.Error("expected Entrypoint to stay nil for a node package")
	}
}

func TestValidate_NodePackage_InvalidWASM(t *testing.T) {
	dir := writeNodePackage(t, validNodeManifest, validNode, []byte("not wasm"))

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if result := p.Validate(); result.Valid() {
		t.Fatal("expected invalid package for a non-WASM module file")
	}
}

func TestValidate_NodePackage_MissingModule(t *testing.T) {
	dir := writeNodePackage(t, validNodeManifest, validNode, nil)

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if result := p.Validate(); result.Valid() {
		t.Fatal("expected invalid package for a missing wasm module file")
	}
}

const validSkillManifest = `
apiVersion: orh/v1
kind: skill
name: code-review-checklist
version: 1.0.0
entrypoint: SKILL.md
author:
  github: rifat
dependencies: []
`

const validSkillMD = `---
description: Applies a code review checklist
---

Review the code for correctness and security.
`

func writeSkillPackage(t *testing.T, manifest, skillMD string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	if skillMD != "" {
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
			t.Fatalf("writing SKILL.md: %v", err)
		}
	}
	return dir
}

func TestLoadAndValidate_SkillPackage(t *testing.T) {
	dir := writeSkillPackage(t, validSkillManifest, validSkillMD)

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	result := p.Validate()
	if !result.Valid() {
		t.Fatalf("expected a valid skill package, got errors: %v", result)
	}
	if p.Skill == nil {
		t.Fatal("expected Skill to be populated after Validate")
	}
	if p.Skill.Description != "Applies a code review checklist" {
		t.Errorf("Skill.Description = %q, want %q", p.Skill.Description, "Applies a code review checklist")
	}
	if p.Entrypoint != nil {
		t.Error("expected Entrypoint to stay nil for a skill package")
	}
}

func TestValidate_SkillPackage_EmptyInstructions(t *testing.T) {
	dir := writeSkillPackage(t, validSkillManifest, "---\ndescription: nothing\n---\n\n")

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if result := p.Validate(); result.Valid() {
		t.Fatal("expected invalid package for a skill with empty instructions")
	}
}

func TestValidate_BadDependencyFormat(t *testing.T) {
	manifest := `
apiVersion: orh/v1
kind: orchestration
name: debate-pro
entrypoint: main.orh
dependencies:
  debate:
    source: ""
    version: v1.0.0
`
	dir := writePackage(t, manifest, validArch)

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if result := p.Validate(); result.Valid() {
		t.Fatal("expected invalid package for dependency missing a source")
	}
}
