# RQ-0025: Refactoring Preview and Bounded Commit

* **Status**: Resolved
* **Category**: Architecture & Safety
* **Date**: 2026-09-17

---

## Context

Snapshots and conflict-checked undo recover from completed changes, but higher-risk edit and refactoring capabilities need an inspectable proposal before mutation. Extraction, inline, move, safe delete, and signature change can affect multiple symbols or files and must not rely on blind text edits or partial application.

## Research Questions

1. What revision hash and structural-handle contract detects stale source between inspection and commit?
2. Which affected-file sets can receive an all-or-nothing bounded commit, and how are external changes reported?
3. How can each language backend own semantic analysis and edit validation without creating a generic arbitrary-WorkspaceEdit engine?
4. What preview format provides useful agent and CLI review without requiring an IDE UI?

## Resolution

The Go backend now derives complete-file candidates under the workspace lock and exposes an inspect, dry-run, and commit lifecycle. The host recomputes and validates candidate digests before mutation, records durable HMAC-authenticated receipts and journals in private user configuration state, and treats `.scratch` output as preview-only.

The initial boundary is intentionally narrow: existing regular Go files only, with no resource or workspace-manifest operations. Staging and parent-directory fsync make the mutation durable; a post-rename sync failure produces an explicit indeterminate-commit state. Recovery recognizes only known preimage and postimage states, and a completed mutation remains committed even when post-commit diagnostics report warnings.

## Related Work

* ADR-0018 provides snapshot and undo recovery.
* ADR-0029 keeps trusted LSP rename and WorkspaceEdit interpretation backend-owned.
* RQ-0024 defines the edit and refactoring capability families that depend on this protocol.

## Follow-up

* Apply this protocol to multi-site Go refactoring capabilities after their language-specific planning is approved.
* Expand the boundary only with an explicit resource-operation transaction design.
