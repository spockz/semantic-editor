# Haskell Semantic Refactoring Guide

MCP results follow the [shared output policy](../SKILL.md#copy-typed-results-and-respect-publication).
Use typed continuations with their exact revisions, and retain failure and
publication state when choosing follow-up calls. Structured results can be
model-visible; text keeps exact next-step guards and names structured fields
without repeating source, inventories, diagnostic pages, diffs, or logs.
For diagnostic details, copy the response's `diagnostic_report.arguments` into
`semantic_diagnostics`, choose a category/filter and page size, then follow
`next_offset`. Pages reuse the captured report; a deferred/unavailable count
does not indicate a clean workspace. Use verification only for a fresh report.

This reference provides workflows for inspecting and refactoring Haskell source files with `semedit` primitives. Read this document only when working with Haskell source files.

---

Compact lookup listings use workspace-relative paths and one-based
inclusive declaration lines. Signatures describe declarations; explicit native
selectors remain separate and must be copied verbatim. Use ordinary file reading
for disk-backed results. A navigation result does not imply that this backend
supports symbol inspection or mutation.

## 1. Macro Refactorings in Haskell

Standalone Haskell exposes `semantic_lookup`; declaration outlines and symbol inspection remain unsupported. Read the returned disk location with ordinary file-reading tools. Lookup requires `trust_workspace: true` and `standalone_haskell: true` with a preinstalled `haskell-language-server-wrapper`. This reference defines macro patterns and verification practices.

Failed lookups may return validated spelling alternatives from the server's current document symbols. Retry explicitly with a returned selector and file; suggestion retrieval does not add public outline or mutation support.

---

## 2. Haskell Refactoring Patterns

### Pattern A: Data Type & Record Refactoring

When refactoring Algebraic Data Types (ADTs) or record fields:

1. **Preflight Inspection**:
   - Use `semantic_lookup` to resolve type declarations and constructor definitions.
2. **Record Extraction**:
   - Break large sum-type constructors with extensive record fields into dedicated product types to eliminate partial record selector hazards.
3. **Pattern Match Exhaustiveness**:
   - Ensure all case expressions and top-level function clauses are updated to cover new constructors.
4. **Verification**:
   - Run `ghc -Wall` or `cabal build` to verify exhaustiveness and type soundness.

---

### Pattern B: Extract Pure Function Helper

When decomposing complex monadic or nested `where`/`let` bindings:

1. **Decompose Expression**:
   - Extract pure expressions from `where` clauses into top-level helper functions with explicit type signatures.
2. **Generalize Type Signatures**:
   - Where appropriate, generalize concrete types to standard typeclasses (`Applicative`, `Monad`, `Foldable`).
3. **Verification**:
   - Verify compiler clean state.
