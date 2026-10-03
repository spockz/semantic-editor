# ADR-0057: Exact Go Node Removal and Fail-Closed Safe Delete

* **Status**: Accepted
* **Date**: 2026-09-29

## Context

Structural removal complements declaration and construct insertion, but deletion can change behavior while leaving valid syntax. Go reference discovery through the language server covers active build variants only. Grouped declarations and comments also require precise source ownership when removing a single member.

## Decision

1. Register one Go `remove_construct` operation for top-level functions, methods, types, constants, variables, and switch cases. A declaration target uses `symbol`. A case uses `in_function` as its owner selector and may use `discriminator`, `target_case`, and `construct_path` to select the case.
2. `safe_delete` accepts unexported standalone constants, private top-level functions except `main` and `init`, and single private package variables of basic type with zero or compiler-proven constant initialization. It scans all module Go source, including tests, generated sources, and inactive variants; same-name occurrences outside the target, source symlinks, nested modules, non-Go source, directives, and incomplete proofs refuse. Active packages and test variants must type-check. Imports of `reflect`, `plugin`, or `unsafe` in module source also refuse; synthetic test runners are not module source.
3. Grouped declarations, exported declarations, methods, types, and effectful or unsupported variable initialization refuse without writing. Deleting the final use of a normal import in its file also refuses, preserving imported package initialization even when automatic import cleanup is disabled. Dot imports refuse. The receipt reports only completed checks, and MCP returns `isError=true` on refusal.
4. Ordinary structural removal deletes only the selected node, formats and parses the candidate source before publication, writes atomically, and reports diagnostic deltas. The receipt describes syntax and diagnostics only; it does not claim behavioral equivalence.
5. A grouped declaration removal preserves sibling specs and their attached comments. Removal from a value spec that declares multiple names is refused because deleting one name cannot preserve the expression binding safely. Attached declaration documentation is removed with a standalone declaration.
6. Case removal is syntax-validated but may change control flow. Ordinary removal from `iota` groups may change subsequent values. Callers needing behavioral safety must use a backend guarantee or inspect the resulting change themselves.

## Invariants

* `symbol` names only the declaration being removed; it never selects an enclosing function.
* `in_function` is an owner selector for nested case removal.
* `safe_delete=true` never mutates outside its accepted declaration classes or when any preflight fails.
* Refused safe deletion leaves source bytes unchanged and reports the checks actually completed.
* Safe-delete preflight covers active and inactive module Go files and rechecks their identities and contents before writing. Compiler identity uses the compiler's own file set, including test variants.
* Standalone safe deletion owns a source buffer; batches reuse theirs. Compiler dependency inputs, manifests, auxiliary files, and source membership remain bound through validation and publication.
* Ordinary removal validates syntax before atomic publication and reports diagnostics without inferring behavioral equivalence.

## Consequences

* Go users can remove exact declarations and switch cases through matching CLI and MCP schemas.
* Safe deletion covers a bounded set of unused private declarations. General type/member deletion and runtime initializers remain open work.
* Same-name occurrences beyond the declaration name are conservative blockers, including self-recursion and occurrences a Go resolver would bind to another object.
* Grouped const removal may affect `iota`, and case removal may alter behavior; these remain explicit structural operations rather than behavioral proofs.
