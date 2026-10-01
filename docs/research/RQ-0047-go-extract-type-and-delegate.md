<!-- Holds the resumable implementation handover for go extract type and delegate, including semantic proof and public batch contracts. -->
# RQ-0047: Go Extract Type and Delegate

* **Status**: Open
* **Category**: Go Refactoring & Implementation Handover
* **Date**: 2026-09-30

## Outcome and current evidence

Separate selected state and behavior into a new Go type without requiring the caller to reproduce existing members. Provide distinct `semantic_extract_type` / `extract-type` and `semantic_extract_delegate` / `extract-delegate` handovers in this shared investigation: they share member identity and layout proofs but have different call-rewrite policies.

No extraction-of-type or delegate operation is registered on main or in audited outstanding worktrees. `semantic_insert_type` can insert a supplied declaration; `semantic_insert_function` already inserts a supplied receiver method. Neither computes member relocation, object composition, or forwarding calls.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this RQ records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

## Bounded delivery and non-goals

First milestone: extract an explicitly selected method-only responsibility from an unexported, non-generic receiver into an existing destination file in the same package. No state is relocated in this milestone, but the new composition field still changes layout and potentially comparability/reflection. Require explicit private-type use analysis for those effects. Type extraction updates selected method call sites to the new component; delegate extraction retains wrappers on the original receiver. Require an explicit instance field name and preserve single receiver evaluation.

Second milestone: private named fields with keyed literals only after proving object identity, zero values, comparability and initialization order. Defer exported structs, positional literals, embedding, reflection/serialization-sensitive types, generic receivers, cross-package relocation and implicit choice of member groups. Do not claim these are safe because the result compiles.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

## Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Existing file of source type; file selector | Required |
| `symbol` | Target: new simple type name, e.g. `readerState` | Required |
| `in_type` | Selector: existing qualified source type `reader` | Required |
| `members` | Typed array of selected `{file, symbol}` member identities | Required, nonempty |
| `destination_file` | Existing destination Go file in the same package | Defaults to `file` |
| `field` | New composition field identifier on original type | Required |
| `access_modifier` | Existing visibility convention for generated type | Defaults to private |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_extract_delegate",
  "params": {
    "file": "reader.go",
    "symbol": "readerState",
    "in_type": "reader",
    "members": [
      {
        "file": "reader.go",
        "symbol": "(*reader).read"
      }
    ],
    "destination_file": "state.go",
    "field": "state"
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "state.go",
  "symbol": "readerState",
  "in_type": "reader",
  "destination_file": "state.go",
  "continuations": [
    {
      "file": "reader.go",
      "symbol": "reader.state"
    },
    {
      "file": "reader.go",
      "symbol": "(*reader).read"
    },
    {
      "file": "state.go",
      "symbol": "(*readerState).read"
    }
  ]
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

## Implementation sequence

1. Resolve one source type and exact member objects; inventory all receiver uses, literals, method values and escapes. Define how inspection emits `in_type` and typed `members` without model reconstruction.
2. Write distinct type-extraction and delegate-extraction contracts: the former redirects usages, the latter keeps forwarding methods and interface method sets. Require a closed `members` schema before registration.
3. Build a shared member/dependency planner. Validate non-selected dependencies and receiver aliasing; refuse when behavior relies on original receiver state or identity.
4. Plan the new type, composition field, selected methods, initialization and call changes together. Preserve attached docs/tags and untouched siblings.
5. Check pointer/value method sets, import bindings and complete overlay result; register both operations with the existing coordinator and atomic multi-output Apply.
6. Publish explicit different capability descriptions for each mode; expand to selected fields only after layout/identity proof tests pass.

## Required batch behavior

Batch proof for each operation: insert a method into the source type → extract it to the already scaffolded destination → rename the generated method → replace its body. Inspect the new receiver and old wrapper from the working overlay. A disk probe before final publication must still see the original files. Final receipts identify both generated type and surviving callable locations. Repeat with an earlier unsaved call-site change and with a refused second extraction.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

## Public acceptance and refusal coverage

* `extract_type_method_only.txtar`: two private methods, selected one redirected through a new component; unrelated methods/comments unchanged. Assert behavior and resulting symbol locations.
* `extract_delegate_method_only.txtar`: selected method wrapper keeps original interface satisfaction and evaluates the receiver once. Assert pointer and zero-value behavior.
* Refuse field identity dependencies, method values/expressions without a proven rewrite, positional literals, embedding, inaccessible symbols, generated files and package cycles before writing.
* Compare an explicit state mutation counter and nil receiver outcome before/after; a compile-only test is insufficient.
* Failure in destination planning leaves source, destination, and callers byte-identical.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

## Semantic risks and unresolved decisions

* Moving fields changes address identity, struct layout, tags, promoted methods, equality and exported API. A composition field itself can change comparability or reflection; restrict first delivery to private types with no such observed uses.
* Extract type and extract delegate must not silently share the same wrapper policy. Unchanged dispatch and interface method sets require explicit evidence.
* Nil receivers, receiver-copy behavior and forwarding evaluation are semantic constraints.
* The proposed `field` name must align with field-insertion schemas; never name the owner `symbol` while the new type is the target.

## Decision gates and completion

Before implementation, choose the method-only receiver representation and prove that creation/zero-value initialization preserves the selected scope. If this cannot be proved, narrow to a pure stateless helper responsibility and document it. Accept field support separately after layout, alias and serialization evidence. Do not implement a broad extraction through text replacement.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this RQ Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

## Dependencies, ownership, and entry points

Depends on typed schemas/live member receipts and [field insertion](RQ-0056-go-insert-field.md). Contract implementation and interface extraction can develop in parallel. Share method-set analysis with [implement contract](RQ-0050-go-implement-contract.md); do not merge overlapping registry/index changes concurrently.

* Read `internal/astedit/type.go`, `function.go`, `move_construct.go` and `move_cross_package.go`.
* Extend `internal/operation/wire_engine_declarations.go` through new typed operation definitions; inspect `internal/backend/golang/backend.go`.
* Integrate pending `internal/sourcebuf/buffer.go`, `internal/godiagnostics/manager.go`, and `internal/mcp/buffers.go` after their owning changes are safely available.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_type / semantic_extract_delegate`:**

> Use `semantic_extract_type` instead of reconstructing selected Go members when extracting a supported private method responsibility into a new composed type and redirecting its selected callers. Use `semantic_extract_delegate` for the same bounded extraction when existing methods must remain as forwarding wrappers. Choose the new type `symbol` and composition `field`; select the existing owner with `in_type` and exact `members`. Pass a generated type/member continuation's complete `file` and `symbol` to `semantic_inspect_symbol` or `semantic_rename`; pass only callable continuations to `semantic_replace_body`. Unsupported state, layout, or identity effects refuse.

**Parameter description requirements:**

`symbol`: Target: fresh simple generated type name; no existing qualified owner symbol supplies it. `in_type` and `members`: Selectors: proposed `semantic_inspect_symbol.in_type` and `semantic_inspect_symbol.members` from the typed member inspection, gated on the shared schema extension. `field`: Replacement: new simple composition field identifier. Keep it separate from a returned qualified member `symbol`; never strip a receiver/type qualification to reuse it.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
