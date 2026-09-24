# RQ-0003: AST Indexing Latency & Caching

* **Status**: Open
* **Category**: Performance & Indexing
* **Last Updated**: 2026-09-15

---

## 1. Problem Context

Tree-sitter can parse a single $1\text{k}$-line source file in $\sim 1\text{–}3\text{ms}$. However, scanning an entire $100\text{k}\text{–}500\text{k}$ LOC codebase to locate an unqualified symbol query can take several hundred milliseconds or seconds if executed naively from cold storage on every CLI command.

To keep agent turn latency within the target $<50\text{ms}$ flow-state threshold, symbol resolution must be extremely fast.

---

## 2. Exploration Paths

### A. Scoped On-Demand Parsing

* Instead of scanning the whole repo, require the agent to specify the target file or package (`--file pkg/auth/token.go` or `--package auth`).
* Tree-sitter parses *only* the specified file/package in memory ($<5\text{ms}$), eliminating the need for a persistent full-repo index.

### B. In-Memory Persistent Symbol Table

* If full-repo queries (`--symbol GlobalType`) are required without specifying a file:
  * Maintain an in-memory symbol table in the background daemon (`semeditd`).
  * Use OS file watchers (`fsnotify` / `inotify`) to update the index incrementally when files are modified.

---

## 3. Sources & Prior Art

* **Tree-sitter Incremental Parsing**: [Tree-sitter Documentation](https://tree-sitter.github.io/tree-sitter/using-parsers#incremental-parsing).
* **Aider Tree-sitter Repo Map**: [Aider Repo Map Architecture](https://aider.chat/docs/repomap.html): ranks and extracts code symbols efficiently.
* **Rust `ra_ap_syntax` / Rowan**: [rust-analyzer architecture](https://rust-analyzer.github.io/blog): lossless syntax trees with fast incremental edits.
