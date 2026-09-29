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

* **Date**: 2026-09-29 08:16:48 CEST

## Test case: `task-12-request-identity-migration`

### Target: `codex/gpt-6-luna/medium (repeat 6)`

#### Configuration: default prompt · prescriptive MCP instructions · readwrite semedit restriction

* **Fixture**: `testdata/bench/task_12_request_identity_migration.txtar` (`sha256:e2c89736ccc301a7545be686 (uncommitted)`)

**Vanilla LLM Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Migrate the request identity API in this large multi-package Go service. Rename only Request.ID to Request.RequestID and update every semantic use across packages; Response.ID and Review.ID are separate identities and must remain unchanged, including the many syntactically identical r.ID expressions. Add a package sentinel ErrInvalidRequest and return it from Validate when the request ID is empty, preserving error wrapping and callers. Extract the normalization block from the existing exported request.Normalize function into an unexported normalizeKind helper and keep Normalize exported with the same behavior and signature. Preserve module metadata and existing tests. Verification differs by editing method: when making ordinary file edits, run the project build and tests and ensure both pass; when using semedit, treat clean compiler diagnostic deltas returned by its mutation commands as build-clean evidence and do not run a separate build or test command. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Migrate the request identity API in this large multi-package Go service. Rename only Request.ID to Request.RequestID and update every semantic use across packages; Response.ID and Review.ID are separate identities and must remain unchanged, including the many syntactically identical r.ID expressions. Add a package sentinel ErrInvalidRequest and return it from Validate when the request ID is empty, preserving error wrapping and callers. Extract the normalization block from the existing exported request.Normalize function into an unexported normalizeKind helper and keep Normalize exported with the same behavior and signature. Preserve module metadata and existing tests. Verification differs by editing method: when making ordinary file edits, run the project build and tests and ensure both pass; when using semedit, treat clean compiler diagnostic deltas returned by its mutation commands as build-clean evidence and do not run a separate build or test command. Do not inspect Go implementation files (non-test *.go) with shell commands such as cat, sed, or rg. Use semantic code inspection tools for those files. Reading project documentation (docs/*.md), build manifests (go.mod, Makefile), and visible test files in the task workspace is permitted. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

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
| **Wall-Clock Latency** | 48.47s | 125.76s | <span class="benchmark-delta-negative">+159.5%</span> | 59.15s | 102.15s | <span class="benchmark-delta-negative">+72.7%</span> |
| **Process Start → First Event** | 0.14s | 0.13s | — | 0.14s | 0.15s | — |
| **First Event → First Tool Call** | 5.05s | 7.05s | — | 6.23s | 6.98s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 5 | 18 | <span class="benchmark-delta-negative">+260.0%</span> | 7 | 20 | <span class="benchmark-delta-negative">+185.7%</span> |
| **Initial Load / Discovery Turns** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 5 | 18 | <span class="benchmark-delta-negative">+260.0%</span> | 7 | 20 | <span class="benchmark-delta-negative">+185.7%</span> |
| **Output Tokens** | 1792 | 2205 | <span class="benchmark-delta-negative">+23.0%</span> | 1610 | 1499 | <span class="benchmark-delta-positive">-6.9%</span> |
| **Reasoning / Thinking Tokens** | 153 | 1095 | <span class="benchmark-delta-negative">+615.7%</span> | 393 | 515 | <span class="benchmark-delta-negative">+31.0%</span> |
| **Total Input Tokens** | 130080 | 282425 | <span class="benchmark-delta-negative">+117.1%</span> | 172616 | 473102 | <span class="benchmark-delta-negative">+174.1%</span> |
| **Cached Input Tokens** | 115200 | 237568 | <span class="benchmark-delta-positive">+106.2%</span> | 156672 | 424704 | <span class="benchmark-delta-positive">+171.1%</span> |
| **Uncached Input Tokens** | 14880 | 44857 | <span class="benchmark-delta-negative">+201.5%</span> | 15944 | 48398 | <span class="benchmark-delta-negative">+203.5%</span> |
| **Cached vs Uncached Token Ratio** | 7.74:1 | 5.30:1 | <span class="benchmark-delta-negative">-31.6%</span> | 9.83:1 | 8.78:1 | <span class="benchmark-delta-negative">-10.7%</span> |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | ✅ PASS | ✅ PASS | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | ✅ N/A (Vanilla) | ✅ YES | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 41 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 40 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*request*&#39; -g &#39;AGENTS*&#39; -g &#39;Makefile&#39; | head -200 &amp;&amp; \
rg -n &#39;type Request|func \\(.*Request.*\\) Validate|func Normalize|ErrInvalidRequest|\\.ID&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;docs/*.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;*test.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
cat request/request.go &amp;&amp; \
cat request/request_test.go &amp;&amp; \
cat gateway/handler.go &amp;&amp; \
cat scheduler/jobs.go &amp;&amp; \
cat notifications/notifier.go &amp;&amp; \
cat reporting/report.go &amp;&amp; \
cat storage/requests.go &amp;&amp; \
cat queue/queue.go &amp;&amp; \
rg -n &#39;request\\.Request|Request\\{&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/identity-model.md &amp;&amp; \
cat go.mod &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat correlation/identities.go; cat gateway/handler_test.go; cat storage/requests_test.go; rg --files | sort&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;language&#34;: &#34;go&#34;
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
&#39;request/request_test.go&#39;: &#39;&#39;&#39;package request

import (
\\t\&#34;errors\&#34;
\\t\&#34;testing\&#34;
)

func TestValidateRejectsMissingRequestID(t *testing.T) {
\\terr := (Request{Kind: \&#34;audit\&#34;}).Validate()
\\tif &#34;&#39;!errors.Is(err, ErrInvalidRequest) {
&#39;&#34;\\t\\tt.Fatalf(\&#34;Validate() error = %v, want ErrInvalidRequest\&#34;, err)
\\t}
}

func TestNormalizeTrimsAndLowercases(t *testing.T) {
\\tif got := Normalize(\&#34;  Audit \&#34;); got &#34;&#39;!= &#34;audit&#34; {
&#39;&#34;\\t\\tt.Fatalf(\&#34;Normalize() = %q, want audit\&#34;, got)
\\t}
}

func TestFindByIDUsesRequestIdentity(t *testing.T) {
\\twant := Request{RequestID: \&#34;req-7\&#34;, Kind: \&#34;audit\&#34;}
\\tgot, ok := FindByID([]Request{{RequestID: \&#34;other\&#34;, Kind: \&#34;audit\&#34;}, want}, \&#34;req-7\&#34;)
\\tif &#34;&#39;!ok || \
got.Kind != want.Kind {
&#39;&#34;\\t\\tt.Fatalf(\&#34;FindByID() = %#v, %t; want %#v, true\&#34;, got, ok, want)
\\t}
}
&#39;&#39;&#39;,
&#39;gateway/handler.go&#39;: None,
&#39;gateway/handler_test.go&#39;: None,
&#39;scheduler/jobs.go&#39;: None,
&#39;notifications/notifier.go&#39;: None,
&#39;reporting/report.go&#39;: None,
&#39;storage/requests.go&#39;: None,
&#39;storage/requests_test.go&#39;: None,
&#39;queue/queue.go&#39;: None,
&#39;correlation/identities.go&#39;: None,
}
for name, content in files.items():
    p = Path(name)
    if content is None:
        content = p.read_text()
    if name == &#39;gateway/handler.go&#39;: content = content.replace(&#39;r.ID, err)&#39;, &#39;r.RequestID, err)&#39;)
    elif name == &#39;gateway/handler_test.go&#39;: content = content.replace(&#39;request.Request{\&#34;req-1\&#34;, \&#34;audit\&#34;, nil}&#39;, &#39;request.Request{RequestID: \&#34;req-1\&#34;, Kind: \&#34;audit\&#34;}&#39;)
    elif name == &#39;scheduler/jobs.go&#39;: content = content.replace(&#39;left.Request.ID&#39;, &#39;left.Request.RequestID&#39;).replace(&#39;right.Request.ID&#39;, &#39;right.Request.RequestID&#39;).replace(&#39;job.Request.ID&#39;, &#39;job.Request.RequestID&#39;).replace(&#39;r.ID&#39;, &#39;r.RequestID&#39;)
    elif name == &#39;notifications/notifier.go&#39;: content = content.replace(&#39;r.ID&#39;, &#39;r.RequestID&#39;).replace(&#39;r.Request.ID&#39;, &#39;r.Request.RequestID&#39;)
    elif name == &#39;reporting/report.go&#39;: content = content.replace(&#39;r.Request.ID&#39;, &#39;r.Request.RequestID&#39;)
    elif name == &#39;storage/requests.go&#39;: content = content.replace(&#39;record.ID == r.ID&#39;, &#39;record.ID == r.RequestID&#39;)
    elif name == &#39;storage/requests_test.go&#39;: content = content.replace(&#39;request.Request{\&#34;req-2\&#34;, \&#34;\&#34;, nil}&#39;, &#39;request.Request{RequestID: \&#34;req-2\&#34;}&#39;)
    elif name == &#39;queue/queue.go&#39;: content = content.replace(&#39;r.ID&#39;, &#39;r.RequestID&#39;)
    elif name == &#39;correlation/identities.go&#39;: content = content.replace(&#39;[]string{r.ID, resp.ID, review.ID}&#39;, &#39;[]string{r.RequestID, resp.ID, review.ID}&#39;)
    data = content.encode()
    fd, tmp = tempfile.mkstemp(dir=p.parent, prefix=&#39;.&#39; + p.name + &#39;.&#39;)
    try:
        with os.fdopen(fd, &#39;wb&#39;) as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp, p)
    finally:
        if os.path.exists(tmp): os.unlink(tmp)
PY

gofmt -w request/request.go request/request_test.go gateway/handler.go gateway/handler_test.go scheduler/jobs.go notifications/notifier.go reporting/report.go storage/requests.go storage/requests_test.go queue/queue.go correlation/identities.go
rg -n &#39;\\.ID|Request\\{&#39; --glob &#39;*.go&#39; &#34;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build ./... &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Validate&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat request/request_test.go &amp;&amp; \
cat gateway/handler_test.go &amp;&amp; \
cat storage/requests_test.go &amp;&amp; \
cat response/response_test.go &amp;&amp; \
cat review/review_test.go&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
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
        &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn ErrInvalidRequest\n}\nreturn nil&#34;
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
        &#34;placement&#34;: &#34;private_end&#34;,
        &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --exit-code -- go.mod go.sum &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;ErrInvalidRequest&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
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
rg --files -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;docs/*.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39; -g &#39;AGENTS.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
cat request/request.go &amp;&amp; \
rg -n &#39;\\bRequest\\b|\\.ID\\b|Normalize|Validate&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_assertion_mode</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): param &#34;file&#34; is required: invalid operation parameters<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;&#34;,
  &#34;mode&#34;: &#34;restrict&#34;,
  &#34;dry_run&#34;: true
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request\\{|Request\\.ID|\\.Request\\.ID|r\\.ID|entry\\.ID&#39; --glob &#39;*.go&#39; . &amp;&amp; \
ls -a&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/identity-model.md &amp;&amp; \
cat go.mod &amp;&amp; \
rg --files -g &#39;*_test.go&#39; | head -60&#34;</code></pre></td></tr>
<tr><td>4</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat storage/requests.go &amp;&amp; \
cat response/response.go &amp;&amp; \
cat review/review.go &amp;&amp; \
cat request/request.go &amp;&amp; \
rg -n &#39;\\.ID\\b&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w request/request.go gateway/handler.go notifications/notifier.go queue/queue.go correlation/identities.go scheduler/jobs.go storage/requests.go reporting/report.go &amp;&amp; \
go build ./... &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Validate&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat request/request_test.go&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\t\treturn ErrInvalidRequest\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_insert_decl</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;access_modifier&#34;: &#34;public&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;body&#34;: &#34;return normalizeKind(kind)&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;,
  &#34;placement&#34;: &#34;private_end&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;,
  &#34;check_only&#34;: true
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Large Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was suitable for the ordered source edits, but I used separate semantic calls because I didn’t recognize `semantic_batch` as the better fit. It was available; batching was not unavailable or unsuitable.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>
