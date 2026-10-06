# Agent Guidelines for `semedit`

This file provides system instructions, operational workflows, and invariants for AI coding agents operating in this workspace.

---

## 1. Project Purpose & Architecture

`semedit` separates high-level LLM reasoning from syntax-level code transformation:

* **The LLM** plans semantic intent (e.g., rename, organize imports, extract function, replace body).
* **The Host Engine** executes deterministic, zero-token AST modifications via compiler and language server APIs (`gopls`).

Before proposing architectural shifts, consult the indexed decisions and research:

* **ADRs**: [`docs/adr/README.md`](docs/adr/README.md)
* **Research Spikes**: [`docs/research/README.md`](docs/research/README.md)

For benchmark harness or benchmark publication changes, start with [the benchmark architecture guide](tools/benchmark-harness/README.md). It maps responsibilities to source files and tests; update the guide when those boundaries move.

---

## 2. Documentation Governance (ADR & Research Lifecycle)

To conserve context budget and maintain architectural integrity, adhere strictly to these rules:

1. **Context Preservation**: Consult index tables first. Read individual ADR or RQ files only when their specific topic is required for the immediate task.
2. **Research Questions (`docs/research/RQ-XXXX-<kebab-case>.md`)**:
   * Create an RQ when exploring open investigations, benchmark measurements, or competing design options.
   * Start with `Status: Open`. When concluded, update to `Status: Resolved` and summarize findings.
3. **Architecture Decision Records (`docs/adr/XXXX-<kebab-case>.md`)**:
   * Create an ADR when an architectural boundary, tool invariant, schema contract, or system design choice is accepted.
   * Required sections: Status, Date, Context, Decision, Invariants, Consequences.
4. **Mandatory Index Atomicity**:
   * Any creation or modification of an ADR or RQ must update the corresponding table in `docs/adr/README.md` or `docs/research/README.md` in the exact same change.
   * Maintain 4-digit sequential zero-padded numbering (`0011-*.md`, `RQ-0014-*.md`).

---

## 3. Tooling & Workspace Invariants

* **Verification Standard**: Always run `make check` prior to executing specific test or verification tools, ensuring automated formatters, fixers, and dependency tidying apply upfront. Verify all changes via `make check` before concluding work.
* **Workspace Manifest Safety (ADR-0005)**: Never write `go.work` or mutate repository workspace manifests on disk without explicit user approval.
* **Atomic Disk Updates (ADR-0010)**: All file mutations must follow atomic write semantics (temporary file $\to$ `fsync` $\to$ `os.Rename`) with advancing `mtime`.
* **Temporary Work**: Place all temporary scratch files in the gitignored `.scratch/` directory.
* **Git Worktrees in Sandbox**: All git worktrees created for parallel tracks or subagents must reside inside `.scratch/worktrees/` within the repository root. This ensures subagents and tools operate entirely inside the primary workspace boundary, eliminating sandbox permission prompts.
* **Error Handling**: Wrap Go errors with context: `fmt.Errorf("...: %w", err)`. Inspect with `errors.Is` / `errors.As`.
* **Git Safety**: Never execute destructive git operations (`git reset --hard`, `git clean -fd`, `git push --force`) without explicit user confirmation.

---

## 4. Problem Solving & Escalation Workflow

When implementing features, refactoring, or diagnosing test and build failures:

1. **Local Iteration Limit**: Attempt up to 3 times to make a chosen solution work. If it still fails, stop editing the symptom site, step back, examine the broader architectural context, and revise the strategy.
2. **Peer Review Escalation**: If the revised strategy also fails, invoke `codex_review_diff` (or `codex_ask`) to request an independent peer review, supplying the problem statement, attempted approaches, and current diff.
3. **Review Application Limit**: Implement the review suggestions and iterate up to 3 additional attempts. If the issue remains unresolved, halt execution and surface the findings, trade-offs, and blocking decision to the user.
4. **Independent Progress Over Blocking**: When a task comprises multiple parts or tools, if blocked on one component, proceed with the independent parts that can already be completed and verified before presenting the blocking question to the user.

---

## 5. Semantic Editing Dogfooding Invariant

When developing or refactoring code inside this repository, agents must dogfood `semedit` semantic tools rather than falling back to text-based file editing:

* **Navigation & Coordinates**: Use `semantic_lookup` to locate declaration coordinates.
* **Symbol Inspection**: Use `semantic_inspect_symbol` to retrieve complete source declarations, signatures, comments, and package imports.
* **Declaration Outline**: Use `semantic_outline` to inspect hierarchical symbol trees for a file or directory with elided function bodies.
* **References & Usages**: Use `semantic_find_references` to discover call sites, type usages, parameter bindings, and field accesses.
* **Renaming**: Use `semantic_rename` for function, method, type, or variable renames instead of search-and-replace.
* **Construct Lifecycle**: Use `semantic_insert_construct` and `semantic_replace_construct` to insert or mutate inner control flow, clauses, branches, loops, and statement/expressions rather than rewriting entire function bodies or text blocks.
* **Function/Method Bodies**: Use `semantic_replace_body` to modify complete function implementations when whole-body replacement is necessary.
* **New Files**: Use `semantic_scaffold_file` to initialize new source files with inferred package headers.
* **Declarations & Imports**: Use `semantic_insert_declaration`, `semantic_insert_function`, `semantic_insert_type`, `semantic_insert_decl`, and `semantic_organize_imports`.
* **Switch Statements**: Use `semantic_insert_construct` with `kind: "clause"` or `semantic_insert_case` to add dispatch branches.
* **Dependencies**: Use `semantic_add_build_dependency` to add external Go modules and tidy `go.mod`.
* **Verification & Diagnostics**: Use `semantic_verify` to confirm workspace cleanliness after modifications.
* **Composite Refactorings**: Use `semantic_batch` to execute multiple semantic transformations in sequence.

### Editing Tools and Python Usage

* **Semantic Changes**: Use the applicable `semedit` tool first for supported source changes and navigation. When a structural operation is unsupported, consider AST tools such as `ast-grep` before text replacement.
* **Surgical Text Edits**: Use built-in write-file and edit-file tools, including `apply_patch`, for documentation, configuration, fixtures, merge conflict resolution, and source edits that semantic tools cannot perform. Keep edits targeted and preserve the atomic write requirements above.
* **Avoid Python for Routine Work**: Do not use Python to read or slice source files, make surgical text replacements, generate patches, or rewrite files when semantic tools, built-in file tools, `rg`, or `fd` can handle the task. Do not replace Python with another general-purpose scripting language to bypass this rule.
* **Justified Scripting**: Reserve Python for tasks that materially need computation, data processing, or protocol/process coordination and are not adequately served by repository commands or available tools. Briefly state why it is needed. A Python transport helper may invoke semantic tools; it must not perform source text rewrites as a substitute for them.
* **Fallback Discipline**: A failed semantic operation does not justify broad scripted rewrites. Log the failure as required below, then use the smallest appropriate fallback.

### Mandatory Logging of Suboptimal Tool Behavior

If any MCP tool call fails, produces incorrect AST output, panics, or requires an immediate manual text edit to touch up or fix the result, agents **must immediately log the occurrence in [`docs/SUBOPTIMAL_TOOLS.md`](docs/SUBOPTIMAL_TOOLS.md)** detailing the tool, target file, observed failure, workaround applied, and root cause before proceeding.

---

## 6. Cross-Language CLI Txtar Coverage Invariant

`testdata/scripts/*.txtar` is the primary executable contract for public semantic operations. Tests must invoke the `semedit` CLI, assert its observable output or diagnostics, and assert resulting workspace state when an operation mutates files. Unit tests that fake an LSP, compiler, or external tool remain necessary for transport and failure isolation, but do not replace CLI txtar coverage.

For every operation advertised by more than one language backend, maintain comparable txtar coverage for every implemented language-operation pair. A language may omit a scenario only when the capability registry marks that operation unsupported; document the intentional asymmetry in the backend capability metadata and its ADR. When adding an operation to a second language, add or update the corresponding CLI txtar cases in the same change.

---

## 7. Language Capability Expansion & Skill Reference Governance Invariant

Automated test suites (`TestSupportedLanguagesHaveSkillReferences`) verify file existence and index links. However, formulating idiomatic refactoring patterns requires deliberate domain reasoning exceeding automated checks.

Whenever expanding the operational scope of a supported language (introducing a new language backend or implementing new operations for an existing backend), agents and contributors must deliberately verify whether and how to update the corresponding language reference guide in [`skills/semedit/references/<lang>.md`](skills/semedit/references/):

* Capture newly enabled macro refactoring patterns and architectural transformations.
* Document compiler, build tool, or language-server verification practices.
* Maintain language-specific idiom guides so agent planners execute high-level workflows cleanly without manual trial and error.
