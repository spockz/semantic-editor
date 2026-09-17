# RQ-0025: Refactoring Preview and Bounded Commit

* **Status**: Resolved
* **Category**: Architecture & Safety
* **Date**: 2026-09-17

---

## Context

Refactoring batches may call language servers, formatters, compilers, and other
CLI tools. The system needs both a low-latency direct mode and an optional
private workspace mode without treating either mode as a user-approval
protocol.

## Research Questions

1. How can direct execution report partial application honestly without an
   unsafe automatic rollback?
2. How can arbitrary CLI tools receive a private filesystem view without
   copying source bytes for every batch?
3. How can an isolated multi-file result detect stale targets and publish a
   generated patch without a generic client patch API?

## Resolution

`semantic_batch` remains direct: its mutating request authorizes execution,
and successful operations remain applied even if later compiler, test, lint, or
formatter diagnostics fail. Results distinguish successful application with
diagnostics from partial semantic-operation failure and include the resulting
diff. No automatic Git restore, receipt, or second commit request is used.

The separate isolated-batch capability creates a complete logical
`WorkspaceView` for arbitrary CLI tools. It uses copy-on-write clones where the
filesystem supports them and a portable copy or persistent-mirror fallback
otherwise. All tools receive view paths only. The initial boundary excludes
symlinks, special files, resource and manifest operations, file lifecycle
operations, and binary mutation.

An isolated batch records start metadata and validates only final patch targets
for regular-file status, containment, symlink absence, mode, size, and
nanosecond mtime. One target uses direct atomic replacement. Multiple targets
use an internal strict patch applied by plain `git apply`; no repository index
is required, and the generated patch is never caller input. This gives normal
hunk-conflict atomicity without claiming crash recovery.

## Related Work

* ADR-0010 provides atomic disk-write and timestamp rules.
* ADR-0016 retains direct sequential `semantic_batch` behavior.
* ADR-0029 keeps trusted LSP rename and WorkspaceEdit interpretation backend-owned.
* ADR-0032 records the direct and isolated execution boundary.

## Follow-up

* Apply the isolated capability to multi-site refactoring operations after
  their language-specific planning is approved.
* Add binary or resource operations only with their own publication semantics.
