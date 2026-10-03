# Suboptimal Model Context Protocol (MCP) Tools & Dogfooding Feedback Log

## Critical text worker: semantic batch verification refused workspace hook

- Tool: `semantic_batch` with two `semantic_insert_function` edits.
- Target: `internal/mcp/presentation.go`; verification also formatted four unrelated hidden tests under `testdata/bench-oracles/task-12-request-identity-migration`.
- Observed failure: both insertions persisted, then automatic verification refused the configured `golangci-lint` hook because workspace trust was absent. It also reordered imports in four unrelated fixture files and reported semantic/type diagnostics after insertion; whether those diagnostics predated the attempt is unverified.
- Workaround: record the event, restore only the four unrelated import blocks from the captured preimage, then use the repository's trusted `make check` gate and explicitly organize imports in the target file.
- Root cause: batch final verification requires trust for the configured hook, while its normalization ran before that refusal and was not scoped to changed files.

## Critical text worker: semantic function insertion rejected two declarations

- Tool: `semantic_insert_function`.
- Target: `internal/mcp/presentation.go`.
- Observed failure: the snippet contained two functions; the tool accepts exactly one declaration, and no source mutation occurred.
- Workaround: insert the two helper functions sequentially with `semantic_batch`.
- Root cause: the insertion operation validates one declaration per call.

## Critical text worker: batch executor symbol lookup missed

- Tool: `semantic_inspect_symbol`.
- Target: `internal/mcp/batch.go`, selector `Server.executeBatchEntry`.
- Observed failure: symbol not found; no source mutation occurred.
- Workaround: use `semantic_outline` for `internal/mcp/batch.go` and inspect its actual declarations.
- Root cause: the guessed method name did not match the implementation's symbol name.

## Critical text worker: semantic mutation targeted coordinator checkout

- Tool: `semantic_scaffold_file` followed by `semantic_batch` in the shared semedit MCP connector.
- Target: coordinator checkout `internal/mcp/presentation.go` and four hidden test import blocks, instead of the assigned `.scratch/worktrees/critical-text-presentation` worktree.
- Observed failure: the semedit connector resolved its workspace to the coordinator checkout even though the shell was in the assigned worker worktree; the mutations and verification therefore ran against the wrong tree. The coordinator recovered the helper as a scratch artifact and restored the captured import changes.
- Workaround: stop semantic MCP mutations and invoke the semantic CLI through `exec_command` with the exact worktree as `workdir` and a worktree-local Go build cache.
- Root cause: the MCP semantic tools do not expose a workspace-root selector in this sub-agent context.

## Retrospective: inspection guessed the verification handler name

- Tool: `semantic_inspect_symbol`.
- Target: `internal/operation/wire_backend.go`, selector `runVerify`.
- Observed failure: symbol not found; no source mutation occurred.
- Workaround: use the actual `verifyRun` declaration identified by the preceding structural search.
- Root cause: the coordinator submitted an inferred handler name despite already having the real name in tool output. This was a caller mistake, not missing discovery information.

This document records unexpected, buggy, or suboptimal behavior in `semedit` MCP tools during internal dogfooding and real-world agent tasks. Abbreviations used on this page include large language model (LLM), application programming interface (API), command-line interface (CLI), abstract syntax tree (AST), identifier (ID), JavaScript Object Notation (JSON), architecture decision record (ADR), and return on investment (ROI).

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
| **ST-0007** | 2026-09-21 | `mcp__codex_app__automation_update` | Creating a heartbeat to resume the Java support orchestrator | The heartbeat create request rejected an omitted attachment target even though the automation prompt named the desired recipient. | Retry with `destination: "thread"` and the current orchestrator task ID. | Heartbeat attachment is a separate required control-plane field; the error did not identify the current task ID to use. | Include an explicit heartbeat target. |
| **ST-0008** | 2026-09-21 | `semedit` MCP server | Central registry ingress refactor | The configured project MCP server is not loading, so semantic edit operations are unavailable for refactoring its own Go ingress code. | Use a minimal manual patch after recording the outage. | Server startup or discovery failure remains uninvestigated at the user's direction. | Diagnose MCP loading separately; do not block the registry integration. |
| **ST-0009** | 2026-09-21 | `semedit` semantic editing MCP tools | Go rename/cache and diagnostic ingress files in this worktree | Semantic editing tools were unavailable, so agents could not apply required source changes through the dogfooding interface. | Used focused manual patches and gofmt; agents did not delegate any semantic transformation to text replacement. | Project MCP server is unavailable in this environment. | Restore MCP availability before future dogfooding validation. |
| **ST-0010** | 2026-09-21 | `semantic_rename` | `config/config.go` | Go backend could not resolve `Config.Port` or `Port` for rename. | Applied the narrow field rename manually, then used semantic insertion for the helper and semantic import organization. | The resolver scanned only functions, types, and values, not declared named struct fields. | Fixed: qualified struct-field resolution plus CLI txtar regression coverage. |
| **ST-0011** | 2026-09-21 | `semantic_rename` | `tools/benchmark-harness/runner.go` exported type `ClassifierSet` | Reported success after renaming the declaration but left cross-file Go references unchanged, leaving the package uncompilable. | Completed the reference rename manually. | An unwritable Go build cache let `gopls` perform an incomplete rename without a failing exit status. | Fixed by `109d141` with a writable cache fallback, diagnostic failure reporting, and CLI coverage. |
| **ST-0012** | 2026-09-22 | `semantic_rename` | `config/config.go` field `Config.Port` | Go backend failed before resolving the symbol because the workspace loaded a relative GOPATH and reported no gopls views. | Will apply a focused manual rename after recording this failure, then continue with semantic operations where available. | The MCP process inherited an invalid relative GOPATH for this benchmark worktree. | Restore an absolute GOPATH or workspace-specific Go environment before future semantic renames. |
| **ST-0013** | 2026-09-22 | `semedit` semantic editing MCP tools | `tools/benchmark-harness/driver.go`, `internal/mcp/server.go` | Semantic editing tools were unavailable in the isolated benchmark worktree session. | Applied focused manual patches and gofmt. | MCP server discovery was unavailable in this delegated session. | Restore MCP availability before future dogfooding validation. |
| **ST-0014** | 2026-09-22 | `semantic_replace_body` | `cmd/docgen/main.go`, `writeHugoConfig` | Replacing a function body containing raw Go string literals with backticks failed with `syntax error: illegal character U+005C '\\'`. | Applied a focused patch to the function body after recording the failure. | The snippet parser received escaped delimiters from the tool harness instead of preserving the raw string literal syntax needed by the Go parser. | Escape handling should preserve raw-string delimiters in replacement snippets. |
| **ST-0015** | 2026-09-22 | `semantic_insert_function` | task-11 large fixture `audit/transaction.go`, anchor `(*BufferedSink).Begin` | Rejected a valid pointer-receiver method anchor as not found. | The agent retried with `BufferedSink.Begin`; the operation then succeeded. | Declaration insertion parsed receiver names independently and did not accept the pointer spelling already accepted by shared symbol resolution and `semantic_replace_body`. | Normalize relative-placement anchors through `symbol.ParseIdentifier`; cover direct and CLI pointer-receiver insertion. |
| **ST-0016** | 2026-09-22 | `semedit` semantic documentation tools | `cmd/docgen/benchmarks.go`, `cmd/docgen/main.go` | This session exposed no callable semantic MCP editing tools for this documentation-generator change. | Used focused source patches and will run the repository verification suite. | This session did not expose the semedit MCP server. | Make semantic editing MCP tools available to future repository sessions. |
| **ST-0017** | 2026-09-23 | `semantic_scaffold_file`, `semantic_replace_body`, `semantic_insert_decl` | `tools/benchmark-harness/` in the requested isolated worktree | Calls reported success but were bound to the primary checkout, so the scaffold landed on `main` and the replacement did not touch the isolated worktree. | Reverted only the accidental primary-checkout files and applied focused patches in the requested worktree. | Semantic MCP calls do not honor the delegated worktree path in this session. | Bind semantic operations to the calling worktree before future delegated edits. |
| **ST-0018** | 2026-09-23 | `semantic_insert_function` | `tools/benchmark-harness/bench_test.go` | Rejected insertion of an exported Go test function when `access_modifier` was explicitly set to `private`. | Retry using the required `public` access modifier. | The tool validates access modifiers against Go identifier casing and does not infer/override an explicit mismatch. | Use `public` for exported `Test*` functions; report casing mismatch before aborting the insertion. |

| **ST-0019** | 2026-09-23 | `semantic_replace_body` | `internal/pipeline/pipeline.go` | The replacement implementation changed existing import behavior: an explicit alias request did not update the alias of an already imported path and could add a duplicate import. | Restore match-by-path alias updates and add an observable `OrganizeImportsWithOptions` regression test. | The replacement body did not preserve the original existing-import alias update branch. | Preserve alias updates when replacing explicit import handling. |
| **ST-0020** | 2026-09-23 | `semantic_organize_imports` | `internal/backend/java/java.go` in the requested worktree | Rejected the moved Java implementation because the preceding mechanical qualification pass had rewritten struct field labels as package selectors, leaving invalid Go syntax. | Correct the field labels and add the missing contract qualifiers before retrying import organization. | The identifier rewrite treated all bare tokens alike instead of distinguishing declaration fields from package-level references. | Use syntax-aware qualifier rewrites when extracting concrete backend packages. |
| **ST-0021** | 2026-09-23 | `semantic_rename` | `internal/backend/java/java.go` in the requested worktree | Could not resolve the local parameter `backend` in `WithJavaSessionFactory` for a scope-local rename. | Rename the closure parameter and its references with a focused patch. | Workspace rename did not resolve an unexported identifier in the moved package when given its bare symbol name. | Use a scope-aware local rename or report the declaration scope required for resolution. |
| **ST-0022** | 2026-09-23 | `semantic_organize_imports` | `internal/backend/golang/backend.go` in the requested worktree | Import insertion associated the existing `GoBackend` doc comment with a newly added import, so revive no longer recognized the type comment. | Move the comment back above `GoBackend` after imports. | Import insertion did not preserve the comment's attachment to the declaration when adding a new import. | Preserve declaration comments during import organization. |
| **ST-0023** | 2026-09-23 | `semantic_organize_imports` | `internal/backend/rust/backend.go` in the requested worktree | Rejected the moved Rust implementation because mechanical contract qualification rewrote anonymous struct field names such as `Range` into invalid package-qualified fields. | Restore the field names and qualify only contract types. | The identifier rewrite did not distinguish anonymous struct field keys from referenced types. | Use syntax-aware qualification when extracting concrete backend packages. |
| **ST-0024** | 2026-09-23 | `semantic_organize_imports` | `internal/backend/haskell/backend.go` in the requested worktree | Rejected the extracted Haskell implementation after an edited factory adapter retained an extra closing parenthesis. | Remove the unmatched delimiter and retry import organization. | The focused runtime-config adaptation changed a function literal but left its prior wrapper delimiter. | Re-run Go parsing before semantic import organization after changing closure signatures. |
| **ST-0025** | 2026-09-24 | `semantic_insert_function` | `cmd/docgen/benchmark_aggregate.go` in the base checkout, although the requested target was `.scratch/worktrees/benchmark-browser` | Rejected the intended function because its Go raw-string HTML constant contained JavaScript template-literal backticks, terminating the Go literal early. | Reworked the JavaScript messages to avoid nested template literals, then retried in the correct worktree with a manual patch after confirming the semantic server targets the base checkout. | The snippet nested raw-string delimiters, and the semantic server resolved relative paths against the base checkout rather than the requested worktree. | Do not use the semantic server for a separate worktree unless its root can be explicitly selected; avoid nested backticks in Go source snippets. |
| **ST-0026** | 2026-09-24 | `semantic_insert_function` | `cmd/docgen/benchmark_aggregate.go` in the base checkout, although the requested target was `.scratch/worktrees/benchmark-browser` | Rejected the source because it contained both a constant and a function, while the operation accepts exactly one function declaration. | Split the constant and function into separate insertions; after detecting incorrect checkout targeting, removed the unintended base-checkout file and continued in the requested worktree. | The caller did not match the tool's single-declaration input contract, and the tool targeted a different checkout. | Use declaration insertion for constants and function insertion for functions only after confirming the tool's workspace root. |
| **ST-0027** | 2026-09-24 | `semantic_insert_function` | `cmd/docgen/benchmark_aggregate.go` in the base checkout, although the requested target was `.scratch/worktrees/benchmark-browser` | Reported successful insertion but diagnostics found `writeBenchmarkBrowserAssets` redeclared in the base checkout; the intended worktree file remained empty. | Removed the unintended base-checkout file and implemented the function in the requested worktree with a focused patch. | The semantic server resolved the relative file path against the main checkout and its success response obscured that workspace mismatch. | Verify the actual absolute workspace path after semantic operations in a task worktree. |
| **ST-0028** | 2026-09-24 | `semantic_replace_body` | `cmd/docgen/benchmark_aggregate.go` in the requested worktree | Failed to open the target because the semantic server resolved the relative path under `/Users/alessandro/sources/semantic-editor`, where the worktree file does not exist. | Record the tool failure, then apply the focused source change directly in `.scratch/worktrees/benchmark-browser`. | The semantic server ignored the active worktree when resolving relative file paths. | Avoid semantic edits in this worktree until the server can target its root explicitly. |

| **ST-0025** | 2026-09-23 | `semantic_insert_case` | `internal/astedit/insert.go` in the lint worktree | A request targeting the nested `switch opts.Placement` in `calculateInsertionOffset` inserted `default` into the outer switch with the same discriminant instead. The inner switch remained non-exhaustive, and the outer fallback behavior changed. | Moved the branch into the nested switch. Added a CLI reproducer in `testdata/scripts/insert_case.txtar` that asserts an ambiguous selector leaves the file unchanged. | The switch selector matched the first switch when nested switches had the same discriminant. | Reject ambiguous matches with `ErrSwitchAmbiguous`; a source-position selector remains a possible later enhancement. |

| **ST-0026** | 2026-09-23 | `semantic_scaffold_file` | `tools/benchmark-harness/matrix_report.go` in the lint worktree | Scaffolded only `package main`, omitting the repository-required file header explaining why the file exists. | Add the required header using an atomic manual edit. | The scaffold contract infers package names but has no file-header input or repository policy awareness. | Accept an optional header comment in the scaffold request. |

| **ST-0027** | 2026-09-23 | `semantic_scaffold_file` | `internal/symbol/resolver_scan.go` in the lint worktree | Scaffolded only the package line, omitting the repository-required file purpose header. | Add the header with an atomic manual edit. | The scaffold API has no header-comment input. | Add an optional file header argument. |

| **ST-0028** | 2026-09-23 | `semantic_scaffold_file` | `tools/benchmark-harness/matrix_jobs.go` in the lint worktree | Scaffolded the package line without the required file purpose header. | Add the header using an atomic manual edit. | The scaffold API has no file-header input. | Add an optional file header argument. |

| **ST-0029** | 2026-09-23 | `semantic_insert_decl` | `internal/astedit/errors.go` while recording ambiguous switch insertion | Inserting a commented sentinel declaration into the existing `var (...)` group placed a nested `var` token inside the group and attached the new comment to the preceding sentinel, leaving invalid Go syntax. | Repair the group with an atomic source edit, then run `make check`. | The append-to-group rewrite did not unwrap a `var` declaration or preserve comment attachment. | Resolved: `semantic_insert_decl` now extracts the spec and comment from parsed Go, inserts before the next spec doc group, and preflights formatting; AST and CLI txtar regressions pass. |

| **ST-0030** | 2026-09-24 | `semantic_insert_function` | `tools/benchmark-harness/driver.go` | Inserting a helper with `placement: before_symbol` anchored before `ExecuteAgentDriver` placed it between the target function’s doc comment and declaration, attaching the comment to the helper. | Move the doc comment back above `ExecuteAgentDriver` with an atomic source edit. | The insertion anchor does not keep a Go doc comment attached to its declaration. | Preserve target declaration comments when inserting before a symbol. |

| **ST-0031** | 2026-09-25 | `semantic_insert_function` | `internal/mcp/server_test.go` while adding feedback-tool contract coverage | Rejected a `Test...` function when the request explicitly declared private access, because the exported Go identifier casing implied public access. | Retry the insertion with `access_modifier: public`; no source change was made by the rejected call. | The tool enforces agreement between Go identifier casing and the requested access modifier. | Match the access modifier to exported Go test function names. |

| **ST-0032** | 2026-09-25 | `semantic_lookup` | `tools/benchmark-harness/driver.go` | Querying the guessed symbol `Run` returned `symbol not found`; no source change was made. | Use the actual declaration name `ExecuteAgentDriver` for lookup. | The lookup request used a generic guessed symbol rather than the target function identifier. | Check the target declaration name before semantic lookup. |
| **ST-0033** | 2026-09-25 | `semantic_insert_type` | `cmd/worktreecheck/main.go` | The shared access-modifier schema accepted `package-private`, but the Go backend rejected it as unsupported before making a source change. | Retry with the Go-supported `private` modifier. | The generic schema exposes an access modifier that the Go backend does not implement. | Filter unsupported access modifiers by backend or normalize `package-private` for Go. |

| **ST-0034** | 2026-09-25 | `semantic_replace_body` | `cmd/worktreecheck/main.go` | Replacing `removeDoneWorktrees` failed with `syntax error: expected '(', found removeDoneWorktrees` before writing source. | Retry with only the function body statements, as required by the tool. | The request included the full function declaration even though the tool accepts a bare body. | Make the body-only input contract clearer in examples. |
| **ST-0035** | 2026-09-25 | `semantic_replace_body` | `cmd/docgen/benchmark_aggregate.go` | Attempted to update package-level constant `benchmarkBrowserShortcode` via `semantic_replace_body`. Operation failed with `symbol not found` because `semantic_replace_body` is restricted to function and method declarations. | An alternative tool `semantic_insert_decl` duplicates rather than replaces existing package-level constants (`ST-0036`), requiring text editing or AST constant replacement support. | `semedit` lacks a dedicated `semantic_replace_decl` or constant value updater. | Support constant/variable initialization replacement or value modification in `semantic_insert_decl`/`semantic_replace_decl`. |
| **ST-0036** | 2026-09-25 | `semantic_insert_decl` | `cmd/docgen/benchmark_aggregate.go` | Inserting an updated `const benchmarkBrowserShortcode = ...` declaration into a file that already declared `benchmarkBrowserShortcode` prepended a second declaration instead of replacing or reporting a duplicate symbol collision. | Reverted inserted declaration and applied atomic manual text edit (`replace_file_content`). | `semantic_insert_decl` blindly prepends standalone declarations without checking if the identifier already exists in file/package scope. | Detect existing symbol name and reject duplicate insertion or provide an explicit replace/update mode. |
| **ST-0037** | 2026-09-25 | `semantic_replace_decl` | `internal/operation/wire_engine_declarations.go` | The active semedit MCP tool catalog did not expose a callable `semantic_replace_decl` operation while wiring the new registry request. The attempted tool dispatch failed before reaching the server or changing source. | Continue with available semantic insertion/body tools; use a narrow atomic edit for the existing request struct field and parser/parameter wiring if no semantic operation covers those locations. | The running MCP server/catalog is older than the implementation surface in this task; it has no declaration-replacement operation callable from this session. | Refresh/reload the in-tree MCP server capability catalog when new operations are built, and keep declaration replacement available for existing package var specs. |
| **ST-0038** | 2026-09-25 | `semedit verify --path .` | `.scratch/worktrees/construct-replacements` | Verification failed with `gofmt -l .: exit status 2` after copying the Go module cache into this worktree. Direct `gofmt -l .` showed syntax errors in deliberately invalid Go fixtures under `.scratch/go/mod/golang.org/x/tools@v0.26.0`. | Used the passing `make check` gate for workspace verification and isolated the failure with direct `gofmt`; left fixture files untouched. | The Go verify handler passes a directory directly to recursive `gofmt`, which includes ignored `.scratch` dependencies instead of limiting formatting checks to project Go sources. | Enumerate workspace Go files through the repository-aware source walker, excluding `.scratch`, vendor, and dependency caches; retain stderr context when `gofmt` fails. |

| **ST-0039** | 2026-09-25 | MCP tool discovery after `make promote` | `.scratch/worktrees/adr0046-replay` | The promoted binary returned 21 tools through `tools/list`, including `semantic_replace_loop` and `semantic_replace_decl`, but a newly spawned replay subagent still received the parent session's stale callable catalog without those two operations. `semantic_reload` was also absent because the configured server lacks `--live-reload`. | The worker uses existing MCP semantic tools and the promoted `semedit` CLI for new operations; it records each unsupported read/edit fallback separately. | Promotion updates the executable but does not refresh this host session's tool declarations or the child catalog. | Expose a supported refresh/reconnect path, or launch child agents with a fresh MCP discovery from the promoted binary; advertise the active binary version and canonical workspace root. |

| **ST-0040** | 2026-09-25 | promoted CLI `replace-decl` | replay `internal/operation/wire_engine_declarations.go`, `insertDeclParams` | Replacing the existing eight-entry `[]ParameterContract` variable with multiline `source` succeeded but flattened all entries into one long line. A repeated call with one entry per input line also flattened it. | The replay worker recorded the full declaration and reflowed it atomically; main added a two-call regression and fixed the formatter in `42b2449`, then re-promoted the binary. | `parseSingleReplacementDecl` formatted the parsed declaration with a detached token file set, losing source line positions. | Resolved: format the source first, parse it with the file set passed to `format.Node`, and assert multiline layout plus adjacent functions in a regression test. |
| **ST-0041** | 2026-09-25 | `semantic_replace_body` | replay `internal/operation/wire_backend_test.go`, `TestRegisteredDefsHonorContracts` | The worker supplied an abbreviated body to change only the registry count; the tool correctly treated it as the complete replacement and removed the existing schema assertions. | The worker restored the complete original body using a second semantic call and changed only `17` to `19`; final diff confirms the assertions remain. | A whole-body operation has no narrow expression target or safeguard against unintentionally omitted sibling statements. | Add `semantic_replace_expression` or an anchored assertion update, and surface a bounded before/after scope diff prominently in the result. |
| **ST-0042** | 2026-09-25 | `semantic_scaffold_file` | replay `internal/astedit/construct_test.go` | A worktree-relative file request resolved under the primary checkout and returned `file already exists` for a file absent in the replay worktree. | Retry with the explicit `.scratch/worktrees/adr0046-replay/...` path; subsequent semantic inserts targeted the worktree. | The active MCP server root remained the primary checkout, and the child catalog did not announce that binding. | Advertise the canonical root and require or accept an explicit workspace/worktree root on every source operation. |
| **ST-0043** | 2026-09-25 | `semantic_replace_body` | `.scratch/worktrees/grouped-decl-fix/internal/astedit/decl.go`, `InsertDecl` | The first replacement request failed during MCP argument parsing with a missing-comma syntax error before source mutation. | Simplify the body payload quoting and retry. | The tool argument parser rejected escaped quote and backtick sequences embedded in the submitted body. | Retry with plain Go quoting and minimize transport-level escapes. |
| **ST-0044** | 2026-09-25 | `semantic_insert_function` | `.scratch/worktrees/grouped-decl-fix/internal/astedit/decl_test.go`, `TestInsertDecl_PreservesCommentedSentinelSpecs` | The function snippet failed validation with an unterminated string because a newline in a Go string literal was transported as a literal line break. | Retry with an escaped newline in the failure message string. | The body text transport interpreted the newline escape before snippet validation. | Keep multiline Go string literals raw and escape newlines in ordinary Go strings. |
| **ST-0045** | 2026-09-25 | `semantic_replace_body` | `.scratch/worktrees/grouped-decl-fix/internal/astedit/decl.go`, `InsertDecl` | Replacing the complete function body omitted its existing section comments and rewrote the unrelated raw-string trim expression. | Restore the comments and original trim expression with a focused atomic edit after recording the result. | Whole-body replacement only preserves surrounding comments when the caller includes them in the new body. | Review surrounding declarations and comments before body replacement. |
| **ST-0046** | 2026-09-25 | `semantic_organize_imports` | `internal/astedit/decl.go` after integrating the grouped-declaration fix | Reported success but retained the newly added standard-library `go/format` import in a separate group next to the local module import. | Move `go/format` into the standard-library group with a narrow atomic edit. | Import organization preserves existing group boundaries even when a standard-library package is placed in the local import group. | Group imports by package origin during organization. |
| **ST-0047** | 2026-09-25 | Fresh `semedit` MCP JSON-RPC handshake | `.scratch/worktrees/adr0046-replay-2` | The transport-only probe sent Content-Length framing and received JSON-RPC parse error (`id:null`, `code:-32700`). | Corrected the transport-only client to newline-delimited JSON-RPC; no repository source changed. | Client setup used LSP framing for a line-delimited JSON-RPC server. | Keep the replay transport helper aligned with server framing. |
| **ST-0048** | 2026-09-25 | `semantic_lookup` | Guessed operation registry symbols in the replay worktree | Three lookups for `EngineMutations`, `EngineDeclarations`, and `operationRegistry` returned `symbol not found`; no source was changed. | Used known declaration names and will narrow later lookups to symbols confirmed by project structure or the inspected implementation patch. | The guessed names were file-role labels rather than actual Go declarations. | Search by actual Go declaration names. |
| **ST-0049** | 2026-09-25 | `semantic_lookup` | Guessed registry symbols in the replay worktree | Three lookups for `wireEngineMutations`, `registerOperation`, and `Defs` returned `symbol not found`; no source was changed. | Confirmed the existing declaration names from successful lookup and will use only those names going forward. | Guessed registry helper names did not match the repository declarations. | Use the concrete `replaceBodyDef` and `parseReplaceBody` declarations. |
| **ST-0050** | 2026-09-25 | `semantic_insert_decl` | `internal/astedit/errors.go` | Inserting the documented `ErrSymbolCollision` sentinel failed during import organization with `expected IDENT, found var`, indicating the tool placed a nested `var` declaration inside the existing sentinel block. | Failure logged before inspecting the target; repair will use the smallest semantic declaration update if possible, then an atomic manual fix only if semantic repair cannot express the group edit. | The standalone declaration source was not unwrapped when merging into a parenthesized `var` group. | Resolved by the grouped-spec insertion fix with AST and CLI before/after regression coverage. |
| **ST-0051** | 2026-09-25 | `semantic_insert_function` | `internal/astedit/decl.go` | The request submitted `ReplaceDecl` and its helper as two top-level declarations; the tool rejected it before writing with `expected single function declaration, found 2 declarations`. | Split the request into one function declaration and use a local helper closure or a separate `semantic_insert_function` call. No source was changed by the failed call. | The tool accepts exactly one function or method declaration per request. | Keep each insertion request to one declaration. |
| **ST-0052** | 2026-09-25 | `semantic_replace_decl` | `internal/astedit/decl.go`, `DeclOptions` | The tool rejected replacing the `DeclOptions` struct with `replacement of "DeclOptions" requires a type alias`; no source was changed. | Add overwrite routing at the operation boundary, or use a logged atomic source edit only if `DeclOptions` itself must own the option. | `semantic_replace_decl` supports type aliases, not replacing an existing struct declaration. | Provide a struct-field insertion/update operation when AST fields must change. |
| **ST-0053** | 2026-09-25 | `semantic_insert_type` | `internal/operation/wire_engine_declarations.go`, `InsertDeclarationReq`; `wire_engine_mutations.go`, `ReplaceBodyReq` | Inserting new request types placed each new doc comment between the existing type comment and its declaration. `make check` then reported both existing exported types as undocumented. | Reattach each original comment directly above its original declaration with a minimal atomic edit after logging. | Insertion before a symbol did not preserve the target declaration comment attachment. | Keep existing doc comments attached when inserting types before a declaration. |
| **ST-0054** | 2026-09-25 | `semantic_replace_body` | `internal/astedit/decl.go`, `checkDeclCollision` | Replacing the helper body with two-result returns while retaining its one-result signature introduced compile diagnostics (`too many return values`). The operation reported the diagnostics; no source was written after this call. | After logging, update the helper signature with the smallest atomic edit, then update the caller using semantic body replacement. | Function-body replacement cannot change a function signature. | Add a semantic signature-edit operation or clearer distinction between body and declaration edits. |
| **ST-0055** | 2026-09-25 | `semantic_insert_function` | `internal/astedit/loop.go`, helper `loopMatches` | The request tried to traverse loop header expressions by placing `ast.Expr` values in an `ast.BlockStmt.List`, which requires `ast.Stmt`; diagnostics showed four compile errors. | Record the failure and retry the helper using `ast.Inspect` separately on each expression. | Go's AST has distinct `ast.Expr` and `ast.Stmt` interfaces, so a block cannot represent arbitrary header expressions. | Provide AST traversal utilities for expression lists or clearer type-aware examples. |
| **ST-0056** | 2026-09-25 | `semantic_insert_decl` | replay 3 `.scratch/replay-commented-collision.go` | A bare `ErrSymbolCollision = ...` assignment failed snippet validation with `expected declaration`. | Retry with `var ErrSymbolCollision = ...` and the leading comment; the corrected call succeeded. | Client omitted the required top-level declaration keyword. | Keep the full-declaration example in the tool schema. |
| **ST-0057** | 2026-09-25 | `semantic_verify` | replay 3 scratch fixture | The client supplied unsupported `files: [".scratch/replay-commented-collision.go"]`; the call instead used the default path and failed from `gofmt -l .` while traversing unrelated scratch content. | Retry with the advertised `path` parameter; no source mutation was observed. | The caller guessed a parameter, and the operation did not reject the unknown key before acting on its default path. | Reject unknown input keys and make the default workspace-wide formatting scope explicit. |
| **ST-0071** | 2026-09-27 | `semantic_inspect_symbol` | lookup in `internal/mcp/server.go` | Lookup for `executeBatch` failed because that symbol does not exist in the selected file. | Re-query the correct function name via outline; no source mutation occurred. | The caller guessed a symbol name instead of resolving it from the outline. | Resolve candidate names before inspecting declarations. |
| **ST-0072** | 2026-09-27 | `semantic_replace_body` | `internal/mcp/server.go` batch schema | Replacement used an undefined identifier for the MCP tool name, and diagnostics reported a compile error. | Replace the identifier with the literal `semantic_batch` through the semantic editor; no unrelated file changes. | The source snippet accidentally substituted a descriptive local constant that does not exist. | Check semantic-edit diagnostics and correct generated source immediately. |
| **ST-0073** | 2026-09-27 | `semantic_replace_body` | `internal/mcp/batch.go` | Lookup for `ExecuteBatch` failed because the method must be addressed by its receiver-qualified name. | Re-query the declaration as `Server.ExecuteBatch`; no source mutation occurred. | The caller omitted the method receiver qualifier required by the semantic editor. | Use the qualified method name returned by semantic outline or inspection. |
| **ST-0074** | 2026-09-27 | `semantic_insert_function` | `internal/operation/wire_backend_test.go` | The inserted assertion treated `backend.WorkspaceTrust` as a boolean, and diagnostics reported a compile error. | Inspect the workspace trust contract and correct the assertion with a struct field check. | The test author guessed the trust representation instead of inspecting its type. | Inspect field types before writing semantic assertions. |
| **ST-0058** | 2026-09-25 | `semantic_replace_decl` | replay 3 grouped var fixture, symbol `ErrReplayBase` | Source `// ErrReplayBase ...\nvar ErrReplayBase = errors.New("base")` failed with `expected IDENT, found var` while replacing one grouped spec; the file stayed unchanged. | Fixture comment was added with a logged atomic edit; `bf6e142` fixes this in the engine and adds AST plus CLI before/after regressions. | The spec extractor assumed `var` was the first formatted token, so a leading doc comment left the keyword inside the group. | Resolved in `bf6e142`; verify with a fresh promoted binary. |
| **ST-0059** | 2026-09-25 | `semantic_lookup` | replay 3 `internal/astedit/errors.go`, new `ErrDeclCollision` | Lookup returned symbol not found before the new sentinel existed. | Insert the new symbol with `semantic_insert_decl`; the corrected grouped insertion succeeded. | Expected negative lookup against the baseline, not a server defect. | Check whether a declaration exists before choosing lookup versus insertion. |
| **ST-0060** | 2026-09-25 | `semantic_insert_function` | replay 3 `replaceLoopDef` | A one-line `Def` composite snippet omitted a required comma and failed validation without writing source. | Retry with a multiline composite literal and explicit trailing commas. | Client supplied invalid Go syntax. | Include a multiline operation-definition example. |
| **ST-0061** | 2026-09-25 | `semantic_insert_function` | replay 3 `replaceLoopDef` | A second snippet failed because quotes in the nested Go example string were not escaped; no source mutation occurred. | Escape the Go string quotes and retry. | JSON transport escaping did not also escape the nested Go string literal. | Include a nested-source quoting example in the catalog. |
| **ST-0062** | 2026-09-25 | `semantic_insert_decl` | replay 3 scratch fixtures in one directory | Two fixture insertions correctly reported duplicate package symbols from a sibling `.go` file in the same `scratch` package. | Put independent fixtures in separate package directories or use distinct names; no source mutation occurred. | Client placed supposedly independent fixtures in one Go package; package-wide collision detection worked as designed. | Keep a package-scope collision example in test guidance. |

| **ST-0063** | 2026-09-25 | `semantic_scaffold_file` | replay 4 scratch declaration probe | Package inference failed in an empty scratch directory; subsequent insert/replace calls failed because the file did not exist. | Retry scaffolding with `package: "scratch"`, then run the dependent calls. | Client selected inference without a non-test Go sibling; the later failures were cascading setup errors. | Document that inference requires a sibling or explicit package. |
| **ST-0064** | 2026-09-25 | `semantic_insert_decl` | replay 4 independent scratch fixtures | A second fixture in the same Go package reused `target`; package collision detection correctly rejected it without writing. | Put the exact-byte fixture in its own package directory. | Client fixture isolation was insufficient; package-scope collision detection worked. | Keep package-scoped fixture guidance. |
| **ST-0065** | 2026-09-25 | `semantic_insert_function` | replay 4 `replaceLoopDef` | Snippet validation rejected a compact nested `Def` literal with `missing ',' in composite literal`; no source changed. | Retry with a multiline literal and trailing commas. | Client supplied invalid Go source. | Surface a multiline nested-registry example in the catalog. |
| **ST-0066** | 2026-09-25 | `semantic_replace_decl` | replay 4 `internal/astedit/decl.go`, `DeclOptions` | Rejected a valid request to add `Overwrite bool` to an existing struct: `replacement of "DeclOptions" requires a type alias`; no source changed. | Add only the field with an atomic edit after logging the failure. | Declaration replacement supports const, var, and type aliases, not struct member edits. | Add a bounded `semantic_insert_field` with duplicate detection and literal before/after fixtures. |

| **ST-0067** | 2026-09-25 | `semantic_insert_function`, `semantic_scaffold_file` | `internal/astedit/comment_anchor.go` | Inserting `normalizeInsertionOffset` succeeded but left `bytes`, `go/ast`, and `go/token` unresolved; scaffolding also omitted the required file-purpose header. | Log the failure, organize the imports, then add the required WHY header atomically. | The function insertion did not resolve required imports, and the scaffold API has no repository header input. | Improve semantic insertion import resolution and allow a file-purpose header during scaffolding. |

| **ST-0068** | 2026-09-25 | `semantic_scaffold_file` | `internal/backend/kotlin/real_server_integration_test.go` | Scaffolding created the correct package declaration but omitted the repository-required file-purpose header. | Add the WHY header atomically before verification; continue semantic insertion for the test function. | Scaffold schema has no header input, matching the previously recorded ST-0067 limitation. | Add a file-purpose header parameter to scaffolding. |

| **ST-0069** | 2026-09-25 | `semantic_insert_function` | `internal/backend/kotlin/real_server_integration_test.go` | The first insertion rejected exported Go test name `TestRealKotlinLanguageServerIntegration` with `access_modifier: private`; no source changed. | Retry with public access for the exported test function. | Caller supplied visibility inconsistent with Go identifier casing; the tool correctly enforced its invariant. | Keep explicit visibility guidance in agent examples. |

| **ST-0070** | 2026-09-26 | `semantic_insert_construct` (CLI) | `internal/capability/capability.go`, operation-key constants | The insertion failed with `expected declaration, found OpInspect`; the supplied snippet contained constant specs without a top-level `const` declaration, and no source was changed. | Deferred adding capability keys to feature wiring; no source workaround was needed. | Construct insertion validates a complete top-level declaration, while the request supplied only grouped constant specs. | Supply a complete declaration or append valid constant specs through the supported declaration workflow. |

---

## Guidelines for Logging Dogfooding Deficiencies

When an agent or developer uses an MCP tool from `semedit` and encounters any of the following, agents **must** record an entry above:

1. **Bug / Failure**: The tool returned an error or unexpected output for a valid semantic intent.
2. **Follow-up Manual Edit**: The tool modified the AST, but agents needed a manual edit after the tool operation to make the code compile, pass formatting, or adjust surrounding declarations.
3. **Inconvenient Ergonomics**: The parameter schema or error message caused the agent to fail or hallucinate parameters on its first attempt.

## 2026-09-26: semantic_insert_construct switch-case locator

- Tool: `semantic_insert_construct`
- Target: `tools/benchmark-harness/oracle.go`, `parseYAMLFrontmatter`
- Observed failure: inserting a `contexts` case after the `interactive_followups` list case returned `switch statement not found`.
- Workaround: use `semantic_replace_body` for the existing parser function to add the list handling without replacing the file or declaration.
- Root cause: case insertion did not locate the existing `switch targetSlice` when given the string case discriminator.

## 2026-09-26: semantic_scaffold_file omits required file-purpose header

- Tool: `semantic_scaffold_file`
- Target: `tools/benchmark-harness/planner.go`
- Observed behavior: scaffolding created only `package main`, with no place to provide the repository-required WHY header.
- Workaround: add the concise file-purpose header atomically before inserting constructs.
- Root cause: scaffold schema has no file-header input.

## 2026-09-26: semantic_rename argument-name mismatch

- Tool: `semantic_rename`
- Target: `tools/benchmark-harness/planner.go`, type `PlannedJob`
- Observed failure: call rejected with `param "to" is required` because the initial request used `new_name`.
- Workaround: retry with the documented `to` parameter.
- Root cause: caller used a parameter name not present in the live schema.

## 2026-09-26: semantic_rename reports transient call arity during signature migration

- Tool: `semantic_rename`
- Target: `tools/benchmark-harness/driver.go`, `prepareAgentPrompt`
- Observed failure: rename applied but verification reported one callsite with a temporary missing argument while the function signature was being migrated.
- Workaround: complete the wrapper/signature transition using semantic declaration edits, then verify the package after the migration.
- Root cause: the rename operation verified an intermediate state before dependent callsites were adapted.

## 2026-09-26: semantic_replace_construct did not resolve existing method

- Tool: `semantic_replace_construct`
- Target: `tools/benchmark-harness/driver.go`, method `(*Runner).ExecuteAgentDriver`
- Observed failure: replacement returned `symbol not found` for the visible receiver method.
- Workaround: use `semantic_replace_body` for the existing method, preserving its declaration and updating only its body.
- Root cause: method symbol lookup did not accept the unqualified method name in this request.

## 2026-09-26: semantic_insert_construct requires one declaration per call

- Tool: `semantic_insert_construct`
- Target: `tools/benchmark-harness/planner.go`, plan renderer and helper
- Observed failure: inserting two function declarations together returned `multiple declarations found in snippet`.
- Workaround: insert each declaration separately.
- Root cause: construct insertion accepts exactly one declaration per invocation.

## 2026-09-26: semantic_replace_construct rejected provider type and method together

- Tool: `semantic_replace_construct`
- Target: `tools/benchmark-harness/session.go`, `agyProvider`
- Observed failure: rejected the snippet with `multiple declarations found in snippet`; no source change was applied.
- Workaround: split the provider type and its `run` method into separate semantic construct replacements.
- Root cause: the tool accepts exactly one top-level declaration per request.

### 2026-09-26

- Tool: `semantic_replace_body`
- Target: `tools/benchmark-harness/oracle.go`
- Observed failure: the method replacement request used symbol `ExtractVariantTo`; the semantic tool requires the receiver-qualified symbol `(*Task).ExtractVariantTo`.
- Workaround: retry with the receiver-qualified method name.
- Root cause: method lookup requires an explicit receiver.
- Tool: `semantic_insert_construct`
- Target: `tools/benchmark-harness/planner_test.go`
- Observed failure: one request contained a helper, several tests, and a test writer type; insertion accepts one declaration at a time.
- Workaround: insert each declaration separately.
- Root cause: construct insertion validates a single AST declaration per request.
| **ST-0019** | 2026-09-26 | `semantic_insert_construct` | `tools/benchmark-harness/session_test.go` file-level WHY comment | Rejected a comment-only declaration snippet with `no declarations found in snippet`; an initial patch to the log also missed its exact table row. | Appended the entry after reading the actual file; will add the mandatory source comment using a focused atomic edit. | Declaration insertion accepts declaration syntax only; comment insertion is unsupported. | Support file-level comment insertion or document the limitation. |
| **ST-0020** | 2026-09-26 | `semantic_insert_construct` | `tools/benchmark-harness/runner.go` runner test executable field | Rejected inserting an individual struct field as a declaration (`expected declaration, found agyExecutable`). | Will replace the enclosing type declaration using the semantic construct tool. | The insertion operation accepts top-level declarations but not struct fields. | Support field insertion into struct types. |
| **ST-0021** | 2026-09-26 | `semantic_replace_construct` | `tools/benchmark-harness/agy_driver.go` executable selection conditional | Did not resolve the conditional using the short statement discriminator `_, err := os.Stat(agyBin)`. | Will retry with the complete condition text after recording this failure. | Conditional matching requires the parsed full source condition. | Improve construct-resolution diagnostics for short-statement conditionals. |
| **ST-0022** | 2026-09-26 | `semantic_replace_construct` | `tools/benchmark-harness/agy_driver.go` executable selection conditional | Also failed to resolve the full conditional when supplied as a discriminator. | After recording the failure, will apply a focused atomic edit to the conditional. | The tool does not identify short-init conditionals in this method. | Add short-init conditional matching. |
| **ST-0023** | 2026-09-26 | `semantic_rename` | `tools/benchmark-harness/driver.go` follow-up method | Rejected an invalid rename argument object because the tool requires `symbol` and `to` fields. | Will retry with the declared semantic rename schema after logging this failure. | The caller used stale tool parameter names. | Validate MCP inputs from current schemas. |
| **ST-0024** | 2026-09-26 | `semantic` construct operations | `tools/benchmark-harness/driver.go` obsolete helper removal | The available construct APIs do not provide deletion of a declaration, so they cannot remove the superseded runner-owned follow-up method after its loop moved into the session. | Applied a focused deletion of the dead helper after recording the limitation. | The semantic editing API supports insert, replace, and move but not declaration removal. | Add `semantic_delete_construct`. |
| **ST-0025** | 2026-09-26 | `semantic` construct operations | `tools/benchmark-harness/codex_driver.go`, `opencode_driver.go` event decode accounting | Available construct operations do not insert statements into existing scanner loops without replacing whole provider methods. | Applied focused atomic edits to count decode failures and reject empty resumed streams. | The API lacks statement insertion for existing loops. | Add `semantic_insert_statement`. |
| **ST-0026** | 2026-09-26 | `semantic_replace_construct` | `tools/benchmark-harness/driver.go` session cleanup defer | The construct matcher did not resolve the `session.close()` defer by the supplied discriminator. | After logging, apply a focused atomic edit to the defer. | Defer matching did not accept the call expression as discriminator. | Improve defer construct matching. |
- Tool: `semantic_replace_construct`
- Target: `tools/benchmark-harness/planner.go`, the executor selection branch in `ExecutePlan`
- Observed failure: the request did not locate the conditional using discriminator `ctx.Err() != nil`.
- Workaround: retry by matching the exact branch expression from the AST or use a focused declaration replacement.
- Root cause: conditional discriminators require exact backend-resolved syntax.

- Tool: `semantic_replace_body` (Stage 2 prompt policy, 2026-09-26)
- Target: `tools/benchmark-harness/session_policy.go` in the isolated policy worktree.
- Observed failure: the MCP server resolved the relative path under the original task checkout, where the file did not exist; no source was mutated.
- Workaround: use the prebuilt semedit CLI with an explicit isolated working directory. Relocated this log from the original task checkout after detecting the same working-directory error in logging.
- Root cause: the MCP server is bound to its original workspace, and tool invocations do not inherit shell working-directory changes.

- Tool: `semantic_replace_body`
  Target: `internal/backend/golang/references.go` in `.scratch/worktrees/read-inspection-verify`
  Failure: the tool resolved the relative path against the primary checkout and reported the file missing; no source changed.
  Workaround: retry using the nested worktree absolute path if supported, otherwise use the semantic CLI with an explicit workspace.
  Root cause: the connector workspace root differs from the assigned nested implementation worktree.

- Tool: `semantic_replace_body`
  Target: `internal/backend/golang/references.go`
  Failure: the tool could not resolve the method using the bare name `FindReferences`; no source changed.
  Workaround: retry with the receiver-qualified symbol `GoBackend.FindReferences`.
  Root cause: semantic symbol lookup requires the method receiver in its identifier.

- Tool: `semantic_rename`
  Target: `internal/backend/golang/references.go`
  Failure: the connector resolved the Go symbol against the primary checkout and reported it missing; no source changed.
  Workaround: invoke the repository `semedit rename` CLI from the assigned worktree with an explicit file and symbol.
  Root cause: connector workspace binding is the primary checkout, while this task owns a nested worktree.

- Tool: `semantic_insert_construct`
  Target: `internal/capability/capability.go`
  Failure: a const insertion supplied only the spec rather than a declaration, so snippet validation rejected it before edits.
  Workaround: retry with a complete `const (...)` declaration.
  Root cause: the semantic insertion API expects a full declaration for const constructs.

### 2026-09-26: semantic_insert_construct

- Target: `internal/backend/golang/references_test.go`
- Failure: one call supplied two function declarations; the tool rejected the snippet before editing because it accepts exactly one construct.
- Workaround: insert the regression tests in separate calls.
- Root cause: construct-level API enforces a single Go declaration per request.

### 2026-09-26: semantic_replace_construct target-path retry

- Target: `internal/backend/golang/references.go`
- Failure: the absolute target omitted the assigned `.scratch/worktrees/read-inspection-verify` component, so the tool could not open the file; no edit occurred.
- Workaround: retry with the complete assigned worktree path.
- Root cause: an incomplete absolute path.

### 2026-09-26: semantic_replace_construct method selector retry

- Target: `internal/backend/golang/references.go`
- Failure: the method selector was supplied as `FindReferences`; the tool could not resolve it because the receiver is part of the symbol identity. No edit occurred.
- Workaround: retry with receiver-qualified symbol `(GoBackend).FindReferences`.
- Root cause: semantic method lookup requires the receiver name.

### 2026-09-26: semantic edit invoked from wrong worktree

- Tool: `semantic_replace_body` and `semantic_insert_construct`
- Target: `internal/backend/java/java.go` and `internal/backend/java_test.go`
- Failure: the semantic tools successfully applied the Java rename compatibility comment and regression in the original task checkout instead of the assigned integration worktree; those test results did not exercise the integration branch. The root preserved the patch and restored the original checkout clean.
- Workaround: reapply only in `.scratch/worktrees/read-inspection`, explicitly pass that workdir to every command, and verify `pwd` plus `git branch --show-current` before each tool invocation.
- Root cause: the semantic tool used the active checkout binding, which differed from the nested worktree selected for Stage 4.

### 2026-09-26: semantic local variable replacement unsupported

- Tool: `semantic_replace_construct`
- Target: `internal/backend/java/java.go`, local variable `root` in `resolveJavaReadSelection`
- Failure: the tool could not find the local declaration when asked to replace `root := ""` with `var root string`; no edit occurred.
- Workaround: replace the containing function body with `semantic_replace_body` so the local declaration is updated while preserving the function logic.
- Root cause: construct replacement does not resolve this local declaration selector.

### 2026-09-26: no semantic function-deletion operation

- Tool: semantic editing tool catalog
- Target: `internal/backend/java/java.go`, obsolete `javaCharacterOffset`
- Failure: no exposed semantic operation deletes an unused function declaration after coordinate conversion moved to `readlsp.ByteOffset`.
- Workaround: remove only that function with a narrow atomic patch; keep the shared converter as the sole implementation.
- Root cause: the current semantic tool set supports construct insertion/replacement but not declaration deletion.

### 2026-09-26: semantic_insert_construct rejected multiple functions

- Tool: `semantic_insert_construct`
- Target: `internal/backend/rust/backend.go`
- Failure: the tool accepts one declaration per call and rejected a snippet containing five helper functions; no edit occurred.
- Workaround: insert each function separately with explicit absolute worktree paths.
- Root cause: the edit request grouped independent declarations into a single-function operation.

### 2026-09-26: semantic target omitted assigned worktree

- Tool: `semantic_replace_construct`
- Target: original checkout `internal/backend/rust/backend.go`
- Failure: an absolute source path omitted `.scratch/worktrees/read-inspection`, so the tool applied the `Detail` field edit to the parent checkout. The accidental field was removed immediately and the exact patch was saved at `.scratch/wrong-target-rust-field.patch`.
- Workaround: use relative source paths with the prebuilt semedit CLI from the assigned integration worktree, and verify the worktree branch before editing.
- Root cause: the semantic tool was bound to the original checkout and the manually assembled path was incomplete.

### 2026-09-26: no semantic field-level insertion operation

- Tool: semantic editing tool catalog
- Target: `internal/backend/rust/backend.go`, `rustDocumentSymbol.Detail` and `decodeRustDocumentSymbol`
- Failure: available semantic operations edit complete declarations but provide no operation for adding one struct field or one decode clause without replacing the whole function.
- Workaround: apply two narrow atomic patches to the selected struct and JSON decoder; all function additions and body changes use the prebuilt semedit CLI.
- Root cause: semantic editing does not expose field-level or statement-level insertion for these constructs.

### 2026-09-26: no semantic capability-map entry insertion operation

- Tool: semantic editing tool catalog
- Target: `internal/backend/rust/backend.go`, Rust `CapabilityMatrix`
- Failure: the available Go semantic operations do not insert one keyed entry into a map literal.
- Workaround: apply a narrow atomic patch to the operations map; use semedit for the capability method body.
- Root cause: the semantic edit surface has no map-entry insertion operation.

### 2026-09-26: no semantic field or capability-map insertion operation for Scala

- Tool: semantic editing tool catalog
- Target: `internal/backend/scala/backend.go`, `scalaDocumentSymbol` and `CapabilityMatrix`
- Failure: available Go semantic operations do not add one field to an existing struct or one keyed capability entry to a map literal.
- Workaround: apply narrow atomic patches to the struct, decoder, and operations map; use semedit for function additions and body changes.
- Root cause: the semantic edit surface has no field-level or map-entry insertion operation.

### 2026-09-26: no semantic function-deletion operation for obsolete position converters

- Tool: semantic editing tool catalog
- Target: `internal/backend/rust/backend.go` and `internal/backend/scala/backend.go`, old UTF-16 byte-offset helpers
- Failure: the available semantic tools replace function bodies and insert declarations but cannot delete an unused function declaration.
- Workaround: remove only the obsolete character-offset functions and their now-unused imports with narrow atomic patches after delegating conversion to `readlsp.ByteOffset`.
- Root cause: the semantic edit surface has no declaration-deletion operation.

### 2026-09-26: duplicate semantic test type insertion

- Tool: `semedit insert-type`
- Target: `internal/backend/rust_test.go`, `statefulRustSession`
- Failure: an earlier successful insertion was repeated after losing track of the file update, creating a duplicate type declaration that semedit diagnostics reported. The duplicate was immediately removed; one intended declaration remains.
- Workaround: inspect the current file before retrying semantic insertions and remove only the duplicate declaration.
- Root cause: the first insertion result was not checked against the live source before a retry.

### 2026-09-26: semedit removed imports before their references existed

- Tool: `semedit imports` and `semedit replace-body`
- Target: `internal/backend/rust_test.go`, source snapshot regression
- Failure: semedit imports organized the test before the new code referenced `pipeline`, SHA-256, and hex imports, so it removed those unused imports; a subsequent body replacement reported undefined symbols.
- Workaround: rerun semedit imports after the test body was inserted, which restored the required imports and resolved the diagnostics.
- Root cause: imports are cleaned against the file state at operation time and intentionally remove imports not yet referenced.

### 2026-09-26: semantic body selector required receiver qualification

- Tool: `semantic_replace_body` through the prebuilt semedit CLI
- Target: `internal/backend/kotlin/backend.go`, `(*KotlinBackend).Capabilities`
- Failure: selecting the method by bare name `Capabilities` did not resolve its receiver method; no edit occurred.
- Workaround: retry with the fully qualified selector `(*KotlinBackend).Capabilities`.
- Root cause: the CLI body selector requires receiver qualification for this method.

### 2026-09-26: semantic struct extension exposed positional test fixtures

- Tool: `semantic_replace_construct` through the prebuilt semedit CLI
- Target: `language_txtar_test.go`, `fakeDocumentSymbol`
- Failure: after adding optional detail/URI/children fields for hierarchical Kotlin outline fixtures, package test compilation reported too few values in existing positional literals; no runtime test ran.
- Workaround: convert the affected fake symbol fixtures to keyed literals.
- Root cause: Go positional struct literals require values for every field when a test helper struct grows.

- Tool: `semantic_replace_construct`
  - Target: `cmd/docgen/hugo.go`
  - Failure: Attempted to replace the local `config` short declaration to add Hugo `relativeURLs`; tool rejected the operation because it requires an enclosing `function` parameter.
  - Workaround: Use a supported semantic operation on the enclosing function.
  - Root cause: The tool does not support direct selection of a local declaration without identifying its containing function.

- Tool: `semantic_replace_construct`
  - Target: `cmd/docgen/hugo.go`
  - Failure: Retried local `config` declaration replacement with its containing function; tool reported `unsupported Go construct kind "decl"`.
  - Workaround: Use `semantic_replace_body` on `writeHugoConfig`.
  - Root cause: This operation supports control-flow constructs rather than local declarations.

### 2026-09-26: Make outline cannot map continued declarations

- Tool: `semantic_outline`
- Target: `Makefile` and `tools/benchmark-harness/Makefile`
- Observed failure: both selected-file outline requests failed with `make read projection cannot map continued declaration` (lines 59 and 69 respectively).
- Workaround: no outline workaround applied; report the limitation and preserve the request results.
- Root cause: the Make read projection cannot map continued declarations to source ranges.

### 2026-09-26: semantic function insertion rejected test visibility

- Tool: `semantic_insert_function`
- Target: `tools/benchmark-harness/bench_test.go`, `TestCodexTerminationOrigin`
- Failure: insertion rejected because explicit `private` access modifier conflicts with the exported casing required for a Go test function; no file was changed.
- Workaround: retry insertion with inferred visibility.
- Root cause: the requested visibility did not match Go test function naming conventions.

- Tool: `semantic_insert_function`
  - Target: `tools/benchmark-harness/bench_test.go`, `TestCodexTerminationOrigin`
  - Failure: retry without explicit visibility was rejected because `private_end` placement violates the section policy for the exported Go test function; no file was changed.
  - Workaround: insert at file end, outside a visibility-specific section.
  - Root cause: private section placement is incompatible with exported test identifiers.

- Tool: `semantic_replace_construct`
  - Target: `cmd/docgen/benchmark_aggregate.go`, `writeBenchmarkBrowserViewerAssets`
  - Failure: selector `writeGeneratedFile(shortcodePath` did not match an `if` construct; no source change was made.
  - Workaround: retry using a discriminator supported by the construct selector.
  - Root cause: the construct matcher did not accept the full call expression as a discriminator.

- Tool: `semantic_replace_construct`
  - Target: `cmd/docgen/benchmark_aggregate.go`, `writeBenchmarkBrowserViewerAssets`
  - Failure: selector `err := writeGeneratedFile(shortcodePath, []byte(benchmarkBrowserShortcode))` also failed to match the `if` construct; no source change was made.
  - Workaround: retry with a unique identifier discriminator.
  - Root cause: the selector appears to match condition text more narrowly than the full construct expression.

- Tool: `semantic_replace_construct`
  - Target: `cmd/docgen/benchmark_aggregate.go`, `writeBenchmarkBrowserViewerAssets`
  - Failure: selector `benchmarkBrowserShortcode` was not recognized inside the selected function; no source change was made.
  - Workaround: replace the containing function body using a different semantic operation.
  - Root cause: this function generated-write condition could not be selected by the construct operation.

- Tool: `semantic_replace_construct`
  - Target: `cmd/docgen/benchmarks_test.go`, `TestWriteBenchmarkBrowserAssetsPreservesDynamicFieldsAndCopiesPerspectiveAssets`
  - Failure: two identical `strings.Contains` conditions made the selected `if` ambiguous; no source change was made.
  - Workaround: retry with the construct path returned by the tool.
  - Root cause: the test function has two containment checks with the same condition.

- Tool: `semantic_replace_construct`
  - Target: `cmd/docgen/benchmarks_test.go`, shortcode assertion in `TestWriteBenchmarkBrowserAssetsPreservesDynamicFieldsAndCopiesPerspectiveAssets`
  - Failure: replacement was rejected because it contained two statements instead of one complete `if` construct; no source change was made.
  - Workaround: express expectation normalization inside the single replacement condition.
  - Root cause: this operation accepts exactly one complete construct.

- Tool: `semantic_rename`
  - Target: `cmd/docgen/benchmarks_test.go`, `TestRenderBenchmarkDocumentationSelectsBestPairsAndPreservesRunPages`
  - Failure: workspace rename could not resolve the test function symbol; no source change was made.
  - Workaround: retain the existing test name.
  - Root cause: the rename backend did not index or resolve this test declaration for the request.

### 2026-09-26: semantic type overwrite duplicated Server

- Tool: `semantic_insert_structure` with `kind=type` and `overwrite=true`
- Target: `internal/mcp/server.go`, `Server`
- Failure: the tool inserted a second `Server` declaration instead of replacing the existing type, leaving a duplicate declaration and an unresolved `toolMetricsRecord` field type.
- Workaround: undo the insertion before applying a different edit strategy.
- Root cause: the overwrite option did not replace an existing struct type in this operation.

### 2026-09-27: semantic type overwrite duplicated ScaffoldOptions

- Tool: `semantic_insert_type` with `overwrite=true`
- Target: `internal/astedit/scaffold.go`, `ScaffoldOptions`
- Failure: the tool inserted a duplicate type instead of replacing the existing struct declaration, producing Go redeclaration diagnostics.
- Workaround: remove the duplicate declaration and use a supported declaration replacement operation.
- Root cause: the type insertion operation ignored overwrite for an existing struct type.

### 2026-09-27: semantic function insertion rejected multiple declarations

- Tool: `semantic_insert_function`
- Target: `internal/astedit/function_test.go`, duplicate insertion tests
- Failure: insertion was rejected because the snippet contained two function declarations; no source change was made.
- Workaround: insert each test function with a separate semantic operation.
- Root cause: this operation accepts exactly one function or method declaration.

### 2026-09-27: semantic_replace_decl rejected struct replacement

- Tool: `semantic_replace_decl`
- Target: `internal/astedit/scaffold.go`, `ScaffoldOptions`
- Failure: replacing the existing struct declaration failed with `requires a type alias`.
- Workaround: use a narrow atomic source edit to remove the accidental duplicate and update the struct.
- Root cause: this operation supports type aliases but not struct type declarations.

### 2026-09-27: semantic_insert_function rejected multiple declarations

- Tool: `semantic_insert_function`
- Target: `internal/astedit/scaffold_test.go`, scaffold header tests
- Failure: the tool rejected two function declarations supplied in one source snippet with `expected single function declaration`.
- Workaround: submit each test function through a separate semantic insertion call.
- Root cause: this operation accepts one function declaration per call.

### 2026-09-27: semantic_insert_function rejected test visibility

- Tool: `semantic_insert_function` with `access_modifier=private`
- Target: `internal/astedit/scaffold_test.go`, exported-style test function
- Failure: validation rejected the `Test...` name because its capitalization implies public visibility.
- Workaround: insert the test function with inferred/public access.
- Root cause: the explicit access modifier conflicted with Go identifier casing.

### 2026-09-27: semantic symbol inspection used an invalid file path

- Tool: `semantic_inspect_symbol`
- Target: `internal/operation/params.go`, `ParseString`
- Failure: the requested source file path did not exist, so symbol inspection could not resolve the declaration.
- Workaround: locate the declaration file with repository file search, then inspect the symbol in its actual file.
- Root cause: the helper implementation lives outside the assumed `params.go` path.

### 2026-09-27: semantic outline selected a nonexistent CLI directory

- Tool: `semantic_outline`
- Target: `cmd/semedit`
- Failure: the repository has no `cmd/semedit` directory; outline returned a path resolution error.
- Workaround: locate the executable entrypoint with repository file search and inspect `main.go`.
- Root cause: assumed a conventional command subdirectory without confirming the repository layout.

### 2026-09-27: semantic symbol inspection used a nonexistent registry path

- Tool: `semantic_inspect_symbol`
- Target: `internal/operation/registry.go`, `ToolDefinitions`
- Failure: the requested file path did not exist, so the symbol could not be inspected.
- Workaround: use confirmed source paths from the repository file list and targeted search.
- Root cause: assumed a registry filename without checking the package layout.

### 2026-09-27: semantic body replacement could not resolve an MCP method

- Tool: `semantic_replace_body`
- Target: `internal/mcp/server.go`, `Server.Initialize`
- Failure: using the unqualified method name `Initialize` returned `symbol not found`; no source change was made.
- Workaround: retry with the receiver-qualified method identifier.
- Root cause: method body replacement requires a resolvable receiver-qualified symbol in this workspace.

### 2026-09-27: semantic symbol inspection used incorrect test names

- Tool: `semantic_inspect_symbol`
- Target: `internal/mcp/server_test.go`, `TestOperationInputSchema`; `internal/operation/wire_backend_test.go`, `TestOperationExampleContracts`
- Failure: neither guessed test name exists, so both symbol inspections returned `symbol not found`.
- Workaround: locate the actual test function names with a targeted source search, then inspect those declarations.
- Root cause: assumed test identifiers before locating the existing tests.

### 2026-09-27: semantic construct replacement could not resolve MCP method

- Tool: `semantic_replace_construct`
- Target: `internal/mcp/server.go`, `Server.toolsList`
- Failure: the receiver-qualified method name was not found; no source change was made.
- Workaround: locate the exact method name and replace the relevant construct using its registered symbol.
- Root cause: assumed a method name from a nearby tool description without confirming its declaration identifier.

### 2026-09-27: semantic lookup used incorrect MCP method name

- Tool: `semantic_lookup`
- Target: `internal/mcp/server.go`, `toolsList`
- Failure: lookup returned `symbol not found`; no source change was made.
- Workaround: locate the actual declaration identifier with a targeted source search.
- Root cause: the method is named `listTools`, not `toolsList`.

### 2026-09-27: semantic construct replacement could not match batch parse preflight

- Tool: `semantic_replace_construct`
- Target: `internal/mcp/batch.go`, `Server.ExecuteBatch`
- Failure: the discriminator `entry.Parse(raw)` did not match a supported construct selector; no source change was made.
- Workaround: inspect the selector diagnostics or use the exact conditional expression and candidate path.
- Root cause: the construct matcher does not match the `if` initializer using the supplied expression.

### ST-0075: benchmark symbol-resolution misses

- Date: 2026-09-27
- Tool(s): `semantic_lookup`, `semantic_inspect_symbol`, `semantic_replace_body`
- Source: `gpt-6-luna-all-r10-c5-prescriptive-20260927-retry`; 21 failed calls across `task-09-composite-refactor` and `task-11-mixed-sink-api-migration`.
- Failure: calls used guessed, outdated, or unsupported symbol spellings and received `symbol not found` (examples include `Load`, `DefaultConfig`, `NormalizeKind`, and `BufferedSink.Write`).
- Workaround: inspect the current declaration outline/source and use exact qualified symbols; some failures followed earlier edits that removed or renamed the target.
- Root cause: model symbol selection drifted from the current file/catalog state; determine whether candidate suggestions and clearer qualified-name contracts reduce repeated failed probes.

### ST-0076: benchmark calls supplied empty required symbols

- Date: 2026-09-27
- Tool(s): `semantic_inspect_symbol`, `semantic_lookup`
- Source: same benchmark; 9 failed calls split across `task-09-composite-refactor` (4) and `task-11-mixed-sink-api-migration` (5).
- Failure: empty `symbol` values were rejected as required parameters.
- Workaround: populate the symbol from the user request or inspect the file outline before calling.
- Root cause: model emitted incomplete calls while exploring an ambiguous target; consider whether the error response should include a compact next step or whether prompt/schema guidance can prevent empty strings.

### ST-0077: benchmark tool calls rejected by automatic safety review

- Date: 2026-09-27
- Tool(s): `semantic_verify` (11), `semantic_rename` (3), `report_feedback` (1)
- Source: same benchmark; 15 failed calls, primarily in `task-11-mixed-sink-api-migration`.
- Failure: host review rejected workspace-wide verification/rename because the task forbade test or protected-file changes. One `report_feedback` call was also rejected for alleged external disclosure.
- Workaround: use narrowly scoped/read-only checks where available; do not retry rejected actions through an indirect path.
- Root cause: the verification/rename outcomes reflect task prohibitions and broad tool side effects, not semedit execution defects. The current feedback handler appears to return a draft without posting or saving it, so the external-disclosure rationale for that one rejection is unsubstantiated and may be a reviewer false positive; retain it as a host outcome, not a semedit defect.

### ST-0078: package diagnostics blocked semantic reference analysis

- Date: 2026-09-27
- Tool(s): `semantic_find_references`
- Source: same benchmark; 4 failed calls in `task-09-composite-refactor`.
- Failure: package loading failed after the agent left an unused import or temporarily removed `DefaultConfig`, so the Go package no longer type-checked.
- Workaround: restore a compilable intermediate package before requesting references.
- Root cause: semantic analysis correctly depends on a valid package snapshot, but the agent invoked it between mutation steps; consider clearer diagnostics that make the transient source problem and recovery action prominent.

### ST-0079: benchmark calls used parameter names absent from tool contracts

- Date: 2026-09-27
- Tool(s): `semantic_insert_function`, `semantic_outline`, `semantic_rename`, `semantic_lookup`
- Source: same benchmark; 4 failed calls across `task-07-generate-template-main` and `task-11-mixed-sink-api-migration`.
- Failure: arguments used `declaration` instead of `source`, `file` instead of `path`, `new_name` instead of `to`, and `path` instead of `file`.
- Workaround: follow the advertised schema keys.
- Root cause: similar operations use inconsistent names and the model supplied familiar aliases unsupported by these tools; assess cross-tool naming consistency and examples in the generated schema.

### ST-0080: benchmark function insertion rejected multiple declarations

- Date: 2026-09-27
- Tool: `semantic_insert_function`
- Source: same benchmark; 3 failed calls in `task-11-mixed-sink-api-migration`.
- Failure: a single-function operation received snippets containing two function declarations and rejected them.
- Workaround: insert each function as a separate call (or use a batch of single-function calls).
- Root cause: the call contract accepts one declaration, while the model grouped related helpers into one snippet; check whether the error and examples make the single-declaration boundary clear.

### ST-0081: benchmark function insertion rejected `package-private`

- Date: 2026-09-27
- Tool: `semantic_insert_function`
- Source: same benchmark; 1 failed call in `task-11-mixed-sink-api-migration`.
- Failure: the Go backend rejected `access_modifier: package-private`; it supports `infer`, `public`, and `private`.
- Workaround: use `private` or omit the access modifier for inference.
- Root cause: the shared access-modifier vocabulary includes values that Go cannot accept; backend-filtered enum guidance should prevent this call.

### ST-0082: benchmark batch requests failed JSON argument decoding

- Date: 2026-09-27
- Tool: `semantic_batch`
- Source: same benchmark; 2 failed calls in tasks 09 and 11.
- Failure: the submitted edit objects flattened operation fields beside `tool` and omitted the required nested `params` object, producing `invalid arguments: unexpected end of JSON input` for `semantic_rename` and `semantic_insert_function`.
- Workaround: encode each edit as `{"tool": ..., "params": {...}}` according to the advertised batch schema.
- Root cause: the model did not follow the nested batch payload contract; check whether the schema/examples make that nesting sufficiently clear and ensure malformed entries identify the missing `params` field.

### ST-0083: benchmark used `switch` where the construct selector requires `case`

- Date: 2026-09-27
- Tool: `semantic_replace_construct`
- Source: same benchmark; 1 failed call in `task-11-mixed-sink-api-migration`.
- Failure: the call supplied `kind: switch` and a complete switch statement whose selector also changed; the operation supports branch constructs, with enum kind `case`, not replacement of a whole switch statement.
- Workaround: identify the intended branch and provide its `case` clause as the replacement source, or use a broader operation when changing the whole switch.
- Root cause: the tool description says “switch branch” but does not make the branch-only source contract explicit; clarify scope and add a branch-shaped example.

### ST-0084: benchmark body replacement received an incomplete Go body

- Date: 2026-09-27
- Tool: `semantic_replace_body`
- Source: same benchmark; 1 failed call in `task-07-generate-template-main`.
- Failure: replacement body ended before closing the final `if`, so parsing failed with `expected '}', found 'EOF'`.
- Workaround: provide the complete body, including all closing braces.
- Root cause: the model truncated a multi-construct body; determine whether the syntax error could identify the incomplete construct more locally.

### ST-0085: scaffold rejected prose purpose header

- Date: 2026-09-27
- Tool: `semantic_scaffold_file`
- Source: implementation of ADR-0054 in `.scratch/worktrees/verify-config/internal/projectverify/types.go`.
- Failure: `purpose_header` was provided as plain prose and the scaffold parser rejected it with `syntax error: invalid purpose header: scaffold.go:1:1: expected 'package', found This`.
- Workaround: retry with a Go comment as the purpose header.
- Root cause: the call did not encode the required Go comment syntax in the header.

### ST-0086: structural type insertion rejected replacement

- Date: 2026-09-27
- Tool: `semantic_insert_structure`
- Source: ADR-0054 `projectverify` public types in `.scratch/worktrees/verify-config/internal/projectverify/types.go`.
- Failure: attempted to update the newly added `Hook` and `HookResult` structs with `overwrite: true`; the tool rejected the request with `overwrite is unsupported for type structure insertion`.
- Workaround: reconstruct the package type file through semantic scaffolding and insert the complete updated declarations.
- Root cause: the structural insertion tool does not support replacing an existing type even though its shared schema exposes the overwrite field.

### ST-0087: verification request struct replacement is unsupported

- Date: 2026-09-27
- Tool: `semantic_replace_decl`
- Target: `internal/backend/backend.go`, `VerifyRequest`, in the verification integration worktree.
- Failure: the declaration replacement rejected a struct because it only accepts type aliases.
- Workaround: use an AST-aware rewrite for the unsupported struct-field change, preserving atomic publication.
- Root cause: the declaration replacement operation does not cover struct definitions.

### ST-0088: Body replacement did not resolve a worktree-prefixed path

- Date: 2026-09-27
- Tool: `semantic_replace_body`
- Target: `internal/backend/golang/backend.go` in the integration worktree.
- Failure: The body lookup failed after symbol inspection had resolved the declaration.
- Workaround: Use a checked AST rewrite for this unsupported worktree path; preserve the isolated worktree.
- Root cause: Inconsistent resolution of a worktree-prefixed relative path between inspection and mutation.

### ST-0089: Automatic import resolution introduced a package cycle

- Date: 2026-09-27
- Tool: `semantic_replace_body`
- Target: `internal/pipeline/pipeline.go`.
- Failure: Replacing CheckDiagnostics added an adapter import, producing an import cycle because the adapter depends on pipeline.
- Workaround: Move shared gopls discovery to godiagnostics and delegate from both callers.
- Root cause: Import organization resolves imports without validating the package dependency graph before writing.

### ST-0090: Test insertion rejected private access modifier

- Date: 2026-09-27
- Tool: `semantic_insert_function`
- Target: `internal/godiagnostics/diagnostics_test.go`.
- Failure: A Test-prefixed function was rejected with access_modifier private.
- Workaround: Retry with public access to match Go identifier casing.
- Root cause: The visibility contract applies to test declarations too.

### ST-0091: Body replacement temporarily mismatched a helper signature

- Date: 2026-09-27
- Tool: `semantic_replace_body`
- Target: `internal/godiagnostics/diagnostics.go`.
- Failure: The new caller expected loader findings before the helper signature was updated; diagnostics reported an assignment mismatch.
- Workaround: Update the helper contract before final verification.
- Root cause: The multi-declaration contract change was split across calls.

### ST-0092: Scaffold purpose header requires comment syntax

- Date: 2026-09-27
- Tool: `semantic_scaffold_file`
- Target: `internal/godiagnostics/diagnostics_internal_test.go`.
- Failure: The plain-text purpose header was rejected.
- Workaround: Retry with a // comment prefix.
- Root cause: The argument description does not make the Go-comment requirement apparent.

### ST-0093: Function insertion rejected multiple declaration kinds

- Date: 2026-09-27
- Tool: `semantic_insert_function`
- Target: `internal/godiagnostics/diagnostics.go`.
- Failure: A snippet with a type and two functions was rejected.
- Workaround: Insert each declaration separately.
- Root cause: The tool accepts exactly one function declaration.

### ST-0094: function insertion access did not match exported name

- Date: 2026-09-27
- Tool: `semantic_insert_function`
- Source: ADR-0054 discovery functions in `.scratch/worktrees/verify-config/internal/projectverify/config.go`.
- Failure: the batch marked all functions private, and insertion rejected exported `Discover` with `visibility mismatch`.
- Workaround: let insertion infer visibility from function casing.
- Root cause: the batch-level edit applied a single explicit access modifier to public and private declarations.

### ST-0095: construct replacement selector was ambiguous

- Date: 2026-09-27
- Tool: `semantic_replace_construct`
- Source: LSP validation in `.scratch/worktrees/verify-config/internal/projectverify/config.go`.
- Failure: replacing `if item.LSP != nil` in `convertHooks` matched both the form counter and the LSP conversion branch, so the tool rejected the edit and returned both candidate paths.
- Workaround: select the reported construct path for the conversion branch.
- Root cause: repeated equivalent conditions require `construct_path` to disambiguate.

### ST-0096: structural function insertion rejected overwrite

- Date: 2026-09-27
- Tool: `semantic_insert_structure`
- Source: context-aware staging helper in `.scratch/worktrees/verify-config/internal/projectverify/executor.go`.
- Failure: attempted to update the existing `copyWorkspace` signature with `overwrite: true`; the tool rejected it with `overwrite is unsupported for function structure insertion`.
- Workaround: retain the existing helper as a background-context wrapper and add a context-aware helper under a distinct symbol for runtime use.
- Root cause: structural insertion does not support replacing existing functions despite the shared overwrite parameter.

### ST-0097: construct replacement included two statements

- Date: 2026-09-27
- Tool: `semantic_replace_construct`
- Source: hook-status handling in `.scratch/worktrees/verify-config/internal/projectverify/executor.go`.
- Failure: the replacement payload contained the original `if` plus a following `if`, and the tool rejected it because a construct replacement must contain exactly one complete construct.
- Workaround: make the status change within an existing construct or use a bounded full function-body replacement.
- Root cause: the edit payload crossed the selected AST construct boundary.

### ST-0098: semantic_replace_body rejected invalid hook invocation payload

- Date: 2026-09-27
- Tool: `semantic_replace_body`
- Target: `internal/projectverify/executor.go`, `runCheck`
- Observed failure: the replacement body omitted a closing parenthesis in the final `runCommand` call, so the tool rejected the candidate Go syntax before writing.
- Workaround: corrected the payload syntax and retry with the semantic tool.
- Root cause: malformed agent-generated replacement body.

### ST-0099: semantic_replace_construct rejected multi-if payload

- Date: 2026-09-27
- Tool: `semantic_replace_construct`
- Target: `internal/projectverify/executor.go`, `RunPhase`
- Observed failure: the request tried to replace an existing single `if` with two consecutive `if` constructs; the tool requires exactly one construct and rejected it without writing.
- Workaround: use a single replacement branch or restructure with supported single-construct edits.
- Root cause: payload shape did not match the semantic construct operation contract.

### ST-0100: semantic_replace_construct selector used full if statement instead of selector

- Date: 2026-09-27
- Tool: `semantic_replace_construct`
- Target: `internal/projectverify/executor.go`, `runNormalization`
- Observed failure: the selector passed the entire short `if` statement instead of its short initializer or condition; the tool returned available construct paths and made no change.
- Workaround: retry using the listed construct path for the intended command execution branch.
- Root cause: misunderstanding of the operation discriminator field.

### ST-0101: semantic_replace_construct rejected environment setup sequence

- Date: 2026-09-27
- Tool: `semantic_replace_construct`
- Target: `internal/projectverify/executor.go`, `runNormalization`
- Observed failure: the replacement contained an environment setup statement followed by an `if`, while the operation accepts a single complete `if` construct. No change was made.
- Workaround: wrap setup and command execution in one if initializer expression.
- Root cause: replacement payload shape exceeded the tool operation contract.

### ST-0102: semantic_insert_function rejected test access classification

- Date: 2026-09-27
- Tool: `semantic_insert_function`
- Target: `internal/projectverify/config_test.go`
- Observed failure: the test function name begins with `Test` and is exported by Go casing, but the request explicitly selected private access. The tool rejected it without writing.
- Workaround: retry with inferred access.
- Root cause: test function naming convention conflicts with explicit private classification.

### ST-0103: semantic_insert_function rejected two-function payload

- Date: 2026-09-27
- Tool: `semantic_insert_function`
- Target: `internal/projectverify/executor_test.go`
- Observed failure: the request contained two function declarations; the insertion operation accepts exactly one declaration and made no change.
- Workaround: insert each test function separately.
- Root cause: bundled independent tests in one semantic mutation.

### ST-0104: semantic_replace_body rejected command helper payload

- Date: 2026-09-27
- Tool: `semantic_replace_body`
- Target: `internal/projectverify/executor.go`, `runHookCommand`
- Observed failure: the final `runCommandWithEnv` call was missing its closing parenthesis, and the tool rejected the candidate syntax without writing.
- Workaround: correct the call syntax and retry.
- Root cause: malformed generated body.

### ST-0105: body replacement needs a qualified method name after lookup

- Date: 2026-09-27
- Tool: `semantic_replace_body`
- Target: `internal/backend/golang/backend.go`, `GoBackend.Verify`.
- Failure: `symbol: Verify` returned symbol not found although lookup and inspection accepted it and returned this method.
- Workaround: retry with the receiver-qualified symbol from lookup.
- Root cause: method selector resolution differs between inspection and body replacement.

### ST-0106: body replacement left a new standard-library reference unresolved

- Date: 2026-09-27
- Tool: `semantic_replace_body`
- Target: `internal/mcp/batch.go`, `Server.ExecuteBatch`.
- Failure: the replacement succeeded with a new `reflect.DeepEqual` reference but no reflect import; make check caught the undefined identifier.
- Workaround: run semantic import organization before compiling again.
- Root cause: body replacement did not infer the new import in this call.

### ST-0107: semantic_replace_construct could not resolve ExecuteBatch selector

- Tool: `semantic_replace_construct`
- Target: `internal/mcp/batch.go`, `(*Server).ExecuteBatch`
- Observed failure: the request selected `ExecuteBatch` without its `Server` receiver qualifier, so the semantic backend did not find the function and made no change.
- Workaround: retry using the receiver-qualified function name.
- Root cause: method selector requires its full receiver-qualified name.

### ST-0108: semantic batch rejected plan-rendering body string

- Date: 2026-09-27
- Tool: `semantic_batch` with `semantic_replace_body`
- Target: `tools/benchmark-harness/planner.go`, `RenderBenchmarkPlan`
- Failure: the replacement payload contained unescaped newline characters inside Go string literals, so the tool rejected the body before writing it. Earlier independent edits in the batch had already succeeded.
- Workaround: log the failure and retry the remaining body with correctly escaped Go string literals.
- Root cause: newline escaping was lost while constructing the nested tool payload.

### ST-0109: semantic_insert_function rejected multiline test strings

- Date: 2026-09-27
- Tool: `semantic_insert_function`
- Target: `tools/benchmark-harness/cli_integration_test.go`
- Failure: the inserted test payload had an unescaped newline inside a Go string literal, so syntax validation rejected it without writing.
- Workaround: preserve backslashes in nested source text with a raw string payload and retry.
- Root cause: source escaping was lost in the outer JavaScript template literal.

### ST-0110: semantic_insert_structure does not replace an existing type

- Date: 2026-09-27
- Tool: `semantic_insert_structure`
- Target: `cmd/docgen/benchmark_compare.go`, `benchmarkMetricSummary`
- Observed failure: requesting `overwrite: true` returned `overwrite is unsupported for type structure insertion`; no source change was made.
- Workaround: use a structural AST edit supported by the available tools, or make a narrowly scoped atomic declaration edit if no semantic replacement operation exists.
- Root cause: this operation only inserts type structures; its schema rejects replacement despite exposing an overwrite field.

### ST-0111: semantic_rename blocked by a temporarily incomplete aggregate type edit

- Date: 2026-09-27
- Tool: `semantic_rename`
- Target: `cmd/docgen/benchmark_compare.go`, `benchmarkMetricSummary.average`
- Observed failure: renaming `average` to `median` was rejected because the preceding type edit had removed fields still referenced by the average method. No rename was applied.
- Workaround: finish the dependent method updates so the package parses, then retry the semantic rename.
- Root cause: the rename backend requires a clean package and the type and method updates were applied in separate operations.

### ST-0112: semantic_replace_body emitted malformed nested browser replacement code

- Date: 2026-09-27
- Tool: `semantic_replace_body`
- Target: `cmd/docgen/benchmark_aggregate.go`, `scopedBenchmarkBrowserShortcode`
- Observed failure: replacement text containing JavaScript template newlines broke the enclosing Go raw string and produced a compile diagnostic after the edit had been applied.
- Workaround: replace the function body with escaped Go string literals and re-run semantic verification.
- Root cause: nested Go and JavaScript template quoting was not preserved by the submitted payload.

### ST-0113: semantic_lookup did not resolve Go test functions

- Date: 2026-09-28
- Tool: `semantic_lookup`
- Target: Go test functions in `main_test.go`, `internal/backend/rust_test.go`, `internal/pipeline/pipeline_test.go`, `internal/backend/backend_test.go`, `internal/projectverify/executor_test.go`, and `cmd/docgen/benchmarks_test.go`
- Observed failure: lookup returned `symbol not found` for each named test function; no source was changed.
- Workaround: use `rg` to locate test declarations and apply narrowly scoped atomic renames that change only the test identifiers.
- Root cause: the semantic lookup backend does not index Go test functions in this workspace.

### ST-0114: semantic_inspect_symbol requested from the wrong file

- Date: 2026-09-28
- Tool: `semantic_inspect_symbol`
- Target: `renderBenchmarkAggregatesDoc` in `cmd/docgen/benchmark_aggregate.go`
- Observed failure: the request returned no structured symbol result; no source was changed.
- Workaround: locate the declaration first, then inspect it in its actual file.
- Root cause: the caller supplied an incorrect file path for the symbol.

### ST-0115: semantic_replace_body organized the wrong YAML import

- Date: 2026-09-29
- Tool: `semantic_replace_body` with automatic import organization
- Target: `tools/benchmark-harness/oracle.go`, `parseYAMLFrontmatter`
- Observed failure: the body edit succeeded, but import cleanup selected `gopkg.in/yaml.v2`; this repository requires `gopkg.in/yaml.v3`, so diagnostics reported an unavailable module and undefined `yaml`.
- Workaround: replace the import with the existing `gopkg.in/yaml.v3` module and run the repository checks.
- Root cause: automatic import resolution selected a similarly named module without honoring the existing `go.mod` dependency.

### ST-0116: semantic_insert_function rejected an exported benchmark declaration

- Date: 2026-09-29
- Tool: `semantic_insert_function`
- Target: `internal/mcp/startup_bench_test.go`
- Observed failure: the call rejected an exported benchmark function with an explicitly private access modifier.
- Workaround: retry with inferred access.
- Root cause: the requested modifier conflicted with Go identifier casing.

### ST-0117: semantic_verify reached workspace-wide hooks

- Date: 2026-09-29
- Tool: `semantic_verify`
- Target: `internal/mcp/startup_bench_test.go`
- Observed failure: verification surfaced an unrelated harness G204 diagnostic and refused a workspace-wide lint hook without workspace trust.
- Workaround: fix the harness diagnostic and run the repository's `make check` gate.
- Root cause: the selected-file operation also invokes workspace-wide verification hooks.

### ST-0118: semantic_lookup rejected an unsupported parameter

- Date: 2026-09-29
- Tool: `semantic_lookup`
- Target: `TestExecuteAgentUsesOneProcessSessionAndExcludesReflectionMetrics` in `tools/benchmark-harness/session_test.go`
- Observed failure: the call used `query` instead of the supported `symbol` parameter, so validation failed before source inspection; no source changed.
- Workaround: retry using `symbol` and `file`.
- Root cause: incorrect caller parameter shape.

### ST-0119: semantic_lookup did not resolve a Go oracle test

- Date: 2026-09-29
- Tool: `semantic_lookup`
- Target: `TestResponseAndRequestIDsStayDistinct` in `testdata/bench-oracles/task-12-request-identity-migration/response/response_hidden_test.go`
- Observed failure: lookup returned `symbol not found`; no source was changed.
- Workaround: use `rg` to locate the Go test and make narrowly scoped atomic edits to hidden oracle test bodies.
- Root cause: the semantic lookup backend does not index Go test functions in the hidden oracle tree.

### ST-0120: semantic_insert_structure rejected type overwrite

- Date: 2026-09-29
- Tool: `semantic_insert_structure`
- Target: `Server` in `internal/mcp/server.go`
- Observed failure: two attempts to update an existing struct with `kind: type` and `overwrite: true` returned `overwrite is unsupported for type structure insertion`; no source changed.
- Workaround: use an atomic source edit for the struct fields, then continue using semantic tools for function bodies and declarations.
- Root cause: the tool advertises a generic overwrite option but does not support replacing type declarations.

### ST-0121: semantic_scaffold_file rejected a plain purpose header

- Date: 2026-09-29
- Tool: `semantic_scaffold_file`
- Target: `tools/benchmark-harness/rename_five_fixture_test.go`
- Observed failure: a plain-English `purpose_header` produced `expected package, found This`; no source was created.
- Workaround: retry with a Go comment-prefixed purpose header.
- Root cause: the purpose header is parsed as Go source and must include Go comment syntax, which the parameter description did not state.

### ST-0122: semantic_replace_body omitted a required import

- Date: 2026-09-29
- Tool: `semantic_replace_body`
- Target: `TestFiveLocationRenameFixtureRequiresAllFiveRenames` in `tools/benchmark-harness/rename_five_fixture_test.go`
- Observed failure: inserting `exec.CommandContext` succeeded without adding `os/exec`; `make check` reported `undefined: exec`.
- Workaround: run semantic import organization or add the missing import, then rerun repository checks.
- Root cause: automatic import organization did not resolve the new package-qualified reference.

### ST-0123: semantic_verify rejected check-only verification without workspace trust

- Date: 2026-09-29
- Tool: `semantic_verify`
- Target: `internal/operation`
- Observed failure: the check-only request returned an error because the verification hook `golangci-lint` requires workspace trust; its response also contained stale unrelated gopls diagnostics.
- Workaround: use repository-local `make check`, which completed formatting, lint, tests, and documentation verification.
- Root cause: the verification tool applied a trust gate before hooks and diagnostics that need full workspace context.

### ST-0124: semantic_scaffold_file inserted a blank line after the package comment

- Date: 2026-09-29
- Tool: `semantic_scaffold_file`
- Target: `internal/astedit/remove.go`
- Observed failure: the scaffold created a blank line between the package comment and package declaration, and `make check` reported a detached package comment.
- Workaround: remove the blank line and rerun repository checks.
- Root cause: the scaffold formatter preserved spacing that violates the repository's package-comments lint rule.

### ST-0125: semantic_insert_function rejected multiple declarations

- Date: 2026-09-29
- Tool: `semantic_insert_function`
- Target: `internal/operation/wire_remove.go`
- Observed failure: the request contained several constants, variables, and functions; the tool rejected it because it accepts one function declaration only. No source changed.
- Workaround: insert each declaration with its matching semantic insertion operation.
- Root cause: the tool validates a single function snippet and does not accept a file-sized declaration batch.

### ST-0126: semantic inspection tools targeted the parent checkout

- Date: 2026-09-29
- Tool: `semantic_outline`, `semantic_lookup`
- Target: `internal/astedit/move_construct.go` and declarations in `internal/operation` and `internal/backend/golang` in the assigned `.scratch/worktrees/refactor-move` worktree
- Observed failure: semantic outline/lookup requests resolved against the parent checkout instead of this assigned worktree; the move file was reported missing, and a lookup for `ReplaceConstruct` returned not found despite the declaration being in a different file.
- Workaround: use local repository inspection and edits confined to the assigned branch.
- Root cause: the MCP semantic service is bound to the parent checkout and exposes no worktree selector.

### ST-0127: semantic_insert_function inferred an unavailable import

- Date: 2026-09-29
- Tool: `semantic_insert_function` via local `semedit insert-func`
- Target: `mergeMoveImports` in `internal/astedit/move_construct.go`
- Observed failure: the insertion reported success but added `golang.org/x/tools/go/ast/astutil`, which the current module does not provide; the resulting package failed to import.
- Workaround: replace the helper with standard-library AST import construction and remove the invalid dependency.
- Root cause: automatic import organization treated a package path as available without confirming it exists in the configured module dependency graph.

### ST-0128: semantic_scaffold_file rejected plain purpose text

- Date: 2026-09-29
- Tool: `semantic_scaffold_file`
- Target: `internal/astedit/extract_variable.go`
- Observed failure: the request supplied a plain-English `purpose_header`; the tool parsed it as Go source and returned `expected 'package', found Provides`. No file was created.
- Workaround: retry with a Go comment-prefixed purpose header, matching the established tool behavior.
- Root cause: `purpose_header` is interpreted as a Go comment/source prefix, but its parameter description does not make the required `//` explicit.

### ST-0129: semantic tools and edits resolved against the parent checkout

- Date: 2026-09-29
- Tool: `semantic_outline`, `semantic_lookup`, `semantic_scaffold_file`, and worktree-edit shell operations
- Target: `internal/astedit/extract_variable.go` plus README, ADR index, MCP schema, operation wire, capability, and skill files that were meant for the assigned `refactor-extract-variable` worktree
- Observed failure: semantic inspection resolved files in the parent checkout, and an attempted scaffold request failed before creating its target. Subsequent changes intended for this branch appeared in the parent checkout. The parent preserved those changes in `.scratch/extract-variable-parent-snapshot/` and restored its integration checkout; no parent changes remain.
- Workaround: copy the preserved patch and untracked files into the assigned worktree, then continue using only absolute worktree paths and the worktree-local CLI or atomic file updates.
- Root cause: the shared semantic MCP service is bound to the parent checkout; a shell edit command also ran without an explicit worktree directory, allowing checkout context to drift.

### ST-0130: semantic_batch rejected construct movement

- Date: 2026-09-29
- Tool: `semantic_batch` with `semantic_move_construct`
- Target: same-package declaration moves across Go files
- Observed failure: the reported batch request returns `tool is not batchable: semantic_move_construct` during batch preflight, before edits execute. The registry definition sets `Batchable: false`; `ExecuteBatch` rejects non-batchable entries. A local CLI replay was unavailable because the first required `make check` could not resolve uncached modules with `GOPROXY=off`.
- Workaround: mark the operation batchable only after implementing deferred multi-file outcomes, destination receipts, and batch write-set handling; then add public MCP and CLI regression coverage.
- Root cause: batch eligibility and post-processing track only one `WrittenFile`, while moves persist source and destination files.

### ST-0131: batched move verification changed an unrelated source header

- Date: 2026-09-29
- Tool: `semantic_batch` with same-package `semantic_move_construct` operations
- Target: isolated Chi router acceptance fixture, `path_value.go`
- Observed failure: the 19-operation batch returned success and preserved the original test files, but the independent `TestSemeditRoutingDecomposition` oracle found that global final verification removed one blank line between the build header and package clause in untouched `path_value.go`. The acceptance fixture is isolated outside this worktree.
- Workaround: derive built-in Go formatting and import-organizing paths from actual changed files after all batch edits; keep workspace diagnostics and configured normalization hooks on their existing scope. Regression coverage compares the untouched header byte-for-byte.
- Root cause: final `semantic_verify` formatted the entire workspace, including unchanged Go files.

### ST-0132: semantic_insert_structure cannot update an existing struct

- Date: 2026-09-30
- Tool: `semantic_insert_structure` with `kind: type` and `overwrite: true`
- Target: `internal/mcp/server.go`, `Server`; `internal/godiagnostics/manager.go`, `managedSession` and `Document`
- Observed failure: the tool rejected the request with `overwrite is unsupported for type structure insertion`; it made no change.
- Workaround: add session-manager and executable-identity fields with narrowly scoped atomic source updates, then continue using semantic tools for function bodies.
- Root cause: the type insertion tool cannot overwrite an existing struct, and no exposed semantic operation inserts a struct field.

### ST-0133: scaffold header and declaration comments need text repair

- Date: 2026-09-30
- Tool: `semantic_scaffold_file`, `semantic_insert_type`, and `semantic_insert_function`
- Target: `internal/godiagnostics/manager.go`, `internal/atomicfile/write.go`, `internal/atomicfile/write_test.go`, `internal/adapters/golang/rename_edit.go`, and the cancellation contract on `internal/mcp/server.go`, `Serve`
- Observed failure: the scaffold separated its package comment from the package clause with a blank line, triggering `revive`. Inserted exported declarations require Go documentation comments, but no exposed semantic operation adds a comment to an existing declaration.
- Workaround: make narrow atomic comment edits after semantic insertion, including the exported cancellation/reader ownership contract, then run formatting and lint checks.
- Root cause: the scaffold's package-header layout violates the repository linter, and the semantic editing API has no add-comment operation for existing declarations.

### ST-0134: semantic_insert_function rejected malformed test snippet

- Date: 2026-09-30
- Tool: `semantic_insert_function`
- Target: `internal/godiagnostics/diagnostics_test.go`, `TestManagedDiagnosticsReuseAndRefresh`
- Observed failure: the generated test source contained a malformed shell-quote rune literal; the tool reported `illegal rune literal` and left the file unchanged.
- Workaround: simplify the wrapper script to receive paths through environment variables and retry semantic insertion.
- Root cause: the caller escaped a nested Go and shell string incorrectly; the tool correctly rejected invalid Go syntax.

### ST-0135: inspection used an unverified test symbol

- Date: 2026-09-30
- Tool: `semantic_inspect_symbol`
- Target: `internal/mcp/server_test.go`, `TestMCPMetricsFilePersistsAllCalls`
- Observed failure: symbol not found; no files changed.
- Workaround: locate the actual declaration name before inspection.
- Root cause: the caller guessed a test name rather than using lookup.

### ST-0136: generated PID wrapper used a literal dollar sign

- Date: 2026-09-30
- Tool: `semantic_replace_body`
- Target: `internal/mcp/server_test.go`, `newManagedMCPFixture`
- Observed failure: the cancellation test's shell wrapper recorded `$` instead of its process ID; the resulting Go source parsed, but the process cleanup assertion could not read a PID.
- Workaround: correct the wrapper through semantic body replacement using a replacement callback.
- Root cause: JavaScript string replacement interpreted `$$` as a replacement escape before passing the source to the tool.

### ST-0137: failed test-file calls hid their error detail

- Date: 2026-09-30
- Tool: `semantic_scaffold_file` and `semantic_insert_function`
- Target: `internal/gocache/cache_test.go`, `TestGoSubprocessTemporaryFiles`
- Observed failure: both calls returned `isError: true`; the caller initially hid the error detail. A retained retry reported `invalid purpose header: expected package, found Temporary`; no test file existed.
- Workaround: retain complete tool errors, provide the purpose header as a Go comment with `//`, then insert the test.
- Root cause: `purpose_header` expects Go comment source rather than plain text. The caller supplied plain text and discarded the initial diagnostics.

### ST-0138: batch verification invokes untrusted workspace hooks

- Date: 2026-09-30
- Tool: `semantic_batch`
- Target: `internal/godiagnostics/manager.go`
- Observed failure: all edits applied, but final verification failed because the configured golangci-lint hook requires workspace trust, which this tool's schema cannot supply. Batch normalization also formatted unrelated fixtures. A deferred insertion using json.RawMessage lacked its import.
- Workaround: retain applied edits, organize their imports explicitly, restore unrelated formatting atomically, and use `make check` for the configured verification gate.
- Root cause: batch verification and request trust contracts disagree; default batch normalization formats the entire workspace and does not resolve newly introduced imports.

### ST-0139: large migration batch leaves compile integration errors in final verification

- Date: 2026-09-30
- Tool: `semantic_batch` through the semedit MCP CLI
- Target: source-buffer integration across AST operations, the registry, and shared gopls
- Observed failure: the initial migration batch, integration/test batch, and reviewed source-view repair batch applied their mutations, but final verification reported diagnostics and the configured golangci-lint hook failed.
- Workaround: retain the applied edits, run the repository mechanical fixes and compile gate, then repair the concrete integration errors with semantic tools.
- Root cause: the prepared migration needs its compile integration pass; the tool correctly preserved applied edits with diagnostics.

### ST-0140: declaration replacement and scaffolding need comment repair

- Date: 2026-09-30
- Tool: `semantic_remove_construct`, `semantic_insert_type`, and `semantic_scaffold_file`
- Target: `internal/sourcebuf/buffer.go`, `internal/atomicfile/staged.go`, `internal/mcp/buffers.go`, and replaced registry/session/resolver types
- Observed failure: generated package headers have a separating blank line; removing and reinserting types drops exported declaration comments, including an adjacent resolver declaration comment. The formatter does not repair the resulting revive failures.
- Workaround: repair declaration and package comments with a scoped atomic source rewrite; retain semantic tools for function bodies and declarations.
- Root cause: the available tools have no declaration-comment update operation, and scaffold/comment association does not preserve the lint contract.

### ST-0141: buffered consistency repair batch failed final verification

- Date: 2026-09-30
- Tool: `semantic_batch` through the semedit MCP CLI
- Target: source buffer, rename, batch journal, and protocol test helper
- Observed failure: the repair batch applied its edits but returned an error at final verification.
- Workaround: retain the applied edits, inspect the saved response, and run the configured mechanical and compile checks.
- Root cause: the final verification gate checks the complete integration while declaration replacement can require comment repair.

### ST-0142: move overlay migration failed its final integration gate

- Date: 2026-09-30
- Tool: `semantic_batch` through the semedit MCP CLI
- Target: cross-file and cross-package move planner, registry, and package loader
- Observed failure: the prepared move migration, following session/rename repair batch, and same-package scope repair batch applied their edits but final verification returned an error.
- Workaround: retain applied edits, inspect the saved verification result, and run the configured mechanical checks before repairing integration failures.
- Root cause: the complete verification gate also checks callers affected by private signature changes and import cleanup.

### ST-0143: import organization moved a function suppression into imports

- Date: 2026-09-30
- Tool: `semantic_organize_imports`
- Target: `internal/astedit/move_cross_package.go`
- Observed failure: the move planner's existing function comment and `nolint:funlen,gocognit` directive moved into the import block after helper insertion and import organization.
- Workaround: restore the comment and directive at the planner declaration with an atomic comment-only rewrite.
- Root cause: declaration and import insertion reused stale comment positions rather than retaining their declaration association.

### ST-0144: body replacement used an unsupported parameter

- Date: 2026-09-30
- Tool: `semantic_replace_body`
- Target: `internal/astedit/move_construct.go`, `cleanupMoveImports`
- Observed failure: the call rejected `new_body` before mutation.
- Workaround: supply the operation's documented `body` parameter.
- Root cause: the caller used an incorrect parameter name.

### ST-0145: inspection used a guessed helper name

- Date: 2026-10-01
- Tool: `semantic_inspect_symbol`
- Target: worker `internal/contractplan/plan.go`, `renderSignature`
- Observed failure: inspection returned symbol not found; no mutation occurred.
- Workaround: inspect the declaration outline first and select `typeRenderer.signature`.
- Root cause: the caller guessed a helper name instead of obtaining its identity from the semantic outline.

### ST-0150: inspection selected the wrong declaration file

- Date: 2026-10-01
- Tool: `semantic_inspect_symbol`
- Target: `internal/operation/wire_engine_mutations.go`, `FileEditRes`
- Observed failure: inspection returned symbol not found; no mutation occurred.
- Workaround: obtain the declaration file through `semantic_lookup` before inspection.
- Root cause: the caller guessed a declaration's file instead of resolving its semantic location.

### ST-0151: verifier executable built with an older Go type checker

- Date: 2026-10-01
- Tool: semantic verification in development CLI MCP batches
- Target: `internal/astedit/inspect.go` and other embedded struct literals
- Observed failure: Verification rejected promoted literal fields accepted by Go 1.27 builds after `go fix` modernized them.
- Root cause: The discovered `gopls` v0.23.0 executable was built with Go 1.26.6 and used that version of `go/types`.
- Workaround: Rebuilt the same cached v0.23.0 source with Go 1.27.1 into project-local `.scratch/go/bin/gopls`, then supplied that executable to all six workers. Check-only verification passed with no promoted-field errors.

### ST-0152: guessed outline file and unchecked error decoding

- Date: 2026-10-02
- Tool: `semantic_outline`
- Target: `internal/operation/wire_engine_remove.go`
- Observed failure: The selected file preparation failed; the coordinator then attempted to decode the error text as JSON and raised a decoding error. No mutation occurred.
- Root cause: The coordinator guessed the declaration file and failed to check the tool error flag before decoding its response.
- Workaround: Discover the file with `rg --files` or semantic lookup, check `isError`, and decode only a successful structured response.

### ST-0146: semantic_insert_function rejected multi-function snippet

- Date: 2026-10-01
- Tool: `semantic_insert_function`
- Target: `internal/operation/value_schema.go`
- Observed failure: the request contained ten function declarations; the tool rejected it because it accepts one function declaration per call. No source changed.
- Workaround: insert each helper function with a separate semantic insertion call.
- Root cause: the semantic insertion operation validates exactly one function declaration and does not support declaration batches.

### ST-0147: semantic_insert_function rejected private helper visibility

- Date: 2026-10-01
- Tool: `semantic_insert_function`
- Target: `internal/operation/value_schema.go`
- Observed failure: the request explicitly selected public access for lowercase helper `validateBounds`; the tool rejected the visibility mismatch. No source changed from that call.
- Workaround: insert lowercase helpers with private access.
- Root cause: Go access mode must match identifier casing when explicitly supplied.

### ST-0148: exported planner declarations needed documentation comments

- Date: 2026-10-01
- Tool: `semantic_insert_type`, `semantic_insert_function`
- Target: `internal/operation/value_schema.go` and `internal/operation/typed_descriptors.go`
- Observed failure: repository `make check` rejected the successful insertions because exported Go declarations lacked required documentation comments. No incorrect AST was reported.
- Workaround: add Go documentation comments with an atomic source update because no registered semantic operation adds comments to existing declarations.
- Root cause: the semantic insertion operations accept declarations but expose no doc-comment edit operation.

### ST-0149: semedit-next did not expose the outline command

- Date: 2026-10-01
- Tool: `semedit-next outline`
- Target: navigation of `internal/operation` in the assigned worktree
- Observed failure: the checked CLI returned `unknown command "outline"`; no workspace files changed.
- Workaround: use the CLI's supported lookup/inspect commands for declaration navigation.
- Root cause: this CLI build does not register an outline command.

### ST-0150: semedit-next did not expose insert-structure

- Date: 2026-10-01
- Tool: `semedit-next insert-structure`
- Target: adding the schema field to `ParameterContract`
- Observed failure: the checked CLI returned `unknown command "insert-structure"`; no workspace files changed.
- Workaround: check the generalized `insert` command for a supported struct-member operation before choosing another edit path.
- Root cause: this CLI build does not register the structured-field insertion command.

### ST-0151: semedit-next replace-body rejected a declaration wrapper

- Date: 2026-10-01
- Tool: `semedit-next replace-body`
- Target: `ParameterSpecSchema` in `internal/operation/typed_descriptors.go`
- Observed failure: the request passed a complete `func` declaration in `--body`; the CLI rejected the nested declaration before writing.
- Workaround: pass only the function body statements.
- Root cause: `replace-body` accepts statements without a function signature or braces.

### ST-0152: existing struct field insertion has no registered CLI operation

- Date: 2026-10-01
- Tool: `semedit-next insert`
- Target: schema, role, and output-contract fields in `internal/operation/operation.go`
- Observed failure: the checked CLI only inserts top-level declarations and has no struct-member insertion operation; no source changed.
- Workaround: add the exact fields with an atomic source update after checking the CLI help.
- Root cause: this CLI build does not register a field-level Go AST mutation.

### ST-0153: atomic operation struct update did not match declaration layout

- Date: 2026-10-01
- Tool: atomic Python source update
- Target: `internal/operation/operation.go`
- Observed failure: the script expected an incorrect field-block string and aborted before mutation.
- Workaround: inspect the exact struct declarations and update their matching field blocks.
- Root cause: the assumed gofmt alignment differed from the source layout.

### ST-0154: make check exposed missing Go documentation comments

- Date: 2026-10-01
- Tool: `make check` (revive)
- Target: `internal/operation/typed_descriptors.go` and `internal/operation/value_schema.go`
- Observed failure: the linter rejected exported declarations inserted without documentation comments.
- Workaround: add declaration comments atomically; no semantic operation exposes comment-only edits.
- Root cause: inserted declarations did not include the doc comments required by repository lint.

### ST-0155: atomic comment repair script had invalid Python syntax

- Date: 2026-10-01
- Tool: atomic Python source update
- Target: `internal/operation/typed_descriptors.go`, `internal/operation/value_schema.go`, and `internal/operation/operation.go`
- Observed failure: Python rejected the script before execution; no files changed.
- Workaround: simplify the script and apply each atomic file update.
- Root cause: an extra closing parenthesis in the log append expression.

### ST-0156: semedit-next scaffold command name differed from help

- Date: 2026-10-01
- Tool: `semedit-next scaffold --help`
- Target: new `internal/operation/schema_contract.go`
- Observed failure: the CLI reported unknown command `scaffold` and suggested `scaffold-file`; no files changed.
- Workaround: use the registered `scaffold-file` command.
- Root cause: the command is named `scaffold-file`, not `scaffold`.

### ST-0157: semedit-next insert-function command name differed from help

- Date: 2026-10-01
- Tool: `semedit-next insert-function --help`
- Target: `internal/operation/schema_contract.go`
- Observed failure: the CLI reported unknown command `insert-function`; the new scaffold file was already created successfully.
- Workaround: use the registered `insert-func` command.
- Root cause: this CLI build uses the abbreviated command name.

### ST-0158: semantic CLI has no anonymous closure edit for registry parse wrapping

- Date: 2026-10-01
- Tool: `semedit-next` operation inventory
- Target: `internal/operation/operation.go`
- Observed failure: no registered semantic operation can wrap the anonymous `Entry.Parse` and `Entry.Invoke` parse closures with shared contract validation.
- Workaround: update only those closure bodies with atomic source edits after locating them through semantic lookup.
- Root cause: the CLI exposes function-body replacement but no closure-body editing operation.

### ST-0159: registry edit script had a Python indentation error

- Date: 2026-10-01
- Tool: atomic Python source update
- Target: `internal/operation/operation.go` and `internal/operation/schema_contract.go`
- Observed failure: Python rejected a function body because its return statement was outdented; no files changed.
- Workaround: reformat the script and retry the atomic updates.
- Root cause: indentation was lost while composing a nested replacement helper.

### ST-0160: semantic insert-decl rejected const spec without declaration wrapper

- Date: 2026-10-01
- Tool: `semedit-next insert-decl`
- Target: `internal/operation/operation.go`
- Observed failure: the source contained a constant specification rather than a complete `const` declaration; validation failed before mutation.
- Workaround: submit each constant inside its own `const (...)` declaration.
- Root cause: insert-decl parses declaration nodes, not individual specs.

### ST-0161: semedit-next import organization command name differed from remembered tool name

- Date: 2026-10-01
- Tool: `semedit-next organize-imports`
- Target: `internal/operation/schema_contract.go`
- Observed failure: the CLI reported unknown command `organize-imports`; no source changed.
- Workaround: inspect CLI help for the registered import organization command and use it.
- Root cause: the checked build's command spelling differs from the MCP operation name.

### ST-0162: semantic test insertion used private access for exported test function

- Date: 2026-10-01
- Tool: `semedit-next insert-func`
- Target: `internal/mcp/schema_test.go`
- Observed failure: the tool rejected private access for a `Test...` function; no test source changed.
- Workaround: insert the test with public access matching its exported test name.
- Root cause: Go test entrypoints use exported identifiers.

### ST-0163: scaffold-file does not create the mandatory Go file purpose header

- Date: 2026-10-01
- Tool: `semedit-next scaffold-file`
- Target: new `internal/cli/typed_params_test.go`
- Observed failure: the successful scaffold creates only a package declaration; it cannot add the required WHY comment.
- Workaround: add the file purpose comment atomically before inserting declarations.
- Root cause: scaffold-file infers the package but provides no file-header parameter.

### ST-0164: MCP test harness lookup used a mistyped worktree path

- Date: 2026-10-01
- Tool: `rg` via shell
- Target: MCP server constructors
- Observed failure: the workdir did not exist because the command path contained an incorrect UUID fragment; no files changed.
- Workaround: rerun the search from the current worktree directory.
- Root cause: a copied path string was transcribed incorrectly.

### ST-0165: check-log wrapper used zsh's read-only `status` variable

- Date: 2026-10-01
- Tool: shell wrapper around `make check`
- Target: `.scratch/final-make-check.log`
- Observed failure: after `make check` completed, zsh rejected assignment to its read-only `status` variable; the wrapper returned an error before filtering the completed log.
- Workaround: use a task-specific variable name or run `make check` directly with output redirected.
- Root cause: `status` is a reserved read-only parameter in zsh.

### ST-0166: polling a completed make check session returned an unknown process id

- Date: 2026-10-01
- Tool: `write_stdin`
- Target: completion check session
- Observed failure: the shell session had already exited, so the poll returned unknown process id; no workspace files changed.
- Workaround: verify completion from the direct command log and use the command's initial exit result.
- Root cause: a poll arrived after the execution session had been reaped.

### ST-0167: Make-variable cache settings did not reach the linter process

- Date: 2026-10-01
- Tool: `make check`
- Target: full-worktree Go lint
- Observed failure: passing `GOMODCACHE` and `GOCACHE` as Make command-line variables left `golangci-lint` unable to load `pgregory.net/rapid` with `GOPROXY=off`; lint stopped before tests.
- Workaround: export the cache variables in the command environment while keeping `MAKE_JOBS=2` set.
- Root cause: the lint child process did not use the Make-only cache variable values.

### ST-0168: sandbox blocked staging in shared worktree Git metadata

- Date: 2026-10-01
- Tool: `git add`
- Target: assigned branch index under the shared repository's `.git/worktrees` metadata
- Observed failure: sandbox denied creation of the Git index lock outside the writable worktree; no files were staged.
- Workaround: retry the user-authorized staging and commit command with elevated access.
- Root cause: the worktree shares Git administrative storage outside its source directory.

## ST-0169: Root semedit CLI uses short command names

- Tool: `/Users/alessandro/.codex/worktrees/0358327c-724a-44a6-bd87-1445d8d52fd7/semantic-editor/bin/semedit-next`
- Target: no source file was modified; semantic command help lookup only.
- Observed failure: invoking `semantic-replace-body --help` returned `unknown command`.
- Workaround: inspect the root binary command list and use its registered short CLI command names with the assigned worktree as cwd.
- Root cause: MCP semantic operation names are not the native CLI subcommand names.

## ST-0170: Root semedit CLI declaration command lookup

- Tool: `/Users/alessandro/.codex/worktrees/0358327c-724a-44a6-bd87-1445d8d52fd7/semantic-editor/bin/semedit-next`
- Target: no source file was modified; command help lookup only.
- Observed failure: invoking `insert-function --help` returned `unknown command`.
- Workaround: query the root binary's registered command names before using declaration insertion; no source workaround applied.
- Root cause: the MCP tool name does not match a native CLI subcommand, and the command list did not expose an `insert-function` alias.

## ST-0171: Semantic editor cannot add fields to existing Go structs

- Tool: `semedit-next` semantic declaration operations.
- Target: `internal/operation/operation.go` and `internal/mcp/server.go` if struct-field additions are required.
- Observed limitation: the registered semantic operations support whole declarations, functions, and constructors, but expose no operation for adding a field inside an existing struct declaration.
- Workaround: use a minimal AST-anchored atomic source update for the required struct fields, then format and verify with repository tooling.
- Root cause: the current semantic editor capability set has no struct-member insertion operation.

## ST-0172: Scaffold purpose header requires Go comment syntax

- Tool: `semedit-next scaffold-file`.
- Target: `internal/operation/output_projection.go`; scaffold was rejected before file creation.
- Observed failure: the plain-text purpose header was parsed as Go source and failed with `expected 'package', found This`.
- Workaround: pass the purpose header as a Go comment (`// ...`) and retry.
- Root cause: the scaffold tool validates the supplied purpose header as source text rather than wrapping it as a comment.

## ST-0173: Semantic insertion enforces exported Go test name access

- Tool: `semedit-next insert`.
- Target: `internal/mcp/schema_test.go`; insertion was rejected before mutation.
- Observed failure: specifying private visibility for `TestMCPProjectedReceiptUsesFinalDiskIdentityAndRevision` produced a visibility mismatch because Go test names are exported.
- Workaround: retry the test declaration with public visibility.
- Root cause: the generic insertion operation applies Go exported-name casing checks to test functions.

## ST-0174: Tool session poll returned a JavaScript syntax error

- Tool: `functions.exec` polling the pending `semedit-next insert` process.
- Target: `internal/mcp/schema_test.go`; source insertion outcome was not returned by the poll.
- Observed failure: the poll wrapper returned `Script failed SyntaxError: Invalid or unexpected token` instead of process output.
- Workaround: inspect repository state and rerun the insertion if the declaration is absent.
- Root cause: the tool-session polling response failed while decoding or relaying the process result; the specific boundary is not exposed.

## ST-0175: Semantic insertion accepted malformed schema test literals

- Tool: `semedit-next insert`.
- Target: `internal/mcp/schema_test.go`.
- Observed failure: the inserted test declaration produced semantic diagnostics for omitted nested composite literal types and called a nonexistent `SchemaKind.String` method.
- Workaround: explicitly qualify nested `operation.ValueSchema` literals and convert schema kinds with `string(...)`.
- Root cause: the input snippet relied on shorthand composite syntax not accepted in this context and treated a string alias as if it exposed a `String` method.

## ST-0176: zsh polling wrapper used a reserved variable name

- Tool: `functions.exec` session poll wrapper.
- Target: `.scratch/foundation-focused.log` inspection; repository source was not affected.
- Observed failure: the wrapper exited with `zsh: read-only variable: status` before returning the polled command output.
- Workaround: use a non-reserved variable name and inspect the existing log directly.
- Root cause: `status` is a read-only special parameter in zsh.

## ST-0177: Semantic CLI command alias was incorrect

- Tool: `semedit-next inspect --help`.
- Target: declaration inspection for `internal/operation/value_schema.go`; no source was changed.
- Observed failure: the CLI rejected `inspect` and suggested the actual command `inspect-symbol`.
- Workaround: use the registered `inspect-symbol` command.
- Root cause: CLI command names use the native operation names rather than the MCP semantic tool prefix.

## ST-0178: Tooling log command targeted the coordinator checkout

- Tool: atomic Python append for `docs/SUBOPTIMAL_TOOLS.md`.
- Target: coordinator checkout path; the filesystem rejected the write and no file was created there.
- Observed failure: `PermissionError` because the command ran with the coordinator checkout as its working directory, outside this worker’s writable root.
- Workaround: rerun the atomic append from this assigned worktree.
- Root cause: the command used a relative path with the wrong working directory.

## ST-0179: Semantic CLI insertion has no trust-workspace flag

- Tool: `semedit-next insert`.
- Target: `internal/operation/value_schema.go`; the attempted insertion was rejected before mutation.
- Observed failure: the CLI returned `unknown flag: --trust-workspace`.
- Workaround: retry using the supported insertion flags; workspace trust is supplied through MCP initialization options where applicable.
- Root cause: CLI trust configuration differs from the MCP initialization option.

## ST-0180: Semantic body tool cannot replace one statement in a test function

- Tool: `semedit-next replace-body`.
- Target: `internal/mcp/schema_test.go` test statement comparing receipt and disk revisions.
- Observed limitation: the available semantic operation replaces an entire function body and has no statement-level expression replacement; replacing this one assertion would require duplicating the full test body.
- Workaround: perform a narrowly matched atomic source update, then validate with Go formatting, lint, and the test suite.
- Root cause: no registered semantic edit operation targets a single statement.

## ST-0181: Semantic declaration tools cannot add fields to an existing struct

- Tool: `semedit-next insert` and `replace-decl`.
- Target: `internal/operation/receipt_finalize.go` (`EditReceiptDraft`).
- Observed limitation: declaration insertion cannot target struct fields, and `replace-decl` only supports constants, variables, and aliases.
- Workaround: atomically replace the single field line, then format and validate the declaration with repository tooling.
- Root cause: the semantic editor has no struct-field edit operation.

## ST-0182: Semantic tools cannot replace a struct-field use in a composite literal

- Tool: `semedit-next replace-body`.
- Target: `internal/mcp/schema_test.go` receipt finalizer test setup.
- Observed limitation: the edit changes a struct literal field and adds expected postimage setup inside a handler; no registered operation targets a nested literal field or statement range.
- Workaround: atomically replace the one literal statement with its planned digest setup, then rely on formatting and observable MCP tests.
- Root cause: the semantic editor only replaces complete function bodies for this class of Go edit.

## ST-0183: Semantic tools cannot extend one validation branch in Register

- Tool: `semedit-next replace-body`.
- Target: `internal/operation/operation.go` registration validation.
- Observed limitation: correcting the schema/projector guard requires adding one conditional in the existing function; the semantic body tool only replaces the complete 140-line body.
- Workaround: apply the exact conditional as an atomic source update, then run formatter, lint, registration tests, and full check.
- Root cause: no registered operation edits a selected statement or branch.

## ST-0184: Semantic insertion did not create the requested test declaration

- Tool: `semedit-next insert`.
- Target: `internal/operation/value_schema_test.go`.
- Observed failure: the command returned no declaration result, and a follow-up `inspect-symbol` confirmed the requested registration test was absent.
- Workaround: retry with a plain Go address operator in the source snippet and verify the inserted symbol before proceeding.
- Root cause: the insertion payload encoded the address operator as a Unicode escape that was not accepted as Go source.

## ST-0185: Semantic insertion rejected literal newline escapes

- Tool: `semedit-next insert`.
- Target: `internal/operation/value_schema_test.go`.
- Observed failure: the retry returned `syntax error: illegal character U+005C` because the Go snippet contained literal backslash-n sequences.
- Workaround: provide actual line breaks in the source snippet and inspect the resulting declaration.
- Root cause: shell and JavaScript escaping preserved the escape text instead of producing source newlines.

## ST-0186: Semantic tools cannot insert a public-call block inside a test

- Tool: `semedit-next insert` / `replace-body`.
- Target: `internal/mcp/schema_test.go` nested continuation flow test.
- Observed limitation: the required test adds a call and value handoff inside an existing test; available semantic operations only insert top-level declarations or replace an entire function body.
- Workaround: use a single exact-match atomic insertion at the assertion boundary, then verify via formatting, lint, and the public MCP test.
- Root cause: statement-range insertion is not supported.

## ST-0187: Semantic body replacement returned without applying source

- Tool: `semedit-next replace-body`.
- Target: `internal/operation/receipt_finalize.go` function `FinalizeEditReceipt`.
- Observed failure: the command produced no confirmation and left the function referencing the removed `ChangePaths` field, which the full check detected.
- Workaround: retry with actual line breaks in the replacement body and inspect the resulting declaration.
- Root cause: the replacement payload contained literal backslash-n characters from shell/JavaScript escaping.

## ST-0188: Escaped semantic function edits were rejected without inspection

- Tool: `semedit-next replace-body` and `insert`.
- Target: `internal/operation/value_schema.go`, `internal/operation/parse.go`, and `internal/operation/output_projection.go`.
- Observed failure: several function-edit commands encoded `\n` literally; the CLI rejected those snippets and the existing declarations remained unchanged. A later symbol check found the bounded numeric helper absent.
- Workaround: use actual line breaks in Go snippets, check each command exit code, and inspect every edited declaration.
- Root cause: JSON/JavaScript escaping preserved backslash-n text when real source newlines were required.

## ST-0189: Shell quoting broke the integration note update

- Tool: `functions.exec` wrapper running an atomic Python note update.
- Target: `.scratch/foundation-integration.md`; no file was changed by the failed command.
- Observed failure: zsh reported `unmatched "` because an apostrophe in the note text terminated the shell's single-quoted Python argument.
- Workaround: rerun the same atomic update through a quoted heredoc.
- Root cause: shell quoting was not safe for the embedded prose.

## ST-0190: Semantic edit tools cannot add a local binding inside a closure

- Tool: `semedit-next insert` / `replace-body`.
- Target: `internal/mcp/schema_test.go` receipt finalizer test closure.
- Observed limitation: fixing the test requires adding a local request binding inside a nested closure; the available semantic tools only insert top-level declarations or replace the complete enclosing function body.
- Workaround: apply one exact atomic insertion, then verify with `make check` and the observable MCP test.
- Root cause: nested statement insertion is not supported.

## ST-0191: Semantic tools cannot replace one nested result assertion

- Tool: `semedit-next replace-body`.
- Target: `internal/mcp/schema_test.go` removal continuation assertion.
- Observed limitation: the test must accept both an omitted `omitempty` slice and an explicit empty slice; the semantic tool replaces the whole enclosing test body to change a single assertion.
- Workaround: make an exact atomic statement replacement and verify with the full test suite.
- Root cause: statement-level edits are unsupported.

## ST-0192: Semantic tools cannot insert publication reconciliation in a function

- Tool: `semedit-next insert` / `replace-body`.
- Target: `internal/operation/receipt_finalize.go` outcome reconciliation.
- Observed limitation: the function needs a counter and status adjustment around an existing loop; insertion supports declarations only and replacement covers the whole helper.
- Workaround: apply narrow exact atomic changes, then run full formatting, lint, and receipt tests.
- Root cause: statement-range editing is unavailable.

## ST-0193: Exact-match test edit did not match current formatting

- Tool: atomic Python source update.
- Target: `internal/operation/value_schema_test.go` planned change receipt test.
- Observed failure: the script assertion found no exact multiline match and stopped before creating or replacing a file.
- Workaround: inspect the current declaration and edit the exact existing initializer.
- Root cause: the source formatting differed from the assumed indentation and line layout.

## ST-0194: Correction to ST-0193 target description

- Tool: atomic Python source update.
- Target: `internal/operation/value_schema_test.go` planned change receipt test, not the MCP test file.
- Observed issue: ST-0193 named the wrong target path; the failed edit itself still made no changes.
- Workaround: inspect `TestFinalizeEditReceiptIncludesOnlyObservedPostimages` and update its status initializer.
- Root cause: the log entry recorded the caller path instead of the actual intended file.

## ST-0195: Process inspection is blocked by the sandbox

- Tool: `ps -Ao pid,etime,command` while observing final `make check` completion.
- Target: local process status only; no source or process state was changed.
- Observed failure: zsh reported `operation not permitted: ps`.
- Workaround: use the existing command session and its log to determine completion.
- Root cause: process inspection is restricted in this sandbox.

## ST-0196: Redundant poll after `make check` completion failed

- Tool: `functions.exec` polling the final `make check` session.
- Target: final check completion status; repository state was unaffected.
- Observed failure: the polling call returned `Unknown process id` after the preceding poll had already returned exit code 0.
- Workaround: use the recorded successful exit code and check log.
- Root cause: the session had already terminated and been closed.

## ST-0197: Semantic tools cannot extend an existing MCP test handler inline

- Tool: `semedit-next insert` / `replace-body`.
- Target: `internal/mcp/schema_test.go` receipt mutation test.
- Observed limitation: the same-name replacement scenario needs new request parsing, mutation behavior, and response assertions inside one existing test function; available semantic operations only add top-level declarations or replace the entire function body.
- Workaround: make bounded exact-match atomic source updates and validate them through the real public MCP test and full check.
- Root cause: no nested statement or struct-literal insertion operation exists.

## ST-0198: Git worktree metadata is outside the writable root

- Tool: `git add` for the owned foundation files.
- Target: shared worktree index at `/Users/alessandro/sources/semantic-editor/.git/worktrees/flow-foundation/index.lock`.
- Observed failure: the sandbox denied creation of the index lock; no files were staged and the commit did not run.
- Workaround: retry the authorized commit with scoped access to the shared git metadata.
- Root cause: this linked worktree stores index metadata in the common repository directory outside the assigned writable root.

### ST-0169: local CLI scaffolding rejects plain purpose text

- Date: 2026-10-01
- Tool: `semedit scaffold-file` CLI
- Target: `internal/contractplan/plan.go` in the requested worktree
- Observed failure: passing `--purpose-header 'WHY: ...'` returned `syntax error: invalid purpose header: scaffold.go:1:1: expected 'package', found WHY`; the command created no file.
- Workaround: retry with the purpose header expressed as a Go line comment, then verify the file landed in this worktree.
- Root cause: the CLI help says the header explains file purpose but does not state that the value must be valid Go comment syntax.

### ST-0170: function insertion rejected escaped Go string syntax

- Date: 2026-10-01
- Tool: `semedit insert-func` CLI
- Target: `internal/contractplan/implement_contract.go` in the requested worktree
- Observed failure: inserting the pure contract planner returned `validate function snippet: syntax error: expected declaration, found declarations`; the command made no source change.
- Workaround: correct the Go string literal in the `panic` stub source and retry the same semantic insertion.
- Root cause: the shell argument was single-quoted, so unnecessary backslashes before Go double quotes reached the semantic parser and made the snippet invalid.

### ST-0171: retry retained an invalid shell-escaped Go string

- Date: 2026-10-01
- Tool: `semedit insert-func` CLI
- Target: `internal/contractplan/implement_contract.go` in the requested worktree
- Observed failure: retrying the same planner insertion still returned `validate function snippet: syntax error: expected declaration, found declarations`; no planner source was written.
- Workaround: use a Go raw string literal for the explicit panic marker, avoiding shell escaping.
- Root cause: the retry preserved the backslashes because the command construction still represented `\"` inside a single-quoted shell argument.

### ST-0172: planner insertion still failed through inline source argument

- Date: 2026-10-01
- Tool: `semedit insert-func` CLI
- Target: `internal/contractplan/implement_contract.go` in the requested worktree
- Observed failure: a third inline attempt returned `validate function snippet: syntax error: expected declaration, found declarations`; no planner source was written.
- Workaround: pass a multiline source snippet as one argument through a local subprocess argument vector, preserving the semantic CLI edit operation without shell interpolation.
- Root cause: the compact, shell-quoted snippet was not parsed as one function declaration by the CLI insertion validator.

### ST-0173: function insertion rejected grouped test helpers

- Date: 2026-10-01
- Tool: `semedit insert-func` CLI
- Target: `internal/contractplan/planner_test.go` in the requested worktree
- Observed failure: passing three helper functions in one `--source` value returned `multiple declarations found in snippet: expected single function declaration, found 3 declarations`; no source changed.
- Workaround: submit each helper as its own semantic insertion.
- Root cause: `insert-func` enforces a single-function source snippet.

### ST-0174: symbol inspection used the wrong source file

- Date: 2026-10-01
- Tool: `semedit inspect-symbol` CLI
- Target: `PlanImplementContract` in `internal/contractplan/implement_contract.go`
- Observed failure: the request looked for the function in `internal/contractplan/plan.go` and returned `symbol not found`; no source changed.
- Workaround: rerun inspection with the function's actual file from the semantic outline.
- Root cause: the caller supplied a mismatched file/symbol pair.

### ST-0175: final check surfaced new-package lint blockers

- Date: 2026-10-01
- Tool: `make check` / `golangci-lint run --fix`
- Target: `internal/contractplan`
- Observed failure: the repository check stopped with 20 findings: exported planner API declarations lacked documentation comments, and five method/tuple index loops were flagged for Go 1.22 integer-range syntax.
- Workaround: add concise API documentation and rewrite loops using range over the go/types method and tuple sequences.
- Root cause: the first implementation pass had not yet applied the repository's configured revive and intrange rules.

### ST-0176: semantic body edits used invalid iterator loop forms

- Date: 2026-10-01
- Tool: `semedit replace-body` CLI
- Target: `internal/contractplan` helpers
- Observed failure: semantic edits replaced the loops, but diagnostics reported compile errors because `go/types` `Methods()` and `Variables()` return `iter.Seq`, which accepts one range variable; the first tuple rewrite also treated `Variables()` as a slice.
- Workaround: use a one-variable range for iterator sequences and integer range plus `At` for indexed variadic parameter handling.
- Root cause: the caller assumed go/types method and tuple accessors still returned slices in Go 1.27.

### ST-0177: documentation insertion script found a header mismatch

- Date: 2026-10-01
- Tool: local atomic source update script
- Target: `internal/contractplan/extract_interface.go`
- Observed failure: after adding declaration comments to `plan.go`, the script stopped because its expected `WHY` header plus package-clause literal did not match the scaffolded file; no change was made to the target file.
- Workaround: inspect the actual header and apply the package and API comments against the observed text.
- Root cause: the script assumed the scaffold header text had not been reformatted by semantic insertion.

### ST-0178: package comment formatting failed revive validation

- Date: 2026-10-01
- Tool: `make check` / `golangci-lint run --fix`
- Target: `internal/contractplan/extract_interface.go`
- Observed failure: revive reported the package comment detached and then reported no package comment; the check stopped before tests.
- Workaround: place the package documentation directly above the package clause with no blank line between the comment group and the clause.
- Root cause: the scaffolded blank line after the `WHY` comment split the package doc from the package clause.

### ST-0161: `semantic_batch` verification lacked workspace trust

- Date: 2026-10-01
- Tool: `semantic_batch` with `semantic_replace_body`
- Target: `internal/contractplan` planner files
- Observed failure: Every mutation child reported `status=ok`; the final `semantic_verify` returned `isError` because workspace trust was not enabled for the configured `golangci-lint` hook. Successful child edits were published per the runtime contract.
- Workaround: Keep the successful source edits and initialize subsequent local MCP sessions with `initializationOptions.trust_workspace=true`.
- Root cause: The MCP initialization omitted the trust flag required to run the authorized repository lint hook.

### ST-0162: `semantic_batch` verification lacked workspace trust

- Date: 2026-10-01
- Tool: `semantic_batch` with multiple `semantic_replace_body` children
- Target: `internal/contractplan` planner files
- Observed failure: Every mutation child reported `status=ok`; the final `semantic_verify` returned `isError` because workspace trust was not enabled for the configured `golangci-lint` hook. Successful child edits were published per the runtime contract.
- Workaround: Keep the successful source edits and initialize subsequent local MCP sessions with `initializationOptions.trust_workspace=true`.
- Root cause: The MCP initialization omitted the trust flag required to run the authorized repository lint hook.

### ST-0163: `semantic_batch` verification lacked workspace trust

- Date: 2026-10-01
- Tool: `semantic_batch` with planner test edits
- Target: `internal/contractplan/planner_test.go`
- Observed failure: Every mutation child reported `status=ok`; the final `semantic_verify` returned `isError` because workspace trust was not enabled for the configured `golangci-lint` hook. The additional `backend.InspectedSymbol` diagnostics came from gopls v0.23.0 built with Go 1.26 rejecting valid Go 1.27 embedded-field composite literals; they were not cache-related source failures.
- Workaround: Keep the successful source edits, seed the worker-local Go cache, and initialize subsequent local MCP sessions with `initializationOptions.trust_workspace=true`.
- Root cause: The MCP initialization omitted the trust flag and its isolated Go base directory had not yet received the immutable module-cache seed.

### ST-0164: test body replacement removed fixture setup

- Date: 2026-10-01
- Tool: `semantic_replace_body` inside `semantic_batch`
- Target: `internal/contractplan/planner_test.go` (`TestPlanImplementContractGeneratesOnlyMissingPanicStub`)
- Observed failure: `make check` found undefined `err` and `plan` identifiers because the replacement body contained only assertions and omitted the existing fixture and planner invocation.
- Workaround: Restore the complete test body, including source fixture construction and planner invocation, through a semantic body replacement.
- Root cause: The replacement body was authored as a suffix after overlooking that `semantic_replace_body` replaces the entire body.

### ST-0165: scaffold purpose header was not Go comment syntax

- Date: 2026-10-01
- Tool: `semedit scaffold-file` CLI
- Target: `internal/operation/wire_contracts.go`
- Observed failure: scaffolding refused the purpose header with `expected 'package', found WHY`; it created no source file.
- Workaround: retry with the purpose header prefixed as a Go line comment.
- Root cause: the caller passed plain prose where the scaffold parser expects valid Go comment syntax.

### ST-0166: contract handler helper snippet missed a statement separator

- Date: 2026-10-01
- Tool: `semantic_batch` with `semantic_insert_function`
- Target: `internal/operation/wire_contracts.go` (`canonicalContractPath`)
- Observed failure: The insert operation rejected the function with `expected ';', found info` because two statements were joined on one source line; earlier successful type and loader children were persisted in the returned final diff.
- Workaround: Inspect the current file and reinsert the corrected function with statements on separate lines.
- Root cause: The generated function snippet concatenated `}` and the following declaration without a newline or semicolon.

### ST-0167: semantic verification reported mismatched runtime type fields

- Date: 2026-10-01
- Tool: `semantic_batch` final `semantic_verify` with workspace trust enabled
- Target: `internal/operation/wire_contracts.go` (the helper inserts succeeded)
- Observed failure: The final verification hook failed with `unknown field` diagnostics for `backend.InspectedSymbol` fields in `internal/astedit/inspect.go`, although the authoritative `make check` passed on this worktree immediately before these handlers began.
- Workaround: Preserve the successful helpers, inspect the checked source/binary views, and use the repository Make check to validate package correctness while reporting the semantic verification mismatch.
- Root cause: The active gopls v0.23.0 binary was built with Go 1.26 and rejected Go 1.27-valid embedded-field composite literals in `backend.InspectedSymbol`.

### ST-0168: semantic verification repeated the runtime type-view mismatch

- Date: 2026-10-01
- Tool: `semantic_batch` final `semantic_verify` with workspace trust enabled
- Target: `internal/operation/wire_contracts.go` (`contractImportBindings`, candidate rendering, and typecheck helpers)
- Observed failure: All three helper mutations reported success, then the trusted verification hook repeated the same `backend.InspectedSymbol` unknown-field diagnostics recorded in ST-0167.
- Workaround: Retain the successful helpers and validate them through the prescribed Make check; do not repeat workspace verification until the checked binary/package view is reconciled.
- Root cause: The active gopls v0.23.0 binary was built with Go 1.26 and rejected Go 1.27-valid embedded-field composite literals in `backend.InspectedSymbol`.

### ST-0169: contract parser batch could not run the verification hook

- Date: 2026-10-01
- Tool: `semantic_batch` with four `semantic_insert_function` children
- Target: `internal/operation/wire_contracts.go` (contract request parsers)
- Observed failure: All four mutation children reported `status=ok` with `diagnostic_status=deferred`; final `semantic_verify` returned `workspace trust is required to run verification hook "golangci-lint"`. The tool response also included the previously recorded gopls v0.23.0 / Go 1.27 compatibility diagnostics for valid embedded-field composite literals in `backend.InspectedSymbol`. Successful children were persisted.
- Workaround: Preserve all four parsers and record that this MCP connection did not carry the required workspace-trust initialization; stop issuing more MCP writes until a trusted connection is available.
- Root cause: The active MCP connection was initialized without `params.initializationOptions.trust_workspace=true`, so the authorized configured lint hook was denied.

### ST-0170: parser verification ran before operation schemas were declared

- Date: 2026-10-01
- Tool: `semantic_verify` with `trust_workspace=true`
- Target: `internal/operation/wire_contracts.go`
- Observed failure: The trusted `golangci-lint` hook reported undefined `extractInterfaceParams` and `implementContractParams` referenced by the persisted parser children. Its response also repeated gopls v0.23.0 / Go 1.27 compatibility diagnostics for valid embedded-field composite literals in `backend.InspectedSymbol`.
- Workaround: Keep the four parser children and finish the operation parameter declarations before rerunning the prescribed Make check.
- Root cause: The parser functions were inserted before their public operation schema definitions, so this intermediate source state does not type-check.

### ST-0171: trusted batch linter rejected the schema declaration intermediate state

- Date: 2026-10-01
- Tool: `semantic_batch` with two `semantic_insert_decl` children through a locally initialized trusted MCP server
- Target: `internal/operation/wire_contracts.go` (`extractInterfaceParams`, `implementContractParams`)
- Observed failure: Both declaration children returned `status=ok`; the final trusted verification hook reported `golangci-lint` failed. The JSON response was truncated after the failure and gopls v0.23.0 / Go 1.27 compatibility diagnostics, so no more specific linter diagnostic was available from that invocation.
- Workaround: Preserve both parameter declarations and continue the dependent operation wiring before the next authoritative `make check`.
- Root cause: The parameter declarations were added while parser and analysis helpers remain disconnected from registered operation definitions; this is an intermediate implementation state.

### ST-0172: trusted contract-handler batch stopped at intermediate lint

- Date: 2026-10-01
- Tool: `semantic_batch` with ten `semantic_insert_function` children through a locally initialized trusted MCP server
- Target: `internal/operation/wire_contracts.go` (revision, preflight, output, and handler helpers)
- Observed failure: All ten mutation children returned `status=ok`; final `semantic_verify` reported the `golangci-lint` hook failed. The response also repeated gopls v0.23.0 / Go 1.27 compatibility diagnostics for valid embedded-field composite literals in `backend.InspectedSymbol`; the large response was truncated before the complete hook output could be captured.
- Workaround: Preserve the successful helpers. Connect their parsers and operation definitions after the central structured schema API is integrated, then use the prescribed Make check as the next full verification.
- Root cause: The new handler helpers are not yet reachable from registered typed operation definitions, so this is an unfinished integration state.

### ST-0173: contract parser batch could not connect to Go diagnostics

- Date: 2026-10-01
- Tool: `semantic_batch` with two `semantic_replace_body` children through a local trusted MCP server
- Target: `internal/operation/wire_contracts.go` (`parseExtractInterfaceRequest`, `parseImplementContractRequest`)
- Observed failure: Batch diagnostics failed before child execution with `initialize gopls diagnostics session: JSON-RPC client is closed`; no mutations were applied.
- Workaround: Keep both parser bodies unchanged and inspect/restart the local Go diagnostics session before retrying.
- Root cause: The MCP operation could not initialize its retained gopls diagnostics client in the current server process.

### ST-0174: PTY truncated an MCP request sent to the persistent worker

- Date: 2026-10-01
- Tool: local `exec_command` PTY feeding the trusted MCP wrapper
- Target: none; the request contained parser body replacements for `internal/operation/wire_contracts.go`
- Observed failure: The JSON request exceeded the PTY canonical input limit; the wrapper received no complete request or MCP response. I interrupted the wrapper, and no source changes were applied.
- Workaround: Restart the persistent trusted MCP wrapper and send smaller individual requests below the PTY input limit.
- Root cause: A large JSON-RPC request was sent as one canonical-mode PTY line.

### ST-0176: optional revision parser body no longer matched its signature

- Date: 2026-10-01
- Tool: `semantic_batch` with a constant insert, two parameter declaration replacements, and three function body replacements through a local trusted MCP server
- Target: `internal/operation/wire_contracts.go` (`parseExpectedContractRevision` and both request parsers)
- Observed failure: The mutation response ended with a trusted `golangci-lint` failure. Current source inspection shows the function signature still returns `(*string, error)` while the replacement body returns three values; its two callers also expect three values. Earlier declaration children succeeded.
- Workaround: Restore the declared two-result helper contract and have omitted revisions return a non-nil pointer to the empty sentinel; `checkContractRevision` will treat that sentinel as absence.
- Root cause: `semantic_replace_body` changes only a function body, but the edit changed the helper's return arity without changing its declaration.

### ST-0175: persistent MCP FIFO wrapper received an empty line after one edit

- Date: 2026-10-01
- Tool: local `exec_command` FIFO transport for the trusted MCP wrapper
- Target: none; the second parser body request was not passed to MCP
- Observed failure: The wrapper had returned success for the first parser body replacement, then exited with `JSONDecodeError` on an empty line before receiving the next request. No source mutation was applied for the second request.
- Workaround: Use the one-shot trusted MCP wrapper with the checked binary and local gopls path for each call.
- Root cause: The persistent wrapper used an `O_RDWR` FIFO descriptor and interpreted an empty FIFO read as a request line.

### ST-0177: trusted panic-shadow handler batch failed final lint

- Date: 2026-10-01
- Tool: `semantic_batch` with `semantic_insert_function` and `semantic_replace_body` through the root trusted pipe helper
- Target: `internal/operation/wire_contracts.go` (`requirePanicBuiltin`, `runImplementContract`)
- Observed failure: Both mutation children reported `status=ok`; the aggregate result had `isError=true` at final verification. The rendered tool output was truncated before the verification hook diagnostic could be captured. The current source contains both successful children.
- Workaround: Preserve the helper and its call site; inspect the next prescribed Make check after the typed operation definitions make the helpers reachable.
- Root cause: The contract handlers remain disconnected from registered operation definitions while the schema integration is pending; final verification rejects this intermediate state.

### ST-0178: planner builtin guard batch was rejected before mutation

- Date: 2026-10-01
- Tool: `semantic_batch` through the root trusted pipe helper
- Target: `internal/contractplan/implement_contract.go` and `internal/contractplan/planner_test.go`
- Observed failure: The first `semantic_replace_body` child was rejected with `syntax error: string literal not terminated (and 2 more errors)`; the response reported an empty final diff and no child mutation.
- Workaround: Preserve planner sources and use the handler-level builtin-shadow refusal with its no-publication test; do not retry the malformed declaration snippet.
- Root cause: The generated Go body string was malformed during payload construction.

### ST-0179: planner panic-shadow batch failed its final verification

- Date: 2026-10-01
- Tool: `semantic_batch` with `semantic_replace_body` and two `semantic_insert_function` children through the root trusted pipe helper
- Target: `internal/contractplan/implement_contract.go` and `internal/contractplan/planner_test.go`
- Observed failure: The helper returned `isError=true` at final verification. Its large response was truncated after the first child result; inspection of current source confirms the planner guard and both-case refusal test were persisted.
- Workaround: Preserve the edits and use the next required Make check after operation schema/registration integration to validate the whole worktree.
- Root cause: Final lint still sees the incomplete contract operation wiring in `internal/operation/wire_contracts.go`; the panic-shadow edits themselves are present.

### ST-0180: sandbox denied Git index creation for the worktree

- Date: 2026-10-01
- Tool: `git add` for the assigned contracts worktree
- Target: index for branch `codex/flow-contracts`; no index or source mutation occurred
- Observed failure: Git could not create `/Users/alessandro/sources/semantic-editor/.git/worktrees/flow-contracts/index.lock` because the shared Git administrative directory is outside the writable worktree.
- Workaround: Retry the user-authorized explicit-path stage and commit through the sandbox escalation path.
- Root cause: The worktree shares Git administrative storage outside its source directory.

### ST-0181: semantic outline rejected the legacy file parameter

- Date: 2026-10-01
- Tool: `semantic_outline` through the root trusted pipe helper
- Target: `internal/operation/wire_contracts_test.go`; no source mutation occurred
- Observed failure: The tool rejected `file` as an unknown parameter and listed `path` as the supported location parameter.
- Workaround: Retry outline with `path` after recording this failure.
- Root cause: The payload used a stale parameter name from another semantic operation.

### ST-0182: semantic replace-body rejected the legacy path parameter

- Date: 2026-10-01
- Tool: `semantic_replace_body` through the root trusted pipe helper
- Target: `internal/operation/wire_contracts_test.go`; no source mutation occurred
- Observed failure: The tool rejected `path` and listed `file` as the supported source location parameter.
- Workaround: Retry with `file` while keeping the reviewed replacement body unchanged.
- Root cause: Semantic operations have operation-specific location keys; the payload reused the outline key.

### ST-0183: semantic batch rejected a renderer struct-field edit

- Date: 2026-10-01
- Tool: `semantic_batch` with two `semantic_replace_body` children followed by a `semantic_replace_body` for `typeRenderer`
- Target: `internal/contractplan/plan.go` and `internal/contractplan/planner_test.go`
- Observed failure: The first two function-body children reported `status=ok`; the third child reported `symbol not found` for `typeRenderer`, a struct declaration. The batch ended with `status=error` before the remaining test children.
- Workaround: Preserve the two successful changes, inspect actual source bytes, then use supported `semantic_remove_construct` plus `semantic_insert_type` for the whole struct declaration in a later coherent batch.
- Root cause: `semantic_replace_body` only targets function bodies and cannot edit struct fields.

### ST-0184: trusted contract renderer batch failed final lint

- Date: 2026-10-01
- Tool: `semantic_batch` with `semantic_remove_construct` and `semantic_insert_type`, through the root trusted pipe helper
- Target: `internal/contractplan/plan.go`
- Observed failure: Both mutation children reported `status=ok`; final `semantic_verify` reported `verification hook golangci-lint failed` with exit status 1. The response included repository warnings and hints but was truncated before all diagnostics.
- Workaround: Preserve both source mutations, inspect their current bytes, then use the focused test and Make output to identify actionable lint errors.
- Root cause: Not established from the truncated response; do not infer a trust failure or rollback.

### ST-0185: trusted planner proof batch failed final lint

- Date: 2026-10-01
- Tool: `semantic_batch` with `semantic_insert_type`, `semantic_insert_function`, and `semantic_replace_body`
- Target: `internal/contractplan/planner_test.go`
- Observed failure: All seven mutation children reported `status=ok`; final `semantic_verify` reported `golangci-lint` exit status 1. The tool returned the successful source diff, but the diagnostic tail was truncated.
- Workaround: Preserve all inserted proof code, run the required Make check for complete lint/compiler diagnostics, then fix the reported source issue.
- Root cause: Not established from the truncated response; do not infer the successful children rolled back.

### ST-0186: parser proof batch response was truncated

- Date: 2026-10-01
- Tool: `semantic_batch` with `semantic_replace_body` and `semantic_insert_function`
- Target: `internal/operation/wire_contracts.go` and `internal/operation/wire_contracts_test.go`
- Observed failure: The first child reported `status=ok`. The helper returned `isError=true` and its large structured diff was truncated before the second child status and final verification diagnostic could be read.
- Workaround: Preserve the successful source change, inspect current source bytes and run focused parser tests before any retry.
- Root cause: Not established from the truncated response; do not infer that either child was rolled back.

### ST-0187: contract output type replacement passed mutations but failed final lint

- Date: 2026-10-01
- Tool: `semantic_batch` with `semantic_remove_construct` and `semantic_insert_type` through the trusted root pipe helper
- Target: `internal/operation/wire_contracts.go` (`contractOutput` declaration)
- Observed failure: Both mutation children reported `status=ok`; final `semantic_verify` ran the configured `golangci-lint` hook and exited 1. It reported the expected `expected_revision` goconst warning and five unused contract-registration helpers/parameters because public wiring is still incomplete; it did not report a trust failure.
- Workaround: Preserve both successful mutations and continue the coherent typed registration change, then run Make for full lint output.
- Root cause: The batch stopped at repository lint while the in-progress public operation definitions remained unregistered.

### ST-0188: typed contract output helper batch failed final verification

- Date: 2026-10-01
- Tool: `semantic_batch` with seven `semantic_insert_function` children through the trusted root pipe helper
- Target: `internal/operation/wire_contracts.go` (signature/result schemas, projector, receipt finalizer, resolver, and formatter)
- Observed failure: All seven insertion children reported `status=ok`; the final `semantic_verify` ran `golangci-lint` and exited 1 while the newly added contract output functions were still unregistered/unused. The response diagnostics also included the existing `expected_revision` goconst issue; no workspace-trust error occurred.
- Workaround: Preserve all inserted helpers, complete registration and imports in the same operation file, then run the required Make check for actionable diagnostics.
- Root cause: The helper batch was applied before its operation definitions and import organization were wired.

### ST-0189: semantic removal rejected an unsupported variable kind name

- Date: 2026-10-01
- Tool: `semantic_batch` with `semantic_remove_construct` as its first child through the trusted root pipe helper
- Target: `internal/operation/wire_contracts.go`; no mutation occurred
- Observed failure: The tool rejected kind `var` and listed `variable` as the supported construct kind. The batch stopped before any child mutation.
- Workaround: Retry the same coherent replacement batch using the supported `variable` kind.
- Root cause: The payload used a Go token spelling instead of the semantic operation construct vocabulary.

### ST-0190: contract registration batch failed final lint

- Date: 2026-10-01
- Tool: `semantic_batch` with contract output type replacement, registry registration, and import organization through the trusted root pipe helper
- Target: `internal/operation/wire_contracts.go` and `internal/operation/wire_backend.go`
- Observed failure: The output type replacement and `DefaultRegistry` body edit reported `status=ok`. Final `semantic_verify` ran the configured `golangci-lint` hook and returned exit status 1; the tool response was truncated before the full diagnostic list. No trust error was reported.
- Workaround: Preserve the successful mutations, inspect current source, and run `make check` for complete diagnostics before continuing.
- Root cause: Not yet established from the available truncated tool result.

### ST-0191: contract registration rejected continuation schema and lint issues

- Date: 2026-10-01
- Tool: `make check` after registering the contract operations
- Target: `internal/operation/wire_contracts.go` and registry initialization
- Observed failure: `docgen-source` panicked because the declared `result.receipt.continuations[].file` string continuation schema was not equivalent to the shared `SymbolTargetSchema` nonempty file leaf. `golangci-lint` also reported repeated `receiver_kind` literals. No focused tests were run.
- Workaround: Align continuation schemas with the shared descriptor leaf schema and name the repeated request field constant, then rerun Make.
- Root cause: The operation output contract used generic strings for constrained live target fields, and the operation-local field literal crossed the repository goconst threshold.

### ST-0192: contract schema and parser batch failed final verification

- Date: 2026-10-01
- Tool: `semantic_batch` with schema, parameter declaration, and parser updates through the trusted root pipe helper
- Target: `internal/operation/wire_contracts.go`
- Observed failure: The visible children reported `status=ok`; the aggregate response returned `isError=true` at final verification, but its full lint diagnostics were truncated. The changed parameter and parser declarations are present in the current source.
- Workaround: Preserve the completed mutations and use the required Make check to capture complete lint and registry diagnostics.
- Root cause: Not established from the truncated response.

### ST-0193: semantic outline rejected an unsupported depth parameter

- Date: 2026-10-01
- Tool: `semantic_outline` through the trusted root pipe helper
- Target: `internal/operation/typed_descriptors.go`; no mutation occurred
- Observed failure: The request returned `unknown param "depth"` and listed `path`, `language`, `kinds`, and related supported keys.
- Workaround: Retry navigation using only the documented `path` and supported outline parameters.
- Root cause: The payload guessed an unsupported hierarchy-depth parameter instead of following the operation schema.

### ST-0194: semantic replacement payload generator failed before MCP dispatch

- Date: 2026-10-01
- Tool: trusted `semantic_replace_body` helper invocation
- Target: `internal/operation/wire_contracts.go`; no mutation occurred
- Observed failure: The shell's inline Python payload generator had an unclosed parenthesis and emitted no JSON; the helper then rejected empty stdin before starting an MCP request.
- Workaround: Write the exact payload with a simpler JSON-safe script, then retry through the trusted pipe helper.
- Root cause: The one-line Python expression was malformed while escaping a multiline Go body.

## ST-0199: CLI lookup rejected documented selector flags

- Tool: `/Users/alessandro/sources/semantic-editor/bin/semedit-next lookup`.
- Target: declaration navigation in `internal/mcp/server.go`, `internal/mcp/batch.go`, `internal/operation/receipt_finalize.go`, `internal/operation/typed_descriptors.go`, and `internal/operation/wire_read_inspect.go`.
- Observed failure: each `lookup --file <path> --symbol <name>` request returned `param "symbol" is required: invalid operation parameters`.
- Workaround: use the CLI's read-only `outline --path` operation and inspect the specific declaration via its output.
- Root cause: the checked binary's lookup argument decoder did not pass the documented flag into the operation request in this worktree.

## ST-0200: CLI inspection command name differed from expectation

- Tool: `/Users/alessandro/sources/semantic-editor/bin/semedit-next inspect --help`.
- Target: read-only inspection navigation for `semantic_inspect_symbol`.
- Observed failure: the CLI reported `unknown command "inspect"` and suggested `inspect-symbol`.
- Workaround: use `inspect-symbol` for the documented semantic inspection command.
- Root cause: the CLI command uses the `inspect-symbol` spelling.

## ST-0201: zsh split combined semantic inspection arguments incorrectly

- Tool: `/Users/alessandro/sources/semantic-editor/bin/semedit-next inspect-symbol`.
- Target: targeted function source inspection in MCP and operation files.
- Observed failure: the loop's `set -- $spec` under zsh did not preserve the `--file` and `--symbol` arguments, and the CLI rejected each request as missing `symbol`.
- Workaround: issue exact per-symbol commands with quoted arguments.
- Root cause: zsh does not perform the expected shell-style whitespace splitting without its explicit split flag.

## ST-0202: Semantic insert operation uses grouped CLI command

- Tool: `/Users/alessandro/sources/semantic-editor/bin/semedit-next insert-function --help`.
- Target: adding small helper declarations during the foundation fix.
- Observed failure: the CLI reported that `insert-function` is not a registered command.
- Workaround: inspect and use the supported grouped `insert` command for declarations, or use exact atomic updates only when the available semantic operation cannot express a structural change.
- Root cause: the CLI groups declaration insertion under `insert`.

## ST-0203: Shell quoting interrupted semantic body replacement

- Tool: `/Users/alessandro/sources/semantic-editor/bin/semedit-next replace-body`.
- Target: `internal/mcp/server.go` batch output schema builder.
- Observed failure: zsh reported `unmatched \"` while parsing the source passed with `--body`; the semantic operation did not run.
- Workaround: use a shell-safe encoded argument with a source string that contains no shell quote terminators.
- Root cause: the Go string literal contained an apostrophe inside a single-quoted shell argument.

## ST-0204: Semantic inspection used an unresolved generic method name

- Tool: `semedit-next inspect-symbol`.
- Target: `internal/operation/operation.go` dispatch implementation.
- Observed failure: `Registry.Dispatch` was not found in the selected file.
- Workaround: navigate from the declared `Dispatch` symbol in the outline or rely on the already inspected typed dispatch path.
- Root cause: the method is declared on a generic registry receiver with a different qualified selector than the attempted name.

## ST-0205: Semantic insertion visibility did not match Go test casing

- Tool: `semedit-next insert`.
- Target: new test in `internal/operation/value_schema_test.go`.
- Observed failure: the tool rejected `TestFinalizeEditReceiptRequiresObservedCandidateDigest` because visibility was explicitly `private` despite exported Go casing. No declaration was inserted.
- Workaround: retry with `--visibility public`.
- Root cause: the semantic editor enforces access casing for inserted declarations.

## ST-0206: Focused MCP suite found flat-schema assertions after batch union change

- Tool: `go test ./internal/operation ./internal/mcp`.
- Target: MCP schema assertions in `internal/mcp/schema_test.go`.
- Observed failure: three tests expected `results.items.required` and a flat `result` property; the new registry-derived `oneOf` shape made those assertions stale, and one test panicked converting the missing flat property.
- Workaround: update the tests to inspect the matching `tool` discriminator branch.
- Root cause: the observable schema changed from one open row object to per-registered-tool result branches.

## ST-0207: Focused MCP test found missing batch path description

- Tool: `go test ./internal/operation ./internal/mcp`.
- Target: batch result schema for `semantic_move_construct`.
- Observed failure: the move schema assertion panicked because the registry-derived `destination_file` property omitted the existing requested-target description.
- Workaround: include the established `file` and `destination_file` descriptions in each registered batch result branch.
- Root cause: replacing the static result object schema with per-tool branches dropped the static metadata descriptions.

## ST-0208: Focused MCP suite found nullable-schema expectation and flat receipt branch assumption

- Tool: `go test ./internal/operation ./internal/mcp`.
- Target: `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes` and `TestMCPProjectedReceiptUsesFinalDiskIdentityAndRevision` in `internal/mcp/schema_test.go`.
- Observed failure: the nullable schema uses JSON Schema `type: [string, null]`, not `anyOf`; the receipt test also still treated the per-tool result union as one flat object.
- Workaround: assert the emitted nullable type array and select the receipt branch by its `tool` discriminator.
- Root cause: the test expectations did not match the declared ValueSchema JSON serialization or the new branch-specific result schema.

## ST-0209: Semantic editor has no statement-level update for receipt test setup

- Tool: `semedit-next replace-body` / `insert`.
- Target: nested setup statement in `TestMCPProjectedReceiptUsesFinalDiskIdentityAndRevision` in `internal/mcp/schema_test.go`.
- Observed limitation: adding the planned digest to the existing finalized draft requires inserting one statement between an existing composite literal and call; available semantic operations replace whole function bodies or top-level declarations only.
- Workaround: apply one exact-match atomic source update and validate the public MCP test.
- Root cause: no nested statement edit is registered.

## ST-0210: Public batch test assumed the wrong structured-content nesting

- Tool: `go test ./internal/operation ./internal/mcp`.
- Target: `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes` in `internal/mcp/schema_test.go`.
- Observed failure: actual `semantic_batch` `tools/call` completed, but the test panicked because it looked for `results` directly under the wrong response object.
- Workaround: inspect the batch response wrapper and decode its `structuredContent` shape before checking typed values.
- Root cause: semantic_batch uses a distinct structured response wrapper from ordinary single-tool calls.

## ST-0211: Semantic handler lookup omitted its receiver

- Tool: `semedit-next inspect-symbol`.
- Target: `Server.handleBatch` response wrapper in `internal/mcp/server.go`.
- Observed failure: the unqualified `handleBatch` selector was not resolved in the selected file.
- Workaround: retry with the receiver-qualified `Server.handleBatch` symbol.
- Root cause: the function is a method and the semantic selector requires its qualified owner.

## ST-0212: Semantic handler lookup targeted the wrong file

- Tool: `semedit-next inspect-symbol`.
- Target: `Server.handleBatch`.
- Observed failure: the selected `internal/mcp/server.go` file did not declare the method.
- Workaround: use the declaration outline to locate the method file, then inspect it there.
- Root cause: the batch handler is declared in a separate batch source file.

## ST-0213: Semantic editor cannot change one nested response assertion

- Tool: `semedit-next replace-body` / `insert`.
- Target: `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes` in `internal/mcp/schema_test.go`.
- Observed limitation: correcting two local response-path statements requires statement-level edits inside an existing test body; the registered semantic tools only replace whole function bodies or top-level declarations.
- Workaround: perform the narrow exact-match update atomically, then run the public MCP test.
- Root cause: no nested statement replacement operation is available.

## ST-0214: Injected batch registry omitted the final verification operation

- Tool: `go test ./internal/mcp -run TestBatchToolAdvertisesAndEmitsRegisteredResultShapes`.
- Target: actual batch `tools/call` result assertions in `internal/mcp/schema_test.go`.
- Observed failure: after the five test operations, the batch appended a `semantic_verify` error result because the isolated test registry does not register that post-processing operation.
- Workaround: accept and assert that explicit verification row while checking the typed result variants.
- Root cause: the fixture injects only the providers under test, while batch finalization resolves verification through the same registry.

## ST-0215: Batch test needs a nested synthetic-stage assertion

- Tool: `semedit-next replace-body` / `insert`.
- Target: result loop in `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes` in `internal/mcp/schema_test.go`.
- Observed limitation: the test must account for the batch executor's synthetic `semantic_verify` failure row, which requires a statement inside its existing loop.
- Workaround: make a narrow exact-match atomic update and assert the registered synthetic branch and error status.
- Root cause: the semantic editor cannot change nested statements in an existing declaration.

## ST-0216: Scoped commit could not create the shared worktree index lock

- Tool: `git add` for the first review-fix checkpoint.
- Target: only the owned foundation files in `codex/flow-foundation`.
- Observed failure: Git was denied creation of `/Users/alessandro/sources/semantic-editor/.git/worktrees/flow-foundation/index.lock`; no source files were staged and commit did not run.
- Workaround: retry the user-authorized commit with scoped access to the shared Git metadata.
- Root cause: this linked worktree stores its index under the common repository directory outside the assigned writable root.

## ST-0217: Scaffold purpose header lacked Go comment syntax

- Tool: `semedit-next scaffold-file`.
- Target: new `internal/operation/inspect_projection.go`.
- Observed failure: the scaffold parser rejected a plain-text purpose header as invalid Go syntax; no file was created.
- Workaround: retry with a valid `//` Go comment prefix.
- Root cause: scaffold purpose headers are parsed as source comments.

## ST-0218: Semantic editor has no field or parameter insertion for guarded inspect links

- Tool: `semedit-next insert` / `replace-body`.
- Target: `ReplaceBodyReq`, `replaceBodyParams`, and the owner selector parameter in `internal/operation/wire_engine_mutations.go`.
- Observed limitation: the operation surface has no semantic edit for adding one struct field or changing one parameter element inside an existing declaration; whole-body replacement applies only to functions.
- Workaround: make exact-match atomic structural edits with fsync/rename, then verify through registered operations and public MCP calls.
- Root cause: no struct-field or composite-literal/element insertion operation is registered.

## ST-0219: Atomic structural update refused an ambiguous parameter-line match

- Tool: exact-match atomic Python edit.
- Target: `ReplaceBodyReq`, `replaceBodyParams`, and `replaceConstructParams` in `internal/operation/wire_engine_mutations.go`.
- Observed failure: an assertion found an unexpected match count for a repeated parameter declaration and aborted before writing the file.
- Workaround: use unique declaration-scoped source anchors, verify every replacement count, then atomically rename.
- Root cause: identical Go parameter lines recur in multiple operation declarations in the same file.

## ST-0220: Scoped owner-selector update did not match source text

- Tool: exact-match atomic Python edit.
- Target: `replaceConstructParams` in `internal/operation/wire_engine_mutations.go`.
- Observed failure: the declaration-scoped update found no exact match for the assumed owner-selector literal and aborted before writing any source.
- Workaround: re-inspect the actual declaration and use its current text as the anchor.
- Root cause: the source literal wording differs from the stale copied declaration text.

## ST-0221: Declaration-scoped source replacement found another stale literal

- Tool: exact-match atomic Python edit.
- Target: `ReplaceBodyReq` and parameter declarations in `internal/operation/wire_engine_mutations.go`.
- Observed failure: a declaration-scoped assertion still found an unexpected matching text block and aborted before writing.
- Workaround: copy the complete current declaration from semantic inspection and replace only the declaration as a whole.
- Root cause: nested source text was copied from a different declaration revision and did not match byte-for-byte.

## ST-0222: Semantic insertion diagnostics found incorrect MCP test transport calls

- Tool: `semedit-next insert`.
- Target: `TestInspectProjectionFeedsGuardedReplaceBody` in `internal/mcp/schema_test.go`.
- Observed failure: the inserted test passed a writer as a third argument to `Server.Serve`; diagnostics show that `Serve` accepts a reader and the output writer belongs in `NewServer`.
- Workaround: bind each test response buffer when constructing its server, then call `Serve` with only the request reader.
- Root cause: the inserted test used an outdated transport call shape.

## ST-0223: Body replacement exposed a now-unused test import

- Tool: `semedit-next replace-body`.
- Target: `TestInspectProjectionFeedsGuardedReplaceBody` in `internal/mcp/schema_test.go`.
- Observed diagnostic: the corrected test no longer uses the previous `io.Discard` constructor argument, leaving `io` unused.
- Workaround: run the registered semantic import organizer on the test file.
- Root cause: replacing the test transport setup made one earlier import unnecessary.

## ST-0224: Inspect consumer test exposed legacy-role and response-shape assumptions

- Tool: `go test ./internal/operation ./internal/mcp -run TestInspectProjectionFeedsGuardedReplaceBody`.
- Target: typed SourceFields metadata and inspection response assertions.
- Observed failure: assigning `RoleTarget` to legacy file parameters made their untyped `semantic_lookup.file` source links fail; the public inspection response also did not provide the structured-content wrapper shape assumed by the test.
- Workaround: inspect the current compatibility validator and actual response envelope; keep legacy primitive edges compatible while declaring the new typed edge.
- Root cause: existing primitive continuations and projected MCP output use different compatibility and response paths than the new typed links.

## ST-0225: Initial required make check rejected unformatted inspect projection changes

- Tool: `make check` with the requested local module cache and `GOPROXY=off`.
- Target: current foundation projection and MCP integration changes.
- Observed failure: the repository Go linter reported complexity, constant, exported-comment, unused-helper, and preallocation findings before reaching focused tests.
- Workaround: apply the registered semantic refactors and formatting, then rerun `make check` before focused tests.
- Root cause: newly added integration code had not yet gone through the repository's required lint-fix cycle.

## ST-0226: Worktree-relative semantic CLI path was unavailable

- Tool: shell invocation of `./bin/semedit-next --help`.
- Target: semantic tooling in the foundation worktree.
- Observed failure: the assigned worktree has no `bin/semedit-next` path.
- Workaround: use the root-provided absolute CLI path `/Users/alessandro/sources/semantic-editor/bin/semedit-next` with the worktree as the command working directory.
- Root cause: the CLI binary is shared from the main checkout rather than copied into this worktree.

## ST-0227: Semantic replace-body command rejected an inferred function flag

- Tool: `semedit-next replace-body`.
- Target: `ValidateSourceFields` in `internal/mcp/server.go`.
- Observed failure: the command rejected `--function`; its CLI requires the exact owner-selection flag name from its help output.
- Workaround: inspect command help and retry with the supported flag name before using any fallback.
- Root cause: command-line syntax was inferred from a different semantic tool.

## ST-0228: No registered semantic operation removes obsolete top-level declarations

- Tool: repository semantic CLI command inventory.
- Target: now-unused `batchOutputSchema` and `batchResultSchema` helpers in `internal/mcp/server.go`.
- Observed limitation: available insert/replace operations can add or replace declarations but cannot remove top-level declarations. Keeping the obsolete open-schema helpers fails the repository's unused-code lint.
- Workaround: remove only those exact Go AST declarations with a parser-based, atomic temp-file/fsync/rename edit after recording this limitation.
- Root cause: semantic CLI has no declaration deletion operation.

## ST-0229: Focused public inspection tests found a remaining selector compatibility edge

- Tool: focused Go tests for inspect projection and SourceFields validation.
- Target: `semantic_replace_construct.in_function` and its public inspector consumer test.
- Observed failure: the parameter's existing self-output primitive edge is treated as a typed selector edge and registry validation rejects it; the test then panics while reading the failed server response as success.
- Workaround: remove the role annotation from this legacy selector and keep the new inspection selector path typed under the correct consumer, then make the public test report server errors explicitly.
- Root cause: role metadata applies to every SourceFields path on one parameter, including pre-existing primitive continuation paths.

## ST-0230: Declaration replacement used a nonexistent discriminator CLI constant

- Tool: `semedit-next replace-decl`.
- Target: `replaceConstructParams` in `internal/operation/wire_engine_mutations.go`.
- Observed failure: post-edit diagnostics reported undefined `wireCLIDiscriminator`.
- Workaround: restore the declaration's existing literal CLI name `discriminator` and rerun semantic diagnostics.
- Root cause: the existing declaration used a literal instead of the inferred constant name.

## ST-0231: Semantic inspection used a nonexistent test symbol

- Tool: `semedit-next inspect-symbol`.
- Target: `TestValidateValueSchema` in `internal/operation/value_schema_test.go`.
- Observed failure: the exact test symbol was absent, so inspection refused the request.
- Workaround: use the existing test names from the narrow file outline before selecting a test insertion point.
- Root cause: the requested symbol name was inferred rather than confirmed from the file.

## ST-0232: Insert-function argument retained literal newline escapes

- Tool: `semedit-next insert-func`.
- Target: language-neutral projection coverage in `internal/operation/value_schema_test.go`.
- Observed failure: the shell preserved `\n` as literal backslash characters, which the Go parser rejected.
- Workaround: write the complete declaration into a scratch file and pass its contents as the semantic tool source argument.
- Root cause: multiline code was placed directly in a single-quoted shell argument.

## ST-0233: Test declaration access modifier conflicted with exported casing

- Tool: `semedit-next insert-func`.
- Target: `TestProjectInspectResultKeepsLanguageNeutralAmbiguousMatches` in `internal/operation/value_schema_test.go`.
- Observed failure: the insertion tool rejected explicit private access for the uppercase Go test function name.
- Workaround: declare the test with public access to match its required Go test naming.
- Root cause: test functions use exported casing even though tests are not package API.

## ST-0234: Registered semantic operations cannot add documentation comments to an existing type

- Tool: semantic CLI operation inventory.
- Target: exported `InspectContinuation` and `InspectProjection` types in `internal/operation/inspect_projection.go`.
- Observed limitation: no registered operation changes a type declaration's leading documentation comment; the linter requires both public API comments.
- Workaround: add the two leading comments with a parser-positioned atomic write, then verify the declaration with semantic inspection.
- Root cause: semantic type replacement is not available for struct declarations or their doc comments.

## ST-0235: Follow-up make check still reported static integration findings

- Tool: `make check` with local caches and `GOPROXY=off`.
- Target: inspector schema strings, source-field validation, MCP schema literals, and the public inspection test.
- Observed failure: Go lint still reported one complexity finding, repeated schema-name literals, and one test slice preallocation finding.
- Workaround: extract legacy-source compatibility from the validator, reuse named field/tool constants, and preallocate the test edits slice before rerunning the full check.
- Root cause: the first refactor did not sufficiently lower validator complexity and left lint-only cleanup in the integrated path.

## ST-0236: Semantic inspection targeted a wire constant in the wrong file

- Tool: `semedit-next inspect-symbol`.
- Target: `wireInFunction` in `internal/operation/operation.go`.
- Observed failure: the symbol is not declared in that file.
- Workaround: locate the package constant by a narrow name search, then inspect the actual declaration file.
- Root cause: the parameter constant was assumed to live in the shared operation definition file.

## ST-0237: Semantic source inspection command used an invalid working-directory argument

- Tool: `semedit-next inspect-symbol`.
- Target: shared schema constants in `internal/mcp/server.go`.
- Observed failure: the shell executor could not start because the provided worktree directory did not exist as supplied.
- Workaround: retry from the assigned flow-foundation worktree path exactly.
- Root cause: the command invocation had a malformed working-directory argument.

## ST-0238: Semantic operations cannot replace exact schema string values with named constants

- Tool: registered semantic CLI operations.
- Target: repeated schema property/tool-name literals in `internal/operation` and `internal/mcp`.
- Observed limitation: body/declaration operations can replace whole constructs but cannot rewrite selected string-valued expressions to identifiers without replacing entire unrelated implementations.
- Workaround: apply a narrowly scoped Go AST literal rewrite for exact values only, using atomic temp-file/fsync/rename writes.
- Root cause: semantic CLI lacks expression-level replacement for string literals.

## ST-0239: Second semantic outline request used a malformed worktree path

- Tool: `semedit-next outline`.
- Target: the new MCP inspector test's preallocation lint finding.
- Observed failure: command startup failed because the supplied working-directory path was missing its leading slash.
- Workaround: rerun in the assigned absolute worktree path.
- Root cause: the path was mistyped while constructing the command.

## ST-0240: Semantic operations cannot preallocate a local slice declaration

- Tool: registered semantic CLI operation inventory.
- Target: `edits` local in `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes`.
- Observed limitation: the required capacity change is a local declaration initializer; no available semantic operation replaces one local declaration without rewriting the containing test.
- Workaround: update the exact local declaration by AST position, preserving all other test source.
- Root cause: semantic CLI has no local-declaration initializer operation.

## ST-0241: AST schema-literal rewrite also replaced new constant declarations

- Tool: parser-based AST literal rewrite for repeated schema names.
- Target: new constants in `internal/mcp/server.go`.
- Observed failure: matching literals inside the constant declarations themselves were changed into self-references, producing Go initialization cycles.
- Workaround: restore those three declaration values, then constrain any further AST rewrite to skip constant declaration initializers.
- Root cause: the first rewrite matched by exact literal value without excluding the declaration that establishes each constant.

## ST-0242: AST local-declaration edit retained the `var` keyword

- Tool: parser-positioned local slice preallocation edit.
- Target: `edits` in `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes`.
- Observed failure: the generated source contained `var edits :=`, causing Go syntax and type-check errors during `make check`.
- Workaround: use a valid local declaration initializer (`var edits = make(...)`) through an atomic exact-source correction, then rerun checks.
- Root cause: `ast.ValueSpec.Pos` starts at the variable name while the `var` keyword belongs to the enclosing declaration node.

## ST-0243: Full make check test phase failed without retained failure detail

- Tool: full `make check` (`go test -race -shuffle=on -cover -v ./...`).
- Target: repository-wide verification after inspect projection integration.
- Observed failure: formatting, Go lint, doc generation, and Markdown lint passed; the test target exited nonzero, but the very large verbose output was truncated before the failing package/test was visible. Vale reported warnings/suggestions only, and govulncheck was optional/uninstalled.
- Workaround: rerun the repository test target with output redirected to a worktree-local scratch log, then inspect only failing package/test lines before deciding whether any code change is needed.
- Root cause: verbose test output exceeded tool response retention.

## ST-0244: Failure-log command used zsh's read-only status variable

- Tool: shell wrapper around `make test` output capture.
- Target: `.scratch/make-test-failures.log`.
- Observed failure: after the test command returned, zsh rejected assignment to its reserved read-only `status` variable, so the intended filtered summary did not run.
- Workaround: inspect the completed local log directly and use a task-specific variable name in any wrapper.
- Root cause: a zsh special variable name was reused for the exit code.

## ST-0245: Full suite found an outdated inspection output-schema expectation

- Tool: `go test -race -shuffle=on -cover -v ./...` in the required check target.
- Target: `TestMCPToolsAdvertiseStructuredOutputSchemas` in `internal/mcp/schema_test.go`.
- Observed failure: the inspector now correctly advertises its new typed object result, while the generic test branch still expected a legacy string result.
- Workaround: add an explicit inspector object-schema case that checks the closed projection and its required captured identity fields.
- Root cause: schema behavior changed intentionally, but the existing broad assertion table did not gain a case for the inspector.

## ST-0246: Insert-case argument retained literal newline escapes

- Tool: `semedit-next insert-case`.
- Target: explicit `semantic_inspect_symbol` schema test branch.
- Observed failure: the shell passed literal backslash-n sequences and Go parsing rejected the case clause.
- Workaround: provide the clause from a worktree-local scratch file.
- Root cause: multiline syntax was supplied as a single-quoted shell argument.

## ST-0247: Inspector schema test assumed projection fields were top-level

- Tool: focused MCP schema tests after adding the inspector object-schema branch.
- Target: `TestMCPToolsAdvertiseStructuredOutputSchemas` in `internal/mcp/schema_test.go`.
- Observed failure: the standard output schema wraps the typed operation value under `result`; the test looked for `inspection` directly under top-level output properties and panicked on the missing continuation schema.
- Workaround: assert the `inspection` and `continuations` properties under `outputSchema.properties.result`, and guard each type assertion with a test failure.
- Root cause: standard output envelope fields were conflated with the operation's projected value fields.

## ST-0248: Replace-construct could not select a switch case by switch expression

- Tool: `semedit-next replace-construct`.
- Target: the `semantic_inspect_symbol` case in `TestMCPToolsAdvertiseStructuredOutputSchemas`.
- Observed failure: the Go backend searched for a case whose literal value was `name`; it did not treat `name` as the switch selector and reported no matching construct.
- Workaround: replace only the matching `CaseClause` in the Go AST using an atomic parser-positioned edit.
- Root cause: replace-construct case selection currently interprets the discriminator as the case value for this syntax.

## ST-0249: Final make check found the expanded schema test exceeded complexity limits

- Tool: final `make check` Go lint-fix stage.
- Target: `TestMCPToolsAdvertiseStructuredOutputSchemas` in `internal/mcp/schema_test.go`.
- Observed failure: adding explicit nested inspector result-schema assertions raised test cognitive complexity above the repository threshold.
- Workaround: move those assertions into a focused helper and keep the test switch branch as a single helper call.
- Root cause: schema branch assertions were added inline to an already broad catalog test.

## ST-0250: Replace-construct does not locate switch clauses in the test catalog function

- Tool: `semedit-next replace-construct`.
- Target: inspector result-schema case in `TestMCPToolsAdvertiseStructuredOutputSchemas`.
- Observed failure: both the switch expression and explicit case label selectors returned no matching case for this test's catalog switch.
- Workaround: replace only the matching case-clause AST node with a helper call, preserving the surrounding switch and test.
- Root cause: current Go construct selection does not expose a usable selector for this case clause.

## ST-0251: Git staging could not write linked-worktree index metadata

- Tool: `git add` and `git commit` for the requested inspect-projection checkpoint.
- Target: this flow-foundation worktree index.
- Observed failure: Git could not create the linked worktree's `index.lock` under the main repository `.git/worktrees/flow-foundation` metadata path because that location is outside the writable worktree root.
- Workaround: retry the exact staging and commit operation with the already-authorized elevated filesystem access for linked-worktree metadata.
- Root cause: linked worktree source files are writable, but Git stores the shared worktree index outside the worktree directory.

## ST-0259: Semantic lookup resolved against the primary checkout

- Tool: `semantic_lookup` through the connected MCP server.
- Target: `internal/operation/wire_contracts.go` in the assigned `flow-contracts` worktree.
- Observed failure: the server looked for the file under the primary checkout, where this branch-only file does not exist.
- Workaround: use the assigned worktree's checked CLI for semantic navigation and edits; the connected MCP server is attached to a different checkout.
- Root cause: the shared MCP server does not target this worker's isolated worktree.

## ST-0252: Absolute patch path targeted the coordinator checkout

- Tool: built-in `apply_patch` used for a Go backend edit.
- Target: `internal/backend/golang/backend.go` while working in `.scratch/worktrees/flow-foundation`.
- Observed failure: an absolute path omitted `.scratch/worktrees/flow-foundation` and targeted the coordinator checkout instead of this worker worktree; the coordinator checkout received the same one-line context-forwarding edit.
- Workaround: all intended changes were applied and verified in the assigned worker worktree; root will preserve the coordinator mirror in a targeted stash before integration, and I did not modify or restore coordinator files.
- Root cause: the patch request used the primary checkout path rather than the assigned worktree path. Verify every patch target against the assigned worktree root before editing.

## ST-0253: Replace-body rejected escaped test newline literals

- Tool: `semedit replace-body`.
- Target: `TestInspectProjectionFeedsGuardedReplaceBody` in `internal/mcp/schema_test.go`.
- Observed failure: the CLI parser rejected escaped newline literals in the replacement body; it made no source change.
- Workaround: use a surgical patch to add the self-contained module manifest to this test fixture only.
- Root cause: the body flag's layered shell and CLI quoting did not preserve the Go rune and string literals.

## ST-0254: Inspection rejected package-qualified helper selector

- Tool: `semantic_inspect_symbol`.
- Target: `internal/sourcebuf/buffer.go`, selector `sourcebuf.ReadFile`.
- Observed failure: symbol resolution returned `symbol not found`; no mutation occurred.
- Workaround: use `semantic_lookup` to obtain the canonical declaration selector before inspection.
- Root cause: the coordinator constructed a package-qualified selector instead of copying a resolved symbol identity.

### ST-0169: semantic insert accepted invalid Go source

- Date: 2026-10-01
- Tool: `semantic_insert_declaration` via worktree-local `semedit insert`
- Target: `internal/astedit/insert_field_plan.go`
- Observed failure: insertion reported success and returned introduced compiler diagnostics for `ast.StructType.Rbrace` and a malformed `append` call.
- Workaround: record the tool behavior and attempt `semedit replace-body`; the local CLI could not start because the same invalid source prevented its Go package from building, so the three reported defects were corrected with an atomic source update before resuming semantic edits.
- Root cause: the inserted Go snippet passed parsing but the semantic insertion's post-edit diagnostics exposed type-check errors; the planner assumed brace ownership on `*ast.StructType` and emitted an invalid append expression.

### ST-0170: semantic insertion exposed missing planner proof gates

- Date: 2026-10-01
- Tool: `semantic_insert_declaration` via worktree-local `semedit insert`
- Target: `internal/astedit/insert_statement_plan.go`
- Observed failure: insertion reported success but diagnostics identified undefined `ProofBlockBinding` and `ProofControlFlowSet` referenced by the new statement planner.
- Workaround: add the missing named proof requirements to the shared pure-plan contract before continuing.
- Root cause: the planner function was inserted before its integration proof vocabulary was added to the shared plan file; the tool surfaced the package type-check failure after writing.

### ST-0171: semantic insertion exposed helper return-arity errors

- Date: 2026-10-01
- Tool: `semantic_insert_declaration` via worktree-local `semedit insert`
- Target: `internal/astedit/insert_field_plan.go`
- Observed failure: insertion reported success but diagnostics showed two return statements supplied `(int, string)` to a helper declared as `(int, string, error)`.
- Workaround: record the diagnostics and add explicit nil error values to the successful offset returns.
- Root cause: the extracted helper reused byte-range functions with shorter return tuples without adapting those results to its error-bearing signature.

### ST-0172: semantic insertion surfaced an incomplete helper sequence

- Date: 2026-10-01
- Tool: `semantic_insert_declaration` via worktree-local `semedit insert`
- Target: `internal/astedit/insert_element_plan.go`
- Observed failure: the validator insertion reported success but package diagnostics showed `validateTypedMapKey` undefined because the helper was queued for the next insertion.
- Workaround: record the package diagnostic and add the referenced pure helper atomically so the local CLI can build again.
- Root cause: semantic insertion type-checks the package immediately; helper declarations must be added before callers that reference them.

### ST-0173: semantic body replacement exposed an incomplete planner split

- Date: 2026-10-01
- Tool: `semantic_replace_body` via worktree-local `semedit replace-body`
- Target: `internal/astedit/insert_element_plan.go`
- Observed failure: the planner replacement reported success but diagnostics showed the extracted `appendCompositeElement` helper was not present yet and `bytes` had become temporarily unused.
- Workaround: record the diagnostics and add the pure append helper before retrying further semantic edits.
- Root cause: the orchestration body was switched to the planned helper before that helper was inserted; post-edit verification ran against an incomplete intermediate package state.

### ST-0174: semantic_insert_type rejected escaped source newlines

- Date: 2026-10-01
- Tool: `semantic_insert_type` via worktree-local `semedit insert-type`
- Target: `internal/astedit/plan.go`
- Observed failure: the tool returned `no declarations found in snippet`; no source changed.
- Workaround: retry with an actual multiline type declaration.
- Root cause: the shell argument contained literal backslash-n sequences where the Go parser expected source line breaks.

### ST-0175: patch context mismatch left structural planner unchanged

- Date: 2026-10-01
- Tool: `apply_patch`
- Target: `internal/astedit/doc_comment_plan.go` and `internal/astedit/insert_field_plan.go`
- Observed failure: the combined patch could not find the expected `validateInsertedField` context, so none of its changes were applied.
- Workaround: record the failure and retry with exact, narrower contexts after inspecting the current source.
- Root cause: the patch assumed the identifier validation was adjacent to the parser block, but it occurs in a separate validator function.

### ST-0176: combined review-fix patch used stale statement context

- Date: 2026-10-01
- Tool: `apply_patch`
- Target: `internal/astedit/doc_comment_plan.go`, `internal/astedit/insert_statement_plan.go`, and `internal/astedit/insert_element_plan.go`
- Observed failure: the combined patch could not match the expected statement-anchor branch, so none of its hunks were applied.
- Workaround: record the failure and split the changes into smaller patches based on freshly inspected context.
- Root cause: the statement planner has separate selection branches and did not contain the assumed `anchor == nil` guard at that location.

### ST-0177: make check rejected malformed structural planner regression test

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: `internal/astedit/structural_plan_test.go`
- Observed failure: the Go lint/typecheck pass found syntax errors from mismatched test closure delimiters; no focused test ran.
- Workaround: inspect the formatted file around the reported lines and repair the test structure before rerunning the required check.
- Root cause: regression cases were inserted into an existing nested `t.Run` block with incorrect brace placement.

### ST-0178: make check rejected planner test helper arity

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: `internal/astedit/structural_plan_test.go`
- Observed failure: Go typecheck found an assignment arity mismatch when the injection regression captured two results from `parseKeyedElement`, which returns three.
- Workaround: capture all return values in the regression before rerunning repository checks.
- Root cause: the test call did not match the planner helper's `(node, key source, error)` contract.

### ST-0179: make check complexity lint rejected comment planner and test aggregator

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: `internal/astedit/doc_comment_plan.go` and `internal/astedit/structural_plan_test.go`
- Observed failure: gocognit reported complexity 90 for `PlanSetDocComment` and 104 for `TestStructuralSourcePlanners`.
- Workaround: record the failure and extract declaration selection/comment checks and focused tests into named helpers/subtests.
- Root cause: the new ownership guards and regressions pushed already dense functions beyond the repository's configured complexity ceiling.

### ST-0180: repository race/shuffle suite returned failure during make check

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: repository-wide `go test -race -shuffle=on -cover -v ./...`
- Observed failure: the race/shuffle test stage returned non-zero twice; output truncation hid the failing test/package, while formatter, Go lint, Markdown lint, Vale, and Hugo completed.
- Workaround: capture the exact test command output in `.scratch/go/structural-check-tests.log` and inspect failure markers before attributing the cause.
- Root cause: `t.TempDir()` created paths under `/var/folders/...`, a symlink alias of `/private/var/folders/...`; source-buffer path validation correctly rejected those aliases. Re-running `internal/sourcebuf`, `internal/projectverify`, and `internal/mcp` with canonical `TMPDIR=/private/var/folders/wd/twmctgm50s9b4x8mhmfj4_b40000gn/T` passed.

### ST-0181: Git staging was blocked by the isolated filesystem root

- Date: 2026-10-01
- Tool: `git add`
- Target: assigned worktree Git index at `/Users/alessandro/sources/semantic-editor/.git/worktrees/flow-structural/index`
- Observed failure: sandbox denied creation of `index.lock`; source edits remain in the assigned writable worktree and no index mutation occurred.
- Workaround: request a narrow sandbox escalation for explicit owned-path staging and the authorized milestone commit.
- Root cause: this linked worktree's Git metadata is stored outside the configured writable root.

### ST-0182: CRLF range patch addressed coordinator checkout

- Date: 2026-10-01
- Tool: `apply_patch`
- Target: `internal/astedit/doc_comment_plan.go`
- Observed failure: patch application could not find the requested absolute path because it omitted this worker's `.scratch/worktrees/flow-structural` path segment; no edit occurred.
- Workaround: record the error and reapply against the assigned worktree path.
- Root cause: copied coordinator repository path was mistaken for the worker worktree path.

### ST-0183: runtime pattern lookup used absent paths

- Date: 2026-10-01
- Tool: `rg`/`sed`/`git show` source inspection
- Target: `internal/operation/wire_contracts.go` and `internal/operation/default.go`
- Observed failure: the queried paths do not exist in the current integration branch; no files were changed.
- Workaround: enumerate registered source files and inspect the contract worker's actual worktree/branch paths before using that design.
- Root cause: filenames were assumed from the handover summary rather than confirmed in the checked tree.

### ST-0184: research document lookup used assumed filenames

- Date: 2026-10-01
- Tool: `sed`
- Target: `docs/research/RQ-0056-insert-field.md`, `RQ-0057-insert-statement.md`, `RQ-0058-insert-composite-element.md`, `RQ-0059-set-doc-comment.md`, and `RQ-0048-encapsulate-field.md`
- Observed failure: those guessed paths do not exist; repository RQ filenames include the `go-` prefix. No files were changed.
- Workaround: enumerate `docs/research` and locate the exact indexed filenames before reading.
- Root cause: filenames were inferred from operation names rather than the research index.

### ST-0185: repository make check failed in benchmark-harness package tests

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: `tools/benchmark-harness` package tests
- Observed failure: formatting, linting, documentation generation, and other parallel test packages completed, but the race/shuffle test stage reported `FAIL semedit/tools/benchmark-harness`; output truncation hid the failing test detail.
- Workaround: preserve this result, then rerun that package with output captured under `.scratch/` to identify the failing test before continuing.
- Root cause: not yet determined; test output from the initial run was truncated by the command transport.

### ST-0186: focused test retry used the default protected Go build cache

- Date: 2026-10-01
- Tool: `go test -race -shuffle=on -cover ./tools/benchmark-harness`
- Target: `tools/benchmark-harness`
- Observed failure: package setup could not open a build-cache artifact under `~/Library/Caches/go-build` because the sandbox denied access. No test ran.
- Workaround: configure `GOCACHE` under this worktree's ignored `.scratch/go/` directory for subsequent Go commands.
- Root cause: the focused retry set `GOMODCACHE` but omitted `GOCACHE`, so Go selected the user's default cache outside writable roots.

### ST-0187: worktree-local semantic CLI binary was absent

- Date: 2026-10-01
- Tool: `./bin/semedit-next --help`
- Target: assigned `flow-structural` worktree
- Observed failure: the expected `bin/semedit-next` path does not exist, so the CLI did not run and no files changed.
- Workaround: locate the instructed `ROOT/bin/semedit-next` within this worktree before dogfooding; do not use a binary rooted at main.
- Root cause: this worktree has not built or linked the semantic CLI under `bin/`.

### ST-0188: semantic scaffold purpose header omitted Go comment syntax

- Date: 2026-10-01
- Tool: `semedit scaffold-file`
- Target: `internal/operation/wire_structural_plan.go`
- Observed failure: the CLI rejected the purpose header as invalid Go syntax and did not create the file.
- Workaround: retry with a Go comment line as the purpose header.
- Root cause: the command validates the header as source comment text, while the first request supplied plain prose.

### ST-0189: semantic CLI command used coordinator working directory

- Date: 2026-10-01
- Tool: `semedit insert`
- Target: `internal/operation/wire_structural_snapshot.go`
- Observed failure: command resolution used the coordinator checkout, where the worker-created target file does not exist; the CLI failed while reading and made no write.
- Workaround: rerun from the assigned `flow-structural` worktree and verify the resolved workspace in the command result.
- Root cause: the command's working directory argument named the coordinator checkout instead of the assigned worktree.

### ST-0190: semantic declaration insertion left required imports unresolved

- Date: 2026-10-01
- Tool: `semedit insert`
- Target: `internal/operation/wire_structural_snapshot.go`
- Observed failure: the declaration was inserted, but the file lacks imports for its referenced standard and internal package types; the CLI returned no status text.
- Workaround: use the registered import organizer with explicit paths, then inspect the resulting file before further edits.
- Root cause: declaration insertion did not infer imports for type names in this declaration.

### ST-0191: semantic declaration insertion published unresolved import references

- Date: 2026-10-01
- Tool: `semedit insert`
- Target: `internal/operation/wire_structural_snapshot.go`
- Observed failure: the CLI inserted the loader and returned success while reporting 14 new undefined-name diagnostics because its import cleanup was explicitly disabled for this insertion.
- Workaround: record the incorrect status and add the exact standard-library imports with the registered import operation before continuing.
- Root cause: the request set `auto_organize_imports=false`, so inserted AST references were not accompanied by required imports; the CLI result still allowed the syntactic mutation.

### ST-0192: structural registry insertion preceded its parameter helpers

- Date: 2026-10-01
- Tool: `semedit insert`
- Target: `internal/operation/wire_structural_plan.go`
- Observed failure: after inserting the registry helper, semantic verification reported unresolved references to the intentionally not-yet-inserted `stringParam` and `structuralRevision` helpers, along with imports that were then added by the registered import organizer.
- Workaround: record the staged declaration state and add the shared helpers through semantic insertion before registering operation definitions.
- Root cause: the new registry helper was inserted before its declaration dependencies, and automatic import organization was disabled for that insertion.

### ST-0193: repository check stopped during Go autofixes on goconst findings

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: Go lint autofix stage
- Observed failure: lint fix returned four `goconst` findings: two in foundation-owned `internal/operation/typed_descriptors.go`, one in existing `wire_engine_mutations.go`, and one in this worker's structural example. The check stopped before tests.
- Workaround: replace the worker-owned repeated literal with the existing shared parameter constant, then report the two foundation-owned and one unrelated finding for root coordination before retrying checks.
- Root cause: the integrated lint configuration flags repeated string literals across files, including code owned by another track.

### ST-0195: tool orchestration script had a JavaScript syntax error

- Date: 2026-10-01
- Tool: `functions.exec`
- Target: repository inspection command
- Observed failure: the orchestration script failed to parse because a call expression lacked its closing delimiter; no nested command ran and no files changed.
- Workaround: correct the script delimiters and rerun the bounded inspection.
- Root cause: a mismatched closing parenthesis in the tool orchestration source.

### ST-0196: structural CLI behavior tests exposed simulated-output snapshot comparison

- Date: 2026-10-01
- Tool: focused `go test` for `set_doc_comment` and `insert_statement` txtar contracts
- Target: `internal/operation/wire_structural_snapshot.go`
- Observed failure: both handlers rejected their planned outputs because the typecheck loader compared the simulated selected-file overlay with the buffer's preimage bytes. No source mutation was published.
- Workaround: distinguish the intentional `extra` output overlay from captured read-set preimages: still read the original through `sourcebuf` to bind it, but compare loader bytes only for paths not overridden by the simulated output.
- Root cause: the loader conflated the candidate typecheck snapshot with the source-buffer preimage snapshot.

### ST-0197: focused public-operation fixtures exposed test-tree and selector mistakes

- Date: 2026-10-01
- Tool: focused `go test` for the two new txtar contracts
- Target: `testdata/scripts/set_doc_comment.txtar` and `insert_statement.txtar`
- Observed failure: the doc-comment output preflight discovered duplicate declarations because the fixture's `want/` Go snapshots formed extra packages; the malformed-statement case used unqualified `Run` instead of the live receiver-qualified selector. No unexpected source mutation occurred.
- Workaround: move expected source snapshots under ignored `.scratch/want/` paths and use `(*Server).Run` in the malformed-source case.
- Root cause: package-wide overlay typechecking correctly included fixture snapshot directories, and the test selector did not match the method's receiver identity.

### ST-0198: focused txtar cases exposed blank CLI value omission and stale diagnostic wording

- Date: 2026-10-01
- Tool: focused `go test` for the new public-operation txtar contracts
- Target: CLI raw-parameter construction and `testdata/scripts/insert_statement.txtar`
- Observed failure: `--comment ''` was omitted before schema validation, so explicit-empty removal was reported as missing; the malformed-statement fixture expected an older diagnostic phrase.
- Workaround: preserve explicitly changed empty string flags in CLI raw parameters and assert the planner's current precise diagnostic.
- Root cause: CLI ingress drops blank string values without checking the Cobra flag's `Changed` state; test wording did not match the new refusal message.

### ST-0199: repository check failed during parallel race/shuffle tests

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: repository-wide `go test -race -shuffle=on -cover -v ./...` stage
- Observed failure: the benchmark-harness package passed, while another package returned non-zero; the large concurrent output was truncated before the failing test and package could be identified. Formatting, lint, docs generation, and build completed.
- Workaround: capture a targeted repeat of the new CLI txtar tests and exact package output in `.scratch/` before deciding whether the failure belongs to this track or is unrelated contention.
- Root cause: not yet identified from the initial output.

### ST-0200: focused public contracts found two txtar expectation errors

- Date: 2026-10-01
- Tool: focused `go test` for `set_doc_comment` and `insert_statement` txtar contracts
- Target: the two new structural operation fixtures
- Observed failure: empty-comment removal now passes; the statement path refusal includes the precise phrase `construct_path requires its matching expected_revision`, while the fixture expected a shorter nonmatching phrase. The directive case selected `Keep` in `doc.go` even though it lives in `directives.go`.
- Workaround: match the precise path-precondition diagnostic and select `directives.go` for the directive case.
- Root cause: fixture assertions had not been checked against the current exact error text and source-file ownership.

### ST-0201: structural typed-proof insertion required explicit go/types import

- Date: 2026-10-01
- Tool: `semedit insert`
- Target: `internal/operation/wire_structural_snapshot.go`
- Observed failure: the field-proof declaration referenced `go/types` while automatic import organization was disabled; semantic diagnostics reported five unresolved `types` names.
- Workaround: record the report and add `go/types` with the registered import organizer.
- Root cause: type-analysis imports were intentionally deferred until the complete helper declaration was inserted.

### ST-0202: repository check rejected the first field proof helper

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: `internal/operation/wire_structural_snapshot.go`
- Observed failure: Go lint reported cognitive complexity 101 for `validatePrivateStructuralType` and an `intrange` suggestion for iterating struct fields; the check stopped before tests.
- Workaround: split owner resolution and literal/source scanning into named helpers, and use the configured integer-range form.
- Root cause: the initial typed-object, method-set, alias, and literal proof were combined in one helper.

### ST-0203: repository contract test requires explicit tools/list assertions for new structural MCP operations

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: `testdata/scripts/mcp_server.txtar`
- Observed failure: `TestREQ002REQ006PublicRegistryAndCLIContractCoverage` reports that `semantic_insert_field`, `semantic_insert_statement`, and `semantic_set_doc_comment` have no `tools/list` txtar assertions.
- Workaround: add explicit public MCP listing assertions for the newly registered tools, then rerun the required check.
- Root cause: registry contract tests require every registered public MCP tool to be represented in an observable CLI fixture.

### ST-0204: research index inspection used an invalid worktree path

- Date: 2026-10-01
- Tool: `functions.exec` with `exec_command`
- Target: research index inspection
- Observed failure: the repository read command could not start because its `workdir` argument named a nonexistent path.
- Workaround: retry the read from the assigned `flow-structural` worktree.
- Root cause: the command invocation contained a stale/incorrect worktree path.

### ST-0205: operation contract inspection used an invalid worktree path

- Date: 2026-10-01
- Tool: `functions.exec` with `exec_command`
- Target: operation result contract inspection
- Observed failure: the source read could not start because its `workdir` argument named a nonexistent path.
- Workaround: rerun the read from the assigned `flow-structural` worktree.
- Root cause: the command invocation contained a stale/incorrect worktree path.

### ST-0206: focused insert-field txtar stopped on an unterminated assertion

- Date: 2026-10-01
- Tool: focused `go test` for structural CLI txtar contracts
- Target: `testdata/scripts/insert_field.txtar`
- Observed failure: the insert-field duplicate-refusal block returned the expected diagnostic and preserved the file, then the txtar parser rejected a later command with `unterminated quoted argument`.
- Workaround: inspect the exact fixture lines and repair its command quoting before rerunning the focused contract.
- Root cause: the fixture used a shell quote that was not closed in the txtar command language.

### ST-0207: semantic scaffold-file rejected a free-form purpose header

- Date: 2026-10-01
- Tool: `semedit scaffold-file`
- Target: `internal/operation/wire_structural_composite.go`
- Observed failure: the operation rejected the requested prose as a purpose header because it requires Go-comment syntax and treated `This` as invalid Go source. No file was created.
- Workaround: record the tool error and retry with the required `//` comment prefix.
- Root cause: scaffold-file validates the purpose header as source text but the request omitted its Go comment delimiter.

### ST-0208: semantic batch used the wrong registry function target

- Date: 2026-10-01
- Tool: `semantic_batch` through the local trusted MCP helper
- Target: `internal/operation/wire_engine.go`
- Observed failure: the four composite helper functions were inserted in the new file, but the final `semantic_replace_body` child could not find `registerStructuralPlanOps` in `wire_engine.go`; the function is in `wire_structural_plan.go`.
- Workaround: inspect the actual inserted bytes and current diff, then target the owning file in a follow-up registry update.
- Root cause: registry helper location was assumed from the call site rather than checked at its declaration.

### ST-0209: semantic batch applied composite wiring but final configured lint verification failed

- Date: 2026-10-01
- Tool: `semantic_batch` through the local trusted MCP helper
- Target: `internal/operation/wire_structural_composite.go` and `internal/operation/wire_structural_plan.go`
- Observed failure: both mutation children returned success, but the final configured `golangci-lint` verification returned non-zero after the registry update and import organization.
- Workaround: record the tool failure and inspect the actual files plus diagnostics before making any follow-up edits.
- Root cause: pending until the exact lint diagnostics are inspected.

### ST-0210: composite key identity used a non-existent go/constant formatter

- Date: 2026-10-01
- Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
- Target: `internal/operation/wire_structural_composite.go`
- Observed failure: doc generation could not compile the source because `constant.String` resolves to the `constant.String` kind constant, not a formatting function.
- Workaround: record the failure, then change constant value identity formatting to the `go/constant.Value.String` method through a semantic declaration edit.
- Root cause: the go/constant API formatter was misremembered.

### ST-0211: composite helper batch still failed its final configured lint hook

- Date: 2026-10-01
- Tool: `semantic_batch` through the local trusted MCP helper
- Target: `internal/operation/wire_structural_composite.go`
- Observed failure: the function-body replacement and import-organizer children succeeded, but the final `golangci-lint` verification returned non-zero again.
- Workaround: inspect the current source, then capture the repository linter output to identify the exact remaining finding before retrying semantic edits.
- Root cause: pending direct lint diagnostics.

### ST-0212: expected import absence made a combined inspection command nonzero

- Date: 2026-10-01
- Tool: `rg` within `functions.exec`
- Target: `internal/operation/wire_structural_composite.go`
- Observed failure: the command used `rg -n constant\.` to verify that a removed import had no remaining uses; the expected no-match result exited with status 1 and short-circuited the shell chain.
- Workaround: the preceding source excerpt confirmed the import list; use an explicit `if rg ...; then` check when absence is expected.
- Root cause: `rg` reports no matches with a non-zero exit status.

### ST-0213: full check found worker goconst and iteration issues plus pre-existing cross-track lint findings

- Date: 2026-10-01
- Tool: make check with required local Go cache variables
- Target: internal/operation/wire_structural_composite.go, foundation typed_descriptors.go, and existing wire_engine_mutations.go
- Observed failure: golangci-lint reports the worker repeated discriminator literal and integer loop, plus goconst findings in two foundation-owned occurrences and one unrelated existing mutation example.
- Workaround: fix only this worker shared discriminator literal and integer-range loop, retain the other three findings for root coordination, then rerun checks after coordination.
- Root cause: new worker code duplicated a canonical parameter name and used a legacy loop; independent cross-track source already has repeated literals.

### ST-0214: shell quoting interrupted an attempted failure-log append

- Date: 2026-10-01
- Tool: functions.exec with exec_command
- Target: docs/SUBOPTIMAL_TOOLS.md
- Observed failure: unescaped Markdown backticks in the single-quoted shell argument triggered command substitution and truncated the embedded Python source. No log entry was written by that attempt.
- Workaround: record the failure without shell-significant backticks, then continue with the bounded semantic edit.
- Root cause: shell quoting treated Markdown delimiters as executable substitutions.

### ST-0215: construct implementation inspection guessed a nonexistent file path

- Date: 2026-10-01
- Tool: rg through functions.exec
- Target: internal/astedit/replace_construct.go
- Observed failure: the search command named a nonexistent source file and returned a non-zero status.
- Workaround: discover the matching tracked file with rg --files before inspecting construct selectors.
- Root cause: guessed implementation filename was not verified.

### ST-0216: construct selector inspection used an invalid worktree path

- Date: 2026-10-01
- Tool: functions.exec with exec_command
- Target: internal/astedit/construct.go
- Observed failure: the read command could not start because the workdir named a nonexistent checkout.
- Workaround: retry against the assigned flow-structural worktree.
- Root cause: a stale path was copied into the command.

### ST-0217: semantic replace-construct found the loop had already been modernized

- Date: 2026-10-01
- Tool: semantic_batch through the local trusted MCP helper
- Target: internal/operation/wire_structural_composite.go
- Observed failure: the selected loop text was not found; the preceding make check go fix had already converted it to a range loop. The later batch child did not run.
- Workaround: record the actual state and retry only the remaining discriminator-literal cleanup using a focused semantic function-body operation.
- Root cause: a mechanical fixer changed the source between selection discovery and the later operation.

### ST-0218: composite discriminator cleanup passed its child but final lint still refused

- Date: 2026-10-01
- Tool: semantic_batch through the local trusted MCP helper
- Target: internal/operation/wire_structural_composite.go
- Observed failure: semantic_replace_body changed the repeated discriminator literals to the shared parameter constant, then final configured lint verification returned non-zero because remaining diagnostics are outside this worker scope.
- Workaround: record the result and retain the applied source change; coordinate the known unrelated lint findings before requesting another successful MCP verification.
- Root cause: full-worktree golangci-lint includes unresolved cross-track goconst findings.

### ST-0219: fixture generator quoting failed before writing the composite txtar

- Date: 2026-10-01
- Tool: functions.exec with exec_command
- Target: testdata/scripts/insert_composite_element.txtar
- Observed failure: an embedded JSON string lost its source quotes when passed through nested shell and Python quoting, producing a Python syntax error before the final fixture file was mutated.
- Workaround: write the fixture content to a temporary file with a single-quoted here-document, then atomically rename it into testdata/scripts.
- Root cause: nested shell quoting was used where a literal here-document is simpler and preserves argument boundaries.

### ST-0220: make-check log search treated the pattern as an option

- Date: 2026-10-01
- Tool: rg through functions.exec
- Target: .scratch/make-check-pre-composite-tests.log
- Observed failure: rg treated the pattern beginning with three dashes as an option and returned a usage error instead of searching.
- Workaround: record the error, then pass the expression with rg -e.
- Root cause: option-like pattern text was not introduced with -e.

### ST-0221: structural parameter copy edges referenced producers without output metadata

- Date: 2026-10-01
- Tool: make check with required local Go cache variables
- Target: internal/operation/wire_structural_plan.go and wire_structural_composite.go
- Observed failure: MCP and registry tests reject the new parameter SourceFields edges because providers such as semantic_lookup.file have no typed output metadata yet. The foundation projection/finalization hook integration is pending.
- Workaround: record the failure and remove unsupported copy edges from this checkpoint; restore only edges backed by the checked output projection API after root rebase.
- Root cause: operation inputs advertised continuations before the providers exposed a typed live result contract.

### ST-0222: semantic lookup tool resolved against the coordinator checkout

- Date: 2026-10-02
- Tool: `semantic_lookup`
- Target: `internal/astedit/doc_comment_plan.go`
- Observed failure: the MCP server resolved the workspace to the coordinator checkout, where this worker's new planner file does not exist. No source was changed.
- Workaround: use the assigned worktree's local semantic CLI for dogfooding, and verify the resolved workspace path before edits.
- Root cause: the shared MCP server is bound to the coordinator checkout rather than this worker worktree.

### ST-0223: insert-statement fixture expected a rejected CLI option

- Date: 2026-10-02
- Tool: `make check` txtar contract test
- Target: `testdata/scripts/insert_statement.txtar`
- Observed failure: the last refusal case supplied unsupported `--construct-path`; the CLI correctly rejected it as an unknown flag, so the fixture's expected-revision diagnostic did not match. No file changed.
- Workaround: replace that case with a stale `--expected-revision` precondition and assert the planner's current-revision refusal.
- Root cause: the fixture retained an earlier unsupported-path scenario after the operation contract removed that parameter.

### ST-0224: semantic verify reported diagnostics outside its selected file

- Date: 2026-10-02
- Tool: `semantic_verify` through the local CLI
- Target: `internal/operation/wire_structural_plan.go`
- Observed failure: a check-only verification scoped to this file returned 16 type diagnostics in unrelated backend and AST-edit files, while its configured linter passed and the full race/shuffle Go test suite passed. No files changed.
- Workaround: use the successful repository check and focused Go tests as the verification evidence for this checkpoint.
- Root cause: not yet determined; the verifier reported module diagnostics outside the selected file despite a successful full test build.

## ST-0257: Structural insertion used the default external Go cache

- Tool: worktree-local MCP `semantic_batch`, child `semantic_insert_statement`.
- Target: `internal/mcp/server_test.go`, owner `TestMCPSnapshotAndUndo`.
- Observed failure: structural package loading tried the default `~/Library/Caches/go-build` and was denied by the sandbox. The earlier helper insertion and schema-fixture body change applied; the statement child made no change.
- Workaround: retain the applied earlier edits and retry only the statement with the configured writable Go cache. Send the loader environment finding to the structural owner before merging the public tool.
- Root cause: `loadStructuralPackages` omitted the managed Go environment used by other compiler-snapshot operations.

## ST-0258: Statement insertion rejects an owner containing unrelated control flow

- Tool: worktree-local MCP `semantic_insert_statement`.
- Target: `internal/mcp/server_test.go`, owner `TestMCPSnapshotAndUndo`, direct `calcFile := ...` anchor.
- Observed failure: the operation refused the entire owner because it contains later conditional statements; no source change occurred in this retry.
- Workaround: use semantic body replacement for the fixture's additional module initialization, retaining its existing assertions. Clarify the advertised straight-line-owner restriction and review whether unrelated control flow needs exclusion.
- Root cause: the initial statement planner rejects control-flow nodes anywhere in the owning function, beyond the selected direct statement.

## ST-0259: Independent contract preflight review timed out

- Tool: `codex_ask`, read-only Terra review.
- Target: the contract worker's test-aware candidate preflight and public test fixtures.
- Observed failure: the tool returned `context deadline exceeded` without a review report. No approval or findings were produced.
- Workaround: retain root's concrete source review and public success/refusal evidence; request a narrower follow-up review if semantic questions remain. Do not treat the timeout as approval.
- Root cause: the review exceeded the tool's request deadline; the underlying cause is unconfirmed.

## Semantic declaration insertion omitted exported API comments

- **Tool:** `semantic_insert_declaration` / `semantic_insert_function`
- **Target:** `internal/astedit/inline_function_plan.go`, `internal/astedit/type_migration_plan.go`
- **Observed behavior:** The tools inserted the requested declarations and organized imports, but the initial declaration sources omitted Go doc comments. The repository linter then rejected the exported planner names.
- **Workaround:** The semantic tool set has no operation for adding only leading declaration comments, so the required doc comments were added manually after logging this occurrence; then the project checks were rerun.
- **Root cause:** The insertion operations validate declaration structure and imports but do not require Go documentation comments on exported declarations.

## Relocation worker: outline rejected `--file`

- Tool: `semedit outline`.
- Target: read-only outlines of `internal/astedit/inline_function_plan.go`, `internal/astedit/type_migration_plan.go`, and `internal/operation/wire_engine_inline.go`.
- Observed failure: the command rejected `--file` as an unknown flag and made no source change.
- Workaround: use the documented `--path` selector for the same files.
- Root cause: `outline` and symbol-specific navigation use different selector flag names.

## Relocation worker: inspect-symbol target name absent

- Tool: `semedit inspect-symbol`.
- Target: `internal/astedit/move_cross_package.go`.
- Observed failure: `PlanCrossPackageMove` was not a declaration in that file; the read-only request found no symbol and changed no source.
- Workaround: use the file outline to identify the actual planner declaration before inspecting it.
- Root cause: the audit request guessed a helper name from another implementation shape.

## Relocation worker: inspect-symbol target name absent in inspection wire

- Tool: `semedit inspect-symbol`.
- Target: `internal/operation/wire_read_inspect.go`.
- Observed failure: the guessed `runInspect` declaration was not present; the read-only request found no symbol and changed no source.
- Workaround: inspect the file outline and request the actual registered handler by its declared name.
- Root cause: the audit request guessed a handler name without first reading the file outline.

## Relocation worker: absolute patch path targeted the coordinator checkout

- Tool: built-in `apply_patch` used for a schema constant edit.
- Target: `internal/operation/typed_descriptors.go` while working in `.scratch/worktrees/flow-relocation`.
- Observed failure: the absolute patch target omitted `.scratch/worktrees/flow-relocation` and applied one schema-key edit in the coordinator checkout instead of the assigned worker worktree.
- Workaround: stopped editing the coordinator checkout and notified root to preserve or revert its delta; the intended change remains pending in the assigned worker worktree.
- Root cause: the patch target was not checked against the assigned worktree root. Verify the full worktree prefix on every patch target.

## ST-0260: Field insertion refused a one-line struct layout

- Date: 2026-10-02
- Tool: worktree-local MCP `semantic_insert_field` through `semantic_batch` and a direct inspect-to-insert MCP test.
- Target: an existing private struct produced as `type options struct{}` or with a one-line field list.
- Observed failure: the operation refused with `struct type "options" must use a multiline field layout`; no file changed.
- Workaround: provide a multiline field list and assert that bounded form in the current CLI and MCP fixtures.
- Root cause: the initial field insertion planner supports safe multiline struct ranges only. The empty-struct case is a common typed insertion flow and remains a follow-up scope.

## Relocation worker: semantic tools cannot add a struct field

- Tool: semantic insert-structure.
- Target: `BatchResult` in `internal/mcp/batch.go`.
- Observed limitation: the available structural insertion schema supports declarations and switch cases, but not fields inside an existing struct.
- Workaround: add the `OriginFile` evidence field with a scoped patch, then use `semantic_replace_body` for the batch result projection and verify the public MCP schema and CLI txtar behavior.
- Root cause: this checkout has no semantic operation for inserting one field into an existing Go struct.

## 2026-10-02: CLI command discovery used an unsupported spelling

- Tool: `go run . insert-function --help`
- Target: `internal/astedit/safe_delete.go`
- Observed failure: the CLI reported `unknown command "insert-function"`; no source was changed.
- Workaround: use the repository's registered semantic command name after checking CLI help, then use a narrow built-in patch only for syntax the semantic tools cannot express.
- Root cause: the registered operation command uses a different public spelling than the guessed command.

## 2026-10-02: CLI help ran without the assigned worktree cache environment

- Tool: `go run . insert --help`
- Target: relocation worktree CLI
- Observed failure: Go attempted to open a build-cache path under the coordinator user's Library cache and the sandbox denied it; no source was changed.
- Workaround: rerun from the exact relocation worktree with its per-worktree `GOCACHE`, `GOPLSCACHE`, and `GOTMPDIR` settings.
- Root cause: the command omitted the required repository cache environment and was run from the coordinator checkout.

## Integration: batch verification refused workspace hook and reformatted oracle fixtures

- Tool: `semantic_batch` with `semantic_insert_decl` and five `semantic_replace_decl` children.
- Target: shared expected-revision CLI schema constant in `internal/operation`.
- Observed failure: all children persisted, but final verification refused the untrusted `golangci-lint` hook and reported promoted-field diagnostics. A clean git status was recorded at 07:55:50 UTC; the preceding `combined-tools-check-2` exited 2 before this call. The batch ran from 07:56:55 to 07:57:17 UTC. During the call, four non-target standalone hidden Go tests under `testdata/bench-oracles/task-12-request-identity-migration` had their existing imports alphabetized within one group: `correlation/identities_hidden_test.go`, `notifications/notifier_hidden_test.go`, `reporting/report_hidden_test.go`, and `scheduler/jobs_hidden_test.go`. No test bodies, assertions, import paths, or files embedded in txtars changed in this incident.
- Workaround: retain the intended declaration edits, restore unrelated fixture bytes, and run the repository's trusted `make check` gate with the configured module cache.
- Root cause: the recorded diff confirms formatting outside the six requested declaration edits, but does not identify the responsible process or running binary revision. The next `make check` began only after the four files were restored, so neither that run nor the preceding check explains the reorder. Normal `make check` fixture formatting is expected project guardrail behavior; this incident records unresolved provenance and does not seek to exempt that formatting. Current batch source scopes import organization and final Go formatting to changed paths, so attribution to batch-wide import organization is unconfirmed. Verification also requires explicit workspace trust and reported diagnostics inconsistent with the compiler gate.

## Instructions audit: Independent compression review timed out

- Tool: `codex_ask`, read-only Terra review.
- Target: MCP server instructions, generated copy descriptions, and ADR-0007/0055 compatibility.
- Observed failure: `context deadline exceeded` returned without a review report.
- Workaround: retain the captured discovery comparison and local contract analysis; request a smaller question if an unresolved compression decision requires independent review.
- Root cause: the request exceeded the tool deadline; no review approval can be inferred.

## CLI stdin worker: semantic body replacement doubled newline escapes

- Date: 2026-10-02
- Tool: worktree-local `replace-body` semantic CLI operation.
- Target: `internal/cli/cli.go`, function `buildCommand`.
- Observed failure: the operation succeeded structurally but emitted `\\n` in three Go format strings where the existing body used `\n`, changing error output to print literal backslash-n characters.
- Workaround: immediately corrected those three escape sequences with a targeted patch.
- Root cause: the shell/JavaScript quoting for the semantic CLI request over-escaped Go string literals.

## CLI stdin worker: scaffold purpose header rejected prose

- Date: 2026-10-02
- Tool: worktree-local `scaffold-file` semantic CLI operation.
- Target: `internal/cli/stdin_test.go`.
- Observed failure: passing `Package cli tests ...` as `--purpose-header` failed Go parsing with `expected 'package', found Package`; no file was created.
- Workaround: retry with a syntactically valid Go comment header.
- Root cause: the tool validates the purpose header as source text and does not add comment syntax despite the flag description calling it a comment header.

## CLI stdin worker: insert-function snippet quoting failed

- Date: 2026-10-02
- Tool: worktree-local `insert-func` semantic CLI operation.
- Target: `internal/cli/stdin_test.go`, a new test declaration.
- Observed failure: validation rejected the submitted declaration with a Go syntax error around a quoted string; no file change occurred.
- Workaround: simplify the test payload literal and submit valid Go source.
- Root cause: nested shell and JavaScript escaping made the Go string literal invalid before the semantic operation ran.

## CLI stdin worker: insert-field typecheck could not import Cobra

- Date: 2026-10-02
- Tool: worktree-local `insert-field` semantic CLI operation.
- Target: `internal/cli/stdin_test.go`, field on `stdinProbeState`.
- Observed failure: package typechecking stopped with `could not import github.com/spf13/cobra (invalid package name: "")`; no field change occurred.
- Workaround: use a targeted patch for this test-only field and record the package-import tool limitation.
- Root cause: the operation's package typecheck could not resolve the already-declared CLI dependency in this invocation.

## CLI stdin worker: insert-function reported a false Cobra dependency diagnostic

- Date: 2026-10-02
- Tool: worktree-local `insert-func` semantic CLI operation using `--source -`.
- Target: `internal/cli/cli_test.go`, registry payload inventory test.
- Observed failure: the function was inserted, but the tool reported `no required module provides package "github.com/spf13/cobra"` although Cobra is present in the module; its diagnostic claimed `diagnostic_status: changed` with a stale dependency-resolution error.
- Workaround: retain the inserted function and verify it with `make check` and the focused test suite.
- Root cause: the semantic operation's diagnostic environment failed to resolve the current module dependencies; the operation returned a syntactically valid insertion.

## 2026-10-02: Outline request used a guessed parameter name

- Tool: `semantic_outline`, target `internal/mcp/server.go`.
- Failure: request supplied `file`; the tool rejected it because outline uses
  `path`. No source mutation occurred.
- Workaround: read the tool declaration and retry with `path`.
- Root cause: caller reused another tool's parameter name without consulting
  this tool's schema. This is an agent invocation error, not an AST failure.

## 2026-10-02: Worker mutation reached the shared MCP workspace

- Tools: `semantic_scaffold_file` and `semantic_batch`.
- Intended target: `.scratch/worktrees/critical-text-presentation`.
- Observed target: coordinator `internal/mcp/presentation.go`; normalization
  also reordered imports in four task-12 hidden benchmark test files.
- Failure: the worker used a shared MCP connection bound to the coordinator
  workspace. The operation subsequently failed verification's trust gate.
- Workaround: preserve the helper and raw import diffs in coordinator scratch,
  restore only the unintended changes, and use worktree-local CLI mutations.
- Root cause: MCP tools have session-bound workspace scope, while a worker's
  shell working directory does not redirect that shared connection. The
  worker failed to follow its handover's worktree-local CLI requirement.

## Track B semantic lookup: ProjectOutput field lookup failed

- Date: 2026-10-02
- Tool: `semantic_lookup`
- Target: `internal/operation/operation.go`, selector `ProjectOutput`
- Observed failure: lookup returned `symbol not found` because `ProjectOutput` is a field, not a top-level declaration.
- Workaround: locate the owning `Def` declaration through a valid type lookup/inspection; no source edit was needed.
- Root cause: the semantic lookup selector resolves declarations and methods, not fields as standalone symbols.

## Track B semantic inspect: nonexistent pipeline result type

- Date: 2026-10-02
- Tool: `semantic_inspect_symbol`
- Target: `internal/pipeline/pipeline.go`, selector `OrganizeImportsResult`
- Observed failure: inspection returned `symbol not found`; `OrganizeImportsResult` was a caller-invented selector and not a declared symbol.
- Workaround: inspect the actual pipeline API before implementing the report type.
- Root cause: guessed a declaration name instead of discovering the API first.

The need to report successfully written paths motivated adding a report-returning API beside the compatibility wrapper. That API gap was not the cause of the failed inspection.

## Track B semantic inspections: selector names not declarations

- Date: 2026-10-02
- Tool: `semantic_inspect_symbol`
- Target: `internal/operation/typed_descriptors.go`, selectors `RemoveSchema` and `OutputField`
- Observed failure: both inspections returned `symbol not found`; the guessed selectors do not name declarations in that file.
- Workaround: inspect the actual output-contract types and nearby operation definitions using valid semantic selectors and repository search.
- Root cause: guessed schema-helper names without first identifying their declared names.

## Track B semantic inspect: output schema type in a different file

- Date: 2026-10-02
- Tool: `semantic_inspect_symbol`
- Target: `internal/operation/typed_descriptors.go`, selector `ValueSchema`
- Observed failure: inspection returned `symbol not found` because `ValueSchema` is not declared in the selected file.
- Workaround: resolve the declaration with workspace lookup before inspecting its owner file.
- Root cause: used a package type selector with an incorrect file constraint.

## Track B local semantic CLI: command used API name instead of CLI name

- Date: 2026-10-02
- Tool: worktree-local semantic CLI command
- Target: `internal/pipeline/pipeline.go`, insertion of `OrganizeImportsWithOptionsReport`
- Observed failure: `insert-function` was rejected as an unknown command; the CLI registers the command as `insert-func`. No source change occurred.
- Workaround: retry with the registered `insert-func` command after checking its flags.
- Root cause: used the MCP function name instead of the CLI command alias.

## Track B semantic inspect: guessed verify handler name

- Date: 2026-10-02
- Tool: `semantic_inspect_symbol`
- Target: `internal/operation/wire_backend.go`, selector `runVerifyBackend`
- Observed failure: inspection returned `symbol not found`; the handler has a different name.
- Workaround: locate the registered handler name before inspecting it.
- Root cause: guessed a function name from the operation role.

## Track B local semantic replace-body: stale organize-imports import

- Date: 2026-10-02
- Tool: worktree-local `replace-body` semantic CLI operation.
- Target: `internal/operation/wire_engine_imports.go`, `runOrganizeImports`.
- Observed failure: the body replacement left the now-unused `os` import, and diagnostics reported it as an error.
- Workaround: run semantic import organization for the selected file to remove the stale import.
- Root cause: the previous handler body used `os.Stat`; the new changed-file report no longer does.

## Track B local semantic CLI: incorrect checkout working directory

- Date: 2026-10-02
- Tool: worktree-local `replace-body` semantic CLI operation.
- Target: coordinator checkout `internal/operation/wire_remove.go`, `removeDef`.
- Observed failure: the command used the coordinator checkout as its working directory, inserted an unregistered error projection, and diagnostics reported missing type/field declarations. The inserted block was immediately removed; the coordinator file diff is empty.
- Workaround: verify the absolute assigned worktree path in every semantic CLI invocation before executing it.
- Root cause: a copied command used the coordinator path instead of the Track B worktree path.

## Track B local semantic replace-body: nonexistent unknown status constant

- Date: 2026-10-02
- Tool: worktree-local replace-body semantic CLI operation.
- Target: internal/operation/wire_engine_imports.go, runOrganizeImports.
- Observed failure: the replacement referenced EditUnknown, which is not declared; diagnostics reported three undefined-name errors.
- Workaround: use the explicit EditStatus("unknown") value while retaining declared status constants for known states.
- Root cause: EditStatus defines known persisted/no-op/working states but no unknown constant.

## Registry workspace choices: patch applied to coordinator checkout

- Date: 2026-10-03
- Tool: `apply_patch`.
- Target: `internal/operation/profiles.go`, `internal/operation/wire_engine.go`, `internal/operation/wire_engine_declarations.go`, `internal/operation/wire_engine_structures.go`, and `internal/mcp/server.go` in the coordinator checkout.
- Observed failure: patches intended for the assigned `.scratch/worktrees/registry-workspace-choices` checkout were applied to its parent checkout because `apply_patch` paths were anchored at the parent workspace root.
- Workaround: stop edits, identify all changed paths in both checkouts, and make subsequent patches only with verified absolute paths under the assigned worktree.
- Root cause: selected absolute paths under the coordinator checkout instead of the assigned private worktree. The shell command's `workdir` does not affect the path chosen for `apply_patch`.

## Registry workspace choices: test inserted before import organization

- Date: 2026-10-03
- Tool: worktree-local `insert-func` semantic CLI operation.
- Target: `internal/mcp/schema_test.go`, `TestToolsListUsesRegisteredBackendsForOperationInventory`.
- Observed failure: insertion diagnostics reported `undefined: javabackend` because the test import had not been added yet.
- Workaround: immediately run worktree-local semantic import organization with the Java backend package alias; diagnostics then resolved.
- Root cause: the inserted test referenced a new package alias before adding its import.

## 2026-10-03: incorrect function insertion CLI command

- Tool: local `semedit` CLI, `insert-function` command.
- Target: `internal/operation/verify_plan.go`.
- Observed failure: the bundled binary returned `unknown command "insert-function" for "semedit"`.
- Workaround: use the supported `insert-func` command for later function additions; no source was changed by the failed command.
- Root cause: I guessed the CLI subcommand instead of checking its canonical spelling (`insert-func`).

## 2026-10-03: targeted coordinator instead of private worktree

- Tool: `apply_patch`.
- Target: coordinator `internal/operation/verify_hooks.go` rather than the private worktree copy.
- Observed failure: two patches used the coordinator absolute path and changed its function signature/imports; the parent restored the file and saved the diff for reconciliation.
- Workaround: repeat the needed edit only in the verified private worktree and verify `git rev-parse --show-toplevel` before further changes.
- Root cause: I copied the coordinator root into the patch target without the `/.scratch/worktrees/verify-enabled-languages` suffix.

## 2026-10-03: scaffold-file rejected plain purpose text

- Tool: local `semedit scaffold-file` CLI.
- Target: `internal/mcp/enabled_languages_test.go`.
- Observed failure: the CLI rejected the plain `--purpose-header` value as invalid Go source; the target file was not created.
- Workaround: retry with a Go comment-form purpose header after checking the CLI contract.
- Root cause: the CLI flag expects a complete Go file comment, not comment-free prose.

## 2026-10-03: insert-type received literal newline escapes

- Tool: local `semedit insert-type` CLI.
- Target: `internal/mcp/enabled_languages_test.go`.
- Observed failure: the CLI parser received backslash-n characters in place of newlines and rejected the snippet; the file was unchanged.
- Workaround: send the source through stdin using a quoted heredoc.
- Root cause: I quoted the `--source` payload for the shell without converting escaped newlines to actual line breaks.

## 2026-10-03: inserted test type before imports

- Tool: local `semedit insert-type` CLI.
- Target: `internal/mcp/enabled_languages_test.go`.
- Observed failure: the declaration was inserted with a temporary `undefined: golang` diagnostic because the new file had not yet imported the backend package.
- Workaround: add the required imports before semantic verification.
- Root cause: type insertion verifies immediately, before subsequent import edits.

## 2026-10-03: insert-func import alias mismatch

- Tool: local `semedit insert-func` CLI.
- Target: `internal/mcp/enabled_languages_test.go`.
- Observed failure: automatic import organization aliased `internal/backend/golang` as `gobackend`, while the earlier inserted type still referenced `golang`; diagnostics reported `undefined: golang`.
- Workaround: align the type's package qualifier with the generated import alias.
- Root cause: the generated alias is selected from package naming rules and differs from the directory name.

## 2026-10-03: insert-func placement needed exported section

- Tool: local `semedit insert-func` CLI.
- Target: `internal/mcp/enabled_languages_test.go`.
- Observed failure: insertion with `--placement private_end` was rejected because the `Test*` declaration is exported; no mutation occurred.
- Workaround: use `--access public --placement public_end`.
- Root cause: the test function's exported name requires public section placement.

## Verification allowlist: incorrect inspection CLI command

- Date: 2026-10-03
- Tool: local semedit CLI help invocation.
- Target: inspection command discovery.
- Observed failure: `inspect --help` failed with unknown command; the CLI suggested `inspect-symbol`.
- Workaround: use the reported canonical command `inspect-symbol`.
- Root cause: reconstructed the CLI name instead of checking the command inventory.

## 2026-10-03: thread inventory limit rejected

- Tool: `list_threads`.
- Target: outstanding-work inventory for the main integration audit.
- Observed failure: `limit: 60` was rejected because the server maximum is 50.
- Workaround: retry with `limit: 50`.
- Root cause: the exposed tool schema does not specify its server-enforced maximum.

## 2026-10-03: safe-delete peer review timed out

- Tool: `codex_ask`, read-only Terra review.
- Target: safe-delete expansion in `internal/astedit/remove.go` and `safe_delete.go`.
- Observed failure: the peer-review request returned `context deadline exceeded`, without a review report.
- Workaround: continue concrete source review and observable probes; request an independent native review when a worker slot is available.
- Root cause: the external review transport timed out; no source mutation resulted from this call.

## 2026-10-03: safe-delete import cleanup changed initialization

- Tool: private branch CLI `remove-construct --safe-delete`.
- Target: scratch probe `main.go`, deleting an unused function that held the only reference to an imported package.
- Observed failure: deletion succeeded and import organization removed the normal import, also removing that package's observable `init` effect.
- Workaround: reject safe deletion when it would remove the final use of a normal import; add CLI coverage for standalone and buffered requests.
- Root cause: the safe-delete preflight proved declaration references and variable effects but omitted import initialization effects introduced by cleanup.

## 2026-10-03: outline request used the wrong scope field

- Tool: `semantic_outline`.
- Target: relocation worktree `internal/astedit/safe_delete.go`.
- Observed failure: `file` was rejected; the tool requires `path`.
- Workaround: copy the supported parameter name from the exposed schema and retry.
- Root cause: caller failed to consult the tool schema before dispatch; no source changed.

## 2026-10-03: safe-delete misclassified a generated test runner

- Tool: private CLI `remove-construct --safe-delete` in `safe_delete_multifile.txtar`.
- Target: unused private function in a package with same-package tests.
- Observed failure: the compiler-generated test runner's `reflect` import caused an unsupported dynamic-reference refusal. Checking only containment still failed because Go cache files can live under the module's ignored `.scratch` directory.
- Workaround: apply module source-directory boundaries before classifying package imports; retain real module test variants and their typed reference proofs.
- Root cause: loaded package module metadata and filesystem containment do not prove source membership.

## 2026-10-03: concurrent full check lost a diagnostics session

- Tool: CLI `rename` in `rename_diagnostic_delta.txtar` during the flow integration checks.
- Target: fixture `api/server.go`, `Server.OldName`.
- Observed failure: gopls diagnostics shutdown returned `JSON-RPC client is closed` during concurrently running race suites.
- Workaround: rerun the unchanged public fixture with bounded test concurrency and retain the combined integration gate.
- Root cause: transport failure observed; process/resource causation is not established by this log.
