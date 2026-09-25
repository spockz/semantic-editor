# ADR-0047: Trusted Read-Only Kotlin Backend

* **Status**: Accepted
* **Date**: 2026-09-25

---

## Context

The language-neutral operation registry needs a Kotlin path for selected-file symbol lookup and bounded diagnostics. Kotlin tooling can resolve project classpaths by running classpath scripts or Maven/Gradle, so starting a language server against the caller's project before explicit trust would cross the external-tool boundary. The selected preinstalled `fwcd/kotlin-language-server` also publishes diagnostics without a document version, making a quiet server indistinguishable from a clean file unless the client requires an explicit report.

## Decision

Register Kotlin as a trusted read-only backend for hierarchical selected-file lookup and selected-file verification. Bound each complete server operation, including launch, initialization, document open, symbol lookup, and diagnostics wait, by a 30-second deadline; verification retains a shorter diagnostics wait. Accept `.kt` and `.kts` inputs and run the preinstalled `kotlin-language-server` executable through the bounded stdio LSP transport. No download or installation is performed. Lookup resolves exact hierarchical symbols from the selected document only. Verification returns diagnostics only after receiving `textDocument/publishDiagnostics` for the selected scratch URI; formatting and organize-imports requests are rejected.

Before resolving or launching the executable, require request-scoped trust for the canonical original workspace root. Then create a temporary, source-only workspace beneath the original root's `.scratch`, copy the selected source atomically, and run the server there with isolated HOME and XDG configuration/cache directories. The server is closed and the scratch workspace removed after each operation. A unique scratch URI for each request binds versionless reports to that request; when a report includes a version, it must be at least the opened document version. An explicit empty diagnostics array is clean. Missing, malformed, or wrong-URI reports never imply cleanliness and instead time out as an error.

## Invariants

* Trust is checked against the original canonical project root before executable lookup, process launch, or project access by the server.
* Only the selected source file is copied into the isolated workspace; output locations map back to the original selected file.
* Each operation owns a fresh server session, unique source URI, and bounded scratch lifecycle.
* Each operation has a 30-second deadline; verification diagnostics have a shorter bounded wait. Operation timeout errors are explicit, and timeout cleanup removes the process session and scratch workspace.
* A selected-file diagnostics receipt must match the active URI. A supplied version must match or exceed the opened document version; an omitted version is accepted only for the fresh request URI.

* Silence, wrong-URI notifications, and unsupported verification submodes are errors; only an explicit matching empty diagnostics list reports clean verification.
* Kotlin mutation, build import, formatting, and import organization are not advertised or accepted.

## Consequences

Kotlin supports trusted CLI and MCP lookup and selected-file verification without adding Kotlin runtime code or installing a server. The selected Kotlin server may still execute project classpath scripts or build tools after trust; the source-only workspace and isolated user configuration reduce that exposure, while explicit trust remains the authorization gate. Diagnostics depend on the selected server publishing a matching report within the bounded timeout. Versionless reports are usable because each server session and URI are request-scoped. Broader project analysis, cross-file lookup, overload disambiguation, and mutation remain unavailable.
