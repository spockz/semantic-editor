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

* **Date**: 2026-09-22 12:02:22 CEST

## Test case: `task-07-generate-template-main`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: crypto_rand prompt · descriptive MCP instructions

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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 43.86s | 79.15s | N/A | 48.56s | 116.85s | <span class="benchmark-delta-negative">+140.6%</span> |
| **Process Start → First Event** | 0.65s | 0.71s | — | 0.88s | 0.88s | — |
| **First Event → First Tool Call** | 8.86s | 12.00s | — | 7.49s | 9.03s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 3 | N/A | 5 | 11 | <span class="benchmark-delta-negative">+120.0%</span> |
| **Initial Load / Discovery Turns** | 3 | 3 | N/A | 5 | 1 | <span class="benchmark-delta-positive">-80.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 3 | N/A | 5 | 11 | <span class="benchmark-delta-negative">+120.0%</span> |
| **Output Tokens** | 1361 | 1574 | N/A | 1489 | 3854 | <span class="benchmark-delta-negative">+158.8%</span> |
| **Reasoning / Thinking Tokens** | 448 | 647 | N/A | 633 | 1505 | <span class="benchmark-delta-negative">+137.8%</span> |
| **Total Input Tokens** | 122384 | 121989 | N/A | 143858 | 371352 | <span class="benchmark-delta-negative">+158.1%</span> |
| **Cached Input Tokens** | 102912 | 102912 | N/A | 134400 | 349440 | <span class="benchmark-delta-positive">+160.0%</span> |
| **Uncached Input Tokens** | 19472 | 19077 | N/A | 9458 | 21912 | <span class="benchmark-delta-negative">+131.7%</span> |
| **Cached vs Uncached Token Ratio** | 5.29:1 | 5.39:1 | N/A | 14.21:1 | 15.95:1 | <span class="benchmark-delta-positive">+12.2%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 20 lines)
* **MCP Edit**: File main.go modified (4 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;; sed -n &#39;1,200p&#39; main.go; if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,200p&#39; PROJECT_BRIEF.md; fi; if [ -f Makefile ]; then sed -n &#39;1,160p&#39; Makefile; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go; go test ./...; go run .; git diff --check; git status --short; git diff -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w main.go &amp;&amp; \
go run . &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- main.go &amp;&amp; \
printf &#39;\\n--- protected file status ---\\n&#39; &amp;&amp; \
git status --short -- go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; main.go; git diff --no-ext-diff -- main.go; git status --short -- main.go go.mod go.sum; git check-ignore -v main.go go.mod go.sum || \
true&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;\\n--- tracked diff ---\\n&#39; &amp;&amp; \
git diff --no-ext-diff -- main.go&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 20 lines)
* **MCP Edit**: File main.go modified (4 lines -> 20 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- main.go ---&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- project docs ---&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,200p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Hello|crypto/rand|text/template|func main|rand.Int\&#34; . --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;randomInt, err := rand.Int(rand.Reader, big.NewInt(1_000_000))\nif err != nil {\n\tpanic(err)\n}\n\ntmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, randomInt); err != nil {\n\tpanic(err)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w main.go &amp;&amp; \
if [ -f Makefile ]; then make check; else go test ./...; fi &amp;&amp; \
printf &#39;\\n--- output ---\\n&#39; &amp;&amp; \
go run .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- diff ---&#39; &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- checks ---&#39; &amp;&amp; \
if [ -f Makefile ]; then rg -n &#39;&#34;&#39;^(check|test|build|fmt|lint):&#39;&#34;&#39; Makefile; else printf &#39;%s\\n&#39; &#39;no Makefile&#39;; fi&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go go.mod go.sum &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;if [ -f docs/SUBOPTIMAL_TOOLS.md ]; then sed -n &#39;1,160p&#39; docs/SUBOPTIMAL_TOOLS.md; else printf &#39;%s\\n&#39; &#39;missing&#39;; fi&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- main.go go.mod go.sum &amp;&amp; \
sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git diff --no-ext-diff -- main.go&#34;</code></pre></td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;text/template&#34;
  ],
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;GOFLAGS=&#39;-mod=readonly&#39; go test ./... &amp;&amp; \
output=&#34;&#39;$(GOFLAGS=&#39;&#34;&#39;-mod=readonly&#39; go run .) &amp;&amp; \
printf &#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; &amp;&amp; \
case &#34;$output&#34; in &#39;&#34;&#39;Hello World &#39;* ) : ;; *) exit 1 ;; esac &amp;&amp; \
git diff --check &amp;&amp; \
git status --short -- go.mod go.sum main.go docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --check -- main.go docs/SUBOPTIMAL_TOOLS.md &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- main.go ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- status ---&#39; &amp;&amp; \
git status --short -- go.mod go.sum main.go docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- main.go docs/SUBOPTIMAL_TOOLS.md &amp;&amp; \
git ls-files -v main.go docs/SUBOPTIMAL_TOOLS.md&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- paths ---&#39; &amp;&amp; \
pwd &amp;&amp; \
ls -ld . docs docs/SUBOPTIMAL_TOOLS.md main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- repo root ---&#39; &amp;&amp; \
git rev-parse --show-toplevel &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tracked state ---&#39; &amp;&amp; \
git ls-files --error-unmatch main.go 2&gt;&amp;1 || \
true &amp;&amp; \
git check-ignore -v main.go docs/SUBOPTIMAL_TOOLS.md 2&gt;&amp;1 || \
true &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- fixture docs ---&#39; &amp;&amp; \
sed -n &#39;1,80p&#39; docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w main.go &amp;&amp; \
GOFLAGS=&#39;-mod=readonly&#39; go test ./... &gt;/tmp/semedit-test-output &amp;&amp; \
tail -n 5 /tmp/semedit-test-output &amp;&amp; \
sed -n &#39;1,120p&#39; main.go &amp;&amp; \
if [ -e go.mod ] &amp;&amp; \
[ -e go.sum ]; then git diff --no-index --exit-code -- go.mod go.mod &gt;/dev/null 2&gt;&amp;1 || \
true; fi&#34;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w main.go &amp;&amp; \
GOFLAGS=&#39;-mod=readonly&#39; go test ./... &gt;/tmp/semedit-test-output &amp;&amp; \
output=&#34;&#39;$(GOFLAGS=&#39;&#34;&#39;-mod=readonly&#39; go run .) &amp;&amp; \
case \&#34;&#34;&#39;$output&#34; in &#39;&#34;&#39;Hello World &#39;* ) printf &#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; ;; *) exit 1 ;; esac&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-error"><div class="callout-title"><span>✕</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not call a `semantic_*` tool because no semedit MCP tools were available in the exposed tool set. Ordinary editing was sufficient for this small standalone `main.go` change.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 5.61s; turns: 1; tool calls: 0.</p>
</details>

#### Configuration: default prompt · descriptive MCP instructions

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 43.87s | 71.71s | <span class="benchmark-delta-negative">+63.5%</span> | 31.13s | 53.23s | N/A |
| **Process Start → First Event** | 0.83s | 0.43s | — | 0.70s | 0.54s | — |
| **First Event → First Tool Call** | 7.81s | 7.69s | — | 7.39s | 8.66s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | N/A |
| **Internal Tool Cycles** | 5 | 9 | <span class="benchmark-delta-negative">+80.0%</span> | 2 | 6 | N/A |
| **Initial Load / Discovery Turns** | 5 | 2 | <span class="benchmark-delta-positive">-60.0%</span> | 2 | 2 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 5 | 9 | <span class="benchmark-delta-negative">+80.0%</span> | 2 | 6 | N/A |
| **Output Tokens** | 1376 | 1798 | <span class="benchmark-delta-negative">+30.7%</span> | 938 | 1750 | N/A |
| **Reasoning / Thinking Tokens** | 592 | 765 | <span class="benchmark-delta-negative">+29.2%</span> | 488 | 681 | N/A |
| **Total Input Tokens** | 148300 | 274552 | <span class="benchmark-delta-negative">+85.1%</span> | 79563 | 190716 | N/A |
| **Cached Input Tokens** | 134400 | 251136 | <span class="benchmark-delta-positive">+86.9%</span> | 67584 | 167680 | N/A |
| **Uncached Input Tokens** | 13900 | 23416 | <span class="benchmark-delta-negative">+68.5%</span> | 11979 | 23036 | N/A |
| **Cached vs Uncached Token Ratio** | 9.67:1 | 10.72:1 | <span class="benchmark-delta-positive">+10.9%</span> | 5.64:1 | 7.28:1 | N/A |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 15 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;docs/**&#39; | sort&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run .&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go &amp;&amp; \
git status --short -- go.mod go.sum main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git status --short &amp;&amp; \
git diff --no-ext-diff --no-color&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git check-ignore -v main.go || \
true; git ls-files --stage -- main.go; git diff --no-index /dev/null main.go || \
true&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;test -f docs/SUBOPTIMAL_TOOLS.md &amp;&amp; \
sed -n &#39;1,80p&#39; docs/SUBOPTIMAL_TOOLS.md || \
true&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
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
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
gofmt -d main.go &amp;&amp; \
go run main.go &amp;&amp; \
git diff -- main.go&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;main.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- main.go go.mod go.sum &amp;&amp; \
git diff --check -- main.go&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | head -50 &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run main.go &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;ls -la &amp;&amp; \
command -v semedit || \
true &amp;&amp; \
command -v gofmt || \
true&#39;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; go.mod &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run . &amp;&amp; \
git diff -- main.go go.mod go.sum; git status --short -- main.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git diff --no-index /dev/null main.go || \
true&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-error"><div class="callout-title"><span>✕</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not call a `semantic_*` tool because no semantic-editor CLI or MCP tool was discoverable in the fixture environment. Ordinary editing was therefore the available option.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 4.78s; turns: 1; tool calls: 0.</p>
</details>
