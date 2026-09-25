# ADR-0046: Construct-Level AST Replacement and Declarative Updates

* **Status**: Accepted
* **Date**: 2026-09-25

---

## Context

`semedit` originally separated coarse mutations into two extremes:

1. **Top-Level Declaration Insertion**: `semantic_insert_decl`, `semantic_insert_function`, and `semantic_insert_type` add new symbols to files.
2. **Whole-Function Body Replacement**: `semantic_replace_body` replaces the entire body of a function or method by name.

Dogfooding on real-world refactorings (ST-0035, ST-0036) revealed critical ergonomics and turn-efficiency issues when modifying existing code:

1. **Top-Level Constant/Variable Mutation Gap**: `semantic_replace_body` targets only `*ast.FuncDecl` nodes, rejecting package-level constants or variables with `symbol not found`. Meanwhile, `semantic_insert_decl` prepends new declarations without checking for symbol collisions, causing duplicate-symbol compiler errors (`redeclared in this block`) instead of updating the existing value.
2. **Whole-Body Replacement Friction**: Integration tests and data-pipeline handlers frequently span 100+ lines. When updating a localized loop (e.g. modifying an assertion list `for _, want := range []string{...}`), `semantic_replace_body` requires the LLM to emit the entire 140-line body, consuming excessive context tokens, risking boundary syntax errors (e.g. accidental trailing braces), and inviting hallucinated regressions in untouched sections of the function.
3. **Intent-Routing Mismatch**: When an agent's internal reasoning formulates an intent such as *"Update the assertion loop over `want` in test function `TestX`"*, the coarse nature of `semantic_replace_body` creates high semantic distance. The agent defaults to generic text search-and-replace (`replace_file_content`) to avoid regurgitating large functions.

`semantic_insert_case` demonstrated that targeting nested control flow constructs via semantic discriminators (`function` + `switch_on`) is reliable, deterministic, and token-efficient.

---

## Decision

We establish construct-level AST mutation and declarative updates across the engine, CLI, and MCP interfaces:

### 1. Construct-Level Loop Replacement (`semantic_replace_loop`)

Expose a dedicated construct replacement tool targeting `for` and `range` loops within functions:

```json
{
  "name": "semantic_replace_loop",
  "description": "Use this tool instead of replace_body or replace_file_content whenever modifying an existing for loop, range loop, or loop condition inside a function or method (e.g. updating assertion loops, range slices, or loop bounds). Operates directly on the targeted loop construct without regenerating the rest of the function body.",
  "parameters": {
    "properties": {
      "file": {
        "description": "Path to the Go source file",
        "type": "string"
      },
      "function": {
        "description": "Name of the containing function or method (e.g. 'TestWriteAssets' or '(*Server).Serve')",
        "type": "string"
      },
      "loop_on": {
        "description": "Optional expression or variable that identifies the loop (e.g. 'want', 'items', 'i := 0')",
        "type": "string"
      },
      "source": {
        "description": "Replacement Go loop code (e.g. 'for _, want := range [...] { ... }') or bare body statements if mode is body-only",
        "type": "string"
      },
      "auto_organize_imports": {
        "default": true,
        "description": "Automatically clean up and resolve imports after mutation",
        "type": "boolean"
      }
    },
    "required": ["file", "function", "source"]
  }
}
```

#### Discriminator & Ambiguity Rules

* The engine resolves the target loop within `function` using `loop_on` matching against `ast.ForStmt` or `ast.RangeStmt` headers.
* If multiple loops match the same discriminator, the engine fails before mutation and reports each candidate with its enclosing context and ordinal path, matching the ambiguity contract established in ADR-0016 and ADR-0019.

### 2. Top-Level Declaration Replacement & Collision Handling (`semantic_replace_decl` / `overwrite: true`)

Address constant and variable updates through two complementary mechanisms:

1. **`semantic_replace_decl`**: Targets existing package constants, global variables, or type aliases by identifier:

   ```json
   {
     "name": "semantic_replace_decl",
     "description": "Use this tool instead of replace_file_content whenever updating the definition or value of an existing package-level constant, variable, or type alias.",
     "parameters": {
       "properties": {
         "file": { "type": "string", "description": "Target Go source file" },
         "symbol": { "type": "string", "description": "Existing identifier to replace (e.g. 'benchmarkBrowserShortcode')" },
         "source": { "type": "string", "description": "New declaration source snippet" }
       },
       "required": ["file", "symbol", "source"]
     }
   }
   ```

2. **Duplicate Symbol Guard in `semantic_insert_decl`**: `semantic_insert_decl` must detect when the target identifier is already declared in the file or package. Unless explicitly invoked with `overwrite: true` or routed through `semantic_replace_decl`, insertion must fail before mutation rather than prepending an invalid duplicate declaration.

---

## Invariants

1. **Construct-Bounded AST Scope**: Construct operations mutate only the target AST node and its sub-tree; all surrounding statements and sibling declarations remain untouched.
2. **Pre-Mutation Validation**: Replacements are parsed into synthetic AST stubs in memory and formatted via `go/format` before executing atomic disk writes.
3. **Intent-Centric Trigger Descriptions**: Tool descriptions must lead with explicit affirmative trigger scenarios ("Use this tool instead of...") targeting specific mental representations (e.g. assertion loops, range bounds, constants).
4. **Collision Preemption**: Inserting top-level declarations must never create duplicate-symbol compile errors; collisions must be reported as diagnostic errors or routed to intentional replacement.
5. **Atomic Disk Synchronization**: Mutations execute via `pipeline.WriteAtomic` with advancing timestamps and automated diagnostic delta reporting.

---

## Consequences

* **Positive**: Eliminates token bloat by avoiding whole-body regeneration for localized loop/assertion changes; prevents delimiter boundary errors (`syntax error: expected declaration, found '}'`); closes the top-level constant and variable replacement gap; directly aligns tool selection with the agent's natural intent framing.
* **Negative**: Expands the operation registry surface area; requires AST traversal and discriminator matching for nested loop expressions.
