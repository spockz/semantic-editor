# Kotlin Semantic Refactoring Guide

Failed selected-file lookups may include native-validated `suggestions`. Retry
the same lookup with one candidate's `arguments`; preserve its file and qualified
selector. Normal server configuration and workspace trust remain required.
Suggestions do not edit source or retry automatically. See [configuration and limits](../../../docs/symbol-suggestions.md).

MCP results follow the [shared output policy](../SKILL.md#copy-typed-results-and-respect-publication).
Use typed continuations with their exact revisions, and retain failure and
publication state when choosing follow-up calls. Structured results can be
model-visible; text keeps exact next-step guards and names structured fields
without repeating source, inventories, diagnostic pages, diffs, or logs.
For diagnostic details, copy the response's `diagnostic_report.arguments` into
`semantic_diagnostics`, choose a category/filter and page size, then follow
`next_offset`. Pages reuse the captured report; a deferred/unavailable count
does not indicate a clean workspace. Use verification only for a fresh report.

This reference provides lookup guidance and agent-authored manual refactoring patterns for Kotlin. Read it only when working with Kotlin source files.

---

Compact lookup and outline listings use workspace-relative paths and one-based
inclusive declaration lines. Signatures describe declarations; explicit native
selectors remain separate and must be copied verbatim. Use ordinary file reading
for disk-backed results. A navigation result does not imply that this backend
supports symbol inspection or mutation.

## 1. Manual Refactoring Patterns (Not semedit Mutations)

`semedit` core operations for Kotlin (`semantic_lookup`, `semantic_outline`, `semantic_verify`) are indexed in the primary `SKILL.md` table. Kotlin backend operations require workspace trust and a preinstalled `kotlin-language-server`; source mutations are unsupported. The manual patterns below do not imply semedit mutation support; verify any agent-authored edits independently.

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
