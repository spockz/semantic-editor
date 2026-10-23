# ADR-0057: Exact Go Node Removal and Fail-Closed Safe Delete

* **Status**: Accepted
* **Date**: 2026-09-29

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Remove exact Go declarations and switch cases; safe-delete proves unused private constants, eligible functions, bounded variables, method-free defined types, and standalone aliases over a complete module source view. |
| description | Structural removal complements declaration and construct insertion, but deletion can change behavior while leaving valid syntax. Safe deletion combines lexical coverage of inactive source with compiler identity checks for active variants, method-set analysis for defined types, and alias identity checks. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

Structural removal complements declaration and construct insertion, but deletion can change behavior while leaving valid syntax. Go reference discovery through the language server covers active build variants only. Grouped declarations and comments also require precise source ownership when removing a single member.

## Decision

1. Register one Go `remove_construct` operation for top-level functions, methods, types, constants, variables, and switch cases. A declaration target uses `symbol`. A case uses `in_function` as its owner selector and may use `discriminator`, `target_case`, and `construct_path` to select the case.
2. `safe_delete` accepts unexported standalone constants, private top-level functions except `main` and `init`, and standalone private package variables and types under these boundaries:
   * A variable with an explicit type and no initializer may have any Go type. An initialized variable must have a basic type and a compiler-proven constant initializer.
   * A non-generic defined type is eligible only when both compiler-derived method sets, for `T` and `*T`, are empty, including methods promoted through embedded fields.
   * A non-generic type alias is eligible because removing it removes no distinct type or method set.

   The preflight scans all module Go source, including tests, generated sources, and inactive variants; same-name occurrences outside the target, source symlinks, nested modules, non-Go source, directives, and incomplete proofs refuse. Active packages and test variants must type-check, and compiler-resolved references must identify no uses of the declaration. Imports of `reflect`, `plugin`, or `unsafe` in module source also refuse; synthetic test runners are not module source.
3. Grouped declarations, exported declarations, methods, generic types or aliases, defined types with direct or promoted methods, referenced aliases, and effectful or unsupported variable initialization refuse without writing. Deleting the final use of a normal import in its file also refuses, preserving imported package initialization even when automatic import cleanup is disabled. Dot imports refuse. The receipt reports only completed checks, and MCP returns `isError=true` on refusal.
4. Ordinary structural removal deletes only the selected node, formats and parses the candidate source before publication, writes atomically, and reports diagnostic deltas. The receipt describes syntax and diagnostics only; it does not claim behavioral equivalence.
5. A grouped declaration removal preserves sibling specs and their attached comments. Removal from a value spec that declares multiple names is refused because deleting one name cannot preserve the expression binding safely. Attached declaration documentation is removed with a standalone declaration.
6. Case removal is syntax-validated but may change control flow. Ordinary removal from `iota` groups may change subsequent values. Callers needing behavioral safety must use a backend guarantee or inspect the resulting change themselves.

## Invariants

* `symbol` names only the declaration being removed; it never selects an enclosing function.
* `in_function` is an owner selector for nested case removal.
* `safe_delete=true` never mutates outside its accepted declaration classes or when any preflight fails.
* Defined-type deletion requires empty compiler-derived method sets for both `T` and `*T`; alias deletion does not apply this check to the aliased target.
* Zero-value variable deletion requires an explicit type and no initializer; initialized variable deletion remains restricted to basic types with a compiler-proven constant initializer.
* Refused safe deletion leaves source bytes unchanged and reports the checks actually completed.
* Safe-delete preflight scans active and inactive module Go files and captures their identities and content hashes. Before writing, it rechecks those captured files and the buffer's observed disk preimages; it does not lock or rescan package membership. A file added after discovery can remain unchanged and outside the proof, including a new caller. Compiler identity uses the compiler's own active file set, including test variants.
* Safe deletion is not eligible for shared-overlay batch execution. A batch publishes any prior buffered segment before the disk-backed safe-delete operation; active staging rejects the operation. The operation guards its captured source and compiler inputs but does not promise a package-wide transaction against concurrent membership changes.
* Ordinary removal validates syntax before atomic publication and reports diagnostics without inferring behavioral equivalence.

## Consequences

* Go users can remove exact declarations and switch cases through matching CLI and MCP schemas.
* Safe deletion covers unused private constants, eligible functions, bounded variables, method-free defined types, and unreferenced standalone aliases. Deletion of exported declarations, methods, grouped types, defined types with methods, and variables with effectful initializers remains out of scope.
* Same-name occurrences beyond the declaration name are conservative blockers, including self-recursion and occurrences a Go resolver would bind to another object.
* Grouped const removal may affect `iota`, and case removal may alter behavior; these remain explicit structural operations rather than behavioral proofs.
