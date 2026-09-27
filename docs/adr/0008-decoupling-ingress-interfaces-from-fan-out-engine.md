# ADR-0008: Decoupling Ingress Interfaces from Downstream Fan-Out Engine

* **Status**: Accepted
* **Date**: 2026-09-15
* **Last Updated**: 2026-09-27

## Context

Refactoring in modern projects often spans multiple file types (e.g. updating a function definition in `.go` and updating references in `.md` documentation).

Initial designs conflated the **ingress interface** (MCP vs. LSP) with the **transformation behavior** (single-tool routing vs. cross-domain coordination). However, how a client triggers an edit is orthogonal to how the refactoring engine resolves and transforms downstream files.

Workspace diagnostics provide another use case for LSP ingress. A downstream server may support document pull diagnostics without supporting `workspace/diagnostic`, as with the current gopls integration. Semedit can aggregate diagnostics across the selected workspace and expose a standard LSP endpoint to editors and LSP-native agents, alongside its MCP and CLI interfaces.

## Decision

1. **Three-Layer Decoupling**:
   * **Layer 1 (Ingress)**: Pluggable protocol adapters (MCP, LSP, CLI).
   * **Layer 2 (Core Engine)**: Target resolution, multi-domain fan-out, changeset merging, formatting, and diagnostics.
   * **Layer 3 (Downstream Adapters)**: Dedicated language engines (`gopls`, `marksman`, `rust-analyzer`, Tree-sitter).
2. **Symmetric Protocol Execution**:
   * An incoming `textDocument/rename` over LSP triggers the exact same multi-domain fan-out engine as an incoming `semantic_rename` over MCP or a `semedit rename` CLI call.
   * Cross-domain edits (updating code and documentation simultaneously) are returned as a unified `WorkspaceEdit` over LSP, or as an atomic changeset over MCP and CLI.

3. **Optional LSP Diagnostics Endpoint**:
   * Share diagnostic collection across ingress adapters. Use downstream `workspace/diagnostic` where supported; otherwise collect `textDocument/diagnostic` reports across the applicable source scope.
   * Expose standard document and workspace diagnostic reports over LSP. MCP may use the same report model with explicit coverage metadata, but remains an MCP interface rather than an LSP endpoint.
   * Start with initialization, document synchronization, and document/workspace diagnostics; add rename and code actions over the shared edit-planning core afterward. Completion, hover, and a comprehensive language-server proxy are not prerequisites.
   * This is an accepted architectural direction, not a claim that LSP ingress is implemented. Protocol mapping and lifecycle questions remain open in [RQ-0012](../research/RQ-0012-lsp-multiplexing-vs-multi-daemon-orchestration.md).

## Invariants

* Core refactoring and fan-out logic must be completely agnostic of the ingress protocol.
* Any cross-domain refactoring capability available via MCP must be identically invocable via LSP and CLI.

* LSP diagnostics and edit planning must use the client’s current unsaved document contents and versions. Disk-only verification is not a substitute for document synchronization.
* LSP edit planning returns client-applied `WorkspaceEdit` results without first committing those edits to disk. MCP and CLI may apply the same planned changes through their existing publication path.
* Diagnostic requests are read-only. Normalization and mutating fixes require explicit actions; filesystem checks follow publication as specified in [ADR-0054](0054-project-verification-hooks.md).
* Advertise only supported capabilities. Report actual analysis scope and failures; an incomplete collection must not appear as a successful empty workspace report.
* Result IDs and unchanged reports require reliable versioned diagnostic state. Until that exists, return full reports without claiming incremental reuse.

## Consequences

* **Positive**: LSP ingress reuses the core engine and offers workspace diagnostics even when a downstream server only supports document pull, alongside cross-language refactoring for IDEs and agent clients.
* **Negative**: Requires separating edit planning from publication, mapping changesets to `WorkspaceEdit`, and maintaining document versions, downstream sessions, cancellation, and diagnostic completion. Sharing result structures alone does not implement the LSP lifecycle.
