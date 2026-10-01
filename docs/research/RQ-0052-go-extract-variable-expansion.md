<!-- This handover exists to expand the bounded Go extraction contract without losing the evaluation guarantees that made the first operation safe. -->
# RQ-0052: Go Extract Variable Expansion

* **Status**: Open
* **Category**: Semantics & Refactoring
* **Date**: 2026-09-30

## Context

ADR-0059 and `semantic_extract_variable` cover one exact, side-effect-free first argument in a direct call statement. The implementation permits later call arguments because the extracted expression is the first argument and later evaluation order is retained. It refuses callable owner parameters/locals, builtins, functions declared only in another file, and files with import bindings that could shadow the call; the call must resolve syntactically to one unshadowed same-file package function. It still rejects returns, nested expressions, constant-only expressions, comments, and other forms. Expansion can reduce routine manual work, but expression movement can change evaluation order, typing, scope, and observable behavior.

## Research questions

1. Which next expression positions can be extracted while preserving Go evaluation order and inferred type?
2. Can repeated identical expression sites share one new local without moving evaluation across branches or loops?
3. Which supported cases need `go/types` or retained gopls facts rather than syntax-only proofs?
4. How should target selection and refreshed continuation receipts behave after inserting a declaration?
5. Which expansion is useful enough for a separate milestone instead of broadening one hard-to-audit contract?

## Current evidence and limits

* `internal/astedit/extract_variable.go` implements the bounded engine; `internal/operation/wire_extract.go` defines `semantic_extract_variable`, CLI schema, and receipt.
* ADR-0059 defines `symbol` as the new local, `in_function` as owner selector, `discriminator` as exact whole-expression source, and `construct_path` as a zero-based occurrence path.
* The current implementation accepts only the first whole argument of a direct call statement in the owner's direct body. It requires a typed owner input and an unshadowed same-file package function callee; it rejects callable owner parameters/locals, builtins, cross-file callees, constants-only expressions, calls, receives, address-taking, literals, comments, returns, and nested positions.
* `testdata/scripts/extract_variable_typed_context.txtar` checks defined numeric and interface owner parameter types and a zero-write refusal for a callable parameter.
* Existing file edits are batch-buffered in the authoritative pending source-buffer state. Expansion must continue to read and edit that same overlay.
* This is an expansion of the existing tool, not a second `extract_expression` alias and not a constant-extraction operation.

## Candidate outcomes and non-goals

The first-argument case in calls with later arguments is implemented and covered because it moves the first argument before an unshadowed same-file package function call; it does not extract later arguments. Further work can compare direct RHS assignments and direct return expressions only after type/context and ordering proofs. Repeated-site extraction is a separate candidate: one local is only valid when all sites execute at the same frequency and control-flow point.

Do not move expressions out of short-circuit operands, conditionals, loop headers/bodies, defer/go calls, or closures in the first expansion. Do not infer arbitrary interfaces, generate constants, perform extract-function work, or claim behavioral equivalence from formatting and diagnostics alone.

## Proposed API questions

Retain `semantic_extract_variable` / `extract-variable`. Proposed primitive inputs:

* `file` string: exact workspace-relative Go file.
* `symbol` string: new local name; simple Go identifier, not qualified.
* `in_function` string: fully-qualified owner selector copied from an owner receipt.
* `discriminator` string: exact source bytes of the whole selected expression, whitespace significant.
* `construct_path` string: proposed canonical zero-based path when source text occurs more than once; determine whether it is required for all nested positions or only ambiguity.
* `expected_revision` string: investigate a canonical source revision token with identical input/output name and format; it must bind the fragile expression selection and be checked against the overlay.

All of `file`, `symbol`, `in_function`, and `discriminator` are required. `construct_path` is optional only when the selected text is unique; if supplied, it is zero-based and requires the matching `expected_revision`. No default owner or expression is inferred. `expected_revision` is an optional opaque string on an initial request and required when following a revision-bound receipt. Import-organizing options are not needed because this rewrite does not introduce imports.

Input positions such as `argument_index` or `in_return` are unresolved alternatives. Prefer one schema that names the structural role without overloading `symbol`. Any new receipt-fed input must have an exact compatible `SourceFields` entry and prose linking it to its source tool and field. Caller-authored choices have no invented producer edge. Do not add structured selector objects until recursive schema support exists.

## Continuation receipt proposal

Return the new local as a live `symbol`; return current `file` and `in_function`. Return a refreshed `discriminator` only if it still selects a live expression for a safe next edit. Return a refreshed zero-based `construct_path` only with the matching `expected_revision`. Put removed expression text and old path under evidence, clearly marked non-reusable.

```json
{
  "file": "pkg/serve.go",
  "symbol": "normalized",
  "in_function": "(*Server).Serve",
  "discriminator": "strings.TrimSpace(raw)",
  "construct_path": "2.1.0",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/serve.go"],
  "diagnostic_status": "unchanged"
}
```

The example path is zero-based and is copyable only with its exact `expected_revision`; decide revision encoding before implementation. Removed source is evidence, never a continuation.

## Safety questions and hazards

* Multiple arguments: extracting argument 2 into a preceding local evaluates it before argument 1, changing order. Only permit if the selected argument is first or preceding arguments are proven pure and stable; syntax-only purity classification may be too weak.
* Assignment and return contexts: a preceding local can change when other operands are evaluated and whether a value is evaluated on a branch.
* Repeated occurrences: evaluating once instead of N times changes side effects, panics, allocation identity, and timing. Reject calls and receives unless repeated evaluation behavior is explicitly preserved.
* Type changes: untyped constants, nil, generic inference, and assignability may change when a new declaration is introduced. Require type facts or reject.
* Name binding: selected names may be shadowed in sibling scopes; identifier spelling alone is insufficient.
* Comments, directives, and build constraints must remain attached and unchanged.
* A request can become stale after an earlier batch edit; selection and revision validation use the live overlay.

## Ordered milestones

1. Build a table of expression contexts, evaluation order, type rules, and refusal reasons using real Go examples.
2. Select exactly one next useful context and prove the transform with `go/types` or another existing authoritative service.
3. Specify how `construct_path`, ambiguity receipts, and optional `expected_revision` identify the selected live expression.
4. Implement overlay-aware selection, validation, complete-source formatting, `Buffer.Apply` publication, and diagnostics using the configured scope after finalization.
5. Expand the CLI/MCP schema and Go capability, with no compatibility alias or non-Go claim.

## Public behavior tests to add

* CLI txtar success for a typed expression in the selected newly supported context; assert exact final file tree, local placement, receipt, and diagnostics.
* CLI txtar refusal for an expression whose movement reorders a side-effecting sibling; assert refusal reason and byte-identical source.
* CLI txtar refusal for stale `expected_revision` or ambiguous repeated source; assert valid fresh selector evidence but no mutation.
* CLI txtar sequential-command success edits the owner, extracts from the resulting on-disk source, then applies a later body edit; assert each command result and the final source. The MCP batch case separately covers the unsaved overlay.
* MCP `semantic_batch` case: statically specify an owner-body edit followed by extraction against its known unsaved result and a later edit. Any `construct_path` and `expected_revision` are literal fixture-derived values, not references to the earlier result. After the batch, make a separate follow-up mutation using the returned local `symbol`, then run a standalone lookup to prove the new local remains live. A separate refusal invocation proves the earlier successful batch mutation remains published.
* Add refusal case where an unsafe extraction follows a successful insertion; the batch stops, the insertion remains eligible for final publication, and any repair/later edit runs in a separate invocation.

## Shared handover contract

This plan depends on [RQ-0024 shared handover contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract) for common policy. The pending authoritative `main` snapshot is commit `6c95d1260d8ddab7a98a3832b5faa7401ed81581` at `/Users/alessandro/sources/semantic-editor`, including uncommitted session and source-buffer migration work. The target worktree is older and lacks these changes. Rebase implementation work on the integrated authoritative state before coding; do not copy code from the stale worktree blindly.

All existing Go edit operations must be `Batchable` and `BufferedBatch`. Read and write through `internal/sourcebuf`; each operation observes the current overlay, not disk-only contents. A refusal must not write its own target or publish an intermediate source revision. Earlier successful mutations remain eligible for normal batch finalization and publication after the batch stops. Each operation validates all of its outputs together with `Buffer.Apply`; batch finalization stages the completed outputs and publishes them through `Buffer.Flush`, atomically per file. Per-file atomicity does not promise global rollback after publication starts. No operation may write an intermediate revision or introduce a batch barrier.

Keep the retained `gopls` process and shared MCP session. Buffered execution defers source normalization until the full changed-source set is ready and normalizes only changed sources. Diagnostic checks retain the configured project/module scope, selected coverage, and completeness; they are not narrowed to changed files. Use only existing `diagnostic_status` values: `unchanged`, `changed`, `unavailable`, or `deferred`. Do not invent a diagnostic-delta JSON shape in this handover; implementation must use the registered `DiagnosticDelta` output schema. Preserve existing diagnostic deltas and changed-file receipts. A batch stops at its first refusal; prior successful overlay mutations remain eligible for normal finalization/publication, while later edits require a separate invocation. Batch entries are a static ordered list with no dynamic receipt references or interpolation. `semantic_lookup` is not batchable and must run outside the batch, after it, when fresh lookup is needed. A no-op must not claim a source change.

The current `sourcebuf.Buffer.Apply` accepts existing files. New-file creation, package moves, and resource operations require a separate source-buffer resource design or an explicit successful scaffold prerequisite; do not smuggle them through direct disk writes. Safe operations must prove against the overlay and revalidate every source they read before apply. Failure to enumerate or revalidate the source set is a refusal, never a silent downgrade. For operations scoped inside a named Go type, use `in_type` consistently as the owner selector; when a new member is the target, `symbol` identifies it as `Type.Member`.

Any `expected_revision` proposed below is an opaque input/output field with that exact same name; copy its live output value verbatim to bind fragile selectors against the current overlay. Its encoding is a decision gate, not a guessed format.

Use ADR-0055 roles: `symbol` is the primary target; `file` is the workspace-relative Go file; `in_function` is an owner selector; `discriminator` is exact source text. `construct_path` is zero-based and may be used or returned only with the matching `expected_revision` for that source revision. Receipts return selectors only while they address live source. Removed names, old expressions, and stale paths belong under evidence fields and are not continuations. Add `SourceFields` to primitive input parameters and text that identifies the exact receipt field. Current registry schemas support primitive fields only; if a proposal needs nested edits or lists, land recursive schema support before advertising it. Go only until another backend has comparable behavior and CLI txtar coverage.

## Ordered research and implementation steps

1. Audit the current operation registry, engine, backend capability metadata, MCP buffer adapter, and `sourcebuf` on the rebased main. Record any contract drift before proposing code.
2. Decide the unresolved selection, type, comment, resource, and refusal rules below. Update RQ-0024 only through its owner, then prepare an ADR if the design changes a public invariant.
3. Implement engine behavior behind a typed operation contract; use semantic navigation/insertion tools where they support the target. Log any failed semantic tool call in `docs/SUBOPTIMAL_TOOLS.md` before manual repair.
4. Register CLI and MCP schemas, descriptions, `SourceFields`, result formatting, Go capability metadata, and current documentation. Do not advertise other languages.
5. Add the literal CLI txtar cases below, including final file trees, receipts/diagnostics, unchanged refusal state, and the stated mixed batch order. Run repository checks only after implementation is authorized; this document task does not run tests.
6. Update this RQ to Resolved with findings and link the accepted ADR after evidence and review establish the contract.

## Cross-cutting acceptance gates

* Every public success and refusal has CLI txtar coverage; an MCP batch case additionally proves overlay composition. Record each concrete archive path and observable contract in `docs/MANUAL_EDITS.md`; keep literal commands and before/after trees in the txtar fixture itself.
* Refusal tests assert observable diagnostics and unchanged target bytes; batch tests also assert earlier successes remain in the final workspace.
* Tests exercise observable CLI/MCP behavior, not private helper structure or shallow fake LSPs. For each successful operation, include a separate follow-up mutation using live receipt fields, then a standalone `semantic_lookup` or internal probe after that mutation to prove the target is fresh; never include lookup as a batch child. For removal, use a still-live owner or sibling target because the deleted symbol is historical evidence.
* Add exact schema and capability assertions when a new operation is introduced. Preserve matching CLI and MCP names, types, defaults, and value formats.
* No global rollback claim, speculative preview engine, framework-specific behavior, or non-Go capability is implied.

## Related contract records

* [RQ-0024 shared handover contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract) (parent-owned shared rules and index entry)
* ADR-0055, `docs/adr/0055-tool-parameter-schema-harmonization.md`
* ADR-0010 atomic disk update invariant; ADR-0032 direct batch execution; ADR-0042 structured MCP output schemas
* `docs/MANUAL_EDITS.md` records prior manual edits and capability gaps; preserve literal txtar fixtures.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_variable`:**

> Use this tool instead of copying an expression and replacing its uses to extract one supported Go expression into a preceding local without changing observable evaluation. Choose a fresh simple `symbol`; `in_function`, `discriminator`, and any revision-bound `construct_path` select the expression. Pass the new local's live `file`, `symbol`, and `in_function` to `semantic_inline_variable` when its binding/use contract supports the result. Pass live `file` and `in_function` to construct operations. Reuse an expression selector only if the result explicitly recomputes a current eligible expression; old source text is evidence.

**Parameter description requirements:**

`symbol`: Target: fresh simple local identifier; qualified declaration outputs and an already-created local do not supply a new name. `in_function`: Selector: live same-named owner from `semantic_extract_variable.in_function` or a compatible construct receipt. `discriminator`: Selector: current selected expression only; remove `semantic_extract_variable.discriminator` as an automatic source when it represents replaced text. A new lookup/inspection projection must prove its live context before reuse.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
