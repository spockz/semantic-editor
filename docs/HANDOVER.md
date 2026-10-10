<!-- WHY: Comprehensive implementation handover detailing architecture, algorithms, and acceptance criteria for systemic friction resolution across ingress, workspace governance, manifest safety, batch verification, construct mutators, and schema feedback. -->
# Handover: Systemic Friction Resolution and Architecture Hardening

* **Date**: 2026-10-09
* **Governance**: Implementation handover following ADR-0005, ADR-0032, ADR-0042, ADR-0055, ADR-0057, and ADR-0065.
* **Standard**: British English spelling, Oxford commas, assertive phrasing, and executable acceptance criteria.

This specification provides exhaustive engineering instructions, algorithmic steps, edge cases, and test requirements for resolving systemic friction logged in [`docs/SUBOPTIMAL_TOOLS.md`](file:///Users/alessandro/sources/semantic-editor/docs/SUBOPTIMAL_TOOLS.md) and [`docs/MANUAL_EDITS.md`](file:///Users/alessandro/sources/semantic-editor/docs/MANUAL_EDITS.md).

---

## Track 1: Ingress Leniency and Robustness Principle (ADR-0065)

Implements Postel's robustness principle (*lenient in, precise out*) across MCP ingress handlers while maintaining strict AST outputs.

### 1.1 Package Directory Resolution in Symbol Lookup

* **Target Files**:
  * [`internal/symbol/resolver.go`](file:///Users/alessandro/sources/semantic-editor/internal/symbol/resolver.go) (`resolveTargetFiles`)
  * [`internal/operation/wire_read_inspect.go`](file:///Users/alessandro/sources/semantic-editor/internal/operation/wire_read_inspect.go)
* **Root Cause**:
  In `resolveTargetFiles(rootDir, filePath string)`, passing a directory path in `filePath` resolves to `[]string{resolvedPath}` without checking `os.Stat`. Subsequent readers call `os.ReadFile(resolvedPath)`, which aborts with `is a directory`.
* **Algorithmic Steps**:
  1. In `resolveTargetFiles`, inspect `fi, err := os.Stat(resolvedPath)`.
  2. If `err == nil && fi.IsDir()`, read all entries via `os.ReadDir(resolvedPath)`.
  3. Filter entries matching `strings.HasSuffix(entry.Name(), ".go")` and not `strings.HasSuffix(entry.Name(), "_test.go")` (or include test files if configured).
  4. Return the sorted slice of absolute file paths.
  5. If multiple files declare an identical unexported identifier, return all matching candidates with file paths and line numbers rather than picking arbitrarily.
* **Edge Cases**:
  * Empty directories: Return a clean error indicating no Go files exist in the specified package.
  * Mixed build tags: Respect package-level build tags if applicable.
* **Acceptance Criteria**:
  * **Scenario 1 (Directory lookup)**:
    * **Given** a package directory `internal/telemetry/` containing `telemetry.go` and `span.go`.
    * **When** invoking `semantic_lookup` with `file: "internal/telemetry"` and `symbol: "Start"`.
    * **Then** the engine resolves `Start` in `internal/telemetry/telemetry.go` with exit status 0 and returns its declaration coordinates.
  * **Scenario 2 (Ambiguous symbol across files)**:
    * **Given** a package with `a.go` declaring `type helper struct{}` and `b.go` declaring `type helper struct{}`.
    * **When** invoking `semantic_lookup` with `file: "pkg"` and `symbol: "helper"`.
    * **Then** the engine fails closed with an error enumerating both candidate coordinates `a.go:10:1` and `b.go:15:1`.
* **Test Plan**:
  * Unit test in `internal/symbol/resolver_test.go`.
  * CLI txtar scenario in `testdata/scripts/symbol_lookup_directory.txtar`.

---

### 1.2 Outer Block Delimiter Stripping in Body Replacement

* **Target Files**:
  * [`internal/astedit/body.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/body.go) (`validateBodySnippet`)
* **Root Cause**:
  `validateBodySnippet` wraps input in `synthetic := "package dummy\n\nfunc _() {\n" + bodyTrimmed + "\n}\n"`. When callers supply a body enclosed in outer braces `{ ... }`, parsing either nests the block redundantly or causes syntax errors if trailing tokens are present.
* **Algorithmic Steps**:
  1. Inspect `bodyTrimmed := strings.TrimSpace(bodySource)`.
  2. If `strings.HasPrefix(bodyTrimmed, "{") && strings.HasSuffix(bodyTrimmed, "}")`:
     * Scan brace balancing from index 0. If the initial opening `{` pairs with the final `}` at `len(bodyTrimmed)-1` without closing at intermediate depth 0, strip the delimiters: `bodyTrimmed = strings.TrimSpace(bodyTrimmed[1 : len(bodyTrimmed)-1])`.
  3. Validate the unwrapped snippet against `validateBodySnippet`.
* **Edge Cases**:
  * Disjoint sequential blocks (e.g. `{ a := 1 }\n{ b := 2 }`): Must NOT strip outer braces because index 0 does not pair with the final character.
* **Acceptance Criteria**:
  * **Scenario 1 (Enclosed body stripped)**:
    * **Given** a target function `func Run() error { return nil }`.
    * **When** invoking `semantic_replace_body` with `body: "{\n\treturn errors.New(\"failed\")\n}"`.
    * **Then** the function body is replaced with `return errors.New("failed")` without redundant nested braces.
  * **Scenario 2 (Bare statements accepted)**:
    * **Given** a target function `func Run() error`.
    * **When** invoking `semantic_replace_body` with `body: "return nil"`.
    * **Then** the function body is formatted cleanly inside the existing function braces.
* **Test Plan**:
  * Unit test in `internal/astedit/body_test.go`.
  * CLI txtar test in `testdata/scripts/replace_body_braces.txtar`.

---

### 1.3 Plain-Text Purpose Comment Wrapping

* **Target Files**:
  * [`internal/astedit/scaffold.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/scaffold.go) (`scaffoldSourceWithPurpose`, `formatScaffoldSource`)
* **Root Cause**:
  `scaffoldSourceWithPurpose` parses `purposeHeader + "\n\npackage " + packageName`. If `purposeHeader` does not start with `//` or `/*`, parsing aborts with `ErrSyntax: expected 'package', found ...`.
* **Algorithmic Steps**:
  1. Inspect `purposeHeader := strings.TrimSpace(opts.PurposeHeader)`.
  2. If `purposeHeader != ""`:
     * Split into lines: `lines := strings.Split(purposeHeader, "\n")`.
     * If any non-empty line lacks a leading `//` or `/*`, convert each line into a Go comment by prepending "// " (preserving existing comment markers if already present).
     * Join lines and append a trailing newline.
  3. Format the file and ensure `package` declaration immediately follows the purpose comment block.
* **Edge Cases**:
  * Empty purpose: Produce standard package declaration with zero header comments.
  * Existing doc comments: Leave comments starting with `//` or `/*` intact.
* **Acceptance Criteria**:
  * **Scenario 1 (Raw prose string wrapped)**:
    * **Given** a scaffolding request for `config.go` in package `config`.
    * **When** supplying `purpose: "Provides application configuration primitives and parsing."`.
    * **Then** `config.go` starts with `// Provides application configuration primitives and parsing.\npackage config`.
  * **Scenario 2 (Pre-formatted comment preserved)**:
    * **Given** `purpose: "// WHY: Isolates credential loading.\n// Package auth owns token verification."`.
    * **When** invoking `semantic_scaffold_file`.
    * **Then** the lines are preserved verbatim above `package auth`.
* **Test Plan**:
  * Unit test in `internal/astedit/scaffold_test.go`.
  * CLI txtar test in `testdata/scripts/scaffold_purpose_wrapping.txtar`.

---

### 1.4 Construct Kind Synonym Aliasing and Receiver Inference

* **Target Files**:
  * [`internal/operation/wire_engine_mutations.go`](file:///Users/alessandro/sources/semantic-editor/internal/operation/wire_engine_mutations.go)
  * [`internal/operation/wire_engine_constructs.go`](file:///Users/alessandro/sources/semantic-editor/internal/operation/wire_engine_constructs.go)
  * [`internal/astedit/construct_declaration.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/construct_declaration.go)
* **Root Cause**:
  `req.Kind = backend.ConstructKind(kindStr)` performs literal string conversion. Natural language synonyms (`variable`, `constant`, `procedure`) produce an unknown kind error. Method symbols containing receivers fail when callers pass `kind: "function"`.
* **Algorithmic Steps**:
  1. Create a helper `canonicalizeConstructKind(kind string, symbol string) backend.ConstructKind`.
  2. Map incoming kind strings:
     * `"variable"` $\to$ `backend.ConstructVar`
     * `"constant"` $\to$ `backend.ConstructConst`
     * `"procedure"` $\to$ `backend.ConstructFunction`
  3. If `kind == backend.ConstructFunction` and `symbol` matches method receiver patterns (`strings.Contains(symbol, ".")` or `strings.HasPrefix(symbol, "(")`), promote kind to `backend.ConstructMethod`.
* **Edge Cases**:
  * Unknown kinds: Return an informative error listing all supported kinds.
* **Acceptance Criteria**:
  * **Scenario 1 (Synonym aliasing)**:
    * **When** invoking `semantic_insert_construct` with `kind: "variable"`.
    * **Then** the engine executes the insertion using canonical kind `var`.
  * **Scenario 2 (Method inference)**:
    * **When** replacing construct on `symbol: "(*Server).Start"` with `kind: "function"`.
    * **Then** the engine infers `kind: "method"` and completes replacement without error.
* **Test Plan**:
  * Unit test in `internal/operation/wire_engine_mutations_test.go`.
  * CLI txtar test in `testdata/scripts/construct_synonyms.txtar`.

---

## Track 2: Workspace and Worktree Governance

Guarantees subagents and parallel git worktrees operate within isolated boundaries without colliding with the primary workspace.

### 2.1 Optional Dynamic `workspace_root` Ingress Parameter

* **Target Files**:
  * [`internal/mcp/schema.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/schema.go)
  * [`internal/operation/wire_backend.go`](file:///Users/alessandro/sources/semantic-editor/internal/operation/wire_backend.go)
  * [`internal/mcp/workspace/workspace_root.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/workspace/workspace_root.go)
* **Root Cause**:
  Subagent sessions operating in worktrees under `.scratch/worktrees/` require routing tool calls to their private directory while the MCP daemon was initialized at the repository root.
* **Algorithmic Steps**:
  1. Add `workspace_root` as an optional parameter to `wireRequestParams`.
  2. If present in tool invocation:
     * Canonicalise relative to the daemon root: `effectiveRoot := filepath.Clean(filepath.Join(daemonRoot, req.WorkspaceRoot))`.
     * Check that `effectiveRoot` resides within allowed workspace bounds.
     * Route all file path calculations and compiler diagnostic sessions through `effectiveRoot`.
  3. If omitted: Default to the session's configured working directory.
* **Edge Cases**:
  * Non-existent directory: Reject immediately with descriptive error before parsing files.
* **Acceptance Criteria**:
  * **Scenario 1 (Worktree targeting)**:
    * **Given** a worktree at `.scratch/worktrees/feature-a`.
    * **When** calling `semantic_lookup` with `workspace_root: ".scratch/worktrees/feature-a"` and `file: "pkg/service.go"`.
    * **Then** the engine resolves symbols using the worktree files.
* **Test Plan**:
  * Unit test in `internal/mcp/workspace/workspace_root_test.go`.

---

### 2.2 Strict Workspace Boundary Enforcement

* **Target Files**:
  * [`internal/mcp/workspace/workspace_root.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/workspace/workspace_root.go)
* **Root Cause**:
  Relative traversal paths (`../../`) or symlinks pointing outside the workspace boundary risk unintentional mutations.
* **Algorithmic Steps**:
  1. In `ValidateWorkspaceBoundary(root, target string) error`:
     * Resolve symlinks using `filepath.EvalSymlinks`.
     * Verify that `rel, err := filepath.Rel(canonicalRoot, canonicalTarget)` satisfies `!strings.HasPrefix(rel, "..") && rel != ".."`.
  2. Reject any violation with `ErrWorkspaceBoundaryExceeded`.
* **Acceptance Criteria**:
  * **Scenario 1 (Path escape blocked)**:
    * **When** a request targets `../../etc/passwd` or `/tmp/other-project/file.go`.
    * **Then** the request is rejected with `target path lies outside active workspace root`.
* **Test Plan**:
  * Unit test in `internal/mcp/workspace/workspace_root_test.go`.

---

## Track 3: Workspace Manifest Safety and Toolchain Isolation

Protects `go.mod` and `go.sum` from unintended modifications during routine code edits.

### 3.1 Readonly Module Flag (`GOFLAGS="-mod=readonly"`)

* **Target Files**:
  * [`internal/gocache/cache.go`](file:///Users/alessandro/sources/semantic-editor/internal/gocache/cache.go#L122)
  * [`internal/pipeline/pipeline.go`](file:///Users/alessandro/sources/semantic-editor/internal/pipeline/pipeline.go#L546)
* **Root Cause**:
  `gocache.Environment` explicitly clears `GOFLAGS` (`"GOFLAGS="`). When `pipeline.organizeImportFile` calls `imports.Process`, `go list` runs under modern Go toolchains, automatically upgrading `go.mod` directives and reordering indirect dependencies on older checkouts.
* **Algorithmic Steps**:
  1. In `gocache.Environment`, change `"GOFLAGS="` to `"GOFLAGS=-mod=readonly"`.
  2. In `pipeline.organizeImportFile`, ensure any subprocess spawned by `imports.Process` retains `-mod=readonly`.
  3. Ensure `go.mod` is marked strictly immutable for read-only and code-formatting pipelines.
* **Edge Cases**:
  * Tool dependencies intentionally added: `semantic_add_build_dependency` must explicitly omit `-mod=readonly` to permit required additions.
* **Acceptance Criteria**:
  * **Scenario 1 (`go.mod` preservation during import organization)**:
    * **Given** an older Go module with directive `go 1.18`.
    * **When** executing `semantic_replace_body` with `auto_organize_imports: true`.
    * **Then** the Go code is formatted and imports organized without modifying `go.mod` or bumping the Go directive.
* **Test Plan**:
  * CLI txtar test in `testdata/scripts/gomod_readonly_preservation.txtar`.

---

### 3.2 Respect Inherited `GOMODCACHE` in Offline Sessions

* **Target Files**:
  * [`internal/gocache/cache.go`](file:///Users/alessandro/sources/semantic-editor/internal/gocache/cache.go)
  * [`internal/mcp/server.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/server.go)
* **Root Cause**:
  `gocache.Environment` unconditionally points `GOMODCACHE` to a scratch directory under `.scratch/`. In offline environments with `GOPROXY=off`, external module downloads fail because pre-cached dependencies in the inherited host `GOMODCACHE` are inaccessible.
* **Algorithmic Steps**:
  1. Check if `os.Getenv("GOPROXY") == "off"` or offline session flag is enabled.
  2. If the parent environment exports a valid `GOMODCACHE` directory, inherit that path instead of overriding it with an empty scratch directory.
* **Acceptance Criteria**:
  * **Scenario 1 (Offline session using host module cache)**:
    * **Given** `GOPROXY=off` and host `GOMODCACHE=/var/cache/go/mod`.
    * **When** running type checking or gopls verification.
    * **Then** the engine reads packages from `/var/cache/go/mod` without failing with network errors.
* **Test Plan**:
  * Unit test in `internal/gocache/cache_test.go`.

---

### 3.3 Non-Go Directory Filtering in Gopls Root Discovery

* **Target Files**:
  * [`internal/godiagnostics/diagnostics.go`](file:///Users/alessandro/sources/semantic-editor/internal/godiagnostics/diagnostics.go#L329)
* **Root Cause**:
  `selectPackageFiles` checks `if len(fileSet) == 0 { return nil, nil, fmt.Errorf("no Go source files found in project root %q", root) }`. In projects where the root directory contains non-Go folders (e.g. `docs/`, `scripts/`) or where Go files reside solely in nested packages, gopls diagnostic queries abort.
* **Algorithmic Steps**:
  1. In `selectPackageFiles`, do not abort if immediate root has zero Go files as long as subpackages contain valid Go files.
  2. Filter out non-Go directories (`docs`, `scripts`, `node_modules`, `dist`) before asserting package coverage.
* **Acceptance Criteria**:
  * **Scenario 1 (Repository with Go files solely in subpackages)**:
    * **Given** a repository with root `main` package absent, but `cmd/app/main.go` and `internal/core/core.go` present.
    * **When** running `CheckDiagnostics` on the repository root.
    * **Then** the engine successfully collects diagnostics from `cmd/app` and `internal/core` without aborting.
* **Test Plan**:
  * Unit test in `internal/godiagnostics/diagnostics_test.go`.

---

### 3.4 Preserve Explicit Nested-Module Imports

* **Target Files**:
  * [`internal/pipeline/pipeline.go`](file:///Users/alessandro/sources/semantic-editor/internal/pipeline/pipeline.go#L546)
* **Root Cause**:
  `imports.Process` purges imports that cannot be resolved from the top-level module manifest. Explicit imports requested via `opts.AddImports` targeting nested modules are erroneously stripped.
* **Algorithmic Steps**:
  1. Record explicit import paths requested in `opts.AddImports`.
  2. After `imports.Process` finishes, check if any requested explicit imports were stripped.
  3. Re-inject missing explicit imports and format the import declaration block.
* **Acceptance Criteria**:
  * **Scenario 1 (Nested module import retained)**:
    * **Given** an explicit addition `opts.AddImports = []string{"semedit.tools/package-closure"}`.
    * **When** `organizeImportFile` executes.
    * **Then** the import `semedit.tools/package-closure` remains present in the target source file.
* **Test Plan**:
  * Unit test in `internal/pipeline/pipeline_test.go`.

---

## Track 4: Batch Execution and Verification Hardening

Improves clarity and feedback when executing multi-step batch transformations.

### 4.1 Uniform `trust_workspace` and `skip_hooks` Parameters

* **Target Files**:
  * [`internal/mcp/schema.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/schema.go)
  * [`internal/mcp/batch.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/batch.go)
  * [`internal/mcp/server.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/server.go)
  * [`internal/operation/wire_backend.go`](file:///Users/alessandro/sources/semantic-editor/internal/operation/wire_backend.go)
* **Root Cause**:
  `trust_workspace` and hook skip capabilities are inconsistently supported across endpoints.
* **Algorithmic Steps**:
  1. Add `skip_hooks` (boolean, default false) and `trust_workspace` (boolean, default false) to all verification endpoints (`semantic_batch`, `semantic_verify`, and server initialization options).
  2. When `skip_hooks: true`, bypass execution of external project verification hooks (linters, test commands) while preserving deterministic AST formatting and gopls diagnostics.
* **Acceptance Criteria**:
  * **Scenario 1 (Hooks skipped during batch)**:
    * **Given** a batch request with `skip_hooks: true`.
    * **When** mutations are applied.
    * **Then** formatting and AST diagnostics run, but external Makefile or shell hooks are skipped.
* **Test Plan**:
  * Unit test in `internal/mcp/batch_test.go`.

---

### 4.2 Scoped Post-Batch Verification

* **Target Files**:
  * [`internal/mcp/batch.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/batch.go)
  * [`internal/projectverify/`](file:///Users/alessandro/sources/semantic-editor/internal/projectverify/)
* **Root Cause**:
  Post-batch verification verifies the entire workspace, including packages untouched by the batch, degrading execution performance.
* **Algorithmic Steps**:
  1. Track the set of files mutated during the batch: `mutatedPaths := []string{...}`.
  2. Derive the set of affected packages and their direct dependents.
  3. Scope verification hooks and diagnostics specifically to affected packages.
* **Acceptance Criteria**:
  * **Scenario 1 (Scoped linting)**:
    * **Given** a change affecting only `pkg/logger/logger.go`.
    * **When** post-batch verification runs.
    * **Then** verification executes for `pkg/logger` and direct callers, leaving unrelated packages untouched.
* **Test Plan**:
  * Unit test in `internal/mcp/batch_verification_test.go`.

---

### 4.3 Decouple Mutation Status from Hook Exit Codes

* **Target Files**:
  * [`internal/mcp/batch.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/batch.go#L816-L821)
* **Root Cause**:
  If mutations succeed on disk but verification hooks return a non-zero exit code, `response.Status` is set to `"error"`, masking successful disk writes.
* **Algorithmic Steps**:
  1. Introduce distinct statuses:
     * `status: "applied"`: All mutations applied, verification passed.
     * `status: "applied_with_diagnostics"`: All mutations applied to disk, but post-verification hooks reported warnings or errors.
     * `status: "error"`: An AST mutation failed; disk changes rolled back.
  2. When verification hooks fail after mutations succeed, set `response.Status = "applied_with_diagnostics"`.
* **Acceptance Criteria**:
  * **Scenario 1 (Mutations applied despite linter warning)**:
    * **Given** successful AST mutations on disk followed by a linter exit code 1.
    * **When** batch response is generated.
    * **Then** `response.Status` is `"applied_with_diagnostics"`, and the final file diff is included.
* **Test Plan**:
  * Unit test in `internal/mcp/batch_test.go`.

---

### 4.4 Multi-Turn Interactive Staging (`semantic_stage` to `semantic_commit`)

* **Target Files**:
  * Implementation of ADR-0032 in `internal/mcp/` and `internal/operation/`.
* **Root Cause**:
  Sequential single-tool calls currently emit intermediate compiler diagnostics per turn.
* **Algorithmic Steps**:
  1. Implement `semantic_stage`: applies AST modifications to an in-memory buffer without disk mutation or diagnostic emissions.
  2. Implement `semantic_commit`: flushes staged buffers to disk atomically, runs `gofmt` and import organization, and performs a single end-of-batch verification pass.
* **Acceptance Criteria**:
  * **Scenario 1 (Two-stage interdependent insertion)**:
    * **Given** function `A()` calling not-yet-created function `B()`.
    * **When** staging `A()` via `semantic_stage` followed by staging `B()`.
    * **Then** no intermediate undefined-symbol diagnostics are returned until `semantic_commit` executes.
* **Test Plan**:
  * CLI txtar test in `testdata/scripts/staging_lifecycle.txtar`.

---

## Track 5: AST Construct Mutator Hardening

Extends structural mutators to handle common Go declaration patterns.

### 5.1 Group-Splitting Mutators for `var (...)` and `const (...)` Blocks

* **Target Files**:
  * [`internal/astedit/move_construct.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/move_construct.go#L98, #L405)
  * [`internal/astedit/construct.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/construct.go)
* **Root Cause**:
  `move_construct.go` byte-slices whole `ast.Decl` spans and guards with `if len(group.Specs) > 1 { return fmt.Errorf("cannot move one member of a grouped declaration; split the group first") }`.
* **Algorithmic Steps**:
  1. When moving an identifier that resides inside an `*ast.GenDecl` with `len(group.Specs) > 1`:
     * Locate the matching `*ast.ValueSpec` or `*ast.TypeSpec`.
     * Extract the spec and remove it from `group.Specs`.
     * If `group.Tok == token.CONST`: Inspect if the spec or sibling specs utilise `iota` or implicit expressions. If `iota` is present, fail with `cannot move member of const group utilizing iota`.
     * If the spec has multiple names (`var a, b = 1, 2`), fail with `cannot move single identifier from multi-name spec; split variables first`.
  2. Re-format the source group in place with the surviving specs.
  3. At the destination, construct a standalone `*ast.GenDecl` containing the extracted spec and insert it at the target placement.
* **Edge Cases**:
  * Doc comments: Retain doc comments associated with the moved spec and do not strip sibling comments.
* **Acceptance Criteria**:
  * **Scenario 1 (Move member of grouped var)**:
    * **Given** `var (\n\ta = 1\n\tb = 2\n)` in `source.go`.
    * **When** moving `b` to `dest.go`.
    * **Then** `source.go` retains `var (\n\ta = 1\n)` and `dest.go` contains `var b = 2`.
* **Test Plan**:
  * Unit test in `internal/astedit/move_construct_test.go`.
  * CLI txtar test in `testdata/scripts/move_grouped_declaration.txtar`.

---

### 5.2 Permit Multiple Package Initialisers (`init`)

* **Target Files**:
  * [`internal/astedit/collision.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/collision.go#L77-L80, #L113-L125)
  * [`internal/astedit/function.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/function.go)
* **Root Cause**:
  `fileDeclarationCollisions` treats `init` as an ordinary unique identifier and enforces package uniqueness, rejecting insertions when a sibling file declares `func init()`.
* **Algorithmic Steps**:
  1. In `declarationNames` in `collision.go`:
     * Exempt `init`: `if name != "_" && name != "init" { wanted[name] = struct{}{} }`.
  2. In `fileDeclarationCollisions`:
     * Skip checking collisions when the function name is `"init"`.
* **Acceptance Criteria**:
  * **Scenario 1 (Insert second init function)**:
    * **Given** a package with `a.go` declaring `func init() {}`.
    * **When** inserting `func init() { setup() }` into `b.go`.
    * **Then** the insertion succeeds with exit status 0.
* **Test Plan**:
  * Unit test in `internal/astedit/collision_test.go`.
  * CLI txtar test in `testdata/scripts/insert_multiple_init.txtar`.

---

### 5.3 Narrow Safe-Delete Directive Refusals

* **Target Files**:
  * [`internal/astedit/safe_delete.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/safe_delete.go#L136-L140)
  * [`internal/astedit/remove.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/remove.go)
* **Root Cause**:
  `safeDeletePackageProblem` scans all packages in the module for `reflect`, `plugin`, or `unsafe` imports and directives. Unexported private symbols in unrelated packages cannot be targeted dynamically by other packages.
* **Algorithmic Steps**:
  1. In `verifySafeDeleteTarget`:
     * If the target symbol is unexported, restrict the `reflect`, `unsafe`, and directive checks exclusively to the target's declaring package.
     * Imports of `reflect` or `unsafe` in unrelated packages within the same module must not block safe deletion of private unexported functions.
* **Acceptance Criteria**:
  * **Scenario 1 (Private function deleted despite unsafe in unrelated package)**:
    * **Given** package `core` declaring unused unexported `func clean()`, while package `driver` imports `unsafe`.
    * **When** invoking `semantic_remove_construct` with `safe_delete: true` on `clean`.
    * **Then** `clean` is safely deleted without refusal.
* **Test Plan**:
  * Unit test in `internal/astedit/safe_delete_test.go`.
  * CLI txtar test in `testdata/scripts/safe_delete_narrowed_scope.txtar`.

---

### 5.4 Anchored Statement Insertion in Loops (`kind: "statement"`)

* **Target Files**:
  * [`internal/astedit/construct_span.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/construct_span.go#L23-L25)
  * [`internal/astedit/insert_statement_plan.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/insert_statement_plan.go#L180)
* **Root Cause**:
  `isDirectOwnerStatementCandidate` requires `!strings.Contains(candidate.path, ".")`, discarding any candidate with depth $> 0$ (such as statements inside `for` loops or `if` blocks).
* **Algorithmic Steps**:
  1. Permit nested candidates when matching statement anchors if `intent.Discriminator` or `intent.ConstructPath` identifies a statement inside a nested block.
  2. Compute insertion offset relative to the nested statement's line boundaries.
* **Acceptance Criteria**:
  * **Scenario 1 (Statement inserted inside loop)**:
    * **Given** a function containing `for i := 0; i < 10; i++ { process(i) }`.
    * **When** inserting statement `log.Println(i)` with anchor `process(i)` and `placement: "before"`.
    * **Then** the statement is inserted inside the loop immediately preceding `process(i)`.
* **Test Plan**:
  * Unit test in `internal/astedit/construct_insert_test.go`.
  * CLI txtar test in `testdata/scripts/insert_statement_in_loop.txtar`.

---

## Track 6: Schema Harmonisation and Diagnostic Feedback

Eliminates parameter divergences and provides actionable diagnostic remediation.

### 6.1 Harmonise `file` and `path` in `semantic_outline`

* **Target Files**:
  * [`internal/operation/wire_read_inspect.go`](file:///Users/alessandro/sources/semantic-editor/internal/operation/wire_read_inspect.go#L47-L52)
  * [`internal/mcp/schema.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/schema.go)
* **Root Cause**:
  `outlineReadParams` only accepts `path`. Callers supplying `file` are rejected by `CheckParams`.
* **Algorithmic Steps**:
  1. Add `file` as an alias parameter in `outlineReadParams` (or accept both `file` and `path`).
  2. In `parseOutlineRead`, read `path`; if empty, fall back to `file`.
* **Acceptance Criteria**:
  * **Scenario 1 (`file` argument accepted)**:
    * **When** invoking `semantic_outline` with `file: "main.go"`.
    * **Then** the outline is returned successfully without schema errors.
* **Test Plan**:
  * Unit test in `internal/operation/wire_read_inspect_test.go`.

---

### 6.2 Actionable Schema Error Feedback

* **Target Files**:
  * [`internal/mcp/server.go`](file:///Users/alessandro/sources/semantic-editor/internal/mcp/server.go)
  * [`internal/operation/wire_engine_mutations.go`](file:///Users/alessandro/sources/semantic-editor/internal/operation/wire_engine_mutations.go)
* **Root Cause**:
  Unknown construct kinds or invalid parameters return a generic error message without enumerating acceptable options.
* **Algorithmic Steps**:
  1. When validating enum parameters (e.g. `kind`, `placement`), format errors as:
     `unsupported construct kind "xyz"; valid kinds are: [function, method, type, const, var, if, loop, case, statement]`.
* **Acceptance Criteria**:
  * **Scenario 1 (Informative enum error)**:
    * **When** passing an invalid construct kind `kind: "macro"`.
    * **Then** the error message explicitly enumerates all valid construct kinds.
* **Test Plan**:
  * Unit test in `internal/mcp/server_test.go`.

---

### 6.3 Expose Disambiguation Paths for Repeated Selectors

* **Target Files**:
  * [`internal/astedit/construct.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/construct.go#L164)
  * [`internal/astedit/insert_statement_plan.go`](file:///Users/alessandro/sources/semantic-editor/internal/astedit/insert_statement_plan.go#L167)
* **Root Cause**:
  When a selector matches multiple identical statements (e.g. multiple `if err != nil`), the ambiguity error must provide exact paths for disambiguation.
* **Algorithmic Steps**:
  1. When candidate matches $> 1$, format each candidate with its exact index path (e.g. `construct_path: "2.0.1"` at line 42).
* **Acceptance Criteria**:
  * **Scenario 1 (Ambiguity diagnostics with paths)**:
    * **When** targeting `if err != nil` when multiple instances exist.
    * **Then** the error lists candidate `construct_path` identifiers for the caller to provide.
* **Test Plan**:
  * Unit test in `internal/astedit/construct_test.go`.

---

## Verification Protocol

1. Run `make lint-markdown` to confirm markdown formatting compliance.
2. Run `make check` before concluding each task.
3. Ensure all new txtar scripts in `testdata/scripts/` execute and pass.
