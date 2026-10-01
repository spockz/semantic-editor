# RQ-0004: LSP Daemon Lifecycle Management

* **Status**: Open
* **Category**: Runtime Architecture & Process Management
* **Last Updated**: 2026-09-30

---

## 1. Problem Context

Language servers exhibit radically different startup profiles:

* `gopls`: Boots in $<50\text{ms}$ on typical Go packages; CLI mode can run ephemeral 1-shot commands.
* `rust-analyzer`: Takes $1\text{–}10\text{s}$ to load crates and macro expansions.
* `jdtls` (Eclipse) / `HLS` (Haskell): Can take $5\text{–}30\text{s}$ to initialize the JVM/GHC environment and build project indices.

Running heavy LSPs ephemerally on every CLI command would result in unacceptable latency ($>10\text{s}$ per edit).

---

## 2. Exploration Paths

### A. Local Domain Socket Daemon (`semeditd`)

* Implement a lightweight background daemon listening on a local Unix domain socket (e.g. `/tmp/semedit-<hash>.sock`).
* The daemon keeps the active LSP server warm in memory.
* CLI commands (`semedit rename ...`) act as thin RPC clients sending requests to the daemon, receiving edits in $<20\text{ms}$.

### B. Auto-Spawn & Idle Timeout

* The daemon starts automatically upon the first `semedit` invocation.
* Shuts down automatically after an idle timeout (e.g. 30 minutes of inactivity) to prevent runaway memory usage on host machines.

---

## 3. Sources & Prior Art

* **Microsoft `multilspy`**: [GitHub multilspy](https://github.com/microsoft/multilspy): headless multi-language server lifecycle management in Python.
* **`gopls` Daemon Mode**: [gopls documentation](https://pkg.go.dev/golang.org/x/tools/gopls): shares gopls instances across processes via a remote socket.
* **Neovim / Kakoune LSP Daemons**: Architectures using persistent background servers for headless editor clients.

## 4. MCP Go LSP Sessions

A copied `task-12-rename-scale-05-multifile` fixture has eight Go files. Twelve
warm-cache trials of fresh gopls pull sessions measured initialization at roughly
40–90 ms and diagnostic requests at 159–361 ms. The shutdown response took
1.03–1.08 seconds in every timed trial. These are fixture measurements, not a
general startup guarantee or timings recovered from historical agent runs.

[ADR-0060](../adr/0060-mcp-owned-go-diagnostic-sessions.md) accepts persistent Go
LSP sessions owned by an MCP instance. They synchronize editable source
buffers before each scan and rename, synchronize AST refactorings including
deferred batch edits, and preserve requested diagnostic scope. Shutdown moves
to instance end or session replacement. The benchmark harness retains MCP tool
timing records beside the transcript through `mcp_metrics_file` provenance.

This research remains open for CLI daemon sharing, idle policy, other language
servers, and native gopls refactoring actions with validated operation contracts.
