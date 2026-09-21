# RQ-0030: Registry-Backed CLI Batches

* **Status**: Open
* **Date**: 2026-09-21
* **Category**: Operations & CLI

## 1. Question

Should direct semantic batch execution become a first-class operation in
`internal/operation`, exposed consistently through both the CLI and MCP, with
the CLI accepting a structured batch plan from a file or standard input?

## 2. Context

`semantic_batch` currently exists only as an MCP transport feature. It
executes an ordered sequence of registered batchable operations, stops at the
first failure, and performs formatting or import organization once for each
written file at the end of a successful batch. The behavior is semantically
independent of MCP, but its request, response, and execution types live in
`internal/mcp`.

This leaves CLI users and scripts to launch separate commands, even when they
need the same semantics: one process, ordered edits, deferred normalization,
and a structured record of partial success. A CLI batch is useful for
reproducible scripted refactors and CI, not only for interactive terminal use.

The current registry is the source of truth for normal operations across CLI,
MCP, batch dispatch, and generated documentation (ADR-0034). Batch itself is
the remaining composition path outside that registration model. ADR-0032 also
records a missing direct-batch conformance property: every outcome must report
the final workspace diff, including formatting and import changes, on success,
diagnostics, and partial semantic-operation failure.

## 3. Candidate Interface

The CLI should favor a structured plan over a heavily escaped command-line
argument:

```text
semedit batch --plan edits.json
semedit batch --plan - < edits.json
```

The candidate plan contains ordered operation keys and parameter objects:

```json
{
  "edits": [
    {
      "operation": "replace_body",
      "params": {
        "file": "calc.go",
        "symbol": "Compute",
        "body": "return x * 10"
      }
    },
    {
      "operation": "organize_imports",
      "params": {
        "file": "calc.go"
      }
    }
  ],
  "auto_organize_imports": false
}
```

`operation` is intentionally the registry key, not an MCP tool name. The same
canonical plan can be represented directly in an MCP argument object, while
the CLI only owns plan-file and standard-input decoding. The operation registry
continues to validate whether every referenced entry is batchable.

## 4. Invariants to Evaluate

1. Batch is a composite operation registered once, with CLI command, MCP tool
   schema, dispatch, documentation, and capability visibility derived from the
   same definition.
2. The plan format is versioned and validated before the first mutation. Bad
   JSON, unknown fields, unknown operations, and non-batchable operations fail
   before any edit is applied.
3. Execution remains direct and fail-fast under ADR-0032: an earlier successful
   edit remains applied when a later operation fails.
4. Every response includes per-entry status and a final workspace diff that
   includes semantic edits plus deferred formatting/import changes, whether the
   batch succeeds or fails partway through.
5. A batch never calls itself recursively.
6. CLI plan-file handling is an ingress concern only; operation behavior,
   authorization, and output are independent of CLI versus MCP.
7. Batch parameter parsing preserves the distinction between JSON arrays and
   strings, avoiding shell-quoting heuristics or compatibility aliases.

## 5. Design Alternatives

### A. Keep MCP-Only Batch

Maintain the existing MCP implementation and let CLI callers chain commands.
This avoids a new CLI plan syntax but keeps composition outside the registry
and loses deferred normalization and one-receipt behavior for scripts.

### B. Registry Batch with JSON Argument Only

Add a generic object-array parameter to the registry and require CLI callers
to pass a JSON value directly through a flag. This is mechanically compact but
awkward and error-prone for humans and shell scripts.

### C. Registry Batch with Plan File or Standard Input

Register a canonical batch request and let the CLI decode a plan file or
standard input into it. This is the leading hypothesis: the CLI owns an
ergonomic transport encoding while MCP uses native structured arguments.

### D. General Workflow Language

Introduce YAML steps, variables, conditionals, or arbitrary commands. This
would exceed semantic operation composition, create a second build system, and
weaken the registry's validation boundary. It is out of scope.

## 6. Evaluation Plan

1. Extract the current batch request, result, and execution logic from
   `internal/mcp` into an ingress-neutral package or registry definition.
2. Define a typed parameter contract for an ordered object array and compare
   CLI plan-file decoding with direct JSON flag input.
3. Add CLI txtar contracts for successful multi-file batches, malformed plans,
   unknown/non-batchable operations, deferred import normalization, and partial
   failures.
4. Add matching MCP contracts that use the same canonical operation keys and
   result schema.
5. Implement final workspace-diff capture and test that it includes
   post-processing changes on success and partial failure.
6. Measure process startup, total execution time, and output size against
   equivalent chained CLI commands and equivalent MCP calls.

## 7. Open Questions

1. Should plan files be JSON only, or should YAML be supported after the
   canonical JSON schema is established?
2. Should `--plan -` be the only standard-input form, or should a missing
   `--plan` default to standard input?
3. How should the final workspace diff represent files created, deleted, or
   changed only by formatting/import organization?
4. Does the batch response need both per-operation diffs and the final
   workspace diff, or is the latter sufficient when paired with statuses?
5. Should an explicit per-batch normalization setting supersede project-level
   defaults proposed in RQ-0029, and how is that precedence recorded?
6. Is a batch itself batchable? The current leading answer is no, to preserve
   an acyclic execution model and one normalization point.

## 8. Next Steps

1. Resolve the final workspace-diff contract from ADR-0032 before changing
   ingress adapters.
2. Specify the canonical batch-plan schema and parameter type.
3. Prototype plan parsing without registering or exposing a new command.
4. Create an ADR only after the operation boundary, validation timing, and
   partial-failure receipt have executable evidence.
