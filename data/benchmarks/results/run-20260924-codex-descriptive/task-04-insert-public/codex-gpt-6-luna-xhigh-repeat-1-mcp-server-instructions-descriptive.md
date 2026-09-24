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

## Test case: `task-04-insert-public`

### Target: `codex/gpt-6-luna/xhigh (repeat 1)`

#### Configuration: default prompt · descriptive MCP instructions

* **Run Provenance**: `binary=44027cf7b57a0b4e`

* **Fixture**: [testdata/bench/task_04_insert_public.txtar](https://github.com/spockz/semantic-editor/blob/dd39d52587030a6aae7bbbf05b381f54bb386d3a/testdata/bench/task_04_insert_public.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add public constructor func InitServer() *Server placed before private helpers. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Add public constructor func InitServer() *Server placed before private helpers. Prefer using semantic editor operations if applicable. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct {
	Port int
}

func (s *Server) Start() {}

func (s *Server) internalRun() {}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 43.86s | 64.65s | <span class="benchmark-delta-negative">+47.4%</span> | 45.21s | 37.75s | <span class="benchmark-delta-positive">-16.5%</span> |
| **Process Start → First Event** | 0.12s | 0.13s | — | 0.48s | 0.14s | — |
| **First Event → First Tool Call** | 12.07s | 11.52s | — | 9.86s | 8.01s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 3 | 4 | <span class="benchmark-delta-negative">+33.3%</span> | 6 | 4 | <span class="benchmark-delta-positive">-33.3%</span> |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 3 | 4 | <span class="benchmark-delta-negative">+33.3%</span> | 6 | 4 | <span class="benchmark-delta-positive">-33.3%</span> |
| **Output Tokens** | 1670 | 1905 | <span class="benchmark-delta-negative">+14.1%</span> | 1747 | 1285 | <span class="benchmark-delta-positive">-26.4%</span> |
| **Reasoning / Thinking Tokens** | 992 | 1514 | <span class="benchmark-delta-negative">+52.6%</span> | 797 | 883 | <span class="benchmark-delta-negative">+10.8%</span> |
| **Total Input Tokens** | 79776 | 110162 | <span class="benchmark-delta-negative">+38.1%</span> | 100523 | 111253 | <span class="benchmark-delta-negative">+10.7%</span> |
| **Cached Input Tokens** | 64512 | 94976 | <span class="benchmark-delta-positive">+47.2%</span> | 87808 | 92928 | <span class="benchmark-delta-positive">+5.8%</span> |
| **Uncached Input Tokens** | 15264 | 15186 | <span class="benchmark-delta-positive">-0.5%</span> | 12715 | 18325 | <span class="benchmark-delta-negative">+44.1%</span> |
| **Cached vs Uncached Token Ratio** | 4.23:1 | 6.25:1 | <span class="benchmark-delta-positive">+48.0%</span> | 6.91:1 | 5.07:1 | <span class="benchmark-delta-negative">-26.6%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/server.go modified (9 lines -> 13 lines)
* **MCP Edit**: File api/server.go modified (9 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;api/*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;README*&#39; | sort &amp;&amp; \
sed -n &#39;1,240p&#39; api/server.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os
import tempfile

path = Path(&#39;api/server.go&#39;)
original = path.read_bytes()
needle = b&#39;func (s *Server) Start() {}\\n\\n&#39;
if original.count(needle) &#34;&#39;!= 1:
    raise SystemExit(&#39;&#34;&#39;expected insertion point exactly once&#39;)
updated = original.replace(
    needle,
    needle + b&#39;func InitServer() *Server {\\n\\treturn &amp;Server{}\\n}\\n\\n&#39;,
)
fd, temporary = tempfile.mkstemp(prefix=&#39;.server.go.&#39;, dir=path.parent)
try:
    with os.fdopen(fd, &#39;wb&#39;) as target:
        target.write(updated)
        target.flush()
        os.fsync(target.fileno())
    os.chmod(temporary, path.stat().st_mode)
    os.replace(temporary, path)
    directory_fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)
except BaseException:
    try:
        os.unlink(temporary)
    except FileNotFoundError:
        pass
    raise
PY
sed -n &#39;1,120p&#39; api/server.go&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;placement&#34;: &#34;public_end&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- api/server.go go.mod go.sum api/server_test.go &amp;&amp; \
sed -n &#39;1,120p&#39; api/server.go&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File api/server.go modified (9 lines -> 13 lines)
* **MCP Edit**: File api/server.go modified (9 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;type Server|func .*Server|&#39;&#39;^func [a-z]&#34; --glob &#39;&#34;&#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; api/server.go &amp;&amp; \
rg -n \&#34;type Server|func \\(.*\\) Init|func [a-z].*Server|func InitServer\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; api/server.go&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;source&#34;: &#34;func InitServer() *Server {\n\treturn &amp;Server{}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;before_symbol&#34;,
  &#34;target_symbol&#34;: &#34;internalRun&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; api/server.go &amp;&amp; \
git diff -- api/server.go go.mod go.sum api/server_test.go &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os
import tempfile

path = Path(&#39;api/server.go&#39;)
data = path.read_bytes()
old = b&#39;func (s *Server) Start() {}\\n\\nfunc (s *Server) internalRun() {}\\n&#39;
new = b&#39;func (s *Server) Start() {}\\n\\nfunc InitServer() *Server {\\n\\treturn &amp;Server{}\\n}\\n\\nfunc (s *Server) internalRun() {}\\n&#39;
if data.count(old) &#34;&#39;!= 1:
    raise SystemExit(&#39;&#34;&#39;expected server method block exactly once&#39;)
mode = path.stat().st_mode
fd, temporary = tempfile.mkstemp(prefix=&#39;.server.go.&#39;, dir=path.parent)
try:
    with os.fdopen(fd, &#39;wb&#39;) as stream:
        stream.write(data.replace(old, new))
        stream.flush()
        os.fsync(stream.fileno())
    os.chmod(temporary, mode)
    os.replace(temporary, path)
    directory_fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)
except BaseException:
    try:
        os.unlink(temporary)
    except FileNotFoundError:
        pass
    raise
PY
sed -n &#39;1,120p&#39; api/server.go
git diff -- api/server.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel &amp;&amp; \
git status --short -- api/server.go &amp;&amp; \
git ls-files --stage -- api/server.go &amp;&amp; \
git diff --no-index /dev/null api/server.go&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

