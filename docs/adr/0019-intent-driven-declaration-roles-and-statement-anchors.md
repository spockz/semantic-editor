# ADR-0019: Intent-Driven Declaration Roles and Statement Anchors

* **Status**: Accepted
* **Date**: 2026-09-17

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Generic insertion resolves direct statement anchors; repeated direct matches require reported candidate paths and matching source revisions, and expressions do not participate in insertion selection. |
| description | Insertion intent spans top-level declarations, fields, switch cases, and statements anchored inside a function. Separate public tools for each syntax kind increase the catalog without changing the core operation: add one validated construct at a selected location. Agents also need deterministic ambiguity reporting when a selector matches more than one node. |
| role | principle |
| scope | source-code |
| area | backend |
| languages | go |

## Context

Insertion intent spans top-level declarations, fields, switch cases, and statements anchored inside a function. Separate public tools for each syntax kind increase the catalog without changing the core operation: add one validated construct at a selected location. Agents also need deterministic ambiguity reporting when a selector matches more than one node.

## Decision

semantic_insert_construct (insert-construct) is the generic insertion operation. It requires a backend-supported kind and source snippet. Operation-specific backend capabilities determine which kinds and selectors the active language can execute. Current Go kinds include declarations, fields, switch cases, and statement/expression; this taxonomy does not claim support in another language.

Declaration and field insertion use their relevant placement contract. in_type selects a field owner, target_symbol optionally selects a neighboring field, and an existing declaration collision fails before mutation. Insert operations have no overwrite parameter. Whole declaration updates use semantic_replace_construct with a declaration symbol and kind. semantic_scaffold_file retains its separate file-level overwrite option.

For Go statement insertion, in_function selects the owner and kind=statement/expression selects insertion intent. anchor accepts:

* start: before the first direct statement.
* end: after the final statement, except before a terminal return or panic.
* before_return: before the final direct return.
* before or after: adjacent to a statement selected by discriminator; these values require a discriminator.

Insertion resolves only direct statements in the selected function body. A unique direct statement anchor needs no path; repeated direct statements require the `construct_path` and matching `expected_revision` reported by an ambiguity diagnostic. Expression candidates do not participate in insertion selection. The source must parse as one statement. Invalid snippets and unresolved or ambiguous selectors fail before mutation.

Specialized insertion operations remain registered for CLI and batch use. MCP discovery exposes generic tools by default; --tool-profile=full or matching --expose-tool-variant patterns disclose specialized variants. Catalog filtering does not alter registry capabilities or dispatch.

## Invariants

1. Insertion validates source and selectors before writing.
2. Insertion never silently replaces a declaration or accepts overwrite.
3. Ambiguous node selection reports usable candidate paths and leaves the workspace unchanged.
4. Statement anchor semantics are limited to behavior implemented and tested by the language backend.
5. All public mutation writes use atomic publication and advancing modification times.
6. Each advertised language-operation pair has CLI txtar contract coverage or an explicit unsupported capability.

## Consequences

Generic construct insertion provides one public semantic entry point while preserving specialized commands and batch operations. Statement anchors avoid whole-body replacement for local insertions, and explicit candidate paths prevent implicit selection.
