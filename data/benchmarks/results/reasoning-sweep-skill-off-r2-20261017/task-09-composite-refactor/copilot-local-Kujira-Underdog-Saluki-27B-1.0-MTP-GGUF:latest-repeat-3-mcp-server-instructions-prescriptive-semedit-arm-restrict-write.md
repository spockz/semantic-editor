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

* **Date**: 2026-10-17 21:43:40 CEST

## Test case: `task-09-composite-refactor`

### Target: `copilot/local/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 3)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `copilot_agent_name=Copilot,copilot_agent_version=1.0.88,copilot_model_selection=acp_session_set_config_option,copilot_protocol_version=1,fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,local_model_context_tokens=212992,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=copilot_acp_v1`

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
| **Wall-Clock Latency** | 114.38s | 151.96s | <span class="benchmark-delta-negative">+32.9%</span> | — | — | — |
| **Process Start → First Event** | 17.06s | 43.11s | — | — | — | — |
| **First Event → First Tool Call** | 0.00s | 0.01s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 11 | 9 | <span class="benchmark-delta-positive">-18.2%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 11 | 9 | <span class="benchmark-delta-positive">-18.2%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | — | — | — | — | — | — |
| **Output Tokens** | 1399 | 1448 | <span class="benchmark-delta-negative">+3.5%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 224787 | 304362 | <span class="benchmark-delta-negative">+35.4%</span> | — | — | — |
| **Cached Input Tokens** | 201617 | 264954 | <span class="benchmark-delta-positive">+31.4%</span> | — | — | — |
| **Uncached Input Tokens** | 23170 | 39408 | <span class="benchmark-delta-negative">+70.1%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.70:1 | 6.72:1 | <span class="benchmark-delta-negative">-22.7%</span> | — | — | — |
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
<tr><td>1</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-09-composite-refactor_1792248775616095000/config/config.go&#34;
}</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;\\.Port\\b|Port:&#34;,
  &#34;paths&#34;: &#34;.&#34;,
  &#34;output_mode&#34;: &#34;content&#34;,
  &#34;-n&#34;: true
}</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Config.Addr&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-09-composite-refactor_1792248775616095000/config/config.go&#34;,
  &#34;old_str&#34;: &#34;import (\n\t\&#34;archive/tar\&#34;\n\t\&#34;fmt\&#34;\n)\n\ntype Config struct {\n\tHost string\n\tPort int\n}\n\nfunc (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.Port)\n}&#34;,
  &#34;new_str&#34;: &#34;import (\n\t\&#34;fmt\&#34;\n)\n\ntype Config struct {\n\tHost string\n\tListenPort int\n}\n\nfunc (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)\n}\n\nfunc DefaultConfig() *Config {\n\treturn &amp;Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;
}</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Config.Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./... &amp;&amp; go vet ./config/&#34;,
  &#34;description&#34;: &#34;Build and vet the project&#34;
}</code></pre></td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func DefaultConfig() *Config { return &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080} }&#34;,
  &#34;placement&#34;: &#34;public_end&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go vet ./config/ 2&gt;&amp;1 | head -20&#34;,
  &#34;description&#34;: &#34;Vet the config package&#34;
}</code></pre></td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;config/config.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env GOENV=\&#34;\&#34; GOCACHE=/tmp/gocache go vet ./config/ &amp;&amp; echo VET_OK&#34;,
  &#34;description&#34;: &#34;Vet config package with clean env&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit-explicit_small-explicit_task-09-composite-refactor_1792248518735306000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env GOENV=\&#34;\&#34; GOCACHE=\&#34;$PWD/.gocache\&#34; go vet ./config/ &amp;&amp; echo VET_OK&#34;,
  &#34;description&#34;: &#34;Vet config package with local cache&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./... &amp;&amp; go vet ./...&#34;,
  &#34;description&#34;: &#34;Build and vet the Go module&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;echo \&#34;GOENV=$GOENV\&#34;; echo \&#34;GOFLAGS=$GOFLAGS\&#34;; which go; go version&#34;,
  &#34;description&#34;: &#34;Inspect Go environment variables&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;GOCACHE=/tmp/gocache go build ./... &amp;&amp; GOCACHE=/tmp/gocache go vet ./...&#34;,
  &#34;description&#34;: &#34;Build and vet with a writable Go cache&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env -u GOFLAGS -u GOENV go vet ./config/ &amp;&amp; echo VET_OK&#34;,
  &#34;description&#34;: &#34;Vet config package without GOFLAGS&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;GOCACHE=\&#34;$PWD/.scratch/gocache\&#34; go build ./... &amp;&amp; GOCACHE=\&#34;$PWD/.scratch/gocache\&#34; go vet ./...&#34;,
  &#34;description&#34;: &#34;Build and vet with local Go cache&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env -u GOFLAGS -u GOENV go build ./config/ &amp;&amp; echo BUILD_OK&#34;,
  &#34;description&#34;: &#34;Build config package&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>11</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-09-composite-refactor_1792248775616095000/config/config.go&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `copilot_agent_name=Copilot,copilot_agent_version=1.0.88,copilot_model_selection=acp_session_set_config_option,copilot_protocol_version=1,fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,local_model_context_tokens=212992,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=copilot_acp_v1`

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
| **Wall-Clock Latency** | 114.38s | 259.03s | <span class="benchmark-delta-negative">+126.5%</span> | — | — | — |
| **Process Start → First Event** | 17.06s | 42.37s | — | — | — | — |
| **First Event → First Tool Call** | 0.00s | 0.01s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 11 | 13 | <span class="benchmark-delta-negative">+18.2%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 11 | 13 | <span class="benchmark-delta-negative">+18.2%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | — | — | — | — | — | — |
| **Output Tokens** | 1399 | 2861 | <span class="benchmark-delta-negative">+104.5%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 224787 | 477716 | <span class="benchmark-delta-negative">+112.5%</span> | — | — | — |
| **Cached Input Tokens** | 201617 | 433373 | <span class="benchmark-delta-positive">+114.9%</span> | — | — | — |
| **Uncached Input Tokens** | 23170 | 44343 | <span class="benchmark-delta-negative">+91.4%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.70:1 | 9.77:1 | <span class="benchmark-delta-positive">+12.3%</span> | — | — | — |
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
<tr><td>1</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-09-composite-refactor_1792248775616095000/config/config.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-09-composite-refactor_1792248890454328000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;\\.Port\\b|Port:&#34;,
  &#34;paths&#34;: &#34;.&#34;,
  &#34;output_mode&#34;: &#34;content&#34;,
  &#34;-n&#34;: true
}</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-09-composite-refactor_1792248775616095000/config/config.go&#34;,
  &#34;old_str&#34;: &#34;import (\n\t\&#34;archive/tar\&#34;\n\t\&#34;fmt\&#34;\n)\n\ntype Config struct {\n\tHost string\n\tPort int\n}\n\nfunc (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.Port)\n}&#34;,
  &#34;new_str&#34;: &#34;import (\n\t\&#34;fmt\&#34;\n)\n\ntype Config struct {\n\tHost string\n\tListenPort int\n}\n\nfunc (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)\n}\n\nfunc DefaultConfig() *Config {\n\treturn &amp;Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;\\.Port\\b&#34;,
  &#34;paths&#34;: &#34;.&#34;,
  &#34;output_mode&#34;: &#34;content&#34;,
  &#34;-n&#34;: true
}</code></pre></td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./... &amp;&amp; go vet ./config/&#34;,
  &#34;description&#34;: &#34;Build and vet the project&#34;
}</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): MCP server &#39;semedit&#39;: Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Port in config/config.go: symbol not found&#34;,&#34;suggestions_count&#34;:1,&#34;tool&#34;:&#34;semantic_rename&#34;}],&#34;workspace_root&#34;:&#34;/Volumes/semedit-bench-ram/run_copilot_semedit_smal<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;symbol&#34;: &#34;Port&#34;,
        &#34;to&#34;: &#34;ListenPort&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080}\n}&#34;,
        &#34;placement&#34;: &#34;file_end&#34;
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
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go vet ./config/ 2&gt;&amp;1 | head -20&#34;,
  &#34;description&#34;: &#34;Vet the config package&#34;
}</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): MCP server &#39;semedit&#39;: Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;ok&#34;,&#34;result&#34;:{&#34;status&#34;:&#34;renamed&#34;,&#34;application_state&#34;:&#34;applied_working&#34;,&#34;continuation_unavailable&#34;:&#34;rename is part of an in-progress workspace batch&#34;,&#34;active_count&#34;:0,&#34;diagnostics&#34;:{&#34;introduced_count&#34;:0,&#34;net_delta&#34;:0,<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;symbol&#34;: &#34;Config.Port&#34;,
        &#34;to&#34;: &#34;ListenPort&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080}\n}&#34;,
        &#34;placement&#34;: &#34;file_end&#34;
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
<tr><td>6</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env GOENV=\&#34;\&#34; GOCACHE=/tmp/gocache go vet ./config/ &amp;&amp; echo VET_OK&#34;,
  &#34;description&#34;: &#34;Vet config package with clean env&#34;
}</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn &amp;Config{Host: \&#34;localhost\&#34;, ListenPort: 8080}\n}&#34;,
        &#34;placement&#34;: &#34;file_end&#34;
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
<tr><td>7</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env GOENV=\&#34;\&#34; GOCACHE=\&#34;$PWD/.gocache\&#34; go vet ./config/ &amp;&amp; echo VET_OK&#34;,
  &#34;description&#34;: &#34;Vet config package with local cache&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-09-composite-refactor_1792248890454328000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;echo \&#34;GOENV=$GOENV\&#34;; echo \&#34;GOFLAGS=$GOFLAGS\&#34;; which go; go version&#34;,
  &#34;description&#34;: &#34;Inspect Go environment variables&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./... &amp;&amp; go vet ./...&#34;,
  &#34;description&#34;: &#34;Build and vet the Go module&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env -u GOFLAGS -u GOENV go vet ./config/ &amp;&amp; echo VET_OK&#34;,
  &#34;description&#34;: &#34;Vet config package without GOFLAGS&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;GOCACHE=/tmp/gocache go build ./... &amp;&amp; GOCACHE=/tmp/gocache go vet ./...&#34;,
  &#34;description&#34;: &#34;Build and vet with local Go cache&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env -u GOFLAGS -u GOENV go build ./config/ &amp;&amp; echo BUILD_OK&#34;,
  &#34;description&#34;: &#34;Build config package&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;GOCACHE=.gocache go build ./... &amp;&amp; GOCACHE=.gocache go vet ./...&#34;,
  &#34;description&#34;: &#34;Build and vet with workspace-local Go cache&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-09-composite-refactor_1792248775616095000/config/config.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;mkdir -p /Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-09-composite-refactor_1792248890454328000/.gocache &amp;&amp; GOCACHE=/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-09-composite-refactor_1792248890454328000/.gocache go build ./... &amp;&amp; GOCACHE=/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-09-composite-refactor_1792248890454328000/.gocache go vet ./...&#34;,
  &#34;description&#34;: &#34;Build and vet with workspace-local Go cache&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build -o /tmp/out ./config 2&gt;&amp;1 | head -20&#34;,
  &#34;description&#34;: &#34;Build only the config package&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;rm -rf /Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-09-composite-refactor_1792248890454328000/.gocache&#34;,
  &#34;description&#34;: &#34;Remove temporary Go cache directory&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I&#39;d definitely ask my owner to add semedit&#39;s tools — during this run, the structured gopls verification gave me real confidence that my rename, insert, and import cleanup were correct even when the shell build failed on</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
<p><strong>Capture error:</strong> wait for Copilot session/prompt: context deadline exceeded; Copilot stderr: Warning: COPILOT_CLI_DIST_DIR (/Users/alessandro/Library/Caches/copilot/pkg/darwin-arm64/1.0.88) does not contain a readable index.js; ignoring</p>
</details>

