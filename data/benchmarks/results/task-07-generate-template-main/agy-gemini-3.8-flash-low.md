# Empirical Benchmark Report: Vanilla LLM vs. Semedit MCP

* **Date**: 2026-09-18 08:54:10 CEST

## Task: `task-07-generate-template-main` (Prompt: `crypto_rand`) | Target: `agy/gemini-3.8-flash-low`

* **Fixture**: `testdata/bench/task-07-generate-template-main.txtar`

**Vanilla LLM Prompt**:
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer from crypto/rand, ensuring all necessary standard library packages are imported cleanly. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer from crypto/rand, ensuring all necessary standard library packages are imported cleanly. Prefer using semantic editor operations if applicable. When done, output DONE.

**Vanilla (Verified) Prompt**:
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer from crypto/rand, ensuring all necessary standard library packages are imported cleanly. Verify your changes and ensure 'go build' succeeds with zero errors before concluding. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP (Verified) Prompt**:
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer from crypto/rand, ensuring all necessary standard library packages are imported cleanly. Verify your changes and ensure 'go build' succeeds with zero errors before concluding. Prefer using semantic editor operations if applicable. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package main

func main() {
}
```
</details>

### Standard Directive Comparison

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 44.60s | 52.54s | +17.8% | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 7 | 24 | +242.9% | — | — | — |
| **Initial Load / Discovery Turns** | 3 | 4 | +33.3% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 6 | 23 | +283.3% | — | — | — |
| **Output Tokens** | 841 | 2751 | +227.1% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 67233 | 141083 | +109.8% | — | — | — |
| **Cached Input Tokens** | 101743 | 531948 | +422.8% | — | — | — |
| **Uncached Input Tokens** | 67233 | 141083 | +109.8% | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

#### Small Context Edit Summary
* **Vanilla Edit**: File main.go modified (4 lines -> 24 lines)
* **Vanilla Tools**: `view_file`, `run_command`, `view_file`, `write_to_file`, `run_command`, `run_command`
* **MCP Edit**: File main.go modified (4 lines -> 24 lines)
* **MCP Tools**: `view_file`, `view_file`, `run_command`, `view_file`, `semantic_replace_body`, `semantic_replace_body`, `semantic_replace_body`, `view_file`, `semantic_replace_body`, `view_file`, `view_file`, `view_file`, `semantic_organize_imports`, `view_file`, `semantic_replace_body`, `semantic_organize_imports`, `view_file`, `semantic_organize_imports`, `view_file`, `run_command`, `run_command`, `run_command`, `view_file`

### Verified Directive Comparison (+Self-Correction Loop)

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 20.62s | 35.04s | +69.9% | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 9 | 12 | +33.3% | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 3 | +200.0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 11 | +37.5% | — | — | — |
| **Output Tokens** | 1095 | 1357 | +23.9% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 58254 | 92592 | +58.9% | — | — | — |
| **Cached Input Tokens** | 162728 | 207373 | +27.4% | — | — | — |
| **Uncached Input Tokens** | 58254 | 92592 | +58.9% | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

#### Small (+Verified) Context Edit Summary
* **Vanilla Edit**: File main.go modified (4 lines -> 24 lines)
* **Vanilla Tools**: `view_file`, `write_to_file`, `run_command`, `run_command`, `run_command`, `run_command`, `run_command`, `run_command`
* **MCP Edit**: File main.go modified (4 lines -> 23 lines)
* **MCP Tools**: `view_file`, `run_command`, `view_file`, `semantic_replace_body`, `semantic_replace_body`, `view_file`, `run_command`, `run_command`, `run_command`, `run_command`, `run_command`


---

## Task: `task-07-generate-template-main` (Prompt: `default`) | Target: `agy/gemini-3.8-flash-low`

* **Fixture**: `testdata/bench/task-07-generate-template-main.txtar`

**Vanilla LLM Prompt**:
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP Prompt**:
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Prefer using semantic editor operations if applicable. When done, output DONE.

**Vanilla (Verified) Prompt**:
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Verify your changes and ensure 'go build' succeeds with zero errors before concluding. Do not use semantic editing MCP tools; use standard file editing. When done, output DONE.

**Semedit MCP (Verified) Prompt**:
> In main.go, implement main() to execute a text/template that prints 'Hello World' along with a random integer, ensuring all necessary standard library packages are imported cleanly. Verify your changes and ensure 'go build' succeeds with zero errors before concluding. Prefer using semantic editor operations if applicable. When done, output DONE.

<details><summary><b>Initial Workspace State (Before Edit)</b></summary>

```go
package main

func main() {
}
```
</details>

### Standard Directive Comparison

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 10.35s | 32.43s | +213.4% | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 4 | 12 | +200.0% | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 2 | +100.0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 3 | 11 | +266.7% | — | — | — |
| **Output Tokens** | 444 | 1248 | +181.1% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 53751 | 84528 | +57.3% | — | — | — |
| **Cached Input Tokens** | 40706 | 219576 | +439.4% | — | — | — |
| **Uncached Input Tokens** | 13045 | 84528 | +548.0% | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

#### Small Context Edit Summary
* **Vanilla Edit**: File main.go modified (4 lines -> 14 lines)
* **Vanilla Tools**: `view_file`, `write_to_file`, `run_command`
* **MCP Edit**: File main.go modified (4 lines -> 18 lines)
* **MCP Tools**: `view_file`, `view_file`, `semantic_replace_body`, `semantic_replace_body`, `view_file`, `view_file`, `semantic_organize_imports`, `semantic_replace_body`, `view_file`, `run_command`, `semantic_verify`

### Verified Directive Comparison (+Self-Correction Loop)

| Metric | Vanilla (Small) | MCP (Small) | Δ (Small) | Vanilla (Large) | MCP (Large) | Δ (Large) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Wall-Clock Latency** | 17.40s | 46.51s | +167.4% | — | — | — |
| **Top-Level User Turns** | 1 | 1 | 0% | — | — | — |
| **Internal Tool Cycles** | 9 | 16 | +77.8% | — | — | — |
| **Initial Load / Discovery Turns** | 1 | 2 | +100.0% | — | — | — |
| **MCP Discovery / Schema Turns** | 0 | 0 | 0% | — | — | — |
| **Total Tool Invocations** | 8 | 15 | +87.5% | — | — | — |
| **Output Tokens** | 1031 | 1569 | +52.2% | — | — | — |
| **Reasoning / Thinking Tokens** | 0 | 0 | 0% | — | — | — |
| **Total Input Tokens** | 57898 | 122416 | +111.4% | — | — | — |
| **Cached Input Tokens** | 162725 | 292587 | +79.8% | — | — | — |
| **Uncached Input Tokens** | 57898 | 122416 | +111.4% | — | — | — |
| **Oracle L1: Mutation Policy** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L2: AST Invariants** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L3: Clean Build** | ✅ PASS | ✅ PASS | — | — | — | — |
| **Oracle L4: Verification Test** | ✅ PASS | ✅ PASS | — | — | — | — |
| **MCP Tools Invocation Verified** | ✅ N/A (Vanilla) | ✅ YES | — | — | — | — |

#### Small (+Verified) Context Edit Summary
* **Vanilla Edit**: File main.go modified (4 lines -> 17 lines)
* **Vanilla Tools**: `view_file`, `write_to_file`, `run_command`, `run_command`, `run_command`, `run_command`, `run_command`, `run_command`
* **MCP Edit**: File main.go modified (4 lines -> 19 lines)
* **MCP Tools**: `view_file`, `view_file`, `semantic_replace_body`, `view_file`, `view_file`, `semantic_organize_imports`, `view_file`, `view_file`, `semantic_verify`, `run_command`, `run_command`, `run_command`, `run_command`, `run_command`, `view_file`


---

