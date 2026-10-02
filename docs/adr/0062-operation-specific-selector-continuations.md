<!-- This decision defines live follow-up selectors for edits whose targets are not symbols. -->
# ADR-0061: Operation-Specific Selector Continuations

* **Status**: Accepted
* **Date**: 2026-10-02

## Context

Some structural operations edit an element selected inside a declaration but do
not target a symbol. Encoding the holder as `symbol` would violate ADR-0055's
target/selector distinction, while widening the generic symbol continuation
would conflate unrelated result shapes.

## Decision

An operation may expose a closed, operation-specific `selector_continuations`
array alongside the generic `edit_receipt`. Each record contains only the
selectors needed for a compatible follow-up and the matching full-file
`expected_revision`. For composite-element insertion, the record is
`{file, in_declaration, discriminator, expected_revision}` and only
`semantic_insert_composite_element` consumes it.

Before emitting a record, the finalizer reads the actual final source after
formatting, resolves the original holder, proves exactly one supported literal
is live, refreshes its exact source discriminator, and computes the full-file
revision. It omits the record for buffered `applied_working` results, refusals,
or any outcome where identity or liveness cannot be established. The generic
receipt continues to report publication status, changed files, and the
analysis read set; it does not encode a synthetic symbol.

Input `SourceFields` name the exact selector leaves in the producing
operation's output. Static batch entries still do not interpolate prior
results; a refreshed selector can be copied into a subsequent request after
the batch has finalized.

## Invariants

* Operation-specific selectors do not appear as fabricated `symbol` targets.
* A selector continuation describes current final source, never pre-edit text.
* A construct path is emitted or consumed only with the matching full-file
  revision; unsupported paths are omitted.
* No live continuation is emitted for a buffered batch result.
* The generic edit receipt retains its existing schema and meaning.

## Consequences

* Non-symbol structural edits can expose precise copyable follow-up inputs
  without changing `SymbolTarget` or weakening ADR-0055.
* Each operation owns its selector schema, liveness check, and compatible
  consumer list, so registry and CLI/MCP tests must assert that exact shape.
* A continuation may be omitted when normalization or source changes prevent
  proving a unique live target.
