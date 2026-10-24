<!-- WHY: Defines the narrow Java make-static contract so native action selection and workspace edit safety stay explicit. -->
# ADR-0089: Java Make Static Through JDT LS

* **Status**: Accepted
* **Date**: 2026-10-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Make one exactly resolved Java instance method static through JDT LS's native action. |
| description | Register one canonical `refactor_make_static` operation that resolves a selected Java method, selects exactly one enabled JDT LS `Make Static` action of kind `refactor.inline`, captures its workspace edit, and applies it through the existing verified Java source-buffer lifecycle. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | java |

## Context

JDT LS 1.62.0 returned a direct `WorkspaceEdit` for the `Make Static` action of kind `refactor.inline` on a resolved Java method. The edit adds an explicit receiver parameter to the now-static method and may update call sites in other files. A Java 25 harness compiled the edited project and preserved its behavior. The Java backend already validates native code actions, captures workspace edits, applies them through the source buffer, formats changed files, and checks diagnostics. A Java-only operation can use that lifecycle without adding fields to the shared `CodeActionRequest` contract.

## Decision

Add one canonical `refactor_make_static` operation for Java. Require a trusted workspace, a selected `.java` file, and an exact signature-qualified selector resolving once to a method in that file. Before requesting the action, require that the method's simple name resolves uniquely in the selected file; overloads and same-name methods in another class in that file are refused before any edit. Request the method's selected range from JDT LS with `refactor.inline` context and accept only one enabled action whose kind is `refactor.inline` and title is exactly `Make Static`. After applying the captured edit, inspect the selected file again by the unique method name and publish that post-edit identity as the receipt continuation.

Pass the captured action edit through the existing Java workspace edit validator, staging, formatting, import organization, and diagnostic lifecycle. Support buffered batches through the same source-buffer path. Record the selected method as the post-edit continuation. Do not add a manual modifier rewrite, action-title aliases, or fallback action selection.

## Invariants

* The selector must resolve to one method with an unchanged source revision in the selected file.
* A missing, disabled, duplicate, differently titled, or differently categorized action is refused before publication.
* Every returned file edit is validated against its captured preimage and remains within the trusted workspace.
* Standalone and buffered edits use the existing atomic publication and Java document synchronization lifecycle.
* The operation remains Java-only and uses one canonical registry, CLI, MCP, and receipt identity.

## Consequences

Agents can express the semantic intent directly and rely on JDT LS for the modifier transformation. Support is deliberately tied to the probed action identity; JDT LS versions that omit or rename the action refuse until separately verified. Other language backends gain no capability from this decision.
