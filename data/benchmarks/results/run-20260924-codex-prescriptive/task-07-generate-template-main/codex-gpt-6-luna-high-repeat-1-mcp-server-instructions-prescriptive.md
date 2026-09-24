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

### Target: `codex/gpt-6-luna/high (repeat 1)`

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 36.14s | 73.74s | <span class="benchmark-delta-negative">+104.0%</span> | 53.49s | 80.79s | <span class="benchmark-delta-negative">+51.0%</span> |
| **Process Start → First Event** | 0.13s | 0.12s | — | 0.12s | 0.12s | — |
| **First Event → First Tool Call** | 4.81s | 6.66s | — | 7.65s | 6.70s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 4 | 8 | <span class="benchmark-delta-negative">+100.0%</span> | 3 | 10 | <span class="benchmark-delta-negative">+233.3%</span> |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 4 | 8 | <span class="benchmark-delta-negative">+100.0%</span> | 3 | 10 | <span class="benchmark-delta-negative">+233.3%</span> |
| **Output Tokens** | 1356 | 1640 | <span class="benchmark-delta-negative">+20.9%</span> | 1047 | 2290 | <span class="benchmark-delta-negative">+118.7%</span> |
| **Reasoning / Thinking Tokens** | 778 | 989 | <span class="benchmark-delta-negative">+27.1%</span> | 466 | 1595 | <span class="benchmark-delta-negative">+242.3%</span> |
| **Total Input Tokens** | 97013 | 192564 | <span class="benchmark-delta-negative">+98.5%</span> | 99710 | 244881 | <span class="benchmark-delta-negative">+145.6%</span> |
| **Cached Input Tokens** | 85760 | 177920 | <span class="benchmark-delta-positive">+107.5%</span> | 88832 | 226560 | <span class="benchmark-delta-positive">+155.0%</span> |
| **Uncached Input Tokens** | 11253 | 14644 | <span class="benchmark-delta-negative">+30.1%</span> | 10878 | 18321 | <span class="benchmark-delta-negative">+68.4%</span> |
| **Cached vs Uncached Token Ratio** | 7.62:1 | 12.15:1 | <span class="benchmark-delta-positive">+59.4%</span> | 8.17:1 | 12.37:1 | <span class="benchmark-delta-positive">+51.4%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 23 lines)
* **MCP Edit**: File main.go modified (4 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;docs/**&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os
import tempfile

path = Path(&#39;main.go&#39;)
source = &#39;&#39;&#39;package main

import (
	\&#34;crypto/rand\&#34;
	\&#34;math/big\&#34;
	\&#34;os\&#34;
	\&#34;text/template\&#34;
)

func main() {
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err &#34;&#39;!= nil {
		panic(err)
	}

	tmpl, err := template.New(&#34;greeting&#34;).Parse(&#34;Hello World {{.}}&#39;&#34;\\\\n\&#34;)
	if err &#34;&#39;!= nil {
		panic(err)
	}
	if err := tmpl.Execute(os.Stdout, n); err != nil {
		panic(err)
	}
}
&#39;&#34;&#39;&#39;&#39;
old_mtime = path.stat().st_mtime_ns
fd, temp_name = tempfile.mkstemp(dir=path.parent, prefix=&#39;.main.go.&#39;)
try:
    with os.fdopen(fd, &#39;w&#39;, encoding=&#39;utf-8&#39;, newline=&#39;&#39;) as f:
        f.write(source)
        f.flush()
        os.fsync(f.fileno())
    os.replace(temp_name, path)
    new_time = max(path.stat().st_mtime_ns, old_mtime + 1_000_000)
    os.utime(path, ns=(new_time, new_time))
finally:
    if os.path.exists(temp_name):
        os.unlink(temp_name)
PY

git diff -- main.go go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;randomNumber, err := rand.Int(rand.Reader, big.NewInt(100))\nif err != nil {\n\tlog.Fatal(err)\n}\n\ngreeting := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := greeting.Execute(os.Stdout, randomNumber); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short &amp;&amp; \
rg --files&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short -- main.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ],
  &#34;add&#34;: [
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short -- main.go go.mod go.sum&#39;</code></pre></td></tr>
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
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,24p&#39; main.go&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 21 lines)
* **MCP Edit**: File main.go modified (4 lines -> 24 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; main.go &amp;&amp; \
git status --short -- main.go go.mod go.sum &amp;&amp; \
git diff --no-index /dev/null main.go&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;randomInt, err := rand.Int(rand.Reader, big.NewInt(100))\nif err != nil {\n\tlog.Fatal(err)\n}\n\ngreeting, err := template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World: {{.}}\\n\&#34;)\nif err != nil {\n\tlog.Fatal(err)\n}\nif err := greeting.Execute(os.Stdout, randomInt); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ],
  &#34;add&#34;: [
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
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
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td></tr>
</tbody>
</table>

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
| **Wall-Clock Latency** | 38.19s | 93.86s | <span class="benchmark-delta-negative">+145.8%</span> | 28.49s | 46.62s | <span class="benchmark-delta-negative">+63.6%</span> |
| **Process Start → First Event** | 0.12s | 0.12s | — | 0.13s | 0.13s | — |
| **First Event → First Tool Call** | 8.81s | 7.55s | — | 5.52s | 5.77s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 4 | 11 | <span class="benchmark-delta-negative">+175.0%</span> | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> |
| **Initial Load / Discovery Turns** | 4 | 3 | <span class="benchmark-delta-positive">-25.0%</span> | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 4 | 11 | <span class="benchmark-delta-negative">+175.0%</span> | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> |
| **Output Tokens** | 1036 | 2908 | <span class="benchmark-delta-negative">+180.7%</span> | 818 | 861 | <span class="benchmark-delta-negative">+5.3%</span> |
| **Reasoning / Thinking Tokens** | 651 | 1801 | <span class="benchmark-delta-negative">+176.7%</span> | 408 | 466 | <span class="benchmark-delta-negative">+14.2%</span> |
| **Total Input Tokens** | 115958 | 265373 | <span class="benchmark-delta-negative">+128.9%</span> | 77501 | 155964 | <span class="benchmark-delta-negative">+101.2%</span> |
| **Cached Input Tokens** | 104960 | 231424 | <span class="benchmark-delta-positive">+120.5%</span> | 67584 | 136448 | <span class="benchmark-delta-positive">+101.9%</span> |
| **Uncached Input Tokens** | 10998 | 33949 | <span class="benchmark-delta-negative">+208.7%</span> | 9917 | 19516 | <span class="benchmark-delta-negative">+96.8%</span> |
| **Cached vs Uncached Token Ratio** | 9.54:1 | 6.82:1 | <span class="benchmark-delta-negative">-28.6%</span> | 6.81:1 | 6.99:1 | <span class="benchmark-delta-positive">+2.6%</span> |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS*&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d main.go &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat main.go &amp;&amp; \
rg --files | rg &#39;(&#34;&#39;^|/)SUBOPTIMAL_TOOLS&#39;&#34;\\.md&#34;&#39;$&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;SUBOPTIMAL_TOOLS.md&#39; -g &#39;docs/**&#39; | head -30; ls&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
import os
import tempfile
from pathlib import Path

path = Path(&#39;docs/SUBOPTIMAL_TOOLS.md&#39;)
path.parent.mkdir(parents=True, exist_ok=True)
content = \&#34;\&#34;\&#34;# Suboptimal Tool Behavior

## 2026-09-24: &#34;&#39;`semantic_replace_body` import inference

- Tool: `semantic_replace_body` with `auto_organize_imports: true`
- Target: `main.go`
- Observed failure: The edit required `text/template` and a random integer, but import organization selected `html/template` and `crypto/rand`. Verification then reported that `crypto/rand.Int` requires arguments.
- Workaround: Correct the imports explicitly to `text/template` and `math/rand`, then re-run semantic import organization.
- Root cause: Automatic import resolution chose unintended packages for ambiguous identifiers.
&#34;&#34;&#34;
fd, temp_name = tempfile.mkstemp(prefix=&#39;&#34;&#39;.SUBOPTIMAL_TOOLS.&#39;, dir=path.parent)
try:
    with os.fdopen(fd, &#39;w&#39;) as f:
        f.write(content)
        f.flush()
        os.fsync(f.fileno())
    os.replace(temp_name, path)
    os.utime(path, None)
except BaseException:
    try:
        os.unlink(temp_name)
    except FileNotFoundError:
        pass
    raise
PY&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
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
<tr><td>9</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git diff -- main.go docs/SUBOPTIMAL_TOOLS.md&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 15 lines)
* **MCP Edit**: File main.go modified (4 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;*semantic*&#39; -g &#39;*.md&#39; | head -80 &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\n\tif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\t\tlog.Fatal(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 -c &#39;import os, tempfile; p=\&#34;main.go\&#34;; data=\&#34;\&#34;\&#34;package main

import (
	\\\&#34;log\\\&#34;
	\\\&#34;math/rand\\\&#34;
	\\\&#34;os\\\&#34;
	\\\&#34;text/template\\\&#34;
)

func main() {
	tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\\\n\&#34;))
	if err := tmpl.Execute(os.Stdout, rand.Intn(100)); err &#34;&#39;!= nil {
		log.Fatal(err)
	}
}
&#34;&#34;&#34;; fd, tmp = tempfile.mkstemp(prefix=&#34;.main.go.&#34;, dir=&#34;.&#34;);
f = os.fdopen(fd, &#34;w&#34;); f.write(data); f.flush(); os.fsync(f.fileno()); f.close(); os.replace(tmp, p); d=os.open(&#34;.&#34;, os.O_RDONLY); os.fsync(d); os.close(d)&#39;&#34;&#39; &amp;&amp; \
cat main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;text/template&#34;
  ],
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not combine the calls because I chose to correct the imports and function body in separate steps after the initial import inference produced incorrect packages. Batching was available but I did not use it; it was a sequencing choice, not a limitation.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 5.24s; turns: 1; tool calls: 0.</p>
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
| **Wall-Clock Latency** | 69.75s | 30.03s | <span class="benchmark-delta-positive">-56.9%</span> | 58.38s | 47.40s | <span class="benchmark-delta-positive">-18.8%</span> |
| **Process Start → First Event** | 0.12s | 0.13s | — | 0.12s | 0.12s | — |
| **First Event → First Tool Call** | 18.74s | 10.00s | — | 19.89s | 15.96s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 6 | 3 | <span class="benchmark-delta-positive">-50.0%</span> | 6 | 3 | <span class="benchmark-delta-positive">-50.0%</span> |
| **Initial Load / Discovery Turns** | 6 | 1 | <span class="benchmark-delta-positive">-83.3%</span> | 6 | 1 | <span class="benchmark-delta-positive">-83.3%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 6 | 3 | <span class="benchmark-delta-positive">-50.0%</span> | 6 | 3 | <span class="benchmark-delta-positive">-50.0%</span> |
| **Output Tokens** | 1979 | 927 | <span class="benchmark-delta-positive">-53.2%</span> | 2353 | 1636 | <span class="benchmark-delta-positive">-30.5%</span> |
| **Reasoning / Thinking Tokens** | 1361 | 554 | <span class="benchmark-delta-positive">-59.3%</span> | 1519 | 1077 | <span class="benchmark-delta-positive">-29.1%</span> |
| **Total Input Tokens** | 285897 | 152347 | <span class="benchmark-delta-positive">-46.7%</span> | 196430 | 190868 | <span class="benchmark-delta-positive">-2.8%</span> |
| **Cached Input Tokens** | 261632 | 126720 | <span class="benchmark-delta-negative">-51.6%</span> | 173312 | 163328 | <span class="benchmark-delta-negative">-5.8%</span> |
| **Uncached Input Tokens** | 24265 | 25627 | <span class="benchmark-delta-negative">+5.6%</span> | 23118 | 27540 | <span class="benchmark-delta-negative">+19.1%</span> |
| **Cached vs Uncached Token Ratio** | 10.78:1 | 4.94:1 | <span class="benchmark-delta-negative">-54.1%</span> | 7.50:1 | 5.93:1 | <span class="benchmark-delta-negative">-20.9%</span> |
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
rg --files -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; | sort &amp;&amp; \
cat main.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go&#39;</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
cat main.go&#39;</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- main.go &amp;&amp; \
git check-ignore -v main.go&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 15 lines)
* **MCP Edit**: File main.go modified (4 lines -> 14 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;Makefile&#39; | sort &amp;&amp; \
git status --short &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- main.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tpanic(err)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- main.go go.mod go.sum&#39;</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

