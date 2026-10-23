<!-- WHY: Records the architectural principle of lenient input resolution and canonical, precise output contracts across semantic tools. -->
# ADR-0065: Lenient In, Precise Out Input Resolution and Canonical Output Contracts

* **Status**: Accepted
* **Date**: 2026-10-08

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Accept directory scopes, kind synonyms including element vocabulary, outer block braces, plain-text headers, and casing-derived visibility; normalize before validation and advertise canonical kinds and strict continuation schemas. |
| description | Accept varied agent input while returning canonical output and continuations. |
| role | principle |
| scope | project |
| area | mcp |
| languages | generic |

## Context

Semantic editing tools are designed to separate model reasoning from AST manipulation. In practice, LLM callers exhibit varied formatting habits, use natural language synonyms, and provide coarse scopes:

1. **Path Scopes**: Callers frequently supply a package directory (e.g., `internal/symbol`) instead of an exact file to `semantic_lookup` or `semantic_inspect_symbol`. Rejecting these calls with `is a directory` aborts discovery and forces extra search turns.
2. **Vocabulary and Construct Kinds**: Callers use common synonyms (`kind: "variable"` instead of `"var"`, `kind: "constant"` instead of `"const"`) or specify `kind: "function"` for a receiver-qualified method (`Type.Method`). Rejecting these with `unsupported kind` stalls refactorings.
3. **Payload Delimiters**: When implementing `semantic_replace_body`, models frequently supply an enclosed `{ ... }` block matching standard function syntax. Rejecting the outer braces with `syntax error: expected declaration, found '}'` creates artificial editing failures. Similarly, `semantic_scaffold_file` has rejected plain text purpose headers that lacked explicit `//` comment prefixes.
4. **Visibility Modifiers**: `semantic_insert_function` has rejected insertions when `access_modifier: private` was supplied for capitalized Go identifiers (such as `Test*` functions), aborting valid additions despite unambiguous identifier casing.

These rigid rejections interrupt agent workflows and drive models to abandon semantic tools in favor of unconstrained text editing (`replace_file_content` or `write_to_file`).

## Decision

We adopt the robustness principle (**Lenient In, Precise/Strict Out**) across all semantic tools and ingress adapters:

### 1. Lenient Scope and Path Ingestion

* **Directory and Package Scope**: `semantic_lookup`, `semantic_inspect_symbol`, and `semantic_find_references` accept either an exact file path or a directory path. When a directory path is supplied, the engine resolves symbols across all matching source files within that directory/package.
* **Path Parameter Harmonization**: Tools accepting path selectors (such as `semantic_outline`) accept both `path` and `file` parameter aliases at ingress, normalizing them to the canonical workspace-relative path before execution.

### 2. Lenient Construct Vocabulary and Kind Normalization

* **Kind Aliasing**: Construct operations (`semantic_insert_construct`, `semantic_replace_construct`, `semantic_remove_construct`) map standard synonyms to their canonical internal construct kinds (e.g., `variable` $\to$ `var`, `constant` $\to$ `const`, `procedure` $\to$ `function`).
* **Literal Element Vocabulary**: Insertion normalizes `composite_element`, `literal_element`, `entry`, and `item` to `element` for unconstrained ingress. Advertised enums and output selectors use only `element`. These vocabulary synonyms do not retain the removed `insert_composite_element` operation or expand keyed Go map/struct support to positional lists.
* **Receiver-Aware Function Kind Inference**: When a target symbol includes a receiver identifier (`Type.Method` or `(*Type).Method`), construct operations treat the target as a method even when `kind: "function"` was supplied.

### 3. Lenient Code Payload and Delimiter Ingestion

* **Outer Brace Stripping**: `semantic_replace_body` accepts either bare statement bodies or enclosed statement blocks `{ ... }`. The ingress parser automatically strips outermost balanced enclosing braces before parsing and AST verification.
* **Purpose Header Comment Formatting**: `semantic_scaffold_file` accepts plain-text purpose headers. If comment prefixes are absent, the engine prepends standard comment formatting (`//`) automatically.
* **String Literal and Escape Sanitization**: Snippet parsers normalize raw string backtick delimiters and newline rune literals (`'\n'`) to prevent transport encoding crashes.

### 4. Lenient Access Modifier Ingestion

* **Identifier Casing Precedence in Go**: When Go declaration insertions encounter an explicit `access_modifier` that conflicts with the identifier casing (such as `private` for an exported `Test*` function), the engine infers visibility directly from Go casing rules rather than aborting the operation.

### 5. Uniform Direct and Batch Contracts

* **Same Operation Implementation**: An individual tool call and the corresponding batch child dispatch to the same registered parser, handler, finalizer, and projectors. The operation receives either a fresh request buffer or the batch's shared working buffer and uses the same code to read and mutate it. It must not branch on invocation mode to change aliases, normalization, validation, required fields, semantic transformation, refusal rules, or operation-level output. Batch errors may add the failing edit index, and its response may add a per-edit status and execution envelope.
* **Shared Output Rules**: Both paths use the registered operation's success and known-error projectors, which validate against the same strict result schemas. Batch reports the child's result against the batch working revision; it does not loosen the result schema or change the operation's canonical result shape.
* **No Silent Unknown-Field Dropping**: An unrecognized named parameter is not treated as an optional extension. Callers must use a declared parameter or alias so misspellings and unsupported options fail visibly.

### 6. Precise, Canonical Output Contracts

While ingress is lenient, tool outputs remain strictly canonical, precise, and deterministic:

* **Canonical Coordinates**: Results return exact project-relative file paths, line and column numbers, and fully qualified symbol identifiers.
* **Backend-Owned Normalization**: Each mutation follows its backend's deterministic formatting/import policy. Go uses in-process AST/formatter paths; Java uses pre-publication JDT LS formatting and import actions (ADR-0068). Shared ingress does not impose Go-native formatters on other languages or add formatting to operations that preserve an exact planned splice.
* **Strict Continuation Schemas**: Continuation records echo canonical parameter names and values matching target schemas exactly ([ADR-0055](0055-tool-parameter-schema-harmonization.md), [ADR-0062](0062-operation-specific-selector-continuations.md)).
* **Fail-Closed on True Ambiguity**: Leniency never guesses when an intent is genuinely ambiguous (e.g., multiple declarations matching an unqualified identifier in a directory). The engine fails closed and returns the exact set of candidate options with structural paths.

## Invariants

* Input leniency must never compromise semantic safety; ambiguous matches must fail closed with candidate choices rather than making heuristic selections.
* Individual operations and batch children must dispatch to the same registered operation code and share input normalization, parameter validation, semantic behavior, and operation-level output validation. The only operation-context difference is a fresh request buffer versus a shared batch buffer.
* Unknown parameters remain errors in every ingress path; only declared aliases and normalizations are lenient.
* Outputs emitted by the host engine must always be canonical, syntactically valid, and strictly typed.
* Downstream continuations emitted in receipts must use canonical parameter names, never ingress aliases.

## Consequences

* Eliminates common friction points that derail agent workflows during navigation, scaffolding, and construct replacement.
* Prevents unnecessary agent fallbacks to raw text-editing tools.
* The registry owns normalization, validation, and output projection so ingress adapters do not drift in strictness. Recognized aliases and delimiters do not relax internal AST validation.
