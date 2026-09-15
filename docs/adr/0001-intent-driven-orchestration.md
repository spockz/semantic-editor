# ADR-0001: Intent-Driven Orchestration vs. Text Patching

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Coding agents typically emit raw text diffs or full-file rewrites. This approach suffers from:

* Token asymmetry (spending thousands of output tokens on mechanical syntax changes).
* Context window exhaustion (filling history with diff lines rather than requirements/planning).
* Diff application drift (whitespace mismatches, truncated tokens, hallucinated imports).
* Model and provider variance (diurnal rush hours, quantization changes, and rate limits causing high retry costs).

## Decision

`semedit` inverts the editing model:

1. The LLM acts strictly as a high-level planner and orchestrator, issuing concise semantic intent commands (`rename`, `extract`, `inline`, `organize_imports`).
2. The local host CPU executes mechanical syntax changes using existing language servers (`gopls`, `rust-analyzer`, `tsserver`, `jdtls`) and AST utilities.
3. The compiler/typechecker validates changes locally and returns structured diagnostics to the model.

## Invariants

* The LLM should never be required to generate textual diffs for supported semantic operations.
* All mechanical transformations must execute locally on the host CPU.

## Consequences

* **Positive**: Drastically reduced token costs, millisecond execution latency, elimination of diff application drift, network and provider drift resiliency.
* **Negative**: Requires host environments to have language tooling installed (or bundled).
