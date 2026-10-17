<!-- Records Copilot's persistent benchmark transport and local-model evidence boundary. -->

# ADR-0074: Copilot Cloud and Local Model Benchmark Transport

## Status

Accepted

## Date

2026-10-17

## Context

The benchmark needs to compare the same locally served model across coding
harnesses. Saluki is exposed by llama.cpp. The installed Copilot CLI supports
OpenAI Chat Completions for custom providers, which avoids the Responses
namespace-conversion incompatibility observed with Codex.

Restarting a CLI for each follow-up would restart MCP children and invalidate
the existing live-session comparison. ACP provides a continuing session and
Copilot-specific raw events for per-call usage and tool outcomes.

## Decision

Accept `copilot/<model>` for hosted Copilot models and
`copilot/local/<model>` for local inference, without effort suffix interpretation.
Run one Copilot ACP process and one session per benchmark job. Configure the
local provider through isolated environment and configuration, using offline
mode and Chat Completions over the run-owned local-model relay. Reuse the
existing relay and model-limit discovery for both Copilot and OpenCode.
Hosted targets use normal GitHub model routing and catalog limits, without
requiring a local endpoint or querying llama.cpp properties. Reject cloud
`auto` targets so benchmark identity names one selected model.

For hosted targets, accept supported token environment variables in Copilot's
documented precedence and pass only the selected token as a CLI secret. Keep
state fixture-local instead of importing saved Copilot account configuration.
Remove inherited offline and custom-provider settings for hosted runs.

Load semedit from fixture-local Copilot MCP configuration before creating the
ACP session. Deliver semedit-arm restrictions in repository instructions; the
baseline receives no generated restriction file or semedit server registration.
Preserve any existing fixture instructions when adding restrictions.

Require an authoritative cloud catalog advertising the requested canonical ID.
Select the wire model through ACP `session/set_config_option` before the first
prompt and require its returned current-model confirmation for cloud runs.
The legacy model switch can report success while discarding a non-applied native
switch result; environment defaults and RPC success alone do not establish identity.
Collect response text from standard ACP chunks. Subscribe to required Copilot
raw usage, tool, and error events, avoiding the raw assistant-message size cap.
Validate required per-call usage and
reconcile its sum with deltas of ACP's cumulative prompt usage. Normalize tool
starts and completions while retaining the original protocol evidence. Record
the transport as `copilot_acp_v1` in provenance and resume fingerprints.

## Invariants

* Follow-ups retain the original process, workspace, target, arm, and session ID.
  A failed session cannot silently restart or continue under another identity.
* Missing required usage, truncated generation, omitted critical event payloads,
  mismatched model/session identity, and incomplete tool observations are
  execution failures. Missing optional token buckets remain identified in
  provenance rather than being asserted as measured zeros.
* Baseline and semedit arms use the same selected model and provider mode.
  Local context/output limits are discovered and the wire model alias must
  match llama.cpp properties. Only the explicit `local/` routing prefix is
  removed when validating usage model identity.
* Local runs remove Copilot account and supported provider credentials from
  the inherited environment. Copilot state is isolated in both modes. The deny-default
  sandbox allows fixture writes, required runtime/toolchain reads, and only the
  fixed relay port for local model access. Hosted runs additionally permit
  outbound HTTPS and DNS, including from child tools. Unsupported containment
  fails explicitly. Hosted credentials are not written to runtime configuration.
* Shutdown reaps the owned Copilot process and its MCP descendants. Transcripts
  and MCP timing evidence survive fixture cleanup through existing shared code.

## Consequences

Copilot is a separate harness condition, not a replacement for Codex results.
The integration requires Copilot's ACP raw-event extension and a locally
installed runtime bundle; incompatible versions fail evidence checks. Runtime
containment currently depends on macOS Seatbelt, as does local OpenCode.
Local targets need no Copilot account or hosted inference credentials. Hosted
targets need an explicitly supplied supported GitHub token; they do not reuse
the installed CLI's saved account selection.

This extends [ADR-0067](0067-opencode-run-owned-model-relay.md) to a second
local-model client while preserving its fixed-upstream network boundary.
