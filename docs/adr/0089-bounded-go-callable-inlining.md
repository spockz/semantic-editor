<!-- WHY: Records the distinct resolution and receiver proofs for safely inlining Go free functions and concrete methods. -->
# ADR-0089: Bounded Go Callable Inlining

* **Status**: Accepted
* **Date**: 2026-10-21

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Inline one selected direct Go free-function or concrete-method call with a literal-return body under bounded type and receiver proofs. |
| description | Go callable inlining may substitute only one zero-argument call whose same-package declaration has one direct literal return and one result. A method declaration may be in a separate file of the same active package; it requires a concrete receiver, an unnamed result, and a direct caller-parameter receiver with no indirection. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

[ADR-0058](0058-bounded-go-inline-variable-refactoring.md) defined literal local-variable inlining and deferred call inlining because substitution can change evaluation count, dispatch, scope, and receiver behavior. The implementation now supports a narrow free-function contract plus a concrete receiver-method case whose selector and caller receiver can be proven through Go type information. Java and Rust keep their backend-specific callable-inlining contracts.

## Decision

For Go `refactor_inline_callable`, preserve the existing direct zero-argument free-function form and accept an exact receiver-qualified method target such as `Counter.Value`. Both forms require one uniquely resolved selected call, a non-generic declaration with no parameters and one result, and a body containing exactly one direct return of a basic literal expression. Keep the callee declaration in place and validate the complete candidate package before applying the edit.

For methods, require the declaration and caller to be in the same active Go package, but they may be in different files. Resolve the method on a concrete, non-generic receiver type; refuse promoted or embedded method selection, interface dispatch, pointer indirection, method values, and receiver expressions other than a direct identifier bound to a caller parameter. The caller body must consist only of its one direct return call. The method must have one unnamed result and a body. Reject cross-package declarations, named method results, bodyless declarations, ambiguous matches, and unsupported receiver forms without mutation.

The free-function path continues to require a direct identifier call and retains its existing same-package, resolved-identity, literal-body, and candidate-validation checks. This decision does not extend Java, Rust, Scala, or other backend capabilities.

## Invariants

* The selected call resolves to exactly one declaration through compiler object identity; source spelling alone never establishes the target.
* The callee takes no arguments, has one result, and returns one `ast.BasicLit` expression directly; conversions and computed expressions are refused.
* Free functions use a direct identifier call. Methods use a direct selected method call on a caller parameter with no implicit receiver indirection.
* Method declaration and caller belong to the same active package; cross-file declarations are allowed. The method is concrete, directly selected, non-generic, body-bearing, and has one unnamed result.
* Method inlining refuses interface dispatch, method values, promoted or embedded selections, local or package-variable receivers, receiver side effects, and any caller body with additional statements.
* The callee declaration remains unchanged. Type and source validation run before atomic application; unsupported or stale targets leave source unchanged.
* Other languages retain their own backend capability and edit-capture contracts under ADR-0068 and RQ-0073/RQ-0074.

## Consequences

Go can inline a small concrete method when the receiver binding and dispatch target are explicit, including when the method is declared in another active package file. The caller-parameter restriction leaves local-receiver and polymorphic cases unsupported until separate proofs establish equivalent behavior.
