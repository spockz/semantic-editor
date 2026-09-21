# ADR-0038: Benchmark Experiment Conditions and Provenance

Status: Accepted
Date: 2026-09-21

## Context

Benchmark results became difficult to compare when prompt wording, MCP server
instructions, model settings, and technical runtime settings could all change
without a stable distinction between experimental treatment and execution
provenance. In particular, server-supplied MCP instructions are outside the
user prompt, so folding them into a prompt variant would hide a meaningful
intervention. Conversely, parser limits and similar diagnostics settings do
not describe a product treatment and must not split comparison cells.

## Decision

The benchmark harness records these first-class experiment conditions:

- target, including harness, model, and reasoning level;
- task, project-size variant, prompt variant, and evaluation arm;
- MCP server instruction mode: `none`, `descriptive`, or `prescriptive`.

Prompt variants are runnable only when their txtar fixture declares a
non-empty `prompt_variants` entry. The harness never invents a prompt from
metadata and never expands a missing variant into a matrix cell.

MCP server instruction mode is delivered through the MCP initialization
`instructions` field, not appended to the task prompt. It remains independent
of prompt wording and is part of comparison grouping and result identity.

Technical settings use repeatable `provenance` key-value entries. They are
retained in each result for auditability, but do not create comparison cells or
change result filenames. The old `classifier` spelling is a compatibility
alias only.

The default Make target is `codex/gpt-5.6-luna/medium`; concurrency, timeout,
and parser limits are operational settings rather than experimental conditions
unless a future study explicitly promotes one to a condition.

## Invariants

- A comparison never merges runs with different MCP server instruction modes.
- A comparison never splits only because technical provenance differs.
- Baseline runs receive neither semedit MCP tools nor semedit server
  instructions.
- A task without declared non-empty prompts is reported as skipped, not run
  with an inferred prompt.
- Legacy result fields may be decoded for report continuity, but new results
  use `mcp_server_instructions` and `provenance`.

## Consequences

The harness can compare normal, prompt-steered, and server-steered behavior
without conflating them. Result matrices grow only for deliberate treatments;
technical tuning remains traceable without creating misleading sparse tables.
