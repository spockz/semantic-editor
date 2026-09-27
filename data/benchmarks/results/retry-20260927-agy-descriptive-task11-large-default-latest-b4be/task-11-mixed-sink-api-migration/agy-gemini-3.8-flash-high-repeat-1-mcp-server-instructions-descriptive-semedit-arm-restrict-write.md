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

* **Date**: 2026-09-27 00:05:24 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `agy/gemini-3.8-flash/high (repeat 1)`

#### Configuration: default prompt · descriptive MCP instructions · write semedit restriction

* **Run Provenance**: `binary=b4be7c17a652510620a753882f7b5c4f2e9e4e63b09866928e5549b6917fd70b`

* **Fixture**: [testdata/bench/task_11_mixed_sink_api_migration.txtar](https://github.com/spockz/semantic-editor/blob/2cc7f05ca8384db81d0fdf5daacbc587add75845/testdata/bench/task_11_mixed_sink_api_migration.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | — | — | — | 153.11s | 26.05s | N/A |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | — | — | — | 1 | 0 | N/A |
| **Internal Tool Cycles** | — | — | — | 0 | 0 | 0% |
| **Initial Load / Discovery Turns** | — | — | — | 0 | 0 | 0% |
| **MCP Discovery / Schema Turns** | — | — | — | 0 | 0 | 0% |
| **Total Tool Invocations** | — | — | — | 0 | 0 | 0% |
| **Output Tokens** | — | — | — | 0 | 0 | 0% |
| **Reasoning / Thinking Tokens** | — | — | — | 0 | 0 | 0% |
| **Total Input Tokens** | — | — | — | 0 | 0 | 0% |
| **Cached Input Tokens** | — | — | — | 0 | 0 | 0% |
| **Uncached Input Tokens** | — | — | — | 0 | 0 | 0% |
| **Cached vs Uncached Token Ratio** | — | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | — | — | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

