# Architecture Decision Record (ADR) 0044: Benchmark Experiment Dimensions and Result Semantics

Status: Proposed
Date: 2026-09-23

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

- A run retains all measurements and supporting observations currently
  published in per-run and aggregate tables, including raw values needed to
  reproduce derived metrics; new report fields do not silently replace them.
- A run records its test identity, all resolved experimental conditions, and
  relevant execution metadata.
- Only explicitly selected conditions vary between runs; known dimensions do
  not imply a Cartesian product. Repetitions of one resolved configuration are
  separate observations and do not require a hypothesis.
- Comparison grouping and best-case selection retain ADR-0038 and ADR-0043
  behavior, including task, target, prompt variant, MCP instruction mode,
  context variant, and arm distinctions. Technical provenance does not split
  aggregates.
- Task correctness, expected-strategy evaluation, tool outcomes, and efficiency
  remain separate result fields; none substitutes for another.
- Capability evaluation uses the labels captured with the run, so later tool
  catalog changes cannot reinterpret an old trace.
- Chain matching is deterministic and does not constrain task success unless an
  experiment explicitly makes it part of its success criteria.

## Consequences

Experiments can vary a small, explicit set of conditions while retaining the
current comparison and reporting rules. Descriptive domain labels avoid
introducing taxonomy governance. Implementers can introduce a versioned schema
and migration incrementally, without treating this ADR as a commitment to every
proposed field.
