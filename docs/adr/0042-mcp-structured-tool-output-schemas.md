# ADR-0042: MCP Structured Tool Output Schemas

Status: Accepted
Date: 2026-09-22

## Context

MCP tools exposed input schemas but did not advertise output schemas, and
successful tool calls exposed timing data without placing their formatted
result in `structuredContent`. This left clients unable to discover or safely
consume machine-readable success results. The MCP structured-output facility
is defined by the `2025-06-18` protocol revision, while the server previously
advertised `2024-11-05`.

The central operation registry is the source of direct MCP tools. Batch and
live-reload tools remain transport-level tools with distinct result shapes:
batch execution returns ordered per-edit results and, on success, one final
diagnostic delta; live reload must reply before re-executing the server.

## Decision

Advertise MCP protocol revision `2025-06-18` and provide an `outputSchema` for
every advertised tool. Registry-derived direct tools use one common successful
envelope with a required `result` string and `metrics`, plus optional
`session_metrics`. Their successful `structuredContent` places the existing
formatted text in `result` while retaining the established timing shapes and
human-readable `content` block.

`semantic_batch` uses the same envelope, but its `result` is the concrete
`BatchResponse` shape: required `status` and ordered `results`, with an
optional `diagnostic_delta` describing the captured final state. Successful
batch calls place that response object in `structuredContent.result`; failed
batches retain error semantics and do not claim a final diagnostic delta.

The opt-in `semantic_reload` tool advertises a dedicated output schema and
returns `{ "result": { "status": "reloading" } }` before re-execution. Error
responses remain outside successful output-schema conformance and continue to
retain their existing `content` and `isError` behavior.

## Invariants

- Every tool returned by `tools/list`, including injected-registry tools,
  `semantic_batch`, and opt-in `semantic_reload`, has an `outputSchema`.
- Every successful direct or batch call includes the schema-required
  `structuredContent.result` and preserves `metrics`; direct formatted text
  remains available in `content`.
- Batch `diagnostic_delta` is advertised and returned only when execution
  reaches the final diagnostic capture.
- Output schemas do not over-constrain arbitrary telemetry phase names,
  diagnostic strings, or future envelope fields.
- Reload sends its success response and structured result before invoking the
  platform-specific re-execution path.

## Consequences

MCP clients can discover and validate successful result envelopes without
parsing human-readable text, while existing clients continue to receive the
same `content` wording and timing metrics. The protocol response now identifies
the revision that supports structured tool output. Operation-specific result
projections remain a future deliberate change; the current envelope preserves
compatibility with all registry formatters.
