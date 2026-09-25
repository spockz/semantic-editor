<!-- This open research question records the evidence needed before adding a Bash backend. -->
# RQ-0037: Bash Language Server and Construct Editing

* **Status**: Open
* **Category**: Language Backends & Semantics
* **Date**: 2026-09-25

## Question

Can a structural Bash parser safely select and replace `loop`, `if`, `else`, and `case` constructs after the trusted read-only backend is established?

## Current evidence

[`bash-language-server`](https://github.com/bash-lsp/bash-language-server) runs over LSP and documents declarations, references, document symbols, diagnostics, and formatting. It uses a [Tree-sitter Bash grammar](https://github.com/tree-sitter/tree-sitter-bash). ShellCheck and shfmt are optional programs that the server can invoke when installed. The server documents Node 20 or newer as a runtime requirement. These capabilities establish a plausible lookup path; they do not establish a construct replacement API or guarantee that its symbols cover nested branch syntax.

The scratch probe used bash-language-server 5.8.0 with `start`, Node 26.9.0, and ShellCheck 0.11.0. A selected `.sh` document returned flat symbol records: a function (`kind: 12`) with a `location.range`, and a local variable (`kind: 13`) with a `location.range` and `containerName`. It published an explicit empty diagnostics array for the opened URI at version 1. The probe did not run the fixture script; process descendants were not instrumented, so optional ShellCheck execution remains possible. The registered adapter is source-only, request-scoped, and UTF-16-aware; document symbols do not establish nested branch mutation boundaries.

A Bash file may contain nested `if`/`elif`/`else`, `for`/`while`/`until`, `case` arms, functions, command substitutions, and heredocs. Text delimiters inside quotes or heredocs must not be mistaken for structural boundaries. The proposed file extensions are `.sh` and `.bash`; extensionless scripts and other shell dialects need an explicit policy.

## Investigation

1. The selected-file lookup and diagnostics probe is complete and accepted in ADR-0048. Future parser work must retain request-scoped trust and bounded execution.
2. Compare the language server's parse tree access with a pinned Bash parser for structural targeting. Define discriminator and `construct_path` behavior for nested branches, including `elif` and terminal `else`.
3. Test malformed snippets, heredocs, quoting, comments, and sourced files. Require prewrite parse validation and atomic updates before advertising a mutation capability.
4. Add CLI txtar contracts for every operation actually registered. Keep unsupported constructs out of the active backend's schema.

## Decision gate

Keep Bash construct capabilities unavailable until a parser prototype proves deterministic nesting/discriminator behavior, malformed-snippet rejection, and no mutation on invalid input.
