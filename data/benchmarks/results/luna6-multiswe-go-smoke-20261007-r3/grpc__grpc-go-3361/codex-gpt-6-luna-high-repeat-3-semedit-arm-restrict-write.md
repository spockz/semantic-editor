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

* **Date**: 2026-10-07 08:22:05 CEST

## Test case: `grpc__grpc-go-3361`

### Target: `codex/gpt-6-luna/high (repeat 3)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

* **Run Provenance**: `coding_network=disabled,oracle_patch_installation=checked`

* **Fixture**: [testdata/bench/grpc__grpc-go-3361.txtar](https://github.com/spockz/semantic-editor/blob/43bf7788d286eda6cb43f908eecb81fc2ff8694e/testdata/bench/grpc__grpc-go-3361.txtar)

**LLM Prompt**:
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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 678.80s | 712.90s | N/A | — | — | — |
| **Process Start → First Event** | 0.68s | 0.78s | — | — | — | — |
| **First Event → First Tool Call** | 3.85s | 6.80s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 21 | 13 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 7 | 9 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 21 | 13 | N/A | — | — | — |
| **Output Tokens** | 5931 | 4266 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 3707 | 2664 | N/A | — | — | — |
| **Total Input Tokens** | 673055 | 459355 | N/A | — | — | — |
| **Cached Input Tokens** | 605696 | 379392 | N/A | — | — | — |
| **Uncached Input Tokens** | 67359 | 79963 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.99:1 | 4.74:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*dns*&#39; -g &#39;*roundrobin*&#39; -g &#39;*balancer*&#39; | head -120&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*dns*&#39; -g &#39;*roundrobin*&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; | head -60 &amp;&amp; \
git status --short &amp;&amp; \
git log -5 --oneline&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; | head -60 &amp;&amp; \
git status --short &amp;&amp; \
sed -n &#39;1,240p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
sed -n &#39;1,260p&#39; balancer/roundrobin/roundrobin.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; resolver/dns/dns_resolver.go &amp;&amp; \
sed -n &#39;1,260p&#39; balancer/roundrobin/roundrobin.go &amp;&amp; \
rg -n \&#34;ResolveNow|ReportError|ExitIdle|connection refused|dns\&#34; resolver/dns internal/resolver/dns balancer/roundrobin clientconn.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;240,540p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
rg -n &#39;ResolveNow|ReportError|UpdateState|connection refused|TransientFailure&#39; balancer clientconn.go resolver | head -180 &amp;&amp; \
sed -n &#39;1,180p&#39; go.mod &amp;&amp; \
sed -n &#39;1,200p&#39; README.md&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;95,230p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
sed -n &#39;920,990p&#39; clientconn.go &amp;&amp; \
sed -n &#39;1060,1120p&#39; clientconn.go &amp;&amp; \
rg -n \&#34;ResolveNow|UpdateState|ReportError\&#34; balancer/base balancer_v1_wrapper.go balancer/balancer.go clientconn.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;790,880p&#39; clientconn.go &amp;&amp; \
sed -n &#39;930,980p&#39; clientconn.go &amp;&amp; \
sed -n &#39;1070,1160p&#39; clientconn.go &amp;&amp; \
sed -n &#39;1,250p&#39; balancer/base/balancer.go &amp;&amp; \
git log --oneline -10&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all -- internal/resolver/dns/dns_resolver.go | head -20 &amp;&amp; \
sed -n &#39;1060,1180p&#39; clientconn.go &amp;&amp; \
sed -n &#39;1,110p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
rg -n \&#34;minDNSResRate|dnsResRate|ResolveNow\&#34; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;type resolverWrapper|func \\(.*resolveNow|ResolveNow&#39; resolver_wrapper.go clientconn.go &amp;&amp; \
sed -n &#39;1080,1125p&#39; clientconn.go &amp;&amp; \
rg -n &#39;minDNSResRate|watcher|periodic&#39; internal/resolver/dns/*test.go&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline --grep=&#39;#3165&#39; | head &amp;&amp; \
git log --oneline --all -- clientconn.go | head -15 &amp;&amp; \
rg -n \&#34;resolveNow\\(resolver.ResolveNowOptions\&#34; --glob &#39;*.go&#39; | head -30 &amp;&amp; \
cat go.mod | head -12&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files | rg &#39;resolver.*wrapper|resolver_wrapper|resolver.*go&#34;&#39;$&#39;&#34;&#39; | head -60 &amp;&amp; \
rg -n &#39;func \\(.*resolveNow|ResolveNow\\(&#39; --glob &#39;*.go&#39; | head -100 &amp;&amp; \
rg -n &#39;minDNSResRate|watcher&#39; internal/resolver/dns/*test.go&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --stat --oneline e5e980f2 &amp;&amp; \
git show e5e980f2 --&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;70,135p&#39; resolver_conn_wrapper.go &amp;&amp; \
sed -n &#39;565,590p&#39; dialoptions.go &amp;&amp; \
git show --stat --oneline HEAD &amp;&amp; \
git log --all --oneline -- internal/resolver/dns/dns_resolver.go | head&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --format=fuller --no-patch e5e980f2 &amp;&amp; \
git show --format= --stat e5e980f2&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline --grep=&#39;3165\\|re-resol\\|re-resolv\\|DNS&#39; -i | head -60 &amp;&amp; \
git show 648cf9b0&#34;&#39;^:internal/resolver/dns/dns_resolver.go 2&gt;/dev/null | sed -n &#39;&#34;&#39;170,230p&#39;&#34;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;100,175p&#39; resolver_conn_wrapper.go &amp;&amp; \
git show e5e980f2:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,230p&#39; &amp;&amp; \
git show e5e980f2&#34;&#39;^:internal/resolver/dns/dns_resolver.go 2&gt;&amp;1 | head -10&#39;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --stat e5e980f2 &amp;&amp; \
git show e5e980f2 -- internal/resolver/dns/dns_resolver.go &amp;&amp; \
git show 6f8ce092 -- internal/resolver/dns/dns_resolver.go&#39;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline -S &#39;time.After&#39; -- internal/resolver/dns/dns_resolver.go | head -20 &amp;&amp; \
git log --all --oneline --ancestry-path e5e980f2..HEAD -- internal/resolver/dns/dns_resolver.go | head -20 &amp;&amp; \
git branch --contains HEAD | head&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --format=fuller --stat 6f8ce092 &amp;&amp; \
git show --format= --find-renames 6f8ce092 -- internal/resolver/dns/dns_resolver.go&#39;</code></pre></td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git log --oneline --all --decorate --max-count=25 | head -25 &amp;&amp; \
git show --stat 9dc72d1d &amp;&amp; \
git show --format= --stat 6f8ce092&#39;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git blame -L 1095,1105 clientconn.go &amp;&amp; \
git blame -L 185,225 internal/resolver/dns/dns_resolver.go &amp;&amp; \
git log --oneline --all -- clientconn.go | tail -20&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w internal/resolver/dns/dns_resolver.go &amp;&amp; \
git diff -- internal/resolver/dns/dns_resolver.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --format= -- internal/resolver/dns/dns_resolver.go 6f8ce092 | head&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1210,1325p&#39; internal/resolver/dns/dns_resolver_test.go &amp;&amp; \
rg -n &#39;TestRateLimitedResolve|RateLimited&#39; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git tag -l &#39;v1.25*&#39; | head &amp;&amp; \
git show v1.25.1:internal/resolver/dns/dns_resolver.go 2&gt;/dev/null | sed -n &#39;150,235p&#39; | head -85 &amp;&amp; \
rg -n \&#34;resolver.*poll|DNS.*poll|polling\&#34; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-tree -r --name-only v1.25.1 | rg &#39;dns_resolver.go|resolver/dns&#39; | head -20&#34;</code></pre></td><td>—</td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show v1.25.1:internal/resolver/dns/dns_resolver.go | sed -n &#39;155,235p&#39;&#34;</code></pre></td><td>—</td></tr>
<tr><td>17</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1220,1320p&#39; internal/resolver/dns/dns_resolver_test.go &amp;&amp; \
sed -n &#39;830,975p&#39; internal/resolver/dns/dns_resolver_test.go &amp;&amp; \
sed -n &#39;1,180p&#39; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>18</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>19</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>20</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- internal/resolver/dns/dns_resolver.go &amp;&amp; \
git status --short&#39;</code></pre></td><td>—</td></tr>
<tr><td>21</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
</tbody>
</table>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 678.80s | 614.49s | N/A | — | — | — |
| **Process Start → First Event** | 0.68s | — | — | — | — | — |
| **First Event → First Tool Call** | 3.85s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 0 | N/A | — | — | — |
| **Internal Tool Cycles** | 21 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 7 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 21 | 0 | N/A | — | — | — |
| **Output Tokens** | 5931 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 3707 | 0 | N/A | — | — | — |
| **Total Input Tokens** | 673055 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 605696 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 67359 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 8.99:1 | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context
* **MCP Edit**: diff --git a/balancer/base/balancer.go b/balancer/base/balancer.go
index d952f09f..d7d72918 100644
--- a/balancer/base/balancer.go
+++ b/balancer/base/balancer.go
@@ -21,6 +21,7 @@ package base
 import (
 	"context"
 	"errors"
+	"fmt"
 
 	"google.golang.org/grpc/balancer"
 	"google.golang.org/grpc/connectivity"
@@ -76,6 +77,9 @@ type baseBalancer struct {
 	picker   balancer.Picker
 	v2Picker balancer.V2Picker
 	config   Config
+
+	resolverErr error // the last error reported by the resolver; cleared on successful resolution
+	connErr     error // the last connection error; cleared upon leaving TransientFailure
 }
 
 func (b *baseBalancer) HandleResolvedAddrs(addrs []resolver.Address, err error) {
@@ -83,13 +87,23 @@ func (b *baseBalancer) HandleResolvedAddrs(addrs []resolver.Address, err error)
 }
 
 func (b *baseBalancer) ResolverError(err error) {
-	switch b.state {
-	case connectivity.TransientFailure, connectivity.Idle, connectivity.Connecting:
-		if b.picker != nil {
-			b.picker = NewErrPicker(err)
-		} else {
-			b.v2Picker = NewErrPickerV2(err)
-		}
+	b.resolverErr = err
+	if len(b.subConns) == 0 {
+		b.state = connectivity.TransientFailure
+	}
+	if b.state != connectivity.TransientFailure {
+		// The picker will not change since the balancer does not currently
+		// report an error.
+		return
+	}
+	b.regeneratePicker()
+	if b.picker != nil {
+		b.cc.UpdateBalancerState(b.state, b.picker)
+	} else {
+		b.cc.UpdateState(balancer.State{
+			ConnectivityState: b.state,
+			Picker:            b.v2Picker,
+		})
 	}
 }
 
@@ -99,6 +113,12 @@ func (b *baseBalancer) UpdateClientConnState(s balancer.ClientConnState) error {
 	if grpclog.V(2) {
 		grpclog.Infoln("base.baseBalancer: got new ClientConn state: ", s)
 	}
+	if len(s.ResolverState.Addresses) == 0 {
+		b.ResolverError(errors.New("produced zero addresses"))
+		return balancer.ErrBadResolverState
+	}
+	// Successful resolution; clear resolver error and ensure we return nil.
+	b.resolverErr = nil
 	// addrsSet is the set converted from addrs, it's used for quick lookup of an address.
 	addrsSet := make(map[resolver.Address]struct{})
 	for _, a := range s.ResolverState.Addresses {
@@ -127,24 +147,30 @@ func (b *baseBalancer) UpdateClientConnState(s balancer.ClientConnState) error {
 	return nil
 }
 
+// mergeErrors builds an error from the last connection error and the last
+// resolver error.  Must only be called if b.state is TransientFailure.
+func (b *baseBalancer) mergeErrors() error {
+	// connErr must always be non-nil unless there are no SubConns, in which
+	// case resolverErr must be non-nil.
+	if b.connErr == nil {
+		return fmt.Errorf("last resolver error: %v", b.resolverErr)
+	}
+	if b.resolverErr == nil {
+		return fmt.Errorf("last connection error: %v", b.connErr)
+	}
+	return fmt.Errorf("last connection error: %v; last resolver error: %v", b.connErr, b.resolverErr)
+}
+
 // regeneratePicker takes a snapshot of the balancer, and generates a picker
 // from it. The picker is
-//  - errPicker with ErrTransientFailure if the balancer is in TransientFailure,
+//  - errPicker if the balancer is in TransientFailure,
 //  - built by the pickerBuilder with all READY SubConns otherwise.
-func (b *baseBalancer) regeneratePicker(err error) {
+func (b *baseBalancer) regeneratePicker() {
 	if b.state == connectivity.TransientFailure {
 		if b.pickerBuilder != nil {
 			b.picker = NewErrPicker(balancer.ErrTransientFailure)
 		} else {
-			if err != nil {
-				b.v2Picker = NewErrPickerV2(balancer.TransientFailureError(err))
-			} else {
-				// This means the last subchannel transition was not to
-				// TransientFailure (otherwise err must be set), but the
-				// aggregate state of the balancer is TransientFailure, meaning
-				// there are no other addresses.
-				b.v2Picker = NewErrPickerV2(balancer.TransientFailureError(errors.New("resolver returned no addresses")))
-			}
+			b.v2Picker = NewErrPickerV2(balancer.TransientFailureError(b.mergeErrors()))
 		}
 		return
 	}
@@ -200,6 +226,9 @@ func (b *baseBalancer) UpdateSubConnState(sc balancer.SubConn, state balancer.Su
 	oldAggrState := b.state
 	b.state = b.csEvltr.RecordTransition(oldS, s)
 
+	// Set or clear the last connection error accordingly.
+	b.connErr = state.ConnectionError
+
 	// Regenerate picker when one of the following happens:
 	//  - this sc became ready from not-ready
 	//  - this sc became not-ready from ready
@@ -207,7 +236,7 @@ func (b *baseBalancer) UpdateSubConnState(sc balancer.SubConn, state balancer.Su
 	//  - the aggregated state of balancer became non-TransientFailure from TransientFailure
 	if (s == connectivity.Ready) != (oldS == connectivity.Ready) ||
 		(b.state == connectivity.TransientFailure) != (oldAggrState == connectivity.TransientFailure) {
-		b.regeneratePicker(state.ConnectionError)
+		b.regeneratePicker()
 	}
 
 	if b.picker != nil {


<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*dns*&#39; -g &#39;*roundrobin*&#39; -g &#39;*balancer*&#39; | head -120&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; | head -60 &amp;&amp; \
git status --short &amp;&amp; \
git log -5 --oneline&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; resolver/dns/dns_resolver.go &amp;&amp; \
sed -n &#39;1,260p&#39; balancer/roundrobin/roundrobin.go &amp;&amp; \
rg -n \&#34;ResolveNow|ReportError|ExitIdle|connection refused|dns\&#34; resolver/dns internal/resolver/dns balancer/roundrobin clientconn.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;95,230p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
sed -n &#39;920,990p&#39; clientconn.go &amp;&amp; \
sed -n &#39;1060,1120p&#39; clientconn.go &amp;&amp; \
rg -n \&#34;ResolveNow|UpdateState|ReportError\&#34; balancer/base balancer_v1_wrapper.go balancer/balancer.go clientconn.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --oneline --all -- internal/resolver/dns/dns_resolver.go | head -20 &amp;&amp; \
sed -n &#39;1060,1180p&#39; clientconn.go &amp;&amp; \
sed -n &#39;1,110p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
rg -n \&#34;minDNSResRate|dnsResRate|ResolveNow\&#34; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline --grep=&#39;#3165&#39; | head &amp;&amp; \
git log --oneline --all -- clientconn.go | head -15 &amp;&amp; \
rg -n \&#34;resolveNow\\(resolver.ResolveNowOptions\&#34; --glob &#39;*.go&#39; | head -30 &amp;&amp; \
cat go.mod | head -12&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --stat --oneline e5e980f2 &amp;&amp; \
git show e5e980f2 --&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --format=fuller --no-patch e5e980f2 &amp;&amp; \
git show --format= --stat e5e980f2&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;100,175p&#39; resolver_conn_wrapper.go &amp;&amp; \
git show e5e980f2:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,230p&#39; &amp;&amp; \
git show e5e980f2&#34;&#39;^:internal/resolver/dns/dns_resolver.go 2&gt;&amp;1 | head -10&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline -S &#39;time.After&#39; -- internal/resolver/dns/dns_resolver.go | head -20 &amp;&amp; \
git log --all --oneline --ancestry-path e5e980f2..HEAD -- internal/resolver/dns/dns_resolver.go | head -20 &amp;&amp; \
git branch --contains HEAD | head&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>11</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git log --oneline --all --decorate --max-count=25 | head -25 &amp;&amp; \
git show --stat 9dc72d1d &amp;&amp; \
git show --format= --stat 6f8ce092&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git blame -L 1095,1105 clientconn.go &amp;&amp; \
git blame -L 185,225 internal/resolver/dns/dns_resolver.go &amp;&amp; \
git log --oneline --all -- clientconn.go | tail -20&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --format= -- internal/resolver/dns/dns_resolver.go 6f8ce092 | head&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git tag -l &#39;v1.25*&#39; | head &amp;&amp; \
git show v1.25.1:internal/resolver/dns/dns_resolver.go 2&gt;/dev/null | sed -n &#39;150,235p&#39; | head -85 &amp;&amp; \
rg -n \&#34;resolver.*poll|DNS.*poll|polling\&#34; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-tree -r --name-only v1.25.1 | rg &#39;dns_resolver.go|resolver/dns&#39; | head -20&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show v1.25.1:internal/resolver/dns/dns_resolver.go | sed -n &#39;155,235p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>17</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1220,1320p&#39; internal/resolver/dns/dns_resolver_test.go &amp;&amp; \
sed -n &#39;830,975p&#39; internal/resolver/dns/dns_resolver_test.go &amp;&amp; \
sed -n &#39;1,180p&#39; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>18</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>19</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>20</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- internal/resolver/dns/dns_resolver.go &amp;&amp; \
git status --short&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>21</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
</tbody>
</table>

