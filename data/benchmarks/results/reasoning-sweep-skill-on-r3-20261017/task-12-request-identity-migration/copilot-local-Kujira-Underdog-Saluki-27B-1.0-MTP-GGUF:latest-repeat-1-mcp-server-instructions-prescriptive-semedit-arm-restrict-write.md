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

### Target: `copilot/local/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `copilot_agent_name=Copilot,copilot_agent_version=1.0.88,copilot_model_selection=acp_session_set_config_option,copilot_protocol_version=1,fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,local_model_context_tokens=212992,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=copilot_acp_v1`

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
| **Wall-Clock Latency** | 300.01s | 300.02s | <span class="benchmark-delta-negative">+0.0%</span> | — | — | — |
| **Process Start → First Event** | 17.58s | 37.57s | — | — | — | — |
| **First Event → First Tool Call** | 0.01s | 0.01s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 34 | 16 | <span class="benchmark-delta-positive">-52.9%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 18 | 1 | <span class="benchmark-delta-positive">-94.4%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 34 | 16 | <span class="benchmark-delta-positive">-52.9%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | — | — | — | — | — | — |
| **Output Tokens** | 4950 | 3196 | <span class="benchmark-delta-positive">-35.4%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 173584 | 613146 | <span class="benchmark-delta-negative">+253.2%</span> | — | — | — |
| **Cached Input Tokens** | 155077 | 567811 | <span class="benchmark-delta-positive">+266.1%</span> | — | — | — |
| **Uncached Input Tokens** | 18507 | 45335 | <span class="benchmark-delta-negative">+145.0%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.38:1 | 12.52:1 | <span class="benchmark-delta-positive">+49.5%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;find . -type f -name \&#34;*.go\&#34; | head -50 &amp;&amp; echo \&#34;---\&#34; &amp;&amp; ls -la&#34;,
  &#34;description&#34;: &#34;List Go files and repo root&#34;
}</code></pre></td><td><code>skill</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;skill&#34;: &#34;semedit&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request.go&#34;
}</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request_test.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit-explicit_small-explicit_task-12-request-identity-migration_1792297398829340000/request/request.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/correlation/identities.go&#34;
}</code></pre></td><td><code>semedit/semantic_find_references</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/response/response.go&#34;
}</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/response/response_test.go&#34;
}</code></pre></td><td><code>semedit/semantic_diagnostics</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;report_id&#34;: &#34;FYLVXLXIAXNU3A5TUMC2ESMEZE&#34;,
  &#34;category&#34;: &#34;active&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/review/review.go&#34;
}</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): MCP server &#39;semedit&#39;: Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 0 params: unknown parameter \&#34;kind\&#34; for \&#34;semantic_insert_decl\&#34;; supported keys are [access, access_modifier, auto-organize-imports, auto_organize_imports, file, g<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;kind&#34;: &#34;var&#34;,
        &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
        &#34;placement&#34;: &#34;file_end&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;Validate&#34;,
        &#34;source&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)\n}\nreturn nil&#34;
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
<tr><td>8</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/review/review_test.go&#34;
}</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): MCP server &#39;semedit&#39;: Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 1 params: unknown parameter \&#34;source\&#34; for \&#34;semantic_replace_body\&#34;; supported keys are [auto-organize-imports, auto_organize_imports, bash-bin, bash_bin, body, exp<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
        &#34;placement&#34;: &#34;file_end&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;Validate&#34;,
        &#34;source&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)\n}\nreturn nil&#34;
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
<tr><td>9</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/storage/requests.go&#34;
}</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): MCP server &#39;semedit&#39;: Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;ok&#34;,&#34;diagnostic_status&#34;:&#34;deferred&#34;,&#34;result&#34;:{&#34;result&#34;:&#34;Successfully inserted declaration.&#34;,&#34;edit_receipt&#34;:{&#34;status&#34;:&#34;applied_working&#34;,&#34;changes_count&#34;:1,&#34;operation&#34;:&#34;insert_decl&#34;,&#34;read_set_count&#34;:1,&#34;schema_version&#34;:&#34;s<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;source&#34;: &#34;var ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)&#34;,
        &#34;placement&#34;: &#34;file_end&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;request/request.go&#34;,
        &#34;symbol&#34;: &#34;Validate&#34;,
        &#34;body&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)\n}\nreturn nil&#34;
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
<tr><td>10</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/storage/requests_test.go&#34;
}</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): MCP server &#39;semedit&#39;: Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;in_function is required for nested kind=if&#34;} {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;in_function is required for nested kind=if&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;if&#34;,
  &#34;source&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)\n}&#34;,
  &#34;discriminator&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn errors.New(\&#34;invalid request\&#34;)\n}&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/reporting/report.go&#34;
}</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): MCP server &#39;semedit&#39;: Critical result (JSON): {&#34;application_state&#34;:&#34;not_applied&#34;,&#34;error&#34;:&#34;if \&#34;if r.RequestID == \\\&#34;\\\&#34; {\\n\\treturn errors.New(\\\&#34;invalid request\\\&#34;)\\n}\&#34; not found in Request.Validate; available candidates (select an if by its condition or short initializer, or set construct_path):\nconstruct_path 0 at /Volumes/semedit-bench-ram/run_copilot_semedit-explicit_small-explicit_task-12-request-identity-migration_1792297398829340000/request/request.go:15:2: if r.RequestID == \&#34;\&#34;\nfile: req<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;if&#34;,
  &#34;source&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)\n}&#34;,
  &#34;discriminator&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn errors.New(\&#34;invalid request\&#34;)\n}&#34;,
  &#34;in_function&#34;: &#34;Request.Validate&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/scheduler/jobs.go&#34;
}</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>): MCP server &#39;semedit&#39;: Critical result (JSON): {&#34;application_state&#34;:&#34;not_applied&#34;,&#34;error&#34;:&#34;selection revision mismatch\nfile: request/request.go\nin_function: Request.Validate\nkind: if\ndiscriminator: r.RequestID == \&#34;\&#34;\nCurrent replacement refused before writing for request/request.go. The supplied expected_selection_revision differs from the token recomputed from the current ordered candidate list of paths and normalized constructs. Reselect from the current choices and token for this request target with<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;if&#34;,
  &#34;source&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)\n}&#34;,
  &#34;discriminator&#34;: &#34;r.RequestID == \&#34;\&#34;&#34;,
  &#34;construct_path&#34;: &#34;0&#34;,
  &#34;expected_selection_revision&#34;: &#34;daa34228defc3b154ff24adbbce2a4de0c4d9b3065dbc4b8df574ea62de0763b&#34;,
  &#34;in_function&#34;: &#34;Request.Validate&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/queue/queue.go&#34;
}</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;if&#34;,
  &#34;source&#34;: &#34;if r.RequestID == \&#34;\&#34; {\n\treturn fmt.Errorf(\&#34;request ID is empty: %w\&#34;, ErrInvalidRequest)\n}&#34;,
  &#34;discriminator&#34;: &#34;r.RequestID == \&#34;\&#34;&#34;,
  &#34;construct_path&#34;: &#34;0&#34;,
  &#34;expected_selection_revision&#34;: &#34;14ada3464191f96f90efec80600ed64e1d62956f483fd4618fec1a10362c3dcc&#34;,
  &#34;in_function&#34;: &#34;Request.Validate&#34;
}</code></pre></td></tr>
<tr><td>14</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/notifications/notifier.go&#34;
}</code></pre></td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;source&#34;: &#34;func normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;placement&#34;: &#34;before_symbol&#34;,
  &#34;target_symbol&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>15</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/gateway/handler.go&#34;
}</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;source&#34;: &#34;return normalizeKind(kind)&#34;,
  &#34;discriminator&#34;: &#34;return kind&#34;,
  &#34;in_function&#34;: &#34;Normalize&#34;
}</code></pre></td></tr>
<tr><td>16</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/gateway/handler_test.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit-explicit_small-explicit_task-12-request-identity-migration_1792297398829340000/request/request.go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/response/audit.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>18</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls docs/ &amp;&amp; cat go.mod &amp;&amp; cat docs/* 2&gt;/dev/null | head -50&#34;,
  &#34;description&#34;: &#34;Inspect docs and go.mod&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>19</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request.go&#34;,
  &#34;old_str&#34;: &#34;type Request struct {\n\tID      string\n\tKind    string\n\tPayload []byte\n}\n\nfunc (r Request) Validate() error {\n\tif r.ID == \&#34;\&#34; {\n\t\treturn errors.New(\&#34;invalid request\&#34;)\n\t}\n\treturn nil\n}&#34;,
  &#34;new_str&#34;: &#34;type Request struct {\n\tRequestID string\n\tKind      string\n\tPayload   []byte\n}\n\nvar ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)\n\nfunc (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn ErrInvalidRequest\n\t}\n\treturn nil\n}&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>20</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request.go&#34;,
  &#34;old_str&#34;: &#34;func Normalize(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;new_str&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}\n\nfunc normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>21</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request.go&#34;,
  &#34;old_str&#34;: &#34;\t\tif r.ID == id {&#34;,
  &#34;new_str&#34;: &#34;\t\tif r.RequestID == id {&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>22</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/correlation/identities.go&#34;,
  &#34;old_str&#34;: &#34;\treturn []string{r.ID, resp.ID, review.ID}&#34;,
  &#34;new_str&#34;: &#34;\treturn []string{r.RequestID, resp.ID, review.ID}&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>23</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/storage/requests.go&#34;,
  &#34;old_str&#34;: &#34;\t\tif record.ID == r.ID {&#34;,
  &#34;new_str&#34;: &#34;\t\tif record.ID == r.RequestID {&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>24</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/scheduler/jobs.go&#34;,
  &#34;old_str&#34;: &#34;\treturn left.Request.ID == right.Request.ID&#34;,
  &#34;new_str&#34;: &#34;\treturn left.Request.RequestID == right.Request.RequestID&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>25</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/scheduler/jobs.go&#34;,
  &#34;old_str&#34;: &#34;\t\tif job.Request.ID == r.ID &amp;&amp; job.Attempts == 0 { pending = append(pending, job) }&#34;,
  &#34;new_str&#34;: &#34;\t\tif job.Request.RequestID == r.RequestID &amp;&amp; job.Attempts == 0 { pending = append(pending, job) }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>26</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/reporting/report.go&#34;,
  &#34;old_str&#34;: &#34;\tfor _, r := range rows { grouped[r.Request.ID] = append(grouped[r.Request.ID], r) }&#34;,
  &#34;new_str&#34;: &#34;\tfor _, r := range rows { grouped[r.Request.RequestID] = append(grouped[r.Request.RequestID], r) }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>27</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/queue/queue.go&#34;,
  &#34;old_str&#34;: &#34;\t\tif entry.Key == r.ID { return true }&#34;,
  &#34;new_str&#34;: &#34;\t\tif entry.Key == r.RequestID { return true }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>28</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/queue/queue.go&#34;,
  &#34;old_str&#34;: &#34;\tfor _, r := range requests { keys = append(keys, r.ID) }&#34;,
  &#34;new_str&#34;: &#34;\tfor _, r := range requests { keys = append(keys, r.RequestID) }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>29</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/notifications/notifier.go&#34;,
  &#34;old_str&#34;: &#34;\tif r.ID == \&#34;\&#34; { return \&#34;unassigned\&#34; }\n\treturn \&#34;request:\&#34; + r.ID&#34;,
  &#34;new_str&#34;: &#34;\tif r.RequestID == \&#34;\&#34; { return \&#34;unassigned\&#34; }\n\treturn \&#34;request:\&#34; + r.RequestID&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>30</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/notifications/notifier.go&#34;,
  &#34;old_str&#34;: &#34;\tfor _, r := range notices { ids = append(ids, r.Request.ID) }&#34;,
  &#34;new_str&#34;: &#34;\tfor _, r := range notices { ids = append(ids, r.Request.RequestID) }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>31</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/gateway/handler.go&#34;,
  &#34;old_str&#34;: &#34;\t\treturn response.Response{}, fmt.Errorf(\&#34;submit %s: %w\&#34;, r.ID, err)&#34;,
  &#34;new_str&#34;: &#34;\t\treturn response.Response{}, fmt.Errorf(\&#34;submit %s: %w\&#34;, r.RequestID, err)&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>32</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;\\.ID\\b&#34;,
  &#34;paths&#34;: &#34;.&#34;,
  &#34;output_mode&#34;: &#34;content&#34;,
  &#34;-n&#34;: true,
  &#34;glob&#34;: &#34;*.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>33</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./... &amp;&amp; go test ./...&#34;,
  &#34;description&#34;: &#34;Build and test all packages&#34;,
  &#34;initial_wait&#34;: 120
}</code></pre></td><td>—</td></tr>
<tr><td>34</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go env GOCACHE GOTOOLCHAIN GOFLAGS GOPATH 2&gt;&amp;1; which go; go version&#34;,
  &#34;description&#34;: &#34;Inspect Go environment&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `copilot_agent_name=Copilot,copilot_agent_version=1.0.88,copilot_model_selection=acp_session_set_config_option,copilot_protocol_version=1,fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,local_model_context_tokens=212992,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=copilot_acp_v1`

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
| **Wall-Clock Latency** | 300.01s | 300.01s | <span class="benchmark-delta-negative">+0.0%</span> | — | — | — |
| **Process Start → First Event** | 17.58s | 37.61s | — | — | — | — |
| **First Event → First Tool Call** | 0.01s | 0.01s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 34 | 23 | <span class="benchmark-delta-positive">-32.4%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 18 | 1 | <span class="benchmark-delta-positive">-94.4%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 34 | 23 | <span class="benchmark-delta-positive">-32.4%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | — | — | — | — | — | — |
| **Output Tokens** | 4950 | 3507 | <span class="benchmark-delta-positive">-29.2%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 173584 | 303837 | <span class="benchmark-delta-negative">+75.0%</span> | — | — | — |
| **Cached Input Tokens** | 155077 | 263405 | <span class="benchmark-delta-positive">+69.9%</span> | — | — | — |
| **Uncached Input Tokens** | 18507 | 40432 | <span class="benchmark-delta-negative">+118.5%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.38:1 | 6.51:1 | <span class="benchmark-delta-negative">-22.3%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;find . -type f -name \&#34;*.go\&#34; | head -50 &amp;&amp; echo \&#34;---\&#34; &amp;&amp; ls -la&#34;,
  &#34;description&#34;: &#34;List Go files and repo root&#34;
}</code></pre></td><td><code>skill</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;skill&#34;: &#34;semedit&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request.go&#34;
}</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request_test.go&#34;
}</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;request&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/correlation/identities.go&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;\\.ID\\b&#34;,
  &#34;paths&#34;: &#34;.&#34;,
  &#34;output_mode&#34;: &#34;content&#34;,
  &#34;-n&#34;: true
}</code></pre></td></tr>
<tr><td>5</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/response/response.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/request/request.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/response/response_test.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/gateway/handler.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/review/review.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/gateway/handler_test.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/review/review_test.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/correlation/identities.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/storage/requests.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/notifications/notifier.go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/storage/requests_test.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/queue/queue.go&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/reporting/report.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/reporting/report.go&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/scheduler/jobs.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/scheduler/jobs.go&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/queue/queue.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/storage/requests.go&#34;
}</code></pre></td></tr>
<tr><td>14</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/notifications/notifier.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/storage/requests_test.go&#34;
}</code></pre></td></tr>
<tr><td>15</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/gateway/handler.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/review/review.go&#34;
}</code></pre></td></tr>
<tr><td>16</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/gateway/handler_test.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/review/review_test.go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/response/audit.go&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/response/response_test.go&#34;
}</code></pre></td></tr>
<tr><td>18</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls docs/ &amp;&amp; cat go.mod &amp;&amp; cat docs/* 2&gt;/dev/null | head -50&#34;,
  &#34;description&#34;: &#34;Inspect docs and go.mod&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/response/response.go&#34;
}</code></pre></td></tr>
<tr><td>19</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request.go&#34;,
  &#34;old_str&#34;: &#34;type Request struct {\n\tID      string\n\tKind    string\n\tPayload []byte\n}\n\nfunc (r Request) Validate() error {\n\tif r.ID == \&#34;\&#34; {\n\t\treturn errors.New(\&#34;invalid request\&#34;)\n\t}\n\treturn nil\n}&#34;,
  &#34;new_str&#34;: &#34;type Request struct {\n\tRequestID string\n\tKind      string\n\tPayload   []byte\n}\n\nvar ErrInvalidRequest = errors.New(\&#34;invalid request\&#34;)\n\nfunc (r Request) Validate() error {\n\tif r.RequestID == \&#34;\&#34; {\n\t\treturn ErrInvalidRequest\n\t}\n\treturn nil\n}&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;\\.ID\\b&#34;,
  &#34;paths&#34;: &#34;.&#34;,
  &#34;output_mode&#34;: &#34;content&#34;,
  &#34;-n&#34;: true,
  &#34;head_limit&#34;: 100
}</code></pre></td></tr>
<tr><td>20</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request.go&#34;,
  &#34;old_str&#34;: &#34;func Normalize(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;,
  &#34;new_str&#34;: &#34;func Normalize(kind string) string {\n\treturn normalizeKind(kind)\n}\n\nfunc normalizeKind(kind string) string {\n\tkind = strings.TrimSpace(kind)\n\tkind = strings.ToLower(kind)\n\treturn kind\n}&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;find . -name \&#34;*.go\&#34; -not -path \&#34;./.git/*\&#34; | sort&#34;,
  &#34;description&#34;: &#34;List all Go files in the workspace&#34;
}</code></pre></td></tr>
<tr><td>21</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/request/request.go&#34;,
  &#34;old_str&#34;: &#34;\t\tif r.ID == id {&#34;,
  &#34;new_str&#34;: &#34;\t\tif r.RequestID == id {&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/request/request_test.go&#34;
}</code></pre></td></tr>
<tr><td>22</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/correlation/identities.go&#34;,
  &#34;old_str&#34;: &#34;\treturn []string{r.ID, resp.ID, review.ID}&#34;,
  &#34;new_str&#34;: &#34;\treturn []string{r.RequestID, resp.ID, review.ID}&#34;
}</code></pre></td><td><code>view</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_semedit_small-implicit_task-12-request-identity-migration_1792297096547717000/response/audit.go&#34;
}</code></pre></td></tr>
<tr><td>23</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/storage/requests.go&#34;,
  &#34;old_str&#34;: &#34;\t\tif record.ID == r.ID {&#34;,
  &#34;new_str&#34;: &#34;\t\tif record.ID == r.RequestID {&#34;
}</code></pre></td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;symbol&#34;: &#34;Request.ID&#34;,
  &#34;to&#34;: &#34;RequestID&#34;,
  &#34;file&#34;: &#34;request/request.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>24</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/scheduler/jobs.go&#34;,
  &#34;old_str&#34;: &#34;\treturn left.Request.ID == right.Request.ID&#34;,
  &#34;new_str&#34;: &#34;\treturn left.Request.RequestID == right.Request.RequestID&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>25</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/scheduler/jobs.go&#34;,
  &#34;old_str&#34;: &#34;\t\tif job.Request.ID == r.ID &amp;&amp; job.Attempts == 0 { pending = append(pending, job) }&#34;,
  &#34;new_str&#34;: &#34;\t\tif job.Request.RequestID == r.RequestID &amp;&amp; job.Attempts == 0 { pending = append(pending, job) }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>26</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/reporting/report.go&#34;,
  &#34;old_str&#34;: &#34;\tfor _, r := range rows { grouped[r.Request.ID] = append(grouped[r.Request.ID], r) }&#34;,
  &#34;new_str&#34;: &#34;\tfor _, r := range rows { grouped[r.Request.RequestID] = append(grouped[r.Request.RequestID], r) }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>27</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/queue/queue.go&#34;,
  &#34;old_str&#34;: &#34;\t\tif entry.Key == r.ID { return true }&#34;,
  &#34;new_str&#34;: &#34;\t\tif entry.Key == r.RequestID { return true }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>28</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/queue/queue.go&#34;,
  &#34;old_str&#34;: &#34;\tfor _, r := range requests { keys = append(keys, r.ID) }&#34;,
  &#34;new_str&#34;: &#34;\tfor _, r := range requests { keys = append(keys, r.RequestID) }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>29</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/notifications/notifier.go&#34;,
  &#34;old_str&#34;: &#34;\tif r.ID == \&#34;\&#34; { return \&#34;unassigned\&#34; }\n\treturn \&#34;request:\&#34; + r.ID&#34;,
  &#34;new_str&#34;: &#34;\tif r.RequestID == \&#34;\&#34; { return \&#34;unassigned\&#34; }\n\treturn \&#34;request:\&#34; + r.RequestID&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>30</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/notifications/notifier.go&#34;,
  &#34;old_str&#34;: &#34;\tfor _, r := range notices { ids = append(ids, r.Request.ID) }&#34;,
  &#34;new_str&#34;: &#34;\tfor _, r := range notices { ids = append(ids, r.Request.RequestID) }&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>31</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Volumes/semedit-bench-ram/run_copilot_baseline-diff_small_task-12-request-identity-migration_1792296795990057000/gateway/handler.go&#34;,
  &#34;old_str&#34;: &#34;\t\treturn response.Response{}, fmt.Errorf(\&#34;submit %s: %w\&#34;, r.ID, err)&#34;,
  &#34;new_str&#34;: &#34;\t\treturn response.Response{}, fmt.Errorf(\&#34;submit %s: %w\&#34;, r.RequestID, err)&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>32</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;\\.ID\\b&#34;,
  &#34;paths&#34;: &#34;.&#34;,
  &#34;output_mode&#34;: &#34;content&#34;,
  &#34;-n&#34;: true,
  &#34;glob&#34;: &#34;*.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>33</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./... &amp;&amp; go test ./...&#34;,
  &#34;description&#34;: &#34;Build and test all packages&#34;,
  &#34;initial_wait&#34;: 120
}</code></pre></td><td>—</td></tr>
<tr><td>34</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go env GOCACHE GOTOOLCHAIN GOFLAGS GOPATH 2&gt;&amp;1; which go; go version&#34;,
  &#34;description&#34;: &#34;Inspect Go environment&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task stopped because its execution timed out. For benchmark analysis only, do not use tools, edit files, or retry the task. In one sentence, say whether semedit&#39;s tools helped or hindered the work before the timeout, grounded in what actually happened. If you could not assess their value, say so.</pre>
<p><strong>Capture error:</strong> cannot continue failed Copilot process: wait for Copilot session/prompt: context deadline exceeded; Copilot stderr: Warning: COPILOT_CLI_DIST_DIR (/Users/alessandro/Library/Caches/copilot/pkg/darwin-arm64/1.0.88) does not contain a readable index.js; ignoring</p>
</details>

#### Standard vs Semedit in Small Context: Timeout Diagnostic Reflection

Best-effort analysis of a timed-out run, captured outside task measurements. Provider identity and continuation errors remain recorded separately.

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task stopped because its execution timed out. Do not use tools, edit files, or retry the task. Based only on the conversation so far, identify the last action underway, where time appears to have gone (model reasoning, tool wait, repeated verification or undo, exploration, or another cause), and the strongest evidence for that diagnosis. Clearly separate observed evidence from inference, and answer in a few concise sentences.</pre>
<p><strong>Capture error:</strong> cannot continue failed Copilot process: wait for Copilot session/prompt: context deadline exceeded; Copilot stderr: Warning: COPILOT_CLI_DIST_DIR (/Users/alessandro/Library/Caches/copilot/pkg/darwin-arm64/1.0.88) does not contain a readable index.js; ignoring</p>
</details>

