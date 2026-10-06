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

* **Date**: 2026-10-06 09:19:03 CEST

## Test case: `task-13-rename-followup-scale`

### Target: `agy/gemini-3.7-flash-medium (repeat 11)`

#### Configuration: explicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_13_rename_followup_scale.txtar` (`sha256:c23b2265504e34efa29f9269 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Apply only row 1 of rename-plan.md. Rename that Processor method and update its call sites. Leave rows 2 through 16 unchanged. Run go test ./... before finishing. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> For this initial stage only, read row 1 of rename-plan.md, lookup Processor.LoadInput in pipeline/workflow.go, then rename the returned symbol to ReadRequest with semantic_rename. Update its callers and leave rows 2 through 16 untouched. Parallelize independent read-only lookups. Perform mutations serially in the stated order or inside one ordered semantic_batch; never run overlapping mutations concurrently. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package pipeline provides independent processing stages for the rename scale benchmark.
package pipeline

type Processor struct{}

func (Processor) LoadInput(value int) int { return value + 1 }
func (Processor) CheckInput(value int) int { return value + 2 }
func (Processor) NormalizeValue(value int) int { return value + 3 }
func (Processor) MapFields(value int) int { return value + 4 }
func (Processor) RouteValue(value int) int { return value + 5 }
func (Processor) StampEvent(value int) int { return value + 6 }
func (Processor) CountItems(value int) int { return value + 7 }
func (Processor) MergeParts(value int) int { return value + 8 }
func (Processor) FilterNoise(value int) int { return value + 9 }
func (Processor) SortRecords(value int) int { return value + 10 }
func (Processor) EncodePayload(value int) int { return value + 11 }
func (Processor) DecodeReply(value int) int { return value + 12 }
func (Processor) OpenChannel(value int) int { return value + 13 }
func (Processor) CloseChannel(value int) int { return value + 14 }
func (Processor) FlushBuffer(value int) int { return value + 15 }
func (Processor) SaveResult(value int) int { return value + 16 }

func Run(processor Processor, value int) int {
	value = processor.LoadInput(value)
	value = processor.CheckInput(value)
	value = processor.NormalizeValue(value)
	value = processor.MapFields(value)
	value = processor.RouteValue(value)
	value = processor.StampEvent(value)
	value = processor.CountItems(value)
	value = processor.MergeParts(value)
	value = processor.FilterNoise(value)
	value = processor.SortRecords(value)
	value = processor.EncodePayload(value)
	value = processor.DecodeReply(value)
	value = processor.OpenChannel(value)
	value = processor.CloseChannel(value)
	value = processor.FlushBuffer(value)
	value = processor.SaveResult(value)
	return value
}
```
</details>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 2.48s | 2.44s | N/A | — | — | — |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 0 | 0 | 0% | — | — | — |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 0 | 0 | 0% | — | — | — |
| **Output Tokens** | 0 | 0 | 0% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Uncached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached vs Uncached Token Ratio** | — | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

#### Configuration: implicit prompt · prescriptive MCP instructions · write semedit restriction

* **Fixture**: `testdata/bench/task_13_rename_followup_scale.txtar` (`sha256:c23b2265504e34efa29f9269 (uncommitted)`)

**Vanilla LLM Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Apply only row 1 of rename-plan.md. Rename that Processor method and update its call sites. Leave rows 2 through 16 unchanged. Run go test ./... before finishing. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Do not edit tests. You are forbidden to modify protected files: "go.mod", "rename-plan.md".
>
> Apply only row 1 of rename-plan.md. Rename that Processor method and update its call sites. Leave rows 2 through 16 unchanged. Run go test ./... before finishing. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
// Package pipeline provides independent processing stages for the rename scale benchmark.
package pipeline

type Processor struct{}

func (Processor) LoadInput(value int) int { return value + 1 }
func (Processor) CheckInput(value int) int { return value + 2 }
func (Processor) NormalizeValue(value int) int { return value + 3 }
func (Processor) MapFields(value int) int { return value + 4 }
func (Processor) RouteValue(value int) int { return value + 5 }
func (Processor) StampEvent(value int) int { return value + 6 }
func (Processor) CountItems(value int) int { return value + 7 }
func (Processor) MergeParts(value int) int { return value + 8 }
func (Processor) FilterNoise(value int) int { return value + 9 }
func (Processor) SortRecords(value int) int { return value + 10 }
func (Processor) EncodePayload(value int) int { return value + 11 }
func (Processor) DecodeReply(value int) int { return value + 12 }
func (Processor) OpenChannel(value int) int { return value + 13 }
func (Processor) CloseChannel(value int) int { return value + 14 }
func (Processor) FlushBuffer(value int) int { return value + 15 }
func (Processor) SaveResult(value int) int { return value + 16 }

func Run(processor Processor, value int) int {
	value = processor.LoadInput(value)
	value = processor.CheckInput(value)
	value = processor.NormalizeValue(value)
	value = processor.MapFields(value)
	value = processor.RouteValue(value)
	value = processor.StampEvent(value)
	value = processor.CountItems(value)
	value = processor.MergeParts(value)
	value = processor.FilterNoise(value)
	value = processor.SortRecords(value)
	value = processor.EncodePayload(value)
	value = processor.DecodeReply(value)
	value = processor.OpenChannel(value)
	value = processor.CloseChannel(value)
	value = processor.FlushBuffer(value)
	value = processor.SaveResult(value)
	return value
}
```
</details>

| Metric | Vanilla (Small) | <span role="img" aria-label="Semantic tool invocation not verified" title="Semantic tool invocation not verified">⚠</span> MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 2.48s | 5.46s | N/A | — | — | — |
| **Process Start → First Event** | — | — | — | — | — | — |
| **First Event → First Tool Call** | — | — | — | — | — | — |
| **MCP Initialize → First Semantic Call** | — | — | — | — | — | — |
| **MCP Server Start → Initialize** | — | — | — | — | — | — |
| **Top-Level User Turns** | 1 | 1 | N/A | — | — | — |
| **Internal Tool Cycles** | 0 | 0 | 0% | — | — | — |
| **Initial Load / Discovery Turns** | 0 | 0 | 0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 0 | 0 | 0% | — | — | — |
| **Output Tokens** | 0 | 0 | 0% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Uncached Input Tokens** | 0 | 0 | 0% | — | — | — |
| **Cached vs Uncached Token Ratio** | — | — | — | — | — | — |
| **Oracle L1: Mutation Policy** | — | — | — | — | — | — |
| **Oracle L2: AST Invariants** | — | — | — | — | — | — |
| **Oracle L3: Clean Build** | — | — | — | — | — | — |
| **Oracle L4: Verification Test** | — | — | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ⚠️ NO (Fallback) | — | — | — | — |

