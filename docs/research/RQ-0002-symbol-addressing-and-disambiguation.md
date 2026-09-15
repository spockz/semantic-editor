# RQ-0002: Symbol Addressing & Disambiguation

* **Status**: Open
* **Category**: Addressing & Coordinate Resolution
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

Compilers and Language Server Protocol methods strictly require exact line and character coordinates (`file.go:42:15`) or byte offsets. LLMs are poor at line counting and character arithmetic.

However, addressing by identifier name alone is ambiguous when multiple symbols share the same name (e.g. multiple methods named `Validate`, or identical property names across distinct structs/classes in the same package).

---

## 2. Exploration Paths

### A. Hierarchical Qualification Syntax

Define a natural identifier syntax that allows unambiguous resolution without coordinates:

* `TypeName.MethodName` (e.g. `TokenService.Validate`).
* Package-scoped: `pkg/auth::TokenService.Validate`.
* Function-local: `func ProcessRequest -> var token`.

### B. Two-Tier Disambiguation Protocol

1. The agent supplies an identifier query: `semedit rename --symbol Validate --to Verify`.
2. If only one match exists in scope $\rightarrow$ immediately resolve and execute.
3. If multiple matches exist $\rightarrow$ return candidate signatures and enclosing scopes to the agent:

   ```json
   {
     "ambiguous": true,
     "candidates": [
       {"id": "sym_1", "signature": "(s *SessionService) Validate() error", "file": "session.go"},
       {"id": "sym_2", "signature": "(t *TokenService) Validate() error", "file": "token.go"}
     ]
   }
   ```

4. The agent specifies the candidate ID in one follow-up turn without line hunting.

---

## 3. Sources & Prior Art

* **Tree-sitter Query Syntax**: [Tree-sitter Pattern Matching](https://tree-sitter.github.io/tree-sitter/using-parsers#pattern-matching-with-queries).
* **SCIP (Source Code Intelligence Protocol)**: [Sourcegraph SCIP Specification](https://github.com/sourcegraph/scip) — hierarchical symbol naming (`scip-go`, `scip-java`).
* **`gopls` Definition & Symbol RPCs**: [gopls documentation](https://pkg.go.dev/golang.org/x/tools/gopls).
