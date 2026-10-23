# ADR-0070: Opt-In Go Runtime Isolation

* **Status**: Accepted
* **Date**: 2026-10-14

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Normal CLI/MCP inherit Go state; explicit bases and benchmark fixtures enable isolation. Development Make targets share module/build caches with the primary checkout while retaining worktree-local tools and temporary state; explicit cache overrides remain available. Benchmark isolation can independently select a persistent module cache. |
| description | ADR-0041 introduced project-local Go state to keep sandboxed benchmark processes within their fixture. Applying that default to normal CLI and MCP use also replaced the user's caches, Go environment file, and workspace selection. New projects downloaded dependencies again and created `.scratch/go` even when the caller already had usable Go state. |
| role | principle |
| scope | tooling |
| area | build |
| languages | go |

## Context

ADR-0041 introduced project-local Go state to keep sandboxed benchmark processes within their fixture. Applying that default to normal CLI and MCP use also replaced the user's caches, Go environment file, and workspace selection. New projects downloaded dependencies again and created `.scratch/go` even when the caller already had usable Go state.

## Decision

Normal CLI and MCP operations inherit the user's Go environment. A nonempty explicit Go base directory enables managed isolation. MCP exposes that opt-in through `--go-base-dir`; benchmark preparation selects fixture-local state and configures its semantic server explicitly. Empty or omitted base directories do not enable isolation.

Normal operation preserves cache, temporary-directory, tool installation, environment-file, workspace, and unrelated build-flag configuration. Read-only protection for ordinary module operations remains independent of isolation. Explicit dependency operations may update module files. Offline module-cache seeding requires an explicit managed base and never writes into an inherited user cache.

This decision supersedes ADR-0041's default Go state policy. It leaves canonical request roots, workspace manifest safety, and atomic file publication unchanged.

Repository development Make targets share downloaded modules (`GOMODCACHE`) and
compiled artifacts (`GOCACHE`) under the primary Git checkout's `.scratch/go`.
Both Make entry points use the same configuration and discover the primary
checkout through Git worktree metadata. Source exports without Git metadata use
their own `.scratch/go`. Explicit environment or command-line cache paths take
precedence; `GO_CACHE_ROOT` overrides the shared base. `GO_SCRATCH` selects local
state independently. Temporary files, binaries, plugins, environment files,
gopls state, and linter caches stay local to each worktree. Worktree removal
must not clear shared caches. Benchmark fixture isolation remains explicit and
unchanged; development cache sharing does not select fixture cache policy.

[ADR-0074](0074-ram-backed-benchmark-workspaces-and-persistent-module-cache.md)
adds an explicit independent module-cache selection for benchmark isolation.
`--go-module-cache` preserves that SSD cache alongside a disposable RAM Go base;
ordinary callers without the option retain this decision's behavior. This direct
selection does not seed or copy the cache.

## Invariants

* Normal CLI and MCP use do not create `.scratch/go` or replace the caller's cache paths.
* Inherited `GOWORK` and unrelated Go flags continue to select the user's workspace and build configuration.
* Explicit isolation validates managed directories before invoking Go tools and overrides inherited state paths.
* Benchmark fixture shells and MCP subprocesses retain isolated Go state, including selected worktree roots.
* Module-cache seeding requires explicit isolation and offline dependency access.
* No workspace manifest is created or modified to select either mode.
* Source snapshots, move preflight, and directory verification exclude the selected runtime cache and temporary directories. Buffers capture that policy from their operation context and continue to detect changes to application source.

## Consequences

Ordinary projects reuse existing downloaded modules and build caches. Benchmark runners must request isolation rather than rely on an incidental semedit default. Tests prove both inherited state and explicit fixture isolation, including subprocess behavior and failure boundaries.
