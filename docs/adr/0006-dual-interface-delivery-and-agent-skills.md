# ADR-0006: Dual-Interface Delivery (CLI + MCP) Paired with Skills

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Exposing an editing tool exclusively via the Model Context Protocol (MCP) or exclusively via a CLI introduces trade-offs:

* CLI-only is fast for shell scripts and terminal agents, but lacks rich interactive discoverability in graphical agent harnesses.
* MCP-only incurs JSON-RPC messaging overhead and cannot be invoked as a fast, scriptable 1-shot command.
* Furthermore, due to extensive pre-training on line diffs, frontier LLMs exhibit "model inertia"—they reflexively emit text search/replace patches even when specialized MCP tools are present.

## Decision

1. **Dual-Interface Binary**: The core engine compiles into a single binary (`semedit`) supporting:
   * **CLI Mode** (`semedit <command>`): Sub-millisecond execution for terminal scripts and direct agent commands.
   * **MCP Daemon Mode** (`semedit mcp`): Native Model Context Protocol server for Claude Code, Cursor, Windsurf, Antigravity, etc.
2. **Synchronized Agent Skills (`SKILL.md`)**: Ship alongside a clear, platform-agnostic `SKILL.md` containing operational rules and decision trees that override model diff inertia and enforce semantic tool routing.

## Invariants

* Every semantic operation supported over MCP must also be invocable via CLI.
* Releases must ship with updated agent steering instructions (`SKILL.md`).

## Consequences

* **Positive**: Maximum compatibility with diverse agent architectures; overcomes model bias against structured tooling.
* **Negative**: Requires maintaining both CLI flag parsers and MCP JSON-RPC schemas for each operation.
