# ADR-0015: Hugo + Lotus Docs Documentation Presentation Layer

* **Status**: Accepted
* **Date**: 2026-09-16

## Context

The documentation site was emitted as standalone HTML by `cmd/docgen` and uploaded directly from `dist/docs` by the GitHub Pages workflow. The generator correctly derives executable examples from `testdata/scripts/*.txtar`, but the custom HTML presentation did not provide the standard copy controls users expect for code blocks.

Hugo with the Lotus Docs theme provides the required copy control for fenced code blocks, along with repository links, navigation, search, and responsive code-block presentation. The theme is a native Hugo/Go build dependency and does not require a Python or Node runtime.

## Decision

Adopt Hugo with the Lotus Docs theme as the documentation presentation layer while retaining `cmd/docgen` as the deterministic synthesis engine and source-of-truth boundary.

The migration will:

1. Make `cmd/docgen` emit a temporary Hugo source tree with Markdown pages and fenced CLI, MCP, and source-code examples.
2. Configure Hugo Modules to import Lotus Docs and its Bootstrap dependency.
3. Make `verify-docs` the shared build-and-assert target: `make check` invokes it locally and the Pages workflow invokes it before publishing `dist/docs`.
4. Preserve direct GitHub links for every generated test archive and keep executable `txtar` files as the example source of truth.

The migration details remain subject to [RQ-0019](../research/RQ-0019-mkdocs-material-migration.md).

## Invariants

* `testdata/scripts/*.txtar` remains the authoritative source for executable examples.
* Documentation generation must not require manually duplicated example content.
* Every copyable example must render as a fenced code block so the theme can attach its standard control.
* Generated documentation remains deterministic and is regenerated and checked in CI before publication.
* Repository and source links must resolve to the canonical GitHub repository rather than duplicated files on the docs host.

## Consequences

* The project gains standard code-copy controls, responsive code presentation, navigation, search, and repository integration.
* The project takes on a Hugo Extended build dependency and an additional theme-module fetch/build step in Pages CI.
* The custom standalone HTML template and its bespoke styling become transitional code and can be removed after Markdown output reaches parity.
* The generated source tree remains temporary under `.scratch/docgen`; only the rendered `dist/docs` site is published.
