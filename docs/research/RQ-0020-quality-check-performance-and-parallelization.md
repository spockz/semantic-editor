# RQ-0020: End-to-End Quality Check Performance and Parallelization

* **Status**: Open
* **Date**: 2026-09-16
* **Category**: Build & Test Infrastructure

## 1. Question

How can the complete project quality check become faster without reducing coverage, determinism, or confidence? The scope includes formatting, dependency consistency, Go unit tests, integration and property-based tests, linting, vulnerability checks, documentation generation, and published-site assertions.

## 2. Context and Measurement Boundary

The top-level `make check` target is the project's quality gate:

```text
fmt -> tidy -> lint -> vuln -> test -> verify-docs
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

Four jobs reduced elapsed validation time by approximately 10% relative to the serial baseline. Eight jobs provided no further improvement. These runs are observational only: the current Make graph allows `fmt` and `tidy` to mutate files while lint, tests, and documentation generation read them, so parallel success does not yet establish a safe correctness policy.

Hugo reported approximately 0.43 seconds of the run. Documentation rendering is therefore not the dominant cost; Go test packages, including the gopls-backed property test and integration scripts, account for most of the work.

## 4. Parallelization Avenues

### A. Go Package Scheduling

`go test ./...` already executes independent package test binaries concurrently according to Go's package scheduling. As more packages are added, this provides additional parallelism without changing individual tests. The benchmark runner should record the package scheduling setting separately from test-level parallelism.

### B. Unit and Integration Test Parallelism

Tests that call `t.Parallel()` can run concurrently within a test binary. Each candidate test must first prove isolation of temporary directories, module caches, environment variables, ports, subprocesses, and generated artifacts. Integration tests that share a daemon, network port, or mutable fixture must remain serialized or receive per-test resources.

### C. Rapid Property Tests

Rapid generated cases run sequentially within one property check. `t.Parallel()` or `go test -parallel` does not shard those cases. If they become a bottleneck, use process-level sharding with explicit, non-overlapping Rapid seeds, isolated failure artifacts, and aggregated exit status. Preserve per-shard shrinking and reproducibility.

### D. Check-Target Dependency Graph

Separate mutating preparation (`fmt`, `tidy`) from read-only checks. A safe graph could run formatting and tidying first, then fan out lint, vulnerability scanning, unit tests, integration tests, and documentation verification. This would make `make -j` useful without allowing readers to observe files while they are being rewritten.

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
2. Should the Makefile expose separate `check-prepare`, `check-tests`, `check-docs`, and `check` targets with explicit dependencies?
3. Does package-level parallelism saturate CPU, memory, gopls processes, or filesystem I/O first as the repository grows?
4. Should Rapid property cases be sharded in-process or through multiple test-binary processes once their runtime justifies the added complexity?
5. Which checks should remain uncached for release confidence, and which can use deterministic cache-compatible flags during local development?

## 7. Next Steps

1. Inventory unit and integration tests for shared-state hazards and add parallel-safe candidates incrementally.
2. Prototype a dependency-ordered Make graph and compare it with the current serial baseline.
3. Add structured quality-check timing and cache fields to the benchmark runner's result schema.
4. Re-run this baseline when additional language packages and integration suites land.
