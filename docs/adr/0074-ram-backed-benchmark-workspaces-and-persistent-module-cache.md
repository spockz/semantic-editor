<!-- Defines benchmark storage ownership independently of ordinary project builds. -->
# ADR-0074: RAM-Backed Benchmark Workspaces and Persistent Module Cache

* **Status**: Accepted
* **Date**: 2026-10-17

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Make detects managed tmpfs; setup populates SSD modules with configured proxy/checksum settings before offline execution, disables private-module bypass, and discovers warmup targets without AST assertions. Per-job RAM caches remain isolated. Darwin linker inode refresh preserves execution; sizing uses concurrent workspace peaks and memory headroom. |
| description | Extracting repositories, compiling Go, running gopls and agents, and deleting their disposable state causes substantial macOS filesystem churn and SSD writes. Ordinary `/tmp` and APFS scratch directories do not provide a RAM storage contract. Copying a populated module cache into every fixture magnifies the problem without improving experiment isolation. |
| role | process |
| scope | tooling |
| area | benchmark |
| languages | go |

## Context

Extracting repositories, compiling Go, running gopls and agents, and deleting
their disposable state causes substantial macOS filesystem churn and SSD writes.
Ordinary `/tmp` and APFS scratch directories do not provide a RAM storage
contract. Copying a populated module cache into every fixture magnifies the
problem without improving experiment isolation.

## Decision

Benchmark execution optionally selects an existing native tmpfs through
`--ram-fixture-root` and a persistent SSD module cache through
`--module-cache`. The harness validates the filesystem and fails explicitly if
RAM storage is unavailable. Default execution remains supported. Ordinary
project build configuration is unchanged.

Benchmark Make targets select a managed volume at the platform default
mountpoint automatically when its management marker exists, using the existing
SSD `GOMODCACHE` unless explicitly overridden. Native filesystem validation
still runs before job setup; an invalid selected volume fails explicitly.
An explicit empty `BENCH_RAM_ROOT` selects SSD fixtures. Direct CLI invocations
continue to require explicit storage flags.

The complete disposable workspace belongs to one independent job. Its source,
build cache, Go and operating-system temporary directories, gopls cache, tool
installation directory, and agent runtime state reside under that workspace.
Pinned repository caches, transcripts, checkpoints, manifests, and published
results remain in persistent storage. Hydration reads the pinned SSD cache and
materializes a fresh RAM workspace. Evidence required after cleanup is published
before deleting the workspace. Cleanup owns job directories, never the volume
or shared module cache.

The Go environment abstraction accepts a separately selected module cache.
MCP startup forwards that selection explicitly alongside `--go-base-dir`, so
server initialization cannot replace it with a fixture-local module cache.
Before measured execution, setup runs `go mod download all` against the selected
SSD cache in a disposable oracle copy, including declared private-test modules.
It honors process-environment proxy and checksum settings, uses the installed
toolchain, and leaves the visible fixture manifests unchanged. Concurrent
downloads use Go's native module-cache locking; no module sources are copied
into RAM. Setup then checks complete extraction and resolves dependencies
offline. Download, checksum, storage, and missing-dependency errors stop setup
before agent launch. The private copy is removed before provider launch.
Measured dependency access remains offline (`GOPROXY=off`, `GONOPROXY=none`,
`GOSUMDB=off`, `GOTOOLCHAIN=local`). Disabling private-module proxy bypass prevents
inherited private-module settings from triggering direct VCS access.
Setting `GOPROXY=off` on the harness blocks uncached module downloads during
setup; also set `GOSUMDB=off` to suppress checksum-database lookups when fully
offline. Cached archives can still be extracted there before timing.
Independent jobs never share mutable build or gopls caches. Prewarming requires
an empty job build cache, permits the configured populated module cache, and
retains existing timing boundaries and treatment applicability. Go warmup targets
come from a fixture-local Go AST assertion file when present, otherwise Go's
build-selected visible package inventory; an AST assertion is not a prerequisite
for warming a Go repository. Target discovery uses a disposable cache and does
not prime the measured job cache. CLI warmup leaves reusable disk-cache state
but does not keep a live gopls process for the agent.

Platform scripts perform privileged mounting outside the harness. macOS uses
native `mount_tmpfs -e -s` for case-sensitive tmpfs at the suggested
`/Volumes/semedit-bench-ram` path. Linux uses native `mount -t tmpfs` with the
invoking user's uid/gid, mode 0700, and executable files. A requested Linux
no-swap option must be supported explicitly; unsupported kernels fail visibly.
Neither platform uses disk images, overlays, union mounts, or FUSE.

Native macOS smoke testing exposed stale signature state on Go linker outputs:
execution was killed although signature verification succeeded. Republishing
the identical bytes to a fresh inode on the same tmpfs restored execution.
Darwin tmpfs fixture environments therefore use Go's tool-execution hook to
atomically refresh successful executable linker outputs, with fsync and
preserved permissions. The hook passes through other tools and their failures.
It applies equally to all benchmark arms, agents, MCP/gopls, prewarming, and
oracle commands. Linux and SSD fixtures do not enable this workaround. No
executable is relocated to SSD, and signing enforcement remains enabled.

Both platforms share an estimator based on current representative disposable
workspaces, including per-fixture build and runtime state. It reports largest
and typical samples, concurrency, safety factor, fixed overhead, and rounding.
Peak capacity is the largest representative footprint times concurrent jobs
times a configurable safety factor, plus fixed overhead, rounded upward.
Historical jobs are not summed. Persistent modules and durable evidence are
excluded; repository caches contribute only materialized per-fixture source.
Missing retained samples produce an explicitly incomplete estimate, with cache
and fixture source sizes available as partial evidence.

Mounting also requires an independent memory budget based on current memory
availability, execution headroom, and applicable Linux cgroup limits. A tmpfs
limit is a ceiling, not an immediate allocation. Unsafe automatic capacities
are refused with the requirement, budget, and missing evidence. Users can reduce
concurrency, free memory, or explicitly justify a manual limit. Capacity and
inode exhaustion remain possible; setup and execution failures must remain
visible, with durable evidence retained where available.

## Invariants

* RAM mode verifies actual tmpfs and never silently falls back to SSD.
* Persistent modules are accessed directly, never copied into RAM per job.
* Each job owns its complete disposable workspace and mutable caches.
* Durable evidence survives successful cleanup, cancellation, and setup failure.
* Repeated mounting validates existing capacity and filesystem type; a nonempty
  ordinary directory or incompatible mount is never covered or reused.
* Unmounting is explicit, checks ownership and mount identity, and never forces
  teardown of an active filesystem. Successive runs can reuse one mount.
* Sizing excludes historical aggregate usage and persistent state, reports
  incomplete samples, and checks current memory independently.
* Storage and cache policy are recorded as technical provenance and applied
  consistently across comparable jobs. A storage change is not evidence that
  historical SSD and RAM observations are interchangeable.
* Normal CLI, MCP, and project builds retain ADR-0070 behavior.

## Consequences

Disposable writes avoid the SSD filesystem while resident in RAM. tmpfs may
still cause SSD writes through swapping under memory pressure; macOS provides
no guarantee that these pages remain unswapped. Linux no-swap support varies by
kernel and is opt-in. RAM storage reduces filesystem writes but does not
guarantee lower kernel CPU: compilation, metadata operations, memory management,
and gopls still consume CPU.

Developers operate a reusable mount. Setup maintains the selected module cache
before timing and requires network access only for uncached modules, unless
the configured proxy explicitly disables it. Estimates need recalibration as workloads grow and can be
incomplete after cleanup. Paired comparisons must preserve cache policy,
prewarm timing, concurrency, task identity, and durable evidence. Physical SSD
write comparisons require platform counters and an otherwise quiet machine;
logical file sizes alone do not establish device writes.
