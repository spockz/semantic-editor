<!-- This ADR records the accepted generic construct lifecycle and its bounded public catalog contract. -->
# ADR-0053: Unified Construct Lifecycle and Public Catalog

* **Status**: Accepted
* **Date**: 2026-10-03

Supersedes the conflicting future-interface proposals in the previous draft. Detailed current behavior and scope are described in ADR-0046; this record settles the public naming, capability, and catalog contract.

## Context

The registry had separate structure and statement insertion operations, alongside declaration replacement and control-flow replacement. Their overlap enlarged the MCP catalog and left kind support inconsistently described. The handover decision consolidates them under a generic construct lifecycle while retaining focused specialized operations where they provide a distinct scope, such as function-body replacement.

## Decision

The lifecycle consists of insert_construct, replace_construct, remove_construct, and move_construct. There are no insert_structure or insert_statement operations, CLI commands, or compatibility aliases. Generic insertion accepts no overwrite parameter. Existing declarations are updated through replace_construct. Whole declaration replacement selects by symbol and does not require in_function. Nested selection uses the applicable owner, discriminator, and optional construct_path.

The shared kind taxonomy includes statement/expression where implemented. Backend capability metadata is operation-specific: each backend advertises only kinds it can execute for that operation. MCP schemas consume those lists; no cross-language support is implied by the shared names. Repeated direct-statement insertion selection uses the reported construct_path and matching expected_revision; expression candidates are excluded from insertion anchors.

semantic_replace_body remains a specialized Go operation that changes a function or method body and preserves its signature. It is not renamed to replace_construct_body. It is tagged replace:function for optional catalog discovery.

move_construct expresses relocation only. It accepts destination placement, has no replacement source or overwrite option, and performs same-file reordering atomically. It does not sort declarations automatically.

The MCP catalog defaults to base: generic lifecycle, rename, navigation, verification, batch, snapshots, and undo. full exposes all registered tools and support utilities. Repeatable --expose-tool-variant verb:kind adds selected specialized variants to base; comma-separated patterns and * wildcards are accepted. Invalid patterns and patterns matching no specialized operation fail before server startup. Filtering changes only top-level discovery; specialized operations remain available to CLI and batch dispatch.

## Invariants

1. Names and schemas are sourced from the typed operation registry.
2. Catalog filtering does not delete registry operations or alter backend capabilities.
3. CLI and batch can dispatch specialized operations regardless of MCP discovery profile.
4. Insertion, replacement, removal, and movement refuse invalid or ambiguous selections before mutation.
5. Declaration insertion never silently becomes replacement.
6. Only implemented operation/backend kind combinations are advertised and covered by public CLI txtar tests.
7. File mutation preserves atomic publication and advancing modification times.

## Consequences

The generic lifecycle provides a consistent public vocabulary and keeps default MCP discovery focused. Full and filtered profiles retain access to specialized workflows without coupling discoverability to executable capabilities.
