<!-- This open research question compares Makefile ingress and parsing before backend registration. -->
# RQ-0038: Makefile Language Server Versus Parser

* **Status**: Open
* **Category**: Language Backends & Semantics
* **Date**: 2026-09-25

## Question

Which parser can safely support replacement of `rule`, `if`, `else`, `variable`, and `include` constructs now that a bounded read-only lookup ingress is available?

## Current evidence

The proposed `makefile-language-server` name does not identify a verified distribution in this investigation. A concrete candidate, [`make-ls`](https://github.com/owenrumney/make-ls), documents stdio LSP support, document symbols for targets, variables, and conditionals, include resolution, and a custom parser. A separate [Tree-sitter Make grammar](https://github.com/tree-sitter-grammars/tree-sitter-make) is available; its existence alone does not prove mutation-quality parsing.

The scratch probe used pinned make-ls v0.1.22 over stdio. `initialize` reported UTF-16 and server version `0.1.0`; selected-file `documentSymbol` returned target records (`kind: 12`) with whole-rule ranges and name-only selection ranges. The six-second diagnostics observation produced no `publishDiagnostics`, so verification is not registered. Source review confirmed the server does not execute recipes, while its include resolver reads include paths directly, including absolute paths. The adapter creates an isolated source-only workspace but does not confine absolute or traversing relative includes outside the copied source.

The [GNU Make manual](https://www.gnu.org/software/make/manual/make.html) defines rules, variable assignment forms, includes, and conditionals. Its conditionals act on makefile text, so a conditional may split a larger rule across branches. Recipe lines normally use tabs but `.RECIPEPREFIX` may change the prefix. Includes can expand variables and wildcards. These features make a naive line or symbol-range replacement unsafe.

The v0.1.22 symbol probe confirms target and variable records can be mapped only when their name selection range matches a plausible declaration in the selected source. Conditional records use the full block range and a condition-expression name, so the read-only backend filters them rather than reporting a line in the selected file as their location. Included targets are also filtered when make-ls maps their range onto the selected file's include line.

## Investigation

1. The selected-file lookup probe is complete and accepted in ADR-0048. Broader filename variations and parser behavior remain to be measured before mutation.
2. Compare its parser with the Tree-sitter grammar on ordinary, pattern, static pattern, and double-colon rules; target-specific and multiline variables; nested conditionals; includes; comments; and non-default recipe prefixes.
3. Define whether `else` means a GNU Make conditional branch, not a shell `else` inside a recipe. Specify discriminator and ordinal path rules before exposing each construct kind.
4. Prototype in-memory validation and atomic writes. Add CLI txtar fixtures for every registered operation, including rejection without mutation for ambiguous or malformed input.

## Decision gate

Keep Makefile verification and construct capabilities unavailable until parser and include-boundary prototypes prove bounded deterministic behavior, safe conditional and recipe boundaries, and no invalid-input mutation.
