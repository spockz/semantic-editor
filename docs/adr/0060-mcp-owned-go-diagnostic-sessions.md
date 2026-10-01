# ADR-0060: MCP-Owned Go LSP Sessions

## Status

Accepted.

## Date

2026-09-30

## Context

Go verification uses gopls `textDocument/diagnostic` pull requests. Starting and
stopping a server for every scan adds process initialization and shutdown to every
tool response. A copied eight-file rename fixture measured about 1.2 seconds per
fresh verification, with about 1 second in the shutdown response. MCP already
provides a persistent process associated with one agent workspace.

ADR-0054 requires diagnostics for the committed revision and selected scope.
Retaining a server therefore also requires explicit buffer synchronization and
invalidation of package and module state.

## Decision

Let each Model Context Protocol (MCP) server instance own a lazy Go LSP
session manager. Reuse one gopls process per enclosing Go module, including when
requests select different subdirectories of that module. Separate modules in one
workspace have separate processes. Command-line interface (CLI) rename invocations use the same protocol and edit
policy with a temporary session closed when the invocation ends. Go rename uses
`textDocument/prepareRename` and `textDocument/rename` on the retained session.
The Go adapter owns versioned WorkspaceEdit validation and atomic publication,
following ADR-0029; transport and lifecycle code do not apply edits.

Resolve the enclosing module with the same project-local Go environment and
`GOWORK=off` policy used for diagnostics. Canonicalize scan paths so symlink aliases
do not create distinct document URIs or sessions. Refresh the package graph on each
scan. Keep the module's Go sources and imported local replacement sources open;
read their current disk contents before requesting diagnostics or rename. Send full-content
changes with increasing versions, save notifications, and watched-file change
notifications when contents change. Refresh retained sessions after every
registered mutation, including failed operations that may have published edits
and intermediate batch operations whose diagnostics are deferred.
Request full diagnostic reports only for the selected scan scope.

Within a buffered Go batch, package loading uses the source overlay and session
snapshots read the same working bytes as AST edits and symbol lookup. Send only
`didChange` for unpublished edits. Retain each document's saved state so an
unchanged final digest still triggers `didSave` and watcher notifications after
publication. The source buffer validates the complete disk read set and source
membership before publication. A publication failure discards unpublished bytes
and retained sessions; later calls reload the actual disk revision. Direct uses
of the batch API without MCP ownership share a temporary manager for the batch
and close it when the batch returns.

Preserve deterministic AST implementations for extraction, inlining, movement,
removal, insertion, and replacement, along with pipeline formatting/import policy.
They share the retained session for synchronization and diagnostic postprocessing.
The gopls actions do not satisfy the accepted selector, caller-name, comment,
type/evaluation, and refusal contracts in ADR-0056, ADR-0058, and ADR-0059.
Native gopls refactoring expansion requires its own validated operation contract.
The ingress installs a language-neutral mutation observer on the operation
context; the registry invokes it on success and failure without importing a Go
backend. Batch completion also refreshes sessions after partial postprocessing.
Rename applies text edits verbatim; standalone and final batch pipelines own
formatting and import normalization. Standalone rename normalizes only files
whose returned edits change their bytes; an empty edit plan skips normalization.
Buffered package discovery binds main-module manifests before loading, then
binds newly discovered local replacement sources and auxiliary files and reloads
until the captured input scope is stable.

Restart a session when the executable path, file identity, size, modification time,
environment, source file set, module
manifests, or auxiliary package inputs change. Fingerprint local module `go.mod`
and `go.sum` files, preserving the distinction between missing and empty files.
Versioned dependency caches and the installed Go toolchain remain installation
state rather than editable workspace buffers.

Close sessions at MCP end or reload. Canceling stdio or pipe input closes the input
to unblock an idle read before cleanup. Blocking inputs passed to `Serve` must
implement `io.Closer` to support this cancellation. Failed protocol requests or
buffer synchronization discard the affected session, so later calls start fresh.
Graceful shutdown is bounded; transport close terminates and reaps the process.

## Invariants

- Sessions belong to one MCP instance and are never shared between agent threads.
- Selected diagnostic coverage stays independent of session/module coverage.
- Every retained editable buffer is synchronized before a dependent diagnostic
  or rename request; watcher debounce alone is not a completion receipt.
- Rename requires existing regular canonical Go files within the enclosing module,
  exact open-document versions, valid UTF-16 ranges, non-overlapping edits, and
  unchanged analysis preimages. Bind the full resolver input snapshot and selected
  declaration identity before requesting rename. Recheck analysis inputs and target
  file identity after staging and immediately before each atomic publication,
  accounting for outputs already published by this operation. All outputs are
  validated before any publication;
  partial I/O publication remains an explicit error without rollback.
- Rename advertises versioned document edits and UTF-16 positions, without resource
  operations, edit annotations, or server-initiated mutation support.
- Ordinary source content changes reuse the process. Configuration and file-set
  changes restart it before returning diagnostics for the new state.
- Missing, malformed, unsupported, canceled, and incomplete diagnostic reports
  remain explicit failures, distinct from clean reports.
- Required subprocesses use the established project-local Go state boundary;
  session reuse does not create or modify workspace manifests.
- MCP tool timings remain recoverable through the persisted metrics file defined
  in ADR-0052. Process shutdown at MCP exit is outside an individual tool response.

## Consequences

Repeated MCP diagnostics retain gopls analysis state and avoid per-call shutdown.
The first scan still pays startup cost. Source discovery, input hashing, and disk
reads remain per-call work; retaining source buffers consumes memory proportional
to the active module and its local dependencies. File additions/removals and
configuration changes pay one replacement shutdown/startup cost.

The real-gopls tests in `internal/godiagnostics/diagnostics_test.go` prove process
reuse, source freshness, subtree scope, local dependency freshness, and parent
manifest invalidation. MCP tests in `internal/mcp/server_test.go` prove instance
ownership, reuse through rename, extraction, inlining, movement, deferred batch edits, and
idle cancellation/process cleanup. Go adapter protocol tests prove unsafe rename
edits are refused before writing, and CLI txtar cases cover Unicode targets and
existing multi-file rename contracts. Broader
CLI daemon sharing and other language lifecycle policy remain RQ-0004 work.
