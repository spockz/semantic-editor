<!-- This open research question records the evidence needed before adding a Bash backend. -->
# RQ-0037: Bash Language Server and Construct Editing

* **Status**: Open
* **Category**: Language Backends & Semantics
* **Date**: 2026-09-25

## Question

Can a trusted, file-scoped Bash backend use `bash-language-server` for lookup and diagnostics while a parser safely selects and replaces `loop`, `if`, `else`, and `case` constructs?

## Current evidence

[`bash-language-server`](https://github.com/bash-lsp/bash-language-server) runs over LSP and documents declarations, references, document symbols, diagnostics, and formatting. It uses a [Tree-sitter Bash grammar](https://github.com/tree-sitter/tree-sitter-bash). ShellCheck and shfmt are optional programs that the server can invoke when installed. The server documents Node 20 or newer as a runtime requirement. These capabilities establish a plausible lookup path; they do not establish a construct replacement API or guarantee that its symbols cover nested branch syntax.

A Bash file may contain nested `if`/`elif`/`else`, `for`/`while`/`until`, `case` arms, functions, command substitutions, and heredocs. Text delimiters inside quotes or heredocs must not be mistaken for structural boundaries. The proposed file extensions are `.sh` and `.bash`; extensionless scripts and other shell dialects need an explicit policy.

## Investigation

1. Probe bounded stdio initialization and selected-file document symbols with a pinned, preinstalled server. Confirm exact symbol kinds, ranges, UTF-16 offsets, timeout behavior, and whether project files or optional tools are executed. Gate process launch behind request-scoped workspace trust per ADR-0023.
2. Compare the language server's parse tree access with a pinned Bash parser for structural targeting. Define discriminator and `construct_path` behavior for nested branches, including `elif` and terminal `else`.
3. Test malformed snippets, heredocs, quoting, comments, and sourced files. Require prewrite parse validation and atomic updates before advertising a mutation capability.
4. Add CLI txtar contracts for every operation actually registered. Keep unsupported constructs out of the active backend's schema.

## Decision gate

Select an ingress and parser only after a small selected-file prototype proves deterministic matching, bounded execution, and no mutation on invalid input. Until then, no Bash backend or construct capability is accepted.
