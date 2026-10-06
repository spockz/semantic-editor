# Haskell Semantic Refactoring Guide

This reference provides workflows for inspecting and refactoring Haskell source files with `semedit` primitives. Read this document only when working with Haskell source files.

---

## 1. Macro Refactorings in Haskell

`semedit` core operations for standalone Haskell (`semantic_lookup`, `semantic_outline`) are indexed in the primary `SKILL.md` table. Standalone Haskell lookup requires `trust_workspace: true` and `standalone_haskell: true` with a preinstalled `haskell-language-server-wrapper`. This reference defines macro patterns and verification practices.

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
