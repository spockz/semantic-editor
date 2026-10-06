# Scala Semantic Refactoring Guide

This reference provides workflows for inspecting and refactoring Scala source files with `semedit` primitives. Read this document only when working with Scala source files.

---

## 1. Macro Refactorings in Scala

`semedit` core operations for Scala (`semantic_lookup`, `semantic_outline`) are indexed in the primary `SKILL.md` table. Scala backend operations require workspace trust and a preinstalled Metals language server. This reference defines macro patterns and verification practices.

---

## 2. Scala Refactoring Patterns

### Pattern A: Case Class to Class Migration

When a Scala `case class` evolves to require mutable internal state, private constructor logic, or specialized caching:

1. **Inspection**:
   - Use `semantic_lookup` to locate companion objects, apply methods, and copy calls.
2. **Structural Transformation**:
   - Remove `case` modifier from class header.
   - Implement explicit constructor, accessor methods, `canEqual`, `equals`, `hashCode`, and `toString` if equality semantics must be preserved.
   - Add explicit factory methods in the companion object if `apply` instantiation is expected by callers.
3. **Verification**:
   - Execute `sbt compile` or `mill __.compile` to confirm zero broken call sites.

---

### Pattern B: Extract Trait for Component Decoupling

When abstracting concrete service implementations:

1. **Define Trait**:
   - Extract public method signatures into a new `trait`.
2. **Extend Trait**:
   - Update concrete class to extend the trait (`class ServiceImpl extends ServiceTrait`).
3. **Caller Generalization**:
   - Update dependent constructors and method parameters to accept the trait type.
4. **Verification**:
   - Verify compiler clean state.
