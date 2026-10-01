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

These are **candidate implementation requirements**, not new accepted schema or transport decisions. Extend [RQ-0027's foundation investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations); record accepted boundaries through the existing ADR lifecycle.

### Schema and continuation prerequisites

1. ADR-0055 governs every CLI/MCP/batch input and result: one logical concept has one name and value format. `symbol` is the primary target, never the existing owner of a new target. `file` is an exact file selector; `in_function` selects an owner; `discriminator` and `construct_path` select a construct; `destination_file` selects a destination; existing placement names retain their meaning. Candidate new node anchoring uses `before_construct` / `after_construct` consistently for statements/elements; composite append remains `append`. Field-only bounds are `type_start` / `type_end`, with default `type_end`; relative member placement retains `before_symbol` / `after_symbol` and qualified `target_symbol`. Record that vocabulary once before publishing it.
2. Proposed owner selectors are `in_type` for a type and `in_declaration` for a declaration holding a composite element. Use `receiver_kind` for a pointer/value generation mode rather than changing the meaning of construct `kind`. Add matching inspection/ambiguity projections so callers can copy those fields; do not require them to reassemble qualified names. Accept these names once for all new operations, not independently in each worker.
3. Existing primitive `ParameterContract` kinds cannot represent parameter/member mappings. First implement closed recursive object/array schemas, presence-aware CLI JSON decoding and identical MCP/child-batch validation. Reject unknown schema kinds/fields. Missing, null, empty, false and index zero retain distinct meanings.
4. Use the candidate shared descriptor vocabulary in RQ-0027. In particular, a parameter mapping has `source_index`, `parameter` and `argument`; do not introduce parallel positional arrays or opaque JSON strings. Per-call argument customization is a separately typed/validated extension, not text substitution.
5. A live target is `{file, symbol, expected_revision?}`. Secondary live targets use a named typed `continuations` array of the same records. A full continuation record may include the exact construct/owner selectors accepted by its consuming tool. Proposed `expected_revision` has that same name in inspection results and mutation input/results; compare against the relevant current file revision. Numeric construct paths require a matching revision, not silent reuse after an insertion. This does not replace complete semantic analysis read-set validation.
6. Deleted symbols/cases, moved-from files and old expression paths are historical evidence, not live `SourceFields`. After final normalization re-resolve the surviving target. A moved target's live `file` is its actual destination; retain requested source/destination separately in historical operation evidence. An extraction's returned helper must have the exact qualification accepted by downstream rename/body tools. New-name inputs must not list qualified output fields as sources unless their accepted value format is genuinely compatible.
7. Child receipts record the revision they describe. Later children may rename/move/remove the same target; final continuations must track that identity or be omitted. Do not retroactively claim earlier target locations are final. Unsupported/unresolved projections are explicit, not apparently valid guessed names.
8. Extend `SourceFields` validation beyond field existence: identical parameter names, compatible shape/value format and role, live branch availability and source description. Nested continuation paths need an intentional registry extension. Navigation-only line/column remain labeled; no byte-offset continuation or full before/after diagnostic arrays.
9. Apply [RQ-0027's model-facing description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit): every operation handover supplies candidate semantic routing text, named producer→input and live result→consumer guidance. Fresh creation names and caller-authored replacements have no invented copy edges. Validate generated discovery/CLI descriptions with real continuation calls; observable argument flow is the evidence boundary, not a guarantee about internal model reasoning.

### Shared-buffer execution and error contract

* Every operation's initial existing-Go-file scope must be `Batchable` **and** `BufferedBatch`. A hidden operation-owned disk barrier does not satisfy this request. Existing batches contain fixed raw parameters, not automatic receipt references/interpolation. `semantic_lookup` is currently read-only and not batchable: use later mutating children to prove working-revision resolution, an internal read probe to observe unpublished bytes, and a separate lookup after completion. Read-only batch eligibility would require its own explicit registry decision. Future file creation/deletion/package movement is gated on resource support or explicit scaffolding prerequisites.
* Resolve symbols, references, imports, compiler types and LSP snapshots from one current `sourcebuf` overlay. Package loading binds source membership, manifests, auxiliary inputs and local replacements until discovery is stable. Local replacement modules can be read inputs without becoming writable outputs.
* Define the permitted output module separately from analysis scope. Cross-module writes, nested-module scope leakage, symlink aliases and out-of-scope files refuse unless an explicit accepted policy authorizes them. `Buffer.Apply` write containment does not by itself prove semantic scope.
* Build a complete validated output plan, merge it over the full working overlay for required type/semantic checks, then apply all changes through one `Buffer.Apply`. A failed planner/preflight contributes zero working mutations. Compilation is one check, not a proof of evaluation order, control flow, address identity or reference completeness. If an earlier child leaves required semantic inputs incomplete, report the concrete analysis refusal; do not invent a blanket batch ban or require unrelated intermediate source to be clean without a proof need.
* Synchronize gopls with unpublished bytes before dependent queries. Reuse one process per module/MCP instance under pending ADR-0060; ordinary edits must not spawn fresh gopls CLI calls. A file-set/configuration change can legitimately restart a session and must be tested as such.
* Native code actions are candidates. Returned edits, commands, interactive dialogs and server `workspace/applyEdit` have different contracts. The current rename edit path does not establish safe support for all of them. Add explicit capture/validation and protocol evidence before adopting a command-based action; never let it publish independently.
* Defer built-in formatting/imports and baseline/final diagnostics to the batch coordinator; normalize only actual changed sources. Empty writes skip normalization. Preserve configured hook scopes/trust and diagnostic coverage/completeness; a missing or failed report is not clean.
* Refusal stops the batch; later children are not executed. Retain earlier successful children. Report ordered results, attempted versus applied state and final persisted diff. Stage outputs before atomic per-file publication, but report partial I/O publication honestly with no automatic global rollback. Synchronize after success; discard unpublished state and invalidate/reload sessions after failed publication.

### Acceptance matrix for every handover

Use CLI txtar as the observable contract, including the public MCP batch surface. Every operation needs standalone success/no-op/refusal, copied receipt→next-call evidence, default/false import organization, exact untouched-header/sibling bytes, semantic behavior fixtures, stale selectors/preimages and cancellation. Add an all-output preflight failure, earlier unsaved edit→operation→later resolution/edit, refused child retaining prior successes, final persisted diff, and real-gopls one-process/no-intermediate-publication proof. Mock transport tests supplement those proofs. No new backend is advertised without comparable CLI cases.

Run `make check` before focused tests and before completion. This applies to future implementation workers as well as this documentation handoff. Follow repository semantic dogfooding and failure logging. Keep temporary work and parallel worker checkouts under `.scratch/`; do not mutate manifests or invoke paid benchmarks as a side effect of implementation.

## Individual resumable handovers

Every linked investigation starts Open. Each records its own bounded first delivery, unresolved semantic gates, parameter/receipt candidates, implementation paths, batch sequence, executable acceptance cases, and candidate model-facing descriptions with provenance and consumer guidance.

| RQ | Function/family | Audited status | Main dependency |
| :--- | :--- | :--- | :--- |
| [0041](RQ-0041-go-change-signature.md) | Change signature | Missing | Parameter mapping and evaluation proofs |
| [0042](RQ-0042-go-inline-function.md) | Inline function/method | Missing | Call selection and once-only evaluation |
| [0043](RQ-0043-go-extract-constant.md) | Extract constant | Missing | Typed constant/context proof |
| [0044](RQ-0044-go-extract-interface.md) | Extract interface | Missing | Exact method sets and member selection |
| [0045](RQ-0045-go-introduce-parameter.md) | Introduce parameter | Missing | 0041 signature planner |
| [0046](RQ-0046-go-introduce-parameter-object.md) | Introduce parameter object | Missing | 0041 plus field/evaluation mapping |
| [0047](RQ-0047-go-extract-type-and-delegate.md) | Extract type and extract delegate | Missing | Member dependencies and identity/layout proof |
| [0048](RQ-0048-go-encapsulate-field.md) | Encapsulate field | Missing | 0056 field/member planning |
| [0049](RQ-0049-go-type-migration.md) | Type migration | Missing | Type identity graph; later 0041 |
| [0050](RQ-0050-go-implement-contract.md) | Implement interface/contract | Missing | Method-set/signature comparison |
| [0051](RQ-0051-go-extract-function-expansion.md) | Broader extract function/method | Existing bounded feature | Live-in/live-out and control-flow proof |
| [0052](RQ-0052-go-extract-variable-expansion.md) | Broader extract variable | Existing bounded feature | Expression evaluation/context proof |
| [0053](RQ-0053-go-inline-variable-expansion.md) | Broader inline variable | Existing bounded feature | Def/use and evaluation proof |
| [0054](RQ-0054-go-safe-delete-expansion.md) | Broader safe delete | Existing constant-only safety | Complete source/reference/resource coverage |
| [0055](RQ-0055-go-move-construct-expansion.md) | Broader cross-package/type/package moves | Existing buffered free-function moves | Dependency group and resource proof |
| [0056](RQ-0056-go-insert-field.md) | Focused struct-field insertion | Missing | Member collision/layout contract |
| [0057](RQ-0057-go-insert-statement.md) | Anchored statement insertion | Missing | Revision-bound node targeting |
| [0058](RQ-0058-go-insert-composite-element.md) | Composite-element insertion | Missing | Typed literal/element targeting |
| [0059](RQ-0059-go-set-doc-comment.md) | Documentation-comment edit | Missing | Comment ownership/directive preservation |

## Parallel implementation and integration order

1. **Foundation tracks:** extend RQ-0027 for recursive schemas/CLI parity and live revision-bound receipts. Pure analysis/planner prototypes can proceed independently; publish schemas only after one shared decision. Reconcile pending main's buffer/session baseline before source workers branch from it.
2. **First independent operation tracks:** change signature; inline function; constant extraction; interface extraction/contract implementation; broader statement extraction; expression extraction/inlining; safe-delete reference coverage; dependency-aware movement; focused fields/statements/elements/comments. Assign one owner per shared file (notably `wire_extract.go`) and separate pure planner files. Method-set analysis is shared by interface and contract tracks; evaluation/alias analysis is shared by signature and extraction/inline tracks.
3. **Dependent composition tracks:** introduced parameter/object use the signature planner; encapsulation and type/delegate extraction use field/member planning. Type identity normalization can prototype early; general migration follows those proofs. Whole-package/resource movement follows a separate resource-operation gate.
4. **Integration:** workers own their operation source/tests and findings. A coordinator owns registry schema/capability/provenance updates, shared docs/index and sequential reviewed merges. Rebase on the current reviewed integration tip; run required checks after shared changes. Do not run concurrent whole-tree formatter writes across shared working files.
5. **Review gates:** resolve semantic proof gaps with a focused stronger reviewer; do not inflate initial support to avoid a refusal. Complete each bounded advertised scope and its batch/public tests. Update existing relevant ADRs and their index when accepting an extension; preserve separate Open follow-ups.

The initial handovers are Go-first. Java pull-up/push-down/superclass operations and other-language refactoring parity need their own backend investigations; framework-specific generation remains in skills. These are scope qualifications, not claims that hierarchy refactoring is implemented.
