# ADR-0027: Haskell Standalone File-Scoped Lookup Backend

* **Status**: Accepted
* **Date**: 2026-09-17

## Context

Haskell Language Server (HLS) can provide hierarchical document symbols, but its normal cradle discovery may read project configuration and invoke build tooling. This slice must keep Haskell lookup explicitly standalone and read-only.

## Decision

1. Register Haskell only for explicitly requested, trusted lookup of one `.hs` file. The CLI and MCP require an explicit standalone Haskell input; auto detection does not select Haskell. `.lhs` and boot files remain unsupported.
2. Reject the selected file when its directory or any ancestor within the selected root contains `hie.yaml`, `stack.yaml`, `cabal.project`, `*.cabal`, or `package.yaml`. No cradle discovery, project mode, Cabal, Stack, compilation, diagnostics, or malformed-source fallback occurs.
3. Require explicit workspace trust before GHC/HLS discovery, version probes, or process launch. Both `ghc` and `haskell-language-server-wrapper` must already exist. The wrapper's documented probe must identify HLS and GHC versions that exactly match the GHC probe; no installer or bootstrap command runs.
4. Launch `haskell-language-server-wrapper --lsp` only in the standalone file directory. Configure HLS to disable project checks and mutating or unrelated plugins wherever supported, then send `didOpen` and `textDocument/documentSymbol` over one UTF-16 hierarchical session.
5. Resolve only module, top-level values, types, classes, constructors, fields, and instances represented by HLS hierarchy. Ambiguity returns all candidates with selection ranges. Locals, pattern synonyms, duplicate record fields, reexports, generated or Template Haskell symbols remain outside the contract.

## Invariants

* Untrusted Haskell requests cannot reach GHC/HLS discovery, version probing, or process launch.
* Standalone Haskell lookup never invokes a project tool or mutates source, manifests, or project metadata.
* A Haskell backend cannot silently hold sessions for multiple canonical roots.
* Flat responses, malformed positions, out-of-root symbols, and malformed-source fallback are rejected.

## Consequences

Trusted standalone Haskell files gain deterministic local symbol lookup while preserving the external-tool trust boundary. Real HLS integration remains opt-in; normal tests inject a fake session and do not require GHC or HLS.
