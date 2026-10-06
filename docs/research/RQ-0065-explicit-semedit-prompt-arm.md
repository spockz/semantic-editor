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

## Resolution Criteria

Resolve after repeated runs cover the existing benchmark fixtures and report the three arms with paired deltas. Summarize which measured costs change from implicit to explicit, whether the explicit plan alters correctness or tool success, and what remains unidentifiable from current timing telemetry.
