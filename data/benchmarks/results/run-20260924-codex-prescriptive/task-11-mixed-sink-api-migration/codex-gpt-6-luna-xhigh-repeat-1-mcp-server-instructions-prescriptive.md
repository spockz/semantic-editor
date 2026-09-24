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

### Target: `codex/gpt-6-luna/xhigh (repeat 1)`

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

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 128.71s | 162.37s | <span class="benchmark-delta-negative">+26.2%</span> | 255.74s | 0.00s | <span class="benchmark-delta-positive">-100.0%</span> |
| **Process Start → First Event** | 0.14s | 0.12s | — | 0.56s | 0.17s | — |
| **First Event → First Tool Call** | 6.83s | 8.95s | — | 10.01s | 8.46s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 13 | 12 | <span class="benchmark-delta-positive">-7.7%</span> | 13 | 25 | <span class="benchmark-delta-negative">+92.3%</span> |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | 2 | 4 | <span class="benchmark-delta-negative">+100.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 13 | 12 | <span class="benchmark-delta-positive">-7.7%</span> | 13 | 25 | <span class="benchmark-delta-negative">+92.3%</span> |
| **Output Tokens** | 6011 | 6684 | <span class="benchmark-delta-negative">+11.2%</span> | 9521 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **Reasoning / Thinking Tokens** | 4105 | 5223 | <span class="benchmark-delta-negative">+27.2%</span> | 6857 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **Total Input Tokens** | 204372 | 283833 | <span class="benchmark-delta-negative">+38.9%</span> | 383051 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **Cached Input Tokens** | 185088 | 257536 | <span class="benchmark-delta-positive">+39.1%</span> | 352768 | 0 | <span class="benchmark-delta-negative">-100.0%</span> |
| **Uncached Input Tokens** | 19284 | 26297 | <span class="benchmark-delta-negative">+36.4%</span> | 30283 | 0 | <span class="benchmark-delta-positive">-100.0%</span> |
| **Cached vs Uncached Token Ratio** | 9.60:1 | 9.79:1 | <span class="benchmark-delta-positive">+2.0%</span> | 11.65:1 | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)
* **MCP Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;*.md&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS*&#39; -g &#39;*.go&#39; -g &#39;*.md&#39; | head -200 &amp;&amp; \
git status --short &amp;&amp; \
rg -n \&#34;Audit|audit|probe|Probe|event kind|EventKind|event_kind\&#34; . --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/audit-sinks.md&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat audit/event_test.go &amp;&amp; \
cat audit/summary_test.go &amp;&amp; \
cat service/dispatch_test.go &amp;&amp; \
cat metrics/writer_test.go &amp;&amp; \
cat legacy/export_test.go &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/summary.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md &amp;&amp; \
cat docs/audit-sinks.md &amp;&amp; \
cat audit/event.go &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat audit/sink.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
cat metrics/writer.go &amp;&amp; \
cat legacy/export.go&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/sink.go&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | sort | sed -n &#39;1,160p&#39; &amp;&amp; \
sed -n &#39;1,240p&#39; Makefile 2&gt;/dev/null &amp;&amp; \
cat go.mod&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/writer.go&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;event.Kind = audit.NormalizeKind(event.Kind)\nreturn d.sink.Write(event)&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; legacy/export_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/writer_test.go&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\nfor _, event := range events {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\tif err := writeEvent(sink, event); err != nil {\n\t\treturn err\n\t}\n}\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Classify|Normalize|DispatchAll|NewDispatcher|\\.Record\\(|Kind|health|probe&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; .&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod &amp;&amp; \
cat audit/normalize.go &amp;&amp; \
cat service/dispatch.go &amp;&amp; \
cat audit/summary.go &amp;&amp; \
git diff -- audit/normalize.go service/dispatch.go audit/summary.go&#39;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os
import stat
import tempfile
import time

updates = {
    Path(&#39;audit/normalize.go&#39;): &#39;&#39;&#39;// Package audit normalizes event kinds before audit delivery.
package audit

import \&#34;strings\&#34;

func NormalizeKind(kind string) string {
	return strings.ToLower(strings.Join(strings.Fields(kind), \&#34; \&#34;))
}
&#39;&#39;&#39;,
    Path(&#39;audit/summary.go&#39;): &#39;&#39;&#39;// Package audit groups delivery kinds for operational reporting.
package audit

func Classify(kind string) string {
	switch NormalizeKind(kind) {
	case \&#34;health\&#34;, \&#34;probe\&#34;:
		return \&#34;control\&#34;
	default:
		return \&#34;data\&#34;
	}
}
&#39;&#39;&#39;,
    Path(&#39;service/dispatch.go&#39;): &#39;&#39;&#39;// Package service coordinates application work with the configured audit sink.
package service

import \&#34;example.com/auditapp/audit\&#34;

type Dispatcher struct {
	sink audit.Sink
}

func NewDispatcher(sink audit.Sink) *Dispatcher {
	return &amp;Dispatcher{sink: sink}
}

func (d *Dispatcher) Record(event audit.Event) error {
	event.Kind = audit.NormalizeKind(event.Kind)
	return d.sink.Write(event)
}

func DispatchAll(sink audit.Sink, events []audit.Event) error {
	for _, event := range events {
		event.Kind = audit.NormalizeKind(event.Kind)
		if err := sink.Write(event); err &#34;&#39;!= nil {
			return err
		}
	}
	return nil
}
&#39;&#34;&#39;&#39;&#39;,
}

for path, content in updates.items():
    old_stat = path.stat()
    fd, temp_name = tempfile.mkstemp(prefix=f&#39;.{path.name}.&#39;, dir=path.parent)
    try:
        with os.fdopen(fd, &#39;w&#39;, encoding=&#39;utf-8&#39;, newline=&#39;&#39;) as temp:
            temp.write(content)
            temp.flush()
            os.fsync(temp.fileno())
        os.chmod(temp_name, stat.S_IMODE(old_stat.st_mode))
        mtime_ns = max(time.time_ns(), old_stat.st_mtime_ns + 1)
        os.utime(temp_name, ns=(mtime_ns, mtime_ns))
        os.replace(temp_name, path)
        dir_fd = os.open(path.parent, os.O_RDONLY)
        try:
            os.fsync(dir_fd)
        finally:
            os.close(dir_fd)
    except BaseException:
        try:
            os.unlink(temp_name)
        except FileNotFoundError:
            pass
        raise
PY
gofmt -w audit/normalize.go audit/summary.go service/dispatch.go
git diff -- audit/normalize.go audit/summary.go service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,120p&#39; service/dispatch.go &amp;&amp; \
git status --short -- audit/normalize.go audit/summary.go service/dispatch.go &amp;&amp; \
git ls-files --stage -- audit/normalize.go audit/summary.go service/dispatch.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;set -eu
before_mod=$(shasum -a 256 go.mod go.sum)
gofmt -d audit/normalize.go audit/summary.go service/dispatch.go
go test ./...
after_mod=$(shasum -a 256 go.mod go.sum)
if [ &#34;$before_mod&#34; != &#34;$after_mod&#34; ]; then
  printf &#39;&#34;&#39;%s\\n&#39; &#39;go.mod/go.sum changed during verification&#39; &gt;&amp;2
  exit 1
fi
printf &#39;%s\\n&#39; &#39;go.mod and go.sum unchanged&#39; &#34;</code></pre></td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;set -eu
before_mod=$(shasum -a 256 go.mod)
if [ -e go.sum ]; then before_sum=present; else before_sum=absent; fi
gofmt -d audit/normalize.go audit/summary.go service/dispatch.go
go test ./...
after_mod=$(shasum -a 256 go.mod)
if [ -e go.sum ]; then after_sum=present; else after_sum=absent; fi
if [ &#34;$before_mod&#34; != &#34;$after_mod&#34; ] || \
[ &#34;$before_sum&#34; != &#34;$after_sum&#34; ]; then
  printf &#39;&#34;&#39;%s\\n&#39; &#39;protected module metadata changed during verification&#39; &gt;&amp;2
  exit 1
fi
printf &#39;go.mod unchanged; go.sum remains %s\\n&#39; \&#34;&#34;&#39;$after_sum&#34;&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 15 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;README*&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;docs/**&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;docs/**&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!*.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!*.mod&#39;&#34;&#39; &amp;&amp; \
git status --short &amp;&amp; \
sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/sink.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff --stat&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -la &amp;&amp; \
sed -n &#39;1,260p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/transaction.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,240p&#39; service/replay.go &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/metrics.go &amp;&amp; \
sed -n &#39;1,240p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,260p&#39; legacy/export.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,240p&#39; audit/normalize.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,280p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; metrics/writer_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; legacy/export_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; db/db.go &amp;&amp; \
sed -n &#39;1,260p&#39; journal/file.go &amp;&amp; \
sed -n &#39;1,260p&#39; models/models.go &amp;&amp; \
cat go.mod&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; audit/fanout.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/transaction.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile, time

root = Path.cwd()
changes = {
    &#39;audit/normalize.go&#39;: &#39;&#39;&#39;// Package audit owns event normalization at its delivery boundary.
package audit

import (
\\t\&#34;strings\&#34;
)

func NormalizeKind(kind string) string {
\\treturn strings.Join(strings.Fields(strings.ToLower(kind)), \&#34; \&#34;)
}

func NormalizeEvent(event Event) Event {
\\tevent.Kind = NormalizeKind(event.Kind)
\\treturn event
}
&#39;&#39;&#39;,
    &#39;audit/summary.go&#39;: &#39;&#39;&#39;// Package audit groups delivery kinds for operational reporting.
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
    &#39;audit/sink.go&#39;: &#39;&#39;&#39;// Package audit exposes the established event sink contract and a memory implementation.
package audit

type Sink interface {
\\tWrite(Event) error
}

type TransactionalSink interface {
\\tSink
\\tBegin() error
\\tCommit() error
}

type MemorySink struct {
\\tEvents []Event
}

func NewMemorySink() *MemorySink {
\\treturn &amp;MemorySink{}
}

func (m *MemorySink) Write(event Event) error {
\\tm.Events = append(m.Events, NormalizeEvent(event))
\\treturn nil
}
&#39;&#39;&#39;,
    &#39;audit/fanout.go&#39;: &#39;&#39;&#39;// FanoutSink delivers a normalized event to each configured audit destination.
package audit

type FanoutSink struct {
\\tFirst  Sink
\\tSecond Sink
}

func (f FanoutSink) Write(event Event) error {
\\tevent = NormalizeEvent(event)
\\tif err := f.First.Write(event); err &#34;&#39;!= nil {
&#39;&#34;\\t\\treturn err
\\t}
\\treturn f.Second.Write(event)
}
&#39;&#39;&#39;,
    &#39;audit/transaction.go&#39;: &#39;&#39;&#39;// BufferedSink exposes transactional delivery without owning the persistence protocol.
package audit

type BufferedSink struct {
\\tSink
\\tqueue []Event
}

func (b *BufferedSink) Begin() error {
\\tb.queue = b.queue[:0]
\\treturn nil
}

func (b *BufferedSink) Write(event Event) error {
\\tb.queue = append(b.queue, NormalizeEvent(event))
\\treturn nil
}

func (b *BufferedSink) Commit() error {
\\tfor _, event := range b.queue {
\\t\\tif err := b.Sink.Write(NormalizeEvent(event)); err &#34;&#39;!= nil {
&#39;&#34;\\t\\t\\treturn err
\\t\\t}
\\t}
\\treturn nil
}
&#39;&#39;&#39;,
    &#39;journal/file.go&#39;: &#39;&#39;&#39;// Package journal contains a separate audit implementation used by batch jobs.
package journal

import \&#34;example.com/auditapp/audit\&#34;

type JournalSink struct {
\\tEntries []audit.Event
}

func (j *JournalSink) Write(event audit.Event) error {
\\tj.Entries = append(j.Entries, audit.NormalizeEvent(event))
\\treturn nil
}

var _ audit.Sink = (*JournalSink)(nil)
&#39;&#39;&#39;,
    &#39;service/dispatch.go&#39;: &#39;&#39;&#39;// Package service coordinates application work with the configured audit sink.
package service

import \&#34;example.com/auditapp/audit\&#34;

type Dispatcher struct {
\\tsink audit.Sink
}

func NewDispatcher(sink audit.Sink) *Dispatcher {
\\treturn &amp;Dispatcher{sink: sink}
}

func (d *Dispatcher) Record(event audit.Event) error {
\\treturn deliver(d.sink, event)
}

func DispatchAll(sink audit.Sink, events []audit.Event) error {
\\tfor _, event := range events {
\\t\\tif err := deliver(sink, event); err &#34;&#39;!= nil {
&#39;&#34;\\t\\t\\treturn err
\\t\\t}
\\t}
\\treturn nil
}

func deliver(sink audit.Sink, event audit.Event) error {
\\treturn sink.Write(audit.NormalizeEvent(event))
}
&#39;&#39;&#39;,
    &#39;service/replay.go&#39;: &#39;&#39;&#39;// Package service contains replay helpers for previously captured audit events.
package service

import \&#34;example.com/auditapp/audit\&#34;

func Replay(sink audit.Sink, events []audit.Event) error {
\\tfor _, event := range events {
\\t\\tif err := deliver(sink, event); err &#34;&#39;!= nil {
&#39;&#34;\\t\\t\\treturn err
\\t\\t}
\\t}
\\treturn nil
}
&#39;&#39;&#39;,
    &#39;docs/audit-sinks.md&#39;: &#39;&#39;&#39;# Audit delivery

The service records operational events at a package boundary so deployments can choose an appropriate destination. Event labels arrive from command-line flags, hooks, and external integrations, so audit delivery trims and lowercases them and collapses whitespace before sending them to a sink. Audit reporting classifies &#34;&#39;`health` and `probe` events as control traffic.

Metrics collection and archival export use separate protocols. Changes to the audit boundary should avoid coupling those components to event persistence details.
&#39;&#34;&#39;&#39;&#39;,
}

for name, content in changes.items():
    path = root / name
    old_stat = path.stat()
    fd, temp_name = tempfile.mkstemp(prefix=f&#39;.{path.name}.&#39;, dir=path.parent)
    try:
        with os.fdopen(fd, &#39;w&#39;, encoding=&#39;utf-8&#39;, newline=&#39;&#39;) as stream:
            stream.write(content)
            stream.flush()
            os.fsync(stream.fileno())
            next_mtime = max(time.time_ns(), old_stat.st_mtime_ns + 1)
            os.utime(stream.fileno(), ns=(old_stat.st_atime_ns, next_mtime))
            os.fsync(stream.fileno())
        os.replace(temp_name, path)
        dir_fd = os.open(path.parent, os.O_RDONLY)
        try:
            os.fsync(dir_fd)
        finally:
            os.close(dir_fd)
    except BaseException:
        try:
            os.unlink(temp_name)
        except FileNotFoundError:
            pass
        raise
PY&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,200p&#39; db/db.go &amp;&amp; \
sed -n &#39;1,180p&#39; journal/file.go &amp;&amp; \
sed -n &#39;1,180p&#39; Makefile&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go &amp;&amp; \
shasum -a 256 go.mod &amp;&amp; \
test ! -e go.sum &amp;&amp; \
printf &#39;&#34;&#39;go.sum absent\\n&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;Sink|Write\\(|DispatchAll|Record\\(|Kind|probe|Probe|metrics|legacy|export\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; .&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*test.go&#39; -g &#39;*.go&#39; -g &#39;*.md&#39;&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go vet ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,240p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; audit/summary_test.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,220p&#39; health/health.go&#34;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,160p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,120p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,180p&#39; audit/transaction.go &amp;&amp; \
sed -n &#39;1,180p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,120p&#39; service/replay.go &amp;&amp; \
sed -n &#39;1,120p&#39; journal/file.go &amp;&amp; \
sed -n &#39;1,120p&#39; docs/audit-sinks.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat audit/summary.go &amp;&amp; \
cat service/replay.go &amp;&amp; \
cat go.mod &amp;&amp; \
cat AGENTS.override.md&#39;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func deliver|\\.Write\\(event|Classify\\(\&#34; -g &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- audit service metrics legacy go.mod go.sum docs/audit-sinks.md AGENTS.override.md&#39;</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/summary.go audit/sink.go audit/fanout.go audit/transaction.go journal/file.go service/dispatch.go service/replay.go&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;shasum -a 256 go.mod&#39;</code></pre></td><td><code>semedit/semantic_insert_type</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;type deliverySink struct {\n\tsink Sink\n}&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;,
  &#34;placement&#34;: &#34;file_end&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;test ! -e go.sum&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NewDeliverySink(sink Sink) Sink {\n\treturn deliverySink{sink: sink}\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;file_end&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func (d deliverySink) Write(event Event) error {\n\tevent.Kind = normalizeKind(event.Kind)\n\treturn d.sink.Write(event)\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;placement&#34;: &#34;file_end&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch normalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;NewDispatcher&#34;,
  &#34;body&#34;: &#34;return &amp;Dispatcher{sink: audit.NewDeliverySink(sink)}&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;delivery := audit.NewDeliverySink(sink)\nfor _, event := range events {\n\tif err := delivery.Write(event); err != nil {\n\t\treturn err\n\t}\n}\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;return DispatchAll(sink, events)&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,180p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,220p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,120p&#39; service/replay.go&#34;</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- audit/normalize.go audit/summary.go service/dispatch.go service/replay.go &amp;&amp; \
git status --short -- go.mod go.sum audit service metrics legacy&#39;</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os
import tempfile
import time

path = Path(&#39;audit/normalize.go&#39;)
old = path.read_bytes()
source = b&#39;// Package audit leaves room for normalization rules owned by the delivery boundary.\\n&#39;
replacement = b&#39;// Package audit owns event kind normalization at the delivery boundary.\\n&#39;
if not old.startswith(source):
    raise SystemExit(&#39;expected package comment was not found&#39;)
updated = replacement + old[len(source):]
previous_mtime = path.stat().st_mtime_ns
fd, temp_name = tempfile.mkstemp(prefix=&#39;.normalize.go.&#39;, dir=path.parent)
try:
    with os.fdopen(fd, &#39;wb&#39;) as temp_file:
        temp_file.write(updated)
        temp_file.flush()
        os.fsync(temp_file.fileno())
    next_mtime = max(time.time_ns(), previous_mtime + 1)
    os.utime(temp_name, ns=(next_mtime, next_mtime))
    os.replace(temp_name, path)
    directory_fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)
except BaseException:
    try:
        os.unlink(temp_name)
    except FileNotFoundError:
        pass
    raise
PY&#34;</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/summary.go service/dispatch.go service/replay.go&#39;</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum audit service metrics legacy &amp;&amp; \
test ! -e go.sum &amp;&amp; \
echo &#39;&#34;&#39;go.sum absent and unchanged&#39; || \
echo &#39;go.sum exists&#39;&#34;</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test -mod=readonly ./...&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I had discovered `semantic_batch`, but chose separate calls so I could inspect each edit result as I went. Batching was usable and could have run these dependent edits in order; this was my choice, not a limitation.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
<p>Reflection wall-clock: 7.57s; turns: 1; tool calls: 0.</p>
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
| **Wall-Clock Latency** | 211.74s | 0.00s | <span class="benchmark-delta-positive">-100.0%</span> | 0.00s | 0.00s | — |
| **Process Start → First Event** | 0.11s | 0.12s | — | 0.58s | 0.16s | — |
| **First Event → First Tool Call** | 21.90s | 36.03s | — | 37.51s | 15.28s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 14 | 11 | <span class="benchmark-delta-positive">-21.4%</span> | 10 | 21 | <span class="benchmark-delta-negative">+110.0%</span> |
| **Initial Load / Discovery Turns** | 4 | 1 | <span class="benchmark-delta-positive">-75.0%</span> | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 14 | 11 | <span class="benchmark-delta-positive">-21.4%</span> | 10 | 21 | <span class="benchmark-delta-negative">+110.0%</span> |
| **Output Tokens** | 10821 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 0 | 0 | 0% |
| **Reasoning / Thinking Tokens** | 8338 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 0 | 0 | 0% |
| **Total Input Tokens** | 339654 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 0 | 0 | 0% |
| **Cached Input Tokens** | 302592 | 0 | <span class="benchmark-delta-negative">-100.0%</span> | 0 | 0 | 0% |
| **Uncached Input Tokens** | 37062 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 0 | 0 | 0% |
| **Cached vs Uncached Token Ratio** | 8.16:1 | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | — | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | — | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | — | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File audit/normalize.go modified (2 lines -> 8 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n -i \&#34;audit|probe|delivery|persist|metrics|legacy export\&#34; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; | head -200 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;*.go&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; | sed -n &#39;1,160p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files . &amp;&amp; \
sed -n &#39;1,240p&#39; audit/event.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,280p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,260p&#39; service/dispatch.go &amp;&amp; \
sed -n &#39;1,220p&#39; metrics/writer.go &amp;&amp; \
sed -n &#39;1,220p&#39; legacy/export.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd &amp;&amp; \
git status --short --branch&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/audit-sinks.md &amp;&amp; \
sed -n &#39;1,280p&#39; audit/event_test.go &amp;&amp; \
sed -n &#39;1,280p&#39; service/dispatch_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,260p&#39; audit/summary_test.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat AGENTS.override.md docs/audit-sinks.md&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func NormalizeEvent(event Event) Event {\n\tevent.Kind = normalizeKind(event.Kind)\n\treturn event\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in audit/event_test.go audit/summary_test.go service/dispatch_test.go metrics/writer_test.go legacy/export_test.go; do printf &#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; cat &#34;$f&#34;; done&#39;</code></pre></td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in audit/event.go audit/normalize.go audit/sink.go audit/summary.go service/dispatch.go cmd/migrate/main.go metrics/writer.go legacy/export.go; do printf &#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; cat &#34;$f&#34;; done&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch normalizeKind(kind) {\ncase \&#34;health\&#34;, \&#34;probe\&#34;:\n\treturn \&#34;control\&#34;\ndefault:\n\treturn \&#34;data\&#34;\n}&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort &amp;&amp; \
git status --short -- .&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;(*MemorySink).Write&#34;,
  &#34;body&#34;: &#34;m.Events = append(m.Events, NormalizeEvent(event))\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
rg -n \&#34;\\.Write\\(|DispatchAll|Classify\\(\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;(*Dispatcher).Record&#34;,
  &#34;body&#34;: &#34;return d.sink.Write(audit.NormalizeEvent(event))&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os
import stat
import tempfile
import time

root = Path(\&#34;.\&#34;)
scratch = root / \&#34;.scratch\&#34;
scratch.mkdir(exist_ok=True)
updates = {
    Path(\&#34;audit/normalize.go\&#34;): \&#34;\&#34;\&#34;// Package audit normalizes event kinds shared by delivery and reporting.
package audit

import \&#34;strings\&#34;

func NormalizeKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}
\&#34;\&#34;\&#34;,
    Path(\&#34;audit/summary.go\&#34;): \&#34;\&#34;\&#34;// Package audit groups delivery kinds for operational reporting.
package audit

func Classify(kind string) string {
	switch NormalizeKind(kind) {
	case \&#34;health\&#34;, \&#34;probe\&#34;:
		return \&#34;control\&#34;
	default:
		return \&#34;data\&#34;
	}
}
\&#34;\&#34;\&#34;,
    Path(\&#34;service/dispatch.go\&#34;): \&#34;\&#34;\&#34;// Package service coordinates application work with the configured audit sink.
package service

import \&#34;example.com/auditapp/audit\&#34;

type Dispatcher struct {
	sink audit.Sink
}

func NewDispatcher(sink audit.Sink) *Dispatcher {
	return &amp;Dispatcher{sink: sink}
}

func (d *Dispatcher) Record(event audit.Event) error {
	return deliver(d.sink, event)
}

func DispatchAll(sink audit.Sink, events []audit.Event) error {
	for _, event := range events {
		if err := deliver(sink, event); err &#34;&#39;!= nil {
			return err
		}
	}
	return nil
}

func deliver(sink audit.Sink, event audit.Event) error {
	event.Kind = audit.NormalizeKind(event.Kind)
	return sink.Write(event)
}
&#34;&#34;&#34;,
}
for path, content in updates.items():
    original = path.stat()
    fd, temporary_name = tempfile.mkstemp(prefix=f&#34;{path.name}.&#34;, suffix=&#34;.tmp&#34;, dir=scratch)
    temporary = Path(temporary_name)
    try:
        with os.fdopen(fd, &#34;wb&#34;) as output:
            output.write(content.encode(&#34;utf-8&#34;))
            output.flush()
            os.fsync(output.fileno())
        os.chmod(temporary, stat.S_IMODE(original.st_mode))
        os.replace(temporary, path)
        new_mtime_ns = max(time.time_ns(), original.st_mtime_ns + 1)
        os.utime(path, ns=(original.st_atime_ns, new_mtime_ns))
        directory_fd = os.open(path.parent, os.O_RDONLY)
        try:
            os.fsync(directory_fd)
        finally:
            os.close(directory_fd)
    finally:
        if temporary.exists():
            temporary.unlink()
try:
    scratch.rmdir()
except OSError:
    pass
PY&#39;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;writeEvent := audit.Sink.Write\nfor _, event := range events {\n\tif err := writeEvent(sink, audit.NormalizeEvent(event)); err != nil {\n\t\treturn err\n\t}\n}\nreturn nil&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find . -maxdepth 2 -type f -not -path &#39;./.git/*&#39; -print | sort&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; audit/normalize.go &amp;&amp; \
sed -n &#39;1,180p&#39; audit/summary.go &amp;&amp; \
sed -n &#39;1,180p&#39; audit/sink.go &amp;&amp; \
sed -n &#39;1,220p&#39; service/dispatch.go &amp;&amp; \
sha256sum go.mod go.sum audit/event_test.go service/dispatch_test.go audit/summary_test.go metrics/writer_test.go legacy/export_test.go&#34;</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d audit/normalize.go audit/summary.go service/dispatch.go&#39;</code></pre></td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- . &amp;&amp; \
git diff -- . &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td><td>—</td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;set -eu
GOTOOLCHAIN=local GOPROXY=off go test -mod=readonly ./...
mkdir -p .scratch
cat &gt; .scratch/audit_verify.go &lt;&lt;&#39;EOF&#39;
package main

import (
	\&#34;fmt\&#34;

	\&#34;example.com/auditapp/audit\&#34;
	\&#34;example.com/auditapp/legacy\&#34;
	\&#34;example.com/auditapp/metrics\&#34;
	\&#34;example.com/auditapp/service\&#34;
)

type sink struct { events []audit.Event }
func (s *sink) Write(event audit.Event) error { s.events = append(s.events, event); return nil }

func main() {
	first := &amp;sink{}
	if err := service.NewDispatcher(first).Record(audit.Event{ID: \&#34;one\&#34;, Kind: \&#34; \\tPrObE\\n\&#34;}); err &#34;&#39;!= nil { panic(err) }
	if len(first.events) != 1 || \
first.events[0].Kind != &#34;probe&#34; { panic(fmt.Sprintf(&#34;recorded events: %#v&#34;, first.events)) }
	second := &amp;sink{}
	if err := service.DispatchAll(second, []audit.Event{{ID: &#34;two&#34;, Kind: &#34; HeAlTh &#34;}}); err != nil { panic(err) }
	if len(second.events) != 1 || \
second.events[0].Kind != &#34;health&#34; { panic(fmt.Sprintf(&#34;dispatched events: %#v&#34;, second.events)) }
	if audit.Classify(&#34; PROBE &#34;) != &#34;control&#34; { panic(&#34;probe was not classified as control traffic&#34;) }
	if metrics.Classify(&#34;probe&#34;) != &#34;sample&#34; || \
legacy.Classify(&#34;probe&#34;) != &#34;archive&#34; { panic(&#34;metrics or legacy protocol changed&#34;) }
}
EOF
GOTOOLCHAIN=local GOPROXY=off go run -mod=readonly .scratch/audit_verify.go
rm .scratch/audit_verify.go
rmdir .scratch&#39;</code></pre></td><td>—</td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;find .scratch -maxdepth 3 -print -ls&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS.md&#39; -g &#39;Makefile&#39; -g &#39;*.go&#39; -g &#39;*.yaml&#39; -g &#39;*.yml&#39; -g &#39;*.json&#39; | head -250&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; | head -100 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;ls -la&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -la &amp;&amp; \
printf &#39;\\n--- top-level docs and config ---\\n&#39; &amp;&amp; \
for f in PROJECT_BRIEF.md Makefile go.mod; do if [ -f \&#34;&#34;&#39;$f&#34; ]; then echo &#34;--- $f ---&#34;; sed -n &#39;&#34;&#39;1,220p&#39; \&#34;&#34;&#39;$f&#34;; fi; done &amp;&amp; \
printf &#39;&#34;&#39;\\n--- event and delivery source ---\\n&#39; &amp;&amp; \
for f in audit/event.go audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go; do echo \&#34;--- &#34;&#39;$f ---&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done &amp;&amp; \
printf &#39;&#34;&#39;\\n--- call sites ---\\n&#39; &amp;&amp; \
rg -n \&#34;Audit|audit|Probe|probe|Kind|kind\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short --branch&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in service/dispatch.go service/replay.go journal/file.go audit/summary.go metrics/writer.go metrics/metrics.go legacy/export.go audit/*_test.go service/*_test.go journal/*_test.go; do if [ -f &#34;$f&#34; ]; then echo &#34;--- $f ---&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; fi; done &amp;&amp; \
printf &#39;&#34;&#39;\\n--- all files ---\\n&#39; &amp;&amp; \
rg --files&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in audit/event.go audit/normalize.go audit/sink.go audit/fanout.go audit/transaction.go audit/summary.go; do sed -n &#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): read Write in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-prefer_discover_semedit_task-11-mixed-sink-api-migration_1790267354901164000/audit: read /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_large-prefer_discover_semedit_task-11-mixed-sink-api-migration_1790267354901164000/audit: is a directory<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;audit&#34;,
  &#34;symbol&#34;: &#34;MemorySink.Write&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in service/dispatch.go service/replay.go api/router.go db/db.go journal/file.go metrics/metrics.go metrics/writer.go legacy/export.go; do sed -n &#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find docs -maxdepth 2 -type f -print &amp;&amp; \
if [ -f docs/SUBOPTIMAL_TOOLS.md ]; then sed -n &#39;1,240p&#39; docs/SUBOPTIMAL_TOOLS.md; fi&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n --glob &#39;*.go&#39; &#39;Event|event|Sink|Persist|Append|probe|Probe|metrics|legacy&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
import os
from pathlib import Path
import tempfile

path = Path(&#39;docs/SUBOPTIMAL_TOOLS.md&#39;)
path.parent.mkdir(parents=True, exist_ok=True)
entry = &#39;&#39;&#39;# Suboptimal Tool Behavior

## 2026-09-24: semantic lookup target resolution

- Tool: &#34;&#39;`semantic_lookup`
- Target: `audit/sink.go` (initially passed the `audit` directory)
- Observed failure: lookup rejected the directory target because it requires a source file path.
- Workaround applied: retry lookup against the concrete Go source file.
- Root cause: the lookup call supplied a package directory instead of a file.
&#39;&#34;&#39;&#39;&#39;
fd, temp_name = tempfile.mkstemp(prefix=&#39;.SUBOPTIMAL_TOOLS.&#39;, dir=path.parent)
try:
    with os.fdopen(fd, &#39;w&#39;, encoding=&#39;utf-8&#39;) as stream:
        stream.write(entry)
        stream.flush()
        os.fsync(stream.fileno())
    if path.exists():
        current = path.stat()
        new_time = max(current.st_mtime_ns + 1_000_000, current.st_mtime_ns + 1)
        os.utime(temp_name, ns=(current.st_atime_ns, new_time))
    os.replace(temp_name, path)
    directory_fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)
except BaseException:
    try:
        os.unlink(temp_name)
    except FileNotFoundError:
        pass
    raise
PY&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;audit/sink.go&#34;,
  &#34;symbol&#34;: &#34;MemorySink.Write&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in audit/event_test.go audit/summary_test.go service/dispatch_test.go legacy/export_test.go metrics/writer_test.go; do sed -n &#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in docs/audit-sinks.md service/dispatch.go service/replay.go journal/file.go audit/summary.go metrics/writer.go metrics/metrics.go legacy/export.go audit/transaction.go; do if [ -f &#34;$f&#34; ]; then echo &#34;--- $f ---&#34;; sed -n &#39;&#34;&#39;1,260p&#39; \&#34;&#34;&#39;$f&#34;; fi; done&#39;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!*.go&#39;&#34;&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!*.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!*.mod&#39;&#34;&#39; . &amp;&amp; \
for f in cmd/migrate/main.go cmd/replay/main.go; do sed -n &#39;1,180p&#39; \&#34;&#34;&#39;$f&#34;; done&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/audit-sinks.md &amp;&amp; \
rg -n --glob &#39;*.go&#39; &#39;NewDispatcher|DispatchAll|Replay|FanoutSink|BufferedSink|TransactionalSink|Classify|Normalize|normalize&#39; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;audit/transaction.go&#34;,
  &#34;symbol&#34;: &#34;BufferedSink.Begin&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;journal/file.go&#34;,
  &#34;symbol&#34;: &#34;JournalSink.Write&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in audit/event_test.go audit/summary_test.go audit/transaction.go audit/fanout.go audit/sink.go service/dispatch_test.go api/router.go api/router_test.go; do if [ -f &#34;$f&#34; ]; then echo &#34;--- $f ---&#34;; sed -n &#39;&#34;&#39;1,280p&#39; \&#34;&#34;&#39;$f&#34;; fi; done &amp;&amp; \
rg -n &#34;Normalize|Classify|NewDispatcher|DispatchAll|Replay|JournalSink|FanoutSink|BufferedSink|TransactionalSink&#34; --glob &#39;&#34;&#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/normalize.go&#34;,
  &#34;source&#34;: &#34;// NormalizeKind trims surrounding whitespace and lowercases an event kind.\nfunc NormalizeKind(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;source&#34;: &#34;func deliverEvent(sink audit.Sink, event audit.Event) error {\n\tevent.Kind = audit.NormalizeKind(event.Kind)\n\treturn sink.Write(event)\n}&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;NewDispatcher&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;Dispatcher.Record&#34;,
  &#34;body&#34;: &#34;return deliverEvent(d.sink, event)&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/dispatch.go&#34;,
  &#34;symbol&#34;: &#34;DispatchAll&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := deliverEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;service/replay.go&#34;,
  &#34;symbol&#34;: &#34;Replay&#34;,
  &#34;body&#34;: &#34;for _, event := range events {\n\t\tif err := deliverEvent(sink, event); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;audit/summary.go&#34;,
  &#34;symbol&#34;: &#34;Classify&#34;,
  &#34;body&#34;: &#34;switch NormalizeKind(kind) {\n\tcase \&#34;health\&#34;, \&#34;probe\&#34;:\n\t\treturn \&#34;control\&#34;\n\tdefault:\n\t\treturn \&#34;data\&#34;\n\t}&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;make check&#39;</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GOWORK=off go test ./...&#39;</code></pre></td></tr>
</tbody>
</table>

