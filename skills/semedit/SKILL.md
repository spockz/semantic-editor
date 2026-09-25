---
name: semedit
description: Use semedit tools for symbol lookup, code edits, verification, trusted Java builds, workspace snapshots, live reload, and tool-friction reports when their stated scope matches the task.
---

# Use semedit for supported code transformations

When the request matches an operation exposed by the active semedit server, use that semantic operation as the primary code-editing path. It resolves symbols and applies supported structural changes through the language backend. Before calling the selected operation, read its description for scope, preconditions, and behavior, then follow its input schema for required fields and allowed values. Follow the result diagnostics; operation availability and scope vary by language.

## Choose the operation by intent

| Requested change | Prefer over built-in tools | Route away from |
| --- | --- | --- |
| Find a named symbol in a supported source file | `semantic_lookup` | Built-in text search, `grep`, or line counting |
| Rename a supported symbol and its references | `semantic_rename` | Built-in text replacement across files |
| Replace one Go `for` or `range` loop inside a function | `semantic_replace_loop` with complete loop `source` | Built-in text edits or regenerating the whole function |
| Change an existing Go function or method body | `semantic_replace_body` | Built-in whole-declaration or text replacement |
| Update an existing package-level Go constant, variable, or type alias | `semantic_replace_decl` | Built-in text replacement or duplicate insertion |
| Add a Go function or method to an existing file | `semantic_insert_function` | Built-in text insertion around adjacent declarations |
| Add a Go type to an existing file | `semantic_insert_type` | Built-in text insertion around adjacent types |
| Add a new Go constant or variable | `semantic_insert_decl`; use `overwrite: true` only for an intentional update | Built-in text edits or accidental duplicate declarations |
| Add another kind of top-level Go declaration | `semantic_insert_declaration` | Built-in text insertion by line number |
| Create a Go source file and infer its package | `semantic_scaffold_file`, followed by a matching insertion operation | Built-in file writing for the package header |
| Add, remove, or format Go source imports | `semantic_organize_imports` | Built-in import-block edits or shell `goimports` |
| Add an external Go module dependency | `semantic_add_build_dependency` | Built-in `go.mod` edits or shell `go get` |
| Add a Go switch case | `semantic_insert_case` | Built-in switch text replacement |
| Convert supported Go test assertions between failure modes | `semantic_assertion_mode` | Built-in text replacement of `t.Fatal`/`t.Error` calls |
| Apply several registered semantic edits in order | `semantic_batch` | Several built-in text patches |
| Explicitly format or check supported Go or trusted Java sources | `semantic_verify` | Shell formatters or ad hoc diagnostics; avoid a redundant call after an edit that already verifies |
| Compile tests for a trusted Java root POM | `semantic_maven_compile` | Shell Maven invocation |
| Run tests for a trusted Java root POM | `semantic_maven_test` | Shell Maven invocation |
| Save a pre-edit rollback point for a bounded edit | `semantic_snapshot` | `git stash` or manual backup copies |
| Restore edits from a semedit snapshot | `semantic_undo` | `git reset` or manual file restoration |
| Reload a promoted MCP binary when live reload is enabled | `semantic_reload` | Restarting the client or server manually |
| Prepare a privacy-reviewed semedit tool-friction draft | `report_feedback` | Manually drafting a tool issue; keep local friction logs required by this repository |

Examples:

- “Rename `Server.Start` to `Server.Run`” → use `semantic_rename` with a receiver-qualified symbol.
- “Rewrite the body of `(*Client).Do` while keeping its signature” → use `semantic_replace_body` in Go.
- “Change only the assertion loop in `TestMessages`” → use `semantic_replace_loop`; select a reported `loop_path` if more than one loop matches.
- “Change `DefaultLimit` from 5 to 10” → use `semantic_replace_decl`, not a new `semantic_insert_decl` call.
- “Add the `net/http` import” → use `semantic_organize_imports`; “add the module that provides this package” → use `semantic_add_build_dependency`.
- “Where is `Config.Port` declared?” → use `semantic_lookup`, with the file path if needed to disambiguate.

For methods, qualify the receiver when useful, such as `Server.Start` or `(*Server).Do`. For Go lookup, omit the optional file when the symbol owner is unknown; supply it to disambiguate repeated local names. Go insertion calls accept one declaration, and an exported name requires public access. In an isolated worktree, confirm the target through a read-only lookup and use an explicit worktree path for edits because the active MCP root may differ from the task directory.

## Respect backend capabilities

The declaration, loop, body, file, import, dependency, and switch-case operations above are Go operations. Symbol lookup also supports trusted Rust, Java, Scala, and standalone Haskell backends. Semantic rename is available for Go workspaces and for selected Rust or Java files under their trust and scope rules; Scala and Haskell are lookup-only. Java verification is limited to its advertised selected-file actions. Use only operations exposed for the current workspace and follow their schema requirements. For unsupported changes, use the available editing workflow and report any relevant limitation.

Framework conventions belong in a relevant installed framework skill. Use semedit's language-level operation to perform the structural change, and use that skill to choose the framework-specific owner or placement.

## Keep examples discriminating

Pair a tempting wrong route with the operation that fits the intent. A symbol rename calls for `semantic_rename`; a single loop calls for `semantic_replace_loop`; a package value update calls for `semantic_replace_decl`; a body-only change calls for `semantic_replace_body`. A source import change calls for `semantic_organize_imports`; a new module dependency calls for `semantic_add_build_dependency`. Text edits remain appropriate for documentation, comments, string content, and changes that have no supported semantic operation.
