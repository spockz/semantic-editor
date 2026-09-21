# RQ-0030: Java Maven Hermeticity and Network Policy

Status: Open

## Question

How should Maven cache reuse, dependency prefetching, and network policy evolve while preserving request-scoped trust and bounded execution?

## Scope

Measure the latency and reproducibility trade-offs of scratch-local versus explicitly supplied caches. Evaluate whether a future trusted prefetch operation can be safely separated from fixed compile/test actions.

The current bounded actions use a scratch-local Maven user home, configuration directory, and repository for every request. They run with network access disabled unless `allow_network=true`; stdout/stderr are bounded and failures retain a structured result plus log tail. A selected root must be canonically contained by the trusted request workspace. Cache reuse and controlled prefetching remain open follow-up work.
