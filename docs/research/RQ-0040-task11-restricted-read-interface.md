<!-- This investigation preserves trace evidence for read-interface decisions without treating historical benchmark outcomes as a controlled capability experiment. -->
# RQ-0040: Task 11 Restricted Read Interface

* **Status**: Resolved
* **Category**: Benchmarks & Tool Design
* **Date**: 2026-09-26

## Question and evidence boundary

What did task 11 models need to read, what did paired baselines read, and which needs could the semedit interface satisfy?

The latest retained September 26 trajectories contain four Codex GPT-6 Luna high configurations: small/large context crossed with default/discovery prompts. Their main sessions began at 07:47, 07:50, 07:55, and 08:00 local time. Corresponding baseline sessions began at 07:47, 07:48, 07:52, and 07:56. An earlier 07:26 attempt is separate and is not counted again. Antigravity trajectories from the same morning provide a harness comparison.

No consolidated September 26 report was found in the main checkout's result directory. Conclusions below come from retained provider transcripts, extracted workspaces, the fixture, and current interface source. They establish concrete blockers, not an overall failure rate or the causal size of the restriction's effect. Older September 24 reports are not substituted for this run. No paid benchmark was rerun.

The actual historical prompt forbade shell source reads, permitted built-in inspection, required semantic lookup where supported, and required an explanation when lookup could not be used. It did **not** require every read to use semedit. The new `read|write|readwrite` metadata and default must not be assigned retroactively to these observations. See [ADR-0050](../adr/0050-benchmark-planning-sessions-and-arm-policy.md).

## Observed Codex blocker

All four inspected initial semedit trajectories found useful declarations but reported that no available tool could return their bodies. All four retained workspaces still contain the original empty normalization file, health-only classifier, and unnormalized `Dispatcher.Record` delivery. These are task failures independent of whether the existing code compiles.

For example, the small/default trajectory found `Event`, `Sink`, and `Dispatcher.Record`. The latter response contained only:

```json
{
  "symbol": "Dispatcher.Record",
  "file": "service/dispatch.go",
  "line": 14,
  "column": 22,
  "offset": 285,
  "kind": "method",
  "receiver": "Dispatcher"
}
```

The model explicitly reported that it could not safely modify the delivery path without its implementation. The small/discovery session inspected an 85-tool inventory and reached the same conclusion. The discovery prompt therefore did not resolve a missing capability.

Models guessed many names such as `AuditEvent`, `Deliver`, `Persist`, `NormalizeEvent`, `ProbeKind`, and `IsControl`. A string literal such as `"probe"` is not a declared `Probe` symbol. Missing-symbol errors here are frequently expected search misses, not backend malfunctions. Exact-name lookup cannot replace repository exploration when the model does not yet know the vocabulary.

Fallback repository-investigation tools also failed: the small/default session received `codex command failed: exit status 1: Reading prompt from stdin...`; large/discovery reported backend workspace rejection. These are separate fallback integration failures. The available semedit tools could not provide the missing context even after lookup succeeded.

## What paired baselines actually read

| Read intent | Observed baseline evidence | What semedit could supply |
| :--- | :--- | :--- |
| Discover repository layout and targets | `rg --files`, searches for audit/probe/persistence terms, project instructions | Exact known declaration locations only; no file inventory, text search, or symbol outline. |
| Understand delivery behavior | `audit/event.go`, `audit/sink.go`, `service/dispatch.go`, `audit/summary.go` | Locations for concrete declarations, but no signatures, field types, method bodies, switch cases, or surrounding imports. |
| Find where new normalization belongs | Read `audit/normalize.go` and its ownership comment | No declaration exists in this file before the task; symbol lookup cannot discover the empty extension point. |
| Preserve unrelated protocols | `metrics/writer.go`, `legacy/export.go`, and their tests | Ambiguous `Write` candidates name concrete receivers, but omit their incompatible signatures and behavior. |
| Inspect indirect and cross-file delivery | Large baselines read `audit/fanout.go`, `audit/transaction.go`, `journal/file.go`, `service/replay.go`; search for `.Write(`, `Sink`, `Record`, `DispatchAll`, `Classify` | No read-only references, implementation search, call sites, or method-expression context such as `writeEvent := audit.Sink.Write`. |
| Understand existing expectations | `audit/event_test.go`, `audit/summary_test.go`, `service/dispatch_test.go`, metrics/legacy tests | Workspace lookup excludes `_test.go`; explicit-file lookup can locate a declaration but still cannot show its assertions. Protected against editing does not mean forbidden to read. |
| Read architectural context and metadata | `docs/audit-sinks.md`, `go.mod`, `cmd/migrate/main.go`; large runs also read replay entry points | No documentation or manifest reader. Lookup cannot return the command body or its environment-variable wiring. |
| Confirm edits and scope | `git diff`, selective rereads, protected-file status checks | Some mutations return diffs, but there is no uniform source/context read surface. Verification diagnostics cannot reconstruct behavior. |

These are observed reads, not a claim that every file was essential. The smallest useful context is the event and sink declarations, the classifier and delivery bodies, package/import context, and the empty normalization file. Documentation, tests, and call sites help establish the correct boundary and avoid changing unrelated protocols.

## Antigravity comparison

All four restricted Antigravity Gemini 3.8 Flash high jobs could read fixture source with built-in `view_file`, as could their paired baselines. They used it to obtain the source missing from lookup results. The same written restriction therefore created different effective capabilities across harnesses.

| Context / prompt | Baseline transcript ID | Restricted transcript ID | Example successful restricted read |
| :--- | :--- | :--- | :--- |
| Small / default | `dafff12e-050c-4112-8bcc-e586558fe423` | `e39bab78-9664-49a3-b54d-37bca21a0b68` | `audit/normalize.go`, transcript lines 20-21 |
| Small / discovery | `8a63f037-b631-4a9a-8859-307ca5af3ad9` | `cc4c0be3-8d71-4cd3-8850-b9750618ca89` | `audit/normalize.go`, lines 26-27 |
| Large / default | `2a523915-4b3e-4b27-80d3-638dd0a8e68f` | `3fff60d3-be7d-4ac3-ab20-2e56a7aec20e` | `audit/summary.go`, lines 22-23 |
| Large / discovery | `d28f0670-f0f9-43de-bce5-81c2157acdd5` | `557641db-2671-4ce5-8417-2ee9bcee1000` | `audit/normalize.go`, lines 22-23 |

These logs are under `~/.gemini/antigravity-cli/brain/<ID>/.system_generated/logs/transcript.jsonl`. Each first line preserves the actual prompt. Successful reads have a tool invocation and following response; those two records are one read.

Large/discovery did fail to read a guessed `mcp/semedit/instructions.md` (lines 10-11), but that nonexistent tool-instruction path is separate from fixture source access. Missing `Normalize` and pre-insertion `NormalizeKind` lookups were expected misses, followed by successful file reads. Unqualified `Classify` ambiguity reflects real audit and legacy declarations and is resolved by file scoping.

Small/discovery also encountered a distinct verification failure: `semantic_verify(path=".")` invoked `gofmt -l .`, which descended into `.scratch/go/mod/` dependency testdata with intentionally invalid Go (lines 126-130). Verification scoped to `audit` and `service` then succeeded (lines 131-133). This is not a read-capability failure. Provider traces alone do not establish final hidden-oracle outcomes for these jobs.

## Interface findings

The [lookup result type](../../internal/backend/backend.go) and [formatter/definition](../../internal/operation/wire_backend.go) provide location metadata and ambiguity candidates. They have no source text, declaration end range, signature, enclosing scope, imports, or source revision. The tool description says to use lookup instead of grep when locating a named symbol; that is narrower than retrieving code needed for a semantic edit.

Two additional correctness gaps are supported by actual outputs and [resolver source](../../internal/symbol/resolver.go):

1. `Sink.Write` returns symbol-not-found although `Sink` declares `Write(Event) error`. Member scanning recognizes struct fields and concrete methods, not interface method declarations.
2. `Dispatcher.sink` returns the field plus unrelated `sink` parameters as ambiguous candidates. `Event.kind` similarly returns unrelated `kind` parameters. `scanFile` appends `scanFunctionVariables` by bare name without preserving the receiver filter. Qualification therefore does not reliably constrain lookup.

These defects are distinct from absent source retrieval. Fixing them improves navigation but would not let a model read `Dispatcher.Record`.

## Interpretation and candidate changes

The main blocker is **successful lookup without inspectable code**. Replacing a body requires knowing which existing statements, error handling, and side effects must survive. Refusal to overwrite unseen code is reasonable under the supplied restriction.

Recommended order, without accepting a new architecture in this investigation:

1. Ensure every restricted harness has a callable, permitted source reader before a paid run. Record whether reads are provided by semedit or by the harness. Fail setup explicitly if the selected policy leaves no source-inspection path.
2. If semedit alone is the intended read surface, add bounded source retrieval for a known symbol or file, with package/import context, declaration bodies, revision, and explicit truncation. File-based access is necessary for empty files. Add file/symbol inventory so agents can discover names before exact lookup.
3. Provide references/call-site context and read access to visible tests and documentation, or explicitly retain a generic read-only path for these needs. A symbol projection alone cannot satisfy them.
4. Fix interface-member lookup and receiver-filter leakage. Describe lookup as metadata-only until source retrieval exists.
5. Improve post-edit context as explored in [RQ-0027](RQ-0027-preventing-read-files.md), but do not treat edit receipts as a solution to the first read: no edit has occurred at the initial blocker.

Read restrictions are not the only explanation for task failures. Large baselines had full source access yet initially introduced wrappers or differently named helpers, then required corrective prompts to create the exact `NormalizeKind` symbol required by the oracle. The fixture's visible task describes behavior; the oracle also requires a particular declaration/file. Existing tests can pass while hidden acceptance fails. Separate read starvation, semantic misunderstanding, oracle naming requirements, mutation failures, and provider failures in reporting.

## Combined lookup and read recommendations

The parallel review, "Empirical Analysis: Benchmark Task 11 Failures Under Semedit Read Restrictions", identifies three complementary capabilities: `semantic_inspect_symbol`, `semantic_outline`, and `semantic_find_references`. Its central diagnosis agrees with the traces: a declaration location is insufficient to understand that declaration, preserve its behavior during editing, or assess its uses elsewhere.

Use those capabilities as the basis for the read-interface work. The additions below address concrete needs from task 11: initial discovery, existing implementation details, interface contracts, cross-file usages, and non-code context. Operation names remain proposals until the interface contract is accepted.

### Features to add

| Priority | Capability | Required content and task 11 acceptance example |
| :--- | :--- | :--- |
| First | `semantic_inspect_symbol` | Return exact declaration source, signature, directly associated documentation, canonical file, source range, and revision. Include the relevant package/import context so types and calls are understandable. Inspect `Event`, `Sink`, `Classify`, and `Dispatcher.Record`, including fields, interface methods, existing cases, and delivery statements. |
| First | File discovery and bounded file reading | Discover files under a selected workspace root and read a selected file even when it contains no declarations. This is necessary to find and inspect empty `audit/normalize.go` and its ownership comment; symbol-only inspection cannot reach it. Results must expose excluded paths and truncation. |
| First | Documentation, manifest, and visible-test reading | Provide or retain a permitted reader for `docs/audit-sinks.md`, `go.mod`, and visible test assertions. A harness reader can supply this capability; a semedit-only configuration needs an equivalent. Preserve mutation protections and hidden-oracle isolation. |
| Next | `semantic_outline` | Return ordered file/package declarations, signatures, struct fields, interface methods, imports, and comments, with explicit body elision. Show exported and unexported declarations and support explicit inclusion of visible tests. Discover the audit, metrics, and legacy `Classify` declarations without guessing names or reading every body. |
| Next | `semantic_find_references` | Return reference locations, enclosing declarations, and bounded source context or targets usable by the reader. Cover direct uses and method expressions such as `writeEvent := audit.Sink.Write`. Distinguish references from implementation relationships and call relationships; state which relationships and workspace scopes the backend can establish. |

The minimum usable read surface is discovery plus source inspection, with a path for non-code and test reads. Outlines reduce discovery cost; references help establish cross-file effects. Each answers a different question and must not depend on performing a mutation first.

### Existing behavior to improve

| Priority | Improvement | Acceptance criteria |
| :--- | :--- | :--- |
| First | Lookup-to-inspection handoff | Make a successful lookup immediately usable by inspection: stable canonical symbol/file selectors and explicit ambiguity candidates. Until source inspection exists, describe lookup as metadata-only and identify the permitted reader for bodies and surrounding context. |
| First | Read-policy capability matching | Ensure the selected harness exposes the permitted reading operations. Codex's observed missing reader and Antigravity's usable `view_file` require different capability handling despite similar instructions. Missing capability must be explicit before normal task execution. |
| First | Separate read and mutation permissions | Protected visible tests and manifests remain readable. Clarify allowed source and non-code readers; semantic mutation requirements must not imply that lookup can replace all reading. Hidden acceptance files remain inaccessible. |
| Next | Lookup discovery and recovery | Keep absent names, legitimate ambiguity, unsupported targets, and execution failures distinct. For unknown names, direct the model to file/package discovery. For real candidates, return copyable file/owner selectors. Preserve genuine ambiguity rather than guessing a match. |
| Next | Bounded, trustworthy read results | Supply source revision, explicit completeness/truncation, and a way to retrieve omitted content. Preserve source ordering and distinguish exact source from an outline with elided bodies. A small output must not silently conceal behavior relevant to an edit. |
| Later | Post-edit context | Apply RQ-0027's changed-source and enclosing-context proposals to avoid unnecessary confirmation reads. Include changed imports and usable continuation targets. This complements initial source inspection; it cannot replace it. |

### Lookup defects to fix

| Priority | Defect | Observable acceptance criteria |
| :--- | :--- | :--- |
| First | Interface methods are missing from lookup | `Sink.Write` resolves the interface method declaration rather than returning not-found. Define embedded-interface behavior explicitly and keep declarations distinct from implementations. |
| First | Receiver-qualified lookup includes unrelated locals | `Dispatcher.sink` resolves its field without same-named constructor/function parameters. `Event.kind` does not return unrelated `kind` parameters. Unqualified local lookup continues to report real ambiguity. |

Missing source retrieval and discovery are new capabilities. Interface-member omission and receiver-filter leakage are defects in existing lookup behavior. Fixing the defects alone will not provide readable function bodies.

### Lookup and read questions requiring investigation

1. **Smallest sufficient source context.** Start with exact declaration source plus relevant imports. Measure when enclosing declarations, sibling signatures, or a package outline provide additional information needed for the next edit. Keep initial inspection distinct from confirmation reads.
2. **Discovery without known names.** Determine whether file inventory plus outlines is enough for task 11 and related tasks, or whether bounded content search is also needed. Literal values such as `"probe"` and ownership comments are not declaration names and cannot be found by exact symbol lookup.
3. **Reference semantics.** Define behavior for interface methods, concrete implementations, method expressions, tests, generated sources, and external dependencies. A references query must not imply a complete dynamic call graph. Validate the precise relationships each backend reports.
4. **Test and non-code coverage.** Decide which reader owns documentation and manifests, and make test inclusion explicit. Workspace discovery currently excludes test files; selected-file lookup still returns only locations. Both discovery and readable assertions are necessary.
5. **Delivery through each harness.** Check that the model can actually call the advertised reader and consume its returned source. Compare generic reading and semantic reading while keeping semantic mutations fixed. The key question is whether the supplied context enables correct edits, not whether shell reads disappear.
6. **Source consistency and language support.** Define revision, truncation, pagination, and supported-language behavior before accepting the public contract. A Go implementation does not establish cross-language support. New advertised backend-operation pairs need CLI txtar coverage for their observable read behavior.

## Retained evidence identifiers

Provider transcripts are local evidence, not portable committed benchmark reports. Codex IDs below identify the main session; auxiliary/resumed sessions must not be counted as independent benchmark jobs.

| Context / prompt | Baseline Codex session | Semedit Codex session |
| :--- | :--- | :--- |
| Small / default | `01a0dc41-2452-78e2-9385-169d0bcd5d10` | `01a0dc41-2476-7721-8f33-89c372dca825` |
| Small / discovery | `01a0dc42-6b19-70e0-b5e2-392837749e3f` | `01a0dc44-0bd4-78b0-8f83-7b1f17bbf936` |
| Large / default | `01a0dc45-b632-7911-b340-6dc4b25bbec1` | `01a0dc48-9fb8-7961-8ed0-7f140feb6b5d` |
| Large / discovery | `01a0dc48-f89a-7691-a0a6-51b61d5b281e` | `01a0dc4c-fa96-7c00-b7ca-f31b528c1e85` |

Codex logs reside under `~/.codex/sessions/2026/09/26/`. Retained semedit fixture directories under `.scratch/benchmarks/` end in `1790401651214709000`, `1790401842070039000`, `1790402142087554000`, and `1790402427478618000`, respectively. The source contract is [task 11](../../testdata/bench/task_11_mixed_sink_api_migration.txtar); hidden expectations are in [its oracle directory](../../testdata/bench-oracles/task-11-mixed-sink-api-migration/).

## Resolution

The observed Codex blocker is an absent permitted source reader after successful semantic lookup. Antigravity had such a reader and used it. Baseline traces establish demand for discovery, implementation bodies, contracts, tests, documentation, and call-site context that semedit lookup cannot supply. Two additional lookup defects are independently supported by source and traces. Candidate interface changes remain proposals; no runtime code or benchmark outcomes were changed.
