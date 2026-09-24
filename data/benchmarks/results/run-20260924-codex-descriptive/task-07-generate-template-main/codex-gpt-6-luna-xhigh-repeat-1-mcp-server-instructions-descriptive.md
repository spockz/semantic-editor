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

### Target: `codex/gpt-6-luna/xhigh (repeat 1)`

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
| **Wall-Clock Latency** | 65.13s | 54.89s | <span class="benchmark-delta-positive">-15.7%</span> | 69.24s | 55.06s | <span class="benchmark-delta-positive">-20.5%</span> |
| **Process Start → First Event** | 0.13s | 0.12s | — | 0.12s | 0.13s | — |
| **First Event → First Tool Call** | 16.51s | 8.03s | — | 8.02s | 9.46s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> | 3 | 4 | <span class="benchmark-delta-negative">+33.3%</span> |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> | 3 | 4 | <span class="benchmark-delta-negative">+33.3%</span> |
| **Output Tokens** | 1526 | 1737 | <span class="benchmark-delta-negative">+13.8%</span> | 3388 | 2183 | <span class="benchmark-delta-positive">-35.6%</span> |
| **Reasoning / Thinking Tokens** | 1015 | 1260 | <span class="benchmark-delta-negative">+24.1%</span> | 2764 | 1648 | <span class="benchmark-delta-positive">-40.4%</span> |
| **Total Input Tokens** | 77030 | 134353 | <span class="benchmark-delta-negative">+74.4%</span> | 83401 | 106568 | <span class="benchmark-delta-negative">+27.8%</span> |
| **Cached Input Tokens** | 62464 | 118272 | <span class="benchmark-delta-positive">+89.3%</span> | 70656 | 91904 | <span class="benchmark-delta-positive">+30.1%</span> |
| **Uncached Input Tokens** | 14566 | 16081 | <span class="benchmark-delta-negative">+10.4%</span> | 12745 | 14664 | <span class="benchmark-delta-negative">+15.1%</span> |
| **Cached vs Uncached Token Ratio** | 4.29:1 | 7.35:1 | <span class="benchmark-delta-positive">+71.5%</span> | 5.54:1 | 6.27:1 | <span class="benchmark-delta-positive">+13.1%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 24 lines)
* **MCP Edit**: File main.go modified (4 lines -> 19 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
cat go.mod&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
import os
import tempfile
from pathlib import Path

path = Path(&#39;main.go&#39;)
content = &#39;&#39;&#39;package main

import (
\\t\&#34;crypto/rand\&#34;
\\t\&#34;log\&#34;
\\t\&#34;math/big\&#34;
\\t\&#34;os\&#34;
\\t\&#34;text/template\&#34;
)

func main() {
\\tn, err := rand.Int(rand.Reader, big.NewInt(100))
\\tif err &#34;&#39;!= nil {
&#39;&#34;\\t\\tlog.Fatal(err)
\\t}

\\ttmpl, err := template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\\\n\&#34;)
\\tif err &#34;&#39;!= nil {
&#39;&#34;\\t\\tlog.Fatal(err)
\\t}
\\tif err := tmpl.Execute(os.Stdout, n); err &#34;&#39;!= nil {
&#39;&#34;\\t\\tlog.Fatal(err)
\\t}
}
&#39;&#39;&#39;
fd, temp_name = tempfile.mkstemp(prefix=&#39;.main.go.&#39;, dir=&#39;.&#39;)
try:
    with os.fdopen(fd, &#39;w&#39;) as temp_file:
        temp_file.write(content)
        temp_file.flush()
        os.fsync(temp_file.fileno())
    os.replace(temp_name, path)
    os.utime(path, None)
except Exception:
    try:
        os.unlink(temp_name)
    except FileNotFoundError:
        pass
    raise
PY
cat main.go&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;greeting := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nvalue, err := rand.Int(rand.Reader, big.NewInt(100))\nif err != nil {\n\tpanic(err)\n}\nif err := greeting.Execute(os.Stdout, value); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go go.mod go.sum &amp;&amp; \
cat main.go&#39;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git ls-files --stage -- main.go go.mod go.sum&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 25 lines)
* **MCP Edit**: File main.go modified (4 lines -> 25 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os
import tempfile
import time

path = Path(&#39;main.go&#39;)
old_mtime = path.stat().st_mtime_ns
content = &#39;&#39;&#39;// Package main renders a greeting with a cryptographically generated integer.
package main

import (
\\t\&#34;crypto/rand\&#34;
\\t\&#34;log\&#34;
\\t\&#34;math/big\&#34;
\\t\&#34;os\&#34;
\\t\&#34;text/template\&#34;
)

func main() {
\\tn, err := rand.Int(rand.Reader, big.NewInt(100))
\\tif err &#34;&#39;!= nil {
&#39;&#34;\\t\\tlog.Fatalf(\&#34;generate random integer: %v\&#34;, err)
\\t}

\\tgreeting, err := template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\\\n\&#34;)
\\tif err &#34;&#39;!= nil {
&#39;&#34;\\t\\tlog.Fatalf(\&#34;parse greeting template: %v\&#34;, err)
\\t}
\\tif err := greeting.Execute(os.Stdout, n); err &#34;&#39;!= nil {
&#39;&#34;\\t\\tlog.Fatalf(\&#34;execute greeting template: %v\&#34;, err)
\\t}
}
&#39;&#39;&#39;
fd, temp_name = tempfile.mkstemp(prefix=&#39;.main.go.&#39;, dir=str(path.parent))
try:
    with os.fdopen(fd, &#39;w&#39;) as temp_file:
        temp_file.write(content)
        temp_file.flush()
        os.fsync(temp_file.fileno())
    os.replace(temp_name, path)
    new_mtime = max(time.time_ns(), old_mtime + 1)
    os.utime(path, ns=(new_mtime, new_mtime))
finally:
    if os.path.exists(temp_name):
        os.unlink(temp_name)
PY
gofmt -w main.go
git diff -- main.go go.mod go.sum
git status --short -- main.go go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go &amp;&amp; \
git check-ignore -v main.go go.mod go.sum || \
true&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;number, err := rand.Int(rand.Reader, big.NewInt(100))\nif err != nil {\n\tlog.Fatalf(\&#34;generate random integer: %v\&#34;, err)\n}\n\ngreeting, err := template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World: {{.}}\\n\&#34;)\nif err != nil {\n\tlog.Fatalf(\&#34;parse greeting template: %v\&#34;, err)\n}\n\nif err := greeting.Execute(os.Stdout, number); err != nil {\n\tlog.Fatalf(\&#34;execute greeting template: %v\&#34;, err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git diff -- main.go&#39;</code></pre></td></tr>
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
| **Wall-Clock Latency** | 42.44s | 130.08s | <span class="benchmark-delta-negative">+206.5%</span> | 38.34s | 58.61s | <span class="benchmark-delta-negative">+52.9%</span> |
| **Process Start → First Event** | 0.13s | 0.12s | — | 0.54s | 0.12s | — |
| **First Event → First Tool Call** | 8.81s | 9.69s | — | 7.64s | 9.11s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 7 | <span class="benchmark-delta-negative">+133.3%</span> | 2 | 5 | <span class="benchmark-delta-negative">+150.0%</span> |
| **Initial Load / Discovery Turns** | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> | 1 | 3 | <span class="benchmark-delta-negative">+200.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 7 | <span class="benchmark-delta-negative">+133.3%</span> | 2 | 5 | <span class="benchmark-delta-negative">+150.0%</span> |
| **Output Tokens** | 1935 | 5759 | <span class="benchmark-delta-negative">+197.6%</span> | 1546 | 1619 | <span class="benchmark-delta-negative">+4.7%</span> |
| **Reasoning / Thinking Tokens** | 1319 | 4732 | <span class="benchmark-delta-negative">+258.8%</span> | 1066 | 1161 | <span class="benchmark-delta-negative">+8.9%</span> |
| **Total Input Tokens** | 78393 | 196618 | <span class="benchmark-delta-negative">+150.8%</span> | 59152 | 135786 | <span class="benchmark-delta-negative">+129.6%</span> |
| **Cached Input Tokens** | 63488 | 173056 | <span class="benchmark-delta-positive">+172.6%</span> | 49408 | 116224 | <span class="benchmark-delta-positive">+135.2%</span> |
| **Uncached Input Tokens** | 14905 | 23562 | <span class="benchmark-delta-negative">+58.1%</span> | 9744 | 19562 | <span class="benchmark-delta-negative">+100.8%</span> |
| **Cached vs Uncached Token Ratio** | 4.26:1 | 7.34:1 | <span class="benchmark-delta-positive">+72.4%</span> | 5.07:1 | 5.94:1 | <span class="benchmark-delta-positive">+17.2%</span> |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os
import tempfile
import time

path = Path(&#39;main.go&#39;)
content = &#39;&#39;&#39;package main

import (
	\&#34;math/rand\&#34;
	\&#34;os\&#34;
	\&#34;text/template\&#34;
)

func main() {
	t := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\\\n\&#34;))
	if err := t.Execute(os.Stdout, rand.Intn(100)); err &#34;&#39;!= nil {
		panic(err)
	}
}
&#39;&#34;&#39;&#39;&#39;
old_mtime = path.stat().st_mtime_ns if path.exists() else 0
fd, tmp_name = tempfile.mkstemp(dir=path.parent, prefix=f&#39;.{path.name}.&#39;)
try:
    with os.fdopen(fd, &#39;w&#39;, encoding=&#39;utf-8&#39;, newline=&#39;&#39;) as f:
        f.write(content)
        f.flush()
        os.fsync(f.fileno())
    os.replace(tmp_name, path)
    new_time = max(time.time_ns(), old_mtime + 1)
    os.utime(path, ns=(new_time, new_time))
    dir_fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(dir_fd)
    finally:
        os.close(dir_fd)
except BaseException:
    try:
        os.unlink(tmp_name)
    except FileNotFoundError:
        pass
    raise
PY&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; main.go&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;t := template.Must(template.New(\&#34;hello\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := t.Execute(os.Stdout, rand.Int()); err != nil {\n\tpanic(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; main.go; rg --files docs | rg &#39;SUBOPTIMAL_TOOLS\\.md&#34;&#39;$&#39;&#34;&#39; || \
true&#34;</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os
import tempfile

path = Path(&#39;docs/SUBOPTIMAL_TOOLS.md&#39;)
path.parent.mkdir(parents=True, exist_ok=True)
old = path.read_text() if path.exists() else &#39;&lt;&#34;&#39;!-- This log records semantic editing tool failures and their corrections. --&gt;&#39;&#34;\\n\\n# Suboptimal Tool Behavior\\n&#39;
entry = &#39;&#39;&#39;\\n## 2026-09-24: semantic_replace_body import inference\\n\\n- Tool: &#34;&#39;`semantic_replace_body` with `auto_organize_imports: true`&#39;&#34;\\n- Target: &#34;&#39;`main.go`&#39;&#34;\\n- Observed failure: import inference selected &#34;&#39;`crypto/rand` and `html/template` for `rand.Int()` and `template.New(...)`. Diagnostics reported that `crypto/rand.Int` requires an `io.Reader` and `*big.Int`; the request requires a standard random integer and `text/template`.&#39;&#34;\\n- Workaround: explicitly replace the inferred imports with &#34;&#39;`math/rand` and `text/template`, then re-check diagnostics.&#39;&#34;\\n- Root cause: package inference selected packages by matching identifiers without preserving the intended standard-library package semantics.\\n&#39;&#39;&#39;
new = old.rstrip() + &#39;\\n&#39; + entry
fd, tmp_name = tempfile.mkstemp(prefix=&#39;.SUBOPTIMAL_TOOLS.&#39;, dir=path.parent)
try:
    with os.fdopen(fd, &#39;w&#39;) as tmp:
        tmp.write(new)
        tmp.flush()
        os.fsync(tmp.fileno())
    os.replace(tmp_name, path)
    os.utime(path, None)
    dir_fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(dir_fd)
    finally:
        os.close(dir_fd)
except BaseException:
    try:
        os.unlink(tmp_name)
    except FileNotFoundError:
        pass
    raise
PY&#34;</code></pre></td></tr>
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
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,80p&#39; main.go &amp;&amp; \
git status --short &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- main.go docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 16 lines)
* **MCP Edit**: File main.go modified (4 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; main.go &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;docs/**&#39; | head -80 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
import os
import tempfile
import time
from pathlib import Path

path = Path(&#39;main.go&#39;)
source = &#39;&#39;&#39;package main

import (
	\&#34;math/rand\&#34;
	\&#34;os\&#34;
	\&#34;text/template\&#34;
	\&#34;time\&#34;
)

func main() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	greeting := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\\\n\&#34;))
	if err := greeting.Execute(os.Stdout, rng.Intn(100)); err &#34;&#39;!= nil {
		panic(err)
	}
}
&#39;&#34;&#39;&#39;&#39;
old_mtime_ns = path.stat().st_mtime_ns
fd, temp_name = tempfile.mkstemp(dir=path.parent, prefix=f&#39;.{path.name}.&#39;)
try:
    with os.fdopen(fd, &#39;w&#39;, encoding=&#39;utf-8&#39;, newline=&#39;&#39;) as stream:
        stream.write(source)
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temp_name, path)
    directory_fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)
    new_mtime_ns = max(time.time_ns(), old_mtime_ns + 1)
    os.utime(path, ns=(new_mtime_ns, new_mtime_ns))
finally:
    if os.path.exists(temp_name):
        os.unlink(temp_name)
PY
sed -n &#39;1,120p&#39; main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat main.go &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod&#39;</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tlog.Fatal(err)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short -- main.go go.mod go.sum &amp;&amp; \
git diff -- main.go go.mod go.sum&#39;</code></pre></td></tr>
</tbody>
</table>

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
| **Wall-Clock Latency** | 81.03s | 58.56s | <span class="benchmark-delta-positive">-27.7%</span> | 101.52s | 108.55s | <span class="benchmark-delta-negative">+6.9%</span> |
| **Process Start → First Event** | 0.12s | 0.13s | — | 0.13s | 0.12s | — |
| **First Event → First Tool Call** | 23.88s | 18.26s | — | 23.34s | 32.29s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 6 | 4 | <span class="benchmark-delta-positive">-33.3%</span> | 4 | 4 | 0% |
| **Initial Load / Discovery Turns** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 6 | 4 | <span class="benchmark-delta-positive">-33.3%</span> | 4 | 4 | 0% |
| **Output Tokens** | 3504 | 1722 | <span class="benchmark-delta-positive">-50.9%</span> | 5042 | 3651 | <span class="benchmark-delta-positive">-27.6%</span> |
| **Reasoning / Thinking Tokens** | 2337 | 1241 | <span class="benchmark-delta-positive">-46.9%</span> | 4170 | 2916 | <span class="benchmark-delta-positive">-30.1%</span> |
| **Total Input Tokens** | 262880 | 228941 | <span class="benchmark-delta-positive">-12.9%</span> | 172110 | 202467 | <span class="benchmark-delta-negative">+17.6%</span> |
| **Cached Input Tokens** | 233216 | 196864 | <span class="benchmark-delta-negative">-15.6%</span> | 146944 | 168192 | <span class="benchmark-delta-positive">+14.5%</span> |
| **Uncached Input Tokens** | 29664 | 32077 | <span class="benchmark-delta-negative">+8.1%</span> | 25166 | 34275 | <span class="benchmark-delta-negative">+36.2%</span> |
| **Cached vs Uncached Token Ratio** | 7.86:1 | 6.14:1 | <span class="benchmark-delta-negative">-21.9%</span> | 5.84:1 | 4.91:1 | <span class="benchmark-delta-negative">-16.0%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File main.go modified (4 lines -> 16 lines)
* **MCP Edit**: File main.go modified (4 lines -> 18 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;main.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short -- main.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl, err := template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;)\nif err != nil {\n\tlog.Fatal(err)\n}\nif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - main.go &lt;&lt;&#39;PY&#39;
import os
import pathlib
import tempfile
import time

path = pathlib.Path(\&#34;main.go\&#34;)
old_stat = path.stat()
content = \&#34;\&#34;\&#34;// main.go defines the executable entry point that renders the greeting template.
package main

import (
	\&#34;log\&#34;
	\&#34;math/rand\&#34;
	\&#34;os\&#34;
	\&#34;text/template\&#34;
)

func main() {
	greeting := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\\\n\&#34;))
	if err := greeting.Execute(os.Stdout, rand.Intn(100)); err &#34;&#39;!= nil {
		log.Fatal(err)
	}
}
&#34;&#34;&#34;
fd, temporary_name = tempfile.mkstemp(prefix=&#34;.main.go.&#34;, dir=path.parent)
try:
    with os.fdopen(fd, &#34;w&#34;, encoding=&#34;utf-8&#34;, newline=&#34;&#34;) as temporary:
        temporary.write(content)
        temporary.flush()
        os.fsync(temporary.fileno())
    os.chmod(temporary_name, old_stat.st_mode)
    mtime_ns = max(time.time_ns(), old_stat.st_mtime_ns + 1)
    os.utime(temporary_name, ns=(mtime_ns, mtime_ns))
    with open(temporary_name, &#34;rb&#34;) as temporary:
        os.fsync(temporary.fileno())
    os.replace(temporary_name, path)
    directory_fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)
except BaseException:
    try:
        os.unlink(temporary_name)
    except FileNotFoundError:
        pass
    raise
PY
&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short -- main.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- main.go go.mod go.sum &amp;&amp; \
git status --short -- main.go go.mod go.sum&#39;</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go &amp;&amp; \
git status --short &amp;&amp; \
git rev-parse --show-toplevel&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File main.go modified (4 lines -> 19 lines)
* **MCP Edit**: File main.go modified (4 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;main.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -la &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;main.go&#39; -g &#39;AGENTS.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat main.go &amp;&amp; \
cat go.mod &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -not -path &#39;./.scratch/*&#39; -print&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat main.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;main.go&#34;,
  &#34;symbol&#34;: &#34;main&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;tmpl := template.Must(template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World {{.}}\\n\&#34;))\nif err := tmpl.Execute(os.Stdout, rand.Intn(100)); err != nil {\n\tlog.Fatal(err)\n}&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
import os
import pathlib
import tempfile
import time

path = pathlib.Path(&#39;main.go&#39;)
old_stat = path.stat()
content = &#39;&#39;&#39;// This file keeps the fixture&#39;s standalone template example executable from its root.\\npackage main\\n\\nimport (\\n\\t\&#34;log\&#34;\\n\\t\&#34;math/rand\&#34;\\n\\t\&#34;os\&#34;\\n\\t\&#34;text/template\&#34;\\n)\\n\\nfunc main() {\\n\\tgreeting, err := template.New(\&#34;greeting\&#34;).Parse(\&#34;Hello World&#34;&#39;! Random integer: {{.}}&#39;&#34;\\\\n\&#34;)\\n\\tif err &#34;&#39;!= nil {&#39;&#34;\\n\\t\\tlog.Fatal(err)\\n\\t}\\n\\tif err := greeting.Execute(os.Stdout, rand.Intn(100)); err &#34;&#39;!= nil {&#39;&#34;\\n\\t\\tlog.Fatal(err)\\n\\t}\\n}\\n&#39;&#39;&#39;
fd, temp_name = tempfile.mkstemp(prefix=&#39;.main.go.&#39;, dir=path.parent)
try:
    with os.fdopen(fd, &#39;w&#39;, encoding=&#39;utf-8&#39;, newline=&#39;&#39;) as temp:
        temp.write(content)
        temp.flush()
        os.fsync(temp.fileno())
    os.chmod(temp_name, old_stat.st_mode)
    now = time.time_ns()
    mtime = max(now, old_stat.st_mtime_ns + 1)
    os.utime(temp_name, ns=(old_stat.st_atime_ns, mtime))
    os.replace(temp_name, path)
    dir_fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(dir_fd)
    finally:
        os.close(dir_fd)
except BaseException:
    try:
        os.unlink(temp_name)
    except FileNotFoundError:
        pass
    raise
PY&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- main.go go.mod go.sum &amp;&amp; \
cat main.go&#39;</code></pre></td></tr>
</tbody>
</table>

