<!-- Records the controlled prompt-arm experiment for diagnosing semedit planning and execution costs. -->

# RQ-0065: Explicit Semedit Prompt Arm

Status: Open
Date: 2026-10-05

## Question

How much of semedit's observed latency, tool-call count, and token use comes from selecting and planning semantic operations versus executing the MCP tools themselves?

## Experiment

Add an explicit semedit prompt arm for every benchmark fixture. Each explicit prompt names the semantic tool calls, order, and arguments needed for that fixture. Independent read-only lookups may run in parallel; mutations run serially or in one ordered `semantic_batch`. Compare three conditions under the same target, fixture, context, repeat, MCP server-instruction mode, and arm restriction:

1. **baseline**: standard file-editing prompt, one shared baseline run;
2. **semedit-implicit**: existing task prompt with semedit steering;
3. **semedit-explicit**: explicit MCP action plan with the same semedit steering.

Keep the existing report columns, measurements, and deltas. Pair the implicit and explicit rows against the same baseline result. Do not expect explicit prompts to reduce token usage automatically: their longer instructions may increase input tokens while reducing model-side planning turns or tool selection errors.

## Controls and Analysis

Hold target model, fixture oracle, context, repeat count, MCP server instructions, policy, timeout, and concurrency constant. Use the same baseline within each repeat and rotate arm execution order when collecting repeated measurements. Analyze wall time, top-level turns, internal tool cycles, tool invocations, tool errors, total tokens, and input/output token components together. Where telemetry permits, separate provider/model wait from MCP execution time to distinguish planning overhead from tool latency.

The explicit instructions should be a one-to-one task plan, not a broader strategy change: retain task requirements and safeguards, name exact tools/arguments, parallelize only independent reads, and serialize mutations. Review whether outcomes and oracle validity remain comparable before interpreting efficiency.

External fixtures use private `.benchmark/explicit.md` with complete ordered
tool payloads. Execute each authored recipe through semedit from its pinned base
and pass the private oracle before treating it as a successful explicit plan.
Missing recipes fail planning. Human fix replay can diagnose a fixture but its
timing cannot establish semedit performance; legacy replay results are excluded
from comparisons and publication. Record recipe and fixture digests alongside
validation evidence. The explicit arm still runs through the coding agent, so
report provider overhead separately from tool execution where telemetry permits.
A validated efficient sequence establishes a practical execution baseline,
without proving that no faster sequence exists.

### Implementation status, 2026-10-08

External explicit arms now use ordinary agent execution with exact private
prompts, shared baseline pairing, equal policy/skill treatment, and rotated arm
order. Missing recipes stop planning; legacy Git replay records are excluded.
All five external fixtures now have self-contained one-call recipes: grpc 2735
uses 32 child operations, grpc 2951 uses 59, grpc 3361 uses 7, go-zero 2463 uses
29, and go-zero 964 uses 17. Each fresh pinned-source replay and ordinary oracle
passed. [Saved prompts and receipts](../../data/benchmarks/fixture-validation/)
record exact payloads and hashes. The two invalid acceptance fixtures received
documented private test corrections described in [RQ-0066](RQ-0066-luna6-multiswe-go-smoke-oracle-integrity.md).
No new measured model run was performed.

All five external fixtures now declare three progressive corrective hints
authored from their human fixes. Vanilla and implicit share the task content;
explicit receives no such ladder because its initial prompt already supplies
the complete recipe. Record first-attempt and eventual success separately using
interaction-step evidence, including the cost of hints. These revised fixtures
define a new run plan; earlier unhinted smoke measurements remain historical.

### Single-iteration hinted smoke, 2026-10-07

The [hinted smoke](../../data/benchmarks/results/luna6-multiswe-go-hints-smoke-20261007-r1/smoke-summary.md)
ran go-zero 2463 once per arm at Luna 6 medium and high. Both explicit cells
passed on their initial task turn (184.349 s medium, 238.085 s high). Both
vanilla cells exhausted the three hints and failed acceptance on the context
argument required by `logc.Alert`. Both implicit cells timed out during the final
hint turn at the twelve-minute job limit; their incomplete final candidates were
not evaluated. No acceptance-test installation error occurred in this run.

The other twenty-four requested cells were blocked before execution by the four
missing explicit recipes. An initial outer-sandbox provider startup failure is
archived separately and excluded from task outcomes. Six checkpoints, transcripts,
timing evidence, and unchanged input hashes are preserved. This single iteration
does not meet the repeated-run resolution criteria below.

## Resolution Criteria

Resolve after repeated runs cover the existing benchmark fixtures and report the three arms with paired deltas. Summarize which measured costs change from implicit to explicit, whether the explicit plan alters correctness or tool success, and what remains unidentifiable from current timing telemetry.
