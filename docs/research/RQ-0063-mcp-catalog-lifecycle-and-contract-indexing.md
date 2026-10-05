<!-- Records registry-contract migration and production catalog-cache measurements separately from test validation. -->

# RQ-0063: MCP Catalog Lifecycle and Contract Indexing

Status: Resolved
Date: 2026-10-04

## Question

Can MCP registry validation avoid schema synthesis, while a server-owned catalog
cache preserves initialization-time language detection and request-root boundaries?

## Accepted constraints

ADR-0041 establishes source-language and verification discovery after initialization
resolves the workspace root and Go state directories. Discovery and dispatch must
use the same language snapshot. Lightweight tests without an initialize handshake
may obtain that snapshot lazily. A successful reinitialization or reload refreshes
session state.

Request-selected roots may refer to another in-tree module or worktree. Cache
language snapshots per canonical effective root within the server's session
generation, and share that state across transient request views. Do not reuse the
configured root's languages for a different selected root. A successful empty
language set is a valid cached result; discovery errors remain explicit and retryable.

Schema maps are immutable once published into the catalog cache. Internal wire
serialization may reuse them; any caller that can modify nested maps receives an
independent copy. Different servers and injected registries never share mutable
catalog data.

Live continuations require role, consumer, leaf-name, and schema compatibility.
Navigation coordinates and historical evidence remain informational fields, not
live target or selector continuations. Nested continuation paths retain their
structure. Source-field validation uses indexes built from these declared
contracts instead of generating a tool's JSON Schema to inspect it.

## Production measurement method

The production probe launches a real MCP binary in a fixed Go workspace, performs
the initialize handshake, and requests the full catalog twelve times per session.
Seven independent processes provide medians for startup through initialization,
first discovery, repeated discovery, and total child CPU time. JSON decoding and
catalog hashing occur in the probe, outside the server; no `mcptest` schema
validator runs in the subprocess or timed request path.
Request round-trip timings include probe-side JSON decoding; hashing follows the
timed interval. Child CPU measurements exclude the probe process.

The protocol helper and full observations are retained locally under
`.scratch/catalog-probe/`. This is a warm-state, small-workspace measurement,
not a claim about cold compilation, LLM throughput, or allocations attributable
to test-runner validation. The pre-change binary SHA-256 is
`5da7b4d341266eaf2ddc8e2248823d50e185441d5e917ab0bac2ffc8ae74ca19`.
The rebuilt binary SHA-256 is
`8119878c0486579e57f89c1d11b8a7f152c22d5eea57959f4417e232d8224434`.
The final comparison reran the preserved baseline immediately before the rebuilt
binary after the checks completed. Both advertised the same 41 tool names,
including the CLI's live-reload tool, with stable catalogs within every session.

| Metric | Before | After |
| :--- | ---: | ---: |
| Startup through initialize, median | 8.124 ms | 5.424 ms |
| First full `tools/list`, median | 5.345 ms | 5.221 ms |
| Repeated full `tools/list`, median | 4.371 ms | 3.167 ms |
| Child CPU per session, median | 73.765 ms | 31.598 ms |

Repeated discovery used 27.5% less round-trip time and the twelve-request session
used 57.2% less server CPU. First discovery was nearly unchanged. Response size
grew from 496,498 to 503,221 bytes because the resolved contracts and descriptions
changed. These combined migration/cache measurements do not isolate garbage
collection costs or prove the earlier profiling percentages. The initial
baseline and final paired samples remain separate artifacts.

## Inventory and verification

The effective fallback inventory contained twelve producers: `semantic_lookup`,
`semantic_replace_body`, `semantic_replace_construct`, `semantic_insert_case`,
`semantic_move_construct`, `semantic_scaffold_file`, `semantic_insert_declaration`,
`semantic_insert_function`, `semantic_insert_type`, `semantic_insert_decl`,
`semantic_replace_decl`, and `semantic_insert_construct`. Each now declares its
live sibling fields in an explicit legacy-envelope contract. Modern typed
outputs retain their projector and result-schema requirements.

Lookup coordinates and kind, insertion coordinates, move origin paths, and
scaffold package names are informational metadata. Every source edge checks
leaf-name equality and schema assignability. Live edges additionally check roles
and declared consumers; only scaffold `package` consumes role-less metadata.

Strict validation removed two renamed source edges: move `file` into
`destination_file`, and lookup `file` into introduce-parameter-object
`destination_file`. Inspection owner selectors use correctly named nested paths.
Target identities, their file companions, and revision tokens retain Target roles;
owner and discriminator fields retain Selector roles. Missing consumer declarations
on existing typed producers were completed rather than bypassed.

The old schema-based fallback helpers have been removed. CLI-only definitions
retain parameter/output validation but do not become MCP providers or consumers.
Declared non-optional sibling fields appear in the native-success schema's
required list; generic and known-error branches retain their own requirements.

Focused tests and the complete `GOFLAGS=-p=2 GOMAXPROCS=2 make check` passed,
including race detection, shuffled ordering, schema conformance, lifecycle,
lint, and documentation checks. The MCP package passed in 182.626 seconds.
An earlier full run's only failure was the previously observed gopls diagnostics
shutdown error in a rename identity test; the unchanged isolated race test and
the bounded full rerun passed. The incident is logged in `SUBOPTIMAL_TOOLS.md`.
The optional `govulncheck` executable was unavailable, so that scan did not run.
`make build-next` rebuilt the measured development binary.

Lifecycle tests prove frozen languages, lazy pre-handshake discovery, refresh
after successful initialization, failed reinitialization preservation, selected
root isolation, retryable discovery failures, defensive schema copies, shared
concurrent fills, and rejection of stale catalog builds. Schema-validating direct
and batch wire tests prove the language guard blocks a newly added language
until refresh, then reaches its handler. No global mutable catalog is shared
between servers.
