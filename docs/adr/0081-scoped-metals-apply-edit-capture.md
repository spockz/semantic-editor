<!-- Scope command-based Scala inline edits to one disposable Metals session. -->
# ADR-0081: Scoped Metals Apply-Edit Capture for Scala Inline-Value

* **Status**: Proposed
* **Date**: 2026-10-20

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Keep a future command-based inline-value action in a disposable session; do not advertise it until the configured runtime returns the action and the production edit-capture transaction is verified. |
| description | The pinned Metals source implements inline-value as a `refactor.inline` command. The exact configured Metals 1.6.9 probe, however, returned an empty `textDocument/codeAction` list for the bounded source under the current no-build/BSP-disabled settings. The command response and apply-edit envelope therefore remain unverified for this runtime. If a production action becomes available, its server-initiated `workspace/applyEdit` request must be scoped to one disposable session because the command response does not identify or await that callback. Scala's normal session rejects server apply-edit requests. |
| role | contract |
| scope | source-code |
| area | language-server |
| languages | scala |

## Context

The pinned Metals source implements inline-value as a `refactor.inline` command. The exact configured Metals 1.6.9 probe, however, returned an empty `textDocument/codeAction` list for the bounded source under the current no-build/BSP-disabled settings. The command response and apply-edit envelope therefore remain unverified for this runtime. If a production action becomes available, its server-initiated `workspace/applyEdit` request must be scoped to one disposable session because the command response does not identify or await that callback. Scala's normal session rejects server apply-edit requests.

## Decision

If enabled after a successful live trace, Scala inline-value will run through an operation-scoped disposable Metals session that advertises `workspace.applyEdit`. The prototype captures exactly one callback while the inline command is active, waits for both command completion and edit capture, then revokes capture and closes the process before staging. Until that trace succeeds, Scala inline-value is not part of the public capability contract.

The initial capability is limited to one untyped immutable decimal `Int` local in a zero-argument `Unit` owner, used only as direct arguments of one same-object callee with matching strict `Int` parameters. The backend accepts only the unversioned same-file edit envelope, validates each exact declaration/use span, independently derives the complete postimage, and refuses timeouts, missing or duplicate callbacks, or edits outside the captured operation.

## Invariants

* Persistent Scala sessions do not enable or handle `workspace/applyEdit`.
* A disposable inline session permits one active edit-capture callback and no other mutation method.
* The callback is revoked and in-flight handling completes before the session is closed and source is staged.
* The command result and captured edit must both succeed; failure publishes no source change.
* The captured workspace edit must match one declaration deletion and every validated use replacement in the selected file, and its complete postimage must match the independently derived result.
* The Scala capability matrix, operation contract, language reference, and CLI txtars advertise only independently verified behavior.

## Consequences

The disposable capture design can keep uncorrelated server edits isolated without relaxing the persistent server-request policy. The capability remains unexposed until Metals returns the expected action and the production handler records its command and callback behavior.
