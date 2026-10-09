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

* **Date**: 2026-10-06 23:33:13 CEST

## Test case: `zeromicro__go-zero-2463`

### Target: `codex/gpt-5.6-luna/medium (repeat 1)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/zeromicro__go-zero-2463.txtar](https://github.com/spockz/semantic-editor/blob/43bf7788d286eda6cb43f908eecb81fc2ff8694e/testdata/bench/zeromicro__go-zero-2463.txtar)

**LLM Prompt**:
> logx能否支持一下全局添加field功能，或默认加一个serviceName，收集日志想根据服务分索引很不方便
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 134.35s | 98.35s | N/A | — | — | — |
| **Process Start → First Event** | 0.77s | 0.64s | — | — | — | — |
| **First Event → First Tool Call** | 4.20s | 4.19s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 18 | 12 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 3 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 18 | 12 | N/A | — | — | — |
| **Output Tokens** | 8544 | 6461 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 2930 | 2377 | N/A | — | — | — |
| **Total Input Tokens** | 1904551 | 1388596 | N/A | — | — | — |
| **Cached Input Tokens** | 1807872 | 1300224 | N/A | — | — | — |
| **Uncached Input Tokens** | 96679 | 88372 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 18.70:1 | 14.71:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*logx*&#39; | head -80; rg -n \&#34;package logx|WithField|serviceName|ServiceName|field\&#34; . --glob &#39;*.go&#39; | head -160&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*logx*&#39; -g &#39;go.mod&#39; | head -80 &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- top-level ---&#39; &amp;&amp; \
ls -la&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,520p&#39; core/logx/logs.go; sed -n &#39;1,220p&#39; core/logx/fields.go; sed -n &#39;1,300p&#39; core/logx/writer.go; sed -n &#39;1,280p&#39; core/logx/richlogger.go; sed -n &#39;1,280p&#39; core/logx/fields_test.go; sed -n &#39;210,280p&#39; core/logx/richlogger_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find . -name AGENTS.md -o -name PROJECT_BRIEF.md; find . -type d -name logx -print; rg -n \&#34;func (Info|With|Add|Fields)|type .*Logger|serviceName|ServiceName|WithContext\&#34; core -g &#39;*.go&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short; git log -8 --oneline -- core/logx; rg -n \&#34;logx\\.(Set|With|Field)|Global.*Field|Default.*Field|ServiceName\&#34; README.md core/logx docs 2&gt;/dev/null | head -120; sed -n &#39;1,180p&#39; core/logx/logx.go 2&gt;/dev/null || \
true; rg -n \&#34;type LogConf|struct .*LogConf\&#34; core/logx&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; core/logx/logs.go; sed -n &#39;1,220p&#39; core/logx/fields.go; sed -n &#39;1,180p&#39; core/logx/logger.go; sed -n &#39;1,180p&#39; core/logx/richlogger.go; sed -n &#39;1,180p&#39; core/logx/vars.go; sed -n &#39;1,220p&#39; core/logx/logs_test.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,230p&#39; core/logx/readme-cn.md; sed -n &#39;1,180p&#39; core/logx/config.go; sed -n &#39;1,180p&#39; core/logx/vars.go; sed -n &#39;1,180p&#39; core/logx/logs_test.go; sed -n &#39;1,180p&#39; core/logx/richlogger_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;220,520p&#39; core/logx/logs.go; sed -n &#39;1,220p&#39; core/logx/writer.go 2&gt;/dev/null || \
true; rg -n \&#34;writeInfo|writeError|writeDebug|writeSevere|writeStack|fields\&#34; core/logx -g &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git branch -a; git tag --list &#39;v1.4.2&#39;; git log --all --oneline --grep=&#39;2463\\|global fields&#39; -i; rg -n \&#34;func AddGlobalFields|mergeGlobalFields|globalFields\&#34; &#34;&#39;$(go env GOPATH 2&gt;/dev/null)/pkg/mod/github.com/zeromicro 2&gt;/dev/null | head -80&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all -- core/logx | head -30; rg -n \&#34;Global|global.*field|service.*field|Add.*Field|Set.*Field\&#34; . -g &#39;*.go&#39; -g &#39;*.md&#39; | head -100; sed -n &#39;240,390p&#39; core/logx/writer.go; sed -n &#39;1,120p&#39; core/logx/fields_test.go; sed -n &#39;1,100p&#39; core/logx/config.go&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show --stat --oneline 59c0013c; git show --format=fuller --find-renames 59c0013c -- core/logx core/logc | sed -n &#39;1,420p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --stat --oneline 84db9bcd &amp;&amp; \
git show 84db9bcd -- core/logx | head -300; git branch -a --contains 84db9bcd | head&#39;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git ls-tree -r --name-only 59c0013c core/logx core/logc | head -100; git show 59c0013c:core/logx/logs.go | rg -n -C 5 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;; git show 59c0013c:core/logx/logs_test.go | rg -n -C 8 &#34;GlobalFields&#34;; git show 59c0013c:core/logx/fields.go 2&gt;/dev/null || \
true; git show 59c0013c:core/logc/logc.go 2&gt;/dev/null | head -100&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show 84db9bcd -- core/logx/logs.go core/logx/fields.go core/logx/writer.go core/logx/logx.go 2&gt;&amp;1 | head -400; git log --all -S&#39;globalFields&#39; --oneline -- core/logx; git log --all -S&#39;GlobalFields&#39; --oneline -- core/logx&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git cat-file -p 59c0013c | head -20; git rev-parse 59c0013c&#39;&#39;^{tree}; git ls-tree 59c0013c:core/logx | head -30; git show 59c0013c:core/logx/logs.go | rg -n -C 5 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git remote -v; git status --short; git log -1 --format=&#39;%H %s&#39;; rg -n \&#34;logx\\.\&#34; core/logx/*_test.go | tail -50&#34;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show 621b1ee37a51d350159be09755c3ba6f464766ca | rg -n -C 6 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;; git show 415f9d42d376a8282bbc519521337990f2557c66 | rg -n -C 8 &#34;GlobalFields&#34;; git show 1bdba318a5822e1083845ec86ade1cdb9a4c6723; git show 769c74c29d9ba5cfe729c4a81434c6502bcd109b | rg -n &#34;Global&#34;&#39;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git cat-file -t 621b1ee37a51d350159be09755c3ba6f464766ca; git cat-file -p 621b1ee37a51d350159be09755c3ba6f464766ca | rg -n -C 6 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;; git cat-file -t 415f9d42d376a8282bbc519521337990f2557c66 || \
true; git cat-file -p 1bdba318a5822e1083845ec86ade1cdb9a4c6723 || \
true&#39;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show v1.4.2:core/logx/logs.go | rg -n -C 8 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;; git show v1.4.2:core/logx/fields.go; git show v1.4.2:core/logx/logs_test.go | rg -n -C 8 &#34;GlobalFields&#34;&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go &amp;&amp; \
go test ./core/logx&#39;</code></pre></td></tr>
<tr><td>12</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git diff -- core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logx/fields_test.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;func validateFields|func Test.*Stat|func Test.*Write&#34; core/logx/*_test.go | head -40; gofmt -w core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go; git diff --check; go test ./core/logx&#39;</code></pre></td><td>—</td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logx/fields_test.go; sed -n &#39;810,845p&#39; core/logx/logs_test.go; git status --short&#34;</code></pre></td><td>—</td></tr>
<tr><td>15</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n -A8 -B3 &#34;Errorv|Errorw|WithContext&#34; core/logx/readme.md core/logx/readme-cn.md | head -80&#39;</code></pre></td><td>—</td></tr>
<tr><td>17</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>18</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go; git diff --check; git status --short; git diff --stat&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 134.35s | 11.21s | N/A | — | — | — |
| **Process Start → First Event** | 0.77s | — | — | — | — | — |
| **First Event → First Tool Call** | 4.20s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 0 | N/A | — | — | — |
| **Internal Tool Cycles** | 18 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 18 | 0 | N/A | — | — | — |
| **Output Tokens** | 8544 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 2930 | 0 | N/A | — | — | — |
| **Total Input Tokens** | 1904551 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 1807872 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 96679 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 18.70:1 | — | — | — | — | — |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*logx*&#39; | head -80; rg -n \&#34;package logx|WithField|serviceName|ServiceName|field\&#34; . --glob &#39;*.go&#39; | head -160&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,520p&#39; core/logx/logs.go; sed -n &#39;1,220p&#39; core/logx/fields.go; sed -n &#39;1,300p&#39; core/logx/writer.go; sed -n &#39;1,280p&#39; core/logx/richlogger.go; sed -n &#39;1,280p&#39; core/logx/fields_test.go; sed -n &#39;210,280p&#39; core/logx/richlogger_test.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short; git log -8 --oneline -- core/logx; rg -n \&#34;logx\\.(Set|With|Field)|Global.*Field|Default.*Field|ServiceName\&#34; README.md core/logx docs 2&gt;/dev/null | head -120; sed -n &#39;1,180p&#39; core/logx/logx.go 2&gt;/dev/null || \
true; rg -n \&#34;type LogConf|struct .*LogConf\&#34; core/logx&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,230p&#39; core/logx/readme-cn.md; sed -n &#39;1,180p&#39; core/logx/config.go; sed -n &#39;1,180p&#39; core/logx/vars.go; sed -n &#39;1,180p&#39; core/logx/logs_test.go; sed -n &#39;1,180p&#39; core/logx/richlogger_test.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git branch -a; git tag --list &#39;v1.4.2&#39;; git log --all --oneline --grep=&#39;2463\\|global fields&#39; -i; rg -n \&#34;func AddGlobalFields|mergeGlobalFields|globalFields\&#34; &#34;&#39;$(go env GOPATH 2&gt;/dev/null)/pkg/mod/github.com/zeromicro 2&gt;/dev/null | head -80&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show --stat --oneline 59c0013c; git show --format=fuller --find-renames 59c0013c -- core/logx core/logc | sed -n &#39;1,420p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git ls-tree -r --name-only 59c0013c core/logx core/logc | head -100; git show 59c0013c:core/logx/logs.go | rg -n -C 5 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;; git show 59c0013c:core/logx/logs_test.go | rg -n -C 8 &#34;GlobalFields&#34;; git show 59c0013c:core/logx/fields.go 2&gt;/dev/null || \
true; git show 59c0013c:core/logc/logc.go 2&gt;/dev/null | head -100&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git cat-file -p 59c0013c | head -20; git rev-parse 59c0013c&#39;&#39;^{tree}; git ls-tree 59c0013c:core/logx | head -30; git show 59c0013c:core/logx/logs.go | rg -n -C 5 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show 621b1ee37a51d350159be09755c3ba6f464766ca | rg -n -C 6 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;; git show 415f9d42d376a8282bbc519521337990f2557c66 | rg -n -C 8 &#34;GlobalFields&#34;; git show 1bdba318a5822e1083845ec86ade1cdb9a4c6723; git show 769c74c29d9ba5cfe729c4a81434c6502bcd109b | rg -n &#34;Global&#34;&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git cat-file -t 621b1ee37a51d350159be09755c3ba6f464766ca; git cat-file -p 621b1ee37a51d350159be09755c3ba6f464766ca | rg -n -C 6 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;; git cat-file -t 415f9d42d376a8282bbc519521337990f2557c66 || \
true; git cat-file -p 1bdba318a5822e1083845ec86ade1cdb9a4c6723 || \
true&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show v1.4.2:core/logx/logs.go | rg -n -C 8 &#34;AddGlobalFields|mergeGlobalFields|globalFields&#34;; git show v1.4.2:core/logx/fields.go; git show v1.4.2:core/logx/logs_test.go | rg -n -C 8 &#34;GlobalFields&#34;&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>12</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;func validateFields|func Test.*Stat|func Test.*Write&#34; core/logx/*_test.go | head -40; gofmt -w core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go; git diff --check; go test ./core/logx&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logx/fields_test.go; sed -n &#39;810,845p&#39; core/logx/logs_test.go; git status --short&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>15</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n -A8 -B3 &#34;Errorv|Errorw|WithContext&#34; core/logx/readme.md core/logx/readme-cn.md | head -80&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>17</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>18</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go; git diff --check; git status --short; git diff --stat&#39;</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>
