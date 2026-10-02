# ADR-0059: Bounded Go Extract Variable

* **Status**: Accepted
* **Date**: 2026-09-29

## Context

Extracting an expression into a local can change evaluation order, scope, type inference, comment ownership, or the binding of later identifiers. The operation needs a narrow contract that can prove the rewrite while retaining a useful common case.

## Decision

Expose `semantic_extract_variable` and the `extract-variable` CLI command for Go. `symbol` names the new local, `in_function` selects its function or method owner, `file` selects the exact source file, and `discriminator` is the exact source text of the whole selected expression. The only supported expression is the exact first argument of a direct call expression statement in the owner's direct body; later arguments remain in place and retain their order. `construct_path` selects an occurrence when the same exact expression appears more than once; paths are zero-based direct statement indexes and require `expected_revision`, the matching full selected-file SHA-256 revision.

The selected expression must be side-effect-free and have resolved type information tied to an owner parameter or receiver, so assigning it to a local preserves contextual and defined numeric types. Calls, receives, address-taking, composite literals, comments anywhere in the owner body, returns, nested expression positions, receiver calls, constant-only expressions, and shifts with an untyped left operand are unsupported. The direct call target must resolve to an unshadowed same-file package function. The new local name must not occur in the owner body or collide with a package declaration or explicit import name. The local declaration is inserted immediately before the call, and only the selected first-argument bytes are replaced. The full candidate package is typechecked before the source buffer applies the edit.

The transformation validates formatted Go syntax and publishes through ADR-0010 atomic writes. The typed output receipt reports the actual publication state, changed-file revisions, and consulted read set. The CLI retains the diff and diagnostic details. The current common receipt schema does not project the new local and owner as live continuations. The operation is available only for Go.

## Invariants

* The selected expression is copied verbatim using `discriminator`; whitespace is significant.
* Ambiguous selections fail before writing and report valid `construct_path` values.
* Unsupported and unsafe expression forms fail before writing.
* The rewrite preserves the selected first argument's place in call evaluation and leaves the receiver and later arguments in their original order.
* `semantic_extract_variable` has CLI txtar coverage for a successful resulting workspace state and an unchanged ambiguity refusal.
* Other expression contexts, constant extraction, and non-Go backends remain unadvertised.

## Consequences

The operation covers a focused extraction useful for naming an argument computation before a simple call. It uses resolved Go types and a complete candidate-package typecheck while retaining the call statement's evaluation boundary; module-wide usage discovery remains outside this bounded operation. The restrictions exclude many otherwise valid Go extractions and return-style expressions until their type and evaluation guarantees can be proved.
