<!-- Defines which benchmark observations the harness derives mechanically and where agent interpretation remains necessary. -->

# ADR-0077: Deterministic Benchmark Evidence Analysis

## Status

Accepted

## Date

2026-10-19

## Goals

The analyzer's primary goal is to turn selected benchmark and transcript
exports into concise, reproducible observations with direct evidence links.
Its broader goal is to support a closed improvement cycle: measure agent work,
find repeated friction, make a focused tooling change, rerun the same conditions,
and retain the change only when correctness and the intended outcomes improve.

## Context

Benchmark exports and Codex transcripts preserve tool calls, outcomes, turns,
reflections, measured results, and experiment conditions. An operator must
otherwise inspect these records manually to find repeated failures, parameter
rejections, follow-up actions, or differences between matched runs. The broader
goal is a repeatable improvement loop: collect comparable evidence, identify a
candidate tooling change, implement it, rerun the same benchmark conditions,
and retain or reject the change based on correctness and measured outcomes.

The analysis tool must reduce the evidence-reading burden without turning
correlation or heuristic text matches into claims about causality, tool choice,
or language-server capability. Transcript exports may be paginated and may
omit outcomes; missing observations must remain visibly incomplete.

## Decision

Add a deterministic `benchmark-harness observe` command that consumes selected
JSONL records produced by `benchmark-harness analyze`. It emits a compact JSON
report containing:

* exact-condition cohort summaries for run count, success, failed tool calls,
  wall-clock p50/p95/p99, token totals, and harness-recorded cost;
* when provider transcript timestamps are available, per-call duration and
  run-level summed tool wait split into semantic and other calls, unioned
  tool-occupied wall time, residual non-tool wall time, and timing coverage;
* per-tool call counts split into transport failures, functional failures, and
  unknown outcomes;
* tool failures separated by transport versus functional status, with narrow
  text signals for missing required parameters, unsupported or unavailable
  behavior, timeouts, and verification or syntax failures;
* exact tool-and-argument repeats and the next observed tool after a failure;
* the same call, failure, repeat, and next-action observations when Codex
  transcript records provide the relevant call IDs and order; and
* keyword-based themes in benchmark reflections, linked to the source record.

Every observation carries source and run or transcript references. The command
accepts selected JSONL pages as separate inputs and reports whether an input's
last page indicates more data. It does not scan the archived benchmark-result
tree or Codex state; selection and transcript export remain the responsibility
of `analyze`.

## Invariants

* Observations are derived only from explicit inputs and preserve links to the
  evidence that produced them.
* Cohorts retain task, arm, harness, model, reasoning effort, variant, skill,
  batching, and instruction conditions. Unmatched conditions are not combined.
* Transport failures and server/tool functional rejections remain distinct.
  Unknown outcomes and pagination gaps are reported, never treated as success
  or proof that no event occurred.
* Timing is derived only from paired provider item start/completion timestamps
  retained in transcript sidecars. `tool_wait_ms` sums overlapping calls,
  while `semantic_tool_wait_ms` and `other_tool_wait_ms` split that sum by the
  tool's semantic identity, and `tool_occupied_ms` unions all call intervals.
  Per-tool rows preserve the operation name and duration percentiles for
  matched arm comparisons.
  `non_tool_wall_clock_ms` subtracts occupied tool time from the harness run
  wall clock and includes model generation plus provider, harness, and startup
  overhead; it is not a direct measurement of model inference. Cohort timing
  percentiles use only runs with complete tool-call timing and report their
  coverage separately.
* Heuristic labels such as `unsupported_or_unavailable_signal` and reflection
  themes are search aids, not authoritative diagnoses. Tool mis-selection,
  misunderstood capability, causal attribution, and whether a response caused
  a tactic change require review of the linked trace.
* A subsequent different tool is an observed sequence fact, not proof that the
  agent understood or followed the error. A missing subsequent call may also
  result from incomplete input.
* The analyzer does not mutate benchmark results, prompts, tools, or source
  code, and does not modify task measurements.
* The improvement loop retains a fixed benchmark method and conditions when
  evaluating a tooling change. Candidate changes are accepted only when the
  intended correctness or usability outcome improves without an unacceptable
  correctness, latency, token, or cost regression. Research questions and ADRs
  record unresolved alternatives and accepted architectural changes.

## Consequences

Operators can filter and summarize selected benchmark and interactive Codex
evidence before asking an agent to interpret it. Static observations are
repeatable and cheap, while semantic judgments remain explicit human or agent
work with citations to trace records. The separate JSON report can be retained
alongside the paginated evidence export and compared across fixed benchmark
conditions.

The analyzer does not yet prove capability gaps against each backend's native
LSP, AST engine, or IDE refactorings. Such conclusions require a capability
catalog and external implementation research; the report only surfaces
signals and evidence for that next step. See the [benchmark harness guide](../../tools/benchmark-harness/README.md)
for command ownership and the end-to-end improvement workflow.
