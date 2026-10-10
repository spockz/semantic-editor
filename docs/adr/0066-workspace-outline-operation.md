<!-- WHY: Records the architectural decision to introduce workspace_outline for repository package orientation and multi-module discovery. -->
# Architecture Decision Record (ADR) 0066: Workspace Outline and Multi-Module Discovery

* **Status**: Accepted
* **Date**: 2026-10-10

## Context

When navigating unfamiliar codebases, Artificial Intelligence (AI) coding agents need high-level orientation before inspecting declarations or editing code. Previously, agents faced two extremes:

1. **Unconstrained Shell Scanning**: Executing `find`, `tree`, or `ls` commands, which consumes context budget, risks tool rejection under read restrictions, and yields unannotated filesystem trees without package or module context.
2. **Declaration-Level Outlines**: Running `semantic_outline` (`semedit outline`), which projects symbol-level declaration trees (types, methods, functions) for a specific file or directory. When run on repository roots, it emits thousands of symbol definitions, overflowing the context budget.

Furthermore, multi-module repositories (e.g. Go monorepos with submodules in `tools/`, `cmd/`, or subpackages declaring separate `go.mod` files) lacked a deterministic discovery mechanism to locate auxiliary module roots without exploratory text searching.

## Decision

We introduce a dedicated, zero-token read operation: **`workspace_outline`** (`semedit workspace-outline` / `semantic_workspace_outline`):

1. **Workspace Scope (`LevelWorkspace`)**:
   The operation inspects the active workspace root (or an optional scoped subtree) and discovers all modules and packages. It operates strictly at directory and package boundaries without requiring individual source file arguments.
2. **Depth-Ordered Package Listing**:
   The engine orders discovered packages primarily by directory depth (root level first, then one level deeper, etc.) and secondarily alphabetically by path. Each entry reports:
   * Relative directory path.
   * Package identifier.
   * Single-line package synopsis extracted from `doc.go` or package doc comments.
   * Total source file count.
3. **Module Manifest Discovery**:
   The engine discovers all module boundary manifests (`go.mod` files) across the repository and classifies them into the root module versus nested submodules, recording their declared module path and language version.
4. **Dual Output Delivery**:
   * **Command-Line Interface (CLI) and Model Context Protocol (MCP) Text Content**: Emits clean, depth-ordered lines mimicking `ls`, minimising token consumption (~10 tokens per package) and allowing linear Large Language Model (LLM) attention scanning.
   * **Structured MCP Schema**: Returns typed `WorkspaceOutlineResult` for automated harnesses and programmatic clients.
5. **Bidirectional Navigation Integration**:
   * MCP server prescriptive instructions recommend `semantic_workspace_outline` as the primary starting orientation tool before inspecting declarations or mutating code.
   * `semantic_outline` refers callers to `semantic_workspace_outline` to determine candidate package directories.

## Invariants

* Read operations preserve workspace immutability: zero disk writes, unmodified git trees, and intact file timestamps.
* Package discovery respects standard directory exclusions (`.git`, `.scratch`, `vendor`, `node_modules`, `.cache`, build targets).
* Depth ordering is deterministic: packages at shallower directory depths strictly precede deeper packages.
* Discovery fails closed on unreadable or non-directory target paths.

## Consequences

* **Positive**: Provides agents with a compact, token-efficient table of contents for any workspace; eliminates blind shell exploration; cleanly separates repository-level orientation (`workspace_outline`) from symbol-level inspection (`semantic_outline`).
* **Negative**: Requires scanning directory trees and parsing package clauses, though restricted to header parsing (`parser.PackageClauseOnly`) to maintain sub-second latency.
