# ADR-0002: Operation-Specific Analysis and Recovery

* **Status**: Accepted
* **Date**: 2026-09-15

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Clarified 2026-10-06: typed operation-language requirements and exact broken-code bitmaps govern published tolerance; try supported normalization and allow independent edits instead of requiring a blanket CST fallback. |
| description | Agents routinely edit code that has syntax, import, dependency, or type errors. Language servers can retain partial syntax and symbol information in that state, but each operation needs different facts. A source outline, import cleanup, local AST edit, and cross-package refactoring do not share one compilation precondition. Document symbols can be available while a semantic rename lacks sufficient binding information. |
| role | architecture |
| scope | project |
| area | backend |
| languages | generic |

## Context

Agents routinely edit code that has syntax, import, dependency, or type errors.
Language servers can retain partial syntax and symbol information in that state,
but each operation needs different facts. A source outline, import cleanup,
local AST edit, and cross-package refactoring do not share one compilation
precondition. Document symbols can be available while a semantic rename lacks
sufficient binding information.

The original decision specified a blanket compiler-to-CST fallback. That wording
overstated the implementation requirement. The intended recovery behavior has
two parts: try available mechanical normalization that can repair errors, and
allow operations whose required analysis is independent of the broken code.
Agents already have file-editing tools for repairs outside semedit's operations.

## Decision

Use the implemented parser, compiler, language-server, and normalization tools
according to each operation's requirements. Replace the original blanket CST
fallback requirement with explicit operation-language contracts in the central
registry described by [ADR-0034](0034-central-operation-registry.md).

Each supported pair records its analysis scope, preconditions, capabilities,
and tolerance of existing errors. MCP descriptions, CLI command help, and
generated documentation consume that same metadata. Describe the implemented
adapter's capabilities rather than every capability of its upstream server.

Use closed enums for analysis scopes and prerequisites, flags for capabilities
and guarantees, and explicit request variants for conditional modes. Broken-code
tolerance uses six fault flags: syntax and type errors in the target file, another
file in its package, or another package. Three disjoint 64-bit sets classify each
exact combination as supported, rejected, or conditional; an unclassified
combination is unknown. Conditional cases name their recovery requirement.
Do not infer support for combined faults from individually supported faults.
Support means the operation can meet its declared scoped guarantees when its
other prerequisites hold; it does not promise a clean project or automatic repair.

Try supported formatting, import organization, and configured normalization
before verification, subject to their own syntax, tool, scope, and trust
requirements. [ADR-0054](0054-project-verification-hooks.md) defines the
normalization and checking lifecycle; check-only verification skips writes.
Normalization may repair compilation errors but does not promise arbitrary
syntax repair.

Allow an operation when its own analysis and safety checks succeed. Local Go
AST edits require the target file to parse, without a clean module typecheck.
Typed operations can require a package or active-module proof. LSP operations
require the server to return usable analysis and validated responses; partial
analysis does not establish universal rename safety.

Repairs beyond supported operations remain with the agent's existing editing
tools. This decision does not require semedit to duplicate every operation in
ast-grep, introduce manual-edit operations, or manufacture missing type facts.

## Invariants

* The registry describes every implemented language-operation pair, including
  conditional requirements such as safe-delete or cross-package movement.
* Existing compilation errors do not imply a global editing ban. An operation
  may still fail when errors affect its declared analysis or publication scope.
* Syntax recovery, formatting success, and semantic refactoring safety are
  distinct capabilities. Missing analysis or diagnostic coverage is explicit.
* semedit delegates to existing parsers, language servers, and normalization
  tools; it does not implement a second generic repair engine.

## Consequences

* Agents can choose an available repair or independent edit using the published
  requirements, without assuming the entire project must compile first.
* Typed operations retain conservative failure behavior when their proofs are
  incomplete; diagnostic operations can report existing errors.
* Capability changes require updating the registry contract and its tests.
  Public descriptions and documentation follow from that source.
* Deterministic supported and rejected combinations can generate CLI scenarios.
  Conditional cases need controlled recovery conditions; unknown cases provide
  no success or failure oracle.
