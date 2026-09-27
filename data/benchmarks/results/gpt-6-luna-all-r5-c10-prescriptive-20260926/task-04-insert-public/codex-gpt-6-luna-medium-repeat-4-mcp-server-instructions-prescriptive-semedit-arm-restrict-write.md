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

* **Date**: 2026-09-26 23:14:21 CEST

## Test case: `task-04-insert-public`

### Target: `codex/gpt-6-luna/medium (repeat 4)`

#### Configuration: default prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/task_04_insert_public.txtar](https://github.com/spockz/semantic-editor/blob/c03f3625ef877adfd183a978ca333ca208eff0f0/testdata/bench/task_04_insert_public.txtar)

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 25.33s | 62.27s | <span class="benchmark-delta-negative">+145.8%</span> | 24.37s | 42.12s | <span class="benchmark-delta-negative">+72.9%</span> |
| **Process Start → First Event** | 1.63s | 1.78s | — | 1.10s | 0.68s | — |
| **First Event → First Tool Call** | 9.95s | 10.04s | — | 6.90s | 8.22s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 4 | 6 | <span class="benchmark-delta-negative">+50.0%</span> | 2 | 4 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | 2 | 2 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 4 | 6 | <span class="benchmark-delta-negative">+50.0%</span> | 2 | 4 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Output Tokens** | 482 | 648 | <span class="benchmark-delta-negative">+34.4%</span> | 360 | 441 | <span class="benchmark-delta-negative">+22.5%</span> |
| **Reasoning / Thinking Tokens** | 86 | 237 | <span class="benchmark-delta-negative">+175.6%</span> | 84 | 120 | <span class="benchmark-delta-negative">+42.9%</span> |
| **Total Input Tokens** | 95751 | 189216 | <span class="benchmark-delta-negative">+97.6%</span> | 76472 | 138965 | <span class="benchmark-delta-negative">+81.7%</span> |
| **Cached Input Tokens** | 87808 | 167936 | <span class="benchmark-delta-positive">+91.3%</span> | 64512 | 122368 | <span class="benchmark-delta-positive">+89.7%</span> |
| **Uncached Input Tokens** | 7943 | 21280 | <span class="benchmark-delta-negative">+167.9%</span> | 11960 | 16597 | <span class="benchmark-delta-negative">+38.8%</span> |
| **Cached vs Uncached Token Ratio** | 11.05:1 | 7.89:1 | <span class="benchmark-delta-negative">-28.6%</span> | 5.39:1 | 7.37:1 | <span class="benchmark-delta-positive">+36.7%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/server.go modified (9 lines -> 13 lines)
* **MCP Edit**: File api/server.go modified (9 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/server.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;api/server.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;include_unexported&#34;: true
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 -c &#39;from pathlib import Path; import os, tempfile, time; p=Path(\&#34;api/server.go\&#34;); old=p.read_text(); marker=\&#34;func (s *Server) internalRun() {}\\n\&#34;; assert marker in old; new=old.replace(marker, \&#34;func InitServer() *Server {\\n\\treturn &amp;Server{}\\n}\\n\\n\&#34;+marker, 1); fd, tmp=tempfile.mkstemp(dir=str(p.parent), prefix=\&#34;.server.go.\&#34;); f=os.fdopen(fd,\&#34;w\&#34;); f.write(new); f.flush(); os.fsync(f.fileno()); os.replace(tmp,p); os.utime(p, ns=(time.time_ns(), time.time_ns()))&#39; &amp;&amp; \
git diff -- api/server.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;symbol&#34;: &#34;Server&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/server.go; git status --short --untracked-files=no;&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;symbol&#34;: &#34;Server.Start&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_end&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;api/server.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;include_unexported&#34;: true
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/server.go modified (9 lines -> 13 lines)
* **MCP Edit**: File api/server.go modified (9 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; | head -80&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*server*.go&#39; -g &#39;*Server*.go&#39; -g &#39;README.md&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; api/server.go &amp;&amp; \
sed -n &#39;1,160p&#39; api/server_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;ls -la &amp;&amp; \
cat api/server.go&#39;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_end&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/server.go &amp;&amp; \
go build ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because the operations were sequentially dependent: I needed the outline and symbol inspection to determine the constructor’s placement and body before inserting it. The batch tool was available, but batching these discovery steps with the edit was unsuitable.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

