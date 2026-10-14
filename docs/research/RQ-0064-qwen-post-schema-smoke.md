<!-- Records repeated Qwen benchmark evidence after MCP output-contract repairs. -->

# RQ-0064: Qwen Smoke After MCP Schema Repairs

Status: Open
Date: 2026-10-05

## Question

After rebuilding and promoting the repaired MCP server, does
`opencode/qwen36-iq4-xs` use semantic tools reliably across ten independent
iterations, and what do its diagnostic self-reflections report?

## Method

Repeat tasks 01, 09, 11, and 12 from the original smoke with paired baseline and semedit
arms, small/default context, concurrency one, twelve-minute job timeouts,
write-only prompt steering, and no server-wide MCP instructions. OpenCode's
adapter does not install a native permission policy for this flag, so native
editing remains available. This measures tool adoption under a prompted
preference, not compliance with an enforced semantic-only mutation gate.
Each trial starts an
independent fixture workspace and provider session. Preserve the existing
OpenCode model mapping and use the explicit served model identifier.

Promotion runs the full repository checks before replacing the stable binary.
Record hashes for the stable binary, development binary actually used by the
harness, and rebuilt harness. Record exact invocation and terminal reports.

Audit measured task turns separately from diagnostic reflections. Count
semantic selections, successful semantic calls, failed semantic calls, direct
editing calls, and exact schema-validation errors from raw events. Reconstruct
usage and terminal capture independently. Report agent opinions separately
from observed tool outcomes. Missing reflection on failed or timed-out runs
does not indicate a positive or negative recommendation.

The intervening task 12 oracle edits only regroup imports and do not change
acceptance criteria. Historical single-trial measurements provide context,
rather than a controlled ten-trial comparison against the old binary.

## Evidence

Promotion passed `GOFLAGS=-p=2 GOMAXPROCS=2 make promote bench-build`, including the full
race-enabled test suite, linting, and generated documentation checks. The
harness was rebuilt with `make bench-build`.

- Stable and development binary SHA-256:
  `8119878c0486579e57f89c1d11b8a7f152c22d5eea57959f4417e232d8224434`.
- Harness SHA-256:
  `9b6e0b3f3f786895237c35e83b80769d5d1755c3660696217da5931f1e4e4678`.
- Source base revision: `d8ec4f5a18d0f91946f5d416e506135b33355fca`, with the
  tracked working changes captured locally in `.scratch/qwen-post-schema-source.diff`
  and untracked-file hashes in `.scratch/qwen-post-schema-untracked-sha256.txt`.
- OpenCode version: 1.18.30.
- Endpoint: `http://192.168.1.122:1234/v1`; served model: `qwen36-iq4-xs`.

The promoted-binary regression probe captured `tools/list` and raw responses,
then compiled those exact advertised schemas with the independent validator.
A successful insertion, the old replacement request missing `symbol`, and a
failed batch all passed output validation. The two failures now return valid
`isError:true` tool results rather than an output-schema rejection. The
replacement's input passes the advertised structural schema but fails the
operation's conditional requirement. Raw evidence is retained locally in
`.scratch/qwen-schema-probe/wire-exchanges-final.json`.

The first attempt, `opencode-qwen36-iq4-xs-post-schema-10x-20261005-0001`, was
interrupted after its first semedit trial exposed a missing gopls launch path.
That trial passed using sixteen semantic calls, with ten readable operation
errors and zero output-schema rejections. Its reflections complained about the
missing language server. This setup-invalid attempt is excluded from the
experiment and retained locally outside published result data.

The path-corrected attempt,
`opencode-qwen36-iq4-xs-post-schema-10x-20261005-0005`, used the installed
gopls v0.23.0 directory explicitly on the launch path. It was also interrupted:
the harness failed to recognize OpenCode's `semedit_semantic_*` prefix and
could send a false non-use reflection despite observed semantic calls. Its
first treated trial had three completed semantic calls and no schema failures
before interruption; those partial results are excluded. The prefix classifier
and reflection dispatch were repaired before the final ten-iteration run.

The harness now normalizes OpenCode's provider prefix consistently for tool-use
verification, expected-tool coverage, and reflection dispatch. Raw-event
regression tests cover a semantic call suppressing non-use reflection,
consecutive semantic calls triggering batch reflection, and an explicit batch
suppressing that question. Independent review and the full checks passed.

The final run is `opencode-qwen36-iq4-xs-post-schema-10x-20261005-0022`:

```bash
PATH="$(go env GOPATH)/bin:$PWD/.scratch/go/bin:$PATH" \
  .scratch/bin/benchmark-harness --target opencode/qwen36-iq4-xs \
  --tasks task-01-rename-local,task-09-composite-refactor,task-11-mixed-sink-api-migration,task-12-request-identity-migration \
  --variants small:default --repeats 10 --concurrency 1 --timeout 12m \
  --mcp-server-instructions none --semedit-arm-restrict write \
  --run-id opencode-qwen36-iq4-xs-post-schema-10x-20261005-0022 \
  --out-json .scratch/opencode-qwen36-iq4-xs-post-schema-10x-20261005-0022.json \
  --out-md .scratch/opencode-qwen36-iq4-xs-post-schema-10x-20261005-0022.md \
  --provenance semedit_sha256=8119878c0486579e57f89c1d11b8a7f152c22d5eea57959f4417e232d8224434 \
  --provenance harness_sha256=9b6e0b3f3f786895237c35e83b80769d5d1755c3660696217da5931f1e4e4678 \
  --provenance gopls_version=0.23.0
```

The model, task prompts, editing policy, and timeout are unchanged. Availability
of gopls and corrected tool-use accounting are explicit conditions of this run;
any improvement over historical single trials cannot be attributed solely to
schema repairs.

## Results

All eighty jobs completed: ten independent paired iterations per task. Both
arms passed 36/40 jobs. Tool adoption means at least one semantic call selected
during the task, including calls returning an operation error; successful batch
children are not counted as separate outer calls.

| Task | Baseline passes | Semedit passes | Semedit trials using tools | Completed semantic calls | Operation errors | Schema rejections |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 01: local rename | 10/10 | 10/10 | 10/10 | 20 | 0 | 0 |
| 09: composite refactor | 10/10 | 10/10 | 10/10 | 23 | 7 | 0 |
| 11: mixed sink migration | 9/10 | 7/10 | 10/10 | 30 | 256 | 0 |
| 12: request identity migration | 7/10 | 9/10 | 10/10 | 59 | 45 | 0 |
| Total | 36/40 | 36/40 | 40/40 | 132 | 308 | 0 |

Task 11's semedit failures comprise one oracle failure and two twelve-minute
timeouts. The remaining four baseline failures and task 12's semedit failure
are oracle failures. The model selecting tools does not establish that it chose
the correct operation or completed the task with those tools alone.

Tasks 01 and 09 used no native editing or shell source rewrites in the semedit
arm. Task 11 and task 12 include direct editing fallbacks. OpenCode's write flag
was prompt steering throughout this run; these mixed workflows are retained
in the results. Repeated wrong method-kind requests dominate task 11's error
count, including 167 identical attempts in one timed-out trial.

The 440 semantic calls contain zero observed client output-schema rejections.
The earlier schema mismatch is resolved in this experiment, including readable
failure results and partially applied failed batches. OpenCode transcripts
retain rendered tool text rather than a complete raw structured-content/schema
exchange, so the benchmark is evidence of no observed client rejection, not
independent validation of every retained wire payload. The promoted-binary
probe and the MCP conformance suite supply direct schema-validation evidence.

Two independent operation defects were identified during measurement: inspection emits a pointer-method
owner that statement insertion cannot reuse without rewriting its selector,
and generic declaration insertion can publish an import after a function
before reporting a formatting failure. These are logged in
`docs/SUBOPTIMAL_TOOLS.md`. The binary and experimental settings stayed fixed
during measurement; these defects were not repaired mid-run.

## Invocation audit and subsequent repairs

The follow-up audit reviewed actual requests and rendered results across all
forty semedit transcripts, grouping repeated attempts by invocation shape.
Four server defects are established by concrete calls; reflection opinions
and operation errors alone do not establish a defect.

| Invocation | Server defect | Regression requirement |
| --- | --- | --- |
| Task 11, `3xNC7tZ6lYwNLKK9FnSzqT7Lyi86fzLz` | Statement insertion rejects the canonical `Dispatcher.Record` owner produced by inspection for a pointer method. | Copy the inspection continuation unchanged into insertion; cover canonical and legacy pointer spellings and generic receivers. |
| Task 11, `GFBMhnI2HvHpEdAWE7NoxGHoyRWgRPaD` | An import inserted after a function is written before formatting reports illegal ordering. | Reject the combined invalid source before publication; preserve bytes and modification time through the API, CLI, standalone MCP, and batch MCP. |
| Task 12, `ftB7IjfGfJMcpqvIRperTywhhL4cJ2cK` | A move with `target_symbol: "Request.RequestID"` and omitted placement ignores the anchor and moves the whole type to file end. | Reject an incompatible placement anchor before mutation; preserve valid default and relative moves. |
| Task 12, `pLpqxVBBDUGaNYUEqb9YU4phAUdpOaBy` | A valid `var` declaration with a leading doc comment is rejected by lexical prefix checking. | Parse the declaration kind, preserve its comment, and reject wrong kinds or multiple declarations before mutation. |

The pointer selector and import publication regressions failed against the
measured implementation before their fixes. Selector resolution now uses the
shared symbol parser and AST receiver identity. Declaration publication parses
the complete proposed Go file before its atomic write, preserving the
pre-publication syntax invariant from ADR-0011.

The move parser and direct AST entry point now reject anchors with omitted,
file, or section placement. Batch parsing checks parameters before dispatch;
a malformed later child's move parameters therefore prevent earlier children
from running. This differs from a later execution failure after successful
children, which retains their work under ADR-0032.

Specific `var`, `const`, and `type` insertion now reuses the Go declaration
parser to require one declaration of the selected kind. Leading comments
survive insertion; wrong kinds and multiple declarations are refused.

Public CLI regressions cover all four fixes. Actual MCP tests independently
validate advertised response schemas for the inspection-to-insertion sequence
and standalone/batch import and move failures, including unchanged file bytes
and modification times. Focused regressions passed before the integrated check.
The move constraint is enforced by shared registry parsing and direct API
validation; the registry's flat input schema does not encode the conditional
dependency. Its parameter description states the accepted combinations.

Integrated verification passed `GOMAXPROCS=2 make GOFLAGS=-p=2 check`,
including race-enabled tests, schema assertions, lint, vulnerability checks,
and generated documentation verification. The log is retained in
`.scratch/qwen-invocation-fixes-check-2.log`. A separate fresh-binary probe
also confirms that the captured variable insertion with default grouping
and placement after `Request` preserves its attached doc comment.

Public executable regressions:

- [Pointer receiver continuation](../../testdata/scripts/statement_insert_pointer_receiver_continuation.txtar).
- [Import publication preflight](../../testdata/scripts/construct_import_preflight.txtar).
- [Move placement anchor](../../testdata/scripts/move_construct_target_symbol_contract.txtar).
- [Commented declarations](../../testdata/scripts/commented_declaration_insert.txtar).

These source
changes follow the frozen experiment; they do not change its recorded
outcomes, binary hashes, or pass rates.

Wrong `kind: "function"` requests for receiver methods, missing selectors,
malformed snippets, stale anchors, and a fabricated `snapshot_id: "latest"`
remain caller errors. ADR-0032 deliberately retains successful earlier batch
children when a later child fails. The model's intended field rename explains
its move misuse, but does not excuse silently ignoring a supplied parameter.

No retained invocation demonstrates an MCP write outside its benchmark
workspace. The native OpenCode shell did read outside assigned fixtures,
including parent-repository source and task 12 hidden tests; some runs then
built or tested that parent checkout. The pre-existing repository-root
`identityservice/` directory remains preserved.

The detailed audit is retained locally in
`.scratch/qwen-invocation-bug-review.json`. Corrections to the initial
commented-variable and move classifications are recorded in
`docs/SUBOPTIMAL_TOOLS.md`.

## Rerun after operation repairs

The requested follow-up repeats the frozen experiment's four tasks and ten
paired iterations using newly built and promoted binaries containing all four
operation repairs. The target, model identifier, endpoint, context, timeout,
concurrency, tool instructions, and editing policy remain the same.
Record the new binary hashes and run identity separately. Audit actual
invocations, tool adoption, schema rejections, the repaired failure shapes,
native fallback, and diagnostic self-reflections against the preceding run.

The promotion checks passed and the eighty-job run started:

- Run: `opencode-qwen36-iq4-xs-post-bugfix-10x-20261005-0546`.
- Stable and development binary SHA-256:
  `303dc6c19ce5c028387740f796ed7662225e9f82bbe3f62251e8dbfbdc5ab4a3`.
- Harness SHA-256:
  `55e1775f9febea8aefd92d0e51506c92b9feb2b92a2edc1e19cf29d9f177eb1d`.
- OpenCode 1.18.30 and gopls v0.23.0 remain unchanged.
- A separate isolated OpenCode preflight returned `READY` with complete
  terminal usage before measurement. This preflight is excluded from results.
- Promotion check log: `.scratch/qwen-post-bugfix-promote.log`.
- Run log and eventual aggregate reports use the run ID above under `.scratch/`.
- Source diff and untracked-file hashes were captured under that same prefix
  before launch.

The original 80-job scheduler was interrupted after 74 jobs, and its final six
task 12 jobs ran under a separate continuation ID with the same frozen
binaries, target, model, and settings. Across both logs, all 80 paired jobs are
accounted for: 70 raw oracle passes and 10 raw failures. No combined JSON report
was produced for the interrupted original run, so the two reports and raw
transcripts remain the source of truth. The execution scope violations below
make several apparent successes invalid as isolated correctness measurements.

The completed task 12 audit found two independent benchmark validity problems.
Native OpenCode tools read outside the assigned fixture, including the
pre-existing parent `identityservice/request/request_hidden_test.go`. One
baseline enumerated hidden-test filenames; another read hidden-test contents
and built/tested the parent checkout. Preserve the raw outcomes, but do not use
those trials as clean measurements of task correctness or tool adoption.

The task instruction also requires preserving existing tests, while its
executable mutation policy protects only `go.mod` and `go.sum`. Across the ten
task 12 semedit trials, eight raw oracle results pass and two fail. Only three
trials comply with the task's explicit test-preservation instruction; seven
trials made nineteen successful test-file writes in total. The oracle did not
enforce this instruction because the custom frontmatter parser ignored the new
`forbid_test_edits` scalar. The oracle now parses and enforces that flag. Do not
retroactively rewrite raw outcomes; rerun only after a verified process
boundary prevents outside reads.

## Confirmed regression fixes and remaining boundary

Invocation-level reproduction confirmed that adding an explicit import to a
file without an import block could transfer a function doc comment to the new
import declaration. The insertion now anchors the generated import AST
positions, with regressions for plain and quoted paths through the public
pipeline API, a partially successful MCP batch, and CLI txtar.

The task 12 mutation policy now parses `forbid_test_edits` and `Evaluate`
rejects modified, created, and deleted `*_test.go` files from the real
before/after workspace snapshot.

Fixture containment remains unresolved. A deny-by-default Seatbelt profile
fails to apply in this Codex app environment with `sandbox_apply: Operation not
permitted`. Automatic review also rejected wildcard-host outbound access because
it exceeded the configured model endpoint. The profile therefore denies all
network egress; OpenCode cannot reach the remote Qwen host under it. No updated
binary was promoted and no post-fix model smoke was run. A full `make check`
reached the suite but failed in OpenCode process tests when this environment
refused to apply Seatbelt. Focused pipeline, MCP batch, and oracle-policy tests
passed. The new development binaries compile, but the stable binary remains
unchanged pending a verified exact-host containment path.

## Self-reflections

All 36 passing semedit trials have an integration reflection. Four failed
semedit trials have none. There are seventeen additional batch reflections;
their diagnostic turns are excluded from task token and tool measurements.

Tasks 01 and 09 consistently recommend the tools for rename/refactoring work.
Task 11 is mixed: several trials praise diagnostics and batching, while others
criticize symbol matching, construct selection, and the need to use text edits.
One says it had to "abandon the semantic approach entirely"; its trace still
contains successful semantic edits and verification, so that wording overstates
the fallback.

Task 12 is also mixed despite nine passes. Trial 1 says semantic rename saved
it from "manually hunting down every reference" and recommends adding the
tools. Trial 5 praises diagnostic deltas while acknowledging "symbol not found"
hiccups. Trial 6 calls the tools "more like a liability than a help" after
requesting a declaration move while intending a field rename; the returned
diff disclosed the move. Trial 10 says they are "a helpful supplement but not
yet reliable enough to fully trust for complex migrations."

Several reflections expose awareness gaps: trial 4 did not know batch could
span files, trial 8 asked for a dedicated rename tool that already existed,
and trial 7 treated sequential dependencies as making batching unsuitable.
Claims of batch atomicity also conflict with earlier children remaining
applied when later children fail. These are model opinions, not capability
or transaction guarantees established by the traces. No reflection reports
the old output-schema mismatch.

## Measurement audit

The independent transcript audit found eighty unique transcripts with no
token, tool-count, or task-turn discrepancies against the final report.
Seventy-eight runs have complete terminal usage. The two task 11 timeouts
retain partial usage and are excluded from complete-usage token medians.
Prompt totals include repeated cached context; they are not unique context
sizes or provider billing estimates.

| Task | Baseline median prompt tokens | Semedit median prompt tokens | Baseline median output tokens | Semedit median output tokens | Complete usage captures, baseline/semedit |
| --- | ---: | ---: | ---: | ---: | ---: |
| 01 | 46,879 | 159,514 | 853 | 463 | 10/10 |
| 09 | 62,996.5 | 196,071.5 | 1,308.5 | 1,071 | 10/10 |
| 11 | 122,722 | 914,355 | 3,437 | 5,283 | 10/8 |
| 12 | 371,110.5 | 1,397,087.5 | 7,729 | 11,793 | 10/10 |

All retained diagnostic starts follow the final task finish. The strict
timestamp-bound helper nevertheless reports four cross-clock discrepancies:
one timeout duration is 7.13 ms below its serialized task interval, and three
durations exceed the serialized diagnostic-start offset by 6.336, 23.112, and
15.058 ms. Preserve those findings rather than increasing its tolerance.

Independent source review establishes why numeric bounds are not comparable:
`wall_clock_ms` uses `time.Since(processStartedAt)` and Go's monotonic clock;
JSON transcript timestamps retain only civil time. The aggregate duration is
captured before requesting a diagnostic reflection, so reflection time is
excluded by construction. These artifacts support ordering and usage audits,
not exact duration bounds across clock domains. No specific wall-clock
adjustment is asserted without supporting system evidence. Exact retrospective
bounds require additional persisted monotonic timing data in a future run.
Pass rates, call counts, tokens, and reflections do not depend on these bounds.

Saved paired result records are retained under
`data/benchmarks/results/opencode-qwen36-iq4-xs-post-schema-10x-20261005-0022/`.
The aggregate report and independent audits are retained locally as:

- `.scratch/opencode-qwen36-iq4-xs-post-schema-10x-20261005-0022.json` and `.md`.
- `.scratch/qwen-post-schema-audit-final.json`.
- `.scratch/qwen-post-schema-clock-audit-final.json`.
- `.scratch/qwen-post-schema-qualitative-audit-20261005-0022.json`.

## Conclusion

Qwen selects semantic tools in all forty trials, and the prior output-schema
rejection does not recur. Schema repairs do not remove operation misuse,
retry loops, native editing fallbacks, or the independent server defects above.
Task 12 improves over its paired baseline while task 11 worsens; aggregate
success is equal. Ten trials per task and intervening harness/setup repairs
do not establish a controlled causal improvement against the old binary.
