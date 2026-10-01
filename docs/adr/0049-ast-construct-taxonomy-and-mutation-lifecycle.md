# ADR-0049: Semantic Construct Movement

* **Status**: Accepted
* **Date**: 2026-09-30

## Context

Declaration relocation preserves existing source and comments without asking a caller to reproduce the construct. Same-file movement is an AST relocation; cross-file movement also changes imports and package placement. Cross-package movement affects references, imports, visibility, module boundaries, and build variants, so it needs a compiler-backed reference set and complete preflight.

## Decision

Provide `semantic_move_construct` as a Go operation with `file` selecting the source declaration and optional `destination_file` selecting its destination. The operation returns `file`, `destination_file`, and the primary target `symbol` as copyable continuations. It preserves declaration comments and grouped declaration integrity, validates every affected file and type relationship before publication, and uses atomic writes for each file.

Same-file and same-package moves retain their existing declaration scope. In `semantic_batch`, cross-file and cross-package moves consume the current unpublished source revision. The bounded cross-package subset moves an exported free function between existing files in distinct packages of one Go module. It resolves references through typed package information, updates callers and imports throughout the module, and typechecks an in-memory overlay before writing. It refuses receiver methods, unexported symbols, source-package dependencies, import cycles, unresolved package references, symlink ambiguity, generated or build-constrained selected files, and generated or build-constrained files that might reference the moved symbol. Unrelated generated or build-constrained files do not block the operation. Per-file atomic publication does not promise a transaction across multiple files; if a later publication step fails, the operation reports which earlier files were written.

Buffered moves validate and apply all planned source, destination, and reference
edits together. Cross-package loads bind module manifests and Go inputs, include earlier
buffered edits, discover and bind editable local replacements until stable, and
use read-only module loading. The final typecheck merges planned outputs over the
complete working overlay. All outputs retain their expected working preimages.
Same-package moves retain structural scope, including test and matching inactive
files, and do not require unrelated intermediate batch revisions to typecheck.
Their import-name resolution uses the same bound overlay and module policy.
Subsequent operations select the destination file and resolve the moved symbol
from that working revision. Content-only moves reuse the MCP-owned gopls process
under ADR-0060; publication remains the batch engine's responsibility.

## Invariants

* `file` always names the source file and `destination_file` always names the destination file.
* `symbol` identifies only the declaration being moved; placement anchors use `target_symbol`.
* A move never accepts replacement source text.
* Cross-package moves are limited to exported free functions in one Go module; they update all type-resolved references and imports or refuse before writing. Batched moves retain selected-path workspace checks and validate the current buffer state rather than rejecting directory changes. A batch receipt identifies requested source and destination paths; `final_diff` is authoritative about persisted writes.
* A failed buffered move contributes no output changes; completed earlier batch children retain the existing partial-application contract.
* Each file write follows ADR-0010; multi-file publication reports partial failures honestly.

## Consequences

* Callers can relocate source constructs with copyable file and symbol fields and receive diagnostic deltas.
* Scope remains explicit; source-local dependencies and hidden or ambiguous reference sets fail closed.
* Multiple per-file atomic writes can leave a reported partial result if a later write fails.
