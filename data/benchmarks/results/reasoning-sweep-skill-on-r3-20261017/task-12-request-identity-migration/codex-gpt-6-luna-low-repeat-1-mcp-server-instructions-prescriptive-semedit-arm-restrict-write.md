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

* **Date**: 2026-10-18 06:58:34 CEST

## Test case: `task-12-request-identity-migration`

### Target: `codex/gpt-6-luna/low (repeat 1)`

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
| **Wall-Clock Latency** | 31.16s | 39.45s | <span class="benchmark-delta-negative">+26.6%</span> | — | — | — |
| **Process Start → First Event** | 0.31s | 0.17s | — | — | — | — |
| **First Event → First Tool Call** | 3.78s | 5.60s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 13 | 19 | <span class="benchmark-delta-negative">+46.2%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 4 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 13 | 19 | <span class="benchmark-delta-negative">+46.2%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.007370 | $0.011065 | +50.1% | — | — | — |
| **Output Tokens** | 2083 | 1458 | <span class="benchmark-delta-positive">-30.0%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 315914 | 681662 | <span class="benchmark-delta-negative">+115.8%</span> | — | — | — |
| **Cached Input Tokens** | 280704 | 642560 | <span class="benchmark-delta-positive">+128.9%</span> | — | — | — |
| **Uncached Input Tokens** | 35210 | 39102 | <span class="benchmark-delta-negative">+11.1%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 7.97:1 | 16.43:1 | <span class="benchmark-delta-positive">+106.1%</span> | — | — | — |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.work&#39; | head -200&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;.&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39;; rg -n &#39;\\bRequest\\b|\\.ID\\b|Normalize|Validate|ErrInvalid&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: cat: PROJECT_BRIEF.md: No such file or directory<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat PROJECT_BRIEF.md&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: package request import ( &#34;errors&#34; &#34;strings&#34; ) type Request struct { ID string Kind string Payload []byte } func (r Request) Validate() error { if r.ID == &#34;&#34; { return errors.New(&#34;invalid request&#34;) } return nil } func Normalize(kind string) string { kind = strings.TrimSpace(kind) kind = strings.ToLower(kind) return kind } func FindByID(requests []Request, id string) (Request, bool) { for _, r := range requests { if r.ID == id { return r, true } } return Request{}, false } package request import &#34;testing&#34; func<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
cat request/request_test.go &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | head -100 &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,240p&#39; request/request.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/identity-model.md &amp;&amp; \
cat gateway/handler.go &amp;&amp; \
cat queue/queue.go &amp;&amp; \
cat scheduler/jobs.go &amp;&amp; \
cat notifications/notifier.go &amp;&amp; \
cat reporting/report.go &amp;&amp; \
cat storage/requests.go &amp;&amp; \
cat correlation/identities.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;unknown param \&#34;query\&#34;; supported keys are [bash-bin, bash_bin, file, ghc-bin, ghc-version, ghc_bin, ghc_version, haskell-standalone, hls-bin, hls-version, hls_bin, hls_version, import-maven, import_maven, java-bin, java-version, java_bin, java_version, jdtls-home, jdtls_home, kotlin-bin, kotlin_bin, language, make-bin, make_bin, metals-bin, metals-home, metals_bin, metals_home, standalone_haskell, symbol, trust-workspace, trust_workspace]: in<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;query&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;request&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Response.ID&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: zsh:1: unmatched &#39;<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\b(r|left\\.Request|right\\.Request|job\\.Request|record|resp|review)\\.ID\\b|Request\\{\&#34; --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\.ID\\b|Request\\{&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;Review.ID&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;unknown param \&#34;new_name\&#34;; supported keys are [auto_organize_imports, file, import-maven, import_maven, java-bin, java_bin, jdtls-home, jdtls_home, language, symbol, to, trust-workspace, trust_workspace]: invalid operation parameters&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;new_name&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/msan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/trace/tracev2 compile: version &#34;go1.27.2&#34; does not ma<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w request/request.go request/request_test.go gateway/handler.go queue/queue.go scheduler/jobs.go notifications/notifier.go reporting/report.go storage/requests.go correlation/identities.go &amp;&amp; \
go test ./... &amp;&amp; \
go build ./...&#39;</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64 /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/pkg/tool/darwin_arm64 go1.27.1 /Volumes/semedit-bench-ram/run_codex_baseline-diff_small_task-12-request-identity-migration_1792287600129595000/.scratch/go/build /opt/homebrew/bin/go fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOV<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go env GOROOT GOTOOLDIR GOVERSION GOCACHE; command -v go; git status --short; git diff --stat; git diff -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Request&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -d /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@*/bin/go 2&gt;/dev/null; env | rg &#39;&#34;&#39;^(GOROOT|GOTOOLCHAIN|GOENV|PATH)=&#39;&#34;&#39;; ls -la&#34;</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Request.Validate&#34;,
  &#34;source&#34;: &#34;func (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn fmt.Errorf(\&#34;%w: request ID is empty\&#34;, ErrInvalidRequest)\n\t}\n\treturn nil\n}&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;PATH=&#34;/Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/bin:$PATH&#34; GOTOOLCHAIN=local go test ./... &amp;&amp; \
PATH=&#34;/Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/bin:$PATH&#34; GOTOOLCHAIN=local go build ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,100p&#39; request/request.go&#34;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;request/request.go&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_extract_function</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;cannot extract Normalize: only one leading := declaration with matching named results is supported&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;,
  &#34;start_line&#34;: 25,
  &#34;end_line&#34;: 26,
  &#34;symbol&#34;: &#34;normalizeKind&#34;,
  &#34;kind&#34;: &#34;function&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;placement&#34;: &#34;after_symbol&#34;,
  &#34;target_symbol&#34;: &#34;ErrInvalidRequest&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-explicit_small-explicit_task-12-request-identity-migration_1792287675504569000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;,
      &#34;params&#34;: {
        &#34;language&#34;: &#34;go&#34;,
        &#34;file&#34;: &#34;request/request.go&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_diagnostics</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;report_id&#34;: &#34;MJJARVKLBYU3HLWKWAOWDAKORX&#34;,
  &#34;category&#34;: &#34;active&#34;,
  &#34;limit&#34;: 10,
  &#34;offset&#34;: 0
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
| **Wall-Clock Latency** | 31.16s | 41.47s | <span class="benchmark-delta-negative">+33.1%</span> | — | — | — |
| **Process Start → First Event** | 0.31s | 0.17s | — | — | — | — |
| **First Event → First Tool Call** | 3.78s | 3.25s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 13 | 24 | <span class="benchmark-delta-negative">+84.6%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 4 | 1 | <span class="benchmark-delta-positive">-75.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 13 | 24 | <span class="benchmark-delta-negative">+84.6%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.007370 | $0.019879 | +169.7% | — | — | — |
| **Output Tokens** | 2083 | 2371 | <span class="benchmark-delta-negative">+13.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 315914 | 1132724 | <span class="benchmark-delta-negative">+258.6%</span> | — | — | — |
| **Cached Input Tokens** | 280704 | 1050880 | <span class="benchmark-delta-positive">+274.4%</span> | — | — | — |
| **Uncached Input Tokens** | 35210 | 81844 | <span class="benchmark-delta-negative">+132.4%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 7.97:1 | 12.84:1 | <span class="benchmark-delta-positive">+61.1%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **Oracle L2: AST Invariants** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **Oracle L3: Clean Build** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File request/request.go modified (34 lines -> 40 lines)
* **MCP Edit**: File request/request.go modified (34 lines -> 41 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;Makefile&#39; -g &#39;go.work&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; -g &#39;go.sum&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39;; rg -n &#39;\\bRequest\\b|\\.ID\\b|Normalize|Validate|ErrInvalid&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/SKILL.md&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: package request import ( &#34;errors&#34; &#34;strings&#34; ) type Request struct { ID string Kind string Payload []byte } func (r Request) Validate() error { if r.ID == &#34;&#34; { return errors.New(&#34;invalid request&#34;) } return nil } func Normalize(kind string) string { kind = strings.TrimSpace(kind) kind = strings.ToLower(kind) return kind } func FindByID(requests []Request, id string) (Request, bool) { for _, r := range requests { if r.ID == id { return r, true } } return Request{}, false } package request import &#34;testing&#34; func<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
cat request/request_test.go &amp;&amp; \
rg --files -g &#39;&#34;&#39;!go.mod&#39;&#34;&#39; -g &#39;&#34;&#39;!go.sum&#39;&#34;&#39; | head -100 &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: cat: .agents/skills/semedit/references/golang.md: No such file or directory<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat .agents/skills/semedit/references/golang.md &amp;&amp; \
cat request/request.go &amp;&amp; \
cat request/request_test.go&#39;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat docs/identity-model.md &amp;&amp; \
cat gateway/handler.go &amp;&amp; \
cat queue/queue.go &amp;&amp; \
cat scheduler/jobs.go &amp;&amp; \
cat notifications/notifier.go &amp;&amp; \
cat reporting/report.go &amp;&amp; \
cat storage/requests.go &amp;&amp; \
cat correlation/identities.go &amp;&amp; \
cat go.mod&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;SKILL.md&#39; -g &#39;golang.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS*&#39; | head -50 &amp;&amp; \
cat request/request.go &amp;&amp; \
cat request/request_test.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;include_tests&#34;: true
}</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: zsh:1: unmatched &#39;<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\b(r|left\\.Request|right\\.Request|job\\.Request|record|resp|review)\\.ID\\b|Request\\{\&#34; --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Validate&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;source&#34;: &#34;func (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn fmt.Errorf(\&#34;%w: request ID is empty\&#34;, ErrInvalidRequest)\n\t}\n\treturn nil\n}&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;\\.ID\\b|Request\\{&#39; --glob &#39;*.go&#39;&#34;</code></pre></td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;placement&#34;: &#34;file_start&#34;,
  &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;placement&#34;: &#34;file_end&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/msan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/trace/tracev2 compile: version &#34;go1.27.2&#34; does not ma<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w request/request.go request/request_test.go gateway/handler.go queue/queue.go scheduler/jobs.go notifications/notifier.go reporting/report.go storage/requests.go correlation/identities.go &amp;&amp; \
go test ./... &amp;&amp; \
go build ./...&#39;</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Normalize&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;source&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64 /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/pkg/tool/darwin_arm64 go1.27.1 /Volumes/semedit-bench-ram/run_codex_baseline-diff_small_task-12-request-identity-migration_1792287600129595000/.scratch/go/build /opt/homebrew/bin/go fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOV<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go env GOROOT GOTOOLDIR GOVERSION GOCACHE; command -v go; git status --short; git diff --stat; git diff -- go.mod go.sum&#39;</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;FindByID&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;source&#34;: &#34;func FindByID(requests []Request, id string) (Request, bool) {\n\tfor _, r := range requests {\n\t\tif r.RequestID == id {\n\t\t\treturn r, true\n\t\t}\n\t}\n\treturn Request{}, false\n}&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;ls -d /Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@*/bin/go 2&gt;/dev/null; env | rg &#39;&#34;&#39;^(GOROOT|GOTOOLCHAIN|GOENV|PATH)=&#39;&#34;&#39;; ls -la&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: 2026-10-18 03:40:56.475 xcodebuild[83285:24865942] DVTFilePathFSEvents: Failed to start fs event stream. 2026-10-18 03:40:56.572 xcodebuild[83285:24865941] [MT] DVTDeveloperPaths: Failed to get length of DARWIN_USER_CACHE_DIR from confstr(3), error = Error Domain=NSPOSIXErrorDomain Code=5 &#34;Input/output error&#34;. Using NSCachesDirectory instead. warning: Limiting comparison with pathspecs is only supported if both paths are directories. usage: git diff --no-index [&lt;options&gt;] &lt;path&gt; &lt;path&gt; [&lt;pathspec&gt;...] Diff <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff -- . &#39;:&#34;&#39;!go.mod&#39;&#34;&#39; &#39;:&#34;&#39;!go.sum&#39;&#34;&#39; &amp;&amp; \
git status --short &amp;&amp; \
cat request/request.go &amp;&amp; \
rg &#39;\\b(ID|RequestID)\\b&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;PATH=&#34;/Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/bin:$PATH&#34; GOTOOLCHAIN=local go test ./... &amp;&amp; \
PATH=&#34;/Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/bin:$PATH&#34; GOTOOLCHAIN=local go build ./...&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: fatal: not a git repository (or any parent up to mount point /Volumes) Stopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
git diff &amp;&amp; \
cat request/request.go &amp;&amp; \
rg &#39;\\b(ID|RequestID)\\b&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: package request import ( &#34;errors&#34; &#34;fmt&#34; &#34;strings&#34; ) var ErrInvalidRequest = errors.New(&#34;invalid request&#34;) type Request struct { RequestID string Kind string Payload []byte } func (r Request) Validate() error { if r.RequestID == &#34;&#34; { return fmt.Errorf(&#34;%w: request ID is empty&#34;, ErrInvalidRequest) } return nil } func Normalize(kind string) string { return normalizeKind(kind) } func FindByID(requests []Request, id string) (Request, bool) { for _, r := range requests { if r.RequestID == id { return r, true } } <pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat request/request.go &amp;&amp; \
rg &#39;\\b(ID|RequestID)\\b&#39; --glob &#39;*.go&#39; &amp;&amp; \
shasum -a 256 go.mod go.sum&#34;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;symbol&#34;: &#34;Request.RequestID&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><code>semedit/semantic_diagnostics</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;report_id&#34;: &#34;C3RE4W3P4U4SHEDL3XONXRX2XG&#34;,
  &#34;category&#34;: &#34;active&#34;,
  &#34;limit&#34;: 50
}</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request\\{&#39; --glob &#39;*.go&#39; &amp;&amp; \
rg -n &#39;Request\\(&#39; --glob &#39;*.go&#39;&#34;</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat request/request_test.go&#39;</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request_test.go&#34;,
  &#34;symbol&#34;: &#34;TestFindByIDUsesRequestIdentity&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;source&#34;: &#34;func TestFindByIDUsesRequestIdentity(t *testing.T) {\n\twant := Request{RequestID: \&#34;req-7\&#34;, Kind: \&#34;audit\&#34;}\n\tgot, ok := FindByID([]Request{{RequestID: \&#34;other\&#34;, Kind: \&#34;audit\&#34;}, want}, \&#34;req-7\&#34;)\n\tif !ok || got.Kind != want.Kind {\n\t\tt.Fatalf(\&#34;FindByID() = %#v, %t; want %#v, true\&#34;, got, ok, want)\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;replace_construct TestSubmit in /Volumes/semedit-bench-ram/run_codex_semedit_small-implicit_task-12-request-identity-migration_1792287631851360000/gateway/handler_test.go: symbol not found&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;gateway/handler_test.go&#34;,
  &#34;symbol&#34;: &#34;TestSubmit&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;source&#34;: &#34;func TestSubmit(t *testing.T) {\n\tclient := &amp;fakeClient{response: response.Response{ID: \&#34;resp-1\&#34;, RequestID: \&#34;req-1\&#34;, Status: 202}}\n\tresp, err := (Handler{client: client}).Submit(request.Request{RequestID: \&#34;req-1\&#34;, Kind: \&#34;audit\&#34;})\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n\tif client.got.Kind != \&#34;audit\&#34; || resp.ID != \&#34;resp-1\&#34; || resp.RequestID != \&#34;req-1\&#34; {\n\t\tt.Fatalf(\&#34;unexpected submit result: %#v %#v\&#34;, client.got, resp)\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat gateway/handler_test.go &amp;&amp; \
cat storage/requests_test.go&#39;</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;gateway/handler_test.go&#34;,
  &#34;symbol&#34;: &#34;TestSubmitCarriesRequestAndResponseIdentities&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;source&#34;: &#34;func TestSubmitCarriesRequestAndResponseIdentities(t *testing.T) {\n\tclient := &amp;testClient{}\n\tresp, err := (Handler{client: client}).Submit(request.Request{RequestID: \&#34;req-1\&#34;, Kind: \&#34;audit\&#34;})\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n\tif client.got.Kind != \&#34;audit\&#34; || resp.ID != \&#34;resp-1\&#34; || resp.RequestID != \&#34;req-1\&#34; {\n\t\tt.Fatalf(\&#34;request=%#v response=%#v\&#34;, client.got, resp)\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;storage/requests_test.go&#34;,
  &#34;symbol&#34;: &#34;TestLookupMatchesRequestIdentity&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;source&#34;: &#34;func TestLookupMatchesRequestIdentity(t *testing.T) {\n\tgot, ok := Lookup([]RequestRecord{{ID: \&#34;req-1\&#34;}, {ID: \&#34;req-2\&#34;}}, request.Request{RequestID: \&#34;req-2\&#34;})\n\tif !ok || got.ID != \&#34;req-2\&#34; {\n\t\tt.Fatalf(\&#34;Lookup() = %#v, %t\&#34;, got, ok)\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Request\\{\&#34;|Request\\{[&#34;&#39;^A-Za-z&#39;&#34;\\n]*\&#34;&#39; --glob &#39;*.go&#39; || \
true; rg -n &#39;request\\.Request\\{&#39; --glob &#39;*.go&#39;; cat go.mod; test &#34;&#39;! -e go.sum &amp;&amp; \
echo &#39;&#34;&#39;go.sum absent&#39;&#34;</code></pre></td></tr>
</tbody>
</table>

