<!-- This handover exists to expand safe deletion beyond its accepted constant-only scope without turning structural deletion into an unproven behavior claim. -->
# RQ-0054: Go Safe Delete Expansion

* **Status**: Open
* **Category**: Semantics & Safety
* **Date**: 2026-09-30

## Context

ADR-0057 defines `safe_delete` on `semantic_remove_construct`; only an unexported standalone constant is currently accepted after a conservative module source scan. Ordinary removal is structural and does not prove behavior. The pending main has source-buffer batching for ordinary edits, while safe-delete preflight still needs a complete overlay/read-set design so it does not create a disk-only barrier or inspect stale bytes.

## Research questions

1. Which additional declaration kinds can be safely deleted and what exact reference, directive, resource, or language-specific checks establish the guarantee?
2. How can the module source scan include unsaved buffered sources and prove the scanned file set and contents remain stable through `Buffer.Apply`?
3. What does a complete Go source set mean across build tags, tests, generated files, vendor and ignored directories, symlinks, and nested modules?
4. How should refusal checks, scan coverage, and overlay revisions be serialized without implying a stronger guarantee than was established?
5. Can safe deletion become a normal buffered operation with no intermediate flush, barrier, or silent downgrade?

## Current evidence and limits

* `internal/astedit/remove.go` handles exact structural removal and safe-delete preflight; `internal/operation/wire_remove.go` registers `semantic_remove_construct` / `remove-construct`, `safe_delete`, checks, and current Go-only scope.
* ADR-0057 accepts safe deletion only for unexported standalone constants; grouped constants, exported constants, all other node kinds, uncertain source sets, directives, and any same-name module identifier are refused.
* Normal removal already reads through `sourcebuf`; current safe preflight must be refactored to read its complete source set through the same overlay and return a read-set proof.
* `internal/sourcebuf/buffer.go` `Buffer.Apply` atomically validates the complete output set against source revisions for existing files; `internal/mcp/buffers.go` finalizes buffered edits, normalizes changed sources, and retains configured diagnostic scope and coverage.
* The pending main retains a gopls session. Do not introduce a safe-delete barrier that flushes changes to disk or restarts/resynchronizes that session.

## Candidate outcomes and non-goals

Evaluate separate classes: unreferenced private free functions/types/variables; package-private methods; exported API; resource-bearing declarations; grouped declarations with iota; cases/statements. A class is safe-delete-supported only if the engine can explain and verify its complete relevant reference set and collateral effects. For many classes the answer should remain refusal.

No safe-delete success may be based solely on syntax validity, active-build references, text grep without stated completeness, or compiler diagnostics. Do not silently execute ordinary removal when `safe_delete=true`. No claim of semantic preservation for case/statement removal or broad program equivalence.

## Proposed operation contract

Keep one `semantic_remove_construct` / `remove-construct`; `file` and `kind` are required. `symbol` is required for declaration kinds; `in_function` is required for cases; `discriminator`, `target_case`, and `construct_path` are optional selectors only where applicable. `safe_delete` is Boolean default `false`; `auto_organize_imports` is Boolean default `true`. `expected_revision` is an optional opaque token on a first request and required for follow-up; it represents the selected file revision only, never the complete source set. Preserve ADR-0057 selector roles. Proposed output adds `safety_status`, `checks_run`, `source_set_revision`, and a stable list of affected paths. Decide which are public schema fields versus structured refusal payload fields before implementation. Refusal must say the guarantee could not be established and include checks that completed.

`expected_revision` is a primitive selected-file revision token with the exact same input/output name: optional on an initial request and required on receipt-based follow-up. Keep the full scan read-set digest separately as `source_set_revision` historical evidence; it is not a retry token and must not be conflated with `expected_revision`. The complete scan must observe overlay bytes for all existing source files, track every file read and metadata identity, and pass that read set to apply-time validation. If the current `Buffer.Apply` API does not accept read-only dependencies, research a read-set extension instead of weakening the proof. Scan-set changes, unreadable/unparseable files, symlinks or unsupported source boundaries refuse. No direct disk scan may ignore an unsaved identifier occurrence.

## Continuation receipt proposal

Deleted `symbol` is not a live target and must not be returned as a reusable continuation. Return the containing file and safety evidence only; for case deletion, the owner `in_function` remains live. Keep the removed target name, old expression, and old `construct_path` under `removed` evidence, not at top level as input-shaped fields.

```json
{
  "file": "pkg/config.go",
  "safety_status": "refused",
  "expected_revision": "opaque-selected-file-revision",
  "checks_run": ["overlay_source_set", "parse_all_go_files", "identifier_scan"],
  "source_set_revision": "sha256:historical-read-set",
  "removed": {"symbol": "legacyLimit", "kind": "const"},
  "changed_files": [],
  "diagnostic_status": "unchanged"
}
```

The selected-file token in `expected_revision` is a retry input; `source_set_revision` is historical complete-read-set evidence only. The digest format and whether refusal can be a normal structured result are unresolved. Do not invent a success receipt for this refusal example; MCP `isError` and structured error shape must match the established contract.

## Safety and consistency hazards

* Build tags and test variants: a reference in an inactive variant still matters for deletion; active package loading alone is incomplete.
* Generated code, linkname, cgo, reflection, string-based registries, and external callers may defeat name-based reference proof.
* Same spelling is conservative evidence, not binding identity; over-refusal is acceptable, false-safe success is not.
* Grouped const deletion can renumber iota siblings. Multi-name specs can bind shared RHS expressions.
* Nested modules and symlinks can make module boundaries ambiguous; fail closed until source-set semantics are documented.
* A buffer overlay can add or remove a reference after the initial scan. Both bytes and path set must be revalidated before publication.
* `Buffer.Apply` validates and applies all outputs atomically to the working overlay. `Buffer.Flush` provides per-file atomic disk publication without a global rollback guarantee; read-set validation must occur before publication begins.

## Ordered milestones

1. Define the source universe and overlay read-set proof with explicit ignored-directory/nested-module rules.
2. Identify the smallest next target class and enumerate unsupported dynamic-reference channels.
3. Design additive `Buffer` read-set validation; confirm no direct writes, flush, or barrier are needed.
4. Implement false-safe-resistant refusal behavior, checks-run receipts, configured-scope diagnostics, and CLI/MCP schemas.
5. After evidence, revise ADR-0057; leave unsupported classes unadvertised.

## Public behavior tests to add

* CLI txtar safe success for currently supported standalone private constant and for any newly supported class; assert file tree, checks, and exact surviving declarations.
* CLI txtar refusal for grouped/iota constants, exported symbols, resource/directive cases, incomplete source set, and another identifier in an inactive build-tag file; assert target bytes unchanged.
* CLI txtar overlay refusal: an earlier unsaved edit inserts a reference to the target, then `safe_delete=true` must see it and refuse. The batch stops; the earlier edit remains eligible for publication. A repair and later successful mutation are separate invocations; use a still-live target from the eligible earlier mutation, then verify it with a standalone lookup.
* CLI txtar refusal when an earlier buffered removal changes the scan source set or when the read-set revision changes before apply; no safe downgrade.
* MCP batch with a successful edit followed by safe-delete refusal proves the refusal sees the overlay and does not roll back the earlier mutation; the batch stops there. Run any repair and later successful edit in a separate invocation; consume a live surviving symbol, then verify it with a standalone lookup.
* Receipt/schema cases distinguish ordinary structural remove from `safe_delete=true`; diagnostics must not call an ordinary removal safe.

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

**Tool description for `semantic_remove_construct`:**

> Use this tool instead of deleting source blocks by text to remove one supported Go declaration or case. `symbol` targets a declaration; cases use `in_function` and current case selectors. `safe_delete` defaults to false; true requires complete supported reference and resource proof or refuses. A removed target and its old selectors are historical evidence, never continuation arguments. Pass a still-live `file` to `semantic_organize_imports` or a live case owner's `file` and `in_function` to `semantic_replace_construct` with a fresh selection. Structural deletion alone does not prove behavioral equivalence.

**Parameter description requirements:**

`symbol`: Target: existing declaration; copy a compatible live `semantic_lookup.symbol`, never the deleted `semantic_remove_construct.symbol`. `target_case`, `discriminator`, and `construct_path`: Selectors: current case/switch projections only; removed case paths cannot supply the next deletion. `expected_revision`: Precondition: selected-file token. `source_set_revision`: Evidence: full scan proof, not a consuming revision parameter.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
