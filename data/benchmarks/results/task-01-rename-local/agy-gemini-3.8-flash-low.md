# Empirical Benchmark Report: Vanilla LLM vs. Semedit MCP

* **Date**: 2026-09-18 08:54:10 CEST

## Task: `task-01-rename-local` | Target: `agy/gemini-3.8-flash-low`

* **Fixture**: `testdata/bench/task_01_rename_local.txtar` (`sha256:50c61232c233e7d42ed86846 (uncommitted)`)

**Vanilla LLM Prompt**:
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Prefer using semantic editor operations if applicable. When done, output DONE.

**Vanilla (Verified) Prompt**:
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Verify your changes and ensure 'go test ./...' passes with zero errors before concluding. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP (Verified) Prompt**:
> Rename the unexported method Server.oldName to Server.newName and update all call-sites. Verify your changes and ensure 'go test ./...' passes with zero errors before concluding. Prefer using semantic editor operations if applicable. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package api

type Server struct{}

func (s *Server) oldName() string {
	return "ok"
}

func Run() string {
	s := &Server{}
	return s.oldName()
}
```
</details>

### Standard Directive Comparison

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 18.23s | 16.83s | **-7.7%** | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 6 | 8 | +33.3% | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 4 | +100.0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 5 | 7 | +40.0% | — | — | — |
| **Output Tokens** | 698 | 714 | +2.3% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 86460 | 52690 | **-39.1%** | — | — | — |
| **Cached Input Tokens** | 93553 | 142408 | +52.2% | — | — | — |
| **Uncached Input Tokens** | 86460 | 52690 | **-39.1%** | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

#### Small Context Edit Summary
* **Vanilla Edit**: File api/server.go modified (12 lines -> 12 lines)
* **Vanilla Tools**: `grep_search`, `view_file`, `replace_file_content`, `replace_file_content`, `run_command`
* **MCP Edit**: File api/server.go modified (12 lines -> 12 lines)
* **MCP Tools**: `view_file`, `find_by_name`, `view_file`, `view_file`, `semantic_rename`, `view_file`, `run_command`

### Verified Directive Comparison (+Self-Correction Loop)

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 28.57s | 15.74s | **-44.9%** | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 8 | 8 | 0% | — | — | — |
| **Initial Load / Discovery Turns** | 2 | 3 | +50.0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 7 | 7 | 0% | — | — | — |
| **Output Tokens** | 945 | 705 | **-25.4%** | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 70508 | 53330 | **-24.4%** | — | — | — |
| **Cached Input Tokens** | 134310 | 142412 | +6.0% | — | — | — |
| **Uncached Input Tokens** | 70508 | 53330 | **-24.4%** | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

#### Small (+Verified) Context Edit Summary
* **Vanilla Edit**: File api/server.go modified (12 lines -> 12 lines)
* **Vanilla Tools**: `grep_search`, `view_file`, `replace_file_content`, `run_command`, `run_command`, `run_command`, `view_file`
* **MCP Edit**: File api/server.go modified (12 lines -> 12 lines)
* **MCP Tools**: `grep_search`, `view_file`, `view_file`, `semantic_rename`, `view_file`, `run_command`, `run_command`


---

