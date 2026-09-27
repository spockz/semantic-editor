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

* **Date**: 2026-09-27 01:46:01 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-6-luna/medium (repeat 8)`

#### Configuration: default prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/task_11_mixed_sink_api_migration.txtar](https://github.com/spockz/semantic-editor/blob/2cc7f05ca8384db81d0fdf5daacbc587add75845/testdata/bench/task_11_mixed_sink_api_migration.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 109.62s | 196.23s | <span class="benchmark-delta-negative">+79.0%</span> | 170.46s | 175.35s | <span class="benchmark-delta-negative">+2.9%</span> |
| **Process Start → First Event** | 0.12s | 0.12s | — | 0.13s | 0.12s | — |
| **First Event → First Tool Call** | 4.58s | 6.69s | — | 4.54s | 5.26s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 5 | 5 | 0% | 5 | 2 | <span class="benchmark-delta-positive">-60.0%</span> |
| **Internal Tool Cycles** | 12 | 44 | <span class="benchmark-delta-negative">+266.7%</span> | 19 | 27 | <span class="benchmark-delta-negative">+42.1%</span> |
| **Initial Load / Discovery Turns** | 2 | 4 | <span class="benchmark-delta-negative">+100.0%</span> | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 12 | 44 | <span class="benchmark-delta-negative">+266.7%</span> | 19 | 27 | <span class="benchmark-delta-negative">+42.1%</span> |
| **Output Tokens** | 13154 | 13952 | <span class="benchmark-delta-negative">+6.1%</span> | 20141 | 7660 | <span class="benchmark-delta-positive">-62.0%</span> |
| **Reasoning / Thinking Tokens** | 5767 | 5048 | <span class="benchmark-delta-positive">-12.5%</span> | 9590 | 3570 | <span class="benchmark-delta-positive">-62.8%</span> |
| **Total Input Tokens** | 1650359 | 2804497 | <span class="benchmark-delta-negative">+69.9%</span> | 3073510 | 1094968 | <span class="benchmark-delta-positive">-64.4%</span> |
| **Cached Input Tokens** | 1540864 | 2594048 | <span class="benchmark-delta-positive">+68.4%</span> | 2887680 | 1020160 | <span class="benchmark-delta-negative">-64.7%</span> |
| **Uncached Input Tokens** | 109495 | 210449 | <span class="benchmark-delta-negative">+92.2%</span> | 185830 | 74808 | <span class="benchmark-delta-positive">-59.7%</span> |
| **Cached vs Uncached Token Ratio** | 14.07:1 | 12.33:1 | <span class="benchmark-delta-negative">-12.4%</span> | 15.54:1 | 13.64:1 | <span class="benchmark-delta-negative">-12.2%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | ❌ FAIL | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 15 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | head -100 &amp;&amp; \
rg -n \&#34;audit|probe|event kind|event_kind|Kind\&#34; -g &#39;*.go&#39; -g &#39;*.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS*.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event.go audit/normalize.go audit/sink.go service/dispatch.go audit/summary.go docs/audit-sinks.md&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS*.md&#39; -g &#39;docs/**&#39; | sort&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; &amp;&amp; \
cat Makefile&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git status --short &amp;&amp; \
git diff -- audit/normalize.go service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve Deliver: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Deliver&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go service/dispatch.go; git status --short -- audit/normalize.go service/dispatch.go go.mod go.sum; cat audit/normalize.go&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve Normalize: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd; ls -la; cat audit/normalize.go audit/sink.go service/dispatch.go audit/summary.go; git rev-parse --show-toplevel; git status --short --untracked-files=no -- audit service go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve Probe: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Probe&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --no-index /dev/null audit/normalize.go; go test ./...; git diff --no-index /dev/null go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go &amp;&amp; \
go test ./... &amp;&amp; \
git status --short -- audit/normalize.go service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/summary.go &amp;&amp; \
go test ./... &amp;&amp; \
git status --short -- audit go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch.go audit/normalize.go; git status --short -- service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve Normalize in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-default_task-11-mixed-sink-api-migration_1790463914583233000/audit/normalize.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...; git status --short -- service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event_test.go audit/summary_test.go service/dispatch_test.go&#39;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\tif audit.Classify(event.Kind) == \&#34;control\&#34; &amp;&amp; event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tevent.Kind = audit.NormalizeKind(event.Kind)\n\t\tif audit.Classify(event.Kind) == \&#34;control\&#34; &amp;&amp; event.Kind == \&#34;probe\&#34; {\n\t\t\tcontinue\n\t\t}\n\t\tif err := sink.Write(event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): This action was rejected due to unacceptable risk. Reason: Workspace-wide semantic verification runs formatting and may write files, potentially modifying the explicitly protected test files; a read-only or source-scoped verification is required. Do not bypass this rejection through a workaround or indirect execution. Continue with a safer alternative, or carry out checks to prove that the action is authorized or low risk before trying again. Complete unaffected work without asking for confirmation. Report <pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum audit/*_test.go service/*_test.go&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum &amp;&amp; \
git diff --exit-code -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum &amp;&amp; \
git diff --exit-code -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>31</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>32</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;
}</code></pre></td></tr>
<tr><td>33</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./audit&#39;</code></pre></td></tr>
<tr><td>34</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>35</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>36</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum &amp;&amp; \
git diff --exit-code -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>37</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;
}</code></pre></td></tr>
<tr><td>38</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./service&#39;</code></pre></td></tr>
<tr><td>39</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>40</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum &amp;&amp; \
git diff --exit-code -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>41</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>42</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;
}</code></pre></td></tr>
<tr><td>43</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>44</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 19 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 16 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/fanout.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat service/dispatch.go&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/transaction.go audit/summary.go db/db.go metrics/writer.go legacy/export.go health/health.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; service/dispatch.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;audit\\.|Write\\(|Classify|probe|health|BufferedSink|FanoutSink&#39; --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/transaction.go audit/summary.go audit/normalize.go service/replay.go health/health.go metrics/writer.go legacy/export.go &amp;&amp; \
rg -n &#39;probe|Probe|Kind|\\.Write\\(&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat journal/file.go service/replay.go api/router.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;go.mod&#39; -g &#39;*test.go&#39; | sort&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): validate function snippet: multiple declarations found in snippet: expected single function declaration, found 2 declarations<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}\n\nfunc Deliver(sink Sink, event Event) error {\n\tevent.Kind = NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn sink.Write(event)\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files docs | head&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
gofmt -w audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
git status --short &amp;&amp; \
git diff --&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func Deliver(sink Sink, event Event) error {\n\tevent.Kind = NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn sink.Write(event)\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
git status --short -- go.mod go.sum audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;return audit.Deliver(d.sink, event)&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel &amp;&amp; \
git status --short --untracked-files=all | rg &#39;run_codex_baseline|audit/normalize.go|go.mod|go.sum&#39;&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := audit.Deliver(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd &amp;&amp; \
cat audit/normalize.go audit/fanout.go audit/sink.go audit/transaction.go audit/summary.go service/dispatch.go service/replay.go journal/file.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := audit.Deliver(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/audit-sinks.md &amp;&amp; \
cat go.mod &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
rg -n &#39;NormalizeKind|NormalizeEvent|IsProbe&#39; audit service journal&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;(FanoutSink).Write&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\tif err := Deliver(f.First, event); err != nil {\n\t\treturn err\n\t}\n\treturn Deliver(f.Second, event)&#34;
}</code></pre></td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Commit&#34;,
  &#34;body&#34;: &#34;for _, event := range b.queue {\n\t\tif err := Deliver(b.Sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/summary.go audit/normalize.go &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
rg -n &#39;Classify\\(|probe|health&#39; audit&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./audit&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>17</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch.go &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat &gt; /tmp/suboptimal-entry &lt;&lt;&#39;EOF&#39;
# Suboptimal semantic tool behavior

- Tool: &#34;&#39;`semantic_insert_function`
- Target: `audit/normalize.go`
- Observed failure: a request containing two function declarations was rejected because the operation accepts exactly one function declaration.
- Workaround: split the operation into two successful single-function insertions; no manual source edit was needed.
- Root cause: the tool validates its snippet as one declaration.
EOF
mkdir -p docs
if [ -f docs/SUBOPTIMAL_TOOLS.md ]; then cat docs/SUBOPTIMAL_TOOLS.md /tmp/suboptimal-entry &gt; /tmp/suboptimal-combined; else cp /tmp/suboptimal-entry /tmp/suboptimal-combined; fi
python3 -c &#39;&#34;&#39;import os; p=\&#34;docs/SUBOPTIMAL_TOOLS.md\&#34;; d=open(\&#34;/tmp/suboptimal-combined\&#34;,\&#34;rb\&#34;).read(); f=open(p+\&#34;.tmp\&#34;,\&#34;wb\&#34;); f.write(d); f.flush(); os.fsync(f.fileno()); f.close(); os.replace(p+\&#34;.tmp\&#34;,p)&#39;
cat audit/normalize.go
git status --short&#34;</code></pre></td></tr>
<tr><td>18</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./service&#39;</code></pre></td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>19</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/fanout.go audit/sink.go audit/transaction.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
cat go.mod &amp;&amp; \
cat go.sum&#39;</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/audit-sinks.md &amp;&amp; \
cat audit/event_test.go audit/summary_test.go service/dispatch_test.go &amp;&amp; \
rg -n &#39;NormalizeKind|Deliver|FanoutSink|BufferedSink&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;Deliver&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\treturn sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;(FanoutSink).Write&#34;,
  &#34;body&#34;: &#34;if err := Deliver(f.First, event); err != nil {\n\t\treturn err\n\t}\n\treturn Deliver(f.Second, event)&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;test ! -e go.sum &amp;&amp; \
printf &#39;&#34;&#39;go.sum absent\\n&#39;; cat go.mod; rg -n &#39;probe|NormalizeKind|Deliver&#39; audit service --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch`; it was discovered and callable, but I chose individual semantic operations because each edit targeted a distinct function body and I handled them sequentially. That was a judgment call, not a technical limitation.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Configuration: prefer_discover_semedit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/task_11_mixed_sink_api_migration.txtar](https://github.com/spockz/semantic-editor/blob/2cc7f05ca8384db81d0fdf5daacbc587add75845/testdata/bench/task_11_mixed_sink_api_migration.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 103.63s | 290.60s | <span class="benchmark-delta-negative">+180.4%</span> | 57.03s | 269.11s | <span class="benchmark-delta-negative">+371.9%</span> |
| **Process Start → First Event** | 0.12s | 0.12s | — | 0.12s | 0.13s | — |
| **First Event → First Tool Call** | 8.65s | 10.64s | — | 6.16s | 6.08s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 5 | 5 | 0% | 1 | 5 | <span class="benchmark-delta-negative">+400.0%</span> |
| **Internal Tool Cycles** | 11 | 30 | <span class="benchmark-delta-negative">+172.7%</span> | 8 | 47 | <span class="benchmark-delta-negative">+487.5%</span> |
| **Initial Load / Discovery Turns** | 1 | 5 | <span class="benchmark-delta-negative">+400.0%</span> | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 11 | 30 | <span class="benchmark-delta-negative">+172.7%</span> | 8 | 47 | <span class="benchmark-delta-negative">+487.5%</span> |
| **Output Tokens** | 12181 | 15159 | <span class="benchmark-delta-negative">+24.4%</span> | 1980 | 27001 | <span class="benchmark-delta-negative">+1263.7%</span> |
| **Reasoning / Thinking Tokens** | 4832 | 4856 | <span class="benchmark-delta-negative">+0.5%</span> | 703 | 16307 | <span class="benchmark-delta-negative">+2219.6%</span> |
| **Total Input Tokens** | 1873778 | 4741794 | <span class="benchmark-delta-negative">+153.1%</span> | 356382 | 3482764 | <span class="benchmark-delta-negative">+877.3%</span> |
| **Cached Input Tokens** | 1749760 | 4514816 | <span class="benchmark-delta-positive">+158.0%</span> | 332288 | 3225600 | <span class="benchmark-delta-positive">+870.7%</span> |
| **Uncached Input Tokens** | 124018 | 226978 | <span class="benchmark-delta-negative">+83.0%</span> | 24094 | 257164 | <span class="benchmark-delta-negative">+967.3%</span> |
| **Cached vs Uncached Token Ratio** | 14.11:1 | 19.89:1 | <span class="benchmark-delta-positive">+41.0%</span> | 13.79:1 | 12.54:1 | <span class="benchmark-delta-negative">-9.1%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | ✅ PASS | ❌ FAIL | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 15 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;tools/benchmark-harness/README.md&#39; -g &#39;*.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; | head -120&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README.md&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files &amp;&amp; \
cat go.mod &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
cat AGENTS.override.md &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/normalize.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,240p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,240p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,220p&#39; cmd/migrate/main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,200p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,180p&#39; service/dispatch_test.go &amp;&amp; \
cat go.mod&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff -- audit/normalize.go service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): unknown param &#34;file&#34;; supported keys are [bash-bin, bash_bin, ghc-bin, ghc-version, ghc_bin, ghc_version, haskell-standalone, hls-bin, hls-version, hls_bin, hls_version, import-maven, import_maven, include-tests, include-unexported, include_tests, include_unexported, java-bin, java-version, java_bin, java_version, jdtls-home, jdtls_home, kind, kinds, kotlin-bin, kotlin_bin, language, make-bin, make_bin, metals-bin, metals-home, metals_bin, metals_home, path, standalone_haskell, trust-workspace, trust_worksp<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel &amp;&amp; \
git status --short --untracked-files=all -- audit/normalize.go service/dispatch.go go.mod go.sum &amp;&amp; \
git diff --no-index /dev/null audit/normalize.go; git diff --no-index /dev/null service/dispatch.go&#39;</code></pre></td><td><code>semedit/report_feedback</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;semantic_outline&#34;,
  &#34;interface&#34;: &#34;mcp&#34;,
  &#34;intent&#34;: &#34;Inspect the Go audit sink before making a semantic source edit.&#34;,
  &#34;manual_touchups&#34;: &#34;None.&#34;,
  &#34;observed_result&#34;: &#34;The call was rejected because the argument name file was unsupported; the tool reported path as the supported parameter.&#34;,
  &#34;parameters&#34;: {
    &#34;file&#34;: &#34;audit/sink.go&#34;
  },
  &#34;suspected_cause&#34;: &#34;The exposed schema documentation and runtime parameter parser disagree on the path argument.&#34;,
  &#34;target_context&#34;: &#34;audit/sink.go&#34;,
  &#34;unexpected_reason&#34;: &#34;Expected the documented file argument to select the Go source file.&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
cat go.mod &amp;&amp; \
if [ -f go.sum ]; then cat go.sum; else printf &#39;go.sum absent\\n&#39;; fi &amp;&amp; \
sed -n &#39;1,180p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,220p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,160p&#39; audit/event.go&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/sink.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
if [ -f go.sum ]; then cat go.sum; else printf &#39;go.sum absent\\n&#39;; fi &amp;&amp; \
sed -n &#39;1,160p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,160p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,160p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,180p&#39; service/dispatch.go&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff --check&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event.Kind = normalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
if [ -f go.sum ]; then cat go.sum; else printf &#39;go.sum absent\\n&#39;; fi &amp;&amp; \
sed -n &#39;1,200p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/normalize.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/normalize.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,100p&#39; metrics/writer.go &amp;&amp; \
git status --short -- audit/normalize.go audit/sink.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
if [ -f go.sum ]; then cat go.sum; else printf &#39;go.sum absent\\n&#39;; fi &amp;&amp; \
sed -n &#39;1,140p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,180p&#39; service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;Makefile&#39; -g &#39;makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;README*&#39; -g &#39;*.md&#39; | sort &amp;&amp; \
git diff -- audit/normalize.go audit/sink.go go.mod go.sum &amp;&amp; \
git status --short -- audit/normalize.go audit/sink.go go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- audit go.mod go.sum &amp;&amp; \
sed -n &#39;1,100p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/event.go&#34;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Classif|probe|control|normalize|Kind\&#34; . --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; &amp;&amp; \
git status --short -- audit go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat audit/summary_test.go&#39;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch normalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git status --short -- audit go.mod go.sum &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): unknown param &#34;new_name&#34;; supported keys are [auto_organize_imports, file, import-maven, import_maven, java-bin, java_bin, jdtls-home, jdtls_home, language, symbol, to, trust-workspace, trust_workspace]: invalid operation parameters<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;new_name&#34;: &#34;NormalizeKind&#34;,
  &#34;file&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/report_feedback</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;semantic_rename&#34;,
  &#34;interface&#34;: &#34;mcp&#34;,
  &#34;intent&#34;: &#34;Export the audit event-kind normalization helper so service delivery can normalize events before invoking configured sinks.&#34;,
  &#34;manual_touchups&#34;: &#34;None.&#34;,
  &#34;observed_result&#34;: &#34;The call was rejected because new_name was unsupported; the runtime schema lists to as the rename destination parameter.&#34;,
  &#34;parameters&#34;: {
    &#34;symbol&#34;: &#34;normalizeKind&#34;,
    &#34;new_name&#34;: &#34;NormalizeKind&#34;,
    &#34;file&#34;: &#34;audit/normalize.go&#34;
  },
  &#34;suspected_cause&#34;: &#34;The exposed tool argument documentation differs from the runtime parameter parser.&#34;,
  &#34;target_context&#34;: &#34;audit/normalize.go&#34;,
  &#34;unexpected_reason&#34;: &#34;Expected the documented new_name parameter to specify the rename destination.&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;to&#34;: &#34;NormalizeKind&#34;,
  &#34;file&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\tif err := writeEvent(sink, event); err != nil {\n\t\treturn err\n\t}\n}\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/report_feedback</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;semantic_replace_body&#34;,
  &#34;interface&#34;: &#34;mcp&#34;,
  &#34;intent&#34;: &#34;Normalize events in DispatchAll before delivering them to the configured audit sink.&#34;,
  &#34;manual_touchups&#34;: &#34;A follow-up semantic body replacement is needed to restore the function-local writeEvent method value that was omitted from the supplied replacement body.&#34;,
  &#34;observed_result&#34;: &#34;The tool applied the body but returned diagnostics showing writeEvent was undefined.&#34;,
  &#34;parameters&#34;: {
    &#34;file&#34;: &#34;service/dispatch.go&#34;,
    &#34;symbol&#34;: &#34;DispatchAll&#34;
  },
  &#34;suspected_cause&#34;: &#34;The replacement body omitted a local method value used by the previous implementation.&#34;,
  &#34;target_context&#34;: &#34;service/dispatch.go&#34;,
  &#34;unexpected_reason&#34;: &#34;The edit should preserve successful compilation and existing dispatch behavior while adding normalization.&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\nfor _, event := range events {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\tif err := writeEvent(sink, event); err != nil {\n\t\treturn err\n\t}\n}\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git status --short -- audit service go.mod go.sum &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,100p&#39; service/dispatch.go &amp;&amp; \
git status --short -- audit service go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 25 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 17 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;docs/**&#39; -g &#39;tools/benchmark-harness/**&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat audit/fanout.go &amp;&amp; \
cat audit/transaction.go &amp;&amp; \
cat service/dispatch.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
rg --files&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event_test.go audit/summary.go audit/summary_test.go audit/fanout.go audit/transaction.go legacy/export.go metrics/writer.go api/router.go &amp;&amp; \
cat AGENTS.override.md&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;include_tests&#34;: false
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Sink|\\.Write\\(|DispatchAll|Classify|probe|health&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;include_tests&#34;: false
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat journal/file.go service/replay.go service/dispatch_test.go api/router_test.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
rg -n &#39;Audit|audit|probe|Event|Sink&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat cmd/replay/main.go cmd/migrate/main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go service/dispatch.go service/replay.go journal/file.go metrics/writer.go legacy/export.go &amp;&amp; \
cat go.mod&#39;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func Normalize(event Event) Event {\n\tevent.Kind = strings.ToLower(strings.TrimSpace(event.Kind))\n\treturn event\n}&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
git diff -- go.mod go.sum &#39;*_test.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;add&#34;: [
    &#34;strings&#34;
  ]
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func Deliver(sink Sink, event Event) error {\n\tevent = Normalize(event)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn sink.Write(event)\n}&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;kind = Normalize(Event{Kind: kind}).Kind\n\tswitch kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event = Normalize(event)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;FanoutSink.Write&#34;,
  &#34;body&#34;: &#34;if err := Deliver(f.First, event); err != nil {\n\t\treturn err\n\t}\n\treturn Deliver(f.Second, event)&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Commit&#34;,
  &#34;body&#34;: &#34;for _, event := range b.queue {\n\t\tif err := Deliver(b.Sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;return audit.Deliver(d.sink, event)&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := audit.Deliver(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := audit.Deliver(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;journal/file.go&#34;,
  &#34;symbol&#34;: &#34;(*JournalSink).Write&#34;,
  &#34;body&#34;: &#34;event = audit.Normalize(event)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\tj.Entries = append(j.Entries, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): This action was rejected due to unacceptable risk. Reason: Verification over the entire workspace is not read-only: it may format and write files, including tests that the user explicitly forbade editing; the requested scope is broader than authorized. Do not bypass this rejection through a workaround or indirect execution. Continue with a safer alternative, or carry out checks to prove that the action is authorized or low risk before trying again. Complete unaffected work without asking for confirmation. R<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- go.mod go.sum &#39;*_test.go&#39; &amp;&amp; \
git diff -- audit service journal&#34;</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;audit/sink.go&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;audit/fanout.go&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;audit/transaction.go&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;service/replay.go&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;journal/file.go&#34;
}</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- go.mod go.sum &amp;&amp; \
git status --short -- &#39;*.go&#39; &amp;&amp; \
git diff -- audit service journal&#34;</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
cat audit/normalize.go audit/summary.go &amp;&amp; \
cat service/dispatch.go&#39;</code></pre></td></tr>
<tr><td>31</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go service/dispatch.go service/replay.go journal/file.go &amp;&amp; \
cat go.mod &amp;&amp; \
if test -f go.sum; then cat go.sum; else echo &#39;go.sum absent&#39;; fi&#34;</code></pre></td></tr>
<tr><td>32</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>33</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>34</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
if test -f go.sum; then cat go.sum; else echo &#39;go.sum absent&#39;; fi &amp;&amp; \
cat audit/normalize.go audit/summary.go&#34;</code></pre></td></tr>
<tr><td>35</td><td>—</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;include_tests&#34;: false
}</code></pre></td></tr>
<tr><td>36</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>37</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
if test -f go.sum; then cat go.sum; else echo &#39;go.sum absent&#39;; fi&#34;</code></pre></td></tr>
<tr><td>38</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Record&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>39</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>40</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>41</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>42</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\treturn event&#34;
}</code></pre></td></tr>
<tr><td>43</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;kind = NormalizeKind(kind)\n\tswitch kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>44</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>45</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>46</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>47</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
if test -f go.sum; then cat go.sum; else echo &#39;go.sum absent&#39;; fi&#34;</code></pre></td></tr>
</tbody>
</table>

