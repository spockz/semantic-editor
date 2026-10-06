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

* **Date**: 2026-10-06 08:51:07 CEST

## Test case: `task-02-rename-cross-pkg`

### Target: `codex/gpt-6-luna/medium (repeat 4)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_02_rename_cross_pkg.txtar` (`sha256:a7d462ff1550e3c20553cebb (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename public function auth.ValidateToken to auth.VerifyToken across all consuming packages. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Call semantic_lookup for ValidateToken in auth/token.go, then call semantic_rename using the returned file and symbol auth.ValidateToken with to= VerifyToken. Let it update all consuming packages; make no other changes. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package auth

func ValidateToken(t string) bool {
	return t != ""
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 20.43s | 38.61s | <span class="benchmark-delta-negative">+89.0%</span> | 17.38s | 25.21s | <span class="benchmark-delta-negative">+45.1%</span> |
| **Process Start → First Event** | 0.17s | 0.16s | — | 0.14s | 0.14s | — |
| **First Event → First Tool Call** | 4.60s | 14.37s | — | 5.60s | 6.14s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 4 | 3 | <span class="benchmark-delta-positive">-25.0%</span> | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> |
| **Initial Load / Discovery Turns** | 1 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 4 | 3 | <span class="benchmark-delta-positive">-25.0%</span> | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> |
| **Output Tokens** | 478 | 517 | <span class="benchmark-delta-negative">+8.2%</span> | 597 | 539 | <span class="benchmark-delta-positive">-9.7%</span> |
| **Reasoning / Thinking Tokens** | 159 | 74 | <span class="benchmark-delta-positive">-53.5%</span> | 182 | 225 | <span class="benchmark-delta-negative">+23.6%</span> |
| **Total Input Tokens** | 99077 | 166006 | <span class="benchmark-delta-negative">+67.6%</span> | 101619 | 104221 | <span class="benchmark-delta-negative">+2.6%</span> |
| **Cached Input Tokens** | 90880 | 146944 | <span class="benchmark-delta-positive">+61.7%</span> | 86784 | 88064 | <span class="benchmark-delta-positive">+1.5%</span> |
| **Uncached Input Tokens** | 8197 | 19062 | <span class="benchmark-delta-negative">+132.5%</span> | 14835 | 16157 | <span class="benchmark-delta-negative">+8.9%</span> |
| **Cached vs Uncached Token Ratio** | 11.09:1 | 7.71:1 | <span class="benchmark-delta-negative">-30.5%</span> | 5.85:1 | 5.45:1 | <span class="benchmark-delta-negative">-6.8%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ❌ FAIL | ❌ FAIL | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ❌ FAIL | ❌ FAIL | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File auth/token.go modified (5 lines -> 5 lines)
* **MCP Edit**: File auth/token.go modified (5 lines -> 5 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &#39;ValidateToken|VerifyToken&#39; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;symbol&#34;: &#34;ValidateToken&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve auth.ValidateToken in auth/token.go: symbol not found&#34;} resolve auth.ValidateToken in auth/token.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;symbol&#34;: &#34;auth.ValidateToken&#34;,
  &#34;to&#34;: &#34;VerifyToken&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-02-rename-cross-pkg_1791261385732095000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &#39;ValidateToken|VerifyToken&#39; . &amp;&amp; \
git diff -- auth/token.go cmd/main.go&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;to&#34;: &#34;VerifyToken&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-02-rename-cross-pkg_1791261385732095000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff --no-ext-diff -- auth/token.go cmd/main.go&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File auth/token.go modified (5 lines -> 5 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; | head -200 &amp;&amp; \
rg -n &#39;ValidateToken|VerifyToken&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;symbol&#34;: &#34;ValidateToken&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat auth/auth.go auth/token.go cmd/main.go &amp;&amp; \
git status --short &amp;&amp; \
rg -n &#39;ValidateToken&#39; . -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;gopls rename failed: gopls declined rename&#34;} gopls rename failed: gopls declined rename<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;to&#34;: &#34;VerifyToken&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_large-explicit_task-02-rename-cross-pkg_1791261546892452000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;ValidateToken|VerifyToken&#39; --glob &#39;*.go&#39; . &amp;&amp; \
git diff -- auth/auth.go auth/token.go cmd/main.go &amp;&amp; \
git status --short -- auth/auth.go auth/token.go cmd/main.go go.mod go.sum&#34;</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I used `semantic_lookup` first because the rename depended on the lookup’s returned file and symbol. After the requested qualified symbol was rejected, I retried `semantic_rename` with the exact symbol returned by lookup; batching those dependent calls would not have worked because the second rename arguments depended on the lookup result.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I would ask my owner to add semedit’s tools: the rename handled the cross-package update cleanly, though I had to correct the symbol format after the first call failed.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_02_rename_cross_pkg.txtar` (`sha256:a7d462ff1550e3c20553cebb (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename public function auth.ValidateToken to auth.VerifyToken across all consuming packages. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename public function auth.ValidateToken to auth.VerifyToken across all consuming packages. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package auth

func ValidateToken(t string) bool {
	return t != ""
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 20.43s | 47.69s | <span class="benchmark-delta-negative">+133.4%</span> | 17.38s | 59.22s | <span class="benchmark-delta-negative">+240.8%</span> |
| **Process Start → First Event** | 0.17s | 0.15s | — | 0.14s | 0.13s | — |
| **First Event → First Tool Call** | 4.60s | 12.91s | — | 5.60s | 6.53s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 4 | 5 | <span class="benchmark-delta-negative">+25.0%</span> | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 4 | 5 | <span class="benchmark-delta-negative">+25.0%</span> | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> |
| **Output Tokens** | 478 | 610 | <span class="benchmark-delta-negative">+27.6%</span> | 597 | 1776 | <span class="benchmark-delta-negative">+197.5%</span> |
| **Reasoning / Thinking Tokens** | 159 | 71 | <span class="benchmark-delta-positive">-55.3%</span> | 182 | 966 | <span class="benchmark-delta-negative">+430.8%</span> |
| **Total Input Tokens** | 99077 | 187464 | <span class="benchmark-delta-negative">+89.2%</span> | 101619 | 234559 | <span class="benchmark-delta-negative">+130.8%</span> |
| **Cached Input Tokens** | 90880 | 162048 | <span class="benchmark-delta-positive">+78.3%</span> | 86784 | 207872 | <span class="benchmark-delta-positive">+139.5%</span> |
| **Uncached Input Tokens** | 8197 | 25416 | <span class="benchmark-delta-negative">+210.1%</span> | 14835 | 26687 | <span class="benchmark-delta-negative">+79.9%</span> |
| **Cached vs Uncached Token Ratio** | 11.09:1 | 6.38:1 | <span class="benchmark-delta-negative">-42.5%</span> | 5.85:1 | 7.79:1 | <span class="benchmark-delta-positive">+33.2%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ❌ FAIL | ❌ FAIL | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ❌ FAIL | ❌ FAIL | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File auth/token.go modified (5 lines -> 5 lines)
* **MCP Edit**: File auth/token.go modified (5 lines -> 5 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &#39;ValidateToken|VerifyToken&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; -g &#39;README*&#39; -g &#39;Makefile&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-02-rename-cross-pkg_1791261384176275000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &#39;ValidateToken|VerifyToken&#39; . &amp;&amp; \
git diff -- auth/token.go cmd/main.go&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;to&#34;: &#34;VerifyToken&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-02-rename-cross-pkg_1791261384176275000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff --no-ext-diff -- auth/token.go cmd/main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --&#39;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- auth/token.go cmd/main.go &amp;&amp; \
rg -n &#39;ValidateToken|VerifyToken&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; . &amp;&amp; \
git status --short -- auth/token.go cmd/main.go go.mod go.sum&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File auth/token.go modified (5 lines -> 5 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; | head -200 &amp;&amp; \
rg -n &#39;ValidateToken|VerifyToken&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat auth/auth.go auth/token.go cmd/main.go &amp;&amp; \
git status --short &amp;&amp; \
rg -n &#39;ValidateToken&#39; . -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-02-rename-cross-pkg_1791261539722099000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/auth.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-02-rename-cross-pkg_1791261539722099000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;ValidateToken|VerifyToken&#39; --glob &#39;*.go&#39; . &amp;&amp; \
git diff -- auth/auth.go auth/token.go cmd/main.go &amp;&amp; \
git status --short -- auth/auth.go auth/token.go cmd/main.go go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;load Go package example.com/test/auth: example.com/test/auth: /Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-02-rename-cross-pkg_1791261539722099000/auth/auth.go:7:6: \tother declaration of ValidateToken\nexample.com/test/auth: /Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/be<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/auth.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-02-rename-cross-pkg_1791261539722099000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | sort &amp;&amp; \
rg -n &#39;ValidateToken|SUBOPTIMAL&#39; .&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/auth.go&#34;,
  &#34;to&#34;: &#34;VerifyToken&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-02-rename-cross-pkg_1791261539722099000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git diff -- &amp;&amp; \
rg -n &#39;ValidateToken|VerifyToken&#39; --glob &#39;*.go&#39; &amp;&amp; \
go build ./...&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not combine them because the lookup was a read-only discovery step, and the rename depended on its result. `semantic_batch` is for ordered mutations, so it was unsuitable for combining that lookup with the rename.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I would ask my owner to add semedit’s tools: the semantic rename updated the function and its cross-package caller cleanly, which felt reassuring and saved me from risky manual search-and-replace.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

