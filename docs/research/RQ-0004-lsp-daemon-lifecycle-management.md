# RQ-0004: LSP Daemon Lifecycle Management

* **Status**: Open
* **Category**: Runtime Architecture & Process Management
* **Last Updated**: 2026-09-15

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

* **Microsoft `multilspy`**: [GitHub multilspy](https://github.com/microsoft/multilspy) — headless multi-language server lifecycle management in Python.
* **`gopls` Daemon Mode**: [gopls documentation](https://pkg.go.dev/golang.org/x/tools/gopls) — sharing gopls instances across processes via remote socket.
* **Neovim / Kakoune LSP Daemons**: Architectures using persistent background servers for headless editor clients.
