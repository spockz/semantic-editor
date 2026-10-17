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
`baseline-diff`, `semedit-agentic`, and `semedit-explicit`. All use the agent
session lifecycle. The explicit prompt contains exact semedit operations,
complete arguments, batch grouping, and order, authored and validated by
executing semedit from the pinned base through the normal oracle. External
fixtures keep this prompt in private `.benchmark/explicit.md`; only the explicit
arm receives it. Missing or blank recipes fail planning before provider launch.
Baseline and agentic receive the verbatim issue. Test and fix patches remain
private; only the private oracle installs the test patch. Human fix replay is a
fixture diagnostic, excluded from semedit performance comparisons, including
legacy `execution_mode=ground_truth_fix_patch_replay` records. The optional
skill condition applies equally to both semedit arms and participates in job
and pair identities independently of the arm.

The private human fix may also inform an authored `interactive_followups` hint
ladder. External baseline and agentic receive identical progressive task content
after unsuccessful attempts, with ordinary arm steering preserved. The complete
explicit recipe has no corrective hint ladder. Hints remain independent of the
agent's hidden oracle output and do not expose raw patch text or hidden tests.
Harness errors terminate the session; only graded task failures advance hints.
Each received hint, verdict, and cost is retained in the measured session so
first-attempt success can be distinguished from success after hints.

External repository caches are stored under the configured benchmark scratch
directory. Each run workspace is hydrated from the cache and checked out at its
declared base commit before agent timing begins. The harness uses copy-on-write
or reflink copy where supported and falls back to a regular recursive copy
without changing the measurement boundary.

When enabled, the optional semedit skill is installed in every semedit agent
workspace, including first-party fixtures, before the initial workspace
snapshot. Baseline workspaces never receive it. The skill condition is recorded
in each job identity and comparison-pair identity so skill and no-skill runs
cannot share checkpoints or be resumed as the same plan.

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

Shared workspace code owns directory creation, atomic runtime file publication,
Go cache preparation, and verification prewarm for every harness. The generic
session invokes provider configuration preparation before taking the original
workspace snapshot. Adapters select their protocol-specific paths and contents,
process arguments, permissions, and event decoding. Acceptance test insertion,
grading, selection of follow-up prompts, and semantic diagnostic interrogation
remain generic and use the original session. Harness-created configuration must
never be graded as an agent edit.

Prepare Go production and test dependencies in each fixture's local cache before
timed agent work, without changing module manifests. Apply this to every provider
and arm. Bound preparation and each acceptance evaluation separately to five
minutes while preserving parent cancellation. Task turns share the configured
agent deadline. Oracle cancellation is an execution error, not a failed answer.
Retain primary provider failures together with any cleanup failures. Configure
local OpenCode models with their loaded context capacity so overflow handling
can operate; output-token exhaustion is an explicit incomplete-task failure.

Codex benchmark threads are ephemeral in the app-server protocol. Require the
server to confirm this at thread creation. The original live thread remains
available for all task turns, while the harness transcript is the durable
record for human inspection and agent analysis. Native conversation persistence
is not required, and benchmark jobs must not create stored app conversations
or workspace groups. Preserve each job's isolated cwd rather than grouping
jobs under a shared top-level project.

Every coding harness retains its original live coding session and MCP service
for the entire task. Initial work, corrective hints, staged turns, and diagnostic
questions all use that original identity. Missing or changed identities and
provider failures terminate continuation without creating a replacement.
Freeze workspace, model, effort, MCP configuration, and sandbox settings.
Close and reap the provider before collecting final MCP metrics and deleting
the workspace. Record each adapter's transport in provenance and resume
fingerprints to prevent mixing legacy CLI-resume jobs with persistent jobs.

Codex owns one app-server process and transport for the entire task session.
Initialize the connection and start the thread once. Send every subsequent hint,
stage, and diagnostic question to that original thread ID with `turn/start`.
Repeated CLI resume processes lose MCP and gopls memory even when their thread
IDs match. A dead connection or changed identity fails explicitly without a
replacement process. Freeze workspace, model, effort, MCP configuration, and
network-denied sandbox settings for the lifetime. Await a matching completed
turn notification before another turn. Account for usage with cumulative token
snapshot differences. Close and reap the provider before collecting final MCP
metrics and deleting the workspace.
OpenCode owns one `serve` process. Thin `run --attach` clients submit each turn
with the same absolute directory and original session ID; tools and MCP state
remain in the server's retained directory instance. Its sandbox permits only
the selected localhost listener port in addition to existing permissions.

Cloud benchmark results estimate comparable token OPEX using a dated public
Standard API rate card. The estimate prices uncached input, cached input, and
generated output separately, and records its rate-card identity with the result.
Reasoning effort does not change a model's unit rates; observed token usage
determines the per-run difference. Gemini thinking tokens are charged as output;
Codex reasoning tokens are already included in its output-token counter. Local
model hardware CAPEX remains unpriced because it depends on the user's hardware
and allocation policy.

As of 2026-10-17, the estimator uses these Standard text-token rates per million. OpenAI rates use short-context pricing; GPT-6 Sol and GPT-6.1 Sol are separate model IDs with different cached-input prices:

| Target model | Uncached input | Cached input | Output |
| :--- | ---: | ---: | ---: |
| GPT-6 Astra | $10.00 | $1.00 | $50.00 |
| GPT-6 Sol | $2.00 | $0.20 | $10.00 |
| GPT-6.1 Sol | $2.00 | $0.10 | $10.00 |
| GPT-6 Luna | $0.10 | $0.01 | $0.50 |
| GPT-5.6 / GPT-5.6 Sol | $4.00 | $0.40 | $20.00 |
| GPT-5.6 Terra | $2.00 | $0.20 | $12.00 |
| GPT-5.6 Luna | $0.20 | $0.02 | $1.20 |
| GPT-5.6 Cyber | $12.50 | $1.25 | $75.00 |
| Gemini 3.8 Flash | $0.75 | $0.075 | $3.75 |
| Gemini 4 Argon | $2.00 | $0.10 | $10.00 |

The formula is `uncached_input × input_rate + cached_input × cached_rate +
billed_output × output_rate`, with token counts divided by 1,000,000. The
Gemini Flash introductory rates are published through 2026-12-31, and Gemini 4 Argon introductory rates through 2026-11-30; the estimator switches to their published global Standard rates on the following dates. Google Agent Platform lists non-global rates 10% above global for these models:

| Target model | Period | Non-global input | Non-global cached input | Non-global output |
| :--- | :--- | ---: | ---: | ---: |
| Gemini 3.8 Flash | Introductory | $0.825 | $0.0825 | $4.125 |
| Gemini 3.8 Flash | Standard from 2027-01-01 | $1.65 | $0.165 | $8.25 |
| Gemini 4 Argon | Introductory | $2.20 | $0.11 | $11.00 |
| Gemini 4 Argon | Standard from 2026-12-01 | $4.40 | $0.22 | $22.00 |

Benchmark results do not record routing region, so estimates explicitly assume global routing. Gemini Batch and Flex rates are 50% of Standard rates, but benchmark runs use interactive requests and do not select those service tiers. The pricing ID and effective rate-card date are recorded with each estimate, and documentation renders these stored values instead of repricing historical runs. Future provider price changes still require a code update. The harness does not retain per-request context lengths to apply OpenAI long-context multipliers. Sources: [OpenAI API pricing](https://developers.openai.com/api/docs/pricing),
[GPT-6 model catalog](https://developers.openai.com/api/docs/models), [GPT-6 Sol pricing](https://developers.openai.com/api/docs/models/gpt-6-sol), [GPT-6.1 Sol pricing](https://developers.openai.com/api/docs/models/gpt-6.1-sol), [GPT-5.6 Sol pricing](https://developers.openai.com/api/docs/models/gpt-5.6-sol), [GPT-5.6 Terra pricing](https://developers.openai.com/api/docs/models/gpt-5.6-terra), [GPT-5.6 Luna pricing](https://developers.openai.com/api/docs/models/gpt-5.6-luna), [GPT-5.6 Cyber pricing](https://developers.openai.com/api/docs/models/gpt-5.6-cyber),
[Gemini API pricing](https://ai.google.dev/gemini-api/docs/pricing.md), [Gemini 4 Argon pricing](https://ai.google.dev/gemini-api/docs/latest-model), [Gemini 3.8 deprecation routing](https://ai.google.dev/gemini-api/docs/deprecations), and [Google Agent Platform Gemini pricing](https://cloud.google.com/gemini-enterprise-agent-platform/generative-ai/pricing).
macOS 26 requires read access to the exact root directory vnode for loader
startup; grant `(literal "/")` while retaining fixture and outside-file
containment checks.
OpenCode 1.18.30's attached client can exit successfully before its turn is
complete. Client exit alone cannot finalize benchmark usage. For a truncated
stream, wait for the retained server session to reach idle, then reconcile
persisted parts from newly created assistant messages. Exclude prior-turn
messages, deduplicate usage parts, retain completed tool outcomes and provider
errors, and reject missing terminal counters. The sandbox permits read and
execution access to the exact selected Go runtime and gopls dependencies;
Go caches and other writes remain inside the fixture.
Agy owns one process with stream-JSON input and output. Each prompt is a new
input message in the original conversation. Normalize cumulative usage into
per-turn differences and retain only newly appended transcript observations.

The primary evaluation matrix standardizes on three canonical arms:

1. `baseline-diff`: Vanilla LLM editing with native file manipulation / patch tools (no semedit tools).
2. `semedit-agentic`: Autonomous agentic AST discovery, navigation, and editing using semedit MCP tools.
3. `semedit-explicit`: Agent execution of an exact validated semedit recipe, with complete arguments, ordered batches, and no operation-selection search.

The legacy `--mcp-server-instructions=none,descriptive,prescriptive` Cartesian array is deprecated.
New benchmark plans and runner configurations default to `prescriptive` server
instructions. Explicit `none` passes empty `--instructions` to every coding
harness's MCP process, overriding the server default. Explicit `descriptive`
remains available. Historical records with missing instruction metadata retain
their original `none` interpretation.
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
non-resumable sessions keep the existing diagnostic skip behavior. When a semedit
task times out but its provider exposed a resumable identity and the parent job is
still active, make two best-effort diagnostic continuations on that same identity:
first ask where execution stalled using observed evidence, then ask for integration
feedback grounded in the partial run. Bound this work separately from the expired
task deadline, but honor parent cancellation. Store the timeout analysis in
`diagnostic_timeout_reflection`; neither reflection changes task measurements or
success. Only diagnostic-reflection turns may continue a failed provider session.
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
- New runtime defaults do not reinterpret historical instruction conditions.
- Explicit `none` disables server-wide guidance even when MCP startup defaults
  to prescriptive guidance.
- Follow-ups reuse the task session and workspace and never receive hidden oracle output.
- Every coding harness retains the original live session and MCP connection.
  A fresh backend process with the same saved identity does not satisfy continuity.
- Missing or changed continuation identities fail explicitly, and failed
  sessions cannot dispatch another turn or silently create a replacement.
- Provider cleanup precedes MCP evidence retention and workspace deletion.
- Diagnostic questioning never changes measured task aggregates.
- Cloud token estimates use explicit rate-card metadata and never imply actual
  invoice charges; unavailable token usage remains unavailable rather than zero.
- Transcript failures remain explicit; missing telemetry is not measured zero.
- Explicit recipes must pass the oracle when executed through semedit. Git fix
  replay cannot substitute for recipe validation or measured execution.
- Explicit wall time includes provider overhead; tool timing measures semedit
  execution. A known efficient recipe does not prove a global time minimum.
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
