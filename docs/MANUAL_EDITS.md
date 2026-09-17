# Manual Edits Tracking Log

This document tracks all code edits performed manually (e.g. via text-based `replace_file_content` or `write_to_file`) during the development of `semedit`.

The objective is to categorize the underlying intent of each manual edit, identify which editing capabilities are absent from `semedit`, and prioritize future semantic tools.

---

## Log of Manual Edits

| ID | File | Nature of Change | AST / Semantic Intent | Missing Capability in `semedit` | Candidate Semantic Tool |
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

---

## Dogfooded Semantic Operations (Executed via `semedit` MCP Server)

Subagents executed the following edits deterministically via `semedit` MCP tools instead of manual text edits:

| Operation ID | Target File | Target Symbol | New Name | Ingress Interface | Verification Result |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SE-0001** | `internal/symbol/resolver.go` | `Symbol.FormatQualifiedName` | `BuildQualifiedName` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call sites renamed deterministically; `go test` passed. |
| **SE-0002** | `internal/astedit/insert.go` | `validateSnippet` | `verifySnippetSyntax` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call site renamed deterministically; `go test` passed. |
| **SE-0003** | `internal/astedit/function.go` | `verifyFunctionSnippet` | `parseFunctionSnippet` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call site renamed deterministically; `go test` passed. |

---

## Capability Gap Analysis & Future Tool Roadmap

### Completed Capabilities (Promoted to CLI `./bin/semedit` and Registered on MCP Server)

1. **`organize_imports` with Explicit Add/Remove** (Completed):
   - Supports plain import paths, custom aliases `alias "path"`, blank imports `_ "path"`, and explicit import removal alongside auto-resolution.
2. **`insert_function` / `insert_type` / `insert_decl`** (Completed):
   - Receiver-aware method clustering, strict public/private section partitioning, access modifier inference (`infer`, `public`, `private`), and `const`/`var` group merging (`append_group`).
3. **`replace_body` / `scaffold_file` / `insert_case` / `batch`** (Completed - ADR-0016):
   - `semantic_replace_body`: Scoped function/method body replacement by symbol name with in-memory validation and diff generation.
   - `semantic_scaffold_file`: Directory and file creation with sibling non-test package inference and overwrite guard.
   - `semantic_insert_case`: AST switch case clause insertion across multiple placement modes (`first`, `last`, `before_default`, `before`, `after`) with expression-matching anchor verification.
   - `semantic_batch`: Sequential fail-fast multi-edit orchestration with per-file deferred import optimization.

### Remaining Gaps & Next Highest-ROI Capabilities

1. **Function Block & Registration Statement Insertion (`insert_statement` / `append_call`)**:
   - **Need**: Solves ME-0002, ME-0018, ME-0027, and `rootCmd.AddCommand(...)` registration. Inserting a statement or call expression inside a specific function block (e.g. adding a command to a CLI root or an HTTP route to a router) without rewriting the whole function.
2. **Compound Literal / Collection Element Insertion (`insert_element`)**:
   - **Need**: Solves appending tool definitions into `tools/list` JSON/slice arrays or registering handlers in static tables.
3. **In-Tree Live-Reload for Self-Modification (`--live-edits` & `semantic_reload`)**:
   - **Need**: Solves the chicken-and-egg MCP dogfooding friction explored in [RQ-0021](research/RQ-0021-mcp-in-tree-live-reload.md). Enables the MCP server to reload in-place preserving stdio descriptors and notify the agent harness via `notifications/tools/list_changed`.
