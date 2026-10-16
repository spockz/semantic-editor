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

Independent review on 2026-10-15 reproduced two cache invalidation gaps. Adding
`bad.c` after a clean incremental capture returned no findings, while explicit
verification reported that C sources were not allowed in that package. Changing
build tags also left a retained dependency graph without a newly enabled reverse
importer. Filesystem freshness alone cannot establish that analysis coverage is current.

## Decision

Let each Model Context Protocol (MCP) server instance own a lazy Go LSP
session manager. Reuse one gopls process per enclosing Go module, including when
requests select different subdirectories of that module. Separate modules in one
workspace have separate processes. Command-line interface (CLI) rename invocations use the same protocol and edit
policy with a temporary session closed when the invocation ends. Go rename uses
`textDocument/prepareRename` and `textDocument/rename` on the retained session.
The Go adapter owns versioned WorkspaceEdit validation and atomic publication,
following ADR-0029; transport and lifecycle code do not apply edits.

Resolve the enclosing module with the same effective Go environment used for
diagnostics. ADR-0070 preserves inherited workspace and cache settings by default;
an explicitly managed base selects isolated state and `GOWORK=off`. Canonicalize scan paths so symlink aliases
do not create distinct document URIs or sessions. Retain the package graph while
its structural inputs remain unchanged. Validate source membership, package and
import headers, build constraints, manifests, local replacements, and auxiliary
inputs; content digests detect changes even when modification times are preserved.
Share effective environment discovery across a request tree. Keep the module's Go sources and imported local replacement sources open;
read their current disk contents before requesting diagnostics or rename. Send full-content
changes with increasing versions. Configure `diagnosticsTrigger: "Save"` so
ordinary content synchronization does not schedule background diagnosis alongside
explicit pulls. Omit duplicate save and type-2 watcher notifications for ordinary
open Go document changes; retain membership and non-Go input notifications.
These scheduling and save-elision rules apply only to gopls: this client uses
`textDocument/diagnostic` pulls and relies on the verified disk-hash comparison
below. They are not cross-backend LSP defaults. JDT LS, Kotlin, and Bash adapters
consume version-matched `publishDiagnostics` via `WaitDiagnostics`; suppressing
their change-triggered diagnostics would break verification. Java also requires
pre-publication changes for LSP normalization and post-publication `didSave`
for its working-copy lifecycle (ADR-0068).
Retain save-notification capability and send a targeted `didSave` when changed or
published bytes have the same disk file identity and modification time as the
previous synchronized observation. gopls may otherwise reuse its old disk hash
and consider the correct overlay unsaved. These metadata are notification-policy
evidence only; source digests still establish freshness. The fallback preserves
saved-overlay semantics but does not evict gopls's underlying disk cache.

The verified mechanics in gopls v0.23.0 explain this policy. In
`internal/cache/session.go:1049–1075`, `overlayFS.updateOverlays` handles
`file.Open` and `file.Change` by reading its delegate cache and setting
`overlay.saved = (readErr == nil && fh.Identity().Hash == hash)`. The delegate
uses `memoizedFS.ReadFile` (`internal/cache/fs_memoized.go:61–120`), which stats
file identity (device/inode) and mtime through `robustio.GetFileID`. Changed
identity or mtime causes a fresh read. Freshly read entries with an mtime less
than two seconds old are evicted rather than retained. Atomic publication before
`didChange` replaces the inode and advances mtime, so gopls reads matching disk
bytes and marks the overlay saved without `didSave`.

`fileWasSaved` (`internal/cache/snapshot.go:1992–2012`) first returns false
for equal file identities `{URI, Hash}`; otherwise two overlays qualify only
for an unsaved-to-saved transition. A same-content `didSave` after an ordinary
atomic edit therefore adds no such transition. The stale-disk-hash hazard is
an external in-place write preserving both device/inode and an old mtime
(at least two seconds), for which `memoizedFS` may retain old bytes. The targeted
save fallback conservatively covers unchanged identity/mtime without relying on
the cache age. This is version-specific implementation evidence, not a general
LSP guarantee; notification-policy changes require real-gopls regression checks.

Refresh retained sessions after registered mutations on published disk state,
including failed operations that may have published edits. Defer the observer
while a source buffer owns unpublished changes. The retained-session query
boundary synchronizes current working bytes before diagnostics or rename, so
AST-only children do not create unnecessary unsaved gopls overlays. AST-only
batches send changed content after publication. Private stage start, drop/replay,
and abort controls own their replay or reconciliation and skip the outer disk
observer; stage commit retains publication synchronization. Ordinary abort and
expiry reconcile only overlapping retained Go sessions against the current disk
view, closing staged-only documents and preserving the daemon. They never restore
old disk preimages over external edits. Cleanup reserves the staging scope until
reconciliation completes. Failure discards affected sessions and returns the
error; uncertain publication still uses conservative invalidation. This recovery
does not change another language server's working-copy lifecycle.
MCP publication records changed paths before flushing or committing, and refreshes
Go sessions for Go source, compiler auxiliary, or module/workspace inputs rather
than every non-Go edit under the full profile. Known non-Go-only request targets
skip the outer Go mutation observer. Unknown or directory-scoped targets retain
conservative synchronization. Every later actual Go query still validates its
complete scope, including embedded inputs; this routing does not authorize stale
analysis or change another backend's lifecycle.
Request full diagnostic reports only for the selected scan scope.

Resolve the current effective Go environment before selecting any cached package
graph, including graphs used to select batch verification coverage. Match the
complete environment-sensitive cache key before taking a filesystem fingerprint
fast path. Build tags and other build settings supplied through process variables
or the Go configuration file participate in that identity. An environment change
must refresh dependency selection before freezing a verification closure.

Fingerprint membership as well as contents for all compiler-recognized inputs in
the selected modules and editable local replacements. This includes newly added
non-Go inputs such as C, assembly, and header files, even when no previous package
load recorded them. Creation, deletion, or retargeting invalidates discovery and
diagnostic findings. Uncertain input classification or environment identity
requires fresh discovery or explicit full coverage rather than cached clean results.
Apply pathname-based input selection before requesting
leaf metadata: unrelated archived results and documentation do not require stat
calls. Selected compiler, manifest, embedded, and known auxiliary inputs retain
metadata, symlink-target, and content validation, including explicit failures.

Across fingerprint observations, retain a compact structural encoding and
parse-validity flag indexed by source path and exact content digest. The embed
scan prepares this representation; the hashing walk still reads current bytes
and reuses it only when the digest matches. A mismatch reparses the current
source. Parse errors use the complete current bytes rather than a partial AST.
Preserve the existing structural byte order, directive comments, cgo preambles,
membership, overlays, symlink checks, and read failures. Keep entries within the
same canonical scope/environment identity and prune absent paths after successful
fingerprinting. Retain neither whole syntax trees nor source buffers.
This removes repeated parsing without claiming a coherent filesystem snapshot
or replacing content validation with modification-time checks.

Fresh package discovery supplies the current package findings directly rather
than immediately loading the same metadata again. A successful default full
verification may seed the next incremental capture only when its files and
package findings exactly cover the managed scope. Explicit-pattern and buffered
captures do not seed the disk baseline. Replace scope-specific package findings
on such a seed rather than preserving stale sibling entries. Managed package
loads use the active source overlay; standalone disk-only collection remains
disk-only. Diagnostic results enter the cache only after all requested pulls
succeed; cancellation or a failed pull cannot publish partial results.

When dependency discovery changes and either the old or new package graph has
metadata errors, invalidate diagnostic coverage for the complete scope. Local
cache invalidation alone is insufficient: gopls v0.23.0 can retain a missing-
metadata diagnostic after an import cycle is repaired. Recreate that client's
workspace view by removing and re-adding its workspace folder after current
documents have synchronized. Preserve the process and open document versions.
Advertise client workspace-folder support and require the server's advertised
support for folder-change notifications before using this recovery. Run the
remove/add pair with a bounded context independent of request cancellation;
transport failure discards the affected session and returns an error. A server
without folder-change support restarts only on this exceptional transition.
Ordinary import edits, unchanged broken revisions, and body edits do not reload
the view. This recovery addresses gopls metadata behavior; other LSP backends
retain their own invalidation and diagnostic completion contracts.

Within a buffered Go batch, package loading uses the source overlay and session
snapshots read the same working bytes as AST edits and symbol lookup. Send only
`didChange` when a query needs unpublished edits. Retain each document's saved state so an
unchanged final digest still triggers a full-content `didChange` with an advancing
version after publication. This lets gopls compare the overlay with the newly
published disk file without scheduling ordinary save-triggered diagnosis.
The transition is required when `!previous.saved && !sourcebuf.Dirty(ctx, path)`,
even if source bytes did not change during flush. Avoiding eager pre-flush changes
keeps AST-only edits out of the unsaved-overlay state: `Snapshot.buildOverlays`
(`internal/cache/snapshot.go:492–504`, used by Go command invocation near line 481)
materializes unsaved overlays as temporary `-overlay` inputs during `go list`.
Lazy synchronization avoids that work and the second post-flush content update
unless an actual query needed unpublished bytes.
Unpublished creations may have no disk identity; unexpected stat errors fail
synchronization. The source buffer validates the complete disk read set and source
membership before publication. A publication failure discards unpublished bytes
and retained sessions; later calls reload the actual disk revision. Direct uses
of the batch API without MCP ownership share a temporary manager for the batch
and close it when the batch returns.

Preserve deterministic AST implementations for extraction, inlining, movement,
removal, insertion, and replacement, along with pipeline formatting/import policy.
They share the retained session for synchronization and diagnostic postprocessing.
An upstream gopls action is not a validated replacement merely because its name
resembles an accepted operation. Delegation must satisfy the selector,
caller-name, comment, type/evaluation, and refusal contracts in ADR-0056,
ADR-0058, ADR-0059, and ADR-0049. Native gopls refactoring expansion requires
its own validated operation contract; newer experimental move support remains
separate from the currently integrated implementation.
The ingress installs a language-neutral mutation observer on the operation
context; the registry invokes it on success and failure without importing a Go
backend. Batch completion also refreshes sessions after partial postprocessing.
Rename applies text edits verbatim; standalone and final batch pipelines own
formatting and import normalization. Standalone rename normalizes only files
whose returned edits change their bytes; an empty edit plan skips normalization.
Buffered package discovery binds main-module manifests before loading, then
binds newly discovered local replacement sources and auxiliary files and reloads
until the captured input scope is stable.

Restart a session when the gopls executable's path, file identity, size, modification time,
environment, or module configuration changes. Ordinary Go file creation and
deletion reconcile open documents and watched-file notifications on the retained
process. Structural changes invalidate discovery even when the process remains
usable. Keep a separate diagnostic boundary fingerprint from the same observation:
it retains every path, symlink target, manifest, auxiliary input, package name,
compiler directive, cgo import/preamble, and parse-validity distinction, while
excluding ordinary Go import lists. Malformed source contributes its complete
bytes. Import changes still rebuild the current package graph. Retain diagnostic
findings and their input digests only when this boundary and the exact source
and module lists match. Synchronize current documents, refresh changed packages
and their transitive reverse importers using that graph, replace affected package
findings, and merge unaffected cached diagnostics. Changes to build constraints,
package membership, configuration, embedded/compiler inputs, cgo boundaries, or
malformed source retain the full-refresh fallback, as does uncertain graph
coverage. This Go diagnostic-cache rule also
applies when an external editor changes imports before a tool call; it does not
trust a tool's declared target as the only changed input.

Compute discovery and diagnostic boundary hashes in one existing walk, sharing
the compact per-file encodings and parse results. Fresh buffered scope loading
does not compute an additional fingerprint before returning to its owning cache.
These observations do not replace fresh-byte validation or create a coherent
filesystem snapshot. Fingerprint local module `go.mod`
and `go.sum` files, preserving the distinction between missing and empty files.
Versioned dependency caches and the installed Go toolchain remain installation
state rather than editable workspace buffers.

Close sessions at MCP end or reload. Canceling stdio or pipe input closes the input
to unblock an idle read before cleanup. Blocking inputs passed to `Serve` must
implement `io.Closer` to support this cancellation. Failed protocol requests or
buffer synchronization discard the affected session, so later calls start fresh.
Graceful shutdown is bounded; transport close terminates and reaps the process.

## Invariants

### Shared contracts and backend-owned mechanisms

The import-refresh optimization lives in `internal/godiagnostics`; it is not
implemented in the generic LSP transport. Separate discovery and diagnostic
boundary hashes, Go compiler input classification, `go/packages` dependency
selection, and retained per-file pull results are Go-owned mechanisms. Import
organization still edits only its selected files; diagnostic refresh can cover
additional dependent packages. Changing imports does not authorize dropping
unrelated findings or limiting validation to the requested edit target.

| Boundary | Shared behavior | Current backend implementation |
| --- | --- | --- |
| Diagnostic feedback | Accurate pre/post findings, original severities, explicit incomplete coverage, no forced rollback | Go retains validated findings and refreshes affected package closures; Java computes its backend-owned diagnostic deltas. There is no universal cross-language incremental cache. |
| Source buffering | Lazy preimages for actual reads/writes and explicitly observed trees; conflict-checked atomic publication (ADR-0032) | Language-specific normalization and dependency discovery decide which inputs must be observed. |
| Synchronization | A dependent server query must see the current working bytes | Go AST-only edits can wait until publication; Java must synchronize before LSP formatting, imports, and chained queries (ADR-0068). |
| Diagnostic completion | A result must have evidence for the selected revision and scope | Go requests `textDocument/diagnostic`; Java, Kotlin, and Bash wait for matching push-diagnostic receipts. |
| Background scheduling and saves | Follow the server's lifecycle; transport delivery is not analysis completion | Only gopls uses this `diagnosticsTrigger: "Save"` and ordinary save/watcher elision policy, including its targeted save fallback. Java retains explicit saves. |
| Rust | Shared transport and edit/publication contracts where supported | Rust verification currently returns unsupported. Its `checkOnSave` setting does not establish a diagnostic cache or coverage equivalent to Go. |

Before extending scoped diagnostic reuse to another backend, define its dependency
coverage, invalidation inputs, and revision-matched completion evidence, then
compare retained results against fresh captures with unrelated errors and
external edits present. Standard LSP message names alone do not establish those
properties. Go's measured reduction in pull count does not establish a latency
improvement for any other backend.

- Sessions belong to one MCP instance and are never shared between agent threads.
- Selected diagnostic coverage stays independent of session/module coverage.
- Every package graph consumer uses the current canonical scope and complete
  effective environment key; a matching filesystem fingerprint cannot bypass
  environment validation.
- Input membership validation includes newly added compiler-recognized non-Go
  files. Tracking only previously discovered auxiliary files is insufficient.
- Every retained editable buffer is synchronized before a dependent diagnostic
  or rename request; watcher debounce alone is not a completion receipt.
- AST-only buffered children defer synchronization until a dependent query or
  publication. Replay and query boundaries must synchronize the current buffer,
  including after dropping an earlier staged step.
- Rename requires existing regular canonical Go files within the enclosing module,
  exact open-document versions, valid UTF-16 ranges, non-overlapping edits, and
  unchanged analysis preimages. Bind the full resolver input snapshot and selected
  declaration identity before requesting rename. Recheck each output's canonical
  path, regular-file identity, and exact preimage after staging and immediately
  before its atomic publication. All outputs are
  validated before any publication;
  partial I/O publication remains an explicit error without rollback.
- Rename advertises versioned document edits and UTF-16 positions, without resource
  operations, edit annotations, or server-initiated mutation support.
- Ordinary Go source content and membership changes reuse the process. Incompatible
  configuration changes restart it before returning diagnostics for the new state.
- Automatic diagnostic captures may reuse revision-validated findings for unaffected
  packages. Changed packages and their complete reverse dependency closure are
  refreshed; uncertain coverage falls back to a full capture. Explicit verification
  remains full. Batch scope selection uses the same retained dependency graph.
- Import-only discovery refresh preserves diagnostics only under an equal
  diagnostic boundary and identical source/module membership. All changed source
  digests participate in refresh, including concurrent edits outside the tool target.
- A zero-write optimistic source conflict preserves the session for fresh
  synchronization and one bounded rename replan. Partial publication is explicit.
- Missing, malformed, unsupported, canceled, and incomplete diagnostic reports
  remain explicit failures, distinct from clean reports.
- Required subprocesses use the selected inherited or explicitly isolated Go
  state boundary under ADR-0070;
  session reuse does not create or modify workspace manifests.
- MCP tool timings remain recoverable through the persisted metrics file defined
  in ADR-0052. Process shutdown at MCP exit is outside an individual tool response.

## Consequences

Both review findings are fixed. Compiler input classification uses `go/build`
independently of the host platform and current build constraints, and graph
selection resolves the effective environment before using an environment-keyed
scope. Real-gopls regressions compare retained captures with fresh full captures
after C-file creation/removal and symlink retargeting, and compare retained and
fresh graph closures after GOFLAGS or GOENV build-tag changes. These checks also
pass under the race detector; unchanged captures still reuse cached diagnostics.

Repeated MCP diagnostics retain gopls analysis state and avoid per-call shutdown.
The first scan still pays startup cost. Source discovery, input hashing, and disk
reads remain per-call work; retained LSP documents and diagnostic caches consume
memory proportional to the active module and its local dependencies. Operation
buffers still capture preimages lazily under ADR-0032. Configuration changes may pay a
replacement shutdown/startup cost; ordinary file additions/removals do not.

The real-gopls tests in `internal/godiagnostics/diagnostics_test.go` prove process
reuse, source freshness, subtree scope, local dependency freshness, and parent
manifest invalidation. MCP tests in `internal/mcp/server_test.go` prove instance
ownership, reuse through rename, extraction, inlining, movement, deferred batch edits, and
idle cancellation/process cleanup. Go adapter protocol tests prove unsafe rename
edits are refused before writing, and CLI txtar cases cover Unicode targets and
existing multi-file rename contracts. Broader
CLI daemon sharing and other language lifecycle policy remain RQ-0004 work.
