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

### Target: `codex/gpt-5.6-luna/high (repeat 1)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

* **Fixture**: [testdata/bench/zeromicro__go-zero-2463.txtar](https://github.com/spockz/semantic-editor/blob/43bf7788d286eda6cb43f908eecb81fc2ff8694e/testdata/bench/zeromicro__go-zero-2463.txtar)

**LLM Prompt**:
> logx能否支持一下全局添加field功能，或默认加一个serviceName，收集日志想根据服务分索引很不方便
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 440.81s | 215.42s | N/A | — | — | — |
| **Process Start → First Event** | 0.24s | 0.13s | — | — | — | — |
| **First Event → First Tool Call** | 8.82s | 3.56s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 35 | 27 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 3 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 35 | 27 | N/A | — | — | — |
| **Output Tokens** | 18195 | 13381 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 8131 | 5738 | N/A | — | — | — |
| **Total Input Tokens** | 3294590 | 3932265 | N/A | — | — | — |
| **Cached Input Tokens** | 3147520 | 3785216 | N/A | — | — | — |
| **Uncached Input Tokens** | 147070 | 147049 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 21.40:1 | 25.74:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*logx*&#39; -g &#39;go.mod&#39; | sed -n &#39;1,120p&#39; &amp;&amp; \
printf &#39;\\n--- root docs ---\\n&#39; &amp;&amp; \
if [ -f AGENTS.md ]; then sed -n &#39;1,240p&#39; AGENTS.md; fi &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*logx*&#39; -g &#39;*.go&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;(&#34;&#39;^|/)logx|logx&#39;&#34;&#39; | sed -n &#39;1,160p&#39; &amp;&amp; \
printf &#39;\\n--- module ---\\n&#39; &amp;&amp; \
sed -n &#39;1,100p&#39; go.mod &amp;&amp; \
printf &#39;\\n--- top-level dirs ---\\n&#39; &amp;&amp; \
find . -maxdepth 2 -type d | sort | sed -n &#39;1,120p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- root docs ---&#39; &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; | head -80 &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logx files ---&#39; &amp;&amp; \
rg --files | rg &#39;(&#34;&#39;^|/)logx(/|$)&#39;&#34;&#39; | head -200&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- fields.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/fields.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logs.go ---&#39; &amp;&amp; \
sed -n &#39;1,320p&#39; core/logx/logs.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logger.go ---&#39; &amp;&amp; \
sed -n &#39;1,320p&#39; core/logx/logger.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- vars/config ---&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/vars.go &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/config.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tests ---&#39; &amp;&amp; \
sed -n &#39;1,300p&#39; core/logx/fields_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/logs_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; core/logx/readme-cn.md &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- config ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/config.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- vars ---&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/vars.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- fields ---&#39; &amp;&amp; \
sed -n &#39;1,280p&#39; core/logx/fields.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logs ---&#39; &amp;&amp; \
sed -n &#39;1,320p&#39; core/logx/logs.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;220,520p&#39; core/logx/logs.go &amp;&amp; \
printf &#39;\\n--- richlogger ---\\n&#39; &amp;&amp; \
sed -n &#39;1,360p&#39; core/logx/richlogger.go &amp;&amp; \
printf &#39;\\n--- writer interfaces and outputs ---\\n&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/writer.go &amp;&amp; \
sed -n &#39;1,300p&#39; core/logx/logwriter.go &amp;&amp; \
printf &#39;\\n--- docs fields references ---\\n&#39; &amp;&amp; \
rg -n \&#34;ContextWithFields|WithFields|Field\\(|ServiceName|serviceName|LogField|Infow|Errorw\&#34; core/logx --glob &#39;*.go&#39; --glob &#39;*.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;320,760p&#39; core/logx/logs.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logger ---&#39; &amp;&amp; \
sed -n &#39;1,340p&#39; core/logx/logger.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- richlogger ---&#39; &amp;&amp; \
sed -n &#39;1,360p&#39; core/logx/richlogger.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logwriter ---&#39; &amp;&amp; \
sed -n &#39;1,360p&#39; core/logx/logwriter.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- writer ---&#39; &amp;&amp; \
sed -n &#39;1,300p&#39; core/logx/writer.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all -- core/logx | sed -n &#39;1,80p&#39; &amp;&amp; \
printf &#39;\\n--- current branch/status ---\\n&#39; &amp;&amp; \
git status --short --branch &amp;&amp; \
printf &#39;\\n--- writer output ---\\n&#39; &amp;&amp; \
sed -n &#39;250,360p&#39; core/logx/writer.go &amp;&amp; \
printf &#39;\\n--- rich tests context ---\\n&#39; &amp;&amp; \
sed -n &#39;1,300p&#39; core/logx/richlogger_test.go &amp;&amp; \
printf &#39;\\n--- readme API area ---\\n&#39; &amp;&amp; \
sed -n &#39;1,130p&#39; core/logx/readme-cn.md&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;300,620p&#39; core/logx/writer.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- fields tests ---&#39; &amp;&amp; \
sed -n &#39;1,280p&#39; core/logx/fields_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logs tests ---&#39; &amp;&amp; \
sed -n &#39;1,420p&#39; core/logx/logs_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- richlogger tests ---&#39; &amp;&amp; \
sed -n &#39;1,420p&#39; core/logx/richlogger_test.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- search service name/fields APIs ---&#39; &amp;&amp; \
rg -n \&#34;ServiceName|ContextWithFields|WithFields|LogField|Infow|Errorw|global field|Global\&#34; core/logx core | head -300&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show --stat --oneline 59c0013c &amp;&amp; \
git show --format=fuller --no-ext-diff 59c0013c -- core/logx core/logc | sed -n &#39;1,420p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git log --oneline -8 -- core/logx &amp;&amp; \
rg -n \&#34;Set.*Field|Global.*Field|Add.*Field|serviceName\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39; | head -200 &amp;&amp; \
sed -n &#39;1,120p&#39; core/service/serviceconf.go &amp;&amp; \
sed -n &#39;540,660p&#39; core/logx/logs_test.go&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git cat-file -p 59c0013c | sed -n &#39;1,40p&#39;; printf &#39;\\n--- tree paths ---\\n&#39;; git ls-tree -r --name-only 59c0013c | rg &#39;core/logx|core/logc&#39; | sed -n &#39;1,120p&#39;; printf &#39;\\n--- current ancestry ---\\n&#39;; git log -1 --format=&#39;%H %P %s&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline -- core/logx | head -30; git remote -v; git branch -a --no-color | head -40; rg -n \&#34;SetGlobal|GlobalFields|global fields|AddField|AddFields\&#34; . --glob &#39;*.go&#39; --glob &#39;*.md&#39; --glob &#39;&#34;&#39;!tools/goctl/api/parser/g4/gen/**&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --no-ext-diff --stat HEAD 59c0013c -- core/logx core/logc &amp;&amp; \
git diff --no-ext-diff HEAD 59c0013c -- core/logx core/logc | sed -n &#39;1,520p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show --stat --oneline 84db9bcd &amp;&amp; \
git show --format=fuller --find-renames 84db9bcd -- core/logx | sed -n &#39;1,360p&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- related commits ---&#39; &amp;&amp; \
git log --all --oneline --grep=&#39;global field\\|global fields\\|service name&#39; -i | head -50&#34;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- core/logx/fields.go at target commit ---&#39; &amp;&amp; \
git show 59c0013c:core/logx/fields.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- core/logx/logs.go relevant ---&#39; &amp;&amp; \
git show 59c0013c:core/logx/logs.go | sed -n &#39;1,520p&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- core/logc/logs.go ---&#39; &amp;&amp; \
git show 59c0013c:core/logc/logs.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- core/logc tests ---&#39; &amp;&amp; \
git show 59c0013c:core/logc/logs_test.go | sed -n &#39;1,360p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline -S&#39;globalFields&#39; -- core/logx; git log --all --oneline -S&#39;AddGlobal&#39; -- core/logx; git log --all --oneline -S&#39;GlobalField&#39; -- core/logx; git log --all --oneline -S&#39;GlobalFields&#39; -- core/logx; git log --all --oneline -S&#39;global fields&#39; -- core/logx | head -40&#34;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git cat-file -p HEAD | sed -n &#39;1,8p&#39;; git merge-base --is-ancestor 59c0013c HEAD; echo target_is_ancestor=&#34;&#39;$?; git merge-base --is-ancestor HEAD 59c0013c; echo head_is_ancestor=$?; git log --oneline --decorate --all --graph -20 | sed -n &#39;&#34;&#39;1,80p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git rev-parse master &amp;&amp; \
git show master:core/logx/logs.go 2&gt;&amp;1 | rg -n \&#34;Global|global|ServiceName|field\&#34; | head -80; git show master:core/logx/fields.go 2&gt;&amp;1 | sed -n &#39;1,160p&#39;&#34;</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all -- core/logc | sed -n &#39;1,30p&#39;; git ls-tree HEAD core/logc core/logx/fields.go; git ls-tree 59c0013c core/logc core/logx/fields.go; git status --short; git log --oneline HEAD -- core/logx/fields.go | sed -n &#39;1,20p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;func mergeGlobalFields|AddGlobalFields|globalFields&#34; /Users/alessandro/go/pkg/mod 2&gt;/dev/null | head -100&#39;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- fields after feature ---&#39; &amp;&amp; \
git show a32f6d76:core/logx/fields.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logs after feature relevant ---&#39; &amp;&amp; \
git show a32f6d76:core/logx/logs.go | rg -n -C 5 &#39;AddGlobalFields|mergeGlobalFields|writeDebug|writeError|writeInfo|writeSlow|writeStat&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- rich after feature relevant ---&#39; &amp;&amp; \
git show a32f6d76:core/logx/richlogger.go | rg -n -C 8 &#39;buildFields|global|fieldsKey&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logc at feature ---&#39; &amp;&amp; \
git show 59c0013c:core/logc/logs.go 2&gt;&amp;1 | sed -n &#39;1,240p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; core/logx/readme.md &amp;&amp; \
cat go.mod | sed -n &#39;1,60p&#39; &amp;&amp; \
rg -n \&#34;go test|golangci|gofmt|Makefile\&#34; README.md Makefile .github 2&gt;/dev/null | head -100&#34;</code></pre></td></tr>
<tr><td>13</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>14</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logc/logs.go &amp;&amp; \
go test ./core/logx ./core/logc&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go &amp;&amp; \
go test ./core/logx&#39;</code></pre></td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func validateFields|func Test.*Stat|func Test.*Global|globalFields|SetLevel\&#34; core/logx/*_test.go | sed -n &#39;1,200p&#39; &amp;&amp; \
sed -n &#39;720,820p&#39; core/logx/logs_test.go &amp;&amp; \
git diff -- core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logc/logs.go | sed -n &#39;1,420p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./core/logx&#39;</code></pre></td></tr>
<tr><td>17</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;818,850p&#39; core/logx/logs_test.go &amp;&amp; \
printf &#39;\\n--- git diff stat ---\\n&#39; &amp;&amp; \
git diff --stat &amp;&amp; \
printf &#39;\\n--- tracked/untracked ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>18</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find .scratch -maxdepth 4 -type d -name &#39;clickhouse-go*&#39; -o -name &#39;testify*&#39; | sed -n &#39;1,40p&#39;; go env GOMODCACHE GOPATH GOPROXY; rg -n \&#34;AddGlobalFields|mergeGlobalFields\&#34; &#34;&#39;$(go env GOPATH 2&gt;/dev/null)/pkg/mod/github.com/zeromicro 2&gt;/dev/null | sed -n &#39;&#34;&#39;1,80p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/logs.go &amp;&amp; \
go test ./core/logx&#39;</code></pre></td></tr>
<tr><td>19</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in core/logx/readme.md core/logx/readme-cn.md core/logx/logger.go core/logx/richlogger.go core/logx/logs.go core/logx/fields.go; do printf &#39;%s &#39; \&#34;&#34;&#39;$f&#34;; git ls-tree 59c0013c &#34;$f&#34; | awk &#39;&#34;&#39;{print &#34;&#39;$3}&#39;&#34;&#39;; git ls-tree HEAD \&#34;&#34;&#39;$f&#34; | awk &#39;&#34;&#39;{print &#34;&#39;$3}&#39;&#34;&#39;; done&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git diff -- core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logx/fields_test.go core/logx/readme.md core/logx/readme-cn.md &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>20</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in core/logx/writer.go core/logx/richlogger.go core/logx/logs.go core/logx/fields_test.go core/logx/logs_test.go; do printf &#39;%s\\n&#39; \&#34;&#34;&#39;$f&#34;; git ls-tree 59c0013c &#34;$f&#34;; git ls-tree HEAD &#34;$f&#34;; done; printf &#39;&#34;&#39;\\n--- target tree top ---\\n&#39;; git ls-tree 59c0013c core/logx | head -1; git ls-tree HEAD core/logx | head -1&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;find .scratch -maxdepth 2 -type f -o -type d | head -40&#39;</code></pre></td></tr>
<tr><td>21</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git cat-file -t deeb1c84f006bc9eb7bbba98e75cc8a8602e6e5e; git show 59c0013c:core/logx/writer.go 2&gt;&amp;1 | sed -n &#39;1,360p&#39;; printf &#39;\\n--- target fields test ---\\n&#39;; git show 59c0013c:core/logx/fields_test.go 2&gt;&amp;1 | sed -n &#39;1,260p&#39;; printf &#39;\\n--- target logs test global refs ---\\n&#39;; git show 59c0013c:core/logx/logs_test.go 2&gt;&amp;1 | rg -n -C 8 &#39;Global|global|merge&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test -race ./core/logx&#39;</code></pre></td></tr>
<tr><td>22</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./core/...&#39;</code></pre></td></tr>
<tr><td>23</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go core/logc/logs.go &amp;&amp; \
go test ./core/logx ./core/logc&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --check &amp;&amp; \
git status --short &amp;&amp; \
nl -ba core/logx/fields.go | sed -n &#39;1,120p&#39; &amp;&amp; \
nl -ba core/logx/logs.go | sed -n &#39;410,465p&#39; &amp;&amp; \
nl -ba core/logx/richlogger.go | sed -n &#39;120,145p&#39;&#34;</code></pre></td></tr>
<tr><td>24</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --check &amp;&amp; \
git diff --stat &amp;&amp; \
git status --short &amp;&amp; \
git diff -- core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logx/fields_test.go core/logc/logs.go | sed -n &#39;1,520p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rm -rf .scratch &amp;&amp; \
git status --short &amp;&amp; \
git diff --check&#39;</code></pre></td></tr>
<tr><td>25</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;du -sh .scratch 2&gt;/dev/null; find .scratch -maxdepth 3 -type f | sed -n &#39;1,40p&#39;; git check-ignore -v .scratch .scratch/go .scratch/go/mod 2&gt;/dev/null || \
true; rg -n \&#34;&#34;&#39;^&#39;&#34;\\.scratch|scratch\&#34; .gitignore .git/info/exclude 2&gt;/dev/null || \
true&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rm -rf .scratch &amp;&amp; \
git status --short &amp;&amp; \
git diff --check&#39;</code></pre></td></tr>
<tr><td>26</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test -race ./core/logx ./core/logc&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;mv .scratch /private/tmp/go-zero-logx-test-cache &amp;&amp; \
git status --short &amp;&amp; \
git diff --check&#39;</code></pre></td></tr>
<tr><td>27</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short --untracked-files=all | sed -n &#39;1,120p&#39;; printf &#39;\\n--- new package ---\\n&#39;; sed -n &#39;1,260p&#39; core/logc/logs.go; printf &#39;\\n--- diff check ---\\n&#39;; git diff --check; printf &#39;\\n--- package compile list ---\\n&#39;; go list ./core/logx ./core/logc&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --stat &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>28</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rm -rf -- .scratch/go &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td><td>—</td></tr>
<tr><td>29</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rm -rf -- .scratch/go &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td><td>—</td></tr>
<tr><td>30</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -ld .scratch .scratch/go .scratch/go/mod .scratch/go/mod/github.com/stretchr/testify@v1.8.0 2&gt;&amp;1; find .scratch/go -maxdepth 2 -type f -writable 2&gt;/dev/null | sed -n &#39;1,40p&#39;; git status --short --untracked-files=all 2&gt;/dev/null | tail -n 20&#34;</code></pre></td><td>—</td></tr>
<tr><td>31</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;chmod -R u+w -- .scratch/go &amp;&amp; \
rm -rf -- .scratch/go &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td><td>—</td></tr>
<tr><td>32</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;chmod -R u+w .scratch/go &amp;&amp; \
rm -rf .scratch/go &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td><td>—</td></tr>
<tr><td>33</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;len\\(.*Fields|assert\\.Equal.*\\[\\]LogField|Fields\\)|entry\\[|map\\[string\\]interface\\{\\}\&#34; core/logx/*_test.go | sed -n &#39;1,240p&#39;&#34;</code></pre></td><td>—</td></tr>
<tr><td>34</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go core/logc/logs.go &amp;&amp; \
git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td><td>—</td></tr>
<tr><td>35</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;nl -ba core/logx/fields.go | sed -n &#39;1,100p&#39;; nl -ba core/logx/logs.go | sed -n &#39;410,460p&#39;; nl -ba core/logx/richlogger.go | sed -n &#39;120,155p&#39;; nl -ba core/logc/logs.go | sed -n &#39;1,190p&#39;; nl -ba core/logx/fields_test.go | sed -n &#39;30,75p&#39;&#34;</code></pre></td><td>—</td></tr>
</tbody>
</table>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 440.81s | 10.10s | N/A | — | — | — |
| **Process Start → First Event** | 0.24s | — | — | — | — | — |
| **First Event → First Tool Call** | 8.82s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 0 | N/A | — | — | — |
| **Internal Tool Cycles** | 35 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 35 | 0 | N/A | — | — | — |
| **Output Tokens** | 18195 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 8131 | 0 | N/A | — | — | — |
| **Total Input Tokens** | 3294590 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 3147520 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 147070 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 21.40:1 | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
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
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*logx*&#39; -g &#39;go.mod&#39; | sed -n &#39;1,120p&#39; &amp;&amp; \
printf &#39;\\n--- root docs ---\\n&#39; &amp;&amp; \
if [ -f AGENTS.md ]; then sed -n &#39;1,240p&#39; AGENTS.md; fi &amp;&amp; \
if [ -f PROJECT_BRIEF.md ]; then sed -n &#39;1,240p&#39; PROJECT_BRIEF.md; fi&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;(&#34;&#39;^|/)logx|logx&#39;&#34;&#39; | sed -n &#39;1,160p&#39; &amp;&amp; \
printf &#39;\\n--- module ---\\n&#39; &amp;&amp; \
sed -n &#39;1,100p&#39; go.mod &amp;&amp; \
printf &#39;\\n--- top-level dirs ---\\n&#39; &amp;&amp; \
find . -maxdepth 2 -type d | sort | sed -n &#39;1,120p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- fields.go ---&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/fields.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logs.go ---&#39; &amp;&amp; \
sed -n &#39;1,320p&#39; core/logx/logs.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logger.go ---&#39; &amp;&amp; \
sed -n &#39;1,320p&#39; core/logx/logger.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- vars/config ---&#39; &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/vars.go &amp;&amp; \
sed -n &#39;1,220p&#39; core/logx/config.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- tests ---&#39; &amp;&amp; \
sed -n &#39;1,300p&#39; core/logx/fields_test.go &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/logs_test.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;220,520p&#39; core/logx/logs.go &amp;&amp; \
printf &#39;\\n--- richlogger ---\\n&#39; &amp;&amp; \
sed -n &#39;1,360p&#39; core/logx/richlogger.go &amp;&amp; \
printf &#39;\\n--- writer interfaces and outputs ---\\n&#39; &amp;&amp; \
sed -n &#39;1,260p&#39; core/logx/writer.go &amp;&amp; \
sed -n &#39;1,300p&#39; core/logx/logwriter.go &amp;&amp; \
printf &#39;\\n--- docs fields references ---\\n&#39; &amp;&amp; \
rg -n \&#34;ContextWithFields|WithFields|Field\\(|ServiceName|serviceName|LogField|Infow|Errorw\&#34; core/logx --glob &#39;*.go&#39; --glob &#39;*.md&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all -- core/logx | sed -n &#39;1,80p&#39; &amp;&amp; \
printf &#39;\\n--- current branch/status ---\\n&#39; &amp;&amp; \
git status --short --branch &amp;&amp; \
printf &#39;\\n--- writer output ---\\n&#39; &amp;&amp; \
sed -n &#39;250,360p&#39; core/logx/writer.go &amp;&amp; \
printf &#39;\\n--- rich tests context ---\\n&#39; &amp;&amp; \
sed -n &#39;1,300p&#39; core/logx/richlogger_test.go &amp;&amp; \
printf &#39;\\n--- readme API area ---\\n&#39; &amp;&amp; \
sed -n &#39;1,130p&#39; core/logx/readme-cn.md&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show --stat --oneline 59c0013c &amp;&amp; \
git show --format=fuller --no-ext-diff 59c0013c -- core/logx core/logc | sed -n &#39;1,420p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git cat-file -p 59c0013c | sed -n &#39;1,40p&#39;; printf &#39;\\n--- tree paths ---\\n&#39;; git ls-tree -r --name-only 59c0013c | rg &#39;core/logx|core/logc&#39; | sed -n &#39;1,120p&#39;; printf &#39;\\n--- current ancestry ---\\n&#39;; git log -1 --format=&#39;%H %P %s&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --no-ext-diff --stat HEAD 59c0013c -- core/logx core/logc &amp;&amp; \
git diff --no-ext-diff HEAD 59c0013c -- core/logx core/logc | sed -n &#39;1,520p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- core/logx/fields.go at target commit ---&#39; &amp;&amp; \
git show 59c0013c:core/logx/fields.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- core/logx/logs.go relevant ---&#39; &amp;&amp; \
git show 59c0013c:core/logx/logs.go | sed -n &#39;1,520p&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- core/logc/logs.go ---&#39; &amp;&amp; \
git show 59c0013c:core/logc/logs.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- core/logc tests ---&#39; &amp;&amp; \
git show 59c0013c:core/logc/logs_test.go | sed -n &#39;1,360p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git cat-file -p HEAD | sed -n &#39;1,8p&#39;; git merge-base --is-ancestor 59c0013c HEAD; echo target_is_ancestor=&#34;&#39;$?; git merge-base --is-ancestor HEAD 59c0013c; echo head_is_ancestor=$?; git log --oneline --decorate --all --graph -20 | sed -n &#39;&#34;&#39;1,80p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all -- core/logc | sed -n &#39;1,30p&#39;; git ls-tree HEAD core/logc core/logx/fields.go; git ls-tree 59c0013c core/logc core/logx/fields.go; git status --short; git log --oneline HEAD -- core/logx/fields.go | sed -n &#39;1,20p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;printf &#39;%s\\n&#39; &#39;--- fields after feature ---&#39; &amp;&amp; \
git show a32f6d76:core/logx/fields.go &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logs after feature relevant ---&#39; &amp;&amp; \
git show a32f6d76:core/logx/logs.go | rg -n -C 5 &#39;AddGlobalFields|mergeGlobalFields|writeDebug|writeError|writeInfo|writeSlow|writeStat&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- rich after feature relevant ---&#39; &amp;&amp; \
git show a32f6d76:core/logx/richlogger.go | rg -n -C 8 &#39;buildFields|global|fieldsKey&#39; &amp;&amp; \
printf &#39;%s\\n&#39; &#39;--- logc at feature ---&#39; &amp;&amp; \
git show 59c0013c:core/logc/logs.go 2&gt;&amp;1 | sed -n &#39;1,240p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>13</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>14</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logc/logs.go &amp;&amp; \
go test ./core/logx ./core/logc&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func validateFields|func Test.*Stat|func Test.*Global|globalFields|SetLevel\&#34; core/logx/*_test.go | sed -n &#39;1,200p&#39; &amp;&amp; \
sed -n &#39;720,820p&#39; core/logx/logs_test.go &amp;&amp; \
git diff -- core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logc/logs.go | sed -n &#39;1,420p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>17</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;818,850p&#39; core/logx/logs_test.go &amp;&amp; \
printf &#39;\\n--- git diff stat ---\\n&#39; &amp;&amp; \
git diff --stat &amp;&amp; \
printf &#39;\\n--- tracked/untracked ---\\n&#39; &amp;&amp; \
git status --short&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>18</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find .scratch -maxdepth 4 -type d -name &#39;clickhouse-go*&#39; -o -name &#39;testify*&#39; | sed -n &#39;1,40p&#39;; go env GOMODCACHE GOPATH GOPROXY; rg -n \&#34;AddGlobalFields|mergeGlobalFields\&#34; &#34;&#39;$(go env GOPATH 2&gt;/dev/null)/pkg/mod/github.com/zeromicro 2&gt;/dev/null | sed -n &#39;&#34;&#39;1,80p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>19</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in core/logx/readme.md core/logx/readme-cn.md core/logx/logger.go core/logx/richlogger.go core/logx/logs.go core/logx/fields.go; do printf &#39;%s &#39; \&#34;&#34;&#39;$f&#34;; git ls-tree 59c0013c &#34;$f&#34; | awk &#39;&#34;&#39;{print &#34;&#39;$3}&#39;&#34;&#39;; git ls-tree HEAD \&#34;&#34;&#39;$f&#34; | awk &#39;&#34;&#39;{print &#34;&#39;$3}&#39;&#34;&#39;; done&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>20</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in core/logx/writer.go core/logx/richlogger.go core/logx/logs.go core/logx/fields_test.go core/logx/logs_test.go; do printf &#39;%s\\n&#39; \&#34;&#34;&#39;$f&#34;; git ls-tree 59c0013c &#34;$f&#34;; git ls-tree HEAD &#34;$f&#34;; done; printf &#39;&#34;&#39;\\n--- target tree top ---\\n&#39;; git ls-tree 59c0013c core/logx | head -1; git ls-tree HEAD core/logx | head -1&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>21</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git cat-file -t deeb1c84f006bc9eb7bbba98e75cc8a8602e6e5e; git show 59c0013c:core/logx/writer.go 2&gt;&amp;1 | sed -n &#39;1,360p&#39;; printf &#39;\\n--- target fields test ---\\n&#39;; git show 59c0013c:core/logx/fields_test.go 2&gt;&amp;1 | sed -n &#39;1,260p&#39;; printf &#39;\\n--- target logs test global refs ---\\n&#39;; git show 59c0013c:core/logx/logs_test.go 2&gt;&amp;1 | rg -n -C 8 &#39;Global|global|merge&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>22</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>23</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go core/logc/logs.go &amp;&amp; \
go test ./core/logx ./core/logc&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>24</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --check &amp;&amp; \
git diff --stat &amp;&amp; \
git status --short &amp;&amp; \
git diff -- core/logx/fields.go core/logx/logs.go core/logx/richlogger.go core/logx/fields_test.go core/logc/logs.go | sed -n &#39;1,520p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>25</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;du -sh .scratch 2&gt;/dev/null; find .scratch -maxdepth 3 -type f | sed -n &#39;1,40p&#39;; git check-ignore -v .scratch .scratch/go .scratch/go/mod 2&gt;/dev/null || \
true; rg -n \&#34;&#34;&#39;^&#39;&#34;\\.scratch|scratch\&#34; .gitignore .git/info/exclude 2&gt;/dev/null || \
true&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>26</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test -race ./core/logx ./core/logc&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>27</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short --untracked-files=all | sed -n &#39;1,120p&#39;; printf &#39;\\n--- new package ---\\n&#39;; sed -n &#39;1,260p&#39; core/logc/logs.go; printf &#39;\\n--- diff check ---\\n&#39;; git diff --check; printf &#39;\\n--- package compile list ---\\n&#39;; go list ./core/logx ./core/logc&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>28</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rm -rf -- .scratch/go &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>29</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rm -rf -- .scratch/go &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>30</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -ld .scratch .scratch/go .scratch/go/mod .scratch/go/mod/github.com/stretchr/testify@v1.8.0 2&gt;&amp;1; find .scratch/go -maxdepth 2 -type f -writable 2&gt;/dev/null | sed -n &#39;1,40p&#39;; git status --short --untracked-files=all 2&gt;/dev/null | tail -n 20&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>31</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;chmod -R u+w -- .scratch/go &amp;&amp; \
rm -rf -- .scratch/go &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>32</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;chmod -R u+w .scratch/go &amp;&amp; \
rm -rf .scratch/go &amp;&amp; \
git status --short --untracked-files=all | sed -n &#39;1,80p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>33</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;len\\(.*Fields|assert\\.Equal.*\\[\\]LogField|Fields\\)|entry\\[|map\\[string\\]interface\\{\\}\&#34; core/logx/*_test.go | sed -n &#39;1,240p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>34</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d core/logx/fields.go core/logx/fields_test.go core/logx/logs.go core/logx/richlogger.go core/logc/logs.go &amp;&amp; \
git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>35</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;nl -ba core/logx/fields.go | sed -n &#39;1,100p&#39;; nl -ba core/logx/logs.go | sed -n &#39;410,460p&#39;; nl -ba core/logx/richlogger.go | sed -n &#39;120,155p&#39;; nl -ba core/logc/logs.go | sed -n &#39;1,190p&#39;; nl -ba core/logx/fields_test.go | sed -n &#39;30,75p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>
