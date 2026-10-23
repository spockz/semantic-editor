# ADR-0049: Semantic Construct Movement

* **Status**: Accepted
* **Date**: 2026-09-30

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Go cross-package moves use bounded compiler proofs for functions, constants, aliases, variables, and defined-type groups. Java movement is backend-owned; whole-package rename and experimental upstream MoveDeclaration do not expand accepted semedit capabilities. |
| description | Declaration relocation preserves existing source and comments without asking a caller to reproduce the construct. Same-file movement is an AST relocation; cross-file movement also changes imports and package placement. Cross-package movement affects references, imports, visibility, module boundaries, and build variants, so it needs a compiler-backed reference set and complete preflight. Go cross-package declaration forms share the proof and publication boundary in ADR-0102, with declaration-specific limits stated there. |
| role | architecture |
| scope | source-code |
| area | backend |
| languages | generic |

## Context

Declaration relocation preserves existing source and comments without asking a caller to reproduce the construct. Same-file movement is an AST relocation; cross-file movement also changes imports and package placement. Cross-package movement affects references, imports, visibility, module boundaries, and build variants, so it needs a compiler-backed reference set and complete preflight.

## Decision

This record defines the Go movement contract. Java's JDT LS-backed movement is
governed separately by ADR-0068. Gopls package rename relocates an entire package;
it is not a general file move or a merge into another existing package. Newer
upstream `MoveDeclaration` work is experimental and disabled by default; its
existence does not expand semedit's accepted Go subset or replace the validated
implementation described here. Delegating further moves requires capability
and refusal-contract validation, including callers, private dependencies, and
cycles, before advertisement.

Provide `semantic_move_construct` as a Go operation with `file` selecting the source declaration and optional `destination_file` selecting its destination. The operation returns `file`, `destination_file`, and the primary target `symbol` as copyable continuations. It preserves declaration comments and grouped declaration integrity, validates every affected file and type relationship before publication, and uses atomic writes for each file.

Same-file and same-package moves retain their existing declaration scope. In `semantic_batch`, cross-file and cross-package moves consume the current unpublished source revision. The bounded cross-package subset moves exported functions, standalone constants, aliases whose type expressions use only predeclared type names, uninitialized variables with predeclared structural type expressions, initialized predeclared scalar variables with proven constant values, and defined-type groups with their co-located exported direct methods between existing files in distinct packages of one Go module. ADR-0102 defines the shared operation and publication contract and records the proof boundary for each declaration shape. These paths resolve references through typed package information, update callers and imports throughout the module, and typecheck an in-memory overlay before writing. Each declaration shape retains its explicit refusal boundaries for test references, dependencies, method sets, and initialization effects. Unrelated generated or build-constrained files do not block a move when the complete source proof excludes them. Per-file atomic publication does not promise a transaction across multiple files; if a later publication step fails, the operation reports which earlier files were written.

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
* Cross-package Go moves are limited to the declaration shapes and proof boundaries defined by ADR-0102. They update every captured type-resolved reference and import or refuse before writing. Batched moves retain selected-path workspace checks and validate the current buffer state rather than rejecting directory changes. A batch receipt identifies requested source and destination paths; `final_diff` is authoritative about persisted writes.
* A failed buffered move contributes no output changes; completed earlier batch children retain the existing partial-application contract.
* Each file write follows ADR-0010; multi-file publication reports partial failures honestly.

## Consequences

* Callers can relocate source constructs with copyable file and symbol fields and receive diagnostic deltas.
* Scope remains explicit; source-local dependencies and hidden or ambiguous reference sets fail closed.
* Multiple per-file atomic writes can leave a reported partial result if a later write fails.
