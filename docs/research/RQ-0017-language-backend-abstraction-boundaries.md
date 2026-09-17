# RQ-0017: Language Backend Abstraction Boundaries and Package Placement

* **Status**: Resolved
* **Date**: 2026-09-16
* **Category**: Architecture & Modular Design

---

## 1. Context & Motivation

In [ADR-0012](../adr/0012-access-modifiers-and-section-placement.md) and [RQ-0016](RQ-0016-cross-language-access-modifiers-and-section-clustering.md), we introduced `LanguageBackend` as a capability declaration and access modifier resolution interface:

```go
type LanguageBackend interface {
    Name() string
    SupportedAccessModifiers() []AccessModifier
    ResolveEffectiveAccess(mod AccessModifier, identifier string) (AccessModifier, error)
    ValidateModifier(mod AccessModifier, identifier string) error
}
```

The original access-modifier interface and its default implementation `GolangBackend` reside directly in `internal/astedit/access.go`. Common ingress operations additionally need a language boundary that does not depend on Go AST coordinates.

However, the concept of a programming language backend encompasses broader domain responsibilities than AST-level insertion alone:

* **Symbol Addressing & Coordinates**: Parsing identifiers, resolving receivers, and distinguishing types vs. functions (`internal/symbol`).
* **Compiler & LSP Tooling**: Executing compiler diagnostics, rename transformations, and import optimization (`internal/adapters/golang`).
* **Access & Visibility Modeling**: Validating modifiers, section ordering rules, and keyword constraints (`internal/astedit`).
* **Dependency Tooling**: Module file acquisition and package tidying (`internal/adapters/golang`).

This research spike investigates the appropriate architectural boundary, timing, and package topology for extracting language abstractions from `internal/astedit`.

---

## 2. Competing Architectural Options

### Option A: Retain in `internal/astedit` Until Multi-Subsystem Demand (Current Strategy)

* **Design**: Keep `LanguageBackend` and `GolangBackend` inside `internal/astedit/access.go` while `astedit` is the sole consumer of access modifier capabilities.
* **Pros**:
  * Prevents premature abstraction layers and package proliferation.
  * Zero additional import boundaries or package circular dependencies.
  * Code remains local to the immediate consumers (`InsertFunction`, `InsertType`, `InsertDecl`).
* **Cons**:
  * If `internal/symbol` or `internal/pipeline` needs access modifier rules in the future, it creates a dependency on `internal/astedit`.

### Option B: Dedicated Core Package (`internal/language` or `internal/backend`)

* **Design**: Create `internal/language` defining `Language`, `Backend`, `AccessModifier`, and capabilities matrix. Subsystems (`astedit`, `symbol`, `pipeline`, `adapters`) import `internal/language`.
* **Pros**:
  * Clean separation of concerns; language definitions are decoupled from AST mutation algorithms.
  * Extensible plugin registry for new languages (Java, TypeScript, Python, Rust).
* **Cons**:
  * Introduces an extra package when only Go is currently implemented end-to-end.
  * Increases cognitive overhead and package graph depth prematurely.

---

## 3. Evaluation Criteria & Trigger Conditions

We evaluate the decision to extract `LanguageBackend` against the following concrete triggers:

1. **Secondary Subsystem Consumer**: When a second package (such as `internal/symbol` or `internal/adapters`) requires access modifier resolution or language capability querying.
2. **Second Production Language Backend**: When a second language backend (e.g., Java or TypeScript) is introduced to `semedit`.
3. **Cross-Subsystem Configuration**: When language backend options are surfaced as global CLI flags or MCP configuration profiles.

Until at least one trigger condition is satisfied, `LanguageBackend` remains anchored in `internal/astedit/access.go`.

---

## 4. Resolution

The accepted boundary is `internal/backend`. It owns language IDs, project selection, UTF-16-neutral protocol locations and diagnostics, operation capabilities, typed errors, a backend registry, and the ingress-facing service. The registered Go adapter delegates existing `internal/symbol`, `internal/adapters/golang`, and `internal/pipeline` behavior. This satisfies the second-backend trigger for the common service boundary while leaving Go AST insertion APIs in `internal/astedit`.

`auto` selects Go from `.go` files or `go.mod`/`go.work`; no other language backend is registered. Service checks consult backend capabilities before dispatch. ADR-0003 is qualified to permit backend-native lookup without requiring broken-source fallback.

## 5. Next Steps

* Monitor coupling between `internal/astedit`, `internal/symbol`, and `internal/adapters/golang`.
* Revisit package extraction when implementing the first non-Go language prototype.
