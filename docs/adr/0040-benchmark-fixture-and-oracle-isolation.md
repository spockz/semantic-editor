# ADR-0040: Benchmark Fixture and Oracle Isolation

Status: Accepted
Date: 2026-09-21

## Context

Small single-edit tasks stopped discriminating between direct file editing and
semantic operations as agents became reliable at one-shot changes. A useful
benchmark must expose related, non-uniform code across a compiling project
without revealing the implementation plan or allowing an agent to modify the
acceptance test that proves the result.

## Decision

Benchmark tasks are executable txtar fixtures with explicit visible prompts
and small or large project-size variants. Visible source, documentation prose,
and ordinary unit tests model the working repository, but must not encode the
solution or act as a direct acceptance oracle.

The harness executes an agent inside a fresh extracted fixture. It does not
add task-aware source overlays or runtime restrictions beyond that fixture
boundary. It presents the fixture-declared protected-file list as a static
prompt guardrail, using a generic instruction not to edit tests and including
recovery guidance to restore a protected file if an earlier turn changed it.
This exposes a task constraint, not the oracle's observed result. The oracle
enforces mutation policy, structural postconditions,
clean compilation, and behavioral verification. Hidden acceptance tests are
stored outside the extracted fixture and copied only after the agent exits.
The copy path is root-scoped so an agent-created symlink cannot redirect a
hidden test outside the benchmark workspace.

New semantic-routing scenarios should require several coordinated edits across
related but inconsistent code, while preserving unrelated protocols with
similar names. Task 11 is the reference shape: normalize an audit delivery
boundary, handle control traffic, preserve other sink protocols, and work
across small and large variants without revealing a symbol-by-symbol plan.

## Invariants

- Every runnable task has a non-empty txtar prompt variant.
- The model cannot inspect or alter hidden acceptance tests before execution
  completes.
- The prompt may state fixture-declared protected files, but never reveals an
  oracle result, a failed condition, or hidden-test output.
- Visible tests may represent existing behavior but must not be a solution
  checklist.
- The oracle rejects disallowed mutation, validates semantic postconditions,
  and runs build and hidden behavioral checks independently.
- Small and large variants preserve the same task contract while changing the
  amount of related code an agent must navigate.

## Consequences

Benchmark fixtures better measure semantic routing and multi-step planning
than trivial body replacement. They cost more to author because the visible
repository narrative and hidden oracle must be designed separately.
