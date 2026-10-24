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

* **Date**: 2026-10-18 06:58:35 CEST

## Test case: `task-12-request-identity-migration`

### Target: `codex/gpt-6.1-sol/low (repeat 1)`

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
| **Wall-Clock Latency** | 35.21s | 50.23s | <span class="benchmark-delta-negative">+42.6%</span> | — | — | — |
| **Process Start → First Event** | 0.06s | 0.07s | — | — | — | — |
| **First Event → First Tool Call** | 4.42s | 4.02s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 8 | 18 | <span class="benchmark-delta-negative">+125.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 18 | <span class="benchmark-delta-negative">+125.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.100396 | $0.183665 | +82.9% | — | — | — |
| **Output Tokens** | 1501 | 1508 | <span class="benchmark-delta-negative">+0.5%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 164 | 189 | <span class="benchmark-delta-negative">+15.2%</span> | — | — | — |
| **Total Input Tokens** | 221688 | 632587 | <span class="benchmark-delta-negative">+185.4%</span> | — | — | — |
| **Cached Input Tokens** | 188416 | 577152 | <span class="benchmark-delta-positive">+206.3%</span> | — | — | — |
| **Uncached Input Tokens** | 33272 | 55435 | <span class="benchmark-delta-negative">+66.6%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.66:1 | 10.41:1 | <span class="benchmark-delta-positive">+83.9%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 41 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 41 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;*.go&#39; -g &#39;*.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/identity-model.md go.mod; rg -n &#39;&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{}</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: AGENTS.override.md 2026-10-18 04:02:45.548 xcodebuild[3910:25035173] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 04:02:45.645 xcodebuild[3910:25035171] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files --hidden -g &#39;&#34;&#39;! .git&#39;&#34;&#39; -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;*.yml&#39; -g &#39;*.yaml&#39;; git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;*.md&#39; -g &#39;&#34;&#39;!AGENTS.override.md&#39;&#34;&#39;&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
files = [&#39;request/request.go&#39;, &#39;queue/queue.go&#39;, &#39;scheduler/jobs.go&#39;, &#39;correlation/identities.go&#39;, &#39;gateway/handler.go&#39;, &#39;notifications/notifier.go&#39;, &#39;reporting/report.go&#39;, &#39;storage/requests.go&#39;]
for name in files:
    p = Path(name)
    s = p.read_text()
    s = s.replace(&#39;r.ID&#39;, &#39;r.RequestID&#39;).replace(&#39;.Request.ID&#39;, &#39;.Request.RequestID&#39;)
    if name == &#39;request/request.go&#39;:
        s = s.replace(&#39;ID      string&#39;, &#39;RequestID string&#39;)
        s = s.replace(&#39;type Request struct {&#39;, &#39;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)\\n\\ntype Request struct {&#39;)
        s = s.replace(&#39;return errors.New(\&#34;invalid request\&#34;)&#39;, &#39;return ErrInvalidRequest&#39;)
        s = s.replace(&#39;func Normalize(kind string) string {&#39;, &#39;func Normalize(kind string) string {\\n\\treturn normalizeKind(kind)\\n}\\n\\nfunc normalizeKind(kind string) string {&#39;)
    p.write_text(s)
PY

gofmt -w request/request.go queue/queue.go scheduler/jobs.go correlation/identities.go gateway/handler.go notifications/notifier.go reporting/report.go storage/requests.go&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go; rg -n &#39;\\.ID|RequestID&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Validate&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/profilerecord compile: version &#34;g<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
comments = {
&#39;request/request.go&#39;: &#39;Request identity validation and kind normalization belong at the public request boundary.&#39;,
&#39;queue/queue.go&#39;: &#39;Queue keys correlate entries with request identities.&#39;,
&#39;scheduler/jobs.go&#39;: &#39;Job matching uses the identity of the enclosed request.&#39;,
&#39;correlation/identities.go&#39;: &#39;Correlation preserves the separate request, response, and review identity domains.&#39;,
&#39;gateway/handler.go&#39;: &#39;The gateway validates requests before forwarding them to the client.&#39;,
&#39;notifications/notifier.go&#39;: &#39;Notification routing follows the identity of the associated request.&#39;,
&#39;reporting/report.go&#39;: &#39;Reports group rows by their associated request identity.&#39;,
&#39;storage/requests.go&#39;: &#39;Storage lookup compares record keys with incoming request identities.&#39;,
}
for name, comment in comments.items():
    p = Path(name)
    p.write_text(&#39;// &#39; + comment + &#39;\\n&#39; + p.read_text())
Path(&#39;gateway/identity_migration_test.go&#39;).write_text(&#39;&#39;&#39;// These checks preserve sentinel matching across the gateway error boundary.
package gateway

import (
    \&#34;errors\&#34;
    \&#34;testing\&#34;

    \&#34;example.com/identityservice/request\&#34;
)

func TestSubmitWrapsInvalidRequest(t *testing.T) {
    _, err := (Handler{}).Submit(request.Request{})
    if &#34;&#39;!errors.Is(err, request.ErrInvalidRequest) {
        t.Fatalf(&#34;Submit() error = %v, want ErrInvalidRequest&#34;, err)
    }
    if err == request.ErrInvalidRequest {
        t.Fatal(&#34;Submit() lost its error context&#34;)
    }
    if err := (request.Request{RequestID: &#34;req-1&#34;}).Validate(); err != nil {
        t.Fatalf(&#34;Validate() rejected a request identity: %v&#34;, err)
    }
}
&#39;&#34;&#39;&#39;&#39;)
PY
gofmt -w request/request.go queue/queue.go scheduler/jobs.go correlation/identities.go gateway/handler.go gateway/identity_migration_test.go notifications/notifier.go reporting/report.go storage/requests.go
go build ./...
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go version; go env GOROOT GOTOOLDIR GOCACHE; printenv GOROOT GOTOOLDIR GOTOOLCHAIN; rg --files --hidden .scratch | head -40&#39;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;env -u GOROOT go build ./... &amp;&amp; \
env -u GOROOT go test ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/identity-model.md&#39;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;auto_organize_imports&#34;: false
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 0 params: unknown parameter \&#34;symbol\&#34; for \&#34;semantic_insert_decl\&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, file, group, placement, sou<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: false,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;ErrInvalidRequest&#34;,
        &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;Request.Validate&#34;,
        &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n return fmt.Errorf(\&#34;%w: RequestID is empty\&#34;, ErrInvalidRequest)\n}\nreturn nil&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Verification returned 3 diagnostics. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;available&#34;,&#34;report_id&#34;:&#34;4VTW2L54FXI3R3TFUXUQEVEJ4P&#34;,&#34;arguments&#34;:{&#34;category&#34;:&#34;active&#34;,&#34;limit&#34;:50,&#34;offset&#34;:0,&#34;report_id&#34;:&#34;4VTW2L54FXI3R3TFUXUQEVEJ4P&#34;},&#34;coverage&#34;:[{&#34;status&#34;:&#34;verified&#34;,&#34;build_exclusions&#34;:[&#34;Go build context and build tags&#34;,&#34;testdata&#34;,&#34;build&#34;,&#34;dist&#34;,&#34;target&#34;,&#34;vendor&#34;,&#34;node_modules&#34;,&#34;.scratch&#34;,&#34;.git&#34;,&#34;bazel-*&#34;,&#34;_*&#34;,&#34;.*&#34;],&#34;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: false,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;Request.Validate&#34;,
        &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n return fmt.Errorf(\&#34;%w: RequestID is empty\&#34;, ErrInvalidRequest)\n}\nreturn nil&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: true,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;Request.Validate&#34;,
        &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n return fmt.Errorf(\&#34;%w: RequestID is empty\&#34;, ErrInvalidRequest)\n}\nreturn nil&#34;,
        &#34;auto_organize_imports&#34;: true
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;line range 23-24 does not select a contiguous top-level statement region in Normalize&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;start_line&#34;: 23,
  &#34;end_line&#34;: 24
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat -n request/request.go&#39;</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;,&#34;tool&#34;:&#34;semantic_extract_function&#34;}],&#34;workspace_root&#34;:&#34;/Volumes/semedit-bench-ram/run_codex_<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_extract_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;in_function&#34;: &#34;Normalize&#34;,
        &#34;symbol&#34;: &#34;normalizeKind&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;start_line&#34;: 24,
        &#34;end_line&#34;: 25
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;source&#34;: &#34;func normalizeKind(kind string) string {\nkind = strings.TrimSpace(kind)\nkind = strings.ToLower(kind)\nreturn kind\n}&#34;
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
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: 2026-10-18 04:04:40.164 xcodebuild[5823:25051115] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 04:04:40.259 xcodebuild[5823:25051114] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. warning: Not a git repository. Use --no-index to compare two paths outside a working tree usage: git diff --no-index [&lt;options&gt;] &lt;path&gt; &lt;path&gt; [&lt;pathspec&gt;...] Diff outpu<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --stat
 git diff -- go.mod go.sum &#39;*_test.go&#39; response review
 git diff -- request/request.go&#34;</code></pre></td></tr>
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
| **Wall-Clock Latency** | 35.21s | 39.92s | <span class="benchmark-delta-negative">+13.4%</span> | — | — | — |
| **Process Start → First Event** | 0.06s | 0.16s | — | — | — | — |
| **First Event → First Tool Call** | 4.42s | 5.09s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 8 | 17 | <span class="benchmark-delta-negative">+112.5%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 17 | <span class="benchmark-delta-negative">+112.5%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.100396 | $0.107976 | +7.6% | — | — | — |
| **Output Tokens** | 1501 | 1082 | <span class="benchmark-delta-positive">-27.9%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 164 | 56 | <span class="benchmark-delta-positive">-65.9%</span> | — | — | — |
| **Total Input Tokens** | 221688 | 350754 | <span class="benchmark-delta-negative">+58.2%</span> | — | — | — |
| **Cached Input Tokens** | 188416 | 318080 | <span class="benchmark-delta-positive">+68.8%</span> | — | — | — |
| **Uncached Input Tokens** | 33272 | 32674 | <span class="benchmark-delta-positive">-1.8%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 5.66:1 | 9.73:1 | <span class="benchmark-delta-positive">+71.9%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 41 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 40 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;*.go&#39; -g &#39;*.md&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;README*&#39; -g &#39;Makefile&#39; -g &#39;SKILL.md&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat docs/identity-model.md go.mod; rg -n &#39;&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: AGENTS.override.md 2026-10-18 04:02:45.548 xcodebuild[3910:25035173] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 04:02:45.645 xcodebuild[3910:25035171] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files --hidden -g &#39;&#34;&#39;! .git&#39;&#34;&#39; -g &#39;AGENTS*&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;Makefile&#39; -g &#39;*.yml&#39; -g &#39;*.yaml&#39;; git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: cat: .agents/skills/semedit/references/golang.md: No such file or directory AGENTS.override.md 2026-10-18 04:03:18.900 xcodebuild[4797:25042244] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 04:03:18.996 xcodebuild[4797:25042243] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. fatal: not a git repository (or any parent up to mount point /Volumes) Sto<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat .agents/skills/semedit/references/golang.md; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README*&#39; -g &#39;AGENTS*&#39;; git status --short&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
files = [&#39;request/request.go&#39;, &#39;queue/queue.go&#39;, &#39;scheduler/jobs.go&#39;, &#39;correlation/identities.go&#39;, &#39;gateway/handler.go&#39;, &#39;notifications/notifier.go&#39;, &#39;reporting/report.go&#39;, &#39;storage/requests.go&#39;]
for name in files:
    p = Path(name)
    s = p.read_text()
    s = s.replace(&#39;r.ID&#39;, &#39;r.RequestID&#39;).replace(&#39;.Request.ID&#39;, &#39;.Request.RequestID&#39;)
    if name == &#39;request/request.go&#39;:
        s = s.replace(&#39;ID      string&#39;, &#39;RequestID string&#39;)
        s = s.replace(&#39;type Request struct {&#39;, &#39;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)\\n\\ntype Request struct {&#39;)
        s = s.replace(&#39;return errors.New(\&#34;invalid request\&#34;)&#39;, &#39;return ErrInvalidRequest&#39;)
        s = s.replace(&#39;func Normalize(kind string) string {&#39;, &#39;func Normalize(kind string) string {\\n\\treturn normalizeKind(kind)\\n}\\n\\nfunc normalizeKind(kind string) string {&#39;)
    p.write_text(s)
PY

gofmt -w request/request.go queue/queue.go scheduler/jobs.go correlation/identities.go gateway/handler.go notifications/notifier.go reporting/report.go storage/requests.go&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go; rg -n &#39;\\.ID|RequestID&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/profilerecord compile: version &#34;g<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
comments = {
&#39;request/request.go&#39;: &#39;Request identity validation and kind normalization belong at the public request boundary.&#39;,
&#39;queue/queue.go&#39;: &#39;Queue keys correlate entries with request identities.&#39;,
&#39;scheduler/jobs.go&#39;: &#39;Job matching uses the identity of the enclosed request.&#39;,
&#39;correlation/identities.go&#39;: &#39;Correlation preserves the separate request, response, and review identity domains.&#39;,
&#39;gateway/handler.go&#39;: &#39;The gateway validates requests before forwarding them to the client.&#39;,
&#39;notifications/notifier.go&#39;: &#39;Notification routing follows the identity of the associated request.&#39;,
&#39;reporting/report.go&#39;: &#39;Reports group rows by their associated request identity.&#39;,
&#39;storage/requests.go&#39;: &#39;Storage lookup compares record keys with incoming request identities.&#39;,
}
for name, comment in comments.items():
    p = Path(name)
    p.write_text(&#39;// &#39; + comment + &#39;\\n&#39; + p.read_text())
Path(&#39;gateway/identity_migration_test.go&#39;).write_text(&#39;&#39;&#39;// These checks preserve sentinel matching across the gateway error boundary.
package gateway

import (
    \&#34;errors\&#34;
    \&#34;testing\&#34;

    \&#34;example.com/identityservice/request\&#34;
)

func TestSubmitWrapsInvalidRequest(t *testing.T) {
    _, err := (Handler{}).Submit(request.Request{})
    if &#34;&#39;!errors.Is(err, request.ErrInvalidRequest) {
        t.Fatalf(&#34;Submit() error = %v, want ErrInvalidRequest&#34;, err)
    }
    if err == request.ErrInvalidRequest {
        t.Fatal(&#34;Submit() lost its error context&#34;)
    }
    if err := (request.Request{RequestID: &#34;req-1&#34;}).Validate(); err != nil {
        t.Fatalf(&#34;Validate() rejected a request identity: %v&#34;, err)
    }
}
&#39;&#34;&#39;&#39;&#39;)
PY
gofmt -w request/request.go queue/queue.go scheduler/jobs.go correlation/identities.go gateway/handler.go gateway/identity_migration_test.go notifications/notifier.go reporting/report.go storage/requests.go
go build ./...
go test ./...&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go version; go env GOROOT GOTOOLDIR GOCACHE; printenv GOROOT GOTOOLDIR GOTOOLCHAIN; rg --files --hidden .scratch | head -40&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Validate&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;env -u GOROOT go build ./... &amp;&amp; \
env -u GOROOT go test ./...&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_diagnostics</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;report_id&#34;: &#34;IHOHSTXD7IEHYHGSUDJ4UUE65T&#34;,
  &#34;category&#34;: &#34;active&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;line range 21-22 does not select a contiguous top-level statement region in Normalize&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 21,
  &#34;end_line&#34;: 22,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat -n request/request.go&#39;</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;placement&#34;: &#34;before_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Request.Validate&#34;,
  &#34;kind&#34;: &#34;if&#34;,
  &#34;discriminator&#34;: &#34;r.RequestID == \&#34;\&#34;&#34;,
  &#34;source&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n return ErrInvalidRequest\n}&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 24,
  &#34;end_line&#34;: 25,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\nkind = strings.TrimSpace(kind)\nkind = strings.ToLower(kind)\nreturn kind\n}&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\nreturn normalizeKind(kind)\n}&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;RequestID|\\.ID|%w|ErrInvalidRequest&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Batch-Use Reflection

Consecutive semantic MCP calls were detected without `semantic_batch`. This diagnostic follow-up is excluded from one-shot, interactive-recovery, correctness, tool-use, token, turn, and latency metrics.

<div class="callout callout-warning"><div class="callout-title"><span>⚠</span> Why semantic edits were not batched</div><div class="callout-desc">Batching was discovered and available. I used standalone calls to inspect each result before continuing, but some edits could have been combined with semantic_batch; this was a workflow choice, not a tool limitation.</div></div>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes and do not run tools. During this task you made consecutive semantic_* MCP calls without using semantic_batch. In one to three sentences, explain why you did not combine those operations with semantic_batch. State whether batching was not discovered, was unsuitable for the operations, could not be used, or had another reason. Do not retry the task.</pre>
</details>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit: I was relieved that its typed rename preserved the lookalike identities and reported compiler diagnostic deltas, though its extraction tool couldn’t handle this normalization block.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

