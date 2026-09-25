# ADR-0009: Coexistence with LSP-MCP Servers via Distinct Naming & Profiles

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Agent environments (such as Antigravity or Claude Code) may already have an LSP-backed MCP server configured (e.g. `gopls mcp` or `agent-lsp`), providing read-only navigation tools like `goToDefinition` and `findReferences`.

The Model Context Protocol (MCP) does not support cross-server discovery; a server cannot query the client to see what other servers are connected. Registering duplicate tool definitions increases system prompt token overhead and can confuse the planner.

However, raw LSP-MCP tools remain coordinate-based (`line/character`), whereas `semedit`'s navigation tools are intent-based (`symbol: "Type.Method"`). Completely removing navigation would force agents back into manual line counting when no external symbol resolver is present.

## Decision

1. **Primary Identity as Mutating Engine**: `semedit`'s core toolset focuses on transactional mutations (`semantic_rename`, `organize_imports`, `extract_function`, `verify_diagnostics`).
2. **Distinct Intent-Based Naming**: Navigation tools in `semedit` use explicit intent names that never collide with raw LSP methods:
   * Use `resolve_symbol_location` (takes qualified identifier string) instead of `goToDefinition`.
   * Use `query_symbol_references` instead of `findReferences`.
3. **Configurable MCP Profiles**: Support a `--profile` flag on `semedit mcp`:
   * `--profile=mutations-only`: Exposes only mutating and verification tools; relies on the environment's existing LSP-MCP for navigation.
   * `--profile=full` (Default): Exposes both intent-based symbol location and mutation tools.
4. **Dynamic Language Scoping at Initialization**:
   During the `initialize` handshake, the server extracts workspace context (`rootUri`, `workspaceFolders`, or client `initializationOptions`) and identifies the active language backend. The tool schemas returned to the client (in `tools/list` and capability responses) are dynamically scoped:
   * **Language-Specific Tool Availability**: Tools with handlers only for a specific language (e.g. build dependency management, framework-specific actions) are omitted when that language is absent.
   * **Dynamic Parameter Enums**: Operation parameter schemas with dynamic constraints (such as `kind` in `semantic_replace_construct` or `access_modifier` in insertion tools) emit `enum` definitions restricted strictly to the valid constructs/modifiers supported by the active language backend, enabling LLM token decoders to enforce valid tokens via logit masking.
   * **Dynamic Change Propagation**: When workspace focus shifts or an explicit language context changes, the server emits `notifications/tools/list_changed` so the client re-requests the adapted schema definitions.

## Invariants

* `semedit` tool names must never collide with standard LSP method names.
* Mutating operations (`semantic_rename`, `organize_imports`, `verify_diagnostics`) are always exposed across all profiles.
* Tool schemas and parameter enums presented after `initialize` must strictly reflect the capabilities of the active language backend without polluting the schema with unsupported cross-language constructs.

## Consequences

* **Positive**: Seamless coexistence with existing LSP-MCP servers; eliminates planner confusion; allows users to minimize context token footprint via profiles; enforces schema correctness via language-scoped logit masking from initialization.
* **Negative**: Introduces dynamic schema generation in the MCP server layer requiring reactive updates via `notifications/tools/list_changed` when languages switch.
