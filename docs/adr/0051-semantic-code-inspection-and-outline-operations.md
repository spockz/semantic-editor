# ADR-0051: Semantic Code Inspection and Outline Operations

Status: Accepted
Date: 2026-09-26

## Context

Under read-restriction policies (such as `--semedit-arm-restrict=read|readwrite`), benchmark agents are steered toward semantic operations and prohibited from raw shell inspection. Existing semantic lookup provided point coordinates (`line`, `column`, `offset`), lacking declaration bodies, signatures, enclosing scope, and file context. Agents attempting semantic refactorings (such as replacing method bodies) refused to modify unseen code.

Additionally, agents lacked capabilities to discover declarations across unfamiliar files or directory hierarchies before knowing exact symbol identifiers, had no typed reference discovery mechanism, and experienced member lookup failures on Go interfaces (`Sink.Write`) and receiver filter parameter leakage.

## Decision

We introduce three dedicated, zero-token read operations across the core engine, CLI, and MCP servers:

1. **Symbol Inspection (`semantic_inspect_symbol` / `semedit inspect`)**:
   Returns complete source declarations, signatures, doc comments, enclosing package/imports, and content digests for queried symbols. Ambiguous queries return all matching candidates across the workspace in one structured response.

2. **Hierarchical Outline (`semantic_outline` / `semedit outline`)**:
   Projects declaration hierarchies for a selected file or directory. Function and method bodies are elided to conserve context budget. Supports kind filtering (`kinds`), unexported symbol inclusion (`include_unexported`), and test symbol inclusion (`include_tests`). Preserves package documentation comments on empty extension points.

3. **Symbol References (`semantic_find_references` / `semedit references`)**:
   Separates declaration definitions from usages. Resolves call sites, parameter bindings, type usages, and field accesses across workspace source files.

We extend cross-language parity for these read capabilities:

- Generic Language Server Protocol (LSP) document symbol handling (`readlsp`) powers Java, Kotlin, Rust, and Scala outline projections.
- Dedicated adapters provide outline projections for Makefile and Bash scripts.
- The Go backend provides AST-native inspection, outline, and typed reference resolution.

## Invariants

- Read operations preserve workspace immutability: disk state, git worktrees, and file modification timestamps remain untouched.
- Ambiguous symbol queries return complete candidate sets with distinct locations rather than aborting or arbitrarily selecting a single match.
- Hierarchical outlines omit executable statement bodies while retaining canonical signatures, doc comments, and parent-child declaration relationships.
- Language backends declare read capabilities explicitly in the backend registry; unsupported combinations reject execution with structured errors.
- Every public language-operation combination maintains executable CLI txtar contract test coverage.

## Consequences

- Agents operating under read restrictions inspect and discover code structure through semantic tools without falling back to unconstrained shell commands.
- Context window consumption remains compact due to body-elided structural outlines.
- Language backends maintain consistent read and inspection contracts across Go, Java, Kotlin, Rust, Scala, Make, and Bash.
