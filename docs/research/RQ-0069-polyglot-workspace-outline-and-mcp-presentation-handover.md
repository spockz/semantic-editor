<!-- WHY: Provides the complete implementation handover, architectural invariants, and testing specifications to resolve polyglot workspace outline discovery, language detection hierarchy ascent, module description schemas, package-to-module bindings, MCP payload deduplication, and description token condensation. -->
# RQ-0069: Polyglot Workspace Outline, Language Detection, and MCP Presentation Implementation Handover

* **Status**: Resolved
* **Category**: Workspace Discovery & MCP Presentation Handover
* **Date**: 2026-10-13

Abbreviations on this page include Architecture Decision Record (ADR), Abstract Syntax Tree (AST), Command-Line Interface (CLI), Language Server (LS), Language Server Protocol (LSP), Model Context Protocol (MCP), Project Object Model (POM), Remote Procedure Call (RPC), Uniform Resource Identifier (URI), and JavaScript Object Notation (JSON).

This document provides the complete engineering handover for a Luna agent to implement the architectural corrections for `semantic_workspace_outline`, multi-language detection, MCP response deduplication, and schema description condensation following the October 2026 field evaluation on polyglot workspaces.

---

## 1. Executive Summary & Non-Negotiable Invariants

### 1.1 Core Problem Statement

During external evaluation of `semedit` on a polyglot repository (Go root module containing internal ownership subtrees alongside documentation and contract-test Java sources), the tool exposed critical architectural defects:

1. **Single-Language Exclusivity**: Automatic language detection operates as an exclusive waterfall. Detecting a `pom.xml` or `.java` source file suppresses Go discovery entirely, yielding seven Java package directories and zero Go modules.
2. **Missing Go in Tool Registration**: `backend.LanguageGo` is omitted from `workspaceOutlineDef.Handlers`, preventing explicit `language: "go"` selection in the generated schema enum.
3. **Sub-Tree Detection Refusal**: Targeting a sub-tree lacking an immediate root `go.mod` fails language detection with `operation is not supported for language` because `detectLanguage()` does not traverse parent directories to locate the enclosing module manifest and rejects multi-language presence.
4. **Sub-Module Scoping Inversion**: Scoping an outline to a sub-module path reports sub-module packages while continuing to report the repository root module under `Modules:`.
5. **Missing Module Metadata**: `WorkspaceModule` lacks descriptions, synopses, and package associations.
6. **Triplicate Payload Serialisation**: Directory and workspace outlines emit three identical full result representations in MCP tool calls (~1 MB per call).
7. **Description Context Bloat**: Hexadecimal fault bitmasks and 30-item downstream tool routing graphs inflate schema token overhead.

### 1.2 Non-Negotiable Invariants

1. **Strict Linter Invariant**: **No linters or lint rules may ever be modified, relaxed, disabled, or suppressed.** Do not edit `.golangci.yml`, Makefile lint flags, or linter configurations. All implementation tracks must conform strictly to existing static analysis rules.
2. **Clean `make check` Gate**: Every track and the integrated deliverable must pass `make check` cleanly before acceptance.
3. **Polyglot Workspace Invariant**: Real-world projects routinely contain multiple programming languages. The detection of a `pom.xml`, Gradle file, or `.java` file **never** precludes the existence of Go modules, and vice versa. In `auto` mode, workspace discovery must aggregate all active language ecosystems rather than selecting one and suppressing others.
4. **Lenient-In, Precise-Out Principle (ADR-0065)**: Inputs accept broad paths, subdirectories, or auto-detection; outputs report exact module boundaries, package paths, and language ownership.
5. **Zero-Cruft Migration Invariant**: Complete migrations in-place across all affected tools and tests. Do not leave transitional compatibility shims or legacy wrappers.

---

## 2. Parallel Implementation Tracks & Ownership

Workers execute in isolated `.scratch/worktrees/workspace-*` checkouts and integrate sequentially through `make check`:

| Track | Branch / Worktree | Owned Packages & Files | Prerequisite Track |
| :--- | :--- | :--- | :--- |
| **Track 1: Polyglot Language Registry & Hierarchy Ascent** | `.scratch/worktrees/workspace-lang-detect` | `internal/backend/backend.go`, `internal/operation/wire_read_project.go`, `internal/operation/dispatch.go` | Baseline `main` |
| **Track 2: Workspace Outline Polyglot Aggregator & Wire Alignment** | `.scratch/worktrees/workspace-polyglot-wire` | `internal/operation/wire_workspace_outline.go`, `internal/backend/read.go` | Track 1 |
| **Track 3: Go Scoped Module Hierarchy & `go.work` Integration** | `.scratch/worktrees/workspace-go-scoping` | `internal/astedit/workspace_outline.go`, `internal/astedit/workspace_outline_test.go` | Track 2 |
| **Track 4: Module Descriptors & Package-to-Module Associations** | `.scratch/worktrees/workspace-module-meta` | `internal/backend/read.go`, `internal/astedit/workspace_outline.go`, `internal/backend/java/workspace_outline.go` | Track 3 |
| **Track 5: MCP Critical Text Projection Pruning & Deduplication** | `.scratch/worktrees/workspace-mcp-dedup` | `internal/mcp/presentation/presentation.go`, `internal/mcp/server.go`, `internal/mcp/presentation_test.go` | Baseline `main` |
| **Track 6: Schema Description Condensation & Token Optimisation** | `.scratch/worktrees/workspace-schema-tokens` | `internal/capability/contract_model.go`, `internal/mcp/server.go`, `internal/mcp/schema_test.go` | Baseline `main` |
| **Track 7: Contract Txtar Fixtures, Docgen, & Skill Synchronisation** | `.scratch/worktrees/workspace-docs-tests` | `testdata/scripts/workspace_outline*.txtar`, `cmd/docgen/txtar.go`, `skills/semedit/SKILL.md`, `skills/semedit/references/go.md` | Tracks 1–6 |

---

## 3. Detailed Component Specifications

### 3.1 Track 1: Polyglot Language Registry & Hierarchy Ascent

**Files to modify:**

* `internal/backend/backend.go`
* `internal/operation/wire_read_project.go`
* `internal/operation/dispatch.go`

#### Concrete Implementation Steps

1. **Parent Manifest Ascent for Sub-Tree Scopes**:
   In `detectLanguage(project ProjectContext)` in [`internal/backend/backend.go`](../../internal/backend/backend.go):
   * When `project.File` or `project.RootDir` points to a subdirectory lacking a local `go.mod`, do not terminate detection immediately.
   * Traverse parent directories upwards (terminating at filesystem root or git boundary) searching for `go.mod` or `go.work`. If an enclosing Go manifest is discovered, record `LanguageGo` as a detected candidate for that subtree.
   * Similarly, ascend to locate parent Maven root `pom.xml` files with `<modules>` referencing the subtree.
2. **Multi-Language Disambiguation Without Failure**:
   * Replace the failing check `if len(autoLanguages) != 1 { return "", nil }` in `detectLanguage()`:
   * When multiple languages are detected, inspect the immediate target path (`project.File` / `project.RootDir`):
     * If the target directory contains `.go` files or sits within a resolved Go module, and the operation targets Go sources, return `LanguageGo`.
     * For workspace-wide discovery (`OperationWorkspaceOutline`), return `LanguageAuto` directly to allow the polyglot aggregator in Track 2 to run all matching backends.
3. **Explicit Language Parameter Exposure**:
   Ensure all read wire operations permit explicit language specification (`"go"`, `"java"`, `"rust"`), ensuring user/agent overrides bypass auto-detection heuristics entirely.

---

### 3.2 Track 2: Workspace Outline Polyglot Aggregator & Wire Registration

**Files to modify:**

* `internal/operation/wire_workspace_outline.go`
* `internal/backend/read.go`

#### Concrete Implementation Steps

1. **Register Go in Workspace Outline Dispatchers**:
   Update `workspaceOutlineDef()` in [`internal/operation/wire_workspace_outline.go`](../../internal/operation/wire_workspace_outline.go):

   ```go
   Handlers: map[backend.LanguageID]func(context.Context, CallContext, WorkspaceOutlineReq) (WorkspaceOutlineRes, error){
       backend.LanguageAuto: runWorkspaceOutline,
       backend.LanguageGo:   runWorkspaceOutline,
       backend.LanguageJava: runWorkspaceOutline,
   },
   ```

   This ensures the MCP tool schema publishes `language: ["auto", "go", "java"]`.
2. **Polyglot Aggregator Implementation in `runWorkspaceOutline`**:
   Replace the mutually exclusive if/else structure in `runWorkspaceOutline`:

   ```go
   func runWorkspaceOutline(ctx context.Context, cc CallContext, req WorkspaceOutlineReq) (WorkspaceOutlineRes, error) {
       project := effectiveProject(cc, req.Project)
       targetLanguage := project.Language

       // Explicit single-language request
       if targetLanguage == backend.LanguageJava {
           result, err := javabackend.WorkspaceOutline(project.RootDir, req.Path, req.IncludeTests)
           if err != nil {
               return WorkspaceOutlineRes{}, err
           }
           return WorkspaceOutlineRes{Result: result}, nil
       }
       if targetLanguage == backend.LanguageGo {
           result, err := astedit.WorkspaceOutlineGo(cc.WorkDir, req.Path, req.IncludeTests)
           if err != nil {
               return WorkspaceOutlineRes{}, err
           }
           return WorkspaceOutlineRes{Result: result}, nil
       }

       // Auto mode: Polyglot aggregation
       combined := &backend.WorkspaceOutlineResult{
           Scope: backend.ReadScope{
               Kind:     "workspace",
               Path:     req.Path,
               Complete: true,
           },
           Modules:  make([]backend.WorkspaceModule, 0),
           Packages: make([]backend.WorkspacePackage, 0),
       }

       // 1. Discover Go
       if isGo, _ := astedit.LooksLikeGoWorkspace(cc.WorkDir, req.Path); isGo {
           if goRes, err := astedit.WorkspaceOutlineGo(cc.WorkDir, req.Path, req.IncludeTests); err == nil && goRes != nil {
               combined.Modules = append(combined.Modules, goRes.Modules...)
               combined.Packages = append(combined.Packages, goRes.Packages...)
           }
       }

       // 2. Discover Java
       if isJava, _ := javabackend.LooksLikeJavaWorkspace(project.RootDir, req.Path); isJava {
           if javaRes, err := javabackend.WorkspaceOutline(project.RootDir, req.Path, req.IncludeTests); err == nil && javaRes != nil {
               combined.Modules = append(combined.Modules, javaRes.Modules...)
               combined.Packages = append(combined.Packages, javaRes.Packages...)
           }
       }

       // Format and deduplicate combined results
       sortWorkspaceResults(combined)
       return WorkspaceOutlineRes{Result: combined}, nil
   }
   ```

3. **Refine `LooksLikeJavaWorkspace`**:
   In [`internal/backend/java/workspace_outline.go`](../../internal/backend/java/workspace_outline.go), require a build manifest (`pom.xml`, `build.gradle`, `build.gradle.kts`) or restrict recursive `.java` walking so that isolated code samples in documentation directories do not trigger full Java reactor discovery unless explicitly scoped to them.

---

### 3.3 Track 3: Go Scoped Module Hierarchy & `go.work` Integration

**Files to modify:**

* `internal/astedit/workspace_outline.go`
* `internal/astedit/workspace_outline_test.go`

#### Concrete Implementation Steps

1. **Fix Sub-Module Scoping in `WorkspaceOutlineGo`**:
   In [`internal/astedit/workspace_outline.go`](../../internal/astedit/workspace_outline.go#L75-L84):
   * Currently, `moduleScanDir` defaults to `absRoot` whenever `targetDir` is inside `absRoot`, causing `discoverGoModules` to always report the root module.
   * When `path` is specified (`path != ""` and `path != "."`):
     * Resolve the module enclosing `targetDir` (walking upwards to find the closest `go.mod`).
     * Also discover any nested modules located beneath `targetDir`.
     * Do **not** inject the root module `.` as the primary module of a sub-path unless `targetDir` is truly the root module.
2. **`go.work` Workspace Support**:
   * Inspect the workspace root for `go.work`.
   * When `go.work` is present, parse the `use (...)` directives using `golang.org/x/mod/modfile.ParseWork`.
   * Register every declared module directory in `WorkspaceOutlineResult.Modules` with `IsRoot: false` (marking only the workspace root or primary module as `IsRoot: true`).

---

### 3.4 Track 4: Module Descriptors & Package-to-Module Associations

**Files to modify:**

* `internal/backend/read.go`
* `internal/astedit/workspace_outline.go`
* `internal/backend/java/workspace_outline.go`
* `internal/operation/wire_workspace_outline.go`

#### Concrete Implementation Steps

1. **Enrich `WorkspaceModule` and `WorkspacePackage` Schemas**:
   In [`internal/backend/read.go`](../../internal/backend/read.go#L95-L115):

   ```go
   type WorkspacePackage struct {
       Path       string  `json:"path"`
       Name       string  `json:"name"`
       Synopsis   string  `json:"synopsis"`
       Doc        *string `json:"doc,omitempty"`
       Depth      int     `json:"depth"`
       FilesCount int     `json:"files_count"`
       Module     string  `json:"module,omitempty"` // Owning module path / coordinate
       Language   string  `json:"language,omitempty"`
   }

   type WorkspaceModule struct {
       Path          string  `json:"path"`
       ModulePath    string  `json:"module_path,omitempty"`
       GoVersion     string  `json:"go_version,omitempty"`
       GroupID       string  `json:"group_id,omitempty"`
       ArtifactID    string  `json:"artifact_id,omitempty"`
       Version       string  `json:"version,omitempty"`
       Packaging     string  `json:"packaging,omitempty"`
       IsRoot        bool    `json:"is_root"`
       Language      string  `json:"language,omitempty"`
       Synopsis      string  `json:"synopsis,omitempty"`
       Doc           *string `json:"doc,omitempty"`
       PackagesCount int     `json:"packages_count,omitempty"`
   }
   ```

2. **Extract Go Module Documentation & Bindings**:
   * In `parseGoMod`: parse comments preceding the `module <path>` declaration.
   * If `go.mod` lacks a doc comment, fallback to the root package's synopsis if `package main` or `package <mod>` exists in the module root directory.
   * During package walking, set `pkg.Module = owningModule.ModulePath` and `pkg.Language = "go"`.
3. **Format String Enhancement**:
   In `formatWorkspaceOutline()` in [`internal/operation/wire_workspace_outline.go`](../../internal/operation/wire_workspace_outline.go#L87-L124), display the module synopsis alongside module coordinates, and print language tags in polyglot mode.

---

### 3.5 Track 5: MCP Critical Text Projection Pruning & Output Deduplication

**Files to modify:**

* `internal/mcp/presentation/presentation.go`
* `internal/mcp/server.go`
* `internal/mcp/presentation_test.go`

#### Concrete Implementation Steps

1. **Prune Bulk Arrays in `sanitizePresentationMap`**:
   In [`internal/mcp/presentation/presentation.go`](../../internal/mcp/presentation/presentation.go#L139-L160):
   * Currently, `sanitizePresentationMap` deletes `diff`, `inspection`, `source`, `body`, and `read_set`, but preserves `files`, `packages`, and `modules`.
   * Add pruning for bulk collections:

     ```go
     delete(result, "files")
     delete(result, "packages")
     delete(result, "modules")
     delete(result, "symbols")
     ```

   * Ensure `Critical result (JSON):` contains only high-level summary metadata (`status`, `scope`, `workspace_root`, `diagnostic_status`, `diagnostic_delta`).
2. **Deduplicate `server.go` MCP Envelope**:
   In [`internal/mcp/server.go`](../../internal/mcp/server.go#L2224-L2228):
   * When `structuredContent` is provided in the JSON-RPC response, the text body in `content[0].text` must not duplicate the raw structured JSON. For outline tools, format a concise summary for `content[0].text` while preserving the full data tree inside `structuredContent`.

---

### 3.6 Track 6: Tool Schema Description Matrix Condensation & Token Optimisation

**Files to modify:**

* `internal/capability/contract_model.go`
* `internal/mcp/server.go`
* `internal/mcp/schema_test.go`

#### Concrete Implementation Steps

1. **Condense Fault Tolerance Descriptions**:
   In [`internal/capability/contract_model.go`](../../internal/capability/contract_model.go#L501-L523):
   * Remove raw 64-bit hexadecimal masks (`masks: supported=0x5555555555555555...`) and 64-case permutation breakdowns from public tool descriptions.
   * Replace with concise, natural operational statements:

     ```go
     return fmt.Sprintf("Requires valid %s syntax; external type errors do not prevent execution", b.Summary())
     ```

   * Expose exact bitmasks and formal case matrices via dedicated MCP resource endpoints (`semedit://contracts/...`) or reference documents rather than embedding them in every tool description.
2. **Condense Continuation Routing Graphs**:
   In `internal/mcp/server.go`:
   * Replace the dynamic 30-tool enumeration (`file copied from the same result record to semantic_assertion_mode, ...`) with:

     ```text
     Copy forward resolved 'file' and selector fields from structured result records to subsequent semantic tool calls.
     ```

---

### 3.7 Track 7: CLI Txtar Contracts, Docgen, & Skill Synchronisation

**Files to modify:**

* `testdata/scripts/workspace_outline_polyglot.txtar`
* `testdata/scripts/workspace_outline_scoping.txtar`
* `cmd/docgen/txtar.go`
* `skills/semedit/SKILL.md`
* `skills/semedit/references/go.md`

#### Concrete Implementation Steps

1. **New Comprehensive `.txtar` Contract Tests**:
   * `workspace_outline_polyglot.txtar`: Repository containing root `go.mod`, Go packages, nested `pom.xml`, and Java sources. Asserts that `semedit workspace-outline` outputs both Go and Java modules and packages in `auto` mode, and filters strictly when `--language go` or `--language java` is passed.
   * `workspace_outline_scoping.txtar`: Multi-module Go repository (`.` and `submod/`). Asserts that `--path submod` outputs `submod` as the module, attributes packages to `submod`, and does not misrepresent the root module as the target.
2. **Docgen & Skill Synchronisation**:
   * Update `skills/semedit/SKILL.md` to document polyglot aggregation in `semantic_workspace_outline`.
   * Update `skills/semedit/references/go.md` with multi-module `go.work` and scoped subdirectory navigation patterns.

---

## 4. Multi-Language Test Fixtures & Level-1/2/3 Testing Strategy

Every change must be validated across three distinct testing tiers:

### 4.1 Level 1: Hermetic Unit Tests

* `internal/backend/backend_test.go`: Test parent manifest ascent when detecting language on subdirectories.
* `internal/astedit/workspace_outline_test.go`: Test `WorkspaceOutlineGo` on scoped subdirectories, multi-module fixtures, and `go.work` manifests.
* `internal/mcp/presentation/presentation_test.go`: Assert that `CriticalText()` strips bulk `files`, `packages`, and `modules` arrays from the JSON prefix.

### 4.2 Level 2: CLI `.txtar` Executable Contracts

* Execute `go test -run 'TestREQ001SemanticIntentCLIContracts/workspace_outline' ./...` ensuring all CLI txtar tests run hermetically and verify observable output boundaries. This is the repository's actual script test entry point.

### 4.3 Level 3: Polyglot Integration Scenario

* Create a test fixture under `.scratch/` mimicking SimKit:
  * Root `go.mod`
  * Subdirectories `proprietary/` and `open-source/` containing Go packages
  * Auxiliary `docs/examples/` containing `.java` source files
* Assert:
  1. `semedit workspace-outline` discovers the Go module and all Go packages.
  2. `semedit workspace-outline --path open-source` resolves Go automatically without language detection errors.
  3. `semedit workspace-outline --language go` executes cleanly.

---

## 5. Acceptance Verification Checklist

Before declaring implementation complete:

* [x] **No Linters Changed**: Confirm zero modifications to `.golangci.yml`, Makefile lint flags, or lint configurations.
* [x] **`make check` Passes**: Run `make check` and confirm zero failures across linters and core tests.
* [x] **Polyglot Aggregation**: Confirm `workspace-outline` reports both Go and Java components in mixed repositories.
* [x] **Sub-Tree Language Detection**: Confirm `path: "subdir"` resolves language cleanly via parent manifest ascent.
* [x] **Module Scoping Accuracy**: Confirm scoped outline requests report the targeted module rather than the root workspace module.
* [x] **Module Metadata Present**: Confirm `WorkspaceModule` includes `synopsis` and package counts.
* [x] **Zero MCP Triplication**: Verify that MCP outline tool output does not duplicate bulk file records in text and structured content.
* [x] **Description Token Budget**: Verify that hexadecimal bitmasks and 30-item routing graphs are removed from tool descriptions.
* [x] **Skill Documentation Synchronised**: Confirm `SKILL.md` and reference guides reflect updated schemas and capabilities.

## 6. Implementation Results (2026-10-14)

All seven tracks are complete. Auto discovery combines Go and Java, explicit language selection filters results, and ancestor manifest detection respects repository and ignored-cache boundaries. Go scopes report their nearest module and descendants; `go.work` use entries report canonical module descriptors, including external modules, without walking external packages. Java scopes discover declared and independent Maven modules, retain precise ownership, and avoid parsing unrelated siblings.

Both producers emit language ownership, module documentation, and scoped package counts, including zero. The CLI displays module synopses and mixed-language tags. MCP outline text contains a summary and critical metadata while structured content retains complete records once. Compact descriptions preserve conditional and mixed-fault guidance; generated capability reference documentation retains the exact masks and case counts.

Validation: `make check` and `go test ./...` pass in the primary checkout. The complete suite uses project-local tools on `PATH` and loopback permission for its local test servers. CLI contracts cover mixed discovery, filters, scoped ownership, local/external workspace modules, malformed manifests, independent Maven modules, and malformed out-of-scope siblings. The SimKit-style smoke fixture confirms root discovery, `--path open-source`, and `--language go` despite Java documentation samples. No lint configuration or repository workspace manifest changed.

The existing Go reference is `skills/semedit/references/golang.md`; it was updated instead of adding a duplicate `go.md`. ADR-0066 and its index now record the discovery and projection boundaries. Docgen writes the updated capability reference before the full site command rejects an existing version-1 benchmark report with a missing duration unit; the unrelated benchmark artifacts remain untouched.
