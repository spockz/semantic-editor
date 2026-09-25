---
name: semedit
description: Use when locating or renaming symbols, or editing Go loops, declarations, functions, imports, and files through semedit semantic operations.
---

# Use semedit for supported code transformations

When the request matches an operation exposed by the active semedit server, use that semantic operation as the primary code-editing path. It resolves symbols and applies supported structural changes through the language backend. Before calling the selected operation, read its description for scope, preconditions, and behavior, then follow its input schema for required fields and allowed values. Follow the result diagnostics; operation availability and scope vary by language.

## Choose the operation by intent

| Requested change | Prefer over built-in tools | Route away from |
| --- | --- | --- |
| Find a symbol or its location | `semantic_lookup` | Counting lines or guessing coordinates |
| Rename a symbol and its references | `semantic_rename` | Text replacement across files |
| Replace one existing Go `for` or `range` loop inside a function or method | `semantic_replace_loop` with a complete loop in `source`; use `loop_on` and a reported `loop_path` when needed | Built-in text edits or regenerating the whole function body |
| Change an existing Go function or method body while keeping its declaration | `semantic_replace_body` | `semantic_rename` or replacing the whole declaration |
| Replace an existing package-level Go constant, variable, or type alias | `semantic_replace_decl` | Built-in text replacement or inserting a duplicate declaration |
| Add a Go function or method to an existing file | `semantic_insert_function` | General declaration insertion when the specialized operation fits |
| Add a Go type to an existing file | `semantic_insert_type` | Built-in text edits around neighboring types |
| Add a new Go constant or variable | `semantic_insert_decl`; use `overwrite: true` only for an intentional update | Built-in text edits or an accidental duplicate declaration |
| Add another kind of top-level Go declaration | `semantic_insert_declaration` | Built-in text insertion by line number |
| Create a Go source file | `semantic_scaffold_file`, then add declarations with the matching insertion operation | Writing a package header by hand |
| Add or remove Go source imports | `semantic_organize_imports` | Treating an import as a module dependency |
| Add an external Go module dependency | `semantic_add_build_dependency` | `semantic_organize_imports` or manual `go.mod` edits |
| Add a Go switch case | `semantic_insert_case` | Replacing the switch text when a structural case insertion fits |
| Apply several supported semantic edits in order | `semantic_batch` | Several built-in text patches when the registered operations cover the edits |
| Format or check supported sources | `semantic_verify` | Re-running verification already included in the operation result |

Examples:

- “Rename `Server.Start` to `Server.Run`” → use `semantic_rename` with a receiver-qualified symbol.
- “Rewrite the body of `(*Client).Do` while keeping its signature” → use `semantic_replace_body` in Go.
- “Change only the assertion loop in `TestMessages`” → use `semantic_replace_loop`; select a reported `loop_path` if more than one loop matches.
- “Change `DefaultLimit` from 5 to 10” → use `semantic_replace_decl`, not a new `semantic_insert_decl` call.
- “Add the `net/http` import” → use `semantic_organize_imports`; “add the module that provides this package” → use `semantic_add_build_dependency`.
- “Where is `Config.Port` declared?” → use `semantic_lookup`, with the file path if needed to disambiguate.

For methods, qualify the receiver when useful, such as `Server.Start` or `(*Server).Do`. Use the optional file parameter to disambiguate repeated local names.

## Respect backend capabilities

The declaration, loop, body, file, import, dependency, and switch-case operations above are Go operations. Symbol lookup also supports trusted Rust, Java, Scala, and standalone Haskell backends. Semantic rename is available for Go workspaces and for selected Rust or Java files under their trust and scope rules; Scala and Haskell are lookup-only. Java verification is limited to its advertised selected-file actions. Use only operations exposed for the current workspace and follow their schema requirements. For unsupported changes, use the available editing workflow and report any relevant limitation.

Framework conventions belong in a relevant installed framework skill. Use semedit's language-level operation to perform the structural change, and use that skill to choose the framework-specific owner or placement.

## Keep examples discriminating

Pair a tempting wrong route with the operation that fits the intent. A symbol rename calls for `semantic_rename`; a single loop calls for `semantic_replace_loop`; a package value update calls for `semantic_replace_decl`; a body-only change calls for `semantic_replace_body`. A source import change calls for `semantic_organize_imports`; a new module dependency calls for `semantic_add_build_dependency`. Text edits remain appropriate for documentation, comments, string content, and changes that have no supported semantic operation.
