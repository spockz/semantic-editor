# Research Questions & Spikes (Index)

This directory tracks open technical investigations, spikes, and design questions for `semedit`.

To conserve LLM context budget during pair-programming sessions, consult this index first. Only read the specific research file relevant to the immediate sub-problem being solved.

---

## Index of Open Research Questions

| ID | Topic | Category | Status | Summary |
| :--- | :--- | :--- | :---: | :--- |
| [RQ-0001](RQ-0001-interactive-snapshot-undo-ux.md) | Interactive Snapshot & Undo UX | UX & Harness | **Open** | Presenting on-demand snapshots/undo as a 1-click native UI choice without typing. |
| [RQ-0002](RQ-0002-symbol-addressing-and-disambiguation.md) | Symbol Addressing & Disambiguation | Addressing | **Open** | Disambiguating duplicate or overloaded symbol names without line/col coordinates. |
| [RQ-0003](RQ-0003-ast-indexing-latency-and-caching.md) | AST Indexing Latency & Caching | Performance | **Open** | Evaluating Tree-sitter whole-repo scan latency vs. in-memory daemon cache. |
| [RQ-0004](RQ-0004-lsp-daemon-lifecycle-management.md) | LSP Daemon Lifecycle Management | Runtime | **Open** | Managing persistent background LSPs over Unix sockets vs ephemeral process execution. |
| [RQ-0005](RQ-0005-monorepo-workspace-topologies.md) | Monorepo Workspace Topologies | Workspaces | **Open** | Handling single-language monorepos (`go.work`, Cargo) safely without dirtying git roots. |
| [RQ-0006](RQ-0006-multi-step-refactoring-diagnostic-deltas.md) | Multi-Step Refactorings & Diagnostic Deltas | Workflows | **Resolved** | Managing intermediate breaking states by tracking compiler diagnostic deltas. |
| [RQ-0007](RQ-0007-intentional-variable-unification.md) | Intentional Variable Unification | Semantics | **Open** | Bypassing compiler collision aborts to coalesce duplicate declarations. |
| [RQ-0008](RQ-0008-structured-documentation-and-diagrams.md) | Structured Docs & Diagrams (MD/ADR/Mermaid) | Documents | **Open** | Division of labor between LSP/AST tooling, skills, and LLM for docs and diagrams. |
| [RQ-0009](RQ-0009-read-only-vs-mutating-lsp-in-agents.md) | Read-Only vs. Mutating LSP in Agents | Harness | **Open** | Extending agent planners from read-only LSP consumers into full mutating actors. |
| [RQ-0010](RQ-0010-agent-harness-integration-matrix.md) | Agent Harness Integration Matrix | Integration | **Open** | Mapping connection protocols and steering rule formats across major agent platforms. |
| [RQ-0011](RQ-0011-cache-invalidation-and-file-synchronization.md) | Cache Invalidation & File Synchronization | Systems | **Resolved** | Maintaining consistency across Tree-sitter caches, LSP buffer states, and disk edits. |
| [RQ-0012](RQ-0012-lsp-multiplexing-vs-multi-daemon-orchestration.md) | Ingress Protocols (LSP/MCP/CLI) & Downstream Fan-Out | Architecture | **Open** | Decoupling ingress interfaces from the cross-language refactoring fan-out core. |
| [RQ-0013](RQ-0013-mcp-tool-overlap-and-lsp-coexistence.md) | MCP Tool Overlap & Coexistence | Integration | **Open** | Handling coexistence with raw LSP-MCP servers via distinct naming and profile flags. |
| [RQ-0014](RQ-0014-benchmarking-token-latency-and-eval-harness.md) | Benchmark Methodology, Token/Latency & Eval Harness | Benchmarks | **Open** | Measuring empirical token reduction, latency deltas, and failure rates across A/B arms. |
| [RQ-0015](RQ-0015-multi-dimensional-benchmark-matrix.md) | Multi-Dimensional Benchmark Matrix & Evaluation Runner | Benchmarks | **Open** | Formalizing the 5D evaluation matrix (harness, model, reasoning, ingress, testcase) and runner. |
| [RQ-0016](RQ-0016-cross-language-access-modifiers-and-section-clustering.md) | Cross-Language Access Modifiers & Section Clustering | Semantics | **Open** | Abstracting visibility modifiers, backend capability declarations, and section placement. |
