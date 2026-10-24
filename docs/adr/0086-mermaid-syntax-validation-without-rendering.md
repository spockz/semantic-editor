<!-- WHY: Defines every required decision field and its stable metadata table keys for adr-tools-created records. -->
# ADR-0086: Mermaid Syntax Validation Without Rendering

* **Status**: Accepted
* **Date**: 2026-10-21

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Validate Mermaid diagrams in Markdown through Mermaid's parser without launching a browser. |
| description | Use a Markdown lexer to locate Mermaid fences and Mermaid's syntax-only parser to validate them in repository checks. This keeps malformed diagrams from passing while avoiding browser startup and generated render artifacts. |
| role | process |
| scope | tooling |
| area | build |
| languages | markdown, mermaid |

## Context

Mermaid diagrams are embedded in project Markdown, and malformed syntax can break published documentation. Mermaid CLI can parse embedded diagrams by rendering them through Puppeteer, but rendering adds browser installation and startup requirements to a syntax-only check. The Mermaid API documents a syntax parser that validates without rendering; its parser needs a DOM implementation for labels in Node.

## Decision

Install pinned Mermaid, Marked, and jsdom packages through the existing local `make tools` workflow. Use Marked to identify Mermaid fenced blocks, initialize jsdom before importing Mermaid, and call `mermaid.parse` on each diagram. Run the validator and its positive/negative tests as part of `make check` and `make check-ci`. Keep generated files under `.scratch` only; no rendered output is published.

## Invariants

* Validation uses Mermaid's own parser rather than a second Mermaid grammar.
* Invalid Mermaid syntax fails repository checks with the Markdown file and diagram number.
* Validation does not launch a browser or modify Markdown source files.
* Only fenced blocks whose Markdown language is `mermaid` are parsed.

## Consequences

Mermaid syntax regressions are caught during local and CI checks without requiring Chromium. Mermaid parser upgrades remain an explicit versioned tooling change; structural graph transformations remain outside this decision and require a separate syntax-preserving design.
