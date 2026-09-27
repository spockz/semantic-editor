# ADR-0052: Opt-In MCP Timing Metrics

Status: Accepted
Date: 2026-09-26

## Context

MCP responses included request phase timings and server startup timings on every semantic call. Repeated telemetry enlarged ordinary results and competed with task content for model attention. Users still need those measurements for diagnosis and benchmark analysis.

## Decision

Direct semantic and batch responses return their structured result without timing fields. The server retains timing records for the latest 100 timed calls in memory and exposes them through a separate `semantic_metrics` tool. The tool returns the most recent records first, defaults to 10 records, and accepts a limit from 1 through 100. Each record identifies its sequence, tool name, completion time, request metrics, and optional startup metrics. The metrics query is not itself recorded.

## Invariants

- Standard semantic and batch output schemas require only `structuredContent.result`; they do not advertise request or startup timing fields.
- Timed successes and errors enter the in-memory history with their tool name. The history remains bounded to 100 records and is safe for concurrent access.
- `semantic_metrics` is available in full and mutations-only profiles and does not add its own timing record.
- Startup timing stays attached to the first timed semantic call and can be read from that call's history record.
- Timing history is scoped to one running MCP server process and is discarded when that process exits.

## Consequences

Routine tool results stay smaller, while clients can request timing data when it is useful. Clients that need timings must call `semantic_metrics` after the measured operation. This makes telemetry collection explicit and avoids imposing timing payloads on every model-visible result.
