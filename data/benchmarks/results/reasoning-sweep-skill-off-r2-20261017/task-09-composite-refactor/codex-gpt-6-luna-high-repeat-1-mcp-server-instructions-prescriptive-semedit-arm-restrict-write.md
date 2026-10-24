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

* **Date**: 2026-10-17 21:43:42 CEST

## Test case: `task-09-composite-refactor`

### Target: `codex/gpt-6-luna/high (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

* **Fixture**: [testdata/bench/task_09_composite_refactor.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_09_composite_refactor.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert DefaultConfig() *Config returning Host localhost and ListenPort 8080, and clean up unused imports. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

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

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 11.48s | 23.57s | <span class="benchmark-delta-negative">+105.4%</span> | — | — | — |
| **Process Start → First Event** | 0.03s | 0.03s | — | — | — | — |
| **First Event → First Tool Call** | 3.55s | 6.46s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 5 | <span class="benchmark-delta-negative">+25.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 5 | <span class="benchmark-delta-negative">+25.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.002283 | $0.006680 | +192.6% | — | — | — |
| **Output Tokens** | 551 | 1154 | <span class="benchmark-delta-negative">+109.4%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 172 | 592 | <span class="benchmark-delta-negative">+244.2%</span> | — | — | — |
| **Total Input Tokens** | 87816 | 261478 | <span class="benchmark-delta-negative">+197.8%</span> | — | — | — |
| **Cached Input Tokens** | 75264 | 222720 | <span class="benchmark-delta-positive">+195.9%</span> | — | — | — |
| **Uncached Input Tokens** | 12552 | 38758 | <span class="benchmark-delta-negative">+208.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.00:1 | 5.75:1 | <span class="benchmark-delta-negative">-4.2%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 16 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;type Config|\\.Port\\b|ListenPort|DefaultConfig\&#34; .&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;config&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat config/config.go &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Addr&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: warning: Not a git repository. Use --no-index to compare two paths outside a working tree usage: git diff --no-index [&lt;options&gt;] &lt;path&gt; &lt;path&gt; Diff output format options -p, --patch generate patch -s, --no-patch suppress diff output -u generate patch -U, --unified[=&lt;n&gt;] generate diffs with &lt;n&gt; lines context -W, --[no-]function-context generate diffs with &lt;n&gt; lines context --raw generate the diff in raw format --patch-with-raw synonym for &#39;-p --raw&#39; --patch-with-stat synonym for &#39;-p --stat&#39; --numstat machine<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- config/config.go&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Addr&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;symbol&#34;: &#34;Config.Port&#34;,
        &#34;to&#34;: &#34;ListenPort&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;source&#34;: &#34;func DefaultConfig() *Config { return &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080} }&#34;,
        &#34;placement&#34;: &#34;public_end&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;
      }
    }
  ]
}</code></pre></td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

* **Fixture**: [testdata/bench/task_09_composite_refactor.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_09_composite_refactor.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert DefaultConfig() *Config returning Host localhost and ListenPort 8080, and clean up unused imports. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert DefaultConfig() *Config returning Host localhost and ListenPort 8080, and clean up unused imports. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

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
| **Wall-Clock Latency** | 11.48s | 27.07s | <span class="benchmark-delta-negative">+135.9%</span> | — | — | — |
| **Process Start → First Event** | 0.03s | 0.03s | — | — | — | — |
| **First Event → First Tool Call** | 3.55s | 3.94s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 10 | <span class="benchmark-delta-negative">+150.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 10 | <span class="benchmark-delta-negative">+150.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.002283 | $0.007510 | +228.9% | — | — | — |
| **Output Tokens** | 551 | 1860 | <span class="benchmark-delta-negative">+237.6%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 172 | 804 | <span class="benchmark-delta-negative">+367.4%</span> | — | — | — |
| **Total Input Tokens** | 87816 | 309559 | <span class="benchmark-delta-negative">+252.5%</span> | — | — | — |
| **Cached Input Tokens** | 75264 | 270848 | <span class="benchmark-delta-positive">+259.9%</span> | — | — | — |
| **Uncached Input Tokens** | 12552 | 38711 | <span class="benchmark-delta-negative">+208.4%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.00:1 | 7.00:1 | <span class="benchmark-delta-positive">+16.7%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 18 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;type Config|\\.Port\\b|ListenPort|DefaultConfig\&#34; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat config/config.go &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242981274030000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat config/config.go &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;go.mod&#39;&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: warning: Not a git repository. Use --no-index to compare two paths outside a working tree usage: git diff --no-index [&lt;options&gt;] &lt;path&gt; &lt;path&gt; Diff output format options -p, --patch generate patch -s, --no-patch suppress diff output -u generate patch -U, --unified[=&lt;n&gt;] generate diffs with &lt;n&gt; lines context -W, --[no-]function-context generate diffs with &lt;n&gt; lines context --raw generate the diff in raw format --patch-with-raw synonym for &#39;-p --raw&#39; --patch-with-stat synonym for &#39;-p --stat&#39; --numstat machine<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- config/config.go&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;config/config.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242981274030000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242981274030000&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242981274030000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;expected_revision is only valid for field, element, and statement/expression insertion: invalid operation parameters&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080}\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;group&#34;: &#34;append&#34;,
  &#34;expected_revision&#34;: &#34;cd1f9a46b731a7411c7d397dc6beb17b7f78f019f0bf41d46743b149c284bc35&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242981274030000&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080}\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;group&#34;: &#34;append&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242981274030000&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: warning: Not a git repository. Use --no-index to compare two paths outside a working tree usage: git diff --no-index [&lt;options&gt;] &lt;path&gt; &lt;path&gt; Diff output format options -p, --patch generate patch -s, --no-patch suppress diff output -u generate patch -U, --unified[=&lt;n&gt;] generate diffs with &lt;n&gt; lines context -W, --[no-]function-context generate diffs with &lt;n&gt; lines context --raw generate the diff in raw format --patch-with-raw synonym for &#39;-p --raw&#39; --patch-with-stat synonym for &#39;-p --stat&#39; --numstat machine<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- config/config.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat config/config.go&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was available and suitable for the rename and function insertion, but I overlooked `semantic_batch` and ran the operations as separate calls.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools: I found the semantic rename and insertion useful, and the rename also cleaned an unused import, though I missed the available batch tool and had to correct one insertion call.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

