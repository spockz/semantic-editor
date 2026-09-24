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

## Test case: `task-09-composite-refactor`

### Target: `codex/gpt-5.6-luna/high (repeat 1)`

#### Configuration: default prompt · prescriptive MCP instructions

* **Run Provenance**: `binary=44027cf7b57a0b4e`

* **Fixture**: [testdata/bench/task_09_composite_refactor.txtar](https://github.com/spockz/semantic-editor/blob/dd39d52587030a6aae7bbbf05b381f54bb386d3a/testdata/bench/task_09_composite_refactor.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert helper function DefaultConfig() *Config returning default values, and clean up unused imports. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert helper function DefaultConfig() *Config returning default values, and clean up unused imports. Prefer using semantic editor operations if applicable. When done, output DONE.

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
| **Wall-Clock Latency** | 44.40s | 78.92s | <span class="benchmark-delta-negative">+77.7%</span> | 45.85s | 79.08s | <span class="benchmark-delta-negative">+72.5%</span> |
| **Process Start → First Event** | 0.12s | 0.14s | — | 0.12s | 0.12s | — |
| **First Event → First Tool Call** | 7.99s | 8.02s | — | 8.98s | 8.27s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 4 | 10 | <span class="benchmark-delta-negative">+150.0%</span> | 6 | 8 | <span class="benchmark-delta-negative">+33.3%</span> |
| **Initial Load / Discovery Turns** | 4 | 1 | <span class="benchmark-delta-positive">-75.0%</span> | 6 | 2 | <span class="benchmark-delta-positive">-66.7%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 4 | 10 | <span class="benchmark-delta-negative">+150.0%</span> | 6 | 8 | <span class="benchmark-delta-negative">+33.3%</span> |
| **Output Tokens** | 1740 | 2349 | <span class="benchmark-delta-negative">+35.0%</span> | 1724 | 2720 | <span class="benchmark-delta-negative">+57.8%</span> |
| **Reasoning / Thinking Tokens** | 751 | 954 | <span class="benchmark-delta-negative">+27.0%</span> | 553 | 1325 | <span class="benchmark-delta-negative">+139.6%</span> |
| **Total Input Tokens** | 124885 | 333236 | <span class="benchmark-delta-negative">+166.8%</span> | 167097 | 274387 | <span class="benchmark-delta-negative">+64.2%</span> |
| **Cached Input Tokens** | 110080 | 290816 | <span class="benchmark-delta-positive">+164.2%</span> | 151552 | 241152 | <span class="benchmark-delta-positive">+59.1%</span> |
| **Uncached Input Tokens** | 14805 | 42420 | <span class="benchmark-delta-negative">+186.5%</span> | 15545 | 33235 | <span class="benchmark-delta-negative">+113.8%</span> |
| **Cached vs Uncached Token Ratio** | 7.44:1 | 6.86:1 | <span class="benchmark-delta-negative">-7.8%</span> | 9.75:1 | 7.26:1 | <span class="benchmark-delta-negative">-25.6%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;Makefile&#39; -g &#39;config/config.go&#39; -g &#39;*.go&#39; | sort &amp;&amp; \
printf &#39;\\n--- config/config.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; config/config.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;AGENTS*.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; -g &#39;.codex/**&#39; | sort &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,220p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- config/config_test.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; config/config_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- git status ---&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tracked files ---&#39; &amp;&amp; \
git ls-files&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- go.mod ---&#39; &amp;&amp; \
sed -n &#39;1,160p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- references ---&#39; &amp;&amp; \
rg -n &#39;\\.Port\\b|Config\\{|DefaultConfig|semantic_(rename|insert|replace|batch)|semedit&#39; -g &#39;*.go&#39; -g &#39;*.md&#39; -g &#39;*.txtar&#39; . || \
true&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w config/config.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- config/config.go ---&#39; &amp;&amp; \
sed -n &#39;1,200p&#39; config/config.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- module files ---&#39; &amp;&amp; \
rg --files -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- diff ---&#39; &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- test ---&#39; &amp;&amp; \
go test ./config&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; config/config_test.go &amp;&amp; \
git status --short &amp;&amp; \
git diff -- config/config.go config/config_test.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git status --short -- config/config.go config/config_test.go go.mod go.sum &amp;&amp; \
git diff --check -- config/config.go&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;,
  &#34;placement&#34;: &#34;public_start&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; config/config.go &amp;&amp; \
git diff -- config/config.go config/config_test.go go.mod go.sum &amp;&amp; \
git status --short -- config/config.go config/config_test.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-files --stage -- config/config.go config/config_test.go go.mod go.sum; git status --short --untracked-files=all | rg &#39;config|go\\.mod|go\\.sum&#39; || \
true; git diff --no-ext-diff -- config/config.go; git diff --no-index /dev/null config/config.go || \
true&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d config/config.go &amp;&amp; \
go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 18 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,220p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; -g &#39;cmd/**&#39; -g &#39;*semantic*&#39; | sort &amp;&amp; \
printf &#39;\\n--- config/config.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go &amp;&amp; \
printf &#39;\\n--- top-level ---\\n&#39; &amp;&amp; \
ls -la&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &#39;Config|\\.Port|Port:&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- references ---&#39; &amp;&amp; \
rg -n --glob &#39;*.go&#39; &#39;\\b(Config|\\.Port|Port:|DefaultConfig|Addr\\()\\b&#39; . &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- go.mod ---&#39; &amp;&amp; \
sed -n &#39;1,80p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- source files ---&#39; &amp;&amp; \
rg --files -g &#39;*.go&#39; | sort | xargs -n1 sh -c &#39;echo --- \&#34;&#34;&#39;$0&#34;; sed -n &#34;1,180p&#34; &#34;$0&#34;&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; config/config_test.go &amp;&amp; \
git status --short &amp;&amp; \
git diff -- config/config.go&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
go test ./config &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
git status --short -- config/config.go config/config_test.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080}\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config.go &amp;&amp; \
git ls-files --stage -- config/config.go config/config_test.go go.mod go.sum &amp;&amp; \
git status --short --untracked-files=all -- config&#34;</code></pre></td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff --no-ext-diff -- config/config.go config/config_test.go go.mod go.sum; git status --short --untracked-files=all -- config/config.go config/config_test.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- config/config.go ---&#39; &amp;&amp; \
sed -n &#39;1,180p&#39; config/config.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- status ---&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- diff ---&#39; &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected-file diff check ---&#39; &amp;&amp; \
git diff --quiet -- go.mod go.sum; printf &#39;protected_diff_exit=%s\\n&#39; \&#34;&#34;&#39;$?&#34;&#39;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- git root ---&#39; &amp;&amp; \
git rev-parse --show-toplevel &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tracked config ---&#39; &amp;&amp; \
git ls-files --stage -- config/config.go config/config_test.go go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- config file metadata ---&#39; &amp;&amp; \
stat -f &#39;%N %z bytes %Sm&#39; config/config.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- make check ---&#39; &amp;&amp; \
make check&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -d config/config.go &amp;&amp; \
go test ./... &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- relevant references ---&#39; &amp;&amp; \
rg -n --glob &#39;*.go&#39; &#39;\\b(Port|ListenPort|DefaultConfig)\\b&#39; config &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- fixture status ---&#39; &amp;&amp; \
git status --short -- config go.mod go.sum&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I had discovered `semantic_batch`, but used individual calls because the rename and function insertion had distinct tool-specific arguments and each standalone operation provided its own verification and import handling. Batching was possible; it was a deliberate clarity and risk-reduction choice, not a tooling limitation.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 7.26s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I had discovered `semantic_batch`, but used separate semantic calls so each rename, insertion, and import cleanup could independently verify diagnostics before the next operation. Batching was possible but not used; it was a workflow choice, not a discovery or capability issue.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 7.90s; turns: 1; tool calls: 0.</p>
</details>

