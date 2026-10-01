<!-- This handover exists to define precise field insertion into an existing Go struct and avoid duplicating the already-supported receiver-method declaration tool. -->
# RQ-0056: Go Insert Field

* **Status**: Open
* **Category**: Structural Editing
* **Date**: 2026-09-30

## Context

Existing declaration insertion handles complete functions/methods, new type declarations, and top-level constants/variables. It cannot add a field to an existing struct proportionately. `docs/MANUAL_EDITS.md` records this gap and suggests `semantic_insert_field`. Adding a field affects keyed/unkeyed composite literals, embedded-field promotion, tags, and API shape, so syntax insertion alone is not always semantics-preserving.

## Research questions

1. What field source forms are supported: named, grouped names, embedded fields, tags, generic type expressions, and documentation comments?
2. How should insertion detect field-name collisions and promoted-name conflicts without claiming full selector compatibility?
3. What source coverage should warn or refuse when positional composite literals exist?
4. Should the operation insert only source or also update in-package keyed literals? The latter is a separate coordinated transform.
5. What is the stable selector and receipt for the existing type and new field?

## Current evidence and scope boundary

* `internal/astedit/function.go` and `internal/operation/wire_engine_declarations.go` already insert complete receiver method declarations; Go methods are file-level declarations. A generic add-member tool for methods would duplicate existing capability.
* `internal/astedit/type.go` and `semantic_insert_type` add a new struct/interface/type alias, not a field inside an existing type.
* `docs/MANUAL_EDITS.md` rows ME-0041 and ME-0047 state that existing tools cannot insert one struct field; they suggest `semantic_insert_field`.
* A field insertion tool is a genuine missing operation. It is distinct from method insertion and from whole-type replacement.
* Pending main has sourcebuf overlays and batch support; the root worktree is behind that migration.

## Outcomes and non-goals

Target a single named field inserted into one existing, non-alias Go struct. Investigate grouped fields and embeddings as later milestones. Preserve surrounding fields and comments. Do not rewrite composite literals, implement interfaces, alter method sets, perform field renames, or generate accessors in the first release.

Avoid claiming that positional composite literals remain valid. Before accepting the operation, research package/module scan cost and whether the tool should warn/refuse or require caller acknowledgment. No generic `semantic_add_member` alias is proposed because receiver methods are already expressible through `semantic_insert_function`.

## Proposed API

Propose `semantic_insert_field` / `insert-field`, Go only, existing-file scope, `Batchable: true`, `BufferedBatch: true`.

* `file` string: workspace-relative source file.
* `in_type` string: fully-qualified struct type selector; common selector spelling shared with future member/type plans. It is a scope selector.
* `symbol` string: target new field in qualified continuation form `Type.Field`; validate the simple field suffix and require it to agree with source.
* `source` string: exactly one Go field declaration, including optional tag and field doc comment; never accept an entire struct body.
* `placement` enum: `type_start`, `type_end`, `before_symbol`, or `after_symbol` within the selected struct; default `type_end`. `before_symbol` / `after_symbol` retain the existing qualified anchor semantics.
* `target_symbol` string: existing field name required only for before/after placement.
* `expected_revision` string: investigate stale-struct protection and exact receipt identity.

`file`, `in_type`, `symbol`, and `source` are required primitive strings. `placement` is an optional enum with values `type_start`, `type_end`, `before_symbol`, `after_symbol`; default is `type_end` and the start/end values follow the shared RQ-0024 placement vocabulary. `target_symbol` is required only for `before_symbol` / `after_symbol` and keeps existing qualified anchor semantics. `auto_organize_imports` is Boolean default `true`. `expected_revision` is an optional opaque string initially and required on receipt-based follow-up. `construct_path` if introduced for nested/ambiguous type targets must be zero-based and accepted only with its matching revision. Use `SourceFields` only for proven compatible file, owner-type, revision, and existing-anchor sources, with exact source-field names in their descriptions. The new field identity is a caller-authored target, not an existing-field receipt copied into another insertion. Avoid a nested field descriptor until recursive schema support exists.

## Continuation receipt proposal

Return the new live field as `symbol` in `Type.Field` format and exact `file`/`in_type`. Return refreshed placement fields only when an anchor still exists. Include changed files and diagnostics.

```json
{
  "file": "pkg/config.go",
  "in_type": "Config",
  "symbol": "Config.Timeout",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/config.go"],
  "diagnostic_status": "unchanged"
}
```

Do not return byte offsets or an obsolete pre-insertion `construct_path`. The field type and source are evidence, not necessarily a reusable operation target.

## Semantic hazards and refusals

* Duplicate direct field names are invalid; embedded-field promotions can create ambiguous selectors and need an explicit policy.
* Unkeyed composite literals in the package or downstream packages can break when a field is added. Syntax validity does not establish API compatibility.
* Tags affect encoding, reflection, persistence, and external consumers; preserve source bytes exactly and do not synthesize tags.
* A field name can collide with promoted names or method names and affect selector resolution.
* Generic struct type declarations and aliases need careful type resolution; initial release may refuse them.
* Documentation comments and directives attach to fields differently from type docs; preserve adjacent comments or refuse ambiguous ownership.
* Same-batch replacement may create/remove the target struct. Resolve against the current overlay and require it to exist at the insertion step.

## Ordered milestones

1. Specify supported `ast.Field` shape, placement, comment/tag ownership, type resolution, and collision rules.
2. Study positional literal and external API compatibility policy; define refusal/warning behavior and checks.
3. Implement one field insertion against overlay source, validate parse/type/diagnostics and publication read set.
4. Register primitive schema, tool descriptions, `SourceFields`, Go capability, CLI and MCP receipts.
5. Expand to embeddings/grouped fields only with separate evidence and comparable txtar coverage.

## Public behavior tests to add

* CLI txtar success inserts a single typed field before/after a named field and at last; assert exact struct source, comments, tag, formatted output, and receipt.
* CLI txtar refusal for duplicate field, malformed field, wrong `in_type`, missing anchor, multiple declarations, unkeyed literal compatibility risk per selected policy, and stale revision; source remains byte-identical.
* CLI txtar sequential commands first create the target type, then insert the field and a method that reads it; assert each resulting tree and receipt. The separate MCP batch uses statically specified entries and fixture-derived revisions, with no result interpolation. A separate follow-up mutation consumes `Config.Timeout`, then a standalone lookup proves the field target remains live.
* MCP `semantic_batch` with an earlier unsaved type edit, then a field insertion and a later method insertion proves each operation sees the overlay; use static selectors and matching fixture-derived revisions. Diagnostics retain configured coverage at finalization; a separate follow-up lookup proves freshness.
* MCP batch refusal after an earlier successful edit proves no rollback of the earlier success and no partial field write.
* Include receipt/schema assertions that distinguish `symbol=Type.Field` from `in_type=Type`; do not add duplicate method-add tests because existing insertion covers methods.

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

**Tool description for `semantic_insert_field`:**

> Use this tool instead of replacing a struct body to insert one supported Go field declaration. `in_type` selects the struct; `symbol` is the new qualified `Type.Field` target and `source` supplies exactly that field. Placement defaults to `type_end`; relative placement needs `target_symbol`. The engine validates collisions and the advertised layout and literal constraints. Pass the generated field's live `file` and qualified `symbol` to `semantic_rename`, `semantic_inspect_symbol`, or `semantic_encapsulate_field` when available. Repeating this insertion requires a different fresh field target.

**Parameter description requirements:**

`in_type`: Selector: proposed `semantic_inspect_symbol.in_type` once that exact owner projection is implemented. `symbol`: Target: new qualified field identity; prefer an engine-provided same-shaped proposal and require agreement with `source` and `in_type`. No existing field continuation supplies a fresh new field identity. `source`: Replacement: one field declaration, not the full struct. `target_symbol`: Selector: existing qualified field anchor; copy only proposed `semantic_inspect_symbol.target_symbol` after the same-named qualified-field anchor projection is implemented; otherwise no automatic copy edge is available. `MemberTarget.symbol` does not feed `target_symbol` by renaming.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
