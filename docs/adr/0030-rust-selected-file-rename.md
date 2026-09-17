# ADR-0030: Rust Selected-File Semantic Rename

Status: Accepted  
Date: 2026-09-17

## Context

Rust rename requires an external rust-analyzer process, but workspace mutation must remain bounded and trusted.

## Decision

Rust supports trusted semantic rename only for the selected canonical regular `.rs` file. The backend resolves the symbol, asks rust-analyzer to prepare and calculate the rename, validates the complete single-file edit, and atomically writes the result. Cargo and diagnostics are not invoked.

## Invariants

- Trust is checked before process discovery or launch.
- Only one selected file, one valid LSP edit representation, and version 1 are accepted.
- Resource operations, annotations, foreign URIs, malformed UTF-16 ranges, overlap, and stale preimages are rejected before writing.
- Session cleanup after a successful commit does not turn the rename into a failure.

## Consequences

Rust rename is intentionally narrower than Go rename and does not claim cross-file support.
