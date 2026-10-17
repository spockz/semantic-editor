<!-- Records observed storage evidence without treating partial samples as measured peak RAM usage. -->

# RQ-0072: Benchmark RAM Storage Measurement

Status: Open
Date: 2026-10-17

## Question

What capacity covers concurrent benchmark workspace peaks, and how much does
native tmpfs reduce physical solid-state drive (SSD) writes under otherwise
identical conditions?

## Initial inspection

Read-only inspection of the primary development checkout found these allocated
disk totals using `du -sk` on 2026-10-17:

| Directory | Allocated KiB | Interpretation |
| :--- | ---: | :--- |
| `.scratch/benchmarks` | 19,720,380 | Historical and live jobs plus persistent state; never sum into RAM capacity. |
| `.scratch/go/build` | 37,127,888 | Project-wide cache; not a representative per-fixture cache. |
| `.scratch/go/tmp` | 6,539,968 | Project-wide temporary state; not a per-job peak measurement. |

The new checkout initially had no benchmark samples and empty build/tmp paths.
A complete primary-checkout scan failed explicitly when an active OpenCode
job's gopls cache files disappeared during measurement. A failed scan supplied
no capacity recommendation. Stop benchmark activity or inspect stable retained
samples before automatic mounting.

Two stable pinned repository materializations were then measured together:

* `go-zero/63cfe60f1a7583865757780ebfe2a5309a542134`
* `grpc-go/648cf9b00e25b3af7839d6678a8fe319505c3b80`

The largest measured source footprint was 14,401,536 bytes, with a typical
footprint around 12 MiB and a maximum of 1,091 files. Neither sample included
fixture build, temporary, gopls, or agent runtime state. At concurrency two,
safety factor 1.75, and 1 GiB fixed overhead, the rounded partial calculation
was 2 GiB. Its confidence was explicitly `uncertain`, so it cannot authorize an
automatic mount. This is source materialization evidence, not a 2 GiB peak
capacity claim for external-repository benchmarks.

The same invocation measured 24 GiB physical memory and approximately 9.9 GiB
currently available memory. Availability is transient and must be remeasured
at mounting time; it is independent of the storage estimate.

Reproduce source-only inspection with:

```sh
root=/path/to/existing/semantic-editor
./tools/benchmark-harness/scripts/benchmark-ram-size.sh estimate --concurrency 2 \
  "$root/.scratch/benchmarks/cache/repos/go-zero/63cfe60f1a7583865757780ebfe2a5309a542134" \
  "$root/.scratch/benchmarks/cache/repos/grpc-go/648cf9b00e25b3af7839d6678a8fe319505c3b80"
```

## Native macOS smoke validation

The developer mounted a managed 2 GiB native tmpfs at
`/Volumes/semedit-bench-ram`. The first Task 09 Luna 6 smoke run,
`ram-hard-luna6-smoke-20261017`, exercised one repeat of all three arms at
concurrency one. Process observations confirmed fixture-local `GOCACHE`,
`GOTMPDIR`, `TMPDIR`, `GOPLSCACHE`, and `GOBIN` on tmpfs, with `GOMODCACHE`
accessing the existing SSD directory directly. No fixture-local module cache
was created. The largest sampled complete workspace occupied 167,776 KiB
(about 164 MiB). Sampling was periodic and is a lower bound on peak usage,
not an exhaustive capacity measurement for other tasks or external repositories.

All arms passed policy, AST, and build checks, but Go test execution failed
with `signal: killed`. A separate tiny Go program reproduced the failure:
the linker output was killed on tmpfs, while a byte-identical copy to a fresh
inode on the same tmpfs executed successfully. Native ad-hoc re-signing also
permitted execution. This is consistent with the Darwin signature-cache issue
described in Go's [`purgeSignatureCache`](https://go.dev/src/cmd/link/internal/ld/outbuf_darwin.go),
which uses best-effort invalidation after memory-mapped linker writes.
The volume was executable; no SSD fallback was used. Results and transcripts
survived cleanup of all three job workspaces.

After enabling the Darwin tmpfs linker-output refresh, the separate run
`ram-hard-luna6-smoke-linkrefresh-20261017` passed all four oracle levels in
all three Luna 6 arms. Its largest sampled job footprint was 139,808 KiB
(about 137 MiB), at concurrency one. Each job used an independent RAM build
cache and the same direct SSD module cache. The volume remained mounted at
2 GiB after job cleanup, with only its management marker and macOS filesystem
event state remaining. Durable reports, checkpoints, transcripts, and MCP
metrics remained on SSD. This was tagged `purpose=storage-validation`, not
designed as a performance comparison.

The real tmpfs integration tests also passed (25.318 seconds): exact bytes and
executable permissions survived atomic inode refresh, linker failures retained
their exit status, and repeated Go test/run/build commands executed successfully
with cached outputs and a wrapper path containing spaces. The recovered
`semantic_batch` cache-directory diagnostic is recorded separately in
`docs/SUBOPTIMAL_TOOLS.md`. Follow-up investigation traced it to generated
test-main files inside project-local caches entering package-pattern selection;
exclusion now applies equally to internal and external build caches.

## External SWE repository observation

Run `ram-swe-2951-luna6-retry-20261017` exercised `grpc__grpc-go-2951` with
Luna 6 medium, one repeat of all three arms, concurrency one, on the existing
2 GiB native macOS tmpfs. This task ties for the highest complexity profile and
has the largest explicit recipe, with 59 operations. Largest sampled complete
workspace footprints were 415,776 KiB for baseline, 1,156,896 KiB for semedit
agentic, and 396,016 KiB for semedit explicit. Whole-volume usage peaked at
1,149,024 KiB (1.096 GiB), with 948,128 KiB (926 MiB) still available.
Peak sampled inode usage was 33,674 of 1,000,000. One directory sample raced
deletion; raw diagnostics remain available. Periodic sampling provides lower
bounds rather than exact peaks, and component maxima need not coincide.

Process environments confirmed fixture and oracle Go caches on tmpfs, with
the existing primary-checkout module cache accessed directly on SSD and
dependency fetching disabled. No private module cache appeared in RAM. All
fixtures were removed; durable results survived and the volume stayed mounted.
The largest workspace included compiler temporary state and roughly 433 MiB
of gopls state, explaining why source checkout size alone underestimates demand.

All three arms failed acceptance: agent test edits conflicted with the private
test patch in baseline and agentic; the explicit agent submitted an empty batch
instead of the prescribed operations. No observed failure involved capacity
exhaustion or SSD fallback. These incomplete solutions do not establish full
successful-task capacity or safety at concurrency two.

Initial preflight also exposed Go's omission of module `Dir` when the fixture
has only a go.mod checksum. The fix checks source directories, absent partial
markers, and archive hashes rather than relying on that optional inventory
field. Regression tests cover checksum omission and incomplete extraction.

Raw samples, reproduction commands, outcomes, and filtered process-path evidence:
[storage observation](../../data/benchmarks/results/ram-swe-2951-luna6-retry-20261017/storage-observation/README.md).

Subsequent provenance inspection confirmed that both semedit SWE arms had skipped
CLI prewarming because the harness required an AST assertion file. That is a
limitation of the recorded run above, not evidence of a warmed starting cache.
Warmup discovery now uses visible build-selected Go files when no AST assertion
exists. Shared-cache setup first runs `go mod download all` in a disposable oracle
copy and then validates offline access before timing. A native integration test
using the same grpc-go fixture and SSD module cache passed: real verify warmup
took 21.220 seconds, left both RAM build and gopls caches populated, created no
private module cache, and cleaned up without launching a model. Historical
benchmark results retain their original skipped-warmup provenance; future
comparisons must apply the corrected warmup policy to both storage modes.

## Physical write comparison

The macOS host exposes I/O Registry block-storage counters, including
`Bytes (Write)` for the internal SSD. `iostat` was blocked by sandbox access to
`kern.boottime`. An escalated `sudo -n true` check returned `a password is
required`, initially preventing an unattended native tmpfs mount. The developer
subsequently mounted the volume for the smoke validation above. No paired
physical-write measurement was performed, and no reduction percentage is claimed.

The [harness guide](../../tools/benchmark-harness/README.md#comparing-ssd-writes)
defines alternating paired runs with the same populated module cache, empty
per-job build caches, concurrency, fixtures, binary, prewarm policy, and durable
device-counter evidence. That paired measurement and successful external-task
capacity measurements remain outstanding. Keep this investigation open until
both are available. The accepted storage contract is
[ADR-0074](../adr/0074-ram-backed-benchmark-workspaces-and-persistent-module-cache.md).
