<!-- WHY: Records the first-release semantic proof boundaries for three Go signature refactorings. -->
# ADR-0064: Bounded Go Signature Refactoring

* **Status**: Accepted
* **Date**: 2026-10-03

## Context

Changing a function signature also changes its callers. Safe edits need exact
typed bindings and a complete view of every call that must change. Broad
workspace rewriting would promise more than the current proof can establish.

## Decision

The bounds below apply to Go. Java signature and parameter refactorings use
JDT LS under ADR-0068 and retain their backend-specific accepted edit shapes.

Provide `semantic_change_signature`, `semantic_introduce_parameter`, and
`semantic_introduce_parameter_object` for private, non-generic, non-variadic
Go free functions. Each analyzes the complete active call set and test
variants in a snapshot/overlay, then typechecks the candidate overlay before
applying edits.

`semantic_change_signature` appends one parameter and one closed argument,
preserving existing parameter and call-argument order.
`semantic_introduce_parameter` replaces a selected compile-time constant
expression in the function body with a new appended parameter; callers pass
the same typed constant value.
`semantic_introduce_parameter_object` replaces all parameters with one
same-file private struct whose fields map every parameter using its exact
type; generated literals preserve the original argument evaluation order.

The operations refuse cases outside their proof, including incomplete or
ambiguous call sets, unsupported function or type shapes, unsafe argument
evaluation, and candidate type errors. Only finalized standalone results may
include `function_target: {file, symbol, expected_revision}`, classified from
the final source as the exact original private free function. Generic receipt
continuations route to inspection only; inspection needs file and symbol and
does not consume a revision. Buffered `applied_working` batch children expose
no live target.

## Invariants

* Only private, non-generic, non-variadic free functions are eligible.
* Every changed call is accounted for across the complete active call set
  and test variants visible in the snapshot/overlay.
* Parameter and argument ordering, exact type identity, and original argument
  evaluation order are preserved as specified by the selected operation.
* Candidate package overlays typecheck before mutation; unsupported or
  unproven cases fail closed.
* A live function target is emitted only for a finalized standalone result
  after final-source classification; batch working results emit none.

## Consequences

The tools support common signature changes without claiming public API,
method, generic, variadic, or incomplete-workspace coverage. A finalized
target can be copied to any of the three signature tools or to body
replacement. Generic receipt continuations for changed identities remain
inspect-only and need just file and symbol; they do not consume a revision.
