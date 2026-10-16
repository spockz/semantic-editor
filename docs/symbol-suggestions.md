<!-- WHY: This document records the user-facing configuration and safety contract shared by CLI and MCP symbol suggestions. -->

# Symbol suggestions

Symbol suggestions give a small, evidence-backed retry hint when a supported
native target lookup fails. They never choose or execute an alternative. The
caller must review the file, selector, kind, signature, and reason, then retry
the original semantic operation with the suggested arguments.

Suggestions are currently available for Go `lookup`, `rename`,
`inspect_symbol`, `find_references`, `replace_body`, `remove_construct`, `replace_construct`,
`change_signature`, `extract_function`, `extract_variable`, `set_doc_comment`,
`insert_field`, `extract_type`, and keyed-element/field `insert_construct`
target failures.
Extraction alternatives replace the existing `in_function` owner; the requested
new helper or variable name stays unchanged. Candidate spelling is
retrieved from the workspace's native symbol inventory by default, ranked
locally, constrained to the requested file or package scope, and checked by
the operation's native target validator. At most three validated candidates
are returned from at most 50 candidates, with selectors limited to 256 bytes
for operations and 256 runes for ranking. Ordinary native target inspection is
bounded to 1.5 seconds; rename and reference-search validation may take up to
5 seconds for compiler or gopls startup. Extraction validation also uses a
5-second compiler budget, as do structural planners. Codanna retrieval is bounded to 500 milliseconds.
A candidate rejected by scope or operation-specific validation
is omitted. A valid target request does not trigger retrieval.

Native discovery inspects at most 256 files, 64 MiB in total, 2 MiB per file,
and 4096 symbols. Files beyond these limits may be omitted; indexed candidates
can still be checked directly outside the native discovery set. Batch discovery
can include current declarations, but newly inserted symbols are not guaranteed
to appear. Failed staging replay omits alternatives from the discarded view.
Destructive batch edits invalidate identity evidence; indexed candidates are
omitted when an old declaration cannot be distinguished from a reused name.

Trusted Java selected-file `lookup`, `inspect_symbol`, `find_references`, and
`rename` failures use the current native JDT LS outline for generic discovery.
They require the same installed server and workspace trust as an exact Java
request. Candidate selectors preserve overload signatures; containing-type and
method rename history remains scoped to the original identity. Java discovery
does not broaden a selected file into a workspace scan. Operations without a
native non-mutating preflight retain the original failure without alternatives.

Rust, Scala, Kotlin, Haskell, Bash, and Make selected-file lookup failures use the
backend's native document-symbol candidates. Make also supports symbol-inspection
recovery. These paths retain their normal workspace-trust and installed-tool
requirements. Native selectors, including qualified Rust paths, are returned
unchanged; no candidate widens the selected file scope.
Haskell retains the `standalone_haskell` requirement and does not gain a public
outline operation through suggestion retrieval.

Set `SEMEDIT_SUGGESTIONS=off` to disable retrieval. The default `generic` mode
uses native source or language-server enumeration and does not invoke an external search provider.
`codanna` selects the optional Codanna CLI provider;
`codanna,generic` combines provider results with native enumeration,
including when provider retrieval succeeds. Use an installed Codanna CLI on
`PATH`, or select another executable with `SEMEDIT_CODANNA_BIN`. semedit does
not install Codanna, its models, or dependencies, and does not explicitly run
an index or rebuild command. Codanna's search command may maintain its own
index/cache as part of normal CLI startup; this does not write source files.
Provider results are candidates only: each suggestion still has to pass scope
checks and the operation's native validation. semedit does not send source
code to Codanna.

MCP places suggestions in a typed `suggestions` array on the standalone error
envelope and on failed batch rows. CLI errors include compact candidate lines
after the original error. Unsupported languages and operations retain their
ordinary failure behavior. Configuration uses process environment variables
for both CLI and MCP.

The executable Go CLI contracts are in
[`testdata/scripts/symbol_suggestions_go.txtar`](../testdata/scripts/symbol_suggestions_go.txtar),
[`testdata/scripts/symbol_suggestions_go_scope.txtar`](../testdata/scripts/symbol_suggestions_go_scope.txtar),
[`testdata/scripts/symbol_suggestions_replace_body_filter.txtar`](../testdata/scripts/symbol_suggestions_replace_body_filter.txtar),
[`testdata/scripts/symbol_suggestions_batch_rename.txtar`](../testdata/scripts/symbol_suggestions_batch_rename.txtar),
[`testdata/scripts/symbol_suggestions_batch_delete.txtar`](../testdata/scripts/symbol_suggestions_batch_delete.txtar),
[`testdata/scripts/symbol_suggestions_batch_scaffold_barrier.txtar`](../testdata/scripts/symbol_suggestions_batch_scaffold_barrier.txtar),
[`testdata/scripts/symbol_suggestions_disabled.txtar`](../testdata/scripts/symbol_suggestions_disabled.txtar),
and [`testdata/scripts/symbol_suggestions_valid_target_no_provider.txtar`](../testdata/scripts/symbol_suggestions_valid_target_no_provider.txtar).
