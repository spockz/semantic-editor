# ADR-0020: Exclusive Golang Runtime, Tooling Ecosystem, and Native Binaries

* **Status**: Accepted
* **Date**: 2026-09-17

---

## Context

`semedit` is designed as a deterministic, zero-token AST code transformation engine and developer tool. Maintaining a unified, lightweight, and hermetic developer environment requires strict discipline over language runtimes, package managers, and external dependencies.

Polyglot toolchains and multi-runtime dependencies (e.g. Python virtual environments, `uv`, `node`, `npm`/`pnpm`) introduce severe operational trade-offs:

1. **Host Friction & Portability**: Users and AI coding harnesses are forced to maintain Python or Node.js runtime environments, global PATH configurations, and package manager state.
2. **Sandbox Permission Gaps**: Script interpreters and dynamic language package managers frequently trigger OS sandbox permission interruptions when traversing files or installing virtual environments.
3. **Distribution & Build Complexity**: Non-native helper scripts complicate single-binary release pipelines, CI checks, and container packaging.

---

## Decision

1. **Exclusive Golang Implementation**:
   * All production code, internal libraries, CLI commands, testing harnesses, benchmark engines, and helper tools in `semedit` must be implemented **exclusively in Go**.
   * Tooling must leverage the Go standard library, official `x/tools` packages, and compiled Go utilities.

2. **Native Binaries for External Tooling**:
   * Any external tools utilized by the build, linting, formatting, or documentation pipelines must be distributed and executed as **standalone native compiled binaries** (e.g. `gofmt`, `golangci-lint`, `hugo`, `vale`).

   * Qualified language-server exceptions use preinstalled, explicitly configured tools. Java 21+ with direct pinned JDT LS supports the trusted Java operations in ADR-0068; Metals and HLS retain their separately bounded contracts. The preinstalled Kotlin, Bash, and Make server exceptions are defined in ADR-0047 and ADR-0048. These external runtimes do not become host implementation languages. Semedit never bootstraps these tools through GHCup, Coursier, Cabal, Stack, or another package manager.
   * Java Maven verification is a narrowly bounded exception: only explicit request-scoped trust may launch a pre-existing wrapper or system Maven binary for the fixed `test-compile` and `test` goals. It uses direct process execution, scratch-local state, and offline mode by default; it is not part of the JDT LS runtime.

3. **Strict Prohibition & Exception Policy for Node.js and Python**:
   * Node.js and Python are **strictly prohibited** by default for any repository code, scripts, runners, or auxiliary tooling.
   * **Last-Resort Exception Criteria**: Node.js or Python may only be considered if all of the following conditions are met:
     1. There is no viable Go-native or standalone native binary alternative.
     2. The tool provides a **material, demonstrable benefit** of being the acknowledged best-in-class standard for the specific problem domain.
     3. **Mandatory User Approval**: The developer must explicitly request and obtain user approval for the specific use case before adding any Node.js or Python code, configuration, or dependency to the repository.

---

## Invariants

* No `package.json`, `pyproject.toml`, virtual environments (`.venv`), or Python/Node scripts may be added to the repository without documented, explicit user approval.
* Java runtime versions and direct Metals/JDT LS paths follow their backend configuration contracts; semedit never downloads, installs, embeds, bootstraps with Coursier, or launches a Python JDT LS launcher such as `jdtls.py`. JDT LS delegation is not restricted to the original lookup slice.
* Continuous Integration (`make check`) must execute with a pure Go toolchain alongside pre-installed native binary executables.
* All evaluation, benchmarking, and AST verification harnesses (e.g. `tools/benchmark-harness`) must compile and execute as native Go programs.

---

## Consequences

* **Positive**: Guarantees a hermetic, single-toolchain repository; eliminates multi-runtime setup friction; ensures fast native execution and painless subagent autonomy inside OS sandboxes.
* **Negative**: Go-native implementations of certain specialized tools (e.g. complex statistical plotting or LLM API billing aggregators) must be authored in Go rather than pulling off-the-shelf Python packages.
