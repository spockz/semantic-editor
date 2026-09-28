# RQ-0012: Ingress Protocols (LSP vs. MCP vs. CLI) & Downstream Fan-Out

* **Status**: Open
* **Category**: Protocol Architecture & Decoupling
* **Last Updated**: 2026-09-27

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

## 5. Workspace Diagnostics Through LSP Ingress

[ADR-0008](../adr/0008-decoupling-ingress-interfaces-from-fan-out-engine.md) accepts an optional LSP endpoint over the shared core. A concrete first use case is providing `workspace/diagnostic` to editors and LSP-native agents when downstream language servers only expose document pull diagnostics. Current gopls integration uses `textDocument/diagnostic`; absence of workspace pull does not mean absence of workspace analysis.

The collector should use downstream workspace pull where supported and aggregate document reports otherwise. Preserve standard diagnostic fields, document URIs, and known versions. MCP can share that report model and include semedit coverage metadata without claiming wire compatibility with LSP. The broader verification workflow also includes normalization and filesystem checks; an LSP diagnostic request must remain read-only.

### Implementation Questions Still Open

* How should a session route initialized workspace folders, source-driven language selection, and document open/change/close events to downstream servers while preserving unsaved contents and versions?
* How should source enumeration, build constraints, nested modules, and provider failures determine report coverage? Define a protocol-compatible failure path for incomplete collection and how additional coverage details reach LSP clients without inventing fields in standard reports.
* How should document and workspace requests share collection, cancellation, progress, and partial results? Partial findings followed by a failure must not be mistaken for a complete clean result.
* What persistent state and invalidation rules justify result IDs and unchanged reports? Initially prefer full reports; do not promise incremental reuse before the state model supports it.
* How should the shared engine separate edit planning from disk publication so LSP rename/code actions return version-aware, client-applied `WorkspaceEdit` results while MCP/CLI retain their application workflow?
* How should clients use semedit alongside existing language servers without duplicate diagnostics or competing edit providers? Keep the initial advertised capabilities bounded rather than forwarding every downstream feature.

### First Milestone and Evidence

Implement initialization, capability negotiation, document synchronization, and document/workspace diagnostics before mutation requests. Keep rename and code actions as the next milestone; completion and hover are outside the initial scope.

Use an actual LSP client session to demonstrate that an unsaved edit changes diagnostics, a correction clears them, and document versions remain consistent. Include a diagnostic in an unopened source file, multiple applicable source scopes, and a failed provider. Exercise both native workspace pull and document aggregation, cancellation, and a request made before complete collection. Confirm requests do not alter disk contents or run mutating normalization hooks. Later mutation coverage must prove that planning returns edits without modifying files before the client applies them.

This research remains **Open**: the architectural direction is accepted, but LSP ingress and the lifecycle behavior above are not implemented by merely sharing diagnostic structures.

---

## Protocol Surface and Adoption Boundary

The architectural direction remains a shared semantic engine with multiple ingress adapters; this does not establish LSP as the only interface. Before adding LSP ingress, enumerate the client and server methods in the targeted LSP version and map each method to an existing semedit operation, a possible adapter-only feature, or an explicit non-goal. Keep protocol methods distinct from product operations: matching names do not imply matching semantics, safety, or workspace scope.

For each overlap, record whether standard LSP already supplies the user value, whether semedit adds intent-level selection or stronger validation, and what extra state an LSP adapter would require (document synchronization, versions, cancellation, progress, client-applied edits, capabilities, and partial-failure reporting). Evaluate a full LSP-only ingress against the supported MCP and CLI harnesses, including whether each can register a stdio LSP server as an agent tool and preserve semantic tool schemas and steering. Retain MCP/CLI unless evidence shows equivalent discovery, invocation, and result handling across target harnesses.

Related investigations: [RQ-0010](RQ-0010-agent-harness-integration-matrix.md) tracks harness protocol support; [RQ-0013](RQ-0013-mcp-tool-overlap-and-lsp-coexistence.md) tracks overlap and coexistence; [RQ-0038](RQ-0038-makefile-language-server-versus-parser.md) tracks Makefile LSP/parser evidence.

## 6. Sources & Prior Art

* **LSP 3.17 Specification on WorkspaceEdit**: [WorkspaceEdit](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#workspaceEdit).
* **Model Context Protocol Specification**: [modelcontextprotocol.io](https://modelcontextprotocol.io).
* **`efm-langserver`**: [GitHub efm-langserver](https://github.com/mattn/efm-langserver).

* **LSP 3.17 Workspace Diagnostics**: [workspace/diagnostic](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#workspace_diagnostic).
* **Gopls Diagnostics**: [Document pull and asynchronous analysis](https://go.dev/gopls/features/diagnostics).
