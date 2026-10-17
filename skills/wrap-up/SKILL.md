---
name: wrap-up
description: Wrap up a semedit session by logging semantic-tool QoL, verifying and merging linked-worktree changes into local main, removing the completed worktree, and archiving the chat.
---

<!-- This skill preserves semantic-tool evidence and closes completed sessions through verified integration, worktree cleanup, and chat archival. -->

# Wrap up a session

Invoke as `$wrap-up` or ask to wrap up this thread/session. Creating or editing this skill does not invoke it. Follow an explicit narrower request (for example, log only, do not merge, keep the worktree, or keep the chat open).

## Assess semantic-tool quality of life

1. Identify the current repository root, branch or detached HEAD, and linked-worktree status using `git rev-parse --show-toplevel`, `git rev-parse --absolute-git-dir`, `git rev-parse --path-format=absolute --git-common-dir`, and `git worktree list --porcelain`. Different Git and common directories identify a linked worktree. Inspect tracked, staged, and untracked changes before writing.
2. Review the current thread's available history, tool arguments, results, retries, and follow-up checks. Use the [local transcript extractor](scripts/extract_semantic_calls.py) as described below when Codex logs are available; otherwise retrieve older turns through available thread-history tools. Include every actually invoked `semantic_*` operation, including navigation, verification, failed attempts, calls made through orchestration/CLI helpers, and child operations in `semantic_batch`. Do not count catalog listings or suggested calls as execution. Give each attempt a stable call/turn reference or chronological ordinal; associate batch children with their parent rather than double-counting them.
3. Assess the observed agent quality-of-life effect for each attempt: beneficial, neutral, harmful, or unknown. State the intent and target, outcome and mutation state, effort avoided, and friction introduced (discovery, arguments, output volume, latency, retries, context, correctness, or manual repair). A successful call can still be harmful; a useful validation rejection can be beneficial. Compare with the smallest plausible alternative, label estimates, and do not invent token/time savings. Distinguish tool defects, caller errors, expected contract rejections, and environment failures. Report unknown root causes as unknown.
4. Read `docs/SUBOPTIMAL_TOOLS.md` and `docs/MANUAL_EDITS.md` in this checkout. These versioned repository logs are the shared destination; append here and share through the merge rather than editing a sibling checkout. Preserve existing entries and uncommitted changes. Add a dated session assessment to `docs/SUBOPTIMAL_TOOLS.md`, including positive and neutral results. Reference existing incident entries instead of duplicating them. Record manual source-edit fallbacks and capability gaps in `docs/MANUAL_EDITS.md` using its existing format. Documentation-only edits need no invented semantic-tool failure.
5. Cover each attempt in the assessment. Group identical calls only when all references, counts, and common outcomes remain explicit; split retries or differing outcomes. Include this workflow's semantic calls too, and reconcile the inventory before final verification. Record the available-history boundary and omitted/unrecoverable portions; never claim exhaustive coverage of unavailable history. If there were no semantic calls, record that fact. Update the same session entry on repeated wrap-up invocations, adding newly observed calls rather than duplicating prior assessments.

### Read local transcript evidence

Run the helper from the repository root with Python 3.9+:

```sh
python3 skills/wrap-up/scripts/extract_semantic_calls.py \
  --thread-id <UUID-or-codex-URL> --include-children --limit 20
```

It reads `$CODEX_HOME` (default `~/.codex`); use `--codex-home` for another local log root. Follow `page.next_cursor` with `--cursor`, repeating the original thread/home/child options, until `page.has_more` is false. Pages preserve frozen source prefixes, tolerate later appends, and reject rewritten prefixes. Start a fresh traversal to include calls made during wrap-up. Results preserve the recorded data without additional clipping: save large pages under `.scratch/` and inspect selected fields instead of overflowing tool output.

Read `recorded_item` for exact arguments, status, result, and engine duration. Item start/completion timestamps measure the event envelope separately. Batch `children` pair requests with recorded outcomes where decodable; missing outcomes remain unknown, and child durations are unavailable. Review `command_review_candidate` items manually for CLI/helper invocations; mentions, catalog listings, and test-generated calls do not prove execution.

Report coverage limitations, missing child logs, absent results, and undecodable or truncated historical evidence explicitly. The extractor covers retained canonical completed MCP events across compaction and pairs direct semantic requests/responses in legacy logs without those events. Mixed legacy/canonical records with unmatched IDs require manual reconciliation to avoid double-counting. Calls embedded in unrecorded orchestration or missing transcripts remain outside proven coverage. `read_thread` pagination alone is insufficient when it omits MCP results.

### Required efficiency and runtime comparisons

For every assessed attempt, name the applicable local built-in alternative and the steps needed to reach the same observable result. Examples include shell `rg` plus file reads for navigation, `apply_patch` plus formatting and checks for edits, or a local checker for verification. Prefer tools actually available in the session. If no equivalent exists, state the capability gap; do not compare a semantic rename with an unchecked text replacement as though their guarantees were equivalent.

Assess agent efficiency separately from runtime. Compare tool-call count, round trips, input preparation, output/context volume, retries, manual repairs, and verification effort across the complete workflows. Include semantic discovery/setup costs and built-in search/read/edit/check costs. Quantify observed counts or sizes when available; label hypothetical baseline steps and estimates explicitly. Record whether the semantic workflow improved, matched, or worsened agent efficiency, or whether evidence is insufficient, with the reason. A faster command does not by itself establish lower agent effort.

Record semantic and local built-in wall-clock runtimes in milliseconds or seconds, plus their measurement source and scope. Use complete call timestamps or explicit timing results; distinguish end-to-end latency from engine-only execution, and do not treat a partial orchestration yield or polling wait as the full runtime. Include setup, transport, retries, repair, and equivalent checks in workflow totals when measured. For batches, record the parent runtime and mark child timing unavailable unless separately instrumented; never assign the whole batch duration to each child or sum overlapping calls as elapsed time.

Use existing comparable measurements first. If a safe, bounded comparison is practical, time the built-in workflow on the same input and scope. Read-only operations may run locally; replay mutations only in isolated fixtures under `.scratch/`, starting both workflows from identical pre-operation state and checking equivalent results. Never replay a mutation on the live checkout just to obtain timing. Record relevant environment, warm/cold server and cache state, timing boundaries, and sample count. Include the cost of wrap-up-only measurement separately from the original session workflow. Do not launch a broad benchmark campaign as part of wrap-up.

Specify both runtimes for every attempt, using `unavailable` with a concrete reason where measurement cannot be recovered or safely obtained. Separate measured results from estimates; never present an estimated local baseline as measured. Calculate a semantic/built-in runtime ratio only for comparable measured scopes with a nonzero baseline. Keep unmatched correctness guarantees or timing boundaries explicit, and mark speed comparisons inconclusive when they prevent a fair comparison. Missing timing evidence limits the conclusion but does not block the remaining wrap-up steps.

Suggested assessment format (adapt to the current log):

```markdown
## YYYY-MM-DD: session wrap-up, <thread title or ID>

- Context: branch/HEAD, workspace, task, history coverage.
- Totals: attempts by rating; batch parents and children counted separately.

| Calls | Tool / target / intent | Observed outcome / mutation state | Built-in alternative / agent efficiency | QoL effect and evidence | Friction / workaround / cause |
| :--- | :--- | :--- | :--- | :--- | :--- |
| <call reference> | <operation and target> | <result> | <baseline steps, effort comparison, evidence> | <rating and concrete effect> | <details or none> |

| Calls | Semantic runtime | Local built-in runtime | Scope / timing source / samples / cache state | Semantic / built-in ratio or limitation |
| :--- | :--- | :--- | :--- | :--- |
| <same call reference> | <measured duration or unavailable: reason> | <measured duration or unavailable: reason> | <measurement evidence and comparability> | <ratio or why inconclusive> |

- Net effect: agent-efficiency and runtime conclusions separately, what helped, what cost effort, and actionable improvements.
- Related incidents/manual fallbacks: existing log references, if any.
```

Repository rules still require immediate logging when a tool fails; wrap-up reconciles those records rather than deferring incident logging until the end. Publish log changes atomically with an advancing mtime, following `AGENTS.md`.

## Verify and integrate a linked worktree

A request to run this workflow authorizes committing all changes in the current session checkout, merging into local `main`, removing the completed linked worktree, and archiving this chat. Do not request the same authorization again. This does not authorize publishing, pushing, deleting branches, removing the primary or another chat's checkout, or overwriting another chat's work. The friction-log and skill-file exception below permits reconciling and committing only those destination edits when needed for integration; other changes in another checkout remain outside this authorization.

If this is the primary checkout, finish the logs and run `make check` as required by repository policy; leave changes uncommitted unless the user also requested a commit. Do not switch branches to simulate a worktree merge.

For a linked worktree:

1. Confirm the checkout belongs to this thread using its workspace, branch, and task evidence. Locate local `main` with `git worktree list --porcelain`; inspect its branch, HEAD, and staged, unstaged, and untracked changes. Dirty friction logs (`docs/SUBOPTIMAL_TOOLS.md`, `docs/MANUAL_EDITS.md`) and repository skill files under `skills/` or `.agents/skills/` do not block integration. These edits may be merged with the session changes, preserving both sides and existing entries; this exception permits committing the reconciled log/skill changes when needed. Do not overwrite or drop them, and recheck their preimages before publication so concurrent changes are not lost. For other changed paths or an advanced `main`, update the session checkout/worktree: rebase its branch onto current committed `main`, apply or replay the session changes, resolve conflicts there, and rerun the verification gate before retrying integration. Leave unrelated dirty destination changes untouched; if they still prevent integration, complete the verified session commit and report that specific blocker. Never stash, discard, or commit non-exempt destination changes. If local `main` is missing or is actively changing during integration, preserve independent progress and report the blocker. If the branch exists but no checkout has `main`, use an isolated integration worktree under `.scratch/worktrees/` in the repository root, subject to workspace permissions.
2. Finish all source, documentation, and friction-log changes. Run `make check` from this worktree and require exit status zero. Capture the result and the resulting diff/status in thread evidence. Repository checks apply fixes and may change files: inspect those changes, and rerun `make check` until a successful run makes no further versioned or nonignored untracked content changes. Any subsequent edit invalidates that evidence, including log edits. A remembered run or file timestamp alone is insufficient; if final-state coverage cannot be proved, rerun.
3. Review the complete diff, staged changes, and nonignored untracked files. Commit all current-checkout changes, including the logs, using `git add --all` and a descriptive commit message. Do not force-add ignored scratch files or manufacture an empty commit. For detached HEAD, create a uniquely named `codex/` branch to retain the work. Record the resulting commit SHA. Verify the checkout is clean and commit hooks did not introduce unchecked content; if they did, check and commit the changes before integration.
4. Recheck the destination's branch, HEAD, and dirty-path classification immediately before merging. Integrate the recorded source commit with `git -C <main-checkout> merge --ff-only <source-commit>` when possible. Allowed log/skill edits may be reconciled rather than treated as a cleanliness blocker; preserve their contents and staged/unstaged evidence until the reconciled result is checked and committed. If histories diverged or other integration changes are needed, rebase the session branch onto current committed `main`, apply or replay the session changes, and resolve conflicts in the session worktree using both sides' intent. Reconcile any allowed destination log/skill edits there as needed. Run `make check` again until the combined tree passes without further content changes, commit the verified result, and retry the fast-forward into `main`. Do not reuse verification from before the rebase or conflict resolution. If a conflict needs a user decision, preserve the state and report it; never choose a side wholesale or use a destructive Git operation. Never bypass another worktree's branch protection with `--ignore-other-worktrees`.
5. If destination `main` moves during integration, rebase the session branch onto the new committed `main`, apply or replay the session changes, and rerun the verification gate on the combined result; do not force the ref. Verify the source commit is an ancestor of `main`, report the final `main` SHA, and confirm both checkouts are clean. A fast-forward publishes the already checked tree; do not edit logs afterward just to record completion. If the destination differs from the checked tree or any post-merge action changes content, run `make check` there and handle resulting changes before declaring completion.

Stop integration on a failing check or unresolved conflict. Preserve completed logs and commits and report the exact remaining step. Do not call a partially completed wrap-up successful.

## Remove the completed worktree and archive the chat

Perform cleanup only after the requested work and checks succeed. For a linked worktree, require the final committed HEAD to be an ancestor of local `main`, with no staged, unstaged, or nonignored untracked changes. Preserve any needed ignored artifacts before removal; disposable build caches need no preservation. Never remove the primary checkout or a checkout shared with another active chat.

Before either managed archival or Git worktree removal, clear any disposable Go module cache inside the completed worktree. Go makes cached module files read-only, which can prevent worktree deletion. Discover the actual cache path from the session's build configuration or environment; for this repository it may be `.scratch/go/mod`. Resolve the cache and worktree paths and confirm the cache is strictly inside that worktree, with no symlink escape. From a surviving checkout, run `GOMODCACHE='<absolute-worktree-local-cache-path>' go clean -modcache` and require successful completion before proceeding. This command handles read-only module files. Skip an absent cache; leave external or shared caches untouched. If cleanup fails, preserve the worktree registration, leave the chat open, and report the error.

1. For a managed worktree attached to this chat, use `list_artifacts` to identify its exact worktree identity key, then call `archive_worktree` with that key. This saves a recoverable snapshot and removes the checkout; it does not archive the chat. Respect primary, pinned, shared, submodule, and embedded-repository restrictions. Do not bypass a managed-worktree rejection with shell deletion.
2. For an ordinary Git linked worktree, run `git -C <main-checkout> worktree remove <completed-worktree-path>` from a surviving checkout. Also remove any clean temporary integration worktree created by this workflow once its purpose is complete. Recheck ownership and status immediately before removal, use exact discovered paths, and never use `--force`, `rm -rf`, or a repository-wide cleanup command. Keep the branch and its commits. If the current working directory was removed, use the surviving checkout explicitly for any remaining shell commands.
3. Confirm removal succeeded through the tool result and `git worktree list --porcelain`. If removal or artifact preservation fails, leave the chat open and report the blocker. If this is the primary checkout, skip worktree removal; its verified logs may remain uncommitted as specified above.
4. Once all applicable steps succeed, call `set_thread_archived` with `archived: true`, omitting `threadId` to target this chat. Inspect the result and report failure explicitly. If the archival tool is unavailable, report that limitation and leave the chat open; do not substitute a raw archive directive. Do not archive a chat with failing checks, unresolved integration, or incomplete required cleanup.

## Final report

State semantic-call coverage, agent-efficiency findings, and measured runtime comparisons or timing limitations. Link the changed logs in a surviving checkout, give the final check result, and report commit/merge SHAs, worktree removal, and chat archival or the concrete blocker. Keep claims bounded by available evidence.
