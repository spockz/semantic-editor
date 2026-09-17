# ADR-0021: Registered Language Backend and Ingress Service Boundary

* **Status**: Accepted
* **Date**: 2026-09-17

## Context

CLI and MCP previously called Go symbol resolution, gopls rename, and diagnostics directly. That made language selection and operation support ingress concerns, and left no stable place for a second backend. The first foundation slice needs a bounded extension point without migrating Go AST mutations or inventing a generic changeset engine.

## Decision

1. `internal/backend` owns the language-neutral contract: language IDs, project context, UTF-16 protocol locations, diagnostics, capabilities, typed boundary errors, backend registration, and the shared service.
2. `Backend` implementations are registered by `LanguageID` in a `Registry`. `auto` selects the registered backend from a selected source extension or an unambiguous project marker; no unavailable language is advertised or selected.
3. The ingress-facing `Service` is the only service-level capability gate. CLI and MCP common lookup, rename, and verify operations call this service.
4. The Go adapter delegates to the existing resolver, gopls adapter, and diagnostic pipeline. Go AST-specific commands remain on their established paths in this slice.
5. Native backend lookup is allowed under the qualification in ADR-0003. This decision does not add a broken-source fallback or a generic changeset engine.

## Invariants

* New backend and service contracts do not expose `go/token.Position`.
* Capabilities are the single source for service-level operation rejection.
* The registry contains only implemented backends; Java, Rust, and Scala are registered for their explicitly documented lookup-only capabilities, while Python, TypeScript, and Haskell remain unavailable.
* Existing Go CLI and MCP behavior remains unchanged for common operations unless an explicit language selection is supplied.

## Consequences

* CLI and MCP share selection, capability rejection, location conversion, and common operation orchestration.
* A second backend can be added without making ingress packages depend on language-specific coordinates.
* Existing Go AST edits are intentionally outside this boundary until a later, separately reviewed slice.
