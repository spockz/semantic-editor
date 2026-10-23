# ADR-0034: Central Operation Registry

* **Status**: Accepted
* **Date**: 2026-09-18

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Typed registrations, enum prerequisites, request variants, and exact broken-code bitmaps drive CLI help, enabled MCP descriptions, docs, and generated test cases; feature packages retain typed execution lifecycles. |
| description | Semantic operations were dispatched from three unconnected sites: Cobra handlers in `main.go`, a `switch` in `internal/mcp/server.go`, and a second `switch` in `internal/mcp/batch.go`. Capability metadata lived in a fourth place (per-backend `CapabilityMatrix` literals consumed by `cmd/docgen`), and batch reimplemented rename without trust checks or diagnostic deltas. Any new operation required coordinated edits across all sites, and documentation drifted from behavior by construction. |
| role | architecture |
| scope | project |
| area | mcp |
| languages | generic |

## Context

Semantic operations were dispatched from three unconnected sites: Cobra
handlers in `main.go`, a `switch` in `internal/mcp/server.go`, and a second
`switch` in `internal/mcp/batch.go`. Capability metadata lived in a fourth
place (per-backend `CapabilityMatrix` literals consumed by `cmd/docgen`), and
batch reimplemented rename without trust checks or diagnostic deltas. Any new
operation required coordinated edits across all sites, and documentation
drifted from behavior by construction.

## Decision

Establish `internal/operation` as the single registry owning handler and
metadata per operation. Cobra commands (`internal/cli`), MCP `tools/list`
schemas plus dispatch, batch entries, and the docgen capability matrix all
derive from registered entries. Typed `Def` values supply handlers and parameter
contracts; the registry-owned language contract catalog supplies each implemented
pair's analysis scope, preconditions, capabilities, and existing-error tolerance.
The production registry rejects missing or incomplete language contracts.
Registration makes an operation invokable, documented, and tested.

`internal/operation/language_contracts.go` owns the language contracts, and
`internal/capability.OperationContract` carries their typed transfer shape.
Each contract contains exact request variants with closed analysis and
prerequisite enums, capability and guarantee flags, and broken-code support
bitmaps. Variant selectors distinguish file/scope selection, same/cross-package
movement, safe deletion, inferred/explicit packages, normalization/check-only,
and batch behavior. Variant lookup uses exact equality, not flag containment.

`CodeFaults` contains six independent flags: target-file syntax/type errors,
syntax/type errors in another file of that package, and syntax/type errors in
another package. `CombinationSet` contains one bit per exact fault mask:
`uint64(1) << faults`. Supported, rejected, and conditional sets must be
disjoint; omitted combinations are unknown. Conditional support carries a
closed recovery-condition enum. Healthy source must be supported when all
non-source prerequisites hold. A support classification promises the variant's
scoped guarantees, not compilation of the entire project. Rejection does not
imply zero writes unless the operation's publication contract says so.

`Cases` enumerates all 64 classifications for fixture generation and fails for
invalid declarations. Generated CLI scenarios exercise supported and rejected
combinations; conditional cases require explicit control of their recovery
condition, and unknown cases cannot supply an oracle. No subset or superset
closure is inferred, including when two individual faults are supported.
CLI long help and MCP descriptions render them through `Entry.Description`;
MCP discovery renders only enabled concrete languages plus applicable auto
dispatch. The capability matrix carries the same contract into generated docs
for all implemented languages. Lone auto handlers for workspace snapshot/undo
are language-independent capabilities. Auto verification selects source coverage
and project hooks; it does not advertise unsupported concrete language backends.

These contracts describe the implemented analysis and checks. They do not add
a clean-project compilation gate or automatically reject a request based on
descriptive metadata. Handler safety checks remain the execution authority.
Package/module type preflight, target parsing, LSP analysis, normalization, and
diagnostic coverage remain distinct requirements, as clarified in
[ADR-0002](0002-dual-engine-architecture.md).

Execution code is grouped into feature packages where a feature has a cohesive
capture, validation, or publication lifecycle. `internal/contractedit` owns
typed contract-edit execution, including source capture, rendering, test
preflight, read-set checks, and checked publication. `internal/operation`
retains request parsing, schemas, dispatch, presentation, and receipt
finalization. `internal/navigation` owns backend lookup and rename execution,
including post-rename identity proof. `internal/verification` owns project-plan
discovery, trust checks, normalization, language-server verification, and
coverage results. Feature packages accept typed request DTOs and backend
services; they do not import `internal/operation` or receive its `CallContext`.

Typing contract: each def pairs typed `Parse` (raw map to request) with typed
`Run` handlers in `Def[Req,Res]`; type erasure lives only inside the
`Register` closure. Handlers are per language
(`Handlers map[LanguageID]func(ctx, CallContext, Req) (Res, error)`), and
multiple languages share one func value where logic is identical. A lone
`LanguageAuto` handler marks a language-independent operation invoked without
backend language resolution. The registry never imports backend-level
dispatch; backend leaves stay typed underneath.

Presentation stays at the edge but is standardized: CLI commands derive their
name, summary, flags, validation, result formatting, and error shape from the
registered def. MCP derives its schema and dispatch from the same def. No
historical CLI output or positional-argument compatibility adapters remain.
`semantic_reload` stays hand-wired: reply-before-reexec cannot fit
`Dispatch`'s return-a-value contract.

Batch entries recurse through `Dispatch` with `InBatch`/`DeferImports`
context: nesting stays rejected, import organization stays deferred to one
end-of-batch pass, and batch rename now travels trust enforcement plus
diagnostic deltas like standalone rename (prior bypass read as accidental).

## Invariants

* One registration per operation; unregistered names are rejected identically
  by CLI, MCP, and batch dispatch.
* Every production operation has one complete language contract per handler;
  unsupported language pairs have no advertised contract. CLI help, enabled MCP
  descriptions, and generated documentation project that same metadata.
* Production contracts reject unknown enum values, duplicate variant selectors,
  overlapping fault sets, missing recovery conditions, and unsupported healthy
  cases. Descriptions are generated from these values rather than independent
  prose declarations.
* `Parse` output flows into `Run` input with full compile-time checking;
  dispatch-site exhaustiveness is enforced by contract tests, not the compiler.
* Backend `Capabilities()` remains the enforcement source for registry-level
  ops; engine and workspace ops declare their languages explicitly.
* Trust applies to workspace-bound external sessions; one-shot local
  toolchain invocations (including network-touching `go get`) operate under
  ambient authority, with `LevelBuild` as the disclosure mechanism.
* Interface names are signals, not contracts: renames (`semantic_lookup`,
  `semantic_add_build_dependency`) ship with mechanical reference updates and
  no compatibility shims.

## Consequences

* Adding an operation means writing one def, its language contracts, and
  ingress-neutral tests; CLI, MCP, docs, and batch metadata follow from the
  registry.
* CLI contracts are intentionally registry-defined rather than
  backward-compatible with historical command-specific spellings or output.
* `internal/capability` keeps typed contract shapes, validation, and shared
  rendering for public surfaces and generated tests; backend
  `CapabilityMatrix` methods and the cross-literal conformance test are
  deleted as redundant.
* The lookup missing-symbol validation message is uniformized as a deliberate
  micro-fix (the call always failed; only the text changes).
* Follow-ups outside this decision: a trust gate for build-level operations
  was considered and rejected (recorded rationale above); isolated-batch
  atomicity for multi-edit file creation remains evaluated under ADR-0032.
