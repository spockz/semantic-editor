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
   * `semantic_extract_function` / `semantic_extract_variable` (extracts within the selected backend's supported proof boundary).
   * `semantic_inline_variable` / `semantic_inline_function` (only for implemented backend combinations; a generic name does not establish Go call-inlining support).
   * `semantic_implement_contract` (generates supported contract members).
   * `semantic_organize_imports` (organizes imports in its selected scope).
   * `semantic_insert_construct`, `semantic_replace_construct`, and `semantic_remove_construct` (operate on supported structural kinds). There is no blanket CST-repair or `apply_ast_rewrite` capability; ADR-0002 defines operation-specific recovery.
2. **Explicit Routing Prompts in Tool Descriptions**: Every tool description must explicitly instruct the planner to prioritize it over text-replacement tools:
   * Example: *"Use this tool instead of `replace_file_content` whenever renaming an identifier, type, or function across one or more files. Executes deterministically via the host compiler/LSP without coordinate hunting."*
3. **Bundled Verification**: Mutations follow their backend's normalization and diagnostic-feedback contract. Validated diagnostic reuse under ADR-0060 preserves feedback without mandating fresh full sweeps. Staged children report working application and deferred verification; final publication reports actual coverage. MCP text gives actionable status/counts and a captured-report query, while bulk evidence stays structured under ADR-0042 and ADR-0073. Explicit `semantic_verify` remains available for a fresh full selected-scope capture.

## Invariants

* Raw LSP protocol names (e.g. `textDocument/*`) must never be exposed as agent tool names.
* Tool descriptions must explicitly declare priority over generic text diff tools for their respective refactoring domain.

## Consequences

* **Positive**: LLM planners naturally select semantic tools without requiring heavy system prompt persuasion; eliminates coordinate-hunting turns.
* **Negative**: Schemas must be carefully tuned to avoid false-positive tool selection for tasks requiring novel creative generation.
