# Architecture Decision Records (ADRs)

This directory contains records of all foundational architectural decisions, system invariants, and constraints for `semedit`.

To preserve LLM context budget during development, consult this index first and view individual ADR files only when detailed context on a specific decision is required.

---

## Index of Decisions

| ID | Title | Status | Date | Core Invariant / Summary |
| :--- | :--- | :---: | :---: | :--- |
| [ADR-0001](0001-intent-driven-orchestration.md) | Intent-Driven Orchestration vs. Text Patching | **Accepted** | 2026-09-15 | LLMs emit semantic intents; local host tools execute syntax transformations. |
| [ADR-0002](0002-dual-engine-architecture.md) | Dual-Engine Architecture (LSP + Error-Tolerant CST) | **Accepted** | 2026-09-15 | Route to LSP when compilable; drop to CST (`ast-grep`) when syntax is broken. |
| [ADR-0003](0003-symbol-based-addressing.md) | Symbol-Based Addressing Over File Coordinates | **Accepted** | 2026-09-15 | Models target symbol identifiers (`Type.Method`); Tree-sitter resolves byte offsets. |
| [ADR-0004](0004-staged-execution-and-diagnostic-feedback.md) | Staged Execution & Diagnostic Reporting (No Forced Rollback) | **Accepted** | 2026-09-15 | Report compiler diagnostics to the agent; never unilaterally roll back intermediate breaking edits. |
| [ADR-0005](0005-workspace-manifest-safety-and-monorepo-scoping.md) | Workspace Manifest Safety & Monorepo Scoping | **Accepted** | 2026-09-15 | Never write `go.work` or workspace manifests on disk without user approval; postpone concurrent polyglot LSPs. |
| [ADR-0006](0006-dual-interface-delivery-and-agent-skills.md) | Dual-Interface Delivery (CLI + MCP) Paired with Skills | **Accepted** | 2026-09-15 | Provide both CLI and MCP server, packaged with `SKILL.md` to overcome model diff inertia. |
| [ADR-0007](0007-semantic-mcp-tool-naming-and-descriptions.md) | Semantic MCP Tool Naming & Schema Descriptions | **Accepted** | 2026-09-15 | Name tools after high-level semantic intents to guide planner routing over text replacement. |
| [ADR-0008](0008-decoupling-ingress-interfaces-from-fan-out-engine.md) | Decoupling Ingress Interfaces from Fan-Out Engine | **Accepted** | 2026-09-15 | Core multi-domain refactoring is protocol-agnostic; LSP, MCP, and CLI execute the exact same fan-out logic. |
| [ADR-0009](0009-mcp-coexistence-and-tool-scoping.md) | Coexistence with LSP-MCP Servers via Distinct Naming & Profiles | **Accepted** | 2026-09-15 | Coexist with existing LSP-MCP servers using distinct intent names and `--profile` scoping. |
| [ADR-0010](0010-disk-synchronization-and-cache-invalidation.md) | Disk Synchronization & Cache Invalidation Across Heterogeneous Tools | **Accepted** | 2026-09-15 | Enforce atomic write/rename, advancing timestamps, and synchronous Tree-sitter mtime checks for cross-tool consistency. |
| [ADR-0011](0011-ast-declaration-insertion-and-import-management.md) | Top-Level Declaration Insertion and Automated Import Management | **Accepted** | 2026-09-16 | Parse validation, public-precedes-private section placement, and import optimization. |
| [ADR-0012](0012-access-modifiers-and-section-placement.md) | Cross-Language Access Modifiers and Section Placement | **Accepted** | 2026-09-16 | Universal access modifier enum, backend capability declarations, and strict section partitioning. |
