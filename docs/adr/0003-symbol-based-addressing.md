# ADR-0003: Symbol-Based Addressing Over File Coordinates

* **Status**: Accepted
* **Date**: 2026-09-15

## Context

Compilers and Language Server Protocol methods often require exact file coordinates (e.g. `path/to/file.go:42:15` or byte offsets). LLMs are notoriously poor at character counting and tracking line numbers, especially as earlier edits cause line-number drift.

In current agent-LSP setups, the model must spend 2–3 preliminary turns querying symbols and reading lines just to pass coordinates to a rename or code-action command.

This semantic-editing layer lets the model name the intended symbol directly instead of computing the location offsets required by a raw LSP request. It removes a separate coordinate-calculation step and a common source of location and line-drift mistakes.

## Decision

`semedit` decouples the agent from file coordinates where the active backend supports symbol addressing:

1. The agent addresses targets using semantic symbol identifiers (e.g. `--file pkg/auth/token.go --symbol "TokenService.Validate"`).
2. The selected language backend resolves the identifier to its exact native coordinate and feeds it into the underlying language server or compiler adapter. Go currently uses its existing AST resolver.
3. For multi-step interactions, `semedit` generates ephemeral session symbol handles (`sym_e4f91b`) that remain stable regardless of line shifts.

Go symbol addressing accepts direct short names scoped by `file`, receiver-qualified names such as `Type.Method`, and three-part `package.Type.Method` identifiers. A known file and symbol can therefore be sent directly to a mutation without a preliminary discovery lookup.

Backend-native lookup is also permitted when an adapter can resolve symbols more accurately than a shared resolver. This decision does not require a broken-source fallback engine; syntax-error recovery remains outside this foundation slice. Scope and path resolution follow the lenient in, precise out contract defined in [ADR-0065](0065-lenient-in-precise-out-input-resolution.md), accepting package directories as well as specific files.

## Invariants

* The LLM should never be forced to calculate or guess byte offsets or column numbers.
* Coordinate resolution must occur locally on the host before invoking an LSP or compiler adapter.
* A backend may use its native symbol lookup while preserving the language-neutral service contract.

## Consequences

* **Positive**: Eliminates the "coordinate tax", saves 2–3 turns per refactoring, prevents failed edits from line drift.
* **Negative**: Requires Tree-sitter grammars and symbol query definitions per supported language.
