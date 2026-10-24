# ADR-0032: Direct Batch Execution and Deferred Workspace Views

Status: Accepted
Date: 2026-09-30

## Context

Semantic batches may combine language-server edits, formatters, compilers, and
other command-line tools. Callers prefer direct low-latency mutation and accept
a partially applied batch when later validation reports diagnostics.

However, real-world agent interactions reveal fundamental tensions between the
theoretical benefits of multi-edit batching and the actual mechanics of LLM
generation and turn economics:

1. **JSON Envelope & Schema Penalty**:
   Calling an individual native tool (`semantic_replace_decl`) benefits from
   constrained decoding and logit masking at the model engine layer, mapping
   reasoning directly into validated parameters. Conversely, `semantic_batch`
   requires the model to manually synthesize a nested polymorphic JSON array
   (`[{"tool": "...", "parameters": {...}}]`). This increases cognitive overhead,
   degrades schema adherence, risks syntax escaping errors, and contradicts the
   model's habituated single-intent tool selection.
2. **Turn Economics and Quadratic Token Churn**:
   When models avoid `semantic_batch` due to envelope friction, they fall back to
   sequential single-tool turns. Each sequential turn re-evaluates the entire
   prompt history (system instructions, tool definitions, prior conversation),
   imposing quadratic token churn ($O(N \times \text{history})$) and compounding
   network and inference prefill latency ($N \times \text{TTFT}$). While prompt prefix
   caching discounts input costs, it does not make turns free or eliminate
   round-trip turn latency.
3. **MCP Specification Evolution (2025-06-18)**:
   The Model Context Protocol specification removed native JSON-RPC batch arrays
   (`[{...}, {...}]`) to preserve transport simplicity. Multi-tool execution in
   modern harnesses (Codex, Antigravity) operates via model-generated parallel
   tool calls emitted within a single assistant message (`tool_calls: [...]`), which
   the harness translates into pipelined or concurrent JSON-RPC requests over stdio.

A temporary copy-on-write workspace is a useful orchestration primitive, but it
is not a different semantic-editor capability. An LLM or another controller can
create such a workspace through its own filesystem or workspace-management MCP,
run `semedit` there, inspect the resulting diff, and decide how to carry the
result forward. Embedding that lifecycle here would couple the semantic engine
to a provider, platform, repository, cache, and publication policy it does not
need to own.

## Decision

We unify batch execution across two complementary access paths: direct multi-edit
plans and state-journaled staging transactions.

### Operation-Call Parity

A batch child is the same registered operation exposed as an individual tool;
both paths must use the same parser, handler, finalizer, and output projectors.
Given the same arguments and source revision, they apply the same normalization,
validation, semantic transformation, refusal rules, and operation-level output
contract. Batch execution must not accept extra fields, weaken required
parameters, suppress operation failures, or return a looser result schema. It
may add its edit index, execution status, and shared final diff/diagnostic
envelope around the child result.

The operation implementation must not fork on whether it was invoked directly
or as a batch child. The caller supplies its source buffer through the same
operation dependency: an individual call receives a fresh request buffer, while
batch children receive the shared working buffer so later children observe
earlier edits. The operation reads and writes that buffer using the same code in
both cases. It must not inspect invocation-mode flags to select different
normalization, validation, transformation, refusal, or result behavior.

Buffer ownership and orchestration remain outside the operation. The caller
chooses when to publish, normalize, and verify the buffer; staged execution also
keeps its shared buffer reversible so callers can drop selected steps or abort
before commit. Those lifecycle differences may change when effects become
visible, but not the operation's behavior or canonical result contract
(ADR-0065). Batch may add its edit index, execution status, and shared final
diff/diagnostic envelope around the same child result.

### 1. Direct Batch Mode (`semantic_batch`)

MCP batching is a startup-configured catalog feature. `--enable-batching` is
disabled by default and must be explicitly enabled. When disabled, the server omits
`semantic_batch`, rejects calls to that unadvertised name, and removes
batch-specific guidance from exposed operation descriptions and input schemas.
When enabled, it advertises `semantic_batch` where eligible operations exist
and adds matching guidance to those operations' descriptions and input schemas.
This toggle controls MCP discovery and invocation only; individual operations,
CLI dispatch, and retained staging remain available independently. Catalog text
must never direct an agent to a disabled feature.

Choose buffering according to the required preparation and publication boundary:

| Operation context | Publication policy |
| --- | --- |
| Interactive `semantic_stage` through commit, drop, or abort | Strictly buffered; steps remain reversible in memory until commit. |
| Multi-file AST/CST or delegated refactoring plans requiring preflight or chained edits, including Go moves/signature changes and Java parameter-object caller rewrites or extract-then-rename | Strictly buffered; validate and normalize the complete plan before publishing any output. |
| Overlapping or coupled steps in a `semantic_batch` segment | Shared buffer; coalesce formatting, imports, and publication per changed file. |
| Standalone single-file Go AST mutation, such as body replacement, function insertion, or construct replacement | Prepare and format in process using lazy target/read-set preimages, then publish through the checked atomic publisher; no implicit workspace-wide source capture. |
| Standalone single-file Java CST mutation | Use a target-file buffer for JDT LS normalization and version-matched push diagnostics before publication, then send `didSave`; no full-workspace source scan. |
| Standalone Go `semantic_rename` | Obtain the complete gopls `WorkspaceEdit`, stage all outputs, and publish directly with checked target preimages. |

`sourcebuf.Prepare` may create an owned lazy buffer when the caller supplies
none. Construction and write authorization resolve roots without enumerating
or reading source trees. Reads and writes capture their selected preimages;
explicit analysis observations may capture a wider selected package/module tree.
This lifecycle is shared by Go and Java without fixed source extensions.
Caller-owned buffers remain authoritative and are never flushed by a child.
Preparation failure discards the failed operation's changes while preserving
successful earlier batch edits. New-file scaffolding may use a single atomic
publication at its existing batch barrier.

Buffering preserves imports needed by later steps rather than allowing early
`goimports` to strip them. Repeated edits to one file incur one final `fsync`
per segment, rather than one per child. The measured twenty-file rename spent
82.10 ms in `fsync` (about 4.1 ms/file; RQ-0070). Coalescing avoids repeated
writes to the same file; twenty distinct changed files still require twenty
durable replacements. Direct writes retain temporary-file, `fsync`, rename,
advancing-mtime, and final preimage checks.

Keep `semantic_batch` as the direct mode for single-turn batched plans. The
mutating request is its own authorization. Eligible Go operations share a
request-scoped source buffer with the retained gopls session. AST reads, sibling
collision checks, symbol resolution, package loading, rename edits, and result
inspection use the same working revision. Synchronize unpublished bytes with
gopls only immediately before a dependent LSP query. AST-only children send no
pre-flush changes; standalone Go edits and Go AST-only batches notify gopls after
publication. A query that opened an unsaved overlay still requires the
same-content post-flush update described in ADR-0060. Built-in formatting and imports operate
on the final buffers before publication. A later compiler, test, lint, or formatter
diagnostic returns `applied_with_diagnostics`; it does not roll back successful
edits.

This timing is specific to the Go in-process AST and formatter pipeline.
Java's embedded CST splices source ranges but delegates formatting,
`source.organizeImports`, chained refactorings, and diagnostic verification to
JDT LS (ADR-0068). Java must send current buffered bytes via `didChange` before
those requests, wait for matching push diagnostics, publish the final bytes,
and send `didSave`. A target-file buffer suffices for a single-file pipeline;
multi-file postprocessing and chained caller rewrites retain all affected
buffers until validated. Shared buffering policy does not imply shared LSP
notification or diagnostic-trigger settings.

The operation registry explicitly marks buffer-safe operations; new operations
default to direct disk execution. Go rename, replacement, insertion, extraction,
inlining, assertion rewrites, ordinary removal, imports, and declaration movement
can buffer existing regular Go files. Scaffolding, dependency changes, safe
deletion and other language backends are publication
barriers. Publish the preceding segment, execute the disk operation, and start a
new source buffer. External normalization and checks run after publication.

A batch selects one canonical `workspace_root`; child operations may omit that
field or repeat the outer root. A different, malformed, or null child root fails
before dispatch. Top-level `trust_workspace=true` supplies root-scoped trust to
all child contexts and final verification. A child may still set its own trust
flag for that operation, but child trust alone never authorizes final hooks.
Direct `ExecuteBatch` calls create and release a temporary diagnostics manager
when the caller supplies none. MCP calls share a session manager by module and
environment; a different worktree module gets its own Go state and does not
reuse the configured-root gopls session. These scopes do not change direct
batches' existing fail-fast, partial-application behavior.

Bind preimages lazily for the actual read/write set: exact bytes, file identities,
permissions, missing-file expectations, and symlink targets. Explicitly observed
package trees, including local replacements used by overlay `go/packages`
validation, also bind membership, manifests, and auxiliary analysis inputs.
This applies to both `.go` and `.java` inputs and every source-buffer caller,
including Java rename, staged code actions, and interface verification.
Lazy binding must not omit package siblings or inputs on which semantic
validation depends. `sourcebuf.New` must not eagerly read every workspace
`.go` and `.java` file, and per-output checks must not rescan unrelated trees.
Stage and sync every changed output before replacing any source. Validate the
complete observed read set and tree membership before publication, then check
each target's preimage immediately before its sorted replacement, accounting
for already published outputs. Full observed inputs and membership are checked
before staging and again before the first replacement; only each output's
canonical path, identity, mode, and exact bytes are rechecked between replacements.
Tree selections retain their extensions, recursion scope, and exclusion policy
through fork/replay and publication. Ignore only the publisher's owned temporary
files when rechecking membership after staging. Go package loaders explicitly
select Go trees; symbol collision checks observe their package directory without
implicitly selecting nested packages. Generic/Java observations do not query
the Go environment, while Go observations retain effective cache exclusions.
Each file replacement is atomic with an advancing modification time. Ordinary
filesystem operations cannot make multiple file replacements globally atomic.
A conflict before publication changes no source files; a late failure reports
the published count and leaves the authoritative disk diff available.

On a child failure, publish the buffered edits reached so far without final
normalization. Publication cleanup has a bounded context independent of request
cancellation so completed earlier edits retain the existing partial-application
contract. Drop unpublished bytes and invalidate retained analysis if publication
fails. Final diagnostics and disk hooks never inspect an unpublished revision.

A multi-file rename validates every working preimage, output scope, and disk input
before changing any buffer bytes. Failure leaves that child's outputs untouched.
Direct disk rename validates all planned output preimages before the first write,
then checks only the corresponding target's canonical path, regular-file identity,
and exact contents after staging and immediately before its atomic replacement.
It does not rescan every module input between outputs. A zero-write conflict may
retry once using fresh symbol resolution and fresh LSP edit coordinates for the
same declaration identity. A partial publication is reported with published paths
and is never replayed or automatically rolled back. The per-target check and
replacement are separate filesystem operations; they are not a compare-and-swap.
Cross-file and cross-package moves use the same all-output application boundary.
Cross-package compiler preflight loads the current source overlay and merges the complete
move plan over it for the final typecheck. Same-package moves retain structural
validation, including matching test-file and build constraints. Editable local replacement inputs are
bound read-only and reloaded until discovery reaches a stable source revision;
package loading uses the project environment and refuses manifest updates.
Resolve and bind the enclosing main module before package loading, including when
the MCP workspace starts in a subdirectory. Local replacement modules remain
read-only inputs. A request-local publication journal preserves the first preimage
of explicitly changed external files across barriers. Final formatting and diff
include those files; verification and hooks retain the requested workspace root.

For a fully successful batch, capture diagnostics once before the first edit,
apply every operation, coalesce deferred formatting and import organization,
then capture diagnostics once after post-processing. Built-in Go formatting and
import organization use the sorted set of existing files whose bytes actually
changed in the effective disk and buffered source view; diagnostics, trust checks, configured
normalization hooks, and checks keep their established workspace scope. An empty
changed-file set skips built-in Go formatting. Return that single final
diagnostic delta on the batch response. Do not run a diagnostic pass for each
child operation and do not require a separate `semantic_verify` after a
successful batch. If an operation or post-processing step fails, preserve the
existing fail-fast partial-application behavior and omit the final delta. The
response still includes receipts through the failing edit and a final diff of
actual persisted writes when that diff can be captured.

### 2. State-Journaled Staging (`semantic_stage` $\to$ edits $\to$ `semantic_commit`)

To eliminate the JSON envelope penalty while preserving turn and token efficiency,
we establish state-journaled staging as a peer batching mechanism:

1. **Immediate Syntax & Structural Validation**:
   When a staging session is active, individual semantic tools validate syntax,
   delimiters, and target AST nodes immediately in memory against the staging
   buffer. If a snippet is syntactically invalid or targets a non-existent symbol,
   the individual tool call fails fast immediately in that turn with `isError: true`.
   Invalid AST mutations are never staged.
2. **Deterministic Step Attribution & Stable Step IDs**:
   Successful staged calls append to an in-memory modification journal and return
   a lightweight native `stage_receipt` containing a stable, content-addressed step identifier
   (`step_id: "step_7a3f81"`, matching the ephemeral handle pattern of ADR-0003),
   an optional ordinal index for display, historical target information, and a
   working-view diff hunk. This is not a live ordinary-result continuation or
   persisted disk diff. Disk writes and final diagnostic feedback remain deferred;
   a backend may still run the in-buffer checks needed by a dependent query.
3. **Attributed Commit & Diagnostic Deltas**:
   `semantic_commit` normalizes the buffered Go revision with coalesced formatting
   and imports, then publishes through checked per-file atomic writes (ADR-0010)
   and collects the compiler diagnostic delta. It does not promise an atomic
   transaction across files; partial publication is reported explicitly. Any newly
   introduced diagnostics include candidate step IDs based on changed paths.
   These links identify relevant edits rather than proving a single originating
   step; diagnostics in dependent files can have no candidates.
4. **Granular Reversion (`semantic_stage_drop`)**:
   `semantic_stage_drop` removes a single staged mutation from the in-memory journal
   before commit. To prevent off-by-one errors and re-indexing hazards (where
   dropping step 2 shifts step 3 into index 2, risking accidental clobbering by
   subsequent agent commands), `semantic_stage_drop` addresses mutations by their
   stable `step_id` (e.g. `step_id: "step_7a3f81"`). The engine recalculates the
   in-memory AST buffer across the remaining journal entries and returns the updated diff.
5. **Clean Abort (`semantic_stage_abort`)**:
   `semantic_stage_abort` drops the entire in-memory journal with zero disk mutations,
   providing clean cancellation without git worktree or stash pollution.

The source buffer is a revision of the supplied workspace, without an embedded
`WorkspaceView` or isolated-batch MCP operation. There is no principled safety distinction between direct one-file and
direct multi-file edits: both are authorized mutations in the caller's working
copy. Direct operation is also appropriate when formatters, generators, and
other cooperative tools need to write their normal derived files.

If a future caller needs speculation before publication, it owns workspace
isolation and publication. `semedit` continues to operate on the workspace it
is given. This is intentionally provider-neutral: it neither selects nor
requires a copy-on-write implementation, Git worktree, overlay filesystem, or
operating-system sandbox.

## Invariants

- Lazy source preimages and checked publication are language-neutral. Diagnostic
  cache coverage is backend-owned: Go's import-aware reverse-package refresh
  (ADR-0060) does not change Java's normalization, push-diagnostic, or save
  lifecycle (ADR-0068), and does not establish equivalent caches for other LSPs.

- No second approval, authenticated receipt, or client-supplied patch exists.
- The MCP startup toggle gates batch discovery and calls without changing
  registered operation behavior, CLI availability, or retained staging.
- Batch guidance appears in MCP descriptions and schemas only when batching is
  enabled; a disabled catalog contains no batch-specific guidance.
- Direct mode reports applied operations and the resulting diff; it does not
  promise rollback or crash recovery.
- Buffered Go edits share one working revision with gopls; save and watcher
  notifications follow actual disk publication. Unpublished bytes synchronize
  only before dependent LSP queries; AST-only steps defer synchronization.
- LSP synchronization follows backend semantics. Java retains pre-publication
  changes for normalization and push verification, followed by post-publication
  saves. Push-diagnostic backends must not inherit gopls's suppression policy.
- Staging, coupled batch segments, and multi-file AST/CST preflight retain buffers.
  Standalone single-file pipelines use target-scoped preparation and checked
  atomic publication; Java retains its normalization buffer. Neither buffer
  construction nor write authorization captures a full workspace implicitly.
- A batch child and its standalone tool counterpart use the same registered
  parser, handler, finalizer, and output projectors. Their only operation-context
  difference is whether they receive a fresh request buffer or the batch's
  shared working buffer. Buffer publication and staging drop/abort remain
  lifecycle responsibilities outside the child operation implementation.
- Lazy preimages cover every semantic read and write and explicitly observed
  package tree. Membership changes and exact-content conflicts remain failures;
  reducing unrelated scans cannot weaken pre-publication validation.
- A successful batch or committed stage has exactly one pre-edit and one
  final post-processing diagnostic capture for its reported delta. Eligible Go
  captures may reuse revision- and coverage-validated findings under ADR-0060;
  two logical captures do not mandate two fresh full-workspace analyses. Go
  children defer formatting, imports, and diagnostic captures. Backend-required
  in-buffer checks and normalization, such as Java's JDT LS pipeline, may run
  before publication; they do not replace the final reported diagnostic state.
- Syntactic and structural AST validation errors fail fast immediately at the
  individual tool invocation site; they are never deferred to commit.
- `semantic_commit` supplies candidate step IDs for relevant changed paths when
  available. These links do not prove causality; dependent-file diagnostics may
  have no candidate step.
- The successful batch response carries the final diagnostic delta. A failed
  batch does not claim a final diagnostic state it did not capture.
- A higher-level controller may provide a private working copy, but
  semantic-editor makes no isolation or security claim about it.
- `semedit` does not manage copy creation, tool confinement, diff publication,
  rollback, or workspace cleanup for a controller-owned copy.

## Consequences

Direct batches and state-journaled staging together solve both scripted
orchestration and model-ergonomic interactive refactoring:

- **Turn and Token Efficiency**: Eliminates redundant intermediate verification
  passes and prevents quadratic prompt-history re-reading.
- **Model Alignment**: Enables agents to use their natural, un-nested native tool
  schemas without incurring JSON serialization overhead or losing logit-constrained
  grammar validation.
- **Clear Failure Diagnosis**: Combines instant feedback on syntax mistakes with
  exact step attribution on downstream compiler diagnostics.
- **Bounded Diagnostic Cost**: Keeps compiler verification overhead bounded to
  $O(1)$ diagnostic captures regardless of edit count.

## Deferred Design Record

This decision retains the investigation because it may become relevant for a
future speculative-edit workflow, not because current direct batches are
unsafe. Reopen it only for a concrete requirement such as previewing an
unbounded third-party tool write set, comparing candidate transformations before
choosing one, or preserving a stable input while another actor is concurrently
editing the live tree.

Prior art confirms that this is an orchestration-layer problem with
platform-specific trade-offs:

- Apple documents APFS copy-on-write clones through `clonefile` and
  `copyfile`; the [APFS tools and APIs guide](https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/APFS_Guide/ToolsandAPIs/ToolsandAPIs.html)
  describes the platform primitive.
- [cow](https://github.com/joeinnes/cow) is one external workspace manager. It
  exposes a separate MCP that creates APFS CoW copies, runs commands there, and
  extracts a branch or patch. It is evidence that an agent-level controller can
  own this lifecycle, not a dependency or recommendation for semantic-editor.
- [Git worktrees](https://git-scm.com/docs/git-worktree) are a portable
  repository-oriented alternative, with their own checkout and untracked-file
  trade-offs.
- [Bubblewrap](https://manpages.debian.org/unstable/bubblewrap/bwrap.1.en.html)
  is a Linux-specific process-isolation mechanism. It is relevant only if a
  future requirement is operating-system confinement; a private working copy
  alone is not a security boundary.

The retained, unmerged exploration is
`codex/direct-isolated-batch` at `de22f2c3951f048e3fe8a97b149e270981c3b983`.
It is not a current product feature. The dedicated Sol review found these
constraints before any future reuse:

1. A view nested beneath the live repository, with inherited environment or
   incomplete argument mapping, lets Git-aware tools rediscover and mutate the
   live root. A future design must establish an execution boundary appropriate
   to its stated, non-security isolation claim.
2. A denylist cannot enforce an "existing regular source files only" boundary.
   A future publisher needs a positive source-file classifier and must exclude
   resources, manifests, and private scratch paths.
3. Start metadata alone is insufficient. Publication must preserve full mode
   and validated preimage bytes, then revalidate immediately before publishing,
   so an intervening edit cannot become the patch preimage.
4. A generated unified patch must reject or correctly quote control-bearing
   filenames and verify that its parsed targets exactly match the validated
   change set.
5. Direct batches need one final workspace diff, including formatter and
   import changes, on success, diagnostics, and partial semantic-operation
   failure.

The review also found that the prototype omitted empty directories and recreated
parent directories with the wrong mode. These are design constraints, not
implementation work authorized by this ADR.

The review's final-diff finding remains a direct-batch conformance task. It is
independent of, and must not be used to justify, an embedded workspace view.

## Restoration integration: 2026-10-12

Retained staging uses the source buffer shared with coupled batch segments.
Standalone AST operations reuse its lazy preimage and checked publication
mechanism; they do not implicitly capture all workspace source files.
The server binds each journal to its canonical request root and current Go module
owner, rejects overlapping roots and competing stages in one module, and rechecks
ownership before publication. Reinitialization, reload, and disconnect discard
volatile state and invalidate retained diagnostics. Ordinary abort and expiry
discard the journal without writing disk, then reconcile overlapping retained Go
sessions to the current disk view. Existing documents receive current disk bytes;
staged-only documents close. Concurrent external edits remain authoritative.
An overlapping staging reservation remains active until reconciliation finishes,
with language-server work outside the staging lock. Failed reconciliation reports
an error and discards affected Go sessions rather than serving abandoned overlays.
This retained-process recovery is Go-specific; other backends retain their own
document and save lifecycles. Failed replay and uncertain publication retain the
conservative invalidation policy.

Runtime-directory filtering is shared across language backends. When complete
absolute Go runtime paths are explicit in the process environment and no managed
base or module-cache seed needs resolution, those paths determine the exclusions
without a `go env` subprocess. Missing, relative, or malformed paths and managed
state retain effective configuration resolution. Each filter observes current
process paths; this optimization introduces no process-wide environment cache.

Step IDs remain unique across drop and replay. Rejected edits and failed replay
preserve the accepted revision. Publication distinguishes zero writes, partial
writes, and complete publication: zero-write failure retains the journal, partial
publication records its error, count, and actual diff and permits only abort,
and complete publication consumes the journal. New Go files and their parent
directories remain unpublished until commit. Ordinary batches share this publisher.

The typed optional native `stage_receipt` companion carries staged step selectors.
It is indexed separately from ordinary `result` continuations; staged children
never advertise live disk selectors. Diagnostic rows carry candidate step IDs
based on changed paths. These candidates identify relevant recorded edits, not
proof of one causal origin; dependent-file diagnostics can have no candidates.

Provisional immutable limits remain 50 accepted steps, 10 MiB retained canonical
requests, and 10 MiB changed-source bytes. Stage age defaults to 30 minutes.
These are retention policies, not measured heap, RSS, or latency guarantees.
[RQ-0067](../research/RQ-0067-staging-resource-accounting-and-defaults.md) remains open.
