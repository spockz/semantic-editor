# Manual Edits Tracking Log

This document tracks all code edits performed manually (e.g. via text-based `replace_file_content` or `write_to_file`) during the development of `semedit`.

The objective is to categorize the underlying intent of each manual edit, identify which editing capabilities are absent from `semedit`, and prioritize future semantic tools.

---

## Log of Manual Edits

| ID | File | Nature of Change | IDE Capability Family | Edit / Refactoring Capability Gap | Candidate Semantic Tool |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **ME-0001** | `main.go` | Added `os`, `strings`, `semedit/internal/mcp` to imports | Import addition | Import management | `organize_imports` / `add_import` |
| **ME-0002** | `main.go` | Added `case "mcp":` to command dispatch switch | Branch statement insertion | AST statement insertion into control flow | `apply_ast_rewrite` / `insert_statement` |
| **ME-0003** | `main.go` | Added `runMCP(...)` helper function | Function declaration insertion | Top-level declaration addition | `insert_function` / `insert_decl` |
| **ME-0004** | `internal/pipeline/pipeline.go` | Added `DiagnosticDelta` struct declaration | Type declaration insertion | Top-level type addition | `insert_type` / `insert_decl` |
| **ME-0005** | `internal/pipeline/pipeline.go` | Added `ComputeDelta(...)` and `FindModuleRoot(...)` | Function declaration insertion | Top-level function addition | `insert_function` / `insert_decl` |
| **ME-0006** | `internal/pipeline/pipeline.go` | Modified `CheckDiagnostics` to resolve module root | Method body / statement replacement | Scoped block modification | `replace_symbol_body` |
| **ME-0007** | `internal/mcp/server.go` | Added diagnostic delta calculation around `golang.Rename` | Scoped statement wrapping / insertion | Method body enhancement | `replace_symbol_body` |
| **ME-0008** | `Makefile` | Added `build-next` and `promote` targets | Build configuration edit | Non-code file mutation (outside AST scope) | N/A (Build automation) |
| **ME-0009** | `.agents/plugins/.../mcp_config.json` | Registered `semedit` and `semedit-next` servers | Configuration JSON update | Structured JSON mutation | N/A (Configuration) |
| **ME-0010** | `main.go` | Added diagnostic delta calculation in `runRename` | Function body enhancement | Scoped statement wrapping | `replace_symbol_body` |
| **ME-0011** | `internal/adapters/golang/property_test.go` | Added Rapid metamorphic property test file | New test file creation | File creation / scaffolding | `scaffold_test_file` |
| **ME-0012** | `testdata/scripts/rename_diagnostic_delta.txtar` | Added txtar acceptance test | Test script creation | Test file generation | `scaffold_txtar_test` |
| **ME-0013** | `internal/mcp/server.go` | Extracted `rootUri` and `workspaceFolders` in `initialize` | Client configuration parsing | Handled at transport layer | N/A (Protocol handling) |
| **ME-0014** | `internal/mcp/server.go` | Refactored `if-else` chain to `switch` statement for `gocritic` | Control flow refactoring | AST refactoring (`if_to_switch`) | `refactor_ast` |
| **ME-0015** | `internal/adapters/golang/deps.go` | Added `AddDependency` function to manage `go get` and `go mod tidy` | Package dependency addition | External module acquisition | `add_dependency` |
| **ME-0016** | `internal/astedit/insert.go` | Added `InsertDeclaration` with placement qualifiers and visibility checking | Declaration insertion engine | AST declaration insertion | `insert_declaration` |
| **ME-0017** | `internal/pipeline/pipeline.go` | Added `OrganizeImports` function and suggestions in `ComputeDelta` | Import management engine | AST import formatting and auto-detection | `organize_imports` |
| **ME-0018** | `main.go` | Added `insert`, `imports`, and `get` CLI subcommands | CLI command dispatch branches | AST statement insertion into control flow | `apply_ast_rewrite` / `insert_statement` |
| **ME-0019** | `internal/mcp/server.go` | Added `semantic_insert_declaration`, `semantic_organize_imports`, and `semantic_add_dependency` | MCP tool registration | Function declaration addition | `insert_declaration` |
| **ME-0020** | `internal/adapters/golang/property_test.go` | Added `configureRapidChecks` helper for fast property checks | Test configuration helper | Function declaration addition | `insert_declaration` |
| **ME-0022** | `internal/astedit/access.go` & `access_test.go` | Added `AccessModifier` enum and `LanguageBackend` capability validation | Access modifier abstraction | Cross-language access model | `insert_function` / `insert_type` |
| **ME-0023** | `internal/astedit/function.go` & `function_test.go` | Added `InsertFunction` with receiver clustering and public/private section partitioning | Function insertion engine | AST function/method insertion | `insert_function` |
| **ME-0024** | `internal/astedit/type.go` & `type_test.go` | Added `InsertType` with section placement and auto-imports | Type declaration insertion engine | AST type insertion | `insert_type` |
| **ME-0025** | `internal/astedit/decl.go` & `decl_test.go` | Added `InsertDecl` with `const (...)` / `var (...)` group merging | Declaration group merging engine | AST declaration group insertion | `insert_decl` |
| **ME-0026** | `internal/pipeline/pipeline.go` & `pipeline_test.go` | Added `OrganizeImportsWithOptions` and `ImportOptions` (`Add`, `Remove`) | Import management engine | Explicit import addition and removal | `organize_imports` |
| **ME-0027** | `main.go` | Added `insert-func`, `insert-type`, `insert-decl` CLI subcommands and `--add`/`--remove` to `imports` | CLI command dispatch branches | AST statement insertion into control flow | `apply_ast_rewrite` / `insert_statement` |
| **ME-0028** | `internal/mcp/server.go` & `server_test.go` | Registered `semantic_insert_function`, `semantic_insert_type`, `semantic_insert_decl`, updated `semantic_organize_imports` | MCP tool registration | Function declaration addition | `insert_declaration` |
| **ME-0029** | `Makefile` | Added `rm -f bin/semedit` and `build-next` prerequisite to `promote` and `build` | Build automation fix | Non-code file mutation (outside AST scope) | N/A (Build automation) |
| **ME-0030** | `internal/...` | Standardized on exported package-level sentinel errors and `%w` wrapping (ADR-0013) | Domain error standardization | AST error refactoring | `standardize_errors` |
| **ME-0031** | `main.go` | Migrated CLI entry point from `flag.FlagSet` to Cobra subcommands (ADR-0014) | CLI command framework migration | Full CLI restructure | N/A (CLI Framework) |
| **ME-0032** | `internal/...`, `main.go` | Implemented structured record errors (`SymbolError`, `SyntaxError`, `PlacementError`, `VisibilityMismatchError`) with source locations, compiler standard stderr formatting, and LSP-compatible MCP error payloads (ADR-0013) | Domain error structure and diagnostic location enrichment | AST error type scaffolding / Location-aware error mapping | `standardize_errors` / `refactor_ast` |
| **ME-0033** | `internal/mcp/server.go`, `internal/astedit/...`, `internal/symbol/...`, `main.go` | Resolved Codex peer review findings for ADR-0013: UTF-16 character translation, virtual snippet URIs, single-render CLI locations, URL escaping, and VisibilityMismatchError coordinates | Error record and location delivery refinement | Diagnostic location formatting / LSP encoding | `refactor_ast` |
| **ME-0034** | `internal/astedit/body.go`, `main.go`, `internal/mcp/server.go` | Added `semantic_replace_body` engine, Cobra subcommand, and MCP tool | Scoped function/method body replacement | Scoped block modification | `semantic_replace_body` |
| **ME-0035** | `internal/astedit/scaffold.go`, `main.go`, `internal/mcp/server.go` | Added `semantic_scaffold_file` engine, Cobra subcommand, and MCP tool | New file creation with sibling package inference | File creation / scaffolding | `semantic_scaffold_file` |
| **ME-0036** | `internal/astedit/switchcase.go`, `main.go`, `internal/mcp/server.go` | Added `semantic_insert_case` engine, Cobra subcommand, and MCP tool | Switch case clause insertion | AST statement insertion into control flow | `semantic_insert_case` |
| **ME-0037** | `internal/mcp/batch.go`, `internal/mcp/server.go` | Added `semantic_batch` engine and MCP tool for sequential multi-edit execution | Multi-edit composition and orchestration | Protocol-level multi-edit orchestration | `semantic_batch` |
| **ME-0038** | `internal/mcp/server.go`, `internal/mcp/reload_*.go`, `main.go`, `Makefile` | Added `semantic_reload` tool, `--live-reload` flag, listChanged capability, and atomic binary promotion (ADR-0017) | MCP server lifecycle & live re-exec | Dynamic MCP tool reload & in-place exec | `semantic_reload` |
| **ME-0039** | `task-07-generate-template-main` benchmark trace | After `semantic_replace_body` and `semantic_organize_imports`, the agent manually re-read `main.go` to inspect the selected imports | Post-edit context feedback | Mutation results omit the post-format enclosing scope and selected imports, forcing confirmation reads before the next semantic operation | `post_edit_context` / enclosing-scope projection |
| **ME-0040** | `internal/astedit/construct.go`, `replace_decl.go`, `collision.go`, `construct_test.go` | Created new engine files, declarations, and AST tests in an isolated worktree | File scaffolding and declaration insertion | `semantic_scaffold_file`, `semantic_insert_type`, `semantic_insert_function`, and `semantic_insert_decl` fit parts of the edit, but two workers could not verify the MCP server's active checkout root before a write | Explicit worktree binding and root confirmation |
| **ME-0041** | `internal/astedit/decl.go`, `internal/operation/wire_engine_declarations.go` | Added `Overwrite` fields, a collision preflight, and one `insertDeclParams` element | Member, statement, and composite element insertion | Existing tools add top-level declarations or replace whole function bodies; they cannot insert one struct field, guard, or literal element | `semantic_insert_field`, `semantic_insert_statement`, `semantic_insert_composite_element` |
| **ME-0042** | `internal/mcp/schema_test.go`, `tools/benchmark-harness/driver.go` | Added an MCP schema test and extracted two benchmark driver helpers | Function insertion and body replacement | Existing tools fit, but this worker avoided MCP writes because the configured server root might select the primary checkout | Explicit worktree binding and root confirmation |
| **ME-0043** | `internal/operation/wire_backend_test.go`, `tools/benchmark-harness/agy_driver.go` | Changed a registry count, lowercased one error string, and added a security justification comment | Expression and comment replacement | Whole-body replacement is too broad for one literal or comment | `semantic_replace_expression` and targeted comment editing |
| **ME-0044** | `testdata/scripts/construct_replacements*.txtar`, `mcp_server.txtar`, ADR and friction Markdown | Authored CLI/MCP commands and literal before/after fixture trees; updated documentation | Test archive and Markdown editing | Go AST tools do not structure txtar archives or Markdown tables | Keep literal txtar fixtures; use structured fixture/document editing outside Go AST operations |
| **ME-0045** | `internal/operation/wire_*.go`, `internal/mcp/server.go` | Reworded tool-summary string literals after the planner-selection audit | Expression or string-literal replacement | No operation can replace a scoped literal without regenerating its enclosing function body or declaration | `semantic_replace_expression` with function or declaration owner, exact old value, and ambiguity rejection |
| **ME-0046** | `skills/semedit/SKILL.md`, this log | Added one built-in-preference row per tool and recorded the catalog audit | Structured Markdown editing | Go AST operations do not edit Markdown tables or prose | A document-aware table editor; retain atomic text editing until available |

---

## Dogfooded Semantic Operations (Executed via `semedit` MCP Server)

Subagents executed the following edits deterministically via `semedit` MCP tools instead of manual text edits:

| Operation ID | Target File | Target Symbol | New Name | Ingress Interface | Verification Result |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SE-0001** | `internal/symbol/resolver.go` | `Symbol.FormatQualifiedName` | `BuildQualifiedName` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call sites renamed deterministically; `go test` passed. |
| **SE-0002** | `internal/astedit/insert.go` | `validateSnippet` | `verifySnippetSyntax` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call site renamed deterministically; `go test` passed. |
| **SE-0003** | `internal/astedit/function.go` | `verifyFunctionSnippet` | `parseFunctionSnippet` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call site renamed deterministically; `go test` passed. |
| **SE-0004** | `internal/operation/wire_engine_mutations.go`, `wire_engine_declarations.go` | `ReplaceLoopReq`, `ReplaceDeclReq` | New request types | `semedit` (`semantic_insert_type`) | Explicit absolute worktree targets; `make check` passed after integration. |
| **SE-0005** | `internal/operation/wire_engine_mutations.go`, `wire_engine_declarations.go` | Replacement parsers, handlers, definitions | New functions | `semedit` (`semantic_insert_function`) | Registry and CLI/MCP schemas passed `make check`. |
| **SE-0006** | `internal/operation/wire_engine_mutations.go`, `wire_engine_declarations.go` | `replaceLoopParams`, `replaceDeclParams` | New declarations | `semedit` (`semantic_insert_decl`) | Parameter contracts passed registry and schema tests. |
| **SE-0007** | `internal/operation/wire_engine.go`, `wire_engine_declarations.go` | Registry calls and overwrite parsing | Updated bodies | `semedit` (`semantic_replace_body`) | Registry count and CLI string-shape tests passed. |
| **SE-0008** | `cmd/docgen/benchmark_aggregate.go` | `writeBenchmarkBrowserAssets` and extracted helpers | Body replacement and new functions | `semedit` (`semantic_replace_body`, `semantic_insert_function`) | Documentation generation and benchmark tests passed. |

---

## Edit / Refactoring Capability Gap Analysis

### Implemented Edit / Refactoring Capabilities

1. **`organize_imports` with Explicit Add/Remove** (Completed):
   - Supports plain import paths, custom aliases `alias "path"`, blank imports `_ "path"`, and explicit import removal alongside auto-resolution.
2. **`insert_function` / `insert_type` / `insert_decl`** (Completed):
   - Receiver-aware method clustering, strict public/private section partitioning, access modifier inference (`infer`, `public`, `private`), and `const`/`var` group merging (`append_group`).
3. **`replace_body` / `scaffold_file` / `insert_case` / `batch`** (Completed - ADR-0016):
   - `semantic_replace_body`: Scoped function/method body replacement by symbol name with in-memory validation and diff generation.
   - `semantic_scaffold_file`: Directory and file creation with sibling non-test package inference and overwrite guard.
   - `semantic_insert_case`: AST switch case clause insertion across multiple placement modes (`first`, `last`, `before_default`, `before`, `after`) with expression-matching anchor verification.
   - `semantic_batch`: Sequential fail-fast multi-edit orchestration with per-file deferred import optimization.
4. **In-Tree Live-Reload for Self-Modification** (Completed - ADR-0017):
   - `semantic_reload` & `--live-reload`: In-place stdio re-exec via `syscall.Exec`, dynamic schema discovery via `notifications/tools/list_changed`, and atomic Makefile binary promotion.

### Unimplemented Edit / Refactoring Capability Families

1. **Function Block & Registration Statement Insertion (`insert_statement` / `append_call`)**:
   - **Need**: Solves ME-0002, ME-0018, ME-0027, and `rootCmd.AddCommand(...)` registration. Inserting a statement or call expression inside a specific function block (e.g. adding a command to a CLI root or an HTTP route to a router) without rewriting the whole function.
2. **Compound Literal / Collection Element Insertion (`insert_element`)**:
   - **Need**: Solves appending tool definitions into `tools/list` JSON/slice arrays or registering handlers in static tables.

## ADR-0046 Post-Commit Semantic-Tool Audit

Three workers audited their own edits after commit `bce450f` reached `main` and `make check` passed. The operation worker used absolute worktree paths successfully with `semantic_insert_type`, `semantic_insert_function`, `semantic_insert_decl`, and `semantic_replace_body` for new request types, parameters, parsers, handlers, registry calls, and the `writeBenchmarkBrowserAssets` extraction. The AST and integration workers used atomic scoped edits because prior worktree-binding failures ([ST-0017](SUBOPTIMAL_TOOLS.md), [ST-0028](SUBOPTIMAL_TOOLS.md)) left the active MCP root uncertain. Successful absolute-path calls in one worker show that the missing piece is a reliable root contract, not a blanket inability to edit a worktree.

| Actual edit | Existing semantic fit | Why the worker did or did not use it | Improvement |
| :--- | :--- | :--- | :--- |
| Add `LoopOptions`, `ReplaceDeclOptions`, engine helpers, and AST test functions | Scaffold a Go file, then insert types and functions | AST worker avoided a possible write to the primary checkout; operation worker used these tools on explicit absolute paths | Return the canonical active root in tool results and reject a target outside an explicitly selected worktree |
| Add `ErrDeclCollision` | `semantic_insert_decl` can add a sentinel variable | AST worker avoided an unverified MCP root | Root confirmation before mutation; preserve a comment attached to the inserted spec |
| Insert `Overwrite bool` into `DeclOptions` and `InsertDeclReq` | No member-level operation | Inserting a second type would collide; replacing the whole type declaration would regenerate unrelated fields | `semantic_insert_field` keyed by file, type, field, and placement |
| Insert collision handling after target parsing in `InsertDecl` | `semantic_replace_body` could regenerate the complete body | The body also contains declaration placement and grouping; replacing it for one preflight risks unrelated behavior | `semantic_insert_statement` with a function and a verified before/after statement anchor |
| Add `overwrite` to the existing `insertDeclParams` slice | New `semantic_replace_decl` could replace the whole variable, but it was absent from the live catalog ([ST-0037](SUBOPTIMAL_TOOLS.md)) | The desired change was one element in an existing composite literal | Refresh the catalog after building; add `semantic_insert_composite_element` with a named variable and anchor |
| Change `len(entries) != 17` to `len(entries) != 19`, or lowercase one `fmt.Errorf` string | Whole-body replacement is possible but disproportionate | Both are one-expression edits | `semantic_replace_expression` scoped to the owning function, with ambiguity rejection |
| Add the `agyBin` G204 comment | No comment-level operation | Replacing `runAgy` would rewrite a large function for one line | Insert or replace a comment attached to a selected statement |
| Add CLI/MCP txtar commands and wanted trees | No Go AST operation applies to the archive container | The worker authored the archives directly and compared full resulting files | Retain public CLI txtar contracts with literal before/after trees |

The new [CLI fixture](../testdata/scripts/construct_replacements.txtar) and [MCP batch fixture](../testdata/scripts/construct_replacements_mcp_batch.txtar) are executable regression examples. They confirm that a failed duplicate insertion and an ambiguous loop selection leave the file unchanged, then compare the exact trees after `--overwrite`, `replace-decl`, and `replace-loop`. The loop fixture preserves statements outside the edited loop.

### Candidate Regression Fixtures for Missing Operations

**Field insertion.** Given `type InsertDeclReq struct { File string }`, inserting `Overwrite bool` should produce `type InsertDeclReq struct { File string; Overwrite bool }` after formatting. A second insertion should report a collision without changing the file.

**Anchored statement insertion.** Given a function body with `parseTarget()` followed by `appendDecl()`, inserting `checkPackageCollisions()` after `parseTarget()` should preserve `appendDecl()` and all other statements. A repeated or missing anchor should fail before mutation.

**Composite element insertion.** Given `var params = []ParameterContract{{Name: "file"}}`, inserting `{Name: "overwrite", Type: ParamBoolean}` after the file entry should retain the first element and order. This is the exact shape needed for `insertDeclParams`.

**Expression replacement.** In `TestRegisteredDefsHonorContracts`, replace only the literal in `len(entries) != 17` with `19`. If two matching expressions exist in the same function, require a path instead of silently choosing one.

**Declaration and loop replacement.** The committed txtar trees already exercise a same-file collision, explicit overwrite, and complete loop replacement. Extend them with a grouped `const` whose later omitted expression inherits the target's value; replacing the target must fail without changing that sibling. A nested-loop fixture should use a reported `loop_path` to modify the inner loop while preserving its parent and a sibling loop.

**Workspace verification.** `semedit verify --path .` failed after the Go cache was copied into `.scratch/go/mod`, because recursive `gofmt` entered intentionally malformed upstream test fixtures ([ST-0038](SUBOPTIMAL_TOOLS.md)). Verification of `internal/astedit` returned zero diagnostics. Directory verification should enumerate project source files while excluding `.scratch` and dependency caches, and report formatter stderr on failure.

## Holistic Read and Write Tool Audit (2026-09-25)

The promoted full-profile server advertises 21 tools: 19 registry operations, `semantic_batch`, and `report_feedback`. `semantic_reload` is a twenty-second conditional tool when `--live-reload` is enabled. Only `semantic_lookup` is read-only. `semantic_verify` can format or change imports, `semantic_snapshot` writes a journal, and the Maven tools execute a build. A planner must not classify them as harmless reads.

| Axis | Available now | Missing capability and concrete example |
| :--- | :--- | :--- |
| Symbol read | `semantic_lookup` resolves a named symbol in Go, trusted Rust/Java/Scala, or standalone Haskell | Find all references or callers of `InsertDecl`; lookup returns a declaration, not a reference set. A read-only `semantic_references` would replace `rg 'InsertDecl'` for this intent. |
| Source/structure read | Lookup reports a symbol location | Return the selected AST node, enclosing body, imports, and canonical workspace root. Before changing `insertDeclParams`, an agent needs the current composite literal without shell `sed` or a full-file read; before a worktree write, it needs proof of the target root. |
| Diagnostics read | Mutations return diagnostic deltas; `semantic_verify` checks and formats | Inspect current diagnostics without writing or running formatters. `semantic_diagnostics` should give the same revision-scoped diagnostic information as mutation receipts. |
| Go writes | Rename, body/loop/declaration replacement, declaration/function/type/case insertion, scaffolding, imports, dependencies, assertion conversion, batch | Insert `Overwrite bool` into `DeclOptions`; insert a collision guard after a named statement; append `{Name: "overwrite"}` to `insertDeclParams`; change only `len(entries) != 17`; attach a `#nosec` comment. These need field, anchored-statement, composite-element, expression, and attached-comment operations. |
| Other-language writes | Trusted Rust and Java selected-file rename; bounded Java formatting/import actions | Go-only structural operations remain unavailable in Rust, Java, Scala, and Haskell. Extend only after backend capability metadata and matching CLI txtar coverage define the supported pair. |
| Workspace/control | Snapshot/undo, trusted fixed Maven compile/test, optional live reload, feedback draft | Preview a proposed edit and its diagnostics without mutation, and select a worktree/root explicitly. A built server should announce the canonical active root and refreshed tool catalog before an agent writes. |

The catalog descriptions now begin with an affirmative `Use this tool instead of...` trigger. The dedicated MCP schema test checks every full-profile tool and the conditional live-reload catalog. The skill table has a distinct built-in alternative for each tool, including Maven, snapshot, undo, assertion conversion, feedback, and conditional reload. Selection remains bounded by each schema: `semantic_replace_loop` requires a complete loop, `semantic_replace_decl` requires the same existing identifier in the replacement, and `semantic_verify` may write formatting. The `ExampleRaw` field in registry entries still does not appear as concrete examples in MCP input schemas; selected parameter descriptions should surface symbol, path, placement, and ambiguity examples. Backend-specific scope is in prose rather than generated from capabilities, so stale descriptions are possible when a backend changes.

### Re-evaluation of the ADR-0046 implementation diff

The preserved pre-implementation diff is `.scratch/adr0046-original-implementation.patch` (`d9226f9..bce450f`). The current promoted binary adds `semantic_replace_loop` and `semantic_replace_decl` to the operations available when the earlier workers ran. The table below distinguishes available operations from edits for which the new tools are still too coarse.

| Original diff site | Tool that should be selected now | Remaining reason for a fallback |
| :--- | :--- | :--- |
| New `internal/astedit/construct.go`, `collision.go`, `replace_decl.go`, and `construct_test.go` | `semantic_scaffold_file`, then `semantic_insert_type`, `semantic_insert_function`, and `semantic_insert_decl` | Source reads and attached comment placement have no focused semantic operation. |
| `ErrDeclCollision` in `internal/astedit/errors.go` | `semantic_insert_decl` for the new sentinel | An existing grouped declaration may need a comment-preserving insertion contract. |
| `InsertDecl` collision preflight and operation parsers/handlers | `semantic_replace_body` for changed function bodies; `semantic_insert_function` for new helpers | Replacing a long body to add one guard is disproportionate; an anchored statement insertion is missing. |
| `DeclOptions.Overwrite` and `InsertDeclReq.Overwrite` | No existing member-level operation | `semantic_insert_field` should target the named struct and reject a duplicate field. |
| `insertDeclParams` addition and any existing package variable update | `semantic_replace_decl` can replace the complete variable definition now | A single composite element still requires regenerating the entire value; `semantic_insert_composite_element` would preserve siblings. |
| Existing `benchmarkBrowserShortcode` constant in `cmd/docgen/benchmark_aggregate.go` | `semantic_replace_decl` should be used now; the earlier `semantic_replace_body` attempt failed because the symbol was a constant ([ST-0035](SUBOPTIMAL_TOOLS.md)) | The current MCP catalog in this task remains stale; the promoted semantic CLI can perform the declaration replacement. |
| `registerEngineOps` registrations | `semantic_replace_body` | `semantic_insert_statement` could add each registration at an anchored call without rewriting the rest of the function. |
| `internal/mcp/schema_test.go` new tests and new request/operation definitions | `semantic_insert_function`, `semantic_insert_type`, `semantic_insert_decl`; `semantic_batch` when ordering helps | Schema assertions inside an existing test need whole-body replacement until scoped statement/expression edits exist. |
| Registry count or one error string in `wire_backend_test.go`, plus security comment in `agy_driver.go` | No proportional operation; `semantic_replace_body` is a broad fallback | `semantic_replace_expression` and attached-comment editing would mutate the intended node only. |
| Existing loops in tests or handlers when a loop alone changes | `semantic_replace_loop`, selected by `loop_on` and `loop_path` | It cannot add a new loop or change surrounding statements; full replacement source is required. |
| Go imports in any changed file | `semantic_organize_imports`, or automatic import organization on the selected edit | Package import edits must not be mistaken for external module dependency changes. |
| `testdata/scripts/*.txtar`, ADR prose, and friction logs | No Go semantic operation applies | Keep literal before/after trees for executable regression fixtures; use atomic document/archive edits. |

A useful read regression fixture would put two methods named `Run` in different packages and assert that lookup returns the selected declaration and an explicit root. A write regression fixture would start with `type DeclOptions struct { Group string }` and insert `Overwrite bool`; a repeated call must fail before mutation. For a composite fixture, start with `var params = []ParameterContract{{Name: "file"}}`, add `{Name: "overwrite"}` after the existing element, and compare the exact resulting tree. For an expression fixture, change only `len(entries) != 17` to `len(entries) != 19` inside `TestRegisteredDefsHonorContracts`, rejecting ambiguous matches. These are safe to store literally because they come from this codebase.
