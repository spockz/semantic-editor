<!-- Defines consistent tool argument names and copyable provenance for continuation values. -->

# ADR-0055: Tool Parameter Schema Harmonization and Continuation Contracts

* **Status**: Accepted
* **Date**: 2026-09-28

## Context

The MCP tool surface has accumulated naming divergences across input parameters
and result fields as tools were added independently. These divergences force the
model to reassemble or rename values between tool calls, adding inference steps
that the engine has already resolved. They also make the tool surface inconsistent
from the model's perspective, reducing routing precision and increasing context
consumption.

Two related issues compound this:

1. Result fields include data that has no corresponding tool input parameter
   (`offset`, standalone `receiver`, full `before`/`after` diagnostic arrays),
   bloating every tool response with information the model cannot act on
   structurally.

2. Tool descriptions do not state which result fields are reusable as input
   arguments to follow-up calls, nor which are navigation-only hints. The model
   must infer this from cross-referencing multiple tool schemas.

The full audit and design session are recorded in
[RQ-0027 §8](../research/RQ-0027-preventing-read-files.md).

## Decision

### 1. Continuation Field–Parameter Isomorphism

Every field in a structured tool result intended for use as a follow-up tool
argument must match the target tool's parameter name and value format exactly.
No field may require the model to reassemble, rename, or reinterpret its value
before passing it to a subsequent semedit call.

### 2. Schema-Wide Parameter Name Consistency

A logical concept uses the same JSON parameter name in every tool that accepts
or returns it. No two tools use different names for the same concept.

### 3. Parameter Role Distinction: Target vs Selector

Every parameter must have a declared role:

* **Target** — the thing the operation acts on directly. `symbol` is always a
  target. Result continuations echo target parameters back so the next tool can
  address the same unit.
* **Selector** — a scope constraint that narrows where the engine searches for
  the real target. Selectors are not the thing being changed.

`symbol` is reserved exclusively for the primary operation target across the
entire tool surface. Parameters that scope a search without being the changed
unit must use distinct names.

### 4. Canonical Parameter Renames

The following renames resolve all audited divergences:

| Tool | Old name | New name | Rationale |
| :--- | :--- | :--- | :--- |
| `semantic_replace_construct` | `function` | `in_function` | Selector role; not the mutation target |
| `semantic_insert_case` | `func` | `in_function` | Same selector role; unify with replace_construct |
| `semantic_insert_case` | `switch_on` | `discriminator` | Unify with replace_construct's existing name |
| `semantic_insert_case` | `switch_path` | `construct_path` | Unify with replace_construct's existing name |
| `semantic_insert_case` | `anchor` | `target_case` | Distinguishes case selector from symbol-addressed `target_symbol` |

`in_function` accepts the fully-qualified form (`"Server.ServeHTTP"`,
`"(*Client).Do"`) and is a scope selector, not a continuation target. Result
receipts for construct operations return `in_function` so the next call can
address the same scope without re-resolving.

### 5. Diagnostic Delta Serialization

`DiagnosticDelta.Before` and `DiagnosticDelta.After` (full pre- and post-edit
diagnostic lists) are tagged `json:"-"` in both `pipeline.DiagnosticDelta` and
`backend.DiagnosticDelta`. Only `introduced`, `resolved`, `net_delta`, and
`suggestions` are serialized into tool results. The full arrays remain in the
Go struct for internal delta computation.

### 6. LookupResult Field Policy

* `Receiver` as a standalone field is omitted from structured output. The
  qualified `symbol` value already encodes it. Returning both forces the model
  to reconstruct what the engine resolved.
* `Offset` is omitted from structured output. It is an internal Go LSP
  byte-offset artifact with no tool-parameter mapping.
* `Line` and `Column` remain as flat fields in results without a wrapper
  sub-object. They are valid for file-navigation reads but are not semedit tool
  parameters. Their tool and parameter descriptions state this explicitly.

### 7. Tool Description Cross-Reference Requirement

Every tool whose result contains fields that are reusable as semedit tool
arguments must declare this in its description. Navigation-only fields (`line`,
`column`) must be labeled as such. Example for `semantic_lookup`:

> Returns `symbol`, `file`, `kind`, `line`, and `column`. Pass `symbol` and
> `file` to follow-up semedit calls. Use `line` and `column` only for
> file-navigation reads; they are not accepted as semedit tool parameters.

Every input parameter description whose value originates from a prior result
field must name the source tool and field (Direction A cross-reference). Example
for `semantic_rename.symbol`:

> Target: qualified symbol being renamed (e.g. `Server.ServeHTTP`,
> `(*Client).Do`, `ValidateToken`). Copy verbatim from the `symbol` field of
> any preceding `semantic_lookup`, `semantic_rename`, or mutation receipt.

### 8. SourceFields Registry Annotation

`ParameterContract` gains a `SourceFields []string` field. Its value lists the
tool-and-field pairs that feed this parameter (e.g.
`["semantic_lookup.symbol", "semantic_rename.symbol"]`). The schema generator
appends the Direction A cross-reference clause to the parameter description
automatically when `SourceFields` is non-empty. A build-time registry check
verifies that every named source field exists in the referenced tool's output
schema.

### 9. Existing-Declaration Selector Provenance

For an existing declaration, an agent reuses a suitable result record and
copies its selector values verbatim from discovery or the latest valid
mutation receipt; it performs discovery only when no suitable record is
available. A record is
suitable for the declared consumer and target/selector role, workspace root,
backend, and current view, with no known invalidation. Copy companion
`file`, owner/target, and `expected_revision`
values from that same record when present. Never reconstruct qualified names,
owner names, paths, or revisions from prose or combine values from separate
records. Creation names have no mandatory discovered-name producer; callers
choose their intent in the required input format.

The inspection projection demonstrates this contract with one
`result.continuations[]` object per captured match. `file`, `symbol`, and
`expected_revision` are required together; applicable Go owner selectors such
as `in_function`, `in_type`, and `in_declaration` are optional on that same
object. Copy `expected_revision` from the typed field without renaming the
legacy `revision` in inspection text. Only declared consumer paths are
reusable. This rule does not require another caller lookup when a suitable
record exists; guarded operations may internally re-inspect and validate it.

Whether to enforce provenance at runtime, and the explicit discovery,
creation, external-client, and recovery modes, remain open in
[RQ-0027](../research/RQ-0027-preventing-read-files.md#existing-declaration-selector-copying-and-enforcement-2026-10-02).

### 10. Critical Information in Text Responses

Every tool-result response, including an `isError` result, must also put the
information needed to decide and continue safely in its textual `content`.
Essential information must not be available only in `structuredContent`. Put
it before large source bodies, diffs, or logs. JSON-RPC protocol errors are a
separate response form and do not acquire a tool-result schema.

* The outcome and any failure, refusal, limitation, or unresolved ambiguity.
* Known applied changes and verification status, including partial application
  on an error. If application state is unknown, say so; `isError` alone does
  not establish that no changes were applied.
* Available live continuation records, preserving exact parameter names,
  values, nesting, and same-record companions such as `file`, `symbol`, owner
  selectors, and `expected_revision`. Label navigation-only fields as such,
  separately from copyable arguments. Do not rename a legacy `revision` field
  to manufacture a guard.
* For result-oriented utilities, the requested result defined by the tool
  contract, such as measurements from `semantic_metrics`, rather than only a
  count or acknowledgement. Ordinary calls retain ADR-0052's timing policy.

Text and structured essentials must come from the same validated result
projection. Never reconstruct a new symbol from the requested name, invent a
revision, or advertise a deleted, stale, or unproven target as a live
continuation. Confirm a mutation's final revision only from its final relevant
view; live targets must resolve in that same view and revision. Batch
intermediate results are not final continuations. Historical evidence must be
labeled separately. State whether ambiguous candidate lists are complete or
truncated, and how to narrow or retrieve more only when that route is
supported; do not silently cut a selector or companion value.

The shared essential prefix uses the compact JSON encoding in ADR-0042.
Declared top-level continuations may be optional: absence means that no
copyable argument was established. Omission is permitted only when the
registered continuation contract marks the field optional; a present value
still receives the same schema validation. Never fill an absent continuation
with an empty or inferred selector. Consumers copy a record only when its
required same-record companions are present.

The whole-tool implementation review and repair evidence are in
[RQ-0027](../research/RQ-0027-preventing-read-files.md). Presentation budgets
remain open. Clients control forwarding and truncation; the server guarantees
essential content and ordering, not its eventual visibility to a model.
ADR-0042 defines the structured-JSON compatibility boundary and private known
error projections.

## Invariants

* `symbol` is exclusively a primary operation target. No tool uses `symbol` as
  a scope selector.
* `in_function` is exclusively a scope selector for construct operations. No
  other tool uses this name.
* `DiagnosticDelta.Before` and `.After` must never appear in serialized tool
  output.
* `LookupResult.Offset` and standalone `LookupResult.Receiver` must never
  appear in serialized tool output.
* `SourceFields` entries must resolve to existing output fields; broken
  cross-references fail the build.
* Existing-declaration arguments are copied from one suitable result record;
  companion values never come from other records or reconstruction.
* Text exposes outcome, failures/refusals/limitations, ambiguity and candidate
  completeness, known application and verification state, and available live
  continuations before bulk payloads. Its shared essential projection matches
  structured output exactly in values, nesting, and record ordering.
* CLI parameter names (`CLIName`) and JSON parameter names (`JSONName`) for
  the renamed parameters must be updated atomically with the rename.
* All txtar test cases referencing renamed parameters must be updated in the
  same change as the wire-layer rename.

## Consequences

* **Positive**: Model argument construction becomes a verbatim copy from prior
  results; no reassembly or inference. Tool descriptions are self-documenting
  as a consistent I/O graph. Diagnostic result payloads shrink by two full
  diagnostic list serializations per mutation. `SourceFields` keeps
  cross-reference text consistent as tools evolve. Text-only clients retain
  the information needed for a safe next call.
* **Negative**: `SourceFields` build check requires that output schemas be
  machine-readable, which ADR-0042 already establishes. Parameter renames
  require txtar fixtures to be updated in the same change. Essential fields
  appear in both representations; shared projection and bounded presentation
  must prevent divergence and unnecessary bulk duplication.
