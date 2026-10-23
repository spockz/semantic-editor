# ADR-0006: Dual-Interface Delivery (CLI + MCP) Paired with Skills

* **Status**: Accepted
* **Date**: 2026-09-15

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Offer CLI and MCP interfaces paired with `SKILL.md` and progressive language reference guides; govern skill updates upon language capability expansion. |
| description | Pair CLI and MCP interfaces with shared skills and language reference guides. |
| role | architecture |
| scope | project |
| area | mcp |
| languages | generic |

## Context

Exposing an editing tool exclusively via the Model Context Protocol (MCP) or exclusively via a CLI introduces trade-offs:

* CLI-only is fast for shell scripts and terminal agents, but lacks rich interactive discoverability in graphical agent harnesses.
* MCP-only incurs JSON-RPC messaging overhead and cannot be invoked as a fast, scriptable 1-shot command.
* Furthermore, extensive pre-training on line diffs gives frontier LLMs a form of "model inertia": they reflexively emit text search/replace patches even when specialized MCP tools are available.

## Decision

1. **Dual-Interface Binary**: The core engine compiles into a single binary (`semedit`) supporting:
   * **CLI Mode** (`semedit <command>`): Sub-millisecond execution for terminal scripts and direct agent commands.
   * **MCP Daemon Mode** (`semedit mcp`): Native Model Context Protocol server for Claude Code, Cursor, Windsurf, Antigravity, etc.
2. **Synchronized Agent Skills (`SKILL.md`)**: Ship alongside a clear, platform-agnostic `SKILL.md` containing operational rules and decision trees that override model diff inertia and enforce semantic tool routing.
3. **Progressive Per-Language Reference Guides**: Maintain modular reference guides (`skills/semedit/references/<lang>.md`) covering language-specific macro refactoring patterns and build verification workflows, conditionally loaded by agent harnesses.

## Invariants

* Every semantic operation supported over MCP must also be invocable via CLI.
* Releases must ship with updated agent steering instructions (`SKILL.md`).
* Whenever the operational scope of a supported language expands (or a new language backend is introduced), contributors must verify whether and how to update the corresponding language reference guide (`skills/semedit/references/<lang>.md`) to document new macro refactoring patterns and compiler verification workflows.

## Consequences

* **Positive**: Maximum compatibility with diverse agent architectures; overcomes model bias against structured tooling.
* **Negative**: Requires maintaining both CLI flag parsers and MCP JSON-RPC schemas for each operation.
