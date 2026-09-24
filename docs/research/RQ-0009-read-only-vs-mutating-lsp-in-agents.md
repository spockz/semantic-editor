# RQ-0009: Read-Only vs. Mutating LSP in Agent Harnesses

* **Status**: Open
* **Category**: Agent Harness & LSP Integration
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

Direct developer feedback and guidance from agent harnesses like Antigravity reveal the current state of LSP integration in coding agents:

```markdown
# Antigravity Navigation, LSP Integration & Editing Guidelines (Observed Guidance)
1. Codebase Architecture & Navigation:
   - Antigravity does not maintain a persistent in-memory symbol/line map; navigate directly using explicit paths, symbol names, or @-mentions.
   - Avoid broad grep scans when architecture boundaries are known.
2. LSP & Tool Prioritization:
   - Use `goToDefinition` / `getWorkspaceSymbols` instead of `grep_search`.
   - Use `findReferences` to capture every call site before changing signatures.
   - Keep tool descriptions semantic so the planner routes symbol resolution directly to the LSP.
3. Editing & Verification Cycle:
   - Confine edits to minimal required file chunks.
   - Validate modifications post-edit using LSP diagnostics (`getDiagnostics`).
```

### The Core Finding: The "Read-Only" Asymmetry via MCP

Modern agent planners **already understand how to interact with Language Servers**, provided they are **exposed as an MCP server** (e.g. `gopls mcp`, `agent-lsp`, or custom MCP wrappers):

* Agent harnesses do not speak raw stateful LSP (JSON-RPC over stdio with buffer synchronization); they rely on MCP as the standard adapter layer.
* Planners are already prompted to prioritize MCP tools named after semantic LSP operations (`goToDefinition`, `findReferences`, `workspaceSymbols`).
* **The Blind Spot**: Planners currently use these MCP-LSP tools exclusively as **read-only advisors and verifiers**:
  * **Read (MCP-LSP)**: `goToDefinition`, `findReferences`, `workspaceSymbols`.
  * **Verify (MCP-LSP)**: `getDiagnostics`.
  * **Write (LLM Text Diff)**: The agent finds 40 call sites via LSP, but then manually streams 40 text-replacement diff chunks over the network.

This preserves all the primary failure modes: token churn, diff application drift, provider variance, and network retry fragility.

---

## 2. The Opportunity for `semedit`

Because harnesses like Antigravity already possess prompt logic to route semantic queries to MCP-wrapped LSPs, `semedit` does not need to teach the planner what an LSP is. Instead, `semedit` acts as the **mutating MCP server** that completes the loop:

```text
Current Agent Loop:
LSP Locate ──► LLM Text Diffs (thousands of tokens) ──► LSP Diagnostics

The semedit Closed Loop:
LSP Locate ──► Host LSP Mutate (20 tokens) ──► Host Diagnostics & Formatting
```

---

## 3. Exploration Paths

### A. Planner Tool Routing & Semantic Framing

Antigravity explicitly confirms: *"Keep tool descriptions and schemas semantic so the planner routes symbol resolution directly to the LSP."*

* We must test tool descriptions to determine which phrasing causes the planner to select `semantic_rename` or `extract_function` over built-in tools like `replace_file_content`.

### B. Bundled Mutation & Diagnostic Feedback

Instead of forcing the agent to make two separate tool calls (`apply_edit` $\rightarrow$ `getDiagnostics`), evaluate bundling the diagnostic check and code formatting directly inside the response of the mutation tool.

---

## 4. Sources & Prior Art

* **Antigravity LSP & Tool Prioritization Guidelines** (Observed harness behavior, 2026).
* **LSP Specification 3.17**: `textDocument/rename`, `textDocument/codeAction`, `workspace/applyEdit`.
* **Blackwell Systems `agent-lsp`**: [GitHub agent-lsp](https://github.com/blackwell-systems/agent-lsp): exposes LSP tools via MCP.
