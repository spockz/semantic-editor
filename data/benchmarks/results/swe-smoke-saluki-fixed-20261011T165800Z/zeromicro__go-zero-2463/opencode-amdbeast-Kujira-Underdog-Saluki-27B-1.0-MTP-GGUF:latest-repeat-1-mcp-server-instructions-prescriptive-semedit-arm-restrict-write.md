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

## Test case: `zeromicro__go-zero-2463`

### Target: `opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest (repeat 1)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/zeromicro__go-zero-2463.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/zeromicro__go-zero-2463.txtar)

**Vanilla LLM Prompt**:
> logx能否支持一下全局添加field功能，或默认加一个serviceName，收集日志想根据服务分索引很不方便
>

| Metric | Vanilla (Small) | Semedit Explicit (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 299.99s | 299.99s | <span class="benchmark-delta-positive">-0.0%</span> | — | — | — |
| **Process Start → First Event** | 16.66s | 108.97s | — | — | — | — |
| **First Event → First Tool Call** | 29.37s | 9.64s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 2 | 1 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **Internal Tool Cycles** | 12 | 6 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 12 | 5 | <span class="benchmark-delta-positive">-58.3%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 12 | 6 | <span class="benchmark-delta-positive">-50.0%</span> | — | — | — |
| **Output Tokens** | 2129 | 2866 | <span class="benchmark-delta-negative">+34.6%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 65352 | 116234 | <span class="benchmark-delta-negative">+77.9%</span> | — | — | — |
| **Cached Input Tokens** | 46604 | 56061 | <span class="benchmark-delta-positive">+20.3%</span> | — | — | — |
| **Uncached Input Tokens** | 18748 | 60173 | <span class="benchmark-delta-negative">+221.0%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 2.49:1 | 0.93:1 | <span class="benchmark-delta-negative">-62.5%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | — | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | — | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | — | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Baseline vs Semedit Explicit in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/logx*&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_zeromicro__go-zero-2463_1791742672174104000&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;zeromicro/logx&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_zeromicro__go-zero-2463_1791742672174104000/core/logx/ \u0026\u0026 ls /Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_zeromicro__go-zero-2463_1791742672174104000/core/logc/ 2\u003e/dev/null || echo \&#34;logc dir does not exist\&#34;&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/zeromicro/logx)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;markdown&#34;,
  &#34;url&#34;: &#34;https://github.com/zeromicro/logx&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Tool execution aborted<pre class="benchmark-tool-arguments"><code class="language-json">{}</code></pre></td></tr>
<tr><td>4</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://pkg.go.dev/zeromicro/logx)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;markdown&#34;,
  &#34;url&#34;: &#34;https://pkg.go.dev/zeromicro/logx&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_zeromicro__go-zero-2463_1791742672174104000/core/logx/fields.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_zeromicro__go-zero-2463_1791742672174104000/core/logx/writer.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;core/logx/**&#34;
}</code></pre></td><td><code>semedit_semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Critical result (JSON): {&#34;application_state&#34;:&#34;unknown&#34;,&#34;error&#34;:&#34;batch execution error: batch diagnostics before in \&#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-explicit_small-explicit_zeromicro__go-zero-2463_1791742672174104000/tools/goctl\&#34;: check Go diagnostics with gopls: load Go diagnostics session scope: err: exit status 1: stderr: go: github.com/emicklei/proto@v1.10.0: Get \&#34;https://proxy.golang.org/github.com/emicklei/proto/@v/v1.10.0.mod\&#34;: dial tcp: lookup pr<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;auto_organize_imports&#34;: true,
  &#34;edits&#34;: [
    {
      &#34;params&#34;: {
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;package&#34;: &#34;logc&#34;,
        &#34;purpose_header&#34;: &#34;// Package logc provides context-aware logging wrappers.&#34;
      },
      &#34;tool&#34;: &#34;semantic_scaffold_file&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logx/fields.go&#34;,
        &#34;source&#34;: &#34;var globalFields atomic.Value&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logx/fields.go&#34;,
        &#34;source&#34;: &#34;var globalFieldsLock sync.Mutex&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_decl&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logx/fields.go&#34;,
        &#34;source&#34;: &#34;func AddGlobalFields(fields ...LogField) {\n\tglobalFieldsLock.Lock()\n\tdefer globalFieldsLock.Unlock()\n\n\told := globalFields.Load()\n\tif old == nil {\n\t\tglobalFields.Store(append([]LogField(nil), fields...))\n\t} else {\n\t\tglobalFields.Store(append(old.([]LogField), fields...))\n\t}\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logx/writer.go&#34;,
        &#34;source&#34;: &#34;func combineGlobalFields(fields []LogField) []LogField {\n\tglobals := globalFields.Load()\n\tif globals == nil {\n\t\treturn fields\n\t}\n\n\treturn append(globals.([]LogField), fields...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;body&#34;: &#34;fields = combineGlobalFields(fields)\n\nswitch atomic.LoadUint32(\u0026encoding) {\ncase plainEncodingType:\n\twritePlainAny(writer, level, val, buildFields(fields...)...)\ndefault:\n\tentry := make(logEntry)\n\tfor _, field := range fields {\n\t\tentry[field.Key] = field.Value\n\t}\n\tentry[timestampKey] = getTimestamp()\n\tentry[levelKey] = level\n\tentry[contentKey] = val\n\twriteJson(writer, entry)\n}&#34;,
        &#34;file&#34;: &#34;core/logx/writer.go&#34;,
        &#34;symbol&#34;: &#34;output&#34;
      },
      &#34;tool&#34;: &#34;semantic_replace_body&#34;
    },
    {
      &#34;params&#34;: {
        &#34;access_modifier&#34;: &#34;public&#34;,
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;kind&#34;: &#34;type&#34;,
        &#34;source&#34;: &#34;type (\n\tLogConf = logx.LogConf\n\tLogField = logx.LogField\n)&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_construct&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// AddGlobalFields adds global fields.\nfunc AddGlobalFields(fields ...LogField) {\n\tlogx.AddGlobalFields(fields...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Alert alerts v in alert level, and the message is written to error log.\nfunc Alert(_ context.Context, v string) {\n\tlogx.Alert(v)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Close closes the logging.\nfunc Close() error {\n\treturn logx.Close()\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Error writes v into error log.\nfunc Error(ctx context.Context, v ...interface{}) {\n\tgetLogger(ctx).Error(v...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Errorf writes v with format into error log.\nfunc Errorf(ctx context.Context, format string, v ...interface{}) {\n\tgetLogger(ctx).Errorf(fmt.Errorf(format, v...).Error())\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Errorv writes v into error log with json content.\n// No call stack attached, because not elegant to pack the messages.\nfunc Errorv(ctx context.Context, v interface{}) {\n\tgetLogger(ctx).Errorv(v)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Errorw writes msg along with fields into error log.\nfunc Errorw(ctx context.Context, msg string, fields ...LogField) {\n\tgetLogger(ctx).Errorw(msg, fields...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Field returns a LogField for the given key and value.\nfunc Field(key string, value interface{}) LogField {\n\treturn logx.Field(key, value)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Info writes v into access log.\nfunc Info(ctx context.Context, v ...interface{}) {\n\tgetLogger(ctx).Info(v...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Infof writes v with format into access log.\nfunc Infof(ctx context.Context, format string, v ...interface{}) {\n\tgetLogger(ctx).Infof(format, v...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Infov writes v into access log with json content.\nfunc Infov(ctx context.Context, v interface{}) {\n\tgetLogger(ctx).Infov(v)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Infow writes msg along with fields into access log.\nfunc Infow(ctx context.Context, msg string, fields ...LogField) {\n\tgetLogger(ctx).Infow(msg, fields...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Must checks if err is nil, otherwise logs the error and exits.\nfunc Must(err error) {\n\tlogx.Must(err)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// MustSetup sets up logging with given config c. It exits on error.\nfunc MustSetup(c logx.LogConf) {\n\tlogx.MustSetup(c)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// SetLevel sets the logging level. It can be used to suppress some logs.\nfunc SetLevel(level uint32) {\n\tlogx.SetLevel(level)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// SetUp sets up the logx. If already set up, just return nil.\n// we allow SetUp to be called multiple times, because for example\n// we need to allow different service frameworks to initialize logx respectively.\n// the same logic for SetUp\nfunc SetUp(c LogConf) error {\n\treturn logx.SetUp(c)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Slow writes v into slow log.\nfunc Slow(ctx context.Context, v ...interface{}) {\n\tgetLogger(ctx).Slow(v...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Slowf writes v with format into slow log.\nfunc Slowf(ctx context.Context, format string, v ...interface{}) {\n\tgetLogger(ctx).Slowf(format, v...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Slowv writes v into slow log with json content.\nfunc Slowv(ctx context.Context, v interface{}) {\n\tgetLogger(ctx).Slowv(v)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// Sloww writes msg along with fields into slow log.\nfunc Sloww(ctx context.Context, msg string, fields ...LogField) {\n\tgetLogger(ctx).Sloww(msg, fields...)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;auto_organize_imports&#34;: true,
        &#34;file&#34;: &#34;core/logc/logs.go&#34;,
        &#34;source&#34;: &#34;// getLogger returns the logx.Logger with the given ctx and correct caller.\nfunc getLogger(ctx context.Context) logx.Logger {\n\treturn logx.WithContext(ctx).WithCallerSkip(1)\n}&#34;
      },
      &#34;tool&#34;: &#34;semantic_insert_function&#34;
    },
    {
      &#34;params&#34;: {
        &#34;add&#34;: [
          &#34;github.com/zeromicro/go-zero/core/logx&#34;
        ],
        &#34;file&#34;: &#34;core/logc/logs.go&#34;
      },
      &#34;tool&#34;: &#34;semantic_organize_imports&#34;
    }
  ],
  &#34;trust_workspace&#34;: true
}</code></pre></td></tr>
<tr><td>7</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/logger.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>8</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/fields.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>9</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/fields_test.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>10</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/richlogger.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>11</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/lesslogger.go&#34;
}</code></pre></td><td>—</td></tr>
<tr><td>12</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/logs.go&#34;
}</code></pre></td><td>—</td></tr>
</tbody>
</table>

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Run Provenance**: `fixture_dependency_preparation=go-list-readonly-deps-test,local_model_context_tokens=65536,local_model_limits_source=llama.cpp-props,local_model_output_tokens=8192,provider_transport=opencode_attached_server_v1`

* **Fixture**: [testdata/bench/zeromicro__go-zero-2463.txtar](https://github.com/spockz/semantic-editor/blob/3089a8ddc50051bfbd8b5f556d022c75c8040b30/testdata/bench/zeromicro__go-zero-2463.txtar)

**Vanilla LLM Prompt**:
> logx能否支持一下全局添加field功能，或默认加一个serviceName，收集日志想根据服务分索引很不方便
>

**Semedit MCP Prompt**:
> logx能否支持一下全局添加field功能，或默认加一个serviceName，收集日志想根据服务分索引很不方便 Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> Semedit Agentic (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 299.99s | 300.02s | N/A | — | — | — |
| **Process Start → First Event** | 16.66s | 29.71s | — | — | — | — |
| **First Event → First Tool Call** | 29.37s | 31.43s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 2 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 12 | 15 | N/A | — | — | — |
| **Initial Load / Discovery Turns** | 12 | 15 | N/A | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 12 | 15 | N/A | — | — | — |
| **Output Tokens** | 2129 | 2034 | N/A | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 65352 | 286011 | N/A | — | — | — |
| **Cached Input Tokens** | 46604 | 214769 | N/A | — | — | — |
| **Uncached Input Tokens** | 18748 | 71242 | N/A | — | — | — |
| **Cached vs Uncached Token Ratio** | 2.49:1 | 3.01:1 | N/A | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | — | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | — | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | — | — | — | — | — |
| **Oracle L4: Verification Test** | ❌ FAIL | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

##### Baseline vs Semedit Agentic in Small Context

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;**/logx*&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la&#34;
}</code></pre></td></tr>
<tr><td>2</td><td><code>grep</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;zeromicro/logx&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -10 2\u003e/dev/null; git status 2\u003e/dev/null&#34;
}</code></pre></td></tr>
<tr><td>3</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://github.com/zeromicro/logx)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;markdown&#34;,
  &#34;url&#34;: &#34;https://github.com/zeromicro/logx&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls core/logx&#34;
}</code></pre></td></tr>
<tr><td>4</td><td><code>webfetch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional failed" title="Functional failed">✗</span>): Transport error (GET https://pkg.go.dev/zeromicro/logx)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;format&#34;: &#34;markdown&#34;,
  &#34;url&#34;: &#34;https://pkg.go.dev/zeromicro/logx&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-2463_1791742363353089000/core/logx/logger.go&#34;
}</code></pre></td></tr>
<tr><td>5</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls -la&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-2463_1791742363353089000/core/logx/richlogger.go&#34;
}</code></pre></td></tr>
<tr><td>6</td><td><code>glob</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;pattern&#34;: &#34;core/logx/**&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-2463_1791742363353089000/core/logx/logs.go&#34;
}</code></pre></td></tr>
<tr><td>7</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/logger.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-2463_1791742363353089000/core/logx/config.go&#34;
}</code></pre></td></tr>
<tr><td>8</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/fields.go&#34;
}</code></pre></td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_semedit-agentic_small-implicit_zeromicro__go-zero-2463_1791742363353089000/core/logx/fields.go&#34;
}</code></pre></td></tr>
<tr><td>9</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/fields_test.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;git log --oneline -20&#34;
}</code></pre></td></tr>
<tr><td>10</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/richlogger.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls .scratch .scratch/*&#34;
}</code></pre></td></tr>
<tr><td>11</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/lesslogger.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;export DEVELOPER_DIR=$(ls -d /Library/Developer/CommandLineTools 2\u003e/dev/null || echo /usr/local/share); git -C . log --oneline -10 2\u003e\u00261 | head -20&#34;
}</code></pre></td></tr>
<tr><td>12</td><td><code>read</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;filePath&#34;: &#34;/Users/alessandro/sources/semantic-editor/.scratch/benchmarks/run_opencode_baseline-diff_small_zeromicro__go-zero-2463_1791742053259183000/core/logx/logs.go&#34;
}</code></pre></td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;ls .git; cat .git/HEAD 2\u003e/dev/null&#34;
}</code></pre></td></tr>
<tr><td>13</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;which git; ls -la $(dirname $(which git)) | head&#34;
}</code></pre></td></tr>
<tr><td>14</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;export DEVELOPER_DIR=/Users/alessandro/devtools; git --version 2\u003e\u00261 | head -3&#34;
}</code></pre></td></tr>
<tr><td>15</td><td>—</td><td><code>bash</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;command&#34;: &#34;find / -name \&#34;git\&#34; -type f 2\u003e/dev/null | grep -v \&#34;Library/Developer\&#34; | grep -iE \&#34;bin/git$\&#34; | head&#34;
}</code></pre></td></tr>
</tbody>
</table>

