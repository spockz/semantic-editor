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

* **Date**: 2026-10-05 04:35:31 CEST

## Test case: `task-09-composite-refactor`

### Target: `opencode/qwen36-iq4-xs (repeat 6)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

* **Run Provenance**: `gopls_version=0.23.0,harness_sha256=9b6e0b3f3f786895237c35e83b80769d5d1755c3660696217da5931f1e4e4678,semedit_sha256=8119878c0486579e57f89c1d11b8a7f152c22d5eea57959f4417e232d8224434`

* **Fixture**: [testdata/bench/task_09_composite_refactor.txtar](https://github.com/spockz/semantic-editor/blob/83d88a5482d8614eee37246b1ddc53ebb78d4435/testdata/bench/task_09_composite_refactor.txtar)

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
| **Wall-Clock Latency** | 60.86s | 90.29s | <span class="benchmark-delta-negative">+48.4%</span> | — | — | — |
| **Process Start → First Event** | 11.65s | 50.60s | — | — | — | — |
| **First Event → First Tool Call** | 5.03s | 5.36s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 9 | 4 | <span class="benchmark-delta-positive">-55.6%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 9 | 4 | <span class="benchmark-delta-positive">-55.6%</span> | — | — | — |
| **Output Tokens** | 1623 | 997 | <span class="benchmark-delta-positive">-38.6%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 72969 | 166533 | <span class="benchmark-delta-negative">+128.2%</span> | — | — | — |
| **Cached Input Tokens** | 65180 | 132708 | <span class="benchmark-delta-positive">+103.6%</span> | — | — | — |
| **Uncached Input Tokens** | 7789 | 33825 | <span class="benchmark-delta-negative">+334.3%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.37:1 | 3.92:1 | <span class="benchmark-delta-negative">-53.1%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 18 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791154699818869000/config/config.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-09-composite-refactor_1791154761036271000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>write</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;content&#34;: &#34;package config\n\nimport (\n\t\&#34;fmt\&#34;\n)\n\ntype Config struct {\n\tHost string\n\tPort int\n}\n\nfunc DefaultConfig() *Config {\n\treturn \u0026Config{\n\t\tHost:   \&#34;localhost\&#34;,\n\t\tPort:   8080,\n\t}\n}\n\nfunc (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.Port)\n}\n&#34;,
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791154699818869000/config/config.go&#34;
}</code></pre></td><td><code>semedit_semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Config.Port&#34;,
        &#34;to&#34;: &#34;ListenPort&#34;
      },
      &#34;tool&#34;: &#34;semantic_rename&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;config/config.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;source&#34;: &#34;func DefaultConfig() *Config {\n\treturn \u0026Config{Host: \&#34;localhost\&#34;, ListenPort: 8080}\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;remove&#34;: [
          &#34;archive/tar&#34;
        ],
        &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-09-composite-refactor_1791154761036271000&#34;
      },
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;
    }
  ],
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-09-composite-refactor_1791154761036271000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791154699818869000/config/config.go&#34;,
  &#34;newString&#34;: &#34;\tListenPort int&#34;,
  &#34;oldString&#34;: &#34;\tPort int&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-09-composite-refactor_1791154761036271000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791154699818869000/config/config.go&#34;,
  &#34;newString&#34;: &#34;\t\tListenPort: 8080,&#34;,
  &#34;oldString&#34;: &#34;\t\tPort:   8080,&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go test ./config/...&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-09-composite-refactor_1791154761036271000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791154699818869000/config/config.go&#34;,
  &#34;newString&#34;: &#34;\tc.ListenPort&#34;,
  &#34;oldString&#34;: &#34;\tc.Port&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791154699818869000/config/config.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>7</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791154699818869000/config/config.go&#34;,
  &#34;newString&#34;: &#34;\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)&#34;,
  &#34;oldString&#34;: &#34;\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, \tc.ListenPort)&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>8</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791154699818869000 \u0026\u0026 go build ./config/&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>9</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791154699818869000/config/config.go&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semitedit&#39;s batch operations are a genuine joy—I could rename a struct field, add a new function, and auto-clean unused imports all in one atomic, verified step instead of juggling multiple fragile text edits.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

