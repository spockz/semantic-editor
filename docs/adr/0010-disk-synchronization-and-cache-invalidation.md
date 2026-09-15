# ADR-0010: Disk Synchronization & Cache Invalidation Across Heterogeneous Tools

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

When `semedit` operates in an agent environment, it coexists with direct file modifications (`write_to_file`) and external read-only language servers (e.g., `agent-lsp`, `gopls mcp`).

This introduces two distinct synchronization challenges:

1. **Internal Cache Invalidation**: `semedit`'s own Tree-sitter CST and LSP client must never serve or compute over stale ASTs after external file changes.
2. **External LSP Cache Consistency**: When `semedit` modifies code on disk, external read-only language servers must reliably observe live data on subsequent agent turns without race conditions or partial file reads.

## Decision

1. **Tree-sitter On-Demand Synchronous Invalidation**:
   * Tree-sitter does not run a background file watcher.
   * `semedit` performs an `os.Stat(file).ModTime()` check before any AST query. If `stat.ModTime() > cached_mtime`, the cache is invalidated and re-parsed synchronously.
2. **Internal LSP Buffer Flushing**:
   * When `semedit` applies edits via its internal `gopls` instance, it sends `textDocument/didSave` and `workspace/didChangeWatchedFiles` before returning the tool result, blocking until the LSP flushes its snapshot.
3. **Atomic Disk Write Protocol**:
   * When committing changes to disk, `semedit` writes to a sibling temporary file, flushes via `fsync()`, and atomically renames (`os.Rename()`) over the target file.
   * `semedit` enforces strictly advancing modification times (`mtime = max(now, old_mtime + 1ms)`) to prevent low-resolution filesystems from swallowing file modification notifications.
4. **Harness Turn-Boundary Reliance**:
   * External LSPs use OS watchers (`inotify`, `kqueue`) with 50ms-100ms debounce windows. Because agent harnesses serialize tool execution and LLM inference requires 500ms-2000ms per turn, external watchers are guaranteed to process disk invalidations before the next read query arrives.
5. **Unified Broker as Primary Target**:
   * `--profile=full` remains the recommended architecture, eliminating cross-daemon competition and file-watcher latency by unifying intent navigation and mutation in a single synchronized engine.

## Invariants

* `semedit` never writes partial or non-fsynced source files directly to target paths.
* Internal Tree-sitter AST queries must always evaluate against the current on-disk `mtime`.
* `semedit` tool completion implies all modified files have been synced to disk and internal LSP sessions updated.

## Consequences

* **Positive**: Deterministic cache freshness without relying on complex, error-prone cross-server IPC; protects external read-only LSPs against partial file reads; eliminates race conditions between write and read tool calls.
* **Negative**: Ephemeral disk I/O overhead from atomic temp-file creation and fsync (negligible for typical refactoring file counts).
