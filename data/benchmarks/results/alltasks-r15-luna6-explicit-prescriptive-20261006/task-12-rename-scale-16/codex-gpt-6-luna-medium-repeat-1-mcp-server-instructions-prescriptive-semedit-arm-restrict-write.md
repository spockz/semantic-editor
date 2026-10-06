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

* **Date**: 2026-10-06 08:51:07 CEST

## Test case: `task-12-rename-scale-16`

### Target: `codex/gpt-6-luna/medium (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_12_rename_scale_16.txtar` (`sha256:34bd10c3dadf2e5d71e76049 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Apply every Processor method rename in rename-plan.md and update all call sites. Preserve behavior and run go test ./... before finishing. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Read all rows of rename-plan.md. In parallel, lookup each old Processor method in pipeline/workflow.go. Then execute semantic_rename for all sixteen entries, serially in plan order or in an ordered semantic_batch, using the returned symbols: LoadInput→ReadRequest, CheckInput→ValidateRequest, NormalizeValue→CanonicalizeValue, MapFields→ProjectFields, RouteValue→DispatchValue, StampEvent→AnnotateEvent, CountItems→TallyItems, MergeParts→CombineParts, FilterNoise→PruneNoise, SortRecords→OrderRecords, EncodePayload→MarshalPayload, DecodeReply→ParseReply, OpenChannel→ConnectChannel, CloseChannel→ReleaseChannel, FlushBuffer→DrainBuffer, SaveResult→PersistResult. Update all call sites and preserve behavior. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package pipeline provides independent processing stages for the rename scale benchmark.
package pipeline

type Processor struct{}

func (Processor) LoadInput(value int) int { return value + 1 }
func (Processor) CheckInput(value int) int { return value + 2 }
func (Processor) NormalizeValue(value int) int { return value + 3 }
func (Processor) MapFields(value int) int { return value + 4 }
func (Processor) RouteValue(value int) int { return value + 5 }
func (Processor) StampEvent(value int) int { return value + 6 }
func (Processor) CountItems(value int) int { return value + 7 }
func (Processor) MergeParts(value int) int { return value + 8 }
func (Processor) FilterNoise(value int) int { return value + 9 }
func (Processor) SortRecords(value int) int { return value + 10 }
func (Processor) EncodePayload(value int) int { return value + 11 }
func (Processor) DecodeReply(value int) int { return value + 12 }
func (Processor) OpenChannel(value int) int { return value + 13 }
func (Processor) CloseChannel(value int) int { return value + 14 }
func (Processor) FlushBuffer(value int) int { return value + 15 }
func (Processor) SaveResult(value int) int { return value + 16 }

func Run(processor Processor, value int) int {
	value = processor.LoadInput(value)
	value = processor.CheckInput(value)
	value = processor.NormalizeValue(value)
	value = processor.MapFields(value)
	value = processor.RouteValue(value)
	value = processor.StampEvent(value)
	value = processor.CountItems(value)
	value = processor.MergeParts(value)
	value = processor.FilterNoise(value)
	value = processor.SortRecords(value)
	value = processor.EncodePayload(value)
	value = processor.DecodeReply(value)
	value = processor.OpenChannel(value)
	value = processor.CloseChannel(value)
	value = processor.FlushBuffer(value)
	value = processor.SaveResult(value)
	return value
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 35.11s | 68.39s | <span class="benchmark-delta-negative">+94.8%</span> | — | — | — |
| **Process Start → First Event** | 0.28s | 0.14s | — | — | — | — |
| **First Event → First Tool Call** | 4.10s | 7.35s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 18 | <span class="benchmark-delta-negative">+350.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 18 | <span class="benchmark-delta-negative">+350.0%</span> | — | — | — |
| **Output Tokens** | 1236 | 746 | <span class="benchmark-delta-positive">-39.6%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 56 | 227 | <span class="benchmark-delta-negative">+305.4%</span> | — | — | — |
| **Total Input Tokens** | 102208 | 149389 | <span class="benchmark-delta-negative">+46.2%</span> | — | — | — |
| **Cached Input Tokens** | 93952 | 117248 | <span class="benchmark-delta-positive">+24.8%</span> | — | — | — |
| **Uncached Input Tokens** | 8256 | 32141 | <span class="benchmark-delta-negative">+289.3%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 11.38:1 | 3.65:1 | <span class="benchmark-delta-negative">-67.9%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File pipeline/workflow.go modified (41 lines -> 41 lines)
* **MCP Edit**: File pipeline/workflow.go modified (41 lines -> 41 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*vendor*&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat rename-plan.md&#39;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat rename-plan.md &amp;&amp; \
cat pipeline/workflow.go &amp;&amp; \
cat pipeline/workflow_test.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;LoadInput&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;CheckInput&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;NormalizeValue&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;MapFields&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;RouteValue&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;StampEvent&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;CountItems&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;MergeParts&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;FilterNoise&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;SortRecords&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;EncodePayload&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;DecodeReply&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;OpenChannel&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;CloseChannel&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;FlushBuffer&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;SaveResult&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.LoadInput&#34;,
        &#34;to&#34;: &#34;ReadRequest&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.CheckInput&#34;,
        &#34;to&#34;: &#34;ValidateRequest&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.NormalizeValue&#34;,
        &#34;to&#34;: &#34;CanonicalizeValue&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.MapFields&#34;,
        &#34;to&#34;: &#34;ProjectFields&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.RouteValue&#34;,
        &#34;to&#34;: &#34;DispatchValue&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.StampEvent&#34;,
        &#34;to&#34;: &#34;AnnotateEvent&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.CountItems&#34;,
        &#34;to&#34;: &#34;TallyItems&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.MergeParts&#34;,
        &#34;to&#34;: &#34;CombineParts&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.FilterNoise&#34;,
        &#34;to&#34;: &#34;PruneNoise&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.SortRecords&#34;,
        &#34;to&#34;: &#34;OrderRecords&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.EncodePayload&#34;,
        &#34;to&#34;: &#34;MarshalPayload&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.DecodeReply&#34;,
        &#34;to&#34;: &#34;ParseReply&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.OpenChannel&#34;,
        &#34;to&#34;: &#34;ConnectChannel&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.CloseChannel&#34;,
        &#34;to&#34;: &#34;ReleaseChannel&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.FlushBuffer&#34;,
        &#34;to&#34;: &#34;DrainBuffer&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
        &#34;symbol&#34;: &#34;Processor.SaveResult&#34;,
        &#34;to&#34;: &#34;PersistResult&#34;
      }
    }
  ],
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-12-rename-scale-16_1791267138860697000&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semantic lookups found all 16 methods, and one ordered rename batch updated their call sites with zero diagnostics, so I’d ask my owner to add semedit’s tools; that was a reassuringly direct way to make a broad rename safely.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_12_rename_scale_16.txtar` (`sha256:34bd10c3dadf2e5d71e76049 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Apply every Processor method rename in rename-plan.md and update all call sites. Preserve behavior and run go test ./... before finishing. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Apply every Processor method rename in rename-plan.md and update all call sites. Preserve behavior and run go test ./... before finishing. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package pipeline provides independent processing stages for the rename scale benchmark.
package pipeline

type Processor struct{}

func (Processor) LoadInput(value int) int { return value + 1 }
func (Processor) CheckInput(value int) int { return value + 2 }
func (Processor) NormalizeValue(value int) int { return value + 3 }
func (Processor) MapFields(value int) int { return value + 4 }
func (Processor) RouteValue(value int) int { return value + 5 }
func (Processor) StampEvent(value int) int { return value + 6 }
func (Processor) CountItems(value int) int { return value + 7 }
func (Processor) MergeParts(value int) int { return value + 8 }
func (Processor) FilterNoise(value int) int { return value + 9 }
func (Processor) SortRecords(value int) int { return value + 10 }
func (Processor) EncodePayload(value int) int { return value + 11 }
func (Processor) DecodeReply(value int) int { return value + 12 }
func (Processor) OpenChannel(value int) int { return value + 13 }
func (Processor) CloseChannel(value int) int { return value + 14 }
func (Processor) FlushBuffer(value int) int { return value + 15 }
func (Processor) SaveResult(value int) int { return value + 16 }

func Run(processor Processor, value int) int {
	value = processor.LoadInput(value)
	value = processor.CheckInput(value)
	value = processor.NormalizeValue(value)
	value = processor.MapFields(value)
	value = processor.RouteValue(value)
	value = processor.StampEvent(value)
	value = processor.CountItems(value)
	value = processor.MergeParts(value)
	value = processor.FilterNoise(value)
	value = processor.SortRecords(value)
	value = processor.EncodePayload(value)
	value = processor.DecodeReply(value)
	value = processor.OpenChannel(value)
	value = processor.CloseChannel(value)
	value = processor.FlushBuffer(value)
	value = processor.SaveResult(value)
	return value
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 35.11s | 63.78s | <span class="benchmark-delta-negative">+81.6%</span> | — | — | — |
| **Process Start → First Event** | 0.28s | 0.29s | — | — | — | — |
| **First Event → First Tool Call** | 4.10s | 3.24s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 19 | <span class="benchmark-delta-negative">+375.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 19 | <span class="benchmark-delta-negative">+375.0%</span> | — | — | — |
| **Output Tokens** | 1236 | 687 | <span class="benchmark-delta-positive">-44.4%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 56 | 115 | <span class="benchmark-delta-negative">+105.4%</span> | — | — | — |
| **Total Input Tokens** | 102208 | 207278 | <span class="benchmark-delta-negative">+102.8%</span> | — | — | — |
| **Cached Input Tokens** | 93952 | 188416 | <span class="benchmark-delta-positive">+100.5%</span> | — | — | — |
| **Uncached Input Tokens** | 8256 | 18862 | <span class="benchmark-delta-negative">+128.5%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 11.38:1 | 9.99:1 | <span class="benchmark-delta-negative">-12.2%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File pipeline/workflow.go modified (41 lines -> 41 lines)
* **MCP Edit**: File pipeline/workflow.go modified (41 lines -> 41 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*vendor*&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;rename-plan.md&#39; -g &#39;*.go&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat rename-plan.md &amp;&amp; \
cat pipeline/workflow.go &amp;&amp; \
cat pipeline/workflow_test.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat rename-plan.md &amp;&amp; \
cat AGENTS.override.md &amp;&amp; \
rg -n &#39;type Processor|func \\(.*Processor|Processor\\)&#39; . --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.LoadInput&#34;,
  &#34;to&#34;: &#34;ReadRequest&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.CheckInput&#34;,
  &#34;to&#34;: &#34;ValidateRequest&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.NormalizeValue&#34;,
  &#34;to&#34;: &#34;CanonicalizeValue&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.MapFields&#34;,
  &#34;to&#34;: &#34;ProjectFields&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.RouteValue&#34;,
  &#34;to&#34;: &#34;DispatchValue&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.StampEvent&#34;,
  &#34;to&#34;: &#34;AnnotateEvent&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.CountItems&#34;,
  &#34;to&#34;: &#34;TallyItems&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.MergeParts&#34;,
  &#34;to&#34;: &#34;CombineParts&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.FilterNoise&#34;,
  &#34;to&#34;: &#34;PruneNoise&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.SortRecords&#34;,
  &#34;to&#34;: &#34;OrderRecords&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.EncodePayload&#34;,
  &#34;to&#34;: &#34;MarshalPayload&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.DecodeReply&#34;,
  &#34;to&#34;: &#34;ParseReply&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.OpenChannel&#34;,
  &#34;to&#34;: &#34;ConnectChannel&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.CloseChannel&#34;,
  &#34;to&#34;: &#34;ReleaseChannel&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.FlushBuffer&#34;,
  &#34;to&#34;: &#34;DrainBuffer&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;file&#34;: &#34;pipeline/workflow.go&#34;,
  &#34;symbol&#34;: &#34;Processor.SaveResult&#34;,
  &#34;to&#34;: &#34;PersistResult&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I had not confirmed that `semantic_batch` could dispatch `semantic_rename`; I saw its tool entry but didn’t inspect its supported operation schema. I therefore ran the renames consecutively as standalone semantic calls.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I would ask my owner to add semedit’s tools: semantic rename updated each `Processor` method and its call sites reliably, which made this bulk refactor feel straightforward.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

