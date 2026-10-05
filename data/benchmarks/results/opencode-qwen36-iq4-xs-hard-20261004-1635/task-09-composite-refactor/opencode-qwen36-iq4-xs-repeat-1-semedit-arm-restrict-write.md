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

* **Date**: 2026-10-04 17:08:47 CEST

## Test case: `task-09-composite-refactor`

### Target: `opencode/qwen36-iq4-xs (repeat 1)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 65.65s | 719.92s | N/A | — | — | — |
| **Process Start → First Event** | 12.59s | 53.73s | — | — | — | — |
| **First Event → First Tool Call** | 6.32s | 5.70s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 6 | 239 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 2 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 6 | 239 | N/A | — | — | — |
| **Output Tokens** | 1196 | 22033 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 54563 | 11711445 | N/A | — | — | — |
| **Cached Input Tokens** | 47084 | 11667255 | N/A | — | — | — |
| **Uncached Input Tokens** | 7479 | 44190 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.30:1 | 264.02:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | — | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | — | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | — | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 22 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791124752202500000/config/config.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-09-composite-refactor_1791124821018039000/config/config.go&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791124752202500000/config/config.go&#34;,
  &#34;newString&#34;: &#34;import (\n\t\&#34;fmt\&#34;\n)&#34;,
  &#34;oldString&#34;: &#34;import (\n\t\&#34;archive/tar\&#34;\n\t\&#34;fmt\&#34;\n)&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-09-composite-refactor_1791124821018039000&#34;,
  &#34;pattern&#34;: &#34;Config\\.Port|\\.Port\\b&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791124752202500000/config/config.go&#34;,
  &#34;newString&#34;: &#34;type Config struct {\n\tHost string\n\t// ListenPort is the port the server listens on.\n\tListenPort int\n}&#34;,
  &#34;oldString&#34;: &#34;type Config struct {\n\tHost string\n\tPort int\n}&#34;
}</code></pre></td><td><code>semedit_semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Port&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791124752202500000/config/config.go&#34;,
  &#34;newString&#34;: &#34;func (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)\n}\n\nfunc DefaultConfig() *Config {\n\treturn \u0026Config{\n\t\tHost:       \&#34;localhost\&#34;,\n\t\tListenPort: 8080,\n\t}\n}&#34;,
  &#34;oldString&#34;: &#34;func (c *Config) Addr() string {\n\treturn fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.Port)\n}&#34;
}</code></pre></td><td><code>semedit_semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Config.ListenPort&#34;,
  &#34;to&#34;: &#34;ListenPort&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-09-composite-refactor_1791124752202500000/config/config.go&#34;
}</code></pre></td><td><code>semedit_semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Config&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./config/&#34;
}</code></pre></td><td><code>semedit_semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;kind&#34;: &#34;type&#34;,
  &#34;source&#34;: &#34;type Config struct {\n\tHost string\n\tListenPort int\n}&#34;,
  &#34;symbol&#34;: &#34;Config&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit_semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;in_function&#34;: &#34;Config.Addr&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;source&#34;: &#34;return fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit_semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;in_function&#34;: &#34;(*Config).Addr&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;source&#34;: &#34;return fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit_semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;discriminator&#34;: &#34;c.Port&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;in_function&#34;: &#34;Config.Addr&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;source&#34;: &#34;return fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit_semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;discriminator&#34;: &#34;c.Port&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;in_function&#34;: &#34;(*Config).Addr&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;source&#34;: &#34;return fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit_semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;discriminator&#34;: &#34;fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.Port)&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;in_function&#34;: &#34;(*Config).Addr&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;source&#34;: &#34;return fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit_semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;discriminator&#34;: &#34;return fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.Port)&#34;,
  &#34;file&#34;: &#34;config/config.go&#34;,
  &#34;in_function&#34;: &#34;(*Config).Addr&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;source&#34;: &#34;return fmt.Sprintf(\&#34;%s:%d\&#34;, c.Host, c.ListenPort)&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>31</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>32</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>33</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>34</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>35</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>36</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>37</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>38</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>39</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>40</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>41</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>42</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>43</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>44</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>45</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>46</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>47</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>48</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>49</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>50</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>51</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>52</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>53</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>54</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>55</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>56</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>57</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>58</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>59</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>60</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>61</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>62</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>63</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>64</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>65</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>66</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>67</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>68</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>69</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>70</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>71</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>72</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>73</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>74</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>75</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>76</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>77</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>78</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>79</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>80</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>81</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>82</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>83</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>84</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>85</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>86</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>87</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>88</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>89</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>90</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>91</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>92</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>93</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>94</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>95</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>96</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>97</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>98</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>99</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>100</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>101</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>102</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>103</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>104</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>105</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>106</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>107</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>108</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>109</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>110</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>111</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>112</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>113</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>114</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>115</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>116</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>117</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>118</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>119</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>120</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>121</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>122</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>123</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>124</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>125</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>126</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>127</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>128</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>129</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>130</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>131</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>132</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>133</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>134</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>135</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>136</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>137</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>138</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>139</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>140</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>141</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>142</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>143</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>144</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>145</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>146</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>147</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>148</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>149</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>150</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>151</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>152</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>153</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>154</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>155</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>156</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>157</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>158</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>159</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>160</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>161</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>162</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>163</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>164</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>165</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>166</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>167</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>168</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>169</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>170</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>171</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>172</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>173</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>174</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>175</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>176</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>177</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>178</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>179</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>180</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>181</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>182</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>183</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>184</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>185</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>186</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>187</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>188</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>189</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>190</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>191</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>192</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>193</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>194</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>195</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>196</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>197</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>198</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>199</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>200</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>201</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>202</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>203</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>204</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>205</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>206</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>207</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>208</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>209</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>210</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>211</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>212</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>213</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>214</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>215</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>216</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>217</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>218</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>219</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>220</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>221</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>222</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>223</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>224</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>225</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>226</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>227</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>228</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>229</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>230</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>231</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>232</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>233</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>234</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>235</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>236</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>237</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>238</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
<tr><td>239</td><td>—</td><td><code>semedit_semantic_undo</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;snapshot_id&#34;: &#34;latest&#34;
}</code></pre></td></tr>
</tbody>
</table>

