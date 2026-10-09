<!-- Preserves exact recipe validation evidence outside measured model results for future analysis. -->

# go-zero-2463 explicit recipe validation

The pinned base is `63cfe60f1a7583865757780ebfe2a5309a542134`.
The exact prompt calls `semantic_batch` once with 29 ordered operations.
Every child succeeded, no diagnostics were introduced, and the normal private
acceptance oracle passed. Validation used local prepared source and semedit;
the human fix patch was not applied to the validated replay.

- `explicit-single-call.md`: the prompt embedded in the fixture.
- `mcp-recipe-single-call.json`: the complete ordered operation payload.
- `single-call-batch-result.json`: semantic execution evidence.
- `integrated-oracle-result.json`: acceptance result with the integrated fixture.
- `integrated-validation-receipt.json`: final prompt, recipe, and binary digests.
- `authoring/`: original prompt and receipts, before Markdown normalization and
  final harness rebuild. Normalization leaves the operation payload unchanged.

These are recipe-validation results. They do not measure a coding model's
performance or establish a globally fastest sequence. All five external recipes
are indexed in [the validation overview](../README.md).
