# RQ-0027: Post-Edit Context and Continuation Parameters

* **Status**: Open
* **Category**: Agent Steering & Tool Design
* **Date**: 2026-09-20

---

## 1. Context & Motivation

Observed Gemini Flash trajectories sometimes alternate `edit -> read -> edit ->
read` even after a `semedit` mutation. Some reads are necessary: the next
decision may depend on code outside the change, a compiler diagnostic, or a
concurrent modification. Others merely re-read the target to establish whether
the edit succeeded or to reconstruct an argument that the edit engine already
resolved. Those reads add a model turn, tokens, cost, and latency without adding
new information.

The current response surface does not consistently establish enough post-edit
state to distinguish those cases. `FileEditRes` reports a success sentence and
diagnostic delta; only replace-body and insert-case currently include a diff.
Rename renders only a success sentence even though its backend result contains a
resolved lookup, diagnostic delta, and affected paths. Thus an agent that needs
the exact changed declaration, canonical file, or a follow-up target must infer
it again or read the source.

This RQ asks how an edit result can serve as a compact, trustworthy hand-off to
the next tool call without turning every mutation into a whole-file dump. It
does not assume that a prompt can prohibit reads: tool selection remains a
model- and harness-dependent inference, not an enforceable belief state.

## 2. Research Questions

1. Which post-edit facts prevent confirmation-only reads while preserving task
   completion and correctness?
2. What is the smallest useful code representation: a diff hunk, the changed
   AST unit, or the enclosing declaration/block?
3. Which normalized, authoritative values should be returned as copyable
   arguments for subsequent semantic operations, rather than re-derived by the
   model?
4. How should a response show truncation, diagnostic state, and source revision
   so an agent knows when a read is still necessary?
5. Which response policy works across models and harnesses rather than only for
   the observed Gemini Flash trajectory?

## 3. How an Agent Decides to Read

There is no public mechanism that exposes an LLM's internal decision to read a
file. In the common agent loop, the model receives an observation and selects
the next action from the information then present. ReAct describes this
interleaving explicitly: actions collect external information and observations
update the action plan. A read is therefore likely when the preceding result
does not make the mutation, its scope, or the next operation's arguments
legible enough to act safely.

Tool interface guidance supports this interpretation. Anthropic recommends
returning identifiers required for later calls and using concise defaults with
pagination, range selection, filtering, or truncation for potentially large
results. MCP guidance also notes that concise server instructions can help
models choose workflows but cannot guarantee behaviour; tool descriptions and
result design remain material. Consequently, a rule such as "do not read after
an edit" is a nudge only. The tool result must instead state what changed,
whether it was verified, and precisely what information remains unavailable.

More code is not automatically better context. Long-context retrieval work
finds that relevant material can be underused when buried in a long prompt, and
repository-level code-completion work likewise treats context as a restricted
and ranked resource. No source establishes a transferable best number of code
lines or tokens for post-edit results. The budget must be measured by operation,
model, harness, and task instead of hard-coded from a single observation.

## 4. Candidate Response Contract

Every successful mutation should return a compact, structured **edit receipt**
as well as a short human-readable outcome. The receipt is authoritative for the
revision that the operation wrote; it is not a cache guarantee after another
writer changes the workspace.

```json
{
  "schema_version": "semedit.edit-receipt/v1",
  "status": "applied",
  "operation": "semantic_insert_function",
  "effective_params": {
    "file": "api/server.go",
    "language": "go",
    "auto_organize_imports": true
  },
  "changes": [
    {
      "path": "api/server.go",
      "revision": "sha256:...",
      "after_range": {
        "start": { "line": 34, "column": 1 },
        "end": { "line": 41, "column": 2 }
      },
      "semantic_unit": {
        "kind": "function",
        "symbol": "InitServer",
        "owner": ""
      },
      "scope_projection": {
        "mode": "focus_and_outline",
        "language": "go",
        "focus": {
          "range": { "start": { "line": 34 }, "end": { "line": 41 } },
          "source": "func InitServer() *Server { /* ... */ }"
        },
        "outline": "package api\\n\\nfunc InitServer() *Server { /* ... */ }\\nfunc (*Server) Serve(...) { /* elided */ }",
        "elisions": [{ "range": { "start": { "line": 48 }, "end": { "line": 63 } }, "reason": "sibling_body" }],
        "truncated": false
      }
    }
  ],
  "continuations": [
    {
      "name": "inserted_function",
      "stale_if_revision_differs": true,
      "arguments": {
        "file": "api/server.go",
        "symbol": "InitServer",
        "language": "go",
        "expected_revision": "sha256:..."
      }
    }
  ],
  "diagnostics": { "introduced": [], "resolved": [], "remaining": [] },
  "truncated": false
}
```

`schema_version` makes the machine contract independently evolvable from its
human-readable rendering. `revision` is a post-write content digest, and paths
are canonical, project-relative paths. A follow-up editor accepts an optional
`expected_revision` for its target file and rejects a mismatch before writing;
the receipt must never encode a stale line/byte offset as an implicit target.
Version 1 returns copyable arguments rather than opaque handles. The result
identifies an unavailable or truncated region and the smallest range/symbol to
read, rather than falsely implying that the shown snippet is the whole file.

The structured receipt is canonical. Its text rendering is deliberately a
compact projection for clients that only surface text. Both representations use
the same paths, ranges, revisions, and elision decisions; the text must never
contain information absent from the structured receipt. This gives an LLM a
small, predictable object to inspect first, while preserving a readable fallback
for existing clients.

### 4.1 Truncated Enclosing-Scope Projection

The default context should be a **post-format enclosing-scope projection**, not
an arbitrary hunk or a whole file. It has two layers:

1. **Focus**: the changed declaration or case, returned verbatim when it fits
   the budget. This is the only layer that establishes the exact transformed
   code.
2. **Outline**: the smallest AST scope enclosing the focus, with sibling bodies
   replaced by an explicit elision marker while retaining headers, ordering, and
   selectors. It establishes available targets and structural placement without
   resending implementation detail.

An outline is display-only context, never valid source to feed back into a
mutation argument. Its receipt fields remain the authoritative machine-readable
data. Use a visible `/* elided */` or `…` marker, a revision digest, and
`truncated: true`; never silently omit code or present an incomplete projection
as a full source file.

| Operation shape | Focus (verbatim) | Enclosing-scope outline |
| :--- | :--- | :--- |
| Declaration/function/type insertion | The inserted declaration. | File package/import header and ordered top-level declaration headers. |
| Method body replacement or method insertion | The changed method, when it fits. | The enclosing class/type or file, retaining each field, constructor, and method header in source order while eliding non-focus bodies. |
| Switch-case insertion | The inserted case, including its full body when it fits. | The complete switch header, every case/default guard or selector, and only pre-existing single-line case bodies. |
| Import organization | The post-format import declaration. | Package header, import declaration, and top-level declaration headers. |
| Workspace rename | The renamed primary declaration. | Per-file outlines plus affected-file counts; do not concatenate complete files. |
| Scaffold | No focus source. | Created-file package/import header and declaration outline only, even when the complete file would fit. |

For a switch projection, preserve `switch` initialization and tag (or the
tagless condition), every `case` expression and `default`, and any existing
one-physical-line right-hand side. Replace every other pre-existing case body
with `/* elided */`. The changed case remains the focus exception: returning it
in full lets the model confirm the write it just requested. If that focus alone
exceeds the cap, return its header, range, digest, and `truncated: true`; a read
is then necessary to inspect its body.

For method operations, a headers-only outline is sufficient for the next
operation to choose a receiver, overload, relative placement, or sibling target.
It is not sufficient to reason about the changed implementation. The separate
focus layer makes that distinction explicit rather than encouraging a model to
infer omitted code.

For a scaffold operation, do not make a special exception for a small file:
return only the outline. The package clause, imports, and declaration headers
are enough to select the next insertion target; returning full generated source
would establish an inconsistent expectation that creation receives a whole-file
dump while all other edits receive bounded context.

The first experiment should compare 0, 128, 256, 512, 1,024, and 2,048 output
token caps. If the focus or outline exceeds its cap, remove lower-priority
outline detail first: unrelated bodies, then non-adjacent headers, then the
focus body. A client may offer an explicit `post_edit_context` mode, but the
baseline must have a safe useful default without an extra request.

### 4.2 Continuation Parameters

The receipt should return values that the engine, not the model, has already
normalized or resolved. This reduces coordinate and naming reconstruction while
keeping each next operation independently validated.

| Return when known | Reusable by | Reason |
| :--- | :--- | :--- |
| Canonical project-relative `file` and detected `language` | All file- or backend-scoped operations | Prevents path and backend re-detection. |
| Canonical qualified `symbol`, `kind`, `receiver`/`owner` | replace body, rename, insert relative to symbol, lookup | Prevents recomputing qualified symbol spelling. |
| Resolved `target_symbol`, effective `placement`, and container kind | Follow-up insertion | Preserves the engine's placement decision. |
| Canonical switch discriminant and changed case selector | insert case | Preserves AST-normalized anchor inputs. |
| Resolved package name and created file path | scaffold follow-ups | Allows immediate insertion into the new file. |
| Affected paths, post-write revisions, and diagnostic delta | verify, batch recovery, or selective read | Explains scope and freshness without code dumps. |

Return only parameters that are meaningful and accepted by a next operation.
Do not echo raw user input, broad workspace trust, credentials, or irrelevant
defaults. `continuations` should name the intended next target and contain
copyable argument fragments, not prescribe an action the agent must take. For
example, a body replacement may return a `function_target` usable by rename or
lookup; it must not fabricate a `source` argument for a future insertion.

Each continuation carries the edited file's `expected_revision` alongside its
copyable arguments. For a workspace mutation, each continuation is scoped to
one affected path; a bounded aggregate summary lists the other affected paths
and revisions. This keeps the common follow-up small and gives the next editor a
clear freshness precondition without inventing a global workspace revision.

### 4.3 Language Projection Rules

The receipt schema is language-neutral; projection quality is capability-based.
`semantic_unit.kind` and `owner` describe source structure without imposing Go
terms such as receiver or package on every backend. A backend may return one of
the following projection modes:

| Mode | Requirement | Result |
| :--- | :--- | :--- |
| `focus_and_outline` | The backend can parse the post-write source accurately. | Exact changed unit plus the operation-specific outline. |
| `focus_only` | The backend knows the changed post-write range but cannot safely derive its enclosing scope. | Exact bounded focus, range, revision, and `outline_unavailable` reason. |
| `outline_only` | The operation intentionally has no useful focus, such as scaffold. | The bounded outline and explicit absence of focus source. |
| `metadata_only` | The mutation has no source unit, such as a build-manifest edit. | Paths, revisions, diagnostics, and continuation values only. |

Go host-engine edits should implement `focus_and_outline` first because their
AST and post-format source are locally available. Rust and Java rename must
return only ranges and projections their trusted language-server edit and
language parser can establish. Until such a provider exists, they use
`focus_only`; they must not infer a class, method, or outline using text
heuristics. Lookup-only language slices have no edit receipt requirement.

Generate projections after the operation's final formatter/import pass. This is
what makes the receipt's source, ranges, and revision mutually consistent. A
batch creates its receipts only after its deferred final formatting pass; each
receipt then carries the final revision rather than an already-stale
intermediate one.

## 5. Task-007 Trace Evidence

The large-context, default-prompt `task-07-generate-template-main` JSONL
transcript records this exact sequence:

```text
view main.go
-> view semantic_replace_body schema
-> semantic_replace_body(main.go, main, auto_organize_imports=true)
-> view main.go
-> view semantic_organize_imports schema
-> semantic_organize_imports(main.go)
-> view main.go
-> view semantic_verify schema
-> semantic_verify(main.go)
-> run main.go
```

The source reads and mutations are materially different from the abbreviated
benchmark report's doubled `view_file` sequence:

| Step | Action | What it established |
| :--- | :--- | :--- |
| 1 | Read `main.go` | The initial empty `main` function. |
| 2 | Read replace-body schema | The parameters required for the edit. |
| 3 | Replace `main` body with `auto_organize_imports: true` | The body used bare `template` and `rand` identifiers. The tool result returned a body diff and a diagnostic, but not the selected imports. |
| 4 | Read `main.go` | The line-numbered response showed `4: "crypto/rand"`, `5: "html/template"`, and `14: val := rand.Int()`: auto-import had selected incompatible packages. |
| 5 | Read organize-imports schema | The parameters required to correct the import resolution. |
| 6 | Organize imports | Added `math/rand/v2` and `text/template`; removed `crypto/rand` and `html/template`. |
| 7 | Read `main.go` | The line-numbered response showed `4: "math/rand/v2"`, `5: "os"`, and `6: "text/template"`; it confirmed the post-edit import declaration and retained body. |
| 8 | Read verify schema | The parameters required for verification. |
| 9-10 | Verify `main.go`, then run it | Verification returned zero diagnostics; execution completed successfully. |

This is two `edit -> source read -> next edit` transitions, not two pairs of
source reads. The intervening schema reads are tool-discovery overhead, outside
the post-write-context hypothesis.

Both source responses annotate every displayed line as `<line_number>: <source
line>`. The line numbers aid text-edit tools but are additional context churn
for a semantic follow-up that needs only the import declaration and enclosing
scope. A scope projection should preserve source order and optional line ranges
as structured metadata, while presenting its outline without line-number
prefixes by default.

The enclosing-scope strategy directly addresses both source reads. After the
body replacement, the focus (`main`) plus file outline (especially the import
declaration) would show that automatic import resolution chose the wrong
packages, allowing the model to call organize-imports without reading the file.
After import organization, the import focus plus file outline would confirm the
corrected imports before verification. It cannot eliminate the schema reads or
the initial source read, and it must still preserve diagnostics: the first edit
result's `rand.Int` error was useful but insufficient to identify which packages
had been selected.

The generated benchmark result still strips arguments and result bodies. Future
reports should preserve a redacted, normalized trace with canonical paths and
ranges, mutation revision, diagnostics, and result byte/token size so this
classification does not require retrieving private JSONL files.

## 6. Solution Directions

### A. Result-Only Confirmation

Improve outcome text and tool descriptions: explicitly report that the mutation
was applied, formatted, and checked, with diagnostic delta and affected paths.
This is the smallest change and should reduce reads made only to confirm
success. It cannot supply source needed for a subsequent edit.

### B. Bounded Enclosing-Scope Projection

Return the operation-specific focus plus enclosing-scope outline described
above, with a post-write revision and explicit truncation. This should replace
reads whose only purpose is to inspect the edited target, choose a related
member, or confirm its placement. It risks context cost and must not pretend to
answer questions about elided bodies or code outside the enclosing scope.

### C. Typed Continuation Parameters

Return a schema-validated edit receipt with normalized target facts and
operation-specific continuation fragments. This directly addresses reads made
to rediscover a file, symbol, anchor, package, or language. It requires a
versioned output schema and stale-revision checks, not raw line/byte offsets.

### D. Harness Steering

Document a conditional rule in server instructions and tool descriptions:
after a successful receipt, use its snippet and continuation parameters; read
only when the next decision needs omitted code, a fresh revision, or a failed/
ambiguous diagnostic. This complements A-C but cannot substitute for them,
because instructions are probabilistic and clients vary in whether they expose
server instructions.

## 7. Evaluation Plan

Extend the RQ-0015 matrix with a response-policy dimension and preserve the
same model, harness, prompt, task, and workspace seed across paired runs:

| Arm | Edit response policy |
| :--- | :--- |
| Control | Current result formatting. |
| A | Confirmation and diagnostics only. |
| B | A plus bounded enclosing-scope projection. |
| C | B plus typed continuation parameters and revision checks. |
| D | C plus harness/server steering where supported. |

For each of the existing single-edit, composite-refactor, import, and
diagnostic-recovery tasks, record at least five randomized paired trials per
model and harness. Add trajectory labels for each read immediately following an
edit: `confirmation_only`, `missing_local_context`, `missing_cross_file_context`,
`freshness_check`, `diagnostic_investigation`, or `other` (with transcript
evidence). A read is avoidable only when the preceding receipt already contains
the exact needed fact at the same revision; do not reward suppression of a read
that would have prevented an incorrect edit.

Primary outcomes are task success against the existing AST/build/test oracles,
post-edit read count, turns to success, total prompt/output tokens, and
wall-clock latency. Secondary outcomes are stale-continuation rejection rate,
receipt size, and the proportion of reads correctly classified as necessary.
Report medians, IQRs, and paired confidence intervals; do not aggregate token
cost across providers as if units were interchangeable.

## 8. Repository Implications and Open Decisions

The ongoing CLI/MCP unification is a prerequisite, not an assumed implementation
detail. Do not attach this feature to the current transitional response paths.
Once that work is merged, review the resulting registry, dispatch, response,
and batch boundaries, then choose the narrowest shared receipt integration
point. The central operation registry is the likely owner of the typed result
shape, but that conclusion must be revalidated against the migrated design.

The eventual integration should add an output schema and structured MCP content
without weakening CLI readability or ADR-0004's requirement that every edit
returns actionable diagnostics. The multi-file rename and batch forms need a
bounded aggregate receipt rather than one unbounded snippet per file.

Open decisions:

1. Whether continuation data is explicit copyable arguments only or includes
   short-lived opaque handles.
2. Whether the default cap is global, operation-specific, or client-negotiated.
3. How MCP structured output and text fallback remain equivalent across clients.
4. Whether a scope projection is generated by each backend or a shared
   post-write source mapper after the unification review.
5. Which harnesses reliably surface server instructions and structured output.

## 9. Sources

* [ReAct: Synergizing Reasoning and Acting in Language Models](https://arxiv.org/abs/2210.03629): action observations update later plans, framing tool results as decision context.
* [Writing effective tools for AI agents](https://www.anthropic.com/engineering/writing-tools-for-agents): return identifiers needed by subsequent calls; bound large results with pagination, range selection, filtering, or truncation.
* [Server Instructions: Giving LLMs a user manual for your server](https://blog.modelcontextprotocol.io/posts/2025-11-03-using-server-instructions/): concise workflow guidance can help but cannot guarantee model behaviour.
* [Lost in the Middle: How Language Models Use Long Contexts](https://arxiv.org/abs/2307.03172): relevant information can be less usable in the middle of long context.
* [REPOFUSE: Repository-Level Code Completion with Fused Dual Context](https://arxiv.org/abs/2402.14323): repository code context benefits from relevance-aware restricted-size selection.
* [Hierarchical Context Pruning](https://arxiv.org/abs/2406.18294): removing function implementations while retaining dependency structure can reduce context without significantly reducing repository-level completion accuracy.

## 10. Next Steps

1. Extend the benchmark result schema to capture redacted, normalized tool
   traces and post-mutation result sizes; re-run task-007 before assigning a
   definitive cause to each read.
2. After the CLI/MCP unification lands, review its result and define the
   receipt integration boundary, AST projection rules, and minimal versioned
   schema; implement them behind an experiment flag for one single-file and one
   multi-file operation.
3. Run the response-policy matrix, then choose a default cap and continuation
   representation from completion and efficiency evidence.
4. If the result contract is accepted, record its invariants in an ADR and add
   CLI txtar coverage for emitted receipt fields and stale-continuation errors.
