# ADR-0012: Cross-Language Access Modifiers and Section Placement

* **Status**: Accepted
* **Date**: 2026-09-16

---

## Context

Refactoring and AST insertion tools (`insert_function`, `insert_type`, `insert_decl`) must accommodate varying visibility and access control models across target programming languages:

1. **Naming-Derived Visibility (e.g., Go)**:
   * Visibility is dictated strictly by the capitalization of the declared identifier (`ast.IsExported`).
   * No access modifier keywords (`public`, `private`, `protected`) exist in the language grammar.
   * File structure conventionally maintains a clean partition where public (exported) declarations precede private (unexported) declarations.

2. **Keyword-Based Visibility (e.g., Java, C#, TypeScript)**:
   * Visibility is dictated by explicit keywords (`public`, `private`, `protected`, and package-private when absent).
   * Identifier casing does not alter access semantics.
   * Within class or file boundaries, declarations are organized into visibility tiers.

Without a unified access modifier abstraction, agent planners either attempt to supply invalid keywords to Go (e.g. `public func Foo()`) or fail to enforce proper file-level section ordering.

---

## Decision

1. **Unified `access_modifier` Attribute**:
   All insertion and refactoring operations accept an `access_modifier` attribute supporting:
   * `"infer"` (default): Infer visibility dynamically according to target language rules.
   * `"public"`: Exported / globally visible.
   * `"private"`: Unexported / file- or class-scoped.
   * `"protected"`: Subclass-scoped.
   * `"package-private"`: Package-scoped without external export.

2. **Language Backend Capability Configuration**:
   Each language backend statically declares which access modifiers it supports:
   * **Go Backend**: Supports `{"infer", "public", "private"}`. Rejects `"protected"` and `"package-private"` with an immediate validation error.
   * **Java Backend**: Supports `{"infer", "public", "private", "protected", "package-private"}`.

3. **Dynamic Inference Mechanism (`"infer"`)**:
   * For Go: Evaluates the declared identifier's initial rune via `unicode.IsUpper` / `ast.IsExported`. Uppercase identifiers resolve to `public`; lowercase identifiers resolve to `private`.
   * For Java: Evaluates explicit keyword modifiers in the snippet, defaulting to `package-private` when absent.

4. **Strict Section Placement Invariant**:
   * A declaration with effective access `public` must be placed in the public section.
   * A declaration with effective access `private` must be placed in the private section.
   * In Go, public declarations strictly precede private declarations. When inserting a method for a receiver:
     * A public method clusters with the receiver's public methods (or at `public_end`).
     * A private method clusters with the receiver's private methods (or at `private_end`).
     * A private method must never be placed ahead of public declarations.

5. **Explicit Modifier Validation**:
   When an explicit access modifier is provided (e.g., `access_modifier: "public"` in Go), the engine validates that the snippet's identifier casing matches the requested modifier. A mismatch (e.g. `access_modifier: "public"` for `func helper()`) is rejected before touching disk.

---

## Invariants

* If an access modifier is supplied that is not supported by the active language backend (e.g. `protected` in Go), the operation must be rejected with zero disk mutation.
* Public declarations must never be inserted into a private section, and private declarations must never be inserted into a public section.
* When `infer` is supplied, the engine resolves effective visibility before calculating placement offsets.

---

## Consequences

* **Positive**: Generalizes across both naming-based (Go) and keyword-based (Java, TypeScript) languages; protects repository integrity against invalid modifiers; ensures consistent visibility partitioning.
* **Negative**: Agents cannot place private helper methods directly adjacent to an individual public method if it violates the file's public-precedes-private section layout.
