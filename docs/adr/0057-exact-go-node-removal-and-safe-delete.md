# ADR-0057: Exact Go Node Removal and Fail-Closed Safe Delete

* **Status**: Accepted
* **Date**: 2026-09-29

## Context

Structural removal complements declaration and construct insertion, but deletion can change behavior while leaving valid syntax. Go reference discovery through the language server covers active build variants only. Grouped declarations and comments also require precise source ownership when removing a single member.

## Decision

1. Register one Go `remove_construct` operation for top-level functions, methods, types, constants, variables, and switch cases. A declaration target uses `symbol`. A case uses `in_function` as its owner selector and may use `discriminator`, `target_case`, and `construct_path` to select the case.
2. `safe_delete` is an option on this same operation. Its only successful class is an unexported standalone constant. The preflight parses and scans every Go source file under the nearest module root, including tests, generated sources, and inactive build-tag variants. It rejects source-set symlinks, unparseable files, relevant `//go:` directives, any same-name identifier occurrence outside the selected declaration, and any change to the scanned file set or contents before publication. The scan follows the Go loader's ignored-directory and vendor boundaries. A standalone constant has no runtime resource lifecycle and cannot affect sibling `iota` values.
3. `safe_delete=true` refuses every other node kind, grouped constant, exported constant, and incomplete preflight without writing. The receipt carries a refusal status and the checks actually completed; MCP reports the refusal with `isError=true` and structured check fields.
4. Ordinary structural removal deletes only the selected node, formats and parses the candidate source before publication, writes atomically, and reports diagnostic deltas. The receipt describes syntax and diagnostics only; it does not claim behavioral equivalence.
5. A grouped declaration removal preserves sibling specs and their attached comments. Removal from a value spec that declares multiple names is refused because deleting one name cannot preserve the expression binding safely. Attached declaration documentation is removed with a standalone declaration.
6. Case removal is syntax-validated but may change control flow. Ordinary removal from `iota` groups may change subsequent values. Callers needing behavioral safety must use a backend guarantee or inspect the resulting change themselves.

## Invariants

* `symbol` names only the declaration being removed; it never selects an enclosing function.
* `in_function` is an owner selector for nested case removal.
* `safe_delete=true` never mutates outside its accepted standalone-constant class or when any preflight fails.
* Refused safe deletion leaves source bytes unchanged and reports the checks actually completed.
* Safe-constant preflight covers active and inactive Go files in the module source set and rechecks file identities and content before writing.
* Ordinary removal validates syntax before atomic publication and reports diagnostics without inferring behavioral equivalence.

## Consequences

* Go users can remove exact declarations and switch cases through matching CLI and MCP schemas.
* Safe deletion is useful for unexported standalone constants; all other constructs remain an explicit fail-closed scope until the backend can prove a broader guarantee.
* Same-name occurrences are conservative blockers even when a Go resolver would determine that they refer to another object.
* Grouped const removal may affect `iota`, and case removal may alter behavior; these remain explicit structural operations rather than behavioral proofs.
