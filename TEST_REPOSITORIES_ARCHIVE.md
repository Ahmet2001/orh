# ORH Test Repository Archive

This document preserves the purpose and structure of the temporary GitHub
repositories that were used while developing and validating ORH. The
repositories were intentionally disposable integration fixtures; their roles
are recorded here before removal so the project's development history remains
understandable.

## `Ahmet2001/orh-test-package`

- Phase: Phase 3 — GitHub package system
- Purpose: Verified the first end-to-end orchestration package flow: resolving
  a GitHub repository, validating `orh.yaml` and `main.orh`, caching it locally,
  and running the downloaded architecture with Ollama.
- Files: `README.md`, `orh.yaml`, `main.orh`
- Tags: none

## `Ahmet2001/orh-test-subagent`

- Phase: Phase 4 — architecture composition
- Purpose: Provided a reusable sub-orchestration with named input/output ports.
  It verified recursive package resolution, graph flattening, component/model
  prefixing, and execution of an imported architecture as part of a parent
  architecture.
- Files: `README.md`, `orh.yaml`, `main.orh`
- Tags: `v1.0.0`

## `Ahmet2001/orh-test-toolbox`

- Phase: Phase 5 — toolbox and MCP integration
- Purpose: Exposed a minimal `add` tool through a Python MCP server. It verified
  toolbox package loading, relative server command execution, MCP
  initialization and discovery, direct tool testing, and runtime tool calls.
- Files: `README.md`, `orh.yaml`, `math.tb`, `server.py`
- Tags: `v1.0.0`

## `Ahmet2001/orh-test-tool-user`

- Phase: Phase 5 — agent tool calling
- Purpose: Defined an orchestration package that depended on
  `orh-test-toolbox` and attached its `add` tool to an agent. It verified the
  complete architecture → dependency → MCP registry → model tool call →
  tool result → final answer loop with Ollama.
- Files: `orh.yaml`, `main.orh`
- Tags: none

## `Ahmet2001/orh-test-node`

- Phase: Phase 6 and 6.1 — custom WASM nodes
- Purpose: Contained the TinyGo source, compiled WASM module, and `.node`
  manifest for the `smart-router` integration fixture. Version `v1.0.0`
  verified payload-based dynamic routing. Version `v2.0.0` additionally
  verified the `orh_call_model` and `orh_call_tool` host ABI and enforcement of
  the node's model/tool allow-lists.
- Files: `README.md`, `orh.yaml`, `router.node`, `router.wasm`, `src/go.mod`,
  `src/main.go`
- Tags: `v1.0.0`, `v2.0.0`

## `Ahmet2001/orh-test-skill`

- Phase: Phase 6.2 — reusable skill packages
- Purpose: Supplied a versioned `kind: skill` package whose instruction asked
  an agent to answer with one uppercase word. It verified skill package
  resolution, `SKILL.md` parsing, prompt composition, and observable behavior
  changes in a real Ollama run.
- Files: `README.md`, `orh.yaml`, `SKILL.md`
- Tags: `v1.0.0`

## Removal

The six repositories above were removed from GitHub on September 6, 2026 after
their integration-test responsibilities had been documented here. Their
important behavior remains covered by the main ORH repository's unit and WASM
fixture tests.
