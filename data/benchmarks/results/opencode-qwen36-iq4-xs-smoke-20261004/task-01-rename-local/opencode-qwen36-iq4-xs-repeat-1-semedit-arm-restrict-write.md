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

* **Date**: 2026-10-04 15:34:45 CEST

## Test case: `task-01-rename-local`

### Target: `opencode/qwen36-iq4-xs (repeat 1)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_01_rename_local.txtar` (`sha256:224b2797e1b13aee9c9d8d34 (uncommitted)`)

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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 37.31s | 128.80s | N/A | — | — | — |
| **Process Start → First Event** | 11.73s | 50.96s | — | — | — | — |
| **First Event → First Tool Call** | 2.77s | 2.43s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 7 | 15 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 4 | 3 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 15 | N/A | — | — | — |
| **Output Tokens** | 845 | 2451 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 37962 | 502103 | N/A | — | — | — |
| **Cached Input Tokens** | 30426 | 468683 | N/A | — | — | — |
| **Uncached Input Tokens** | 7536 | 33420 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.04:1 | 14.02:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File api/server.go modified (12 lines -> 12 lines)
* **MCP Edit**: File api/server.go modified (12 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;func \\(s \\*Server\\) oldName&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;oldName&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;func \\(s \\*Server\\) \\.oldName&#34;
}</code></pre></td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/*.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;\\.oldName\\(&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-01-rename-local_1791119132308236000/api/server.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-01-rename-local_1791119094680598000/api/server.go&#34;
}</code></pre></td><td><code>semedit_semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;symbol&#34;: &#34;Server.oldName&#34;,
  &#34;to&#34;: &#34;newName&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-01-rename-local_1791119094680598000/api/server.go&#34;,
  &#34;newString&#34;: &#34;func (s *Server) newName() string {&#34;,
  &#34;oldString&#34;: &#34;func (s *Server) oldName() string {&#34;
}</code></pre></td><td><code>semedit_semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;symbol&#34;: &#34;oldName&#34;,
  &#34;to&#34;: &#34;newName&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-01-rename-local_1791119094680598000/api/server.go&#34;,
  &#34;newString&#34;: &#34;\treturn s.newName()&#34;,
  &#34;oldString&#34;: &#34;\treturn s.oldName()&#34;
}</code></pre></td><td><code>semedit_semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;symbol&#34;: &#34;Server.oldName&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small-default_task-01-rename-local_1791119094680598000/api/server.go&#34;
}</code></pre></td><td><code>semedit_semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Server.oldName&#34;,
        &#34;to&#34;: &#34;newName&#34;
      },
      &#34;tool&#34;: &#34;semantic_rename&#34;
    }
  ],
  &#34;trust_workspace&#34;: true,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-01-rename-local_1791119132308236000&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit_semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;kind&#34;: &#34;method&#34;,
  &#34;source&#34;: &#34;func (s *Server) newName() string {\n\treturn \&#34;ok\&#34;\n}&#34;,
  &#34;symbol&#34;: &#34;Server.oldName&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit_semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func (s *Server) newName() string {\n\treturn \&#34;ok\&#34;\n}&#34;,
  &#34;symbol&#34;: &#34;Server.oldName&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit_semantic_remove_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;api/server.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;safe_delete&#34;: true,
  &#34;symbol&#34;: &#34;Server.oldName&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit_semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): MCP error -32602: Structured content does not match the tool&#39;s output schema: data must have required property &#39;result&#39;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;oldName&#34;,
  &#34;to&#34;: &#34;newName&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-01-rename-local_1791119132308236000/api/server.go&#34;,
  &#34;newString&#34;: &#34;func (s *Server) newName() string {&#34;,
  &#34;oldString&#34;: &#34;func (s *Server) oldName() string {&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-01-rename-local_1791119132308236000/api/server.go&#34;,
  &#34;newString&#34;: &#34;\treturn s.newName()&#34;,
  &#34;oldString&#34;: &#34;\treturn s.oldName()&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-01-rename-local_1791119132308236000/api/server.go&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_opencode_semedit_small-default_task-01-rename-local_1791119132308236000 \u0026\u0026 go test ./...&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did attempt to call semantic_* tools (semantic_rename, semantic_replace_construct, semantic_remove_construct, semantic_batch), but all of them failed with MCP error -32602 indicating the server&#39;s response schema was invalid (missing required &#39;result&#39; property). The semedit MCP server was not functioning correctly, which forced me to fall back to direct file edits instead.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I&#39;d love to recommend semedit&#39;s semantic rename and refactor tools—they&#39;re exactly the kind of type-safe, reference-aware operations I reach for first—but this experience left me frustrated because every single call crashed with a server schema error, so I had no choice but to fall back to basic text edits and never actually got to experience their value.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

