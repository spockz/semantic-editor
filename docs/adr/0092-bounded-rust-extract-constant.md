<!-- WHY: Record the validated rust-analyzer contract for one bounded Rust local-to-constant extraction and its fail-closed arithmetic and binding proofs. -->
# ADR-0092: Bounded Rust Extract Constant

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Extract one bounded Rust integer initializer into a local constant while preserving its original local binding. |
| description | Delegate one positive decimal integer addition initializer to rust-analyzer's native extract-constant assist, prove its exact versioned edit and non-overflowing type, and rename only the generated constant and retained alias use. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | rust |

## Context

RQ-0074 identified a native rust-analyzer `extract_constant` assist, but its resolved action changes a local declaration into a constant and inserts a retained local alias before its first later use. The assist's identity includes a server-generated ordinal, its resolve response omits client-side action data, and its edit shape cannot be treated as a generic extraction. Rust also permits local type-namespace bindings to shadow primitive spellings, and a runtime integer addition that overflows may be invalid in a constant expression.

## Decision

Expose the assist through `refactor_extract_value` with `kind: "constant"`. Accept one exact, single-line local initializer consisting only of a `+` chain of positive decimal integer literals with identical explicit fixed-width integer suffixes. Prove the mathematical sum fits that integer type before requesting or staging the edit. Refuse `isize` and `usize` because the target width is not established.

Select the pinned rust-analyzer `extract_constant:RefactorExtract:1:` action by its exact observed identity, kind, and title. Resolve the original opaque action, permit the native resolve response to omit its data, and accept only the exact four-edit same-file versioned result: convert the original `let` into a typed `const`, preserve the original local as an alias inserted before its first subsequent use, then use rust-analyzer rename to change only the generated constant and its alias reference to the requested name. Refuse edits whose complete candidate differs from this proven shape.

Before contacting rust-analyzer, refuse possible primitive-type shadowing from generic parameters, type-namespace declarations, explicit imports, external crate aliases, glob imports, macros, or unresolved name expansion. Requested-name collision checks and the existing trusted-workspace and source-buffer publication boundaries remain in force. Unsupported cases remain unsupported; do not substitute `extract_static` or `promote_local_to_const`.

## Invariants

* The source selection exactly matches one single-line `let name = expression;` initializer.
* The expression is a positive decimal `+` chain with one identical fixed-width integer suffix; the complete sum fits that type.
* The native action has the observed identity and kind, and its versioned edit targets only the selected file.
* The validated edit produces one typed local constant and preserves the original local binding as an alias before its first later use.
* Rename changes only the generated constant and its alias reference; failed proofs leave source bytes unchanged.
* `isize`, `usize`, arithmetic overflow, shadowed primitive names, ambiguous actions, and other edit shapes refuse.

## Consequences

Rust now supports one native constant-extraction case through the existing value-extraction intent. It depends on the pinned rust-analyzer assist shape and refuses other server contracts. Multi-file edits, broader expressions, promotion of an existing local, static extraction, and module extraction remain separate investigations.
