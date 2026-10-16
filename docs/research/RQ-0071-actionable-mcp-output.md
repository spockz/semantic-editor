<!-- Records why MCP output compaction must preserve actionable data across client presentation policies. -->

# RQ-0071: Actionable MCP Output and Model Visibility

Status: Resolved
Date: 2026-10-14

## Question

Can all tools avoid repeated formatter output while preserving enough information
for the next action? Is structured content visible to the model?

## Findings

The earlier outline fix reached main through `491b4d2a71c0`; its original commit
`664f5355cd19` was not merged directly. Its presentation changes recognized only
outline shapes. Rename still returned all existing diagnostics in `active`, and
other tools still appended their formatter output after the critical JSON.

Structured content is not inherently harness-only. As inspected on 2026-10-14:

- [VS Code's MCP adapter](https://github.com/microsoft/vscode/blob/main/src/vs/workbench/contrib/mcp/common/mcpLanguageModelToolContribution.ts)
  skips text when structured content exists and explicitly pushes its JSON to
  the assistant's model input. This is source verification, not a live test of
  the user's installed version.
- [Copilot CLI issue 4515](https://github.com/github/copilot-cli/issues/4515)
  reports both representations entering model context in 1.0.80, with a later
  reproduction reported in 1.0.89-7. This is a report, not an independent local
  reproduction or a guarantee for every client version.
- The [advertised MCP revision](https://modelcontextprotocol.io/specification/2025-06-18/server/tools#structured-content)
  recommends serialized JSON text for backward compatibility. It does not
  guarantee a particular model-context policy. Compact text is the explicit
  compatibility tradeoff recorded in ADR-0042.
- These observations do not establish whether Copilot desktop 1.1.29 forwards
  `content`, `structuredContent`, or both to its model.

## Shared policy and coverage

| Output family | Actionable `content` | Canonical detail in `structuredContent` |
| --- | --- | --- |
| Lookup and inspection | Resolution outcome, ambiguity choices, exact selectors and revisions; name the source field | Complete source and symbol records |
| References, declaration outlines, workspace outlines | Scope, counts, available cursor or target fields | Full location and discovery inventories |
| All registry mutations, including dependencies and undo | Application state, edit receipt, exact live continuations and guards, change counts | Diffs, read sets, and other bulk mutation evidence |
| Rename and known applied errors | Origin, proven final identity, remaining diagnostic count, report query and change summary | Complete diagnostic inventory and change arrays |
| Verification and build checks | Failure or clean status, coverage and diagnostic counts, report query and next action | Diagnostic arrays, hook output and logs |
| Batch | Final status, child outcome summary, partial publication evidence, final diagnostic counts and query | Ordered child detail, final and child diffs, verification arrays; no child continuation is live |
| Staging | Opaque stage and step IDs, limits, conflict or publication status, commit guards | Complete journal, diff, and verification detail |
| Diagnostic pages | Report ID, category, total, returned count, filter and `next_offset` | Requested `items` array |
| Generic and typed errors | Error, location, refusal, application state, ambiguity and recovery guard | Full typed failure evidence |
| Metrics and reload | Record count and `result.records`, or reload status | Metric records and raw telemetry |
| Feedback | Draft status, `result.markdown` for review, and submission limitation | Complete reviewable feedback artifact |

The shared presenter applies across registry tools; feedback, metrics, reload,
batch, and diagnostic paging need the same outcome rule despite their separate
transport paths. Small guards may appear in both channels, but bulk data is
not serialized twice. A text-only client receives the exact next-step guards
and a pointer to the structured field or page cursor. No absolute text limit
may cut a selector, revision, refusal, or partial-publication guard. Structured
payloads are not mutated by presentation.

## Verification

Regression coverage exercises the public shared presenter across output families,
preserves large read payloads in structured results, rejects duplicate formatter
JSON in text, and applies a 2 KiB budget to representative small outcomes; this
budget never truncates required guards. Rename projector tests use 500
existing warnings and prove that success and applied-error responses retain only
their count plus complete diagnostic changes. Advertised-schema tests migrate
with the shared rename contract. Existing MCP integration, staging, failure,
continuation, and CLI contracts run under `make check`.

`make check GO_SCRATCH=/Users/alessandro/sources/semantic-editor/.scratch/go`
passed after these changes, including actual MCP lookup, inspection, references,
rename, and verification calls and persisted rename-state assertions. The
separate active MCP `semantic_verify` hook failed to load `packageclosure.so`
because its linter and plugin use different Go package versions. That result
is recorded in `docs/SUBOPTIMAL_TOOLS.md`; it is not reported as clean semantic
verification. The pinned repository check passed independently.

These tests cover server output, not the final context construction of every
external client. Existing latency measurements remain historical and are not
reclassified by this output change. No diagnostic-analysis or gopls lifecycle
behavior changes in this work.

## Captured diagnostic follow-up

[ADR-0073](../adr/0073-captured-diagnostic-reports.md) adds scoped
remaining counts and copyable `semantic_diagnostics` queries to verified MCP
outcomes. The query pages captured snapshots and can select diagnostic category
and filter text. It performs no new analysis, so collecting details does not
repeat the earlier verification or restart gopls. Native verification and change
evidence remain available; unavailable or deferred collection never implies zero.

The production MCP integration test verifies two real remaining diagnostics,
one-item pages, a mutation's new report, stable older pages after later source
changes, filter companions, and one retained gopls process. Cache regressions
cover eviction, immutable inputs/pages, invalid pagination, and unknown versus
measured-zero categories. A CLI txtar contract advertises the query in both
profiles and verifies explicit failures without source mutation. Populated
50-item pages pass the enforced 10 ms read budget over 1,000 calls per fixture;
see the [measurements](RQ-0070-warm-mcp-operation-costs.md#captured-diagnostic-page-benchmark).
