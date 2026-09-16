# RQ-0015: Multi-Dimensional Benchmark Matrix & Evaluation Runner

* **Status**: Open
* **Category**: Benchmarks & Empirical Evaluation
* **Last Updated**: 2026-09-16

---

## 1. Problem Context

Benchmarking `semedit` against baseline text-diff workflows requires evaluating multiple interacting variables. Different agent harnesses (Antigravity, Claude Code, Cursor, Aider, OpenHands) incorporate distinct system prompt designs, context pruning heuristics, and automated error-recovery loops.

Furthermore, model reasoning budgets (extended thinking allocations from 0 to 32k tokens) alter agent behavioral profiles: models with higher reasoning allocations plan multi-file refactorings upfront, whereas models without reasoning tokens engage in immediate iterative trial-and-error editing.

Evaluating `semedit` across these disparate dimensions demands a formalized evaluation matrix, an unambiguous testcase fixture design, and a multi-level correctness oracle.

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

Harnesses introduce distinct middleware layers that influence agent trajectory:

* **`direct-api` (Control Baseline)**: Minimalist headless runner (Python/Go) executing a raw tool loop directly against model APIs with a neutral system prompt. Isolates model capability from harness heuristics.
* **`agy` (Antigravity)**: Production IDE/CLI harness featuring rich tool schemas, subagent orchestration, and session transcripts.
* **`claude-code` / `aider`**: CLI harnesses with built-in diff patchers, compact error reporting, and git integration.
* **`local-runner`**: Direct driver communicating with `llama.cpp` or Ollama servers for zero-cost, local Apple Silicon runs.

### B. Model ($M$)

* **Frontier Cloud**: Claude 3.5 Sonnet, Claude 3.7 Sonnet, GPT-4o, Gemini 2.0 Flash / Pro.
* **Local Open-Weights**: Qwen 2.5 Coder (7B, 14B, 32B), DeepSeek-Coder V2.

### C. Reasoning Level ($R$)

* `none`: Standard generation (temperature = 0, no thinking tokens).
* `low`: ~1,000 reasoning tokens (quick sanity validation).
* `medium`: ~4,000 to 8,000 reasoning tokens.
* `high`: ~16,000 to 32,000 reasoning tokens (deep structural refactoring planning).

### D. Ingress / Toolset ($I$)

* `baseline-diff`: Standard text tools (`view_file`, `replace_file_content`, `grep_search`).
* `semedit-mcp`: Full intent-based MCP tools (`semantic_rename`, `organize_imports`, `insert_declaration`, `verify_diagnostics`).
* `semedit-cli`: Agent executes bash commands invoking `semedit` CLI subcommands (`semedit rename`, `semedit insert`, `semedit organize-imports`).

### E. TestCase Dimension ($T$) & Fixture Design Trade-Offs

The benchmark suite balances hermetic execution against real-world repository scale through two distinct fixture architectures:

#### 1. Synthetic `txtar` Scenarios

* **Properties**: Single-file multi-part archives containing `go.mod`, package sources, test files, and oracle assertions.
* **Advantages**:
  * Near-instant instantiation (<50ms) in temporary directories (`t.TempDir()`).
  * Zero external network or package repository dependencies.
  * Precise, hermetic attribution of failure modes.
* **Trade-Offs**: Omits the large symbol tables, deep vendor trees, and cross-module link times present in enterprise repositories.

#### 2. Real-World Open-Source Go Repositories

* **Target Subsystems**: Pinned Git commits of established open-source libraries:
  * `go-chi/chi`: Lightweight HTTP router exercising interface implementations and middleware handlers.
  * `gin-gonic/gin`: Web framework featuring extensive test suites and context trees.
  * `kubernetes/kubernetes` (sub-packages such as `staging/src/k8s.io/client-go`): Deep package graphs and massive symbol tables.
* **Advantages**:
  * Exposes agent search heuristics to realistic code layouts and cross-package references.
  * Validates whether semantic tools maintain low latency on non-trivial AST parse trees.
* **Trade-Offs**: High disk usage; slower test execution (`go test` takes seconds to minutes); external dependency fetch requirements.

#### 3. Two-Tier Suite Strategy

* **Tier 1 (Hermetic Core Suite)**: 10 synthetic `txtar` scenarios running in seconds during CI test runs (`make test`).
* **Tier 2 (Real-World Benchmark Suite)**: 5 real repository tasks using pinned Git submodules, evaluated during dedicated release benchmarking.

---

## 4. Initial 10-Task Benchmark Suite Specification

The Tier 1 suite standardizes 10 canonical refactoring tasks spanning rename, declaration insertion, import optimization, and diagnostic recovery:

| Task ID | Category | Initial Source Fixture | Description & Natural Language Prompt | Semantic Invariants Checked |
| :--- | :--- | :--- | :--- | :--- |
| `task-01-rename-local` | Rename | `testdata/bench/task_01_rename_local.txtar` | "Rename the unexported method `Server.oldName` to `Server.newName` and update call-sites." | `Server.newName` exists; `oldName` absent; clean compile. |
| `task-02-rename-cross-pkg` | Rename | `testdata/bench/task_02_rename_cross_pkg.txtar` | "Rename public function `auth.ValidateToken` to `auth.VerifyToken` across all consuming packages." | Call sites in `cmd/main.go` resolve to `VerifyToken`; clean test pass. |
| `task-03-rename-interface` | Rename | `testdata/bench/task_03_rename_interface.txtar` | "Rename interface method `Storage.Save` to `Storage.Persist` along with all struct implementations." | Concrete struct receivers and interface callers updated consistently. |
| `task-04-insert-public` | Insert | `testdata/bench/task_04_insert_public.txtar` | "Add public constructor `func InitServer() *Server` placed before private helpers." | Constructor placed in public declarations section; package builds cleanly. |
| `task-05-insert-method` | Insert | `testdata/bench/task_05_insert_method.txtar` | "Add a new method `func (s *Server) Stop()` placed immediately after `Server.Start`." | Method receiver attaches to `Server`; position matches adjacency rule. |
| `task-06-imports-cleanup` | Imports | `testdata/bench/task_06_imports_cleanup.txtar` | "Clean up unused standard library imports and group third-party imports." | Unused imports removed; stdlib precedes third-party; `goimports` clean. |
| `task-07-imports-qualify` | Imports | `testdata/bench/task_07_imports_qualify.txtar` | "Import `github.com/google/uuid` and initialize a UUID inside `GenerateID()`." | Import statement inserted; `uuid.NewString()` resolves without diagnostics. |
| `task-08-delta-recovery` | Recovery | `testdata/bench/task_08_delta_recovery.txtar` | "Repair signature mismatch compile errors between caller and callee in `service/`." | Compiler diagnostic count drops from $N > 0$ to $0$; tests pass. |
| `task-09-composite-refactor` | Multi-Step | `testdata/bench/task_09_composite_refactor.txtar` | "Rename `Config.Port`, insert helper `func DefaultConfig()`, and clean imports." | All three operations succeed without reverting intermediate states. |
| `task-10-disambiguate-var` | Semantics | `testdata/bench/task_10_disambiguate_var.txtar` | "Disambiguate shadowing variable `ctx` in handler without altering package context." | Target variable renamed; parent scope variables remain untouched. |

---

## 5. Correctness Oracle Architecture

Evaluating agent task completion requires strict semantic validation rather than superficial exit codes or brittle line diffs.

```text
                     Agent Output Code
                            │
                            ▼
              ┌───────────────────────────┐
              │  Level 1: AST Invariants  │ ──► FAIL: Obsolete symbol present or
              │  - Target symbol exists   │           target declaration missing
              │  - Zero stale occurrences │
              └─────────────┬─────────────┘
                            │ PASS
                            ▼
              ┌───────────────────────────┐
              │  Level 2: Compiler Clean  │ ──► FAIL: Syntax or typecheck errors
              │  - go build ./... exits 0 │           (diagnostic delta > 0)
              └─────────────┬─────────────┘
                            │ PASS
                            ▼
              ┌───────────────────────────┐
              │  Level 3: Behavioral Test │ ──► FAIL: Unit or integration test
              │  - go test ./... exits 0  │           assertion failure
              └─────────────┬─────────────┘
                            │ PASS
                            ▼
              ┌───────────────────────────┐
              │  Level 4: Diff Audit Log  │ ──► PASS: Record line churn and
              │  - git diff recorded      │           patch footprint metrics
              └───────────────────────────┘
```

### Oracle Level Comparison

1. **Compiler Pass/Fail (`go build ./...`)**:
   * *Strengths*: Fast, universal. Catches syntax errors, unresolved identifiers, and type mismatches.
   * *Limitations*: An agent can achieve a passing build by deleting the caller, leaving a stub, or reverting the file entirely.
2. **Behavioral Test Suite (`go test ./...`)**:
   * *Strengths*: Confirms runtime correctness and preserves external behavior.
   * *Limitations*: Tests only cover paths exercise by assertions. Refactorings on untested code paths pass despite incomplete symbol renames.
3. **AST Semantic Invariant Verification (Core `semedit` Oracle)**:
   * *Strengths*: Inspects the AST directly via Go parser APIs (`go/parser`, `go/types`). Asserts:
     * Declared identifier matches expected new name.
     * All references across packages resolve to the expected object definition.
     * Target obsolete identifier occurrence count equals zero.
     * Whitespace, comment layout, and irrelevant formatting differences do not trigger false failures.
   * *Limitations*: Requires authoring specific AST assertion rules per benchmark task.
4. **Golden Diff Comparison (`cmp` / `git diff`)**:
   * *Strengths*: Fast byte comparison against human golden reference.
   * *Limitations*: Overly fragile; fails valid solutions due to harmless comment rewrites or alternate blank line spacing. Used strictly for audit telemetry rather than pass/fail gating.

---

## 6. Harness Non-Determinism Management in Matrix Evaluations

Stochastic variability in language model outputs requires systematic statistical controls across matrix evaluations:

1. **Deterministic Sampling Parameters**: Set `temperature = 0.0` and anchor provider-supported seed values across all arms.
2. **Replication Budget**: Run $N \ge 5$ independent trials for each $\langle H, M, R, I, T \rangle$ configuration.
3. **Statistical Robustness**:
   * Report median and interquartile range (IQR) for token counts and execution latency to minimize outlier skew.
   * Calculate 95% bootstrap confidence intervals across repeated runs.
4. **Success Probability Metrics**:
   * **$Pass@1$**: Proportion of runs achieving full Level 1-3 oracle pass on the initial attempt.
   * **$Pass@k$**: Probability that at least one of $k$ repeated trials satisfies the oracle, computed as:
     $$Pass@k = 1 - \frac{\binom{N - C}{k}}{\binom{N}{k}}$$
     where $N$ is total trials and $C$ is passing trials.

---

## 7. Output Result Schema (`eval_result.json`)

Each benchmark run emits a single JSON record capturing configuration, outcomes, and resource metrics:

```json
{
  "run_id": "run_20260916_qwen25_semedit_001",
  "dimensions": {
    "harness": "direct-api",
    "model": "qwen2.5-coder-14b-instruct",
    "reasoning_level": "none",
    "ingress_mode": "semedit-mcp",
    "test_case_id": "task-02-rename-cross-pkg"
  },
  "outcome": {
    "status": "SUCCESS",
    "ast_verified": true,
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

## 8. Prior Art & References

* **SWE-bench**: Standard benchmark for autonomous repository issue resolution on GitHub.
* **Aider LLM Leaderboards**: Refactoring benchmarks comparing unified diff formats with whole-file replacements.
* **HumanEval / MultiPL-E**: Unit test evaluation harnesses for code synthesis models.
* **Go `testscript`**: Hermetic shell script and archive testing engine for CLI workflows.
