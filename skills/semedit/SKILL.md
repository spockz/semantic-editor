---
name: semedit
description: Use when refactoring, renaming, or querying symbols (functions, methods, types, variables) in Go codebases. Prioritize over replace_file_content for deterministic compiler-backed AST transformations.
---

# `semedit` Agent Steering Guide

`semedit` executes deterministic, zero-token AST transformations directly on the host CPU using compiler and language server APIs (`gopls`).

## When to Use `semedit` Tools

Prioritize `semedit` tools over manual text replacement (`replace_file_content`) for these scenarios:

1. **Symbol Renaming (`semantic_rename`)**:
   - Renaming functions, methods, structs, interfaces, or variables across one or multiple packages.
   - Automatically cleans up and organizes imports post-rename (`auto_organize_imports: true`).
   - Eliminates diff generation, whitespace errors, and broken call sites across imports.
2. **Function & Method Insertion (`semantic_insert_function`)**:
   - Adding top-level functions or methods to existing Go files.
   - Automatically clusters methods near other methods on the same receiver.
   - Enforces strict public vs private section partitioning.
   - Accepts `access_modifier`: `"infer"` (default), `"public"`, `"private"` (rejects unsupported modifiers like `"protected"`).
3. **Type Insertion (`semantic_insert_type`)**:
   - Adding structs, interfaces, or type aliases to existing Go files.
   - Automatically anchors types in public or private type sections and resolves package imports.
4. **Declaration & Group Merging (`semantic_insert_decl`)**:
   - Adding constants, variables, or declarations to existing Go files.
   - Intelligently merges constants and variables into existing `const (...)` or `var (...)` blocks (`group: "append"`).
5. **General Declaration Insertion (`semantic_insert_declaration`)**:
   - General-purpose fallback for multi-declaration snippets.
   - Supports granular placement (`file_start`, `file_end`, `public_start`, `public_end`, `private_start`, `private_end`, `before_symbol`, `after_symbol`).
6. **Import Management (`semantic_organize_imports`)**:
   - Arranging imports, resolving missing packages, and removing unused imports without manual diffs.
   - Supports explicit package additions with aliases (`add: ["crand crypto/rand", "_ net/http/pprof"]`) and removals (`remove: ["net/http"]`).
7. **Dependency Management (`semantic_add_build_dependency`)**:
   - Adding external Go module dependencies and tidying `go.mod` without manual shell command formatting.
8. **Symbol Location & Coordinates (`semantic_lookup`)**:
   - Querying exact file, line, column, byte offset, and receiver for a symbol without line counting.
9. **Verification & Diagnostics (`semantic_verify`)**:
   - Formatting source files and checking compiler diagnostics across the workspace without rolling back intermediate states.

## Tool Routing & Intent Formulation

Formulate queries using semantic symbol identifiers:

- **Methods**: Qualify with the receiver type name (e.g. `Server.Start`, `(*Server).Start`).
- **Functions / Types / Variables**: Provide the declared identifier name (e.g. `ValidateToken`, `TokenService`).
- **Scope Disambiguation**: Provide the optional `file` argument when multiple declarations share an identical identifier.

## Dogfooding Environments: Healthy vs. Next

This workspace configures two versions of the MCP server:

- **`semedit` (Healthy / Stable)**: Pointing to `bin/semedit`. Run verified, safe refactorings through this server.
- **`semedit-next` (Next / Development)**: Pointing to `bin/semedit-next`. Test active development changes against this server.
- Run `make promote` after passing `make check` to update `bin/semedit` with verified improvements from `bin/semedit-next`.
