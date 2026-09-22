<!-- Why this review brief lives in docs: it gives future agents a repeatable, evidence-based process for reviewing Go product quality, test adequacy, accepted ADR adherence, helper tools, and lifecycle behavior. -->

# Go Quality and Test Adequacy Review Instructions

Use this brief for a read-only, evidence-based review of the Go project. Do not implement fixes or modify source, tests, fixtures, ADRs, generated docs, or benchmark results.

## Scope and setup

1. Read AGENTS.md and relevant project context. Consult docs/adr/README.md and docs/research/README.md before opening individual records.
2. Work in a clean, separate Git worktree under .scratch/worktrees/. Record the baseline commit and worktree status. Do not disturb the user's checkout or its uncommitted work.
3. Review product and helper code separately:
   - Main product: root command and internal packages shipped as semedit.
   - Helper tools: cmd/docgen, tools/benchmark-harness, and development/build support.
   - Explain when a helper defect can undermine a product claim or benchmark result.
4. Inspect go.mod, Makefile, and configured linters before choosing checks. Follow the repository-required make check sequence before running targeted tests or verification. Because make check may format files or update generated artifacts, use a separate disposable worktree for execution. Record its commit and status before and after. Do not claim checks passed unless they completed successfully.

## Review dimensions

### Go code quality

Assess idiomatic Go, package/API cohesion, naming, comments, error wrapping and inspection, method/function size and responsibility, duplicated logic, readability, constants and unexplained literals, older APIs or patterns, portability, performance/resource bounds, and security/input boundaries. Distinguish long files from oversized functions. Explain when duplication appears intentional or safer than abstraction.

### Test adequacy: verify tests prove their stated behavior

Inventory unit, integration, property/fuzz, CLI, MCP, cross-language txtar, docgen, and benchmark tests. For each layer, record the command that runs it and the behavior it actually proves.

Trace every testdata/scripts/*.txtar scenario through its runner and build an assertion matrix with one row per command:

- command and expected success/failure;
- stdout/stderr assertions;
- exact file or workspace-state comparisons;
- semantic assertions and relevant diagnostics;
- external command/tool invocation assertions.

Keep these distinctions explicit: a successful exec proves process success; stdout/stderr matching proves only the matched text is present; cmp proves the compared file equals its golden file; a fixture's go test proves only what those fixture tests assert. Identify commands with no result assertion, modified files omitted from golden comparisons, and success cases that would still pass if the intended operation did nothing.

Inspect fake LSPs, language servers, and build tools. Check that they validate expected methods, arguments, ordering, flags, edits, and side effects. Check that unexpected or missing calls fail visibly rather than producing plausible success. For command-sensitive behavior, determine whether tests detect both a missing expected invocation and an unexpected extra invocation.

Evaluate oracle strength and failure sensitivity. For key behaviors, state the regression each test would catch and propose a controlled negative control or mutation that should make it fail. Do not mutate the review baseline. For benchmark failures, trace setup, observed tool/command activity, hidden oracle execution, result persistence, and publication filtering. Check that missing assertions, skipped oracles, incomplete runs, and oracle failures cannot be reported as verified success.

Review boundary cases: cancellation, timeout, malformed data, missing tools, permission failures, partial writes, and cleanup. A cancellation test must assert timely termination or a returned cancellation result, not merely call cancel.

### Accepted ADR adherence and consistency

Create a matrix covering every accepted ADR: ID/title, applicable invariant, implementation locations, test evidence, and conclusion (adheres, deviation, not applicable with reason, or unclear). Read each relevant ADR in full; the index is a discovery aid, not the complete contract. Cite evidence for both adherence and deviation.

Separately review ADR consistency. Identify contradictory requirements, duplicated normative decisions, overlapping scope, outdated premises, unresolved status, and drift among ADRs, code, capability metadata, and docs. Distinguish useful summaries/cross-references from repeated rules that can diverge. For each issue, cite the records and state which ADR should own the rule. Do not rewrite ADRs during review. Any later ADR or research-question change must update its index in the same change.

### Cancellation and resource lifecycle

Trace contexts, goroutines, subprocesses, pipes, LSP sessions, channels, locks, temporary files/directories, and file handles from creation through normal shutdown, cancellation, and error paths. Record ownership and cleanup. Check that cancellation reaches child work, goroutines and processes terminate promptly, close/wait behavior is sound, blocked I/O and races are addressed, cleanup errors are surfaced where meaningful, and failures do not leave misleading partial state. Link lifecycle claims to tests that assert termination or cleanup.

### Explicit Rust offset verification

Independently verify the suspected Rust multiline byte-offset issue:

- Trace the selected symbol position through rustCandidate and rustByteOffset.
- Use a concrete source with at least two lines and a symbol on a later line. Compare the returned offset with its absolute UTF-8 byte offset; include non-ASCII text where relevant.
- Compare newline-terminated and final-line paths, equivalent language backends, and existing tests.
- State confirmed, disproved, or uncertain, with code and test evidence. Claim runtime verification only if a test or reproducer was actually run.

## Reporting

Cite findings with file paths and line numbers. Separate confirmed defects from risks, style opportunities, and open questions. Prioritize by impact and confidence. For each finding give the observation, consequence, evidence, test coverage/gap, and recommended next step.

End with separate concise assessments for product and helper code, the highest-priority test gaps, ADR adherence and consistency, lifecycle findings, the Rust offset conclusion, and checks run/not run with reasons. Do not present review recommendations as implemented fixes.
