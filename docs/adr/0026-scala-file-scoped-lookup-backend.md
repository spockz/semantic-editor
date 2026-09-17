# ADR-0026: Scala File-Scoped Lookup Backend

* **Status**: Accepted
* **Date**: 2026-09-17

## Context

Scala symbol lookup needs Metals' hierarchical document symbols, but a lookup request must not become an implicit sbt, Maven, Gradle, Mill, or Scala CLI execution surface. This slice is deliberately limited to a selected `.scala` file and one managed Metals LSP session.

## Decision

1. Register Scala only for trusted read-only lookup. Rename, verification, formatting, imports, dependency changes, build import, compilation, and structural edits remain unavailable and unadvertised.
2. Require a selected `.scala` file and an explicit workspace root when project markers are present. A file with no markers may use its containing directory as a deliberate standalone read-only root; no build import is attempted.
3. Require request-scoped workspace trust before Metals or Java discovery and process launch. The caller must provide a direct, pinned preinstalled Metals executable or distribution and a preinstalled Java 21+ executable plus recorded major version. No Coursier bootstrap or external installation is allowed.
4. Store Metals data and logs only below `<workspace>/.scratch/`; `.metals` state is not created or removed, and the invocation does not rely on or guarantee its absence outside the configured data directory.
5. Start Metals with build import/autobuild disabled, no HTTP UI, and no BSP generation or Bloop switch. Server requests that ask to import a build or create configuration are rejected. No BSP process is started.
6. Use one managed session per trusted canonical workspace. Send `didOpen` and `textDocument/documentSymbol`, negotiate UTF-16, and resolve only hierarchical classes, objects, traits, enums, methods, fields, and nested types by `selectionRange`.

## Invariants

* Untrusted Scala requests cannot reach Java or Metals validation, discovery, or process launch.
* Scala lookup never invokes sbt, Maven, Gradle, Mill, Scala CLI, BSP, Bloop, scalafmt, or scalafix and never mutates source, manifests, or project metadata.
* Flat responses, malformed positions, out-of-root URIs, overload signatures, givens, extensions, package objects, generated or macro-derived symbols, and cross-file SemanticDB search are outside the advertised subset.
* A Scala backend cannot silently hold sessions for multiple canonical roots.

## Consequences

Trusted Scala files gain deterministic local symbol lookup while preserving the explicit external-tool boundary and Go-only implementation policy. Real Metals integration remains opt-in; normal tests inject a fake session and do not require Java or Metals.
