<!-- Records quality-check resource measurements and the limits of each comparison. -->

# RQ-0020: End-to-End Quality Check Performance and Parallelization

* **Status**: Open
* **Date**: 2026-09-16
* **Category**: Build & Test Infrastructure

## 1. Question

How can the complete project quality check become faster without reducing coverage, determinism, or confidence? The scope includes formatting, dependency consistency, Go unit tests, integration and property-based tests, linting, vulnerability checks, documentation generation, and published-site assertions.

## 2. Context and Measurement Boundary

The top-level `make check` target is the project's quality gate:

```text
fmt -> tidy -> {lint, vuln, test, verify-docs}
```

Its runtime is a validation and CI-harness measurement. It is not a measure of `semedit` editing effectiveness. In agent benchmarks, a direct `semedit` invocation against a `txtar` fixture remains the control for host edit cost. The same validation procedure can then be applied to the direct control and the agent arm so the additional agent and LLM overhead is visible without attributing test-suite time to the semantic edit itself.

## 3. Baseline Measurement

The baseline was measured on macOS arm64 with Go 1.27.1, Hugo 0.166.0 Extended, warmed Go build caches, and the repository's current checks:

| Invocation | Wall-clock time | Result |
| :--- | ---: | :--- |
| `make -j1 check` (first run) | 14.69 s | Pass |
| `make -j1 check` (warmed repeat) | 15.05 s | Pass |
| `make -j4 check` | 13.20 s | Pass |
| `make -j8 check` | 13.39 s | Pass |

Four jobs reduced elapsed validation time by approximately 10% relative to the serial baseline. Eight jobs provided no further improvement. These historical runs predate the current dependency barrier, so they do not measure the current Make graph.

Hugo reported approximately 0.43 seconds of the run. Documentation rendering is therefore not the dominant cost; Go test packages, including the gopls-backed property test and integration scripts, account for most of the work.

### 2026-09-24 Hyperfine Measurement

Hyperfine 1.20.0 ran all seven commands in one invocation with three warmups and five measured runs per command. The host used macOS arm64, Go 1.27.1, and Hugo 0.166.0 Extended. Every timed command ran the full `make check` gate with race tests and the current `fmt -> tidy` dependency barrier. The Go test command uses `-shuffle=on`, so this measures repeated full validation on warmed caches rather than cached test output.

| Command | Median (s) | IQR (s) | Mean ± standard deviation (s) |
| :--- | ---: | ---: | ---: |
| `make check` | 42.510 | 1.103 | 43.356 ± 3.241 |
| `make -j2 check` | 36.328 | 6.038 | 36.489 ± 3.449 |
| `make -j3 check` | 28.812 | 4.004 | 30.152 ± 2.799 |
| `make -j4 check` | 30.745 | 2.628 | 29.981 ± 2.201 |
| `make -j6 check` | 30.615 | 1.337 | 30.331 ± 1.194 |
| `make -j8 check` | 28.900 | 1.387 | 28.267 ± 1.129 |
| `make -j16 check` | 27.192 | 1.507 | 28.336 ± 2.246 |

Eight jobs had the lowest mean, 34.8% below the serial mean. Sixteen jobs were effectively tied with eight and had more variation. These command labels reflect the Makefile at measurement time. The later `MAKE_JOBS ?= 8` setting makes bare `make check` use eight jobs; on GNU Make 3.81, select another count with `make MAKE_JOBS=N check`, including `MAKE_JOBS=1` for serial execution. Hyperfine reported statistical outliers. The commands ran in the listed order in an active, uncommitted worktree; this is a local warmed-workspace result, not a clean or quiet-system comparison. The historical baseline above used a different project state and cannot be directly compared with this measurement.

### 2026-10-16 MCP Test Setup

An uncached MCP package run passed in 142.214 seconds while `make check` ran
concurrently. Its Go CPU profile sampled 288.41 CPU-seconds in the test process.
`mcptest.compileSchema` accounted for 98.56 seconds cumulatively (34.17%) and
catalog discovery accounted for 95.66 seconds (33.17%). These stacks overlap;
the percentages must not be added. The profile excludes compiler and language
server child-process CPU and is not a quiet-system performance baseline.

Each schema-validating server eagerly probed the advertised and full catalogs,
compiling every output schema. A real `tools/list` response compiled those
schemas again. This repeated work also allocated heavily and increased garbage
collection pressure. Identical catalog checks and numeric-validation cases
created additional independent wrappers without needing independent server state.

The optimization keeps catalog selection local to each server and reuses compiled
schemas by exact content, with one concurrent compilation per distinct byte
sequence. Live catalog responses still validate their schemas, and hidden but
callable tools retain full-catalog fallback validation. Success, failure,
request-ID correlation, and integer-precision checks remain mandatory.

Independent review rejected deferred catalog discovery as unsafe. A first call
can replace a Go source file with Rust source before validation; a fresh probe
then omits the responding Go tool even though the actual server retains its
original language snapshot. A wire regression failed for both full and base
catalogs under that implementation. Capture both catalogs before mutation and
preserve the full fallback across live list responses; successful initialization
refreshes both snapshots, including when the selected root is unchanged.

The constructor's `profile` argument and the `WithToolProfile` option control
different filters. Reuse must not infer catalog equality from the constructor
argument alone. Shared compilation must not share mutable catalog maps across
roots, registries, enabled languages, or initialization and reload lifecycles.

Catalog-only checks share a captured response through named subtests. Numeric
validation retains separate direct and batch sessions and checks each response
by ID; mutation and process-lifecycle tests keep independent state. No global
test concurrency limit is introduced: limiting concurrency trades elapsed time
for lower peak demand and requires separate resource measurements.

Five interleaved before/after pairs measured the numeric-validation test using
fresh test-binary processes, warmed project-local Go caches, default concurrency,
and `-test.count=1`. The unchanged baseline binary and optimized binary ran from
`internal/mcp`; each before run immediately preceded its corresponding after run.
Every run passed. CPU values are `/usr/bin/time -p` accounting for the process
and waited-for children; they do not directly measure `kernel_task`.

| Numeric-validation measurement | Before median (IQR), seconds | After median (IQR), seconds |
| :--- | ---: | ---: |
| Wall time | 9.04 (0.57) | 2.51 (0.41) |
| User CPU | 3.90 (0.20) | 0.64 (0.07) |
| System CPU | 1.17 (0.14) | 0.64 (0.28) |
| User + system CPU per run | 5.05 (0.32) | 1.28 (0.38) |

This selected test reduced median elapsed time by about 72% and median combined
CPU by about 75%. These are local measurements of the combined optimizations,
not isolated attribution to each change or a whole-suite speedup estimate.
Other checkouts ran CPU-heavy tests on the same host, so the machine was not
otherwise idle. Raw paired results remain in `.scratch/mcp-paired-*.time` and
their matching `.log` files.

One exploratory full-suite pair used the same fresh-process command without
profiling. Both runs passed: before was 84.21 seconds wall, 70.48 user, and
353.83 system; after was 88.52 seconds wall, 26.99 user, and 140.60 system.
Combined CPU fell from 424.31 to 167.59 seconds, about 60%, while elapsed time
did not improve. These single samples were separated in time on the busy host;
they are evidence of reduced accounted CPU, not a stable latency comparison.
The raw files are `.scratch/mcp-before-full.*` and `.scratch/mcp-after-full.*`.

## 4. Parallelization Avenues

### A. Go Package Scheduling

`go test ./...` already executes independent package test binaries concurrently according to Go's package scheduling. As more packages are added, this provides additional parallelism without changing individual tests. The benchmark runner should record the package scheduling setting separately from test-level parallelism.

Package boundaries also define build and test cache invalidation. The MCP package previously held presentation projection and workspace root/path policy in the same package test binary as transport dispatch. Those cohesive helpers and focused tests now live in `internal/mcp/presentation` and `internal/mcp/workspace`. A change limited to server dispatch can leave those package test results reusable; a change to a helper still invalidates its own package and dependents that exercise it. The root package retains dispatch integration coverage.

For the current direct Go package graph, `internal/mcp` has 14 outgoing imports under `semedit/internal` and 3 incoming package users (instability 0.82). Each extracted package has 1 outgoing internal import and 1 incoming user (instability 0.50). These are coupling indicators, not package-size targets: no universal count identifies a good seam. Use the graph to find high-efferent packages, then inspect cohesive responsibilities and test dependencies before splitting. Afferent counts show who would be affected by a change, not which source files belong together.

### B. Unit and Integration Test Parallelism

Tests that call `t.Parallel()` can run concurrently within a test binary. Each candidate test must first prove isolation of temporary directories, module caches, environment variables, ports, subprocesses, and generated artifacts. Integration tests that share a daemon, network port, or mutable fixture must remain serialized or receive per-test resources.

### C. Rapid Property Tests

Rapid generated cases run sequentially within one property check. `t.Parallel()` or `go test -parallel` does not shard those cases. If they become a bottleneck, use process-level sharding with explicit, non-overlapping Rapid seeds, isolated failure artifacts, and aggregated exit status. Preserve per-shard shrinking and reproducibility.

### D. Check-Target Dependency Graph

The Make graph now completes mutating preparation (`fmt`, then `tidy`) before linting, vulnerability scanning, tests, and documentation generation. `make -j` can fan out those checks without readers observing source rewrites. Documentation generation writes only to its own output directories, and Vale selects Markdown files outside the temporary Go cache.

### E. Caching

Record cache state for every measurement. A successful package run can be reused when inputs and flags are cache-compatible. The current `make check` path runs `make test-core`, which invokes plain `go test` and is cache-compatible. `make test-race`, `make test-harness`, and `make test-all` use `-shuffle=on` and are intended to rerun randomized suites. Use `-count=1` for uncached execution measurements. Never compare cached validation time with a real validation run.

ADR-0093 establishes the storage policy: in-memory fixtures where they preserve
the tested behavior, otherwise private temporary workspaces on the
benchmark-managed tmpfs. Persistent module and reusable build caches stay in
their configured locations until complete-check capacity measurements support
moving the build cache. Core, race, harness, short, property, transcript, and
worktree test entry points run through a wrapper that verifies the managed
tmpfs and provides private `TMPDIR` and `GOTMPDIR` workspaces. On macOS, the
wrapper passes the benchmark linker-output refresh hook through Go test
subprocesses so test binaries linked on tmpfs remain executable. The helper
binary and reusable caches remain persistent. Peak-space measurement during a
full check remains outstanding.

#### 2026-10-22 Core Suite Profile and Cache Probe

The elevated `make check` passed. Its slow package test times were `cmd/semedit` 102.394 seconds, `internal/astedit` 87.913 seconds, `internal/godiagnostics` 147.405 seconds, and `internal/mcp` 79.897 seconds. Package times overlap because Go runs packages concurrently; they are not additive wall time.

A required `make check` rerun after the nested-block extraction changes also passed. The Go test phase reported `cmd/semedit` at 143.776 seconds, `internal/astedit` at 111.642 seconds, `internal/godiagnostics` at 163.644 seconds, and `internal/mcp` at 122.712 seconds. These package wall times overlap and the host was not controlled, so they identify where to profile next but do not establish a regression against the earlier run.

The CLI package runs one `testscript.Run` over all 510 script archives; the testscript library runs each archive as a parallel subtest. Two consecutive `GODEBUG=gocachetest=1 go test ./cmd/semedit` probes both missed the test-result cache and each took about 94 seconds. The hash trace includes generated testscript workspaces and changing `.scratch/go/tmp` directory metadata, but it does not yet isolate which input causes the miss. Go checks the previous run's recorded paths, so a vanished random temporary path can hash as missing on both lookup and save; random names alone are not proof of cache invalidation. The earlier `internal/adr` probe likewise establishes a miss but does not prove its `t.TempDir()` path caused it.

Source inspection identifies a concrete environment-key candidate for the CLI miss: `tools/run-with-test-tmp.sh` creates a unique RAM-disk root for each wrapper invocation and exports a changing `GOTMPDIR`, while `cmd/semedit/main_test.go` reads and forwards `GOTMPDIR` into every testscript environment. Go's test-result cache includes environment values that the test process reads. This should prevent result-cache reuse across wrapper invocations even when source and fixtures stay unchanged; measure it with `GODEBUG=gocachetest=1` before changing the wrapper. Compare repeated runs within one wrapper invocation and separate wrapper invocations, then test any stable-path proposal for concurrent-run isolation and cleanup. Keep private per-run `TMPDIR` and tmpfs placement intact unless the experiment proves another arrangement safe.

Before changing test roots, compare consecutive test-input hashes under an identical environment and identify the specific changing input. Then prototype deterministic, isolated workspaces only if the evidence shows they remove that input while preserving cleanup and concurrency safety. Separately, moving fixture groups into distinct Go test packages can limit test-result invalidation to changed shards, but package splitting is not automatically a wall-time win: testscript already parallelizes archives, so multiple packages may multiply concurrent CLI builds and subprocesses. Any split needs a shared harness or prebuilt CLI binary, complete stdout/diagnostic/final-workspace assertions, and a bounded total concurrency comparison against the current single-package run.

An uncached JSON timing run of `internal/godiagnostics` found the longest test functions were sequential edits (14.94s), import-closure refresh (13.48s), metadata-error graph refresh (11.60s), parse-memo freshness (9.97s), and fresh-discovery comparison (9.49s). Some already batch ordered edits against one fixture and session; preserve that pattern when it proves session reuse. The suite currently has no `t.Parallel()` calls, and fixture creation owns a fresh temporary module and `SessionManager`, so combining unrelated cases would change isolation. Group assertions only when they can share one workspace/session and retain the same lifecycle; the MCP catalog checks demonstrate that pattern. Profile `internal/astedit` at test-function granularity before considering its setup changes.

#### 2026-10-22 Test Topology Review

The default `make check` runs `lint-check`, `test-core`, and the two worktree/environment scripts after shared formatting and dependency preparation. `test-core` invokes `go test` across the core package list, so Go already schedules independent package test binaries concurrently. Increasing the number of directories alone does not change that schedule: a directory becomes a separate cache and scheduling unit only when it is a distinct Go package.

An independent read-only audit of successful `.scratch` check logs found 231.19–233.82 seconds wall time, 640.53–643.33 seconds user time, and 1,596.91–1,862.13 seconds system time. The logs do not preserve the timing wrapper or `MAKE_JOBS` override, so these results cannot be attributed to the default 12 Make jobs. Per-package test times were approximately 141–149 seconds for `internal/godiagnostics`, 85–100 seconds for `cmd/semedit`, 77–85 seconds for `internal/mcp`, 38–90 seconds for `internal/astedit`, and 17–23 seconds for `internal/operation`; they overlap and are not additive. At the time of these check logs, `cmd/semedit`'s 510 txtars ran as parallel testscript subtests inside a single package test action. `prepareFakeLSPs` already builds its fake servers once using package-level `sync.Once`; the repeated executable lookups in fixture setup are not a credible major cost. Existing concurrency traces do not record active worker counts, and there is no equivalent per-test timing trace for `internal/mcp`.

This evidence supports measuring a bounded `MAKE_JOBS` and Go `-parallel` matrix before changing concurrency defaults: Make's own parallel prerequisites, Go's package scheduler, and the parallel testscript subtests can overlap. No code change yet shows that reducing or increasing any of these limits improves elapsed time. Splitting the CLI fixtures into separate Go packages could retain package test-cache entries for untouched shards after a localized fixture change, but would require extracting the `package main` test harness and `TestMain` dispatch. It cannot avoid running every shard in a full check and may multiply setup and concurrent CLI work. Keep this as an incremental-cache experiment, not a full-check speedup claim. A directory split without a distinct Go package has no scheduling or test-result-cache effect. The separate evidence for setup sharing remains: group assertions only when they use the same workspace/session lifecycle; the fake-LSP setup is already shared, and existing diagnostics edit sequences deliberately reuse one session to prove its cache transitions.

The broad `internal/astedit` package contains many test files and both same-package and external-package tests. Moving files into directories is not a low-risk cache-only change: tests that inspect unexported implementation details would need a real production package boundary or would have to become black-box tests. Split only along cohesive code ownership seams, not to meet a test-file or package-size target. Prefer test-result sharding where an independently changing contract already has an appropriate package boundary.

Setup grouping has a different benefit from package splitting. Keep related assertions or ordered operations on one temporary workspace, compiler/session, catalog snapshot, or LSP lifecycle when they prove behavior over that same state. This can remove repeated startup and fixture construction and is already used by diagnostics sequential-edit tests and MCP catalog tests. Keep separate fixtures for independent failure modes or tests that need distinct lifecycle state; sharing a setup can make failures order-dependent and does not improve Go's package-level test-result cache. For expensive packages, time individual test functions first and group only when the profile shows repeated equivalent setup dominates.

The investigation order is therefore: (1) test whether changing `GOTMPDIR` explains cross-invocation CLI cache misses; (2) collect per-test timings for `cmd/semedit` and `internal/astedit`; (3) identify repeated expensive setup that can be shared without changing test isolation; (4) prototype a small CLI fixture shard only if the cache miss or fixture-change pattern justifies it; and (5) compare at least five paired warm and uncached runs, recording wall time, CPU, peak concurrency, and coverage. Package splitting is primarily an incremental-cache optimization; it should not be presented as a cold-suite speedup until measurements show one.

#### 2026-10-22 Full-Check Concurrency Probe

The default `make check` (12 Make jobs) failed once in `internal/godiagnostics/TestManagedDiagnosticsReuseAndRefresh` after the gopls session shutdown exceeded its context deadline; the package reported 177.306 seconds. Running that test alone through the verified RAM wrapper passed in 4.652 seconds. A complete `make MAKE_JOBS=4 check` then passed; package times included `cmd/semedit` 160.376 seconds, `internal/astedit` 78.606 seconds, `internal/godiagnostics` 168.451 seconds, and `internal/mcp` 120.116 seconds. These package durations overlap, and there was no wall-clock/CPU/peak-memory capture for a paired comparison. The pass at four jobs plus the isolated test pass make concurrency pressure a plausible cause of the first shutdown timeout, but do not establish that four jobs are faster or that the default should change. Repeat the bounded job-count comparison with wall time, CPU, and peak process/memory measurements before changing concurrency defaults.

#### 2026-10-23 Full-Check Package Timing Observation

The elevated `make check` passed. The `test-core` output reported `cmd/semedit` at 193.335 seconds, `internal/astedit` at 80.512 seconds, `internal/godiagnostics` at 166.780 seconds, and `internal/mcp` at 126.479 seconds. Go schedules these package test binaries concurrently, so the package durations overlap and do not add up to wall time. This run did not capture the check's total wall time, CPU, peak process count, or memory. The result identifies current profiling targets but cannot establish that any one package dominates elapsed time or that changing package boundaries would improve it. Continue with the cache-key and per-test timing experiments below before changing test topology or concurrency defaults.

## 5. Evaluation Protocol

For each proposed optimization:

1. Run the full serial check as the correctness reference.
2. Run the optimized check repeatedly on a clean and a warmed workspace.
3. Compare wall-clock time, CPU time, peak process count, package/test coverage, and failure behavior.
4. Repeat under race detection and with integration tests enabled.
5. Confirm that a seeded failure remains reproducible and that generated documentation is identical.

Report median and interquartile range over at least five repetitions. Keep direct `semedit` task latency, agent task latency, and quality-check latency as separate fields in benchmark results.

## 6. Open Questions

1. Which unit and integration tests are fully isolated enough to opt into `t.Parallel()`?
2. Which Make job limit balances validation latency with CPU and memory use on CI hosts?
3. Does package-level parallelism saturate CPU, memory, gopls processes, or filesystem I/O first as the repository grows?
4. Should Rapid property cases be sharded in-process or through multiple test-binary processes once their runtime justifies the added complexity?
5. Which checks should remain uncached for release confidence, and which can use deterministic cache-compatible flags during local development?
6. Which test-result boundaries can be split into packages without duplicating expensive setup or weakening black-box CLI coverage?

## 7. Next Steps

1. Verify the changing-`GOTMPDIR` cache-key hypothesis with paired same-wrapper and separate-wrapper CLI test runs before changing temporary workspace roots.
2. Collect test-function timings for `cmd/semedit` and `internal/astedit`; inspect expensive setup reuse in the top cases.
3. Measure peak tmpfs use under the full concurrent check before considering RAM-backed build caches; validate failure behavior when the volume is missing or undersized.
4. Prototype package-level CLI fixture sharding only if incremental cache savings outweigh duplicated harness setup and parallel resource demand.
5. Inventory isolated candidates for test-level parallelism and repeat the job-count comparison on a quiet CI workspace.
6. Add structured quality-check timing and cache fields to the benchmark runner's result schema, then refresh the baseline as new packages land.
