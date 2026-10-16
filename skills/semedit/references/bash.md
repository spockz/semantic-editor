# Bash Semantic Refactoring Guide

Failed selected-file lookups may include native-validated `suggestions`. Retry
the same lookup with one candidate's `arguments`. The normal installed server
and workspace trust remain required; suggestions never edit shell source or
execute a mutation retry. See [configuration and limits](../../../docs/symbol-suggestions.md).

This reference provides workflows for inspecting and refactoring shell scripts with `semedit` primitives. Read this document only when working with Bash scripts.

---

Compact lookup and outline listings use workspace-relative paths and one-based
inclusive declaration lines. Signatures describe declarations; explicit native
selectors remain separate and must be copied verbatim. Use ordinary file reading
for disk-backed results. A navigation result does not imply that this backend
supports symbol inspection or mutation.

## 1. Macro Refactorings in Bash

`semedit` core operations for Bash (`semantic_outline`, `semantic_lookup`, `semantic_verify`) are indexed in the primary `SKILL.md` table. Bash backend operations require workspace trust and an installed `bash-language-server`. This reference defines script modularization patterns and ShellCheck verification practices.

---

## 2. Bash Refactoring Patterns

### Pattern A: Modularize Script into Functions

When decomposing a monolithic procedural shell script into testable, scoped functions:

1. **Preflight Inspection**:
   - Use `semantic_outline` to inspect existing global commands and defined functions.
2. **Function Extraction**:
   - Wrap procedural sequences into discrete functions with explicit `local` variable declarations.
   - Guard main entrypoint behind a standard `main "$@"` invocation.
3. **Defensive Hardening**:
   - Ensure `set -euo pipefail` is present at the script top.
   - Quote all variable expansions (`"$var"`, `"$@"`).
4. **Verification**:
   - Run `semantic_verify` with `language: "bash"` and `trust_workspace: true` to trigger ShellCheck diagnostics.

---

### Pattern B: Extract Shared Utility Script

When sharing functions across multiple shell scripts:

1. **Isolate Common Utilities**:
   - Move shared functions into a library script (e.g., `lib/common.sh`).
2. **Source Library**:
   - Source the library defensively using directory-relative anchors:
     `source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/common.sh"`.
3. **Verification**:
   - Verify script execution and ShellCheck clean state.
