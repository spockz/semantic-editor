# Go Semantic Refactoring Guide

This reference provides concrete patterns and workflows for executing Go refactorings with `semedit` primitives. Read this document only when refactoring or modifying Go source files.

---

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
   - Use `semantic_find_references` to locate all direct callers across the module.
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
   - Use `semantic_insert_composite_element` to append keyed elements to struct or map literals.
3. **Declaration Reorganization & Deletion**:
   - Use `semantic_move_construct` to relocate declarations between files or packages.
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
