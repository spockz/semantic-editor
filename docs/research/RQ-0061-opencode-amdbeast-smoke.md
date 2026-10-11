<!-- Records OpenCode model reachability, solving measurements, and live integration limitations. -->

# RQ-0061: OpenCode AMD Beast Benchmark Smoke

Status: Open
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

## Saluki transport investigation, 2026-10-11

The desktop now advertises
`Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest` through the same HTTP endpoint.
The smallest discovery smoke selects `task-01-rename-local`, small context,
one repeat, concurrency one, and baseline, implicit, and explicit arms.
The target includes the provider because its model ID contains a slash:
`opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest`.

Run `saluki-opencode-smallest-20261011084912` failed all three arms before a
session ID or model events were emitted. The endpoint's model listing worked
outside Seatbelt, while a numeric TCP probe inside the generated profile
returned `Operation not permitted`. These are transport failures with zero
observed semantic calls, not model-solving measurements. Evidence is retained
under `data/benchmarks/results/saluki-opencode-smallest-20261011084912`.

Two syntax checks on macOS 26.6.2 rejected exact IPv4 and IPv6 remote hosts.
`remote tcp` and `remote ip` require `localhost` or `*`; alternative predicates
`remote-ip`, `ip-address`, `socket-remote-address`, and `local-ip` are unbound.
A live wildcard-port probe reached both the desktop and loopback on port 1234;
a loopback-only rule reached loopback and denied the desktop. A wildcard rule
therefore cannot preserve the endpoint boundary. Probe commands are retained
in `.scratch/seatbelt-endpoint-probes.sh`.

[ADR-0067](../adr/0067-opencode-run-owned-model-relay.md) selects one
fixed-destination loopback relay per benchmark run, shared by its jobs and
closed at run completion. Separate runs receive OS-assigned ports.

## Saluki relay rerun, 2026-10-11

Run ID: `saluki-opencode-relay-20261011T095400Z`. Fresh binaries followed a
passing `make check`; focused OpenCode, relay, sandbox, session, and Make target
tests passed. The same smallest three-arm plan ran through the run-owned relay.
All three generated profiles used relay port 53477, with distinct session
ports. A TCP probe after run completion returned `Connection refused` on 53477,
confirming listener cleanup.

| Arm | Observed tool work | Task seconds | Recorded result |
| :--- | :--- | ---: | :--- |
| Baseline | Five native calls, including two successful edits | 19.752 | Failed: incomplete terminal usage |
| Implicit semedit | 32 calls; two semantic rename attempts failed | 258.206 | Failed: incomplete terminal usage |
| Explicit semedit | 32 calls; semantic lookup and outline succeeded; two rename attempts failed | 208.970 | Failed: incomplete terminal usage |

These elapsed values describe failed attempts, not successful solving
measurements. The adapter correctly rejected all three streams because they
lacked complete terminal token usage; acceptance oracles were not evaluated.
The baseline transcript ends after a step start despite OpenCode exiting zero.
The relay remained active for the later arms, so this is not relay teardown or
a general egress denial. The evidence does not yet identify whether OpenCode's
attached-client lifecycle or the provider caused the missing terminal event.

Both semantic arms connected to MCP. Successful explicit lookup and outline
confirm tool discovery and execution for those reads. Every rename failed with
`check Go diagnostics with gopls: resolve Go diagnostics module: exit status 2`;
the tool response omits the underlying process diagnostic. A separate `go env
GOMOD` probe under the active fixture profile succeeded, so the precise Go
diagnostics failure remains unresolved. The failures are recorded in
[the tool log](../SUBOPTIMAL_TOOLS.md).

[The retained report](../../data/benchmarks/results/saluki-opencode-relay-20261011T095400Z/task-01-rename-local/opencode-amdbeast-Kujira-Underdog-Saluki-27B-1.0-MTP-GGUF:latest-repeat-1-mcp-server-instructions-prescriptive-semedit-arm-restrict-write.md)
contains partial telemetry. Do not publish those counters as complete model
measurements or interpret this run as an acceptance pass.

## Terminal and Go runtime diagnosis, 2026-10-11

OpenCode 1.18.30's attached client returns from `finish()` before awaiting its
completion loop. A successful client exit therefore does not establish turn
completion. The harness retains strict usage validation and recovers the
missing tail from the owned server after the session becomes idle, excluding
assistant messages from previous turns and deduplicating usage parts.
See the versioned [client source](https://raw.githubusercontent.com/anomalyco/opencode/v1.18.30/packages/opencode/src/cli/cmd/run.ts)
and [session status source](https://raw.githubusercontent.com/anomalyco/opencode/v1.18.30/packages/opencode/src/session/status.ts).

The retained implicit-arm shell output identifies the rename failure:
`go: cannot find GOROOT directory` names the repository's cached Go 1.27.2
toolchain, which the fixture profile cannot read. The inherited `PATH` and
`GOROOT` selected that runtime. The selected `gopls` wrapper also needs its
backing executable and canonical shell interpreter; those reads were denied.
These are runtime containment failures before source diagnostics or rename
execution. The shared diagnostics manager now includes Go stderr while
preserving the wrapped process error, and regression tests verify failed
diagnostics leave the source unchanged.

The corrected smallest smoke run, `saluki-opencode-terminal-runtime-20261011T110500Z`,
exited successfully with one repeat of all three arms. `make check` and the
focused OpenCode suite passed. Every arm completed in one solving turn with
complete terminal usage, and both semantic arms verified MCP use and a
successful rename.

| Arm | Task seconds | Semantic calls | Acceptance |
| :--- | ---: | :--- | :--- |
| Baseline | 31.409 | None | Pass |
| Implicit semedit | 45.782 | Outline and rename succeeded | Pass |
| Explicit semedit | 36.766 | Lookup and rename succeeded | Pass |

These are single smoke samples, not comparative performance conclusions.
The baseline also built and tested the fixture inside Seatbelt, confirming the
selected toolchain's read and execution permissions. The corrected profile
grants access to the exact selected Go runtime and gopls wrapper dependencies,
while retaining fixture-only writes and selected-port networking.

## Hard and SWE Saluki rerun, 2026-10-11

The larger suites reopen this investigation: successful semantic execution is
established, but preparation, sandbox runtime access, and provider cleanup still
prevent clean measurements. The earlier smallest smoke results remain valid.

Configuration: OpenCode with
`amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest`, one repeat of each
arm, concurrency one, small context, a five-minute solving deadline,
prescriptive MCP instructions, semantic write restriction, and no skill
delivery. The loaded endpoint advertises a 65,536-token context; each generation
has an 8,192-token output allowance. One relay is shared by each run's jobs.

Hard artifacts: `data/benchmarks/results/hard-smoke-saluki-fixed-20261011T165800Z`.
SWE artifacts: `data/benchmarks/results/swe-smoke-saluki-fixed-20261011T165800Z`.
Each job's `provenance.session_transcript` identifies its retained transcript
outside the disposable fixture. These OpenCode jobs create no Codex sidebar
projects. `make check` passed before the rebuilt binary and both suites ran.

| Hard task | Baseline | Implicit semedit | Explicit semedit |
| :--- | :--- | :--- | :--- |
| Composite refactor | Pass, 28.1 s | Output limit, 216.1 s | Pass, 88.1 s |
| Mixed sink migration | Pass, 194.5 s | Pass, 133.7 s | Pass, 107.5 s |
| Request identity | Pass, 178.5 s | Output limit, 269.7 s | Deadline, 300.0 s |

All six hard passes succeeded at the first acceptance check. Composite implicit
generated repetitive tool namespace text. Identity implicit reached a plan but
continued reconsidering it until output exhaustion, without making a semantic
call. Raising the output limit is not a demonstrated remedy. Explicit identity
completed rename and sentinel edits but did not finish normalization before the
deadline; its method replacement attempts used `kind: "function"`, which was
correctly rejected. The subsequent statement edit succeeded.

The mixed sink task used 32 baseline calls, 16 implicit calls, and 13 explicit
calls. Both semantic arms recovered from incorrect arguments using tool
feedback. These are single samples, not performance estimates, and setup and
diagnostic reflections are excluded from the timed task measurements.

| Suite | Arm | Successful runs | Acceptance passes | Semantic use observed |
| :--- | :--- | ---: | ---: | ---: |
| Hard | Baseline | 3/3 | 3/3 | N/A |
| Hard | Implicit semedit | 1/3 | 1/3 | 2/3 |
| Hard | Explicit semedit | 2/3 | 2/3 | 3/3 |
| SWE | Baseline | 0/5 | 0/5 | N/A |
| SWE | Implicit semedit | 0/5 | 0/5 | 1/5 |
| SWE | Explicit semedit | 0/5 | 2/5 | 3/5 |

Acceptance pass counts use the whole planned suite as the denominator. A
timeout without grading is acceptance-unknown, not an evaluated rejection.
Implicit SWE is serialized as `semedit-agentic`; the hard equivalent is
`semedit`. Semantic use includes failed calls and inspection, not just edits.

| SWE task | Baseline | Implicit semedit | Explicit semedit |
| :--- | :--- | :--- | :--- |
| gRPC #2735 | Deadline, 300.0 s | Deadline, 300.0 s | Acceptance pass; cleanup error, 316.5 s |
| gRPC #2951 | Deadline, 300.0 s | API socket closed, 251.0 s | Deadline, 300.0 s |
| gRPC #3361 | Deadline, 300.0 s | Deadline, 300.0 s | Deadline, 300.0 s |
| go-zero #2463 | Failed first test check, then deadline, 300.0 s | Deadline, 300.0 s | Nested-module dependency failure, then deadline, 300.0 s |
| go-zero #964 | Deadline, 300.0 s | Deadline, 300.0 s | Acceptance pass; cleanup error, 190.7 s |

Both explicit acceptance passes completed after one successful semantic batch
and passed their first oracle. The solving deadline was met in #2735; its final
oracle extended total task time beyond 300 seconds, as permitted by the shared
lifecycle. In #2463 baseline, the first acceptance check failed on missing
logging APIs; the second interactive step timed out. Remaining SWE jobs never
reached an acceptance check. The 15 SWE runs contain twelve deadline failures,
one API transport failure, and two cleanup failures.

Across both suites, eight solutions passed acceptance but only six runs
completed successfully. One solution failed an intermediate acceptance check;
fifteen jobs have no acceptance verdict. The previous larger smoke had hard
6/9 and SWE 0/15 successful runs. The new strict totals are unchanged, with two
SWE acceptance passes now visible. Different individual outcomes in this
single-repeat rerun do not establish a causal performance improvement.

Retained passing-but-invalidated transcripts:

- [gRPC #2735 explicit](../../.scratch/benchmarks/transcripts/run_opencode_semedit-explicit_small-explicit_grpc__grpc-go-2735_1791739970636830000.json)
- [go-zero #964 explicit](../../.scratch/benchmarks/transcripts/run_opencode_semedit-explicit_small-explicit_zeromicro__go-zero-964_1791743589500670000.json)

### Confirmed environment and measurement faults

- The generated Seatbelt profile denies the selected Xcode/Command Line Tools
  root. Host Git works and the developer directories exist. A sandboxed
  `git --version` reproduces the job's `xcrun` error when using the inherited
  `DEVELOPER_DIR`. The fix must resolve the selected developer root outside
  the sandbox, pass it explicitly, and permit only that root and necessary
  metadata. This affected hard identity and SWE gRPC #3361.
- Explicit gRPC #2735 and go-zero #964 passed every oracle level after one
  successful semantic batch each. Both then failed with `reap OpenCode server process group:
  operation not permitted`. The error originates from signaling the group
  after `cmd.Wait()` reaps its leader. Do not suppress this error: descendant
  cleanup must be proven, and a Darwin sandbox regression is needed.
- Root-only `go list -mod=readonly -deps -test ./...` preparation does not cover
  nested modules. Explicit go-zero #2463's batch diagnostics entered
  `tools/goctl`, attempted to download `github.com/emicklei/proto@v1.10.0`, and
  failed because the sandbox denies that network access. Preparation must
  cover all modules used by diagnostics without mutating workspace manifests.
- The oracle copy excludes `.scratch` and obtains a new fixture-local Go cache.
  It therefore cannot reuse the prepared cache. The #2463 baseline acceptance
  log contains dependency downloads in the timed oracle. Reuse a prepared
  dependency cache or prepare the grader's isolated cache before solving;
  preserve hidden-test isolation.
- SWE semantic warmup reports `no_go_target` because these fixtures have no
  AST target metadata, despite containing Go. Do not describe those jobs as
  prewarmed; generic setup needs a target-discovery policy.

### Other failures and interpretation limits

Implicit gRPC #2951 reports an unexpectedly closed API socket. The desktop
endpoint subsequently returned a healthy status, but the recorded evidence
does not identify which transport component closed the connection. The relay
currently discards per-connection dial/copy errors, limiting attribution.

The harness fetched each pinned repository before execution. Failed GitHub
fetches in #2735 and #964 were model-initiated upstream lookups, not fixture
hydration. Provide required reference material locally and state the offline
boundary in the model prompt. Do not substitute a later upstream solution for
the pinned benchmark code.

Timeout and transport failures retain incomplete usage counters. Zero or small
recorded counts cannot establish zero or small consumption. The five-minute
deadline also makes unfinished work an acceptance-unknown outcome, rather than
proof that a completed solution is wrong. The run is diagnostic evidence for
the harness and model's tool-use behavior; it is not a clean model comparison.
