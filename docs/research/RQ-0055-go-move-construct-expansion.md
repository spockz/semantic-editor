<!-- This handover exists to expand the bounded Go move operation only where source identity, references, and multi-file publication can be proven. -->
# RQ-0055: Go Move Construct Expansion

* **Status**: Open
* **Category**: Semantics & Refactoring
* **Date**: 2026-09-30

## Context

Same-file and same-package movement for all currently supported declaration kinds is already implemented; cross-package movement is tightly bounded to an exported free function. Cross-package relocation must preserve comments, imports, references, package bindings, and accessibility. The authoritative pending main already contains buffered batch and shared-session migration work; expansion must preserve that lifecycle.

## Research questions

1. Which declaration kinds and scopes can be moved without changing identity, visibility, initialization, or method sets?
2. What reference and package loading evidence is complete enough for cross-package moves, including inactive variants, generated/test files, dot imports, and overlays?
3. How should destination creation/resource semantics interact with current existing-file-only `Buffer.Apply`?
4. How are same-file, same-package, and cross-package receipts kept copyable under ADR-0055?
5. Which source-set and per-file publication failures can be reported without claiming a global rollback?

## Current evidence and limits

* `internal/astedit/move_construct.go` contains same-file/package movement and dispatch; `internal/astedit/move_cross_package.go` contains bounded package movement and reference rewrite/type-checking.
* `internal/operation/wire_engine_mutations.go` defines `semantic_move_construct` / `move-construct`; it is workspace-scoped and batch-buffered on the authoritative pending main. `destination_file` is optional and defaults to `file` for existing same-file/same-package moves.
* Cross-package behavior currently supports an exported free function in one module and refuses methods, unexported symbols, source-package dependencies, import cycles, ambiguous build/generated files, and unprovable references. Verify the final merged implementation before expanding this snapshot-specific list.
* Commit `50b80b2` contains cross-package reference-binding hardening; the authoritative `6c95d12` source already includes its relevant reference checks although that commit object is not an ancestor.
* `sourcebuf.Buffer.Apply` currently supports existing-file changes. A destination file/package move requiring a new file needs a source-buffer resource contract or an explicit scaffold prerequisite.

## Candidate outcomes and non-goals

Do not reimplement same-file or same-package movement; it already covers all supported declarations. The first new milestone is a bounded cross-package dependency group consisting of a named type plus its directly associated methods, with dependency/reference completeness and type identity proved as one move. Keep unrelated declarations, new-file creation, files/module moves, and cross-module moves out of scope until resource and project-boundary contracts exist.

Do not advertise arbitrary syntax-node movement, package renaming, arbitrary unrelated multi-symbol moves, copy semantics, cross-module moves, or behavior-preserving API migrations. Moving a declaration is distinct from rewriting its callers or changing its signature.

## Proposed input and continuation contract

Keep `semantic_move_construct` / `move-construct`. `file` is the exact source file; `symbol` is the moved declaration; `destination_file` is the sole destination field and is optional, defaulting to `file` for same-file/same-package moves; `placement` and `target_symbol` control placement in the destination file where supported. Do not add a second `destination` or destination package string. Infer destination package only from a validated target file/sibling context.

`file` and `symbol` are required; `destination_file` is optional and defaults to `file` for existing same-file/same-package moves, but is required when selecting a cross-package destination. `placement` is an optional enum using the existing placement values; absent placement uses the current engine’s established default, which must be confirmed in the rebase audit. `target_symbol` is optional unless placement is `before_symbol` or `after_symbol`. `auto_organize_imports` is Boolean default `true`; `expected_revision` is an optional primitive token for the selected source `file` revision and required for receipt-based follow-up. The complete move read set, including destination and dependency files, is captured separately as `read_set_revision` historical evidence; the output `expected_revision` binds the live destination file.

Cross-package reference changes must be typed, identity-checked, and validated against a complete overlay. If a structured relocation plan is proposed, first deliver recursive schema support; current `ParameterContract` accepts primitives. Define complete read-set evidence and apply-time validation separately from the selected-file `expected_revision`; do not overload that token to stand for every dependency file.

## Continuation receipt proposal

After success, top-level `file` is the destination and top-level `symbol` is the destination-file symbol name. Return `destination_file` as the destination too, and put the historical source under `moved_from`, including original source file and symbol plus requested placement/anchor as history metadata. Keep attached comments with the construct. Return a `continuations` array with a `SymbolTarget` entry containing `file`, the destination-file `symbol`, and its `expected_revision`. Do not echo stale placement or target anchors as live continuations; changed-file and diagnostic details remain separate receipt data.

```json
{
  "file": "api/service.go",
  "destination_file": "api/service.go",
  "symbol": "Service",
  "expected_revision": "opaque-destination-revision",
  "read_set_revision": "sha256:historical-move-read-set",
  "moved_from": {"file": "legacy/service.go", "symbol": "Service", "placement": "after_symbol", "target_symbol": "Other"},
  "continuations": [{"file": "api/service.go", "symbol": "Service", "expected_revision": "opaque-destination-revision"}],
  "changed_files": ["legacy/service.go", "api/service.go"],
  "diagnostic_status": "unchanged"
}
```

The cross-package dependency-group example assumes the destination file already exists. Input `expected_revision` binds only the selected source file revision; `read_set_revision` records the complete source, destination, and dependency read set as historical evidence. The output `expected_revision` binds the destination file. For same-package moves, omitted `destination_file` defaults to `file`. Only the `SymbolTarget` with destination file, file-local symbol, and output revision is live; `moved_from` placement and anchor fields are historical metadata.

## Semantic hazards and refusal conditions

* Moving vars can change package initialization order and dependency cycles; require an explicit initialization-order proof or refuse.
* Moving methods changes method sets when receiver types move or are aliased; keep receiver/type scope fixed in the first expansion.
* Type and constant moves can break unexported references or iota values and can change API visibility.
* Comments, build tags, doc comments, compiler directives, and grouped declarations need explicit ownership rules.
* Cross-package import rewrites can resolve to a shadowing package/local or collide with aliases; validate object identity in the typechecked overlay.
* Active package loads miss inactive build variants; conservative source-set scan or explicit refusal is required.
* Earlier buffered edits may change the declaration or references; preflight and rewrite from the current overlay, not disk.
* Per-file atomic writes do not give whole-move rollback; preflight every affected output before the publication phase and report partial failure precisely.

## Ordered milestones

1. Re-audit supported kinds, source-set enumeration, existing-file constraints, and receipt after integration of pending main.
2. Start with a cross-package type plus its directly associated methods; enumerate identity, dependency-group closure, reference completeness, visibility, and method-set conditions.
3. Decide whether new-file/package destinations need sourcebuf resource support or remain refused.
4. Implement overlay planning, complete affected-file typechecking, identity validation, atomic per-file publication, and partial-failure reporting.
5. Add Go-only capability/schema/CLI tests; expand ADR-0049 only after proof gates pass.

## Public behavior tests to add

* CLI txtar cases preserve same-file/same-package behavior for existing declaration kinds, then add a cross-package type-plus-method dependency-group success; assert exact source/destination trees, comments, imports, and destination `SymbolTarget` receipt.
* CLI txtar refusal for initialization-order risk, grouped declaration/iota effects, method receiver mismatch, inaccessible destination, and unparseable affected file; assert all preflight refusal cases leave files unchanged.
* CLI txtar cross-package reference cases for active and inactive build variants, dot-import/alias collision, untyped reference, and shadowing destination names; assert no wrong-binding rewrite.
* MCP mixed batch: statically specify an edit to a source type, then move the known type-plus-method dependency group into an existing destination package, then apply a later statically targeted edit in the destination. Use fixture-derived revisions for the exact intermediate overlay; no batch interpolation is available. Run a separate follow-up mutation using the returned destination-qualified `SymbolTarget`, then a standalone lookup to prove it remains live.
* MCP batch with an earlier successful unrelated edit and a refused move stops at refusal; the earlier edit remains eligible for publication and move outputs stay unchanged. Any repair/later success is a separate invocation.
* Case destination creation separately: current `Buffer.Apply` cannot create files; verify explicit refusal unless sourcebuf resource support or scaffold prerequisite is part of the accepted design.

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

**Tool description for `semantic_move_construct`:**

> Use this tool instead of cutting declarations and repairing references by text to move a supported Go declaration to an existing `destination_file`. The destination defaults to `file`; cross-package moves use the same retained session and batch overlay within the supported module boundary. The live result `file` is the declaration's destination. Pass that `file`, qualified `symbol`, and matching accepted `expected_revision` directly to `semantic_rename` or `semantic_replace_body` for callables. Moved-from paths and read-set evidence are historical data. Broader type, dependency-group, or resource moves require their separate support gates.

**Parameter description requirements:**

`file`: Source selector: current file of the live target; after a move copy `semantic_move_construct.file`, which must name the destination. `symbol`: Target: exact moved declaration identity copied from live `semantic_lookup.symbol`, `semantic_rename.symbol`, or `semantic_move_construct.symbol`, with that same record's `file`. `destination_file`: Destination selector: caller-selected existing path; reusing a destination receipt means explicitly choosing another move destination, not locating the current target. `read_set_revision` is not `expected_revision`.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
