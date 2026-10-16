# Kotlin Semantic Refactoring Guide

Failed selected-file lookups may include native-validated `suggestions`. Retry
the same lookup with one candidate's `arguments`; preserve its file and qualified
selector. Normal server configuration and workspace trust remain required.
Suggestions do not edit source or retry automatically. See [configuration and limits](../../../docs/symbol-suggestions.md).

This reference provides workflows for inspecting and refactoring Kotlin source files with `semedit` primitives. Read this document only when working with Kotlin source files.

---

## 1. Macro Refactorings in Kotlin

`semedit` core operations for Kotlin (`semantic_lookup`, `semantic_outline`, `semantic_verify`) are indexed in the primary `SKILL.md` table. Kotlin backend operations require workspace trust and a preinstalled `kotlin-language-server`. This reference defines macro patterns and verification practices.

---

## 2. Kotlin Refactoring Patterns

### Pattern A: Class to Data Class Migration

When converting a standard Kotlin class with boilerplate properties into a `data class`:

1. **Preflight Inspection**:
   - Use `semantic_outline` to inspect properties, constructors, and methods.
   - Verify that all primary constructor parameters are `val` or `var`, and the class does not extend open classes.
2. **Transformation**:
   - Add the `data` modifier to the class declaration.
   - Remove manual `equals`, `hashCode`, `toString`, and `copy` implementations redundant with compiler-generated implementations.
3. **Verification**:
   - Run `semantic_verify` with `language: "kotlin"` and `trust_workspace: true` to check diagnostics, followed by `gradle test` or `mvn test`.

---

### Pattern B: Extract Interface & Extension Function

When decoupling concrete classes or extracting reusable behaviors:

1. **Extract Interface**:
   - Declare interface with required member signatures.
   - Bind the concrete class to the interface.
2. **Extension Function Extraction**:
   - Move non-core utility methods into top-level extension functions to keep class bodies focused on core state.
3. **Verification**:
   - Run Kotlin diagnostics verification.
