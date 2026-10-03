# ADR-0005: Workspace Manifest Safety & Monorepo Scoping

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

In multi-project repositories (e.g. Go monorepos with multiple `go.mod` files, Cargo workspaces, or multi-package TypeScript setups), language servers require workspace coordination to resolve cross-project symbols.

However, automatically generating or modifying workspace configuration files (such as `go.work`) on disk can corrupt existing developer environments, dirty git working trees, or introduce unintended file changes. Furthermore, running multiple language servers for different languages concurrently can cause host memory exhaustion.

## Decision

1. **Strict Manifest Invariant**: `semedit` will **never** automatically create or modify a workspace manifest (`go.work`, `Cargo.toml`, `pom.xml`, etc.) on disk without explicit user approval.
2. **In-Memory LSP Workspaces**: When multiple disconnected sub-projects exist, `semedit` passes the directory roots in-memory to the language server via LSP `workspaceFolders` or linked-project configurations.
3. **Single-Language-Per-Session Scoping (Phase 2)**: Concurrent polyglot language servers are scoped to one language server type per workspace session, as adopted through [ADR-0009](0009-mcp-coexistence-and-tool-scoping.md). Each incremental backend ([ADR-0024](0024-rust-file-scoped-lookup-backend.md) through [ADR-0037](0037-java-bounded-maven-actions.md), [ADR-0047](0047-kotlin-read-only-backend.md), [ADR-0048](0048-bash-and-makefile-read-only-backends.md)) registers independently; the active session's tool surface is dynamically scoped to enabled backends. Cross-language polyglot refactorings (e.g. editing Rust FFI and Python bindings simultaneously) remain unsupported.

## Invariants

* No workspace manifest may be written to disk without explicit user confirmation.
* Only one language server type is active per workspace session; single-language-per-session scoping is enforced through the dynamic backend registry ([ADR-0009](0009-mcp-coexistence-and-tool-scoping.md)).

## Consequences

* **Positive**: Guarantees zero unprompted git pollution, prevents host RAM starvation, and supports read-only lookup across multiple languages within a single session without cross-language mutation.
* **Negative**: Cross-language polyglot refactorings (e.g. editing Rust FFI and Python bindings simultaneously) remain unsupported.
