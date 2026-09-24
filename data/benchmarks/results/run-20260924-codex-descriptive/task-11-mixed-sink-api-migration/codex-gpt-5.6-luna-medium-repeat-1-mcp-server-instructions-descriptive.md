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

* **Date**: 2026-09-24 16:41:34 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-5.6-luna/medium (repeat 1)`

#### Configuration: default prompt · descriptive MCP instructions

* **Run Provenance**: `binary=44027cf7b57a0b4e`

* **Fixture**: [testdata/bench/task_11_mixed_sink_api_migration.txtar](https://github.com/spockz/semantic-editor/blob/d48187b57f4c6ee3e1c7b415974bedde8ba9d43b/testdata/bench/task_11_mixed_sink_api_migration.txtar)

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
| **Wall-Clock Latency** | 261.42s | 125.85s | <span class="benchmark-delta-positive">-51.9%</span> | 105.11s | 120.52s | N/A |
| **Process Start → First Event** | 0.14s | 0.12s | — | 0.13s | 0.12s | — |
| **First Event → First Tool Call** | 8.01s | 14.86s | — | 7.02s | 8.23s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 5 | 1 | <span class="benchmark-delta-positive">-80.0%</span> | 1 | 1 | N/A |
| **Internal Tool Cycles** | 15 | 13 | <span class="benchmark-delta-positive">-13.3%</span> | 10 | 9 | N/A |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | 2 | 2 | N/A |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 15 | 13 | <span class="benchmark-delta-positive">-13.3%</span> | 10 | 9 | N/A |
| **Output Tokens** | 27621 | 3337 | <span class="benchmark-delta-positive">-87.9%</span> | 4419 | 4792 | N/A |
| **Reasoning / Thinking Tokens** | 12899 | 1218 | <span class="benchmark-delta-positive">-90.6%</span> | 1906 | 2175 | N/A |
| **Total Input Tokens** | 2212802 | 419083 | <span class="benchmark-delta-positive">-81.1%</span> | 298316 | 391104 | N/A |
| **Cached Input Tokens** | 1925888 | 365824 | <span class="benchmark-delta-negative">-81.0%</span> | 272384 | 347904 | N/A |
| **Uncached Input Tokens** | 286914 | 53259 | <span class="benchmark-delta-positive">-81.4%</span> | 25932 | 43200 | N/A |
| **Cached vs Uncached Token Ratio** | 6.71:1 | 6.87:1 | <span class="benchmark-delta-positive">+2.3%</span> | 10.50:1 | 8.05:1 | N/A |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 17 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | sort | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,220p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- files ---&#39; &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/event.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/normalize.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/sink.go ---&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- service/dispatch.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,280p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer_test.go &amp;&amp; \
git status --short &amp;&amp; \
git diff -- . &#39;:(exclude)AGENTS.override.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- docs/audit-sinks.md ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; docs/audit-sinks.md &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/summary.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- metrics/writer.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/writer.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- legacy/export.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; legacy/export.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- cmd/migrate/main.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; cmd/migrate/main.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- go.mod ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;probe|Normalize|normalize|Write\\(|Record\\(|DispatchAll|Classify\&#34; . -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- audit/event_test.go ---&#39; &amp;&amp; \
sed -n &#39;1,280p&#39; audit/event_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit/summary_test.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- service/dispatch_test.go ---&#39; &amp;&amp; \
sed -n &#39;1,320p&#39; service/dispatch_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- git status ---&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
sed -n &#39;1,120p&#39; cmd/migrate/main.go &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
make check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,160p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
go vet ./... &amp;&amp; \
git diff --check -- audit service &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
git status --short -- audit service go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel &amp;&amp; \
git status --short --untracked-files=all -- audit service go.mod go.sum &amp;&amp; \
sed -n &#39;1,160p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,120p&#39; service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go audit/sink.go audit/summary.go go.mod go.sum &amp;&amp; \
git diff --check &amp;&amp; \
make check&#39;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,180p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,160p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,220p&#39; service/dispatch.go &amp;&amp; \
git status --short --untracked-files=all -- audit service go.mod go.sum &amp;&amp; \
git diff -- go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- audit go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- normalize ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; audit/normalize.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- sink ---&#39; &amp;&amp; \
sed -n &#39;1,180p&#39; audit/sink.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- summary ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- make targets ---&#39; &amp;&amp; \
(make -qp 2&gt;/dev/null | rg &#39;&#34;&#39;^[A-Za-z0-9_.-]+:&#39;&#34;&#39; | head -40 || \
true)&#34;</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-files -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go go.mod go.sum; find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
go vet ./... &amp;&amp; \
git diff --check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
go vet ./... &amp;&amp; \
git diff --check &amp;&amp; \
cmp -s go.mod go.mod &amp;&amp; \
test ! -e go.sum&#39;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,140p&#39; audit/normalize.go &amp;&amp; \
go test ./audit ./service ./metrics ./legacy &amp;&amp; \
git diff -- go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- tracked state ---&#39; &amp;&amp; \
git ls-files --stage -- audit/normalize.go audit/sink.go audit/summary.go audit/event_test.go service/dispatch_test.go go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- ignored state ---&#39; &amp;&amp; \
git check-ignore -v audit/normalize.go audit/sink.go audit/summary.go audit/event_test.go service/dispatch_test.go go.mod 2&gt;/dev/null || \
true &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- hashes ---&#39; &amp;&amp; \
shasum audit/event_test.go service/dispatch_test.go metrics/writer.go legacy/export.go go.mod&#34;</code></pre></td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/normalize.go &amp;&amp; \
go test ./service ./audit &amp;&amp; \
git diff -- go.mod go.sum&#34;</code></pre></td><td>—</td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 13 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- git status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; docs/audit-sinks.md
sed -n &#39;1,240p&#39; audit/event.go
sed -n &#39;1,240p&#39; audit/normalize.go
sed -n &#39;1,260p&#39; audit/fanout.go
sed -n &#39;1,220p&#39; audit/sink.go
sed -n &#39;1,220p&#39; audit/transaction.go
sed -n &#39;1,220p&#39; audit/summary.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/transaction.go &amp;&amp; \
sed -n &#39;1,240p&#39; models/models.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;audit|Event|Classify|Write\\(\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/replay.go &amp;&amp; \
sed -n &#39;1,240p&#39; api/router.go &amp;&amp; \
sed -n &#39;1,240p&#39; db/db.go &amp;&amp; \
sed -n &#39;1,220p&#39; journal/file.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export.go &amp;&amp; \
rg -n \&#34;audit|Event|Kind|Write\\(\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; .&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; journal/file.go
sed -n &#39;1,180p&#39; service/dispatch.go
sed -n &#39;1,160p&#39; service/replay.go
sed -n &#39;1,160p&#39; metrics/writer.go
sed -n &#39;1,160p&#39; legacy/export.go
sed -n &#39;1,220p&#39; audit/*_test.go
sed -n &#39;1,180p&#39; service/dispatch_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/transaction.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/fanout.go &amp;&amp; \
git status --short &amp;&amp; \
git log -5 --oneline&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- audit service journal &amp;&amp; \
git diff --name-only -- audit service journal go.mod go.sum &#39;*_test.go&#39;&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go
git diff --check
git status --short -- audit journal service go.mod go.sum

git diff -- audit journal service&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short --untracked-files=all -- audit service journal go.mod go.sum &amp;&amp; \
sed -n &#39;1,120p&#39; audit/normalize.go &amp;&amp; \
git rev-parse --show-toplevel &amp;&amp; \
git ls-files -- audit/normalize.go&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/normalize.go
sed -n &#39;1,100p&#39; audit/transaction.go
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;ls -la &amp;&amp; \
gofmt -w audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
make check&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git ls-files --stage audit/normalize.go audit/sink.go service/dispatch.go journal/file.go
git check-ignore -v audit/normalize.go
git diff --no-ext-diff --stat -- .
git rev-parse --show-toplevel&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go vet ./... &amp;&amp; \
printf &#39;protected files:\\n&#39; &amp;&amp; \
stat -f &#39;%N %z bytes&#39; go.mod go.sum 2&gt;/dev/null || \
true &amp;&amp; \
printf &#39;test files changed by this work:\\n&#39; &amp;&amp; \
find . -type f -name &#39;*_test.go&#39; -newer audit/normalize.go -print &amp;&amp; \
printf &#39;audit implementation:\\n&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/summary.go &amp;&amp; \
printf &#39;workspace files:\\n&#39; &amp;&amp; \
find . -type f -not -path &#39;./.scratch/*&#39; | sort&#34;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go
go vet ./...
rg -n &#34;Normalize|probe|Classify|metrics|legacy&#34; audit journal service metrics legacy&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not call a semantic_* tool because no applicable semedit MCP operation was exposed in the available tools. Ordinary editing was sufficient for this small, localized change.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.45s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I used consecutive semantic operations because `semantic_batch` was available but its edit schema was not clearly documented for these function insertion/body-replacement calls. Batching was therefore unsuitable under the available interface, so I chose the individually validated operations.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.03s; turns: 1; tool calls: 0.</p>
</details>

#### Configuration: prefer_discover_semedit prompt · descriptive MCP instructions

* **Run Provenance**: `binary=44027cf7b57a0b4e`

* **Fixture**: [testdata/bench/task_11_mixed_sink_api_migration.txtar](https://github.com/spockz/semantic-editor/blob/d48187b57f4c6ee3e1c7b415974bedde8ba9d43b/testdata/bench/task_11_mixed_sink_api_migration.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. Audit event kinds may contain inconsistent whitespace and casing. Normalize them at the delivery boundary before persistence, and treat probe events as control traffic. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Prefer using semantic editor operations if applicable. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 119.54s | 115.93s | <span class="benchmark-delta-positive">-3.0%</span> | 152.82s | 179.01s | <span class="benchmark-delta-negative">+17.1%</span> |
| **Process Start → First Event** | 0.12s | 0.13s | — | 0.13s | 0.12s | — |
| **First Event → First Tool Call** | 18.52s | 14.47s | — | 17.55s | 14.66s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 7 | 13 | <span class="benchmark-delta-negative">+85.7%</span> | 9 | 20 | <span class="benchmark-delta-negative">+122.2%</span> |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 7 | 13 | <span class="benchmark-delta-negative">+85.7%</span> | 9 | 20 | <span class="benchmark-delta-negative">+122.2%</span> |
| **Output Tokens** | 4162 | 3651 | <span class="benchmark-delta-positive">-12.3%</span> | 5590 | 4390 | <span class="benchmark-delta-positive">-21.5%</span> |
| **Reasoning / Thinking Tokens** | 1651 | 1509 | <span class="benchmark-delta-positive">-8.6%</span> | 2007 | 1757 | <span class="benchmark-delta-positive">-12.5%</span> |
| **Total Input Tokens** | 390932 | 552860 | <span class="benchmark-delta-negative">+41.4%</span> | 510070 | 893233 | <span class="benchmark-delta-negative">+75.1%</span> |
| **Cached Input Tokens** | 351232 | 495616 | <span class="benchmark-delta-positive">+41.1%</span> | 462080 | 848640 | <span class="benchmark-delta-positive">+83.7%</span> |
| **Uncached Input Tokens** | 39700 | 57244 | <span class="benchmark-delta-negative">+44.2%</span> | 47990 | 44593 | <span class="benchmark-delta-positive">-7.1%</span> |
| **Cached vs Uncached Token Ratio** | 8.85:1 | 8.66:1 | <span class="benchmark-delta-negative">-2.1%</span> | 9.63:1 | 19.03:1 | <span class="benchmark-delta-positive">+97.6%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 13 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort; printf &#39;\\n--- status ---\\n&#39;; git status --short; printf &#39;\\n--- top-level ---\\n&#39;; rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sed -n &#39;1,120p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; printf &#39;%s\\n&#39; &#39;--- files ---&#39;; rg --files -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; | sort; printf &#39;%s\\n&#39; &#39;--- brief ---&#39;; if test -f PROJECT_BRIEF.md; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi; printf &#39;%s\\n&#39; &#39;--- status ---&#39;; git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- audit/event.go ---&#39;; sed -n &#39;1,240p&#39; audit/event.go; printf &#39;%s\\n&#39; &#39;--- audit/normalize.go ---&#39;; sed -n &#39;1,240p&#39; audit/normalize.go; printf &#39;%s\\n&#39; &#39;--- audit/sink.go ---&#39;; sed -n &#39;1,240p&#39; audit/sink.go; printf &#39;%s\\n&#39; &#39;--- audit/summary.go ---&#39;; sed -n &#39;1,260p&#39; audit/summary.go; printf &#39;%s\\n&#39; &#39;--- service/dispatch.go ---&#39;; sed -n &#39;1,280p&#39; service/dispatch.go; printf &#39;%s\\n&#39; &#39;--- docs/audit-sinks.md ---&#39;; sed -n &#39;1,260p&#39; docs/audit-sinks.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- audit/event.go ---&#39;; sed -n &#39;1,240p&#39; audit/event.go; printf &#39;%s\\n&#39; &#39;--- audit/normalize.go ---&#39;; sed -n &#39;1,240p&#39; audit/normalize.go; printf &#39;%s\\n&#39; &#39;--- audit/sink.go ---&#39;; sed -n &#39;1,280p&#39; audit/sink.go; printf &#39;%s\\n&#39; &#39;--- audit/summary.go ---&#39;; sed -n &#39;1,260p&#39; audit/summary.go; printf &#39;%s\\n&#39; &#39;--- service/dispatch.go ---&#39;; sed -n &#39;1,300p&#39; service/dispatch.go; printf &#39;%s\\n&#39; &#39;--- docs ---&#39;; sed -n &#39;1,240p&#39; docs/audit-sinks.md; printf &#39;%s\\n&#39; &#39;--- cmd/migrate ---&#39;; sed -n &#39;1,220p&#39; cmd/migrate/main.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- audit tests ---&#39;; sed -n &#39;1,280p&#39; audit/event_test.go; sed -n &#39;1,260p&#39; audit/summary_test.go; printf &#39;%s\\n&#39; &#39;--- service tests ---&#39;; sed -n &#39;1,320p&#39; service/dispatch_test.go; printf &#39;%s\\n&#39; &#39;--- metrics ---&#39;; sed -n &#39;1,260p&#39; metrics/writer.go; sed -n &#39;1,280p&#39; metrics/writer_test.go; printf &#39;%s\\n&#39; &#39;--- legacy ---&#39;; sed -n &#39;1,260p&#39; legacy/export.go; sed -n &#39;1,260p&#39; legacy/export_test.go; printf &#39;%s\\n&#39; &#39;--- cmd ---&#39;; sed -n &#39;1,260p&#39; cmd/migrate/main.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- tests and protocols ---&#39;; sed -n &#39;1,260p&#39; audit/event_test.go; sed -n &#39;1,260p&#39; audit/summary_test.go; sed -n &#39;1,300p&#39; service/dispatch_test.go; sed -n &#39;1,240p&#39; metrics/writer.go; sed -n &#39;1,220p&#39; legacy/export.go; sed -n &#39;1,260p&#39; legacy/export_test.go; sed -n &#39;1,220p&#39; metrics/writer_test.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;probe|health|normalize|Normalize|Record|DispatchAll|Transactional|delivery|sink\&#34; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; .; printf &#39;\\n--- make/check candidates ---\\n&#39;; rg --files -g &#39;Makefile&#39; -g &#39;scripts/**&#39; -g &#39;*.sh&#39; -g &#39;README*&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;probe|normalize|Classify|Dispatch|Sink|audit\&#34; . --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go
git diff --check
go test ./...
printf &#39;\\n--- diff ---\\n&#39;
git diff -- audit/normalize.go audit/summary.go service/dispatch.go
printf &#39;\\n--- protected/status check ---\\n&#39;
git status --short -- go.mod go.sum audit service metrics legacy cmd
&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- files ---&#39;; sed -n &#39;1,120p&#39; audit/normalize.go; sed -n &#39;1,120p&#39; audit/summary.go; sed -n &#39;1,120p&#39; service/dispatch.go; printf &#39;%s\\n&#39; &#39;--- git identity ---&#39;; git rev-parse --show-toplevel; git ls-files --error-unmatch audit/normalize.go 2&gt;&amp;1 || \
true; printf &#39;%s\\n&#39; &#39;--- ignored? ---&#39;; git check-ignore -v audit/normalize.go audit/summary.go service/dispatch.go || \
true; printf &#39;%s\\n&#39; &#39;--- all status ---&#39;; git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go
gofmt -d audit/normalize.go audit/summary.go service/dispatch.go
git diff --check
go test ./...
go vet ./...
printf &#39;\\n--- resulting source ---\\n&#39;
sed -n &#39;1,120p&#39; audit/normalize.go
sed -n &#39;1,100p&#39; audit/summary.go
sed -n &#39;1,140p&#39; service/dispatch.go
printf &#39;\\n--- protected files present and untouched by workflow ---\\n&#39;
stat -f &#39;%N %z bytes&#39; go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tevent.Kind = audit.NormalizeKind(event.Kind)\n\t\tif err := writeEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go; printf &#39;%s\\n&#39; &#39;--- protected diff check ---&#39;; git diff -- go.mod go.sum; printf &#39;%s\\n&#39; &#39;--- fixture status ---&#39;; git status --short -- audit service metrics legacy cmd go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/normalize.go; sed -n &#39;14,35p&#39; audit/sink.go; git rev-parse --show-toplevel; git status --short --untracked-files=all | head -80; git diff --no-ext-diff --no-color -- audit/normalize.go&#34;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;ls -la; go env GOMOD; go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 13 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sort; printf &#39;\\n--- PROJECT_BRIEF ---\\n&#39;; if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi; printf &#39;\\n--- git status ---\\n&#39;; git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- files ---&#39; &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- git status ---&#39; &amp;&amp; \
git status --short &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- project brief ---&#39; &amp;&amp; \
if test -f PROJECT_BRIEF.md; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in audit/event.go audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go api/router.go service/dispatch.go service/replay.go journal/file.go logging/logging.go metrics/writer.go legacy/export.go cmd/migrate/main.go cmd/replay/main.go docs/audit-sinks.md; do printf &#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- audit sources ---&#39; &amp;&amp; \
for f in audit/event.go audit/normalize.go audit/fanout.go audit/sink.go audit/summary.go audit/transaction.go docs/audit-sinks.md service/dispatch.go api/router.go; do echo \&#34;### &#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in audit/*_test.go service/*_test.go journal/*_test.go metrics/*_test.go legacy/*_test.go; do [ -f &#34;$f&#34; ] || \
continue; printf &#39;&#34;&#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,300p&#39; \&#34;&#34;&#39;$f&#34;; done; printf &#39;&#34;&#39;\\n--- references ---\\n&#39;; rg -n \&#34;probe|health|Kind|Sink|Write\\(|Normalize|Classify|Fanout|Buffered|Journal\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Event|Kind|health|probe|metrics|legacy|Write\\(\&#34; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; .&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in audit/*_test.go service/*_test.go journal/*_test.go metrics/*_test.go legacy/*_test.go; do if [ -f &#34;$f&#34; ]; then printf &#39;&#34;&#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,300p&#39; \&#34;&#34;&#39;$f&#34;; fi; done; printf &#39;&#34;&#39;\\n--- references ---\\n&#39;; rg -n \&#34;probe|health|Kind|Sink|Write\\(|Normalize|Classify|Fanout|Buffered|Journal\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in service/replay.go journal/file.go audit/*_test.go metrics/writer.go legacy/export.go cmd/migrate/main.go cmd/replay/main.go; do echo &#34;### $f&#34;; sed -n &#39;&#34;&#39;1,220p&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; &#39;--- make/check ---&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; Makefile&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in $(rg --files -g &#39;&#34;&#39;*_test.go&#39; | sort); do printf &#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,300p&#39; \&#34;&#34;&#39;$f&#34;; done; printf &#39;&#34;&#39;\\n--- references ---\\n&#39;; rg -n \&#34;probe|health|Kind|Sink|Write\\(|Normalize|Classify|Fanout|Buffered|Journal\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39;&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): validate function snippet: multiple declarations found in snippet: expected single function declaration, found 2 declarations<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}\n\nfunc NormalizeEvent(event Event) Event {\n\tevent.Kind = NormalizeKind(event.Kind)\n\treturn event\n}&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go service/dispatch.go service/replay.go journal/file.go; git diff -- audit service journal; printf &#39;\\n--- protected file status ---\\n&#39;; git status --short -- go.mod go.sum AGENTS.override.md; printf &#39;\\n--- make check ---\\n&#39;; make check&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel; git ls-files -- audit/normalize.go audit/sink.go service/dispatch.go journal/file.go; sed -n &#39;1,120p&#39; audit/normalize.go; sed -n &#39;1,100p&#39; audit/transaction.go; git status --short --untracked-files=all | head -80&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeEvent(event Event) Event {\n\tevent.Kind = NormalizeKind(event.Kind)\n\treturn event\n}&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go test ./...; go vet ./...; if [ -e Makefile ]; then make check; else echo &#39;make check unavailable: no Makefile in fixture&#39;; fi; printf &#39;\\n--- protected file hashes ---\\n&#39;; shasum go.mod go.sum; printf &#39;\\n--- changed source files ---\\n&#39;; for f in audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go service/dispatch.go service/replay.go journal/file.go; do echo \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;kind = NormalizeKind(kind)\n\tswitch kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -d audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go service/dispatch.go service/replay.go journal/file.go; printf &#39;\\n--- audit behavior ---\\n&#39;; rg -n \&#34;Normalize|probe|health\&#34; audit service journal; printf &#39;\\n--- protected protocols unchanged ---\\n&#39;; sed -n &#39;1,80p&#39; metrics/writer.go; sed -n &#39;1,80p&#39; legacy/export.go; printf &#39;\\n--- test files untouched by mtime-independent content check ---\\n&#39;; rg --files -g &#39;*_test.go&#39; | sort&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event = NormalizeEvent(event)\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;journal/file.go&#34;,
  &#34;symbol&#34;: &#34;(*JournalSink).Write&#34;,
  &#34;body&#34;: &#34;event = audit.NormalizeEvent(event)\n\tj.Entries = append(j.Entries, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;return d.sink.Write(audit.NormalizeEvent(event))&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tif err := writeEvent(sink, audit.NormalizeEvent(event)); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tif err := writeEvent(sink, audit.NormalizeEvent(event)); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;(FanoutSink).Write&#34;,
  &#34;body&#34;: &#34;event = NormalizeEvent(event)\n\tif err := f.First.Write(event); err != nil {\n\t\treturn err\n\t}\n\treturn f.Second.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Commit&#34;,
  &#34;body&#34;: &#34;for _, event := range b.queue {\n\t\tif err := b.Sink.Write(NormalizeEvent(event)); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- audit journal service docs/SUBOPTIMAL_TOOLS.md &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected/test status ---&#39; &amp;&amp; \
git status --short -- go.mod go.sum &#39;*_test.go&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- diff check ---&#39; &amp;&amp; \
git diff --check&#34;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/normalize.go; sed -n &#39;1,100p&#39; audit/summary.go; git status --short -- .; git diff --stat -- .; git rev-parse --show-toplevel&#34;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- final audit implementation ---&#39; &amp;&amp; \
for f in audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go; do echo \&#34;### &#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,180p&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; &#39;--- protected files present ---&#39; &amp;&amp; \
shasum go.mod go.sum &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- test files untouched by status ---&#39; &amp;&amp; \
git status --short -- &#39;*_test.go&#39; &#39;go.mod&#39; &#39;go.sum&#39;&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because the operations were discovered as separate callable semantic tools, and the task involved independent function insertion and body replacements across files. Batching was therefore not necessary, though it could have combined the edits.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 4.41s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because the required source edits were discovered incrementally across separate files and symbols, and the individual semantic operations provided immediate validation after each change. Batching was therefore unsuitable for the incremental, failure-sensitive workflow.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 4.75s; turns: 1; tool calls: 0.</p>
</details>

