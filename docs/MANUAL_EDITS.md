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
| **ME-0021** | `Makefile` | Added `test-property-full` and `test-fuzz` targets | Build configuration edit | Non-code file mutation (outside AST scope) | N/A (Build automation) |

---

## Dogfooded Semantic Operations (Executed via `semedit` MCP Server)

Subagents executed the following edits deterministically via `semedit` MCP tools instead of manual text edits:

| Operation ID | Target File | Target Symbol | New Name | Ingress Interface | Verification Result |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SE-0001** | `internal/symbol/resolver.go` | `Symbol.FormatQualifiedName` | `BuildQualifiedName` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call sites renamed deterministically; `go test` passed. |
| **SE-0002** | `internal/astedit/insert.go` | `validateSnippet` | `verifySnippetSyntax` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call site renamed deterministically; `go test` passed. |

---

## Capability Gap Analysis & Future Tool Roadmap

Based on the manual edits above, the highest-ROI semantic editing capabilities to add next after `semantic_rename` are:

1. **`organize_imports`** (High ROI):
   - **Need**: Solves ME-0001. When code references a new symbol from another package, the model currently falls back to line-based diffs to add imports.
   - **Engine**: Standard LSP code action (`source.organizeImports`), `gopls imports`, or `imports.Process`.
   - **Chaining Option**: Offer `auto_organize_imports: true` as an opt-in parameter or flag across other semantic mutation tools (`semantic_insert_declaration`, `semantic_rename`).
2. **`insert_function` / `insert_type` / `insert_decl`** (High ROI):
   - **Need**: Solves ME-0003, ME-0004, ME-0005. Adding new top-level functions or types to an existing file without having to read and rewrite entire files.
   - **Engine**: Go AST parser identifies target insertion boundary; formats via `pipeline.Format`.
   - **Placement Qualifiers**:
     - Boundary: `file_start`, `file_end` (default).
     - Section: `public_start`, `public_end`, `private_start`, `private_end`.
     - Relative: `before_symbol`, `after_symbol`.
   - **Visibility Qualifiers**: Explicit `visibility: "public" | "private"` validation (asserting or ensuring exported capitalization rules).
3. **`replace_symbol_body`** (Medium ROI):
   - **Need**: Solves ME-0006, ME-0007. Replacing only the body of an existing function (`func Foo(...) { <body> }`) without touching signature, comments, or surrounding declarations.
   - **Engine**: Tree-sitter or Go AST locates function body braces `{ ... }` and replaces only the body range.
