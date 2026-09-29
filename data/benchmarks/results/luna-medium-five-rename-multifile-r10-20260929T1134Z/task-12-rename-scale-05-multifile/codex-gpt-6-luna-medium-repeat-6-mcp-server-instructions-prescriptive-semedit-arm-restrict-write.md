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

* **Date**: 2026-09-29 13:35:39 CEST

## Test case: `task-12-rename-scale-05-multifile`

### Target: `codex/gpt-6-luna/medium (repeat 6)`

#### Configuration: default prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_12_rename_scale_05_multifile.txtar` (`sha256:6be74fd39c7f37d9302cb8cb (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Perform these 5 renames consecutively, possibly batched if you know how. Use rename-plan.md, update every call site, preserve behavior, and run go test ./... before finishing. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Perform these 5 renames consecutively, possibly batched if you know how. Use rename-plan.md, update every call site, preserve behavior, and run go test ./... before finishing. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package pipeline keeps the rename targets together so the oracle can check all five declarations.
package pipeline

type Processor struct{}

func (Processor) LoadInput(value int) int { return value + 1 }
func (Processor) CheckInput(value int) int { return value + 2 }
func (Processor) NormalizeValue(value int) int { return value + 3 }
func (Processor) MapFields(value int) int { return value + 4 }
func (Processor) RouteValue(value int) int { return value + 5 }
```
</details>

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 32.76s | 37.28s | <span class="benchmark-delta-negative">+13.8%</span> | — | — | — |
| **Process Start → First Event** | 0.49s | 0.18s | — | — | — | — |
| **First Event → First Tool Call** | 6.79s | 6.36s | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 5 | 4 | <span class="benchmark-delta-positive">-20.0%</span> | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 0 | <span class="benchmark-delta-positive">-100.0%</span> | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 5 | 4 | <span class="benchmark-delta-positive">-20.0%</span> | — | — | — |
| **Output Tokens** | 729 | 581 | <span class="benchmark-delta-positive">-20.3%</span> | — | — | — |
| **Reasoning / Thinking Tokens** | 89 | 152 | <span class="benchmark-delta-negative">+70.8%</span> | — | — | — |
| **Total Input Tokens** | 122992 | 118371 | <span class="benchmark-delta-positive">-3.8%</span> | — | — | — |
| **Cached Input Tokens** | 107008 | 101120 | <span class="benchmark-delta-negative">-5.5%</span> | — | — | — |
| **Uncached Input Tokens** | 15984 | 17251 | <span class="benchmark-delta-negative">+7.9%</span> | — | — | — |
| **Cached vs Uncached Token Ratio** | 6.69:1 | 5.86:1 | <span class="benchmark-delta-negative">-12.4%</span> | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

##### Standard vs Semedit in Small Context
* **Vanilla Edit**: File pipeline/processor.go modified (10 lines -> 10 lines)
* **MCP Edit**: File pipeline/processor.go modified (10 lines -> 10 lines)

<table class="benchmark-tool-calls">
<thead><tr><th>#</th><th>Vanilla</th><th>Semedit MCP</th></tr></thead>
<tbody>
<tr><td>1</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;pwd &amp;&amp; \
rg --files&#39;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;pwd &amp;&amp; \
rg --files -g &#39;rename-plan.md&#39; -g &#39;AGENTS.md&#39; -g &#39;*.go&#39; | head -80&#34;</code></pre></td></tr>
<tr><td>2</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;cat rename-plan.md &amp;&amp; \
rg -n \&#34;[A-Za-z_]\&#34; --glob &#39;*.go&#39;&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;cat rename-plan.md &amp;&amp; \
git status --short&#39;</code></pre></td></tr>
<tr><td>3</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;python3 - &lt;&lt;&#39;PY&#39;
from pathlib import Path
import os, tempfile
replacements = {
    &#39;LoadInput&#39;: &#39;ReadRequest&#39;,
    &#39;CheckInput&#39;: &#39;ValidateRequest&#39;,
    &#39;NormalizeValue&#39;: &#39;CanonicalizeValue&#39;,
    &#39;MapFields&#39;: &#39;ProjectFields&#39;,
    &#39;RouteValue&#39;: &#39;DispatchValue&#39;,
}
for path in [Path(&#39;pipeline/processor.go&#39;), *Path(&#39;pipeline&#39;).glob(&#39;stage_*.go&#39;)]:
    original = path.read_text()
    updated = original
    for old, new in replacements.items():
        updated = updated.replace(old, new)
    if updated == original:
        continue
    fd, temporary = tempfile.mkstemp(prefix=f&#39;.{path.name}.&#39;, dir=path.parent)
    try:
        with os.fdopen(fd, &#39;w&#39;) as output:
            output.write(updated)
            output.flush()
            os.fsync(output.fileno())
        os.replace(temporary, path)
    except BaseException:
        try:
            os.unlink(temporary)
        except FileNotFoundError:
            pass
        raise
PY
git diff -- pipeline/processor.go pipeline/stage_input.go pipeline/stage_validation.go pipeline/stage_normalization.go pipeline/stage_mapping.go pipeline/stage_routing.go&#34;</code></pre></td><td><code>semedit/semantic_batch</code> (<span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span>)<pre class="benchmark-tool-arguments"><code class="language-json">{
  &#34;edits&#34;: [
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.LoadInput&#34;,
        &#34;to&#34;: &#34;ReadRequest&#34;,
        &#34;language&#34;: &#34;go&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.CheckInput&#34;,
        &#34;to&#34;: &#34;ValidateRequest&#34;,
        &#34;language&#34;: &#34;go&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.NormalizeValue&#34;,
        &#34;to&#34;: &#34;CanonicalizeValue&#34;,
        &#34;language&#34;: &#34;go&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.MapFields&#34;,
        &#34;to&#34;: &#34;ProjectFields&#34;,
        &#34;language&#34;: &#34;go&#34;
      }
    },
    {
      &#34;tool&#34;: &#34;semantic_rename&#34;,
      &#34;params&#34;: {
        &#34;symbol&#34;: &#34;Processor.RouteValue&#34;,
        &#34;to&#34;: &#34;DispatchValue&#34;,
        &#34;language&#34;: &#34;go&#34;
      }
    }
  ]
}</code></pre></td></tr>
<tr><td>4</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#34;git status --short &amp;&amp; \
cat pipeline/processor.go &amp;&amp; \
cat pipeline/stage_input.go &amp;&amp; \
rg -n &#39;LoadInput|CheckInput|NormalizeValue|MapFields|RouteValue|ReadRequest|ValidateRequest|CanonicalizeValue|ProjectFields|DispatchValue&#39; pipeline&#34;</code></pre></td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td></tr>
<tr><td>5</td><td><span role="img" aria-label="Transport succeeded" title="Transport succeeded">✓</span> <span role="img" aria-label="Functional succeeded" title="Functional succeeded">✓</span><pre class="benchmark-shell-command"><code class="language-shell">/bin/zsh -lc &#39;go test ./...&#39;</code></pre></td><td>—</td></tr>
</tbody>
</table>
