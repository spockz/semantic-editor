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

* **Date**: 2026-09-22 12:08:49 CEST

## Test case: `task-04-insert-public`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: default prompt · prescriptive MCP instructions

* **Fixture**: `testdata/bench/task_04_insert_public.txtar` (`sha256:a78832217230b078ce5bd605 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add public constructor func InitServer() *Server placed before private helpers. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add public constructor func InitServer() *Server placed before private helpers. Prefer using semantic editor operations if applicable. When done, output DONE.

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 44.46s | 49.72s | <span class="benchmark-delta-negative">+11.8%</span> | 49.38s | 61.82s | N/A |
| **Process Start → First Event** | 0.79s | 0.61s | — | 0.65s | 0.14s | — |
| **First Event → First Tool Call** | 7.24s | 7.19s | — | 8.88s | 8.23s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | N/A |
| **Internal Tool Cycles** | 5 | 7 | <span class="benchmark-delta-negative">+40.0%</span> | 5 | 9 | N/A |
| **Initial Load / Discovery Turns** | 5 | 2 | <span class="benchmark-delta-positive">-60.0%</span> | 5 | 3 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 5 | 7 | <span class="benchmark-delta-negative">+40.0%</span> | 5 | 9 | N/A |
| **Output Tokens** | 1397 | 1501 | <span class="benchmark-delta-negative">+7.4%</span> | 1283 | 1755 | N/A |
| **Reasoning / Thinking Tokens** | 545 | 490 | <span class="benchmark-delta-positive">-10.1%</span> | 479 | 413 | N/A |
| **Total Input Tokens** | 141549 | 225416 | <span class="benchmark-delta-negative">+59.2%</span> | 145093 | 190262 | N/A |
| **Cached Input Tokens** | 126208 | 206592 | <span class="benchmark-delta-positive">+63.7%</span> | 120064 | 152576 | N/A |
| **Uncached Input Tokens** | 15341 | 18824 | <span class="benchmark-delta-negative">+22.7%</span> | 25029 | 37686 | N/A |
| **Cached vs Uncached Token Ratio** | 8.23:1 | 10.97:1 | <span class="benchmark-delta-positive">+33.4%</span> | 4.80:1 | 4.05:1 | N/A |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/server.go modified (9 lines -> 13 lines)
* **MCP Edit**: File api/server.go modified (9 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!*.test&#39;&#34;&#39; | head -200 &amp;&amp; \
rg -n \&#34;type Server|func .*Server|private|helper\&#34; -g &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*test*&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | head -80 &amp;&amp; \
printf &#39;\\n--- docs ---\\n&#39; &amp;&amp; \
(test -f PROJECT_BRIEF.md &amp;&amp; \
sed -n &#39;1,220p&#39; PROJECT_BRIEF.md || \
true) &amp;&amp; \
printf &#39;\\n--- git status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; api/server.go &amp;&amp; \
printf &#39;\\n--- files ---\\n&#39; &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;AGENTS*&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go &amp;&amp; \
printf &#39;\\n--- files ---\\n&#39; &amp;&amp; \
rg --files -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;AGENTS*&#39; -g &#39;*.md&#39; -g &#39;*.go&#39; -g &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w api/server.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff -- api/server.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_end&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- api/server.go &amp;&amp; \
git diff --check -- api/server.go &amp;&amp; \
git diff --quiet -- go.mod go.sum; test $? -eq 0&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; api/server.go &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- api/server.go &amp;&amp; \
printf &#39;\\n--- protected/test status ---\\n&#39; &amp;&amp; \
git status --short -- api/server.go go.mod go.sum api/server_test.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel &amp;&amp; \
git status --short -- api/server.go api/server_test.go go.mod go.sum &amp;&amp; \
sed -n &#39;1,80p&#39; api/server.go &amp;&amp; \
git ls-files --stage -- api/server.go api/server_test.go go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short --untracked-files=all &amp;&amp; \
printf &#39;\\n--- HEAD file ---\\n&#39; &amp;&amp; \
git show HEAD:api/server.go &amp;&amp; \
printf &#39;\\n--- check ---\\n&#39; &amp;&amp; \
git diff --no-ext-diff --exit-code -- api/server.go; printf &#39;diff_exit=%s\\n&#39; &#34;&#39;$?&#39;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel; git ls-files --stage -- api/server.go; git status --short -- api/server.go; git diff --no-index /dev/null api/server.go || \
true&#39;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;ls -la &amp;&amp; \
(test -f Makefile &amp;&amp; \
make check || \
true) &amp;&amp; \
(test -f go.mod &amp;&amp; \
go test ./... || \
true)&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/server.go modified (9 lines -> 13 lines)
* **MCP Edit**: File api/server.go modified (9 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; | sort &amp;&amp; \
rg -n \&#34;type Server|func .*Server|private|helper\&#34; -g &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | sort | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; api/server.go &amp;&amp; \
sed -n &#39;1,160p&#39; api/server_test.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; AGENTS.override.md&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- api/server.go &amp;&amp; \
go test ./api/...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go &amp;&amp; \
sed -n &#39;1,180p&#39; api/router.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; api/server.go &amp;&amp; \
git status --short -- api/server.go &amp;&amp; \
git diff HEAD -- api/server.go &amp;&amp; \
git show HEAD:api/server.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tools ---&#39; &amp;&amp; \
rg -n &#39;semantic_(insert|verify|batch)|codex_review&#39; &lt;(printf &#39;%s\\n&#39; &#39;&#39;) 2&gt;/dev/null || \
true&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w api/server.go &amp;&amp; \
go test ./... &amp;&amp; \
git status --short --untracked-files=all --ignored api/server.go api/server_test.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;InitServer|internalRun|type Server|Server\\{\&#34; . --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &amp;&amp; \
rg -n \&#34;make check|go test|gofmt\&#34; README* .github Makefile 2&gt;/dev/null || \
true&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; api/server_test.go &amp;&amp; \
sed -n &#39;1,160p&#39; testdata/bench/task_04_insert_public.txtar 2&gt;/dev/null || \
true &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w api/server.go &amp;&amp; \
go test ./...&#39;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- api/server.go &amp;&amp; \
git diff --quiet -- go.mod go.sum; test $? -eq 0 &amp;&amp; \
echo &#39;&#34;&#39;protected files unchanged&#39;; git diff --name-only -- api/server.go api/server_test.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; api/server.go &amp;&amp; \
git status --short -- api/server.go api/server_test.go go.mod go.sum &amp;&amp; \
git diff --no-index /dev/null api/server.go &gt;/dev/null; true&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-error"><div class="callout-title"><span>✕</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not call a `semantic_*` tool because no semedit MCP tools were exposed in the available tool list. I used ordinary editing instead.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.25s; turns: 1; tool calls: 0.</p>
</details>
