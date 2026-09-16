# RQ-0006: Multi-Step Refactorings & Diagnostic Deltas

* **Status**: Resolved
* **Category**: Workflows & Verification
* **Last Updated**: 2026-09-16

---

## 1. Problem Context

Real-world refactoring is rarely a single atomic step. When an agent extracts a method, renames a public interface, or moves a type, intermediate states will temporarily fail compilation (e.g. implementation methods don't match the new signature yet).

If a tool enforces binary pass/fail and automatically rolls back on the first error, legitimate multi-step refactoring workflows are blocked.

---

## 2. Exploration Paths

### A. Diagnostic Delta Tracking ($\Delta \text{errors}$)

Rather than returning a binary failure, `semedit` compares compiler diagnostics before and after the edit:

* **Error Delta ($\Delta$)**: `{"before": 0, "after": 2, "delta": +2, "diagnostics": [...]}`.
* Subsequent steps that fix implementation sites report decreasing deltas: `{"before": 2, "after": 1, "delta": -1}`.
* Once the refactoring sequence is finished, the error count reaches 0.

### B. Composite Refactoring Sessions

Introduce a session abstraction:

```bash
semedit session start --name "rename-interface-method"
semedit rename --symbol "Reader.Read" --to "ReadBytes"
semedit rename --symbol "FileReader.Read" --to "ReadBytes"
semedit session finish
```

* Intermediate steps are recorded.
* The final verification and test pass are evaluated upon `session finish`.

---

## 3. Sources & Prior Art

* **Language Server Protocol Diagnostics**: [LSP textDocument/publishDiagnostics](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_publishDiagnostics).
* **Rust `ra_ap_ide` Diagnostics Engine**: [rust-analyzer internals](https://github.com/rust-lang/rust-analyzer).
* **Grit.io Multi-Step Rewriters**: [GritQL documentation](https://www.grit.io/docs) - chained AST rewriting passes.

---

## 4. Resolution & Findings

We implemented Path A (Diagnostic Delta Tracking) in `internal/pipeline`:

* `ComputeDelta` calculates `net_delta`, `introduced`, and `resolved` diagnostic lists across pre- and post-refactoring states.
* Both CLI `semedit rename` and MCP `semantic_rename` emit structured delta feedback.
* Rather than aborting or unilaterally rolling back intermediate edits, the agent receives clear diagnostic signals showing whether an edit fixed or introduced errors.
* Multi-step acceptance tests (`testdata/scripts/rename_diagnostic_delta.txtar`) and Tier 3 property tests validate that diagnostic deltas correctly guide multi-step workflows to completion.
