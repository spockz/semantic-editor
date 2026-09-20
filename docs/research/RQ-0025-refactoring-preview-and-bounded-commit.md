# RQ-0025: Refactoring Preview and Bounded Commit

* **Status**: Resolved
* **Category**: Architecture & Safety
* **Date**: 2026-09-17

---

## Context

Refactoring batches may call language servers, formatters, compilers, and other
CLI tools. The system needs a low-latency direct mode without treating a
mutating request as a multi-step user-approval protocol. We also investigated
whether private copy-on-write workspace execution belongs in semantic-editor.

## Research Questions

1. How can direct execution report partial application honestly without an
   unsafe automatic rollback?
2. Does private workspace execution belong in the semantic-editor capability
   boundary, or with the LLM/controller that invokes it?
3. If this is revisited, which publication and execution-boundary constraints
   must prevent a superficially isolated implementation from mutating or
   overwriting the live workspace incorrectly?

## Resolution

`semantic_batch` remains direct: its mutating request authorizes execution,
and successful operations remain applied even if later compiler, test, lint, or
formatter diagnostics fail. Its result contract distinguishes successful
application with diagnostics from partial semantic-operation failure and must
include the final resulting diff. No automatic Git restore, receipt, or second
commit request is used. The Sol review found that the retained prototype did
not yet produce that final batch-wide diff after formatting; correct this direct
mode gap separately from any future workspace-view work.

Private workspace execution is deferred and does not belong in this component
today. A controller may create a CoW copy, worktree, overlay, or ordinary copy;
invoke semantic-editor in that workspace; and inspect or publish the resulting
diff itself. This supports tools beyond gopls without making semantic-editor a
provider-specific workspace manager. It also avoids pretending that a copied
directory is a security sandbox. There is no separate justification for
isolating multi-file edits while allowing direct single-file edits: both are
authorized direct mutations.

The dedicated Sol review of the retained prototype establishes the minimum
re-entry bar. A future design must prevent Git-aware tools from discovering the
live parent repository through cwd, environment, arguments, response files, or
relative paths; use a positive source-file publication boundary; preserve full
metadata and preimage bytes through a final stale check; make generated patch
filenames grammar-safe and validate their target set; and return a final
batch-wide diff after formatting. It must also correctly preserve empty
directories and directory modes in the workspace view.

## Related Work

* ADR-0010 provides atomic disk-write and timestamp rules.
* ADR-0016 retains direct sequential `semantic_batch` behavior.
* ADR-0029 keeps trusted LSP rename and WorkspaceEdit interpretation backend-owned.
* ADR-0032 records direct execution and the deferred workspace-view boundary.

## Follow-up

* Reopen only for an explicit speculative-workflow requirement: previewing an
  unbounded third-party tool write set, comparing candidate transformations, or
  holding a stable input during concurrent live edits.
* Start from the retained but unmerged `codex/direct-isolated-batch` reference
  commit `de22f2c3951f048e3fe8a97b149e270981c3b983`, then satisfy the dedicated
  Sol review constraints recorded in ADR-0032 before proposing implementation.
