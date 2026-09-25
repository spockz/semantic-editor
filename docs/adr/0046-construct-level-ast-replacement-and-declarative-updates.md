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

### 1. Construct-Level Replacement (`semantic_replace_construct`)

To avoid tool catalog explosion while providing construct-level precision, we unify inner-function control flow mutations under a single construct replacement tool parameterized by a `kind` discriminator:

```json
{
  "name": "semantic_replace_construct",
  "description": "Use this tool instead of replace_body or replace_file_content whenever modifying an existing control-flow construct (for loop, if condition/branch, switch case, select branch) inside a function or method. Operates directly on the targeted construct without regenerating the surrounding body.",
  "parameters": {
    "properties": {
      "file": {
        "description": "Path to the source file",
        "type": "string"
      },
      "function": {
        "description": "Name of the containing function or method (e.g. 'TestWriteAssets' or '(*Server).Serve')",
        "type": "string"
      },
      "kind": {
        "description": "Structural construct kind to replace. Enums are dynamically constrained by the active language backend (e.g. ['loop', 'if', 'case', 'select'] for Go).",
        "type": "string"
      },
      "discriminator": {
        "description": "Expression, condition, or selector identifying the construct (e.g. 'want', 'err != nil', or 'case \"stop\":')",
        "type": "string"
      },
      "construct_path": {
        "description": "Optional ordinal path reported when multiple constructs match (e.g. '0', '0.1')",
        "type": "string"
      },
      "source": {
        "description": "Complete replacement code for the targeted construct",
        "type": "string"
      },
      "auto_organize_imports": {
        "default": true,
        "description": "Automatically clean up and resolve imports after mutation",
        "type": "boolean"
      }
    },
    "required": ["file", "function", "kind", "source"]
  }
}
```

#### Language Construct Taxonomy & Grammar Neutrality

The term `construct` neutralizes cross-language grammar differences (e.g. whether `if...else` or `switch/match` is classified as a statement or an expression) while providing precision targeting for nested blocks:

| Language | Supported `kind` Values | Primary AST Construct Mappings |
| :--- | :--- | :--- |
| **Go** | `["loop", "if", "else", "case", "select", "defer"]` | `*ast.ForStmt`, `*ast.RangeStmt`, `*ast.IfStmt`, `*ast.CaseClause`, `*ast.SelectStmt` / `*ast.CommClause`, `*ast.DeferStmt` |
| **Rust** | `["loop", "if", "else", "match"]` | `ExprForLoop`, `ExprWhile`, `ExprLoop`, `ExprIf`, `ExprMatch` / `Arm` patterns |
| **Java** | `["loop", "if", "else", "case", "try_catch", "try_with_resources", "synchronized"]` | `ForStatement`, `EnhancedForStatement`, `WhileStatement`, `IfStatement`, `SwitchCase`, `TryStatement`, `SynchronizedStatement` |
| **Kotlin** | `["loop", "if", "else", "when", "case", "try_catch"]` | `KtForExpression`, `KtWhileExpression`, `KtDoWhileExpression`, `KtIfExpression`, `KtWhenExpression`, `KtWhenEntry`, `KtTryExpression` |
| **Scala** | `["loop", "if", "else", "match", "for_comprehension", "try_catch"]` | Imperative loops, `if` expressions, pattern matching, monadic `for ... yield` comprehensions, `try ... catch` |
| **C#** | `["loop", "if", "else", "case", "try_catch", "lock", "using", "query", "yield"]` | Iteration statements, selection statements, switch arms, `try ... catch`, `lock`, `using`, LINQ `from ... select`, `yield return/break` |
| **Python** | `["loop", "if", "else", "case", "try_except", "with", "yield"]` | `ast.For`, `ast.While`, `ast.If`, `ast.MatchCase`, `ast.Try`, `ast.With`, `ast.Yield` / `ast.YieldFrom` |

##### 1. Conditionals & Branch Modeling (`if`, `else`, `case`, `when`, `match`)

In compiler ASTs across languages (Go `*ast.IfStmt`, Rust `ExprIf`, Java `JCIf`, C# `IfStatementSyntax`, Kotlin `KtIfExpression`), there is no standalone AST node for `else if`. Instead, `else if` is an `if` node nested within the `else` slot of a parent `if`.

To support deterministic mutation without synthetic grammar types, `semantic_replace_construct` distinguishes three mutation targets:

1. **Root Conditional Replacement (`kind: "if"`)**: Supplying the condition of the root `if` (e.g. `discriminator: "err != nil"`) replaces the entire conditional chain and all attached branches.
2. **Branch Replacement (`kind: "if"`)**: Supplying the condition of an `else if` branch (e.g. `discriminator: "count > 0"`) replaces only that nested `if` node; the parent condition and sibling branches remain intact.
3. **Terminal Fallback Replacement (`kind: "else"`)**: Terminal `else` blocks lack a boolean condition. Targeting `kind: "else"` selects the unconditioned fallback block (e.g. `*ast.BlockStmt`), allowing direct fallback mutation without regenerating the preceding conditional tree.
4. **Pattern & Switch Matching (`case`, `when`, `match`)**:
   * `case`: Targets switch clauses (`*ast.CaseClause`, `SwitchCase`).
   * `when`: Kotlin unifies switches, condition cascades, and type pattern matching under `when` (`KtWhenExpression`), where `case` addresses individual branch entries (`KtWhenEntry`).
   * `match`: Rust (`ExprMatch`) and Scala pattern matching arms.

##### 2. Iteration & Comprehensions (`loop`, `for_comprehension`, `query`)

* `loop`: Imperative iteration constructs (`for`, `while`, `do-while`, Rust `loop`).
* `for_comprehension`: Monadic `for ... yield` in Scala, representing sequence transformations distinct from imperative loops.
* `query`: Declarative LINQ comprehensions (`from ... where ... select`) in C#.

##### 3. Coroutine & Generator Suspension (`yield`)

* In Python, JavaScript, and C# (`yield return`, `yield break`), `yield` is a primitive suspension construct. Mutating `kind: "yield"` rewrites generator emissions or yield points without regenerating the surrounding iteration.
* **The Kotlin Distinction**: In Kotlin, `yield` is not a language keyword or statement; it is a suspending standard library member function on `SequenceScope<T>` (`sequence { yield(x) }`). In AST terms, it is a function call (`KtCallExpression`) within a coroutine builder lambda rather than a primitive AST control construct.

##### 4. Resource & Lifecycle Management (`defer`, `with`, `using`, `try_with_resources`)

* `defer`: Go (`*ast.DeferStmt`), Swift, and Zig function-exit execution blocks.
* `with`: Python context manager scopes (`ast.With`).
* `using`: C# deterministic disposal blocks (`UsingStatementSyntax`).
* `try_with_resources`: Java deterministic resource acquisition and disposal (`TryStatement` declaring resources implementing `AutoCloseable`). Distinguishing `try_with_resources` from general `try_catch` enables mutating resource bindings (`try (BufferedReader br = ...)`) and their managed block without disturbing trailing catch/finally clauses.

##### 5. Exception & Error Handling (`try_catch`, `try_except`)

* `try_catch`: Java, C#, JavaScript, and Scala (`try ... catch ... finally`). In Kotlin, `try` is an expression producing a value (`KtTryExpression`).
* `try_except`: Python structured exception handling (`ast.Try`).

##### 6. Concurrency & Synchronization (`select`, `lock`, `synchronized`)

* `select`: Go channel concurrency multiplexer (`*ast.SelectStmt` / `*ast.CommClause`).
* `synchronized`: Java monitor synchronization blocks (`SynchronizedStatement`).
* `lock`: C# mutual exclusion blocks (`LockStatementSyntax`).

#### Declarative Registry Announcement (Zero Special Registry Interfaces)

To avoid introducing one-off registry interfaces (e.g. `ConstructReplacingBackend`), construct capabilities follow the declarative metadata pattern established in ADR-0012 for access modifiers:

1. **Backend Capability Slice**: In `internal/backend`, `Backend` announces supported constructs via `SupportedConstructs() []ConstructKind`.
2. **Dynamic Parameter Contract**: `operation.ParameterContract` supports `DynamicEnums: func(backend.Backend) []string`.
3. **Dynamic MCP Schema Filtering at Initialization**: During the `initialize` handshake and subsequent `tools/list` requests, `internal/mcp/server.go` resolves `DynamicEnums` against the active workspace backend established during initialization. The server dynamically emits only the constructs valid for that language (e.g. emitting `["loop", "if", "else", "case", "select", "defer"]` for Go, or `["loop", "if", "else", "when", "case", "try_catch"]` for Kotlin). This prevents catalog dilution and allows LLM decoding engines to enforce valid tokens via logit masking from the initial tool exposure. If the active language changes, `notifications/tools/list_changed` pushes the updated schema to the client.

#### Discriminator & Ambiguity Rules

* The engine traverses `function` searching for nodes matching `kind`.
* The target node is selected using `discriminator` matching against headers (loop conditions, range expressions, if predicates, case values).
* If multiple constructs match, the engine fails before mutation and reports each candidate with its line number, context, and ordinal path (`construct_path`), matching the ambiguity contract in ADR-0016 and ADR-0019.

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
