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

* **Date**: 2026-10-06 22:13:38 CEST

## Test case: `grpc__grpc-go-2735`

### Target: `codex/gpt-5.6-luna/medium (repeat 1)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/grpc__grpc-go-2735.txtar](https://github.com/spockz/semantic-editor/blob/43bf7788d286eda6cb43f908eecb81fc2ff8694e/testdata/bench/grpc__grpc-go-2735.txtar)

**LLM Prompt**:

> ### Use case(s) - what problem will this feature solve?
>
> In some cases we want to retry connections more aggressively compared to the default behavior (e.g. for health checkers in tests in order to detect that server is up faster).
>
> ### Proposed Solution
>
> Add tunable parameters as described in the [documentation](https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md).
>
> It can be done by extending `grpc.BackoffConfig` struct (and therefore is not a breaking change).
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 0.00s | 0.00s | — | — | — | — |
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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 0.00s | 0.00s | — | — | — | — |
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
