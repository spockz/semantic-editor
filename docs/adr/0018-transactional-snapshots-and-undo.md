# ADR-0018: Transactional Snapshots and Conflict-Checked Undo

* **Status**: Accepted
* **Date**: 2026-09-17

---

## Context

Refactoring operations orchestrated by AI agents or developers frequently require experimentation across multiple files. Under ADR-0004 (Staged Execution & Diagnostic Reporting), the engine deliberately avoids automatic rollback on compiler errors, permitting multi-step refactorings that temporarily break code.

However, when an agent or developer decides to abandon a refactoring path, resorting to coarse git operations (`git checkout`, `git reset`, or `git stash`) risks destroying uncommitted user changes, polluting git reflogs, or clobbering out-of-band edits. A lightweight, scoped content journal is required to enable safe, targeted rollback of touched files with strict conflict verification.

---

## Decision

We introduce content-addressed pre-edit snapshotting and conflict-checked undo capabilities across the engine, CLI, and MCP interfaces:

1. **Content-Addressed Storage Journal**:
   * Snapshots reside in `.scratch/snapshots/<snapshot_id>/` and never mutate git trees or invoke `git stash create`.
   * `manifest.json`: Records `id`, `created_at`, `label`, `description`, `base_commit`, and a list of `files` containing `path`, `mode`, `preimage_sha256`, and `post_edit_sha256`.
   * `blobs/<sha256>`: Stores raw preimage byte content addressed by SHA-256 hash.

2. **Conflict Checking & Atomic Restore**:
   * Preflight validation: Before performing any disk write, verify every touched file. If any file has changed from its recorded `post_edit_sha256` (and is not already at preimage state), abort with `ErrConflict` / `*ConflictError` and execute zero disk writes.
   * Atomic restoration: All file restores use ADR-0010 atomic write semantics (`pipeline.WriteAtomic` tempfile $\to$ `fsync` $\to$ `os.Rename` with advancing `mtime`).
   * Scoped mutation: Restores only mutate files listed in the snapshot manifest; files outside the manifest remain untouched.

3. **Interfaces**:
   * **MCP `semantic_snapshot`**: Captures pre-edit state of specified files or the workspace root, persists journal, and returns `snapshot_id`, `created_at`, and `captured_files`.
   * **MCP `semantic_undo`**: Validates hashes, preflights writes, restores blobs atomically, runs compiler diagnostic deltas, and returns `restored_files`, `restored_diff`, and `diagnostics`. Annotated with `destructiveHint: true`.
   * **CLI `semedit snapshot`**: Captures snapshots with `--label`, `--desc`, `--paths`, and `--record-post`.
   * **CLI `semedit undo`**: Restores snapshots by ID or `"latest"`.

4. **Integration with ADR-0004**:
   * Snapshots are created on demand. Compiler errors never trigger automatic rollback; undo is strictly an explicit tool or user action.

---

## Invariants

1. **Zero Disk Writes on Conflict**: If any file fails post-edit checksum validation or blob integrity during preflight, the restore aborts immediately with zero file modifications.
2. **Manifest Isolation**: Undo operations must never touch, alter, or delete any file not explicitly tracked in `manifest.json`.
3. **Atomic Restore Semantics**: Every restored file must be written via `pipeline.WriteAtomic` ensuring `fsync` and monotonically advancing `mtime` for language server cache invalidation.
4. **Git Tree Preservation**: Snapshot storage operates purely within `.scratch/snapshots/` and must never invoke destructive git commands or create git stash commits.
5. **Explicit Reversion Only**: In compliance with ADR-0004, the engine never rolls back automatically; rollback requires explicit user or agent invocation.

---

## Consequences

* **Positive**: Enables AI agents to safely explore speculative refactoring steps with guaranteed, conflict-checked rollback.
* **Positive**: Preserves uncommitted user edits outside the refactored files.
* **Positive**: Emits MCP `destructiveHint: true` to prompt client confirmation dialogs in interactive harnesses.
* **Neutral**: Storage overhead in `.scratch/snapshots/` scales with the number and size of snapshot preimages; scratch files remain gitignored.
