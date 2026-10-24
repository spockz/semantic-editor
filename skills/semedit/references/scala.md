# Scala Semantic Refactoring Guide

Failed selected-file lookups may include native-validated `suggestions`. Retry
the same lookup with one candidate's `arguments`; preserve its file and qualified
selector. Normal server configuration and workspace trust remain required.
Suggestions do not edit source or retry automatically. See [configuration and limits](../../../docs/symbol-suggestions.md).

MCP results follow the [shared output policy](../SKILL.md#copy-typed-results-and-respect-publication).
Use typed continuations with their exact revisions, and retain failure and
publication state when choosing follow-up calls. Structured results can be
model-visible; text keeps exact next-step guards and names structured fields
without repeating source, inventories, diagnostic pages, diffs, or logs.
For diagnostic details, copy the response's `diagnostic_report.arguments` into
`semantic_diagnostics`, choose a category/filter and page size, then follow
`next_offset`. Pages reuse the captured report; a deferred/unavailable count
does not indicate a clean workspace. Use verification only for a fresh report.

This reference provides workflows for inspecting and refactoring Scala source files with `semedit` primitives. Read this document only when working with Scala source files.

---

Compact lookup and outline listings use workspace-relative paths and one-based
inclusive declaration lines. Signatures describe declarations; explicit native
selectors remain separate and must be copied verbatim. Use ordinary file reading
for disk-backed results. A navigation result does not imply that this backend
supports symbol inspection or mutation.

## 1. Macro Refactorings in Scala

`semantic_lookup` and `semantic_outline` provide read operations. `refactor_extract_value` and `refactor_extract_callable` capture native Metals edits, with guarded preflight restricting each operation to the exact source shapes below. In a batch, supported edits stay in the shared working overlay and a following `semantic_lookup` reads that overlay before publication. Scala backend operations require workspace trust, a pinned preinstalled Metals language server, and Java 21 or newer. This reference defines the supported boundaries and verification practices.

### Bounded extract-value action

Use `refactor_extract_value` only when the selected expression is integer-literal arithmetic using `+`, `-`, or `*`, in a file containing exactly one object and two methods: a zero-argument expression-bodied `Int` owner that directly calls a strict `Int` identity callee with the expression as its sole argument, and a callee whose body returns its parameter. Extra imports or declarations, overloads, by-name parameters, and other callee bodies refuse. The requested `symbol` names the new local; `in_function` names the owner method and `discriminator` is the exact expression. The implementation captures one direct three-edit Metals action, stages its candidate in the LSP session, validates a local `prepareRename`/`rename` pair, then stages the final source through the shared buffer.

Do not route calls, named values, by-name arguments, overloads, implicit conversions, branches, loops, non-`Int` contexts, or expressions using division/remainder to this operation. It does not provide Scala compiler diagnostics. After applying the refactoring, compile with the project's configured build tool when project verification is needed; semedit does not start build tools for this operation.

### Bounded extract-callable action

Use `refactor_extract_callable` with `kind: "function"`, a 1-based contiguous `start_line`/`end_line` pair, `in_function` naming the owning method, and `symbol` naming the new helper. The bounded contract accepts one or more complete adjacent direct calls to the same unshadowed callee, with numeric literal arithmetic or string literal arguments. Metals must return the method-local `ExtractMethodCodeAction`; object-level extraction actions, other action identities, extra parameters, non-`Unit` helpers, control flow, non-literal arguments, and edits outside the selected file are refused. The backend resolves the lazy action, verifies its exact replacement and insertion ranges and helper body, preserves all selected calls in order, and emits braces around the helper body when extracting multiple calls. It stages the candidate in the LSP session, validates a versioned rename for the helper declaration and call, then publishes through the shared source buffer.

This operation does not infer live-ins or live-outs and does not provide Scala compiler diagnostics. Keep a selected call's evaluation point unchanged. After extraction, compile with the project's configured build tool when project verification is needed; semedit does not start build tools for this operation.

The owner-boundary scanner fails closed when the method body contains interpolation strings, apostrophe syntax, backquoted identifiers, or XML literals because those forms need Scala lexical parsing to count braces safely.

### Inline-value is not supported

Do not route Scala `refactor_inline_value` requests. The configured Metals runtime returned no action for the bounded source during the live probe, so the operation remains outside the public capability contract until a production transaction is captured. See [RQ-0073](../../../docs/research/RQ-0073-metals-refactoring-capability-contract.md).

Mutable or explicitly typed locals, negative or computed initializers, extra methods, overloads, shadowed names, extra or non-call uses, mismatched parameter types, foreign-file edits, duplicate callbacks, and edits that differ from the expected postimage are refused. The operation does not provide Scala compiler diagnostics; the no-build policy remains in force.

---

## 2. Scala Refactoring Patterns

### Pattern A: Case Class to Class Migration

When a Scala `case class` evolves to require mutable internal state, private constructor logic, or specialized caching:

1. **Inspection**:
   - Use `semantic_lookup` to locate companion objects, apply methods, and copy calls.
2. **Structural Transformation**:
   - Remove `case` modifier from class header.
   - Implement explicit constructor, accessor methods, `canEqual`, `equals`, `hashCode`, and `toString` if equality semantics must be preserved.
   - Add explicit factory methods in the companion object if `apply` instantiation is expected by callers.
3. **Verification**:
   - Execute `sbt compile` or `mill __.compile` to confirm zero broken call sites.

---

### Pattern B: Extract Trait for Component Decoupling

When abstracting concrete service implementations:

1. **Define Trait**:
   - Extract public method signatures into a new `trait`.
2. **Extend Trait**:
   - Update concrete class to extend the trait (`class ServiceImpl extends ServiceTrait`).
3. **Caller Generalization**:
   - Update dependent constructors and method parameters to accept the trait type.
4. **Verification**:
   - Verify compiler clean state.
