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

* **Date**: 2026-10-11 20:36:25 CEST

## Test case: `grpc__grpc-go-2951`

### Target: `opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/grpc__grpc-go-2951.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/grpc__grpc-go-2951.txtar)

**Vanilla LLM Prompt**:
> If a resolver could not resolve a name, there's currently no way to report error back to ClientConn, and RPCs on the ClientConn will fail with timeout.
> 
> This method allows resolver to report error to ClientConn, and failfast RPCs should fail with this error (or RPC errors that contain info from this error).
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 299.99s | 299.99s | N/A | — | — | — |
| **Process Start → First Event** | 8.44s | 128.27s | — | — | — | — |
| **First Event → First Tool Call** | 4.74s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 24 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 24 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 24 | 0 | N/A | — | — | — |
| **Output Tokens** | 11008 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 306581 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 282532 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 24049 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 11.75:1 | — | — | — | — | — |
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
  &#34;command&#34;: &#34;ls -la&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -10 \u0026\u0026 git status&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which git; ls /usr/local/bin/git /usr/bin/git 2\u003e/dev/null&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;/usr/bin/git --version; /usr/bin/git log --oneline -5 2\u003e\u00261&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;file /usr/bin/git; head -c 200 /usr/bin/git | od -c | head -5&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;alias git 2\u003e/dev/null; git config --global alias.git 2\u003e/dev/null; ls -la ~/.gitconfig 2\u003e/dev/null; cat ~/.gitconfig 2\u003e/dev/null; echo \&#34;---\&#34;; ls -la .git/hooks/ | head -20&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;/usr/bin/git -c core.abbrev=full log --oneline -5 2\u003e\u00261; echo \&#34;exit: $?\&#34;&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env -i /usr/bin/git --version 2\u003e\u00261; echo \&#34;exit: $?\&#34;&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>9</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /opt/homebrew/bin/git /opt/homebrew/opt/git/bin/git /usr/local/opt/git/bin/git /Library/Developer/CommandLineTools/usr/bin/git 2\u003e\u00261; ls /var/select/ 2\u003e\u00261&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>10</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;otool -L /usr/bin/git 2\u003e\u00261; strings /usr/bin/git 2\u003e/dev/null | grep -i -E \&#34;xcode|select|developer|git\&#34; | head -30&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>11</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;echo hello; pwd; ls -d /tmp&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>12</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which go gofmt 2\u003e\u00261; ls /usr/local/go/bin 2\u003e\u00261; ls /opt/homebrew/bin/go 2\u003e\u00261&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>13</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver/resolver.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>14</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): File not found: /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver_wrapper.go Did you mean one of these? /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver_wrapper.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>15</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000&#34;,
  &#34;pattern&#34;: &#34;ReportError&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>16</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000&#34;,
  &#34;pattern&#34;: &#34;TODO: add Err&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>17</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver_conn_wrapper.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>18</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver_conn_wrapper_test.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>19</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/clientconn.go&#34;,
  &#34;pattern&#34;: &#34;updateResolverState|resolverState|curResolverState|resolverErr&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>20</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/clientconn.go&#34;,
  &#34;limit&#34;: 120,
  &#34;offset&#34;: 500
}</code></pre></td><td>none recorded</td></tr>
<tr><td>21</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/version.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>22</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/go/mod/google.golang.org/ 2\u003e/dev/null&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>23</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/balancer/balancer.go&#34;
}</code></pre></td><td>none recorded</td></tr>
<tr><td>24</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/balancer_conn_wrappers.go&#34;
}</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

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
| **Wall-Clock Latency** | 299.99s | 251.02s | N/A | — | — | — |
| **Process Start → First Event** | 8.44s | 26.26s | — | — | — | — |
| **First Event → First Tool Call** | 4.74s | 4.26s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 24 | 3 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 24 | 3 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 24 | 3 | N/A | — | — | — |
| **Output Tokens** | 11008 | 402 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 306581 | 49783 | N/A | — | — | — |
| **Cached Input Tokens** | 282532 | 24822 | N/A | — | — | — |
| **Uncached Input Tokens** | 24049 | 24961 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 11.75:1 | 0.99:1 | N/A | — | — | — |
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
  &#34;command&#34;: &#34;ls -la&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791740591395719000&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -10 \u0026\u0026 git status&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791740591395719000/resolver/resolver.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which git; ls /usr/local/bin/git /usr/bin/git 2\u003e/dev/null&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2951_1791740591395719000/resolver_conn_wrapper.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;/usr/bin/git --version; /usr/bin/git log --oneline -5 2\u003e\u00261&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;file /usr/bin/git; head -c 200 /usr/bin/git | od -c | head -5&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;alias git 2\u003e/dev/null; git config --global alias.git 2\u003e/dev/null; ls -la ~/.gitconfig 2\u003e/dev/null; cat ~/.gitconfig 2\u003e/dev/null; echo \&#34;---\&#34;; ls -la .git/hooks/ | head -20&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>7</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;/usr/bin/git -c core.abbrev=full log --oneline -5 2\u003e\u00261; echo \&#34;exit: $?\&#34;&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>8</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;env -i /usr/bin/git --version 2\u003e\u00261; echo \&#34;exit: $?\&#34;&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>9</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /opt/homebrew/bin/git /opt/homebrew/opt/git/bin/git /usr/local/opt/git/bin/git /Library/Developer/CommandLineTools/usr/bin/git 2\u003e\u00261; ls /var/select/ 2\u003e\u00261&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>10</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;otool -L /usr/bin/git 2\u003e\u00261; strings /usr/bin/git 2\u003e/dev/null | grep -i -E \&#34;xcode|select|developer|git\&#34; | head -30&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>11</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;echo hello; pwd; ls -d /tmp&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>12</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which go gofmt 2\u003e\u00261; ls /usr/local/go/bin 2\u003e\u00261; ls /opt/homebrew/bin/go 2\u003e\u00261&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>13</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver/resolver.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>14</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): File not found: /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver_wrapper.go Did you mean one of these? /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver_wrapper.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>15</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000&#34;,
  &#34;pattern&#34;: &#34;ReportError&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>16</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000&#34;,
  &#34;pattern&#34;: &#34;TODO: add Err&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>17</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver_conn_wrapper.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>18</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/resolver_conn_wrapper_test.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>19</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/clientconn.go&#34;,
  &#34;pattern&#34;: &#34;updateResolverState|resolverState|curResolverState|resolverErr&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>20</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/clientconn.go&#34;,
  &#34;limit&#34;: 120,
  &#34;offset&#34;: 500
}</code></pre></td><td>—</td></tr>
<tr><td>21</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/version.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>22</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/go/mod/google.golang.org/ 2\u003e/dev/null&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>23</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/balancer/balancer.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>24</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2951_1791740289633819000/balancer_conn_wrappers.go&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

