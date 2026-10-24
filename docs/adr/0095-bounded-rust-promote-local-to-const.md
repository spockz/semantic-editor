<!-- WHY: Records the separate native assist contract and the deliberately small source proof for promoting one Rust scalar local. -->
# ADR-0095: Bounded Rust Promote Local to Constant

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Promote one fixed-width integer local to a same-function constant through rust-analyzer. |
| description | Accept only one immutable simple local initialized by a range-valid decimal literal, with every use as the direct value argument of its immediately following call; prove the native edit and whole-file result exactly. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | rust |

## Context

The pinned rust-analyzer `promote_local_to_const` assist changes `let` to `const` and renames the binding to uppercase snake case. Its lazy identity contains an action-list ordinal, and the resolved result omits the original action `data`. Blind use is unsafe: a generated-name collision can turn a local binding into a refutable constant pattern, and promoting a `Drop` value changes destruction timing. These semantics differ from Rust extract-constant, which preserves the local alias.

## Decision

Expose this assist as the separate canonical operation `refactor_promote_local_to_const`. Restrict it to one immutable, explicitly typed local with a lowercase snake-case name, a fixed-width primitive integer type, and a nonnegative decimal literal proven to fit that type. The local must be the only declaration in a simple zero-argument function body, followed immediately by one direct call whose sole argument is that local. Require exactly two source occurrences of the local and refuse any source occurrence of the generated uppercase name.

Select the lazy action by title, `refactor` kind, and the `promote_local_to_const:Refactor:<ordinal>:` identity. Bind resolution to the original action even though returned `data` is absent. Accept only a versioned same-file edit whose complete, independently derived result changes exactly `let` to `const` and renames the declaration and direct call argument. Publish through the existing trusted source-buffer lifecycle.

## Invariants

* Only fixed-width `i8` through `i128` and `u8` through `u128` types are eligible; type shadows, unresolved names, and non-decimal or out-of-range literals refuse.
* The complete source is scanned for generated-name collisions and unexpected or extra uses; macros, attributes, imports, comments, strings, references, mutation, aggregates, destructuring, and ambiguous scope refuse.
* The local has exactly one value use, as the sole argument of its immediately following direct call. No custom or `Drop` type can enter the accepted shape.
* The server edit must be versioned, same-file, and byte-for-byte equal to the independently computed whole-file result before publication.
* This operation remains distinct from `refactor_extract_value` with `kind=constant`; no general assist fallback or alias is provided.

## Consequences

Agents can request this narrow promotion explicitly while broader Rust local promotion remains unsupported. The conservative source shape gives up many valid rust-analyzer cases to avoid unsound constant-pattern resolution and destruction-order changes. Standalone success/refusal and buffered-batch CLI txtars, focused runtime/compiler coverage, and a clean independent review are required for the integration checkpoint.
