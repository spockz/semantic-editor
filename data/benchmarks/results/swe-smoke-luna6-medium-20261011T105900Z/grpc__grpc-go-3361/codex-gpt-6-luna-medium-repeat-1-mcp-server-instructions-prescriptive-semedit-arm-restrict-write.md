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

* **Date**: 2026-10-11 13:20:27 CEST

## Test case: `grpc__grpc-go-3361`

### Target: `codex/gpt-6-luna/medium (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=codex_app_server_v2`

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

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 299.66s | 85.13s | <span class="benchmark-delta-positive">-71.6%</span> | — | — | — |
| **Process Start → First Event** | 0.04s | 0.06s | — | — | — | — |
| **First Event → First Tool Call** | 5.85s | 15.79s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 4 | 1 | <span class="benchmark-delta-positive">-75.0%</span> | — | — | — |
| **Internal Tool Cycles** | 20 | 1 | <span class="benchmark-delta-positive">-95.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 11 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 20 | 1 | <span class="benchmark-delta-positive">-95.0%</span> | — | — | — |
| **Output Tokens** | 5173 | 2043 | <span class="benchmark-delta-positive">-60.5%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 2282 | 18 | <span class="benchmark-delta-positive">-99.2%</span> | — | — | — |
| **Total Input Tokens** | 712677 | 73659 | <span class="benchmark-delta-positive">-89.7%</span> | — | — | — |
| **Cached Input Tokens** | 663296 | 32000 | <span class="benchmark-delta-negative">-95.2%</span> | — | — | — |
| **Uncached Input Tokens** | 49381 | 41659 | <span class="benchmark-delta-positive">-15.6%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 13.43:1 | 0.77:1 | <span class="benchmark-delta-negative">-94.3%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;*dns*&#39; -g &#39;*roundrobin*&#39; | head -80 &amp;&amp; \
git status --short --branch&#34;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
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
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; | head -40; sed -n &#39;1,240p&#39; resolver/dns/dns_resolver.go; sed -n &#39;1,240p&#39; balancer/roundrobin/roundrobin.go; rg -n &#39;ResolveNow|ReportError|ExitIdle|UpdateState|ResolverError|re-resolv|re-resolv&#39; balancer resolver&#34;</code></pre></td><td>—</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;1,280p&#39; balancer/base/balancer.go; rg -n &#39;ResolveNow\\(&#39; --glob &#39;*.go&#39; | head -60; git log -8 --oneline&#34;</code></pre></td><td>—</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;minDNSResRate|ticker|re-resolv|ResolveNow&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;150,260p&#39; balancer_conn_wrappers.go; rg -n &#39;TransientFailure|ResolveNow&#39; balancer_conn_wrappers.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: /* * * Copyright 2017 gRPC authors. * * Licensed under the Apache License, Version 2.0 (the &#34;License&#34;); * you may not use this file except in compliance with the License. * You may obtain a copy of the License at * * http://www.apache.org/licenses/LICENSE-2.0 * * Unless required by applicable law or agreed to in writing, software * distributed under the License is distributed on an &#34;AS IS&#34; BASIS, * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. * See the License for the specific la<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,150p&#39; balancer/balancer.go; sed -n &#39;1,180p&#39; balancer/base/balancer_test.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>7</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- balancer/base/balancer.go internal/resolver/dns/dns_resolver.go &amp;&amp; \
git status --short&#39;</code></pre></td><td>—</td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; resolver_conn_wrapper.go; sed -n &#39;1,260p&#39; balancer/base/balancer.go; rg -n &#39;UpdateClientConnState\\(|ResolveNow\\(&#39; balancer/base&#34;</code></pre></td><td>—</td></tr>
<tr><td>10</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>11</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- balancer/base/balancer.go internal/resolver/dns/dns_resolver.go&#39;</code></pre></td><td>—</td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;ErrBadResolverState|UpdateClientConnState\\(&#39; --glob &#39;*.go&#39; | head -60&#34;</code></pre></td><td>—</td></tr>
<tr><td>14</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;ConnectionError|ResolverState.Err|TransientFailureError|errors.Join|combine&#39; balancer clientconn.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;220,330p&#39; balancer/balancer.go; sed -n &#39;360,415p&#39; balancer/balancer.go; sed -n &#39;90,165p&#39; balancer/base/balancer.go; &#34;</code></pre></td><td>—</td></tr>
<tr><td>17</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;type State struct&#39; resolver/resolver.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>18</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;150,170p&#39; resolver/resolver.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>19</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>20</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w balancer/base/balancer.go &amp;&amp; \
git diff -- balancer/base/balancer.go&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=codex_app_server_v2`

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

| Metric | Vanilla (Small) | Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 299.66s | 299.73s | <span class="benchmark-delta-negative">+0.0%</span> | — | — | — |
| **Process Start → First Event** | 0.04s | 0.06s | — | — | — | — |
| **First Event → First Tool Call** | 5.85s | 3.81s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 4 | 4 | 0% | — | — | — |
| **Internal Tool Cycles** | 20 | 26 | <span class="benchmark-delta-negative">+30.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 11 | 12 | <span class="benchmark-delta-negative">+9.1%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 20 | 26 | <span class="benchmark-delta-negative">+30.0%</span> | — | — | — |
| **Output Tokens** | 5173 | 8229 | <span class="benchmark-delta-negative">+59.1%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 2282 | 4735 | <span class="benchmark-delta-negative">+107.5%</span> | — | — | — |
| **Total Input Tokens** | 712677 | 2130363 | <span class="benchmark-delta-negative">+198.9%</span> | — | — | — |
| **Cached Input Tokens** | 663296 | 2021376 | <span class="benchmark-delta-positive">+204.7%</span> | — | — | — |
| **Uncached Input Tokens** | 49381 | 108987 | <span class="benchmark-delta-negative">+120.7%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 13.43:1 | 18.55:1 | <span class="benchmark-delta-positive">+38.1%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;go.mod&#39; -g &#39;*dns*&#39; -g &#39;*roundrobin*&#39; | head -80 &amp;&amp; \
git status --short --branch&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;*.go&#39; | head -80 &amp;&amp; \
git status --short&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39; -g &#39;README.md&#39; | head -40; sed -n &#39;1,240p&#39; resolver/dns/dns_resolver.go; sed -n &#39;1,240p&#39; balancer/roundrobin/roundrobin.go; rg -n &#39;ResolveNow|ReportError|ExitIdle|UpdateState|ResolverError|re-resolv|re-resolv&#39; balancer resolver&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg --files -g &#39;AGENTS.md&#39; -g &#39;PROJECT_BRIEF.md&#39;; rg -n \&#34;Re-Resolve|ResolveNow|ReportError|roundrobin|DNS\&#34; resolver balancer | head -160; git log --oneline -8&#34;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;1,280p&#39; balancer/base/balancer.go; rg -n &#39;ResolveNow\\(&#39; --glob &#39;*.go&#39; | head -60; git log -8 --oneline&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,280p&#39; resolver/dns/dns_resolver.go; sed -n &#39;130,190p&#39; balancer/balancer.go; sed -n &#39;1,140p&#39; balancer/roundrobin/roundrobin.go; rg -n \&#34;ResolveNow|ReportError|resolver.*Error|failed to connect|TransientFailure\&#34; *.go resolver&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;minDNSResRate|ticker|re-resolv|ResolveNow&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;150,260p&#39; balancer_conn_wrappers.go; rg -n &#39;TransientFailure|ResolveNow&#39; balancer_conn_wrappers.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,300p&#39; internal/resolver/dns/dns_resolver.go; sed -n &#39;110,230p&#39; balancer_conn_wrappers.go; sed -n &#39;800,870p&#39; clientconn.go; sed -n &#39;110,175p&#39; pickfirst.go; rg -n \&#34;ResolveNow\\(\&#34; --glob &#39;*.go&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: /* * * Copyright 2017 gRPC authors. * * Licensed under the Apache License, Version 2.0 (the &#34;License&#34;); * you may not use this file except in compliance with the License. * You may obtain a copy of the License at * * http://www.apache.org/licenses/LICENSE-2.0 * * Unless required by applicable law or agreed to in writing, software * distributed under the License is distributed on an &#34;AS IS&#34; BASIS, * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. * See the License for the specific la<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,150p&#39; balancer/balancer.go; sed -n &#39;1,180p&#39; balancer/base/balancer_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git show --stat --oneline HEAD; git show HEAD&#39;&#39;^:internal/resolver/dns/dns_resolver.go 2&gt;/dev/null | rg -n &#34;watcher|ResolveNow|Timer&#34;; rg -n &#34;UpdateSubConnState&#34; balancer/base&#39;</code></pre></td></tr>
<tr><td>6</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;param \&#34;path\&#34; is required: invalid operation parameters&#34;} param &#34;path&#34; is required: invalid operation parameters<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791716660877355000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;.&#34;,
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791716660877355000&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- balancer/base/balancer.go internal/resolver/dns/dns_resolver.go &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;replacement must preserve the selected function symbol \&#34;dnsResolver.watcher\&#34;&#34;} replacement must preserve the selected function symbol &#34;dnsResolver.watcher&#34;<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791716660877355000&#34;,
  &#34;file&#34;: &#34;internal/resolver/dns/dns_resolver.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;dnsResolver.watcher&#34;,
  &#34;source&#34;: &#34;func (d *dnsResolver) watcher() {\n\tdefer d.wg.Done()\n\tfor {\n\t\tstate, err := d.lookup()\n\t\tif err != nil {\n\t\t\td.cc.ReportError(err)\n\t\t} else {\n\t\t\td.cc.UpdateState(*state)\n\t\t}\n\n\t\tt := time.NewTimer(minDNSResRate)\n\t\tselect {\n\t\tcase &lt;-d.ctx.Done():\n\t\t\tt.Stop()\n\t\t\treturn\n\t\tcase &lt;-d.rn:\n\t\t\tt.Stop()\n\t\tcase &lt;-t.C:\n\t\t}\n\t}\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; resolver_conn_wrapper.go; sed -n &#39;1,260p&#39; balancer/base/balancer.go; rg -n &#39;UpdateClientConnState\\(|ResolveNow\\(&#39; balancer/base&#34;</code></pre></td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791716660877355000&#34;,
  &#34;file&#34;: &#34;internal/resolver/dns/dns_resolver.go&#34;,
  &#34;kind&#34;: &#34;method&#34;,
  &#34;symbol&#34;: &#34;dnsResolver.watcher&#34;,
  &#34;source&#34;: &#34;func (d *dnsResolver) watcher() {\n\tdefer d.wg.Done()\n\tfor {\n\t\tstate, err := d.lookup()\n\t\tif err != nil {\n\t\t\td.cc.ReportError(err)\n\t\t} else {\n\t\t\td.cc.UpdateState(*state)\n\t\t}\n\n\t\tt := time.NewTimer(minDNSResRate)\n\t\tselect {\n\t\tcase &lt;-d.ctx.Done():\n\t\t\tt.Stop()\n\t\t\treturn\n\t\tcase &lt;-d.rn:\n\t\t\tt.Stop()\n\t\tcase &lt;-t.C:\n\t\t}\n\t}\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>10</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;165,215p&#39; balancer/base/balancer.go&#34;</code></pre></td></tr>
<tr><td>11</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/profilerecord compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # cmp compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go to<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./internal/resolver/dns ./balancer/roundrobin&#39;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- balancer/base/balancer.go internal/resolver/dns/dns_resolver.go&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- internal/resolver/dns/dns_resolver.go &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>13</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;ErrBadResolverState|UpdateClientConnState\\(&#39; --glob &#39;*.go&#39; | head -60&#34;</code></pre></td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791716660877355000&#34;,
  &#34;file&#34;: &#34;balancer/base/balancer.go&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;in_function&#34;: &#34;baseBalancer.UpdateSubConnState&#34;,
  &#34;anchor&#34;: &#34;after&#34;,
  &#34;discriminator&#34;: &#34;b.scStates[sc] = s&#34;,
  &#34;source&#34;: &#34;if s == connectivity.TransientFailure &amp;&amp; oldS != connectivity.TransientFailure {\n\t\tb.cc.ResolveNow(resolver.ResolveNowOptions{})\n\t}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>14</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/msan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goarch compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/goos compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/trace/tracev2 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # cmp compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff -- balancer/base/balancer.go internal/resolver/dns/dns_resolver.go &amp;&amp; \
go test ./balancer/base ./balancer/roundrobin ./internal/resolver/dns&#39;</code></pre></td></tr>
<tr><td>15</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;ConnectionError|ResolverState.Err|TransientFailureError|errors.Join|combine&#39; balancer clientconn.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,180p&#39; balancer/base/balancer.go; sed -n &#39;120,185p&#39; resolver_conn_wrapper.go; sed -n &#39;180,245p&#39; resolver_conn_wrapper.go; rg -n \&#34;resolverState|UpdateClientConnState|ResolveNow|resolver.*retry\&#34; balancer/base&#34;</code></pre></td></tr>
<tr><td>16</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;220,330p&#39; balancer/balancer.go; sed -n &#39;360,415p&#39; balancer/balancer.go; sed -n &#39;90,165p&#39; balancer/base/balancer.go; &#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;ErrBadResolverState|updateResolverState&#34; clientconn.go balancer/balancer.go balancer/base&#39;</code></pre></td></tr>
<tr><td>17</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;type State struct&#39; resolver/resolver.go&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: } if cc.dopts.defaultServiceConfig != nil { cc.applyServiceConfigAndBalancer(cc.dopts.defaultServiceConfig, addrs) } else { cc.applyServiceConfigAndBalancer(emptyServiceConfig, addrs) } } func (cc *ClientConn) updateResolverState(s resolver.State, err error) error { defer cc.firstResolveEvent.Fire() cc.mu.Lock() // Check if the ClientConn is already closed. Some fields (e.g. // balancerWrapper) are set to nil when closing the ClientConn, and could // cause nil pointer panic if we don&#39;t have this check. if c<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;575,635p&#39; clientconn.go; sed -n &#39;385,410p&#39; balancer/balancer.go; rg -n \&#34;ErrBadResolverState|resolver.*empty|empty.*address\&#34; balancer/base/*test*&#34;</code></pre></td></tr>
<tr><td>18</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;150,170p&#39; resolver/resolver.go&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg --files balancer/base&#39;</code></pre></td></tr>
<tr><td>19</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791716660877355000&#34;,
  &#34;file&#34;: &#34;balancer/base/balancer.go&#34;,
  &#34;kind&#34;: &#34;statement/expression&#34;,
  &#34;in_function&#34;: &#34;baseBalancer.UpdateClientConnState&#34;,
  &#34;anchor&#34;: &#34;before&#34;,
  &#34;discriminator&#34;: &#34;return nil&#34;,
  &#34;source&#34;: &#34;if len(s.ResolverState.Addresses) == 0 {\n\t\treturn balancer.ErrBadResolverState\n\t}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
<tr><td>20</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w balancer/base/balancer.go &amp;&amp; \
git diff -- balancer/base/balancer.go&#39;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/msan compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # math/bits compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/profilerecord compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/godebugs compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/asan compile: version &#34;go1.27.2&#34; does not match go tool v<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;90,145p&#39; balancer/base/balancer.go; git diff --stat; git diff -- balancer/base/balancer.go; go test ./balancer/base ./balancer/roundrobin ./internal/resolver/dns&#34;</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,260p&#39; balancer/base/balancer.go; sed -n &#39;135,210p&#39; balancer/base/base.go; sed -n &#39;130,175p&#39; balancer_conn_wrappers.go; rg -n \&#34;TransientFailureError|errors.New\\(|fmt.Errorf\\(\&#34; balancer/base&#34;</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;func TransientFailureError|type .*TransientFailure&#34; balancer&#39;</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;290,320p&#39; balancer/balancer.go; head -30 go.mod&#34;</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791716660877355000&#34;,
  &#34;file&#34;: &#34;balancer/base/balancer.go&#34;,
  &#34;kind&#34;: &#34;field&#34;,
  &#34;in_type&#34;: &#34;baseBalancer&#34;,
  &#34;placement&#34;: &#34;type_end&#34;,
  &#34;source&#34;: &#34;resolverErr error&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_insert_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791716660877355000&#34;,
  &#34;file&#34;: &#34;balancer/base/balancer.go&#34;,
  &#34;kind&#34;: &#34;field&#34;,
  &#34;in_type&#34;: &#34;baseBalancer&#34;,
  &#34;placement&#34;: &#34;type_end&#34;,
  &#34;source&#34;: &#34;connectionErr error&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport unknown" title="Transport unknown">?</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-3361_1791716660877355000&#34;,
  &#34;file&#34;: &#34;balancer/base/balancer.go&#34;,
  &#34;kind&#34;: &#34;method&#34;,
  &#34;symbol&#34;: &#34;baseBalancer.ResolverError&#34;,
  &#34;source&#34;: &#34;func (b *baseBalancer) ResolverError(err error) {\n\tb.resolverErr = err\n\tswitch b.state {\n\tcase connectivity.TransientFailure, connectivity.Idle, connectivity.Connecting:\n\t\tif b.picker != nil {\n\t\t\tb.picker = NewErrPicker(b.combinedError(b.connectionErr))\n\t\t} else {\n\t\t\tb.v2Picker = NewErrPickerV2(b.combinedError(b.connectionErr))\n\t\t}\n\t}\n}&#34;,
  &#34;auto_organize_imports&#34;: true
}</code></pre></td></tr>
</tbody>
</table>

