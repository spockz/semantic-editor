<!-- WHY: This handover keeps the Go signature-change investigation actionable without prematurely accepting a public contract or choosing a backend mechanism. -->
# RQ-0041: Go Change Signature Handover

* **Status**: Open
* **Category**: Product & Architecture
* **Date**: 2026-09-30

## Purpose and shared contract

Plan a Go-first `semantic_change_signature` operation that changes a function or method declaration and its supported call sites from one resolved signature to another. This is an implementation handover, not an accepted design. Apply the shared invariants in [RQ-0024 § Shared Handover Contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract) and report unresolved decision gates there.

Sibling Go investigations: [inline function](RQ-0042-go-inline-function.md), [extract constant](RQ-0043-go-extract-constant.md), [extract interface](RQ-0044-go-extract-interface.md), [introduce parameter](RQ-0045-go-introduce-parameter.md), and [introduce parameter object](RQ-0046-go-introduce-parameter-object.md).

## Observable outcome and boundaries

A successful operation reports the changed file set, final diff, diagnostics, and a `continuations` entry with reusable `file`, surviving `symbol`, and `expected_revision`. Callers can resolve that symbol again and perform a later semantic edit. Keep the original function or method alive; never return a deleted identifier as a live target.

First release: one target function or method declaration in an existing Go file, with call-site rewrites allowed across multiple existing files in the same package within one module. Preserve result arity, types, parameter order, and existing parameter names exactly; append one new parameter only when every call site maps safely. Refuse unresolved references, unsafe method-set changes, unproven build variants, exported API migration, variadic/generic transformations, and ambiguous function-value adaptation. Prove the complete call set before publication. Result changes and caller/body result mappings are later work. No framework migration, compatibility shim, speculative preview API, or new-file creation is in scope.

## Audited starting state

The authoritative audit tree is `/Users/alessandro/sources/semantic-editor` at `6c95d12`, including shared uncommitted buffers; it was read-only. Six requested operations were absent in all worktrees. Bounded extraction, inline, move, and removal work already exists. The source tree has a central typed registry, Go AST edits, one-process gopls sessions, and a new `sourcebuf` overlay path. There is no Go change-signature AST operation or registry entry. Current `gopls` client support does not establish command/interactive/applyEdit support. A pending ADR-0060 in the audited tree describes MCP-owned diagnostic sessions; treat it as context, not as a settled refactoring transport contract.

## Candidate implementation paths

* Public operation: `internal/operation/wire_engine_mutations.go` or a narrowly scoped sibling wire file; mirror `wire_extract.go` and `wire_engine_inline.go` for typed parsing, registry definition, MCP/CLI naming, formatting, and receipts.
* AST and semantic analysis: `internal/astedit/function.go`, `decl.go`, `extract.go`, and `internal/symbol/resolver.go`; use `internal/adapters/golang/gopls.go` and `internal/lsp/client.go` for reference/type evidence.
* Buffering/publication: `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, and `internal/mcp/batch.go`. Every result file must be staged as a complete set of `sourcebuf.Change` values and committed to the overlay with one `Buffer.Apply` only after all preimages validate.
* Wire schema: `internal/operation/operation.go`, `parse.go`, and MCP schema generation. Depend on RQ-0024's closed recursive object/array schema and presence-aware CLI JSON support before exposing structured parameter/result descriptors. Do not encode structures in opaque strings.

## Proposed public contract

Names are proposals; settle only after the transport and batch gates. `semantic_change_signature` / CLI `change-signature`; Go backend only initially. Target roles follow ADR-0055: `symbol` names the declaration changed, while `in_function` is only a scope selector where needed. Every source field must validate against the referenced output field by both name and type, and parameter descriptions must identify those sources.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target | string | none | Existing `.go` file path, copied verbatim from a file continuation. |
| `symbol` | target | string | none | Fully qualified function or method; copied from lookup/receipt. |
| `parameters` | replacement | array of closed `ParameterMapping` objects | none | Ordered list determines final signature order. Each entry is `{source_index?: integer, parameter: ParameterSpec{name: string, type: string, variadic?: boolean}, argument?: Goexpr}`. Existing parameters map by zero-based `source_index`; release one keeps them in their original order/name/type. Append one new parameter without `source_index` and with `argument`; reorder/rename mappings are later work. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token from lookup or preceding mutation receipt; binds the selected declaration file only. Capture and validate the complete caller/reference set separately. |

Do not accept unparsed declaration fragments. Release one preserves result count and types; result rename or shape changes require a later explicit result-mapping design. Preserve old parameter bindings by `source_index`, and prove that each new `argument` is available with the same resolved meaning at every affected call. Initially allow only side-effect-free closed expressions: literals/constants or package-level symbols resolving to the same object at every call. Function calls, reads of caller-local names, and mapping extensions are gated. Return the surviving declaration through a `continuations` entry with `file`, `symbol`, and `expected_revision`.

## Decision gates and risks

1. Establish whether gopls exposes a suitable code action and whether the response is a plain `WorkspaceEdit`. If it requires `workspace/executeCommand`, interactive selection, or client-side `workspace/applyEdit`, stop adoption until the transport gate adds and tests that support. Do not claim the current session can apply it.
2. Determine a complete call/reference set across packages, tests, build tags, generated files, method expressions, interfaces, function values, and external API boundaries. First release refuses any call set it cannot prove complete.
3. Decide how named/unnamed result equivalence, variadic parameters, generic signatures, receiver types, and package aliases normalize. Reject rather than guess.
4. Precompute every edited declaration/call file, import delta, and verification input. A late invalid call must leave all overlay bytes unchanged.
5. Match `expected_revision` to the shared RQ-0024 receipt lifecycle. Resolve output continuations after normalization; never emit stale selector paths as live continuations.

## Ordered implementation handover

1. Inspect current function/method resolution, references, batch segmentation, and diagnostic-session ownership; record exact gopls evidence and transport limitations.
2. Prototype a read-only resolver over sourcebuf overlays. Enumerate all direct call sites and unsupported reference forms; prove ambiguity and build-variant refusal.
3. Define a closed typed signature patch schema and CLI encoding with presence semantics. Add registry shape checks for `SourceFields` name and type and tool descriptions with source-field provenance.
4. Implement a pure planner that returns all file preimages and proposed bytes. Preserve comments, unrelated bytes, and declaration layout; use AST/token edits rather than pretty-printing whole files.
5. Validate every output, syntax/type information, collision, import update, and revision before one `Buffer.Apply`. For direct calls, publish each final file atomically; document partial multi-file publication failure honestly.
6. Mark `Batchable: true` and `BufferedBatch: true` for existing Go files. Use the shared overlay for both AST and LSP reads, defer verification/import normalization, then verify and organize once at the batch segment boundary. Earlier successful edits remain after a later edit fails; do not promise whole-batch rollback.
7. Add registry CLI/MCP support, result formatting, capabilities, and txtar coverage; keep new-file creation behind an explicit `semantic_scaffold_file` prerequisite until sourcebuf supports new resources.

## Required evidence and tests

Add public CLI txtar cases that invoke the registered CLI command, assert stdout/diagnostics and exact workspace state, and use the MCP `semantic_batch` surface for batch cases.

* Positive: change a private function signature with direct calls in two existing files; verify declaration and both calls, unchanged result types/arity, successful diagnostics, and a `continuations` entry usable verbatim by lookup and another edit.
* Refusal: unresolved function value, interface implementation break, duplicate parameter name, invalid type, stale `expected_revision`, and one unsupported/build-tag call site. For each, assert byte-identical files and no diff.
* All-output preflight: arrange two caller files where the second conflicts or cannot be transformed; prove the first file and target declaration were not applied.
* Buffered sequence: earlier unsaved `semantic_replace_body` → signature change → a later mutation using the normal resolver, then a standalone lookup/edit using the returned continuation. Assert later steps see the overlay, disk remains unchanged until final flush, normalization/verification run once, and all published files contain only final bytes.
* Retained server: instrument a batch to prove the resolver probe and each edit use the same one-process gopls session over the overlay; do not launch `gopls` CLI as a substitute.
* Byte and no-op: retain comments, unrelated whitespace, line endings, and imports outside planned edits. A replacement signature identical to the current signature must return an empty diff and perform no write.

## Completion decision

Resolve only after the transport gate, call-set completeness proof, typed schema gate, and required txtar/buffered evidence pass. If safe argument migration cannot be established across existing module files, publish the precise unsupported cases and revise the first-release boundary before implementation expands.

## Concrete fixture sketches

Create `testdata/scripts/change_signature_go.txtar` as a public CLI scenario and `testdata/scripts/mcp_batch_change_signature.txtar` for buffered behavior. Include ordinary package source, two direct call sites, a test call site, and a deliberately byte-irregular unrelated Go file.

The positive batch should follow this order. `expected_revision` is omitted for a fresh overlay target; the operation itself resolves all call sites from the overlay. This JSON is a request-shape sketch, not a receipt-interpolation example:

```json
{
  "edits": [
    {
      "tool": "semantic_replace_body",
      "params": {
        "file": "uses.go",
        "symbol": "handle",
        "body": "return process(1)"
      }
    },
    {
      "tool": "semantic_change_signature",
      "params": {
        "file": "service.go",
        "symbol": "process",
        "parameters": [
          {
            "source_index": 0,
            "parameter": {
              "name": "id",
              "type": "int"
            }
          },
          {
            "parameter": {
              "name": "ctx",
              "type": "context.Context"
            },
            "argument": "nil"
          }
        ]
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "uses.go",
        "source": "func Verify() error { _ = process(1, nil); return nil }"
      }
    }
  ]
}
```

Make the fixture's first edit introduce a valid `process(1)` call in `uses.go`, a second existing module file; the final signature appends `ctx` after `id`, preserving existing evaluation order. The new `ctx` argument uses `nil`, a side-effect-free closed expression assignable to `context.Context` at every call site. Assert the later mutation resolves the returned signature and succeeds; a standalone follow-up may consume the emitted continuation. If schema cannot express a valid argument migration, that is a blocking contract finding, not a reason to weaken the proof.

The standalone positive txtar must call `semedit change-signature ...` and compare every changed caller file. Add a second MCP txtar whose final caller has a type error; assert `isError`, original target/caller files byte-for-byte, and no partial diff. Add stale-revision and unsupported function-value fixtures, each with explicit diagnostic assertions.

A focused MCP buffer test must inspect disk after the operation handler returns but before batch flush. A separate stale-preimage case must change a captured input on disk before `Buffer.Apply` or `Flush`, assert `sourcebuf.ErrConflict`, and prove no planned buffered output was published: every output must still equal original disk bytes while sourcebuf reads show all planned edits. It must also assert one gopls process/session ID from the earlier edit through the later mutation, one verification/import pass, and preservation of earlier successful overlay edits after a subsequent batch failure. Per-file final publication may report a partial count if an OS write fails; do not assert whole-batch rollback.

## Expansion milestone and artifact checklist

Milestone two may map caller-local arguments only with per-call binding proofs; milestone three may alter results or handle exported APIs only with explicit caller/body mapping and compatibility contracts. Each expansion records supported and refused Go syntax in capability metadata and adds comparable CLI txtar fixtures. The handover is complete when a reviewer can run the fixture, copy a result continuation verbatim, and predict every file in the final diff.

## Expected research artifacts

Record a signature-to-call-site matrix, one explicit supported/refused syntax table, and the gopls transport trace that justifies the selected implementation route. Keep those findings with the implementation proposal so future language backends can compare capability without inheriting Go-specific assumptions.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_change_signature`:**

> Use this tool instead of editing signatures and callers separately to append one parameter to a supported private Go function. Supply the complete ordered `parameters` mapping; existing parameters retain their order, names, and types. The engine updates every proven direct caller and refuses incomplete or unsupported use sets. Pass each surviving `continuations` entry's `file` and `symbol` verbatim to `semantic_rename` or `semantic_replace_body`; keep its `expected_revision` with that target when accepted by the consuming tool.

**Parameter description requirements:**

`symbol`: Target: qualified existing callable. Copy verbatim from `semantic_lookup.symbol` or `semantic_rename.symbol`, with the corresponding `file`. `parameters`: Replacement: caller-authored final mapping; `source_index` is the zero-based original parameter position, not a navigation line or symbol. Never infer a default caller argument from a nearby result.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
