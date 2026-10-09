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

* **Date**: 2026-10-06 23:33:14 CEST

## Test case: `zeromicro__go-zero-2463`

### Target: `codex/gpt-6-luna/medium (repeat 1)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/zeromicro__go-zero-2463.txtar](https://github.com/spockz/semantic-editor/blob/43bf7788d286eda6cb43f908eecb81fc2ff8694e/testdata/bench/zeromicro__go-zero-2463.txtar)

**LLM Prompt**:
> logx能否支持一下全局添加field功能，或默认加一个serviceName，收集日志想根据服务分索引很不方便
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 62.15s | 87.04s | N/A | — | — | — |
| **Process Start → First Event** | 0.93s | 0.54s | — | — | — | — |
| **First Event → First Tool Call** | 3.46s | 4.83s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 19 | 16 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 3 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 19 | 16 | N/A | — | — | — |
| **Output Tokens** | 4084 | 2772 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 1002 | 587 | N/A | — | — | — |
| **Total Input Tokens** | 568631 | 437770 | N/A | — | — | — |
| **Cached Input Tokens** | 526080 | 406016 | N/A | — | — | — |
| **Uncached Input Tokens** | 42551 | 31754 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 12.36:1 | 12.79:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*logx*&#39; -g &#39;go.mod&#39; | head -100 &amp;&amp; \
rg -n \&#34;package logx|type.*Logger|WithFields|AddField|serviceName|ServiceName\&#34; --glob &#39;*.go&#39; | head -160&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*logx*&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; .. . | head -60; sed -n &#39;1,240p&#39; core/logx/richlogger.go; sed -n &#39;1,200p&#39; core/logx/fields.go; sed -n &#39;1,120p&#39; core/logx/logger.go; sed -n &#39;1,140p&#39; core/logx/config.go; sed -n &#39;1,140p&#39; core/logx/vars.go&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find .. -name AGENTS.md -print; rg --files | rg &#39;(&#34;&#39;^|/)logx/&#39;&#34;&#39; | head; rg -n &#39;serviceName|WithFields|Add.*Field|Fields&#39; logx&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; core/logx/logs.go; sed -n &#39;1,180p&#39; core/logx/fields_test.go; sed -n &#39;1,300p&#39; core/logx/richlogger_test.go; rg -n \&#34;Set.*Log|LogConf|Reset.*Logger|SetWriter|WithContext\&#34; core/logx&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat PROJECT_BRIEF.md 2&gt;/dev/null; rg -n &#39;serviceName|WithFields|Fields|Add.*Field&#39; core/logx; sed -n &#39;1,240p&#39; core/logx/logs.go; sed -n &#39;1,220p&#39; core/logx/fields_test.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;240,430p&#39; core/logx/logs.go; rg -n \&#34;func write|buildFields\&#34; core/logx/logs.go; head -80 core/logx/readme-cn.md; head -90 core/logx/readme.md; rg -n \&#34;global.*field|AddField\&#34; README* core/logx&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,230p&#39; core/logx/richlogger.go; sed -n &#39;1,110p&#39; core/logx/fields.go; rg -n &#39;func write|writeInfo|writeError|plain&#39; core/logx/logs.go core/logx/writer.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;417,475p&#39; core/logx/logs.go; rg -n \&#34;func \\(.*\\) (Alert|Severe|Stat|Stack|Debug|Info)\&#34; core/logx&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;400,475p&#39; core/logx/logs.go; rg -n &#39;func NewLogger|type Logger|New.*Logger&#39; core/logx&#34;</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;func addCaller|ServiceName&#39; core/logx/logs.go&#34;</code></pre></td></tr>
<tr><td>7</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;330,350p&#39; core/logx/logs.go&#34;</code></pre></td></tr>
<tr><td>8</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;20,48p&#39; core/logx/logs.go; sed -n &#39;220,250p&#39; core/logx/logs.go&#34;</code></pre></td></tr>
<tr><td>9</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>10</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;WithDuration&#34; core/logx/readme*.md; tail -20 core/logx/readme.md; git diff --stat&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/logs.go core/logx/richlogger.go &amp;&amp; \
git diff -- core/logx/logs.go core/logx/richlogger.go&#39;</code></pre></td></tr>
<tr><td>12</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;245,280p&#39; core/logx/writer.go; sed -n &#39;48,86p&#39; core/logx/readme-cn.md&#34;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/logs.go core/logx/richlogger.go core/logx/richlogger_test.go &amp;&amp; \
go test ./core/logx&#39;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- core/logx/logs.go core/logx/richlogger.go core/logx/richlogger_test.go core/logx/readme.md core/logx/readme-cn.md; sed -n &#39;170,230p&#39; core/logx/writer.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;SetUp|ServiceName&#39; core/logx/readme-cn.md core/logx/readme.md | head&#34;</code></pre></td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;func output|func write&#34; core/logx/writer.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,44p&#39; core/logx/readme-cn.md; sed -n &#39;1,44p&#39; core/logx/readme.md&#34;</code></pre></td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;266,320p&#39; core/logx/writer.go&#34;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>17</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>18</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>19</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 62.15s | 10.53s | N/A | — | — | — |
| **Process Start → First Event** | 0.93s | — | — | — | — | — |
| **First Event → First Tool Call** | 3.46s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 0 | N/A | — | — | — |
| **Internal Tool Cycles** | 19 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 19 | 0 | N/A | — | — | — |
| **Output Tokens** | 4084 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 1002 | 0 | N/A | — | — | — |
| **Total Input Tokens** | 568631 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 526080 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 42551 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 12.36:1 | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

* **MCP Edit**: diff --git a/core/logx/fields.go b/core/logx/fields.go
index 08bcf636..1bdba318 100644
--- a/core/logx/fields.go
+++ b/core/logx/fields.go
@@ -1,11 +1,32 @@
 package logx

-import "context"
+import (

* "context"
* "sync"
* "sync/atomic"
+)

-var fieldsContextKey contextKey
+var (

* fieldsContextKey contextKey
* globalFields     atomic.Value
* globalFieldsLock sync.Mutex
+)

 type contextKey struct{}

+// AddGlobalFields adds global fields.
+func AddGlobalFields(fields ...LogField) {

* globalFieldsLock.Lock()
* defer globalFieldsLock.Unlock()
*
* old := globalFields.Load()
* if old == nil {
* globalFields.Store(append([]LogField(nil), fields...))
* } else {
* globalFields.Store(append(old.([]LogField), fields...))
* }
+}
*

 // ContextWithFields returns a new context with the given fields.
 func ContextWithFields(ctx context.Context, fields ...LogField) context.Context {
  if val := ctx.Value(fieldsContextKey); val != nil {
diff --git a/core/logx/writer.go b/core/logx/writer.go
index 46c7eb6a..deeb1c84 100644
--- a/core/logx/writer.go
+++ b/core/logx/writer.go
@@ -263,7 +263,18 @@ func buildFields(fields ...LogField) []string {
  return items
 }

+func combineGlobalFields(fields []LogField) []LogField {

* globals := globalFields.Load()
* if globals == nil {
* return fields
* }
*
* return append(globals.([]LogField), fields...)
+}
*

 func output(writer io.Writer, level string, val interface{}, fields ...LogField) {

* fields = combineGlobalFields(fields)
*
  switch atomic.LoadUint32(&encoding) {
  case plainEncodingType:
   writePlainAny(writer, level, val, buildFields(fields...)...)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*logx*&#39; -g &#39;go.mod&#39; | head -100 &amp;&amp; \
rg -n \&#34;package logx|type.*Logger|WithFields|AddField|serviceName|ServiceName\&#34; --glob &#39;*.go&#39; | head -160&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; .. . | head -60; sed -n &#39;1,240p&#39; core/logx/richlogger.go; sed -n &#39;1,200p&#39; core/logx/fields.go; sed -n &#39;1,120p&#39; core/logx/logger.go; sed -n &#39;1,140p&#39; core/logx/config.go; sed -n &#39;1,140p&#39; core/logx/vars.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; core/logx/logs.go; sed -n &#39;1,180p&#39; core/logx/fields_test.go; sed -n &#39;1,300p&#39; core/logx/richlogger_test.go; rg -n \&#34;Set.*Log|LogConf|Reset.*Logger|SetWriter|WithContext\&#34; core/logx&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;240,430p&#39; core/logx/logs.go; rg -n \&#34;func write|buildFields\&#34; core/logx/logs.go; head -80 core/logx/readme-cn.md; head -90 core/logx/readme.md; rg -n \&#34;global.*field|AddField\&#34; README* core/logx&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;417,475p&#39; core/logx/logs.go; rg -n \&#34;func \\(.*\\) (Alert|Severe|Stat|Stack|Debug|Info)\&#34; core/logx&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>7</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>8</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>9</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>10</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;WithDuration&#34; core/logx/readme*.md; tail -20 core/logx/readme.md; git diff --stat&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>12</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/logs.go core/logx/richlogger.go core/logx/richlogger_test.go &amp;&amp; \
go test ./core/logx&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- core/logx/logs.go core/logx/richlogger.go core/logx/richlogger_test.go core/logx/readme.md core/logx/readme-cn.md; sed -n &#39;170,230p&#39; core/logx/writer.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;func output|func write&#34; core/logx/writer.go&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;266,320p&#39; core/logx/writer.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>17</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>18</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>19</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>
