# ADR-0056: Bounded Go Call-Statement Extraction

* **Status**: Accepted
* **Date**: 2026-09-29

## Context

Extraction can move code into a named helper and rewrite its original use site.
Correctness depends on data flow, evaluation order, control flow, name resolution,
and comment ownership. ADR-0053 separates extraction from insertion and movement
but leaves its detailed schema and guarantees to a later decision.

The initial implementation needs a narrow Go contract that can prove its source
transformation without broad type inference or changing execution scope.

## Decision

Expose `semantic_extract_function` and the `extract-function` CLI command for Go.
The primary target is the new helper name in `symbol`. The existing owner is the
`in_function` selector. `file` identifies the workspace-relative Go source file.
`kind` is `function` or `method`. `start_line` and `end_line` select an inclusive
range by exact start and end lines of direct statements in the owner's body.

Only a contiguous region of direct call expression statements, optionally
preceded by one single-variable `:=` declaration, is supported. The declaration
must produce one local used after the selected region. That local becomes the
helper's sole return value. Its type is inferred only for basic literals, a
boolean literal, a named owner parameter, or a supported unary/parenthesized
form of those expressions. The helper receives each referenced named owner
parameter and, when required, the receiver.
Function extraction passes a referenced receiver as a normal argument. Method
extraction preserves the receiver on the new method. The selected calls retain
their order and are replaced by one helper invocation. The helper is inserted
immediately after the owner declaration.

The engine refuses a request before writing when it cannot establish the
following conditions:

* Each boundary line identifies exactly one direct statement and the inclusive
  range selects adjacent statements.
* Selected statements are calls, optionally with one leading single-variable
  short declaration whose one inferred result is live after the region.
  Other assignments/declarations, multiple outputs, and unsupported result
  types are refused. Returns, branches, loops, labels, and other statement
  forms are unsupported.
* Free identifiers resolve to named owner parameters, the receiver, same-file
  package declarations, imports, or predeclared names. Owner-local declarations
  anywhere in the function cannot shadow a selected identifier.
* A forwarded parameter or receiver cannot be address-taken or closure-captured
  anywhere in the owner body. Selected direct `recover()` calls are refused.
  Selector calls rooted at a forwarded value are refused because syntax alone
  cannot prove whether implicit pointer-receiver dispatch mutates a copied
  value.
* The selected region cannot take the address of the local returned by the
  helper, because the helper's local has a different address from the caller's
  result variable.
* The owner is non-generic, has no variadic parameters, and has no named results.
* The selected region contains no function literals or composite literals, and
  the owner body contains no comments. This avoids changing closure scope,
  misclassifying map-literal key identifiers, or guessing comment ownership.
* The generated helper name is a valid simple identifier and does not collide
  with a package declaration or same-receiver method.

The output receipt returns `symbol`, `file`, and `in_function` with those exact
names for follow-up calls. The line range is omitted because its coordinates no
longer identify the moved statements after a successful mutation. Receipts carry
standard diagnostic status and delta fields. Writes follow ADR-0010's atomic
publication rules.

## Invariants

* `symbol` names the new helper; `in_function` only selects the existing owner.
* `start_line` and `end_line` are positive, inclusive, one-based line numbers,
  not byte offsets. The selected range must match AST statement boundaries.
* The operation preserves call order and argument evaluation order.
* Unsupported or ambiguous selections fail before mutation.
* `semantic_extract_function` is advertised only for Go and has CLI txtar
  coverage for its observable success and refusal behavior.
* Broader statement extraction, general return-value inference, expression
  extraction, other languages, and gopls-driven extraction remain unadvertised
  until their own safety guarantees are established.

## Consequences

The operation reduces a common manual helper extraction to one semantic call
for call regions and one returned local with a deliberately small type-inference
boundary. Its supported region is intentionally limited; many ordinary
refactorings that use other local data flow or control flow will receive an
explicit refusal. The shared schema and registry expose the operation through
both CLI and MCP, and its receipt follows ADR-0055 continuation naming.
