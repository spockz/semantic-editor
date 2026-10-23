# ADR-0010: Disk Synchronization & Cache Invalidation Across Heterogeneous Tools

* **Status**: Accepted
* **Date**: 2026-09-15

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Preserve checked atomic writes and current working revisions; synchronize and verify through backend-owned receipts. Timestamps and external watcher delays do not prove freshness or completion; buffered results do not imply disk publication. |
| description | When `semedit` operates in an agent environment, it coexists with direct file modifications (`write_to_file`) and external read-only language servers (e.g., `agent-lsp`, `gopls mcp`). |
| role | invariant |
| scope | project |
| area | backend |
| languages | generic |

## Context

When `semedit` operates in an agent environment, it coexists with direct file modifications (`write_to_file`) and external read-only language servers (e.g., `agent-lsp`, `gopls mcp`).

This introduces two distinct synchronization challenges:

1. **Internal Cache Invalidation**: `semedit`'s own Tree-sitter CST and LSP client must never serve or compute over stale ASTs after external file changes.
2. **External LSP Cache Consistency**: When `semedit` modifies code on disk, external read-only language servers must reliably observe live data on subsequent agent turns without race conditions or partial file reads.

## Decision

1. **Tree-sitter On-Demand Synchronous Invalidation**:
   * Tree-sitter does not run a background file watcher.
   * Syntax queries use the current captured source view, including unpublished buffered edits. A timestamp change can invalidate cached syntax, but timestamps alone cannot prove freshness when external writers preserve them. Retained analysis uses content and membership validation under ADR-0032 and ADR-0060.
2. **Backend-Owned LSP Synchronization**:
   * A dependent LSP query must see the current working bytes. Go AST-only edits defer synchronization until publication or an actual query; gopls uses the pull-diagnostic, change-notification, and targeted save-fallback policy in ADR-0060.
   * Java synchronizes before LSP formatting, imports, and chained refactorings and retains explicit post-publication saves and version-matched push diagnostics under ADR-0068. Go notification suppression is not a transport-wide default.
   * Notification delivery does not prove analysis completion. The backend must obtain its revision- and scope-matched diagnostic result before reporting completed verification.
3. **Atomic Disk Write Protocol**:
   * When committing changes to disk, `semedit` writes to a sibling temporary file, flushes via `fsync()`, and atomically renames (`os.Rename()`) over the target file.
   * `semedit` enforces strictly advancing modification times (`mtime = max(now, old_mtime + 1ms)`) to prevent low-resolution filesystems from swallowing file modification notifications.
   * Validate captured preimages and canonical paths before publication. Lazy capture covers actual reads/writes and explicitly observed trees; it does not require a full workspace scan for every file write (ADR-0032).
4. **External Watcher Boundary**:
   * Atomic publication makes complete files available to external readers. Watcher debounce and time spent between agent turns do not guarantee that an independent language server has processed the change. Its client owns synchronization and completion evidence; semedit cannot promise freshness for another daemon.
5. **Unified Broker as Primary Target**:
   * The full execution profile offers navigation and mutation through one host. Each backend retains its own server and lifecycle; a profile does not merge language-server processes or guarantee absence of background contention.

## Invariants

* `semedit` never writes partial or non-fsynced source files directly to target paths.
* Syntax and semantic queries evaluate the correct current disk or working-buffer revision, not a timestamp-only approximation.
* Final published results identify completed disk writes and required synchronization. Buffered child results identify `applied_working`; returning such a result does not publish the stage or batch to disk.

## Consequences

* **Positive**: Checked atomic publication protects readers from partial files, while backend-owned synchronization preserves the working revision for dependent queries.
* **Negative**: Temporary-file, validation, and fsync work has a measurable cost. Atomic writes do not provide a multi-file transaction or eliminate external-writer races; conflicts and partial publication remain explicit.
