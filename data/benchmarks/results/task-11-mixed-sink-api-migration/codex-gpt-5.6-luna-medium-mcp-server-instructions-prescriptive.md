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

* **Date**: 2026-09-22 12:08:49 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: default prompt · prescriptive MCP instructions

* **Fixture**: `testdata/bench/task_11_mixed_sink_api_migration.txtar` (`sha256:ac52f8e94dba22fb0aea5899 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Prefer using semantic editor operations if applicable. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 83.82s | 141.18s | <span class="benchmark-delta-negative">+68.4%</span> | 215.45s | 147.08s | N/A |
| **Process Start → First Event** | 0.29s | 0.15s | — | 0.16s | 0.19s | — |
| **First Event → First Tool Call** | 7.76s | 7.89s | — | 11.93s | 6.90s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | N/A |
| **Internal Tool Cycles** | 8 | 14 | <span class="benchmark-delta-negative">+75.0%</span> | 7 | 16 | N/A |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | 2 | 3 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 8 | 14 | <span class="benchmark-delta-negative">+75.0%</span> | 7 | 16 | N/A |
| **Output Tokens** | 3357 | 3601 | <span class="benchmark-delta-negative">+7.3%</span> | 4601 | 6732 | N/A |
| **Reasoning / Thinking Tokens** | 1363 | 1743 | <span class="benchmark-delta-negative">+27.9%</span> | 1818 | 2128 | N/A |
| **Total Input Tokens** | 231438 | 348487 | <span class="benchmark-delta-negative">+50.6%</span> | 305758 | 451592 | N/A |
| **Cached Input Tokens** | 197120 | 292608 | <span class="benchmark-delta-positive">+48.4%</span> | 275456 | 420608 | N/A |
| **Uncached Input Tokens** | 34318 | 55879 | <span class="benchmark-delta-negative">+62.8%</span> | 30302 | 30984 | N/A |
| **Cached vs Uncached Token Ratio** | 5.74:1 | 5.24:1 | <span class="benchmark-delta-negative">-8.8%</span> | 9.09:1 | 13.58:1 | N/A |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;\\n--- docs/brief ---\\n&#39; &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | sed -n &#39;1,160p&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in audit/event.go audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go metrics/writer.go legacy/export.go docs/audit-sinks.md cmd/migrate/main.go; do printf &#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/event.go; sed -n &#39;1,240p&#39; audit/normalize.go; sed -n &#39;1,240p&#39; audit/sink.go; sed -n &#39;1,260p&#39; service/dispatch.go; sed -n &#39;1,240p&#39; docs/audit-sinks.md; sed -n &#39;1,200p&#39; Makefile 2&gt;/dev/null || \
true&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in audit/event_test.go audit/summary_test.go service/dispatch_test.go metrics/writer_test.go legacy/export_test.go; do printf &#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,280p&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
printf &#39;&#34;&#39;\\n--- module ---\\n&#39; &amp;&amp; \
sed -n &#39;1,80p&#39; go.mod &amp;&amp; \
printf &#39;\\n--- all references ---\\n&#39; &amp;&amp; \
rg -n &#39;Normalize|Classify|Write\\(|Record\\(|DispatchAll|probe|health|Kind&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; audit/summary.go; sed -n &#39;1,260p&#39; legacy/export.go; sed -n &#39;1,260p&#39; metrics/writer.go; sed -n &#39;1,260p&#39; cmd/migrate/main.go; sed -n &#39;1,260p&#39; audit/event_test.go; sed -n &#39;1,260p&#39; audit/summary_test.go; sed -n &#39;1,260p&#39; service/dispatch_test.go; sed -n &#39;1,240p&#39; legacy/export_test.go; sed -n &#39;1,220p&#39; metrics/writer_test.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;kind = NormalizeKind(kind)\n\tswitch kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --check &amp;&amp; \
printf &#39;\\n--- fixture diff ---\\n&#39; &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
printf &#39;\\n--- protected-file diff ---\\n&#39; &amp;&amp; \
git diff --quiet -- go.mod go.sum; printf &#39;protected_diff_exit=%s\\n&#39; \&#34;&#34;&#39;$?&#34; &amp;&amp; \
printf &#39;&#34;&#39;\\n--- fixture status ---\\n&#39; &amp;&amp; \
git status --short -- audit service metrics legacy cmd go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- git root ---&#39; &amp;&amp; \
git rev-parse --show-toplevel &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- files ---&#39; &amp;&amp; \
ls -l audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum 2&gt;&amp;1 || \
true &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- contents ---&#39; &amp;&amp; \
sed -n &#39;1,100p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,100p&#39; service/dispatch.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- local status ---&#39; &amp;&amp; \
git status --short --untracked-files=all -- . &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tracked ---&#39; &amp;&amp; \
git ls-files -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent.Kind = audit.NormalizeKind(event.Kind)\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go vet ./... &amp;&amp; \
gofmt -l audit service metrics legacy cmd &amp;&amp; \
git diff --check -- audit/normalize.go audit/summary.go service/dispatch.go; test ! -e go.sum &amp;&amp; \
printf &#39;&#34;&#39;go.sum absent and untouched\\n&#39;; grep -qx &#39;module example.com/auditapp&#39; go.mod &amp;&amp; \
printf &#39;go.mod preserved\\n&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- audit service; git status --short; sed -n &#39;1,120p&#39; audit/normalize.go; sed -n &#39;1,120p&#39; service/dispatch.go&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go audit/summary.go service/dispatch.go audit/sink.go audit/event.go metrics/writer.go legacy/export.go go.mod go.sum; git diff --name-only -- audit service legacy metrics go.mod go.sum; git status --short -- audit service legacy metrics go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel; git ls-files -- audit/normalize.go audit/summary.go service/dispatch.go go.mod; git check-ignore -v audit/normalize.go audit/summary.go service/dispatch.go go.mod || \
true; sed -n &#39;1,100p&#39; audit/summary.go; sed -n &#39;1,100p&#39; go.mod&#34;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -d audit/normalize.go audit/summary.go service/dispatch.go; rg -n &#39;NormalizeKind|case \&#34;health\&#34;|case \&#34;probe\&#34;|Classify\\(&#39; audit service metrics legacy; test &#34;&#39;! -e Makefile &amp;&amp; \
echo &#39;&#34;&#39;no Makefile in fixture&#39;; go test ./...&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 13 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 17 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sed -n &#39;1,160p&#39;; printf &#39;\\n--- status ---\\n&#39;; git status --short; printf &#39;\\n--- brief ---\\n&#39;; if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | sed -n &#39;1,160p&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- audit/event.go ---&#39;; sed -n &#39;1,240p&#39; audit/event.go; printf &#39;%s\\n&#39; &#39;--- audit/normalize.go ---&#39;; sed -n &#39;1,240p&#39; audit/normalize.go; printf &#39;%s\\n&#39; &#39;--- audit/fanout.go ---&#39;; sed -n &#39;1,280p&#39; audit/fanout.go; printf &#39;%s\\n&#39; &#39;--- audit/sink.go ---&#39;; sed -n &#39;1,240p&#39; audit/sink.go; printf &#39;%s\\n&#39; &#39;--- audit/transaction.go ---&#39;; sed -n &#39;1,260p&#39; audit/transaction.go; printf &#39;%s\\n&#39; &#39;--- docs ---&#39;; sed -n &#39;1,260p&#39; docs/audit-sinks.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/transaction.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- references ---&#39;; rg -n \&#34;audit|Event|Probe|probe|Kind|Fanout|Buffered|MemorySink|Write\\(\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39; .; printf &#39;%s\\n&#39; &#39;--- service dispatch ---&#39;; sed -n &#39;1,280p&#39; service/dispatch.go; printf &#39;%s\\n&#39; &#39;--- service replay ---&#39;; sed -n &#39;1,240p&#39; service/replay.go; printf &#39;%s\\n&#39; &#39;--- db ---&#39;; sed -n &#39;1,240p&#39; db/db.go; printf &#39;%s\\n&#39; &#39;--- journal ---&#39;; sed -n &#39;1,240p&#39; journal/file.go; printf &#39;%s\\n&#39; &#39;--- legacy ---&#39;; sed -n &#39;1,240p&#39; legacy/export.go; printf &#39;%s\\n&#39; &#39;--- metrics ---&#39;; sed -n &#39;1,240p&#39; metrics/writer.go; sed -n &#39;1,220p&#39; metrics/metrics.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
sed -n &#39;1,240p&#39; docs/audit-sinks.md&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/summary.go; sed -n &#39;1,220p&#39; audit/summary_test.go; sed -n &#39;1,220p&#39; audit/event_test.go; sed -n &#39;1,220p&#39; audit/transaction.go; sed -n &#39;1,160p&#39; journal/file.go; sed -n &#39;1,120p&#39; go.mod; rg -n \&#34;delivery|control|persist|normalize|probe|sink\&#34; . --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,220p&#39; db/db.go &amp;&amp; \
sed -n &#39;1,220p&#39; journal/file.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w audit/normalize.go audit/summary.go audit/fanout.go audit/sink.go audit/transaction.go journal/file.go; git diff -- audit/normalize.go audit/summary.go audit/fanout.go audit/sink.go audit/transaction.go journal/file.go; printf &#39;\\n--- protected diff ---\\n&#39;; git diff -- go.mod go.sum; printf &#39;\\n--- test files changed in fixture ---\\n&#39;; git status --short -- audit &#39;*_test.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;audit|probe|Kind|Record|DispatchAll|Sink|normalize\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39; --glob &#39;*.txtar&#39; . | sed -n &#39;1,260p&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/transaction.go&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -la; git rev-parse --show-toplevel; git ls-files -- audit/normalize.go audit/summary.go journal/file.go; if [ -f Makefile ]; then make check; else echo &#39;make check unavailable: no Makefile&#39;; fi; go test ./...&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; service/replay.go &amp;&amp; \
sed -n &#39;1,260p&#39; testdata/bench/task_11_mixed_sink_api_migration.txtar 2&gt;/dev/null || \
true &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w audit/normalize.go audit/summary.go audit/fanout.go audit/sink.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go; go test ./...; go vet ./...; printf &#39;\\n--- protected files ---\\n&#39;; ls go.mod go.sum 2&gt;/dev/null || \
true; printf &#39;\\n--- changed source excerpts ---\\n&#39;; sed -n &#39;1,120p&#39; audit/normalize.go; sed -n &#39;1,100p&#39; audit/summary.go; sed -n &#39;1,100p&#39; audit/transaction.go; sed -n &#39;1,100p&#39; service/dispatch.go; sed -n &#39;1,80p&#39; service/replay.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;\\.Write\\(|Write\\(\&#34; --glob &#39;*.go&#39; . &amp;&amp; \
rg -n \&#34;semedit|semantic_\&#34; . --hidden -g &#39;&#34;&#39;!/.git/**&#39;&#34;&#39; | sed -n &#39;1,120p&#39; &amp;&amp; \
find . -maxdepth 2 -type f -name &#39;Makefile&#39; -o -name &#39;*.sh&#39; -print&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;command -v semedit || \
true; command -v semantic-editor || \
true; rg -n \&#34;func .*Normalize|Normaliz|probe\&#34; audit service journal docs --glob &#39;*.go&#39; --glob &#39;*.md&#39;&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod; find . -maxdepth 2 -type f | sort; git diff -- audit service journal metrics legacy go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go service/dispatch.go service/replay.go journal/file.go &amp;&amp; \
git diff --check &amp;&amp; \
go test ./...&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit journal service metrics legacy go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel; git status --short -- audit service journal metrics legacy go.mod go.sum; sed -n &#39;1,160p&#39; audit/normalize.go; sed -n &#39;1,100p&#39; audit/summary.go; sed -n &#39;1,120p&#39; audit/transaction.go; git diff --no-index /dev/null audit/normalize.go || \
true&#34;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go service/dispatch.go service/replay.go journal/file.go&#39;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short --untracked-files=all -- audit service journal metrics legacy go.mod go.sum; rg -n \&#34;probe|strings|Normalize|audit\\.Deliver\&#34; audit service journal metrics legacy --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
go vet ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not discover any callable `semantic_*` tools from the available semedit MCP server in the session. I therefore used ordinary repository editing and verification.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 4.36s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because I treated the independent function insert and body replacements as simpler sequential operations, despite batching being available. Batching was suitable but not chosen.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 8.21s; turns: 1; tool calls: 0.</p>
</details>
