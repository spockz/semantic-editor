# ADR-0037: Java Bounded Maven Actions

* **Status**: Accepted
* **Date**: 2026-09-21

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Fixed trusted in-workspace root-POM test-compile/test with validated launchers, bounded output metadata/log tails, Unix process-group cancellation, isolated scratch state, and network-disabled-by-default execution. |
| description | Java projects need explicit build and test verification, while JDT LS must remain a separate semantic service. Build tools are external processes and can execute project-controlled code or access the network. |
| role | contract |
| scope | tooling |
| area | build |
| languages | java |

## Context

Java projects need explicit build and test verification, while JDT LS must remain a separate semantic service. Build tools are external processes and can execute project-controlled code or access the network.

## Decision

The central operation registry exposes Java-only `maven-compile` and `maven-test` actions. They run only the fixed `test-compile` and `test` goals against a canonical root `pom.xml`, after request-scoped trust. A requested root must remain within the canonical trusted workspace. `maven_tool` selects `auto`, `wrapper`, or `system`; wrappers and system executables are validated before launch. Execution uses direct argv, a fixed two-minute bound, bounded output capture with truncation metadata and a log tail, and scratch-local Maven home, configuration, and repository. Network access is disabled by default and requires the positive `allow_network=true` request.

## Invariants

* JDT LS and Maven execution are separate operation paths.
* No shell, user-supplied goals, profiles, properties, or arbitrary arguments are accepted.
* Trust is never persisted and must contain the canonical selected root for each request.
* Failures retain typed process/exit information, bounded stdout/stderr, truncation metadata, and a log tail.
* Cancellation and timeouts terminate the Maven process group on Unix, including descendants holding output pipes open after the launcher exits. Windows terminates the direct process. Captured output and typed cancellation errors remain available.

## Consequences

Build verification is predictable and auditable, at the cost of requiring explicit configuration for non-default Maven installations, an in-workspace root, and explicit opt-in for network access.
