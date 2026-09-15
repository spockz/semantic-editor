# RQ-0012: Ingress Protocols (LSP vs. MCP vs. CLI) & Downstream Fan-Out

* **Status**: Open
* **Category**: Protocol Architecture & Decoupling
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

Repositories often combine code (`.go`, `.rs`) with documentation (`.md`) and configuration (`.yaml`). When refactoring, an edit should ideally propagate across both code and referencing documentation.

Historically, tools treated "being an LSP proxy" and "being an agent tool" as mutually exclusive architectures. In reality, the **ingress interface** (how the client triggers an edit) is orthogonal to the **fan-out engine** (how `semedit` coordinates downstream language tools).

---

## 2. The Decoupled 3-Layer Architecture

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   Layer 1: Ingress Interfaces                          │
│                                                                        │
│   ┌────────────────────┐ ┌───────────────────┐ ┌───────────────────┐  │
│   │     MCP Ingress    │ │    LSP Ingress    │ │    CLI Ingress    │  │
│   │  (Agent Harnesses) │ │ (IDEs & LSP Agents│ │ (Scripts & Shell) │  │
│   └─────────┬──────────┘ └─────────┬─────────┘ └─────────┬─────────┘  │
└─────────────┼──────────────────────┼─────────────────────┼────────────┘
              │                      │                     │
              ▼                      ▼                     ▼
┌────────────────────────────────────────────────────────────────────────┐
│              Layer 2: Unified Fan-Out & Transaction Core               │
│                                                                        │
│  • Resolves symbol targets across code and documentation.              │
│  • Dispatches fan-out commands to active downstream adapters.          │
│  • Merges partial changes into a single unified `WorkspaceEdit`.       │
│  • Runs post-edit formatting, diagnostic checks, and snapshotting.     │
└────────────────────────────────────┬───────────────────────────────────┘
                                     │
        ┌────────────────────────────┼────────────────────────────┐
        ▼                            ▼                            ▼
┌──────────────┐             ┌──────────────┐             ┌──────────────┐
│  gopls (Go)  │             │ marksman(MD) │             │ Tree-sitter  │
└──────────────┘             └──────────────┘             └──────────────┘
                    Layer 3: Downstream Tool Adapters
```

---

## 3. Ingress Protocol Characteristics

| Ingress | Best Suited For | Capability Handling | Output Payload |
| :--- | :--- | :--- | :--- |
| **MCP** | AI Agent Harnesses (Antigravity, Claude Code, Cursor) | High-level declarative tool schemas (`semantic_rename`). | JSON tool results with bundled diagnostics and preview diffs. |
| **LSP** | IDEs (VS Code, Neovim) & LSP-native agents | Dynamic registration (`client/registerCapability`) scoped by `documentSelector`. | Standard LSP `WorkspaceEdit`. |
| **CLI** | Shell scripts, CI/CD, and non-MCP agents (Aider) | Direct command-line arguments (`semedit rename`). | Formatted stdout / stderr with exit codes. |

---

## 4. Downstream Fan-Out & Cross-Domain Coordination

Regardless of which ingress triggered the edit, the core engine:

1. **Identifies Target Scope**: Queries the primary language server (e.g. `gopls` for `.go`).
2. **Finds Secondary References**: Queries secondary tools (e.g. `marksman` or Tree-sitter for `.md` references and doc links).
3. **Merges Edit Payloads**: Combines file changes into a single atomic change-set:
   * Returns it as an LSP `WorkspaceEdit` if triggered via LSP.
   * Applies it and returns a diagnostic summary if triggered via MCP or CLI.

---

## 5. Sources & Prior Art

* **LSP 3.17 Specification on WorkspaceEdit**: [WorkspaceEdit](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#workspaceEdit).
* **Model Context Protocol Specification**: [modelcontextprotocol.io](https://modelcontextprotocol.io).
* **`efm-langserver`**: [GitHub efm-langserver](https://github.com/mattn/efm-langserver).
