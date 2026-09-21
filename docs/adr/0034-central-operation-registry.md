# ADR-0034: Central Operation Registry

Status: Accepted
Date: 2026-09-18

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
derive from registered `Def` values. Registration is the act that makes an
operation invokable, documented, and tested.

Typing contract: each def pairs typed `Parse` (raw map to request) with typed
`Run` handlers in `Def[Req,Res]`; type erasure lives only inside the
`Register` closure. Handlers are per language
(`Handlers map[LanguageID]func(ctx, CallContext, Req) (Res, error)`), and
multiple languages share one func value where logic is identical. A lone
`LanguageAuto` handler marks a language-independent operation invoked without
backend language resolution. The registry never imports backend-level
dispatch; backend leaves stay typed underneath.

Presentation stays at the edge: CLI output shapes, MCP error prefixes, and
validation message formats live in ingress adapters, driven by def metadata
where uniform (required-parameter messages derive from contracts) and by
explicit quirk tables where history demands it (rename rendering, undo
conflict mapping). `semantic_reload` stays hand-wired: reply-before-reexec
cannot fit `Dispatch`'s return-a-value contract.

Batch entries recurse through `Dispatch` with `InBatch`/`DeferImports`
context: nesting stays rejected, import organization stays deferred to one
end-of-batch pass, and batch rename now travels trust enforcement plus
diagnostic deltas like standalone rename (prior bypass read as accidental).

## Invariants

- One registration per operation; unregistered names are rejected identically
  by CLI, MCP, and batch dispatch.
- `Parse` output flows into `Run` input with full compile-time checking;
  dispatch-site exhaustiveness is enforced by contract tests, not the compiler.
- Backend `Capabilities()` remains the enforcement source for registry-level
  ops; engine and workspace ops declare their languages explicitly.
- Trust applies to workspace-bound external sessions; one-shot local
  toolchain invocations (including network-touching `go get`) operate under
  ambient authority, with `LevelBuild` as the disclosure mechanism.
- Interface names are signals, not contracts: renames (`semantic_lookup`,
  `semantic_add_build_dependency`) ship with mechanical reference updates and
  no compatibility shims.

## Consequences

- Adding an operation means writing one def plus ingress-neutral tests; CLI,
  MCP, docs, and batch coverage follow by construction.
- `internal/capability` keeps only transfer shapes for docgen; backend
  `CapabilityMatrix` methods and the cross-literal conformance test are
  deleted as redundant.
- The lookup missing-symbol validation message is uniformized as a deliberate
  micro-fix (the call always failed; only the text changes).
- Follow-ups outside this decision: a trust gate for build-level operations
  was considered and rejected (recorded rationale above); isolated-batch
  atomicity for multi-edit file creation remains evaluated under ADR-0032.
