<!-- Records OpenCode model reachability, solving measurements, and live integration limitations. -->

# RQ-0061: OpenCode AMD Beast Benchmark Smoke

Status: Resolved
Date: 2026-10-03

## Question

Can OpenCode run the host's `qwen36-iq4-xs` model through the benchmark harness
with concurrency one, faithful solving metrics, and diagnostic follow-ups
excluded from those metrics?

## Experiment

Target: `opencode/qwen36-iq4-xs`, with no effort suffix or variant flag. The
adapter selects `amdbeast` internally; benchmark identities retain the short
target and explicit served model ID.
Use paired baseline and semedit arms, one repeat, small context, write restriction,
and no server-wide Model Context Protocol (MCP) instructions.

Run local rename first as the simplest editing task. Then run composite refactor,
mixed sink API migration, and request identity migration as the three complex
tasks: they require coordinated API, behavior, or cross-package changes rather
than repeated mechanical renames. The overhead-only greeting probe is outside
this editing selection. Commands are in the [harness guide](../../tools/benchmark-harness/README.md#opencode-amd-beast-smoke-run).

## Preparation and earlier failed attempts

- The installed OpenCode command-line interface (CLI) reports version 1.18.30.
- Both repository and user OpenCode configuration identify the provider as
  `amdbeast`, model key `qwen36-coder`, and endpoint
  `http://192.168.1.122:1234/v1`.
- OpenCode started a benchmark session and logged selection of that provider
  and model without a thinking variant. Its API calls repeatedly failed with
  connection errors. Independent endpoint probes also refused connections
  outside the sandbox. Those attempts produced no successful solving measurement.
- The interrupted attempt selected `small:default`, which excludes task 01
  because it then had no named prompt variants. Its artifacts remain in
  `.scratch/opencode-failed-initial`, with provider evidence under
  `.scratch/benchmarks`; they are outside the published result directory.
- Review found that the existing OpenCode adapter ignored all token usage.
  Preparation adds step usage accounting, completeness checks, and process
  regressions for correction and diagnostic isolation under ADR-0039/0050.

The user authorized stopping while the model was unreachable, then requested
fresh binaries and a rerun once the host was available.

## 2026-10-04 rerun preparation

The user confirmed that the model host serves HTTP and requested fresh binaries
and a new smoke run. Initial local probes failed before HTTP with `No route to
host`, despite a host route through `en0`. A 404 for the bare `/v1` path does not
establish failure of `/v1/models` or `/v1/chat/completions`.

The rebuilt harness exposed that task 01 had no named prompt variants and was
excluded even with `--variants small`. It now declares its existing instruction
as the default prompt, matching the planner's fixture contract. The documented
simple run can therefore schedule both arms. No zero-job output counts as a
successful model benchmark.

Enabling Codex in macOS Local Network settings restored endpoint access.
`/v1/models` returned `qwen36-iq4-xs`, owned by `llamacpp`, with an IQ4_XS
quantization label and context capacity 131072. The smoke run uses that explicit
model ID rather than the earlier `qwen36-coder` alias. Both freshly built
binaries followed a passing `make check`.

## Token semantics

OpenCode 1.18.30's [usage normalization](https://github.com/anomalyco/opencode/blob/v1.18.30/packages/opencode/src/session/session.ts)
subtracts cache read/write from input and reasoning from output. Reconstruct
total input as input plus both cache buckets, preserve cache reads separately,
and include cache writes in uncached input. Keep output and reasoning separate.
The [JSON event loop](https://github.com/anomalyco/opencode/blob/v1.18.30/packages/opencode/src/cli/cmd/run.ts)
emits root-session step finishes and omits child-session usage. Disable delegated
agent tasks in both arms rather than publishing incomplete solving counts.
Deduplicate by step part ID, not assistant message ID, because several paid
model steps can share one message.

## Live result, 2026-10-04

Run ID: `opencode-qwen36-iq4-xs-smoke-20261004`. Eight jobs ran in the declared
order with concurrency one. All four baseline arms and the first three semedit
arms passed their acceptance oracles. The request-identity semedit arm reached
the twelve-minute job deadline without a terminal usage record; its partial
counters remain failure evidence, not a complete solving measurement.

| Task | Arm | Outcome | Input tokens | Output tokens | Task seconds |
| :--- | :--- | :--- | ---: | ---: | ---: |
| Local rename | baseline | Pass | 37,962 | 845 | 37.313 |
| Local rename | semedit | Pass | 502,103 | 2,451 | 128.800 |
| Composite refactor | baseline | Pass | 62,995 | 1,390 | 56.663 |
| Composite refactor | semedit | Pass | 433,483 | 2,564 | 134.801 |
| Mixed sink migration | baseline | Pass | 137,323 | 4,093 | 140.104 |
| Mixed sink migration | semedit | Pass | 738,175 | 2,501 | 179.266 |
| Request identity migration | baseline | Pass | 803,739 | 8,656 | 329.986 |
| Request identity migration | semedit | Timeout; incomplete usage | n/a | n/a | n/a |

Input includes cache reads and writes. Serialized results retain separate
cached and uncached counters; reasoning counters were zero. Fixture reports
are saved under `data/benchmarks/results/opencode-qwen36-iq4-xs-smoke-20261004`.
The combined JSON/Markdown reports, raw transcripts, and independent audit
outputs remain in `.scratch`.

Reconstructing distinct step usage from each task transcript exactly matches
all recorded counters. Seven runs have complete terminal measurements; the
eighth explicitly fails completeness. Task turn and tool counts also match.
The mixed-sink semedit run has two solving turns, so its corrective turn is
included. Six diagnostic turns contributed 218,035 input and 1,508 output
tokens that appear only in diagnostic transcripts. They do not enter any task
totals. Timestamp checks place each task clock after the last solving response
and before the first diagnostic process where one exists.

This resolves the harness and measurement question: OpenCode reaches the
explicit served model and produces valid solving measurements with diagnostic
isolation. It does not establish a general semedit performance benefit. Live
schema validation failures masked underlying operation errors, and the local
rename used a direct-edit fallback. Construct mutations did succeed in the
composite and mixed-sink tasks. Some later diagnostic opinions incorrectly
claimed that every semantic call failed; raw tool records take precedence.
The final semedit attempt repeated 70 rejected construct calls before its
deadline. These failures are recorded in [the tool log](../SUBOPTIMAL_TOOLS.md).

## Task 004 follow-up, 2026-10-04

Run ID: `opencode-qwen36-iq4-xs-task004-20261004-1622`. The requested
`task-04-insert-public` fixture ran with the same target, small/default context,
one repeat, concurrency one, no thinking variant, and a twelve-minute limit.
Fresh binaries followed a passing `make check`. Both arms passed on their first
solving turn, including constructor placement, non-nil return, compilation,
tests, and protected-file checks.

| Arm | Input tokens | Cached input | Uncached input | Output tokens | Task seconds |
| :--- | ---: | ---: | ---: | ---: | ---: |
| baseline | 46,635 | 39,045 | 7,590 | 921 | 43.163 |
| semedit | 194,396 | 162,126 | 32,270 | 952 | 83.381 |

The semedit task used one successful `semantic_insert_construct` call with
`kind=function` and placement before `internalRun`. It reported no functional
tool failures or schema errors. The model's two diagnostic answers confirmed
the insertion and recommended the tool based on this run; these remain agent
opinions.

Transcript audits match all task token buckets, turn counts, and tool counts,
and confirm complete terminal usage in both arms. Timestamp checks confirm
that diagnostic questioning starts after the measured task clock ends. The
two diagnostic turns used 66,288 input and 431 output tokens, excluded from the
table and all task totals. Results are saved under
`data/benchmarks/results/opencode-qwen36-iq4-xs-task004-20261004-1622`.

## Three difficult tasks repeat, 2026-10-04

Run ID: `opencode-qwen36-iq4-xs-hard-20261004-1635`. The user requested another
attempt on composite refactoring, mixed-sink migration, and request-identity
migration after task 004. The run retains `opencode/qwen36-iq4-xs`, small/default
context, one repeat, concurrency one, write restriction, no server-wide MCP
instructions, no thinking variant, and a twelve-minute job deadline. Fresh
binaries followed another passing `make check`.

| Task | Arm | Outcome | Input tokens | Output tokens | Task seconds |
| :--- | :--- | :--- | ---: | ---: | ---: |
| Composite refactor | baseline | Pass | 54,563 | 1,196 | 65.646 |
| Composite refactor | semedit | Timeout; incomplete usage | n/a | n/a | 719.918 |
| Mixed sink migration | baseline | Pass | 105,719 | 4,337 | 130.400 |
| Mixed sink migration | semedit | Pass; direct-edit fallback | 1,074,059 | 4,010 | 174.911 |
| Request identity migration | baseline | Pass | 248,625 | 9,478 | 242.302 |
| Request identity migration | semedit | Pass; no semantic calls | 488,456 | 13,245 | 367.750 |

All successful arms passed in one solving turn. Task 09's semedit arm made two
successful construct mutations, but ultimately repeated 227 rejected undo
requests. Its 239 tool calls include 234 failures. The advertised MCP output
schema rejects error responses lacking `result`, masking the underlying
operation diagnostics. Its timeout duration and partial counters are retained
as failure evidence, not as a complete solving measurement.

Task 11's semedit arm made two successful semantic insertions, seven rejected
construct replacements, and two rejected batch requests before using direct
edits. Task 12's treated arm used fourteen direct edits and no semantic tools.
Its diagnostic opinion called the migration straightforward, despite the
fixture's identity distinctions. These results establish acceptance success
under the selected arm prompts; they do not establish successful semantic-only
execution or a semantic-tool performance benefit. Tool records take precedence
over task 11's diagnostic claim that all semantic calls failed.

Independent transcript audits match all recorded token buckets, turn counts,
and tool counts. Five runs have complete terminal measurements; task 09's
timeout explicitly fails completeness. Four diagnostic turns used 172,256
input and 941 output tokens, excluded from all task totals. Timestamp audits
confirm that diagnostic questioning begins after each measured solving clock.
Failures are recorded in [the tool log](../SUBOPTIMAL_TOOLS.md). Per-fixture
reports are saved under
`data/benchmarks/results/opencode-qwen36-iq4-xs-hard-20261004-1635`; the combined
reports, raw transcripts, and independent audits remain in `.scratch`.
