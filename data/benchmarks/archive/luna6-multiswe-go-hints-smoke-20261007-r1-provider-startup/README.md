<!-- Preserves a smoke attempt that failed before model work and excludes it from task solve measurements. -->

# Luna 6 hinted smoke: provider startup failure

The first attempt at `luna6-multiswe-go-hints-smoke-20261007-r1` ran under the outer workspace sandbox. All six go-zero 2463 jobs failed before model work because Codex could not write its state database or initialize its app-server client (`Operation not permitted`). Each provider process exited with code 1.

These are infrastructure failures, even though the progress renderer printed `FAIL`. They are not assignment failures and must not contribute to solve rates. No task iteration completed. Raw reports and the invocation log are preserved here unchanged. Referenced transcripts remain in `.scratch/benchmarks/transcripts`.

The full five-task plan also failed before launching providers because grpc 2735 has no nonblank explicit semedit recipe. The same recipe is missing for grpc 2951, grpc 3361, and go-zero 964. `full-plan-error.log` preserves the planning failure.

The six runnable go-zero 2463 cells were retried with host access using the original run ID, one iteration per cell, Luna 6 medium/high, and all three arms. Coding-shell network access remained disabled; source hydration used the pinned local cache.
