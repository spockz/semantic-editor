<!-- WHY: This handover scopes parameter introduction around complete call-site migration, typed argument meaning, and observable batch safety. -->
# RQ-0045: Go Introduce Parameter Handover

* **Status**: Open
* **Category**: Product & Architecture
* **Date**: 2026-09-30

## Purpose and shared contract

Plan a Go-first `semantic_introduce_parameter` operation that adds one parameter to a function or method and updates every supported direct call site with a caller-supplied Go expression. This remains open research, not an accepted decision. Follow [RQ-0024 § Shared Handover Contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract).

Sibling plans: [change signature](RQ-0041-go-change-signature.md), [inline function](RQ-0042-go-inline-function.md), [extract constant](RQ-0043-go-extract-constant.md), [extract interface](RQ-0044-go-extract-interface.md), and [introduce parameter object](RQ-0046-go-introduce-parameter-object.md).

## Observable outcome and boundaries

Success replaces one selected safe expression in the owner body with a new parameter reference, appends that parameter in release one, inserts its closed `argument` at every proven direct call site, type-checks all outputs, and returns the surviving target continuation plus diff/diagnostics. A later lookup/edit must be able to consume the target receipt.

First release: one private, non-generic, non-variadic free function or receiver method with a complete same-package set of statically resolved direct calls and one exact owner-body expression selection. The ordered final `parameters` array maps existing positions and appends one new parameter in release one; reordering is deferred. `discriminator` selects one safe expression in the target body, and optional `construct_path` disambiguates only within the bound file revision. Release one requires both the selected expression and its supplied argument to be compile-time constants with the same typed value at every call. Moving their evaluation from the body to the caller must preserve observable behavior; mutable package state and function calls are refused. Refuse external/exported API boundaries, unknown callers, interface dispatch, method values/expressions, function values, generated/build-constrained files, expressions with inconsistent binding/availability, and type mismatches. Caller-local bindings and mapping extensions are later work.

## Audited starting state

Audit source was `/Users/alessandro/sources/semantic-editor` at `6c95d12` and was read-only, including uncommitted shared buffers. The requested six operations were absent in all worktrees. There is no signature/call-site mutation operation. Existing registry/wire patterns include body replacement, extraction, rename, movement, and batched edits; Go references/type information are available through adapter/session paths. `sourcebuf` and `Buffer.Apply` now provide a shared overlay in the source tree. Current LSP session does not establish command, interactive, or applyEdit support. ADR-0060 is pending in that tree and describes diagnostic-session ownership only.

## Candidate implementation paths

* Semantic proof: `internal/adapters/golang/gopls.go`, `internal/lsp/client.go`, `internal/symbol/resolver.go`; enumerate all direct references in sourcebuf view and type-check each proposed call.
* AST edits: implement focused logic in `internal/astedit/introduce_parameter.go`, drawing from `function.go`, `extract.go`, `rename` edit handling, and `move_construct.go`'s overlay planning patterns.
* Public surface: central registry via a dedicated `internal/operation/wire_signature.go` or `wire_engine_mutations.go`, using `operation.FileEditRes` if receipt fields can describe all outputs safely.
* Batch: `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, and `internal/mcp/batch.go`; never put a disk-planner/barrier in the operation. Plan all call files first and apply as one overlay change set.
* Structured parameter object `{name,type}` requires the shared closed recursive schema and presence-aware CLI JSON prerequisite. `argument` is a string containing one parsed Go expression, not opaque serialized data.

## Proposed public contract

Proposed MCP `semantic_introduce_parameter`, CLI `introduce-parameter`, Go-only initially. ADR-0055 `symbol` is the changed declaration target; `in_function` is not a replacement for that target. Parameter value shape must be typed, and result continuations must match input names/types.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target file | string | none | Existing declaration file. |
| `symbol` | target | string | none | Qualified function or method. |
| `parameters` | replacement | array of closed `ParameterMapping` objects | none | Complete ordered final parameter list. Each entry is `{source_index?: integer, parameter: ParameterSpec{name: string, type: string, variadic?: boolean}, argument?: Goexpr}`. Existing parameters use zero-based `source_index` in unchanged order/name/type; the one appended parameter omits it and supplies `argument`. |
| `discriminator` | owner-body selector | string | none | Exact source of one safe expression in the target function/method body to replace with the new parameter reference. |
| `construct_path` | disambiguator | string | omit if unique | Candidate path from a current ambiguity result; valid only for the selected file revision. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token for the selected declaration `file` only; the complete call/reference set is captured and validated separately. |

The ordered `parameters` array determines final signature order; no separate `position` selector is accepted. The selected `discriminator` (and optional `construct_path`) names the exact owner-body expression to replace with the new parameter reference; `expected_revision` binds that fragile selection to the selected declaration file. `symbol` already targets the containing function, so no redundant `in_function` owner selector is needed. Prove each new argument has the same binding and availability at every call. The selected owner-body expression must be safe to replace without changing control flow, and its type must match the new parameter. The call-site `argument` must have the same binding and availability at every call. Same binding, availability, and assignability are necessary but do not establish equivalence. Prove the supplied argument preserves the selected expression's value and evaluation timing/count. Release one admits only equivalent compile-time constants; reject mutable package-variable reads, function calls, and caller-local names. Return the changed declaration in a shared `continuations` entry with `file`, `symbol`, and `expected_revision`; call-site paths are diff evidence, not live selectors.

## Decision gates and semantic risks

1. Prove a complete reference set using typed resolution. A textual scan is not enough; refuse when build variants, generated files, external packages, reflection, or unresolved references could hide callers.
2. Verify new parameter type syntax, identifier uniqueness, receiver shadowing, and body bindings. Parameter insertion can shadow package names or alter resolution; check all affected scopes.
3. Check expression assignability, equal typed constant value, and preservation of observable evaluation timing/count at each call. Appending the argument evaluates it before the callee body; same-object resolution alone cannot justify moving a mutable read. For example, `toggle(); return packageFlag` must refuse selecting `packageFlag`, since a caller argument reads it before `toggle`. Broader expressions require an independent equivalence and ordering proof before expansion.
4. Exported functions and methods implementing interfaces need an explicit compatibility policy and all implementers/callers; keep them refused until that migration boundary is proven.
5. Gopls refactoring actions are candidates. Current session has no established command, interactive, or applyEdit support; a transport gate is required before adopting any action with those requirements. Do not use gopls CLI.
6. Match revision receipt creation to RQ-0024 after final normalization. Never return stale owner-body `discriminator` or `construct_path` values as follow-up targets.

## Ordered implementation handover

1. Build a read-only typed call-graph prototype for a private function. Classify supported direct calls and every refusal source; compare with gopls reference responses.
2. Define complete call-set and argument-expression semantics before adding a public registry entry. Use parser/type-checker validation on each caller against the sourcebuf overlay.
3. Implement a pure multi-file planner capturing original bytes and source ranges for declaration and calls. Preserve comments, unrelated formatting, and argument evaluation order.
4. Preflight all outputs, imports, collisions, expected revisions, and diagnostics inputs. Apply the entire output set in one `Buffer.Apply`; direct multi-file publication remains per-file atomic and can partially publish only with explicit reported count on failure.
5. Add typed wire schema and CLI JSON support after shared prerequisite; register the Go operation with `Batchable: true` and `BufferedBatch: true` for existing files.
6. In buffered batch, use sourcebuf for AST and LSP, retain one gopls process, defer verification/import cleanup and coalesce them at the boundary. Earlier successful edits remain if a later edit fails; there is no whole-batch rollback guarantee.
7. Add CLI/MCP help, capabilities, and txtar coverage. New callers/files cannot be created by this operation; use explicit `semantic_scaffold_file` first if a later workflow needs a new existing file.

## Required evidence and tests

Tests must invoke the public CLI command or MCP `semantic_batch`, assert user-visible output/diagnostics, and inspect disk/overlay state.

* Positive: append `bool` to a private function, select `true` in the body and provide the equivalent closed `true` argument, update calls across two existing files in one package, type-check, and consume the returned continuation in a standalone follow-up.
* Refusals: missing call site in a build-tagged file, interface implementation, method value, function variable, duplicate parameter, invalid type, non-assignable argument, side-effect-order hazard, mutable package-state read after a callee mutation, unequal constant value, stale revision, and generated source. Assert every file remains unchanged.
* All-output preflight: two caller files, second contains untransformable call; assert neither first caller nor declaration changes.
* Buffered sequence: earlier unsaved `semantic_insert_function` or body edit creates a call → introduce parameter → a later mutation resolves and uses the migrated signature; disk is unchanged before flush; final diff contains only final bytes; later failure retains earlier successful buffered edits.
* Retained server: prove call/reference resolution and the later mutation use one gopls process/session over the exact overlay, with no gopls CLI.
* Byte/no-op: preserve comments, line endings, argument spacing, and unrelated imports. If parameter and all calls already match exactly, produce an empty diff and no disk write; do not claim a mutation occurred.

## Completion decision

Resolve only after call-set completeness and argument-order proof hold for the stated boundary and the public batch sequence passes. Any need for project-wide/API migration requires a separate scope decision before expanding the operation.

## Concrete fixture sketches

Add `testdata/scripts/introduce_parameter_go.txtar` and `testdata/scripts/mcp_batch_introduce_parameter.txtar`. Use a private function called from two existing files plus one test file. Include a build-tagged unresolved caller refusal and a caller expression with visible side effects for the ordering test.

The batch must make an earlier unsaved caller edit, replace the selected owner-body expression with the new parameter reference, append the parameter, and prove a later verifier insertion resolves the migrated declaration. Omit `expected_revision` for this fresh overlay target. This JSON is a request-shape sketch, not receipt interpolation; a standalone follow-up can consume the returned continuation:

```json
{
  "edits": [
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Run() bool { return open(1) }"
      }
    },
    {
      "tool": "semantic_introduce_parameter",
      "params": {
        "file": "open.go",
        "symbol": "open",
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
              "name": "enabled",
              "type": "bool"
            },
            "argument": "true"
          }
        ],
        "discriminator": "true"
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Verify() bool { return open(1, true) }"
      }
    }
  ]
}
```

The owner starts with `return true`, so passing the same literal preserves its original value for every input; assert introduction replaces the selected `true` expression with `enabled` and changes callers to pass `true`. Snapshot the result before the verifier insertion. The later verifier insertion confirms the new signature resolves without overwriting the transformation. The original `id` argument remains at index zero. Import organization occurs once at the segment boundary. The standalone `exec semedit introduce-parameter ...` case checks all callers, imports, diagnostics, and final compilation with the exact expected files.

Refusal txtars must cover an uneditable second caller, a function-value use, interface implementation, stale revision, duplicate parameter, nonassignable argument, and argument-order hazard such as changing call order around a side-effecting argument. Add `toggle(); return packageFlag` with a proposed `packageFlag` caller argument to prove that same binding and side-effect-free reads still refuse when evaluation timing changes; also refuse unequal constants. Each refusal asserts the target and every caller remain unchanged.

A buffer-focused test should assert the call-set and argument type-check complete before `Buffer.Apply`, then inspect the owner-body replacement and all caller outputs and original disk bytes before flush; a separate stale-preimage case changes a captured input before `Buffer.Apply`/`Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published. Prove one gopls process/session across earlier edit, call discovery, and later lookup; one coalesced verification/import pass; and retention of earlier successful operations when a subsequent batch item fails. Avoid any assertion of whole-batch rollback after final per-file publication begins.

## Expansion milestone and artifact checklist

Milestone two may accept caller-local or mutable expressions only with per-call binding/availability, equal-value, and observable evaluation timing/count proofs; multiple added parameters and other mappings remain gated by independent argument-order evidence; milestone three may consider exported APIs or interface implementers under a migration contract. Each supported language form must be advertised through backend capability metadata and exercised by txtar. Do not infer a default argument from nearby syntax.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_introduce_parameter`:**

> Use this tool instead of editing a body, signature, and callers separately to replace one supported Go body expression with a new parameter. Supply `discriminator` and the complete ordered `parameters` mapping with an appended argument. Release one accepts equivalent compile-time constants and refuses changed values or evaluation timing. Pass the surviving callable's `file` and `symbol` from `continuations` to `semantic_rename` or `semantic_replace_body`. The replaced body expression is historical evidence, not a live selector.

**Parameter description requirements:**

`symbol`: Target: existing qualified callable copied from `semantic_lookup.symbol` with its `file`. `discriminator`: Selector: exact current body expression, copied only from proposed `semantic_inspect_symbol.discriminator` for this current callable expression and revision. `parameters`: Replacement: final ordered mapping. Its new `argument` is a caller-supplied Go expression, never a guessed default from the removed body text.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
