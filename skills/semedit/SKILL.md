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
   - Eliminates diff generation, whitespace errors, and broken call sites across imports.
2. **Symbol Location & Coordinates (`resolve_symbol_location`)**:
   - Querying exact file, line, column, byte offset, and receiver for a symbol.
   - Eliminates manual coordinate hunting or line-counting turns.
3. **Verification & Diagnostics (`semantic_verify`)**:
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
