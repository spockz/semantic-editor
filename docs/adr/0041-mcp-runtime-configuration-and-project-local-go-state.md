# ADR-0041: MCP Runtime Configuration and Project-Local Go State

Status: Accepted
Date: 2026-09-21

## Context

MCP clients can launch `semedit` from a benchmark fixture while configuration
often lives outside that fixture. Relative executable paths therefore resolve
unreliably, and inherited global Go caches can be unreadable or unwritable in
sandboxed sessions. A Makefile environment fixes only its own shell recipes;
it does not configure Go subprocesses launched later by an MCP server.
The same issue affects Codex agent shells: a fixture sandbox may read a host
Go cache but cannot safely update it, while inherited `GOWORK` or `GOFLAGS`
can redirect work outside the fixture.

## Decision

MCP registration uses an absolute path to the semedit binary and starts the
server with the client project directory as its working directory. The server
offers optional initialization instructions and a `--go-base-dir` setting.
When no base directory is specified, it uses `<server-cwd>/.scratch/go`.

The benchmark harness passes the complete semantic MCP registration through
per-launch Codex configuration overrides: `enabled=true`, the absolute
`bin/semedit-next` command, and `mcp --profile full` arguments. Fixture
directories do not contain the repository's `.codex/config.toml`, so overriding
only the command would launch the executable without its MCP subcommand and
silently leave semantic tools unavailable. Server instructions add their
`--instructions` argument to this explicit base command.

Before reading JSON-RPC input or advertising tools, the MCP server verifies
that the configured Go base directory and its build, module, temporary, and
binary subdirectories are readable and writable. Startup fails with a clear
error if that validation fails.

The server carries the selected base directory through each request context.
Every semedit-launched Go subprocess, including `gopls`, `go vet`, `go get`,
and `go mod tidy`, receives a controlled environment: `GOENV`, `GOCACHE`,
`GOMODCACHE`, `GOTMPDIR`, `GOBIN`, and `GOPATH` all resolve below that base
directory. `GOWORK=off` and an empty `GOFLAGS` prevent inherited workspace or
overlay configuration from changing the fixture's build. Codex benchmark
processes receive the same environment before they start, and the semedit MCP
registration explicitly forwards those variables to its stdio child.

## Invariants

- MCP configuration names an absolute executable, not a fixture-relative
  binary path.
- A benchmark semantic arm supplies complete MCP command registration rather
  than relying on configuration inherited from the fixture directory.
- Server instructions are a server configuration input, not a task-prompt
  rewrite.
- MCP startup fails before tool use when its Go state directory is inaccessible.
- Go subprocess state is derived from the selected request context, never from
  an inaccessible inherited cache path.
- Codex benchmark shells and semedit MCP children use the same fixture-local
  Go state; inherited `GOWORK` and `GOFLAGS` never select files outside it.
- No project workspace manifest is created or modified to establish Go state.

## Consequences

MCP operation is reproducible in fixture sandboxes and fails early when the
configured state location is invalid. The server owns more process-environment
setup, but callers no longer need bespoke cache exports for semantic tool
calls. Fixtures begin with isolated Go state. A future speed optimization may
explicitly preseed that state through the harness, but must not fall back to a
mutable host cache; ADR-0038 governs the seed and its provenance. The inherited
`GOPROXY` and `GOTOOLCHAIN` policy remains benchmark provenance.
