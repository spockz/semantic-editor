<!-- This ADR records the accepted proof boundary for rewriting private Go field accesses into generated accessors. -->
# ADR-0079: Go Private Field Encapsulation Boundary

* **Status**: Accepted
* **Date**: 2026-10-20

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Rewrite only typed, direct private-field reads and standalone writes whose pointer-accessor calls preserve evaluation order; validate the multi-file overlay and source membership before one buffer apply. Compiler-derived required imports are planned atomically with collision-free aliases. MCP may start in a child directory only when initialization explicitly authorizes the module-root workspace. |
| description | The field-encapsulation handover proposes generating accessors and rewriting references across a Go module. A text-only rewrite or a package-local type check cannot prove that every reference has been handled, that generated methods preserve method sets, or that the new calls preserve Go evaluation and panic order. The initial operation therefore needs an explicit boundary that can be proved from typed source snapshots. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

The field-encapsulation handover proposes generating accessors and rewriting references across a Go module. A text-only rewrite or a package-local type check cannot prove that every reference has been handled, that generated methods preserve method sets, or that the new calls preserve Go evaluation and panic order. The initial operation therefore needs an explicit boundary that can be proved from typed source snapshots.

## Decision

The first implementation supports one direct unexported field on an unexported, non-generic named struct. It preserves the field declaration, storage, tags, layout, and keyed literal initialization. It generates private pointer-receiver accessors with explicit distinct names and the field's compiler-resolved type.

The operation rewrites only direct field reads that are the entire right-hand expression of a single-value assignment or declaration, or the sole expression of a return statement. It rewrites only standalone one-target, one-value `=` assignments. Receivers must be simple identifiers. It refuses any other resolved field use, including compound writes, increment/decrement, address-taking, nested or side-effecting receiver expressions, parallel assignments, and reads embedded in expressions with other effects. This boundary preserves receiver and value evaluation counts and the assignment's receiver-before-right-hand-side order.

Planning enumerates typed references across the module's active Go package graph, including test variants, and matches field identity by declaration position where independent type-check variants do not share object identity. It refuses structs with embedded fields and refuses the selected owner embedded directly or through an alias in any same-package struct. It also refuses accessor names found in any same-package interface or constraint method set, including anonymous interfaces and test variants, because a new method can change interface satisfaction without causing a compile error. Positional literals and unsupported source shapes are refused. A Go file in the owner package directory that is absent from active typed package variants causes refusal; the operation does not guess about inactive build-tag or GOOS/GOARCH variants. Accessor declarations and every caller edit are typechecked together. The planner revalidates all source preimages and observed Go-tree membership, then stages the complete edit set with one `sourcebuf.Buffer.Apply` call.

The current receipt and continuation contract is relative to the authorized caller workspace. The caller workspace must be the Go module root; the operation refuses a complete owner package that extends outside that workspace before applying edits. The MCP process may start in a module subdirectory when initialization explicitly selects and authorizes the module root. A request-scoped workspace narrowed below the module root does not grant ancestor access, and the operation must refuse it rather than widening the root implicitly. `mcp_encapsulate_field_module_root_from_subdirectory.txtar` proves the explicit-root case through a multi-step buffered batch whose sources are outside the process starting directory.

When `auto_organize_imports` is true (the default), the planner derives required package paths and names from compiler-resolved field types, reuses usable destination imports, chooses deterministic collision-free aliases for missing ones, and adds those imports to the accessor destination's in-memory candidate before overlay validation. When false, a missing required import causes refusal before staging. Required imports remain part of the semantic edit in batches even when batch normalization is deferred; the batch-level flag controls final cleanup. The import planner preserves the original analysis preimages and applies one combined edit set.

The operation is Go-only. Public accessors, external API migration, reflection-sensitive use, compound assignments, and generated-file support require separate proof and remain unsupported.

The operation is not idempotent. Repeating a completed request encounters the existing accessor names and refuses without changing source bytes; it does not return a successful no-op receipt. A future idempotent form would need to prove that both accessors and every supported field reference already match the requested transformation.

## Invariants

* Field storage and keyed literal initialization remain unchanged.
* Generated accessor methods use pointer receivers and remain private in this release.
* Only explicitly supported direct reads and standalone writes are rewritten; every other resolved use causes refusal before staging.
* Receiver expressions and assignment right-hand sides retain their original evaluation count and order within the supported shapes.
* All planned files typecheck as one overlay, and source preimages plus Go-tree membership are revalidated before a single buffer application.
* Required external type imports are derived from compiler types and included in the validated candidate when enabled; disabling import additions refuses before staging if a usable import is missing.
* Multiple newly required packages share one reserved-name set; aliases also avoid existing declarations/imports and predeclared identifiers used by the destination.
* Buffered and standalone calls use the same validation and planning contract; a refusal adds no edits from this operation.

## Consequences

The initial operation covers common isolated field reads and writes without promising arbitrary expression rewrites or public API migration. Some valid Go programs will be refused until a later proof covers their evaluation, method-set, reflection, or build-variant behavior. CLI txtars must demonstrate cross-file rewrites, exact preservation of unrelated same-spelled fields, runtime receiver/value order, and fail-closed behavior for every excluded shape.
