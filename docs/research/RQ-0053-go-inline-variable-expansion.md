<!-- This handover exists to expand inline-variable support while preserving binding, evaluation count, and dispatch semantics. -->
# RQ-0053: Go Inline Variable Expansion

* **Status**: Open
* **Category**: Semantics & Refactoring
* **Date**: 2026-09-30

## Context

ADR-0058 exposes a conservative `semantic_inline_variable` for a literal local used as a direct call argument or direct return value in the immediately following statement. It refuses expressions and call inlining. Broader inline-variable support is attractive, but substituting an initializer at use sites can duplicate evaluation, observe a different value, alter scope, or select another binding.

## Research questions

1. Which additional initializer forms can be substituted while preserving evaluation count and order?
2. Can multiple use sites be handled safely, and under what single-execution/control-flow proof?
3. Which expression contexts can receive a typed replacement without changing dispatch or method receiver behavior?
4. Should call inlining remain a separate operation and contract?
5. How can an earlier buffered edit safely feed a later inline request with stable binding identity?

## Current evidence and limits

* `internal/astedit/inline.go` contains current inline-variable analysis and mutation; `internal/operation/wire_engine_inline.go` defines the Go operation and schema.
* ADR-0058 accepts one local literal declaration with no explicit type and uses in the immediately following statement in the same lexical block. It inserts default-type conversions for untyped literals and refuses additional references, comments, intervening statements, ambiguous binding, or uncertain import names.
* No inline-function/call expansion is implemented by this tool. A generic `semantic_inline` would obscure materially different proof obligations.
* The authoritative pending main uses `sourcebuf` for current source and buffered execution. The older root branch is not a safe implementation base.

## Candidate outcomes and non-goals

Investigate in order: multiple direct uses in one next statement; direct uses later in the same block with no intervening writes or control-flow boundary; side-effect-free composite expressions with type facts. Each item should be accepted only with a binding and evaluation proof. Inline function/call should be a separate RQ or ADR with call graph, dispatch, receiver, and control-flow analysis.

Do not duplicate an initializer expression that may call, receive, mutate, panic, allocate identity-sensitive objects, or depend on a variable that changes between declaration and use. Do not inline across closures, defer/go, loops, branches, labels, or statement boundaries until exact execution counts are proven. Do not treat successful compilation as behavioral proof.

## Proposed API questions

Keep `semantic_inline_variable` / `inline-variable`, with primitive inputs:

* `file` string: workspace-relative Go file.
* `symbol` string: local target name; `in_function` selects its owner.
* `expected_revision` string: candidate binding/source revision token for stale requests.
* Existing `in_function` format remains fully-qualified and is a selector, not the mutation target.
* Require `file`, `symbol`, and `in_function`; no defaults. `expected_revision` is an optional opaque string on the first request and required for a follow-up based on a prior receipt. If one declaration has multiple same-name uses, determine whether the operation always transforms all proven uses or accepts a zero-based `construct_path` use selector; any path requires the matching `expected_revision`. Avoid a hidden default that changes a subset.

If a future call-inlining operation is considered, define it separately and avoid reusing variable-specific parameters for function targets. Every value copied from a receipt needs `SourceFields` plus exact schema-source text.

## Continuation receipt proposal

After success, the removed local no longer exists as a live `symbol`. Do not return it as a reusable target. Return `file`, `in_function`, changed files, and a `removed_binding` evidence object with original name/location and transformed use count. If the function owner remains live, return it verbatim. Include `expected_revision` under that exact name, paired with any returned zero-based `construct_path`; bind it to the selected file revision.

```json
{
  "file": "pkg/serve.go",
  "in_function": "Server.Serve",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/serve.go"],
  "removed_binding": {"name": "limit", "uses_rewritten": 2},
  "diagnostic_status": "unchanged"
}
```

`removed_binding.name` is historical evidence and must not be advertised as a follow-up `symbol`.

## Semantic hazards and refusal rules

* Substitution can duplicate evaluation for multiple use sites, changing side effects, panics, allocation identity, and timing.
* A write to the binding or an alias-visible value between declaration and use makes the initializer stale.
* Shadowed declarations and short declarations require identity-aware resolution; function-wide spelling matches can over-refuse or under-refuse.
* Conversion insertion can change constant representability, rounding, overflow behavior, typed nil behavior, or generic inference.
* A method call expression substituted as an argument can change receiver evaluation order or copy a value.
* Comments attached to the declaration or any use cannot be discarded or relocated by approximation.
* Stale selectors after a previous buffered edit must be checked against overlay bytes and revision evidence.

## Ordered milestones

1. Enumerate candidate expression classes and build a semantic table of evaluation multiplicity, type preservation, and control-flow constraints.
2. Choose one class and prove it with resolved object identity and type facts; keep a fail-closed boundary for all others.
3. Decide whether all uses are atomic target or whether use-site selection belongs in a separate operation.
4. Implement overlay reads and edits; validate the full candidate source, then defer normalization to batch finalization and preserve the configured diagnostic scope.
5. Register Go-only capability and update receipt cross-references with no inline-function alias.

## Public behavior tests to add

* CLI txtar success for multiple safe direct uses with preserved single initializer evaluation; assert exact final tree and `uses_rewritten` receipt evidence.
* CLI txtar refusal for a call initializer used twice; assert no duplicate side effect and unchanged source.
* CLI txtar refusal for a reassigned/shadowed binding and for a declaration comment whose ownership cannot be preserved.
* CLI txtar success/refusal at the newly accepted expression context; assert exact inferred type and no unsupported call-inlining claim.
* MCP mixed batch: statically specify a body edit, inline a uniquely selected binding from the known unsaved result, then make a later edit. Do not interpolate a receipt in the batch; use only fixture-known names, and pair any zero-based path with its exact revision. Run a separate follow-up mutation using a live receipt target, then a standalone lookup to prove freshness.
* MCP batch with one successful edit followed by an unsafe inline refusal stops at that refusal; the earlier mutation remains eligible for publication. Any repair or later success is a separate invocation.

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

**Tool description for `semantic_inline_variable`:**

> Use this tool instead of replacing variable uses by text to inline one supported local Go binding while preserving value, evaluation count, and ordering. `symbol` targets the local and `in_function` selects its owner. The successful operation removes the binding. Pass the live `file` and `in_function` to `semantic_replace_construct` with a fresh supported construct selection. The removed binding name is historical evidence and must never be copied into another symbol-target call. Wider initializer and use forms are advertised only after their proof cases pass.

**Parameter description requirements:**

`symbol`: Target: existing local binding in the selected owner; copy only a proven compatible live local identity. Remove `semantic_inline_variable.symbol` from sources after successful deletion. `in_function`: Selector: copy the live `semantic_inline_variable.in_function` or compatible owner projection. Its file-scoped revision belongs to that owner context, not the removed binding.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
