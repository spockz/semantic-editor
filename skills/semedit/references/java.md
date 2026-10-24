# Java Semantic Refactoring Guide
<!-- WHY: Route Java semedit tasks to supported operations and their verified boundaries without loading uncommon migration recipes. -->

## Make an inferred local type explicit

Inspect the signature-qualified owner method to obtain its current file revision, then call `refactor_make_type_explicit` with `file`, the unique local binding `symbol`, `in_function` (for example `Service.count()`), and that `expected_revision`. The operation requests JDT LS's exact `Change type of '<local>' to '<type>'` action of kind `refactor` and accepts only one eager selected-file edit replacing the CST `var` token. Initializer and use bytes remain unchanged before normal formatting and verification. The receipt continues at the owner method because locals are not document-symbol targets.

This supports primitive types and denotable named or generic types requiring no import edits. Modified or annotated declarations, duplicate locals, fields, parameters, loop scopes (`for`, enhanced `for`, `while`, and `do` bodies), lambda/nested-class scopes, anonymous or non-denotable types, stale revisions, commands, lazy actions, resource edits, annotations, and extra edits refuse without applying the native edit. Explicit replacements containing a reserved `var` component also refuse, including `var[]` and `List<var>`. It does not change an already explicit type.

MCP results follow the [shared output policy](../SKILL.md#copy-typed-results-and-respect-publication).
Use typed continuations with their exact revisions, and retain failure and
publication state when choosing follow-up calls. Structured results can be
model-visible; text keeps exact next-step guards and names structured fields
without repeating source, inventories, diagnostic pages, diffs, or logs.
For diagnostic details, copy the response's `diagnostic_report.arguments` into
`semantic_diagnostics`, choose a category/filter and page size, then follow
`next_offset`. Pages reuse the captured report; a deferred/unavailable count
does not indicate a clean workspace. Use verification only for a fresh report.
Rename returns `active_count` and diagnostic changes rather than every existing
workspace warning; request `semantic_verify` for a full diagnostic inventory.

This reference explains how to use `semedit` with Java, including bounded transformations and Maven integration. Read it when navigating, inspecting, setting up, verifying, or editing Java source files.

For less frequent refactoring recipes, see [advanced Java refactorings](java/advanced-refactorings.md): interface/contract changes, parameter and signature changes, parameter objects, declaration movement, callable inlining, Make Static, and class-shape changes.

Trusted Java requests reuse a JDT LS session per workspace root and keep selected document versions synchronized with the source overlay. A trusted root with a regular `pom.xml` enables JDT LS Maven import by default; pass `import_maven: false` to disable it. JDT LS import does not execute Maven.

The default MCP base catalog includes Java-capable `semantic_workspace_outline`, `refactor_change_signature`, `refactor_extract_callable`, and `semantic_set_doc_comment` operations. `--tool-profile full` exposes all registered tools, while execution `--profile mutations-only` independently removes read-only tools such as workspace outline.

Selected-file lookup, inspection, reference search, and rename failures may expose
up to three structured `suggestions`. They come from the current native JDT LS
outline or an optional provider and must pass native target validation. Retry
the original operation with a candidate's `arguments`; preserve its overload
signature and file scope. Earlier retained renames can contribute former-name
evidence, but only the current selector is returned. Suggestions do not change
source, choose a target, or retry a mutation. `SEMEDIT_SUGGESTIONS=generic` is the
default; `codanna,generic` adds the optional installed search provider and `off`
disables recovery. Normal Java workspace trust and server configuration remain
required. See [configuration and limits](../../../docs/symbol-suggestions.md).

Use `semantic_scaffold_file` with `language: "java"` to create a source file. `package: "infer"` uses a consistent package declaration from sibling `.java` files, or derives it from a path below `src/main/java` or `src/test/java` when no sibling declares one. Conflicting sibling packages fail closed. A scaffolded file contains only its package header; add declarations with the corresponding semantic operations.

Use `semantic_workspace_outline` with `language: "java"` to list only Maven reactor modules and Java packages. Auto mode aggregates Java and Go in mixed repositories; a Java build manifest does not suppress Go discovery. Isolated Java documentation samples do not activate repository-wide Java discovery; select their directory explicitly. Package synopses prefer `package-info.java` Javadocs, then the primary public type's Javadoc. Module metadata carries Maven descriptions and scoped package counts; package records identify their owning module and language. `include_tests: true` includes test-source packages. MCP workspace outline text summarizes package and module records, which appear once in `structuredContent`. Declaration outlines use compact navigation listings. Use `semantic_outline` with a Java file or directory path to inspect declarations; directory scope supports `include_unexported: false` and `include_tests: true`. `semantic_lookup` and `semantic_inspect_symbol` can also search a selected Java directory.

For overloaded methods and constructors, query by base name to receive all candidates, or provide a simple erased signature such as `OrderService.process(Order, boolean)` to choose one overload. Signature matching ignores spacing after commas. Compact navigation keeps this native selector separate from the descriptive signature; copy it verbatim with its workspace-relative file to `semantic_inspect_symbol`. Declaration line ranges are one-based and inclusive, while reference positions identify occurrences. Disk-backed results also support ordinary file reading.

Use `semantic_find_references` with a selected Java file and a uniquely qualified symbol to ask JDT LS for workspace references. Returned locations are checked to remain inside the trusted root and carry source revisions. Overloaded methods still require a selector that resolves to exactly one declaration.

Use `refactor_extract_value` with `kind: "variable"` to extract an exact expression into a local, or `kind: "constant"` for a compile-time constant. Field extraction is unsupported because stock JDT LS generates a mutable instance field and a method-local assignment, and semedit has no verified contract for that edit shape. Supply `in_function`, exact-source `discriminator`, and the caller-chosen `symbol`; if the expression is ambiguous, copy `construct_path` and `expected_revision` from the current inspection result. JDT LS performs the selected extraction action, semedit validates and renames the generated declaration, then formats and verifies the file.

Use `refactor_extract_callable` to extract a contiguous range of direct statements from one Java method into a helper method. Supply the owner `in_function`, helper `symbol`, `kind: "method"`, and 1-based `start_line` / `end_line` matching complete statement boundaries. Nested or partial control-flow selections fail closed; JDT LS infers parameters and return values, then semedit renames its generated helper, formats, and verifies the file.

Use `refactor_inline_value` to inline one local declaration in a Java method. Supply the declaration `symbol` and `in_function`; semedit resolves the declaration with the Java CST and requests JDT LS `refactor.inline`. The action’s edits are validated before formatting and diagnostics. Use this for a binding JDT LS can prove safe to inline; do not hand-replace its references.

Use `semantic_replace_body` with a Java method selector and bare replacement statements to change a method body while preserving its signature. Use `semantic_replace_decl` to replace one selected method, field, or nested type declaration; the replacement must preserve the selected declaration kind and name. Use `semantic_remove_construct` to remove a method-local construct by `in_function` and exact `discriminator`, or a method, constructor, field declaration, or nested type by its unique symbol. Declaration deletion also removes a directly attached Javadoc comment. Java `safe_delete` is limited to private declarations and requires complete JDT LS workspace reference coverage with no usages; it refuses on any remaining references or incomplete analysis. The embedded CST validates the result, then JDT LS formats the file, organizes imports, and verifies diagnostics.

Use `semantic_insert_field` with `language: "java"`, workspace trust, a selected `in_type`, qualified field `symbol`, and one complete field declaration in `source`. Omit `in_type` only when the file has exactly one top-level type and the new symbol uses that type name. `type_end` inserts within the field section by static/instance category and visibility; `type_start` and explicit `before_symbol` / `after_symbol` anchors override that ordering. Duplicate fields, ambiguous type owners, invalid declarations, and stale `expected_revision` values are refused. Constructor updates are not synthesized.

Use `semantic_set_doc_comment` with `language: "java"`, workspace trust, `scope: "declaration"`, and a unique declaration `symbol` to set or remove its attached Javadoc. Supply plain text without comment delimiters; use an explicit empty string to remove the current Javadoc. The operation preserves declaration annotations, rejects ambiguous or stale selections, and formats and verifies through JDT LS.

Use `semantic_insert_function` with `language: "java"`, workspace trust, and one complete method or constructor declaration in `source`. Select the owner with `in_type`; it may be omitted only when the selected file has exactly one top-level type. `type_end` places the declaration in its member tier, ordered by visibility, before nested types. Explicit `type_start`, `before_symbol`, and `after_symbol` placements are available. Formatting and diagnostics run through JDT LS after the CST validates the member.

Use `semantic_insert_type` with `language: "java"`, workspace trust, and one complete nested class, interface, enum, or record declaration. Select its containing type with `in_type`; it may be omitted only when the file has exactly one top-level type. `type_end` orders nested types by visibility, and explicit member anchors are supported. The selected owner remains the top-level type for source editing; the inserted nested type name is returned as a qualified continuation.

Use `semantic_inspect_construct` with a trusted Java file and the exact method selector to list method-local loop, conditional, switch, case, try, synchronized, array-element, `Map.of`/`Map.ofEntries`/`List.of`/`Set.of`, and statement/expression candidates. Copy `construct_path` and `expected_selection_revision` from the result when selecting an ambiguous target. `semantic_insert_construct` inserts one method-local `loop`, `if`, `switch`, `try_with_resources`, `synchronized`, or `statement/expression` snippet using `start`, `end`, `before_return`, `before`, or `after`. Relative anchors require an exact `discriminator`; repeated matches require `construct_path` plus the inspected file revision. To append an initializer element, use `kind: "element"`, `in_declaration` for the field, the exact current initializer as `discriminator`, and the new argument source. Add one expression to `List.of`/`Set.of`/arrays, a key/value pair to `Map.of`, or one `Map.entry(...)` to `Map.ofEntries`; JDT LS verifies resulting types. `semantic_replace_construct` stages one selected replacement in the source buffer, rejects stale candidate revisions and invalid Java syntax, then delegates formatting, import organization, and diagnostics to JDT LS. Java declaration insertion and non-method declaration removal remain unavailable.

Use `semantic_insert_case` to add a complete Java `case` or `default` clause to one selected switch in a method. Select its owner with `in_function`, identify a tagged switch with `discriminator`, and choose `first`, `last`, `before_default`, `before`, or `after`. Before/after require `target_case`; an ambiguous switch requires the inspected `construct_path` and matching `expected_selection_revision`. Duplicate labels and stale selections fail closed before JDT LS formatting and diagnostics.

`refactor_rename` applies JDT LS text edits across Java files in the trusted workspace after validating each file preimage. A standalone public top-level type rename can also move its `.java` file when JDT LS returns one `RenameFile` operation. The destination must be absent and in the same package directory; buffered batches refuse resource renames.

Use `semantic_organize_imports` with one selected `.java` file and `trust_workspace: true` to add fully qualified type imports, remove exact imports, then format and organize the file through JDT LS. The operation edits the shared source buffer and commits through the verified Java backend. `semantic_verify` also accepts a Java directory path; formatting and import organization run on every Java source beneath that scope, including test sources.

Use `semantic_add_build_dependency` with `language: "java"`, `trust_workspace: true`, and a Maven coordinate such as `org.example:codec:2.1`. Set `file` to a module directory or `pom.xml` in a reactor; omit it to target the root POM. Existing ancestor `<dependencyManagement>` entries produce a versionless module dependency. Otherwise a versioned dependency is added to the nearest ancestor management section and the selected module; standalone POMs receive a versioned direct dependency. The operation edits POM XML without running Maven and notifies an already-running JDT LS session to refresh project configuration.

Use `semantic_stage` before a multi-turn Java change that must stay unpublished. Java mutations remain in the shared source overlay and return `stage_receipt.step_id`; copy that selector to `semantic_stage_drop` to remove one staged edit, use `semantic_stage_abort` to discard the session, or use `semantic_commit` to publish and run Java diagnostics. Staging requires a retained MCP session. The standalone `stage`, `stage-drop`, `stage-abort`, and `commit` CLI commands fail with a session-required error and do not publish changes.

---

## Java workflow patterns

The primary `SKILL.md` table indexes semedit's core Java operations. The advanced reference contains less frequent signature, parameter, interface and contract changes, parameter objects, declaration moves, callable inlining, Make Static, and class-shape migrations.
