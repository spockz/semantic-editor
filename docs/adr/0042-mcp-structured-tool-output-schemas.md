<!-- Explains the structured-output contract and client presentation boundary for MCP tools. -->
# ADR-0042: MCP Structured Tool Output Schemas

* **Status**: Accepted
* **Date**: 2026-09-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Advertise typed output schemas; give every tool actionable compact text while retaining canonical full structured results and exact next-step guards. |
| description | MCP tools exposed input schemas but did not advertise output schemas, and successful tool calls exposed timing data without placing their formatted result in `structuredContent`. This left clients unable to discover or safely consume machine-readable success results. The MCP structured-output facility is defined by the `2025-06-18` protocol revision, while the server previously advertised `2024-11-05`. |
| role | contract |
| scope | tooling |
| area | mcp |
| languages | generic |

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
envelope with the operation's typed `result`. Their successful
`structuredContent` uses the registered projector and schema; `content` gives
a compact actionable view of that result. ADR-0052 defines the
ordinary timing metadata boundary.

`semantic_batch` uses the same envelope, but its `result` is the concrete
`BatchResponse` shape: required `status` and ordered `results`, with an
optional `diagnostic_delta` describing the captured final state. Successful
batch calls place that response object in `structuredContent.result`; failed
batches retain error semantics and do not claim a final diagnostic delta.

The opt-in `semantic_reload` tool returns
`{ "result": { "status": "reloading" } }` before re-execution. Every emitted
structured result, including execution errors, must conform to the advertised
output schema. RQ-0062 found that exempting error responses caused clients to
reject otherwise useful errors.

This decision records the original envelope. ADR-0052 later moved ordinary
timing metrics out of direct results. Every registered operation with an output
contract now uses a typed `ResultSchema` and `ProjectOutput`. Formatted text is
presentation only; no parallel legacy continuation envelope remains. Mutating
results include a finalized `edit_receipt`, and buffered children omit live
continuations until publication.

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
For symbol navigation, ADR-0051 defines a compact projection instead of exposing
the complete native declaration or reference records. Its listing appears once
in the emitted payload; structured content is treated as model-visible. Required
scope, ambiguity, source-view, and continuation fields remain schema-validated.

compact JSON and, optionally, a short plain-language summary. It preserves
exact essential values and record order from the validated structured
projection. Every tool, including transport-level utilities, must expose the
outcome and the information needed for the next action in `content`: exact live
continuation selectors and revisions, ambiguity choices, refusal and failure
reasons, partial or unknown publication state, and diagnostic counts with a
copyable `semantic_diagnostics` query when a report exists. Buffered batch
children are historical evidence, never live continuations. Numeric guards
must survive serialization without floating-point conversion or truncation.

`structuredContent` remains the canonical full result. `content` must not
serialize the whole structured result again. Source bodies, complete reference
and outline inventories, diagnostic arrays and pages, mutation diffs, analysis
read sets, hook logs, metric records, and feedback artifacts stay in structured
fields. Compact text names the relevant fields and page cursors so a text-only
reader knows where to inspect them. Small fields needed for immediate action
may appear in both channels. No absolute text byte cap may truncate a selector,
revision, ambiguity choice, or failure guard. Error text already present in the
critical projection is not appended again.

Rename success and known applied-error results report `active_count` instead of
the full `active` workspace diagnostic array. Their diagnostic delta retains
introduced, resolved, and suggested messages. Use `semantic_verify` for an
explicit current diagnostic inventory. This changes the shared Go, Rust, and
Java rename output schema; the native backend analysis remains unchanged.
Verified MCP outcomes also expose scoped `diagnostic_report` counts and a
copyable `semantic_diagnostics` query. Page the captured report rather than
rerunning verification; see [ADR-0073](0073-captured-diagnostic-reports.md).

Structured content must be treated as potentially model-visible. VS Code's
MCP adapter forwards it instead of textual content; Copilot CLI has reports of
forwarding both. Those observations do not establish what Copilot desktop
1.1.29 forwards. Client choice cannot be inferred from the MCP protocol
revision. See [RQ-0071](../research/RQ-0071-actionable-mcp-output.md) for
source evidence and output-family coverage.

Batch text first labels child results as intermediate evidence. Its critical
projection omits child continuation records because later edits and publication
can invalidate them. Feedback uses its own presentation path but follows the
same rule: text reports draft status and names `result.markdown` for review,
while the complete artifact appears once in structured content.

Known error outcomes use an opt-in operation-owned error projector paired with
a result schema that is included in the advertised output contract.
Registration and runtime validation reject live
`continuations` and `expected_revision` fields in that error projection. A
typed error can report known applied or partial changes without satisfying the
successful result schema. An unmatched error reports unknown application
state. A failed error projector retains the original operation error and
reports the projection failure; it does not replace the operation failure.
This leaves registry dispatch error semantics unchanged.

The shared envelope uses mutually exclusive `oneOf` branches:

* A native result requires the operation's typed `result` and forbids a
  top-level `error`.
* A generic error requires `error` and `application_state` and forbids `result`.
* A known error requires `error`, `application_state`, and a `result` validated
  against the registered error schema.

The outer MCP `isError` flag identifies execution failure; it is not part of
`structuredContent` and cannot select a branch of its JSON Schema. Verification
and batches retain native outcome objects on domain failures. Their payloads
record `failure` and `status` respectively, and tests correlate them with
`isError`. Protocol errors remain JSON-RPC error objects with numeric codes.

Verification and Maven declare native result schemas in the registry. Maven
known errors use the same declared native result shape through its error
projector. Generic transport error handling does not inject provider-specific
result objects. Batch children use status-correlated success, generic-error,
and known-error branches, so a generic branch cannot hide an invalid typed
result. Unknown batch tool names can produce generic failure rows only.

Batch verification metadata uses the native verification schema independently
of the standalone operation's output projector, because batch execution retains
the native verification result. An injected standalone projector may therefore
change its own result shape without changing the batch metadata contract.

Batch diagnostic arrays use empty arrays for zero introduced or resolved
diagnostics. An unavailable delta is represented by omission of the whole
delta, preserving that distinction.

Each server lazily caches its fully resolved tool catalog against its canonical
workspace root and active-language snapshot. The catalog includes the server's
fixed registry, backend capabilities, profile, variants, and reload setting.
Cached maps become immutable after construction: wire serialization reuses them,
while mutable callers receive independent copies. Successful reinitialization
and reload discard cached catalogs together with language snapshots. Catalog
data is never shared across separate server instances or injected registries.

Every operation with an output contract declares a typed result schema and
projector. Continuation aliases may be derived from declared typed result fields;
their roles, consumers, schemas, and optionality remain registry-owned. The MCP
adapter validates the projected result before publication. It has no ad hoc
envelope extraction or separately declared informational metadata mode.

## Invariants

* Every tool returned by `tools/list`, including injected-registry tools,
  `semantic_batch`, and opt-in `semantic_reload`, has an `outputSchema`.
* Every captured tool result, successful or failed, is validated against its
  advertised output schema in the MCP protocol tests. Deliberate failures cover
  every tool in the actual full catalog; protocol failures are checked separately.
* Every native direct or batch outcome includes the schema-required
  `structuredContent.result`; every tool's `content` retains the actionable
  outcome and next-step guards without repeating the full result.
* Full serialized structured JSON in text follows MCP's SHOULD recommendation.
  A compact essential projection is a bounded compatibility tradeoff, not
  equivalent protocol guidance compliance. Client presentation and model
  visibility remain client-controlled.
* Source, inventory, page, diff, read-set, log, metric-record, and feedback
  artifact payloads appear once in structured fields. Text identifies those
  fields and cursors, and no absolute cap truncates essential guards.
* Error projections are validated against their paired advertised schema.
  Generic errors cannot carry an undeclared result, and native outcomes cannot
  carry a top-level error. Error state does not produce live continuation guards.
* Batch `diagnostic_delta` is advertised and returned only when execution
  reaches the final diagnostic capture.
* Output schemas do not over-constrain arbitrary telemetry phase names,
  diagnostic strings, or future envelope fields.
* Reload sends its success response and structured result before invoking the
  platform-specific re-execution path.

## Consequences

MCP clients can discover and validate both native results and error envelopes without
parsing human-readable text. The protocol response identifies the revision
that supports structured tool output. Typed projections provide
operation-specific results through the common transport envelope, while
formatters supply the human-readable presentation.
