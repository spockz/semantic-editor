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

* **Date**: 2026-10-17 00:16:09 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-6-luna/medium (repeat 2)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,provider_transport=codex_app_server_v2`

* **Fixture**: [testdata/bench/task_11_mixed_sink_api_migration.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_11_mixed_sink_api_migration.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Audit event kinds may contain inconsistent whitespace and casing. Add the public audit.NormalizeKind(kind string) string function in audit/normalize.go to remove surrounding whitespace and lowercase the kind. Apply it in Dispatcher.Record before persistence, and classify normalized probe events as control traffic, like health. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit
```
</details>

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 15.89s | 80.19s | <span class="benchmark-delta-negative">+404.8%</span> | — | — | — |
| **Process Start → First Event** | 0.03s | 0.03s | — | — | — | — |
| **First Event → First Tool Call** | 3.62s | 4.55s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 5 | <span class="benchmark-delta-negative">+400.0%</span> | — | — | — |
| **Internal Tool Cycles** | 5 | 32 | <span class="benchmark-delta-negative">+540.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 5 | 32 | <span class="benchmark-delta-negative">+540.0%</span> | — | — | — |
| **Output Tokens** | 806 | 5331 | <span class="benchmark-delta-negative">+561.4%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 199 | 1127 | <span class="benchmark-delta-negative">+466.3%</span> | — | — | — |
| **Total Input Tokens** | 122190 | 1166409 | <span class="benchmark-delta-negative">+854.6%</span> | — | — | — |
| **Cached Input Tokens** | 99328 | 1076608 | <span class="benchmark-delta-positive">+983.9%</span> | — | — | — |
| **Uncached Input Tokens** | 22862 | 89801 | <span class="benchmark-delta-negative">+292.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.34:1 | 11.99:1 | <span class="benchmark-delta-positive">+175.9%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/sink.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat audit/event_test.go &amp;&amp; \
cat service/dispatch_test.go&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 0 params: unknown parameter \&#34;content\&#34; for \&#34;semantic_scaffold_file\&#34;; supported keys are [auto-organize-imports, auto_organize_imports, file, language, overwrite, package, purpose-heade<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_scaffold_file&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;content&#34;: &#34;package audit\n&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;function&#34;: &#34;func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) }&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;function&#34;: &#34;Dispatcher.Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;function&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: cat: docs/SUBOPTIMAL_TOOLS.md: No such file or directory<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/SUBOPTIMAL_TOOLS.md&#39;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 1 params: unknown parameter \&#34;body\&#34; for \&#34;semantic_insert_function\&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, bash-bin, bash_bin, file,<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_scaffold_file&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;package&#34;: &#34;audit&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;signature&#34;: &#34;func NormalizeKind(kind string) string&#34;,
      &#34;body&#34;: &#34;return strings.ToLower(strings.TrimSpace(kind))&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;function&#34;: &#34;Dispatcher.Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;function&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 2 params: unknown parameter \&#34;function\&#34; for \&#34;semantic_replace_body\&#34;; supported keys are [auto-organize-imports, auto_organize_imports, bash-bin, bash_bin, body, expected-revision, expe<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;source&#34;: &#34;func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) }&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;function&#34;: &#34;Dispatcher.Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;function&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/2a\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;source&#34;: &#34;func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) }&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;symbol&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;audit.Classify&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/2a\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;,
  &#34;trust_workspace&#34;: true,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_scaffold_file&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;package&#34;: &#34;audit&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;source&#34;: &#34;func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) }&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;symbol&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: cat: .agents/skills/semedit/references/golang.md: No such file or directory<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/references/golang.md&#39;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum &amp;&amp; \
test -f audit/normalize.go &amp;&amp; \
sed -n &#39;1,80p&#39; audit/normalize.go || \
true&#34;</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/2a\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;source&#34;: &#34;func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) }&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;symbol&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum service/dispatch.go audit/normalize.go &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/4b\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
      &#34;body&#34;: &#34;event.Kind = strings.ToLower(strings.TrimSpace(event.Kind))\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;
}</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/2a\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792186044635294000&#34;,
  &#34;auto_organize_imports&#34;: false,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;source&#34;: &#34;func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) }&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;symbol&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>31</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;command -v semedit &amp;&amp; \
semedit --help&#39;</code></pre></td></tr>
<tr><td>32</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,provider_transport=codex_app_server_v2`

* **Fixture**: [testdata/bench/task_11_mixed_sink_api_migration.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_11_mixed_sink_api_migration.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Audit event kinds may contain inconsistent whitespace and casing. Add the public audit.NormalizeKind(kind string) string function in audit/normalize.go to remove surrounding whitespace and lowercase the kind. Apply it in Dispatcher.Record before persistence, and classify normalized probe events as control traffic, like health. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Audit event kinds may contain inconsistent whitespace and casing. Add the public audit.NormalizeKind(kind string) string function in audit/normalize.go to remove surrounding whitespace and lowercase the kind. Apply it in Dispatcher.Record before persistence, and classify normalized probe events as control traffic, like health. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 15.89s | 25.32s | <span class="benchmark-delta-negative">+59.4%</span> | — | — | — |
| **Process Start → First Event** | 0.03s | 0.02s | — | — | — | — |
| **First Event → First Tool Call** | 3.62s | 4.42s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 5 | 15 | <span class="benchmark-delta-negative">+200.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 5 | 15 | <span class="benchmark-delta-negative">+200.0%</span> | — | — | — |
| **Output Tokens** | 806 | 1334 | <span class="benchmark-delta-negative">+65.5%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 199 | 232 | <span class="benchmark-delta-negative">+16.6%</span> | — | — | — |
| **Total Input Tokens** | 122190 | 346492 | <span class="benchmark-delta-negative">+183.6%</span> | — | — | — |
| **Cached Input Tokens** | 99328 | 309760 | <span class="benchmark-delta-positive">+211.9%</span> | — | — | — |
| **Uncached Input Tokens** | 22862 | 36732 | <span class="benchmark-delta-negative">+60.7%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.34:1 | 8.43:1 | <span class="benchmark-delta-positive">+94.1%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;audit/*.go&#39; -g &#39;Makefile&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/sink.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat audit/event_test.go &amp;&amp; \
cat service/dispatch_test.go&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: cat: PROJECT_BRIEF.md: No such file or directory<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat PROJECT_BRIEF.md&#39;</code></pre></td></tr>
<tr><td>4</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat audit/event.go&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;probe|health|Record\\(\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat audit/normalize.go&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;source&#34;: &#34;func Classify(kind string) string {\n\tswitch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;kind&#34;: &#34;method&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;source&#34;: &#34;func (d *Dispatcher) Record(event audit.Event) error {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: make[1]: warning: jobserver unavailable: using -j1. Add `+&#39; to parent make rule. make[1]: *** No rule to make target `check&#39;. Stop.<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: M ../../../Makefile M ../../../cmd/semedit/benchmark_exclusions_test.go M ../../../docs/SUBOPTIMAL_TOOLS.md M ../../../docs/adr/0050-benchmark-planning-sessions-and-arm-policy.md M ../../../docs/adr/0070-opt-in-go-runtime-isolation.md M ../../../docs/adr/README.md M ../../../tools/benchmark-harness/Makefile M ../../../tools/benchmark-harness/README.md M ../../../tools/benchmark-harness/driver.go M ../../../tools/benchmark-harness/main.go M ../../../tools/benchmark-harness/planner.go M ../../../tools/benchma<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git diff -- audit service &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
rg --files -g &#39;Makefile&#39; -g &#39;*.yml&#39; -g &#39;*.yaml&#39; -g &#39;README*&#39;&#34;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;check_only&#34;: true
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was unsuitable for these operations: adding a function and replacing two distinct function or method declarations were separate semantic edits, and I did not establish that `semantic_batch` supports those operation types. I made the calls consecutively without checking batch support.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit’s semantic tools let me add and update the Go code through structured edits, which felt reassuring, though I couldn’t assess batching; I’d ask my owner to add them for this kind of task.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

