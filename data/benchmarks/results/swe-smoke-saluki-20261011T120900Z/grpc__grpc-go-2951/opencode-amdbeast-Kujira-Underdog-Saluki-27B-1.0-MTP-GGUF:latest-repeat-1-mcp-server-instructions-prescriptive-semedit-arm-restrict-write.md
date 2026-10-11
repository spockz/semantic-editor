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

* **Date**: 2026-10-11 15:11:43 CEST

## Test case: `grpc__grpc-go-2951`

### Target: `opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/grpc__grpc-go-2951.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/grpc__grpc-go-2951.txtar)

**Vanilla LLM Prompt**:
> If a resolver could not resolve a name, there's currently no way to report error back to ClientConn, and RPCs on the ClientConn will fail with timeout.
> 
> This method allows resolver to report error to ClientConn, and failfast RPCs should fail with this error (or RPC errors that contain info from this error).
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 299.79s | 299.80s | N/A | — | — | — |
| **Process Start → First Event** | 20.95s | 46.16s | — | — | — | — |
| **First Event → First Tool Call** | 5.44s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 17 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 17 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 17 | 0 | N/A | — | — | — |
| **Output Tokens** | 8730 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 371328 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 332717 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 38611 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.62:1 | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/resolver/resolver.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/resolver&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000 \u0026\u0026 git log --oneline -10 2\u003e/dev/null; git status 2\u003e/dev/null | head -20&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000 \u0026\u0026 cat go.mod | head -20 \u0026\u0026 cat version.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000&#34;,
  &#34;pattern&#34;: &#34;ReportError&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/grpc/grpc-go/pull/3334)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://github.com/grpc/grpc-go/pull/3334&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/grpc/grpc-go/v1.25.0/resolver/resolver.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/grpc/grpc-go/v1.25.0/resolver/resolver.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>9</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/clientconn.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>10</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/resolver_conn_wrapper.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>11</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000&#34;,
  &#34;pattern&#34;: &#34;firstResolveEvent&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>12</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000&#34;,
  &#34;pattern&#34;: &#34;waitForResolvedAddrs|toRPCErr|connectionError&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>13</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/stream.go&#34;,
  &#34;limit&#34;: 120,
  &#34;offset&#34;: 140
}</code></pre></td><td>none recorded</td></tr>
<tr><td>14</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/picker_wrapper.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>15</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/grpc/grpc-go/v1.25.0/clientconn.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/grpc/grpc-go/v1.25.0/clientconn.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>16</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls ~/go/pkg/mod/google.golang.org/ 2\u003e/dev/null; ls /Users/alessandro/sources 2\u003e/dev/null | head -30&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>17</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://proxy.golang.org/google.golang.org/grpc/@v/v1.25.0.zip)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://proxy.golang.org/google.golang.org/grpc/@v/v1.25.0.zip&#34;
}</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/grpc__grpc-go-2951.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/grpc__grpc-go-2951.txtar)

**Vanilla LLM Prompt**:
> If a resolver could not resolve a name, there's currently no way to report error back to ClientConn, and RPCs on the ClientConn will fail with timeout.
> 
> This method allows resolver to report error to ClientConn, and failfast RPCs should fail with this error (or RPC errors that contain info from this error).
>

**Semedit MCP Prompt**:
> If a resolver could not resolve a name, there's currently no way to report error back to ClientConn, and RPCs on the ClientConn will fail with timeout.
> 
> This method allows resolver to report error to ClientConn, and failfast RPCs should fail with this error (or RPC errors that contain info from this error). Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 299.79s | 299.79s | N/A | — | — | — |
| **Process Start → First Event** | 20.95s | 25.92s | — | — | — | — |
| **First Event → First Tool Call** | 5.44s | 1.23s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 17 | 23 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 17 | 23 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 17 | 23 | N/A | — | — | — |
| **Output Tokens** | 8730 | 9310 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 371328 | 881126 | N/A | — | — | — |
| **Cached Input Tokens** | 332717 | 826673 | N/A | — | — | — |
| **Uncached Input Tokens** | 38611 | 54453 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.62:1 | 15.18:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/resolver/resolver.go&#34;
}</code></pre></td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/resolver*.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/resolver&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/resolver/resolver.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000 \u0026\u0026 git log --oneline -10 2\u003e/dev/null; git status 2\u003e/dev/null | head -20&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/resolver_conn_wrapper.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000 \u0026\u0026 cat go.mod | head -20 \u0026\u0026 cat version.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -5 \u0026\u0026 git status&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000&#34;,
  &#34;pattern&#34;: &#34;ReportError&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/picker_wrapper.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/grpc/grpc-go/pull/3334)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://github.com/grpc/grpc-go/pull/3334&#34;
}</code></pre></td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/grpc/grpc-go/pull/2951/files)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;markdown&#34;,
  &#34;url&#34;: &#34;https://github.com/grpc/grpc-go/pull/2951/files&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/grpc/grpc-go/v1.25.0/resolver/resolver.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/grpc/grpc-go/v1.25.0/resolver/resolver.go&#34;
}</code></pre></td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/grpc/grpc-go/pull/2951.patch)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;text&#34;,
  &#34;url&#34;: &#34;https://github.com/grpc/grpc-go/pull/2951.patch&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/clientconn.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -10 2\u003e\u00261; echo \&#34;---\&#34;; cat go.mod&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/resolver_conn_wrapper.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git --git-dir=.git log --oneline -10 2\u003e\u00261&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000&#34;,
  &#34;pattern&#34;: &#34;firstResolveEvent&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/clientconn.go&#34;,
  &#34;pattern&#34;: &#34;func \\(c \\*ClientConn\\)&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000&#34;,
  &#34;pattern&#34;: &#34;waitForResolvedAddrs|toRPCErr|connectionError&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;clientconn.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000&#34;,
  &#34;pattern&#34;: &#34;func \\(c \\*ClientConn\\)&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/stream.go&#34;,
  &#34;limit&#34;: 120,
  &#34;offset&#34;: 140
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/clientconn.go&#34;,
  &#34;pattern&#34;: &#34;ClientConn\\)&#34;
}</code></pre></td></tr>
<tr><td>14</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791721002165963000/picker_wrapper.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/clientconn.go&#34;,
  &#34;limit&#34;: 250,
  &#34;offset&#34;: 470
}</code></pre></td></tr>
<tr><td>15</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/grpc/grpc-go/v1.25.0/clientconn.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/grpc/grpc-go/v1.25.0/clientconn.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/clientconn.go&#34;,
  &#34;limit&#34;: 200,
  &#34;offset&#34;: 720
}</code></pre></td></tr>
<tr><td>16</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls ~/go/pkg/mod/google.golang.org/ 2\u003e/dev/null; ls /Users/alessandro/sources 2\u003e/dev/null | head -30&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/balancer_conn_wrappers.go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://proxy.golang.org/google.golang.org/grpc/@v/v1.25.0.zip)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://proxy.golang.org/google.golang.org/grpc/@v/v1.25.0.zip&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/balancer.go&#34;,
  &#34;pattern&#34;: &#34;ClientConnState|UpdateClientConnState|type V2Balancer|type Balancer interface&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/balancer/balancer.go&#34;,
  &#34;limit&#34;: 80,
  &#34;offset&#34;: 260
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/grpc/grpc-go/v1.26.0/resolver/resolver.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;text&#34;,
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/grpc/grpc-go/v1.26.0/resolver/resolver.go&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000 \u0026\u0026 git log --oneline -10 2\u003e\u00261 || echo \&#34;git failed\&#34;&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/grpc/grpc-go/blob/v1.26.0/resolver/resolver.go?raw=true)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;text&#34;,
  &#34;url&#34;: &#34;https://github.com/grpc/grpc-go/blob/v1.26.0/resolver/resolver.go?raw=true&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791721302219967000/resolver/resolver.go&#34;,
  &#34;limit&#34;: 30,
  &#34;offset&#34;: 115
}</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://pkg.go.dev/google.golang.org/grpc@v1.26.0/resolver)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;markdown&#34;,
  &#34;url&#34;: &#34;https://pkg.go.dev/google.golang.org/grpc@v1.26.0/resolver&#34;
}</code></pre></td></tr>
</tbody>
</table>

