# ADR-0043: Benchmark Run Aggregation and Best-Case Publication

Status: Accepted
Date: 2026-09-22

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
exactly one top-level user turn then wins. Model cost is credit consumption:
(uncached input tokens × input credits + cached input tokens × cached credits +
(reasoning tokens + visible output tokens) × output credits) / 1,000,000.
Thinking tokens are reasoning tokens and use the output rate. The declared
credits per one million tokens are:

| Model | Input | Cached input | Output |
| :--- | ---: | ---: | ---: |
| GPT-6 Astra | 250 | 25 | 1,250 |
| GPT-6 Sol | 50 | 5 | 250 |
| GPT-5.6 Sol | 100 | 10 | 500 |
| GPT-5.6 Terra | 50 | 5 | 300 |
| GPT-6 Luna | 2.5 | 0.25 | 12.5 |
| GPT-5.6 Luna | 5 | 0.5 | 30 |

Stable run identity breaks remaining ties.

Every published detailed and aggregate table also reports model-specific
credits when the target model has a declared rate schedule. This is the same
per-million-token credit cost used for best-case selection. Cache-adjusted
token units remain published telemetry, but do not rank best-case pairs.

Oracle outcome is absolute: no speed, token, or one-shot advantage can cause a
failed arm to outrank a passing arm. The tenfold override applies only after
the pair outcome class is equal.

The aggregate page reports minimum, maximum, and arithmetic mean for every
numeric telemetry metric and binary oracle outcomes, separately for each task,
target, prompt variant, MCP instruction mode, context variant, and arm.
Technical provenance remains auditable metadata and does not create aggregate
cells.

## Invariants

- A selected pair always comes from one recorded comparison and contains both
  oracle-evaluated arms.
- No non-outcome metric can override an oracle pass/fail difference.
- A cost-based speed override requires at least a tenfold model-cost
  improvement and equal pair outcome classes.
- One-shot status is considered only when outcome, speed, and model cost
  remain within their stated comparison bands.
- Aggregation never combines experimental conditions defined by ADR-0038.
- Aggregate measurements retain arm and context identity; no cross-arm average
  is presented as an MCP effect.
- A cached token contributes one tenth of an uncached token only to the
  published cache-adjusted-token metric, not to raw telemetry values or
  best-case selection.
- A reported model cost is measured in credits and only available for an explicitly
  declared target-model rate schedule; unavailable cost is never treated as
  zero.
- Root-level legacy result files remain publishable as the `legacy` run until
  all writers use explicit run identifiers.

## Consequences

The front page answers the product question using the most favorable observed
evidence without disguising it as an average. Readers can inspect variation in
the aggregate page and reproduce every displayed observation from its run
page. The five-percent speed and model-cost bands make cost and one-shot
completion relevant where elapsed-time and cost differences are practically
close. The tenfold override captures exceptional cost reductions without
allowing any resource metric to outweigh correctness.
