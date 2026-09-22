# ADR-0004: Staged Execution & Diagnostic Reporting (No Forced Rollback)

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Initial designs proposed strictly atomic transactions with automatic rollback whenever an edit resulted in compiler errors.

However, real-world refactoring is frequently a multi-step process that inherently requires **intermediate breaking states** (e.g. altering an interface method signature breaks all implementers until follow-up edits land). Unilaterally rolling back the working directory on the first step prevents legitimate multi-step workflows.

## Decision

1. **Staged Execution with Auto-Formatting**: `semedit` applies transformations to disk and immediately runs the language's canonical formatter (`gofmt`, `rustfmt`, `prettier`, etc.).
2. **Diagnostic Delta Reporting**: A standalone edit captures diagnostics both
   immediately before and immediately after mutation. The pre-edit baseline
   distinguishes pre-existing failures from diagnostics introduced or resolved
   by that edit; the post-edit capture supplies the current actionable state.
   Rather than blocking or reverting, `semedit` returns that structured
   diagnostic feedback ($\Delta \text{errors}$) directly to the LLM. A cached
   baseline is not valid unless it is bound to a workspace revision and
   invalidated by every external writer, so direct standalone operations do not
   reuse one.
3. **On-Demand Snapshots / Undo**: `semedit` maintains lightweight pre-edit snapshots so the user or agent can explicitly revert if desired, but **never enforces automatic rollback**.

## Invariants

* The engine must never automatically revert code without explicit instruction from the agent or user.
* Every edit must return actionable compiler/linter diagnostics.
* A standalone diagnostic delta compares the workspace state immediately before
  and after its edit. Batches may share one pre-edit baseline and one final
  post-processing capture across their ordered operations as specified by
  ADR-0032.

## Consequences

* **Positive**: Supports natural multi-step refactoring workflows while providing immediate feedback on whether compiler errors are increasing or decreasing.
* **Negative**: Codebase can remain in a broken state if an agent abandons a multi-step refactoring midway. Standalone edits also pay for two package-level diagnostic captures; `semantic_batch` amortizes those captures across a coordinated edit set.
