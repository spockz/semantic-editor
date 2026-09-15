# RQ-0014: Benchmark Methodology, Token/Latency Measurement & Eval Harness

* **Status**: Open
* **Category**: Benchmarks & Empirical Evaluation
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

Marketing claims of "$100\times$ fewer tokens" and "$10\times$ faster refactorings" are analytically derived from token mechanics, but require rigorous empirical proof through automated A/B evaluations across real codebases.

To prove these claims, `semedit` requires an automated evaluation harness (`semedit-bench`) capable of running head-to-head trials against baseline agent editing tools, tracking token expenditure, turn latency, and success rates.

---

## 2. Analytical Derivation vs. Empirical Reality

### A. The Baseline Text-Diff Tax (Observed OSS Agent Behavior)

In existing harnesses (SWE-bench, Claude Code, Cursor, Aider), a multi-file refactoring (e.g. renaming a method across 10 files) incurs:

1. **Search Phase**: 1–2 turns of `grep_search` / `findReferences` (~1,000 prompt tokens).
2. **Read Phase**: 5–10 turns of `view_file` to locate coordinates (~10,000 prompt tokens).
3. **Write Phase**: 10 distinct `replace_file_content` or diff patch tool calls:
   * Each turn emits 100–300 output tokens of verbatim code chunks (~2,000 output tokens total).
   * **Quadratic Context Accumulation**: Because agent harnesses resend prior conversation history on every turn, a 15-turn task accumulates:
     $$\text{Total Input Tokens} = \sum_{t=1}^{N} \text{History}_t \approx 150,000 - 300,000\text{ tokens}$$
4. **Error Recovery**: In 20–30% of cases, whitespace mismatches or forgotten call-sites require 3–5 additional repair turns.

* **Total Cost**: ~2,500 output tokens, ~250k input tokens, 45–90 seconds wall-clock time.

### B. The `semedit` Semantic Intent Model

1. **Turn 1**: Agent calls `semantic_rename(symbol: "Server.Start", new_name: "Serve")`.
   * Model emits ~25 output tokens (arguments only).
   * Host `gopls` executes AST rename across all 10 files in ~120ms.
   * Tool returns structured status and verification delta.

* **Total Cost**: ~25 output tokens ($100\times$ reduction), ~8k input tokens ($25\times$ reduction), ~3 seconds wall-clock time ($15\times-30\times$ faster).

---

## 3. Eval Harness Architecture (`semedit-bench`)

```text
                          ┌────────────────────────┐
                          │  Evaluation Task Suite │
                          │ (txtar / Real Go Repos)│
                          └───────────┬────────────┘
                                      │
                   ┌──────────────────┴──────────────────┐
                   ▼                                     ▼
        ┌─────────────────────┐               ┌─────────────────────┐
        │ Arm A: Baseline     │               │ Arm B: Semedit      │
        │ (grep + file_edit)  │               │ (semantic_rename)   │
        └──────────┬──────────┘               └──────────┬──────────┘
                   │                                     │
                   └──────────────────┬──────────────────┘
                                      ▼
                        ┌───────────────────────────┐
                        │ Metric Extraction Engine  │
                        │ - Output tokens emitted   │
                        │ - Cumulative input tokens │
                        │ - Wall-clock duration     │
                        │ - Turns to success        │
                        │ - First-pass compile rate │
                        └───────────────────────────┘
```

### Metrics Recorded Per Task

1. `output_tokens`: Exact completion tokens emitted by the LLM.
2. `cumulative_input_tokens`: Cumulative sum of prompt tokens across all turns.
3. `turns_to_success`: Number of tool call round-trips to reach a passing state.
4. `wall_clock_ms`: Total execution time from initial prompt to final verification.
5. `first_pass_clean`: Boolean; did `go test ./...` pass immediately without repair turns?
6. `patch_apply_failures`: Count of failed line-matching or syntax errors.

---

## 4. Execution Platform Feasibility

### A. Antigravity (`agy`)

* **Mechanism**: Spawning isolated subagents via `invoke_subagent` into sandboxed git workspaces (`.scratch/benchmarks/run_<id>/`).
* **Telemetry Source**: `<appDataDir>/brain/<subagent-id>/.system_generated/logs/transcript.jsonl`.
* **Data Extracted**: Every turn records step duration, exact tool calls, and output payload sizes.

### B. OpenAI / Codex API

* **Mechanism**: Python test harness (`scripts/bench.py` via `uv`).
* **Telemetry Source**: Official API response payloads (`response.usage.prompt_tokens`, `response.usage.completion_tokens`).
* **Benefit**: Strict, deterministic token billing counts direct from the model provider.

### C. Local Models (`llama.cpp` / Ollama)

* **Mechanism**: `llama.cpp` server mode (`/v1/chat/completions`) running Qwen 2.5 Coder (7B, 14B, or 32B) on Apple Silicon Metal.
* **Telemetry Source**: `llama.cpp` server returns precise timing metrics: `tokens_predicted`, `tokens_evaluated`, `timings/eval_duration`.
* **Benefit**: Zero API cost, completely offline, zero rate-limiting, perfect reproducibility for automated CI regression testing.

---

## 5. Next Steps

1. Create benchmark task fixtures in `testdata/bench/` based on real Go refactorings.
2. Build `scripts/eval_harness.py` supporting both local `llama.cpp` and API backends.
3. Publish benchmark tables in `docs/research/benchmarks/` with empirical numbers before finalizing outward-facing README marketing copy.
