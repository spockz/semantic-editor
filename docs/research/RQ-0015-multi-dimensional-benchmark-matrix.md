# RQ-0015: Multi-Dimensional Benchmark Matrix & Evaluation Runner

* **Status**: Open
* **Category**: Benchmarks & Empirical Evaluation
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

Benchmarking `semedit` against baseline text-diff workflows is not a simple 1:1 comparison. Different agent harnesses (Antigravity, Claude Code, Cursor, Aider, OpenHands) possess varying levels of built-in system prompt engineering, context pruning, and automated error-recovery heuristics.

Furthermore, model reasoning levels (extended thinking budgets from 0 to 32k tokens) drastically influence whether a model plans comprehensively upfront or jumps directly into iterative trial-and-error editing.

To produce scientifically sound, reproducible metrics, `semedit` requires a formalized multi-dimensional evaluation matrix and a decoupled runner architecture.

---

## 2. The Evaluation Matrix Formulation

Every benchmark execution represents an evaluation tuple across 5 input dimensions:

$$\mathbf{Run} = \langle \text{Harness}, \text{Model}, \text{ReasoningLevel}, \text{IngressMode}, \text{TestCase} \rangle \longrightarrow \mathbf{Result}$$

```text
   INPUT DIMENSIONS                                                    OUTPUT RESULT
┌──────────────────────────────────────────────────┐                ┌───────────────────────────────────┐
│ Harness        : agy | claude-code | direct-api  │                │ Outcome                           │
│ Model          : claude-3-7 | gpt-4o | qwen-2.5  │                │ - Compilation Clean (exit 0)      │
│ ReasoningLevel : none | low | medium | high      │ ─────────────► │ - Semantic Correctness (AST match)│
│ IngressMode    : baseline-diff | semedit-mcp/cli │                │ - First-Pass Success Rate         │
│ TestCase       : {source, prompt, oracle}        │                │ - Turns to Completion             │
└──────────────────────────────────────────────────┘                │ Latency & Cost                    │
                                                                    │ - Wall-Clock Duration (seconds)   │
                                                                    │ - Input / Output / Thinking Tokens│
                                                                    │ - Total Run Cost ($USD)           │
                                                                    └───────────────────────────────────┘
```

---

## 3. Dimension Specifications

### A. Harness ($H$)

Harnesses differ in their middleware intelligence:

* **`direct-api` (Control Baseline)**: Minimalist headless runner (Python/Go) executing a raw tool loop directly against model APIs with a neutral system prompt. Isolates model capability from harness heuristics.
* **`agy` (Antigravity)**: Production IDE/CLI harness featuring rich tool schemas, subagent orchestration, and session transcripts.
* **`claude-code` / `aider`**: CLI harnesses with built-in diff patchers, compact error reporting, and git integration.
* **`local-runner`**: Direct driver communicating with `llama.cpp` or Ollama servers for zero-cost, local Apple Silicon runs.

### B. Model ($M$)

* **Frontier Cloud**: Claude 3.5 Sonnet, Claude 3.7 Sonnet, GPT-4o, Gemini 2.0 Flash / Pro.
* **Local OSS**: Qwen 2.5 Coder (7B, 14B, 32B), DeepSeek-Coder V2.

### C. Reasoning Level ($R$)

* `none`: Standard generation (temperature = 0, no thinking tokens).
* `low`: ~1,000 reasoning tokens (quick sanity check).
* `medium`: ~4,000–8,000 reasoning tokens.
* `high`: ~16,000–32,000 reasoning tokens (deep architectural refactoring).

### D. Ingress / Toolset ($I$)

* `baseline-diff`: Standard text tools (`view_file`, `replace_file_content`, `grep_search`).
* `semedit-mcp`: Full intent-based MCP tools (`semantic_rename`, `organize_imports`, `verify_diagnostics`).
* `semedit-cli`: Agent issues bash commands invoking `semedit` CLI subcommands.

### E. TestCase ($T$)

Defined as a standalone fixture containing:

1. `source`: Git commit or `txtar` archive of the initial repository state.
2. `prompt`: The natural language request presented to the agent.
3. `oracle`: Acceptance criteria:
   * **Syntactic**: `go build ./...` exits 0.
   * **Behavioral**: `go test ./...` passes.
   * **Semantic**: Target AST symbols match expected locations; old symbols have 0 occurrences.

---

## 4. Output Result Schema (`eval_result.json`)

```json
{
  "run_id": "run_20260915_qwen25_semedit_001",
  "dimensions": {
    "harness": "direct-api",
    "model": "qwen2.5-coder-14b-instruct",
    "reasoning_level": "none",
    "ingress_mode": "semedit-mcp",
    "test_case_id": "go-rename-cross-package-10files"
  },
  "outcome": {
    "status": "SUCCESS",
    "compiled": true,
    "tests_passed": true,
    "first_pass_clean": true,
    "turns_to_success": 1,
    "tool_calls_breakdown": {
      "semantic_rename": 1,
      "verify_diagnostics": 1
    }
  },
  "resource_metrics": {
    "wall_clock_seconds": 2.45,
    "tokens": {
      "prompt_tokens": 1240,
      "cached_tokens": 0,
      "completion_tokens": 28,
      "reasoning_tokens": 0,
      "total_tokens": 1268
    },
    "estimated_cost_usd": 0.00003
  }
}
```

---

## 5. Runner Architecture & Execution Strategy

### A. Sandboxed Execution Pipeline

1. **Workspace Isolation**: Each run instantiates a clean git worktree or isolated temporary directory (`t.TempDir()`). Never execute benchmarks in the working tree.
2. **Matrix Combinator**: The runner CLI accepts matrix filters:

   ```bash
   semedit-bench run \
     --harness=direct-api,agy \
     --model=qwen2.5-coder-7b,claude-3-7-sonnet \
     --reasoning=none,medium \
     --suite=core-refactoring \
     --output=results/run_01.jsonl
   ```

3. **Automated Verification**:
   * The runner automatically checks `go build ./...` and `go test ./...` post-run.
   * Compares the resulting AST symbol table against the pre-compiled oracle.

### B. Normalizing for Harness Intelligence

* By comparing `direct-api` (pure model + tools) against `agy` / `claude-code`, the benchmark isolates:
  1. How much the **tooling itself** improves the outcome.
  2. How much the **harness scaffolding** compensates for poor tools or models.

---

## 6. Prior Art & References

* **SWE-bench**: Standard benchmark for agentic issue resolution on GitHub repositories.
* **Aider LLM Leaderboards**: Refactoring benchmarks comparing unified diff vs. whole-file rewrite formats.
* **HumanEval / MultiPL-E**: Unit test-based code generation benchmarks.
