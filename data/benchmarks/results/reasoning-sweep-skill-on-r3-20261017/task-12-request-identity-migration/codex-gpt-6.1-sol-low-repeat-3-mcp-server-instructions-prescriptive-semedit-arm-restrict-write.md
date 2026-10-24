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

* **Date**: 2026-10-18 06:58:32 CEST

## Test case: `task-12-request-identity-migration`

### Target: `codex/gpt-6.1-sol/low (repeat 3)`

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
| **Wall-Clock Latency** | 38.53s | 38.98s | <span class="benchmark-delta-negative">+1.2%</span> | — | — | — |
| **Process Start → First Event** | 0.16s | 0.07s | — | — | — | — |
| **First Event → First Tool Call** | 4.09s | 4.18s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 10 | 15 | <span class="benchmark-delta-negative">+50.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 10 | 15 | <span class="benchmark-delta-negative">+50.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.061140 | $0.153623 | +151.3% | — | — | — |
| **Output Tokens** | 1450 | 1082 | <span class="benchmark-delta-positive">-25.4%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 180 | 120 | <span class="benchmark-delta-positive">-33.3%</span> | — | — | — |
| **Total Input Tokens** | 163403 | 452496 | <span class="benchmark-delta-negative">+176.9%</span> | — | — | — |
| **Cached Input Tokens** | 147456 | 401152 | <span class="benchmark-delta-positive">+172.0%</span> | — | — | — |
| **Uncached Input Tokens** | 15947 | 51344 | <span class="benchmark-delta-negative">+222.0%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 9.25:1 | 7.81:1 | <span class="benchmark-delta-negative">-15.5%</span> | — | — | — |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md docs/identity-model.md go.mod; rg -n &#39;.&#39; request gateway notifications queue review storage scheduler correlation response reporting&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat .agents/skills/semedit/references/golang.md; rg --files -g &#39;*BRIEF*&#39; -g &#39;*AGENTS*&#39; -g &#39;README*&#39;&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short; rg --files --hidden -g &#39;&#34;&#39;! .git&#39;&#34;&#39; -g &#39;*Make*&#39; -g &#39;*README*&#39; -g &#39;*BRIEF*&#39; -g &#39;AGENTS*&#39;&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
files = [&#39;request/request.go&#39;, &#39;notifications/notifier.go&#39;, &#39;scheduler/jobs.go&#39;, &#39;queue/queue.go&#39;, &#39;gateway/handler.go&#39;, &#39;correlation/identities.go&#39;, &#39;storage/requests.go&#39;, &#39;reporting/report.go&#39;]
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

gofmt -w request/request.go notifications/notifier.go scheduler/jobs.go queue/queue.go gateway/handler.go correlation/identities.go storage/requests.go reporting/report.go&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
comments = {
&#39;request/request.go&#39;: &#39;Request identity and kind normalization belong at the public request boundary.&#39;,
&#39;notifications/notifier.go&#39;: &#39;Notification destinations and batches correlate notices with their requests.&#39;,
&#39;scheduler/jobs.go&#39;: &#39;Jobs are matched by request identity so attempts remain tied to the right request.&#39;,
&#39;queue/queue.go&#39;: &#39;Queue keys track request identity for membership and batching.&#39;,
&#39;gateway/handler.go&#39;: &#39;The gateway validates requests before sending them and preserves upstream response identity.&#39;,
&#39;correlation/identities.go&#39;: &#39;Correlation retains the separate identities of requests, responses, and reviews.&#39;,
&#39;storage/requests.go&#39;: &#39;Storage lookup connects a request identity with its stored record.&#39;,
&#39;reporting/report.go&#39;: &#39;Report rows are grouped by the identity of their associated request.&#39;,
}
for name, comment in comments.items():
    p = Path(name)
    p.write_text(&#39;// &#39; + comment + &#39;\\n&#39; + p.read_text())
PY&#34;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Validate&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34;<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w request/request.go notifications/notifier.go scheduler/jobs.go queue/queue.go gateway/handler.go gateway/identity_migration_test.go correlation/identities.go storage/requests.go reporting/report.go
 go build ./...
 go test ./...&#39;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go version; go env GOROOT GOTOOLDIR GOTOOLCHAIN GOCACHE; env | rg &#39;&#34;&#39;^(GO|PATH)&#39;&#34;&#39; ; rg -n &#39;\\.ID|RequestID|normalizeKind|ErrInvalidRequest&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;env -u GOROOT go build ./...&#39;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Verification returned 3 diagnostics. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;available&#34;,&#34;report_id&#34;:&#34;CLQOVIFZY3NNKL5AXJXTJ6GOGB&#34;,&#34;arguments&#34;:{&#34;category&#34;:&#34;active&#34;,&#34;limit&#34;:50,&#34;offset&#34;:0,&#34;report_id&#34;:&#34;CLQOVIFZY3NNKL5AXJXTJ6GOGB&#34;},&#34;coverage&#34;:[{&#34;status&#34;:&#34;verified&#34;,&#34;build_exclusions&#34;:[&#34;Go build context and build tags&#34;,&#34;testdata&#34;,&#34;build&#34;,&#34;dist&#34;,&#34;target&#34;,&#34;vendor&#34;,&#34;node_modules&#34;,&#34;.scratch&#34;,&#34;.git&#34;,&#34;bazel-*&#34;,&#34;_*&#34;,&#34;.*&#34;],&#34;<pre class="benchmark-tool-arguments"><code class="language-json">{
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
        &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\nreturn fmt.Errorf(\&#34;%w: RequestID is empty\&#34;, ErrInvalidRequest)\n}\nreturn nil&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;env -u GOROOT go test ./...&#39;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;add&#34;: [
          &#34;fmt&#34;
        ]
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 25,
  &#34;end_line&#34;: 27,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
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
<tr><td>15</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: 2026-10-18 04:07:32.132 xcodebuild[9988:25085029] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 04:07:32.227 xcodebuild[9988:25085024] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. warning: Not a git repository. Use --no-index to compare two paths outside a working tree usage: git diff --no-index [&lt;options&gt;] &lt;path&gt; &lt;path&gt; [&lt;pathspec&gt;...] Diff outpu<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --stat; git diff -- request/request.go; git diff --name-only -- go.mod go.sum &#39;*_test.go&#39; response review&#34;</code></pre></td></tr>
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
| **Wall-Clock Latency** | 38.53s | 29.07s | <span class="benchmark-delta-positive">-24.5%</span> | — | — | — |
| **Process Start → First Event** | 0.16s | 0.07s | — | — | — | — |
| **First Event → First Tool Call** | 4.09s | 5.43s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 10 | 11 | <span class="benchmark-delta-negative">+10.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 1 | <span class="benchmark-delta-positive">-66.7%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 10 | 11 | <span class="benchmark-delta-negative">+10.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.061140 | $0.089718 | +46.7% | — | — | — |
| **Output Tokens** | 1450 | 801 | <span class="benchmark-delta-positive">-44.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 180 | 45 | <span class="benchmark-delta-positive">-75.0%</span> | — | — | — |
| **Total Input Tokens** | 163403 | 260950 | <span class="benchmark-delta-negative">+59.7%</span> | — | — | — |
| **Cached Input Tokens** | 147456 | 231680 | <span class="benchmark-delta-positive">+57.1%</span> | — | — | — |
| **Uncached Input Tokens** | 15947 | 29270 | <span class="benchmark-delta-negative">+83.5%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 9.25:1 | 7.92:1 | <span class="benchmark-delta-negative">-14.4%</span> | — | — | — |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; rg --files -g &#39;*.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.mod&#39;&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat AGENTS.override.md docs/identity-model.md go.mod; rg -n &#39;.&#39; request gateway notifications queue review storage scheduler correlation response reporting&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short; rg --files --hidden -g &#39;&#34;&#39;! .git&#39;&#34;&#39; -g &#39;*Make*&#39; -g &#39;*README*&#39; -g &#39;*BRIEF*&#39; -g &#39;AGENTS*&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: gateway/handler_test.go gateway/handler.go notifications/notifier.go request/request.go request/request_test.go queue/queue.go review/review_test.go review/review.go storage/requests.go storage/requests_test.go docs/identity-model.md scheduler/jobs.go correlation/identities.go AGENTS.override.md response/response_test.go response/response.go response/audit.go reporting/report.go cat: .agents/skills/semedit/references/golang.md: No such file or directory<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;*.md&#39; -g &#39;*.go&#39;; cat .agents/skills/semedit/references/golang.md&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
files = [&#39;request/request.go&#39;, &#39;notifications/notifier.go&#39;, &#39;scheduler/jobs.go&#39;, &#39;queue/queue.go&#39;, &#39;gateway/handler.go&#39;, &#39;correlation/identities.go&#39;, &#39;storage/requests.go&#39;, &#39;reporting/report.go&#39;]
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

gofmt -w request/request.go notifications/notifier.go scheduler/jobs.go queue/queue.go gateway/handler.go correlation/identities.go storage/requests.go reporting/report.go&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
comments = {
&#39;request/request.go&#39;: &#39;Request identity and kind normalization belong at the public request boundary.&#39;,
&#39;notifications/notifier.go&#39;: &#39;Notification destinations and batches correlate notices with their requests.&#39;,
&#39;scheduler/jobs.go&#39;: &#39;Jobs are matched by request identity so attempts remain tied to the right request.&#39;,
&#39;queue/queue.go&#39;: &#39;Queue keys track request identity for membership and batching.&#39;,
&#39;gateway/handler.go&#39;: &#39;The gateway validates requests before sending them and preserves upstream response identity.&#39;,
&#39;correlation/identities.go&#39;: &#39;Correlation retains the separate identities of requests, responses, and reviews.&#39;,
&#39;storage/requests.go&#39;: &#39;Storage lookup connects a request identity with its stored record.&#39;,
&#39;reporting/report.go&#39;: &#39;Report rows are grouped by the identity of their associated request.&#39;,
}
for name, comment in comments.items():
    p = Path(name)
    p.write_text(&#39;// &#39; + comment + &#39;\\n&#39; + p.read_text())
PY&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # Identity model Requests, responses, reviews, and storage records have distinct identity domains. A response or review may carry a request correlation ID, but its own ID remains the identity used by its indexes. Request kinds are normalized at the public request boundary. package request import ( &#34;errors&#34; &#34;strings&#34; ) type Request struct { ID string Kind string Payload []byte } func (r Request) Validate() error { if r.ID == &#34;&#34; { return errors.New(&#34;invalid request&#34;) } return nil } func Normalize(kind string)<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/identity-model.md request/request.go request/request_test.go; git status --short&#39;</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34;<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w request/request.go notifications/notifier.go scheduler/jobs.go queue/queue.go gateway/handler.go gateway/identity_migration_test.go correlation/identities.go storage/requests.go reporting/report.go
 go build ./...
 go test ./...&#39;</code></pre></td><td><code>semedit/semantic_diagnostics</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;report_id&#34;: &#34;LXCJ5ZGZJNUORTPXGUAIDDYKJR&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;go version; go env GOROOT GOTOOLDIR GOTOOLCHAIN GOCACHE; env | rg &#39;&#34;&#39;^(GO|PATH)&#39;&#34;&#39; ; rg -n &#39;\\.ID|RequestID|normalizeKind|ErrInvalidRequest&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 22,
  &#34;end_line&#34;: 23,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;normalizeKind&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;env -u GOROOT go build ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;nl -ba request/request.go; cat gateway/handler.go storage/requests.go&#39;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;env -u GOROOT go test ./...&#39;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
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
        &#34;body&#34;: &#34;{\nif r.RequestID == \&#34;\&#34; { return ErrInvalidRequest }\nreturn nil\n}&#34;
      }
    },
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
        &#34;body&#34;: &#34;{ return normalizeKind(kind) }&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go; rg -n &#39;\\.ID|RequestID|Request\\{&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>I’d ask my owner to add semedit: I felt reassured when its semantic rename preserved unrelated IDs across packages, and its compiler diagnostic deltas gave me useful verification, though the extraction tool refused this normalization block.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

