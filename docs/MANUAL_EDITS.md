# Manual Edits Tracking Log

This document tracks all code edits performed manually (e.g. via text-based `replace_file_content` or `write_to_file`) during the development of `semedit`.

The objective is to categorize the underlying intent of each manual edit, identify which editing capabilities are absent from `semedit`, and prioritize future semantic tools.

---

## Log of Manual Edits

| ID | File | Nature of Change | IDE Capability Family | Edit / Refactoring Capability Gap | Candidate Semantic Tool |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **ME-0001** | `main.go` | Added `os`, `strings`, `semedit/internal/mcp` to imports | Import addition | Import management | `organize_imports` / `add_import` |
| **ME-0002** | `main.go` | Added `case "mcp":` to command dispatch switch | Branch statement insertion | AST statement insertion into control flow | `apply_ast_rewrite` / `insert_statement` |
| **ME-0003** | `main.go` | Added `runMCP(...)` helper function | Function declaration insertion | Top-level declaration addition | `insert_function` / `insert_decl` |
| **ME-0004** | `internal/pipeline/pipeline.go` | Added `DiagnosticDelta` struct declaration | Type declaration insertion | Top-level type addition | `insert_type` / `insert_decl` |
| **ME-0005** | `internal/pipeline/pipeline.go` | Added `ComputeDelta(...)` and `FindModuleRoot(...)` | Function declaration insertion | Top-level function addition | `insert_function` / `insert_decl` |
| **ME-0006** | `internal/pipeline/pipeline.go` | Modified `CheckDiagnostics` to resolve module root | Method body / statement replacement | Scoped block modification | `replace_symbol_body` |
| **ME-0007** | `internal/mcp/server.go` | Added diagnostic delta calculation around `golang.Rename` | Scoped statement wrapping / insertion | Method body enhancement | `replace_symbol_body` |
| **ME-0008** | `Makefile` | Added `build-next` and `promote` targets | Build configuration edit | Non-code file mutation (outside AST scope) | N/A (Build automation) |
| **ME-0009** | `.agents/plugins/.../mcp_config.json` | Registered `semedit` and `semedit-next` servers | Configuration JSON update | Structured JSON mutation | N/A (Configuration) |
| **ME-0010** | `main.go` | Added diagnostic delta calculation in `runRename` | Function body enhancement | Scoped statement wrapping | `replace_symbol_body` |
| **ME-0011** | `internal/adapters/golang/property_test.go` | Added Rapid metamorphic property test file | New test file creation | File creation / scaffolding | `scaffold_test_file` |
| **ME-0012** | `testdata/scripts/rename_diagnostic_delta.txtar` | Added txtar acceptance test | Test script creation | Test file generation | `scaffold_txtar_test` |
| **ME-0013** | `internal/mcp/server.go` | Extracted `rootUri` and `workspaceFolders` in `initialize` | Client configuration parsing | Handled at transport layer | N/A (Protocol handling) |
| **ME-0014** | `internal/mcp/server.go` | Refactored `if-else` chain to `switch` statement for `gocritic` | Control flow refactoring | AST refactoring (`if_to_switch`) | `refactor_ast` |
| **ME-0015** | `internal/adapters/golang/deps.go` | Added `AddDependency` function to manage `go get` and `go mod tidy` | Package dependency addition | External module acquisition | `add_dependency` |
| **ME-0016** | `internal/astedit/insert.go` | Added `InsertDeclaration` with placement qualifiers and visibility checking | Declaration insertion engine | AST declaration insertion | `insert_declaration` |
| **ME-0017** | `internal/pipeline/pipeline.go` | Added `OrganizeImports` function and suggestions in `ComputeDelta` | Import management engine | AST import formatting and auto-detection | `organize_imports` |
| **ME-0018** | `main.go` | Added `insert`, `imports`, and `get` CLI subcommands | CLI command dispatch branches | AST statement insertion into control flow | `apply_ast_rewrite` / `insert_statement` |
| **ME-0019** | `internal/mcp/server.go` | Added `semantic_insert_declaration`, `semantic_organize_imports`, and `semantic_add_dependency` | MCP tool registration | Function declaration addition | `insert_declaration` |
| **ME-0020** | `internal/adapters/golang/property_test.go` | Added `configureRapidChecks` helper for fast property checks | Test configuration helper | Function declaration addition | `insert_declaration` |
| **ME-0022** | `internal/astedit/access.go` & `access_test.go` | Added `AccessModifier` enum and `LanguageBackend` capability validation | Access modifier abstraction | Cross-language access model | `insert_function` / `insert_type` |
| **ME-0023** | `internal/astedit/function.go` & `function_test.go` | Added `InsertFunction` with receiver clustering and public/private section partitioning | Function insertion engine | AST function/method insertion | `insert_function` |
| **ME-0024** | `internal/astedit/type.go` & `type_test.go` | Added `InsertType` with section placement and auto-imports | Type declaration insertion engine | AST type insertion | `insert_type` |
| **ME-0025** | `internal/astedit/decl.go` & `decl_test.go` | Added `InsertDecl` with `const (...)` / `var (...)` group merging | Declaration group merging engine | AST declaration group insertion | `insert_decl` |
| **ME-0026** | `internal/pipeline/pipeline.go` & `pipeline_test.go` | Added `OrganizeImportsWithOptions` and `ImportOptions` (`Add`, `Remove`) | Import management engine | Explicit import addition and removal | `organize_imports` |
| **ME-0027** | `main.go` | Added `insert-func`, `insert-type`, `insert-decl` CLI subcommands and `--add`/`--remove` to `imports` | CLI command dispatch branches | AST statement insertion into control flow | `apply_ast_rewrite` / `insert_statement` |
| **ME-0028** | `internal/mcp/server.go` & `server_test.go` | Registered `semantic_insert_function`, `semantic_insert_type`, `semantic_insert_decl`, updated `semantic_organize_imports` | MCP tool registration | Function declaration addition | `insert_declaration` |
| **ME-0029** | `Makefile` | Added `rm -f bin/semedit` and `build-next` prerequisite to `promote` and `build` | Build automation fix | Non-code file mutation (outside AST scope) | N/A (Build automation) |
| **ME-0030** | `internal/...` | Standardized on exported package-level sentinel errors and `%w` wrapping (ADR-0013) | Domain error standardization | AST error refactoring | `standardize_errors` |
| **ME-0031** | `main.go` | Migrated CLI entry point from `flag.FlagSet` to Cobra subcommands (ADR-0014) | CLI command framework migration | Full CLI restructure | N/A (CLI Framework) |
| **ME-0032** | `internal/...`, `main.go` | Implemented structured record errors (`SymbolError`, `SyntaxError`, `PlacementError`, `VisibilityMismatchError`) with source locations, compiler standard stderr formatting, and LSP-compatible MCP error payloads (ADR-0013) | Domain error structure and diagnostic location enrichment | AST error type scaffolding / Location-aware error mapping | `standardize_errors` / `refactor_ast` |
| **ME-0033** | `internal/mcp/server.go`, `internal/astedit/...`, `internal/symbol/...`, `main.go` | Resolved Codex peer review findings for ADR-0013: UTF-16 character translation, virtual snippet URIs, single-render CLI locations, URL escaping, and VisibilityMismatchError coordinates | Error record and location delivery refinement | Diagnostic location formatting / LSP encoding | `refactor_ast` |
| **ME-0034** | `internal/astedit/body.go`, `main.go`, `internal/mcp/server.go` | Added `semantic_replace_body` engine, Cobra subcommand, and MCP tool | Scoped function/method body replacement | Scoped block modification | `semantic_replace_body` |
| **ME-0035** | `internal/astedit/scaffold.go`, `main.go`, `internal/mcp/server.go` | Added `semantic_scaffold_file` engine, Cobra subcommand, and MCP tool | New file creation with sibling package inference | File creation / scaffolding | `semantic_scaffold_file` |
| **ME-0036** | `internal/astedit/switchcase.go`, `main.go`, `internal/mcp/server.go` | Added `semantic_insert_case` engine, Cobra subcommand, and MCP tool | Switch case clause insertion | AST statement insertion into control flow | `semantic_insert_case` |
| **ME-0037** | `internal/mcp/batch.go`, `internal/mcp/server.go` | Added `semantic_batch` engine and MCP tool for sequential multi-edit execution | Multi-edit composition and orchestration | Protocol-level multi-edit orchestration | `semantic_batch` |
| **ME-0038** | `internal/mcp/server.go`, `internal/mcp/reload_*.go`, `main.go`, `Makefile` | Added `semantic_reload` tool, `--live-reload` flag, listChanged capability, and atomic binary promotion (ADR-0017) | MCP server lifecycle & live re-exec | Dynamic MCP tool reload & in-place exec | `semantic_reload` |
| **ME-0039** | `task-07-generate-template-main` benchmark trace | After `semantic_replace_body` and `semantic_organize_imports`, the agent manually re-read `main.go` to inspect the selected imports | Post-edit context feedback | Mutation results omit the post-format enclosing scope and selected imports, forcing confirmation reads before the next semantic operation | `post_edit_context` / enclosing-scope projection |
| **ME-0040** | `internal/astedit/construct.go`, `replace_decl.go`, `collision.go`, `construct_test.go` | Created new engine files, declarations, and AST tests in an isolated worktree | File scaffolding and declaration insertion | `semantic_scaffold_file`, `semantic_insert_type`, `semantic_insert_function`, and `semantic_insert_decl` fit parts of the edit, but two workers could not verify the MCP server's active checkout root before a write | Explicit worktree binding and root confirmation |
| **ME-0041** | `internal/astedit/decl.go`, `internal/operation/wire_engine_declarations.go` | Added `Overwrite` fields, a collision preflight, and one `insertDeclParams` element | Member, statement, and composite element insertion | Existing tools add top-level declarations or replace whole function bodies; they cannot insert one struct field, guard, or literal element | `semantic_insert_field`, `semantic_insert_statement`, `semantic_insert_composite_element` |
| **ME-0042** | `internal/mcp/schema_test.go`, `tools/benchmark-harness/driver.go` | Added an MCP schema test and extracted two benchmark driver helpers | Function insertion and body replacement | Existing tools fit, but this worker avoided MCP writes because the configured server root might select the primary checkout | Explicit worktree binding and root confirmation |
| **ME-0043** | `internal/operation/wire_backend_test.go`, `tools/benchmark-harness/agy_driver.go` | Changed a registry count, lowercased one error string, and added a security justification comment | Expression and comment replacement | Whole-body replacement is too broad for one literal or comment | `semantic_replace_expression` and targeted comment editing |
| **ME-0044** | `testdata/scripts/construct_replacements*.txtar`, `mcp_server.txtar`, ADR and friction Markdown | Authored CLI/MCP commands and literal before/after fixture trees; updated documentation | Test archive and Markdown editing | Go AST tools do not structure txtar archives or Markdown tables | Keep literal txtar fixtures; use structured fixture/document editing outside Go AST operations |
| **ME-0045** | `internal/operation/wire_*.go`, `internal/mcp/server.go` | Reworded tool-summary string literals after the planner-selection audit | Expression or string-literal replacement | No operation can replace a scoped literal without regenerating its enclosing function body or declaration | `semantic_replace_expression` with function or declaration owner, exact old value, and ambiguity rejection |
| **ME-0046** | `skills/semedit/SKILL.md`, this log | Added one built-in-preference row per tool and recorded the catalog audit | Structured Markdown editing | Go AST operations do not edit Markdown tables or prose | A document-aware table editor; retain atomic text editing until available |
| **ME-0047** | ADR-0046 replay worktree Go source | Inserted existing struct fields, one collision guard/call, a helper parameter type correction, and attached lint comments | Bounded member, statement, signature, and comment edits | Available tools create declarations or replace whole bodies; they cannot target these nodes proportionately | `semantic_insert_field`, `semantic_insert_statement`, `semantic_change_signature`, and `semantic_set_doc_comment` |
| **ME-0048** | ADR-0046 replay txtar, ADR, and index | Added literal CLI/MCP before/after trees and updated prose | Structured fixture and document editing | Go AST tools do not edit txtar containers or Markdown tables | Preserve literal executable fixtures and use atomic document edits |

---

## Dogfooded Semantic Operations (Executed via `semedit` MCP Server)

Subagents executed the following edits deterministically via `semedit` MCP tools instead of manual text edits:

| Operation ID | Target File | Target Symbol | New Name | Ingress Interface | Verification Result |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SE-0001** | `internal/symbol/resolver.go` | `Symbol.FormatQualifiedName` | `BuildQualifiedName` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call sites renamed deterministically; `go test` passed. |
| **SE-0002** | `internal/astedit/insert.go` | `validateSnippet` | `verifySnippetSyntax` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call site renamed deterministically; `go test` passed. |
| **SE-0003** | `internal/astedit/function.go` | `verifyFunctionSnippet` | `parseFunctionSnippet` | `semedit` (`semantic_rename`) | Zero-token diff, declaration and call site renamed deterministically; `go test` passed. |
| **SE-0004** | `internal/operation/wire_engine_mutations.go`, `wire_engine_declarations.go` | `ReplaceLoopReq`, `ReplaceDeclReq` | New request types | `semedit` (`semantic_insert_type`) | Explicit absolute worktree targets; `make check` passed after integration. |
| **SE-0005** | `internal/operation/wire_engine_mutations.go`, `wire_engine_declarations.go` | Replacement parsers, handlers, definitions | New functions | `semedit` (`semantic_insert_function`) | Registry and CLI/MCP schemas passed `make check`. |
| **SE-0006** | `internal/operation/wire_engine_mutations.go`, `wire_engine_declarations.go` | `replaceLoopParams`, `replaceDeclParams` | New declarations | `semedit` (`semantic_insert_decl`) | Parameter contracts passed registry and schema tests. |
| **SE-0007** | `internal/operation/wire_engine.go`, `wire_engine_declarations.go` | Registry calls and overwrite parsing | Updated bodies | `semedit` (`semantic_replace_body`) | Registry count and CLI string-shape tests passed. |
| **SE-0008** | `cmd/docgen/benchmark_aggregate.go` | `writeBenchmarkBrowserAssets` and extracted helpers | Body replacement and new functions | `semedit` (`semantic_replace_body`, `semantic_insert_function`) | Documentation generation and benchmark tests passed. |

---

## Edit / Refactoring Capability Gap Analysis

### Implemented Edit / Refactoring Capabilities

1. **`organize_imports` with Explicit Add/Remove** (Completed):
   - Supports plain import paths, custom aliases `alias "path"`, blank imports `_ "path"`, and explicit import removal alongside auto-resolution.
2. **`insert_function` / `insert_type` / `insert_decl`** (Completed):
   - Receiver-aware method clustering, strict public/private section partitioning, access modifier inference (`infer`, `public`, `private`), and `const`/`var` group merging (`append_group`).
3. **`replace_body` / `scaffold_file` / `insert_case` / `batch`** (Completed - ADR-0016):
   - `semantic_replace_body`: Scoped function/method body replacement by symbol name with in-memory validation and diff generation.
   - `semantic_scaffold_file`: Directory and file creation with sibling non-test package inference and overwrite guard.
   - `semantic_insert_case`: AST switch case clause insertion across multiple placement modes (`first`, `last`, `before_default`, `before`, `after`) with expression-matching anchor verification.
   - `semantic_batch`: Sequential fail-fast multi-edit orchestration with per-file deferred import optimization.
4. **In-Tree Live-Reload for Self-Modification** (Completed - ADR-0017):
   - `semantic_reload` & `--live-reload`: In-place stdio re-exec via `syscall.Exec`, dynamic schema discovery via `notifications/tools/list_changed`, and atomic Makefile binary promotion.

### Unimplemented Edit / Refactoring Capability Families

1. **Function Block & Registration Statement Insertion (`insert_statement` / `append_call`)**:
   - **Need**: Solves ME-0002, ME-0018, ME-0027, and `rootCmd.AddCommand(...)` registration. Inserting a statement or call expression inside a specific function block (e.g. adding a command to a CLI root or an HTTP route to a router) without rewriting the whole function.
2. **Compound Literal / Collection Element Insertion (`insert_element`)**:
   - **Need**: Solves appending tool definitions into `tools/list` JSON/slice arrays or registering handlers in static tables.

## ADR-0046 Post-Commit Semantic-Tool Audit

Three workers audited their own edits after commit `bce450f` reached `main` and `make check` passed. The operation worker used absolute worktree paths successfully with `semantic_insert_type`, `semantic_insert_function`, `semantic_insert_decl`, and `semantic_replace_body` for new request types, parameters, parsers, handlers, registry calls, and the `writeBenchmarkBrowserAssets` extraction. The AST and integration workers used atomic scoped edits because prior worktree-binding failures ([ST-0017](SUBOPTIMAL_TOOLS.md), [ST-0028](SUBOPTIMAL_TOOLS.md)) left the active MCP root uncertain. Successful absolute-path calls in one worker show that the missing piece is a reliable root contract, not a blanket inability to edit a worktree.

| Actual edit | Existing semantic fit | Why the worker did or did not use it | Improvement |
| :--- | :--- | :--- | :--- |
| Add `LoopOptions`, `ReplaceDeclOptions`, engine helpers, and AST test functions | Scaffold a Go file, then insert types and functions | AST worker avoided a possible write to the primary checkout; operation worker used these tools on explicit absolute paths | Return the canonical active root in tool results and reject a target outside an explicitly selected worktree |
| Add `ErrDeclCollision` | `semantic_insert_decl` can add a sentinel variable | AST worker avoided an unverified MCP root | Root confirmation before mutation; preserve a comment attached to the inserted spec |
| Insert `Overwrite bool` into `DeclOptions` and `InsertDeclReq` | No member-level operation | Inserting a second type would collide; replacing the whole type declaration would regenerate unrelated fields | `semantic_insert_field` keyed by file, type, field, and placement |
| Insert collision handling after target parsing in `InsertDecl` | `semantic_replace_body` could regenerate the complete body | The body also contains declaration placement and grouping; replacing it for one preflight risks unrelated behavior | `semantic_insert_statement` with a function and a verified before/after statement anchor |
| Add `overwrite` to the existing `insertDeclParams` slice | New `semantic_replace_decl` could replace the whole variable, but it was absent from the live catalog ([ST-0037](SUBOPTIMAL_TOOLS.md)) | The desired change was one element in an existing composite literal | Refresh the catalog after building; add `semantic_insert_composite_element` with a named variable and anchor |
| Change `len(entries) != 17` to `len(entries) != 19`, or lowercase one `fmt.Errorf` string | Whole-body replacement is possible but disproportionate | Both are one-expression edits | `semantic_replace_expression` scoped to the owning function, with ambiguity rejection |
| Add the `agyBin` G204 comment | No comment-level operation | Replacing `runAgy` would rewrite a large function for one line | Insert or replace a comment attached to a selected statement |
| Add CLI/MCP txtar commands and wanted trees | No Go AST operation applies to the archive container | The worker authored the archives directly and compared full resulting files | Retain public CLI txtar contracts with literal before/after trees |

The new [CLI fixture](../testdata/scripts/construct_replacements.txtar) and [MCP batch fixture](../testdata/scripts/construct_replacements_mcp_batch.txtar) are executable regression examples. They confirm that a failed duplicate insertion and an ambiguous loop selection leave the file unchanged, then compare the exact trees after `--overwrite`, `replace-decl`, and `replace-loop`. The loop fixture preserves statements outside the edited loop.

### Candidate Regression Fixtures for Missing Operations

**Field insertion.** Given `type InsertDeclReq struct { File string }`, inserting `Overwrite bool` should produce `type InsertDeclReq struct { File string; Overwrite bool }` after formatting. A second insertion should report a collision without changing the file.

**Anchored statement insertion.** Given a function body with `parseTarget()` followed by `appendDecl()`, inserting `checkPackageCollisions()` after `parseTarget()` should preserve `appendDecl()` and all other statements. A repeated or missing anchor should fail before mutation.

**Composite element insertion.** Given `var params = []ParameterContract{{Name: "file"}}`, inserting `{Name: "overwrite", Type: ParamBoolean}` after the file entry should retain the first element and order. This is the exact shape needed for `insertDeclParams`.

**Expression replacement.** In `TestRegisteredDefsHonorContracts`, replace only the literal in `len(entries) != 17` with `19`. If two matching expressions exist in the same function, require a path instead of silently choosing one.

**Declaration and loop replacement.** The committed txtar trees already exercise a same-file collision, explicit overwrite, and complete loop replacement. Extend them with a grouped `const` whose later omitted expression inherits the target's value; replacing the target must fail without changing that sibling. A nested-loop fixture should use a reported `loop_path` to modify the inner loop while preserving its parent and a sibling loop.

**Workspace verification.** `semedit verify --path .` failed after the Go cache was copied into `.scratch/go/mod`, because recursive `gofmt` entered intentionally malformed upstream test fixtures ([ST-0038](SUBOPTIMAL_TOOLS.md)). Verification of `internal/astedit` returned zero diagnostics. Directory verification should enumerate project source files while excluding `.scratch` and dependency caches, and report formatter stderr on failure.

## Holistic Read and Write Tool Audit (2026-09-25)

The promoted full-profile server advertises 21 tools: 19 registry operations, `semantic_batch`, and `report_feedback`. `semantic_reload` is a twenty-second conditional tool when `--live-reload` is enabled. Only `semantic_lookup` is read-only. `semantic_verify` can format or change imports, `semantic_snapshot` writes a journal, and the Maven tools execute a build. A planner must not classify them as harmless reads.

| Axis | Available now | Missing capability and concrete example |
| :--- | :--- | :--- |
| Symbol read | `semantic_lookup` resolves a named symbol in Go, trusted Rust/Java/Scala, or standalone Haskell | Find all references or callers of `InsertDecl`; lookup returns a declaration, not a reference set. A read-only `semantic_references` would replace `rg 'InsertDecl'` for this intent. |
| Source/structure read | Lookup reports a symbol location | Return the selected AST node, enclosing body, imports, and canonical workspace root. Before changing `insertDeclParams`, an agent needs the current composite literal without shell `sed` or a full-file read; before a worktree write, it needs proof of the target root. |
| Diagnostics read | Mutations return diagnostic deltas; `semantic_verify` checks and formats | Inspect current diagnostics without writing or running formatters. `semantic_diagnostics` should give the same revision-scoped diagnostic information as mutation receipts. |
| Go writes | Rename, body/loop/declaration replacement, declaration/function/type/case insertion, scaffolding, imports, dependencies, assertion conversion, batch | Insert `Overwrite bool` into `DeclOptions`; insert a collision guard after a named statement; append `{Name: "overwrite"}` to `insertDeclParams`; change only `len(entries) != 17`; attach a `#nosec` comment. These need field, anchored-statement, composite-element, expression, and attached-comment operations. |
| Other-language writes | Trusted Rust and Java selected-file rename; bounded Java formatting/import actions | Go-only structural operations remain unavailable in Rust, Java, Scala, and Haskell. Extend only after backend capability metadata and matching CLI txtar coverage define the supported pair. |
| Workspace/control | Snapshot/undo, trusted fixed Maven compile/test, optional live reload, feedback draft | Preview a proposed edit and its diagnostics without mutation, and select a worktree/root explicitly. A built server should announce the canonical active root and refreshed tool catalog before an agent writes. |

The catalog descriptions now begin with an affirmative `Use this tool instead of...` trigger. The dedicated MCP schema test checks every full-profile tool and the conditional live-reload catalog. The skill table has a distinct built-in alternative for each tool, including Maven, snapshot, undo, assertion conversion, feedback, and conditional reload. Selection remains bounded by each schema: `semantic_replace_loop` requires a complete loop, `semantic_replace_decl` requires the same existing identifier in the replacement, and `semantic_verify` may write formatting. The `ExampleRaw` field in registry entries still does not appear as concrete examples in MCP input schemas; selected parameter descriptions should surface symbol, path, placement, and ambiguity examples. Backend-specific scope is in prose rather than generated from capabilities, so stale descriptions are possible when a backend changes.

### Re-evaluation of the ADR-0046 implementation diff

The preserved pre-implementation diff is `.scratch/adr0046-original-implementation.patch` (`d9226f9..bce450f`). The current promoted binary adds `semantic_replace_loop` and `semantic_replace_decl` to the operations available when the earlier workers ran. The table below distinguishes available operations from edits for which the new tools are still too coarse.

| Original diff site | Tool that should be selected now | Remaining reason for a fallback |
| :--- | :--- | :--- |
| New `internal/astedit/construct.go`, `collision.go`, `replace_decl.go`, and `construct_test.go` | `semantic_scaffold_file`, then `semantic_insert_type`, `semantic_insert_function`, and `semantic_insert_decl` | Source reads and attached comment placement have no focused semantic operation. |
| `ErrDeclCollision` in `internal/astedit/errors.go` | `semantic_insert_decl` for the new sentinel | An existing grouped declaration may need a comment-preserving insertion contract. |
| `InsertDecl` collision preflight and operation parsers/handlers | `semantic_replace_body` for changed function bodies; `semantic_insert_function` for new helpers | Replacing a long body to add one guard is disproportionate; an anchored statement insertion is missing. |
| `DeclOptions.Overwrite` and `InsertDeclReq.Overwrite` | No existing member-level operation | `semantic_insert_field` should target the named struct and reject a duplicate field. |
| `insertDeclParams` addition and any existing package variable update | `semantic_replace_decl` can replace the complete variable definition now | A single composite element still requires regenerating the entire value; `semantic_insert_composite_element` would preserve siblings. |
| Existing `benchmarkBrowserShortcode` constant in `cmd/docgen/benchmark_aggregate.go` | `semantic_replace_decl` should be used now; the earlier `semantic_replace_body` attempt failed because the symbol was a constant ([ST-0035](SUBOPTIMAL_TOOLS.md)) | The current MCP catalog in this task remains stale; the promoted semantic CLI can perform the declaration replacement. |
| `registerEngineOps` registrations | `semantic_replace_body` | `semantic_insert_statement` could add each registration at an anchored call without rewriting the rest of the function. |
| `internal/mcp/schema_test.go` new tests and new request/operation definitions | `semantic_insert_function`, `semantic_insert_type`, `semantic_insert_decl`; `semantic_batch` when ordering helps | Schema assertions inside an existing test need whole-body replacement until scoped statement/expression edits exist. |
| Registry count or one error string in `wire_backend_test.go`, plus security comment in `agy_driver.go` | No proportional operation; `semantic_replace_body` is a broad fallback | `semantic_replace_expression` and attached-comment editing would mutate the intended node only. |
| Existing loops in tests or handlers when a loop alone changes | `semantic_replace_loop`, selected by `loop_on` and `loop_path` | It cannot add a new loop or change surrounding statements; full replacement source is required. |
| Go imports in any changed file | `semantic_organize_imports`, or automatic import organization on the selected edit | Package import edits must not be mistaken for external module dependency changes. |
| `testdata/scripts/*.txtar`, ADR prose, and friction logs | No Go semantic operation applies | Keep literal before/after trees for executable regression fixtures; use atomic document/archive edits. |

A useful read regression fixture would put two methods named `Run` in different packages and assert that lookup returns the selected declaration and an explicit root. A write regression fixture would start with `type DeclOptions struct { Group string }` and insert `Overwrite bool`; a repeated call must fail before mutation. For a composite fixture, start with `var params = []ParameterContract{{Name: "file"}}`, add `{Name: "overwrite"}` after the existing element, and compare the exact resulting tree. For an expression fixture, change only `len(entries) != 17` to `len(entries) != 19` inside `TestRegisteredDefsHonorContracts`, rejecting ambiguous matches. These are safe to store literally because they come from this codebase.

### Independent ADR-0046 replay results

A Luna high replay started from `d9226f9` in `.scratch/worktrees/adr0046-replay`, with the original implementation diff at `.scratch/adr0046-original-implementation.patch` supplied for comparison but never applied. Its Go cache was copied into that worktree. It reimplemented the AST engine, operation wiring, CLI/MCP coverage, and matching ADR/index changes using the promoted binary and available semantic MCP tools. The replay remains isolated and uncommitted; `make check` and `git diff --check` pass there. The canonical implementation remains on `main`.

The replay used `semantic_lookup`, `semantic_scaffold_file`, `semantic_insert_type`, `semantic_insert_function`, `semantic_replace_body`, and semantic import organization for fitting source work. It used the promoted CLI `replace-decl` on the existing `insertDeclParams` variable, and the new CLI/MCP txtar fixtures exercised `replace-loop`, `replace-decl`, overwrite, batch, and discovery. The child MCP catalog still lacked the two new callable operations after binary promotion ([ST-0039](SUBOPTIMAL_TOOLS.md)); source reads needed a narrow file-read fallback because lookup returned locations without source. A worktree-relative scaffold call resolved under the primary checkout ([ST-0042](SUBOPTIMAL_TOOLS.md)).

| Replay edit and before/after example | Tool fit and observed limit |
| :--- | :--- |
| `type DeclOptions struct { TargetSymbol string }` gained `Overwrite bool` | `semantic_insert_type` would add a second type, so a bounded `semantic_insert_field` is needed. The replay made a scoped atomic field edit. |
| `InsertDecl` gained `handleInsertDeclCollisions(...)` immediately after parsing | `semantic_replace_body` would regenerate the entire function; an anchored `semantic_insert_statement` would preserve its other placement logic. |
| `insertDeclParams` gained `{Name: "overwrite", Type: ParamBoolean}` | The promoted CLI `replace-decl` accepted a complete variable replacement, but the old formatter flattened eight entries into one line twice. [ST-0040](SUBOPTIMAL_TOOLS.md) records the regression; `42b2449` fixes multiline preservation on `main`. A composite-element operation remains the proportional edit. |
| `TestRegisteredDefsHonorContracts` changed only `17` to `19` | An abbreviated `semantic_replace_body` request removed unrelated assertions; the worker restored the full body through another semantic call. The final diff changes only the expected count. [ST-0041](SUBOPTIMAL_TOOLS.md) shows why `semantic_replace_expression` is needed. |
| `processInteractiveFollowups` initially accepted `beforeFiles []string`; the actual argument type was `workspaceSnapshot` | No semantic signature-change operation exists. A minimal atomic type edit corrected the parameter after compiler diagnostics. |
| `semantic_scaffold_file` created `package astedit` without this repository's required WHY header | File scaffolding needs an optional initial comment, or a `semantic_set_doc_comment` operation for package and declaration comments. The worker added the header atomically before semantic insertion. |
| `cmd := exec.CommandContext(ctx, agyBin, args...)` gained a scoped `// #nosec G204 ...` justification | No operation attaches a comment to a selected expression or statement. Replacing the whole body would be excessive. |
| `testdata/scripts/construct_replacements_cli.txtar` preserved both `target.go` and `sibling.go` after a rejected package collision | Literal before/after trees are appropriate for txtar contracts. Go AST operations cannot edit the archive container; the replay used atomic fixture edits. |

The replay's tests also exposed a false-confidence risk: a first green `make check` did not cover method names or `_` in collision preflight. A final review added cases proving a receiver method `A` can coexist with package constant `A`, repeated `_` declarations are allowed, and duplicate incoming names fail without a write. Its loop tests reject non-loop source, restrict `loop_on` to header components, and use one reported hierarchical path when candidates are ambiguous. The main collision walker already ignores receiver methods and `_`; it scans all sibling `.go` files without build-tag filtering, so mutually exclusive build files remain a candidate false-positive case to test and address.

**Tool priorities from the replay:** first expose canonical root, active binary/catalog version, declaration outline, source excerpts, and read-only diagnostics; then add field, anchored-statement, composite-element, expression, signature, and comment operations with ambiguity rejection and literal-tree fixtures. Keep `semantic_replace_loop` for complete loop-to-loop edits; replacing a loop with a call requires a general selected-statement replacement. Finally, preserve multiline input layout and return a bounded post-edit scope in mutation receipts so callers can notice unintended sibling changes before continuing.

## Fresh-Server ADR-0046 Replay (Iteration 2)

A second Luna high worker started from `d9226f9` in `.scratch/worktrees/adr0046-replay-2` with a fresh `mcp --profile full --live-reload` process. Its `tools/list` exposed 22 operations, including `semantic_replace_loop` and `semantic_replace_decl`. The worker read the original patch for context without applying it and rebuilt the operations through semantic calls where they fit. `go test ./...` passed, including 53 txtar workflows; `git diff --check` passed. `make check` reported exactly four inherited lint findings captured before source edits (funlen, gocognit, gosec, ST1005). The replay remains isolated and uncommitted.

The following records enumerate source reads and edits that still required a fallback. `R2-MR` identifiers distinguish this replay from the earlier audit; the matching failed MCP calls are in `SUBOPTIMAL_TOOLS.md`.

This section records source reads that the advertised semantic MCP tools cannot provide. `semantic_lookup` returns declaration locations only; the complete `tools/list` has no source read or AST display operation.

### R2-MR-0001: Read located Go declarations to plan semantic edits

- Target context: `internal/astedit/decl.go`, `internal/astedit/switchcase.go`, `internal/astedit/errors.go`, and operation wiring files.
- Attempted route: `semantic_lookup` for `InsertDecl` and `InsertCase` returned only file, line, column, offset, and kind. The advertised tool list has no source read or AST display operation.
- Missing capability: Read a declaration's source or AST after locating it semantically.
- Literal before/after example: Before, lookup returned `{"symbol":"InsertDecl","file":"internal/astedit/decl.go","line":29,"column":6,"offset":714,"kind":"function"}`. After fallback read, the actual `func InsertDecl(...)` body is available for implementation planning.
- Fallback: Read only relevant located source files with `cat`; no source text was edited by this fallback.

### R2-MR-0002: Read operation registration contracts

- Target context: `internal/operation/wire_engine_mutations.go`, `wire_engine_declarations.go`, and operation registry files.
- Attempted route: `semantic_lookup` located `replaceBodyDef` and `parseReplaceBody`, but returns no function body. `tools/list` exposes no source read or AST display tool.
- Missing capability: Read operation contracts and handler wiring through the semantic interface.
- Literal before/after example: Before, lookup returned `{"symbol":"replaceBodyDef","file":"internal/operation/wire_engine_mutations.go","line":67,"column":6,"offset":2844,"kind":"function"}`. After fallback read, the current `Def[ReplaceBodyReq, FileEditRes]` literal is visible for copying its contract pattern.
- Fallback: Read the named operation files with `cat` before selecting semantic insertions or necessary registration edits. A subsequent inventory confirmed there is no `internal/operation/registry.go`; the registry lives in `operation.go`.

### R2-MR-0003: Locate CLI ingress wiring

- Target context: CLI command generation for registry operations.
- Attempted route: A shell `rg` query for guessed command-constructor text produced no output; this was an inappropriate source-search route under the replay protocol. The MCP registry schema and `registerEngineOps` location are sufficient to determine whether CLI wiring is registry-derived.
- Missing capability: Semantic lookup is name-based and does not provide textual/structural search for an unknown command wiring point.
- Literal before/after example: Before, `rg -n 'replace-body|insert-decl|OperationDefs|Definitions|wire' main.go cmd` produced no matches. After, inspect the `registerEngineOps` declaration through the semantic location and establish that ingress is registry-derived.
- Fallback: No shell source search will be used further; use known symbols and registry contracts.

### R2-MR-0004: Repair nested declaration group insertion

- Target: `internal/astedit/errors.go`, the `ErrSymbolCollision` sentinel in the package error group.
- Attempted route: `semantic_insert_decl` inserted a nested `var` declaration into the existing parenthesized `var` group and failed during import organization with `expected IDENT, found var` (also logged as ST-0050). `semantic_replace_decl` could not select a grouped declaration spec.
- Missing capability: Insert one documented declaration spec into a specific grouped `var` declaration while preserving its comment attachment.
- Literal before/after example: Before the manual repair, the new text was `var (\n\tvar ErrSymbolCollision = errors.New(...)\n)` inside the existing `var (...)` group. After, the group contains the spec `ErrSymbolCollision = errors.New("symbol collision")` beside its own comment, without a nested `var` keyword.
- Fallback: After preserving `.scratch/errors.go.baseline` and `.scratch/semantic-insert-decl-errors.go.failed`, atomically move the sentinel spec into the existing group and verify with `semantic_verify`.

### R2-MR-0005: Add overwrite fields to existing option and request structs

- Targets: `internal/astedit/decl.go` (`DeclOptions`) and `internal/operation/wire_engine_declarations.go` (`InsertDeclReq`).
- Attempted route: `semantic_replace_decl` on `DeclOptions` was rejected because the operation only replaces type aliases. The tool catalog has no struct-field insertion or field-aware type rewrite operation. The same limitation applies to `InsertDeclReq`.
- Missing capability: Add a named field to an existing Go struct while preserving its other fields.
- Literal before/after example: Before: `type DeclOptions struct { ... AutoOrganizeImports bool }`. After: the same struct ends with `AutoOrganizeImports bool` and `Overwrite bool`; `InsertDeclReq` likewise gains `Overwrite bool`.
- Fallback: Apply only those two field insertions with atomic writes after this log entry. Other related behavior and registry contract edits will use semantic MCP tools.

### R2-MR-0006: Add required Go file header after semantic scaffold

- Target: new `internal/astedit/loop.go`.
- Attempted route: `semantic_scaffold_file` created the correct package clause but exposes no file-purpose header parameter. `semantic_insert_function` cannot place a comment before the package clause.
- Missing capability: Scaffold a Go file with its mandatory purpose comment.
- Literal before/after example: Before: `package astedit`. After: `// Package astedit replaces selected loop nodes without regenerating their enclosing function.` followed by `package astedit`.
- Fallback: Atomically prepend only the package purpose comment; all declarations will use semantic insertion tools.

### R2-MR-0007: Restore request type doc-comment attachment

- Targets: `internal/operation/wire_engine_declarations.go` and `wire_engine_mutations.go`.
- Attempted route: `semantic_insert_type` successfully inserted new request types, but moved each existing request type's doc comment above the new type. `make check` reported the resulting revive violations. The advertised API cannot move comments independently.
- Missing capability: Move a declaration comment with its original AST declaration when inserting a sibling type.
- Literal before/after example: Before in declarations: `// InsertDeclarationReq inserts one top-level Go declaration.` then a blank line, then `// ReplaceDeclReq ...` and `type ReplaceDeclReq`. After: `// ReplaceDeclReq ...` is immediately above `type ReplaceDeclReq`; `// InsertDeclarationReq ...` is immediately above `type InsertDeclarationReq`. Apply the same ordering for `ReplaceBodyReq` and `ReplaceLoopReq`.
- Fallback: Move only those two existing comments atomically after this log entry.

### R2-MR-0008: Inspect test conventions and prepare contract fixtures

- Targets: `internal/astedit/decl_test.go`, `switchcase_test.go`, `testdata/scripts/*.txtar`, and the ADR-0046 tool schema section.
- Attempted route: The advertised semantic MCP tools can locate named declarations but cannot read tests, txtar scripts, or Markdown content. No semantic fixture or document read capability is listed.
- Missing capability: Read source and contract fixture contents after locating them; create or edit txtar and Markdown artifacts.
- Literal before/after example: Before, semantic lookup provides only `file`, `line`, `column`, `offset`, and `kind`. After fallback read, the actual `func TestInsertDecl...` body and `--`-delimited txtar fixture contents are available for matching project conventions.
- Fallback: Read only the named test and contract files. Txtar and Markdown changes will use atomic writes after this entry.

### R2-MR-0009: Record non-semantic test-source inspection

- Targets: `internal/astedit/decl_test.go` and `switchcase_test.go`.
- Attempted route: The lack of a semantic source reader was documented in R2-MR-0008, but a subsequent inspection command used `sed` and `rg` to read and locate test source. This was outside the replay's allowed source-search route.
- Missing capability: Semantic test-source reading and unknown-name structural lookup.
- Literal before/after example: Before, the inspection searched `internal/astedit/decl_test.go` and `switchcase_test.go` through shell text utilities. After, the existing test package, imports, and helper conventions are visible.
- Fallback: No source modifications came from that shell inspection. Future test edits will use semantic insertion, and no further shell source searches will be used.

### R2-MR-0010: Document loop path selection in ADR-0046 and its index

- Targets: `docs/adr/0046-construct-level-ast-replacement-and-declarative-updates.md` and `docs/adr/README.md`.
- Attempted route: The semantic MCP tools do not read or edit Markdown. The accepted ADR documents ambiguity but omits the `loop_path` parameter exposed by the operation.
- Missing capability: Read and update a Markdown contract and its mandatory index atomically.
- Literal before/after example: Before, the `semantic_replace_loop` schema has `loop_on` but no path property. After, it includes `loop_path` with the candidate path supplied by an ambiguity diagnostic; the index summary mentions both selectors.
- Fallback: Use one atomic write for the ADR and index update.

### R2-MR-0011: Add executable CLI and MCP batch fixtures

- Targets: `testdata/scripts/replace_loop*.txtar`, `replace_decl.txtar`, `insert_decl_collision.txtar`, `construct_replacements_mcp_batch.txtar`, and `mcp_server.txtar`.
- Attempted route: No semantic operation creates or edits txtar contract scripts; the available tools edit Go source only.
- Missing capability: Create/update executable text fixtures with before/after file snapshots and ingress commands.
- Literal before/after example: Before, no `replace_loop.txtar` exists. After, the script invokes `semedit replace-loop`, compares the changed Go file against a literal `want/...` section, and runs `go test ./api`.
- Fallback: Create or update only these txtar/Markdown files with atomic writes after this entry.

### R2-MR-0012: Change helper signature to return replacement status

- Target: `internal/astedit/decl.go`, `checkDeclCollision`.
- Attempted route: `semantic_replace_body` added a handled flag to distinguish a completed overwrite from an ordinary insertion, but function-body replacement cannot change the declaration signature. The resulting diagnostics were logged as ST-0054.
- Missing capability: Change an existing function's result types while preserving its declaration.
- Literal before/after example: Before: `func checkDeclCollision(...) error`. After: `func checkDeclCollision(...) (bool, error)` to report both whether overwrite handled the request and any error.
- Fallback: Change only this function signature atomically after this log entry; the caller change remains a semantic body replacement.

## Commented Declaration Replacement Follow-up

`semantic_replace_body` updates the existing replacement routines, and `semantic_insert_function` adds the AST regression. The existing `replacementDecl` struct needs two comment-presence fields so replacement can preserve an old doc comment when the new snippet has none, or replace it when the snippet supplies one. `semantic_replace_decl` accepts type aliases, not an existing struct body, and there is no field-insertion operation. An atomic edit will add only those two fields. Before: `type replacementDecl struct { kind token.Token; text, specText string }`. After: it also records `hasLeadingComment` and `hasTrailingComment`. The CLI txtar archive is not Go source and has no semantic edit operation; its literal before/after fixture will be written atomically.

## Catalog Description and Skill Routing Review (2026-09-25)

An independent worker inspected a fresh full-profile, live-reload MCP catalog at `46d5816` and all 22 skill routes. Commit `bf6e142` tightened the following wording without changing operation semantics or schema types. The catalog test now requires every description to start with `Use this tool instead of`; the skill has one selection row for every exposed tool.

| Tool family | Previous selection risk | Wording now exposed to the agent |
| :--- | :--- | :--- |
| `semantic_verify` | A reader could treat verification as read-only. | Go runs formatting before diagnostics and can write files; selected Java formatting/import actions can write; there is no dry-run. |
| `semantic_batch` | “Stop at first failure” did not say prior edits remain. | Earlier successful edits stay written if a later edit fails; inspect the returned final diff. |
| Maven compile/test | Fixed goals were clear, but trust, offline mode, and writes were understated. | `trust_workspace=true` is required, network is opt-in, Maven writes build output and temporary data under `.scratch`. |
| Imports and dependencies | Agents could omit `file` or confuse an import with a module. | Omitting `file` organizes imports across the workspace; dependency addition can use the network and update both `go.mod` and `go.sum`. |
| Assertion rewrite and rename | Trust/preview and selected-file scope were easy to miss. | Assertion `dry_run=true` previews without trust or writes; Rust/Java rename requires a selected file and workspace trust. |
| Declaration insertion | The generic and specialized descriptions overlapped. | Functions/methods, types, and const/var declarations route to their dedicated insertion tools; generic insertion is for placement controls that fit better. |
| Relative paths | A worktree agent could assume the task directory is the server root. | Paths resolve from the active MCP workspace root; an explicit worktree path is needed when that root differs. |

The wording audit still found contract gaps that prose alone cannot fix. `tools/list` omits the active canonical root and does not emit the registry's `ExampleRaw` examples. Several operation schemas advertise broad language enums that their handlers do not implement; Rust/Java rename's selected-file requirement is described but not encoded conditionally. A future catalog test should reject unsupported language-operation pairs and expose a worked `loop_path`, nested batch, and worktree-root example. Read-only source excerpts/references and current diagnostics remain absent, while `semantic_verify` is a formatting action. The missing field, statement, composite-element, expression, signature, and comment edits remain structural capability gaps; the replay cases above show their exact target trees.

## ADR-0046 Replay 3 Findings

A third Luna high worker started from `d9226f9` in `.scratch/worktrees/adr0046-replay-3` with the `46d5816` promoted binary and a fresh full-profile, live-reload MCP process exposing 22 tools. It reconstructed the declaration/loop operations with CLI and MCP batch txtar trees. Full `go test ./...`, the targeted construct and batch cases, and `git diff --check` passed. `make check` reported only the four lint failures present before edits (funlen, gocognit, gosec, ST1005). The replay remains isolated and uncommitted.

The fresh `semantic_insert_decl` successfully inserted a documented sentinel into the existing `var (...)` block, preserving both its own comment and the prior sentinel's comment. `semantic_verify` found zero diagnostics. The grouped-spec `semantic_replace_decl` failure below was reproduced against the older replay binary and is addressed in `bf6e142`; the next fresh-server pass must verify it end to end.

- Intent: route `InsertDecl` through package-level collision validation before it writes a declaration, and wire the new construct and declaration operations into existing registry code.
- Semantic route attempted: `semantic_lookup` located `InsertDecl` in `internal/astedit/decl.go`. The fresh 22-tool catalog has no source-read operation. `semantic_replace_body` can replace a known body but cannot reveal the current body, so applying it safely would require recovering the complete body first.
- Literal example: before, `InsertDecl` parsed a snippet and continued to group-append or insert it; after, `InsertDecl` parses the target, invokes `handleInsertDeclCollisions(...)`, returns the collision error before mutation, and only then follows the existing insertion path.
- Fallback: read the exact target implementation after documenting this constraint, then make the smallest atomic AST-aware source change; keep new Go declarations on semantic insertion tools wherever possible.
- Fixture-only fallback: `semantic_replace_decl` was called with `{file: ".scratch/replay-commented-collision.go", symbol: "ErrReplayBase", source: "// ErrReplayBase is the pre-existing grouped sentinel.\nvar ErrReplayBase = errors.New(\"base\")"}` to replace one spec within an existing grouped var, not the group. It returned `syntax error: 7:1: expected 'IDENT', found 'var'`; the file was unchanged by the failed call. The fixture's literal before/after is `ErrReplayBase = errors.New("base")` becoming `// ErrReplayBase is the pre-existing grouped sentinel.\nErrReplayBase = errors.New("base")`. Apply only that comment attachment atomically, then verify.

- Intent: add `DeclOptions.Overwrite` and run package collision checks before any grouped or standalone insertion.
- Semantic route attempted: `semantic_lookup` found `InsertDecl`; the catalog has no field-level AST insertion or source-read operation. `semantic_replace_body` requires a complete known body, so the field and insertion guard need a documented narrow fallback.
- Literal before/after: `type DeclOptions struct { TargetSymbol string; AutoOrganizeImports bool }` becomes `type DeclOptions struct { TargetSymbol string; Overwrite bool; AutoOrganizeImports bool }`; the current parse fallback `if err != nil && fileNode == nil { return appendToEOF(...) }` becomes a contextual parse error, followed by `handleInsertDeclCollisions(...)` and an early return when it performs an overwrite.
- Fallback: after reading the exact target, atomically add the field and replace only that parse/collision gate; keep the rest of `InsertDecl` unchanged.

- Intent: expose `overwrite` on `semantic_insert_decl` so the operation request reaches `DeclOptions.Overwrite`.
- Semantic route attempted: existing struct fields and a slice member do not have a dedicated semantic insertion tool. `semantic_replace_body` applies to functions, not fields or variable initializer lists.
- Literal before/after: `InsertDeclReq` gains `Overwrite bool` after `TargetSymbol`; `insertDeclParams` gains `{Name:"overwrite", CLIName:"overwrite", JSONName:"overwrite", Type:ParamBoolean, Description:"Replace an existing declaration with the same name (default false)}` after the auto-import parameter.
- Fallback: atomically add these two narrow declarations; update `parseInsertDecl` and `runInsertDecl` through semantic_replace_body.

- Intent: add required file-purpose and exported API comments to the new AST and operation files.
- Semantic route attempted: the fresh catalog has no comment insertion/edit operation for an existing type or declaration; semantic type/function insertion does not attach comments to an existing symbol.
- Literal before/after: `package astedit` becomes a WHY package comment followed by `package astedit`; `type LoopOptions struct { ... }` gains `// LoopOptions configures construct-level loop replacement.`; `func ReplaceDecl(...)` gains `// ReplaceDecl replaces one existing package-level constant, variable, or type alias.` Similar Go doc comments are added to exported request types and project-context methods in `wire_engine_construct.go`.
- Fallback: atomically add only those comments and package purpose comments; implementations remain semantic-tool edits.

- Intent: give the new external-package AST tests a file-purpose comment.
- Semantic route attempted: no MCP tool edits package clauses or file headers after scaffolding.
- Literal before/after: `package astedit_test` becomes `// Package astedit_test verifies construct-level AST replacement and collision behavior.\npackage astedit_test`.
- Fallback: atomically add only the file-purpose comment.

The retained grouped-comment fixture result is:

Before insertion:

```go
var (
 // ErrReplayBase is the pre-existing grouped sentinel.
 ErrReplayBase = errors.New("base")
)
```

After `semantic_insert_decl` inserted the commented sentinel:

```go
var (
 // ErrReplayBase is the pre-existing grouped sentinel.
 ErrReplayBase = errors.New("base")
 // ErrSymbolCollision reports when an insertion would duplicate a package declaration.
 ErrSymbolCollision = errors.New("symbol collision")
)
```

## Fresh-Server ADR-0046 Replay (Iteration 4)

A fourth Luna high worker started from the original `d9226f9` baseline in `.scratch/worktrees/adr0046-replay-4`, with the Go cache copied into that worktree. It launched the promoted `mcp --profile full --live-reload` binary as a fresh stdio process and confirmed all 22 catalog descriptions start with `Use this tool instead of`. The original `d9226f9..bce450f` diff was context, not a patch applied to the replay. The replay worktree remains uncommitted and isolated; the canonical implementation on main has broader AST tests and `loop_path` disambiguation.

The worker reconstructed the loop and declaration operations, overwrite and collision behavior, registry/CLI/MCP wiring, and literal CLI/MCP batch txtar trees. A full `go test ./...` passed before its final cross-file collision assertion; afterward the targeted script and `internal/astedit`, `internal/operation`, and `internal/mcp` packages passed. `git diff --check` passed. Final `make check` completed Markdown lint and doc generation, then stopped only on the same four lint findings measured on the baseline: `driver.go` funlen, `benchmark_aggregate.go` gocognit, and `agy_driver.go` gosec and ST1005. The replay did not add focused AST unit tests or `loop_path` support, so its green public cases are evidence of tool fit rather than a replacement for the reviewed main implementation.

| Replay intent and literal example | Tool used or precise fallback |
| :--- | :--- |
| Create `replace_loop.go` and `replace_decl.go`, then define operations and registry handlers | `semantic_scaffold_file`, `semantic_insert_type`, `semantic_insert_function`, and `semantic_insert_decl` created the source structure. The empty scratch probe initially needed an explicit package (ST-0063); the production files had inferable packages. |
| Update the existing `insertDeclParams` composite value | `semantic_replace_decl` succeeded. Before, the final adjacent entries were `target_symbol` and `auto_organize_imports`; after, `{Name: "overwrite", Type: ParamBoolean, Default: false}` appears between them while both siblings remain. This is an edit the earlier workers could and should have made semantically once the fresh server exposed the operation. A future composite-element tool would avoid supplying the full initializer. |
| Add `Overwrite bool` to `DeclOptions` and `InsertDeclReq` | `semantic_replace_decl` rejected the struct target with `requires a type alias` (ST-0066). Before: `type DeclOptions struct { TargetSymbol string; AutoOrganizeImports bool }`. After: the same fields plus `Overwrite bool` between them. The worker applied a narrow atomic field edit; `semantic_insert_field` is the missing operation. |
| Add collision checking before `InsertDecl` writes and update operation parsing/handlers | `semantic_insert_function` added the replay helper; `semantic_replace_body` rewired existing function bodies. A scoped statement insertion at a named AST anchor would avoid rewriting an entire body for one guard. |
| Add `ErrSymbolCollision` to the existing error declarations | `semantic_insert_decl` handled the new sentinel. The grouped comment-preservation defect from replay 2 was already fixed and verified by replay 3; the fourth replay did not need a text repair here. |
| Add package WHY headers, Go doc comments, a `#nosec` annotation, and one local slice preallocation | The current catalog has no package/header comment, attached comment, or local expression/statement edit. Before `var out []string`; after `out := make([]string, 0, len(s.Names))`. These were bounded atomic edits recorded in the replay worktree's `docs/MANUAL_EDITS.md`. |
| Change the registry-count assertion and create `.txtar` CLI/MCP fixtures | A whole test-body replacement would be disproportionate for one expected count; no Go semantic operation edits txtar archive sections. Atomic edits preserved the literal before/after source trees, including a rejected cross-file collision that leaves both files unchanged. |

The fourth replay also confirmed that `semantic_replace_decl` now accepts a documented full declaration when targeting one grouped var spec: it changed only the selected spec and its comment, leaving the sibling spec and comment byte-for-byte intact. The first exact-byte expectation differed only in `gofmt` spacing; another probe reused a package-level symbol and was correctly rejected (ST-0064). One malformed registry definition snippet was rejected before mutation (ST-0065). These are client or fixture errors, distinct from the unsupported struct-field operation.

**Post-commit tool-use decision:** The original ADR-0046 implementation is already committed on main. With today's promoted server, new declarations/files, existing function bodies, loop nodes, package const/var/alias values, imports, and batchable sequences should be routed through the corresponding semantic tools. The fourth replay demonstrates the formerly manual `insertDeclParams` update can use `semantic_replace_decl`. Remaining proportional operations are read-only source/outline/references and diagnostics, explicit active-root reporting, struct-field insertion, anchored statements, composite elements, expressions, signatures, and attached comments/file headers. The catalog still needs worked schema examples, capability-filtered language choices, and unknown-parameter rejection so a guessed key cannot silently trigger a default-scope operation. Each proposed operation can use literal before/after trees from this repository as regression fixtures.
