<!-- Records the semantic boundary for moving selected Go fields and one method into a same-file delegate. -->
# ADR-0080: Bounded Go Delegate Extraction

* **Status**: Accepted
* **Date**: 2026-10-20

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Define the `refactor_extract_delegate` operation to move a contiguous private field run and one direct pointer method into a same-file delegate while keeping the owner's original method wrapper. Validate receiver uses, recover behavior, nil receiver forwarding, source membership, and the full overlay before one buffered apply. |
| description | Field-only extraction leaves behavior on the original Go type. Moving a method changes how nil receivers and `recover` behave unless the owner retains a forwarding method and receiver uses are proved. The first delegate operation therefore needs a distinct wrapper policy from `semantic_extract_type`. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

Field-only extraction leaves behavior on the original Go type. Moving a method changes how nil receivers and `recover` behave unless the owner retains a forwarding method and receiver uses are proved. The first delegate operation therefore needs a distinct wrapper policy from `semantic_extract_type`.

## Decision

`refactor_extract_delegate` moves one contiguous, declaration-ordered run of direct private fields and one direct pointer-receiver method from a private, non-generic Go struct in one existing source file. It creates one private delegate type in that file, composes it on the owner, rewrites typed direct field selectors throughout the selected file, and moves the method body to a pointer receiver on the delegate. The original method remains on the owner with its exact signature and forwards to the delegate.

The wrapper initializes a nullable delegate pointer and assigns `&owner.field` only when the owner is nonnil. This preserves method bodies that return before touching selected fields on a nil receiver. It also preserves the owner's interface satisfaction, method values, and method expressions because the method identity remains on the owner.

The moved method receiver may appear only as the immediate base of a direct selector for one of the selected fields. The operation refuses receiver escape, reassignment, comparison, address-taking, other field or method access, and resolved builtin `recover` uses, including nested occurrences. It accepts named nonblank parameters, nonvariadic methods, and unnamed results. It refuses method comments, generics, named results, grouped or unnamed parameters, generated sources, inactive build variants, unsafe or reflection package use, selected-field uses outside the source file, keyed literals, address-taking, tags, comments, embedded fields, and exported API exposure.

The complete source overlay and membership preimages are validated before one buffered apply. Standalone and batch requests use the same plan and receipt boundary. The generated type continuation is resolved from final source. Method relocation adds a call frame, so stack inspection and call-stack identity are outside the preservation claim.

## Invariants

* Selected fields form one contiguous run in declaration order; this preserves recursive struct equality comparison order.
* The owner retains the original method signature and method-set membership.
* A nil owner is passed as a nil delegate; field dereference occurs only where the moved body would have accessed a selected field.
* Receiver identifiers are resolved by compiler object identity; same-spelled shadowed locals are not rewritten.
* The moved receiver has no uses beyond direct selected-field selectors, and builtin `recover` does not occur in the moved body.
* All selectors, method identity, source membership, candidate output, and overlay types are checked before the edit is applied.
* Refusal contributes no partial source edits, including when called as a buffered batch child.

## Consequences

The operation supports a useful state-and-behavior extraction while preserving the owner API and ordinary nil-field fault timing. It changes layout, reflection/serialization presentation, pointer identity, and stack traces; those behaviors remain outside this bounded contract. More signatures, cross-file movement, external consumers, and additional source shapes require independent evidence and comparable CLI coverage.
