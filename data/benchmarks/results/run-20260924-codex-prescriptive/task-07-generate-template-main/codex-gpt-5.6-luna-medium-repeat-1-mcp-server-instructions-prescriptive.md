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

## Test case: `task-07-generate-template-main`

### Target: `codex/gpt-5.6-luna/medium (repeat 1)`

#### Configuration: crypto_rand prompt · prescriptive MCP instructions

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 45.73s | 80.01s | <span class="benchmark-delta-negative">+75.0%</span> | 36.34s | 47.21s | N/A |
| **Process Start → First Event** | 0.47s | 0.48s | — | 0.12s | 0.13s | — |
| **First Event → First Tool Call** | 7.29s | 8.50s | — | 8.22s | 11.22s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | N/A |
| **Internal Tool Cycles** | 4 | 9 | <span class="benchmark-delta-negative">+125.0%</span> | 4 | 4 | N/A |
| **Initial Load / Discovery Turns** | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> | 4 | 1 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 1 | <span class="benchmark-delta-negative">+100.0%</span> | 0 | 0 | 0% |
| **Total Tool Invocations** | 4 | 9 | <span class="benchmark-delta-negative">+125.0%</span> | 4 | 4 | N/A |
| **Output Tokens** | 1444 | 2190 | <span class="benchmark-delta-negative">+51.7%</span> | 1194 | 1635 | N/A |
| **Reasoning / Thinking Tokens** | 641 | 879 | <span class="benchmark-delta-negative">+37.1%</span> | 467 | 717 | N/A |
| **Total Input Tokens** | 121660 | 340143 | <span class="benchmark-delta-negative">+179.6%</span> | 100148 | 122679 | N/A |
| **Cached Input Tokens** | 98816 | 276736 | <span class="benchmark-delta-positive">+180.1%</span> | 74496 | 108032 | N/A |
| **Uncached Input Tokens** | 22844 | 63407 | <span class="benchmark-delta-negative">+177.6%</span> | 25652 | 14647 | N/A |
| **Cached vs Uncached Token Ratio** | 4.33:1 | 4.36:1 | <span class="benchmark-delta-positive">+0.9%</span> | 2.90:1 | 7.38:1 | N/A |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 20 lines)
* **MCP Edit**: File main.go modified (4 lines -> 19 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; | sort &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sed -n &#39;1,80p&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- main.go ---&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- docs ---&#39; &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;docs/README.md&#39; -g &#39;Makefile&#39; -g &#39;AGENTS.md&#39; | sed -n &#39;1,80p&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
make check &amp;&amp; \
go run . &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
git status --short -- main.go go.mod go.sum&#39;</code></pre></td><td><code>cua_repl/js</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;code&#34;: &#34;const s = await cua.getState(); nodeRepl.write(s)&#34;,
  &#34;title&#34;: &#34;Inspect semantic editing surfaces&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go run . &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
git status --short -- main.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World %d\\n\&#34;))\n\trandomValue, err := rand.Int(rand.Reader, big.NewInt(100))\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tif err := tmpl.Execute(os.Stdout, randomValue.Int64()); err != nil {\n\t\tpanic(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git status --short &amp;&amp; \
git diff --no-ext-diff -- main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- status ---&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- diff ---&#39; &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- compile/run ---&#39; &amp;&amp; \
GO111MODULE=off go run main.go&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\n\trandomValue, err := rand.Int(rand.Reader, big.NewInt(100))\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tif err := tmpl.Execute(os.Stdout, randomValue.Int64()); err != nil {\n\t\tpanic(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;text/template&#34;
  ],
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- run ---&#39; &amp;&amp; \
GO111MODULE=off go run main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected files ---&#39; &amp;&amp; \
git diff --quiet -- go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;protected files unchanged&#39; || \
exit 1 &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tracked status for fixture ---&#39; &amp;&amp; \
git status --short -- main.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;remove&#34;: [
    &#34;crypto/rand&#34;,
    &#34;math/big&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ],
  &#34;add&#34;: [
    &#34;crypto/rand&#34;,
    &#34;math/big&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; main.go &amp;&amp; \
GO111MODULE=off gofmt -d main.go &amp;&amp; \
GO111MODULE=off go run main.go &gt;/tmp/semedit-main-output &amp;&amp; \
sed -n &#39;1p&#39; /tmp/semedit-main-output &amp;&amp; \
case \&#34;&#34;&#39;$(sed -n &#39;&#34;&#39;1p&#39; /tmp/semedit-main-output)\&#34; in &#39;Hello World &#39;* ) ;; * ) exit 1 ;; esac &amp;&amp; \
git diff --quiet -- go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;DONE&#39;&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 20 lines)
* **MCP Edit**: File main.go modified (4 lines -> 24 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | head -80 &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- Makefile/package files ---\\n&#39; &amp;&amp; \
rg --files -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sed -n &#39;1,80p&#39; &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- docs candidates ---\\n&#39; &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*.md&#39; | sed -n &#39;1,80p&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- available semantic tooling ---&#39; &amp;&amp; \
command -v semedit || \
true &amp;&amp; \
command -v semantic-editor || \
true &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- fixture files ---&#39; &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git status --short -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
git diff --check &amp;&amp; \
go test ./... &amp;&amp; \
go run .&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go run .&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go go.mod go.sum &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not discover any callable `semantic_*` tools in the available tool context, and no semedit CLI was present in the fixture. I therefore used ordinary editing, which was sufficient for this minimal `main.go` change.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 5.88s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was not used because the semantic editing tools were discovered individually and the operations were executed sequentially to correct and verify each change. No further file changes or tool calls were made.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.11s; turns: 1; tool calls: 0.</p>
</details>

#### Configuration: default prompt · prescriptive MCP instructions

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
| **Wall-Clock Latency** | 28.51s | 67.42s | <span class="benchmark-delta-negative">+136.4%</span> | 50.37s | 71.65s | <span class="benchmark-delta-negative">+42.3%</span> |
| **Process Start → First Event** | 0.12s | 0.44s | — | 0.13s | 0.12s | — |
| **First Event → First Tool Call** | 5.85s | 9.98s | — | 11.50s | 7.09s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> |
| **Initial Load / Discovery Turns** | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> |
| **Output Tokens** | 1056 | 1982 | <span class="benchmark-delta-negative">+87.7%</span> | 1507 | 2028 | <span class="benchmark-delta-negative">+34.6%</span> |
| **Reasoning / Thinking Tokens** | 509 | 793 | <span class="benchmark-delta-negative">+55.8%</span> | 715 | 738 | <span class="benchmark-delta-negative">+3.2%</span> |
| **Total Input Tokens** | 100080 | 367982 | <span class="benchmark-delta-negative">+267.7%</span> | 121096 | 416483 | <span class="benchmark-delta-negative">+243.9%</span> |
| **Cached Input Tokens** | 82688 | 338944 | <span class="benchmark-delta-positive">+309.9%</span> | 87552 | 381440 | <span class="benchmark-delta-positive">+335.7%</span> |
| **Uncached Input Tokens** | 17392 | 29038 | <span class="benchmark-delta-negative">+67.0%</span> | 33544 | 35043 | <span class="benchmark-delta-negative">+4.5%</span> |
| **Cached vs Uncached Token Ratio** | 4.75:1 | 11.67:1 | <span class="benchmark-delta-positive">+145.5%</span> | 2.61:1 | 10.88:1 | <span class="benchmark-delta-positive">+317.0%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*AGENTS.override.md&#39;&#34;&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;PROJECT_BRIEF.md&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*.md&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;; printf &#39;\\n--- main.go ---\\n&#39;; sed -n &#39;1,160p&#39; main.go; printf &#39;\\n--- project brief ---\\n&#39;; if test -f PROJECT_BRIEF.md; then sed -n &#39;1,200p&#39; PROJECT_BRIEF.md; fi; printf &#39;\\n--- status ---\\n&#39;; git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go go.mod go.sum &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go; printf &#39;\\n--- manifests ---\\n&#39;; ls -1 go.mod go.sum Makefile 2&gt;/dev/null || \
true; printf &#39;\\n--- checks ---\\n&#39;; rg -n &#39;&#34;&#39;^(check|test|fmt)|make check&#39;&#34;&#39; Makefile 2&gt;/dev/null || \
true&#34;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
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
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go; printf &#39;\\n--- go env/version ---\\n&#39;; go version; printf &#39;\\n--- diff ---\\n&#39;; git diff -- main.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go run .; printf &#39;\\n--- semantic verification ---\\n&#39;&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;main.go&#34;
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- files ---\\n&#39; &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | head -50&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sort &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
if [ -f main.go ]; then sed -n &#39;1,240p&#39; main.go; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run main.go &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- main.go &amp;&amp; \
git status --short -- go.mod go.sum main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;t := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := t.Execute(os.Stdout, rand.Int()); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git status --short &amp;&amp; \
git ls-files --stage -- main.go &amp;&amp; \
git diff --no-index /dev/null main.go || \
true&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git diff -- main.go&#34;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;t := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := t.Execute(os.Stdout, mathrand.Int()); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go run main.go &amp;&amp; \
git diff --check &amp;&amp; \
git status --short -- go.mod go.sum main.go&#39;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git diff --stat &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was unsuitable because the operations required sequential correction: the initial body edit inferred incorrect imports, so I needed to revise the body and then explicitly add/remove imports. I used standalone semantic operations to inspect and resolve each intermediate diagnostic.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 4.20s; turns: 1; tool calls: 0.</p>
</details>

#### Configuration: prefer_discover_semedit prompt · prescriptive MCP instructions

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
| **Wall-Clock Latency** | 61.92s | 63.01s | <span class="benchmark-delta-negative">+1.8%</span> | 54.84s | 88.56s | <span class="benchmark-delta-negative">+61.5%</span> |
| **Process Start → First Event** | 0.12s | 0.12s | — | 0.13s | 0.14s | — |
| **First Event → First Tool Call** | 13.92s | 12.58s | — | 15.59s | 14.71s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> | 5 | 13 | <span class="benchmark-delta-negative">+160.0%</span> |
| **Initial Load / Discovery Turns** | 4 | 1 | <span class="benchmark-delta-positive">-75.0%</span> | 5 | 4 | <span class="benchmark-delta-positive">-20.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 2 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Tool Invocations** | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> | 5 | 13 | <span class="benchmark-delta-negative">+160.0%</span> |
| **Output Tokens** | 1618 | 1773 | <span class="benchmark-delta-negative">+9.6%</span> | 1789 | 2760 | <span class="benchmark-delta-negative">+54.3%</span> |
| **Reasoning / Thinking Tokens** | 560 | 868 | <span class="benchmark-delta-negative">+55.0%</span> | 831 | 1264 | <span class="benchmark-delta-negative">+52.1%</span> |
| **Total Input Tokens** | 171967 | 228466 | <span class="benchmark-delta-negative">+32.9%</span> | 198598 | 503850 | <span class="benchmark-delta-negative">+153.7%</span> |
| **Cached Input Tokens** | 129024 | 203776 | <span class="benchmark-delta-positive">+57.9%</span> | 179968 | 466176 | <span class="benchmark-delta-positive">+159.0%</span> |
| **Uncached Input Tokens** | 42943 | 24690 | <span class="benchmark-delta-positive">-42.5%</span> | 18630 | 37674 | <span class="benchmark-delta-negative">+102.2%</span> |
| **Cached vs Uncached Token Ratio** | 3.00:1 | 8.25:1 | <span class="benchmark-delta-positive">+174.7%</span> | 9.66:1 | 12.37:1 | <span class="benchmark-delta-positive">+28.1%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;*.md&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort | head -200 &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project docs ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
printf &#39;\\n--- Makefile targets ---\\n&#39; &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; AGENTS.override.md; printf &#39;\\n--- main.go ---\\n&#39;; sed -n &#39;1,200p&#39; main.go; printf &#39;\\n--- go.mod ---\\n&#39;; sed -n &#39;1,120p&#39; go.mod; printf &#39;\\n--- git status ---\\n&#39;; git status --short&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w main.go &amp;&amp; \
go run . &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short -- main.go go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go; git diff --no-ext-diff --no-textconv -- main.go; git ls-files --stage -- main.go; git check-ignore -v main.go || \
true&#34;</code></pre></td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;math/rand&#34;
  ],
  &#34;remove&#34;: [
    &#34;crypto/rand&#34;
  ]
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go run main.go&#39;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go go.mod go.sum &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;Makefile&#39; -g &#39;main.go&#39; -g &#39;README*&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39;; if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi; if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; fi; sed -n &#39;1,220p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;Makefile&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; | sort&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort; git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- go.mod ---\\n&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
printf &#39;\\n--- files ---\\n&#39; &amp;&amp; \
rg --files | sort&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go; go test ./...; go run .&#39;</code></pre></td><td><code>cua_repl/js</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;code&#34;: &#34;const s = await cua.getState(); nodeRepl.write(JSON.stringify(s));&#34;,
  &#34;title&#34;: &#34;Inspect connected UI surfaces&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go go.mod go.sum; git status --short -- main.go go.mod go.sum&#39;</code></pre></td><td><code>cua_repl/js</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;code&#34;: &#34;nodeRepl.write(&#39;&#39;);&#34;,
  &#34;title&#34;: &#34;Continue with repository editing&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go; git diff --no-index /dev/null main.go || \
true&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tpanic(err)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git diff -- main.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
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
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;main.go&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go run . &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- main.go&#39;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel &amp;&amp; \
git ls-files --error-unmatch main.go &amp;&amp; \
git diff --no-ext-diff -- main.go &amp;&amp; \
git status --short --untracked-files=all -- main.go go.mod go.sum&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because it was not necessary for the small sequential edits: the body replacement exposed an import mismatch that required inspecting the result before choosing the corrective import operation. Batching would have obscured that dependency and was therefore unsuitable here.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.30s; turns: 1; tool calls: 0.</p>
</details>

