# ADR-0019: Intent-Driven Declaration Roles and Precision Statement Anchors

* **Status**: Accepted
* **Date**: 2026-09-17

---

## Context

AI coding agents refactoring code currently face an undesirable trade-off between brittle text editing and cognitive overload:

1. **Text Diffs (`replace_file_content`)**: The model must read lines, calculate token offsets, and emit fragile string replacements.
2. **Low-Level AST Probing (`supported_locations` -> `insert_statement`)**: Forcing the LLM to inspect AST nodes, parse container IDs, and pick an insertion slot turns the agent into a manual AST compiler, burning tokens and requiring multi-turn roundtrips for standard declarations.

In practice, an agent's semantic intent is usually high-level: *"Add a sentinel error `ErrNotFound`"*, *"Add a package-level mutex `var mu sync.Mutex`"*, or *"Add a shutdown log statement before returning from `Serve`"*.

The host engine already possesses canonical knowledge of language grammar and file layout conventions. Requiring the model to specify spatial coordinates for standard entities violates Intent-Driven Orchestration ([ADR-0001](0001-intent-driven-orchestration.md)).

---

## Decision

We establish a two-tier insertion architecture pairing zero-turn declarative roles with precision block anchors:

### 1. Tier 1: Fast-Path Zero-Turn Declarative Roles (`semantic_insert_decl`)

The agent emits its declaration intent in a **single tool turn** by specifying a semantic `role`:

```json
{
  "name": "semantic_insert_decl",
  "description": "Insert a top-level Go declaration using its semantic role. The host engine deterministically resolves idiomatic placement and formatting without requiring spatial coordinates.",
  "inputSchema": {
    "file": "string (required) — relative path to source file",
    "role": "string (required) — semantic role: 'sentinel_error', 'constant', 'global_variable', 'type', 'constructor', 'init_registration'",
    "source": "string (required) — declaration source code",
    "access_modifier": "string (optional, default 'infer') — 'public', 'private', or 'infer'",
    "auto_organize_imports": "bool (optional, default false)"
  }
}
```

#### Placement Rules by Semantic Role

* **`sentinel_error`**:
  * In Go: Targets the package-level `var (...)` error block. If an existing `var` block containing `Err*` definitions exists, appends inside that block. If absent, creates a new `var` declaration in the sentinel errors section directly below imports and above types.
  * In Java: Synthesizes a nested or package-private exception class.
* **`constant`**:
  * Targets the package `const (...)` block preceding variable and type definitions.
* **`global_variable`**:
  * Targets package-level `var` block following constants.
* **`type`**:
  * Placed according to visibility invariants ([ADR-0012](0012-access-modifiers-and-section-placement.md)): public types in public sections, private types in private sections.
* **`constructor`**:
  * Placed directly adjacent to the instantiated type definition.
* **`init_registration`**:
  * Appended to the `init()` block (or creates `func init()` if absent).

### 2. Tier 2: Precision Statement Anchors Inside Function Bodies (`semantic_insert_statement`)

When surgical insertion inside an existing function body or control flow block is required, the agent specifies the target function and anchor pattern directly:

```json
{
  "name": "semantic_insert_statement",
  "description": "Insert statements into a function, method, or control structure at a semantic anchor point.",
  "inputSchema": {
    "file": "string (required) — relative path to source file",
    "func": "string (required) — target function or method name (e.g. 'Server.Serve')",
    "anchor": "string (optional, default 'end') — 'start', 'end', 'before_return', 'after:<snippet>', 'before:<snippet>'",
    "statement": "string (required) — statement source code",
    "auto_organize_imports": "bool (optional, default false)"
  }
}
```

#### Anchor Semantics

* **`start`**: First statement inside the target function/block.
* **`end`**: Last statement inside the target function/block (automatically places before terminal return if present to avoid unreachable code).
* **`before_return`**: Immediately preceding the terminal `return` statement in the block.
* **`after:<snippet>` / `before:<snippet>`**: Matches statement whose AST contains `<snippet>`.

### 3. Inspection Tool (`semantic_supported_locations`)

Maintained as a diagnostic inspection tool for complex, deeply nested control structures (or multi-language discovery) when direct pattern matching is ambiguous.

---

## Invariants

1. **Zero-Turn Execution**: Declaring entities via standard semantic roles must never require a preceding inspection or discovery call.
2. **Canonical Section Layout**: The engine must enforce language-idiomatic file organization (imports $\to$ constants $\to$ sentinel errors $\to$ types $\to$ constructors $\to$ methods $\to$ private helpers).
3. **Dead Code Prevention**: Inserting with `anchor: "end"` must not place statements after a terminal `return` or `panic`.
4. **Atomic Disk Mutation**: All writes execute through `pipeline.WriteAtomic` with advancing timestamps ([ADR-0010](0010-disk-synchronization-and-cache-invalidation.md)).

---

## Consequences

* **Positive**: Minimizes agent token expenditure; eliminates spatial coordinate reasoning; preserves idiomatic code layout automatically; provides a seamless single-turn workflow for common additions.
* **Negative**: Complex non-standard file layouts may require explicit anchor patterns rather than generic semantic roles.
