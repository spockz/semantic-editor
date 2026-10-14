# ADR-0041: MCP Runtime Configuration and Project-Local Go State

Status: Accepted
Date: 2026-09-21

The default Go isolation policy below is superseded by [ADR-0070](0070-opt-in-go-runtime-isolation.md). Normal CLI and MCP use inherit the user's Go state; benchmark fixtures and explicit `--go-base-dir` configurations retain isolation. Workspace-root selection and the other runtime invariants remain in force.

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
defaults to prescriptive initialization instructions and offers a `--go-base-dir`
setting. `WithInstructions` and `--instructions` override that guidance;
explicit empty instructions omit the initialization field. Omission retains
the prescriptive default, including direct `NewServer` callers.
The launch CWD is only an initial fallback. During MCP `initialize`, resolve the
workspace root supplied by the client before deriving the default Go state
directory as `<workspace-root>/.scratch/go`; retain an explicitly configured
`--go-base-dir`. The workspace root and Go cache base directory are distinct.

Each MCP operation may also select an optional `workspace_root` beneath the
configured server root. Resolve relative roots against that root, canonicalize
the result, and require an existing directory contained by the configured tree.
This permits an in-repository worktree such as `.scratch/worktrees/task` while
keeping paths contained. Resolve relative file selectors beneath the selected
root, normalize in-root absolute file selectors to relative paths, and reject
parent traversal, symlinks, or escape across recursively checked file fields.
Pass the canonical effective root through project and call contexts, trust
scopes, continuations, and the default Go cache path so later operations keep
using the same workspace boundary. A session manager reuses Go state for the
configured root and same-module subdirectories; a different worktree module
gets its own `<workspace-root>/.scratch/go` state.

After those directories are set, detect source languages and verification
configuration in the resolved workspace, as specified by ADR-0054. Do not retain
an activation plan derived from a different launch CWD. Detection establishes
which checks apply; filesystem checks execute only after edit publication.
ADR-0054 implements this verification-signal integration at the resolved
initialization boundary.

Retain the detected active source languages as a session snapshot. Both tool
discovery and language-policy validation use that snapshot rather than rescanning
the filesystem on each request. Tests that omit the initialize handshake obtain
a snapshot lazily against the configured root. A request-selected module or
worktree obtains its own snapshot under the same server session; transient
request views share those records instead of repeating detection. A successful
reinitialization refreshes all root-specific snapshots, while a failed attempt
retains the previous usable configuration. Reload refreshes session state.

The benchmark harness enables `--semedit-prewarm-verify` by default for Codex
semedit jobs with a Go oracle target. An explicit false value disables it, and
fixtures without a Go target skip it. The treatment runs the promoted semedit
`verify --check-only` CLI on the extracted Go fixture before starting the
semedit agent. It uses the same fixture-local Go environment as the Codex
process and refuses a non-empty initial build or module cache. The preflight
runs in a separate process; it may populate disk caches but does not retain a
`gopls` process. Fixture setup stays outside agent wall time; preflight duration is recorded
separately. Agent wall time begins after successful Codex process start and
continues through task turns and oracle checks. The baseline arm does not receive this treatment.

The benchmark harness passes the complete semantic MCP registration through
per-launch Codex configuration overrides: `enabled=true`, the absolute
`bin/semedit-next` command, and `mcp --profile full` arguments. Fixture
directories do not contain the repository's `.codex/config.toml`, so overriding
only the command would launch the executable without its MCP subcommand and
silently leave semantic tools unavailable. Server instructions add their
`--instructions` argument to this explicit base command.

During initialization, after resolving the workspace and base directories and
before advertising tools or executing operations, the MCP server verifies that
the selected Go base directory and its build, module, temporary, and binary
subdirectories are readable and writable. Initialization fails with a clear
error if that validation fails.

The server carries the selected base directory through each request context.
Every semedit-launched Go subprocess, including `gopls`, `go vet`, `go get`,
and `go mod tidy`, receives a controlled environment: `GOENV`, `GOCACHE`,
`GOMODCACHE`, `GOTMPDIR`, `GOBIN`, and `GOPATH` all resolve below that base
directory. `GOWORK=off` and an empty `GOFLAGS` prevent inherited workspace or
overlay configuration from changing the fixture's build. Codex benchmark
processes receive the same environment before they start, and the semedit MCP
registration explicitly forwards those variables to its stdio child.

Go child subprocesses also receive `TMPDIR`, `TMP`, and `TEMP` pointing to
the managed temporary directory. gopls creates temporary module copies through
the operating system temporary-file API rather than `GOTMPDIR`; retaining its
process must not expose those transient files in workspace snapshots.

## Invariants

- MCP configuration names an absolute executable, not a fixture-relative
  binary path.
- Detect languages and verification signals during initialization after resolving
  the workspace root and base directories, even when launch CWD differs.
- Discovery and dispatch share an immutable active-language snapshot for each
  canonical effective root. Source-file changes do not implicitly change the
  session's language policy; reinitialization or reload refreshes it.
- A benchmark semantic arm supplies complete MCP command registration rather
  than relying on configuration inherited from the fixture directory.
- Server instructions are a server configuration input, not a task-prompt
  rewrite.
- Default initialization guidance is prescriptive; explicit empty guidance
  remains distinct from an omitted override.
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
calls. Fixtures begin with isolated Go state. The default verify preflight is an explicit per-fixture cache seed; it does
not fall back to a mutable host cache. ADR-0038 governs its provenance. The inherited
`GOPROXY` and `GOTOOLCHAIN` policy remains benchmark provenance.

## Restored module safeguards: 2026-10-12

Ordinary Go subprocesses receive `GOFLAGS=-mod=readonly`; the explicit dependency
operation enables writable module changes for its own request. Benchmark builds
also retain `-buildvcs=false`. Dynamic workspace routing remains supported; the
archived restriction to one startup module is not restored.

The optional MCP `--go-module-cache-seed` source seeds a private managed cache
through bounded, symlink-safe, atomic copies. It requires explicit offline
`GOPROXY=off`, reports copy/conflict failures, and never makes the source cache
writable. An unset seed preserves current cache selection.
