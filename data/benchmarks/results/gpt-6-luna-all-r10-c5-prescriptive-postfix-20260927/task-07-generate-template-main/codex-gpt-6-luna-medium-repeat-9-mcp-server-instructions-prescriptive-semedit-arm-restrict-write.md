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

* **Date**: 2026-09-27 03:12:40 CEST

## Test case: `task-07-generate-template-main`

### Target: `codex/gpt-6-luna/medium (repeat 9)`

#### Configuration: crypto_rand prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: [testdata/scripts/generate_template_main.txtar](https://github.com/spockz/semantic-editor/blob/c03f3625ef877adfd183a978ca333ca208eff0f0/testdata/scripts/generate_template_main.txtar)

**Vanilla LLM Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer from crypto/rand, ensuring all necessary standard library packages are imported cleanly. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer from crypto/rand, ensuring all necessary standard library packages are imported cleanly. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package main

func main() {
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 19.69s | 45.28s | <span class="benchmark-delta-negative">+129.9%</span> | — | — | — |
| **Process Start → First Event** | 0.12s | 0.13s | — | — | — | — |
| **First Event → First Tool Call** | 4.43s | 5.84s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 3 | 7 | <span class="benchmark-delta-negative">+133.3%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 3 | 7 | <span class="benchmark-delta-negative">+133.3%</span> | — | — | — |
| **Output Tokens** | 535 | 887 | <span class="benchmark-delta-negative">+65.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 122 | 309 | <span class="benchmark-delta-negative">+153.3%</span> | — | — | — |
| **Total Input Tokens** | 95771 | 186809 | <span class="benchmark-delta-negative">+95.1%</span> | — | — | — |
| **Cached Input Tokens** | 85760 | 168960 | <span class="benchmark-delta-positive">+97.0%</span> | — | — | — |
| **Uncached Input Tokens** | 10011 | 17849 | <span class="benchmark-delta-negative">+78.3%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.57:1 | 9.47:1 | <span class="benchmark-delta-positive">+10.5%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 21 lines)
* **MCP Edit**: File main.go modified (4 lines -> 24 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md &amp;&amp; \
cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;randomBytes := make([]byte, 8)\n if _, err := rand.Read(randomBytes); err != nil {\n  panic(err)\n }\n randomInt := binary.BigEndian.Uint64(randomBytes)\n tmpl, err := template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\&#34;)\n if err != nil {\n  panic(err)\n }\n if err := tmpl.Execute(os.Stdout, randomInt); err != nil {\n  panic(err)\n }&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;crypto/rand&#34;,
    &#34;encoding/binary&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
if [ -f docs/SUBOPTIMAL_TOOLS.md ]; then sed -n &#39;1,180p&#39; docs/SUBOPTIMAL_TOOLS.md; else printf &#39;missing\\n&#39;; fi &amp;&amp; \
go run .&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was not discovered before making the consecutive calls. I used separate semantic operations and won’t retry the task.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Configuration: default prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: [testdata/scripts/generate_template_main.txtar](https://github.com/spockz/semantic-editor/blob/c03f3625ef877adfd183a978ca333ca208eff0f0/testdata/scripts/generate_template_main.txtar)

**Vanilla LLM Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package main

func main() {
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 13.89s | 42.63s | <span class="benchmark-delta-negative">+206.9%</span> | — | — | — |
| **Process Start → First Event** | 0.12s | 0.12s | — | — | — | — |
| **First Event → First Tool Call** | 5.09s | 4.30s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 3 | 9 | <span class="benchmark-delta-negative">+200.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 3 | 9 | <span class="benchmark-delta-negative">+200.0%</span> | — | — | — |
| **Output Tokens** | 390 | 678 | <span class="benchmark-delta-negative">+73.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 71 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **Total Input Tokens** | 75907 | 280052 | <span class="benchmark-delta-negative">+268.9%</span> | — | — | — |
| **Cached Input Tokens** | 66560 | 258304 | <span class="benchmark-delta-positive">+288.1%</span> | — | — | — |
| **Uncached Input Tokens** | 9347 | 21748 | <span class="benchmark-delta-negative">+132.7%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 7.12:1 | 11.88:1 | <span class="benchmark-delta-positive">+66.8%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 15 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;AGENTS.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;SUBOPTIMAL_TOOLS.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;tmp=$(mktemp ./.main.go.XXXXXX) &amp;&amp; \
cat &gt; &#34;$tmp&#34; &lt;&lt;&#39;&#34;&#39;EOF&#39;
package main

import (
	\&#34;log\&#34;
	\&#34;math/rand\&#34;
	\&#34;os\&#34;
	\&#34;text/template\&#34;
)

func main() {
	tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))
	if err := tmpl.Execute(os.Stdout, rand.Int()); err &#34;&#39;!= nil {
		log.Fatal(err)
	}
}
EOF
sync &#34;$tmp&#34; &amp;&amp; \
mv &#34;$tmp&#34; main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\n\tif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\t\tpanic(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;math/rand&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ],
  &#34;add&#34;: [
    &#34;texttemplate text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := texttemplate.Must(texttemplate.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\n\tif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\t\tpanic(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;texttemplate text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
go build ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I discovered `semantic_batch`, but failed to use it for the dependent edits. The function-body replacement, import organization, and follow-up body correction should have been combined into an ordered batch; I used separate calls instead.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Configuration: prefer_discover_semedit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: [testdata/scripts/generate_template_main.txtar](https://github.com/spockz/semantic-editor/blob/c03f3625ef877adfd183a978ca333ca208eff0f0/testdata/scripts/generate_template_main.txtar)

**Vanilla LLM Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package main

func main() {
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 14.09s | 33.60s | <span class="benchmark-delta-negative">+138.4%</span> | — | — | — |
| **Process Start → First Event** | 0.12s | 0.12s | — | — | — | — |
| **First Event → First Tool Call** | 5.63s | 5.49s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 1 | 5 | <span class="benchmark-delta-negative">+400.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 1 | 5 | <span class="benchmark-delta-negative">+400.0%</span> | — | — | — |
| **Output Tokens** | 465 | 757 | <span class="benchmark-delta-negative">+62.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 183 | 239 | <span class="benchmark-delta-negative">+30.6%</span> | — | — | — |
| **Total Input Tokens** | 104146 | 196446 | <span class="benchmark-delta-negative">+88.6%</span> | — | — | — |
| **Cached Input Tokens** | 84992 | 174336 | <span class="benchmark-delta-positive">+105.1%</span> | — | — | — |
| **Uncached Input Tokens** | 19154 | 22110 | <span class="benchmark-delta-negative">+15.4%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.44:1 | 7.88:1 | <span class="benchmark-delta-positive">+77.7%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
go build ./...&#39;</code></pre></td></tr>
</tbody>
</table>

