<!-- WHY: This handover makes the parameter-object transformation reviewable by exposing its schema prerequisite, evaluation-order hazards, and bounded migration proof. -->
# RQ-0046: Go Introduce Parameter Object Handover

* **Status**: Open
* **Category**: Product & Architecture
* **Date**: 2026-09-30

## Purpose and shared contract

Plan a Go-first `semantic_introduce_parameter_object` operation that groups a selected function's parameters into one named struct and migrates its body and supported callers. This is an implementation handover, not an accepted decision. Apply [RQ-0024 § Shared Handover Contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract).

Sibling investigations: [change signature](RQ-0041-go-change-signature.md), [inline function](RQ-0042-go-inline-function.md), [extract constant](RQ-0043-go-extract-constant.md), [extract interface](RQ-0044-go-extract-interface.md), and [introduce parameter](RQ-0045-go-introduce-parameter.md).

## Observable outcome and boundaries

Success inserts a new package-level struct, replaces selected function parameters with one object parameter, rewrites body references to fields, migrates each supported call to a keyed struct literal, and type-checks all outputs. Return the surviving function and the created struct as distinct targets in the shared `continuations` array, each with copyable `file`, `symbol`, and `expected_revision`. Planned names and removed targets stay outside live continuations.

First release: migrate all parameters of one private, non-generic, non-variadic free function into a new struct in the same existing file and package. One target declaration may have direct callers across multiple existing files in the same package within one module; preflight every caller. This boundary keeps evaluation order stable by constructing fields in original argument order as the sole argument. Preserve named parameter types and body meaning. Refuse partial grouping, exported APIs, unknown callers, interfaces, function values, generic types, promoted fields, conflicting names, comments/directives that cannot be preserved, or arguments that cannot become keyed fields safely. Partial grouping may be a later milestone only with explicit evaluation-order preservation proof. No new-file creation.

## Audited starting state

The read-only authoritative audit tree is `/Users/alessandro/sources/semantic-editor` at `6c95d12`, with shared uncommitted files/buffers included. These six operations were absent across worktrees. Existing signature-related primitives do not implement migration to a parameter object. Declaration/field/type helpers, reference resolution, registry-backed CLI/MCP operations, one-process gopls sessions, `sourcebuf`, and `Buffer.Apply` exist in the audited source. Current transport does not establish support for command execution, interaction, or applyEdit. ADR-0060 remains a pending diagnostic-session record in that tree, not a decision for this operation.

## Candidate implementation paths

* Semantic resolution: `internal/adapters/golang/gopls.go`, `internal/lsp/client.go`, `internal/symbol/resolver.go`, and `internal/astedit/type.go`; enumerate all direct callers and body bindings from the shared overlay.
* AST planning: a focused `internal/astedit/introduce_parameter_object.go`, using `function.go`, `decl.go`, `insert.go`, `comment_anchor.go`, and `move_construct.go` for declaration/source-range patterns. Keep this transformation distinct from single-parameter introduction.
* Wire and schemas: dedicated `internal/operation/wire_parameter_object.go` (or signature-focused file), `operation/operation.go`, `parse.go`, and MCP schema generation. Nested `fields` structures require closed recursive schemas and presence-aware CLI JSON; do not represent them as JSON strings.
* Buffering: `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, and `internal/mcp/batch.go`; produce the struct, changed declaration, and every call-site edit as a complete `Change` set and invoke `Buffer.Apply` once.

## Proposed public contract

Proposed MCP `semantic_introduce_parameter_object`, CLI `introduce-parameter-object`, Go-only. `symbol` is the function/method target. `destination_file` names where the new struct is declared; do not use it as the source path. `in_function` is not appropriate for this declaration-level target. The accepted all-parameters form maps every original parameter by zero-based index into a generated struct field; mappings are explicit and revision-bound.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target file | string | none | Existing file containing function/method. |
| `symbol` | target | string | none | Qualified function or method. |
| `destination_file` | destination target | string | `file` | Must equal `file` in release one; a separate existing same-package file is milestone two. New files require a later resource scope decision and prior scaffold. |
| `parameter` | new callable parameter | closed `ParameterSpec` | none | `{name: string, type: string, variadic?: boolean}` for the sole object argument; `type` is the new package-level struct name. |
| `fields` | ordered field mappings | array of closed `ParameterToFieldMapping` objects | none | Each entry is `{source_index: integer, field: FieldSpec{name: string, type: string, tag?: string}}`; `source_index` identifies an original function parameter. Array order determines struct field order. Every original parameter appears exactly once; field type must preserve the selected parameter type in release one. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token for the selected declaration `file` only; body, call set, and destination are captured and validated separately. |

Validate `parameter` and field names plus source indices structurally; require unique, complete indices and preserve each source parameter type. Do not accept arbitrary source fragments. Return the function and created struct as separate `continuations` entries with `file`, `symbol`, and `expected_revision`. Do not overload `symbol` or invent a `generated` object. Historical/planned names are not live continuations.

## Decision gates and semantic risks

1. Structured map types are not supported by today's registry contract. RQ-0024 must add typed recursive object/array schemas, presence-aware CLI JSON parsing, and `SourceFields` name-and-type checks before this tool's `fields` array is exposed.
2. Determine parameter grouping and Go call evaluation order. Release one requires all parameters and a keyed literal as the single argument; preserve field order to match original left-to-right argument evaluation. Partial grouping requires temporary statements or a separate proof.
3. Rewrite references only when they bind to the target parameter; protect shadowed locals, closure captures, labels, comments, and docs. Type-check all rewritten uses.
4. Prove complete caller references in the same module and reject function values, interface dispatch, unknown external callers, generated/build-specific files, and exported APIs until supported.
5. Define field-export policy and API visibility. A private function may still need exported fields if object crosses package boundaries; first release should keep type and function package-private.
6. Gopls code actions remain candidates. Current client lacks established command, interactive, and applyEdit support; block that implementation route until transport tests pass. Never use gopls CLI.
7. Validate all preimages/revisions and normalize before computing live receipt continuations; never return stale call paths.

## Ordered implementation handover

1. Resolve the shared schema prerequisite and cite its decision in the eventual implementation change. Build typed call/reference inventory for private function scope.
2. Prototype an all-parameters conversion for one function and straight-line callers. Compare type-checked before/after behavior; assert keyed literal evaluation order explicitly.
3. Define object/field receipt shape, CLI nested JSON syntax, and source-field name/type checks. Add registry validation before exposing the operation.
4. Produce all AST changes in memory: struct declaration, parameter list, body reference rewrites, and every caller literal. Preserve comments and all bytes outside planned edits.
5. Validate destination package/name, all references, scopes, types, expected revision, and every output. Call `Buffer.Apply` once after successful complete preflight.
6. Register with `Batchable: true` and `BufferedBatch: true` for existing Go files. Run inside the batch's one sourcebuf overlay and one gopls session; coalesce verification/import formatting. Earlier successful buffered edits remain when later edits fail; no whole-batch rollback is promised.
7. Add CLI/MCP docs, capability metadata, and txtar cases. Require explicit `semantic_scaffold_file` before targeting any newly created file; sourcebuf new-resource semantics remain undefined.

## Bounded release and expansion milestones

* **Milestone 1:** all parameters of one private function in one package; same-file struct; direct typed calls only; keyed literal preserves evaluation order.
* **Milestone 2:** destination in an existing same-package file; exported types/functions only after external API/reference completeness is demonstrated.
* **Milestone 3:** partial grouping only if the planner preserves argument evaluation order across interleaved retained parameters and proves behavior for nested expressions.
* **Milestone 4:** methods, generic functions, interface dispatch, multiple packages, and newly scaffolded resources only after each has explicit capability metadata and independent preflight/publication evidence.

## Required evidence and tests

Use public CLI txtar operations and MCP `semantic_batch`; assert tool output/diagnostics, exact diff, final workspace, and valid JSON matching the proposed `ParameterMapping` schema. Keep per-operation refusal cases fail-closed.

* Positive: transform a private `write(path, data string, mode uint32)` into `write(options writeOptions)`; assert struct and keyed call literal, field reference rewrites, compile/type result, and follow-up lookup/edit using surviving receipt fields.
* Refusals: partial field selection in milestone 1, unknown call site, method value/interface dispatch, duplicate struct/field name, generic function, stale revision, invalid destination package, body shadowing/capture, and side-effecting calls that violate order. Assert no file bytes change.
* All-output preflight: separate destination file plus multiple caller files; make the last caller stale/untransformable and prove neither destination, declaration, body, nor earlier caller is changed.
* Buffered sequence: earlier unsaved call insertion → introduce parameter object → later mutation resolves and uses the new type/field names. A standalone follow-up can consume the function/type continuations. Prove later calls see overlay, disk remains unchanged until flush, final normalization/verification run once, and failure later in the batch retains earlier successful buffered operations.
* Retained server: verify one-process gopls identity across prior edit, complete call/type analysis, and a subsequent resolver probe. No gopls CLI evidence.
* Byte/no-op: preserve function comments, call comments, line endings, imports, and unrelated bytes. If the function already takes the requested object and all call literals match, return an empty diff with no write; never report a duplicate live declaration.

## Completion decision

Keep open until schema support, all-caller proof, and evaluation-order semantics are resolved and the public batch evidence passes. Do not widen from all-parameter migration to arbitrary partial grouping based only on successful type checking.

## Concrete fixture sketches

Add `testdata/scripts/introduce_parameter_object_go.txtar` and `testdata/scripts/mcp_batch_introduce_parameter_object.txtar`. Use a private three-parameter function, two existing callers across existing same-module files, and one callback-like refusal. Release one keeps the destination equal to source; separate existing-file destinations are a later milestone.

Create an earlier unsaved call, perform the conversion, then run a later mutation with statically known `file`/`symbol` targets so the registry resolves the new signature and struct. Consume returned `continuations` in a separate follow-up call; current `semantic_batch` has no receipt interpolation or read-only batching. Omit `expected_revision` for these fresh overlay targets:

```json
{
  "edits": [
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Run() error { return write(\"a\", \"b\", 7) }"
      }
    },
    {
      "tool": "semantic_introduce_parameter_object",
      "params": {
        "file": "write.go",
        "symbol": "write",
        "destination_file": "write.go",
        "parameter": {
          "name": "options",
          "type": "writeOptions"
        },
        "fields": [
          {
            "source_index": 0,
            "field": {
              "name": "path",
              "type": "string"
            }
          },
          {
            "source_index": 1,
            "field": {
              "name": "data",
              "type": "string"
            }
          },
          {
            "source_index": 2,
            "field": {
              "name": "mode",
              "type": "uint32"
            }
          }
        ]
      }
    },
    {
      "tool": "semantic_replace_body",
      "params": {
        "file": "write.go",
        "symbol": "write",
        "body": "return save(options.path, options.data, options.mode)"
      }
    }
  ]
}
```

The fixture must declare matching source parameter types and helper signatures. Assert generated struct fields, `write(options writeOptions)`, field-bound body uses, and keyed `writeOptions{path: ..., data: ..., mode: ...}` call. The standalone CLI txtar invokes `semedit introduce-parameter-object ...` and compares every affected file. Assert the result uses two `continuations` entries, one for `write` and one for `writeOptions`, with real final revisions.

Refusal fixtures include partial fields in milestone one, side-effecting arguments that would be reordered, closure capture/shadowing, interface dispatch, unknown caller, stale revision, duplicate object/field names, and a destination with another package. For multi-file outputs make the last caller stale and assert no output changes.

The buffer test should prove the full struct/declaration/body/call set is validated before one `Buffer.Apply`; a separate stale-preimage case changes a captured input before `Buffer.Apply`/`Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published; prior disk bytes remain visible until final flush while later mutations resolve the overlay. Read-only `semantic_lookup` is not part of `semantic_batch` because its registry entry is not batchable. Assert same gopls process/session, coalesced verification/import normalization, and successful earlier operations retained if a later batch entry fails. Add an explicit test that final publication is per-file atomic and does not claim whole-batch rollback.

## Expansion milestone and artifact checklist

Milestone two may allow a separate existing same-package destination file after publication preflight is proven; milestone three may evaluate partial grouping, external callers, or new-file destinations independently. Capability metadata must say when generic functions, methods, and interface dispatch remain unsupported. The typed receipt must make the surviving function target and any nested new-object continuation unambiguous.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_introduce_parameter_object`:**

> Use this tool instead of reconstructing signatures, fields, and call arguments to group all parameters of one supported private Go free function into a named struct. Supply the sole object `parameter` and ordered `fields` mappings. Release one creates the struct in the source file and updates every proven direct caller while preserving argument order. Pass the callable continuation's `file` and `symbol` to `semantic_replace_body`, or the created type continuation to `semantic_inspect_symbol` or `semantic_rename`. Choose the appropriate complete target record.

**Parameter description requirements:**

`symbol`: Target: existing callable copied from `semantic_lookup.symbol` with its `file`. `parameter`: Replacement: caller-chosen object parameter name and new struct type. `fields`: Replacement: ordered `ParameterToFieldMapping` records; `source_index` refers to the original signature, not the final field ordinal. `destination_file`: Destination selector: defaults to `file` and must equal it in release one. Generated qualified type names never feed the new simple-name input implicitly.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
