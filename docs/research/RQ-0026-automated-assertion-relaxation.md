# RQ-0026: Automated Assertion Relaxation (Fail-Fast to Continue-on-Failure)

* **Status**: Open
* **Category**: Semantics
* **Date**: 2026-09-18

---

## 1. Context & Motivation

ADR-0033 (exhaustive assertion reporting) required remediating 444 fail-fast
call sites across 30 test files: 216 mechanical swaps (`t.Fatalf` to
`t.Errorf`, message and args untouched), 49 restructures (value check guarding
a later dereference or index, converted to else-wrap, `Errorf` + `return`, or
`Errorf` + `continue`), and 179 legit fatal preconditions left alone. The work
was done by audit-driven manual editing. The question is which parts admit a
deterministic engine operation (`semantic_relax_assertions`) and what machinery
each part needs. Three solution directions were proposed and probed
empirically with ast-grep 0.45.3 (preinstalled; Go grammar builtin).

---

## 2. Direction A: Syntactic Find + Mechanical Replace

The common case (216 of 265 changed sites) is one token: receiver and
argument list are byte-identical, only the method name changes. This is the
OpenRewrite shape: a declarative find-by-method-pattern step plus a rename
step. Prior art is `org.openrewrite.java.ChangeMethodName` (`methodPattern`
plus `newMethodName`, composable in YAML recipe lists, with `rewriteDryRun`
preview versus `rewriteRun` apply).

### Probe outcomes (verified 2026-09-18, ast-grep 0.45.3)

* Literal call patterns match: `t.Foo(1, 2)` finds the call site.
* Metavariables in selector position **misparse**: `$R.$M($$$)` and
  `$T.Fatalf($$$ARGS)` parse as `type_conversion_expression`
  (`qualified_type`), confirmed via `--debug-query`, and match nothing. This
  is a Go-grammar sharp edge, not a fundamental limit.
* Workaround succeeds: `kind: call_expression` with
  `has: {field: function, regex: ^t\.Fatalf?$}` finds **25 of 25** remaining
  `t.Fatal*` sites in `internal/snapshot/snapshot_test.go` (100% recall
  against the audit baseline).
* `--rewrite` renders the rewritten call to stdout **without touching disk**
  (worktree verified clean), matching the staged preview-then-apply ethos of
  ADR-0004. Post-rewrite `gofmt` plus parse validation (ADR-0011 precedent)
  and compiler diagnostics close the loop.

### Known gap: receiver type knowledge

ast-grep is syntactic: it cannot verify the receiver is `*testing.T` (or
`*testing.B`, rapid `*T`). Practical mitigations, in ascending strength:

1. Scope to `*_test.go` with receiver-name convention (`t`, `rt`).
2. Resolve the receiver type via `go/types` or the existing gopls pipeline
   before rewriting (the engine already owns both).
3. OpenRewrite contrast: its `MethodMatcher` is type-attributed, matching on
   resolved method type rather than syntax. A semedit equivalent gets this
   property from option 2, not from the pattern language.

**Verdict: feasible now.** Direction A is a solved shape: constrained
structural find, stdout preview, scored or verified apply.

---

## 3. Direction B: Assertion Pair-Map as Engine Knowledge

The row-loop variant is the same lookup as Direction A, plus the knowledge
that the replacement has an identical signature so argument names and order
transfer untouched. That knowledge belongs in the engine, not in each call
site. Precedent: ADR-0012 stores per-backend access-modifier support as
capability declarations; assertion pairs follow the same shape.

### Initial taxonomy (Go backend first)

| Abort (fail-fast) | Continue | Signature relation |
| :--- | :--- | :--- |
| `t.Fatalf` / `t.Fatal` | `t.Errorf` / `t.Error` | Identical (stdlib fact) |
| `require.*` (testify) | `assert.*` (testify) | Identical by construction |
| `t.FailNow` | structured return / `t.Errorf` + `return` | Requires control-flow edit |

Cross-language pairs exist but differ in depth (JUnit `Assert` to AssertJ
`assertSoftly` needs a rule object; Rust `assert!` has no direct soft form;
Jest `expect` already continues). Per ADR-0021 and ADR-0028, the operation is
specified language-neutrally in the capability matrix while only the Go
backend implements it; lookup-only backends advertise it as unsupported.

**Verdict: feasible now.** The pair-map is a static table plus the Direction A
find/replace machinery. No new analysis required.

---

## 4. Direction C: Restructure Search (Guard Clauses)

The 49 restructures share three fixed shapes, which together covered every
observed case in the remediation:

1. **Count-guard plus index**: `if len(x) != N { Fatalf }` with `x[i]` below
   becomes `Errorf` + `return` (or `continue` in loops).
2. **Error-guard plus dereference**: `if err != nil { Fatalf }` with the
   guarded value used below becomes an else-wrapped (inverted) check.
3. **Loop-row failure**: any `Fatalf` inside a row/case loop becomes `Errorf`
   * `continue` so sibling cases survive.

### Probe outcomes

* Candidate discovery works relationally: `kind: if_statement` with
  `has: {regex: t\.Fatalf?\(}` finds **25 of 25** guard sites in
  `internal/snapshot/snapshot_test.go`. Nested `has`-with-`field` composition
  returned nothing and needed the regex formulation; rule authoring needs care
  but no new engine primitive.
* Rewriting is three fixed templates plus `gofmt`, selected by which shape
  the use-below takes (index, dereference, loop row). Sites matching none of
  the three shapes are reported and skipped, never bare-swapped.
* `semantic_replace_body` fits only whole small functions; applying it per
  call site is disproportionate (full-body rewrite for a guard inversion) and
  risks unrelated regressions. A targeted guard-inversion primitive (or the
  ast-grep template path above) is the proportionate vehicle.

**Verdict: feasible with bounded scope.** Discovery plus three templates
covers the observed space; the escape hatch (report-and-skip) keeps unknown
shapes safe.

---

## 5. Next Steps

1. Propose `semantic_relax_assertions` (intent name per ADR-0007): scope
   (file or symbol), pair-map from the Go backend capability declaration,
   policy for unsafe sites (`else-wrap` / `continue` / `return` /
   `report-and-skip`), dry-run plan output before apply.
2. Record the decision in an ADR; add the capability-registry entry and CLI
   txtar cases per ADR-0028 in the same change.
3. Keep the ast-grep grammar quirk (selector-pattern misparse) documented for
   rule authors until rules move behind the engine operation.
