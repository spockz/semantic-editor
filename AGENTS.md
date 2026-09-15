# Agent Guidelines for `semedit`

This file provides system instructions and operating invariants for AI coding agents operating in this workspace.

---

## 1. Project Purpose & Architecture

`semedit` separates high-level LLM reasoning from syntax-level code transformation:

* **The LLM** plans semantic intent (e.g., rename, organize imports, extract function).
* **The Host Engine** executes deterministic, zero-token AST modifications via compiler and language server APIs (`gopls`).

Before proposing architectural shifts, consult the indexed decisions and research:

* **ADRs**: [docs/adr/README.md](docs/adr/README.md)
* **Research Spikes**: [docs/research/README.md](docs/research/README.md)

---

## 2. Documentation Governance (ADR & Research Lifecycle)

To conserve context budget and maintain architectural integrity, follow these rules:

1. **Context Preservation**: Always consult the index tables first. Never read individual ADR or RQ files unless their specific topic is required for your immediate task.
2. **Research Questions (`docs/research/RQ-XXXX-<kebab-case>.md`)**:
   * Create an RQ when exploring open investigations, benchmark measurements, or competing design options.
   * Start with `Status: Open`. When concluded, update to `Status: Resolved` and summarize findings.
3. **Architecture Decision Records (`docs/adr/XXXX-<kebab-case>.md`)**:
   * Create an ADR when an architectural boundary, tool invariant, schema contract, or system design choice is accepted.
   * Required sections: Status, Date, Context, Decision, Invariants, Consequences.
4. **Mandatory Index Atomicity**:
   * Any creation or modification of an ADR or RQ **must** update the corresponding table in `docs/adr/README.md` or `docs/research/README.md` in the exact same change.
   * Maintain 4-digit sequential zero-padded numbering (`0011-*.md`, `RQ-0014-*.md`).

---

## 3. Tooling & Workspace Invariants

* **Verification Standard**: Always verify changes via `make check` before concluding work.
* **Workspace Manifest Safety (ADR-0005)**: Never write `go.work` or mutate repository workspace manifests on disk without explicit user approval.
* **Atomic Disk Updates (ADR-0010)**: All file mutations must follow atomic write semantics (temporary file $\to$ `fsync` $\to$ `os.Rename`) with advancing `mtime`.
* **Temporary Work**: Place all temporary scratch files in the gitignored `.scratch/` directory.
* **Error Handling**: Wrap Go errors with context: `fmt.Errorf("...: %w", err)`. Inspect with `errors.Is` / `errors.As`.
* **Git Safety**: Never execute destructive git operations (`git reset --hard`, `git clean -fd`, `git push --force`) without explicit user confirmation.
