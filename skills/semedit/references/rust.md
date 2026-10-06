# Rust Semantic Refactoring Guide

This reference provides workflows for executing Rust refactorings with `semedit` primitives. Read this document only when refactoring or inspecting Rust source files.

---

## 1. Macro Refactorings in Rust

`semedit` core operations for Rust are indexed in the primary `SKILL.md` table. High-level idioms (e.g., Newtype Pattern, Trait Extraction) are orchestrated by the agent composing targeted edits and verifying via `cargo`.

---

## 2. Rust Refactoring Patterns

### Pattern A: Primitive to Newtype Pattern

When strengthening domain types from primitive values (e.g., `u64` $\to$ `UserId`):

1. **Preflight Inspection**:
   - Use `semantic_outline` to scan module structures and `semantic_lookup` to locate existing type usages.
2. **Newtype Declaration**:
   - Define tuple struct: `#[derive(Debug, Clone, Copy, PartialEq, Eq)] pub struct UserId(pub u64);`.
3. **Signature Updates & Symbol Rename**:
   - Update target function parameters to require `UserId`.
   - Use `semantic_rename` with `file` and `trust_workspace: true` to update symbol names consistently across the target file.
4. **Call-Site Wrapping & Verification**:
   - Wrap raw arguments with `UserId(...)` at call sites.
   - Verify zero compiler warnings via `cargo check` or `cargo test`.

---

### Pattern B: Extract Trait for Mocking / Dependency Injection

When abstracting concrete structs behind traits:

1. **Trait Discovery & Definition**:
   - Use `semantic_outline` to inspect concrete methods on the target struct.
   - Declare trait with the required signatures.
2. **Implement Trait for Concrete Struct**:
   - `impl TraitName for ConcreteType { ... }`.
3. **Generic / Trait Object Callers**:
   - Update callers to take `impl TraitName` or `&dyn TraitName`.
4. **Verification**:
   - Run `cargo test`.
