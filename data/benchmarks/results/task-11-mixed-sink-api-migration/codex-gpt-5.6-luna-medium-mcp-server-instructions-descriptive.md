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

* **Date**: 2026-09-22 12:02:22 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-5.6-luna/medium`

#### Configuration: default prompt · descriptive MCP instructions

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
| **Wall-Clock Latency** | 202.53s | 106.79s | <span class="benchmark-delta-positive">-47.3%</span> | 300.01s | 123.89s | N/A |
| **Process Start → First Event** | 0.27s | 0.20s | — | 0.14s | 0.50s | — |
| **First Event → First Tool Call** | 14.75s | 8.20s | — | 14.23s | 7.59s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | N/A |
| **Internal Tool Cycles** | 12 | 14 | <span class="benchmark-delta-negative">+16.7%</span> | 9 | 9 | N/A |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | 2 | 2 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 12 | 14 | <span class="benchmark-delta-negative">+16.7%</span> | 9 | 9 | N/A |
| **Output Tokens** | 4296 | 3161 | <span class="benchmark-delta-positive">-26.4%</span> | 4797 | 4922 | N/A |
| **Reasoning / Thinking Tokens** | 1505 | 1169 | <span class="benchmark-delta-positive">-22.3%</span> | 1832 | 2245 | N/A |
| **Total Input Tokens** | 337211 | 398188 | <span class="benchmark-delta-negative">+18.1%</span> | 324099 | 267622 | N/A |
| **Cached Input Tokens** | 297472 | 363264 | <span class="benchmark-delta-positive">+22.1%</span> | 298752 | 248064 | N/A |
| **Uncached Input Tokens** | 39739 | 34924 | <span class="benchmark-delta-positive">-12.1%</span> | 25347 | 19558 | N/A |
| **Cached vs Uncached Token Ratio** | 7.49:1 | 10.40:1 | <span class="benchmark-delta-positive">+39.0%</span> | 11.79:1 | 12.68:1 | N/A |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ❌ FAIL | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ❌ FAIL | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ❌ FAIL | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 13 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; | sort | sed -n &#39;1,160p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; | sed -n &#39;1,120p&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/event.go&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/normalize.go&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/sink.go&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- service/dispatch.go&#39; &amp;&amp; \
sed -n &#39;1,280p&#39; service/dispatch.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- cmd/migrate/main.go&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; cmd/migrate/main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- Makefile&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; Makefile 2&gt;/dev/null || \
true&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,220p&#39; docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,220p&#39; audit/summary.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- audit tests&#39; &amp;&amp; \
sed -n &#39;1,280p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/summary_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- service tests&#39; &amp;&amp; \
sed -n &#39;1,320p&#39; service/dispatch_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- metrics&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/writer.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- legacy&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; legacy/export.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- module/docs&#39; &amp;&amp; \
sed -n &#39;1,160p&#39; go.mod &amp;&amp; \
sed -n &#39;1,220p&#39; AGENTS.override.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; Makefile 2&gt;/dev/null || \
true; printf &#39;\\n-- files --\\n&#39;; rg -n \&#34;Audit|audit|Probe|probe|kind|Kind|Persist|persist|Deliver|deliver\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39;&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;normalize|probe|health|audit|delivery|sink|Classify|Kind\&#34; . --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!*.json&#39;&#34;&#39; --glob &#39;&#34;&#39;!*.md&#39;&#34;&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- benchmark prompt&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; testdata/bench/task_11_mixed_sink_api_migration.txtar&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find . -maxdepth 3 -type f -print | sort &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- git diff fixture&#39; &amp;&amp; \
git diff -- audit service metrics legacy cmd/migrate go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; service/dispatch_test.go; sed -n &#39;1,160p&#39; audit/summary_test.go; sed -n &#39;1,160p&#39; audit/event_test.go; sed -n &#39;1,160p&#39; metrics/writer.go; sed -n &#39;1,160p&#39; legacy/export.go; sed -n &#39;1,120p&#39; go.mod; find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; docs/audit-sinks.md &amp;&amp; \
git log -5 --oneline --all -- . &#39;:(exclude)AGENTS.override.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
git diff --check &amp;&amp; \
make check&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check -- audit service metrics legacy cmd/migrate go.mod go.sum &amp;&amp; \
make check&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
git status --short -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go audit/event_test.go service/dispatch_test.go metrics/writer.go legacy/export.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent.Kind = audit.NormalizeKind(event.Kind)\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,100p&#39; service/dispatch.go &amp;&amp; \
git status --short &amp;&amp; \
git ls-files -v audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;test -z &#34;$(gofmt -l audit/*.go service/*.go metrics/*.go legacy/*.go cmd/migrate/*.go)&#34; &amp;&amp; \
go vet ./... &amp;&amp; \
test ! -e go.sum &amp;&amp; \
git diff --no-index -- /dev/null /dev/null &gt;/dev/null 2&gt;&amp;1 || \
true
printf &#39;&#34;&#39;%s\\n&#39; &#39;--- protected files&#39; &amp;&amp; \
stat -f &#39;%N %z bytes %Sm&#39; go.mod 2&gt;/dev/null &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- implementation&#39; &amp;&amp; \
git diff --no-index /dev/null audit/normalize.go || \
true &amp;&amp; \
git diff --no-index /dev/null audit/sink.go || \
true &amp;&amp; \
git diff --no-index /dev/null audit/summary.go || \
true &amp;&amp; \
git diff --no-index /dev/null service/dispatch.go || \
true&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit service go.mod go.sum &amp;&amp; \
git status --short -- audit service go.mod go.sum &amp;&amp; \
gofmt -d audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;set -e
test -z &#34;$(gofmt -l audit/*.go service/*.go metrics/*.go legacy/*.go cmd/migrate/*.go)&#34;
go vet ./...
test ! -e go.sum
printf &#39;&#34;&#39;%s\\n&#39; &#39;verification passed&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/normalize.go; sed -n &#39;1,120p&#39; audit/summary.go; sed -n &#39;1,140p&#39; service/dispatch.go; git status --short --untracked-files=all | sed -n &#39;1,80p&#39;; git diff --stat -- audit service go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel; git ls-files --error-unmatch audit/normalize.go audit/summary.go service/dispatch.go 2&gt;&amp;1; git check-ignore -v audit/normalize.go audit/summary.go service/dispatch.go || \
true; git diff --no-index /dev/null audit/normalize.go &gt;/dev/null; printf &#39;protected hashes\\n&#39;; shasum go.mod go.sum 2&gt;/dev/null || \
true; printf &#39;changed fixture files\\n&#39;; find audit service -type f -maxdepth 2 -print&#34;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;,
  &#34;trust_workspace&#34;: false
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 9 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | sed -n &#39;1,160p&#39; &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | sed -n &#39;1,160p&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- audit/event.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/normalize.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/sink.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/fanout.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/fanout.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/transaction.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/transaction.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- docs/audit-sinks.md ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; docs/audit-sinks.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
printf &#39;\\n--- audit files ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/transaction.go &amp;&amp; \
printf &#39;\\n--- models ---\\n&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; models/models.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- service/dispatch.go ---&#39; &amp;&amp; \
sed -n &#39;1,280p&#39; service/dispatch.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- service/replay.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; service/replay.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit tests ---&#39; &amp;&amp; \
for f in audit/*_test.go; do echo \&#34;--- &#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,280p&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; &#39;--- other sink uses ---&#39; &amp;&amp; \
rg -n \&#34;audit|Event\\{|\\.Write\\(|Probe|probe|kind\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!audit/*_test.go&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;audit|Event|Write\\(|Probe|probe|metrics|legacy\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; . &amp;&amp; \
printf &#39;\\n--- tests for behavior clues ---\\n&#39; &amp;&amp; \
sed -n &#39;1,280p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,280p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,240p&#39; api/router.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in audit/summary.go journal/file.go metrics/writer.go legacy/export.go service/dispatch_test.go db/db.go; do echo &#34;--- $f&#34;; sed -n &#39;&#34;&#39;1,220p&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; &#39;--- go.mod ---&#39; &amp;&amp; \
sed -n &#39;1,100p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- Makefile? ---&#39; &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; else echo none; fi &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- all files ---&#39; &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,180p&#39; journal/file.go &amp;&amp; \
sed -n &#39;1,180p&#39; service/replay.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/fanout.go &amp;&amp; \
git status --short &amp;&amp; \
git log -1 --oneline&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
git diff -- audit service journal metrics legacy go.mod go.sum &amp;&amp; \
printf &#39;\\n--- protected status ---\\n&#39; &amp;&amp; \
git status --short -- go.mod go.sum &#39;*_test.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;NewDispatcher|DispatchAll|Replay|\\.Write\\(.*Event|audit\\.Sink|FanoutSink|BufferedSink|Classify\\(\&#34; --glob &#39;*.go&#39; . &amp;&amp; \
printf &#39;\\n--- module/tooling ---\\n&#39; &amp;&amp; \
sed -n &#39;1,160p&#39; go.mod &amp;&amp; \
sed -n &#39;1,220p&#39; Makefile 2&gt;/dev/null || \
true &amp;&amp; \
printf &#39;\\n--- hidden-ish fixture files ---\\n&#39; &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*.go&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,100p&#39; &amp;&amp; \
git diff --no-ext-diff -- audit/normalize.go &amp;&amp; \
sed -n &#39;1,80p&#39; audit/normalize.go &amp;&amp; \
git ls-files --stage audit/normalize.go&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
make check&#39;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel &amp;&amp; \
git status --short -- audit service go.mod go.sum &amp;&amp; \
sed -n &#39;1,120p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,160p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,100p&#39; service/replay.go &amp;&amp; \
go test ./...&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- .scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790071063778703000 &amp;&amp; \
git diff -- .scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790071063778703000/audit/normalize.go .scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790071063778703000/audit/summary.go .scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790071063778703000/service/dispatch.go .scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790071063778703000/service/replay.go &amp;&amp; \
git diff --quiet -- .scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790071063778703000/go.mod .scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790071063778703000/go.sum; printf &#39;protected_diff_exit=%s\\n&#39; \&#34;&#34;&#39;$?&#34;&#39;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go vet ./... &amp;&amp; \
printf &#39;\\n--- fixture mtimes/status ---\\n&#39; &amp;&amp; \
stat -f &#39;%N %Sm&#39; -t &#39;%Y-%m-%d %H:%M:%S&#39; go.mod go.sum audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
printf &#39;\\n--- test files in fixture ---\\n&#39; &amp;&amp; \
find . -type f -name &#39;*_test.go&#39; -print | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;test -z &#34;$(gofmt -d audit/normalize.go audit/summary.go service/dispatch.go service/replay.go)&#34; &amp;&amp; \
go vet ./... &amp;&amp; \
git diff --check -- .scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790071063778703000&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not discover any callable `semantic_*` tools from the available semedit MCP server in the tool context. I therefore used ordinary editing for this small production-only change.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 4.70s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because it was not necessary for the small, independent edits, and I used the dedicated semantic operations directly. Batching was available but unsuitable here because each operation targeted a separate existing function or file and the dedicated calls provided immediate per-edit verification.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.63s; turns: 1; tool calls: 0.</p>
</details>
