<!-- Maintains the capability audit and resumable refactoring handover map so existing work is reused before new operations are planned. -->
# RQ-0024: IDE Edit and Refactoring Capability Taxonomy

* **Status**: Open
* **Category**: Product & Architecture
* **Date**: 2026-09-17

---

## Context

IDE documentation groups semantic operations into capability families rather than individual commands. `semedit` needs the same vocabulary to distinguish implemented operations from future work without treating a language-specific command as a universal promise.

## Research Questions

1. Which IDE capability families should `semedit` expose as edit or refactoring capabilities?
2. Which families have a safe Go-first implementation boundary, and which require a language-server or compiler-specific backend?
3. Which capabilities need in-operation conflict detection or atomic multi-file mutation, and which previews belong to the caller or controller?
4. How should safe deletion be expressed and bounded across structural node types and language backends?
5. Which high-intent composite operations (e.g. scaffolding test fixtures, implementing interface/trait contracts, member additions) should be exposed to eliminate multi-turn refactoring loops without causing engine bloat or cross-language schema pollution?
6. How should composite tools balance universal developer intent against target-language syntactic and idiomatic differences?

## Audited Capability Taxonomy

| IDE Capability Family | Audited `semedit` Mapping | Remaining investigation |
| :--- | :--- | :--- |
| Navigation and symbol lookup | Go inspection, outline and references; bounded other-language lookup | Preserve explicit language/operation limits. |
| Rename | Go module-wide retained-session rename, including shared batch overlays; selected-file Rust/Java rename | Do not reimplement Go rename; other-language workspace scope is separate. |
| Structural insertion/replacement/removal | Go declarations, receiver methods, bodies, constructs and exact declaration/case removal | Focused fields, statements, composite elements and documentation comments remain missing. |
| High-intent composites | Package-inferred file scaffold; supplied method/type insertion | Generate missing interface methods without duplicating method insertion. |
| Extract and introduce | Bounded Go function/method and variable extraction | Broader data flow and expression positions; constant/interface/type extraction and introduced parameters. |
| Inline | Bounded Go literal local-variable inlining | Function/call inlining and computed/distant variable uses. |
| Move and copy | Go declaration movement; bounded exported free-function movement between packages | Typed dependency groups, types/methods and whole-package/resource operations. Copy is not advertised. |
| Signature and type transformation | No dedicated change-signature/type-migration operation | Caller, assignment and compatibility proofs. |
| Hierarchy refactoring | No implementation | Java/Rust-specific hierarchy operations need separate backend contracts; do not invent inheritance for Go. |
| Preview/conflicts | Direct batches, final persisted diffs, diagnostic deltas and conflict-checked snapshots | Isolation/speculative execution remain caller-owned; no whole-batch transaction promise. |

## Finding: Structural Removal and Safe Delete (2026-09-24)

Deletion should be offered as the structural counterpart to insertion, with removal operations targeting exact declarations, statements, cases, or collection elements. Safe deletion is an option on a supported removal operation, such as `safe_delete: true`; it is not a parallel family of safe-delete methods.

Without `safe_delete`, the operation removes only the requested structural node and still validates the result and reports diagnostics. With `safe_delete`, the backend performs the strongest relevant preflight it supports, including reference completeness and resource constraints where applicable. It must refuse the mutation if the requested safety checks fail or cannot establish the advertised guarantee, and report which checks ran. For statements, cases, and collection elements, structural or diagnostic validation does not prove behavioral equivalence.

The default for `safe_delete` and the required checks remain operation- and backend-specific. For direct multi-file edits, preserve RQ-0025's boundary: semantic-editor applies the authorized operation and reports its diff and diagnostics; speculative preview and workspace-copy workflows belong to the caller or controller. Reference and resource preflights remain part of safe deletion itself.

## Finding: High-Intent Composite Operations & Cross-Language Alignment (2026-09-25)

To reduce agent turn counts and token churn, refactoring workflows often call for composite mutations (e.g. creating a test file with sample test cases and assertions, or generating all required methods to satisfy an interface). However, baking framework- or language-specific composites into the core engine creates severe design tension:

1. **The Language Specificity Dilemma**:
   * Generic low-level tools (`replace_file_content`, line edits) impose heavy token taxes and context loss.
   * Overly specialized tools (`scaffold_cobra_subcommand`, `add_spring_controller_endpoint`) cause engine bloat, tight version coupling, and schema pollution for other languages.
2. **Universal Semantic Intents vs. Language Syntactic Realizations**:
   Instead of inventing divergent tools per language, composite operations should be defined around **universal developer refactoring intents**:
   * `semantic_add_member`: Adds a receiver method or struct field in Go; adds a method or property in Java/Rust.
   * `semantic_implement_contract`: Generates missing method declarations satisfying an `interface` in Go; generates `@Override` stubs in Java or `impl Trait` blocks in Rust.
   * `semantic_scaffold_fixture`: Creates canonical test file structures across languages (`testing.T` tables in Go, `@Test` methods in JUnit/Java).
   The central operation registry routes the universal intent to the registered language backend adapter (ADR-0021).
3. **Dynamic MCP Tool Scoping (ADR-0009, ADR-0021)**:
   To prevent attention degradation and schema dilution from an oversized tool catalog, the MCP server must dynamically filter `tools/list` using detected repository languages, exposing only the composite operations supported by the active backend.
4. **Engine Primitives vs. Agent Skills Boundary (RQ-0023)**:
   Composites that depend on third-party frameworks (e.g. Cobra, Spring, Akka) remain strictly in Tier 2 (Agent Skills in `skills/*/SKILL.md`), which steer the LLM to orchestrate clean Tier 1 engine primitives (`semantic_replace_loop`, `semantic_replace_decl`, `semantic_insert_case`) without hardcoding external libraries into the Go compiler/LSP engine.

## Sources

* [IntelliJ IDEA refactoring catalog](https://www.jetbrains.com/help/idea/refactoring-source-code.html)
* [Eclipse JDT refactoring actions](https://help.eclipse.org/latest/topic/org.eclipse.jdt.doc.user/reference/ref-menu-refactor.htm)
* [Visual Studio Code refactoring](https://code.visualstudio.com/docs/editing/refactoring)

## Next Steps

* Use this taxonomy in capability matrices, ADRs, and feature proposals.
* Define exact-node removal counterparts for the structural insertion operations, with revision-aware targeting.
* Specify `safe_delete` checks, defaults, and refusal behavior per removal operation and backend; do not expose a second method family.
* Design universal composite schemas for `semantic_implement_contract` and `semantic_add_member` under the central operation registry.
* Apply RQ-0023 and RQ-0025 boundaries: keep framework-specific patterns in Agent Skills and speculative workspace isolation in the caller/controller.

## Pre-handover audit and ownership

The 2026-09-30 audit used committed main `6c95d1260d8ddab7a98a3832b5faa7401ed81581` **and the uncommitted main checkout**, rather than this older worktree alone. No `PROJECT_BRIEF.md` was present in the scanned repository. Read the current README, ADR/research indexes, ADR-0055, pending ADR-0060, the operation registry and capability matrix before continuing.

Evidence and work already available:

* `cd40dea` implements bounded extract function/method, `3d03ecb` extract variable, and `74c0d39` inline variable. They are ancestors of main; their retained worker checkouts are clean. Do not plan these as absent operations.
* Main's pending `sourcebuf`, `godiagnostics/manager.go`, `adapters/golang/rename_edit.go`, `mcp/buffers.go` and move-planner changes already implement retained sessions, module-wide rename, cross-file and bounded cross-package buffered moves. The owning chat, **Audit benchmark timing metrics** (`01a0ec81-8a87-7830-b961-12e52643e349`), reports full checks and both chi acceptance replays passing. These are pending changes, not extra commits on the audited HEAD.
* **Improve semedit benchmark proof** (`01a0ee97-1694-70a1-8660-447d196858e5`) owns chi preservation fixtures/oracles, a saved older CLI rename patch, and benchmark fixture/smoke work. Its legacy adapter must not replace the shared-session implementation. Public unrelated-file preservation cases are already integrated into pending main.
* Archived **Implement unified construct engine**, **Implement ADR55 handover**, and **Improve semantic insertion results** records were checked. They cover existing structural insertion, schema/receipt work and copy-through provenance, not hidden implementations of the new refactoring families. The archived replay chat plans reconstruction rather than new refactorings.
* The outstanding `50b80b2` cross-package worker commit is not an ancestor by hash, but its `rejectUntypedMoveReferences` and destination-reference validation already appear in main's committed source. Treat equivalent integrated work by source comparison, not commit ancestry alone.
* Existing `semantic_insert_function` inserts a supplied receiver method; `semantic_insert_type` inserts a supplied struct/interface. No separate generic add-method tool is needed. A focused struct-field edit and inferred contract implementation remain missing; `docs/MANUAL_EDITS.md` ME-0041/ME-0047 and literal replay cases provide evidence.
* The remaining worktrees/branch source scans found no registered implementation for signature changes, function inlining, constant/interface/type/delegate extraction, introduced parameters/objects, field encapsulation, type migration or contract generation. Older dirty trees contain prior replacement/configuration/benchmark work.

Before starting a worker, repeat `git status`, HEAD, branch/worktree and relevant registry checks. Outstanding main changes are owned by other chats and were not copied into this documentation branch. Integrate or obtain their reviewed baseline safely first; never reset main, copy a stale adapter over it, or assume a branch created from `6c95d12` includes those pending files. The handovers describe that target baseline explicitly.

## Shared Handover Contract

The authoritative implementation contract is now in [RQ-0060: Shared handover contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract). It preserves the ADR-0055 naming and continuation rules, shared-buffer execution requirements, and acceptance matrix previously recorded here.

## Individual resumable handovers

All 19 tool handovers from RQ-0041–0059 are consolidated into [RQ-0060: Go Refactoring Tool Implementation and Batch Support](RQ-0060-go-refactoring-tool-implementation.md#tool-handover-map). The original files forward to their corresponding sections. Their consolidation does not mark the implementation complete. Use the consolidated map for contracts, dependencies, parallel ownership, tests, and review status.

## Parallel implementation and integration order

Follow [RQ-0060: Parallel ownership and delivery](RQ-0060-go-refactoring-tool-implementation.md#parallel-ownership-and-delivery). Separate Luna workers retain ownership of their code and fixtures; the coordinator owns shared contracts and reviewed integration.

The initial handovers are Go-first. Java pull-up/push-down/superclass operations and other-language refactoring parity need their own backend investigations; framework-specific generation remains in skills.
