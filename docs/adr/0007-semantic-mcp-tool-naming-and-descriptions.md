# ADR-0007: Semantic MCP Tool Naming & Schema Descriptions for Planner Routing

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Agent harnesses (including Antigravity, Claude Code, and Cursor) route tasks using planner heuristics that match user intent against tool names and schema descriptions.

Antigravity’s observed guidance explicitly states:
> *"Keep tool descriptions and schemas semantic so the planner routes symbol resolution directly to the LSP."*

If tools are named generically (e.g. `edit_code`, `run_command`, `ast_tool`) or named after raw LSP protocol methods (e.g. `textDocument_rename`, `workspace_executeCommand`), the LLM planner fails to realize they replace text edits. Consequently, the model defaults to its habituated tools (`replace_file_content`, `str_replace_editor`, `apply_diff`).

## Decision

1. **Semantic Intent Naming**: All mutating tools exposed via the Model Context Protocol (MCP) must use clear, intent-level semantic verbs:
   * `semantic_rename` (replaces raw symbol text search/replace across packages).
   * `extract_function` / `extract_variable` (extracts scoped statements into a new function/variable with compiler-inferred parameters and returns).
   * `inline_symbol` (inlines variable or method calls deterministically).
   * `implement_interface_members` / `generate_trait_stub` (generates required method stubs).
   * `organize_imports` (adds missing and removes unused imports).
   * `apply_ast_rewrite` (executes structural CST patterns for uncompilable or bulk edits).
2. **Explicit Routing Prompts in Tool Descriptions**: Every tool description must explicitly instruct the planner to prioritize it over text-replacement tools:
   * Example: *"Use this tool instead of `replace_file_content` whenever renaming an identifier, type, or function across one or more files. Executes deterministically via the host compiler/LSP without coordinate hunting."*
3. **Bundled Verification**: Mutating tools automatically format the code and return structured compiler diagnostics in their response, removing the need for a separate `getDiagnostics` turn.

## Invariants

* Raw LSP protocol names (e.g. `textDocument/*`) must never be exposed as agent tool names.
* Tool descriptions must explicitly declare priority over generic text diff tools for their respective refactoring domain.

## Consequences

* **Positive**: LLM planners naturally select semantic tools without requiring heavy system prompt persuasion; eliminates coordinate-hunting turns.
* **Negative**: Schemas must be carefully tuned to avoid false-positive tool selection for tasks requiring novel creative generation.
