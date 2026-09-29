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

### Target: `codex/gpt-6-luna/medium (repeat 5)`

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
| **Wall-Clock Latency** | 52.76s | 98.95s | <span class="benchmark-delta-negative">+87.5%</span> | 62.11s | 107.41s | <span class="benchmark-delta-negative">+72.9%</span> |
| **Process Start → First Event** | 0.13s | 0.15s | — | 0.14s | 0.15s | — |
| **First Event → First Tool Call** | 5.18s | 6.09s | — | 5.52s | 5.33s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 7 | 15 | <span class="benchmark-delta-negative">+114.3%</span> | 9 | 11 | <span class="benchmark-delta-negative">+22.2%</span> |
| **Initial Load / Discovery Turns** | 4 | 3 | <span class="benchmark-delta-positive">-25.0%</span> | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 7 | 15 | <span class="benchmark-delta-negative">+114.3%</span> | 9 | 11 | <span class="benchmark-delta-negative">+22.2%</span> |
| **Output Tokens** | 1995 | 1992 | <span class="benchmark-delta-positive">-0.2%</span> | 1953 | 1806 | <span class="benchmark-delta-positive">-7.5%</span> |
| **Reasoning / Thinking Tokens** | 324 | 753 | <span class="benchmark-delta-negative">+132.4%</span> | 358 | 996 | <span class="benchmark-delta-negative">+178.2%</span> |
| **Total Input Tokens** | 175051 | 400614 | <span class="benchmark-delta-negative">+128.9%</span> | 247633 | 346482 | <span class="benchmark-delta-negative">+39.9%</span> |
| **Cached Input Tokens** | 156672 | 350464 | <span class="benchmark-delta-positive">+123.7%</span> | 229632 | 316160 | <span class="benchmark-delta-positive">+37.7%</span> |
| **Uncached Input Tokens** | 18379 | 50150 | <span class="benchmark-delta-negative">+172.9%</span> | 18001 | 30322 | <span class="benchmark-delta-negative">+68.4%</span> |
| **Cached vs Uncached Token Ratio** | 8.52:1 | 6.99:1 | <span class="benchmark-delta-negative">-18.0%</span> | 12.76:1 | 10.43:1 | <span class="benchmark-delta-negative">-18.3%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 40 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 40 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; | head -100 &amp;&amp; \
rg -n &#39;type Request|func \\(.*Request.*Normalize|func .*Validate|Request\\.ID|\\.ID&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;README*&#39; | head -60&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
cat go.mod &amp;&amp; \
rg --files -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files | head -100 &amp;&amp; \
cat AGENTS.override.md&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request\\{|\\.Request\\.|\\.ID|invalid request|Normalize\\(&#39; --glob &#39;*.go&#39; . &amp;&amp; \
rg --files&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/identity-model.md &amp;&amp; \
cat request/request.go&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in gateway/handler.go notifications/notifier.go request/request.go request/request_test.go queue/queue.go storage/requests.go scheduler/jobs.go correlation/identities.go reporting/report.go; do printf &#39;\\n--- %s ---\\n&#39; \&#34;&#34;&#39;$f&#34;; cat &#34;$f&#34;; done&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile
changes = {
&#39;request/request.go&#39;: [
(&#39;\&#34;errors\&#34;\\n&#39;, &#39;\&#34;errors\&#34;\\n&#39;),
(&#39;ID      string&#39;, &#39;RequestID string&#39;),
(&#39;if r.ID == \&#34;\&#34; {\\n\\t\\treturn errors.New(\&#34;invalid request\&#34;)\\n\\t}&#39;, &#39;if r.RequestID == \&#34;\&#34; {\\n\\t\\treturn ErrInvalidRequest\\n\\t}&#39;),
(&#39;func Normalize(kind string) string {\\n\\tkind = strings.TrimSpace(kind)\\n\\tkind = strings.ToLower(kind)\\n\\treturn kind\\n}&#39;, &#39;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)\\n\\nfunc Normalize(kind string) string {\\n\\treturn normalizeKind(kind)\\n}\\n\\nfunc normalizeKind(kind string) string {\\n\\tkind = strings.TrimSpace(kind)\\n\\tkind = strings.ToLower(kind)\\n\\treturn kind\\n}&#39;),
(&#39;if r.ID == id {&#39;, &#39;if r.RequestID == id {&#39;),
],
&#39;gateway/handler.go&#39;: [(&#39;r.ID, err&#39;, &#39;r.RequestID, err&#39;)],
&#39;notifications/notifier.go&#39;: [(&#39;r.ID&#39;, &#39;r.RequestID&#39;), (&#39;r.Request.ID&#39;, &#39;r.Request.RequestID&#39;)],
&#39;queue/queue.go&#39;: [(&#39;r.ID&#39;, &#39;r.RequestID&#39;)],
&#39;storage/requests.go&#39;: [(&#39;record.ID == r.ID&#39;, &#39;record.ID == r.RequestID&#39;)],
&#39;scheduler/jobs.go&#39;: [(&#39;left.Request.ID&#39;, &#39;left.Request.RequestID&#39;), (&#39;right.Request.ID&#39;, &#39;right.Request.RequestID&#39;), (&#39;job.Request.ID&#39;, &#39;job.Request.RequestID&#39;), (&#39;r.ID&#39;, &#39;r.RequestID&#39;)],
&#39;correlation/identities.go&#39;: [(&#39;{r.ID, resp.ID, review.ID}&#39;, &#39;{r.RequestID, resp.ID, review.ID}&#39;)],
&#39;reporting/report.go&#39;: [(&#39;r.Request.ID&#39;, &#39;r.Request.RequestID&#39;)],
&#39;gateway/handler_test.go&#39;: [(&#39;request.Request{\&#34;req-1\&#34;, \&#34;audit\&#34;, nil}&#39;, &#39;request.Request{RequestID: \&#34;req-1\&#34;, Kind: \&#34;audit\&#34;}&#39;)],
&#39;request/request_test.go&#39;: [(&#39;import \&#34;testing\&#34;&#39;, &#39;import (\\n\\t\&#34;errors\&#34;\\n\\t\&#34;testing\&#34;\\n)&#39;), (&#39;if err == nil {\\n\\t\\tt.Fatal(\&#34;Validate() accepted an empty request identity\&#34;)\\n\\t}&#39;, &#39;if &#34;&#39;!errors.Is(err, ErrInvalidRequest) {&#39;&#34;\\n\\t\\tt.Fatalf(\&#34;Validate() error = %v, want ErrInvalidRequest\&#34;, err)\\n\\t}&#39;), (&#39;Request{\&#34;req-7\&#34;, \&#34;audit\&#34;, nil}&#39;, &#39;Request{RequestID: \&#34;req-7\&#34;, Kind: \&#34;audit\&#34;}&#39;), (&#39;[]Request{{\&#34;other\&#34;, \&#34;audit\&#34;, nil}, want}&#39;, &#39;[]Request{{RequestID: \&#34;other\&#34;, Kind: \&#34;audit\&#34;}, want}&#39;)],
&#39;storage/requests_test.go&#39;: [(&#39;request.Request{\&#34;req-2\&#34;, \&#34;\&#34;, nil}&#39;, &#39;request.Request{RequestID: \&#34;req-2\&#34;}&#39;)],
}
for name, replacements in changes.items():
    path = Path(name)
    data = path.read_text()
    for old, new in replacements:
        if old not in data:
            raise SystemExit(f&#39;missing expected text in {name}: {old&#34;&#39;!r}&#39;&#34;&#39;)
        data = data.replace(old, new)
    fd, tmp = tempfile.mkstemp(prefix=f&#39;.{path.name}.&#39;, dir=path.parent)
    try:
        with os.fdopen(fd, &#39;w&#39;) as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp, path)
    except:
        try: os.unlink(tmp)
        except FileNotFoundError: pass
        raise
PY
gofmt -w gateway/handler.go gateway/handler_test.go notifications/notifier.go request/request.go request/request_test.go queue/queue.go storage/requests.go storage/requests_test.go scheduler/jobs.go correlation/identities.go reporting/report.go&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;(*Request).Validate&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\.ID|Request\\{[&#34;&#39;^&#39;&#34;\\n]*\&#34;req-|type Request struct|normalizeKind|ErrInvalidRequest&#39; --glob &#39;*.go&#39; . &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git diff --stat &amp;&amp; \
go build ./...&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./... &amp;&amp; \
git status --short &amp;&amp; \
git diff -- request/request.go request/request_test.go gateway/handler.go gateway/handler_test.go notifications/notifier.go queue/queue.go storage/requests.go storage/requests_test.go scheduler/jobs.go correlation/identities.go reporting/report.go&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
        &#34;access_modifier&#34;: &#34;public&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;Request.Validate&#34;,
        &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\t\treturn ErrInvalidRequest\n\t}\n\treturn nil&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;Normalize&#34;,
        &#34;body&#34;: &#34;return normalizeKind(kind)&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
        &#34;access_modifier&#34;: &#34;private&#34;,
        &#34;placement&#34;: &#34;private_start&#34;
      }
    }
  ],
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short -- go.mod go.sum &amp;&amp; \
rg -n &#39;\\b(Request\\s*\\{[&#34;&#39;^}]*&#39;&#34;\\bID:|\\.Request\\.ID|\\bRequest\\s*\\{)&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --stat &amp;&amp; \
git diff -- request/request.go correlation/identities.go gateway/handler.go notifications/notifier.go queue/queue.go reporting/report.go request/request_test.go scheduler/jobs.go storage/requests.go gateway/handler_test.go storage/requests_test.go response review&#39;</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
cat request/request.go &amp;&amp; \
git diff -- request/request.go &amp;&amp; \
git show HEAD:request/request.go | head -50 &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 41 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 40 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
cat request/request.go &amp;&amp; \
rg -n &#39;\\.ID|Request\\{|Normalize|Validate|ErrInvalid&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
rg -n &#39;\\b(ID|Normalize|Validate)\\b|\\.ID\\b|ErrInvalidRequest&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in gateway/handler.go notifications/notifier.go queue/queue.go storage/requests.go scheduler/jobs.go reporting/report.go request/request_test.go; do echo ===$f; cat &#34;$f&#34;; done; rg -n &#39;&#34;&#39;request\\.Request|Request\\b|RequestID&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;for f in gateway/handler.go notifications/notifier.go queue/queue.go storage/requests.go scheduler/jobs.go reporting/report.go request/request_test.go; do printf &#39;%s\\n&#39; \&#34;&#34;&#39;$f&#34;; cat &#34;$f&#34;; done; rg -n &#39;&#34;&#39;request\\.Request|Request\\b|RequestID&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_insert_decl</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\t\treturn ErrInvalidRequest\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\.ID|Request\\{&#39; --glob &#39;*.go&#39; . &amp;&amp; \
gofmt -w request/request.go gateway/handler.go notifications/notifier.go queue/queue.go storage/requests.go scheduler/jobs.go reporting/report.go &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git diff --stat &amp;&amp; \
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;body&#34;: &#34;return normalizeKind(kind)&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w correlation/identities.go &amp;&amp; \
go build ./... &amp;&amp; \
go test ./... &amp;&amp; \
git status --short &amp;&amp; \
git diff -- request/request.go gateway/handler.go notifications/notifier.go queue/queue.go storage/requests.go scheduler/jobs.go reporting/report.go correlation/identities.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
cat correlation/identities.go &amp;&amp; \
rg -n &#39;RequestID|\\.ID\\b|ErrInvalidRequest|invalid request|Normalize|normalizeKind&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --stat &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git diff --&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git ls-files -v request/request.go gateway/handler.go notifications/notifier.go&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was not discovered before I started the edits. The operations were suitable to combine with `semantic_batch`, but I used the individual semantic tools instead.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>
