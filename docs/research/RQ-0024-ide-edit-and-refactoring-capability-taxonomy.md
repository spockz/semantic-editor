# RQ-0024: IDE Edit and Refactoring Capability Taxonomy

* **Status**: Open
* **Category**: Product & Architecture
* **Date**: 2026-09-17

---

## Context

IDE documentation groups semantic operations into capability families rather than individual commands. `semedit` needs the same vocabulary to distinguish implemented operations from future work without treating a language-specific command as a universal promise.

## Research Questions

1. Which IDE capability families should `semedit` expose as edit or refactoring capabilities?
2. Which families have a safe Go-first implementation boundary, and which require a language-server or compiler-specific backend?
3. Which capabilities need in-operation conflict detection or atomic multi-file mutation, and which previews belong to the caller or controller?
4. How should safe deletion be expressed and bounded across structural node types and language backends?
5. Which high-intent composite operations (e.g. scaffolding test fixtures, implementing interface/trait contracts, member additions) should be exposed to eliminate multi-turn refactoring loops without causing engine bloat or cross-language schema pollution?
6. How should composite tools balance universal developer intent against target-language syntactic and idiomatic differences?

## Initial Capability Taxonomy

| IDE Capability Family | Current `semedit` Mapping | Research Outcome Needed |
| :--- | :--- | :--- |
| Navigation and symbol lookup | Implemented | Keep language-specific symbol qualification explicit. |
| Rename | Go workspace; Rust and Java selected-file | Define the path from selected-file to transactional multi-file rename. |
| Structural insertion, replacement, and removal | Go declarations, bodies, loops, switch cases; structural removals planned | Complete statement and collection-element insertion plus exact-node removals paired with those insertions. |
| High-intent composite operations | File scaffold with package infer | Map universal intents (`implement_contract`, `add_member`, `scaffold_fixture`) to language-specific backend realizations. |
| Extract and introduce | Unimplemented | Define expression selection, data-flow, and control-flow safety. |
| Inline | Unimplemented | Define side-effect and evaluation-order constraints. |
| Move and copy | Unimplemented | Define reference completeness and resource-operation transaction rules. |
| Signature and type transformation | Unimplemented | Define call-graph, compatibility, and migration policy. |
| Hierarchy refactoring | Unimplemented | Qualify by language rather than inventing inheritance concepts for Go. |
| Preview and conflict handling | Direct batches and state-journaled staging report diffs/deltas; snapshots provide undo | Keep speculative workspace execution and candidate previews with the caller/controller; revisit only for a concrete workflow. |

## Finding: Structural Removal and Safe Delete (2026-09-24)

Deletion should be offered as the structural counterpart to insertion, with removal operations targeting exact declarations, statements, cases, or collection elements. Safe deletion is an option on a supported removal operation, such as `safe_delete: true`; it is not a parallel family of safe-delete methods.

Without `safe_delete`, the operation removes only the requested structural node and still validates the result and reports diagnostics. With `safe_delete`, the backend performs the strongest relevant preflight it supports, including reference completeness and resource constraints where applicable. It must refuse the mutation if the requested safety checks fail or cannot establish the advertised guarantee, and report which checks ran. For statements, cases, and collection elements, structural or diagnostic validation does not prove behavioral equivalence.

The default for `safe_delete` and the required checks remain operation- and backend-specific. For direct multi-file edits, preserve RQ-0025's boundary: semantic-editor applies the authorized operation and reports its diff and diagnostics; speculative preview and workspace-copy workflows belong to the caller or controller. Reference and resource preflights remain part of safe deletion itself.

## Finding: High-Intent Composite Operations & Cross-Language Alignment (2026-09-25)

To reduce agent turn counts and token churn, refactoring workflows often call for composite mutations (e.g. creating a test file with sample test cases and assertions, or generating all required methods to satisfy an interface). However, baking framework- or language-specific composites into the core engine creates severe design tension:

1. **The Language Specificity Dilemma**:
   * Generic low-level tools (`replace_file_content`, line edits) impose heavy token taxes and context loss.
   * Overly specialized tools (`scaffold_cobra_subcommand`, `add_spring_controller_endpoint`) cause engine bloat, tight version coupling, and schema pollution for other languages.
2. **Universal Semantic Intents vs. Language Syntactic Realizations**:
   Instead of inventing divergent tools per language, composite operations should be defined around **universal developer refactoring intents**:
   * `semantic_add_member`: Adds a receiver method or struct field in Go; adds a method or property in Java/Rust.
   * `semantic_implement_contract`: Generates missing method declarations satisfying an `interface` in Go; generates `@Override` stubs in Java or `impl Trait` blocks in Rust.
   * `semantic_scaffold_fixture`: Creates canonical test file structures across languages (`testing.T` tables in Go, `@Test` methods in JUnit/Java).
   The central operation registry routes the universal intent to the registered language backend adapter (ADR-0021).
3. **Dynamic MCP Tool Scoping (ADR-0009, ADR-0021)**:
   To prevent attention degradation and schema dilution from an oversized tool catalog, the MCP server must dynamically filter `tools/list` using detected repository languages, exposing only the composite operations supported by the active backend.
4. **Engine Primitives vs. Agent Skills Boundary (RQ-0023)**:
   Composites that depend on third-party frameworks (e.g. Cobra, Spring, Akka) remain strictly in Tier 2 (Agent Skills in `skills/*/SKILL.md`), which steer the LLM to orchestrate clean Tier 1 engine primitives (`semantic_replace_loop`, `semantic_replace_decl`, `semantic_insert_case`) without hardcoding external libraries into the Go compiler/LSP engine.

## Sources

* [IntelliJ IDEA refactoring catalog](https://www.jetbrains.com/help/idea/refactoring-source-code.html)
* [Eclipse JDT refactoring actions](https://help.eclipse.org/latest/topic/org.eclipse.jdt.doc.user/reference/ref-menu-refactor.htm)
* [Visual Studio Code refactoring](https://code.visualstudio.com/docs/editing/refactoring)

## Next Steps

* Use this taxonomy in capability matrices, ADRs, and feature proposals.
* Define exact-node removal counterparts for the structural insertion operations, with revision-aware targeting.
* Specify `safe_delete` checks, defaults, and refusal behavior per removal operation and backend; do not expose a second method family.
* Design universal composite schemas for `semantic_implement_contract` and `semantic_add_member` under the central operation registry.
* Apply RQ-0023 and RQ-0025 boundaries: keep framework-specific patterns in Agent Skills and speculative workspace isolation in the caller/controller.
