# RQ-0011: Cache Invalidation, File Synchronization & Cross-LSP Consistency

* **Status**: Resolved
* **Category**: Systems & State Synchronization
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

When an AI agent operates in a workspace, it alternates between semantic tool calls (`semedit rename`), direct file writes (`write_to_file`), and read-only queries from a coexisting LSP-MCP server (`goToDefinition`, `findReferences`).

This introduces two distinct synchronization boundaries:

1. **Internal State Desync**: `semedit`'s own Tree-sitter AST or `gopls` subprocess working on stale data after external agent text edits.
2. **External LSP Stale Reads**: An external read-only LSP serving stale data after `semedit` mutates files on disk.

---

## 2. Technical Findings

### A. Internal Tools Freshness (`semedit` Engine)

* **Tree-sitter On-Demand Parsing**:
  * Tree-sitter does not require an asynchronous background file watcher.
  * `semedit` checks `os.Stat(file).ModTime()` prior to evaluating AST queries. If `stat.ModTime() > cached_mtime`, it discards the cached node and re-parses.
  * In Go and Rust, Tree-sitter parses a 2,000-line source file in under 2ms. Synchronous stat checking has zero race condition window and zero memory leak risk.
* **Internal LSP Synchronization**:
  * When `semedit` applies edits via its internal `gopls` instance, it sends `textDocument/didSave` and `workspace/didChangeWatchedFiles` before returning the tool result.
  * It blocks until the LSP server flushes its internal snapshot, ensuring verification steps (`verify_diagnostics`) evaluate current state.

### B. External Read-Path LSP Synchronization (Coexistence Model)

When the agent reads via an external LSP-MCP server (e.g., `agent-lsp`, `gopls mcp`) and writes via `semedit`:

1. **OS Watcher Debounce vs. LLM Turn Latency**:
   * External LSPs monitor the filesystem via OS notifications (`inotify`, `kqueue`, `FSEvents`) with a 50ms-100ms debounce timer to coalesce file writes.
   * Tool calls are serialized by the agent harness. An external read query can only execute on a subsequent agent turn:
     $$\text{semedit write tool call} \longrightarrow \text{tool returns} \longrightarrow \text{LLM inference} \longrightarrow \text{LSP read tool call}$$
   * Agent token generation takes 500ms-2000ms. This inference window strictly exceeds the OS watcher debounce period, allowing the external LSP to invalidate its package cache before the next read query arrives.

2. **Atomic Disk Write Protocol**:
   * External file watchers can misfire or ingest partial files if writes are streamed.
   * `semedit` writes new file contents to a sibling temporary file, calls `fsync()`, and executes an atomic rename (`os.Rename()`) over the destination.
   * To guard against filesystems with 1-second timestamp resolutions, `semedit` guarantees strictly advancing `mtime` (`mtime = max(now, old_mtime + 1ms)`).

3. **In-Memory Buffer Overlays (IDE vs. Headless)**:
   * **Headless agents (Antigravity CLI, Claude Code)**: No GUI editor exists; all LSPs operate against disk files. Invalidation via disk notifications is 100% reliable.
   * **IDE agents (Cursor, VS Code)**: Active editor tabs hold in-memory document overlays (`textDocument/didOpen`). The host editor monitors disk changes and reloads tabs, propagating `textDocument/didChange` to its internal LSP.

---

## 3. Resolution & Invariants

1. **Unified Broker Preference (`--profile=full`)**:
   * Running separate LSPs for reading and writing wastes system memory and creates avoidable file-watcher race conditions.
   * By default, `semedit` serves both intent-based navigation (`resolve_symbol_location`) and mutation (`semantic_rename`) over the same synchronized engine.
2. **Deterministic Mutation Protocol**:
   * When coexisting with external LSPs (`--profile=mutations-only`), `semedit` enforces atomic writes (`tmp` + `fsync` + `rename`) and advancing timestamps to guarantee external OS watchers trigger cleanly.

---

## 4. Sources & Prior Art

* **LSP 3.17 Specification**: [workspace/didChangeWatchedFiles](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#workspace_didChangeWatchedFiles).
* **gopls File Cache & Snapshot Architecture**: [golang/tools/gopls](https://github.com/golang/tools/tree/master/gopls).
* **Tree-sitter Incremental Parsing**: [tree-sitter.github.io](https://tree-sitter.github.io/tree-sitter/).
