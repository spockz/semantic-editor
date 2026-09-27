# ADR-0054: Project Verification Hooks

## Status

Accepted. The version-1 configuration and verification coordinator implement
this contract; supported LSP actions remain backend-specific.

## Date

2026-09-27

## Context

Projects already own formatter, linter, ordering, and test policy through their
build tools and language-server configuration. RQ-0028 documents existing
external ordering tools and funcorder's upstream autofix history. Semedit needs
one discoverable way to invoke project policy without implementing equivalent
rewrites or requiring the agent to rediscover each project's commands.

RQ-0029 previously proposed boolean normalization settings and excluded arbitrary
commands. The selected direction instead allows the project to specify the
shell commands and supported language-server actions it wants executed.

## Decision

Use a versioned `.semedit.yaml` at the selected project root to define ordered
verification hooks, including shell commands and language-server actions.
The project owns tool installation, versions, and tool-specific configuration.
Semedit owns hook selection, invocation, sequencing, and result reporting.
Declaration sorting remains in external tools.

Separate mutating normalization from checking so configured verification can run
normalization before diagnostics and tests, while a check-only invocation skips
normalization. Run configured hooks once at the explicit verification or outer
batch boundary, not recursively during each child edit's internal diagnostics.
By default, relevant source files enable their configured language servers for
compilation/type checking and diagnostics in the selected scope. Go support
requires gopls: if it is unavailable, Go operation cannot proceed. Do not provide
a degraded mode or substitute a `go vet` subprocess for the required LSP.
Unsupported verification capabilities must be reported explicitly.

Detection, server configuration, and capability support are separate decisions.
Go has a built-in gopls default. For automatic multi-language verification,
Bash and Make are also selected when their source files are detected and their
respective LSP executables are installed. Other detected languages require an
explicit server configuration or LSP hook selection; registry membership alone
does not select them. Missing Bash or Make executables prevent automatic
selection, while a missing explicitly selected server is an error.
Explicit language or file requests still select their backend directly.
Record detected but unconfigured languages as excluded coverage. A selected
server with unsupported verification fails explicitly; no selected server is
not a clean result. Aggregate completed results when a later server fails.

Defaults depend on project signals. Go sources and an applicable golangci-lint
configuration enable its filesystem check. Without that linter configuration,
do not invoke golangci-lint automatically. An explicit `.semedit.yaml` selection
can enable golangci-lint or another linter even without an automatically detected
linter configuration. Explicit settings can disable or override detected checks;
resolve a single effective check rather than running both copies. Tool presence
on PATH alone does not enable a check. An enabled check with an unavailable tool
fails explicitly. Automatic detection enables checks, not mutating fixes.

Absence of `.semedit.yaml` means source-selected LSP verification plus checks
enabled by recognized project signals. It does not preserve the current Go
`go vet` subprocess as the default.

Run language and verification-signal detection during MCP `initialize`, after
the workspace root and derived base directories have been resolved. Process CWD
may differ from the workspace directory and must not determine the resulting
plan. Keep discovery within that project boundary; applicable nested tool
configurations partition scope. Refresh the plan when relevant sources or
configuration change. ADR-0041 defines the initialization boundary.

Preserve server diagnostic severity. Informational findings and warnings remain
visible, but do not become compilation failures for applied renames; only newly
introduced error-severity findings do. Project lint policy is enforced by the
configured linter's outcome, as distinct from the LSP diagnostic delta (ADR-0004).

A language server's compiler diagnostics are not evidence that a standalone build
or test command ran. Request supported server-native compilation where available;
otherwise report the precise type-checking and diagnostic scope provided. Full
builds, tests, and project lint policy remain separately configured checks.

A shell hook is explicitly shell code. The execution contract must identify its
shell and working directory rather than infer an interactive user environment.
Prefer support for direct executable arguments as well, for hooks that do not
need shell syntax. Arguments and selected-file inputs must retain their boundaries.

LSP hooks distinguish requesting a code action by kind from executing a named
server command with structured arguments. The initial runtime supports Java
`source.organizeImports` normalization; other action kinds and named commands
fail explicitly until a backend implements them. Code actions must be resolved and
selected through the active language backend. A command must be supported by
that server and backend; neither transport is a universal raw-RPC escape hatch.
An edit-plus-command action applies its validated edit before its command.

This decision adds a distinct project-hook capability. It does not broaden the
fixed Java formatting, import, or Maven actions in ADR-0036 and ADR-0037, nor
make currently read-only backends support mutations automatically.

### Committed revision and verification sequence

LSP-produced edits and direct AST edits share the same publication boundary:
validate and atomically commit client-applied edits, then synchronize every
relevant active LSP document or workspace view with that committed revision.
An LSP producing an edit does not itself guarantee its publication to disk.

At an explicit verification or outer batch boundary:

1. Complete the semantic action and commit its affected files before invoking
   filesystem tools. Both LSP-buffer edits and direct file writes must have
   completed publication to disk before golangci-lint or another filesystem
   check starts; an action in a buffer or pending workspace edit is insufficient.
2. Run any configured normalization hooks in order. Before each filesystem hook,
   publish preceding LSP edits; before each dependent LSP action, synchronize
   preceding filesystem changes. Retain the external-write publication requirement
   below for tools that modify files themselves.
3. Obtain compilation/type-checking results and diagnostics from the configured
   LSPs for the final normalized revision and selected scope.
4. Run configured check hooks in order against that committed state. A
   golangci-lint check reads the filesystem after the action has been written;
   `--fix` belongs to the mutating normalization phase, followed by LSP
   synchronization and fresh verification.

Check-only mode skips normalization and mutating server actions. No-config mode
still obtains LSP verification. Internal child-edit diagnostics do not recursively
run the project hook chain.

Each required LSP must provide a backend-defined completion/result receipt for
the relevant revision and scope. Sending a save/change notification or waiting
for a watcher debounce is not proof of completed analysis. Stale diagnostics,
missing reports, partial server coverage, and timeouts must remain distinguishable
from an explicit clean result. Record which servers and checks contributed; a
clean result from one server cannot stand in for another required server.

Use normal supported server completion and diagnostic mechanisms to establish
this sequencing. Reuse the existing atomic-write contract and propagate actual
I/O and protocol errors. This work does not add independent fsync audits,
filesystem transaction isolation, or defenses against hypothetical unrelated
writers interleaving with semedit. Those concerns must not expand the scope of
the verification integration.

## Invariants

- Resolve one configuration from the selected project root. Do not merge parent
  repository configuration implicitly. Reject unknown fields and unsupported
  versions instead of silently ignoring them.
- Configuration expresses project policy but cannot grant itself workspace
  trust or override the execution environment's permissions. Every applicable
  command hook, including detected golangci checks, requires workspace trust.
  Disabled hooks and hooks outside the selected scope do not require it.
- Preserve configured order. Stop on failure; report completed hooks, changed
  files, and partial results. Missing executables, unsupported actions, and
  timeouts are failures, not successful no-ops.
- A supported code action with no applicable edit may be a successful no-op;
  unsupported capabilities and ambiguous matches must remain distinguishable.
- Shell edits must be synchronized with active language-server documents before
  a dependent LSP hook runs. Validate LSP edit scope and document versions and
  commit client-applied edits atomically.
- Run external normalization in a staged project copy. Validate all changed
  paths before publishing files with ADR-0010 atomic writes. Reject deletions,
  manifest changes, symlinks, special files, and writes outside the requested
  scope. Report any files already published if publication subsequently fails.
  Check hooks run on the committed project and must be non-mutating.
- Hook results identify the effective configuration, selected scope, execution
  outcome, and bounded output. Cancellation and timeouts cover the hook chain.
- Tools must already be available through project setup. Verification does not
  silently install tools or mutate workspace manifests.

## Consequences

Project policy becomes explicit and reusable from CLI, MCP, and batch workflows.
Default verification follows the configured language servers and synchronized
source revision instead of a Go-specific filesystem subprocess. Go diagnostics use fresh gopls pull requests for each selected module scan root,
retaining diagnostic severity and explicit server failures. Receipts identify
the `./...` package pattern and build exclusions; a selected subproject is not
silently expanded to its parent module.
Teams can use existing shell targets or server-native actions without teaching
semedit each formatting policy. Shell hooks also introduce project-code execution
and potentially broader edits, which require explicit scope and result semantics.

Version 1 uses ordered `verify.normalize` and `verify.check` lists. Each hook has
a unique `id`, language filters, an optional project-relative `cwd` and `timeout`,
and exactly one of `exec`, `shell`, or `lsp`. A disabled hook contains no
execution form. Direct `exec` arguments remain separate; `shell` identifies its
executable, argument list, and script. Named LSP commands retain structured
arguments. Unknown fields, unsupported versions, and invalid combinations fail.
RQ-0029 records the schema and remaining backend capability boundaries.
