# RQ-0021: In-Tree Live-Reload & Dynamic Tool Schema Discovery for MCP Server

* **Status**: Resolved
* **Category**: Runtime & Developer Experience
* **Date**: 2026-09-17

---

## Context & Problem Statement

When AI coding agents (such as Antigravity, Claude Code, or Cursor) dogfood `semedit` to implement new semantic editing capabilities for `semedit` itself, a significant workflow friction occurs:

1. The agent introduces a new capability (e.g., `semantic_replace_body`, `semantic_insert_case`) and compiles it via `make promote` into `bin/semedit`.
2. The agent's active MCP connection is attached to the *already running* `semedit mcp` process started when the agent session began.
3. The running server binary on memory does not reflect the newly compiled binary on disk.
4. The MCP client (the agent harness) cannot discover the newly registered tools in `tools/list` unless the entire server process restarts or emits a dynamic tools notification.
5. Because restarting an MCP connection externally often requires restarting the IDE, reloading the window, or tearing down the agent environment, agents resort to manual editing tools (`write_to_file`, `replace_file_content`) instead of dogfooding their freshly created semantic tools.

To enable true iterative self-hosting, `semedit mcp` needs a mechanism to detect binary updates, reload itself in-place without breaking the transport pipe, and notify the MCP client of schema updates.

Crucially, **in production deployments, unexpected in-flight process restarts or file-watching overhead must be avoided.** Therefore, this behavior must be strictly opt-in and gated behind a CLI flag: `--live-reload`.

---

## Protocol & Architectural Constraints

### 1. The MCP Transport Model

MCP standard servers communicate over stdio (`stdin` / `stdout`) using JSON-RPC 2.0. If the server process exits abruptly, the host harness detects an EOF on `stdout` and marks the server disconnected or errored.

### 2. Dynamic Schema Discovery (`notifications/tools/list_changed`)

The Model Context Protocol specification defines the standard notification:

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/tools/list_changed"
}
```

When a server emits this notification, compliant clients (including Antigravity and modern MCP harnesses) re-issue a `tools/list` request, dynamically updating their available tool schemas without breaking the session.

### 3. Unix In-Place Execution (`syscall.Exec`)

Under POSIX/Unix systems (macOS, Linux), a process can replace its own execution image with a newly built executable via `syscall.Exec(os.Args[0], os.Args, os.Environ())`:

* The PID remains unchanged.
* Open file descriptors (specifically `stdin` descriptor `0` and `stdout` descriptor `1`) are preserved across `execve`.
* The client pipe remains connected with zero network/stdio disconnection events.

---

## Investigated Approaches

### Approach A: Supervisor / Process Proxy

* **Architecture**: When launched with `semedit mcp --live-reload`, the command acts as a lightweight supervisor process that spawns `bin/semedit mcp --worker` as a child process and proxies stdio pipes. When `bin/semedit` is recompiled, the supervisor terminates the child, spawns the new binary, and forward `notifications/tools/list_changed`.
* **Pros**: Complete crash isolation; if the newly compiled binary fails to boot, the supervisor can report an error or rollback.
* **Cons**: Introduces process hierarchy, pipe buffering latency, and signal forwarding complexity.

### Approach B: In-Process Binary Watcher + In-Place `syscall.Exec`

* **Architecture**: When `--live-reload` is active, the server tracks the `mtime` and inode/size of `os.Executable()`.
  * Before handling any tool invocation (or via an explicit background poll / `mtime` check), if the binary on disk is newer than process start time:
    1. Complete any current active in-flight request.
    2. Invoke `syscall.Exec(os.Args[0], os.Args, os.Environ())` in-place.
    3. The fresh binary boots, handles initialization, and immediately transmits `notifications/tools/list_changed` to the client.
* **Pros**: Zero additional processes; minimal code footprint; leverages POSIX file descriptor preservation; stdio never closes.
* **Cons**: Only applicable on POSIX systems (macOS/Linux; Windows requires fallback or process spawning); in-flight state is wiped (acceptable since `semedit` is stateless).

### Approach C: Explicit Admin Tool (`semantic_reload`)

* **Architecture**: When `--live-reload` is set, `semedit mcp` registers an additional meta-tool: `semantic_reload`.
  * After the agent runs `make promote`, it explicitly calls `semantic_reload`.
  * The server re-execs or refreshes its internal tool registry and returns success, followed by `notifications/tools/list_changed`.
* **Pros**: 100% deterministic; reload never happens mid-command or while a compiler is half-way through writing the binary.
* **Cons**: Requires the LLM agent to remember to invoke `semantic_reload`.

---

## Recommended Hybrid Architecture

A unified solution combining **Approach B & C** under the `--live-reload` flag:

1. **Flag Guard**: Add `--live-reload` (boolean flag, default `false`) to `semedit mcp`. When absent, `semedit mcp` runs in standard immutable production mode with zero file watching or re-exec overhead.
2. **Explicit Administrative Tool**: When `--live-reload=true`, the server registers `semantic_reload` in `tools/list`:
   * Schema: `"description": "Reloads the semedit MCP server in-place after recompilation (make promote) and emits notifications/tools/list_changed to discover newly added tools."`
   * Action: Waits for drain, re-executes `os.Executable()` via `syscall.Exec` (preserving stdio), and sends `notifications/tools/list_changed`.
3. **Passive Check on Request**: Optionally, when `--live-reload=true`, check binary `mtime` before executing each tool turn. If stale, automatically trigger reload before executing the request.

---

## Open Questions

1. **Harness Compatibility**: Do all target harnesses (Antigravity, Claude Code, Cursor, Windsurf) immediately re-fetch `tools/list` upon receiving `notifications/tools/list_changed`, or do some require an explicit client turn?
2. **Binary Replacement Race Window**: During `make promote`, `cp bin/semedit-next bin/semedit` briefly opens the file for writing. If a reload triggers while the file copy is in progress, `syscall.Exec` could encounter `ETXTBSY` or execute a truncated binary. (Mitigated by atomic rename: `mv bin/semedit-next bin/semedit`).
3. **Cross-Platform Compatibility**: How to handle Windows if `syscall.Exec` is not supported without spawning a child process?

---

## Next Steps

1. Draft ADR for `semedit mcp --live-reload` and `semantic_reload`.
2. Ensure `Makefile` promotion uses atomic rename (`mv` or atomic swap) rather than `cp` to prevent binary execution collisions.
3. Implement `semantic_reload` and stdio descriptor-preserving re-exec in `internal/mcp`.
