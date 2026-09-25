# RQ-0036: MCP Feedback Report Shape and Delivery

* **Status**: Resolved
* **Category**: Developer Feedback & Privacy
* **Last Updated**: 2026-09-25

## Question

How should the feedback utility collect enough detail to reproduce and fix
friction while protecting project information and avoiding unexpected side
effects?

## Existing Evidence

`docs/SUBOPTIMAL_TOOLS.md` captures the tool or operation, target context,
observed behavior, manual workaround, root cause or missing invariant, and a
tracking fix. `docs/MANUAL_EDITS.md` separately records manual edits and the
capability gap they expose. A useful report therefore needs the attempted task,
exact command and parameters, actual result, why that result was unexpected,
and manual touch-ups. Context and a proposed cause or fix help maintainers but
may not be known by the reporter.

## Options Considered

1. **A semantic operation in the central registry.** This would provide CLI,
   MCP, documentation, and batch wiring from one definition. Rejected because
   feedback reporting does not edit a workspace or resolve a language backend,
   and batching reports with source mutations has no useful semantics.
2. **A separate MCP tool that returns a structured draft.** Chosen. It matches
   the todo's MCP use case, keeps the required fields visible in `tools/list`,
   and can produce both human-readable Markdown and machine-readable data for
   the user to review and manually post as a GitHub issue.
3. **Append each report to the active workspace log.** Rejected because
   `semedit` can run inside unrelated user projects; an implicit docs write
   would be surprising and may persist sensitive content in version control.
4. **Submit reports automatically to a hosted endpoint or issue tracker.**
   Deferred because no destination or authentication model is configured.
   Automatic submission would create an external side effect and require a
   separate consent and data-handling decision. Manual posting by the user
   after review is the intended workflow.

## Resolution

The MCP tool requires the five core details: user intent, interface and command
with parameters, observed result, expectation gap, and manual touch-ups. The
interface identifies MCP, CLI, or other use. Target context, suspected cause,
and suggested fix are optional and map to the existing logs. The result is a
Markdown and structured-content draft with a generated date. The user reviews
accuracy and sensitive content, then may post the Markdown as a GitHub issue.
The tool neither saves nor submits it. For this repository, the established
`SUBOPTIMAL_TOOLS.md` and `MANUAL_EDITS.md` bookkeeping continues separately;
the feedback draft does not replace those records.

The tool description tells the model not to include source code, credentials,
personal or customer data, private paths, or proprietary information and
intellectual property. Public project-specific details in open-source projects
are generally okay. When project visibility or a detail's sensitivity is
unclear, the model asks the user and omits that detail until approved. This
instruction guides the model; it is not a content classifier or a substitute
for user review.

The accepted interface and privacy contract is recorded in
[ADR-0045](../adr/0045-structured-mcp-feedback-reports.md).
