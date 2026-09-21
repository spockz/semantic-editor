# RQ-0028: Semantic Quality Checks and Normalization

* **Status**: Open
* **Date**: 2026-09-21
* **Category**: Operations & Quality

## 1. Question

Should `semedit` expose language-aware `check` and `fix` operations through
the central operation registry, and should successful semantic edits advance a
workspace toward a documented normalized form by running formatting, import
organization, and conservative lint fixes?

## 2. Context

The repository's `make check` quality gate already separates automatic repair
from diagnosis: formatting, `go fix`, import organization, and
`golangci-lint --fix` can rewrite source, while remaining linter findings are
reported for an author to decide. The operation registry now makes a supported
operation available consistently through the CLI, MCP, batch dispatch, and
generated documentation (ADR-0034).

Today, individual semantic insertions organize imports and Go edits are
formatted as needed, but callers cannot ask `semedit` to assess or normalize a
whole workspace through the same operation surface. They must instead know and
invoke repository-specific build targets. That breaks the intended division of
labor: an agent plans semantic intent, while the host makes deterministic
source transformations.

## 3. Proposed Model

Investigate two explicitly separate operations:

| Operation | Effect | Intended result |
| :--- | :--- | :--- |
| `check` | Read-only | Structured diagnostics describing parse, type, formatting, import, and lint deviations. |
| `fix` | Mutating | Applies only deterministic, tool-supported repairs and returns an edit receipt plus post-fix diagnostics. |

`fix` is not permission to silence every warning. It may run only repairs that
the selected language tool declares as automatic and semantics-preserving for
its supported version. A finding without an offered fix remains a diagnostic.
For Go, the initial candidate normalization sequence is:

```text
format -> organize imports -> go fix -> golangci-lint --fix -> format -> organize imports
```

The research must determine which steps are appropriate for a semantic
operation, their order, and whether dependency or workspace mutations such as
`go mod tidy` belong outside this operation because their scope is broader than
source normalization.

## 4. Invariants to Preserve

1. `check` never changes files, generated artifacts, module manifests, or
   workspace configuration.
2. `fix` reports every changed file and every tool/version used, using the
   same receipt model as semantic edits.
3. Only explicit auto-fixes run. Advisory diagnostics, heuristic refactors,
   and behavior-affecting migration suggestions require a separate semantic
   operation or caller approval.
4. The operation is language-qualified. A project with multiple backends
   receives only the normalizers declared by each backend's capabilities.
5. Normalization is idempotent: a second successful `fix` produces no source
   changes and equivalent diagnostics.
6. Tool failures, unavailable tools, and partial normalization are represented
   explicitly; they must not look like a clean workspace.
7. Batch behavior preserves atomic-disk-update requirements and makes the
   normalization point observable, rather than running hidden whole-workspace
   rewrites after every constituent operation.

## 5. Design Alternatives

### A. Explicit Workspace Operations Only

Register `check` and `fix` as standalone workspace operations. Callers choose
when to normalize, so a narrow semantic insertion remains narrow and batch
work has one obvious normalization point.

### B. Implicit Normalization After Every Mutation

Every mutating operation runs the relevant normalizers before returning. This
gives strong local consistency but can cause surprising unrelated edits,
repeated tool startup, and ambiguous failure semantics when the edit succeeds
but normalization does not.

### C. Default Local Normalization Plus Explicit Whole-Workspace Operations

Keep existing per-file formatting and import organization where required to
produce valid output. Add explicit workspace `check` and `fix` operations for
cross-file quality tools. This is the current leading hypothesis because it
preserves predictable edit scope while offering an agent-native path to a
normalized workspace.

## 6. Evaluation Plan

1. Define a language-neutral result schema for diagnostics, changed files,
   skipped tools, tool versions, and partial failures.
2. Prototype Go `check` and `fix` defs in `internal/operation`, deriving CLI,
   MCP, batch, and docgen exposure from the registry.
3. Add CLI txtar contracts for clean workspaces, format/import repairs,
   linter-supported repairs, unfixed advisory diagnostics, unavailable tools,
   and idempotence.
4. Measure affected-file count, elapsed time, tool startup cost, and whether a
   post-edit fix creates unrelated diffs in representative repositories.
5. Verify that an edit receipt distinguishes the primary semantic change from
   each normalization rewrite, so agents can review and explain the result.
6. Repeat the prototype with a second language backend before accepting a
   language-neutral contract.

## 7. Open Questions

1. Is the correct unit a file, package, module, workspace, or an explicit
   caller-supplied scope?
2. Which normalizers are sufficiently deterministic and semantics-preserving
   to run without an additional approval boundary?
3. Should `check` expose raw tool diagnostics, a normalized common schema, or
   both?
4. How should `fix` behave when one normalizer succeeds and a later one fails:
   preserve the successful edits with a partial receipt, or preflight every
   tool and decline to start?
5. Should `fix` run automatically at the end of a semantic batch, be an
   explicit batch entry, or support both modes with distinct receipts?
6. Where should repository policy live: backend capability metadata, project
   configuration, or caller arguments?

## 8. Next Steps

1. Catalogue the Go quality tools already invoked by `make check`, classifying
   each as read-only, safe auto-fix, potentially broad mutation, or external
   dependency/workspace mutation.
2. Specify the proposed operation contracts and scope parameter without
   implementing them.
3. Run a small Go prototype against the public txtar fixture suite and record
   idempotence, error, and receipt behavior.
4. Create an ADR only after the scope, atomicity, and partial-failure policy
   have empirical support.
