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

* **Date**: 2026-10-06 23:33:14 CEST

## Test case: `grpc__grpc-go-3361`

### Target: `codex/gpt-6-luna/medium (repeat 1)`

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
| **Wall-Clock Latency** | 656.62s | 700.85s | N/A | — | — | — |
| **Process Start → First Event** | 0.15s | 0.29s | — | — | — | — |
| **First Event → First Tool Call** | 4.51s | 6.51s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 9 | 12 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 9 | 9 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 9 | 12 | N/A | — | — | — |
| **Output Tokens** | 1551 | 2680 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 680 | 1220 | N/A | — | — | — |
| **Total Input Tokens** | 323287 | 353099 | N/A | — | — | — |
| **Cached Input Tokens** | 282112 | 321280 | N/A | — | — | — |
| **Uncached Input Tokens** | 41175 | 31819 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.85:1 | 10.10:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; | head -200&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; | head -120&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; find .. -name AGENTS.md -o -name PROJECT_BRIEF.md; git status --short; rg -n \&#34;ResolveNow|ResolveNow\\(|UpdateState|ReportError|connection.*error|resolver\&#34; resolver balancer* pickfirst.go clientconn.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39;; rg -n \&#34;ResolveNow|Refresh|resolveNow|backoff|connection refused|round_robin|roundrobin\&#34; resolver balancer&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; balancer/roundrobin/roundrobin.go; sed -n &#39;1,200p&#39; balancer/roundrobin/roundrobin_test.go; sed -n &#39;1,180p&#39; balancer_conn_wrappers.go; sed -n &#39;1070,1120p&#39; clientconn.go; sed -n &#39;1,180p&#39; resolver/dns/dns_resolver.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; resolver/dns/dns_resolver.go; sed -n &#39;140,205p&#39; resolver/resolver.go; rg -n \&#34;ResolveNow\\(|UpdateState|connectivity.TransientFailure|TransientFailure\&#34; clientconn.go balancer&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func \\(.*resetTransport|ResolveNow|UpdateSubConnState|TransientFailure|ConnectionError\&#34; clientconn.go balancer/base balancer/roundrobin; rg --files internal/resolver/dns; sed -n &#39;1,240p&#39; balancer/base/balancer.go; git log -5 --oneline&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;package dns|func \\(.*ResolveNow|ResolveNow|minResolutionInterval|refresh\&#34; internal/resolver/dns&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1090,1160p&#39; clientconn.go; rg -n \&#34;ResolveNow|lookup|refresh|backoff\&#34; internal/resolver/dns/dns_resolver.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;130,250p&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;390,450p&#39; balancer/balancer.go; sed -n &#39;1,120p&#39; balancer/roundrobin/roundrobin.go&#34;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;ResolveNow|resolveNow|resolveNowThrottle|backoff&#34; resolver_wrapper.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;ResolveNow|resolveNow|SubConnState|HandleSubConnStateChange|UpdateSubConnState&#34; balancer/base clientconn.go&#39;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;ResolveNow|resolveNow|resolveNowThrottle\&#34; --glob &#39;*.go&#39; | head -80; rg --files | rg &#39;resolver.*wrapper&#39; &#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;160,230p&#39; balancer/base/balancer.go; sed -n &#39;1070,1110p&#39; clientconn.go; rg -n \&#34;3165|ResolveNow\&#34;&#34;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;130,180p&#39; resolver_conn_wrapper.go; git show --stat --oneline 648cf9b0; git status --short&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short; git log -8 --oneline -- internal/resolver/dns/dns_resolver.go clientconn.go&#39;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;120,230p&#39; internal/resolver/dns/dns_resolver.go; rg -n \&#34;ResolveNow|resolveNow\&#34; -n internal/resolver/dns/dns_resolver.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;minDNSResRate\&#34; internal/resolver/dns; sed -n &#39;1,140p&#39; internal/resolver/dns/dns_resolver.go; &#34;</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>11</td><td>—</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td></tr>
<tr><td>12</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w internal/resolver/dns/dns_resolver.go &amp;&amp; \
git diff -- internal/resolver/dns/dns_resolver.go&#39;</code></pre></td></tr>
</tbody>
</table>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 656.62s | 609.95s | N/A | — | — | — |
| **Process Start → First Event** | 0.15s | — | — | — | — | — |
| **First Event → First Tool Call** | 4.51s | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 0 | N/A | — | — | — |
| **Internal Tool Cycles** | 9 | 0 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 9 | 0 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 9 | 0 | N/A | — | — | — |
| **Output Tokens** | 1551 | 0 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 680 | 0 | N/A | — | — | — |
| **Total Input Tokens** | 323287 | 0 | N/A | — | — | — |
| **Cached Input Tokens** | 282112 | 0 | N/A | — | — | — |
| **Uncached Input Tokens** | 41175 | 0 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.85:1 | — | — | — | — | — |
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
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; -g &#39;go.mod&#39; | head -200&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd; find .. -name AGENTS.md -o -name PROJECT_BRIEF.md; git status --short; rg -n \&#34;ResolveNow|ResolveNow\\(|UpdateState|ReportError|connection.*error|resolver\&#34; resolver balancer* pickfirst.go clientconn.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; balancer/roundrobin/roundrobin.go; sed -n &#39;1,200p&#39; balancer/roundrobin/roundrobin_test.go; sed -n &#39;1,180p&#39; balancer_conn_wrappers.go; sed -n &#39;1070,1120p&#39; clientconn.go; sed -n &#39;1,180p&#39; resolver/dns/dns_resolver.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;func \\(.*resetTransport|ResolveNow|UpdateSubConnState|TransientFailure|ConnectionError\&#34; clientconn.go balancer/base balancer/roundrobin; rg --files internal/resolver/dns; sed -n &#39;1,240p&#39; balancer/base/balancer.go; git log -5 --oneline&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1090,1160p&#39; clientconn.go; rg -n \&#34;ResolveNow|lookup|refresh|backoff\&#34; internal/resolver/dns/dns_resolver.go&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;ResolveNow|resolveNow|resolveNowThrottle|backoff&#34; resolver_wrapper.go&#39;</code></pre></td><td>none recorded</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;ResolveNow|resolveNow|resolveNowThrottle\&#34; --glob &#39;*.go&#39; | head -80; rg --files | rg &#39;resolver.*wrapper&#39; &#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;130,180p&#39; resolver_conn_wrapper.go; git show --stat --oneline 648cf9b0; git status --short&#34;</code></pre></td><td>none recorded</td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;120,230p&#39; internal/resolver/dns/dns_resolver.go; rg -n \&#34;ResolveNow|resolveNow\&#34; -n internal/resolver/dns/dns_resolver.go&#34;</code></pre></td><td>none recorded</td></tr>
</tbody>
</table>
