# ADR-0004: Staged Execution & Diagnostic Reporting (No Forced Rollback)

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Initial designs proposed strictly atomic transactions with automatic rollback whenever an edit resulted in compiler errors.

However, real-world refactoring is frequently a multi-step process that inherently requires **intermediate breaking states** (e.g. altering an interface method signature breaks all implementers until follow-up edits land). Unilaterally rolling back the working directory on the first step prevents legitimate multi-step workflows.

## Decision

1. **Staged Execution with Auto-Formatting**: `semedit` applies transformations to disk and immediately runs the language's canonical formatter (`gofmt`, `rustfmt`, `prettier`, etc.).
2. **Diagnostic Delta Reporting**: The compiler/typechecker runs immediately after the edit. Rather than blocking or reverting, `semedit` returns structured diagnostic feedback ($\Delta \text{errors}$) directly to the LLM.
3. **On-Demand Snapshots / Undo**: `semedit` maintains lightweight pre-edit snapshots so the user or agent can explicitly revert if desired, but **never enforces automatic rollback**.

## Invariants

* The engine must never automatically revert code without explicit instruction from the agent or user.
* Every edit must return actionable compiler/linter diagnostics.

## Consequences

* **Positive**: Supports natural multi-step refactoring workflows while providing immediate feedback on whether compiler errors are increasing or decreasing.
* **Negative**: Codebase can remain in a broken state if an agent abandons a multi-step refactoring midway.
