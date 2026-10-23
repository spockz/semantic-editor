<!-- WHY: Records the shared compiler-proof and publication boundary for supported cross-package Go declaration moves. -->
# ADR-0102: Bounded Go Cross-Package Declaration Moves

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Move supported exported Go declarations between existing packages in the same main module while preserving compiler-proven identity, references, and initialization behavior. |
| description | One `refactor_move_declaration` contract covers bounded type-and-method groups, constants, structural aliases, variables, and free functions. Each declaration form has explicit source, type, reference, destination, test, and initialization proofs; unsupported shapes fail before publication. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

The Go declaration move operation already supports bounded same-file, same-package, and cross-package movement of exported declarations. Every cross-package move must preserve source bindings and package initialization behavior, but the required proof depends on the declaration form: types carry method and interface identity, constants carry exact values, aliases carry unaliased type identity, and variables carry shared storage identity. A successful overlay typecheck alone cannot establish these properties: private method or field identities, shadowed predeclared names, and initialization order can change while compilation succeeds. Keeping the variants in separate ADRs obscures their shared operation and publication boundary.

## Decision

Extend `refactor_move_declaration` with the following cross-package forms. The source and destination must be existing production files in distinct packages of the same main module. Preserve unrelated declarations, rewrite references by compiler object identity, construct one candidate overlay, and validate it before publication.

* **Defined type and methods:** Move one exported, non-alias, non-generic scalar or non-embedded struct type with all direct exported value and pointer methods co-located in the selected production file. Preserve receiver forms, method selections, and value and pointer interface relations.
* **Constant:** Move one exported, standalone, ungrouped constant with one name and value. Its type must be predeclared basic and its compile-time expression must not depend on `iota`, package declarations, imports, or named types. Preserve its exact compiler type and value.
* **Structural alias:** Move one exported, standalone, non-generic alias whose type expression uses only predeclared type names. Require the candidate to retain exact unaliased type identity. Package-defined or imported type dependencies and package-sensitive private anonymous members refuse.
* **Variable:** Move one exported, standalone, ungrouped package variable. With no initializer, its type expression may use only predeclared type names and the candidate must preserve exact type identity. With an initializer, require a predeclared basic scalar type and one compile-time constant initializer. Preserve the variable's shared storage identity across reads, writes, and address-taking uses.

Validate typed identifier bindings, destination imports and collisions, exact value or type identity, method selections and interface relations where applicable, and initialization effects across the captured packages before publication. For variables, run the conservative main-module initialization proof even when imports do not change. Refuse when the source set, package snapshot, binding map, test references, or initialization proof is incomplete or ambiguous. In particular, refuse package-test references that cannot be mapped safely, private direct methods, package-sensitive anonymous private fields, changed initialization order that cannot be proven safe, dependency cycles, and any destination name capture.

## Invariants

* The moved type and every direct `T`/`*T` method must be discovered from compiler objects and reconciled with declarations in the selected production file. Aliases, generic types, embedded structs, methods in sibling/test/constrained files, and private methods refuse.
* Constants must retain exact compiler type and value and cannot depend on `iota`, package/import declarations, or named types. Aliases and zero-value variables must use only predeclared type names and retain exact (for aliases, unaliased) type identity. Initialized variables must use a predeclared basic scalar type and one compile-time constant initializer.
* Every moved identifier use must retain its source object identity or map to the corresponding relocated declaration. Variable reads, writes, and address-taking uses must continue to refer to one shared storage location.
* Captured interface implementation relations for both `T` and `*T` remain equivalent. Exported anonymous fields and interface methods retain structural identity; package-sensitive private members in anonymous types refuse.
* Initialization behavior must be proven unchanged, including for variable moves with unchanged imports. Missing or dynamic initialization evidence, `init` functions, panic-capable initializers, import cycles, or changed ordering that cannot be proven safe refuse before publication.
* Any package-test reference, inactive or untyped source mention, or incomplete source evidence that cannot be mapped safely refuses. External consumers outside the captured main module are not covered.
* No compatibility promise is made for moved type nominal identity, reflection names, or unknown external consumers. New files, resources, and cross-module moves are outside the contract.
* Standalone and batch execution validate the complete candidate before source-buffer publication and preserve the existing operation receipt contract.

## Consequences

Agents can move supported Go declaration forms through one operation and one publication boundary when the compiler snapshot proves the affected module behavior. Type, constant, alias, and variable forms retain distinct fail-closed evidence requirements without fragmenting the shared cross-package move decision. Unsupported package tests, private identity, initialization, and incomplete source-set cases fail closed. Broader file, resource, and cross-module moves require separate evidence and decisions. This decision narrows and extends the Go contract under [ADR-0049](0049-ast-construct-taxonomy-and-mutation-lifecycle.md); it does not change the shared public operation vocabulary in [ADR-0078](0078-shared-refactoring-intent-vocabulary-and-naming.md).
