# ADR-0033: Exhaustive Assertion Reporting (Continue-on-Failure Tests)

Status: Accepted
Date: 2026-09-18

## Context

A test run that stops at the first failed assertion hides remaining defects
and forces repeated fix-and-rerun cycles. Each cycle costs a full `go test`
or txtar invocation plus fresh agent context. The repository already favors
`t.Errorf` for value checks and reserves `t.Fatalf` for setup failures (see
`internal/symbol/resolver_test.go`, `internal/bench/bench_test.go`,
`main_test.go`), and `make test` runs `go test` without `-failfast` so all
packages report. This practice was implicit. It needs an explicit contract so
new tests preserve single-run diagnostic completeness.

## Decision

1. Report every failed value assertion in a single run. Use continue-on-failure
   assertions (`t.Errorf`, `t.Error`, testify `assert`) for all checks where
   the test remains executable after the failure.
2. Reserve abort-on-failure (`t.Fatalf`, `t.FailNow`, testify `require`) for
   truly fatal preconditions only: setup that makes continuation impossible,
   meaningless, or panic-prone. Examples: `t.TempDir` replacement failures,
   fixture file writes that later reads depend on, constructor errors whose
   result is dereferenced below, nil prerequisites, missing testscript setup.
3. Table-driven tests iterate all rows. A row failure logs its case and the
   loop continues. Never `return`, `t.FailNow`, or `require.*` inside a row
   loop for a value mismatch.
4. Prefer independent `t.Run` subtests per case so `-run` filters and parallel
   execution isolate failures without aborting siblings.
5. Keep `make test` and CI free of `-failfast` as the default. Explicit
   opt-in `-failfast` remains available for local triage of cascading
   failures, never as the committed default.
6. Keep txtar scenarios independent so one script failure never masks others.
   Within a script, order assertions so a single run surfaces the full
   diagnostic picture where the harness permits.

## Invariants

- A value-check failure never aborts its test function, table loop, or package run.
- Abort-on-failure appears only at fatal preconditions where the next line
  cannot execute soundly without the guarded value.
- Table loops and subtests report each failing case with its input or name.
- Default test entry points (`make test`, `make check`, CI) run the full
  suite to completion and report all package failures.

## Consequences

One test run yields the complete fix list, reducing fix-and-rerun cycles.
Reviews treat abort-on-failure in a value check as a defect and request
conversion to a continuing assertion. Fatal-precondition aborts stay narrow
and documented by the setup code they guard. Future testify adoption follows
the same split: `assert` by default, `require` only for fatal preconditions.
