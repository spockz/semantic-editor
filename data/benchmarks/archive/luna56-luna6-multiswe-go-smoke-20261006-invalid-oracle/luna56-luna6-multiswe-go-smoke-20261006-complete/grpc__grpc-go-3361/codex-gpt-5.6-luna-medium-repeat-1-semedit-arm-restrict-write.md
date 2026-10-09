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

* **Date**: 2026-10-06 23:33:13 CEST

## Test case: `grpc__grpc-go-3361`

### Target: `codex/gpt-5.6-luna/medium (repeat 1)`

#### Configuration: default prompt · none MCP instructions · write semedit restriction

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
>
> When using `grpc.WithBalancerName(roundrobin.Name)`, with a `dns:///` Dial target, errors do not trigger re-resolve.
>
> Detail: a client starts with `dns:///my-service` as a dial target and the round_robin load balancer.  `my-service` DNS returns an IP `192.168.0.19`.  Due to an operational change, `my-service` is moved and DNS is updated to return `192.168.0.20`.  The service stops running on `192.168.0.19`, refusing connections.  
>
> This is approximately the workflow that would occur when using Kubernetes and a headless service if a pod is replaced.  This happens for a variety of reasons, including routine updates and rescheduling.
>
> ### What did you expect to see?
>
> (a) Upon receiving a connection refused, DNS should be re-queried for an updated endpoint list
> (b) Without receiving an error from an existing endpoint, the load balancer should be repopulated occasionally with the latest records in DNS as endpoint sets change
>
> This was the behavior previous to 1.26.  By rolling code back to grpc 1.25.1, we were able to restore this behavior.  
> <https://github.com/grpc/grpc-go/pull/3165> is likely the culprit.
>
> ### What did you see instead?
>
> (a) Endpoint list is never re-queried and the client simply throws a persistent error:
>
> ```
> rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp 10.1.5.253:443: connect: connection refused"
> ```
>
> (b) Since _every_ subsequent request fails, the endpoint list does not seem to getting updated with new records in DNS at all.
>
> ### Additional
>
> I recognize the API around DNS and load balancing may have changed recently (and are evolving) but I'm not seeing any clear indication of how to retain the expected behavior.
>
>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 672.48s | 669.70s | N/A | — | — | — |
| **Process Start → First Event** | 0.30s | 0.62s | — | — | — | — |
| **First Event → First Tool Call** | 3.69s | 4.25s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 14 | 15 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 6 | 4 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 14 | 15 | N/A | — | — | — |
| **Output Tokens** | 5540 | 5966 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 2837 | 2836 | N/A | — | — | — |
| **Total Input Tokens** | 502443 | 655625 | N/A | — | — | — |
| **Cached Input Tokens** | 456960 | 598016 | N/A | — | — | — |
| **Uncached Input Tokens** | 45483 | 57609 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 10.05:1 | 10.38:1 | N/A | — | — | — |
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
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*dns*&#39; -g &#39;*roundrobin*&#39; | head -80 &amp;&amp; \
git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*dns*&#39; -g &#39;*roundrobin*&#39; | head -80 &amp;&amp; \
git status --short &amp;&amp; \
git log -8 --oneline&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
sed -n &#39;1,300p&#39; balancer/roundrobin/roundrobin.go &amp;&amp; \
rg -n \&#34;ResolveNow|BuildOption|dns|re-resolv|resolver\&#34; internal/resolver balancer/clientconn.go clientconn.go | head -160&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; resolver/dns/dns_resolver.go; sed -n &#39;1,320p&#39; balancer/roundrobin/roundrobin.go; rg -n \&#34;ResolveNow|Backoff|connection|re-resolv|dns\&#34; resolver internal balancer clientconn.go | head -200&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;560,690p&#39; clientconn.go; sed -n &#39;930,980p&#39; clientconn.go; sed -n &#39;1070,1130p&#39; clientconn.go; sed -n &#39;1420,1475p&#39; clientconn.go; git log --oneline -8 -- internal/resolver/dns/dns_resolver.go clientconn.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1000,1170p&#39; clientconn.go; sed -n &#39;150,225p&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;1,180p&#39; balancer/base/balancer.go; git show --stat --oneline HEAD &amp;&amp; \
git log --all --oneline -- clientconn.go | head -20&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline --decorate --max-count=30; sed -n &#39;1180,1260p&#39; clientconn.go; sed -n &#39;1230,1320p&#39; internal/resolver/dns/dns_resolver_test.go; sed -n &#39;1300,1365p&#39; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline --grep=&#39;3165\\|re-resolv\\|resolve.*error\\|roundrobin.*dns&#39; -i | head -40; git log --all --oneline --since=&#39;2019-01-01&#39; --until=&#39;2020-01-01&#39; -- clientconn.go internal/resolver/dns/dns_resolver.go balancer/base/balancer.go | head -80; sed -n &#39;1080,1165p&#39; clientconn.go; sed -n &#39;300,390p&#39; internal/resolver/dns/dns_resolver.go&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline --grep=&#39;resolve\\|re-resolv\\|roundrobin\\|round_robin&#39; -i --max-count=40; git show 648cf9b0&#34;&#39;^:internal/resolver/dns/dns_resolver.go | sed -n &#39;&#34;&#39;150,215p&#39;; git show 648cf9b0:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,215p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --format=fuller --stat e5e980f2; git show --format= --find-renames e5e980f2 -- clientconn.go internal/resolver/dns/dns_resolver.go; git show --format= --find-renames 9dc72d1d -- clientconn.go internal/resolver/dns/dns_resolver.go; git show --format= --find-renames 6f8ce092 -- clientconn.go internal/resolver/dns/dns_resolver.go&#39;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline -S&#39;resolveNow(resolver.ResolveNowOptions{})&#39; -- clientconn.go | head -20; git blame -L 1080,1110 clientconn.go; git log --all --oneline --since=2019-01-01 --until=2020-01-01 -- clientconn.go internal/resolver/dns/dns_resolver.go | head -30&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git branch --show-current; git rev-parse HEAD; git log -1 --format=&#39;%H %ad %s&#39; --date=iso; git blame -L 1090,1108 clientconn.go; git blame -L 185,220 internal/resolver/dns/dns_resolver.go; git show e5e980f2&#34;&#39;^:clientconn.go | sed -n &#39;&#34;&#39;1080,1120p&#39;&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show --stat --oneline e5e980f2; git show --format=fuller --find-renames e5e980f2 -- internal/resolver/dns/dns_resolver.go | sed -n &#39;1,260p&#39;; git show --stat --oneline 6f8ce092; git show --format=fuller 6f8ce092 -- internal/resolver/dns/dns_resolver.go | sed -n &#39;1,300p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --format=fuller 6f8ce092 --; git show --format=fuller 9dc72d1d --; git show --format=fuller e5e980f2 --name-only | head -80; rg -n &#34;minDNSResRate|dnsResRate|resolveNow&#34; internal/resolver/dns clientconn*_test.go balancer | head -100&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show 9dc72d1d --format= -- internal/resolver/dns/dns_resolver.go | head -260; git show 6f8ce092&#39;&#39;^:internal/resolver/dns/dns_resolver.go 2&gt;&amp;1 | sed -n &#39;&#34;&#39;160,235p&#39;; git show 6f8ce092:internal/resolver/dns/dns_resolver.go 2&gt;&amp;1 | sed -n &#39;160,235p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1200,1320p&#39; internal/resolver/dns/dns_resolver_test.go; git show 9dc72d1d&#34;&#39;^:internal/resolver/dns/dns_resolver.go &gt;/tmp/dns-before.go 2&gt;/dev/null || \
true; if test -f /tmp/dns-before.go; then sed -n &#39;&#34;&#39;180,235p&#39; /tmp/dns-before.go; fi; git fsck --no-reflogs --unreachable 2&gt;/dev/null | head -30&#34;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-tree -r --name-only e5e980f2 | rg &#39;dns_resolver.go&#34;&#39;$&#39;&#34;&#39;; git show e5e980f2:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,250p&#39;; git show 9dc72d1d:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,250p&#39;; git show 6f8ce092:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,250p&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;105,150p&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;1,155p&#39; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; internal/resolver/dns/dns_resolver_test.go; sed -n &#39;180,245p&#39; internal/resolver/dns/dns_resolver.go; rg -n \&#34;watcher|RateLimited|ResolveNow|replaceDNSResRate\&#34; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>11</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w internal/resolver/dns/dns_resolver.go &amp;&amp; \
GOCACHE=&#34;$PWD/.scratch/go-build&#34; TMPDIR=&#34;$PWD/.scratch/tmp&#34; go test ./internal/resolver/dns ./balancer/roundrobin ./...&#39;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w internal/resolver/dns/dns_resolver.go &amp;&amp; \
go test ./internal/resolver/dns&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- internal/resolver/dns/dns_resolver.go &amp;&amp; \
git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>13</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d internal/resolver/dns/dns_resolver.go; git diff --check; git diff -- internal/resolver/dns/dns_resolver.go; git status --short&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;gofmt -w internal/resolver/dns/dns_resolver.go &amp;&amp; \
git diff --check &amp;&amp; \
sed -n &#39;195,240p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
</tbody>
</table>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 672.48s | 609.98s | N/A | — | — | — |
| **Process Start → First Event** | 0.30s | — | — | — | — | — |
| **First Event → First Tool Call** | 3.69s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 0 | N/A | — | — | — |
| **Internal Tool Cycles** | 14 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 6 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 14 | 0 | N/A | — | — | — |
| **Output Tokens** | 5540 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 2837 | 0 | N/A | — | — | — |
| **Total Input Tokens** | 502443 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 456960 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 45483 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 10.05:1 | — | — | — | — | — |
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
* "fmt"

  "google.golang.org/grpc/balancer"
  "google.golang.org/grpc/connectivity"
@@ -76,6 +77,9 @@ type baseBalancer struct {
  picker   balancer.Picker
  v2Picker balancer.V2Picker
  config   Config
*
* resolverErr error // the last error reported by the resolver; cleared on successful resolution
* connErr     error // the last connection error; cleared upon leaving TransientFailure
 }

 func (b *baseBalancer) HandleResolvedAddrs(addrs []resolver.Address, err error) {
@@ -83,13 +87,23 @@ func (b*baseBalancer) HandleResolvedAddrs(addrs []resolver.Address, err error)
 }

 func (b *baseBalancer) ResolverError(err error) {

* switch b.state {
* case connectivity.TransientFailure, connectivity.Idle, connectivity.Connecting:
* if b.picker != nil {
* b.picker = NewErrPicker(err)
* } else {
* b.v2Picker = NewErrPickerV2(err)
* }
* b.resolverErr = err
* if len(b.subConns) == 0 {
* b.state = connectivity.TransientFailure
* }
* if b.state != connectivity.TransientFailure {
* // The picker will not change since the balancer does not currently
* // report an error.
* return
* }
* b.regeneratePicker()
* if b.picker != nil {
* b.cc.UpdateBalancerState(b.state, b.picker)
* } else {
* b.cc.UpdateState(balancer.State{
* ConnectivityState: b.state,
* Picker:            b.v2Picker,
* })
  }
 }

@@ -99,6 +113,12 @@ func (b *baseBalancer) UpdateClientConnState(s balancer.ClientConnState) error {
  if grpclog.V(2) {
   grpclog.Infoln("base.baseBalancer: got new ClientConn state: ", s)
  }

* if len(s.ResolverState.Addresses) == 0 {
* b.ResolverError(errors.New("produced zero addresses"))
* return balancer.ErrBadResolverState
* }
* // Successful resolution; clear resolver error and ensure we return nil.
* b.resolverErr = nil
  // addrsSet is the set converted from addrs, it's used for quick lookup of an address.
  addrsSet := make(map[resolver.Address]struct{})
  for _, a := range s.ResolverState.Addresses {
@@ -127,24 +147,30 @@ func (b *baseBalancer) UpdateClientConnState(s balancer.ClientConnState) error {
  return nil
 }

+// mergeErrors builds an error from the last connection error and the last
+// resolver error.  Must only be called if b.state is TransientFailure.
+func (b *baseBalancer) mergeErrors() error {

* // connErr must always be non-nil unless there are no SubConns, in which
* // case resolverErr must be non-nil.
* if b.connErr == nil {
* return fmt.Errorf("last resolver error: %v", b.resolverErr)
* }
* if b.resolverErr == nil {
* return fmt.Errorf("last connection error: %v", b.connErr)
* }
* return fmt.Errorf("last connection error: %v; last resolver error: %v", b.connErr, b.resolverErr)
+}
*

 // regeneratePicker takes a snapshot of the balancer, and generates a picker
 // from it. The picker is
-//  - errPicker with ErrTransientFailure if the balancer is in TransientFailure,
+//  - errPicker if the balancer is in TransientFailure,
 //  - built by the pickerBuilder with all READY SubConns otherwise.
-func (b *baseBalancer) regeneratePicker(err error) {
+func (b*baseBalancer) regeneratePicker() {
  if b.state == connectivity.TransientFailure {
   if b.pickerBuilder != nil {
    b.picker = NewErrPicker(balancer.ErrTransientFailure)
   } else {

* if err != nil {
* b.v2Picker = NewErrPickerV2(balancer.TransientFailureError(err))
* } else {
* // This means the last subchannel transition was not to
* // TransientFailure (otherwise err must be set), but the
* // aggregate state of the balancer is TransientFailure, meaning
* // there are no other addresses.
* b.v2Picker = NewErrPickerV2(balancer.TransientFailureError(errors.New("resolver returned no addresses")))
* }
* b.v2Picker = NewErrPickerV2(balancer.TransientFailureError(b.mergeErrors()))
   }
   return
  }
@@ -200,6 +226,9 @@ func (b *baseBalancer) UpdateSubConnState(sc balancer.SubConn, state balancer.Su
  oldAggrState := b.state
  b.state = b.csEvltr.RecordTransition(oldS, s)

* // Set or clear the last connection error accordingly.
* b.connErr = state.ConnectionError
*
  // Regenerate picker when one of the following happens:
  //  - this sc became ready from not-ready
  //  - this sc became not-ready from ready
@@ -207,7 +236,7 @@ func (b *baseBalancer) UpdateSubConnState(sc balancer.SubConn, state balancer.Su
  //  - the aggregated state of balancer became non-TransientFailure from TransientFailure
  if (s == connectivity.Ready) != (oldS == connectivity.Ready) ||
   (b.state == connectivity.TransientFailure) != (oldAggrState == connectivity.TransientFailure) {
* b.regeneratePicker(state.ConnectionError)
* b.regeneratePicker()
  }

  if b.picker != nil {

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;go.mod&#39; -g &#39;*dns*&#39; -g &#39;*roundrobin*&#39; | head -80 &amp;&amp; \
git status --short&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
sed -n &#39;1,300p&#39; balancer/roundrobin/roundrobin.go &amp;&amp; \
rg -n \&#34;ResolveNow|BuildOption|dns|re-resolv|resolver\&#34; internal/resolver balancer/clientconn.go clientconn.go | head -160&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;560,690p&#39; clientconn.go; sed -n &#39;930,980p&#39; clientconn.go; sed -n &#39;1070,1130p&#39; clientconn.go; sed -n &#39;1420,1475p&#39; clientconn.go; git log --oneline -8 -- internal/resolver/dns/dns_resolver.go clientconn.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline --decorate --max-count=30; sed -n &#39;1180,1260p&#39; clientconn.go; sed -n &#39;1230,1320p&#39; internal/resolver/dns/dns_resolver_test.go; sed -n &#39;1300,1365p&#39; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline --grep=&#39;resolve\\|re-resolv\\|roundrobin\\|round_robin&#39; -i --max-count=40; git show 648cf9b0&#34;&#39;^:internal/resolver/dns/dns_resolver.go | sed -n &#39;&#34;&#39;150,215p&#39;; git show 648cf9b0:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,215p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git log --all --oneline -S&#39;resolveNow(resolver.ResolveNowOptions{})&#39; -- clientconn.go | head -20; git blame -L 1080,1110 clientconn.go; git log --all --oneline --since=2019-01-01 --until=2020-01-01 -- clientconn.go internal/resolver/dns/dns_resolver.go | head -30&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git show --stat --oneline e5e980f2; git show --format=fuller --find-renames e5e980f2 -- internal/resolver/dns/dns_resolver.go | sed -n &#39;1,260p&#39;; git show --stat --oneline 6f8ce092; git show --format=fuller 6f8ce092 -- internal/resolver/dns/dns_resolver.go | sed -n &#39;1,300p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show 9dc72d1d --format= -- internal/resolver/dns/dns_resolver.go | head -260; git show 6f8ce092&#39;&#39;^:internal/resolver/dns/dns_resolver.go 2&gt;&amp;1 | sed -n &#39;&#34;&#39;160,235p&#39;; git show 6f8ce092:internal/resolver/dns/dns_resolver.go 2&gt;&amp;1 | sed -n &#39;160,235p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git ls-tree -r --name-only e5e980f2 | rg &#39;dns_resolver.go&#34;&#39;$&#39;&#34;&#39;; git show e5e980f2:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,250p&#39;; git show 9dc72d1d:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,250p&#39;; git show 6f8ce092:internal/resolver/dns/dns_resolver.go | sed -n &#39;150,250p&#39;&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,120p&#39; internal/resolver/dns/dns_resolver_test.go; sed -n &#39;180,245p&#39; internal/resolver/dns/dns_resolver.go; rg -n \&#34;watcher|RateLimited|ResolveNow|replaceDNSResRate\&#34; internal/resolver/dns/dns_resolver_test.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>11</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w internal/resolver/dns/dns_resolver.go &amp;&amp; \
go test ./internal/resolver/dns&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>13</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>none recorded</td></tr>
<tr><td>14</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -d internal/resolver/dns/dns_resolver.go; git diff --check; git diff -- internal/resolver/dns/dns_resolver.go; git status --short&#39;</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>
