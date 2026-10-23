<!-- WHY: Records the revision-bound, bounded Go proof for replacing multiple selected direct calls to one literal-returning free function without widening other backend contracts. -->
# ADR-0118: Bounded Go Multi-Site Callable Inlining

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Inline an explicitly selected set of Go calls to one bounded free function. |
| description | Bind source-ordered exact call occurrences to the caller revision and replace only eligible direct-owner return or one-target assignment calls after validating each changed site in the typed package overlay. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | go |

## Context

ADR-0089 supports one selected Go call to a bounded literal-returning callable and keeps Java and Rust behind their own backend contracts. Repeated calls to the same simple free function are common, but selecting them by spelling alone is ambiguous and replacing only some occurrences must not accidentally include nested or unsupported contexts. Each occurrence must resolve to the same callee, belong to a supported evaluation position, and remain type-correct after replacement.

## Decision

Extend `refactor_inline_callable` for Go with optional `call_site.occurrence_indices`. The indices are zero-based, strictly increasing, unique, and refer to exact `call_site.discriminator` matches in source order within the selected caller function. Multi-site selection requires the current `call_site.expected_revision`, selects at least two matches, and is available only for same-package free functions. Without indices, the existing single-call contract remains in force.

The callee must be a non-generic, non-variadic zero-argument free function with one direct literal return of a predeclared basic result type. Every selected occurrence must resolve to that exact callee and be either a complete direct-owner return expression or the complete right-hand call in a direct-owner one-target `:=` or `=` assignment. Calls in nested blocks, expressions, branches, loops, closures, or any other context refuse. Keep the callee declaration in place.

Plan all selected replacements against the captured typed package overlay, reject stale or invalid selectors, duplicate/overlapping spans, and comment/directive overlap, then apply the disjoint edits in descending source order. Validate the resulting package and each relocated replacement's conversion binding and type before one atomic publication. Do not broaden Java, Rust, or Go method-call behavior; their multi-site selectors refuse.

## Invariants

* Occurrence indices are revision-bound and are interpreted over exact discriminator matches in source order within one selected owner; malformed, duplicate, nonascending, negative, out-of-range, or stale selections refuse without writing.
* Every selected call resolves by compiler identity to the same eligible free function. The caller and callee remain within the same active package.
* Each selected call is the complete expression of one direct-owner return or one direct-owner single-target short declaration/simple assignment. Unsupported contexts and overlapping edits refuse.
* Candidate replacements are checked in the typed package overlay, including their conversion target, before a single publication; an error at any selected site leaves all source files unchanged.
* Existing one-call Go, Java, and Rust behavior remains under its own capability contract. Java and Rust reject `occurrence_indices`; no multi-site capability is claimed for them.
* Standalone and buffered-batch CLI txtars cover selected subsets, runtime behavior, invalid and stale selectors, unsupported contexts, and unchanged-source refusal.

## Consequences

Go callers can inline a deliberate subset of matching literal-return free-function calls in supported direct-owner contexts with one revision-checked request. Multi-site method inlining, nested or control-flow contexts, parameterized or generic callees, cross-package calls, and cross-language multi-site behavior remain unsupported until their separate proof and CLI evidence are established.
