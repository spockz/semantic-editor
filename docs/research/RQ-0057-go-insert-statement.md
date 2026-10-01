<!-- This handover exists to define safe insertion of one Go statement into a live block using ADR-0055 owner and construct selectors. -->
# RQ-0057: Go Insert Statement

* **Status**: Open
* **Category**: Structural Editing
* **Date**: 2026-09-30

## Context

Current tools can replace a whole function body, replace a supported whole construct, and insert switch cases. They cannot insert one statement into an existing block. `docs/MANUAL_EDITS.md` ME-0041 and ME-0047 identify guard/call insertion as a repeated gap. A syntactically valid statement can still alter control flow, variable scope, defer timing, or execution count.

## Research questions

1. Which block slots are safe and useful: before/after a direct statement, first/last, or named control-flow slot?
2. How should an existing statement be selected under ADR-0055 without exposing byte offsets or stale line coordinates?
3. Which short declarations and labels can be inserted without shadowing or changing goto legality?
4. How should receipts identify an insertion that has no symbol and no longer has the original anchor text?
5. What behavior can a syntax-only AST operation promise, and what requires type or control-flow analysis?

## Current evidence and scope

* `internal/astedit/body.go`, `construct.go`, and `switchcase.go` implement broader replacement or case insertion, not general statement insertion.
* `internal/operation/wire_engine_mutations.go` exposes body/construct/case operations through typed contracts; no `semantic_insert_statement` registration exists in the current source scan.
* `docs/MANUAL_EDITS.md` names guard/call insertion as a missing capability and proposes `semantic_insert_statement`.
* The current receipt contract uses `in_function` for owner selection and `discriminator` / `construct_path` for a selected construct. A statement insertion itself has no fabricated `symbol`.

## Outcomes and non-goals

First release should insert exactly one parsed statement adjacent to one uniquely selected direct statement in an existing function or method body. Consider `first`/`last` slots only after proving comments and scope boundaries. Refuse control-flow constructs or label interactions that cannot be checked. This is a new primitive, distinct from replacing the whole function body or inserting a switch case.

Do not insert arbitrary statement lists, split/merge blocks, rewrite expressions, create labels, insert into generated files with directives, or claim behavior-preserving guard transformations. Framework-specific patterns remain skills.

## Proposed API

Propose `semantic_insert_statement` / `insert-statement`, Go only, existing source, batchable and buffered.

* `file` string: workspace-relative file.
* `in_function` string: fully-qualified containing function/method selector.
* `discriminator` string: exact current source of the direct statement used as anchor.
* `construct_path` string: zero-based direct statement path; required when same text is ambiguous, and refreshed on success only if it still refers to a live statement.
* `placement` enum: `before_construct` or `after_construct` per the shared placement vocabulary; first/last remain unresolved.
* `source` string: exactly one Go statement, no enclosing block or function.
* `expected_revision` string: investigate exact owner/block revision binding.

`file`, `in_function`, `source`, and `placement` are required; `placement` accepts exactly `before_construct` or `after_construct`, with no implicit default. `discriminator` is required for the selected anchor and `construct_path` is optional only when unique; if present, it is zero-based and requires its matching `expected_revision`. `expected_revision` is optional on an initial request and required when following a prior receipt. `auto_organize_imports` is Boolean default `true` if the inserted statement can introduce imported identifiers; confirm whether import edits belong in this primitive.

Do not add `symbol`; inserted statements have no declaration target by definition. Use `SourceFields` annotations on copyable file/owner/selector values. If adding structured statement arrays, wait for recursive schema support.

## Continuation receipt proposal

Return `file`, `in_function`, placement, and a refreshed selector for a still-live chosen statement if unambiguous. The inserted statement text can be evidence, but is not a `symbol`. The original anchor source/path is stale after insertion and belongs under `anchor_before` evidence unless recomputed.

```json
{
  "file": "pkg/handler.go",
  "in_function": "Handler.Serve",
  "placement": "before_construct",
  "discriminator": "if ready { return nil }",
  "construct_path": "3",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/handler.go"],
  "diagnostic_status": "unchanged"
}
```

Use the shared zero-based `construct_path` encoding and require the matching `expected_revision`; define any remaining path spelling once through the shared contract. Never return a fabricated symbol for an expression or statement.

## Semantic hazards and refusal rules

* `:=` can introduce shadowed locals or change which later identifiers bind; type/name resolution must validate the resulting block.
* `return`, `break`, `continue`, `goto`, labels, `defer`, and `go` alter control flow or timing; do not treat insertion as inert.
* Inserting before a declaration can make variables unavailable; after a declaration may move code into a different scope.
* Comments/directives attached to the anchor can shift ownership; only insert at AST-safe boundaries and preserve bytes.
* A repeated statement discriminator requires an explicit path; silently selecting first/nearest is invalid.
* A previous batch edit may change anchors. Resolve text and path against current overlay, with stale expected revision refusal.
* Compile success does not prove intended guard behavior or resource cleanup ordering.

## Ordered milestones

1. Define the initial allowed statement grammar and explicit control-flow refusal list.
2. Settle selector path and expected revision format across construct operations.
3. Implement exact direct-statement selection, parse/type validation, overlay writes, and fresh receipt generation.
4. Add registry/CLI/MCP schema and Go-only capability metadata.
5. Consider additional slots or statement forms only after focused user workflows justify them.

## Public behavior tests to add

* CLI txtar success inserts a simple assignment or call with `before_construct` and `after_construct` relative to an exact direct statement; assert exact final tree and refreshed selector.
* CLI txtar refusal for repeated discriminator without path, stale path/revision, nested statement anchor, malformed/multiple statements, illegal label/goto placement, and short-declaration collision; assert unchanged source.
* CLI txtar refusal for unsafe defer/goto/return form unless the accepted first-release grammar explicitly supports it.
* MCP batch: statically specify body replacement, statement insertion against the known unsaved body, and a later edit; any zero-based `construct_path` is paired with the matching fixture-derived `expected_revision`. No result is interpolated inside the batch. Run a separate follow-up mutation against a live owner/declaration, then a standalone lookup to prove the target remains fresh.
* MCP batch with one successful declaration edit followed by insertion refusal stops at refusal; the earlier edit remains eligible for publication and no intermediate write occurs. Any later success is a separate invocation.
* Assert receipts contain `in_function` and selector evidence but no `symbol` for the inserted statement.

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

**Tool description for `semantic_insert_statement`:**

> Use this tool instead of replacing a function body to insert one Go statement before or after a selected current direct statement. `in_function` selects the owner; `discriminator` and an optional revision-bound `construct_path` select the anchor. Supply one statement as `source` and required `placement=before_construct|after_construct`. Pass live result `file`, `in_function`, and recomputed eligible anchor fields to another `semantic_insert_statement` call with a new `source`. Pass the owner to `semantic_replace_construct` with that tool's own required selection. Old anchor paths are evidence, not live selectors.

**Parameter description requirements:**

`in_function`: Selector: copy a live same-named owner projection from `semantic_insert_statement` or a compatible construct receipt. `discriminator`: Selector: exact current direct-statement anchor; copy `semantic_insert_statement.discriminator` only when it is a recomputed live anchor. `construct_path` and `expected_revision` must be copied from the same selected-file result. `source`: Replacement: caller-authored single statement; an anchor result never supplies it implicitly.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
