# RQ-0014: Benchmark Methodology, Token/Latency Measurement & Eval Harness

* **Status**: Open
* **Category**: Benchmarks & Empirical Evaluation
* **Last Updated**: 2026-09-16

---

## 1. Problem Context

Analytical derivations predict up to $100\times$ token reductions and $10\times$ speedups when agents leverage semantic AST tools instead of raw text search and replace. Confirming these predictions requires automated empirical evaluation across standardized refactoring tasks.

The `semedit` project requires an automated evaluation harness (`semedit-bench`). This harness runs controlled A/B evaluations against baseline agent editing workflows, recording input and output token consumption, turn latency, patch failures, and completion rates.

---

## 2. Analytical Derivation vs. Empirical Reality

### A. The Baseline Text-Diff Tax (Observed OSS Agent Behavior)

In existing harnesses (SWE-bench, Claude Code, Cursor, Aider), a multi-file refactoring (such as renaming a method across 10 files) incurs substantial overhead:

1. **Search Phase**: 1 to 2 turns of `grep_search` or symbol reference lookups (~1,000 prompt tokens).
2. **Read Phase**: 5 to 10 turns of `view_file` to locate coordinates (~10,000 prompt tokens).
3. **Write Phase**: 10 distinct `replace_file_content` or diff patch tool calls:
   * Each turn emits 100 to 300 output tokens of verbatim code chunks (~2,000 output tokens total).
   * **Quadratic Context Accumulation**: Agent harnesses retransmit prior conversation history on every turn. A 15-turn task accumulates:
     $$\text{Total Input Tokens} = \sum_{t=1}^{N} \text{History}_t \approx 150,000 - 300,000\text{ tokens}$$
4. **Error Recovery**: In 20% to 30% of trials, whitespace mismatches or forgotten call-sites trigger 3 to 5 repair turns.

* **Total Cost**: ~2,500 output tokens, ~250k input tokens, 45 to 90 seconds wall-clock time.

### B. The `semedit` Semantic Intent Model

1. **Turn 1**: The agent emits a single semantic intent call: `semantic_rename(symbol: "Server.Start", new_name: "Serve")`.
   * The model emits ~25 output tokens (arguments only).
   * Host `gopls` executes the AST rename across all 10 files within ~120ms.
   * The tool returns structured status and verification diagnostics.

* **Total Cost**: ~25 output tokens ($100\times$ reduction), ~8k input tokens ($25\times$ reduction), ~3 seconds wall-clock time ($15\times-30\times$ speedup).

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
2. `cumulative_input_tokens`: Cumulative sum of prompt tokens across all conversation turns.
3. `reasoning_tokens`: Thinking tokens generated during extended reasoning phases.
4. `turns_to_success`: Count of model-tool round trips required to reach the passing state.
5. `wall_clock_ms`: Total elapsed time from initial prompt submission to final verification.
6. `first_pass_clean`: Boolean flag indicating whether the initial attempt passed without repair turns.
7. `patch_apply_failures`: Count of failed line matches or malformed diffs during editing.
8. `diagnostic_deltas`: Net compiler diagnostic count before and after execution.

---

## 4. Evaluation Harness Implementation Approaches

Three distinct architectural approaches present viable avenues for building `semedit-bench`.

### Approach A: Go-Native Test Harness

* **Mechanism**: Leverages Go standard testing libraries, the `testscript`/`txtar` engine (`github.com/rogpeppe/go-internal/testscript`), and `os/exec`.
* **Execution Flow**: Each test script defines an isolated workspace via `txtar`. The Go runner boots `semedit` as an internal sub-process or direct library call, orchestrating agent turns through an internal loop.
* **Strengths**: Highest portability; requires zero dependencies beyond the Go toolchain. Integrates directly into `make test` and standard CI pipelines. Ensures hermetic execution inside temporary directories (`t.TempDir()`).
* **Weaknesses**: Demands a custom LLM API client loop in Go; lacks off-the-shelf Python tokenizer libraries (`tiktoken`, Hugging Face `tokenizers`).

### Approach B: Standalone Python/uv Runner

* **Mechanism**: A dedicated runner package managed by `uv` (`uv run bench.py`). Uses standard LLM SDKs (`openai`, `anthropic`, `litellm`) to drive chat completion loops.
* **Execution Flow**: Python scripts configure model endpoints (OpenAI-compatible APIs, local `llama.cpp` server, or Ollama). The runner exposes tools (Arm A text edit functions vs. Arm B `semedit` CLI/MCP commands), inspects response payloads, and controls child processes.
* **Strengths**: Unmatched token tracking fidelity; provider response payloads report authoritative usage data (`usage.prompt_tokens`, `usage.completion_tokens_details.reasoning_tokens`). Access to mature visualization libraries (`matplotlib`, `tabulate`, `pandas`) simplifies report generation.
* **Weaknesses**: Introduces Python and `uv` dependencies into a Go repository. Requires dual-language maintenance.

### Approach C: In-Tree CLI Benchmark Command (`semedit bench`)

* **Mechanism**: Implements a native Cobra subcommand (`semedit bench --matrix ...`) directly inside the `semedit` binary (consistent with ADR-0014).
* **Execution Flow**: The CLI reads task fixtures from `testdata/bench/`, creates ephemeral scratch workspaces in `.scratch/benchmarks/`, executes the A/B evaluation loop against configured endpoints, and emits structured JSONL telemetry.
* **Strengths**: Zero external setup friction; distribution occurs via the single compiled binary. Developers run benchmarks anywhere without managing Python virtual environments or external harnesses.
* **Weaknesses**: Increases `semedit` binary footprint with evaluation code. Demands careful design to avoid coupling core AST transformation packages to benchmarking harness logic.

### Comparative Assessment

| Evaluation Criterion | Approach A: Go-Native (`testscript`) | Approach B: Python/uv Runner | Approach C: In-Tree CLI (`semedit bench`) |
| :--- | :--- | :--- | :--- |
| **Portability** | High (Pure Go toolchain) | Moderate (Requires Python 3.12+ and `uv`) | High (Self-contained binary) |
| **Setup Friction** | Minimal (Zero extra installs) | Low to Moderate (`uv sync`) | Minimal (Single executable) |
| **Token Tracking Fidelity** | Moderate (Custom API parsing) | Highest (Native provider SDK usage payloads) | High (Direct API client integration) |
| **Workspace Isolation** | High (`testscript` temp sandboxes) | High (Temp directories and git worktrees) | High (Managed `.scratch/` sandboxes) |
| **Local Model Driver** | Complex (Manual HTTP harness) | Simple (`llama.cpp` / Ollama integration) | Moderate (HTTP client calling `/v1`) |
| **Reporting & Graphs** | Basic (Markdown template output) | Advanced (Rich tables, chart generation) | Moderate (Markdown template output) |
| **CI Integration** | Native (`go test ./...`) | Separate workflow step | Standalone CLI step |

---

## 5. Key Open Questions & Methodology Invariants

### A. Token Accounting Fidelity

Evaluating token savings requires high-fidelity accounting across three potential sources:

1. **Local Tokenizers (`tiktoken`, Hugging Face)**:
   * *Properties*: Fast, offline, deterministic.
   * *Limitations*: Local tokenizers omit provider-specific chat template overhead, system prompt envelopes, and specialized tool-calling tokens. This divergence creates systematic error deltas between local estimates and actual billing.
2. **Provider API Usage Payloads**:
   * *Properties*: Represents the exact billing ground truth reported by model providers (`usage.prompt_tokens`, `usage.completion_tokens`, `usage.prompt_tokens_details.cached_tokens`).
   * *Limitations*: Requires live API connections. Obscures token boundaries within individual message components unless calculated separately.
3. **Agent Transcript Step Analysis**:
   * *Properties*: Parses session logs directly (such as `<appDataDir>/brain/<id>/.system_generated/logs/transcript.jsonl` in Antigravity or session JSON in Claude Code). Captures full system prompts, middleware scaffolding, and actual agent state accumulation.
   * *Limitations*: Tightly binds extraction logic to harness-specific log formats.
4. **Resolution Strategy**: Use provider API usage headers as the primary truth source for headless API trials. Supplement with transcript log extractors for harness-integrated benchmarks. Retain local tokenizers strictly for offline sanity checks.

### B. Testcase Fixture Design

Evaluating realistic agent behavior requires balancing hermetic determinism against real-world structural complexity, while strictly avoiding **information leakage**:

1. **Enhanced Synthetic `txtar` Fixtures**:
   * *Properties*: Single-file archives stored in a dedicated `testdata/bench/` suite. Crucially, these must include **YAML frontmatter** containing the natural language prompt and oracle expectations, while strictly excluding `want/` golden files from the model's environment.
   * *Advantages*: Small footprint, fast initialization (<100ms), fully self-contained.
   * *Limitations*: May underestimate search latency in large repos.
2. **Real-World Repository Snapshots**:
   * *Properties*: Pinned Git commits of established mid-sized Go repositories (`chi`, `gin`).
   * *Limitations*: Large disk footprints; slow checkout and compile cycles.
3. **Resolution Strategy**: Adopt a two-tiered suite (Tier 1 Core `txtar`, Tier 2 Ecosystem). **Crucial Invariant**: The environment must be strictly hermetic. Remove any network dependencies (e.g., fetching `google/uuid`) in favor of local mock packages.

### C. Correctness Oracle Architecture

The oracle must treat the model-controlled workspace as **hostile**. Evaluating correctness requires a multi-stage validation chain:

1. **Precondition Validation**: Confirm target exists and initial fixture doesn't already satisfy the task.
2. **Mutation-Policy Validation**: Reject unauthorized changes (e.g., modifying `go.mod`, tests, or build scripts), deletions of packages, or fake dummy declarations.
3. **Type-Resolved Structural Validation (AST Oracle)**:
   * Verify the required postcondition by resolving object identities (not just identifier spelling).
   * Ensure the exported API shape, signatures, and receiver bindings are preserved.
   * *Note*: The simple "old identifier occurs zero times" check is insufficient and prone to false negatives.
4. **Isolated Compilation and Behavioral Verification**:
   * Copy *only* accepted candidate source files into a fresh, isolated verifier workspace.
   * Inject hidden tests only *after* the model loses access.
   * Run tests with network disabled and trusted `go` binaries.
5. **Diff Audit**: Record normalized textual and AST diffs for analysis.

**Adversarial Oracle Tests**: Every oracle implementation must be tested against cheating candidates (e.g., no-ops, commenting out assertions, leaving stale call sites).

### D. Harness Non-Determinism Management

LLM generation exhibits variance across repeated invocations, even when specifying `temperature = 0`, due to GPU floating-point non-associativity, concurrency variations, and provider Mixture-of-Experts routing:

1. **Seed Anchoring**: Specify consistent random seed parameters where supported by the provider API.
2. **Sample Size**: Execute a minimum of $N = 5$ trials per matrix cell to calculate meaningful statistical distributions.
3. **Statistical Metrics**: Report median and interquartile range (IQR) alongside mean and standard deviation for token counts and latency.
4. **Reliability Metrics**: Compute First-Pass Success Rate ($Pass@1$) and Pass-at-$k$ ($Pass@k$) to evaluate how effectively each toolset mitigates cascading failures.

---

## 6. Phased Implementation Roadmap

Developing `semedit-bench` proceeds through four structured phases:

### Phase 1: Test Case Fixture Schema & Initial 10-Task Benchmark Suite

* Formalize the task definition schema (`task.yaml` / frontmatter within `txtar` fixtures).
* Curate 10 canonical Go refactoring fixtures in `testdata/bench/`:
  1. Local function rename within a single package.
  2. Cross-package function rename across multiple consuming packages.
  3. Struct method rename across interface implementations.
  4. Top-level declaration insertion with public-precedes-private section placement.
  5. Method insertion with receiver targeting.
  6. Automated import organization and unused import removal.
  7. Missing package import insertion with symbol qualification.
  8. Diagnostic delta error recovery across interdependent packages.
  9. Multi-step composite refactoring (rename, declaration insertion, and import cleanup).
  10. Intentional variable unification and collision disambiguation.
* Build automated AST correctness validators for each task.

### Phase 2: Headless A/B Evaluation Runner

* Construct the test execution harness starting with **Approach B** (Python/LiteLLM) to validate extraction metrics, followed by a migration to **Approach C** (Cobra CLI) for the final developer tooling.
* Support evaluation arms:
  * **Arm A (Baseline)**: Standard agent toolset (`grep_search`, `view_file`, `replace_file_content`).
  * **Arm B (Semedit)**: Baseline text tools **plus** the semantic intent toolset (`semantic_rename`, `semantic_verify`, `semantic_insert_declaration`, `semantic_organize_imports`).
* Provide strict execution isolation via ephemeral scratch directories (`.scratch/benchmarks/run_<id>/`).
* Support dual execution backends: local `llama.cpp` server for cost-free iteration and direct provider APIs (Anthropic, OpenAI) for frontier validation.

### Phase 3: Telemetry Extraction Engine

* Instrument turn execution to capture:
  * Prompt tokens, completion tokens, and reasoning tokens.
  * Turn counts and tool call distributions.
  * Wall-clock duration per turn and per task.
  * Tool execution failures and error recovery round trips.
  * Compiler and test suite exit codes.
* Serialize execution metrics into standardized JSONL format (`eval_result.jsonl`).

### Phase 4: Automated Markdown Report & Visualization Generator

* Implement post-processing pipeline parsing `eval_result.jsonl`.
* Generate comparative markdown summary tables in `docs/research/benchmarks/`.
* Compute aggregate efficiency ratios:
  $$\text{Token Reduction Factor} = \frac{\text{Tokens}_{\text{Arm A}}}{\text{Tokens}_{\text{Arm B}}}, \quad \text{Speedup Factor} = \frac{\text{Latency}_{\text{Arm A}}}{\text{Latency}_{\text{Arm B}}}$$
* Render ASCII / Mermaid trade-off diagrams comparing frontier cloud models and local open-weights models across reasoning tiers.

---

## 7. Next Steps

1. Implement the task schema and initial `txtar` benchmarks in `testdata/bench/`.
2. Construct the prototype evaluation runner based on Approach B for fast prototyping, with an eventual migration path to Approach C for in-tree developer execution.
3. Execute initial baseline evaluations on Qwen 2.5 Coder 14B/32B and Claude 3.7 Sonnet.
4. Publish the first empirical benchmark report in `docs/research/benchmarks/`.
