<!-- WHY: Records the first typed composite-initializer extension to Go inline-value and its fail-closed proof boundary. -->
# ADR-0094: Bounded Go Struct Literal Inline Value

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Inline one keyed scalar-only Go struct literal into its sole immediate return after proving binding and type identity. |
| description | Extend Go inline-value beyond basic literals for a deliberately narrow, side-effect-free named struct initializer. Keep the original operation, selectors, receipt, and batch lifecycle. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

ADR-0058 limits Go inline-value to basic literals because substituting general expressions can duplicate effects, observe later mutation, alter evaluation order, or select a different binding. A keyed struct literal with only scalar fields and literal constant values has no such evaluation effects, but compilation of the edited file alone does not prove that the initializer and replacement have the same named type or refer to the same local binding.

## Decision

Supersede only ADR-0058's basic-literal initializer restriction with the following additional Go case. Keep the rest of ADR-0058's operation, receipt, overlay, publication, and refusal contract in force.

Inline one local inferred `:=` binding whose initializer is a keyed composite literal of a same-package, non-generic named struct. Every declared struct field, including omitted fields, must have a scalar Go basic type; every supplied field value must be a basic literal or the predeclared boolean constant `true` or `false`. Require exactly one resolved reference to the binding, used as the sole expression of the immediately following direct return statement in the owner's body. The owner must have exactly one unnamed result. Keep the original composite literal source and its type spelling at the return site.

Resolve the binding, named struct, field set, initializer type, and return type through compiler type information loaded from the current source-buffer overlay. Prove reference identity with `types.Info`, reject all other references or writes, verify that the candidate return expression resolves to the original binding type, and revalidate the compiler input read set before publication. Do not infer equivalence from spelling or successful candidate compilation alone.

## Invariants

* Do not broaden the existing basic-literal path or change the canonical operation inputs, receipt shape, or batch lifecycle.
* Reject explicit local types, positional literals, generic structs, imported or unresolved struct types, non-basic fields, nested/reference-bearing fields, nonliteral field values, multiple or indirect uses, non-immediate/non-direct returns, named or multiple owner results, and comments or directives in the moved or removed region.
* The inline-value transformation preserves the exact initializer type spelling and all literal source bytes, including numeric token spelling. It does not format the candidate, add imports, or synthesize conversions. Standalone output retains those bytes; buffered batches apply the repository's normal Go formatting during finalization and may normalize token spelling.
* Use the overlay-aware package loader and compiler-input capture already used by Go refactoring planners. Candidate validation must run against that same overlay, and publication must retain the existing read-set and atomic-output guarantees.
* CLI txtars must prove a successful interface-return case with its dynamic named type, refusal cases with byte-identical sources, and standalone plus buffered-batch behavior. The buffered-batch case records the formatter's final normalization separately from the transformation's source-preservation behavior.

## Consequences

Go inline-value can remove one safe, short-lived named scalar struct temporary while preserving the value and dynamic type returned. The narrow field and use restrictions avoid alias identity, side effects, control-flow, and evaluation-order analysis. Slices, maps, pointers, nested aggregates, variable reads, calls, conversions, multiple uses, and broader return shapes remain unsupported until separately proven.
