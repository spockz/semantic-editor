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

* **Date**: 2026-09-26 23:13:54 CEST

## Test case: `task-07-generate-template-main`

### Target: `codex/gpt-6-luna/medium (repeat 5)`

#### Configuration: crypto_rand prompt · descriptive MCP instructions · write semedit restriction

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
| **Wall-Clock Latency** | 29.92s | 62.54s | <span class="benchmark-delta-negative">+109.0%</span> | — | — | — |
| **Process Start → First Event** | 0.64s | 0.95s | — | — | — | — |
| **First Event → First Tool Call** | 14.80s | 15.40s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 3 | 10 | <span class="benchmark-delta-negative">+233.3%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 3 | 10 | <span class="benchmark-delta-negative">+233.3%</span> | — | — | — |
| **Output Tokens** | 616 | 960 | <span class="benchmark-delta-negative">+55.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 183 | 289 | <span class="benchmark-delta-negative">+57.9%</span> | — | — | — |
| **Total Input Tokens** | 76680 | 166364 | <span class="benchmark-delta-negative">+117.0%</span> | — | — | — |
| **Cached Input Tokens** | 68608 | 149760 | <span class="benchmark-delta-positive">+118.3%</span> | — | — | — |
| **Uncached Input Tokens** | 8072 | 16604 | <span class="benchmark-delta-negative">+105.7%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.50:1 | 9.02:1 | <span class="benchmark-delta-positive">+6.1%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 20 lines)
* **MCP Edit**: File main.go modified (4 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 -c &#39;from pathlib import Path; import os, tempfile; p=Path(\&#34;main.go\&#34;); data=\&#34;\&#34;\&#34;package main

import (
\\t\\\&#34;crypto/rand\\\&#34;
\\t\\\&#34;math/big\\\&#34;
\\t\\\&#34;os\\\&#34;
\\t\\\&#34;text/template\\\&#34;
)

func main() {
\\tn, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
\\tif err &#34;&#39;!= nil {
&#39;&#34;\\t\\tpanic(err)
\\t}

\\ttmpl := template.Must(template.New(\\\&#34;greeting\\\&#34;).Parse(\\\&#34;Hello World {{.}}\\\\n\\\&#34;))
\\tif err := tmpl.Execute(os.Stdout, n); err &#34;&#39;!= nil {
&#39;&#34;\\t\\tpanic(err)
\\t}
}
\&#34;\&#34;\&#34;; fd, name=tempfile.mkstemp(dir=str(p.parent), prefix=\&#34;.main.go.\&#34;); f=os.fdopen(fd,\&#34;w\&#34;); f.write(data); f.flush(); os.fsync(f.fileno()); f.close(); os.replace(name,p)&#39;&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;n, err := rand.Int(rand.Reader, big.NewInt(100))\nif err != nil {\n\tlog.Fatal(err)\n}\ntmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World: {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, n); err != nil {\n\tlog.Fatal(err)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build -o /tmp/semedit-hello-world .&#39;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- main.go go.mod go.sum &amp;&amp; \
git diff -- main.go&#39;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;text/template&#34;
  ],
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- main.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build -o /tmp/semedit-hello-world .&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was unsuitable because replacing `main()` needed to run first so import organization could infer the packages actually used; the import correction depended on the resulting imports. I used separate semantic operations to preserve that order.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Configuration: default prompt · descriptive MCP instructions · write semedit restriction

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
| **Wall-Clock Latency** | 28.47s | 53.63s | <span class="benchmark-delta-negative">+88.4%</span> | — | — | — |
| **Process Start → First Event** | 0.13s | 0.17s | — | — | — | — |
| **First Event → First Tool Call** | 8.48s | 6.06s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 2 | 7 | <span class="benchmark-delta-negative">+250.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 2 | 7 | <span class="benchmark-delta-negative">+250.0%</span> | — | — | — |
| **Output Tokens** | 621 | 1358 | <span class="benchmark-delta-negative">+118.7%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 213 | 259 | <span class="benchmark-delta-negative">+21.6%</span> | — | — | — |
| **Total Input Tokens** | 57132 | 220343 | <span class="benchmark-delta-negative">+285.7%</span> | — | — | — |
| **Cached Input Tokens** | 48384 | 198400 | <span class="benchmark-delta-positive">+310.1%</span> | — | — | — |
| **Uncached Input Tokens** | 8748 | 21943 | <span class="benchmark-delta-negative">+150.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.53:1 | 9.04:1 | <span class="benchmark-delta-positive">+63.5%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 17 lines)
* **MCP Edit**: File main.go modified (4 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;AGENTS*&#39; | sort &amp;&amp; \
sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;main.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
import os
import tempfile
from pathlib import Path

path = Path(&#39;main.go&#39;)
content = &#39;&#39;&#39;package main

import (
	\&#34;math/rand\&#34;
	\&#34;os\&#34;
	\&#34;text/template\&#34;
)

func main() {
	tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.Random}}\\\\n\&#34;))
	data := struct {
		Random int
	}{Random: rand.Intn(100)}
	if err := tmpl.Execute(os.Stdout, data); err &#34;&#39;!= nil {
		panic(err)
	}
}
&#39;&#34;&#39;&#39;&#39;
fd, temp_name = tempfile.mkstemp(dir=path.parent, prefix=&#39;.main.go.&#39;)
try:
    with os.fdopen(fd, &#39;w&#39;) as f:
        f.write(content)
        f.flush()
        os.fsync(f.fileno())
    os.replace(temp_name, path)
finally:
    if os.path.exists(temp_name):
        os.unlink(temp_name)
PY
gofmt -w main.go
go run main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;\ttmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\n\tif err := tmpl.Execute(os.Stdout, rand.Int()); err != nil {\n\t\tlog.Fatal(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat main.go &amp;&amp; \
rg --files -g &#39;SUBOPTIMAL_TOOLS.md&#39; -g &#39;Makefile&#39;&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;remove&#34;: [
    &#34;crypto/rand&#34;
  ],
  &#34;add&#34;: [
    &#34;math/rand&#34;
  ]
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile
p = Path(&#39;docs/SUBOPTIMAL_TOOLS.md&#39;)
p.parent.mkdir(parents=True, exist_ok=True)
old = p.read_text() if p.exists() else &#39;# Suboptimal Tool Behavior\\n&#39;
entry = &#39;&#39;&#39;\\n## 2026-09-26: &#34;&#39;`semantic_replace_body` selected the wrong `rand` package&#39;&#34;\\n\\n- Tool: &#34;&#39;`semantic_replace_body` on `main.go`&#39;&#34;\\n- Observed failure: &#34;&#39;`rand.Int()` was resolved from `crypto/rand`, producing a vet diagnostic because this API requires an `io.Reader` and `*big.Int`.&#39;&#34;\\n- Workaround applied: changed imports with &#34;&#39;`semantic_organize_imports` to use `math/rand`, whose `Int()` returns a random integer.&#39;&#34;\\n- Root cause: package selection for the ambiguous &#34;&#39;`rand.Int()` selector chose `crypto/rand` instead of `math/rand`.&#39;&#34;\\n&#39;&#39;&#39;
fd, name = tempfile.mkstemp(dir=p.parent, prefix=&#39;.SUBOPTIMAL_TOOLS.&#39;)
try:
    with os.fdopen(fd, &#39;w&#39;) as f:
        f.write(old + entry)
        f.flush()
        os.fsync(f.fileno())
    os.replace(name, p)
finally:
    if os.path.exists(name):
        os.unlink(name)
PY
cat main.go
cat docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Configuration: prefer_discover_semedit prompt · descriptive MCP instructions · write semedit restriction

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
| **Wall-Clock Latency** | 35.92s | 44.10s | <span class="benchmark-delta-negative">+22.8%</span> | — | — | — |
| **Process Start → First Event** | 0.17s | 0.16s | — | — | — | — |
| **First Event → First Tool Call** | 19.00s | 17.65s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 2 | 4 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 2 | 4 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Output Tokens** | 687 | 709 | <span class="benchmark-delta-negative">+3.2%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 342 | 300 | <span class="benchmark-delta-positive">-12.3%</span> | — | — | — |
| **Total Input Tokens** | 164957 | 167221 | <span class="benchmark-delta-negative">+1.4%</span> | — | — | — |
| **Cached Input Tokens** | 143872 | 145920 | <span class="benchmark-delta-positive">+1.4%</span> | — | — | — |
| **Uncached Input Tokens** | 21085 | 21301 | <span class="benchmark-delta-negative">+1.0%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.82:1 | 6.85:1 | <span class="benchmark-delta-positive">+0.4%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 15 lines)
* **MCP Edit**: File main.go modified (4 lines -> 18 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;README*&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w main.go &amp;&amp; \
cat main.go &amp;&amp; \
git status --short -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;tmpl, err := template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;)\nif err != nil {\n\tlog.Fatal(err)\n}\nif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tlog.Fatal(err)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
go run main.go&#39;</code></pre></td></tr>
</tbody>
</table>

