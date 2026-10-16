# Scala Semantic Refactoring Guide

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

This reference provides workflows for inspecting and refactoring Scala source files with `semedit` primitives. Read this document only when working with Scala source files.

---

Compact lookup and outline listings use workspace-relative paths and one-based
inclusive declaration lines. Signatures describe declarations; explicit native
selectors remain separate and must be copied verbatim. Use ordinary file reading
for disk-backed results. A navigation result does not imply that this backend
supports symbol inspection or mutation.

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
