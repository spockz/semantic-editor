<!-- Tracks the evidence needed before Scala gains native Metals refactoring capabilities. -->
# RQ-0073: Metals Refactoring Capability Contract

* **Status**: Open
* **Category**: Product & Architecture
* **Date**: 2026-10-19

## Context

Scala exposes trusted lookup and outline plus bounded `refactor_extract_value` and `refactor_extract_callable` slices. The value slice accepts one exact expression-bodied `Int` call shape; callable extraction accepts guarded contiguous direct calls to the same callee and creates a method-local `Unit` helper. The refactoring handover identifies inline and other actions as candidates, but these cannot be inferred from shared operation names or from the existence of a Metals code-action class. A Scala mutation must capture native edits against the current source buffer, preserve evaluation order, honor the requested target name, and keep build import disabled.

## Findings

* Metals `ExtractValueCodeAction` returns direct same-file text edits and generates a fresh local name. Its broad `refactor.extract` kind is shared with other actions; the captured action has no distinguishing data, so the bounded operation identifies the exact title pattern `Extract [expression] as value` and validates the exact same-file edit shape.
* Extracting an expression can hoist it from a branch, loop condition, or later call argument. Syntax-based action availability alone does not prove preserved evaluation order or by-name behavior.
* The pinned Metals action provider queries supported compiler actions, which loads a presentation compiler, including its standalone fallback. Disabling BSP and build import does not establish that action discovery is offline or compiler-free.
* Metals inline-value is command-based and requires a server `workspace/applyEdit` collector. The persistent Scala session rejects server-initiated apply-edit requests. ADR-0081 defines an operation-scoped disposable capture session for the bounded inline-value slice.
* A live probe using the installed Metals 1.6.9, Java 25, a trusted scratch project, and the current no-build/BSP-disabled settings returned an empty `textDocument/codeAction` list for the bounded inline-value source. It did not reach command execution or `workspace/applyEdit`; the command/callback ordering and edit envelope remain unverified against the configured runtime. The current capability must remain gated until a live project setup makes the action available and the complete transaction is captured.
* Metals 1.6.8 was run with the production section-aware `workspace/configuration` response and rejection behavior for unsupported server requests. For `object Thing { def run(): Int = consume(1 + 2) }`, it returned the exact direct action `Extract \`1 + 2\` as value`, kind`refactor.extract`, without action data, with three same-file edits: replace the expression by`newValue`, replace the space after`=` with a block opening and `val newValue`, and insert the method closing brace. The captured candidate accepted`prepareRename` for the generated use and returned exactly two version-2 rename edits, one for the declaration and one for the use.
* The current value-extraction implementation is an experimental prototype. It captures a genuine Metals action, but a regex preflight restricts it to one object containing exactly two methods and one direct call from a zero-argument expression-bodied `Int` method into a strict identity `Int` callee. It refuses other expressions, overloaded or non-Int callees, by-name arguments, ambiguous owner methods, and source shapes outside that prototype. Its arithmetic grammar only allows Int-range literals and `+`, `-`, or `*`, keeping the inserted local's inferred type equal to the original argument type. Metals action discovery still relies on the configured presentation compiler; no Scala compiler diagnostics are reported.
* `extract_variable_scala.txtar` exercises the full CLI capture and rename transaction, compares the complete output file, and confirms a representative unsupported selection leaves source bytes unchanged.
* Metals 1.6.9 returns a lazy method-local `ExtractMethodCodeAction` for one or more contiguous direct-call statements to the same callee. Resolving a multi-line range produces the same-file replacement and local helper insertion, with the returned helper body containing the selected statements in source order. The object-level alternative inserts a parameter and is rejected. A version-2 `prepareRename`/`rename` transaction edits exactly the helper definition and call. The bounded preflight accepts numeric literal arithmetic or string literal arguments and refuses shadowed callees, control flow, captured values, non-`Unit` helpers, and foreign-file edits. The implementation validates both native edits; multi-call extraction emits a braced helper body so Scala 2 and Scala 3 preserve the entire sequence inside the helper. The CLI txtar compares the complete output for both single-call and three-call selections and checks unchanged sources on refusal.

Sources:

* [Pinned Metals ExtractValue implementation](https://github.com/scalameta/metals/blob/9bfd8010e0f332403d0c8b212dfbd03c4fcba705/metals/src/main/scala/scala/meta/internal/metals/codeactions/ExtractValueCodeAction.scala)
* [Pinned Metals code-action provider](https://github.com/scalameta/metals/blob/9bfd8010e0f332403d0c8b212dfbd03c4fcba705/metals/src/main/scala/scala/meta/internal/metals/codeactions/CodeActionProvider.scala)
* [Pinned Metals compiler lifecycle](https://github.com/scalameta/metals/blob/9bfd8010e0f332403d0c8b212dfbd03c4fcba705/metals/src/main/scala/scala/meta/internal/metals/Compilers.scala)
* [Pinned Metals inline-value implementation](https://github.com/scalameta/metals/blob/9bfd8010e0f332403d0c8b212dfbd03c4fcba705/metals/src/main/scala/scala/meta/internal/metals/codeactions/InlineValueCodeAction.scala)

## Questions

1. Can extraction remain usable with only preprovisioned compiler artifacts and strict network-denied startup across supported Metals distributions?
2. Which additional Scala syntax contexts can preserve evaluation order and type after extraction, and which must remain refused?
3. What operation-scoped server-request and edit-capture contract is needed for command-based inline-value actions?

## Acceptance Criteria

* For each additional operation, verify its action identity, range behavior, source-version semantics, compiler lifecycle, and workspace-edit shapes against the configured Metals distribution.
* Keep runtime capabilities, an operation-specific language contract, documentation, fake-session tests, and CLI txtars in the same change as each Scala-operation pair.
* A CLI txtar must assert the complete changed file and observable evaluation order or count, plus unchanged bytes for representative refusals.
* Advertise only independently verified operation slices. The bounded inline-value slice requires a disposable session and exact source/edit proof; do not generalize it to other Scala syntax or advertise signature changes, moves, or contract generation until each exact operation contract is independently verified.

## Next Steps

* Verify Metals action discovery in an offline fixture with a preinstalled compiler artifact cache and the existing no-build settings.
* Verify that extract-value and extract-callable action discovery work with preprovisioned compiler artifacts and strict network-denied startup.
* Verify extract-value, extract-callable, and inline-value with preprovisioned compiler artifacts and strict network-denied startup. The inline-value probe must return the command and captured edit with the production handler before the capability can be advertised.
* Investigate additional Scala refactorings independently; the scoped inline-value collector does not establish support for other Metals commands.
