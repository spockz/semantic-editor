<!-- This open research question compares Makefile ingress and parsing before backend registration. -->
# RQ-0038: Makefile Language Server Versus Parser

* **Status**: Open
* **Category**: Language Backends & Semantics
* **Date**: 2026-09-25

## Question

Which bounded ingress and parser can support file-scoped Makefile lookup and safe replacement of `rule`, `if`, `else`, `variable`, and `include` constructs?

## Current evidence

The proposed `makefile-language-server` name does not identify a verified distribution in this investigation. A concrete candidate, [`make-ls`](https://github.com/owenrumney/make-ls), documents stdio LSP support, document symbols for targets, variables, and conditionals, include resolution, and a custom parser. A separate [Tree-sitter Make grammar](https://github.com/tree-sitter-grammars/tree-sitter-make) is available; its existence alone does not prove mutation-quality parsing.

The [GNU Make manual](https://www.gnu.org/software/make/manual/make.html) defines rules, variable assignment forms, includes, and conditionals. Its conditionals act on makefile text, so a conditional may split a larger rule across branches. Recipe lines normally use tabs but `.RECIPEPREFIX` may change the prefix. Includes can expand variables and wildcards. These features make a naive line or symbol-range replacement unsafe.

## Investigation

1. Probe a pinned, preinstalled Makefile LSP on `Makefile`, `makefile`, `GNUmakefile`, and `*.mk`; confirm document symbol ranges, included-file behavior, timeout limits, and subprocess effects under workspace trust.
2. Compare its parser with the Tree-sitter grammar on ordinary, pattern, static pattern, and double-colon rules; target-specific and multiline variables; nested conditionals; includes; comments; and non-default recipe prefixes.
3. Define whether `else` means a GNU Make conditional branch, not a shell `else` inside a recipe. Specify discriminator and ordinal path rules before exposing each construct kind.
4. Prototype in-memory validation and atomic writes. Add CLI txtar fixtures for every registered operation, including rejection without mutation for ambiguous or malformed input.

## Decision gate

Choose LSP, CST, or a combined approach only after the prototype proves bounded, deterministic selected-file behavior and no recipe execution. Keep the backend and its construct enum unregistered until that evidence exists.
