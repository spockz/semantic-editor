# ADR-0002: Dual-Engine Architecture (LSP + Error-Tolerant CST)

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Code editing engines traditionally fall into two extremes:

1. **Compiler/LSP Engines** (`gopls`, `rust-analyzer`): Possess complete type attribution, cross-package call graphs, and scope awareness, but fail or drop type tables when code contains syntax errors.
2. **CST / Pattern Engines** (`ast-grep`, `Tree-sitter`, `Comby`): Resilient to incomplete or broken syntax, but type-blind across package boundaries.

If an agent relies only on LSPs, it gets permanently deadlocked when syntax is broken. If it relies only on CSTs, it cannot perform safe cross-file refactorings.

## Decision

`semedit` implements a **Dual-Engine Hierarchical Dispatcher**:

* **Primary Route (Healthy Codebase)**: Routes to Tier 1 Compiler/LSP engines for type-checked cross-package renames, function extraction, and interface stubbing.
* **Fallback Route (Broken Codebase)**: If syntax errors prevent the compiler from building an AST, `semedit` drops down to Tier 3 error-tolerant CST tools (`ast-grep` / Tree-sitter) for structural repairs before handing control back to the compiler.

## Invariants

* `semedit` is an orchestration broker, not a new parser or rewrite engine; it delegates to existing LSPs and tools.
* A broken syntax state must never completely lock the agent out of tool-assisted editing.

## Consequences

* **Positive**: Bridges the gap between type-safety and syntax-error recovery without manual intervention.
* **Negative**: Requires maintaining two execution paths (LSP RPC + Tree-sitter CST).
