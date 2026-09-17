# ADR-0022: Bounded Protocol-Neutral LSP Transport

* **Status**: Accepted
* **Date**: 2026-09-17

## Context

Future read-only language backends need a reusable bidirectional JSON-RPC transport. The transport must keep protocol framing and request lifecycle concerns out of language adapters while avoiding an implicit commitment to server discovery, language-specific coordinates, or source mutation.

## Decision

Add `internal/lsp` as a small transport package. It owns strict `Content-Length` framing, bounded payloads, JSON-RPC request correlation, cancellation notifications, asynchronous server-request and notification callbacks, supplied-process shutdown, and bounded stderr capture. Callers supply streams and payloads; the package does not launch a configured language server or interpret LSP methods.

## Invariants

* Framing errors are typed and distinguish malformed headers, malformed payloads, and payload limits.
* Every outbound request has a preserved JSON-RPC ID and concurrent requests are independently correlated.
* Reader dispatch never executes a caller handler synchronously; server requests receive a deliberate result or error response.
* Protocol output is written only to the supplied protocol stream. Stderr capture is bounded and separate.
* The package defines no Go token coordinates, WorkspaceEdit, changeset, source mutation, or broken-source fallback behavior.

## Consequences

Read-only Java, Rust, Scala, and Haskell lookup adapters can reuse one tested transport without sharing process or language assumptions. LSP initialize and lookup payloads remain caller-owned, keeping this slice small and allowing later adapters to choose their supported protocol version and capabilities.
