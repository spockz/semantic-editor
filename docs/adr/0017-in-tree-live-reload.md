# ADR-0017: In-Tree Live-Reload and Dynamic Tool Discovery for MCP Server

* **Status**: Accepted
* **Date**: 2026-09-17

---

## Context

When AI coding agents dogfood `semedit` to develop new semantic editing capabilities within `semedit` itself, a workflow friction emerges:

1. The agent introduces a new capability (e.g. `semantic_replace_body`, `semantic_insert_case`) and compiles it into `bin/semedit` via `make promote`.
2. The agent harness remains connected to the long-running MCP process launched at session inception.
3. The running server process does not reload the updated binary from disk, and the client retains the initial static `tools/list` schema.
4. Restarting the MCP server externally typically requires tearing down the agent environment or restarting the IDE session, forcing agents to fall back to text replacement tools (`replace_file_content`).

Furthermore, arbitrary passive file watching during active refactoring is dangerous: recompilations or in-flight tool requests could trigger sudden process re-execs mid-operation.

---

## Decision

We introduce an opt-in live-reload mechanism and dynamic tool discovery protocol for `semedit mcp`:

1. **CLI Flag Gate (`--live-reload`)**:
   * Added `--live-reload` (boolean, default `false`) to `semedit mcp`.
   * When disabled, the server runs in standard production mode with no reload capabilities or overhead.

2. **Explicit Administrative Tool (`semantic_reload`)**:
   * Registered in `tools/list` strictly when `--live-reload` is enabled.
   * Takes an empty parameter schema (`additionalProperties: false`).
   * Transmits `{status: "ok", message: "server reloading"}` before executing in-place re-exec.
   * Invokes in-place process re-exec via `syscall.Exec` on Unix systems (preserving stdio file descriptors 0, 1, and 2 across `execve`).
   * Provides a fallback error on Windows where in-place `execve` is unsupported.

3. **Dynamic Tool Schema Discovery (`notifications/tools/list_changed`)**:
   * In `initialize` handler, the server advertises `"capabilities": {"tools": {"listChanged": true}}`.
   * When an initialized handshake completes (`notifications/initialized`) under `--live-reload`, the server emits `notifications/tools/list_changed` to prompt compliant clients to refresh their tool definitions.

4. **Atomic Promotion in Build Automation**:
   * In `Makefile`, the `promote` target replaces `rm -f bin/semedit && cp bin/semedit-next bin/semedit` with atomic rename:

     ```make
     cp bin/semedit-next bin/semedit.tmp && mv -f bin/semedit.tmp bin/semedit
     ```

   * This eliminates the race condition where `syscall.Exec` could encounter `ETXTBSY` or execute a partially copied binary during promotion.

---

## Invariants

1. **Strict Opt-In Invariant**: Live reload and `semantic_reload` must remain quiescent unless `--live-reload` is explicitly supplied at startup.
2. **Explicit Trigger Over Passive Watching**: Server reloading is triggered exclusively by explicit tool invocation (`semantic_reload`), preventing mid-execution termination during compilations.
3. **Stdio Descriptor Preservation**: On Unix systems, re-exec must use `syscall.Exec(os.Executable(), os.Args, os.Environ())` so file descriptors 0, 1, and 2 remain open and client communication is uninterrupted.
4. **Atomic Promotion Guarantee**: Promotion of new binaries to `bin/semedit` must proceed via temporary file copy and atomic rename (`mv -f`).

---

## Consequences

* Enables continuous self-modification and dogfooding without agent harness restarts.
* Client harnesses dynamically discover newly promoted semantic tools via standard MCP notifications.
* Production MCP server instances remain immutable and protected from accidental restarts.
