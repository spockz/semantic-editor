# Makefile Semantic Refactoring Guide

Failed selected-file lookups and symbol inspections may include native-validated
`suggestions`. Retry the same operation with one candidate's `arguments`. The
normal installed server and workspace trust remain required; suggestions never
edit the Makefile or retry automatically. See [configuration and limits](../../../docs/symbol-suggestions.md).

This reference provides workflows for inspecting and refactoring Makefiles with `semedit` primitives. Read this document only when working with Makefiles.

---

Compact lookup and outline listings use workspace-relative paths and one-based
inclusive declaration lines. Copy native target or variable selectors verbatim.
Read disk-backed source directly, or use supported `semantic_inspect_symbol`
inspection to retrieve source and revision information. Navigation does not grant
mutation capability.

## 1. Macro Refactorings in Makefiles

`semedit` core operations for Makefiles are indexed in the primary `SKILL.md` table. Makefile backend operations require workspace trust and an installed `make-ls`. This reference defines target structuring patterns and verification practices.

---

## 2. Makefile Refactoring Patterns

### Pattern A: Standardize Check and Sub-Targets

When structuring build and verification targets in a project Makefile:

1. **Preflight Inspection**:
   - Use `semantic_outline` to locate declared targets, variables, and include directives.
   - Use `semantic_lookup` to resolve specific target or variable declaration coordinates.
   - Use `semantic_inspect_symbol` to retrieve the rule's exact recipe lines and snapshot revision before modifying recipes.
2. **Target Hierarchy Organization**:
   - Establish a clean `.DEFAULT_GOAL := check`.
   - Structure `check` as an umbrella target depending on modular sub-targets (`check: format lint test`).
   - Declare all action targets in `.PHONY` (`.PHONY: check test lint format`).
3. **Recipe Hardening**:
   - Ensure recipes run with defensive shell options:
     `SHELL := /bin/bash`
     `.SHELLFLAGS := -eu -o pipefail -c`
4. **Verification**:
   - Execute dry-run verification: `make -n check`.

---

### Pattern B: Extract Reusable Flags and Paths

When deduplicating recipe paths and tool flags:

1. **Variable Extraction**:
   - Use `semantic_lookup` to check for colliding variable definitions.
   - Extract repeated commands, tool versions, and output directories into uppercase variables at the top of the file.
   - Use `?=` for overrideable environment parameters and `:=` for immediate expansions.
2. **Verification**:
   - Validate variable resolution with `make -p` or target dry-run.
