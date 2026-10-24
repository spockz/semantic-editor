# RQ-0008: Structured Documentation & Diagram Refactoring (Markdown, ADRs, Mermaid)

* **Status**: Resolved
* **Category**: Document & Diagram Semantics
* **Last Updated**: 2026-10-22

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

* **`marksman`**: The installed `2026-02-08` build advertises document/workspace symbols, references, and a rename provider. In a disposable project-root probe, it resolved references from inline anchor and wiki links to an H1 heading. Renaming that heading changed the heading but returned no edits for either referencing link. Do not claim cross-link-safe heading rename until the supported syntax and actual edits are verified. The probe also confirmed that cross-file indexing requires a recognized workspace root (VCS root or `.marksman.toml`); Marksman ignores paths matched by repository ignore files by default.
* **`markdown-oxide`**: Fast Rust-based PKM language server with deep link-graph resolution.

### B. Specialized Document Types: ADR Management

ADRs have strict structural schemas and require index synchronization (e.g. updating `docs/adr/README.md` when adding `ADR-0007`).

* **Record fields**: Every new ADR requires a title, concise summary, description, status, date, Context, Decision, Invariants, and Consequences. A structured metadata table records decision role (decision kind), scope, area, and applicable language labels. Document the decision-kind vocabulary and definitions together in ADR-0083: `first-principle` for foundational premises, `principle` for general design guidance, `invariant` for requirements that must remain true, `architecture` for structural choices, `contract` for interface or protocol rules, and `process` for lifecycle or workflow choices. These are known-good values, not an exhaustive enumeration. ADR-0083 defines the taxonomy; it does not collect the project's individual principles. Give each independently adoptable or revisable principle its own ADR, and keep a cohesive set together only when the principles share one rationale and consequence. Known scope values are `project`, `documentation`, `source-code`, and `tooling`; known area values are `backend`, `cli`, `mcp`, `gui`, `benchmark`, `build`, and `language-server`. Language labels remain open-ended, with `go`, `rust`, `scala`, `java`, `haskell`, `kotlin`, `bash`, `makefile`, `markdown`, and `generic` as known-good values.
* **Table contract**: An adjacent HTML comment tells editors to preserve the metadata table's field names and structure. The parser identifies fields by canonical header name rather than column position and fails clearly on missing, duplicate, or renamed required headers; it never guesses. Values remain editable.
* **Tool responsibility**: Use adr-tools for numbering, template-based creation, and ADR links/supersession where its behavior fits the repository. Configure the existing `docs/adr` directory with the repository `.adr-dir` marker and a project template. Test the template creation path with pinned adr-tools 3.0.0, provisioned by `make adr-tools`; `make check` and CI require this target. Do not rely on adr-tools' basic generated title TOC to maintain this repository's richer index. semedit owns deterministic index generation and paginated ADR search.
* **Index and search**: Treat each ADR file as the source of truth and generate the `docs/adr/README.md` decision table from its fields. A `semantic_search_adrs` tool accepts decision context and optional taxonomy filters, returns paginated candidate ADRs with ID, title, status, summary, description, labels, and a path relative to the semedit working directory. Query matching covers all taxonomy labels and the Context, Decision, Invariants, and Consequences sections. The agent can open the returned path with its native file tools when full text or a small edit is needed. No model is required to maintain metadata or the index.
* **Skill responsibility**: Teach the agent the required ADR schema, taxonomy values, creation workflow, and search/read workflow. Keep accepted decisions and the active requirements section curated; generated index rows must not overwrite those authored sections.

### C. Diagram ASTs: Mermaid Validation & Structural Editing

* **Complete:** The repository validates Mermaid fences in Markdown with Marked and Mermaid's `parse` API. The Node check uses jsdom to support Mermaid label sanitization without starting a browser or writing rendered output; the pinned dependencies are installed by `make tools` and run under `make check` and CI.
  * A malformed node label fails with its Markdown file and diagram number before changes are accepted.
* **Structural Diagram Operations**:
  * Deferred. Node renames and edge insertion need a syntax-preserving editing contract; Mermaid syntax validation does not imply structural transformation support. Keep topology changes agent-authored until an AST/CST edit approach is selected and tested.

---

## 4. Sources & Prior Art

* **Marksman (Markdown LSP)**: [GitHub marksman](https://github.com/artempyanykh/marksman): document symbol resolution and cross-file link renaming.
* **Mermaid Parser & AST**: [mermaid.js documentation](https://mermaid.js.org) and [tree-sitter-mermaid](https://github.com/monaqa/tree-sitter-mermaid).
* **ADR Tools by Nat Pryce**: [GitHub adr-tools](https://github.com/npryce/adr-tools): CLI for managing ADR creation and index linking.
* **Unified / Remark (`mdast`)**: [unifiedjs.com](https://unifiedjs.com): Concrete Syntax Tree parser and transformer ecosystem for Markdown.

## 5. Prioritized Delivery Plan

The Markdown/ADR extension is the first priority. After it is complete, finish Go support and conduct independent implementation tracks for the other supported languages in parallel. Luna implements each track; an independent Sol 6.1 agent reviews its work. Follow those tracks with repeated generalization, refactoring, and code-quality sweeps, also reviewed by Sol 6.1. Update the relevant open research question as evidence and decisions change; create or revise an ADR when an architectural choice is accepted and update its index in the same change.

### Priority 1: Markdown and ADR support

1. **Done (bounded evidence):** Probe the installed Marksman LSP and record its actual Markdown boundary. It provides heading symbols and reference lookup, but the tested heading rename did not update inline-anchor or wiki-link references. Do not advertise Markdown rename or build a second link parser until a syntax-specific edit contract is verified. Keep agent-native file editing available for prose-level adjustments. Revisit rename with focused txtar/LSP fixtures if Markdown refactoring becomes an advertised capability.
2. **Complete:** Finalize the ADR record fields, taxonomy values, metadata table contract, and repository template. Template creation and numbering work with `.adr-dir` and `docs/adr/templates/template.md`; `adr link` and `adr new -s` only update a `## Status` section and therefore do not work with the repository's `* **Status**:` metadata line. Use adr-tools for creation and numbering, edit supersession through native file tools, then regenerate the index with semedit.
3. **Complete migration gate:** The initial migration converted all 83 then-existing ADR IDs before Priority 2 began. The current register has 96 active numbered ADR files and the archived revoked-record tombstone for ADR-0023; IDs remain continuous from ADR-0001 through ADR-0097. The integrated `TestReadAllRepositoryRecordsAreMigrated` gate passed against its 117-record snapshot on 2026-10-22. Each current record has the required title, metadata, and sections, plus decision role, scope, area, and applicable open-ended language labels. ADR-0083 centrally defines the known-good decision-kind labels; it does not collect individual principles. A three-part semantic audit checked the prior 113-record snapshot for content-to-metadata fit and corrected the expanded Go construct scope and title for ADR-0011, moved ADR-0028 and ADR-0033 to the `testing` area, shortened ADR-0050's index summary, and repaired unfinished or duplicated descriptions in ADR-0054 and ADR-0068. The audit retained ADR-0049's `generic` language label because it defines shared movement policy while assigning Go and Java implementations to separate decisions. The current final records ADR-0092 through ADR-0097 were checked against their implementation boundaries; no old-format records or missing required sections remain.
4. **Complete:** Deterministic ADR validation and creation/index synchronization generate only the decision-register section from validated ADR files, preserve authored sections, and fail with record-specific errors on malformed metadata or required sections.
5. **Complete:** Paginated `semantic_search_adrs` returns stable ADR IDs and semedit-working-directory-relative paths, searching validated metadata and the Context, Decision, Invariants, and Consequences sections.
6. **Complete:** CLI txtar contracts cover template creation with the pinned adr-tools dependency, migrated records, index updates, taxonomy filters, pagination, malformed metadata, and path safety. ADR skills and references are synchronized.
7. **Complete:** Markdown Mermaid fences are found structurally and validated by Mermaid's parser in the repository check workflow. The validator reports file and diagram identity, passes valid fixtures, rejects invalid syntax, and creates no persistent render artifacts.

## Resolution

The repository-specific boundary is established: ADRs use the validated schema, deterministic index, and paginated search; Markdown headings and links remain available to native agent editing because the probed Marksman rename did not return reference edits; Mermaid fences receive syntax-only validation; semantic graph edits remain deferred until a syntax-preserving edit contract is selected. No new Markdown rename or Mermaid graph mutation capability is advertised.

Further implementation can be reopened when a verified Markdown edit contract or a concrete Mermaid structural transformation is required.

### Priority 2: Parallel language implementation tracks

1. Once Priority 1 is complete, run the remaining Go work from [RQ-0024](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md) and [RQ-0060](RQ-0060-go-refactoring-tool-implementation.md) alongside independent work for each other supported language. Verify current implementation and caller state before selecting each track's gaps; assign file/test ownership so parallel changes do not collide.
2. Go may use additional manual AST/compiler-backed implementation where the backend does not natively provide an operation. Keep each capability bounded, fail closed, and covered by observable CLI txtar contracts.
3. In Swift and every other non-Go backend, preserve existing manual implementations and add new operations only when the selected backend natively supports them and edit-capture, scope, and safety contracts are verified. Record unsupported operations as explicit gaps rather than recreating them manually. Keep Scala at its current implemented scope, tidy existing code, and defer parity work.
4. Add comparable CLI txtar coverage for each newly advertised language-operation pair. Update capability records, skills, relevant research questions, ADRs, and their indexes as decisions are reached.
5. Before a language track is released as complete, Luna runs and repeats its internal cleanup sweeps: consolidate reuse within each backend, then across operations in that language, and perform a code-quality review. A Sol 6.1 agent reviews the implementation and sweep; address findings and repeat review until it finds no actionable issues. Only then does the track join the cross-language generalization sweeps.

### Metadata accuracy checkpoint

A semantic review of the then-current 88 ADR metadata records found and corrected three scope-label mismatches: ADR-0036 now identifies its language-server area rather than build execution, ADR-0045 identifies its workflow role as process rather than architecture, and ADR-0061 is labeled Go only because Java contract generation is governed separately by ADR-0068. ADR-0033 remains `area: generic` because it is a cross-cutting test-process rule and `generic` is the known-good value for decisions without a narrower area. ADR-0071 remains `languages: generic` because its bounded suggestion design spans language-specific implementations and validators; the metadata does not claim identical refactoring capability for each backend.

### Cleanup checkpoints

After each language track passes its internal cleanup gate and Sol 6.1 review, include it in the generalization, refactoring, and code-quality sweeps across completed tracks. Luna performs the sweeps; Sol 6.1 independently reviews them. Consider abstractions shared between languages only after each backend and language has first consolidated its own code; preserve language-specific proof boundaries. Repeat cross-language sweeps and reviews after refactoring because each pass may reveal clearer reuse boundaries or new cleanup. Stop when review finds no actionable correctness, consolidation, or quality issues. Record accepted architecture changes in ADRs and update the corresponding research records and indexes as work progresses.

### 2026-10-23 Go safe-delete defined-type expansion (complete)

Safe deletion now accepts private standalone nongeneric defined types with empty value and pointer method sets across scalar, struct, array, slice, map, channel, pointer, function, and interface underlyings. AST and CLI coverage pass on the managed RAM disk. This boundary is consolidated into ADR-0057, and the research/ADR indexes are synchronized. Full `make check` exited 0 on 2026-10-23, and the reused Sol 6.1 reviewer found no actionable issues.

### 2026-10-22 Go map-range extraction checkpoint

The bounded Go map-range callable extraction, now part of [ADR-0056](../adr/0056-go-call-statement-extraction.md), is complete. AST and standalone, buffered-batch, and unchanged-source refusal CLI coverage pass. Go-local cleanup shares map-operand proof and removes an unused result; Sol 6.1 reviewed the implementation and follow-up fixes with no remaining actionable findings. Full `make check` passed on 2026-10-22. Continue the remaining Go gaps in RQ-0060, then repeat backend-local and cross-language cleanup sweeps before the integrated Gemini final review.

### 2026-10-22 Go slice-identifier range extraction checkpoint

The additive direct local/parameter slice-identifier range form consolidated into [ADR-0056](../adr/0056-go-call-statement-extraction.md) preserves the existing direct slice-factory route. Focused AST and standalone/buffered-batch/refusal CLI coverage pass, including nil/empty slices, nonzero-offset subslices, captured length, backing-array mutation, spare-capacity and forced-allocation append behavior, and observable continue/break behavior. Sol 6.1's implementation, fixture, semantic-oracle, lint-refactor, ADR-boundary, and migrated refusal-fixture findings were resolved; successive re-reviews found no remaining actionable issues. The final elevated `make check` exited 0 on 2026-10-22. Continue the remaining Go gaps and repeat backend-local and cross-language cleanup sweeps before the integrated Gemini final review.

### 2026-10-22 Go two-index slice-expression range lane (complete)

The [ADR-0056](../adr/0056-go-call-statement-extraction.md) implementation and AST boundaries cover direct local/parameter slices ranged as `items[:]`, `items[lo:]`, `items[:hi]`, or `items[lo:hi]`, with pure typed integer bounds and the existing alias/capture/data-flow proofs. Standalone success/refusal fixtures and batch success/refusal fixtures are present. Sol review found no remaining actionable issues, and the integrated elevated `make check` exited 0 on 2026-10-22, including the complete CLI fixture suite.

### 2026-10-22 Go multi-site callable inlining (complete)

[ADR-0097](../adr/0097-bounded-go-multi-site-callable-inlining.md) adds revision-bound selection of multiple exact free-function call sites in eligible direct-owner returns and one-target assignment RHSs. Standalone success, refusal, unsupported-context, and buffered-batch txtars are present. Go-local cleanup and Sol review found no remaining actionable issues, and the integrated elevated `make check` exited 0 on 2026-10-22, including the complete CLI fixture suite.

### 2026-10-22 Go inline-value declaration checkpoint (complete under ADR-0091)

ADR-0091 extends Go computed-value inlining to an adjacent single-name local `var` initializer. The bounded operation preserves type and evaluation semantics, and has standalone/batch CLI coverage, cleanup, clean Sol 6.1 review, and integrated elevated `make check` evidence.

### 2026-10-22 Go three-index slice-range extraction (complete under ADR-0092)

ADR-0092 extends callable extraction to direct local/parameter full-slice range expressions while preserving the original expression and Go's native bound evaluation. Standalone and buffered-batch fixtures cover exact output, nil/empty and capacity-based behavior, iteration-variable shadowing, invalid-bound panic timing, effectful and non-integer maximum refusals, and unchanged source. Sol 6.1 found no remaining actionable issues after a batch-refusal coverage fix; focused CLI coverage and the integrated elevated `make check` passed on 2026-10-22. Continue the remaining Go gaps and repeat backend-local and cross-language cleanup sweeps before the integrated Gemini final review.
