<!-- Indexes exact benchmark recipes and validation receipts separately from measured model results. -->

# Multi-SWE Go explicit recipes

Each fixture contains one self-contained prompt that asks for exactly one
`semantic_batch` with complete ordered arguments. All five recipes passed fresh
pinned-source replay and their ordinary private acceptance oracles.

| Fixture | Operations | Exact prompt | Validation receipt |
| :--- | ---: | :--- | :--- |
| grpc-go-2735 | 32 | [Prompt](grpc__grpc-go-2735/explicit-single-call.md) | [Receipt](grpc__grpc-go-2735/integrated-validation-receipt.json) |
| grpc-go-2951 | 59 | [Prompt](grpc__grpc-go-2951/explicit-single-call.md) | [Receipt](grpc__grpc-go-2951/integrated-validation-receipt.json) |
| grpc-go-3361 | 7 | [Prompt](grpc__grpc-go-3361/explicit-single-call.md) | [Receipt](grpc__grpc-go-3361/integrated-validation-receipt.json) |
| go-zero-2463 | 29 | [Prompt](zeromicro__go-zero-2463/explicit-single-call.md) | [Receipt](zeromicro__go-zero-2463/integrated-validation-receipt.json) |
| go-zero-964 | 17 | [Prompt](zeromicro__go-zero-964/explicit.md) | [Receipt](zeromicro__go-zero-964/integrated-validation-receipt.json) |

Receipts retain source pins, binary and artifact hashes, batch outcomes, and
oracle evidence. grpc-go-3361 uses a private fixed TLS verification clock;
go-zero-964 corrects a test API call and pins an arm64-compatible test dependency
with inlining disabled. Original acceptance artifacts and failures remain saved.

These files validate recipes. Performance comparisons require measured coding
agent runs with the same fixture versions and declared conditions.
