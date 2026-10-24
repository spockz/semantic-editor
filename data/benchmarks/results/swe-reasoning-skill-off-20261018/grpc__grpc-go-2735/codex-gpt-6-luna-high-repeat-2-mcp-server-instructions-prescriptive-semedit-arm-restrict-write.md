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

* **Date**: 2026-10-18 07:32:19 CEST

## Test case: `grpc__grpc-go-2735`

### Target: `codex/gpt-6-luna/high (repeat 2)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline`

* **Fixture**: [testdata/bench/grpc__grpc-go-2735.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/grpc__grpc-go-2735.txtar)

**Vanilla LLM Prompt**:
> ### Use case(s) - what problem will this feature solve?
> 
> In some cases we want to retry connections more aggressively compared to the default behavior (e.g. for health checkers in tests in order to detect that server is up faster).
> 
> ### Proposed Solution
> 
> Add tunable parameters as described in the [documentation](https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md).
> 
> It can be done by extending `grpc.BackoffConfig` struct (and therefore is not a breaking change).
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 0.00s | 0.00s | — | — | — | — |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 0 | 0 | 0% | — | — | — |
| **Internal Tool Cycles** | 0 | 0 | 0% | — | — | — |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 0 | 0 | 0% | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | — | — | — | — | — | — |
| **Output Tokens** | 0 | 0 | 0% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Uncached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached vs Uncached Token Ratio** | — | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline`

* **Fixture**: [testdata/bench/grpc__grpc-go-2735.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/grpc__grpc-go-2735.txtar)

**Vanilla LLM Prompt**:
> ### Use case(s) - what problem will this feature solve?
> 
> In some cases we want to retry connections more aggressively compared to the default behavior (e.g. for health checkers in tests in order to detect that server is up faster).
> 
> ### Proposed Solution
> 
> Add tunable parameters as described in the [documentation](https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md).
> 
> It can be done by extending `grpc.BackoffConfig` struct (and therefore is not a breaking change).
>

**Semedit MCP Prompt**:
> ### Use case(s) - what problem will this feature solve?
> 
> In some cases we want to retry connections more aggressively compared to the default behavior (e.g. for health checkers in tests in order to detect that server is up faster).
> 
> ### Proposed Solution
> 
> Add tunable parameters as described in the [documentation](https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md).
> 
> It can be done by extending `grpc.BackoffConfig` struct (and therefore is not a breaking change). Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

| Metric | Vanilla (Small) | Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 0.00s | 184.85s | — | — | — | — |
| **Process Start → First Event** | — | 0.07s | — | — | — | — |
| **First Event → First Tool Call** | — | 4.33s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 0 | 1 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Internal Tool Cycles** | 0 | 37 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 0 | 3 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 0 | 37 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | — | $0.044187 | — | — | — | — |
| **Output Tokens** | 0 | 14394 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 8586 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Total Input Tokens** | 0 | 2806493 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Cached Input Tokens** | 0 | 2707328 | <span class="benchmark-delta-positive">+100.0%</span> | — | — | — |
| **Uncached Input Tokens** | 0 | 99165 | <span class="benchmark-delta-negative">+100.0%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | — | 27.30:1 | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*backoff*&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;doc/connection-backoff.md&#39; -g &#39;backoff.go&#39; | head -50 &amp;&amp; \
sed -n &#39;1,240p&#39; backoff.go&#34;</code></pre></td></tr>
<tr><td>3</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;(&#34;&#39;^|/)(AGENTS&#39;&#34;\\.md|PROJECT_BRIEF\\.md|connection-backoff\\.md)&#34;&#39;$&#39;&#34;&#39; | head -40 &amp;&amp; \
rg -n &#39;BackoffConfig|DefaultBackoffConfig|InitialBackoff|Multiplier|Jitter&#39; --glob &#39;*.go&#39; | head -180&#34;</code></pre></td></tr>
<tr><td>4</td><td>none recorded</td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td>none recorded</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;backoff.go&#34;,
  &#34;include_tests&#34;: false
}</code></pre></td></tr>
<tr><td>6</td><td>none recorded</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;backoff.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td>none recorded</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;backoff.go&#34;,
  &#34;symbol&#34;: &#34;BackoffConfig&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>none recorded</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
  &#34;symbol&#34;: &#34;Exponential&#34;
}</code></pre></td></tr>
<tr><td>9</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;220,280p&#39; dialoptions.go &amp;&amp; \
sed -n &#39;215,250p&#39; clientconn.go &amp;&amp; \
sed -n &#39;1,220p&#39; internal/backoff/backoff.go &amp;&amp; \
rg -n &#39;backoffStrategy|backoffTimeout|MaxDelay|connectionBackoff&#39; clientconn.go&#34;</code></pre></td></tr>
<tr><td>10</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;Backoff\\(|MinConnectTimeout|connectTimeout|ConnectTimeout|connectionTimeout&#39; --glob &#39;*.go&#39; | head -120 &amp;&amp; \
sed -n &#39;1,180p&#39; clientconn.go &amp;&amp; \
rg -n &#39;connectRetryNum|connectDeadline|transport.Dial&#39; clientconn.go&#34;</code></pre></td></tr>
<tr><td>11</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;40,90p&#39; dialoptions.go &amp;&amp; \
sed -n &#39;500,545p&#39; dialoptions.go &amp;&amp; \
sed -n &#39;935,985p&#39; clientconn.go &amp;&amp; \
sed -n &#39;630,725p&#39; clientconn_test.go &amp;&amp; \
sed -n &#39;810,850p&#39; clientconn_test.go&#34;</code></pre></td></tr>
<tr><td>12</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;backoff.go&#34;,
  &#34;kind&#34;: &#34;type&#34;,
  &#34;symbol&#34;: &#34;BackoffConfig&#34;,
  &#34;source&#34;: &#34;type BackoffConfig struct {\n\t// BaseDelay is the amount of time to wait before retrying after the first failure.\n\tBaseDelay time.Duration\n\t// Multiplier is the factor applied to the backoff after each failed retry.\n\tMultiplier float64\n\t// Jitter is the fraction of the backoff delay to randomize.\n\tJitter float64\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n\t// MinConnectTimeout is the minimum time to give a connection attempt to complete.\n\tMinConnectTimeout time.Duration\n}&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
  &#34;kind&#34;: &#34;type&#34;,
  &#34;symbol&#34;: &#34;Exponential&#34;,
  &#34;source&#34;: &#34;type Exponential struct {\n\t// BaseDelay is the amount of time to wait before retrying after the first failure.\n\tBaseDelay time.Duration\n\t// Multiplier is the factor applied to the backoff after each failed retry.\n\tMultiplier float64\n\t// Jitter is the fraction of the backoff delay to randomize.\n\tJitter float64\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n}&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Exponential.Backoff&#34;,
  &#34;source&#34;: &#34;func (bc Exponential) Backoff(retries int) time.Duration {\n\tbase := bc.BaseDelay\n\tif base &lt;= 0 {\n\t\tbase = baseDelay\n\t}\n\tif retries == 0 {\n\t\treturn base\n\t}\n\tmultiplier := bc.Multiplier\n\tif multiplier &lt;= 0 {\n\t\tmultiplier = factor\n\t}\n\tjitter := bc.Jitter\n\tif jitter &lt;= 0 {\n\t\tjitter = 0.2\n\t}\n\tbackoff, max := float64(base), float64(bc.MaxDelay)\n\tfor backoff &lt; max &amp;&amp; retries &gt; 0 {\n\t\tbackoff *= multiplier\n\t\tretries--\n\t}\n\tif backoff &gt; max {\n\t\tbackoff = max\n\t}\n\t// Randomize backoff delays so that if a cluster of requests start at\n\t// the same time, they won&#39;t operate in lockstep.\n\tbackoff *= 1 + jitter*(grpcrand.Float64()*2-1)\n\tif backoff &lt; 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;24,75p&#39; backoff.go&#34;</code></pre></td></tr>
<tr><td>16</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;backoff.go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;symbol&#34;: &#34;DefaultBackoffConfig&#34;,
  &#34;source&#34;: &#34;var DefaultBackoffConfig = BackoffConfig{\n\tBaseDelay:        time.Second,\n\tMultiplier:       1.6,\n\tJitter:           0.2,\n\tMaxDelay:         120 * time.Second,\n\tMinConnectTimeout: 20 * time.Second,\n}&#34;
}</code></pre></td></tr>
<tr><td>17</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;dialoptions.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;WithBackoffMaxDelay&#34;,
  &#34;source&#34;: &#34;func WithBackoffMaxDelay(md time.Duration) DialOption {\n\tconfig := DefaultBackoffConfig\n\tconfig.MaxDelay = md\n\treturn WithBackoffConfig(config)\n}&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;dialoptions.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;WithBackoffConfig&#34;,
  &#34;source&#34;: &#34;func WithBackoffConfig(b BackoffConfig) DialOption {\n\tminConnectTimeout := b.MinConnectTimeout\n\tif minConnectTimeout &lt;= 0 {\n\t\tminConnectTimeout = DefaultBackoffConfig.MinConnectTimeout\n\t}\n\treturn newFuncDialOption(func(o *dialOptions) {\n\t\to.bs = backoff.Exponential{\n\t\t\tBaseDelay:  b.BaseDelay,\n\t\t\tMultiplier: b.Multiplier,\n\t\t\tJitter:     b.Jitter,\n\t\t\tMaxDelay:   b.MaxDelay,\n\t\t}\n\t\to.minConnectTimeout = func() time.Duration {\n\t\t\treturn minConnectTimeout\n\t\t}\n\t})\n}&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;820,875p&#39; clientconn_test.go&#34;</code></pre></td></tr>
<tr><td>20</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;not_applied&#34;,&#34;error&#34;:&#34;if \&#34;if cc.dopts.bs == nil\&#34; not found in DialContext; available candidates (select an if by its condition or short initializer, or set construct_path):\nconstruct_path 5 at /Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792301345050558000/clientconn.go:146:2: if channelz.IsOn()\nconstruct_path 5.0.0 at /Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_17923013450<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn.go&#34;,
  &#34;kind&#34;: &#34;if&#34;,
  &#34;in_function&#34;: &#34;DialContext&#34;,
  &#34;discriminator&#34;: &#34;if cc.dopts.bs == nil&#34;,
  &#34;source&#34;: &#34;if cc.dopts.bs == nil {\n\t\tcc.dopts.bs = backoff.Exponential{\n\t\t\tBaseDelay:  DefaultBackoffConfig.BaseDelay,\n\t\t\tMultiplier: DefaultBackoffConfig.Multiplier,\n\t\t\tJitter:     DefaultBackoffConfig.Jitter,\n\t\t\tMaxDelay:   DefaultBackoffConfig.MaxDelay,\n\t\t}\n\t}\n\tif cc.dopts.minConnectTimeout == nil {\n\t\tcc.dopts.minConnectTimeout = func() time.Duration {\n\t\t\treturn DefaultBackoffConfig.MinConnectTimeout\n\t\t}\n\t}&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;not_applied&#34;,&#34;error&#34;:&#34;selection revision mismatch\nfile: clientconn.go\nin_function: DialContext\nkind: if\nCurrent replacement refused before writing for clientconn.go. The supplied expected_selection_revision differs from the token recomputed from the current ordered candidate list of paths and normalized constructs. Reselect from the current choices and token for this request target without another lookup while the buffer is current. Do not change source to r<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn.go&#34;,
  &#34;kind&#34;: &#34;if&#34;,
  &#34;in_function&#34;: &#34;DialContext&#34;,
  &#34;construct_path&#34;: &#34;15&#34;,
  &#34;expected_selection_revision&#34;: &#34;184fa68da5334adb6aaf07a82d3b9923606ee54c87207c9ca975ec33f243f805&#34;,
  &#34;source&#34;: &#34;if cc.dopts.bs == nil {\n\t\tcc.dopts.bs = backoff.Exponential{\n\t\t\tBaseDelay:  DefaultBackoffConfig.BaseDelay,\n\t\t\tMultiplier: DefaultBackoffConfig.Multiplier,\n\t\t\tJitter:     DefaultBackoffConfig.Jitter,\n\t\t\tMaxDelay:   DefaultBackoffConfig.MaxDelay,\n\t\t}\n\t}&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn.go&#34;,
  &#34;kind&#34;: &#34;if&#34;,
  &#34;in_function&#34;: &#34;DialContext&#34;,
  &#34;construct_path&#34;: &#34;15&#34;,
  &#34;expected_selection_revision&#34;: &#34;52bece0eec22fa87ff35bcbf3eb2085df3226022d4de56a3a3dae9453309b094&#34;,
  &#34;source&#34;: &#34;if cc.dopts.bs == nil {\n\t\tcc.dopts.bs = backoff.Exponential{\n\t\t\tBaseDelay:  DefaultBackoffConfig.BaseDelay,\n\t\t\tMultiplier: DefaultBackoffConfig.Multiplier,\n\t\t\tJitter:     DefaultBackoffConfig.Jitter,\n\t\t\tMaxDelay:   DefaultBackoffConfig.MaxDelay,\n\t\t}\n\t}&#34;
}</code></pre></td></tr>
<tr><td>23</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn_test.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;TestWithBackoffMaxDelay&#34;,
  &#34;source&#34;: &#34;func (s) TestWithBackoffMaxDelay(t *testing.T) {\n\tmd := DefaultBackoffConfig.MaxDelay / 2\n\texpected := DefaultBackoffConfig\n\texpected.MaxDelay = md\n\ttestBackoffConfigSet(t, &amp;expected, WithBackoffMaxDelay(md))\n}&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn_test.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;testBackoffConfigSet&#34;,
  &#34;source&#34;: &#34;func testBackoffConfigSet(t *testing.T, expected *BackoffConfig, opts ...DialOption) {\n\topts = append(opts, WithInsecure())\n\tconn, err := Dial(\&#34;passthrough:///foo:80\&#34;, opts...)\n\tif err != nil {\n\t\tt.Fatalf(\&#34;unexpected error dialing connection: %v\&#34;, err)\n\t}\n\tdefer conn.Close()\n\n\tif conn.dopts.bs == nil {\n\t\tt.Fatalf(\&#34;backoff config not set\&#34;)\n\t}\n\n\tactual, ok := conn.dopts.bs.(backoff.Exponential)\n\tif !ok {\n\t\tt.Fatalf(\&#34;unexpected type of backoff config: %#v\&#34;, conn.dopts.bs)\n\t}\n\n\texpectedValue := backoff.Exponential{\n\t\tBaseDelay:  expected.BaseDelay,\n\t\tMultiplier: expected.Multiplier,\n\t\tJitter:     expected.Jitter,\n\t\tMaxDelay:   expected.MaxDelay,\n\t}\n\tif actual != expectedValue {\n\t\tt.Fatalf(\&#34;unexpected backoff config on connection: %v, want %v\&#34;, actual, expected)\n\t}\n\n\texpectedMinConnectTimeout := expected.MinConnectTimeout\n\tif expectedMinConnectTimeout &lt;= 0 {\n\t\texpectedMinConnectTimeout = DefaultBackoffConfig.MinConnectTimeout\n\t}\n\tif got := conn.dopts.minConnectTimeout(); got != expectedMinConnectTimeout {\n\t\tt.Fatalf(\&#34;unexpected minimum connect timeout: %v, want %v\&#34;, got, expectedMinConnectTimeout)\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn_test.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;TestWithBackoffConfig&#34;,
  &#34;source&#34;: &#34;func (s) TestWithBackoffConfig(t *testing.T) {\n\tpartial := BackoffConfig{MaxDelay: DefaultBackoffConfig.MaxDelay / 2}\n\ttestBackoffConfigSet(t, &amp;partial, WithBackoffConfig(partial))\n\n\tconfig := BackoffConfig{\n\t\tBaseDelay:        500 * time.Millisecond,\n\t\tMultiplier:       1.2,\n\t\tJitter:            0.1,\n\t\tMaxDelay:         DefaultBackoffConfig.MaxDelay / 2,\n\t\tMinConnectTimeout: 5 * time.Second,\n\t}\n\ttestBackoffConfigSet(t, &amp;config, WithBackoffConfig(config))\n}&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Exponential.Backoff&#34;,
  &#34;source&#34;: &#34;func (bc Exponential) Backoff(retries int) time.Duration {\n\tbase := bc.BaseDelay\n\tif base &lt;= 0 {\n\t\tbase = baseDelay\n\t}\n\tif retries == 0 {\n\t\treturn base\n\t}\n\tmultiplier := bc.Multiplier\n\tif multiplier &lt;= 0 {\n\t\tmultiplier = factor\n\t}\n\tjitterFactor := bc.Jitter\n\tif jitterFactor &lt;= 0 {\n\t\tjitterFactor = jitter\n\t}\n\tbackoff, max := float64(base), float64(bc.MaxDelay)\n\tfor backoff &lt; max &amp;&amp; retries &gt; 0 {\n\t\tbackoff *= multiplier\n\t\tretries--\n\t}\n\tif backoff &gt; max {\n\t\tbackoff = max\n\t}\n\t// Randomize backoff delays so that if a cluster of requests start at\n\t// the same time, they won&#39;t operate in lockstep.\n\tbackoff *= 1 + jitterFactor*(grpcrand.Float64()*2-1)\n\tif backoff &lt; 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;
}</code></pre></td></tr>
<tr><td>27</td><td>none recorded</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;dialoptions.go&#34;,
  &#34;symbol&#34;: &#34;WithBackoffConfig&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>none recorded</td><td><code>semedit/semantic_set_doc_comment</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;dialoptions.go&#34;,
  &#34;scope&#34;: &#34;declaration&#34;,
  &#34;symbol&#34;: &#34;WithBackoffConfig&#34;,
  &#34;expected_revision&#34;: &#34;61dbd35fdddf5ccbf25fdbbb6783ea44802fc51e937ff4f4ee3549834b90e421&#34;,
  &#34;comment&#34;: &#34;WithBackoffConfig configures the dialer to use the provided backoff parameters after connection failures. Zero values for BaseDelay, Multiplier, Jitter, and MinConnectTimeout use their defaults.&#34;
}</code></pre></td></tr>
<tr><td>29</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff -- backoff.go dialoptions.go clientconn.go internal/backoff/backoff.go clientconn_test.go&#39;</code></pre></td></tr>
<tr><td>30</td><td>none recorded</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;construct_path requires expected_revision: invalid operation parameters&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn.go&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;in_function&#34;: &#34;DialContext&#34;,
  &#34;construct_path&#34;: &#34;15&#34;,
  &#34;anchor&#34;: &#34;after&#34;,
  &#34;source&#34;: &#34;if cc.dopts.minConnectTimeout == nil {\n\t\tcc.dopts.minConnectTimeout = func() time.Duration {\n\t\t\treturn DefaultBackoffConfig.MinConnectTimeout\n\t\t}\n\t}&#34;
}</code></pre></td></tr>
<tr><td>31</td><td>none recorded</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn.go&#34;,
  &#34;symbol&#34;: &#34;DialContext&#34;
}</code></pre></td></tr>
<tr><td>32</td><td>none recorded</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;plan statement insertion: discriminator is required for after insertion&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn.go&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;in_function&#34;: &#34;DialContext&#34;,
  &#34;construct_path&#34;: &#34;15&#34;,
  &#34;anchor&#34;: &#34;after&#34;,
  &#34;expected_revision&#34;: &#34;c57b25c879323395f6e61287d13a2324e57f784692db6a6010840adf7e589b84&#34;,
  &#34;source&#34;: &#34;if cc.dopts.minConnectTimeout == nil {\n\t\tcc.dopts.minConnectTimeout = func() time.Duration {\n\t\t\treturn DefaultBackoffConfig.MinConnectTimeout\n\t\t}\n\t}&#34;
}</code></pre></td></tr>
<tr><td>33</td><td>none recorded</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;plan statement insertion: statement anchor \&#34;if cc.dopts.bs == nil\&#34; not found in \&#34;DialContext\&#34;&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn.go&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;in_function&#34;: &#34;DialContext&#34;,
  &#34;anchor&#34;: &#34;after&#34;,
  &#34;discriminator&#34;: &#34;if cc.dopts.bs == nil&#34;,
  &#34;expected_revision&#34;: &#34;c57b25c879323395f6e61287d13a2324e57f784692db6a6010840adf7e589b84&#34;,
  &#34;source&#34;: &#34;if cc.dopts.minConnectTimeout == nil {\n\t\tcc.dopts.minConnectTimeout = func() time.Duration {\n\t\t\treturn DefaultBackoffConfig.MinConnectTimeout\n\t\t}\n\t}&#34;
}</code></pre></td></tr>
<tr><td>34</td><td>none recorded</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn_test.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;testBackoffConfigSet&#34;,
  &#34;source&#34;: &#34;func testBackoffConfigSet(t *testing.T, expected *BackoffConfig, opts ...DialOption) {\n\topts = append(opts, WithInsecure())\n\tconn, err := Dial(\&#34;passthrough:///foo:80\&#34;, opts...)\n\tif err != nil {\n\t\tt.Fatalf(\&#34;unexpected error dialing connection: %v\&#34;, err)\n\t}\n\tdefer conn.Close()\n\n\tif conn.dopts.bs == nil {\n\t\tt.Fatalf(\&#34;backoff config not set\&#34;)\n\t}\n\n\tactual, ok := conn.dopts.bs.(backoff.Exponential)\n\tif !ok {\n\t\tt.Fatalf(\&#34;unexpected type of backoff config: %#v\&#34;, conn.dopts.bs)\n\t}\n\n\texpectedValue := backoff.Exponential{\n\t\tBaseDelay:  expected.BaseDelay,\n\t\tMultiplier: expected.Multiplier,\n\t\tJitter:     expected.Jitter,\n\t\tMaxDelay:   expected.MaxDelay,\n\t}\n\tif actual != expectedValue {\n\t\tt.Fatalf(\&#34;unexpected backoff config on connection: %v, want %v\&#34;, actual, expected)\n\t}\n\n\texpectedMinConnectTimeout := expected.MinConnectTimeout\n\tif expectedMinConnectTimeout &lt;= 0 {\n\t\texpectedMinConnectTimeout = DefaultBackoffConfig.MinConnectTimeout\n\t}\n\tgotMinConnectTimeout := minConnectTimeout\n\tif conn.dopts.minConnectTimeout != nil {\n\t\tgotMinConnectTimeout = conn.dopts.minConnectTimeout()\n\t}\n\tif gotMinConnectTimeout != expectedMinConnectTimeout {\n\t\tt.Fatalf(\&#34;unexpected minimum connect timeout: %v, want %v\&#34;, gotMinConnectTimeout, expectedMinConnectTimeout)\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>35</td><td>none recorded</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/trace/tracev2 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/msan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; do<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test . ./internal/backoff&#39;</code></pre></td></tr>
<tr><td>36</td><td>none recorded</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go version &amp;&amp; \
command -v go &amp;&amp; \
go env GOROOT GOCACHE &amp;&amp; \
rg --files .scratch | head -40 &amp;&amp; \
git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>37</td><td>none recorded</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: go version go1.27.2 darwin/arm64 E1018 07:32:11.238565 22162 pickfirst.go:61] pickfirstBalancer: failed to NewSubConn: grpc: cannot create SubConn with empty address list E1018 07:32:12.359209 22162 pickfirst.go:61] pickfirstBalancer: failed to NewSubConn: grpc: cannot create SubConn with empty address list --- FAIL: Test (4.23s) --- FAIL: Test/AddressesRemovedPickfirst (0.00s) call_test.go:187: failed to listen: listen tcp 127.0.0.1:0: bind: operation not permitted --- FAIL: Test/AllServersDownPickfirst (0<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;/Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/bin/go version &amp;&amp; \
PATH=&#34;/Users/alessandro/sources/semantic-editor/.scratch/go/mod/golang.org/toolchain@v0.0.1-go1.27.2.darwin-arm64/bin:$PATH&#34; go test . ./internal/backoff&#39;</code></pre></td></tr>
</tbody>
</table>

