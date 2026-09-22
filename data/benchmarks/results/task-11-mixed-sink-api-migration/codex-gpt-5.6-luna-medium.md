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

* **Date**: 2026-09-22 11:55:01 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: default prompt · none MCP instructions

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
| **Wall-Clock Latency** | 75.35s | 112.97s | <span class="benchmark-delta-negative">+49.9%</span> | 200.22s | 189.98s | N/A |
| **Process Start → First Event** | 0.42s | 0.14s | — | 0.15s | 0.14s | — |
| **First Event → First Tool Call** | 7.39s | 7.74s | — | 7.21s | 8.19s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 5 | 2 | N/A |
| **Internal Tool Cycles** | 6 | 13 | <span class="benchmark-delta-negative">+116.7%</span> | 17 | 13 | N/A |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | 5 | 2 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 6 | 13 | <span class="benchmark-delta-negative">+116.7%</span> | 17 | 13 | N/A |
| **Output Tokens** | 3097 | 3291 | <span class="benchmark-delta-negative">+6.3%</span> | 30789 | 12091 | N/A |
| **Reasoning / Thinking Tokens** | 1096 | 1417 | <span class="benchmark-delta-negative">+29.3%</span> | 12809 | 5786 | N/A |
| **Total Input Tokens** | 199735 | 410335 | <span class="benchmark-delta-negative">+105.4%</span> | 2783452 | 1043054 | N/A |
| **Cached Input Tokens** | 177920 | 376320 | <span class="benchmark-delta-positive">+111.5%</span> | 2612224 | 963328 | N/A |
| **Uncached Input Tokens** | 21815 | 34015 | <span class="benchmark-delta-negative">+55.9%</span> | 171228 | 79726 | N/A |
| **Cached vs Uncached Token Ratio** | 8.16:1 | 11.06:1 | <span class="benchmark-delta-positive">+35.6%</span> | 15.26:1 | 12.08:1 | N/A |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 13 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*test*&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | head -80 &amp;&amp; \
printf &#39;\\n--- docs ---\\n&#39; &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; -g &#39;Makefile&#39; -g &#39;AGENTS*&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*AGENTS.override.md&#39;&#34;&#39; | sed -n &#39;1,120p&#39; &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
printf &#39;\\n--- normalize ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
printf &#39;\\n--- sink ---\\n&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/sink.go &amp;&amp; \
printf &#39;\\n--- dispatch ---\\n&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
printf &#39;\\n--- summary ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/summary.go &amp;&amp; \
printf &#39;\\n--- docs ---\\n&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; docs/audit-sinks.md &amp;&amp; \
printf &#39;\\n--- tests ---\\n&#39; &amp;&amp; \
rg --files -g &#39;*_test.go&#39; -g &#39;*.txtar&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- docs/audit-sinks.md ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit files ---&#39; &amp;&amp; \
for f in audit/*.go service/dispatch.go metrics/writer.go legacy/export.go cmd/migrate/main.go; do echo \&#34;--- &#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; audit/event_test.go &amp;&amp; \
printf &#39;\\n--- summary tests ---\\n&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary_test.go &amp;&amp; \
printf &#39;\\n--- dispatch tests ---\\n&#39; &amp;&amp; \
sed -n &#39;1,320p&#39; service/dispatch_test.go &amp;&amp; \
printf &#39;\\n--- other ---\\n&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,260p&#39; legacy/export.go &amp;&amp; \
printf &#39;\\n--- git status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- tests ---&#39; &amp;&amp; \
for f in service/dispatch_test.go audit/*_test.go metrics/writer_test.go legacy/export_test.go; do echo \&#34;--- &#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,280p&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; &#39;--- go.mod ---&#39; &amp;&amp; \
sed -n &#39;1,80p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- relevant symbols ---&#39; &amp;&amp; \
rg -n \&#34;Record|DispatchAll|Classify|probe|health|normalize|Write\\(\&#34; .&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
git diff -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
if [ -f Makefile ]; then make check; else echo &#39;make check unavailable: no Makefile in fixture&#39;; fi &amp;&amp; \
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- scoped status ---&#39; &amp;&amp; \
git status --short -- . &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- scoped diff check ---&#39; &amp;&amp; \
git diff --check -- . &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected diffs ---&#39; &amp;&amp; \
git diff --quiet -- go.mod go.sum &amp;&amp; \
echo &#39;go.mod and go.sum unchanged&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- test diffs ---&#39; &amp;&amp; \
if git diff --quiet -- &#39;*_test.go&#39;; then echo &#39;tests unchanged&#39;; else git diff --name-only -- &#39;*_test.go&#39;; exit 1; fi &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- production diff ---&#39; &amp;&amp; \
git diff -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/normalize.go&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tracked fixture files ---&#39; &amp;&amp; \
git ls-files -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- resulting code ---&#39; &amp;&amp; \
sed -n &#39;1,160p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,140p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,140p&#39; service/dispatch.go&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent.Kind = audit.NormalizeKind(event.Kind)\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
printf &#39;\\n--- protected diff check ---\\n&#39; &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
printf &#39;\\n--- changed files ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel &amp;&amp; \
git ls-files --error-unmatch audit/normalize.go service/dispatch.go audit/summary.go &amp;&amp; \
git status --short --untracked-files=all -- audit service &amp;&amp; \
git diff --no-ext-diff --stat -- audit service&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;set +e
make check
make_status=$?
printf &#39;&#34;&#39;make_check_exit=%s\\n&#39; \&#34;&#34;&#39;$make_status&#34;
go test ./...
test_status=$?
printf &#39;&#34;&#39;go_test_exit=%s\\n&#39; \&#34;&#34;&#39;$test_status&#34;
gofmt -d audit service metrics legacy cmd
gofmt_status=$?
printf &#39;&#34;&#39;gofmt_check_exit=%s\\n&#39; \&#34;&#34;&#39;$gofmt_status&#34;
exit $(( make_status || \
test_status || \
gofmt_status ))&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- final audit/service sources ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,140p&#39; service/dispatch.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected files present ---&#39; &amp;&amp; \
ls -l go.mod go.sum 2&gt;&amp;1 &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- fixture file mtimes/status ---&#39; &amp;&amp; \
git status --short -- audit service metrics legacy cmd 2&gt;&amp;1&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 9 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 9 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sed -n &#39;1,120p&#39; &amp;&amp; \
printf &#39;\\n--- docs ---\\n&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; PROJECT_BRIEF.md 2&gt;/dev/null || \
true &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | sed -n &#39;1,160p&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- event.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- normalize.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- sink.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/sink.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- fanout.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/fanout.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- transaction.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/transaction.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- docs ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; docs/audit-sinks.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/transaction.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Event|Sink|Write\\(|probe|Probe|kind|audit\&#34; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,220p&#39; service/replay.go &amp;&amp; \
sed -n &#39;1,240p&#39; api/router.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/fanout.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in audit/summary.go metrics/writer.go legacy/export.go journal/file.go audit/*_test.go service/dispatch.go service/replay.go; do echo --- $f; sed -n &#39;&#34;&#39;1,240p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;audit|probe|health|Kind|Write\\(\&#34; --glob &#39;*.go&#39; --glob &#39;*.txtar&#39; --glob &#39;*.md&#39; .&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n -C 4 \&#34;mixed-sink|normalize|probe|delivery boundary|control traffic\&#34; . --glob &#39;&#34;&#39;!*.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,180p&#39; journal/file.go &amp;&amp; \
sed -n &#39;1,180p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,180p&#39; testdata/bench/task_11_mixed_sink_api_migration.txtar &amp;&amp; \
rg --files -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39; -g &#39;*.sh&#39; .&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
git diff -- audit journal service &amp;&amp; \
make check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- audit journal service &amp;&amp; \
rg --files -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39; -g &#39;*.sh&#39; .; gofmt -w audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go; git diff --check&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/normalize.go; sed -n &#39;1,100p&#39; audit/transaction.go; git status --short -- audit journal service; git diff --stat -- audit journal service&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; audit/normalize.go; sed -n &#39;1,100p&#39; audit/sink.go; git status --short -- audit service journal; git diff --stat -- audit service journal; git diff -- go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel; git ls-files -v -- audit/normalize.go audit/sink.go audit/transaction.go journal/file.go service/dispatch.go; git diff --no-index /dev/null audit/normalize.go | sed -n &#39;1,80p&#39;; git status --short --untracked-files=all | rg &#39;audit|journal|service&#39; || \
true; rg -n \&#34;NormalizeEvent|case \\\&#34;health\\\&#34;\&#34; audit journal service&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-files --stage audit/normalize.go audit/sink.go journal/file.go service/dispatch.go; git diff --no-ext-diff --no-index /dev/null audit/normalize.go | sed -n &#39;1,80p&#39;; git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/summary.go &amp;&amp; \
test -z &#34;$(gofmt -d audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go)&#34; &amp;&amp; \
go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go vet ./... &amp;&amp; \
git diff --no-ext-diff -- go.mod go.sum &amp;&amp; \
git status --short --ignored -- audit service journal metrics legacy go.mod go.sum | sed -n &#39;1,120p&#39; &amp;&amp; \
rg -n \&#34;NormalizeEvent|probe|strings\&#34; audit service journal metrics legacy&#34;</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- audit boundary ---&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,160p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,160p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,160p&#39; audit/transaction.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- classification ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected status ---&#39; &amp;&amp; \
git status --short -- go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- files ---&#39; &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; audit/normalize.go; sed -n &#39;1,160p&#39; audit/sink.go; sed -n &#39;1,160p&#39; audit/summary.go; sed -n &#39;1,140p&#39; audit/fanout.go; sed -n &#39;1,140p&#39; audit/transaction.go; sed -n &#39;1,140p&#39; service/dispatch.go; sed -n &#39;1,120p&#39; journal/file.go; sha256sum go.mod go.sum 2&gt;/dev/null || \
true; git diff --no-ext-diff -- go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
test -z &#34;$(gofmt -d audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go)&#34; &amp;&amp; \
go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Normalize|normaliz|delivery boundary|persistence|Sink|probe|health\&#34; . --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,160p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/event.go &amp;&amp; \
git status --short -- go.mod go.sum &amp;&amp; \
go test ./audit ./service ./journal&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go &amp;&amp; \
git diff --check &amp;&amp; \
go test ./... &amp;&amp; \
go vet ./... &amp;&amp; \
sha256sum go.mod go.sum 2&gt;/dev/null || \
true&#39;</code></pre></td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td>—</td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,160p&#39; service/replay.go &amp;&amp; \
git status --short -- go.mod go.sum &amp;&amp; \
go test ./service ./audit&#34;</code></pre></td><td>—</td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td>—</td></tr>
<tr><td>17</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
test -z &#34;$(gofmt -d audit/normalize.go audit/summary.go service/dispatch.go)&#34; &amp;&amp; \
git status --short -- go.mod go.sum &amp;&amp; \
go test ./...&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not call a `semantic_*` tool because none was exposed as an available callable semedit MCP tool in the session. I used ordinary repository editing instead.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 4.46s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because it was discovered but appeared unnecessary for the small, independent edits, so I used the dedicated semantic operations directly. Batching would have been suitable but was simply not chosen.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 5.71s; turns: 1; tool calls: 0.</p>
</details>
