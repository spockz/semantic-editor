<!-- Defines storage policy for test fixtures and temporary test workspaces. -->
# ADR-0093: RAM-Backed Test Workspaces and In-Memory Fixtures

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Use in-memory fixtures and verified benchmark tmpfs for test workspaces and Go compiler scratch, including the build of the macOS linker-refresh helper; retain its small executable, source, and reusable caches on persistent storage. |
| description | Reuse the benchmark harness's verified RAM volume for disposable test state to reduce SSD scratch writes. Preserve persistent source, module, and reusable build caches; measure check-specific capacity before moving additional caches. |
| role | process |
| scope | tooling |
| area | testing |
| languages | generic |

## Context

The complete quality check creates temporary workspaces, compiler scratch
files, and subprocess fixtures. The benchmark harness already manages a
verified native tmpfs for disposable workspace state, while ordinary `/tmp`
and repository scratch directories can reside on the SSD. Many source-editing
unit tests can keep fixture contents in memory, but tests that invoke the Go
toolchain, command-line programs, or language servers must exercise real file
and process behavior. On macOS, Go binaries linked from tmpfs may be killed
unless the benchmark harness's linker-output refresh hook is active.

## Decision

Use in-memory source buffers and in-memory filesystem fixtures for unit tests
when those preserve the behavior under test. Tests that require real filesystem
or subprocess behavior use private temporary directories on the same
benchmark-managed tmpfs selected by the repository's benchmark configuration.
The check targets route both `TMPDIR` and `GOTMPDIR` there after verifying that
the selected mount is the managed tmpfs. On macOS, they pass Go's `toolexec`
hook through test subprocesses so successful linker outputs in tmpfs receive
the same inode refresh used by benchmark fixture builds; this avoids the
observed killed Go test binaries. The hook is built once into the persistent
tool directory, with its disposable Go compiler scratch on the same managed
tmpfs. Checks fail explicitly if a selected RAM mount is absent,
incompatible, or lacks sufficient capacity; they do not silently fall back to
SSD-backed test workspaces.

The source checkout, persistent Go module cache, reusable Go build cache,
tool installations, and durable test evidence remain in their configured
persistent locations. This decision limits disposable test writes without
discarding reusable compilation results. Move the Go build cache to RAM only
after measuring the complete check's peak footprint and confirming the volume
has adequate memory headroom. Do not copy the module cache into tmpfs.

## Invariants

* In-memory fixtures must preserve the filesystem semantics relevant to the
  assertion; tests of permissions, symlinks, process boundaries, or atomic
  publication continue to use a real filesystem.
* Filesystem-backed test directories, including Go compiler scratch, are
  unique per check invocation and removed after the test process exits,
  including failure and cancellation.
* The selected temporary root must be verified as the benchmark-managed tmpfs
  before any test uses it. A user must explicitly select a different storage
  policy to permit SSD-backed temporary test files.
* Module sources and reusable build artifacts remain in persistent caches;
  test workspaces must not copy them into the RAM volume.
* Capacity measurements cover concurrent `make check` jobs and include memory
  headroom. tmpfs can still swap under memory pressure, so it does not promise
  zero physical SSD writes.
* Reports, logs needed for diagnosis, and other durable evidence are published
  outside the temporary test directory before cleanup.

## Consequences

Unit fixtures that need only source contents avoid filesystem creation. Tests
that require operating-system behavior keep their existing assertions while
using RAM-backed workspaces and compiler scratch. Developers must provision
the shared benchmark tmpfs before running filesystem-backed checks;
unsupported or undersized storage is reported before test execution. Reusable
caches remain fast across runs, while their growth and the check's peak RAM use
remain separate measurement concerns.
