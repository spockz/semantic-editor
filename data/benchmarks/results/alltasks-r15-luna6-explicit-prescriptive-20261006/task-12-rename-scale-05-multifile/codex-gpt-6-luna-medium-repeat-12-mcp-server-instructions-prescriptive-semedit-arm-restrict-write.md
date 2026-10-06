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

* **Date**: 2026-10-06 08:51:12 CEST

## Test case: `task-12-rename-scale-05-multifile`

### Target: `codex/gpt-6-luna/medium (repeat 12)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_12_rename_scale_05_multifile.txtar` (`sha256:196c9f19b89240721cf7c261 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Perform these 5 renames consecutively, possibly batched if you know how. Use rename-plan.md, update every call site, preserve behavior, and run go test ./... before finishing. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Read the first five rows of rename-plan.md. In parallel call semantic_lookup with file/symbol pairs: pipeline/stage_input.go/Processor.LoadInput, pipeline/stage_validation.go/Processor.CheckInput, pipeline/stage_normalization.go/Processor.NormalizeValue, pipeline/stage_mapping.go/Processor.MapFields, pipeline/stage_routing.go/Processor.RouteValue. Then call semantic_rename in plan order using each returned file/symbol and to values ReadRequest, ValidateRequest, CanonicalizeValue, ProjectFields, DispatchValue. A semantic_batch may contain these ordered renames. Update all call sites and preserve behavior. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package pipeline keeps the rename targets together so the oracle can check all five declarations.
package pipeline

type Processor struct{}

func (Processor) LoadInput(value int) int { return value + 1 }
func (Processor) CheckInput(value int) int { return value + 2 }
func (Processor) NormalizeValue(value int) int { return value + 3 }
func (Processor) MapFields(value int) int { return value + 4 }
func (Processor) RouteValue(value int) int { return value + 5 }
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 26.21s | 61.70s | <span class="benchmark-delta-negative">+135.4%</span> | — | — | — |
| **Process Start → First Event** | 0.21s | 0.29s | — | — | — | — |
| **First Event → First Tool Call** | 4.82s | 6.33s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 14 | <span class="benchmark-delta-negative">+250.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 14 | <span class="benchmark-delta-negative">+250.0%</span> | — | — | — |
| **Output Tokens** | 672 | 1462 | <span class="benchmark-delta-negative">+117.6%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 38 | 434 | <span class="benchmark-delta-negative">+1042.1%</span> | — | — | — |
| **Total Input Tokens** | 104122 | 223116 | <span class="benchmark-delta-negative">+114.3%</span> | — | — | — |
| **Cached Input Tokens** | 93952 | 196608 | <span class="benchmark-delta-positive">+109.3%</span> | — | — | — |
| **Uncached Input Tokens** | 10170 | 26508 | <span class="benchmark-delta-negative">+160.6%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 9.24:1 | 7.42:1 | <span class="benchmark-delta-negative">-19.7%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File pipeline/processor.go modified (10 lines -> 10 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;rename-plan.md&#39; -g &#39;*.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,7p&#39; rename-plan.md&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat rename-plan.md &amp;&amp; \
git status --short &amp;&amp; \
rg -n &#39;\\b[A-Za-z_][A-Za-z0-9_]*\\b&#39; pipeline --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,12p&#39; rename-plan.md&#34;</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.LoadInput in pipeline/stage_input.go: symbol not found&#34;} resolve Processor.LoadInput in pipeline/stage_input.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_input.go&#34;,
  &#34;symbol&#34;: &#34;Processor.LoadInput&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.CheckInput in pipeline/stage_validation.go: symbol not found&#34;} resolve Processor.CheckInput in pipeline/stage_validation.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_validation.go&#34;,
  &#34;symbol&#34;: &#34;Processor.CheckInput&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.NormalizeValue in pipeline/stage_normalization.go: symbol not found&#34;} resolve Processor.NormalizeValue in pipeline/stage_normalization.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_normalization.go&#34;,
  &#34;symbol&#34;: &#34;Processor.NormalizeValue&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.MapFields in pipeline/stage_mapping.go: symbol not found&#34;} resolve Processor.MapFields in pipeline/stage_mapping.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_mapping.go&#34;,
  &#34;symbol&#34;: &#34;Processor.MapFields&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.RouteValue in pipeline/stage_routing.go: symbol not found&#34;} resolve Processor.RouteValue in pipeline/stage_routing.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_routing.go&#34;,
  &#34;symbol&#34;: &#34;Processor.RouteValue&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;SUBOPTIMAL_TOOLS.md&#39; -g &#39;stage_*.go&#39; -g &#39;rename-plan.md&#39; -g &#39;go.mod&#39;&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.LoadInput in pipeline/stage_input.go: symbol not found&#34;} resolve Processor.LoadInput in pipeline/stage_input.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_input.go&#34;,
  &#34;symbol&#34;: &#34;Processor.LoadInput&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-12-rename-scale-05-multifile_1791266751675040000&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.CheckInput in pipeline/stage_validation.go: symbol not found&#34;} resolve Processor.CheckInput in pipeline/stage_validation.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_validation.go&#34;,
  &#34;symbol&#34;: &#34;Processor.CheckInput&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-12-rename-scale-05-multifile_1791266751675040000&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.NormalizeValue in pipeline/stage_normalization.go: symbol not found&#34;} resolve Processor.NormalizeValue in pipeline/stage_normalization.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_normalization.go&#34;,
  &#34;symbol&#34;: &#34;Processor.NormalizeValue&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-12-rename-scale-05-multifile_1791266751675040000&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.MapFields in pipeline/stage_mapping.go: symbol not found&#34;} resolve Processor.MapFields in pipeline/stage_mapping.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_mapping.go&#34;,
  &#34;symbol&#34;: &#34;Processor.MapFields&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-12-rename-scale-05-multifile_1791266751675040000&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve Processor.RouteValue in pipeline/stage_routing.go: symbol not found&#34;} resolve Processor.RouteValue in pipeline/stage_routing.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;pipeline/stage_routing.go&#34;,
  &#34;symbol&#34;: &#34;Processor.RouteValue&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-12-rename-scale-05-multifile_1791266751675040000&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_12_rename_scale_05_multifile.txtar` (`sha256:196c9f19b89240721cf7c261 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Perform these 5 renames consecutively, possibly batched if you know how. Use rename-plan.md, update every call site, preserve behavior, and run go test ./... before finishing. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Perform these 5 renames consecutively, possibly batched if you know how. Use rename-plan.md, update every call site, preserve behavior, and run go test ./... before finishing. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package pipeline keeps the rename targets together so the oracle can check all five declarations.
package pipeline

type Processor struct{}

func (Processor) LoadInput(value int) int { return value + 1 }
func (Processor) CheckInput(value int) int { return value + 2 }
func (Processor) NormalizeValue(value int) int { return value + 3 }
func (Processor) MapFields(value int) int { return value + 4 }
func (Processor) RouteValue(value int) int { return value + 5 }
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 26.21s | 51.28s | <span class="benchmark-delta-negative">+95.6%</span> | — | — | — |
| **Process Start → First Event** | 0.21s | 0.13s | — | — | — | — |
| **First Event → First Tool Call** | 4.82s | 5.71s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 9 | <span class="benchmark-delta-negative">+125.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 4 | 9 | <span class="benchmark-delta-negative">+125.0%</span> | — | — | — |
| **Output Tokens** | 672 | 891 | <span class="benchmark-delta-negative">+32.6%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 38 | 197 | <span class="benchmark-delta-negative">+418.4%</span> | — | — | — |
| **Total Input Tokens** | 104122 | 191266 | <span class="benchmark-delta-negative">+83.7%</span> | — | — | — |
| **Cached Input Tokens** | 93952 | 143616 | <span class="benchmark-delta-positive">+52.9%</span> | — | — | — |
| **Uncached Input Tokens** | 10170 | 47650 | <span class="benchmark-delta-negative">+368.5%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 9.24:1 | 3.01:1 | <span class="benchmark-delta-negative">-67.4%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File pipeline/processor.go modified (10 lines -> 10 lines)
* **MCP Edit**: File pipeline/processor.go modified (10 lines -> 10 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;rename-plan.md&#39; -g &#39;*.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;rename-plan.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat rename-plan.md &amp;&amp; \
git status --short &amp;&amp; \
rg -n &#39;\\b[A-Za-z_][A-Za-z0-9_]*\\b&#39; pipeline --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat rename-plan.md &amp;&amp; \
cat AGENTS.override.md &amp;&amp; \
rg -n &#39;semantic_rename|semantic_lookup|semantic_batch&#39; . --glob &#39;&#34;&#39;!rename-plan.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;LoadInput&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;CheckInput&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;NormalizeValue&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;MapFields&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;RouteValue&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.LoadInput&#34;,
        &#34;to&#34;: &#34;ReadRequest&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.CheckInput&#34;,
        &#34;to&#34;: &#34;ValidateRequest&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.NormalizeValue&#34;,
        &#34;to&#34;: &#34;CanonicalizeValue&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.MapFields&#34;,
        &#34;to&#34;: &#34;ProjectFields&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.RouteValue&#34;,
        &#34;to&#34;: &#34;DispatchValue&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
      }
    }
  ],
  &#34;workspace_root&#34;: &#34;/Users/alessandro/.codex-workspaces/worktrees/318f14ee-b4fe-4414-9a7d-5427835dd89b/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-rename-scale-05-multifile_1791266761821366000&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git status --short &amp;&amp; \
git diff --name-only&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Using semedit’s semantic renames, I updated all five methods and their call sites in one batch with no diagnostics, and I’d ask my owner to add these tools to my coding setup.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

