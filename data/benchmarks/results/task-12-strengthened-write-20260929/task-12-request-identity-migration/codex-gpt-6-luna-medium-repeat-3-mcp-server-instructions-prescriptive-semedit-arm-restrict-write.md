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

* **Date**: 2026-09-29 07:45:33 CEST

## Test case: `task-12-request-identity-migration`

### Target: `codex/gpt-6-luna/medium (repeat 3)`

#### Configuration: default prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_12_request_identity_migration.txtar` (`sha256:e2c89736ccc301a7545be686 (uncommitted)`)

**Vanilla LLM Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Migrate the request identity API in this large multi-package Go service. Rename only Request.ID to Request.RequestID and update every semantic use across packages; Response.ID and Review.ID are separate identities and must remain unchanged, including the many syntactically identical r.ID expressions. Add a package sentinel ErrInvalidRequest and return it from Validate when the request ID is empty, preserving error wrapping and callers. Extract the normalization block from the existing exported request.Normalize function into an unexported normalizeKind helper and keep Normalize exported with the same behavior and signature. Preserve module metadata and existing tests. Verification differs by editing method: when making ordinary file edits, run the project build and tests and ensure both pass; when using semedit, treat clean compiler diagnostic deltas returned by its mutation commands as build-clean evidence and do not run a separate build or test command. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Migrate the request identity API in this large multi-package Go service. Rename only Request.ID to Request.RequestID and update every semantic use across packages; Response.ID and Review.ID are separate identities and must remain unchanged, including the many syntactically identical r.ID expressions. Add a package sentinel ErrInvalidRequest and return it from Validate when the request ID is empty, preserving error wrapping and callers. Extract the normalization block from the existing exported request.Normalize function into an unexported normalizeKind helper and keep Normalize exported with the same behavior and signature. Preserve module metadata and existing tests. Verification differs by editing method: when making ordinary file edits, run the project build and tests and ensure both pass; when using semedit, treat clean compiler diagnostic deltas returned by its mutation commands as build-clean evidence and do not run a separate build or test command. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package request

import (
	"errors"
	"strings"
)

type Request struct {
	ID      string
	Kind    string
	Payload []byte
}

func (r Request) Validate() error {
	if r.ID == "" {
		return errors.New("invalid request")
	}
	return nil
}

func Normalize(kind string) string {
	kind = strings.TrimSpace(kind)
	kind = strings.ToLower(kind)
	return kind
}

func FindByID(requests []Request, id string) (Request, bool) {
	for _, r := range requests {
		if r.ID == id {
			return r, true
		}
	}
	return Request{}, false
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 42.74s | 85.19s | <span class="benchmark-delta-negative">+99.3%</span> | 38.77s | 120.90s | <span class="benchmark-delta-negative">+211.9%</span> |
| **Process Start → First Event** | 0.13s | 0.15s | — | 0.14s | 0.15s | — |
| **First Event → First Tool Call** | 5.63s | 6.45s | — | 4.42s | 6.88s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 5 | 13 | <span class="benchmark-delta-negative">+160.0%</span> | 5 | 17 | <span class="benchmark-delta-negative">+240.0%</span> |
| **Initial Load / Discovery Turns** | 3 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 5 | 13 | <span class="benchmark-delta-negative">+160.0%</span> | 5 | 17 | <span class="benchmark-delta-negative">+240.0%</span> |
| **Output Tokens** | 1620 | 759 | <span class="benchmark-delta-positive">-53.1%</span> | 1353 | 2369 | <span class="benchmark-delta-negative">+75.1%</span> |
| **Reasoning / Thinking Tokens** | 96 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | 332 | 970 | <span class="benchmark-delta-negative">+192.2%</span> |
| **Total Input Tokens** | 129732 | 459970 | <span class="benchmark-delta-negative">+254.6%</span> | 127599 | 406282 | <span class="benchmark-delta-negative">+218.4%</span> |
| **Cached Input Tokens** | 109056 | 413696 | <span class="benchmark-delta-positive">+279.3%</span> | 113152 | 359680 | <span class="benchmark-delta-positive">+217.9%</span> |
| **Uncached Input Tokens** | 20676 | 46274 | <span class="benchmark-delta-negative">+123.8%</span> | 14447 | 46602 | <span class="benchmark-delta-negative">+222.6%</span> |
| **Cached vs Uncached Token Ratio** | 5.27:1 | 8.94:1 | <span class="benchmark-delta-positive">+69.5%</span> | 7.83:1 | 7.72:1 | <span class="benchmark-delta-negative">-1.5%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 41 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 41 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*request*&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; | head -100 &amp;&amp; \
rg -n &#39;type Request|func \\(.*Request.*\\) Validate|func Normalize|\\.ID\\b|ErrInvalidRequest|Normalize\\(&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat PROJECT_BRIEF.md 2&gt;/dev/null; cat request/request.go; cat gateway/handler.go; cat notifications/notifier.go; cat queue/queue.go; cat storage/requests.go; cat scheduler/jobs.go; cat correlation/identities.go; cat reporting/report.go; cat go.mod; cat request/request_test.go&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request&#34;,
  &#34;include_tests&#34;: true
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\bRequest\\s*\\{|request\\.Request|\\.ID\\b|FindByID|Validate\\(&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile
files = {
&#39;request/request.go&#39;: &#39;&#39;&#39;package request

import (
\\t\&#34;errors\&#34;
\\t\&#34;fmt\&#34;
\\t\&#34;strings\&#34;
)

var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)

type Request struct {
\\tRequestID string
\\tKind      string
\\tPayload   []byte
}

func (r Request) Validate() error {
\\tif r.RequestID == \&#34;\&#34; {
\\t\\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)
\\t}
\\treturn nil
}

func Normalize(kind string) string {
\\treturn normalizeKind(kind)
}

func normalizeKind(kind string) string {
\\tkind = strings.TrimSpace(kind)
\\tkind = strings.ToLower(kind)
\\treturn kind
}

func FindByID(requests []Request, id string) (Request, bool) {
\\tfor _, r := range requests {
\\t\\tif r.RequestID == id {
\\t\\t\\treturn r, true
\\t\\t}
\\t}
\\treturn Request{}, false
}
&#39;&#39;&#39;,
&#39;gateway/handler.go&#39;: None,
&#39;notifications/notifier.go&#39;: None,
&#39;queue/queue.go&#39;: None,
&#39;storage/requests.go&#39;: None,
&#39;scheduler/jobs.go&#39;: None,
&#39;correlation/identities.go&#39;: None,
&#39;reporting/report.go&#39;: None,
&#39;gateway/handler_test.go&#39;: None,
&#39;request/request_test.go&#39;: None,
&#39;storage/requests_test.go&#39;: None,
}
for name in files:
    if name &#34;&#39;!= &#39;&#34;&#39;request/request.go&#39;:
        p=Path(name); old=p.read_text(); new=old
        if name == &#39;gateway/handler.go&#39;: new=new.replace(&#39;r.ID, err)&#39;, &#39;r.RequestID, err)&#39;)
        elif name == &#39;notifications/notifier.go&#39;: new=new.replace(&#39;r.ID&#39;, &#39;r.RequestID&#39;).replace(&#39;r.Request.ID&#39;,&#39;r.Request.RequestID&#39;)
        elif name == &#39;queue/queue.go&#39;: new=new.replace(&#39;r.ID&#39;,&#39;r.RequestID&#39;)
        elif name == &#39;storage/requests.go&#39;: new=new.replace(&#39;record.ID == r.ID&#39;,&#39;record.ID == r.RequestID&#39;)
        elif name == &#39;scheduler/jobs.go&#39;: new=new.replace(&#39;left.Request.ID&#39;,&#39;left.Request.RequestID&#39;).replace(&#39;right.Request.ID&#39;,&#39;right.Request.RequestID&#39;).replace(&#39;job.Request.ID&#39;,&#39;job.Request.RequestID&#39;).replace(&#39;r.ID&#39;,&#39;r.RequestID&#39;)
        elif name == &#39;correlation/identities.go&#39;: new=new.replace(&#39;r.ID, resp.ID&#39;,&#39;r.RequestID, resp.ID&#39;)
        elif name == &#39;reporting/report.go&#39;: new=new.replace(&#39;r.Request.ID&#39;,&#39;r.Request.RequestID&#39;)
        elif name == &#39;gateway/handler_test.go&#39;: new=new.replace(&#39;request.Request{\&#34;req-1\&#34;, \&#34;audit\&#34;, nil}&#39;, &#39;request.Request{RequestID: \&#34;req-1\&#34;, Kind: \&#34;audit\&#34;}&#39;)
        elif name == &#39;request/request_test.go&#39;: new=new.replace(&#39;Request{\&#34;req-7\&#34;, \&#34;audit\&#34;, nil}&#39;, &#39;Request{RequestID: \&#34;req-7\&#34;, Kind: \&#34;audit\&#34;}&#39;).replace(&#39;Request{{\&#34;other\&#34;, \&#34;audit\&#34;, nil}, want}&#39;, &#39;Request{{RequestID: \&#34;other\&#34;, Kind: \&#34;audit\&#34;}, want}&#39;)
        elif name == &#39;storage/requests_test.go&#39;: new=new.replace(&#39;request.Request{\&#34;req-2\&#34;, \&#34;\&#34;, nil}&#39;, &#39;request.Request{RequestID: \&#34;req-2\&#34;}&#39;)
        files[name]=new
for name, content in files.items():
    p=Path(name)
    if content is None: continue
    fd,tmp=tempfile.mkstemp(dir=p.parent, prefix=&#39;.&#39;+p.name+&#39;.&#39;)
    try:
        with os.fdopen(fd,&#39;w&#39;) as f:
            f.write(content); f.flush(); os.fsync(f.fileno())
        os.replace(tmp,p)
    finally:
        if os.path.exists(tmp): os.unlink(tmp)
PY
gofmt -w request/request.go gateway/handler.go notifications/notifier.go queue/queue.go storage/requests.go scheduler/jobs.go correlation/identities.go reporting/report.go gateway/handler_test.go request/request_test.go storage/requests_test.go&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Validate&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;r\\.ID|Request\\.ID|Request\\{\&#34;|Request\\{\\{&#39; --glob &#39;*.go&#39; . &amp;&amp; \
go build ./... &amp;&amp; \
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_insert_decl</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;group&#34;: &#34;append&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\t\treturn fmt.Errorf(\&#34;request ID: %w\&#34;, ErrInvalidRequest)\n\t}\n\treturn nil&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;body&#34;: &#34;return normalizeKind(kind)&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;include_tests&#34;: true
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 40 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 40 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; | head -200 &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -160 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
rg -n &#39;\\.ID|\\bID\\b|Normalize|Validate&#39; --glob &#39;*.go&#39; &amp;&amp; \
rg --files -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;README.md&#39; | head -80 &amp;&amp; \
cat go.mod &amp;&amp; \
rg -n &#39;func \\(.*Request.*\\) Validate|func Normalize|type Request|\\.ID\\b|Request\\s*\\{&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat gateway/handler.go; cat request/request_test.go; cat queue/queue.go notifications/notifier.go correlation/identities.go scheduler/jobs.go reporting/report.go storage/requests.go; cat Makefile 2&gt;/dev/null; cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile
root=Path(&#39;.&#39;)
changes={
&#39;request/request.go&#39;: [(&#39;ID      string&#39;,&#39;RequestID string&#39;),(&#39;if r.ID == \&#34;\&#34; {\\n\\t\\treturn errors.New(\&#34;invalid request\&#34;)&#39;,&#39;if r.RequestID == \&#34;\&#34; {\\n\\t\\treturn ErrInvalidRequest&#39;),(&#39;func (r Request) Validate() error {&#39;,&#39;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)\\n\\nfunc (r Request) Validate() error {&#39;),(&#39;func Normalize(kind string) string {\\n\\tkind = strings.TrimSpace(kind)\\n\\tkind = strings.ToLower(kind)\\n\\treturn kind\\n}&#39;,&#39;func Normalize(kind string) string {\\n\\treturn normalizeKind(kind)\\n}\\n\\nfunc normalizeKind(kind string) string {\\n\\tkind = strings.TrimSpace(kind)\\n\\tkind = strings.ToLower(kind)\\n\\treturn kind\\n}&#39;),(&#39;r.ID == id&#39;,&#39;r.RequestID == id&#39;)],
&#39;gateway/handler.go&#39;: [(&#39;r.ID, err&#39;,&#39;r.RequestID, err&#39;)],
&#39;queue/queue.go&#39;: [(&#39;r.ID&#39;,&#39;r.RequestID&#39;)],
&#39;notifications/notifier.go&#39;: [(&#39;r.ID&#39;,&#39;r.RequestID&#39;),(&#39;r.Request.ID&#39;,&#39;r.Request.RequestID&#39;)],
&#39;correlation/identities.go&#39;: [(&#39;r.ID, resp.ID&#39;,&#39;r.RequestID, resp.ID&#39;)],
&#39;scheduler/jobs.go&#39;: [(&#39;left.Request.ID&#39;,&#39;left.Request.RequestID&#39;),(&#39;right.Request.ID&#39;,&#39;right.Request.RequestID&#39;),(&#39;job.Request.ID&#39;,&#39;job.Request.RequestID&#39;),(&#39;r.ID&#39;,&#39;r.RequestID&#39;)],
&#39;reporting/report.go&#39;: [(&#39;r.Request.ID&#39;,&#39;r.Request.RequestID&#39;)],
&#39;storage/requests.go&#39;: [(&#39;r.ID&#39;,&#39;r.RequestID&#39;)],
}
for name,repls in changes.items():
 p=root/name
 text=p.read_text()
 for old,new in repls:
  if old not in text: raise SystemExit(f&#39;missing {old&#34;&#39;!r} in {name}&#39;&#34;&#39;)
  text=text.replace(old,new)
 fd,tmp=tempfile.mkstemp(dir=p.parent,prefix=p.name+&#39;.&#39;)
 try:
  with os.fdopen(fd,&#39;w&#39;) as f:
   f.write(text); f.flush(); os.fsync(f.fileno())
  os.replace(tmp,p)
 finally:
  if os.path.exists(tmp): os.unlink(tmp)
PY
&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.Validate&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\.ID|\\bID\\s+string|ErrInvalidRequest|normalizeKind&#39; --glob &#39;*.go&#39;; git diff -- request/request.go gateway/handler.go queue/queue.go notifications/notifier.go correlation/identities.go scheduler/jobs.go reporting/report.go storage/requests.go; go build ./... &amp;&amp; \
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat request/request.go &amp;&amp; \
cat request/request_test.go&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_insert_decl</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\t\treturn ErrInvalidRequest\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;body&#34;: &#34;return normalizeKind(kind)&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- request/request.go request/request_test.go gateway/handler.go gateway/handler_test.go notifications/notifier.go queue/queue.go reporting/report.go scheduler/jobs.go storage/requests.go storage/requests_test.go correlation/identities.go go.mod go.sum &amp;&amp; \
git status --short -- request/request.go request/request_test.go gateway/handler.go gateway/handler_test.go notifications/notifier.go queue/queue.go reporting/report.go scheduler/jobs.go storage/requests.go storage/requests_test.go correlation/identities.go go.mod go.sum &amp;&amp; \
rg -n &#39;\\bRequest\\s*\\{\\s*ID\\b|Request\\.ID|\\.Request\\.ID|r\\.ID&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.Validate&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
rg -n &#39;\\.ID\\b|RequestID|ErrInvalidRequest|normalizeKind&#39; --glob &#39;*.go&#39; . &amp;&amp; \
git diff --check&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I used separate semantic MCP calls because I treated each operation as a dependent step, checking the result and diagnostics before proceeding to the next. `semantic_batch` had been discovered and was suitable for sequencing edits, so batching was possible; I should have used it for the dependent mutations.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I discovered `semantic_batch` before editing, but did not use it because I chose to run each operation separately to get immediate diagnostic feedback between dependent edits. Batching was suitable and available; that was my workflow choice.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>
