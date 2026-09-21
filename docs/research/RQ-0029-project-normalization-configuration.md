# RQ-0029: Project Normalization Configuration

* **Status**: Open
* **Date**: 2026-09-21
* **Category**: Configuration & Quality

## 1. Question

Should `semedit` support a checked-in declarative project configuration, with
`.semantic-editing.yaml` as a candidate name, that defines default behavior
for automatic formatting, import optimization, and lint-supported fixes after
semantic edits?

## 2. Context

RQ-0028 considers registry-backed `check` and `fix` operations that connect
semantic editing with a project's existing quality policy. A project needs a
way to decide whether a mutation should normally leave its affected files
formatted, its imports organized, and safe lint fixes applied. These decisions
are policy, not language-engine facts: teams vary in their appetite for
automatic rewrites and in the tools and versions configured through files such
as `.golangci.yml` and `.markdownlint-cli2.yaml`.

The configuration must not replace those tool-specific files or reproduce
their rule sets. It controls only whether `semedit` invokes an available
normalizer by default. The normalizer itself continues to honor the project's
existing configuration and only performs repairs it explicitly supports.

## 3. Candidate Shape

The following is an exploratory shape, not a committed schema or filename:

```yaml
version: 1
normalization:
  format: true
  organize_imports: true
  lint_fix: false
```

The three defaults intentionally have independent values. Formatting and
import organization commonly maintain validity and local consistency after a
structural edit. Lint fixes can make a wider set of changes, even when tools
label them automatic, so projects may reasonably require explicit `fix`
invocation or set `lint_fix: true` only after review. The research must test
whether this distinction holds across languages and repositories rather than
hard-code it as universal policy.

Future extensions might allow language-scoped settings while preserving a
small, auditable core:

```yaml
normalization:
  format: true
languages:
  go:
    organize_imports: true
    lint_fix: false
```

## 4. Configuration Contract to Investigate

1. The file is declarative data only. It cannot name arbitrary commands,
   download tools, or relax workspace trust requirements.
2. Absence of the file produces documented built-in defaults, preserving
   current behavior for existing projects.
3. Explicit operation arguments override the project default for one
   invocation; the result receipt records both the effective value and its
   source.
4. Language capability and tool availability bound configuration. Setting
   `lint_fix: true` cannot make an unsupported backend or unavailable linter
   appear available.
5. The resolver finds one project configuration from the selected workspace
   root and does not silently inherit a parent repository's policy across a
   workspace boundary.
6. Invalid, unknown, or future-version configuration fails with an actionable
   diagnostic rather than falling back to a surprising permissive policy.
7. Configuration is read before a mutation plan is committed, so every
   automatic normalization step is predictable and reported.

## 5. Alternatives

### A. No semedit Configuration

Require flags for every call and leave project defaults solely in build tools.
This maximizes per-call explicitness but forces agents and users to repeat
policy decisions and makes consistent defaults difficult to discover.

### B. Declarative Project Configuration

Use a repository file such as `.semantic-editing.yaml` for a narrow set of
defaults, with CLI and MCP overrides. This makes policy reviewable in version
control and preserves normal project ownership of linter configuration.

### C. Reuse a Build System Target

Treat a target such as `make fix` as the sole policy source. This avoids a new
file but couples semantic defaults to arbitrary build-script behavior and
cannot reliably separate safe local normalization from broad dependency,
generation, or test side effects.

## 6. Evaluation Plan

1. Survey representative Go projects for formatting, import, and lint-fix
   expectations, including repositories that intentionally defer lint fixes.
2. Prototype parsing and validation for the minimal three-setting schema
   without enabling mutation from configuration.
3. Add txtar coverage for no config, each default, explicit CLI/MCP override,
   invalid version, unknown key, unsupported language, and nested workspace
   roots.
4. Verify that repeated edits with the same configuration are idempotent and
   that receipts explain which defaults took effect.
5. Evaluate a second language backend before fixing cross-language names or
   default values in an ADR.

## 7. Open Questions

1. Is `.semantic-editing.yaml` discoverable and distinctive enough, or should
   the project use a different name or a section in an existing configuration
   convention?
2. Should default settings apply after every mutation, only batch completion,
   or only explicit workspace `fix` operations?
3. What precedence should apply among built-in defaults, project config,
   language-specific config, and per-call flags?
4. Are booleans sufficient, or should settings express modes such as `never`,
   `changed_files`, and `workspace`?
5. How are multiple module roots and nested repositories resolved without
   surprising configuration inheritance?
6. Should configuration select tool versions, or must that remain exclusively
   the responsibility of the project's package and build tooling?

## 8. Next Steps

1. Decide the exact default behavior to preserve when no configuration exists.
2. Compare filename, discovery, and precedence choices with the project's
   workspace-root safety rules.
3. Define a strict versioned schema and diagnostic format.
4. Defer implementation and an ADR until RQ-0028 resolves the scope and
   authority of semantic `check` and `fix` operations.
