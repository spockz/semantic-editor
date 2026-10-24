<!-- This README introduces the implemented product, evidence, and boundaries; detailed contracts live in the linked code and decision records. -->
# semantic-editor (`semedit`)

**Intent-driven code editing for AI agents: LLMs describe the change; local compiler, AST, and language-server tools perform the transformation.**

`semedit` exposes a command-line interface (CLI) and a Model Context Protocol (MCP) server. Go has the broadest editing support. Seven additional language backends provide explicitly bounded lookup, rename, or diagnostics capabilities.

The efficiency ambition remains **up to 100× fewer generated edit tokens and 10× faster execution** in favorable refactoring scenarios. These are analytical upper bounds we aim to approach. Current benchmarks demonstrate speed and token savings in individual cases; improved first-attempt correctness remains a goal.

## Getting started

### Install the CLI

`semedit` is a Go executable, so install the Go toolchain version required by [go.mod](go.mod) or newer. Then install the CLI:

```sh
go install github.com/spockz/semantic-editor/cmd/semedit@latest
```

For a local checkout, run `go install ./cmd/semedit` from the repository root.

Go installs executables into `GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is unset. Add that directory to your shell's `PATH`, then check the CLI:

```sh
semedit --help
```

User installation does not require the development linters or Hugo.

### Language setup

Each language backend uses its own preinstalled tools in addition to the `semedit` CLI. Follow the language guide for operation limits and examples.

#### Go

Install [`gopls`](https://go.dev/gopls/) for Go semantic rename and diagnostics; AST-based navigation does not require it:

```sh
go install golang.org/x/tools/gopls@latest
```

See the [Go guide](skills/semedit/references/golang.md).

#### Rust

Install the [`rust-analyzer` binary](https://rust-analyzer.github.io/book/installation.html) and select a `.rs` file. Rust requires explicit workspace trust and confines rename to the selected file. The backend disables build scripts, procedural macros, and check-on-save; rust-analyzer may run Cargo metadata while loading the project. See the [Rust guide](skills/semedit/references/rust.md).

#### Java

Install a Java 21+ runtime and an [Eclipse JDT LS distribution](https://github.com/eclipse-jdtls/eclipse.jdt.ls). Java requests require explicit workspace trust and `jdtls_home` pointing to that distribution. See the [Java guide](skills/semedit/references/java.md).

#### Scala

Install Java 21+ and a pinned [Metals distribution](https://scalameta.org/metals/docs/integrations/new-editor/). Requests require explicit workspace trust and a selected `.scala` file; configure `metals_bin` or `metals_home` and the Java executable/version. Semedit does not invoke a build tool. See the [Scala guide](skills/semedit/references/scala.md).

#### Haskell

Install GHC and a matching `haskell-language-server-wrapper` using the [HLS installation guide](https://haskell-language-server.readthedocs.io/en/stable/installation.html). Requests can pin them with `--ghc-bin` and `--hls-bin`, and optionally require recorded versions with `--ghc-version` and `--hls-version`. Lookup is limited to an explicitly standalone, trusted `.hs` file; Cabal/Stack project markers are rejected. See the [Haskell guide](skills/semedit/references/haskell.md).

#### Kotlin

Install the [`fwcd/kotlin-language-server`](https://github.com/fwcd/kotlin-language-server) executable. Semedit currently expects this server, which its upstream describes as deprecated; requests require explicit trust and a selected `.kt` or `.kts` file. See the [Kotlin guide](skills/semedit/references/kotlin.md).

#### Bash

Install [`bash-language-server`](https://github.com/bash-lsp/bash-language-server) and make its executable available to semedit. Requests require explicit workspace trust and a selected `.sh` or `.bash` file; the backend does not execute scripts. See the [Bash guide](skills/semedit/references/bash.md).

#### Makefile

Install [`make-ls`](https://github.com/owenrumney/make-ls) and make the executable available on `PATH`. Requests require explicit workspace trust and a selected Makefile; the backend does not run recipes. See the [Makefile guide](skills/semedit/references/make.md).

### Install into a coding harness

Register the installed executable with Codex or GitHub Copilot CLI. Choose the command for your harness:

```sh
semedit install codex --scope user --binary "$(command -v semedit)"
semedit install copilot --scope user --binary "$(command -v semedit)"
```

For registration in a single project, run the selected command from that project's root with `--scope workspace` instead of `--scope user`, or supply `--workspace /absolute/path/to/project`. The installer records the absolute executable path and configures the harness to launch `semedit mcp --profile full`. It preserves unrelated configuration and rejects conflicting semedit registrations. Add `--dry-run` to preview a registration. Commands print concise human-readable output by default; pass `--json` to `install`, `integration status`, or `uninstall` for structured JSON on stdout. Successful installs print the config path and restart guidance. JSON mode prints only JSON.

Check or remove a registration using the same target and scope:

```sh
semedit integration status codex --scope user
semedit uninstall codex --scope user
```

Use `copilot` in place of `codex` for GitHub Copilot CLI. Restart or reconnect the harness's MCP session after installation and confirm that its tool inventory includes `semantic_lookup`. The installer registers the MCP server but does not install the [semedit steering skill](skills/semedit/SKILL.md); install the `skills/semedit/` directory through your harness's skill mechanism if you want the accompanying workflow guidance.

For other MCP clients, configure a stdio server with the absolute path to `semedit`, arguments `["mcp"]`, and the intended project as its working directory. See [Build and use](#6-build-and-use) for CLI editing examples and MCP profiles.

## 1. Why semantic editing

A rename or declaration move can require a text-editing agent to read and reproduce code across many files. With semantic editing, the agent names the symbol and the intended change, and local tooling handles coordinates and mechanical edits.

```text
Agent intent → CLI or MCP → Registered operation → Language tooling → Result and diagnostics
```

This reduces generated diffs, repeated source reads, and opportunities for patch-application mistakes. Go symbol lookup uses Go's AST; other backends use native language-server symbols. A rename or move can reuse existing source without asking the model to reproduce it. Insertion and replacement still require the model to supply new source, and tool schemas, results, reasoning, and verification all contribute to total session cost.

Deterministic editing does not prove the agent chose the correct change. Operations can fail, language tools can time out, and an applied edit can introduce compiler diagnostics. The workflow exposes these outcomes so an agent can continue a multi-step refactoring.

## 2. Performance: ambition and measured evidence

### Analytical targets

The original multi-file refactoring model estimates the potential benefit of replacing repeated text-editing turns with a compact semantic request:

| Mechanism | Favorable analytical scenario | Target |
| :--- | :--- | :--- |
| Generated edit tokens | Roughly 2,000–3,000 diff tokens replaced by about 25 argument tokens for a rename | Up to 100× fewer output tokens |
| Remote turns and latency | A sequence of searches, reads, edits, and repairs replaced by one semantic request plus local execution | Up to 10× faster completion |
| Context reuse | Less repeated source and repair history in subsequent prompts | Lower context consumption and model cost |
| Edit reliability | Structural validation removes classes of malformed text patches | Fewer repair loops; improved first-attempt correctness remains to be demonstrated |

These are theoretical upper-bound scenarios, not average measured performance or guarantees. End-to-end results depend on task, model, harness, cache behavior, language-server startup, and verification cost. [RQ-0014](docs/research/RQ-0014-benchmarking-token-latency-and-eval-harness.md) and [RQ-0015](docs/research/RQ-0015-multi-dimensional-benchmark-matrix.md) track the broader evaluation.

### Measured results as of 2026-09-26

The generated benchmark documentation reports these three headline metrics for historical runs whose semedit restriction policy is recorded as `unspecified`:

| Measure | Result | Evidence |
| :--- | :--- | :--- |
| Best speed improvement | **56.9% less elapsed time**, about **2.32×** speedup | `task-07-generate-template-main`, standard small context, [Codex prescriptive run](data/benchmarks/results/run-20260924-codex-prescriptive/task-07-generate-template-main/) |
| Best token reduction | **87.5% fewer cache-adjusted token units** | `task-11-mixed-sink-api-migration`, standard small context, [Codex prescriptive run](data/benchmarks/results/run-20260924-codex-prescriptive/task-11-mixed-sink-api-migration/) |
| First-time right | **MCP 195/209 (93.3%) vs baseline 196/209 (93.8%)**, −0.5 percentage points | Initial correctness-oracle results across all publishable standard-context paired observations |

The speed and token figures are separate best cases where both the baseline and semedit arms passed their correctness oracle. They are not averages or necessarily the same model/task pair. Cache-adjusted token units are uncached input + visible output + reasoning + cached input / 10; they differ from the generated-output-token target above. The first-time-right comparison excludes verified/self-correction contexts and shows no improvement yet.

[ADR-0043](docs/adr/0043-benchmark-run-aggregation-and-best-case-publication.md) defines publication and selection rules. The generated site includes complete run evidence, aggregate ranges, and an interactive benchmark browser. Run `make docs` to generate it under `dist/docs/`; the underlying [run records](data/benchmarks/results/) remain available in the repository.

The harness now resolves one execution plan, retains every terminal job outcome, and distinguishes `read`, `write`, and `readwrite` semedit policies. New runs default to `write`; historical unspecified policies remain separate. Diagnostic questioning about tool use stays outside measured task tokens, time, turns, and correctness outcomes. See [ADR-0050](docs/adr/0050-benchmark-planning-sessions-and-arm-policy.md) and the [benchmark architecture guide](tools/benchmark-harness/README.md).

## 3. Current language support

Support is operation-specific. A shared interface does not imply that every language supports every edit.

| Language | Tooling | Implemented scope |
| :--- | :--- | :--- |
| **Go** | Go parser/AST, `gopls`, Go formatting/import tooling | Symbol lookup, workspace rename, bounded callable extraction with typed data flow and revision-bound nested-block paths, value extraction, single or explicitly selected repeated constant extraction, callable/value inlining, structural edits, scaffolding, imports, dependencies, assertion transforms, and verification |
| **Rust** | `rust-analyzer` | Trusted selected-file lookup, rename, outline, bounded local-value and callable extraction, checked fixed-width decimal constant extraction, and bounded local-value/callable inlining; edits outside the selected `.rs` file are rejected. The backend disables build scripts, procedural macros, and check-on-save; rust-analyzer may run Cargo metadata during project loading |
| **Java** | Eclipse JDT LS; separate bounded Maven actions | Trusted lookup, rename, selected-file callable/value/constant extraction, value/callable inlining, signature changes, contract/interface implementation, declaration moves, formatting, import organization, diagnostics, and explicit Maven actions; field extraction is unsupported pending a safe native edit contract |
| **Scala** | Metals | Trusted selected-file lookup and outline; bounded value/callable extraction validates native edit transactions. Inline-value remains unavailable until the configured Metals runtime returns its action and the production edit transaction is verified. No build import or compiler diagnostics |
| **Haskell** | Haskell Language Server and matching GHC | Explicitly standalone, trusted `.hs` lookup only; project cradles, compilation, and mutation are unavailable |
| **Kotlin** | `kotlin-language-server` | Trusted selected-file `.kt`/`.kts` lookup and diagnostics in a source-only scratch workspace; no mutation |
| **Bash** | `bash-language-server` | Trusted selected-file `.sh`/`.bash` lookup and diagnostics in a source-only scratch workspace; no mutation or script execution by the backend |
| **Makefile** | `make-ls` | Trusted selected-file target and assignment-variable lookup; no conditional-symbol lookup, diagnostics, mutation, or recipe execution by the backend |

External-tool backends require preinstalled tooling and explicit request-scoped workspace trust. Some require configured distribution paths and recorded runtime versions. Kotlin and Bash verification require an explicit matching diagnostics report; silence does not mean the file is clean. Scratch workspaces limit the copied source but are not an operating-system sandbox: external tools may perform additional reads or launch their own helpers after trust.

Java's JDT LS operations do not execute Maven or Gradle. Maven import requires explicit opt-in; the separate Maven commands run fixed goals against a trusted in-workspace root POM, with scratch-local state and offline mode by default. See [ADR-0035](docs/adr/0035-java-maven-import-boundary.md), [ADR-0036](docs/adr/0036-java-bounded-verify-actions.md), and [ADR-0037](docs/adr/0037-java-bounded-maven-actions.md).

Kotlin, Bash, and Makefile boundaries are documented in [ADR-0047](docs/adr/0047-kotlin-read-only-backend.md) and [ADR-0048](docs/adr/0048-bash-and-makefile-read-only-backends.md). TypeScript/JavaScript, Python, C#, Elixir, Elm, Dart, and C/C++ do not currently have registered runtime backends.

## 4. Editing operations

Registered semantic operations share CLI and MCP definitions. MCP `--profile` selects execution behavior (`full` or `mutations-only`); `--tool-profile` independently selects discovery (`base` by default, or `full`). The default base catalog includes `semantic_workspace_outline`, `refactor_change_signature`, `refactor_extract_callable`, and `semantic_set_doc_comment`. Each refactoring has one canonical `refactor_*` MCP name and one canonical `refactor-*` CLI command. `mutations-only` filters read-only tools even when the full tool catalog is selected. Use command help or the active MCP tool schema for exact parameters and language constraints.

| Intent | CLI command | MCP tool |
| :--- | :--- | :--- |
| Locate a symbol | `lookup` | `semantic_lookup` |
| Rename a symbol and supported references | `refactor-rename` | `refactor_rename` |
| Add a Go construct | `insert-construct` | `semantic_insert_construct` |
| Replace or upsert a Go construct | `replace-construct` | `semantic_replace_construct` |
| Remove one Go declaration or switch case; safe deletion fails closed until preflights are complete | `remove-construct` | `semantic_remove_construct` |
| Move a Go declaration within a file or package; cross-package moves support exported functions, constants, aliases with predeclared structural type expressions, variables, and bounded defined-type groups within one module | `refactor-move-declaration` | `refactor_move_declaration` |
| Replace a Go function or method body | `replace-body` | `semantic_replace_body` |
| Extract supported Go or Java statement regions, contiguous Rust or Scala direct calls to the same callee, or inspected Go direct-statement paths with the matching file revision into a helper | `refactor-extract-callable` | `refactor_extract_callable` |
| Extract a supported Go, Java, Rust, or bounded Scala value; Go supports guarded call arguments, repeated scalar expressions within one call or contiguous direct-assignment range, simple assignment RHS expressions in direct or nested statement blocks, and direct or nested returns with one unnamed result; Go and Java support their documented constant forms, and Rust accepts checked fixed-width decimal addition initializers | `refactor-extract-value` | `refactor_extract_value` |
| Make one Java method static through JDT LS and update resolved callers | `refactor-make-static` | `refactor_make_static` |
| Make one unique Java method-local `var` type explicit through a direct JDT LS token edit, using its inspected file revision | `refactor-make-type-explicit` | `refactor_make_type_explicit` |
| Inline a supported Go, Java, or bounded Rust value; inline one supported callable or selected multiple Go free-function calls | `refactor-inline-value`, `refactor-inline-callable` | `refactor_inline_value`, `refactor_inline_callable` |
| Encapsulate one private Go struct field with typed accessor generation and supported direct reference rewrites | `refactor-encapsulate-field` | `refactor_encapsulate_field` |
| Move a contiguous run of private Go fields and one pointer method into a same-file delegate while retaining the owner wrapper | `refactor-extract-delegate` | `refactor_extract_delegate` |
| Replace a Go constant, variable, or type alias declaration | `replace-decl` | `semantic_replace_decl` |
| Create a Go file with an inferred package | `scaffold-file` | `semantic_scaffold_file` |

| Organize Go imports | `organize-imports` | `semantic_organize_imports` |
| Add a Go module dependency | `add-build-dependency` | `semantic_add_build_dependency` |
| Change supported Go test assertion failure modes | `assertion-mode` | `semantic_assertion_mode` |
| Format/check supported sources | `verify` | `semantic_verify` |
| Compile or run Java tests | `maven-compile`, `maven-test` | `semantic_maven_compile`, `semantic_maven_test` |
| Capture or restore a workspace snapshot | `snapshot`, `undo` | `semantic_snapshot`, `semantic_undo` |

Cross-package construct moves support exported free functions, standalone constants, aliases whose structural type expressions use only predeclared type names, and bounded defined types with their co-located exported direct methods. Alias moves require the candidate overlay to preserve exact unaliased type identity, so package-sensitive private members of anonymous types refuse. Moves require existing files in the same Go module and compiler-checked reference rewrites. Test-file references to moved constants, aliases, or defined-type groups; unsupported dependencies; import cycles; and ambiguous source evidence are refused. External consumers, defined-type nominal identity, and reflection names are outside the guarantee.

[ADR-0049](docs/adr/0049-ast-construct-taxonomy-and-mutation-lifecycle.md) separates the construct lifecycle:

* **Insert** creates a new construct at a chosen location and rejects declaration collisions. It cannot overwrite existing code.
* **Replace** changes a construct in place. Declaration upsert can insert an absent declaration at its canonical layout position; replacement has no placement parameter.
* **Move** relocates an existing declaration within its file or package, or a supported exported Go declaration across packages, carrying attached comments without requiring its source again.

These operations currently execute Go edits. The broader construct taxonomy in the ADRs does not enable mutation in other languages. The former discrete insertion tools, including `insert-declaration`, `insert-function`, `insert-type`, `insert-decl`, and `insert-case`, have been retired from the registry.

The MCP-only `semantic_batch` runs registered edits sequentially against the supplied workspace. Eligible Go edits, declaration moves, and rename share in-memory source buffers and the MCP instance's gopls process. Cross-package moves typecheck their complete plans against the current working revision and rewrite module-wide references. Built-in formatting and imports operate on those buffers before publication. Disk-only operations publish the preceding segment first. Outputs are staged together and replaced atomically per file; multi-file publication is not a filesystem transaction. Batches coalesce diagnostics, scope formatting to changed files, and return the actual persisted diff. Earlier successful edits remain applied if a later edit fails. CLI batch exposure remains an [open investigation](docs/research/RQ-0030-registry-backed-cli-batches.md).

`report_feedback` is a separate MCP utility that returns a structured issue draft for user review and manual posting. It does not save or submit the report. Development-only `semantic_reload` is available when the server starts with `--live-reload`.

## 5. Architecture and mutation boundaries

The implementation is Go-native. It combines Go AST transformations with adapters around external language tools; it does not implement a universal AST or its own typechecker.

```mermaid
flowchart TD
    Agent[Agent intent] --> CLI[CLI]
    Agent --> MCP[MCP server]
    CLI --> Operations[Typed operation registry]
    MCP --> Operations
    Operations --> Service[Backend selection and capability gates]
    Operations --> AST[Go AST operations]
    Service --> Go[Go resolver and gopls adapter]
    Service --> External[Bounded external language-server adapters]
    AST --> Pipeline[Atomic writes, formatting, and diagnostics]
    Go --> Pipeline
    External --> Results[Structured results and errors]
    Pipeline --> Results
```

[ADR-0021](docs/adr/0021-language-backend-service-boundary.md) defines the backend service boundary; [ADR-0034](docs/adr/0034-central-operation-registry.md) defines the operation registry. Language backends own their native lookup, trusted edit validation, and diagnostics. The MCP server publishes structured output schemas and timing metrics alongside human-readable results ([ADR-0042](docs/adr/0042-mcp-structured-tool-output-schemas.md)).

Source updates use atomic replacement and advancing timestamps. Edits may remain applied with compiler diagnostics so multi-step refactorings can pass through temporarily broken states. Snapshots provide explicit, conflict-checked undo. Direct batches do not provide transaction-wide rollback or an isolated working copy; callers own any workspace isolation they need. See [ADR-0010](docs/adr/0010-disk-synchronization-and-cache-invalidation.md), [ADR-0018](docs/adr/0018-transactional-snapshots-and-undo.md), and [ADR-0032](docs/adr/0032-direct-and-isolated-batch-execution.md).

Verification uses configured language servers and can write files: Go verification formats before collecting gopls diagnostics, and Java formatting/import modes mutate the selected file. `--check-only` skips normalization. Project-root `.semedit.yaml` hooks select additional normalization and checks; applicable golangci configuration automatically enables its filesystem check after edits are published. Bash and Make servers are selected automatically when matching sources and server executables are present; selected unsupported verification fails explicitly. See the [configuration contract](docs/research/RQ-0029-project-normalization-configuration.md) for overrides, scope, and supported actions. Workspace manifests such as `go.work` are not created or changed implicitly; explicit dependency operations can update `go.mod` and `go.sum`.

The original dual-engine design includes an error-tolerant CST fallback. Automatic routing to Tree-sitter, `ast-grep`, or Comby is not implemented. The current Go resolver uses `go/parser`, and Go edits use native AST tooling. There is no automatic discovery of another server's `gopls` MCP tools or short-circuit routing through that server.

## 6. Build and use

Use the Go version declared in [go.mod](go.mod), `make`, and the tools required by the selected backend. Go semantic rename requires `gopls`. Development checks also need the repository's configured formatters, linters, and Hugo tooling; see [CONTRIBUTING.md](CONTRIBUTING.md) and the [Makefile](Makefile).

```sh
make check
make build
./bin/semedit-next --help
./bin/semedit-next lookup --file cmd/semedit/main.go --symbol main
```

`make build` produces the development binary `bin/semedit-next`. `make promote` runs checks and promotes it to the stable `bin/semedit` used by long-lived integrations.

The tracked `.codex/config.toml.example` and `.agents/plugins/workspace-servers/mcp_config.json.example` development templates use `{{SEMEDIT_BINARY}}` and `{{SEMEDIT_NEXT_BINARY}}` placeholders. Install the development configurations from the checkout:

```sh
make install-mcp
```

This builds the binaries and replaces the placeholders with absolute paths to `bin/semedit` and `bin/semedit-next` in the local configurations. Generated configurations are ignored by Git; edit the `.example` templates and rerun the target to change them. Restart the harness after installation. Use `semedit install codex` or `semedit install copilot` for normal user or workspace registration as described in Getting started.

For a Go project, run the binary from the intended workspace and name the declaration rather than calculating coordinates:

```sh
semedit refactor-rename --file server.go --symbol Server.Start --to Serve
semedit replace-body --file server.go --symbol Server.Serve --body 'return nil'
```

The second example assumes `Server.Serve` returns an error. The replacement body contains statements without outer braces.

Start a stdio MCP server from the intended workspace:

```sh
semedit mcp --enabled-languages=go
```

`--profile=full` is the default; `--profile=mutations-only` omits lookup tools. `--enabled-languages` limits the exposed language set. Normal CLI and MCP use inherit the user's Go environment, including cache paths and workspace selection. MCP's `--go-base-dir` explicitly enables isolated Go state beneath the selected directory; the benchmark runner uses this mode for its fixtures. Ordinary semantic operations keep module files read-only, while explicit dependency operations may update them. Configure the client with the executable path and working directory explicitly. The [semedit steering skill](skills/semedit/SKILL.md) explains intent selection and operation boundaries.

MCP initialization includes prescriptive guidance by default: inspect the tool inventory and use applicable semantic tools for mutations. `--instructions 'Custom guidance'` replaces it; `--instructions=` omits server-wide guidance. Individual tool descriptions remain available.

## 7. Open research and future scope

The [ADR index](docs/adr/README.md) records accepted decisions and their evolution; the [research index](docs/research/README.md) tracks investigations. A resolved feasibility question does not necessarily mean its integration has shipped.

* **Broader refactoring support:** the registered Go, Java, Rust, and Scala operations remain deliberately bounded by operation-specific syntax and edit-safety contracts. Additional data-flow shapes, language-operation pairs, and richer disambiguation remain under investigation. Bash and Makefile construct editing need parser evidence beyond language-server symbol ranges ([RQ-0024](docs/research/RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md), [RQ-0060](docs/research/RQ-0060-go-refactoring-tool-implementation.md), [RQ-0037](docs/research/RQ-0037-bash-language-server-and-construct-editing.md), [RQ-0038](docs/research/RQ-0038-makefile-language-server-versus-parser.md)).
* **Post-edit context and normalization:** bounded result projections, project quality policy, and configurable normalization remain under investigation ([RQ-0027](docs/research/RQ-0027-preventing-read-files.md) through [RQ-0029](docs/research/RQ-0029-project-normalization-configuration.md)).
* **Benchmark integration:** native instruction-delivery channels have been identified, but the transport implementation described in [RQ-0039](docs/research/RQ-0039-benchmark-policy-instruction-transport.md) remains deferred. [ADR-0044](docs/adr/0044-benchmark-experiment-dimensions-and-result-semantics.md) remains proposed.
* **Runtime and workspace evolution:** persistent server lifecycle, caching, and broader workspace topologies remain research topics. Concurrent polyglot server orchestration and embedded speculative workspace isolation are not current features.

## 8. Requirements and invariant tests

The [active requirements catalog](docs/adr/README.md#active-requirements-invariants) is the project-level source of architectural invariants. Each requirement has a stable `REQ-NNN` identifier and links to the ADRs that define its scope. Treat these invariants as requirements, not as a complete description of the test suite.

Every active requirement MUST have at least one regression test that exercises the observable behavior. Make the link mechanically discoverable in one of two ways: include the exact requirement ID in the test or subtest name (for example, `TestREQ004AtomicMutation` or a txtar scenario named `req-006-rename-go`), or have the test case output record the requirement ID together with the observed and expected invariant value. Test names and output should identify the requirement directly; a prose-only mapping is not sufficient.

A test need not map to an invariant. Unit, property, integration, protocol-failure, regression, and benchmark tests may cover behavior outside the invariant catalog. Requirement coverage and overall suite size are different measures. A change that adds or changes an invariant MUST update its requirement entry and add or update its linked regression test in the same change.

Current test anchors use the requirement ID in the Go test name:

| Requirement | Regression test anchor |
| :--- | :--- |
| REQ-001 | [`TestREQ001SemanticIntentCLIContracts`](cmd/semedit/main_test.go) |
| REQ-002 | [`TestREQ002REQ006PublicRegistryAndCLIContractCoverage`](cmd/semedit/main_test.go) |
| REQ-003 | [`TestREQ003BackendRejectsUnsafeRenameWithoutWrite`](internal/backend/rust_test.go) |
| REQ-004 | [`TestREQ004WriteAtomicPreservesPermissionsAndAdvancesMtime`](internal/pipeline/pipeline_test.go), [`TestREQ004RenameReportsIntroducedDiagnostics`](internal/backend/backend_test.go) |
| REQ-005 | [`TestREQ005WorkspaceTrustRequiresExplicitCanonicalConsent`](internal/backend/backend_test.go) |
| REQ-006 | [`TestREQ002REQ006PublicRegistryAndCLIContractCoverage`](cmd/semedit/main_test.go) |
| REQ-007 | [`TestREQ007RunPhasePreservesHookOrderAndPartialFailure`](internal/projectverify/executor_test.go) |
| REQ-008 | [`TestREQ008LoadAllBenchmarkComparisonsExcludesIncompleteRuns`](cmd/docgen/benchmarks_test.go) |

## 9. Development and documentation

`make check` runs mechanical fixes, formatting, dependency tidying, linting, tests, and generated-documentation checks. Public semantic behavior is exercised through CLI [txtar contracts](testdata/scripts/), with comparable coverage for each implemented language-operation pair. Unit and property tests complement those contracts ([ADR-0028](docs/adr/0028-cross-language-cli-txtar-coverage.md)).

Filesystem-backed tests place disposable workspaces and Go compiler temporary files on the managed benchmark RAM volume; provision it with `make bench-ram-mount` before running checks. Unit fixtures should use in-memory buffers when that preserves the behavior under test. Set `ALLOW_SSD_TEST_TEMP=1` only when an explicit SSD-backed temporary run is intended ([ADR-0093](docs/adr/0093-ram-backed-test-workspaces-and-in-memory-fixtures.md)).

The Go documentation generator combines registry-derived capabilities, executable examples, and benchmark evidence into a Hugo/Hextra site. Start with [CONTRIBUTING.md](CONTRIBUTING.md) for development and [the benchmark architecture guide](tools/benchmark-harness/README.md) for benchmark changes.

## 10. Repository Layout

The main source, documentation, and test directories are:

```text
semedit/
├── cmd/semedit/            # Executable entry point for CLI and MCP modes
├── cmd/docgen/             # Documentation site and benchmark publication generator
├── internal/
│   ├── cli/                # CLI commands generated from the operation registry
│   ├── mcp/                # MCP server, tool schemas, and feedback reports
│   ├── operation/          # Shared operation definitions, dispatch, and batch execution
│   ├── capability/         # Language capability documentation contracts
│   ├── backend/            # Language-neutral service and language implementations
│   ├── backends/           # Default backend registration and runtime configuration
│   ├── adapters/golang/    # gopls integration and Go dependency operations
│   ├── astedit/            # Go AST construct insertion, replacement, and relocation
│   ├── symbol/             # Go AST symbol resolution
│   ├── lsp/                # Bounded stdio JSON-RPC transport for language servers
│   ├── pipeline/           # Atomic writes, formatting, and compiler diagnostics
│   ├── snapshot/           # Snapshot journal and conflict-checked undo
│   ├── maven/              # Bounded Java Maven verification actions
│   ├── integration/        # Agent harness installation and configuration
│   ├── gocache/            # Go environment and opt-in isolated state
│   ├── telemetry/          # Operation timing and metrics
│   ├── assertmode/         # Go test assertion transformations
│   └── testtools/          # Shared test support
├── tools/benchmark-harness/ # Benchmark planning, execution, and correctness oracles
├── data/benchmarks/         # Benchmark run records and supporting data
├── testdata/
│   ├── scripts/            # CLI txtar contracts for public operations
│   ├── bench/              # Benchmark task fixtures
│   └── bench-oracles/      # Hidden benchmark acceptance tests
├── docs/
│   ├── adr/                # Architecture decisions and index
│   └── research/           # Research questions, findings, and index
├── skills/                 # Agent steering and documentation review skills
├── .github/                # CI and repository automation
├── Makefile                # Formatting, linting, tests, builds, and documentation checks
├── CONTRIBUTING.md         # Development workflow and test authoring guide
└── README.md
```
