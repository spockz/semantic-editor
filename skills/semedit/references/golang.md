# Go Semantic Refactoring Guide

MCP results follow the [shared output policy](../SKILL.md#copy-typed-results-and-respect-publication).
Use typed continuations with their exact revisions, and retain failure and
publication state when choosing follow-up calls. Structured results can be
model-visible; text keeps exact next-step guards and names structured fields
without repeating source, inventories, diagnostic pages, diffs, or logs.
For diagnostic details, copy the response's `diagnostic_report.arguments` into
`semantic_diagnostics`, choose a category/filter and page size, then follow
`next_offset`. Pages reuse the captured report; a deferred/unavailable count
does not indicate a clean workspace. Use verification only for a fresh report.
Rename returns `active_count` and diagnostic changes rather than every existing
workspace warning; request `semantic_verify` for a full diagnostic inventory.

Use `semantic_metrics` with `limit: 1` after a slow Go operation to inspect
`records[].metrics.phases`: `gopls.request.textDocument/diagnostic`,
`gopls.request.textDocument/prepareRename`, and `gopls.request.textDocument/rename`
identify call types with counts and cumulative client elapsed milliseconds.
`gopls.notify.*` measures outbound notification writes without a server reply.
`gopls.send.<method>` measures serialization, queuing, and transport completion;
`gopls.wait.<method>` measures response waiting after a successful send.
Nested phase durations overlap, and a request's wait is not gopls CPU time.
Exact `go.*` phases additionally separate environment/package loading, lock
waits, source synchronization, symbol resolution, revision validation, and
publication. A host-enabled Go execution trace shows these as ordered regions
with goroutine blocking stacks; trace collection is off by default.
Fingerprint children `go.scope.embed_patterns` and `go.scope.hash_inputs`
separate embed-directive scanning from structural input hashing.

This reference provides concrete patterns and workflows for executing Go refactorings with `semedit` primitives. Read this document only when refactoring or modifying Go source files.

The default MCP base catalog includes `semantic_workspace_outline`, `semantic_change_signature`, `semantic_extract_function`, and `semantic_set_doc_comment`; `--tool-profile full` exposes all registered tools. The execution profile `mutations-only` independently removes read-only tools, including workspace outline.

Install the CLI with `go install github.com/spockz/semantic-editor/cmd/semedit@latest`, then install `gopls` with `go install golang.org/x/tools/gopls@latest`. From a checkout, build the executable with `go install ./cmd/semedit` at the repository root.

When scripting semedit harness setup from a Go project, use `--json` with `install`, `integration status`, or `uninstall` for structured administrative output. Semantic operation output is unchanged.

Normal semantic operations inherit `GOENV`, `GOMODCACHE`, `GOCACHE`, `GOPATH`, `GOWORK`, temporary-directory settings, and gopls cache settings. They reuse the user's existing Go state and workspace configuration. Module files remain read-only for ordinary operations; use the explicit dependency operation when changing dependencies. MCP `--go-base-dir` enables isolated state intentionally, and benchmark fixtures opt into it. Do not configure `.scratch/go` caches for ordinary projects merely to use semedit.

Benchmark RAM fixtures can combine `mcp --go-base-dir <fixture>/.scratch/go`
with `--go-module-cache <persistent-ssd-cache>`. The latter selects modules
directly, without cache copying; build and gopls state stay private to the
fixture. Set benchmark dependency access offline and prepopulate required
modules before timing. Missing dependencies must fail setup. A populated module
cache is compatible with prewarming an empty job build cache. This runtime
configuration does not authorize dependency edits or workspace-manifest changes.

## Target suggestions

When a supported Go `lookup`, `rename`, `inspect_symbol`, `find_references`, `replace_body`,
`remove_construct`, `replace_construct`, `change_signature`, `extract_function`,
`extract_variable`, `set_doc_comment`, `insert_field`, `extract_type`, or
keyed-element/field `insert_construct`
target fails, inspect the structured `suggestions` field when present. It is
bounded to three alternatives that pass the requested file/package scope and
the operation's native target checks. Retry the original operation with the
candidate `arguments`; do not apply a candidate from spelling similarity
alone. A candidate's file, one-based location, kind, optional signature, and
reason provide review context. In a batch, suggestions describe the current
working overlay; a publication failure clears them because that overlay may no
longer exist. The original error remains authoritative until a retry succeeds.

`SEMEDIT_SUGGESTIONS` accepts `off`, `generic` (default), `codanna`, or
`codanna,generic`. Generic mode uses native Go declaration enumeration.
Codanna modes use an installed CLI on `PATH`, optionally selected with
`SEMEDIT_CODANNA_BIN`. semedit does not install Codanna, models, or
dependencies, and does not explicitly run an index or rebuild command. A
Codanna search may maintain its own index/cache during normal CLI startup.
Provider output is only a candidate source and is combined with generic
candidates in `codanna,generic` mode, then revalidated against native source.
Extraction alternatives change only the existing `in_function` selector; a fresh
helper or variable name is not a spelling-recovery target.
Ordinary native target inspection is capped at 1.5 seconds; rename, extraction, and reference-search
validation may take up to 5 seconds for compiler or gopls startup. Codanna retrieval
is capped at 500 ms. Ranking examines at most 50 candidates, operation
selectors are limited to 256 bytes and ranking selectors to 256 runes, and at
most three results are returned.
See [the suggestion contract](../../../docs/symbol-suggestions.md)
for the complete scope and lifecycle boundary.

---

## Workspace discovery

Start with `semantic_workspace_outline`. Auto mode aggregates supported Go and Java ecosystems in mixed repositories. Pass `language: "go"` to discover only Go packages and modules. A `path` such as `internal/api` scopes package discovery while resolving its nearest enclosing module; a nested module path reports that module and any modules below it rather than unrelated root modules.

Go discovery reads `go.work` use directives without changing the workspace manifest. Module records include documentation from comments before the `module` directive, falling back to root package documentation, and counts of packages in the requested scope. Package records carry `module` and `language` ownership. Use these records to choose the exact source scope for subsequent lookup and edits; do not infer ownership from directory names alone.

MCP workspace outline text carries a concise summary and critical metadata; package
and module records appear once in `structuredContent`. Declaration outlines,
lookup, and references instead use compact navigation listings. Paths resolve from
the active workspace root. Declaration lines are one-based and inclusive; reference
positions identify occurrences. Copy qualified method selectors verbatim. Use
ordinary file reading for disk source or `semantic_inspect_symbol` for source and
guarded revision continuations. Overlay labels identify unpublished contents.

## Publication and verification

Automatic Go diagnostics reuse findings only after validating their input revision.
Changed packages and their reverse dependencies are refreshed; unknown coverage
falls back to a full scan. `semantic_verify` explicitly requests full analysis.
Ordinary source creation and deletion synchronize the retained gopls process.
Rename may re-resolve once after a conflict before any file is written. After
partial publication, inspect the published paths and diff before continuing;
the operation does not replay the remaining stale edit plan.

Standalone AST mutations stage source edits, formatting, and import organization
in one working buffer before atomic publication. In a batch, children share that
buffer and return `applied_working` receipts without live follow-up selectors.
Use the finalized standalone result or perform discovery after batch publication
before copying a continuation. A failed batch can preserve earlier edits; its
persisted diff is authoritative. Successful editing operations already verify
diagnostics, so a separate `semantic_verify` is needed only for a new explicit
verification request or subsequent external changes.

For a refactoring that needs several turns, start `semantic_stage` in the selected
Go workspace. Successful mutations return a native `stage_receipt`; reads use the
same overlay and source files remain unchanged until `semantic_commit`. During an
API migration, use `semantic_stage_drop` with a returned `stage_receipt.step_id`
to remove a mistaken edit and replay the surviving steps. Use
`semantic_stage_abort` to discard the entire tentative migration. A conflicting input, failed replay, or
budget rejection preserves accepted edits. One module cannot host competing
stages, and workspace-root changes cannot bypass an active journal.

Zero-write commit failures retain the stage. Partial publication reports the
actual persisted diff and freezes further edits until abort. Complete publication
consumes the stage; later verification problems report `applied_with_diagnostics`.
Inspect publication evidence before retrying. `skip_hooks` on batch, verification,
or commit skips external commands while preserving formatting and Go diagnostics.
Stage budgets are policy limits, not promises about heap usage or latency.

Use `semantic_inspect_construct` to obtain matching normalized candidates when a
nested selector is ambiguous. Copy `construct_path` and the corresponding
`expected_selection_revision`; a stale revision refuses the replacement and
returns current choices. This guard is optional and covers the matching candidate
set, not the whole file. Control selectors accept formatting differences.

Lookup locations use captured source, including staged bytes. Public line and
column values are one-based UTF-16; neutral ranges are zero-based UTF-16. Go token
positions remain byte-based internally, so do not mix those coordinate systems.
Directory lookup searches the selected package. Whole method replacement with
`kind: "function"` accepts a qualified selector or infers the receiver from the
complete method source; receiver identity and pointer/value form stay protected.

Brace-wrapped replacement bodies and plain-text scaffold purposes are accepted.
Nested statement anchors, multiple package initializers, safe grouped-member
moves, and explicitly requested nested-module imports retain their structural
checks. Ordinary Go commands preserve manifests with `-mod=readonly`; use the
explicit dependency operation when changing module requirements.

Generic `kind: "field"` insertion uses the typed field planner: its owner must
be a private non-generic struct, and package-overlay validation must prove
layout compatibility. An exported owner, duplicate field, or unsupported
positional literal is refused before publication.

## 1. Macro Refactorings in Go

`semedit` core AST operations for Go are indexed in the primary `SKILL.md` table. This reference defines idiomatic macro refactoring patterns composed from those primitives.

---

## 2. Go Refactoring Patterns

### Pattern A: Extract State to Composed Type & Delegate

When decomposing a large struct with grouped fields into a nested composition:

1. **Preflight & Inspection**:
   - Use `semantic_outline` to inspect struct layout and `semantic_lookup` / `semantic_inspect_symbol` to examine field coordinates, types, and existing methods.
2. **Extract State**:
   - Use `semantic_extract_type` to move direct private fields from the owner struct into a new private composed struct in the same file.
   - Use `semantic_insert_field` if supplemental tracking fields are needed on either struct.
3. **Move or Wrap Methods**:
   - Use `semantic_insert_function` to introduce delegating wrapper methods on the outer struct.
   - Use `semantic_replace_construct` to update inner method receiver calls.
4. **Execute in Batch & Verify**:
   - Combine field extraction and method insertions inside `semantic_batch` to prevent intermediate broken compilation.
   - Run `semantic_verify` to validate syntax and package diagnostics.

---

### Pattern B: Function Signature Modernization & Parameter Objects

When a function exceeds parameter limits or requires expanded configuration:

1. **Call-Site Discovery**:
   - Use `semantic_find_references` to locate typed references across the module; inspect enclosing declarations to distinguish calls from other uses.
2. **Transform Parameters**:
   - Use `semantic_introduce_parameter` to add a single new argument with caller updates.
   - Use `semantic_introduce_parameter_object` to bundle multiple parameters into a typed struct, converting callers to construct keyed literals.
   - Use `semantic_change_signature` to reorder parameters, rename bindings, or adjust return types.
   - Use `semantic_rename` if function or method identifiers are renamed across the workspace.
3. **Verify**:
   - Validate with `semantic_verify`.

---

### Pattern C: Interface Extraction and Contract Conformance

When decoupling components behind an interface:

1. **Extract Interface**:
   - Call `semantic_extract_interface` with the target file, new interface name, and selected methods.
2. **Implement Contract**:
   - Use `semantic_implement_contract` to stub missing interface methods on a concrete type with `panic_stub`.
3. **Implement Method Bodies**:
   - Use `semantic_replace_body` to replace generated placeholder stubs with complete logic.
4. **Verify**:
   - Validate with `semantic_verify`.

---

### Pattern D: Expression Extraction and Inlining

When simplifying complex expressions or cleaning up temporary variables:

1. **Extract Values & Expressions**:
   - Use `semantic_extract_variable` to extract repeated function call arguments into a preceding local variable.
   - Use `semantic_extract_constant` to promote repeated literal expressions to package-level constants.
   - Use `semantic_extract_function` to extract contiguous call statements into a local helper function.
2. **Inline Values & Functions**:
   - Use `semantic_inline_variable` to inline single-use local variables into adjacent statements.
   - Use `semantic_inline_function` to eliminate trivial literal-return helper functions.
3. **Verify**:
   - Confirm clean diagnostics via `semantic_verify`.

---

### Pattern E: Struct, Literal, & Declaration Lifecycles

When evolving package types, control flow branches, or struct elements:

1. **Scaffolding & Type Insertion**:
   - Use `semantic_scaffold_file` to create a new source file with package headers inferred from siblings.
   - Use `semantic_insert_type` to add structs, interfaces, or type aliases.
   - Use `semantic_insert_decl` or `semantic_insert_declaration` to add top-level variables and constants.
   - Use `semantic_replace_decl` to update existing package-level constant, variable, or type declarations.
2. **Branching & Literal Construction**:
   - Use `semantic_insert_construct` to add inner control-flow constructs (loops, if-else, defer).
   - Use `semantic_insert_case` to add dispatch branches to existing switch statements.
   - Use `semantic_insert_construct` with `kind: "element"`, `in_declaration`, exact `discriminator`, and `source` to append a keyed map entry or struct literal property. Copy inspection's `expected_revision` guard when available. Placement defaults to `append`; no other placement is accepted for elements.
   - Struct type declarations use `kind: "field"` and `in_type`. Positional slice/array elements are unsupported; replace the enclosing construct or function body when that is the intended scope.
3. **Declaration Reorganization & Deletion**:
   - Use `semantic_move_construct` to relocate one supported declaration between files or packages.
   - Use `semantic_move_file` for a complete closed production-file declaration set, including private helpers and types with their methods. Select an existing production `destination_file` in another existing package of the same module. The operation appends declarations and leaves a package-only source shell; it does not delete the source path.
   - Matching tests are included by default (`move_matching_tests: false` opts out). The exact sibling test file must be dependency-closed with the moving declarations; the derived destination test path must be absent. Same-package tests retain private access; external test packages and imports follow the destination.
   - Refusals identify source dependencies or callers requiring a larger selection. Resolve shared test helpers, `TestMain`, other-file methods, fixtures, and initialization effects explicitly. Never export private declarations or absorb unrelated files to bypass a refusal. Package `main`, package variables, `init`, compiler linkage directives, build constraints, generated files, embeds, and relative fixture dependencies are outside this bounded operation.
   - File movement is Go-only. It validates all candidate edits and active test variants together before checked publication; a publication failure can leave explicitly reported partial writes. Use the returned changed paths and publication state before continuing.
   - MCP outcomes expose captured diagnostic counts and a `semantic_diagnostics` query. Page the report by category, filter, offset, and limit instead of collecting diagnostics again. Reports are immutable and may expire; deferred or unavailable counts are not clean verification.
   - Use `semantic_remove_construct` for safe, collision-checked deletion of unreferenced top-level declarations or switch cases.
   - Use `semantic_migrate_type` to migrate unexported struct fields between identical type identities.
   - Use `semantic_set_doc_comment` to add or update top-level declaration documentation comments.
4. **Verify**:
   - Validate workspace state with `semantic_verify`.

---

### Pattern F: Dependencies, Imports & Test Assertions

When managing dependencies, source imports, or test suites:

1. **Imports & Build Modules**:
   - Use `semantic_add_build_dependency` to add external Go modules and tidy `go.mod`/`go.sum`.
   - Use `semantic_organize_imports` to format import blocks and clean unused imports.
2. **Test Assertion Mode Migration**:
   - Use `semantic_assertion_mode` to toggle test assertions between fail-fast (`t.Fatal`) and continue-on-failure (`t.Error`) modes.
3. **Verify**:
   - Validate with `semantic_verify`.
