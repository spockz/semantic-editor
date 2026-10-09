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

* **Date**: 2026-10-07 08:22:05 CEST

## Test case: `zeromicro__go-zero-2463`

### Target: `codex/gpt-6-luna/high (repeat 2)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

* **Run Provenance**: `coding_network=disabled,oracle_patch_installation=checked`

* **Fixture**: [testdata/bench/zeromicro__go-zero-2463.txtar](https://github.com/spockz/semantic-editor/blob/43bf7788d286eda6cb43f908eecb81fc2ff8694e/testdata/bench/zeromicro__go-zero-2463.txtar)

**LLM Prompt**:
> logx能否支持一下全局添加field功能，或默认加一个serviceName，收集日志想根据服务分索引很不方便
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 72.57s | 126.71s | N/A | — | — | — |
| **Process Start → First Event** | 0.65s | 0.64s | — | — | — | — |
| **First Event → First Tool Call** | 4.14s | 5.69s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 13 | 14 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 5 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 13 | 14 | N/A | — | — | — |
| **Output Tokens** | 5699 | 5524 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 3540 | 3204 | N/A | — | — | — |
| **Total Input Tokens** | 427362 | 410651 | N/A | — | — | — |
| **Cached Input Tokens** | 380416 | 374528 | N/A | — | — | — |
| **Uncached Input Tokens** | 46946 | 36123 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.10:1 | 10.37:1 | N/A | — | — | — |
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
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*logx*&#39; -g &#39;*.go&#39; | head -120&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; | head -60 &amp;&amp; \
rg -n \&#34;WithFields|AddField|serviceName|logx\&#34; logx&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;(&#34;&#39;^|/)(AGENTS&#39;&#34;\\.md|PROJECT_BRIEF\\.md)&#34;&#39;$|logx/&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -a &amp;&amp; \
rg --files | rg &#39;(&#34;&#39;^|/)logx|AGENTS&#39;&#34;\\.md|PROJECT_BRIEF\\.md&#39; | head -100 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; .. | head -60 &amp;&amp; \
sed -n &#39;1,240p&#39; core/logx/config.go &amp;&amp; \
sed -n &#39;1,240p&#39; core/logx/fields.go &amp;&amp; \
sed -n &#39;1,200p&#39; core/logx/readme-cn.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; core/logx/config.go &amp;&amp; \
sed -n &#39;1,240p&#39; core/logx/fields.go &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/logs.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; core/logx/logs.go &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/vars.go &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/logwriter.go &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/fields_test.go &amp;&amp; \
rg &#39;ServiceName|serviceName|Set.*Field|Add.*Field&#39; core/logx&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; core/logx/readme.md &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/readme-cn.md&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;220,560p&#39; core/logx/logs.go &amp;&amp; \
sed -n &#39;1,300p&#39; core/logx/logger.go &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/richlogger.go &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/writer.go &amp;&amp; \
sed -n &#39;1,200p&#39; core/logx/readme.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;NewLogger|WithFields|Fields|Config\\{\&#34; core/logx&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg &#39;func output|func (write|marshal)&#39; core/logx -n &amp;&amp; \
rg &#39;LogConf\\{&#39; --glob &#39;*.go&#39; | head -80 &amp;&amp; \
rg &#39;ServiceName =&#39; rest zrpc -n | head -60&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; core/logx/richlogger.go &amp;&amp; \
sed -n &#39;1,330p&#39; core/logx/writer.go &amp;&amp; \
rg -n \&#34;ServiceName\&#34;&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;250,330p&#39; core/logx/writer.go &amp;&amp; \
rg &#39;ServiceName&#39; rest zrpc core/service -n | head -80 &amp;&amp; \
rg &#39;logx.SetUp|Log.SetUp|SetUp\\(.*Log&#39; -n rest zrpc&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;260,470p&#39; core/logx/logs.go &amp;&amp; \
sed -n &#39;1,120p&#39; core/logx/vars.go &amp;&amp; \
sed -n &#39;1,130p&#39; core/logx/fields_test.go &amp;&amp; \
sed -n &#39;210,280p&#39; core/logx/richlogger_test.go&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; core/service/serviceconf.go &amp;&amp; \
rg &#39;SetUp\\(sc.Log|SetUp\\(.*Log&#39; -n core/service rest zrpc&#34;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>9</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/logs.go core/logx/vars.go core/logx/writer.go &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- core/logx/logs.go core/logx/vars.go core/logx/writer.go core/logx/readme-cn.md core/logx/readme.md &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build ./core/logx&#39;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>12</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/fields.go core/logx/logs.go core/logx/richlogger.go &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logx/readme.md core/logx/readme-cn.md&#39;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff --stat&#39;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
</tbody>
</table>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 72.57s | 10.94s | N/A | — | — | — |
| **Process Start → First Event** | 0.65s | — | — | — | — | — |
| **First Event → First Tool Call** | 4.14s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 0 | N/A | — | — | — |
| **Internal Tool Cycles** | 13 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 13 | 0 | N/A | — | — | — |
| **Output Tokens** | 5699 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 3540 | 0 | N/A | — | — | — |
| **Total Input Tokens** | 427362 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 380416 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 46946 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.10:1 | — | — | — | — | — |
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
+	"context"
+	"sync"
+	"sync/atomic"
+)
 
-var fieldsContextKey contextKey
+var (
+	fieldsContextKey contextKey
+	globalFields     atomic.Value
+	globalFieldsLock sync.Mutex
+)
 
 type contextKey struct{}
 
+// AddGlobalFields adds global fields.
+func AddGlobalFields(fields ...LogField) {
+	globalFieldsLock.Lock()
+	defer globalFieldsLock.Unlock()
+
+	old := globalFields.Load()
+	if old == nil {
+		globalFields.Store(append([]LogField(nil), fields...))
+	} else {
+		globalFields.Store(append(old.([]LogField), fields...))
+	}
+}
+
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
+	globals := globalFields.Load()
+	if globals == nil {
+		return fields
+	}
+
+	return append(globals.([]LogField), fields...)
+}
+
 func output(writer io.Writer, level string, val interface{}, fields ...LogField) {
+	fields = combineGlobalFields(fields)
+
 	switch atomic.LoadUint32(&encoding) {
 	case plainEncodingType:
 		writePlainAny(writer, level, val, buildFields(fields...)...)


<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*logx*&#39; -g &#39;*.go&#39; | head -120&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;(&#34;&#39;^|/)(AGENTS&#39;&#34;\\.md|PROJECT_BRIEF\\.md)&#34;&#39;$|logx/&#39;&#34;&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;&#34;&#39;!vendor&#39;&#34;&#39; .. | head -60 &amp;&amp; \
sed -n &#39;1,240p&#39; core/logx/config.go &amp;&amp; \
sed -n &#39;1,240p&#39; core/logx/fields.go &amp;&amp; \
sed -n &#39;1,200p&#39; core/logx/readme-cn.md&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; core/logx/logs.go &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/vars.go &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/logwriter.go &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/fields_test.go &amp;&amp; \
rg &#39;ServiceName|serviceName|Set.*Field|Add.*Field&#39; core/logx&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;220,560p&#39; core/logx/logs.go &amp;&amp; \
sed -n &#39;1,300p&#39; core/logx/logger.go &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/richlogger.go &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/writer.go &amp;&amp; \
sed -n &#39;1,200p&#39; core/logx/readme.md&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg &#39;func output|func (write|marshal)&#39; core/logx -n &amp;&amp; \
rg &#39;LogConf\\{&#39; --glob &#39;*.go&#39; | head -80 &amp;&amp; \
rg &#39;ServiceName =&#39; rest zrpc -n | head -60&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;250,330p&#39; core/logx/writer.go &amp;&amp; \
rg &#39;ServiceName&#39; rest zrpc core/service -n | head -80 &amp;&amp; \
rg &#39;logx.SetUp|Log.SetUp|SetUp\\(.*Log&#39; -n rest zrpc&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; core/service/serviceconf.go &amp;&amp; \
rg &#39;SetUp\\(sc.Log|SetUp\\(.*Log&#39; -n core/service rest zrpc&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>9</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/logs.go core/logx/vars.go core/logx/writer.go &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- core/logx/logs.go core/logx/vars.go core/logx/writer.go core/logx/readme-cn.md core/logx/readme.md &amp;&amp; \
git status --short&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build ./core/logx&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>12</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff --stat&#39;</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>

