# RQ-0039: Benchmark Policy Instruction Transport

* **Status**: Resolved
* **Date**: 2026-09-26
* **Category**: Benchmarks & Integration

## Question

Can the benchmark's `read`, `write`, and `readwrite` restrictions travel through a provider CLI's developer-instruction channel instead of being appended to initial and corrective user prompts?

This investigation concerns benchmark restriction delivery. The production MCP server's `PrescriptiveInstructions` remains intact and separately selectable through the existing MCP instruction experiment condition.

## Evidence

### Codex

Installed version: `codex-cli 0.155.1`.

The [official configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference) documents `developer_instructions` as an optional additional session instruction string. The installed `codex exec --help` and `codex exec resume --help` both expose `-c key=value`, parsed as TOML. The [official CLI reference](https://learn.chatgpt.com/docs/developer-commands?surface=cli) documents configuration overrides and non-interactive continuation.

A no-inference probe used the installed prompt renderer:

```sh
codex -c 'developer_instructions="BENCHMARK_POLICY_PROBE: Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed."' \
  debug prompt-input 'BENCHMARK_TASK_PROBE: Rename Old to New.'
```

The rendered input contained the policy in a `developer` message and the task in a separate `user` message. The sanitized receipt is in `.scratch/developer-instructions-research/codex-prompt-input.json`; that scratch artifact is local evidence rather than a repository dependency. The command only rendered context. No provider inference or benchmark trial was run. The first sandboxed attempt could not access Codex's state directory; the same renderer succeeded with approved local access.

The resulting CLI integration shape is:

```sh
codex exec --json \
  -c 'developer_instructions="<resolved policy text>"' \
  '<task prompt>'

codex exec resume --json \
  -c 'developer_instructions="<same resolved policy text>"' \
  '<session ID>' '<corrective task prompt>'
```

These are illustrative commands, not executed model calls. Other existing harness flags and MCP overrides still apply. Reapply the explicit override on each process invocation; do not rely on undocumented persistence across resumes. This investigation verified role rendering and CLI syntax, not a resumed request sent to a model.

Use `developer_instructions` rather than `model_instructions_file`: the latter replaces built-in instructions according to the configuration reference. A CLI override also replaces any existing value of the same config key. Preserve or deliberately control the common developer instructions across both arms before composing the semedit policy. Additive relative to native base instructions does not mean automatically appended to an ambient `developer_instructions` value.

### OpenCode

Installed version: `1.18.30`. The [rules documentation](https://opencode.ai/docs/rules/) supports an `instructions` array of file paths, combined with existing `AGENTS.md` instructions. The [configuration documentation](https://opencode.ai/docs/config/) supports a per-process `OPENCODE_CONFIG` path. The harness already supplies such a config in `opencode_driver.go:openCodeEnvironment`.

Add an absolute path to a generated per-job policy file to that config's `instructions` array. The file can live outside the extracted fixture. Preserve the same config and file on every initial, corrective, and diagnostic invocation, including `run --session <id>`.

Version-matched source confirms the composition: [instruction.ts](https://github.com/anomalyco/opencode/blob/v1.18.30/packages/opencode/src/session/instruction.ts) reads configured absolute paths; [prompt.ts](https://github.com/anomalyco/opencode/blob/v1.18.30/packages/opencode/src/session/prompt.ts) calls `instruction.system()` in the request loop; [llm/request.ts](https://github.com/anomalyco/opencode/blob/v1.18.30/packages/opencode/src/session/llm/request.ts) appends this content after the native provider or selected agent prompt. The eventual API representation is provider-dependent, including system messages or an instructions parameter. This is additive system context, not a universal developer-role field.

Avoid changing `agent.prompt`, which replaces the native provider prompt. The installed CLI does expose `--agent`, but selecting a different agent is unnecessary. Instruction-file lookup/read failures can become missing content internally, so the benchmark must validate its own policy file and capture delivery evidence. Source inspection establishes intended composition; no live initial/resume request was captured.

### Agy

Installed version: `1.2.11`. The [rules documentation](https://www.agy.dev/docs/rules/) specifies that `.agents/rules/*.md` with `trigger: always_on` injects full content into the system prompt on every turn. Generate `.agents/rules/benchmark-policy.md` in the extracted semedit workspace before launch, retaining it for all follow-ups:

```markdown
---
trigger: always_on
description: Semedit benchmark source inspection and editing policy
---

<centrally resolved policy text>
```

The tracked fixture archive stays unchanged. Baseline gets no benchmark policy file. Validate the frontmatter: missing or invalid triggers may silently discard a rule. The headless CLI also has `--agent`, but a custom agent profile changes more than additive policy delivery. Rule delivery is documented; a live resumed request remains unverified.

### Pi

Installed version: `0.85.1`. Its [versioned CLI reference](https://github.com/earendil-works/pi/blob/v0.85.1/packages/coding-agent/README.md#other-options) and installed help expose repeatable `--append-system-prompt <text>`. Use this instead of `--system-prompt`, which replaces the default prompt:

```sh
pi --print --mode json --append-system-prompt '<policy text>' '<task prompt>'
pi --print --mode json --session '<session ID or file>' \
  --append-system-prompt '<same policy text>' '<follow-up prompt>'
```

The installed `dist/main.js` passes the CLI value into runtime resource loading after session selection, including resumed sessions. `dist/core/system-prompt.js` appends it to the base prompt. Reapply the option on each process invocation; the task prompt and fixture need no policy additions. Prefer inline text over a path: the loader can turn an unreadable path into literal prompt text.

One isolation caveat: explicit CLI append sources bypass automatic `APPEND_SYSTEM.md` discovery in `dist/core/resource-loader.js`. Preserve the intended common append content explicitly in both arms, or isolate it equally. Native base instructions are preserved, but ambient append instructions are not automatically composed with this flag. Pi is not currently a supported benchmark adapter; this records its capability, not completed Pi integration. No model inference was run.

### GitHub Copilot CLI

Installed version: `1.0.88`, checked with `copilot --version` and `--help`. The standalone executable first needed permission to unpack its bundled package into the local cache. Subsequent discovery probes ran in the sandbox with an isolated `COPILOT_HOME`, offline mode, and automatic updates disabled. No model inference was run.

The installed CLI exposes `COPILOT_CUSTOM_INSTRUCTIONS_DIRS`, `--resume`, `--session-id`, `--output-format json`, and `instruction list --json`. Its public help does not expose an inline additive system/developer-instruction flag. The [custom-instruction documentation](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions) supports `.github/copilot-instructions.md`, `AGENTS.md`, and additional directories. Instructions are combined; changes require a new or restarted/resumed session.

A local no-inference probe created independent Git fixtures for baseline and each policy, then ran `copilot --no-auto-update -C <fixture> instruction list --json`:

* Baseline returned an empty instruction list.
* Each of `read`, `write`, and `readwrite` returned `.github/copilot-instructions.md` with `defaultDisabled: false`.
* A separate fixture with `COPILOT_CUSTOM_INSTRUCTIONS_DIRS` pointing to a generated external `AGENTS.md` returned a generic nested-agent instruction source. This listing does not establish exact effective content.

The local receipt is `.scratch/developer-instructions-research/copilot/instruction-discovery.json`. Discovery proves source registration, not exact model-visible instruction content, effective message role, or delivery on resumed model requests. These still need protocol-level verification. Copilot has no benchmark adapter yet.

For the proposed adapter, file definition composes the central policy with preserved common content in the extracted fixture's `.github/copilot-instructions.md`. Command construction preserves this working directory and uses an explicit session identifier for follow-ups. An external instruction directory is an alternative that needs the environment override on every invocation. Record the selected route separately; do not label it developer-role delivery.

Delegation needs separate coverage. The [CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference#repository-custom-instructions-for-subagents) says the main session and general-purpose subagent receive repository instructions, while built-in explore/task/code-review subagents do not. Custom subagents require `include-custom-instructions: true`, and even then do not inherit the main session's extra instruction directories. Neither file route alone proves policy coverage across all delegated work. A future adapter must explicitly validate and record its delegation configuration rather than claim universal enforcement.

## Proposed integration boundary

Keep policy text generation central in `session_policy.go`, using separate read and write clauses plus shared build/test permission. Separate the resolved task prompt from resolved policy instructions in `AgentExecution`. The provider adapter selects a known native instruction mechanism, otherwise explicit user-prompt fallback, as requested by the user. It must not invent policy wording. Record the resolved route; fallback is an intentional experimental condition, never mislabeled as native delivery. A failure of a selected native route is an error, not grounds for silent fallback.

For a Codex-specific implementation, `runCodex` in `codex_driver.go` would pass the resolved policy through `-c developer_instructions=...` on initial and resumed task calls. `prepareAgentPromptWithPolicy` and `ResolveAgentExecution` would stop appending that policy to user prompts for this explicit transport. Baseline receives no benchmark policy. Preserve the existing task prompt, fixture guards, completion marker, and production MCP instructions unless changing those separately is intentional.

### Launch construction and fixture setup

The user requested two separate adapter responsibilities. Pass one resolved launch-parameter structure into both, containing the benchmark execution condition, target/model/effort, arm, central policy text and selected transport, instruction mode, workspace/runtime paths, and the current turn's prompt, phase, and resume identifier. Keep common instruction composition explicit in that structure rather than rediscovering it differently on each call.

* A command builder returns the executable, argument slice, working directory, and environment overrides. It does not launch a process or write files. Call it for initial, corrective, and diagnostic invocations so policy delivery cannot disappear on resume.
* A separate file-definition function returns the required generated files, their contents and modes, and whether they belong to the extracted fixture or per-job runtime directory. It does not alter tracked fixture archives. The session validates reserved-path collisions and writes these files atomically before snapshotting and launching. Retain and validate the same files for later turns; do not overwrite task changes to conceal a changed policy file.

Codex uses the invocation's developer-instruction override. Pi would use additive system-prompt arguments. OpenCode uses the invocation's config environment and generated config/policy files, which may live outside the fixture. Agy requires an always-on rule in the extracted fixture. Copilot would use generated repository instructions, subject to the delegation limits above. Baseline never receives a benchmark restriction file or argument, although ordinary provider setup files may still be needed. Unknown harnesses receive the explicit prompt fallback chosen during planning.

The session owns file creation, execution, transcript recording, measurement, and cleanup. The provider owns the launch syntax and file formats. Central policy generation owns the restriction wording; neither command nor file construction redefines read/write semantics. This is the proposed next implementation boundary, not an assertion that the current drivers already expose these two functions.

Build subprocess arguments as argument slices, with a correct TOML string encoder for the setting value. Do not interpolate policy text into a shell command. Render the resolved policy text and its delivery mechanism in the execution plan and retain them with result evidence.

Resolve delivery once during planning. Preflight generated instruction files and reserved-path collisions, and take the task workspace snapshot after harness setup. GPT-6 Sol reviewed these boundaries and the explicit fallback decision.

Instruction delivery changes priority and is an experimental condition. Record transport, effective role, and policy text version or digest; keep conditions separate in job/pair identities, result filenames, comparison keys, best-case selection, and browser aggregation. Both arms share the selected comparison condition, while applicability remains semedit-only. Hash policy content, not ephemeral absolute paths; per-job paths must not split comparison identity. Historical records without transport metadata remain explicit legacy or unspecified records, never silently developer-delivered.

The same selected policy and delivery route remain active during corrective turns and later diagnostic investigations. Diagnostic reflections remain outside benchmark metrics. They also need instruction compatibility: task-edit restrictions must not require tools during an explanation-only turn whose prompt forbids tool use. Define the phase scope in developer policy wording or use a demonstrated turn-scoped mechanism; a later lower-priority user prompt cannot be assumed to override developer instructions.

## Findings

Codex supports additional developer instructions; OpenCode supports additive configured instruction files; Agy documents always-on system rules; Pi supports additive system-prompt CLI arguments; Copilot supports custom instruction files with verified local discovery. These native channels preserve the task user prompt, but are different experimental transports. Unknown harnesses use an explicit, recorded user-prompt fallback.

The feasibility question is resolved with the evidence limits above. Transport implementation remains deferred. Preserve the common instruction base across both arms and the production MCP instructions. Generate policy wording centrally, apply it only to semedit, and keep delivery metadata in comparison conditions. Agy needs generated workspace rules; OpenCode can use a policy file outside the fixture; Codex and Pi need invocation settings only.

## Required verification before changing execution

1. Capture initial and resumed Codex model-visible inputs using a controlled local transport or equivalent protocol-level test. Assert the exact developer text, separate task user text, and preserved native instructions.
2. Exercise all three policies and baseline isolation at the actual CLI adapter boundary; prove policy text does not appear twice through user and developer messages.
3. Check ordinary corrective turns, compaction/continuation as supported, and explanation-only reflection without counting reflection telemetry.
4. Verify existing common developer instructions are preserved or intentionally isolated equally across arms; preserve production MCP instructions.
5. Prove policy transport and text identity remain distinct through serialization, comparison, publication, and historical loading.
6. Establish native composition and resume semantics separately for each provider. Test explicit user-prompt fallback for unknown harnesses, and reject native-delivery setup failures rather than silently changing transport. Verify generated instruction files remain present and unchanged across follow-ups.

## Related architecture

See the [benchmark architecture guide](../../tools/benchmark-harness/README.md), [ADR-0050](../adr/0050-benchmark-planning-sessions-and-arm-policy.md), and [RQ-0031](RQ-0031-harness-native-integration-benchmarking.md). No benchmark execution behavior changes as part of this investigation.
