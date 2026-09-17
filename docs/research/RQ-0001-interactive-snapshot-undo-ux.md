# RQ-0001: Interactive Snapshot & Undo UX (No-Type Affirmation)

* **Status**: Resolved
* **Category**: UX & Agent Harness Integration
* **Last Updated**: 2026-09-17

---

## 1. Problem Context

When `semedit` performs a refactoring that introduces diagnostics or breaking changes, the user or agent needs the ability to review, accept, or revert. Forcing the user to manually type shell commands (e.g. `semedit undo` or `git checkout`) creates high interaction friction, slows down pair programming, and breaks flow state.

The research challenge is to present snapshot review and rollback as a native, 1-click UI decision across different agent harnesses without requiring free-form text input.

---

## 2. Exploration Paths

### A. Harness-Native Decision Modals

* **Antigravity / Specialized Coding Agents**: Trigger interactive structured choice modals (e.g. `ask_question` with binary options: `[Keep Changes]` vs `[Revert to Pre-Edit Snapshot]`). The execution blocks until the user clicks, providing an instant zero-typing experience.
* **CLI & Terminal Harnesses (Claude Code, Aider, CLI)**: Use terminal raw-mode interactive selectors (ANSI arrow-key select menus or single-keystroke `[y/N]` prompts) rather than waiting for command lines.
* **IDE Agents (Cursor, Windsurf, VS Code)**: Integrate with the editor's native inline diff review bar ("Accept / Reject") by emitting standard LSP `WorkspaceEdit` previews.

### B. MCP Elicitation & Tool Approvals

* Investigate whether the Model Context Protocol (MCP) specification supports interactive client-side sampling, notification dialogs, or confirmation hooks that agent UIs render natively.

### C. Snapshot Storage Mechanism

* Evaluate storage backends that do not pollute git commit logs:
  1. Git internal stash / shadow reflogs (`git stash create`).
  2. Isolated patch files saved in `.scratch/snapshots/`.
  3. Git shadow worktrees.

---

## 3. Sources & Prior Art

* **Cursor Shadow Workspaces & Review Bar**: [Cursor Technical Blog](https://cursor.com/blog) — inline diff acceptance flows.
* **Agent-LSP Simulate Mode**: [Blackwell Systems agent-lsp](https://github.com/blackwell-systems/agent-lsp) — dry-run simulation of LSP edits in memory.
* **Antigravity Interactive Question Modals**: Built-in harness capability for blocked, structured choice presentation.
* **Model Context Protocol Specification**: [modelcontextprotocol.io](https://modelcontextprotocol.io) — interactive tool call semantics.

---

## 4. Resolution & Architecture Decision

Formalized in [ADR-0018: Transactional Snapshots and Conflict-Checked Undo](../adr/0018-transactional-snapshots-and-undo.md):

1. **Content-Addressed Storage Journal**: Implemented scoped content journals under `.scratch/snapshots/<id>/` containing `manifest.json` and `blobs/<sha256>`, completely avoiding git stash reflog pollution and shadow commit overhead.
2. **Preflight Conflict Detection**: Restores perform strict preflight validation and abort with `ErrConflict` / `*ConflictError` and zero disk writes if on-disk state diverged from recorded `post_edit_sha256`.
3. **Atomic Restore**: Restorations enforce ADR-0010 atomic write semantics (`pipeline.WriteAtomic`) with advancing `mtime`.
4. **Dual Interface & Tool Safety**: Exposed via `semantic_snapshot` and `semantic_undo` (annotated with `destructiveHint: true` to trigger harness confirmation dialogs) alongside Cobra CLI subcommands `semedit snapshot` and `semedit undo`.
