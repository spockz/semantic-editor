# ADR-0044: Benchmark Experiment Dimensions and Result Semantics

* **Status**: Accepted
* **Date**: 2026-10-06

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Codify the 7-axis task complexity framework on a 5-dot scale, pre-wall-clock CoW monorepo hydration, and deterministic strategy evaluation. |
| description | The harness adds benchmark choices as named variants, which can hide what a run actually changes and encourage broad matrix expansion. The harness already has meaningful task and prompt variants, evaluation arms, Model Context Protocol (MCP) instruction modes, and technical provenance. ADR-0038 defines which conditions distinguish comparisons, and ADR-0043 defines aggregation and publication behavior. A more explicit experiment model must preserve those established distinctions. |
| role | process |
| scope | tooling |
| area | benchmark |
| languages | generic |

## Context

The harness adds benchmark choices as named variants, which can hide what a run
actually changes and encourage broad matrix expansion. The harness already has
meaningful task and prompt variants, evaluation arms, Model Context Protocol
(MCP) instruction modes, and technical provenance. ADR-0038 defines which conditions distinguish
comparisons, and ADR-0043 defines aggregation and publication behavior. A more
explicit experiment model must preserve those established distinctions.

## Decision

Represent each benchmark run by its test, resolved configuration,
conditions varied for that run, execution metadata, and observed result. Vary
only conditions explicitly selected by the benchmark invocation. Named
variants may remain as convenient aliases, but must resolve to recorded
conditions before execution.

A test has a stable identifier (ID) and a short, descriptive domain label.
Keep domain labels lightweight: the project does not need a central registry. Record language, fixture
size, and prompt variant as separate attributes or conditions; they are not
synonyms for domain.

An experiment may group selected tests, a baseline configuration, controls,
and conditions varied. Its hypothesis is optional descriptive metadata. A
benchmark invocation may repeat the same resolved configuration without an
experiment hypothesis; each repetition is a separate observation identified
by its repetition number. Record the fully resolved conditions on every run.
Keep technical provenance auditable without using it to create comparison
groups.

### The 7-Axis Task Complexity Framework

Rather than using an opaque scalar difficulty score, each benchmark fixture has an explicit 7-axis complexity profile scored on a 5-dot scale (`●○○○○` to `●●●●●`). Each axis captures specific strengths and failure modes of LLM coding agents:

| Dimension | Why It Matters / Agent Dynamics | Vanilla Agent Profile | Semedit MCP Profile |
| --- | --- | --- | --- |
| **1. Locality & Fan-Out** ($D_{\text{loc}}$) | **Multi-file degradation cliff**: LLMs excel at 1-file edits, but success drops below 20% once edits span $>3$ files due to omitted peripheral call sites. | **Strong at `●○○○○`** (1-turn local patch); **Collapses at `●●●●●`** (misses call sites in downstream packages). | **Dominates at `●●●●●`**: Compiler traces symbol usages across packages and updates all files deterministically in $O(1)$ tool output tokens. |
| **2. Context Burden** ($D_{\text{ctx}}$) | **Working memory saturation & 'Lost-in-the-Middle'**: Loading thousands of lines of context causes models to lose track of early instructions and file structures. | **Strong at `●○○○○`** (<50 lines fits in prompt); **Poor at `●●●●●`** (wastes thousands of tokens grepping and reading unrelated code). | **Dominates at `●●●●●`**: Operates via compiler-indexed symbols; zero prompt tokens spent reading unmodified files into memory. |
| **3. Symbol Ambiguity** ($D_{\text{amb}}$) | **Token-level pattern matching vs. Type Graph**: LLMs lack an internal compiler. When identical names exist (`r.ID`, `Close()`, `Get()`), regex and textual search cannot distinguish receiver types. | **Strong at `●○○○○`** (unique names like `ParseXMLToAST`); **Fails at `●●●●●`** (false-positive search/replace breaks unrelated structs). | **Dominates at `●●●●●`**: The language server type checker resolves exact struct field bindings and receiver types with 100% precision. |
| **4. Change Composition** ($D_{\text{comp}}$) | **State tracking & error compounding**: RefactorBench shows agents fail multi-hop refactoring (22% vs 87% human) because intermediate steps break state and compound errors ($P \approx p^N$). | **Moderate at `●●○○○`**; struggles with multi-step sequential dependencies. | **Strong at `●●●●●` with `semantic_batch`**: Executes compound mutations atomically in an in-memory session, avoiding intermediate broken states. |
| **5. Invariant Strictness** ($D_{\text{inv}}$) | **Probabilistic generation vs. AST invariants**: LLM text generation is prone to whitespace drift, dropped comments, and damaged compiler directives (`//go:embed`, `//go:build`). | **Fragile at `●●●●●`** (patch reject errors, linter failures from formatting churn). | **Dominates at `●●●●●`**: AST-level transformations isolate target nodes without collateral damage to comments, whitespace, or build tags. |
| **6. Edit Scale** ($D_{\text{scl}}$) | **Output token budgets & generation exhaustion**: Generating diffs for 20+ call sites linearly bloats output tokens and increases truncation risk. | **Collapses at `●●●●●`** (exceeds token limits, truncates diffs, becomes 'lazy' with `// ...existing code...`). | **Dominates at `●●●●●`**: Zero-token output transformation; a ~25-token tool call triggers $N$ refactorings in the host engine. |
| **7. Staging & Verification** ($D_{\text{stg}}$) | **Paranoia tax & diagnostic loops**: When tools lack feedback, agents run defensive verification turns (`git diff`, `go test`), doubling latency and token spend. | **Fast at `●○○○○`** (single-shot diff + test in one command); fragile under complex compilation breaks. | **Strong at `●●●●●`**: Integrated compiler diagnostic deltas terminate invalid trajectories and provide immediate verification signals. |

Retain every measurement and supporting observation already used in benchmark
reports. This includes wall-clock and startup latencies; top-level, internal,
initial-load, MCP-discovery, and corrective interactive turns; tool invocation
counts and ordered calls; initial-context, total/cached/uncached input, output,
and reasoning tokens; oracle stages; and semantic-tool verification. Preserve
per-call transport and functional outcomes, failures, arguments, and MCP server
phase timings. Keep raw observations alongside derived ratios and declared-rate
model costs. Aggregate numeric metrics and oracle outcomes using the dimensions
and arms defined by ADR-0043. Missing telemetry stays distinguishable from
zero, and diagnostic reflections remain outside task measurements as in
ADR-0038.

Expected tool-use strategy is an optional evaluation separate from task
correctness, tool success, and efficiency. A trace preserves ordered tool-call
events and the tool identity and capability labels captured for that run. A
chain may use exact tools, ordered capabilities, or informational expectations.
For ordered chains, match steps left to right against distinct trace events;
one event can satisfy at most one step. For multiple valid alternatives, select
the first matching alternative in its declared order. Final-answer events are
not tool calls. Missing or unknown telemetry remains unknown. Do not count it as a match or
failure.

This ADR establishes semantics, not a normative data schema. Define and
validate a versioned schema alongside implementation.

## Invariants

* A run retains all measurements and supporting observations currently
  published in per-run and aggregate tables, including raw values needed to
  reproduce derived metrics; new report fields do not silently replace them.
* A run records its test identity, all resolved experimental conditions, and
  relevant execution metadata.
* Only explicitly selected conditions vary between runs; known dimensions do
  not imply a Cartesian product. Repetitions of one resolved configuration are
  separate observations and do not require a hypothesis.
* Comparison grouping and best-case selection retain ADR-0038 and ADR-0043
  behavior, including task, target, prompt variant, MCP instruction mode,
  context variant, and arm distinctions. Technical provenance does not split
  aggregates.
* Task correctness, expected-strategy evaluation, tool outcomes, and efficiency
  remain separate result fields; none substitutes for another.
* Capability evaluation uses the labels captured with the run, so later tool
  catalog changes cannot reinterpret an old trace.
* Chain matching is deterministic and does not constrain task success unless an
  experiment explicitly makes it part of its success criteria.
* Large monorepo fixture workspaces (e.g. Multi-SWE-bench Go) are hydrated via
  local Copy-on-Write cloning (`cp -Rc` / reflink) from pre-cached shallow clones
  in `.scratch/cache/repos/` strictly outside measured agent wall-clock time.

## Consequences

Experiments can vary a small, explicit set of conditions while retaining the
current comparison and reporting rules. Descriptive domain labels avoid
introducing taxonomy governance. Implementers can introduce a versioned schema
and migration incrementally, without treating this ADR as a commitment to every
proposed field.
