# Luna medium edit scaling, 10 repeats

Source revision: `5c3edbf`. Target: `codex/gpt-6-luna/medium`. Each of the six
fixtures ran in the baseline and semedit arms, small context, default prompt,
10 repeats, concurrency 5, prescriptive MCP server instructions, semedit write
policy, and a five-minute job timeout. All 120 jobs passed their oracles.

| Distinct method renames | Baseline median wall time | Semedit median wall time | Median paired semedit time penalty | Baseline median total tokens | Semedit median total tokens | Median paired semedit token penalty |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 0, greeting | 4.48 s | 3.69 s | -0.14 s | 18,703 | 18,702 | -1 |
| 1 | 18.64 s | 34.02 s | +17.66 s | 77,510 | 154,555 | +68,254 |
| 2 | 20.03 s | 41.78 s | +23.64 s | 96,661 | 204,560 | +106,916 |
| 4 | 19.00 s | 62.05 s | +42.58 s | 97,142 | 213,846 | +109,273 |
| 8 | 20.06 s | 89.83 s | +66.71 s | 88,478 | 217,378 | +133,948 |
| 16 | 22.30 s | 147.84 s | +122.25 s | 99,179 | 265,038 | +153,080 |

Token totals are prompt plus output tokens and include cached prompt tokens. The
greeting probe's median was 12,032 cached prompt tokens, about 6,664 uncached
prompt tokens, and 7 output tokens per arm. No semantic tool ran in the greeting
probe, so it measures end-to-end task overhead with the server configured, not
semantic tool startup or a useful edit's fixed cost. The oracle checks that
regular workspace files stayed unchanged; it does not check the reply text.

For every edit count from 1 through 16, semedit was slower in all 10 paired
repeats. Semedit used a median of exactly one `semantic_rename` call per
requested rename, plus lookup and verification calls. Baseline runs often
applied the explicit mapping in one file-edit command. This fixture family
therefore finds no semedit time or token crossover in the measured range. Do
not extrapolate a break-even count from the greeting probe: the edit task and
tool-use path are different.
