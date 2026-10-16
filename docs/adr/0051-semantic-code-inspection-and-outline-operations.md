<!-- Records semantic read operations and the typed inspection evidence they expose for follow-up edits. -->

# ADR-0051: Semantic Code Inspection and Outline Operations

Status: Accepted
Date: 2026-09-26

## Context

Under read-restriction policies (such as `--semedit-arm-restrict=read|readwrite`), benchmark agents are steered toward semantic operations and prohibited from raw shell inspection. Existing semantic lookup provided point coordinates (`line`, `column`, `offset`), lacking declaration bodies, signatures, enclosing scope, and file context. Agents attempting semantic refactorings (such as replacing method bodies) refused to modify unseen code.

Additionally, agents lacked capabilities to discover declarations across unfamiliar files or directory hierarchies before knowing exact symbol identifiers, had no typed reference discovery mechanism, and experienced member lookup failures on Go interfaces (`Sink.Write`) and receiver filter parameter leakage.

## Decision

We introduce three dedicated, zero-token read operations across the core engine, CLI, and MCP servers:

1. **Symbol Inspection (`semantic_inspect_symbol` / `semedit inspect`)**:
   Returns complete source declarations, signatures, doc comments, enclosing package/imports, and content digests for queried symbols. Ambiguous queries return all matching candidates across the workspace in one structured response.

   MCP structured inspection also projects one typed continuation record for
   each captured match, keeping its project-relative `file`, qualified
   `symbol`, and `expected_revision` together. Go owner selectors are
   included in that same record when applicable. These values are copyable
   only for consumers declared by the operation contract; the legacy
   inspection text remains a separate representation and uses its own
   `revision` field.

2. **Hierarchical Outline (`semantic_outline` / `semedit outline`)**:
   Projects declaration hierarchies for a selected file or directory. Function and method bodies are elided to conserve context budget. Supports kind filtering (`kinds`), unexported symbol inclusion (`include_unexported`), and test symbol inclusion (`include_tests`). Native records preserve package documentation comments on empty extension points; agent navigation presents concise declaration summaries.

3. **Symbol References (`semantic_find_references` / `semedit references`)**:
   Separates declaration definitions from usages. Resolves call sites, parameter bindings, type usages, and field accesses across workspace source files.

We extend cross-language parity for these read capabilities:

- Generic Language Server Protocol (LSP) document symbol handling (`readlsp`) powers Java, Kotlin, Rust, and Scala outline projections.
- Dedicated adapters provide outline projections for Makefile and Bash scripts.
- The Go backend provides AST-native inspection, outline, and typed reference resolution.

## Invariants

- Read operations preserve workspace source and manifest bytes, their modification timestamps, and git-tracked state. Backend-owned scratch/runtime caches follow their own lifecycle; read-only semantic behavior does not promise that a language server performs zero disk I/O.
- Ambiguous symbol queries return complete candidate sets with distinct locations rather than aborting or arbitrarily selecting a single match.
- Hierarchical outlines omit executable statement bodies while retaining canonical signatures, doc comments, and parent-child declaration relationships.
- Go outlines propagate the request context through source selection and parsing,
  honor cancellation, and read the active buffered source view. Source discovery
  shares the request's Go environment cache rather than starting an independent
  background-context environment lookup.
  Complete explicit runtime paths permit shared source filtering without a Go
  subprocess; configuration-derived paths still require effective resolution
  (ADR-0032).
- Language backends declare read capabilities explicitly in the backend registry; unsupported combinations reject execution with structured errors.
- Every public language-operation combination maintains executable CLI txtar contract test coverage.

## Consequences

- Agents operating under read restrictions inspect and discover code structure through semantic tools without falling back to unconstrained shell commands.
- Context window consumption remains compact due to body-elided structural outlines.
- Language backends maintain consistent read and inspection contracts across Go, Java, Kotlin, Rust, Scala, Make, and Bash.

## Restored construct inspection and canonical coordinates: 2026-10-12

Go construct inspection returns ordered matching candidates, normalized syntax,
structural paths, and a selection revision. Guarded replacement rejects a stale
revision before writing and returns current typed candidate choices. A selection
revision describes that candidate set rather than the whole file.

Neutral backend locations use captured source and zero-based UTF-16 ranges.
Public lookup line and column fields use one-based UTF-16 coordinates; field names
remain stable, but Go byte-column behavior is deliberately canonicalized. Go
compiler and AST positions stay byte-based internally. Conversion to a byte
position uses captured source, never a best-effort disk reread. Staged lookup and
inspection therefore observe the same unpublished revision.

## Compact navigation projection: 2026-10-16

Lookup and its ambiguity candidates, declaration outlines, and reference results
share a normalized declaration summary and line-oriented renderer. Backend data
remains typed; the agent-facing projection carries the useful navigation listing
once, together with scope, completeness, source-view information, and required
continuation fields. Full serialized declaration trees are not paired with the
compact listing. This replaces the declaration-outline delivery requirement in
ADR-0066 without changing workspace package or module records.

CLI output and MCP text carry that listing. MCP structured results retain compact
status summaries, counts, scope, source view, and required resolved target fields.
The text listing therefore reaches clients that forward only text, while clients
that expose both MCP representations receive no duplicate bulk listing.

Locations resolve from the active workspace root. Declaration extents display as
one-based inclusive lines, converting native end-exclusive ranges correctly.
A start-only location remains a point. Descriptive signatures and exact native
selectors remain distinct; receiver and overload qualification must survive the
projection. Missing signatures or unsupported inspection are not invented.

References retain occurrence coordinates separately from declaration extents.
Lookup local bindings retain their own navigation-only section; they are neither
semantic references nor resolved inspection or mutation targets.
Grouping uses known enclosing declarations without inventing call relationships;
unresolved owners remain plain occurrences. Native result limits, ambiguity,
ordering, and coverage limitations remain intact. Snapshot metadata and selectors
describe one source view. Overlay groups identify unpublished source and require
overlay-aware inspection; they do not promise disk contents or live mutation
continuations.

Inspection remains the explicit source-bearing operation with its revision and
same-record continuation contract. Construct inspection retains anonymous
construct selectors and selection revisions. Registry review found no additional
search, caller, callee, or related-symbol operation. Existing ambiguous Go, Java,
and Rust rename alternatives and Go and Java reference alternatives use the same
declaration-summary conventions; adding a symbol suggestion
provider or an external search integration is outside this decision.
Typed ambiguous navigation errors retain their complete alternatives in MCP text;
the structured error carries their headline instead of duplicating the listing.
Error state and continuation restrictions remain unchanged.

Java source snapshots carry their disk or working provenance through lookup,
outline, and references. Reference target, occurrence, and revision paths are
workspace-relative; internal mutation planning resolves them against that root.
Flat language-server symbols without a declaration extent remain start-only
navigation points rather than presenting their identifier range as a declaration.
