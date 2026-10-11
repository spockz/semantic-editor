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

* **Date**: 2026-10-11 12:30:15 CEST

## Test case: `task-01-rename-local`

### Target: `opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/task_01_rename_local.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_01_rename_local.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct{}

func (s *Server) oldName() string {
	return "ok"
}

func Run() string {
	s := &Server{}
	return s.oldName()
}
```
</details>

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 31.41s | 36.77s | <span class="benchmark-delta-negative">+17.1%</span> | — | — | — |
| **Process Start → First Event** | 9.58s | 24.06s | — | — | — | — |
| **First Event → First Tool Call** | 1.20s | 1.90s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 7 | 3 | <span class="benchmark-delta-positive">-57.1%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 4 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 3 | <span class="benchmark-delta-positive">-57.1%</span> | — | — | — |
| **Output Tokens** | 634 | 276 | <span class="benchmark-delta-positive">-56.5%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 44670 | 94216 | <span class="benchmark-delta-negative">+110.9%</span> | — | — | — |
| **Cached Input Tokens** | 37326 | 70386 | <span class="benchmark-delta-positive">+88.6%</span> | — | — | — |
| **Uncached Input Tokens** | 7344 | 23830 | <span class="benchmark-delta-negative">+224.5%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.08:1 | 2.95:1 | <span class="benchmark-delta-negative">-41.9%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File api/server.go modified (12 lines -> 12 lines)
* **MCP Edit**: File api/server.go modified (12 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;oldName&#34;
}</code></pre></td><td><code>semedit_semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Server.oldName&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/*.go&#34;
}</code></pre></td><td><code>semedit_semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Server.oldName&#34;,
  &#34;to&#34;: &#34;newName&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000/api/server.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;grep -rn \&#34;oldName\&#34; . 2\u003e/dev/null; go build ./... 2\u003e\u00261 | head -20&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000/api/server_test.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000/api/server.go&#34;,
  &#34;newString&#34;: &#34;func (s *Server) newName() string {&#34;,
  &#34;oldString&#34;: &#34;func (s *Server) oldName() string {&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000/api/server.go&#34;,
  &#34;newString&#34;: &#34;\treturn s.newName()&#34;,
  &#34;oldString&#34;: &#34;\treturn s.oldName()&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>7</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000 \u0026\u0026 go build ./... \u0026\u0026 go test ./...&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/task_01_rename_local.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_01_rename_local.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct{}

func (s *Server) oldName() string {
	return "ok"
}

func Run() string {
	s := &Server{}
	return s.oldName()
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 31.41s | 45.78s | <span class="benchmark-delta-negative">+45.8%</span> | — | — | — |
| **Process Start → First Event** | 9.58s | 23.81s | — | — | — | — |
| **First Event → First Tool Call** | 1.20s | 1.15s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 7 | 5 | <span class="benchmark-delta-positive">-28.6%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 4 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 5 | <span class="benchmark-delta-positive">-28.6%</span> | — | — | — |
| **Output Tokens** | 634 | 516 | <span class="benchmark-delta-positive">-18.6%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 44670 | 120186 | <span class="benchmark-delta-negative">+169.1%</span> | — | — | — |
| **Cached Input Tokens** | 37326 | 95775 | <span class="benchmark-delta-positive">+156.6%</span> | — | — | — |
| **Uncached Input Tokens** | 7344 | 24411 | <span class="benchmark-delta-negative">+232.4%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.08:1 | 3.92:1 | <span class="benchmark-delta-negative">-22.8%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File api/server.go modified (12 lines -> 12 lines)
* **MCP Edit**: File api/server.go modified (12 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;oldName&#34;
}</code></pre></td><td><code>semedit_semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/*.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-implicit_task-01-rename-local_1791714515532303000/api/server.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000/api/server.go&#34;
}</code></pre></td><td><code>semedit_semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Server.oldName&#34;,
  &#34;to&#34;: &#34;newName&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000/api/server_test.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-implicit_task-01-rename-local_1791714515532303000/api/server.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000/api/server.go&#34;,
  &#34;newString&#34;: &#34;func (s *Server) newName() string {&#34;,
  &#34;oldString&#34;: &#34;func (s *Server) oldName() string {&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-implicit_task-01-rename-local_1791714515532303000 \u0026\u0026 go build ./... \u0026\u0026 go vet ./... \u0026\u0026 go test ./... 2\u003e\u00261 || true&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000/api/server.go&#34;,
  &#34;newString&#34;: &#34;\treturn s.newName()&#34;,
  &#34;oldString&#34;: &#34;\treturn s.oldName()&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>7</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_task-01-rename-local_1791714483942844000 \u0026\u0026 go build ./... \u0026\u0026 go test ./...&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I was genuinely impressed by how the semantic rename tool handled the declaration and call-site in a single, clean atomic operation, and the outline gave me a perfect structural map before I touched anything—I&#39;d absolutely ask my owner to add semedit&#39;s tools to my toolkit.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

