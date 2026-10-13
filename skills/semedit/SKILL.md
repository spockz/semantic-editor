---
name: semedit
description: Use semedit tools for symbol lookup, code inspection, outlines, references, code edits, verification, trusted Java builds, workspace snapshots, live reload, and tool-friction reports when their stated scope matches the task.
---

# Use semedit for supported code transformations

When the request matches an operation exposed by the active semedit server, use that semantic operation as the primary code-editing path. It resolves symbols and applies supported structural changes through the language backend. Before calling the selected operation, read its description for scope, preconditions, and behavior, then follow its input schema for required fields and allowed values. Follow the result diagnostics; operation availability and scope vary by language.

## Choose the operation by intent

| Requested change | Use instead of built-in tools | Route away from |
| --- | --- | --- |
| Discover packages, synopses, and module boundaries across the workspace | `semantic_workspace_outline` with optional `path` and `include_tests` | Directory listing, `find`, or `tree` shell exploration |
| Find a named symbol in Go, Java, Rust, Scala, or Haskell | `semantic_lookup` | Built-in text search, `grep`, or line counting |
| Inspect a Go symbol's declaration, signature, doc comment, and imports | `semantic_inspect_symbol` | Raw file reading or guessing signatures |
| Inspect matching nested Go constructs before selecting an ambiguous target | `semantic_inspect_construct`; copy a candidate path and selection revision from the same result | Guessing structural paths or retrying unchanged selectors |
| View declaration hierarchy in Go, Java, Kotlin, Rust, Scala, Makefile, or Bash | `semantic_outline` with optional `kinds` and `include_unexported` (use `semantic_workspace_outline` first to discover packages) | Reading whole files or manual grep navigation |
| Find usages, call sites, or references of a Go symbol | `semantic_find_references` | Broad text search or regex |
| Rename a symbol and its references in Go, or in selected trusted Rust or Java files | `semantic_rename` | Built-in text replacement across files |
| Replace one Go control-flow construct, clause, or statement/expression inside a function | `semantic_replace_construct` with `kind`, optional `discriminator` and `construct_path`, and complete replacement `source` | Built-in text edits or regenerating the whole function |
| Insert one Go control-flow construct, loop, branch, clause, or statement/expression inside a function | `semantic_insert_construct` with `kind`, `in_function`, and `source`; statements use `anchor` (`start`, `end`, `before_return`, `before`, or `after`), while clauses use `placement` | Built-in text edits or line insertion |
| Append a keyed map/dictionary entry or struct/object initializer property in Go | `semantic_insert_construct` with `kind: "element"`, `in_declaration`, exact `discriminator`, and `source`; copy `expected_revision` from inspection when available | `kind: "field"` changes struct type declarations; positional slice/array items are unsupported |
| Change an existing Go function or method body | `semantic_replace_body` | Built-in whole-declaration or text replacement |
| Extract contiguous Go call statements with named-parameter data flow | `semantic_extract_function` with `kind`, `start_line`, and `end_line`; refusals include aliased inputs, direct `recover()`, and uncertain pointer-receiver dispatch | Copying statements into a new helper manually |
| Extract one exact typed whole argument from a direct single-argument Go call statement | `semantic_extract_variable` with `symbol`, `in_function`, and exact-source `discriminator`; use `construct_path` when the expression repeats | Rewriting the call around an inferred local manually |
| Update an existing package-level Go constant, variable, or type alias | `semantic_replace_decl` | Built-in text replacement or duplicate insertion |
| Add one Go function or method to an existing file | `semantic_insert_function` | Built-in text insertion or `replace_file_content` |
| Add one Go struct, interface, or type alias | `semantic_insert_type` | Built-in text insertion or `replace_file_content` |
| Add one field to a private non-generic Go struct type | `semantic_insert_construct` with `kind: "field"`, `in_type`, and one named field in `source`; typed overlay checks guard layout compatibility | Exported owners and duplicate fields are refused |
| Add one Go constant or variable | `semantic_insert_decl` | Built-in text insertion or `replace_file_content` |
| Add a top-level Go declaration when generic placement controls fit the task | `semantic_insert_declaration`; prefer the function, type, or constant/variable operation for those common kinds | Built-in text insertion by line number |
| Create a Go source file and infer its package | `semantic_scaffold_file`, followed by a matching insertion operation | Built-in file writing for the package header |
| Add, remove, or format Go source imports | `semantic_organize_imports`; set `file` to scope writes, or omit it only when workspace-wide changes are intended | Built-in import-block edits or shell `goimports` |
| Add an external Go module dependency | `semantic_add_build_dependency`; updates `go.mod`/`go.sum` and may use the network | Built-in `go.mod`/`go.sum` edits or shell `go get`/`go mod tidy` |
| Add a Go switch case | `semantic_insert_case` | Built-in switch text replacement |
| Convert Go test assertions (t.Fatal/t.Error) between failure modes | `semantic_assertion_mode`; writes require `trust_workspace: true`, while `dry_run: true` previews without trust or writes | Built-in text replacement of `t.Fatal`/`t.Error` calls |
| Apply several registered semantic edits in order | `semantic_batch`; successful earlier edits remain written if a later edit fails | Several built-in text patches |
| Explicitly format or check Go or trusted Java sources | `semantic_verify`; Go verification runs formatting before diagnostics and can write files, and Java format/import flags also write. There is no dry-run | Shell formatters or ad hoc diagnostics; avoid a redundant call after an edit that already verifies |
| Compile tests for a trusted Java root POM | `semantic_maven_compile`; set `trust_workspace: true`, runs offline unless `allow_network: true`, and writes build outputs plus temporary Maven data under `.scratch` | Shell Maven invocation |
| Run tests for a trusted Java root POM | `semantic_maven_test`; set `trust_workspace: true`, runs offline unless `allow_network: true`, and writes build outputs plus temporary Maven data under `.scratch` | Shell Maven invocation |
| Save a pre-edit rollback point for a bounded edit | `semantic_snapshot` | `git stash` or manual backup copies |
| Restore edits from a semedit snapshot | `semantic_undo` | `git reset` or manual file restoration |
| Reload a promoted MCP binary when live reload is enabled | `semantic_reload` | Restarting the client or server manually |
| Prepare a privacy-reviewed semedit tool-friction draft | `report_feedback` | Manually drafting a tool issue; keep local friction logs required by this repository |

Examples:

- “Rename `Server.Start` to `Server.Run`” → use `semantic_rename` with a receiver-qualified symbol.
- “Rewrite the body of `(*Client).Do` while keeping its signature” → use `semantic_replace_body` in Go.
- “Change only the assertion loop in `TestMessages`” → use `semantic_replace_construct` with `kind: "loop"`; select a reported `construct_path` if more than one loop matches.
- “Change `DefaultLimit` from 5 to 10” → use `semantic_replace_decl`, not a new `semantic_insert_decl` call.
- “Add the `net/http` import” → use `semantic_organize_imports`; “add the module that provides this package” → use `semantic_add_build_dependency`.
- “Where is `Config.Port` declared?” → use `semantic_lookup`, with the file path if needed to disambiguate.

For methods, qualify the receiver when useful, such as `Server.Start` or `(*Server).Do`. For Go lookup, omit the optional file when the symbol owner is unknown; supply it to disambiguate repeated local names. Go insertion calls accept one declaration. Omitted access follows identifier casing; explicit access must agree with that casing. Relative file and path parameters resolve from the active semedit MCP workspace root, not the task's current directory. If MCP is rooted at the repository and the target is an isolated worktree, include `.scratch/worktrees/<name>/...` in the path; for Rust or Java rename, the selected file must belong to the trusted active workspace. Maven `root` must be absolute and remain inside the trusted workspace; use an absolute worktree path to target an isolated worktree.

### Copy typed results and respect publication

Operations with declared output contracts return a typed `result` object. Copy
target and selector fields only from its declared continuations and keep values
from the same record together. Formatted text is presentation, not a parallel
source of continuation fields. Mutations return `edit_receipt`; buffered batch
children report `applied_working` and omit live continuations. A batch failure
can retain earlier successful edits, so inspect its persisted diff before retrying.
Standalone AST edits prepare edits, formatting, and imports in one source buffer
and publish the prepared result once.

When the advertised Go tools include retained staging, use `semantic_stage` for
edits that span turns and should remain unpublished. Staged reads see the working
revision. Copy a mutation's native `stage_receipt.step_id` to `semantic_stage_drop`,
or use `semantic_stage_abort` to discard the journal; `semantic_commit` publishes
and verifies it. Ordinary `result` continuations are not live while staged. Read
[the Go publication guide](references/golang.md#publication-and-verification)
for conflicts, limits, partial publication, and verification choices.

### Direct targeting over preliminary lookup

When the target file and symbol name are already known from the task description or an error trace, call `semantic_rename`, `semantic_replace_body`, or `semantic_replace_construct` directly with `file` and a short `symbol`, such as `file: "server.go", symbol: "Start"` or `symbol: "Server.Start"`. Use `semantic_lookup` first only when the file location or symbol definition is genuinely unknown.

## Respect backend capabilities

The declaration, loop, body, file, import, dependency, and switch-case operations above are Go operations. Symbol inspection and reference discovery currently support Go workspaces. Declaration outline supports Go, Java, Kotlin, Rust, Scala, Makefile, and Bash backends. Symbol lookup supports Go, trusted Rust, Java, Scala, and standalone Haskell backends. Semantic rename is available for Go workspaces and for selected Rust or Java files; Rust/Java rename requires both `file` and `trust_workspace: true`. Scala and Haskell are lookup-only. Java verification is limited to its advertised selected-file actions and requires workspace trust. Use only operations exposed for the current workspace and follow their schema requirements. For unsupported changes, use the available editing workflow and report any relevant limitation.

Framework conventions belong in a relevant installed framework skill. Use semedit's language-level operation to perform the structural change, and use that skill to choose the framework-specific owner or placement.

## Harness administration output

`install`, `integration status`, and `uninstall` print human-readable results by default. Pass `--json` when scripting these administrative commands; JSON is written to stdout without extra guidance. This flag does not change semantic-operation or MCP tool output.

## Core Operations vs. Macro Refactorings

`semedit` strictly demarcates low-level engine responsibilities from high-level architectural refactorings:

1. **Core Engine Primitives**: Zero-token, deterministic AST mutations and compiler/LSP queries (`semantic_lookup`, `semantic_outline`, `semantic_rename`, `semantic_replace_body`, `semantic_insert_construct`, `semantic_replace_construct`, `semantic_batch`). The engine executes exact syntactic transformations and enforces local compiler invariants; it does not implement framework-specific heuristics or whole-project modernization lifecycles.
2. **Skill-Orchestrated Macro Refactorings**: High-level, cross-paradigm, or idiomatic refactorings (such as decomposing classes, introducing design patterns, or modernizing language constructs) are composed by the agent using core operations in `semantic_batch`. Whole-project migrations that alter external serialization or ecosystem contracts belong in dedicated linting/upgrade tools rather than core engine commands.

### General Refactoring Examples

- **Encapsulate State**: Rather than a monolithic engine command, create accessors via declaration insertions, rewrite verified caller sites, and update field access atomically within a single `semantic_batch`.
- **Extract Abstraction / Delegate**: Derive interfaces or composite types from existing symbols, rewrite dependent call sites, and verify overlay clean state before completing.
- **Decompose Large Functions**: Extract pure expressions or straight-line statements into scoped helpers, updating callers with preserved argument evaluation order.

## Language-Specific Refactoring References

To conserve context budget across agent harnesses (OpenAI Codex, Google Antigravity, GitHub Copilot), **do not read language reference files unless actively modifying or refactoring source files in that specific language**:

- **Go**: Consult [references/golang.md](references/golang.md) only when refactoring Go types, signatures, or constructs.
- **Java**: Consult [references/java.md](references/java.md) only when refactoring Java classes, records, accessors, or Maven builds.
- **Rust**: Consult [references/rust.md](references/rust.md) only when refactoring Rust structs, newtypes, traits, or modules.
- **Scala**: Consult [references/scala.md](references/scala.md) only when inspecting or refactoring Scala symbols.
- **Kotlin**: Consult [references/kotlin.md](references/kotlin.md) only when inspecting or refactoring Kotlin symbols.
- **Haskell**: Consult [references/haskell.md](references/haskell.md) only when inspecting standalone Haskell symbols.
- **Bash**: Consult [references/bash.md](references/bash.md) only when inspecting or refactoring Bash scripts.
- **Makefile**: Consult [references/make.md](references/make.md) only when inspecting or refactoring Makefiles.

## Keep examples discriminating

To add `"metrics": metrics` to a Go map literal, use
`semantic_insert_construct` with `kind: "element"`, the inspection record's
`file`, `in_declaration`, and `expected_revision`, the exact current literal
as `discriminator`, and only the new keyed entry as `source`. A successful
standalone result may supply a refreshed `selector_continuations` record for
the next insertion; copy all companions from that record. Batch children omit
live records. Do not use `kind: "field"` for literal properties, or infer
positional list support from the generic kind name.

Pair a tempting wrong route with the operation that fits the intent. A symbol rename calls for `semantic_rename`; an inner control-flow construct or statement/expression insertion calls for `semantic_insert_construct`; a single construct, branch, clause, or statement/expression replacement calls for `semantic_replace_construct`; a package value update calls for `semantic_replace_decl`; a body-only change calls for `semantic_replace_body`. Use `semantic_insert_function`, `semantic_insert_type`, and `semantic_insert_decl` for their respective declaration kinds; use `semantic_insert_declaration` when its generic placement controls fit better. A source import change calls for `semantic_organize_imports`; a new module dependency calls for `semantic_add_build_dependency`. Text edits remain appropriate for documentation, comments, string content, and changes that have no supported semantic operation.
