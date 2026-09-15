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

## Invariants

* `semedit` tool names must never collide with standard LSP method names.
* Mutating operations (`semantic_rename`, `organize_imports`, `verify_diagnostics`) are always exposed across all profiles.

## Consequences

* **Positive**: Seamless coexistence with existing LSP-MCP servers; eliminates planner confusion; allows users to minimize context token footprint via profiles.
* **Negative**: Introduces a CLI configuration option that users must be aware of when tuning token efficiency.
