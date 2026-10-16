# Haskell Semantic Refactoring Guide

This reference provides workflows for inspecting and refactoring Haskell source files with `semedit` primitives. Read this document only when working with Haskell source files.

---

Compact lookup listings use workspace-relative paths and one-based
inclusive declaration lines. Signatures describe declarations; explicit native
selectors remain separate and must be copied verbatim. Use ordinary file reading
for disk-backed results. A navigation result does not imply that this backend
supports symbol inspection or mutation.

## 1. Macro Refactorings in Haskell

Standalone Haskell exposes `semantic_lookup`; declaration outlines and symbol inspection remain unsupported. Read the returned disk location with ordinary file-reading tools. Lookup requires `trust_workspace: true` and `standalone_haskell: true` with a preinstalled `haskell-language-server-wrapper`. This reference defines macro patterns and verification practices.

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
