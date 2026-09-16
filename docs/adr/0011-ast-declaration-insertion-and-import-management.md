# ADR-0011: Top-Level Declaration Insertion and Automated Import Management

* **Status**: Accepted
* **Date**: 2026-09-16

## Context

When modifying Go source files, agent planners frequently need to add new top-level declarations (types, methods, functions, constants) or adjust package imports when referencing external types.

Without semantic AST insertion tools, agents fall back to line-based text diff tools (`replace_file_content`) or full file rewrites (`write_to_file`). This introduces several failure modes:

1. Syntax corruption due to hallucinated indentation or partial brace replacement.
2. Inconsistent file organization violating Go conventions (e.g. exported functions placed haphazardly below helper methods).
3. Lingering unused imports or missing package imports requiring secondary manual editing turns.

## Decision

1. **Pre-Insertion Syntax Validation**:
   `semedit` parses the snippet to be inserted with `go/parser` in isolation before touching the target file. If the snippet contains invalid Go syntax, the operation is rejected immediately with zero disk mutation.

2. **Canonical Section Invariants (Public Precedes Private)**:
   In idiomatic Go, exported declarations precede unexported declarations. `semedit` supports explicit placement qualifiers:
   * **Boundary**: `file_start` (immediately below package/import headers), `file_end` (bottom of file, default).
   * **Sections**:
     * `public_start`: Preceding the first exported declaration. If no exported declarations exist, it lands immediately preceding the first private declaration (or at `file_start`).
     * `public_end`: Following the last exported declaration. If no exported declarations exist, it lands immediately preceding the first private declaration (identical to `public_start`).
     * `private_start`: Preceding the first unexported declaration. If no unexported declarations exist, it lands at `file_end`.
     * `private_end`: Following the last unexported declaration (or at `file_end`).
   * **Relative**: `before_symbol` and `after_symbol` targeting qualified symbols (`Type.Method` or `Function`) resolved via `symbol.Resolve`.

3. **Visibility Validation**:
   Callers may optionally specify `visibility: "public" | "private"`. `semedit` validates that the declared top-level identifier complies with Go capitalization rules (`ast.IsExported`), preventing accidental scope mismatches.

4. **Automated Import Chaining (`auto_organize_imports`)**:
   `semantic_insert_declaration` and `semantic_rename` support `auto_organize_imports: bool` (default `true`). Following AST mutation, the engine executes in-process import resolution via `golang.org/x/tools/imports`.

5. **Actionable Unresolved Dependency Guidance**:
   If an imported package cannot be resolved because it is missing from `go.mod`, `semedit` does not silently discard the import or roll back changes. Instead, it emits an actionable diagnostic suggestion (`Run 'go get <pkg>' or invoke semantic_add_dependency`).

6. **Manifest Safety Invariant (ADR-0005)**:
   Adding dependencies requires an explicit tool invocation (`semantic_add_dependency` / `semedit get`). `semedit` will never initiate unprompted external network requests or background dependency mutations.

## Invariants

* Snippets containing invalid syntax must never be written to disk.
* File updates must strictly use atomic writes (`pipeline.WriteAtomic`) and advancing `mtime` ([ADR-0010](0010-disk-synchronization-and-cache-invalidation.md)).
* When public section placement is requested on a file with exclusively private declarations, the declaration must land preceding the first private declaration.
* Dependency fetching remains an explicit tool action and is never triggered automatically in the background.

## Consequences

* **Positive**: Eliminates manual line-based text edits for adding declarations and imports; guarantees syntactical validity and idiomatic Go file structure.
* **Negative**: Complex multi-declaration snippets containing interrelated unexported types must be provided as coherent AST blocks.
