# Research Questions & Spikes (Index)

This directory tracks open technical investigations, spikes, and design questions for `semedit`.

To conserve LLM context budget during pair-programming sessions, consult this index first. Only read the specific research file relevant to the immediate sub-problem.

---

## Index of Open Research Questions

| ID | Topic | Category | Status | Summary |
| :--- | :--- | :--- | :---: | :--- |
| [RQ-0001](RQ-0001-interactive-snapshot-undo-ux.md) | Interactive Snapshot & Undo UX | UX & Harness | **Resolved** | Content-addressed storage journal, preflight conflict checking, atomic restore, and destructiveHint MCP integration. |
| [RQ-0002](RQ-0002-symbol-addressing-and-disambiguation.md) | Symbol Addressing & Disambiguation | Addressing | **Open** | Disambiguating duplicate or overloaded symbol names without line/col coordinates. |
| [RQ-0003](RQ-0003-ast-indexing-latency-and-caching.md) | AST Indexing Latency & Caching | Performance | **Open** | Evaluating Tree-sitter whole-repo scan latency vs. in-memory daemon cache. |
| [RQ-0004](RQ-0004-lsp-daemon-lifecycle-management.md) | LSP Daemon Lifecycle Management | Runtime | **Open** | Managing persistent background LSPs over Unix sockets vs ephemeral process execution. |
| [RQ-0005](RQ-0005-monorepo-workspace-topologies.md) | Monorepo Workspace Topologies | Workspaces | **Open** | Handling single-language monorepos (`go.work`, Cargo) safely without dirtying git roots. |
| [RQ-0006](RQ-0006-multi-step-refactoring-diagnostic-deltas.md) | Multi-Step Refactorings & Diagnostic Deltas | Workflows | **Resolved** | Managing intermediate breaking states by tracking compiler diagnostic deltas. |
| [RQ-0007](RQ-0007-intentional-variable-unification.md) | Intentional Variable Unification | Semantics | **Open** | Bypassing compiler collision aborts to coalesce duplicate declarations. |
| [RQ-0008](RQ-0008-structured-documentation-and-diagrams.md) | Structured Docs & Diagrams (MD/ADR/Mermaid) | Documents | **Open** | Division of labor between LSP/AST tooling, skills, and LLM for docs and diagrams. |
| [RQ-0009](RQ-0009-read-only-vs-mutating-lsp-in-agents.md) | Read-Only vs. Mutating LSP in Agents | Harness | **Open** | Extending agent planners from read-only LSP consumers into full mutating actors. |
| [RQ-0010](RQ-0010-agent-harness-integration-matrix.md) | Agent Harness Integration Matrix, MCP Discovery & Steering Delivery | Integration | **Open** | Mapping connection protocols, MCP discovery strings (tools/list), instruction distribution, and steering rules across major agent platforms. |
| [RQ-0011](RQ-0011-cache-invalidation-and-file-synchronization.md) | Cache Invalidation & File Synchronization | Systems | **Resolved** | Maintaining consistency across Tree-sitter caches, LSP buffer states, and disk edits. |
| [RQ-0012](RQ-0012-lsp-multiplexing-vs-multi-daemon-orchestration.md) | Ingress Protocols (LSP/MCP/CLI) & Downstream Fan-Out | Architecture | **Open** | Decoupling ingress interfaces from the cross-language refactoring fan-out core. |
| [RQ-0013](RQ-0013-mcp-tool-overlap-and-lsp-coexistence.md) | MCP Tool Overlap & Coexistence | Integration | **Open** | Handling coexistence with raw LSP-MCP servers via distinct naming and profile flags. |
| [RQ-0014](RQ-0014-benchmarking-token-latency-and-eval-harness.md) | Benchmark Methodology, Token/Latency & Eval Harness | Benchmarks | **Open** | Measuring token reduction, latency deltas, harness architectures, and telemetry extraction. |
| [RQ-0015](RQ-0015-multi-dimensional-benchmark-matrix.md) | Multi-Dimensional Benchmark Matrix & Evaluation Runner | Benchmarks | **Open** | Formalizing the 5D evaluation matrix, 10-task testcase suite, AST correctness oracles, and reporting with a direct semedit control. |
| [RQ-0016](RQ-0016-cross-language-access-modifiers-and-section-clustering.md) | Cross-Language Access Modifiers & Section Clustering | Semantics | **Open** | Abstracting visibility modifiers, backend capability declarations, and section placement. |
| [RQ-0017](RQ-0017-language-backend-abstraction-boundaries.md) | Language Backend Abstraction Boundaries | Architecture | **Resolved** | `internal/backend` owns the registered language-neutral service boundary; Go is the only registered runtime backend. |
| [RQ-0018](RQ-0018-automated-code-derived-documentation.md) | Automated Code-Derived Documentation & Test Examples | Documentation | **Open** | Generating capability docs, agent skills, and schemas from code and txtar tests with make check build/output assertions. |
| [RQ-0019](RQ-0019-mkdocs-material-migration.md) | Hugo + Lotus Docs Migration for Code-Derived Documentation | Documentation | **Resolved** | Hugo Markdown pages, standard code-copy controls, pinned modules, direct GitHub source links, dist/docs publication, and verify-docs output assertions are implemented. |
| [RQ-0020](RQ-0020-quality-check-performance-and-parallelization.md) | End-to-End Quality Check Performance & Parallelization | Build & Test Infrastructure | **Open** | Measuring and reducing full `make check` overhead across tests, integration tests, linting, security, docs, package parallelism, and caching. |
| [RQ-0021](RQ-0021-mcp-in-tree-live-reload.md) | In-Tree Live-Reload & Dynamic Tool Schema Discovery | Runtime & Developer Experience | **Resolved** | In-place stdio re-exec via `syscall.Exec` and `notifications/tools/list_changed` for self-modification dogfooding, gated by `--live-reload` (ADR-0017). |
| [RQ-0022](RQ-0022-block-level-semantic-anchors-and-control-flow-slots.md) | Block-Level Semantic Anchors & Control Structure Slots | Semantics | **Open** | Ephemeral handles and slot taxonomy for compound structures (`if`, `for..in`, `range`, `select`, `try-catch`) and location discovery. |
| [RQ-0023](RQ-0023-layered-domain-boundaries-engine-vs-skills.md) | Layered Domain Boundaries: AST Engine vs Framework Skills | Architecture & Agent Steering | **Open** | Strict separation: language-level AST primitives in core engine vs framework idioms (e.g. Akka, Spring, Cobra) in Agent Skills. |
| [RQ-0024](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md) | IDE Edit and Refactoring Capability Taxonomy | Product & Architecture | **Open** | Maps IDE capability families to implemented, language-qualified, and planned semantic-editor capabilities. |
| [RQ-0025](RQ-0025-refactoring-preview-and-bounded-commit.md) | Refactoring Preview and Bounded Commit | Architecture & Safety | **Resolved** | Direct batches report applied diagnostics without rollback; isolated CoW workspace views publish generated multi-file patches through `git apply`. |
