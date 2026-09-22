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

## Test case: `task-07-generate-template-main`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: crypto_rand prompt · none MCP instructions

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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 45.32s | 106.62s | N/A | 69.48s | 38.79s | N/A |
| **Process Start → First Event** | 1.24s | 0.89s | — | 0.87s | 1.04s | — |
| **First Event → First Tool Call** | 7.97s | 18.28s | — | 15.54s | 9.67s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | 1 | 1 | N/A |
| **Internal Tool Cycles** | 6 | 6 | N/A | 3 | 2 | N/A |
| **Initial Load / Discovery Turns** | 6 | 6 | N/A | 3 | 2 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 6 | 6 | N/A | 3 | 2 | N/A |
| **Output Tokens** | 1458 | 2015 | N/A | 1152 | 1191 | N/A |
| **Reasoning / Thinking Tokens** | 573 | 893 | N/A | 484 | 582 | N/A |
| **Total Input Tokens** | 163039 | 145420 | N/A | 101837 | 80317 | N/A |
| **Cached Input Tokens** | 148480 | 116992 | N/A | 83712 | 71680 | N/A |
| **Uncached Input Tokens** | 14559 | 28428 | N/A | 18125 | 8637 | N/A |
| **Cached vs Uncached Token Ratio** | 10.20:1 | 4.12:1 | N/A | 4.62:1 | 8.30:1 | N/A |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 20 lines)
* **MCP Edit**: File main.go modified (4 lines -> 20 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sort &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- project docs ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
printf &#39;\\n--- make targets ---\\n&#39; &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;if [ -f Makefile ]; then make check; else printf &#39;%s\\n&#39; &#39;No Makefile; skipped make check&#39;; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GO111MODULE=off go test .&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go run main.go &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- main.go &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-files --error-unmatch -- main.go &gt;/dev/null 2&gt;&amp;1; printf &#39;tracked=%s\\n&#39; \&#34;&#34;&#39;$?&#34;; sed -n &#39;&#34;&#39;1,120p&#39; main.go; git check-ignore -v main.go || \
true; git diff --no-index /dev/null main.go || \
true&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 21 lines)
* **MCP Edit**: File main.go modified (4 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*&#39; -g &#39;docs/**&#39; | sort &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,220p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
sed -n &#39;1,220p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w main.go &amp;&amp; \
go run . &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run main.go &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git status --short -- main.go go.mod go.sum &amp;&amp; \
git diff --no-ext-diff --no-index /dev/null main.go || \
true&#34;</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not discover any callable `semantic_*` tools from the available semedit MCP server in the tool context. Ordinary editing was therefore used instead.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.19s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Large Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-error"><div class="callout-title"><span>✕</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not call a `semantic_*` tool because no semedit MCP server tools were available in the tool context. I therefore used ordinary editing, which was sufficient for this small change.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 9.34s; turns: 1; tool calls: 0.</p>
</details>

#### Configuration: default prompt · none MCP instructions

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
| **Wall-Clock Latency** | 34.87s | 48.94s | N/A | 50.65s | 74.54s | <span class="benchmark-delta-negative">+47.2%</span> |
| **Process Start → First Event** | 1.37s | 1.14s | — | 0.87s | 1.09s | — |
| **First Event → First Tool Call** | 8.14s | 8.98s | — | 8.54s | 11.56s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 5 | N/A | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> |
| **Initial Load / Discovery Turns** | 3 | 1 | N/A | 4 | 1 | <span class="benchmark-delta-positive">-75.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 5 | N/A | 4 | 7 | <span class="benchmark-delta-negative">+75.0%</span> |
| **Output Tokens** | 987 | 1806 | N/A | 1693 | 2084 | <span class="benchmark-delta-negative">+23.1%</span> |
| **Reasoning / Thinking Tokens** | 348 | 710 | N/A | 871 | 1021 | <span class="benchmark-delta-negative">+17.2%</span> |
| **Total Input Tokens** | 99406 | 146024 | N/A | 123448 | 220074 | <span class="benchmark-delta-negative">+78.3%</span> |
| **Cached Input Tokens** | 81664 | 135424 | N/A | 100864 | 180992 | <span class="benchmark-delta-positive">+79.4%</span> |
| **Uncached Input Tokens** | 17742 | 10600 | N/A | 22584 | 39082 | <span class="benchmark-delta-negative">+73.1%</span> |
| **Cached vs Uncached Token Ratio** | 4.60:1 | 12.78:1 | N/A | 4.47:1 | 4.63:1 | <span class="benchmark-delta-positive">+3.7%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- docs ---\\n&#39; &amp;&amp; \
for f in PROJECT_BRIEF.md docs/adr/README.md docs/research/README.md; do if [ -f \&#34;&#34;&#39;$f&#34; ]; then echo &#34;### $f&#34;; sed -n &#39;&#34;&#39;1,160p&#39; \&#34;&#34;&#39;$f&#34;; fi; done&#39;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
rg -n \&#34;Hello World|random|template\&#34; . --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- go.mod (read-only) ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- status ---&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- available checks ---&#39; &amp;&amp; \
(command -v make || \
true) &amp;&amp; \
(command -v go || \
true) &amp;&amp; \
(command -v semedit || \
true) &amp;&amp; \
(test -f Makefile &amp;&amp; \
sed -n &#39;1,160p&#39; Makefile || \
true)&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go run . &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
go test ./... &amp;&amp; \
output=$(go run .) &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; \&#34;&#34;&#39;$output&#34; | rg -q &#39;&#34;&#39;&#34;&#39;^Hello World -?[0-9]+$&#39;&#34;&#39; &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- main.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- main.go ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- fixture status ---&#39; &amp;&amp; \
git status --short --untracked-files=all -- main.go go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected file hashes ---&#39; &amp;&amp; \
shasum go.mod go.sum 2&gt;/dev/null || \
true&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 16 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- main.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!*go.sum&#39;&#34;&#39; | head -80 &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- main.go ---&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- project brief ---&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- make targets ---&#39; &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- AGENTS.override.md ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; AGENTS.override.md &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- related Go files ---&#39; &amp;&amp; \
sed -n &#39;1,160p&#39; api/router.go &amp;&amp; \
sed -n &#39;1,120p&#39; api/router_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;semantic_(replace_body|insert|organize|verify)|semedit\&#34; . --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | head -120; command -v semedit || \
true; find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w main.go &amp;&amp; \
go test ./... &amp;&amp; \
printf &#39;\\n--- diff ---\\n&#39; &amp;&amp; \
git diff -- main.go &amp;&amp; \
printf &#39;\\n--- protected files status ---\\n&#39; &amp;&amp; \
git status --short -- go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;const source = \&#34;Hello World {{.}}\\n\&#34;\n\ttmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(source))\n\tif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\t\tlog.Fatal(err)\n\t}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;\\n--- tracked state ---\\n&#39; &amp;&amp; \
git ls-files -v main.go go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- go.mod ---&#39; &amp;&amp; \
sed -n &#39;1,80p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- status ---&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
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
if [ -f Makefile ]; then make check; else gofmt -d main.go &amp;&amp; \
go test ./...; fi &amp;&amp; \
git diff -- main.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go run .&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not discover any callable `semantic_*` tools from the available semedit MCP server in the task environment. I therefore used ordinary editing, which was sufficient for this single-file change.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 5.30s; turns: 1; tool calls: 0.</p>
</details>
