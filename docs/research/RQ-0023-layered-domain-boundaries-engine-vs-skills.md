# RQ-0023: Layered Domain Boundaries: Generic AST Engine vs Framework-Specific Agent Skills

* **Status**: Open
* **Category**: Architecture & Agent Steering
* **Date**: 2026-09-17

---

## 1. Context & Architectural Boundary

Modern programming languages and frameworks feature specialized idioms and architectural conventions:

* **Scala / Akka**: Actors typically define their protocol messages (commands, events, replies) as immutable `case class` or `case object` definitions housed strictly inside the actor class's **companion object**.
* **Go / Cobra**: CLI subcommands are initialized in `init()` blocks or factory functions with flags registered directly on the `*cobra.Command` instance.
* **React / Next.js**: State hooks, effect hooks, and event handlers follow specific topological ordering inside functional component closures.
* **Java / Spring**: Dependency injection fields (`@Autowired`), lifecycle hooks (`@PostConstruct`), and endpoint mapping methods follow explicit container conventions.

Attempting to bake framework-specific patterns (e.g., Akka actor message placement, Cobra command registration, Spring bean annotations) into the core `semedit` AST transformation engine creates severe anti-patterns:
1. **Engine Bloat**: Explosion of specialized tool flags, ad-hoc AST walkers, and brittle pattern matchers.
2. **Grammar Rigidity**: The engine becomes coupled to specific framework versions and third-party libraries.
3. **Loss of Composability**: Breaks the clean separation between generic language semantics and application architecture.

---

## 2. The Two-Tier Architecture: Engine Primitives vs Specialized Skills

We formalize a strict two-tier separation of concerns:

```text
 ┌──────────────────────────────────────────────────────────────────┐
 │  Tier 2: Specialized Agent Skills (Context / Steering Layer)      │
 │  - "Semantically Editing Akka / Scala" (skills/akka/SKILL.md)     │
 │  - "Semantically Editing Cobra CLIs" (skills/cobra/SKILL.md)     │
 │  - Injects idiomatic rules: e.g. "Place protocol case classes in  │
 │    the companion object; use semantic_insert_decl(target: obj)"  │
 └───────────────────────────────┬──────────────────────────────────┘
                                 │ orchestrates with domain intent
                                 ▼
 ┌──────────────────────────────────────────────────────────────────┐
 │  Tier 1: semedit Host Engine (Deterministic AST Execution Layer) │
 │  - Exposes language-level structural containers & slots:         │
 │    companion object, class body, statement_list, switch, clauses  │
 │  - Validates syntax, access modifiers, and format in memory       │
 │  - Atomic disk synchronization (ADR-0010)                        │
 └──────────────────────────────────────────────────────────────────┘
```

### Tier 1: The Core `semedit` Engine (Language-Level Grammar & Containers)
The engine only understands language grammar constructs, structural containers, and semantic roles:
* In Scala: `class`, `trait`, `object` (including companion objects), `def`, `val/var`, `case class`.
* In Java: `class`, `interface`, `record`, fields, methods, constructors, inner classes.
* In Go: `package`, `import`, `type`, `func`, `method`, `const/var`.
* Inside blocks: `statement_list`, `expression`, `optional_statement`, `clause_list`.

The engine provides discovery (`semantic_supported_locations`) and targeted mutations (`insert_declaration`, `insert_statement`, `replace_expression`, `insert_case`).

### Tier 2: Agent Skills (Framework & Architecture-Specific Steering)
Higher-level framework rules, architectural patterns, and design conventions belong in **Agent Skills** (e.g. `skills/akka/SKILL.md`, `skills/cobra/SKILL.md`):
* Teaches the LLM planner the domain pattern: *"When adding a new command to an Akka Actor `CartActor`, declare the command as a `case class` inside `object CartActor` using `semantic_insert_type(owner: 'CartActor', container: 'companion_object')`"*.
* Recommends multi-tool composition chains to accomplish full domain workflows without bloating tool schemas.

---

## 3. Invariants & Guidelines for Tool Authors

1. **Language-Grammar Scope**: A core `semedit` tool or placement enum must be justifiable strictly in terms of the language's formal specification or standard compiler grammar, never a third-party framework or library.
2. **Extensibility via Generic Containers**: Language backends must expose idiomatic structural containers (such as Scala's companion objects or Java's static nested classes) as generic targets so skills can anchor to them.
3. **Explicit Skill Documentation**: When a language or ecosystem relies heavily on convention (such as Akka protocols or Go CLI commands), `semedit` documentation must explicitly direct users and agents to encapsulate those conventions in dedicated skills rather than extending the core engine.

---

## 4. Next Steps

* Reference this boundary in ADR-0001 (Intent-Driven Orchestration) and ADR-0012 (Access Modifiers).
* Maintain clear skill scaffolding patterns for downstream language ecosystems.
