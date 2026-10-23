<!-- WHY: Records the architectural decision to introduce workspace_outline for repository package orientation and multi-module discovery. -->
# ADR-0066: Workspace Outline and Multi-Module Discovery

* **Status**: Accepted
* **Date**: 2026-10-10

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Go/Java orientation preserves source/manifests and exact module scope. Benchmark artifacts are pruned before discovery, snapshots, verification copies, formatting, and native Go traversal; documentation reads bounded published-result paths. |
| description | Provide bounded package and module orientation before declaration-level inspection. |
| role | architecture |
| scope | tooling |
| area | backend |
| languages | go, java |

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
   Auto mode aggregates supported Go and Java ecosystems without making their presence mutually exclusive. Explicit language selection restricts discovery to that backend. Go discovery reads module manifests and `go.work` use directives. Scoped requests report the nearest enclosing module and nested modules within the requested subtree, with packages attributed to their exact owning module. Modules carry synopses, documentation, language ownership, and package counts for the selected scope.
4. **Dual Output Delivery**:
   * **Command-Line Interface (CLI)**: Emits depth-ordered package lines and module coordinates with synopses; mixed results identify their languages.
   * **Model Context Protocol (MCP)**: Returns complete package and module records once in structured content. Text contains a concise summary and critical scope/status metadata without repeating bulk arrays or serialized result strings. Declaration outlines instead use the compact navigation projection defined in ADR-0051; package and module records retain their own schema.
5. **Bidirectional Navigation Integration**:
   * MCP server prescriptive instructions recommend `semantic_workspace_outline` as the primary starting orientation tool before inspecting declarations or mutating code.
   * `semantic_outline` refers callers to `semantic_workspace_outline` to determine candidate package directories.

## Invariants

* Discovery does not mutate workspace source or manifests, their timestamps, or git-tracked files. Runtime/environment cache activity follows the selected backend policy; source immutability is not a universal zero-disk-I/O claim.
* Package discovery respects standard directory exclusions (`.git`, `.scratch`, `vendor`, `node_modules`, `.cache`, build targets).
* `data/benchmarks` is an artifact boundary, pruned before source discovery, buffered tree observation, snapshots, verification copies, and benchmark workspace copies descend into it. A source directory named `benchmarks` outside `data` is not excluded by this rule. Explicit file selection retains its existing semantics.
* The user-approved module boundary at `data/benchmarks/go.mod` excludes artifact trees from native Go package discovery, tests, fixers, and module tidying. Formatting prunes the same path directly; filtering package names after discovery is insufficient.
* Documentation intake is a separate, bounded reader: only result JSON at `results[/run[/task]]` is eligible. Archive, validation, checkpoint, evidence, hidden, dependency, and captured workspace directories are pruned before descent; candidate report validation remains strict.
* Depth ordering is deterministic: packages at shallower directory depths strictly precede deeper packages.
* Discovery fails closed on unreadable or non-directory target paths.
* Parent manifest ascent stops at the applicable repository boundary. Broad input paths never change the precision of module ownership in results.
* Discovery errors propagate with backend context; a failed backend never becomes an apparently complete empty result.
* Isolated documentation examples do not activate repository-wide Java discovery without a Java build manifest; an explicitly selected example directory can discover its sources.
* Reading `go.work` never authorizes creating or changing a workspace manifest.

## Consequences

* **Positive**: Provides agents with a compact, token-efficient table of contents for any workspace; eliminates blind shell exploration; cleanly separates repository-level orientation (`workspace_outline`) from symbol-level inspection (`semantic_outline`).
* **Negative**: Requires scanning directory trees and parsing package clauses, though restricted to header parsing (`parser.PackageClauseOnly`) to maintain sub-second latency.
