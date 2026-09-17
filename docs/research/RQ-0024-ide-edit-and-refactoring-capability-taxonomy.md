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
3. Which capabilities need preview, conflict detection, or an atomic multi-file commit before they can be offered?

## Initial Capability Taxonomy

| IDE Capability Family | Current `semedit` Mapping | Research Outcome Needed |
| :--- | :--- | :--- |
| Navigation and symbol lookup | Implemented | Keep language-specific symbol qualification explicit. |
| Rename | Go workspace; Rust and Java selected-file | Define the path from selected-file to transactional multi-file rename. |
| Structural insertion and replacement | Go declarations, bodies, and switch cases | Complete statement and collection-element insertion. |
| Extract and introduce | Unimplemented | Define expression selection, data-flow, and control-flow safety. |
| Inline | Unimplemented | Define side-effect and evaluation-order constraints. |
| Move, copy, and safe delete | Unimplemented | Define reference completeness and resource-operation transaction rules. |
| Signature and type transformation | Unimplemented | Define call-graph, compatibility, and migration policy. |
| Hierarchy refactoring | Unimplemented | Qualify by language rather than inventing inheritance concepts for Go. |
| Preview and conflict handling | Snapshot and undo only | Determine a bounded inspect, dry-run, and commit protocol. |

## Sources

* IntelliJ IDEA refactoring catalog: https://www.jetbrains.com/help/idea/refactoring-source-code.html
* Eclipse JDT refactoring actions: https://help.eclipse.org/latest/topic/org.eclipse.jdt.doc.user/reference/ref-menu-refactor.htm
* Visual Studio Code refactoring: https://code.visualstudio.com/docs/editing/refactoring

## Next Steps

* Use this taxonomy in capability matrices, ADRs, and feature proposals.
* Resolve the preview/commit question in RQ-0025 before multi-file edit or refactoring families are advertised.
