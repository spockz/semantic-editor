<!-- This decision records why the public Go module and executable use their canonical package paths. -->
# ADR-0069: Canonical Go Module and CLI Package

* **Status**: Accepted
* **Date**: 2026-10-14

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Use `github.com/spockz/semantic-editor` as the root module and `cmd/semedit` as the sole CLI executable package; exclude retained benchmark results from install archives using a nested module. |
| description | The repository's Go module used the local-only path `semedit`, and its root `main` package prevented installing the public executable through the repository's canonical GitHub path. Internal imports, development builds, and CLI tests were coupled to that layout. |
| role | architecture |
| scope | tooling |
| area | build |
| languages | go |

## Context

The repository's Go module used the local-only path `semedit`, and its root `main` package prevented installing the public executable through the repository's canonical GitHub path. Internal imports, development builds, and CLI tests were coupled to that layout.

## Decision

Use `github.com/spockz/semantic-editor` as the root module path and place the CLI and MCP executable in `cmd/semedit`. The supported remote installation target is `github.com/spockz/semantic-editor/cmd/semedit@latest`; local checkout builds use `./cmd/semedit`.

## Invariants

* Internal Go imports use the canonical module path.
* The executable remains named `semedit` and is built from `cmd/semedit`.
* Root CLI tests resolve repository fixtures and support binaries from the repository root without changing the package-wide working directory.
* No compatibility executable remains at the module root.
* Benchmark result records remain in the checkout but are excluded from the installable root module archive by a nested module boundary.

## Consequences

The module can be installed by its public repository path, while the conventional `cmd/` layout separates the executable from library packages. A nested module excludes benchmark result records, including colon-containing paths rejected by Go module archives, while preserving them in the repository. Development build commands and fixture tests must account for the executable's new location.
