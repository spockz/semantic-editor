# ADR-0036: Java Bounded Verification Actions

* **Status**: Accepted
* **Date**: 2026-09-21

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Trusted selected-file Java formatting, organize-imports, and bounded diagnostics through the central operation registry; no Maven/Gradle/build execution. |
| description | Java verification needs a shared operation boundary while preserving explicit trust and no-build constraints. |
| role | contract |
| scope | tooling |
| area | language-server |
| languages | java |

## Context

Java verification needs a shared operation boundary while preserving explicit trust and no-build constraints.

## Decision

The registry exposes trusted Java selected-file formatting, source.organizeImports, and bounded diagnostics through CLI `verify` and MCP `semantic_verify`. JDT LS edits are validated and written atomically; Maven, Maven Wrapper, Gradle, and build/test commands are never executed.

## Invariants

* Trust is explicit and request-scoped.
* Edits cannot target another URI, command, resource operation, annotation, malformed range, or overlap.
* Empty organize-import actions are no-ops; non-empty actions must be `source.organizeImports`.
* Diagnostics readiness is bounded and timeout failures are explicit.

## Consequences

Java verification remains narrower than a general build and is covered by hermetic backend and CLI txtar tests.
