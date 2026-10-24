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

* **Date**: 2026-10-18 08:06:47 CEST

## Test case: `grpc__grpc-go-3361`

### Target: `codex/gpt-6-luna/medium (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/grpc__grpc-go-3361.txtar` (`sha256:e5118f092bc301743fb7bd28 (uncommitted)`)

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
>
> Do not edit tests.

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 27.92s | 0.00s | N/A | — | — | — |
| **Process Start → First Event** | 0.30s | — | — | — | — | — |
| **First Event → First Tool Call** | 3.66s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 0 | N/A | — | — | — |
| **Internal Tool Cycles** | 8 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 8 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 0 | N/A | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.005155 | — | — | — | — | — |
| **Output Tokens** | 877 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 211 | 0 | N/A | — | — | — |
| **Total Input Tokens** | 181376 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 149120 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 32256 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.62:1 | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; | head -80 &amp;&amp; \
git status --short --branch&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;find .. -name AGENTS.md -o -name PROJECT_BRIEF.md | head -30; rg -n &#34;ResolveNow|ReportError|NewAddress|UpdateState|roundrobin|connection refused&#34; resolver balancer&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; resolver/dns/dns_resolver.go; rg -n \&#34;ReportError|ResolveNow|UpdateState|NewAddress\&#34; balancer resolver&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func .*ResolveNow|ResolveNow\\(|ResolveNowOptions|ReportError|connectivityStateEvaluator|TransientFailure|Idle\&#34; internal/resolver/dns balancer/roundrobin balancer/base clientconn.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;160,255p&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;800,875p&#39; clientconn.go; sed -n &#39;930,980p&#39; clientconn.go;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1040,1170p&#39; clientconn.go; sed -n &#39;50,230p&#39; balancer/roundrobin/roundrobin.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;rn:|minDNSResRate|ResolveNow&#34; internal/resolver/dns/dns_resolver.go; git log -8 --oneline -- internal/resolver/dns/dns_resolver.go&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all --grep=&#39;3165&#39; &amp;&amp; \
git log -p -S &#39;resolveNow(resolver.ResolveNowOptions{})&#39; -- clientconn.go | head -200; git status --short&#34;</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/grpc__grpc-go-3361.txtar` (`sha256:e5118f092bc301743fb7bd28 (uncommitted)`)

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
>
> Do not edit tests.

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
> I recognize the API around DNS and load balancing may have changed recently (and are evolving) but I'm not seeing any clear indication of how to retain the expected behavior.
> 
>
>
> Do not edit tests. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 27.92s | 0.00s | N/A | — | — | — |
| **Process Start → First Event** | 0.30s | — | — | — | — | — |
| **First Event → First Tool Call** | 3.66s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 0 | N/A | — | — | — |
| **Internal Tool Cycles** | 8 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 8 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 0 | N/A | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.005155 | — | — | — | — | — |
| **Output Tokens** | 877 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 211 | 0 | N/A | — | — | — |
| **Total Input Tokens** | 181376 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 149120 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 32256 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 4.62:1 | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;*.go&#39; | head -80 &amp;&amp; \
git status --short --branch&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;find .. -name AGENTS.md -o -name PROJECT_BRIEF.md | head -30; rg -n &#34;ResolveNow|ReportError|NewAddress|UpdateState|roundrobin|connection refused&#34; resolver balancer&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; resolver/dns/dns_resolver.go; rg -n \&#34;ReportError|ResolveNow|UpdateState|NewAddress\&#34; balancer resolver&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func .*ResolveNow|ResolveNow\\(|ResolveNowOptions|ReportError|connectivityStateEvaluator|TransientFailure|Idle\&#34; internal/resolver/dns balancer/roundrobin balancer/base clientconn.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;160,255p&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;800,875p&#39; clientconn.go; sed -n &#39;930,980p&#39; clientconn.go;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1040,1170p&#39; clientconn.go; sed -n &#39;50,230p&#39; balancer/roundrobin/roundrobin.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;rn:|minDNSResRate|ResolveNow&#34; internal/resolver/dns/dns_resolver.go; git log -8 --oneline -- internal/resolver/dns/dns_resolver.go&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all --grep=&#39;3165&#39; &amp;&amp; \
git log -p -S &#39;resolveNow(resolver.ResolveNowOptions{})&#39; -- clientconn.go | head -200; git status --short&#34;</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>

