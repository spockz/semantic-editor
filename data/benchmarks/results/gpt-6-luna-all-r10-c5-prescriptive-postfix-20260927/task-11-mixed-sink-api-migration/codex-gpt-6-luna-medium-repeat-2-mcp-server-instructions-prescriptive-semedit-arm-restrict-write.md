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

* **Date**: 2026-09-27 03:12:40 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-6-luna/medium (repeat 2)`

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
| **Wall-Clock Latency** | 137.52s | 191.47s | <span class="benchmark-delta-negative">+39.2%</span> | 53.78s | 265.34s | <span class="benchmark-delta-negative">+393.4%</span> |
| **Process Start → First Event** | 0.13s | 0.46s | — | 0.13s | 0.13s | — |
| **First Event → First Tool Call** | 4.89s | 4.73s | — | 4.83s | 6.85s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 5 | 5 | 0% | 1 | 5 | <span class="benchmark-delta-negative">+400.0%</span> |
| **Internal Tool Cycles** | 13 | 35 | <span class="benchmark-delta-negative">+169.2%</span> | 7 | 42 | <span class="benchmark-delta-negative">+500.0%</span> |
| **Initial Load / Discovery Turns** | 3 | 3 | 0% | 1 | 4 | <span class="benchmark-delta-negative">+300.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 13 | 35 | <span class="benchmark-delta-negative">+169.2%</span> | 7 | 42 | <span class="benchmark-delta-negative">+500.0%</span> |
| **Output Tokens** | 14936 | 15370 | <span class="benchmark-delta-negative">+2.9%</span> | 1689 | 22516 | <span class="benchmark-delta-negative">+1233.1%</span> |
| **Reasoning / Thinking Tokens** | 7193 | 6475 | <span class="benchmark-delta-positive">-10.0%</span> | 700 | 10120 | <span class="benchmark-delta-negative">+1345.7%</span> |
| **Total Input Tokens** | 1749546 | 2825970 | <span class="benchmark-delta-negative">+61.5%</span> | 197041 | 5063968 | <span class="benchmark-delta-negative">+2470.0%</span> |
| **Cached Input Tokens** | 1636096 | 2645760 | <span class="benchmark-delta-positive">+61.7%</span> | 179968 | 4816896 | <span class="benchmark-delta-positive">+2576.5%</span> |
| **Uncached Input Tokens** | 113450 | 180210 | <span class="benchmark-delta-negative">+58.8%</span> | 17073 | 247072 | <span class="benchmark-delta-negative">+1347.2%</span> |
| **Cached vs Uncached Token Ratio** | 14.42:1 | 14.68:1 | <span class="benchmark-delta-positive">+1.8%</span> | 10.54:1 | 19.50:1 | <span class="benchmark-delta-positive">+85.0%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ❌ FAIL | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; -g &#39;*.go&#39; | head -100 &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;tools/benchmark-harness/README.md&#39; -g &#39;AGENTS*&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -120 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event.go audit/normalize.go audit/sink.go service/dispatch.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
cat go.mod &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,220p&#39; service/dispatch.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch_test.go audit/event_test.go audit/summary.go metrics/writer.go legacy/export.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/event_test.go audit/summary.go audit/summary_test.go service/dispatch_test.go legacy/export.go metrics/writer.go &amp;&amp; \
rg -n \&#34;probe|DispatchAll|Record\\(\&#34; . --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat Makefile 2&gt;/dev/null || \
true; rg -n &#39;probe|health|Normalize|DispatchAll|Record\\(&#39; --glob &#39;&#34;&#39;!*.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;Makefile&#39; -g &#39;*.md&#39; -g &#39;go.sum&#39; -g &#39;AGENTS.md&#39;&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
make check&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel; git status --short --untracked-files=all -- service audit; sed -n &#39;1,80p&#39; audit/normalize.go; sed -n &#39;1,100p&#39; service/dispatch.go; git check-ignore -v audit/normalize.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go; git diff -- go.mod go.sum; git status --short -- audit service go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/sink.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git status --short -- go.mod go.sum audit/sink.go&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;placement&#34;: &#34;private_end&#34;,
  &#34;source&#34;: &#34;func deliver(sink audit.Sink, event audit.Event) error {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn sink.Write(event)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/summary.go audit/normalize.go; git diff -- go.mod go.sum; rg -n &#39;probe|health|Classify|NormalizeKind&#39; audit service metrics legacy&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;return deliver(d.sink, event)&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := deliver(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; service/dispatch.go; git diff -- go.mod go.sum; go test ./...&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go service/dispatch.go go.mod go.sum &amp;&amp; \
git status --short -- audit/normalize.go service/dispatch.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go service/dispatch.go &amp;&amp; \
git status --short -- . &amp;&amp; \
git diff --no-index /dev/null audit/normalize.go; git diff --no-index /dev/null service/dispatch.go&#39;</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
git status --short -- go.mod go.sum audit/normalize.go service/dispatch.go &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go audit/normalize.go &amp;&amp; \
git status --short -- go.mod go.sum &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service/dispatch.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;deliver&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;NormalizeKind&#34;,
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>31</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>32</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>33</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;dispatcher := NewDispatcher(sink)\n\tfor _, event := range events {\n\t\tif err := dispatcher.Record(event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>34</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>35</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 13 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 12 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*vendor*&#39;&#34;&#39; | head -100&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,240p&#39; audit/*.go &amp;&amp; \
sed -n &#39;1,220p&#39; service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;*.md&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Record|DispatchAll|Classify|probe|audit|Sink|Normalize|health&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
cat go.mod &amp;&amp; \
ls -la&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
rg -n \&#34;Event|Kind|Probe|Persist|Deliver|Sink\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch_test.go service/replay.go legacy/export.go legacy/export_test.go metrics/writer.go audit/summary_test.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event.go audit/normalize.go audit/sink.go service/dispatch.go service/replay.go audit/fanout.go audit/transaction.go &amp;&amp; \
cat Makefile 2&gt;/dev/null || \
true&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat db/db.go journal/file.go audit/normalize.go&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Record&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
go test ./... &amp;&amp; \
git status --short &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go service/replay.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func Deliver(sink Sink, event Event) error {\n\tevent.Kind = strings.ToLower(strings.TrimSpace(event.Kind))\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn sink.Write(event)\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- audit service &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
git check-ignore -v audit/normalize.go || \
true&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): replace_body Record in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790469880831246000/service/dispatch.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Record&#34;,
  &#34;body&#34;: &#34;return audit.Deliver(d.sink, event)&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;return audit.Deliver(d.sink, event)&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;mkdir -p docs &amp;&amp; \
python3 -c &#39;from pathlib import Path; p=Path(\&#34;docs/SUBOPTIMAL_TOOLS.md\&#34;); old=p.read_text() if p.exists() else \&#34;# Suboptimal tool behavior\\n\&#34;; entry=\&#34;\\n## 2026-09-27: semantic_replace_body method selector\\n\\n- Tool: &#34;&#39;`semantic_replace_body`&#39;&#34;\\n- Target: &#34;&#39;`service/dispatch.go`&#39;&#34;\\n- Observed failure: selecting &#34;&#39;`Record` returned `symbol not found` for a receiver method.&#39;&#34;\\n- Workaround: retried with the receiver-qualified selector &#34;&#39;`(*Dispatcher).Record`; replacement succeeded without a manual source edit.&#39;&#34;\\n- Root cause: method selectors require receiver qualification in this workspace.\\n\&#34;; p.write_text(old+entry)&#39;&#34;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := audit.Deliver(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := audit.Deliver(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;(FanoutSink).Write&#34;,
  &#34;body&#34;: &#34;if err := Deliver(f.First, event); err != nil {\n\t\treturn err\n\t}\n\treturn Deliver(f.Second, event)&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Commit&#34;,
  &#34;body&#34;: &#34;for _, event := range b.queue {\n\t\tif err := Deliver(b.Sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/normalize.go service/dispatch.go service/replay.go &amp;&amp; \
git status --short &amp;&amp; \
rg -n &#39;\\.Write\\(event\\)&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; .&#34;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- audit go.mod go.sum service &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/event.go audit/sink.go service/dispatch.go service/replay.go audit/fanout.go audit/transaction.go&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/audit-sinks.md &amp;&amp; \
cat audit/event_test.go service/dispatch_test.go audit/summary.go audit/normalize.go &amp;&amp; \
rg -n &#39;normalize|Normalize|Kind|probe|Probe|Write\\(&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; . &amp;&amp; \
rg -n &#39;normalize|Normalize|Kind|probe|Probe&#39; --glob &#39;*_test.go&#39; .&#34;</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;Deliver&#34;,
  &#34;body&#34;: &#34;event.Kind = strings.ToLower(strings.TrimSpace(event.Kind))\n\treturn sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch strings.ToLower(strings.TrimSpace(kind)) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 -c &#39;from pathlib import Path; p=Path(\&#34;docs/SUBOPTIMAL_TOOLS.md\&#34;); old=p.read_text() if p.exists() else \&#34;# Suboptimal tool behavior\\n\&#34;; entry=\&#34;\\n## 2026-09-27: semantic_replace_body left required import unresolved\\n\\n- Tool: &#34;&#39;`semantic_replace_body`&#39;&#34;\\n- Target: &#34;&#39;`audit/summary.go`&#39;&#34;\\n- Observed result: body replacement succeeded, but the tool reported introduced undefined &#34;&#39;`strings` diagnostics.&#39;&#34;\\n- Workaround: run semantic import organization for the updated source file.\\n- Root cause: replacing a body does not automatically add imports used by the replacement.\\n\&#34;; p.write_text(old+entry)&#39;&#34;</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff --exit-code -- go.mod go.sum &amp;&amp; \
git diff --check&#39;</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --exit-code -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --exit-code -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;
}</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>31</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;Deliver&#34;
}</code></pre></td></tr>
<tr><td>32</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>33</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;
}</code></pre></td></tr>
<tr><td>34</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>35</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;Deliver&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\treturn sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>36</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>37</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>38</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 -c &#39;from pathlib import Path; p=Path(\&#34;docs/SUBOPTIMAL_TOOLS.md\&#34;); old=p.read_text() if p.exists() else \&#34;# Suboptimal tool behavior\\n\&#34;; entry=\&#34;\\n## 2026-09-27: semantic_replace_body left unused import\\n\\n- Tool: &#34;&#39;`semantic_replace_body`&#39;&#34;\\n- Target: &#34;&#39;`audit/summary.go`&#39;&#34;\\n- Observed result: replacing inline normalization with &#34;&#39;`NormalizeKind` left an unused `strings` import, reported as a diagnostic.&#39;&#34;\\n- Workaround: organize imports for &#34;&#39;`audit/summary.go`.&#39;&#34;\\n- Root cause: body replacement does not clean up imports.\\n\&#34;; p.write_text(old+entry)&#39;&#34;</code></pre></td></tr>
<tr><td>39</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>40</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>41</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>42</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --exit-code -- go.mod go.sum &amp;&amp; \
git diff --check&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not batch the operations because I treated each semantic edit as a separate, incremental change and did not consider `semantic_batch`. Batching was available and suitable for the source edits; it was an oversight.</div></div>

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
| **Wall-Clock Latency** | 122.78s | 190.00s | <span class="benchmark-delta-negative">+54.7%</span> | 114.56s | 204.93s | <span class="benchmark-delta-negative">+78.9%</span> |
| **Process Start → First Event** | 0.13s | 0.12s | — | 0.13s | 0.13s | — |
| **First Event → First Tool Call** | 11.76s | 6.06s | — | 7.01s | 6.25s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 5 | 5 | 0% | 5 | 2 | <span class="benchmark-delta-positive">-60.0%</span> |
| **Internal Tool Cycles** | 12 | 24 | <span class="benchmark-delta-negative">+100.0%</span> | 13 | 38 | <span class="benchmark-delta-negative">+192.3%</span> |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | 4 | 3 | <span class="benchmark-delta-positive">-25.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 12 | 24 | <span class="benchmark-delta-negative">+100.0%</span> | 13 | 38 | <span class="benchmark-delta-negative">+192.3%</span> |
| **Output Tokens** | 14056 | 21324 | <span class="benchmark-delta-negative">+51.7%</span> | 13497 | 7472 | <span class="benchmark-delta-positive">-44.6%</span> |
| **Reasoning / Thinking Tokens** | 6607 | 12505 | <span class="benchmark-delta-negative">+89.3%</span> | 3981 | 3288 | <span class="benchmark-delta-positive">-17.4%</span> |
| **Total Input Tokens** | 2553460 | 3385420 | <span class="benchmark-delta-negative">+32.6%</span> | 2179335 | 1452848 | <span class="benchmark-delta-positive">-33.3%</span> |
| **Cached Input Tokens** | 2390784 | 3152384 | <span class="benchmark-delta-positive">+31.9%</span> | 1986048 | 1353984 | <span class="benchmark-delta-negative">-31.8%</span> |
| **Uncached Input Tokens** | 162676 | 233036 | <span class="benchmark-delta-negative">+43.3%</span> | 193287 | 98864 | <span class="benchmark-delta-positive">-48.9%</span> |
| **Cached vs Uncached Token Ratio** | 14.70:1 | 13.53:1 | <span class="benchmark-delta-negative">-8.0%</span> | 10.28:1 | 13.70:1 | <span class="benchmark-delta-positive">+33.3%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 10 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 9 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;README*&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
git status --short &amp;&amp; \
rg -n \&#34;event|probe|Persist|Deliver|Sink|Kind\&#34; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md &amp;&amp; \
rg --files | sort &amp;&amp; \
rg -n &#34;audit|Event|probe|persist|Sink|Delivery|Kind&#34;&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event.go audit/normalize.go audit/sink.go service/dispatch.go metrics/writer.go legacy/export.go cmd/migrate/main.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go &amp;&amp; \
cat audit/summary.go&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;README*&#39; -g &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;include_unexported&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func Prepare(event Event) (Event, bool) {\n\tevent.Kind = strings.ToLower(strings.TrimSpace(event.Kind))\n\treturn event, event.Kind != \&#34;probe\&#34;\n}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go service/dispatch.go &amp;&amp; \
git status --short -- audit/normalize.go service/dispatch.go go.mod go.sum &amp;&amp; \
git diff --check -- audit/normalize.go service/dispatch.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go&#39;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/event.go audit/sink.go service/dispatch.go &amp;&amp; \
git status --short -- audit/normalize.go audit/event.go audit/sink.go service/dispatch.go go.mod go.sum &amp;&amp; \
git diff --no-index /dev/null audit/normalize.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event, deliver := audit.Prepare(event)\n\tif !deliver {\n\t\treturn nil\n\t}\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff --check -- audit/normalize.go service/dispatch.go &amp;&amp; \
git status --short -- audit/normalize.go service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tevent, deliver := audit.Prepare(event)\n\t\tif !deliver {\n\t\t\tcontinue\n\t\t}\n\t\tif err := sink.Write(event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Classify|Normalize|probe|control|Kind\&#34; --glob &#39;*.go&#39; &amp;&amp; \
cat audit/normalize.go audit/event.go audit/summary.go service/dispatch.go metrics/writer.go legacy/export.go &amp;&amp; \
git status --short -- audit go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event, deliver := Prepare(event)\n\tif !deliver {\n\t\treturn nil\n\t}\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/summary.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff --check -- audit/summary.go &amp;&amp; \
git status --short -- audit/summary.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- audit/normalize.go audit/sink.go service/dispatch.go &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch.go audit/normalize.go &amp;&amp; \
go test ./service ./audit &amp;&amp; \
git diff --check -- service/dispatch.go audit/normalize.go &amp;&amp; \
git status --short -- service/dispatch.go audit/normalize.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff --check -- service/dispatch.go audit/normalize.go audit/summary.go &amp;&amp; \
git status --short -- service/dispatch.go audit/normalize.go audit/summary.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd; cat audit/normalize.go; cat audit/sink.go; cat service/dispatch.go; git status --short -- audit/normalize.go audit/sink.go service/dispatch.go go.mod go.sum; git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event_test.go audit/summary_test.go audit/sink.go audit/summary.go docs/audit-sinks.md; cat go.mod; git diff --no-index /dev/null audit/normalize.go&#39;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;Prepare&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\treturn event, event.Kind != \&#34;probe\&#34;&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/summary.go&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Record&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 18 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 17 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS*&#39; -g &#39;*.go&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;README.md&#39; -g &#39;docs/**&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
rg -n \&#34;Audit|audit|Probe|probe|Sink|Deliver|Persist|EventKind|kind\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
rg --files&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/fanout.go audit/sink.go audit/transaction.go service/dispatch.go service/replay.go journal/file.go audit/summary.go metrics/writer.go legacy/export.go &amp;&amp; \
rg --files&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;include_tests&#34;: false
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
git status --short &amp;&amp; \
cat audit/event.go&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service&#34;,
  &#34;include_tests&#34;: false
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/fanout.go audit/transaction.go service/dispatch.go service/replay.go &amp;&amp; \
git diff -- audit/normalize.go audit/fanout.go audit/transaction.go service/dispatch.go service/replay.go &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git status --short -- audit/normalize.go audit/fanout.go audit/transaction.go service/dispatch.go service/replay.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd; cat audit/normalize.go; git rev-parse --show-toplevel; git status --short&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve audit.MemorySink.Write in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-prefer_discover_semedit_task-11-mixed-sink-api-migration_1790470660221726000/audit/sink.go: invalid identifier: &#34;audit.MemorySink.Write&#34; (expected [Receiver.]Name)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;audit.MemorySink.Write&#34;,
  &#34;file&#34;: &#34;audit/sink.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/normalize.go audit/fanout.go audit/transaction.go service/dispatch.go service/replay.go; git diff -- go.mod go.sum; git status --short --untracked-files=no -- go.mod go.sum audit/normalize.go audit/fanout.go audit/transaction.go service/dispatch.go service/replay.go; rg -n &#39;NormalizeKind|Deliver\\(|\\.Write\\(event\\)&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;FanoutSink.Write&#34;,
  &#34;file&#34;: &#34;audit/fanout.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/fanout.go &amp;&amp; \
git diff --check -- go.mod go.sum audit/normalize.go audit/fanout.go audit/transaction.go service/dispatch.go service/replay.go &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
cat audit/normalize.go audit/fanout.go&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;BufferedSink.Commit&#34;,
  &#34;file&#34;: &#34;audit/transaction.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go audit/normalize.go; git diff -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/summary.go &amp;&amp; \
git diff --check -- audit/summary.go go.mod go.sum &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
cat audit/summary.go&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch.go audit/normalize.go; git diff -- go.mod go.sum; git diff --check -- service/dispatch.go audit/normalize.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;file&#34;: &#34;service/replay.go&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/summary.go service/dispatch.go; git diff -- go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/sink.go audit/summary.go audit/event.go audit/transaction.go audit/fanout.go service/dispatch.go service/replay.go&#39;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git diff --check -- service/dispatch.go audit/normalize.go audit/summary.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat db/db.go journal/file.go&#39;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Write&#34;,
  &#34;file&#34;: &#34;audit/sink.go&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): validate function snippet: multiple declarations found in snippet: expected single function declaration, found 2 declarations<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}\n\nfunc IsControlKind(kind string) bool {\n\tswitch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn true\n\tdefault:\n\t\treturn false\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/report_feedback</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;mcp__semedit__semantic_insert_function&#34;,
  &#34;interface&#34;: &#34;mcp&#34;,
  &#34;intent&#34;: &#34;Add audit kind normalization and control classification helpers using semantic source editing.&#34;,
  &#34;parameters&#34;: {
    &#34;file&#34;: &#34;audit/normalize.go&#34;,
    &#34;source&#34;: &#34;two function declarations&#34;
  },
  &#34;observed_result&#34;: &#34;The semantic insertion rejected a snippet containing two functions because it accepts one declaration at a time.&#34;,
  &#34;unexpected_reason&#34;: &#34;The helper insertion was valid Go but the tool requires exactly one declaration per call.&#34;,
  &#34;manual_touchups&#34;: &#34;None; no files were changed by this failed call.&#34;,
  &#34;target_context&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func IsControlKind(kind string) bool {\n\tswitch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn true\n\tdefault:\n\t\treturn false\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: true,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/sink.go&#34;,
        &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
        &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;journal/file.go&#34;,
        &#34;symbol&#34;: &#34;(*JournalSink).Write&#34;,
        &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\tj.Entries = append(j.Entries, event)\n\treturn nil&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/summary.go&#34;,
        &#34;symbol&#34;: &#34;Classify&#34;,
        &#34;body&#34;: &#34;if IsControlKind(kind) {\n\t\treturn \&#34;control\&#34;\n\t}\n\treturn \&#34;data\&#34;&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): This action was rejected due to unacceptable risk. Reason: Verification on path &#34;.&#34; may format and modify the entire workspace, including tests the user explicitly forbade editing; a narrower read-only or source-scoped verification is needed. Do not bypass this rejection through a workaround or indirect execution. Continue with a safer alternative, or carry out checks to prove that the action is authorized or low risk before trying again. Complete unaffected work without asking for confirmation. Report anyt<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/sink.go&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;journal/file.go&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build ./...&#39;</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- audit/normalize.go audit/sink.go audit/summary.go journal/file.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/sink.go audit/summary.go journal/file.go &amp;&amp; \
git rev-parse --show-toplevel &amp;&amp; \
git status --short -- audit/normalize.go audit/sink.go audit/summary.go journal/file.go go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- audit go.mod go.sum &amp;&amp; \
git diff -- audit go.mod go.sum &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
cat audit/normalize.go audit/sink.go audit/summary.go journal/file.go&#39;</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event_test.go audit/summary_test.go service/dispatch_test.go audit/transaction.go audit/fanout.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/sink.go&#34;,
        &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
        &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;journal/file.go&#34;,
        &#34;symbol&#34;: &#34;(*JournalSink).Write&#34;,
        &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\tj.Entries = append(j.Entries, event)\n\treturn nil&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
        &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;symbol&#34;: &#34;DispatchAll&#34;,
        &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent.Kind = audit.NormalizeKind(event.Kind)\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/replay.go&#34;,
        &#34;symbol&#34;: &#34;Replay&#34;,
        &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent.Kind = audit.NormalizeKind(event.Kind)\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>31</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>32</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/sink.go&#34;
}</code></pre></td></tr>
<tr><td>33</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>34</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;journal/file.go&#34;
}</code></pre></td></tr>
<tr><td>35</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>36</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service/replay.go&#34;
}</code></pre></td></tr>
<tr><td>37</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build ./...&#39;</code></pre></td></tr>
<tr><td>38</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --exit-code -- go.mod go.sum; git status --short -- go.mod go.sum; git diff --stat -- audit service journal go.mod go.sum&#39;</code></pre></td></tr>
</tbody>
</table>

