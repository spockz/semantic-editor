# ADR-0028: Cross-Language CLI Txtar Coverage

* **Status**: Accepted
* **Date**: 2026-09-17

## Context

The repository's txtar scripts execute the public `semedit` CLI against isolated workspaces and assert observable output and filesystem results. They provide a stronger cross-language contract than backend unit tests, which may inject fake language-server, compiler, or tool transports. As language backends gain equivalent operations, testing only their internal adapters permits ingress, flag, workspace-selection, and capability-advertisement drift.

## Decision

1. Treat `testdata/scripts/*.txtar` as the primary executable contract for public semantic operations.
2. Write txtar scenarios through the CLI first. They must assert command output or diagnostics and, for mutations, the resulting workspace state.
3. Keep comparable txtar coverage for every implemented language-operation pair when an operation is advertised by more than one language backend.
4. Retain fake-tool unit tests for protocol framing, tool discovery, malformed responses, and failure handling. They complement, but never replace, CLI txtar scenarios.
5. A missing language-operation scenario is permitted only when the capability registry marks the operation unsupported and the backend metadata and ADR document that asymmetry. Adding an operation to a second language includes the corresponding CLI txtar work in the same change.

## Invariants

* Public operation coverage is exercised through the CLI, not only through backend interfaces or MCP handlers.
* Comparable language capabilities have comparable txtar contract coverage.
* Tests that need unavailable external toolchains use deterministic fake tools behind the CLI; opt-in real-tool tests provide supplementary integration evidence.
* Unsupported operations are omitted deliberately and truthfully from capability discovery and txtar matrices.

## Consequences

New language backends must add CLI txtar fixtures as part of their delivery. Existing backend unit tests remain valuable for narrow transport behavior, but documentation and release confidence rely on the public CLI scenarios. The initial lookup-only language backends need follow-up txtar fixtures that drive fake language-server toolchains through their CLI configuration.
