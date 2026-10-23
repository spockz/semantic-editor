# ADR-0043: Benchmark Run Aggregation and Best-Case Publication

* **Status**: Accepted
* **Date**: 2026-09-22

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Publish per-run harness-recorded USD estimates with captured pricing metadata; never reprice historical results; aggregate by pricing ID and date; retain unavailable costs and oracle-first best-case selection. |
| description | Repeated benchmark executions produce independent paired observations, and a single execution may cover only part of the benchmark matrix. Publishing every observation on the primary empirical page would obscure the strongest measured MCP outcome, while publishing only that outcome would hide variance and raw evidence. Cached prompt tokens are materially cheaper than uncached input, so raw token totals do not adequately describe the cost trade-off of loading additional reusable context. |
| role | process |
| scope | tooling |
| area | benchmark |
| languages | generic |

## Context

Repeated benchmark executions produce independent paired observations, and a
single execution may cover only part of the benchmark matrix. Publishing every
observation on the primary empirical page would obscure the strongest measured
MCP outcome, while publishing only that outcome would hide variance and raw
evidence. Cached prompt tokens are materially cheaper than uncached input, so
raw token totals do not adequately describe the cost trade-off of loading
additional reusable context.

## Decision

Store each execution below `data/benchmarks/results/<run-id>/` and generate a
page for that run containing its publishable observations only. The primary
benchmark page selects one complete baseline/MCP pair for every task, target,
prompt variant, MCP instruction mode, and context variant across all runs.
It explicitly calls these best-case outcomes measured so far and links to the
complete run pages and aggregate statistics.

The primary page begins with headline best improvements for speed and
cache-adjusted token units, calculated only from selected pairs where both
arms passed their oracle. It also compares the baseline and MCP initial oracle
pass rates across all publishable standard-context paired observations, rather
than selected best-case outcomes or verified/self-correction contexts.
The primary page also shows the distribution of corrective interactive turns
attempted after the initial prompt, separately for baseline and MCP arms across
those same standard-context paired observations.

Selection ranks a pair by MCP benefit: MCP oracle pass while baseline fails,
then both pass, then both fail, then baseline pass while MCP fails. Within an
equal outcome class, a pair with both arms passing is preferred by relative
wall-clock improvement. Speed differences within five percentage points are
treated as comparable and are resolved by lower relative model cost. A cost
improvement of at least tenfold overrides a non-comparable wall-clock
regression; a smaller improvement does not. Cost differences within five
percentage points are also treated as comparable; a Semedit completion in
exactly one top-level user turn then wins. Cost comparisons use the harness's
per-run `estimated_cost_usd`; docgen does not infer cost from token counts or
apply current rates to historical runs. The harness also stores the pricing ID,
rate-card date, and estimate basis alongside each value. Runs without a recorded
estimate have unavailable cost and cannot decide cost-based ties.

Stable run identity breaks remaining ties.

Every published detailed and aggregate table reports the recorded USD estimate
when present. Aggregate cost groups stay separated by pricing ID and rate-card
date, and reports without recorded estimates remain unavailable rather than
being repriced. Cache-adjusted token units remain published telemetry, but do
not rank best-case pairs.

Oracle outcome is absolute: no speed, token, or one-shot advantage can cause a
failed arm to outrank a passing arm. The tenfold override applies only after
the pair outcome class is equal.

The aggregate overview links to one detail page per benchmark. Each detail page
reports the median, interquartile range, 90th percentile, and maximum for every
numeric telemetry metric, plus pass counts and rates for binary outcomes.
Quantiles use linear interpolation between adjacent ordered session
observations pooled across all included runs; pages never average per-run
quantiles. Arm identity uses the task, provider/model/effort target,
arm role, prompt variant and exact prompt configuration, MCP instruction mode,
restriction policy, context variant, and any additional recorded execution
configuration such as result variant or interactive mode. Run/session identity,
timestamps, provenance, and observed usage or outcome metrics never split an
arm. The aggregate also reports session and token totals, per-arm model cost
when the target has a declared rate schedule, and totals across arms with known
cost formulas. Unknown costs are explicitly unavailable and excluded from the
known-cost total.

## Invariants

* A selected pair always comes from one recorded comparison and contains both
  oracle-evaluated arms.
* No non-outcome metric can override an oracle pass/fail difference.
* A cost-based speed override requires at least a tenfold model-cost
  improvement and equal pair outcome classes.
* One-shot status is considered only when outcome, speed, and model cost
  remain within their stated comparison bands.
* Aggregation never combines experimental conditions defined by ADR-0038.
* Aggregate measurements retain arm and context identity; no cross-arm average
  is presented as an MCP effect.
* A cached token contributes one tenth of an uncached token only to the
  published cache-adjusted-token metric, not to raw telemetry values or
  best-case selection.
* A reported model cost is the per-run USD estimate recorded by the harness with
  its pricing metadata; docgen never recalculates it from a current rate card.
  Missing estimates remain unavailable and are never treated as zero.
* Repeated sessions with the same experiment-defining configuration share one
  aggregate arm across runs; changes to a recorded treatment field keep arms
  separate.
* Aggregate quantiles are recomputed from the pooled session-level values,
  while session/token/cost totals sum observations across the included arms.
* Root-level legacy result files remain publishable as the `legacy` run until
  all writers use explicit run identifiers.

## Consequences

The front page answers the product question using the most favorable observed
evidence without disguising it as an average. Readers can inspect variation in
per-benchmark detail pages and reproduce every displayed observation from its
run page. The compact overview provides cross-benchmark session, token, and
known-cost totals. The five-percent speed and model-cost bands make cost and one-shot
completion relevant where elapsed-time and cost differences are practically
close. The tenfold override captures exceptional cost reductions without
allowing any resource metric to outweigh correctness.
