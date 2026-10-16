# ADR-0009: Coexistence with LSP-MCP Servers via Distinct Naming & Profiles

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Agent environments (such as Antigravity or Claude Code) may already have an LSP-backed MCP server configured (e.g. `gopls mcp` or `agent-lsp`), providing read-only navigation tools like `goToDefinition` and `findReferences`.

The Model Context Protocol (MCP) does not support cross-server discovery; a server cannot query the client to see what other servers are connected. Registering duplicate tool definitions increases system prompt token overhead and can confuse the planner.

However, raw LSP-MCP tools remain coordinate-based (`line/character`), whereas `semedit`'s navigation tools are intent-based (`symbol: "Type.Method"`). Completely removing navigation would force agents back into manual line counting when no external symbol resolver is present.

## Decision

1. **Primary Identity as Semantic Engine**: The core toolset exposes checked mutations and verification, including `semantic_rename`, `semantic_organize_imports`, `semantic_extract_function`, and `semantic_verify`. Atomic publication is per file; the catalog does not imply a multi-file transaction.
2. **Distinct Intent-Based Naming**: Navigation tools in `semedit` use explicit intent names that never collide with raw LSP methods:
   * Use `semantic_lookup` (takes a qualified identifier string) instead of `goToDefinition`.
   * Use `semantic_find_references` instead of `findReferences`.
3. **Separate Execution and Catalog Profiles**: `--profile=mutations-only` remains an orthogonal execution filter for read-only tools. `--tool-profile=base` is the default MCP catalog and exposes generic construct insertion, replacement, removal, and movement; rename; lookup, inspect, outline, references; verify; batch; snapshot; and undo. `--tool-profile=full` exposes every registered operation and support utility. Live reload remains opt-in.
   Specialized operations stay registered, executable through the CLI, and eligible for batch dispatch regardless of catalog discovery. The repeatable `--expose-tool-variant verb:kind` flag selectively adds specialized variants to the base catalog. Values accept comma-separated patterns and `*` wildcards, such as `insert:*`, `*:function`, or `insert:function,insert:type`. Invalid patterns and patterns matching no registered variant fail before the server starts. Variant metadata is explicit on each operation and discovery filters only the top-level catalog; it does not change backend capabilities or batch schemas.
4. **Dynamic Language Scoping at Initialization**:
   During the `initialize` handshake, the server extracts workspace context (`rootUri`, `workspaceFolders`, or client `initializationOptions`) and identifies the enabled language backends. The tool schemas returned to the client (in `tools/list` and capability responses) are dynamically scoped:
   * **Language-Specific Tool Availability**: Tools with handlers only for a specific language (e.g. build dependency management, framework-specific actions) are omitted when that language is absent.
   * **Dynamic Parameter Enums**: Operation parameter schemas with dynamic constraints (such as `kind` in `semantic_replace_construct` or `access_modifier` in insertion tools) emit `enum` definitions restricted strictly to the valid constructs/modifiers supported by the active language backend, enabling LLM token decoders to enforce valid tokens via logit masking.
   * **Dynamic Change Propagation**: When workspace focus shifts or an explicit language context changes, the server emits `notifications/tools/list_changed` so the client re-requests the adapted schema definitions.

   Scope each operation to the enabled workspace backends that implement it.
   When an operation supports multiple active languages, advertise the union
   of their supported choices. A mixed Go/Java workspace does not make Java
   modifiers available to an operation implemented only for Go.

   Resolve the selected file or explicit language when executing a request;
   that backend validates its own modifiers and construct constraints. A
   choice valid for another supported backend can therefore appear in the
   union while being invalid for the selected file. Workspace discovery does
   not establish every file's valid argument combinations.

   Derive direct-tool and batch-child schemas from the same registry metadata.
   Keep automatic language selection distinct from concrete backend support.
   Omit an example when it contradicts the scoped parameter contract.
   Omit an operation whose required dynamic choices cannot be supplied by
   registered backends; keep other available operations in the catalog.

   Enforce the explicit language allowlist during execution, including
   automatic verification and the final batch boundary. Reject explicitly
   selected disabled source languages before normalization. Exclude disabled
   backends from built-in formatting, LSP hooks, and diagnostics; report their source
   coverage as disabled. No enabled verifiable source remains an error.
   Project shell hooks retain their configured policy over the allowed source
   scope; the language allowlist does not interpret or rewrite shell commands.

## Invariants

* `semedit` tool names must never collide with standard LSP method names.
* The base catalog exposes the generic editing lifecycle and core navigation and workspace tools. Full adds specialized operations and support utilities. Variant filters add only tagged specialized operations.
* Catalog filtering never deletes registry operations or changes CLI or batch dispatch.
* Tool schemas and parameter enums presented after `initialize` must reflect the capabilities of supported enabled workspace backends without including choices from unrelated backends.
* An operation's choices are the union of its supported enabled workspace backends; execution validates the selected backend. Broad mixed-language unions are valid, while choices from backends with no implementation of that operation are not.

## Consequences

* **Positive**: Seamless coexistence with existing LSP-MCP servers; eliminates planner confusion; allows users to minimize context token footprint via profiles; enforces schema correctness via language-scoped logit masking from initialization.
* **Negative**: Introduces dynamic schema generation in the MCP server layer requiring reactive updates via `notifications/tools/list_changed` when languages switch.
