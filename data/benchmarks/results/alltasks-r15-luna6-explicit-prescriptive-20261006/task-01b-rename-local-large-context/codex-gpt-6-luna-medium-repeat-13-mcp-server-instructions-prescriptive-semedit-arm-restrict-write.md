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

* **Date**: 2026-10-06 08:51:10 CEST

## Test case: `task-01b-rename-local-large-context`

### Target: `codex/gpt-6-luna/medium (repeat 13)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_01b_rename_local_large_context.txtar` (`sha256:ca97ab78ffff6b89538762bc (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename the unexported method Server.oldName to Server.newName in api/server.go and update all call-sites within that package. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Call semantic_lookup for Server.oldName in api/server.go, then call semantic_rename with the returned file and symbol Server.oldName and to= newName. Let the rename update package call sites; do not edit unrelated files. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | — | — | — | 29.22s | 25.88s | <span class="benchmark-delta-positive">-11.4%</span> |
| **Process Start → First Event** | — | — | — | 0.13s | 0.29s | — |
| **First Event → First Tool Call** | — | — | — | 5.77s | 6.81s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | — | — | — | 1 | 1 | 0% |
| **Internal Tool Cycles** | — | — | — | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> |
| **Initial Load / Discovery Turns** | — | — | — | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **MCP Discovery / Schema Turns** | — | — | — | 0 | 0 | 0% |
| **Total Tool Invocations** | — | — | — | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> |
| **Output Tokens** | — | — | — | 790 | 246 | <span class="benchmark-delta-positive">-68.9%</span> |
| **Reasoning / Thinking Tokens** | — | — | — | 91 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **Total Input Tokens** | — | — | — | 121599 | 86508 | <span class="benchmark-delta-positive">-28.9%</span> |
| **Cached Input Tokens** | — | — | — | 112128 | 75776 | <span class="benchmark-delta-negative">-32.4%</span> |
| **Uncached Input Tokens** | — | — | — | 9471 | 10732 | <span class="benchmark-delta-negative">+13.3%</span> |
| **Cached vs Uncached Token Ratio** | — | — | — | 11.84:1 | 7.06:1 | <span class="benchmark-delta-negative">-40.4%</span> |
| **Oracle L1: Mutation Policy** | — | — | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | — | — | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | — | — | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | — | — | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | — | — | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/server.go modified (12 lines -> 12 lines)
* **MCP Edit**: File api/server.go modified (12 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;oldName|newName\&#34; api/server.go api --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Server.oldName&#34;,
  &#34;file&#34;: &#34;api/server.go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; api/server.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;symbol&#34;: &#34;Server.oldName&#34;,
  &#34;to&#34;: &#34;newName&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_large-explicit_task-01b-rename-local-large-context_1791261321573653000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;oldName|newName\&#34; api --glob &#39;*.go&#39; &amp;&amp; \
git diff -- api/server.go&#34;</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I didn’t combine them because the task explicitly required calling `semantic_lookup` first and then using its returned file and symbol in `semantic_rename`. The rename depended on the lookup result, so the operations were sequentially dependent rather than independent batch steps.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit’s lookup and rename tools handled this method rename and its call sites cleanly, which was reassuring; based on this task, I’d ask my owner to add them.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_01b_rename_local_large_context.txtar` (`sha256:ca97ab78ffff6b89538762bc (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename the unexported method Server.oldName to Server.newName in api/server.go and update all call-sites within that package. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename the unexported method Server.oldName to Server.newName in api/server.go and update all call-sites within that package. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | — | — | — | 29.22s | 35.66s | <span class="benchmark-delta-negative">+22.1%</span> |
| **Process Start → First Event** | — | — | — | 0.13s | 0.22s | — |
| **First Event → First Tool Call** | — | — | — | 5.77s | 5.49s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | — | — | — | 1 | 1 | 0% |
| **Internal Tool Cycles** | — | — | — | 4 | 4 | 0% |
| **Initial Load / Discovery Turns** | — | — | — | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> |
| **MCP Discovery / Schema Turns** | — | — | — | 0 | 0 | 0% |
| **Total Tool Invocations** | — | — | — | 4 | 4 | 0% |
| **Output Tokens** | — | — | — | 790 | 417 | <span class="benchmark-delta-positive">-47.2%</span> |
| **Reasoning / Thinking Tokens** | — | — | — | 91 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **Total Input Tokens** | — | — | — | 121599 | 119297 | <span class="benchmark-delta-positive">-1.9%</span> |
| **Cached Input Tokens** | — | — | — | 112128 | 105216 | <span class="benchmark-delta-negative">-6.2%</span> |
| **Uncached Input Tokens** | — | — | — | 9471 | 14081 | <span class="benchmark-delta-negative">+48.7%</span> |
| **Cached vs Uncached Token Ratio** | — | — | — | 11.84:1 | 7.47:1 | <span class="benchmark-delta-negative">-36.9%</span> |
| **Oracle L1: Mutation Policy** | — | — | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | — | — | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | — | — | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | — | — | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | — | — | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/server.go modified (12 lines -> 12 lines)
* **MCP Edit**: File api/server.go modified (12 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;oldName|newName\&#34; api/server.go api --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd &amp;&amp; \
rg -n &#34;oldName|newName&#34; api&#39;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; api/server.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Server.oldName&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-01b-rename-local-large-context_1791261316408105000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Server.oldName&#34;,
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;to&#34;: &#34;newName&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-01b-rename-local-large-context_1791261316408105000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;oldName|newName\&#34; api --glob &#39;*.go&#39; &amp;&amp; \
git diff -- api/server.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- api/server.go &amp;&amp; \
git status --short &amp;&amp; \
rg -n &#34;oldName|newName&#34; api/server.go&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was unsuitable because the task required one semantic operation: renaming a method and its references. I used lookup first to resolve the symbol, then `semantic_rename`; lookup does not modify source, so there were no multiple mutations to combine with `semantic_batch`.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit’s semantic rename made this small refactor feel reassuringly precise by updating the method and its call site together, so I’d ask my owner to add its tools.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

