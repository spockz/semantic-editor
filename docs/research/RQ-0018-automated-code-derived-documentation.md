# RQ-0018: Automated Code-Derived Capability Documentation and Test-Driven Examples

* **Status**: Open
* **Date**: 2026-09-16
* **Category**: Documentation & Developer Experience

---

## 1. Context & Motivation

Developer tooling, compiler-backed editors, and AI agent extensions frequently suffer from documentation drift. As engineering teams introduce new operations, placement modes, access rules, or target languages, manual documentation updates lag behind. This divergence creates significant failure modes across three distinct user categories:

1. **Human Developers**: Encounter outdated CLI flags, inaccurate placement options, or undocumented syntax constraints.
2. **AI Agent Planners**: Incur avoidable tool execution errors and token waste by attempting operations that target backends reject (for example, supplying `protected` visibility to the Go backend, or omitting receiver parameters).
3. **Integration Harnesses**: Receive stale Model Context Protocol (MCP) tool schemas whose input parameter validation rules diverge from the host compiler's actual behavior.

In `semedit`, two high-fidelity sources of truth already exist in the codebase:

* **Compiler and AST Transformation Engines**: Go packages (`internal/astedit`, `internal/adapters/golang`, `internal/symbol`) implement exact validation constraints, placement algorithms, and visibility rules.
* **Executable Test Archives (`txtar`)**: The test suite under `testdata/scripts/*.txtar` captures realistic input files, refactoring commands, and expected resulting states verified through compiler execution.

Rather than authoring reference guides, agent skill files, and MCP schemas by hand, `semedit` requires an automated synthesis engine. This engine extracts declarative capability matrices from Go code, mines executable examples from passing `txtar` test scripts, and emits deterministic documentation artifacts. A continuous integration (CI) verification gate enforces zero drift between code capabilities and published documentation.

```text
┌─────────────────────────────────────────────────────────────┐
│                    SOURCES OF TRUTH                         │
│                                                             │
│   ┌──────────────────────────┐  ┌────────────────────────┐  │
│   │   LanguageBackend Code   │  │   testdata/*.txtar     │  │
│   │ (Declarative Matrix Go)  │  │  (Executable Tests)    │  │
│   └─────────────┬────────────┘  └───────────┬────────────┘  │
└─────────────────┼───────────────────────────┼───────────────┘
                  │                           │
                  ▼                           ▼
┌─────────────────────────────────────────────────────────────┐
│              AUTOMATED SYNTHESIS (cmd/docgen)               │
│                                                             │
│       AST Parser  ───►  Extractor  ───►  Renderer           │
└──────────────────────────────┬──────────────────────────────┘
                               │
       ┌───────────────────────┼───────────────────────┐
       ▼                       ▼                       ▼
┌──────────────┐      ┌─────────────────┐     ┌────────────────┐
│  Human Docs  │      │ Agent SKILL.md  │     │   MCP Schema   │
│ (Markdown)   │      │ (Few-Shot Prompts)│   │ (JSON Schemas) │
└──────────────┘      └─────────────────┘     └────────────────┘
```

---

## 2. Architecture: Declarative Capability Matrix in Code

### 2.1 Code-Embedded Source of Truth

Documentation must derive directly from executable Go structs and interfaces rather than external prose. Every language backend implements a structured capability interface that exposes its supported operations, access modifiers, placement options, and semantic constraints:

```go
package capability

// Operation identifies a discrete semantic editing action.
type Operation string

const (
    OpRename     Operation = "rename"
    OpInsertFunc Operation = "insert_func"
    OpInsertType Operation = "insert_type"
    OpInsertDecl Operation = "insert_decl"
    OpImports    Operation = "imports"
    OpGet        Operation = "get"
)

// LanguageMatrix declares capabilities and constraints for a specific language.
type LanguageMatrix struct {
    Language            string                     `json:"language"`
    DisplayName         string                     `json:"display_name"`
    Maturity            string                     `json:"maturity"` // "production", "preview", "planned"
    Operations          map[Operation]OpCapability `json:"operations"`
    AccessModifiers     []AccessModifier           `json:"access_modifiers"`
    PlacementQualifiers []Placement                `json:"placement_qualifiers"`
    Limitations         []ConstraintRule           `json:"limitations"`
}

// ParameterContract formalizes argument contracts across CLI and MCP interfaces.
type ParameterContract struct {
    Name        string   `json:"name"`
    CLIName     string   `json:"cli_name"`
    JSONName    string   `json:"json_name"`
    Type        string   `json:"type"` // "string", "boolean", "array"
    Description string   `json:"description"`
    Required    bool     `json:"required"`
    Default     string   `json:"default,omitempty"`
    Enums       []string `json:"enums,omitempty"`
}

// OpCapability describes backend support and parameter contracts for an operation.
type OpCapability struct {
    Supported           bool                `json:"supported"`
    Summary             string              `json:"summary"`
    Parameters          []ParameterContract `json:"parameters"`
    SupportsDryRun      bool                `json:"supports_dry_run"`
    SupportsDiagnostics bool                `json:"supports_diagnostics"`
}

// ConstraintRule formalizes a language-specific restriction.
type ConstraintRule struct {
    Code        string `json:"code"`
    Description string `json:"description"`
    Rationale   string `json:"rationale"`
}
```

### 2.2 Cross-Language Scope

The capability matrix formalizes support across current and future languages:

| Language | Status | Supported Operations | Access Modifiers | Primary Constraints |
| :--- | :---: | :--- | :--- | :--- |
| **Go** | Production | `rename`, `insert_func`, `insert_type`, `insert_decl`, `imports`, `get` | `infer`, `public`, `private` | Rejects `protected`/`package-private`; enforces public-precedes-private; clusters receiver methods. |
| **Java** | Planned | `rename`, `insert_func`, `insert_type`, `insert_decl`, `imports`, `get` | `infer`, `public`, `protected`, `package-private`, `private` | Requires single public class per file matching filename; enforces visibility section ordering. |
| **Python** | Planned | `rename`, `insert_func`, `insert_type`, `insert_decl`, `imports`, `get` | `infer`, `public`, `private` | Derives visibility from identifier prefix (`_`); rejects explicit keyword modifiers. |
| **TypeScript** | Planned | `rename`, `insert_func`, `insert_type`, `insert_decl`, `imports`, `get` | `infer`, `public`, `protected`, `private` | Manages export declarations; distinguishes type-only imports from value imports. |

### 2.3 Documented Language Limitations

Language backends declare concrete limitations as code objects. For the Go backend (`GolangBackend`), these include:

1. **Access Modifier Rejection**: Go lacks `protected` and `package-private` scopes. Supplying these modifiers returns `ErrUnsupportedModifier`.
2. **Identifier Casing Alignment**: Identifier casing dictates export status. Supplying `public` with a lowercase identifier or `private` with an uppercase identifier triggers `VisibilityMismatchError`.
3. **Section Ordering**: Public declarations precede private declarations within any generated or reorganized file.
4. **Receiver Clustering**: Methods bound to a specific receiver type cluster adjacent to existing methods for that receiver while still respecting public-precedes-private section boundaries.

### 2.4 Verifying the Source of Truth: Unit and Rapid Property Tests

To ensure that declared capabilities match actual engine behavior with zero divergence, the test harness employs two validation strategies:

1. **Rapid Property Testing (`pgregory.net/rapid`)**:
   * Generates arbitrary permutations of operations, access modifiers, placement options, and identifiers.
   * Asserts the fundamental invariant: For any operation $O$, language $L$, and parameter set $P$, if $L$ declares $(O, P)$ as supported, the engine accepts execution without returning `ErrUnsupportedOperation` or `ErrUnsupportedModifier`.
   * Asserts the converse invariant: If $L$ declares $(O, P)$ as unsupported, the engine rejects execution with an error wrapping `ErrUnsupportedOperation` or `ErrUnsupportedModifier`.
2. **Exhaustive Conformance Matrix Verification**:
   * A dedicated test iterates over every declared `LanguageMatrix` and validates all fields across runtime implementations:
     * Asserts that `ValidateModifier`, `ValidatePlacement`, and `SupportedAccessModifiers` match matrix definitions exactly.
     * Validates every `ParameterContract` against CLI Cobra flags (`pflag.Flag`) and MCP input schemas (`tools/list` JSON Schema properties).
     * Asserts that declared enum sets, default values, and required parameter flags are identical across CLI, MCP, and internal AST handlers.

---

## 3. Automated Extraction from `txtar` Test Archives & Workflows

### 3.1 Structure of `txtar` Test Archives

The repository maintains golden regression tests in `testdata/scripts/*.txtar`. These archives represent executable specifications containing:

* Human-readable intent and scenario descriptions in header comments.
* Script execution steps (`exec semedit insert ...`, `cmp ...`).
* Pre-transformation input files (`api/server.go`, `go.mod`).
* Post-transformation expected files (`want/api/server.go`).

Example archive snippet (`testdata/scripts/insert_declaration.txtar`):

```txtar
# Test declaration insertion with placement qualifiers and import organization
exec semedit insert --file api/server.go --placement public_start --source 'func InitServer() *Server { return &Server{} }'
exec semedit insert --file api/server.go --placement after_symbol --target Server.Start --source 'func (s *Server) Stop() {}'
cmp api/server.go want/api/server.go
exec go test ./...

-- go.mod --
module example.com/test
go 1.23

-- api/server.go --
package api

import (
    "fmt"
)

type Server struct{}

func (s *Server) Start() {
    fmt.Println("start")
}

func (s *Server) internalRun() {}

-- want/api/server.go --
package api

import (
    "fmt"
)

func InitServer() *Server { return &Server{} }

type Server struct{}

func (s *Server) Start() {
    fmt.Println("start")
}

func (s *Server) Stop() {}

func (s *Server) internalRun() {}
```

### 3.2 Automated Example Extraction

The extraction engine parses `txtar` archives into structured documentation examples:

* **Intent Extraction**: Reads header comments and optional doc annotations (for example, `# @doc:prompt Add a Stop method to the Server struct`).
* **Invocation Mapping**: Extracts CLI commands (`semedit insert --file ...`) and automatically derives the corresponding MCP tool call representation:

```json
{
  "name": "semantic_insert_function",
  "arguments": {
    "file": "api/server.go",
    "source": "func (s *Server) Stop() {}",
    "placement": "after_symbol",
    "target_symbol": "Server.Start"
  }
}
```

* **Diff Computation**: Computes unified diffs between the input file and the target `want/` file, generating concise visual diff snippets:

```diff
@@ -10,3 +10,5 @@
 func (s *Server) Start() {
     fmt.Println("start")
 }
+
+func (s *Server) Stop() {}
```

* **Multi-Step Refactoring Chains**: Extracts sequential commands within complex tests (such as `rename_diagnostic_delta.txtar`), illustrating multi-turn refactoring workflows.
* **Schema Validation & Execution Replay**:
  * Every synthesized MCP example payload is validated against the active MCP `tools/list` JSON schema to verify parameter compatibility.
  * In conformance testing, generated MCP tool calls are replayed via simulated `tools/call` requests in a hermetic test directory to guarantee they produce the exact expected diff without errors.

Because every example originates from an actively executing test in `testdata/scripts/`, documentation snippets never become obsolete or syntactically invalid.

---

## 4. Generation Pipeline & Target Artifacts

The synthesis tool (`cmd/docgen`) coordinates capability matrices, docstrings, and extracted test examples to generate four distinct target artifacts:

```text
┌─────────────────────────────────────────────────────────────┐
│                 cmd/docgen SYNTHESIS PIPELINE               │
├──────────────────────────────┬──────────────────────────────┤
│ 1. docs/reference/           │ Human Markdown Reference     │
│    capabilities.md           │ - Comprehensive matrices     │
│                              │ - Language limitation tables │
│                              │ - Executable diff examples   │
├──────────────────────────────┼──────────────────────────────┤
│ 2. skills/semedit/SKILL.md   │ Agent Steering Guide         │
│                              │ - Frontmatter tool triggers  │
│                              │ - Routing instructions       │
│                              │ - Test-derived few-shot pairs│
├──────────────────────────────┼──────────────────────────────┤
│ 3. CLI Help & Manual Pages   │ Developer Interface          │
│    (Cobra docs / manpages)   │ - Flag descriptions          │
│                              │ - Placement qualifier enums  │
│                              │ - Shell command examples     │
├──────────────────────────────┼──────────────────────────────┤
│ 4. MCP Tool Schemas          │ Agent Protocol Definitions   │
│    (internal/mcp/server.go)  │ - JSON schema enum values    │
│                              │ - Parameter docstrings       │
│                              │ - Required argument lists    │
└──────────────────────────────┴──────────────────────────────┘
```

### 4.1 Target 1: Human Reference Documentation (`docs/reference/capabilities.md`)

* Produces detailed markdown reference tables indexing all supported languages, operations, and placement qualifiers.
* Details concrete error scenarios (for example, `ErrVisibilityMismatch`, `ErrUnsupportedModifier`).
* Pairs each operation with real-world code snippets and diffs extracted from `txtar` archives.

### 4.2 Target 2: Agent Skill Instructions (`skills/semedit/SKILL.md`)

* Generates agent steering guidance tailored to LLM reasoning patterns.
* Embeds concise few-shot exemplars mapping natural language user intents to precise tool invocations.
* Highlights non-obvious operational rules (such as automatic import resolution and receiver clustering) to prevent redundant planner tool calls.

### 4.3 Target 3: CLI `--help` Text and Manual Pages

* Leverages `github.com/spf13/cobra/doc` to generate man pages (`docs/man/semedit.1`) and CLI markdown trees (`docs/cli/`).
* Ensures command descriptions and flag usage strings reflect the capability matrix without manual duplication.

### 4.4 Target 4: MCP Tool Schemas (`internal/mcp/server.go`)

* Derives MCP `tools/list` schema properties directly from the code matrix.
* Keeps enum constraints (such as `access_modifier` and `placement` values) synchronized with backend validation logic.

---

## 5. CI Drift Invariants & Automated Verification

To eliminate documentation drift permanently, the repository establishes an automated verification invariant enforced through `make check`.

### 5.1 Deterministic Output & Atomic Disk Updates

The generator operates deterministically and safely:

* Sorts all map keys, language entries, and operations alphabetically before rendering.
* Uses fixed newline sequences (`\n`) and standard markdown formatting.
* Employs Go `text/template` with strict data contracts.
* **Atomic Disk Mutations (ADR-0010)**: All generated files are written using atomic file replacement semantics:
  1. Render contents into a temporary scratch file in `.scratch/docgen/`.
  2. Flush and sync bytes to storage (`f.Sync()`).
  3. Replace the target file atomically via `os.Rename`.
  4. Advance file modification time (`mtime`) to invalidate file caches.

### 5.2 CI Documentation Build and Output Gate

The `Makefile` exposes a single generation path and a verification target that depends on it:

```makefile
.PHONY: docgen
docgen: docgen-source ## Build the generated Hugo site into dist/docs
    hugo --source .scratch/docgen --destination "$(CURDIR)/dist/docs" --cleanDestinationDir --minify
    touch dist/docs/.nojekyll

.PHONY: verify-docs
verify-docs: docgen ## Build the site and assert the published files exist
    test -f "$(CURDIR)/dist/docs/index.html"
    test -f "$(CURDIR)/dist/docs/docs/index.html"
    test -f "$(CURDIR)/dist/docs/docs/getting-started/index.html"
    test -f "$(CURDIR)/dist/docs/docs/reference/index.html"
    test -f "$(CURDIR)/dist/docs/.nojekyll"
```

The top-level `make check` target invokes `verify-docs` alongside `fmt`, `tidy`, `lint`, `vuln`, and `test`. Any pull request or commit that modifies language capabilities or test archives rebuilds the Hugo site and fails immediately if a required published page is missing. Generated Hugo source remains temporary in `.scratch/docgen`; only the built `dist/docs` tree is published.

---

## 6. Open Research Questions

1. **Selective vs. Comprehensive `txtar` Mining**:
   * Should `docgen` automatically parse all test archives under `testdata/scripts/`, or should it selectively process only archives annotated with an explicit doc marker (such as `# @doc:include`)?
   * Trade-off: Explicit markers avoid cluttering documentation with negative tests and edge-case regression scripts, whereas automatic extraction ensures total coverage.

2. **Multi-Language Skill Token Budget**:
   * As `semedit` expands to Java, Python, and TypeScript, generating complete multi-language examples inside `skills/semedit/SKILL.md` risks inflating agent context token usage.
   * How should the generator balance comprehensive cross-language coverage against prompt token compactness? Should the agent skill load language-specific sub-guides dynamically?

3. **Schema Generation Timing (Compile-Time vs. Runtime)**:
   * Should `docgen` emit static Go files containing MCP schemas before compilation, or should `internal/mcp` query the capability matrix dynamically at runtime?
   * Trade-off: Static generation allows offline schema inspection and zero runtime reflection overhead, while runtime construction eliminates intermediate generated Go source files.

4. **Diff Formatting Preferences for LLM Steering**:
   * Do LLM planners exhibit higher routing accuracy when presented with unified diffs (`+` / `-` lines) or complete before-and-after source code blocks?
   * Measuring empirical success rates across benchmark arms will guide template formatting.

---

## 7. Proposed Implementation Plan

### Phase 1: Declarative Capability Package

* Define `internal/capability` package containing `LanguageMatrix`, `OpCapability`, and `ConstraintRule`.
* Populate `GolangMatrix` reflecting current Go engine capabilities and constraints.
* Add Rapid property tests asserting that `astedit` and `adapters/golang` behavior mirrors the declared matrix.

### Phase 2: `txtar` Parsing & Example Extraction

* Create `internal/docgen/extract` using `golang.org/x/tools/txtar`.
* Implement parsers for test headers, CLI execution steps, and file diffs.
* Add support for `# @doc:prompt` and `# @doc:operation` annotations in `testdata/scripts/*.txtar`.

### Phase 3: Synthesis Engine & Templates

* Implement `cmd/docgen` with templates for:
  * `docs/reference/capabilities.md` (human documentation).
  * `skills/semedit/SKILL.md` (agent guide with executable few-shot examples).
  * MCP tool definitions and schemas.

### Phase 4: CI Integration & Drift Gate

* Add `make docgen`, `make docgen-source`, and `make verify-docs` to `Makefile`.
* Integrate `verify-docs` into `make check` and call the same target from the Pages workflow.
* Validate that repository passes `make check` and markdown linter checks with zero warnings.
