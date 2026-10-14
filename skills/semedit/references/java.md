# Java Semantic Refactoring Guide

This reference explains how to execute bounded and multi-step Java code transformations using `semedit` primitives and Maven integration. Read this document only when refactoring or modernizing Java source files.

Trusted Java requests reuse a JDT LS session per workspace root and keep selected document versions synchronized with the source overlay. A trusted root with a regular `pom.xml` enables JDT LS Maven import by default; pass `import_maven: false` to disable it. JDT LS import does not execute Maven.

The default MCP base catalog includes Java-capable `semantic_workspace_outline`, `semantic_change_signature`, `semantic_extract_function`, and `semantic_set_doc_comment` operations. `--tool-profile full` exposes all registered tools, while execution `--profile mutations-only` independently removes read-only tools such as workspace outline.

Use `semantic_scaffold_file` with `language: "java"` to create a source file. `package: "infer"` uses a consistent package declaration from sibling `.java` files, or derives it from a path below `src/main/java` or `src/test/java` when no sibling declares one. Conflicting sibling packages fail closed. A scaffolded file contains only its package header; add declarations with the corresponding semantic operations.

Use `semantic_workspace_outline` with `language: "java"` to list only Maven reactor modules and Java packages. Auto mode aggregates Java and Go in mixed repositories; a Java build manifest does not suppress Go discovery. Isolated Java documentation samples do not activate repository-wide Java discovery; select their directory explicitly. Package synopses prefer `package-info.java` Javadocs, then the primary public type's Javadoc. Module metadata carries Maven descriptions and scoped package counts; package records identify their owning module and language. `include_tests: true` includes test-source packages. MCP outline text summarizes the result; complete records appear once in `structuredContent`. Use `semantic_outline` with a Java file or directory path to inspect declarations; directory scope supports `include_unexported: false` and `include_tests: true`. `semantic_lookup` and `semantic_inspect_symbol` can also search a selected Java directory.

For overloaded methods and constructors, query by base name to receive all candidates, or provide a simple erased signature such as `OrderService.process(Order, boolean)` to choose one overload. Signature matching ignores spacing after commas.

Use `semantic_find_references` with a selected Java file and a uniquely qualified symbol to ask JDT LS for workspace references. Returned locations are checked to remain inside the trusted root and carry source revisions. Overloaded methods still require a selector that resolves to exactly one declaration.

Use `semantic_extract_variable` to extract an exact expression from one Java method. Supply `in_function`, exact-source `discriminator`, and the caller-chosen local `symbol`. If the expression is ambiguous, copy `construct_path` and `expected_revision` from the current inspection result. The operation requests JDT LS `refactor.extract.variable`, captures direct or intercepted workspace edits, renames the generated local, then formats and verifies the file. Unsupported or ambiguous JDT LS actions fail before publication.

Use `semantic_extract_constant` for a Java compile-time constant expression. Supply `in_function`, exact-source `discriminator`, and the caller-chosen constant `symbol`; use `construct_path` with its matching `expected_revision` when the expression repeats. JDT LS `refactor.extract.constant` creates the field, after which semedit renames the generated identifier, formats, and verifies the file. Keep this separate from local extraction: use `semantic_extract_variable` for runtime expressions.

Use `semantic_extract_function` to extract a contiguous range of direct statements from one Java method into a helper method. Supply the owner `in_function`, helper `symbol`, `kind: "method"`, and 1-based `start_line` / `end_line` matching complete statement boundaries. Nested or partial control-flow selections fail closed; JDT LS infers parameters and return values, then semedit renames its generated helper, formats, and verifies the file.

Use `semantic_implement_contract` with the concrete class `file` and `symbol`, the interface name in `contract`, and its source in optional `contract_file`. The operation adds the interface to the class when absent, then uses JDT LS `java/listOverridableMethods` and `java/addOverridableMethods` to generate only missing method stubs. Java currently writes to the selected class file, so `destination_file` must be omitted or equal `file`; the generated methods are placeholders and need a follow-up `semantic_replace_body` before they represent completed behavior.

Use `semantic_extract_interface` with the concrete type in `in_type`, a new interface `symbol`, and exact direct method selectors in `members`. Java extracts public instance methods only, appends a package-private interface to the same source file, adds it to the class, then formats and verifies through JDT LS. `destination_file` must be omitted or equal `file`; overload selectors must identify one method signature.

Use `semantic_change_signature` with one signature-qualified Java method `symbol` and the complete ordered `parameters` mapping. Preserve each existing parameter with its `source_index`; added parameters require a closed `argument` expression so JDT LS can update every resolved call site. The returned workspace edit is validated and applied across the affected Java files, then each file is formatted and checked for diagnostics. Do not split the signature change and caller updates into separate edits.

Use `semantic_introduce_parameter` when one exact expression in a Java method should become a new parameter. Supply the complete ordered `parameters` mapping, preserve existing entries by `source_index`, add the new parameter with its call-site `argument`, and identify the expression with `discriminator` plus `construct_path` when needed. JDT LS updates the method signature and resolved callers; semedit then replaces the selected expression in the method body and verifies every changed file.

Use `semantic_introduce_parameter_object` when a Java method's full parameter list should become one cohesive value object. Supply a `parameter` with the new argument name and type, map every original argument once through `fields` in `source_index` order, and select an existing same-file `destination_file`. The operation generates a package-private immutable class with a constructor and accessors, asks JDT LS to replace the method parameters and update references, then rewrites each constructor call with its original arguments in order and replaces method-body parameter uses with accessors. It refuses public or protected methods, cross-package callers, annotated or otherwise unsupported parameter syntax, argument comments, field type changes, non-invocation references, incomplete call-site edits, and stale source revisions. JDT LS formats and checks every changed file before publication.

Use `semantic_move_construct` to move a Java static method, static field, or static nested type to an existing destination type. Supply the source `file` and `symbol`, the destination type's `destination_file`, and its qualified name as `target_symbol`; omit Go placement options. JDT LS performs the move, updates references and imports across affected files, and semedit verifies the resulting workspace edit. Instance methods and non-static nested types are refused.

Use `semantic_inline_function` to inline one Java invocation. Select the callee with `file` and `symbol`; use `call_site` to select the caller file, its `in_function`, and the exact call expression as `discriminator`. The expression must resolve uniquely in the method, and JDT LS `refactor.inline` edits only that selected caller file.

Use `semantic_inline_variable` to inline one local declaration in a Java method. Supply the declaration `symbol` and `in_function`; semedit resolves the declaration with the Java CST and requests JDT LS `refactor.inline`. The action’s edits are validated before formatting and diagnostics. Use this for a binding JDT LS can prove safe to inline; do not hand-replace its references.

Use `semantic_replace_body` with a Java method selector and bare replacement statements to change a method body while preserving its signature. Use `semantic_replace_decl` to replace one selected method, field, or nested type declaration; the replacement must preserve the selected declaration kind and name. Use `semantic_remove_construct` to remove a method-local construct by `in_function` and exact `discriminator`, or a method, constructor, field declaration, or nested type by its unique symbol. Declaration deletion also removes a directly attached Javadoc comment. Java `safe_delete` is limited to private declarations and requires complete JDT LS workspace reference coverage with no usages; it refuses on any remaining references or incomplete analysis. The embedded CST validates the result, then JDT LS formats the file, organizes imports, and verifies diagnostics.

Use `semantic_insert_field` with `language: "java"`, workspace trust, a selected `in_type`, qualified field `symbol`, and one complete field declaration in `source`. Omit `in_type` only when the file has exactly one top-level type and the new symbol uses that type name. `type_end` inserts within the field section by static/instance category and visibility; `type_start` and explicit `before_symbol` / `after_symbol` anchors override that ordering. Duplicate fields, ambiguous type owners, invalid declarations, and stale `expected_revision` values are refused. Constructor updates are not synthesized.

Use `semantic_set_doc_comment` with `language: "java"`, workspace trust, `scope: "declaration"`, and a unique declaration `symbol` to set or remove its attached Javadoc. Supply plain text without comment delimiters; use an explicit empty string to remove the current Javadoc. The operation preserves declaration annotations, rejects ambiguous or stale selections, and formats and verifies through JDT LS.

Use `semantic_insert_function` with `language: "java"`, workspace trust, and one complete method or constructor declaration in `source`. Select the owner with `in_type`; it may be omitted only when the selected file has exactly one top-level type. `type_end` places the declaration in its member tier, ordered by visibility, before nested types. Explicit `type_start`, `before_symbol`, and `after_symbol` placements are available. Formatting and diagnostics run through JDT LS after the CST validates the member.

Use `semantic_insert_type` with `language: "java"`, workspace trust, and one complete nested class, interface, enum, or record declaration. Select its containing type with `in_type`; it may be omitted only when the file has exactly one top-level type. `type_end` orders nested types by visibility, and explicit member anchors are supported. The selected owner remains the top-level type for source editing; the inserted nested type name is returned as a qualified continuation.

Use `semantic_inspect_construct` with a trusted Java file and the exact method selector to list method-local loop, conditional, switch, case, try, synchronized, array-element, `Map.of`/`Map.ofEntries`/`List.of`/`Set.of`, and statement/expression candidates. Copy `construct_path` and `expected_selection_revision` from the result when selecting an ambiguous target. `semantic_insert_construct` inserts one method-local `loop`, `if`, `switch`, `try_with_resources`, `synchronized`, or `statement/expression` snippet using `start`, `end`, `before_return`, `before`, or `after`. Relative anchors require an exact `discriminator`; repeated matches require `construct_path` plus the inspected file revision. To append an initializer element, use `kind: "element"`, `in_declaration` for the field, the exact current initializer as `discriminator`, and the new argument source. Add one expression to `List.of`/`Set.of`/arrays, a key/value pair to `Map.of`, or one `Map.entry(...)` to `Map.ofEntries`; JDT LS verifies resulting types. `semantic_replace_construct` stages one selected replacement in the source buffer, rejects stale candidate revisions and invalid Java syntax, then delegates formatting, import organization, and diagnostics to JDT LS. Java declaration insertion and non-method declaration removal remain unavailable.

Use `semantic_insert_case` to add a complete Java `case` or `default` clause to one selected switch in a method. Select its owner with `in_function`, identify a tagged switch with `discriminator`, and choose `first`, `last`, `before_default`, `before`, or `after`. Before/after require `target_case`; an ambiguous switch requires the inspected `construct_path` and matching `expected_selection_revision`. Duplicate labels and stale selections fail closed before JDT LS formatting and diagnostics.

`semantic_rename` applies JDT LS text edits across Java files in the trusted workspace after validating each file preimage. A standalone public top-level type rename can also move its `.java` file when JDT LS returns one `RenameFile` operation. The destination must be absent and in the same package directory; buffered batches refuse resource renames.

Use `semantic_organize_imports` with one selected `.java` file and `trust_workspace: true` to add fully qualified type imports, remove exact imports, then format and organize the file through JDT LS. The operation edits the shared source buffer and commits through the verified Java backend. `semantic_verify` also accepts a Java directory path; formatting and import organization run on every Java source beneath that scope, including test sources.

Use `semantic_add_build_dependency` with `language: "java"`, `trust_workspace: true`, and a Maven coordinate such as `org.example:codec:2.1`. Set `file` to a module directory or `pom.xml` in a reactor; omit it to target the root POM. Existing ancestor `<dependencyManagement>` entries produce a versionless module dependency. Otherwise a versioned dependency is added to the nearest ancestor management section and the selected module; standalone POMs receive a versioned direct dependency. The operation edits POM XML without running Maven and notifies an already-running JDT LS session to refresh project configuration.

Use `semantic_stage` before a multi-turn Java change that must stay unpublished. Java mutations remain in the shared source overlay and return `stage_receipt.step_id`; copy that selector to `semantic_stage_drop` to remove one staged edit, use `semantic_stage_abort` to discard the session, or use `semantic_commit` to publish and run Java diagnostics. Staging requires a retained MCP session. The standalone `stage`, `stage-drop`, `stage-abort`, and `commit` CLI commands fail with a session-required error and do not publish changes.

---

## 1. Macro Refactorings in Java

`semedit` core operations for Java are indexed in the primary `SKILL.md` table. High-level patterns (e.g., Class $\to$ Record, Record $\to$ Class, Encapsulate Field) are composed by the agent using core operations and verified via Maven.

---

## 2. Java Refactoring Patterns

### Pattern A: Modernizing Class to Record (Class $\to$ Record)

When migrating an immutable data carrier class (POJO/DTO) to a Java 16+ `record`:

1. **Preflight & Inspection**:
   - Use `semantic_outline` and `semantic_inspect_symbol` to inspect the target class: ensure fields are `private final`, state is immutable, and the class does not extend an external class (`extends Object` only).
   - Verify serialization/reflection dependencies (e.g., JPA entities require mutable classes with no-arg constructors and cannot become records; Jackson DTOs may need `@JsonNaming` or record creator annotations).
2. **Atomic Transformation**:
   - Convert the class header to record syntax:

     ```java
     // Before
     public final class Point {
         private final int x;
         private final int y;
         public Point(int x, int y) { this.x = x; this.y = y; }
         public int getX() { return x; }
         public int getY() { return y; }
     }

     // After
     public record Point(int x, int y) {}
     ```

   - Eliminate redundant canonical constructors, standard getters (`getX()`), `equals()`, `hashCode()`, and `toString()`.
   - Preserve validation logic in a compact constructor:

     ```java
     public record BoundedRange(int min, int max) {
         public BoundedRange {
             if (min > max) throw new IllegalArgumentException("min > max");
         }
     }
     ```

3. **Call-Site Adjustment & Renaming**:
   - Update call sites from JavaBean getter access (`point.getX()`) to record component access (`point.x()`). Use `semantic_rename` with `file` and `trust_workspace: true` when renaming symbols across trusted workspace files.
4. **Verification**:
   - Run `semantic_verify` with `format: true` and `organize_imports: true` to format the modified file and clean unused imports.
   - Execute `semantic_maven_compile` with `trust_workspace: true` to ensure zero compilation or caller breaks.

---

### Pattern B: De-recording (Record $\to$ Class)

When a `record` outgrows its immutable constraints (e.g., requires mutable caches, lazy initialization, or inheritance hierarchy):

1. **Inspection**:
   - Use `semantic_lookup` and `semantic_inspect_symbol` to capture current record components and constructor invariants.
2. **Expansion**:
   - Convert `record Point(int x, int y)` into a standard class with explicit `private final` fields.
   - Synthesize canonical constructor, component accessors (`public int x() { return x; }`), and explicit `equals`/`hashCode`/`toString` implementations.
3. **Verification**:
   - Format source with `semantic_verify` and compile via `semantic_maven_compile`. Because existing callers invoke `point.x()`, accessor compatibility is maintained without breaking downstream callers.

---

### Pattern C: Encapsulate Field

When converting exposed public or package-private fields into encapsulated properties:

1. **Preflight Inspection**:
   - Use `semantic_lookup` and `semantic_inspect_symbol` to check field access and annotations.
2. **Accessor Addition & Refactoring**:
   - Introduce explicit getter: `public T getFieldName() { return this.fieldName; }`.
   - Introduce explicit setter: `public void setFieldName(T value) { this.fieldName = value; }`.
   - Update field visibility from `public` to `private`.
   - Use `semantic_rename` with `file` and `trust_workspace: true` to rename callers where method signatures match.
3. **Verification**:
   - Format with `semantic_verify` and run `semantic_maven_test` to confirm behavior and unit test passes.
