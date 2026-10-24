# Rust Semantic Refactoring Guide

For a failed selected-file lookup, inspect optional `suggestions` and retry the
same lookup with a candidate's `arguments`. Native outline and lookup checks
validate each selector; Rust qualification is preserved. Suggestions require the
normal trusted workspace and installed server, never edit source, and can be
disabled with `SEMEDIT_SUGGESTIONS=off`. See [configuration and limits](../../../docs/symbol-suggestions.md).

MCP results follow the [shared output policy](../SKILL.md#copy-typed-results-and-respect-publication).
Use typed continuations with their exact revisions, and retain failure and
publication state when choosing follow-up calls. Structured results can be
model-visible; text keeps exact next-step guards and names structured fields
without repeating source, inventories, diagnostic pages, diffs, or logs.
For diagnostic details, copy the response's `diagnostic_report.arguments` into
`semantic_diagnostics`, choose a category/filter and page size, then follow
`next_offset`. Pages reuse the captured report; a deferred/unavailable count
does not indicate a clean workspace. Use verification only for a fresh report.
Rename returns `active_count` and diagnostic changes rather than every existing
workspace warning; request `semantic_verify` for a full diagnostic inventory.

Install the external server from the official rust-analyzer source checkout with
`cargo xtask install --server` using the latest stable Rust toolchain. Ensure
the installed `rust-analyzer` executable is on the MCP server process's `PATH`;
the backend does not invoke Cargo to install or build it. See the official
[rust-analyzer binary installation guide](https://rust-analyzer.github.io/book/rust_analyzer_binary.html).

This reference provides workflows for executing Rust refactorings with `semedit` primitives. Read it when working with Rust source files. For exact eligibility rules, edit-capture contracts, and refusal boundaries for `refactor_extract_value`, `refactor_extract_callable`, `refactor_inline_value`, `refactor_inline_callable`, and `refactor_promote_local_to_const`, read the [Rust refactoring boundaries](rust/refactoring-boundaries.md). Common operation routing remains in the primary [tool guide](../SKILL.md#choose-the-operation-by-intent).

---

Compact lookup and outline listings use workspace-relative paths and one-based
inclusive declaration lines. Signatures describe declarations; explicit native
selectors remain separate and must be copied verbatim. Use ordinary file reading
for disk-backed results. A navigation result does not imply that this backend
supports symbol inspection or mutation.

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
   - Use `refactor_rename` with `file` and `trust_workspace: true` to update symbol names consistently across the target file.
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
