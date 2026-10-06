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

* **Date**: 2026-10-06 08:51:08 CEST

## Test case: `task-05-insert-method`

### Target: `codex/gpt-6-luna/medium (repeat 15)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_05_insert_method.txtar` (`sha256:59f934b113c870cd0f9fa4bc (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add a new method func (s *Server) Stop() placed immediately after Server.Start. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Call semantic_lookup with {\"file\":\"api/server.go\",\"symbol\":\"Server.Start\"}. Then call semantic_insert_function with {\"file\":\"api/server.go\",\"source\":\"func (s *Server) Stop() { s.running = false }\",\"placement\":\"after_symbol\",\"target_symbol\":\"Server.Start\"}. Do not alter Start or unrelated code. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct {
	running bool
}

func (s *Server) Start() {
	s.running = true
}

func (s *Server) IsRunning() bool {
	return s.running
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 13.86s | 30.24s | <span class="benchmark-delta-negative">+118.2%</span> | 26.20s | 32.26s | <span class="benchmark-delta-negative">+23.1%</span> |
| **Process Start → First Event** | 0.15s | 0.16s | — | 0.21s | 0.17s | — |
| **First Event → First Tool Call** | 4.87s | 8.90s | — | 24.54s | 11.09s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Initial Load / Discovery Turns** | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 1 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Output Tokens** | 257 | 708 | <span class="benchmark-delta-negative">+175.5%</span> | 0 | 563 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Reasoning / Thinking Tokens** | 28 | 309 | <span class="benchmark-delta-negative">+1003.6%</span> | 0 | 165 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Input Tokens** | 78107 | 172737 | <span class="benchmark-delta-negative">+121.2%</span> | 0 | 189666 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Cached Input Tokens** | 71680 | 157696 | <span class="benchmark-delta-positive">+120.0%</span> | 0 | 157952 | <span class="benchmark-delta-positive">+100.0%</span> |
| **Uncached Input Tokens** | 6427 | 15041 | <span class="benchmark-delta-negative">+134.0%</span> | 0 | 31714 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Cached vs Uncached Token Ratio** | 11.15:1 | 10.48:1 | <span class="benchmark-delta-negative">-6.0%</span> | — | 4.98:1 | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/server.go modified (13 lines -> 17 lines)
* **MCP Edit**: File api/server.go modified (13 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func \\(s \\*Server\\) Start|type Server struct\&#34; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;symbol&#34;: &#34;Server.Start&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/server.go&#39;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-05-insert-method_1791262625765427000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;api/server.go&#34;,
        &#34;source&#34;: &#34;func (s *Server) Stop() { s.running = false }&#34;,
        &#34;placement&#34;: &#34;after_symbol&#34;,
        &#34;target_symbol&#34;: &#34;Server.Start&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **MCP Edit**: File api/server.go modified (13 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; | head -100&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;symbol&#34;: &#34;Server.Start&#34;
}</code></pre></td></tr>
<tr><td>2</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;kind&#34;: &#34;method&#34;,
  &#34;source&#34;: &#34;func (s *Server) Stop() { s.running = false }&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Server.Start&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools: the semantic lookup and insertion let me place `Stop` precisely with no unrelated edits, which made this small change feel reassuringly controlled.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because the requested `semantic_insert_function` tool was unavailable, and I treated `semantic_insert_construct` as its equivalent after the lookup. Batching could have run the lookup and insertion in order, but I didn’t choose it; that was my decision, not a tooling limitation.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>This run was reassuringly direct: semedit located `Server.Start` and inserted `Server.Stop` while resolving an existing diagnostic, so I’d ask my owner to add its tools.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_05_insert_method.txtar` (`sha256:59f934b113c870cd0f9fa4bc (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add a new method func (s *Server) Stop() placed immediately after Server.Start. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add a new method func (s *Server) Stop() placed immediately after Server.Start. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct {
	running bool
}

func (s *Server) Start() {
	s.running = true
}

func (s *Server) IsRunning() bool {
	return s.running
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 13.86s | 28.17s | <span class="benchmark-delta-negative">+103.3%</span> | 26.20s | 43.58s | <span class="benchmark-delta-negative">+66.3%</span> |
| **Process Start → First Event** | 0.15s | 0.16s | — | 0.21s | 0.13s | — |
| **First Event → First Tool Call** | 4.87s | 7.74s | — | 24.54s | 12.88s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | 1 | 4 | <span class="benchmark-delta-negative">+300.0%</span> |
| **Initial Load / Discovery Turns** | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 1 | 3 | <span class="benchmark-delta-negative">+200.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | 1 | 4 | <span class="benchmark-delta-negative">+300.0%</span> |
| **Output Tokens** | 257 | 325 | <span class="benchmark-delta-negative">+26.5%</span> | 0 | 706 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Reasoning / Thinking Tokens** | 28 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 0 | 89 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Input Tokens** | 78107 | 104374 | <span class="benchmark-delta-negative">+33.6%</span> | 0 | 221861 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Cached Input Tokens** | 71680 | 88064 | <span class="benchmark-delta-positive">+22.9%</span> | 0 | 202496 | <span class="benchmark-delta-positive">+100.0%</span> |
| **Uncached Input Tokens** | 6427 | 16310 | <span class="benchmark-delta-negative">+153.8%</span> | 0 | 19365 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Cached vs Uncached Token Ratio** | 11.15:1 | 5.40:1 | <span class="benchmark-delta-negative">-51.6%</span> | — | 10.46:1 | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/server.go modified (13 lines -> 17 lines)
* **MCP Edit**: File api/server.go modified (13 lines -> 16 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func \\(s \\*Server\\) Start|type Server struct\&#34; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Server.Start&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-05-insert-method_1791262634530586000&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/server.go&#39;</code></pre></td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;kind&#34;: &#34;method&#34;,
  &#34;source&#34;: &#34;func (s *Server) Stop() {\n}&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Server.Start&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-05-insert-method_1791262634530586000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **MCP Edit**: File api/server.go modified (13 lines -> 17 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; | head -100&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
rg -n &#39;func \\(s \\*Server\\) Start|type Server&#39; api/server.go&#34;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/server.go&#39;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-05-insert-method_1791262801640980000&#34;,
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;kind&#34;: &#34;method&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;(*Server).Start&#34;,
  &#34;source&#34;: &#34;func (s *Server) Stop() {\n\ts.running = false\n}&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was not needed because the task required only one mutation: inserting `Server.Stop` after `Server.Start`. I used `semantic_lookup` first to locate the anchor, then a single `semantic_insert_construct` call; there were no multiple operations to combine with `semantic_batch`.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>In this run, semedit let me locate `Server.Start` and insert `Server.Stop` in the right place with a single checked semantic edit, which was reassuringly precise; I’d ask my owner to add its tools for coding tasks like this.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit let me add `Server.Stop` exactly after `Start` and confirm it resolved an existing undefined-method diagnostic, so I’d ask my owner to add its tools for coding tasks that need precise source edits.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

