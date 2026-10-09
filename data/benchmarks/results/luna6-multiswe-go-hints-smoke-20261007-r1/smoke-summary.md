<!-- Records the requested single-iteration smoke with task, timeout, and planning outcomes kept distinct. -->

# Luna 6 hinted Multi-SWE Go smoke, one iteration per cell

Run ID: `luna6-multiswe-go-hints-smoke-20261007-r1`.

Only `zeromicro__go-zero-2463` was runnable. The full five-task plan exited with code 1 before provider launch because explicit recipes are missing for grpc 2735, grpc 2951, grpc 3361, and go-zero 964. Twenty-four cells were blocked; they were not executed or counted as failures.

## Completed cells

| Effort | Vanilla | Semedit implicit | Semedit explicit |
| :--- | :--- | :--- | :--- |
| Medium | Task fail, 190.074 s | Timeout, 719.391 s | Pass, 184.349 s |
| High | Task fail, 326.677 s | Timeout, 719.513 s | Pass, 238.085 s |

Each cell ran one independent iteration. Vanilla and implicit received the original issue and up to three progressive hints in the same continuing session. Explicit received the validated 29-operation recipe and no hints. Both explicit cells passed on their first task turn. Both vanilla cells exhausted all three hints and failed the acceptance tests: their `logc.Alert` signature did not accept the required context argument. High vanilla also reported `TestStructedLogAlert` failing.

Both implicit cells were terminated by the harness's twelve-minute job timeout during their fourth task turn. Their last completed oracle checks failed; no final oracle ran on the incomplete final-turn candidate. These are timed-out attempts, distinct from the two graded acceptance failures. No acceptance-test installation errors occurred in the completed host-access run. The invocation exited with code 1 after saving all six checkpoints and reports.

## Settings and provenance

Luna 6 medium/high, small context, one repeat, concurrency four, twelve-minute job timeout, write restriction, MCP instruction mode none, optional skill disabled, verify prewarm requested. The external fixture has no configured Go prewarm target, so that prewarm was skipped. Pinned source commit: `63cfe60f1a7583865757780ebfe2a5309a542134`. Each workspace was copied from the local cache before provider launch. The cache had no Git remotes; coding-shell network access was disabled.

`make check` passed before execution. The harness, semedit binary, and fixture hashes matched at launch and after execution. Evidence includes the invocation and preflight logs, source fixture, input hashes, all six transcripts, and semantic MCP timing records where recorded. `smoke-summary.json` preserves metrics and explicit outcome classifications; `jobs/` contains the original harness checkpoints, and the task subdirectory contains paired reports.

The first outer-sandbox attempt failed before any model work. Those infrastructure records were moved to [the provider-startup archive](../../archive/luna6-multiswe-go-hints-smoke-20261007-r1-provider-startup/README.md) and excluded from this table. The harness printed `FAIL` for those startup errors and these timeout errors; use the explicit classifications above when interpreting the run.
