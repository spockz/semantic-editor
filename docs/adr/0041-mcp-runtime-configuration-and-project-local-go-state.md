# ADR-0041: MCP Runtime Configuration and Project-Local Go State

Status: Accepted
Date: 2026-09-21

## Context

MCP clients can launch `semedit` from a benchmark fixture while configuration
often lives outside that fixture. Relative executable paths therefore resolve
unreliably, and inherited global Go caches can be unreadable or unwritable in
sandboxed sessions. A Makefile environment fixes only its own shell recipes;
it does not configure Go subprocesses launched later by an MCP server.

## Decision

MCP registration uses an absolute path to the semedit binary and starts the
server with the client project directory as its working directory. The server
offers optional initialization instructions and a `--go-base-dir` setting.
When no base directory is specified, it uses `<server-cwd>/.scratch/go`.

Before reading JSON-RPC input or advertising tools, the MCP server verifies
that the configured Go base directory and its build, module, temporary, and
binary subdirectories are readable and writable. Startup fails with a clear
error if that validation fails.

The server carries the selected base directory through each request context.
Every semedit-launched Go subprocess, including `gopls`, `go vet`, `go get`,
and `go mod tidy`, receives a controlled environment: `GOENV`, `GOCACHE`,
`GOMODCACHE`, `GOTMPDIR`, and `GOBIN` all resolve below that base directory.
CLI calls without an explicit MCP setting retain the workspace-local default.

## Invariants

- MCP configuration names an absolute executable, not a fixture-relative
  binary path.
- Server instructions are a server configuration input, not a task-prompt
  rewrite.
- MCP startup fails before tool use when its Go state directory is inaccessible.
- Go subprocess state is derived from the selected request context, never from
  an inaccessible inherited cache path.
- No project workspace manifest is created or modified to establish Go state.

## Consequences

MCP operation is reproducible in fixture sandboxes and fails early when the
configured state location is invalid. The server owns more process-environment
setup, but callers no longer need bespoke cache exports for semantic tool
calls.
