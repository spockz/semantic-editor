<!-- Records the repeated Luna 6 smoke run after fixing private test-patch installation. -->

# RQ-0066: Luna 6 Multi-SWE-bench Go Smoke and Oracle Integrity

Status: Resolved
Date: 2026-10-07

## Questions

Which of the five derived Go tasks pass with the dataset acceptance tests
installed, and are the outcomes stable across three independent iterations at
medium and high effort? Do the human-fix controls pass the same oracle?

## Prior evidence

The 2026-10-06 Luna 5.6/Luna 6 smoke runs are invalid for solve-rate comparisons.
The private oracle excluded `.git` while remaining under an agent checkout. Git
discovered the parent checkout, silently skipped the test patches, and returned
exit code 0. Earlier pilots also contain copy and sandbox setup failures.

All 11 result directories and 48 referenced evidence items are preserved in the
[archive](../../data/benchmarks/archive/luna56-luna6-multiswe-go-smoke-20261006-invalid-oracle/README.md).
The archive sits outside the active results tree.

## Method

- Run ID: `luna6-multiswe-go-smoke-20261007-r3`.
- Targets: `codex/gpt-6-luna/medium` and `codex/gpt-6-luna/high`.
- Fixtures: `grpc__grpc-go-2735`, `grpc__grpc-go-2951`, `grpc__grpc-go-3361`,
  `zeromicro__go-zero-2463`, and `zeromicro__go-zero-964`.
- Small context, default prompt, three independent iterations, concurrency four,
  twelve-minute job timeout, write restriction, no optional semedit skill.
- Ninety jobs: sixty coding attempts across baseline and semedit-agentic arms,
  plus thirty deterministic human-fix patch replays.
- Every coding workspace receives a fresh copy of the pinned local checkout
  before provider launch. All five caches have the expected HEAD and no remotes.
  Coding shell network access is disabled; absent source caches fail immediately.
- Private patch application bounds Git discovery, rejects skipped patches, and
  reverse-checks successful application. Test installation errors produce harness
  `ERROR` outcomes; acceptance-test failures produce task `FAIL` outcomes.

The corrected harness passed `make check` and `make test-harness` before this run.
The run's development-binary Make target later encountered unrelated lint errors
in concurrent `tools/package-closure` changes. The harness built successfully;
the development binary was compiled directly with the repository's local Go state.

## Evidence and findings

Results are checkpointed under
`data/benchmarks/results/luna6-multiswe-go-smoke-20261007-r3`.
Do not infer solve rates from incomplete jobs or from fixtures whose human-fix
control fails. Record control, harness, timeout, and task outcomes separately.

The first two tasks completed with twelve passing human-fix controls, no coding
passes, seven acceptance failures, and seventeen setup conflicts. Conflicts arise
when agents modify test files whose base context the dataset patch expects.
These attempts are ungradable; future oracle assembly should prepare acceptance
test content from the pinned base independently of agent test modifications.

The grpc 3361 fixture's `testdata/server1.pem` expires on 2025-11-01 at
02:20:24 UTC. Its `test.configDial` installs credentials from `ca.pem` and enables
`grpc.WithBlock` without a dial timeout; `TestCZSocketGetSecurityValueTLS` uses this
path. The pinned fixture therefore has an expired-certificate environment problem
on the 2026-10-07 host, independently of the requested balancer change. Record
this limitation when interpreting its controls and coding outcomes.

## Final measurements

All ninety jobs completed, and the invocation exited with code 1 after saving
reports. No Luna 6 coding attempt passed at either effort across three repetitions.
The sixty coding attempts comprise nineteen graded task failures, nineteen
acceptance-test setup errors, and twenty-two further outcomes on invalid fixtures.
These counts are disjoint: setup errors take priority, including two on go-zero 964.

Human-fix controls passed all six repetitions for grpc 2735, grpc 2951, and
go-zero 2463. All six controls failed for grpc 3361 and go-zero 964. The former
has the expired TLS certificate described above. The latter fails compilation
with `gomonkey`'s undefined `buildJmpDirective` and an acceptance test calling
`UnaryCrashInterceptor()` without its four required arguments. Repairing host
architecture compatibility alone does not repair that test API mismatch.

The nineteen setup conflicts are ungraded. Both failing-control fixtures are
invalid for overall model solve-rate inference. The nineteen remaining graded
failures establish failed dataset acceptance outcomes for those candidates.
Future comparisons need independently prepared acceptance tests and passing
human-fix preflights; these measurements do not support ranking medium versus high.

[The complete summary](../../data/benchmarks/results/luna6-multiswe-go-smoke-20261007-r3/smoke-summary.md)
contains counts by fixture, effort, and coding arm. All sixty coding transcripts,
the invocation log, verification log, binary hashes, and environment evidence are
preserved alongside the reports. Binary hashes were unchanged throughout the run.
The later `make check` stalled in Go linting beyond its configured five-minute
timeout and was stopped after roughly ten minutes; Markdown lint passed.

## Acceptance fixture repairs, 2026-10-08

Explicit recipe authoring repaired both invalid fixtures while preserving their
assertions and original artifacts. grpc 3361 keeps the same certificates, CA,
server name, and trust checks, with a private TLS verification clock fixed to
2020-01-01 inside their validity interval. go-zero 964 passes
`UnaryCrashInterceptor` as a function value and pins the test-only
`gomonkey/v2 v2.14.3` dependency, including its checksums, in the private test
patch. Its oracle command declares `-gcflags=all=-l` to ensure monkeypatches
execute on current Go toolchains. Stale test dependency hunks were removed from
the human reference patch; production semantic edits add no test dependency.

The fresh single-batch recipes pass the ordinary corrected oracles. Original
and corrected patches, failure receipts, successful replays, and hashes are
preserved under [fixture validation](../../data/benchmarks/fixture-validation/).
These corrections change fixture identities. They do not reclassify the ninety
historical jobs or constitute new model solve-rate or performance measurements.
