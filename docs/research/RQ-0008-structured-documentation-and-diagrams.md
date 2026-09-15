# RQ-0008: Structured Documentation & Diagram Refactoring (Markdown, ADRs, Mermaid)

* **Status**: Open
* **Category**: Document & Diagram Semantics
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

Software projects are not code alone; documentation, decision records (ADRs), and architectural diagrams (e.g. Mermaid) are core project artifacts. When refactoring code, documentation easily drifts:

* Renaming symbols or files breaks markdown links and heading references.
* Adding ADRs requires updating markdown index tables, often leading to formatting errors and merge conflicts.
* Editing Mermaid diagrams via raw text frequently results in syntax errors (e.g. unescaped parentheses in node labels) that break rendering.

The research question: **How should `semedit` handle structured document formats like Markdown, ADRs, and Mermaid, and where is the boundary between local host tooling, agent skills, and the LLM?**

---

## 2. Division of Labor: Tool vs. Skill vs. LLM

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        Division of Labor                               │
├───────────────────────┬───────────────────────┬────────────────────────┤
│ Host Tooling / LSP    │ Agent Skill (SKILL.md)│ LLM                    │
├───────────────────────┼───────────────────────┼────────────────────────┤
│ • Cross-link renaming │ • ADR template schema │ • Architectural intent │
│ • Table formatting    │ • Index sync rules    │ • Diagram topology     │
│ • Mermaid syntax lint │ • Heading hierarchies │ • Rationales & prose   │
│ • Frontmatter parsing │ • Review checklists   │ • Review decisions     │
└───────────────────────┴───────────────────────┴────────────────────────┘
```

---

## 3. Exploration Paths

### A. Markdown as a First-Class Language Server

Markdown already has dedicated Language Servers:

* **`marksman`**: A compiler-like LSP for Markdown that maintains a cross-document link graph. Supports:
  * Renaming document headings and automatically updating all referencing wiki-links/anchor links across the repo.
  * Cross-file reference diagnostics (dead links, broken anchor tags).
* **`markdown-oxide`**: Fast Rust-based PKM language server with deep link-graph resolution.

### B. Specialized Document Types: ADR Management

ADRs have strict structural schemas and require index synchronization (e.g. updating `docs/adr/README.md` when adding `ADR-0007`).

* **Tool Responsibility**:
  * A deterministic command (`semedit doc adr add "Title"`) that generates the next numbered file and automatically updates the index table without token churn.
* **Skill Responsibility**:
  * Teaches the agent the required ADR sections (`Context`, `Decision`, `Invariants`, `Consequences`).

### C. Diagram ASTs: Mermaid Validation & Structural Editing

* **Mermaid Syntax Validation**: Mermaid CLI (`@mermaid-js/mermaid-cli` / `mmdc`) or a lightweight Tree-sitter grammar (`tree-sitter-mermaid`) can validate diagrams locally.
  * If the LLM generates an invalid node label (e.g. `A[Node (with parens)]` without quotes), local validation catches it immediately and reports the syntax error before saving.
* **Structural Diagram Operations**:
  * Investigating whether common graph operations (e.g. renaming a node ID across arrows, inserting an intermediate node) can be performed via CST search/replace rather than full diagram regeneration.

---

## 4. Sources & Prior Art

* **Marksman (Markdown LSP)**: [GitHub marksman](https://github.com/artempyanykh/marksman) — document symbol resolution and cross-file link renaming.
* **Mermaid Parser & AST**: [mermaid.js documentation](https://mermaid.js.org) and [tree-sitter-mermaid](https://github.com/monaqa/tree-sitter-mermaid).
* **ADR Tools by Nat Pryce**: [GitHub adr-tools](https://github.com/npryce/adr-tools) — CLI for managing ADR creation and index linking.
* **Unified / Remark (`mdast`)**: [unifiedjs.com](https://unifiedjs.com) — Concrete Syntax Tree parser and transformer ecosystem for Markdown.
