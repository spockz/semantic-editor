<!-- Records the proposed follow-up to ADR-0046 so replacement scope and unreleased API changes have one reviewable contract. -->
# Architecture Decision Record (ADR) 0053: Whole-Construct and Body Replacement

Status: Proposed
Date: 2026-09-27

Follow-up to [ADR-0046: Construct-Level AST Replacement and Declarative Updates](0046-construct-level-ast-replacement-and-declarative-updates.md).
This proposal is pending acceptance and implementation. ADR-0046 remains the current accepted contract.

## Context

ADR-0046 addressed localized control-flow edits that otherwise required regenerating
an entire function body, and package-level constant and variable updates that lacked
a replacement operation. It scoped `semantic_replace_construct` to constructs
inside a containing function and introduced `semantic_replace_decl` for constants,
variables, and type aliases. It did not establish a general replacement contract
for complete functions, methods, or classes.

The current `semantic_replace_body` is a specialized function or method body edit.
A function is also a construct, so whole-construct replacement and body replacement
can share a targeting model while offering different mutation scopes. The term
construct should not inherently exclude declarations.

The current discriminator selects the existing construct; it does not constrain
the replacement header. For example, selecting `if count > 0` and supplying
`if count > 10 { return true }` replaces the complete selected conditional.
A separate new discriminator is unnecessary because the new condition is already
part of the replacement source.

Discussion of moving `Server.IsRunning` before `Server.Start` exposed a separate
capability gap: insertion rejects an existing method, and replacement in place
does not relocate declarations. Alphabetical ordering and movement are distinct
from the replacement scopes proposed here.

## Decision

Propose two explicit operations with a shared selection model:

| Operation | Replacement source | Preserved scope |
| :--- | :--- | :--- |
| `semantic_replace_construct` | The complete selected construct, including its header or declaration signature | Surrounding constructs and sibling declarations |
| `semantic_replace_construct_body` | Only the contents of the selected body, without repeating its owning header or outer body delimiters | The owning header or signature, other body slots, and surrounding constructs |

Generalize supported construct kinds beyond inner control flow to functions,
methods, and type or class declarations where a language backend implements the
corresponding operation. This is a conceptual taxonomy, not a claim that every
language or kind is immediately executable. Advertise supported kinds and body
slots per operation and backend through the existing capability registry.

### Selection and replacement source

In both operations, selectors identify existing code only. Use symbol identity for
named declarations and an enclosing scope plus kind and discriminator or candidate
path for nested constructs. Whole-declaration replacement must not require a
containing function when the target is itself a function or type.

Whole replacement receives all syntax for the target, including a function's
header, receiver, parameters, results, and body. Body replacement receives only
body contents. For a function, it preserves the signature; for a loop, it preserves
the loop header; for a supported class body, it replaces member contents while
preserving the class declaration header.

Constructs with multiple bodies, such as `if`/`else` and `try`/`catch`/`finally`,
require an unambiguous body slot or selected branch. Body replacement changes only
that slot. Reject unsupported or bodyless targets before mutation; do not silently
turn body replacement into whole replacement or create a missing body.

### Unreleased API transition

Replace `semantic_replace_body` with `semantic_replace_construct_body`. The project
is unreleased, so do not retain the old tool or command as a compatibility alias,
add a deprecation period, or maintain dual schemas.

Update the engine, operation registry, Model Context Protocol (MCP) schemas,
command-line interface (CLI), batch dispatch, tests, documentation, and affected
fixture or control references together. This proposal does not decide whether
`semantic_replace_decl` should also be consolidated; its existing scope remains
separate until explicitly resolved.

### Explicit extraction and movement

Both replacement operations change code at its existing location. Neither
implicitly sorts, moves, or reinserts declarations. Replacing `Server.IsRunning`
with identical source leaves its position unchanged.

Provide explicit extraction and move tools as distinct refactoring intents.
[ADR-0007](0007-semantic-mcp-tool-naming-and-descriptions.md) already identifies
function and variable extraction, and [RQ-0024](../research/RQ-0024-ide-edit-and-refactoring-capability-taxonomy.md)
tracks extraction and movement as refactoring capability families. Their detailed
schemas and backend guarantees remain separate implementation work.

A move selects an existing declaration and a destination with placement intent.
Moving within the same package or file is valid: it can reposition the declaration
without changing its identity. For example, moving `Server.IsRunning` before
`Server.Start` in the same file expresses the required reordering directly.
Do not treat destination equality alone as a no-op; the requested position must
also already be satisfied.

The engine can implement this as removal and reinsertion of the existing syntax,
carrying attached comments and preserving the declaration contents. The caller
should not have to reproduce the function source or calculate line offsets.
Same-file relocation must commit as one atomic file update, without an observable
duplicate declaration or intermediate deletion. Explicit relative placement is a
general relocation capability, not an automatic ordering policy or a dedicated
sorting tool.

Extraction selects existing expressions or statements, creates a named declaration,
and rewrites the selected use site with the appropriate reference or invocation.
Its data-flow and control-flow checks distinguish it from insertion, copying, or
replacement. Moving across files or packages likewise needs explicit reference,
import, visibility, collision, and multi-file failure semantics. This proposal
records those tool boundaries without claiming that these broader refactorings
are already implemented.

### Project-owned ordering and normalization

Semedit must not implement automatic declaration ordering or comparable style
policies. Specialized formatters and linters own those rules and their optional
automatic fixes: for example, golangci-lint and project-selected Java or Scala
tools. Semedit may call out to an explicitly configured tool or supported language
server action, or the project may run its existing quality hooks independently.
The project owns tool selection, versions, configuration, and check/fix commands.
Do not reproduce those rules in insertion, replacement, or movement algorithms.
Explicit placement supplied by a caller remains supported; it does not authorize
implicit sorting of surrounding declarations.

Retain necessary syntax validation and existing formatter/import integrations,
while delegating style policy to the selected tools. A linter diagnostic alone is
not a supported automatic fix. Only invoke fixes that the external tool provides
and the project or caller has enabled.

Keep project normalization distinct from backend verification. Where configured,
a callout can run once after an edit or batch, followed by diagnostics and the
project's selected checks. Report the invoked tool and scope, changed files,
completion or failure, and remaining diagnostics. Unavailable tools, skipped
checks, timeouts, and partial external edits must not appear as successful project
verification. A language-server diagnostic result does not imply that project
linters, compilation commands, or tests have run.

Go verification formats selected sources unless check-only is requested, then
collects gopls diagnostics. Golangci configuration enables a filesystem lint
check; explicit project hooks can select additional checks. Separate builds and
tests require configured hooks. This repository's `make check` owns its broader
quality pipeline. Java
verification exposes bounded selected-file language-server actions and diagnostics;
its Maven build/test actions are separate. Other backends must report their own
advertised verification scope rather than inherit a Go-specific quality contract.

[ADR-0054](0054-project-verification-hooks.md) selects project-owned shell and
LSP hooks and defaults compilation/type checking and diagnostics to configured
language servers synchronized with committed edits. Filesystem lint checks run
after publication; mutating fixes precede fresh LSP verification.
[RQ-0028](../research/RQ-0028-semantic-quality-normalization.md) and
[RQ-0029](../research/RQ-0029-project-normalization-configuration.md) track the
external-tool policy and the hook execution contract. ADR-0054 supersedes
RQ-0029's earlier prohibition on command definitions. Its verification runtime
is independent of the pending replacement API changes proposed here.

## Invariants

1. Selectors locate existing targets; replacement source never doubles as a selector.
2. Body replacement preserves the owning header or signature and all unselected body slots.
3. Whole replacement affects only the selected construct and its subtree; it does not implicitly update external references.
4. Missing or ambiguous targets, unsupported kinds or body slots, syntax errors, and declaration collisions fail before mutation.
5. Preserve surrounding code and attached comments outside the selected scope, subject to declared formatting and import organization behavior.
6. Retain atomic writes with advancing timestamps and diagnostic reporting from ADR-0046.
7. Advertise only implemented language-operation capabilities. Maintain comparable CLI txtar coverage for each advertised language-operation pair.
8. Remove the old body-replacement interface when the new interface ships; no backward-compatibility alias is required.
9. Delegate automatic ordering and style normalization to project-selected tools or hooks; do not implement those policies in semantic editing operations.

## Consequences

The two operations express whether the caller wants to preserve or replace a
construct's header. Function-body editing becomes one case of a broader model,
while complete declaration replacement closes a gap left by ADR-0046. Body edits
retain their lower token cost and narrower mutation scope.

Implementations must distinguish statement bodies, member bodies, and individual
branch slots. Whole-declaration replacement also requires explicit handling of
symbol identity, collisions, and declaration comments. Renaming the unreleased
operation requires coordinated updates across all interfaces and callers.

## Details to resolve before implementation

- Final selector and body-slot field names, required-field combinations, and ambiguity diagnostics.
- Initial supported kinds for each backend, including the treatment of expression-bodied declarations and Go struct or interface contents.
- Whether whole replacement permits changing a declaration's name or receiver, and how resulting external-reference diagnostics are reported.
- Comment ownership at body and declaration boundaries, including annotations and compiler directives.
- Separate extraction and move contracts, including destination/placement selection, same-file reordering, and guarantees for cross-file or cross-package edits.

Validation must distinguish whole replacement from body replacement: changing a
condition through complete source must work, while a body edit must preserve that
condition. Include function signature preservation, complete declaration changes,
selection among multiple body slots, unchanged siblings, collision rejection,
bodyless-target rejection, and removal of the old CLI and MCP names.
