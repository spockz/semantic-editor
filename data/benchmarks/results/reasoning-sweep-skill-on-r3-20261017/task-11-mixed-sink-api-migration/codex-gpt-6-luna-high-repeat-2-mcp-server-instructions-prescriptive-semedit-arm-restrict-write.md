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

* **Date**: 2026-10-18 06:58:33 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-6-luna/high (repeat 2)`

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
| **Wall-Clock Latency** | 27.10s | 51.61s | <span class="benchmark-delta-negative">+90.4%</span> | — | — | — |
| **Process Start → First Event** | 0.17s | 0.16s | — | — | — | — |
| **First Event → First Tool Call** | 3.38s | 3.80s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 9 | 18 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 9 | 18 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.005203 | $0.011465 | +120.4% | — | — | — |
| **Output Tokens** | 2030 | 4187 | <span class="benchmark-delta-negative">+106.3%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 947 | 2330 | <span class="benchmark-delta-negative">+146.0%</span> | — | — | — |
| **Total Input Tokens** | 196711 | 619606 | <span class="benchmark-delta-negative">+215.0%</span> | — | — | — |
| **Cached Input Tokens** | 172032 | 584320 | <span class="benchmark-delta-positive">+239.7%</span> | — | — | — |
| **Uncached Input Tokens** | 24679 | 35286 | <span class="benchmark-delta-negative">+43.0%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.97:1 | 16.56:1 | <span class="benchmark-delta-positive">+137.6%</span> | — | — | — |
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
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: 2026-10-18 01:07:50.604 xcodebuild[86224:24025981] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 01:07:50.701 xcodebuild[86224:24025980] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;ls -la &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792278408979913000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files docs audit service metrics legacy cmd &amp;&amp; \
cat docs/*&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792278408979913000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go &amp;&amp; \
cat service/dispatch_test.go &amp;&amp; \
cat audit/summary_test.go&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792278408979913000&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792278408979913000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not m<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: cat: .agents/skills/semedit/references/golang.md: No such file or directory<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/references/golang.md&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GOCACHE=&#34;$PWD/.scratch/go/tmp/gocache&#34; go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd &amp;&amp; \
rg --files .agents&#39;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit import &#34;strings&#34; func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) } // Package audit groups delivery kinds for operational reporting. package audit func Classify(kind string) string { switch NormalizeKind(kind) { case &#34;health&#34;, &#34;probe&#34;: return &#34;control&#34; default: return &#34;data&#34; } } // Package service coordinates application work with the configured audit sink. package serv<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -d audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
sed -n &#39;1,24p&#39; service/dispatch.go &amp;&amp; \
ls -l go.mod go.sum&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;semantic_insert_function|semantic_replace_body|semantic_organize_imports|semantic_batch&#34; .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;(&#34;&#39;^|/)PROJECT_BRIEF&#39;&#34;\\.md&#34;&#39;$|(&#39;&#39;^|/)go&#39;&#34;\\.mod&#34;&#39;$|audit/.*&#39;&#34;\\.go&#34;&#39;$|service/dispatch&#39;&#34;\\.go&#34;&#39;$&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Verification returned 2 diagnostics. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;available&#34;,&#34;report_id&#34;:&#34;MC4DOUFRWBLZDJTS72XH7VCV3V&#34;,&#34;arguments&#34;:{&#34;category&#34;:&#34;active&#34;,&#34;limit&#34;:50,&#34;offset&#34;:0,&#34;report_id&#34;:&#34;MC4DOUFRWBLZDJTS72XH7VCV3V&#34;},&#34;coverage&#34;:[{&#34;status&#34;:&#34;verified&#34;,&#34;build_exclusions&#34;:[&#34;Go build context and build tags&#34;,&#34;testdata&#34;,&#34;build&#34;,&#34;dist&#34;,&#34;target&#34;,&#34;vendor&#34;,&#34;node_modules&#34;,&#34;.scratch&#34;,&#34;.git&#34;,&#34;bazel-*&#34;,&#34;_*&#34;,&#34;.*&#34;],&#34;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792278408979913000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) }&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792278408979913000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
        &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792278408979913000&#34;,
  &#34;edits&#34;: [
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
<tr><td>15</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-11-mixed-sink-api-migration_1792278408979913000&#34;,
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
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: 2026-10-18 01:07:34.790 xcodebuild[85525:24020446] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 01:07:34.901 xcodebuild[85525:24020444] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. warning: Limiting comparison with pathspecs is only supported if both paths are directories. usage: git diff --no-index [&lt;options&gt;] &lt;path&gt; &lt;path&gt; [&lt;pathspec&gt;...] Diff <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go service/dispatch.go audit/summary.go metrics legacy go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: warning: Limiting comparison with pathspecs is only supported if both paths are directories. usage: git diff --no-index [&lt;options&gt;] &lt;path&gt; &lt;path&gt; [&lt;pathspec&gt;...] Diff output format options -p, --patch generate patch -s, --no-patch suppress diff output -u generate patch -U, --unified[=&lt;n&gt;] generate diffs with &lt;n&gt; lines context -W, --[no-]function-context generate diffs with &lt;n&gt; lines context --raw generate the diff in raw format --patch-with-raw synonym for &#39;-p --raw&#39; --patch-with-stat synonym for &#39;-p --stat<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go service/dispatch.go audit/summary.go&#39;</code></pre></td></tr>
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
| **Wall-Clock Latency** | 27.10s | 44.17s | <span class="benchmark-delta-negative">+63.0%</span> | — | — | — |
| **Process Start → First Event** | 0.17s | 0.15s | — | — | — | — |
| **First Event → First Tool Call** | 3.38s | 6.78s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 9 | 16 | <span class="benchmark-delta-negative">+77.8%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 1 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 9 | 16 | <span class="benchmark-delta-negative">+77.8%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.005203 | $0.010308 | +98.1% | — | — | — |
| **Output Tokens** | 2030 | 3679 | <span class="benchmark-delta-negative">+81.2%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 947 | 1345 | <span class="benchmark-delta-negative">+42.0%</span> | — | — | — |
| **Total Input Tokens** | 196711 | 558620 | <span class="benchmark-delta-negative">+184.0%</span> | — | — | — |
| **Cached Input Tokens** | 172032 | 526592 | <span class="benchmark-delta-positive">+206.1%</span> | — | — | — |
| **Uncached Input Tokens** | 24679 | 32028 | <span class="benchmark-delta-negative">+29.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.97:1 | 16.44:1 | <span class="benchmark-delta-positive">+135.9%</span> | — | — | — |
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
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: /Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000 audit/summary.go audit/normalize.go audit/event.go audit/sink.go AGENTS.override.md go.mod 2026-10-18 01:06:08.565 xcodebuild[84544:24011807] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 01:06:08.696 xcodebuild[84544:24011805] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output erro<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;audit/*.go&#39; -g &#39;go.mod&#39; -g &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: 2026-10-18 01:07:50.604 xcodebuild[86224:24025981] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 01:07:50.701 xcodebuild[86224:24025980] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;ls -la &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat AGENTS.override.md&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files docs audit service metrics legacy cmd &amp;&amp; \
cat docs/*&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go &amp;&amp; \
cat service/dispatch_test.go &amp;&amp; \
cat audit/summary_test.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &#39;Dispatcher|func \\(.*Record|probe|health|control&#39; .&#34;</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not m<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;param \&#34;symbol\&#34; is required: invalid operation parameters&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GOCACHE=&#34;$PWD/.scratch/go/tmp/gocache&#34; go test ./...&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: // Package audit leaves room for normalization rules owned by the delivery boundary. package audit import &#34;strings&#34; func NormalizeKind(kind string) string { return strings.ToLower(strings.TrimSpace(kind)) } // Package audit groups delivery kinds for operational reporting. package audit func Classify(kind string) string { switch NormalizeKind(kind) { case &#34;health&#34;, &#34;probe&#34;: return &#34;control&#34; default: return &#34;data&#34; } } // Package service coordinates application work with the configured audit sink. package serv<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -d audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
sed -n &#39;1,24p&#39; service/dispatch.go &amp;&amp; \
ls -l go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Record&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 0: tool is not batchable: mcp__semedit__semantic_insert_construct&#34;,&#34;tool&#34;:&#34;mcp__semedit__semantic_insert_construct&#34;}],&#34;workspace_root&#34;:&#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;mcp__semedit__semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;placement&#34;: &#34;file_end&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
        &#34;auto_organize_imports&#34;: true,
        &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;mcp__semedit__semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
        &#34;source&#34;: &#34;func (d *Dispatcher) Record(event audit.Event) error {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)\n}&#34;,
        &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;mcp__semedit__semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/summary.go&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;symbol&#34;: &#34;Classify&#34;,
        &#34;source&#34;: &#34;func Classify(kind string) string {\n\tswitch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}\n}&#34;,
        &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;source&#34;: &#34;func (d *Dispatcher) Record(event audit.Event) error {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\treturn d.sink.Write(event)\n}&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;source&#34;: &#34;func Classify(kind string) string {\n\tswitch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}\n}&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat go.mod &amp;&amp; \
rg --files -g &#39;*_test.go&#39;&#34;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # math/bits compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # cmp compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool ve<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
go test ./...&#39;</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;check_only&#34;: true,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-11-mixed-sink-api-migration_1792278359560641000&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools: I felt more confident making focused Go edits with symbol-aware inspection and diagnostic checks, though the batch edit I tried wasn’t supported and the test run hit a Go toolchain mismatch.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

