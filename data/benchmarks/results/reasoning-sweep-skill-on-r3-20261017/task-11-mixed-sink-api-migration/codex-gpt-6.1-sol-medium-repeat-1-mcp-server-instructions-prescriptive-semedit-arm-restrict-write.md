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

* **Date**: 2026-10-18 06:58:32 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-6.1-sol/medium (repeat 1)`

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
| **Wall-Clock Latency** | 37.72s | 28.05s | <span class="benchmark-delta-positive">-25.6%</span> | — | — | — |
| **Process Start → First Event** | 0.16s | 0.16s | — | — | — | — |
| **First Event → First Tool Call** | 4.81s | 4.40s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 7 | 11 | <span class="benchmark-delta-negative">+57.1%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 11 | <span class="benchmark-delta-negative">+57.1%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.059444 | $0.100231 | +68.6% | — | — | — |
| **Output Tokens** | 936 | 739 | <span class="benchmark-delta-positive">-21.0%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 211 | 50 | <span class="benchmark-delta-positive">-76.3%</span> | — | — | — |
| **Total Input Tokens** | 184338 | 224443 | <span class="benchmark-delta-negative">+21.8%</span> | — | — | — |
| **Cached Input Tokens** | 167680 | 187392 | <span class="benchmark-delta-positive">+11.8%</span> | — | — | — |
| **Uncached Input Tokens** | 16658 | 37051 | <span class="benchmark-delta-negative">+122.4%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 10.07:1 | 5.06:1 | <span class="benchmark-delta-negative">-49.8%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 9 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 6 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;AGENTS*&#39; -g &#39;*BRIEF*&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; -g &#39;*.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short; cat AGENTS.override.md docs/audit-sinks.md audit/normalize.go audit/event.go audit/sink.go service/dispatch.go audit/summary.go; cat service/dispatch_test.go audit/event_test.go audit/summary_test.go go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;*.md&#39; -g &#39;Makefile&#39; -g &#39;.semedit*&#39; -g &#39;&#34;&#39;! .git&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # cmp compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/trace/tracev2 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode compile: version &#34;go1.27.2&#34; does not match go tool versi<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
p = Path(&#39;audit/normalize.go&#39;)
p.write_text(&#39;&#39;&#39;// Package audit owns normalization rules shared by delivery and reporting.
package audit

import \&#34;strings\&#34;

// NormalizeKind returns a trimmed, lowercase event kind.
func NormalizeKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}
&#39;&#39;&#39;)
p = Path(&#39;service/dispatch.go&#39;)
s = p.read_text().replace(&#39;func (d *Dispatcher) Record(event audit.Event) error {\\n&#39;, &#39;func (d *Dispatcher) Record(event audit.Event) error {\\n\\tevent.Kind = audit.NormalizeKind(event.Kind)\\n&#39;)
p.write_text(s)
p = Path(&#39;audit/summary.go&#39;)
s = p.read_text().replace(&#39;switch kind {&#39;, &#39;switch NormalizeKind(kind) {&#39;).replace(&#39;case \&#34;health\&#34;:&#39;, &#39;case \&#34;health\&#34;, \&#34;probe\&#34;:&#39;)
p.write_text(s)
PY

gofmt -w audit/normalize.go audit/summary.go service/dispatch.go
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/profilerecord compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/msan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/trace/tracev2 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; doe<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GOCACHE=&#34;$PWD/.scratch/go/cache&#34; go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md AGENTS.override.md&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files --hidden -g &#39;&#34;&#39;!**/cache/**&#39;&#34;&#39; -g &#39;&#34;&#39;!**/.git/**&#39;&#34;&#39;; go version; go env GOROOT GOTOOLDIR GOFLAGS GOTOOLCHAIN; command -v go&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: 2026-10-18 01:17:52.523 xcodebuild[3275:24161253] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 01:17:52.619 xcodebuild[3275:24161252] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short&#39;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: signal: killed FAIL example.com/auditapp/audit 0.402s ? example.com/auditapp/cmd/migrate [no test files] signal: killed FAIL example.com/auditapp/legacy 0.654s signal: killed FAIL example.com/auditapp/metrics 2.532s signal: killed FAIL example.com/auditapp/service 1.514s FAIL<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;GOROOT= GOFLAGS=&#39;-mod=readonly -buildvcs=false&#39; GOCACHE=\&#34;&#34;&#39;$PWD/.scratch/go/cache&#34; go test ./...&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;audit.Classify&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GOROOT= GOCACHE=&#34;$PWD/.scratch/go/cache&#34; go test ./...&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: false,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) }&#34;
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
        &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\nreturn \&#34;control\&#34;\ndefault:\nreturn \&#34;data\&#34;\n}&#34;
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
| **Wall-Clock Latency** | 37.72s | 39.74s | <span class="benchmark-delta-negative">+5.3%</span> | — | — | — |
| **Process Start → First Event** | 0.16s | 0.16s | — | — | — | — |
| **First Event → First Tool Call** | 4.81s | 5.19s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 7 | 16 | <span class="benchmark-delta-negative">+128.6%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 16 | <span class="benchmark-delta-negative">+128.6%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.059444 | $0.156254 | +162.9% | — | — | — |
| **Output Tokens** | 936 | 1218 | <span class="benchmark-delta-negative">+30.1%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 211 | 116 | <span class="benchmark-delta-positive">-45.0%</span> | — | — | — |
| **Total Input Tokens** | 184338 | 556978 | <span class="benchmark-delta-negative">+202.2%</span> | — | — | — |
| **Cached Input Tokens** | 167680 | 510464 | <span class="benchmark-delta-positive">+204.4%</span> | — | — | — |
| **Uncached Input Tokens** | 16658 | 46514 | <span class="benchmark-delta-negative">+179.2%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 10.07:1 | 10.97:1 | <span class="benchmark-delta-positive">+9.0%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 9 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;AGENTS*&#39; -g &#39;*BRIEF*&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; -g &#39;go.mod&#39; -g &#39;*.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*README*&#39; -g &#39;*Makefile*&#39; -g &#39;*.go&#39; -g &#39;SKILL.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short; cat AGENTS.override.md docs/audit-sinks.md audit/normalize.go audit/event.go audit/sink.go service/dispatch.go audit/summary.go; cat service/dispatch_test.go audit/event_test.go audit/summary_test.go go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # cmp compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/trace/tracev2 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode compile: version &#34;go1.27.2&#34; does not match go tool versi<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
p = Path(&#39;audit/normalize.go&#39;)
p.write_text(&#39;&#39;&#39;// Package audit owns normalization rules shared by delivery and reporting.
package audit

import \&#34;strings\&#34;

// NormalizeKind returns a trimmed, lowercase event kind.
func NormalizeKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}
&#39;&#39;&#39;)
p = Path(&#39;service/dispatch.go&#39;)
s = p.read_text().replace(&#39;func (d *Dispatcher) Record(event audit.Event) error {\\n&#39;, &#39;func (d *Dispatcher) Record(event audit.Event) error {\\n\\tevent.Kind = audit.NormalizeKind(event.Kind)\\n&#39;)
p.write_text(s)
p = Path(&#39;audit/summary.go&#39;)
s = p.read_text().replace(&#39;switch kind {&#39;, &#39;switch NormalizeKind(kind) {&#39;).replace(&#39;case \&#34;health\&#34;:&#39;, &#39;case \&#34;health\&#34;, \&#34;probe\&#34;:&#39;)
p.write_text(s)
PY

gofmt -w audit/normalize.go audit/summary.go service/dispatch.go
go test ./...&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/profilerecord compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/msan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/trace/tracev2 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; doe<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GOCACHE=&#34;$PWD/.scratch/go/cache&#34; go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files --hidden -g &#39;&#34;&#39;!&#39;&#34;\\.git/**&#39; -g &#39;&#34;&#39;!&#39;&#34;\\.agents/**&#39; -g &#39;&#34;&#39;!&#39;&#34;\\.scratch/**&#39; -g &#39;&#34;&#39;!&#39;&#34;\\.codex/**&#39;&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files --hidden -g &#39;&#34;&#39;!**/cache/**&#39;&#34;&#39; -g &#39;&#34;&#39;!**/.git/**&#39;&#34;&#39;; go version; go env GOROOT GOTOOLDIR GOFLAGS GOTOOLCHAIN; command -v go&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: signal: killed FAIL example.com/auditapp/audit 0.402s ? example.com/auditapp/cmd/migrate [no test files] signal: killed FAIL example.com/auditapp/legacy 0.654s signal: killed FAIL example.com/auditapp/metrics 2.532s signal: killed FAIL example.com/auditapp/service 1.514s FAIL<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;GOROOT= GOFLAGS=&#39;-mod=readonly -buildvcs=false&#39; GOCACHE=\&#34;&#34;&#39;$PWD/.scratch/go/cache&#34; go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: cat: .agents/skills/semedit/references/golang.md: No such file or directory # Audit delivery The service records operational events at a package boundary so deployments can choose an appropriate destination. Event labels arrive from command-line flags, hooks, and external integrations, so they are not always formatted consistently. Metrics collection and archival export use separate protocols. Changes to the audit boundary should avoid coupling those components to event persistence details. # Benchmark fixt<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md .agents/skills/semedit/references/golang.md AGENTS.override.md&#39;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GOROOT= GOCACHE=&#34;$PWD/.scratch/go/cache&#34; go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: 2026-10-18 01:17:08.436 xcodebuild[1822:24149603] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 01:17:08.532 xcodebuild[1822:24149601] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
cat audit/normalize.go&#39;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go service/dispatch_test.go audit/event_test.go audit/summary_test.go&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Verification returned 2 diagnostics. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;available&#34;,&#34;report_id&#34;:&#34;MROHLX7VIHT2JGBX6WA6TENSU4&#34;,&#34;arguments&#34;:{&#34;category&#34;:&#34;active&#34;,&#34;limit&#34;:50,&#34;offset&#34;:0,&#34;report_id&#34;:&#34;MROHLX7VIHT2JGBX6WA6TENSU4&#34;},&#34;coverage&#34;:[{&#34;status&#34;:&#34;verified&#34;,&#34;build_exclusions&#34;:[&#34;Go build context and build tags&#34;,&#34;testdata&#34;,&#34;build&#34;,&#34;dist&#34;,&#34;target&#34;,&#34;vendor&#34;,&#34;node_modules&#34;,&#34;.scratch&#34;,&#34;.git&#34;,&#34;bazel-*&#34;,&#34;_*&#34;,&#34;.*&#34;],&#34;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\nreturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;kind&#34;: &#34;statement/expression&#34;,
        &#34;in_function&#34;: &#34;Dispatcher.Record&#34;,
        &#34;anchor&#34;: &#34;start&#34;,
        &#34;source&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)&#34;,
        &#34;expected_revision&#34;: &#34;550fd5283aef8cc729610e7e44cbc24e1e152985732114c581e243ab32319079&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/summary.go&#34;,
        &#34;symbol&#34;: &#34;Classify&#34;,
        &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\nreturn \&#34;control\&#34;\ndefault:\nreturn \&#34;data\&#34;\n}&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;add&#34;: [
          &#34;strings&#34;
        ]
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/msan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not matc<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test -mod=readonly ./...&#39;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;env -u GOROOT go test -mod=readonly ./...&#39;</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go audit/summary.go service/dispatch.go&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit: its targeted edits and diagnostic checks gave me confidence in this change, though fixing the missing import took an extra step.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

