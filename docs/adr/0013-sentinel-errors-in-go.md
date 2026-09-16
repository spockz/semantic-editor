# ADR-0013: Sentinel Errors, Structured Records, and Location Invariants in Go

* **Status**: Accepted
* **Date**: 2026-09-16

---

## Context

In Go codebases, error handling without structured error types or sentinel errors leads to brittle patterns:

1. **Fragile String Matching**: Callers and test suites resort to `strings.Contains(err.Error(), "pattern")` to inspect failure modes, making tests sensitive to minor phrasing alterations or formatting tweaks.
2. **Eager String Formatting Penalties**: Creating errors via `fmt.Errorf("%w: %s in %s", ErrNotFound, sym, file)` immediately pays string allocation and formatting penalties, discarding structured fields and forcing callers into string parsing.
3. **Loss of Source Coordinates**: When code snippets or files fail to parse (e.g., AST syntax errors or JSON deserialization failures), omission of precise source coordinates hampers automated agent correction and developer diagnosis.
4. **Inconsistent Protocol Delivery**: CLI output requires standard compiler coordinate strings (`file.go:line:col`), whereas LSP and MCP protocols require structured, 0-indexed range objects.

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
       ErrSyntax              = errors.New("syntax error")
   )
   ```

2. **Structured Record Structs with `Unwrap()` for Contextual Errors**:
   When an error condition carries dynamic contextual variables (e.g., symbol identifiers, file paths, requested access modifiers), packages must define focused record structs embedding typed fields and the underlying sentinel:

   ```go
   type SymbolError struct {
       Op     string
       File   string
       Symbol string
       Err    error
   }

   func (e *SymbolError) Error() string {
       if e.File != "" {
           return fmt.Sprintf("%s %s in %s: %v", e.Op, e.Symbol, e.File, e.Err)
       }
       return fmt.Sprintf("%s %s: %v", e.Op, e.Symbol, e.Err)
   }

   func (e *SymbolError) Unwrap() error { return e.Err }
   ```

   * **Lazy Evaluation**: `Error() string` is evaluated strictly on demand when formatted for display; error creation copies only typed struct fields with zero string allocation.
   * **Sentinel Compatibility**: Implementing `Unwrap() error` guarantees automatic traversal by `errors.Is(err, targetErr)`.
   * **Programmatic Inspection**: Downstream callers extract typed fields directly using `errors.As(err, &targetStruct)` without string scraping.

3. **Mandatory Location Invariants for Coordinate-Bound Errors**:
   Any error arising from parsing, AST transformation, syntax validation, or JSON deserialization must capture source coordinates:
   * **Internal Representation**: Embed or record standard `token.Position` (`Filename`, `Offset`, `Line`, `Column`).
   * **CLI Delivery**: Render standard compiler location format using `pos.String()` (`"api/server.go:12:4: syntax error: ..."`).
   * **MCP Delivery**: Serialize coordinates into LSP-native 0-indexed ranges:

     ```json
     {
       "error": "syntax_error",
       "location": {
         "uri": "file:///path/to/api/server.go",
         "range": {
           "start": { "line": 11, "character": 3 },
           "end": { "line": 11, "character": 3 }
         }
       }
     }
     ```

4. **Context Wrapping via `%w` for Simple Propagation**:
   Functions that merely propagate external errors across architectural layers without introducing new structured domain fields must wrap errors using `fmt.Errorf("...: %w", err)`.

5. **Inspection Exclusively via `errors.Is` and `errors.As`**:
   All call sites, CLI/MCP error handlers, and test suites must inspect errors using `errors.Is(err, targetErr)` or `errors.As(err, targetType)`. Direct string matching on `err.Error()` is strictly prohibited for categorizing error conditions.

---

## Invariants

* Every discrete error condition emitted by a public package function must wrap a package-level sentinel error or standard library error.
* Errors capturing contextual variables must implement `Unwrap() error` returning the underlying sentinel error.
* Errors originating from syntax parsing or position-dependent AST operations must preserve `token.Position` coordinates.
* Tests asserting error identity must use `errors.Is(err, Err...)`. Tests asserting structured metadata must extract types using `errors.As(err, &record)`. Substring assertions (`strings.Contains`) are restricted to dynamic contextual payloads.
* MCP error responses must emit LSP-compatible 0-indexed ranges for location-bound errors.

---

## Consequences

* **Positive**:
  * Zero eager string formatting overhead along programmatic failure and verification paths.
  * Direct extraction of typed parameters (`Symbol`, `File`, `Line`, `Column`) by MCP servers and language tooling.
  * Uniform, compiler-standard CLI error output and LSP-standard MCP error payloads.
  * Full backward compatibility with `errors.Is(err, Err...)` checks across existing test suites.
* **Negative**:
  * Requires declaring small record structs for domain errors with dynamic variables.
