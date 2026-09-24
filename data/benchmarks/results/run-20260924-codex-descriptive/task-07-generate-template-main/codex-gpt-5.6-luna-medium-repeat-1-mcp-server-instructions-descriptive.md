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

* **Date**: 2026-09-24 16:41:35 CEST

## Test case: `task-07-generate-template-main`

### Target: `codex/gpt-5.6-luna/medium (repeat 1)`

#### Configuration: crypto_rand prompt · descriptive MCP instructions

* **Run Provenance**: `binary=44027cf7b57a0b4e`

* **Fixture**: [testdata/scripts/generate_template_main.txtar](https://github.com/spockz/semantic-editor/blob/d48187b57f4c6ee3e1c7b415974bedde8ba9d43b/testdata/scripts/generate_template_main.txtar)

**Vanilla LLM Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer from crypto/rand, ensuring all necessary standard library packages are imported cleanly. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer from crypto/rand, ensuring all necessary standard library packages are imported cleanly. Prefer using semantic editor operations if applicable. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package main

func main() {
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 44.43s | 97.13s | <span class="benchmark-delta-negative">+118.6%</span> | 43.00s | 74.80s | <span class="benchmark-delta-negative">+74.0%</span> |
| **Process Start → First Event** | 1.47s | 1.46s | — | 0.12s | 0.15s | — |
| **First Event → First Tool Call** | 8.17s | 8.17s | — | 8.15s | 9.72s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> | 4 | 8 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Initial Load / Discovery Turns** | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> | 4 | 1 | <span class="benchmark-delta-positive">-75.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> | 4 | 8 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Output Tokens** | 1493 | 3229 | <span class="benchmark-delta-negative">+116.3%</span> | 1548 | 1983 | <span class="benchmark-delta-negative">+28.1%</span> |
| **Reasoning / Thinking Tokens** | 606 | 1698 | <span class="benchmark-delta-negative">+180.2%</span> | 589 | 643 | <span class="benchmark-delta-negative">+9.2%</span> |
| **Total Input Tokens** | 122958 | 301534 | <span class="benchmark-delta-negative">+145.2%</span> | 143436 | 280978 | <span class="benchmark-delta-negative">+95.9%</span> |
| **Cached Input Tokens** | 91648 | 270592 | <span class="benchmark-delta-positive">+195.3%</span> | 107776 | 253440 | <span class="benchmark-delta-positive">+135.2%</span> |
| **Uncached Input Tokens** | 31310 | 30942 | <span class="benchmark-delta-positive">-1.2%</span> | 35660 | 27538 | <span class="benchmark-delta-positive">-22.8%</span> |
| **Cached vs Uncached Token Ratio** | 2.93:1 | 8.75:1 | <span class="benchmark-delta-positive">+198.8%</span> | 3.02:1 | 9.20:1 | <span class="benchmark-delta-positive">+204.5%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 20 lines)
* **MCP Edit**: File main.go modified (4 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project docs ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | head -80 &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project docs ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,220p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run main.go &amp;&amp; \
git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff -- main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;n, err := rand.Int(rand.Reader, big.NewInt(100))\nif err != nil {\n\tlog.Fatal(err)\n}\n\ntmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World: {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, n); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git check-ignore -v main.go go.mod go.sum 2&gt;/dev/null || \
true &amp;&amp; \
test &#34;&#39;! -e go.mod &amp;&amp; \
test ! -e go.sum &amp;&amp; \
git diff --check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- main.go &amp;&amp; \
printf &#39;\\n--- format check ---\\n&#39; &amp;&amp; \
test -z \&#34;&#34;&#39;$(gofmt -d main.go)&#34; &amp;&amp; \
echo gofmt-ok &amp;&amp; \
printf &#39;&#34;&#39;\\n--- run ---\\n&#39; &amp;&amp; \
go run ./main.go &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;text/template&#34;
  ],
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;mkdir -p docs&#39;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- main.go ---&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- validation ---&#39; &amp;&amp; \
test -z \&#34;&#34;&#39;$(gofmt -d main.go)&#34; &amp;&amp; \
echo gofmt-ok &amp;&amp; \
GO111MODULE=off go test ./... &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; &#39;--- execution ---&#39; &amp;&amp; \
GO111MODULE=off go run ./main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected files ---&#39; &amp;&amp; \
if [ -e go.mod ] || \
[ -e go.sum ]; then echo &#39;protected file present; unchanged check requires git status&#39;; else echo &#39;no go.mod or go.sum in fixture&#39;; fi &amp;&amp; \
git status --short -- main.go go.mod go.sum docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;crypto/rand&#34;,
    &#34;log&#34;,
    &#34;math/big&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ],
  &#34;remove&#34;: [
    &#34;crypto/rand&#34;,
    &#34;log&#34;,
    &#34;math/big&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; main.go &amp;&amp; \
test -z \&#34;&#34;&#39;$(gofmt -d main.go)&#34; &amp;&amp; \
GO111MODULE=off go test ./... &amp;&amp; \
GO111MODULE=off go run ./main.go&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 25 lines)
* **MCP Edit**: File main.go modified (4 lines -> 20 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | head -80 &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project docs ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,200p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;.codex/**&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project brief ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
printf &#39;\\n--- make targets ---\\n&#39; &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,200p&#39; Makefile; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
make check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- files ---&#39; &amp;&amp; \
rg --files -uu | sort &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- git status ---&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- go.mod ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- available tools ---&#39; &amp;&amp; \
command -v semedit || \
true &amp;&amp; \
command -v go || \
true&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go test ./... &amp;&amp; \
go run . &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short -- main.go go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nn, err := rand.Int(rand.Reader, big.NewInt(100))\nif err != nil {\n\tlog.Fatal(err)\n}\nif err := tmpl.Execute(os.Stdout, n); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git status --short &amp;&amp; \
git diff --no-ext-diff --no-textconv -- main.go; git ls-files --stage main.go; git check-ignore -v main.go || \
true&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
printf &#39;\\n--- test ---\\n&#39; &amp;&amp; \
go test ./... &amp;&amp; \
printf &#39;\\n--- run ---\\n&#39; &amp;&amp; \
go run .&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ],
  &#34;add&#34;: [
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; main.go &amp;&amp; \
printf &#39;\\n--- verification ---\\n&#39; &amp;&amp; \
go test ./... &amp;&amp; \
output=&#34;&#39;$(go run .) &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; &amp;&amp; \
case &#34;$output&#34; in &#39;&#34;&#39;Hello World &#39;* ) ;; * ) exit 1 ;; esac &amp;&amp; \
test \&#34;&#34;&#39;$(git diff -- go.mod go.sum | wc -c | tr -d &#39;&#34;&#39; &#39;)\&#34; -eq 0&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;remove&#34;: [
    &#34;crypto/rand&#34;,
    &#34;log&#34;,
    &#34;math/big&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ],
  &#34;add&#34;: [
    &#34;crypto/rand&#34;,
    &#34;log&#34;,
    &#34;math/big&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,40p&#39; main.go &amp;&amp; \
gofmt -d main.go &amp;&amp; \
go test ./... &gt;/tmp/semedit-test-output &amp;&amp; \
tail -n 5 /tmp/semedit-test-output &amp;&amp; \
output=&#34;&#39;$(go run .) &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; &amp;&amp; \
case &#34;$output&#34; in &#39;&#34;&#39;Hello World &#39;* ) ;; * ) exit 1 ;; esac&#34;</code></pre></td></tr>
</tbody>
</table>

#### Configuration: default prompt · descriptive MCP instructions

* **Run Provenance**: `binary=44027cf7b57a0b4e`

* **Fixture**: [testdata/scripts/generate_template_main.txtar](https://github.com/spockz/semantic-editor/blob/d48187b57f4c6ee3e1c7b415974bedde8ba9d43b/testdata/scripts/generate_template_main.txtar)

**Vanilla LLM Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Prefer using semantic editor operations if applicable. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package main

func main() {
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 36.93s | 60.47s | <span class="benchmark-delta-negative">+63.7%</span> | 44.29s | 50.00s | <span class="benchmark-delta-negative">+12.9%</span> |
| **Process Start → First Event** | 0.14s | 0.14s | — | 0.12s | 0.13s | — |
| **First Event → First Tool Call** | 8.20s | 8.26s | — | 8.17s | 13.15s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> | 5 | 4 | <span class="benchmark-delta-positive">-20.0%</span> |
| **Initial Load / Discovery Turns** | 4 | 1 | <span class="benchmark-delta-positive">-75.0%</span> | 5 | 1 | <span class="benchmark-delta-positive">-80.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> | 5 | 4 | <span class="benchmark-delta-positive">-20.0%</span> |
| **Output Tokens** | 1116 | 1756 | <span class="benchmark-delta-negative">+57.3%</span> | 1700 | 890 | <span class="benchmark-delta-positive">-47.6%</span> |
| **Reasoning / Thinking Tokens** | 479 | 653 | <span class="benchmark-delta-negative">+36.3%</span> | 756 | 401 | <span class="benchmark-delta-positive">-47.0%</span> |
| **Total Input Tokens** | 120420 | 231911 | <span class="benchmark-delta-negative">+92.6%</span> | 145999 | 167339 | <span class="benchmark-delta-negative">+14.6%</span> |
| **Cached Input Tokens** | 105984 | 183040 | <span class="benchmark-delta-positive">+72.7%</span> | 128256 | 144896 | <span class="benchmark-delta-positive">+13.0%</span> |
| **Uncached Input Tokens** | 14436 | 48871 | <span class="benchmark-delta-negative">+238.5%</span> | 17743 | 22443 | <span class="benchmark-delta-negative">+26.5%</span> |
| **Cached vs Uncached Token Ratio** | 7.34:1 | 3.75:1 | <span class="benchmark-delta-negative">-49.0%</span> | 7.23:1 | 6.46:1 | <span class="benchmark-delta-negative">-10.7%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 15 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sed -n &#39;1,80p&#39; &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project docs ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go run main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; main.go &amp;&amp; \
printf &#39;\\n--- protected-file status ---\\n&#39; &amp;&amp; \
git status --short -- go.mod go.sum main.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go go.mod go.sum &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;math/rand&#34;,
    &#34;text/template&#34;
  ],
  &#34;remove&#34;: [
    &#34;crypto/rand&#34;,
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; main.go &amp;&amp; \
printf &#39;\\n--- make check ---\\n&#39; &amp;&amp; \
make check &amp;&amp; \
printf &#39;\\n--- run ---\\n&#39; &amp;&amp; \
go run .&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d main.go &amp;&amp; \
go test ./... &amp;&amp; \
go run .&#39;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 15 lines)
* **MCP Edit**: File main.go modified (4 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; | head -80 &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project docs ---\\n&#39; &amp;&amp; \
for f in PROJECT_BRIEF.md Makefile; do if [ -f \&#34;&#34;&#39;$f&#34; ]; then echo &#34;### $f&#34;; sed -n &#39;&#34;&#39;1,220p&#39; \&#34;&#34;&#39;$f&#34;; fi; done&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;AGENTS*&#39; | sort &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
printf &#39;\\n--- relevant source conventions ---\\n&#39; &amp;&amp; \
sed -n &#39;1,160p&#39; api/router.go &amp;&amp; \
sed -n &#39;1,120p&#39; logging/logging.go&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tlog.Fatal(err)\n}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
git diff --check &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;log&#34;,
    &#34;math/rand&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go run . &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- main.go go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
go run .&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git status --short --untracked-files=all &amp;&amp; \
git diff --no-ext-diff --no-color -- main.go&#34;</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because it was not discovered as an available callable operation during the task. The sequential operations were otherwise suitable and completed successfully.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 7.75s; turns: 1; tool calls: 0.</p>
</details>

#### Configuration: prefer_discover_semedit prompt · descriptive MCP instructions

* **Run Provenance**: `binary=44027cf7b57a0b4e`

* **Fixture**: [testdata/scripts/generate_template_main.txtar](https://github.com/spockz/semantic-editor/blob/d48187b57f4c6ee3e1c7b415974bedde8ba9d43b/testdata/scripts/generate_template_main.txtar)

**Vanilla LLM Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Prefer using semantic editor operations if applicable. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package main

func main() {
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 36.35s | 80.26s | <span class="benchmark-delta-negative">+120.8%</span> | 46.11s | 71.58s | <span class="benchmark-delta-negative">+55.2%</span> |
| **Process Start → First Event** | 0.12s | 0.11s | — | 0.12s | 0.12s | — |
| **First Event → First Tool Call** | 14.21s | 16.53s | — | 12.69s | 16.10s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 2 | 8 | <span class="benchmark-delta-negative">+300.0%</span> | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 2 | 8 | <span class="benchmark-delta-negative">+300.0%</span> | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> |
| **Output Tokens** | 1240 | 2076 | <span class="benchmark-delta-negative">+67.4%</span> | 1753 | 1978 | <span class="benchmark-delta-negative">+12.8%</span> |
| **Reasoning / Thinking Tokens** | 559 | 778 | <span class="benchmark-delta-negative">+39.2%</span> | 791 | 927 | <span class="benchmark-delta-negative">+17.2%</span> |
| **Total Input Tokens** | 168427 | 325525 | <span class="benchmark-delta-negative">+93.3%</span> | 145269 | 303447 | <span class="benchmark-delta-negative">+108.9%</span> |
| **Cached Input Tokens** | 144896 | 298240 | <span class="benchmark-delta-positive">+105.8%</span> | 129280 | 273920 | <span class="benchmark-delta-positive">+111.9%</span> |
| **Uncached Input Tokens** | 23531 | 27285 | <span class="benchmark-delta-negative">+16.0%</span> | 15989 | 29527 | <span class="benchmark-delta-negative">+84.7%</span> |
| **Cached vs Uncached Token Ratio** | 6.16:1 | 10.93:1 | <span class="benchmark-delta-positive">+77.5%</span> | 8.09:1 | 9.28:1 | <span class="benchmark-delta-positive">+14.7%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 15 lines)
* **MCP Edit**: File main.go modified (4 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
if [ -f main.go ]; then sed -n &#39;1,240p&#39; main.go; fi &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,200p&#39; Makefile; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;README*&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; | sort; sed -n &#39;1,240p&#39; PROJECT_BRIEF.md 2&gt;/dev/null; sed -n &#39;1,220p&#39; main.go 2&gt;/dev/null&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
git diff --check &amp;&amp; \
go test ./... &amp;&amp; \
go run . &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sort; git status --short; sed -n &#39;1,120p&#39; go.mod 2&gt;/dev/null; sed -n &#39;1,160p&#39; AGENTS.override.md&#34;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;tmpl := texttemplate.Must(texttemplate.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;math/rand&#34;,
    &#34;texttemplate text/template&#34;
  ],
  &#34;remove&#34;: [
    &#34;crypto/rand&#34;,
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go; git diff -- main.go; git diff -- go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;set -e
gofmt -w main.go
go test ./...
go vet ./...
git diff --check
test ! -e Makefile || \
make check
if git diff --quiet -- go.mod go.sum; then
  echo &#39;&#34;&#39;protected module files unchanged&#39;
else
  echo &#39;protected module files changed&#39; &gt;&amp;2
  exit 1
fi
sed -n &#39;1,120p&#39; main.go
git diff -- main.go&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 15 lines)
* **MCP Edit**: File main.go modified (4 lines -> 19 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- available files ---&#39; &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- project brief ---&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; else printf &#39;%s\\n&#39; &#39;(none)&#39;; fi &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- main.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; printf &#39;\\n-- files --\\n&#39;; rg --files -g &#39;&#34;&#39;!*AGENTS.override.md&#39;&#34;&#39; | sort; printf &#39;\\n-- main.go --\\n&#39;; sed -n &#39;1,200p&#39; main.go; printf &#39;\\n-- project brief --\\n&#39;; if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi; printf &#39;\\n-- make targets --\\n&#39;; if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- module metadata ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- main references ---&#39; &amp;&amp; \
rg -n \&#34;Hello World|text/template|rand|func main|main\\(\&#34; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; go.mod; printf &#39;\\n-- related source patterns --\\n&#39;; rg -n \&#34;template|rand|Hello|func main\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- go.mod go.sum main.go &amp;&amp; \
git status --short -- go.mod go.sum main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tpanic(err)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tracked diff ---&#39; &amp;&amp; \
git diff -- main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- status ---&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go; printf &#39;\\n-- git diff -- --\\n&#39;; git diff -- main.go go.mod go.sum; printf &#39;\\n-- status --\\n&#39;; git status --short&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nvalue, err := rand.Int(rand.Reader, big.NewInt(1_000_000))\nif err != nil {\n\tpanic(err)\n}\nif err := tmpl.Execute(os.Stdout, value); err != nil {\n\tpanic(err)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go; printf &#39;\\n-- module files unchanged --\\n&#39;; git diff --no-ext-diff -- go.mod go.sum; printf &#39;\\n-- checks --\\n&#39;; gofmt -d main.go; go test ./...&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;output=$(go run .); case &#34;$output&#34; in &#39;&#34;&#39;Hello World &#39;* ) printf &#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34;;; *) printf &#39;&#34;&#39;unexpected output: %s\\n&#39; \&#34;&#34;&#39;$output&#34; &gt;&amp;2; exit 1;; esac&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was unsuitable because the operations depended on the results of the preceding semantic edits: the initial import inference selected incorrect packages, so each subsequent correction required inspecting and adjusting the current file state. The semantic operations were callable individually, and no retry was needed.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 7.28s; turns: 1; tool calls: 0.</p>
</details>

