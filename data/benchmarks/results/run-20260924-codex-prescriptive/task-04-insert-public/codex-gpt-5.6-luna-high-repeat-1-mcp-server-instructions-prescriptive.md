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

* **Date**: 2026-09-24 18:34:15 CEST

## Test case: `task-04-insert-public`

### Target: `codex/gpt-5.6-luna/high (repeat 1)`

#### Configuration: default prompt · prescriptive MCP instructions

* **Run Provenance**: `binary=44027cf7b57a0b4e`

* **Fixture**: [testdata/bench/task_04_insert_public.txtar](https://github.com/spockz/semantic-editor/blob/dd39d52587030a6aae7bbbf05b381f54bb386d3a/testdata/bench/task_04_insert_public.txtar)

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 35.83s | 50.33s | <span class="benchmark-delta-negative">+40.5%</span> | 54.26s | 52.46s | <span class="benchmark-delta-positive">-3.3%</span> |
| **Process Start → First Event** | 0.58s | 0.12s | — | 0.12s | 0.13s | — |
| **First Event → First Tool Call** | 8.70s | 7.24s | — | 8.20s | 10.48s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 5 | 6 | <span class="benchmark-delta-negative">+20.0%</span> | 6 | 6 | 0% |
| **Initial Load / Discovery Turns** | 5 | 2 | <span class="benchmark-delta-positive">-60.0%</span> | 6 | 2 | <span class="benchmark-delta-positive">-66.7%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 5 | 6 | <span class="benchmark-delta-negative">+20.0%</span> | 6 | 6 | 0% |
| **Output Tokens** | 1145 | 1683 | <span class="benchmark-delta-negative">+47.0%</span> | 1849 | 1943 | <span class="benchmark-delta-negative">+5.1%</span> |
| **Reasoning / Thinking Tokens** | 347 | 738 | <span class="benchmark-delta-negative">+112.7%</span> | 803 | 861 | <span class="benchmark-delta-negative">+7.2%</span> |
| **Total Input Tokens** | 102686 | 208466 | <span class="benchmark-delta-negative">+103.0%</span> | 168967 | 230867 | <span class="benchmark-delta-negative">+36.6%</span> |
| **Cached Input Tokens** | 84736 | 177152 | <span class="benchmark-delta-positive">+109.1%</span> | 141568 | 200704 | <span class="benchmark-delta-positive">+41.8%</span> |
| **Uncached Input Tokens** | 17950 | 31314 | <span class="benchmark-delta-negative">+74.5%</span> | 27399 | 30163 | <span class="benchmark-delta-negative">+10.1%</span> |
| **Cached vs Uncached Token Ratio** | 4.72:1 | 5.66:1 | <span class="benchmark-delta-positive">+19.8%</span> | 5.17:1 | 6.65:1 | <span class="benchmark-delta-positive">+28.8%</span> |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | sort &amp;&amp; \
rg -n \&#34;type Server|func .*Server|private|helpers\&#34; -g &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | sed -n &#39;1,120p&#39; &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; api/server.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go &amp;&amp; \
printf &#39;\\n--- tests ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; api/server_test.go &amp;&amp; \
printf &#39;\\n--- top-level files ---\\n&#39; &amp;&amp; \
rg --files -g &#39;&#34;&#39;!api/server.go&#39;&#34;&#39; -g &#39;&#34;&#39;!api/server_test.go&#39;&#34;&#39; | sed -n &#39;1,120p&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; Makefile 2&gt;/dev/null || \
true; sed -n &#39;1,120p&#39; go.mod; sed -n &#39;1,120p&#39; go.sum 2&gt;/dev/null || \
true&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;before_symbol&#34;,
  &#34;target_symbol&#34;: &#34;internalRun&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; api/server_test.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- api/server.go &amp;&amp; \
printf &#39;\\n--- file ---\\n&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; api/server.go &amp;&amp; \
printf &#39;\\n--- test ---\\n&#39; &amp;&amp; \
go test ./...&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w api/server.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff --check &amp;&amp; \
git status --short -- api/server.go api/server_test.go go.mod go.sum &amp;&amp; \
git diff -- api/server.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- api/server.go api/server_test.go go.mod go.sum &amp;&amp; \
printf &#39;\\n--- HEAD server ---\\n&#39; &amp;&amp; \
git show HEAD:api/server.go &amp;&amp; \
printf &#39;\\n--- protected hashes ---\\n&#39; &amp;&amp; \
shasum go.mod go.sum 2&gt;/dev/null || \
true&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,80p&#39; &amp;&amp; \
git ls-files --stage | sed -n &#39;1,40p&#39; &amp;&amp; \
printf &#39;\\n--- ignore ---\\n&#39; &amp;&amp; \
git check-ignore -v api/server.go api/server_test.go go.mod || \
true&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/server.go modified (9 lines -> 13 lines)
* **MCP Edit**: File api/server.go modified (9 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*_test.go&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort | sed -n &#39;1,120p&#39; &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | sort; printf &#39;\\n--- brief ---\\n&#39;; if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,220p&#39; PROJECT_BRIEF.md; fi; printf &#39;\\n--- git status ---\\n&#39;; git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go &amp;&amp; \
printf &#39;\\n--- router ---\\n&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; api/router.go &amp;&amp; \
printf &#39;\\n--- project docs ---\\n&#39; &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*.md&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- api/server.go ---&#39;; nl -ba api/server.go; printf &#39;%s\\n&#39; &#39;--- api/router.go ---&#39;; nl -ba api/router.go; printf &#39;%s\\n&#39; &#39;--- Makefile ---&#39;; if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; fi; printf &#39;%s\\n&#39; &#39;--- go files ---&#39;; for f in api/*.go auth/*.go cache/*.go config/*.go db/*.go health/*.go logging/*.go metrics/*.go models/*.go utils/*.go; do printf &#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,180p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Server|InitServer|internalRun|Start\&#34; . -g &#39;*.go&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_end&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; api/server_test.go &amp;&amp; \
sed -n &#39;1,160p&#39; AGENTS.override.md&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- api/server.go &amp;&amp; \
git status --short -- api/server.go api/server_test.go go.mod go.sum &amp;&amp; \
if [ -f Makefile ]; then make check; else go test ./...; fi&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- diff ---&#39;; git diff -- api/server.go; printf &#39;%s\\n&#39; &#39;--- changed files in fixture ---&#39;; git status --short -- api/server.go api/server_test.go go.mod go.sum; printf &#39;%s\\n&#39; &#39;--- go test ---&#39;; go test ./...&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; api/server.go &amp;&amp; \
gofmt -d api/server.go &amp;&amp; \
git rev-parse --show-toplevel &amp;&amp; \
git status --short --untracked-files=all -- api/server.go api/server_test.go go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;nl -ba api/server.go; git status --short --untracked-files=all; git diff --no-ext-diff -- api/server.go; git show HEAD:api/server.go | nl -ba&#39;</code></pre></td></tr>
</tbody>
</table>

