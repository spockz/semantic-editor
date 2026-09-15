# RQ-0010: Agent Harness Integration Matrix & Steering Configurations

* **Status**: Open
* **Category**: Integration & Ecosystem
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

Because modern agent harnesses (Antigravity, Claude Code, Cursor, Windsurf, Continue) have standardized on the **Model Context Protocol (MCP)** for external tools, `semedit` does not need custom IDE plugins.

However, each platform has distinct configuration conventions for:

1. Registering local MCP servers.
2. Directing agent planners to prioritize specific tools (prompt rules, markdown rulebooks, skills).

---

## 2. Integration Matrix

| Agent Harness | Connection Protocol | Integration Tier | Registration Mechanism | Steering Configuration File |
| :--- | :--- | :---: | :--- | :--- |
| **Google Antigravity** | Stdio MCP | **Tier 1** | Project or user MCP config | `skills/semedit/SKILL.md` |
| **Claude Code** | Stdio MCP / CLI | **Tier 1** | `claude mcp add semedit -- semedit mcp` | `CLAUDE.md` |
| **Cursor** | Stdio MCP | **Tier 1** | Cursor Settings $\rightarrow$ MCP Server | `.cursor/rules/semedit.mdc` |
| **Windsurf** | Stdio MCP | **Tier 1** | `~/.codeium/windsurf/mcp_config.json` | `.windsurfrules` |
| **Continue.dev** | Stdio MCP | **Tier 1** | `~/.continue/config.json` (`mcpServers`) | `.continuerules` |
| **VS Code Copilot** | Stdio MCP | **Tier 1** | `.vscode/mcp.json` | `.github/copilot-instructions.md` |
| **Aider** | CLI Subprocess | **Tier 2** | Direct invocation via `/run semedit ...` | `.aider.conf.yml` |

---

## 3. Exploration Paths

### A. The `semedit init` Scaffolder

Investigate providing a CLI command (`semedit init --harness <type>`) that automatically generates the appropriate MCP configuration block and corresponding rule files for the active workspace.

### B. Standardizing the Universal Rule Template

Draft a concise, standardized rule block explaining tool priorities that can be injected into any of the steering files:

```markdown
# Semantic Refactoring Rules
- NEVER use raw text search/replace (e.g. `replace_file_content`) for symbol renaming or function extraction.
- ALWAYS use `semantic_rename` for renaming types, functions, variables, or packages.
- ALWAYS use `extract_function` when factoring code into helpers so parameters and return types are inferred by the compiler.
```

---

## 4. Sources & Prior Art

* **Model Context Protocol Specification**: [modelcontextprotocol.io](https://modelcontextprotocol.io).
* **Anthropic Claude Code MCP Documentation**: [Claude Code Docs](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code).
* **Cursor Rules Architecture**: [Cursor Rules Documentation](https://docs.cursor.com/context/rules-for-ai).
