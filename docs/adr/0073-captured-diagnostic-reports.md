<!-- Defines captured diagnostic evidence ownership so file-move callers can inspect details without repeating compiler or language-server analysis. -->
# ADR-0073: Captured Diagnostic Reports

## Status

Accepted.

## Date

2026-10-16

## Context

File relocation can affect many callers and test variants. MCP text needs to
retain publication outcomes, changed paths, refusals, and valid continuations
without repeating the complete diff or diagnostic inventory. Reading evidence
must not trigger another verification or shift pages after later edits.

## Decision

Reuse completed native diagnostic captures for immutable, server-owned reports.
Verified edit, rename, verify, batch, stage-commit, and undo outcomes attach
`diagnostic_report` metadata with captured counts, status, scope or coverage when
known, and a copyable `semantic_diagnostics` query. Deferred and unavailable
diagnostics omit unknown counts and remain explicit.

The MCP-only read utility accepts `report_id`, `category` (`active`, `introduced`,
`resolved`, or `suggestions`), `offset`, `limit`, and optional case-insensitive
`contains`. Default pages contain 50 items and at most 100. `total` is the
filtered category count; `next_offset` exists only when another page remains.
An offset equal to the count yields an empty final page; larger offsets refuse.
Unknown, expired, and uncaptured categories are explicit errors.

Retain at most 16 reports and 16 MiB of encoded report evidence, evicting oldest
reports first. Report IDs are opaque and local to the originating server.
Oversized evidence returns an unavailable query state. This bounded subsystem
adapts the unmerged warm-MCP investigation without importing its unrelated
performance, graph-cache, or notification changes.

## Invariants

* Counts and pages describe the same captured workspace revision.
* Pagination performs no compiler, language-server, formatter, or hook call.
* Snapshots and returned page memory are independent of later caller mutation.
* Missing baselines do not invent introduced or resolved counts.
* Structured results retain detailed evidence. File-move text retains actionable
  publication state, paths, refusals, continuations, and captured counts.
* Query metadata points only to an advertised utility with copyable arguments.

## Consequences

Callers can inspect existing evidence without restarting gopls or collecting a
second diagnostic inventory. Expiration requires a newer report or an explicit
verification request. The cache bounds retained evidence, not the size of every
structured operation result or project-hook log.
