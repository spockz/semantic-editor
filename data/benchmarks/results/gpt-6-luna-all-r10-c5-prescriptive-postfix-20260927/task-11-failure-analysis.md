# Task 11 failure analysis

Runs compared:

- `gpt-6-luna-all-r10-c5-prescriptive-20260927-retry` (before second friction-fix tranche)
- `gpt-6-luna-all-r10-c5-prescriptive-postfix-20260927` (after second tranche)

All are prescriptive-only, 10 repeats, paired baseline and semedit arms, 40 jobs per arm for task 11.

## Oracle outcomes

| Run | Baseline | Semedit |
| --- | ---: | ---: |
| Before | 18/40 | 15/40 |
| After | 23/40 | 18/40 |

After-fix paired outcomes: both passed 10, baseline only passed 13, semedit only passed 8, both failed 9. Semedit therefore still trails baseline by five cases. The earlier run had 9 baseline-only passes and 6 semedit-only passes (both failed 16, both passed 9).

## Combined semedit failure groups

| Oracle result | Before | After | Assessment |
| --- | ---: | ---: | --- |
| `TestHiddenRecordNormalizesBeforeDelivery` fails | 21 | 17 | Material behavior: the sink received no normalized probe event. |
| Required `NormalizeKind` symbol missing from `audit/normalize.go` | 4 | 4 | Material API/AST contract failure. |
| No oracle result after execution ended | 0 | 1 | Harness timeout/process completion issue; do not treat as a code-level oracle result. |

The hidden test checks three behaviors: `NormalizeKind("  PrObE  ") == "probe"`; `Classify("probe") == "control"`; and `Dispatcher.Record` writes exactly one event to the sink with kind `"probe"`. Whitespace and casing are test input; the failures are about missing delivery or an absent required declaration, not formatting.

A representative recorded `semantic_replace_body` body normalizes the event and then returns early when its kind is `probe`. That implements suppression, while the oracle requires control classification and continued delivery. The prompt phrase “treat probe events as control traffic” can invite that interpretation. Suggested wording: explicitly say to retain and deliver probes once after normalization, while classifying them as control traffic.

## Semantic tool use

The fixture’s `diagnostic_expected_tools` are `semantic_lookup`, `semantic_inspect_symbol`, and `semantic_replace_body`. Among the 22 after-fix semedit failures, complete telemetry showed all three in 6 cases; 16 missed at least one. Missing-tool counts overlap: lookup was absent in 11, inspect-symbol in 5, and replace-body in 2. Five of the 17 hidden-test failures nevertheless used the expected set, so tool coverage alone does not explain the behavioral defect.

Total semantic calls observed across the 22 failing jobs (including retries):

| Tool | Calls |
| --- | ---: |
| `semantic_inspect_symbol` | 145 |
| `semantic_replace_body` | 108 |
| `semantic_verify` | 79 |
| `semantic_insert_function` | 45 |
| `semantic_outline` | 31 |
| `semantic_lookup` | 14 |
| `semantic_batch` | 12 |
| `report_feedback` | 5 |
| `semantic_organize_imports` | 4 |
| `semantic_find_references` | 4 |
| `semantic_insert_type` | 1 |

`semantic_insert_function` was needed to add the required `audit.NormalizeKind` helper, but is absent from the fixture’s expected-tool list. It is a good candidate to add to that list. In the four missing-symbol failures, agents attempted different APIs (`Normalize`, `Deliver`, `NormalizeEvent`) instead of the required `NormalizeKind`; one call first failed because it used unsupported parameter `name`. One run did submit a `NormalizeKind` insertion, but the oracle still reported the symbol absent, so inspect that run’s timeout/partial execution before concluding the insertion tool itself failed.

There were 27 failed semantic-call observations across the after-fix failing jobs: inspect-symbol 12, verify 8, replace-body 4, lookup 1, insert-function 1, batch 1. The eight verify rejections came from host safety review because workspace verification could modify protected tests; they are not semedit operation failures. Other examples include missing/incorrect symbols, an invalid insert-function parameter, and a batch child edit error. None indicates whitespace-only defects.

## Run IDs for inspection

After-fix missing-`NormalizeKind` cases:

- Repeat 1: `job-898d4060142c63a1`, `job-9bb30a2e9f3ad859`
- Repeat 5: `job-2ce5f46460c5e3f8`
- Repeat 10: `job-4cb4df574c9a14bf`

After-fix no-oracle case: repeat 10, `job-6e86f0cc6be5a6f0`.

The 17 hidden delivery-test failures are recorded in the per-repeat reports for repeats 1–10. See the matching `codex-gpt-6-luna-medium-repeat-N-mcp-server-instructions-prescriptive-semedit-arm-restrict-write.md` files in this directory. Their job IDs are also present in the JSON results.

## Best next inspection targets

1. Fix the interpretation: probes remain in the sink after normalization; “control” means classification, not dropping.
2. Make the required exported helper contract explicit and check why successful insertion calls can still leave `NormalizeKind` absent.
3. Include `semantic_insert_function` in expected-tool telemetry for this fixture; examine the 16 failing runs missing one or more currently expected navigation/edit tools.
4. Reduce repeated workspace-wide `semantic_verify` attempts under test-edit restrictions; use an explicitly read-only verification path where available.
