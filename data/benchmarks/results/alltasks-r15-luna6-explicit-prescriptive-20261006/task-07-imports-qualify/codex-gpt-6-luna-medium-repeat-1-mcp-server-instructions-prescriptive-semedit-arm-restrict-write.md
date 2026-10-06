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

## Test case: `task-07-imports-qualify`

### Target: `codex/gpt-6-luna/medium (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_07_imports_qualify.txtar` (`sha256:89605b7218ae79ee5e062a11 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Import the local pkg/mockuuid package into api/id.go and use mockuuid.NewString() inside GenerateID(). Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Call semantic_lookup for GenerateID in api/id.go, inspect that function, then call semantic_replace_body for GenerateID with body `return mockuuid.NewString()` and auto_organize_imports=true. Ensure the import is example.com/test/pkg/mockuuid; make no unrelated edits. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

func GenerateID() string {
	return ""
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 14.54s | 37.61s | <span class="benchmark-delta-negative">+158.6%</span> | 16.70s | 60.17s | <span class="benchmark-delta-negative">+260.4%</span> |
| **Process Start → First Event** | 0.17s | 0.17s | — | 0.13s | 0.14s | — |
| **First Event → First Tool Call** | 5.46s | 9.99s | — | 5.29s | 6.92s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 3 | 0% | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> |
| **Initial Load / Discovery Turns** | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 3 | 0% | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> |
| **Output Tokens** | 216 | 691 | <span class="benchmark-delta-negative">+219.9%</span> | 439 | 1081 | <span class="benchmark-delta-negative">+146.2%</span> |
| **Reasoning / Thinking Tokens** | 9 | 267 | <span class="benchmark-delta-negative">+2866.7%</span> | 0 | 421 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Input Tokens** | 80159 | 267260 | <span class="benchmark-delta-negative">+233.4%</span> | 78569 | 333469 | <span class="benchmark-delta-negative">+324.4%</span> |
| **Cached Input Tokens** | 61440 | 241152 | <span class="benchmark-delta-positive">+292.5%</span> | 71680 | 304384 | <span class="benchmark-delta-positive">+324.6%</span> |
| **Uncached Input Tokens** | 18719 | 26108 | <span class="benchmark-delta-negative">+39.5%</span> | 6889 | 29085 | <span class="benchmark-delta-negative">+322.2%</span> |
| **Cached vs Uncached Token Ratio** | 3.28:1 | 9.24:1 | <span class="benchmark-delta-positive">+181.4%</span> | 10.40:1 | 10.47:1 | <span class="benchmark-delta-positive">+0.6%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ❌ FAIL | ❌ FAIL | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ❌ FAIL | ❌ FAIL | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/id.go modified (5 lines -> 7 lines)
* **MCP Edit**: File api/id.go modified (5 lines -> 5 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; api/id.go &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;GenerateID&#34;,
  &#34;file&#34;: &#34;api/id.go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files pkg/mockuuid&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;GenerateID&#34;,
  &#34;file&#34;: &#34;api/id.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;api/id.go&#34;,
        &#34;symbol&#34;: &#34;GenerateID&#34;,
        &#34;body&#34;: &#34;return mockuuid.NewString()&#34;,
        &#34;auto_organize_imports&#34;: true,
        &#34;expected_revision&#34;: &#34;70ea4ac7adff2becdc68d04abcf08dfa08c7961ff0a436b8cdeefe391a045624&#34;
      }
    }
  ]
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/id.go modified (5 lines -> 7 lines)
* **MCP Edit**: File api/id.go modified (5 lines -> 7 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; api/id.go &amp;&amp; \
rg --files | rg &#39;(&#34;&#39;^|/)pkg/mockuuid/&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;api/id.go&#39; -g &#39;AGENTS*.md&#39; -g &#39;go.mod&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; pkg/mockuuid/uuid.go; rg -n &#39;&#34;&#39;^module &#39;&#34;&#39; go.mod&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;GenerateID&#34;,
  &#34;file&#34;: &#34;api/id.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;GenerateID&#34;,
  &#34;file&#34;: &#34;api/id.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;api/id.go&#34;,
        &#34;symbol&#34;: &#34;GenerateID&#34;,
        &#34;body&#34;: &#34;return mockuuid.NewString()&#34;,
        &#34;auto_organize_imports&#34;: true,
        &#34;expected_revision&#34;: &#34;70ea4ac7adff2becdc68d04abcf08dfa08c7961ff0a436b8cdeefe391a045624&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;api/id.go&#34;,
        &#34;source&#34;: &#34;import mockuuid \&#34;example.com/test/pkg/mockuuid\&#34;&#34;,
        &#34;placement&#34;: &#34;file_start&#34;
      }
    }
  ]
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools: they let me replace a function body structurally and resolve its import with diagnostics, though I had to add the import separately when automatic organization left it unresolved.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_07_imports_qualify.txtar` (`sha256:89605b7218ae79ee5e062a11 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Import the local pkg/mockuuid package into api/id.go and use mockuuid.NewString() inside GenerateID(). Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Import the local pkg/mockuuid package into api/id.go and use mockuuid.NewString() inside GenerateID(). Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

func GenerateID() string {
	return ""
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 14.54s | 34.57s | <span class="benchmark-delta-negative">+137.7%</span> | 16.70s | 55.13s | <span class="benchmark-delta-negative">+230.2%</span> |
| **Process Start → First Event** | 0.17s | 0.13s | — | 0.13s | 0.13s | — |
| **First Event → First Tool Call** | 5.46s | 5.66s | — | 5.29s | 3.33s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 4 | <span class="benchmark-delta-negative">+33.3%</span> | 3 | 11 | <span class="benchmark-delta-negative">+266.7%</span> |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | 2 | 2 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 4 | <span class="benchmark-delta-negative">+33.3%</span> | 3 | 11 | <span class="benchmark-delta-negative">+266.7%</span> |
| **Output Tokens** | 216 | 534 | <span class="benchmark-delta-negative">+147.2%</span> | 439 | 1639 | <span class="benchmark-delta-negative">+273.3%</span> |
| **Reasoning / Thinking Tokens** | 9 | 146 | <span class="benchmark-delta-negative">+1522.2%</span> | 0 | 750 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Input Tokens** | 80159 | 165392 | <span class="benchmark-delta-negative">+106.3%</span> | 78569 | 420829 | <span class="benchmark-delta-negative">+435.6%</span> |
| **Cached Input Tokens** | 61440 | 145920 | <span class="benchmark-delta-positive">+137.5%</span> | 71680 | 392704 | <span class="benchmark-delta-positive">+447.9%</span> |
| **Uncached Input Tokens** | 18719 | 19472 | <span class="benchmark-delta-negative">+4.0%</span> | 6889 | 28125 | <span class="benchmark-delta-negative">+308.3%</span> |
| **Cached vs Uncached Token Ratio** | 3.28:1 | 7.49:1 | <span class="benchmark-delta-positive">+128.3%</span> | 10.40:1 | 13.96:1 | <span class="benchmark-delta-positive">+34.2%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ❌ FAIL | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ❌ FAIL | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/id.go modified (5 lines -> 7 lines)
* **MCP Edit**: File api/id.go modified (5 lines -> 7 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; api/id.go &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;api/id.go&#39; -g &#39;docs/SUBOPTIMAL_TOOLS.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files pkg/mockuuid&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/id.go&#34;,
  &#34;symbol&#34;: &#34;GenerateID&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/id.go&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;in_function&#34;: &#34;GenerateID&#34;,
  &#34;discriminator&#34;: &#34;return \&#34;\&#34;&#34;,
  &#34;source&#34;: &#34;return mockuuid.NewString()&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/id.go&#34;,
  &#34;symbol&#34;: &#34;GenerateID&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/id.go modified (5 lines -> 7 lines)
* **MCP Edit**: File api/id.go modified (5 lines -> 7 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; api/id.go &amp;&amp; \
rg --files | rg &#39;(&#34;&#39;^|/)pkg/mockuuid/&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;api/id.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;SUBOPTIMAL_TOOLS.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; pkg/mockuuid/uuid.go; rg -n &#39;&#34;&#39;^module &#39;&#34;&#39; go.mod&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/id.go &amp;&amp; \
cat go.mod&#39;</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;GenerateID&#34;,
  &#34;file&#34;: &#34;api/id.go&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files pkg&#39;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;GenerateID&#34;,
  &#34;file&#34;: &#34;api/id.go&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;NewString&#34;,
  &#34;file&#34;: &#34;pkg/mockuuid/uuid.go&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;SUBOPTIMAL_TOOLS.md&#39; -g &#39;docs/**&#39; -g &#39;pkg/mockuuid/uuid.go&#39;&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;api/id.go&#34;,
        &#34;symbol&#34;: &#34;GenerateID&#34;,
        &#34;body&#34;: &#34;return mockuuid.NewString()&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;NewString&#34;,
  &#34;file&#34;: &#34;pkg/mockuuid/uuid.go&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/id.go&#34;,
  &#34;kind&#34;: &#34;decl&#34;,
  &#34;source&#34;: &#34;import \&#34;example.com/test/pkg/mockuuid\&#34;&#34;,
  &#34;placement&#34;: &#34;file_start&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- api/id.go go.mod go.sum &amp;&amp; \
cat api/id.go &amp;&amp; \
go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was unsuitable here: the first semantic operation inspected `GenerateID` to confirm its current body, and that result determined the exact replacement selector for the next operation. The import was handled automatically by the replacement operation, so no separate edit sequence needed batching.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools: I could replace the return expression and resolve the local import through a focused semantic edit, which felt reassuringly direct.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit’s semantic tools let me replace `GenerateID` precisely and then add the missing local import, which felt reassuring; based on this run, I’d ask my owner to add them.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

