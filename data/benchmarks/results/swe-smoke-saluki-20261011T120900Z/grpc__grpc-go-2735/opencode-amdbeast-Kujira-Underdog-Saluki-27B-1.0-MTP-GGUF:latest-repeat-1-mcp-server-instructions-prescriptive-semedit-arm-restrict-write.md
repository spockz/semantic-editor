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

## Test case: `grpc__grpc-go-2735`

### Target: `opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=opencode_attached_server_v1`

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
| **Wall-Clock Latency** | 299.81s | 176.93s | <span class="benchmark-delta-positive">-41.0%</span> | — | — | — |
| **Process Start → First Event** | 12.58s | 35.50s | — | — | — | — |
| **First Event → First Tool Call** | 2.32s | 125.61s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 31 | 1 | <span class="benchmark-delta-positive">-96.8%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 10 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 31 | 1 | <span class="benchmark-delta-positive">-96.8%</span> | — | — | — |
| **Output Tokens** | 11570 | 4181 | <span class="benchmark-delta-positive">-63.9%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 468473 | 63622 | <span class="benchmark-delta-positive">-86.4%</span> | — | — | — |
| **Cached Input Tokens** | 453059 | 29459 | <span class="benchmark-delta-negative">-93.5%</span> | — | — | — |
| **Uncached Input Tokens** | 15414 | 34163 | <span class="benchmark-delta-negative">+121.6%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 29.39:1 | 0.86:1 | <span class="benchmark-delta-negative">-97.1%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | — | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | — | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | — | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | — | ❌ FAIL | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/backoff*.go&#34;
}</code></pre></td><td><code>semedit_semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_grpc__grpc-go-2735_1791720824982408000\&#34;: check Go diagnostics with gopls: load Go diagnostics session scope: err: exit status 1: stderr: go: cloud.google.com/go@v0.26.0: Get \&#34;https://proxy.golang.org/cloud.google.com/go/@v/v0.26.0.mod\&#34;: dial tcp: lookup proxy.golang.org: no such host\<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: false,
  &#34;edits&#34;: [
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;symbol&#34;: &#34;factor&#34;,
        &#34;to&#34;: &#34;multiplier&#34;
      },
      &#34;tool&#34;: &#34;semantic_rename&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;group&#34;: &#34;append&#34;,
        &#34;kind&#34;: &#34;const&#34;,
        &#34;source&#34;: &#34;const maxDelay = 120 * time.Second&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;type Exponential struct {\n\t// baseDelay is the amount of time to backoff after the first connection\n\t// failure.\n\tbaseDelay time.Duration\n\t// multiplier is the factor with which to multiply backoffs after a failed\n\t// retry.\n\tmultiplier float64\n\t// jitter is the factor with which backoffs are randomized.\n\tjitter float64\n\t// maxDelay is the upper bound of backoff delay.\n\tmaxDelay time.Duration\n}&#34;,
        &#34;symbol&#34;: &#34;Exponential&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;access_modifier&#34;: &#34;public&#34;,
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;// StrategyBuilder helps build an implementation of the Strategy interface,\n// with setters provided to set individual parameters of the backoff strategy.\ntype StrategyBuilder interface {\n\t// BaseDelay sets the amount of time to backoff after the first connection\n\t// failure.\n\tBaseDelay(time.Duration) StrategyBuilder\n\t// Multiplier sets the the factor with which to multiply backoffs after a\n\t// failed retry.\n\tMultiplier(float64) StrategyBuilder\n\t// Jitter sets the factor with which backoffs are randomized.\n\tJitter(float64) StrategyBuilder\n\t// MaxDelay sets the upper bound of backoff delay.\n\tMaxDelay(time.Duration) StrategyBuilder\n\t// Build builds an implementation of the Strategy interface configured using\n\t// the above setter methods.\n\tBuild() Strategy\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;// exponentialBuilder implements the StrategyBuilder interface for an\n// exponential backoff strategy.\ntype exponentialBuilder struct {\n\tbd     time.Duration\n\tmltpr  float64\n\tjitter float64\n\tmd     time.Duration\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;access_modifier&#34;: &#34;public&#34;,
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;source&#34;: &#34;// NewExponentialBuilder returns an implementation of the StrategyBuilder\n// interface used to build an exponential backoff strategy implementation, as\n// defined in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\nfunc NewExponentialBuilder() StrategyBuilder {\n\treturn \u0026exponentialBuilder{\n\t\tbd:     baseDelay,\n\t\tmltpr:  multiplier,\n\t\tjitter: jitter,\n\t\tmd:     maxDelay,\n\t}\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) BaseDelay(bd time.Duration) StrategyBuilder {\n\teb.bd = bd\n\treturn eb\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) Multiplier(m float64) StrategyBuilder {\n\teb.mltpr = m\n\treturn eb\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) Jitter(j float64) StrategyBuilder {\n\teb.jitter = j\n\treturn eb\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) MaxDelay(md time.Duration) StrategyBuilder {\n\teb.md = md\n\treturn eb\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// BaseDelay helps implement StrategyBuilder.\nfunc (eb *exponentialBuilder) Build() Strategy {\n\treturn Exponential{\n\t\tbaseDelay:  eb.bd,\n\t\tmultiplier: eb.mltpr,\n\t\tjitter:     eb.jitter,\n\t\tmaxDelay:   eb.md,\n\t}\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;internal/backoff/backoff.go&#34;,
        &#34;kind&#34;: &#34;method&#34;,
        &#34;source&#34;: &#34;// Backoff returns the amount of time to wait before the next retry given the\n// number of retries.\n// Helps implement Strategy.\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn bc.baseDelay\n\t}\n\tbackoff, max := float64(bc.baseDelay), float64(bc.maxDelay)\n\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\n\t\tbackoff *= bc.multiplier\n\t\tretries--\n\t}\n\tif backoff \u003e max {\n\t\tbackoff = max\n\t}\n\t// Randomize backoff delays so that if a cluster of requests start at\n\t// the same time, they won&#39;t operate in lockstep.\n\tbackoff *= 1 + bc.jitter*(grpcrand.Float64()*2-1)\n\tif backoff \u003c 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;,
        &#34;symbol&#34;: &#34;Exponential.Backoff&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;// BackoffConfig defines the parameters for the default gRPC backoff strategy.\n// Deprecated: use ConnectParams instead.\ntype BackoffConfig struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n}&#34;,
        &#34;symbol&#34;: &#34;BackoffConfig&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;access_modifier&#34;: &#34;public&#34;,
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;backoff.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;// ConnectParams defines the parameters for connecting and retrying. Users\n// are encouraged to use this instead of BackoffConfig.\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\n//\n// This API is EXPERIMENTAL.\ntype ConnectParams struct {\n\t// BackoffBaseDelay is the amount of time to backoff after the first\n\t// connection failure.\n\tBackoffBaseDelay time.Duration\n\t// BackoffMultiplier is the factor with which to multiply backoffs after a\n\t// failed retry. Should ideally be greater than 1.\n\tBackoffMultiplier float64\n\t// BackoffJitter is the factor with which backoffs are randomized.\n\tBackoffJitter float64\n\t// BackoffMaxDelay is the upper bound of backoff delay.\n\tBackoffMaxDelay time.Duration\n\t// MinConnectTimeout is the minimum amount of time we are willing to give a\n\t// connection to complete.\n\tMinConnectTimeout time.Duration\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;backoff.go&#34;,
        &#34;kind&#34;: &#34;var&#34;,
        &#34;source&#34;: &#34;// DefaultBackoffConfig uses values specified for backoff in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\n// Deprecated: use ConnectParams instead.\nvar DefaultBackoffConfig = BackoffConfig{\n\tMaxDelay: 120 * time.Second,\n}&#34;,
        &#34;symbol&#34;: &#34;DefaultBackoffConfig&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;in_type&#34;: &#34;dialOptions&#34;,
        &#34;kind&#34;: &#34;field&#34;,
        &#34;placement&#34;: &#34;type_end&#34;,
        &#34;source&#34;: &#34;mct time.Duration&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;clientconn.go&#34;,
        &#34;kind&#34;: &#34;const&#34;,
        &#34;symbol&#34;: &#34;minConnectTimeout&#34;
      },
      &#34;tool&#34;: &#34;semantic_remove_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;group&#34;: &#34;append&#34;,
        &#34;kind&#34;: &#34;const&#34;,
        &#34;source&#34;: &#34;const (\n\t// Minimum time to give a connection to complete.\n\tminConnectTimeout = 20 * time.Second\n)&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;source&#34;: &#34;// WithConnectParams configures the dialer to use the provided backoff\n// parameters for the algorithm defined in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\n// This will override all the default values with the ones provided here. So,\n// use with caution.\n//\n// This API is EXPERIMENTAL.\nfunc WithConnectParams(p ConnectParams) DialOption {\n\tb := backoff.NewExponentialBuilder()\n\tb.BaseDelay(p.BackoffBaseDelay).Multiplier(p.BackoffMultiplier).Jitter(p.BackoffJitter).MaxDelay(p.BackoffMaxDelay)\n\treturn newFuncDialOption(func(o *dialOptions) {\n\t\to.bs = b.Build()\n\t\to.mct = p.MinConnectTimeout\n\t})\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;source&#34;: &#34;// WithBackoffMaxDelay configures the dialer to use the provided maximum delay\n// when backing off after failed connection attempts.\n//\n// Deprecated: use WithConnectParams instead.\nfunc WithBackoffMaxDelay(md time.Duration) DialOption {\n\treturn WithBackoffConfig(BackoffConfig{MaxDelay: md})\n}&#34;,
        &#34;symbol&#34;: &#34;WithBackoffMaxDelay&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;source&#34;: &#34;// WithBackoffConfig configures the dialer to use the provided backoff\n// parameters after connection failures.\n//\n// Deprecated: use WithConnectParams instead.\nfunc WithBackoffConfig(b BackoffConfig) DialOption {\n\treturn withBackoff(backoff.NewExponentialBuilder().MaxDelay(b.MaxDelay).Build())\n}&#34;,
        &#34;symbol&#34;: &#34;WithBackoffConfig&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;body&#34;: &#34;return dialOptions{\n\tdisableRetry:    !envconfig.Retry,\n\treqHandshake:    envconfig.RequireHandshake,\n\thealthCheckFunc: internal.HealthCheckFunc,\n\tcopts: transport.ConnectOptions{\n\t\tWriteBufferSize: defaultWriteBufSize,\n\t\tReadBufferSize:  defaultReadBufSize,\n\t},\n\tmct: minConnectTimeout,\n}&#34;,
        &#34;file&#34;: &#34;dialoptions.go&#34;,
        &#34;symbol&#34;: &#34;defaultDialOptions&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_body&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;discriminator&#34;: &#34;cc.dopts.bs == nil&#34;,
        &#34;file&#34;: &#34;clientconn.go&#34;,
        &#34;in_function&#34;: &#34;DialContext&#34;,
        &#34;kind&#34;: &#34;if&#34;,
        &#34;source&#34;: &#34;if cc.dopts.bs == nil {\n\tcc.dopts.bs = backoff.NewExponentialBuilder().Build()\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;discriminator&#34;: &#34;dialDuration := minConnectTimeout&#34;,
        &#34;file&#34;: &#34;clientconn.go&#34;,
        &#34;in_function&#34;: &#34;addrConn.resetTransport&#34;,
        &#34;kind&#34;: &#34;statement/expression&#34;,
        &#34;source&#34;: &#34;dialDuration := ac.dopts.mct&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;balancer/grpclb/grpclb.go&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;symbol&#34;: &#34;defaultBackoffConfig&#34;,
        &#34;to&#34;: &#34;defaultBackoffStrategy&#34;
      },
      &#34;tool&#34;: &#34;semantic_rename&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;balancer/grpclb/grpclb.go&#34;,
        &#34;source&#34;: &#34;// defaultBackoffStrategy configures the backoff strategy that&#39;s used when the\n// init handshake in the RPC is unsuccessful. It&#39;s not for the clientconn\n// reconnect backoff.\n//\n// It has the same value as the default grpc.DefaultBackoffConfig.\n//\n// TODO: make backoff configurable.\nvar defaultBackoffStrategy = backoff.NewExponentialBuilder().Build()&#34;,
        &#34;symbol&#34;: &#34;defaultBackoffStrategy&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_decl&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;balancer/xds/xds_client.go&#34;,
        &#34;language&#34;: &#34;go&#34;,
        &#34;symbol&#34;: &#34;defaultBackoffConfig&#34;,
        &#34;to&#34;: &#34;defaultBackoffStrategy&#34;
      },
      &#34;tool&#34;: &#34;semantic_rename&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;balancer/xds/xds_client.go&#34;,
        &#34;kind&#34;: &#34;var&#34;,
        &#34;source&#34;: &#34;var defaultBackoffStrategy = backoff.NewExponentialBuilder().Build()&#34;,
        &#34;symbol&#34;: &#34;defaultBackoffStrategy&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;health/client.go&#34;,
        &#34;kind&#34;: &#34;var&#34;,
        &#34;source&#34;: &#34;var backoffStrategy = backoff.NewExponentialBuilder().Build()&#34;,
        &#34;symbol&#34;: &#34;backoffStrategy&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;health/client.go&#34;,
        &#34;kind&#34;: &#34;const&#34;,
        &#34;safe_delete&#34;: false,
        &#34;symbol&#34;: &#34;maxDelay&#34;
      },
      &#34;tool&#34;: &#34;semantic_remove_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;file&#34;: &#34;resolver/dns/dns_resolver.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;// dnsResolver watches for the name resolution update for a non-IP target.\ntype dnsResolver struct {\n\tfreq       time.Duration\n\tbackoff    backoff.Strategy\n\tretryCount int\n\thost       string\n\tport       string\n\tresolver   netResolver\n\tctx        context.Context\n\tcancel     context.CancelFunc\n\tcc         resolver.ClientConn\n\t// rn channel is used by ResolveNow() to force an immediate resolution of the target.\n\trn chan struct{}\n\tt  *time.Timer\n\t// wg is used to enforce Close() to return after the watcher() goroutine has finished.\n\t// Otherwise, data race will be possible. [Race Example] in dns_resolver_test we\n\t// replace the real lookup functions with mocked ones to facilitate testing.\n\t// If Close() doesn&#39;t wait for watcher() goroutine finishes, race detector sometimes\n\t// will warns lookup (READ the lookup function pointers) inside watcher() goroutine\n\t// has data race with replaceNetFunc (WRITE the lookup function pointers).\n\twg                   sync.WaitGroup\n\tdisableServiceConfig bool\n}&#34;,
        &#34;symbol&#34;: &#34;dnsResolver&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: false,
        &#34;body&#34;: &#34;\thost, port, err := parseTarget(target.Endpoint, defaultPort)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\t// IP address.\n\tif net.ParseIP(host) != nil {\n\t\thost, _ = formatIP(host)\n\t\taddr := []resolver.Address{{Addr: host + \&#34;:\&#34; + port}}\n\t\ti := \u0026ipResolver{\n\t\t\tcc: cc,\n\t\t\tip: addr,\n\t\t\trn: make(chan struct{}, 1),\n\t\t\tq:  make(chan struct{}),\n\t\t}\n\t\tcc.NewAddress(addr)\n\t\tgo i.watcher()\n\t\treturn i, nil\n\t}\n\n\t// DNS address (non-IP).\n\tctx, cancel := context.WithCancel(context.Background())\n\td := \u0026dnsResolver{\n\t\tfreq:                 b.minFreq,\n\t\tbackoff:              backoff.NewExponentialBuilder().MaxDelay(b.minFreq).Build(),\n\t\thost:                 host,\n\t\tport:                 port,\n\t\tctx:                  ctx,\n\t\tcancel:               cancel,\n\t\tcc:                   cc,\n\t\tt:                    time.NewTimer(0),\n\t\trn:                   make(chan struct{}, 1),\n\t\tdisableServiceConfig: opts.DisableServiceConfig,\n\t}\n\n\tif target.Authority == \&#34;\&#34; {\n\t\td.resolver = defaultResolver\n\t} else {\n\t\td.resolver, err = customAuthorityResolver(target.Authority)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t}\n\n\td.wg.Add(1)\n\tgo d.watcher()\n\treturn d, nil&#34;,
        &#34;file&#34;: &#34;resolver/dns/dns_resolver.go&#34;,
        &#34;symbol&#34;: &#34;(*dnsBuilder).Build&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_body&#34;
    }
  ],
  &#34;trust_workspace&#34;: true
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;BackoffConfig&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>3</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/backoff.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>4</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/internal/backoff/backoff.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/dialoptions.go&#34;,
  &#34;limit&#34;: 80,
  &#34;offset&#34;: 230
}</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -10 \u0026\u0026 git status&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>7</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/clientconn_test.go&#34;,
  &#34;limit&#34;: 60,
  &#34;offset&#34;: 810
}</code></pre></td><td>—</td></tr>
<tr><td>8</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;pattern&#34;: &#34;baseDelay|factor|jitter&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>9</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;internal/backoff/*&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>10</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/clientconn.go&#34;,
  &#34;limit&#34;: 25,
  &#34;offset&#34;: 225
}</code></pre></td><td>—</td></tr>
<tr><td>11</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/backoff.go&#34;,
  &#34;newString&#34;: &#34;// DefaultBackoffConfig uses values specified for backoff in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\nvar DefaultBackoffConfig = BackoffConfig{\n\tMaxDelay:      120 * time.Second,\n\tBaseDelay:     1 * time.Second,\n\tJitter:        0.2,\n\tBackoffFactor: 1.6,\n}\n\n// BackoffConfig defines the parameters for the default gRPC backoff strategy.\ntype BackoffConfig struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n\t// BaseDelay is the base delay for backoff.\n\tBaseDelay time.Duration\n\t// Jitter is the randomize factor for backoff delay.\n\tJitter float64\n\t// BackoffFactor is the factor for backoff delay.\n\tBackoffFactor float64\n}&#34;,
  &#34;oldString&#34;: &#34;// DefaultBackoffConfig uses values specified for backoff in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\nvar DefaultBackoffConfig = BackoffConfig{\n\tMaxDelay: 120 * time.Second,\n}\n\n// BackoffConfig defines the parameters for the default gRPC backoff strategy.\ntype BackoffConfig struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n}&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>12</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/internal/backoff/backoff.go&#34;,
  &#34;newString&#34;: &#34;// Exponential implements exponential backoff algorithm as defined in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\ntype Exponential struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n\t// BaseDelay is the base delay for backoff.\n\tBaseDelay time.Duration\n\t// Jitter is the randomize factor for backoff delay.\n\tJitter float64\n\t// BackoffFactor is the factor for backoff delay.\n\tBackoffFactor float64\n}\n\n// Backoff returns the amount of time to wait before the next retry given the\n// number of retries.\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn bc.BaseDelay\n\t}\n\tbackoff, max := float64(bc.BaseDelay), float64(bc.MaxDelay)\n\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\n\t\tbackoff *= bc.BackoffFactor\n\t\tretries--\n\t}\n\tif backoff \u003e max {\n\t\tbackoff = max\n\t}\n\t// Randomize backoff delays so that if a cluster of requests start at\n\t// the same time, they won&#39;t operate in lockstep.\n\tbackoff *= 1 + bc.Jitter*(grpcrand.Float64()*2-1)\n\tif backoff \u003c 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;,
  &#34;oldString&#34;: &#34;const (\n\t// baseDelay is the amount of time to wait before retrying after the first\n\t// failure.\n\tbaseDelay = 1.0 * time.Second\n\t// factor is applied to the backoff after each retry.\n\tfactor = 1.6\n\t// jitter provides a range to randomize backoff delays.\n\tjitter = 0.2\n)\n\n// Exponential implements exponential backoff algorithm as defined in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\ntype Exponential struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n}\n\n// Backoff returns the amount of time to wait before the next retry given the\n// number of retries.\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn baseDelay\n\t}\n\tbackoff, max := float64(baseDelay), float64(bc.MaxDelay)\n\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\n\t\tbackoff *= factor\n\t\tretries--\n\t}\n\tif backoff \u003e max {\n\t\tbackoff = max\n\t}\n\t// Randomize backoff delays so that if a cluster of requests start at\n\t// the same time, they won&#39;t operate in lockstep.\n\tbackoff *= 1 + jitter*(grpcrand.Float64()*2-1)\n\tif backoff \u003c 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>13</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/dialoptions.go&#34;,
  &#34;newString&#34;: &#34;// WithBackoffConfig configures the dialer to use the provided backoff\n// parameters after connection failures.\nfunc WithBackoffConfig(b BackoffConfig) DialOption {\n\treturn withBackoff(backoff.Exponential{\n\t\tMaxDelay:      b.MaxDelay,\n\t\tBaseDelay:     b.BaseDelay,\n\t\tJitter:        b.Jitter,\n\t\tBackoffFactor: b.BackoffFactor,\n\t})\n}&#34;,
  &#34;oldString&#34;: &#34;// WithBackoffConfig configures the dialer to use the provided backoff\n// parameters after connection failures.\n//\n// Use WithBackoffMaxDelay until more parameters on BackoffConfig are opened up\n// for use.\nfunc WithBackoffConfig(b BackoffConfig) DialOption {\n\treturn withBackoff(backoff.Exponential{\n\t\tMaxDelay: b.MaxDelay,\n\t})\n}&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>14</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/clientconn.go&#34;,
  &#34;newString&#34;: &#34;\tif cc.dopts.bs == nil {\n\t\tcc.dopts.bs = backoff.Exponential{\n\t\t\tMaxDelay:      DefaultBackoffConfig.MaxDelay,\n\t\t\tBaseDelay:     DefaultBackoffConfig.BaseDelay,\n\t\t\tJitter:        DefaultBackoffConfig.Jitter,\n\t\t\tBackoffFactor: DefaultBackoffConfig.BackoffFactor,\n\t\t}\n\t}&#34;,
  &#34;oldString&#34;: &#34;\tif cc.dopts.bs == nil {\n\t\tcc.dopts.bs = backoff.Exponential{\n\t\t\tMaxDelay: DefaultBackoffConfig.MaxDelay,\n\t\t}\n\t}&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>15</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/clientconn_test.go&#34;,
  &#34;newString&#34;: &#34;\texpectedValue := backoff.Exponential{\n\t\tMaxDelay:      expected.MaxDelay,\n\t\tBaseDelay:     expected.BaseDelay,\n\t\tJitter:        expected.Jitter,\n\t\tBackoffFactor: expected.BackoffFactor,\n\t}\n\tif actual != expectedValue {\n\t\tt.Fatalf(\&#34;unexpected backoff config on connection: %v, want %v\&#34;, actual, expected)\n\t}&#34;,
  &#34;oldString&#34;: &#34;\texpectedValue := backoff.Exponential{\n\t\tMaxDelay: expected.MaxDelay,\n\t}\n\tif actual != expectedValue {\n\t\tt.Fatalf(\&#34;unexpected backoff config on connection: %v, want %v\&#34;, actual, expected)\n\t}&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>16</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;pattern&#34;: &#34;backoff.Exponential|DefaultBackoffConfig&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>17</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go version \u0026\u0026 go build ./...&#34;,
  &#34;timeout&#34;: 300000,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>18</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls vendor 2\u003e/dev/null; go env GOPROXY GOFLAGS GOMODCACHE; ls $(go env GOMODCACHE) 2\u003e/dev/null | head&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>19</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/go/mod&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>20</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/go.mod&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>21</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -d /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/*/vendor 2\u003e/dev/null; ls /Users/alessandro/go/pkg 2\u003e/dev/null | head; ls ~/go 2\u003e/dev/null | head&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>22</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;nslookup proxy.golang.org 2\u003e\u00261 | head -5; nslookup github.com 2\u003e\u00261 | head -5&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>23</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls go.sum 2\u003e/dev/null; ls internal/grpcrand/&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>24</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./internal/backoff/ 2\u003e\u00261 | head -20&#34;,
  &#34;timeout&#34;: 120000,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>25</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;gofmt -l backoff.go dialoptions.go clientconn.go clientconn_test.go internal/backoff/backoff.go&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>26</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;gofmt -d internal/backoff/backoff.go | head -40&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>27</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which -a git 2\u003e/dev/null; ls /opt/homebrew/bin/git /usr/local/bin/git 2\u003e/dev/null&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>28</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;gofmt -d clientconn.go | head -60&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>29</td><td><code>write</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;content&#34;: &#34;package main\n\nimport (\n\t\&#34;fmt\&#34;\n\t\&#34;math/rand\&#34;\n\t\&#34;testing\&#34;\n\t\&#34;time\&#34;\n)\n\ntype Exponential struct {\n\tMaxDelay      time.Duration\n\tBaseDelay     time.Duration\n\tJitter        float64\n\tBackoffFactor float64\n}\n\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn bc.BaseDelay\n\t}\n\tbackoff, max := float64(bc.BaseDelay), float64(bc.MaxDelay)\n\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\n\t\tbackoff *= bc.BackoffFactor\n\t\tretries--\n\t}\n\tif backoff \u003e max {\n\t\tbackoff = max\n\t}\n\tbackoff *= 1 + bc.Jitter*(rand.Float64()*2-1)\n\tif backoff \u003c 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}\n\nfunc TestDefault(t *testing.T) {\n\t// Default values must match old behavior: base 1s, factor 1.6, jitter 0.2, max 120s.\n\tbc := Exponential{\n\t\tMaxDelay:      120 * time.Second,\n\t\tBaseDelay:     1 * time.Second,\n\t\tJitter:        0.2,\n\t\tBackoffFactor: 1.6,\n\t}\n\tif d := bc.Backoff(0); d != time.Second {\n\t\tt.Fatalf(\&#34;Backoff(0) = %v, want %v\&#34;, d, time.Second)\n\t}\n\tif d := bc.Backoff(1); d != time.Second {\n\t\tt.Fatalf(\&#34;Backoff(1) = %v, want %v\&#34;, d, time.Second)\n\t}\n\t// After many retries, delay must be bounded by MaxDelay (with jitter).\n\tfor i := 100; i \u003c 1000; i++ {\n\t\td := bc.Backoff(i)\n\t\tif d \u003e 120*time.Second*1.2+time.Second || d \u003c= 0 {\n\t\t\tt.Fatalf(\&#34;Backoff(%d) = %v, want in (0, 144s]\&#34;, i, d)\n\t\t}\n\t}\n}\n\nfunc TestCustom(t *testing.T) {\n\t// Aggressive retry: base 100ms, factor 1.1, no jitter, max 1s.\n\tbc := Exponential{\n\t\tMaxDelay:      1 * time.Second,\n\t\tBaseDelay:     100 * time.Millisecond,\n\t\tJitter:        0,\n\t\tBackoffFactor: 1.1,\n\t}\n\tif d := bc.Backoff(0); d != 100*time.Millisecond {\n\t\tt.Fatalf(\&#34;Backoff(0) = %v, want 100ms\&#34;, d)\n\t}\n\tif d := bc.Backoff(1); d != 100*time.Millisecond {\n\t\tt.Fatalf(\&#34;Backoff(1) = %v, want 100ms\&#34;, d)\n\t}\n\t// 100ms * 1.1^10 ≈ 259ms\n\tif d := bc.Backoff(10); d != time.Duration(float64(100*time.Millisecond)*(1.1**10)) {\n\t\tt.Fatalf(\&#34;Backoff(10) = %v, want %v\&#34;, d, time.Duration(float64(100*time.Millisecond)*(1.1**10)))\n\t}\n\t// Bounded by max delay.\n\tif d := bc.Backoff(100); d != time.Second {\n\t\tt.Fatalf(\&#34;Backoff(100) = %v, want 1s\&#34;, d)\n\t}\n}\n\nfunc TestZeroBase(t *testing.T) {\n\tbc := Exponential{MaxDelay: time.Second, BaseDelay: 0, Jitter: 0, BackoffFactor: 1.6}\n\tif d := bc.Backoff(5); d != 0 {\n\t\tt.Fatalf(\&#34;Backoff(5) = %v, want 0\&#34;, d)\n\t}\n}\n\nfunc TestNegativeResult(t *testing.T) {\n\t// Huge jitter with tiny base can go negative; result must clamp to 0.\n\tbc := Exponential{MaxDelay: time.Millisecond, BaseDelay: time.Millisecond, Jitter: 10, BackoffFactor: 1.6}\n\td := bc.Backoff(1)\n\tif d \u003c 0 {\n\t\tt.Fatalf(\&#34;Backoff(1) = %v, want \u003e= 0\&#34;, d)\n\t}\n}\n\nfunc main() {\n\tr := rand.New(rand.NewSource(1))\n\t_ = r\n\tif err := func() error {\n\t\treturn nil\n\t}(); err != nil {\n\t\tfmt.Println(err)\n\t}\n}\n&#34;,
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/backoff_test.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>30</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;rm /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/backoff_test.go \u0026\u0026 mkdir -p /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/btest \u0026\u0026 cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/btest \u0026\u0026 printf &#39;package btest\\n\\nimport (\\n\\t\&#34;testing\&#34;\\n\\t\&#34;time\&#34;\\n)\\n\\ntype Exponential struct {\\n\\tMaxDelay      time.Duration\\n\\tBaseDelay     time.Duration\\n\\tJitter        float64\\n\\tBackoffFactor float64\\n}\\n\\nfunc (bc Exponential) Backoff(retries int) time.Duration {\\n\\tif retries == 0 {\\n\\t\\treturn bc.BaseDelay\\n\\t}\\n\\tbackoff, max := float64(bc.BaseDelay), float64(bc.MaxDelay)\\n\\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\\n\\t\\tbackoff *= bc.BackoffFactor\\n\\t\\tretries--\\n\\t}\\n\\tif backoff \u003e max {\\n\\t\\tbackoff = max\\n\\t}\\n\\tbackoff *= 1 + bc.Jitter*(2*0.5-1)*0 // placeholder\\n\\treturn time.Duration(backoff)\\n}\\n&#39; \u003e /dev/null&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>31</td><td><code>write</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;content&#34;: &#34;package btest\n\nimport (\n\t\&#34;testing\&#34;\n\t\&#34;time\&#34;\n)\n\ntype Exponential struct {\n\tMaxDelay      time.Duration\n\tBaseDelay     time.Duration\n\tJitter        float64\n\tBackoffFactor float64\n}\n\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn bc.BaseDelay\n\t}\n\tbackoff, max := float64(bc.BaseDelay), float64(bc.MaxDelay)\n\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\n\t\tbackoff *= bc.BackoffFactor\n\t\tretries--\n\t}\n\tif backoff \u003e max {\n\t\tbackoff = max\n\t}\n\tbackoff *= 1 + bc.Jitter*(0.999*2-1)*0.5\n\t_ = backoff\n\treturn time.Duration(0)\n}\n\nfunc TestX(t *testing.T) {}\n&#34;,
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/btest/backoff_test.go&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `provider_transport=opencode_attached_server_v1`

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

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 299.81s | 299.82s | N/A | — | — | — |
| **Process Start → First Event** | 12.58s | 25.14s | — | — | — | — |
| **First Event → First Tool Call** | 2.32s | 2.64s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 31 | 25 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 10 | 25 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 31 | 25 | N/A | — | — | — |
| **Output Tokens** | 11570 | 8655 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 468473 | 875655 | N/A | — | — | — |
| **Cached Input Tokens** | 453059 | 823037 | N/A | — | — | — |
| **Uncached Input Tokens** | 15414 | 52618 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 29.39:1 | 15.64:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/backoff*.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;BackoffConfig&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/backoff.go&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/backoff.go&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000&#34;,
  &#34;pattern&#34;: &#34;BackoffConfig&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/internal/backoff/backoff.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/internal/backoff/ \u0026\u0026 ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/internal/&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/dialoptions.go&#34;,
  &#34;limit&#34;: 80,
  &#34;offset&#34;: 230
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/doc.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -10 \u0026\u0026 git status&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/internal/backoff/backoff.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/clientconn_test.go&#34;,
  &#34;limit&#34;: 60,
  &#34;offset&#34;: 810
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000 \u0026\u0026 git log --oneline -10 \u0026\u0026 git status&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;pattern&#34;: &#34;baseDelay|factor|jitter&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/dialoptions.go&#34;,
  &#34;limit&#34;: 60,
  &#34;offset&#34;: 230
}</code></pre></td></tr>
<tr><td>9</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;internal/backoff/*&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000 \u0026\u0026 /usr/bin/git log --oneline -10 2\u003e\u00261 | head -20&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/clientconn.go&#34;,
  &#34;limit&#34;: 25,
  &#34;offset&#34;: 225
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000 \u0026\u0026 env DEVELOPER_DIR=/usr/local/bin /usr/bin/git log --oneline -10 2\u003e\u00261 | head -20&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/backoff.go&#34;,
  &#34;newString&#34;: &#34;// DefaultBackoffConfig uses values specified for backoff in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\nvar DefaultBackoffConfig = BackoffConfig{\n\tMaxDelay:      120 * time.Second,\n\tBaseDelay:     1 * time.Second,\n\tJitter:        0.2,\n\tBackoffFactor: 1.6,\n}\n\n// BackoffConfig defines the parameters for the default gRPC backoff strategy.\ntype BackoffConfig struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n\t// BaseDelay is the base delay for backoff.\n\tBaseDelay time.Duration\n\t// Jitter is the randomize factor for backoff delay.\n\tJitter float64\n\t// BackoffFactor is the factor for backoff delay.\n\tBackoffFactor float64\n}&#34;,
  &#34;oldString&#34;: &#34;// DefaultBackoffConfig uses values specified for backoff in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\nvar DefaultBackoffConfig = BackoffConfig{\n\tMaxDelay: 120 * time.Second,\n}\n\n// BackoffConfig defines the parameters for the default gRPC backoff strategy.\ntype BackoffConfig struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n}&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which -a git 2\u003e\u00261; ls /usr/local/bin/git* 2\u003e\u00261; ls /opt/homebrew/bin/git* 2\u003e\u00261&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/internal/backoff/backoff.go&#34;,
  &#34;newString&#34;: &#34;// Exponential implements exponential backoff algorithm as defined in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\ntype Exponential struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n\t// BaseDelay is the base delay for backoff.\n\tBaseDelay time.Duration\n\t// Jitter is the randomize factor for backoff delay.\n\tJitter float64\n\t// BackoffFactor is the factor for backoff delay.\n\tBackoffFactor float64\n}\n\n// Backoff returns the amount of time to wait before the next retry given the\n// number of retries.\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn bc.BaseDelay\n\t}\n\tbackoff, max := float64(bc.BaseDelay), float64(bc.MaxDelay)\n\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\n\t\tbackoff *= bc.BackoffFactor\n\t\tretries--\n\t}\n\tif backoff \u003e max {\n\t\tbackoff = max\n\t}\n\t// Randomize backoff delays so that if a cluster of requests start at\n\t// the same time, they won&#39;t operate in lockstep.\n\tbackoff *= 1 + bc.Jitter*(grpcrand.Float64()*2-1)\n\tif backoff \u003c 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;,
  &#34;oldString&#34;: &#34;const (\n\t// baseDelay is the amount of time to wait before retrying after the first\n\t// failure.\n\tbaseDelay = 1.0 * time.Second\n\t// factor is applied to the backoff after each retry.\n\tfactor = 1.6\n\t// jitter provides a range to randomize backoff delays.\n\tjitter = 0.2\n)\n\n// Exponential implements exponential backoff algorithm as defined in\n// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.\ntype Exponential struct {\n\t// MaxDelay is the upper bound of backoff delay.\n\tMaxDelay time.Duration\n}\n\n// Backoff returns the amount of time to wait before the next retry given the\n// number of retries.\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn baseDelay\n\t}\n\tbackoff, max := float64(baseDelay), float64(bc.MaxDelay)\n\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\n\t\tbackoff *= factor\n\t\tretries--\n\t}\n\tif backoff \u003e max {\n\t\tbackoff = max\n\t}\n\t// Randomize backoff delays so that if a cluster of requests start at\n\t// the same time, they won&#39;t operate in lockstep.\n\tbackoff *= 1 + jitter*(grpcrand.Float64()*2-1)\n\tif backoff \u003c 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000 \u0026\u0026 /usr/bin/git --version 2\u003e\u00261&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/dialoptions.go&#34;,
  &#34;newString&#34;: &#34;// WithBackoffConfig configures the dialer to use the provided backoff\n// parameters after connection failures.\nfunc WithBackoffConfig(b BackoffConfig) DialOption {\n\treturn withBackoff(backoff.Exponential{\n\t\tMaxDelay:      b.MaxDelay,\n\t\tBaseDelay:     b.BaseDelay,\n\t\tJitter:        b.Jitter,\n\t\tBackoffFactor: b.BackoffFactor,\n\t})\n}&#34;,
  &#34;oldString&#34;: &#34;// WithBackoffConfig configures the dialer to use the provided backoff\n// parameters after connection failures.\n//\n// Use WithBackoffMaxDelay until more parameters on BackoffConfig are opened up\n// for use.\nfunc WithBackoffConfig(b BackoffConfig) DialOption {\n\treturn withBackoff(backoff.Exponential{\n\t\tMaxDelay: b.MaxDelay,\n\t})\n}&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;file /usr/bin/git 2\u003e\u00261; head -5 /usr/bin/git 2\u003e\u00261&#34;
}</code></pre></td></tr>
<tr><td>14</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/clientconn.go&#34;,
  &#34;newString&#34;: &#34;\tif cc.dopts.bs == nil {\n\t\tcc.dopts.bs = backoff.Exponential{\n\t\t\tMaxDelay:      DefaultBackoffConfig.MaxDelay,\n\t\t\tBaseDelay:     DefaultBackoffConfig.BaseDelay,\n\t\t\tJitter:        DefaultBackoffConfig.Jitter,\n\t\t\tBackoffFactor: DefaultBackoffConfig.BackoffFactor,\n\t\t}\n\t}&#34;,
  &#34;oldString&#34;: &#34;\tif cc.dopts.bs == nil {\n\t\tcc.dopts.bs = backoff.Exponential{\n\t\t\tMaxDelay: DefaultBackoffConfig.MaxDelay,\n\t\t}\n\t}&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/dialoptions.go&#34;,
  &#34;limit&#34;: 60,
  &#34;offset&#34;: 1
}</code></pre></td></tr>
<tr><td>15</td><td><code>edit</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/clientconn_test.go&#34;,
  &#34;newString&#34;: &#34;\texpectedValue := backoff.Exponential{\n\t\tMaxDelay:      expected.MaxDelay,\n\t\tBaseDelay:     expected.BaseDelay,\n\t\tJitter:        expected.Jitter,\n\t\tBackoffFactor: expected.BackoffFactor,\n\t}\n\tif actual != expectedValue {\n\t\tt.Fatalf(\&#34;unexpected backoff config on connection: %v, want %v\&#34;, actual, expected)\n\t}&#34;,
  &#34;oldString&#34;: &#34;\texpectedValue := backoff.Exponential{\n\t\tMaxDelay: expected.MaxDelay,\n\t}\n\tif actual != expectedValue {\n\t\tt.Fatalf(\&#34;unexpected backoff config on connection: %v, want %v\&#34;, actual, expected)\n\t}&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;clientconn.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/clientconn.go&#34;,
  &#34;pattern&#34;: &#34;bs |backoff&#34;
}</code></pre></td></tr>
<tr><td>16</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;pattern&#34;: &#34;backoff.Exponential|DefaultBackoffConfig&#34;
}</code></pre></td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;markdown&#34;,
  &#34;url&#34;: &#34;https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md&#34;
}</code></pre></td></tr>
<tr><td>17</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go version \u0026\u0026 go build ./...&#34;,
  &#34;timeout&#34;: 300000,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/grpc/grpc/master/doc/connection-backoff.md)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;markdown&#34;,
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/grpc/grpc/master/doc/connection-backoff.md&#34;
}</code></pre></td></tr>
<tr><td>18</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls vendor 2\u003e/dev/null; go env GOPROXY GOFLAGS GOMODCACHE; ls $(go env GOMODCACHE) 2\u003e/dev/null | head&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/Documentation/&#34;
}</code></pre></td></tr>
<tr><td>19</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/go/mod&#34;
}</code></pre></td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000&#34;,
  &#34;pattern&#34;: &#34;**/connection-backoff*&#34;
}</code></pre></td></tr>
<tr><td>20</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/go.mod&#34;
}</code></pre></td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://grpc.io/blog/connection-backoff/)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;markdown&#34;,
  &#34;url&#34;: &#34;https://grpc.io/blog/connection-backoff/&#34;
}</code></pre></td></tr>
<tr><td>21</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -d /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/*/vendor 2\u003e/dev/null; ls /Users/alessandro/go/pkg 2\u003e/dev/null | head; ls ~/go 2\u003e/dev/null | head&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/.git/ 2\u003e\u00261; cat /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/.git/HEAD 2\u003e\u00261&#34;
}</code></pre></td></tr>
<tr><td>22</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;nslookup proxy.golang.org 2\u003e\u00261 | head -5; nslookup github.com 2\u003e\u00261 | head -5&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cat /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/.git/config 2\u003e\u00261&#34;
}</code></pre></td></tr>
<tr><td>23</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls go.sum 2\u003e/dev/null; ls internal/grpcrand/&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/clientconn.go&#34;,
  &#34;limit&#34;: 30,
  &#34;offset&#34;: 225
}</code></pre></td></tr>
<tr><td>24</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;go build ./internal/backoff/ 2\u003e\u00261 | head -20&#34;,
  &#34;timeout&#34;: 120000,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la /usr/bin/git 2\u003e\u00261; find /usr -name \&#34;git\&#34; -type f 2\u003e\u00261 | head -20; find /opt -name \&#34;git\&#34; -type f 2\u003e\u00261 | head -20&#34;
}</code></pre></td></tr>
<tr><td>25</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;gofmt -l backoff.go dialoptions.go clientconn.go clientconn_test.go internal/backoff/backoff.go&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_grpc__grpc-go-2735_1791720524955937000/clientconn.go&#34;,
  &#34;limit&#34;: 60,
  &#34;offset&#34;: 940
}</code></pre></td></tr>
<tr><td>26</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;gofmt -d internal/backoff/backoff.go | head -40&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>27</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which -a git 2\u003e/dev/null; ls /opt/homebrew/bin/git /usr/local/bin/git 2\u003e/dev/null&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>28</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;gofmt -d clientconn.go | head -60&#34;,
  &#34;workdir&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>29</td><td><code>write</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;content&#34;: &#34;package main\n\nimport (\n\t\&#34;fmt\&#34;\n\t\&#34;math/rand\&#34;\n\t\&#34;testing\&#34;\n\t\&#34;time\&#34;\n)\n\ntype Exponential struct {\n\tMaxDelay      time.Duration\n\tBaseDelay     time.Duration\n\tJitter        float64\n\tBackoffFactor float64\n}\n\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn bc.BaseDelay\n\t}\n\tbackoff, max := float64(bc.BaseDelay), float64(bc.MaxDelay)\n\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\n\t\tbackoff *= bc.BackoffFactor\n\t\tretries--\n\t}\n\tif backoff \u003e max {\n\t\tbackoff = max\n\t}\n\tbackoff *= 1 + bc.Jitter*(rand.Float64()*2-1)\n\tif backoff \u003c 0 {\n\t\treturn 0\n\t}\n\treturn time.Duration(backoff)\n}\n\nfunc TestDefault(t *testing.T) {\n\t// Default values must match old behavior: base 1s, factor 1.6, jitter 0.2, max 120s.\n\tbc := Exponential{\n\t\tMaxDelay:      120 * time.Second,\n\t\tBaseDelay:     1 * time.Second,\n\t\tJitter:        0.2,\n\t\tBackoffFactor: 1.6,\n\t}\n\tif d := bc.Backoff(0); d != time.Second {\n\t\tt.Fatalf(\&#34;Backoff(0) = %v, want %v\&#34;, d, time.Second)\n\t}\n\tif d := bc.Backoff(1); d != time.Second {\n\t\tt.Fatalf(\&#34;Backoff(1) = %v, want %v\&#34;, d, time.Second)\n\t}\n\t// After many retries, delay must be bounded by MaxDelay (with jitter).\n\tfor i := 100; i \u003c 1000; i++ {\n\t\td := bc.Backoff(i)\n\t\tif d \u003e 120*time.Second*1.2+time.Second || d \u003c= 0 {\n\t\t\tt.Fatalf(\&#34;Backoff(%d) = %v, want in (0, 144s]\&#34;, i, d)\n\t\t}\n\t}\n}\n\nfunc TestCustom(t *testing.T) {\n\t// Aggressive retry: base 100ms, factor 1.1, no jitter, max 1s.\n\tbc := Exponential{\n\t\tMaxDelay:      1 * time.Second,\n\t\tBaseDelay:     100 * time.Millisecond,\n\t\tJitter:        0,\n\t\tBackoffFactor: 1.1,\n\t}\n\tif d := bc.Backoff(0); d != 100*time.Millisecond {\n\t\tt.Fatalf(\&#34;Backoff(0) = %v, want 100ms\&#34;, d)\n\t}\n\tif d := bc.Backoff(1); d != 100*time.Millisecond {\n\t\tt.Fatalf(\&#34;Backoff(1) = %v, want 100ms\&#34;, d)\n\t}\n\t// 100ms * 1.1^10 ≈ 259ms\n\tif d := bc.Backoff(10); d != time.Duration(float64(100*time.Millisecond)*(1.1**10)) {\n\t\tt.Fatalf(\&#34;Backoff(10) = %v, want %v\&#34;, d, time.Duration(float64(100*time.Millisecond)*(1.1**10)))\n\t}\n\t// Bounded by max delay.\n\tif d := bc.Backoff(100); d != time.Second {\n\t\tt.Fatalf(\&#34;Backoff(100) = %v, want 1s\&#34;, d)\n\t}\n}\n\nfunc TestZeroBase(t *testing.T) {\n\tbc := Exponential{MaxDelay: time.Second, BaseDelay: 0, Jitter: 0, BackoffFactor: 1.6}\n\tif d := bc.Backoff(5); d != 0 {\n\t\tt.Fatalf(\&#34;Backoff(5) = %v, want 0\&#34;, d)\n\t}\n}\n\nfunc TestNegativeResult(t *testing.T) {\n\t// Huge jitter with tiny base can go negative; result must clamp to 0.\n\tbc := Exponential{MaxDelay: time.Millisecond, BaseDelay: time.Millisecond, Jitter: 10, BackoffFactor: 1.6}\n\td := bc.Backoff(1)\n\tif d \u003c 0 {\n\t\tt.Fatalf(\&#34;Backoff(1) = %v, want \u003e= 0\&#34;, d)\n\t}\n}\n\nfunc main() {\n\tr := rand.New(rand.NewSource(1))\n\t_ = r\n\tif err := func() error {\n\t\treturn nil\n\t}(); err != nil {\n\t\tfmt.Println(err)\n\t}\n}\n&#34;,
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/backoff_test.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>30</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;rm /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/backoff_test.go \u0026\u0026 mkdir -p /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/btest \u0026\u0026 cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/btest \u0026\u0026 printf &#39;package btest\\n\\nimport (\\n\\t\&#34;testing\&#34;\\n\\t\&#34;time\&#34;\\n)\\n\\ntype Exponential struct {\\n\\tMaxDelay      time.Duration\\n\\tBaseDelay     time.Duration\\n\\tJitter        float64\\n\\tBackoffFactor float64\\n}\\n\\nfunc (bc Exponential) Backoff(retries int) time.Duration {\\n\\tif retries == 0 {\\n\\t\\treturn bc.BaseDelay\\n\\t}\\n\\tbackoff, max := float64(bc.BaseDelay), float64(bc.MaxDelay)\\n\\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\\n\\t\\tbackoff *= bc.BackoffFactor\\n\\t\\tretries--\\n\\t}\\n\\tif backoff \u003e max {\\n\\t\\tbackoff = max\\n\\t}\\n\\tbackoff *= 1 + bc.Jitter*(2*0.5-1)*0 // placeholder\\n\\treturn time.Duration(backoff)\\n}\\n&#39; \u003e /dev/null&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>31</td><td><code>write</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;content&#34;: &#34;package btest\n\nimport (\n\t\&#34;testing\&#34;\n\t\&#34;time\&#34;\n)\n\ntype Exponential struct {\n\tMaxDelay      time.Duration\n\tBaseDelay     time.Duration\n\tJitter        float64\n\tBackoffFactor float64\n}\n\nfunc (bc Exponential) Backoff(retries int) time.Duration {\n\tif retries == 0 {\n\t\treturn bc.BaseDelay\n\t}\n\tbackoff, max := float64(bc.BaseDelay), float64(bc.MaxDelay)\n\tfor backoff \u003c max \u0026\u0026 retries \u003e 0 {\n\t\tbackoff *= bc.BackoffFactor\n\t\tretries--\n\t}\n\tif backoff \u003e max {\n\t\tbackoff = max\n\t}\n\tbackoff *= 1 + bc.Jitter*(0.999*2-1)*0.5\n\t_ = backoff\n\treturn time.Duration(0)\n}\n\nfunc TestX(t *testing.T) {}\n&#34;,
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_grpc__grpc-go-2735_1791720224916127000/.scratch/opencode/tmp/opencode/btest/backoff_test.go&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

