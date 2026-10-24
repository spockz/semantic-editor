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

* **Date**: 2026-10-17 21:43:40 CEST

## Test case: `task-09-composite-refactor`

### Target: `agy/gemini-4-argon-low (repeat 3)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=agy_stream_json_v1`

* **Fixture**: [testdata/bench/task_09_composite_refactor.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_09_composite_refactor.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert DefaultConfig() *Config returning Host localhost and ListenPort 8080, and clean up unused imports. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package config

import (
	"archive/tar"
	"fmt"
)

type Config struct {
	Host string
	Port int
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
```
</details>

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 30.64s | 140.28s | <span class="benchmark-delta-negative">+357.8%</span> | — | — | — |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 5 | 9 | <span class="benchmark-delta-negative">+80.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 5 | <span class="benchmark-delta-negative">+66.7%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 5 | 13 | <span class="benchmark-delta-negative">+160.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | — | — | — | — | — | — |
| **Output Tokens** | 767 | 1194 | <span class="benchmark-delta-negative">+55.7%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 24849 | 38950 | <span class="benchmark-delta-negative">+56.7%</span> | — | — | — |
| **Cached Input Tokens** | 53406 | 160808 | <span class="benchmark-delta-positive">+201.1%</span> | — | — | — |
| **Uncached Input Tokens** | 24849 | 38950 | <span class="benchmark-delta-negative">+56.7%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 2.15:1 | 4.13:1 | <span class="benchmark-delta-positive">+92.1%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 16 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000/config/config.go\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing config/config.go\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View config/config.go\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_lookup.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Reading semantic_lookup schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Read semantic_lookup schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>run_command</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;CommandLine&#34;: &#34;\&#34;rg \\\&#34;Port\\\&#34;\&#34;&#34;,
  &#34;Cwd&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000\&#34;&#34;,
  &#34;WaitMsBeforeAsync&#34;: &#34;5000&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Searching for Port references\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Search for Port references\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_inspect_symbol.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Reading semantic_inspect_symbol schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Read semantic_inspect_symbol schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000/config/config_test.go\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing config/config_test.go\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View config/config_test.go\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_rename.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Reading semantic_rename schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Read semantic_rename schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>replace_file_content</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AllowMultiple&#34;: &#34;false&#34;,
  &#34;Description&#34;: &#34;\&#34;Rename Config.Port to Config.ListenPort, add DefaultConfig(), and remove unused import in config/config.go\&#34;&#34;,
  &#34;EndLine&#34;: &#34;17&#34;,
  &#34;Instruction&#34;: &#34;\&#34;Update Config struct, Addr method, imports, and add DefaultConfig function\&#34;&#34;,
  &#34;ReplacementContent&#34;: &#34;\&#34;import (\\n\\t\\\&#34;fmt\\\&#34;\\n)\\n\\ntype Config struct {\\n\\tHost       string\\n\\tListenPort int\\n}\\n\\nfunc DefaultConfig() *Config {\\n\\treturn \u0026Config{\\n\\t\\tHost:       \\\&#34;localhost\\\&#34;,\\n\\t\\tListenPort: 8080,\\n\\t}\\n}\\n\\nfunc (c *Config) Addr() string {\\n\\treturn fmt.Sprintf(\\\&#34;%s:%d\\\&#34;, c.Host, c.ListenPort)\\n}\&#34;&#34;,
  &#34;StartLine&#34;: &#34;3&#34;,
  &#34;TargetContent&#34;: &#34;\&#34;import (\\n\\t\\\&#34;archive/tar\\\&#34;\\n\\t\\\&#34;fmt\\\&#34;\\n)\\n\\ntype Config struct {\\n\\tHost string\\n\\tPort int\\n}\\n\\nfunc (c *Config) Addr() string {\\n\\treturn fmt.Sprintf(\\\&#34;%s:%d\\\&#34;, c.Host, c.Port)\\n}\&#34;&#34;,
  &#34;TargetFile&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000/config/config.go\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Updating config/config.go\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Update config/config.go\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_insert_construct.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Reading semantic_insert_construct schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Read semantic_insert_construct schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>run_command</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;CommandLine&#34;: &#34;\&#34;go test ./...\&#34;&#34;,
  &#34;Cwd&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000\&#34;&#34;,
  &#34;WaitMsBeforeAsync&#34;: &#34;5000&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Running tests\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Run go test ./...\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_batch.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Reading semantic_batch schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Read semantic_batch schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>semedit/semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;symbol\&#34;:\&#34;Config.Port\&#34;}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_lookup\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Looking up Config.Port\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Lookup Config.Port in config/config.go\&#34;&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;symbol\&#34;:\&#34;Config.Addr\&#34;}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_inspect_symbol\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Inspecting Config.Addr\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Inspect Config.Addr in config/config.go\&#34;&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Created At: 2026-10-17T16:04:29+02:00 Completed At: 2026-10-17T16:04:31+02:00 Encountered error in step execution: Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Volumes/semedit-bench-ram/run_agy_semedit-explicit_small-explicit_task-09-composite-refactor_1792245838483331000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/92\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;edits\&#34;:[{\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;symbol\&#34;:\&#34;Config.Port\&#34;,\&#34;to\&#34;:\&#34;ListenPort\&#34;,\&#34;tool\&#34;:\&#34;semantic_rename\&#34;},{\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;placement\&#34;:\&#34;public_end\&#34;,\&#34;source\&#34;:\&#34;func DefaultConfig() *Config { return \u0026Config{Host: \\\&#34;localhost\\\&#34;, ListenPort: 8080} }\&#34;,\&#34;tool\&#34;:\&#34;semantic_insert_function\&#34;},{\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;tool\&#34;:\&#34;semantic_organize_imports\&#34;}]}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_batch\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Applying semantic refactoring batch\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Run semantic_batch for rename, insert_function, and organize_imports\&#34;&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;symbol\&#34;:\&#34;Config.Port\&#34;,\&#34;to\&#34;:\&#34;ListenPort\&#34;}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_rename\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Renaming Config.Port to ListenPort\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Rename Config.Port to ListenPort\&#34;&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;kind\&#34;:\&#34;function\&#34;,\&#34;placement\&#34;:\&#34;public_end\&#34;,\&#34;source\&#34;:\&#34;func DefaultConfig() *Config { return \u0026Config{Host: \\\&#34;localhost\\\&#34;, ListenPort: 8080} }\&#34;}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_insert_construct\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Inserting DefaultConfig function\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Insert DefaultConfig in config/config.go\&#34;&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Created At: 2026-10-17T16:05:10+02:00 Completed At: 2026-10-17T16:05:11+02:00 Encountered error in step execution: Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Volumes/semedit-bench-ram/run_agy_semedit-explicit_small-explicit_task-09-composite-refactor_1792245838483331000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/92\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;edits\&#34;:[{\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;tool\&#34;:\&#34;semantic_organize_imports\&#34;}]}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_batch\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Organising imports in config.go\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Organise imports in config/config.go\&#34;&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>semedit/semantic_organize_imports</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;file\&#34;:\&#34;config/config.go\&#34;}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_organize_imports\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Organising imports in config.go\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Call semantic_organize_imports on config/config.go\&#34;&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>run_command</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;CommandLine&#34;: &#34;\&#34;go test ./...\&#34;&#34;,
  &#34;Cwd&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_semedit-explicit_small-explicit_task-09-composite-refactor_1792245838483331000\&#34;&#34;,
  &#34;WaitMsBeforeAsync&#34;: &#34;5000&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Running Go test suite\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Run go test ./...\&#34;&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=agy_stream_json_v1`

* **Fixture**: [testdata/bench/task_09_composite_refactor.txtar](https://github.com/spockz/semantic-editor/blob/bfcdd5ea10ca60e6edb295d274e686d5ea68cb90/testdata/bench/task_09_composite_refactor.txtar)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert DefaultConfig() *Config returning Host localhost and ListenPort 8080, and clean up unused imports. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "go.sum".
>
> In config/config.go: rename Config.Port to Config.ListenPort, insert DefaultConfig() *Config returning Host localhost and ListenPort 8080, and clean up unused imports. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package config

import (
	"archive/tar"
	"fmt"
)

type Config struct {
	Host string
	Port int
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 30.64s | 75.51s | <span class="benchmark-delta-negative">+146.4%</span> | — | — | — |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 5 | 8 | <span class="benchmark-delta-negative">+60.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 7 | <span class="benchmark-delta-negative">+133.3%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 5 | 13 | <span class="benchmark-delta-negative">+160.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | — | — | — | — | — | — |
| **Output Tokens** | 767 | 1219 | <span class="benchmark-delta-negative">+58.9%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 24849 | 36585 | <span class="benchmark-delta-negative">+47.2%</span> | — | — | — |
| **Cached Input Tokens** | 53406 | 135096 | <span class="benchmark-delta-positive">+153.0%</span> | — | — | — |
| **Uncached Input Tokens** | 24849 | 36585 | <span class="benchmark-delta-negative">+47.2%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 2.15:1 | 3.69:1 | <span class="benchmark-delta-positive">+71.8%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context
* **Vanilla Edit**: File config/config.go modified (15 lines -> 21 lines)
* **MCP Edit**: File config/config.go modified (15 lines -> 21 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000/config/config.go\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing config/config.go\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View config/config.go\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_semedit_small-implicit_task-09-composite-refactor_1792246028774401000/config/config.go\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing config.go and semedit instructions\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View config.go and semedit instructions\&#34;&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>run_command</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;CommandLine&#34;: &#34;\&#34;rg \\\&#34;Port\\\&#34;\&#34;&#34;,
  &#34;Cwd&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000\&#34;&#34;,
  &#34;WaitMsBeforeAsync&#34;: &#34;5000&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Searching for Port references\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Search for Port references\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/instructions.md\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing semedit instructions\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View semedit instructions\&#34;&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000/config/config_test.go\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing config/config_test.go\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View config/config_test.go\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_rename.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing semantic_rename schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View semantic_rename schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>replace_file_content</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AllowMultiple&#34;: &#34;false&#34;,
  &#34;Description&#34;: &#34;\&#34;Rename Config.Port to Config.ListenPort, add DefaultConfig(), and remove unused import in config/config.go\&#34;&#34;,
  &#34;EndLine&#34;: &#34;17&#34;,
  &#34;Instruction&#34;: &#34;\&#34;Update Config struct, Addr method, imports, and add DefaultConfig function\&#34;&#34;,
  &#34;ReplacementContent&#34;: &#34;\&#34;import (\\n\\t\\\&#34;fmt\\\&#34;\\n)\\n\\ntype Config struct {\\n\\tHost       string\\n\\tListenPort int\\n}\\n\\nfunc DefaultConfig() *Config {\\n\\treturn \u0026Config{\\n\\t\\tHost:       \\\&#34;localhost\\\&#34;,\\n\\t\\tListenPort: 8080,\\n\\t}\\n}\\n\\nfunc (c *Config) Addr() string {\\n\\treturn fmt.Sprintf(\\\&#34;%s:%d\\\&#34;, c.Host, c.ListenPort)\\n}\&#34;&#34;,
  &#34;StartLine&#34;: &#34;3&#34;,
  &#34;TargetContent&#34;: &#34;\&#34;import (\\n\\t\\\&#34;archive/tar\\\&#34;\\n\\t\\\&#34;fmt\\\&#34;\\n)\\n\\ntype Config struct {\\n\\tHost string\\n\\tPort int\\n}\\n\\nfunc (c *Config) Addr() string {\\n\\treturn fmt.Sprintf(\\\&#34;%s:%d\\\&#34;, c.Host, c.Port)\\n}\&#34;&#34;,
  &#34;TargetFile&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000/config/config.go\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Updating config/config.go\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Update config/config.go\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_insert_construct.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing semantic_insert_construct schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View semantic_insert_construct schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>run_command</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;CommandLine&#34;: &#34;\&#34;go test ./...\&#34;&#34;,
  &#34;Cwd&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_baseline-diff_small_task-09-composite-refactor_1792245997627762000\&#34;&#34;,
  &#34;WaitMsBeforeAsync&#34;: &#34;5000&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Running tests\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Run go test ./...\&#34;&#34;
}</code></pre></td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_remove_construct.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing semantic_remove_construct schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View semantic_remove_construct schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>6</td><td>—</td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_workspace_outline.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing semantic_workspace_outline schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View semantic_workspace_outline schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>—</td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Users/alessandro/.gemini/antigravity-cli/mcp/semedit/semantic_batch.json\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing semantic_batch schema\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View semantic_batch schema\&#34;&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;include_tests\&#34;:true,\&#34;path\&#34;:\&#34;.\&#34;}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_workspace_outline\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Running semantic_workspace_outline\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Run semantic_workspace_outline\&#34;&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Created At: 2026-10-17T16:07:41+02:00 Completed At: 2026-10-17T16:07:42+02:00 Encountered error in step execution: Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Volumes/semedit-bench-ram/run_agy_semedit_small-implicit_task-09-composite-refactor_1792246028774401000\&#34;: check Go diagnostics with gopls: incomplete Go package coverage for \&#34;./.scratch/go/build/92\&#34;: package has errors but no selected Go files&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;auto_organize_imports\&#34;:true,\&#34;edits\&#34;:[{\&#34;auto_organize_imports\&#34;:true,\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;symbol\&#34;:\&#34;Config.Port\&#34;,\&#34;to\&#34;:\&#34;ListenPort\&#34;,\&#34;tool\&#34;:\&#34;semantic_rename\&#34;},{\&#34;auto_organize_imports\&#34;:true,\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;kind\&#34;:\&#34;function\&#34;,\&#34;source\&#34;:\&#34;func DefaultConfig() *Config {\\n\\treturn \u0026Config{\\n\\t\\tHost:       \\\&#34;localhost\\\&#34;,\\n\\t\\tListenPort: 8080,\\n\\t}\\n}\&#34;,\&#34;tool\&#34;:\&#34;semantic_insert_construct\&#34;}]}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_batch\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Executing semantic batch refactor\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Execute semantic batch refactor\&#34;&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;auto_organize_imports\&#34;:true,\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;kind\&#34;:\&#34;function\&#34;,\&#34;source\&#34;:\&#34;func DefaultConfig() *Config {\\n\\treturn \u0026Config{\\n\\t\\tHost: \\\&#34;localhost\\\&#34;,\\n\\t\\tPort: 8080,\\n\\t}\\n}\&#34;}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_insert_construct\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Inserting DefaultConfig function\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Insert DefaultConfig function\&#34;&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><code>semedit/semantic_rename</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;Arguments&#34;: &#34;{\&#34;auto_organize_imports\&#34;:true,\&#34;file\&#34;:\&#34;config/config.go\&#34;,\&#34;symbol\&#34;:\&#34;Config.Port\&#34;,\&#34;to\&#34;:\&#34;ListenPort\&#34;}&#34;,
  &#34;ServerName&#34;: &#34;\&#34;semedit\&#34;&#34;,
  &#34;ToolName&#34;: &#34;\&#34;semantic_rename\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Renaming Config.Port to Config.ListenPort\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Rename Config.Port to Config.ListenPort\&#34;&#34;
}</code></pre></td></tr>
<tr><td>12</td><td>—</td><td><code>view_file</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;AbsolutePath&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_semedit_small-implicit_task-09-composite-refactor_1792246028774401000/config/config.go\&#34;&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Viewing updated config.go\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;View updated config.go\&#34;&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>run_command</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;CommandLine&#34;: &#34;\&#34;go test ./...\&#34;&#34;,
  &#34;Cwd&#34;: &#34;\&#34;/Volumes/semedit-bench-ram/run_agy_semedit_small-implicit_task-09-composite-refactor_1792246028774401000\&#34;&#34;,
  &#34;WaitMsBeforeAsync&#34;: &#34;5000&#34;,
  &#34;toolAction&#34;: &#34;\&#34;Running tests\&#34;&#34;,
  &#34;toolSummary&#34;: &#34;\&#34;Run tests\&#34;&#34;
}</code></pre></td></tr>
</tbody>
</table>

#### Standard vs Semedit in Small Context: Semedit Integration Feedback

Agent opinion based on this run; excluded from benchmark measurements.

<blockquote>Although `semantic_batch` stumbled on pre-existing build cache directories in `.scratch`, dropping down to standalone `semantic_insert_construct` and `semantic_rename` felt wonderfully reassuring because they cleared unused imports, refactored field references across the package, and verified zero remaining diagnostics in two clean steps, making me eager to ask my owner to keep semedit&#39;s tools in my toolkit.</blockquote>

<details><summary>Session reflection</summary>

<p><strong>Prompt:</strong></p><pre>The benchmark task is complete. For benchmark analysis only, do not make further file changes, do not run tools, and do not retry the task. Write a one-sentence user-facing summary about the value that semedit&#39;s tools provide for coding, from your perspective as the coding AI agent that performed this task. Use a bit of emotion, grounded in your actual experience during this run, and make clear whether you would ask your owner to add semedit&#39;s tools or not. This sentence will be the first thing a potential new user sees about semedit&#39;s usefulness to an AI agent. Be candid: a negative or mixed recommendation is welcome; if you did not use the tools or could not assess them, say so and do not invent benefits.</pre>
</details>

