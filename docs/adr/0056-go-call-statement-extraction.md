<!-- WHY: Defines the shared proof boundary for every supported Go callable extraction form. -->
# ADR-0056: Bounded Go Call-Statement Extraction

* **Status**: Accepted
* **Date**: 2026-09-29

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Go callable extraction supports typed, revision-bound statement regions, bounded branches and loops, switch-case calls or assignments, and unambiguous ordinary comments, with live-in/live-out proofs and candidate validation. |
| description | Extraction can move code into a named helper and rewrite its original use site. Correctness depends on data flow, evaluation order, control flow, name resolution, and comment ownership. ADR-0053 separates extraction from insertion and movement but leaves its detailed schema and guarantees to a later decision. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

Extraction can move code into a named helper and rewrite its original use site.
Correctness depends on data flow, evaluation order, control flow, name resolution,
and comment ownership. ADR-0053 separates extraction from insertion and movement
but leaves its detailed schema and guarantees to a later decision.

The initial implementation needs a narrow Go contract that can prove its source
transformation without broad type inference or changing execution scope.

## Decision

This record defines the Go extraction implementation and proof limits. Java
extraction delegates to JDT LS under ADR-0068; its separate contract does not
broaden these Go guarantees.

Expose `refactor_extract_callable` and the `refactor-extract-callable` CLI command for Go.
The primary target is the new helper name in `symbol`. The existing owner is the
`in_function` selector. `file` identifies the workspace-relative Go source file.
`kind` is `function` or `method`. `start_line` and `end_line` select an inclusive
range by exact start and end lines of direct statements in the owner's body.
Go callers may instead select `construct_path` and `end_construct_path` from
`semantic_inspect_construct`, and copy that result's `expected_revision` to bind
the selector and full-file precondition to one captured source snapshot. These
paths may descend through explicit blocks, if/else bodies, counted-loop bodies,
and range-loop bodies. In one expression-switch case/default body, a range may
select contiguous direct call statements, or exactly one supported assignment,
among the clause's direct statements; path traversal does not descend into
nested case statements. Other assignments, declarations, other statement kinds,
ranges crossing cases, type-switch clauses, and select clauses refuse.

Nested paths, bounded control flow, expression-switch cases, and comment
preservation reuse this selector contract. They do not add another operation
or selector vocabulary.

Only a contiguous region of direct call expression statements, straight-line
simple assignments, or one already-supported complete control-flow statement
is supported. A path-selected call/assignment region may be inside a nested
lexical block; the replacement call remains at the selected statements' exact
execution point. Assignments may use `:=` or `=` and may mix
declarations with writes to existing bindings when Go object identity proves the
resulting scope. Tuple assignments are accepted when `go/types` resolves a
matching result tuple. Each identifier target must resolve to a variable; blank
targets are allowed. The engine derives live-ins and live-outs from typed reads
and writes rather than limiting assignments to owner parameters.

Bindings read before their first selected write become helper parameters.
Bindings written in the region and read after it become helper results in
first-write order. Existing owner locals assigned without a prior selected read
are declared inside the helper and returned when needed. A later write also
preserves a live-out declaration, with an explicit blank use at the owner when
Go would otherwise report it unused. Short declarations and plain assignments
may be mixed. Compound assignments, increment/decrement, non-identifier
assignment targets, control-flow statements, and unnameable helper types refuse.
The helper types come from resolved `go/types` objects, including named scalar,
struct, pointer, and interface types whose names are visible at package scope.
Function-local named types and type parameters remain unsupported. The original
and transformed package are both type-checked before application, so assignment
right-hand sides may include effectful expressions when moving the selected
statements preserves their order.

The helper receives each live-in owner parameter or local and, for function
extraction, a referenced receiver. Method extraction preserves the receiver on
the new method. The selected statements retain their order and are replaced by
one helper invocation. The helper is inserted immediately after the owner
declaration.
Function extraction passes a referenced receiver as a normal argument. Method
extraction preserves the receiver on the new method. The selected calls retain
their order and are replaced by one helper invocation. The helper is inserted
immediately after the owner declaration.

For method extraction, the requested name must be unexported and absent from
the receiver's value and pointer member sets, including promoted methods and
fields, and from interface method declarations in the package. Private helper
names cannot add implementations of external interfaces; the remaining checks
prevent changes to package-local interface satisfaction and method dispatch
while leaving the program type-correct.

The engine refuses a request before writing when it cannot establish the
following conditions:

* Each line boundary identifies exactly one direct owner-body statement. Path
  endpoints identify statements in the same lexical block and the inclusive
  range selects adjacent statements. Nested path traversal supports explicit
  blocks, if/else bodies, counted-loop bodies, and range-loop bodies, plus
  direct-call ranges or one supported assignment among one expression-switch
  case/default body's direct statements. The switch-case traversal does not
  descend into those statements. Switch case ranges cannot cross cases; other
  assignments, declarations, and statement kinds refuse. Type-switch and select
  clause paths refuse.
* Selected statements are direct calls or simple `:=` and `=` assignments to
  resolved identifier targets. Short declarations, existing locals, owner
  parameters, tuple assignments, and mixed declaration/assignment regions use
  typed object identity to derive live-ins, live-outs, and package-helper
  declarations. Assignment targets that are selectors, indexes, dereferences,
  or otherwise non-identifiers refuse. Compound assignments, increment and
  decrement, returns, branches, loops, labels, and other statement forms remain
  unsupported.
* Free identifiers resolve through the loaded Go type information to owner
  parameters, the receiver, same-file package declarations, imports, predeclared
  names, and safe owner locals. Shadowing is resolved by object identity rather
  than by identifier spelling.
* A forwarded parameter or receiver cannot be address-taken or closure-captured
  anywhere in the owner body. Selected direct `recover()` calls are refused.
  Selector calls rooted at a forwarded value are accepted only when type
  information proves the dispatch safe for the copied value. Method extraction
  includes its receiver in these checks even when the selected statements do
  not reference it, because the replacement method call implicitly reads and
  may copy the receiver.
* The selected region cannot take the address of the local returned by the
  helper, because the helper's local has a different address from the caller's
  result variable.
* The owner is non-generic, has no variadic parameters, and has no named results.
* The selected region contains no function literals or composite literals.
  Ordinary comments move only under the bounded comment ownership rule above;
  directives and partial or external comment ownership refuse.
* The generated helper name is a valid simple identifier and does not collide
  with a package declaration, same-receiver method, or predeclared Go identifier.

The typed output receipt reports publication state, the changed file revision,
and scoped package-local read-set evidence. Compiler package inputs, including
transitive source and module inputs used to validate the candidate, are captured
and revalidated internally before apply; the receipt does not claim that its
local read set is a complete transitive proof. Standalone function extraction returns the live helper in
`continuations` with `file`, `symbol`, and the selected file SHA-256
`expected_revision`. Batch child receipts report `applied_working` and omit live
continuations. Obtain fresh selectors with inspection or lookup after batch
publication. The line range is omitted because its coordinates no longer
identify the moved statements after a successful mutation. CLI output retains
the diff and diagnostic details. Writes follow ADR-0010's atomic publication
rules.

### Bounded control-flow forms

One complete owner-level `if/else` may move when it has no init clause, both
branches are blocks, the condition is a pure resolved scalar expression over
scalar owner inputs, constants, and literals, and each branch contains only
direct call statements. Reject missing else branches, else-if, nested control
flow, branch writes or exits, non-scalar owner inputs, and every alias, capture,
or evaluation-order hazard. Evaluate the condition and branch arguments in
their original branch and order.

One complete direct-owner counted loop may move when it short-declares one
integer index, uses a pure initializer and bound, compares with `<`, `<=`, `>`,
`>=`, and updates the same index with a direction-matched `++` or `--`. Its
body may contain direct calls, recursively supported scalar if/else statements,
and unlabelled `break` or `continue` targeting that loop. One complete range
loop may move for supported string/integer scalar, direct same-package
scalar-slice factory, local receive-capable channel, eligible map with scalar
key and element types, or an eligible slice identifier/two-index/three-index
expression over a nameable base slice with pure integer bounds. Preserve the
original range clause and iteration bindings. A range factory must be a direct
same-package function call returning one scalar-element slice, with pure scalar
arguments. A channel must be a direct receive-capable local or parameter used
only as the range operand. Reject owner-state writes, unsupported operands,
nested loops/switches/selects, labels, returns, unsupported exits, and any
unproven evaluation or type behavior. These are bounded forms, not a general
control-flow or termination proof.

An expression-switch case/default body accepts a contiguous direct-call range.
It also accepts exactly one simple assignment to a pre-existing owner local or
named owner parameter when typed object identity proves that binding and a
read after the enclosing switch. Return the assignment value from the helper
and assign it back at the original clause position. Receivers, switch-init or
clause-local targets, package variables, fields, `:=`, compound/tuple
assignments, cross-case ranges, type-switch/select paths, and assignments
without a post-switch read refuse.

### Comment ownership

Move ordinary comment groups only when every comment group inside the owner
body lies wholly within the selected statement range. Preserve content and
relative order, subject to Go formatting normalization. Refuse directive groups
and comments partly or wholly outside the selection. Keep a comment trailing
the owner's closing brace with that declaration, inserting the helper after its
complete trailing-comment line only when the anchor proves the placement.

## Invariants

* `symbol` names the new helper; `in_function` only selects the existing owner.
* `start_line` and `end_line` are positive, inclusive, one-based line numbers,
  not byte offsets. They select direct owner-body statements at exact AST
  boundaries. Alternatively, `construct_path` and optional
  `end_construct_path` select revision-bound zero-based statement paths returned
  by `semantic_inspect_construct`. Nested paths can select statements in
  explicit blocks, if/else bodies, counted-loop bodies, and range-loop bodies.
  An expression-switch case/default body accepts direct-call ranges or one
  supported assignment among that clause's direct statements;
  traversal does not descend into nested case statements. Both endpoints must
  identify adjacent statements in the same lexical block. Other assignments,
  declarations, other statement kinds, and cross-case ranges refuse. Type-switch
  and select clause paths refuse. Path selection cannot be mixed with line
  selectors. Unsupported
  paths, out-of-range paths, and stale file revisions refuse before mutation.
* The operation preserves statement order and assignment evaluation order.
  Resolved live-ins become parameters and live-outs become results in first-write
  order. Each helper signature type must be expressible from package-visible
  source names; the original and generated packages are type-checked before
  application.
* Unsupported or ambiguous selections fail before mutation.
* Live-out analysis uses resolved Go object identity. A later write preserves
  the declaration even when no later read occurs; the owner gets a blank use
  when Go would otherwise report an unused local. Plain `=` and `:=` targets
  are not reads, while compound assignments and increment/decrement are.
* `_test.go` targets load the applicable internal or external test variant for
  both original type resolution and transformed-candidate compilation.
* Original-source result binding and transformed-candidate compilation remain
  distinct validations. Function extraction retains package syntax and type
  information to prove object identity, live-in/live-out flow, and safe method
  dispatch. Both function checks retain compiler-error handling and
  captured-input revalidation; one program's loaded types cannot prove that the
  other program compiles. The original check also applies when there are no live
  results: introducing a helper could otherwise conceal a pre-existing
  unresolved call to that name in another source file.
* The Go `refactor_extract_callable` implementation has CLI txtar coverage for
  its observable success and refusal behavior. Java coverage is governed by
  ADR-0068 and the cross-language coverage invariant in ADR-0028.
* Broader statement extraction, general return-value inference, expression
  extraction, and gopls-driven extraction are not added to this Go contract
  until their own safety guarantees are established.

## Consequences

This record now contains the complete Go `refactor_extract_callable` contract,
including the bounded branch, loop, switch-case, and comment-preservation
previously separate callable-extraction extensions.

The operation reduces common manual helper extractions to one semantic call.
Compiler-derived types and object-identity flow support multiple live-ins and
live-outs while retaining fail-closed boundaries for control flow, aliasing,
captures, ambiguous comment ownership, and unnameable helper signatures. The
shared schema and registry expose the operation through both CLI and MCP, and
its receipt follows ADR-0055 continuation naming.
