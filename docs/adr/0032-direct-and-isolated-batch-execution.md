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

### 1. Direct Batch Mode (`semantic_batch`)

Standalone AST mutations use the same source-buffer mechanism. The operation
owns a buffer only when the caller supplies none, authorizes its write scope,
and prepares edits, import organization, and formatting against that working
revision before flushing once. An operation using a caller-owned batch buffer
does not flush it. Preparation failure discards the failed operation's changes
while preserving successful earlier batch edits. New-file scaffolding may use
a single atomic publication at its existing batch barrier.

Keep `semantic_batch` as the direct mode for single-turn batched plans. The
mutating request is its own authorization. Eligible Go operations share a
request-scoped source buffer with the retained gopls session. AST reads, sibling
collision checks, symbol resolution, package loading, rename edits, and result
inspection use the same working revision. Intermediate edits send unsaved LSP
changes without publishing source files. Built-in formatting and imports operate
on the final buffers before publication. A later compiler, test, lint, or formatter
diagnostic returns `applied_with_diagnostics`; it does not roll back successful
edits.

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

Bind all existing Go source bytes, file identities, permissions, source membership,
and symlink targets in each segment's workspace and local replacement modules.
Also bind manifest and auxiliary analysis inputs. Stage and sync every changed
output before replacing any source. Recheck the complete preimage before and
between sorted per-file replacements, accounting for already published outputs.
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
   a lightweight stage receipt containing a stable, content-addressed step identifier
   (`step_id: "step_7a3f81"`, matching the ephemeral handle pattern of ADR-0003),
   an optional ordinal index for display, the targeted symbol, and an AST diff hunk.
   Surrounding disk writes and full compiler diagnostics remain deferred.
3. **Attributed Commit & Diagnostic Deltas**:
   `semantic_commit` applies all staged mutations atomically to disk via
   `pipeline.WriteAtomic` (ADR-0010), executes coalesced formatting and
   `goimports` once, and collects the compiler diagnostic delta. Any newly
   introduced diagnostics explicitly link to the originating step
   (`attributed_step_id`, `attributed_tool`), providing deterministic attribution
   without requiring the agent to deduce which edit caused a compiler error.
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

- No second approval, authenticated receipt, or client-supplied patch exists.
- Direct mode reports applied operations and the resulting diff; it does not
  promise rollback or crash recovery.
- Buffered Go edits share one working revision with gopls; save and watcher
  notifications follow actual disk publication. Intermediate source revisions
  remain private to the batch.
- A successful batch or committed stage has exactly one pre-edit and one
  post-processing diagnostic capture; individual operations defer their own
  verification, formatting, and import organization while staged.
- Syntactic and structural AST validation errors fail fast immediately at the
  individual tool invocation site; they are never deferred to commit.
- `semantic_commit` must attribute introduced compiler diagnostics to the specific
  staging step and tool that touched the affected file or construct.
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
