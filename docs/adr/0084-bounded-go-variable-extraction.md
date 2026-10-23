<!-- WHY: Keep the Go value-extraction operation's accepted source forms and shared semantic proof in one decision. -->
# ADR-0084: Bounded Go Variable Extraction

* **Status**: Accepted
* **Date**: 2026-10-21

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Extract typed Go expressions into locals at supported call, assignment, declaration, return, and condition sites, including bounded repeated-expression ranges, while preserving evaluation and type. |
| description | Go value extraction uses one operation across statement forms and explicit repeated-site ranges. Each form keeps generated locals at proven evaluation points and relies on compiler-backed identity, stability, type, and candidate checks; unsupported or ambiguous shapes refuse without writing. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

`refactor_extract_value(kind=variable)` initially handled one direct-call argument. Assignment RHSs, declaration initializers, and return expressions use the same typed expression extraction mechanism but have different local placement and control-flow boundaries. Recording each statement form as a separate decision obscures the shared contract and creates redundant records as supported sites expand.

Moving an expression can change evaluation order, scope, type inference, comments, binding identity, named-result assignment, or defer behavior. The operation must retain a narrow, compiler-validated contract at each supported site.

## Decision

Keep one Go variable-extraction contract for the following forms:

* **Call argument:** Extract one exact argument of a direct, unshadowed same-file package-function call expression statement. Every sibling argument must be a compiler-proven constant, resolved predeclared `nil`, or a bare scalar owner input. Generic, variadic, method, indirect, and function-value callees refuse.
* **Repeated arguments in one call:** One exact discriminator may select every matching argument in a selected direct call. Each occurrence must have identical resolved type and use a bounded scalar expression over stable scalar owner inputs and constants. Reject any selected input whose address can escape explicitly or through pointer-receiver selection, or which a nested closure captures. Every nonmatching sibling must satisfy the call-argument whitelist above. Keep panics and defers in the original owner frame.
* **Repeated arguments across calls:** An explicit inclusive `construct_path` to `end_construct_path` and matching `expected_revision` may select consecutive direct calls in one owner block. Every call must contain a matching argument. Replace every match with one local inserted immediately before the first call. Apply the same per-occurrence type, stable-input, escape/capture, and sibling-argument proofs as repeated arguments within one call. Without an endpoint, retain single-call selection behavior; do not infer a range.
* **Assignment RHS:** Extract the whole RHS of one simple `=` assignment with one named identifier target. The target's left-hand evaluation is therefore unchanged. The assignment may be in the owner body or any nested statement block.
* **Short-declaration RHS:** Extract the whole RHS of one `:=` declaration with exactly one named target and one RHS.
* **Local `var` initializer:** Extract the whole RHS of one local declaration containing exactly one `ValueSpec`, one named target, and one RHS. The original declaration, target, and explicit type remain in place.
* **Return expression:** Extract the whole expression of a direct or nested return owned by the selected function or method. The owner must have exactly one explicit result field, unnamed or named with one nonblank identifier. Keep the original return so it assigns a named result before deferred functions run.
* **Repeated assignment range:** An explicit revision-bound inclusive construct-path range may select two or more consecutive direct assignments in one block. Each statement must be a one-target, one-RHS simple `=` assignment whose RHS exactly matches the discriminator. The targets must be distinct existing owner locals and cannot be owner inputs or inputs used by the expression. Require identical resolved scalar expression types and stable, non-address-escaped, non-captured owner inputs. Insert one local before the range and replace every selected RHS; mixed statements, changed targets, or unsupported expressions refuse.
* **Complete `if` condition:** Extract the entire condition of one `if` with no init clause. Insert the local immediately before the `if` in the same containing block. Require a pure expression with a resolved typed input, preserve its exact resolved type, and use the full-file revision selector when multiple candidates match. Nested conditions remain in their original branch or loop body.

For all forms, require one exact whole expression with resolved type information and at least one supported typed input. The original expression and generated local must retain identical resolved Go type identity. The expression must satisfy the operation's side-effect restrictions; context-dependent defaulting, unsupported expression shapes, and unsafe changes to sibling argument order refuse. Keep each generated local immediately before its source statement or selected range in the same lexical block; never hoist it across a branch, loop iteration, switch clause, select clause, or nested block. Do not traverse function-literal bodies.

Use the existing `construct_path` to select an assignment, declaration, or return when multiple exact matches exist, and require its matching `expected_revision`. A unique exact match needs no path. Keep exact discriminator matching and reject stale or ambiguous selections. Validate the complete candidate package and captured compiler inputs through the source-buffer overlay before apply.

Refuse multiple or blank targets, multi-value declarations, compound or tuple assignments, non-identifier assignment targets, naked returns, multiple result fields, blank or multiply named result fields, tuple-valued or partial expressions, closure returns, comments in unsupported ownership positions, name collisions, untyped-only expressions, unsupported calls or side effects, and failed candidate proofs without changing source.

## Invariants

* The extracted local executes once at the original expression evaluation point and remains in the same lexical control-flow block.
* Existing assignment, declaration, return, result assignment, and deferred execution retain their original order and binding behavior.
* Candidate validation proves exact resolved expression type identity and validates all captured compiler inputs before publication.
* A path selector remains revision-bound; the operation does not infer a range from missing or stale evidence.
* Standalone and buffered-batch CLI txtars cover every advertised source form, repeated-site range, runtime behavior, and unchanged refusal boundaries.
* This Go-specific contract does not change Java, Rust, Scala, or other backend extraction behavior.

## Consequences

Go variable extraction has one documented decision for its supported expression sites, with per-site restrictions stated together. Broader expressions, control-flow movement, unsupported result shapes, and additional call-range selection remain outside this contract until their evaluation and flow proofs are established.
