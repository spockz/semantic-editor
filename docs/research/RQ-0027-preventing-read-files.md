<!-- Preserves continuation research and the shared schema/receipt gates needed for resumable semantic refactoring handovers. -->
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

### Invariant: Continuation Field–Parameter Isomorphism

Every field in a structured tool result that is intended for use as a follow-up
tool argument **must match the parameter name and granularity that the target
tool's input schema declares**. No field may require the model to reassemble,
rename, or reinterpret its value before passing it to a subsequent semedit call.

Concretely:

* A result field named `symbol` must carry the fully-qualified symbol string
  accepted verbatim by `semantic_rename`, `semantic_replace_body`,
  `semantic_inspect_symbol`, and `semantic_find_references` (e.g.
  `"Server.HandleRequest"`, not `"HandleRequest"` + a separate `receiver` field).
* A result field named `file` must carry the canonical project-relative path
  accepted verbatim by all file-scoped tools.
* A result field named `kind` must use the same enum vocabulary as the `kind`
  parameter of `semantic_replace_construct` — not Go AST token names, not LSP
  `SymbolKind` integers.
* Fields with no corresponding tool parameter (`line`, `column`, `offset`,
  `receiver` as a standalone field, raw `before`/`after` diagnostic arrays)
  must be absent from structured continuation data. They may appear in the
  human-readable text content layer but never in `structuredContent`.

This invariant applies to every continuation block in an edit receipt and to
every navigation result that carries addressable output. Its purpose is to make
the model's next argument a verbatim copy from the preceding result, removing
reassembly inference entirely from the tool-call trajectory.

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

## 8. Holistic Input/Output Parameter Harmonization

The continuation field–parameter isomorphism invariant (§4) requires that
output fields feed directly into follow-up tool inputs. That invariant is only
meaningful if the input schemas themselves use consistent names and granularity
for the same logical concept across all tools. Both sides of the interface must
be harmonized together.

### Invariant: Schema-Wide Parameter Name Consistency

A logical concept must use the same JSON parameter name in every tool that
accepts or returns it. No two tools may use different names for the same thing,
and no name may carry different semantics in different tools.

### Current Inconsistencies (Audit 2026-09-28)

The following table records every parameter name divergence identified in
`internal/operation/`. Each row is a blocking inconsistency: an output field
conforming to one name cannot feed verbatim into a tool using the other name.

| Concept | Consistent name | Divergent names found | Affected tools |
| :--- | :--- | :--- | :--- |
| **Containing function for construct ops** | `in_function` (selector role, not `symbol`) | `function` (`semantic_replace_construct`), `func` (`semantic_insert_case`) | replace_construct uses `function`; insert_case uses `func` |
| **Relative case placement anchor** | `target_case` | `anchor` (`semantic_insert_case`) — collides with symbol-addressed `target_symbol` in other tools | insert_case vs all other insertion tools |
| **File scope** | `file` | `path` (`semantic_outline` accepts files and directories; `semantic_verify` uses both `path` and `file` with different semantics) | outline, verify |
| **Replacement source text** | distinguish `body` (bare statements) from `source` (complete declaration) | `semantic_replace_construct` uses `source` for a complete construct — a third meaning | replace_construct vs replace_body vs insert_* |
| **Ambiguity path** | `construct_path` | `switch_path` (`semantic_insert_case`) — same concept, different name | replace_construct vs insert_case |
| **Switch discriminant** | `discriminator` | `switch_on` (`semantic_insert_case`) — same concept, different name | replace_construct vs insert_case |
| **Kind vocabulary** | one shared semedit enum | `LookupResult.Kind` uses Go AST token strings; `kinds` filter in outline uses a broader cross-language vocabulary; mutation `kind` inputs use a third vocabulary | navigation results vs outline filter vs mutation inputs |

### Parameter Role Distinction: Target vs Selector

Before deciding canonical names, every parameter must have a declared role:

* **Target** — the thing the operation acts on directly. The receipt's
  continuation fields echo target parameters back so the next tool can address
  the same unit. Examples: `symbol` in `semantic_rename` and
  `semantic_replace_body`; `file` in `semantic_organize_imports`.
* **Selector** — a scope constraint that narrows where the engine searches for
  the real target. The engine uses it to resolve ambiguity; it is not the thing
  being changed. Examples: `file` as optional scope disambiguator in
  `semantic_rename`; the containing function in construct operations.

`semantic_replace_construct` and `semantic_insert_case` both take a containing
function parameter. That parameter is a **selector**, not a target — it says
"look inside this function," while `kind` and `discriminator` identify the
actual construct to mutate. Renaming it to `symbol` would be wrong: `symbol` is
reserved for the primary target across the entire tool surface, and a receipt
field named `symbol` must refer to the operated-on unit, not its container.

The correct canonical name for the containing-function selector is `in_function`.
It is unambiguous (`symbol` is the target; `in_function` is the scope), it
reads as a preposition phrase, and it does not collide with any other parameter.
A result receipt for a construct operation would therefore look like:

```json
{
  "file": "api/server.go",
  "in_function": "Server.ServeHTTP",
  "kind": "if",
  "discriminator": "err != nil"
}
```

where `in_function` and `file` together supply the scope selectors for a
follow-up `semantic_replace_construct` call, and `kind` + `discriminator`
identify the construct within that scope.

### Design Decisions Required Before Harmonization

1. **`function`/`func` → `in_function`**: Rename both to `in_function` in
   `semantic_replace_construct` and `semantic_insert_case`. Do not rename to
   `symbol`; the containing function is a scope selector, not the operation
   target, and conflating the two roles would break the isomorphism invariant
   for receipts.

2. **`source` vs `body`**: These are legitimately distinct scopes — `body` is
   bare statements without braces; `source` is a complete declaration. They
   should keep different names. `semantic_replace_construct`'s `source` is a
   third scope (complete construct including keyword and braces); its parameter
   description must state this explicitly. Do not collapse all three to one name.

3. **`path` vs `file`**: `semantic_outline` uses `path` because it accepts
   both files and directories. `semantic_verify` uses `path` for Go scope and
   `file` for external language scope — a load-bearing distinction. Consider
   unifying both to `path` with documented single-file vs directory semantics,
   or splitting verify into language-specific parameter sets.

4. **`anchor` → `target_case`**: Rename insert_case's `anchor` to
   `target_case` to remove ambiguity with symbol-addressed `target_symbol`
   placement.

5. **`switch_path` → `construct_path`**: Unify to `construct_path` across
   insert_case and replace_construct.

6. **`switch_on` → `discriminator`**: Unify to `discriminator` across both
   tools.

7. **Kind vocabulary**: Define one shared kind enum used in mutation inputs,
   navigation filters, and continuation outputs. The enum must not use Go AST
   token names or LSP SymbolKind integers.

8. **Diagnostic delta serialization**: `DiagnosticDelta.Before` and
   `DiagnosticDelta.After` (full pre- and post-edit diagnostic lists) must be
   tagged `json:"-"` in both `pipeline.DiagnosticDelta` and
   `backend.DiagnosticDelta`. Only `introduced`, `resolved`, `net_delta`, and
   `suggestions` reach the model. The full arrays remain in the Go struct for
   internal delta computation and must never be serialized into tool results.

9. **`LookupResult` field policy**:
   * `Receiver` as a standalone field must be omitted from structured output.
     The qualified `symbol` value already encodes it (e.g. `"(*Server).Handle"`);
     a separate `receiver` field forces the model to reconstruct what the engine
     already resolved.
   * `Offset` must be omitted from structured output entirely. It is an internal
     Go LSP byte-offset artifact with no tool-parameter mapping.
   * `Line` and `Column` must remain as flat fields in results (no sub-object
     wrapper). They are valid for file-navigation reads but are not semedit tool
     parameters. Their tool description must state this explicitly (see item 10).

10. **Tool description cross-reference requirement**: Every tool description
    must explicitly state which result fields are continuation arguments for
    semedit tools and which are navigation-only hints. Example for
    `semantic_lookup`:

    > Returns `symbol`, `file`, `kind`, `line`, and `column`. Pass `symbol`
    > and `file` to follow-up semedit calls. Use `line` and `column` only for
    > file-navigation reads; they are not accepted as semedit tool parameters.

    Every input parameter description must also state which result field(s) feed
    it (Direction A cross-reference). Example for `semantic_rename.symbol`:

    > Target: qualified symbol being renamed (e.g. `Server.ServeHTTP`,
    > `(*Client).Do`, `ValidateToken`). Copy verbatim from the `symbol` field of
    > any preceding `semantic_lookup`, `semantic_rename`, or mutation receipt.

11. **`SourceFields` registry annotation**: Add a `SourceFields []string` field
    to `ParameterContract` in `internal/operation/`. The schema generator must
    append the Direction A cross-reference clause to the parameter description
    automatically when `SourceFields` is non-empty, keeping descriptions
    consistent without manual maintenance. A registry build-time check must
    verify that every named source field exists in the referenced tool's output
    schema, catching broken cross-references before release.

### Prerequisite for Receipt Implementation

The continuation field–parameter isomorphism invariant cannot be correctly
implemented while input-schema divergences exist. A receipt that emits `symbol`
for the containing function cannot feed `semantic_replace_construct` which
currently expects `function`. Resolve the input schemas first, then implement
the receipt shape against the canonical names.

## 9. Repository Implications and Open Decisions

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
6. Whether `body` and `source` should be distinct named parameters or collapsed
   to one name with scope declared in the description.
7. Whether `path` (file or directory) and `file` (file only) should be unified
   or kept distinct with documented semantics.

## 10. Sources

* [ReAct: Synergizing Reasoning and Acting in Language Models](https://arxiv.org/abs/2210.03629): action observations update later plans, framing tool results as decision context.
* [Writing effective tools for AI agents](https://www.anthropic.com/engineering/writing-tools-for-agents): return identifiers needed by subsequent calls; bound large results with pagination, range selection, filtering, or truncation.
* [Server Instructions: Giving LLMs a user manual for your server](https://blog.modelcontextprotocol.io/posts/2025-11-03-using-server-instructions/): concise workflow guidance can help but cannot guarantee model behaviour.
* [Lost in the Middle: How Language Models Use Long Contexts](https://arxiv.org/abs/2307.03172): relevant information can be less usable in the middle of long context.
* [REPOFUSE: Repository-Level Code Completion with Fused Dual Context](https://arxiv.org/abs/2402.14323): repository code context benefits from relevance-aware restricted-size selection.
* [Hierarchical Context Pruning](https://arxiv.org/abs/2406.18294): removing function implementations while retaining dependency structure can reduce context without significantly reducing repository-level completion accuracy.

## 11. Next Steps

1. Extend the benchmark result schema to capture redacted, normalized tool
   traces and post-mutation result sizes; re-run task-007 before assigning a
   definitive cause to each read.
2. Resolve the seven input-schema divergences in §8 before implementing the
   receipt shape; record the canonical names in an ADR as a schema invariant.
3. After the CLI/MCP unification lands, review its result and define the
   receipt integration boundary, AST projection rules, and minimal versioned
   schema; implement them behind an experiment flag for one single-file and one
   multi-file operation.
4. Run the response-policy matrix, then choose a default cap and continuation
   representation from completion and efficiency evidence.
5. If the result contract is accepted, record its invariants in an ADR and add
   CLI txtar coverage for emitted receipt fields and stale-continuation errors.

## Refactoring Handover Foundations

Sections above preserve the historical naming/receipt investigation. ADR-0055 naming harmonization has since landed; the remaining gates below concern typed descriptors and lifecycle rather than redoing that harmonization.

The audit supporting [RQ-0024's operation handovers](RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md#individual-resumable-handovers) found that ADR-0055 name harmonization is implemented, but richer typed input and live-target lifecycle need further work. Reuse this investigation rather than creating a parallel receipt architecture RQ.

### Audited gaps

`internal/operation/operation.go` currently supports only string, boolean, string-array and integer input contracts. Unknown input kinds fall back to strings in CLI/MCP construction. `ValidateSourceFields` in `internal/mcp/server.go` validates existence, not semantic/value compatibility. Add registration-time rejection of unknown kinds before new structured mappings become public.

Existing inline-variable receipts echo the removed binding; removal echoes removed symbols/cases; extract-variable echoes the old expression selector. A moved symbol lives at its destination, while requested source `file` describes the operation's origin. These can be useful historical evidence but are unsafe generic live-target continuations. The handovers must not copy those gaps into new APIs.

### Candidate shared descriptors

These types are proposals, subject to schema and semantic review. Native structured JSON is required; do not encode mappings as strings, positional tuples or parallel arrays.

| Descriptor | Named fields | Required semantics |
| :--- | :--- | :--- |
| `SymbolTarget` | `file`, `symbol`, optional `expected_revision` | Exact live declaration/binding target; secondary targets use `continuations: SymbolTarget[]`. |
| `ConstructSelector` | `file`, `in_function` or `in_declaration`, `kind`, optional `discriminator`, `construct_path`, `expected_revision` | Exact current-node selection; paths require a matching revision and kind/owner agreement. |
| `ParameterSpec` | `name`, `type`, optional `variadic` | Go declaration shape; not a caller default expression. |
| `ResultSpec` | `type`, optional `name` | Result declaration shape; named results require body/defer analysis. |
| `ParameterMapping` | optional `source_index`, required `parameter: ParameterSpec`, optional `argument` | Ordered array defines final order. `source_index` is zero-based; duplicates/range errors refuse. New arguments are checked in each caller scope and preserve evaluation order/count. |
| `FieldSpec` | `name`, `type`, optional `tag` | Field declaration shape, separate from callable parameter names. |
| `ParameterToFieldMapping` | `source_index`, `field: FieldSpec` | Zero-based existing parameter mapped to a generated field; parameter object takes `fields` rather than redefining `parameters`. |
| `MemberTarget` | `file`, `symbol` | Exact resolved existing member identity; no reconstruction from member spelling. |
| `MemberMapping` | `source_member`, `target_member` | Named resolved member mapping; field initialization/evaluation policy remains operation-specific. |
| `CallSiteSelector` | `file`, `in_function`, `discriminator`, optional `construct_path`, `expected_revision` | Structured `call_site` selects a caller independently of primary callee `file`/`symbol`; requires a shared call-node taxonomy gate. |

Use a closed recursive schema that supports properties, required sets, arrays/items, enums, constraints and explicit nullability. Generate CLI JSON decoding, direct MCP schemas, batch child validation, examples and output continuations from the same definitions. Treat missing, null, empty arrays, index zero and false as distinct; use presence-aware optional decoding. Conditional requiredness, unsupported fields and output-branch availability must be checked explicitly.

When a new mapping carries a named type, import alias or caller expression, validate resolution in the actual destination/caller file. A textually identical expression does not establish identical object identity or scope. Initially restrict mappings to cases whose evaluation and binding can be proved; expose richer per-call mappings only through a separately validated typed schema.

### Receipt lifecycle and selector gate

1. Preserve exact names and value formats at every declared `SourceFields` edge. Check schemas/roles, branch presence and lifecycle eligibility, not field existence alone. Extend nested path/schema-reference support intentionally for `continuations`; the current top-level convention is insufficient.
2. Emit proposed `expected_revision` in both consuming inputs and source results. It binds the selected file's current revision, not the full compiler input set. Fresh unambiguous symbol resolution can omit it; a structural path recovered earlier requires it. A mismatch fails before selecting another node.
3. Capture working child evidence when an operation applies; re-resolve final live continuations after normalization/publication. Track target identity across later rename/move/remove children or omit unresolved targets. Keep historical operation inputs and deleted targets separate from continuation projections.
4. For a move, a live `{file,symbol}` record uses the destination file. For inline/delete, omit removed live symbols and return surviving owner/construct targets under explicitly named projections. For expression extraction, return the new binding, not the old call-argument discriminator as if it still described that use.
5. Distinguish refused/no working edit, applied working edit, persisted edit, post-publication diagnostic failure and partial publication. Do not invent success on missing/failed diagnostics, and do not infer zero mutation from a generic error.
6. On partial publication, derive continuations/diffs from actual disk bytes and invalidate retained analysis state. Only a proven surviving target can be offered as the next-call input.

### Foundation implementation and proof sequence

* First add closed structured schema representation, parsing and CLI/MCP parity with unknown-kind rejection. Prove missing/null/empty/zero/false distinctions, nested unknown fields and compatible `SourceFields` shapes through public schemas/CLI failures.
* Then add revision-aware inspection/ambiguity projections, canonical owner fields and target identity tracking. Prove stale selector refusal after an earlier batch insertion and live helper/member continuation chains without qualified-name reconstruction.
* Repair existing removal/inline/extraction/movement continuation eligibility before using their result fields as new operation inputs. Preserve historical operation evidence without claiming deleted/moved-from targets are live.
* Integrate final receipt resolution with the batch coordinator; prove rename→move→remove can invalidate an earlier receipt, and failure/partial publication yields honest final target availability.
* Share the validated all-output planner and read/write scope rules from RQ-0024. Keep local replacement read inputs distinct from writable module outputs; sourcebuf containment alone is insufficient authorization.

The foundation worker owns `internal/operation/operation.go`, parsing/CLI definitions, `internal/mcp/server.go` schema generation, continuation provenance and receipt finalization. Operation workers consume the reviewed contract instead of independently inventing schemas. The existing diagnostic serialization and navigation-only coordinate rules of ADR-0055 remain intact. Record accepted schema/receipt decisions in ADR-0055 or the relevant accepted receipt ADR with atomic index updates; keep this RQ Open until evidence resolves its outstanding choices.

### Model-facing description audit

The follow-up review checks ADR-0007 intent routing and ADR-0055 sections 1–3 and 7–8 against generated descriptions, parameter provenance, and live-result meaning. Matching names alone is insufficient. This investigation concerns the observable flow **select target → express edit → consume result → continue**. Description conformance does not establish how a model reasons internally or guarantee tool selection. Keep empirical model/harness evidence separate from schema conformance.

#### Findings from the current handovers and registered surface

* The handovers already state target/owner roles, batch requirements, and candidate receipt names. Most initially specified that descriptions must be added without supplying their actual wording. Each RQ-0041–0059 now has candidate tool text, parameter provenance/creation exceptions, and description acceptance criteria.
* Registry operation summaries generally contain ADR-0007's semantic action and priority over manual/text editing. `parameterDescription` in `internal/mcp/server.go` generates named producer-field clauses from `SourceFields`; this implements ADR-0055 Direction A mechanically for populated entries.
* The same file's `toolSchema` appends field names with generic “copyable” labels. That alone does not identify compatible follow-up tools, mode/branch restrictions, or selector requirements. Direction B needs explicit result→consumer guidance generated from validated continuation contracts, rather than inferring every output string is reusable.
* `internal/operation/wire_extract.go` assigns qualified existing-symbol sources to simple new-name inputs for function/local extraction. A model following that copy instruction can submit a receiver-qualified existing symbol where a fresh simple identifier is required. Creation-name inputs need caller intent, not those producer edges; the created qualified output feeds existing-target tools instead.
* `wire_engine_inline.go` uses `semantic_inline_variable.symbol` as a producer after the binding is deleted. Removal advertises removed symbols/cases/paths, and extraction advertises old expression text. Those fields cannot support live-copy instructions. Remove or condition these edges after lifecycle repair; keep historical evidence explicitly separate.
* `ValidateSourceFields` checks advertised field existence, not name/value compatibility, target role, surviving branch, or revision. Descriptions can be mechanically generated and still be wrong. Strengthen the shared gate before wiring new operations.

These observations were checked in this worktree and the pending main checkout. They are foundation implementation tasks; this documentation review does not modify the outstanding source migration. Do not propagate stale worktree batch restrictions into descriptions of the safely integrated baseline.

#### Description contract and reusable wording

Tool descriptions lead with the developer's semantic action and when to choose it over manual/text edits (ADR-0007). Then state the bounded supported scope and essential defaults/refusals. End with the exact live result fields and named compatible next tools (ADR-0055 Direction B). Internal package names, transport mechanics, and long unsupported-case inventories belong in implementation documentation unless they change the caller's decision. Prefer a few complete sentences over repeating the entire schema.

The following templates apply to every operation handover. Replace the producing tool/field with a validated edge, never an invented output. Examples naming proposed fields become public only after those producers and consumers exist.

| Input class | Required model-facing wording and provenance |
| :--- | :--- |
| Existing declaration `symbol` | “Target: qualified existing [declaration kind]. Copy verbatim from the `symbol` field of `semantic_lookup` or [validated live producer], with the `file` from that same target record.” Validate the accepted declaration kind and qualification. |
| New target `symbol` | “Target: new [simple identifier / qualified field identity] to create. Choose a fresh name in the documented format.” Do not claim an existing object or the preceding created object supplies a fresh name. Its generated output can feed existing-target consumers without feeding this creation input. |
| `file` | “Selector: exact workspace-relative source file for this target or owner. Copy the `file` field of [named compatible producer] with its associated live target/selector.” A move result uses the destination. A new destination choice is a different role. |
| `in_function`, `in_type`, `in_declaration` | “Selector: qualified existing [owner kind], not the new target. Copy the identically named field from [named live owner projection].” A bare `symbol` result is not an automatic differently named owner projection; add the engine projection instead of telling the model to reconstruct it. |
| `discriminator` | “Selector: exact current source text of [expression / statement / literal / switch]. Copy the `discriminator` field of [named live matching-node producer].” Preserve whitespace and node context; historical selected text is not a copy edge. |
| `construct_path`, `expected_revision` | “Selector: current zero-based path from [named matching candidate projection]; copy it with that selected file's `expected_revision`.” Revision wording must say “Precondition: opaque selected-file revision; copy verbatim from [named same-file producer].” Never mix records, reuse old paths, or substitute a full read-set digest. |
| `destination_file`, `target_symbol` | Describe the destination/anchor role and conditional requirement, including the exact default. A destination `file` field is not automatically advertised under another name; expose a compatible same-named projection or take caller intent. |
| `source`, `comment`, `type`, mappings and new accessor/field names | Describe the exact caller-authored replacement format, including delimiters, scope, typed mapping shape, and empty/null behavior. Use a producer only if it emits that exact value in the same role and context. Current `comment` can feed `comment`; deleted text is evidence. |
| `kind`, `scope`, `placement`, Boolean and generation policies | State whether it selects a mode, constrains scope, or chooses behavior; list accepted values, required/default behavior, and relevant false/empty distinctions. Lookup's descriptive `kind` does not feed a helper-kind or receiver-mode enum. User choices need no fabricated producer. |
| Nested `call_site`, `members`, `fields`, `parameters`, `continuations` | Name the object/array role and its exact typed field meanings. Copy an eligible complete record from a named same-shaped projection; do not rebuild receiver/type qualification or map positional tuples. Registry support for nested paths must precede advertised `SourceFields` edges. |

The handovers identify candidate inspection producers as `semantic_inspect_symbol.in_function`, `.in_type`, `.in_declaration`, `.members`, `.call_site`, `.discriminator`, `.construct_path`, `.expected_revision`, `.target_symbol`, `.contract`, and `.contract_file`. These are proposed role-specific extensions, not existing output guarantees. The foundation must settle how the inspection request selects the owner, member set, call, literal, anchor, or interface context, then emit only eligible complete projections with their associated file revisions. Refuse an unavailable or ambiguous context instead of returning a guessed copy value. If a different inspection operation owns a projection, update its source ID, description, schema, and handovers together before registration. Existing qualified `symbol` and `file` outputs remain usable by compatible existing-target consumers.

Direction B should say, for example: “Pass this live callable continuation's `file` and `symbol` directly to `semantic_replace_body` or `semantic_rename`. Copy `expected_revision` only when accepted by that consumer. The generated qualified method symbol is not this extraction tool's new simple-name input.” Descriptions must expose remaining required parameters as caller choices or explicit same-named projections. A compatible pair is a next-call target, not necessarily the entire next request.

Navigation `line`/`column`, diagnostic evidence, old locations, removed declarations, old expressions, and descriptive categories are labeled as such. Do not classify them as copyable by field presence alone. Existing extraction `start_line`/`end_line` are explicit current statement-boundary selectors, not lookup navigation fields renamed by the model.

All edit descriptions inherit the published batch behavior: dependent calls see the current overlay; normalization and diagnostics follow the coordinator's documented boundary. Do not promise clean diagnostics or disk writes after each child, whole-batch rollback, or automatic receipt interpolation. Required user source remains a caller choice; discourage rereads only when the existing receipt supplies the needed live context.

#### Acceptance and evidence gates

1. Inspect the emitted `tools/list` tool descriptions, parameter descriptions, and result schemas together, plus generated CLI help. Registry summaries alone miss generated misleading copy clauses.
2. For every advertised edge, validate the producer tool and field/path, identical name and accepted shape/value format, role/context, live branch, selected-file revision, and complete companion selectors. New-name inputs without producers are valid; blanket nonempty `SourceFields` requirements are not.
3. Execute a public producer call followed by a compatible consumer using verbatim returned values. Cover qualified methods, local-owner context, cross-file moves, generated methods/types, and refreshed construct selectors. No receiver concatenation, field renaming, offset conversion, or copying old deleted targets should be necessary.
4. Negative cases must reject or omit stale/removed fields, wrong-kind descriptive values, qualified-to-simple creation names, mismatched path/revision pairs, unavailable nested projections, and refused/partially published target branches. Assert emitted descriptions do not instruct those copies.
5. Compare static batch descriptions with actual capability metadata and failure behavior. A sequential receipt→next-call proof is separate from a batch with fixed inputs; read-only lookups remain outside existing batches.
6. Model-flow claims require visible harness trajectories: record chosen semantic tool, argument validity, unnecessary rereads/reconstruction, successful next call, and task outcome. Compare using the existing RQ-0014/RQ-0015 benchmark methodology with controlled model, task, harness and fixture. Do not treat a prose review or passing schema test as evidence of a model's internal reasoning, and do not start paid runs merely to complete this documentation audit.

Accept description/provenance fixes with the shared foundation worker, then operation owners adapt the candidate text to their proven release scope. This extends the existing investigation; it creates no new accepted ADR invariant.

## Holistic tool descriptions and overlap audit, 2026-10-02

### Scope and evidence

The initial discovery capture used main `32e2184`, the full Go profile, and live reload: 26 registered operations plus batch, metrics, reload, and feedback. The reviewed foundation subsequently merged into main at `c74dede`. A fresh `tools/list` capture at that commit, with an explicit Go/Java allowlist in an empty discovery workspace, advertises 28 registered operations plus those four server tools. Java contributes the two Maven operations. These counts describe these discovery configurations; language detection and profiles can hide operations. The ignored local evidence files are `.scratch/parallel-tools/catalog-audit-main.json` and `catalog-current.json`.

The audit uses accepted ADR-0007 for intent routing, ADR-0009 for discovery scoping, ADR-0019 for insertion tiers, ADR-0034 for registry ownership, ADR-0046 for current construct replacement, ADR-0051 for inspection, and ADR-0055 for parameter and continuation contracts. ADR-0053 remains Proposed: its future body-tool rename and broader whole-construct support are design options, not current promises. ADR-0052 supersedes ADR-0042's ordinary timing payload requirement.

### Complete registered routing map

Tool names in this table omit the `semantic_` prefix, except `report_feedback`.

| Family and tools | Selection rule and overlap assessment |
| :--- | :--- |
| `lookup`, `outline`, `inspect_symbol`, `find_references` | Locate a named declaration; inventory body-elided hierarchy; read complete declaration/signature/comments; find typed uses. Outline should explicitly say bodies are elided. Inspection is the source/revision producer. References are usage evidence, not another declaration inventory. |
| `insert_function`, `insert_type`, `insert_decl` | Create a complete function/method, named type/alias, or package constant/variable respectively. Choose the specialized declaration tool when its kind matches. `insert_decl.overwrite` overlaps `replace_decl`; the latter is the direct replacement intent. |
| `insert_declaration`, `insert_structure`, `insert_case` | Generic top-level declaration placement; a broader multi-kind insertion interface; or switch-case insertion. The generic declaration description already prefers specialized tools. `insert_structure` needs equally explicit routing and kind-dependent requirements to explain its remaining use. |
| `replace_body`, `replace_construct`, `replace_decl` | Replace callable statements while retaining its signature; replace a supported nested control-flow construct; or replace a package constant/variable/type alias. Current construct replacement does not imply whole-function, struct, or method replacement. |
| `extract_function`, `extract_variable`, `inline_variable` | Create a helper from selected statements; name an expression; replace a local binding's uses and remove it. New names are caller choices. The extracted live declaration can feed later existing-target edits; removed bindings and replaced expressions cannot. |
| `rename`, `move_construct`, `remove_construct` | Change binding identity through the language service; relocate the selected node; or delete it. Moving a declaration is distinct from renaming it. A safe-delete policy is a mode of removal; its execution boundary must be distinguished from ordinary buffered removal. |
| `scaffold_file`, `organize_imports`, `add_build_dependency` | Create a source file, rewrite source imports, or change module dependencies. Import organization does not install modules. Scaffolding's package name and dependency's module path/version are different values despite their current shared parameter spelling. |
| `assertion_mode` | Convert supported Go test assertions between fail-fast and continue-on-failure modes. Dry-run previews differ from trusted writes. This is a focused test-policy transformation, not general body replacement or verification. |
| `verify`, `maven_compile`, `maven_test` | Check/optionally normalize supported sources; invoke the fixed Maven `test-compile` goal; or invoke the fixed Maven `test` goal. Maven commands require trust and their documented offline/network policy. They are not batch children. Source diagnostics and build/test execution are distinct choices. |
| `snapshot`, `undo` | Capture explicit restoration state or perform conflict-checked restoration. These are recovery operations, not implicit batch rollback. Their machine-consumable snapshot identity should be assessed separately from human result text. |
| `batch` | Execute statically supplied operation requests in order. Dependent buffered edits see current source overlays; barriers and final normalization follow coordinator policy. Earlier successful edits may remain applied after failure. Result fields are used in later requests, not interpolated automatically into later children. |
| `metrics`, `reload`, `report_feedback` | Read opt-in recent timing evidence, refresh a development server, or record tool feedback. These operational tools do not overlap semantic source transformations. Reload is present only when enabled. |

### Findings and repair order

1. **Incorrect copy guidance can produce invalid edits.** Main extraction creation-name inputs still list existing qualified-symbol producers, although they require fresh simple identifiers. Inline/delete results advertise deleted targets, and expression extraction echoes old expression text as if it were an eligible current discriminator for other construct kinds. Remove those producer edges; preserve historical evidence separately. Validate role, value format, companion selectors, node kind, and liveness in addition to field existence.
2. **Moved targets have an incorrect live source-file pairing.** The current move operation records a destination, but the advertised copyable pair uses `symbol` with the original `file`. A subsequent existing-target request needs a live `{file, symbol}` record whose `file` is the destination. Keep origin and requested destination as explicit operation evidence. Requiring the model to rename `destination_file` into `file` violates the intended automatic flow.
3. **Insertion routing is insufficiently distinct.** Explain why `insert_structure` should be selected when specialized declaration/case tools exist, and encode conditional selector/placement requirements in its schema. Preserve meaningful differences between complete `source`, callable `body`, and case-clause input. A general-purpose schema should not imply every kind accepts every placement or anchor.
4. **Shared concepts have inconsistent names or vocabularies.** Generic declaration `visibility` and specialized `access_modifier` represent the same concept. Structure insertion uses `target_symbol` for some case anchors whereas the dedicated case tool uses `target_case`. Const/var kind spellings also differ between insertion and removal. Select one vocabulary through an accepted contract change, updating every ingress, output, example, and producer edge together. Rename dependency's module path/version concept separately from a source package name.
5. **Discovery schemas retain unsupported choices.** Go-only reference discovery exposes other language enum values and unrelated backend configuration. Go-only structure insertion exposes access values unavailable to Go. Apply ADR-0009's supported-language intersection to parameter branches, not only tool visibility. The Go discovery capture proves these advertised choices; it does not establish behavior of every other backend/profile configuration.
6. **Read results need typed continuation context.** The new foundation exposes captured inspection continuations and selected-file revision guards. It does not supply every proposed member, call-site, expression, or composite-owner producer. Legacy outline/reference result strings still hide useful structured information. Add only actual supported projections; do not use handover sketches as existing guarantees.
7. **Descriptions should explain required companions compactly.** Prefer a compatible live target record over long flat lists of producers. State which same-record file/owner/revision accompanies a symbol or construct selector and which remaining arguments are caller choices. Declared source fields currently prove less than this full semantic compatibility contract.
8. **Accepted examples need refresh.** ADR-0019 examples refer to old statement/declaration arguments and an unavailable supported-locations tool. Update them to the actual accepted public surface or label them as proposals. Keep ADR-0053's proposed replacement expansion visibly separate from current behavior.

### Pending public tools and overlap

At this audit checkpoint, contract extraction/implementation and statement/comment tools have public worker implementations under review; signature, relocation, and extraction tracks are still advancing independently. They are not all merged public features. RQ-0060 records implementation status and semantic gates.

The intended additional routes are: `extract_interface` derives signatures from selected existing direct methods; `implement_contract` creates missing method implementations; `set_doc_comment` changes owned documentation; and `insert_statement` adds one supported statement adjacent to a selected statement. Their descriptions must distinguish these intents from `insert_type`, `insert_function`, `replace_body`, and `insert_structure`, and disclose bounded support. Review already found and corrected contract aliases that bypassed live receipt checks, same-file generated-method revision evidence, structural manifest capture after analysis, and no-op read-set drift. Passing worker checks is followed by combined integration checks before main delivery.

This audit verifies visible tool selection guidance and argument/result continuity. It does not measure internal model reasoning or demonstrate lower token use. Producer-to-consumer execution and negative liveness cases are the acceptance evidence; any model-routing comparison belongs to the existing benchmark methodology and requires separately authorized runs.
