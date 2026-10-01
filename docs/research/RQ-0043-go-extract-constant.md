<!-- WHY: This handover scopes Go extract-constant research around exact expression selection, constant representability, and safe overlay publication. -->
# RQ-0043: Go Extract Constant Handover

* **Status**: Open
* **Category**: Product & Architecture
* **Date**: 2026-09-30

## Purpose and shared contract

Plan `semantic_extract_constant` for one selected Go constant expression, replacing it with a package-level constant while preserving meaning and source locality. The plan is a research handover, not an accepted contract. Apply [RQ-0024 § Shared Handover Contract](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#shared-handover-contract).

Sibling handovers: [change signature](RQ-0041-go-change-signature.md), [inline function](RQ-0042-go-inline-function.md), [extract interface](RQ-0044-go-extract-interface.md), [introduce parameter](RQ-0045-go-introduce-parameter.md), [introduce parameter object](RQ-0046-go-introduce-parameter-object.md).

## Observable outcome and boundaries

A successful operation inserts a named `const` declaration in the selected existing file, replaces exactly one selected expression with its identifier, and validates the edited package. Return the new constant through a `continuations` array entry `{file, symbol, expected_revision}` resolved after formatting/import normalization. Keep historical planned names separate from live continuation targets. It does not return the old expression selector as an active continuation after the replacement.

First release: one unambiguous compile-time constant expression inside a function or method in one existing file; infer the exact constant type/value with Go type information; insert in package scope near the file's other declarations; replace one exact AST expression. Refuse `iota`-dependent or context-sensitive constants, expressions whose type/value cannot be represented exactly, duplicate identifier collisions, directives/comments that cannot be safely anchored, cross-file use, and any expression with runtime evaluation. No extraction of variables, multiple occurrences, enum generation, generated files, or new-file creation.

## Current implementation checkpoint (2026-10-01)

`extract-constant` and `semantic_extract_constant` are registered for Go. The pure planner accepts only resolved built-in basic constant types and refuses named/non-basic types, local-constant dependencies, `iota`, runtime expressions, collisions, ambiguous exact-text selections, and selections with comments. It preserves an explicit built-in type where the original expression is typed and leaves untyped constants untyped. Public CLI and buffered MCP batch fixtures exercise success and refusal.

The operation still lacks the shared `expected_revision` input and typed finalized continuation projection. Its package loader observes the selected package directory but does not capture and validate every consulted dependency/configuration input. The shared output-projector and complete read-set APIs remain required before this handover can close.

## Audited starting state

The authoritative source tree `/Users/alessandro/sources/semantic-editor` at `6c95d12` was audited read-only, including uncommitted shared buffers. At that audit point the operation was absent. The current implementation checkpoint above supersedes that historical baseline. The registry, sourcebuf overlays, batch executor, and single-process gopls sessions are present in the audited tree. Current LSP client does not prove command, interactive, or applyEdit support; a pending ADR-0060 in that source tree concerns diagnostic sessions only.

## Candidate implementation paths

* AST edit: new focused `internal/astedit/extract_constant.go`, following `extract_variable.go`, `insert.go`, `decl.go`, and `comment_anchor.go`; avoid whole-file formatting rewrites.
* Semantic analysis: `internal/adapters/golang/gopls.go`, `internal/lsp/client.go`, and `internal/symbol/resolver.go`; evaluate gopls code actions as candidates only after transport shape is proven.
* Wire: a distinct operation definition in `internal/operation/wire_extract.go` or sibling. Existing `ExtractVariableReq` provides selector conventions but has a different local-variable semantics.
* Buffered execution: `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, `internal/mcp/batch.go`. Plan both the declaration insertion and replacement as one complete output and call `Buffer.Apply` once.
* Schema gate: shared recursive object/array schema plus presence-aware CLI JSON and type-checked `SourceFields`; do not use opaque JSON strings.

## Proposed public contract

Proposed MCP name `semantic_extract_constant`, CLI `extract-constant`, Go backend only. Use ADR-0055 roles: `symbol` names the new constant, `in_function` scopes the selected expression, and `discriminator` / `construct_path` select the exact expression. The owner selector is not a target. `expected_revision` is optional for a fresh target and binds the selected `file` only; capture and validate the owner selection and complete output preimages separately.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target file | string | none | Existing `.go` file where expression and declaration will live. |
| `symbol` | created target | string | none | New package-level identifier, e.g. `maxRetries`. |
| `in_function` | scope selector | string | none | Qualified owner function or method. |
| `discriminator` | construct selector | string | none | Exact source spelling of one candidate constant expression. |
| `construct_path` | disambiguator | string | omit if unique | Current candidate path, valid only for bound revision. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token for the selected `file` only; bind expression selection to that file revision and validate all planned outputs separately. |

Receipt fields are the surviving `file`, new `symbol`, and final revision, with exact field types matching their inputs. Do not echo `discriminator` or path as a reusable live selection after mutation. Source-field descriptions must name their producing tools and schema validation must compare name and type. Use the shared `continuations` array shape rather than an operation-specific `generated` object.

## Decision gates and risks

1. Determine whether a selected expression is a Go constant expression in its exact context, including untyped constants, conversions, shifts, array lengths, and `iota`. Preserve assignability and default typing; reject any uncertain case.
2. Choose declaration placement and comment anchoring that preserve build directives and file-level comments. Existing import-group or declaration layout must not be normalized unnecessarily.
3. Expression selection must be syntax-backed and revision-bound. If matching by exact source text produces multiple candidates, require a fresh candidate path; never reuse a stale path.
4. Code actions may be investigated, but any action needing command execution, user interaction, or `workspace/applyEdit` is blocked until transport support is implemented and tested. Do not substitute gopls CLI.
5. Calculate both replacement and insertion before `Buffer.Apply`; validate collision, type-check, and every output preimage first. Empty or same-name no-op behavior must be explicit.

## Ordered implementation handover

1. Map current extract-variable selection and declaration anchoring behavior; inspect gopls constant metadata from the retained server.
2. Define a read-only expression classifier that returns constant value and exact type information from the shared overlay. Record supported constant forms before mutation work.
3. Implement a pure planner with original byte ranges and two edits. Preserve comments, line endings, and unrelated file bytes; handle same declaration/file position overlaps explicitly.
4. Run syntax/type/collision checks and revision validation against all outputs. On success call `Buffer.Apply` once with all outputs; direct mode stages atomic per-file writes, with no multi-file rollback promise.
5. Add the registry contract, schema and descriptions after the shared typed-schema gate. Set both `Batchable` and `BufferedBatch` for existing Go files.
6. Make buffered batches defer imports/verification and coalesce them once per segment. Later edits must see the new identifier through sourcebuf and the same gopls process.
7. Add command help, MCP description, capability metadata, and CLI txtar cases. If creation of a new destination file is ever required, first invoke explicit `semantic_scaffold_file`; sourcebuf new-resource support is not designed.

## Required evidence and tests

Tests must exercise public CLI output or public MCP batch calls and inspect exact workspace state.

* Positive: extract `3 * 60` to a uniquely named `const retryWindow = 3 * 60` in an existing function; assert typed equivalence, declaration placement, returned target continuation, and follow-up lookup/edit using that receipt.
* Refusals: runtime expression, `iota` outside a proven safe declaration context, name collision, duplicate selection, non-constant conversion, stale revision, malformed type state, and unanchorable directive/comment. Assert diagnostics and byte-identical source.
* All-output preflight: include a second planned output (if the chosen type/import/declaration plan requires it), make that output stale, and assert neither the expression nor declaration changes. If first release is strictly one-file, still prove the complete output set is prevalidated before either edit applies.
* Buffered sequence: unsaved `semantic_replace_body` creates the expression → extract constant → a later mutation resolves and uses the new symbol. Assert the overlay is visible, disk bytes stay unchanged until final flush, and a separate post-batch lookup/edit consumes the returned continuation; a later failure keeps prior successful buffered edits only.
* Retained server: prove the same one-process gopls session sees the earlier edit, resolves the expression, and supports the later mutation; perform any read-only lookup after the batch because `semantic_lookup` is not batchable. Do not use gopls CLI.
* Byte/no-op: preserve comments, CRLF/LF choice, directives, imports, and unrelated bytes. A selected expression already equal to a collision-free constant reference must either be a documented refusal or produce an empty diff with no write; do not report a successful mutation with an empty/ambiguous target.

## Completion decision

Resolve the question only after the supported constant subset is explicit, exact type preservation is demonstrated, and the public buffered txtar sequence passes. Expand to multiple occurrences or files only with a new completeness and preflight proof.

## Concrete fixture sketches

Add `testdata/scripts/extract_constant_go.txtar` plus `testdata/scripts/mcp_batch_extract_constant.txtar`. Use one package with an expression nested in a direct return, a name collision case, a nonconstant expression, and an unrelated source file with deliberately non-gofmt spacing.

The buffered sequence should create the selected expression in an earlier unsaved edit, extract it, then apply a later mutation through normal resolution. Use a standalone post-batch lookup to consume the returned continuation:

```json
{
  "edits": [
    {
      "tool": "semantic_replace_body",
      "params": {
        "file": "retry.go",
        "symbol": "Retry",
        "body": "return 3 * 60"
      }
    },
    {
      "tool": "semantic_extract_constant",
      "params": {
        "file": "retry.go",
        "symbol": "retryWindow",
        "in_function": "Retry",
        "discriminator": "3 * 60"
      }
    },
    {
      "tool": "semantic_replace_body",
      "params": {
        "file": "retry.go",
        "symbol": "Retry",
        "body": "return int(retryWindow)"
      }
    }
  ]
}
```

The fixture must prove the conversion remains assignable for the function's actual result type; adjust the exact expression/type so the expected source compiles. `exec semedit extract-constant ...` is the standalone public CLI test. Compare the declaration location, exact replacement, diagnostics, and unrelated file bytes.

Add refusal cases for runtime calls, `iota` in an unsupported context, a duplicate name, two identical expression candidates without `construct_path`, and stale `expected_revision`. Assert no source mutation for every refusal. A same-expression already-constant case must specify either a structured no-op with empty diff or explicit refusal.

The batch buffer test must inspect original disk bytes before final flush and overlay bytes after extraction; a separate stale-preimage test changes a captured input before `Buffer.Apply`/`Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published; a later lookup/edit must use the new symbol without restarting gopls. Assert same one-process session, one final verification/import pass, complete output preflight before `Buffer.Apply`, and earlier successful edits retained on a later operation error. Publication remains atomic per file only.

## Expansion milestone and artifact checklist

Milestone two may permit multiple exact expression occurrences only with a stable occurrence selector and atomic preflight; milestone three may support existing destination files only with package and visibility checks. Keep `iota`, generated declarations, and cross-package promotion out until separately evidenced. The handover is complete when the result's new constant resolves from its advertised `file`/`symbol` pair.

## Expected research artifacts

Record the Go constant-expression forms accepted and refused, typed-value preservation examples, and exact comment/directive placement behavior. Include one gopls metadata trace only if it contributes evidence beyond the AST/type checker.

## Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_constant`:**

> Use this tool instead of copying an expression into a new declaration to extract one supported Go constant expression. Choose a fresh simple `symbol`; `in_function`, `discriminator`, and an optional revision-bound `construct_path` select the existing expression. The engine preserves contextual typing or refuses the extraction. Pass the created constant's live `file` and `symbol` to `semantic_rename` or `semantic_inspect_symbol`. Replaced expression text and old paths are evidence, not selectors for another extraction.

**Parameter description requirements:**

`symbol`: Target: caller-chosen new constant identifier such as `maxRetries`; no preceding existing-symbol result supplies this new name. `in_function` and `discriminator`: Selectors: copy `in_function` from a live `semantic_extract_function.in_function` or proposed `semantic_inspect_symbol.in_function`, and `discriminator` from proposed `semantic_inspect_symbol.discriminator` for that current expression; do not copy `semantic_lookup.symbol` or replaced expression evidence. `construct_path` is paired with its selected-file `expected_revision`.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.
