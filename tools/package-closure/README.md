# Package Closure Linter

This tool is being developed to find candidate Go package boundaries that could reduce unrelated test and build invalidation. It applies the Common Closure Principle as an investigation aid: code that changes for the same reasons should tend to stay together, while code with weak coupling and independent change histories may belong in separate packages.

## Requirements

- Analyze packages meeting a configurable minimum source-file count. The default is three files.
- Make inclusion of `_test.go` files configurable. The default excludes them from candidate groups, while reporting multi-file test reach separately from production coupling.
- Collect inexpensive compiler-derived package and file data first: direct package imports, per-file import counts, exported declarations, and typed references between files.
- Start from one cluster per source file, pairing conventional sibling `_test.go` files with their source file when tests are included. Repeatedly merge the most coupled clusters and score each resulting boundary.
- Invoke Git only for the highest-ranked candidates whose static cut is promising. Use Git's `--find-renames` behavior as-is; do not implement rename tracking.
- Report candidate groups with their static score, Git co-change rate, and separate count of tests spanning each proposed boundary.
- Integrate the linter into the repository's golangci-lint runs. Build the Go plugin natively alongside the pinned golangci-lint binary before invoking it.
- Evaluate the report against the `internal/mcp` package and record whether it agrees with the earlier assessment that presentation and workspace-path policy were distinct seams.

## Analysis

The golangci plugin API supplies compiler syntax and type information to an analyzer, but does not provide a supported way to consume another linter's diagnostics as shared input. This linter therefore derives equivalent initial signals from the same compiled package: `types.Package.Imports`, exported package declarations, each file's imports, and cross-file object references.

All production file-pair coupling is computed before Git is started. Typed production references weigh more than shared import paths, which contribute a small weight. Tests that reference multiple source files do not add production coupling edges. Their file sets are tracked separately, and each candidate reports how many test files would span its boundary. A sibling pair such as `presentation.go` and `presentation_test.go` also counts as dedicated test coverage when test syntax is excluded. When tests are included, conventional sibling source/test files begin as one indivisible cluster. The analyzer then repeatedly merges the pair of clusters with the greatest weighted Jaccard coupling: their shared production edge weight divided by the total production edge weight touching either cluster. Every intermediate cluster is scored against its complement, including one-file package candidates. This agglomerative tree limits candidate generation to boundaries created by production coupling instead of enumerating arbitrary bipartitions. A split's static cut ranks candidates; it is not a hard rejection threshold because a heavily referenced API file may still change independently. The highest-ranked candidates are compared with one package-level Git history scan. Reports rank splits with dedicated tests on both sides first, then low co-change and static cut; cross-boundary test count remains a separate review signal.

Git history uses `git log --find-renames` over the package directory and the current file paths. The analyzer does not synthesize rename chains. For each candidate, its co-change rate is commits touching both sides divided by commits touching either side. Candidates need enough history before a report is emitted.

These scores are evidence for review, not a proof of cohesion. Compiler references cannot fully model behavioral coupling, reflection, interface dispatch, or reasons for future changes. The linter does not move files automatically.

## Configuration

Settings live under `linters.settings.custom.packageclosure` in the root `.golangci.yml`:

| Setting | Default | Meaning |
| --- | ---: | --- |
| `min-files` | 3 | Minimum non-filtered files in a package before analysis. |
| `include-tests` | false | Include test files as candidate-group nodes. |
| `min-group-files` | 1 | Minimum source files on each side of a candidate split. One-file packages are valid candidates; Git co-change supplies their cohesion evidence. |
| `history-commits` | 300 | Git commits inspected for a promising package candidate. |
| `max-history-candidates` | 512 | Highest-ranked static candidates to score against the package's one Git history scan. |
| `max-reports` | 8 | Highest-ranked low-co-change seams reported per package. |
| `min-history-commits` | 4 | Minimum commits touching either side before reporting. |

## Build and Run

The repository Makefile installs the pinned golangci-lint binary and builds `packageclosure.so` for the current host before each lint run:

```sh
make build-golangci-lint
make lint-go
```

The plugin and golangci-lint binary are built with the same Go toolchain and dependency versions. Go's plugin loader requires matching toolchains and platforms, so this target builds natively for the current machine rather than cross-compiling. Make runs `packageclosure` as a separate advisory pass with exit code zero, then runs the regular blocking linters with `packageclosure` disabled. Candidate reports therefore remain visible without failing `make check`.

## MCP Assessment

The earlier MCP assessment found that result presentation and workspace-root/path policy had separate responsibilities and low dependency surfaces. The extracted packages are now `internal/mcp/presentation` and `internal/mcp/workspace`. On the committed pre-split tree, the analyzer ranks `workspace_root.go` as a one-file candidate (static cut 0.06; co-change 0.01 over 80 commits; one dedicated test file; one test file spans the boundary) and `presentation.go` as another (static cut 0.12; co-change 0.04 over 80 commits; one dedicated test file; two test files span the boundary). These match the two files previously assessed as separate seams. The spanning tests are reported separately and do not increase production coupling. On the current tree, it reports compatibility wrappers because the implementations have already moved. File history and sibling test files recover the original boundary evidence; the current package no longer contains those implementations.
