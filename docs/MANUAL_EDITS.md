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

### Remaining Gaps & Next Highest-ROI Capabilities

1. **`replace_symbol_body`** (Highest Current ROI):
   - **Need**: Solves ME-0006, ME-0007, ME-0010. Replacing only the body of an existing function (`func Foo(...) { <body> }`) without touching signature, comments, or surrounding declarations.
   - **Engine**: Tree-sitter or Go AST locates function body braces `{ ... }` and replaces only the body range.
2. **AST Statement / Branch Insertion (`insert_statement`)**:
   - **Need**: Solves ME-0002, ME-0018, ME-0027. Adding a `case` branch to a `switch` statement or an entry to a router/table without rewriting the entire function.
3. **File Scaffolding (`scaffold_file`)**:
   - **Need**: Solves ME-0011, ME-0012, ME-0022-0025. Creating a new Go file with package header and skeleton declarations atomically.
