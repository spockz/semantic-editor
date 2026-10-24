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

* **Date**: 2026-10-17 21:43:41 CEST

## Test case: `task-09-composite-refactor`

### Target: `codex/gpt-6-luna/medium (repeat 3)`

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
| **Wall-Clock Latency** | 10.62s | 20.43s | <span class="benchmark-delta-negative">+92.3%</span> | — | — | — |
| **Process Start → First Event** | 0.03s | 0.02s | — | — | — | — |
| **First Event → First Tool Call** | 3.40s | 5.09s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 3 | 7 | <span class="benchmark-delta-negative">+133.3%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 3 | 7 | <span class="benchmark-delta-negative">+133.3%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.002544 | $0.005302 | +108.5% | — | — | — |
| **Output Tokens** | 475 | 1197 | <span class="benchmark-delta-negative">+152.0%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 36 | 315 | <span class="benchmark-delta-negative">+775.0%</span> | — | — | — |
| **Total Input Tokens** | 68564 | 237579 | <span class="benchmark-delta-negative">+246.5%</span> | — | — | — |
| **Cached Input Tokens** | 50560 | 211712 | <span class="benchmark-delta-positive">+318.7%</span> | — | — | — |
| **Uncached Input Tokens** | 18004 | 25867 | <span class="benchmark-delta-negative">+43.7%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 2.81:1 | 8.18:1 | <span class="benchmark-delta-positive">+191.4%</span> | — | — | — |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; config/config.go &amp;&amp; \
rg -n &#39;\\.Port\\b|Port:&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; | head -40&#34;</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; | head -40&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: package config import ( &#34;fmt&#34; ) type Config struct { Host string ListenPort int } func DefaultConfig() *Config { return &amp;Config{ Host: &#34;localhost&#34;, ListenPort: 8080, } } func (c *Config) Addr() string { return fmt.Sprintf(&#34;%s:%d&#34;, c.Host, c.ListenPort) } fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; config/config.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-09-composite-refactor_1792242909090507000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-09-composite-refactor_1792242909090507000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;symbol&#34;: &#34;Config.Addr&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-09-composite-refactor_1792242909090507000&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat config/config.go&#39;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-09-composite-refactor_1792242909090507000&#34;,
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
| **Wall-Clock Latency** | 10.62s | 21.60s | <span class="benchmark-delta-negative">+103.3%</span> | — | — | — |
| **Process Start → First Event** | 0.03s | 0.02s | — | — | — | — |
| **First Event → First Tool Call** | 3.40s | 4.00s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.002544 | $0.004711 | +85.2% | — | — | — |
| **Output Tokens** | 475 | 802 | <span class="benchmark-delta-negative">+68.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 36 | 52 | <span class="benchmark-delta-negative">+44.4%</span> | — | — | — |
| **Total Input Tokens** | 68564 | 240786 | <span class="benchmark-delta-negative">+251.2%</span> | — | — | — |
| **Cached Input Tokens** | 50560 | 219648 | <span class="benchmark-delta-positive">+334.4%</span> | — | — | — |
| **Uncached Input Tokens** | 18004 | 21138 | <span class="benchmark-delta-negative">+17.4%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 2.81:1 | 10.39:1 | <span class="benchmark-delta-positive">+270.0%</span> | — | — | — |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; config/config.go &amp;&amp; \
rg -n &#39;\\.Port\\b|Port:&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;config/config.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # Benchmark fixture instruction override This directory is a synthetic benchmark fixture and is the complete benchmark workspace. Follow the benchmark user prompt. You may inspect and use any file in this directory, including documentation. Do not inspect, read, or use paths outside this directory. Run only checks that the user prompt or files in this directory require. Do not modify this AGENTS.override.md file. package config import ( &#34;archive/tar&#34; &#34;fmt&#34; ) type Config struct { Host string Port int } func <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md &amp;&amp; \
cat config/config.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: package config import ( &#34;fmt&#34; ) type Config struct { Host string ListenPort int } func DefaultConfig() *Config { return &amp;Config{ Host: &#34;localhost&#34;, ListenPort: 8080, } } func (c *Config) Addr() string { return fmt.Sprintf(&#34;%s:%d&#34;, c.Host, c.ListenPort) } fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; config/config.go &amp;&amp; \
git status --short&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242944081595000&#34;,
  &#34;path&#34;: &#34;config&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td>—</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;config/config.go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242944081595000&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242944081595000&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;file&#34;: &#34;config/config.go&#34;,
      &#34;symbol&#34;: &#34;Config.Port&#34;,
      &#34;to&#34;: &#34;ListenPort&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;file&#34;: &#34;config/config.go&#34;,
      &#34;kind&#34;: &#34;function&#34;,
      &#34;placement&#34;: &#34;after_symbol&#34;,
      &#34;target_symbol&#34;: &#34;Config&#34;,
      &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080}\n}&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-09-composite-refactor_1792242944081595000&#34;
    }
  ]
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit made the field rename, constructor insertion, and import cleanup feel precise and low risk, so I’d ask my owner to add its tools.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

