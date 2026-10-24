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

* **Date**: 2026-10-17 21:43:42 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-6-luna/low (repeat 2)`

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
| **Wall-Clock Latency** | 19.98s | 22.25s | <span class="benchmark-delta-negative">+11.3%</span> | — | — | — |
| **Process Start → First Event** | 0.03s | 0.03s | — | — | — | — |
| **First Event → First Tool Call** | 3.43s | 3.48s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 8 | 5 | <span class="benchmark-delta-positive">-37.5%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 5 | <span class="benchmark-delta-positive">-37.5%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.003930 | $0.007096 | +80.6% | — | — | — |
| **Output Tokens** | 827 | 982 | <span class="benchmark-delta-negative">+18.7%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 162573 | 305438 | <span class="benchmark-delta-negative">+87.9%</span> | — | — | — |
| **Cached Input Tokens** | 141568 | 265984 | <span class="benchmark-delta-positive">+87.9%</span> | — | — | — |
| **Uncached Input Tokens** | 21005 | 39454 | <span class="benchmark-delta-negative">+87.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.74:1 | 6.74:1 | <span class="benchmark-delta-positive">+0.0%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 6 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;audit/**&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg -n \&#34;func \\(.*Dispatcher\\) Record|func Classify|NormalizeKind|&#34;&#39;^import|&#39;&#39;^package&#34; --glob &#39;&#34;&#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit // Package audit defines the event delivery boundary used by the fixture. package audit type Event struct { ID string `json:&#34;id&#34;` Kind string `json:&#34;kind&#34;` } // Package audit exposes the established event sink contract and a memory implementation. package audit type Sink interface { Write(Event) error } type TransactionalSink interface { Sink Begin() error Commit() error } type MemorySink struct { Events []Eve<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/summary.go service/dispatch.go&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files&#39;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch.go &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
cat AGENTS.override.md &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go &amp;&amp; \
cat service/dispatch_test.go &amp;&amp; \
cat audit/event_test.go&#39;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;.&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;function&#34;,
      &#34;source&#34;: &#34;func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) }&#34;,
      &#34;placement&#34;: &#34;file_end&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;file&#34;: &#34;service/dispatch.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;function&#34;,
      &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
      &#34;source&#34;: &#34;func (d *Dispatcher) Record(event audit.Event) error {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)\n}&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;file&#34;: &#34;audit/summary.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;function&#34;,
      &#34;symbol&#34;: &#34;Classify&#34;,
      &#34;source&#34;: &#34;func Classify(kind string) string {\n\tswitch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}\n}&#34;
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;file&#34;: &#34;audit/normalize.go&#34;,
      &#34;language&#34;: &#34;go&#34;
    }
  ]
}</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/goexperiment compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # crypto/internal/fips140/alias compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # encoding compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # crypto/internal/constanttime compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf16 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # crypto/internal/fips140deps/time compile: vers<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
sha256sum go.mod&#39;</code></pre></td><td>—</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;sha256sum go.mod; git -C . status --short 2&gt;/dev/null || \
true; git status --short 2&gt;/dev/null || \
true; go version&#39;</code></pre></td><td>—</td></tr>
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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 19.98s | 22.37s | N/A | — | — | — |
| **Process Start → First Event** | 0.03s | 0.02s | — | — | — | — |
| **First Event → First Tool Call** | 3.43s | 3.49s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 8 | 8 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 1 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 8 | N/A | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.003930 | $0.004047 | +3.0% | — | — | — |
| **Output Tokens** | 827 | 932 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 162573 | 164954 | N/A | — | — | — |
| **Cached Input Tokens** | 141568 | 143488 | N/A | — | — | — |
| **Uncached Input Tokens** | 21005 | 21466 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.74:1 | 6.68:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 10 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;audit/**&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;audit/**&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | sort&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit // Package audit defines the event delivery boundary used by the fixture. package audit type Event struct { ID string `json:&#34;id&#34;` Kind string `json:&#34;kind&#34;` } // Package audit exposes the established event sink contract and a memory implementation. package audit type Sink interface { Write(Event) error } type TransactionalSink interface { Sink Begin() error Commit() error } type MemorySink struct { Events []Eve<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat PROJECT_BRIEF.md 2&gt;/dev/null; cat audit/normalize.go; cat audit/sink.go; rg -n &#34;Dispatcher|Record|probe|health|Kind&#34; audit&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: ./service/dispatch.go:6:type Dispatcher struct { ./service/dispatch.go:10:func NewDispatcher(sink audit.Sink) *Dispatcher { ./service/dispatch.go:11: return &amp;Dispatcher{sink: sink} ./service/dispatch.go:14:func (d *Dispatcher) Record(event audit.Event) error { ./audit/summary_test.go:7: if got := Classify(&#34;health&#34;); got != &#34;control&#34; { ./service/dispatch_test.go:12: dispatcher := NewDispatcher(sink) ./service/dispatch_test.go:13: if err := dispatcher.Record(audit.Event{ID: &#34;evt-1&#34;, Kind: &#34;audit&#34;}); err != ni<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Dispatcher|Record\\(|Classify\\(\&#34; . -g &#39;*.go&#39;; cat audit/summary.go; cat audit/event.go; rg --files -g &#39;AGENTS.md&#39; -g &#39;*semedit*&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; | sort; git status --short&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch.go &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
cat AGENTS.override.md &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch.go; command -v semedit || \
true; ls -la&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go &amp;&amp; \
cat service/dispatch_test.go &amp;&amp; \
cat audit/event_test.go&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: .scratch/go/build/README .scratch/go/tmp/xcrun_db<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;find .scratch -maxdepth 3 -type f -print; rg -n &#34;semedit&#34; .&#39;</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/goexperiment compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # crypto/internal/fips140/alias compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # encoding compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # crypto/internal/constanttime compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf16 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # crypto/internal/fips140deps/time compile: vers<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./... &amp;&amp; \
sha256sum go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # crypto/internal/constanttime compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf16 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # sync/atomic compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/cpu compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # crypto/internal/boring/sig compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; FAIL example.com/auditapp/audit [build failed] FAIL examp<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;sha256sum go.mod; git -C . status --short 2&gt;/dev/null || \
true; git status --short 2&gt;/dev/null || \
true; go version&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short 2&gt;/dev/null || \
true; cat audit/normalize.go; cat audit/summary.go; cat service/dispatch.go; sha256sum go.mod; rg -n &#39;&#34;&#39;^func NormalizeKind|NormalizeKind&#39;&#34;\\(event.Kind\\)|case \&#34;health\&#34;, \&#34;probe\&#34;&#39; audit service&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Tool-Use Reflection

No semantic MCP invocation was confirmed during the task. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why no semantic edit tool was used</div><div class="callout-desc">I did not discover the semedit MCP server’s `semantic_*` tools in the available tool list, so I used ordinary file editing instead.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. In one to three sentences, explain why you did not call any semantic_* tool from the available semedit MCP server while completing this task. State whether you did not discover the tools, judged ordinary editing simpler, could not use the server, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I couldn’t assess semedit’s coding value because I didn’t discover or use its tools in this run, so I can’t honestly recommend asking my owner to add them.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

