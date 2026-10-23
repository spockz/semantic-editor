<!-- WHY: This ADR records the accepted safety boundary for selector suggestions and the invariants that govern their use. -->
# ADR-0071: Bounded Native Symbol Suggestions

* **Status**: Accepted
* **Date**: 2026-10-16

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Return up to three scope-bound native alternatives through shared structural preflights and language-server lookups; retain rename and invalidation identities across batch barriers and keep original errors authoritative. |
| description | Misspelled or under-qualified symbol selectors often produce a correct native failure but no efficient next step. An approximate text match alone is unsafe: it can cross file or receiver scope, select the wrong declaration kind, or mistake a provider's index for current source truth. Suggestion providers must remain optional and must not weaken operation validation or introduce hidden workspace writes. |
| role | contract |
| scope | source-code |
| area | backend |
| languages | generic |

## Context

Misspelled or under-qualified symbol selectors often produce a correct native
failure but no efficient next step. An approximate text match alone is unsafe:
it can cross file or receiver scope, select the wrong declaration kind, or
mistake a provider's index for current source truth. Suggestion providers must
remain optional and must not weaken operation validation or introduce hidden
workspace writes.

## Decision

On eligible native target lookup failures, semedit may retrieve a bounded set of
candidate selectors, rank them locally, apply scope and operation-specific
native validation, and return up to three alternatives with exact argument
replacements and source evidence. Suggestions are opt-out with
`SEMEDIT_SUGGESTIONS=off`; default `generic` mode enumerates native Go
declarations. Ordinary native target inspection is bounded to 1.5 seconds;
rename and reference-search validation may take up to 5 seconds for compiler or gopls startup.
Codanna retrieval is bounded to 500 milliseconds. Candidate ranking is
bounded to 50 items, selectors are limited to 256 bytes for operation inputs
and 256 runes for ranking, and at most three alternatives are returned.
Optional provider modes may retrieve candidates, with
`codanna,generic` combining provider and generic candidates. Codanna uses an
installed CLI on `PATH`, optionally selected with `SEMEDIT_CODANNA_BIN`.
Codanna CLI startup can synchronize its own index before dispatching commands
([upstream `src/main.rs`](https://github.com/bartolli/codanna/blob/main/src/main.rs#L3329-L3371)), so provider calls may update Codanna-managed index/cache state.
semedit does not install Codanna, models, or dependencies, nor explicitly run
an index or rebuild command. Suggestions never write source files.

The error remains an error until the caller explicitly retries. A valid target
does not invoke candidate retrieval. A provider result is never authoritative:
the selected semantic operation's normal target resolver and validation still
decide whether an edit can proceed. Returned MCP fields contain no provider
payload and are limited to the retry arguments, relative file, one-based
location, declaration kind, optional signature, and a short reason. Only Go
`lookup`, `rename`, `inspect_symbol`, `find_references`, `replace_body`, `remove_construct`,
`replace_construct`, `change_signature`, `extract_function`, and
`extract_variable`, `set_doc_comment`, `insert_field`, `extract_type`, and
keyed-element/field `insert_construct` participate in the Go implementation.
Structural operations register the same revision and source-plan preflight used
by execution; recovery never invokes their mutation handler. Trusted selected-file
Java lookup, inspection, reference search, and rename use the native JDT LS
outline and non-mutating backend validation. Alias history retains the language,
overload signature, and resource path of each accepted rename. Operations without
a native target preflight preserve their existing errors. Rust, Scala, Kotlin,
Haskell, Bash, and Make selected-file lookups use native document-symbol candidates and exact
lookup validation; Make symbol inspection also reuses its native inspector.
Destructive steps retain invalidation evidence across publication barriers,
preventing stale indexed identities from attaching to reused names.

## Invariants

* Candidate count, selector length, retrieval time, and provider output are bounded.
* Suggestions are returned only after the operation-specific native validator accepts a candidate in the selected file or package scope.
* Same-spelling declarations of the wrong receiver or kind are not proposed when they fail validation.
* No source or module manifest is written to produce a suggestion. A Codanna CLI search may maintain its provider-managed index/cache.
* Disabling suggestions preserves the native failure behavior.
* Batch publication failure removes candidate suggestions from rows whose overlay state is no longer safe to reuse, without changing successful prefix statuses.

## Consequences

Callers receive enough structured evidence to retry without a second broad
search. They remain responsible for accepting the alternative and resubmitting
the operation. External providers can improve recall but require an installed
CLI and maintained provider index/cache. Other language backends can be
added only with equivalent scope and target validation.
