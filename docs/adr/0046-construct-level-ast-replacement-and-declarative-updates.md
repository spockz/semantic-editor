# ADR-0046: Construct-Level AST Replacement and Declarative Updates

* **Status**: Accepted
* **Date**: 2026-09-25

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Insert/replace/remove use per-operation kinds; keyed Go map and struct literal elements use insert_construct(kind=element) with exact selectors and typed overlay validation. Movement remains explicit. |
| description | Whole-function body replacement is unnecessarily broad for localized edits, while declaration insertion must refuse collisions rather than emit duplicate names. The public API also needs a stable construct lifecycle across declaration and nested syntax operations. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

Whole-function body replacement is unnecessarily broad for localized edits, while declaration insertion must refuse collisions rather than emit duplicate names. The public API also needs a stable construct lifecycle across declaration and nested syntax operations.

## Decision

The generic lifecycle uses four registry operations: insert_construct, replace_construct, remove_construct, and move_construct. Insert, replace, and remove accept a backend-supported kind; move selects a named declaration by symbol and has no kind parameter. Construct kinds and selector requirements are declared per operation and backend, and MCP schemas are derived from those executable capabilities. This does not imply support for every kind or language.

replace_construct replaces one selected construct in place. A named declaration is selected by symbol and does not require in_function. Go declaration replacement currently includes functions, methods, types, constants, and variables; the new declaration must preserve the selected name and collisions fail before mutation. Nested constructs use in_function, kind, and their applicable discriminator or construct_path.

Go also supports statement/expression replacement. A selector addresses an exact expression slot or a whole statement, and the replacement source is validated for that slot before writing. Overlapping statement and expression candidates require an explicit candidate path. For conditionals, a matching root `if` condition selects the full conditional chain, while a matching nested `else if` condition selects only that nested branch. `kind=else` selects the terminal fallback block. Nested closures remain owned by their enclosing construct and cannot be selected as independent outer-body statements. `discriminator` must match the supported header, condition, or expression exactly; ambiguity reports candidate paths before any write. `remove_construct` removes named declarations, cases, and direct statements. Expression-only removal is refused. `insert_construct` supports supported declaration kinds, fields, cases, and direct-statement anchors; it does not accept overwrite.

semantic_replace_body remains a specialized operation for changing a Go function or method body while preserving its signature. It is also available as the replace:function variant in full or filtered MCP catalogs. Generic whole-function replacement uses semantic_replace_construct.

`insert_construct(kind="element")` appends one keyed element to a typed Go map
or struct literal. `in_declaration` selects its top-level holder and
`discriminator` is the complete exact current literal source. Placement is
`append` only. Duplicate keys or fields, unknown fields, dynamic map keys,
comments, positional literals, ambiguous selection, and stale revisions fail
before mutation. The typed planner validates the package overlay and preserves
its exact source splice without a second import-organizing pass. Struct type
field insertion uses `kind="field"` and `in_type` instead.

`move_construct` explicitly relocates one declaration. Use `semantic_insert_construct` to add a declaration or `semantic_replace_construct` to update it in place. Movement requires a destination and placement, carries no replacement source or overwrite flag, and does not sort declarations automatically.

## Invariants

1. Validate source, selection, declaration identity, and collisions before publication.
2. Ambiguous selection reports usable candidate paths and leaves files unchanged.
3. Whole declaration replacement is selected by symbol without requiring an enclosing function.
4. Expression removal is rejected; direct statement removal is supported where advertised.
5. Move intent remains separate from replacement and insertion; it cannot overwrite a destination declaration.
6. Only implemented per-operation and per-backend construct capabilities are advertised.
7. CLI txtar tests cover each advertised language-operation pair.
8. Mutation publication remains atomic with advancing file modification times.

## Consequences

The generic four-operation lifecycle reduces MCP tool catalog size while the registry keeps specialized variants available for CLI, batch, and opt-in MCP discovery. Narrow replacement and explicit movement preserve sibling code and avoid implicit ordering policy.
