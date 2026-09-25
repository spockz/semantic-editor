# Todo

Thoughts to be further worked out:

- Have a mcp tool for automatic fixing. Could dispatch to lsps, markdownlint if configured (apparently that cli also has automatic fixes). This could then also be added as a flag to all tooling.
- Have a cli flag (also for the mcp server) `--enabled-languages` that causes the MCP server to only expose capabilities that are available for any of the enabled languages and have descriptions specified to the enabled languages to increase the chances the tool is selected automatically.
- The getting started instructions should also contain instructions on how to install the skills and plugins into Antigravity(-cli), codex, opencode, github copilot, and pi.
- The cli should probably have a command to install itself as a plugin with the aforementioned coding harnesses. The instructions can
  then just point to running the appropriate command. The plugin should allow for global installation or workspace installation.
- Align the ADRs that define how semantic tools are structured and exposed through the CLI and MCP interfaces, including at least ADR-0046, ADR-0011, ADR-0016, and ADR-0019. Reconcile their guidance with the public `insert_structure` contract.
