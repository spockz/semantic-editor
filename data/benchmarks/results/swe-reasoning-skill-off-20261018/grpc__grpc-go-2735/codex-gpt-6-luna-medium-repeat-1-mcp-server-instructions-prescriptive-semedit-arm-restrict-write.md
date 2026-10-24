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

### Target: `codex/gpt-6-luna/medium (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

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

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 29.35s | 37.84s | <span class="benchmark-delta-negative">+28.9%</span> | — | — | — |
| **Process Start → First Event** | 0.07s | 0.16s | — | — | — | — |
| **First Event → First Tool Call** | 3.97s | 28.48s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 12 | 1 | <span class="benchmark-delta-positive">-91.7%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 6 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 12 | 1 | <span class="benchmark-delta-positive">-91.7%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.005990 | $0.005617 | -6.2% | — | — | — |
| **Output Tokens** | 2771 | 4605 | <span class="benchmark-delta-negative">+66.2%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 579 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **Total Input Tokens** | 257318 | 80146 | <span class="benchmark-delta-positive">-68.9%</span> | — | — | — |
| **Cached Input Tokens** | 234752 | 52224 | <span class="benchmark-delta-negative">-77.8%</span> | — | — | — |
| **Uncached Input Tokens** | 22566 | 27922 | <span class="benchmark-delta-negative">+23.7%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 10.40:1 | 1.87:1 | <span class="benchmark-delta-negative">-82.0%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | — | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | — | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | — | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | — | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*backoff*&#39; -g &#39;*.go&#39; | head -100 &amp;&amp; \
git status --short --branch&#34;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;ok&#34;,&#34;result&#34;:{&#34;status&#34;:&#34;renamed&#34;,&#34;application_state&#34;:&#34;applied_working&#34;,&#34;continuation_unavailable&#34;:&#34;rename is part of an in-progress workspace batch&#34;,&#34;active_count&#34;:0,&#34;diagnostics&#34;:{&#34;introduced_count&#34;:0,&#34;net_delta&#34;:0,&#34;resolved_count&#34;:0,&#34;su<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: false,
  &#34;trust_workspace&#34;: true,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;symbol&#34;: &#34;factor&#34;,
        &#34;to&#34;: &#34;multiplier&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;const&#34;,
        &#34;group&#34;: &#34;append&#34;,
        &#34;source&#34;: &#34;const maxDelay = 120 * time.Second&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;symbol&#34;: &#34;Exponential&#34;,
        &#34;source&#34;: &#34;type Exponential struct {\n\t// baseDelay is the amount of time to backoff after the first connection\n\t// failure.\n\tbaseDelay time.Duration\n\t// multiplier is the factor with which to multiply backoffs after a failed\n\t// retry.\n\tmultiplier float64\n\t// jitter is the factor with which backoffs are randomized.\n\tjitter float64\n\t// maxDelay is the upper bound of backoff delay.\n\tmaxDelay time.Duration\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;access_modifier&#34;: &#34;public&#34;,
        &#34;source&#34;: &#34;// StrategyBuilder helps build an implementation of the Strategy interface,\n// with setters provided to set individual parameters of the backoff strategy.\ntype StrategyBuilder interface {\n\t// BaseDelay sets the amount of time to backoff after the first connection\n\t// failure.\n\tBaseDelay(time.Duration) StrategyBuilder\n\t// Multiplier sets the the factor with which to multiply backoffs after a\n\t// failed retry.\n\tMultiplier(float64) StrategyBuilder\n\t// Jitter sets the factor with which backoffs are randomized.\n\tJitter(float64) StrategyBuilder\n\t// MaxDelay sets the upper bound of backoff delay.\n\tMaxDelay(time.Duration) StrategyBuilder\n\t// Build builds an implementation of the Strategy interface configured using\n\t// the above setter methods.\n\tBuild() Strategy\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;// exponentialBuilder implements the StrategyBuilder interface for an\n// exponential backoff strategy.\ntype exponentialBuilder struct {\n\tbd     time.Duration\n\tmltpr  float64\n\tjitter float64\n\tmd     time.Duration\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;access_modifier&#34;: &#34;public&#34;,
        &#34;source&#34;: &#34;// NewExponentialBuilder returns an implementation of the StrategyBuilder\n// interface used to build an exponential backoff strategy implementation, as\n// defined in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\nfunc NewExponentialBuilder() StrategyBuilder {\n\treturn &amp;exponentialBuilder{\n\t\tbd:     baseDelay,\n\t\tmltpr:  multiplier,\n\t\tjitter: jitter,\n\t\tmd:     maxDelay,\n\t}\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) BaseDelay(bd time.Duration) StrategyBuilder {\n\teb.bd = bd\n\treturn eb\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) Multiplier(m float64) StrategyBuilder {\n\teb.mltpr = m\n\treturn eb\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) Jitter(j float64) StrategyBuilder {\n\teb.jitter = j\n\treturn eb\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) MaxDelay(md time.Duration) StrategyBuilder {\n\teb.md = md\n\treturn eb\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) Build() Strategy {\n\treturn Exponential{\n\t\tbaseDelay:  eb.bd,\n\t\tmultiplier: eb.mltpr,\n\t\tjitter:     eb.jitter,\n\t\tmaxDelay:   eb.md,\n\t}\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;symbol&#34;: &#34;Exponential.Backoff&#34;,
        &#34;source&#34;: &#34;// Backoff returns the amount of time to wait before the next retry given the\n// number of retries.\n// Helps implement Strategy.\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn bc.baseDelay\n\t}\n\tbackoff, max := float64(bc.baseDelay), float64(bc.maxDelay)\n\tfor backoff &lt; max &amp;&amp; retries &gt; 0 {\n\t\tbackoff *= bc.multiplier\n\t\tretries--\n\t}\n\tif backoff &gt; max {\n\t\tbackoff = max\n\t}\n\t// Randomize backoff delays so that if a cluster of requests start at\n\t// the same time, they won&#39;t operate in lockstep.\n\tbackoff *= 1 + bc.jitter*(grpcrand.Float64()*2-1)\n\tif backoff &lt; 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;symbol&#34;: &#34;BackoffConfig&#34;,
        &#34;source&#34;: &#34;// BackoffConfig defines the parameters for the default gRPC backoff strategy.\n// Deprecated: use ConnectParams instead.\ntype BackoffConfig struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;access_modifier&#34;: &#34;public&#34;,
        &#34;source&#34;: &#34;// ConnectParams defines the parameters for connecting and retrying. Users\n// are encouraged to use this instead of BackoffConfig.\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\n//\n// This API is EXPERIMENTAL.\ntype ConnectParams struct {\n\t// BackoffBaseDelay is the amount of time to backoff after the first\n\t// connection failure.\n\tBackoffBaseDelay time.Duration\n\t// BackoffMultiplier is the factor with which to multiply backoffs after a\n\t// failed retry. Should ideally be greater than 1.\n\tBackoffMultiplier float64\n\t// BackoffJitter is the factor with which backoffs are randomized.\n\tBackoffJitter float64\n\t// BackoffMaxDelay is the upper bound of backoff delay.\n\tBackoffMaxDelay time.Duration\n\t// MinConnectTimeout is the minimum amount of time we are willing to give a\n\t// connection to complete.\n\tMinConnectTimeout time.Duration\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;backoff.go&#34;,
        &#34;kind&#34;: &#34;var&#34;,
        &#34;symbol&#34;: &#34;DefaultBackoffConfig&#34;,
        &#34;source&#34;: &#34;// DefaultBackoffConfig uses values specified for backoff in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\n// Deprecated: use ConnectParams instead.\nvar DefaultBackoffConfig = BackoffConfig{\n\tMaxDelay: 120 * time.Second,\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;kind&#34;: &#34;field&#34;,
        &#34;in_type&#34;: &#34;dialOptions&#34;,
        &#34;placement&#34;: &#34;type_end&#34;,
        &#34;source&#34;: &#34;mct time.Duration&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_remove_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;clientconn.go&#34;,
        &#34;kind&#34;: &#34;const&#34;,
        &#34;symbol&#34;: &#34;minConnectTimeout&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;kind&#34;: &#34;const&#34;,
        &#34;group&#34;: &#34;append&#34;,
        &#34;source&#34;: &#34;const (\n\t// Minimum time to give a connection to complete.\n\tminConnectTimeout = 20 * time.Second\n)&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_insert_function&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;source&#34;: &#34;// WithConnectParams configures the dialer to use the provided backoff\n// parameters for the algorithm defined in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\n// This will override all the default values with the ones provided here. So,\n// use with caution.\n//\n// This API is EXPERIMENTAL.\nfunc WithConnectParams(p ConnectParams) DialOption {\n\tb := backoff.NewExponentialBuilder()\n\tb.BaseDelay(p.BackoffBaseDelay).Multiplier(p.BackoffMultiplier).Jitter(p.BackoffJitter).MaxDelay(p.BackoffMaxDelay)\n\treturn newFuncDialOption(func(o *dialOptions) {\n\t\to.bs = b.Build()\n\t\to.mct = p.MinConnectTimeout\n\t})\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;symbol&#34;: &#34;WithBackoffMaxDelay&#34;,
        &#34;source&#34;: &#34;// WithBackoffMaxDelay configures the dialer to use the provided maximum delay\n// when backing off after failed connection attempts.\n//\n// Deprecated: use WithConnectParams instead.\nfunc WithBackoffMaxDelay(md time.Duration) DialOption {\n\treturn WithBackoffConfig(BackoffConfig{MaxDelay: md})\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;symbol&#34;: &#34;WithBackoffConfig&#34;,
        &#34;source&#34;: &#34;// WithBackoffConfig configures the dialer to use the provided backoff\n// parameters after connection failures.\n//\n// Deprecated: use WithConnectParams instead.\nfunc WithBackoffConfig(b BackoffConfig) DialOption {\n\treturn withBackoff(backoff.NewExponentialBuilder().MaxDelay(b.MaxDelay).Build())\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;symbol&#34;: &#34;defaultDialOptions&#34;,
        &#34;body&#34;: &#34;return dialOptions{\n\tdisableRetry:    !envconfig.Retry,\n\treqHandshake:    envconfig.RequireHandshake,\n\thealthCheckFunc: internal.HealthCheckFunc,\n\tcopts: transport.ConnectOptions{\n\t\tWriteBufferSize: defaultWriteBufSize,\n\t\tReadBufferSize:  defaultReadBufSize,\n\t},\n\tmct: minConnectTimeout,\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;clientconn.go&#34;,
        &#34;kind&#34;: &#34;if&#34;,
        &#34;in_function&#34;: &#34;DialContext&#34;,
        &#34;discriminator&#34;: &#34;cc.dopts.bs == nil&#34;,
        &#34;source&#34;: &#34;if cc.dopts.bs == nil {\n\tcc.dopts.bs = backoff.NewExponentialBuilder().Build()\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;clientconn.go&#34;,
        &#34;kind&#34;: &#34;statement/expression&#34;,
        &#34;in_function&#34;: &#34;addrConn.resetTransport&#34;,
        &#34;discriminator&#34;: &#34;dialDuration := minConnectTimeout&#34;,
        &#34;source&#34;: &#34;dialDuration := ac.dopts.mct&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer/grpclb/grpclb.go&#34;,
        &#34;symbol&#34;: &#34;defaultBackoffConfig&#34;,
        &#34;to&#34;: &#34;defaultBackoffStrategy&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_decl&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer/grpclb/grpclb.go&#34;,
        &#34;symbol&#34;: &#34;defaultBackoffStrategy&#34;,
        &#34;source&#34;: &#34;// defaultBackoffStrategy configures the backoff strategy that&#39;s used when the\n// init handshake in the RPC is unsuccessful. It&#39;s not for the clientconn\n// reconnect backoff.\n//\n// It has the same value as the default grpc.DefaultBackoffConfig.\n//\n// TODO: make backoff configurable.\nvar defaultBackoffStrategy = backoff.NewExponentialBuilder().Build()&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer/xds/xds_client.go&#34;,
        &#34;symbol&#34;: &#34;defaultBackoffConfig&#34;,
        &#34;to&#34;: &#34;defaultBackoffStrategy&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;balancer/xds/xds_client.go&#34;,
        &#34;kind&#34;: &#34;var&#34;,
        &#34;symbol&#34;: &#34;defaultBackoffStrategy&#34;,
        &#34;source&#34;: &#34;var defaultBackoffStrategy = backoff.NewExponentialBuilder().Build()&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;health/client.go&#34;,
        &#34;kind&#34;: &#34;var&#34;,
        &#34;symbol&#34;: &#34;backoffStrategy&#34;,
        &#34;source&#34;: &#34;var backoffStrategy = backoff.NewExponentialBuilder().Build()&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_remove_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;health/client.go&#34;,
        &#34;kind&#34;: &#34;const&#34;,
        &#34;symbol&#34;: &#34;maxDelay&#34;,
        &#34;safe_delete&#34;: false,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;resolver/dns/dns_resolver.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;symbol&#34;: &#34;dnsResolver&#34;,
        &#34;source&#34;: &#34;// dnsResolver watches for the name resolution update for a non-IP target.\ntype dnsResolver struct {\n\tfreq       time.Duration\n\tbackoff    backoff.Strategy\n\tretryCount int\n\thost       string\n\tport       string\n\tresolver   netResolver\n\tctx        context.Context\n\tcancel     context.CancelFunc\n\tcc         resolver.ClientConn\n\t// rn channel is used by ResolveNow() to force an immediate resolution of the target.\n\trn chan struct{}\n\tt  *time.Timer\n\t// wg is used to enforce Close() to return after the watcher() goroutine has finished.\n\t// Otherwise, data race will be possible. [Race Example] in dns_resolver_test we\n\t// replace the real lookup functions with mocked ones to facilitate testing.\n\t// If Close() doesn&#39;t wait for watcher() goroutine finishes, race detector sometimes\n\t// will warns lookup (READ the lookup function pointers) inside watcher() goroutine\n\t// has data race with replaceNetFunc (WRITE the lookup function pointers).\n\twg                   sync.WaitGroup\n\tdisableServiceConfig bool\n}&#34;,
        &#34;auto_organize_imports&#34;: false
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_body&#34;,
      &#34;params&#34;: {
        &#34;file&#34;: &#34;resolver/dns/dns_resolver.go&#34;,
        &#34;symbol&#34;: &#34;(*dnsBuilder).Build&#34;,
        &#34;body&#34;: &#34;\thost, port, err := parseTarget(target.Endpoint, defaultPort)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\t// IP address.\n\tif net.ParseIP(host) != nil {\n\t\thost, _ = formatIP(host)\n\t\taddr := []resolver.Address{{Addr: host + \&#34;:\&#34; + port}}\n\t\ti := &amp;ipResolver{\n\t\t\tcc: cc,\n\t\t\tip: addr,\n\t\t\trn: make(chan struct{}, 1),\n\t\t\tq:  make(chan struct{}),\n\t\t}\n\t\tcc.NewAddress(addr)\n\t\tgo i.watcher()\n\t\treturn i, nil\n\t}\n\n\t// DNS address (non-IP).\n\tctx, cancel := context.WithCancel(context.Background())\n\td := &amp;dnsResolver{\n\t\tfreq:                 b.minFreq,\n\t\tbackoff:              backoff.NewExponentialBuilder().MaxDelay(b.minFreq).Build(),\n\t\thost:                 host,\n\t\tport:                 port,\n\t\tctx:                  ctx,\n\t\tcancel:               cancel,\n\t\tcc:                   cc,\n\t\tt:                    time.NewTimer(0),\n\t\trn:                   make(chan struct{}, 1),\n\t\tdisableServiceConfig: opts.DisableServiceConfig,\n\t}\n\n\tif target.Authority == \&#34;\&#34; {\n\t\td.resolver = defaultResolver\n\t} else {\n\t\td.resolver, err = customAuthorityResolver(target.Authority)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t}\n\n\td.wg.Add(1)\n\tgo d.watcher()\n\treturn d, nil&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find .. -name AGENTS.md -o -name PROJECT_BRIEF.md | head -30 &amp;&amp; \
rg -n \&#34;BackoffConfig|backoffConfig|connection backoff|ConnectParams\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39; | head -150&#34;</code></pre></td><td>—</td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; backoff.go &amp;&amp; \
sed -n &#39;200,280p&#39; clientconn.go &amp;&amp; \
sed -n &#39;220,280p&#39; dialoptions.go &amp;&amp; \
rg -n \&#34;backoffStrategy|backoff.*Config|DefaultBackoffConfig\&#34; --glob &#39;*.go&#39;&#34;</code></pre></td><td>—</td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;type Exponential|func \\(.*Exponential|package backoff\&#34; internal/backoff &amp;&amp; \
cat go.mod | head -10 &amp;&amp; \
sed -n &#39;810,855p&#39; clientconn_test.go&#34;</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat internal/backoff/backoff.go &amp;&amp; \
sed -n &#39;855,875p&#39; clientconn_test.go &amp;&amp; \
rg -n \&#34;Jitter|Multiplier|InitialBackoff\&#34;&#34;</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;backoff\\.Exponential\\{|MaxDelay:\&#34; --glob &#39;*.go&#39; | head -100&#34;</code></pre></td><td>—</td></tr>
<tr><td>7</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>8</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # cmp compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not ma<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w backoff.go internal/backoff/backoff.go dialoptions.go clientconn.go clientconn_test.go &amp;&amp; \
go test . ./internal/backoff&#39;</code></pre></td><td>—</td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git diff --stat &amp;&amp; \
git diff -- backoff.go internal/backoff/backoff.go dialoptions.go clientconn.go clientconn_test.go &amp;&amp; \
go version &amp;&amp; \
which go&#39;</code></pre></td><td>—</td></tr>
<tr><td>11</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td>—</td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,fixture_storage=tmpfs,module_cache=/Users/alessandro/sources/semantic-editor/.scratch/go/mod,module_cache_network=offline,module_cache_preparation=go-mod-download-all-before-offline-execution,provider_transport=codex_app_server_v2`

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
| **Wall-Clock Latency** | 29.35s | 119.51s | <span class="benchmark-delta-negative">+307.2%</span> | — | — | — |
| **Process Start → First Event** | 0.07s | 0.07s | — | — | — | — |
| **First Event → First Tool Call** | 3.97s | 3.40s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 12 | 28 | <span class="benchmark-delta-negative">+133.3%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 6 | 3 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 12 | 28 | <span class="benchmark-delta-negative">+133.3%</span> | — | — | — |
| **Estimated Cloud Token Cost (USD, list-rate equivalent)** | $0.005990 | $0.033232 | +454.8% | — | — | — |
| **Output Tokens** | 2771 | 9377 | <span class="benchmark-delta-negative">+238.4%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 579 | 4757 | <span class="benchmark-delta-negative">+721.6%</span> | — | — | — |
| **Total Input Tokens** | 257318 | 2077139 | <span class="benchmark-delta-negative">+707.2%</span> | — | — | — |
| **Cached Input Tokens** | 234752 | 1990784 | <span class="benchmark-delta-positive">+748.0%</span> | — | — | — |
| **Uncached Input Tokens** | 22566 | 86355 | <span class="benchmark-delta-negative">+282.7%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 10.40:1 | 23.05:1 | <span class="benchmark-delta-positive">+121.6%</span> | — | — | — |
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
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*backoff*&#39; -g &#39;*.go&#39; | head -100 &amp;&amp; \
git status --short --branch&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*backoff*&#39; -g &#39;*.go&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;find .. -name AGENTS.md -o -name PROJECT_BRIEF.md | head -30 &amp;&amp; \
rg -n \&#34;BackoffConfig|backoffConfig|connection backoff|ConnectParams\&#34; --glob &#39;*.go&#39; --glob &#39;*.md&#39; | head -150&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;type BackoffConfig|BackoffConfig|connection-backoff|backoff&#34;&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;1,220p&#39; backoff.go &amp;&amp; \
sed -n &#39;200,280p&#39; clientconn.go &amp;&amp; \
sed -n &#39;220,280p&#39; dialoptions.go &amp;&amp; \
rg -n \&#34;backoffStrategy|backoff.*Config|DefaultBackoffConfig\&#34; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat backoff.go &amp;&amp; \
sed -n &#39;1,115p&#39; internal/backoff/backoff.go &amp;&amp; \
sed -n &#39;230,280p&#39; dialoptions.go&#34;</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;type Exponential|func \\(.*Exponential|package backoff\&#34; internal/backoff &amp;&amp; \
cat go.mod | head -10 &amp;&amp; \
sed -n &#39;810,855p&#39; clientconn_test.go&#34;</code></pre></td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: ./reflection/grpc_testing/test.pb.go:172:// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream. ./reflection/grpc_reflection_v1alpha/reflection.pb.go:794:// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream. ./credentials/alts/internal/proto/grpc_gcp/handshaker.pb.go:1011:// For semantics around ctx use and closing/ending strea<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;semedit|semantic\&#34; . --glob &#39;&#34;&#39;!vendor/**&#39;&#34;&#39; | head -80; rg --files -g &#39;PROJECT_BRIEF.md&#39; -g &#39;AGENTS.md&#39;&#34;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat internal/backoff/backoff.go &amp;&amp; \
sed -n &#39;855,875p&#39; clientconn_test.go &amp;&amp; \
rg -n \&#34;Jitter|Multiplier|InitialBackoff\&#34;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short&#39;</code></pre></td></tr>
<tr><td>6</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n \&#34;backoff\\.Exponential\\{|MaxDelay:\&#34; --glob &#39;*.go&#39; | head -100&#34;</code></pre></td><td><code>semedit/semantic_workspace_outline</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;path&#34;: &#34;&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;930,990p&#39; clientconn.go; cat .git/config | head -40; git status --short&#34;</code></pre></td></tr>
<tr><td>8</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;rg -n &#34;minConnectTimeout|WithMinConnect&#34; dialoptions.go clientconn.go backoff.go&#39;</code></pre></td></tr>
<tr><td>9</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: # internal/unsafeheader compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/byteorder compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/coverage/rtcov compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # cmp compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # unicode/utf8 compile: version &#34;go1.27.2&#34; does not match go tool version &#34;go1.27.1&#34; # internal/runtime/pprof/label compile: version &#34;go1.27.2&#34; does not ma<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;gofmt -w backoff.go internal/backoff/backoff.go dialoptions.go clientconn.go clientconn_test.go &amp;&amp; \
go test . ./internal/backoff&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;backoff.go&#34;,
  &#34;symbol&#34;: &#34;BackoffConfig&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git diff --stat &amp;&amp; \
git diff -- backoff.go internal/backoff/backoff.go dialoptions.go clientconn.go clientconn_test.go &amp;&amp; \
go version &amp;&amp; \
which go&#39;</code></pre></td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
  &#34;symbol&#34;: &#34;Exponential&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><code>file_change</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;45,75p&#39; dialoptions.go; sed -n &#39;220,245p&#39; clientconn.go&#34;</code></pre></td></tr>
<tr><td>12</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git diff --check &amp;&amp; \
git status --short&#39;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Verification failed: verify bash source &#34;balancer/grpclb/regenerate.sh&#34;: workspace trust required: verify: bash: /Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000 Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;applied_with_diagnostics&#34;,&#34;verification&#34;:{&#34;fail<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;file&#34;: &#34;backoff.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;type&#34;,
      &#34;symbol&#34;: &#34;BackoffConfig&#34;,
      &#34;source&#34;: &#34;type BackoffConfig struct {\n\t// BaseDelay is the amount of time to wait before retrying after the first\n\t// failure.\n\tBaseDelay time.Duration\n\t// Multiplier is the factor applied to the backoff after each retry.\n\tMultiplier float64\n\t// Jitter provides a range to randomize backoff delays.\n\tJitter float64\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n}&#34;,
      &#34;auto_organize_imports&#34;: true
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;type&#34;,
      &#34;symbol&#34;: &#34;Exponential&#34;,
      &#34;source&#34;: &#34;type Exponential struct {\n\t// BaseDelay is the amount of time to wait before retrying after the first\n\t// failure.\n\tBaseDelay time.Duration\n\t// Multiplier is the factor applied to the backoff after each retry.\n\tMultiplier float64\n\t// Jitter provides a range to randomize backoff delays.\n\tJitter float64\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n}&#34;,
      &#34;auto_organize_imports&#34;: true
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;function&#34;,
      &#34;symbol&#34;: &#34;Backoff&#34;,
      &#34;source&#34;: &#34;func (bc Exponential) Backoff(retries int) time.Duration {\n\tbaseDelayValue := bc.BaseDelay\n\tif baseDelayValue &lt;= 0 {\n\t\tbaseDelayValue = baseDelay\n\t}\n\tmultiplier := bc.Multiplier\n\tif multiplier &lt;= 0 {\n\t\tmultiplier = factor\n\t}\n\tjitterValue := bc.Jitter\n\tif bc.BaseDelay == 0 &amp;&amp; bc.Multiplier == 0 &amp;&amp; bc.Jitter == 0 {\n\t\tjitterValue = jitter\n\t}\n\tbackoff, max := float64(baseDelayValue), float64(bc.MaxDelay)\n\tfor backoff &lt; max &amp;&amp; retries &gt; 0 {\n\t\tbackoff *= multiplier\n\t\tretries--\n\t}\n\tif backoff &gt; max {\n\t\tbackoff = max\n\t}\n\tbackoff *= 1 + jitterValue*(grpcrand.Float64()*2-1)\n\tif backoff &lt; 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;,
      &#34;auto_organize_imports&#34;: true
    }
  ]
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --stat &amp;&amp; \
git diff -- backoff.go internal/backoff/backoff.go | head -180; sed -n &#39;25,55p&#39; backoff.go; sed -n &#39;255,273p&#39; dialoptions.go&#34;</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;820,865p&#39; clientconn_test.go&#34;</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Batch child results are intermediate and do not provide final continuation targets. Critical result (JSON): {&#34;diagnostic_report&#34;:{&#34;status&#34;:&#34;unavailable&#34;,&#34;reason&#34;:&#34;No completed final batch diagnostic inventory was captured.&#34;},&#34;result&#34;:{&#34;status&#34;:&#34;error&#34;,&#34;results&#34;:[{&#34;status&#34;:&#34;error&#34;,&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;edit 3 params: param \&#34;source\&#34; is required: invalid operation parameters&#34;,&#34;tool&#34;:&#34;semantic_replace_construct&#34;}],&#34;workspace_root&#34;:&#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-im<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;file&#34;: &#34;backoff.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;var&#34;,
      &#34;symbol&#34;: &#34;DefaultBackoffConfig&#34;,
      &#34;source&#34;: &#34;var DefaultBackoffConfig = BackoffConfig{\n\tBaseDelay: 1 * time.Second,\n\tMultiplier: 1.6,\n\tJitter: 0.2,\n\tMaxDelay: 120 * time.Second,\n}&#34;,
      &#34;auto_organize_imports&#34;: true
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;file&#34;: &#34;dialoptions.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;function&#34;,
      &#34;symbol&#34;: &#34;WithBackoffConfig&#34;,
      &#34;source&#34;: &#34;func WithBackoffConfig(b BackoffConfig) DialOption {\n\treturn withBackoff(backoff.Exponential{\n\t\tBaseDelay: b.BaseDelay,\n\t\tMultiplier: b.Multiplier,\n\t\tJitter: b.Jitter,\n\t\tMaxDelay: b.MaxDelay,\n\t})\n}&#34;,
      &#34;auto_organize_imports&#34;: true
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;file&#34;: &#34;dialoptions.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;function&#34;,
      &#34;symbol&#34;: &#34;WithBackoffMaxDelay&#34;,
      &#34;source&#34;: &#34;func WithBackoffMaxDelay(md time.Duration) DialOption {\n\treturn WithBackoffConfig(BackoffConfig{MaxDelay: md})\n}&#34;,
      &#34;auto_organize_imports&#34;: true
    },
    {
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;,
      &#34;file&#34;: &#34;clientconn.go&#34;,
      &#34;language&#34;: &#34;go&#34;,
      &#34;kind&#34;: &#34;statement/expression&#34;,
      &#34;symbol&#34;: &#34;&#34;,
      &#34;in_function&#34;: &#34;ClientConn&#34;,
      &#34;source&#34;: &#34;&#34;,
      &#34;auto_organize_imports&#34;: true
    }
  ]
}</code></pre></td></tr>
<tr><td>16</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git diff --stat &amp;&amp; \
sed -n &#39;25,45p&#39; backoff.go; sed -n &#39;255,275p&#39; dialoptions.go&#34;</code></pre></td></tr>
<tr><td>17</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;backoff.go&#34;,
  &#34;kind&#34;: &#34;var&#34;,
  &#34;symbol&#34;: &#34;DefaultBackoffConfig&#34;,
  &#34;source&#34;: &#34;var DefaultBackoffConfig = BackoffConfig{\n\tBaseDelay: 1 * time.Second,\n\tMultiplier: 1.6,\n\tJitter: 0.2,\n\tMaxDelay: 120 * time.Second,\n}&#34;
}</code></pre></td></tr>
<tr><td>18</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;dialoptions.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;WithBackoffConfig&#34;,
  &#34;source&#34;: &#34;func WithBackoffConfig(b BackoffConfig) DialOption {\n\treturn withBackoff(backoff.Exponential{\n\t\tBaseDelay: b.BaseDelay,\n\t\tMultiplier: b.Multiplier,\n\t\tJitter: b.Jitter,\n\t\tMaxDelay: b.MaxDelay,\n\t})\n}&#34;
}</code></pre></td></tr>
<tr><td>19</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;dialoptions.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;WithBackoffConfig&#34;,
  &#34;source&#34;: &#34;func WithBackoffConfig(b BackoffConfig) DialOption {\n\treturn withBackoff(backoff.Exponential{\n\t\tBaseDelay: b.BaseDelay,\n\t\tMultiplier: b.Multiplier,\n\t\tJitter: b.Jitter,\n\t\tMaxDelay: b.MaxDelay,\n\t})\n}&#34;
}</code></pre></td></tr>
<tr><td>20</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn_test.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;testBackoffConfigSet&#34;,
  &#34;source&#34;: &#34;func testBackoffConfigSet(t *testing.T, expected *BackoffConfig, opts ...DialOption) {\n\topts = append(opts, WithInsecure())\n\tconn, err := Dial(\&#34;passthrough:///foo:80\&#34;, opts...)\n\tif err != nil {\n\t\tt.Fatalf(\&#34;unexpected error dialing connection: %v\&#34;, err)\n\t}\n\tdefer conn.Close()\n\n\tif conn.dopts.bs == nil {\n\t\tt.Fatalf(\&#34;backoff config not set\&#34;)\n\t}\n\n\tactual, ok := conn.dopts.bs.(backoff.Exponential)\n\tif !ok {\n\t\tt.Fatalf(\&#34;unexpected type of backoff config: %#v\&#34;, conn.dopts.bs)\n\t}\n\n\texpectedValue := backoff.Exponential{\n\t\tBaseDelay: expected.BaseDelay,\n\t\tMultiplier: expected.Multiplier,\n\t\tJitter: expected.Jitter,\n\t\tMaxDelay: expected.MaxDelay,\n\t}\n\tif actual != expectedValue {\n\t\tt.Fatalf(\&#34;unexpected backoff config on connection: %v, want %v\&#34;, actual, expected)\n\t}\n}&#34;
}</code></pre></td></tr>
<tr><td>21</td><td>—</td><td><code>semedit/semantic_inspect_symbol</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;resolve NewClient in /Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000/clientconn.go: symbol not found&#34;}<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn.go&#34;,
  &#34;symbol&#34;: &#34;NewClient&#34;,
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;
}</code></pre></td></tr>
<tr><td>22</td><td>—</td><td><span role="img" aria-label="Transport failed" title="Transport failed">✗</span> <span role="img" aria-label="Functional unknown" title="Functional unknown">?</span>: 103:func Dial(target string, opts ...DialOption) (*ClientConn, error) { 123:func DialContext(ctx context.Context, target string, opts ...DialOption) (conn *ClientConn, err error) { 342:func (csm *connectivityStateManager) updateState(state connectivity.State) { 365:func (csm *connectivityStateManager) getState() connectivity.State { 371:func (csm *connectivityStateManager) getNotifyChan() &lt;-chan struct{} { 413:func (cc *ClientConn) WaitForStateChange(ctx context.Context, sourceState connectivity.State) bool<pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;rg -n &#39;&#34;&#39;^func &#39;&#34;&#39; clientconn.go | head -30; sed -n &#39;230,245p&#39; clientconn.go; rg -n &#39;&#34;&#39;^func TestWithBackoff&#39;&#34;&#39; clientconn_test.go&#34;</code></pre></td></tr>
<tr><td>23</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn_test.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;TestWithBackoffConfig&#34;,
  &#34;source&#34;: &#34;func (s) TestWithBackoffConfig(t *testing.T) {\n\tb := BackoffConfig{\n\t\tBaseDelay: 500 * time.Millisecond,\n\t\tMultiplier: 1.2,\n\t\tJitter: 0.1,\n\t\tMaxDelay: DefaultBackoffConfig.MaxDelay / 2,\n\t}\n\texpected := b\n\ttestBackoffConfigSet(t, &amp;expected, WithBackoffConfig(b))\n}&#34;
}</code></pre></td></tr>
<tr><td>24</td><td>—</td><td><code>semedit/semantic_set_doc_comment</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;dialoptions.go&#34;,
  &#34;scope&#34;: &#34;declaration&#34;,
  &#34;symbol&#34;: &#34;WithBackoffConfig&#34;,
  &#34;comment&#34;: &#34;WithBackoffConfig configures the dialer to use the provided backoff\nparameters after connection failures.&#34;
}</code></pre></td></tr>
<tr><td>25</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;clientconn_test.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;TestWithBackoffConfigDefault&#34;,
  &#34;source&#34;: &#34;func (s) TestWithBackoffConfigDefault(t *testing.T) {\n\texpected := BackoffConfig{MaxDelay: DefaultBackoffConfig.MaxDelay}\n\ttestBackoffConfigSet(t, &amp;expected)\n}&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>semedit/semantic_replace_construct</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;workspace_root&#34;: &#34;/Volumes/semedit-bench-ram/run_codex_semedit-agentic_small-implicit_grpc__grpc-go-2735_1792300196768082000&#34;,
  &#34;language&#34;: &#34;go&#34;,
  &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
  &#34;kind&#34;: &#34;function&#34;,
  &#34;symbol&#34;: &#34;Backoff&#34;,
  &#34;source&#34;: &#34;func (bc Exponential) Backoff(retries int) time.Duration {\n\tbaseDelayValue := bc.BaseDelay\n\tif baseDelayValue &lt;= 0 {\n\t\tbaseDelayValue = baseDelay\n\t}\n\tmultiplier := bc.Multiplier\n\tif multiplier &lt;= 0 {\n\t\tmultiplier = factor\n\t}\n\tjitterValue := bc.Jitter\n\tif bc.BaseDelay == 0 &amp;&amp; bc.Multiplier == 0 &amp;&amp; bc.Jitter == 0 {\n\t\tjitterValue = jitter\n\t}\n\tif retries == 0 {\n\t\treturn baseDelayValue\n\t}\n\tbackoff, max := float64(baseDelayValue), float64(bc.MaxDelay)\n\tfor backoff &lt; max &amp;&amp; retries &gt; 0 {\n\t\tbackoff *= multiplier\n\t\tretries--\n\t}\n\tif backoff &gt; max {\n\t\tbackoff = max\n\t}\n\tbackoff *= 1 + jitterValue*(grpcrand.Float64()*2-1)\n\tif backoff &lt; 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;
}</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;git status --short &amp;&amp; \
git diff --stat &amp;&amp; \
git diff -- backoff.go dialoptions.go internal/backoff/backoff.go clientconn_test.go&#39;</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;sed -n &#39;510,540p&#39; dialoptions.go; rg -n &#39;WithBackoffConfig|BaseDelay|Multiplier|Jitter&#39; --glob &#39;*.go&#39; | head -60&#34;</code></pre></td></tr>
</tbody>
</table>

