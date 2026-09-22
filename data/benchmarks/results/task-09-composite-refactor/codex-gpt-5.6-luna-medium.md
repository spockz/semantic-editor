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

## Test case: `task-09-composite-refactor`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: default prompt · none MCP instructions

* **Fixture**: `testdata/bench/task_09_composite_refactor.txtar` (`sha256:56702db0bd2306695e2f66f5 (uncommitted)`)

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
| **Wall-Clock Latency** | 46.94s | 65.12s | <span class="benchmark-delta-negative">+38.7%</span> | 62.72s | 208.85s | <span class="benchmark-delta-negative">+233.0%</span> |
| **Process Start → First Event** | 0.14s | 0.19s | — | 0.21s | 0.14s | — |
| **First Event → First Tool Call** | 7.68s | 12.65s | — | 12.37s | 20.73s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 6 | 7 | <span class="benchmark-delta-negative">+16.7%</span> | 4 | 12 | <span class="benchmark-delta-negative">+200.0%</span> |
| **Initial Load / Discovery Turns** | 6 | 3 | <span class="benchmark-delta-positive">-50.0%</span> | 4 | 3 | <span class="benchmark-delta-positive">-25.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 6 | 7 | <span class="benchmark-delta-negative">+16.7%</span> | 4 | 12 | <span class="benchmark-delta-negative">+200.0%</span> |
| **Output Tokens** | 1630 | 1435 | <span class="benchmark-delta-positive">-12.0%</span> | 1106 | 3137 | <span class="benchmark-delta-negative">+183.6%</span> |
| **Reasoning / Thinking Tokens** | 435 | 407 | <span class="benchmark-delta-positive">-6.4%</span> | 255 | 1163 | <span class="benchmark-delta-negative">+356.1%</span> |
| **Total Input Tokens** | 167680 | 256843 | <span class="benchmark-delta-negative">+53.2%</span> | 122321 | 428782 | <span class="benchmark-delta-negative">+250.5%</span> |
| **Cached Input Tokens** | 150528 | 235008 | <span class="benchmark-delta-positive">+56.1%</span> | 98816 | 382976 | <span class="benchmark-delta-positive">+287.6%</span> |
| **Uncached Input Tokens** | 17152 | 21835 | <span class="benchmark-delta-negative">+27.3%</span> | 23505 | 45806 | <span class="benchmark-delta-negative">+94.9%</span> |
| **Cached vs Uncached Token Ratio** | 8.78:1 | 10.76:1 | <span class="benchmark-delta-positive">+22.6%</span> | 4.20:1 | 8.36:1 | <span class="benchmark-delta-positive">+98.9%</span> |
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
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | sort &amp;&amp; \
printf &#39;\\n--- config/config.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; config/config.go &amp;&amp; \
printf &#39;\\n--- PROJECT_BRIEF.md ---\\n&#39; &amp;&amp; \
if test -f PROJECT_BRIEF.md; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;config/config.go&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Config|\\.Port|ListenPort|DefaultConfig|Addr\\(\&#34; --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &#39;\\bPort\\b|DefaultConfig|archive/tar&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; config/config_test.go &amp;&amp; \
printf &#39;\\n--- git status ---\\n&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;\\n--- module ---\\n&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
printf &#39;\\n--- Makefile candidates ---\\n&#39; &amp;&amp; \
if test -f Makefile; then sed -n &#39;1,200p&#39; Makefile; else echo &#39;(none)&#39;; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config_test.go &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -not -name &#39;go.mod&#39; -not -name &#39;go.sum&#39; -print | sort&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
go test ./config &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
git status --short -- config/config.go go.mod go.sum config/config_test.go&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; config/config.go &amp;&amp; \
git status --short --untracked-files=all | head -40 &amp;&amp; \
git diff --no-ext-diff -- config/config.go&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Config&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git ls-files --stage -- config/config.go config/config_test.go; git check-ignore -v config/config.go || \
true; git diff --no-index /dev/null config/config.go || \
true&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config.go &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
if [ -x ./make ]; then ./make check; elif [ -f Makefile ]; then make check; else go test ./config; fi&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check -- config/config.go &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
git status --short -- config/config.go config/config_test.go go.mod go.sum&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*&#39; | sort &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;AGENTS.md&#39; -g &#39;docs/adr/README.md&#39; -g &#39;Makefile&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,220p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
if [ -f docs/adr/README.md ]; then sed -n &#39;1,160p&#39; docs/adr/README.md; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; &#39;\\b(Port|ListenPort|DefaultConfig|\\.Addr\\()\\b&#39; . &amp;&amp; \
rg --files -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &#39;Config|\\.Port|ListenPort|DefaultConfig|archive/tar&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config_test.go &amp;&amp; \
sed -n &#39;1,160p&#39; api/router.go &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config_test.go &amp;&amp; \
rg -n --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &#39;8080|localhost|Host:&#39; .&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- config/config.go go.mod go.sum config/config_test.go&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): calculate function offset: target symbol not found: strategy &#34;before_symbol&#34;, target symbol &#34;(*Config).Addr&#34;, in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-default_task-09-composite-refactor_1790070673717973000/config/config.go<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;,
  &#34;placement&#34;: &#34;before_symbol&#34;,
  &#34;target_symbol&#34;: &#34;(*Config).Addr&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;if [ -f docs/SUBOPTIMAL_TOOLS.md ]; then sed -n &#39;1,220p&#39; docs/SUBOPTIMAL_TOOLS.md; else echo &#39;__MISSING__&#39;; fi&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- config/config.go docs/SUBOPTIMAL_TOOLS.md &amp;&amp; \
make check&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config.go; echo &#39;---&#39;; git status --short -- config/config.go config/config_test.go docs/SUBOPTIMAL_TOOLS.md; echo &#39;---&#39;; git rev-parse --show-toplevel; echo &#39;---&#39;; rg -n &#39;ListenPort|DefaultConfig|archive/tar&#39; config/config.go docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- files&#39; &amp;&amp; \
rg --files -g &#39;go.mod&#39; -g &#39;Makefile&#39; -g &#39;*.go&#39; | sort &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- module&#39; &amp;&amp; \
if [ -f go.mod ]; then sed -n &#39;1,80p&#39; go.mod; fi &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected status&#39; &amp;&amp; \
git status --short -- go.mod go.sum config/config_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- go test&#39; &amp;&amp; \
go test ./...&#34;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -l docs/SUBOPTIMAL_TOOLS.md 2&gt;/dev/null || \
true; git status --short --untracked-files=all | rg &#39;config|SUBOPTIMAL|go\\.mod|go\\.sum&#39; || \
true; git check-ignore -v config/config.go docs/SUBOPTIMAL_TOOLS.md 2&gt;/dev/null || \
true; git diff --no-index /dev/null docs/SUBOPTIMAL_TOOLS.md 2&gt;/dev/null | sed -n &#39;1,80p&#39; || \
true&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because it was not selected after discovering the individual semantic operations, and the rename and insertion were executed as separate sequential calls. Batching would have been suitable, but I did not recognize and apply it in time.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.95s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because it was discovered but not used; I chose separate semantic operations to handle the rename, insertion, and import cleanup independently. The insertion initially failed on its target symbol, so sequential calls made the failure and required fallback explicit.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 8.09s; turns: 1; tool calls: 0.</p>
</details>
