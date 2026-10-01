<!-- Holds the resumable implementation handover for go type migration, including semantic proof and public batch contracts. -->
# RQ-0049: Go Type Migration

* **Status**: Open
* **Category**: Go Refactoring & Implementation Handover
* **Date**: 2026-09-30

## Outcome and current evidence

Provide `semantic_migrate_type` / `migrate-type` for an explicitly selected type use or declaration and a permitted graph of dependent uses. The operation must distinguish exact equivalence from an intentional compatibility change.

No type-migration operation exists in audited code. Rename changes identity spelling; replacing a declaration does not propagate conversions through callers, assignments or interface conformance. Signature changes in RQ-0041 cover callable boundaries, not a general type-use graph.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this RQ records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

## Bounded delivery and non-goals

First release is behavior-preserving alias normalization: migrate a selected unexported declaration/field from an alias to its identical aliased type, or the reverse only where exact type identity is proved. Do not guess conversions. Accept scope within one enclosing module and existing regular Go files.

Later gates cover distinct named types, explicit conversion rules and function-result/parameter changes. Defer numeric narrowing, representation changes, map-key comparability changes, unsafe layout, serialization-sensitive exports and generic inference until separate semantics are proved. This narrower first release is intentional; it does not advertise arbitrary type migration.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

## Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Existing declaration file selector | Required |
| `symbol` | Target declaration or qualified field whose type changes | Required |
| `type` | Desired Go type expression resolved in target declaration scope | Required |
| `in_function` | Owner selector only for supported local targets | Optional; first release excludes locals |
| `scope` | Permitted propagation boundary | Defaults to module; only module initially |
| `conversions` | Future closed typed conversion rules, never unchecked source substitution | Omit in first release; reject supplied value |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_migrate_type",
  "params": {
    "file": "config.go",
    "symbol": "config.timeout",
    "type": "time.Duration",
    "scope": "module"
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "config.go",
  "symbol": "config.timeout",
  "type": "time.Duration",
  "expected_revision": "<post-edit-file-revision>"
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

## Implementation sequence

1. Define target kinds and alias-identity evidence; resolver must return both type object and declaration position from current overlay.
2. Determine affected type-use/assignment/interface graph with exact object identities. Keep unrelated identical spellings untouched.
3. Decide which declarations change and which already remain assignable; first release admits only exact identity, including contextual imported type name resolution.
4. Bind module/local replacement inputs; plan minimal type spelling and imports across all outputs. Conversion maps cannot bypass missing reference coverage.
5. Validate alias cycles, method-set identity, generic constraints, nil/comparability and full merged-overlay results before applying.
6. Register first scope truthfully. Add distinct-type migration only with explicit conversion schema shared with signature transformations and observable compatibility checks.

## Required batch behavior

Batch: insert an alias without publication → migrate an existing private field to it → rename the alias → edit a downstream callable using that type. All resolution must use the working revision, including sibling files outside the MCP starting directory. Refusal of a distinct-type conversion must retain only earlier successful mutations and leave no attempted output bytes.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

## Public acceptance and refusal coverage

* Alias-to-original and reverse identity fixtures with value behavior, method sets, interface assignments, nil and map-key use unchanged.
* Same-spelled type in another package remains unchanged; imported type alias resolves correctly when package name differs from directory.
* Refuse distinct numeric/struct types, overflow, changed comparability, inaccessible desired type, alias cycle, exported external-consumer uncertainty, unsupported conversion rules and non-Go files before writes.
* Record a diagnostic introduced by an unrelated earlier batch child without claiming the entire workspace was clean.
* No-op identical type produces empty final diff and preserves headers exactly.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

## Semantic risks and unresolved decisions

* Assignability is weaker than identity and behavior preservation. Explicit conversions can truncate, allocate, change methods or alter observable interface values.
* Aliases and new defined types have different identity/method/generic semantics.
* Desired `type` text must resolve separately at each edited file with stable imports; copying aliases blindly is unsafe.
* Complete exported external consumers cannot normally be discovered from the current module. The first contract must refuse that promise.

## Decision gates and completion

Choose one unambiguous target-kind vocabulary and verify exact alias identity through the compiler. Define an explicit compatibility policy before accepting distinct named types or conversions. If a migration needs signature updates, call a shared planner rather than invoking independently publishing child operations.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this RQ Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

## Dependencies, ownership, and entry points

Depends on structured schemas, typed whole-plan references, and [change signature](RQ-0041-go-change-signature.md) for future callable transformations. Develop the read-only identity graph after common foundations; schedule broad migration after signature, member and interface proof utilities stabilize.

* Inspect `internal/astedit/type.go`, `decl.go`, `replace_decl.go`, and compiler-overlay loading in `move_cross_package.go`.
* Integrate `internal/operation`, `internal/backend/golang/backend.go`, pending `sourcebuf` and retained session manager; keep conversion planning in a language backend, not transport.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_migrate_type`:**

> Use this tool instead of changing type annotations and conversions by text to migrate one supported Go declaration to an identity-equivalent type spelling. Supply the existing qualified `symbol` and desired `type`. Initial support is limited to proven alias identity; distinct-type conversions and broader compatibility changes refuse. Pass the surviving declaration's exact `file`, `symbol`, and resolved `type` to a later `semantic_migrate_type` request, or its `file` and `symbol` to `semantic_inspect_symbol` or `semantic_rename`. An identical result is a no-op.

**Parameter description requirements:**

`symbol`: Target: existing declaration, copied from a compatible `semantic_lookup.symbol` with its `file`. `type`: Replacement: desired Go type expression in the target scope; a previous `semantic_migrate_type.type` is copyable only in that same scope with unchanged import bindings. `scope`: Selector: module boundary, default module; it does not authorize writes to read-only replacement modules. Reject unsupported `conversions` in release one.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
