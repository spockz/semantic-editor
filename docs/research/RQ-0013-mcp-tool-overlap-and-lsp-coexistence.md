# RQ-0013: MCP Tool Overlap, Coexistence & Multi-Server Detection

* **Status**: Open
* **Category**: Protocol & Ecosystem Coexistence
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

In many agent setups (e.g. Antigravity, Claude Code), a user may already have an LSP-mimicking MCP server configured (such as `gopls mcp` or `agent-lsp`), providing read-only navigation tools like `goToDefinition` and `findReferences`.

If `semedit mcp` also registers identical navigation tools:

1. **Tool Redundancy & Context Bloat**: Every tool definition consumes prompt tokens on every turn.
2. **Planner Confusion**: Redundant tool schemas increase planner routing ambiguity.
3. **Can `semedit` Detect Existing MCP Servers?**: The Model Context Protocol is client-server with no cross-server discovery API.

---

## 2. Technical Findings

### A. MCP Protocol Limitations on Discovery

* The MCP specification provides no mechanism for a server to inspect other servers connected to the same client.
* Host-side detection (inspecting client config files like `~/.gemini/antigravity/mcp/` or local `gopls` sockets) is brittle across diverse client platforms.

### B. Coordinate-Based vs. Intent-Based Navigation

* Raw LSP-MCP tools strictly require coordinates: `goToDefinition(uri, line, character)`.
* `semedit`'s symbol lookup is **intent-based**: `resolve_symbol_location(symbol: "Server.Start")`.
* Even when an LSP-MCP server is running, the agent still benefits from `semedit`'s symbol resolution to bridge high-level symbol queries to coordinates.

---

## 3. Exploration Paths

### A. Configurable MCP Profiles

Provide CLI flags to control which toolsets `semedit mcp` registers:

* `--profile=mutations-only`: Registers only mutating tools (`semantic_rename`, `organize_imports`, `extract_function`, `verify`). Relies on existing LSP-MCP for navigation.
* `--profile=full` (Default): Exposes both intent-based symbol resolution and mutation tools.

### B. Distinct Semantic Naming

Ensure `semedit` tool names never collide with standard LSP method names:

* Use `resolve_symbol_location` instead of `goToDefinition`.
* Use `query_symbol_references` instead of `findReferences`.

---

### C. LSP Method-to-Operation Inventory

Build a versioned inventory of LSP client requests, server requests, notifications, and lifecycle methods relevant to semantic lookup, references, diagnostics, rename, code actions, formatting, and workspace edits. Map each to semedit's operation registry as exact semantic overlap, partial overlap, transport support only, or no counterpart. Record capability negotiation and state requirements, not just method-name similarity. Include Makefile's existing `make-ls`-backed selected-file lookup under [ADR-0048](../adr/0048-bash-and-makefile-read-only-backends.md); [RQ-0038](RQ-0038-makefile-language-server-versus-parser.md) governs whether its parser can safely support mutation.

Do not treat an LSP method inventory as a migration decision. Compare client/harness integration, operation intent and schemas, unsaved-buffer semantics, server lifecycle, cross-language scope, and client-applied edit behavior through [RQ-0012](RQ-0012-lsp-multiplexing-vs-multi-daemon-orchestration.md) and [RQ-0010](RQ-0010-agent-harness-integration-matrix.md).

## 4. Sources & Prior Art

* **MCP Specification**: [modelcontextprotocol.io](https://modelcontextprotocol.io).
* **Antigravity Tool Prioritization Guidelines** (Observed harness behavior, 2026).
* **Blackwell Systems `agent-lsp`**: [GitHub agent-lsp](https://github.com/blackwell-systems/agent-lsp).
