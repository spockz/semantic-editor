# RQ-0032: Harness-Native Semantic Editor Installation

* **Status**: Open
* **Date**: 2026-09-21
* **Category**: Integration & Distribution

## 1. Question

How should `semedit` install and manage its local MCP integration for GitHub
Copilot and Codex first, while safely supporting user and workspace scope,
future steering skills, and later harness-native plugins or extensions?

## 2. Scope

This RQ covers the product-delivery assignment: making a preinstalled
`semedit` binary available to coding harnesses through their native
configuration mechanisms. Initial targets are GitHub Copilot and Codex. Gemini
CLI, OpenCode, and Pi follow only after the installation contract is proven.

Each target is independently installable and usable. A user selects one target
per installation request; installing Copilot must neither install nor require
the Codex integration, and the converse is equally true. The delivery order is
an implementation priority, not a bundled end-user installation.

It does not implement benchmark-harness execution or measure performance;
RQ-0031 owns evaluation of the delivered integrations. It also does not require
a skill to exist before MCP installation ships. When a canonical skill is
available, its target-specific installation becomes part of this contract.

## 3. Context

`semedit mcp` is the portable stdio server entry point, but MCP clients do not
share one configuration-file schema, precedence model, plugin format, or trust
model. Copilot and Codex both support local MCP servers, while their workspace
and user registrations differ. Plugins and extensions may bundle MCP
registration with skills and other executable behavior, but are not a portable
replacement for direct MCP configuration.

The unstructured project notes call for one CLI command that performs global or
workspace installation. The command must add convenience without turning a
harness configuration file into an unreviewed or destructive mutation surface.

## 4. Candidate Product Contract

The candidate public interface is:

```text
semedit install copilot --scope=user|workspace [options]
semedit install codex --scope=user|workspace [options]
semedit integration status <target>
semedit uninstall <target> --scope=user|workspace
```

The final spelling is intentionally open. Every target adapter receives the
same logical request:

| Field | Meaning |
| :--- | :--- |
| `target` | Harness-specific adapter, initially `copilot` or `codex`. |
| `scope` | `user` for a developer's configuration or `workspace` for a project-owned configuration. |
| `binary` | Absolute, preinstalled `semedit` executable to launch. |
| `mcp_profile` | Intended tool profile, initially evaluated as `mutations-only` or `full`. |
| `enabled_languages` | Optional future capability filter; omitted until the MCP server supports it. |
| `skills` | `auto`, `required`, or `off`; `auto` is a no-op until a skill exists. |
| `dry_run` | Reports the exact target files and semantic changes without writing. |

The adapter renders the target's native registration rather than inventing a
new runtime protocol. A future plugin or extension is a distributable wrapper
around the same server command and skill assets, not a replacement server.

## 5. Candidate Designs

### A. Documentation Only

Publish per-harness copy-and-paste commands and configuration snippets. This
has minimal implementation cost, but leaves conflict handling, version drift,
and global-versus-workspace choices to every user.

### B. Harness-Specific CLI Adapters

Add an installation command with small target adapters that inspect, render,
merge, validate, report, and remove only their own named server definition.
This is the leading candidate. It provides one user entry point while keeping
the on-disk result native and reviewable.

### C. Universal Semedit Plugin

Create one package format claimed to install across all harnesses. This is
rejected as a product boundary: plugin manifests, lifecycle hooks, package
resolvers, permissions, and update semantics are harness-specific. The common
unit should remain the binary invocation and, later, canonical skill content.

### D. Direct Mutation of Every Known Configuration File

Scan common configuration paths and register `semedit` everywhere. This is
rejected because it violates user intent, crosses trust boundaries, and creates
unreviewable global side effects.

## 6. Required Invariants

1. The installer launches only an explicit, preinstalled `semedit` binary. It
   never downloads, bootstraps, or silently replaces an executable.
2. User and workspace scope are explicit. The default must not write global
   configuration without an affirmative scope selection.
3. A request affects only its selected target. It never installs, enables,
   validates, or requires a plugin, skill, or configuration entry for another
   harness.
4. All configuration writes are atomic and preserve unrelated entries,
   comments where the target format permits them, and file permissions.
5. An existing matching `semedit` registration is idempotent. A conflicting
   registration reports the exact difference and requires an explicit replace
   choice; it is never silently overwritten.
6. `status` reports effective configuration, source path, command, profile,
   skill state, and connection validation without exposing secrets.
7. `uninstall` removes only the entry, skill, or package ledger created by the
   installer; it preserves user-authored configuration and cannot delete a
   shared skill directory it does not own.
8. Native harness permission and enterprise allowlist failures remain explicit
   diagnostics. The installer must not weaken sandbox, trust, or approval
   settings to force registration.
9. The MCP server remains useful without skills. Future skills contain
   steering only and do not carry a second mutation implementation.
10. Installation does not write project language manifests such as `go.work`.

## 7. Initial Delivery Sequence

1. Define a typed, ingress-neutral integration specification and a renderer
   contract before adding Cobra commands.
2. Implement a Copilot adapter for user and workspace registrations, plus
   `dry_run`, status, conflict detection, and targeted removal.
3. Implement the equivalent Codex adapter with the same logical request and
   observability contract.
4. Add deterministic fixture tests for empty, matching, conflicting, malformed,
   and unrelated target configurations; exercise the public CLI through txtar
   where practical.
5. Add generated Getting Started documentation from the adapter metadata.
6. When a canonical skill is introduced, add adapter-owned installation in
   target-native directories and a checksum or ledger sufficient for safe
   removal.
7. Package Codex and any other target-native plugin or extension only after
   direct MCP installation is stable and its package lifecycle is documented.

## 8. Open Questions

1. Should the installer invoke the harness's native `mcp add` command when it
   exists, or should it own format-aware configuration merging for deterministic
   behavior and workspace scope?
2. Which target-specific workspace paths are appropriate for project-owned
   registrations, and should all of them be committed to source control?
3. What source-of-truth and precedence policy applies when a user registration,
   workspace registration, and plugin all define the same server name?
4. Is `mutations-only` the right initial default for minimizing tool context,
   or does reliable onboarding require the full profile?
5. How should a future `--enabled-languages` filter affect already installed
   registrations: explicit update, new named server, or profile selection?
6. Can platform-native plugins reference a separately installed Go binary
   safely, or must a package provide a pinned binary distribution mechanism?
7. Does installation belong in the central operation registry, or is it an
   administrative CLI concern outside semantic operation dispatch?

## 9. Resolution Criteria

Resolve each initial target adapter independently when its installation is:

* explicit about scope and executable path;
* idempotent and non-destructive in configuration fixtures;
* verifiably connected to `semedit mcp` with an expected tool catalog;
* independently removable without affecting unrelated harness state; and
* documented with generated, platform-specific Getting Started guidance.

Create an ADR once the final public command shape, ownership ledger, and
configuration-conflict policy are accepted.
