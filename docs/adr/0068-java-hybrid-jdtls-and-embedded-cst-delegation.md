<!-- WHY: Records the architectural decision to achieve Java parity with Go via persistent JDT LS delegation, embedded pure-Go Tree-sitter CST range resolution, and dependencyManagement-aware Maven POM mutations. -->
# Architecture Decision Record (ADR) 0068: Hybrid Java Delegation via Persistent JDT LS and Embedded Pure-Go CST Locator

* **Status**: Accepted
* **Date**: 2026-10-13

## Context

 Abbreviations on this page include Architecture Decision Record (ADR), Abstract Syntax Tree (AST), Concrete Syntax Tree (CST), Command-Line Interface (CLI), Central Processing Unit (CPU), Input/Output (I/O), Java Development Kit (JDK), Java Development Tools (JDT), Java Virtual Machine (JVM), Language Server (LS), Language Server Protocol (LSP), Large Language Model (LLM), Model Context Protocol (MCP), Project Object Model (POM), Remote Procedure Call (RPC), Secure Hash Algorithm (SHA), Uniform Resource Identifier (URI), Unicode Transformation Format (UTF), and WebAssembly (WASM).

`semedit` provides production-grade semantic operations for Go (`LanguageGo`), whereas Java (`LanguageJava`) currently offers a selected-file preview covering single-file `lookup`, `outline`, `inspect_symbol`, `rename`, `verify`, and root-POM Maven compile/test execution ([ADR-0025](0025-java-file-scoped-lookup-backend.md), [ADR-0031](0031-java-selected-file-rename.md), [ADR-0035](0035-java-maven-import-boundary.md), [ADR-0036](0036-java-bounded-verify-actions.md), [ADR-0037](0037-java-bounded-maven-actions.md)).

Bringing Java to parity with Go requires resolving four structural tensions while preserving [ADR-0020](0020-exclusive-golang-runtime-and-tooling-ecosystem.md) (`CGO_ENABLED=0`, pure-Go host binary):

1. **Delegation Over Reimplementation**: Eclipse JDT LS natively provides type-checked refactorings (`refactor.extract.*`, `refactor.inline.*`, `source.overrideMethods`, `refactor.changeSignature`), workspace-wide rename, reference discovery, formatting, and import organisation. Reimplementing a Java type-checker or refactoring engine in Go duplicates JDT LS capabilities.
2. **Sub-Method Construct Addressing**: JDT LS `textDocument/documentSymbol` stops at class, field, constructor, and method declarations, and `textDocument/selectionRange` emits untyped parent ranges that cannot enumerate sub-method constructs (`loop`, `if`, `else`, `switch`, `try_catch`, `try_with_resources`, `synchronized`, `element`) without a text anchor.
3. **Method and Constructor Overloading**: Java supports method and constructor overloading, requiring deterministic selector syntax that stays lenient on unambiguous names ([ADR-0065](0065-lenient-in-precise-out-input-resolution.md)) while disambiguating overloads cleanly.
4. **Multi-Module Maven Reactor Dependency Management**: Enterprise Java repositories manage versions centrally in parent or root `<dependencyManagement>` sections while omitting `<version>` tags in child module `<dependencies>`.

## Decision

`semedit` adopts a **hybrid delegation-first architecture** for Java parity:

1. **Persistent MCP-Owned JDT LS Session with Default Trusted Maven Reactor Import**:
   * Each MCP server instance maintains one persistent JDT LS session per canonical trusted Java workspace root (mirroring [ADR-0060](0060-mcp-owned-go-diagnostic-sessions.md) for `gopls`).
   * Mutations keep the JDT LS session warm across commits and batch steps, synchronising uncommitted `sourcebuf` overlays via `textDocument/didOpen` and `textDocument/didChange` and publishing disk commits via `textDocument/didSave`.
   * When a trusted workspace root contains a `pom.xml`, Maven reactor import activates by default (`import_maven: true`), storing all JDT LS workspace and M2E state strictly under `.scratch/`.
2. **Pure-Go Embedded Tree-sitter CST Locator for Sub-Method Ranges**:
   * `semedit` embeds a pure-Go Tree-sitter Java grammar (`CGO_ENABLED=0`, via `wazero` or `gotreesitter` with `//go:embed`) strictly as a **read-only sub-method range locator**.
   * Given a method declaration slice from JDT LS `documentSymbol`, the CST locator resolves `construct_path` (e.g. `loop[0]/if[1]`) and `discriminator` selectors into exact UTF-16 `Range` coordinates for `semantic_inspect_construct`, `semantic_insert_construct`, `semantic_replace_construct`, `semantic_remove_construct`, `semantic_insert_case`, and JDT LS `textDocument/codeAction` refactoring requests.
   * All formatting, import organisation, type checking, reference proofs, and diagnostic evaluation remain delegated to JDT LS.
3. **Overload-Aware Symbol Addressing and Enclosing-Type Defaults**:
   * Method and constructor selectors accept bare names (e.g. `Widget.process`) when the enclosing type declares a single method with that name, and accept parameter-type signature qualification (e.g. `Widget.process(String, int)`) when overloaded.
   * `semantic_outline`, `semantic_inspect_symbol`, and ambiguity candidate lists emit copyable signature-qualified symbols whenever overloads exist.
   * Declaration insertions (`semantic_insert_function`, `semantic_insert_field`, `semantic_insert_decl`, `semantic_insert_type`) default `in_type` to the file's sole top-level type when omitted, requiring explicit `in_type` only for nested types or multi-type files.
   * Within an enclosing Java type, placement partitions members by kind first (constants and fields $\to$ constructors $\to$ methods $\to$ nested types) and by visibility tier second (`public` $\to$ `protected` $\to$ `package-private` $\to$ `private`).
4. **Workspace-Wide Rename with Standalone `RenameFile` Support**:
   * `semantic_rename` accepts validated multi-file `TextDocumentEdit` sets across the trusted Java workspace and computes pre/post diagnostic deltas.
   * Standalone `semantic_rename` calls accept a single validated `RenameFile` resource operation when renaming a `public` top-level Java class, interface, enum, or record, updating the returned continuation `file` to the new `.java` path. Buffered batch steps refuse `RenameFile` resource operations.
5. **Intercepted CodeAction Edits and Follow-Up In-Buffer Naming**:
   * Refactorings delegated via `textDocument/codeAction` and `workspace/executeCommand` intercept both direct `CodeAction.edit` payloads and server-to-client `workspace/applyEdit` RPCs into `sourcebuf.Buffer`.
   * When JDT LS synthesises a default identifier during extraction (`refactor.extract.function`, `refactor.extract.variable`, `refactor.extract.constant`), `semedit` chains an in-buffer `textDocument/rename` on the newly created declaration range so the caller's requested `symbol` name applies atomically.
6. **Reactor-Smart `semantic_add_build_dependency` and Intentional `assertion_mode` Asymmetry**:
   * `semantic_add_build_dependency` supports Maven projects (`groupId:artifactId[:version][:scope][:classifier]`). When targeting a child module in a Maven reactor, the planner inspects ancestor `<dependencyManagement>` blocks: if managed, the child `pom.xml` receives a versionless `<dependency>`; if unmanaged and an ancestor POM declares `<dependencyManagement>`, the planner registers the versioned coordinate in the ancestor `<dependencyManagement>` and inserts the versionless `<dependency>` into the child `pom.xml`, then notifies JDT LS via `java/projectConfigurationUpdate`.
   * `semantic_assertion_mode` remains intentionally Go-specific, documented in the capability matrix per [ADR-0028](0028-cross-language-cli-txtar-coverage.md).

## Invariants

* The `semedit` binary builds and passes `make check` with `CGO_ENABLED=0` and zero custom Java OSGi plugin compilation steps.
* JDT LS never mutates workspace source files directly on disk; every edit passes through `sourcebuf.Buffer` validation and `pipeline.WriteAtomic`.
* Multi-file edits and `RenameFile` targets must reside canonically within the trusted workspace root.
* Every newly advertised Java operation includes both deterministic CLI `.txtar` contract coverage (with `cmd/docgen` rendering `.java` examples) and real-JDT LS integration test coverage against multi-package and multi-module Maven fixtures.

## Consequences

* **Positive**: Achieves full functional parity with Go while delegating type-checked refactorings, formatting, imports, and diagnostics to JDT LS; reuses the embedded pure-Go Tree-sitter CST locator pattern across future language backends (Rust, Kotlin, Scala, Bash).
* **Negative**: Chaining in-buffer renames after JDT LS extraction code actions adds one extra round-trip over the warm JDT LS stdio session.
