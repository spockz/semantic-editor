<!-- Summarizes completed smoke jobs while keeping task, setup, and fixture outcomes distinct. -->

# Luna 6 Multi-SWE-bench Go smoke, three iterations

Run: `luna6-multiswe-go-smoke-20261007-r3`.
Completed all 90 jobs: 60 independent coding attempts and 30 human-fix controls.
Luna 6 medium and high each ran three repetitions of both coding arms on five
fixtures. Small context, default prompt, concurrency four, twelve-minute job
timeout, write restriction, no optional semedit skill, and no MCP server instructions.
The invocation exited with code 1 after saving every checkpoint and grouped report.

## Results

No coding attempt passed. Among 60 attempts, 19 failed a graded oracle, 19 had
acceptance-test installation errors, and 22 further attempts used fixtures whose
human-fix controls failed. Setup errors take priority in these disjoint counts;
two occurred on the already invalid go-zero 964 fixture.

P = pass; F = graded task failure; E = harness setup error; I = invalid fixture.
Each coding cell totals three independent attempts. Human-fix controls are
source-patch replays and do not run a model.

| Fixture | Medium baseline | Medium semedit | High baseline | High semedit | Human-fix passes |
| --- | --- | --- | --- | --- | --- |
| grpc-go #2735 | 0P / 2F / 1E / 0I | 0P / 1F / 2E / 0I | 0P / 1F / 2E / 0I | 0P / 0F / 3E / 0I | 6/6 |
| grpc-go #2951 | 0P / 3F / 0E / 0I | 0P / 0F / 3E / 0I | 0P / 0F / 3E / 0I | 0P / 0F / 3E / 0I | 6/6 |
| grpc-go #3361 | 0P / 0F / 0E / 3I | 0P / 0F / 0E / 3I | 0P / 0F / 0E / 3I | 0P / 0F / 0E / 3I | 0/6 |
| go-zero #2463 | 0P / 3F / 0E / 0I | 0P / 3F / 0E / 0I | 0P / 3F / 0E / 0I | 0P / 3F / 0E / 0I | 6/6 |
| go-zero #964 | 0P / 0F / 0E / 3I | 0P / 0F / 0E / 3I | 0P / 0F / 1E / 2I | 0P / 0F / 1E / 2I | 0/6 |

## Interpretation

- grpc-go 2735: four graded failures and eight test-installation errors. The
  dataset acceptance tests require APIs such as `NewExponentialBuilder` and
  `WithConnectParams` that the graded implementations did not provide.
- grpc-go 2951: three graded failures and nine test-installation errors. Graded
  candidates failed the dataset acceptance compilation contract.
- go-zero 2463: twelve graded failures, with all six human-fix controls passing.
  Acceptance failures include missing `logc` exports such as `Info`, `Field`, and
  `AddGlobalFields`.
- grpc-go 3361: all six human-fix controls fail. The pinned TLS server certificate
  expired on 2025-11-01 at 02:20:24 UTC. Blocking TLS dial tests hang on this host;
  some coding jobs also emitted assignment-related assertions before their deadline.
  Its overall outcomes cannot establish model solve rates.
- go-zero 964: all six human-fix controls fail acceptance compilation. Errors
  include `gomonkey` v2.0.2's missing `buildJmpDirective` on the ARM64 host and
  tests calling `UnaryCrashInterceptor()` with no arguments, although the pinned
  source requires four. These controls remain invalid even if the dependency's
  architecture problem is repaired. Two candidates additionally conflict with
  the dataset test patch.

Test-installation conflicts arise when candidates modify test files whose base
context the dataset patch expects. The harness reports these as `ERROR` and
retains no task verdict for them. Future oracle assembly should prepare acceptance
tests from the pinned base independently of candidate test edits. Failing-control
fixtures need repair and a passing preflight before further model comparisons.

## Evidence

- `jobs/`: all 90 original job checkpoints; `manifest.json`: the original plan.
- Per-fixture directories: the harness's original JSON and Markdown reports.
- `evidence/transcripts.tar.gz`: preserved coding-session transcripts.
- `evidence/summary.json.txt`: structured counts, job classifications, hashes, and
  transcript mappings. The text suffix keeps evidence out of the report loader.
- `evidence/environment.json.txt`: Go version, certificate details, and binary hashes.
- `evidence/benchmark.log` and `evidence/make-check.log`: invocation and verification logs.
- Binary snapshots remain under `.scratch/luna6-smoke-evidence-20261007-r3`.

The harness and semedit binary hashes remained unchanged throughout the run.
`make check` had passed before the harness changes were measured, and the harness
suite passed. The latest repository-wide `make check` stalled during Go linting
past its configured five-minute timeout and was stopped after roughly ten minutes.
Markdown lint passed; this later verification is incomplete.

Previous invalid Luna 5.6/Luna 6 runs are preserved in
[the archive](../../archive/luna56-luna6-multiswe-go-smoke-20261006-invalid-oracle/README.md).
