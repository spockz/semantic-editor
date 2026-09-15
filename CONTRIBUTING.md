# Contributing & Testing Guide

This document defines the development workflow, testing standards, and contribution guidelines for `semedit`.

---

## 1. Development Workflow & Quick Start

### Prerequisites

* Go 1.23+
* `gopls` (`go install golang.org/x/tools/gopls@latest`)
* `make`

### Standard Verification Commands

All changes must pass the root check recipe before submission:

```bash
make check        # Runs formatting, imports optimization, linting, and all tests
make test         # Runs all test tiers (Unit, txtar, and Property-based)
```

---

## 2. The 3-Tier Testing Pyramid

All three tiers run locally during standard development (`go test ./...` / `make test`). Property-based tests run fast in-memory and are not deferred to CI.

```text
                    ┌───────────────────────────────┐
                    │ Tier 3: Property Invariants   │  ~100ms
                    │ (rapid: Invertibility, Refs)  │
                    ├───────────────────────────────┤
                    │ Tier 2: txtar Integration     │  ~50ms / test
                    │ (testscript + live gopls)     │
                    ├───────────────────────────────┤
                    │ Tier 1: Unit Tests            │  <5ms
                    │ (Parsers, Mappings, Protocols)│
                    └───────────────────────────────┘
```

---

## 3. How to Author and Update Tests by Tier

### Tier 1: Unit Tests

* **Scope**: Fast, isolated unit tests for symbol parsing, identifier decomposition (`Type.Method`), coordinate arithmetic, and protocol serializers.
* **Location**: Colocated with implementation code (e.g. `internal/symbol/resolver_test.go`).
* **Authoring Pattern**: Standard Go table-driven tests.

  ```go
  func TestParseQualifiedSymbol(t *testing.T) {
      tests := []struct {
          input    string
          wantRecv string
          wantName string
      }{
          {"Server.Start", "Server", "Start"},
          {"auth.TokenService.Validate", "TokenService", "Validate"},
      }
      for _, tt := range tests {
          recv, name, err := symbol.Parse(tt.input)
          if err != nil || recv != tt.wantRecv || name != tt.wantName {
              t.Fatalf("Parse(%q) = (%q, %q), want (%q, %q)", tt.input, recv, name, tt.wantRecv, tt.wantName)
          }
      }
  }
  ```

* **How to Run**:

  ```bash
  go test ./internal/symbol/...
  ```

---

### Tier 2: Example-Based Integration Tests (`txtar`)

* **Scope**: Multi-file end-to-end refactoring scenarios executing against a real `gopls` instance inside an isolated sandbox (`t.TempDir()`).
* **Location**: `testdata/scripts/*.txtar`.
* **Technology**: `github.com/rogpeppe/go-internal/testscript`.
* **File Structure**:

  ```text
  # Run the semedit command
  exec semedit rename --file api/server.go --symbol "Server.Start" --to "Serve"

  # Assert file transformations
  cmp api/server.go want/api/server.go

  # Assert compiler health in the sandbox
  exec go test ./...

  -- go.mod --
  module example.com/test
  go 1.23

  -- api/server.go --
  package api
  type Server struct{}
  func (s *Server) Start() {}

  -- want/api/server.go --
  package api
  type Server struct{}
  func (s *Server) Serve() {}
  ```

* **How to Author a New Test**:
  1. Create a new file `testdata/scripts/your_case.txtar`.
  2. Add the input files (`-- file.go --`) and the command line (`exec semedit ...`).
  3. Leave the `want/` section blank or omit it.
* **How to Update (`-update` workflow)**:
  Run the test suite with `-update` to automatically capture the resulting files into the `want/` section:

  ```bash
  go test ./... -update
  ```

  Inspect the generated diff with `git diff testdata/scripts/`. Verify that the change is correct before committing.

---

## 4. Tier 3: Property-Based & Metamorphic Tests

* **Scope**: Validates mathematical refactoring invariants across hundreds of generated code variations.
* **Location**: `internal/adapters/golang/property_test.go`.
* **Technology**: `pgregory.net/rapid`.
* **Core Invariants Tested**:
  1. **Round-Trip Invertibility ($R^{-1}(R(P)) \equiv P$)**: Renaming $A \to B$ followed immediately by $B \to A$ must produce the exact original source code (modulo formatting).
  2. **Reference Count Conservation**: If symbol $A$ had $N$ references prior to rename, symbol $B$ must have exactly $N$ references after rename.
  3. **Compilation Health**: If the package compiled with zero diagnostics before refactoring, it must compile with zero diagnostics after refactoring.
  4. **Behavioral Equivalence**: All package unit tests that passed before refactoring must pass after refactoring.
* **How to Run**:
  Standard run (runs default iteration count, completes in $<200\text{ms}$):

  ```bash
  go test -run TestProperty_ ./internal/adapters/golang/...
  ```

  Deep fuzz/stress run (increases iteration count):

  ```bash
  go test -run TestProperty_ -rapid.checks=2000 ./internal/adapters/golang/...
  ```

* **How to Update**:
  When adding new Go language constructs (e.g. generics, type aliases, interface embedding), extend the AST generator (`GenerateRandomGoPackage`) in `property_test.go` to include the new construct. The property assertions automatically apply to all generated variations.

---

## 5. Coding Standards & Invariants

* **Error Handling**: Wrap errors with context using `fmt.Errorf("...: %w", err)` and inspect using `errors.Is` / `errors.As`.
* **No Unsolicited File Writes**: Tooling must never create or modify workspace configuration files (`go.work`, `Cargo.toml`) on disk without explicit user approval.
* **Sandboxing**: Temporary scratch files or experimental code must always use a gitignored `.scratch/` directory.
* **Comments**: Top-of-file comment explaining the file's WHY and purpose. Comment non-obvious constraints; avoid decorative comment banners.

---

## 6. Architectural Decisions (ADR) & Research Governance

To maintain documentation integrity and conserve LLM context windows, all architectural changes and technical investigations must adhere to the following governance:

### A. Research Questions (`docs/research/`)

* **When to Create**: Open technical spikes, protocol feasibility investigations, benchmarks, or trade-off analyses before deciding on an architecture.
* **Naming**: `docs/research/RQ-XXXX-<kebab-case-title>.md` (sequential 4-digit zero-padded, e.g. `RQ-0014-*.md`).
* **Lifecycle**: Begins with `Status: Open`. When consensus or empirical evidence is established, update to `Status: Resolved` and record key findings.

### B. Architecture Decision Records (`docs/adr/`)

* **When to Create**: Any accepted architectural boundary, immutable invariant, protocol/tool contract, or structural system decision.
* **Naming**: `docs/adr/XXXX-<kebab-case-title>.md` (sequential 4-digit zero-padded, e.g. `0011-*.md`).
* **Required Structure**: Status, Date, Context, Decision, Invariants, Consequences.

### C. Master Index Invariant

* Any addition or status change to an ADR or RQ **must** update the index table in [docs/adr/README.md](docs/adr/README.md) or [docs/research/README.md](docs/research/README.md) within the same commit.
* Agents must consult the index first and avoid reading full ADR or RQ files unless directly relevant to the immediate task.
