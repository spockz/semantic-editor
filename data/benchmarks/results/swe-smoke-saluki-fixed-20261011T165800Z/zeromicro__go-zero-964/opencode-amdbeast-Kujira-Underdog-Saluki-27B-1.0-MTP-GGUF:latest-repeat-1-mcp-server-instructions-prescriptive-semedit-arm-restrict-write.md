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

## Test case: `zeromicro__go-zero-964`

### Target: `opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/zeromicro__go-zero-964.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/zeromicro__go-zero-964.txtar)

**Vanilla LLM Prompt**:
> 如题。
> etcd打开了rbac authentication。但discov.EtcdConf中没有user等字段，无法进行身份认证。连接etcd时会报错：
> etcdserver: user name is empty
>

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 300.00s | 190.66s | <span class="benchmark-delta-positive">-36.4%</span> | — | — | — |
| **Process Start → First Event** | 8.55s | 31.70s | — | — | — | — |
| **First Event → First Tool Call** | 6.94s | 111.68s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 25 | 1 | <span class="benchmark-delta-positive">-96.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 25 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 25 | 1 | <span class="benchmark-delta-positive">-96.0%</span> | — | — | — |
| **Output Tokens** | 6793 | 2659 | <span class="benchmark-delta-positive">-60.9%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 400505 | 69728 | <span class="benchmark-delta-positive">-82.6%</span> | — | — | — |
| **Cached Input Tokens** | 366075 | 26441 | <span class="benchmark-delta-negative">-92.8%</span> | — | — | — |
| **Uncached Input Tokens** | 34430 | 43287 | <span class="benchmark-delta-negative">+25.7%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 10.63:1 | 0.61:1 | <span class="benchmark-delta-negative">-94.3%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | — | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | — | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | — | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | — | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/discov/**&#34;
}</code></pre></td><td><code>semedit_semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: true,
  &#34;edits&#34;: [
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/config.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;type EtcdConf struct {\n\tHosts []string\n\tKey   string\n\tUser  string `json:\&#34;,optional\&#34;`\n\tPass  string `json:\&#34;,optional=User\&#34;`\n}&#34;,
        &#34;symbol&#34;: &#34;EtcdConf&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;body&#34;: &#34;if len(c.Hosts) == 0 {\n\treturn errors.New(\&#34;empty etcd hosts\&#34;)\n}\nif len(c.Key) == 0 {\n\treturn errors.New(\&#34;empty etcd key\&#34;)\n}\nif (c.User == \&#34;\&#34;) != (c.Pass == \&#34;\&#34;) {\n\treturn errors.New(\&#34;etcd user and pass must be paired\&#34;)\n}\nreturn nil&#34;,
        &#34;file&#34;: &#34;core/discov/config.go&#34;,
        &#34;symbol&#34;: &#34;EtcdConf.Validate&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_body&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/config.go&#34;,
        &#34;source&#34;: &#34;func (c EtcdConf) EnableAuth() bool {\n\treturn c.User != \&#34;\&#34; \u0026\u0026 c.Pass != \&#34;\&#34;\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/internal/registry.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;type cluster struct {\n\tendpoints  []string\n\tuser       string\n\tpass       string\n\tkey        string\n\tvalues     map[string]map[string]string\n\tlisteners  map[string][]UpdateListener\n\twatchGroup *threading.RoutineGroup\n\tdone       chan lang.PlaceholderType\n\tlock       sync.Mutex\n}&#34;,
        &#34;symbol&#34;: &#34;cluster&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/internal/registry.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;source&#34;: &#34;func DialClient(endpoints []string, user, pass string) (EtcdClient, error) {\n\treturn clientv3.New(clientv3.Config{\n\t\tEndpoints:            endpoints,\n\t\tAutoSyncInterval:     autoSyncInterval,\n\t\tDialTimeout:          DialTimeout,\n\t\tDialKeepAliveTime:    dialKeepAliveTime,\n\t\tDialKeepAliveTimeout: DialTimeout,\n\t\tRejectOldCluster:     true,\n\t\tUsername:             user,\n\t\tPassword:             pass,\n\t})\n}&#34;,
        &#34;symbol&#34;: &#34;DialClient&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;body&#34;: &#34;cli, err := NewClient(c.endpoints, c.user, c.pass)\nif err != nil {\n\treturn nil, err\n}\n\ngo c.watchConnState(cli)\n\nreturn cli, nil&#34;,
        &#34;file&#34;: &#34;core/discov/internal/registry.go&#34;,
        &#34;symbol&#34;: &#34;cluster.newClient&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_body&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/internal/registry.go&#34;,
        &#34;source&#34;: &#34;func newClusterWithAuth(endpoints []string, user, pass string) *cluster {\n\tc := newCluster(endpoints)\n\tc.user = user\n\tc.pass = pass\n\treturn c\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/internal/registry.go&#34;,
        &#34;source&#34;: &#34;func (r *Registry) getClusterWithAuth(endpoints []string, user, pass string) *cluster {\n\tclusterKey := fmt.Sprintf(\&#34;%s|%q|%q\&#34;, getClusterKey(endpoints), user, pass)\n\tr.lock.Lock()\n\tdefer r.lock.Unlock()\n\tc, ok := r.clusters[clusterKey]\n\tif !ok {\n\t\tc = newClusterWithAuth(endpoints, user, pass)\n\t\tr.clusters[clusterKey] = c\n\t}\n\treturn c\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/internal/registry.go&#34;,
        &#34;source&#34;: &#34;func (r *Registry) GetConnWithAuth(endpoints []string, user, pass string) (EtcdClient, error) {\n\treturn r.getClusterWithAuth(endpoints, user, pass).getClient()\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/internal/registry.go&#34;,
        &#34;source&#34;: &#34;func (r *Registry) MonitorWithAuth(endpoints []string, user, pass, key string, l UpdateListener) error {\n\treturn r.getClusterWithAuth(endpoints, user, pass).monitor(key, l)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/publisher.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;// A Publisher can be used to publish the value to an etcd cluster on the given key.\ntype Publisher struct {\n\tendpoints  []string\n\tuser       string\n\tpass       string\n\tkey        string\n\tfullKey    string\n\tid         int64\n\tvalue      string\n\tlease      clientv3.LeaseID\n\tquit       *syncx.DoneChan\n\tpauseChan  chan lang.PlaceholderType\n\tresumeChan chan lang.PlaceholderType\n}&#34;,
        &#34;symbol&#34;: &#34;Publisher&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/publisher.go&#34;,
        &#34;source&#34;: &#34;func NewPublisherWithAuth(endpoints []string, user, pass, key, value string, opts ...PublisherOption) *Publisher {\n\tpublisher := \u0026Publisher{\n\t\tendpoints:  endpoints,\n\t\tuser:       user,\n\t\tpass:       pass,\n\t\tkey:        key,\n\t\tvalue:      value,\n\t\tquit:       syncx.NewDoneChan(),\n\t\tpauseChan:  make(chan lang.PlaceholderType),\n\t\tresumeChan: make(chan lang.PlaceholderType),\n\t}\n\tfor _, opt := range opts {\n\t\topt(publisher)\n\t}\n\treturn publisher\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/discov/publisher.go&#34;,
        &#34;source&#34;: &#34;func (p *Publisher) KeepAliveWithAuth() error {\n\tcli, err := internal.GetRegistry().GetConnWithAuth(p.endpoints, p.user, p.pass)\n\tif err != nil {\n\t\treturn err\n\t}\n\tp.lease, err = p.register(cli)\n\tif err != nil {\n\t\treturn err\n\t}\n\tproc.AddWrapUpListener(func() {\n\t\tp.Stop()\n\t})\n\treturn p.keepAliveAsync(cli)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;zrpc/internal/rpcpubserver.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;source&#34;: &#34;func NewRpcPubServer(etcdEndpoints []string, etcdKey, listenOn string, opts ...ServerOption) (Server, error) {\n\tregisterEtcd := func() error {\n\t\tpubListenOn := figureOutListenOn(listenOn)\n\t\tpubClient := discov.NewPublisher(etcdEndpoints, etcdKey, pubListenOn)\n\t\treturn pubClient.KeepAlive()\n\t}\n\treturn newKeepAliveServer(registerEtcd, listenOn, opts...), nil\n}&#34;,
        &#34;symbol&#34;: &#34;NewRpcPubServer&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;zrpc/internal/rpcpubserver.go&#34;,
        &#34;source&#34;: &#34;func NewRpcPubServerWithEtcdAuth(etcdEndpoints []string, user, pass, etcdKey, listenOn string, opts ...ServerOption) (Server, error) {\n\tregisterEtcd := func() error {\n\t\tpubListenOn := figureOutListenOn(listenOn)\n\t\tpubClient := discov.NewPublisherWithAuth(etcdEndpoints, user, pass, etcdKey, pubListenOn)\n\t\treturn pubClient.KeepAliveWithAuth()\n\t}\n\treturn newKeepAliveServer(registerEtcd, listenOn, opts...), nil\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;zrpc/internal/rpcpubserver.go&#34;,
        &#34;source&#34;: &#34;func newKeepAliveServer(registerEtcd func() error, listenOn string, opts ...ServerOption) keepAliveServer {\n\treturn keepAliveServer{\n\t\tregisterEtcd: registerEtcd,\n\t\tServer:       NewRpcServer(listenOn, opts...),\n\t}\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;zrpc/server.go&#34;,
        &#34;kind&#34;: &#34;function&#34;,
        &#34;source&#34;: &#34;func NewServer(c RpcServerConf, register internal.RegisterFn) (*RpcServer, error) {\n\tvar err error\n\tif err = c.Validate(); err != nil {\n\t\treturn nil, err\n\t}\n\n\tvar server internal.Server\n\tmetrics := stat.NewMetrics(c.ListenOn)\n\tserverOptions := []internal.ServerOption{internal.WithMetrics(metrics), internal.WithMaxRetries(c.MaxRetries)}\n\n\tif c.HasEtcd() {\n\t\tif c.Etcd.EnableAuth() {\n\t\t\tserver, err = internal.NewRpcPubServerWithEtcdAuth(c.Etcd.Hosts, c.Etcd.User, c.Etcd.Pass, c.Etcd.Key, c.ListenOn, serverOptions...)\n\t\t} else {\n\t\t\tserver, err = internal.NewRpcPubServer(c.Etcd.Hosts, c.Etcd.Key, c.ListenOn, serverOptions...)\n\t\t}\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t} else {\n\t\tserver = internal.NewRpcServer(c.ListenOn, serverOptions...)\n\t}\n\n\tserver.SetName(c.Name)\n\tif err = setupInterceptors(server, c, metrics); err != nil {\n\t\treturn nil, err\n\t}\n\n\trpcServer := \u0026RpcServer{\n\t\tserver:   server,\n\t\tregister: register,\n\t}\n\tif err = c.SetUp(); err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn rpcServer, nil\n}&#34;,
        &#34;symbol&#34;: &#34;NewServer&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_construct&#34;
    }
  ],
  &#34;trust_workspace&#34;: true
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;EtcdConf&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>3</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/config.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>4</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/internal/etcdclient.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>5</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/internal/registry.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>6</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000&#34;,
  &#34;pattern&#34;: &#34;DialClient|NewClient&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>7</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/clients.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>8</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/internal/registry_test.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>9</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/config_test.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>10</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/publisher.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>11</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/subscriber.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>12</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000&#34;,
  &#34;pattern&#34;: &#34;NewSubscriber|NewPublisher|GetConn|Monitor\\(&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>13</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/resolver/discovbuilder.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>14</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/rpcpubserver.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>15</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000&#34;,
  &#34;pattern&#34;: &#34;Etcd\\.Hosts|Etcd\\.Key|Etcd\\.User|Etcd\\.Password&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>16</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/config.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/config.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>17</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/internal/registry.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/internal/registry.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>18</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/zeromicro/go-zero/blob/master/core/discov/config.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://github.com/zeromicro/go-zero/blob/master/core/discov/config.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>19</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc&#34;,
  &#34;pattern&#34;: &#34;BuildDiscovTarget|discovBuilder|RegisterName|RegisterResolver&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>20</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/target.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>21</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/resolver/resolver.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>22</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/target_test.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>23</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://api.github.com/repos/zeromicro/go-zero/contents/core/discov/config.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://api.github.com/repos/zeromicro/go-zero/contents/core/discov/config.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>24</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/go.mod&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>25</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/config.go&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/zeromicro__go-zero-964.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/zeromicro__go-zero-964.txtar)

**Vanilla LLM Prompt**:
> 如题。
> etcd打开了rbac authentication。但discov.EtcdConf中没有user等字段，无法进行身份认证。连接etcd时会报错：
> etcdserver: user name is empty
>

**Semedit MCP Prompt**:
> 如题。
> etcd打开了rbac authentication。但discov.EtcdConf中没有user等字段，无法进行身份认证。连接etcd时会报错：
> etcdserver: user name is empty Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

| Metric | Vanilla (Small) | Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 300.00s | 300.01s | <span class="benchmark-delta-negative">+0.0%</span> | — | — | — |
| **Process Start → First Event** | 8.55s | 26.05s | — | — | — | — |
| **First Event → First Tool Call** | 6.94s | 4.26s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 25 | 30 | <span class="benchmark-delta-negative">+20.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 25 | 1 | <span class="benchmark-delta-positive">-96.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 25 | 30 | <span class="benchmark-delta-negative">+20.0%</span> | — | — | — |
| **Output Tokens** | 6793 | 7421 | <span class="benchmark-delta-negative">+9.2%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 400505 | 931401 | <span class="benchmark-delta-negative">+132.6%</span> | — | — | — |
| **Cached Input Tokens** | 366075 | 882882 | <span class="benchmark-delta-positive">+141.2%</span> | — | — | — |
| **Uncached Input Tokens** | 34430 | 48519 | <span class="benchmark-delta-negative">+40.9%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 10.63:1 | 18.20:1 | <span class="benchmark-delta-positive">+71.1%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/discov/**&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;EtcdConf&#34;
}</code></pre></td><td><code>semedit_semantic_lookup</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;language&#34;: &#34;go&#34;,
  &#34;symbol&#34;: &#34;EtcdConf&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/config.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/config.go&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/internal/etcdclient.go&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core&#34;,
  &#34;pattern&#34;: &#34;EtcdConf|etcd.NewClient|etcd.New&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/internal/registry.go&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core&#34;,
  &#34;pattern&#34;: &#34;etcd\\.&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000&#34;,
  &#34;pattern&#34;: &#34;DialClient|NewClient&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/internal/etcdclient.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/clients.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/internal/etcdclient_mock.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/internal/registry_test.go&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000&#34;,
  &#34;pattern&#34;: &#34;NewEtcdClient|clientv3\\.NewClient|NewClient&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/config_test.go&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000&#34;,
  &#34;pattern&#34;: &#34;EtcdConf&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/publisher.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/internal/vars.go&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/core/discov/subscriber.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/internal/registry.go&#34;,
  &#34;limit&#34;: 80,
  &#34;offset&#34;: 200
}</code></pre></td></tr>
<tr><td>12</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000&#34;,
  &#34;pattern&#34;: &#34;NewSubscriber|NewPublisher|GetConn|Monitor\\(&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core&#34;,
  &#34;pattern&#34;: &#34;DialClient&#34;
}</code></pre></td></tr>
<tr><td>13</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/resolver/discovbuilder.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/internal/registry.go&#34;,
  &#34;limit&#34;: 45,
  &#34;offset&#34;: 280
}</code></pre></td></tr>
<tr><td>14</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/rpcpubserver.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): File not found: /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/conf.go<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/conf.go&#34;
}</code></pre></td></tr>
<tr><td>15</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000&#34;,
  &#34;pattern&#34;: &#34;Etcd\\.Hosts|Etcd\\.Key|Etcd\\.User|Etcd\\.Password&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/internal&#34;
}</code></pre></td></tr>
<tr><td>16</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/config.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/config.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/clients.go&#34;
}</code></pre></td></tr>
<tr><td>17</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/internal/registry.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/internal/registry.go&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov&#34;,
  &#34;pattern&#34;: &#34;conf\\.|Etcd\\.&#34;
}</code></pre></td></tr>
<tr><td>18</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/zeromicro/go-zero/blob/master/core/discov/config.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://github.com/zeromicro/go-zero/blob/master/core/discov/config.go&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000&#34;,
  &#34;pattern&#34;: &#34;\\.Etcd\\b&#34;
}</code></pre></td></tr>
<tr><td>19</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc&#34;,
  &#34;pattern&#34;: &#34;BuildDiscovTarget|discovBuilder|RegisterName|RegisterResolver&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/subscriber.go&#34;
}</code></pre></td></tr>
<tr><td>20</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/target.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/zrpc/config.go&#34;
}</code></pre></td></tr>
<tr><td>21</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/resolver/resolver.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/internal/registry.go&#34;,
  &#34;limit&#34;: 200,
  &#34;offset&#34;: 1
}</code></pre></td></tr>
<tr><td>22</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/internal/target_test.go&#34;
}</code></pre></td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;include&#34;: &#34;*.go&#34;,
  &#34;path&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/zrpc&#34;,
  &#34;pattern&#34;: &#34;BuildDiscovTarget|DialClient|etcd&#34;
}</code></pre></td></tr>
<tr><td>23</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://api.github.com/repos/zeromicro/go-zero/contents/core/discov/config.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://api.github.com/repos/zeromicro/go-zero/contents/core/discov/config.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/core/discov/publisher.go&#34;
}</code></pre></td></tr>
<tr><td>24</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/go.mod&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/zrpc/client.go&#34;
}</code></pre></td></tr>
<tr><td>25</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-964_1791742980177158000/zrpc/config.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000/zrpc/internal/rpcpubserver.go&#34;
}</code></pre></td></tr>
<tr><td>26</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000 \u0026\u0026 git log --oneline -10 \u0026\u0026 echo \&#34;---\&#34; \u0026\u0026 git status&#34;
}</code></pre></td></tr>
<tr><td>27</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000 \u0026\u0026 git --git-dir=.git log --oneline -10 2\u003e\u00261; echo \&#34;---\&#34;; git --git-dir=.git status 2\u003e\u00261&#34;
}</code></pre></td></tr>
<tr><td>28</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;export DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer 2\u003e/dev/null; cd /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-964_1791743284880510000 \u0026\u0026 git --git-dir=.git log --oneline -10 2\u003e\u00261; echo \&#34;---\&#34;; git --git-dir=.git status 2\u003e\u00261&#34;
}</code></pre></td></tr>
<tr><td>29</td><td>—</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/config.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://raw.githubusercontent.com/zeromicro/go-zero/master/core/discov/config.go&#34;
}</code></pre></td></tr>
<tr><td>30</td><td>—</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/zeromicro/go-zero/blob/master/core/discov/config.go)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;url&#34;: &#34;https://github.com/zeromicro/go-zero/blob/master/core/discov/config.go&#34;
}</code></pre></td></tr>
</tbody>
</table>

