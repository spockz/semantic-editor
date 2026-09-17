# ADR-0025: Java File-Scoped Lookup Backend

* **Status**: Accepted
* **Date**: 2026-09-17

## Context

Java symbol lookup needs JDT LS project indexing, but the lookup slice must remain read-only and must not turn Maven or Gradle configuration into an execution surface. JDT LS is a user-installed external tool and requires an explicit trust boundary and a supported Java runtime.

## Decision

1. Register Java only for `resolve_symbol_location` / CLI `lookup`. Rename, verification, formatting, imports, dependency changes, structural edits, build import, and build execution remain unavailable and unadvertised.
2. Require a selected `.java` file and either an explicit root or the nearest unambiguous ancestor containing `pom.xml`, `build.gradle`, or `build.gradle.kts`. Same-level Maven and Gradle markers are rejected; build tools are never invoked.
3. Require request-scoped `WorkspaceTrust` before JDT LS or Java binary discovery. JDT LS home is explicit and preinstalled; Java 21+ may be an explicit binary or the user's `PATH`. No download, embedded Java library, or `jdtls.py` launcher is permitted.
4. Use one managed JDT LS session per trusted canonical workspace. Its data directory is a stable hash below `<root>/.scratch/jdtls/`; source and project manifests are never written.
5. Initialize with UTF-16 and hierarchical document-symbol capabilities, disable build import/autobuild settings, and resolve only hierarchical package, class/interface/enum/record, field, method, constructor, and nested-type symbols using `selectionRange`.

## Invariants

* Untrusted Java requests cannot reach `LookPath`, Java version validation, JDT LS path validation, or process launch.
* Java lookup never invokes Maven or Gradle and never mutates source or project metadata.
* Out-of-root symbol URIs, malformed responses, unsupported flat symbols, and invalid positions are rejected.
* Overloads without signatures, locals, generated or annotation-derived symbols, and malformed-source fallback are outside the advertised subset.
* A Java session cannot silently coexist with another Java workspace session in one backend instance.

## Consequences

Trusted Java projects gain deterministic file-scoped lookup while preserving explicit external-tool consent and the Go-only implementation policy. Real JDT LS integration is opt-in at runtime; tests inject a fake session and do not require Java or a server installation.
