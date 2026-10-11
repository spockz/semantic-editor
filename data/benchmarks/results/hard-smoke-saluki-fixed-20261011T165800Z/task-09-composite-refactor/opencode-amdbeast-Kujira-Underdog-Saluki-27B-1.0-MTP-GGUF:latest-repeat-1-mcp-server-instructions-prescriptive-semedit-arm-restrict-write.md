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

* **Date**: 2026-10-11 19:22:27 CEST

## Test case: `task-09-composite-refactor`

### Target: `opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

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
| **Wall-Clock Latency** | 28.06s | 88.12s | <span class="benchmark-delta-negative">+214.1%</span> | — | — | — |
| **Process Start → First Event** | 8.65s | 26.44s | — | — | — | — |
| **First Event → First Tool Call** | 2.07s | 5.04s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 8 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 8 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Output Tokens** | 590 | 2165 | <span class="benchmark-delta-negative">+266.9%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 36200 | 180008 | <span class="benchmark-delta-negative">+397.3%</span> | — | — | — |
| **Cached Input Tokens** | 29121 | 152715 | <span class="benchmark-delta-positive">+424.4%</span> | — | — | — |
| **Uncached Input Tokens** | 7079 | 27293 | <span class="benchmark-delta-negative">+285.5%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.11:1 | 5.60:1 | <span class="benchmark-delta-positive">+36.0%</span> | — | — | — |
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
<tr><td>1</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-09-composite-refactor_1791737789729246000/config/config.go&#34;
}</code></pre></td><td><code>semedit_semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Config.Port&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-09-composite-refactor_1791737789729246000&#34;,
  &#34;pattern&#34;: &#34;\\.Port&#34;
}</code></pre></td><td><code>semedit_semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Config.Addr&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-09-composite-refactor_1791737789729246000/config/config.go&#34;,
  &#34;newString&#34;: &#34;import (\n\t\&#34;fmt\&#34;\n)\n\ntype Config struct {\n\tHost string\n\tListenPort int\n}\n\nfunc (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)\n}\n\nfunc DefaultConfig() *Config {\n\treturn \u0026Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;,
  &#34;oldString&#34;: &#34;import (\n\t\&#34;archive/tar\&#34;\n\t\&#34;fmt\&#34;\n)\n\ntype Config struct {\n\tHost string\n\tPort int\n}\n\nfunc (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.Port)\n}&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_task-09-composite-refactor_1791738036259022000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./config/&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-09-composite-refactor_1791737789729246000&#34;
}</code></pre></td><td><code>semedit_semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 1 params: unknown parameter \&#34;kind\&#34; for \&#34;semantic_insert_function\&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, file, placement, source, target, target_symbol] Required parameters: [file, source]&#34;,&#34;tool&#34;:&#34;semantic_insert_function&#34;}],&#34;workspace_root&#34;:&#34;/U<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;symbol&#34;: &#34;Config.Port&#34;,
        &#34;to&#34;: &#34;ListenPort&#34;
      },
      &#34;tool&#34;: &#34;semantic_rename&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;placement&#34;: &#34;public_end&#34;,
        &#34;source&#34;: &#34;func DefaultConfig() *Config { return \u0026Config{Host: \&#34;localhost\&#34;, ListenPort: 8080} }&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;
      },
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_task-09-composite-refactor_1791738036259022000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit_semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;symbol&#34;: &#34;Config.Port&#34;,
        &#34;to&#34;: &#34;ListenPort&#34;
      },
      &#34;tool&#34;: &#34;semantic_rename&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;placement&#34;: &#34;public_end&#34;,
        &#34;source&#34;: &#34;func DefaultConfig() *Config { return \u0026Config{Host: \&#34;localhost\&#34;, ListenPort: 8080} }&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;
      },
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_task-09-composite-refactor_1791738036259022000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./... \u0026\u0026 go test ./... 2\u003e\u00261 | tail -20&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_task-09-composite-refactor_1791738036259022000&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

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
| **Wall-Clock Latency** | 28.06s | 216.06s | <span class="benchmark-delta-negative">+670.1%</span> | — | — | — |
| **Process Start → First Event** | 8.65s | 24.31s | — | — | — | — |
| **First Event → First Tool Call** | 2.07s | 1.23s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **Output Tokens** | 590 | 8317 | <span class="benchmark-delta-negative">+1309.7%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 36200 | 47356 | <span class="benchmark-delta-negative">+30.8%</span> | — | — | — |
| **Cached Input Tokens** | 29121 | 23152 | <span class="benchmark-delta-negative">-20.5%</span> | — | — | — |
| **Uncached Input Tokens** | 7079 | 24204 | <span class="benchmark-delta-negative">+241.9%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.11:1 | 0.96:1 | <span class="benchmark-delta-negative">-76.7%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | — | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | — | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | — | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-09-composite-refactor_1791737789729246000/config/config.go&#34;
}</code></pre></td><td><code>semedit_semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-09-composite-refactor_1791737789729246000&#34;,
  &#34;pattern&#34;: &#34;\\.Port&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-implicit_task-09-composite-refactor_1791737818212883000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-09-composite-refactor_1791737789729246000/config/config.go&#34;,
  &#34;newString&#34;: &#34;import (\n\t\&#34;fmt\&#34;\n)\n\ntype Config struct {\n\tHost string\n\tListenPort int\n}\n\nfunc (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)\n}\n\nfunc DefaultConfig() *Config {\n\treturn \u0026Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;,
  &#34;oldString&#34;: &#34;import (\n\t\&#34;archive/tar\&#34;\n\t\&#34;fmt\&#34;\n)\n\ntype Config struct {\n\tHost string\n\tPort int\n}\n\nfunc (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.Port)\n}&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./config/&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-09-composite-refactor_1791737789729246000&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

