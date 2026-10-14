<!-- WHY: Provides the complete implementation handover, delegation protocols, example Java project specifications, and testing contracts to bring Java support to parity with Go. -->
# RQ-0068: Java Parity and JDT LS Delegation Implementation Handover

* **Status**: Resolved
* **Category**: Java Backend & Refactoring Handover
* **Date**: 2026-10-13

Abbreviations on this page include Architecture Decision Record (ADR), Application Programming Interface (API), Abstract Syntax Tree (AST), Concrete Syntax Tree (CST), Command-Line Interface (CLI), Central Processing Unit (CPU), Data Transfer Object (DTO), Input/Output (I/O), Integrated Development Environment (IDE), Java Archive (JAR), Java Development Kit (JDK), Java Development Tools (JDT), Java Virtual Machine (JVM), JavaScript Object Notation (JSON), Language Server (LS), Language Server Protocol (LSP), Large Language Model (LLM), Model Context Protocol (MCP), Open Services Gateway Initiative (OSGi), Plain Old Java Object (POJO), Project Object Model (POM), Remote Procedure Call (RPC), Secure Hash Algorithm (SHA), Uniform Resource Identifier (URI), Unicode Transformation Format (UTF), WebAssembly (WASM), and Extensible Markup Language (XML).

This document provides the complete engineering handover to bring `LanguageJava` from its current `Selected-file refactoring preview` tier ([`internal/backend/java/java.go`](../../internal/backend/java/java.go)) to `Production` parity with `LanguageGo` under [ADR-0068](../adr/0068-java-hybrid-jdtls-and-embedded-cst-delegation.md).

---

## 1. Executive Summary & Delegation Matrix

Rather than reimplementing a Java compiler or refactoring engine in Go, `semedit` acts as a deterministic orchestrator across three components:

1. **Persistent JDT LS Session (`internal/backend/java`)**: Owns workspace indexing, symbol hierarchy (`textDocument/documentSymbol`, `workspace/symbol`), reference proofs (`textDocument/references`), workspace-wide renames (`textDocument/prepareRename`, `textDocument/rename`), code-action refactorings (`textDocument/codeAction`, `workspace/executeCommand`), source formatting (`textDocument/formatting`), import organisation (`source.organizeImports`), and compiler diagnostics (`textDocument/publishDiagnostics`).
2. **Embedded Pure-Go Tree-sitter CST Locator (`internal/javacst`)**: Uses an embedded `tree-sitter-java` grammar (`CGO_ENABLED=0`) strictly as a read-only sub-method range resolver. It maps `construct_path` (e.g. `loop[0]/if[1]`) and `discriminator` snippets inside a method's JDT LS `documentSymbol` range into exact UTF-16 `Range` coordinates.
3. **Reactor-Aware Maven POM Planner (`internal/maven/pomedit`)**: Parses and updates root, parent, and child `pom.xml` files for `semantic_workspace_outline` and `semantic_add_build_dependency` while respecting `<dependencyManagement>` inheritance.

| Operation Family | Public Operations | Execution Owner | AST / CST Role |
| :--- | :--- | :--- | :--- |
| **Workspace & Package Orientation** | `semantic_workspace_outline`, directory-scoped `semantic_outline`, `semantic_lookup`, `semantic_inspect_symbol` | JDT LS (`workspace/symbol`, multi-file `documentSymbol`) + `pom.xml`/`package-info.java` scanner | None (JDT LS hierarchical symbols + header scan). |
| **References & Safe Delete Proof** | `semantic_find_references`, `semantic_remove_construct(safe_delete=true)` | JDT LS (`textDocument/references` with `includeDeclaration: false`) | None (JDT LS reference proof across reactor). |
| **Rename, Verify, & Imports** | `semantic_rename` (multi-file + standalone `RenameFile`), `semantic_verify` (file and scope), `semantic_organize_imports` | Persistent JDT LS (`rename`, `formatting`, `source.organizeImports`, versioned diagnostics) | None (validated `WorkspaceEdit` over `sourcebuf`). |
| **Declaration Lifecycle** | `semantic_scaffold_file`, `semantic_insert_declaration`, `semantic_insert_function`, `semantic_insert_type`, `semantic_insert_decl`, `semantic_insert_field`, `semantic_replace_decl`, `semantic_replace_body`, `semantic_remove_construct`, `semantic_set_doc_comment` | Orchestrated in `sourcebuf` using JDT LS `documentSymbol` ranges + JDT LS `formatting` and `source.organizeImports` | Embedded CST locates exact `{ ... }` body braces and leading Javadoc attachment ranges. |
| **Sub-Method Construct Lifecycle** | `semantic_inspect_construct`, `semantic_insert_construct`, `semantic_replace_construct`, `semantic_remove_construct`, `semantic_insert_case` | Embedded CST resolves target UTF-16 `Range`; `sourcebuf` splices snippet; JDT LS formats, organises imports, and verifies diagnostics | Embedded CST enumerates and resolves `loop`, `if`, `else`, `switch`, `case`, `try_catch`, `try_with_resources`, `synchronized`, `statement/expression`, and `element`. |
| **Type-Checked Refactorings** | `semantic_extract_function`, `semantic_extract_variable`, `semantic_extract_constant`, `semantic_inline_variable`, `semantic_inline_function`, `semantic_implement_contract`, `semantic_extract_interface`, `semantic_change_signature`, `semantic_introduce_parameter`, `semantic_introduce_parameter_object`, `semantic_move_construct` | JDT LS `textDocument/codeAction` and `workspace/executeCommand` (intercepting `workspace/applyEdit` into `sourcebuf`) + chained in-buffer `textDocument/rename` | Embedded CST maps `discriminator` / `construct_path` to the exact UTF-16 selection `Range` required by JDT LS `codeAction`. |
| **Staging & Batch Execution** | `semantic_batch`, `semantic_stage`, `semantic_stage_drop`, `semantic_stage_abort`, `semantic_commit` | Shared `internal/sourcebuf` + JDT LS `didOpen`/`didChange` unsaved buffer synchronisation | None (`sourcebuf` overlay coherence). |
| **Build Dependency Management** | `semantic_add_build_dependency` | `internal/maven/pomedit` (`<dependencyManagement>`-aware XML mutation) + JDT LS `java/projectConfigurationUpdate` | Deterministic XML token editing preserving comments and formatting. |

---

## 2. Parallel Implementation Tracks & Ownership

Workers execute in isolated `.scratch/worktrees/java-*` checkouts and integrate sequentially through `make check`:

| Track | Branch / Worktree | Owned Packages & Files | Prerequisite Track |
| :--- | :--- | :--- | :--- |
| **Track 1: Persistent JDT LS Session, Overlays, & Staging** | `.scratch/worktrees/java-session` | `internal/backend/java/session_manager.go`, `internal/backend/java/java.go`, `internal/operation/wire_staging.go` | Baseline `main` |
| **Track 2: Read Parity, References, & Overload Addressing** | `.scratch/worktrees/java-reads` | `internal/backend/java/symbols.go`, `internal/backend/java/references.go`, `internal/operation/wire_workspace_outline.go`, `internal/operation/wire_read_references.go` | Track 1 |
| **Track 3: Multi-File Rename, Scope Verify, & Organize Imports** | `.scratch/worktrees/java-rename-verify` | `internal/backend/java/rename.go`, `internal/backend/java/verify.go`, `internal/operation/wire_engine_imports.go` | Track 1 |
| **Track 4: Embedded Pure-Go Java CST Locator** | `.scratch/worktrees/java-cst` | `internal/javacst/*.go` (zero-CGO embedded grammar + sub-method construct path resolver) | Baseline `main` |
| **Track 5: Declaration & Sub-Method Construct Lifecycle** | `.scratch/worktrees/java-constructs` | `internal/backend/java/declarations.go`, `internal/backend/java/constructs.go`, `internal/operation/wire_engine_*.go`, `internal/operation/wire_remove.go` | Tracks 1, 2, 4 |
| **Track 6: JDT LS CodeAction Refactorings** | `.scratch/worktrees/java-refactorings` | `internal/backend/java/codeactions.go`, `internal/backend/java/refactor.go`, `internal/operation/wire_extract*.go`, `internal/operation/wire_signature*.go`, `internal/operation/wire_contract*.go` | Tracks 1, 3, 4 |
| **Track 7: Reactor-Smart Maven Dependency Management** | `.scratch/worktrees/java-maven-deps` | `internal/maven/pomedit/*.go`, `internal/operation/wire_engine_imports.go` (`add_build_dependency`) | Baseline `main` |
| **Track 8: Docgen, Reference Fixtures, & Skill Sync** | `.scratch/worktrees/java-docs-tests` | `cmd/docgen/txtar.go`, `main_test.go`, `testdata/scripts/*_java.txtar`, `internal/backend/java/real_server_integration_test.go`, `skills/semedit/SKILL.md`, `skills/semedit/references/java.md` | Tracks 1–7 |

---

## 3. Detailed Component Specifications

### 3.1 Track 1: Persistent JDT LS Session, `sourcebuf` Overlays, & Maven Import

1. **Session Retention & Lifecycle**:
   * Replace the single-operation `b.invalidateJavaSessionLocked(session)` teardown in [`JavaBackend.Rename`](../../internal/backend/java/java.go) and [`JavaBackend.Verify`](../../internal/backend/java/java.go) with a persistent per-root session manager modelled on [`internal/godiagnostics`](../../internal/godiagnostics).
   * Track open documents and monotonic document versions (`map[string]int`) inside `javaProcessSession`.
   * Before executing any query or refactoring, synchronise dirty `sourcebuf.Buffer` overlays to JDT LS via `textDocument/didOpen` (version 1) or `textDocument/didChange` (incrementing version $v+1$).
   * When `sourcebuf.Buffer.Flush` writes files atomically to disk, send `textDocument/didChange` followed by `textDocument/didSave`. If a batch child or standalone preflight fails and discards its working overlay, restore the JDT LS document state to the surviving buffer content via `textDocument/didChange`.
2. **Default Trusted Maven Reactor Import**:
   * In `effectiveJavaConfig(project)`, when `project.WorkspaceTrust.Allows(root)` holds and `pom.xml` exists at `root`, default `ImportMaven` to `true` unless the caller explicitly sets `import_maven: false`.
   * Store JDT LS `-data` workspace state and M2E indices under `<root>/.scratch/jdtls-data/<fingerprint>/`.
3. **Reverse RPC Interception (`workspace/applyEdit`)**:
   * Register a server-request handler on `lsp.Client` for `workspace/applyEdit` and `window/showMessageRequest`.
   * When a refactoring command triggers `workspace/applyEdit` from JDT LS, capture the `ApplyWorkspaceEditParams.Edit` payload into the active request context and return `{"applied": true}` to JDT LS without writing to disk, allowing `semedit` to validate and stage the edit inside `sourcebuf.Buffer`.

### 3.2 Track 2: Overload-Aware Addressing, `workspace_outline`, Scope Reads, & `find_references`

1. **Overload-Aware Symbol Selector Grammar**:
   * Normalise Java method and constructor `DocumentSymbol` entries into two canonical fields:
     * `BaseName`: identifier without parameter list (e.g. `process`).
     * `SignatureName`: identifier with erased simple parameter types (e.g. `process(String, int)`).
   * When `selectJavaSymbol` or `matchJavaReadSymbols` evaluates a `symbol` query:
     * If `symbol` includes parentheses (e.g. `OrderService.process(Order, boolean)`), match against `SignatureName` (tolerating whitespace differences after commas per [ADR-0065](../adr/0065-lenient-in-precise-out-input-resolution.md)).
     * If `symbol` omits parentheses (e.g. `OrderService.process`), match against `BaseName`. If multiple overloads match, return `LookupResult{Candidates: ...}` where each candidate's `QualifiedName` uses `SignatureName` (`OrderService.process(Order)` and `OrderService.process(Order, boolean)`).
2. **`semantic_workspace_outline` for Java**:
   * Register `backend.LanguageJava` (and `backend.LanguageAuto` detection when `pom.xml` or `.java` files exist) in [`wire_workspace_outline.go`](../../internal/operation/wire_workspace_outline.go).
   * Discover root and nested `pom.xml` modules (recording `<groupId>:<artifactId>:<version>` and `<packaging>`), scan `src/main/java` and `src/test/java` (when `include_tests: true`) for Java packages, and extract package synopses from `package-info.java` Javadoc comments or the primary public type's Javadoc.
3. **Directory-Scoped `outline`, `lookup`, and `inspect_symbol`**:
   * Lift the single-file restriction in `resolveJavaReadSelection` so `path` / `file` accepts package directories (per [ADR-0065](../adr/0065-lenient-in-precise-out-input-resolution.md)).
   * Support `include_unexported: false` by filtering out `private` and `package-private` declarations using JDT LS symbol modifier flags or declaration keywords.
4. **`semantic_find_references`**:
   * Register `backend.LanguageJava` in [`wire_read_references.go`](../../internal/operation/wire_read_references.go) and [`language_contracts.go`](../../internal/operation/language_contracts.go).
   * Resolve the target declaration via overload-aware `Lookup`, then invoke JDT LS `textDocument/references` with `context: {"includeDeclaration": false}`. Validate that all returned URIs reside within the trusted workspace root and project them into `backend.ReferencesResult`.

### 3.3 Track 3: Workspace-Wide Rename, Scope Verify, & `semantic_organize_imports`

1. **Workspace-Wide `semantic_rename` & Standalone `RenameFile`**:
   * Advertise `workspace.workspaceEdit.documentChanges: true` and `workspace.workspaceEdit.resourceOperations: ["rename"]` during JDT LS `initialize`.
   * Validate all `TextDocumentEdit` entries across every affected `.java` file in the trusted workspace root against current `sourcebuf` preimages.
   * When renaming a `public` top-level type produces a `RenameFile` operation (`oldUri` $\to$ `newUri`):
     * In a buffered batch (`cc.BatchBuffer != nil`), reject `RenameFile` with a clear sentinel error (`ErrJavaBatchResourceRenameUnsupported`).
     * In a standalone call, validate that `newUri` lies inside the same package directory, does not collide with an existing file, write the updated content atomically to `newPath`, remove `oldPath`, notify JDT LS via `workspace/didRenameFiles`, and return `newPath` in `result.target.file`.
   * Compute pre- and post-rename workspace diagnostics across all affected files to populate `DiagnosticDelta` (`GuaranteeDiagnosticDelta`).
2. **Scope-Wide `semantic_verify` & Standalone `semantic_organize_imports`**:
   * Allow `semantic_verify` to accept a directory or workspace scope (`VariantSelectedScope`), formatting and organising imports across all selected `.java` files when requested.
   * Enable `backend.LanguageJava` on `semantic_organize_imports` (`semedit imports`). Support `--add <fqcn>` (inserting `import <fqcn>;` before invoking JDT LS `source.organizeImports` + `textDocument/formatting`) and `--remove <fqcn>` (removing matching import lines and formatting).

### 3.4 Track 4: Embedded Pure-Go Java CST Locator (`internal/javacst`)

1. **Zero-CGO Grammar Integration**:
   * Create `internal/javacst` using a pure-Go Tree-sitter runtime (`CGO_ENABLED=0`) and an embedded `tree-sitter-java` grammar artifact.
2. **Canonical Construct Mapping**:
   * Map Java CST node types to `backend.ConstructKind`:
     * `for_statement`, `enhanced_for_statement`, `while_statement`, `do_statement` $\to$ `ConstructLoop` (`"loop"`)
     * `if_statement` $\to$ `ConstructIf` (`"if"`) and `else` consequence $\to$ `ConstructElse` (`"else"`)
     * `switch_expression`, `switch_statement` $\to$ `ConstructSwitch` (`"switch"`), `switch_block_statement_group` / `switch_rule` $\to$ `ConstructCase` (`"case"`)
     * `try_statement` $\to$ `ConstructTryCatch` (`"try_catch"`), `try_with_resources_statement` $\to$ `ConstructTryWithResources` (`"try_with_resources"`), `catch_clause` / `finally_clause` $\to$ `clause`
     * `synchronized_statement` $\to$ `ConstructSynchronized` (`"synchronized"`)
     * `Map.of(...)`, `Map.ofEntries(...)`, `List.of(...)`, `Set.of(...)`, and array/collection initialiser entries $\to$ `ConstructElement` (`"element"`)
     * Statements and expressions inside a block $\to$ `ConstructStatementOrExpr` (`"statement/expression"`)
3. **Range & Path API**:
   * Expose `InspectConstructs(source []byte, methodRange backend.Range, kind backend.ConstructKind, discriminator string) ([]ConstructCandidate, error)` returning 0-indexed hierarchical paths (e.g. `loop[0]/if[1]`), exact UTF-16 `Range`s, and SHA-256 `expected_revision` hashes.

### 3.5 Track 5: Declaration & Sub-Method Construct Lifecycle

1. **`semantic_scaffold_file`**:
   * Infer the Java `package <pkg>;` declaration from sibling `.java` files in the target directory or from the relative path beneath `src/main/java` or `src/test/java`.
2. **Enclosing-Type Resolution & Member Section Partitioning**:
   * When `in_type` is omitted on `semantic_insert_function`, `semantic_insert_field`, `semantic_insert_decl`, or `semantic_insert_type`, resolve the file's sole top-level type via JDT LS `documentSymbol`; if the file declares zero or multiple top-level types (and `kind` is not a new top-level type), fail closed with candidate `in_type` selectors.
   * Partition class members by kind tier:
     1. Static constants and fields (`static final` $\to$ `static`)
     2. Instance fields
     3. Constructors
     4. Methods
     5. Nested types (`class`, `interface`, `enum`, `record`)
   * Within each kind tier, order by visibility (`public` $\to$ `protected` $\to$ `package-private` $\to$ `private`) per [ADR-0012](../adr/0012-access-modifiers-and-section-placement.md).
   * After splicing into `sourcebuf.Buffer`, invoke JDT LS `textDocument/formatting` and `source.organizeImports` on the working buffer before committing.
3. **`semantic_remove_construct` and `safe_delete`**:
   * For declarations (methods, constructors, fields, nested types) and sub-method constructs (`case`, `loop`, `if`), locate the full node range (including attached leading Javadoc and annotations via `internal/javacst`).
   * When `safe_delete: true`, run JDT LS `textDocument/references` (`includeDeclaration: false`) across the workspace reactor and verify zero external usages before deleting.

### 3.6 Track 6: JDT LS CodeAction Refactorings

1. **Extraction Operations (`extract_function`, `extract_variable`, `extract_constant`)**:
   * Resolve the target range inside `in_function`:
     * `extract_function`: validate `start_line` and `end_line` inside the method body via `internal/javacst`.
     * `extract_variable` and `extract_constant`: resolve `discriminator` and optional `construct_path` inside `in_function` via `internal/javacst` to an exact UTF-16 `Range`.
   * Request `textDocument/codeAction` from JDT LS with `context.only` set to `["refactor.extract.function"]`, `["refactor.extract.variable"]`, or `["refactor.extract.constant"]`.
   * Capture the resulting `WorkspaceEdit` (either directly from `CodeAction.edit` or intercepted via `workspace/executeCommand` $\to$ `workspace/applyEdit`) and apply it to `sourcebuf.Buffer`.
   * Locate the newly synthesized method/variable/constant identifier in the modified buffer and execute an immediate in-buffer `textDocument/rename` to the caller's requested `symbol` name, followed by JDT LS `formatting` and `source.organizeImports`.
2. **Inlining (`inline_variable`, `inline_function`)**:
   * Resolve the variable or method declaration via JDT LS `documentSymbol` / `internal/javacst`, request `textDocument/codeAction` with `only: ["refactor.inline"]`, validate preimages across all affected files, and apply to `sourcebuf.Buffer`.
3. **Contracts & Signatures (`implement_contract`, `extract_interface`, `change_signature`, `introduce_parameter`, `introduce_parameter_object`, `move_construct`)**:
   * `implement_contract`: Ensure `implements <Interface>` is present on `in_type` in the buffer, then use JDT LS `java/listOverridableMethods` and `java/addOverridableMethods` to select and generate the unimplemented methods. The `source.overrideMethods` action opens an interactive method picker and is not suitable for this deterministic operation.
   * `extract_interface`: Inspect selected public methods on `in_type` via JDT LS `documentSymbol`, synthesize the interface declaration, add `implements <Interface>` to `in_type`, and verify via JDT LS diagnostics.
   * `change_signature`, `introduce_parameter`, `introduce_parameter_object`: Use JDT LS `refactor.changeSignature` command or resolve all callers via `textDocument/references`, rewrite call sites and declaration signature in `sourcebuf.Buffer`, and verify zero introduced JDT LS errors across the workspace overlay.
   * `move_construct`: Move top-level or static declarations between files/packages, updating `package` and `import` statements across all `textDocument/references` callers in `sourcebuf.Buffer`.

### 3.7 Track 7: Reactor-Smart Maven `semantic_add_build_dependency`

1. **Coordinate Grammar & Target Selection**:
   * Accept `dependency` in Maven format `groupId:artifactId[:version][:scope][:classifier]` alongside an optional `file` or `path` selecting the target submodule in a multi-module reactor (defaulting to root `pom.xml`).
2. **Ancestor `<dependencyManagement>` Resolution**:
   * Parse the target module's `pom.xml` and walk its `<parent>` / reactor ancestor chain up to the trusted workspace root `pom.xml`.
   * **Case A (Already Managed)**: If an ancestor POM's `<dependencyManagement><dependencies>` already declares `(groupId, artifactId)` (matching optional `classifier`), insert `<dependency><groupId>...</groupId><artifactId>...</artifactId></dependency>` into the target module's `<dependencies>` **without `<version>`** (and omitting `<scope>` or `<classifier>` unless explicitly overriding the managed entry).
   * **Case B (Unmanaged in a Managed Reactor)**: If `(groupId, artifactId)` is absent from ancestor `<dependencyManagement>`, a `version` is provided, and the root/parent POM contains a `<dependencyManagement>` section, insert the versioned `<dependency>` into the ancestor `<dependencyManagement><dependencies>` and insert the versionless `<dependency>` into the target module's `<dependencies>`.
   * **Case C (Standalone POM)**: If the project is a single-module POM without `<dependencyManagement>`, insert the full versioned `<dependency>` directly into `<dependencies>`.
3. **JDT LS Synchronisation**:
   * Write modified `pom.xml` files atomically via `pipeline.WriteAtomic` and send `workspace/didChangeWatchedFiles` / `java/projectConfigurationUpdate` to the active JDT LS session.

---

## 4. Example Java Projects & Testing Strategy

Every Java operation must be verified at three levels before completion:

### 4.1 Level 1: Hermetic Unit & CST Tests (`go test ./internal/javacst/... ./internal/backend/java/... ./internal/maven/...`)

* **CST Locator Tests (`internal/javacst/locator_test.go`)**: Verify UTF-16 range and `construct_path` extraction across Java 21 syntax: enhanced `for` loops, labeled loops, `switch` expressions with arrow rules (`case X ->`), `try`-with-resources, multi-catch (`catch (IOException | SQLException e)`), `synchronized` blocks, lambdas, text blocks (`"""`), and `Map.of` / `List.of` elements.
* **Fake-Session Backend Tests (`internal/backend/java_test.go`)**: Test session reuse across consecutive calls, `didOpen`/`didChange` version increments, `workspace/applyEdit` interception, overload ambiguity reporting, and multi-file preimage validation.
* **Maven POM Planner Tests (`internal/maven/pomedit/pomedit_test.go`)**: Verify Case A (already managed), Case B (register in parent `<dependencyManagement>` + versionless child entry), and Case C (standalone POM), ensuring XML comments and indentation survive untouched.

### 4.2 Level 2: CLI & MCP `.txtar` Contract Fixtures (`testdata/scripts/*_java.txtar`) and `cmd/docgen` Integration

1. **Updating `cmd/docgen/txtar.go` for Java Example Generation**:
   * Currently, `firstTxtarExampleFiles` in [`cmd/docgen/txtar.go`](../../cmd/docgen/txtar.go) falls back only to `.go` files when selecting input code for documentation rendering:

     ```go
     // Update firstTxtarExampleFiles in cmd/docgen/txtar.go to recognise all supported source extensions:
     for name, content := range files {
         ext := filepath.Ext(name)
         if (ext == ".go" || ext == ".java" || ext == ".rs" || ext == ".scala" || ext == ".kt" || ext == ".hs" || ext == ".sh" || name == "Makefile") && !strings.HasPrefix(name, "want") {
             return txtarFirstFiles{input: name, inputContent: content}
         }
     }
     ```

2. **Extending `runFakeJava` in `cmd/semedit/language_txtar_test.go`**:
   * Extend the deterministic JDT LS test stand-in (`runFakeJava` in [`language_txtar_test.go`](../../cmd/semedit/language_txtar_test.go)) to handle `textDocument/references`, multi-file `textDocument/rename` (including `RenameFile` for public classes), `textDocument/codeAction` (`refactor.extract.*`, `refactor.inline.*`, `source.overrideMethods`), and `workspace/symbol` so `make check` runs all Java `.txtar` scripts hermetically in milliseconds.
3. **Required `testdata/scripts/*_java.txtar` Suite**:
   * `workspace_outline_java.txtar`: Multi-module Maven reactor with `core/pom.xml`, `api/pom.xml`, and `package-info.java` synopses.
   * `outline_java_directory.txtar`: Directory-scoped package outline with `--include-unexported=false`.
   * `lookup_java_overloads.txtar`: Disambiguating `OrderService.process(Order)` vs `OrderService.process(Order, boolean)`.
   * `find_references_java.txtar`: Cross-package reference discovery across `src/main/java` and `src/test/java`.
   * `rename_java_multifile.txtar`: Cross-file method and field rename with diagnostic delta output, plus standalone public top-level class rename (`Widget.java` $\to$ `Gadget.java`).
   * `organize_imports_java.txtar`: Adding and removing imports via `semedit imports --language java`.
   * `scaffold_file_java.txtar`: Inferring `package com.example.billing;` from sibling `.java` files.
   * `insert_members_java.txtar`: Inserting fields, constructors, and methods into a Java class with `public` / `protected` / `package-private` / `private` section ordering.
   * `replace_body_java.txtar` & `replace_decl_java.txtar`: Replacing overloaded method bodies and field declarations.
   * `inspect_construct_java.txtar`, `insert_construct_java.txtar`, `insert_construct_java_loop.txtar`, `insert_construct_java_element.txtar`, `replace_construct_java.txtar`, and `insert_case_java.txtar`: Cover Java method-local constructs and collection/array initializer element appends with the shared inspect/edit selectors.
   * `remove_construct_java_safe_delete.txtar`: Removing unused private methods with `--safe-delete` and refusing deletion when `textDocument/references` reports active callers.
   * `extract_function_java.txtar`, `extract_variable_java.txtar`, `extract_constant_java.txtar`, `inline_variable_java.txtar`, and `inline_function_java.txtar`: Delegate extraction and inlining through JDT LS code actions plus chained rename.
   * `implement_contract_java.txtar` & `change_signature_java.txtar`: Generating `@Override` stubs for a Java interface and updating method signatures across callers.
   * `add_build_dependency_java_reactor.txtar`: Adding a managed dependency (version omitted in child `pom.xml`) and a new dependency (version registered in root `<dependencyManagement>`, versionless entry in child `pom.xml`).
   * `mcp_batch_java_shared_buffers.txtar` & `mcp_stage_java_lifecycle.txtar`: Multi-step MCP batch and stage/commit lifecycle on a Java project.

### 4.3 Level 3: Real-JDT LS & Maven Multi-Module Reference Project (`internal/backend/java/real_server_integration_test.go`)

Fake LSP sessions verify protocol wiring, but cannot prove real Eclipse JDT LS AST/code-action behaviour. Create `internal/backend/java/real_server_integration_test.go` (following the pattern in [`internal/backend/kotlin/real_server_integration_test.go`](../../internal/backend/kotlin/real_server_integration_test.go)), gated on `SEMEDIT_REAL_JDTLS_HOME` and `SEMEDIT_REAL_JAVA_HOME` (JDK 21+).

#### Reference Multi-Module Maven Fixture Layout

The integration test materialises a realistic two-module Maven 4 / Java 21 reactor under `.scratch/java-real-integration-*`:

```text
acme-ledger/
├── pom.xml                                  # Root reactor POM with <dependencyManagement>
├── ledger-domain/
│   ├── pom.xml                              # Domain module POM
│   └── src/main/java/com/acme/ledger/domain/
│       ├── package-info.java                # /** Core immutable ledger domain models and contracts. */
│       ├── Money.java                       # Java 21 record with compact constructor validation
│       ├── EntryType.java                   # Enum (DEBIT, CREDIT)
│       ├── LedgerEntry.java                 # Record using Money and EntryType
│       └── AuditSink.java                   # Interface contract (void record(LedgerEntry entry);)
└── ledger-service/
    ├── pom.xml                              # Depends on ledger-domain (version managed by root POM)
    ├── src/main/java/com/acme/ledger/service/
    │   ├── package-info.java                # /** Transactional ledger posting and balance services. */
    │   ├── InMemoryAuditSink.java           # Target for semantic_implement_contract (implements AuditSink)
    │   └── LedgerService.java               # Overloaded post(...) methods, loops, switch expressions, try-with-resources
    └── src/test/java/com/acme/ledger/service/
        └── LedgerServiceTest.java           # Caller fixture for cross-module rename, find_references, and maven_test
```

#### Reference Project Source Files

1. **Root `pom.xml`**:

   ```xml
   <project xmlns="http://maven.apache.org/POM/4.0.0"
            xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
            xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
     <modelVersion>4.0.0</modelVersion>
     <groupId>com.acme.ledger</groupId>
     <artifactId>acme-ledger-parent</artifactId>
     <version>1.0.0-SNAPSHOT</version>
     <packaging>pom</packaging>
     <properties>
       <maven.compiler.release>21</maven.compiler.release>
       <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
     </properties>
     <modules>
       <module>ledger-domain</module>
       <module>ledger-service</module>
     </modules>
     <dependencyManagement>
       <dependencies>
         <dependency>
           <groupId>com.acme.ledger</groupId>
           <artifactId>ledger-domain</artifactId>
           <version>${project.version}</version>
         </dependency>
         <dependency>
           <groupId>org.slf4j</groupId>
           <artifactId>slf4j-api</artifactId>
           <version>2.0.12</version>
         </dependency>
       </dependencies>
     </dependencyManagement>
   </project>
   ```

2. **`ledger-service/src/main/java/com/acme/ledger/service/LedgerService.java`**:

   ```java
   package com.acme.ledger.service;

   import com.acme.ledger.domain.AuditSink;
   import com.acme.ledger.domain.EntryType;
   import com.acme.ledger.domain.LedgerEntry;
   import com.acme.ledger.domain.Money;
   import java.util.ArrayList;
   import java.util.List;
   import java.util.Map;

   public class LedgerService {
       private final AuditSink auditSink;
       private final List<LedgerEntry> entries = new ArrayList<>();
       private final Map<String, Integer> limits = Map.of(
           "DAILY", 1000
       );

       public LedgerService(AuditSink auditSink) {
           this.auditSink = auditSink;
       }

       void post(LedgerEntry entry) {
           post(entry, true);
       }

       void post(LedgerEntry entry, boolean notifyAudit) {
           long cents = entry.amount().cents();
           if (cents <= 0) {
               throw new IllegalArgumentException("amount must be positive");
           }
           if (cents > 1000) {
               throw new IllegalArgumentException("amount exceeds posting limit");
           }
           entries.add(entry);
           if (notifyAudit) {
               auditSink.record(entry);
           }
       }

       public long netBalanceCents() {
           long total = 0L;
           for (LedgerEntry entry : entries) {
               long signed = switch (entry.type()) {
                   case CREDIT -> entry.amount().cents();
                   case DEBIT -> -entry.amount().cents();
               };
               total += signed;
           }
           return total;
       }
   }
   ```

#### End-to-End Scenarios Verified in `TestRealJavaJDTLSIntegration`

1. **Workspace & Package Orientation**: Run `semantic_workspace_outline` on `acme-ledger/`; assert both `ledger-domain` and `ledger-service` modules and both `package-info.java` synopses return in depth order.
2. **Overload Inspection & Reference Discovery**: Run `semantic_lookup` on `LedgerService.post` (asserting 2 overload candidates: `LedgerService.post(LedgerEntry)` and `LedgerService.post(LedgerEntry, boolean)`), then run `semantic_find_references` on `LedgerService.post(LedgerEntry, boolean)` and `Money.cents`.
3. **Contract Implementation**: Run `semantic_implement_contract` on `InMemoryAuditSink` for `AuditSink`; assert JDT LS generates `@Override public void record(LedgerEntry entry)`, then use `semantic_replace_body` on that method.
4. **Sub-Method Construct & Element Editing**: Run `semantic_inspect_construct` (`kind: "loop"`, `in_function: "LedgerService.netBalanceCents"`) and `semantic_insert_construct` (`kind: "element"`, `in_declaration: "LedgerService.limits"`, `source: "\"MONTHLY\", 5000"`).
5. **CodeAction Extraction + Chained Rename**: Run `semantic_extract_variable` in `LedgerService.post(LedgerEntry, boolean)` with `discriminator: "entry.amount().cents()"`, `symbol: "amountCents"`; run `semantic_extract_constant` with `discriminator: "1000"`, `symbol: "MAX_POSTING_CENTS"`; then run `semantic_introduce_parameter_object` for the same-package method and verify both main and test callers preserve argument order.
6. **Cross-Module Rename & Standalone Public Class Rename**:
   * Rename `AuditSink.record` $\to$ `AuditSink.append` and assert `AuditSink.java`, `LedgerService.java`, and `InMemoryAuditSink.java` update atomically in a single persistent JDT LS session with `net_delta: 0` diagnostics.
   * Rename `EntryType` $\to$ `PostingKind` and assert `EntryType.java` renames on disk to `PostingKind.java` while updating imports across `ledger-domain` and `ledger-service`.
7. **Reactor-Smart Dependency Insertion**:
   * Run `semantic_add_build_dependency` on `ledger-service/pom.xml` with `org.slf4j:slf4j-api:2.0.12` (already in root `<dependencyManagement>`); assert `ledger-service/pom.xml` omits `<version>`.
   * Run `semantic_add_build_dependency` on `ledger-service/pom.xml` with `com.google.guava:guava:33.0.0-jre`; assert root `pom.xml` `<dependencyManagement>` receives `<version>33.0.0-jre</version>` and `ledger-service/pom.xml` receives versionless `<dependency>`.

#### Commands to Execute Tests

Implementation and verification update (2026-10-13): all eight tracks are implemented.
The complete `go test ./...` suite passes with the repository-local tools on `PATH`,
a canonical `TMPDIR`, and local networking enabled for integration tests. Java
`.txtar` contracts pass, including Track 6 refactorings and cross-package public-type
moves. The real JDT LS test verifies the installed server's `moveResource` response,
package rewrite, caller import updates, and published resource rename. The orchestration
resolves the destination package from the selected destination file and reconciles
references under the active workspace root when JDT LS omits `RenameFile`.

Repository-wide checks remain red: `make check` reports 327 Go lint findings, and
`make check-ci` also rejects the archived benchmark result
`hard-smoke-luna6-medium-20261011T105400Z` because its v1 report has no duration unit.
These check failures do not affect the passing Go test suite or the Java integration.

```bash
# 1. Run standard formatting, linters, unit tests, and hermetic CLI txtar suite:
make check

# 2. Run focused Java backend unit and CST locator tests:
go test -v ./internal/javacst/... ./internal/backend/java/... ./internal/maven/...

# 3. Run CLI txtar contract tests for Java operations:
go test -v -run 'TestREQ001SemanticIntentCLIContracts/.*java.*' .

# 4. Run real JDT LS + JDK 21 multi-module integration suite:
SEMEDIT_REAL_JDTLS_HOME=/path/to/jdtls \
SEMEDIT_REAL_JAVA_HOME=/path/to/jdk-21 \
go test -v -run TestRealJavaJDTLSIntegration ./internal/backend/java/...

# 5. Verify documentation generation and skill reference links:
make check-ci
```
