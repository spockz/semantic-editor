# RQ-0005: Monorepo Workspace Topologies

* **Status**: Resolved
* **Category**: Workspaces & Configuration Safety
* **Last Updated**: 2026-10-04

---

## 1. Problem Context

Repositories frequently house multiple sub-projects (e.g. multiple `go.mod` files, Cargo workspaces with multiple crates, or multi-package TypeScript setups).

Language servers need to know the workspace boundaries to correctly resolve cross-module references. However:

1. Writing or modifying root workspace configuration files (such as `go.work`) on disk without permission can corrupt user environments and dirty git status.
2. Running multiple language servers for different languages concurrently can consume gigabytes of host RAM and is out of scope for Phase 1.

---

## 2. Exploration Paths

### A. Non-Destructive In-Memory Attachment

* When `semedit` detects multiple modules without a root manifest:
  * **Strict Invariant**: It will **never** write a `go.work` or `Cargo.toml` to disk without explicit user approval.
  * Instead, it initializes the language server with an in-memory multi-folder configuration (`workspaceFolders` array in LSP `initialize`), keeping the working tree clean.

### B. User Approval Flow for On-Disk Manifests

* If on-disk workspace manifests are strongly recommended for language stability (e.g. `go.work`), `semedit` prompts the user with an interactive Yes/No choice before generating it.

---

## 3. Sources & Prior Art

* **Go Workspaces (`go.work`)**: [Go Blog: Get familiar with workspaces](https://go.dev/blog/get-familiar-with-workspaces).
* **`gopls` Multi-Module Workspace Support**: [gopls documentation on multi-module setups](https://github.com/golang/tools/blob/master/gopls/doc/workspace.md).
* **Cargo Workspaces**: [The Rust Cargo Book: Workspaces](https://doc.rust-lang.org/cargo/reference/workspaces.html).
* **LSP `workspace/workspaceFolders` Specification**: [LSP Specification 3.17](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#workspace_workspaceFolders).

---

## 4. Resolution & Architecture Decision

Resolved and codified in [ADR-0005: Workspace Manifest Safety & Monorepo Scoping](../adr/0005-workspace-manifest-safety-and-monorepo-scoping.md) and [ADR-0041: MCP Runtime Configuration and Project-Local Go State](../adr/0041-mcp-runtime-configuration-and-project-local-go-state.md):

1. **Non-Destructive Invariant**: `semedit` strictly prohibits writing or mutating workspace manifests (`go.work`, `Cargo.toml`) on disk without explicit user approval.
2. **Contained Workspace Roots**: Runtime execution resolves contained canonical workspace roots per request, isolating build caches, LSP subprocesses, and temporary states under `.scratch/` within the resolved root.
3. **Scoped Sessions**: Language servers operate within explicit module or project boundaries rather than unbounded cross-module multi-language attachments.
