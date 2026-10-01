<!-- This handover exists to define focused Go doc-comment editing while protecting compiler directives and comment ownership. -->
# RQ-0059: Go Set Doc Comment

* **Status**: Open
* **Category**: Structural Editing & Documentation
* **Date**: 2026-09-30

## Context

Tool receipts and source declarations can be structurally correct while lacking or carrying stale documentation comments. `docs/MANUAL_EDITS.md` ME-0047 records attached lint comments as a manual edit. Whole declaration/body replacement is disproportionate for comment-only changes. Go comments may be doc comments, ordinary comments, build constraints, compiler directives, cgo preambles, or comments attached to AST nodes; treating all as interchangeable risks changing build behavior.

## Research questions

1. Which targets are supported: package/file header, top-level declaration, method/function, type, field, or parameter?
2. What exact comment string format should callers pass, including `//` or `/* */` delimiters and trailing newline policy?
3. How can the tool distinguish documentation text from `//go:`, `// +build`, `//line`, `//export`, cgo, and lint directives?
4. Should an empty string remove only a target's doc comment, and how are detached ordinary comments preserved?
5. How are a file-level comment and a declaration-targeted comment represented without overloading `symbol`?

## Current evidence and scope

* `docs/MANUAL_EDITS.md` ME-0047 identifies attached lint/doc comment changes as outside current focused semantic tools and suggests `semantic_set_doc_comment`.
* Go AST parsers retain comments but attachment and source positions vary; arbitrary text replacement can move comments across declarations or alter build/compiler semantics.
* Existing insertion tools accept declaration source containing comments; they do not provide a target-specific comment setter.
* The pending authoritative main has sourcebuf overlays. Comment updates must compose with earlier uncommitted declaration/move edits and preserve the retained gopls session.

## Outcomes and non-goals

First release should set or remove only a declaration's Go doc comment, or choose a file/package comment as a separately specified target. Keep comment text literal and preserve exact source line endings where possible. Refuse any target region containing directives or ambiguous attached comments. Do not edit arbitrary inline comments, lint suppressions, build tags, cgo preambles, license banners, or comments inside function bodies in the first release.

## Proposed API

Propose `semantic_set_doc_comment` / `set-doc-comment`, Go only, batchable and buffered.

* `file` string: exact workspace-relative Go file.
* `symbol` string: target declaration for declaration scope; use canonical qualified symbol format.
* `scope` enum: proposed `declaration` or `file`; decide if a dedicated file-header mode is safe and discoverable.
* `comment` string: plain text body with no `//`, `/*`, or `*/` delimiters. The engine emits `//` doc-comment lines and handles line breaks consistently. Explicit empty string removes the selected doc comment; missing or null is invalid and never means remove.
* `in_function` string: only if the first release supports nested/local declarations; recommendation is to exclude them initially.
* `expected_revision` string: stale target/comment attachment guard.

`file`, `scope`, and `comment` are required. `scope` has exact values `declaration` and (if accepted) `file`, with no default because the target mode must be explicit. `symbol` is required for declaration scope and forbidden for file scope. Missing `comment` is invalid; explicit empty string means remove the selected doc comment. Since current `CheckParams` rejects empty required strings and CLI `rawParams` omits blank flags, research a presence-aware CLI flag/raw-parameter path and an allow-empty string contract for this parameter so `--comment=''` reaches the handler. MCP must preserve absent, null, and empty as distinct states. `expected_revision` is optional initially and required for receipt-based follow-up. No import option applies.

A file scope has no declaration `symbol`; schema rules must express that without treating a missing symbol as success. `SourceFields` should connect `file` and `symbol` to matching lookup/edit receipts. If a structured comment representation is desired, defer until recursive schema support.

## Continuation receipt proposal

For declaration scope, return `file` and live `symbol`; for file-header scope, return `file` and `scope` only. Return the exact plaintext `comment` body so it can be copied directly into a later call; the model never has to reconstruct comment delimiters. A successful removal has empty current comment, but removed text is historical evidence.

```json
{
  "file": "pkg/config.go",
  "symbol": "Config",
  "scope": "declaration",
  "comment": "Configuration for the service.",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/config.go"],
  "diagnostic_status": "unchanged"
}
```

If a file scope is accepted, omit `symbol` and return `scope: file`. Never return a file header as a declaration symbol.

## Semantic hazards and invariants

* Build tags and `//go:` directives affect compilation/linking and must never be inserted, removed, or moved by doc-comment operations.
* `//line`, `//export`, `//go:linkname`, `//go:generate`, and cgo preamble adjacency need explicit recognition/refusal policy.
* A comment may attach to a grouped declaration, a field, a method, or the next declaration; preserve all non-target comments byte-for-byte.
* Empty `comment` is an explicit removal request; absent/null input is invalid, preserving distinction between removal and missing data.
* Multiple matching symbols, generated files, malformed source, detached comment clusters, and stale revision refuse before mutation.
* Documentation text can contain directive-looking lines; reject directive prefixes conservatively rather than normalizing or stripping them.
* Formatting should not rewrite unrelated comments. Validate the resulting AST and publish through the overlay only.

## Ordered milestones

1. Decide declaration-only versus file-header support and define comment input encoding/newline semantics.
2. Build a directive/comment ownership matrix from Go syntax and repository fixtures.
3. Specify empty versus missing behavior, stale revision, and exact receipts under ADR-0055.
4. Implement node selection and comment-range edits with byte-preserving unrelated source, buffer-aware writes, and refusal before apply.
5. Add CLI/MCP registry, schemas, Go capability, and literal CLI txtar contracts; consider file headers only in a follow-up milestone.

## Public behavior tests to add

* CLI txtar success sets a declaration doc comment and replaces an existing one; assert exact text, adjacent comments, and receipt.
* CLI txtar `--comment=''` explicitly removes the doc comment; an omitted flag refuses unchanged. MCP cases separately assert omitted `comment`, `comment: null`, and `comment: ""`; only the explicit empty string removes the comment. Malformed, ambiguous, or stale targets refuse unchanged in both interfaces.
* CLI txtar refuses targets adjacent to build tags, `//go:linkname`, `//export`, cgo preambles, and detached ordinary comments; assert directive/comment bytes remain exactly unchanged.
* CLI txtar test file-level header separately if accepted; cover package docs and license comment boundary without moving build constraints.
* MCP batch: statically specify insertion of a declaration, setting its doc comment by the known `symbol` in that unsaved result, and a later edit. No receipt is interpolated inside the batch; use fixture-derived selector and revision literals. A separate follow-up mutation consumes the declaration `symbol`, then a standalone lookup proves the target remains live.
* MCP batch with a successful declaration edit followed by comment-selection refusal stops at refusal; the earlier edit remains eligible for publication, the wrong node stays untouched, and no intermediate formatting is published. Later edits run separately.

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

**Tool description for `semantic_set_doc_comment`:**

> Use this tool instead of replacing source declarations to set or remove an owned Go documentation comment. `scope=declaration` requires its live `symbol`; file scope, if supported, forbids `symbol`. Supply plain `comment` text without delimiters. An explicit empty string removes the comment; missing or null is invalid. The engine preserves directives or refuses ambiguous ownership. Pass live `file`, `scope`, declaration `symbol` when applicable, and current `comment` directly to another `semantic_set_doc_comment` request. Pass a declaration target to `semantic_inspect_symbol` or `semantic_rename`; removed comment text is evidence.

**Parameter description requirements:**

`symbol`: Target: live declaration copied from compatible `semantic_lookup.symbol` with its `file`; forbidden in file mode. `scope`: Selector: required explicit supported target mode, copied from `semantic_set_doc_comment.scope` when repeating that mode. `comment`: Replacement: caller text or current `semantic_set_doc_comment.comment`; empty is a present value, not omission. Use only the current result, never historical removed text, as its automatic source.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
