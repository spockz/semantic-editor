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
* CLI parameter names (`CLIName`) and JSON parameter names (`JSONName`) for
  the renamed parameters must be updated atomically with the rename.
* All txtar test cases referencing renamed parameters must be updated in the
  same change as the wire-layer rename.

## Consequences

* **Positive**: Model argument construction becomes a verbatim copy from prior
  results; no reassembly or inference. Tool descriptions are self-documenting
  as a consistent I/O graph. Diagnostic result payloads shrink by two full
  diagnostic list serializations per mutation. `SourceFields` keeps
  cross-reference text consistent as tools evolve.
* **Negative**: `SourceFields` build check requires that output schemas be
  machine-readable, which ADR-0042 already establishes. Parameter renames
  require txtar fixtures to be updated in the same change.
