---
name: semedit
description: Use semedit tools for symbol lookup, code edits, verification, trusted Java builds, workspace snapshots, live reload, and tool-friction reports when their stated scope matches the task.
---

# Use semedit for supported code transformations

When the request matches an operation exposed by the active semedit server, use that semantic operation as the primary code-editing path. It resolves symbols and applies supported structural changes through the language backend. Before calling the selected operation, read its description for scope, preconditions, and behavior, then follow its input schema for required fields and allowed values. Follow the result diagnostics; operation availability and scope vary by language.

## Choose the operation by intent

| Requested change | Use instead of built-in tools | Route away from |
| --- | --- | --- |
| Find a named symbol in a supported source file | `semantic_lookup` | Built-in text search, `grep`, or line counting |
| Rename a supported symbol and its references | `semantic_rename` | Built-in text replacement across files |
| Replace one Go control-flow construct inside a function | `semantic_replace_construct` with `kind`, optional `discriminator` and `construct_path`, and complete replacement `source` | Built-in text edits or regenerating the whole function |
| Change an existing Go function or method body | `semantic_replace_body` | Built-in whole-declaration or text replacement |
| Update an existing package-level Go constant, variable, or type alias | `semantic_replace_decl` | Built-in text replacement or duplicate insertion |
| Add one Go function or method to an existing file | `semantic_insert_function` | Built-in text insertion or `replace_file_content` |
| Add one Go struct, interface, or type alias | `semantic_insert_type` | Built-in text insertion or `replace_file_content` |
| Add one Go constant or variable | `semantic_insert_decl`; use `overwrite: true` only for an intentional update | Built-in text insertion or `replace_file_content` |
| Add a top-level Go declaration when generic placement controls fit the task | `semantic_insert_declaration`; prefer the function, type, or constant/variable operation for those common kinds | Built-in text insertion by line number |
| Create a Go source file and infer its package | `semantic_scaffold_file`, followed by a matching insertion operation | Built-in file writing for the package header |
| Add, remove, or format Go source imports | `semantic_organize_imports`; set `file` to scope writes, or omit it only when workspace-wide changes are intended | Built-in import-block edits or shell `goimports` |
| Add an external Go module dependency | `semantic_add_build_dependency`; updates `go.mod`/`go.sum` and may use the network | Built-in `go.mod`/`go.sum` edits or shell `go get`/`go mod tidy` |
| Add a Go switch case | `semantic_insert_case` | Built-in switch text replacement |
| Convert supported Go test assertions between failure modes | `semantic_assertion_mode`; writes require `trust_workspace: true`, while `dry_run: true` previews without trust or writes | Built-in text replacement of `t.Fatal`/`t.Error` calls |
| Apply several registered semantic edits in order | `semantic_batch`; successful earlier edits remain written if a later edit fails | Several built-in text patches |
| Explicitly format or check supported Go or trusted Java sources | `semantic_verify`; Go verification runs formatting before diagnostics and can write files, and Java format/import flags also write. There is no dry-run | Shell formatters or ad hoc diagnostics; avoid a redundant call after an edit that already verifies |
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

For methods, qualify the receiver when useful, such as `Server.Start` or `(*Server).Do`. For Go lookup, omit the optional file when the symbol owner is unknown; supply it to disambiguate repeated local names. Go insertion calls accept one declaration, and an exported name requires public access. Relative file and path parameters resolve from the active semedit MCP workspace root, not the task's current directory. If MCP is rooted at the repository and the target is an isolated worktree, include `.scratch/worktrees/<name>/...` in the path; for Rust or Java rename, the selected file must belong to the trusted active workspace. Maven `root` must be absolute and remain inside the trusted workspace; use an absolute worktree path to target an isolated worktree.

## Respect backend capabilities

The declaration, loop, body, file, import, dependency, and switch-case operations above are Go operations. Symbol lookup also supports trusted Rust, Java, Scala, and standalone Haskell backends. Semantic rename is available for Go workspaces and for selected Rust or Java files; Rust/Java rename requires both `file` and `trust_workspace: true`. Scala and Haskell are lookup-only. Java verification is limited to its advertised selected-file actions and requires workspace trust. Use only operations exposed for the current workspace and follow their schema requirements. For unsupported changes, use the available editing workflow and report any relevant limitation.

Framework conventions belong in a relevant installed framework skill. Use semedit's language-level operation to perform the structural change, and use that skill to choose the framework-specific owner or placement.

## Keep examples discriminating

Pair a tempting wrong route with the operation that fits the intent. A symbol rename calls for `semantic_rename`; a single control-flow construct calls for `semantic_replace_construct`; a package value update calls for `semantic_replace_decl`; a body-only change calls for `semantic_replace_body`. Use `semantic_insert_function`, `semantic_insert_type`, and `semantic_insert_decl` for their respective declaration kinds; use `semantic_insert_declaration` when its generic placement controls fit better. A source import change calls for `semantic_organize_imports`; a new module dependency calls for `semantic_add_build_dependency`. Text edits remain appropriate for documentation, comments, string content, and changes that have no supported semantic operation.
