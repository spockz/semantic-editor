<!-- WHY: Defines the typed identity and evaluation-order boundary for grouping keyed initializers during field extraction. -->
# ADR-0094: Bounded Go Keyed-Literal Type Extraction

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Group a contiguous source-order run of selected keyed Go fields into the extracted same-file composition value. |
| description | Field extraction must update ordinary keyed initialization without moving effects across unselected initializers, changing field bindings, or capturing a local type name. Preserve original value-expression bytes and zero initialization while refusing unsupported literal edits before buffered application. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

[ADR-0063](0063-bounded-go-field-only-type-extraction.md) extracts a contiguous declaration-ordered run of private fields, but refuses every selected keyed initializer. This excludes common construction sites even when the initializer can be grouped without changing Go-specified operand ordering.

This decision supersedes ADR-0063 only for the keyed-use refusal described below. Its private owner, declaration-order, layout, source-set, method-ownership, and publication constraints remain in force. Delegate extraction retains its independent keyed-use refusal under ADR-0080.

## Decision

Extend Go `refactor_extract_type` to rewrite keyed literals of the explicit selected owner type in the selected file. The literal's explicitly initialized selected fields must occupy one contiguous run of elements in source order. Replace that run with `field: newType{original selected key/value source}`, preserving its internal order and value-expression bytes. Selected fields omitted by the original literal remain zero-valued; literals containing no selected keys require no rewrite.

Resolve field keys through compiler objects and admit uses by physical filename and byte offset. Preserve the existing contiguous declaration-order selection independently of literal-key order. Verify that the generated type name is not shadowed at each changed literal, then typecheck the complete candidate overlay before one buffered application.

Refuse interleaved selected and unselected keys, implicit or aliased owner-type syntax, comments inside changed owner literals, source-line directives in a file containing a changed literal, overlapping nested or selector edits, positional literals, selected uses in another file, stale revisions, and every existing unsupported owner or field shape.

## Invariants

* Each selected initializer is identified by its resolved field object, never by spelling alone. Unrelated same-named fields remain unchanged.
* The changed run preserves the original selected key/value source bytes and evaluates each selected expression once. It preserves Go-specified ordering of function calls, method calls, receives, and binary logical operations. Go leaves ordinary operand order, including variable reads and indexing relative to adjacent calls, unspecified; the transformation does not promise a compiler-specific schedule. See the [Go specification's order-of-evaluation rules](https://go.dev/ref/spec#Order_of_evaluation).
* Omitted selected fields retain Go's normal zero initialization.
* Physical file coordinates establish source identity; adjusted `//line` filenames never authorize an edit.
* Visible local declarations, imports, or predeclared names cannot capture the newly introduced type reference.
* The common source-edit planner rejects overlapping ranges before application. An unsupported literal contributes no partial working edit.
* The owner and extracted type remain private and in the selected existing file; methods remain attached to the owner. Delegate extraction keeps its prior literal policy.
* Standalone continuations and buffered receipt semantics use the existing structural-plan lifecycle.

## Consequences

Field-only extraction supports common full and partial keyed construction sites without a new operation, parameter, or publication path. Source-order contiguity remains distinct from declaration-order contiguity: both must hold for their respective proofs. Broader literal nesting, cross-file uses, layout-sensitive consumers, and delegate literal migration remain open.

Standalone and batch CLI txtars cover call count and Go-specified function-call ordering, pointer literals, omitted fields, unrelated field identities, overlay composition, and byte-preserving refusals. The integrated `make check` passed on 2026-10-22. Sol 6.1's first review identified an overbroad evaluation-order statement; the ADR and tool contract now limit the guarantee to Go-specified ordering and explicitly disclaim a compiler-specific schedule. Follow-up review confirmed the narrowed contract and fixtures; no actionable findings remain.
