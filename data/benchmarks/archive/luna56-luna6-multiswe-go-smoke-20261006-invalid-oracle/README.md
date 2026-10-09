<!-- Preserves invalid smoke runs and their evidence for later harness analysis. -->

# Archived Multi-SWE-bench Go smoke runs

Archived on 2026-10-07. These results are invalid for task solve rates and model
comparisons. The private oracle copies silently skipped the dataset test patches
while `git apply` returned exit code 0. Earlier pilot runs also include repository
copy and sandbox setup failures.

All 11 original run directories retain their original JSON, manifests,
checkpoints, and rendered Markdown reports. `evidence/` preserves referenced transcripts, retained workspace
data, and available invocation logs. `archive-manifest.json` maps original evidence
paths to archived copies and records transcript checksums and any missing files.

This archive is outside `data/benchmarks/results`, so the publication loader does
not ingest it. A fresh Luna 6 run uses the corrected patch installation and harness
error handling.

The repository's Markdown autofixer reached archived reports on 2026-10-07.
Use JSON, checkpoints, and transcripts as canonical execution evidence. All 48
checksummed evidence items were reverified and matched. Archive and validation
Markdown now have formatter exclusions; the original validation prompt was
restored from its retained source copy and its checksum verified.
