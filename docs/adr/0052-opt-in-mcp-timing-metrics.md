# ADR-0052: Opt-In MCP Timing Metrics

Status: Accepted
Date: 2026-09-26

## Context

MCP responses included request phase timings and server startup timings on every semantic call. Repeated telemetry enlarged ordinary results and competed with task content for model attention. Users still need those measurements for diagnosis and benchmark analysis.

## Decision

Direct semantic and batch responses return their structured result without timing fields. The server retains timing records for the latest 100 timed calls in memory and exposes them through a separate `semantic_metrics` tool. The tool returns the most recent records first, defaults to 10 records, and accepts a limit from 1 through 100. Each record identifies its sequence, tool name, completion time, request metrics, and optional startup metrics. The metrics query is not itself recorded.

An optional `--metrics-file` path persists every timed record as a JSON array using atomic writes. A new server process loads an existing file and continues its sequence, allowing benchmark follow-up turns to share one file. Benchmark semedit arms enable this path inside their disposable fixture workspace; the harness retains task-only records beside the session transcript at the end of the run and records the durable path in provenance.

## Invariants

Gopls clients contribute request-scoped phases named `gopls.request.<method>`
and `gopls.notify.<method>`. Each records the attempted call count and cumulative
client-side elapsed milliseconds. Requests include transport queueing, writing,
and waiting for the correlated response or error/cancellation; notifications
measure transport submission through write completion, without a server reply.
They do not measure asynchronous server work triggered by a notification;
that work may affect subsequent requests. Internal best-effort cancellation
frames are not separately timed as notification calls.
These are elapsed timings, not gopls CPU measurements. Parameters, source text,
and responses are not retained by this instrumentation. Other language clients
remain uninstrumented unless explicitly configured with a timing prefix.

Method phases keep their exact names even inside a scoped before/after
verification phase. They nest within operation phases and must not be summed
with enclosing phases as disjoint time. Concurrent calls can also overlap.
Each request additionally records `gopls.send.<method>` for serialization,
outbound queuing, and transport write completion, followed by
`gopls.wait.<method>` for response delivery or termination. A failed send has
no response-wait phase. The request duration includes both children; a response
can arrive during the send phase, so wait duration is client waiting rather
than server execution time.
Durations accumulate before conversion to milliseconds so repeated submillisecond
calls are not individually truncated. Zero milliseconds can still have a
nonzero count. Errors and cancellations contribute attempted-call timing;
these phase counters do not distinguish successful calls from failures.

- Standard semantic and batch output schemas require only `structuredContent.result`; they do not advertise request or startup timing fields.
- Timed successes and errors enter the in-memory history with their tool name. The history remains bounded to 100 records and is safe for concurrent access.
- `semantic_metrics` is available in full and mutations-only profiles and does not add its own timing record.
- Startup timing stays attached to the first timed semantic call and can be read from that call's history record.
- The in-memory history is scoped to one running MCP server process and is discarded when that process exits. File persistence is opt-in, preserves all records, and reports read, decode, or write failures as server errors.
- Benchmark evidence excludes diagnostic reflection calls from the retained task metrics file.

## Consequences

When Go execution tracing is enabled, timing scopes also emit native
`runtime/trace` regions using the resolved phase name. This includes scopes
without an attached metrics recorder. Trace regions are ordered and expose
scheduling and blocking stacks; metrics remain aggregate count/duration values
with their existing schema. Native tracing is disabled by default and does
not add timeline data to ordinary MCP responses or metrics history.

Exact `go.*` regions attribute environment resolution, session use and lock
waits, module/package scope loading, document synchronization, snapshots,
revision validation, symbol resolution, and rename publication. Fingerprinting
additionally contains `go.scope.embed_patterns` and `go.scope.hash_inputs`
to separate initial parsing/structural preparation and embed-directive collection
from fresh-byte validation and input hashing. The hashing phase can reuse compact
structural encodings only when the newly read content digest matches within the
same scope/environment identity. Function extraction separately records
`go.extract.validate_original`, `go.extract.validate_candidate`, and nested
`go.extract.packages_load` phases so the two program validations and compiler
reloads remain distinguishable. All regions nest
within operation and verification phases, so inclusive durations overlap.
Trace files describe our process, including subprocess waits; they do not
capture CPU execution inside gopls or Go subprocesses. Native
`disk.create`, `disk.write`, `disk.sync`, `disk.close`, `disk.validate`,
and `disk.rename` regions separate atomic publication costs. These regions
contain no file names or bytes and do not add metrics fields. Benchmarks enable
tracing explicitly, and traced results are diagnostic rather than acceptance
measurements.

Routine tool results stay smaller, while clients can request timing data when it is useful. Benchmark runs can recover server timings after the MCP process exits. Atomic persistence adds disk work to each timed tool response in those runs; the request's recorded `total_ms` is captured before persistence, while the task wall clock includes it.
