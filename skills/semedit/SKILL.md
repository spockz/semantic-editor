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
| Inspect matching nested Go or Java constructs before selecting an ambiguous target | `semantic_inspect_construct`; copy a candidate path and selection revision from the same result | Guessing structural paths or retrying unchanged selectors |
| View declaration hierarchy in Go, Java, Kotlin, Rust, Scala, Makefile, or Bash | `semantic_outline` with optional `kinds` and `include_unexported` (use `semantic_workspace_outline` first to discover packages) | Reading whole files or manual grep navigation |
| Find usages, call sites, or references of a Go or trusted Java symbol | `semantic_find_references` | Broad text search or regex |
| Rename a symbol and its references in Go, or in selected trusted Rust or Java files | `semantic_rename` | Built-in text replacement across files |
| Replace one supported Go or Java method-local construct | `semantic_replace_construct` with `kind`, optional `discriminator` and `construct_path`, selection revision, and complete replacement `source` | Built-in text edits or regenerating the whole function |
| Remove one supported Go declaration/case or Java method, constructor, field declaration, nested type, or method-local construct | `semantic_remove_construct` with the exact declaration symbol or `in_function` plus `discriminator`; Java declaration deletion removes attached Javadoc and validates the source | Java `safe_delete` accepts only private declarations, requires complete JDT LS workspace reference coverage, and refuses if any usages remain |
| Insert one Go control-flow construct or one Java loop, conditional, switch, try-with-resources, synchronized block, statement, or expression inside a method | `semantic_insert_construct` with `kind`, `in_function`, and `source`; Go statements use `anchor` and clauses use `placement`; Java requires an explicit anchor and exact `discriminator` for `before`/`after` | Built-in text edits or line insertion; Java declaration and case insertion remain unsupported |
| Append a keyed Go map entry or Java `Map.of`/`Map.ofEntries`/`List.of`/`Set.of`/array initializer element | `semantic_insert_construct` with `kind: "element"`, `in_declaration`, exact initializer `discriminator`, and `source`; for Java maps pass a key/value pair to `Map.of` or one `Map.entry(...)` to `Map.ofEntries`; copy `expected_revision` when available | `kind: "field"` changes type declarations; positional Go slice/array items and Java collection mutations remain unsupported |
| Change an existing Go or Java function or method body | `semantic_replace_body` with the short or qualified method `symbol` and bare replacement statements | Built-in whole-declaration or text replacement |
| Extract contiguous Go call statements with named-parameter data flow | `semantic_extract_function` with `kind`, `start_line`, and `end_line`; refusals include aliased inputs, direct `recover()`, and uncertain pointer-receiver dispatch | Copying statements into a new helper manually |
| Extract one exact Go call argument or a Java expression into a local | `semantic_extract_variable` with `symbol`, `in_function`, and exact-source `discriminator`; Java uses the JDT LS `refactor.extract.variable` action and supports `construct_path` plus `expected_revision` for repeated expressions | Rewriting the call around an inferred local manually |
| Extract one exact Java constant expression into a class constant | `semantic_extract_constant` with `symbol`, `in_function`, and exact-source `discriminator`; Java uses JDT LS `refactor.extract.constant` and supports `construct_path` plus `expected_revision` for repeated expressions | Manually moving an expression into a field and replacing its uses |
| Extract contiguous Java statements into a method | `semantic_extract_function` with `in_function`, helper `symbol`, `kind: "method"`, and the selected statements’ 1-based `start_line` and `end_line`; Java validates direct statement boundaries and delegates to JDT LS | Copying statements manually or selecting partial/nested control flow |
| Inline one Java local variable | `semantic_inline_variable` with `symbol` and `in_function`; Java resolves the local declaration through the CST and delegates the refactoring to JDT LS `refactor.inline` | Replacing local references by text |
| Inline one Java method call | `semantic_inline_function` with the callee `file`/`symbol` and a closed `call_site` selecting the caller file, method, and exact call expression; Java delegates that call to JDT LS `refactor.inline` | Expanding a helper body manually at its caller |
| Implement a Java interface on an existing class | `semantic_implement_contract` with the class `file`/`symbol`, interface `contract` and optional `contract_file`; Java adds the interface when needed, then selects and generates missing methods with JDT LS `java/listOverridableMethods` and `java/addOverridableMethods` | Hand-writing method signatures or stubs |
| Extract a Java interface from selected methods | `semantic_extract_interface` with `file`, concrete `in_type`, new interface `symbol`, and exact `members`; Java accepts public instance methods and appends a package-private interface in the same source file | Copying methods or hand-building interface signatures |
| Change a Java method signature and update resolved callers | `semantic_change_signature` with a signature-qualified method `symbol` and the complete ordered `parameters` mapping; preserve existing entries by `source_index`, and give each added parameter a JDT LS-compatible `argument` expression | Editing a method declaration or its callers independently |
| Turn one exact Java method expression into a new parameter and update resolved callers | `semantic_introduce_parameter` with a signature-qualified method `symbol`, complete ordered `parameters`, the expression `discriminator`, and `construct_path` when the CST reports ambiguity | Replacing the expression and changing callers independently |
| Replace all parameters of a Java method with a value object | `semantic_introduce_parameter_object` with the complete `fields` mapping, a new `parameter` and type, and the existing same-file `destination_file`; the method and every direct caller must be in one package | Rewriting the signature and call sites independently or using a package-private type across packages |
| Move one declaration between Java types or Go files | `semantic_move_construct`; Java requires `destination_file` and the destination type in `target_symbol`, and delegates static-member movement plus reference updates to JDT LS; Go uses explicit placement options | Moving Java declarations with text edits or treating Java `target_symbol` as a Go placement anchor |
| Add a Java switch case | `semantic_insert_case` with the owning method, switch `discriminator`, complete case clause, placement, and inspected `construct_path` plus `expected_selection_revision` when ambiguous | Editing switch bodies by text or selecting a switch by line number |
| Update an existing Go declaration or a Java method, field, or nested type | `semantic_replace_decl`; Java replacements must preserve declaration kind and name | Built-in text replacement or duplicate insertion |
| Add one Go function or method to an existing file | `semantic_insert_function` | Built-in text insertion or `replace_file_content` |
| Add one Go struct, interface, or type alias | `semantic_insert_type` | Built-in text insertion or `replace_file_content` |
| Add one nested Java class, interface, enum, or record | `semantic_insert_type` with a complete nested declaration, optional `in_type`, and member placement | Java cannot infer the owner when the file declares multiple top-level types |
| Add one field to a private non-generic Go struct or a Java class, enum, or record | `semantic_insert_field` with `in_type` and a qualified `symbol`; Java `type_end` respects static/instance and visibility ordering | Go exported owners and duplicate fields are refused; Java insertion does not generate constructor updates |
| Set or remove a Java declaration's Javadoc | `semantic_set_doc_comment` with `language: "java"`, declaration `symbol`, and plain comment text; pass an empty string to remove the attached Javadoc | Editing comment text without checking declaration ownership |
| Add a Java method or constructor to a class, enum, record, or interface | `semantic_insert_function` with a complete declaration in `source`, optional `in_type`, and member placement | Java cannot infer the owner when the file declares multiple top-level types |
| Add one Go constant or variable | `semantic_insert_decl` | Built-in text insertion or `replace_file_content` |
| Add a top-level Go declaration when generic placement controls fit the task | `semantic_insert_declaration`; prefer the function, type, or constant/variable operation for those common kinds | Built-in text insertion by line number |
| Create a Go or Java source file and infer its package | `semantic_scaffold_file`, followed by a matching insertion operation; Java inference uses sibling `.java` files or `src/main/java` / `src/test/java` | Built-in file writing for the package header |
| Add, remove, or format Go imports, or update imports in one trusted Java file | `semantic_organize_imports`; Go supports file/workspace scopes, Java requires `language: "java"`, trust, and one `.java` file | Built-in import-block edits or shell `goimports` |
| Add a Go module or Java Maven dependency | `semantic_add_build_dependency`; Go updates `go.mod`/`go.sum`; Java updates a trusted reactor using `dependencyManagement` | Direct manifest edits or shell package-manager commands |
| Add a Go switch case | `semantic_insert_case` | Built-in switch text replacement |
| Convert Go test assertions (t.Fatal/t.Error) between failure modes | `semantic_assertion_mode`; writes require `trust_workspace: true`, while `dry_run: true` previews without trust or writes | Built-in text replacement of `t.Fatal`/`t.Error` calls |
| Apply several registered semantic edits in order | `semantic_batch`; successful earlier edits remain written if a later edit fails | Several built-in text patches |
| Explicitly format or check Go or trusted Java sources | `semantic_verify`; Go verification runs formatting before diagnostics and can write files, and Java accepts a file or directory scope for JDT LS formatting/import organization. There is no dry-run | Shell formatters or ad hoc diagnostics; avoid a redundant call after an edit that already verifies |
| Compile tests for a trusted Java root POM | `semantic_maven_compile`; set `trust_workspace: true`, runs offline unless `allow_network: true`, and writes build outputs plus temporary Maven data under `.scratch` | Shell Maven invocation |
| Run tests for a trusted Java root POM | `semantic_maven_test`; set `trust_workspace: true`, runs offline unless `allow_network: true`, and writes build outputs plus temporary Maven data under `.scratch` | Shell Maven invocation |
| Save a pre-edit rollback point for a bounded edit | `semantic_snapshot` | `git stash` or manual backup copies |
| Restore edits from a semedit snapshot | `semantic_undo` | `git reset` or manual file restoration |
| Reload a promoted MCP binary when live reload is enabled | `semantic_reload` | Restarting the client or server manually |
| Prepare a privacy-reviewed semedit tool-friction draft | `report_feedback` | Manually drafting a tool issue; keep local friction logs required by this repository |

Examples:

- “Rename `Server.Start` to `Server.Run`” → use `semantic_rename` with a receiver-qualified symbol.
- “Rewrite the body of `(*Client).Do` while keeping its signature” → use `semantic_replace_body` in Go; use the same operation with a Java method selector for Java.
- “Change only the assertion loop in `TestMessages`” → use `semantic_replace_construct` with `kind: "loop"`; select a reported `construct_path` if more than one loop matches.
- “Add a retry loop after the validation conditional in `LedgerService.post`” → inspect the Java method, then use `semantic_insert_construct` with `kind: "loop"`, `anchor: "after"`, and the exact conditional `discriminator`.
- “Change `DefaultLimit` from 5 to 10” → use `semantic_replace_decl`, not a new `semantic_insert_decl` call.
- “Change the initializer of `LedgerService.limit`” → use `semantic_replace_decl` with the complete Java field declaration, keeping its name and declaration kind.
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

When the advertised Go or Java tools include retained staging, use
`semantic_stage` for edits that span turns and should remain unpublished. Staged
reads see the working revision. Copy a mutation's native `stage_receipt.step_id`
to `semantic_stage_drop`, or use `semantic_stage_abort` to discard the journal;
`semantic_commit` publishes and verifies it. Ordinary `result` continuations are
not live while staged. Staging controls require one retained MCP session; their
standalone CLI commands fail with a session-required error. Read the relevant
language publication guide for conflicts, limits, partial publication, and
verification choices.

### Direct targeting over preliminary lookup

When the target file and symbol name are already known from the task description or an error trace, call `semantic_rename`, `semantic_replace_body`, or `semantic_replace_construct` directly with `file` and a short `symbol`, such as `file: "server.go", symbol: "Start"` or `symbol: "Server.Start"`. Use `semantic_lookup` first only when the file location or symbol definition is genuinely unknown.

## Respect backend capabilities

The declaration, removal, and switch-case mutations above are mostly Go operations. Java supports method-local construct removal and exact method, constructor, field declaration, and nested type deletion. Java `safe_delete` checks private visibility and fails closed unless JDT LS proves complete workspace reference coverage with no usages. Java can insert one field into a class, enum, or record through `semantic_insert_field`; fields are ordered by static/instance category and visibility. `semantic_insert_function` inserts one method or constructor, and `semantic_insert_type` inserts one nested type into an explicitly selected or uniquely inferred owner; both order by member tier and visibility, then JDT LS formats and verifies the result. Java method bodies can be replaced while preserving the declaration signature; method-local loop, conditional, switch, try-with-resources, synchronized, statement, and expression insertion plus supported construct replacement use the embedded pure-Go CST locator and JDT LS formatting, import organization, and verification. Java scaffolding infers the package from sibling Java sources or a Maven source path and requires no language-server process. `semantic_organize_imports` supports Go workspace scopes and a selected trusted Java file; Java accepts fully qualified type names for explicit add/remove requests and delegates formatting and import ordering to JDT LS. Symbol inspection supports Go workspaces and trusted Java files or directories; reference discovery supports Go workspaces and trusted selected Java files. Declaration outline supports Go, Java, Kotlin, Rust, Scala, Makefile, and Bash; Java accepts a file or directory path, supports public-only filtering, and can include test sources. Java lookup and inspection accept overload signatures such as `OrderService.process(Order, boolean)`; omitting parentheses returns overload candidates. Symbol lookup supports Go, trusted Rust, Java, Scala, and standalone Haskell backends. Semantic rename is available for Go workspaces and trusted Rust or Java files; Java validates cross-file edits and supports a standalone public-type file rename within its package directory. Buffered Java batches refuse resource renames. Rust/Java rename requires both `file` and `trust_workspace: true`. Scala and Haskell are lookup-only. Java verification is limited to its advertised selected-file actions and requires workspace trust. Trusted Java workspaces with a regular root `pom.xml` enable JDT LS Maven import by default; set `import_maven: false` to disable import. This imports the project in JDT LS and never runs Maven. Use only operations exposed for the current workspace and follow their schema requirements.

Framework conventions belong in a relevant installed framework skill. Use semedit's language-level operation to perform the structural change, and use that skill to choose the framework-specific owner or placement.

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
live records. For Java, select one field initializer with `in_declaration`
and its exact current factory call or array initializer as `discriminator`;
append one expression to `List.of`/`Set.of`/arrays, a key/value pair to
`Map.of`, or a `Map.entry(...)` to `Map.ofEntries`. JDT LS verifies the
resulting Java types. Do not use `kind: "field"` for initializer elements,
or infer support for mutable collection calls from the generic kind name.

Pair a tempting wrong route with the operation that fits the intent. A symbol rename calls for `semantic_rename`; an inner control-flow construct or statement/expression insertion calls for `semantic_insert_construct`; a single construct, branch, clause, or statement/expression replacement calls for `semantic_replace_construct`; a package value update calls for `semantic_replace_decl`; a body-only change calls for `semantic_replace_body`. Use `semantic_insert_function`, `semantic_insert_type`, and `semantic_insert_decl` for their respective declaration kinds; use `semantic_insert_declaration` when its generic placement controls fit better. A source import change calls for `semantic_organize_imports`; a new module dependency calls for `semantic_add_build_dependency`. Text edits remain appropriate for documentation, comments, string content, and changes that have no supported semantic operation.
