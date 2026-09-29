# ADR-0049: Semantic Construct Movement

* **Status**: Accepted
* **Date**: 2026-09-26

## Context

Declaration relocation preserves existing source and comments without asking a caller to reproduce the construct. Same-file movement is a textual AST relocation; cross-file movement also changes file-scoped imports and package-level placement. Cross-package movement can affect references, imports, visibility, module boundaries, and build variants.

## Decision

Provide `semantic_move_construct` as a Go operation with `file` selecting the source declaration and `destination_file` selecting its destination. The operation returns those same fields and the primary target `symbol` as copyable continuations. It preserves declaration comments and grouped declaration integrity, validates source and destination before publication, and uses atomic writes for each file.

The current supported scope is same-file and same-directory Go package declarations. Cross-package movement fails closed until reference, import, visibility, and dependency completeness can be established for a bounded subset. Files with differing build constraints also fail closed. Per-file atomic publication does not promise a transaction across multiple files; if a later publication step fails, the operation reports which earlier file was written.

## Invariants

* `file` always names the source file and `destination_file` always names the destination file.
* `symbol` identifies only the declaration being moved; placement anchors use `target_symbol`.
* A move never accepts replacement source text.
* Cross-package moves preserve the exported API and update all supported references and imports, or refuse before writing.
* Each file write follows ADR-0010; multi-file publication reports partial failures honestly.

## Consequences

* Callers can relocate source constructs with copyable file and symbol fields and receive diagnostic deltas.
* Scope remains explicit; unsupported cases fail closed until reference completeness is proven.
* Multiple per-file atomic writes can leave a reported partial result if a later write fails.
