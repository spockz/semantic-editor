# ADR-0050: Benchmark Planning, Sessions, and Arm Policy

Status: Accepted
Date: 2026-09-26

## Context

The benchmark harness had separate single-run and matrix execution paths.
Fixture expansion could silently omit failures or apply a large-context overlay
to archives with only one supported context. Provider selection was repeated
across task turns and diagnostic reflections. Fixed semantic-arm steering also
combined source-inspection and modification restrictions without recording the
choice as an experimental condition.

## Decision

All benchmark executions construct one resolved plan before starting a backend.
The plan contains ordered jobs, stable job and comparison-pair identities,
loaded fixtures, resolved initial and conditional follow-up prompts, selected
conditions, output settings, and explicit exclusions. A scheduler dispatches
these jobs without rediscovering fixtures or expanding dimensions. Agent targets
always schedule paired baseline and semedit jobs. Direct control uses the same
scheduler. Complete plan rendering precedes execution.

Fixtures declare supported physical contexts. An archive without additional
context declarations has only its established base context. Verification
variants remain treatments over those physical contexts. The planner intersects
requests with supported contexts, and reports exclusions with requested and
available contexts. Extraction applies large overlays only where declared.

A session owns its workspace, initial snapshot, selected provider adapter,
continuation identity, transcript, turn boundaries, and result accumulation.
Provider adapters retain protocol-specific decoding. Every terminal job outcome,
including setup failure and cancellation before dispatch, is retained. Progress
uses completion order; final results retain plan order.

The command-line interface (CLI) exposes
`--semedit-arm-restrict=read|write|readwrite`. The explicitly selected default is
`write`, including the benchmark Makefile variable `SEMEDIT_ARM_RESTRICT`.

| Policy | Semedit-arm steering |
| :--- | :--- |
| `read` | Forbid shell source inspection; require semantic or built-in code inspection. Add no write restriction. |
| `write` | Require semantic tools for supported code modifications. Add no read restriction. |
| `readwrite` | Apply both restrictions. |

Shell builds and tests remain permitted under every policy. Steering is generated
centrally and applied only to semedit task turns, including declared follow-ups.
Baseline behavior remains unchanged. Both agent arms record the selected policy
as a comparison condition, with separate per-job applicability metadata. Direct
control has no applicable restriction. Empty or invalid explicit policy values
fail planning before execution.

Comparisons, output identities, best-case selection, aggregates, and browser
views distinguish the policy. Historical records without policy metadata remain
an unspecified condition, never implicitly `write`. Existing duration units,
oracle publication rules, provenance treatment, and ranking rules remain intact.
New records with planned job identities retain exact fixture task IDs in report
grouping, so generated-large and physical-large fixtures cannot overwrite one
another. Historical records retain their previous task-name normalization; this
does not reinterpret old task 01b observations.

Measured results contain only the original task answer and subsequent interactive
attempts to solve the task. Freeze task metrics before diagnostic questioning.
Any non-use or batching reflection has a separate diagnostic record and contributes
nothing to measured tokens, turns, tools, elapsed time, or oracle outcomes.
Cumulative provider transcripts must contribute each task event at most once.

The CLI defaults to the control target, all discoverable benchmark fixtures,
requested `small,large` contexts, all declared prompts, one repeat, four workers,
and five minutes per job. The configured output directory requires an explicit
run identifier; an empty output directory disables per-run files. `--matrix`
is a deprecated no-op. `--harness` is a deprecated single-target alias and cannot
be combined with `--target`. Explicit `--arm` selection is rejected. Validated
compound context/prompt variants remain supported. Make targets retain their
configured agent target and both use the same plan-and-execute path.

## Invariants

- Planning errors prevent all backend execution; intentional exclusions remain
  distinct from errors and empty plans are reported explicitly.
- The printed plan and execution consume the same jobs and resolved prompts.
- Unsupported contexts are never relabelled or populated with generic overlays.
- Every planned job has exactly one terminal result and advances progress.
- Agent pairs share experiment conditions but only semedit receives policy steering.
- Different policies, including historical unspecified policy, never merge silently.
- Follow-ups reuse the task session and workspace and never receive hidden oracle output.
- Diagnostic questioning never changes measured task aggregates.
- Transcript failures remain explicit; missing telemetry is not measured zero.

## Consequences

Execution becomes inspectable and reproducible across narrow and broad selections.
Failures remain visible instead of disappearing from result totals. Policy studies
can vary inspection and editing constraints independently, at the cost of an
additional comparison dimension and explicit fixture context metadata. Legacy
reports remain publishable without claiming an unrecorded restriction policy.

## Implementation guide

The [benchmark harness architecture guide](../../tools/benchmark-harness/README.md) maps this decision to current entry points, state ownership, source files, publication code, and regression tests. Start there when locating a change.
