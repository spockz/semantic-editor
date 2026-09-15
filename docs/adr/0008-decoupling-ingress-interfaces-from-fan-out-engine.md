# ADR-0008: Decoupling Ingress Interfaces from Downstream Fan-Out Engine

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Refactoring in modern projects often spans multiple file types (e.g. updating a function definition in `.go` and updating references in `.md` documentation).

Initial designs conflated the **ingress interface** (MCP vs. LSP) with the **transformation behavior** (single-tool routing vs. cross-domain coordination). However, how a client triggers an edit is orthogonal to how the refactoring engine resolves and transforms downstream files.

## Decision

1. **Three-Layer Decoupling**:
   * **Layer 1 (Ingress)**: Pluggable protocol adapters (MCP, LSP, CLI).
   * **Layer 2 (Core Engine)**: Target resolution, multi-domain fan-out, changeset merging, formatting, and diagnostics.
   * **Layer 3 (Downstream Adapters)**: Dedicated language engines (`gopls`, `marksman`, `rust-analyzer`, Tree-sitter).
2. **Symmetric Protocol Execution**:
   * An incoming `textDocument/rename` over LSP triggers the exact same multi-domain fan-out engine as an incoming `semantic_rename` over MCP or a `semedit rename` CLI call.
   * Cross-domain edits (updating code and documentation simultaneously) are returned as a unified `WorkspaceEdit` over LSP, or as an atomic changeset over MCP and CLI.

## Invariants

* Core refactoring and fan-out logic must be completely agnostic of the ingress protocol.
* Any cross-domain refactoring capability available via MCP must be identically invocable via LSP and CLI.

## Consequences

* **Positive**: LSP multiplexing is simply an ingress adapter over the core engine; enables seamless cross-language refactoring across both IDEs and AI agent harnesses.
* **Negative**: Requires mapping core changeset objects cleanly into LSP `WorkspaceEdit` structures.
