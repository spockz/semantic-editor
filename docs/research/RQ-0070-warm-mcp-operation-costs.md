<!-- Keeps latency expectations and benchmark validity together while the warm-session investigation remains open. -->

# RQ-0070: Warm MCP Operation Costs and Session Continuity

Status: Open
Date: 2026-10-14

## Expectations

The user requires p99 below one second for all warm operations and reads within
10 ms. Cold analysis of very large projects is a separate measurement. Ordinary
source edits, additions, deletions, filename changes, and same-module moves should
retain the gopls process. Fresh diagnostics and correct reference updates remain
part of the operation; deferring necessary work until the next request does not
satisfy the latency target.

The narrower mutation budgets below are proposals, not accepted service promises.
Interpret the 10 ms read target as p99 for now; record maximum latency as well so
tail failures remain visible. Define fixture size, response size, machine, Go and
gopls versions, and sample count alongside every result.

| Operation family | Tools | Warm p99 target | Status |
| --- | --- | --- | --- |
| Reads | `semantic_workspace_outline`, `semantic_outline`, `semantic_lookup`, `semantic_inspect_symbol`, `semantic_inspect_construct`, `semantic_find_references`, `semantic_metrics` | 10 ms | User target; measure full response |
| Local edits | `semantic_replace_body`, `semantic_replace_construct`, `semantic_replace_decl`, `semantic_insert_construct`, `semantic_remove_construct`, declaration/function/type/field/case/composite insertion, `semantic_set_doc_comment`, `semantic_assertion_mode` | 50 ms | Proposed; general ceiling remains below 1 s |
| Symbol rename | `semantic_rename`, including callers and verification | 100 ms | Proposed; general ceiling remains below 1 s |
| Same-package declaration move | `semantic_move_construct` | 100 ms | Proposed |
| Same-package filename change | No general Go file-rename operation exists | 50 ms | Proposed capability; zero gopls restarts |
| File creation | `semantic_scaffold_file` | 50 ms | Proposed; zero gopls restarts |
| Semantic refactorings | Extract/inline function, variable, constant, interface/type; implement contract; change signature; introduce parameter/object; migrate type; cross-package move | 500 ms | Proposed; scale callers and edit size |
| Imports and verification | `semantic_organize_imports`, `semantic_verify` | Below 1 s | User general ceiling; distinguish selected and workspace scopes |
| Composition and staging | `semantic_batch`, `semantic_stage`, `semantic_stage_drop`, `semantic_stage_abort`, `semantic_commit` | Below 1 s | User general ceiling; declare number of edits and changed bytes |
| Recovery | `semantic_snapshot`, `semantic_undo` | Below 1 s | User general ceiling; declare snapshot scope |
| External execution and lifecycle | `semantic_add_build_dependency`, `semantic_maven_compile`, `semantic_maven_test`, `semantic_reload`, `report_feedback` | Below 1 s where applicable to warm operations | Open: network, external build, and intentional reload boundaries need explicit treatment |

## Existing move capabilities

Go `semantic_move_construct` relocates declarations within or between files.
Cross-package movement supports exported functions, standalone constants, aliases
whose type expressions use only predeclared type names, zero-valued variables with
predeclared structural types, and bounded defined-type groups within one module,
subject to reference completeness, test-reference, dependency, build-constraint,
method-set, initialization, and type-check guards. It
does not provide a general file relocation or package migration operation. Java
public-type rename can rename its file within the package; Java type moves can
update the package and references. Keep these separate benchmark scenarios.

## Measurement contract

Use one retained MCP server and real gopls for each measured sequence. CLI calls
start temporary sessions and cannot stand in for warm MCP calls. Warm compiler
caches and an explicit diagnostic request before timing, then warm the selected
operation. Time request encoding through complete response decoding, including
mandatory formatting, diagnostics, and continuation construction. Exclude fixture
generation, initialization, warm-up, and final correctness checks.

Count gopls process starts and require exactly one across warm-up and the measured
sequence. File-membership churn must eventually satisfy the same invariant, not
be hidden by an extra warm-up after a restart. Measure the first read after each
edit in addition to repeated reads on an unchanged revision. Use reversible edits
so each iteration performs actual work. Check published sources, references, and
diagnostics outside the timed region. Never accept error responses as fast success.

Report p50, p99, maximum, sample count, CPU time, allocations where attributable,
response bytes, and gopls starts. Separate cold process startup, warm-process
first use, repeated steady-state calls, and first call after an edit. Collect at
least 100 samples before interpreting p99; 1,000 or more is preferable. Short smoke
runs validate the harness but do not establish tail latency.

## Initial implementation

`internal/mcp/warm_cost_bench_test.go` starts the production MCP server over JSON-RPC
pipes in the benchmark process and launches real gopls. It does not include a
separate semedit executable or operating-system stdio scheduling. It covers lookup,
symbol inspection, file outline, workspace outline, references, and reversible
exported-symbol rename in one-package and twenty-package fixtures. Consumer
packages exercise cross-package reference updates. It reports p99, sample count,
budget ratio, and gopls starts. The existing managed-gopls fixture supplies process
counting; no analysis mock substitutes for gopls.

Budgets report by default, allowing baseline collection while targets are unmet.
Set `SEMEDIT_BENCH_ENFORCE_BUDGETS=1` to fail a case that exceeds its target or the
one-second ceiling. Session continuity always fails on an unexpected restart.

```sh
make check
go test ./internal/mcp -run '^$' -bench '^BenchmarkMCPWarmOperationCost$' -benchtime=1000x -count=1 -timeout=30m
```

Use the repository Go environment and configured gopls on PATH. A `-benchtime=1x`
run is only a harness smoke check. Do not compare timings gathered while the full
test suite, linters, or another benchmark is running.

## Measured baseline: 2026-10-14

The full run completed 1,000 measured calls for each of twelve cases, totaling
12,000 calls in 476.718 seconds. Repository checks completed before measurement;
no other repository checks or benchmarks ran concurrently. Initialization and
warm-up remained outside the timed samples. Every case retained one gopls process.

The machine reported Apple M5 Pro, Darwin arm64, and Go 1.27.2. The configured
gopls is a local development build compiled with Go 1.27.1; the retained metadata
identifies both its launcher and executable by SHA-256. This is one independent
run, so it establishes a baseline rather than cross-run variability or confidence
intervals.

| Operation | 1 package p99 | 20 packages p99 | Agreed target |
| --- | --- | --- | --- |
| Lookup | 0.756 ms | 0.688 ms | Within 10 ms |
| Symbol inspection | 0.630 ms | 0.665 ms | Within 10 ms |
| File outline | 7.849 ms | 6.342 ms | Within 10 ms |
| Workspace outline | 0.723 ms | 2.462 ms | Within 10 ms |
| References | 30.200 ms | 41.050 ms | Within 10 ms: both miss |
| Exported-symbol rename | 139.500 ms | 292.800 ms | Below 1 s: both meet |

All measured cases meet the general one-second warm ceiling. References miss the
read target in both fixtures. Rename misses the narrower proposed 100 ms budget,
while meeting the agreed general ceiling. Execution success does not mean all
performance targets passed; latency enforcement was disabled for baseline
collection, while process-continuity and operation-success assertions were active.

Retained evidence:

* [Structured results and environment metadata](assets/warm-mcp-operation-costs-20261014.json)
* [Raw Go benchmark JSON event stream](assets/warm-mcp-operation-costs-20261014.jsonl)

The structured results include sample counts, mean and p99 latency, target
outcomes, source hashes, binary hashes, and limitations. The raw stream contains
aggregate benchmark results, not individual request samples. CPU costs and
response sizes remain unmeasured. These small synthetic fixtures do not establish
latency on this repository, large projects, file-membership changes, or unmeasured
tool families.

## Same-package file rename benchmarks

`internal/mcp/file_rename_bench_test.go` adds two benchmark families, each with
one-package and twenty-package fixtures:

| Benchmark | Timed work | Continuity requirement |
| --- | --- | --- |
| `BenchmarkFileRenameFilesystemCost` | Actual `os.Rename` only | Filesystem lower bound; no MCP or gopls process |
| `BenchmarkMCPFileRenameCost/.../semantic_lookup` | Actual rename followed by the first MCP lookup of `Value` at its new path | One gopls start across warm-up, rename, lookup, and the subsequent verification check |
| `BenchmarkMCPFileRenameCost/.../semantic_verify` | Actual rename followed by the first full MCP verification | One gopls start across warm-up and the measured rename/verification sequence |

These cases rename `value.go` to `renamed.go` and back in the same directory.
Neither name changes test status or platform build selection, and file contents
remain identical. Consumer packages and declarations remain unchanged. There is
no general Go file-rename MCP operation, so these benchmarks explicitly measure
an external filesystem event and the production MCP server's reaction to it.

Before timing an MCP case, warm diagnostics and the selected reaction and assert
that gopls has started exactly once. After every rename, outside timing, assert
that the old path is absent and the destination retains the original bytes.
Validate that lookup resolves `Value` in the new file, or that verification
contains no diagnostics and includes completed Go coverage.

Lookup can complete without synchronizing gopls. Therefore its case performs an
additional verification outside the primary timed sample and checks the process
count afterward. A deferred restart still fails continuity. On any restart, log
the individual rename, reaction, and supplementary synchronization durations and
abort immediately, rather than collecting or publishing a misleading warm p99.
This also prevents spending 1,000 iterations repeatedly starting cold servers.

Successful cases report total p99 plus separate rename and reaction p99 values.
The filesystem-only case and rename-plus-lookup use the proposed 50 ms filename
budget. The first lookup also has the agreed 10 ms read target. Rename-plus-verify
uses the general one-second ceiling. Latency enforcement remains opt-in; process
continuity always fails on a restart.

```sh
make check
go test ./internal/mcp -run '^$' -bench '^(BenchmarkFileRenameFilesystemCost|BenchmarkMCPFileRenameCost)$' -benchtime=1000x -count=1 -timeout=30m
```

The measured run requested 1,000 iterations per case. Both filesystem-only cases
completed all samples: p99 was 0.05179 ms for one package and 0.04792 ms for twenty
packages. All four MCP cases failed after the first observed rename because the
gopls start count increased from one to two. They emitted no warm p99.

| MCP reaction | Packages | First reaction, single observation | Supplementary synchronization, outside timing | gopls starts |
| --- | --- | --- | --- | --- |
| Lookup | 1 | 0.3833 ms | 1103.1460 ms | 2 |
| Lookup | 20 | 0.3535 ms | 1112.3580 ms | 2 |
| Verification | 1 | 1047.0659 ms | None | 2 |
| Verification | 20 | 1032.7126 ms | None | 2 |

These are single failure observations, not percentile estimates. Correct path
lookup, unchanged source bytes, and clean Go verification all passed before the
continuity assertion rejected the case. The cheap lookup therefore does not prove
that gopls remained synchronized: its next verification still pays the restart.
The benchmark command exits with status 1 for these four contract violations;
normal repository `make check` passed before the run.

Retained evidence:

* [File-rename results, validity status, and source hashes](assets/warm-mcp-file-renames-20261014.json)
* [Raw file-rename benchmark JSON event stream](assets/warm-mcp-file-renames-20261014.jsonl)

The cases expose the existing source-file-set invalidation policy. Runtime restart
behavior remains unchanged. Package-changing moves and filename changes that
alter build constraints remain separate, unimplemented benchmark scenarios.

## Remaining cases and questions

### Limits of the initial measurements

The retained baseline measures a narrow steady state, not a general tool latency
guarantee. In particular:

* Fixtures contain one or twenty tiny packages with shallow caller fan-out. They
  omit realistic source volume, test variants, build constraints, replacements,
  generated code, embedded files, and multiple workspace modules.
* Most reads repeat an unchanged revision. They do not establish first-read
  latency after mutation, diagnostic churn, failure recovery, or mixed workloads.
  Requests are sequential and do not establish latency under request queues or
  concurrent clients. The original rename alternates two names; batch and
  staging cohorts alternate two content states, which can favor hot-content
  caching over a diverse edit trace.
* Fixtures have no project-configured verification hooks. Built-in formatting
  and diagnostics are exercised, but an external linter, build, or test hook can
  dominate real-project mutation time and is not represented by these numbers.
* The production server runs inside the Go benchmark process over pipes. Results
  include JSON-RPC work, but exclude launching and scheduling a separate semedit
  executable and any coding client integration. The benchmark build and local
  gopls are identified, but are not automatically the versions or configuration
  used by the installed tool in another project.
* Historical raw streams retain aggregates rather than individual observations.
  They do not report response size, phase attribution, language-server CPU, or
  language-server allocations. Benchmark-process allocation measurements would
  not measure gopls allocations.
* Each published cohort has one independent run. A thousand consecutive calls
  provide only about ten observations in the slowest one percent; they do not
  establish cross-run variability, confidence intervals, or behavior under
  machine contention.
* The original success guard rejects MCP errors but does not validate every
  native outcome. Original page cases validate populated data during warm-up
  only. Original rename continuity is checked before its final verification.
  Strengthening these checks does not retroactively change retained evidence.
* Latency enforcement was disabled for the original baseline. Successful
  execution is distinct from meeting the target: references already miss 10 ms.
  Same-package external file rename already fails process continuity entirely.
* Go suppresses the normal metric line for failed cases. The original failed
  diagnostic-page run retains its p99 error but not all aggregate statistics.
  Expanded cases must emit their aggregates before enforcing budgets.

File-rename lookup timing also excludes its supplementary verification. That
separate synchronization exposes a restart, but the short lookup sample is not
the cost of fully reconciling the renamed file with gopls.

The expansion adds representative tool families and strengthens result checks.
It does not remove the fixture, transport, CPU, and independent-run limitations
above. Cold analysis remains separate from warm cost, and ordinary file changes
still require incremental synchronization rather than a newly warmed daemon.

### Expanded measurement requirements

`BenchmarkMCPWarmOperationActualWork` and its named component collector live in
`internal/mcp/warm_actual_bench_test.go`, sharing the production MCP client and
fixture helpers with the original benchmark. Its cases cover body edits with
dependent reads, 100-record metric queries, selected-file snapshots, construct
inspection, clean verification, dirty import organization, two-file batches,
and retained staging through commit or abort. Each runs on one- and twenty-package
fixtures. All helper files are installed before gopls initialization.

New reads and check-only verification are explicitly repeated unchanged-revision
workloads. The body/read cohort uses a fresh valid integer return value on each
iteration, while batch and staging alternate two valid final states. Every
published effect is checked. Import-organizing inputs are dirtied with an atomic rewrite
outside timing; subsequent synchronization and import organization remain inside
the timed tool call. This is an external-edit reaction workload rather than an
unchanged-file import no-op.

The edit/read sequence times the edit, its first lookup, and the following
inspection separately, plus their total. No validating semantic read may warm
that lookup before measurement. The later inspection observes edited source but
is not an independently isolated first-read measurement.
Workflow totals sum the measured MCP call durations and exclude benchmark-only
oracles between calls, as well as any model or user time between requests.

Staging reports control and overlay costs separately from publication. A staged
edit alone is incomplete work; the completed stage/edit/commit sequence must
include final verification and source publication before it is compared with a
completed direct mutation. Abort is a control workload, not a successful edit.
Snapshot cases capture `value.go` only; their real immutable journals accumulate
under the fixture's excluded `.scratch/snapshots` directory. They do not measure
capturing every source file in the twenty-package workspace.

Stage abort is retained as a continuity probe because development measurements
found that aborting a staged edit makes the next verification start a new gopls.
It measures the first stage/edit/abort and its subsequent synchronization,
rejects any additional process, emits `WARM_COST_REJECTED`, and stops that cohort.
It does not repeatedly warm a replacement daemon or publish a percentile from a
single rejected observation. A deliberate failing benchmark is useful evidence
of an unmet lifecycle requirement and must not be removed to make the suite pass.

Every measured response must satisfy its native outcome, not just omit
`isError`: clean verification needs verified Go coverage, a batch needs a clean
final result and its intended edits, and a commit needs actual publication.
Diagnostic pages must retain their expected populated contents on every call.
Check process continuity after all dependent reads and final synchronization.

Report p50 and maximum alongside p99, with complete consumed response bytes
including both text and structured content. These are wire-size measurements,
not token counts or estimates of what a particular client forwards to its model.
Read targets are 10 ms; new mutation and control targets use the accepted
strictly-below-one-second ceiling, not the narrower provisional family targets.

The benchmark MCP client bounds each RPC at 30 seconds. A timeout cancels its
server, closes both transport pipes, and fails the cohort; it is never recorded
as a successful latency sample. This is a benchmark guard, not a production tool
timeout. In the local Go 1.27.2 runner, `testing.M.Run` stops the test alarm before
calling `runBenchmarks`, so `go test -timeout` alone does not bound these RPCs.
Development reruns stalled while writing the request after a references
response. The synchronous pipe client could decode a complete JSON object before
draining its newline, leaving the server blocked on response output while the
client tried to write its next request. This is a benchmark transport defect,
not evidence of a gopls references hang. The client must consume complete newline
frames and count their full byte lengths. Interrupted pre-correction runs are not
measurements of successful warm performance.

Go also executes a one-call calibration before a fixed-count benchmark. Its
aggregate is logged but must not trigger the percentile budget gate: a single
slow call previously prevented the requested import-organizing measurement.
Latency enforcement now requires `b.N > 1`; correctness and process-continuity
checks remain unconditional. Smoke aggregates still do not establish p99.

### Expanded measurements: 2026-10-15

The expanded measurements contain 25,200 timed MCP calls across eighteen
completed cohorts. Seventeen cohorts have 1,000 observations each; the slow
twenty-package import case has 200. Edit/read and staging observations contain
three timed calls each. Warm-up, one-call calibration, supplementary validation,
and the two rejected abort probes are excluded from that call count.

All completed cohorts retain exactly one gopls process. All populated read
cohorts meet their 10 ms p99 budget. Check-only verification uses the separately
declared one-second verification budget. The table reports p99 in milliseconds;
the machine-readable evidence also retains p50, maximum, sample counts, response
bytes, source hashes, Go 1.27.2, the gopls executable hash, and Apple M5 Pro host
identity.

| Measured component | One package p99 ms | Twenty packages p99 ms | Budget ms |
| --- | --- | --- | --- |
| Body replacement, fresh values | 128.034 | 225.246 | 1000 |
| First lookup after replacement | 0.983 | 0.976 | 10 |
| Following symbol inspection | 0.993 | 1.104 | 10 |
| Edit, lookup, and inspection total | 129.694 | 225.945 | 1000 |
| Metrics, 100 records | 0.885 | 0.908 | 10 |
| Snapshot, selected `value.go` | 54.405 | 65.111 | 1000 |
| Construct inspection | 0.517 | 0.433 | 10 |
| Clean check-only verification | 66.733 | 113.953 | 1000 |
| Organize freshly dirtied imports | 241.442 | **1852.009** | 1000 |
| Two-file batch with final verification | 248.966 | 396.070 | 1000 |
| Stage creation | 78.152 | 121.849 | 1000 |
| Deferred staged body edit | 26.044 | 49.341 | 1000 |
| Commit with publication and verification | 131.209 | 189.243 | 1000 |
| Stage, edit, and commit total | 214.088 | 322.266 | 1000 |
| Captured diagnostic page, 50 items | 0.557 | 0.561 | 10 |

Twenty-package import organization fails the one-second requirement without a
restart or project-configured external verification hooks. Its 200 observations
have median 1732.466 ms and maximum 1869.282 ms. This smaller sample gives only
about two observations in the slowest one percent and does not qualify a passing
SLO; even its median exceeds the budget. Phase attribution remains unresolved.
The one-package following inspection also reached a maximum of 10.402 ms despite
meeting its p99 read target. These are p99 results, not a guarantee that every
read finishes within 10 ms.

Both abort probes fail continuity on their first observation: process count is
one immediately after abort and two after follow-up verification. That follow-up
takes 144.784 ms for one package and 176.360 ms for twenty packages. Neither probe
has a published warm percentile. Together with the separately retained filename
change probes, this leaves ordinary session continuity unmet.

The [combined report](assets/warm-mcp-expanded-costs-20261015.json) links three
separately identified runs:

| Run | Completed observations | Outcome and evidence |
| --- | --- | --- |
| Initial expanded run | Fifteen cohorts, 1,000 each | [Summary](assets/warm-mcp-expanded-partial-costs-20261015.json), [event stream](assets/warm-mcp-expanded-partial-costs-20261015.jsonl). Retained 21,000 calls; interrupted after a one-call import calibration incorrectly blocked the requested full run. Includes the one-package abort rejection. |
| Slow import run | One cohort, 200 | [Summary](assets/warm-mcp-expanded-slow-imports-20261015.json), [event stream](assets/warm-mcp-expanded-slow-imports-20261015.jsonl). Completed with a latency-budget failure. |
| Remaining batch and staging run | Two cohorts, 1,000 each | [Summary](assets/warm-mcp-expanded-remaining-costs-20261015.json), [event stream](assets/warm-mcp-expanded-remaining-costs-20261015.jsonl). Retained 4,000 calls; includes the twenty-package abort rejection. |

The follow-up source differs only in the post-sampling calibration budget guard;
other recorded source hashes and the gopls binary are unchanged. Source identities
remain separate rather than describing these commands as one uninterrupted run.
Each summary records the exact command arguments and repository Go scratch root.
All measurements ran without concurrent test, lint, or benchmark jobs from this
task. Fixed cohort ordering, one independent run, and the synthetic workloads
still prevent generalizing the table into a project-wide latency guarantee.

### Reproducible cold/hot table on twenty source files

`make bench-mcp-tools` runs `BenchmarkMCPColdHotToolCosts` and
`BenchmarkMCPColdHotFileEdits`. The tables report cold/hot p99, sample counts,
and lifecycle/budget status. The original baseline matrix has exactly twenty
active Go source files in twenty packages;
scenario helpers are folded into existing files, and immutable `.scratch`
journals are excluded. Earlier package-count benchmarks can add helper files,
so their recorded measurements must not be relabeled as this exact corpus.

Cold means a fresh in-process MCP server per sample, with no semantic warm-up
before the timed first tool call. Required gopls startup and analysis occur
inside that call. `initialize`, source reset, and final correctness checks are
outside timing. Shared compiler and disk caches are retained. A tool using only
AST data may start no gopls at all. Hot means a prepared session retaining its
gopls through all measured calls and final synchronization.

Commit and abort require a staged edit; populated metric reads require recorded
calls; diagnostic pages require a captured report. These prerequisites preclude
an isolated first-call cold measurement. The table marks that value unavailable
and reports a separate cold workflow total. Hot metric and page prerequisites
are captured once, while each commit receives a real pending edit. Setup must
not silently turn a cold sample into a warm sample or an edit into a no-op.

Defaults are `COLD_SAMPLES=100` and `HOT_SAMPLES=1000`. The cold p99 therefore has
about one observation in the slowest one percent; increasing its count improves
tail resolution but does not replace independent runs. `TOOL_FILTER` selects a
tool, and `ENFORCE_BUDGETS=true` enables hot latency failures. Native-result and
process-continuity failures remain fatal. Contract-only runs require
`TOOL_SMOKE=true COLD_SAMPLES=2 HOT_SAMPLES=2` and must not label those timings
as qualified p99 measurements. The outer Go benchmark uses `-benchtime=1x`;
explicit sample loops, rather than that outer count, establish the reported
percentiles.

### Current coverage and deferred work

The Make target now includes a source-edit registry covering the supported Go
file-edit families and undo, with a registry coverage check. Build dependency
additions are explicitly excluded. `CORPUS=repository` exports committed HEAD
once, records its revision, and clones it per fixture using macOS copy-on-write
(`cp -cR`) or a portable full copy. It adds disclosed refactoring probes to
`internal/telemetry` and preserves the repository package graph. Setup and
cloning are untimed; cold calls use fresh clones, and hot calls retain a session
while restoring probe baselines outside the timer. This is a distinct corpus
from the original twenty-package measurements; no historical timings are
relabeled. Native mutation, diagnostics, and lifecycle validation must pass
before any qualified percentile is claimed.

The expanded registry has 33 scenarios across 31 source-edit tools, including
separate composite-element insertion and safe-delete variants. The repository
corpus has multiple Go modules, so lifecycle accounting attributes each gopls
start to its working module root. Distinct modules may each start one process;
a repeated root is a restart. A hot request that starts an additional module
session or loses its instrumentation log cannot claim a retained-session p99.
The initial repository trial used the old total-process check and is invalid
as lifecycle evidence; its native request failures remain useful findings.

Repository testing exposed safe-delete refusal on nested Go modules, structural
reads of compiler-cache artifacts outside the workspace, and Go structural
dispatch selecting Make at the repository root. Explicit Go selection is used
where the schema advertises it. These failures must remain visible rather than
removing repository packages or presenting refused operations as fast edits.

A qualified representative run of the expanded source-edit runner measured
`semantic_set_doc_comment` on twenty files across nineteen packages, with 100
cold and 100 hot observations: cold p99 **730.598 ms**, hot p99 **391.333 ms**.
The hot session retained its gopls process. This validates the measurement path,
not the full suite's performance. Aggregate evidence, benchmark source hashes,
and the development gopls binary hash are retained in
[`mcp-source-edit-costs-20261015.json`](assets/mcp-source-edit-costs-20261015.json).
The source-edit contract run exercised all 33 small-corpus scenarios; scaffold
creation failed its restart gate. Focused repository checks retained sessions
for body edits and ordinary removal; safe deletion and structural routing
failures remain visible.

A separate process CPU profile of 100 warm exported-symbol renames on the
twenty-package fixture measured p50 **282.469 ms**, p99 **336.145 ms**, and
maximum **365.820 ms**, retaining one gopls process. Profiling covered 33.01 s
elapsed and 6.04 s sampled CPU in the test process, including MCP, setup,
verification, and Go benchmark calibration. External gopls and Go subprocess
CPU is excluded. Filesystem syscalls account for 49.01% of sampled CPU, with
revision checks, source reads, and canonical path resolution on their stacks.
This supports mostly off-CPU execution for this fixture, but does not identify
the waits or establish that gopls requests are necessary. Profiled latency is
diagnostic evidence, not an unprofiled acceptance result. The raw profile and
matching binary remain in `.scratch/mcp-warm-rename.cpu.pprof` and
`.scratch/mcp-profile.test`; the log is `.scratch/mcp-warm-rename-profile.log`.

A matching process CPU profile of 100 dirty organize-imports calls on the
twenty-package fixture measured p50 **1636.714 ms**, p99 **1818.766 ms**, and
maximum **1819.525 ms**, retaining one gopls process. Profile duration was
173.93 s with 2.92 s sampled CPU, approximately 1.68% of one core averaged
over the profile. Scope includes setup, calibration, and validation; external
gopls and Go subprocess CPU is excluded. The process is mostly off-CPU, but
this does not attribute the waits to gopls or establish that our request
sequence is necessary. Organizing imports uses in-process `imports.Process`;
surrounding diagnostic captures use gopls. The raw CPU profile, matching test
binary, and benchmark log remain in `.scratch/mcp-warm-imports.cpu.pprof`,
`.scratch/mcp-imports-profile.test`, and `.scratch/mcp-warm-imports-profile.log`.
The one-second warm operation budget remains unmet.

The attempted full repository 100-cold/100-hot matrix has no final table and
its process is no longer alive. It provides no completed performance result.

Gopls requests and notification writes now contribute exact method phases to
`semantic_metrics`, including attempted counts and cumulative client elapsed
milliseconds. `make bench-mcp-gopls GOPLS_CALL_SAMPLES=10` retrieves these records
outside each operation timer. A ten-call attribution run per tool, on twenty
files across twenty packages, retained one gopls process in each cohort:

| Operation | Mean tool total ms | Diagnostic requests per call | Mean diagnostic request ms per tool call | Mean import formatting ms |
| --- | ---: | ---: | ---: | ---: |
| Rename | 303.4 | 40 | 8.5 | 8.5 |
| Organize imports | 1717.2 | 40 | 1532.8 | 13.1 |

The gopls rename request itself averaged 0.7 ms. Import removal spent most
elapsed time inside diagnostic requests, not the in-process import rewrite.
These are small diagnostic cohorts, not qualified p99 measurements. Timing
includes client transport and response waits; nested phases overlap. Notification
writes cannot measure asynchronous server work, which may surface in a later
request. Neither this result nor the CPU profile establishes why those
diagnostic replies are slow or whether their work is necessary. Aggregate
summaries, all ten measured records per tool, and source hashes are retained in
[`mcp-gopls-call-attribution-20261015.json`](assets/mcp-gopls-call-attribution-20261015.json).

Ordered native Go execution tracing is now available through
`make bench-mcp-gopls GOPLS_TRACE=.scratch/mcp-go-orchestration.trace`.
Timing scopes emit named regions only when runtime tracing is enabled, including
when a metrics recorder is absent. Exact `go.*` phases also appear in ordinary
`semantic_metrics` records. A trace-parser regression verifies the resolved
region labels. The retained trace covers our process, including blocked stacks,
setup, calibration, and verification; it does not profile child-process CPU.

Ten traced warm renames on twenty packages averaged **294.0 ms**. The principal
non-overlapping orchestration phases were:

| Phase | Mean calls per rename | Mean elapsed ms |
| --- | ---: | ---: |
| Session scope discovery | 4 | 109.4 |
| Additional diagnostic package enumeration | 2 | 41.2 |
| Rename publication of twenty files | 1 | 97.0 |
| Go environment resolution | 4 | 27.0 |

Scope discovery includes four scope-level package loads averaging 83.1 ms
combined; do not add that child total to the scope total. Publication includes
41 revision validations (20.2 ms combined) and many of the 502 snapshot reads
(17.1 ms across the whole rename). Snapshot/revision/publication phases overlap.
The trace preserves ordering and blocking stacks for inspecting that nesting.
Session synchronization averaged 2.1 ms; recorded lock-wait totals rounded to
zero milliseconds. Symbol resolution totals also rounded below one millisecond.
Diagnostic RPCs averaged 8.6 ms, so these orchestration costs remain material
despite a warm gopls process. The dirty import-removal cohort averaged 1765.5 ms,
including 1582.0 ms in diagnostic requests, consistent with the earlier finding.

All per-call records, source hashes, trace provenance, and phase aggregates are
retained in
[`mcp-go-orchestration-trace-20261015.json`](assets/mcp-go-orchestration-trace-20261015.json).
The binary trace is `.scratch/mcp-go-orchestration.trace`; inspect it using
`go tool trace`. These small traced cohorts are diagnostic evidence, not
qualified p99 or acceptance measurements. At that revision, atomic publication
and concurrent writer checks were unchanged; the later optimization is recorded below.

The discovery, publication, and diagnostic reuse follow-up is implemented under
ADR-0004, ADR-0032, and ADR-0060. Unaffected packages retain validated diagnostic
findings; changed packages and their reverse dependencies refresh their findings.
Unknown coverage falls back to full analysis. Explicit verification remains full.

1. Add local-variable and package-local symbol renames, including diverse new
   names alongside the original two-name exported-symbol fan-out case. Measure
   refusal and no-op paths separately.
2. Add external create/delete events. Scaffold is covered. Same-package external rename
   cases detect source-set invalidation. Incremental source membership is now
   covered by real-session create/delete tests; expand the measured event matrix.
3. Add guarded cross-package moves, stage-step dropping, and failed-operation
   recovery. Same-package declaration moves, extraction, signature changes,
   and undo now have source-edit cases.
4. Add larger fixtures with test variants, build constraints, local replacements,
   embedded files, generated sources, and larger responses.
5. Record finer phase timing and process CPU without confusing benchmark-process
   allocations with gopls allocations. Extend to an executable stdio transport.
6. Decide budgets for unbounded batch size and large responses. Build dependency
   additions and build/test execution are out of scope for this stage.
7. Incremental source-file-set synchronization is implemented. Extend continuity
   measurements to recovery and retained-stage lifecycle operations.
8. Repeated runs, concurrency, executable/Copilot overhead, and a continuous
   performance gate are accepted deferrals. Normal `make check` does not run
   performance measurements. The manual target remains the acceptance boundary.

The investigation remains open. The measured orchestration costs now motivate
accepted implementation changes in ADR-0004, ADR-0032, and ADR-0060: validated
discovery and diagnostic reuse, affected reverse dependency refresh, incremental
source membership synchronization, and target-only rename publication checks.
The earlier measurements remain evidence for their original implementation.

## Captured diagnostic page benchmark

`BenchmarkMCPDiagnosticPageCost` in
`internal/mcp/diagnostic_page_bench_test.go` measures the production MCP transport
with one- and twenty-package fixtures. Each captures at least 50 real diagnostic
findings before timing, warms a 50-item page, then samples complete query calls.
It requires exactly one gopls process and applies the 10 ms read p99 target.
Unlike a verification benchmark, this measures retrieval of a captured result;
each page must not repeat analysis. The earlier twelve-case results remain
historical observations of their original source revision.

Measured on 2026-10-15 with the read budget enforced:

| Fixture packages | Calls | Diagnostics per page | p99 | gopls starts |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 1,000 | 50 | 3.977 ms | 1 |
| 20 | 1,000 | 50 | 1.649 ms | 1 |

Both fixtures captured 50 real diagnostics before timing. The benchmark includes
MCP encoding, dispatch, presentation, response decoding, and result checks. It
does not measure another diagnostic analysis. Both cases passed the 10 ms read
budget and one-second general ceiling.

An earlier run, concurrent with repository checks and before the presentation
optimization, measured 8.044 ms and 16.061834 ms p99; the second case failed the
read budget. Inspection found that response ordering rebuilt a field-priority
map per key comparison. The presenter now shares that lookup and skips an
irrelevant outline JSON round trip. The rerun passes, but the two runs are not
a controlled A/B experiment, so their difference is not a precise causal speedup.

* [Results, source hashes, and provenance](assets/warm-mcp-diagnostic-pages-20261015.json)
* [Passing raw benchmark stream](assets/warm-mcp-diagnostic-pages-20261015.jsonl)
* [Earlier budget-miss stream](assets/warm-mcp-diagnostic-pages-pre-optimization-20261014.jsonl)

## Validated reuse and rename publication optimization

The retained manager now reuses package metadata while structural inputs remain
unchanged. It binds cached inputs into each new source buffer, tracks all source
digests including local replacements, and invalidates on membership, imports,
build constraints, cgo preambles, embedded inputs, symlink targets, and effective
Go configuration. Automatic captures reuse exact findings and refresh affected
reverse dependencies, including test variants. Explicit verification stays full.

Direct rename preflights every output, then checks only that target's canonical
path, regular-file identity, and exact bytes immediately before atomic replacement.
A conflict before any write permits one fresh lookup and LSP plan for the same
declaration selector. Partial publication reports the published paths and is never
replayed. The final check and filesystem replacement are separate operations.

The final run used `make bench-mcp-gopls GOPLS_CALL_SAMPLES=100` with the canonical
repository Go scratch directory. Each cohort used 20 Go files in 20 packages,
an in-process MCP connection, a warm gopls process, and 100 measured calls.
One calibration call per tool is excluded. Imports were dirtied before each timed
call; synchronization, import organization, diagnostics, and response consumption
remained inside timing.

| Tool | Calls | p50 ms | p99 ms | Maximum ms | gopls processes |
| --- | ---: | ---: | ---: | ---: | ---: |
| `semantic_rename` | 100 | 127.480 | 137.021 | 151.048 | 1 |
| `semantic_organize_imports` | 100 | 1641.999 | 1826.381 | 1857.174 | 1 |

Rename averages 127.83 ms in request metrics. It performs 22 snapshot reads and
zero module-wide revision checks, compared with the earlier trace's 502 reads
and 41 checks. Warm package-scope loads drop from four to zero. Publication across
20 files averages 81.57 ms, the remaining package listing 22.51 ms, environment
setup 6.32 ms, and diagnostic RPCs 3.47 ms. The first measured rename establishes
the cached diagnostic baseline; subsequent unchanged baselines need no pulls.

Dirty imports still miss the one-second target. They average 1651.15 ms overall,
including 1565.00 ms in diagnostic RPCs. Each call makes 40 diagnostic requests
and two structural package-scope loads, with zero duplicate diagnostic package
listings. Removing those duplicate listings did not materially reduce wall time:
the earlier run averaged 1650.86 ms while overlapping listings with gopls work.
The expensive diagnostic wait remains the next investigation.

Historical traced cohorts and this untraced run have different sample counts;
the phase counts identify removed work without establishing a controlled speedup.
`make check`, focused race tests, and CLI contracts pass, including virtual files,
subdirectory batches, source membership changes, fresh-coordinate rename retry,
and preservation of external edits after partial publication.

[Measurements, per-call metrics, source hashes, and provenance](assets/mcp-go-reuse-costs-20261015.json).

## Cache review and individual-call attribution (2026-10-15)

Independent review reproduced two false-clean cases: a previously unseen C input
was omitted from structural membership checks, and a cached dependency graph
could bypass effective Go environment validation. Both are fixed. Real-gopls
retained-versus-fresh regressions cover C input creation/removal, symlink
retargeting, and GOFLAGS/GOENV build-tag changes, including a passing race run.
ADR-0060 and its index now describe the corrected implementation.

Request metrics now separate transport completion (`gopls.send.<method>`) from
response waiting (`gopls.wait.<method>`). Native traces additionally separate
atomic creation, writing, fsync, close, final validation, and rename, plus scope
fingerprinting and buffer input binding. No request parameters or source content
are recorded by these phases. Enclosing phases remain inclusive.

Before filtering unrelated leaf metadata, ten untraced observations per scenario,
with preparation and correctness checks
outside timing, produced the following means:

| Operation | Mean wall time | Diagnostic RPC elapsed | Diagnostic pulls |
| --- | ---: | ---: | ---: |
| Root unused-import removal, 20 packages | 1608.54 ms | 1522.40 ms | 40 |
| Leaf unused-import removal, 20 packages | 199.19 ms | 116.60 ms | 40 |
| Exported rename, 20 changed files | 136.66 ms | 4.10 ms | 20 |
| Body edit in fan-out root | 72.63 ms | 9.70 ms | 40 |
| Repository telemetry import probe | 4807.17 ms | 3355.40 ms | 814 |

The repository corpus is committed revision
`1ff9655e5719c2d078c4e274e4a4817f006cced9`, exported to a disposable directory
and cloned using macOS copy-on-write. Its existing 28 findings are preserved by
exact diagnostic comparison against a full verification baseline. Full
verification runs outside timing after each operation; the next dirty source
is injected afterward. External lint hooks are explicitly disabled in the
fixture. Three Go module processes remain retained; the synthetic fixtures each
retain one. Every sample and final continuity check rejects restarts. The initial
summary helper incorrectly printed one start for the multi-module case; its
callers now supply measured start totals. The per-module log and continuity
checks were correct during the ten-observation run.

The synthetic native trace retains three measured calls per scenario. Root
imports have 19 slow dependent-file requests per capture: median request time is
59.03 ms before mutation and 22.81 ms after. Leaf imports make the same number
of requests, but median request time is about 0.19 ms, with expensive work
concentrated in the changed package. Transport time is below one aggregate
millisecond per synthetic operation at metrics resolution. This identifies a
sequence of expensive pulls, rather than one long first request.

Rename publication averages 82.10 ms in fsync, 1.62 ms in final conflict
validation, and 3.71 ms in renames. The current atomic durability policy dominates
that portion of the operation. The remaining package listings average 24.10 ms
per rename and 39.00 ms across the two body-edit captures. Source snapshots and
declaration scans do not account for the major remaining delay.

The CPU profile of our process contains 1.06 sampled CPU-seconds during a
23.69-second synthetic run, including setup, calibration, verification, and
tracing. Child-process CPU is excluded. Native scheduler Running intervals must
not be treated as CPU samples: some file sync syscalls remain inside such an
interval while blocking in the operating system.

[Observations, ordered request durations, disk phases, and provenance](assets/mcp-go-critical-path-20261015.json).

The repository trace exposes seven scope fingerprints per operation, averaging
975.40 ms in aggregate. Its whole-run CPU profile contains 15.22 sampled
CPU-seconds over 53.84 seconds; `scopeFingerprint` accounts for 8.19 cumulative
CPU-seconds, including setup and the full verification oracle. Of 407 pulls per
capture, the median takes about 0.75 ms. Roughly eleven per capture exceed five
milliseconds, with a maximum of 259.62 ms. Response waiting totals about 3.26
seconds per operation. This separates the RPC bottleneck from substantial local
fingerprinting work.

The fingerprint walker previously requested metadata for every non-directory
before testing whether it contributed to analysis. Pathname-based selection now
runs first. Compiler, manifest, embedded, and known auxiliary inputs retain
metadata, symlink-target and content validation; unrelated leaves are skipped.
A focused independent review checked the selector's metadata independence and
identified the intentional behavior change: failures from metadata requests on
irrelevant leaves are no longer observable because those requests are omitted.
Directory-walk and selected-input failures remain explicit.

A subsequent untraced ten-observation run with this filter reports:

| Operation | Mean wall time | Diagnostic RPC elapsed | Scope fingerprint elapsed |
| --- | ---: | ---: | ---: |
| Root unused-import removal | 1584.85 ms | 1499.90 ms | 4.50 ms |
| Leaf unused-import removal | 194.01 ms | 111.10 ms | 4.00 ms |
| Exported rename, 20 changed files | 131.60 ms | 4.00 ms | 3.80 ms |
| Body edit in fan-out root | 70.42 ms | 9.60 ms | 2.10 ms |
| Repository telemetry import probe | 4580.16 ms | 3309.70 ms | 882.80 ms |

These sequential runs show a modest repository improvement, with expensive
diagnostics and seven repeated fingerprints still present. The filter does not
solve either remaining bottleneck. The updated report helper prints the actual
three retained repository module processes; no restarts occurred. Full checks
and the real-gopls invalidation regressions remain the correctness gate.

The next substantial opportunities are reducing repeated structural scans and
the diagnostic pull workload without weakening freshness, and removing repeated
package listing for content-only edits where its findings remain valid. The
trace and existing metadata identify the affected package closure; any narrower
diagnostic strategy still needs equivalence checks against full verification.

## Background analysis and duplicate parsing investigation (2026-10-15)

The requested Gemini technical review has not completed. The configured bridge
allows only another repository's synthetic test root. After the user explicitly
approved sharing relevant source and detailed measurements, a separate bridge
used its normal read-only Seatbelt policy with a bounded source bundle. It
returned a conversation ID and an empty response. A direct CLI retry under the
same policy exposed `SecPolicyCreateSSL error: 0` during Gemini eligibility
checking, before any model turn or token usage. Sharing approval is resolved;
transport/runtime failure remains. No external technical review is claimed.
The findings below come from local source inspection and controlled real-gopls
experiments.

The installed binary's debug paths and call sites identify the local source in
`~/go/pkg/mod/golang.org/x/tools/gopls@v0.23.0`. Its module requirements match the
devel binary's x/tools dependency; this is not an assumption about an unmodified
release. `server.Diagnostic` calls `golang.DiagnoseFile`, which obtains package
diagnostics and calls `Snapshot.Analyze`. Enabling `pullDiagnostics` advertises
the pull provider; it does not stop background push diagnostics. File changes,
saves, and watched-file events still enter `didModifyFiles`, clone snapshots,
and schedule background diagnosis. Our client sends all three for saved Go
content changes, but consumes explicit pull results rather than those pushes.

A temporary local LSP transport filter compared notification policies against
the same real gopls. It retained full-content changes, create/delete events,
non-Go watcher events, and all explicit diagnostic requests. The production
policy has not been changed. Three measured root-import calls per policy gave:

| Experimental policy | Mean time |
| --- | ---: |
| Forward all events | 1728.16 ms |
| Omit watcher type-2 changes for open Go documents | 1600.59 ms |
| Omit saves | 1700.58 ms |
| Omit saves and those watcher changes | 1610.87 ms |
| Same omissions, with `diagnosticsTrigger: "Save"` | 206.74 ms |

The last policy suppresses automatic diagnosis on our content-change events.
It does not disable analyzers or reduce the explicit diagnostic scope. Initial
load and membership events may still trigger background diagnostics. Omitting
redundant notifications alone does not account for the large improvement.

A separate five-observation root confirmation averaged 1695.35 ms for the
control and 212.78 ms for the candidate, with forty pulls per operation in both.
On the archived repository, three observations averaged 4574.36 ms versus
2409.50 ms; diagnostic RPC elapsed fell from 3301.00 ms to 1138.67 ms, with all
814 pulls retained. Full verification outside timing preserved the repository's
28 existing findings, and each module retained one process.

The proxy additionally hashed every complete per-file diagnostic item set,
normalizing only module-root paths and ordering. Identical open-document
content states produced identical sets across policies: forty comparisons for
the synthetic fixture and 839 across the repository's three modules, including
both dirty and clean states. No diagnostic fields were removed from those item
comparisons. This is stronger than comparing only remaining diagnostic counts.

A further control/candidate pass checked all 6,088 diagnostic responses for
variation within an identical content state: none varied. The complete item
sets again matched across all 879 file-state comparisons between policies.
This pass used a separate consistency-checking proxy revision, retained
alongside the proxy revision used for the timed confirmations.

An independent import experiment, with five observations each and the existing
synchronization policy, averaged 1572.79 ms for unused `fmt`, 1390.25 ms for
unused `errors`, and 851.70 ms for unused `unsafe`. Every call restored the
original source, captured a clean final inventory, and reported resolved
diagnostics for the dirty input. Dependency size contributes to the cost, but
does not by itself explain the background-analysis overhead.

Two new timing children separate scope embed scanning from structural hashing.
The repository candidate still spends 879.67 ms in seven fingerprints:
380.33 ms in `go.scope.embed_patterns` and 499.00 ms in `go.scope.hash_inputs`.
`scopeEmbedPatterns` reads and fully parses the selected Go files; the hashing
walk then reads and fully parses those same source files again. Seven is the
sum of five manager uses (before capture, synchronization of three retained
modules, after capture), plus two fresh-scope fingerprints after cache misses.

Package metadata inspection also found 39 generated test-driver files outside
their declared module directory. The manager currently includes them in its
document set because it filters package module ownership but not each Go file's
location. The archived main-module process opens 446 documents while the
diagnostic selection requests 407. These generated cache files are additional
read/parse/open work; their independent latency contribution has not yet been
measured.

Before adopting the candidate synchronization policy, prove saved/unsaved state,
immediate explicit verification, external edits with preserved timestamps,
file membership, build constraints, cgo/embed inputs, local replacements, and
virtual/staged batch behavior through the existing real-gopls and CLI contracts.
The `Save` setting has no manual mode and still reacts to saves and membership
events, so changing initialization alone is insufficient. Client save capability
negotiation also needs to match the selected notification policy.

The next local optimization is a coherent content/AST observation reused within
one fingerprint, with no unchecked cache shared across independent manager
uses. It must preserve missing versus empty files, invalid-source fallback,
overlay deletion, directory membership, symlink retargets, and fresh reads on
the next observation. Generated-file filtering should use the original path's
module ownership before canonicalizing valid source symlinks.

[Experiment observations, proxy source, full diagnostic comparisons, source hashes, and provenance](assets/mcp-go-push-pull-investigation-20261015.json).

## Adopted pull-oriented notification policy (2026-10-16)

The user supplied a further Gemini source review after the automated consultation
failed. Its initial claim that only saves mark overlays saved was incorrect:
`overlayFS.updateOverlays` also compares ordinary content changes with disk.
Tracing `memoizedFS.ReadFile` confirms that ordinary atomic replacements change
disk identity and that preserved-inode/mtime external edits can reuse a stale
disk hash. The revised review identified the buffered-publication transition:
after an overlay is flushed, unchanged content still needs a notification.

ADR-0060 now adopts `diagnosticsTrigger: "Save"` with explicit diagnostic pulls.
The implementation sends full-content, advancing-version changes for content
updates and for unpublished-to-published transitions even when the digest is
unchanged. Ordinary open-Go-file saves and type-2 watcher notifications are
omitted. Creation/deletion and non-Go input notifications remain intact.

A targeted save fallback handles changed or published bytes whose disk identity
and modification time equal the prior synchronized observation. This preserves
saved-overlay semantics when gopls's disk hash may be stale. It does not evict
gopls's underlying disk cache. File metadata only selects this fallback; source
digests and existing publication checks continue to establish freshness.
Missing disk files are accepted only for unpublished buffered creations;
unexpected stat errors and notification failures discard the affected session.

Real-gopls regressions compare retained diagnostics against fresh full results
for ordinary atomic edits and preserved-mtime external writes, including error
introduction and repair. MCP regressions cover batch and stage publication,
buffered scaffolding, abort, subsequent cross-file rename, and package reload.
They require clean final diagnostic reports and retained processes after reload.
The CLI txtar contract additionally checks staged body publication followed by
batched scaffold publication, rename, and verification. The protocol fixture
requires distinct staged and published content updates without ordinary saves.

Independent implementation review found no blocking regression. It confirmed
that partial synchronization failures discard the session and that digest-based
freshness remains separate from notification-policy metadata.

A fresh native MCP run without the proxy measured ten calls per scenario:

| Scenario | Previous mean | Adopted-policy mean | Adopted-policy p99 |
| --- | ---: | ---: | ---: |
| Root imports, twenty files/packages | 1584.85 ms | 200.28 ms | 214.13 ms |
| Leaf imports, twenty files/packages | 194.01 ms | 183.69 ms | 192.27 ms |
| Exported API rename | 131.60 ms | 126.19 ms | 130.72 ms |
| Body edit | 70.42 ms | 69.97 ms | 98.66 ms |
| Repository imports | 4580.16 ms | 2423.75 ms | 2757.70 ms |

The previous native cohorts are historical observations on the same machine and
corpus, retained with their own source hashes. Root imports improve by 87.4%
and repository imports by 47.1%. Root and repository calls retain forty and 814
explicit diagnostic pulls respectively. Session-reuse and final diagnostic
oracles passed; the repository retains its 28 baseline findings. All selected
race regressions passed. Repository imports remain above the one-second goal;
fingerprint reuse remains a separate optimization.

[Native observations, phase attribution, source hashes, and provenance](assets/mcp-go-notification-policy-20261016.json).

## Fingerprint structural encoding reuse (2026-10-16)

The embed scan and hashing walk previously fully parsed the same Go sources.
Each fingerprint now owns a map from path to exact content digest, parse-validity
flag, and compact structural byte encoding. The initial pass parses once and
records package/import headers, directive comments, and cgo preambles in the
existing byte order. The hashing walk preserves fresh reads and every filesystem
check. Matching bytes reuse the encoding; changed bytes reparse. Invalid source
always hashes the current complete bytes. No ASTs or source buffers are retained
in the map, and no entry survives the fingerprint invocation.

This deliberately preserves repeated reads and the existing observation boundaries.
It does not introduce timestamp-based freshness or claim an atomic filesystem
snapshot. Independent review confirmed serialization equivalence and complete
migration of disk, symlink, and overlay callers.

Real-gopls regressions cover valid embed-pattern changes, selected asset changes,
added/removed matches, malformed-source changes, and repair, comparing retained
findings with fresh results. Existing preserved-mtime, compiler-input membership,
symlink, and MCP publication cases continue to exercise the shared path.

One attempted missing-pattern test exposed an existing diagnostic discrepancy:
retained incremental discovery reports both a package-level and located gopls
finding for `pattern missing/*.dat: no matching files found`, while fresh full
diagnostics report only the located finding. Retained discovery requests
`packages.NeedEmbedFiles`; fresh file selection requests `NeedName|NeedFiles`.
The compact encoding does not alter that diagnostic merge. This reporting issue
remains open; the valid embed and malformed-source regressions do not normalize
or discard findings to conceal it.

Ten native calls per scenario, using the same archived corpus and no transport
proxy, measured repository imports at 2244.14 ms mean and 2611.67 ms p99,
compared with the previous notification-policy mean of 2423.75 ms. The seven
fingerprints fell from 884.3 to 575.3 ms (34.9%); the hashing phase fell from
500.3 to 169.7 ms. Initial parsing/encoding and embed collection cost 405.1 ms.
The 814 gopls pulls remain intact and cost 1257.0 ms in this run. End-to-end
improvement is 7.4%; small-fixture means are 211.92 ms for root imports,
191.46 ms for leaf imports, 128.98 ms for rename, and 72.40 ms for body edits,
slightly higher than the prior cohort. The shared optimization targets the large
fingerprint cost; these runs do not show an improvement for every scenario.

Session-reuse and final diagnostic oracles passed, including the repository's
28 existing findings. Selected real-gopls freshness, environment, symlink,
compiler-input, and MCP publication tests passed under the race detector.
Repository latency remains above one second. Repeated observations and gopls
request granularity remain inspection points; this change shares no unchecked
cached input across them.

[Native observations, phase attribution, source hashes, and provenance](assets/mcp-go-fingerprint-reuse-20261016.json).

## Lazy buffered synchronization (2026-10-16)

Mutation observers previously synchronized retained sessions after each AST child,
even while edits lived only in source buffers. The Go observer now defers in a
buffered context. The retained-session query boundary still refreshes working
bytes before diagnostics or rename. AST-only batches send one content update
per changed document after publication;
if an earlier query opened an unsaved overlay, the existing same-content
dirty-to-published change notification remains necessary. Private stage start,
drop/replay, and abort skip the outer disk observer; commit retains it. Replay
queries consume the replayed buffer rather than an earlier staged revision.

Real-gopls tests require zero session synchronization or LSP traffic after
AST-only staged edits, one content change for a three-edit AST-only batch,
rename visibility for an unpublished declaration, and correct rename lookup
after dropping a previous staged rename. They also assert unchanged disk bytes
until commit, clean final diagnostics, and process continuity. The CLI txtar
contract checks the AST-only child's metrics and the final renamed source.
Selected publication, staging, shared-buffer, and failure tests pass under the
race detector. Protocol freshness and partial-publication invalidation remain
at their existing boundaries.

The existing native warm benchmark ran twenty measured iterations per scenario
before and after the change. The stage fixture has twenty Go files/packages;
the two-file batch adds one file, giving twenty-one Go files in twenty packages.

| Component | Before p50 | After p50 | Before p99 | After p99 |
| --- | ---: | ---: | ---: | ---: |
| Two-file batch | 90.35 ms | 83.65 ms | 109.39 ms | 111.18 ms |
| Staged AST edit | 3.46 ms | 1.08 ms | 4.68 ms | 1.79 ms |
| Stage commit | 54.56 ms | 53.48 ms | 70.46 ms | 60.90 ms |
| Stage/edit/commit total | 91.81 ms | 88.16 ms | 111.87 ms | 101.94 ms |

All cohorts retain one gopls process. Staged-edit p99 improves by 61.6%; total
stage/edit/commit p99 improves by 8.9%. Batch median improves, while its p99
shows no gain. Each policy uses one independent cohort with aggregate timing.

[Native observations, commands, source hashes, and provenance](assets/mcp-go-lazy-overlay-sync-20261016.json).

## Buffering scope and lazy preimages: accepted next boundary (2026-10-16)

At this investigation stage, source inspection distinguished the implemented
lazy LSP notification policy from a remaining source-buffer cost. `sourcebuf.New` in
`internal/sourcebuf/buffer.go` enumerates the workspace `.go` and `.java` source
set and loads every file. `sourcebuf.Prepare` constructs that buffer for a
standalone AST operation when no caller-owned buffer exists. `Buffer.Apply`
calls `assertDisk`; publication calls it before and between per-file
replacements. Each assertion re-walks observed source trees and re-reads all
captured file preimages. For K outputs and M captured sources, publication
alone can therefore read O(K × M) files, in addition to construction and child
application. This is a source-derived cost model, not a new timing measurement.

ADR-0032 now accepts the following boundary:

| Context | Required approach | Implementation status |
| --- | --- | --- |
| Interactive stage, drop/replay, abort, commit | Reversible in-memory buffer until publication | Retained staging implemented. |
| Multi-file AST plans requiring call-site or overlay typecheck validation: move, signature change, type migration, contract implementation | Buffer the complete plan; preparation failures publish nothing | Preserve this preflight boundary during optimization. |
| Overlapping or coupled batch steps | Share a buffer and normalize/flush once per changed file per segment | Shared buffering implemented. |
| Standalone single-file Go AST edit | Format in process, then target-scoped `atomicfile.WriteAtomicChecked` without a workspace-wide buffer | Migration outstanding; current owned buffer still captures the workspace. |
| Standalone single-file Java CST edit | Target-file buffer through JDT LS formatting, imports, and push verification, then publish and save | Lazy preimages outstanding; preserve required pre-flush LSP requests. |
| Standalone gopls rename | Stage the complete `WorkspaceEdit`, validate output preimages, publish checked targets | Direct target-checked publication implemented. |

The lazy-preimage migration must bind the actual read/write set and explicitly
observed package trees, including source membership and auxiliary inputs used
by overlay package loading. It must preserve exact-content, identity, mode,
symlink, missing-file, and package-collision checks. Validate the complete
observed input set before publication and each output immediately before its
replacement; avoid repeatedly scanning unrelated workspace sources. This is
accepted implementation work, not a performance result from the lazy-sync
benchmark above.

Buffering also coalesces import normalization so an early AST child does not
strip imports used by a later child. Existing trace evidence measured 82.10 ms
in `fsync` for twenty changed rename files (about 4.1 ms/file). Coalescing repeated
edits to the same file saves repeated durable writes; it cannot collapse twenty
distinct output files into one atomic replacement or one `fsync`.

The notification policy follows gopls v0.23.0 source mechanics, now recorded in
ADR-0060. `overlayFS.updateOverlays` compares the incoming content hash with a
delegate disk handle on open/change. `memoizedFS` stats device/inode and mtime,
reads on a changed cache key, and does not retain freshly read files whose mtime
is less than two seconds old. Atomic publication before `didChange` yields
matching disk bytes and `overlay.saved = true`. Equal `{URI, Hash}` identities
make `fileWasSaved` return false even after a redundant save notification.
An external in-place rewrite preserving identity and an old mtime can instead
leave the delegate hash stale; the targeted save fallback covers that case
without substituting timestamps for semedit's content validation.

An actual pre-flush LSP query still needs unpublished bytes. Such an overlay is
unsaved, and `Snapshot.buildOverlays` materializes it into temporary Go-command
`-overlay` inputs. Publication must send a same-content change when
`!previous.saved && !sourcebuf.Dirty(ctx, path)` to let gopls observe the newly
matching disk file. AST-only steps avoid both the unnecessary unsaved state
and the second content update. These are version-specific mechanisms, verified
against `internal/cache/session.go:1049–1075`, `fs_memoized.go:61–120`, and
`snapshot.go:481–504,1992–2012`; they are not a portable LSP saved-state contract.

## Cross-backend synchronization and buffering (2026-10-16)

Lazy preimages are a shared source-buffer requirement for both Go and Java;
notification timing follows each backend's analysis and normalization model.

| Backend and editing engine | Diagnostics and normalization | Publication synchronization | Buffering implications |
| --- | --- | --- | --- |
| gopls + in-process `go/ast`, `gofmt`, `goimports` | Explicit `textDocument/diagnostic` pulls; Save-trigger scheduling avoids duplicate ordinary change-triggered background analysis. | Disk first for AST-only edits, then `didChange`; atomic publication lets gopls infer saved content. Retain queried-overlay post-flush repair and the preserved-metadata save fallback. | Defer AST-child notifications until a real LSP query; use buffers for staging, coupled steps, and multi-file preflight. |
| JDT LS + embedded `javacst` range splicing | LSP formatting/import organization and code-action refactorings; version-matched push diagnostics. | Send working bytes before LSP normalization, chained rename, and verification; publish final bytes, then `didSave`. | Single-file pipeline needs a target buffer; parameter-object caller rewrites and extract-then-rename require their complete plan buffered. |
| rust-analyzer + delegated LSP operations | Repository settings disable `checkOnSave`; this does not disable all diagnostics or make Go's pull policy applicable. | Preserve its own document/VFS lifecycle; do not copy gopls save elision or Java save semantics. | In-memory server views do not use Go's `Snapshot.buildOverlays`; cargo/build/cache work remains a separate cost. |

At this investigation stage Java's `prepareJavaRenameExecution`, `stageJavaCodeAction`, and
`stageAndVerifyJavaInterface` reached the same eager `sourcebuf.New`
capture as Go's `sourcebuf.Run`/`Prepare`. Both `.go` and `.java` enumeration
and full captured-preimage rereads must migrate to actual read/write sets,
plus explicitly observed trees used by semantic analysis. Go overlay package
loads still bind source membership, local replacements, manifests, and auxiliary
inputs. This accepted design requirement is implemented in the follow-up below;
a Java latency comparison remains separate work.

Java formatting and import organization run through JDT LS, not the embedded
CST parser. `verifyJavaCodeActionFile` performs formatting, import organization,
buffer publication, and `waitJavaVerifyDiagnostics` against the current document
version before disk flush. Kotlin's session and the text-LSP Bash adapter likewise
consume `WaitDiagnostics` push results. Their change-triggered diagnostics are
required; suppressing them can cause timeout instead of reducing duplicate work.

JDT LS has an explicit working-copy save lifecycle, unlike gopls's inferred
saved flag. The upstream lifecycle handler updates and validates working-copy
content on change; save refreshes the resource and recreates the working copy
([source](https://github.com/eclipse-jdtls/eclipse.jdt.ls/blob/main/org.eclipse.jdt.ls.core/src/org/eclipse/jdt/ls/core/internal/handlers/BaseDocumentLifeCycleHandler.java)).
The local Java adapter retains pending saves until disk bytes match its digest.
Save notification must remain; it is not a synchronous completion receipt for
all Eclipse build jobs. No claim is made that JDT LS or rust-analyzer performs
no disk I/O, only that Go's temporary `go list -overlay` cost is not a universal
LSP buffer cost. Backend-specific latency measurements remain separate work.

## Lazy source-buffer lifecycle implementation (2026-10-16)

The shared preimage migration is implemented. `New` and `AuthorizeWrites`
validate roots without directory traversal or content capture. Reads and writes
bind their actual files; explicit tree observations bind only caller-selected
extensions and retain their own recursion and exclusion policy. Generic and Java
observations require no Go executable. Explicit Go analysis observations retain
effective Go cache exclusions without the former `containsGoSources` probe.
Package-level symbol scans observe their own directory; recursive discovery and
Go compiler preflight explicitly select wider trees where required. Observed
trees conservatively capture selected source preimages for existing overlay
loaders; this is intentionally broader than a single target for those analyses.

The generic buffer no longer restricts reads/writes to Go and Java. Go overlay
consumers explicitly request `.go`, avoiding manifests and Java/CST bytes in
Go package overlays. Fork/replay and retained publication preserve each tree's
selection identity. Tests distinguish unobserved sibling changes, which do not
invalidate a target edit, from observed input content, identity, permission,
alias, missing-file, and membership conflicts, which remain failures.

Publication validates the entire observed read set before staging and after
staging. The second membership check excludes only owned temporary outputs.
Each replacement then checks its target's canonical path, identity, mode, and
exact preimage. Removing the full-read-set assertion from the output loop
changes that component from O(K × M) captured-file reads to O(M + K), for K
outputs and M captured inputs. Explicit tree enumeration remains at observation
and full validation boundaries. Writes remain atomic per file, not across files;
a late I/O or conflict failure still reports partial publication.

MCP records changed paths before `Flush`/`PublishRetained` resets dirty state.
Go publication synchronization selects Go compiler/module inputs, and known
non-Go request targets skip the outer Go observer. Uncertain targets retain
conservative routing; every actual Go query still refreshes full input freshness,
including embed assets. Java and other backend notification policies are unchanged.

The real-gopls regression warms a clean session, queries dirty buffered source,
publishes identical working bytes, and requires a versioned post-flush change
without a redundant save. Fixture-local tool wrappers distinguish gopls child
Go invocations from host discovery, so an actual package reload can test temporary
overlay use rather than inferring server state from semedit's document map.
This section records implementation and verification scope; prior timing tables
retain their original source revisions.

## ADR invariant coverage audit (2026-10-16)

The handover's eager-buffer production changes are already implemented. Coverage
is extended at the observable boundaries rather than adding a public buffer
introspection API or duplicating private session-state assertions.

| Decision | Regression boundary |
| --- | --- |
| ADR-0004 | `TestManagedDiagnosticsSequentialEditsReuseBaselineAndClosure` compares each retained capture with a fresh full capture, requires zero pulls for unchanged baselines, and checks changed-package and reverse-dependency pull counts. A provider type change creates an error in a transitive consumer; an unrelated cached finding must survive edits and repairs. |
| ADR-0032 | `TestLazyCaptureIgnoresUntouchedMixedTreeAndSymlinks` constructs a buffer beside mixed-language files and broken aliases, reads a target changed after construction, and publishes Rust, shell, and Makefile bytes while leaving unrelated external edits intact. `TestLazyCaptureDetectsTargetChangesAfterFirstRead` requires a conflict after an accessed target changes. |
| ADR-0060 | Existing `TestPublishedOverlayFlushSynchronizesFromDisk` exercises a real retained gopls session, a dirty query, identical-byte publication, the required post-flush change without a save, and an actual subsequent package reload without a temporary overlay argument. |
| ADR-0068 | `TestApplyCodeActionKeepsJavaCSTEditInOverlayUntilFormattedSave` uses the public code-action entry point with an instrumented session to check pre-publication normalization, exact-version push diagnostics, and a mandatory save after final disk publication. It checks client behavior, not Eclipse internals. |

`mcp_staged_diagnostic_delta.txtar` and `mcp_batch_introduced_go_error.txtar`
exercise the CLI's MCP transport. Each applies a valid helper insertion followed
by a parseable compiler-breaking edit, checks `applied_with_diagnostics` and
pageable report metadata, and compares the final file including both edits.
Invalid Go syntax remains rejected before publication; the non-rollback contract
does not require accepting a syntactically invalid replacement body.

## Scoped diagnostic refresh after import changes (2026-10-16)

Automatic operation captures already enabled incremental diagnostics. The full
pulls came from `prepareSession`: any structural fingerprint change, including
ordinary import cleanup, discarded the complete diagnostic inventory and its
input digests before the incremental selector could reuse them.

Discovery freshness and diagnostic-cache validity now have separate hashes from
the same existing input observation. The diagnostic boundary excludes ordinary
Go import lists while retaining package names, all compiler directives, cgo
imports/preambles, parse validity, source paths, symlink targets, module inputs,
and auxiliary/embedded bytes. Malformed source hashes its complete bytes. A
matching boundary plus identical source/module lists permits cache retention
while package discovery reloads. Current document digests then select every
changed package and its transitive reverse importers; unaffected findings remain
in the returned workspace inventory. The duplicate fingerprint inside buffered
fresh-scope loading is removed. Explicit verification remains full.

A benchmark that writes an unused import before each call has changed the true
pre-edit state: its baseline may require an affected-closure refresh as well as
the post-edit capture. A heavily imported root package can legitimately have a
large reverse closure. These cases must remain visible instead of claiming zero
pre-edit work or limiting analysis to the selected file alone.

Real-gopls regressions compare retained and fresh full captures for import
addition/removal, transitive type errors and repair, unchanged baselines,
unrelated cached findings, simultaneous external sibling changes, and embedded
input invalidation. The MCP regression proves a warmed no-op makes zero pulls,
leaf cleanup makes two pulls (one before and one after), and explicit verification
still requests all twenty files. The CLI contract checks resolved findings and
retention of an unrelated error. Canonical checks and focused race tests pass;
independent review found no blocking cache-boundary or lifetime issue.

Ten measured warm calls per scenario produced:

| Scenario | Historical mean | Current mean | Current p99 | Diagnostic pulls, historical → current |
| --- | ---: | ---: | ---: | ---: |
| Root imports, twenty files/packages | 211.92 ms | 245.10 ms | 265.17 ms | 40 → 40 |
| Leaf imports, twenty files/packages | 191.46 ms | 206.45 ms | 214.71 ms | 40 → 2 |
| Repository telemetry imports | 2244.14 ms | 2189.90 ms | 2518.48 ms | 814 → 646 |

The root package's reverse closure covers the full small corpus. Leaf pull count
falls by 95%, but those two pulls still take 82.6 ms and the complete operation
does not improve against the historical cohort. Repository mean improves by
2.4%, with 20.6% fewer diagnostic pulls. It remains above one second: diagnostic
pulls cost 943.0 ms, seven fingerprints cost 599.9 ms, and package scope/diagnostic
metadata loads cost 278.3/213.7 ms. Parent phase totals overlap these components
and must not be added again. The historical cohorts predate lazy source-buffer
migration and retain their own source hashes; they are not randomized controls.
Final oracles preserve the repository's 28 baseline findings and its three
retained module sessions.

[Observations, phase attribution, source hashes, and verification provenance](assets/mcp-go-scoped-import-refresh-20261016.json).

## Follow-up optimization investigation (2026-10-16)

The baseline matrix was started with 100 cold and 1,000 warm calls per case,
scheduled sequentially on the twenty-file and archived repository corpora.
The twenty-file matrix completed; at the user's request, the repository phase
and queued candidate checks/measurements were stopped before starting. Candidate
source changes remain isolated in `.scratch/performance-candidate`. No candidate
improvement is established until correctness checks and comparative measurements
finish.

1. **Fresh discovery metadata:** the current incremental selector can prefer
   cached package findings after `loadScope` has already reloaded imports, then
   call `loadGoFiles` again. The candidate carries freshness only on the current
   `Session` wrapper, selects current discovery findings, and commits them only
   after successful diagnostic assembly. Explicit patterns and cache-hit body
   edits retain their existing paths. Discovery and diagnostic loads differ in
   dependency expansion and buffered overlays, so missing imports, cycles,
   repairs, subtree scope, and unrelated findings require exact fresh-oracle parity.
2. **Repeated fingerprint parsing:** a second candidate memoizes only compact
   parse-derived structural/boundary encodings, validity, and embed patterns,
   keyed by the current source path and newly read content hash. Each fingerprint
   still walks membership, reads working bytes, resolves symlinks, and checks
   configuration, manifests, compiler inputs, and embed contents. Memo entries
   are staged and committed/pruned only after successful fingerprint completion,
   under the manager lock. Patterns bind to the current source directory; no AST,
   source contents, stat-based freshness, or completed fingerprint is reused.
   `go.scope.parse` instrumentation distinguishes actual parsing from cache hits.
3. **Package-level pulls:** inspected gopls v0.23.0 advertises
   `InterFileDependencies: true` and `WorkspaceDiagnostics: false`.
   `server.Diagnostic` has outstanding related-document/result-ID support;
   `golang.DiagnoseFile` filters both package and analyzer findings to the
   requested URI. One representative-file pull cannot cover sibling findings.
   Analyzer results themselves use recipe-keyed caches and shared in-flight
   work, so invoking `Analyze` per pull does not prove complete reanalysis.
4. **Import-only closure pruning:** an unchanged declaration such as
   `var Value = api.Value()` can infer a different exported type after changing
   only the aliased import path. Import cycles and their repair also affect
   dependents without changing exported declaration text. Retain reverse closure
   unless semantic export-data and analyzer-fact equivalence can be established;
   import-list classification alone is insufficient.
5. **Bounded concurrency:** current pulls are serial; the JSON-RPC client has
   correlated concurrent requests and gopls marks diagnostic collection
   asynchronous. Compare one, two, and four workers with unchanged selected
   files, deterministic result assembly, cancellation/draining on failure, and
   cache publication only after success. Aggregate per-request wait durations
   overlap when concurrent and must not be treated as elapsed wall time.

Independent reviews approved the metadata integration and compact parse-memo
boundary, while rejecting representative-file pulls and general import-only
closure pruning. Candidate regressions use real gopls and fresh-result oracles;
they are prepared but have not passed canonical verification yet. The baseline
engine was preserved throughout the run; queued candidate checks did not start.

The isolated candidate also permits a successful default full capture to seed
the next incremental baseline, but only when its selected files and package
findings exactly cover the managed scope. Explicit-pattern and buffered captures
do not seed that baseline. Managed package loads now receive the same working
source overlays as gopls; standalone disk-only collection retains disk metadata.

Import transitions involving metadata errors conservatively discard diagnostic
caches without replacing the gopls process. Both the old and new dependency
graphs are inspected, including imported packages: a broken graph may omit
dependency edges needed for a safe selective refresh. An unchanged errored
revision still permits reuse. A real-gopls candidate regression creates and
repairs cycles and missing imports, expects all three files (including an
unrelated package) to be pulled at each transition, checks fresh-result parity
and no reinitialization,
and expects zero pulls for the unchanged broken revision. This regression and
the other candidate changes remain unverified pending the sequential check run.

An independent source review found no blocking issue in the integrated
metadata-error fallback or full-capture seeding boundary. Runtime checks and
comparative measurements remain outstanding.

## Full twenty-file matrix and stop boundary (2026-10-16)

The baseline completed in 12,897.340 seconds (3 h 34 min 57 s), using 100 cold
or prerequisite-workflow samples and 1,000 warm samples per accepted case.
There are 33 source-edit cases, including the generated safe-delete variant,
and 17 read/lifecycle cases. Earlier progress messages counted only the 32
literal source-edit table entries. Across all 50 scenarios, 46 passed their
configured budgets, three exceeded them, and one lost process continuity.
The 49 qualified cases contain 53,900 measured calls. The abort case rejected
its first cold-workflow and warm samples and withholds both percentiles.

| Case | Cold p99 | Warm p99 | Outcome |
| --- | ---: | ---: | --- |
| Function extraction | 1377.246 ms | 1172.781 ms | Exceeds the 1 s warm ceiling |
| Declaration move, same package | 1071.150 ms | 571.396 ms | Warm budget passes |
| Import organization, source-edit fixture | 609.035 ms | 174.258 ms | Passes |
| Import organization, dirty fan-out workflow | 436.271 ms | 349.558 ms | Passes |
| Rename, local symbol fixture | 521.119 ms | 94.061 ms | Passes |
| Rename, exported fan-out workflow | 401.811 ms | 166.355 ms | Passes |
| File outline | 16.941 ms | 19.478 ms | Exceeds the 10 ms read target |
| References | 75.801 ms | 75.539 ms | Exceeds the 10 ms read target |
| Stage abort | Withheld | Withheld | Restarts retained gopls |

All 32 other source-edit warm cases pass the one-second ceiling. Six of the
eight read cases meet 10 ms; lookup, symbol inspection, construct inspection,
workspace outline, metrics, and captured diagnostic pages have warm p99 values
between 0.743 and 2.856 ms. Body replacement measures 107.161 ms in its isolated
source fixture and 127.991 ms in the fan-out workflow. Explicit verification
measures 58.950 ms. Snapshot creation is a mutating lifecycle operation and
measures 32.475 ms against its one-second budget.

The whole benchmark exits with status 1 because abort loses continuity.
Budget enforcement was disabled, so the three `BUDGET` rows do not themselves
cause that exit. The raw stream records an extra gopls start in both the abort
cold-workflow check (2 instead of at most 1) and first warm sample (4 total
instead of the 3 existing starts). Source inspection identifies the direct
mechanism: `stageManager.Abort` calls `godiagnostics.InvalidateWithContext`,
which discards every retained module session. The benchmark's post-abort
verification then starts a replacement process. Restoring the disk source view
without discarding a healthy process is the next lifecycle investigation.

The slow paths are not all gopls work. Go reference search creates a new typed
`go/packages` load over `./...` on every call (`loadReferencePackages` in
`internal/backend/golang/references.go`), independently of the retained gopls
session. Go function extraction loads compiler information for output-type
checks and again for the transformed candidate (`validateOutputTypeNames` and
`validateExtractionCandidate`); `loadExtractionPackages` can reload while binding
additional compiler inputs. These are inspection points, not measured phase
attribution for the current table. File outline is an in-process AST path;
its Go backend drops the request context, then source selection resolves Go
environment exclusions through a background context. Preserving the request
context and inspecting repeated `go env` setup is a concrete next check for
its 19.478 ms result.

The two import rows intentionally measure different preparation boundaries.
The source-edit fixture resets its file and explicitly verifies before each
warm call, synchronizing the dirty source view and warming gopls outside timing.
In the measured engine, explicit full verification does not populate the
automatic incremental diagnostic cache or advance its input digests. The timed
mutation can still collect its pre-edit baseline; full-capture seeding exists
only in the unmeasured candidate. The read/lifecycle workflow inserts the unused
import immediately before the timed call without that explicit pre-call
verification, so its synchronization also occurs inside timing. This difference
does not by itself attribute the entire timing gap or prove one versus two
diagnostic captures. Rename fixtures likewise differ: one is a local
symbol/caller pair; the other fans out across packages. These rows must remain
separate rather than being presented as a regression comparison.

The suite's wall time includes extensive work outside reported tool durations.
Each source-edit case has 100 fresh cold sessions and roughly 2,100 explicit
verification calls around its 1,100 measured calls, plus source resets and
session teardown. Across 33 cases that is roughly 105,600 tool calls before
the separate read/lifecycle matrix or operation-specific prerequisites.
The benchmark records summary percentiles, so an exact wall-time split between
measurement, preparation, and verification cannot be recovered from this log.
It also delays row output until the complete corresponding matrix, which
unnecessarily delayed feedback during this run.

Historical fan-out references measured rename at 137.021 ms p99 and imports
at 1826.381 ms p99 with 100 samples. The current 1,000-sample fan-out rows are
166.355 ms and 349.558 ms respectively. Imports are substantially faster in
these observations; rename is higher. Different revisions, sample counts,
and cohorts prevent assigning either change to one optimization. The isolated
metadata, parse-memo, full-capture reuse, and concurrency candidates were not
included in this baseline and remain unmeasured.

The repository corpus never started. The waiting validation and concurrency
jobs were cancelled, and the suspended launcher was terminated only after all
twenty-file benchmark children exited. No new measurements are scheduled.
This full run did not collect a CPU profile or per-phase attribution.

[Complete results, source hashes, and measurement boundaries](assets/mcp-20-file-full-matrix-20261016.json)
and [raw benchmark output](assets/mcp-20-file-full-matrix-20261016.log).

## Assessment of external source review (2026-10-16)

Gemini's read-only review agrees with the observed abort invalidation path,
fresh typed reference loads, extraction compiler loads, outline context loss,
and the two rejected diagnostic shortcuts. It also agrees that the isolated
candidate's cache commits and worker result publication are guarded. These
remain source-review findings rather than executed candidate verification.

Several recommendations need narrower correctness and measurement claims:

* Whole-map package-finding replacement is deliberate invalidation when a fresh
  capture advances shared session state. Keeping sibling entries without their
  own validated revision can return stale findings. A subsequent cache miss
  with `packageFindingsDirty == false` uses existing scope findings; it does not
  automatically perform a new package load on every alternating subtree query.
* Standalone `CheckDetailedForPatterns` still passes a nil overlay to
  `loadGoFiles`; managed calls receive working overlays. The review's statement
  that standalone buffered behavior was changed is inaccurate.
* Ordinary stage abort should discard the staged view and synchronize current
  disk contents, including closing staged-only files, rather than restore old
  disk preimages over possible concurrent edits. Retaining the process must
  still preserve cache invalidation, membership, and build-input correctness.
  This is not a zero-risk one-line substitution.
* Original-code and transformed-candidate type information describe different
  source programs. Reusing the original `packages.Load` result cannot prove
  that the candidate compiles. A narrower inspection point is that
  `validateOutputTypeNames` loads a package before iterating its outputs even
  when there are none; the measured extraction fixture has no live-out results.
  Skipping that work still requires proving any original-input validation
  contract is preserved, alongside transformed-candidate validation.
* Propagating outline's context is justified, but cross-request environment
  caching needs environment/configuration freshness checks. A sub-5 ms outline
  result or a 50% extraction speedup has not been measured.
* The existing smoke guard accepts at most two samples per cohort and requires
  `TOOL_SMOKE=true`. Proposed 2/10 and 5/20 commands are rejected by the current
  runner. The valid small correctness mode is explicitly marked smoke and is
  separate from the retained 100/1,000 acceptance runs.

Whole-package deletion and direct external-test dependency changes are useful
additional regression scenarios, not demonstrated false-clean defects. The
candidate already clears caches whenever selected file membership changes,
and current tests cover file removal and external test-package presence.
More focused assertions would strengthen coverage of those exact scenarios.

No code, checks, or measurements were started for this assessment. The requested
stop boundary remains in effect.

## Repair implementation and focused qualification (2026-10-16)

The subsequent repair request authorized focused implementation and measurement,
without restarting the full matrix or the repository corpus.

### Accepted repairs

* Ordinary Go stage abort and expiry now reconcile retained documents against
  current disk contents instead of invalidating the process. Staged-only
  documents close, external disk edits survive, and a scoped reservation blocks
  concurrent stage transitions until reconciliation completes. Failed recovery
  discards the affected session rather than retaining an abandoned overlay.
  This recovery is specific to the managed Go session; other backend publication
  and save lifecycles remain separate.
* Outline propagates request cancellation and the working source buffer. Shared
  directory exclusion resolution skips `go env` only when the process already
  supplies complete, absolute runtime directory settings and neither a managed
  base nor a module seed overrides them. Partial, relative, GOENV-derived, and
  managed settings retain normal resolution. There is no global environment
  cache that could conceal a later configuration change.
* The isolated diagnostic candidates are validated and promoted: exact-digest
  structural parse memoisation, fresh discovery metadata reuse, exact default
  full-capture baseline seeding, and all-or-nothing diagnostic collector cache
  publication. Buffered/pattern-restricted captures cannot seed a disk baseline.
  The collector remains at one worker. Whole-map package-finding invalidation
  remains deliberate; sibling findings are not retained without freshness proof.
* Import-cycle creation and repair exposed stale missing-metadata diagnostics
  inside gopls v0.23.0, reproducible with raw pulls and without semedit caches.
  Save notifications, watcher notifications, and document close/open did not
  clear them. A workspace-folder remove/add pair creates a fresh view in the same
  process and clears the repaired-cycle error while retaining open documents.
  Recovery now uses that negotiated capability only on discovery transitions
  involving metadata errors. Unsupported servers fall back to replacement on
  this exceptional transition; normal body/import edits do not reload the view.
* Function extraction no longer requests returned syntax and expression-type
  maps that it does not consume. Constant/variable extraction retains them.
  Original-program and transformed-candidate compiler checks remain distinct,
  with captured compiler inputs revalidated before publication. Even outputless
  extraction needs the original check: a generated helper could otherwise make
  an unresolved call in another file compile, concealing invalid input.

### Correctness evidence

The integrated `make check` passes, including executable CLI contracts for
stage-query-abort continuity and import-path reverse-dependency diagnostics.
Focused race checks pass for abort/publication, source exclusions/environment
resolution, and original extraction compilation. The diagnostic candidate's
metadata/import-cycle regressions pass five repeated race-enabled runs.
The raw-server experiment establishes the stale-cycle mechanism separately from
the regression checks; the workspace-view reload is not inferred solely from a
passing cache test.

### Measurement boundary and remaining work

Focused qualifications use the same twenty-package, exactly twenty-Go-file
corpus with 100 fresh-session and 1,000 warm samples per case. Initialization and
shared disk caches follow the existing harness boundary. Abort's fresh workflow
includes its required stage/edit preparation; its warm row times abort itself.
The diagnostic/source-edit runner still performs untimed fixture verification.
Only the selected cases are rerun; the earlier fifty-case matrix is unchanged.

The repaired binary SHA-256 is
`93898cb5ae8c9fd5af247f2907de4823cfdcb7b9d757ff105ba4e30ed8f9701e`.
The paired outline baseline uses the previously frozen full-matrix binary,
SHA-256
`035303e7dd26df0e136db5b9ae53abcd4a554430ab1f1edb666c935a09a59257`.
Both use Go 1.27.2 on Darwin arm64.

| Focused case | Cold p99 | Prepared workflow p99 | Warm p99 | Result |
| --- | ---: | ---: | ---: | --- |
| Outline, frozen baseline | 9.427 ms | N/A | 13.777 ms | Misses 10 ms read target |
| Outline, repaired | 2.385 ms | N/A | 3.267 ms | Pass |
| Stage abort, repaired | N/A | 944.507 ms | 4.998 ms | Pass; one retained process across 1,000 warm calls |
| References, repaired binary | 160.053 ms | N/A | 88.799 ms | Still misses 10 ms read target |
| Function extraction, repaired binary | 8,993.936 ms | N/A | 5,709.032 ms | Above 1 s target; contended run |

The outline observations differ by 76.3%. After extraction finished, the user
reported that another chat was running heavy tests using eight cores. Host load
was not recorded alongside samples, so these timings cannot establish an
uncontended improvement or regression. The 5.709 s extraction observation must
not be attributed to these changes by comparison with the older 1.173 s row.
No additional timing runs are started under that competing load. Correctness
checks and process-continuity assertions remain useful independent evidence.

The older full-matrix outline and reference rows are historical observations
rather than paired controls for this repair. References were not changed in
this repair, so their difference from the older matrix does not establish a
regression caused by these changes.

[Focused results, hashes, contention context, and raw-log links](assets/mcp-focused-repairs-20261016.json)
preserve the observations independently of the earlier full-matrix artifact.

References remain an independent read-path problem: each request performs a
module-wide typed `packages.Load` rather than consulting the retained language
server. A replacement must preserve exact symbol origin, cross-package results,
and test variants.

Extraction instrumentation separates original validation, candidate validation,
and each package load. Reducing requested return fields is not proof of reduced
type-checking CPU: the pinned x/tools loader still allocates internal type
information before clearing unrequested returned fields. Another inspection
point is repeated typed loading while binding compiler read sets. Metadata-only
discovery followed by typed validation could avoid that duplicate compiler
work, but must rebuild overlays after binding and retain both program checks.

## Individual ADR alignment audit (2026-10-16)

Reviewed all 70 active ADR source records individually, beginning with the ADR
index, plus the archive entry for revoked ADR-0023. The table records each
outcome; unrelated accepted decisions remain in force rather than being marked
obsolete merely because this work concerns performance.

The comparison uses the accepted retained-session, lazy-buffer, diagnostic
coverage, compact-output, and backend-owned synchronization contracts. It also
checks later implemented Java capabilities and opt-in Go isolation against older
capability exclusions. Source checks include `gocache.Environment`, backend
capability declarations, diagnostic-delta structs, and MCP presentation. The
audit clarifies existing decisions; it does not adopt experimental gopls
`MoveDeclaration`, broaden Go refactoring proofs, or establish new timing results.
Alignment is not proof that every backend meets the latency targets. Kotlin's
fresh source-isolated request sessions, for example, remain its accepted safety
boundary rather than inheriting Go's retained-session policy automatically.

Twenty-nine records required corrections or explicit scope/supersession notes.
Their ADR index rows change alongside the records. The remaining 41 active
records are aligned; ADR-0023 stays revoked with no invented rationale.

| ADR | Outcome | Individual assessment |
| --- | --- | --- |
| 0001 | Aligned | Semantic intent remains separate from local transformation. |
| 0002 | Aligned | Operation/language-specific analysis and refusal contracts remain authoritative. |
| 0003 | Aligned | Symbol addressing and ambiguity handling do not require repeated workspace scans. |
| 0004 | Aligned | Valid pre/post findings permit checked reuse; explicit verification and no forced rollback remain. |
| 0005 | Corrected | Preserve manifest safety; replace the superseded single-language session limit with enabled-backend scoping. |
| 0006 | Aligned | CLI/MCP delivery and skill maintenance remain required. |
| 0007 | Corrected | Canonical registered names replace hypothetical tools; bundled feedback permits validated reuse, deferred stages, and compact report pointers. |
| 0008 | Corrected | Planned ingress/fan-out does not promise implemented cross-domain edits or multi-file atomic publication. |
| 0009 | Corrected | Use current semantic tool names and enabled-backend scoping; discovery profiles do not imply transactions. |
| 0010 | Corrected | Remove watcher completion guarantees and mtime-only freshness; distinguish working results, checked publication, and backend-specific sync. |
| 0011 | Aligned | Declaration/import management retains atomic and manifest boundaries. |
| 0012 | Aligned | Access/placement support remains capability-specific. |
| 0013 | Aligned | Explicit typed errors and source locations remain compatible. |
| 0014 | Aligned | CLI parsing and stdin policy do not change analysis lifecycle. |
| 0015 | Aligned | Documentation presentation remains independent of semantic performance. |
| 0016 | Aligned | Body/scaffold/case/batch vocabulary does not mandate eager diagnostic work. |
| 0017 | Aligned | Explicit executable live reload is separate from ordinary source edits retaining gopls. |
| 0018 | Aligned | Explicit conflict-checked undo does not impose automatic rollback on mutations. |
| 0019 | Aligned | Typed anchors and ambiguity choices remain precise. |
| 0020 | Corrected | Go host implementation does not restrict approved JDT LS delegation to lookup. |
| 0021 | Corrected | Registry capabilities reflect later implemented extensions, not permanent lookup-only limits. |
| 0022 | Aligned | Generic transport remains protocol-neutral and mutation-free. |
| 0023 | Revoked | Source record absent; archive reserves its number and preserves unknown rationale. |
| 0024 | Clarified | Original Rust lookup slice is extended by selected-file rename and outline. |
| 0025 | Clarified | Later Java decisions supersede lookup-only, import, and overload exclusions. |
| 0026 | Clarified | Scala outline extends the initial lookup slice without enabling mutations/builds. |
| 0027 | Aligned | Haskell remains explicitly standalone trusted lookup only. |
| 0028 | Aligned | Each advertised backend/operation pair still needs CLI contract coverage. |
| 0029 | Aligned | Backend owns edit validation, publication, normalization, and diagnostics. |
| 0030 | Aligned | Rust selected-file rename remains deliberately bounded. |
| 0031 | Clarified | Java's single-file/reset-after-commit slice is superseded by warm workspace rename under ADR-0068. |
| 0032 | Corrected | Staged output is a working receipt; publication is atomic per file; step attribution is candidate-only; captures can reuse validated findings. |
| 0033 | Aligned | Assertion reporting is independent of buffering and diagnostics policy. |
| 0034 | Aligned | Typed registry contracts remain the authority for dispatch and capabilities. |
| 0035 | Corrected | Trusted-POM import default supersedes opt-in; retain reactor/no-build boundaries and fix the incorrect ADR number in the heading. |
| 0036 | Aligned | Bounded Java verify actions remain separate from general builds and other refactorings. |
| 0037 | Aligned | Explicit fixed Maven compile/test actions do not become implicit JDT LS behavior. |
| 0038 | Aligned | Benchmark treatment/provenance rules remain valid; no harness policy changed. |
| 0039 | Aligned | Ordered outcomes, durations, and explicit validity remain required. |
| 0040 | Aligned | Fixture/oracle isolation remains independent of internal tool timings. |
| 0041 | Corrected | Qualify all managed-path/environment claims by explicit isolation; ordinary workspace routing does not enable it. |
| 0042 | Aligned | Actionable text preserves guards without duplicating structured bulk; report pagination remains explicit. |
| 0043 | Aligned | Agent benchmark aggregation does not replace internal per-tool measurement. |
| 0044 | Aligned | Experiment dimensions and fixture hydration remain separate concerns. |
| 0045 | Aligned | Feedback stays a user-reviewed draft, without implicit submission. |
| 0046 | Aligned | Shared construct kinds advertise only implemented backend combinations. |
| 0047 | Aligned | Kotlin remains read-only; matching push receipts must not inherit gopls suppression. |
| 0048 | Clarified | Later outlines and Make inspection extend lookup; diagnostic/mutation bounds remain backend-specific. |
| 0049 | Clarified | Go bounded moves, Java movement, whole-package rename, and experimental declaration moves are distinct contracts. |
| 0050 | Aligned | Agent-session continuity and explicit fixture isolation remain valid; this audit does not rerun the harness. |
| 0051 | Clarified | Outline context/buffered reads match the repair; source immutability does not forbid backend cache I/O. |
| 0052 | Aligned | Opt-in metrics/traces attribute nested elapsed phases without claiming subprocess CPU. |
| 0053 | Corrected | Specialized body replacement uses each supported backend, including Java. |
| 0054 | Clarified | Explicit full verification differs from cached mutation feedback; pre-flush Java queries remain allowed. |
| 0055 | Corrected | Bulk utility metrics/pages remain structured once, consistent with compact text in ADR-0042. |
| 0056 | Corrected | Preserve both Go compiler validations while separating Java extraction from Go-only proof limits. |
| 0057 | Aligned | Safe-delete's explicit module read-set proof is necessary analysis, not eager buffer construction. |
| 0058 | Clarified | Go call-inlining exclusion does not disable separately implemented Java inlining. |
| 0059 | Corrected | Bound the Go variable-extraction proof without denying separate constant/Java operations. |
| 0060 | Corrected | Inherit environment unless isolation is explicit; executable freshness means gopls, not source files; retained LSP state differs from lazy operation buffers. |
| 0061 | Clarified | Go typed contract generation and Java delegation keep distinct proofs. |
| 0062 | Aligned | Final-source selector liveness and omitted buffered continuations remain required. |
| 0063 | Aligned | Go field extraction remains bounded and keeps methods on the original type. |
| 0064 | Clarified | Go private-function signature bounds do not constrain Java's distinct implementation. |
| 0065 | Corrected | Shared ingress does not impose Go-native formatting on Java or exact-splice operations. |
| 0066 | Clarified | Workspace orientation returns bulk once and preserves source/manifests without promising zero runtime/cache I/O. |
| 0067 | Aligned | Run-owned model relay lifecycle is separate from language-server ownership. |
| 0068 | Corrected | Replace blanket functional-parity claims with operation-specific Java support and backend-owned lifecycle. |
| 0069 | Aligned | Canonical module/executable paths do not affect analysis caching. |
| 0070 | Aligned | Inherited Go state and explicit benchmark isolation remain authoritative. |
| 0071 | Aligned | Immutable diagnostic pages read captured evidence without new analysis. |

All original decision dates and unaffected safety limits are preserved. No
benchmark is restarted for this audit; contention caveats on the preceding
measurements remain in force.

Verification passes: `make check`, fresh ADR/research index-atomicity tests,
`git diff --check`, and `make verify-docs`. The first site build could not resolve
the configured search-asset CDN; the successful retry reused existing local
Hugo cache entries for the same FlexSearch 0.8.143 dependency without changing
site configuration or downloading a replacement.

## Benchmark artifact traversal investigation (2026-10-16)

Two user-supplied filesystem traces captured substantial metadata churn during
concurrent checks. The second spans 20:39:20–20:39:30 local time and contains
7,527,547 filesystem records, including 6,055,374 attributed to `go` and
4,290,591 Go `lstat`/`stat` calls. At least 335,641 Go records visibly reference
benchmark results, archives, or fixture validation. Paths are sometimes
truncated, so those path counts are lower bounds. Record counts are not CPU
times and do not establish an FSEvents leak or attribute historical process
write totals.

The root cause includes exclusions applied after `go list ./...` discovery,
unbounded non-Go directory walkers, formatting that does not prune artifact
trees, and repository-corpus exports that include benchmark archives. A
user-approved `data/benchmarks/go.mod` boundary now stops native Go discovery at
the data tree. Shared discovery filters, buffered tree observations, snapshots,
verification staging, and benchmark copies prune the artifact boundary before
descent. Formatting applies the same exclusion explicitly. Repository benchmark
exports select source-tree entries before `git archive`, excluding artifact
data even for historical commits.

Documentation still consumes published results. Its loader is bounded to JSON
at `results[/run[/task]]`, pruning deeper captured workspaces and reserved
archive, validation, checkpoint, evidence, hidden, and dependency directories.
Malformed eligible reports continue to produce errors. This preserves all
tracked non-checkpoint result layouts found during the audit; there were no
deeper tracked result JSON files outside jobs/evidence trees.

Regression coverage exercises native Go discovery and both formatting passes
against an unreadable archive, documentation intake with malformed artifact
JSON, source-buffer publication despite concurrent archive changes,
verification normalization without staging artifacts, and CLI workspace
outline/snapshot behavior. Performance matrices remain paused; these changes
address unnecessary traversal rather than claiming a measured latency gain.
