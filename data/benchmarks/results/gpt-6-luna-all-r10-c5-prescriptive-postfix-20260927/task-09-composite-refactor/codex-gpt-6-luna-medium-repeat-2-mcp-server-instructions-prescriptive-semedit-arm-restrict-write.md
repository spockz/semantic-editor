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

## Test case: `task-09-composite-refactor`

### Target: `codex/gpt-6-luna/medium (repeat 2)`

#### Configuration: default prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/task_09_composite_refactor.txtar](https://github.com/spockz/semantic-editor/blob/c03f3625ef877adfd183a978ca333ca208eff0f0/testdata/bench/task_09_composite_refactor.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert helper function DefaultConfig() *Config returning default values, and clean up unused imports. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert helper function DefaultConfig() *Config returning default values, and clean up unused imports. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package config

import (
	"archive/tar"
	"fmt"
)

type Config struct {
	Host string
	Port int
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 23.27s | 74.51s | <span class="benchmark-delta-negative">+220.2%</span> | 31.68s | 70.21s | <span class="benchmark-delta-negative">+121.6%</span> |
| **Process Start → First Event** | 0.13s | 0.12s | — | 0.12s | 0.12s | — |
| **First Event → First Tool Call** | 4.49s | 6.43s | — | 6.43s | 7.29s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 5 | 11 | <span class="benchmark-delta-negative">+120.0%</span> | 5 | 13 | <span class="benchmark-delta-negative">+160.0%</span> |
| **Initial Load / Discovery Turns** | 5 | 1 | <span class="benchmark-delta-positive">-80.0%</span> | 3 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 5 | 11 | <span class="benchmark-delta-negative">+120.0%</span> | 5 | 13 | <span class="benchmark-delta-negative">+160.0%</span> |
| **Output Tokens** | 885 | 2429 | <span class="benchmark-delta-negative">+174.5%</span> | 959 | 1285 | <span class="benchmark-delta-negative">+34.0%</span> |
| **Reasoning / Thinking Tokens** | 186 | 1079 | <span class="benchmark-delta-negative">+480.1%</span> | 137 | 416 | <span class="benchmark-delta-negative">+203.6%</span> |
| **Total Input Tokens** | 116665 | 405292 | <span class="benchmark-delta-negative">+247.4%</span> | 117019 | 290406 | <span class="benchmark-delta-negative">+148.2%</span> |
| **Cached Input Tokens** | 104960 | 377088 | <span class="benchmark-delta-positive">+259.3%</span> | 105984 | 264448 | <span class="benchmark-delta-positive">+149.5%</span> |
| **Uncached Input Tokens** | 11705 | 28204 | <span class="benchmark-delta-negative">+141.0%</span> | 11035 | 25958 | <span class="benchmark-delta-negative">+135.2%</span> |
| **Cached vs Uncached Token Ratio** | 8.97:1 | 13.37:1 | <span class="benchmark-delta-positive">+49.1%</span> | 9.60:1 | 10.19:1 | <span class="benchmark-delta-positive">+6.1%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 19 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 19 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; config/config.go &amp;&amp; \
rg -n &#39;\\.Port|Port:&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;config/config.go&#39; -g &#39;AGENTS.md&#39; | head -40&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; | head -80 &amp;&amp; \
rg -n &#39;DefaultConfig|ListenPort|Host|config.Config|Config\\{&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Config&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat config/config_test.go &amp;&amp; \
rg --files&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files | head -80&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat &gt; config/config.go.tmp &lt;&lt;&#39;EOF&#39;
package config

import \&#34;fmt\&#34;

type Config struct {
	Host       string
	ListenPort int
}

func DefaultConfig() *Config {
	return &amp;Config{
		Host:       \&#34;localhost\&#34;,
		ListenPort: 8080,
	}
}

func (c *Config) Addr() string {
	return fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)
}
EOF
mv config/config.go.tmp config/config.go&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Config&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- config/config.go go.mod go.sum &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat config/config.go&#39;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_insert_structure</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): overwrite is unsupported for type structure insertion<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;kind&#34;: &#34;type&#34;,
  &#34;source&#34;: &#34;type Config struct {\n\tHost string\n\tListenPort int\n}&#34;,
  &#34;overwrite&#34;: true,
  &#34;access_modifier&#34;: &#34;public&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;SUBOPTIMAL|PROJECT_BRIEF&#39; || \
true&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): load Go package example.com/test/config: example.com/test/config: /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-default_task-09-composite-refactor_1790468333602217000/config/config.go:4:2: &#34;archive/tar&#34; imported and not used<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile
root = Path(&#39;.&#39;)
log = root / &#39;docs&#39; / &#39;SUBOPTIMAL_TOOLS.md&#39;
log.parent.mkdir(parents=True, exist_ok=True)
entry = &#39;&#39;&#39;# Suboptimal Tool Behavior\\n\\n- Tool: &#34;&#39;`semantic_insert_structure`&#39;&#34;\\n  Target: &#34;&#39;`config/config.go`&#39;&#34;\\n  Observed failure: rejected &#34;&#39;`overwrite=true` for an existing type structure.&#39;&#34;\\n  Workaround: use a bounded source edit because the available structural operation cannot update this declaration.\\n  Root cause: the tool supports insertion only for this structure kind.\\n- Tool: &#34;&#39;`semantic_find_references`&#39;&#34;\\n  Target: &#34;&#39;`config/config.go`&#39;&#34;\\n  Observed failure: Go package loading failed because &#34;&#39;`archive/tar` is already unused.&#39;&#34;\\n  Workaround: make the requested source cleanup and declaration changes directly, then run the required checks.\\n  Root cause: package diagnostics block semantic reference queries before imports are cleaned.\\n&#39;&#39;&#39;
source = &#39;&#39;&#39;package config\\n\\nimport \&#34;fmt\&#34;\\n\\ntype Config struct {\\n\\tHost       string\\n\\tListenPort int\\n}\\n\\nfunc DefaultConfig() *Config {\\n\\treturn &amp;Config{\\n\\t\\tHost:       \&#34;localhost\&#34;,\\n\\t\\tListenPort: 8080,\\n\\t}\\n}\\n\\nfunc (c *Config) Addr() string {\\n\\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)\\n}\\n&#39;&#39;&#39;

def atomic_write(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, name = tempfile.mkstemp(dir=path.parent, prefix=path.name + &#39;.&#39;)
    try:
        with os.fdopen(fd, &#39;w&#39;) as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
        os.replace(name, path)
        os.utime(path, None)
    finally:
        if os.path.exists(name):
            os.unlink(name)

atomic_write(log, entry)
atomic_write(root / &#39;config&#39; / &#39;config.go&#39;, source)
PY
&#34;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- config/config.go docs/SUBOPTIMAL_TOOLS.md&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 19 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; config/config.go &amp;&amp; \
rg -n &#39;\\.Port\\b|Port:&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Config&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;DefaultConfig|ListenPort|Host|port|default&#39; --glob &#39;*.go&#39; --glob &#39;*.md&#39; .&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Config&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat config/config_test.go; rg -n &#39;config\\.Config|Config\\{&#39; --glob &#39;*.go&#39; .; rg --files&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve DefaultConfig in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-default_task-09-composite-refactor_1790468529325490000/config/config.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;DefaultConfig&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile
path = Path(&#39;config/config.go&#39;)
content = &#39;&#39;&#39;package config

import \&#34;fmt\&#34;

type Config struct {
\\tHost       string
\\tListenPort int
}

func DefaultConfig() *Config {
\\treturn &amp;Config{
\\t\\tHost:       \&#34;localhost\&#34;,
\\t\\tListenPort: 8080,
\\t}
}

func (c *Config) Addr() string {
\\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)
}
&#39;&#39;&#39;
fd, temp_name = tempfile.mkstemp(dir=path.parent, prefix=path.name + &#39;.&#39;)
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
gofmt -w config/config.go
git diff -- config/config.go&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat config/config.go; git status --short -- config/config.go config/config_test.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Config.Addr&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;public_start&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- config/config.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff --stat&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;DefaultConfig&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Config&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I discovered `semantic_batch`, but mistakenly called the rename, insertion, and import cleanup as separate operations instead of batching them. The edits were sequential and batchable; there was no technical blocker.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

