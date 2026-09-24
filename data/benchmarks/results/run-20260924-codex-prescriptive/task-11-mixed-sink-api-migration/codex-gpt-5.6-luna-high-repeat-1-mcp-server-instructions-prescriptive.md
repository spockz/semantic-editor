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

* **Date**: 2026-09-24 18:34:15 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-5.6-luna/high (repeat 1)`

#### Configuration: default prompt · prescriptive MCP instructions

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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 115.43s | 123.12s | N/A | 162.74s | 274.96s | <span class="benchmark-delta-negative">+69.0%</span> |
| **Process Start → First Event** | 0.12s | 0.12s | — | 0.12s | 0.12s | — |
| **First Event → First Tool Call** | 7.63s | 9.46s | — | 9.04s | 8.71s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | 1 | 1 | 0% |
| **Internal Tool Cycles** | 8 | 10 | N/A | 10 | 25 | <span class="benchmark-delta-negative">+150.0%</span> |
| **Initial Load / Discovery Turns** | 1 | 1 | N/A | 2 | 2 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 8 | 10 | N/A | 10 | 25 | <span class="benchmark-delta-negative">+150.0%</span> |
| **Output Tokens** | 5219 | 5378 | N/A | 7192 | 8881 | <span class="benchmark-delta-negative">+23.5%</span> |
| **Reasoning / Thinking Tokens** | 3292 | 3332 | N/A | 3875 | 5073 | <span class="benchmark-delta-negative">+30.9%</span> |
| **Total Input Tokens** | 233129 | 285581 | N/A | 384319 | 856359 | <span class="benchmark-delta-negative">+122.8%</span> |
| **Cached Input Tokens** | 207360 | 240640 | N/A | 348672 | 766976 | <span class="benchmark-delta-positive">+120.0%</span> |
| **Uncached Input Tokens** | 25769 | 44941 | N/A | 35647 | 89383 | <span class="benchmark-delta-negative">+150.7%</span> |
| **Cached vs Uncached Token Ratio** | 8.05:1 | 5.35:1 | N/A | 9.78:1 | 8.58:1 | <span class="benchmark-delta-negative">-12.3%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!*AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sed -n &#39;1,160p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;&#34;&#39;!*_test.go&#39;&#34;&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sort; printf &#39;\\n--- docs ---\\n&#39;; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;docs/**&#39; -g &#39;AGENTS.md&#39; | sort&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,260p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,280p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- docs/audit-sinks.md ---&#39;; sed -n &#39;1,240p&#39; docs/audit-sinks.md; printf &#39;%s\\n&#39; &#39;--- source ---&#39;; for f in audit/event.go audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go metrics/writer.go legacy/export.go cmd/migrate/main.go; do echo \&#34;--- &#34;&#39;$f ---&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,300p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,240p&#39; legacy/export_test.go &amp;&amp; \
sed -n &#39;1,240p&#39; metrics/writer_test.go &amp;&amp; \
git status --short &amp;&amp; \
git diff -- audit service legacy metrics docs&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- tests ---&#39;; for f in &#34;&#39;$(rg --files -g &#39;&#34;&#39;*_test.go&#39; | sort); do echo \&#34;--- &#34;&#39;$f ---&#34;; sed -n &#39;&#34;&#39;1,300p&#39; \&#34;&#34;&#39;$f&#34;; done; printf &#39;&#34;&#39;%s\\n&#39; &#39;--- repo files ---&#39;; rg --files -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;.golangci*&#39; -g &#39;scripts/**&#39; -g &#39;*.sh&#39; | sort; printf &#39;%s\\n&#39; &#39;--- git status ---&#39;; git status --short&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; cmd/migrate/main.go &amp;&amp; \
find . -maxdepth 3 -type f -not -path &#39;./.git/*&#39; -print | sort &amp;&amp; \
git ls-files --stage -- service audit legacy metrics docs cmd go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- module ---&#39;; sed -n &#39;1,120p&#39; go.mod; printf &#39;%s\\n&#39; &#39;--- all source references ---&#39;; rg -n \&#34;Normalize|normalize|Write\\(|DispatchAll|Record\\(|Classify|probe|health|MemorySink|Sink\&#34; . -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;; printf &#39;%s\\n&#39; &#39;--- command files ---&#39;; find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git log --oneline --all --decorate -8; git branch --show-current; git diff -- audit service metrics legacy go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git log --oneline --all -- audit/normalize.go audit/sink.go service/dispatch.go | head -20; git show HEAD:audit/normalize.go; git show HEAD:audit/sink.go&#39;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short --untracked-files=all -- . &amp;&amp; \
git diff --no-ext-diff -- go.mod go.sum &amp;&amp; \
sed -n &#39;1,180p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,180p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,220p&#39; service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short --untracked-files=all -- .; git ls-files -- . | head -50; ls -la&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel &amp;&amp; \
git status --short --untracked-files=all -- audit service legacy metrics docs cmd go.mod go.sum &amp;&amp; \
git diff --stat -- audit service legacy metrics docs cmd go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/summary.go
go test ./...&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go vet ./...
make check
printf &#39;%s\\n&#39; &#39;--- changed implementation ---&#39;
sed -n &#39;1,120p&#39; audit/normalize.go
sed -n &#39;1,120p&#39; audit/sink.go
sed -n &#39;1,80p&#39; audit/summary.go
printf &#39;%s\\n&#39; &#39;--- protected files present ---&#39;
stat -f &#39;%N %z bytes&#39; go.mod audit/event_test.go service/dispatch_test.go metrics/writer_test.go legacy/export_test.go&#34;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 13 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | sort &amp;&amp; \
printf &#39;\\n--- status ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;docs/**&#39; | sed -n &#39;1,160p&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -la &amp;&amp; \
printf &#39;\\n--- docs ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
printf &#39;\\n--- event.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
printf &#39;\\n--- normalize.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
printf &#39;\\n--- fanout.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,280p&#39; audit/fanout.go &amp;&amp; \
printf &#39;\\n--- sink.go ---\\n&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/transaction.go &amp;&amp; \
sed -n &#39;1,220p&#39; docs/audit-sinks.md&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- all Go sources ---&#39; &amp;&amp; \
rg -n \&#34;audit|Event|Kind|Probe|probe|Write\\(|Fanout|metrics|legacy|Normalize|normalize\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; . &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- audit tests ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,300p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,300p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,300p&#39; audit/transaction.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- service dispatch ---&#39; &amp;&amp; \
sed -n &#39;1,320p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,360p&#39; service/dispatch_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- metrics and legacy ---&#39; &amp;&amp; \
sed -n &#39;1,280p&#39; metrics/metrics.go &amp;&amp; \
sed -n &#39;1,320p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,280p&#39; legacy/export.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/transaction.go &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/metrics.go &amp;&amp; \
sed -n &#39;1,280p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,280p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/replay.go &amp;&amp; \
rg -n \&#34;audit|Event|Probe|probe|Write\\(\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; .&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- remaining sources ---&#39; &amp;&amp; \
for f in api/router.go auth/auth.go cache/cache.go config/config.go db/db.go health/health.go journal/file.go logging/logging.go models/models.go service/replay.go utils/utils.go; do echo \&#34;--- &#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; &#39;--- module ---&#39; &amp;&amp; \
sed -n &#39;1,80p&#39; go.mod &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- git diff in fixture ---&#39; &amp;&amp; \
git diff -- . &#39;:(exclude)go.mod&#39; &#39;:(exclude)go.sum&#39; &#39;:(exclude)AGENTS.override.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; journal/file.go &amp;&amp; \
sed -n &#39;1,260p&#39; api/router.go &amp;&amp; \
sed -n &#39;1,240p&#39; db/db.go &amp;&amp; \
sed -n &#39;1,240p&#39; logging/logging.go &amp;&amp; \
sed -n &#39;1,240p&#39; health/health.go &amp;&amp; \
sed -n &#39;1,240p&#39; cache/cache.go &amp;&amp; \
sed -n &#39;1,240p&#39; models/models.go &amp;&amp; \
rg --files -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;*brief*&#39; -g &#39;AGENTS*&#39; -g &#39;docs/**&#39; | sort&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find . -maxdepth 2 -type f -not -name AGENTS.override.md -not -name go.mod -not -name go.sum -print | sort &amp;&amp; \
printf &#39;\\n--- available checks ---\\n&#39; &amp;&amp; \
find . -maxdepth 2 -type f \\( -name &#39;Makefile&#39; -o -name &#39;Taskfile.yml&#39; -o -name &#39;README*&#39; -o -name &#39;PROJECT_BRIEF.md&#39; \\) -print&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all --decorate -12 &amp;&amp; \
git diff -- audit go.mod go.sum service journal metrics legacy docs/audit-sinks.md &amp;&amp; \
git show HEAD:go.mod &amp;&amp; \
find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- diff stat ---&#39; &amp;&amp; \
git diff --stat -- audit journal service &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- diff ---&#39; &amp;&amp; \
git diff -- audit journal service &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected file status ---&#39; &amp;&amp; \
git status --short -- go.mod go.sum AGENTS.override.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,300p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; legacy/export_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer_test.go &amp;&amp; \
go env GOMOD &amp;&amp; \
go test ./...&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse --show-toplevel &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tracked fixture files ---&#39; &amp;&amp; \
git ls-files -- audit journal service | head -100 &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- source snapshots ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- status scoped ---&#39; &amp;&amp; \
git status --short --untracked-files=all -- audit journal service&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): validate function snippet: multiple declarations found in snippet: expected single function declaration, found 2 declarations<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}\n\nfunc NormalizeEvent(event Event) Event {\n\tevent.Kind = NormalizeKind(event.Kind)\n\treturn event\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w audit/transaction.go &amp;&amp; \
go test ./... &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- protected files unchanged by mtime check ---&#39; &amp;&amp; \
stat -f &#39;%N %m&#39; go.mod go.sum 2&gt;/dev/null || \
true &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- changed source mtimes ---&#39; &amp;&amp; \
stat -f &#39;%N %m&#39; audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go&#34;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeEvent(event Event) Event {\n\tevent.Kind = NormalizeKind(event.Kind)\n\treturn event\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go vet ./...&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;kind = NormalizeKind(kind)\n\tswitch kind {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event = NormalizeEvent(event)\n\tm.Events = append(m.Events, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;FanoutSink.Write&#34;,
  &#34;body&#34;: &#34;event = NormalizeEvent(event)\n\tif err := f.First.Write(event); err != nil {\n\t\treturn err\n\t}\n\treturn f.Second.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Commit&#34;,
  &#34;body&#34;: &#34;for _, event := range b.queue {\n\t\tif err := b.Sink.Write(NormalizeEvent(event)); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;return d.sink.Write(audit.NormalizeEvent(event))&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tif err := writeEvent(sink, audit.NormalizeEvent(event)); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\n\tfor _, event := range events {\n\t\tif err := writeEvent(sink, audit.NormalizeEvent(event)); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;journal/file.go&#34;,
  &#34;symbol&#34;: &#34;(*JournalSink).Write&#34;,
  &#34;body&#34;: &#34;event = audit.NormalizeEvent(event)\n\tj.Entries = append(j.Entries, event)\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;source&#34;: &#34;func (b *BufferedSink) Write(event Event) error {\n\tb.queue = append(b.queue, NormalizeEvent(event))\n\treturn nil\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit journal service &amp;&amp; \
git diff --check &amp;&amp; \
git diff --name-only -- go.mod go.sum &amp;&amp; \
go test ./...&#39;</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go docs/SUBOPTIMAL_TOOLS.md &amp;&amp; \
printf &#39;\\nChanged files in fixture:\\n&#39; &amp;&amp; \
git status --short -- audit journal service docs go.mod go.sum &amp;&amp; \
printf &#39;\\nProtected-file diff check:\\n&#39; &amp;&amp; \
git diff --quiet -- go.mod go.sum; printf &#39;exit=%s\\n&#39; \&#34;&#34;&#39;$?&#34;&#39;</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; audit/normalize.go; sed -n &#39;1,160p&#39; audit/summary.go; sed -n &#39;1,180p&#39; audit/transaction.go; sed -n &#39;1,140p&#39; service/dispatch.go; git status --short --untracked-files=all | sed -n &#39;1,120p&#39;; git rev-parse --show-toplevel; git ls-files audit/normalize.go audit/summary.go&#34;</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git check-ignore -v audit/normalize.go docs/SUBOPTIMAL_TOOLS.md || \
true; git ls-files --error-unmatch audit/normalize.go || \
true; git status --short --untracked-files=all -- .scratch/benchmarks/run_codex_semedit_large-default_task-11-mixed-sink-api-migration_1790264881860244000 | sed -n &#39;1,80p&#39;; git diff --no-index /dev/null audit/normalize.go | sed -n &#39;1,120p&#39;&#34;</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go vet ./... &amp;&amp; \
git diff --check &amp;&amp; \
test ! -e go.sum &amp;&amp; \
printf &#39;&#34;&#39;go.sum absent and untouched\\n&#39; &amp;&amp; \
printf &#39;production files:\\n&#39; &amp;&amp; \
for f in audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go; do test -f \&#34;&#34;&#39;$f&#34; &amp;&amp; \
printf &#39;&#34;&#39;%s\\n&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
printf &#39;&#34;&#39;test files unchanged by task operations\\n&#39;&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not discover any callable `semantic_*` tools from the available semedit MCP server in this environment. I therefore used ordinary editing for the small, localized change.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 5.90s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I used individual semantic operations because the edits targeted distinct existing functions across multiple files, while `semantic_batch` was not necessary for these simple, sequential changes. It was available but unsuitable here because one operation initially required separate single-function insertions after rejecting a multi-declaration snippet.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 5.28s; turns: 1; tool calls: 0.</p>
</details>

#### Configuration: prefer_discover_semedit prompt · prescriptive MCP instructions

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
| **Wall-Clock Latency** | 163.52s | 178.96s | <span class="benchmark-delta-negative">+9.4%</span> | 197.79s | 247.02s | <span class="benchmark-delta-negative">+24.9%</span> |
| **Process Start → First Event** | 0.12s | 0.13s | — | 0.13s | 0.12s | — |
| **First Event → First Tool Call** | 19.14s | 17.10s | — | 16.21s | 19.07s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 19 | 17 | <span class="benchmark-delta-positive">-10.5%</span> | 11 | 22 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Initial Load / Discovery Turns** | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 19 | 17 | <span class="benchmark-delta-positive">-10.5%</span> | 11 | 22 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Output Tokens** | 6933 | 5637 | <span class="benchmark-delta-positive">-18.7%</span> | 8823 | 7677 | <span class="benchmark-delta-positive">-13.0%</span> |
| **Reasoning / Thinking Tokens** | 3364 | 3222 | <span class="benchmark-delta-positive">-4.2%</span> | 4500 | 4546 | <span class="benchmark-delta-negative">+1.0%</span> |
| **Total Input Tokens** | 482477 | 792585 | <span class="benchmark-delta-negative">+64.3%</span> | 571582 | 1119409 | <span class="benchmark-delta-negative">+95.8%</span> |
| **Cached Input Tokens** | 418304 | 745216 | <span class="benchmark-delta-positive">+78.2%</span> | 525312 | 1063936 | <span class="benchmark-delta-positive">+102.5%</span> |
| **Uncached Input Tokens** | 64173 | 47369 | <span class="benchmark-delta-positive">-26.2%</span> | 46270 | 55473 | <span class="benchmark-delta-negative">+19.9%</span> |
| **Cached vs Uncached Token Ratio** | 6.52:1 | 15.73:1 | <span class="benchmark-delta-positive">+141.4%</span> | 11.35:1 | 19.18:1 | <span class="benchmark-delta-positive">+68.9%</span> |
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
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; -g &#39;*.txtar&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- files ---&#39;; rg --files -g &#39;&#34;&#39;!tests/**&#39;&#34;&#39; -g &#39;&#34;&#39;!testdata/**&#39;&#34;&#39; | sort | sed -n &#39;1,160p&#39;; printf &#39;%s\\n&#39; &#39;--- brief ---&#39;; if test -f PROJECT_BRIEF.md; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi; printf &#39;%s\\n&#39; &#39;--- status ---&#39;; git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff --stat&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- docs/audit-sinks.md ---&#39;; sed -n &#39;1,260p&#39; docs/audit-sinks.md; printf &#39;%s\\n&#39; &#39;--- audit files ---&#39;; for f in audit/event.go audit/normalize.go audit/sink.go audit/summary.go; do printf &#39;\\n### %s\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done; printf &#39;&#34;&#39;%s\\n&#39; &#39;--- consumers ---&#39;; for f in service/dispatch.go cmd/migrate/main.go metrics/writer.go legacy/export.go; do printf &#39;\\n### %s\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in audit/event_test.go audit/summary_test.go service/dispatch_test.go metrics/writer_test.go legacy/export_test.go; do printf &#39;\\n### %s\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,320p&#39; \&#34;&#34;&#39;$f&#34;; done; printf &#39;&#34;&#39;\\n--- module ---\\n&#39;; sed -n &#39;1,120p&#39; go.mod; printf &#39;\\n--- symbols ---\\n&#39;; rg -n \&#34;Record|DispatchAll|Write\\(|Classify|probe|health|Kind|normalize|normalize\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,260p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,240p&#39; cmd/migrate/main.go &amp;&amp; \
sed -n &#39;1,160p&#39; go.mod&#34;</code></pre></td><td><code>cua_repl/js</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;code&#34;: &#34;await cua.getState()&#34;,
  &#34;title&#34;: &#34;Inspect connected tool surfaces&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,300p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,300p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/writer_test.go &amp;&amp; \
sed -n &#39;1,240p&#39; legacy/export_test.go&#34;</code></pre></td><td><code>cua_repl/js</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;code&#34;: &#34;await cua.getState({emit:false})&#34;,
  &#34;title&#34;: &#34;Confirm workspace tool surface&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;Makefile&#39; -g &#39;*.yaml&#39; -g &#39;*.yml&#39; -g &#39;*.sh&#39; -g &#39;README*&#39; -g &#39;PROJECT_BRIEF.md&#39; | sort | xargs -r -n1 sh -c &#39;echo --- \&#34;&#34;&#39;$0&#34;; sed -n &#34;1,220p&#34; &#34;$0&#34;&#39;&#34;&#39;&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --hidden --glob &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &#39;Kind|Classify|\\.Write\\(|DispatchAll|Record\\(&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- fixture metadata ---&#39;; rg --files -g &#39;.agents/**&#39; -g &#39;.codex/**&#39; -g &#39;*.md&#39; | sort; printf &#39;%s\\n&#39; &#39;--- all source ---&#39;; rg -n \&#34;&#34;&#39;^func |&#39;&#39;^type |&#39;&#39;^var |&#39;&#39;^const &#34; --glob &#39;&#34;&#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git log --oneline --all --decorate -8 -- . &amp;&amp; \
git diff -- audit/event.go audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go metrics/writer.go legacy/export.go&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\nfor _, event := range events {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\tif err := writeEvent(sink, event); err != nil {\n\t\treturn err\n\t}\n}\nreturn nil&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go audit/event_test.go audit/summary_test.go service/dispatch_test.go metrics/writer.go legacy/export.go go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event.Kind = NormalizeKind(event.Kind)\nm.Events = append(m.Events, event)\nreturn nil&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
git diff -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- diff ---&#39;; git diff -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go; printf &#39;%s\\n&#39; &#39;--- protected file diff ---&#39;; git diff -- go.mod go.sum; printf &#39;%s\\n&#39; &#39;--- changed files ---&#39;; git status --short -- audit service metrics legacy cmd go.mod go.sum; printf &#39;%s\\n&#39; &#39;--- top-level ---&#39;; ls -la&#34;</code></pre></td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,200p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,140p&#39; service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go; do printf &#39;\\n### %s\\n&#39; \&#34;&#34;&#39;$f&#34;; sed -n &#39;&#34;&#39;1,140p&#39; \&#34;&#34;&#39;$f&#34;; done; printf &#39;&#34;&#39;\\n--- git root ---\\n&#39;; git rev-parse --show-toplevel; git status --short --untracked-files=all -- .&#34;</code></pre></td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel &amp;&amp; \
git ls-files --stage -- audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go &amp;&amp; \
git check-ignore -v audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go || \
true&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --no-ext-diff --no-index /dev/null audit/normalize.go; git diff --stat --no-ext-diff&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit service cmd/migrate metrics legacy; go test ./...&#39;</code></pre></td></tr>
<tr><td>17</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go&#39;</code></pre></td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>18</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go vet ./...&#39;</code></pre></td><td>—</td></tr>
<tr><td>19</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 13 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 13 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; -g &#39;*.md&#39; | sort | sed -n &#39;1,240p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; | sort &amp;&amp; \
echo &#39;--- status ---&#39; &amp;&amp; \
git status --short &amp;&amp; \
echo &#39;--- docs candidates ---&#39; &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; | sort&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; AGENTS.override.md &amp;&amp; \
sed -n &#39;1,260p&#39; docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/transaction.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in docs/audit-sinks.md audit/event.go audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go service/dispatch.go service/replay.go metrics/metrics.go metrics/writer.go legacy/export.go; do echo &#34;--- $f ---&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in api/router.go auth/auth.go cache/cache.go cmd/migrate/main.go cmd/replay/main.go config/config.go db/db.go health/health.go journal/file.go legacy/export.go logging/logging.go metrics/metrics.go metrics/writer.go models/models.go service/dispatch.go service/replay.go utils/utils.go audit/summary.go; do echo &#34;--- $f&#34;; sed -n &#39;&#34;&#39;1,280p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in audit/*_test.go service/*_test.go metrics/*_test.go legacy/*_test.go api/*_test.go; do echo &#34;--- $f ---&#34;; sed -n &#39;&#34;&#39;1,280p&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
echo &#39;&#34;&#39;--- go.mod ---&#39; &amp;&amp; \
sed -n &#39;1,120p&#39; go.mod &amp;&amp; \
echo &#39;--- Makefile? ---&#39; &amp;&amp; \
if [ -f Makefile ]; then sed -n &#39;1,220p&#39; Makefile; else echo &#39;(none)&#39;; fi&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in audit/*_test.go service/*_test.go metrics/*_test.go legacy/*_test.go api/*_test.go; do echo &#34;--- $f&#34;; sed -n &#39;&#34;&#39;1,300p&#39; \&#34;&#34;&#39;$f&#34;; done; echo &#39;&#34;&#39;--- git status&#39;; git status --short; echo &#39;--- git diff&#39;; git diff -- . &#39;:(exclude)go.mod&#39; &#39;:(exclude)go.sum&#39;&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve Normalize in audit/normalize.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; go.mod; rg -n \&#34;audit|probe|normalize|Sink|Record|Replay|DispatchAll|Classify\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;probe|Normalize|normalize|Classify|Sink|Record|DispatchAll|Replay|audit\&#34; . -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all -- audit service journal docs/audit-sinks.md | sed -n &#39;1,80p&#39;; git branch --all --no-color | sed -n &#39;1,80p&#39;; find . -maxdepth 3 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in journal/file.go api/router.go cmd/migrate/main.go cmd/replay/main.go; do echo &#34;--- $f ---&#34;; sed -n &#39;&#34;&#39;1,220p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; audit/transaction.go; sed -n &#39;1,80p&#39; audit/normalize.go; git diff -- audit journal service&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_insert_declaration</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;visibility&#34;: &#34;public&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit journal service metrics legacy go.mod go.sum; git status --short -- audit journal service metrics legacy go.mod go.sum; test ! -e Makefile &amp;&amp; \
echo &#39;&#34;&#39;no Makefile in fixture&#39; || \
true; go vet ./...&#34;</code></pre></td><td><code>semedit/semantic_insert_declaration</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;visibility&#34;: &#34;public&#34;,
  &#34;source&#34;: &#34;func NormalizeEvent(event Event) Event {\n\tevent.Kind = NormalizeKind(event.Kind)\n\treturn event\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; audit/normalize.go; sed -n &#39;1,120p&#39; audit/sink.go; sed -n &#39;1,120p&#39; audit/fanout.go; sed -n &#39;1,160p&#39; audit/transaction.go; sed -n &#39;1,100p&#39; audit/summary.go; sed -n &#39;1,80p&#39; journal/file.go; sed -n &#39;1,120p&#39; service/dispatch.go; sed -n &#39;1,100p&#39; service/replay.go; git rev-parse --show-toplevel; git status --short --ignored -- audit/normalize.go audit/sink.go&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;event = NormalizeEvent(event)\nm.Events = append(m.Events, event)\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
go test -race ./...&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/fanout.go&#34;,
  &#34;symbol&#34;: &#34;FanoutSink.Write&#34;,
  &#34;body&#34;: &#34;event = NormalizeEvent(event)\nif err := f.First.Write(event); err != nil {\n\treturn err\n}\nreturn f.Second.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;(*BufferedSink).Commit&#34;,
  &#34;body&#34;: &#34;for _, event := range b.queue {\n\tevent = NormalizeEvent(event)\n\tif err := b.Sink.Write(event); err != nil {\n\t\treturn err\n\t}\n}\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event = audit.NormalizeEvent(event)\nreturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\nfor _, event := range events {\n\tevent = audit.NormalizeEvent(event)\n\tif err := writeEvent(sink, event); err != nil {\n\t\treturn err\n\t}\n}\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\nfor _, event := range events {\n\tevent = audit.NormalizeEvent(event)\n\tif err := writeEvent(sink, event); err != nil {\n\t\treturn err\n\t}\n}\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;journal/file.go&#34;,
  &#34;symbol&#34;: &#34;(*JournalSink).Write&#34;,
  &#34;body&#34;: &#34;event = audit.NormalizeEvent(event)\nj.Entries = append(j.Entries, event)\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go service/dispatch.go service/replay.go journal/file.go docs/SUBOPTIMAL_TOOLS.md &amp;&amp; \
echo &#39;--- protected-file diff ---&#39; &amp;&amp; \
git diff --name-only -- go.mod go.sum audit/*_test.go service/*_test.go metrics/*_test.go legacy/*_test.go api/*_test.go &amp;&amp; \
echo &#39;--- changed files ---&#39; &amp;&amp; \
git status --short -- audit docs service journal metrics legacy api go.mod go.sum &amp;&amp; \
echo &#39;--- formatting check ---&#39; &amp;&amp; \
gofmt -d audit journal service&#34;</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; audit/normalize.go; sed -n &#39;1,80p&#39; audit/sink.go; sed -n &#39;1,80p&#39; service/dispatch.go; sed -n &#39;1,80p&#39; docs/SUBOPTIMAL_TOOLS.md; pwd; git rev-parse --show-toplevel; git ls-files --stage -- audit/normalize.go audit/sink.go&#34;</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
test ! -e go.sum &amp;&amp; \
cmp -s go.mod go.mod&#39;</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go vet ./... &amp;&amp; \
echo &#39;--- protected files present/unchanged by targeted operations ---&#39; &amp;&amp; \
ls -l go.mod go.sum 2&gt;/dev/null || \
true &amp;&amp; \
echo &#39;--- source references ---&#39; &amp;&amp; \
rg -n \&#34;Normalize(Event|Kind)|probe|strings\\.ToLower|metrics|legacy\&#34; audit service journal metrics legacy&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because the individual semantic operations were already discovered and callable, while the batch schema and supported operation composition were not established. Sequential calls were suitable and allowed each mutation to be independently verified.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 5.58s; turns: 1; tool calls: 0.</p>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">`semantic_batch` was discovered but not used because its generic edit schema was ambiguous for these heterogeneous, cross-file operations. I used sequential semantic calls to preserve clear per-operation diagnostics and failure handling.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 6.61s; turns: 1; tool calls: 0.</p>
</details>

