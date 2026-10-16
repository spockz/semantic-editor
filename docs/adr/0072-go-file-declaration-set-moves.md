<!-- Records the bounded file-move proof so callers can distinguish closed declaration relocation from general package or resource migration. -->
# ADR-0072: Go File Declaration Set Moves

## Status

Accepted.

## Date

2026-10-16

## Context

Moving declarations individually can reject a valid final package split. A
private helper may need to move with its caller, and a type must retain its
methods. Intermediate revisions can contain unresolved references or import
cycles even when the complete candidate is valid. Filename matching alone does
not establish that tests can change package or working directory safely.

## Decision

Expose the Go-only `semantic_move_file` operation (`semedit move-file`), with
required `file` and `destination_file` and optional `move_matching_tests`
(default true). Both production files must already exist as regular canonical
Go files in different existing packages of one main module. Append the complete
selected file's functions, constants, types, and methods in source order, keeping
comments and declaration groups. The operation leaves a package-only source
shell; it does not delete the source path. A destination package declaration
collision refuses, including collisions in other destination files.

Resolve the declaration set and its module callers together. Moving declarations
may use private declarations within the set. A dependency on a source-package
declaration left behind refuses with a file and symbol location; callers must
explicitly restructure or expand their selection. Do not export private names
or absorb unrelated declarations. Methods must move with their receiver type;
remaining callers of moved private declarations refuse.

Include the exact sibling `foo_test.go` only when its resolved package-local
dependencies fit within the moving production and test sets. Derive its new path
from the destination basename, require that path to be absent, and leave a
package-only test shell at the old path. Same-package tests follow the new package
without changing private visibility. External tests follow the new external test
package and resolved import references. `move_matching_tests: false` leaves the
test in its original package, subject to the same caller and visibility proof.
Shared test helpers, `TestMain`, other-file receiver methods, build variants,
fixtures, and resource dependencies are explicit refusal boundaries.

Package `main`, package variables, `init` functions, and compiler linkage
directives are unsupported. Generated or constrained
selected files, hidden potential references, embeds, relative fixture paths, and
unproven package initialization effects refuse before publication. This contract
does not claim general file, package, resource, or cross-module migration.

For captured affected package and test contexts, compare the ordered sequence
of potentially effectful package initialization before and after the move. Use
the Go specification's [global import-path first-ready order](https://go.dev/ref/spec#Program_initialization),
including transitive dependencies. Treat cgo, initialization functions, and
package variable initializers as potentially effectful. Missing typed inputs,
unpaired relevant contexts, or changed effect sequences refuse. An imported
standard-library package receives no special exemption.

Initialization proof covers captured main-module package and test-program roots,
including combined internal/external test variants and companion/stable-test
import unions. It cannot establish behavior for unknown external or future
consumers. Ambiguous effectful package identities refuse conservatively.

## Invariants

* One source buffer contains the complete plan, including caller and test edits.
* Compiler information identifies declarations and uses; lexical matches alone
  never authorize a reference rewrite.
* Validate the final overlay, source and destination packages, reverse
  dependencies, and relevant active test variants before publication. A passing
  compilation alone is insufficient proof of rewritten reference identity.
* Preparation failure changes no output bytes. Retain exact preimages, absent
  destination checks, and the complete captured read set through publication.
* Publication uses the existing per-file atomic writer and reports zero,
  partial, or complete publication honestly; it does not promise a multi-file
  filesystem transaction or rollback after a published write.
* Ordinary source membership changes synchronize retained gopls documents and
  watcher events inside the Go diagnostic backend. Executable, environment, and
  manifest changes retain their invalidation behavior.
* Registry-derived CLI/MCP descriptions and capability matrices expose Go-only
  support; unsupported backend combinations remain explicit.

## Consequences

Closed files with private helpers and coherent type/method sets can move as one
validated revision. Source shells preserve the original package path and use the
existing checked content publisher. Resource movement, arbitrary initialization
behavior, and declarations spread across multiple selected files require a
separate contract. Refusals are actionable selection evidence, not permission to
move dependencies automatically.
