<!-- WHY: Records the native action and exact token-edit boundary for making one inferred Java local type explicit. -->
# ADR-0096: Java Native Inferred Local Type Expansion

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Make one Java method-local inferred type explicit through a direct JDT LS token edit. |
| description | Bind one unique unmodified local var to its signature-qualified method and fresh source revision; accept only the native denotable type replacement at the exact CST var token. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | java |

## Context

JDT LS 1.62.0 supplies enabled `Change type of '<local>' to '<type>'` actions of kind `refactor`. Live probes captured an eager selected-file `changes` edit replacing only `var` with `int` or an already imported `ArrayList<String>`. The configured session does not advertise code-action resolution support, and no command or edit callback was required. Native anonymous-type declarations offered no such action. Primitive and generic overload execution remained `int:3/array:1` before and after the captured edits.

The capability investigation is tracked in [RQ-0024](../research/RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md). This decision follows the Java delegation boundary in [ADR-0068](0068-java-hybrid-jdtls-and-embedded-cst-delegation.md).

## Decision

Expose `refactor_make_type_explicit` / `refactor-make-type-explicit` for Java. Require a source file, one unique local binding name, an exact signature-qualified method owner, and its current file `expected_revision`. Resolve the binding and identifier/type ranges with the Java CST, then request the native `refactor` action at its exact identifier selection. Accept exactly one action with the exact native title template and one eager, unannotated selected-file `changes` edit. Its replacement must equal the title's denotable explicit type, and its range must equal the CST `var` token.

Reuse the Java source-buffer, formatting, import-organization, diagnostics, and publication lifecycle. The receipt continues at the owner method because locals are not native document-symbol targets. [Make Static](0089-java-make-static-jdtls-action.md) and this action share one typed Java method-action receipt finalizer.

## Invariants

* Require the fresh selected-file revision before any native action; revalidate the full-file CST identifier selection before the request and token range before staging.
* Preserve initializer, declaration-name, and use bytes in the native edit. Normal formatting and verification remain part of the existing lifecycle.
* Refuse duplicate local names, non-var declarations, modifiers/annotations, fields, parameters, constructors, all loop bodies (`for`, enhanced `for`, `while`, and `do`), and lambda/nested-class binding scopes. Ordinary nested blocks are supported when the name remains unique.
* Refuse anonymous-class declarations and non-denotable replacement types, including any reserved `var` type component inside arrays, generics, or qualified types. Do not approximate the inferred type with a supertype or guess a replacement locally.
* Refuse missing, duplicate, disabled, command, lazy, null, annotated, resource, foreign-file, import-changing, wrong-token, and extra edits. There is no generic action fallback or arbitrary type-change capability.
* [Standalone](../../testdata/scripts/refactor_make_type_explicit_java.txtar) and [buffered](../../testdata/scripts/mcp_batch_make_type_explicit_java_overlay.txtar) CLI txtars, fake-protocol refusals, exact source comparisons, and an optional real-JDK compiler/runtime oracle establish the bounded contract. Deliberate primitive and generic widening changes the oracle output, proving its overload assertions are sensitive.

## Consequences

Agents can explicitly request this native type expansion without manual type inference. Valid declarations requiring imports, modified declarations, loop/lambda scopes, anonymous types, and other server edit shapes remain unsupported until separately captured and reviewed. Scala parity and other native Java actions remain separate gaps.
