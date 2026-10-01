<!-- Holds the resumable implementation handover for go implement contract, including semantic proof and public batch contracts. -->
# RQ-0050: Go Implement Contract

* **Status**: Open
* **Category**: Go Refactoring & Implementation Handover
* **Date**: 2026-09-30

## Outcome and current evidence

Generate the missing receiver methods for an existing Go interface through `semantic_implement_contract` / `implement-contract`, without requiring the model to reproduce each signature.

The capability is proposed in RQ-0024 but absent from all audited implementations. Existing method insertion accepts complete supplied source. gopls documents an implement-interface action, but code-action command/dialog/edit handling is not automatically available through the current rename-only edit contract.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this RQ records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

## Bounded delivery and non-goals

First release handles one non-generic concrete named type and one non-generic ordinary interface, including interface embedding after resolved method-set flattening. Generate only genuinely missing methods into an existing same-package file. Pointer versus value receiver is explicit and checked against method sets. A same-named method with incompatible signature is a refusal, not a missing method.

Generated panic stubs are editing assistance, not functional implementations. The request explicitly selects a stub policy; no inferred business logic or silent zero-value bodies. Defer generic constraints, inaccessible external unexported methods, ambiguous promoted methods and delegated implementations.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

## Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Existing concrete type declaration file | Required |
| `symbol` | Target concrete type `Worker` | Required |
| `contract` | Selector: qualified interface name resolved from `contract_file` | Required |
| `contract_file` | Existing declaration file for the interface; can be read-only | Required |
| `destination_file` | Existing same-package file for generated methods | Defaults to `file` |
| `receiver_kind` | Receiver convention `pointer` or `value` | Required |
| `body_policy` | Generated implementation policy; `panic_stub` initially | Required |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_implement_contract",
  "params": {
    "file": "worker.go",
    "symbol": "Worker",
    "contract": "Runner",
    "contract_file": "runner.go",
    "destination_file": "worker_methods.go",
    "receiver_kind": "pointer",
    "body_policy": "panic_stub"
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "worker.go",
  "symbol": "Worker",
  "contract": "Runner",
  "contract_file": "runner.go",
  "destination_file": "worker_methods.go",
  "continuations": [
    {
      "file": "worker_methods.go",
      "symbol": "(*Worker).Run"
    }
  ]
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

## Implementation sequence

1. Inspect exact concrete/interface objects and compute pointer/value method sets from current overlay. Resolve embedded interface methods and duplicate/conflicting signatures.
2. Compare documented gopls action behavior with the required receiver, body and source-preservation contract. If command/dialog/applyEdit support is necessary, implement a scoped capture/validation gate under a separate accepted transport decision; do not invoke the CLI or let a server publish.
3. Prefer deterministic signature-driven generation when the action cannot satisfy the contract. Preserve parameter/result types, variadics and cross-package import bindings; choose collision-free local names only where source names are absent.
4. Build all missing methods in one plan, preserving existing implementations byte-for-byte. Check inaccessible methods, receiver conflicts and import collisions before Apply.
5. Prove selected concrete pointer/value type satisfies the interface in the resulting typed overlay. That proof validates signatures only; receipt must state that generated bodies are stubs.
6. Register typed contract selectors and generated callable continuations; expose Go support and empty-plan no-op behavior.

## Required batch behavior

Batch: insert an interface and concrete type into existing files → implement contract → rename one generated method through gopls → replace its stub body. The latter edits must resolve unsaved generated declarations. Also change the interface signature in an earlier child and verify generated methods match the revised interface, not disk. Refused conflicting method generation leaves earlier edits intact and contributes no half-generated method set.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

## Public acceptance and refusal coverage

* Generate two missing methods and retain one existing method/body/comment exactly; assertion demonstrates selected receiver method-set conformance.
* Embedded interface, imported parameter types, variadic signatures and multiple results retain their declarations and compile.
* Pointer versus value receiver acceptance/refusal, conflicting existing signature, inaccessible external method and promoted ambiguity cause zero attempted edits.
* Existing complete implementation is a no-op with no formatting of untouched files.
* Invoke the generated panic stub and recover its explicit placeholder panic; never label the body behavior implemented.
* Protocol test captures/refuses unsolicited or unsupported code-action resource edits if a native action route is chosen.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

## Semantic risks and unresolved decisions

* Structural name matching does not establish interface satisfaction: signatures, package identity of unexported methods and pointer/value method sets matter.
* Code-action availability is not a promise of safe client behavior; standard actions may execute commands and interactive actions may need dialogs.
* A panic stub deliberately changes behavior when called; this operation is code generation with visible placeholder policy, not an equivalence guarantee.
* Canonical `contract`/`contract_file` selectors and `receiver_kind` vocabulary need a registry/schema decision; do not create inconsistent `interface`, `trait` and `contract` fields per backend.

## Decision gates and completion

Choose a deterministic stub-body policy and how callers inspect it before exposing the operation. Resolve native-action transport requirements or use the AST generation route. Validate the same requested pointer/value type against the interface rather than a superficially passing assertion on a different type.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this RQ Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

## Dependencies, ownership, and entry points

Can develop alongside [extract interface](RQ-0044-go-extract-interface.md) after common schemas/session foundations. Share method-set comparison and generated declaration receipt logic. Do not add a duplicate generic method-insertion operation; [field insertion](RQ-0056-go-insert-field.md) is a distinct gap.

* Read `internal/astedit/function.go`, `type.go`, `internal/operation/wire_engine_declarations.go`, `internal/backend/golang/backend.go` and `internal/lsp/client.go`.
* Reuse pending `internal/adapters/golang/rename_edit.go` validation concepts, `sourcebuf.Buffer.Apply`, and session synchronization.
* Primary native-action source: [gopls transformation catalog](https://go.dev/gopls/features/transformation).

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_implement_contract`:**

> Use this tool instead of reconstructing interface signatures to generate missing Go methods for a concrete type. `symbol` targets the concrete type; `contract` and `contract_file` select the interface. Choose `receiver_kind` and the explicit `body_policy`; initial `panic_stub` methods panic when called and are not completed implementations. Pass each generated method continuation's exact `file` and `symbol` to `semantic_replace_body` or `semantic_rename`, and the concrete type target to `semantic_inspect_symbol`. Existing implemented methods remain intact.

**Parameter description requirements:**

`symbol`: Target: concrete type copied from a compatible `semantic_lookup.symbol` with its `file`. `contract` and `contract_file`: Selectors: interface identity and declaration file; proposed `semantic_inspect_symbol.contract` and `semantic_inspect_symbol.contract_file` must emit that exact interface context before advertising a copy edge. `receiver_kind`: Policy: required pointer/value choice; descriptive lookup `kind` does not feed it. `body_policy`: Policy: required `panic_stub`; do not imply a useful runtime implementation.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
