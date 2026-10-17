<!-- Retains storage evidence for a native tmpfs benchmark independently of disposable fixtures. -->

# SWE-2951 RAM storage observation

On 2026-10-17, `grpc__grpc-go-2951` ran against `codex/gpt-6-luna/medium`,
one repeat of three arms, concurrency one, timeout 12 minutes. This task ties
for the highest complexity profile and has the largest explicit recipe (59
operations). This is storage validation, not a controlled performance comparison.

| Arm | Largest sampled complete workspace | Outcome |
| :--- | ---: | :--- |
| baseline-diff | 415,776 KiB (406 MiB) | Acceptance patch conflicts after agent test edits. |
| semedit-agentic | 1,156,896 KiB (1,130 MiB) | Acceptance patch conflicts after agent test edits. |
| semedit-explicit | 396,016 KiB (387 MiB) | Agent submitted an empty batch; unchanged code failed acceptance tests. |

The 2 GiB native tmpfs at `/Volumes/semedit-bench-ram` reached 1,149,024 KiB
of whole-volume consumption (1.096 GiB), leaving 948,128 KiB (926 MiB) free.
Its highest sampled inode consumption was 33,674 of 1,000,000. One directory
measurement raced deletion; `measurement-errors.log` retains the diagnostic.
Periodic samples cannot establish an exact peak. `du` and `df` use different
accounting, and measurements within a sample are sequential.

The complete workspace measurements include nested oracle workspaces, compiler
outputs, temporary files, and agent and gopls state. Independent component
maxima in `summary.json` need not occur simultaneously and must not be summed.
Sources alone occupied about 21 MiB. The 2 GiB mount accommodated this serial
run; this does not establish safety for concurrency two or longer runs.

Allowlisted process environments in `process-paths.log.gz` confirm fixture-local
`GOCACHE`, `GOTMPDIR`, `TMPDIR`, `GOPLSCACHE`, and `GOBIN` on tmpfs, including
private oracle paths. `GOMODCACHE` directly used
`/Users/alessandro/sources/semantic-editor/.scratch/go/mod` on SSD.
`GOPROXY=off`, `GOSUMDB=off`, and `GOTOOLCHAIN=local` remained set. No fixture-local
module cache appeared in any sample. All job directories were removed; the
volume remained mounted, with 8,944 KiB consumed by management and filesystem
event state after cleanup. No capacity failure or SSD workspace fallback was
observed. Persistent reports, transcripts, and checkpoints survived on SSD.

The initial setup attempt, `ram-swe-2951-luna6-20261017`, stopped before model
execution. Go's module inventory omitted `Dir` for an extracted unused module
without a full fixture checksum. Preflight now checks extraction completion
markers directly. A cached x/crypto archive was extracted offline into the
existing SSD cache before the retry; no module sources were copied into RAM.
Focused regression tests and `make check` passed after the fix.

Reproduce from this checkout with the existing mount and populated cache:

```sh
BENCH_STORAGE_OBSERVATION="$PWD/.scratch/swe-2951-new-storage" \
  bash data/benchmarks/results/ram-swe-2951-luna6-retry-20261017/storage-observation/observe-swe-2951.sh
# Run in another terminal, then create the observer output directory's stop file.
PATH="/Users/alessandro/sources/semantic-editor/.scratch/go/bin:$PATH" \
  make bench-smoke-swe TASKS=grpc__grpc-go-2951 CONCURRENCY=1 TIMEOUT=12m \
  BENCH_RAM_ROOT=/Volumes/semedit-bench-ram \
  BENCH_MODCACHE=/Users/alessandro/sources/semantic-editor/.scratch/go/mod \
  RUN_ID=ram-swe-2951-luna6-new PROVENANCE=purpose=ram-capacity-observation
```

The observer requires Bash, jq, du, df, and process-inspection permission. Its
process records contain only the listed cache and offline-policy variables.
`usage.jsonl` and `summary.json` report allocated filesystem storage; they do
not measure resident memory or physical SSD writes. No before/after device-write
comparison was performed, and tmpfs may still cause swap writes.
