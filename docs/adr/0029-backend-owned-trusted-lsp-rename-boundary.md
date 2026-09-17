# ADR-0029: Backend-Owned Trusted LSP Rename Boundary

Status: **Accepted**  
Date: 2026-09-17

## Context

Rename requests cross a language-neutral ingress service, but lookup identity,
LSP protocol details, edit policy, and post-edit validation are language
specific. A generic workspace-edit or changeset engine would either encode
language assumptions in the service or weaken each backend's safety policy.

## Decision

The service selects a backend, checks its rename capability, and enforces the
request-scoped workspace trust boundary. It then delegates one complete rename
request to the backend. The backend owns lookup, ambiguity handling, LSP
transport, workspace-edit application, mutation, formatting/import policy, and
diagnostic postprocessing.

LSP transport remains mutation-free: it only starts sessions and exchanges
protocol messages. Each backend applies a strict, backend-specific
`WorkspaceEdit` policy and rejects unsupported edit shapes. There is no generic
workspace-edit or changeset engine in the service layer.

All backend file mutations use the atomic writer, including preservation of
existing target permissions and advancing modification times. The atomic writer
is therefore a prerequisite for trusted rename implementations.

## Invariants

* Service code contains no language switch for rename execution.
* Backend-owned rename is the sole boundary after capability and trust checks.
* LSP transport cannot mutate workspace files.
* Workspace edits are validated according to the owning backend's policy.
* Atomic writes preserve permissions and advance target modification times.

## Consequences

Go keeps its existing CLI/MCP behavior while owning its full rename pipeline.
Other backends can add trusted rename independently without changing service
or introducing a shared changeset abstraction. Until implemented, unsupported
backends continue to report `ErrUnsupportedOperation` truthfully.
