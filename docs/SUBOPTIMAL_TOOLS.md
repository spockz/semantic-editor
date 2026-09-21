# Suboptimal MCP Tools & Dogfooding Feedback Log

This document records all instances where `semedit` MCP tools exhibited unexpected, buggy, or suboptimal behavior during internal dogfooding and real-world agent tasks.

The goal is to track:

1. **Plain Failures**: MCP operations that failed, panicked, or produced invalid output.
2. **Imperfect Transformations**: Semantic edits that succeeded partially but required an immediate follow-up manual text edit (`replace_file_content` / `write_to_file`) to clean up or touch up.
3. **Ergonomic Friction**: Tool schemas or error messages that confused LLM agents, caused parameter hallucination, or caused retry loops.

---

## Log of Suboptimal Operations & Defects

| ID | Date | Tool / Operation | Target File / Context | Observed Suboptimal Behavior | Workaround / Manual Follow-up | Root Cause & Missing Invariant | Tracking Issue / Fix |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **ST-0001** | 2026-09-16 | `semantic_insert_decl` | `internal/astedit/errors.go` | Attempting to add multiple sentinel errors to an existing parenthesized `var (...)` group. Tool created a new isolated `var` block rather than cleanly extending the existing group. | Used manual `replace_file_content` to splice all new error declarations into the existing `var (...)` block at once (ME-0030, ME-0032). | `append_group` requires precise group targeting; does not support appending to a multi-line parenthesized declaration group by keyword alone. | Needs `target_group: "var"` support in `semantic_insert_decl`. |
| **ST-0002** | 2026-09-16 | `semantic_rename` | `internal/symbol/resolver.go` | In-flight MCP server in running subagent session did not have access to newly compiled capabilities because the MCP process remained on the pre-rebuild binary. | Subagent fell back to standard text editing tools (`write_to_file`, `replace_file_content`) during feature implementation. | The running stdio MCP server cannot observe in-tree binary rebuilds (`bin/semedit`) without a live reload mechanism. | Solved conceptually by [RQ-0021](research/RQ-0021-mcp-in-tree-live-reload.md) (`--live-reload`). |
| **ST-0003** | 2026-09-16 | `semantic_organize_imports` | `cmd/docgen/main.go` | `strings.Title` deprecation fix required adding `unicode` import. `semantic_organize_imports` with auto-resolution failed to add `unicode` because the reference was inside a newly introduced helper function that had syntax errors before formatting. | Fixed manually via `replace_file_content` to add `unicode` and replace deprecated function call simultaneously. | `goimports` fails to resolve imports on files with intermediate syntax errors; lacks atomic "fix and import" transaction. | Addressed by `semantic_batch` once deployed. |
| **ST-0004** | 2026-09-16 | `semantic_insert_function` | `main.go` | Adding Cobra subcommand constructors (`newReplaceBodyCmd`, `newScaffoldFileCmd`) required registering them in `rootCmd.AddCommand(...)`. Tool successfully inserted the functions but could not register them in the caller block. | Manual `replace_file_content` edit to add `.AddCommand(...)` in `main.go`. | `insert_function` only inserts declarations; it cannot modify caller sites or register symbols in existing function bodies. | Identified as high-ROI gap: `semantic_insert_statement` / `append_call`. |
| **ST-0005** | 2026-09-16 | `semantic_replace_body` | Synthetic stub validation | When replacing a function body containing raw quotes or backticks, escaping JSON arguments inside agent tool calls occasionally resulted in unescaped newline syntax errors from the snippet parser. | Escaped strings or piped body through stdin on CLI; manual text edits in agent harness. | Snippet parser lacked whitespace/quote sanitization for outer enclosing quotes passed by LLM harnesses. | Fixed during ADR-0016 implementation by stripping enclosing quote wrappers. |
| **ST-0006** | 2026-09-21 | `mcp__codex_app__list_threads` | Codex task listing while locating the Java support orchestrator | The request rejected `limit: 100`; the API accepts at most 50, but its failure response did not include the supported bound. | Retry with `limit: 50`. | Caller supplied an unsupported pagination value; the tool schema constraint is not surfaced in the response. | Use the documented maximum of 50. |
| **ST-0007** | 2026-09-21 | `mcp__codex_app__automation_update` | Creating a heartbeat to resume the Java support orchestrator | The heartbeat create request rejected an omitted attachment target even though the desired recipient was named in the automation prompt. | Retry with `destination: "thread"` and the current orchestrator task ID. | Heartbeat attachment is a separate required control-plane field; the error did not identify the current task ID to use. | Include an explicit heartbeat target. |
| **ST-0008** | 2026-09-21 | `semedit` MCP server | Central registry ingress refactor | The configured project MCP server is not loading, so semantic edit operations are unavailable for refactoring its own Go ingress code. | Use a minimal manual patch after recording the outage. | Server startup or discovery failure remains uninvestigated at the user's direction. | Diagnose MCP loading separately; do not block the registry integration. |
| **ST-0009** | 2026-09-21 | `semedit` semantic editing MCP tools | Go rename/cache and diagnostic ingress files in this worktree | Semantic editing tools were unavailable, so required source changes could not be applied through the dogfooding interface. | Used focused manual patches and gofmt; no semantic transformation was delegated to text replacement. | Project MCP server is unavailable in this environment. | Restore MCP availability before future dogfooding validation. |

---

## Guidelines for Logging Dogfooding Deficiencies

When an agent or developer uses an MCP tool from `semedit` and encounters any of the following, an entry **must** be recorded above:

1. **Bug / Failure**: The tool returned an error or unexpected output for a valid semantic intent.
2. **Follow-up Manual Edit**: The tool modified the AST, but a subsequent manual edit was required to make the code compile, pass formatting, or adjust surrounding declarations.
3. **Inconvenient Ergonomics**: The parameter schema or error message caused the agent to fail or hallucinate parameters on its first attempt.
