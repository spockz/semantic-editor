# ADR-0039: Benchmark Result Validity and Tool Telemetry

Status: Accepted
Date: 2026-09-21

## Context

Agent harness transcripts use different event shapes. Codex emits
`mcp_tool_call`, while other targets can emit `tool_call`, function-call, or
command events. A list of tool names alone cannot distinguish discovery from a
completed transport request or a tool action that ran and then failed. Partial
harness failures were also reaching generated documentation as if they were
valid experimental observations.

## Decision

Normalize every observed tool invocation into an ordered record containing its
tool name, server identity, transport outcome, functional outcome, failure
detail, and available server metrics. The parser recognizes Codex
`mcp_tool_call` events as tool calls alongside the older event forms.

Transport success means the harness reached the tool endpoint. Functional
success means the tool completed the requested action. Unknown remains an
explicit state when a transcript lacks enough evidence. MCP latency metrics
include total server time and phase totals so validation and formatting cost
can be separated from the edit path.

Codex runs record process-start-to-first-event and first-event-to-first-tool-call latency while
consuming the event stream. The semedit server attaches server-start-to-initialize and
initialize-to-first-semantic-call latency to its first semantic response. These boundaries
separate agent/session startup from semantic-tool execution.

Benchmark report format version 2 serializes every harness and oracle duration field as an
integral number of milliseconds, matching its `*_ms` field name. It declares
`duration_unit: "milliseconds"`; documentation loading recognizes unversioned legacy reports as
nanosecond-encoded Go durations so historical observations remain comparable.

Generated documentation publishes a comparison only when it has at least one
run with an oracle result and no harness execution error. A completed agent
attempt that fails its oracle remains publishable data; a run that never
reaches oracle evaluation does not.

For each project-size and prompt variant, documentation renders the ordered
baseline and semedit tool-call sequences side by side. The report does not
present a task-level tool list as evidence for every underlying run.

## Invariants

- Tool-call order is preserved from the target transcript.
- Transport and functional failure are represented separately and neither is
  silently converted into success.
- Semantic-tool availability claims identify the canonical server or semantic
  tool identity rather than a display label alone.
- Documentation excludes incomplete harness records before aggregation.
- Published tool-call tables are qualified by their run variant and arm.
- Missing structured MCP response content remains an unavailable observation, not a measured
  zero-duration startup boundary.
- Versioned report duration values and their `*_ms` JSON field names use the same milliseconds
  unit; legacy unversioned reports retain their historical nanosecond interpretation.

## Consequences

Reports can show whether semedit was discovered, contacted, rejected, or
completed work, and can attribute server-side latency to its phases. Publication
is less likely to overstate an interrupted benchmark as evidence while
retaining unsuccessful completed attempts as meaningful results.
