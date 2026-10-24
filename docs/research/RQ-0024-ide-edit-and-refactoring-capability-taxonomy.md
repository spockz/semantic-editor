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
7. Which extraction, move, inline, signature, introduction, and hierarchy patterns are provided by current language servers and language-native engines, and how should one agent-facing vocabulary represent them without promising unsupported behavior?
8. Which failed semantic calls in the latest benchmark were genuine missing refactoring capabilities, and which were schema, selector, symbol-resolution, or result-reporting failures?

## Audited Capability Taxonomy

| IDE Capability Family | Audited `semedit` Mapping | Remaining investigation |
| :--- | :--- | :--- |
| Navigation and symbol lookup | Go inspection, outline and references; bounded other-language lookup | Preserve explicit language/operation limits. |
| Rename | Go module-wide retained-session rename, including shared batch overlays; selected-file Rust/Java rename | Do not reimplement Go rename; other-language workspace scope is separate. |
| Structural insertion/replacement/removal | Go declarations, receiver methods, bodies, constructs and exact declaration/case removal | Focused fields, statements, composite elements and documentation comments remain missing. |
| High-intent composites | Package-inferred file scaffold; supplied method/type insertion | Generate missing interface methods without duplicating method insertion. |
| Extract and introduce | Go AST extract function/variable/constant; Java JDT LS extract function/variable/constant/parameter and parameter object | Go extraction data-flow shapes remain bounded; broader language parity is not established by shared operation names. |
| Inline | Go AST inline variable; Java JDT LS inline variable/function | Go call inlining and broader scope semantics remain a gap. |
| Move and copy | Go declaration move planner; Java JDT LS move refactorings; Rust move/module assists | Cross-language operation coverage differs; model files, members, declarations, modules, and packages separately. Copy is not currently a refactoring capability. |
| Signature and type transformation | Go bounded private-function signature change; Java JDT LS change signature | General caller adaptation, exported API compatibility and type migration remain language-specific. |
| Hierarchy refactoring | Java implement/extract interface and member-move paths; no general cross-language implementation | Preserve language type-system distinctions; Go interface satisfaction is structural, not class inheritance. |
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

## Cross-language capability survey and agent-facing vocabulary (2026-10-19)

This section reconciles upstream documentation with the current operation registry. “Upstream” means the cited server or IDE documents the feature; it does not mean `semedit` exposes it. An unlisted upstream feature is **unknown from the cited feature documentation**, not proof that the server cannot do it. LSP code-action kinds standardize broad labels such as `refactor.extract` and `refactor.inline`, not a portable semantic contract or guaranteed implementation.

### Common vocabulary

Expose intent as capability families, then let each language advertise concrete variants and parameter constraints. Preserve upstream action names in capability metadata and diagnostics, but do not make them the canonical cross-language schema.

| Agent intent | Canonical family and relevant variants | Shared semantic contract |
| :--- | :--- | :--- |
| Extract selected code | `extract_callable` (`function`, `method`); `extract_value` (`variable`, `constant`, `field`); `extract_type` (`interface`, `supertype`, `alias`); `extract_parameter_object` | Selection is an exact expression or statement region; destination/name/visibility are explicit; backend reports inferred inputs, outputs, and refused flow shapes. |
| Introduce a construct | `introduce_variable`, `introduce_field`, `introduce_parameter`, `introduce_parameter_object`, `introduce_delegate` | State scope, selected expression, type/initializer, and call-site or field initialization policy. |
| Move code or files | `move_declaration`, `move_member`, `move_file`, `move_module` | State source and destination scopes; backend owns reference updates and reports visibility, dependency, cycle, and package/module limits. |
| Inline | `inline_variable`, `inline_callable`, `inline_type` | State one-use vs all-use scope, declaration removal policy, and side-effect/evaluation-order preconditions. |
| Change callable/API shape | `change_signature` | Describe parameter order/add/remove/rename/type edits and call-site adaptation; preserve overload, default, and compatibility constraints where available. |
| Change type relationships | `extract_interface`, `implement_contract`, `pull_members_up`, `push_members_down`, `encapsulate_field`, `replace_inheritance_with_delegation` | Advertise only where the language has the relevant construct; report the exact set of declarations/call sites changed. |

This is a taxonomy, not a request to collapse every variant into one oversized operation. The MCP catalog should expose the selected intent with the same schema and behavior whether called directly or as a buffered batch child; each backend advertises only supported variants. A structured selection handle is preferable to guessed line ranges for repeated code-selection workflows.

### IntelliJ IDEA names and documented coverage

The current IntelliJ IDEA catalog explicitly names **Safe Delete**, **Copy/Move**, **Extract Method**, **Extract Constant**, **Extract Field**, **Extract Parameter**, **Introduce Variable**, **Rename**, **Inline**, and **Change Signature** as popular refactorings. Its extended multi-caret list also names **Encapsulate Fields**, **Move Static Members**, **Extract Delegate**, **Extract Interface**, and **Extract Superclass**. These are the exact labels documented on that page, not a guarantee that each is available for every language. The extraction UI specifically reasons about inputs and outputs, while Change Signature handles parameter and return-type changes.

### Language mapping

| Language | Upstream native LSP/server or AST capability (exact documented names) | Current `semedit` exposure | Gaps / reuse direction |
| :--- | :--- | :--- | :--- |
| Go | **gopls** code actions: `refactor.extract.function`, `.method`, `.variable`, `.constant`, `.variable-all`, `.constant-all`, `refactor.extract.toNewFile`; `refactor.inline.call`, `.variable`; `refactor.rewrite.moveParamLeft`, `.moveParamRight`, `.removeUnusedParam`. Go’s standard AST supports local deterministic edits but not semantic workspace updates by itself. | Native `go/ast` extraction/inlining/move and signature-edit implementations; gopls for rename/diagnostics ([Go capability registry](../../internal/backend/golang/backend.go), [signature handler](../../internal/operation/wire_signature.go)). Current extract-function implementation accepts a narrow straight-line shape; no general gopls extraction delegation is established. | Reuse AST/type info and gopls where its actions cover a request. gopls documents no general file/package move or full change-signature operation; parameter reordering/removal are narrower actions. Preserve compiler checks for inferred live-outs and reverse dependencies. |
| Java | **JDT LS** LSP action kinds used by the adapter include `refactor.extract.function`, `.variable`, `.constant`, `refactor.inline`, `refactor.change.signature`, `refactor.move`, and `source.overrideMethods`; JDT LS documents these categories and workspace edits. | JDT LS-backed extract function/variable/constant, inline, change signature, introduce parameter/parameter object, move construct, extract interface and implement contract, plus CST edits ([capability registry](../../internal/backend/java/java.go), [action routing](../../internal/backend/java/codeactions.go)). | Prefer JDT LS refactorings, retaining CST only for deterministic local structural edits. Verify exact command availability against the server version and selection context. IntelliJ IDEA has broad Java support; no evidence here that JDT LS is richer for Java. |
| Rust | **rust-analyzer assists**: `extract_function`, `extract_variable`, `extract_constant`, `extract_static`, `extract_type_alias`, `extract_module`, `move_module_to_file`, `inline_call`, `inline_into_callers`, `inline_local_variable`, `inline_type_alias`, and `inline_type_alias_uses`. | Current adapter exposes lookup, rename and outline; no extraction or move refactoring tool is registered ([Rust adapter](../../internal/backend/rust/backend.go)). | Reuse rust-analyzer assists before adding custom syntax edits; add only capability-tested operations and preserve borrow/type-check feedback. Compare product coverage with RustRover, not IntelliJ IDEA’s non-Rust core. |
| Kotlin | JetBrains’ official Kotlin LSP documentation lists rename, organize imports and formatting among supported LSP features; it does not establish extraction or signature refactoring coverage. IntelliJ IDEA documents standard extraction/introduction families for Kotlin in its IDE product. | Current adapter is read-oriented: lookup, verify and outline; no mutation/refactoring exposure ([Kotlin adapter](../../internal/backend/kotlin/backend.go)). | Treat server extraction as undocumented until verified against the selected Kotlin LSP build. Reuse IntelliJ-backed APIs only if the chosen server provides them; otherwise record a gap rather than infer support from IDEA. |
| Scala | Metals documents **Extract Method** (including expression extraction with unavailable values becoming parameters), `extract-member-definition` (move a class/trait/object/enum to a new file), and `inline-value`. | Current adapter exposes lookup and outline only ([Scala adapter](../../internal/backend/scala/backend.go)). | Reuse Metals actions and retain its single-file/context limits. The file-member action is a narrower “move declaration to new file,” not a general package move or change signature. Scala plugin/IDE coverage is the proper same-language comparator. |
| Haskell | Haskell Language Server documents `Change Type Signature`, `Add argument to function`, GADT conversion, record wildcard expansion and import actions through plugins. No general extract/inline/move catalog is established by the cited feature list. | Current adapter exposes lookup only ([Haskell adapter](../../internal/backend/haskell/backend.go)). | Reuse HLS typed actions when present; do not label a type-signature or add-argument action as general Change Signature without verifying caller adaptation and constraints. Extraction remains an open gap/unknown. |
| Bash | bash-language-server documents rename, ShellCheck quick fixes and formatting; its feature list does not document semantic extraction, move, inline or signature refactorings. | Text LSP adapter exposes selected-file lookup/outline/diagnostics; source mutation is unavailable ([text LSP adapter](../../internal/backend/textlsp/backend.go)). | No general semantic refactoring capability established. Shell formatting and diagnostics are not refactorings. |
| Make | make-ls documents language-server navigation/code actions, but its official feature list does not establish semantic extraction, move, inline or signature refactorings. | Text LSP adapter exposes Make lookup/outline and selected operations; source mutation is unavailable ([text LSP adapter](../../internal/backend/textlsp/backend.go)). | Treat semantic refactoring as a gap until an action is verified against actual make-ls behavior and Make’s include/variable semantics. |

### Where server coverage exceeds the IntelliJ comparison

The evidence supports breadth claims only when scope is stated. gopls and rust-analyzer document exact language-specific assists; Metals documents expression-level Extract Method and moving top-level members into a new file; HLS documents language-specific type-signature and argument actions. Some are not named in the general IntelliJ catalog, but that alone does **not** prove an LSP is more capable than IntelliJ IDEA for the same language. IntelliJ IDEA’s primary comparison is Java/Kotlin; compare Go with GoLand, Rust with RustRover, and Scala with its Scala plugin. Current cited sources do not establish a same-language case where an LSP server has broader semantic guarantees than the corresponding IntelliJ product. The defensible conclusion is that LSPs expose several useful, precisely named language actions and a broader set of language-specific integrations; product superiority remains unproven.

### Latest benchmark failed semantic-call audit

The audit covers the latest complete hard pair (`hard-luna6-skill-off-20261018-r1` and `hard-luna6-skill-on-20261018-r1`) and the latest SWE sample (`swe-reasoning-skill-off-20261018`). Counts are failed-marked semedit calls across arms/repeats, not distinct user intents. The baseline arm is excluded. Repeated retries of the same operation/selector are grouped below; batch wrapper failures are kept separate because their 512-byte recorded result often truncates child detail.

| Cohort | Failed-marked semedit calls | Classification relevant to refactoring interface |
| :--- | ---: | :--- |
| Hard, skill off + on | 192 total: 88 `semantic_batch` wrappers and 104 direct non-batch calls | 51 `semantic_extract_function` failures: 36 unsupported straight-line assignment-to-existing-parameter plus return patterns; 4 named-result safety refusals; 9 invalid statement ranges; 1 missing `kind`; 1 unsafe return-type inference. The first and last expose extraction coverage gaps; named-result refusals are safety boundaries; range/schema errors are not feature gaps. |
| SWE, skill off, `grpc__grpc-go-2735` | 21 failed-marked `semedit-agentic` records, including 4 `applied_with_diagnostics`; 7 explicit-arm batch records had `applied_working` child data | A `target_symbol` unknown field, a missing `source`, invalid construct placement, unsupported positional-literal proof, type insertion cardinality, selector errors, and symbol-not-found are schema/selector/precondition issues. The four `applied_with_diagnostics` records are applied transformations with diagnostics, not rejected refactorings. The seven explicit batch records are wrapper/result-shape evidence, not seven unsupported transformations. |

In the hard skill-enabled cohort, extraction specifically failed **21 of 22 calls**. Sixteen repeated the same `only one leading := declaration with matching named results is supported` refusal; two hit the named-result safety guard; one could not infer a safe return type; and two supplied invalid statement ranges. These are repeated attempts in a single benchmark task, not 21 distinct unsupported patterns. The same refusal recurred without a changed semantic precondition, so result guidance is also deficient.

| Failure category | Mapping to upstream comparisons | Implementable work vs. guidance |
| :--- | :--- | :--- |
| 16 one-leading-`:=` refusals | This is a Go AST subset restriction. gopls documents `refactor.extract.function`; IntelliJ IDEA documents Extract Method with inferred inputs/outputs. Neither source establishes this exact Go assignment shape, so treat both as possible reuse/comparison points, not proof of coverage. | **Capability gap:** support the benchmark’s straight-line writes to an existing parameter plus returned value, deriving live-ins/live-outs and preserving behavior. Implement with Go AST plus type information and compiler/diagnostic checks, or verify whether gopls provides a suitable edit. **Guidance gap:** refuse with the actual supported shape and an alternate tactic; tell the agent not to retry an unchanged selection. |
| 2 named-result refusals | Named results and deferred returns are Go-specific control-flow semantics; Java/JDT LS and IntelliJ Extract Method establish the general family, not an equivalent Go guarantee. No gopls documentation cited here proves this case. | **Safety boundary, possibly implementable:** only broaden after modeling all returns, assignments to named results, and `defer` effects. Until then, clearly explain the unsafe live-out and suggest manual helper construction or a simpler selection; repeated identical attempts should be discouraged. |
| 1 return-type inference refusal | Type inference is shared by IDE/server extraction, but no cited source promises success for this exact nested call expression. | **Capability gap:** use `go/types` or an explicit result type when sound; keep a refusal when the type is not denotable. Guidance should ask for a simpler statement selection or explicit helper construction rather than repeating extraction. |
| 2 invalid statement ranges | IntelliJ derives selection from editor context; LSP refactor actions take document ranges and are context-sensitive. This is request targeting, not a missing transformation algorithm. | **Guidance-only:** expose exact accepted contiguous statement selection and diagnostic reason; offer selection/candidate discovery or a corrected range where possible. Do not widen extraction semantics to compensate for bad coordinates. |

For Go, the practical fallback for a refused extraction is to create the helper with the ordinary declaration insertion operation and replace the selected statements using supported structural edits, then verify. If those primitives cannot express the change safely, the tool should say so and stop; it should not invite repeated extraction retries. IntelliJ’s input/output analysis and gopls/rust-analyzer extraction actions show a useful design precedent for reporting inferred parameters and outputs, but server or IDE parity must be established with a Go fixture before claiming the pattern is supported.

The hard-pair direct failures also included: 13 construct-insertion failures, 4 diagnostics calls with invalid parameter names, 9 symbol inspections, 3 lookups, 2 outlines, 2 removals, 7 renames, 11 construct replacements, one documentation-comment input error, and one undo snapshot-not-found. These do not indicate missing extraction/move/inline capabilities. The direct failure examples include `new_name`/`target_symbol`/`workspace_root` or `selector` parameter mismatches, invalid line ranges, and unresolved symbol names. Batch calls accept the same child operation registry, so schema corrections and selector guidance belong in shared operation contracts rather than a batch-only compatibility layer.

The request-identity benchmark task (`data/benchmarks/results/hard-luna6-skill-on-20261018-r1/task-12-request-identity-migration/`, for example [job-3f72f91bea455f6c.json](../../data/benchmarks/results/hard-luna6-skill-on-20261018-r1/jobs/job-3f72f91bea455f6c.json)) asks to extract `kind = strings.TrimSpace(kind); kind = strings.ToLower(kind); return kind` from `Normalize` into a helper. Current Go extraction refuses assignments to an existing parameter because the implementation’s safe subset expects one leading short declaration or supported direct-call shape. This is a genuine `extract_callable` gap: support straight-line existing-variable writes and derive live-in/live-out behavior, while retaining fail-closed handling for branches, named result semantics, `defer`, evaluation order, and inferred result types. A concrete invalid line-range selection remains a targeting problem; a semantic candidate selection handle could improve it without broadening the transformer unsafely.

Batch result strings in the benchmark are frequently truncated at 512 bytes. Therefore the 88 hard-pair batch wrappers and the 7 explicit SWE wrappers must not be counted as 95 failed refactoring children. The recorded examples `unknown parameter "target_symbol" for "semantic_replace_decl"`, missing `source` for `semantic_replace_construct`, `semantic_batch requires non-empty 'edits' list`, and batch outputs beginning `applied_with_diagnostics` or `applied_working` demonstrate a mixture of request-schema, empty-plan, applied-with-diagnostics, and continuation/result-envelope outcomes. A future benchmark should preserve complete batch child status/error fields before attributing unsupported refactorings.

## Sources

* [IntelliJ IDEA refactoring catalog](https://www.jetbrains.com/help/idea/refactoring-source-code.html)
* [IntelliJ Extract Method](https://www.jetbrains.com/help/idea/extract-method-dialog.html)
* [IntelliJ Change Signature](https://www.jetbrains.com/help/idea/change-signature.html)
* [IntelliJ Move refactorings](https://www.jetbrains.com/help/idea/move-refactorings.html)
* [LSP 3.18 Code Action specification](https://github.com/microsoft/language-server-protocol/blob/gh-pages/_specifications/lsp/3.18/language/codeAction.md)
* [gopls transformation features](https://go.dev/gopls/features/transformation)
* [rust-analyzer assists](https://rust-analyzer.github.io/book/assists.html)
* [Eclipse JDT refactoring actions](https://help.eclipse.org/latest/topic/org.eclipse.jdt.doc.user/reference/ref-menu-refactor.htm)
* [Eclipse JDT LS changelog](https://github.com/eclipse-jdtls/eclipse.jdt.ls/blob/main/CHANGELOG.md)
* [Kotlin Language Server documentation](https://kotlinlang.org/docs/kotlin-lsp.html)
* [Metals Extract Method](https://scalameta.org/metals/blog/2022/10/06/aluminium/)
* [Metals editor integrations](https://scalameta.org/metals/docs/integrations/new-editor/)
* [Haskell Language Server features](https://haskell-language-server.readthedocs.io/en/latest/features.html)
* [bash-language-server](https://github.com/bash-lsp/bash-language-server)
* [make-ls](https://github.com/owenrumney/make-ls)
* [Visual Studio Code refactoring](https://code.visualstudio.com/docs/editing/refactoring)

## Next Steps

* Use this taxonomy in capability matrices, ADRs, and feature proposals.
* Refer to the Luna implementation handover document: [LUNA_HANDOVER_REFACTORING_TAXONOMY.md](../handovers/LUNA_HANDOVER_REFACTORING_TAXONOMY.md) for the shared refactoring intent vocabulary and `refactor_*` tool family architecture.
* Define exact-node removal counterparts for the structural insertion operations, with revision-aware targeting.
* Specify `safe_delete` checks, defaults, and refusal behavior per removal operation and backend; do not expose a second method family.
* Design universal composite schemas for `semantic_implement_contract` and `semantic_add_member` under the central operation registry.
* Apply RQ-0023 and RQ-0025 boundaries: keep framework-specific patterns in Agent Skills and speculative workspace isolation in the caller/controller.
* Validate each server action against pinned server versions using executable fixtures; mark capabilities as confirmed, unknown, or unsupported in the backend registry.
* Decide a stable extraction selection/result contract that can express statement ranges, derived inputs/outputs, and explicit safety refusals across Go, Java, Rust, and Scala.
* Add end-to-end tests for Go extraction of existing-parameter assignments and inferred expression results before extending the accepted subset.
* Fix batch benchmark telemetry to retain full child outcomes; do not infer refactoring coverage from truncated wrapper text.

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
