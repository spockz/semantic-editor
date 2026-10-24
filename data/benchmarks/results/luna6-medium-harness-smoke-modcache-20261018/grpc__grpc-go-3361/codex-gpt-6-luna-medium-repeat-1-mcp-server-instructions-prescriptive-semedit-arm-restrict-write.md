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

* **Date**: 2026-10-18 08:08:54 CEST

## Test case: `grpc__grpc-go-3361`

### Target: `codex/gpt-6-luna/medium (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=disk,module_cache=/Users/alessandro/go/pkg/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

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

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 18.97s | 27.02s | <span class="benchmark-delta-negative">+42.4%</span> | — | — | — |
| **Process Start → First Event** | 0.07s | 0.18s | — | — | — | — |
| **First Event → First Tool Call** | 3.39s | 14.41s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 8 | 1 | <span class="benchmark-delta-positive">-87.5%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 8 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 1 | <span class="benchmark-delta-positive">-87.5%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.006398 | $0.004132 | -35.4% | — | — | — |
| **Output Tokens** | 919 | 2066 | <span class="benchmark-delta-negative">+124.8%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 308 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **Total Input Tokens** | 204423 | 73615 | <span class="benchmark-delta-positive">-64.0%</span> | — | — | — |
| **Cached Input Tokens** | 161152 | 47360 | <span class="benchmark-delta-negative">-70.6%</span> | — | — | — |
| **Uncached Input Tokens** | 43271 | 26255 | <span class="benchmark-delta-positive">-39.3%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 3.72:1 | 1.80:1 | <span class="benchmark-delta-negative">-51.6%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; | head -80 &amp;&amp; \
git status --short &amp;&amp; \
git log -5 --oneline&#34;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: true,
  &#34;workspace_root&#34;: &#34;balancer/base&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer.go&#34;,
        &#34;in_type&#34;: &#34;baseBalancer&#34;,
        &#34;kind&#34;: &#34;field&#34;,
        &#34;placement&#34;: &#34;type_end&#34;,
        &#34;source&#34;: &#34;resolverErr error // the last error reported by the resolver; cleared on successful resolution&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer.go&#34;,
        &#34;in_type&#34;: &#34;baseBalancer&#34;,
        &#34;kind&#34;: &#34;field&#34;,
        &#34;placement&#34;: &#34;type_end&#34;,
        &#34;source&#34;: &#34;connErr error // the last connection error; cleared upon leaving TransientFailure&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// mergeErrors builds an error from the last connection error and the last\n// resolver error.  Must only be called if b.state is TransientFailure.\nfunc (b *baseBalancer) mergeErrors() error {\n\t// connErr must always be non-nil unless there are no SubConns, in which\n\t// case resolverErr must be non-nil.\n\tif b.connErr == nil {\n\t\treturn fmt.Errorf(\&#34;last resolver error: %v\&#34;, b.resolverErr)\n\t}\n\tif b.resolverErr == nil {\n\t\treturn fmt.Errorf(\&#34;last connection error: %v\&#34;, b.connErr)\n\t}\n\treturn fmt.Errorf(\&#34;last connection error: %v; last resolver error: %v\&#34;, b.connErr, b.resolverErr)\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer.go&#34;,
        &#34;symbol&#34;: &#34;baseBalancer.ResolverError&#34;,
        &#34;body&#34;: &#34;b.resolverErr = err\nif len(b.subConns) == 0 {\n\tb.state = connectivity.TransientFailure\n}\nif b.state != connectivity.TransientFailure {\n\t// The picker will not change since the balancer does not currently\n\t// report an error.\n\treturn\n}\nb.regeneratePicker()\nif b.picker != nil {\n\tb.cc.UpdateBalancerState(b.state, b.picker)\n} else {\n\tb.cc.UpdateState(balancer.State{\n\t\tConnectivityState: b.state,\n\t\tPicker:            b.v2Picker,\n\t})\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer.go&#34;,
        &#34;symbol&#34;: &#34;baseBalancer.UpdateClientConnState&#34;,
        &#34;body&#34;: &#34;// TODO: handle s.ResolverState.Err (log if not nil) once implemented.\n// TODO: handle s.ResolverState.ServiceConfig?\nif grpclog.V(2) {\n\tgrpclog.Infoln(\&#34;base.baseBalancer: got new ClientConn state: \&#34;, s)\n}\nif len(s.ResolverState.Addresses) == 0 {\n\tb.ResolverError(errors.New(\&#34;produced zero addresses\&#34;))\n\treturn balancer.ErrBadResolverState\n}\n// Successful resolution; clear resolver error and ensure we return nil.\nb.resolverErr = nil\n// addrsSet is the set converted from addrs, it&#39;s used for quick lookup of an address.\naddrsSet := make(map[resolver.Address]struct{})\nfor _, a := range s.ResolverState.Addresses {\n\taddrsSet[a] = struct{}{}\n\tif _, ok := b.subConns[a]; !ok {\n\t\t// a is a new address (not existing in b.subConns).\n\t\tsc, err := b.cc.NewSubConn([]resolver.Address{a}, balancer.NewSubConnOptions{HealthCheckEnabled: b.config.HealthCheck})\n\t\tif err != nil {\n\t\t\tgrpclog.Warningf(\&#34;base.baseBalancer: failed to create new SubConn: %v\&#34;, err)\n\t\t\tcontinue\n\t\t}\n\t\tb.subConns[a] = sc\n\t\tb.scStates[sc] = connectivity.Idle\n\t\tsc.Connect()\n\t}\n}\nfor a, sc := range b.subConns {\n\t// a was removed by resolver.\n\tif _, ok := addrsSet[a]; !ok {\n\t\tb.cc.RemoveSubConn(sc)\n\t\tdelete(b.subConns, a)\n\t\t// Keep the state of this sc in b.scStates until sc&#39;s state becomes Shutdown.\n\t\t// The entry will be deleted in HandleSubConnStateChange.\n\t}\n}\nreturn nil&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;symbol&#34;: &#34;baseBalancer.regeneratePicker&#34;,
        &#34;source&#34;: &#34;// regeneratePicker takes a snapshot of the balancer, and generates a picker\n// from it. The picker is\n//  - errPicker if the balancer is in TransientFailure,\n//  - built by the pickerBuilder with all READY SubConns otherwise.\nfunc (b *baseBalancer) regeneratePicker() {\n\tif b.state == connectivity.TransientFailure {\n\t\tif b.pickerBuilder != nil {\n\t\t\tb.picker = NewErrPicker(balancer.ErrTransientFailure)\n\t\t} else {\n\t\t\tb.v2Picker = NewErrPickerV2(balancer.TransientFailureError(b.mergeErrors()))\n\t\t}\n\t\treturn\n\t}\n\tif b.pickerBuilder != nil {\n\t\treadySCs := make(map[resolver.Address]balancer.SubConn)\n\n\t\t// Filter out all ready SCs from full subConn map.\n\t\tfor addr, sc := range b.subConns {\n\t\t\tif st, ok := b.scStates[sc]; ok &amp;&amp; st == connectivity.Ready {\n\t\t\t\treadySCs[addr] = sc\n\t\t\t}\n\t\t}\n\t\tb.picker = b.pickerBuilder.Build(readySCs)\n\t} else {\n\t\treadySCs := make(map[balancer.SubConn]SubConnInfo)\n\n\t\t// Filter out all ready SCs from full subConn map.\n\t\tfor addr, sc := range b.subConns {\n\t\t\tif st, ok := b.scStates[sc]; ok &amp;&amp; st == connectivity.Ready {\n\t\t\t\treadySCs[sc] = SubConnInfo{Address: addr}\n\t\t\t}\n\t\t}\n\t\tb.v2Picker = b.v2PickerBuilder.Build(PickerBuildInfo{ReadySCs: readySCs})\n\t}\n}&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer.go&#34;,
        &#34;symbol&#34;: &#34;baseBalancer.UpdateSubConnState&#34;,
        &#34;body&#34;: &#34;s := state.ConnectivityState\nif grpclog.V(2) {\n\tgrpclog.Infof(\&#34;base.baseBalancer: handle SubConn state change: %p, %v\&#34;, sc, s)\n}\noldS, ok := b.scStates[sc]\nif !ok {\n\tif grpclog.V(2) {\n\t\tgrpclog.Infof(\&#34;base.baseBalancer: got state changes for an unknown SubConn: %p, %v\&#34;, sc, s)\n\t}\n\treturn\n}\nb.scStates[sc] = s\nswitch s {\ncase connectivity.Idle:\n\tsc.Connect()\ncase connectivity.Shutdown:\n\t// When an address was removed by resolver, b called RemoveSubConn but\n\t// kept the sc&#39;s state in scStates. Remove state for this sc here.\n\tdelete(b.scStates, sc)\n}\n\noldAggrState := b.state\nb.state = b.csEvltr.RecordTransition(oldS, s)\n\n// Set or clear the last connection error accordingly.\nb.connErr = state.ConnectionError\n\n// Regenerate picker when one of the following happens:\n//  - this sc became ready from not-ready\n//  - this sc became not-ready from ready\n//  - the aggregated state of balancer became TransientFailure from non-TransientFailure\n//  - the aggregated state of balancer became non-TransientFailure from TransientFailure\nif (s == connectivity.Ready) != (oldS == connectivity.Ready) ||\n\t(b.state == connectivity.TransientFailure) != (oldAggrState == connectivity.TransientFailure) {\n\tb.regeneratePicker()\n}\n\nif b.picker != nil {\n\tb.cc.UpdateBalancerState(b.state, b.picker)\n} else {\n\tb.cc.UpdateState(balancer.State{ConnectivityState: b.state, Picker: b.v2Picker})\n}&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;ResolveNow|ResolveNow|ResolveNow\\(|ResolveNow|ResolveNow|ResolveNow|ResolveNow|ResolveNow|ResolveNow|ResolveNow|ResolveNow\&#34; resolver balancer&#34;</code></pre></td><td>—</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat resolver/dns/dns_resolver.go &amp;&amp; \
rg -n &#34;roundrobin|RoundRobin|ResolveNow&#34; balancer&#39;</code></pre></td><td>—</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;package dns|ResolveNow|MinResolutionInterval|refresh\&#34; internal/resolver/dns &amp;&amp; \
rg -n \&#34;Connect\\(|ResolveNow|addrConn\&#34; clientconn.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1080,1175p&#39; clientconn.go &amp;&amp; \
sed -n &#39;150,250p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
sed -n &#39;370,425p&#39; balancer/balancer.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;30,110p&#39; balancer/roundrobin/roundrobin.go; rg -n \&#34;HandleSubConnStateChange|TransientFailure\&#34; balancer&#34;</code></pre></td><td>—</td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;170,230p&#39; balancer/base/balancer.go; git log --oneline --all -- internal/resolver/dns/dns_resolver.go | head&#34;</code></pre></td><td>—</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;105,160p&#39; internal/resolver/dns/dns_resolver.go; git show 648cf9b0:clientconn.go | sed -n &#39;1090,1110p&#39; &#34;</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=disk,module_cache=/Users/alessandro/go/pkg/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

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

| Metric | Vanilla (Small) | Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 18.97s | 40.24s | <span class="benchmark-delta-negative">+112.1%</span> | — | — | — |
| **Process Start → First Event** | 0.07s | 0.18s | — | — | — | — |
| **First Event → First Tool Call** | 3.39s | 3.62s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 8 | 11 | <span class="benchmark-delta-negative">+37.5%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 8 | 2 | <span class="benchmark-delta-positive">-75.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 11 | <span class="benchmark-delta-negative">+37.5%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.006398 | $0.009875 | +54.3% | — | — | — |
| **Output Tokens** | 919 | 1756 | <span class="benchmark-delta-negative">+91.1%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 308 | 647 | <span class="benchmark-delta-negative">+110.1%</span> | — | — | — |
| **Total Input Tokens** | 204423 | 510910 | <span class="benchmark-delta-negative">+149.9%</span> | — | — | — |
| **Cached Input Tokens** | 161152 | 467712 | <span class="benchmark-delta-positive">+190.2%</span> | — | — | — |
| **Uncached Input Tokens** | 43271 | 43198 | <span class="benchmark-delta-positive">-0.2%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 3.72:1 | 10.83:1 | <span class="benchmark-delta-positive">+190.7%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; | head -80 &amp;&amp; \
git status --short &amp;&amp; \
git log -5 --oneline&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; | head -80 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;ResolveNow|ResolveNow|ResolveNow\\(|ResolveNow|ResolveNow|ResolveNow|ResolveNow|ResolveNow|ResolveNow|ResolveNow|ResolveNow\&#34; resolver balancer&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;for f in AGENTS.md PROJECT_BRIEF.md; do if [ -f &#34;$f&#34; ]; then cat &#34;$f&#34;; fi; done; rg -n &#34;ResolveNow|ResolveNow&#39;&#34;\\(|ReportError|rr|round_robin|WithBalancerName|Refresh\&#34; resolver balancer.go clientconn.go&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat resolver/dns/dns_resolver.go &amp;&amp; \
rg -n &#34;roundrobin|RoundRobin|ResolveNow&#34; balancer&#39;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;resolver&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;package dns|ResolveNow|MinResolutionInterval|refresh\&#34; internal/resolver/dns &amp;&amp; \
rg -n \&#34;Connect\\(|ResolveNow|addrConn\&#34; clientconn.go&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;resolver/dns/dns_resolver.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1080,1175p&#39; clientconn.go &amp;&amp; \
sed -n &#39;150,250p&#39; internal/resolver/dns/dns_resolver.go &amp;&amp; \
sed -n &#39;370,425p&#39; balancer/balancer.go&#34;</code></pre></td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;clientconn.go&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;30,110p&#39; balancer/roundrobin/roundrobin.go; rg -n \&#34;HandleSubConnStateChange|TransientFailure\&#34; balancer&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat resolver/dns/dns_resolver.go; sed -n &#39;1070,1120p&#39; clientconn.go; sed -n &#39;940,970p&#39; clientconn.go; git log -8 --oneline -- resolver/dns/dns_resolver.go&#34;</code></pre></td></tr>
<tr><td>7</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;170,230p&#39; balancer/base/balancer.go; git log --oneline --all -- internal/resolver/dns/dns_resolver.go | head&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;ResolveNow|resolveNow|refresh|MinResolutionInterval|Backoff&#34; internal/resolver/dns internal/resolver balancer_conn_wrappers.go&#39;</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;105,160p&#39; internal/resolver/dns/dns_resolver.go; git show 648cf9b0:clientconn.go | sed -n &#39;1090,1110p&#39; &#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; internal/resolver/dns/dns_resolver.go;&#34;</code></pre></td></tr>
<tr><td>9</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;file&#34;: &#34;internal/resolver/dns/dns_resolver.go&#34;,
  &#34;symbol&#34;: &#34;dnsResolver.watcher&#34;,
  &#34;language&#34;: &#34;go&#34;
}</code></pre></td></tr>
<tr><td>10</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;internal/resolver/dns/dns_resolver.go&#34;,
  &#34;kind&#34;: &#34;method&#34;,
  &#34;symbol&#34;: &#34;dnsResolver.watcher&#34;,
  &#34;source&#34;: &#34;func (d *dnsResolver) watcher() {\n\tdefer d.wg.Done()\n\tfor {\n\t\tselect {\n\t\tcase &lt;-d.ctx.Done():\n\t\t\treturn\n\t\tcase &lt;-d.rn:\n\t\t}\n\n\t\tstate, err := d.lookup()\n\t\tif err != nil {\n\t\t\td.cc.ReportError(err)\n\t\t} else {\n\t\t\td.cc.UpdateState(*state)\n\t\t}\n\n\t\t// Wait before resolving again, even when no explicit request arrives.\n\t\tt := time.NewTimer(minDNSResRate)\n\t\tselect {\n\t\tcase &lt;-t.C:\n\t\t\td.ResolveNow(resolver.ResolveNowOptions{})\n\t\tcase &lt;-d.ctx.Done():\n\t\t\tt.Stop()\n\t\t\treturn\n\t\t}\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>11</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # math/bits compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool ver<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./internal/resolver/dns ./...&#39;</code></pre></td></tr>
</tbody>
</table>

