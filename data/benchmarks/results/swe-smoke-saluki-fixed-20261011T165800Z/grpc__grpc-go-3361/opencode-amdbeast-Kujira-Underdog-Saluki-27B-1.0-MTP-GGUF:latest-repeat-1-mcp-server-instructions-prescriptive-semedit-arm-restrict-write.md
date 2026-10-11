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

## Test case: `grpc__grpc-go-3361`

### Target: `opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/grpc__grpc-go-3361.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/grpc__grpc-go-3361.txtar)

**Vanilla LLM Prompt**:
> ### What version of gRPC are you using?
> 
> 1.26
> 
> ### What version of Go are you using (`go version`)?
> 
> 1.13.4
> 
> ### What operating system (Linux, Windows, …) and version?
> 
> Linux / Kubernetes / Docker - various. 
> 
> ### What did you do?
> When using `grpc.WithBalancerName(roundrobin.Name)`, with a `dns:///` Dial target, errors do not trigger re-resolve.
> 
> Detail: a client starts with `dns:///my-service` as a dial target and the round_robin load balancer.  `my-service` DNS returns an IP `192.168.0.19`.  Due to an operational change, `my-service` is moved and DNS is updated to return `192.168.0.20`.  The service stops running on `192.168.0.19`, refusing connections.  
> 
> This is approximately the workflow that would occur when using Kubernetes and a headless service if a pod is replaced.  This happens for a variety of reasons, including routine updates and rescheduling.
> 
> ### What did you expect to see?
> (a) Upon receiving a connection refused, DNS should be re-queried for an updated endpoint list
> (b) Without receiving an error from an existing endpoint, the load balancer should be repopulated occasionally with the latest records in DNS as endpoint sets change
> 
> This was the behavior previous to 1.26.  By rolling code back to grpc 1.25.1, we were able to restore this behavior.  
>  https://github.com/grpc/grpc-go/pull/3165 is likely the culprit.
> 
> ### What did you see instead?
> (a) Endpoint list is never re-queried and the client simply throws a persistent error: 
> 
> ```
> rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp 10.1.5.253:443: connect: connection refused"
> ```
> (b) Since _every_ subsequent request fails, the endpoint list does not seem to getting updated with new records in DNS at all.
> 
> ### Additional
> 
> I recognize the API around DNS and load balancing may have changed recently (and are evolving) but I'm not seeing any clear indication of how to retain the expected behavior.
> 
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 300.00s | 300.02s | N/A | — | — | — |
| **Process Start → First Event** | 16.91s | 56.40s | — | — | — | — |
| **First Event → First Tool Call** | 12.45s | 18.67s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 7 | 3 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 7 | 3 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 3 | N/A | — | — | — |
| **Output Tokens** | 1347 | 258 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 66946 | 56881 | N/A | — | — | — |
| **Cached Input Tokens** | 54807 | 26857 | N/A | — | — | — |
| **Uncached Input Tokens** | 12139 | 30024 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.51:1 | 0.89:1 | N/A | — | — | — |
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
  &#34;command&#34;: &#34;ls&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-3361_1791741146563477000/balancer/roundrobin/roundrobin.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls balancer/base&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-3361_1791741146563477000/resolver/resolver.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_grpc__grpc-go-3361_1791741751067150000/balancer/base/balancer.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -15 \u0026\u0026 git status&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which git || ls /usr/bin/git /usr/local/bin/git 2\u003e/dev/null&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;/usr/bin/git log --oneline -15 2\u003e\u00261 | head -20&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>7</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Library/Developer/ 2\u003e\u00261; xcode-select -p 2\u003e\u00261; ls /usr/local/bin/git* /opt/homebrew/bin/git* 2\u003e\u00261&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/grpc__grpc-go-3361.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/grpc__grpc-go-3361.txtar)

**Vanilla LLM Prompt**:
> ### What version of gRPC are you using?
> 
> 1.26
> 
> ### What version of Go are you using (`go version`)?
> 
> 1.13.4
> 
> ### What operating system (Linux, Windows, …) and version?
> 
> Linux / Kubernetes / Docker - various. 
> 
> ### What did you do?
> When using `grpc.WithBalancerName(roundrobin.Name)`, with a `dns:///` Dial target, errors do not trigger re-resolve.
> 
> Detail: a client starts with `dns:///my-service` as a dial target and the round_robin load balancer.  `my-service` DNS returns an IP `192.168.0.19`.  Due to an operational change, `my-service` is moved and DNS is updated to return `192.168.0.20`.  The service stops running on `192.168.0.19`, refusing connections.  
> 
> This is approximately the workflow that would occur when using Kubernetes and a headless service if a pod is replaced.  This happens for a variety of reasons, including routine updates and rescheduling.
> 
> ### What did you expect to see?
> (a) Upon receiving a connection refused, DNS should be re-queried for an updated endpoint list
> (b) Without receiving an error from an existing endpoint, the load balancer should be repopulated occasionally with the latest records in DNS as endpoint sets change
> 
> This was the behavior previous to 1.26.  By rolling code back to grpc 1.25.1, we were able to restore this behavior.  
>  https://github.com/grpc/grpc-go/pull/3165 is likely the culprit.
> 
> ### What did you see instead?
> (a) Endpoint list is never re-queried and the client simply throws a persistent error: 
> 
> ```
> rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp 10.1.5.253:443: connect: connection refused"
> ```
> (b) Since _every_ subsequent request fails, the endpoint list does not seem to getting updated with new records in DNS at all.
> 
> ### Additional
> 
> I recognize the API around DNS and load balancing may have changed recently (and are evolving) but I'm not seeing any clear indication of how to retain the expected behavior.
> 
>

**Semedit MCP Prompt**:
> ### What version of gRPC are you using?
> 
> 1.26
> 
> ### What version of Go are you using (`go version`)?
> 
> 1.13.4
> 
> ### What operating system (Linux, Windows, …) and version?
> 
> Linux / Kubernetes / Docker - various. 
> 
> ### What did you do?
> When using `grpc.WithBalancerName(roundrobin.Name)`, with a `dns:///` Dial target, errors do not trigger re-resolve.
> 
> Detail: a client starts with `dns:///my-service` as a dial target and the round_robin load balancer.  `my-service` DNS returns an IP `192.168.0.19`.  Due to an operational change, `my-service` is moved and DNS is updated to return `192.168.0.20`.  The service stops running on `192.168.0.19`, refusing connections.  
> 
> This is approximately the workflow that would occur when using Kubernetes and a headless service if a pod is replaced.  This happens for a variety of reasons, including routine updates and rescheduling.
> 
> ### What did you expect to see?
> (a) Upon receiving a connection refused, DNS should be re-queried for an updated endpoint list
> (b) Without receiving an error from an existing endpoint, the load balancer should be repopulated occasionally with the latest records in DNS as endpoint sets change
> 
> This was the behavior previous to 1.26.  By rolling code back to grpc 1.25.1, we were able to restore this behavior.  
>  https://github.com/grpc/grpc-go/pull/3165 is likely the culprit.
> 
> ### What did you see instead?
> (a) Endpoint list is never re-queried and the client simply throws a persistent error: 
> 
> ```
> rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp 10.1.5.253:443: connect: connection refused"
> ```
> (b) Since _every_ subsequent request fails, the endpoint list does not seem to getting updated with new records in DNS at all.
> 
> ### Additional
> 
> I recognize the API around DNS and load balancing may have changed recently (and are evolving) but I'm not seeing any clear indication of how to retain the expected behavior. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 300.00s | 300.01s | N/A | — | — | — |
| **Process Start → First Event** | 16.91s | 49.95s | — | — | — | — |
| **First Event → First Tool Call** | 12.45s | 18.45s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 7 | 8 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 7 | 8 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 8 | N/A | — | — | — |
| **Output Tokens** | 1347 | 1190 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 66946 | 196868 | N/A | — | — | — |
| **Cached Input Tokens** | 54807 | 160728 | N/A | — | — | — |
| **Uncached Input Tokens** | 12139 | 36140 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.51:1 | 4.45:1 | N/A | — | — | — |
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
  &#34;command&#34;: &#34;ls&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791741448793714000&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-3361_1791741146563477000/balancer/roundrobin/roundrobin.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791741448793714000/resolver_conn_wrapper.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-3361_1791741146563477000/resolver/resolver.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791741448793714000/naming&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -15 \u0026\u0026 git status&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791741448793714000/naming/dns_resolver.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which git || ls /usr/bin/git /usr/local/bin/git 2\u003e/dev/null&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791741448793714000/resolver&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;/usr/bin/git log --oneline -15 2\u003e\u00261 | head -20&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791741448793714000/balancer&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Library/Developer/ 2\u003e\u00261; xcode-select -p 2\u003e\u00261; ls /usr/local/bin/git* /opt/homebrew/bin/git* 2\u003e\u00261&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791741448793714000/resolver/dns&#34;
}</code></pre></td></tr>
<tr><td>8</td><td>—</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791741448793714000/resolver/dns/dns_resolver.go&#34;
}</code></pre></td></tr>
</tbody>
</table>

