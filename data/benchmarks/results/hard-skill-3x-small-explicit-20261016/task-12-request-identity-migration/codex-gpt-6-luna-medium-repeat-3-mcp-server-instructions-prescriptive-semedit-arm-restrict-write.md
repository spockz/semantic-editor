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

* **Date**: 2026-10-17 00:16:09 CEST

## Test case: `task-12-request-identity-migration`

### Target: `codex/gpt-6-luna/medium (repeat 3)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,provider_transport=codex_app_server_v2`

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
| **Wall-Clock Latency** | 23.37s | 76.37s | <span class="benchmark-delta-negative">+226.8%</span> | — | — | — |
| **Process Start → First Event** | 0.02s | 0.03s | — | — | — | — |
| **First Event → First Tool Call** | 3.75s | 9.95s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 7 | 28 | <span class="benchmark-delta-negative">+300.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 28 | <span class="benchmark-delta-negative">+300.0%</span> | — | — | — |
| **Output Tokens** | 2047 | 4029 | <span class="benchmark-delta-negative">+96.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 301 | 1598 | <span class="benchmark-delta-negative">+430.9%</span> | — | — | — |
| **Total Input Tokens** | 179803 | 1054053 | <span class="benchmark-delta-negative">+486.2%</span> | — | — | — |
| **Cached Input Tokens** | 160256 | 1006336 | <span class="benchmark-delta-positive">+528.0%</span> | — | — | — |
| **Uncached Input Tokens** | 19547 | 47717 | <span class="benchmark-delta-negative">+144.1%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.20:1 | 21.09:1 | <span class="benchmark-delta-positive">+157.2%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 40 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 41 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!*vendor*&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; | head -80 &amp;&amp; \
git status --short &amp;&amp; \
rg -n &#39;type Request|func \\(.*Request.*\\) (Validate|Normalize)|Request\\.ID|\\.ID|func Normalize|ErrInvalidRequest&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
cat request/request_test.go &amp;&amp; \
cat docs/identity-model.md &amp;&amp; \
cat go.mod &amp;&amp; \
rg -n &#39;request\\.Request|Request\\{|\\.Request\\.ID|r\\.ID|Request\\.ID&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Validate&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: ./gateway/handler_test.go:13: return response.Response{ID: &#34;resp-1&#34;, RequestID: &#34;req-1&#34;, Status: 202}, nil ./gateway/handler_test.go:18: resp, err := (Handler{client: client}).Submit(request.Request{RequestID: &#34;req-1&#34;, Kind: &#34;audit&#34;}) ./gateway/handler_test.go:22: if client.got.Kind != &#34;audit&#34; || resp.ID != &#34;resp-1&#34; || resp.RequestID != &#34;req-1&#34; { ./gateway/handler.go:17: return response.Response{}, fmt.Errorf(&#34;submit %s: %w&#34;, r.RequestID, err) ./gateway/handler.go:23: if resp.ID == &#34;&#34; { ./notifications/noti<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request\\.ID|r\\.ID|RequestID|\\.ID&#39; --glob &#39;*.go&#39; . &amp;&amp; \
gofmt -w request/request.go request/request_test.go gateway/handler.go gateway/handler_test.go notifications/notifier.go queue/queue.go storage/requests.go storage/requests_test.go scheduler/jobs.go correlation/identities.go reporting/report.go &amp;&amp; \
go build ./...&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...; git diff -- request/request.go request/request_test.go gateway/handler.go gateway/handler_test.go notifications/notifier.go queue/queue.go storage/requests.go storage/requests_test.go scheduler/jobs.go correlation/identities.go reporting/report.go; git status --short -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 0 params: unknown parameter \&#34;declaration\&#34; for \&#34;semantic_insert_decl\&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, file, group, placement<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;declaration&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;Request.Validate&#34;,
        &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn fmt.Errorf(\&#34;%w: RequestID is empty\&#34;, ErrInvalidRequest)\n}\nreturn nil&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;SUBOPTIMAL_TOOLS|request/request.go&#39;&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-12-request-identity-migration_1792187438182941000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/2d\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
        &#34;placement&#34;: &#34;private_start&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;calculate declaration offset: section placement violation: strategy \&#34;private_start\&#34;, in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-12-request-identity-migration_1792187438182941000/request/request.go&#34;,&#34;location&#34;:{&#34;range&#34;:{&#34;end&#34;:{&#34;character&#34;:0,&#34;line&#34;:0},&#34;start&#34;:{&#34;character&#34;:0,&#34;line&#34;:0}},&#34;uri&#34;:&#34;file:///Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;placement&#34;: &#34;private_start&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;placement&#34;: &#34;file_start&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;source&#34;: &#34;func (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn fmt.Errorf(\&#34;%w: RequestID is empty\&#34;, ErrInvalidRequest)\n\t}\n\treturn nil\n}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;line range 23-25 does not select a contiguous top-level statement region in Normalize&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 23,
  &#34;end_line&#34;: 25,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 24,
  &#34;end_line&#34;: 26,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;placement&#34;: &#34;file_end&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-explicit_small-explicit_task-12-request-identity-migration_1792187438182941000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/2d\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_verify</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;ErrInvalidRequest&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,provider_transport=codex_app_server_v2`

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
| **Wall-Clock Latency** | 23.37s | 50.97s | <span class="benchmark-delta-negative">+118.1%</span> | — | — | — |
| **Process Start → First Event** | 0.02s | 0.03s | — | — | — | — |
| **First Event → First Tool Call** | 3.75s | 3.36s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 7 | 17 | <span class="benchmark-delta-negative">+142.9%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 17 | <span class="benchmark-delta-negative">+142.9%</span> | — | — | — |
| **Output Tokens** | 2047 | 3085 | <span class="benchmark-delta-negative">+50.7%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 301 | 1583 | <span class="benchmark-delta-negative">+425.9%</span> | — | — | — |
| **Total Input Tokens** | 179803 | 711827 | <span class="benchmark-delta-negative">+295.9%</span> | — | — | — |
| **Cached Input Tokens** | 160256 | 671744 | <span class="benchmark-delta-positive">+319.2%</span> | — | — | — |
| **Uncached Input Tokens** | 19547 | 40083 | <span class="benchmark-delta-negative">+105.1%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.20:1 | 16.76:1 | <span class="benchmark-delta-positive">+104.4%</span> | — | — | — |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;&#34;&#39;!*vendor*&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; | head -80 &amp;&amp; \
git status --short &amp;&amp; \
rg -n &#39;type Request|func \\(.*Request.*\\) (Validate|Normalize)|Request\\.ID|\\.ID|func Normalize|ErrInvalidRequest&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; -g &#39;*.go&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
cat request/request_test.go &amp;&amp; \
cat docs/identity-model.md &amp;&amp; \
cat go.mod &amp;&amp; \
rg -n &#39;request\\.Request|Request\\{|\\.Request\\.ID|r\\.ID|Request\\.ID&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
rg -n &#39;\\.ID|\\bID\\b|Normalize|Validate&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>4</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;request.Validate&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;request.Request&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: ./gateway/handler_test.go:13: return response.Response{ID: &#34;resp-1&#34;, RequestID: &#34;req-1&#34;, Status: 202}, nil ./gateway/handler_test.go:18: resp, err := (Handler{client: client}).Submit(request.Request{RequestID: &#34;req-1&#34;, Kind: &#34;audit&#34;}) ./gateway/handler_test.go:22: if client.got.Kind != &#34;audit&#34; || resp.ID != &#34;resp-1&#34; || resp.RequestID != &#34;req-1&#34; { ./gateway/handler.go:17: return response.Response{}, fmt.Errorf(&#34;submit %s: %w&#34;, r.RequestID, err) ./gateway/handler.go:23: if resp.ID == &#34;&#34; { ./notifications/noti<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request\\.ID|r\\.ID|RequestID|\\.ID&#39; --glob &#39;*.go&#39; . &amp;&amp; \
gofmt -w request/request.go request/request_test.go gateway/handler.go gateway/handler_test.go notifications/notifier.go queue/queue.go storage/requests.go storage/requests_test.go scheduler/jobs.go correlation/identities.go reporting/report.go &amp;&amp; \
go build ./...&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...; git diff -- request/request.go request/request_test.go gateway/handler.go gateway/handler_test.go notifications/notifier.go queue/queue.go storage/requests.go storage/requests_test.go scheduler/jobs.go correlation/identities.go reporting/report.go; git status --short -- go.mod go.sum&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
rg -n &#39;\\.ID|Request\\{&#39; --glob &#39;*.go&#39; .&#34;</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;placement&#34;: &#34;private_start&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;calculate declaration offset: section placement violation: strategy \&#34;private_start\&#34;, in /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792187516465852000/request/request.go&#34;,&#34;location&#34;:{&#34;range&#34;:{&#34;end&#34;:{&#34;character&#34;:0,&#34;line&#34;:0},&#34;start&#34;:{&#34;character&#34;:0,&#34;line&#34;:0}},&#34;uri&#34;:&#34;file:///Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit_small-im<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;placement&#34;: &#34;private_start&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;SUBOPTIMAL_TOOLS|README|PROJECT_BRIEF|AGENTS&#39; &amp;&amp; \
cat request/request.go&#34;</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;placement&#34;: &#34;before_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Request&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;method&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;source&#34;: &#34;func (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn ErrInvalidRequest\n\t}\n\treturn nil\n}&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;nl -ba request/request.go&#39;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 24,
  &#34;end_line&#34;: 26,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;kind&#34;: &#34;function&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;placement&#34;: &#34;private_end&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\.ID|RequestID|ErrInvalidRequest|normalizeKind|Normalize&#39; --glob &#39;*.go&#39; . &amp;&amp; \
git status --short &amp;&amp; \
git diff -- go.mod go.sum&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was unsuitable for these operations: the field rename, sentinel insertion, and method replacement were sequential, and the helper extraction needed the updated source state. The extraction tool then refused the normalization block, so I inserted the helper and replaced `Normalize` in separate steps.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit’s tools: the semantic rename safely distinguished `Request.ID` from other IDs across packages, though the normalization extraction refusal made the workflow a little less smooth.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

