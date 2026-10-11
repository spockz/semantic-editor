# Empirical Benchmark Report: Vanilla LLM vs. Semedit MCP

<style>
table.benchmark-tool-calls {
  width: 100%;
  table-layout: fixed;
}

table.benchmark-tool-calls th:first-child,
table.benchmark-tool-calls td:first-child {
  width: 3rem;
}

table.benchmark-tool-calls td {
  min-width: 0;
}

table.benchmark-tool-calls pre.benchmark-shell-command,
table.benchmark-tool-calls pre.benchmark-tool-arguments {
  width: 100%;
  max-width: 32rem;
  margin: 0.5rem 0 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: break-word;
  overflow-x: auto;
}

table.benchmark-tool-calls pre.benchmark-shell-command code {
  white-space: inherit;
}

.benchmark-delta-positive {
  color: var(--bs-success, #198754);
  font-weight: 700;
}

.benchmark-delta-negative {
  color: var(--bs-danger, #dc3545);
  font-weight: 700;
}
</style>

* **Date**: 2026-10-11 10:52:43 CEST

## Test case: `task-01-rename-local`

### Target: `opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/task_01_rename_local.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_01_rename_local.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct{}

func (s *Server) oldName() string {
	return "ok"
}

func Run() string {
	s := &Server{}
	return s.oldName()
}
```
</details>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 65.61s | 66.84s | N/A | — | — | — |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 0 | 0 | 0% | — | — | — |
| **Internal Tool Cycles** | 0 | 0 | 0% | — | — | — |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 0 | 0 | 0% | — | — | — |
| **Output Tokens** | 0 | 0 | 0% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Uncached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached vs Uncached Token Ratio** | — | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/task_01_rename_local.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_01_rename_local.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct{}

func (s *Server) oldName() string {
	return "ok"
}

func Run() string {
	s := &Server{}
	return s.oldName()
}
```
</details>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 65.61s | 65.12s | N/A | — | — | — |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 0 | 0 | 0% | — | — | — |
| **Internal Tool Cycles** | 0 | 0 | 0% | — | — | — |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 0 | 0 | 0% | — | — | — |
| **Output Tokens** | 0 | 0 | 0% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Uncached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached vs Uncached Token Ratio** | — | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

