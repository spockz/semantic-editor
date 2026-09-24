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

## 4. Parallelization Avenues

### A. Go Package Scheduling

`go test ./...` already executes independent package test binaries concurrently according to Go's package scheduling. As more packages are added, this provides additional parallelism without changing individual tests. The benchmark runner should record the package scheduling setting separately from test-level parallelism.

### B. Unit and Integration Test Parallelism

Tests that call `t.Parallel()` can run concurrently within a test binary. Each candidate test must first prove isolation of temporary directories, module caches, environment variables, ports, subprocesses, and generated artifacts. Integration tests that share a daemon, network port, or mutable fixture must remain serialized or receive per-test resources.

### C. Rapid Property Tests

Rapid generated cases run sequentially within one property check. `t.Parallel()` or `go test -parallel` does not shard those cases. If they become a bottleneck, use process-level sharding with explicit, non-overlapping Rapid seeds, isolated failure artifacts, and aggregated exit status. Preserve per-shard shrinking and reproducibility.

### D. Check-Target Dependency Graph

The Make graph now completes mutating preparation (`fmt`, then `tidy`) before linting, vulnerability scanning, tests, and documentation generation. `make -j` can fan out those checks without readers observing source rewrites. Documentation generation writes only to its own output directories, and Vale selects Markdown files outside the temporary Go cache.

### E. Caching

Record cache state for every measurement. A successful package run can be reused when inputs and flags are cache-compatible. The current `make test` command includes `-shuffle=on`, which chooses a new clock-based test-order seed and causes the full randomized invocation to execute again. Use `-count=1` for uncached execution measurements, or a deterministic cache-compatible command for cache studies. Never compare cached validation time with a real validation run.

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

## 7. Next Steps

1. Inventory unit and integration tests for shared-state hazards and add parallel-safe candidates incrementally.
2. Repeat the job-count comparison on a quiet, clean CI workspace and record resource use.
3. Add structured quality-check timing and cache fields to the benchmark runner's result schema.
4. Re-run this baseline when additional language packages and integration suites land.
