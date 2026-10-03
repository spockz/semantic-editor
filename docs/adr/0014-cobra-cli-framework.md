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
   The root command represents `semedit`. Invoking `semedit` with zero
   arguments renders standard Cobra help.

3. **Registry-Derived Subcommands**:
   Build one `*cobra.Command` for every registered operation with a CLI name.
   Command names, summaries, flags, defaults, validation, and result
   rendering derive from the central operation registry rather than per-command
   adapters.

4. **Error & Usage Silence**:
   Set `SilenceErrors: true` and `SilenceUsage: true` on all commands. This ensures Cobra does not automatically print usage messages or unformatted errors, allowing `semedit` to format its stderr outputs and exit codes deterministically for acceptance tests.

5. **Testability Contract**:
   Keep `run(args []string) int` as the programmatic entry point tested by `testscript` / `txtar`. `run` configures a fresh Cobra root command instance with `SetArgs(args)` and captures exit codes explicitly.

6. **CLI Payloads from Stdin**:
   Registry-derived CLI string parameters named `source`, `body`, or `case` accept `-` as a request to read their complete payload from Cobra's stdin. The CLI expands these markers once after flag parsing and before operation parsing or dispatch. This convention stays at CLI ingress; MCP JSON values, including the literal string `-`, retain their existing meaning.

---

## Invariants

* CLI command names, flags, defaults, validation, and output contracts derive
  from the central operation registry. A registry change intentionally changes
  the CLI contract; compatibility shims are not maintained.
* Calling `semedit` without arguments renders command help and exits with code
  0.
* CLI commands must delegate heavy transformation and validation logic to internal domain packages (`internal/astedit`, `internal/pipeline`, `internal/symbol`, `internal/mcp`).
* Commands must return exit code 0 on success and exit code 1 on user or validation errors.
* CLI `--source -`, `--body -`, and `--case -` read one complete payload from stdin. At most one such flag may request stdin in a command; conflicts fail before reading or dispatch. An empty stdin stream remains an explicit empty string, read errors remain errors, and payload bytes are not trimmed. Other string flags keep `-` as a literal value. MCP input and nested batch parameters never inherit CLI stdin expansion.

---

## Consequences

* The CLI entry point gains robust POSIX flag parsing, shorthand flags, and consistent subcommand lifecycle.
* Users can pipe or provide multiline source, body, and case payloads through stdin without placing those snippets in shell arguments or temporary payload files. Registry-derived CLI help identifies the supported flags; MCP schemas and descriptions remain unchanged.
* Dependency on `github.com/spf13/cobra` is introduced into `go.mod`.
* Custom argument splitting and flag set declarations in `main.go` are eliminated in favor of idiomatic Cobra command definitions.
* The CLI presents each successful operation through its registry formatter and
  each failure as `<command>: <error>`, producing the same standardized
  operation semantics as MCP without historical presentation exceptions.
