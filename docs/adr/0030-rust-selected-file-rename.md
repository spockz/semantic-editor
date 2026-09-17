# ADR-0030: Rust Selected-File Semantic Rename

Status: Accepted  
Date: 2026-09-17

## Context

Rust rename requires an external rust-analyzer process, but workspace mutation must remain bounded and trusted.

## Decision

Rust supports trusted semantic rename only for the selected canonical regular `.rs` file. The backend resolves the symbol, asks rust-analyzer to prepare and calculate the rename, validates the complete single-file edit, and atomically writes the result. Cargo and diagnostics are not invoked.

## Invariants

- Trust is checked before process discovery or launch.
- Foreign files, resources, annotations, malformed UTF-16 ranges, overlap, version mismatch, and stale preimages are rejected before writing.
- Session cleanup failure after commit is not reported as rename failure.

## Consequences

Rust rename is intentionally narrower than Go rename and does not claim cross-file support.
