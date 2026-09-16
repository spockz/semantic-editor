# ADR-0013: Sentinel Errors and Error Wrapping Standards in Go

* **Status**: Accepted
* **Date**: 2026-09-16

---

## Context

In Go codebases, error handling without structured error types or sentinel errors leads to brittle patterns:

1. **Fragile String Matching**: Callers and test suites resort to `strings.Contains(err.Error(), "pattern")` to inspect failure modes, making tests sensitive to minor phrasing alterations or formatting tweaks.
2. **Loss of Diagnostic Identity**: When internal functions construct ad-hoc errors using `fmt.Errorf("arbitrary text: %v", err)` without wrapping standard sentinels, upper layers (CLI, MCP transport, pipeline) cannot programmatically classify failures into user-actionable categories (e.g., distinguishing a symbol resolution failure from a compiler syntax error).
3. **Inconsistent API Contracts**: External consumers and tests have no compiler-verifiable error contract to program against.

---

## Decision

1. **Mandatory Package-Level Sentinel Errors**:
   Every discrete domain, syntax, validation, or operational failure condition must be declared as an exported package-level sentinel error prefixed with `Err`:

   ```go
   var (
       ErrNotFound            = errors.New("symbol not found")
       ErrAmbiguous           = errors.New("ambiguous symbol")
       ErrUnsupportedModifier = errors.New("unsupported access modifier")
       ErrVisibilityMismatch  = errors.New("visibility mismatch")
       ErrInvalidSnippet      = errors.New("invalid code snippet")
   )
   ```

2. **Context Wrapping via `%w`**:
   Whenever adding contextual details (e.g., file paths, identifier names, line numbers), functions must wrap the appropriate sentinel error using `fmt.Errorf`:

   ```go
   if err := ValidateAccess(backend, mod, name); err != nil {
       return fmt.Errorf("validate access modifier %q for %s: %w", mod, name, err)
   }
   ```

   Underlying system or standard library errors must also be wrapped with `%w` rather than formatted with `%v` or `%s`.

3. **Inspection Exclusively via `errors.Is` and `errors.As`**:
   All call sites, CLI/MCP error handlers, and test suites must inspect errors using `errors.Is(err, targetErr)` or `errors.As(err, targetType)`. Direct string matching on `err.Error()` is strictly prohibited for categorizing known error conditions.

---

## Invariants

* Every discrete, expected error condition emitted by a public package function must wrap a package-level sentinel error or an established standard library error.
* Tests asserting specific error conditions must use `errors.Is(err, Err...)`. Substring inspection (`strings.Contains`) is restricted to asserting dynamic contextual message payloads, never error identity.
* Sentinel errors must be declared as package-level variables using `errors.New(...)`.

---

## Consequences

* **Positive**:
  * Programmatic classification of errors across CLI, MCP, and pipeline boundaries.
  * Resilient, refactor-safe unit and integration test assertions.
  * Clear, discoverable package error contracts documented in Go package APIs.
* **Negative**:
  * Requires authoring and maintaining package-level sentinel variables across domain packages.
