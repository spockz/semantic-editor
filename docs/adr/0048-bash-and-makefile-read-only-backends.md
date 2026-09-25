<!-- This decision records the trust and capability boundaries established by Bash and Makefile LSP probes. -->
# ADR-0048: Trusted Read-Only Bash and Makefile Backends

* **Status**: Accepted
* **Date**: 2026-09-25

## Context

The source detector already recognizes `.sh`, `.bash`, Makefile variants, and `.mk`, but no registered backend served these files. Probes of bash-language-server 5.8.0 and make-ls v0.1.22 established bounded selected-document symbol responses. The Bash server also published explicit diagnostics for the opened URI; make-ls did not publish diagnostics during the bounded observation. Both servers are external executables and therefore cross the workspace trust boundary.

## Decision

Register trusted read-only Bash lookup and selected-file diagnostics for `.sh` and `.bash`. Register trusted read-only Makefile lookup for `Makefile`, `makefile`, `GNUmakefile`, and `.mk`. Resolve executable paths and start the server only after request-scoped trust for the canonical original root is established. Run each request in a temporary source-only workspace beneath the original root's `.scratch`, use UTF-16 source positions, and bound the complete request and process lifetime.

Diagnostics are clean only after an explicit `publishDiagnostics` message with the unique selected scratch URI and an array. An absent, malformed, stale-version, or wrong-URI report cannot imply success. Makefile verification, formatting, rename, imports, and all construct mutations remain unavailable. In a mixed workspace containing `go.mod`, Go remains the no-file auto lookup backend. make-ls resolves includes; absolute or traversing relative include paths outside the copied source are not confined by the scratch workspace and remain within the explicit-trust boundary.

## Invariants

* Untrusted requests do not resolve or launch external executables.
* Only one selected source file is copied into each request workspace; original-file paths and ranges are returned to callers.
* Timeouts, malformed responses, cleanup failures, and missing diagnostic receipts remain explicit errors.
* Makefile recipes and shell scripts are never executed by the backend.
* Capability matrices and operation schemas advertise only lookup for Make and lookup plus verification for Bash.
* No Bash or Make construct mutation is registered from document-symbol ranges or regular expressions.

## Consequences

The backends provide selected-file lookup without adding language runtimes or parser dependencies. Bash verification depends on a matching diagnostics notification; upstream may invoke an installed ShellCheck against the isolated copy, while the adapter does not execute scripts. Makefile include resolution may read an absolute or traversing relative path outside the copy after trust is granted, and broader parser-backed mutation requires separate evidence and decisions.
