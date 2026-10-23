# ADR-0024: Rust File-Scoped Lookup Backend

* **Status**: Accepted
* **Date**: 2026-09-17

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Original trusted Rust lookup boundary; ADR-0030 and ADR-0051 extend rename/outline. Trust and disabled build execution remain, rather than a permanent lookup-only capability limit. |
| description | Rust lookup needs project-aware symbol information, but this slice must remain read-only and must not turn Cargo manifests or project build scripts into an implicit execution surface. `rust-analyzer` exposes the required hierarchical document symbols through LSP and can run as a managed native process. |
| role | contract |
| scope | source-code |
| area | language-server |
| languages | rust |

## Context

Rust lookup needs project-aware symbol information, but this slice must remain read-only and must not turn Cargo manifests or project build scripts into an implicit execution surface. `rust-analyzer` exposes the required hierarchical document symbols through LSP and can run as a managed native process.

## Decision

1. Register Rust only for `resolve_symbol_location` / CLI `lookup`. Rename, verification, formatting, imports, dependency changes, and structural edits remain unavailable and unadvertised.
2. Require a selected `.rs` file and either an explicit workspace root or a deterministic ancestor `Cargo.toml` discovery. Multiple ancestor manifests are ambiguous; the adapter never guesses or invokes Cargo.
3. Require request-scoped `WorkspaceTrust` before binary discovery or process launch. Use only a preinstalled `rust-analyzer`, with initialization settings that disable build scripts, proc macros, and check-on-save. These settings do not claim a complete sandbox.
4. Use one managed LSP session per trusted canonical workspace in the Rust backend. The session streams initialize, initialized, didOpen, and documentSymbol, negotiates UTF-16 positions, and closes explicitly through the backend lifecycle.
5. Interpret hierarchical `DocumentSymbol` responses only. Resolve exact names or exact `::`-qualified hierarchy, return every candidate on ambiguity, and locate symbols with `selectionRange`.

## Invariants

* Rust lookup never writes source, manifests, or dependency state.
* No untrusted request reaches `exec.LookPath` or starts a process.
* Out-of-root files and symbol URIs are rejected.
* Macros, generated symbols, locals, ambiguous trait methods, and multiple impl resolution are outside the advertised subset.
* The Rust session cannot silently coexist with another language session in the same backend lifecycle.

## Consequences

Rust projects gain deterministic file-scoped lookup while preserving the trust, manifest, and single-language lifecycle boundaries. Real rust-analyzer integration remains opt-in at runtime; normal tests inject a fake session transport.
