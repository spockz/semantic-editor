<!-- Defines provider-independent benchmark execution and the boundaries adapters must implement. -->
# ADR-0050: Benchmark Planning, Sessions, and Arm Policy

* **Status**: Accepted
* **Date**: 2026-09-26

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Coordinate benchmark arms through persistent sessions, shared task orchestration, provider-owned adapters, and durable evidence. |
| description | The benchmark harness had separate single-run and matrix execution paths. Fixture expansion could silently omit failures or apply a large-context overlay to archives with only one supported context. Provider selection was repeated across task turns and diagnostic reflections. Fixed semantic-arm steering also combined source-inspection and modification restrictions without recording the choice as an experimental condition. |
| role | process |
| scope | tooling |
| area | benchmark |
| languages | generic |

## Context

The benchmark harness had separate single-run and matrix execution paths.
Fixture expansion could silently omit failures or apply a large-context overlay
to archives with only one supported context. Provider selection was repeated
across task turns and diagnostic reflections. Fixed semantic-arm steering also
combined source-inspection and modification restrictions without recording the
choice as an experimental condition.

Subsequent adapter additions exposed recurring boundary problems: mutable result
objects carried session configuration, setup and cleanup were optional interface
extensions, and adapter capabilities were repeated across dispatch switches.
Diagnostic questions acquired dedicated result fields and publication exclusions.
The adapter review also identified lost continuation latency, unmeasured failed
startup, and cleanup errors hidden inside wrapped process-exit errors. Shared
function placement alone does not enforce the desired contracts.

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

### Adapter contract and ownership

The following boundaries are the accepted target architecture. They apply to
every adapter, including future adapters, rather than depending on copied
provider-specific guards.

| Owner | Responsibility |
| :--- | :--- |
| Planner and scheduler | Resolve experiment conditions, validate targets and capabilities, prepare jobs, and dispatch each job once. |
| Adapter descriptor | Declare the factory, target validation, transport identity, MCP instruction support, and local-model routing capabilities in one authoritative definition. Shared callers consume it instead of maintaining parallel harness-name switches. |
| Shared session lifecycle | Bind identity, sequence work and diagnostics, select follow-ups, apply deadlines, run acceptance checks, aggregate measurements, retain evidence, and coordinate shutdown. |
| Provider adapter | Translate frozen configuration into native runtime configuration, own the native process/conversation, submit prompts, decode protocol evidence, normalize usage, and reap owned resources. |
| Reporting and publication | Render generic diagnostic records separately and exclude the entire diagnostic container from measurement aggregation. |

Require preparation, session startup/turn execution, and shutdown through an
explicit lifecycle interface. An adapter with no preparation work must provide
an explicit no-op implementation. Required setup or cleanup must not disappear
because an optional type assertion fails. A new adapter supplies its descriptor,
lifecycle implementation, and protocol tests; it does not modify follow-up,
grading, aggregation, diagnostic, or publication policy.

Separate immutable session configuration from turn observations. The shared
wrapper binds workspace, model, effort, arm, and runtime policy once. Turn inputs
contain the prompt and shared phase, not replacement session configuration.
Adapter outputs contain normalized observations and native identity evidence;
the aggregate `RunResult` is not an adapter configuration or mutable output sink.
Shared guards reject identity changes, invalid phases, aggregate reuse, and work
after diagnostics before provider dispatch. Native protocol checks supplement
these guards instead of replacing them.

Represent lifecycle invocations with one shared, typed command structure.
`newInitialPromptCommand` creates the first task prompt without a prior session
identity; `newFollowUpCommand` requires the original session identity and marks
later task or diagnostic prompts as follow-ups. Each command carries the prompt,
turn classification, command kind, and continuation identity needed by the
shared lifecycle. The shared lifecycle selects the initial command once and a
follow-up command for every later prompt, including diagnostics, then passes it
through the adapter contract. Each adapter maps that common request into its
backend-specific CLI arguments or protocol request, environment, working
directory, sandbox, and runtime options. Persistent-protocol adapters may send
a request through an existing process rather than launch a process per turn.
Invocation sites must not bypass the shared lifecycle. Tests must prove that
initial commands do not carry a prior session identity, follow-up commands
require and preserve it, and diagnostics use the follow-up form.

Use one shared task-step mechanism for provider invocation, acceptance checks,
per-turn accounting, and evidence retention. Corrective and staged modes differ
in prompt/checkpoint selection, stop conditions, and overall success policy.
Keep those policies explicit without duplicating the task-step implementation
inside modes or adapters. Tests must exercise the production adapter path;
test-only legacy process launchers must not become parallel execution contracts.

Adapters report successful process launch immediately, including when later
initialization, readiness, authentication, or model selection fails. No launched
process means zero agent wall time; failure after launch retains elapsed work
time. Shared aggregation defines each field's semantics: additive counters are
summed, initial-context measurements remain initial, and startup latency first
observed during measured follow-up work is retained without overwriting an
earlier observation. Missing measurements remain distinct from measured zero.
Diagnostic observations never participate in these reductions. Detailed phase
and evidence boundaries are defined in
[ADR-0076](0076-work-and-diagnostic-measurement-boundaries.md).

Represent diagnostic questions as global definitions with stable identifiers,
eligibility/order policy, prompt, and time budget. Store their outcomes in one
generic diagnostic collection rather than adding a result field and rendering
branch for each question. The collection holds explanations and capture errors;
classified transcripts retain diagnostic measurements separately. Exclude that
container structurally from benchmark metrics and publication, without a list
of individual question names. Adding a question changes global policy and tests,
not adapters or the report schema. Additional native protocol capabilities remain
adapter work and must be advertised explicitly. Version representation changes
and preserve interpretation of historical reports at the read boundary, without
parallel internal diagnostic paths or dual writes.

Shutdown distinguishes native process exits from cleanup failures through
structured outcomes. Attribute diagnostic exits to the dispatched diagnostic
turn, while preserving every genuine cleanup or evidence failure. Finding an
exit error inside a wrapped or joined error does not authorize suppressing its
sibling failures. A failed diagnostic does not change accepted work; later
questions record failure or skip status without silently restarting the provider.

Keep protocol decoding and cumulative-to-delta normalization in adapters and
measurement reduction in shared code. Preserve raw evidence without repeatedly
counting or reparsing prior observations. Document instrumentation and persistence
costs against the measurement boundary when changing transcript storage.

Maintain reusable lifecycle conformance cases and run them through each actual
adapter with controlled native-process fixtures. Cover all arms, initial work,
corrective and staged continuation, diagnostics, usage availability, terminal
identity, failure after launch, follow-up startup latency, cancellation, and
shutdown with wrapped/joined exit and cleanup errors. An unguarded fake adapter
can prove shared guards, but cannot replace transport-specific conformance tests.
Keep the implementation map and adapter onboarding guidance synchronized.

### Remaining execution policy

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

MCP batch availability is a separate orthogonal boolean ablation condition
(`--semedit-batching`, default `false`). When enabled, the harness passes
`--enable-batching` only to semedit MCP servers; baseline arms are unchanged.
The setting is recorded in job results and included in job and pair identities,
so checkpoints cannot mix runs with different batch availability. Benchmark
plans select one value at a time; compare enabled and disabled runs explicitly.

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
Any non-use, batching, or tooling-improvement reflection has a separate diagnostic record and contributes
nothing to measured tokens, turns, tools, elapsed time, or oracle outcomes.
Every semedit-enabled run with a continuation identity receives an additional
tooling-improvement question after any non-use or batching question, whether the
task succeeded or failed. Ask for candid, serious feedback on the most useful
improvement, grounded in the agent's observed run; distinguish observed friction
from suggestions and allow the agent to say it lacked enough exposure. The
existing report representation stores the prompt, response, and capture error in
`semantic_integration_reflection` for compatibility. Render this as agent
opinion, never a performance measurement. If the provider supplies no resumable
identity, or the parent job is cancelled, record why feedback could not be
collected. When a semedit task times out but its provider exposed a resumable
identity and the parent job is still active, make two best-effort diagnostic
continuations on that same identity: first ask where execution stalled using
observed evidence, then ask for tooling-improvement feedback grounded in the
partial run. Bound this work separately from the expired task deadline, but honor
parent cancellation. Store the timeout analysis in
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

* Planning errors prevent all backend execution; intentional exclusions remain
  distinct from errors and empty plans are reported explicitly.
* The printed plan and execution consume the same jobs and resolved prompts.
* Unsupported contexts are never relabelled or populated with generic overlays.
* Every planned job has exactly one terminal result and advances progress.
* Per-job checkpoints use atomic writes; a resume with a mismatched plan fails before dispatch.
* Resume never repeats a saved terminal task failure and never treats signal cancellation as completed work.
* Agent pairs share experiment conditions but only semedit receives policy steering.
* Different policies, including historical unspecified policy, never merge silently.
* New runtime defaults do not reinterpret historical instruction conditions.
* Explicit `none` disables server-wide guidance even when MCP startup defaults
  to prescriptive guidance.
* Follow-ups reuse the task session and workspace and never receive hidden oracle output.
* Every coding harness retains the original live session and MCP connection.
  A fresh backend process with the same saved identity does not satisfy continuity.
* Missing or changed continuation identities fail explicitly, and failed
  sessions cannot dispatch another turn or silently create a replacement.
* Provider cleanup precedes MCP evidence retention and workspace deletion.
* Every adapter implements required lifecycle operations and has one capability
  descriptor; global policy does not branch independently on harness names.
* Session configuration and adapter observations are separate contracts.
* Initial and follow-up invocations use distinct shared command constructors
  and one common invocation representation; provider-specific translation
  stays in each adapter.
* Follow-up invocation requires the original provider session identity.
* Native process launch remains measured when subsequent initialization fails.
* Work reductions preserve measurements first observed during follow-up work.
* Diagnostic extension is global and uses one structurally excluded collection.
* Wrapped or joined process exits cannot hide genuine cleanup failures.
* Diagnostic questioning never changes measured task aggregates.
* Analysis exports read only explicitly selected reports or direct job
  checkpoints. Transcript, raw adapter events, adapter stages, and MCP metrics
  remain opt-in sidecars; missing or incomplete observations stay explicit.
* Interactive Codex chat export is opt-in through an explicitly selected
  Codex home. It reads only regular `.jsonl` and `.jsonl.zst` files beneath
  `sessions/` and `archived_sessions/`, streams records, and omits developer
  instructions and private assistant phases while retaining user messages,
  public assistant final/commentary messages,
  tool calls/results, and per-turn usage. It does not inspect Codex databases,
  settings, or other state; compressed transcripts require the `zstd` CLI.
  Output files remain atomic because a complete chat export can be large.
  `--limit` adds JSONL pages with a `page_info` record and opaque continuation
  cursor; continuation skips completed transcript files and resumes at a
  record ordinal in the active file, making exports practical for bounded
  agent context ingestion. Cursors require stable input files and ordering.
  `--since` accepts an inclusive RFC3339 timestamp and filters benchmark
  reports/checkpoints by report time and Codex records by event time; records
  without a timestamp are excluded when a cutoff is active.
* Benchmark improvement proposals must cite the prompt, tool arguments and
  outcomes, response/next-turn evidence when available, final diff, oracle
  result, and agent feedback. Capability gaps remain hypotheses until evidence
  distinguishes them from tool-selection or parameter mistakes.
* Evaluate tooling changes by rerunning the same task and experiment conditions;
  compare correctness, failed calls, interaction counts, latency, token use, and
  recorded cost before retaining a change.
* Cloud token estimates use explicit rate-card metadata and never imply actual
  invoice charges; unavailable token usage remains unavailable rather than zero.
* Transcript failures remain explicit; missing telemetry is not measured zero.
* Explicit recipes must pass the oracle when executed through semedit. Git fix
  replay cannot substitute for recipe validation or measured execution.
* Explicit wall time includes provider overhead; tool timing measures semedit
  execution. A known efficient recipe does not prove a global time minimum.
* External repository arms share a pinned base commit; acceptance patches remain
  outside the issue prompt and test patches stay outside the agent workspace.

The explicit analysis exporter produces a bounded evidence package for this
cycle. It does not classify an agent mistake or claim a capability gap on its
own; analysis must preserve the distinction between observed facts and causal
hypotheses.

## Consequences

Execution becomes inspectable and reproducible across narrow and broad selections.
Failures remain visible instead of disappearing from result totals. Policy studies
can vary inspection and editing constraints independently, at the cost of an
additional comparison dimension and explicit fixture context metadata. Legacy
reports remain publishable without claiming an unrecorded restriction policy.

The stronger adapter contract requires coordinated migration across adapters,
callers, tests, and report readers. Central descriptors reduce registration drift;
typed lifecycle and observation contracts make missing obligations visible to
the compiler. A generic diagnostic collection removes per-question schema and
publication edits, at the cost of an explicit report representation migration.

### Implementation status of the boundary amendment

The provider interface now requires adapter preparation, pre-run warmup, turn
execution, and shutdown. Preparation binds the session configuration once;
turn requests carry only the prompt, phase, and continuation identity. Typed
stage and turn outputs return measurements, provenance, evidence, and process
identity to the shared session, which records stage outcomes separately from
the turn transcript. The turn transcript stores the complete normalized
observation plus shared elapsed time and failure classification, rather than a
second hand-maintained subset of provider metrics. The shared session reads
continuation identity from the turn stage result so it cannot disagree with a
duplicate identity field. Generic repository hydration, fixture extraction,
dependency setup, semedit prewarm, and workspace snapshots remain in the shared
driver. Adapter configuration preparation follows fixture extraction, and
adapter-specific warmup follows generic dependency and semedit prewarm. The
shared reducer retains the first available startup measurements, including
values first observed on a follow-up turn. Structured shutdown outputs preserve
wrapped process exits and their cleanup failures; Copilot also reports a
successful launch when later initialization fails.

The single adapter descriptor, one common task-step implementation, and generic
diagnostic definitions/storage remain migration work. Existing provider protocol
helpers still use a temporary per-turn `RunResult` internally before translating
it into the typed observation, so that boundary has not completed its full
migration. The adapter lifecycle contract has shared fake-adapter coverage and
provider-specific process tests, but reusable conformance cases do not yet run
through every actual adapter. This status records implemented stages and open
work separately; it does not claim complete conformance. The common typed
invocation structure and distinct initial/follow-up command constructors now
route every adapter through the shared lifecycle. Provider-specific process
arguments and protocol requests remain adapter-owned; that translation is
intentionally not centralized. New invocation paths must continue to use the
shared lifecycle and command structure.

## Implementation guide

The [benchmark harness architecture guide](../../tools/benchmark-harness/README.md) maps this decision to current entry points, state ownership, source files, publication code, and regression tests. Start there when locating a change.
