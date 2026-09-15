# ADR-0005: Workspace Manifest Safety & Monorepo Scoping

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

In multi-project repositories (e.g. Go monorepos with multiple `go.mod` files, Cargo workspaces, or multi-package TypeScript setups), language servers require workspace coordination to resolve cross-project symbols.

However, automatically generating or modifying workspace configuration files (such as `go.work`) on disk can corrupt existing developer environments, dirty git working trees, or introduce unintended file changes. Furthermore, running multiple language servers for different languages concurrently can cause host memory exhaustion.

## Decision

1. **Strict Manifest Invariant**: `semedit` will **never** automatically create or modify a workspace manifest (`go.work`, `Cargo.toml`, `pom.xml`, etc.) on disk without explicit user approval.
2. **In-Memory LSP Workspaces**: When multiple disconnected sub-projects exist, `semedit` passes the directory roots in-memory to the language server via LSP `workspaceFolders` or linked-project configurations.
3. **Single-Language Focus (Phase 1)**: Concurrent polyglot language servers running simultaneously in a single workspace are explicitly postponed to Phase 2. `semedit` scopes its server lifecycle to the active project's language.

## Invariants

* No workspace manifest may be written to disk without explicit user confirmation.
* Only one language server type is active per workspace session in Phase 1.

## Consequences

* **Positive**: Guarantees zero unprompted git pollution and prevents host RAM starvation.
* **Negative**: Cross-language polyglot refactorings (e.g. editing Rust FFI and Python bindings simultaneously) are unsupported in Phase 1.
