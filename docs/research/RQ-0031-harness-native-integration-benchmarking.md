# RQ-0031: Harness-Native Integration Benchmarking

* **Status**: Open
* **Date**: 2026-09-21
* **Category**: Benchmarks & Integration

## 1. Question

How should `tools/benchmark-harness` execute each agent target with that
target's native `semedit` integration artifacts, including MCP configuration,
instructions, future skills, and future plugins, while keeping trials
reproducible and distinguishing their effect from the semantic editor itself?

## 2. Scope

This is a benchmark-harness follow-up, not a requirement for delivering the
`semedit` MCP server or its harness installation commands. It begins only once
a target-specific artifact exists.

The target under test is the user-facing integration as installed for that
harness. Persistent user installation, marketplace publication, and the
correctness of an installer are separate concerns. The direct `semedit`
control, correctness oracle, and separation of edit latency from quality-check
latency remain defined by RQ-0014 and RQ-0015.

## 3. Problem

MCP standardizes server communication, but a harness determines how a server
is registered, how tools are exposed, and how instructions or skills enter the
model context. A benchmark that silently inherits a developer's global
configuration cannot establish which artifact affected the trajectory. A
benchmark that replaces native artifacts with an ad hoc prompt likewise does
not measure the delivered integration.

The harness must therefore reproduce each target's intended integration without
making plugin installation, mutable user configuration, or marketplace state a
hidden trial variable.

## 4. Candidate Matrix Extension

Extend the benchmark tuple only when an integration variation is being tested:

```text
Run = <Harness, Model, ReasoningLevel, IngressMode, IntegrationMode, TestCase>
```

`IntegrationMode` is one of:

| Mode | Server availability | Steering artifact | Purpose |
| :--- | :--- | :--- | :--- |
| `baseline` | No `semedit` server | None | Strong text-edit control. |
| `mcp_only` | Native ephemeral MCP registration | None | Measures tool schema and results alone. |
| `mcp_instructions` | Native ephemeral MCP registration | Harness-native instructions | Measures always-on rule delivery. |
| `mcp_skill` | Native ephemeral MCP registration | Installed skill | Measures progressive skill delivery. |
| `plugin` | Native plugin or extension | Its declared assets | Measures the packaged user experience. |

Modes are separate treatments. They must not be collapsed into a single
"semedit" result, and unavailable modes are reported as unavailable rather
than fabricated with a generic prompt.

## 5. Execution Contract

Each harness adapter supplies a versioned integration fixture containing only
the target's native artifacts and a manifest with their digests. For every
trial, the benchmark runner:

1. Creates a fresh fixture workspace and a fresh harness configuration area.
2. Renders or links the native MCP configuration for the preinstalled
   `semedit` binary without mutating the user's persistent configuration.
3. Installs or exposes only the instructions, skills, or plugin declared by
   the selected `IntegrationMode`.
4. Verifies server connection and captures the discovered, normalized tool
   names before the task prompt is submitted.
5. Runs the unmodified task prompt and existing hidden oracle.
6. Records artifact digests, harness version, model settings, effective tool
   catalog, and evidence of semantic tool calls alongside normal telemetry.

An adapter must use the harness's own ephemeral or isolated configuration
mechanism where it exists. For example, Copilot's session MCP configuration is
not interchangeable with a Codex project TOML file. If a harness cannot isolate
its plugin or skill state while preserving authentication, its plugin mode is
deferred rather than falling back to the developer's global setup.

## 6. Fairness and Safety Invariants

1. The baseline receives neither `semedit` tools nor semantic-edit steering.
2. The task prompt, fixture, model, reasoning level, permissions, timeout, and
   oracle are identical across modes unless the tested native integration itself
   changes one of those properties; that difference is recorded.
3. A semantic-arm success is credited as MCP-assisted only when the normalized
   trace shows at least one mutating `semantic_*` call. Tool discovery alone is
   insufficient.
4. The runner never writes persistent harness configuration, installs global
   plugins, or updates marketplace packages as part of a trial.
5. Secrets and harness authentication remain outside recorded artifacts and
   result telemetry.
6. Plugins, hooks, and extensions are treated as executable supply-chain
   inputs. Their source revision and declared capabilities are recorded, and
   they require the same review boundary as any local executable.
7. Native tool-name rewriting and tool grouping are captured in telemetry so
   cross-harness reports do not assume that a displayed tool name equals the
   server's canonical `semantic_*` name.

## 7. Initial Evaluation Sequence

1. Add a Copilot CLI adapter for `baseline` and `mcp_only`, using its
   session-only MCP configuration mechanism.
2. Add a Codex CLI adapter for the same modes using an isolated configuration
   source instead of ambient user MCP state.
3. Validate the adapters with one fixture and deliberate negative controls:
   missing server, disabled server, wrong binary, and a server whose tools are
   discovered but never called.
4. Add `mcp_skill` only when the canonical `semedit` skill exists and is
   installable in both target-native locations.
5. Add plugin or extension modes only after each package format is published
   or locally reproducible with a pinned source revision.
6. Extend the target-adapter interface for Gemini CLI, OpenCode, and Pi only
   after their native artifact and isolation behavior are documented.

## 8. Open Questions

1. Can each harness expose authoritative token and tool-call telemetry in its
   non-interactive mode, or must a normalized transcript extractor be the
   fallback?
2. How can a harness authenticate in an isolated configuration area without
   copying credentials into benchmark artifacts?
3. Does an installed plugin add tools, hooks, or system instructions beyond
   the equivalent MCP-plus-skill mode, and how should those differences be
   reported?
4. Should the benchmark report present `IntegrationMode` as a sixth matrix
   dimension or as a named treatment group within ingress mode?
5. What is the smallest reproducible artifact fixture for a harness whose
   native plugin manager copies, resolves, or updates packages at install time?

## 9. Next Steps

1. Specify the target-adapter fixture manifest and result fields without
   changing the benchmark runner.
2. Prototype isolated `mcp_only` setup for Copilot CLI and Codex CLI.
3. Confirm that each prototype prevents ambient MCP, instruction, and skill
   state from affecting a trial.
4. Create an ADR only if the fixture format or isolation boundary becomes a
   stable cross-harness architecture contract.
