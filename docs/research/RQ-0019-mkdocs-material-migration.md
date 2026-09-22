# RQ-0019: Hugo + Hextra Migration for Code-Derived Documentation

* **Status**: Resolved
* **Date**: 2026-09-16
* **Category**: Documentation & Developer Experience

## 1. Question

How should `semedit` move the `cmd/docgen` output from standalone HTML to Markdown rendered by Hugo and Hextra while preserving executable example extraction, direct GitHub source links, and CI drift guarantees?

## 2. Context

Hextra exposes a standard code-block copy control. The former site was generated as one complete HTML document, so the theme could not attach its normal controls to the generated `<pre>` elements. The Pages workflow now stages Hugo content and publishes Hugo's output to `dist/docs`.

The migration must preserve the existing code-derived documentation boundary described in [RQ-0018](RQ-0018-automated-code-derived-documentation.md) and [ADR-0015](../adr/0015-mkdocs-material-documentation.md). It must also avoid reintroducing manually maintained examples or creating repository files solely for the docs host.

## 3. Research Questions

1. Should `cmd/docgen` emit generated `content/docs/getting-started.md` and `content/docs/reference.md` pages or several pages grouped by capability and language?
2. Which Markdown fence and attribute structure gives Hextra reliable copy behavior for CLI commands, JSON MCP payloads, diffs, and long lines?
3. Should generated Markdown live beside hand-authored ADR/research content or under a clearly isolated generated subtree?
4. How should `make check` run generation, Hugo build validation, and published-output assertions without requiring a committed HTML artifact?
5. Which pinned Hugo Extended and Hextra versions provide reproducible local and GitHub Pages builds?
6. Can the current custom layout be represented with Hextra configuration without reintroducing a bespoke JavaScript copy implementation?

## 4. Evaluation Criteria

* Clicking the standard copy control returns the intended contents for every generated code block.
* All 11 current `txtar` workflows render without truncation or horizontal page overflow.
* Generated examples continue to link directly to their GitHub source archives.
* A clean checkout can reproduce the site with the documented build commands.
* CI detects generator/build failures, invalid navigation, or missing published pages before deployment.
* The migration does not change the executable test or MCP schema source of truth.

## 5. Findings

1. `cmd/docgen` emits generated Hugo front matter and pages (`content/_index.md`, `content/docs/_index.md`, `content/docs/getting-started.md`, and `content/docs/reference.md`) in `.scratch/docgen`, keeping executable txtar fixtures as the source of truth.
2. CLI, MCP JSON, and diff examples use fenced Markdown blocks, which Hextra renders with its standard copy control.
3. Each txtar workflow includes a GitHub blob link and a raw downloadable link; no duplicate fixture is created for the docs host.
4. Hugo Extended remains controlled by the existing Pages workflow, and Hextra `v0.12.3` is pinned in the generated Hugo module.
5. `make docgen` builds the final site into `dist/docs`, while `make verify-docs` asserts the landing, docs section, Getting Started, reference, and `.nojekyll` files. `make check` invokes `verify-docs`.

The migration satisfies the evaluation criteria. Further page splitting or custom CSS can be considered independently if the generated reference page becomes difficult to navigate.

## 6. References

* [Hextra documentation](https://imfing.github.io/hextra/)
* [Hextra installation and Hugo Module configuration](https://imfing.github.io/hextra/docs/getting-started/)
