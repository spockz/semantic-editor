# RQ-0035: Detecting Consecutive Switches on the Same Expression

* **Status**: Open
* **Date**: 2026-09-24

## Question

Should the Go lint configuration include a rule that reports consecutive switches on the same expression and recommends combining their cases where that preserves behavior?

## Context

Multiple switches on the same expression can be intentional. Consecutive switches may repeat common work across groups of cases, while nested switches may encode distinct logic. A general automatic fusion could change declaration scope, execution order, fallthrough behavior, or side effects. The rule should therefore be evaluated as a diagnostic first, with its scope and false-positive rate measured against repository code and representative examples.

## Investigation

* Define the precise pattern: adjacent switches in the same statement list, identical selector expressions, and no intervening statement.
* Compare existing Go linters and golangci-lint extension points before adding custom analysis.
* Evaluate grouped cases and helper extraction as alternatives, including declaration and side-effect edge cases.
* Decide whether nested repeated-selector switches should be reported separately from consecutive switches.

## Resolution Criteria

Resolve this question after selecting the rule scope, implementation mechanism, diagnostics, and evidence that the rule is useful without encouraging unsafe automatic rewrites.
