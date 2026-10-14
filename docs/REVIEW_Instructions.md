<!-- Why this review brief lives in docs: it gives future agents a repeatable, evidence-based process for reviewing product quality, test adequacy, accepted ADR adherence, helper tools, and multi-protocol lifecycle behavior. -->

# Codebase Quality, Test Adequacy, and Multi-Protocol Review Instructions

Use this brief for a read-only, evidence-based review of the project. Do not implement fixes or modify source, tests, fixtures, ADRs, generated docs, or benchmark results.

## Scope and setup

1. Read AGENTS.md and relevant project context. Consult docs/adr/README.md and docs/research/README.md before opening individual records.
2. Work in a clean, separate Git worktree under .scratch/worktrees/. Record the baseline commit and worktree status. Do not disturb the user's checkout or its uncommitted work.
3. Review product, backend integrations, and helper code:
   * Main product and language backends: core engine, CLI commands, MCP server, and supported language providers (native AST engines, Tree-sitter CST locators, and external LS or compiler delegators).
   * Helper tools: cmd/docgen, tools/benchmark-harness, and development/build support.
   * Explain when a helper or language-delegator defect can undermine core product claims or benchmark validity.
4. Inspect go.mod, Makefile, and configured linters before choosing checks. Follow the repository-required make check sequence before running targeted tests or verification. Because make check may format files or update generated artifacts, use a separate disposable worktree for execution. Record its commit and status before and after. Do not claim checks passed unless they completed successfully.

## Review dimensions

### Code quality and architectural idioms

Assess idiomatic language usage, package/API cohesion, naming, comments, error wrapping and inspection, method/function size and responsibility, duplicated logic, readability, constants and unexplained literals, older APIs or patterns, portability, performance/resource bounds, and security/input boundaries. Distinguish long files from oversized functions. Explain when duplication appears intentional or safer than abstraction. Verify clean decoupling between ingress protocols, core dispatch, and language-specific backends.

### Multi-protocol presentation, schemas, and payload budgets

Inspect operations across every advertised delivery protocol (CLI, MCP JSON-RPC, and structured automation APIs):

* **Schema Conformance**: Verify that runtime parameter validation strictly reflects advertised JSON schemas, and that returned output structures match their documented schema contracts.
* **Envelope Deduplication & Projection Budgets**: For tools returning rich or bulk results (such as directory or declaration outlines), verify that human-readable text representations act as concise, high-level summaries rather than duplicating full data payloads already delivered in structured fields.
* **Continuation & Receipt Consistency**: Verify that selector fields, revisions, and continuation metadata emitted by preceding operations match the parameter requirements of downstream consuming operations.
* **Catalog Profiling**: Check that tool-profile filters (e.g. base vs full vs mutations-only) correctly expose or omit operations without leaving broken cross-references or dangling continuations.

### Polyglot workspaces and hierarchical scope resolution

Inspect workspace discovery, language detection, and target scoping:

* **Polyglot Coexistence**: Verify that automatic language detection and workspace overview operations in mixed-language environments aggregate active ecosystems rather than enforcing single-language mutual exclusivity. The presence of one build system or source type must never silently suppress discovery of another.
* **Hierarchical Context Ascent**: When an operation targets a sub-path or sub-module lacking an immediate local project marker, verify that the resolver traverses parent directories up to the repository boundary to establish enclosing module and workspace context.
* **Target Containment & Scoping Fidelity**: When a specific sub-module or sub-path is selected, verify that the operation bounds its discovery to the requested scope and does not attribute sub-tree artefacts to unrelated root manifests.

### Agent steering and skill synchronisation

Audit documentation integrity between published interfaces and steering documentation:

* Verify that `skills/semedit/SKILL.md` and reference guides in `skills/semedit/references/` accurately reflect current operation names, parameter contracts, placement enums, and construct selectors.
* Check that documented routing tables recommend appropriate tools and provide explicit negative routing guards away from suboptimal shell or text alternatives.
* Confirm that capability expansions, new backends, or schema deprecations are accompanied by corresponding skill updates in the same change.

### Filesystem safety and atomic mutation invariants

Audit the integrity of all file mutations and buffer lifecycles:

* **Atomic Disk Writes**: Verify that mutating operations stage changes in memory (`sourcebuf`), write through temporary files, call `fsync`, and execute atomic renames with advancing timestamps. Ensure incomplete or failing mutations never leave corrupt partial writes on disk.
* **Workspace Manifest Protection**: Confirm that operations never mutate shared repository workspace manifests (such as `go.work`) without explicit authorization.
* **Snapshot & Staging Coherence**: Trace snapshot creation, interactive staging, and undo restoration to ensure working overlays and disk files remain coherent across sequential modifications.

### Test adequacy: verify tests prove their stated behavior

Inventory unit, integration, property/fuzz, CLI, MCP, cross-language txtar, docgen, and benchmark tests. For each layer, record the command that runs it and the behavior it actually proves.

Trace every testdata/scripts/*.txtar scenario through its runner and build an assertion matrix with one row per command:

* command and expected success/failure;
* stdout/stderr assertions;
* exact file or workspace-state comparisons;
* semantic assertions and relevant diagnostics;
* external command/tool invocation assertions.

Keep these distinctions explicit: a successful exec proves process success; stdout/stderr matching proves only the matched text is present; cmp proves the compared file equals its golden file; a fixture's go test proves only what those fixture tests assert. Identify commands with no result assertion, modified files omitted from golden comparisons, and success cases that would still pass if the intended operation did nothing.

Inspect fake LSPs, language servers, and build tools. Check that they validate expected methods, arguments, ordering, flags, edits, and side effects. Check that unexpected or missing calls fail visibly rather than producing plausible success. For command-sensitive behavior, determine whether tests detect both a missing expected invocation and an unexpected extra invocation.

Evaluate oracle strength and failure sensitivity. For key behaviors, state the regression each test would catch and propose a controlled negative control or mutation that should make it fail. Do not mutate the review baseline. For benchmark failures, trace setup, observed tool/command activity, hidden oracle execution, result persistence, and publication filtering. Check that missing assertions, skipped oracles, incomplete runs, and oracle failures cannot be reported as verified success.

Review boundary cases: cancellation, timeout, malformed data, missing tools, permission failures, partial writes, and cleanup. A cancellation test must assert timely termination or a returned cancellation result, not merely call cancel.

### Fault recovery, broken-code support, and error discrimination

Audit engine resilience and error reporting:

* **Fault Tolerance & Broken-Code Matrices**: Verify that syntax-based operations succeed despite external type errors in other packages, and that typed refactorings fail closed with clear refusal diagnostics when encountering malformed dependencies or unparseable target code.
* **Explicit Failure Semantics**: Check that operations distinguish fatal internal execution errors from external environment prerequisites (missing language servers or compilers), emitting actionable diagnostic feedback.
* **Preserving Distinction**: Ensure external failure or unparseable files never collapse silently into deceptively complete empty results.

### Accepted ADR adherence and consistency

Create a matrix covering every accepted ADR: ID/title, applicable invariant, implementation locations, test evidence, and conclusion (adheres, deviation, not applicable with reason, or unclear). Read each relevant ADR in full; the index is a discovery aid, not the complete contract. Cite evidence for both adherence and deviation.

Separately review ADR consistency. Identify contradictory requirements, duplicated normative decisions, overlapping scope, outdated premises, unresolved status, and drift among ADRs, code, capability metadata, and docs. Distinguish useful summaries/cross-references from repeated rules that can diverge. For each issue, cite the records and state which ADR should own the rule. Do not rewrite ADRs during review. Any later ADR or research-question change must update its index in the same change.

### Cancellation, resource lifecycle, and concurrency

Trace contexts, goroutines, subprocesses, pipes, LSP sessions, channels, locks, temporary files/directories, and file handles from creation through normal shutdown, cancellation, and error paths. Record ownership and cleanup. Check that cancellation reaches child work, goroutines and processes terminate promptly, close/wait behavior is sound, blocked I/O and races are addressed, cleanup errors are surfaced where meaningful, and failures do not leave misleading partial state. Link lifecycle claims to tests that assert termination or cleanup.

### Coordinate systems and source position encoding

Audit source position and range translations across all language backends:

* **Encoding Parity**: Verify consistency between 0-indexed and 1-indexed coordinates, and between UTF-8 byte offsets, UTF-16 code units (required by LSP), and Unicode character counts across multi-byte characters and newlines.
* **Multiline & Trailing Construct Bounds**: Verify that construct extractors and locators calculate valid absolute offsets and ranges across multiline declarations, comments, and files with or without trailing newlines.
* **Cross-Backend Alignment**: Confirm that equivalent construct queries (e.g. locating a function body or loop anchor) yield equivalent structural ranges regardless of the underlying parser (compiler AST, Tree-sitter CST, or external LS).

## Reporting

Cite findings with file paths and line numbers. Separate confirmed defects from risks, style opportunities, and open questions. Prioritize by impact and confidence. For each finding give the observation, consequence, evidence, test coverage/gap, and recommended next step.

End with separate concise assessments for product, language backend integrations, helper code, the highest-priority test gaps, multi-protocol presentation and deduplication, ADR adherence and consistency, lifecycle findings, coordinate encoding parity, and checks run/not run with reasons. Do not present review recommendations as implemented fixes.
