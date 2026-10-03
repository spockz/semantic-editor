<!-- Keeps the refactoring implementation handovers, shared batch contract, and review status in one resumable research record. -->
# RQ-0060: Go Refactoring Tool Implementation and Batch Support

* **Status**: Open
* **Category**: Go Refactoring & Implementation Handover
* **Date**: 2026-10-01

## Consolidation and scope

This is the single implementation investigation for the tools and expansions previously planned in RQ-0041–0059. Their complete tool-specific contracts, semantic risks, implementation steps, fixture sketches, and model-facing description drafts are retained below. Repeated shared requirements are recorded once. The earlier files are forwarding records; their consolidation status does not mean their tools are implemented.

[RQ-0024](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md) continues to own the broader IDE capability taxonomy. [RQ-0027](RQ-0027-preventing-read-files.md#refactoring-handover-foundations) continues to investigate reusable schema, receipt, and model-facing description foundations. ADR-0055 governs parameter and continuation names. This record collects implementation work; accepted architectural changes still require the applicable ADR and an atomic index update.

## Current implementation baseline

The session and buffer prerequisites are committed on main as `32e21842591749963cf7d9d1b6ae55c3ed88693d`. Use that source baseline and subsequently reviewed commits. The older `6c95d12` audits in the tool sections are historical evidence, not instructions to branch from that revision or overwrite the retained-session implementation. Preserve newer benchmark changes in the main checkout; their owning chat remains responsible for committing them.

The reviewed shared schema/output foundation is delivered on main at `c74dede`, including recursive input/output schemas, inspection continuations, current-overlay revision guards, and corrected partial-publication receipts. Its combined `make check` passed. References in the original candidate paths to missing primitive-only schemas describe the initial audit, not a reason to stop current workers. Do not merge the private runtime snapshot `42446608c684408e4742db239b8df3d2eca758fb`.

At the 2026-10-02 integration checkpoint, main is at `6718968`. It includes the foundation, statement/documentation-comment tools, interface extraction, contract implementation, function inlining, alias-preserving type migration, field insertion, and corrected batched move destination receipts. The combined `make check` passed with bounded package concurrency after resolving shared test-helper and schema-lint merge defects. Working-tree edits and pure planner checkpoints are implementation evidence, not completed public features. Recheck branch HEAD, pending edits, registered capabilities, and actual CLI/MCP behavior before updating this status. The benchmark owner's pending work was restored after each merge; the latest merge verified all 17 nonoverlapping pending files byte-identical and preserved both overlapping documentation additions.

## Parallel ownership and delivery

| Track | Tool sections | Current checkpoint / outstanding work |
| :--- | :--- | :--- |
| Shared foundation | Recursive schemas, native receipts, inspection copy producers | Reviewed through `2a2bcaa`, merged on main at `c74dede`; full integration check passed. |
| Contracts | Extract interface; implement contract | Reviewed `15609dd` delivered on main at `6718968`: live aliases, same-file revision evidence, guarded no-op receipts, active test-variant preflight, test-symlink refusal, and public producer/consumer tests. |
| Extraction | Extract constant; broader function/variable extraction and variable inlining | Public checkpoint through `d15b5ce` remains in review; transitive compiler-input capture and revalidation are outstanding. |
| Structural | Insert field, statement, composite element; set doc comment; encapsulate field | Statement/comment tools delivered at `930893f`; field insertion through `8b34158`, including empty structs, delivered at `6718968`. Composite-element insertion and encapsulation remain in progress. |
| Signatures | Change signature; introduce parameter; parameter object | Public change-signature implementation is in progress, including parameter binding preservation and buffered CLI proof. The other two tools remain outstanding. |
| Relocation | Inline function; type migration; safe delete and move expansions | Reviewed `1f16f65` and move receipt fix `9f2cd21` delivered at `6718968`. Inlining supports one zero-argument literal-return call; migration preserves exact type identity through aliases. Safe-delete expansion remains in progress. |
| Type extraction | Extract type/delegate | Private same-package struct-field extraction is in progress in `flow-type-extraction`; this bounded checkpoint precedes delegate extraction. |

Luna high workers use separate `.scratch/worktrees/flow-*` checkouts. The coordinator reviews each concrete public checkpoint, sends actionable findings to the same worker, reviews its fixes, and merges approved work sequentially. A track may deliver one bounded tool while other sections remain open. Do not wait for an entire feature family before providing a reviewable public commit. Each section must finish its advertised scope, including batches; an internal helper is not public delivery.

Worker branches may advance while this document is edited. The coordinator's local `.scratch/parallel-tools/coordinator-state.json` records exact worktree, CLI session, review-note, and prompt locations; those local paths supplement the durable plans below. Re-read current state when resuming rather than treating this table as a live monitor.

## Integration order

1. **Foundation tracks:** extend RQ-0027 for recursive schemas/CLI parity and live revision-bound receipts. Pure analysis/planner prototypes can proceed independently; publish schemas only after one shared decision. Use the committed main buffer/session baseline recorded above before source workers branch from it.
2. **First independent operation tracks:** change signature; inline function; constant extraction; interface extraction/contract implementation; broader statement extraction; expression extraction/inlining; safe-delete reference coverage; dependency-aware movement; focused fields/statements/elements/comments. Assign one owner per shared file (notably `wire_extract.go`) and separate pure planner files. Method-set analysis is shared by interface and contract tracks; evaluation/alias analysis is shared by signature and extraction/inline tracks.
3. **Dependent composition tracks:** introduced parameter/object use the signature planner; encapsulation and type/delegate extraction use field/member planning. Type identity normalization can prototype early; general migration follows those proofs. Whole-package/resource movement follows a separate resource-operation gate.
4. **Integration:** workers own their operation source/tests and findings. A coordinator owns registry schema/capability/provenance updates, shared docs/index and sequential reviewed merges. Rebase on the current reviewed integration tip; run required checks after shared changes. Do not run concurrent whole-tree formatter writes across shared working files.
5. **Review gates:** resolve semantic proof gaps with a focused stronger reviewer; do not inflate initial support to avoid a refusal. Complete each bounded advertised scope and its batch/public tests. Update existing relevant ADRs and their index when accepting an extension; preserve separate Open follow-ups.

## Tool handover map

| Section | Original record | Main dependency |
| :--- | :--- | :--- |
| [Change signature](#go-change-signature) | RQ-0041 | Parameter mapping and evaluation proofs |
| [Inline function/method](#go-inline-function) | RQ-0042 | Call selection and once-only evaluation |
| [Extract constant](#go-extract-constant) | RQ-0043 | Typed constant/context proof |
| [Extract interface](#go-extract-interface) | RQ-0044 | Exact method sets and member selection |
| [Introduce parameter](#go-introduce-parameter) | RQ-0045 | 0041 signature planner |
| [Introduce parameter object](#go-introduce-parameter-object) | RQ-0046 | 0041 plus field/evaluation mapping |
| [Extract type and extract delegate](#go-extract-type-and-delegate) | RQ-0047 | Member dependencies and identity/layout proof |
| [Encapsulate field](#go-encapsulate-field) | RQ-0048 | 0056 field/member planning |
| [Type migration](#go-type-migration) | RQ-0049 | Type identity graph; later 0041 |
| [Implement interface/contract](#go-implement-contract) | RQ-0050 | Method-set/signature comparison |
| [Broader extract function/method](#go-extract-function-expansion) | RQ-0051 | Live-in/live-out and control-flow proof |
| [Broader extract variable](#go-extract-variable-expansion) | RQ-0052 | Expression evaluation/context proof |
| [Broader inline variable](#go-inline-variable-expansion) | RQ-0053 | Def/use and evaluation proof |
| [Broader safe delete](#go-safe-delete-expansion) | RQ-0054 | Complete source/reference/resource coverage |
| [Broader cross-package/type/package moves](#go-move-construct-expansion) | RQ-0055 | Dependency group and resource proof |
| [Focused struct-field insertion](#go-insert-field) | RQ-0056 | Member collision/layout contract |
| [Anchored statement insertion](#go-insert-statement) | RQ-0057 | Revision-bound node targeting |
| [Composite-element insertion](#go-insert-composite-element) | RQ-0058 | Typed literal/element targeting |
| [Documentation-comment edit](#go-set-doc-comment) | RQ-0059 | Comment ownership/directive preservation |

## Shared handover contract

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
* Synchronize gopls with unpublished bytes before dependent queries. Reuse one process per module/MCP instance under ADR-0060; ordinary edits must not spawn fresh gopls CLI calls. A file-set/configuration change can legitimately restart a session and must be tested as such.
* Native code actions are candidates. Returned edits, commands, interactive dialogs and server `workspace/applyEdit` have different contracts. The current rename edit path does not establish safe support for all of them. Add explicit capture/validation and protocol evidence before adopting a command-based action; never let it publish independently.
* Defer built-in formatting/imports and baseline/final diagnostics to the batch coordinator; normalize only actual changed sources. Empty writes skip normalization. Preserve configured hook scopes/trust and diagnostic coverage/completeness; a missing or failed report is not clean.
* Refusal stops the batch; later children are not executed. Retain earlier successful children. Report ordered results, attempted versus applied state and final persisted diff. Stage outputs before atomic per-file publication, but report partial I/O publication honestly with no automatic global rollback. Synchronize after success; discard unpublished state and invalidate/reload sessions after failed publication.

### Acceptance matrix for every handover

Use CLI txtar as the observable contract, including the public MCP batch surface. Every operation needs standalone success/no-op/refusal, copied receipt→next-call evidence, default/false import organization, exact untouched-header/sibling bytes, semantic behavior fixtures, stale selectors/preimages and cancellation. Add an all-output preflight failure, earlier unsaved edit→operation→later resolution/edit, refused child retaining prior successes, final persisted diff, and real-gopls one-process/no-intermediate-publication proof. Mock transport tests supplement those proofs. No new backend is advertised without comparable CLI cases.

Run `make check` before focused tests and before completion. This applies to future implementation workers as well as this documentation handoff. Follow repository semantic dogfooding and failure logging. Keep temporary work and parallel worker checkouts under `.scratch/`; do not mutate manifests or invoke paid benchmarks as a side effect of implementation.

### Additional execution and receipt requirements

All existing Go edit operations must be `Batchable` and `BufferedBatch`. Read and write through `internal/sourcebuf`; each operation observes the current overlay, not disk-only contents. A refusal must not write its own target or publish an intermediate source revision. Earlier successful mutations remain eligible for normal batch finalization and publication after the batch stops. Each operation validates all of its outputs together with `Buffer.Apply`; batch finalization stages the completed outputs and publishes them through `Buffer.Flush`, atomically per file. Per-file atomicity does not promise global rollback after publication starts. No operation may write an intermediate revision or introduce a batch barrier.

Keep the retained `gopls` process and shared MCP session. Buffered execution defers source normalization until the full changed-source set is ready and normalizes only changed sources. Diagnostic checks retain the configured project/module scope, selected coverage, and completeness; they are not narrowed to changed files. Use only existing `diagnostic_status` values: `unchanged`, `changed`, `unavailable`, or `deferred`. Do not invent a diagnostic-delta JSON shape in this handover; implementation must use the registered `DiagnosticDelta` output schema. Preserve existing diagnostic deltas and changed-file receipts. A batch stops at its first refusal; prior successful overlay mutations remain eligible for normal finalization/publication, while later edits require a separate invocation. Batch entries are a static ordered list with no dynamic receipt references or interpolation. `semantic_lookup` is not batchable and must run outside the batch, after it, when fresh lookup is needed. A no-op must not claim a source change.

The current `sourcebuf.Buffer.Apply` accepts existing files. New-file creation, package moves, and resource operations require a separate source-buffer resource design or an explicit successful scaffold prerequisite; do not smuggle them through direct disk writes. Safe operations must prove against the overlay and revalidate every source they read before apply. Failure to enumerate or revalidate the source set is a refusal, never a silent downgrade. For operations scoped inside a named Go type, use `in_type` consistently as the owner selector; when a new member is the target, `symbol` identifies it as `Type.Member`.

Any `expected_revision` proposed below is an opaque input/output field with that exact same name; copy its live output value verbatim to bind fragile selectors against the current overlay. Its encoding is a decision gate, not a guessed format.

Use ADR-0055 roles: `symbol` is the primary target; `file` is the workspace-relative Go file; `in_function` is an owner selector; `discriminator` is exact source text. `construct_path` is zero-based and may be used or returned only with the matching `expected_revision` for that source revision. Receipts return selectors only while they address live source. Removed names, old expressions, and stale paths belong under evidence fields and are not continuations. Add `SourceFields` to primitive input parameters and text that identifies the exact receipt field. Nested edits and lists require reviewed recursive schema support before public advertisement; the current worker checkpoint is described in the implementation baseline above. Go only until another backend has comparable behavior and CLI txtar coverage.

## Ordered research and implementation steps

1. Audit the current operation registry, engine, backend capability metadata, MCP buffer adapter, and `sourcebuf` on the rebased main. Record any contract drift before proposing code.
2. Decide the unresolved selection, type, comment, resource, and refusal rules below. Update this consolidated record through the coordinator, then prepare an ADR if the design changes a public invariant.
3. Implement engine behavior behind a typed operation contract; use semantic navigation/insertion tools where they support the target. Log any failed semantic tool call in `docs/SUBOPTIMAL_TOOLS.md` before manual repair.
4. Register CLI and MCP schemas, descriptions, `SourceFields`, result formatting, Go capability metadata, and current documentation. Do not advertise other languages.
5. Add the literal CLI txtar cases below, including final file trees, receipts/diagnostics, unchanged refusal state, and the stated mixed batch order. Run `make check` before focused verification and before completion.
6. Mark the implemented tool section complete with findings and accepted ADR links after review; keep this RQ Open while other sections or their research gates remain outstanding.

## Cross-cutting acceptance gates

* Every public success and refusal has CLI txtar coverage; an MCP batch case additionally proves overlay composition. Record each concrete archive path and observable contract in `docs/MANUAL_EDITS.md`; keep literal commands and before/after trees in the txtar fixture itself.
* Refusal tests assert observable diagnostics and unchanged target bytes; batch tests also assert earlier successes remain in the final workspace.
* Tests exercise observable CLI/MCP behavior, not private helper structure or shallow fake LSPs. For each successful operation, include a separate follow-up mutation using live receipt fields, then a standalone `semantic_lookup` or internal probe after that mutation to prove the target is fresh; never include lookup as a batch child. For removal, use a still-live owner or sibling target because the deleted symbol is historical evidence.
* Add exact schema and capability assertions when a new operation is introduced. Preserve matching CLI and MCP names, types, defaults, and value formats.
* No global rollback claim, speculative preview engine, framework-specific behavior, or non-Go capability is implied.

## Related contract records

* [RQ-0024 shared handover contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract) (parent-owned shared rules and index entry)
* ADR-0055, `docs/adr/0055-tool-parameter-schema-harmonization.md`
* ADR-0010 atomic disk update invariant; ADR-0032 direct batch execution; ADR-0042 structured MCP output schemas
* `docs/MANUAL_EDITS.md` records prior manual edits and capability gaps; preserve literal txtar fixtures.

## Go change signature

Original handover: RQ-0041. Implementation status: Open.

### Purpose and shared contract

Plan a Go-first `semantic_change_signature` operation that changes a function or method declaration and its supported call sites from one resolved signature to another. This is an implementation handover, not an accepted design. Apply the shared invariants in [RQ-0024 § Shared Handover Contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract) and report unresolved decision gates there.

Sibling Go investigations: [inline function](RQ-0060-go-refactoring-tool-implementation.md#go-inline-function), [extract constant](RQ-0060-go-refactoring-tool-implementation.md#go-extract-constant), [extract interface](RQ-0060-go-refactoring-tool-implementation.md#go-extract-interface), [introduce parameter](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter), and [introduce parameter object](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter-object).

### Observable outcome and boundaries

A successful operation reports the changed file set, final diff, diagnostics, and a `continuations` entry with reusable `file`, surviving `symbol`, and `expected_revision`. Callers can resolve that symbol again and perform a later semantic edit. Keep the original function or method alive; never return a deleted identifier as a live target.

First release: one target function or method declaration in an existing Go file, with call-site rewrites allowed across multiple existing files in the same package within one module. Preserve result arity, types, parameter order, and existing parameter names exactly; append one new parameter only when every call site maps safely. Refuse unresolved references, unsafe method-set changes, unproven build variants, exported API migration, variadic/generic transformations, and ambiguous function-value adaptation. Prove the complete call set before publication. Result changes and caller/body result mappings are later work. No framework migration, compatibility shim, speculative preview API, or new-file creation is in scope.

### Audited starting state

The authoritative audit tree is `/Users/alessandro/sources/semantic-editor` at `6c95d12`, including shared uncommitted buffers; it was read-only. Six requested operations were absent in all worktrees. Bounded extraction, inline, move, and removal work already exists. The source tree has a central typed registry, Go AST edits, one-process gopls sessions, and a new `sourcebuf` overlay path. There is no Go change-signature AST operation or registry entry. Current `gopls` client support does not establish command/interactive/applyEdit support. A pending ADR-0060 in the audited tree describes MCP-owned diagnostic sessions; treat it as context, not as a settled refactoring transport contract.

### Candidate implementation paths

* Public operation: `internal/operation/wire_engine_mutations.go` or a narrowly scoped sibling wire file; mirror `wire_extract.go` and `wire_engine_inline.go` for typed parsing, registry definition, MCP/CLI naming, formatting, and receipts.
* AST and semantic analysis: `internal/astedit/function.go`, `decl.go`, `extract.go`, and `internal/symbol/resolver.go`; use `internal/adapters/golang/gopls.go` and `internal/lsp/client.go` for reference/type evidence.
* Buffering/publication: `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, and `internal/mcp/batch.go`. Every result file must be staged as a complete set of `sourcebuf.Change` values and committed to the overlay with one `Buffer.Apply` only after all preimages validate.
* Wire schema: `internal/operation/operation.go`, `parse.go`, and MCP schema generation. Depend on RQ-0024's closed recursive object/array schema and presence-aware CLI JSON support before exposing structured parameter/result descriptors. Do not encode structures in opaque strings.

### Proposed public contract

Names are proposals; settle only after the transport and batch gates. `semantic_change_signature` / CLI `change-signature`; Go backend only initially. Target roles follow ADR-0055: `symbol` names the declaration changed, while `in_function` is only a scope selector where needed. Every source field must validate against the referenced output field by both name and type, and parameter descriptions must identify those sources.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target | string | none | Existing `.go` file path, copied verbatim from a file continuation. |
| `symbol` | target | string | none | Fully qualified function or method; copied from lookup/receipt. |
| `parameters` | replacement | array of closed `ParameterMapping` objects | none | Ordered list determines final signature order. Each entry is `{source_index?: integer, parameter: ParameterSpec{name: string, type: string, variadic?: boolean}, argument?: Goexpr}`. Existing parameters map by zero-based `source_index`; release one keeps them in their original order/name/type. Append one new parameter without `source_index` and with `argument`; reorder/rename mappings are later work. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token from lookup or preceding mutation receipt; binds the selected declaration file only. Capture and validate the complete caller/reference set separately. |

Do not accept unparsed declaration fragments. Release one preserves result count and types; result rename or shape changes require a later explicit result-mapping design. Preserve old parameter bindings by `source_index`, and prove that each new `argument` is available with the same resolved meaning at every affected call. Initially allow only side-effect-free closed expressions: literals/constants or package-level symbols resolving to the same object at every call. Function calls, reads of caller-local names, and mapping extensions are gated. Return the surviving declaration through a `continuations` entry with `file`, `symbol`, and `expected_revision`.

### Decision gates and risks

1. Establish whether gopls exposes a suitable code action and whether the response is a plain `WorkspaceEdit`. If it requires `workspace/executeCommand`, interactive selection, or client-side `workspace/applyEdit`, stop adoption until the transport gate adds and tests that support. Do not claim the current session can apply it.
2. Determine a complete call/reference set across packages, tests, build tags, generated files, method expressions, interfaces, function values, and external API boundaries. First release refuses any call set it cannot prove complete.
3. Decide how named/unnamed result equivalence, variadic parameters, generic signatures, receiver types, and package aliases normalize. Reject rather than guess.
4. Precompute every edited declaration/call file, import delta, and verification input. A late invalid call must leave all overlay bytes unchanged.
5. Match `expected_revision` to the shared RQ-0024 receipt lifecycle. Resolve output continuations after normalization; never emit stale selector paths as live continuations.

### Ordered implementation handover

1. Inspect current function/method resolution, references, batch segmentation, and diagnostic-session ownership; record exact gopls evidence and transport limitations.
2. Prototype a read-only resolver over sourcebuf overlays. Enumerate all direct call sites and unsupported reference forms; prove ambiguity and build-variant refusal.
3. Define a closed typed signature patch schema and CLI encoding with presence semantics. Add registry shape checks for `SourceFields` name and type and tool descriptions with source-field provenance.
4. Implement a pure planner that returns all file preimages and proposed bytes. Preserve comments, unrelated bytes, and declaration layout; use AST/token edits rather than pretty-printing whole files.
5. Validate every output, syntax/type information, collision, import update, and revision before one `Buffer.Apply`. For direct calls, publish each final file atomically; document partial multi-file publication failure honestly.
6. Mark `Batchable: true` and `BufferedBatch: true` for existing Go files. Use the shared overlay for both AST and LSP reads, defer verification/import normalization, then verify and organize once at the batch segment boundary. Earlier successful edits remain after a later edit fails; do not promise whole-batch rollback.
7. Add registry CLI/MCP support, result formatting, capabilities, and txtar coverage; keep new-file creation behind an explicit `semantic_scaffold_file` prerequisite until sourcebuf supports new resources.

### Required evidence and tests

Add public CLI txtar cases that invoke the registered CLI command, assert stdout/diagnostics and exact workspace state, and use the MCP `semantic_batch` surface for batch cases.

* Positive: change a private function signature with direct calls in two existing files; verify declaration and both calls, unchanged result types/arity, successful diagnostics, and a `continuations` entry usable verbatim by lookup and another edit.
* Refusal: unresolved function value, interface implementation break, duplicate parameter name, invalid type, stale `expected_revision`, and one unsupported/build-tag call site. For each, assert byte-identical files and no diff.
* All-output preflight: arrange two caller files where the second conflicts or cannot be transformed; prove the first file and target declaration were not applied.
* Buffered sequence: earlier unsaved `semantic_replace_body` → signature change → a later mutation using the normal resolver, then a standalone lookup/edit using the returned continuation. Assert later steps see the overlay, disk remains unchanged until final flush, normalization/verification run once, and all published files contain only final bytes.
* Retained server: instrument a batch to prove the resolver probe and each edit use the same one-process gopls session over the overlay; do not launch `gopls` CLI as a substitute.
* Byte and no-op: retain comments, unrelated whitespace, line endings, and imports outside planned edits. A replacement signature identical to the current signature must return an empty diff and perform no write.

### Completion decision

Resolve only after the transport gate, call-set completeness proof, typed schema gate, and required txtar/buffered evidence pass. If safe argument migration cannot be established across existing module files, publish the precise unsupported cases and revise the first-release boundary before implementation expands.

### Concrete fixture sketches

Create `testdata/scripts/change_signature_go.txtar` as a public CLI scenario and `testdata/scripts/mcp_batch_change_signature.txtar` for buffered behavior. Include ordinary package source, two direct call sites, a test call site, and a deliberately byte-irregular unrelated Go file.

The positive batch should follow this order. `expected_revision` is omitted for a fresh overlay target; the operation itself resolves all call sites from the overlay. This JSON is a request-shape sketch, not a receipt-interpolation example:

```json
{
  "edits": [
    {
      "tool": "semantic_replace_body",
      "params": {
        "file": "uses.go",
        "symbol": "handle",
        "body": "return process(1)"
      }
    },
    {
      "tool": "semantic_change_signature",
      "params": {
        "file": "service.go",
        "symbol": "process",
        "parameters": [
          {
            "source_index": 0,
            "parameter": {
              "name": "id",
              "type": "int"
            }
          },
          {
            "parameter": {
              "name": "ctx",
              "type": "context.Context"
            },
            "argument": "nil"
          }
        ]
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "uses.go",
        "source": "func Verify() error { _ = process(1, nil); return nil }"
      }
    }
  ]
}
```

Make the fixture's first edit introduce a valid `process(1)` call in `uses.go`, a second existing module file; the final signature appends `ctx` after `id`, preserving existing evaluation order. The new `ctx` argument uses `nil`, a side-effect-free closed expression assignable to `context.Context` at every call site. Assert the later mutation resolves the returned signature and succeeds; a standalone follow-up may consume the emitted continuation. If schema cannot express a valid argument migration, that is a blocking contract finding, not a reason to weaken the proof.

The standalone positive txtar must call `semedit change-signature ...` and compare every changed caller file. Add a second MCP txtar whose final caller has a type error; assert `isError`, original target/caller files byte-for-byte, and no partial diff. Add stale-revision and unsupported function-value fixtures, each with explicit diagnostic assertions.

A focused MCP buffer test must inspect disk after the operation handler returns but before batch flush. A separate stale-preimage case must change a captured input on disk before `Buffer.Apply` or `Flush`, assert `sourcebuf.ErrConflict`, and prove no planned buffered output was published: every output must still equal original disk bytes while sourcebuf reads show all planned edits. It must also assert one gopls process/session ID from the earlier edit through the later mutation, one verification/import pass, and preservation of earlier successful overlay edits after a subsequent batch failure. Per-file final publication may report a partial count if an OS write fails; do not assert whole-batch rollback.

### Expansion milestone and artifact checklist

Milestone two may map caller-local arguments only with per-call binding proofs; milestone three may alter results or handle exported APIs only with explicit caller/body mapping and compatibility contracts. Each expansion records supported and refused Go syntax in capability metadata and adds comparable CLI txtar fixtures. The handover is complete when a reviewer can run the fixture, copy a result continuation verbatim, and predict every file in the final diff.

### Expected research artifacts

Record a signature-to-call-site matrix, one explicit supported/refused syntax table, and the gopls transport trace that justifies the selected implementation route. Keep those findings with the implementation proposal so future language backends can compare capability without inheriting Go-specific assumptions.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_change_signature`:**

> Use this tool instead of editing signatures and callers separately to append one parameter to a supported private Go function. Supply the complete ordered `parameters` mapping; existing parameters retain their order, names, and types. The engine updates every proven direct caller and refuses incomplete or unsupported use sets. Pass each surviving `continuations` entry's `file` and `symbol` verbatim to `semantic_rename` or `semantic_replace_body`; keep its `expected_revision` with that target when accepted by the consuming tool.

**Parameter description requirements:**

`symbol`: Target: qualified existing callable. Copy verbatim from `semantic_lookup.symbol` or `semantic_rename.symbol`, with the corresponding `file`. `parameters`: Replacement: caller-authored final mapping; `source_index` is the zero-based original parameter position, not a navigation line or symbol. Never infer a default caller argument from a nearby result.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go inline function

Original handover: RQ-0042. Implementation status: Open.

### Purpose and shared contract

Plan a Go-first `semantic_inline_function` operation that replaces a selected call with a safe equivalent body expansion and removes the function declaration only when the requested scope proves it unused. This is a handover plan, not an accepted decision. Follow [RQ-0024 § Shared Handover Contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract).

Sibling investigations: [change signature](RQ-0060-go-refactoring-tool-implementation.md#go-change-signature), [extract constant](RQ-0060-go-refactoring-tool-implementation.md#go-extract-constant), [extract interface](RQ-0060-go-refactoring-tool-implementation.md#go-extract-interface), [introduce parameter](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter), and [introduce parameter object](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter-object).

### Observable outcome and boundaries

Success replaces exactly one selected direct call derived from a resolved callee body and preserves evaluation/order. The primary `file` + `symbol` target pair always identifies the callee declaration while it remains live. Return the edited caller/owner as a `continuations` array entry `{file, symbol, expected_revision}`. If deletion is explicitly requested and proven safe, omit the deleted callee from live continuations; record its identity only in separate historical data. Never substitute the owner name into the callee `symbol` field.

Bounded first release: one target non-recursive, non-generic, non-variadic free function called directly once in an existing Go file; caller and callee may be in different existing files within the same package; function body is a single return expression or a small straight-line statement list with a final return; no escaping/address-taking, defer/recover, closures, labels, control flow, named-result dependence, or observable ordering hazard. Preserve the callee by default. Declaration deletion is a separately gated option, default false, and only when a same-package complete reference proof succeeds. No multi-call workspace inlining, method receiver rewrites, framework patterns, or new-file creation.

### Audited starting state

Read-only audit used `/Users/alessandro/sources/semantic-editor` at `6c95d12`, including shared uncommitted state. The six requested operations were absent across worktrees. Existing `internal/astedit/inline.go` and `wire_engine_inline.go` implement inline-variable only; they are not evidence of function inlining. Existing extraction, move, and removal are bounded. A central registry and buffered Go batch path exist in the source tree. Current LSP client does not establish support for code-action command execution, interactive actions, or `workspace/applyEdit`; an ADR-0060 draft is present in that audited tree and remains pending context.

### Candidate implementation paths

* Public wire: add a distinct operation definition alongside `wire_engine_inline.go` or a dedicated `wire_inline_function.go`; pattern the typed request/receipt after `wire_extract.go` and existing inline-variable operation.
* AST planning: `internal/astedit/inline.go`, `extract.go`, `body.go`, `construct.go`, `symbol/resolver.go`; use `gopls.go` for typed call/reference evidence. Keep function-specific semantics separate from inline-variable semantics.
* Buffer: `internal/sourcebuf/buffer.go` plus `internal/mcp/buffers.go` and `batch.go`; calculate all file edits against one overlay revision and call `Buffer.Apply` only after all output preimages validate.
* Schema: registry and MCP schema generation in `internal/operation/operation.go`, `parse.go`, and MCP package; use shared recursive schemas, presence-aware CLI JSON, and `SourceFields` name/type checks from RQ-0024.

### Proposed public contract

Proposed names: MCP `semantic_inline_function`, CLI `inline-function`; Go-only initially. ADR-0055 reserves `file` + `symbol` for the callee declaration target. Select the call using a closed `call_site` object so caller and callee can be in distinct files; same-file primitive selectors may be a documented shorthand only if they do not blur these roles.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target file | string | none | Existing Go file declaring the callee. |
| `symbol` | target function | string | none | Qualified callee declaration; resolves the primary target. |
| `call_site` | selector | closed object | none | `{file: string, in_function: string, discriminator: string, construct_path?: string, expected_revision?: string}` selects one call in an existing caller file. The nested revision binds the caller selection. |
| `remove_declaration` | policy | boolean | false | Request deletion after proving no remaining references. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision from lookup/receipt; no stale path reuse. |

The primary receipt continuation identifies the callee using `file`, `symbol`, and its final `expected_revision` while the callee remains. A separate `continuations` array entry identifies the surviving owner function. When deletion is requested, omit the deleted callee as a live target, return only the owner continuation, and place the removed callee under explicitly historical data. Resolve all continuations after normalization.

### Decision gates and semantic risks

1. Code actions are candidates only. First inspect gopls action kinds, returned edit shape, and whether action resolves without command, user interaction, or `applyEdit`. Current session support does not include those behaviors. A transport gate must implement and test them before this route is adopted; no gopls CLI invocation is acceptable evidence.
2. Prove callee resolution and body availability from the same overlay. Refuse recursion, generic instantiation ambiguity, method values, interface dispatch, function values, external definitions, or build variants not covered by analysis.
3. Preserve Go argument evaluation count and order. Parameter substitution is unsafe when an argument is used more than once, moved across side effects, shadowed, or captured. Require temporaries only if a semantics-preserving typed plan is available; otherwise refuse.
4. Handle return values, naked returns, named results, defer, panic/recover, and local variable scope explicitly. The first release should accept only one unambiguous return shape.
5. Resolve the declaration-deletion receipt after all replacements and normalization. Refuse deletion if any references remain or completeness is unknown.

### Ordered implementation handover

1. Read the existing inline-variable implementation and gopls request/session lifecycle; capture actual supported protocol methods and code-action payloads.
2. Build a read-only call selector over sourcebuf, using `call_site.file`, `call_site.in_function`, `call_site.discriminator`, optional `call_site.construct_path`, and `call_site.expected_revision`; return structured ambiguity/refusal evidence.
3. Prototype the narrow straight-line case with `go/ast` and type information. Produce edits in memory, preserving comments and unaffected bytes.
4. Preflight every output, collision, scope binding, call evaluation, and optional reference deletion. Apply all output bytes to the overlay in one `Buffer.Apply`; no operation-owned disk barrier/planner.
5. Implement typed registry/schema and truthful continuation receipts. Mark `Batchable` and `BufferedBatch` true only when overlay semantics are proven; both are required for existing Go targets.
6. Batch execution must defer verification/import organization, retain earlier successful edits on later failure, and publish only final buffered bytes. Per-file atomic publication is guaranteed, whole-batch rollback is not.
7. Add CLI/MCP documentation and public txtar coverage. If a new helper file becomes necessary, require explicit `semantic_scaffold_file` first until sourcebuf resource support exists.

### Required evidence and tests

Every public test runs the CLI or public MCP `semantic_batch` contract and asserts diagnostics and resulting bytes.

* Positive: inline `return adjust(x)` for a local pure helper with one direct caller; assert expression/body expansion, intact callee by default, and continuations usable in a later lookup/edit.
* Refusal: multi-use side-effecting argument, recursion, indirect/function-value call, generic ambiguity, `defer`, branch/loop body, named result or comment-boundary case outside the supported subset, stale revision, and requested deletion with one remaining reference. Assert no changed bytes.
* Preflight: caller and target are in different files; make one secondary output uneditable/stale and prove no output changed.
* Buffered sequence: prior unsaved body replacement → inline function → later mutation of containing function; later operation sees overlay text, disk stays original until final flush, no intermediate publication occurs, and later failure retains only earlier successful overlay changes.
* Retained one-process proof: count gopls process/session identity over earlier lookup, inline planning, and later lookup; all use the same live session and overlay. Do not use `gopls` CLI.
* Byte preservation/no-op: retain comments and source bytes beyond exact replacements; inlining that produces byte-identical source yields an empty diff and no publication. Verify imports are coalesced once at segment end.

### Completion decision

Keep the question open until call-site semantics, transport capability, and overlay proof are demonstrated. Expand from one call only after the same evaluation-order and reference guarantees are independently established.

### Concrete fixture sketches

Add `testdata/scripts/inline_function_go.txtar` for standalone CLI use and `testdata/scripts/mcp_batch_inline_function.txtar` for the shared-buffer public path. The fixture should place the caller and helper in separate existing files, with an unrelated oddly formatted sibling file whose bytes must remain unchanged.

The batch sequence should establish a call in an earlier unsaved edit, inline that exact call, snapshot the owner body immediately, then issue a later mutation that adds a separate verifier function against the statically known owner. This JSON is a request-shape sketch, not receipt interpolation; a standalone follow-up can consume the returned continuation:

```json
{
  "edits": [
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Run(x int) int { return adjust(x) }"
      }
    },
    {
      "tool": "semantic_inline_function",
      "params": {
        "file": "helper.go",
        "symbol": "adjust",
        "call_site": {
          "file": "caller.go",
          "in_function": "Run",
          "discriminator": "adjust(x)"
        }
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Verify() int { return Run(3) }"
      }
    }
  ]
}
```

Provide `helper.go` with the accepted straight-line body; assert the final diff keeps `adjust` by default and the caller contains the equivalent body. The CLI txtar must invoke `semedit inline-function` and compare expected source bytes, not merely report a successful exit.

Add a public refusal fixture for `adjust(next())` where inlining would duplicate or reorder evaluation, one recursive helper, and an indirect function value. Assert each response identifies the unsupported reason and all source files remain byte-identical. A deletion-request fixture with a second call site must refuse deletion while leaving the inline itself unapplied unless the contract explicitly supports inline-without-delete as a separate successful action.

A focused buffered test must snapshot `Run` from the overlay immediately after inlining and before any later mutation; assert that the selected `adjust(x)` call is gone and the equivalent body is present. Then prove the later verifier insertion resolves `Run` through the same gopls process/session. Disk remains original before flush. A separate stale-preimage test changes a captured input before `Buffer.Apply` or `Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published. Assert one coalesced verification/import pass and that a later intentional batch failure retains prior successful buffered changes without publishing intermediate bytes.

### Expansion milestone and artifact checklist

Milestone two may support callers across packages only after complete reference and visibility proofs; support multiple call sites only after distinct argument evaluation traces are preserved; milestone three may consider methods or cross-file declaration removal after complete references are proven. Every expansion needs a public txtar pair for success/refusal and must update backend capability metadata. The handover is complete when no response suggests the removed helper remains a live target.

### Expected research artifacts

Record a call-body-form matrix with evaluation-order notes, reference completeness evidence for declaration deletion, and the gopls action/transport trace. Preserve the distinction between a candidate action's availability and proven semantic correctness.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_inline_function`:**

> Use this tool instead of copying a helper body into its caller to inline one supported direct Go call. `file` and `symbol` identify the callee; `call_site` selects the caller independently. The helper remains unless `remove_declaration=true` passes complete-reference checks. Pass a surviving callable's `file` and `symbol` from `continuations` to `semantic_rename` or `semantic_replace_body`. A deleted helper is historical evidence and is never offered as a live next-call target.

**Parameter description requirements:**

`symbol`: Target: qualified callee; copy verbatim from `semantic_lookup.symbol` with its `file`. `call_site`: Selector: one current caller location; copy the complete same-shaped `call_site` projection from proposed `semantic_inspect_symbol.call_site` only after that same-shaped producer is implemented. Do not construct it by renaming a callee receipt's fields. `remove_declaration`: Policy: optional Boolean, default false; deletion is part of the same validated operation.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go extract constant

Original handover: RQ-0043. Implementation status: Open.

### Purpose and shared contract

Plan `semantic_extract_constant` for one selected Go constant expression, replacing it with a package-level constant while preserving meaning and source locality. The plan is a research handover, not an accepted contract. Apply [RQ-0024 § Shared Handover Contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract).

Sibling handovers: [change signature](RQ-0060-go-refactoring-tool-implementation.md#go-change-signature), [inline function](RQ-0060-go-refactoring-tool-implementation.md#go-inline-function), [extract interface](RQ-0060-go-refactoring-tool-implementation.md#go-extract-interface), [introduce parameter](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter), [introduce parameter object](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter-object).

### Observable outcome and boundaries

A successful operation inserts a named `const` declaration in the selected existing file, replaces exactly one selected expression with its identifier, and validates the edited package. Return the new constant through a `continuations` array entry `{file, symbol, expected_revision}` resolved after formatting/import normalization. Keep historical planned names separate from live continuation targets. It does not return the old expression selector as an active continuation after the replacement.

First release: one unambiguous compile-time constant expression inside a function or method in one existing file; infer the exact constant type/value with Go type information; insert in package scope near the file's other declarations; replace one exact AST expression. Refuse `iota`-dependent or context-sensitive constants, expressions whose type/value cannot be represented exactly, duplicate identifier collisions, directives/comments that cannot be safely anchored, cross-file use, and any expression with runtime evaluation. No extraction of variables, multiple occurrences, enum generation, generated files, or new-file creation.

### Audited starting state

The authoritative source tree `/Users/alessandro/sources/semantic-editor` at `6c95d12` was audited read-only, including uncommitted shared buffers. Requested operations are absent across worktrees. Current Go tools include extraction of a function and a variable (`internal/astedit/extract.go`, `extract_variable.go`, `wire_extract.go`), inline-variable only (`inline.go`, `wire_engine_inline.go`), and AST declaration/body editing. These are selection and wire-pattern evidence, not an extract-constant implementation. The registry, sourcebuf overlays, batch executor, and single-process gopls sessions are present in the audited tree. Current LSP client does not prove command, interactive, or applyEdit support; a pending ADR-0060 in that source tree concerns diagnostic sessions only.

### Candidate implementation paths

* AST edit: new focused `internal/astedit/extract_constant.go`, following `extract_variable.go`, `insert.go`, `decl.go`, and `comment_anchor.go`; avoid whole-file formatting rewrites.
* Semantic analysis: `internal/adapters/golang/gopls.go`, `internal/lsp/client.go`, and `internal/symbol/resolver.go`; evaluate gopls code actions as candidates only after transport shape is proven.
* Wire: a distinct operation definition in `internal/operation/wire_extract.go` or sibling. Existing `ExtractVariableReq` provides selector conventions but has a different local-variable semantics.
* Buffered execution: `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, `internal/mcp/batch.go`. Plan both the declaration insertion and replacement as one complete output and call `Buffer.Apply` once.
* Schema gate: shared recursive object/array schema plus presence-aware CLI JSON and type-checked `SourceFields`; do not use opaque JSON strings.

### Proposed public contract

Proposed MCP name `semantic_extract_constant`, CLI `extract-constant`, Go backend only. Use ADR-0055 roles: `symbol` names the new constant, `in_function` scopes the selected expression, and `discriminator` / `construct_path` select the exact expression. The owner selector is not a target. `expected_revision` is optional for a fresh target and binds the selected `file` only; capture and validate the owner selection and complete output preimages separately.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target file | string | none | Existing `.go` file where expression and declaration will live. |
| `symbol` | created target | string | none | New package-level identifier, e.g. `maxRetries`. |
| `in_function` | scope selector | string | none | Qualified owner function or method. |
| `discriminator` | construct selector | string | none | Exact source spelling of one candidate constant expression. |
| `construct_path` | disambiguator | string | omit if unique | Current candidate path, valid only for bound revision. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token for the selected `file` only; bind expression selection to that file revision and validate all planned outputs separately. |

Receipt fields are the surviving `file`, new `symbol`, and final revision, with exact field types matching their inputs. Do not echo `discriminator` or path as a reusable live selection after mutation. Source-field descriptions must name their producing tools and schema validation must compare name and type. Use the shared `continuations` array shape rather than an operation-specific `generated` object.

### Decision gates and risks

1. Determine whether a selected expression is a Go constant expression in its exact context, including untyped constants, conversions, shifts, array lengths, and `iota`. Preserve assignability and default typing; reject any uncertain case.
2. Choose declaration placement and comment anchoring that preserve build directives and file-level comments. Existing import-group or declaration layout must not be normalized unnecessarily.
3. Expression selection must be syntax-backed and revision-bound. If matching by exact source text produces multiple candidates, require a fresh candidate path; never reuse a stale path.
4. Code actions may be investigated, but any action needing command execution, user interaction, or `workspace/applyEdit` is blocked until transport support is implemented and tested. Do not substitute gopls CLI.
5. Calculate both replacement and insertion before `Buffer.Apply`; validate collision, type-check, and every output preimage first. Empty or same-name no-op behavior must be explicit.

### Ordered implementation handover

1. Map current extract-variable selection and declaration anchoring behavior; inspect gopls constant metadata from the retained server.
2. Define a read-only expression classifier that returns constant value and exact type information from the shared overlay. Record supported constant forms before mutation work.
3. Implement a pure planner with original byte ranges and two edits. Preserve comments, line endings, and unrelated file bytes; handle same declaration/file position overlaps explicitly.
4. Run syntax/type/collision checks and revision validation against all outputs. On success call `Buffer.Apply` once with all outputs; direct mode stages atomic per-file writes, with no multi-file rollback promise.
5. Add the registry contract, schema and descriptions after the shared typed-schema gate. Set both `Batchable` and `BufferedBatch` for existing Go files.
6. Make buffered batches defer imports/verification and coalesce them once per segment. Later edits must see the new identifier through sourcebuf and the same gopls process.
7. Add command help, MCP description, capability metadata, and CLI txtar cases. If creation of a new destination file is ever required, first invoke explicit `semantic_scaffold_file`; sourcebuf new-resource support is not designed.

### Required evidence and tests

Tests must exercise public CLI output or public MCP batch calls and inspect exact workspace state.

* Positive: extract `3 * 60` to a uniquely named `const retryWindow = 3 * 60` in an existing function; assert typed equivalence, declaration placement, returned target continuation, and follow-up lookup/edit using that receipt.
* Refusals: runtime expression, `iota` outside a proven safe declaration context, name collision, duplicate selection, non-constant conversion, stale revision, malformed type state, and unanchorable directive/comment. Assert diagnostics and byte-identical source.
* All-output preflight: include a second planned output (if the chosen type/import/declaration plan requires it), make that output stale, and assert neither the expression nor declaration changes. If first release is strictly one-file, still prove the complete output set is prevalidated before either edit applies.
* Buffered sequence: unsaved `semantic_replace_body` creates the expression → extract constant → a later mutation resolves and uses the new symbol. Assert the overlay is visible, disk bytes stay unchanged until final flush, and a separate post-batch lookup/edit consumes the returned continuation; a later failure keeps prior successful buffered edits only.
* Retained server: prove the same one-process gopls session sees the earlier edit, resolves the expression, and supports the later mutation; perform any read-only lookup after the batch because `semantic_lookup` is not batchable. Do not use gopls CLI.
* Byte/no-op: preserve comments, CRLF/LF choice, directives, imports, and unrelated bytes. A selected expression already equal to a collision-free constant reference must either be a documented refusal or produce an empty diff with no write; do not report a successful mutation with an empty/ambiguous target.

### Completion decision

Resolve the question only after the supported constant subset is explicit, exact type preservation is demonstrated, and the public buffered txtar sequence passes. Expand to multiple occurrences or files only with a new completeness and preflight proof.

### Concrete fixture sketches

Add `testdata/scripts/extract_constant_go.txtar` plus `testdata/scripts/mcp_batch_extract_constant.txtar`. Use one package with an expression nested in a direct return, a name collision case, a nonconstant expression, and an unrelated source file with deliberately non-gofmt spacing.

The buffered sequence should create the selected expression in an earlier unsaved edit, extract it, then apply a later mutation through normal resolution. Use a standalone post-batch lookup to consume the returned continuation:

```json
{
  "edits": [
    {
      "tool": "semantic_replace_body",
      "params": {
        "file": "retry.go",
        "symbol": "Retry",
        "body": "return 3 * 60"
      }
    },
    {
      "tool": "semantic_extract_constant",
      "params": {
        "file": "retry.go",
        "symbol": "retryWindow",
        "in_function": "Retry",
        "discriminator": "3 * 60"
      }
    },
    {
      "tool": "semantic_replace_body",
      "params": {
        "file": "retry.go",
        "symbol": "Retry",
        "body": "return int(retryWindow)"
      }
    }
  ]
}
```

The fixture must prove the conversion remains assignable for the function's actual result type; adjust the exact expression/type so the expected source compiles. `exec semedit extract-constant ...` is the standalone public CLI test. Compare the declaration location, exact replacement, diagnostics, and unrelated file bytes.

Add refusal cases for runtime calls, `iota` in an unsupported context, a duplicate name, two identical expression candidates without `construct_path`, and stale `expected_revision`. Assert no source mutation for every refusal. A same-expression already-constant case must specify either a structured no-op with empty diff or explicit refusal.

The batch buffer test must inspect original disk bytes before final flush and overlay bytes after extraction; a separate stale-preimage test changes a captured input before `Buffer.Apply`/`Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published; a later lookup/edit must use the new symbol without restarting gopls. Assert same one-process session, one final verification/import pass, complete output preflight before `Buffer.Apply`, and earlier successful edits retained on a later operation error. Publication remains atomic per file only.

### Expansion milestone and artifact checklist

Milestone two may permit multiple exact expression occurrences only with a stable occurrence selector and atomic preflight; milestone three may support existing destination files only with package and visibility checks. Keep `iota`, generated declarations, and cross-package promotion out until separately evidenced. The handover is complete when the result's new constant resolves from its advertised `file`/`symbol` pair.

### Expected research artifacts

Record the Go constant-expression forms accepted and refused, typed-value preservation examples, and exact comment/directive placement behavior. Include one gopls metadata trace only if it contributes evidence beyond the AST/type checker.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_constant`:**

> Use this tool instead of copying an expression into a new declaration to extract one supported Go constant expression. Choose a fresh simple `symbol`; `in_function`, `discriminator`, and an optional revision-bound `construct_path` select the existing expression. The engine preserves contextual typing or refuses the extraction. Pass the created constant's live `file` and `symbol` to `semantic_rename` or `semantic_inspect_symbol`. Replaced expression text and old paths are evidence, not selectors for another extraction.

**Parameter description requirements:**

`symbol`: Target: caller-chosen new constant identifier such as `maxRetries`; no preceding existing-symbol result supplies this new name. `in_function` and `discriminator`: Selectors: copy `in_function` from a live `semantic_extract_function.in_function` or proposed `semantic_inspect_symbol.in_function`, and `discriminator` from proposed `semantic_inspect_symbol.discriminator` for that current expression; do not copy `semantic_lookup.symbol` or replaced expression evidence. `construct_path` is paired with its selected-file `expected_revision`.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go extract interface

Original handover: RQ-0044. Implementation status: Open.

### Purpose and shared contract

Plan a Go-first `semantic_extract_interface` operation that creates a named interface from selected methods of a concrete type while preserving the concrete type. This is an actionable research handover, not an accepted decision. Follow [RQ-0024 § Shared Handover Contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract).

Sibling plans: [change signature](RQ-0060-go-refactoring-tool-implementation.md#go-change-signature), [inline function](RQ-0060-go-refactoring-tool-implementation.md#go-inline-function), [extract constant](RQ-0060-go-refactoring-tool-implementation.md#go-extract-constant), [introduce parameter](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter), [introduce parameter object](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter-object).

### Observable outcome and boundaries

Success inserts a package-level interface declaration from the selected method set in an existing destination file, verifies the concrete type satisfies it, and preserves the concrete type and methods. Return the interface target and, when useful, its source type as distinct entries in the shared `continuations` array; keep planned names and deleted/renamed targets in separate historical data. Do not create a custom `generated` result shape.

First release: one named, non-generic concrete type in one Go package; select a non-empty explicit subset of exported or same-package methods whose signatures are fully resolved; write interface into the same existing file or an existing same-package `destination_file`. Refuse pointer/value method-set ambiguity, embedded/promoted methods, generic types or methods, inaccessible cross-package methods, method expressions that cannot be represented, constraints, conflicting interface names, and incomplete type information. Do not implement the contract, rewrite dependencies, move methods, or create a new file.

### Audited starting state

Read-only code audit was performed against `/Users/alessandro/sources/semantic-editor` at `6c95d12`, including shared uncommitted state. All six requested operations were absent across worktrees. Go type inspection, declaration insertion, scaffold-file, move, and references exist as lower-level capabilities; no extract-interface operation exists. Receiver method insertion is present but is a separate action. Current source tree has typed registry and sourcebuf batch support. The active LSP implementation does not demonstrate execute-command, interactive, or applyEdit support. Its pending ADR-0060 context is not a decision for this feature.

### Candidate implementation paths

* Resolve method sets and types through `internal/adapters/golang/gopls.go`, `internal/lsp/client.go`, `internal/symbol/resolver.go`, and existing type/outline helpers in `internal/astedit/type.go` and `inspect.go`.
* Build declaration bytes and placement in a focused `internal/astedit/extract_interface.go`, reusing safe declaration/comment anchoring from `decl.go`, `insert.go`, and `comment_anchor.go`.
* Register a typed operation in a dedicated wire file or `wire_engine_declarations.go`; follow `wire_extract.go` for parsing and `FileEditRes` formatting while avoiding stale target receipts.
* Buffer outputs through `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, and `internal/mcp/batch.go`; preflight both source and destination before one `Buffer.Apply`.
* Wait for the shared typed recursive schema for any structured method metadata. `members: MemberTarget[]` requires the shared recursive object schema; emitted method signatures are structured output and need matching typed schemas. `SourceFields` must validate both names and types.

### Proposed public contract

Proposed MCP `semantic_extract_interface`, CLI `extract-interface`, Go-only. `file` is the source file; `in_type` is the sole owner-type selector for the concrete source type. `symbol` names the new interface target, and `destination_file` is where it will be declared. Use `expected_revision` only for the selected source `file`; independently capture and validate the member set, method signatures, and destination preimage. Require fresh member selection if the resolved method set changes.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | source file | string | none | Existing `.go` file declaring the concrete type. |
| `in_type` | owner selector | string | none | Qualified concrete type; sole source owner-type selector. |
| `symbol` | created target | string | none | New package-level interface identifier. |
| `destination_file` | destination target | string | same as `file` | Existing `.go` file in the same package. |
| `receiver_kind` | policy | string enum | none | Required `pointer` or `value`; validate satisfaction by that exact method set. |
| `members` | selection | array of `MemberTarget` objects | none | Non-empty list of `{file: string, symbol: string}` entries identifying exact method declarations; use the same member target shape as the field/composite operation plans. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token for the selected source `file` only; member files and destination are captured and validated separately. |

Return the interface as the primary continuation `{file: destination_file, symbol, expected_revision}`. The unchanged source type need not be returned as a continuation. The interface continuation is the primary operation result; do not claim the input `file` + `in_type` pair is the interface target. Never report a removed/replaced method as a live symbol. All generated targets use `continuations`, not `generated`. Method signature output must distinguish name, parameter names/types, results, and variadic status in typed fields.

### Decision gates and semantic risks

1. Require `receiver_kind: pointer | value`, matching implement-contract. Verify satisfaction by `*T` or `T` respectively; method receiver spelling alone does not select the requested method set.
2. Embedded/promoted methods and generic receiver/type parameters need a clear representation and selection proof; refuse them in release one unless gopls resolves exact declarations and interface signatures.
3. Decide whether unexported methods are allowed only for same-package interfaces. Never construct an interface that cannot be satisfied outside its package while implying broader usability.
4. Validate duplicate names, parameter/result signatures, variadic forms, imports, comments, and destination package equality before mutation.
5. Gopls code actions are candidates only. Gate on returned edit shape and transport support; command, interaction, and applyEdit are absent from current session support. Do not use gopls CLI.
6. Resolve interface receipt after formatting/import organization and verify the new symbol is still live at its final destination.

### Ordered implementation handover

1. Prototype read-only method-set enumeration against a real in-process gopls session and compare it with Go type-checker method sets for pointer/value receivers and embedded types.
2. Record refusal diagnostics for unsupported or incomplete method resolution. Do not silently omit methods from an apparent “all methods” result.
3. Define the output signature object schema and source-field continuations, then add `semantic_extract_interface` to the central registry only after the recursive schema/CLI JSON gate.
4. Generate a declaration from type-checked signatures; preserve docs only when attribution is unambiguous and avoid claiming interface docs belong to methods if that changes meaning.
5. Validate the selected source revision, independently captured member/body/destination preimages, name collision, package identity, interface satisfaction, and every changed file before applying one `Buffer.Apply`.
6. Set `Batchable` and `BufferedBatch` true for existing Go files. Use sourcebuf for all reads and edits, no disk planner/barrier, deferred/coalesced verification and imports. Retain prior successful edits on later batch failure; publication is atomic per file, not whole batch.
7. Provide CLI/MCP schema, capability metadata, and public txtar coverage. Require explicit `semantic_scaffold_file` before any new destination file until sourcebuf resource semantics are designed.

### Required evidence and tests

Public CLI txtar tests assert output/diagnostics and complete resulting workspace state; MCP batch examples exercise the same public registry entry.

* Positive: extract `Read` and `Write` from a concrete `Store` into `StoreAPI` in the same existing file; assert signatures, satisfaction by the intended pointer/value type, preserved concrete type, and usable destination/interface receipt for later lookup and edit.
* Refusals: one missing method, pointer/value ambiguity, embedded promoted method, generic type, duplicate interface name, invalid destination package, unresolved method signature, stale revision, and unsupported unexported cross-package method. Assert all relevant files remain byte-identical.
* All-output preflight: source and destination are separate files; stale or unwritable destination prevents the interface declaration and any source edit. First release should not alter source file unless a specific interface assertion is an explicit output.
* Buffered sequence: unsaved insertion of a receiver method → extract interface → later mutation that consumes the new interface type. A standalone follow-up may use the interface continuation for lookup/edit. Prove overlay visibility, no disk publication before final flush, one coalesced verification/import pass, and retention of earlier successful overlay edits if the last batch operation fails.
* Retained server: demonstrate one gopls process/session serves method-set/type checks before and after earlier buffered edits; do not use its CLI.
* Byte/no-op: preserve comments, method bodies, imports, and unrelated bytes. If the requested interface already exists exactly, return an explicit no-op with empty diff; do not write or report a duplicate created target.

### Completion decision

Resolve after type-set semantics and public transport are evidenced, and after the batch test demonstrates that generated interface text is derived from the same unsaved revision later operations inspect.

### Concrete fixture sketches

Add `testdata/scripts/extract_interface_go.txtar` and `testdata/scripts/mcp_batch_extract_interface.txtar`. Use `store.go` with a concrete type and pointer receiver methods, `store_test.go` with a compile-time interface assertion, and an unrelated formatting-sensitive source file. Keep all destination files existing in release-one fixtures.

Exercise an earlier unsaved receiver method, extract an interface from the same overlay, then insert a later declaration that consumes the generated interface type. This JSON is a request-shape sketch, not receipt interpolation; a standalone follow-up can consume the interface continuation:

```json
{
  "edits": [
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "store.go",
        "source": "func (s *Store) Close() error { return nil }"
      }
    },
    {
      "tool": "semantic_extract_interface",
      "params": {
        "file": "store.go",
        "in_type": "Store",
        "symbol": "StoreAPI",
        "receiver_kind": "pointer",
        "destination_file": "store.go",
        "members": [
          {
            "file": "store.go",
            "symbol": "(*Store).Read"
          },
          {
            "file": "store.go",
            "symbol": "(*Store).Close"
          }
        ]
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "store.go",
        "source": "func UseStore(s StoreAPI) {}"
      }
    }
  ]
}
```

Provide signatures that type-check and assert the intended pointer method set satisfies `StoreAPI`, while the concrete type and methods remain unchanged. The standalone CLI txtar invokes `semedit extract-interface ...` and checks the exact generated declaration and live receipt fields.

Refusal fixtures cover value/pointer mismatch, embedded promotion, generic receiver, duplicate interface name, wrong-package destination, missing selected method, and stale revision. For separate source/destination files, force destination preflight failure and compare both files to their original bytes.

The batch test must prove one overlay revision across inserted method, method-set analysis, interface insertion, and later type-resolving mutation; a separate stale-preimage case changes the selected source `file` before `Buffer.Apply`/`Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published. Independently preflight the complete member and destination set. Assert disk is unchanged before flush, all destination preimages are checked before apply, imports/verification coalesce once, and earlier edits survive a later failure in the overlay. Record the same gopls process/session ID through all steps; no gopls CLI. Per-file atomic publication does not imply multi-file rollback.

### Expansion milestone and artifact checklist

Release one includes an existing same-package destination distinct from the source file, with complete destination preflight. Later milestones may consider embedded or generic method sets only after exact type-set proof. Cross-package consumers, new files, and method implementation are separate capability decisions. A no-op on an identical existing interface must have an explicit receipt and no write.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_interface`:**

> Use this tool instead of reconstructing method signatures to create a Go interface from an explicit resolved method selection. Choose the new interface `symbol`; `in_type` and `members` select the concrete source type and methods. Choose the required `receiver_kind` as `pointer` or `value`; satisfaction is checked for that exact method set. `destination_file` defaults to the source `file` and must already exist in the same package. Pass the interface continuation's destination `file` and live `symbol` to `semantic_inspect_symbol` or `semantic_rename`. This operation generates the interface declaration, without implementing its methods or rewriting consumers.

**Parameter description requirements:**

`symbol`: Target: new simple interface identifier chosen by the caller, not the source type name. `in_type`: Selector: copy proposed `semantic_inspect_symbol.in_type` only after the shared inspection extension exists. `members`: Selector: copy proposed `semantic_inspect_symbol.members`, the exact typed member records from that inspection result, with matching `file` and `symbol`; do not rebuild method identities from display signatures. The result `file` names the generated interface's destination. `receiver_kind`: Policy: required `pointer` or `value`, consistent with `semantic_implement_contract`; descriptive lookup `kind` does not feed it.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go introduce parameter

Original handover: RQ-0045. Implementation status: Open.

### Purpose and shared contract

Plan a Go-first `semantic_introduce_parameter` operation that adds one parameter to a function or method and updates every supported direct call site with a caller-supplied Go expression. This remains open research, not an accepted decision. Follow [RQ-0024 § Shared Handover Contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract).

Sibling plans: [change signature](RQ-0060-go-refactoring-tool-implementation.md#go-change-signature), [inline function](RQ-0060-go-refactoring-tool-implementation.md#go-inline-function), [extract constant](RQ-0060-go-refactoring-tool-implementation.md#go-extract-constant), [extract interface](RQ-0060-go-refactoring-tool-implementation.md#go-extract-interface), and [introduce parameter object](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter-object).

### Observable outcome and boundaries

Success replaces one selected safe expression in the owner body with a new parameter reference, appends that parameter in release one, inserts its closed `argument` at every proven direct call site, type-checks all outputs, and returns the surviving target continuation plus diff/diagnostics. A later lookup/edit must be able to consume the target receipt.

First release: one private, non-generic, non-variadic free function or receiver method with a complete same-package set of statically resolved direct calls and one exact owner-body expression selection. The ordered final `parameters` array maps existing positions and appends one new parameter in release one; reordering is deferred. `discriminator` selects one safe expression in the target body, and optional `construct_path` disambiguates only within the bound file revision. Release one requires both the selected expression and its supplied argument to be compile-time constants with the same typed value at every call. Moving their evaluation from the body to the caller must preserve observable behavior; mutable package state and function calls are refused. Refuse external/exported API boundaries, unknown callers, interface dispatch, method values/expressions, function values, generated/build-constrained files, expressions with inconsistent binding/availability, and type mismatches. Caller-local bindings and mapping extensions are later work.

### Audited starting state

Audit source was `/Users/alessandro/sources/semantic-editor` at `6c95d12` and was read-only, including uncommitted shared buffers. The requested six operations were absent in all worktrees. There is no signature/call-site mutation operation. Existing registry/wire patterns include body replacement, extraction, rename, movement, and batched edits; Go references/type information are available through adapter/session paths. `sourcebuf` and `Buffer.Apply` now provide a shared overlay in the source tree. Current LSP session does not establish command, interactive, or applyEdit support. ADR-0060 is pending in that tree and describes diagnostic-session ownership only.

### Candidate implementation paths

* Semantic proof: `internal/adapters/golang/gopls.go`, `internal/lsp/client.go`, `internal/symbol/resolver.go`; enumerate all direct references in sourcebuf view and type-check each proposed call.
* AST edits: implement focused logic in `internal/astedit/introduce_parameter.go`, drawing from `function.go`, `extract.go`, `rename` edit handling, and `move_construct.go`'s overlay planning patterns.
* Public surface: central registry via a dedicated `internal/operation/wire_signature.go` or `wire_engine_mutations.go`, using `operation.FileEditRes` if receipt fields can describe all outputs safely.
* Batch: `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, and `internal/mcp/batch.go`; never put a disk-planner/barrier in the operation. Plan all call files first and apply as one overlay change set.
* Structured parameter object `{name,type}` requires the shared closed recursive schema and presence-aware CLI JSON prerequisite. `argument` is a string containing one parsed Go expression, not opaque serialized data.

### Proposed public contract

Proposed MCP `semantic_introduce_parameter`, CLI `introduce-parameter`, Go-only initially. ADR-0055 `symbol` is the changed declaration target; `in_function` is not a replacement for that target. Parameter value shape must be typed, and result continuations must match input names/types.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target file | string | none | Existing declaration file. |
| `symbol` | target | string | none | Qualified function or method. |
| `parameters` | replacement | array of closed `ParameterMapping` objects | none | Complete ordered final parameter list. Each entry is `{source_index?: integer, parameter: ParameterSpec{name: string, type: string, variadic?: boolean}, argument?: Goexpr}`. Existing parameters use zero-based `source_index` in unchanged order/name/type; the one appended parameter omits it and supplies `argument`. |
| `discriminator` | owner-body selector | string | none | Exact source of one safe expression in the target function/method body to replace with the new parameter reference. |
| `construct_path` | disambiguator | string | omit if unique | Candidate path from a current ambiguity result; valid only for the selected file revision. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token for the selected declaration `file` only; the complete call/reference set is captured and validated separately. |

The ordered `parameters` array determines final signature order; no separate `position` selector is accepted. The selected `discriminator` (and optional `construct_path`) names the exact owner-body expression to replace with the new parameter reference; `expected_revision` binds that fragile selection to the selected declaration file. `symbol` already targets the containing function, so no redundant `in_function` owner selector is needed. Prove each new argument has the same binding and availability at every call. The selected owner-body expression must be safe to replace without changing control flow, and its type must match the new parameter. The call-site `argument` must have the same binding and availability at every call. Same binding, availability, and assignability are necessary but do not establish equivalence. Prove the supplied argument preserves the selected expression's value and evaluation timing/count. Release one admits only equivalent compile-time constants; reject mutable package-variable reads, function calls, and caller-local names. Return the changed declaration in a shared `continuations` entry with `file`, `symbol`, and `expected_revision`; call-site paths are diff evidence, not live selectors.

### Decision gates and semantic risks

1. Prove a complete reference set using typed resolution. A textual scan is not enough; refuse when build variants, generated files, external packages, reflection, or unresolved references could hide callers.
2. Verify new parameter type syntax, identifier uniqueness, receiver shadowing, and body bindings. Parameter insertion can shadow package names or alter resolution; check all affected scopes.
3. Check expression assignability, equal typed constant value, and preservation of observable evaluation timing/count at each call. Appending the argument evaluates it before the callee body; same-object resolution alone cannot justify moving a mutable read. For example, `toggle(); return packageFlag` must refuse selecting `packageFlag`, since a caller argument reads it before `toggle`. Broader expressions require an independent equivalence and ordering proof before expansion.
4. Exported functions and methods implementing interfaces need an explicit compatibility policy and all implementers/callers; keep them refused until that migration boundary is proven.
5. Gopls refactoring actions are candidates. Current session has no established command, interactive, or applyEdit support; a transport gate is required before adopting any action with those requirements. Do not use gopls CLI.
6. Match revision receipt creation to RQ-0024 after final normalization. Never return stale owner-body `discriminator` or `construct_path` values as follow-up targets.

### Ordered implementation handover

1. Build a read-only typed call-graph prototype for a private function. Classify supported direct calls and every refusal source; compare with gopls reference responses.
2. Define complete call-set and argument-expression semantics before adding a public registry entry. Use parser/type-checker validation on each caller against the sourcebuf overlay.
3. Implement a pure multi-file planner capturing original bytes and source ranges for declaration and calls. Preserve comments, unrelated formatting, and argument evaluation order.
4. Preflight all outputs, imports, collisions, expected revisions, and diagnostics inputs. Apply the entire output set in one `Buffer.Apply`; direct multi-file publication remains per-file atomic and can partially publish only with explicit reported count on failure.
5. Add typed wire schema and CLI JSON support after shared prerequisite; register the Go operation with `Batchable: true` and `BufferedBatch: true` for existing files.
6. In buffered batch, use sourcebuf for AST and LSP, retain one gopls process, defer verification/import cleanup and coalesce them at the boundary. Earlier successful edits remain if a later edit fails; there is no whole-batch rollback guarantee.
7. Add CLI/MCP help, capabilities, and txtar coverage. New callers/files cannot be created by this operation; use explicit `semantic_scaffold_file` first if a later workflow needs a new existing file.

### Required evidence and tests

Tests must invoke the public CLI command or MCP `semantic_batch`, assert user-visible output/diagnostics, and inspect disk/overlay state.

* Positive: append `bool` to a private function, select `true` in the body and provide the equivalent closed `true` argument, update calls across two existing files in one package, type-check, and consume the returned continuation in a standalone follow-up.
* Refusals: missing call site in a build-tagged file, interface implementation, method value, function variable, duplicate parameter, invalid type, non-assignable argument, side-effect-order hazard, mutable package-state read after a callee mutation, unequal constant value, stale revision, and generated source. Assert every file remains unchanged.
* All-output preflight: two caller files, second contains untransformable call; assert neither first caller nor declaration changes.
* Buffered sequence: earlier unsaved `semantic_insert_function` or body edit creates a call → introduce parameter → a later mutation resolves and uses the migrated signature; disk is unchanged before flush; final diff contains only final bytes; later failure retains earlier successful buffered edits.
* Retained server: prove call/reference resolution and the later mutation use one gopls process/session over the exact overlay, with no gopls CLI.
* Byte/no-op: preserve comments, line endings, argument spacing, and unrelated imports. If parameter and all calls already match exactly, produce an empty diff and no disk write; do not claim a mutation occurred.

### Completion decision

Resolve only after call-set completeness and argument-order proof hold for the stated boundary and the public batch sequence passes. Any need for project-wide/API migration requires a separate scope decision before expanding the operation.

### Concrete fixture sketches

Add `testdata/scripts/introduce_parameter_go.txtar` and `testdata/scripts/mcp_batch_introduce_parameter.txtar`. Use a private function called from two existing files plus one test file. Include a build-tagged unresolved caller refusal and a caller expression with visible side effects for the ordering test.

The batch must make an earlier unsaved caller edit, replace the selected owner-body expression with the new parameter reference, append the parameter, and prove a later verifier insertion resolves the migrated declaration. Omit `expected_revision` for this fresh overlay target. This JSON is a request-shape sketch, not receipt interpolation; a standalone follow-up can consume the returned continuation:

```json
{
  "edits": [
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Run() bool { return open(1) }"
      }
    },
    {
      "tool": "semantic_introduce_parameter",
      "params": {
        "file": "open.go",
        "symbol": "open",
        "parameters": [
          {
            "source_index": 0,
            "parameter": {
              "name": "id",
              "type": "int"
            }
          },
          {
            "parameter": {
              "name": "enabled",
              "type": "bool"
            },
            "argument": "true"
          }
        ],
        "discriminator": "true"
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Verify() bool { return open(1, true) }"
      }
    }
  ]
}
```

The owner starts with `return true`, so passing the same literal preserves its original value for every input; assert introduction replaces the selected `true` expression with `enabled` and changes callers to pass `true`. Snapshot the result before the verifier insertion. The later verifier insertion confirms the new signature resolves without overwriting the transformation. The original `id` argument remains at index zero. Import organization occurs once at the segment boundary. The standalone `exec semedit introduce-parameter ...` case checks all callers, imports, diagnostics, and final compilation with the exact expected files.

Refusal txtars must cover an uneditable second caller, a function-value use, interface implementation, stale revision, duplicate parameter, nonassignable argument, and argument-order hazard such as changing call order around a side-effecting argument. Add `toggle(); return packageFlag` with a proposed `packageFlag` caller argument to prove that same binding and side-effect-free reads still refuse when evaluation timing changes; also refuse unequal constants. Each refusal asserts the target and every caller remain unchanged.

A buffer-focused test should assert the call-set and argument type-check complete before `Buffer.Apply`, then inspect the owner-body replacement and all caller outputs and original disk bytes before flush; a separate stale-preimage case changes a captured input before `Buffer.Apply`/`Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published. Prove one gopls process/session across earlier edit, call discovery, and later lookup; one coalesced verification/import pass; and retention of earlier successful operations when a subsequent batch item fails. Avoid any assertion of whole-batch rollback after final per-file publication begins.

### Expansion milestone and artifact checklist

Milestone two may accept caller-local or mutable expressions only with per-call binding/availability, equal-value, and observable evaluation timing/count proofs; multiple added parameters and other mappings remain gated by independent argument-order evidence; milestone three may consider exported APIs or interface implementers under a migration contract. Each supported language form must be advertised through backend capability metadata and exercised by txtar. Do not infer a default argument from nearby syntax.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_introduce_parameter`:**

> Use this tool instead of editing a body, signature, and callers separately to replace one supported Go body expression with a new parameter. Supply `discriminator` and the complete ordered `parameters` mapping with an appended argument. Release one accepts equivalent compile-time constants and refuses changed values or evaluation timing. Pass the surviving callable's `file` and `symbol` from `continuations` to `semantic_rename` or `semantic_replace_body`. The replaced body expression is historical evidence, not a live selector.

**Parameter description requirements:**

`symbol`: Target: existing qualified callable copied from `semantic_lookup.symbol` with its `file`. `discriminator`: Selector: exact current body expression, copied only from proposed `semantic_inspect_symbol.discriminator` for this current callable expression and revision. `parameters`: Replacement: final ordered mapping. Its new `argument` is a caller-supplied Go expression, never a guessed default from the removed body text.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go introduce parameter object

Original handover: RQ-0046. Implementation status: Open.

### Purpose and shared contract

Plan a Go-first `semantic_introduce_parameter_object` operation that groups a selected function's parameters into one named struct and migrates its body and supported callers. This is an implementation handover, not an accepted decision. Apply [RQ-0024 § Shared Handover Contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract).

Sibling investigations: [change signature](RQ-0060-go-refactoring-tool-implementation.md#go-change-signature), [inline function](RQ-0060-go-refactoring-tool-implementation.md#go-inline-function), [extract constant](RQ-0060-go-refactoring-tool-implementation.md#go-extract-constant), [extract interface](RQ-0060-go-refactoring-tool-implementation.md#go-extract-interface), and [introduce parameter](RQ-0060-go-refactoring-tool-implementation.md#go-introduce-parameter).

### Observable outcome and boundaries

Success inserts a new package-level struct, replaces selected function parameters with one object parameter, rewrites body references to fields, migrates each supported call to a keyed struct literal, and type-checks all outputs. Return the surviving function and the created struct as distinct targets in the shared `continuations` array, each with copyable `file`, `symbol`, and `expected_revision`. Planned names and removed targets stay outside live continuations.

First release: migrate all parameters of one private, non-generic, non-variadic free function into a new struct in the same existing file and package. One target declaration may have direct callers across multiple existing files in the same package within one module; preflight every caller. This boundary keeps evaluation order stable by constructing fields in original argument order as the sole argument. Preserve named parameter types and body meaning. Refuse partial grouping, exported APIs, unknown callers, interfaces, function values, generic types, promoted fields, conflicting names, comments/directives that cannot be preserved, or arguments that cannot become keyed fields safely. Partial grouping may be a later milestone only with explicit evaluation-order preservation proof. No new-file creation.

### Audited starting state

The read-only authoritative audit tree is `/Users/alessandro/sources/semantic-editor` at `6c95d12`, with shared uncommitted files/buffers included. These six operations were absent across worktrees. Existing signature-related primitives do not implement migration to a parameter object. Declaration/field/type helpers, reference resolution, registry-backed CLI/MCP operations, one-process gopls sessions, `sourcebuf`, and `Buffer.Apply` exist in the audited source. Current transport does not establish support for command execution, interaction, or applyEdit. ADR-0060 remains a pending diagnostic-session record in that tree, not a decision for this operation.

### Candidate implementation paths

* Semantic resolution: `internal/adapters/golang/gopls.go`, `internal/lsp/client.go`, `internal/symbol/resolver.go`, and `internal/astedit/type.go`; enumerate all direct callers and body bindings from the shared overlay.
* AST planning: a focused `internal/astedit/introduce_parameter_object.go`, using `function.go`, `decl.go`, `insert.go`, `comment_anchor.go`, and `move_construct.go` for declaration/source-range patterns. Keep this transformation distinct from single-parameter introduction.
* Wire and schemas: dedicated `internal/operation/wire_parameter_object.go` (or signature-focused file), `operation/operation.go`, `parse.go`, and MCP schema generation. Nested `fields` structures require closed recursive schemas and presence-aware CLI JSON; do not represent them as JSON strings.
* Buffering: `internal/sourcebuf/buffer.go`, `internal/mcp/buffers.go`, and `internal/mcp/batch.go`; produce the struct, changed declaration, and every call-site edit as a complete `Change` set and invoke `Buffer.Apply` once.

### Proposed public contract

Proposed MCP `semantic_introduce_parameter_object`, CLI `introduce-parameter-object`, Go-only. `symbol` is the function/method target. `destination_file` names where the new struct is declared; do not use it as the source path. `in_function` is not appropriate for this declaration-level target. The accepted all-parameters form maps every original parameter by zero-based index into a generated struct field; mappings are explicit and revision-bound.

| Input | Role | Type | Default | Format / use |
| :--- | :--- | :--- | :--- | :--- |
| `file` | target file | string | none | Existing file containing function/method. |
| `symbol` | target | string | none | Qualified function or method. |
| `destination_file` | destination target | string | `file` | Must equal `file` in release one; a separate existing same-package file is milestone two. New files require a later resource scope decision and prior scaffold. |
| `parameter` | new callable parameter | closed `ParameterSpec` | none | `{name: string, type: string, variadic?: boolean}` for the sole object argument; `type` is the new package-level struct name. |
| `fields` | ordered field mappings | array of closed `ParameterToFieldMapping` objects | none | Each entry is `{source_index: integer, field: FieldSpec{name: string, type: string, tag?: string}}`; `source_index` identifies an original function parameter. Array order determines struct field order. Every original parameter appears exactly once; field type must preserve the selected parameter type in release one. |
| `expected_revision` | concurrency target | string | omit for a fresh target | Revision token for the selected declaration `file` only; body, call set, and destination are captured and validated separately. |

Validate `parameter` and field names plus source indices structurally; require unique, complete indices and preserve each source parameter type. Do not accept arbitrary source fragments. Return the function and created struct as separate `continuations` entries with `file`, `symbol`, and `expected_revision`. Do not overload `symbol` or invent a `generated` object. Historical/planned names are not live continuations.

### Decision gates and semantic risks

1. Structured map types are not supported by today's registry contract. RQ-0024 must add typed recursive object/array schemas, presence-aware CLI JSON parsing, and `SourceFields` name-and-type checks before this tool's `fields` array is exposed.
2. Determine parameter grouping and Go call evaluation order. Release one requires all parameters and a keyed literal as the single argument; preserve field order to match original left-to-right argument evaluation. Partial grouping requires temporary statements or a separate proof.
3. Rewrite references only when they bind to the target parameter; protect shadowed locals, closure captures, labels, comments, and docs. Type-check all rewritten uses.
4. Prove complete caller references in the same module and reject function values, interface dispatch, unknown external callers, generated/build-specific files, and exported APIs until supported.
5. Define field-export policy and API visibility. A private function may still need exported fields if object crosses package boundaries; first release should keep type and function package-private.
6. Gopls code actions remain candidates. Current client lacks established command, interactive, and applyEdit support; block that implementation route until transport tests pass. Never use gopls CLI.
7. Validate all preimages/revisions and normalize before computing live receipt continuations; never return stale call paths.

### Ordered implementation handover

1. Resolve the shared schema prerequisite and cite its decision in the eventual implementation change. Build typed call/reference inventory for private function scope.
2. Prototype an all-parameters conversion for one function and straight-line callers. Compare type-checked before/after behavior; assert keyed literal evaluation order explicitly.
3. Define object/field receipt shape, CLI nested JSON syntax, and source-field name/type checks. Add registry validation before exposing the operation.
4. Produce all AST changes in memory: struct declaration, parameter list, body reference rewrites, and every caller literal. Preserve comments and all bytes outside planned edits.
5. Validate destination package/name, all references, scopes, types, expected revision, and every output. Call `Buffer.Apply` once after successful complete preflight.
6. Register with `Batchable: true` and `BufferedBatch: true` for existing Go files. Run inside the batch's one sourcebuf overlay and one gopls session; coalesce verification/import formatting. Earlier successful buffered edits remain when later edits fail; no whole-batch rollback is promised.
7. Add CLI/MCP docs, capability metadata, and txtar cases. Require explicit `semantic_scaffold_file` before targeting any newly created file; sourcebuf new-resource semantics remain undefined.

### Bounded release and expansion milestones

* **Milestone 1:** all parameters of one private function in one package; same-file struct; direct typed calls only; keyed literal preserves evaluation order.
* **Milestone 2:** destination in an existing same-package file; exported types/functions only after external API/reference completeness is demonstrated.
* **Milestone 3:** partial grouping only if the planner preserves argument evaluation order across interleaved retained parameters and proves behavior for nested expressions.
* **Milestone 4:** methods, generic functions, interface dispatch, multiple packages, and newly scaffolded resources only after each has explicit capability metadata and independent preflight/publication evidence.

### Required evidence and tests

Use public CLI txtar operations and MCP `semantic_batch`; assert tool output/diagnostics, exact diff, final workspace, and valid JSON matching the proposed `ParameterMapping` schema. Keep per-operation refusal cases fail-closed.

* Positive: transform a private `write(path, data string, mode uint32)` into `write(options writeOptions)`; assert struct and keyed call literal, field reference rewrites, compile/type result, and follow-up lookup/edit using surviving receipt fields.
* Refusals: partial field selection in milestone 1, unknown call site, method value/interface dispatch, duplicate struct/field name, generic function, stale revision, invalid destination package, body shadowing/capture, and side-effecting calls that violate order. Assert no file bytes change.
* All-output preflight: separate destination file plus multiple caller files; make the last caller stale/untransformable and prove neither destination, declaration, body, nor earlier caller is changed.
* Buffered sequence: earlier unsaved call insertion → introduce parameter object → later mutation resolves and uses the new type/field names. A standalone follow-up can consume the function/type continuations. Prove later calls see overlay, disk remains unchanged until flush, final normalization/verification run once, and failure later in the batch retains earlier successful buffered operations.
* Retained server: verify one-process gopls identity across prior edit, complete call/type analysis, and a subsequent resolver probe. No gopls CLI evidence.
* Byte/no-op: preserve function comments, call comments, line endings, imports, and unrelated bytes. If the function already takes the requested object and all call literals match, return an empty diff with no write; never report a duplicate live declaration.

### Completion decision

Keep open until schema support, all-caller proof, and evaluation-order semantics are resolved and the public batch evidence passes. Do not widen from all-parameter migration to arbitrary partial grouping based only on successful type checking.

### Concrete fixture sketches

Add `testdata/scripts/introduce_parameter_object_go.txtar` and `testdata/scripts/mcp_batch_introduce_parameter_object.txtar`. Use a private three-parameter function, two existing callers across existing same-module files, and one callback-like refusal. Release one keeps the destination equal to source; separate existing-file destinations are a later milestone.

Create an earlier unsaved call, perform the conversion, then run a later mutation with statically known `file`/`symbol` targets so the registry resolves the new signature and struct. Consume returned `continuations` in a separate follow-up call; current `semantic_batch` has no receipt interpolation or read-only batching. Omit `expected_revision` for these fresh overlay targets:

```json
{
  "edits": [
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "caller.go",
        "source": "func Run() error { return write(\"a\", \"b\", 7) }"
      }
    },
    {
      "tool": "semantic_introduce_parameter_object",
      "params": {
        "file": "write.go",
        "symbol": "write",
        "destination_file": "write.go",
        "parameter": {
          "name": "options",
          "type": "writeOptions"
        },
        "fields": [
          {
            "source_index": 0,
            "field": {
              "name": "path",
              "type": "string"
            }
          },
          {
            "source_index": 1,
            "field": {
              "name": "data",
              "type": "string"
            }
          },
          {
            "source_index": 2,
            "field": {
              "name": "mode",
              "type": "uint32"
            }
          }
        ]
      }
    },
    {
      "tool": "semantic_replace_body",
      "params": {
        "file": "write.go",
        "symbol": "write",
        "body": "return save(options.path, options.data, options.mode)"
      }
    }
  ]
}
```

The fixture must declare matching source parameter types and helper signatures. Assert generated struct fields, `write(options writeOptions)`, field-bound body uses, and keyed `writeOptions{path: ..., data: ..., mode: ...}` call. The standalone CLI txtar invokes `semedit introduce-parameter-object ...` and compares every affected file. Assert the result uses two `continuations` entries, one for `write` and one for `writeOptions`, with real final revisions.

Refusal fixtures include partial fields in milestone one, side-effecting arguments that would be reordered, closure capture/shadowing, interface dispatch, unknown caller, stale revision, duplicate object/field names, and a destination with another package. For multi-file outputs make the last caller stale and assert no output changes.

The buffer test should prove the full struct/declaration/body/call set is validated before one `Buffer.Apply`; a separate stale-preimage case changes a captured input before `Buffer.Apply`/`Flush`, asserts `sourcebuf.ErrConflict`, and proves no planned output was published; prior disk bytes remain visible until final flush while later mutations resolve the overlay. Read-only `semantic_lookup` is not part of `semantic_batch` because its registry entry is not batchable. Assert same gopls process/session, coalesced verification/import normalization, and successful earlier operations retained if a later batch entry fails. Add an explicit test that final publication is per-file atomic and does not claim whole-batch rollback.

### Expansion milestone and artifact checklist

Milestone two may allow a separate existing same-package destination file after publication preflight is proven; milestone three may evaluate partial grouping, external callers, or new-file destinations independently. Capability metadata must say when generic functions, methods, and interface dispatch remain unsupported. The typed receipt must make the surviving function target and any nested new-object continuation unambiguous.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_introduce_parameter_object`:**

> Use this tool instead of reconstructing signatures, fields, and call arguments to group all parameters of one supported private Go free function into a named struct. Supply the sole object `parameter` and ordered `fields` mappings. Release one creates the struct in the source file and updates every proven direct caller while preserving argument order. Pass the callable continuation's `file` and `symbol` to `semantic_replace_body`, or the created type continuation to `semantic_inspect_symbol` or `semantic_rename`. Choose the appropriate complete target record.

**Parameter description requirements:**

`symbol`: Target: existing callable copied from `semantic_lookup.symbol` with its `file`. `parameter`: Replacement: caller-chosen object parameter name and new struct type. `fields`: Replacement: ordered `ParameterToFieldMapping` records; `source_index` refers to the original signature, not the final field ordinal. `destination_file`: Destination selector: defaults to `file` and must equal it in release one. Generated qualified type names never feed the new simple-name input implicitly.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go extract type and delegate

Original handover: RQ-0047. Implementation status: Open.

### Outcome and current evidence

Separate selected state and behavior into a new Go type without requiring the caller to reproduce existing members. Provide distinct `semantic_extract_type` / `extract-type` and `semantic_extract_delegate` / `extract-delegate` handovers in this shared investigation: they share member identity and layout proofs but have different call-rewrite policies.

No extraction-of-type or delegate operation is registered on main or in audited outstanding worktrees. `semantic_insert_type` can insert a supplied declaration; `semantic_insert_function` already inserts a supplied receiver method. Neither computes member relocation, object composition, or forwarding calls.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this tool handover records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

### Bounded delivery and non-goals

First milestone: extract an explicitly selected method-only responsibility from an unexported, non-generic receiver into an existing destination file in the same package. No state is relocated in this milestone, but the new composition field still changes layout and potentially comparability/reflection. Require explicit private-type use analysis for those effects. Type extraction updates selected method call sites to the new component; delegate extraction retains wrappers on the original receiver. Require an explicit instance field name and preserve single receiver evaluation.

Second milestone: private named fields with keyed literals only after proving object identity, zero values, comparability and initialization order. Defer exported structs, positional literals, embedding, reflection/serialization-sensitive types, generic receivers, cross-package relocation and implicit choice of member groups. Do not claim these are safe because the result compiles.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

### Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Existing file of source type; file selector | Required |
| `symbol` | Target: new simple type name, e.g. `readerState` | Required |
| `in_type` | Selector: existing qualified source type `reader` | Required |
| `members` | Typed array of selected `{file, symbol}` member identities | Required, nonempty |
| `destination_file` | Existing destination Go file in the same package | Defaults to `file` |
| `field` | New composition field identifier on original type | Required |
| `access_modifier` | Existing visibility convention for generated type | Defaults to private |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_extract_delegate",
  "params": {
    "file": "reader.go",
    "symbol": "readerState",
    "in_type": "reader",
    "members": [
      {
        "file": "reader.go",
        "symbol": "(*reader).read"
      }
    ],
    "destination_file": "state.go",
    "field": "state"
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "state.go",
  "symbol": "readerState",
  "in_type": "reader",
  "destination_file": "state.go",
  "continuations": [
    {
      "file": "reader.go",
      "symbol": "reader.state"
    },
    {
      "file": "reader.go",
      "symbol": "(*reader).read"
    },
    {
      "file": "state.go",
      "symbol": "(*readerState).read"
    }
  ]
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

### Implementation sequence

1. Resolve one source type and exact member objects; inventory all receiver uses, literals, method values and escapes. Define how inspection emits `in_type` and typed `members` without model reconstruction.
2. Write distinct type-extraction and delegate-extraction contracts: the former redirects usages, the latter keeps forwarding methods and interface method sets. Require a closed `members` schema before registration.
3. Build a shared member/dependency planner. Validate non-selected dependencies and receiver aliasing; refuse when behavior relies on original receiver state or identity.
4. Plan the new type, composition field, selected methods, initialization and call changes together. Preserve attached docs/tags and untouched siblings.
5. Check pointer/value method sets, import bindings and complete overlay result; register both operations with the existing coordinator and atomic multi-output Apply.
6. Publish explicit different capability descriptions for each mode; expand to selected fields only after layout/identity proof tests pass.

### Required batch behavior

Batch proof for each operation: insert a method into the source type → extract it to the already scaffolded destination → rename the generated method → replace its body. Inspect the new receiver and old wrapper from the working overlay. A disk probe before final publication must still see the original files. Final receipts identify both generated type and surviving callable locations. Repeat with an earlier unsaved call-site change and with a refused second extraction.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

### Public acceptance and refusal coverage

* `extract_type_method_only.txtar`: two private methods, selected one redirected through a new component; unrelated methods/comments unchanged. Assert behavior and resulting symbol locations.
* `extract_delegate_method_only.txtar`: selected method wrapper keeps original interface satisfaction and evaluates the receiver once. Assert pointer and zero-value behavior.
* Refuse field identity dependencies, method values/expressions without a proven rewrite, positional literals, embedding, inaccessible symbols, generated files and package cycles before writing.
* Compare an explicit state mutation counter and nil receiver outcome before/after; a compile-only test is insufficient.
* Failure in destination planning leaves source, destination, and callers byte-identical.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

### Semantic risks and unresolved decisions

* Moving fields changes address identity, struct layout, tags, promoted methods, equality and exported API. A composition field itself can change comparability or reflection; restrict first delivery to private types with no such observed uses.
* Extract type and extract delegate must not silently share the same wrapper policy. Unchanged dispatch and interface method sets require explicit evidence.
* Nil receivers, receiver-copy behavior and forwarding evaluation are semantic constraints.
* The proposed `field` name must align with field-insertion schemas; never name the owner `symbol` while the new type is the target.

### Decision gates and completion

Before implementation, choose the method-only receiver representation and prove that creation/zero-value initialization preserves the selected scope. If this cannot be proved, narrow to a pure stateless helper responsibility and document it. Accept field support separately after layout, alias and serialization evidence. Do not implement a broad extraction through text replacement.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this tool handover Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

### Dependencies, ownership, and entry points

Depends on typed schemas/live member receipts and [field insertion](RQ-0060-go-refactoring-tool-implementation.md#go-insert-field). Contract implementation and interface extraction can develop in parallel. Share method-set analysis with [implement contract](RQ-0060-go-refactoring-tool-implementation.md#go-implement-contract); do not merge overlapping registry/index changes concurrently.

* Read `internal/astedit/type.go`, `function.go`, `move_construct.go` and `move_cross_package.go`.
* Extend `internal/operation/wire_engine_declarations.go` through new typed operation definitions; inspect `internal/backend/golang/backend.go`.
* Integrate pending `internal/sourcebuf/buffer.go`, `internal/godiagnostics/manager.go`, and `internal/mcp/buffers.go` after their owning changes are safely available.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_type / semantic_extract_delegate`:**

> Use `semantic_extract_type` instead of reconstructing selected Go members when extracting a supported private method responsibility into a new composed type and redirecting its selected callers. Use `semantic_extract_delegate` for the same bounded extraction when existing methods must remain as forwarding wrappers. Choose the new type `symbol` and composition `field`; select the existing owner with `in_type` and exact `members`. Pass a generated type/member continuation's complete `file` and `symbol` to `semantic_inspect_symbol` or `semantic_rename`; pass only callable continuations to `semantic_replace_body`. Unsupported state, layout, or identity effects refuse.

**Parameter description requirements:**

`symbol`: Target: fresh simple generated type name; no existing qualified owner symbol supplies it. `in_type` and `members`: Selectors: proposed `semantic_inspect_symbol.in_type` and `semantic_inspect_symbol.members` from the typed member inspection, gated on the shared schema extension. `field`: Replacement: new simple composition field identifier. Keep it separate from a returned qualified member `symbol`; never strip a receiver/type qualification to reuse it.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go encapsulate field

Original handover: RQ-0048. Implementation status: Open.

### Outcome and current evidence

Generate accessors for one selected field and rewrite resolved reads/writes across the permitted module scope through `semantic_encapsulate_field` / `encapsulate-field`, preserving evaluation count and receiver identity.

Main supports field rename via gopls, but no accessor generation or reference-aware field encapsulation operation. Supplied methods can already be inserted; this plan is for coordinated generation and use rewriting.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this tool handover records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

### Bounded delivery and non-goals

First release targets one unexported named field of a private non-generic struct with ordinary reads and standalone assignments. Getter and setter names are explicit; both receiver and RHS must be evaluated once. Preserve field storage and struct layout. Refuse compound assignments, address-taking, positional/unkeyed literals, embedding, external clients, interface-dependent accessors and reflection-sensitive use.

Later release can lower compound assignments with temporaries after proving evaluation and panic order. Changing field visibility or removing storage is a separate accepted policy.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

### Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Selector: existing field declaration file | Required |
| `symbol` | Target: qualified field, e.g. `config.timeout` | Required |
| `getter` | Target name for generated getter method | Required |
| `setter` | Target name for generated setter method | Required |
| `destination_file` | Existing same-package file for generated accessors | Defaults to `file` |
| `access_modifier` | Accessor visibility using existing values | Defaults to private |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_encapsulate_field",
  "params": {
    "file": "config.go",
    "symbol": "config.timeout",
    "getter": "Timeout",
    "setter": "SetTimeout",
    "destination_file": "accessors.go",
    "access_modifier": "public"
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "config.go",
  "symbol": "config.timeout",
  "destination_file": "accessors.go",
  "continuations": [
    {
      "file": "accessors.go",
      "symbol": "(*config).Timeout"
    },
    {
      "file": "accessors.go",
      "symbol": "(*config).SetTimeout"
    }
  ]
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

### Implementation sequence

1. Resolve exact field object and enumerate reads, writes, literal initialization, address uses, reflection/unsafe indicators and receiver forms from current typed overlay.
2. Specify getter/setter receiver policy and zero/nil semantics; reserve names with receiver method-set collision checks.
3. Generate accessor declarations using existing insertion planning primitives, without applying them independently.
4. Rewrite typed field usages only. Plan receiver/RHS temporaries where proven necessary; reject any use shape outside the initial contract. Initial field storage and literal initialization remain unchanged.
5. Validate the combined accessors and rewritten callers with bound input snapshots and overlay typechecking; apply once.
6. Register CLI/MCP and capability metadata, then prove live accessor continuation chains into rename/body edits.

### Required batch behavior

Batch: insert an unsaved field using the field-insertion operation → encapsulate it → rename the getter → replace the setter body. An earlier unsaved caller must be rewritten and later resolution must locate generated methods. A refused conflicting getter must preserve the earlier inserted field only. Add same-module references outside an MCP starting subdirectory to final-diff coverage.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

### Public acceptance and refusal coverage

* Getter read and setter assignment in another file with exact unrelated field/type byte comparisons.
* Side-effect counter fixture checks receiver evaluation before RHS and once-only calls; nil-pointer evaluation/panic timing unchanged.
* Refuse address-taking, `x.f++`, compound assignment, method-set name collision, promoted ambiguous field and positional literals before writes.
* References to another struct with the same field spelling remain unchanged.
* Default and disabled imports produce the expected accessor/import result without touching unrelated build headers.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

### Semantic risks and unresolved decisions

* Getters can change nil behavior, allocation, receiver copying or method dispatch. Setters can reorder receiver/RHS evaluation, and compound statements can evaluate the receiver more than once.
* Struct literal initialization and field access have distinct contracts; do not reinterpret literal keys as calls.
* Encapsulation cannot safely promise external API migration without a complete external consumer set.
* Generated method names in `continuations` are live symbol targets; `getter`/`setter` identifier strings are configuration, not substitutes for qualified callable continuations.

### Decision gates and completion

Choose pointer/value accessor semantics explicitly and prove evaluation/panic order for accepted read/write forms. Reject unsupported reference forms instead of silently leaving them unrewritten. Record the accepted transformation/visibility invariant before expanding scope.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this tool handover Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

### Dependencies, ownership, and entry points

Depends on [field insertion](RQ-0060-go-refactoring-tool-implementation.md#go-insert-field) for shared collision/member planning and on typed reference/read-set foundations. Independent of general change signature. Share alias/member-set proof utilities with extraction; integrate generated accessor output contracts once.

* Inspect `internal/symbol/resolver.go`, `internal/backend/golang/backend.go`, `internal/astedit/type.go` and the field rename path in pending `internal/adapters/golang/rename_edit.go`.
* Add a focused AST planner and typed registry wire file; use pending `sourcebuf.Buffer.Apply` and existing pipeline result conventions.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_encapsulate_field`:**

> Use this tool instead of generating accessors and replacing field references separately to encapsulate one supported private Go field. Supply its qualified `symbol` and explicit `getter` and `setter` names. The engine preserves field storage and rewrites only proven reads and standalone writes with once-only receiver and value evaluation. Pass the surviving field's `file` and `symbol` to `semantic_inspect_symbol` or `semantic_rename`; pass each generated accessor continuation's complete `file` and `symbol` to `semantic_replace_body` or `semantic_rename`.

**Parameter description requirements:**

`symbol`: Target: qualified existing field; copy `semantic_lookup.symbol` or proposed `semantic_insert_field.symbol` only when it identifies this exact live field, with that producer's matching `file`. `getter` and `setter`: Replacement: fresh simple method names chosen by the caller; generated qualified accessor symbols do not feed these inputs. `destination_file`: Destination selector: existing same-package accessor file, default `file`; it is separate from each accessor's live result `file`.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go type migration

Original handover: RQ-0049. Implementation status: Open.

### Outcome and current evidence

Provide `semantic_migrate_type` / `migrate-type` for an explicitly selected type use or declaration and a permitted graph of dependent uses. The operation must distinguish exact equivalence from an intentional compatibility change.

No type-migration operation exists in audited code. Rename changes identity spelling; replacing a declaration does not propagate conversions through callers, assignments or interface conformance. Signature changes in RQ-0041 cover callable boundaries, not a general type-use graph.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this tool handover records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

### Bounded delivery and non-goals

First release is behavior-preserving alias normalization: migrate a selected unexported declaration/field from an alias to its identical aliased type, or the reverse only where exact type identity is proved. Do not guess conversions. Accept scope within one enclosing module and existing regular Go files.

Later gates cover distinct named types, explicit conversion rules and function-result/parameter changes. Defer numeric narrowing, representation changes, map-key comparability changes, unsafe layout, serialization-sensitive exports and generic inference until separate semantics are proved. This narrower first release is intentional; it does not advertise arbitrary type migration.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

### Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Existing declaration file selector | Required |
| `symbol` | Target declaration or qualified field whose type changes | Required |
| `type` | Desired Go type expression resolved in target declaration scope | Required |
| `in_function` | Owner selector only for supported local targets | Optional; first release excludes locals |
| `scope` | Permitted propagation boundary | Defaults to module; only module initially |
| `conversions` | Future closed typed conversion rules, never unchecked source substitution | Omit in first release; reject supplied value |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_migrate_type",
  "params": {
    "file": "config.go",
    "symbol": "config.timeout",
    "type": "time.Duration",
    "scope": "module"
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "config.go",
  "symbol": "config.timeout",
  "type": "time.Duration",
  "expected_revision": "<post-edit-file-revision>"
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

### Implementation sequence

1. Define target kinds and alias-identity evidence; resolver must return both type object and declaration position from current overlay.
2. Determine affected type-use/assignment/interface graph with exact object identities. Keep unrelated identical spellings untouched.
3. Decide which declarations change and which already remain assignable; first release admits only exact identity, including contextual imported type name resolution.
4. Bind module/local replacement inputs; plan minimal type spelling and imports across all outputs. Conversion maps cannot bypass missing reference coverage.
5. Validate alias cycles, method-set identity, generic constraints, nil/comparability and full merged-overlay results before applying.
6. Register first scope truthfully. Add distinct-type migration only with explicit conversion schema shared with signature transformations and observable compatibility checks.

### Required batch behavior

Batch: insert an alias without publication → migrate an existing private field to it → rename the alias → edit a downstream callable using that type. All resolution must use the working revision, including sibling files outside the MCP starting directory. Refusal of a distinct-type conversion must retain only earlier successful mutations and leave no attempted output bytes.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

### Public acceptance and refusal coverage

* Alias-to-original and reverse identity fixtures with value behavior, method sets, interface assignments, nil and map-key use unchanged.
* Same-spelled type in another package remains unchanged; imported type alias resolves correctly when package name differs from directory.
* Refuse distinct numeric/struct types, overflow, changed comparability, inaccessible desired type, alias cycle, exported external-consumer uncertainty, unsupported conversion rules and non-Go files before writes.
* Record a diagnostic introduced by an unrelated earlier batch child without claiming the entire workspace was clean.
* No-op identical type produces empty final diff and preserves headers exactly.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

### Semantic risks and unresolved decisions

* Assignability is weaker than identity and behavior preservation. Explicit conversions can truncate, allocate, change methods or alter observable interface values.
* Aliases and new defined types have different identity/method/generic semantics.
* Desired `type` text must resolve separately at each edited file with stable imports; copying aliases blindly is unsafe.
* Complete exported external consumers cannot normally be discovered from the current module. The first contract must refuse that promise.

### Decision gates and completion

Choose one unambiguous target-kind vocabulary and verify exact alias identity through the compiler. Define an explicit compatibility policy before accepting distinct named types or conversions. If a migration needs signature updates, call a shared planner rather than invoking independently publishing child operations.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this tool handover Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

### Dependencies, ownership, and entry points

Depends on structured schemas, typed whole-plan references, and [change signature](RQ-0060-go-refactoring-tool-implementation.md#go-change-signature) for future callable transformations. Develop the read-only identity graph after common foundations; schedule broad migration after signature, member and interface proof utilities stabilize.

* Inspect `internal/astedit/type.go`, `decl.go`, `replace_decl.go`, and compiler-overlay loading in `move_cross_package.go`.
* Integrate `internal/operation`, `internal/backend/golang/backend.go`, pending `sourcebuf` and retained session manager; keep conversion planning in a language backend, not transport.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_migrate_type`:**

> Use this tool instead of changing type annotations and conversions by text to migrate one supported Go declaration to an identity-equivalent type spelling. Supply the existing qualified `symbol` and desired `type`. Initial support is limited to proven alias identity; distinct-type conversions and broader compatibility changes refuse. Pass the surviving declaration's exact `file`, `symbol`, and resolved `type` to a later `semantic_migrate_type` request, or its `file` and `symbol` to `semantic_inspect_symbol` or `semantic_rename`. An identical result is a no-op.

**Parameter description requirements:**

`symbol`: Target: existing declaration, copied from a compatible `semantic_lookup.symbol` with its `file`. `type`: Replacement: desired Go type expression in the target scope; a previous `semantic_migrate_type.type` is copyable only in that same scope with unchanged import bindings. `scope`: Selector: module boundary, default module; it does not authorize writes to read-only replacement modules. Reject unsupported `conversions` in release one.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go implement contract

Original handover: RQ-0050. Implementation status: Open.

### Outcome and current evidence

Generate the missing receiver methods for an existing Go interface through `semantic_implement_contract` / `implement-contract`, without requiring the model to reproduce each signature.

The capability is proposed in RQ-0024 but absent from all audited implementations. Existing method insertion accepts complete supplied source. gopls documents an implement-interface action, but code-action command/dialog/edit handling is not automatically available through the current rename-only edit contract.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this tool handover records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

### Bounded delivery and non-goals

First release handles one non-generic concrete named type and one non-generic ordinary interface, including interface embedding after resolved method-set flattening. Generate only genuinely missing methods into an existing same-package file. Pointer versus value receiver is explicit and checked against method sets. A same-named method with incompatible signature is a refusal, not a missing method.

Generated panic stubs are editing assistance, not functional implementations. The request explicitly selects a stub policy; no inferred business logic or silent zero-value bodies. Defer generic constraints, inaccessible external unexported methods, ambiguous promoted methods and delegated implementations.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

### Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Existing concrete type declaration file | Required |
| `symbol` | Target concrete type `Worker` | Required |
| `contract` | Selector: qualified interface name resolved from `contract_file` | Required |
| `contract_file` | Existing declaration file for the interface; can be read-only | Required |
| `destination_file` | Existing same-package file for generated methods | Defaults to `file` |
| `receiver_kind` | Receiver convention `pointer` or `value` | Required |
| `body_policy` | Generated implementation policy; `panic_stub` initially | Required |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_implement_contract",
  "params": {
    "file": "worker.go",
    "symbol": "Worker",
    "contract": "Runner",
    "contract_file": "runner.go",
    "destination_file": "worker_methods.go",
    "receiver_kind": "pointer",
    "body_policy": "panic_stub"
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "worker.go",
  "symbol": "Worker",
  "contract": "Runner",
  "contract_file": "runner.go",
  "destination_file": "worker_methods.go",
  "continuations": [
    {
      "file": "worker_methods.go",
      "symbol": "(*Worker).Run"
    }
  ]
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

### Implementation sequence

1. Inspect exact concrete/interface objects and compute pointer/value method sets from current overlay. Resolve embedded interface methods and duplicate/conflicting signatures.
2. Compare documented gopls action behavior with the required receiver, body and source-preservation contract. If command/dialog/applyEdit support is necessary, implement a scoped capture/validation gate under a separate accepted transport decision; do not invoke the CLI or let a server publish.
3. Prefer deterministic signature-driven generation when the action cannot satisfy the contract. Preserve parameter/result types, variadics and cross-package import bindings; choose collision-free local names only where source names are absent.
4. Build all missing methods in one plan, preserving existing implementations byte-for-byte. Check inaccessible methods, receiver conflicts and import collisions before Apply.
5. Prove selected concrete pointer/value type satisfies the interface in the resulting typed overlay. That proof validates signatures only; receipt must state that generated bodies are stubs.
6. Register typed contract selectors and generated callable continuations; expose Go support and empty-plan no-op behavior.

### Required batch behavior

Batch: insert an interface and concrete type into existing files → implement contract → rename one generated method through gopls → replace its stub body. The latter edits must resolve unsaved generated declarations. Also change the interface signature in an earlier child and verify generated methods match the revised interface, not disk. Refused conflicting method generation leaves earlier edits intact and contributes no half-generated method set.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

### Public acceptance and refusal coverage

* Generate two missing methods and retain one existing method/body/comment exactly; assertion demonstrates selected receiver method-set conformance.
* Embedded interface, imported parameter types, variadic signatures and multiple results retain their declarations and compile.
* Pointer versus value receiver acceptance/refusal, conflicting existing signature, inaccessible external method and promoted ambiguity cause zero attempted edits.
* Existing complete implementation is a no-op with no formatting of untouched files.
* Invoke the generated panic stub and recover its explicit placeholder panic; never label the body behavior implemented.
* Protocol test captures/refuses unsolicited or unsupported code-action resource edits if a native action route is chosen.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

### Semantic risks and unresolved decisions

* Structural name matching does not establish interface satisfaction: signatures, package identity of unexported methods and pointer/value method sets matter.
* Code-action availability is not a promise of safe client behavior; standard actions may execute commands and interactive actions may need dialogs.
* A panic stub deliberately changes behavior when called; this operation is code generation with visible placeholder policy, not an equivalence guarantee.
* Canonical `contract`/`contract_file` selectors and `receiver_kind` vocabulary need a registry/schema decision; do not create inconsistent `interface`, `trait` and `contract` fields per backend.

### Decision gates and completion

Choose a deterministic stub-body policy and how callers inspect it before exposing the operation. Resolve native-action transport requirements or use the AST generation route. Validate the same requested pointer/value type against the interface rather than a superficially passing assertion on a different type.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this tool handover Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

### Dependencies, ownership, and entry points

Can develop alongside [extract interface](RQ-0060-go-refactoring-tool-implementation.md#go-extract-interface) after common schemas/session foundations. Share method-set comparison and generated declaration receipt logic. Do not add a duplicate generic method-insertion operation; [field insertion](RQ-0060-go-refactoring-tool-implementation.md#go-insert-field) is a distinct gap.

* Read `internal/astedit/function.go`, `type.go`, `internal/operation/wire_engine_declarations.go`, `internal/backend/golang/backend.go` and `internal/lsp/client.go`.
* Reuse pending `internal/adapters/golang/rename_edit.go` validation concepts, `sourcebuf.Buffer.Apply`, and session synchronization.
* Primary native-action source: [gopls transformation catalog](https://go.dev/gopls/features/transformation).

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_implement_contract`:**

> Use this tool instead of reconstructing interface signatures to generate missing Go methods for a concrete type. `symbol` targets the concrete type; `contract` and `contract_file` select the interface. Choose `receiver_kind` and the explicit `body_policy`; initial `panic_stub` methods panic when called and are not completed implementations. Pass each generated method continuation's exact `file` and `symbol` to `semantic_replace_body` or `semantic_rename`, and the concrete type target to `semantic_inspect_symbol`. Existing implemented methods remain intact.

**Parameter description requirements:**

`symbol`: Target: concrete type copied from a compatible `semantic_lookup.symbol` with its `file`. `contract` and `contract_file`: Selectors: interface identity and declaration file; proposed `semantic_inspect_symbol.contract` and `semantic_inspect_symbol.contract_file` must emit that exact interface context before advertising a copy edge. `receiver_kind`: Policy: required pointer/value choice; descriptive lookup `kind` does not feed it. `body_policy`: Policy: required `panic_stub`; do not imply a useful runtime implementation.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go extract function expansion

Original handover: RQ-0051. Implementation status: Open.

### Outcome and current evidence

Extend existing `semantic_extract_function` / `extract-function` to routine helper extraction with real local data flow, supported nested control flow, multiple results and attached comments. Do not create another extract-method API: existing `kind=function|method` selects that behavior.

Extraction is ALREADY implemented on main by `cd40dea` and recorded in ADR-0056. It supports contiguous direct calls, optionally one supported leading returned local. The clean extraction worker has no unmerged broader implementation. Pending main changes integrate the same bounded operation with sourcebuf; they do not remove its semantic restrictions.

The audit baseline is committed main `6c95d12` plus its pending shared-buffer/session migration. Recheck that baseline before implementation; this tool handover records a proposed contract, not an implemented capability or an accepted ADR. Follow the [shared handover contract](RQ-0060-go-refactoring-tool-implementation.md#shared-handover-contract) and [receipt/schema investigation](RQ-0027-preventing-read-files.md#refactoring-handover-foundations).

### Bounded delivery and non-goals

Milestone 1: straight-line assignments/declarations and expressions, compiler-derived live-ins/live-outs, multiple supported result values, and comment preservation. Milestone 2: self-contained loops and branches whose exits remain inside the extracted region. Milestone 3: expression/nested-block selection, generic/variadic/named-result owners only after dedicated proofs.

Refuse outward returns, break/continue/goto, direct recover, defer/go lifetime changes, receiver-copy/alias changes, captured/address-escaping state, ambiguous comment ownership and unknown types until supported. No arbitrary full-function rewriting by the model.

Go is the first backend. Do not advertise another language until its capability metadata and comparable public CLI cases exist. Existing Go files must work in both standalone calls and buffered batches from the first release. File creation is a separate explicit scaffold step until source-buffer resource operations have their own accepted contract.

### Proposed public contract

The tool and command names below are candidates. Preserve ADR-0055 target/selector roles and validate every proposed input through the typed operation registry.

| Parameter | Role and value | Required/default |
| :--- | :--- | :--- |
| `file` | Selector: existing owner source file | Required |
| `symbol` | Target: new helper simple identifier | Required |
| `in_function` | Selector: existing qualified owner | Required |
| `kind` | Existing `function` or `method` helper kind | Required; no implicit default |
| `start_line`, `end_line` | Existing inclusive direct-statement boundary selectors | Retain current required behavior |
| `discriminator`, `construct_path` | Candidate revision-bound structural range selectors | Optional future alternative; require shared selection gate |

`auto_organize_imports` follows the existing boolean policy; `expected_revision` is the common optional selector precondition under investigation. Omitted and explicit empty structured values must remain distinct. Unknown fields and unsupported schema kinds must fail explicitly.

Proposed request:

```json
{
  "tool": "semantic_extract_function",
  "params": {
    "file": "service.go",
    "symbol": "prepareRequest",
    "in_function": "(*Service).Serve",
    "kind": "method",
    "start_line": 14,
    "end_line": 19
  }
}
```

Proposed live-target receipt projection (diagnostic and diff fields use existing result conventions):

```json
{
  "file": "service.go",
  "symbol": "(*Service).prepareRequest",
  "in_function": "(*Service).Serve",
  "kind": "method",
  "continuations": [
    {
      "file": "service.go",
      "symbol": "(*Service).prepareRequest"
    }
  ]
}
```

Generate live targets from the resulting source, rather than rebuilding qualified names from input strings. Apply `SourceFields` to identical input/output names and compatible typed values. Secondary generated declarations need named typed continuations, not positional pairs or a concatenated name. A later batch child can invalidate an earlier child's target: child evidence records its working revision, while final continuations must be re-resolved or omitted.

### Implementation sequence

1. Preserve every existing success/refusal txtar and identify the exact restrictions being relaxed; do not relabel the operation as new.
2. Inspect gopls extraction actions under the retained session on representative regions. Accept their output only if helper naming, selection, comment ownership, evaluation and refusal contracts match. Command/edit capture is a prerequisite if needed.
3. Build compiler-based variable identity and CFG live-in/live-out analysis over the current overlay. Distinguish local mutation, address identity, closures, declarations and result ownership.
4. Plan helper parameters/results, receiver policy and caller substitutions together. Avoid copying mutable value receivers or changing address identity; preserve shadowing and evaluation order.
5. Preserve comments by explicit ownership mapping and choose result types from resolved compiler information. Validate combined output with types and behavior fixtures before Apply.
6. Keep old line selectors tied to the current working revision; add structural selectors only after their shared introspection and revision contract is accepted. Return actual qualified helper identities for method extraction.
7. Expand control flow in a separate milestone after straight-line proof gates pass.

### Required batch behavior

Batch: rename the owner or insert local assignments without publication → extract from the current working statement range → rename the resulting helper → replace its body. An earlier insertion shifts line coordinates: resolve the new range or reject an old expected_revision rather than selecting different statements. A later mutation resolves the new helper and revised owner. A refused extraction publishes only earlier successful edits at batch completion.

Resolve all selectors and compiler inputs against the current overlay, including earlier unsaved children. Validate every output and analysis preimage before one `sourcebuf.Buffer.Apply`; a refused operation must contribute no partial working edits. Synchronize the retained session for dependent queries without creating a new gopls process for ordinary content changes. Defer normalization and diagnostic collection according to the existing batch coordinator. Format only sources whose bytes changed; an empty write set skips normalization. Honor configured hook scope and trust.

Earlier completed edits remain eligible for final publication when a later child refuses. Report ordered child results and the actual persisted final diff. Publication is atomic per file and can fail partially; never describe the batch as a filesystem transaction. On publication failure discard unpublished state, invalidate retained sessions, and derive outcomes from actual disk bytes.

### Public acceptance and refusal coverage

* Multiple live-ins and two live-outs with intervening assignments: runtime values and mutation counts unchanged; exact generated helper signature checked.
* Parameter/receiver shadowing, local package-name aliases, supported inferred custom result types and attached comments preserve bindings/source ownership.
* Entire self-contained loop/branch extraction preserves iteration and panic behavior; exits crossing the region boundary refuse before edits.
* Address-taken local/receiver, captured closure, defer/recover/go, named results and generics remain explicit refusal cases until the corresponding milestone.
* Method helper receipt feeds rename and replace_body using the returned qualified `symbol` without synthesizing a receiver name.
* Preserve existing ADR-0056 cases and document only the restrictions actually removed.

Use `testdata/scripts/*.txtar` to invoke the CLI and public MCP batch ingress, assert receipts and resulting files, and execute fixture behavior where semantic preservation is claimed. Include default/false import organization, no-op byte comparison, a malformed request, unsupported selection with zero writes, canceled analysis, and disk-preimage conflict. Real-gopls batch coverage must prove unpublished intermediate bytes and process reuse; shallow protocol mocks supplement failure isolation only.

### Semantic risks and unresolved decisions

* Typechecking does not prove extraction preserves control flow, receiver identity, address escape, captures, panic/recover or defer timing.
* Named result variables participate in deferred closures. Moving their assignments into a helper can change when/what a defer observes.
* Range variables and loop-captured closures require Go-version-aware semantics, not syntax-only substitution.
* Native extraction actions may lose comments or choose names/types contrary to the accepted operation contract. A candidate action is not automatic authorization to weaken ADR-0056.

### Decision gates and completion

Prove straight-line data flow with a compiler-backed liveness model, explicit receiver/alias rules and runtime fixtures before relaxing existing refusals. Agree how new structural selectors coexist with current line selection without silently changing meaning. Update ADR-0056 and its index only for accepted extensions.

Run `make check` before focused verification, then the focused public cases, and `make check` again before handoff. Resolve formatter/import/lint rewrites before investigating remaining failures. Log semantic-tool failures immediately in `docs/SUBOPTIMAL_TOOLS.md`. No `go.work` mutation, runtime bootstrap, paid benchmark, or automatic publication is authorized by this plan.

Completion requires CLI/MCP schema parity, buffered behavior, observable proof cases, honest capability/refusal documentation, and a reviewed commit. Accepted architecture/schema choices require an ADR update and its index in the same change. Keep this tool handover Open until its investigation is resolved; add findings and a bounded follow-up list rather than marking a partial implementation complete.

### Dependencies, ownership, and entry points

Depends on live-target/revision-bound receipt foundations, typed source snapshots, and [change signature](RQ-0060-go-refactoring-tool-implementation.md#go-change-signature) parameter/type planning where reusable. Can run in parallel with inline function after agreeing shared evaluation/alias proof helpers. Parent owns registry/capability/index integration.

* Extend `internal/astedit/extract.go`, `internal/operation/wire_extract.go`, and `internal/backend/golang/backend.go`.
* Read [ADR-0056](../adr/0056-go-call-statement-extraction.md) and existing `testdata/scripts` extraction cases.
* Use pending `sourcebuf`/session overlays; investigate [gopls extraction actions](https://go.dev/gopls/features/transformation) without invoking a subprocess.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_function`:**

> Use this tool instead of copying statements to extract a supported contiguous Go statement range into a helper with inferred parameter and result flow. Choose a fresh simple `symbol`; `in_function` selects the existing owner and required `kind` chooses function or method. The selected `start_line` and `end_line` must match current direct-statement boundaries. Pass the created helper's qualified result `symbol` and matching `file` to `semantic_rename` or `semantic_replace_body`; pass the live `in_function` owner to construct tools. Qualified helper results do not feed this tool's new simple-name input.

**Parameter description requirements:**

`symbol`: Target: new simple helper identifier chosen by the caller. Remove qualified existing-symbol sources from this creation input. `in_function`: Selector: copy `semantic_extract_function.in_function` or another validated live same-named owner projection. `start_line` and `end_line`: Selectors: current inclusive statement boundaries; `semantic_lookup.line` and `column` are navigation hints, not these parameters. Descriptive lookup `kind` does not feed required helper `kind`.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go extract variable expansion

Original handover: RQ-0052. Implementation status: Open.

### Context

ADR-0059 and `semantic_extract_variable` cover one exact, side-effect-free whole argument in a direct one-argument call statement. The narrow implementation rejects returns, nested expressions, multiple arguments, constant-only expressions, comments, and other forms. Expansion can reduce routine manual work, but expression movement can change evaluation order, typing, scope, and observable behavior.

### Research questions

1. Which next expression positions can be extracted while preserving Go evaluation order and inferred type?
2. Can repeated identical expression sites share one new local without moving evaluation across branches or loops?
3. Which supported cases need `go/types` or retained gopls facts rather than syntax-only proofs?
4. How should target selection and refreshed continuation receipts behave after inserting a declaration?
5. Which expansion is useful enough for a separate milestone instead of broadening one hard-to-audit contract?

### Current evidence and limits

* `internal/astedit/extract_variable.go` implements the bounded engine; `internal/operation/wire_extract.go` defines `semantic_extract_variable`, CLI schema, and receipt.
* ADR-0059 defines `symbol` as the new local, `in_function` as owner selector, `discriminator` as exact whole-expression source, and `construct_path` as a zero-based occurrence path.
* The current implementation only accepts one whole argument of a direct single-argument call statement in the owner's direct body. It requires a typed owner input, rejects constants-only expressions, calls, receives, address-taking, literals, comments, returns, and nested positions.
* Existing file edits are batch-buffered in the authoritative pending source-buffer state. Expansion must continue to read and edit that same overlay.
* This is an expansion of the existing tool, not a second `extract_expression` alias and not a constant-extraction operation.

### Candidate outcomes and non-goals

First investigate extracting a whole argument from a direct call with multiple arguments only when the selected argument's position and all earlier/later argument evaluation are proven unchanged. Compare this with extracting a direct RHS in a single assignment or a direct return expression. Repeated-site extraction is a separate candidate: one local is only valid when all sites execute at the same frequency and control-flow point.

Do not move expressions out of short-circuit operands, conditionals, loop headers/bodies, defer/go calls, or closures in the first expansion. Do not infer arbitrary interfaces, generate constants, perform extract-function work, or claim behavioral equivalence from formatting and diagnostics alone.

### Proposed API questions

Retain `semantic_extract_variable` / `extract-variable`. Proposed primitive inputs:

* `file` string: exact workspace-relative Go file.
* `symbol` string: new local name; simple Go identifier, not qualified.
* `in_function` string: fully-qualified owner selector copied from an owner receipt.
* `discriminator` string: exact source bytes of the whole selected expression, whitespace significant.
* `construct_path` string: proposed canonical zero-based path when source text occurs more than once; determine whether it is required for all nested positions or only ambiguity.
* `expected_revision` string: investigate a canonical source revision token with identical input/output name and format; it must bind the fragile expression selection and be checked against the overlay.

All of `file`, `symbol`, `in_function`, and `discriminator` are required. `construct_path` is optional only when the selected text is unique; if supplied, it is zero-based and requires the matching `expected_revision`. No default owner or expression is inferred. `expected_revision` is an optional opaque string on an initial request and required when following a revision-bound receipt. Import-organizing options are not needed because this rewrite does not introduce imports.

Input positions such as `argument_index` or `in_return` are unresolved alternatives. Prefer one schema that names the structural role without overloading `symbol`. Any new receipt-fed input must have an exact compatible `SourceFields` entry and prose linking it to its source tool and field. Caller-authored choices have no invented producer edge. Do not add structured selector objects until recursive schema support exists.

### Continuation receipt proposal

Return the new local as a live `symbol`; return current `file` and `in_function`. Return a refreshed `discriminator` only if it still selects a live expression for a safe next edit. Return a refreshed zero-based `construct_path` only with the matching `expected_revision`. Put removed expression text and old path under evidence, clearly marked non-reusable.

```json
{
  "file": "pkg/serve.go",
  "symbol": "normalized",
  "in_function": "(*Server).Serve",
  "discriminator": "strings.TrimSpace(raw)",
  "construct_path": "2.1.0",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/serve.go"],
  "diagnostic_status": "unchanged"
}
```

The example path is zero-based and is copyable only with its exact `expected_revision`; decide revision encoding before implementation. Removed source is evidence, never a continuation.

### Safety questions and hazards

* Multiple arguments: extracting argument 2 into a preceding local evaluates it before argument 1, changing order. Only permit if the selected argument is first or preceding arguments are proven pure and stable; syntax-only purity classification may be too weak.
* Assignment and return contexts: a preceding local can change when other operands are evaluated and whether a value is evaluated on a branch.
* Repeated occurrences: evaluating once instead of N times changes side effects, panics, allocation identity, and timing. Reject calls and receives unless repeated evaluation behavior is explicitly preserved.
* Type changes: untyped constants, nil, generic inference, and assignability may change when a new declaration is introduced. Require type facts or reject.
* Name binding: selected names may be shadowed in sibling scopes; identifier spelling alone is insufficient.
* Comments, directives, and build constraints must remain attached and unchanged.
* A request can become stale after an earlier batch edit; selection and revision validation use the live overlay.

### Ordered milestones

1. Build a table of expression contexts, evaluation order, type rules, and refusal reasons using real Go examples.
2. Select exactly one next useful context and prove the transform with `go/types` or another existing authoritative service.
3. Specify how `construct_path`, ambiguity receipts, and optional `expected_revision` identify the selected live expression.
4. Implement overlay-aware selection, validation, complete-source formatting, `Buffer.Apply` publication, and diagnostics using the configured scope after finalization.
5. Expand the CLI/MCP schema and Go capability, with no compatibility alias or non-Go claim.

### Public behavior tests to add

* CLI txtar success for a typed expression in the selected newly supported context; assert exact final file tree, local placement, receipt, and diagnostics.
* CLI txtar refusal for an expression whose movement reorders a side-effecting sibling; assert refusal reason and byte-identical source.
* CLI txtar refusal for stale `expected_revision` or ambiguous repeated source; assert valid fresh selector evidence but no mutation.
* CLI txtar sequential-command success edits the owner, extracts from the resulting on-disk source, then applies a later body edit; assert each command result and the final source. The MCP batch case separately covers the unsaved overlay.
* MCP `semantic_batch` case: statically specify an owner-body edit followed by extraction against its known unsaved result and a later edit. Any `construct_path` and `expected_revision` are literal fixture-derived values, not references to the earlier result. After the batch, make a separate follow-up mutation using the returned local `symbol`, then run a standalone lookup to prove the new local remains live. A separate refusal invocation proves the earlier successful batch mutation remains published.
* Add refusal case where an unsafe extraction follows a successful insertion; the batch stops, the insertion remains eligible for final publication, and any repair/later edit runs in a separate invocation.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_extract_variable`:**

> Use this tool instead of copying an expression and replacing its uses to extract one supported Go expression into a preceding local without changing observable evaluation. Choose a fresh simple `symbol`; `in_function`, `discriminator`, and any revision-bound `construct_path` select the expression. Pass the new local's live `file`, `symbol`, and `in_function` to `semantic_inline_variable` when its binding/use contract supports the result. Pass live `file` and `in_function` to construct operations. Reuse an expression selector only if the result explicitly recomputes a current eligible expression; old source text is evidence.

**Parameter description requirements:**

`symbol`: Target: fresh simple local identifier; qualified declaration outputs and an already-created local do not supply a new name. `in_function`: Selector: live same-named owner from `semantic_extract_variable.in_function` or a compatible construct receipt. `discriminator`: Selector: current selected expression only; remove `semantic_extract_variable.discriminator` as an automatic source when it represents replaced text. A new lookup/inspection projection must prove its live context before reuse.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go inline variable expansion

Original handover: RQ-0053. Implementation status: Open.

### Context

ADR-0058 exposes a conservative `semantic_inline_variable` for a literal local used as a direct call argument or direct return value in the immediately following statement. It refuses expressions and call inlining. Broader inline-variable support is attractive, but substituting an initializer at use sites can duplicate evaluation, observe a different value, alter scope, or select another binding.

### Research questions

1. Which additional initializer forms can be substituted while preserving evaluation count and order?
2. Can multiple use sites be handled safely, and under what single-execution/control-flow proof?
3. Which expression contexts can receive a typed replacement without changing dispatch or method receiver behavior?
4. Should call inlining remain a separate operation and contract?
5. How can an earlier buffered edit safely feed a later inline request with stable binding identity?

### Current evidence and limits

* `internal/astedit/inline.go` contains current inline-variable analysis and mutation; `internal/operation/wire_engine_inline.go` defines the Go operation and schema.
* ADR-0058 accepts one local literal declaration with no explicit type and uses in the immediately following statement in the same lexical block. It inserts default-type conversions for untyped literals and refuses additional references, comments, intervening statements, ambiguous binding, or uncertain import names.
* No inline-function/call expansion is implemented by this tool. A generic `semantic_inline` would obscure materially different proof obligations.
* The authoritative pending main uses `sourcebuf` for current source and buffered execution. The older root branch is not a safe implementation base.

### Candidate outcomes and non-goals

Investigate in order: multiple direct uses in one next statement; direct uses later in the same block with no intervening writes or control-flow boundary; side-effect-free composite expressions with type facts. Each item should be accepted only with a binding and evaluation proof. Inline function/call should be a separate RQ or ADR with call graph, dispatch, receiver, and control-flow analysis.

Do not duplicate an initializer expression that may call, receive, mutate, panic, allocate identity-sensitive objects, or depend on a variable that changes between declaration and use. Do not inline across closures, defer/go, loops, branches, labels, or statement boundaries until exact execution counts are proven. Do not treat successful compilation as behavioral proof.

### Proposed API questions

Keep `semantic_inline_variable` / `inline-variable`, with primitive inputs:

* `file` string: workspace-relative Go file.
* `symbol` string: local target name; `in_function` selects its owner.
* `expected_revision` string: candidate binding/source revision token for stale requests.
* Existing `in_function` format remains fully-qualified and is a selector, not the mutation target.
* Require `file`, `symbol`, and `in_function`; no defaults. `expected_revision` is an optional opaque string on the first request and required for a follow-up based on a prior receipt. If one declaration has multiple same-name uses, determine whether the operation always transforms all proven uses or accepts a zero-based `construct_path` use selector; any path requires the matching `expected_revision`. Avoid a hidden default that changes a subset.

If a future call-inlining operation is considered, define it separately and avoid reusing variable-specific parameters for function targets. Every value copied from a receipt needs `SourceFields` plus exact schema-source text.

### Continuation receipt proposal

After success, the removed local no longer exists as a live `symbol`. Do not return it as a reusable target. Return `file`, `in_function`, changed files, and a `removed_binding` evidence object with original name/location and transformed use count. If the function owner remains live, return it verbatim. Include `expected_revision` under that exact name, paired with any returned zero-based `construct_path`; bind it to the selected file revision.

```json
{
  "file": "pkg/serve.go",
  "in_function": "Server.Serve",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/serve.go"],
  "removed_binding": {"name": "limit", "uses_rewritten": 2},
  "diagnostic_status": "unchanged"
}
```

`removed_binding.name` is historical evidence and must not be advertised as a follow-up `symbol`.

### Semantic hazards and refusal rules

* Substitution can duplicate evaluation for multiple use sites, changing side effects, panics, allocation identity, and timing.
* A write to the binding or an alias-visible value between declaration and use makes the initializer stale.
* Shadowed declarations and short declarations require identity-aware resolution; function-wide spelling matches can over-refuse or under-refuse.
* Conversion insertion can change constant representability, rounding, overflow behavior, typed nil behavior, or generic inference.
* A method call expression substituted as an argument can change receiver evaluation order or copy a value.
* Comments attached to the declaration or any use cannot be discarded or relocated by approximation.
* Stale selectors after a previous buffered edit must be checked against overlay bytes and revision evidence.

### Ordered milestones

1. Enumerate candidate expression classes and build a semantic table of evaluation multiplicity, type preservation, and control-flow constraints.
2. Choose one class and prove it with resolved object identity and type facts; keep a fail-closed boundary for all others.
3. Decide whether all uses are atomic target or whether use-site selection belongs in a separate operation.
4. Implement overlay reads and edits; validate the full candidate source, then defer normalization to batch finalization and preserve the configured diagnostic scope.
5. Register Go-only capability and update receipt cross-references with no inline-function alias.

### Public behavior tests to add

* CLI txtar success for multiple safe direct uses with preserved single initializer evaluation; assert exact final tree and `uses_rewritten` receipt evidence.
* CLI txtar refusal for a call initializer used twice; assert no duplicate side effect and unchanged source.
* CLI txtar refusal for a reassigned/shadowed binding and for a declaration comment whose ownership cannot be preserved.
* CLI txtar success/refusal at the newly accepted expression context; assert exact inferred type and no unsupported call-inlining claim.
* MCP mixed batch: statically specify a body edit, inline a uniquely selected binding from the known unsaved result, then make a later edit. Do not interpolate a receipt in the batch; use only fixture-known names, and pair any zero-based path with its exact revision. Run a separate follow-up mutation using a live receipt target, then a standalone lookup to prove freshness.
* MCP batch with one successful edit followed by an unsafe inline refusal stops at that refusal; the earlier mutation remains eligible for publication. Any repair or later success is a separate invocation.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_inline_variable`:**

> Use this tool instead of replacing variable uses by text to inline one supported local Go binding while preserving value, evaluation count, and ordering. `symbol` targets the local and `in_function` selects its owner. The successful operation removes the binding. Pass the live `file` and `in_function` to `semantic_replace_construct` with a fresh supported construct selection. The removed binding name is historical evidence and must never be copied into another symbol-target call. Wider initializer and use forms are advertised only after their proof cases pass.

**Parameter description requirements:**

`symbol`: Target: existing local binding in the selected owner; copy only a proven compatible live local identity. Remove `semantic_inline_variable.symbol` from sources after successful deletion. `in_function`: Selector: copy the live `semantic_inline_variable.in_function` or compatible owner projection. Its file-scoped revision belongs to that owner context, not the removed binding.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go safe delete expansion

Original handover: RQ-0054. Implementation status: Open.

### Context

ADR-0057 defines `safe_delete` on `semantic_remove_construct`; only an unexported standalone constant is currently accepted after a conservative module source scan. Ordinary removal is structural and does not prove behavior. The pending main has source-buffer batching for ordinary edits, while safe-delete preflight still needs a complete overlay/read-set design so it does not create a disk-only barrier or inspect stale bytes.

### Research questions

1. Which additional declaration kinds can be safely deleted and what exact reference, directive, resource, or language-specific checks establish the guarantee?
2. How can the module source scan include unsaved buffered sources and prove the scanned file set and contents remain stable through `Buffer.Apply`?
3. What does a complete Go source set mean across build tags, tests, generated files, vendor and ignored directories, symlinks, and nested modules?
4. How should refusal checks, scan coverage, and overlay revisions be serialized without implying a stronger guarantee than was established?
5. Can safe deletion become a normal buffered operation with no intermediate flush, barrier, or silent downgrade?

### Current evidence and limits

* `internal/astedit/remove.go` handles exact structural removal and safe-delete preflight; `internal/operation/wire_remove.go` registers `semantic_remove_construct` / `remove-construct`, `safe_delete`, checks, and current Go-only scope.
* ADR-0057 accepts safe deletion only for unexported standalone constants; grouped constants, exported constants, all other node kinds, uncertain source sets, directives, and any same-name module identifier are refused.
* Normal removal already reads through `sourcebuf`; current safe preflight must be refactored to read its complete source set through the same overlay and return a read-set proof.
* `internal/sourcebuf/buffer.go` `Buffer.Apply` atomically validates the complete output set against source revisions for existing files; `internal/mcp/buffers.go` finalizes buffered edits, normalizes changed sources, and retains configured diagnostic scope and coverage.
* The pending main retains a gopls session. Do not introduce a safe-delete barrier that flushes changes to disk or restarts/resynchronizes that session.

### Candidate outcomes and non-goals

Evaluate separate classes: unreferenced private free functions/types/variables; package-private methods; exported API; resource-bearing declarations; grouped declarations with iota; cases/statements. A class is safe-delete-supported only if the engine can explain and verify its complete relevant reference set and collateral effects. For many classes the answer should remain refusal.

No safe-delete success may be based solely on syntax validity, active-build references, text grep without stated completeness, or compiler diagnostics. Do not silently execute ordinary removal when `safe_delete=true`. No claim of semantic preservation for case/statement removal or broad program equivalence.

### Proposed operation contract

Keep one `semantic_remove_construct` / `remove-construct`; `file` and `kind` are required. `symbol` is required for declaration kinds; `in_function` is required for cases; `discriminator`, `target_case`, and `construct_path` are optional selectors only where applicable. `safe_delete` is Boolean default `false`; `auto_organize_imports` is Boolean default `true`. `expected_revision` is an optional opaque token on a first request and required for follow-up; it represents the selected file revision only, never the complete source set. Preserve ADR-0057 selector roles. Proposed output adds `safety_status`, `checks_run`, `source_set_revision`, and a stable list of affected paths. Decide which are public schema fields versus structured refusal payload fields before implementation. Refusal must say the guarantee could not be established and include checks that completed.

`expected_revision` is a primitive selected-file revision token with the exact same input/output name: optional on an initial request and required on receipt-based follow-up. Keep the full scan read-set digest separately as `source_set_revision` historical evidence; it is not a retry token and must not be conflated with `expected_revision`. The complete scan must observe overlay bytes for all existing source files, track every file read and metadata identity, and pass that read set to apply-time validation. If the current `Buffer.Apply` API does not accept read-only dependencies, research a read-set extension instead of weakening the proof. Scan-set changes, unreadable/unparseable files, symlinks or unsupported source boundaries refuse. No direct disk scan may ignore an unsaved identifier occurrence.

### Continuation receipt proposal

Deleted `symbol` is not a live target and must not be returned as a reusable continuation. Return the containing file and safety evidence only; for case deletion, the owner `in_function` remains live. Keep the removed target name, old expression, and old `construct_path` under `removed` evidence, not at top level as input-shaped fields.

```json
{
  "file": "pkg/config.go",
  "safety_status": "refused",
  "expected_revision": "opaque-selected-file-revision",
  "checks_run": ["overlay_source_set", "parse_all_go_files", "identifier_scan"],
  "source_set_revision": "sha256:historical-read-set",
  "removed": {"symbol": "legacyLimit", "kind": "const"},
  "changed_files": [],
  "diagnostic_status": "unchanged"
}
```

The selected-file token in `expected_revision` is a retry input; `source_set_revision` is historical complete-read-set evidence only. The digest format and whether refusal can be a normal structured result are unresolved. Do not invent a success receipt for this refusal example; MCP `isError` and structured error shape must match the established contract.

### Safety and consistency hazards

* Build tags and test variants: a reference in an inactive variant still matters for deletion; active package loading alone is incomplete.
* Generated code, linkname, cgo, reflection, string-based registries, and external callers may defeat name-based reference proof.
* Same spelling is conservative evidence, not binding identity; over-refusal is acceptable, false-safe success is not.
* Grouped const deletion can renumber iota siblings. Multi-name specs can bind shared RHS expressions.
* Nested modules and symlinks can make module boundaries ambiguous; fail closed until source-set semantics are documented.
* A buffer overlay can add or remove a reference after the initial scan. Both bytes and path set must be revalidated before publication.
* `Buffer.Apply` validates and applies all outputs atomically to the working overlay. `Buffer.Flush` provides per-file atomic disk publication without a global rollback guarantee; read-set validation must occur before publication begins.

### Ordered milestones

1. Define the source universe and overlay read-set proof with explicit ignored-directory/nested-module rules.
2. Identify the smallest next target class and enumerate unsupported dynamic-reference channels.
3. Design additive `Buffer` read-set validation; confirm no direct writes, flush, or barrier are needed.
4. Implement false-safe-resistant refusal behavior, checks-run receipts, configured-scope diagnostics, and CLI/MCP schemas.
5. After evidence, revise ADR-0057; leave unsupported classes unadvertised.

### Public behavior tests to add

* CLI txtar safe success for currently supported standalone private constant and for any newly supported class; assert file tree, checks, and exact surviving declarations.
* CLI txtar refusal for grouped/iota constants, exported symbols, resource/directive cases, incomplete source set, and another identifier in an inactive build-tag file; assert target bytes unchanged.
* CLI txtar overlay refusal: an earlier unsaved edit inserts a reference to the target, then `safe_delete=true` must see it and refuse. The batch stops; the earlier edit remains eligible for publication. A repair and later successful mutation are separate invocations; use a still-live target from the eligible earlier mutation, then verify it with a standalone lookup.
* CLI txtar refusal when an earlier buffered removal changes the scan source set or when the read-set revision changes before apply; no safe downgrade.
* MCP batch with a successful edit followed by safe-delete refusal proves the refusal sees the overlay and does not roll back the earlier mutation; the batch stops there. Run any repair and later successful edit in a separate invocation; consume a live surviving symbol, then verify it with a standalone lookup.
* Receipt/schema cases distinguish ordinary structural remove from `safe_delete=true`; diagnostics must not call an ordinary removal safe.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_remove_construct`:**

> Use this tool instead of deleting source blocks by text to remove one supported Go declaration or case. `symbol` targets a declaration; cases use `in_function` and current case selectors. `safe_delete` defaults to false; true requires complete supported reference and resource proof or refuses. A removed target and its old selectors are historical evidence, never continuation arguments. Pass a still-live `file` to `semantic_organize_imports` or a live case owner's `file` and `in_function` to `semantic_replace_construct` with a fresh selection. Structural deletion alone does not prove behavioral equivalence.

**Parameter description requirements:**

`symbol`: Target: existing declaration; copy a compatible live `semantic_lookup.symbol`, never the deleted `semantic_remove_construct.symbol`. `target_case`, `discriminator`, and `construct_path`: Selectors: current case/switch projections only; removed case paths cannot supply the next deletion. `expected_revision`: Precondition: selected-file token. `source_set_revision`: Evidence: full scan proof, not a consuming revision parameter.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go move construct expansion

Original handover: RQ-0055. Implementation status: Open.

### Context

Same-file and same-package movement for all currently supported declaration kinds is already implemented; cross-package movement is tightly bounded to an exported free function. Cross-package relocation must preserve comments, imports, references, package bindings, and accessibility. The authoritative pending main already contains buffered batch and shared-session migration work; expansion must preserve that lifecycle.

### Research questions

1. Which declaration kinds and scopes can be moved without changing identity, visibility, initialization, or method sets?
2. What reference and package loading evidence is complete enough for cross-package moves, including inactive variants, generated/test files, dot imports, and overlays?
3. How should destination creation/resource semantics interact with current existing-file-only `Buffer.Apply`?
4. How are same-file, same-package, and cross-package receipts kept copyable under ADR-0055?
5. Which source-set and per-file publication failures can be reported without claiming a global rollback?

### Current evidence and limits

* `internal/astedit/move_construct.go` contains same-file/package movement and dispatch; `internal/astedit/move_cross_package.go` contains bounded package movement and reference rewrite/type-checking.
* `internal/operation/wire_engine_mutations.go` defines `semantic_move_construct` / `move-construct`; it is workspace-scoped and batch-buffered on the authoritative pending main. `destination_file` is optional and defaults to `file` for existing same-file/same-package moves.
* Cross-package behavior currently supports an exported free function in one module and refuses methods, unexported symbols, source-package dependencies, import cycles, ambiguous build/generated files, and unprovable references. Verify the final merged implementation before expanding this snapshot-specific list.
* Commit `50b80b2` contains cross-package reference-binding hardening; the authoritative `6c95d12` source already includes its relevant reference checks although that commit object is not an ancestor.
* `sourcebuf.Buffer.Apply` currently supports existing-file changes. A destination file/package move requiring a new file needs a source-buffer resource contract or an explicit scaffold prerequisite.

### Candidate outcomes and non-goals

Do not reimplement same-file or same-package movement; it already covers all supported declarations. The first new milestone is a bounded cross-package dependency group consisting of a named type plus its directly associated methods, with dependency/reference completeness and type identity proved as one move. Keep unrelated declarations, new-file creation, files/module moves, and cross-module moves out of scope until resource and project-boundary contracts exist.

Do not advertise arbitrary syntax-node movement, package renaming, arbitrary unrelated multi-symbol moves, copy semantics, cross-module moves, or behavior-preserving API migrations. Moving a declaration is distinct from rewriting its callers or changing its signature.

### Proposed input and continuation contract

Keep `semantic_move_construct` / `move-construct`. `file` is the exact source file; `symbol` is the moved declaration; `destination_file` is the sole destination field and is optional, defaulting to `file` for same-file/same-package moves; `placement` and `target_symbol` control placement in the destination file where supported. Do not add a second `destination` or destination package string. Infer destination package only from a validated target file/sibling context.

`file` and `symbol` are required; `destination_file` is optional and defaults to `file` for existing same-file/same-package moves, but is required when selecting a cross-package destination. `placement` is an optional enum using the existing placement values; absent placement uses the current engine’s established default, which must be confirmed in the rebase audit. `target_symbol` is optional unless placement is `before_symbol` or `after_symbol`. `auto_organize_imports` is Boolean default `true`; `expected_revision` is an optional primitive token for the selected source `file` revision and required for receipt-based follow-up. The complete move read set, including destination and dependency files, is captured separately as `read_set_revision` historical evidence; the output `expected_revision` binds the live destination file.

Cross-package reference changes must be typed, identity-checked, and validated against a complete overlay. If a structured relocation plan is proposed, first deliver recursive schema support; current `ParameterContract` accepts primitives. Define complete read-set evidence and apply-time validation separately from the selected-file `expected_revision`; do not overload that token to stand for every dependency file.

### Continuation receipt proposal

After success, top-level `file` is the destination and top-level `symbol` is the destination-file symbol name. Return `destination_file` as the destination too, and put the historical source under `moved_from`, including original source file and symbol plus requested placement/anchor as history metadata. Keep attached comments with the construct. Return a `continuations` array with a `SymbolTarget` entry containing `file`, the destination-file `symbol`, and its `expected_revision`. Do not echo stale placement or target anchors as live continuations; changed-file and diagnostic details remain separate receipt data.

```json
{
  "file": "api/service.go",
  "destination_file": "api/service.go",
  "symbol": "Service",
  "expected_revision": "opaque-destination-revision",
  "read_set_revision": "sha256:historical-move-read-set",
  "moved_from": {"file": "legacy/service.go", "symbol": "Service", "placement": "after_symbol", "target_symbol": "Other"},
  "continuations": [{"file": "api/service.go", "symbol": "Service", "expected_revision": "opaque-destination-revision"}],
  "changed_files": ["legacy/service.go", "api/service.go"],
  "diagnostic_status": "unchanged"
}
```

The cross-package dependency-group example assumes the destination file already exists. Input `expected_revision` binds only the selected source file revision; `read_set_revision` records the complete source, destination, and dependency read set as historical evidence. The output `expected_revision` binds the destination file. For same-package moves, omitted `destination_file` defaults to `file`. Only the `SymbolTarget` with destination file, file-local symbol, and output revision is live; `moved_from` placement and anchor fields are historical metadata.

### Semantic hazards and refusal conditions

* Moving vars can change package initialization order and dependency cycles; require an explicit initialization-order proof or refuse.
* Moving methods changes method sets when receiver types move or are aliased; keep receiver/type scope fixed in the first expansion.
* Type and constant moves can break unexported references or iota values and can change API visibility.
* Comments, build tags, doc comments, compiler directives, and grouped declarations need explicit ownership rules.
* Cross-package import rewrites can resolve to a shadowing package/local or collide with aliases; validate object identity in the typechecked overlay.
* Active package loads miss inactive build variants; conservative source-set scan or explicit refusal is required.
* Earlier buffered edits may change the declaration or references; preflight and rewrite from the current overlay, not disk.
* Per-file atomic writes do not give whole-move rollback; preflight every affected output before the publication phase and report partial failure precisely.

### Ordered milestones

1. Re-audit supported kinds, source-set enumeration, existing-file constraints, and receipt after integration of pending main.
2. Start with a cross-package type plus its directly associated methods; enumerate identity, dependency-group closure, reference completeness, visibility, and method-set conditions.
3. Decide whether new-file/package destinations need sourcebuf resource support or remain refused.
4. Implement overlay planning, complete affected-file typechecking, identity validation, atomic per-file publication, and partial-failure reporting.
5. Add Go-only capability/schema/CLI tests; expand ADR-0049 only after proof gates pass.

### Public behavior tests to add

* CLI txtar cases preserve same-file/same-package behavior for existing declaration kinds, then add a cross-package type-plus-method dependency-group success; assert exact source/destination trees, comments, imports, and destination `SymbolTarget` receipt.
* CLI txtar refusal for initialization-order risk, grouped declaration/iota effects, method receiver mismatch, inaccessible destination, and unparseable affected file; assert all preflight refusal cases leave files unchanged.
* CLI txtar cross-package reference cases for active and inactive build variants, dot-import/alias collision, untyped reference, and shadowing destination names; assert no wrong-binding rewrite.
* MCP mixed batch: statically specify an edit to a source type, then move the known type-plus-method dependency group into an existing destination package, then apply a later statically targeted edit in the destination. Use fixture-derived revisions for the exact intermediate overlay; no batch interpolation is available. Run a separate follow-up mutation using the returned destination-qualified `SymbolTarget`, then a standalone lookup to prove it remains live.
* MCP batch with an earlier successful unrelated edit and a refused move stops at refusal; the earlier edit remains eligible for publication and move outputs stay unchanged. Any repair/later success is a separate invocation.
* Case destination creation separately: current `Buffer.Apply` cannot create files; verify explicit refusal unless sourcebuf resource support or scaffold prerequisite is part of the accepted design.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_move_construct`:**

> Use this tool instead of cutting declarations and repairing references by text to move a supported Go declaration to an existing `destination_file`. The destination defaults to `file`; cross-package moves use the same retained session and batch overlay within the supported module boundary. The live result `file` is the declaration's destination. Pass that `file`, qualified `symbol`, and matching accepted `expected_revision` directly to `semantic_rename` or `semantic_replace_body` for callables. Moved-from paths and read-set evidence are historical data. Broader type, dependency-group, or resource moves require their separate support gates.

**Parameter description requirements:**

`file`: Source selector: current file of the live target; after a move copy `semantic_move_construct.file`, which must name the destination. `symbol`: Target: exact moved declaration identity copied from live `semantic_lookup.symbol`, `semantic_rename.symbol`, or `semantic_move_construct.symbol`, with that same record's `file`. `destination_file`: Destination selector: caller-selected existing path; reusing a destination receipt means explicitly choosing another move destination, not locating the current target. `read_set_revision` is not `expected_revision`.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go insert field

Original handover: RQ-0056. Implementation status: Open.

### Context

Existing declaration insertion handles complete functions/methods, new type declarations, and top-level constants/variables. It cannot add a field to an existing struct proportionately. `docs/MANUAL_EDITS.md` records this gap and suggests `semantic_insert_field`. Adding a field affects keyed/unkeyed composite literals, embedded-field promotion, tags, and API shape, so syntax insertion alone is not always semantics-preserving.

### Research questions

1. What field source forms are supported: named, grouped names, embedded fields, tags, generic type expressions, and documentation comments?
2. How should insertion detect field-name collisions and promoted-name conflicts without claiming full selector compatibility?
3. What source coverage should warn or refuse when positional composite literals exist?
4. Should the operation insert only source or also update in-package keyed literals? The latter is a separate coordinated transform.
5. What is the stable selector and receipt for the existing type and new field?

### Current evidence and scope boundary

* `internal/astedit/function.go` and `internal/operation/wire_engine_declarations.go` already insert complete receiver method declarations; Go methods are file-level declarations. A generic add-member tool for methods would duplicate existing capability.
* `internal/astedit/type.go` and `semantic_insert_type` add a new struct/interface/type alias, not a field inside an existing type.
* `docs/MANUAL_EDITS.md` rows ME-0041 and ME-0047 state that existing tools cannot insert one struct field; they suggest `semantic_insert_field`.
* A field insertion tool is a genuine missing operation. It is distinct from method insertion and from whole-type replacement.
* Pending main has sourcebuf overlays and batch support; the root worktree is behind that migration.

### Outcomes and non-goals

Target a single named field inserted into one existing, non-alias Go struct. Investigate grouped fields and embeddings as later milestones. Preserve surrounding fields and comments. Do not rewrite composite literals, implement interfaces, alter method sets, perform field renames, or generate accessors in the first release.

Avoid claiming that positional composite literals remain valid. Before accepting the operation, research package/module scan cost and whether the tool should warn/refuse or require caller acknowledgment. No generic `semantic_add_member` alias is proposed because receiver methods are already expressible through `semantic_insert_function`.

### Proposed API

Propose `semantic_insert_field` / `insert-field`, Go only, existing-file scope, `Batchable: true`, `BufferedBatch: true`.

* `file` string: workspace-relative source file.
* `in_type` string: fully-qualified struct type selector; common selector spelling shared with future member/type plans. It is a scope selector.
* `symbol` string: target new field in qualified continuation form `Type.Field`; validate the simple field suffix and require it to agree with source.
* `source` string: exactly one Go field declaration, including optional tag and field doc comment; never accept an entire struct body.
* `placement` enum: `type_start`, `type_end`, `before_symbol`, or `after_symbol` within the selected struct; default `type_end`. `before_symbol` / `after_symbol` retain the existing qualified anchor semantics.
* `target_symbol` string: existing field name required only for before/after placement.
* `expected_revision` string: investigate stale-struct protection and exact receipt identity.

`file`, `in_type`, `symbol`, and `source` are required primitive strings. `placement` is an optional enum with values `type_start`, `type_end`, `before_symbol`, `after_symbol`; default is `type_end` and the start/end values follow the shared RQ-0024 placement vocabulary. `target_symbol` is required only for `before_symbol` / `after_symbol` and keeps existing qualified anchor semantics. `auto_organize_imports` is Boolean default `true`. `expected_revision` is an optional opaque string initially and required on receipt-based follow-up. `construct_path` if introduced for nested/ambiguous type targets must be zero-based and accepted only with its matching revision. Use `SourceFields` only for proven compatible file, owner-type, revision, and existing-anchor sources, with exact source-field names in their descriptions. The new field identity is a caller-authored target, not an existing-field receipt copied into another insertion. Avoid a nested field descriptor until recursive schema support exists.

### Continuation receipt proposal

Return the new live field as `symbol` in `Type.Field` format and exact `file`/`in_type`. Return refreshed placement fields only when an anchor still exists. Include changed files and diagnostics.

```json
{
  "file": "pkg/config.go",
  "in_type": "Config",
  "symbol": "Config.Timeout",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/config.go"],
  "diagnostic_status": "unchanged"
}
```

Do not return byte offsets or an obsolete pre-insertion `construct_path`. The field type and source are evidence, not necessarily a reusable operation target.

### Semantic hazards and refusals

* Duplicate direct field names are invalid; embedded-field promotions can create ambiguous selectors and need an explicit policy.
* Unkeyed composite literals in the package or downstream packages can break when a field is added. Syntax validity does not establish API compatibility.
* Tags affect encoding, reflection, persistence, and external consumers; preserve source bytes exactly and do not synthesize tags.
* A field name can collide with promoted names or method names and affect selector resolution.
* Generic struct type declarations and aliases need careful type resolution; initial release may refuse them.
* Documentation comments and directives attach to fields differently from type docs; preserve adjacent comments or refuse ambiguous ownership.
* Same-batch replacement may create/remove the target struct. Resolve against the current overlay and require it to exist at the insertion step.

### Ordered milestones

1. Specify supported `ast.Field` shape, placement, comment/tag ownership, type resolution, and collision rules.
2. Study positional literal and external API compatibility policy; define refusal/warning behavior and checks.
3. Implement one field insertion against overlay source, validate parse/type/diagnostics and publication read set.
4. Register primitive schema, tool descriptions, `SourceFields`, Go capability, CLI and MCP receipts.
5. Expand to embeddings/grouped fields only with separate evidence and comparable txtar coverage.

### Public behavior tests to add

* CLI txtar success inserts a single typed field before/after a named field and at last; assert exact struct source, comments, tag, formatted output, and receipt.
* CLI txtar refusal for duplicate field, malformed field, wrong `in_type`, missing anchor, multiple declarations, unkeyed literal compatibility risk per selected policy, and stale revision; source remains byte-identical.
* CLI txtar sequential commands first create the target type, then insert the field and a method that reads it; assert each resulting tree and receipt. The separate MCP batch uses statically specified entries and fixture-derived revisions, with no result interpolation. A separate follow-up mutation consumes `Config.Timeout`, then a standalone lookup proves the field target remains live.
* MCP `semantic_batch` with an earlier unsaved type edit, then a field insertion and a later method insertion proves each operation sees the overlay; use static selectors and matching fixture-derived revisions. Diagnostics retain configured coverage at finalization; a separate follow-up lookup proves freshness.
* MCP batch refusal after an earlier successful edit proves no rollback of the earlier success and no partial field write.
* Include receipt/schema assertions that distinguish `symbol=Type.Field` from `in_type=Type`; do not add duplicate method-add tests because existing insertion covers methods.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_insert_field`:**

> Use this tool instead of replacing a struct body to insert one supported Go field declaration. `in_type` selects the struct; `symbol` is the new qualified `Type.Field` target and `source` supplies exactly that field. Placement defaults to `type_end`; relative placement needs `target_symbol`. The engine validates collisions and the advertised layout and literal constraints. Pass the generated field's live `file` and qualified `symbol` to `semantic_rename`, `semantic_inspect_symbol`, or `semantic_encapsulate_field` when available. Repeating this insertion requires a different fresh field target.

**Parameter description requirements:**

`in_type`: Selector: proposed `semantic_inspect_symbol.in_type` once that exact owner projection is implemented. `symbol`: Target: new qualified field identity; prefer an engine-provided same-shaped proposal and require agreement with `source` and `in_type`. No existing field continuation supplies a fresh new field identity. `source`: Replacement: one field declaration, not the full struct. `target_symbol`: Selector: existing qualified field anchor; copy only proposed `semantic_inspect_symbol.target_symbol` after the same-named qualified-field anchor projection is implemented; otherwise no automatic copy edge is available. `MemberTarget.symbol` does not feed `target_symbol` by renaming.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go insert statement

Original handover: RQ-0057. Implementation status: Open.

### Context

Current tools can replace a whole function body, replace a supported whole construct, and insert switch cases. They cannot insert one statement into an existing block. `docs/MANUAL_EDITS.md` ME-0041 and ME-0047 identify guard/call insertion as a repeated gap. A syntactically valid statement can still alter control flow, variable scope, defer timing, or execution count.

### Research questions

1. Which block slots are safe and useful: before/after a direct statement, first/last, or named control-flow slot?
2. How should an existing statement be selected under ADR-0055 without exposing byte offsets or stale line coordinates?
3. Which short declarations and labels can be inserted without shadowing or changing goto legality?
4. How should receipts identify an insertion that has no symbol and no longer has the original anchor text?
5. What behavior can a syntax-only AST operation promise, and what requires type or control-flow analysis?

### Current evidence and scope

* `internal/astedit/body.go`, `construct.go`, and `switchcase.go` implement broader replacement or case insertion, not general statement insertion.
* `internal/operation/wire_engine_mutations.go` exposes body/construct/case operations through typed contracts; no `semantic_insert_statement` registration exists in the current source scan.
* `docs/MANUAL_EDITS.md` names guard/call insertion as a missing capability and proposes `semantic_insert_statement`.
* The current receipt contract uses `in_function` for owner selection and `discriminator` / `construct_path` for a selected construct. A statement insertion itself has no fabricated `symbol`.

### Outcomes and non-goals

First release should insert exactly one parsed statement adjacent to one uniquely selected direct statement in an existing function or method body. Consider `first`/`last` slots only after proving comments and scope boundaries. Refuse control-flow constructs or label interactions that cannot be checked. This is a new primitive, distinct from replacing the whole function body or inserting a switch case.

Do not insert arbitrary statement lists, split/merge blocks, rewrite expressions, create labels, insert into generated files with directives, or claim behavior-preserving guard transformations. Framework-specific patterns remain skills.

### Proposed API

Propose `semantic_insert_statement` / `insert-statement`, Go only, existing source, batchable and buffered.

* `file` string: workspace-relative file.
* `in_function` string: fully-qualified containing function/method selector.
* `discriminator` string: exact current source of the direct statement used as anchor.
* `construct_path` string: zero-based direct statement path; required when same text is ambiguous, and refreshed on success only if it still refers to a live statement.
* `placement` enum: `before_construct` or `after_construct` per the shared placement vocabulary; first/last remain unresolved.
* `source` string: exactly one Go statement, no enclosing block or function.
* `expected_revision` string: investigate exact owner/block revision binding.

`file`, `in_function`, `source`, and `placement` are required; `placement` accepts exactly `before_construct` or `after_construct`, with no implicit default. `discriminator` is required for the selected anchor and `construct_path` is optional only when unique; if present, it is zero-based and requires its matching `expected_revision`. `expected_revision` is optional on an initial request and required when following a prior receipt. `auto_organize_imports` is Boolean default `true` if the inserted statement can introduce imported identifiers; confirm whether import edits belong in this primitive.

Do not add `symbol`; inserted statements have no declaration target by definition. Use `SourceFields` annotations on copyable file/owner/selector values. If adding structured statement arrays, wait for recursive schema support.

### Continuation receipt proposal

Return `file`, `in_function`, placement, and a refreshed selector for a still-live chosen statement if unambiguous. The inserted statement text can be evidence, but is not a `symbol`. The original anchor source/path is stale after insertion and belongs under `anchor_before` evidence unless recomputed.

```json
{
  "file": "pkg/handler.go",
  "in_function": "Handler.Serve",
  "placement": "before_construct",
  "discriminator": "if ready { return nil }",
  "construct_path": "3",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/handler.go"],
  "diagnostic_status": "unchanged"
}
```

Use the shared zero-based `construct_path` encoding and require the matching `expected_revision`; define any remaining path spelling once through the shared contract. Never return a fabricated symbol for an expression or statement.

### Semantic hazards and refusal rules

* `:=` can introduce shadowed locals or change which later identifiers bind; type/name resolution must validate the resulting block.
* `return`, `break`, `continue`, `goto`, labels, `defer`, and `go` alter control flow or timing; do not treat insertion as inert.
* Inserting before a declaration can make variables unavailable; after a declaration may move code into a different scope.
* Comments/directives attached to the anchor can shift ownership; only insert at AST-safe boundaries and preserve bytes.
* A repeated statement discriminator requires an explicit path; silently selecting first/nearest is invalid.
* A previous batch edit may change anchors. Resolve text and path against current overlay, with stale expected revision refusal.
* Compile success does not prove intended guard behavior or resource cleanup ordering.

### Ordered milestones

1. Define the initial allowed statement grammar and explicit control-flow refusal list.
2. Settle selector path and expected revision format across construct operations.
3. Implement exact direct-statement selection, parse/type validation, overlay writes, and fresh receipt generation.
4. Add registry/CLI/MCP schema and Go-only capability metadata.
5. Consider additional slots or statement forms only after focused user workflows justify them.

### Public behavior tests to add

* CLI txtar success inserts a simple assignment or call with `before_construct` and `after_construct` relative to an exact direct statement; assert exact final tree and refreshed selector.
* CLI txtar refusal for repeated discriminator without path, stale path/revision, nested statement anchor, malformed/multiple statements, illegal label/goto placement, and short-declaration collision; assert unchanged source.
* CLI txtar refusal for unsafe defer/goto/return form unless the accepted first-release grammar explicitly supports it.
* MCP batch: statically specify body replacement, statement insertion against the known unsaved body, and a later edit; any zero-based `construct_path` is paired with the matching fixture-derived `expected_revision`. No result is interpolated inside the batch. Run a separate follow-up mutation against a live owner/declaration, then a standalone lookup to prove the target remains fresh.
* MCP batch with one successful declaration edit followed by insertion refusal stops at refusal; the earlier edit remains eligible for publication and no intermediate write occurs. Any later success is a separate invocation.
* Assert receipts contain `in_function` and selector evidence but no `symbol` for the inserted statement.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_insert_statement`:**

> Use this tool instead of replacing a function body to insert one Go statement before or after a selected current direct statement. `in_function` selects the owner; `discriminator` and an optional revision-bound `construct_path` select the anchor. Supply one statement as `source` and required `placement=before_construct|after_construct`. Pass live result `file`, `in_function`, and recomputed eligible anchor fields to another `semantic_insert_statement` call with a new `source`. Pass the owner to `semantic_replace_construct` with that tool's own required selection. Old anchor paths are evidence, not live selectors.

**Parameter description requirements:**

`in_function`: Selector: copy a live same-named owner projection from `semantic_insert_statement` or a compatible construct receipt. `discriminator`: Selector: exact current direct-statement anchor; copy `semantic_insert_statement.discriminator` only when it is a recomputed live anchor. `construct_path` and `expected_revision` must be copied from the same selected-file result. `source`: Replacement: caller-authored single statement; an anchor result never supplies it implicitly.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go insert composite element

Original handover: RQ-0058. Implementation status: Complete for the bounded keyed struct/map append described below; see ADR-0062.

### Context

Existing declaration and body tools can replace large source units but cannot insert one element into an existing composite literal. `docs/MANUAL_EDITS.md` ME-0041 identifies composite element insertion as a gap. Composite literals can be keyed or positional, nested, typed, and subject to duplicate keys or evaluation-order changes.

### Research questions

1. Which literal kinds are supported first: struct, array, slice, map, keyed, or positional?
2. How is the containing declaration selected without treating it as the element's symbol owner?
3. How are duplicate keys, unknown struct fields, map key collisions, ellipses, and elided nested types validated?
4. Should insertion append only, or support before/after keyed elements with stable anchors?
5. Can the engine prove the literal is the intended one and that a batch edit has not changed its source revision?

### Current evidence and limits

* `semantic_insert_composite_element` is registered as Go-only, file-scoped, batchable, and buffered. It appends one typed keyed struct or map element over the current source overlay and refuses unsupported selectors and invalid typed keys.
* `docs/MANUAL_EDITS.md` ME-0050 records the delivered composite-element flow; ADR-0062 defines its operation-specific final-source selector continuation.
* This operation has no standalone declaration `symbol`. The holder is selected with `in_declaration`, as required by the shared handover contract; `in_function` may be needed only when the literal is local and declaration selection is insufficient.
* Structural package loading reads the current source-buffer overlay, including earlier successful batch children; the public batch fixture replaces a literal and appends against that unsaved result.

### Outcomes and non-goals

Start with one typed keyed struct or map element in a uniquely selected literal. Research whether slice/array append can be safely supported with explicit ordering. Do not rewrite whole literals, synthesize default values, infer an interface, or insert arbitrary nested element lists. Do not claim no runtime behavior change when adding an element whose value expression has side effects.

### Proposed API

Propose `semantic_insert_composite_element` / `insert-composite-element`, Go-only, existing-file, `Batchable` and `BufferedBatch`.

* `file` string: exact workspace-relative file.
* `in_declaration` string: containing declaration selector (the holder, not `symbol`).
* `discriminator` string: exact source text of the selected composite literal or its uniquely identifying expression.
* `construct_path` string: zero-based AST path to the literal, disambiguating repeated literals in the declaration.
* `source` string: one complete element expression, including `key: value` when keyed; no surrounding comma or braces.
* `placement` enum: first release supports only `append`; it has no other value.
* `expected_revision` string: stale declaration/literal guard proposal.

`file`, `in_declaration`, `discriminator`, and `source` are required primitive strings. `construct_path` is optional only when the literal is unique; if present, it is zero-based and accepted only with its matching `expected_revision`. `placement` is an optional enum with default and only allowed value `append`; do not accept a target anchor in the first release. `expected_revision` is optional initially and required for a receipt-based follow-up. `auto_organize_imports` is Boolean default `true` only if generated imports are in scope; otherwise omit it.

Avoid `symbol` for a non-declaration element. Use `SourceFields` for `file`, `in_declaration`, `discriminator`, and any copyable refreshed path. No nested input objects until recursive registry schema support exists.

### Continuation receipt proposal

Return `file`, `in_declaration`, and refreshed `discriminator` / `construct_path` only when they still select the live literal after insertion. Return inserted element source and literal kind under evidence; it is not a live symbol.

```json
{
  "file": "pkg/routes.go",
  "in_declaration": "routes",
  "discriminator": "map[string]Handler{\"health\": health}",
  "construct_path": "1.0",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/routes.go"],
  "diagnostic_status": "unchanged"
}
```

The zero-based path is valid only with this matching `expected_revision`. A refreshed path must be recomputed after insertion; old element paths belong to evidence only.

### Semantic hazards and refusal policy

* Struct literal keyed fields must exist and must not be duplicated; positional struct literals require exact field order and arity.
* Map keys can collide semantically even when text differs (`1` and a typed constant); type-check or refuse uncertain keys.
* Slice/array insertion order affects runtime behavior and index semantics; append-only may be the only initial option.
* Element expressions can have side effects; inserting them changes execution count. Report structural nature and avoid behavior claims.
* Elided nested literal types and generic inference may require type information unavailable from syntax alone.
* Comments around commas/elements have ownership; preserve or refuse ambiguous comments.
* Repeated literals in one declaration require `construct_path`; never pick the first match silently.
* Each operation must resolve its selector against the current overlay; batch entries remain static, so no refreshed result is interpolated. Use matching fixture-derived revisions for zero-based paths, then verify with a separate post-batch lookup.

### Ordered milestones

1. Choose literal families and keyed/positional policy; document exact refusal classes.
2. Specify `in_declaration`, `construct_path`, `discriminator`, and expected revision formats consistently with ADR-0055.
3. Implement one element insertion over current overlay, with full-literal type/syntax validation and no intermediate writes.
4. Register primitive schemas, capability metadata, receipts, and CLI/MCP tests.
5. Expand to nested, positional, or anchored insertion only when a type-safe policy exists.

### Public behavior tests to add

* CLI txtar success appends one keyed map element and one keyed struct element; assert exact literal and receipt.
* CLI txtar refusal for duplicate map key/struct field, unknown struct field, malformed expression, positional literal under an unsupported policy, and repeated literal without path; target file stays unchanged.
* CLI txtar sequential commands replace the declaration, append to its resulting literal, then edit the declaration again; assert exact results after each command. The MCP batch test separately exercises the unsaved overlay with static fixture-known arguments.
* MCP batch statically specifies declaration replacement, append insertion against the known unsaved literal, and a later edit; any zero-based path is paired with the exact fixture-derived revision. The batch does not interpolate prior results. Verify the final tree and diagnostics under configured scope, then make a separate follow-up mutation using the containing `in_declaration`, followed by a standalone lookup to prove freshness.
* MCP batch with a successful edit followed by malformed/ambiguous insertion stops at refusal; the earlier edit remains eligible for publication and the element is absent. Any later success runs in a separate invocation.
* Schema/receipt cases prove `in_declaration` is the holder selector and no fake `symbol` is returned.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_insert_composite_element`:**

> Use this tool instead of replacing a declaration or composite literal to append one supported Go literal element. `in_declaration` selects the holder; `discriminator` and any revision-bound `construct_path` select the exact literal. Supply one `source` element, including its key when keyed; `placement` defaults to append. Pass the live result `file`, `in_declaration`, refreshed `discriminator`, and matching path/revision directly to another `semantic_insert_composite_element` call with a new element. The inserted element and old literal text are evidence; insertion is not a behavior-preserving refactoring.

**Parameter description requirements:**

`in_declaration`: Selector: copy proposed `semantic_inspect_symbol.in_declaration` only after shared inspection exposes that exact holder field. `discriminator`: Selector: full current literal text from `semantic_insert_composite_element.discriminator` only when regenerated after the edit. `source`: Replacement: one element, without enclosing braces or comma; never confuse it with literal selector text. Descriptive literal `kind` does not feed another tool's kind enum.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Go set doc comment

Original handover: RQ-0059. Implementation status: Open.

### Context

Tool receipts and source declarations can be structurally correct while lacking or carrying stale documentation comments. `docs/MANUAL_EDITS.md` ME-0047 records attached lint comments as a manual edit. Whole declaration/body replacement is disproportionate for comment-only changes. Go comments may be doc comments, ordinary comments, build constraints, compiler directives, cgo preambles, or comments attached to AST nodes; treating all as interchangeable risks changing build behavior.

### Research questions

1. Which targets are supported: package/file header, top-level declaration, method/function, type, field, or parameter?
2. What exact comment string format should callers pass, including `//` or `/* */` delimiters and trailing newline policy?
3. How can the tool distinguish documentation text from `//go:`, `// +build`, `//line`, `//export`, cgo, and lint directives?
4. Should an empty string remove only a target's doc comment, and how are detached ordinary comments preserved?
5. How are a file-level comment and a declaration-targeted comment represented without overloading `symbol`?

### Current evidence and scope

* `docs/MANUAL_EDITS.md` ME-0047 identifies attached lint/doc comment changes as outside current focused semantic tools and suggests `semantic_set_doc_comment`.
* Go AST parsers retain comments but attachment and source positions vary; arbitrary text replacement can move comments across declarations or alter build/compiler semantics.
* Existing insertion tools accept declaration source containing comments; they do not provide a target-specific comment setter.
* The pending authoritative main has sourcebuf overlays. Comment updates must compose with earlier uncommitted declaration/move edits and preserve the retained gopls session.

### Outcomes and non-goals

First release should set or remove only a declaration's Go doc comment, or choose a file/package comment as a separately specified target. Keep comment text literal and preserve exact source line endings where possible. Refuse any target region containing directives or ambiguous attached comments. Do not edit arbitrary inline comments, lint suppressions, build tags, cgo preambles, license banners, or comments inside function bodies in the first release.

### Proposed API

Propose `semantic_set_doc_comment` / `set-doc-comment`, Go only, batchable and buffered.

* `file` string: exact workspace-relative Go file.
* `symbol` string: target declaration for declaration scope; use canonical qualified symbol format.
* `scope` enum: proposed `declaration` or `file`; decide if a dedicated file-header mode is safe and discoverable.
* `comment` string: plain text body with no `//`, `/*`, or `*/` delimiters. The engine emits `//` doc-comment lines and handles line breaks consistently. Explicit empty string removes the selected doc comment; missing or null is invalid and never means remove.
* `in_function` string: only if the first release supports nested/local declarations; recommendation is to exclude them initially.
* `expected_revision` string: stale target/comment attachment guard.

`file`, `scope`, and `comment` are required. `scope` has exact values `declaration` and (if accepted) `file`, with no default because the target mode must be explicit. `symbol` is required for declaration scope and forbidden for file scope. Missing `comment` is invalid; explicit empty string means remove the selected doc comment. Since current `CheckParams` rejects empty required strings and CLI `rawParams` omits blank flags, research a presence-aware CLI flag/raw-parameter path and an allow-empty string contract for this parameter so `--comment=''` reaches the handler. MCP must preserve absent, null, and empty as distinct states. `expected_revision` is optional initially and required for receipt-based follow-up. No import option applies.

A file scope has no declaration `symbol`; schema rules must express that without treating a missing symbol as success. `SourceFields` should connect `file` and `symbol` to matching lookup/edit receipts. If a structured comment representation is desired, defer until recursive schema support.

### Continuation receipt proposal

For declaration scope, return `file` and live `symbol`; for file-header scope, return `file` and `scope` only. Return the exact plaintext `comment` body so it can be copied directly into a later call; the model never has to reconstruct comment delimiters. A successful removal has empty current comment, but removed text is historical evidence.

```json
{
  "file": "pkg/config.go",
  "symbol": "Config",
  "scope": "declaration",
  "comment": "Configuration for the service.",
  "expected_revision": "opaque-revision-token",
  "changed_files": ["pkg/config.go"],
  "diagnostic_status": "unchanged"
}
```

If a file scope is accepted, omit `symbol` and return `scope: file`. Never return a file header as a declaration symbol.

### Semantic hazards and invariants

* Build tags and `//go:` directives affect compilation/linking and must never be inserted, removed, or moved by doc-comment operations.
* `//line`, `//export`, `//go:linkname`, `//go:generate`, and cgo preamble adjacency need explicit recognition/refusal policy.
* A comment may attach to a grouped declaration, a field, a method, or the next declaration; preserve all non-target comments byte-for-byte.
* Empty `comment` is an explicit removal request; absent/null input is invalid, preserving distinction between removal and missing data.
* Multiple matching symbols, generated files, malformed source, detached comment clusters, and stale revision refuse before mutation.
* Documentation text can contain directive-looking lines; reject directive prefixes conservatively rather than normalizing or stripping them.
* Formatting should not rewrite unrelated comments. Validate the resulting AST and publish through the overlay only.

### Ordered milestones

1. Decide declaration-only versus file-header support and define comment input encoding/newline semantics.
2. Build a directive/comment ownership matrix from Go syntax and repository fixtures.
3. Specify empty versus missing behavior, stale revision, and exact receipts under ADR-0055.
4. Implement node selection and comment-range edits with byte-preserving unrelated source, buffer-aware writes, and refusal before apply.
5. Add CLI/MCP registry, schemas, Go capability, and literal CLI txtar contracts; consider file headers only in a follow-up milestone.

### Public behavior tests to add

* CLI txtar success sets a declaration doc comment and replaces an existing one; assert exact text, adjacent comments, and receipt.
* CLI txtar `--comment=''` explicitly removes the doc comment; an omitted flag refuses unchanged. MCP cases separately assert omitted `comment`, `comment: null`, and `comment: ""`; only the explicit empty string removes the comment. Malformed, ambiguous, or stale targets refuse unchanged in both interfaces.
* CLI txtar refuses targets adjacent to build tags, `//go:linkname`, `//export`, cgo preambles, and detached ordinary comments; assert directive/comment bytes remain exactly unchanged.
* CLI txtar test file-level header separately if accepted; cover package docs and license comment boundary without moving build constraints.
* MCP batch: statically specify insertion of a declaration, setting its doc comment by the known `symbol` in that unsaved result, and a later edit. No receipt is interpolated inside the batch; use fixture-derived selector and revision literals. A separate follow-up mutation consumes the declaration `symbol`, then a standalone lookup proves the target remains live.
* MCP batch with a successful declaration edit followed by comment-selection refusal stops at refusal; the earlier edit remains eligible for publication, the wrong node stays untouched, and no intermediate formatting is published. Later edits run separately.

### Model-facing description handover

Apply [RQ-0027's description audit](RQ-0027-preventing-read-files.md#model-facing-description-audit) and the shared batch contract. The following is candidate release wording, not an assertion of current support. Publish only the implemented and proven scope; add broader cases as their gates pass.

**Tool description for `semantic_set_doc_comment`:**

> Use this tool instead of replacing source declarations to set or remove an owned Go documentation comment. `scope=declaration` requires its live `symbol`; file scope, if supported, forbids `symbol`. Supply plain `comment` text without delimiters. An explicit empty string removes the comment; missing or null is invalid. The engine preserves directives or refuses ambiguous ownership. Pass live `file`, `scope`, declaration `symbol` when applicable, and current `comment` directly to another `semantic_set_doc_comment` request. Pass a declaration target to `semantic_inspect_symbol` or `semantic_rename`; removed comment text is evidence.

**Parameter description requirements:**

`symbol`: Target: live declaration copied from compatible `semantic_lookup.symbol` with its `file`; forbidden in file mode. `scope`: Selector: required explicit supported target mode, copied from `semantic_set_doc_comment.scope` when repeating that mode. `comment`: Replacement: caller text or current `semantic_set_doc_comment.comment`; empty is a present value, not omission. Use only the current result, never historical removed text, as its automatic source.

For every remaining parameter, the shared description templates require its role, exact value format, required/default behavior, and named producer when it is receipt-fed. Caller-authored new names, source, mappings, and policy choices need no fabricated `SourceFields` edge. Proposed inspection projections, nested receipts, and revisions must be implemented before their copy instructions are advertised.

**Description acceptance:** inspect generated `tools/list` and CLI help, not just registry summary strings. Prove one compatible producer→input edge and one live result→next-tool edge with verbatim values, assert historical/stale fields are absent from copy guidance, and verify the description does not promise broader support or intermediate disk publication. Batch child parameters stay static; copying a result means a subsequent request, not receipt interpolation.

## Completion and continuation

Update each tool section with its implemented scope, reviewed commits, CLI txtar evidence, batch behavior, refusals, and remaining expansion gates. Record shared contract changes once above, with their ADR and index updates. Resolve this RQ only when every advertised delivery in the chosen scope has passed the worker fix/review cycle and required checks, has merged into main, and any deferred scope is explicitly documented. Consolidation alone closes none of these implementation gates.
