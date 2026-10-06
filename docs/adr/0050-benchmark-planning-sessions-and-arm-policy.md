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

Fixtures with pinned external repositories use the three-arm benchmark matrix:
`baseline-diff`, `semedit-agentic`, and `semedit-explicit`. The latter replays
the source dataset's `fix_patch` as an oracle upper bound and records
`execution_mode=ground_truth_fix_patch_replay`; it is not counted as an agent
tool invocation. External fixtures expose only their verbatim problem statement
to the agent. Their test and fix patches remain private to the harness, and the
test patch is applied only in the private oracle workspace. The optional
`--semedit-skill` condition installs the repository skill into the agent
workspace only for the `semedit-agentic` arm. It participates in job and pair
identities independently of the arm.

External repository caches are stored under the configured benchmark scratch
directory. Each run workspace is hydrated from the cache and checked out at its
declared base commit before agent timing begins. The harness uses copy-on-write
or reflink copy where supported and falls back to a regular recursive copy
without changing the measurement boundary.

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

The primary evaluation matrix standardizes on three canonical arms:

1. `baseline-diff`: Vanilla LLM editing with native file manipulation / patch tools (no semedit tools).
2. `semedit-agentic`: Autonomous agentic AST discovery, navigation, and editing using semedit MCP tools.
3. `semedit-explicit`: Deterministic AST replay decomposed directly from the human ground-truth PR patch (`fix_patch`).

The legacy `--mcp-server-instructions=none,descriptive,prescriptive` Cartesian array is deprecated.
In its place, skill delivery is evaluated as an orthogonal boolean ablation condition (`--semedit-skill=true|false`),
testing whether prompt-delivered macro-refactoring guidelines alter agent tool selection efficiency without inflating primary evaluation arms. Direct control jobs execute under deterministic AST semantics without agent interaction.

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

Fixtures may instead declare `interactive_mode: staged` with an initial edit
count and ordered `staged_followups`. Each follow-up has its own instruction,
cumulative edit count, and oracle. The harness resumes the same session and
workspace for every stage and evaluates every checkpoint, even if an earlier
checkpoint failed. Overall success requires all checkpoints to pass. Every turn
records elapsed time and token usage separately, so first-turn startup and
subsequent continuation costs can be compared. Staged turns are excluded from
corrective-attempt publication metrics.

Measured results contain only the original task answer and subsequent interactive
attempts to solve the task. Freeze task metrics before diagnostic questioning.
Any non-use, batching, or integration-value reflection has a separate diagnostic record and contributes
nothing to measured tokens, turns, tools, elapsed time, or oracle outcomes.
Every successful semedit session with a continuation identity receives an additional
integration-value question after any non-use or batching question. Store its prompt,
response, and capture error in `semantic_integration_reflection`; ask for a candid
one-sentence coding-agent recommendation, including whether the agent would ask
its owner to add semedit. Accept negative recommendations and acknowledge unassessed
tools. Render this as agent opinion, never a performance measurement. Failed or
non-resumable sessions keep the existing diagnostic skip behavior.
Cumulative provider transcripts must contribute each task event at most once.

The CLI defaults to the control target, all discoverable benchmark fixtures,
requested `small,large` contexts, all declared prompts, one repeat, four workers,
and five minutes per job. The configured output directory requires an explicit
run identifier; an empty output directory disables per-run files. `--matrix`
is a deprecated no-op. `--harness` is a deprecated single-target alias and cannot
be combined with `--target`. Explicit `--arm` selection is rejected. Validated
compound context/prompt variants remain supported. Make targets retain their
configured agent target and both use the same plan-and-execute path.

When run output is enabled, write a versioned manifest before dispatch and an
atomic one-run report for each completed job under `jobs/`. Each job record also
stores its resolved identity and prompts, so later analysis does not depend on
the aggregate reports. `--resume` requires the same run ID and matching plan
fingerprint, including fixture bytes and measurement-affecting settings. Resume
loads valid checkpoints and dispatches only jobs without a saved result. Failed
task outcomes count as completed to avoid silently repeating provider work;
signal-cancelled jobs remain pending and can run on resume.

## Invariants

- Planning errors prevent all backend execution; intentional exclusions remain
  distinct from errors and empty plans are reported explicitly.
- The printed plan and execution consume the same jobs and resolved prompts.
- Unsupported contexts are never relabelled or populated with generic overlays.
- Every planned job has exactly one terminal result and advances progress.
- Per-job checkpoints use atomic writes; a resume with a mismatched plan fails before dispatch.
- Resume never repeats a saved terminal task failure and never treats signal cancellation as completed work.
- Agent pairs share experiment conditions but only semedit receives policy steering.
- Different policies, including historical unspecified policy, never merge silently.
- Follow-ups reuse the task session and workspace and never receive hidden oracle output.
- Diagnostic questioning never changes measured task aggregates.
- Transcript failures remain explicit; missing telemetry is not measured zero.
- External repository arms share a pinned base commit; acceptance patches remain
  outside the issue prompt and test patches stay outside the agent workspace.

## Consequences

Execution becomes inspectable and reproducible across narrow and broad selections.
Failures remain visible instead of disappearing from result totals. Policy studies
can vary inspection and editing constraints independently, at the cost of an
additional comparison dimension and explicit fixture context metadata. Legacy
reports remain publishable without claiming an unrecorded restriction policy.

## Implementation guide

The [benchmark harness architecture guide](../../tools/benchmark-harness/README.md) maps this decision to current entry points, state ownership, source files, publication code, and regression tests. Start there when locating a change.
