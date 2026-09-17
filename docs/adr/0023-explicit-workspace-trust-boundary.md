# ADR-0023: Explicit Workspace Trust Boundary for External Tools

* **Status**: Accepted
* **Date**: 2026-09-17

## Context

Future language backends may need to invoke a language server, build server, compiler, or build command that reads project configuration or executes project-controlled code. Backend selection alone must not imply consent to those external tools. Existing Go lookup, rename, and verification behavior already has an established compatibility contract and is outside this prerequisite slice.

## Decision

1. The language-neutral backend contract carries an explicit, request-scoped `WorkspaceTrust` value. Its zero value is untrusted.
2. Trust is granted only for the canonical requested workspace root. Canonicalization resolves absolute paths and existing symlinks; a different root does not inherit consent.
3. Backends declare which operations require trust through capabilities. The service rejects those operations with a typed `WorkspaceTrustError` that unwraps to `ErrWorkspaceTrustRequired` and identifies the operation, language, and canonical workspace.
4. CLI and MCP accept `--trust-workspace` and `trust_workspace` as non-breaking input fields. They carry consent to future backends only; this slice does not discover or launch external tools.
5. Trust is never persisted implicitly. Callers must include it on each request, and no manifest, settings file, or process-global grant is written.

## Invariants

* Untrusted is the default for every new request.
* Consent is scoped to one canonical workspace root and cannot authorize another root.
* Existing Go common-operation behavior remains unchanged because the Go backend does not yet mark its operations as trust-required.
* This boundary does not add language adapters, discovery, process launching, build execution, or changeset behavior.

## Consequences

Future external-tool backends have one auditable ingress gate before they can perform project-aware operations. Existing callers remain source- and behavior-compatible, while callers that opt in can pass explicit consent without creating persistent security state.
