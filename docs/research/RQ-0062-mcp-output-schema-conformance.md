<!-- Records wire-level MCP success and failure schema evidence before contract repairs. -->

# RQ-0062: MCP Output Schema Conformance

Status: Resolved
Date: 2026-10-04

## Question

Do every advertised Model Context Protocol (MCP) tool's successful and failed
structured responses conform to its advertised output schema, including
registry-projected errors and batch children?

## Method

First instrument existing MCP protocol tests with an independent JSON Schema
validator. Discover schemas through `tools/list`, correlate responses with
their request identifiers, and validate emitted `structuredContent` on both
success and failure. Preserve each test's observable wire and workspace
assertions. Add deliberate tool failure scenarios with catalog completeness
checks. Record the strict failing-test evidence before changing any production
schema or error response.

Only after the initial audit, choose a structural contract repair and rerun
the same checks without exceptions for error responses. New malformed-payload
checks must ensure a broad error branch cannot conceal invalid typed results.

## Protocol distinction

The outer tool response has `isError`; JSON-RPC errors have protocol error
codes. Tool `outputSchema` applies to `structuredContent`, so its alternatives
cannot depend on the outer flag. The test suite must validate both the payload
schema and consistency with the outer failure indication.

The [MCP specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
requires structured tool results to satisfy an advertised output schema.
ADR-0042 now requires error conformance as well as successful result conformance.

## Initial source observations

- The shared output schema requires `result`, while generic errors return
  `error` and `application_state` without `result`.
- Registry-owned known errors have private schemas and projectors; their
  projections are not necessarily the same shape as successful results.
- Failed verification returns a native object where the advertised result is
  a string. Maven failures similarly return native data without registry-owned
  output contracts.
- Batch child schemas accept success and error result projections without
  correlating their shape to child status.

## Strict audit before production repairs

The instrumented MCP suite reported 32 distinct failing tests and subtests.
The complete JSON test log is retained locally in
`.scratch/mcp-schema-red-valid-final.jsonl`. Counts include parent tests affected by
failed subtests; they are not a count of distinct defects.

Observed wire deviations:

- Generic execution errors omit `result`, which the shared schema requires.
- Native failed verification returns an object against a string result schema.
- Known error projections, including safe-delete refusals and a deliberately
  disjoint test fixture, are advertised against their success result schema.
- Native Maven error objects lack registry-owned typed output contracts.
- Successful batches serialize nil `introduced` and `resolved` diagnostic
  slices as null, while the schema requires arrays.

Base-profile tests also dispatch supported specialized tools that are hidden
from that profile's catalog. The validator uses the same registry's full catalog
for those results without changing the tested catalog; those calls expose the
same error-contract deviations.

The reusable validator checks both successful and failed tool results using
the advertised schemas and retains exact response bytes. Its confidence tests
cover invalid results, protocol errors, malformed server JSON, request-ID
correlation, and integer values beyond binary floating-point precision.

The deliberate failure matrix enumerates the actual `tools/list` catalog,
rather than assuming the CLI and test server advertise identical operations.
This exposed two missing initial scenarios: parameter introduction and parameter
object introduction. Both were added before production repairs. The matrix
validates native verification and failed batches by their own outcome fields,
and distinguishes unknown or disabled tools' JSON-RPC errors. Existing Maven
tests exercise language-specific native failures separately.

## Accepted repair

Use the existing operation registry's success and error contracts to drive
mutually exclusive native-result, generic-error, and known-error envelope
branches. Keep `isError` as the protocol status and independently check its
consistency with native verification and batch outcomes. Generic errors cannot
contain `result`, preventing them from accepting malformed known projections.
Correlate batch child status with its permitted result schema. Add registry
contracts for verification and Maven, and normalize empty batch diagnostic
arrays at the MCP boundary. ADR-0042 records the resulting invariant.

Batch verification metadata uses the native verification contract independently
of a standalone verifier's output projector. A custom-registry regression
exercises a string standalone projection alongside native batch metadata.

## Final verification

All existing MCP wire-test server constructions now validate captured tool
responses against the corresponding `tools/list` schema. The independent
validator preserves exact numeric values in both schemas and payloads, including
large integer enum values. It retains byte-for-byte wire output and checks
outer `isError` consistency separately from payload conformance.

Deliberate failures cover all 40 tools in the default full catalog, plus both
Maven compile and Maven test failures. The matrix checks failure status,
schema conformance, and unchanged workspace files. Negative payload tests
reject malformed known-error projections, contradictory batch rows and aggregate
statuses, and diagnostic deltas on failed batches.

`make check` passes, including race-enabled shuffled tests, lint, and
documentation verification. Its optional vulnerability scan is skipped because
`govulncheck` is not installed. A one-iteration cold-start MCP benchmark
also validates its verification response using the same subprocess's advertised
schema after timing stops.

## Coverage boundary

The opt-in reload tool's generic error branch can be validated as a payload,
and disabled reload calls are exercised as JSON-RPC errors. Enabled reload
executes the running process itself, so an in-process test must not invoke it.
Its current handler announces `reloading` before executing; if execution then
fails, it sends a second response with the same request ID. That separate
one-response lifecycle defect remains outside this schema repair. The deliberate
failure matrix covers the actual default full catalog, while existing Maven
failure tests cover its language-specific catalog.
