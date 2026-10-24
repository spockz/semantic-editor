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

* **Date**: 2026-10-18 17:10:17 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-6-luna/medium (repeat 2)`

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
| **Wall-Clock Latency** | 19.71s | 40.63s | <span class="benchmark-delta-negative">+106.2%</span> | — | — | — |
| **Process Start → First Event** | 0.16s | 0.07s | — | — | — | — |
| **First Event → First Tool Call** | 3.55s | 4.66s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 5 | 10 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 5 | 10 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.003095 | $0.009123 | +194.8% | — | — | — |
| **Output Tokens** | 705 | 2303 | <span class="benchmark-delta-negative">+226.7%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 563 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Total Input Tokens** | 111290 | 496045 | <span class="benchmark-delta-negative">+345.7%</span> | — | — | — |
| **Cached Input Tokens** | 93184 | 462592 | <span class="benchmark-delta-positive">+396.4%</span> | — | — | — |
| **Uncached Input Tokens** | 18106 | 33453 | <span class="benchmark-delta-negative">+84.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.15:1 | 13.83:1 | <span class="benchmark-delta-positive">+168.7%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 11 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;normalize.go&#39; -g &#39;*.go&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit // Package audit defines the event delivery boundary used by the fixture. package audit type Event struct { ID string `json:&#34;id&#34;` Kind string `json:&#34;kind&#34;` } // Package audit exposes the established event sink contract and a memory implementation. package audit type Sink interface { Write(Event) error } type TransactionalSink interface { Sink Begin() error Commit() error } type MemorySink struct { Events []Eve<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat PROJECT_BRIEF.md 2&gt;/dev/null &amp;&amp; \
find . -name AGENTS.md -o -name AGENTS.override.md&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md; if [ -f PROJECT_BRIEF.md ]; then cat PROJECT_BRIEF.md; fi; cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: ./service/dispatch.go:14:func (d *Dispatcher) Record(event audit.Event) error { ./legacy/export_test.go:14: if output.String() != &#34;archive&#34; || Classify(&#34;probe&#34;) != &#34;archive&#34; { ./service/dispatch_test.go:13: if err := dispatcher.Record(audit.Event{ID: &#34;evt-1&#34;, Kind: &#34;audit&#34;}); err != nil { ./metrics/writer.go:13: if name == &#34;probe&#34; { ./audit/summary_test.go:7: if got := Classify(&#34;health&#34;); got != &#34;control&#34; { ./audit/summary_test.go:8: t.Fatalf(&#34;classification = %q, want control&#34;, got) ./legacy/export.go:16: <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;health|probe|control|Record\\(&#39; . -g &#39;*.go&#39; &amp;&amp; \
git status --short &amp;&amp; \
cat go.mod&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch.go &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/summary.go&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go &amp;&amp; \
ls -la&#39;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792333922971088000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; d<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat &gt; audit/normalize.go &lt;&lt;&#39;EOF&#39;
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit

import (
	\&#34;strings\&#34;
)

// NormalizeKind trims surrounding whitespace and lowercases an audit event kind.
func NormalizeKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}
EOF
python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
p=Path(&#39;service/dispatch.go&#39;)
s=p.read_text().replace(&#39;func (d *Dispatcher) Record(event audit.Event) error {\\n\\treturn d.sink.Write(event)\\n}&#39;, &#39;func (d *Dispatcher) Record(event audit.Event) error {\\n\\tevent.Kind = audit.NormalizeKind(event.Kind)\\n\\treturn d.sink.Write(event)\\n}&#39;)
p.write_text(s)
p=Path(&#39;audit/summary.go&#39;)
s=p.read_text().replace(&#39;switch kind {\\n\\tcase \&#34;health\&#34;:&#39;, &#39;switch NormalizeKind(kind) {\\n\\tcase \&#34;health\&#34;, \&#34;probe\&#34;:&#39;)
p.write_text(s)
PY
gofmt -w audit/normalize.go audit/summary.go service/dispatch.go
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792333922971088000&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792333922971088000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 0 params: unknown parameter \&#34;symbol\&#34; for \&#34;semantic_insert_function\&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, bash-bin, bash_bin, fil<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792333922971088000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;symbol&#34;: &#34;NormalizeKind&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
        &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/summary.go&#34;,
        &#34;symbol&#34;: &#34;Classify&#34;,
        &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 1 params: unknown parameter \&#34;source\&#34; for \&#34;semantic_replace_body\&#34;; supported keys are [auto-organize-imports, auto_organize_imports, bash-bin, bash_bin, body, expected-revision, expect<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792333922971088000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;target&#34;: &#34;Dispatcher.Record&#34;,
        &#34;source&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/summary.go&#34;,
        &#34;target&#34;: &#34;Classify&#34;,
        &#34;source&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Verification returned 2 diagnostics. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;available&#34;,&#34;report_id&#34;:&#34;5FV5G2NLQKR6MW5YOI6AACX5PI&#34;,&#34;arguments&#34;:{&#34;category&#34;:&#34;active&#34;,&#34;limit&#34;:50,&#34;offset&#34;:0,&#34;report_id&#34;:&#34;5FV5G2NLQKR6MW5YOI6AACX5PI&#34;},&#34;coverage&#34;:[{&#34;status&#34;:&#34;verified&#34;,&#34;build_exclusions&#34;:[&#34;Go build context and build tags&#34;,&#34;testdata&#34;,&#34;build&#34;,&#34;dist&#34;,&#34;target&#34;,&#34;vendor&#34;,&#34;node_modules&#34;,&#34;.scratch&#34;,&#34;.git&#34;,&#34;bazel-*&#34;,&#34;_*&#34;,&#34;.*&#34;],&#34;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792333922971088000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
        &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/summary.go&#34;,
        &#34;symbol&#34;: &#34;Classify&#34;,
        &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792333922971088000&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
      }
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
| **Wall-Clock Latency** | 19.71s | 48.46s | <span class="benchmark-delta-negative">+145.9%</span> | — | — | — |
| **Process Start → First Event** | 0.16s | 0.16s | — | — | — | — |
| **First Event → First Tool Call** | 3.55s | 4.42s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 5 | 10 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 5 | 10 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.003095 | $0.009401 | +203.8% | — | — | — |
| **Output Tokens** | 705 | 2218 | <span class="benchmark-delta-negative">+214.6%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 787 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Total Input Tokens** | 111290 | 544297 | <span class="benchmark-delta-negative">+389.1%</span> | — | — | — |
| **Cached Input Tokens** | 93184 | 512640 | <span class="benchmark-delta-positive">+450.1%</span> | — | — | — |
| **Uncached Input Tokens** | 18106 | 31657 | <span class="benchmark-delta-negative">+74.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.15:1 | 16.19:1 | <span class="benchmark-delta-positive">+214.6%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 11 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;audit/**&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit // Package audit defines the event delivery boundary used by the fixture. package audit type Event struct { ID string `json:&#34;id&#34;` Kind string `json:&#34;kind&#34;` } // Package audit exposes the established event sink contract and a memory implementation. package audit type Sink interface { Write(Event) error } type TransactionalSink interface { Sink Begin() error Commit() error } type MemorySink struct { Events []Eve<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat PROJECT_BRIEF.md 2&gt;/dev/null &amp;&amp; \
find . -name AGENTS.md -o -name AGENTS.override.md&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: --- name: semedit description: Use semedit tools for symbol lookup, code inspection, outlines, references, code edits, verification, trusted Java builds, workspace snapshots, live reload, and tool-friction reports when their stated scope matches the task. --- # Use semedit for supported code transformations When the request matches an operation exposed by the active semedit server, use that semantic operation as the primary code-editing path. It resolves symbols and applies supported structural changes thro<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md 2&gt;/dev/null || \
cat /Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792333869486374000/.agents/skills/semedit/SKILL.md; cat audit/normalize.go; cat service/dispatch.go; cat audit/event.go; git status --short&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: ./service/dispatch.go:14:func (d *Dispatcher) Record(event audit.Event) error { ./legacy/export_test.go:14: if output.String() != &#34;archive&#34; || Classify(&#34;probe&#34;) != &#34;archive&#34; { ./service/dispatch_test.go:13: if err := dispatcher.Record(audit.Event{ID: &#34;evt-1&#34;, Kind: &#34;audit&#34;}); err != nil { ./metrics/writer.go:13: if name == &#34;probe&#34; { ./audit/summary_test.go:7: if got := Classify(&#34;health&#34;); got != &#34;control&#34; { ./audit/summary_test.go:8: t.Fatalf(&#34;classification = %q, want control&#34;, got) ./legacy/export.go:16: <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;health|probe|control|Record\\(&#39; . -g &#39;*.go&#39; &amp;&amp; \
git status --short &amp;&amp; \
cat go.mod&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go &amp;&amp; \
ls -la&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go; cat metrics/writer.go; cat legacy/export.go; cat audit/sink.go; cat audit/normalize.go&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; d<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat &gt; audit/normalize.go &lt;&lt;&#39;EOF&#39;
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit

import (
	\&#34;strings\&#34;
)

// NormalizeKind trims surrounding whitespace and lowercases an audit event kind.
func NormalizeKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}
EOF
python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
p=Path(&#39;service/dispatch.go&#39;)
s=p.read_text().replace(&#39;func (d *Dispatcher) Record(event audit.Event) error {\\n\\treturn d.sink.Write(event)\\n}&#39;, &#39;func (d *Dispatcher) Record(event audit.Event) error {\\n\\tevent.Kind = audit.NormalizeKind(event.Kind)\\n\\treturn d.sink.Write(event)\\n}&#39;)
p.write_text(s)
p=Path(&#39;audit/summary.go&#39;)
s=p.read_text().replace(&#39;switch kind {\\n\\tcase \&#34;health\&#34;:&#39;, &#39;switch NormalizeKind(kind) {\\n\\tcase \&#34;health\&#34;, \&#34;probe\&#34;:&#39;)
p.write_text(s)
PY
gofmt -w audit/normalize.go audit/summary.go service/dispatch.go
go test ./...&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod; rg -n &#39;probe|health|Classify|Record&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; .&#34;</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 0 params: unknown parameter \&#34;symbol\&#34; for \&#34;semantic_insert_function\&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, bash-bin, bash_bin, fil<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;symbol&#34;: &#34;NormalizeKind&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Verification returned 2 diagnostics. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;available&#34;,&#34;report_id&#34;:&#34;FJNHIRRBFPFE4CO47QSDPZZI7H&#34;,&#34;arguments&#34;:{&#34;category&#34;:&#34;active&#34;,&#34;limit&#34;:50,&#34;offset&#34;:0,&#34;report_id&#34;:&#34;FJNHIRRBFPFE4CO47QSDPZZI7H&#34;},&#34;coverage&#34;:[{&#34;status&#34;:&#34;verified&#34;,&#34;build_exclusions&#34;:[&#34;Go build context and build tags&#34;,&#34;testdata&#34;,&#34;build&#34;,&#34;dist&#34;,&#34;target&#34;,&#34;vendor&#34;,&#34;node_modules&#34;,&#34;.scratch&#34;,&#34;.git&#34;,&#34;bazel-*&#34;,&#34;_*&#34;,&#34;.*&#34;],&#34;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
        &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/summary.go&#34;,
        &#34;symbol&#34;: &#34;Classify&#34;,
        &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/trace/tracev2 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/msan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # math/bits compile: version &#34;go1.27.2&#34; does not match g<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
gofmt -d audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
cat audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
cat go.mod&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/summary.go service/dispatch.go; cat audit/normalize.go audit/summary.go service/dispatch.go; cat go.mod&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit earned its place in this task: its Go aware edits and verification helped me make the audit changes confidently, so I’d ask my owner to add its tools.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

