# Java Semantic Refactoring Guide

This reference explains how to execute bounded and multi-step Java code transformations using `semedit` primitives and Maven integration. Read this document only when refactoring or modernizing Java source files.

---

## 1. Macro Refactorings in Java

`semedit` core operations for Java are indexed in the primary `SKILL.md` table. High-level patterns (e.g., Class $\to$ Record, Record $\to$ Class, Encapsulate Field) are composed by the agent using core operations and verified via Maven.

---

## 2. Java Refactoring Patterns

### Pattern A: Modernizing Class to Record (Class $\to$ Record)

When migrating an immutable data carrier class (POJO/DTO) to a Java 16+ `record`:

1. **Preflight & Inspection**:
   - Use `semantic_outline` and `semantic_inspect_symbol` to inspect the target class: ensure fields are `private final`, state is immutable, and the class does not extend an external class (`extends Object` only).
   - Verify serialization/reflection dependencies (e.g., JPA entities require mutable classes with no-arg constructors and cannot become records; Jackson DTOs may need `@JsonNaming` or record creator annotations).
2. **Atomic Transformation**:
   - Convert the class header to record syntax:

     ```java
     // Before
     public final class Point {
         private final int x;
         private final int y;
         public Point(int x, int y) { this.x = x; this.y = y; }
         public int getX() { return x; }
         public int getY() { return y; }
     }

     // After
     public record Point(int x, int y) {}
     ```

   - Eliminate redundant canonical constructors, standard getters (`getX()`), `equals()`, `hashCode()`, and `toString()`.
   - Preserve validation logic in a compact constructor:

     ```java
     public record BoundedRange(int min, int max) {
         public BoundedRange {
             if (min > max) throw new IllegalArgumentException("min > max");
         }
     }
     ```

3. **Call-Site Adjustment & Renaming**:
   - Update call sites from JavaBean getter access (`point.getX()`) to record component access (`point.x()`). Use `semantic_rename` with `file` and `trust_workspace: true` when renaming symbols across trusted workspace files.
4. **Verification**:
   - Run `semantic_verify` with `format: true` and `organize_imports: true` to format the modified file and clean unused imports.
   - Execute `semantic_maven_compile` with `trust_workspace: true` to ensure zero compilation or caller breaks.

---

### Pattern B: De-recording (Record $\to$ Class)

When a `record` outgrows its immutable constraints (e.g., requires mutable caches, lazy initialization, or inheritance hierarchy):

1. **Inspection**:
   - Use `semantic_lookup` and `semantic_inspect_symbol` to capture current record components and constructor invariants.
2. **Expansion**:
   - Convert `record Point(int x, int y)` into a standard class with explicit `private final` fields.
   - Synthesize canonical constructor, component accessors (`public int x() { return x; }`), and explicit `equals`/`hashCode`/`toString` implementations.
3. **Verification**:
   - Format source with `semantic_verify` and compile via `semantic_maven_compile`. Because existing callers invoke `point.x()`, accessor compatibility is maintained without breaking downstream callers.

---

### Pattern C: Encapsulate Field

When converting exposed public or package-private fields into encapsulated properties:

1. **Preflight Inspection**:
   - Use `semantic_lookup` and `semantic_inspect_symbol` to check field access and annotations.
2. **Accessor Addition & Refactoring**:
   - Introduce explicit getter: `public T getFieldName() { return this.fieldName; }`.
   - Introduce explicit setter: `public void setFieldName(T value) { this.fieldName = value; }`.
   - Update field visibility from `public` to `private`.
   - Use `semantic_rename` with `file` and `trust_workspace: true` to rename callers where method signatures match.
3. **Verification**:
   - Format with `semantic_verify` and run `semantic_maven_test` to confirm behavior and unit test passes.
