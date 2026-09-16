# ADR-0014: Adoption of Cobra CLI Framework

* **Status**: Accepted
* **Date**: 2026-09-16

---

## Context

The initial CLI interface for `semedit` used the standard library `flag.FlagSet` with manual command dispatch via a string `switch` in `run()`. As `semedit` grew to support multiple subcommands (`lookup`, `rename`, `insert`, `insert-func`, `insert-type`, `insert-decl`, `imports`, `get`, `mcp`), manual dispatch led to repetitive boilerplate:

1. **Repetitive Flag Parsing**: Each subcommand had to instantiate its own `flag.FlagSet`, define custom flag variables, parse arguments, and manually format errors.
2. **Inconsistent Help & Usage**: Standard library `flag` lacks built-in nested subcommand help, auto-generated documentation, and standard posix-compliant flag conventions (such as combining short and long flags seamlessly).
3. **Architectural Standardization**: The learned engineering rule prescribes: *For golang apps without an existing CLI framework, use Cobra*. Adopting Cobra standardizes command lifecycle, flag binding, and subcommand composition across the ecosystem.

---

## Decision

1. **Standardize on `github.com/spf13/cobra`**:
   Refactor `main.go` to declare CLI commands using Cobra's `*cobra.Command` tree.

2. **Root Command & Default Output**:
   The root command represents `semedit`. Invoking `semedit` with zero arguments prints `"semedit initialized"` to preserve existing initialization behavior.

3. **Subcommand Parity**:
   Define dedicated `*cobra.Command` instances for:
   * `lookupCmd`
   * `renameCmd`
   * `insertCmd`
   * `insertFuncCmd`
   * `insertTypeCmd`
   * `insertDeclCmd`
   * `importsCmd`
   * `getCmd`
   * `mcpCmd`

4. **Error & Usage Silence**:
   Set `SilenceErrors: true` and `SilenceUsage: true` on all commands. This ensures Cobra does not automatically print usage messages or unformatted errors, allowing `semedit` to format its stderr outputs and exit codes deterministically for acceptance tests.

5. **Testability Contract**:
   Keep `run(args []string) int` as the programmatic entry point tested by `testscript` / `txtar`. `run` configures a fresh Cobra root command instance with `SetArgs(args)` and captures exit codes explicitly.

---

## Invariants

* All existing CLI command names, flag names, flag defaults, and output contracts must remain backward-compatible with existing scripts and `txtar` tests.
* Calling `semedit` without arguments must output `"semedit initialized\n"` and exit with code 0.
* CLI commands must delegate heavy transformation and validation logic to internal domain packages (`internal/astedit`, `internal/pipeline`, `internal/symbol`, `internal/mcp`).
* Commands must return exit code 0 on success and exit code 1 on user or validation errors.

---

## Consequences

* The CLI entry point gains robust POSIX flag parsing, shorthand flags, and consistent subcommand lifecycle.
* Dependency on `github.com/spf13/cobra` is introduced into `go.mod`.
* Custom argument splitting and flag set declarations in `main.go` are eliminated in favor of idiomatic Cobra command definitions.
