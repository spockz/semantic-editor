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

### Target: `codex/gpt-6-luna/medium (repeat 13)`

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
| **Wall-Clock Latency** | 19.82s | 16.48s | <span class="benchmark-delta-positive">-16.8%</span> | 19.07s | 25.64s | <span class="benchmark-delta-negative">+34.5%</span> |
| **Process Start → First Event** | 0.14s | 0.17s | — | 0.13s | 0.15s | — |
| **First Event → First Tool Call** | 6.17s | 5.90s | — | 5.83s | 11.40s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> |
| **Initial Load / Discovery Turns** | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> |
| **Output Tokens** | 499 | 342 | <span class="benchmark-delta-positive">-31.5%</span> | 546 | 366 | <span class="benchmark-delta-positive">-33.0%</span> |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | 177 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **Total Input Tokens** | 78655 | 87343 | <span class="benchmark-delta-negative">+11.0%</span> | 79438 | 134441 | <span class="benchmark-delta-negative">+69.2%</span> |
| **Cached Input Tokens** | 66560 | 76800 | <span class="benchmark-delta-positive">+15.4%</span> | 71680 | 117504 | <span class="benchmark-delta-positive">+63.9%</span> |
| **Uncached Input Tokens** | 12095 | 10543 | <span class="benchmark-delta-positive">-12.8%</span> | 7758 | 16937 | <span class="benchmark-delta-negative">+118.3%</span> |
| **Cached vs Uncached Token Ratio** | 5.50:1 | 7.28:1 | <span class="benchmark-delta-positive">+32.4%</span> | 9.24:1 | 6.94:1 | <span class="benchmark-delta-negative">-24.9%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ❌ FAIL | — | ✅ PASS | ❌ FAIL | — |
| **Oracle L3: Clean Build** | ✅ PASS | ❌ FAIL | — | ❌ FAIL | ❌ FAIL | — |
| **Oracle L4: Verification Test** | ✅ PASS | ❌ FAIL | — | ❌ FAIL | ❌ FAIL | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File auth/token.go modified (5 lines -> 5 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; &#39;ValidateToken|VerifyToken&#39; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-02-rename-cross-pkg_1791261485126049000&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat auth/token.go &amp;&amp; \
cat cmd/main.go&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve auth.ValidateToken in auth/token.go: symbol not found&#34;} resolve auth.ValidateToken in auth/token.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;symbol&#34;: &#34;auth.ValidateToken&#34;,
  &#34;to&#34;: &#34;VerifyToken&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-02-rename-cross-pkg_1791261485126049000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File auth/token.go modified (5 lines -> 5 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | sort &amp;&amp; \
rg -n &#39;ValidateToken&#39; . --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
sed -n &#39;1,160p&#39; auth/auth.go &amp;&amp; \
sed -n &#39;1,160p&#39; auth/token.go &amp;&amp; \
sed -n &#39;1,160p&#39; cmd/main.go &amp;&amp; \
rg -n &#39;auth\\.|ValidateToken|VerifyToken&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve auth.ValidateToken in auth/token.go: symbol not found&#34;} resolve auth.ValidateToken in auth/token.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;symbol&#34;: &#34;auth.ValidateToken&#34;,
  &#34;to&#34;: &#34;VerifyToken&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_large-explicit_task-02-rename-cross-pkg_1791261651685974000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
</tbody>
</table>

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
| **Wall-Clock Latency** | 19.82s | 39.25s | <span class="benchmark-delta-negative">+98.1%</span> | 19.07s | 91.36s | <span class="benchmark-delta-negative">+379.0%</span> |
| **Process Start → First Event** | 0.14s | 0.22s | — | 0.13s | 0.13s | — |
| **First Event → First Tool Call** | 6.17s | 6.32s | — | 5.83s | 4.56s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 11 | <span class="benchmark-delta-negative">+266.7%</span> | 3 | 10 | <span class="benchmark-delta-negative">+233.3%</span> |
| **Initial Load / Discovery Turns** | 2 | 3 | <span class="benchmark-delta-negative">+50.0%</span> | 2 | 2 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 11 | <span class="benchmark-delta-negative">+266.7%</span> | 3 | 10 | <span class="benchmark-delta-negative">+233.3%</span> |
| **Output Tokens** | 499 | 560 | <span class="benchmark-delta-negative">+12.2%</span> | 546 | 1193 | <span class="benchmark-delta-negative">+118.5%</span> |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | 177 | 125 | <span class="benchmark-delta-positive">-29.4%</span> |
| **Total Input Tokens** | 78655 | 196326 | <span class="benchmark-delta-negative">+149.6%</span> | 79438 | 327489 | <span class="benchmark-delta-negative">+312.3%</span> |
| **Cached Input Tokens** | 66560 | 174336 | <span class="benchmark-delta-positive">+161.9%</span> | 71680 | 303872 | <span class="benchmark-delta-positive">+323.9%</span> |
| **Uncached Input Tokens** | 12095 | 21990 | <span class="benchmark-delta-negative">+81.8%</span> | 7758 | 23617 | <span class="benchmark-delta-negative">+204.4%</span> |
| **Cached vs Uncached Token Ratio** | 5.50:1 | 7.93:1 | <span class="benchmark-delta-positive">+44.1%</span> | 9.24:1 | 12.87:1 | <span class="benchmark-delta-positive">+39.3%</span> |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; &#39;ValidateToken|VerifyToken&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat auth/token.go &amp;&amp; \
cat cmd/main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat PROJECT_BRIEF.md 2&gt;/dev/null; rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;; rg -n &#39;ValidateToken|VerifyToken&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc pwd</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-02-rename-cross-pkg_1791261483130398000&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc pwd</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-02-rename-cross-pkg_1791261483130398000&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc pwd</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/token.go&#34;,
  &#34;to&#34;: &#34;VerifyToken&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-02-rename-cross-pkg_1791261483130398000&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc pwd</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;ValidateToken|VerifyToken&#39; --glob &#39;*.go&#39; . &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-02-rename-cross-pkg_1791261483130398000&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;check_only&#34;: true
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File auth/token.go modified (5 lines -> 5 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | sort &amp;&amp; \
rg -n &#39;ValidateToken&#39; . --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
sed -n &#39;1,160p&#39; auth/auth.go &amp;&amp; \
sed -n &#39;1,160p&#39; auth/token.go &amp;&amp; \
sed -n &#39;1,160p&#39; cmd/main.go &amp;&amp; \
rg -n &#39;auth\\.|ValidateToken|VerifyToken&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
rg -n \&#34;ValidateToken|VerifyToken\&#34; . -g &#39;*.go&#39; -g &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
rg --files&#34;</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve auth.ValidateToken: symbol not found&#34;} resolve auth.ValidateToken: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;auth.ValidateToken&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;mkdir -p docs &amp;&amp; \
cat &gt; docs/SUBOPTIMAL_TOOLS.md &lt;&lt;&#39;EOF&#39;
# Suboptimal tool behavior

- Tool: &#34;&#39;`semantic_lookup`
- Target: Go function `auth.ValidateToken` in `auth/auth.go`
- Observed failure: qualified lookup returned `symbol not found` even though the declaration exists.
- Workaround: look up the unqualified declaration `ValidateToken` within `auth/auth.go`, then use its semantic identity for rename.
- Root cause: the lookup backend does not resolve this package-qualified symbol in the fixture.
EOF&#39;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/auth.go&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;load Go package example.com/test/auth: example.com/test/auth: /Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-implicit_task-02-rename-cross-pkg_1791261648519022000/auth/auth.go:7:6: \tother declaration of ValidateToken\nexample.com/test/auth: /Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/be<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/auth.go&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat &gt;&gt; docs/SUBOPTIMAL_TOOLS.md &lt;&lt;&#39;EOF&#39;
- Tool: &#34;&#39;`semantic_find_references`
- Target: `ValidateToken` in `auth/auth.go`
- Observed failure: package loading fails because `auth/auth.go` and `auth/token.go` both declare `ValidateToken` in the same package.
- Workaround: proceed with the declaration-specific semantic rename if supported, then inspect the resulting diff and build diagnostics.
- Root cause: the fixture starts with an invalid duplicate declaration, preventing Go package analysis.
EOF&#39;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ValidateToken&#34;,
  &#34;file&#34;: &#34;auth/auth.go&#34;,
  &#34;to&#34;: &#34;VerifyToken&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- . &#39;:&#34;&#39;!go.mod&#39;&#34;&#39; &#39;:&#34;&#39;!go.sum&#39;&#34;&#39; &amp;&amp; \
go test ./...&#34;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;ValidateToken|VerifyToken\&#34; -g &#39;*.go&#39; -g &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools: the semantic rename carried `auth.ValidateToken` to `auth.VerifyToken` across packages cleanly, and the workspace verification gave me confidence the change was sound.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

