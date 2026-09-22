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
Fixture-local Go state is necessary for sandbox correctness, but a completely
cold module cache can dominate a short benchmark. Host cache reuse would make
that speed dependent on mutable state outside the experiment.
An interactive baseline must distinguish an agent's first-turn instruction
following from its ability to recover through an intentionally supplied,
reproducible sequence of additional guidance.

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

Every benchmark fixture starts from an isolated state. The harness may
explicitly preseed selected fixture state before the agent starts, such as a
read-only vetted Go module cache. A preseed source is runner configuration,
never implicit host-cache reuse. Until a preseed mechanism is configured, each
fixture uses an empty local Go state.

Before starting a Codex session, the harness writes a benchmark-owned,
fixture-local `AGENTS.override.md`. It defines the fixture as the complete
workspace while allowing any documentation the fixture supplies. It prohibits
using paths outside the fixture, which excludes unrelated host-project material
without artificially depriving the agent of task context. This preserves the
authenticated global Codex profile. Codex does not expose an instruction file's
provenance to the model; the fixture instruction therefore specifies a concrete
path boundary rather than asking the model to distinguish inherited
instructions. The local override is an auditable isolation boundary rather than
an attempt to replace `CODEX_HOME`.

The harness records each arm's initial turn as a one-shot observation. It may
then resume that same harness session for a separate interactive observation.
This applies equally to baseline and semantic-edit arms: the ladder measures
recovery from the same task guidance, while the arm controls the available
tools and server instructions. Each follow-up prompt belongs to the task's
txtar fixture, is ordered, and becomes progressively more specific; its final
step may provide an explicit solution hint. The external oracle decides whether
another step is needed, but its result and hidden-test output are never sent to
the agent. The harness derives a visible mutation-policy guardrail from the
fixture's declared protected-file list for the initial prompt and every
follow-up. Follow-ups additionally tell the agent to restore any protected
file it changed earlier. This is a static task constraint, not failure
feedback or hidden-test output. The harness must not invent an escalation
ladder. A task without declared follow-ups publishes its one-shot observations,
but skips interactive observations for every arm.

When a semantic-edit arm completes all task turns without a confirmed
`semantic_*` MCP invocation, the harness resumes that same session once for a
diagnostic-only reflection. The reflection asks why the model did not use the
available semantic tools and forbids further edits or tool calls. It is stored
and rendered with the run, but it is excluded from one-shot, interactive
recovery, oracle, tool-use, token, turn, and latency measurements. It is
explanatory evidence, not an additional treatment or a correction attempt.

When a semantic-edit arm instead makes two or more consecutive `semantic_*`
calls without any `semantic_batch` invocation, the harness resumes the same
session once to ask why the operations were not batched. This batch-use
reflection is mutually exclusive with the non-use reflection and has the same
diagnostic-only, non-measured status.

## Invariants

- A comparison never merges runs with different MCP server instruction modes.
- A comparison never splits only because technical provenance differs.
- Baseline runs receive neither semedit MCP tools nor semedit server
  instructions.
- A task without declared non-empty prompts is reported as skipped, not run
  with an inferred prompt.
- Legacy result fields may be decoded for report continuity, but new results
  use `mcp_server_instructions` and `provenance`.
- Fixture state is isolated from mutable host state. Any preseed is explicit,
  deterministic, and recorded as technical provenance.
- Every Codex fixture has a harness-owned `AGENTS.override.md` before session
  creation. A fixture that already reserves that filename fails setup rather
  than silently weakening instruction isolation.
- An interactive observation reuses its arm's one-shot harness session and
  workspace; it is not a second independent run.
- Follow-up prompts are task-owned, ordered experimental input. The oracle
  controls continuation without becoming agent-visible feedback, except for
  its declared protected-file list, which is a visible static guardrail.
- A missing task-owned follow-up ladder skips only the interactive-baseline
  and interactive-semantic-edit observations; the harness never supplies
  generic fallback prompts.
- A semantic-tool non-use reflection reuses the completed semantic session and
  is diagnostic-only. It never changes the task workspace, oracle outcome, or
  measured task-turn aggregates.
- A semantic batch-use reflection is requested only for an unbatched contiguous
  sequence of at least two semantic calls. It is diagnostic-only and never
  changes the task workspace, oracle outcome, or measured aggregates.

## Consequences

The harness can compare normal, prompt-steered, and server-steered behavior
without conflating them. Result matrices grow only for deliberate treatments;
technical tuning remains traceable without creating misleading sparse tables.
An explicit preseed can reduce setup cost while preserving reproducibility, at
the cost of maintaining and identifying that seed separately from the fixture.
The fixture-level instruction override prevents unrelated global repository
workflow guidance from becoming an unrecorded input while retaining Codex
authentication and other global runtime state.
The resulting measurements separate first-turn instruction following,
interactive recovery effort, and semantic-edit performance without conflating
session startup or hidden-oracle feedback with any of those treatments.
The attached reflection makes an unconfirmed semantic-tool absence
interpretable without converting the explanation into performance data.
