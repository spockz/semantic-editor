<!-- WHY: This handover defines the evidence needed for a bounded Go function-inline operation while keeping transport and evaluation-order decisions open. -->
# RQ-0042: Go Inline Function Handover

* **Status**: Open
* **Category**: Product & Architecture
* **Date**: 2026-09-30

## Purpose and shared contract

Plan a Go-first `semantic_inline_function` operation that replaces a selected call with a safe equivalent body expansion and removes the function declaration only when the requested scope proves it unused. This is a handover plan, not an accepted decision. Follow [RQ-0024 § Shared Handover Contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract).

Sibling investigations: [change signature](RQ-0041-go-change-signature.md), [extract constant](RQ-0043-go-extract-constant.md), [extract interface](RQ-0044-go-extract-interface.md), [introduce parameter](RQ-0045-go-introduce-parameter.md), and [introduce parameter object](RQ-0046-go-introduce-parameter-object.md).

## Observable outcome and boundaries

Success replaces exactly one selected direct call derived from a resolved callee body and preserves evaluation/order. The primary `file` + `symbol` target pair always identifies the callee declaration while it remains live. Return the edited caller/owner as a `continuations` array entry `{file, symbol, expected_revision}`. If deletion is explicitly requested and proven safe, omit the deleted callee from live continuations; record its identity only in separate historical data. Never substitute the owner name into the callee `symbol` field.

Bounded first release: one target non-recursive, non-generic, non-variadic free function called directly once in an existing Go file; caller and callee may be in different existing files within the same package; function body is a single return expression or a small straight-line statement list with a final return; no escaping/address-taking, defer/recover, closures, labels, control flow, named-result dependence, or observable ordering hazard. Preserve the callee by default. Declaration deletion is a separately gated option, default false, and only when a same-package complete reference proof succeeds. No multi-call workspace inlining, method receiver rewrites, framework patterns, or new-file creation.

## Audited starting state

Read-only audit used `/Users/alessandro/sources/semantic-editor` at `6c95d12`, including shared uncommitted state. The six requested operations were absent across worktrees. Existing `internal/astedit/inline.go` and `wire_engine_inline.go` implement inline-variable only; they are not evidence of function inlining. Existing extraction, move, and removal are bounded. A central registry and buffered Go batch path exist in the source tree. Current LSP client does not establish support for code-action command execution, interactive actions, or `workspace/applyEdit`; an ADR-0060 draft is present in that audited tree and remains pending context.

## Candidate implementation paths

* Public wire: add a distinct operation definition alongside `wire_engine_inline.go` or a dedicated `wire_inline_function.go`; pattern the typed request/receipt after `wire_extract.go` and existing inline-variable operation.
* AST planning: `internal/astedit/inline.go`, `extract.go`, `body.go`, `construct.go`, `symbol/resolver.go`; use `gopls.go` for typed call/reference evidence. Keep function-specific semantics separate from inline-variable semantics.
* Buffer: `internal/sourcebuf/buffer.go` plus `internal/mcp/buffers.go` and `batch.go`; calculate all file edits against one overlay revision and call `Buffer.Apply` only after all output preimages validate.
* Schema: registry and MCP schema generation in `internal/operation/operation.go`, `parse.go`, and MCP package; use shared recursive schemas, presence-aware CLI JSON, and `SourceFields` name/type checks from RQ-0024.

## Proposed public contract

Proposed names: MCP `semantic_inline_function`, CLI `inline-function`; Go-only initially. ADR-0055 reserves `file` + `symbol` for the callee declaration target. Select the call using a closed `call_site` object so caller and callee can be in distinct files; same-file primitive selectors may be a documented shorthand only if they do not blur these roles.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target file | string | none | Existing Go file declaring the callee. |
| `symbol` | target function | string | none | Qualified callee declaration; resolves the primary target. |
| `call_site` | selector | closed object | none | `{file: string, in_function: string, discriminator: string, construct_path?: string, expected_revision?: string}` selects one call in an existing caller file. The nested revision binds the caller selection. |
| `remove_declaration` | policy | boolean | false | Request deletion after proving no remaining references. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision from lookup/receipt; no stale path reuse. |

The primary receipt continuation identifies the callee using `file`, `symbol`, and its final `expected_revision` while the callee remains. A separate `continuations` array entry identifies the surviving owner function. When deletion is requested, omit the deleted callee as a live target, return only the owner continuation, and place the removed callee under explicitly historical data. Resolve all continuations after normalization.

## Decision gates and semantic risks

1. Code actions are candidates only. First inspect gopls action kinds, returned edit shape, and whether action resolves without command, user interaction, or `applyEdit`. Current session support does not include those behaviors. A transport gate must implement and test them before this route is adopted; no gopls CLI invocation is acceptable evidence.
2. Prove callee resolution and body availability from the same overlay. Refuse recursion, generic instantiation ambiguity, method values, interface dispatch, function values, external definitions, or build variants not covered by analysis.
3. Preserve Go argument evaluation count and order. Parameter substitution is unsafe when an argument is used more than once, moved across side effects, shadowed, or captured. Require temporaries only if a semantics-preserving typed plan is available; otherwise refuse.
4. Handle return values, naked returns, named results, defer, panic/recover, and local variable scope explicitly. The first release should accept only one unambiguous return shape.
5. Resolve the declaration-deletion receipt after all replacements and normalization. Refuse deletion if any references remain or completeness is unknown.

## Ordered implementation handover

1. Read the existing inline-variable implementation and gopls request/session lifecycle; capture actual supported protocol methods and code-action payloads.
2. Build a read-only call selector over sourcebuf, using `call_site.file`, `call_site.in_function`, `call_site.discriminator`, optional `call_site.construct_path`, and `call_site.expected_revision`; return structured ambiguity/refusal evidence.
3. Prototype the narrow straight-line case with `go/ast` and type information. Produce edits in memory, preserving comments and unaffected bytes.
4. Preflight every output, collision, scope binding, call evaluation, and optional reference deletion. Apply all output bytes to the overlay in one `Buffer.Apply`; no operation-owned disk barrier/planner.
5. Implement typed registry/schema and truthful continuation receipts. Mark `Batchable` and `BufferedBatch` true only when overlay semantics are proven; both are required for existing Go targets.
6. Batch execution must defer verification/import organization, retain earlier successful edits on later failure, and publish only final buffered bytes. Per-file atomic publication is guaranteed, whole-batch rollback is not.
7. Add CLI/MCP documentation and public txtar coverage. If a new helper file becomes necessary, require explicit `semantic_scaffold_file` first until sourcebuf resource support exists.

## Required evidence and tests

Every public test runs the CLI or public MCP `semantic_batch` contract and asserts diagnostics and resulting bytes.

* Positive: inline `return adjust(x)` for a local pure helper with one direct caller; assert expression/body expansion, intact callee by default, and continuations usable in a later lookup/edit.
* Refusal: multi-use side-effecting argument, recursion, indirect/function-value call, generic ambiguity, `defer`, branch/loop body, named result or comment-boundary case outside the supported subset, stale revision, and requested deletion with one remaining reference. Assert no changed bytes.
* Preflight: caller and target are in different files; make one secondary output uneditable/stale and prove no output changed.
* Buffered sequence: prior unsaved body replacement → inline function → later mutation of containing function; later operation sees overlay text, disk stays original until final flush, no intermediate publication occurs, and later failure retains only earlier successful overlay changes.
* Retained one-process proof: count gopls process/session identity over earlier lookup, inline planning, and later lookup; all use the same live session and overlay. Do not use `gopls` CLI.
* Byte preservation/no-op: retain comments and source bytes beyond exact replacements; inlining that produces byte-identical source yields an empty diff and no publication. Verify imports are coalesced once at segment end.

## Completion decision

Keep the question open until call-site semantics, transport capability, and overlay proof are demonstrated. Expand from one call only after the same evaluation-order and reference guarantees are independently established.

## Concrete fixture sketches

Add `testdata/scripts/inline_function_go.txtar` for standalone CLI use and `testdata/scripts/mcp_batch_inline_function.txtar` for the shared-buffer public path. The fixture should place the caller and helper in separate existing files, with an unrelated oddly formatted sibling file whose bytes must remain unchanged.

The batch sequence should establish a call in an earlier unsaved edit, inline that exact call, snapshot the owner body immediately, then issue a later mutation that adds a separate verifier function against the statically known owner. This JSON is a request-shape sketch, not receipt interpolation; a standalone follow-up can consume the returned continuation:

```json
{
  "edits": [
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Run(x int) int { return adjust(x) }"
      }
    },
    {
      "tool": "semantic_inline_function",
      "params": {
        "file": "helper.go",
        "symbol": "adjust",
        "call_site": {
          "file": "caller.go",
          "in_function": "Run",
          "discriminator": "adjust(x)"
        }
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Verify() int { return Run(3) }"
      }
    }
  ]
}
```

Provide `helper.go` with the accepted straight-line body; assert the final diff keeps `adjust` by default and the caller contains the equivalent body. The CLI txtar must invoke `semedit inline-function` and compare expected source bytes, not merely report a successful exit.

Add a public refusal fixture for `adjust(next())` where inlining would duplicate or reorder evaluation, one recursive helper, and an indirect function value. Assert each response identifies the unsupported reason and all source files remain byte-identical. A deletion-request fixture with a second call site must refuse deletion while leaving the inline itself unapplied unless the contract explicitly supports inline-without-delete as a separate successful action.

A focused buffered test must snapshot `Run` from the overlay immediately after inlining and before any later mutation; assert that the selected `adjust(x)` call is gone and the equivalent body is present. Then prove the later verifier insertion resolves `Run` through the same gopls process/session. Disk remains original before flush. A separate stale-preimage test changes a captured input before `Buffer.Apply` or `Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published. Assert one coalesced verification/import pass and that a later intentional batch failure retains prior successful buffered changes without publishing intermediate bytes.

## Expansion milestone and artifact checklist

Milestone two may support callers across packages only after complete reference and visibility proofs; support multiple call sites only after distinct argument evaluation traces are preserved; milestone three may consider methods or cross-file declaration removal after complete references are proven. Every expansion needs a public txtar pair for success/refusal and must update backend capability metadata. The handover is complete when no response suggests the removed helper remains a live target.

## Expected research artifacts

Record a call-body-form matrix with evaluation-order notes, reference completeness evidence for declaration deletion, and the gopls action/transport trace. Preserve the distinction between a candidate action's availability and proven semantic correctness.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_inline_function`:**

> Use this tool instead of copying a helper body into its caller to inline one supported direct Go call. `file` and `symbol` identify the callee; `call_site` selects the caller independently. The helper remains unless `remove_declaration=true` passes complete-reference checks. Pass a surviving callable's `file` and `symbol` from `continuations` to `semantic_rename` or `semantic_replace_body`. A deleted helper is historical evidence and is never offered as a live next-call target.

**Parameter description requirements:**

`symbol`: Target: qualified callee; copy verbatim from `semantic_lookup.symbol` with its `file`. `call_site`: Selector: one current caller location; copy the complete same-shaped `call_site` projection from proposed `semantic_inspect_symbol.call_site` only after that same-shaped producer is implemented. Do not construct it by renaming a callee receipt's fields. `remove_declaration`: Policy: optional Boolean, default false; deletion is part of the same validated operation.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
