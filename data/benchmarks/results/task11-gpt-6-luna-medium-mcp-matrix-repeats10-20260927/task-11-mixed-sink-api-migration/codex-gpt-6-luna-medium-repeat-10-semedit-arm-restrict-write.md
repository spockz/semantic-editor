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

* **Date**: 2026-09-27 16:41:16 CEST

## Test case: `task-11-mixed-sink-api-migration`

### Target: `codex/gpt-6-luna/medium (repeat 10)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/task_11_mixed_sink_api_migration.txtar](https://github.com/spockz/semantic-editor/blob/83d88a5482d8614eee37246b1ddc53ebb78d4435/testdata/bench/task_11_mixed_sink_api_migration.txtar)

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
| **Wall-Clock Latency** | 59.58s | 172.98s | <span class="benchmark-delta-negative">+190.3%</span> | 30.00s | 88.29s | <span class="benchmark-delta-negative">+194.2%</span> |
| **Process Start → First Event** | 0.12s | 0.12s | — | 0.13s | 0.15s | — |
| **First Event → First Tool Call** | 7.52s | 21.44s | — | 4.86s | 5.39s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 7 | 14 | <span class="benchmark-delta-negative">+100.0%</span> | 6 | 13 | <span class="benchmark-delta-negative">+116.7%</span> |
| **Initial Load / Discovery Turns** | 1 | 3 | <span class="benchmark-delta-negative">+200.0%</span> | 2 | 2 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 7 | 14 | <span class="benchmark-delta-negative">+100.0%</span> | 6 | 13 | <span class="benchmark-delta-negative">+116.7%</span> |
| **Output Tokens** | 1826 | 1798 | <span class="benchmark-delta-positive">-1.5%</span> | 768 | 1697 | <span class="benchmark-delta-negative">+121.0%</span> |
| **Reasoning / Thinking Tokens** | 325 | 461 | <span class="benchmark-delta-negative">+41.8%</span> | 0 | 708 | <span class="benchmark-delta-negative">+100.0%</span> |
| **Total Input Tokens** | 165073 | 236915 | <span class="benchmark-delta-negative">+43.5%</span> | 160328 | 297031 | <span class="benchmark-delta-negative">+85.3%</span> |
| **Cached Input Tokens** | 149504 | 215552 | <span class="benchmark-delta-positive">+44.2%</span> | 146432 | 265728 | <span class="benchmark-delta-positive">+81.5%</span> |
| **Uncached Input Tokens** | 15569 | 21363 | <span class="benchmark-delta-negative">+37.2%</span> | 13896 | 31303 | <span class="benchmark-delta-negative">+125.3%</span> |
| **Cached vs Uncached Token Ratio** | 9.60:1 | 10.09:1 | <span class="benchmark-delta-positive">+5.1%</span> | 10.54:1 | 8.49:1 | <span class="benchmark-delta-negative">-19.4%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 10 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;audit/**&#39; -g &#39;*test.go&#39; | head -200 &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;audit/**&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/normalize.go &amp;&amp; \
sed -n &#39;1,240p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,200p&#39; audit/event.go &amp;&amp; \
rg -n \&#34;Record\\(|probe|health|NormalizeKind\&#34; . --glob &#39;&#34;&#39;!**/*test.go&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/summary.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go &amp;&amp; \
cat Makefile 2&gt;/dev/null || \
true &amp;&amp; \
rg --files -g &#39;README*&#39; -g &#39;AGENTS.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat audit/event.go&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;ls -la &amp;&amp; \
cat go.mod &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Dispatcher|Record\\(|probe|health|control\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile

updates = {
    Path(&#39;audit/normalize.go&#39;): &#39;&#39;&#39;// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit

import (
\\t\&#34;strings\&#34;
)

func NormalizeKind(kind string) string {
\\treturn strings.ToLower(strings.TrimSpace(kind))
}
&#39;&#39;&#39;,
    Path(&#39;service/dispatch.go&#39;): &#39;&#39;&#39;// Package service coordinates application work with the configured audit sink.
package service

import \&#34;example.com/auditapp/audit\&#34;

type Dispatcher struct {
\\tsink audit.Sink
}

func NewDispatcher(sink audit.Sink) *Dispatcher {
\\treturn &amp;Dispatcher{sink: sink}
}

func (d *Dispatcher) Record(event audit.Event) error {
\\tevent.Kind = audit.NormalizeKind(event.Kind)
\\treturn d.sink.Write(event)
}

func DispatchAll(sink audit.Sink, events []audit.Event) error {
\\twriteEvent := audit.Sink.Write
\\tfor _, event := range events {
\\t\\tif err := writeEvent(sink, event); err &#34;&#39;!= nil {
&#39;&#34;\\t\\t\\treturn err
\\t\\t}
\\t}
\\treturn nil
}
&#39;&#39;&#39;,
    Path(&#39;audit/summary.go&#39;): &#39;&#39;&#39;// Package audit groups delivery kinds for operational reporting.
package audit

func Classify(kind string) string {
\\tswitch NormalizeKind(kind) {
\\tcase \&#34;health\&#34;, \&#34;probe\&#34;:
\\t\\treturn \&#34;control\&#34;
\\tdefault:
\\t\\treturn \&#34;data\&#34;
\\t}
}
&#39;&#39;&#39;,
}

for path, content in updates.items():
    fd, temp_name = tempfile.mkstemp(dir=path.parent, prefix=f&#39;.{path.name}.&#39;)
    try:
        with os.fdopen(fd, &#39;w&#39;) as temp_file:
            temp_file.write(content)
            temp_file.flush()
            os.fsync(temp_file.fileno())
        os.replace(temp_name, path)
    except BaseException:
        try:
            os.unlink(temp_name)
        except FileNotFoundError:
            pass
        raise
PY
gofmt -w audit/normalize.go audit/summary.go service/dispatch.go
git diff -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Record&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/normalize.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
sed -n &#39;1,30p&#39; service/dispatch.go &amp;&amp; \
git status --short &amp;&amp; \
git diff --no-index /dev/null audit/normalize.go&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff --check &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md &amp;&amp; \
cat Makefile&#39;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git diff --stat &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
git diff --exit-code -- go.mod go.sum &amp;&amp; \
git diff --name-only -- &#39;*_test.go&#39;&#34;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum &amp;&amp; \
git diff --exit-code -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 10 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;audit/**&#39; | sort &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
rg -n \&#34;func \\(.*Record|Record\\(|health|probe|Kind\&#34; audit&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
rg -n &#39;probe|health|Record\\(&#39; audit service&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;type Dispatcher|func .*Record|Classify|legacy|metrics\&#34; . -g &#39;*.go&#39; -g &#39;*.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go; cat audit/summary.go; cat service/dispatch.go; cat Makefile&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat service/dispatch.go audit/summary.go audit/event.go &amp;&amp; \
rg -n &#34;make check|func main|go test&#34; Makefile&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Record&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files | sort&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git diff -- audit/normalize.go service/dispatch.go audit/summary.go &amp;&amp; \
git status --short -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): { &#34;status&#34;: &#34;error&#34;, &#34;results&#34;: [ { &#34;tool&#34;: &#34;semantic_insert_function&#34;, &#34;status&#34;: &#34;ok&#34; }, { &#34;tool&#34;: &#34;semantic_replace_body&#34;, &#34;symbol&#34;: &#34;Classify&#34;, &#34;status&#34;: &#34;ok&#34;, &#34;diff&#34;: &#34;--- a/Classify\n+++ b/Classify\n-switch kind {\n-\tcase \&#34;health\&#34;:\n-\t\treturn \&#34;control\&#34;\n-\tdefault:\n-\t\treturn \&#34;data\&#34;\n-\t}\n+switch NormalizeKind(kind) {\n+case \&#34;health\&#34;, \&#34;probe\&#34;:\n+\treturn \&#34;control\&#34;\n+default:\n+\treturn \&#34;data\&#34;\n+}\n&#34; }, { &#34;tool&#34;: &#34;semantic_replace_body&#34;, &#34;status&#34;: &#34;error&#34;, &#34;error&#34;: &#34;replace_body Reco<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;access_modifier&#34;: &#34;public&#34;,
        &#34;placement&#34;: &#34;file_end&#34;,
        &#34;auto_organize_imports&#34;: true,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/summary.go&#34;,
        &#34;symbol&#34;: &#34;Classify&#34;,
        &#34;auto_organize_imports&#34;: true,
        &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;symbol&#34;: &#34;Record&#34;,
        &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files | head -60&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum audit/summary_test.go audit/event_test.go service/dispatch_test.go &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum audit/summary_test.go audit/event_test.go service/dispatch_test.go&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">`semantic_batch` was discovered before the edits, and batching was suitable for the independent function insertion and body replacements. I used consecutive calls because I chose to apply and verify each semantic edit separately; there was no technical blocker.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Configuration: prefer_discover_semedit prompt · none MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/task_11_mixed_sink_api_migration.txtar](https://github.com/spockz/semantic-editor/blob/83d88a5482d8614eee37246b1ddc53ebb78d4435/testdata/bench/task_11_mixed_sink_api_migration.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. Audit event kinds may contain inconsistent whitespace and casing. Add the public audit.NormalizeKind(kind string) string function in audit/normalize.go to remove surrounding whitespace and lowercase the kind. Apply it in Dispatcher.Record before persistence, and classify normalized probe events as control traffic, like health. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Before editing, inspect the complete available tool inventory, including deferred or lazy tools. If applicable semantic editing tools are callable, prefer them for source mutations. Audit event kinds may contain inconsistent whitespace and casing. Add the public audit.NormalizeKind(kind string) string function in audit/normalize.go to remove surrounding whitespace and lowercase the kind. Apply it in Dispatcher.Record before persistence, and classify normalized probe events as control traffic, like health. Do not change the metrics or legacy export protocols. Preserve the module metadata and tests, and leave the workspace verified. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package audit leaves room for normalization rules owned by the delivery boundary.
package audit
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 48.52s | 108.66s | <span class="benchmark-delta-negative">+124.0%</span> | 46.79s | 85.08s | <span class="benchmark-delta-negative">+81.8%</span> |
| **Process Start → First Event** | 0.13s | 0.16s | — | 0.13s | 0.13s | — |
| **First Event → First Tool Call** | 9.16s | 10.38s | — | 6.61s | 14.35s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 5 | 17 | <span class="benchmark-delta-negative">+240.0%</span> | 7 | 12 | <span class="benchmark-delta-negative">+71.4%</span> |
| **Initial Load / Discovery Turns** | 1 | 3 | <span class="benchmark-delta-negative">+200.0%</span> | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 5 | 17 | <span class="benchmark-delta-negative">+240.0%</span> | 7 | 12 | <span class="benchmark-delta-negative">+71.4%</span> |
| **Output Tokens** | 1235 | 1958 | <span class="benchmark-delta-negative">+58.5%</span> | 1746 | 1793 | <span class="benchmark-delta-negative">+2.7%</span> |
| **Reasoning / Thinking Tokens** | 273 | 633 | <span class="benchmark-delta-negative">+131.9%</span> | 427 | 813 | <span class="benchmark-delta-negative">+90.4%</span> |
| **Total Input Tokens** | 138260 | 518732 | <span class="benchmark-delta-negative">+275.2%</span> | 153130 | 409096 | <span class="benchmark-delta-negative">+167.2%</span> |
| **Cached Input Tokens** | 122368 | 473600 | <span class="benchmark-delta-positive">+287.0%</span> | 128256 | 376576 | <span class="benchmark-delta-positive">+193.6%</span> |
| **Uncached Input Tokens** | 15892 | 45132 | <span class="benchmark-delta-negative">+184.0%</span> | 24874 | 32520 | <span class="benchmark-delta-negative">+30.7%</span> |
| **Cached vs Uncached Token Ratio** | 7.70:1 | 10.49:1 | <span class="benchmark-delta-positive">+36.3%</span> | 5.16:1 | 11.58:1 | <span class="benchmark-delta-positive">+124.6%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 10 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;README.md&#39; -g &#39;normalize.go&#39; -g &#39;*.go&#39; | head -100 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat PROJECT_BRIEF.md 2&gt;/dev/null; rg --files -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/event_test.go &amp;&amp; \
cat audit/summary_test.go &amp;&amp; \
cat service/dispatch_test.go &amp;&amp; \
cat go.mod &amp;&amp; \
rg -n &#34;probe|Classify|NormalizeKind|verification|check&#34; README.md **/README.md Makefile 2&gt;/dev/null&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile

updates = {
    Path(&#39;audit/normalize.go&#39;): &#39;&#39;&#39;// Package audit owns normalization rules at the event delivery boundary.
package audit

import (
\\t\&#34;strings\&#34;
)

func NormalizeKind(kind string) string {
\\treturn strings.ToLower(strings.TrimSpace(kind))
}
&#39;&#39;&#39;,
    Path(&#39;audit/summary.go&#39;): &#39;&#39;&#39;// Package audit groups delivery kinds for operational reporting.
package audit

func Classify(kind string) string {
\\tswitch NormalizeKind(kind) {
\\tcase \&#34;health\&#34;, \&#34;probe\&#34;:
\\t\\treturn \&#34;control\&#34;
\\tdefault:
\\t\\treturn \&#34;data\&#34;
\\t}
}
&#39;&#39;&#39;,
    Path(&#39;service/dispatch.go&#39;): &#39;&#39;&#39;// Package service coordinates application work with the configured audit sink.
package service

import \&#34;example.com/auditapp/audit\&#34;

type Dispatcher struct {
\\tsink audit.Sink
}

func NewDispatcher(sink audit.Sink) *Dispatcher {
\\treturn &amp;Dispatcher{sink: sink}
}

func (d *Dispatcher) Record(event audit.Event) error {
\\tevent.Kind = audit.NormalizeKind(event.Kind)
\\treturn d.sink.Write(event)
}

func DispatchAll(sink audit.Sink, events []audit.Event) error {
\\twriteEvent := audit.Sink.Write
\\tfor _, event := range events {
\\t\\tif err := writeEvent(sink, event); err &#34;&#39;!= nil {
&#39;&#34;\\t\\t\\treturn err
\\t\\t}
\\t}
\\treturn nil
}
&#39;&#39;&#39;
}
for path, content in updates.items():
    fd, name = tempfile.mkstemp(dir=path.parent, prefix=path.name + &#39;.&#39;)
    try:
        with os.fdopen(fd, &#39;w&#39;) as f:
            f.write(content)
            f.flush()
            os.fsync(f.fileno())
        os.replace(name, path)
    finally:
        if os.path.exists(name):
            os.unlink(name)
PY
gofmt -w audit/normalize.go audit/summary.go service/dispatch.go
git diff -- audit/normalize.go audit/summary.go service/dispatch.go
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;,
  &#34;include_unexported&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum &amp;&amp; \
git diff --no-ext-diff -- audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
cat audit/normalize.go&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Record&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/event.go&#34;,
  &#34;symbol&#34;: &#34;Event&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve health in audit/summary.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;health&#34;,
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): resolve NormalizeKind in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-prefer_discover_semedit_task-11-mixed-sink-api-migration_1790517756961474000/audit/normalize.go: symbol not found<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
        &#34;access_modifier&#34;: &#34;public&#34;,
        &#34;auto_organize_imports&#34;: true
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
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
<tr><td>12</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum audit/*_test.go service/*_test.go metrics legacy &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum audit/*_test.go service/*_test.go metrics legacy&#39;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;symbol&#34;: &#34;NormalizeKind&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Record&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 10 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;audit/**&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;Makefile&#39; -g &#39;README*&#39; | sort &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;README.md&#39; -g &#39;normalize.go&#39; -g &#39;*dispatcher*&#39; -g &#39;Makefile&#39; | head -80 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
printf &#39;\\n--- audit/normalize.go ---\\n&#39; &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
printf &#39;\\n--- audit/event.go ---\\n&#39; &amp;&amp; \
cat audit/event.go &amp;&amp; \
printf &#39;\\n--- audit/sink.go ---\\n&#39; &amp;&amp; \
cat audit/sink.go &amp;&amp; \
printf &#39;\\n--- project files ---\\n&#39; &amp;&amp; \
rg --files | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md; if [ -f PROJECT_BRIEF.md ]; then cat PROJECT_BRIEF.md; fi; rg --files audit | sort; rg -n \&#34;func \\(.*\\) Record|probe|health|Dispatcher\&#34; audit&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat service/dispatch.go &amp;&amp; \
printf &#39;\\n--- dispatch tests ---\\n&#39; &amp;&amp; \
cat service/dispatch_test.go &amp;&amp; \
printf &#39;\\n--- references ---\\n&#39; &amp;&amp; \
rg -n -i &#39;probe|health|Record\\(|control&#39; . -g &#39;*.go&#39; -g &#39;*.md&#39; &amp;&amp; \
printf &#39;\\n--- go.mod (read only) ---\\n&#39; &amp;&amp; \
cat go.mod&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | head -100; rg -n \&#34;Record\\(|type Dispatcher|probe|Classify\\(\&#34; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat audit/summary_test.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile, time

updates = {
    Path(&#39;audit/normalize.go&#39;): &#39;&#39;&#39;// Package audit owns normalization rules at the delivery boundary.\\npackage audit\\n\\nimport (\\n\\t\&#34;strings\&#34;\\n)\\n\\nfunc NormalizeKind(kind string) string {\\n\\treturn strings.ToLower(strings.TrimSpace(kind))\\n}\\n&#39;&#39;&#39;,
    Path(&#39;service/dispatch.go&#39;): &#39;&#39;&#39;// Package service coordinates application work with the configured audit sink.\\npackage service\\n\\nimport \&#34;example.com/auditapp/audit\&#34;\\n\\ntype Dispatcher struct {\\n\\tsink audit.Sink\\n}\\n\\nfunc NewDispatcher(sink audit.Sink) *Dispatcher {\\n\\treturn &amp;Dispatcher{sink: sink}\\n}\\n\\nfunc (d *Dispatcher) Record(event audit.Event) error {\\n\\tevent.Kind = audit.NormalizeKind(event.Kind)\\n\\treturn d.sink.Write(event)\\n}\\n\\nfunc DispatchAll(sink audit.Sink, events []audit.Event) error {\\n\\twriteEvent := audit.Sink.Write\\n\\tfor _, event := range events {\\n\\t\\tif err := writeEvent(sink, event); err &#34;&#39;!= nil {&#39;&#34;\\n\\t\\t\\treturn err\\n\\t\\t}\\n\\t}\\n\\treturn nil\\n}\\n&#39;&#39;&#39;,
    Path(&#39;audit/summary.go&#39;): &#39;&#39;&#39;// Package audit groups delivery kinds for operational reporting.\\npackage audit\\n\\nfunc Classify(kind string) string {\\n\\tswitch NormalizeKind(kind) {\\n\\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\\n\\t\\treturn \&#34;control\&#34;\\n\\tdefault:\\n\\t\\treturn \&#34;data\&#34;\\n\\t}\\n}\\n&#39;&#39;&#39;,
}
for path, content in updates.items():
    fd, temp_name = tempfile.mkstemp(prefix=f&#39;.{path.name}.&#39;, dir=path.parent)
    try:
        with os.fdopen(fd, &#39;w&#39;) as f:
            f.write(content)
            f.flush()
            os.fsync(f.fileno())
        os.replace(temp_name, path)
        now = time.time_ns()
        os.utime(path, ns=(now, now))
    except BaseException:
        try:
            os.unlink(temp_name)
        except FileNotFoundError:
            pass
        raise
PY
&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
git diff -- audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit/normalize.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go go.mod go.sum &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
sed -n &#39;10,20p&#39; service/dispatch.go&#34;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;audit/normalize.go&#34;,
        &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
        &#34;access_modifier&#34;: &#34;public&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;service/dispatch.go&#34;,
        &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
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
  ],
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;audit&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;service&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go audit/summary.go service/dispatch.go; git status --short -- audit/normalize.go audit/summary.go service/dispatch.go audit service go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/normalize.go; cat audit/summary.go; cat service/dispatch.go; git status --short --untracked-files=all&#39;</code></pre></td></tr>
</tbody>
</table>

