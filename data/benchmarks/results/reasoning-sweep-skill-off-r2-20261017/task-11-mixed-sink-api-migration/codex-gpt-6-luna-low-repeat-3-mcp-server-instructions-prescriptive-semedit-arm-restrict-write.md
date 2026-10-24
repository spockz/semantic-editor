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

* **Date**: 2026-10-17 21:43:43 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-6-luna/low (repeat 3)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

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
| **Wall-Clock Latency** | 24.79s | 30.09s | <span class="benchmark-delta-negative">+21.4%</span> | — | — | — |
| **Process Start → First Event** | 0.03s | 0.02s | — | — | — | — |
| **First Event → First Tool Call** | 4.37s | 5.76s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **Internal Tool Cycles** | 6 | 7 | <span class="benchmark-delta-negative">+16.7%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 2 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 6 | 7 | <span class="benchmark-delta-negative">+16.7%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.004219 | $0.006187 | +46.6% | — | — | — |
| **Output Tokens** | 1133 | 1381 | <span class="benchmark-delta-negative">+21.9%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 155180 | 304831 | <span class="benchmark-delta-negative">+96.4%</span> | — | — | — |
| **Cached Input Tokens** | 131840 | 277632 | <span class="benchmark-delta-positive">+110.6%</span> | — | — | — |
| **Uncached Input Tokens** | 23340 | 27199 | <span class="benchmark-delta-negative">+16.5%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.65:1 | 10.21:1 | <span class="benchmark-delta-positive">+80.7%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;audit/**&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func \\(.*\\) Record|func Classify|NormalizeKind|func Classify\&#34; .&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit // Package audit defines the event delivery boundary used by the fixture. package audit type Event struct { ID string `json:&#34;id&#34;` Kind string `json:&#34;kind&#34;` } // Package audit exposes the established event sink contract and a memory implementation. package audit type Sink interface { Write(Event) error } type TransactionalSink interface { Sink Begin() error Commit() error } type MemorySink struct { Events []Eve<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat go.mod &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;audit/**&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*dispatcher*&#39; -g &#39;*metric*&#39; -g &#39;*export*&#39; | sort &amp;&amp; \
rg -n \&#34;type Dispatcher|func \\(.*Dispatcher.*Record|probe|health|legacy|metrics\&#34; . -g &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,100p&#39; audit/normalize.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package service coordinates application work with the configured audit sink. package service import &#34;example.com/auditapp/audit&#34; type Dispatcher struct { sink audit.Sink } func NewDispatcher(sink audit.Sink) *Dispatcher { return &amp;Dispatcher{sink: sink} } func (d *Dispatcher) Record(event audit.Event) error { return d.sink.Write(event) } func DispatchAll(sink audit.Sink, events []audit.Event) error { writeEvent := audit.Sink.Write for _, event := range events { if err := writeEvent(sink, event); err != ni<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat service/dispatch.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
git status --short 2&gt;/dev/null || \
true &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39;&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792249378525626000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 0 params: unknown parameter \&#34;body\&#34; for \&#34;semantic_insert_function\&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, bash-bin, bash_bin, file,<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792249378525626000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;name&#34;: &#34;NormalizeKind&#34;,
      &#34;signature&#34;: &#34;func NormalizeKind(kind string) string&#34;,
      &#34;body&#34;: &#34;return strings.ToLower(strings.TrimSpace(kind))&#34;,
      &#34;position&#34;: &#34;end&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;symbol&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;language&#34;: &#34;go&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go service/dispatch.go audit/summary.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- audit/normalize.go service/dispatch.go audit/summary.go go.mod go.sum 2&gt;/dev/null || \
true &amp;&amp; \
git status --short 2&gt;/dev/null || \
true&#39;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 0 params: param \&#34;placement\&#34; value \&#34;end\&#34; must be one of [file_start, file_end, public_start, public_end, private_start, private_end, before_symbol, after_symbol, type_start, type_end]:<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792249378525626000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
      &#34;placement&#34;: &#34;end&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;symbol&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;language&#34;: &#34;go&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792249378525626000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
      &#34;placement&#34;: &#34;file_end&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
      &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;symbol&#34;: &#34;Classify&#34;,
      &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;language&#34;: &#34;go&#34;
    }
  ]
}</code></pre></td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

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
| **Wall-Clock Latency** | 24.79s | 30.42s | <span class="benchmark-delta-negative">+22.7%</span> | — | — | — |
| **Process Start → First Event** | 0.03s | 0.02s | — | — | — | — |
| **First Event → First Tool Call** | 4.37s | 3.86s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **Internal Tool Cycles** | 6 | 16 | <span class="benchmark-delta-negative">+166.7%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 6 | 16 | <span class="benchmark-delta-negative">+166.7%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.004219 | $0.010069 | +138.7% | — | — | — |
| **Output Tokens** | 1133 | 1932 | <span class="benchmark-delta-negative">+70.5%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 155180 | 555632 | <span class="benchmark-delta-negative">+258.1%</span> | — | — | — |
| **Cached Input Tokens** | 131840 | 516224 | <span class="benchmark-delta-positive">+291.6%</span> | — | — | — |
| **Uncached Input Tokens** | 23340 | 39408 | <span class="benchmark-delta-negative">+68.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.65:1 | 13.10:1 | <span class="benchmark-delta-positive">+131.9%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 10 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;audit/**&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit // Package audit defines the event delivery boundary used by the fixture. package audit type Event struct { ID string `json:&#34;id&#34;` Kind string `json:&#34;kind&#34;` } // Package audit exposes the established event sink contract and a memory implementation. package audit type Sink interface { Write(Event) error } type TransactionalSink interface { Sink Begin() error Commit() error } type MemorySink struct { Events []Eve<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat go.mod &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit // Package audit defines the event delivery boundary used by the fixture. package audit type Event struct { ID string `json:&#34;id&#34;` Kind string `json:&#34;kind&#34;` } // Package service coordinates application work with the configured audit sink. package service import &#34;example.com/auditapp/audit&#34; type Dispatcher struct { sink audit.Sink } func NewDispatcher(sink audit.Sink) *Dispatcher { return &amp;Dispatcher{sink: sink}<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/normalize.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
cat PROJECT_BRIEF.md 2&gt;/dev/null || \
true &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;audit/**&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*dispatcher*&#39; -g &#39;*metric*&#39; -g &#39;*export*&#39; | sort &amp;&amp; \
rg -n \&#34;type Dispatcher|func \\(.*Dispatcher.*Record|probe|health|legacy|metrics\&#34; . -g &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package service coordinates application work with the configured audit sink. package service import &#34;example.com/auditapp/audit&#34; type Dispatcher struct { sink audit.Sink } func NewDispatcher(sink audit.Sink) *Dispatcher { return &amp;Dispatcher{sink: sink} } func (d *Dispatcher) Record(event audit.Event) error { return d.sink.Write(event) } func DispatchAll(sink audit.Sink, events []audit.Event) error { writeEvent := audit.Sink.Write for _, event := range events { if err := writeEvent(sink, event); err != ni<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat service/dispatch.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
git status --short 2&gt;/dev/null || \
true &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/sink.go; sed -n &#39;1,260p&#39; audit/summary.go; rg -n \&#34;health|probe|control|Record\\(\&#34; --glob &#39;*.go&#39; .; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; .&#34;</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service&#34;,
  &#34;include_tests&#34;: false,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go service/dispatch.go audit/summary.go &amp;&amp; \
go test ./... &amp;&amp; \
git diff -- audit/normalize.go service/dispatch.go audit/summary.go go.mod go.sum 2&gt;/dev/null || \
true &amp;&amp; \
git status --short 2&gt;/dev/null || \
true&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;include_tests&#34;: false,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;group&#34;: &#34;append&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;unknown param \&#34;new_name\&#34;; supported keys are [auto_organize_imports, file, import-maven, import_maven, java-bin, java_bin, jdtls-home, jdtls_home, language, symbol, to, trust-workspace, trust_workspace]: invalid operation parameters&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;,
  &#34;new_name&#34;: &#34;NormalizeKind&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;kind&#34;: &#34;decl&#34;,
  &#34;placement&#34;: &#34;file_start&#34;,
  &#34;group&#34;: &#34;standalone&#34;,
  &#34;source&#34;: &#34;import \&#34;strings\&#34;&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;source&#34;: &#34;func Classify(kind string) string {\n\tswitch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}\n}&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;source&#34;: &#34;func (d *Dispatcher) Record(event audit.Event) error {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)\n}&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit import ( &#34;strings&#34; ) func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) } // Package audit groups delivery kinds for operational reporting. package audit func Classify(kind string) string { switch NormalizeKind(kind) { case &#34;health&#34;, &#34;probe&#34;: return &#34;control&#34; default: return &#34;data&#34; } } // Package service coordinates application work with the configured audit sink. package <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; audit/normalize.go; sed -n &#39;1,80p&#39; audit/summary.go; sed -n &#39;1,55p&#39; service/dispatch.go; git status --short 2&gt;/dev/null || \
true; git diff -- go.mod go.sum &#39;*_test.go&#39;&#34;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # crypto/internal/boring/sig compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; FAIL example.com/auditapp/audit [build failed] FAIL example.com/auditapp/cmd/migrate [build failed] FAIL example.com/auditapp/legacy [build failed] FAIL example.com/auditapp/metrics [build failed] FAIL example.com/auditapp/service [build failed] FAIL<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;check_only&#34;: true,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/normalize.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792249437586496000&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I discovered `semantic_batch`, but I did not use it because I treated the edits as separate operations and didn’t recognize that the supported insert and replace operations could be batched. Batching was suitable and available; this was an oversight.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools: they let me make targeted Go edits and check diagnostics, though I regret missing the chance to batch compatible changes.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

