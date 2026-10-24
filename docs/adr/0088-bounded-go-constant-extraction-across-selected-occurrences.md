<!-- WHY: Defines the occurrence-selection and proof contract for repeated Go constant extraction. -->
# ADR-0088: Bounded Go Constant Extraction Across Selected Occurrences

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Replace an explicit subset of equal Go constant expressions with one verified package constant. |
| description | Extend Go constant extraction to selected exact matches in one owner when every occurrence has the same resolved type and exact value. Bind occurrence indices to a source revision and verify every candidate reference resolves to the generated constant before one buffer apply. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

Go constant extraction currently promotes one exact compile-time expression and refuses when the discriminator matches multiple expressions. Replacing several repeated expressions can reduce duplication, but implicit replace-all would make a textual selector unsafe and would silently broaden the edit. The operation already has compiler type facts, a complete candidate-package check, and source-buffer publication, which can support a bounded explicit subset without moving evaluation because all selected expressions are compile-time constants.

## Decision

Extend `refactor_extract_value(kind: "constant")` for Go with optional `occurrence_indices`. The indices are zero-based, strictly increasing, and refer to exact `discriminator` matches in source order within the selected owner function or method. A request with indices must include the current `expected_revision` and cannot combine them with `construct_path`. Without indices, preserve the existing rule that the discriminator must match exactly one expression.

Plan the declaration and all replacements together. Accept the selected expressions only when each is supported by the existing constant syntax classifier and all have identical `go/types` types and exact constant values. Require distinct, non-overlapping expression ranges. Type-check the full candidate package, resolve the generated package constant, and prove that every replacement identifier refers to that exact constant with the selected type and value. Apply the candidate through one `sourcebuf.Buffer.Apply` call.

The index set may select a subset; omitted exact matches remain unchanged. Never infer “all” from an empty or omitted index list. Function-literal bodies are excluded from occurrence counting. Other language backends do not gain occurrence selection.

## Invariants

* Every selected occurrence belongs to one Go owner in one existing source file.
* `occurrence_indices` are revision-bound, sorted, unique, zero-based, and interpreted in source order over exact discriminator matches.
* Every selected expression preserves the same resolved type and exact constant value; unsupported syntax, comments, `iota` context, stale revisions, invalid indices, and name collisions refuse before mutation.
* Candidate type checking proves all generated references resolve to the newly declared package constant with the selected type and value.
* All selected replacements and the declaration are staged by one buffer apply; a refusal applies none of this operation's edits.
* Variable extraction and non-Go constant extraction keep their existing backend-specific contracts.

## Consequences

Go callers can promote selected repeated exact expressions with one declaration while retaining explicit control over which matches change. The source-order index list is compact but depends on the inspected revision; whole-file or cross-owner extraction, implicit replace-all, and broader constant syntax remain unsupported.
