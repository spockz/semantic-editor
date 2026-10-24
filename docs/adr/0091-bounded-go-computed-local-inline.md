<!-- WHY: Keep the shared computed-initializer proof and its accepted immediate Go use sites in one decision. -->
# ADR-0091: Bounded Go Computed-Value Inline Destinations

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Inline one pure computed Go local at its sole immediately following call, assignment, declaration, return, `if`, switch, `defer`, or `go` use. |
| description | Remove an inferred scalar local only when compiler-resolved analysis proves its initializer inputs, sole use, evaluation point, destination binding, and type remain unchanged in the candidate. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

The Go `refactor_inline_value` operation originally supported basic literal and keyed scalar-struct initializers. A computed scalar initializer can also be inlined safely when it has one adjacent use and candidate analysis proves its source bindings, value, and type. Adding each destination as a separate ADR duplicated this proof and made the operation contract difficult to discover.

## Decision

Extend the computed-value form of `refactor_inline_value` to one inferred scalar `:=` local whose initializer consists only of scalar literals, resolved scalar owner inputs or constants, parentheses, and supported pure unary or binary operators. Require exactly one compiler-resolved use and require the destination statement to immediately follow the local in the same owner block.

Accept the local identifier as the complete expression at exactly one of these destinations:

* one direct return expression;
* the sole argument of one direct call to a declared function or builtin;
* the RHS of one simple `=` assignment to a distinct existing identifier;
* the RHS of one `:=` declaration with one target and one value that creates a distinct binding;
* the initializer of one ungrouped local `var` declaration with one name and one value;
* the complete condition of one `if` statement with no init clause;
* the complete tag of one expression switch with no init clause;
* the sole argument of one direct `defer` call; or
* the sole argument of one direct `go` call.

For calls, `defer`, and `go`, reject methods, selectors, indirect targets, function-valued targets, multiple arguments, and variadic expansion. For the `if` condition, permit Go's untyped Boolean defaulting only when the complete candidate condition remains Boolean. Keep expression-switch tags distinct from type switches and reject partial tag or condition occurrences.

The candidate must preserve the original resolved callee and destination binding, initializer input bindings and types, constant values, expression type, and evaluation timing. In particular, defer arguments still evaluate at registration, and `go` arguments still evaluate synchronously before the goroutine starts. Reuse the existing source-buffer, overlay, receipt, and publication lifecycle; do not add selectors, parameters, aliases, or a second implementation.

## Invariants

* The computed local and its one accepted use are adjacent direct statements in the same owner block; no statement, comment, or control-flow boundary intervenes.
* The initializer is side-effect-free and scalar, and its exact binding identities, types, and constant values remain unchanged.
* Candidate analysis proves the sole use and preserves the destination's resolved type, binding identity, and direct callee identity where applicable.
* Reject additional uses, comments or directives in the edited region, shadowing or reassignment, changed bindings or types, grouped or multi-value declarations, partial expressions, init clauses, type switches, unsupported initializers, and failed candidate proofs before writing.
* Standalone and buffered-batch CLI txtars cover each accepted destination, exact results, runtime semantics, and byte-preserving refusals.
* Basic literal inlining, keyed scalar-struct inlining, and callable inlining retain their separate contracts.

## Consequences

Computed scalar locals can be removed consistently across the listed immediate use sites under one shared proof contract. Uses across statements, branches, loops, or multiple sites remain unsupported, as do effectful initializers, non-scalar expressions, and destinations whose native semantics cannot be preserved.
