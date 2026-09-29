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

### Target: `codex/gpt-6-luna/medium (repeat 10)`

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
| **Wall-Clock Latency** | 44.54s | 111.37s | <span class="benchmark-delta-negative">+150.1%</span> | 70.44s | 114.91s | <span class="benchmark-delta-negative">+63.1%</span> |
| **Process Start → First Event** | 0.14s | 0.13s | — | 0.14s | 0.14s | — |
| **First Event → First Tool Call** | 5.82s | 4.68s | — | 6.00s | 4.61s | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | 1 | 1 | 0% |
| **Internal Tool Cycles** | 7 | 16 | <span class="benchmark-delta-negative">+128.6%</span> | 9 | 17 | <span class="benchmark-delta-negative">+88.9%</span> |
| **Initial Load / Discovery Turns** | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> | 3 | 3 | 0% |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | 0 | 0 | 0% |
| **Total Tool Invocations** | 7 | 16 | <span class="benchmark-delta-negative">+128.6%</span> | 9 | 17 | <span class="benchmark-delta-negative">+88.9%</span> |
| **Output Tokens** | 1537 | 1831 | <span class="benchmark-delta-negative">+19.1%</span> | 2119 | 3210 | <span class="benchmark-delta-negative">+51.5%</span> |
| **Reasoning / Thinking Tokens** | 349 | 617 | <span class="benchmark-delta-negative">+76.8%</span> | 471 | 1703 | <span class="benchmark-delta-negative">+261.6%</span> |
| **Total Input Tokens** | 175096 | 454017 | <span class="benchmark-delta-negative">+159.3%</span> | 227720 | 584226 | <span class="benchmark-delta-negative">+156.6%</span> |
| **Cached Input Tokens** | 158720 | 416768 | <span class="benchmark-delta-positive">+162.6%</span> | 204288 | 533504 | <span class="benchmark-delta-positive">+161.2%</span> |
| **Uncached Input Tokens** | 16376 | 37249 | <span class="benchmark-delta-negative">+127.5%</span> | 23432 | 50722 | <span class="benchmark-delta-negative">+116.5%</span> |
| **Cached vs Uncached Token Ratio** | 9.69:1 | 11.19:1 | <span class="benchmark-delta-positive">+15.4%</span> | 8.72:1 | 10.52:1 | <span class="benchmark-delta-positive">+20.6%</span> |
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
rg -n \&#34;type Request struct|func \\(.*Request.*\\) Validate|func Normalize|\\.ID\\b|Request\\{.*ID|ErrInvalidRequest\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;tools/benchmark-harness/README.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; | head -160&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; &amp;&amp; \
cat request/request.go &amp;&amp; \
cat go.mod &amp;&amp; \
rg -n \&#34;\\bRequest\\b|request\\.\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat request/request_test.go &amp;&amp; \
cat gateway/handler.go &amp;&amp; \
cat docs/identity-model.md &amp;&amp; \
&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat PROJECT_BRIEF.md 2&gt;/dev/null; git status --short&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
files = {
&#39;request/request.go&#39;: [(&#39;ID      string&#39;,&#39;RequestID string&#39;), (&#39;r.ID&#39;,&#39;r.RequestID&#39;)],
&#39;gateway/handler.go&#39;: [(&#39;r.ID&#39;,&#39;r.RequestID&#39;)],
&#39;notifications/notifier.go&#39;: [(&#39;r.Request.ID&#39;,&#39;r.Request.RequestID&#39;), (&#39;r.ID&#39;,&#39;r.RequestID&#39;)],
&#39;queue/queue.go&#39;: [(&#39;r.ID&#39;,&#39;r.RequestID&#39;)],
&#39;storage/requests.go&#39;: [(&#39;r.ID&#39;,&#39;r.RequestID&#39;)],
&#39;scheduler/jobs.go&#39;: [(&#39;left.Request.ID&#39;,&#39;left.Request.RequestID&#39;), (&#39;right.Request.ID&#39;,&#39;right.Request.RequestID&#39;), (&#39;job.Request.ID&#39;,&#39;job.Request.RequestID&#39;), (&#39;r.ID&#39;,&#39;r.RequestID&#39;)],
&#39;reporting/report.go&#39;: [(&#39;r.Request.ID&#39;,&#39;r.Request.RequestID&#39;)],
}
for name, replacements in files.items():
    p=Path(name)
    data=p.read_text()
    for old,new in replacements:
        data=data.replace(old,new)
    p.write_text(data)

p=Path(&#39;request/request.go&#39;)
s=p.read_text()
s=s.replace(&#39;func (r Request) Validate() error {\\n\\tif r.RequestID == \&#34;\&#34; {\\n\\t\\treturn errors.New(\&#34;invalid request\&#34;)\\n&#39;, &#39;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)\\n\\nfunc (r Request) Validate() error {\\n\\tif r.RequestID == \&#34;\&#34; {\\n\\t\\treturn fmt.Errorf(\&#34;invalid request: %w\&#34;, ErrInvalidRequest)\\n&#39;)
s=s.replace(&#39;\&#34;errors\&#34;\\n\\t\&#34;strings\&#34;&#39;, &#39;\&#34;errors\&#34;\\n\\t\&#34;fmt\&#34;\\n\\t\&#34;strings\&#34;&#39;)
s=s.replace(&#39;func Normalize(kind string) string {\\n\\tkind = strings.TrimSpace(kind)\\n\\tkind = strings.ToLower(kind)\\n\\treturn kind\\n}&#39;, &#39;func Normalize(kind string) string {\\n\\treturn normalizeKind(kind)\\n}\\n\\nfunc normalizeKind(kind string) string {\\n\\tkind = strings.TrimSpace(kind)\\n\\tkind = strings.ToLower(kind)\\n\\treturn kind\\n}&#39;)
p.write_text(s)
PY
rg -n \&#34;\\.ID\\b|Request\\{.*ID\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat request/request.go &amp;&amp; \
cat gateway/handler.go &amp;&amp; \
cat notifications/notifier.go &amp;&amp; \
cat queue/queue.go &amp;&amp; \
cat storage/requests.go &amp;&amp; \
cat scheduler/jobs.go &amp;&amp; \
cat reporting/report.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w request/request.go gateway/handler.go notifications/notifier.go queue/queue.go storage/requests.go scheduler/jobs.go reporting/report.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
p=Path(&#39;correlation/identities.go&#39;)
s=p.read_text().replace(&#39;return []string{r.ID, resp.ID, review.ID}&#39;, &#39;return []string{r.RequestID, resp.ID, review.ID}&#39;)
p.write_text(s)
PY
gofmt -w correlation/identities.go
go build ./... &amp;&amp; \
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_insert_decl</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\t\treturn fmt.Errorf(\&#34;%w: request ID is empty\&#34;, ErrInvalidRequest)\n\t}\n\treturn nil&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;add&#34;: [
    &#34;fmt&#34;
  ]
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): unknown param &#34;declaration&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, file, placement, source, target, target_symbol]: invalid operation parameters<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;,
  &#34;declaration&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_insert_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;access_modifier&#34;: &#34;private&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_body</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;body&#34;: &#34;return normalizeKind(kind)&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
rg -n &#39;Request\\s*\\{|\\.RequestID|\\.ID&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
</tbody>
</table>

##### Standard vs Semedit in Large Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 40 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 41 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; | head -80 &amp;&amp; \
rg -n &#39;\\bRequest\\b|\\.ID\\b|func .*Normalize|Validate\\(&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;tools/benchmark-harness/README.md&#39; -g &#39;docs/adr/README.md&#39; -g &#39;docs/research/README.md&#39; | head -40&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
cat request/request_test.go &amp;&amp; \
rg -n &#39;\\.ID|Request\\{&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
rg --files | head -80 &amp;&amp; \
rg -n &#39;type Request struct|func \\(.*Request.*Validate|func Normalize|Request\\{&#39; --glob &#39;*.go&#39; | head -100&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files | sort &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
cat docs/identity-model.md &amp;&amp; \
cat go.mod &amp;&amp; \
rg -n &#39;\\.ID|Request\\{|RequestRecord|Normalize\\(&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>4</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\.ID|Request\\{&#39; --glob &#39;*.go&#39; . &amp;&amp; \
rg -n &#39;RequestID|Request\\.ID|r\\.ID&#39; --glob &#39;*.go&#39; . &amp;&amp; \
gofmt -w request/request.go request/request_test.go gateway/handler.go gateway/handler_test.go notifications/notifier.go queue/queue.go storage/requests.go storage/requests_test.go scheduler/jobs.go correlation/identities.go reporting/report.go &amp;&amp; \
go test ./... &amp;&amp; \
go build ./...&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git diff --check &amp;&amp; \
git diff --stat &amp;&amp; \
rg -n &#39;Request\\.ID|request\\.Request\\{[&#34;&#39;^&#39;&#34;\\n]*,|\\.ID&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.sum&#39;&#34;&#39; .&#34;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel &amp;&amp; \
git check-ignore -v request/request.go || \
true &amp;&amp; \
git diff -- request/request.go request/request_test.go gateway/handler.go&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
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
        &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\t\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)\n\t}\n\treturn nil&#34;,
        &#34;auto_organize_imports&#34;: true
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
        &#34;access_modifier&#34;: &#34;private&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;SUBOPTIMAL_TOOLS\\.md&#34;&#39;$|&#39;&#34;\\.scratch&#39; | head -30&#34;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;add&#34;: [
    &#34;fmt&#34;
  ]
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff --check &amp;&amp; \
git diff -- request/request.go gateway/handler.go notifications/notifier.go queue/queue.go reporting/report.go scheduler/jobs.go correlation/identities.go storage/requests.go &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
cat request/request.go&#39;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git rev-parse --show-toplevel &amp;&amp; \
git diff --stat &amp;&amp; \
git check-ignore -v request/request.go || \
true &amp;&amp; \
git status --short --untracked-files=no&#39;</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- go.mod go.sum; rg -n &#39;Request\\.ID|Request\\.RequestID|resp\\.ID|review\\.ID&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I did not use `semantic_batch` because I treated the rename, declaration insertion, body replacements, and import update as separate operations and did not check whether batching supported them. Batching was not discovered.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>
