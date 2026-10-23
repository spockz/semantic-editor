# ADR-0004: Staged Execution & Diagnostic Reporting (No Forced Rollback)

* **Status**: Accepted
* **Date**: 2026-09-15

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Backend-neutral deltas require current, coverage-valid findings. Exact full Go captures may seed incremental baselines; metadata errors require full coverage and server-state recovery. Explicit verification stays full; intermediate edits remain applied. Refactorings protect their observed and touched files without locking the workspace against unrelated concurrent edits. |
| description | Initial designs proposed strictly atomic transactions with automatic rollback whenever an edit resulted in compiler errors. |
| role | invariant |
| scope | project |
| area | backend |
| languages | generic |

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
   validated against current source contents, membership, configuration, and
   auxiliary inputs. A retained Go session may reuse an exact baseline and
   refresh changed packages and their transitive reverse dependencies, merging
   unaffected cached findings. Unknown coverage or structural invalidation
   requires a full capture. Explicit verification requests remain full scans.
   A successful default full Go verification may seed the next incremental
   baseline when it exactly covers the retained scope and current package
   findings. Explicit-pattern and buffered captures do not seed a disk baseline.
   The contract requires a valid pre-edit baseline and updated post-edit
   findings for affected packages and their transitive reverse dependencies;
   it does not require two full workspace sweeps. Duplicate background gopls
   diagnosis is also independent of delta semantics. Retained Go sessions use
   `diagnosticsTrigger: "Save"` to suppress background diagnosis on ordinary
   `didChange`, reuse revision-validated findings, and explicitly pull the
   affected package closure (ADR-0060).
   Import-list changes may refresh package discovery without discarding valid
   findings for unrelated packages. Cache retention requires an unchanged
   non-import analysis boundary and identical source/module membership; current
   content digests still select every changed package and reverse importer.
   Metadata-error transitions require complete coverage; Go may also reload its
   retained workspace view to clear stale server-side findings without replacing
   the process (ADR-0060).
3. **On-Demand Snapshots / Undo**: `semedit` maintains lightweight pre-edit snapshots so the user or agent can explicitly revert if desired, but **never enforces automatic rollback**.
4. **Workspace Concurrency Boundary**: Refactoring operations do not lock the
   workspace or require a complete inventory of concurrent edits and future
   workspace membership. Each operation follows its own observed-file and
   touched-file checks. An unrelated concurrent edit or newly added file/caller
   outside that set may remain unchanged and may be missed by the refactoring;
   this is an accepted outcome, not a reason to add workspace-wide locking or
   transaction machinery. Callers that require a coordinated whole-workspace
   migration must provide that coordination externally.

## Invariants

* Diagnostic deltas and freshness are backend-neutral contracts. The Go
  implementation of baseline reuse and reverse-dependency refresh in ADR-0060
  is not a shared LSP cache or a requirement to use pull diagnostics. Other
  backends must establish their own coverage and completion receipts before
  reusing findings; receiving a successful edit response alone is insufficient.

* The engine must never automatically revert code without explicit instruction from the agent or user.
* Every edit must return actionable compiler/linter diagnostics with their
  original severity. Warnings, information, and hints remain visible in the
  full diagnostic delta. Only newly introduced error-severity diagnostics
  trigger an applied-rename compilation failure; advisory findings do not.
  Configured external lint checks enforce additional project policy through
  their own reported outcomes (ADR-0054).
* A standalone diagnostic delta compares the workspace state immediately before
  and after its edit. Batches may share one pre-edit baseline and one final
  post-processing capture across their ordered operations as specified by
  ADR-0032.
* Concurrent edits outside an operation's observed and touched files need not
  be detected or incorporated. Correctness checks remain scoped to the files
  and evidence the operation actually uses; they do not promise a serializable
  transaction over the workspace.

## Consequences

* **Positive**: Supports natural multi-step refactoring workflows while providing immediate feedback on whether compiler errors are increasing or decreasing, without imposing workspace-wide concurrency coordination.
* **Negative**: Codebase can remain in a broken state if an agent abandons a multi-step refactoring midway. A missing or invalid baseline still requires analysis before mutation. Revision validation and conservative fallback are required to keep incremental deltas equivalent to full captures.
* **Tradeoff**: Concurrent changes outside the operation's file/evidence set may be missed and left behind. This keeps edits scoped and avoids locking unrelated work; complete coordinated migrations need external synchronization.
* **Cost boundary**: Two full sweeps and concurrent push/pull analysis are
  implementation costs, not diagnostic-delta requirements. Reuse and scoped
  refresh must preserve coverage and severity; suppressing ordinary background
  diagnosis must not suppress required publication or invalidation events.
