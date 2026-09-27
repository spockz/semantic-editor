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

* **Date**: 2026-09-27 01:46:00 CEST

## Test case: `task-07-generate-template-main`

### Target: `codex/gpt-6-luna/medium (repeat 5)`

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
| **Wall-Clock Latency** | 18.77s | 35.31s | <span class="benchmark-delta-negative">+88.1%</span> | — | — | — |
| **Process Start → First Event** | 0.12s | 0.12s | — | — | — | — |
| **First Event → First Tool Call** | 4.59s | 4.66s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 2 | 6 | <span class="benchmark-delta-negative">+200.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 2 | 6 | <span class="benchmark-delta-negative">+200.0%</span> | — | — | — |
| **Output Tokens** | 599 | 668 | <span class="benchmark-delta-negative">+11.5%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 252 | 186 | <span class="benchmark-delta-positive">-26.2%</span> | — | — | — |
| **Total Input Tokens** | 76176 | 145612 | <span class="benchmark-delta-negative">+91.2%</span> | — | — | — |
| **Cached Input Tokens** | 66560 | 131328 | <span class="benchmark-delta-positive">+97.3%</span> | — | — | — |
| **Uncached Input Tokens** | 9616 | 14284 | <span class="benchmark-delta-negative">+48.5%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.92:1 | 9.19:1 | <span class="benchmark-delta-positive">+32.8%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 21 lines)
* **MCP Edit**: File main.go modified (4 lines -> 20 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;semantic_(lookup|inspect_symbol|replace_body|organize_imports|verify)|text/template|crypto/rand&#34; .&#39;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;n, err := rand.Int(rand.Reader, big.NewInt(100))\nif err != nil {\n\tpanic(err)\n}\n\ntmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, n); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;crypto/rand&#34;,
    &#34;math/big&#34;,
    &#34;os&#34;,
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not discover `semantic_batch` because it was absent from the semantic tools I inspected. The two operations were therefore issued separately.</div></div>

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
| **Wall-Clock Latency** | 17.86s | 29.85s | <span class="benchmark-delta-negative">+67.2%</span> | — | — | — |
| **Process Start → First Event** | 0.12s | 0.13s | — | — | — | — |
| **First Event → First Tool Call** | 5.49s | 5.17s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> | — | — | — |
| **Output Tokens** | 559 | 638 | <span class="benchmark-delta-negative">+14.1%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 257 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Total Input Tokens** | 76314 | 128477 | <span class="benchmark-delta-negative">+68.4%</span> | — | — | — |
| **Cached Input Tokens** | 66560 | 114176 | <span class="benchmark-delta-positive">+71.5%</span> | — | — | — |
| **Uncached Input Tokens** | 9754 | 14301 | <span class="benchmark-delta-negative">+46.6%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.82:1 | 7.98:1 | <span class="benchmark-delta-positive">+17.0%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 16 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;Makefile&#39; -g &#39;README*&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md &amp;&amp; \
cat main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md &amp;&amp; \
cat main.go&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
cat main.go&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;greeting := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := greeting.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
go run .&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I didn’t use `semantic_batch` because the edit was a single `semantic_replace_body` call; import organization was handled by that operation’s `auto_organize_imports` option. Batching was unnecessary for these operations.</div></div>

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
| **Wall-Clock Latency** | 16.41s | 66.36s | <span class="benchmark-delta-negative">+304.4%</span> | — | — | — |
| **Process Start → First Event** | 0.12s | 0.12s | — | — | — | — |
| **First Event → First Tool Call** | 6.10s | 6.11s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 2 | 9 | <span class="benchmark-delta-negative">+350.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 2 | 9 | <span class="benchmark-delta-negative">+350.0%</span> | — | — | — |
| **Output Tokens** | 518 | 1718 | <span class="benchmark-delta-negative">+231.7%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 204 | 619 | <span class="benchmark-delta-negative">+203.4%</span> | — | — | — |
| **Total Input Tokens** | 98432 | 220436 | <span class="benchmark-delta-negative">+123.9%</span> | — | — | — |
| **Cached Input Tokens** | 87808 | 196608 | <span class="benchmark-delta-positive">+123.9%</span> | — | — | — |
| **Uncached Input Tokens** | 10624 | 23828 | <span class="benchmark-delta-negative">+124.3%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.27:1 | 8.25:1 | <span class="benchmark-delta-negative">-0.2%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 22 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;README*&#39; | sort&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl, err := template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;)\nif err != nil {\n\tpanic(err)\n}\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl, err := template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;)\nif err != nil {\n\tpanic(err)\n}\nvalue, err := rand.Int(rand.Reader, big.NewInt(1000000000))\nif err != nil {\n\tpanic(err)\n}\nif err := tmpl.Execute(os.Stdout, value); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;(&#34;&#39;^|/)SUBOPTIMAL_TOOLS&#39;&#34;\\.md&#34;&#39;$|(&#39;&#39;^|/)main&#39;&#34;\\.go&#34;&#39;$&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 -c &#39;from pathlib import Path; import os, tempfile; p=Path(\&#34;docs/SUBOPTIMAL_TOOLS.md\&#34;); p.parent.mkdir(parents=True, exist_ok=True); existing=p.read_text() if p.exists() else \&#34;# Suboptimal Tool Behavior\\n\&#34;; entry=\&#34;\\n## semantic_replace_body: main.go\\n\\n- Observed: The first replacement used &#34;&#39;`rand.Int()` and semantic diagnostics reported that the resolved `crypto/rand.Int` requires an `io.Reader` and `*big.Int`.&#39;&#34;\\n- Workaround: Replaced the body with &#34;&#39;`rand.Int(rand.Reader, big.NewInt(1000000000))` and checked the tool diagnostics again.&#39;&#34;\\n- Root cause: Import resolution selected &#34;&#39;`crypto/rand`; its `Int` API differs from the zero-argument math/rand function.&#39;&#34;\\n\&#34;; data=(existing.rstrip()+\&#34;\\n\&#34;+entry).encode(); fd,tmp=tempfile.mkstemp(prefix=\&#34;.SUBOPTIMAL_TOOLS.\&#34;, dir=str(p.parent)); f=os.fdopen(fd,\&#34;wb\&#34;); f.write(data); f.flush(); os.fsync(f.fileno()); f.close(); os.replace(tmp,p); d=os.open(p.parent,os.O_RDONLY); os.fsync(d); os.close(d)&#39;&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because it was unsuitable here: the first body replacement produced a diagnostic revealing the resolved `rand.Int` API, so I needed to inspect the result and adapt the implementation before making the next replacement. Batching was discovered in the available tools.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

