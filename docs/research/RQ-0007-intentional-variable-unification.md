# RQ-0007: Intentional Variable Unification

* **Status**: Open
* **Category**: Semantics & Refactoring
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

Standard compiler refactoring tools (`gopls rename`, `gorename`, `rust-analyzer rename`) are built with strict safety checks: if renaming variable `x` to `y` introduces a collision with an existing declaration of `y` (or shadows an outer `y`), the tool aborts with an error.

However, agents frequently refactor code to **eliminate duplication** (e.g. coalescing two redundant local variables, merging duplicate error handlers, or unifying identical config structs).

---

## 2. Exploration Paths

### A. Dedicated `--unify` Mode

Design an explicit unification command or flag:

```bash
semedit unify --from "x" --into "y" --scope "func ProcessData"
```

The operation executes in three steps:

1. Re-bind all read sites of `x` to `y`.
2. Verify that type signatures and scopes are compatible.
3. Remove the declaration of `x` (and its assignment, if redundant).
4. Run compiler type-checking to verify that the coalesced symbol is valid.

### B. Two-Pass Semantic Rewriting via CST

If the underlying LSP strictly refuses the rename:

1. Use Tree-sitter / `ast-grep` to perform the scoped identifier replacement and remove the duplicate declaration.
2. Hand control back to the LSP to verify that the resulting symbol table has zero diagnostic errors.

---

## 3. Sources & Prior Art

* **Russ Cox's `rf` (Refactoring Tool for Go)**: [rsc.io/rf documentation](https://pkg.go.dev/rsc.io/rf): scriptable Go transformations supporting symbol moves and inlining.
* **OpenRewrite Structural Substitutions**: [OpenRewrite Java Recipes](https://docs.openrewrite.org): coalescing duplicate types.
* **Go `gorename` Conflict Detection**: [golang.org/x/tools/cmd/gorename](https://pkg.go.dev/golang.org/x/tools/cmd/gorename).
