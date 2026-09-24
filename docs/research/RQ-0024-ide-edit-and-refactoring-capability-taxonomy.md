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

## Initial Capability Taxonomy

| IDE Capability Family | Current `semedit` Mapping | Research Outcome Needed |
| :--- | :--- | :--- |
| Navigation and symbol lookup | Implemented | Keep language-specific symbol qualification explicit. |
| Rename | Go workspace; Rust and Java selected-file | Define the path from selected-file to transactional multi-file rename. |
| Structural insertion, replacement, and removal | Go declarations, bodies, and switch cases; structural removals planned | Complete statement and collection-element insertion plus exact-node removals paired with those insertions. |
| Extract and introduce | Unimplemented | Define expression selection, data-flow, and control-flow safety. |
| Inline | Unimplemented | Define side-effect and evaluation-order constraints. |
| Move and copy | Unimplemented | Define reference completeness and resource-operation transaction rules. |
| Signature and type transformation | Unimplemented | Define call-graph, compatibility, and migration policy. |
| Hierarchy refactoring | Unimplemented | Qualify by language rather than inventing inheritance concepts for Go. |
| Preview and conflict handling | Direct batches report final diffs; snapshots provide conflict-checked undo | Keep speculative workspace execution and candidate previews with the caller/controller; revisit only for a concrete workflow. |

## Finding: Structural Removal and Safe Delete (2026-09-24)

Deletion should be offered as the structural counterpart to insertion, with removal operations targeting exact declarations, statements, cases, or collection elements. Safe deletion is an option on a supported removal operation, such as `safe_delete: true`; it is not a parallel family of safe-delete methods.

Without `safe_delete`, the operation removes only the requested structural node and still validates the result and reports diagnostics. With `safe_delete`, the backend performs the strongest relevant preflight it supports, including reference completeness and resource constraints where applicable. It must refuse the mutation if the requested safety checks fail or cannot establish the advertised guarantee, and report which checks ran. For statements, cases, and collection elements, structural or diagnostic validation does not prove behavioral equivalence.

The default for `safe_delete` and the required checks remain operation- and backend-specific. For direct multi-file edits, preserve RQ-0025's boundary: semantic-editor applies the authorized operation and reports its diff and diagnostics; speculative preview and workspace-copy workflows belong to the caller or controller. Reference and resource preflights remain part of safe deletion itself.

## Sources

* [IntelliJ IDEA refactoring catalog](https://www.jetbrains.com/help/idea/refactoring-source-code.html)
* [Eclipse JDT refactoring actions](https://help.eclipse.org/latest/topic/org.eclipse.jdt.doc.user/reference/ref-menu-refactor.htm)
* [Visual Studio Code refactoring](https://code.visualstudio.com/docs/editing/refactoring)

## Next Steps

* Use this taxonomy in capability matrices, ADRs, and feature proposals.
* Define exact-node removal counterparts for the structural insertion operations, with revision-aware targeting.
* Specify `safe_delete` checks, defaults, and refusal behavior per removal operation and backend; do not expose a second method family.
* Apply RQ-0025's direct-execution and controller-owned workspace-view boundary when designing multi-file refactorings; reopen it only for a concrete speculative workflow.
