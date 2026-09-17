# semantic-editor (`semedit`)

**Intent-driven code editing for AI agents: LLMs plan intent, host compilers execute zero-token AST refactorings (up to 100x fewer tokens, 10x faster).**

`semedit` separates **semantic intent** (decided by the LLM) from **mechanical syntax transformation** (executed by local host CPU compilers, LSPs, and AST tools).

---

## 1. Problem Statement

Modern coding agents rely on probabilistic token prediction to perform deterministic text manipulation:

* **Token Asymmetry & Churn**: Renaming a symbol across 10 files or extracting a helper function forces the LLM to re-emit hundreds of lines of diffs.
* **Context Exhaustion**: Large diffs consume context window budget, accelerating context compaction and degrading the model's reasoning capacity for subsequent turns.
* **Fragile Transitions**: Line-based search/replace tools frequently fail due to whitespace shifts, unclosed braces, or partial generation, triggering expensive remote repair loops.
* **Provider & Infrastructure Drift**: Hosted model performance fluctuates with diurnal traffic cycles, silent provider-side quantization adjustments, KV-cache eviction policies, and API congestion. Offloading mechanical edits to the host CPU removes this variance from routine syntax operations.

---

## 2. Core Thesis & The Inverted Model

> **LLMs should issue semantic editing commands, not generate textual patches.**

```text
Traditional Agent Flow (Fragile Text Loop):
LLM Plan ──► LLM Streams 1000s of Diff Tokens ──► Network Latency ──► Disk Edit ──► [Syntax Failure / Retry]

Semantic Agent Flow (Deterministic Execution):
LLM Plan ──► LLM Emits 1 Intent (~20 tokens) ──► Host Tooling Transforms Code ──► Compiler Reports Diagnostics
```

Established local development toolchains (compiler typecheckers, language servers such as `gopls`, `rust-analyzer`, `tsserver`, `jdtls`, and AST utilities) already perform precise structural modifications deterministically on the host CPU.

`semedit` bridges AI agents directly to these engines.

---

## 3. Core Hypotheses

1. **Efficiency**: Local CPU code manipulation executes in milliseconds and consumes near-zero remote tokens, costing fractions of a cent per operation.
2. **Determinism**: Semantic transformations preserve syntactic validity across state transitions ($S_n \to S_{n+1}$). Syntax errors and diff-application drift are eliminated for supported operations.
3. **Provider & Network Resiliency**: Offloading editing to the host CPU protects agent sessions from remote API throttling, timeouts, dropped connections, and provider-side inference degradations.
4. **Economics & Planning Horizon Scaling**:
   * *Preserves Context Budget*: Text diffs consume context proportional to file and change size ($\mathcal{O}(\text{diff size})$). Semantic commands scale with intent ($\mathcal{O}(1)$ tokens per edit), preserving context for architecture and reasoning.
   * *Lowers Viability Threshold*: Makes micro-refactoring (renaming, inlining, stubbing) economically viable to delegate.
   * *Preserves Flow State*: Millisecond-range local execution maintains real-time pair programming cadence.
   * *Bounded Blast Radius*: Tooling alters only targeted symbols, eliminating accidental deletions of unrelated lines or comments.

### Efficiency & Turn Mechanics (The "Up to 100x / 10x" Rationale)

The claim of **up to $100\times$ token reduction and $10\times$ lower latency** is derived analytically from the token mechanics of agentic tool loops and observed multi-file refactoring traces:

| Metric | Baseline Agent Text-Diff Flow | `semedit` Semantic Intent Flow | Efficiency Delta |
| :--- | :--- | :--- | :---: |
| **Tool Calls / Turns** | **12–20 turns**: iterative `grep_search` $\to$ `view_file` $\to$ `replace_file_content` per file $\to$ compile repair | **1 turn**: single semantic tool call (`semantic_rename`) | **$10\times - 20\times$ fewer turns** |
| **Output Tokens** | **2,000–3,000 tokens**: verbatim multi-line code diffs across 10+ files | **~25 tokens**: tool arguments only (`symbol`, `new_name`) | **Up to $100\times$ token reduction** |
| **Input Context Churn** | **150k–300k tokens**: quadratic prompt accumulation ($\sum_{t=1}^N \text{history}_t$) as conversation grows | **~1k–5k tokens**: single turn with no accumulated repair history | **$20\times - 50\times$ context savings** |
| **Wall-Clock Latency** | **45–60 seconds**: sequential network round-trips and streaming token generation | **~2–3 seconds**: 1 round-trip + 120ms local AST execution | **$10\times - 20\times$ faster** |
| **Failure / Retry Rate** | **20–30%**: whitespace drift, indentation errors, or unclosed braces | **0%**: compiler-grade deterministic AST modifications | **Eliminates diff drift** |

> [!NOTE]
> **Empirical Validation**: These metrics are analytical upper bounds derived from standard multi-file refactoring operations. Empirical verification across heterogeneous harnesses (Antigravity, Claude Code, direct API), models, and reasoning levels is tracked under [RQ-0014](docs/research/RQ-0014-benchmarking-token-latency-and-eval-harness.md) and [RQ-0015](docs/research/RQ-0015-multi-dimensional-benchmark-matrix.md).

---

## 4. Value-Add on Top of Existing Tooling

Why not just have the LLM call `gopls`, `rust-analyzer`, or `ast-grep` directly?

Existing language servers and CLI tools were built for **interactive human IDE sessions** or **static CI rules**, not autonomous AI agents. `semedit` provides the missing coordination layer:

| Gap / Missing Capability | Why Existing Tools Fall Short | Solution Layer in `semedit` | Implementation Mechanism |
| :--- | :--- | :--- | :--- |
| **1. Intent Addressing (The Coordinate Tax)** | LSPs strictly require exact byte offsets or line/col numbers (`foo.go:42:15`). Models waste 2–3 turns hunting coordinates. | **Symbol Resolver** | Local Tree-sitter AST queries resolve symbol identifiers (`User.SetName`) into precise byte/line/col offsets. |
| **2. Unified Agent Contract** | Each language exposes different CLIs and RPC mechanisms (`gopls` vs `rust-analyzer` vs `tsserver`). | **Broker API / MCP** | One consistent intent schema (CLI & MCP) across Go, Rust, TS, Java, Haskell, Elixir, and Elm. |
| **3. Broken-Code Resilience** | LSPs refuse to start or drop type tables when code has syntax errors, locking the agent out. | **Dual-Engine Dispatcher** | Hierarchical routing: routes to LSP when healthy, drops down to CST pattern tools (`ast-grep`) when uncompilable. |
| **4. Staged Execution & Feedback Loop** | Raw CLI tools mutate files without running formatters or returning unified diagnostic feedback. | **Execution Pipeline** | Stages edits, runs auto-formatters (`gofmt`, `rustfmt`), runs compiler checks, and returns structured diagnostics to the agent. |
| **5. Model Inertia** | Models reflexively generate raw text diffs due to pretraining habits, ignoring tool options. | **Agent Steering Skills** | Ready-to-use `SKILL.md` rules teaching LLMs when to invoke semantic tools over text edits. |

---

## 5. Architecture

`semedit` is **not a new parser, AST library, or rewrite engine**. It is strictly an **orchestration broker and adapter layer** that coordinates existing language servers and utilities.

```text
┌────────────────────────────────────────────────────────────────────────┐
│                          AI Agent / LLM                                │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ High-level Intent (CLI or MCP)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                   Semantic Broker (`semeditd`)                         │
│                                                                        │
│  ┌───────────────────────┐             ┌────────────────────────────┐  │
│  │ Symbol Resolver       │             │ Execution Pipeline         │  │
│  │ (Tree-sitter queries) │             │ (Format, Verify, Snapshot) │  │
│  └───────────┬───────────┘             └─────────────┬──────────────┘  │
└──────────────┼───────────────────────────────────────┼─────────────────┘
               ▼                                       ▼
┌────────────────────────────────────────────────────────────────────────┐
│                          Backend Adapters                              │
├──────────────────────────┬─────────────────────────┬───────────────────┤
│ Tier 1: Compiler / LSP   │ Tier 2: Semantic LST    │ Tier 3: Error-    │
│ (Type-Checked Graph)     │ (Language Frameworks)   │ Tolerant CST      │
├──────────────────────────┼─────────────────────────┼───────────────────┤
│ • Go: gopls              │ • Java: OpenRewrite     │ • ast-grep        │
│ • Rust: rust-analyzer    │ • Python: Rope / LibCST │ • Tree-sitter     │
│ • TS: tsserver           │ • Elixir: Sourceror     │ • Comby           │
│ • Java: Eclipse jdtls    │                         │                   │
│ • Haskell: HLS           │                         │                   │
│ • Elm: elm-lang-server   │                         │                   │
└──────────────────────────┴─────────────────────────┴───────────────────┘
```

### Four Primary Components

1. **Agent Interface & Session Broker**:
   Exposes high-level commands over both CLI (`semedit rename`) and MCP (`semedit mcp`). Manages daemon lifecycle and session state.
2. **Symbol Resolver (Intent Addressing)**:
   Uses Tree-sitter queries to index symbols within files. Translates high-level agent targets (`TokenService.Validate`) into exact file/byte offsets required by LSPs.
3. **Execution & Diagnostic Pipeline**:
   * Applies the transformation to the file system.
   * Automatically executes language formatters (`gofmt`, `rustfmt`, `prettier`, `ruff`).
   * Runs compiler/typechecker passes and returns structured diagnostics directly to the LLM.
   * Provides on-demand snapshotting/undo if the agent or user decides to revert a change. (Edits do *not* automatically revert on compile errors, allowing multi-step refactorings that temporarily break code).
4. **Backend Adapters**:
   * **Tier 1 (Compiler/LSP)**: Direct adapters for language servers (`gopls`, `rust-analyzer`, `tsserver`, `jdtls`, `HLS`, `elm-language-server`).
   * **Tier 2 (Semantic Frameworks)**: Adapters for rich language-specific AST engines (`OpenRewrite`, `Rope`, `Sourceror`).
   * **Tier 3 (Error-Tolerant CST)**: Fallback rewriters (`ast-grep`, `Comby`) used when the codebase contains syntax errors that prevent compiler tools from running.

---

## 6. Language Strategy & Ecosystem Roadmap

`semedit` does not implement a universal AST. Syntax trees cannot resolve cross-file types or inheritance. We build a universal **intent broker** that delegates to native compiler tools.

### Tier 1: Initial Targets

| Language | Primary Engine | Mechanism | Strategic Notes |
| :--- | :--- | :--- | :--- |
| **Go** | `gopls` | Native CLI & LSP | Delegates to `gopls` directly (see section below). |
| **Rust** | `rust-analyzer` | Headless stdio LSP | Code actions: extract function, inline, assists, rename. |
| **TypeScript / JS** | `tsserver` | JSON Server Protocol | Direct `getEditsForRefactor`, file renames, organize imports. |
| **Java** | Eclipse `jdtls` / `OpenRewrite` | LSP / Gradle / Maven | JDT for interactive edits; OpenRewrite for repository migrations. |

### Tier 2: Secondary & Functional Language Expansion

* **Python**: `Rope` + `LibCST` (heuristic type resolution; dynamic runtime bounds).
* **C#**: `Roslyn` (compiler-as-a-library; premier refactoring engine).
* **Haskell**: `Haskell Language Server (HLS)` for explicitly standalone, trusted, read-only hierarchical `.hs` symbol lookup. Project cradles, compilation, diagnostics, rename, `retrie`, and `hlint` remain unavailable.
* **Elixir**: `ElixirLS` / `Lexical` (LSP) for symbol navigation and refactorings paired with `Sourceror` / `Igniter` for lossless AST rewriting that preserves comments and formatting.
* **Elm**: `elm-language-server` paired with `elm-review --fix` and `elm-format`. Elm's compiler provides exceptionally deterministic error payloads, making the automated verification loop nearly zero-friction.
* **Dart**: `Dart Analysis Server` (explicit `edit.getRefactoring` RPC protocol).
* **Scala**: `Metals` + `Scalafix`.

### Explicit Non-Goals (Phase 1)

* **Concurrent Polyglot Language Servers**: Running multiple language servers for different languages concurrently in a single workspace is postponed. Focus is strictly on single-language workspaces (though single-language multi-project repos are supported).
* **C / C++**: Complex compilation databases (`compile_commands.json`) and preprocessor macros require heavy per-project setup.
* **Untyped languages without mature AST refactoring libraries**: Low ROI for strict semantic validation.

---

## 7. How We Handle Go: Delegation vs. Reinvention

`gopls` already provides a CLI (`gopls rename`, `gopls codeaction`) and an experimental MCP server (`gopls mcp`).

**We do not reinvent Go semantic refactoring.** `semedit` delegates Go operations directly to `gopls`:

```text
LLM Intent: `semedit rename --symbol "Server.Start" --to "Serve"`
                               │
                ┌──────────────┴──────────────┐
                │ If gopls MCP is active:     │
                │   Short-circuit to gopls    │
                │ Else:                       │
                │   Drive gopls CLI / LSP     │
                └──────────────┬──────────────┘
                               │
               Resolve symbol coordinates via AST
                               │
            Execute `gopls rename -w file.go:#offset`
                               │
            Auto-run `goimports` + report diagnostics
```

### What `semedit` Adds on Top of `gopls`

1. **Coordinate Resolution**: `gopls` requires byte offsets or line/col numbers. `semedit` resolves symbol names to offsets via local Tree-sitter indexing.
2. **Unified Agent Contract**: The agent uses the same commands across Go, Rust, and TypeScript.
3. **Execution Pipeline**: `gopls` modifies files directly without running formatting, test validation, or reporting structured diagnostic summaries back to the agent.

---

## 8. Deliverables: Engine + Skills

A tool catalog alone is insufficient. Due to pre-training habits, frontier models default to generating raw diffs even when semantic tools are available.

`semedit` ships two synchronized artifacts:

1. **The Execution Binary (`semedit`)**:
   * **CLI Mode**: `semedit rename`, `semedit extract`, `semedit verify`. Sub-millisecond shell commands for scripts and agents.
   * **MCP Server Mode**: `semedit mcp`. Exposes high-level tools to Claude Code, Cursor, Windsurf, Antigravity, and other MCP clients.
2. **The Steering Layer (`SKILL.md`)**:
   * Agent rules and decision trees teaching models when to invoke semantic tools over text edits.
   * Prompts and heuristics for formulating symbol-based queries without hallucinating line numbers.

---

## 9. Proposed Project Layout

```text
semedit/
├── cmd/
│   └── semedit/            # Main entry point (CLI and `semedit mcp` daemon)
├── internal/
│   ├── broker/             # Intent router and session management
│   ├── symbol/             # Tree-sitter locator (symbol query -> file:offset)
│   ├── pipeline/           # Staging, formatting, diagnostics, snapshotting
│   ├── fallback/           # Tier 3 syntax rewriters (ast-grep / comby integration)
│   └── adapters/           # Language-specific drivers
│       ├── golang/         # gopls CLI / LSP driver
│       ├── rust/           # rust-analyzer headless client
│       ├── typescript/     # tsserver JSON driver
│       ├── java/           # jdtls / OpenRewrite runner
│       ├── haskell/        # HLS / retrie runner
│       ├── elixir/         # ElixirLS / Sourceror runner
│       └── elm/            # elm-language-server runner
├── skills/
│   └── semedit/            # Agent skill definitions (SKILL.md, system rules)
├── Makefile                # Standardized check, test, build recipes
└── README.md
```

---

## Appendix: Prior Art & Tooling Capability Matrix

The table below contrasts existing tools across the three tiers against `semedit`:

| Tool / System | Primary Tier & Scope | Type & Compiler Aware? | Resilient to Syntax Errors? | Intent Addressing (No Line/Col)? | Diagnostic Feedback Loop? | Multi-Language Unified? | Core Strength / Limitations |
| :--- | :--- | :---: | :---: | :---: | :---: | :---: | :--- |
| **`gopls`** | Tier 1 (Go LSP / CLI) | **Yes** | No | No (requires byte/line/col) | Raw stderr | No (Go only) | Native Go compiler backing; raw CLI requires offset calculation. |
| **`rust-analyzer`** | Tier 1 (Rust LSP) | **Yes** | Partial | No (requires text ranges) | LSP diagnostics | No (Rust only) | Industry standard Rust refactorings; no standalone edit CLI. |
| **`tsserver`** | Tier 1 (TypeScript) | **Yes** | Partial | No (requires line/col) | JSON events | No (TS/JS only) | First-class refactor API; protocol coupled to editor state. |
| **Eclipse `jdtls`** | Tier 1 (Java LSP) | **Yes** | No | No (requires line/col) | LSP diagnostics | No (Java only) | Deepest classic refactoring catalog; heavy startup overhead. |
| **`OpenRewrite`** | Tier 2 (Java / Polyglot) | **Yes** (LST) | No | **Yes** (Recipe queries) | Build logs | Partial | Gold standard for repo-wide migrations; slow for one-off edits. |
| **`Rope`** | Tier 2 (Python) | Heuristic | No | Partial (Python scopes) | Python exceptions | No (Python only) | Best-in-class Python semantic refactorer; dynamic typing limits. |
| **`HLS`** | Tier 1 (Haskell lookup) | **Yes** | No | Partial | Not requested | No (Haskell only) | Standalone UTF-16 hierarchical document symbols; project cradles and source mutation remain outside the slice. |
| **`ElixirLS` / `Sourceror`** | Tier 1/2 (Elixir) | Partial | Partial | Partial (Sourceror AST) | Mix diagnostics | No (Elixir only) | Lossless CST preserves comments/formatting; homoiconic AST transforms. |
| **`elm-language-server`** | Tier 1 (Elm LSP) | **Yes** | No | No (requires line/col) | Elm compiler JSON | No (Elm only) | Pure compiler guarantees; exceptionally deterministic diagnostic payloads. |
| **`ast-grep` (`sg`)** | Tier 3 (Polyglot CST) | No | **Yes** | **Yes** (Pattern queries) | Syntax check only | **Yes** (Tree-sitter) | Extremely fast pattern rewrites; type-blind across packages. |
| **`GritQL` / Marzano** | Tier 3 (Polyglot CST) | Partial | **Yes** | **Yes** (Declarative pattern) | Dry-run diffs | **Yes** (Tree-sitter) | Declarative AST transformations; lacks full compiler type graph. |
| **`Comby`** | Tier 3 (Polyglot Syntax) | No | **Yes** (Maximum) | **Yes** (Delimiter matching) | None | **Yes** (Any language) | Indestructible syntax search/replace; zero semantic awareness. |
| **`agent-lsp`** | Agent Bridge (MCP) | **Yes** | No | Partial (has symbol lookup) | LSP diagnostics | **Yes** (30+ LSPs) | Exposes raw LSP to agents; lacks broken-code fallback & intent cache. |
| **`semedit` (This Project)** | **Unified Orchestrator** | **Yes** (Tier 1) | **Yes** (Tier 3 Fallback) | **Yes** (Symbol resolver) | **Yes** (Automated pipeline) | **Yes** (Go, Rust, TS, Java, Haskell, Elixir, Elm, Python) | Bridges compiler power to agents with intent routing and safety guards. |

### Detailed Notes on Tool Specializations

1. **Why `gopls` / `rust-analyzer` are not competitors**:
   `semedit` does not implement type-checkers or compiler front-ends. It uses `gopls` and `rust-analyzer` as backend execution workers. The value is bridging the gap between an LLM's natural way of thinking ("Rename function X") and the language server's rigid requirements ("Offset 1042 in buffer 3").
2. **Why `ast-grep` is a partner, not a replacement**:
   Tree-sitter and `ast-grep` are exceptionally good at parsing invalid code and finding structural patterns, but cannot distinguish between two methods with identical names belonging to different types across modules. `semedit` pairs `ast-grep` for local structural repairs with LSPs for project-wide semantic edits.
3. **Why `OpenRewrite` complements interactive refactoring**:
   `OpenRewrite` operates at the batch/recipe layer (e.g. migrating 500 files to a new logging framework). `semedit` delegates batch migration tasks in Java to OpenRewrite while using `jdtls` for granular, interactive single-step edits.
4. **Functional & Pure Language Advantages (Haskell, Elixir, Elm)**:
   * **Haskell**: Enables *equational rewriting* via `retrie`: transformations can replace expressions according to algebraic laws while GHC guarantees semantic equivalence.
   * **Elixir**: Homoiconic syntax enables lossless AST manipulation via `Sourceror` and `Igniter`, allowing the agent to perform safe pattern replacements that respect comments and formatting.
   * **Elm**: The Elm compiler is famous for generating the most precise, human-readable, and machine-parsable error messages in software engineering. Verification and automated diagnostic repair loops are simpler in Elm than in virtually any other ecosystem.

For architectural invariants and historical decisions, see the [Architecture Decision Records (ADRs)](docs/adr/README.md).
For unresolved spikes and open technical challenges, see the [Research Questions Index](docs/research/README.md).
For development workflow, testing standards, and test authoring guides, see [CONTRIBUTING.md](CONTRIBUTING.md).
