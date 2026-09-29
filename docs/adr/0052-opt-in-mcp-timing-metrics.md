# ADR-0052: Opt-In MCP Timing Metrics

Status: Accepted
Date: 2026-09-26

## Context

MCP responses included request phase timings and server startup timings on every semantic call. Repeated telemetry enlarged ordinary results and competed with task content for model attention. Users still need those measurements for diagnosis and benchmark analysis.

## Decision

Direct semantic and batch responses return their structured result without timing fields. The server retains timing records for the latest 100 timed calls in memory and exposes them through a separate `semantic_metrics` tool. The tool returns the most recent records first, defaults to 10 records, and accepts a limit from 1 through 100. Each record identifies its sequence, tool name, completion time, request metrics, and optional startup metrics. The metrics query is not itself recorded.

An optional `--metrics-file` path persists every timed record as a JSON array using atomic writes. A new server process loads an existing file and continues its sequence, allowing benchmark follow-up turns to share one file. Benchmark semedit arms enable this path inside their disposable fixture workspace; the harness retains task-only records beside the session transcript at the end of the run and records the durable path in provenance.

## Invariants

- Standard semantic and batch output schemas require only `structuredContent.result`; they do not advertise request or startup timing fields.
- Timed successes and errors enter the in-memory history with their tool name. The history remains bounded to 100 records and is safe for concurrent access.
- `semantic_metrics` is available in full and mutations-only profiles and does not add its own timing record.
- Startup timing stays attached to the first timed semantic call and can be read from that call's history record.
- The in-memory history is scoped to one running MCP server process and is discarded when that process exits. File persistence is opt-in, preserves all records, and reports read, decode, or write failures as server errors.
- Benchmark evidence excludes diagnostic reflection calls from the retained task metrics file.

## Consequences

Routine tool results stay smaller, while clients can request timing data when it is useful. Benchmark runs can recover server timings after the MCP process exits. Atomic persistence adds disk work to each timed tool response in those runs; the request's recorded `total_ms` is captured before persistence, while the task wall clock includes it.
