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

* **Date**: 2026-10-18 16:11:21 CEST

## Test case: `task-12-request-identity-migration`

### Target: `codex/gpt-6-luna/medium (repeat 2)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

* **Fixture**: [testdata/bench/task_12_request_identity_migration.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_12_request_identity_migration.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Migrate the request identity API in this large multi-package Go service. Rename only Request.ID to Request.RequestID and update every semantic use across packages; Response.ID and Review.ID are separate identities and must remain unchanged, including the many syntactically identical r.ID expressions. Add a package sentinel ErrInvalidRequest and return it from Validate when the request ID is empty, preserving error wrapping and callers. Extract the normalization block from the existing exported request.Normalize function into an unexported normalizeKind helper and keep Normalize exported with the same behavior and signature. Preserve module metadata and existing tests. Verification differs by editing method: when making ordinary file edits, run the project build and tests and ensure both pass; when using semedit, treat clean compiler diagnostic deltas returned by its mutation commands as build-clean evidence and do not run a separate build or test command. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

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

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 36.00s | 124.71s | <span class="benchmark-delta-negative">+246.4%</span> | — | — | — |
| **Process Start → First Event** | 0.07s | 0.07s | — | — | — | — |
| **First Event → First Tool Call** | 4.17s | 4.46s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 8 | 27 | <span class="benchmark-delta-negative">+237.5%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 4 | 2 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 27 | <span class="benchmark-delta-negative">+237.5%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.004508 | $0.022401 | +396.9% | — | — | — |
| **Output Tokens** | 1746 | 5636 | <span class="benchmark-delta-negative">+222.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 287 | 3397 | <span class="benchmark-delta-negative">+1083.6%</span> | — | — | — |
| **Total Input Tokens** | 187611 | 1225952 | <span class="benchmark-delta-negative">+553.5%</span> | — | — | — |
| **Cached Input Tokens** | 168064 | 1144576 | <span class="benchmark-delta-positive">+581.0%</span> | — | — | — |
| **Uncached Input Tokens** | 19547 | 81376 | <span class="benchmark-delta-negative">+316.3%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.60:1 | 14.07:1 | <span class="benchmark-delta-positive">+63.6%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 40 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 41 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; | head -120&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; request/request.go&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
cat request/request.go &amp;&amp; \
rg -n &#39;\\.ID|ID:&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
rg -n &#39;Request\\s*\\{&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\bRequest\\.ID\\b|\\.ID\\b|type Request\\b|func (Validate|Normalize)\\b&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request|Normalize|invalid request|FindByID|\\.ID&#39; --glob &#39;*.go&#39; &amp;&amp; \
ls -la&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files docs &amp;&amp; \
cat docs/*&#39;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/profilerecord compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w request/request.go gateway/handler.go storage/requests.go notifications/notifier.go queue/queue.go correlation/identities.go scheduler/jobs.go reporting/report.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: go version go1.27.1 darwin/arm64 /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64 /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/pkg/tool/darwin_arm64 local gateway/handler.go:23: if resp.ID == &#34;&#34; { review/review.go:10: return r.ID review/review.go:15: if r.ID == id { correlation/identities.go:10: return []string{r.RequestID, resp.ID, review.ID} response/response.go:10: return r.ID respons<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go version &amp;&amp; \
go env GOROOT GOTOOLDIR GOTOOLCHAIN &amp;&amp; \
rg -n &#39;\\.ID|ID:&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff --&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;nl -ba request/request.go&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: gateway/handler.go:17: return response.Response{}, fmt.Errorf(&#34;submit %s: %w&#34;, r.RequestID, err) gateway/handler.go:23: if resp.ID == &#34;&#34; { reporting/report.go:13: grouped[r.Request.RequestID] = append(grouped[r.Request.RequestID], r) storage/requests.go:12: if record.ID == r.RequestID { queue/queue.go:12: if entry.Key == r.RequestID { queue/queue.go:22: keys = append(keys, r.RequestID) request/request.go:14:var ErrInvalidRequest = errors.New(&#34;invalid request&#34;) request/request.go:17: if r.RequestID == &#34;&#34; { r<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request\\{|\\.RequestID|\\.ID|ErrInvalidRequest|normalizeKind&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
rg -n &#39;\\bID\\b&#39; request --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Request&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;source&#34;: &#34;func (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn fmt.Errorf(\&#34;%w: request ID is empty\&#34;, ErrInvalidRequest)\n\t}\n\treturn nil\n}&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;nl -ba request/request.go&#39;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 24,
  &#34;end_line&#34;: 26,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(input string) string {\n\tkind := strings.TrimSpace(input)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;nl -ba request/request.go&#39;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 24,
  &#34;end_line&#34;: 25,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(input string) (normalized string) {\n\tvalue := strings.TrimSpace(input)\n\tnormalized = strings.ToLower(value)\n\treturn\n}&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract from Normalize: named result variables may be affected by the moved statements&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 24,
  &#34;end_line&#34;: 25,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(input string) (result string) {\n\tvalue := strings.TrimSpace(input)\n\tvalue = strings.ToLower(value)\n\treturn value\n}&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract from Normalize: named result variables may be affected by the moved statements&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 24,
  &#34;end_line&#34;: 25,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Normalize&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;organize_imports&#34;: true,
  &#34;trust_workspace&#34;: true
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,90p&#39; request/request.go&#34;</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;decl&#34;,
  &#34;source&#34;: &#34;import (\n\t\&#34;errors\&#34;\n\t\&#34;fmt\&#34;\n\t\&#34;strings\&#34;\n)&#34;,
  &#34;placement&#34;: &#34;file_start&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,35p&#39; request/request.go&#34;</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;organize_imports&#34;: true,
  &#34;trust_workspace&#34;: true
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>27</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;check_only&#34;: true,
  &#34;trust_workspace&#34;: true
}</code></pre></td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

* **Fixture**: [testdata/bench/task_12_request_identity_migration.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_12_request_identity_migration.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> Migrate the request identity API in this large multi-package Go service. Rename only Request.ID to Request.RequestID and update every semantic use across packages; Response.ID and Review.ID are separate identities and must remain unchanged, including the many syntactically identical r.ID expressions. Add a package sentinel ErrInvalidRequest and return it from Validate when the request ID is empty, preserving error wrapping and callers. Extract the normalization block from the existing exported request.Normalize function into an unexported normalizeKind helper and keep Normalize exported with the same behavior and signature. Preserve module metadata and existing tests. Verification differs by editing method: when making ordinary file edits, run the project build and tests and ensure both pass; when using semedit, treat clean compiler diagnostic deltas returned by its mutation commands as build-clean evidence and do not run a separate build or test command. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
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
| **Wall-Clock Latency** | 36.00s | 54.05s | <span class="benchmark-delta-negative">+50.2%</span> | — | — | — |
| **Process Start → First Event** | 0.07s | 0.16s | — | — | — | — |
| **First Event → First Tool Call** | 4.17s | 4.64s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 8 | 13 | <span class="benchmark-delta-negative">+62.5%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 4 | 1 | <span class="benchmark-delta-positive">-75.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 13 | <span class="benchmark-delta-negative">+62.5%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.004508 | $0.011236 | +149.2% | — | — | — |
| **Output Tokens** | 1746 | 2518 | <span class="benchmark-delta-negative">+44.2%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 287 | 779 | <span class="benchmark-delta-negative">+171.4%</span> | — | — | — |
| **Total Input Tokens** | 187611 | 661941 | <span class="benchmark-delta-negative">+252.8%</span> | — | — | — |
| **Cached Input Tokens** | 168064 | 624640 | <span class="benchmark-delta-positive">+271.7%</span> | — | — | — |
| **Uncached Input Tokens** | 19547 | 37301 | <span class="benchmark-delta-negative">+90.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.60:1 | 16.75:1 | <span class="benchmark-delta-positive">+94.8%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 40 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 41 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; | head -120&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -200 &amp;&amp; \
rg -n \&#34;func Normalize|type Request struct|func .*Validate|Request\\.ID|\\.ID\&#34; --glob &#39;*.go&#39; . | head -240&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat go.mod &amp;&amp; \
cat request/request.go &amp;&amp; \
rg -n &#39;\\.ID|ID:&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
rg -n &#39;Request\\s*\\{&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: ./AGENTS.override.md module example.com/identityservice go 1.23 package request import ( &#34;errors&#34; &#34;strings&#34; ) type Request struct { ID string Kind string Payload []byte } func (r Request) Validate() error { if r.ID == &#34;&#34; { return errors.New(&#34;invalid request&#34;) } return nil } func Normalize(kind string) string { kind = strings.TrimSpace(kind) kind = strings.ToLower(kind) return kind } func FindByID(requests []Request, id string) (Request, bool) { for _, r := range requests { if r.ID == id { return r, true } }<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.override.md&#39; -g &#39;.semedit*&#39; -g &#39;*semedit*&#39; . &amp;&amp; \
cat go.mod &amp;&amp; \
cat request/request.go &amp;&amp; \
cat .codex/instructions.md&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request|Normalize|invalid request|FindByID|\\.ID&#39; --glob &#39;*.go&#39; &amp;&amp; \
ls -la&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # Benchmark fixture instruction override This directory is a synthetic benchmark fixture and is the complete benchmark workspace. Follow the benchmark user prompt. You may inspect and use any file in this directory, including documentation. Do not inspect, read, or use paths outside this directory. Run only checks that the user prompt or files in this directory require. Do not modify this AGENTS.override.md file. ./gateway/handler_test.go:18: resp, err := (Handler{client: client}).Submit(request.Request{&#34;re<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md &amp;&amp; \
rg -n \&#34;semedit|normalizeKind|Request\\{\&#34; . --glob &#39;&#34;&#39;!*.sum&#39;&#34;&#39; --glob &#39;&#34;&#39;!go.mod&#39;&#34;&#39; &amp;&amp; \
git status --short &amp;&amp; \
rg --files&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files docs &amp;&amp; \
cat docs/*&#39;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792331598779116000&#34;,
  &#34;include_tests&#34;: false
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;request&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792331598779116000&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/profilerecord compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w request/request.go gateway/handler.go storage/requests.go notifications/notifier.go queue/queue.go correlation/identities.go scheduler/jobs.go reporting/report.go &amp;&amp; \
go test ./...&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792331598779116000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: go version go1.27.1 darwin/arm64 /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64 /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/pkg/tool/darwin_arm64 local gateway/handler.go:23: if resp.ID == &#34;&#34; { review/review.go:10: return r.ID review/review.go:15: if r.ID == id { correlation/identities.go:10: return []string{r.RequestID, resp.ID, review.ID} response/response.go:10: return r.ID respons<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go version &amp;&amp; \
go env GOROOT GOTOOLDIR GOTOOLCHAIN &amp;&amp; \
rg -n &#39;\\.ID|ID:&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
git diff --check &amp;&amp; \
git status --short &amp;&amp; \
git diff --&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;\\.ID|RequestID|func Normalize|invalid request\&#34; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: gateway/handler.go:17: return response.Response{}, fmt.Errorf(&#34;submit %s: %w&#34;, r.RequestID, err) gateway/handler.go:23: if resp.ID == &#34;&#34; { reporting/report.go:13: grouped[r.Request.RequestID] = append(grouped[r.Request.RequestID], r) storage/requests.go:12: if record.ID == r.RequestID { queue/queue.go:12: if entry.Key == r.RequestID { queue/queue.go:22: keys = append(keys, r.RequestID) request/request.go:14:var ErrInvalidRequest = errors.New(&#34;invalid request&#34;) request/request.go:17: if r.RequestID == &#34;&#34; { r<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request\\{|\\.RequestID|\\.ID|ErrInvalidRequest|normalizeKind&#39; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; &amp;&amp; \
rg -n &#39;\\bID\\b&#39; request --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;start_line&#34;: 22,
  &#34;end_line&#34;: 24,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792331598779116000&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;placement&#34;: &#34;private_end&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792331598779116000&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792331598779116000&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;placement&#34;: &#34;public_start&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792331598779116000&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;source&#34;: &#34;func (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)\n\t}\n\treturn nil\n}&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792331598779116000&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: package request import ( &#34;errors&#34; &#34;fmt&#34; &#34;strings&#34; ) var ErrInvalidRequest = errors.New(&#34;invalid request&#34;) type Request struct { RequestID string Kind string Payload []byte } func (r Request) Validate() error { if r.RequestID == &#34;&#34; { return fmt.Errorf(&#34;request ID is empty: %w&#34;, ErrInvalidRequest) } return nil } func Normalize(kind string) string { return normalizeKind(kind) } func FindByID(requests []Request, id string) (Request, bool) { for _, r := range requests { if r.RequestID == id { return r, true } } <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
rg -n \&#34;\\.ID|RequestID\&#34; --glob &#39;*.go&#39; --glob &#39;&#34;&#39;!**/*_test.go&#39;&#34;&#39; . &amp;&amp; \
rg -n \&#34;&#34;&#39;^module|&#39;&#39;^go &#34; go.mod &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">I discovered `semantic_batch`, but I did not use it because I treated the refactor as separate steps that needed their own diagnostic results. That was not a limitation of the operations; they could have been batched.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit gave me a real confidence boost by renaming `Request.ID` across packages while preserving the separate response and review identities, and I’d ask my owner to add its tools.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

