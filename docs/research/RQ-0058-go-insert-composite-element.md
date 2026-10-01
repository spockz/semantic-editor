<!-- This handover exists to define one-element insertion into an existing Go composite literal without conflating the holder with the inserted value. -->
# RQ-0058: Go Insert Composite Element

* **Status**: Open
* **Category**: Structural Editing
* **Date**: 2026-09-30

## Context

Existing declaration and body tools can replace large source units but cannot insert one element into an existing composite literal. `docs/MANUAL_EDITS.md` ME-0041 identifies composite element insertion as a gap. Composite literals can be keyed or positional, nested, typed, and subject to duplicate keys or evaluation-order changes.

## Research questions

1. Which literal kinds are supported first: struct, array, slice, map, keyed, or positional?
2. How is the containing declaration selected without treating it as the element's symbol owner?
3. How are duplicate keys, unknown struct fields, map key collisions, ellipses, and elided nested types validated?
4. Should insertion append only, or support before/after keyed elements with stable anchors?
5. Can the engine prove the literal is the intended one and that a batch edit has not changed its source revision?

## Current evidence and limits

* Current structural editing includes declaration insertion/replacement and case insertion; no `semantic_insert_composite_element` registration or corresponding AST implementation appears in the current source scan.
* `docs/MANUAL_EDITS.md` ME-0041 explicitly proposes `semantic_insert_composite_element` because current tools cannot target one literal element proportionately.
* This operation has no standalone declaration `symbol`. The holder is selected with `in_declaration`, as required by the shared handover contract; `in_function` may be needed only when the literal is local and declaration selection is insufficient.
* The root checkout predates pending `sourcebuf` changes. Implementation must use the integrated overlay and not direct-write disk.

## Outcomes and non-goals

Start with one typed keyed struct or map element in a uniquely selected literal. Research whether slice/array append can be safely supported with explicit ordering. Do not rewrite whole literals, synthesize default values, infer an interface, or insert arbitrary nested element lists. Do not claim no runtime behavior change when adding an element whose value expression has side effects.

## Proposed API

Propose `semantic_insert_composite_element` / `insert-composite-element`, Go-only, existing-file, `Batchable` and `BufferedBatch`.

* `file` string: exact workspace-relative file.
* `in_declaration` string: containing declaration selector (the holder, not `symbol`).
* `discriminator` string: exact source text of the selected composite literal or its uniquely identifying expression.
* `construct_path` string: zero-based AST path to the literal, disambiguating repeated literals in the declaration.
* `source` string: one complete element expression, including `key: value` when keyed; no surrounding comma or braces.
* `placement` enum: first release supports only `append`; it has no other value.
* `expected_revision` string: stale declaration/literal guard proposal.

`file`, `in_declaration`, `discriminator`, and `source` are required primitive strings. `construct_path` is optional only when the literal is unique; if present, it is zero-based and accepted only with its matching `expected_revision`. `placement` is an optional enum with default and only allowed value `append`; do not accept a target anchor in the first release. `expected_revision` is optional initially and required for a receipt-based follow-up. `auto_organize_imports` is Boolean default `true` only if generated imports are in scope; otherwise omit it.

Avoid `symbol` for a non-declaration element. Use `SourceFields` for `file`, `in_declaration`, `discriminator`, and any copyable refreshed path. No nested input objects until recursive registry schema support exists.

## Continuation receipt proposal

Return `file`, `in_declaration`, and refreshed `discriminator` / `construct_path` only when they still select the live literal after insertion. Return inserted element source and literal kind under evidence; it is not a live symbol.

```json
{
  "file": "pkg/routes.go",
  "in_declaration": "routes",
  "discriminator": "map[string]Handler{\"health\": health}",
  "construct_path": "1.0",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/routes.go"],
  "diagnostic_status": "unchanged"
}
```

The zero-based path is valid only with this matching `expected_revision`. A refreshed path must be recomputed after insertion; old element paths belong to evidence only.

## Semantic hazards and refusal policy

* Struct literal keyed fields must exist and must not be duplicated; positional struct literals require exact field order and arity.
* Map keys can collide semantically even when text differs (`1` and a typed constant); type-check or refuse uncertain keys.
* Slice/array insertion order affects runtime behavior and index semantics; append-only may be the only initial option.
* Element expressions can have side effects; inserting them changes execution count. Report structural nature and avoid behavior claims.
* Elided nested literal types and generic inference may require type information unavailable from syntax alone.
* Comments around commas/elements have ownership; preserve or refuse ambiguous comments.
* Repeated literals in one declaration require `construct_path`; never pick the first match silently.
* Each operation must resolve its selector against the current overlay; batch entries remain static, so no refreshed result is interpolated. Use matching fixture-derived revisions for zero-based paths, then verify with a separate post-batch lookup.

## Ordered milestones

1. Choose literal families and keyed/positional policy; document exact refusal classes.
2. Specify `in_declaration`, `construct_path`, `discriminator`, and expected revision formats consistently with ADR-0055.
3. Implement one element insertion over current overlay, with full-literal type/syntax validation and no intermediate writes.
4. Register primitive schemas, capability metadata, receipts, and CLI/MCP tests.
5. Expand to nested, positional, or anchored insertion only when a type-safe policy exists.

## Public behavior tests to add

* CLI txtar success appends one keyed map element and one keyed struct element; assert exact literal and receipt.
* CLI txtar refusal for duplicate map key/struct field, unknown struct field, malformed expression, positional literal under an unsupported policy, and repeated literal without path; target file stays unchanged.
* CLI txtar sequential commands replace the declaration, append to its resulting literal, then edit the declaration again; assert exact results after each command. The MCP batch test separately exercises the unsaved overlay with static fixture-known arguments.
* MCP batch statically specifies declaration replacement, append insertion against the known unsaved literal, and a later edit; any zero-based path is paired with the exact fixture-derived revision. The batch does not interpolate prior results. Verify the final tree and diagnostics under configured scope, then make a separate follow-up mutation using the containing `in_declaration`, followed by a standalone lookup to prove freshness.
* MCP batch with a successful edit followed by malformed/ambiguous insertion stops at refusal; the earlier edit remains eligible for publication and the element is absent. Any later success runs in a separate invocation.
* Schema/receipt cases prove `in_declaration` is the holder selector and no fake `symbol` is returned.

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

**Tool description for `semantic_insert_composite_element`:**

> Use this tool instead of replacing a declaration or composite literal to append one supported Go literal element. `in_declaration` selects the holder; `discriminator` and any revision-bound `construct_path` select the exact literal. Supply one `source` element, including its key when keyed; `placement` defaults to append. Pass the live result `file`, `in_declaration`, refreshed `discriminator`, and matching path/revision directly to another `semantic_insert_composite_element` call with a new element. The inserted element and old literal text are evidence; insertion is not a behavior-preserving refactoring.

**Parameter description requirements:**

`in_declaration`: Selector: copy proposed `semantic_inspect_symbol.in_declaration` only after shared inspection exposes that exact holder field. `discriminator`: Selector: full current literal text from `semantic_insert_composite_element.discriminator` only when regenerated after the edit. `source`: Replacement: one element, without enclosing braces or comma; never confuse it with literal selector text. Descriptive literal `kind` does not feed another tool's kind enum.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
