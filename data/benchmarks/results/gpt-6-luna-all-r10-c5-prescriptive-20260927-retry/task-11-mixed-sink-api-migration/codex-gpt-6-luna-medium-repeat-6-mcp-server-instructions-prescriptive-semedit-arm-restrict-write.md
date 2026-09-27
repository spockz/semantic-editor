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

### Target: `codex/gpt-6-luna/medium (repeat 6)`

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
| **Wall-Clock Latency** | 37.37s | 163.40s | <span class="benchmark-delta-negative">+337.2%</span> | 248.61s | 300.01s | <span class="benchmark-delta-negative">+20.7%</span> |
| **Process Start → First Event** | 0.12s | 0.12s | — | 0.23s | 0.13s | — |
| **First Event → First Tool Call** | 3.86s | 5.81s | — | 5.22s | 4.87s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 5 | <span class="benchmark-delta-negative">+400.0%</span> | 5 | 5 | 0% |
| **Internal Tool Cycles** | 6 | 25 | <span class="benchmark-delta-negative">+316.7%</span> | 14 | 62 | <span class="benchmark-delta-negative">+342.9%</span> |
| **Initial Load / Discovery Turns** | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> | 2 | 5 | <span class="benchmark-delta-negative">+150.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 6 | 25 | <span class="benchmark-delta-negative">+316.7%</span> | 14 | 62 | <span class="benchmark-delta-negative">+342.9%</span> |
| **Output Tokens** | 1326 | 11620 | <span class="benchmark-delta-negative">+776.3%</span> | 18185 | 22963 | <span class="benchmark-delta-negative">+26.3%</span> |
| **Reasoning / Thinking Tokens** | 592 | 3921 | <span class="benchmark-delta-negative">+562.3%</span> | 7748 | 11860 | <span class="benchmark-delta-negative">+53.1%</span> |
| **Total Input Tokens** | 162124 | 2369906 | <span class="benchmark-delta-negative">+1361.8%</span> | 1677358 | 3010068 | <span class="benchmark-delta-negative">+79.5%</span> |
| **Cached Input Tokens** | 141312 | 2184192 | <span class="benchmark-delta-positive">+1445.7%</span> | 1562368 | 2818560 | <span class="benchmark-delta-positive">+80.4%</span> |
| **Uncached Input Tokens** | 20812 | 185714 | <span class="benchmark-delta-negative">+792.3%</span> | 114990 | 191508 | <span class="benchmark-delta-negative">+66.5%</span> |
| **Cached vs Uncached Token Ratio** | 6.79:1 | 11.76:1 | <span class="benchmark-delta-positive">+73.2%</span> | 13.59:1 | 14.72:1 | <span class="benchmark-delta-positive">+8.3%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ❌ FAIL | — |
| **Oracle L4: Verification Test** | ✅ PASS | ❌ FAIL | — | ✅ PASS | ❌ FAIL | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 9 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!*vendor*&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;docs/**&#39; | head -100; cat go.mod; sed -n &#39;1,240p&#39; audit/event.go; sed -n &#39;1,240p&#39; audit/normalize.go; sed -n &#39;1,240p&#39; audit/sink.go; sed -n &#39;1,240p&#39; service/dispatch.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch_test.go audit/normalize.go audit/event_test.go audit/summary.go audit/summary_test.go metrics/writer.go legacy/export.go&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/audit-sinks.md; sed -n &#39;1,200p&#39; metrics/writer.go; sed -n &#39;1,240p&#39; legacy/export.go; sed -n &#39;1,200p&#39; cmd/migrate/main.go; rg --files -g &#39;*semedit*&#39; -g &#39;.codex/**&#39; -g &#39;.agents/**&#39;&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;probe|Normalize|DispatchAll|Record\\(\&#34; . -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;source&#34;: &#34;func Normalize(event Event) Event {\n\tevent.Kind = strings.ToLower(strings.TrimSpace(event.Kind))\n\treturn event\n}&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go service/dispatch.go audit/summary.go &amp;&amp; \
git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff -- audit/normalize.go service/dispatch.go audit/summary.go &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;event = audit.Normalize(event)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; git rev-parse --show-toplevel; sed -n &#39;1,50p&#39; audit/normalize.go; sed -n &#39;1,45p&#39; service/dispatch.go; sed -n &#39;1,30p&#39; audit/summary.go; git check-ignore -v audit/normalize.go service/dispatch.go audit/summary.go || \
true&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent = audit.Normalize(event)\n\t\tif event.Kind == \&#34;probe\&#34; {\n\t\t\tcontinue\n\t\t}\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!*.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!*.mod&#39;&#34;&#39; | sort; git status --short; cat audit/normalize.go; cat service/dispatch.go&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): This action was rejected due to unacceptable risk. Reason: Verification on path &#39;.&#39; may format and write across the entire workspace, including protected tests, despite the user’s explicit prohibition on test edits; use a narrowly scoped read-only check instead. Do not bypass this rejection through a workaround or indirect execution. Continue with a safer alternative, or carry out checks to prove that the action is authorized or low risk before trying again. Complete unaffected work without asking for con<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files; git status --short; sed -n &#39;1,220p&#39; audit/normalize.go; sed -n &#39;1,220p&#39; audit/sink.go; sed -n &#39;1,220p&#39; audit/event.go; sed -n &#39;1,240p&#39; docs/audit-sinks.md; sed -n &#39;1,220p&#39; service/dispatch.go; sed -n &#39;1,220p&#39; audit/event_test.go; sed -n &#39;1,220p&#39; audit/summary.go&#34;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event = Normalize(event)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod; test ! -e go.sum; git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; audit/summary.go; sed -n &#39;1,160p&#39; audit/normalize.go; cat go.mod; test &#34;&#39;! -e go.sum&#39;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch Normalize(Event{Kind: kind}).Kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod; test ! -e go.sum; git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; service/dispatch.go; cat go.mod; test &#34;&#39;! -e go.sum&#39;</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;placement&#34;: &#34;public_start&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\treturn event&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod; test ! -e go.sum; git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 19 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 9 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | head -80 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
cat PROJECT_BRIEF.md 2&gt;/dev/null; sed -n &#39;1,240p&#39; audit/event.go; sed -n &#39;1,260p&#39; audit/fanout.go; sed -n &#39;1,220p&#39; audit/normalize.go; sed -n &#39;1,220p&#39; service/dispatch.go; sed -n &#39;1,240p&#39; db/db.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/sink.go; sed -n &#39;1,240p&#39; audit/transaction.go; sed -n &#39;1,240p&#39; audit/summary.go; sed -n &#39;1,200p&#39; metrics/metrics.go; sed -n &#39;1,240p&#39; metrics/writer.go; sed -n &#39;1,240p&#39; legacy/export.go; sed -n &#39;1,220p&#39; audit/event_test.go; sed -n &#39;1,260p&#39; audit/summary_test.go; sed -n &#39;1,260p&#39; service/dispatch_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel &amp;&amp; \
git status --short -- .&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Write\\(|DispatchAll|Classify|probe|health|normalize&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
cat go.mod&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat journal/file.go; cat api/router.go; cat go.mod; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39;&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;include_unexported&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/transaction.go; cat audit/fanout.go&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service&#34;,
  &#34;include_unexported&#34;: true
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;api&#34;,
  &#34;include_unexported&#34;: true
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go&#39;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; git rev-parse --show-toplevel; git status --short -- audit service metrics legacy go.mod go.sum; git diff -- audit service; sed -n &#39;1,200p&#39; audit/normalize.go; sed -n &#39;1,160p&#39; audit/fanout.go; sed -n &#39;1,180p&#39; audit/sink.go; sed -n &#39;1,180p&#39; audit/transaction.go; sed -n &#39;1,160p&#39; service/dispatch.go&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;MemorySink.Write&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/fanout.go audit/transaction.go service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;FanoutSink.Write&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; audit/normalize.go; sed -n &#39;1,100p&#39; audit/fanout.go; sed -n &#39;1,100p&#39; audit/transaction.go; sed -n &#39;1,100p&#39; service/dispatch.go; git status --short --untracked-files=no -- audit service go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve (*BufferedSink).Write in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790464979075940000/audit/transaction.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Write&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/summary.go; sed -n &#39;1,120p&#39; audit/normalize.go; git diff -- go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,140p&#39; service/dispatch.go; sed -n &#39;1,120p&#39; audit/normalize.go; go test ./...; git diff -- go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;
}</code></pre></td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat health/health.go&#39;</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event.go audit/sink.go audit/transaction.go audit/fanout.go service/replay.go metrics/metrics.go metrics/writer.go legacy/export.go&#39;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event_test.go audit/summary_test.go service/dispatch_test.go api/router.go&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve Write in audit/transaction.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Write&#34;,
  &#34;file&#34;: &#34;audit/transaction.go&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeEvent(event Event) Event {\n\tevent.Kind = strings.ToLower(strings.TrimSpace(event.Kind))\n\treturn event\n}&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event = NormalizeEvent(event)\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;(FanoutSink).Write&#34;,
  &#34;body&#34;: &#34;event = NormalizeEvent(event)\n\tif err := f.First.Write(event); err != nil {\n\t\treturn err\n\t}\n\treturn f.Second.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event = audit.NormalizeEvent(event)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent = audit.NormalizeEvent(event)\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent = audit.NormalizeEvent(event)\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): unsupported Go construct kind &#34;switch&#34;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;function&#34;: &#34;Classify&#34;,
  &#34;kind&#34;: &#34;switch&#34;,
  &#34;discriminator&#34;: &#34;kind&#34;,
  &#34;source&#34;: &#34;switch NormalizeEvent(Event{Kind: kind}).Kind {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat docs/SUBOPTIMAL_TOOLS.md 2&gt;/dev/null || \
true&#39;</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;kind = NormalizeEvent(Event{Kind: kind}).Kind\n\tswitch kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;Commit&#34;
}</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Commit&#34;,
  &#34;body&#34;: &#34;for _, event := range b.queue {\n\t\tevent = NormalizeEvent(event)\n\t\tif err := b.Sink.Write(event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile
p = Path(&#39;docs/SUBOPTIMAL_TOOLS.md&#39;)
p.parent.mkdir(parents=True, exist_ok=True)
text = &#39;&#39;&#39;# Suboptimal semantic tool calls\\n\\n- Tool: &#34;&#39;`semantic_replace_construct`&#39;&#34;\\n- Target: &#34;&#39;`audit/summary.go`&#39;&#34;\\n- Observed failure: the tool rejected construct kind &#34;&#39;`switch` as unsupported.&#39;&#34;\\n- Workaround: used &#34;&#39;`semantic_replace_body` on `Classify` to update the switch within the function body.&#39;&#34;\\n- Root cause: the tool supports only selected construct kinds and does not support switch replacement.\\n&#39;&#39;&#39;
fd, name = tempfile.mkstemp(prefix=&#39;.SUBOPTIMAL_TOOLS.&#39;, dir=p.parent)
try:
    with os.fdopen(fd, &#39;w&#39;) as f:
        f.write(text)
        f.flush()
        os.fsync(f.fileno())
    os.replace(name, p)
finally:
    if os.path.exists(name):
        os.unlink(name)
PY
cat docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
<tr><td>31</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>32</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;
}</code></pre></td></tr>
<tr><td>33</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeEvent&#34;
}</code></pre></td></tr>
<tr><td>34</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;FanoutSink.Write&#34;
}</code></pre></td></tr>
<tr><td>35</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>36</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;
}</code></pre></td></tr>
<tr><td>37</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;
}</code></pre></td></tr>
<tr><td>38</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;
}</code></pre></td></tr>
<tr><td>39</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Commit&#34;
}</code></pre></td></tr>
<tr><td>40</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod; if test -f go.sum; then cat go.sum; else echo &#39;go.sum absent&#39;; fi; cat docs/SUBOPTIMAL_TOOLS.md&#34;</code></pre></td></tr>
<tr><td>41</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;(FanoutSink).Write&#34;,
  &#34;body&#34;: &#34;if err := f.First.Write(event); err != nil {\n\t\treturn err\n\t}\n\treturn f.Second.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>42</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch kind {\n\tcase \&#34;health\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>43</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Commit&#34;,
  &#34;body&#34;: &#34;for _, event := range b.queue {\n\t\tif err := b.Sink.Write(event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>44</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>45</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;return d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>46</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>47</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;kind = NormalizeEvent(Event{Kind: kind}).Kind\n\tswitch kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>48</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
cat go.mod &amp;&amp; \
test ! -e go.sum&#39;</code></pre></td></tr>
<tr><td>49</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod; test ! -e go.sum&#39;</code></pre></td></tr>
<tr><td>50</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>51</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;
}</code></pre></td></tr>
<tr><td>52</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod; test ! -e go.sum&#39;</code></pre></td></tr>
<tr><td>53</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeEvent&#34;
}</code></pre></td></tr>
<tr><td>54</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event = audit.NormalizeEvent(event)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>55</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeEvent&#34;
}</code></pre></td></tr>
<tr><td>56</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>57</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;
}</code></pre></td></tr>
<tr><td>58</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod; test ! -e go.sum&#39;</code></pre></td></tr>
<tr><td>59</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;placement&#34;: &#34;before_symbol&#34;,
  &#34;target_symbol&#34;: &#34;NormalizeEvent&#34;
}</code></pre></td></tr>
<tr><td>60</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeEvent&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\treturn event&#34;
}</code></pre></td></tr>
<tr><td>61</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;kind = NormalizeKind(kind)\n\tswitch kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>62</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
</tbody>
</table>

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
| **Wall-Clock Latency** | 48.02s | 119.83s | <span class="benchmark-delta-negative">+149.5%</span> | 110.62s | 189.80s | <span class="benchmark-delta-negative">+71.6%</span> |
| **Process Start → First Event** | 0.12s | 0.13s | — | 0.13s | 0.12s | — |
| **First Event → First Tool Call** | 5.54s | 8.20s | — | 13.41s | 8.90s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 5 | <span class="benchmark-delta-negative">+400.0%</span> |
| **Internal Tool Cycles** | 6 | 19 | <span class="benchmark-delta-negative">+216.7%</span> | 8 | 30 | <span class="benchmark-delta-negative">+275.0%</span> |
| **Initial Load / Discovery Turns** | 1 | 2 | <span class="benchmark-delta-negative">+100.0%</span> | 2 | 2 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 6 | 19 | <span class="benchmark-delta-negative">+216.7%</span> | 8 | 30 | <span class="benchmark-delta-negative">+275.0%</span> |
| **Output Tokens** | 1601 | 2482 | <span class="benchmark-delta-negative">+55.0%</span> | 4233 | 19756 | <span class="benchmark-delta-negative">+366.7%</span> |
| **Reasoning / Thinking Tokens** | 508 | 1045 | <span class="benchmark-delta-negative">+105.7%</span> | 2018 | 11025 | <span class="benchmark-delta-negative">+446.3%</span> |
| **Total Input Tokens** | 174020 | 658141 | <span class="benchmark-delta-negative">+278.2%</span> | 377052 | 3705799 | <span class="benchmark-delta-negative">+882.8%</span> |
| **Cached Input Tokens** | 157696 | 611328 | <span class="benchmark-delta-positive">+287.7%</span> | 347136 | 3486464 | <span class="benchmark-delta-positive">+904.4%</span> |
| **Uncached Input Tokens** | 16324 | 46813 | <span class="benchmark-delta-negative">+186.8%</span> | 29916 | 219335 | <span class="benchmark-delta-negative">+633.2%</span> |
| **Cached vs Uncached Token Ratio** | 9.66:1 | 13.06:1 | <span class="benchmark-delta-positive">+35.2%</span> | 11.60:1 | 15.90:1 | <span class="benchmark-delta-positive">+37.0%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 15 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | head -160&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.md&#39; -g &#39;*.go&#39; | sort &amp;&amp; \
git status --short &amp;&amp; \
cat go.mod &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,240p&#39; service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,240p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,220p&#39; cmd/migrate/main.go&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,240p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,200p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &amp;&amp; \
cat AGENTS.override.md&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;include_tests&#34;: false
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go; git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat service/dispatch_test.go&#39;</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event.go audit/summary_test.go audit/event_test.go audit/sink.go legacy/export.go metrics/writer.go cmd/migrate/main.go service/dispatch.go&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;kind = NormalizeKind(kind)\n\tswitch kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent.Kind = audit.NormalizeKind(event.Kind)\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;,
  &#34;trust_workspace&#34;: true
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
git status --short -- go.mod go.sum audit/normalize.go audit/summary.go service/dispatch.go metrics/writer.go legacy/export.go&#39;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel; git check-ignore -v audit/normalize.go audit/summary.go service/dispatch.go; git status --short --untracked-files=all -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum metrics/writer.go legacy/export.go&#39;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 25 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 16 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS*&#39; -g &#39;*.go&#39; | head -120 &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS*&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; | head -80 &amp;&amp; \
rg -n \&#34;audit|probe|sink|delivery|event kind\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/fanout.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;find . -maxdepth 2 -type f | sort &amp;&amp; \
cat audit/normalize.go audit/event.go audit/sink.go audit/transaction.go audit/fanout.go service/dispatch.go service/replay.go journal/file.go go.mod &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/replay.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/transaction.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,260p&#39; journal/file.go &amp;&amp; \
sed -n &#39;1,240p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,200p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,200p&#39; metrics/metrics.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md audit/summary.go metrics/writer.go metrics/writer_test.go legacy/export.go api/router.go api/router_test.go&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Sink|\\.Write\\(|Classify|probe|health|Dispatch|Replay&#39; --glob &#39;*.go&#39; . &amp;&amp; \
cat go.mod &amp;&amp; \
git diff -- go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat &gt; /tmp/noop &lt;&lt;&#39;EOF&#39;
EOF
&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat api/router.go cmd/replay/main.go cmd/migrate/main.go&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
make check&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;source&#34;: &#34;func Deliver(sink Sink, event Event) error {\n\tevent.Kind = NormalizeKind(event.Kind)\n\tif event.Kind == \&#34;probe\&#34; {\n\t\treturn nil\n\t}\n\treturn sink.Write(event)\n}&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go service/replay.go audit/*_test.go service/*_test.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;return audit.Deliver(d.sink, event)&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,100p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,80p&#39; service/replay.go &amp;&amp; \
git status --short -- audit go.mod go.sum service&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := audit.Deliver(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := audit.Deliver(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go service/dispatch.go service/replay.go &amp;&amp; \
go test ./...&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- . &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
if [ -f Makefile ]; then make check; else printf &#39;%s\\n&#39; &#39;No Makefile in fixture; go test ./... passed.&#39;; fi&#34;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Deliver&#34;,
  &#34;file&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- go.mod go.sum &amp;&amp; \
git status --short -- . &amp;&amp; \
rg --files -g &#39;Makefile&#39; -g &#39;go.sum&#39; -g &#39;go.mod&#39; .&#34;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;,
  &#34;file&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- go.mod go.sum &amp;&amp; \
rg --files -g &#39;go.mod&#39; -g &#39;go.sum&#39; .&#34;</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;,
  &#34;file&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./audit &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Deliver&#34;,
  &#34;file&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./service&#39;</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;,
  &#34;file&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;
}</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because I treated the edits as sequential: each operation depended on the preceding edit’s updated source state. Batching was discovered, but I did not verify whether it supported these operations in sequence.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I used separate semantic calls because the changes were sequentially dependent: the normalization helper had to be added before delivery code could call it, and later edits targeted different functions. Batching was discovered and available, but it was unsuitable for those dependent edits.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

