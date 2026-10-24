---
name: semedit
description: Use semedit tools for symbol lookup, code inspection, outlines, references, code edits, verification, trusted Java builds, workspace snapshots, live reload, and tool-friction reports when their stated scope matches the task.
---

# Use semedit for supported code transformations

When the request matches an operation exposed by the active semedit server, use that semantic operation as the primary code-editing path. It resolves symbols and applies supported structural changes through the language backend. Before calling the selected operation, read its description for scope, preconditions, and behavior, then follow its input schema for required fields and allowed values. Follow the result diagnostics; operation availability and scope vary by language.

Go edits validate cached diagnostic inputs and refresh affected packages and reverse
dependencies, with full-scan fallback for uncertain coverage. Explicit
`semantic_verify` remains a full verification request. A rename can retry one fresh
plan after a zero-write conflict; inspect published paths and the disk diff after
partial application before planning further edits.

Install the CLI with `go install github.com/spockz/semantic-editor/cmd/semedit@latest`; from a local checkout, use `go install ./cmd/semedit` at the repository root. Register the resulting `semedit` executable with the harness MCP configuration.

Normal CLI and MCP use inherit the user's Go environment and caches. Only explicitly isolated runs use managed Go state: MCP `--go-base-dir` opts into isolation, and benchmark fixtures configure it themselves. Do not add isolated cache configuration to ordinary harness registrations. Ordinary semantic operations preserve read-only module-file protection; dependency operations can update module files explicitly.

Benchmark-specific Go cache isolation and offline preparation are documented in the [benchmark harness guide](../../tools/benchmark-harness/README.md).

The default MCP `--tool-profile base` includes workspace outline, signature change, function extraction, and doc-comment operations. `--tool-profile full` exposes every registered tool; execution `--profile mutations-only` filters read-only tools independently of that catalog choice.

## Architecture decision records

Use `semantic_search_adrs` to find applicable records and `semantic_generate_adr_index` after ADR edits. The [ADR workflow reference](references/adr-workflow.md) covers metadata, authoring, supersession, and indexing. Use native file tools for Markdown prose and links; the probed Marksman rename did not return cross-link edits. `make lint-mermaid` validates Mermaid syntax without structural editing.

Tool descriptions summarize single-fault support and conditional recovery. The generated capability reference retains the exact mixed-fault matrices; do not infer support for combinations from individual fault classes. Copy continuation fields from the same resolved structured record into follow-up calls, including its revision when required. Missing continuation fields mean no resolved target.

For eligible target errors, inspect the optional structured `suggestions` array before searching broadly. Each candidate contains retry `arguments`, a file and source location, declaration kind, optional signature, and reason. Retry the same semantic operation with the candidate arguments and let native validation decide again. Go recovery includes structural owner selectors, signature changes, extraction owners, and declaration edits with native preflights. Java supports selected-file lookup, inspection, reference search, and rename; Rust supports selected-file lookup, bounded extraction, and bounded literal inlining; Scala supports selected-file lookup and bounded extraction; Kotlin, Haskell, Bash, and Make support selected-file lookup, with Make inspection too. Fresh names being introduced are never corrected. See [bounded symbol suggestions](../../docs/symbol-suggestions.md) for the exact operation list, provider modes, and limits.

## Choose the operation by intent

Use the canonical `refactor_*` names for semantic refactorings. Each refactoring has one registered MCP name and one CLI command.

Lookup, declaration outlines, and references return compact navigation summaries,
without declaration bodies. Paths resolve from the selected workspace root. Line
ranges are one-based and inclusive declaration extents; reference coordinates
identify occurrences separately. Copy an explicit `symbol` selector verbatim,
including receiver or overload qualification. Read disk-backed files directly,
or call `semantic_inspect_symbol` with the returned file and native selector where
the backend supports inspection. An overlay source label identifies unpublished
contents that require overlay-aware inspection. Discovery alone does not grant
mutation capability or replace guarded inspection continuations.

The CLI and MCP text contain the navigation listing. MCP structured results retain
counts, scope, source-view metadata, and required resolved target fields without
duplicating declaration or occurrence arrays. Workspace outlines retain their
separate package/module records; inspection still returns source and revisions.

| Requested change | Use instead of built-in tools | Route away from |
| --- | --- | --- |
| Discover packages, synopses, and module boundaries across a polyglot workspace | `semantic_workspace_outline` with optional `path` and `include_tests`; `language: "auto"` aggregates Go and Java, while `"go"` or `"java"` restricts discovery | Directory listing, `find`, or `tree` shell exploration |
| Find a named symbol in Go, Java, Rust, Scala, Kotlin, Bash, Make, or standalone Haskell | `semantic_lookup` | Built-in text search, `grep`, or line counting |
| Retry an eligible symbol-target error | Review its structured `suggestions` and retry the same operation with one candidate's `arguments`; confirm the candidate's `file`, `kind`, and signature where present | Treating a similarity match or provider result as an accepted target |
| Retrieve a discovered declaration's source, signature, doc comment, and revision | `semantic_inspect_symbol` where supported, or ordinary file reading for disk-backed coordinates | Guessing signatures or reading disk for unpublished overlay results |
| Inspect matching nested Go or Java constructs before selecting an ambiguous target | `semantic_inspect_construct`; copy a candidate path and selection revision from the same result | Guessing structural paths or retrying unchanged selectors |
| View declaration hierarchy in Go, Java, Kotlin, Rust, Scala, Makefile, or Bash | `semantic_outline` with optional `kinds` and `include_unexported` (use `semantic_workspace_outline` first to discover packages) | Reading whole files or manual grep navigation |
| Find usages, call sites, or references of a Go or trusted Java symbol | `semantic_find_references` | Broad text search or regex |
| Rename a symbol and its references in Go, or in selected trusted Rust or Java files | `refactor_rename` | Built-in text replacement across files |
| Replace one supported Go or Java method-local construct | `semantic_replace_construct` with `kind`, optional `discriminator` and `construct_path`, selection revision, and complete replacement `source` | Built-in text edits or regenerating the whole function |
| Remove one supported Go declaration/case or Java method, constructor, field declaration, nested type, or method-local construct | `semantic_remove_construct` with the exact declaration symbol or `in_function` plus `discriminator`; Go `safe_delete` proves unused private standalone constants, eligible free functions, explicitly typed zero-value package variables, initialized basic variables with constant values, method-free defined types, and unreferenced aliases; Java declaration deletion removes attached Javadoc and validates the source | Go safe-delete refuses generic types, direct or promoted methods on defined types, incomplete source sets, and unsupported resource effects; Java safe-delete accepts only private declarations, requires complete JDT LS workspace reference coverage, and refuses if any usages remain |
| Insert one Go control-flow construct or one Java loop, conditional, switch, try-with-resources, synchronized block, statement, or expression inside a method | `semantic_insert_construct` with `kind`, `in_function`, and `source`; Go statements use `anchor` and clauses use `placement`; Java requires an explicit anchor and exact `discriminator` for `before`/`after` | Built-in text edits or line insertion; Java declaration and case insertion remain unsupported |
| Append a keyed Go map entry or Java `Map.of`/`Map.ofEntries`/`List.of`/`Set.of`/array initializer element | `semantic_insert_construct` with `kind: "element"`, `in_declaration`, exact initializer `discriminator`, and `source`; for Java maps pass a key/value pair to `Map.of` or one `Map.entry(...)` to `Map.ofEntries`; copy `expected_revision` when available | `kind: "field"` changes type declarations; positional Go slice/array items and Java collection mutations remain unsupported |
| Change an existing Go or Java function or method body | `semantic_replace_body` with the short or qualified method `symbol` and bare replacement statements | Built-in whole-declaration or text replacement |
| Extract supported callable regions | `refactor_extract_callable` with `kind` and either `start_line`/`end_line` or, for Go, inclusive `construct_path` endpoints plus the current file `expected_revision`. Go derives helper inputs and live-outs from object identity and supports bounded direct-call, assignment, conditional, and loop selections, including direct local or parameter map ranges with nameable scalar key and element types, two-index slice ranges, and full-slice ranges (`items[low:high:max]`, `items[:high:max]`) with pure typed integer bounds. Expression-switch case/default paths accept contiguous direct-call ranges and one direct simple `=` assignment to an eligible pre-existing owner local or named parameter (excluding the receiver) that is read after the whole switch; traversal does not descend into nested case statements. Other assignments, declarations, cross-case ranges, type-switch cases, and select clauses refuse. All path endpoints must share one lexical block. See [Go refactoring boundaries](references/golang/refactoring-boundaries.md). Java delegates direct statement selections to JDT LS; Rust and Scala accept only their documented bounded direct-call actions in the [Java](references/java.md), [Rust](references/rust.md), and [Scala](references/scala.md) guides. | Copying statements into a new helper manually; each backend refuses unsupported types, scope, control flow, and clause paths |
| Extract an exact value as a local or constant | `refactor_extract_value` with `kind: "variable"` or `"constant"`, `symbol`, `in_function`, and exact-source `discriminator`; Go compiler-validates supported calls, assignments (including revision-bound ranges of repeated direct assignment RHS expressions), single-spec local variable declarations, direct or nested returns, complete side-effect-free if conditions without init clauses, and call-range selections. For repeated Go constants, select a revision-bound subset with `occurrence_indices`; omission still requires one unique match. Java delegates both kinds to JDT LS; Rust supports bounded direct-call variable extraction and checked fixed-width decimal constant initializers; Scala supports bounded variable extraction. Field extraction is unsupported. See the [Go](references/golang/refactoring-boundaries.md), [Rust](references/rust/refactoring-boundaries.md), and [Scala](references/scala.md) boundaries. | Rewriting the expression manually |
| Inline one Java local variable | `refactor_inline_value` with `symbol` and `in_function`; Java resolves the local declaration through the CST and delegates the refactoring to JDT LS `refactor.inline` | Replacing local references by text |
| Inline one bounded Go value | `refactor_inline_value` with `symbol` and `in_function`; Go supports basic literals, one keyed scalar-only named struct literal at an immediate direct return, and pure scalar expressions at one immediate direct return, sole argument of an immediate direct call, defer call, or `go` call to a declared function or builtin, or the complete RHS/initializer of an immediately following simple assignment, single-name `:=`, ungrouped local variable declaration with one name and one value, complete condition of an immediately following if statement without an init clause, or complete tag of an immediately following expression switch without an init clause. Defer arguments preserve evaluation at registration; `go` call arguments evaluate before the goroutine starts. See [Go refactoring boundaries](references/golang/refactoring-boundaries.md) for proof conditions and refusals. | Inlining effectful expressions, broader aggregates, or values with other uses; grouped or multi-name/multi-value variable declarations remain unsupported |
| Inline one bounded Rust local value | `refactor_inline_value` with `symbol` and `in_function`; Rust accepts only an untyped immutable local initialized by a suffixed decimal integer literal and used as simple arguments on its next call line | Inlining expressions, mutable bindings, or values used outside that call |
| Promote one bounded Rust local to a constant | `refactor_promote_local_to_const` with `file`, `symbol`, and `in_function`; Rust accepts one immutable local with an explicitly typed, range-valid fixed-width integer decimal literal and exactly one use as the sole argument of its immediately following direct call | General local promotion, custom or `Drop` types, extra uses, generated-name collisions, and inferred or pointer-sized types |
| Inline one Java method call | `refactor_inline_callable` with the callee `file`/`symbol` and a closed `call_site` selecting the caller file, method, and exact call expression; Java delegates that call to JDT LS `refactor.inline` | Expanding a helper body manually at its caller |
| Inline one or several bounded Go callable sites | `refactor_inline_callable` with callee `file`/`symbol` and a closed `call_site`; same-package concrete methods may be declared in a different active file. Go free functions additionally accept revision-bound `call_site.occurrence_indices` for selected direct-owner returns or one-target assignment RHSs. See [Go refactoring boundaries](references/golang/refactoring-boundaries.md) for exact proof conditions | Inlining unsupported calls manually instead of checking the backend boundary |
| Inline one bounded Rust callable | `refactor_inline_callable` with same-file callee `file`/`symbol` and a `call_site` naming a caller and exact `callee()` expression; Rust accepts only matching explicit primitive integer return types and a callee body containing one suffixed decimal literal | Inlining methods, parameterized calls, unsuffixed literals, or multi-caller assist results |
| Apply an advanced Java interface, signature, parameter, or type-move refactoring | Use the matching Java operation described in the [Java language guide](references/java.md) or [advanced Java recipes](references/java/advanced-refactorings.md) | Editing signatures, callers, interfaces, or moved declarations independently |
| Change a bounded private Go free-function signature | `refactor_change_signature` with complete `parameters`: append one closed argument, permute only when every direct-call argument is constant, or remove one unused scalar parameter when only its discarded call arguments are constant | Parameter renames, result changes, mixed append/permutation/removal, and nonconstant reordered or discarded arguments |
| Make one Java method static and update its callers | `refactor_make_static` with trusted workspace, exact signature-qualified `symbol`, and a selected `file`; the method name must be unique in that file. See the [advanced Java recipes](references/java/advanced-refactorings.md) for the native action and receipt boundary. | Adding `static` and editing call sites independently |
| Make one Java local's inferred type explicit | `refactor_make_type_explicit` with the local `symbol`, exact signature-qualified `in_function`, selected `file`, and fresh `expected_revision` from method inspection; only one direct native var token replacement is accepted. See the [Java guide](references/java.md#make-an-inferred-local-type-explicit). | Arbitrary type changes, modified or ambiguous locals, fields, parameters, loop/lambda scopes, anonymous types, imports, commands, or extra edits |
| Move a closed Go production-file declaration set and its matching test file to another existing package | `refactor_move_file` with `file`, `destination_file`, and optional `move_matching_tests`; plans the complete set and typed references together | Sequential symbol moves, automatic export of private helpers, or general resource/package migration |
| Read diagnostic evidence from a completed MCP operation | `semantic_diagnostics` with `diagnostic_report.arguments` and optional `category`, `contains`, `offset`, and `limit`; copy the operation's captured query metadata | Rerunning verification to read an existing result; unknown or expired reports require newer captured evidence |
| Add a Java switch case | `semantic_insert_case` with the owning method, switch `discriminator`, complete case clause, placement, and inspected `construct_path` plus `expected_selection_revision` when ambiguous | Editing switch bodies by text or selecting a switch by line number |
| Update an existing Go declaration or a Java method, field, or nested type | `semantic_replace_decl`; Java replacements must preserve declaration kind and name | Built-in text replacement or duplicate insertion |
| Add one Go function or method to an existing file | `semantic_insert_function` | Built-in text insertion or `replace_file_content` |
| Add one Go struct, interface, or type alias | `semantic_insert_type` | Built-in text insertion or `replace_file_content` |
| Add one nested Java class, interface, enum, or record | `semantic_insert_type` with a complete nested declaration, optional `in_type`, and member placement | Java cannot infer the owner when the file declares multiple top-level types |
| Add one field to a private non-generic Go struct or a Java class, enum, or record | `semantic_insert_field` with `in_type` and a qualified `symbol`; Java `type_end` respects static/instance and visibility ordering | Go exported owners and duplicate fields are refused; Java insertion does not generate constructor updates |
| Apply an advanced Go signature, interface, type, field, or declaration refactoring | Read the [advanced Go refactoring recipes](references/golang/advanced-refactorings.md) for operation selection, selectors, boundaries, import, and batch behavior; cross-package declaration movement supports exported functions, standalone constants, aliases with predeclared type expressions, variables, and a bounded defined type with its co-located exported direct methods | Generating accessors or moving fields and methods independently |
| Set or remove a Java declaration's Javadoc | `semantic_set_doc_comment` with `language: "java"`, declaration `symbol`, and plain comment text; pass an empty string to remove the attached Javadoc | Editing comment text without checking declaration ownership |
| Add a Java method or constructor to a class, enum, record, or interface | `semantic_insert_function` with a complete declaration in `source`, optional `in_type`, and member placement | Java cannot infer the owner when the file declares multiple top-level types |
| Add one Go constant or variable | `semantic_insert_decl` | Built-in text insertion or `replace_file_content` |
| Add a top-level Go declaration when generic placement controls fit the task | `semantic_insert_declaration`; prefer the function, type, or constant/variable operation for those common kinds | Built-in text insertion by line number |
| Create a Go or Java source file and infer its package | `semantic_scaffold_file`, followed by a matching insertion operation; Java inference uses sibling `.java` files or `src/main/java` / `src/test/java` | Built-in file writing for the package header |
| Add, remove, or format Go imports, or update imports in one trusted Java file | `semantic_organize_imports`; Go supports file/workspace scopes, Java requires `language: "java"`, trust, and one `.java` file | Built-in import-block edits or shell `goimports` |
| Add a Go module or Java Maven dependency | `semantic_add_build_dependency`; Go updates `go.mod`/`go.sum`; Java updates a trusted reactor using `dependencyManagement` | Direct manifest edits or shell package-manager commands |
| Add a Go switch case | `semantic_insert_case` | Built-in switch text replacement |
| Convert Go test assertions (t.Fatal/t.Error) between failure modes | `semantic_assertion_mode`; writes require `trust_workspace: true`, while `dry_run: true` previews without trust or writes | Built-in text replacement of `t.Fatal`/`t.Error` calls |
| Apply several registered semantic edits in order | `semantic_batch` when exposed by the server's startup configuration; successful earlier edits remain written if a later edit fails. Rust and Scala extraction can be followed by `semantic_lookup` in the same batch to inspect the working overlay. | Several built-in text patches |
| Keep a multi-turn edit unpublished until review | `semantic_stage` to start and apply edits; `semantic_commit` to publish, `semantic_stage_drop` to remove one staged step, or `semantic_stage_abort` to discard the staged journal | `semantic_batch`, whose successful child edits publish as one-shot work |
| Explicitly format or check Go or trusted Java sources | `semantic_verify`; Go verification runs formatting before diagnostics and can write files, and Java accepts a file or directory scope for JDT LS formatting/import organization. There is no dry-run | Shell formatters or ad hoc diagnostics; avoid a redundant call after an edit that already verifies |
| Compile tests for a trusted Java root POM | `semantic_maven_compile`; set `trust_workspace: true`, runs offline unless `allow_network: true`, and writes build outputs plus temporary Maven data under `.scratch` | Shell Maven invocation |
| Run tests for a trusted Java root POM | `semantic_maven_test`; set `trust_workspace: true`, runs offline unless `allow_network: true`, and writes build outputs plus temporary Maven data under `.scratch` | Shell Maven invocation |
| Save a pre-edit rollback point for a bounded edit | `semantic_snapshot` | `git stash` or manual backup copies |
| Restore edits from a semedit snapshot | `semantic_undo` | `git reset` or manual file restoration |
| Reload a promoted MCP binary when live reload is enabled | `semantic_reload` | Restarting the client or server manually |

`semantic_batch` is disabled by default. Use it only when the server was started
with `--enable-batching` and it appears in the current tool inventory; direct
operations and retained staging remain available when it is absent.
| Prepare a privacy-reviewed semedit tool-friction draft | `report_feedback` | Manually drafting a tool issue; keep local friction logs required by this repository |

Advanced Go operations include `refactor_change_signature`, `refactor_introduce_parameter`, `refactor_introduce_parameter_object`, `refactor_extract_interface`, `refactor_implement_contract`, `refactor_encapsulate_field`, `refactor_extract_type`, `refactor_extract_delegate`, `refactor_migrate_type`, and `refactor_move_declaration`; use the [advanced Go recipes](references/golang/advanced-refactorings.md) for their exact inputs and boundaries. Go field-only type extraction also groups one contiguous source-order run of selected keys in an explicit same-file owner literal, preserving value bytes and omitted zero fields; interleaved keys, local type-name capture, literal comments, overlapping edits, and cross-file uses refuse. Delegate extraction retains its keyed-use refusal. Advanced Java operations include `refactor_change_signature`, `refactor_introduce_parameter`, `refactor_implement_contract`, `refactor_extract_interface`, `refactor_introduce_parameter_object`, and `refactor_move_declaration`; use the [Java guide](references/java.md) and [advanced Java recipes](references/java/advanced-refactorings.md) for their exact inputs and boundaries.

For Go signature changes, use `parameters` with `source_index` to append one parameter, preserve an unchanged signature, completely permute existing parameters, or remove exactly one unused scalar parameter. Permutations require compiler-proven constant arguments at every direct call; removal requires a constant only for the argument discarded with the removed formal. Surviving arguments keep their exact source and order, including effectful calls. Eligible folded/named constants and `len`/`unsafe.Sizeof` expressions are supported as proof for reordered or discarded arguments. Tuple/ellipsis arguments, mixed additions/removals/permutations, parameter renames, result changes, affected-region comments, directives, and incomplete caller proofs refuse. Targets remain private nongeneric nonvariadic free functions with bodies. Copy one finalized `function_target` record for later standalone edits; buffered results provide no live target.

Examples:

- “Rename `Server.Start` to `Server.Run`” → use `refactor_rename` with a receiver-qualified symbol.
- “Rewrite the body of `(*Client).Do` while keeping its signature” → use `semantic_replace_body` in Go; use the same operation with a Java method selector for Java.
- “Change only the assertion loop in `TestMessages`” → use `semantic_replace_construct` with `kind: "loop"`; select a reported `construct_path` if more than one loop matches.
- “Add a retry loop after the validation conditional in `LedgerService.post`” → inspect the Java method, then use `semantic_insert_construct` with `kind: "loop"`, `anchor: "after"`, and the exact conditional `discriminator`.
- “Change `DefaultLimit` from 5 to 10” → use `semantic_replace_decl`, not a new `semantic_insert_decl` call.
- “Change the initializer of `LedgerService.limit`” → use `semantic_replace_decl` with the complete Java field declaration, keeping its name and declaration kind.
- “Add the `net/http` import” → use `semantic_organize_imports`; “add the module that provides this package” → use `semantic_add_build_dependency`.
- “Where is `Config.Port` declared?” → use `semantic_lookup`, with the file path if needed to disambiguate.
- “The Go symbol selector failed and returned candidates” → review the candidate evidence, then retry the original operation with one suggestion's arguments; unsupported cases keep the original error.

For methods, qualify the receiver when useful, such as `Server.Start` or `(*Server).Do`. For Go lookup, omit the optional file when the symbol owner is unknown; supply it to disambiguate repeated local names. Go insertion calls accept one declaration. Omitted access follows identifier casing; explicit access must agree with that casing. Relative file and path parameters resolve from the active semedit MCP workspace root, not the task's current directory. If MCP is rooted at the repository and the target is an isolated worktree, include `.scratch/worktrees/<name>/...` in the path; for Rust or Java rename, the selected file must belong to the trusted active workspace. Maven `root` must be absolute and remain inside the trusted workspace; use an absolute worktree path to target an isolated worktree.

### Copy typed results and respect publication

Treat `structuredContent` as the canonical full result, and copy continuation fields only from the same resolved record. Review the [results, diagnostics, and publication reference](references/results-publication.md) for edit receipts, buffered results, diagnostic pages, metrics, and retained staging controls such as `semantic_stage`, `semantic_stage_drop`, `semantic_stage_abort`, and `semantic_commit`.

For timing details, use `semantic_metrics` and consult its on-demand [reference](references/metrics.md).

### Direct targeting over preliminary lookup

When the target file and symbol name are already known from the task description or an error trace, call `refactor_rename`, `semantic_replace_body`, or `semantic_replace_construct` directly with `file` and a short `symbol`, such as `file: "server.go", symbol: "Start"` or `symbol: "Server.Start"`. Use `semantic_lookup` first only when the file location or symbol definition is genuinely unknown.

## Respect backend capabilities

Operation scope, trust requirements, verification, and refusal boundaries vary by language. Use the operation table for intent routing, then consult the relevant language guide for setup and supported behavior; do not infer support from another backend. Rare, multi-step refactorings have separate on-demand references: [Go](references/golang/advanced-refactorings.md), [Java](references/java/advanced-refactorings.md), and [Rust](references/rust/refactoring-boundaries.md). Scala's current extraction boundaries are documented in [its guide](references/scala.md). The language guides also identify read-only backends and required verification tools.

Framework conventions belong in a relevant installed framework skill. Use semedit's language-level operation to perform the structural change, and use that skill to choose the framework-specific owner or placement.

## Harness administration output

`install`, `integration status`, and `uninstall` print human-readable results by default. Pass `--json` when scripting these administrative commands; JSON is written to stdout without extra guidance. This flag does not change semantic-operation or MCP tool output.

## Core Operations vs. Macro Refactorings

`semedit` strictly demarcates low-level engine responsibilities from high-level architectural refactorings:

1. **Core Engine Primitives**: Zero-token, deterministic AST mutations and compiler/LSP queries (`semantic_lookup`, `semantic_outline`, `refactor_rename`, `semantic_replace_body`, `semantic_insert_construct`, `semantic_replace_construct`, `semantic_batch`). The engine executes exact syntactic transformations and enforces local compiler invariants; it does not implement framework-specific heuristics or whole-project modernization lifecycles.
2. **Skill-Orchestrated Macro Refactorings**: High-level, cross-paradigm, or idiomatic refactorings (such as decomposing classes, introducing design patterns, or modernizing language constructs) are composed by the agent using core operations in `semantic_batch`. Whole-project migrations that alter external serialization or ecosystem contracts belong in dedicated linting/upgrade tools rather than core engine commands.

### General Refactoring Examples

- For a private Go field, use the bounded encapsulation operation when direct reads and writes fit its contract; see the advanced Go recipes for its limits.
- For abstraction, delegation, or function decomposition, combine the matching extraction operation with caller updates and verify the resulting overlay.

## Language-Specific Refactoring References

To conserve context across agent harnesses, read only the reference for a language you are actively using with semedit. That includes navigation, inspection, setup, verification, and edits; skip references for unrelated languages.

- **Go**: Consult [references/golang.md](references/golang.md) when using semedit with Go for navigation, inspection, setup, verification, or edits.
- **Java**: Consult [references/java.md](references/java.md) when using semedit with Java for navigation, inspection, setup, verification, or edits.
- **Rust**: Consult [references/rust.md](references/rust.md) when using semedit with Rust for navigation, inspection, setup, verification, or edits.
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
