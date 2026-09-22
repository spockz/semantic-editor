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

* **Date**: 2026-09-22 11:55:01 CEST

## Test case: `task-04-insert-public`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: default prompt · none MCP instructions

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 44.64s | 47.35s | <span class="benchmark-delta-negative">+6.1%</span> | 47.58s | 94.86s | <span class="benchmark-delta-negative">+99.4%</span> |
| **Process Start → First Event** | 0.14s | 0.14s | — | 0.59s | 0.57s | — |
| **First Event → First Tool Call** | 8.55s | 6.80s | — | 7.10s | 13.06s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 5 | 6 | <span class="benchmark-delta-negative">+20.0%</span> | 6 | 6 | 0% |
| **Initial Load / Discovery Turns** | 5 | 2 | <span class="benchmark-delta-positive">-60.0%</span> | 6 | 2 | <span class="benchmark-delta-positive">-66.7%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 5 | 6 | <span class="benchmark-delta-negative">+20.0%</span> | 6 | 6 | 0% |
| **Output Tokens** | 1082 | 1432 | <span class="benchmark-delta-negative">+32.3%</span> | 1400 | 1514 | <span class="benchmark-delta-negative">+8.1%</span> |
| **Reasoning / Thinking Tokens** | 328 | 411 | <span class="benchmark-delta-negative">+25.3%</span> | 426 | 479 | <span class="benchmark-delta-negative">+12.4%</span> |
| **Total Input Tokens** | 142531 | 189689 | <span class="benchmark-delta-negative">+33.1%</span> | 164005 | 258687 | <span class="benchmark-delta-negative">+57.7%</span> |
| **Cached Input Tokens** | 124160 | 163840 | <span class="benchmark-delta-positive">+32.0%</span> | 148480 | 232192 | <span class="benchmark-delta-positive">+56.4%</span> |
| **Uncached Input Tokens** | 18371 | 25849 | <span class="benchmark-delta-negative">+40.7%</span> | 15525 | 26495 | <span class="benchmark-delta-negative">+70.7%</span> |
| **Cached vs Uncached Token Ratio** | 6.76:1 | 6.34:1 | <span class="benchmark-delta-negative">-6.2%</span> | 9.56:1 | 8.76:1 | <span class="benchmark-delta-negative">-8.4%</span> |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; | head -80&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | head -80 &amp;&amp; \
printf &#39;\\n--- docs ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,220p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go; printf &#39;\\n--- tests (read-only) ---\\n&#39;; sed -n &#39;1,220p&#39; api/server_test.go; printf &#39;\\n--- status ---\\n&#39;; git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go &amp;&amp; \
printf &#39;\\n--- test ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; api/server_test.go &amp;&amp; \
printf &#39;\\n--- module ---\\n&#39; &amp;&amp; \
sed -n &#39;1,80p&#39; go.mod&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- api/server.go; go test ./api&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_end&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --name-only -- api/server.go api/server_test.go go.mod go.sum; git diff --check -- api/server.go&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; api/server.go &amp;&amp; \
printf &#39;\\n--- checks ---\\n&#39; &amp;&amp; \
make check&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; api/server.go; git status --short -- api/server.go api/server_test.go go.mod go.sum; git ls-files --stage -- api/server.go api/server_test.go go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go test ./... &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- api/server.go &amp;&amp; \
printf &#39;\\n--- protected status ---\\n&#39; &amp;&amp; \
git status --short -- go.mod go.sum api/server.go api/server_test.go&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short --untracked-files=all -- api/server.go api/server_test.go go.mod go.sum; git ls-files --stage -- api/server.go api/server_test.go go.mod go.sum; git diff --no-index /dev/null api/server.go || \
true&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/server.go modified (9 lines -> 13 lines)
* **MCP Edit**: File api/server.go modified (9 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | sort | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; | sort; printf &#39;\\n--- git status ---\\n&#39;; git status --short; printf &#39;\\n--- top-level ---\\n&#39;; ls -la&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go &amp;&amp; \
printf &#39;\\n--- related refs ---\\n&#39; &amp;&amp; \
rg -n \&#34;type Server|NewServer|InitServer|func [a-z]\&#34; api *.go */*.go 2&gt;/dev/null&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go; printf &#39;\\n--- router ---\\n&#39;; sed -n &#39;1,220p&#39; api/router.go; printf &#39;\\n--- go.mod ---\\n&#39;; sed -n &#39;1,80p&#39; go.mod; printf &#39;\\n--- make/check files ---\\n&#39;; rg --files -g &#39;Makefile&#39; -g &#39;makefile&#39; -g &#39;Taskfile*&#39; -g &#39;README*&#39; -g &#39;PROJECT_BRIEF.md&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; api/server_test.go &amp;&amp; \
sed -n &#39;1,180p&#39; api/router.go &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_end&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w api/server.go &amp;&amp; \
git diff --check &amp;&amp; \
go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- api/server.go; printf &#39;\\n--- status in fixture ---\\n&#39;; git status --short -- api/server.go api/server_test.go go.mod go.sum; printf &#39;\\n--- tests ---\\n&#39;; go test ./...&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- api/server.go &amp;&amp; \
git diff --name-only -- go.mod go.sum api/*_test.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; api/server.go; printf &#39;\\n--- status ---\\n&#39;; git status --short --untracked-files=all; printf &#39;\\n--- diff stat ---\\n&#39;; git diff --stat&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; api/server.go; git diff --no-index /dev/null api/server.go || \
true; git ls-files --stage -- api/server.go api/server_test.go go.mod go.sum; test -f Makefile &amp;&amp; \
make check || \
true&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git ls-files --stage -- api/server.go api/server_test.go go.mod; git check-ignore -v api/server.go || \
true; git diff --no-index /dev/null api/server.go || \
true&#39;</code></pre></td></tr>
</tbody>
</table>
