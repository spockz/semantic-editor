<!-- WHY: Put detailed output, diagnostics, and staging protocols behind one on-demand reference while keeping the main operation router concise. -->
# Results, Diagnostics, and Publication

## Typed results and continuations

Operations with declared output contracts return a typed `result` object. Copy target and selector fields only from its declared continuations, keeping values from the same record together. Formatted text is presentation, not a parallel source of continuation fields. Mutations return `edit_receipt`; buffered batch children report `applied_working` and omit live continuations. A batch failure can retain earlier successful edits, so inspect its persisted diff before retrying. Standalone AST edits prepare edits, formatting, and imports in one source buffer and publish the prepared result once.

Treat `structuredContent` as the canonical full result and potentially model-visible when the client forwards it. Every tool's text carries the outcome and exact next-step guards: live selectors and revisions, ambiguity choices, refusal or failure reason, partial or unknown publication state, and diagnostic counts with a copyable query. Large source bodies, reference and outline inventories, diagnostic arrays and pages, diffs, read sets, logs, metric records, and feedback artifacts appear in structured fields. Text names those fields and page cursors without serializing them again. A compact text view never makes a buffered child continuation live or implies clean verification. Client model visibility varies; Copilot desktop 1.1.29 behavior has not been established by this project.

## Diagnostics and timing

For timing diagnosis, call `semantic_metrics` with `limit: 1` after the operation; see the [timing reference](metrics.md) for phase names, interpretation, and trace regions.

Rename reports `active_count` for existing workspace diagnostics and change evidence in `diagnostics`. Prefer `diagnostic_report`: it supplies the remaining captured count and a copyable `semantic_diagnostics` call. Set `limit` (default 50, maximum 100), follow `next_offset`, and select `category` (`active`, `introduced`, `resolved`, `suggestions`) or `contains` for desired details. Read page `items` from the structured result; text gives the count and cursor. Pages reuse the same immutable report without analysis. Deferred or unavailable does not mean zero; partial counts cover completed scopes only. IDs expire on eviction or server restart. Use `semantic_verify` for a fresh report.

Preserve application state, errors, revisions, and opaque stage/step IDs when choosing the next call.

## Staged publication

When the advertised Go or Java tools include retained staging, use `semantic_stage` for edits that span turns and should remain unpublished. Staged reads see the working revision. Copy a mutation's native `stage_receipt.step_id` to `semantic_stage_drop`, or use `semantic_stage_abort` to discard the journal; `semantic_commit` publishes and verifies it. Ordinary `result` continuations are not live while staged. Staging controls require one retained MCP session; their standalone CLI commands fail with a session-required error. Read the relevant language publication guide for conflicts, limits, partial publication, and verification choices.
