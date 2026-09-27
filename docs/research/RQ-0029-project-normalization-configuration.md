# RQ-0029: Project Normalization Configuration

* **Status**: Resolved
* **Date**: 2026-09-21
* **Resolved**: 2026-09-27
* **Category**: Configuration & Quality

## Question

How should project-owned hooks select external commands and language-server
actions while preserving scope, ordering, and explicit failure reporting?

## Resolution

[ADR-0054](../adr/0054-project-verification-hooks.md) accepts a versioned
`.semedit.yaml` at the selected project root. The earlier candidate filename
`.semantic-editing.yaml` and prohibition on command definitions are superseded.
External tools own style and ordering policy; semedit orchestrates their execution.

```yaml
version: 1
verify:
  normalize:
    - id: go-lint-fixes
      languages: [go]
      timeout: 2m
      exec: [golangci-lint, run, --fix, ./...]
    - id: java-imports
      languages: [java]
      lsp:
        language: java
        action_kind: source.organizeImports
  check:
    - id: go-tests
      languages: [go]
      exec: [go, test, ./...]
```

Each enabled hook requires `id`, `languages`, and exactly one execution form:

* `exec`: executable and arguments as a list, without shell interpolation.
* `shell`: `executable`, `args`, and `script`. For example, `executable: sh`,
  `args: [-c]`, and `script: make check`. There is no implicit shell.
* `lsp`: `language` and either `action_kind` or `command`, with optional structured
  `arguments`. Execution is limited to backend-supported capabilities.

An optional `cwd` names an existing directory relative to the project root.
`timeout` is a positive Go duration up to one hour; the default is two minutes.
Unknown fields, duplicate IDs within a phase, unsupported versions, unknown
languages, and invalid execution combinations fail during discovery.

Lists execute in order. Explicit checks precede detected checks. An explicit
check with ID `golangci-lint` replaces the detected golangci checks, including
nested scopes. To disable them:

```yaml
version: 1
verify:
  check:
    - id: golangci-lint
      disabled: true
```

## Discovery and scope

Discovery runs against the resolved workspace after MCP initialization establishes
its root and Go base directory. It is refreshed at verification boundaries.
No parent `.semedit.yaml` is merged. Source and configuration signals select
checks; installed executables alone do not select filesystem lint checks.

Go uses its required gopls default. Bash and Make can be automatically selected
when matching files and their server executables are present. Other languages
need configured launchers or explicit selection. Unsupported selected verification
fails; detected unconfigured languages remain visible as excluded coverage.

Golangci configuration applies to its descendant sources until a nearer config
takes over. Each detected check uses an explicit config path and Go module working
directory, with buildable package arguments intersected with the selected sources.
Go LSP verification groups sources by module and reports each scan root, the
`./...` package pattern, and build exclusions. A narrower file selection may
require a module diagnostic pass; a selected subproject does not expand upward
to its parent module.

Language filters determine hook applicability. Commands receive their configured
arguments unchanged; semedit does not append source paths or interpolate shell
text. Project-authored commands therefore own their checking scope. Normalization
publication is independently limited to the requested file or directory scope.

## Execution and publication

Explicit verification and the outer batch boundary run the hook chain once.
Child-edit diagnostic passes do not re-enter it. Check-only skips normalization
and mutating server actions. Project configuration cannot grant workspace trust.
Applicable command hooks, including detected golangci checks, require explicit
workspace trust; unrelated and disabled hooks do not.

External normalization runs in a staged project copy. On success, semedit checks
all changed paths and publishes accepted files through the existing atomic-write
pipeline. Deletions, workspace-manifest edits, symlinks, special files, and changes
outside the requested scope fail before publication. A later publication failure
reports files already published. The staging directory is not an OS sandbox.

After normalization is published, fresh backend diagnostics run, followed by
filesystem check hooks on the committed project. Check hooks must be non-mutating.
Checks stop at the first failure, retaining completed results, exit status,
bounded stdout/stderr, changed paths, and partial completion. Cancellation and
hook deadlines propagate as failures. Required tools are never installed implicitly.

## Capability boundaries

The initial LSP hook implementation supports Java `source.organizeImports`
normalization. Other action kinds and named server commands fail explicitly;
configuration cannot grant a backend an unimplemented capability. Additional
actions require backend-owned selection, execution, and CLI coverage.

Gopls verification uses explicit pull results from a fresh session, preserving
severity and reporting incomplete coverage or protocol errors. Diagnostics do
not claim that a standalone build or test ran. Builds and tests are separately
configured checks. New error-severity diagnostics can fail an applied rename;
warnings and informational findings remain visible.
