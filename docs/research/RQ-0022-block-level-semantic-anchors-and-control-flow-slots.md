# RQ-0022: Block-Level Semantic Anchors, Control Structure Slots & Location Discovery

* **Status**: Open
* **Category**: Semantics & Architecture
* **Date**: 2026-09-17

---

## 1. Context & Motivation

Current AST mutation tools in `semedit` (`semantic_replace_body`, `semantic_insert_function`, `semantic_insert_type`, `semantic_insert_case`) address top-level declarations and whole-function bodies. However, real-world refactorings and code additions predominantly happen **inside** functions and compound control flow structures:

1. Inserting statements into an `if` block (e.g. error handling, logging, return).
2. Updating or augmenting loop headers (e.g. `for ... in`, `for ... range`, loop initialization/condition/post).
3. Modifying branch conditions (`if condition`, `while condition`, `match guard`).
4. Inserting clauses into compound structures (`select` communication clauses, `try-catch-finally` handlers).

When an LLM attempts to make these modifications using conventional tools, it faces a dilemma:
* **Text Diff Editors**: Must read the file, locate token/line coordinates, and emit brittle unified diffs or character slices.
* **Naive AST Tooling**: Lacks granular targeting for nested blocks, forcing the LLM to replace the entire function body (wasting tokens and risking accidental regressions).

To keep semantic editing competitive, `semedit` requires an expressive anchor and slot model that targets compound statements directly without requiring line/character numbers.

---

## 2. The Slot Classification Architecture

AST nodes cannot be treated as uniform insertion points. Slots in control flow structures are classified by what mutation operation is valid:

| Slot Kind | Examples Across Languages | Valid Semantic Operations |
| :--- | :--- | :--- |
| `statement_list` | Function body, `if` then-body, `else` body, loop body, `catch` body | `insert_statements` (at `start`, `end`, `before:<id>`, `after:<id>`) |
| `single_statement` | Unbraced Java/C/TypeScript `if` or `for` body | `materialize_block`, `replace_statement` |
| `expression` | `if` condition, loop condition, `for..in` iterable, `switch` selector | `replace_expression` |
| `optional_statement` | Go `if` init statement (`if err := ...; err != nil`), `for` init/post | `set_statement`, `remove_statement` |
| `clause_list` | `switch` cases, `select` clauses, Python `match` cases, Java `catch` blocks | `insert_clause`, `remove_clause` |
| `binding` | Go `range` variables (`k, v`), Java enhanced-for (`Type item : items`), Python `for item in items` | `replace_binding` |

### Architectural Invariant
> **Anchors identify semantic containers and roles; placements identify positions within ordered containers.**

* An `if` statement's `then` block is a `statement_list` container (an anchor).
* A condition is an `expression` slot (target of replacement, not statement insertion).
* An absent `else`, `catch`, or `finally` is a clause creation operation, not an insertion into a non-existent block.

---

## 3. Ephemeral Structural Handles & Discovery

To avoid brittle line/character coordinates or complex AST query expressions, `semedit` employs **ephemeral structural handles** discovered via `semantic_supported_locations`.

### Location Discovery Protocol (`semantic_supported_locations`)

```json
{
  "file": "internal/server.go",
  "owner": "Server.Handle",
  "kinds": ["if", "loop", "switch", "try"]
}
```

The server returns discovered containers and slots within the owner function:

```json
{
  "file": "internal/server.go",
  "document_revision": "sha256:91d3...",
  "owner": "Server.Handle",
  "locations": [
    {
      "id": "loc_if_err_then",
      "construct": {
        "kind": "if",
        "ordinal": 1,
        "summary": "err != nil"
      },
      "slot": {
        "role": "then",
        "value_kind": "statement_list"
      },
      "operations": ["insert_statements"],
      "placements": ["start", "end", "before_statement", "after_statement"],
      "statements": [
        {
          "id": "stmt_ret_err",
          "kind": "return",
          "summary": "return err",
          "terminates": true
        }
      ]
    },
    {
      "id": "loc_if_err_cond",
      "construct": {
        "kind": "if",
        "ordinal": 1
      },
      "slot": {
        "role": "condition",
        "value_kind": "expression"
      },
      "operations": ["replace_expression"]
    }
  ]
}
```

### Granular Insertion Without Coordinates

The agent performs targeted insertion referencing the discovered handle:

```json
{
  "file": "internal/server.go",
  "target_location": "loc_if_err_then",
  "document_revision": "sha256:91d3...",
  "placement": {
    "kind": "before",
    "statement_id": "stmt_ret_err"
  },
  "statement": "log.Printf(\"execution failed: %v\", err)"
}
```

---

## 4. The Dual-Tier Interaction Model: Intent Roles vs Precision Anchors

To prevent turning the LLM into a manual AST compiler (forcing a 2-turn read-analyze-insert loop for common edits), `semedit` provides two complementary interaction tiers:

### Tier A: Fast-Path Zero-Turn Declarative Intent (`semantic_insert_decl`)
When the model intends to declare a standard language entity, it does not need to know where that entity physically lives in the file. It specifies the **semantic role**:

```json
{
  "name": "semantic_insert_decl",
  "inputSchema": {
    "file": "internal/user/user.go",
    "role": "sentinel_error",
    "source": "var ErrNotFound = errors.New(\"user not found\")",
    "auto_organize_imports": true
  }
}
```

The engine automatically enforces language placement invariants without any pre-flight inspection turns:
- `role: "sentinel_error"`:
  * In Go: Placed in the package-level `var (...)` error block directly below imports and above types. If existing `Err*` definitions exist, appends to the existing `var` block.
  * In Java: Created as `public static final class UserNotFoundException extends RuntimeException { ... }` or grouped exception static inner class.
- `role: "constant"`: Placed in the package/class constant section.
- `role: "type"` / `role: "interface"`: Partitioned by public/private visibility (ADR-0012).
- `role: "constructor"`: Grouped directly adjacent to the instantiated type definition.
- `role: "init_registration"`: Appended to the `init()` block (or creates `func init()` if absent).

### Tier B: Precision Anchoring Inside Function Bodies (`semantic_insert_statement`)
When surgical statement insertion inside an existing function or control structure is required, the model specifies an anchor pattern directly:

```json
{
  "name": "semantic_insert_statement",
  "inputSchema": {
    "file": "internal/server.go",
    "func": "Serve",
    "anchor": "before_return",
    "statement": "s.logger.Info(\"server shutdown complete\")"
  }
}
```
Or relative to existing code:
```json
{
  "name": "semantic_insert_statement",
  "inputSchema": {
    "file": "internal/server.go",
    "func": "Serve",
    "anchor": "after:s.init()",
    "statement": "s.metrics.RecordStart()"
  }
}
```

If disambiguation is needed in deeply nested control flow, the discovery tool `semantic_supported_locations` serves as a surgical fallback.

---

## 5. Cross-Language Normalization & Idioms

| Language | Specific Control Structures | Structural Nuances |
| :--- | :--- | :--- |
| **Go** | `if [init;] cond`, `for [init; cond; post]`, `for range`, `select`, `switch` | Implicit block scopes on `if` and `switch`; clauses have implicit blocks. |
| **Java** | `if`, `for (init; cond; update)`, enhanced-for (`for (T item : coll)`), `while`, `do`, `try-with-resources`, `catch`, `finally` | Single-statement bodies allowed; require `materialize_block: true` before inserting multiple statements. |
| **Python** | `if/elif/else`, `for item in iter: ... else:`, `while ... else:`, `try/except/else/finally`, `match/case` | Suites are statement lists; branch headers are expressions/guards; explicit `else` on loops. |
| **TypeScript** | `if/else`, classic `for`, `for..in`, `for..of`, `while`, `do..while`, `try/catch/finally` | Distinct `for..in` (keys) vs `for..of` (values); single-statement bodies allowed. |

---

## 6. Next Steps

1. **Codify Architecture in ADR-0019**: Document Intent-Driven Declarative Roles and Precision Block Anchors.
2. **Implement `semantic_insert_decl` Role Engine**: Support `sentinel_error`, `constant`, `type`, `constructor`, and `init_registration`.
3. **Implement `semantic_insert_statement`**: Support `start`, `end`, `before_return`, and `before/after:<pattern>`.

