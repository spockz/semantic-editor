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

## 4. Cross-Language Normalization & Idioms

| Language | Specific Control Structures | Structural Nuances |
| :--- | :--- | :--- |
| **Go** | `if [init;] cond`, `for [init; cond; post]`, `for range`, `select`, `switch` | Implicit block scopes on `if` and `switch`; clauses have implicit blocks. |
| **Java** | `if`, `for (init; cond; update)`, enhanced-for (`for (T item : coll)`), `while`, `do`, `try-with-resources`, `catch`, `finally` | Single-statement bodies allowed; require `materialize_block: true` before inserting multiple statements. |
| **Python** | `if/elif/else`, `for item in iter: ... else:`, `while ... else:`, `try/except/else/finally`, `match/case` | Suites are statement lists; branch headers are expressions/guards; explicit `else` on loops. |
| **TypeScript** | `if/else`, classic `for`, `for..in`, `for..of`, `while`, `do..while`, `try/catch/finally` | Distinct `for..in` (keys) vs `for..of` (values); single-statement bodies allowed. |

---

## 5. Next Steps

1. **Codify Architecture**: Incorporate slot taxonomy and discovery contracts into upcoming ADR.
2. **Phase 1 Implementation**: Implement `semantic_supported_locations` for Go in `internal/astedit/locations.go`.
3. **Phase 2 Implementation**: Implement `semantic_insert_statement` and `semantic_replace_expression` targeting discovered location handles.
