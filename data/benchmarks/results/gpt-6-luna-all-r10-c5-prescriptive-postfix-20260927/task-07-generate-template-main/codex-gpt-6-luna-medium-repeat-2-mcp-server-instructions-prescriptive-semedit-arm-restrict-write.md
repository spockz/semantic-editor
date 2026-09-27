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

* **Date**: 2026-09-27 03:12:39 CEST

## Test case: `task-07-generate-template-main`

### Target: `codex/gpt-6-luna/medium (repeat 2)`

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
| **Wall-Clock Latency** | 20.60s | 55.04s | <span class="benchmark-delta-negative">+167.2%</span> | — | — | — |
| **Process Start → First Event** | 0.14s | 0.14s | — | — | — | — |
| **First Event → First Tool Call** | 4.28s | 5.19s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 3 | 8 | <span class="benchmark-delta-negative">+166.7%</span> | — | — | — |
| **Output Tokens** | 763 | 1489 | <span class="benchmark-delta-negative">+95.2%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 222 | 436 | <span class="benchmark-delta-negative">+96.4%</span> | — | — | — |
| **Total Input Tokens** | 77752 | 244513 | <span class="benchmark-delta-negative">+214.5%</span> | — | — | — |
| **Cached Input Tokens** | 67584 | 222720 | <span class="benchmark-delta-positive">+229.5%</span> | — | — | — |
| **Uncached Input Tokens** | 10168 | 21793 | <span class="benchmark-delta-negative">+114.3%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.65:1 | 10.22:1 | <span class="benchmark-delta-positive">+53.8%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 24 lines)
* **MCP Edit**: File main.go modified (4 lines -> 20 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;AGENTS*&#39; -g &#39;README*&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile

path = Path(&#39;main.go&#39;)
content = &#39;&#39;&#39;// This small program demonstrates rendering a template with a cryptographically secure random value.
package main

import (
\\t\&#34;crypto/rand\&#34;
\\t\&#34;fmt\&#34;
\\t\&#34;math/big\&#34;
\\t\&#34;os\&#34;
\\t\&#34;text/template\&#34;
)

func main() {
\\tn, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
\\tif err &#34;&#39;!= nil {
&#39;&#34;\\t\\tfmt.Fprintln(os.Stderr, \&#34;generate random integer:\&#34;, err)
\\t\\tos.Exit(1)
\\t}

\\ttmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\\\n\&#34;))
\\tif err := tmpl.Execute(os.Stdout, n); err &#34;&#39;!= nil {
&#39;&#34;\\t\\tfmt.Fprintln(os.Stderr, \&#34;render greeting:\&#34;, err)
\\t\\tos.Exit(1)
\\t}
}
&#39;&#39;&#39;
fd, tmp = tempfile.mkstemp(dir=path.parent, prefix=&#39;.main.go.&#39;)
try:
    with os.fdopen(fd, &#39;w&#39;) as f:
        f.write(content)
        f.flush()
        os.fsync(f.fileno())
    os.replace(tmp, path)
finally:
    if os.path.exists(tmp):
        os.unlink(tmp)
PY
gofmt -w main.go
cat main.go
git diff -- main.go go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;n, err := rand.Int(rand.Reader, big.NewInt(100))\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\ttmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\n\tif err := tmpl.Execute(os.Stdout, n); err != nil {\n\t\tpanic(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
go build ./...&#39;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;n, err := rand.Int(rand.Reader, big.NewInt(100))\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\ttmpl := texttemplate.Must(texttemplate.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\n\tif err := tmpl.Execute(os.Stdout, n); err != nil {\n\t\tpanic(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;texttemplate \&#34;text/template\&#34;&#34;
  ],
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
rg --files&#39;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;mkdir -p docs &amp;&amp; \
python3 -c &#39;from pathlib import Path; import os, tempfile; p=Path(\&#34;docs/SUBOPTIMAL_TOOLS.md\&#34;); data=\&#34;# Suboptimal Tool Behavior\\n\\n- Tool: &#34;&#39;`semantic_replace_body`&#39;&#34;\\n  Target: &#34;&#39;`main.go`&#39;&#34;\\n  Observed failure: automatic import resolution selected &#34;&#39;`html/template` for the `template` identifier, then did not add the explicitly aliased `text/template` import when the body used `texttemplate`.&#39;&#34;\\n  Workaround: used &#34;&#39;`semantic_organize_imports` to add `texttemplate &#39;&#34;\\\&#34;text/template\\\&#34;&#34;&#39;` and remove `html/template`.&#39;&#34;\\n  Root cause: import resolution did not honor the source package requested by this task and did not infer the explicit alias.\\n\&#34;; fd, tmp=tempfile.mkstemp(dir=p.parent); f=os.fdopen(fd,\&#34;w\&#34;); f.write(data); f.flush(); os.fsync(f.fileno()); f.close(); os.replace(tmp,p)&#39; &amp;&amp; \
go build ./... &amp;&amp; \
go run .&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I used separate semantic calls because `semantic_batch` was not among the tools I inspected or discovered during the task. I should have checked whether it was available before making consecutive semantic edits.</div></div>

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
| **Wall-Clock Latency** | 24.48s | 31.99s | <span class="benchmark-delta-negative">+30.7%</span> | — | — | — |
| **Process Start → First Event** | 0.13s | 0.12s | — | — | — | — |
| **First Event → First Tool Call** | 4.09s | 5.54s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 5 | <span class="benchmark-delta-negative">+25.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 5 | <span class="benchmark-delta-negative">+25.0%</span> | — | — | — |
| **Output Tokens** | 588 | 500 | <span class="benchmark-delta-positive">-15.0%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 162 | 86 | <span class="benchmark-delta-positive">-46.9%</span> | — | — | — |
| **Total Input Tokens** | 97031 | 159986 | <span class="benchmark-delta-negative">+64.9%</span> | — | — | — |
| **Cached Input Tokens** | 86784 | 143616 | <span class="benchmark-delta-positive">+65.5%</span> | — | — | — |
| **Uncached Input Tokens** | 10247 | 16370 | <span class="benchmark-delta-negative">+59.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.47:1 | 8.77:1 | <span class="benchmark-delta-positive">+3.6%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 24 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39;&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;rng := rand.New(rand.NewSource(time.Now().UnixNano()))\n\tdata := struct {\n\t\tName   string\n\t\tRandom int\n\t}{\n\t\tName:   \&#34;World\&#34;,\n\t\tRandom: rng.Intn(100),\n\t}\n\ttmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello {{.Name}} {{.Random}}\&#34;))\n\tif err := tmpl.Execute(os.Stdout, data); err != nil {\n\t\tlog.Fatal(err)\n\t}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 -c &#39;from pathlib import Path; import os, tempfile; p=Path(\&#34;main.go\&#34;); data=\&#34;\&#34;\&#34;package main\\n\\nimport (\\n\\t\\\&#34;math/rand\\\&#34;\\n\\t\\\&#34;os\\\&#34;\\n\\t\\\&#34;text/template\\\&#34;\\n)\\n\\nfunc main() {\\n\\ttmpl := template.Must(template.New(\\\&#34;greeting\\\&#34;).Parse(\\\&#34;Hello World {{.}}\\\\n\\\&#34;))\\n\\tif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err &#34;&#39;!= nil {&#39;&#34;\\n\\t\\tpanic(err)\\n\\t}\\n}\\n\&#34;\&#34;\&#34;; fd, name=tempfile.mkstemp(dir=p.parent, prefix=\&#34;.main.go.\&#34;); f=os.fdopen(fd,\&#34;w\&#34;); f.write(data); f.flush(); os.fsync(f.fileno()); f.close(); os.replace(name,p)&#39;&#34;</code></pre></td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;remove&#34;: [
    &#34;html/template&#34;
  ],
  &#34;add&#34;: [
    &#34;text/template&#34;
  ]
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go run .&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was unsuitable here: replacing `main`’s body and organizing imports are separate semantic operations, and the body edit’s automatic import organization handled the imports. I made a separate import change afterward to use `text/template` specifically.</div></div>

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
| **Wall-Clock Latency** | 19.05s | 63.15s | <span class="benchmark-delta-negative">+231.6%</span> | — | — | — |
| **Process Start → First Event** | 0.13s | 0.14s | — | — | — | — |
| **First Event → First Tool Call** | 8.33s | 7.78s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 1 | 11 | <span class="benchmark-delta-negative">+1000.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 1 | 11 | <span class="benchmark-delta-negative">+1000.0%</span> | — | — | — |
| **Output Tokens** | 622 | 1530 | <span class="benchmark-delta-negative">+146.0%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 273 | 475 | <span class="benchmark-delta-negative">+74.0%</span> | — | — | — |
| **Total Input Tokens** | 135029 | 357506 | <span class="benchmark-delta-negative">+164.8%</span> | — | — | — |
| **Cached Input Tokens** | 114432 | 330752 | <span class="benchmark-delta-positive">+189.0%</span> | — | — | — |
| **Uncached Input Tokens** | 20597 | 26754 | <span class="benchmark-delta-negative">+29.9%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.56:1 | 12.36:1 | <span class="benchmark-delta-positive">+122.5%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **MCP Edit**: File main.go modified (4 lines -> 16 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;main.go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;value := rand.Intn(100)\n\ttmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\n\tif err := tmpl.Execute(os.Stdout, value); err != nil {\n\t\tlog.Fatal(err)\n\t}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;add&#34;: [
    &#34;html/template&#34;,
    &#34;log&#34;,
    &#34;math/rand&#34;,
    &#34;os&#34;
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
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;SUBOPTIMAL_TOOLS.md&#39; -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;Makefile&#39;&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build -o /tmp/semedit-hello .&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;main.go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go run .&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;SUBOPTIMAL_TOOLS.md&#39; -g &#39;docs/**&#39;&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was discovered in the tool inventory, but I didn’t use it because I treated body replacement and import organization as separate steps, with the imports dependent on the body edit. In hindsight, they could have been combined in order with `semantic_batch`; batching was suitable.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

