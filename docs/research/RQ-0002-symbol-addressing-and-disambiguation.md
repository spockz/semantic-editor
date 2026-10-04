# RQ-0002: Symbol Addressing & Disambiguation

* **Status**: Resolved
* **Category**: Addressing & Coordinate Resolution
* **Last Updated**: 2026-10-04

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
* **SCIP (Source Code Intelligence Protocol)**: [Sourcegraph SCIP Specification](https://github.com/sourcegraph/scip): hierarchical symbol naming (`scip-go`, `scip-java`).
* **`gopls` Definition & Symbol RPCs**: [gopls documentation](https://pkg.go.dev/golang.org/x/tools/gopls).

---

## 4. Resolution & Architecture Decision

Formalized and implemented across [ADR-0003: Symbol-Based Addressing Over File Coordinates](../adr/0003-symbol-based-addressing.md) and [ADR-0051: Semantic Code Inspection and Outline Operations](../adr/0051-semantic-code-inspection-and-outline-operations.md):

1. **Hierarchical Symbol Qualification**: Models address symbols by name and receiver qualification (`TypeName.MethodName`) directly, eliminating line and character coordinate arithmetic.
2. **Two-Tier Disambiguation Protocol**: The symbol resolver (`internal/symbol/resolver.go`) and neutral backend (`internal/backend/backend.go`) detect duplicate/overloaded symbols, returning structured ambiguous candidate sets (`Ambiguous: true`, `Candidates: [...]`) when unqualified queries match multiple definitions.
3. **Semantic Inspection & Outline Navigation**: `semantic_inspect_symbol`, `semantic_outline`, and `semantic_find_references` provide whole-symbol source retrieval, body-elided outlines, and reference discovery with persistent revision coordinates.
