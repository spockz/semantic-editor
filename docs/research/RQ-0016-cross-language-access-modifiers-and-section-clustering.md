# RQ-0016: Cross-Language Access Modifiers and Section Clustering

* **Status**: Open
* **Date**: 2026-09-16
* **Category**: Semantics & Architecture

---

## 1. Problem Statement

Code transformation engines operating across diverse programming languages encounter divergent paradigms for visibility and access control:

1. **Naming-Derived Visibility (e.g., Go)**:
   * Visibility is strictly derived from the lexical casing of the identifier (`ast.IsExported`): uppercase indicates exported/public (`Validate`), lowercase indicates unexported/private (`validate`).
   * There are no keywords such as `public`, `private`, or `protected`.
   * File organization follows the canonical invariant that public declarations precede private declarations.

2. **Keyword-Based Access Modifiers (e.g., Java, C#, TypeScript, C++)**:
   * Visibility is specified through explicit modifiers: `public`, `private`, `protected`, and `package-private` (default/unspecified in Java).
   * Casing does not determine visibility (e.g. `public void validate()` is public despite lowercase).
   * Within a class or file boundary, public methods conventionally precede protected, package-private, and private methods.

When an LLM planner emits a semantic mutation intent (such as `insert_function` or `insert_type`), it requires an abstracted `access_modifier` attribute that:

* Generalizes cleanly across languages without forcing language-specific syntax onto the planner.
* Allows automatic inference (`"infer"`) based on target language idioms (casing in Go, modifiers in Java).
* Enables each language backend to declare and enforce which access modifiers it supports, rejecting invalid modifiers with helpful diagnostics.
* Determines section clustering (placing public methods in public sections, private methods in private sections, even when grouping by receiver or class).

---

## 2. Access Modifier Abstraction Model

We define a canonical enumeration for access modifiers:

| Modifier | Semantic Definition | Supported in Go | Supported in Java |
| :--- | :--- | :---: | :---: |
| `infer` (default) | Infer visibility from identifier casing or language-level declaration keywords. | Yes | Yes |
| `public` | Accessible outside the declaring package/module. | Yes | Yes |
| `private` | Accessible exclusively within the declaring file/class. | Yes | Yes |
| `protected` | Accessible within the declaring class and derived subclasses. | **No** (Error) | Yes |
| `package-private` | Accessible within the declaring package, but not outside. | **No** (Error) | Yes |

### Backend Capability Declaration

Every language backend implements an interface declaring its supported access modifiers:

```go
type AccessModifier string

const (
    AccessModifierInfer          AccessModifier = "infer"
    AccessModifierPublic         AccessModifier = "public"
    AccessModifierPrivate        AccessModifier = "private"
    AccessModifierProtected      AccessModifier = "protected"
    AccessModifierPackagePrivate AccessModifier = "package-private"
)

type LanguageBackend interface {
    SupportedAccessModifiers() []AccessModifier
    ValidateAccessModifier(mod AccessModifier, identifier string) error
    ResolveEffectiveAccess(mod AccessModifier, identifier string) (AccessModifier, error)
}
```

---

## 3. Section Clustering & Placement Invariants

When inserting a method or function:

1. **Strict Section Partitioning**:
   * A method with effective access `public` must be placed within the public section.
   * A method with effective access `private` must be placed within the private section.
2. **Receiver Clustering within Section**:
   * In Go, methods on a receiver `Server` may be grouped together, but must still observe the public-precedes-private invariant.
   * If adding a private method to `Server`, it must land after the receiver's public methods, clustering with other private methods of `Server` (or at `private_start` / `private_end`).
   * If adding a public method to `Server`, it must land within the public method group of `Server`, preceding all private methods.
3. **Class Body Partitioning (Java/C#)**:
   * When inserting into a class body, methods are clustered into visibility bands: `public` $\to$ `protected` $\to$ `package-private` $\to$ `private`.

---

## 4. Open Research Questions

1. **Receiver Definition Visibility vs. Method Visibility**:
   * In Go, a public struct `type Server struct` can have private methods (`func (s *Server) log()`), and a private struct `type config struct` can have methods with uppercase names that satisfy public interfaces.
   * How should receiver clustering balance type-adjacent placement against file-level public/private sections?
2. **Interface Implementations**:
   * When a method is inserted to implement a specific interface, should the language backend automatically enforce the access modifier required by that interface?
3. **Constructor Proximity vs. Visibility Sections**:
   * Constructors (`NewServer`) are public functions that return a receiver pointer. Should public methods immediately follow the constructor, or should type definitions precede constructors?

---

## 5. Next Steps

* Codify the backend capability contract and section placement rules in ADR-0012.
* Implement `AccessModifier` support in `internal/astedit` and `internal/adapters/golang`.
