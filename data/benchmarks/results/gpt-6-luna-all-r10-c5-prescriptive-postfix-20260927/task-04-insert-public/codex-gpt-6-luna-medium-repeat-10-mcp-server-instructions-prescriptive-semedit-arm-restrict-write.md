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

* **Date**: 2026-09-27 03:12:39 CEST

## Test case: `task-04-insert-public`

### Target: `codex/gpt-6-luna/medium (repeat 10)`

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
| **Wall-Clock Latency** | 15.20s | 103.96s | <span class="benchmark-delta-negative">+584.0%</span> | 17.33s | 16.41s | <span class="benchmark-delta-positive">-5.3%</span> |
| **Process Start → First Event** | 0.14s | 0.13s | — | 0.12s | 0.12s | — |
| **First Event → First Tool Call** | 3.79s | 7.15s | — | 5.04s | 4.25s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> | 3 | 3 | 0% |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> | 3 | 3 | 0% |
| **Output Tokens** | 457 | 1342 | <span class="benchmark-delta-negative">+193.7%</span> | 399 | 289 | <span class="benchmark-delta-positive">-27.6%</span> |
| **Reasoning / Thinking Tokens** | 85 | 618 | <span class="benchmark-delta-negative">+627.1%</span> | 84 | 9 | <span class="benchmark-delta-positive">-89.3%</span> |
| **Total Input Tokens** | 75725 | 204512 | <span class="benchmark-delta-negative">+170.1%</span> | 96255 | 100123 | <span class="benchmark-delta-negative">+4.0%</span> |
| **Cached Input Tokens** | 66560 | 186112 | <span class="benchmark-delta-positive">+179.6%</span> | 85760 | 87808 | <span class="benchmark-delta-positive">+2.4%</span> |
| **Uncached Input Tokens** | 9165 | 18400 | <span class="benchmark-delta-negative">+100.8%</span> | 10495 | 12315 | <span class="benchmark-delta-negative">+17.3%</span> |
| **Cached vs Uncached Token Ratio** | 7.26:1 | 10.11:1 | <span class="benchmark-delta-positive">+39.3%</span> | 8.17:1 | 7.13:1 | <span class="benchmark-delta-negative">-12.7%</span> |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;AGENTS.md&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/server.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/server.go&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 -c &#39;from pathlib import Path; import os, tempfile, time
p=Path(\&#34;api/server.go\&#34;)
s=p.read_text()
old=\&#34;func (s *Server) Start() {}\\n\\nfunc (s *Server) internalRun() {}\\n\&#34;
new=\&#34;func (s *Server) Start() {}\\n\\nfunc InitServer() *Server {\\n\\treturn &amp;Server{}\\n}\\n\\nfunc (s *Server) internalRun() {}\\n\&#34;
if old not in s: raise SystemExit(\&#34;expected insertion point not found\&#34;)
fd, name=tempfile.mkstemp(dir=str(p.parent), prefix=\&#34;.server.go.\&#34;)
try:
    with os.fdopen(fd, \&#34;w\&#34;) as f:
        f.write(s.replace(old,new,1)); f.flush(); os.fsync(f.fileno())
    os.replace(name,p)
    now=time.time_ns(); os.utime(p, ns=(now,now))
finally:
    if os.path.exists(name): os.unlink(name)&#39;&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;api/server.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): unknown param &#34;language&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, file, placement, source, target, target_symbol]: invalid operation parameters<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_start&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;(&#34;&#39;^docs/SUBOPTIMAL_TOOLS&#39;&#34;\\.md&#34;&#39;$|&#39;&#39;^docs/)&#39;&#34;&#39; | head -30 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_start&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 -c &#39;from pathlib import Path; import os; p=Path(\&#34;docs/SUBOPTIMAL_TOOLS.md\&#34;); p.parent.mkdir(parents=True, exist_ok=True); old=p.read_text() if p.exists() else \&#34;# Suboptimal Tool Behavior\\n\&#34;; entry=\&#34;\\n- Tool: &#34;&#39;`semantic_insert_function`; target: `api/server.go`; observed failure: the initial call included an unsupported `language` parameter and was rejected before editing; workaround: retried without that parameter; root cause: passed a parameter not in the tool schema.&#39;&#34;\\n\&#34;; tmp=p.with_suffix(\&#34;.md.tmp\&#34;); f=tmp.open(\&#34;w\&#34;); f.write(old+entry); f.flush(); os.fsync(f.fileno()); f.close(); os.replace(tmp,p)&#39; &amp;&amp; \
cat api/server.go &amp;&amp; \
git diff -- api/server.go docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/server.go modified (9 lines -> 13 lines)
* **MCP Edit**: File api/server.go modified (9 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat PROJECT_BRIEF.md &amp;&amp; \
sed -n &#39;1,240p&#39; api/server.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat api/server.go &amp;&amp; \
cat PROJECT_BRIEF.md 2&gt;/dev/null || \
true &amp;&amp; \
rg -n &#39;semantic_(lookup|insert_function|inspect_symbol)&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go; git status --short; rg -n &#39;type Server|func [a-z].*\\(&#39;&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_start&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I discovered and used `semantic_outline` for inspection, then called `semantic_insert_function` to make the single source change; the initial insertion attempt failed due to an unsupported argument, so I retried it. `semantic_batch` was unsuitable here because there was only one successful mutation to perform, with no sequence of edits to combine.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

