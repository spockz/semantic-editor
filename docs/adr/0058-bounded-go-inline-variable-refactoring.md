# ADR-0058: Bounded Go Inline Variable Refactoring

* **Status**: Accepted
* **Date**: 2026-09-29

## Context

Inline refactoring substitutes a declaration's evaluated value at its use sites and removes the declaration. A syntactically valid substitution can still change behavior when it repeats a side effect, observes a later assignment, crosses a shadowed binding, changes evaluation order, or changes dispatch.

The initial Go operation needs an explicit contract that reports useful edits while refusing transformations whose equivalence it cannot establish.

## Decision

The restrictions below define the Go backend. Java variable and call inlining
are independently delegated to JDT LS under ADR-0068; the Go call-inlining
exclusion does not disable those Java operations.

Register `semantic_inline_variable` and the matching CLI command `inline-variable` for Go. `symbol` targets the local variable or constant; `file` and `in_function` select the source file and containing function. The typed output receipt reports publication state, changed-file revisions, and scoped package-local read-set evidence. Transitive compiler inputs are revalidated internally before apply and are not claimed as complete receipt evidence. The CLI retains its diff and diagnostic details. The removed local name is historical evidence and is not returned as a live continuation; the current owner is returned as the live continuation. Batch child receipts omit live continuations; obtain fresh selectors with inspection or lookup after batch publication.

The implementation accepts one local `:=`, `var`, or `const` binding with a single literal initializer, provided the declaration has no explicit type. It replaces every direct call argument or direct return value in the immediately following statement in the same lexical block, then removes the declaration. Because a local variable initialized by an untyped literal has a concrete default type, the operation inserts the matching conversion (`int`, `float64`, `complex128`, `rune`, or `string`) at each use. Untyped constants retain their literal form. It refuses expression initializers, explicitly typed declarations, local or package bindings, same-package declarations, and imports that could shadow the required default type conversion (unaliased imports are treated as uncertain), any additional identifier reference (including reassignment, address-taking, or shadowing), unsupported use shapes, intervening statements, ambiguous declarations, and declaration comments it cannot preserve.

Call inlining remains unsupported. It needs call resolution and argument-to-parameter substitution that preserve evaluation count, order, dispatch, defer behavior, and scope. A gopls code action may be used later only after its behavior meets that contract and every returned edit passes post-edit validation.

## Invariants

* The initializer is a basic literal or parenthesized basic literal; the engine never duplicates an evaluated expression.
* Every use must be a direct argument of one call expression or a direct return value in the next statement in the same block. All such uses in that statement are rewritten together.
* An explicit declaration type causes refusal; inferred default types are retained by conversion at each use. Same-package sibling declarations are scanned, and unaliased imports are refused because their declared package name may differ from the import path basename.
* A function-wide identifier scan must find only the target binding and its accepted use sites; possible shadowing or reassignment causes refusal.
* The engine validates and formats the complete result before an atomic write; parse or safety failures leave source unchanged.
* The Go implementation has CLI txtar positive and negative coverage; each other advertised backend retains its own contract and comparable coverage.
* The input `symbol` identifies the local binding to remove. `file` and `in_function` constrain where the engine resolves it. The removed name is never a live follow-up target.

## Consequences

* **Positive**: Basic literal cleanup is available through one deterministic semantic request, with an explicit diff and diagnostic delta. A conservative contract avoids evaluation-order and binding changes.
* **Negative**: Common cases using computed expressions, explicit types, references after intervening statements, or function-call inlining require future work and are rejected today.
