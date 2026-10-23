# ADR-0005: Workspace Manifest Safety & Monorepo Scoping

* **Status**: Accepted
* **Date**: 2026-09-15

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Never modify workspace manifests without approval. Enabled-backend unions supersede the original single-language session limit; concurrent languages do not imply shared caches or cross-language refactoring proofs. |
| description | In multi-project repositories (e.g. Go monorepos with multiple `go.mod` files, Cargo workspaces, or multi-package TypeScript setups), language servers require workspace coordination to resolve cross-project symbols. |
| role | invariant |
| scope | project |
| area | backend |
| languages | generic |

## Context

In multi-project repositories (e.g. Go monorepos with multiple `go.mod` files, Cargo workspaces, or multi-package TypeScript setups), language servers require workspace coordination to resolve cross-project symbols.

However, automatically generating or modifying workspace configuration files (such as `go.work`) on disk can corrupt existing developer environments, dirty git working trees, or introduce unintended file changes. Furthermore, running multiple language servers for different languages concurrently can cause host memory exhaustion.

## Decision

1. **Strict Manifest Invariant**: `semedit` will **never** automatically create or modify a workspace manifest (`go.work`, `Cargo.toml`, `pom.xml`, etc.) on disk without explicit user approval.
2. **In-Memory LSP Workspaces**: When multiple disconnected sub-projects exist, `semedit` passes the directory roots in-memory to the language server via LSP `workspaceFolders` or linked-project configurations.
3. **Enabled-Backend Scoping**: The original single-language session proposal is superseded by the enabled-backend union in ADR-0009 and multi-language verification/discovery in ADR-0054 and ADR-0066. Each request resolves its applicable backend; sessions retain their backend-owned lifecycle and workspace boundaries. Supporting several active languages does not advertise a coordinated cross-language refactoring such as changing Rust FFI and Python bindings together.

## Invariants

* No workspace manifest may be written to disk without explicit user confirmation.
* Tool discovery reflects enabled implemented backends; execution validates the selected backend. Multiple enabled languages do not imply shared diagnostic caches or a cross-language mutation proof.

## Consequences

* **Positive**: Prevents unapproved manifest changes and keeps backend activation and scope explicit. Per-backend lifecycle ownership avoids an unconditional promise about total host memory usage.
* **Negative**: Cross-language polyglot refactorings (e.g. editing Rust FFI and Python bindings simultaneously) remain unsupported.
