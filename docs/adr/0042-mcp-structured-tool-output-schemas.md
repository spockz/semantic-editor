<!-- Explains the structured-output contract and client presentation boundary for MCP tools. -->

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

This decision records the original envelope. ADR-0052 later moved ordinary
timing metrics out of direct results, and registered operations now use typed
`ProjectOutput` projections where declared (including inspection). The legacy
formatted-text result applies when an operation has no typed projection.

Structured output is a server-produced representation, not a guarantee that a
client or model will expose every field in its prompt. Follow the MCP
structured-content compatibility guidance, which says servers SHOULD include
serialized JSON in
`TextContent` when returning structured data
([MCP Tools: Structured Content](https://modelcontextprotocol.io/specification/2025-06-18/server/tools#structured-content)).
For large projections, the representations need not be duplicated in model
context; clients may choose which representation to forward. A compact text
projection of essential fields is not the full serialized structured JSON
recommended by MCP. When used instead of that full serialization, it is a
deliberate bounded compatibility tradeoff. The server's concise text must
preserve the essential
outcome, failure/limitations, ambiguity, and live continuation choices before
large source bodies or diffs. A shared validated projection must supply those
essential fields to both representations, exclude stale or deleted targets
from live continuations, and retain any historical evidence with an explicit
label.

The shared essential text prefix starts with `Critical result (JSON):` followed by
compact JSON and a blank line before the existing formatted body. It preserves
the exact essential values and record order from the validated structured
projection. Outcomes and continuations precede bulk source, diffs, and logs;
the prefix excludes those bulk payloads rather than repeating them. Numeric
values must survive serialization without conversion through floating point.
Presentation budgets remain open, and client truncation can still hide fields.

Batch text first labels child results as intermediate evidence. Its critical
projection omits child continuation records because later edits and publication
can invalidate them. Literal artifacts, such as a feedback draft, retain their
requested text when its leading summary already exposes the complete outcome
and limitations; they do not need a duplicate JSON wrapper.

Known error outcomes use an opt-in operation-owned error projector paired with
a private result schema. Registration and runtime validation reject live
`continuations` and `expected_revision` fields in that error projection. A
typed error can report known applied or partial changes without satisfying the
successful result schema. An unmatched error reports unknown application
state. A failed error projector retains the original operation error and
reports the projection failure; it does not replace the operation failure.
This leaves registry dispatch error semantics unchanged.

Batch child result schemas accept the separately validated successful and known
error projections. Their union uses `anyOf` because those shapes can overlap;
the child's status and original error still distinguish the outcome.

## Invariants

- Every tool returned by `tools/list`, including injected-registry tools,
  `semantic_batch`, and opt-in `semantic_reload`, has an `outputSchema`.
- Every successful direct or batch call includes the schema-required
  `structuredContent.result`; `content` retains human-readable output.
- Full serialized structured JSON in text follows MCP's SHOULD recommendation.
  A compact essential projection is a bounded compatibility tradeoff, not
  equivalent protocol guidance compliance. Client presentation and model
  visibility remain client-controlled.
- When a result includes large source or diff payloads, concise views preserve
  status, limitations, ambiguity, and live continuations ahead of bulk text;
  the compact prefix uses the shared essential projection. Truncation limits
  remain open.
- Error projections are validated against their paired private schema;
  successful result guarantees remain unchanged. Error state does not produce
  live continuation guards.
- Batch `diagnostic_delta` is advertised and returned only when execution
  reaches the final diagnostic capture.
- Output schemas do not over-constrain arbitrary telemetry phase names,
  diagnostic strings, or future envelope fields.
- Reload sends its success response and structured result before invoking the
  platform-specific re-execution path.

## Consequences

MCP clients can discover and validate successful result envelopes without
parsing human-readable text. The protocol response identifies the revision
that supports structured tool output. Operations without typed projections
retain compatibility with their existing formatters; declared projections
provide operation-specific results through the same envelope.
