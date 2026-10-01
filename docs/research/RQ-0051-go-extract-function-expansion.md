<!-- Holds the resumable implementation handover for broader go extract function and method, including semantic proof and public batch contracts. -->
# RQ-0051: Broader Go Extract Function and Method

* **Status**: Open
* **Category**: Go Refactoring & Implementation Handover
* **Date**: 2026-09-30

## Outcome and current evidence

Extend existing `semantic_extract_function` / `extract-function` to routine helper extraction with real local data flow, supported nested control flow, multiple results and attached comments. Do not create another extract-method API: existing `kind=function|method` selects that behavior.

Extraction is ALREADY implemented on main by `cd40dea` and recorded in ADR-0056. It supports contiguous direct calls, optionally one leading declaration with distinct named live-out results. The reviewed bounded extension accepts result types only from basic literals, boolean literals, named owner parameters, or supported unary/parenthesized forms. Calls and compound/unsupported result expressions refuse before writing; type identities are syntax-derived and complete compiler typechecking/read-set preflight remain open. Pending main changes integrate the same bounded operation with sourcebuf; they do not remove its other semantic restrictions.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this RQ records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

## Bounded delivery and non-goals

Milestone 1: straight-line assignments/declarations and expressions, compiler-derived live-ins/live-outs, multiple supported result values, and comment preservation. Milestone 2: self-contained loops and branches whose exits remain inside the extracted region. Milestone 3: expression/nested-block selection, generic/variadic/named-result owners only after dedicated proofs.

Refuse outward returns, break/continue/goto, direct recover, defer/go lifetime changes, receiver-copy/alias changes, captured/address-escaping state, ambiguous comment ownership and unknown types until supported. No arbitrary full-function rewriting by the model.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

## Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Selector: existing owner source file | Required |
| `symbol` | Target: new helper simple identifier | Required |
| `in_function` | Selector: existing qualified owner | Required |
| `kind` | Existing `function` or `method` helper kind | Required; no implicit default |
| `start_line`, `end_line` | Existing inclusive direct-statement boundary selectors | Retain current required behavior |
| `discriminator`, `construct_path` | Candidate revision-bound structural range selectors | Optional future alternative; require shared selection gate |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_extract_function",
  "params": {
    "file": "service.go",
    "symbol": "prepareRequest",
    "in_function": "(*Service).Serve",
    "kind": "method",
    "start_line": 14,
    "end_line": 19
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "service.go",
  "symbol": "(*Service).prepareRequest",
  "in_function": "(*Service).Serve",
  "kind": "method",
  "continuations": [
    {
      "file": "service.go",
      "symbol": "(*Service).prepareRequest"
    }
  ]
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

## Implementation sequence

1. Preserve every existing success/refusal txtar and identify the exact restrictions being relaxed; do not relabel the operation as new.
2. Inspect gopls extraction actions under the retained session on representative regions. Accept their output only if helper naming, selection, comment ownership, evaluation and refusal contracts match. Command/edit capture is a prerequisite if needed.
3. Build compiler-based variable identity and CFG live-in/live-out analysis over the current overlay. Distinguish local mutation, address identity, closures, declarations and result ownership.
4. Plan helper parameters/results, receiver policy and caller substitutions together. Avoid copying mutable value receivers or changing address identity; preserve shadowing and evaluation order.
5. Preserve comments by explicit ownership mapping and choose result types from resolved compiler information. Validate combined output with types and behavior fixtures before Apply.
6. Keep old line selectors tied to the current working revision; add structural selectors only after their shared introspection and revision contract is accepted. Return actual qualified helper identities for method extraction.
7. Expand control flow in a separate milestone after straight-line proof gates pass.

## Required batch behavior

Batch: rename the owner or insert local assignments without publication → extract from the current working statement range → rename the resulting helper → replace its body. An earlier insertion shifts line coordinates: resolve the new range or reject an old expected_revision rather than selecting different statements. A later mutation resolves the new helper and revised owner. A refused extraction publishes only earlier successful edits at batch completion.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

## Public acceptance and refusal coverage

* Multiple live-ins and two live-outs with intervening assignments: runtime values and mutation counts unchanged; exact generated helper signature checked.
* Parameter/receiver shadowing, local package-name aliases, supported inferred custom result types and attached comments preserve bindings/source ownership.
* Entire self-contained loop/branch extraction preserves iteration and panic behavior; exits crossing the region boundary refuse before edits.
* Address-taken local/receiver, captured closure, defer/recover/go, named results and generics remain explicit refusal cases until the corresponding milestone.
* Method helper receipt feeds rename and replace_body using the returned qualified `symbol` without synthesizing a receiver name.
* Preserve existing ADR-0056 cases and document only the restrictions actually removed.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

## Semantic risks and unresolved decisions

* Typechecking does not prove extraction preserves control flow, receiver identity, address escape, captures, panic/recover or defer timing.
* Named result variables participate in deferred closures. Moving their assignments into a helper can change when/what a defer observes.
* Range variables and loop-captured closures require Go-version-aware semantics, not syntax-only substitution.
* Native extraction actions may lose comments or choose names/types contrary to the accepted operation contract. A candidate action is not automatic authorization to weaken ADR-0056.

## Decision gates and completion

Prove straight-line data flow with a compiler-backed liveness model, explicit receiver/alias rules and runtime fixtures before relaxing existing refusals. Agree how new structural selectors coexist with current line selection without silently changing meaning. Update ADR-0056 and its index only for accepted extensions.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this RQ Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

## Dependencies, ownership, and entry points

Depends on live-target/revision-bound receipt foundations, typed source snapshots, and [change signature](RQ-0041-go-change-signature.md) parameter/type planning where reusable. Can run in parallel with inline function after agreeing shared evaluation/alias proof helpers. Parent owns registry/capability/index integration.

* Extend `internal/astedit/extract.go`, `internal/operation/wire_extract.go`, and `internal/backend/golang/backend.go`.
* Read [ADR-0056](../adr/0056-go-call-statement-extraction.md) and existing `testdata/scripts` extraction cases.
* Use pending `sourcebuf`/session overlays; investigate [gopls extraction actions](https://go.dev/gopls/features/transformation) without invoking a subprocess.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_function`:**

> Use this tool instead of copying statements to extract a supported contiguous Go statement range into a helper with inferred parameter and result flow. Choose a fresh simple `symbol`; `in_function` selects the existing owner and required `kind` chooses function or method. The selected `start_line` and `end_line` must match current direct-statement boundaries. Pass the created helper's qualified result `symbol` and matching `file` to `semantic_rename` or `semantic_replace_body`; pass the live `in_function` owner to construct tools. Qualified helper results do not feed this tool's new simple-name input.

**Parameter description requirements:**

`symbol`: Target: new simple helper identifier chosen by the caller. Remove qualified existing-symbol sources from this creation input. `in_function`: Selector: copy `semantic_extract_function.in_function` or another validated live same-named owner projection. `start_line` and `end_line`: Selectors: current inclusive statement boundaries; `semantic_lookup.line` and `column` are navigation hints, not these parameters. Descriptive lookup `kind` does not feed required helper `kind`.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
