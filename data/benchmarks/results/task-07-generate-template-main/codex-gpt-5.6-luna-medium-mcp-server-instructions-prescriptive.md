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

## Test case: `task-07-generate-template-main`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: crypto_rand prompt · prescriptive MCP instructions

* **Fixture**: [testdata/scripts/generate_template_main.txtar](https://github.com/spockz/semantic-editor/blob/e16932220de9afe14201fa4af671d1db244f3ca3/testdata/scripts/generate_template_main.txtar)

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
| **Wall-Clock Latency** | 37.99s | 103.29s | <span class="benchmark-delta-negative">+171.9%</span> | 50.01s | 83.89s | <span class="benchmark-delta-negative">+67.7%</span> |
| **Process Start → First Event** | 0.54s | 0.76s | — | 0.51s | 0.91s | — |
| **First Event → First Tool Call** | 9.00s | 19.22s | — | 9.13s | 7.97s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 6 | <span class="benchmark-delta-negative">+100.0%</span> | 5 | 9 | <span class="benchmark-delta-negative">+80.0%</span> |
| **Initial Load / Discovery Turns** | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> | 5 | 2 | <span class="benchmark-delta-positive">-60.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 6 | <span class="benchmark-delta-negative">+100.0%</span> | 5 | 9 | <span class="benchmark-delta-negative">+80.0%</span> |
| **Output Tokens** | 1325 | 1608 | <span class="benchmark-delta-negative">+21.4%</span> | 1640 | 2387 | <span class="benchmark-delta-negative">+45.5%</span> |
| **Reasoning / Thinking Tokens** | 558 | 700 | <span class="benchmark-delta-negative">+25.4%</span> | 577 | 1065 | <span class="benchmark-delta-negative">+84.6%</span> |
| **Total Input Tokens** | 101149 | 202070 | <span class="benchmark-delta-negative">+99.8%</span> | 144433 | 344931 | <span class="benchmark-delta-negative">+138.8%</span> |
| **Cached Input Tokens** | 92928 | 174080 | <span class="benchmark-delta-positive">+87.3%</span> | 134400 | 314368 | <span class="benchmark-delta-positive">+133.9%</span> |
| **Uncached Input Tokens** | 8221 | 27990 | <span class="benchmark-delta-negative">+240.5%</span> | 10033 | 30563 | <span class="benchmark-delta-negative">+204.6%</span> |
| **Cached vs Uncached Token Ratio** | 11.30:1 | 6.22:1 | <span class="benchmark-delta-negative">-45.0%</span> | 13.40:1 | 10.29:1 | <span class="benchmark-delta-negative">-23.2%</span> |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;AGENTS*&#39; | sort; if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi; printf &#39;\\n--- main.go ---\\n&#39;; sed -n &#39;1,240p&#39; main.go; printf &#39;\\n--- Makefile ---\\n&#39;; if [ -f Makefile ]; then sed -n &#39;1,200p&#39; Makefile; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; | sort; printf &#39;\\n--- main.go ---\\n&#39;; sed -n &#39;1,200p&#39; main.go; printf &#39;\\n--- project docs ---\\n&#39;; if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,200p&#39; PROJECT_BRIEF.md; fi; printf &#39;\\n--- Makefile ---\\n&#39;; if [ -f Makefile ]; then sed -n &#39;1,160p&#39; Makefile; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;value, err := rand.Int(rand.Reader, big.NewInt(100))\n\tif err != nil {\n\t\tlog.Fatal(err)\n\t}\n\n\ttmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\n\tif err := tmpl.Execute(os.Stdout, value); err != nil {\n\t\tlog.Fatal(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go; go test ./...; go run .; git diff --check; git status --short; git diff -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go; git diff -- main.go go.mod go.sum; gofmt -d main.go; go run main.go&#34;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;main.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check; git status --short; git diff -- go.mod go.sum; git diff -- main.go&#39;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- main.go go.mod go.sum; git diff --no-ext-diff -- main.go; test -f go.mod &amp;&amp; \
sha256sum go.mod; test -f go.sum &amp;&amp; \
sha256sum go.sum; go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 20 lines)
* **MCP Edit**: File main.go modified (4 lines -> 27 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then printf &#39;\\n--- PROJECT_BRIEF.md ---\\n&#39;; sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;docs/**&#39; | sort &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files | sort &amp;&amp; \
git status --short &amp;&amp; \
git diff -- main.go&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go build ./... &amp;&amp; \
output=$(go run .) &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; &amp;&amp; \
case &#34;$output&#34; in &#39;&#34;&#39;Hello World &#39;* ) ;; * ) exit 1;; esac &amp;&amp; \
git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff -- main.go go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;const max = 1_000_000\n\nvalue, err := rand.Int(rand.Reader, big.NewInt(max))\nif err != nil {\n\tlog.Fatal(err)\n}\n\ntmpl, err := template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;)\nif err != nil {\n\tlog.Fatal(err)\n}\n\nif err := tmpl.Execute(os.Stdout, value); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --check -- main.go &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;\\n--- main.go diff ---\\n&#39; &amp;&amp; \
git diff -- main.go &amp;&amp; \
printf &#39;\\n--- protected file changes ---\\n&#39; &amp;&amp; \
git diff --name-only -- go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
gofmt -d main.go &amp;&amp; \
go test ./... &amp;&amp; \
output=&#34;&#39;$(go run .) &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; &amp;&amp; \
test &#34;$(printf &#39;&#34;&#39;%s&#39; \&#34;&#34;&#39;$output&#34; | awk &#39;&#34;&#39;{print NF}&#39;)\&#34; -eq 3&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -la &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; main.go &amp;&amp; \
printf &#39;\\n--- tracked status ---\\n&#39; &amp;&amp; \
git status --short -- main.go go.mod go.sum &amp;&amp; \
git check-ignore -v main.go go.mod go.sum || \
true &amp;&amp; \
printf &#39;\\n--- files ---\\n&#39; &amp;&amp; \
git ls-files -- main.go go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;text/template&#34;
  ],
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
gofmt -d main.go &amp;&amp; \
go test ./... &amp;&amp; \
output=&#34;&#39;$(go run .) &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; &amp;&amp; \
case &#34;$output&#34; in &#39;&#34;&#39;Hello World &#39;* ) ;; *) exit 1 ;; esac &amp;&amp; \
git diff --name-only -- main.go go.mod go.sum &amp;&amp; \
git diff -- main.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,40p&#39; main.go &amp;&amp; \
gofmt -d main.go &amp;&amp; \
git status --short -- main.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go test ./... &amp;&amp; \
output=$(go run .) &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; &amp;&amp; \
case &#34;$output&#34; in &#39;&#34;&#39;Hello World &#39;* ) ;; *) exit 1 ;; esac &amp;&amp; \
test -z \&#34;&#34;&#39;$(git diff --name-only -- go.mod go.sum)&#34; &amp;&amp; \
git diff -- main.go&#39;</code></pre></td></tr>
</tbody>
</table>

#### Configuration: default prompt · prescriptive MCP instructions

* **Fixture**: [testdata/scripts/generate_template_main.txtar](https://github.com/spockz/semantic-editor/blob/e16932220de9afe14201fa4af671d1db244f3ca3/testdata/scripts/generate_template_main.txtar)

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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 53.64s | 41.44s | N/A | 37.77s | 103.09s | <span class="benchmark-delta-negative">+172.9%</span> |
| **Process Start → First Event** | 0.90s | 0.83s | — | 0.90s | 0.51s | — |
| **First Event → First Tool Call** | 11.11s | 8.87s | — | 8.19s | 10.45s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | 1 | 1 | 0% |
| **Internal Tool Cycles** | 5 | 2 | N/A | 3 | 10 | <span class="benchmark-delta-negative">+233.3%</span> |
| **Initial Load / Discovery Turns** | 5 | 2 | N/A | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 1 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Tool Invocations** | 5 | 2 | N/A | 3 | 10 | <span class="benchmark-delta-negative">+233.3%</span> |
| **Output Tokens** | 1407 | 1312 | N/A | 1090 | 2846 | <span class="benchmark-delta-negative">+161.1%</span> |
| **Reasoning / Thinking Tokens** | 554 | 525 | N/A | 463 | 1303 | <span class="benchmark-delta-negative">+181.4%</span> |
| **Total Input Tokens** | 142874 | 103206 | N/A | 100442 | 417526 | <span class="benchmark-delta-negative">+315.7%</span> |
| **Cached Input Tokens** | 132352 | 93952 | N/A | 90880 | 384512 | <span class="benchmark-delta-positive">+323.1%</span> |
| **Uncached Input Tokens** | 10522 | 9254 | N/A | 9562 | 33014 | <span class="benchmark-delta-negative">+245.3%</span> |
| **Cached vs Uncached Token Ratio** | 12.58:1 | 10.15:1 | N/A | 9.50:1 | 11.65:1 | <span class="benchmark-delta-positive">+22.5%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;AGENTS*&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project brief candidates ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
printf &#39;\\n--- git status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
rg -n \&#34;Hello World|template|random|rand\&#34; . --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- formatted main.go ---&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- run ---&#39; &amp;&amp; \
GO111MODULE=off go run main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected file status ---&#39; &amp;&amp; \
git status --short -- main.go go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- diff ---&#39; &amp;&amp; \
git diff -- main.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run . &amp;&amp; \
git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td><td>—</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
go test ./...&#39;</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-files --stage -- main.go; sed -n &#39;1,120p&#39; main.go; git diff --no-index /dev/null main.go || \
true&#34;</code></pre></td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 18 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project docs ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,220p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run main.go &amp;&amp; \
git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff -- main.go&#39;</code></pre></td><td><code>cua_repl/js</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;code&#34;: &#34;await cua.getState()&#34;,
  &#34;title&#34;: &#34;Inspect available computer surfaces&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- main.go &amp;&amp; \
git diff --name-only -- go.mod go.sum &amp;&amp; \
git diff --check -- main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl, err := template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\\\n\&#34;)\nif err != nil {\n\tlog.Fatal(err)\n}\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- suboptimal log ---\\n&#39; &amp;&amp; \
if [ -f docs/SUBOPTIMAL_TOOLS.md ]; then sed -n &#39;1,220p&#39; docs/SUBOPTIMAL_TOOLS.md; else echo &#39;(missing)&#39;; fi&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;math/rand&#34;
  ],
  &#34;remove&#34;: [
    &#34;crypto/rand&#34;
  ]
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git diff -- main.go docs/SUBOPTIMAL_TOOLS.md &amp;&amp; \
git diff -- go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;tmpl, err := template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;)\nif err != nil {\n\tlog.Fatal(err)\n}\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -d main.go &amp;&amp; \
go run main.go &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;\\n--- protected file diff ---\\n&#39; &amp;&amp; \
git diff --exit-code -- go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- main.go ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- go test ---&#39; &amp;&amp; \
go test . &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- module files timestamps/diff check ---&#39; &amp;&amp; \
git diff --exit-code -- go.mod go.sum&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not discover any callable `semantic_*` tools from the available semedit MCP server in this session. I therefore used ordinary editing, which was sufficient for this single-file change.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.03s; turns: 1; tool calls: 0.</p>
</details>
