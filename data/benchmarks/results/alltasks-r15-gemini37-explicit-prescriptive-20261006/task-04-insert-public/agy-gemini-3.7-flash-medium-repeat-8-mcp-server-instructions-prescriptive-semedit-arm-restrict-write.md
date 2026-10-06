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

* **Date**: 2026-10-06 09:19:04 CEST

## Test case: `task-04-insert-public`

### Target: `agy/gemini-3.7-flash-medium (repeat 8)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_04_insert_public.txtar` (`sha256:f10af4ed1cfb9fbd89122b1a (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add public constructor func InitServer() *Server placed before private helpers. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Call semantic_insert_function with {\"file\":\"api/server.go\",\"source\":\"func InitServer() *Server { return &Server{} }\",\"placement\":\"before_symbol\",\"target_symbol\":\"Server.internalRun\"}. Do not change other declarations. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct {
	Port int
}

func (s *Server) Start() {}

func (s *Server) internalRun() {}
```
</details>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 2.52s | 2.62s | N/A | 2.78s | 3.61s | N/A |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | 1 | 1 | N/A |
| **Internal Tool Cycles** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Output Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Input Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Cached Input Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Uncached Input Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Cached vs Uncached Token Ratio** | — | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_04_insert_public.txtar` (`sha256:f10af4ed1cfb9fbd89122b1a (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add public constructor func InitServer() *Server placed before private helpers. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add public constructor func InitServer() *Server placed before private helpers. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct {
	Port int
}

func (s *Server) Start() {}

func (s *Server) internalRun() {}
```
</details>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 2.52s | 3.04s | N/A | 2.78s | 4.11s | N/A |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | 1 | 1 | N/A |
| **Internal Tool Cycles** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Output Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Input Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Cached Input Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Uncached Input Tokens** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Cached vs Uncached Token Ratio** | — | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

