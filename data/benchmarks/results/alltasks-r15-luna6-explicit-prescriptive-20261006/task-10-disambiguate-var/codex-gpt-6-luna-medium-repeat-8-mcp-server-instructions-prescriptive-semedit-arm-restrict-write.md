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

* **Date**: 2026-10-06 08:51:11 CEST

## Test case: `task-10-disambiguate-var`

### Target: `codex/gpt-6-luna/medium (repeat 8)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_10_disambiguate_var.txtar` (`sha256:efaa19751026d7c4826c1250 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In handler/handler.go, disambiguate the inner shadowing variable ctx inside HandleRequest so it uses reqCtx without shadowing the outer context package or parameter. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Inspect HandleRequest in handler/handler.go, locate only the inner shadowing ctx variable, and call semantic_rename on that local symbol with to= reqCtx. Preserve the outer context parameter/package binding and make no other changes. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package handler

import (
	"context"
)

func HandleRequest(parentCtx context.Context, action string) string {
	// Shadowed ctx in inner block
	ctx := action + "-processed"
	_ = parentCtx
	return ctx
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 32.47s | 45.91s | <span class="benchmark-delta-negative">+41.4%</span> | 37.36s | 28.65s | <span class="benchmark-delta-positive">-23.3%</span> |
| **Process Start → First Event** | 0.14s | 0.30s | — | 0.13s | 0.14s | — |
| **First Event → First Tool Call** | 9.88s | 5.48s | — | 9.25s | 4.62s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 2 | 6 | <span class="benchmark-delta-negative">+200.0%</span> | 2 | 4 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 2 | 6 | <span class="benchmark-delta-negative">+200.0%</span> | 2 | 4 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Output Tokens** | 213 | 964 | <span class="benchmark-delta-negative">+352.6%</span> | 331 | 481 | <span class="benchmark-delta-negative">+45.3%</span> |
| **Reasoning / Thinking Tokens** | 39 | 364 | <span class="benchmark-delta-negative">+833.3%</span> | 0 | 77 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Input Tokens** | 58518 | 169764 | <span class="benchmark-delta-negative">+190.1%</span> | 58705 | 123849 | <span class="benchmark-delta-negative">+111.0%</span> |
| **Cached Input Tokens** | 52480 | 151040 | <span class="benchmark-delta-positive">+187.8%</span> | 52480 | 108288 | <span class="benchmark-delta-positive">+106.3%</span> |
| **Uncached Input Tokens** | 6038 | 18724 | <span class="benchmark-delta-negative">+210.1%</span> | 6225 | 15561 | <span class="benchmark-delta-negative">+150.0%</span> |
| **Cached vs Uncached Token Ratio** | 8.69:1 | 8.07:1 | <span class="benchmark-delta-negative">-7.2%</span> | 8.43:1 | 6.96:1 | <span class="benchmark-delta-negative">-17.5%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File handler/handler.go modified (12 lines -> 11 lines)
* **MCP Edit**: File handler/handler.go modified (12 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; handler/handler.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd &amp;&amp; \
rg -n &#34;func HandleRequest|HandleRequest&#34; handler/handler.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;HandleRequest&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-10-disambiguate-var_1791265010633234000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;HandleRequest&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-10-disambiguate-var_1791265010633234000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;symbol&#34;: &#34;ctx&#34;,
  &#34;to&#34;: &#34;reqCtx&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-10-disambiguate-var_1791265010633234000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- handler/handler.go&#39;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;func HandleRequest|reqCtx|ctx|parentCtx&#34; handler/handler.go &amp;&amp; \
git status --short -- handler/handler.go&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File handler/handler.go modified (12 lines -> 12 lines)
* **MCP Edit**: File handler/handler.go modified (12 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; handler/handler.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; handler/handler.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;HandleRequest&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ctx&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_large-explicit_task-10-disambiguate-var_1791265211247675000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ctx&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;to&#34;: &#34;reqCtx&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_large-explicit_task-10-disambiguate-var_1791265211247675000&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I used `semantic_lookup` and `semantic_inspect_symbol` as independent read-only lookups, then called `semantic_rename` once the target was identified. Batching wasn’t needed: there was only one mutation, and the lookups could run in parallel.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit’s semantic rename let me target the local `ctx` confidently while preserving the surrounding bindings, and I’d ask my owner to add its tools for coding tasks like this.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was unsuitable because the two semantic operations were dependent: `semantic_lookup` had to identify the specific inner `ctx` before `semantic_rename` could target it. The rename was therefore a single mutation performed after the lookup.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit’s lookup pinpointed the shadowing local and its rename changed only that symbol, which felt reassuringly precise; based on this task, I’d ask my owner to add its tools.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_10_disambiguate_var.txtar` (`sha256:efaa19751026d7c4826c1250 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In handler/handler.go, disambiguate the inner shadowing variable ctx inside HandleRequest so it uses reqCtx without shadowing the outer context package or parameter. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In handler/handler.go, disambiguate the inner shadowing variable ctx inside HandleRequest so it uses reqCtx without shadowing the outer context package or parameter. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package handler

import (
	"context"
)

func HandleRequest(parentCtx context.Context, action string) string {
	// Shadowed ctx in inner block
	ctx := action + "-processed"
	_ = parentCtx
	return ctx
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 32.47s | 30.23s | <span class="benchmark-delta-positive">-6.9%</span> | 37.36s | 21.85s | <span class="benchmark-delta-positive">-41.5%</span> |
| **Process Start → First Event** | 0.14s | 0.35s | — | 0.13s | 0.14s | — |
| **First Event → First Tool Call** | 9.88s | 7.19s | — | 9.25s | 5.35s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 2 | 3 | <span class="benchmark-delta-negative">+50.0%</span> | 2 | 5 | <span class="benchmark-delta-negative">+150.0%</span> |
| **Initial Load / Discovery Turns** | 1 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 2 | 3 | <span class="benchmark-delta-negative">+50.0%</span> | 2 | 5 | <span class="benchmark-delta-negative">+150.0%</span> |
| **Output Tokens** | 213 | 392 | <span class="benchmark-delta-negative">+84.0%</span> | 331 | 329 | <span class="benchmark-delta-positive">-0.6%</span> |
| **Reasoning / Thinking Tokens** | 39 | 77 | <span class="benchmark-delta-negative">+97.4%</span> | 0 | 34 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Input Tokens** | 58518 | 197565 | <span class="benchmark-delta-negative">+237.6%</span> | 58705 | 93621 | <span class="benchmark-delta-negative">+59.5%</span> |
| **Cached Input Tokens** | 52480 | 177408 | <span class="benchmark-delta-positive">+238.0%</span> | 52480 | 75776 | <span class="benchmark-delta-positive">+44.4%</span> |
| **Uncached Input Tokens** | 6038 | 20157 | <span class="benchmark-delta-negative">+233.8%</span> | 6225 | 17845 | <span class="benchmark-delta-negative">+186.7%</span> |
| **Cached vs Uncached Token Ratio** | 8.69:1 | 8.80:1 | <span class="benchmark-delta-positive">+1.3%</span> | 8.43:1 | 4.25:1 | <span class="benchmark-delta-negative">-49.6%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File handler/handler.go modified (12 lines -> 11 lines)
* **MCP Edit**: File handler/handler.go modified (12 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; handler/handler.go&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;HandleRequest&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;HandleRequest&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ctx&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;to&#34;: &#34;reqCtx&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File handler/handler.go modified (12 lines -> 12 lines)
* **MCP Edit**: File handler/handler.go modified (12 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; handler/handler.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd &amp;&amp; \
rg -n &#34;func .*HandleRequest|ctx&#34; handler/handler.go&#39;</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc pwd</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ctx&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-10-disambiguate-var_1791265200920417000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc pwd</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ctx&#34;,
  &#34;file&#34;: &#34;handler/handler.go&#34;,
  &#34;to&#34;: &#34;reqCtx&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-10-disambiguate-var_1791265200920417000&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">The operations were unsuitable for `semantic_batch`: lookup and symbol inspection were needed first to identify the exact declaration, and only then could I safely rename it. I used a single semantic rename for the change.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit’s symbol-aware rename made the small change feel precise and gave me confidence it updated the right references, so I’d ask my owner to add its tools.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools: semantic lookup pinpointed the shadowing variable and semantic rename changed it to `reqCtx` cleanly, which made this small fix feel precise and dependable.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

