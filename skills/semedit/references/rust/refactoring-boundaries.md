<!-- WHY: Keep precise Rust refactoring contracts available on demand without making the general Rust guide carry every bounded assist refusal. -->
# Rust Refactoring Boundaries

This guide records the exact source shapes, native edit-capture requirements, and verification limits for Rust refactorings. The primary [tool guide](../../SKILL.md#choose-the-operation-by-intent) lists common operation routing; the [Rust language guide](../rust.md) covers lookup, setup, and language workflows.

## Extract one local constant

Use `refactor_extract_value` with `kind: "constant"`, `symbol`, `in_function`, and the exact initializer `discriminator` for one direct, single-line `let name = expression;` declaration. Rust accepts only a positive decimal `+` chain whose literals all have the same fixed-width integer suffix, and only when the mathematical sum fits that type. For example, `1_i32 + 2_i32` is eligible; `255_u8 + 1_u8`, `isize`/`usize`, calls, variables, and other operators refuse.

The pinned rust-analyzer `extract_constant` assist changes the existing `let` into a typed `const` and inserts a local alias before its first later use. Semedit validates the exact versioned four-edit same-file result, then uses native rename to update only the generated constant and alias reference. It refuses possible type-name shadows from generic parameters, declarations, imports, macros, and unresolved names; requested-name collisions, unexpected action shapes, and unsupported integer ranges also refuse before publication. This operation depends on the captured pinned assist contract; it does not implement `extract_static`. See [ADR-0092](../../../../docs/adr/0092-bounded-rust-extract-constant.md) and [RQ-0074](../../../../docs/research/RQ-0074-rust-refactoring-assist-contract.md).

## Promote one local to a constant

Use the separate `refactor_promote_local_to_const` operation with `file`, `symbol`, and `in_function` for one immutable lowercase snake-case local in a simple zero-argument function. The local must have an explicit fixed-width primitive integer type (`i8` through `i128` or `u8` through `u128`) and a nonnegative decimal literal that fits that type. The function body must contain only that declaration followed immediately by one direct call with the local as its sole argument. For example, `let retry_limit: i32 = 7; consume(retry_limit);` can qualify.

The operation requires exactly two source occurrences of the local, no occurrence of its generated uppercase name anywhere, and no macros, attributes, imports, comments, strings, type shadows, or unresolved syntax that could affect the proof. It refuses references, mutation, extra or non-call uses, aggregates, destructuring, custom and `Drop` types, inferred or pointer-sized types, and any scope beyond the exact function shape. The lazy `promote_local_to_const` action is bound to its original action identity even though the resolve response omits `data`; only the versioned same-file three-edit result matching the independently derived whole-file postimage is accepted. This remains a pinned rust-analyzer contract. It is distinct from constant extraction, which preserves the original local as an alias. See [ADR-0095](../../../../docs/adr/0095-bounded-rust-promote-local-to-const.md) and [RQ-0074](../../../../docs/research/RQ-0074-rust-refactoring-assist-contract.md).

## Extract one local value

Use `refactor_extract_value` only for one exact, fully suffixed integer arithmetic expression that is the sole argument of a simple direct call statement in a trusted Rust function. The expression may use integer literals, parentheses, unary signs, and `+`, `-`, `*`, `/`, or `%`; suffix each literal with the same explicit integer type. Pass the containing function as `in_function`, the exact expression text as `discriminator`, and the requested local name as `symbol`. For example, `1_i32 + 2_i32` is eligible, while `count + tax`, calls, references, and overloaded operators are refused.

The operation accepts only a resolved rust-analyzer `extract_variable` assist with a versioned edit to the selected file; it then renames the introduced binding through rust-analyzer and applies the composed result through the source buffer. The insertion must immediately precede the selected call statement. It refuses command-based actions, stale or unversioned edits, multiple files, snippet edits, wrapped or coalesced selections, repeated textual matches, any requested name already present after Unicode NFC normalization, and files whose whole text contains `!`, `#`, `include`, or both `use` and `*`. These conservative substring checks can refuse harmless comments, strings, attributes such as `#[derive(...)]` and `#[test]`, inequality operators, ordinary imports, or arithmetic. A Rust extraction can share a batch working overlay with a following `semantic_lookup`, which inspects the unpublished source state. These checks avoid unproven shadowing and evaluation/lifetime changes. Rust diagnostics are unavailable. The backend disables build scripts, procedural macros, and check-on-save, while rust-analyzer may run Cargo metadata to load project information. `construct_path` is not supported; if the expression occurs more than once, select a unique source expression or restructure the request. ADR-0078 documents the canonical refactoring name and Rust limits.

## Extract callable statements

Use `refactor_extract_callable` with `kind: "function"`, the containing `in_function`, the new helper `symbol`, and a 1-based contiguous `start_line` and `end_line` pair. Rust accepts one or more contiguous direct calls to the same callee, with no value bindings and only suffixed integer literals or arithmetic in their arguments. The owner and callee must be ordinary top-level functions in the same conventional crate-root file (`lib.rs` or `main.rs` at the crate root or under `src/`) in a single-source Cargo package; the callee must have an implicit unit return. File-level guards reject module declarations and unresolved imports or macros. The selected owner must not use local bindings, local items or imports, control flow, generic or async/unsafe functions, or other argument expressions. Those body restrictions apply to the selected owner; unrelated functions and the callee body are not rejected for containing local bindings or control flow. The backend validates the exact rust-analyzer helper insertion and renames only its declaration and call before staging.

```json
{
  "file": "src/lib.rs",
  "language": "rust",
  "trust_workspace": true,
  "in_function": "run",
  "symbol": "extracted",
  "kind": "function",
  "start_line": 2,
  "end_line": 2
}
```

Rust diagnostics remain unavailable, so inspect the returned diff and run `cargo check` or `cargo test` for project verification.

## Inline one local value

Use `refactor_inline_value` with `symbol` and `in_function` only for one untyped immutable `let` binding initialized by a suffixed decimal integer literal. The binding must be used only as one or more simple identifier arguments on the immediately following ordinary function-call line. The rust-analyzer edit is accepted only when removing the whole declaration and substituting every proven argument yields the exact resulting file. Calls or arithmetic in the initializer, explicit types, `mut`, borrows, macros, shadowing, extra uses, and other argument forms are refused. This narrow rule avoids moving effects, changing inference, or altering borrow and drop behavior.

## Inline one callable

Use `refactor_inline_callable` with a same-file `file` and callee `symbol`, and an exact `call_site` expression of `callee()`. Rust currently accepts only a zero-argument top-level free-function callee whose body is one suffixed decimal integer literal and a top-level caller whose sole tail expression is that call. Both functions must declare the same primitive integer return type, matching the literal suffix. rust-analyzer must return one lazy `refactor.inline` action whose versioned edit replaces only the selected call with that literal; the complete candidate is compared byte-for-byte before staging. Methods, generics, attributes, comments attached to the functions, parameterized or qualified calls, unsuffixed literals, cross-file edits, and inline-into-callers actions are refused. Diagnostics remain unavailable; run `cargo check` or `cargo test` after the edit.
