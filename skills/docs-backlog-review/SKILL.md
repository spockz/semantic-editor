---
name: docs-backlog-review
description: Review this repository's ADRs, research questions, tool-friction logs, and docs bookkeeping to identify actionable work and stale entries.
---

<!-- This skill exists to make the repository's recurring documentation backlog review consistent; it keeps evidence gathering and status triage in one repo-local workflow. -->

# Review documentation backlog

Use this skill when asked what remains to do based on `docs/`, or when asked to refresh that assessment. Produce a read-only, evidence-linked triage unless the user separately asks to edit the documentation.

## Sources

Start with these indexes and logs:

- `docs/adr/README.md`: accepted and proposed decisions.
- `docs/research/README.md`: open and resolved research questions.
- `docs/SUBOPTIMAL_TOOLS.md`: semantic-tool friction and proposed fixes.
- `docs/MANUAL_EDITS.md`: historical manual edits and capability gaps.
- `docs/todo.md`: unstructured work notes.

Then inspect the individual ADRs, RQs, benchmark notes, or other `docs/*.md` records needed to confirm a proposed item or dependency. Read current implementation only when necessary to distinguish completed work from a real gap; do not infer completion from an accepted ADR alone.

## Repeatable review

1. Check `git status --short -- docs` and `git diff -- docs` first. Read relevant untracked files directly. Separate current uncommitted documentation changes from committed backlog, and avoid overwriting or claiming ownership of them.
2. Count proposed/open ADRs and open RQs from their indexes. Treat index status as a starting signal, not proof that every open RQ is funded implementation work.
3. Search for other bookkeeping, including `TODO`, `FIXME`, `TBD`, follow-ups, deferred work, and references to unresolved behavior:

   ```sh
   rg -n -i 'TODO|FIXME|TBD|follow[- ]?up|defer|future work|remaining|unresolved' docs -g '*.md'
   ```

4. Check for duplicate friction IDs:

   ```sh
   awk -F'|' '/^\| \*\*ST-/ {gsub(/[ *]/, "", $2); print $2}' docs/SUBOPTIMAL_TOOLS.md | sort | uniq -c | awk '$1 > 1'
   ```

5. Compare open records with later accepted ADRs, resolved RQs, implementation notes, and newer research. Mark each candidate as actionable, dependent, superseded/likely stale, or needing evidence before its status can be changed.
6. For friction entries, distinguish a product defect from an expected contract rejection, caller mistake, environment limitation, or resolved issue. Do not repeat a proposed fix as outstanding without checking whether the log or current implementation shows it was completed.
7. Group related work and state dependencies. Prioritize correctness and false-success risks, then explicit decisions and cross-cutting dependencies, then bounded feature and research work. Keep speculative or low-priority questions separate from concrete commitments.

## Report

Give a concise snapshot, then group the remaining work by priority or dependency. For each significant item, include its ID, what remains, why it matters, and the local document evidence. Call out bookkeeping cleanup separately from product work. Mention uncommitted documentation changes when present.

Use clickable links to the relevant repository files in the final response. Do not modify ADR/RQ indexes, status fields, friction logs, or TODO notes during an assessment; changing those records is a separate task. If the user asks for those edits, preserve the repository's ADR/RQ index atomicity and file-mutation rules in `AGENTS.md`.
