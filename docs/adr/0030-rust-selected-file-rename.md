# ADR-0030: Rust Selected-File Semantic Rename

Status: Accepted  
Date: 2026-09-17

## Decision

Trusted Rust rename is restricted to one selected canonical regular `.rs` file. The backend resolves the symbol, invokes rust-analyzer prepare/rename, strictly validates a single-file edit, and atomically writes it. Cargo and diagnostics are not invoked.

## Invariants

- Trust is checked before process discovery or launch.
- Foreign files, resources, annotations, malformed UTF-16 ranges, overlap, version mismatch, and stale preimages are rejected before writing.
- Session cleanup failure after commit is not reported as rename failure.

## Consequences

Rust rename is intentionally narrower than Go rename and does not claim cross-file support.
