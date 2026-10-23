<!-- This ADR defines canonical refactoring intent names so planners can route high-level edits consistently while backend capabilities remain language-specific. -->
# ADR-0078: Shared Refactoring Intent Vocabulary and `refactor_*` Tool Family

* **Status**: Accepted
* **Date**: 2026-10-19

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Establish one canonical refactoring name per operation, shared direct/batch validation, canonical output contracts, and language-specific capability gates. |
| description | The operation catalog has accumulated separate names for refactorings with the same user intent, including extract function, extract variable, extract constant, inline, move, and signature change. This makes planner routing depend on backend and implementation history. The capability taxonomy in [RQ-0024](../research/RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md) calls for shared intent names while preserving language-specific contracts. Generic AST operations such as construct insertion and replacement remain separate because they express structural edits rather than refactoring intent. |
| role | contract |
| scope | project |
| area | backend |
| languages | generic |

## Context

The operation catalog has accumulated separate names for refactorings with the same user intent, including extract function, extract variable, extract constant, inline, move, and signature change. This makes planner routing depend on backend and implementation history. The capability taxonomy in [RQ-0024](../research/RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md) calls for shared intent names while preserving language-specific contracts. Generic AST operations such as construct insertion and replacement remain separate because they express structural edits rather than refactoring intent.

## Decision

Adopt `refactor_*` as the only public name family for semantic refactorings in MCP, CLI, registry keys, receipts, and continuation metadata. `refactor_extract_value` accepts a `kind` that selects variable, constant, or field extraction; backend capability contracts determine which kinds each language supports. Go and Java do not currently support field extraction: Go has no verified operation for it, and stock JDT LS produces mutable instance fields with method-local assignments that lack a safe contract for this operation. Rust and Scala also refuse field extraction. Constants use `refactor_extract_value` with `kind: constant`. The catalog also uses canonical operations for extract callable, inline callable/value, change signature, move declaration/file, rename, extract interface, and implement contract.

Direct calls and batch children use the same registered operation contract. Batch execution may stage edits in a shared overlay, but it does not weaken operation validation or backend safety checks.

Each language advertises only variants with a verified implementation and comparable CLI contract coverage. A shared vocabulary does not imply capability parity. New Rust assists and Metals refactorings require separate protocol, edit-capture, lifecycle, and safety investigations before they are advertised.

Scala extraction remains bounded by operation-specific source preflights. Value extraction uses a strict regex preflight for the one documented Int-call shape; callable extraction uses lexical and owner-range guards before capturing and validating native Metals edits for contiguous same-callee call statements. The capability catalog must preserve these separate limits.

## Invariants

* Canonical refactoring operation names use the `refactor_` prefix; generic structural edit and lifecycle operations remain outside that family.
* Refactoring operations have one registered key, CLI command, and MCP name; `refactor_*` is the only public refactoring family.
* MCP discovery, receipts, and continuations use those canonical operation names and parameter vocabulary.
* Direct and batch dispatch share the same operation schema and validation path.
* Backend capability declarations are language-specific and fail closed when edit safety is unproven.
* Every advertised language-operation pair has comparable CLI txtar coverage or is explicitly marked unsupported.
* Scala value extraction and callable extraction retain their distinct bounded source-shape contracts; neither implies general Scala refactoring support.

## Consequences

Canonical names provide a stable intent vocabulary for agents and simplify routing across language backends. The project is unreleased, so the registry exposes no legacy operation-name aliases. Registry, CLI, MCP discovery, receipts, source-field continuations, skill routing, and CLI fixtures use the canonical names. Rust and Scala capability expansion remains gated by independent protocol and lifecycle evidence for each additional operation; existing extraction support remains limited to the verified source shapes documented by RQ-0073 and RQ-0074.
