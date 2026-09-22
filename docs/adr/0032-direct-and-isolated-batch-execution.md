# ADR-0032: Direct Batch Execution and Deferred Workspace Views

Status: Accepted
Date: 2026-09-17

## Context

Semantic batches may combine language-server edits, formatters, compilers, and
other command-line tools. Callers prefer direct low-latency mutation and accept
a partially applied batch when later validation reports diagnostics.

A temporary copy-on-write workspace is a useful orchestration primitive, but it
is not a different semantic-editor capability. An LLM or another controller can
create such a workspace through its own filesystem or workspace-management MCP,
run `semedit` there, inspect the resulting diff, and decide how to carry the
result forward. Embedding that lifecycle here would couple the semantic engine
to a provider, platform, repository, cache, and publication policy it does not
need to own.

## Decision

Keep `semantic_batch` as the direct mode. The mutating request is its own
authorization. Operations and optional formatting write the live workspace in
order. A later compiler, test, lint, or formatter diagnostic returns
`applied_with_diagnostics`; it does not roll back successful edits.

For a fully successful batch, capture diagnostics once before the first edit,
apply every operation, coalesce deferred formatting and import organization,
then capture diagnostics once after post-processing. Return that single final
diagnostic delta on the batch response. Do not run a diagnostic pass for each
child operation and do not require a separate `semantic_verify` after a
successful batch. If an operation or post-processing step fails, preserve the
existing fail-fast partial-application behavior and omit the final delta.

Do not implement an embedded `WorkspaceView` or isolated-batch MCP operation
now. There is no principled safety distinction between direct one-file and
direct multi-file edits: both are authorized mutations in the caller's working
copy. Direct operation is also appropriate when formatters, generators, and
other cooperative tools need to write their normal derived files.

If a future caller needs speculation before publication, it owns workspace
isolation and publication. `semedit` continues to operate on the workspace it
is given. This is intentionally provider-neutral: it neither selects nor
requires a copy-on-write implementation, Git worktree, overlay filesystem, or
operating-system sandbox.

## Invariants

- No second approval, authenticated receipt, or client-supplied patch exists.
- Direct mode reports applied operations and the resulting diff; it does not
  promise rollback or crash recovery.
- A successful batch has exactly one pre-edit and one post-processing
  diagnostic capture; child operations defer their own verification,
  formatting, and import organization.
- The successful batch response carries the final diagnostic delta. A failed
  batch does not claim a final diagnostic state it did not capture.
- A higher-level controller may provide a private working copy, but
  semantic-editor makes no isolation or security claim about it.
- `semedit` does not manage copy creation, tool confinement, diff publication,
  rollback, or workspace cleanup for a controller-owned copy.

## Consequences

Direct batches remain simple and fast, with explicit partial-application
reporting and a bounded diagnostic cost independent of the number of child
operations. A controller that needs an isolated experiment can use an external
workspace mechanism without expanding semantic-editor's trust or lifecycle
surface.

## Deferred Design Record

This decision retains the investigation because it may become relevant for a
future speculative-edit workflow, not because current direct batches are
unsafe. Reopen it only for a concrete requirement such as previewing an
unbounded third-party tool write set, comparing candidate transformations before
choosing one, or preserving a stable input while another actor is concurrently
editing the live tree.

Prior art confirms that this is an orchestration-layer problem with
platform-specific trade-offs:

- Apple documents APFS copy-on-write clones through `clonefile` and
  `copyfile`; the [APFS tools and APIs guide](https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/APFS_Guide/ToolsandAPIs/ToolsandAPIs.html)
  describes the platform primitive.
- [cow](https://github.com/joeinnes/cow) is one external workspace manager. It
  exposes a separate MCP that creates APFS CoW copies, runs commands there, and
  extracts a branch or patch. It is evidence that an agent-level controller can
  own this lifecycle, not a dependency or recommendation for semantic-editor.
- [Git worktrees](https://git-scm.com/docs/git-worktree) are a portable
  repository-oriented alternative, with their own checkout and untracked-file
  trade-offs.
- [Bubblewrap](https://manpages.debian.org/unstable/bubblewrap/bwrap.1.en.html)
  is a Linux-specific process-isolation mechanism. It is relevant only if a
  future requirement is operating-system confinement; a private working copy
  alone is not a security boundary.

The retained, unmerged exploration is
`codex/direct-isolated-batch` at `de22f2c3951f048e3fe8a97b149e270981c3b983`.
It is not a current product feature. The dedicated Sol review found these
constraints before any future reuse:

1. A view nested beneath the live repository, with inherited environment or
   incomplete argument mapping, lets Git-aware tools rediscover and mutate the
   live root. A future design must establish an execution boundary appropriate
   to its stated, non-security isolation claim.
2. A denylist cannot enforce an "existing regular source files only" boundary.
   A future publisher needs a positive source-file classifier and must exclude
   resources, manifests, and private scratch paths.
3. Start metadata alone is insufficient. Publication must preserve full mode
   and validated preimage bytes, then revalidate immediately before publishing,
   so an intervening edit cannot become the patch preimage.
4. A generated unified patch must reject or correctly quote control-bearing
   filenames and verify that its parsed targets exactly match the validated
   change set.
5. Direct batches need one final workspace diff, including formatter and
   import changes, on success, diagnostics, and partial semantic-operation
   failure.

The review also found that the prototype omitted empty directories and recreated
parent directories with the wrong mode. These are design constraints, not
implementation work authorized by this ADR.

The review's final-diff finding remains a direct-batch conformance task. It is
independent of, and must not be used to justify, an embedded workspace view.
