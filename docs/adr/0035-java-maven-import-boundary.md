# ADR-0034: Explicit Maven Import Boundary for Java

Status: Accepted
Date: 2026-09-20

## Context

Java support needs Maven-reactor-aware workspace scope and a path to JDT LS project import. Build-tool execution remains outside the semantic editor boundary, and Gradle import must not be enabled as a side effect.

## Decision

Java workspace discovery may ascend from a module only when an ancestor POM explicitly lists that module under `<modules>`. A trusted request may explicitly opt in to JDT LS Maven import. The adapter passes Maven import settings to JDT LS while keeping Gradle import, autobuild, and project-root metadata generation disabled. Maven and Maven Wrapper commands are never invoked.

## Invariants

- Workspace trust is request-scoped and required before JDT LS starts.
- Reactor scope is proven by explicit `<modules>` membership; parent inheritance alone is insufficient.
- Maven import is opt-in; Gradle import and autobuild remain disabled.
- JDT/M2E metadata is not generated at the project root.
- No Maven, Maven Wrapper, or Gradle process is executed.

## Consequences

Java lookup and selected-file rename can use a reactor root when the POM declares it, while unrelated inherited Maven projects remain module-scoped. Import remains deterministic and safety-bounded, but callers must explicitly request Maven import when project metadata is needed.
