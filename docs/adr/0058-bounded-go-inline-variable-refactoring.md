# ADR-0058: Bounded Go Inline Variable Refactoring

* **Status**: Superseded in part by [ADR-0094](0094-bounded-go-struct-literal-inline-value.md) and [ADR-0103](0103-bounded-go-computed-local-inline.md) for initializer eligibility
* **Date**: 2026-09-29

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | The original Go inline-value contract removes one literal local at direct same-block uses; ADR-0094 and ADR-0103 add bounded initializer cases, while callable inlining remains separate under ADR-0089. |
| description | Inline refactoring substitutes a declaration's evaluated value at its use sites and removes the declaration. A syntactically valid substitution can still change behavior when it repeats a side effect, observes a later assignment, crosses a shadowed binding, changes evaluation order, or changes dispatch. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

Inline refactoring substitutes a declaration's evaluated value at its use sites and removes the declaration. A syntactically valid substitution can still change behavior when it repeats a side effect, observes a later assignment, crosses a shadowed binding, changes evaluation order, or changes dispatch.

The initial Go operation needs an explicit contract that reports useful edits while refusing transformations whose equivalence it cannot establish.

## Decision

The restrictions below define Go inline-value. Go callable inlining has its separate contract in [ADR-0089](0089-bounded-go-callable-inlining.md). Java value and callable inlining are independently delegated to JDT LS under ADR-0068; these backend-specific proofs remain separate.

Register `refactor_inline_value` and the matching CLI command `refactor-inline-value` for Go. `symbol` targets the local variable or constant; `file` and `in_function` select the source file and containing function. The typed output receipt reports publication state, changed-file revisions, and scoped package-local read-set evidence. Transitive compiler inputs are revalidated internally before apply and are not claimed as complete receipt evidence. The CLI retains its diff and diagnostic details. The removed local name is historical evidence and is not returned as a live continuation; the current owner is returned as the live continuation. Batch child receipts omit live continuations; obtain fresh selectors with inspection or lookup after batch publication.

The implementation accepts one local `:=`, `var`, or `const` binding with a single literal initializer, provided the declaration has no explicit type. It replaces every direct call argument or direct return value in the same lexical block after the declaration and before any intervening control-flow boundary, then removes the declaration. Ordinary statements without references to the target may remain between the declaration and its uses; the inliner leaves them in place, so their order relative to the rewritten calls does not change. Because a local variable initialized by an untyped literal has a concrete default type, the operation inserts the matching conversion (`int`, `float64`, `complex128`, `rune`, or `string`) at each use. Untyped constants retain their literal form. It refuses expression initializers, explicitly typed declarations, local or package bindings, same-package declarations, and imports that could shadow the required default type conversion (unaliased imports are treated as uncertain), any additional identifier reference (including reassignment, address-taking, or shadowing), unsupported use shapes, uses inside or after a control-flow boundary, ambiguous declarations, and declaration comments it cannot preserve.

This decision covers local-value inlining only; it does not define callable-inlining behavior.

## Invariants

* The initializer is a basic literal or parenthesized basic literal; the engine never duplicates an evaluated expression.
* Every use must be a direct argument of a call expression or a direct return value in the same block after the declaration, with no control-flow boundary before a use. All such uses are rewritten together.
* An explicit declaration type causes refusal; inferred default types are retained by conversion at each use. Same-package sibling declarations are scanned, and unaliased imports are refused because their declared package name may differ from the import path basename.
* A function-wide identifier scan must find only the target binding and its accepted use sites; possible shadowing or reassignment causes refusal.
* The engine validates and formats the complete result before an atomic write; parse or safety failures leave source unchanged.
* The Go implementation has CLI txtar positive and negative coverage; each other advertised backend retains its own contract and comparable coverage.
* The input `symbol` identifies the local binding to remove. `file` and `in_function` constrain where the engine resolves it. The removed name is never a live follow-up target.

## Consequences

* **Positive**: Basic literal cleanup is available through one deterministic semantic request, with an explicit diff and diagnostic delta. A conservative contract avoids evaluation-order and binding changes.
* **Negative**: Common cases using computed expressions, explicit types, or references inside or after a control-flow boundary are rejected. Callable inlining follows the separate bounded contract in ADR-0089.
