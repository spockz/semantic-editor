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

* **Date**: 2026-10-17 21:43:41 CEST

## Test case: `task-12-request-identity-migration`

### Target: `codex/gpt-6-luna/high (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

* **Fixture**: [testdata/bench/task_12_request_identity_migration.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_12_request_identity_migration.txtar)

**Vanilla LLM Prompt**:
> You are forbidden to modify protected files: "go.mod", "go.sum".
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
| **Wall-Clock Latency** | 55.91s | 86.26s | <span class="benchmark-delta-negative">+54.3%</span> | — | — | — |
| **Process Start → First Event** | 0.07s | 0.13s | — | — | — | — |
| **First Event → First Tool Call** | 4.28s | 4.58s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 13 | 24 | <span class="benchmark-delta-negative">+84.6%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 2 | <span class="benchmark-delta-positive">-33.3%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 13 | 24 | <span class="benchmark-delta-negative">+84.6%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.008155 | $0.019808 | +142.9% | — | — | — |
| **Output Tokens** | 5745 | 8314 | <span class="benchmark-delta-negative">+44.7%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 3473 | 6027 | <span class="benchmark-delta-negative">+73.5%</span> | — | — | — |
| **Total Input Tokens** | 356724 | 1067513 | <span class="benchmark-delta-negative">+199.3%</span> | — | — | — |
| **Cached Input Tokens** | 337664 | 1012224 | <span class="benchmark-delta-positive">+199.8%</span> | — | — | — |
| **Uncached Input Tokens** | 19060 | 55289 | <span class="benchmark-delta-negative">+190.1%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 17.72:1 | 18.31:1 | <span class="benchmark-delta-positive">+3.3%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 40 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 40 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.work&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;type Request|func \\(.*Request.*\\) (Validate|Normalize)|Request\\.ID|\\.ID\\b|func .*Normalize|func .*Validate\&#34; request&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.override.md&#39; &amp;&amp; \
rg -n &#39;\\bRequest\\b|\\.ID\\b|\\.RequestID\\b|ErrInvalidRequest|func \\(.*Validate|func Normalize|func .*Normalize&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;request.go&#39; -g &#39;AGENTS.md&#39; -g &#39;AGENTS.override.md&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat request/request.go &amp;&amp; \
cat request/request_test.go &amp;&amp; \
cat gateway/handler.go &amp;&amp; \
cat scheduler/jobs.go &amp;&amp; \
cat notifications/notifier.go &amp;&amp; \
cat queue/queue.go &amp;&amp; \
cat correlation/identities.go &amp;&amp; \
cat storage/requests.go &amp;&amp; \
cat reporting/report.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat request/request.go &amp;&amp; \
cat request/request_test.go&#39;</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;include_unexported&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: reporting/report.go:6: Request request.Request reporting/report.go:13: grouped[r.Request.RequestID] = append(grouped[r.Request.RequestID], r) gateway/handler_test.go:9:type testClient struct { got request.Request } gateway/handler_test.go:11:func (c *testClient) Send(r request.Request) (response.Response, error) { gateway/handler_test.go:18: resp, err := (Handler{client: client}).Submit(request.Request{&#34;req-1&#34;, &#34;audit&#34;, nil}) gateway/handler_test.go:22: if client.got.Kind != &#34;audit&#34; || resp.ID != &#34;resp-1&#34; |<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w request/request.go request/request_test.go gateway/handler.go scheduler/jobs.go notifications/notifier.go queue/queue.go correlation/identities.go storage/requests.go reporting/report.go &amp;&amp; \
rg -n &#39;\\bRequest\\b|\\.ID\\b|\\.RequestID\\b|ErrInvalidRequest|Normalize|normalizeKind&#39; --glob &#39;*.go&#39; &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git status --short &amp;&amp; \
git diff --&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git diff -- request/request.go request/request_test.go gateway/handler.go scheduler/jobs.go notifications/notifier.go queue/queue.go correlation/identities.go storage/requests.go reporting/report.go &amp;&amp; \
git diff --quiet -- go.mod; printf &#39;go.mod unchanged: %s\\n&#39; \&#34;&#34;&#39;$?&#34;&#39;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # math/bits compile: version &#34;go1.27.2&#34; does not match go too<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build ./...&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; d<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;calculate declaration offset: section placement violation: strategy \&#34;private_start\&#34;, in /Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-12-request-identity-migration_1792259240122739000/request/request.go&#34;,&#34;location&#34;:{&#34;range&#34;:{&#34;end&#34;:{&#34;character&#34;:0,&#34;line&#34;:0},&#34;start&#34;:{&#34;character&#34;:0,&#34;line&#34;:0}},&#34;uri&#34;:&#34;file:///Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-12-request-identity-migration_1792259240<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;placement&#34;: &#34;private_start&#34;,
  &#34;group&#34;: &#34;standalone&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/profilerecord compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not mat<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;mkdir -p .scratch/go-build &amp;&amp; \
GOCACHE=&#34;$PWD/.scratch/go-build&#34; go build ./...&#39;</code></pre></td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;group&#34;: &#34;standalone&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\.ID\\b|\\.RequestID\\b|RequestID|ErrInvalidRequest|Normalize|normalizeKind&#39; --glob &#39;*.go&#39; &amp;&amp; \
gofmt -d request/request.go request/request_test.go gateway/handler.go scheduler/jobs.go notifications/notifier.go queue/queue.go correlation/identities.go storage/requests.go reporting/report.go&#34;</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;source&#34;: &#34;func (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn fmt.Errorf(\&#34;%w: request ID is empty\&#34;, ErrInvalidRequest)\n\t}\n\treturn nil\n}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GOCACHE=&#34;$PWD/.scratch/go-build&#34; go clean -cache &amp;&amp; \
rmdir .scratch/go-build &amp;&amp; \
rmdir .scratch 2&gt;/dev/null || \
true&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat -n request/request.go&#39;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;find .scratch/go-build -maxdepth 2 -print | head -30 &amp;&amp; \
go version&#39;</code></pre></td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;start_line&#34;: 22,
  &#34;end_line&#34;: 24
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\n\treturn strings.ToLower(strings.TrimSpace(kind))\n}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: selected statement on line 22 is not a call or supported local declaration&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;start_line&#34;: 22,
  &#34;end_line&#34;: 22
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\n\tnormalized := kind\n\tnormalized = strings.TrimSpace(normalized)\n\tnormalized = strings.ToLower(normalized)\n\treturn normalized\n}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat -n request/request.go&#39;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;start_line&#34;: 22,
  &#34;end_line&#34;: 25
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;group&#34;: &#34;standalone&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;source&#34;: &#34;func (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn errors.Join(ErrInvalidRequest)\n\t}\n\treturn nil\n}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: warning: Not a git repository. Use --no-index to compare two paths outside a working tree usage: git diff --no-index [&lt;options&gt;] &lt;path&gt; &lt;path&gt; Diff output format options -p, --patch generate patch -s, --no-patch suppress diff output -u generate patch -U, --unified[=&lt;n&gt;] generate diffs with &lt;n&gt; lines context -W, --[no-]function-context generate diffs with &lt;n&gt; lines context --raw generate the diff in raw format --patch-with-raw synonym for &#39;-p --raw&#39; --patch-with-stat synonym for &#39;-p --stat&#39; --numstat machine<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- go.mod go.sum request/request.go request/request_test.go &amp;&amp; \
cat request/request.go&#39;</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;organize_imports&#34;: true
}</code></pre></td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

* **Fixture**: [testdata/bench/task_12_request_identity_migration.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_12_request_identity_migration.txtar)

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
| **Wall-Clock Latency** | 55.91s | 65.02s | <span class="benchmark-delta-negative">+16.3%</span> | — | — | — |
| **Process Start → First Event** | 0.07s | 0.07s | — | — | — | — |
| **First Event → First Tool Call** | 4.28s | 8.05s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 13 | 16 | <span class="benchmark-delta-negative">+23.1%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 13 | 16 | <span class="benchmark-delta-negative">+23.1%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.008155 | $0.010850 | +33.0% | — | — | — |
| **Output Tokens** | 5745 | 4969 | <span class="benchmark-delta-positive">-13.5%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 3473 | 3019 | <span class="benchmark-delta-positive">-13.1%</span> | — | — | — |
| **Total Input Tokens** | 356724 | 483051 | <span class="benchmark-delta-negative">+35.4%</span> | — | — | — |
| **Cached Input Tokens** | 337664 | 443776 | <span class="benchmark-delta-positive">+31.4%</span> | — | — | — |
| **Uncached Input Tokens** | 19060 | 39275 | <span class="benchmark-delta-negative">+106.1%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 17.72:1 | 11.30:1 | <span class="benchmark-delta-negative">-36.2%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 40 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 40 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.work&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | head -200&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;include_tests&#34;: true
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.override.md&#39; &amp;&amp; \
rg -n &#39;\\bRequest\\b|\\.ID\\b|\\.RequestID\\b|ErrInvalidRequest|func \\(.*Validate|func Normalize|func .*Normalize&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;AGENTS.md&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat request/request.go &amp;&amp; \
cat request/request_test.go &amp;&amp; \
cat gateway/handler.go &amp;&amp; \
cat scheduler/jobs.go &amp;&amp; \
cat notifications/notifier.go &amp;&amp; \
cat queue/queue.go &amp;&amp; \
cat correlation/identities.go &amp;&amp; \
cat storage/requests.go &amp;&amp; \
cat reporting/report.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;request&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;request&#34;,
  &#34;include_unexported&#34;: true,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Validate&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: reporting/report.go:6: Request request.Request reporting/report.go:13: grouped[r.Request.RequestID] = append(grouped[r.Request.RequestID], r) gateway/handler_test.go:9:type testClient struct { got request.Request } gateway/handler_test.go:11:func (c *testClient) Send(r request.Request) (response.Response, error) { gateway/handler_test.go:18: resp, err := (Handler{client: client}).Submit(request.Request{&#34;req-1&#34;, &#34;audit&#34;, nil}) gateway/handler_test.go:22: if client.got.Kind != &#34;audit&#34; || resp.ID != &#34;resp-1&#34; |<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w request/request.go request/request_test.go gateway/handler.go scheduler/jobs.go notifications/notifier.go queue/queue.go correlation/identities.go storage/requests.go reporting/report.go &amp;&amp; \
rg -n &#39;\\bRequest\\b|\\.ID\\b|\\.RequestID\\b|ErrInvalidRequest|Normalize|normalizeKind&#39; --glob &#39;*.go&#39; &amp;&amp; \
git diff -- go.mod go.sum &amp;&amp; \
git status --short &amp;&amp; \
git diff --&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git diff -- request/request.go request/request_test.go gateway/handler.go scheduler/jobs.go notifications/notifier.go queue/queue.go correlation/identities.go storage/requests.go reporting/report.go &amp;&amp; \
git diff --quiet -- go.mod; printf &#39;go.mod unchanged: %s\\n&#39; \&#34;&#34;&#39;$?&#34;&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # math/bits compile: version &#34;go1.27.2&#34; does not match go too<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go build ./...&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;auto_organize_imports&#34;: true,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; d<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;language&#34;: &#34;go&#34;,
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;kind&#34;: &#34;var&#34;,
        &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
        &#34;placement&#34;: &#34;public_start&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;language&#34;: &#34;go&#34;,
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;symbol&#34;: &#34;Request.Validate&#34;,
        &#34;source&#34;: &#34;func (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn ErrInvalidRequest\n\t}\n\treturn nil\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;language&#34;: &#34;go&#34;,
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
        &#34;placement&#34;: &#34;file_end&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;language&#34;: &#34;go&#34;,
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;symbol&#34;: &#34;Normalize&#34;,
        &#34;source&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/profilerecord compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not mat<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;mkdir -p .scratch/go-build &amp;&amp; \
GOCACHE=&#34;$PWD/.scratch/go-build&#34; go build ./...&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\.ID\\b|\\.RequestID\\b|RequestID|ErrInvalidRequest|Normalize|normalizeKind&#39; --glob &#39;*.go&#39; &amp;&amp; \
gofmt -d request/request.go request/request_test.go gateway/handler.go scheduler/jobs.go notifications/notifier.go queue/queue.go correlation/identities.go storage/requests.go reporting/report.go&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff --stat &amp;&amp; \
git diff -- go.mod go.sum&#39;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;GOCACHE=&#34;$PWD/.scratch/go-build&#34; go clean -cache &amp;&amp; \
rmdir .scratch/go-build &amp;&amp; \
rmdir .scratch 2&gt;/dev/null || \
true&#39;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.RequestID&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;find .scratch/go-build -maxdepth 2 -print | head -30 &amp;&amp; \
go version&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Review.ID&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Response.ID&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792259171446438000&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Semedit’s semantic rename and structured edit tools made the request migration feel precise and reassuring, and I’d ask my owner to add them for coding work like this.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

