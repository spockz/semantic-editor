# ADR-0045: Structured MCP Feedback Reports

Status: Accepted
Date: 2026-09-25

## Context

`docs/SUBOPTIMAL_TOOLS.md` records command friction, including the affected
operation and context, observed behavior, manual workaround, suspected cause,
and candidate fix. The current manual process does not consistently capture
what the user intended, the exact command parameters, or why the result missed
expectations. An agent also needs clear disclosure guidance before preparing a
report that may be shared with maintainers.

The utility has no configured feedback endpoint or storage destination. Adding
an implicit write to the active workspace could modify an unrelated project;
a remote submission would require a chosen destination, authentication, and an
explicit sharing decision. The intended workflow is to return a draft for the
user to review and, if appropriate, post manually as a GitHub issue.

## Decision

Expose a standalone MCP utility named `report_feedback`, outside the semantic
operation registry. It is not a source transformation and does not belong in
semantic batching or the language-specific CLI operation set.

Require the report to include the user's intent, interface, command name,
command parameters, observed result, expectation gap, and manual follow-up.
The manual follow-up is required; callers write `None` when no touch-up was
needed. Target context, suspected cause, and suggested fix are optional so the
report aligns with existing dogfooding records without asking the model to
invent details.

Return both a Markdown draft and the same report as structured content with a
generated date. The Markdown is suitable as a proposed issue body. The user
reviews it for accuracy and sensitive content before manually posting it to a
GitHub issue. The tool does not write files or post the report. For this
repository, the existing dogfooding ledgers remain the project bookkeeping and
are maintained separately from issue submission. The schema description
instructs the model to omit source code, credentials,
personal and customer data, private paths, and proprietary information or
intellectual property. Public project details for open-source projects are
generally okay to include. If the model is unsure whether a detail is safe or
whether the project is open source, it asks the user and leaves that detail out
until approved.

## Invariants

- The seven core report fields are explicit and required by the MCP input
  schema; optional context, suspected cause, and suggested fix stay optional.
- `parameters` preserves the command arguments as a JSON object and may be
  empty when the command took no parameters.
- A successful result identifies itself as a draft and conforms to an MCP
  output schema with the standard metrics envelope.
- The tool never persists or submits feedback and never claims that the report
  was submitted.
- The returned report is a review draft for possible manual GitHub issue
  submission; it does not replace this repository's dogfooding ledgers.
- Tool guidance treats public open-source details as generally shareable while
  excluding secrets and asking the user when disclosure safety is uncertain.

## Consequences

Agents can assemble consistent, actionable reports that map to existing
feedback records while users retain control of what they share. Users can
review and manually post a draft to the GitHub issue tracker. In this
repository, maintainers continue recording dogfooding details in
`docs/SUBOPTIMAL_TOOLS.md` and `docs/MANUAL_EDITS.md` as applicable. Automatic
submission or persistence requires a separate destination and consent
decision.
