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

* **Date**: 2026-09-22 12:08:49 CEST

## Test case: `task-09-composite-refactor`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: default prompt · prescriptive MCP instructions

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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 48.49s | 115.34s | N/A | 60.43s | 82.56s | <span class="benchmark-delta-negative">+36.6%</span> |
| **Process Start → First Event** | 0.14s | 0.79s | — | 0.14s | 0.22s | — |
| **First Event → First Tool Call** | 7.43s | 14.54s | — | 10.09s | 8.34s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | 1 | 1 | 0% |
| **Internal Tool Cycles** | 6 | 8 | N/A | 6 | 11 | <span class="benchmark-delta-negative">+83.3%</span> |
| **Initial Load / Discovery Turns** | 6 | 1 | N/A | 6 | 4 | <span class="benchmark-delta-positive">-33.3%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 6 | 8 | N/A | 6 | 11 | <span class="benchmark-delta-negative">+83.3%</span> |
| **Output Tokens** | 1669 | 2201 | N/A | 1604 | 2270 | <span class="benchmark-delta-negative">+41.5%</span> |
| **Reasoning / Thinking Tokens** | 441 | 608 | N/A | 393 | 723 | <span class="benchmark-delta-negative">+84.0%</span> |
| **Total Input Tokens** | 188601 | 235206 | N/A | 189880 | 452447 | <span class="benchmark-delta-negative">+138.3%</span> |
| **Cached Input Tokens** | 171776 | 213248 | N/A | 162560 | 420096 | <span class="benchmark-delta-positive">+158.4%</span> |
| **Uncached Input Tokens** | 16825 | 21958 | N/A | 27320 | 32351 | <span class="benchmark-delta-negative">+18.4%</span> |
| **Cached vs Uncached Token Ratio** | 10.21:1 | 9.71:1 | N/A | 5.95:1 | 12.99:1 | <span class="benchmark-delta-positive">+118.2%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; | sort &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;README.md&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; -g &#39;AGENTS*&#39; | sort &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,220p&#39; PROJECT_BRIEF.md; fi &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,180p&#39; Makefile; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | sort &amp;&amp; \
rg -n \&#34;Config|ListenPort|Port|DefaultConfig|Addr\&#34; . --glob &#39;&#34;&#39;!config/config.go&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --hidden -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &#39;\\bPort\\b|Config\\{&#39; .; command -v semedit || \
true; rg -n --hidden -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; &#39;semantic_(rename|insert_function|replace_body|batch|verify)|semedit&#39; . || \
true; rg --files -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;test*&#39; -g &#39;*.sh&#39; -g &#39;*.txtar&#39; | sort&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; config/config_test.go &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
find . -maxdepth 3 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
make check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; config/config_test.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config.go &amp;&amp; \
git status --short -- config/config.go config/config_test.go go.mod go.sum &amp;&amp; \
go test ./config&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
make check&#39;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-files --stage -- config/config.go config/config_test.go go.mod go.sum; git diff --no-ext-diff -- config/config.go; git diff --no-ext-diff --name-only -- config/config_test.go go.mod go.sum; git status --short --untracked-files=all | rg &#39;(&#34;&#39;^|/)(config/config.go|config/config_test.go|go.mod|go.sum)$&#39;&#34;&#39; || \
true&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./config &amp;&amp; \
git diff --check -- config/config.go &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
git status --short -- config/config.go go.mod go.sum config/config_test.go&#39;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
go test ./config &amp;&amp; \
git diff --check -- config/config.go &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
git status --short -- config/config.go go.mod go.sum config/config_test.go&#39;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; config/config.go; git ls-files --stage -- config/config.go config/config_test.go; git status --short --untracked-files=all -- config&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;AGENTS*.md&#39; -g &#39;Makefile&#39; | sort &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; -g &#39;docs/SUBOPTIMAL_TOOLS.md&#39; -g &#39;.codex/**&#39; | sort &amp;&amp; \
sed -n &#39;1,240p&#39; config/config.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Config|ListenPort|DefaultConfig|\\.Port|Addr\\(\&#34; . --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; &#39;\\bPort\\b|Config\\{&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config_test.go &amp;&amp; \
git diff -- config/config.go config/config_test.go &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n -i &#39;default|listen|port|config&#39; --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; . | head -120&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;localhost|8080|default|Port\&#34; README.md . --glob &#39;&#34;&#39;!config/config.go&#39;&#34;&#39; --glob &#39;&#34;&#39;!config/config_test.go&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; || \
true&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config_test.go &amp;&amp; \
rg -n &#39;localhost|127\\.0\\.0\\.1|8080|ListenPort|DefaultConfig&#39; --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; .&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
git diff --check &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
git status --short -- config/config.go config/config_test.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go test ./config &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
git status --short -- config/config.go config/config_test.go go.mod go.sum &amp;&amp; \
if [ -f Makefile ]; then make check; else echo &#39;make check: no Makefile present&#39;; fi&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;,
  &#34;placement&#34;: &#34;before_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Addr&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; config/config.go &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
sed -n &#39;1,220p&#39; Makefile&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
git diff --check &amp;&amp; \
go test ./... &amp;&amp; \
git status --short &amp;&amp; \
git diff -- config/config.go config/config_test.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check -- config/config.go &amp;&amp; \
go test ./... &amp;&amp; \
git status --short &amp;&amp; \
git diff -- config/config.go config/config_test.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel &amp;&amp; \
git diff --no-ext-diff -- config/config.go &amp;&amp; \
git ls-files -- config/config.go &amp;&amp; \
git status --short -- config/config.go config/config_test.go go.mod go.sum &amp;&amp; \
sed -n &#39;1,120p&#39; config/config.go&#34;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-error"><div class="callout-title"><span>✕</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not call a `semantic_*` tool because no semedit MCP server or semantic-operation tools were discoverable in the available tool context. I therefore used ordinary editing.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 7.08s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because I discovered its availability only after identifying the individual semantic operations, and proceeded with the already-known standalone calls. Batching was suitable in principle but was not used due to that discovery/order-of-operations issue.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 4.77s; turns: 1; tool calls: 0.</p>
</details>
