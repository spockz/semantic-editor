<!-- WHY: Records the accepted first-release proof boundary for extracting selected private Go fields into a composed type. -->
# ADR-0063: Bounded Go Field-Only Type Extraction

* **Status**: Accepted
* **Date**: 2026-10-03

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Extract a contiguous declaration-ordered run of direct private fields into a same-file composed type, rewrite typed selectors, preserve equality evaluation order, and preserve method ownership. |
| description | Moving fields into a composed type changes storage layout and every affected selector. Reconstructing those edits by text can confuse same-named fields, while batch children must analyze the same current overlay as earlier edits. The first useful operation should expose only cases for which the engine can identify the selected fields and every affected use precisely. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

Moving fields into a composed type changes storage layout and every affected selector. Reconstructing those edits by text can confuse same-named fields, while batch children must analyze the same current overlay as earlier edits. The first useful operation should expose only cases for which the engine can identify the selected fields and every affected use precisely.

## Decision

Provide `semantic_extract_type` / `extract-type` as a Go-only field extraction. The caller selects exact direct private fields on one private, non-generic struct in an existing source file. The caller chooses fresh private names for the generated type and composition field. The generated type is placed in the same file and package; typed direct selectors in that file are rewritten to go through the composition field. The complete candidate package overlay is typechecked before one edit is applied.

This operation keeps methods on the original type. It does not move methods, synthesize delegates, or promise field-layout compatibility for reflection, serialization, or other consumers.

## Invariants

**Scoped supersession by [ADR-0094](0094-bounded-go-keyed-literal-type-extraction.md):** Field-only extraction now permits a bounded contiguous run of explicitly keyed same-file owner initializers. The historical keyed-use refusal below remains in force for delegate extraction and literal shapes outside ADR-0087.

* Member inputs identify exact `{file, symbol}` fields from the selected owner. Source declarations used for edits are rebound to the parsed selected-file syntax, and selector rewrites require the exact selected Go field object.
* The proof is fail-closed for keyed or non-selector field uses, address-taking including parenthesized selectors, cross-file or external consumers, exported API exposure, generated or build-variant files, embedded or grouped fields, selected fields with tags or comments, and owner declarations with same-line trailing comments.
* Packages that directly import `reflect` or `unsafe` are outside this layout proof and are refused. This check does not establish the absence of reflective behavior in arbitrary dependencies or external consumers.
* The selected owner, new type, and composition field are private. Generic owners, grouped declarations, and new-file destinations are unsupported.
* Selected fields must form one contiguous run in the original declaration order, and callers must list them in that order. Struct equality short-circuits through fields in source order; grouping fields around an unselected interface field can change whether comparison returns or panics.
* Batch children resolve against the current source overlay. Buffered receipts do not expose a live continuation; finalized receipts identify the generated type in the edited file.
* Methods remain attached to the original type, so the operation does not change its declared method set. Candidate package type checking is required but does not broaden the source-use or layout proof.

## Consequences

The bounded operation supports extracting selected state while preserving method ownership. Its CLI and MCP contracts are covered by standalone success/refusal cases, a same-offset cross-file identity regression case, comment and parenthesized-address refusals, ordered-contiguous selection checks, and a two-child batch that consumes a generated type from the working overlay. Method extraction, delegates, exported or generic types, broader literal handling, and reflection-sensitive compatibility remain separate open work in RQ-0060.
