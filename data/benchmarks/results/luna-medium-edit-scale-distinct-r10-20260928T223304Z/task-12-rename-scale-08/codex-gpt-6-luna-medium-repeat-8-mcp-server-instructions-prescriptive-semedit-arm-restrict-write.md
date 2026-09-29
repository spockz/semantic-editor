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

* **Date**: 2026-09-29 00:51:54 CEST

## Test case: `task-12-rename-scale-08`

### Target: `codex/gpt-6-luna/medium (repeat 8)`

#### Configuration: default prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/task_12_rename_scale_08.txtar](https://github.com/spockz/semantic-editor/blob/5c3edbf97d73081ae5bc79c9a3b7e150bbb26d8f/testdata/bench/task_12_rename_scale_08.txtar)

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

func Run(processor Processor, value int) int {
	value = processor.LoadInput(value)
	value = processor.CheckInput(value)
	value = processor.NormalizeValue(value)
	value = processor.MapFields(value)
	value = processor.RouteValue(value)
	value = processor.StampEvent(value)
	value = processor.CountItems(value)
	value = processor.MergeParts(value)
	return value
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 18.36s | 71.36s | <span class="benchmark-delta-negative">+288.7%</span> | — | — | — |
| **Process Start → First Event** | 0.13s | 0.13s | — | — | — | — |
| **First Event → First Tool Call** | 4.67s | 5.03s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 12 | <span class="benchmark-delta-negative">+200.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 12 | <span class="benchmark-delta-negative">+200.0%</span> | — | — | — |
| **Output Tokens** | 488 | 631 | <span class="benchmark-delta-negative">+29.3%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 38 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Total Input Tokens** | 96925 | 199835 | <span class="benchmark-delta-negative">+106.2%</span> | — | — | — |
| **Cached Input Tokens** | 85760 | 179200 | <span class="benchmark-delta-positive">+109.0%</span> | — | — | — |
| **Uncached Input Tokens** | 11165 | 20635 | <span class="benchmark-delta-negative">+84.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 7.68:1 | 8.68:1 | <span class="benchmark-delta-positive">+13.1%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File pipeline/workflow.go modified (25 lines -> 25 lines)
* **MCP Edit**: File pipeline/workflow.go modified (25 lines -> 25 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;rename-plan.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat rename-plan.md &amp;&amp; \
rg -n \&#34;type Processor|Processor|func \\(.*\\)\&#34; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat rename-plan.md &amp;&amp; \
cat AGENTS.override.md 2&gt;/dev/null &amp;&amp; \
find . -name AGENTS.md -print&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat pipeline/workflow.go; rg -n \&#34;semantic_rename|semantic_lookup\&#34; . --hidden --glob &#39;&#34;&#39;!rename-plan.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat pipeline/workflow.go &amp;&amp; \
rg -n &#39;LoadInput|CheckInput|NormalizeValue|MapFields|RouteValue|StampEvent|CountItems|MergeParts&#39; --glob &#39;&#34;&#39;!rename-plan.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Processor.LoadInput&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
p = Path(&#39;pipeline/workflow.go&#39;)
s = p.read_text()
renames = {
    &#39;LoadInput&#39;: &#39;ReadRequest&#39;,
    &#39;CheckInput&#39;: &#39;ValidateRequest&#39;,
    &#39;NormalizeValue&#39;: &#39;CanonicalizeValue&#39;,
    &#39;MapFields&#39;: &#39;ProjectFields&#39;,
    &#39;RouteValue&#39;: &#39;DispatchValue&#39;,
    &#39;StampEvent&#39;: &#39;AnnotateEvent&#39;,
    &#39;CountItems&#39;: &#39;TallyItems&#39;,
    &#39;MergeParts&#39;: &#39;CombineParts&#39;,
}
for old, new in renames.items():
    s = s.replace(old, new)
tmp = p.with_name(p.name + &#39;.tmp&#39;)
tmp.write_text(s)
tmp.replace(p)
PY
git diff -- pipeline/workflow.go &amp;&amp; \
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Processor.LoadInput&#34;,
  &#34;to&#34;: &#34;ReadRequest&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Processor.CheckInput&#34;,
  &#34;to&#34;: &#34;ValidateRequest&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Processor.NormalizeValue&#34;,
  &#34;to&#34;: &#34;CanonicalizeValue&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Processor.MapFields&#34;,
  &#34;to&#34;: &#34;ProjectFields&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Processor.RouteValue&#34;,
  &#34;to&#34;: &#34;DispatchValue&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Processor.StampEvent&#34;,
  &#34;to&#34;: &#34;AnnotateEvent&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Processor.CountItems&#34;,
  &#34;to&#34;: &#34;TallyItems&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Processor.MergeParts&#34;,
  &#34;to&#34;: &#34;CombineParts&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">The renames could have been grouped, but I ran them one at a time because I didn’t check for or use `semantic_batch` before starting. Batching wasn’t unsuitable; it was an execution choice.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

