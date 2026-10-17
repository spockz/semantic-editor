<!-- Defines durable measurement boundaries so post-task interrogation cannot change benchmark work results. -->

# ADR-0076: Work and Diagnostic Measurement Boundaries

## Status

Accepted

## Date

2026-10-17

## Context

Diagnostic questions continue the original coding session after task completion.
Work totals already exclude these questions, but normalized diagnostic counters
were discarded. Late MCP records were classified by wall-clock timestamps, and
a diagnostic process exit could invalidate an already accepted work result.

## Decision

Put task execution, corrective and staged follow-ups, work aggregation, clock
finalization, and diagnostic scheduling in the provider-independent
`agentSession.execute` lifecycle in `session_lifecycle.go`. The job driver invokes
that lifecycle once. Every adapter is wrapped by `agentSession.runTurn`; adapters
provide protocol observations and process timing without deciding measurement
phases or follow-up policy.

Bind workspace, target (including model and effort), and arm at the first turn.
The wrapper supplies canonical result identity and rejects later session changes
before provider dispatch. Turn callers supply no target or arm. Use typed task
and diagnostic kinds, reject invalid kinds, prohibit task work after diagnostics,
and reject continuation destinations that alias the work aggregate. These
guarantees belong to the wrapper and do not depend on adapter implementations.

Keep benchmark measurements limited to initial work, corrective or staged task
turns, and their acceptance checks. Freeze the task clock before interrogation.
Retain each turn's independent measurements in its classified session transcript,
including diagnostic tokens, tools, elapsed time, latency, and availability
provenance. Transcript duration fields use explicit nanosecond suffixes.

Snapshot completed task MCP records before the first diagnostic turn. Persist
that immutable snapshot after provider shutdown. Without diagnostics, capture
the records at shutdown. Do not classify late records by cross-process clocks.

Record process exits attributable to failed diagnostic turns with that turn.
Preserve genuine cleanup and work-evidence failures as execution errors. Keep
diagnostics on the original workspace, model, arm, and continuation identity.

## Invariants

* Diagnostic counters and time never enter work totals or publication aggregates.
* All adapters use the same lifecycle and turn wrapper; provider code cannot
  select diagnostic questions or merge their measurements into work totals.
* Every arm retains its original identity through corrective, staged, and
  diagnostic turns. Missing result metadata cannot silently select a default arm.
* Invalid phase transitions and aggregate reuse fail before provider dispatch.
* Transcript measurements are snapshots; later cumulative usage, provenance,
  and pointer updates cannot mutate an earlier turn's evidence.
* Missing task MCP records remain absent even if diagnostics later create them.
* Diagnostic clock changes, malformed late records, and failed diagnostic
  subprocesses cannot change accepted task measurements or oracle verdicts.
* Invalid work records and genuine cleanup errors remain explicit failures.

## Consequences

Diagnostic costs remain auditable in the transcript without adding diagnostic
metrics to benchmark result objects. Older transcripts may omit the optional
metrics field. This refines the session boundary in [ADR-0050](0050-benchmark-planning-sessions-and-arm-policy.md).
