# ADR-0040: Benchmark Fixture and Oracle Isolation

Status: Accepted
Date: 2026-09-21

## Context

Small single-edit tasks stopped discriminating between direct file editing and
semantic operations as agents became reliable at one-shot changes. A useful
benchmark must expose related, non-uniform code across a compiling project
without revealing the implementation plan or allowing an agent to modify the
acceptance test that proves the result.

## Decision

Benchmark tasks are executable txtar fixtures with explicit visible prompts
and small or large project-size variants. Visible source, documentation prose,
and ordinary unit tests model the working repository, but must not encode the
solution or act as a direct acceptance oracle.

The harness executes an agent inside a fresh extracted fixture. It does not
add task-aware source overlays or runtime restrictions beyond that fixture
boundary. It presents the fixture-declared protected-file list as a static
prompt guardrail, using a generic instruction not to edit tests and including
recovery guidance to restore a protected file if an earlier turn changed it.
This exposes a task constraint, not the oracle's observed result. The oracle
enforces mutation policy, structural postconditions,
clean compilation, and behavioral verification. Hidden acceptance tests are
stored outside the agent-visible fixture. Before provider launch, shared-module
setup may install them in a disposable private copy to download their pinned
dependencies and prove offline access. That copy is removed before launch;
the original fixture manifests remain unchanged. Evaluation installs hidden
tests into its own copy after the agent exits. External Multi-SWE-Bench tasks
use the benchmark's patch protocol in a separate private checkout: start from
the pinned base commit, apply `test.patch`, then apply the candidate's source
patch before running build and test checks. The private test patch never enters
the agent's workspace. These task prompts forbid edits to test files and the
mutation policy enforces that restriction. Ground-truth fix-patch replay
follows the same patch order, while allowing test-file changes that are part
of the benchmark's reference fix.
The copy path is root-scoped so an agent-created symlink cannot redirect a
hidden test outside the benchmark workspace.

Oracle Go test commands accept package paths, `-race`, and the exact
`-gcflags=all=-l` option. The last option is required for acceptance tests that
monkeypatch Go functions: compiler inlining otherwise bypasses the patched
function. It is a declared fixture condition shared by all arms. Other compiler
and linker options remain rejected. Test-only dependencies belong in the private
test patch, independently of a candidate's production edits or human fix patch.

An empty txtar fixture may probe fixed agent overhead. The mutation policy
entry `disallowed_files: ["*"]` rejects changes to regular workspace files,
including newly created files, using the post-extraction snapshot. The snapshot
excludes `.git` and `.scratch` and does not track directory entries. Such a
no-op probe cannot
validate the response text and does not estimate the crossover point for
semantic editing. An edit-count fixture family must use the same task contract
and independently checked structural and behavioral postconditions at each
count.

New semantic-routing scenarios should require several coordinated edits across
related but inconsistent code, while preserving unrelated protocols with
similar names. Task 11 is the reference shape: normalize an audit delivery
boundary, handle control traffic, preserve other sink protocols, and work
across small and large variants without revealing a symbol-by-symbol plan.

## Invariants

The optional storage boundary in
[ADR-0074](0074-ram-backed-benchmark-workspaces-and-persistent-module-cache.md)
places the complete disposable fixture on a validated native tmpfs. Persistent
repository caches remain on SSD and are materialized into each job. The selected
SSD module cache is accessed directly; each job owns independent build,
temporary, gopls, and agent state. Durable transcripts, checkpoints, and reports
remain outside the disposable root. Cleanup removes only owned job directories
after publishing required evidence, never the mountpoint or shared modules.

- Every runnable task has a non-empty txtar prompt variant.
- The model cannot inspect or alter hidden acceptance tests before execution
  completes.
- The prompt may state fixture-declared protected files, but never reveals an
  oracle result, a failed condition, or hidden-test output.
- Visible tests may represent existing behavior but must not be a solution
  checklist.
- The oracle rejects disallowed mutation, validates semantic postconditions,
  and runs build and hidden behavioral checks independently.
- External Multi-SWE-Bench evaluation applies the pinned base, private test
  patch, and candidate patch in that order; test files remain protected in
  model runs.
- Small and large variants preserve the same task contract while changing the
  amount of related code an agent must navigate.
- Acceptance fixture corrections preserve assertions and record original and
  corrected artifacts. Historical results retain their original fixture identity.

## Consequences

Benchmark fixtures better measure semantic routing and multi-step planning
than trivial body replacement. They cost more to author because the visible
repository narrative and hidden oracle must be designed separately.
