<!-- Records preliminary measurements and open policy choices for session-local staged-edit resource accounting. -->

# RQ-0067: Staging Resource Accounting and Defaults

Status: Open
Date: 2026-10-10

## Questions

Which deterministic resource dimensions should bound a session-local staged edit
journal, and what numerical defaults are useful without rejecting ordinary
refactors or implying a process heap, RSS, or latency guarantee? What broader
operation and replay evidence is needed before fixing defaults?

## Context

ADR-0032 names a state-journaled staging lifecycle (`semantic_stage`, native
buffered mutations, `semantic_stage_drop`, `semantic_stage_abort`, and
`semantic_commit`). The accepted lifecycle binds volatile state to an MCP
session and canonical workspace root. Pre-write conflicts retain the journal;
successful publication consumes it; partial publication freezes the journal in
a terminal failure state until abort. Resource limits must preserve those
lifecycle guarantees when a candidate mutation or replay is rejected.

The earlier proposal of 50 steps and 10 MiB did not establish which byte
dimension was bounded or provide measurements for either number.
`sourcebuf.New` eagerly loads each Go file in the selected source set. For
nonempty files, it stores `original: bytes.Clone(data)` and `content: data`, so
the initial preimage and working bytes occupy distinct backing allocations.
The buffer may also observe local replacement-module source sets. Clean
baseline bytes therefore dominate some sessions before any edit, while
overlays, changes, replay, diffs, and publication create transient copies.

## Preliminary Measurement

An ignored scratch probe generated fresh Go modules and used
`sourcebuf.New`, `Buffer.AuthorizeWrites`, and `astedit.ReplaceBody` to apply
sequential body replacements to one target. It retained each serialized
parameter payload and simulated dropping the midpoint by keeping the old buffer
alive, constructing a new buffer, and replaying all other stored requests.
Three runs used the repository-managed Go environment, with a 45-second context
deadline per process. This was a synthetic one-operation-shape probe, not a
production benchmark or an implementation of the MCP staging protocol.

| Go files / requested source bytes | Steps | Buffer init | Appends total / average | Drop rebuild and replay | Tool + params bytes | Dirty preimage + current bytes | Total allocations | End HeapAlloc delta |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 10 / 51,200 | 10 | 1.520 ms | 5.013 ms / 501 µs | 5.248 ms (9 replayed) | 772 | 10,243 | 3,261,896 | 567,264 |
| 100 / 1,572,864 | 50 | 8.763 ms | 31.849 ms / 637 µs | 28.566 ms (49 replayed) | 3,892 | 31,459 | 45,475,288 | 8,592,232 |
| 500 / 10,485,760 | 100 | 33.741 ms | 57.060 ms / 571 µs | 58.605 ms (99 replayed) | 7,794 | 41,946 | 146,795,528 | 61,490,104 |

Raw stdout and detailed method caveats are preserved in the ignored
`.scratch/staging-resource-probe-results.txt`. Total allocations are cumulative
allocations during the measured work. HeapAlloc is one end-of-run sample; it is
not peak heap or peak RSS. The probe did not measure per-step candidate
isolation: appends mutated the active buffer directly. Its replay result only
models one midpoint drop on this simple body-replacement sequence.

The 100-step synthetic case completed appends and replay in under 60 ms, but
that does not establish a useful maximum. It exercises one cheap operation and
small serialized requests; it does not cover rename, multi-file edits, costly
type analysis, diagnostics, hooks, local replacement modules, large bodies,
concurrency, or candidate isolation. The observed allocation totals include
buffering and AST-edit work but do not identify peak simultaneous residency.
These data cannot justify 50 steps, 10 MiB, or replacement defaults.

Candidate isolation is an additional cost and correctness constraint, not a
free extension of this probe. A safe isolated revision likely needs a deep fork
of the complete buffer state (`files` and both byte slices, source trees,
symlink aliases, and authorized write roots) under the buffer lock while
preserving captured disk preimages instead of re-reading disk. The current
mutation handler can also synchronize a retained gopls session while observing
the candidate, even if the handler later errors. Rejection or failed drop
replay must therefore restore the accepted revision in retained sessions or
invalidate those sessions if restoration fails. Fork time/allocations,
gopls-sync cost, and failure recovery are unmeasured and belong in the next
matrix before selecting numeric defaults.

## Accepted Accounting Contract

The accepted deterministic contract has three separate
dimensions, bound to one active MCP session and canonical workspace root:

1. `max_stage_steps`: number of accepted journal entries. Check before
   dispatching another staged mutation. This bounds retained entry count and
   replay cardinality only; it is not a wall-clock limit.
2. `max_stage_request_bytes`: cumulative UTF-8 bytes of the registered tool
   name, one fixed separator byte, and canonical compact JSON for each accepted
   parameter object. Check before dispatch. This bounds retained replay input,
   not Go object overhead.
3. `max_stage_changed_source_bytes`: for each unique dirty path, sum the byte
   lengths of its retained disk preimage and current staged contents. Include
   preimage bytes when a file is deleted and current bytes when a file is new;
   clean loaded paths contribute zero. This is a logical changed-source budget,
   not total source-buffer bytes.

The changed-source budget intentionally excludes clean eager baselines,
canonical path strings, maps, syntax trees, language-server state, diff strings,
and temporary clones. It therefore makes no aggregate Go heap or RSS promise.
Neither step count nor these byte counters establish a replay-time guarantee.
Request context cancellation remains the execution stop mechanism.

All count and payload checks should happen before dispatch. Changed-source
accounting requires checking an isolated candidate revision and swapping it
into the journal only after it fits. A rejected candidate must leave the
previous buffer and journal available. Drop replay should use a candidate
buffer and the request context; cancellation, conflict, invalid replay, or a
limit failure discards the candidate and preserves the old journal. Candidate
isolation and its allocation cost remain unmeasured. If the implementation
cannot isolate a candidate, this contract is not satisfied by mutating the
live buffer and attempting an unverified rollback.

## Provisional Policy Approval

On 2026-10-10, after Gemini review of this accounting and isolation contract,
the user approved configurable provisional defaults of 50 accepted steps,
10 MiB retained request bytes, and 10 MiB changed-source bytes, to learn when
real workloads reach the limits. These are operational policy choices, not
thresholds derived from the preliminary probe. Budgets are immutable server-startup
configuration, with no per-call overrides. ADR-0032 records the accepted contract.

Limit refusals must identify the dimension, current usage, attempted usage,
and configured limit while preserving the accepted stage. The caller can commit,
drop steps, or abort; rewriting replacement code does not solve step exhaustion.
Measurements should distinguish which budget was hit and what operation shape
and stage size produced it, without requiring source payload collection.

## Open Work

- Measure a broader representative operation matrix, including large and
  multi-file mutations, replay/drop positions, local replacements, candidate
  deep-fork cost, gopls synchronization and recovery, and concurrent sessions.
- Evaluate actual limit hits and broader measurements before changing the
  provisional defaults. Keep labels clear that they are operational policy,
  not heap or latency guarantees.
- Add observable tests proving pre-dispatch and candidate rejection preserve
  accepted staged edits, request-context cancellation preserves the old journal
  during drop, and new/deleted path accounting follows the contract if those
  paths become stageable.

## Decision Gate

Keep this RQ open for broader measurements, candidate-isolation costs, and
evaluation of observed limit hits. Acceptance of provisional policy defaults
does not resolve the empirical capacity questions.

## Restoration checkpoint: 2026-10-12

The staged-edit implementation is restored from snapshot `32207d5` and adapted
to the current shared source buffer and canonical typed outputs. The preliminary
measurements above are historical evidence from that snapshot, not measurements
of the restored implementation. Their scratch artifacts have not been reproduced.
The step and byte defaults remain provisional operational policy.

The restored lifecycle also bounds stage age to 30 minutes by default, using
immutable server configuration. This is a retention policy, not a measured
capacity or execution-time guarantee. It leaves the broader measurement questions
open. Public behavior checks cover ownership, replay, budget rejection, and
publication; they cannot substitute for allocation or latency measurements.
