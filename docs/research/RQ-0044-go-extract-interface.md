<!-- WHY: This handover separates Go interface extraction from broad type generation and records the semantic proof needed before offering it as a public operation. -->
# RQ-0044: Go Extract Interface Handover

* **Status**: Open
* **Category**: Product & Architecture
* **Date**: 2026-09-30

## Purpose and shared contract

Plan a Go-first `semantic_extract_interface` operation that creates a named interface from selected methods of a concrete type while preserving the concrete type. This is an actionable research handover, not an accepted decision. Follow [RQ-0024 § Shared Handover Contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract).

Sibling plans: [change signature](RQ-0041-go-change-signature.md), [inline function](RQ-0042-go-inline-function.md), [extract constant](RQ-0043-go-extract-constant.md), [introduce parameter](RQ-0045-go-introduce-parameter.md), [introduce parameter object](RQ-0046-go-introduce-parameter-object.md).

## Observable outcome and boundaries

Success inserts a package-level interface declaration from the selected method set in an existing destination file, verifies the concrete type satisfies it, and preserves the concrete type and methods. Return the interface target and, when useful, its source type as distinct entries in the shared `continuations` array; keep planned names and deleted/renamed targets in separate historical data. Do not create a custom `generated` result shape.

First release: one named, non-generic concrete type in one Go package; select a non-empty explicit subset of exported or same-package methods whose signatures are fully resolved; write interface into the same existing file or an existing same-package `destination_file`. Refuse pointer/value method-set ambiguity, embedded/promoted methods, generic types or methods, inaccessible cross-package methods, method expressions that cannot be represented, constraints, conflicting interface names, and incomplete type information. Do not implement the contract, rewrite dependencies, move methods, or create a new file.

## Audited starting state

Read-only code audit was performed against `/Users/alessandro/sources/semantic-editor` at `6c95d12`, including shared uncommitted state. All six requested operations were absent across worktrees. Go type inspection, declaration insertion, scaffold-file, move, and references exist as lower-level capabilities; no extract-interface operation exists. Receiver method insertion is present but is a separate action. Current source tree has typed registry and sourcebuf batch support. The active LSP implementation does not demonstrate execute-command, interactive, or applyEdit support. Its pending ADR-0060 context is not a decision for this feature.

## Candidate implementation paths

* Resolve method sets and types through `internal/adapters/golang/gopls.go`, `internal/lsp/client.go`, `internal/symbol/resolver.go`, and existing type/outline helpers in `internal/astedit/type.go` and `inspect.go`.
* Build declaration bytes and placement in a focused `internal/astedit/extract_interface.go`, reusing safe declaration/comment anchoring from `decl.go`, `insert.go`, and `comment_anchor.go`.
* Register a typed operation in a dedicated wire file or `wire_engine_declarations.go`; follow `wire_extract.go` for parsing and `FileEditRes` formatting while avoiding stale target receipts.
* Buffer outputs through `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, and `internal/mcp/batch.go`; preflight both source and destination before one `Buffer.Apply`.
* Wait for the shared typed recursive schema for any structured method metadata. `members: MemberTarget[]` requires the shared recursive object schema; emitted method signatures are structured output and need matching typed schemas. `SourceFields` must validate both names and types.

## Proposed public contract

Proposed MCP `semantic_extract_interface`, CLI `extract-interface`, Go-only. `file` is the source file; `in_type` is the sole owner-type selector for the concrete source type. `symbol` names the new interface target, and `destination_file` is where it will be declared. Use `expected_revision` only for the selected source `file`; independently capture and validate the member set, method signatures, and destination preimage. Require fresh member selection if the resolved method set changes.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | source file | string | none | Existing `.go` file declaring the concrete type. |
| `in_type` | owner selector | string | none | Qualified concrete type; sole source owner-type selector. |
| `symbol` | created target | string | none | New package-level interface identifier. |
| `destination_file` | destination target | string | same as `file` | Existing `.go` file in the same package. |
| `receiver_kind` | policy | string enum | none | Required `pointer` or `value`; validate satisfaction by that exact method set. |
| `members` | selection | array of `MemberTarget` objects | none | Non-empty list of `{file: string, symbol: string}` entries identifying exact method declarations; use the same member target shape as the field/composite operation plans. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token for the selected source `file` only; member files and destination are captured and validated separately. |

Return the interface as the primary continuation `{file: destination_file, symbol, expected_revision}`. The unchanged source type need not be returned as a continuation. The interface continuation is the primary operation result; do not claim the input `file` + `in_type` pair is the interface target. Never report a removed/replaced method as a live symbol. All generated targets use `continuations`, not `generated`. Method signature output must distinguish name, parameter names/types, results, and variadic status in typed fields.

## Decision gates and semantic risks

1. Require `receiver_kind: pointer | value`, matching implement-contract. Verify satisfaction by `*T` or `T` respectively; method receiver spelling alone does not select the requested method set.
2. Embedded/promoted methods and generic receiver/type parameters need a clear representation and selection proof; refuse them in release one unless gopls resolves exact declarations and interface signatures.
3. Decide whether unexported methods are allowed only for same-package interfaces. Never construct an interface that cannot be satisfied outside its package while implying broader usability.
4. Validate duplicate names, parameter/result signatures, variadic forms, imports, comments, and destination package equality before mutation.
5. Gopls code actions are candidates only. Gate on returned edit shape and transport support; command, interaction, and applyEdit are absent from current session support. Do not use gopls CLI.
6. Resolve interface receipt after formatting/import organization and verify the new symbol is still live at its final destination.

## Ordered implementation handover

1. Prototype read-only method-set enumeration against a real in-process gopls session and compare it with Go type-checker method sets for pointer/value receivers and embedded types.
2. Record refusal diagnostics for unsupported or incomplete method resolution. Do not silently omit methods from an apparent “all methods” result.
3. Define the output signature object schema and source-field continuations, then add `semantic_extract_interface` to the central registry only after the recursive schema/CLI JSON gate.
4. Generate a declaration from type-checked signatures; preserve docs only when attribution is unambiguous and avoid claiming interface docs belong to methods if that changes meaning.
5. Validate the selected source revision, independently captured member/body/destination preimages, name collision, package identity, interface satisfaction, and every changed file before applying one `Buffer.Apply`.
6. Set `Batchable` and `BufferedBatch` true for existing Go files. Use sourcebuf for all reads and edits, no disk planner/barrier, deferred/coalesced verification and imports. Retain prior successful edits on later batch failure; publication is atomic per file, not whole batch.
7. Provide CLI/MCP schema, capability metadata, and public txtar coverage. Require explicit `semantic_scaffold_file` before any new destination file until sourcebuf resource semantics are designed.

## Required evidence and tests

Public CLI txtar tests assert output/diagnostics and complete resulting workspace state; MCP batch examples exercise the same public registry entry.

* Positive: extract `Read` and `Write` from a concrete `Store` into `StoreAPI` in the same existing file; assert signatures, satisfaction by the intended pointer/value type, preserved concrete type, and usable destination/interface receipt for later lookup and edit.
* Refusals: one missing method, pointer/value ambiguity, embedded promoted method, generic type, duplicate interface name, invalid destination package, unresolved method signature, stale revision, and unsupported unexported cross-package method. Assert all relevant files remain byte-identical.
* All-output preflight: source and destination are separate files; stale or unwritable destination prevents the interface declaration and any source edit. First release should not alter source file unless a specific interface assertion is an explicit output.
* Buffered sequence: unsaved insertion of a receiver method → extract interface → later mutation that consumes the new interface type. A standalone follow-up may use the interface continuation for lookup/edit. Prove overlay visibility, no disk publication before final flush, one coalesced verification/import pass, and retention of earlier successful overlay edits if the last batch operation fails.
* Retained server: demonstrate one gopls process/session serves method-set/type checks before and after earlier buffered edits; do not use its CLI.
* Byte/no-op: preserve comments, method bodies, imports, and unrelated bytes. If the requested interface already exists exactly, return an explicit no-op with empty diff; do not write or report a duplicate created target.

## Completion decision

Resolve after type-set semantics and public transport are evidenced, and after the batch test demonstrates that generated interface text is derived from the same unsaved revision later operations inspect.

## Concrete fixture sketches

Add `testdata/scripts/extract_interface_go.txtar` and `testdata/scripts/mcp_batch_extract_interface.txtar`. Use `store.go` with a concrete type and pointer receiver methods, `store_test.go` with a compile-time interface assertion, and an unrelated formatting-sensitive source file. Keep all destination files existing in release-one fixtures.

Exercise an earlier unsaved receiver method, extract an interface from the same overlay, then insert a later declaration that consumes the generated interface type. This JSON is a request-shape sketch, not receipt interpolation; a standalone follow-up can consume the interface continuation:

```json
{
  "edits": [
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "store.go",
        "source": "func (s *Store) Close() error { return nil }"
      }
    },
    {
      "tool": "semantic_extract_interface",
      "params": {
        "file": "store.go",
        "in_type": "Store",
        "symbol": "StoreAPI",
        "receiver_kind": "pointer",
        "destination_file": "store.go",
        "members": [
          {
            "file": "store.go",
            "symbol": "(*Store).Read"
          },
          {
            "file": "store.go",
            "symbol": "(*Store).Close"
          }
        ]
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "store.go",
        "source": "func UseStore(s StoreAPI) {}"
      }
    }
  ]
}
```

Provide signatures that type-check and assert the intended pointer method set satisfies `StoreAPI`, while the concrete type and methods remain unchanged. The standalone CLI txtar invokes `semedit extract-interface ...` and checks the exact generated declaration and live receipt fields.

Refusal fixtures cover value/pointer mismatch, embedded promotion, generic receiver, duplicate interface name, wrong-package destination, missing selected method, and stale revision. For separate source/destination files, force destination preflight failure and compare both files to their original bytes.

The batch test must prove one overlay revision across inserted method, method-set analysis, interface insertion, and later type-resolving mutation; a separate stale-preimage case changes the selected source `file` before `Buffer.Apply`/`Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published. Independently preflight the complete member and destination set. Assert disk is unchanged before flush, all destination preimages are checked before apply, imports/verification coalesce once, and earlier edits survive a later failure in the overlay. Record the same gopls process/session ID through all steps; no gopls CLI. Per-file atomic publication does not imply multi-file rollback.

## Expansion milestone and artifact checklist

Release one includes an existing same-package destination distinct from the source file, with complete destination preflight. Later milestones may consider embedded or generic method sets only after exact type-set proof. Cross-package consumers, new files, and method implementation are separate capability decisions. A no-op on an identical existing interface must have an explicit receipt and no write.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_interface`:**

> Use this tool instead of reconstructing method signatures to create a Go interface from an explicit resolved method selection. Choose the new interface `symbol`; `in_type` and `members` select the concrete source type and methods. Choose the required `receiver_kind` as `pointer` or `value`; satisfaction is checked for that exact method set. `destination_file` defaults to the source `file` and must already exist in the same package. Pass the interface continuation's destination `file` and live `symbol` to `semantic_inspect_symbol` or `semantic_rename`. This operation generates the interface declaration, without implementing its methods or rewriting consumers.

**Parameter description requirements:**

`symbol`: Target: new simple interface identifier chosen by the caller, not the source type name. `in_type`: Selector: copy proposed `semantic_inspect_symbol.in_type` only after the shared inspection extension exists. `members`: Selector: copy proposed `semantic_inspect_symbol.members`, the exact typed member records from that inspection result, with matching `file` and `symbol`; do not rebuild method identities from display signatures. The result `file` names the generated interface's destination. `receiver_kind`: Policy: required `pointer` or `value`, consistent with `semantic_implement_contract`; descriptive lookup `kind` does not feed it.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
