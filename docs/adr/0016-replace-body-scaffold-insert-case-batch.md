# ADR-0016: Next Capabilities: Replace Body, Scaffold File, Insert Case, and Batch

* **Status**: Accepted
* **Date**: 2026-09-16

---

## Context

Prior to this decision, `semedit` provided AST insertion tools (`semantic_insert_declaration`, `semantic_insert_function`, `semantic_insert_type`, `semantic_insert_decl`), symbol coordinate lookup (`resolve_symbol_location`), symbol renaming (`semantic_rename`), and import management (`semantic_organize_imports`).

However, analysis of manual editing logs (ME-0002, ME-0006, ME-0007, ME-0010, ME-0011, ME-0012, ME-0018, ME-0022–ME-0025, ME-0027) revealed four primary capability gaps:

1. **Symbol Body Modification**: Modifying an existing function or method body forced agents to resort to line-based `replace_file_content` or regex replacements.
2. **File Scaffolding**: Initializing new Go files required manual creation with explicit package clauses, frequently creating mismatches with existing package names or mistakenly inferring test package names (`package foo_test`).
3. **Switch Case Injection**: Adding branch statements to switch blocks in dispatchers or command handlers required line-and-column text replacements.
4. **Multi-Edit Composition**: Executing multiple semantic transformations across one or more files required multiple round-trips between the agent and MCP server, incurring substantial token overhead and latency.

---

## Decision

We introduce four discrete capabilities across the engine, CLI, and MCP interfaces:

1. **`semantic_replace_body` (`replace-body`)**:
   * Targets functions or methods by symbol identifier (`"Foo"`, `"(*T).Foo"`, `"T.Foo"`).
   * Accepts replacement body as bare Go statements without surrounding braces.
   * Validates syntax in memory via synthetic function encapsulation before touching disk.
   * Employs exact byte offsets via `fset.Position(body.Lbrace).Offset` and `fset.Position(body.Rbrace).Offset`.
   * Formats and validates with `go/format` prior to atomic write via `pipeline.WriteAtomic`.
   * Returns unified diff of old body vs new body.
   * Exposes `auto_organize_imports` (default `false`).

2. **`semantic_scaffold_file` (`scaffold-file`)**:
   * Scaffolds new Go files with canonical package headers.
   * Supports `package: "infer"` (default), which scans sibling `.go` files while explicitly skipping `_test.go` files to prevent inheriting `_test` package declarations.
   * Fails with `ErrInferNoSiblings` when no valid sibling Go files exist in the target directory.
   * Fails with `ErrFileExists` if the target exists unless `overwrite: true`.
   * Returns resolved package name.
   * Exposes `auto_organize_imports` as a no-op flag for schema uniformity.

3. **`semantic_insert_case` (`insert-case`)**:
   * Locates switch statements within a target function by function name and optional discriminant expression (`switch_on`).
   * Matches tagless switch statements when `switch_on` is omitted.
   * Canonicalizes discriminant expressions via `go/parser` and `go/format` comparison.
   * If multiple matching switches exist, fails without mutation and reports each candidate's matching-switch path and case labels.
   * Accepts `switch_path` to select a reported candidate. Root matching switches use zero-based indexes (`0`, `1`, ...); nested matching switches use dotted paths such as `0.1`, where only matching switches are counted and the parent occupies index `0` at each nesting level.
   * Validates case clauses via synthetic switch stubs in memory.
   * Supports relative placements: `first`, `last`, `before_default` (default), `before`, and `after`.
   * Employs `ErrSwitchNotFound` when target switch is missing and `ErrAnchorNotFound` when specified anchor case is absent.
   * Returns diff and exposes `auto_organize_imports` (default `false`).

4. **`semantic_batch` (MCP tool)**:
   * Accepts an ordered list of `[{tool, params}]` entries.
   * Executes sequentially in `atomic: false` mode (fail-fast on first error).
   * Writes each edit to disk immediately on success.
   * Stated cross-file atomicity (`atomic: true`) is deferred to preserve correctness without complex in-memory virtual file systems.
   * Aggregates modified files and executes `auto_organize_imports` or `format` once per written file at batch conclusion.
   * Returns structured per-edit status array alongside overall status.

---

## Invariants

1. **Batch Principle (No Individual Aggregation)**: Individual semantic tools must never introduce internal aggregating constructs (e.g., accepting multiple bodies or multiple cases in a single tool call). Multi-edit composition is strictly delegated to `semantic_batch`.
2. **Schema Consistency Rule**: Every modifying tool exposes `auto_organize_imports: bool` (default `false`). For `scaffold_file`, the flag is a no-op but present for contract uniformity.
3. **Zero-Mutation on Parse/Syntax Failure**: Any syntax error in a replacement body or case snippet aborts before disk modification, ensuring files remain untouched.
4. **Byte Offset Coordinates**: All AST splices must calculate exact file byte offsets via `fset.Position(...).Offset` rather than raw `token.Pos`.
5. **Sibling Inference Test Exclusion**: Sibling package inference must ignore all `*_test.go` files.
6. **Canonical Expression Matching**: Switch discriminant matching must compare canonically formatted AST expressions rather than raw substring matching.
7. **Explicit Switch Selection**: When `switch_on` matches multiple switches in the target function, `insert-case` must not choose implicitly or mutate the file. Its diagnostic must provide case labels and usable `switch_path` values; numbering counts only switches matching the requested selector.

---

## Consequences

* Eliminates remaining manual edit triggers (ME-0002, ME-0006, ME-0007, ME-0010, ME-0011, ME-0012, ME-0018, ME-0022–ME-0025, ME-0027) from agent refactoring workflows.
* Reduces round-trip token overhead and execution latency for multi-edit tasks via `semantic_batch`.
* Standardizes on exported sentinel errors (`ErrSwitchNotFound`, `ErrAnchorNotFound`, `ErrInferNoSiblings`, `ErrFileExists`, `ErrNoBody`) conforming to ADR-0013.
