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

* **Date**: 2026-09-22 12:02:22 CEST

## Test case: `task-09-composite-refactor`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: default prompt · descriptive MCP instructions

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 42.44s | 67.37s | <span class="benchmark-delta-negative">+58.7%</span> | 53.60s | 110.23s | N/A |
| **Process Start → First Event** | 0.16s | 0.16s | — | 0.45s | 0.18s | — |
| **First Event → First Tool Call** | 7.43s | 8.24s | — | 7.23s | 12.77s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | N/A |
| **Internal Tool Cycles** | 5 | 7 | <span class="benchmark-delta-negative">+40.0%</span> | 7 | 7 | N/A |
| **Initial Load / Discovery Turns** | 5 | 3 | <span class="benchmark-delta-positive">-40.0%</span> | 7 | 2 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 5 | 7 | <span class="benchmark-delta-negative">+40.0%</span> | 7 | 7 | N/A |
| **Output Tokens** | 1536 | 1753 | <span class="benchmark-delta-negative">+14.1%</span> | 1747 | 1995 | N/A |
| **Reasoning / Thinking Tokens** | 473 | 729 | <span class="benchmark-delta-negative">+54.1%</span> | 501 | 710 | N/A |
| **Total Input Tokens** | 144959 | 215922 | <span class="benchmark-delta-negative">+49.0%</span> | 211595 | 186851 | N/A |
| **Cached Input Tokens** | 129280 | 194304 | <span class="benchmark-delta-positive">+50.3%</span> | 194048 | 156416 | N/A |
| **Uncached Input Tokens** | 15679 | 21618 | <span class="benchmark-delta-negative">+37.9%</span> | 17547 | 30435 | N/A |
| **Cached vs Uncached Token Ratio** | 8.25:1 | 8.99:1 | <span class="benchmark-delta-positive">+9.0%</span> | 11.06:1 | 5.14:1 | N/A |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 18 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*&#39; -g &#39;&#34;&#39;!tests/**&#39;&#34;&#39; -g &#39;&#34;&#39;!testdata/**&#39;&#34;&#39; | sort &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;AGENTS.md&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- files ---&#39; &amp;&amp; \
rg --files -g &#39;&#34;&#39;!tests/**&#39;&#34;&#39; -g &#39;&#34;&#39;!testdata/**&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- references ---&#39; &amp;&amp; \
rg -n --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &#39;\\bPort\\b|DefaultConfig|\\.Port&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; config/config.go; printf &#39;\\n--- go.mod ---\\n&#39;; sed -n &#39;1,120p&#39; go.mod; printf &#39;\\n--- files ---\\n&#39;; rg --files | head -80&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; config/config_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- go.mod ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- git status ---&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; config/config_test.go; rg -n &#39;\\bPort\\b|DefaultConfig|Config\\{&#39; . --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
if [ -f Makefile ]; then make check; else go test ./...; fi &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
git diff --name-only -- config/config.go config/config_test.go go.mod go.sum &amp;&amp; \
git status --short -- config/config.go config/config_test.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config.go &amp;&amp; \
git status --short &amp;&amp; \
git ls-files --stage config/config.go &amp;&amp; \
git diff --no-ext-diff -- config/config.go&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080}\n}&#34;,
  &#34;placement&#34;: &#34;public_end&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; config/config.go; printf &#39;\\n--- status ---\\n&#39;; git status --short; printf &#39;\\n--- diff ---\\n&#39;; git diff -- config/config.go config/config_test.go go.mod go.sum; printf &#39;\\n--- make check availability ---\\n&#39;; if [ -f Makefile ]; then make check; else echo &#39;no Makefile&#39;; fi; printf &#39;\\n--- go test ---\\n&#39;; go test ./...&#34;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- config/config.go config/config_test.go go.mod go.sum; git ls-files --stage -- config/config.go config/config_test.go go.mod go.sum; git diff --no-index /dev/null config/config.go || \
true&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; config/config.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; -g &#39;config/config.go&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*&#39; | sort &amp;&amp; \
printf &#39;\\n--- config/config.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; config/config.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; &#39;\\b(Config|Port|ListenPort|DefaultConfig)\\b|config\\.&#39; . &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Config\\{|\\.Port\\b|DefaultConfig|Addr\\(|Host:\&#34; . -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!*.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; &amp;&amp; \
printf &#39;\\n--- files ---\\n&#39; &amp;&amp; \
rg --files | sort | sed -n &#39;1,160p&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; config/config_test.go &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort | head -80&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;command -v semedit || \
true; command -v gofmt || \
true; sed -n &#39;1,120p&#39; config/config_test.go; sed -n &#39;1,200p&#39; Makefile 2&gt;/dev/null || \
true&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;localhost|8080|ListenPort|DefaultConfig|Config\\{&#39; --glob &#39;&#34;&#39;!config/config.go&#39;&#34;&#39; --glob &#39;&#34;&#39;!config/config_test.go&#39;&#34;&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
make check&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w config/config.go &amp;&amp; \
git diff --check &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
git status --short -- go.mod go.sum config/config.go config/config_test.go&#39;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./config &amp;&amp; \
git diff -- config/config.go go.mod go.sum &amp;&amp; \
git status --short -- config/config.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
printf &#39;\\n--- config diff ---\\n&#39; &amp;&amp; \
git diff -- config/config.go &amp;&amp; \
printf &#39;\\n--- go test ---\\n&#39; &amp;&amp; \
go test ./...&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; config/config.go &amp;&amp; \
git ls-files -v -- config/config.go go.mod go.sum &amp;&amp; \
git diff --no-ext-diff -- config/config.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; config/config.go; printf &#39;\\n--- git metadata ---\\n&#39;; git rev-parse --show-toplevel; git ls-files --stage -- config/config.go; git status --short --untracked-files=all -- config/config.go config/config_test.go go.mod go.sum&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not discover any callable `semantic_*` tools from the available semedit MCP server in this session. I therefore used a direct focused source edit instead.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 9.76s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not combine the operations because `semantic_batch`’s edit schema was not clear from the available tool documentation, while the individual semantic operations were directly applicable and self-verifying. Batching was therefore unsuitable under the time and correctness constraints.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.08s; turns: 1; tool calls: 0.</p>
</details>
