<!-- WHY: Keep infrequent metrics and execution-trace details available on demand without loading them into routine operation routing. -->
# semedit Timing and Trace Reference

Call `semantic_metrics` with `limit: 1` immediately after the operation. Read structured `records[].metrics.phases`.

Go calls report `gopls.request.<method>` and `gopls.notify.<method>` with attempted-call counts and cumulative elapsed milliseconds. `gopls.send.<method>` and `gopls.wait.<method>` separate transport completion from response waiting within each request. Exact `go.*` phases locate environment setup, package discovery, session synchronization, revision checks, and publication. Within fingerprinting, `go.scope.embed_patterns` and `go.scope.hash_inputs` separate directive scanning from input hashing.

These phases include waiting and overlap their enclosing operation phases; they are not CPU measurements or disjoint totals. When the host enables Go execution tracing, the same scopes appear as named timeline regions. Ordinary MCP output does not contain trace events.
