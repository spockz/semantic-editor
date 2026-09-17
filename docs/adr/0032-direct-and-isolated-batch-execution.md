# ADR-0032: Direct and Isolated Batch Execution

Status: Accepted
Date: 2026-09-17

## Context

Semantic batches may combine language-server edits, formatters, compilers, and
other command-line tools. Most callers prefer direct low-latency mutation and
accept a partially applied batch when later validation reports diagnostics.
Some callers instead need external tools to operate on a private filesystem
view before source files are changed.

## Decision

Keep `semantic_batch` as the direct mode. The mutating request is its own
authorization. Operations and optional formatting write the live workspace in
order. A later compiler, test, lint, or formatter diagnostic returns
`applied_with_diagnostics`; it does not roll back successful edits.

Provide a separate isolated batch capability for callers that require it. It
builds a complete logical `WorkspaceView` under `.scratch/views/` and runs all
external tools with that root as their working directory. The view uses
copy-on-write file clones when supported by the workspace filesystem, with a
portable copy or persistent-mirror fallback. This supports arbitrary CLI tools,
not only LSP overlays.

For the isolated capability, one affected file is copied back with the atomic
writer. Multiple affected files produce an internally generated strict unified
patch. After validating only the affected live files' regular-file status,
containment, symlink absence, mode, size, and nanosecond mtime against the
batch-start metadata, the host invokes `git apply` without relaxed, index, or
partial-reject options. Plain `git apply` works from the workspace directory
without requiring repository metadata and rejects the entire patch if a hunk
does not apply.

## Invariants

- No second approval, authenticated receipt, or client-supplied patch exists.
- Direct mode reports applied operations and the resulting diff; it does not
  promise rollback or crash recovery.
- All tools in isolated mode receive only WorkspaceView paths, never live-root
  workspace paths.
- Isolated mode initially permits existing regular source-file modifications
  only. Symlinks, special files, resources, manifests, create/delete/move, and
  binary mutation are unsupported.
- A target whose metadata differs from the batch-start manifest, or whose
  mtime is later than batch start, is stale and is not published.
- Generated multi-file patches are never accepted as caller input and use no
  `--reject`, `--3way`, `--index`, `--cached`, path-rewriting, or
  whitespace-relaxing flags.

## Consequences

Direct batches remain simple and fast, with explicit partial-application
reporting. Isolated batches have a filesystem setup cost, but CoW cloning avoids
duplicating file data on supporting filesystems and gives arbitrary tools a
consistent private tree. The generated patch provides review output and normal
multi-file hunk-conflict atomicity; it does not claim crash or power-loss
recovery.
