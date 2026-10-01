<!-- Holds the resumable implementation handover for go encapsulate field, including semantic proof and public batch contracts. -->
# RQ-0048: Go Encapsulate Field

* **Status**: Open
* **Category**: Go Refactoring & Implementation Handover
* **Date**: 2026-09-30

## Outcome and current evidence

Generate accessors for one selected field and rewrite resolved reads/writes across the permitted module scope through `semantic_encapsulate_field` / `encapsulate-field`, preserving evaluation count and receiver identity.

Main supports field rename via gopls, but no accessor generation or reference-aware field encapsulation operation. Supplied methods can already be inserted; this plan is for coordinated generation and use rewriting.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this RQ records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

## Bounded delivery and non-goals

First release targets one unexported named field of a private non-generic struct with ordinary reads and standalone assignments. Getter and setter names are explicit; both receiver and RHS must be evaluated once. Preserve field storage and struct layout. Refuse compound assignments, address-taking, positional/unkeyed literals, embedding, external clients, interface-dependent accessors and reflection-sensitive use.

Later release can lower compound assignments with temporaries after proving evaluation and panic order. Changing field visibility or removing storage is a separate accepted policy.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

## Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Selector: existing field declaration file | Required |
| `symbol` | Target: qualified field, e.g. `config.timeout` | Required |
| `getter` | Target name for generated getter method | Required |
| `setter` | Target name for generated setter method | Required |
| `destination_file` | Existing same-package file for generated accessors | Defaults to `file` |
| `access_modifier` | Accessor visibility using existing values | Defaults to private |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_encapsulate_field",
  "params": {
    "file": "config.go",
    "symbol": "config.timeout",
    "getter": "Timeout",
    "setter": "SetTimeout",
    "destination_file": "accessors.go",
    "access_modifier": "public"
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "config.go",
  "symbol": "config.timeout",
  "destination_file": "accessors.go",
  "continuations": [
    {
      "file": "accessors.go",
      "symbol": "(*config).Timeout"
    },
    {
      "file": "accessors.go",
      "symbol": "(*config).SetTimeout"
    }
  ]
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

## Implementation sequence

1. Resolve exact field object and enumerate reads, writes, literal initialization, address uses, reflection/unsafe indicators and receiver forms from current typed overlay.
2. Specify getter/setter receiver policy and zero/nil semantics; reserve names with receiver method-set collision checks.
3. Generate accessor declarations using existing insertion planning primitives, without applying them independently.
4. Rewrite typed field usages only. Plan receiver/RHS temporaries where proven necessary; reject any use shape outside the initial contract. Initial field storage and literal initialization remain unchanged.
5. Validate the combined accessors and rewritten callers with bound input snapshots and overlay typechecking; apply once.
6. Register CLI/MCP and capability metadata, then prove live accessor continuation chains into rename/body edits.

## Required batch behavior

Batch: insert an unsaved field using the field-insertion operation → encapsulate it → rename the getter → replace the setter body. An earlier unsaved caller must be rewritten and later resolution must locate generated methods. A refused conflicting getter must preserve the earlier inserted field only. Add same-module references outside an MCP starting subdirectory to final-diff coverage.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

## Public acceptance and refusal coverage

* Getter read and setter assignment in another file with exact unrelated field/type byte comparisons.
* Side-effect counter fixture checks receiver evaluation before RHS and once-only calls; nil-pointer evaluation/panic timing unchanged.
* Refuse address-taking, `x.f++`, compound assignment, method-set name collision, promoted ambiguous field and positional literals before writes.
* References to another struct with the same field spelling remain unchanged.
* Default and disabled imports produce the expected accessor/import result without touching unrelated build headers.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

## Semantic risks and unresolved decisions

* Getters can change nil behavior, allocation, receiver copying or method dispatch. Setters can reorder receiver/RHS evaluation, and compound statements can evaluate the receiver more than once.
* Struct literal initialization and field access have distinct contracts; do not reinterpret literal keys as calls.
* Encapsulation cannot safely promise external API migration without a complete external consumer set.
* Generated method names in `continuations` are live symbol targets; `getter`/`setter` identifier strings are configuration, not substitutes for qualified callable continuations.

## Decision gates and completion

Choose pointer/value accessor semantics explicitly and prove evaluation/panic order for accepted read/write forms. Reject unsupported reference forms instead of silently leaving them unrewritten. Record the accepted transformation/visibility invariant before expanding scope.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this RQ Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

## Dependencies, ownership, and entry points

Depends on [field insertion](RQ-0056-go-insert-field.md) for shared collision/member planning and on typed reference/read-set foundations. Independent of general change signature. Share alias/member-set proof utilities with extraction; integrate generated accessor output contracts once.

* Inspect `internal/symbol/resolver.go`, `internal/backend/golang/backend.go`, `internal/astedit/type.go` and the field rename path in pending `internal/adapters/golang/rename_edit.go`.
* Add a focused AST planner and typed registry wire file; use pending `sourcebuf.Buffer.Apply` and existing pipeline result conventions.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_encapsulate_field`:**

> Use this tool instead of generating accessors and replacing field references separately to encapsulate one supported private Go field. Supply its qualified `symbol` and explicit `getter` and `setter` names. The engine preserves field storage and rewrites only proven reads and standalone writes with once-only receiver and value evaluation. Pass the surviving field's `file` and `symbol` to `semantic_inspect_symbol` or `semantic_rename`; pass each generated accessor continuation's complete `file` and `symbol` to `semantic_replace_body` or `semantic_rename`.

**Parameter description requirements:**

`symbol`: Target: qualified existing field; copy `semantic_lookup.symbol` or proposed `semantic_insert_field.symbol` only when it identifies this exact live field, with that producer's matching `file`. `getter` and `setter`: Replacement: fresh simple method names chosen by the caller; generated qualified accessor symbols do not feed these inputs. `destination_file`: Destination selector: existing same-package accessor file, default `file`; it is separate from each accessor's live result `file`.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
