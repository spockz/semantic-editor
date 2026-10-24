// WHY: Prove the documentation validator finds Mermaid fences and fails closed on malformed diagram syntax.
import assert from "node:assert/strict";
import test from "node:test";

import { validateMermaidMarkdown } from "./validate-mermaid.mjs";

test("validates Mermaid fences across Markdown fence forms", async () => {
	const source = [
		"```mermaid title=flow",
		"flowchart LR",
		"A --> B",
		"```",
		"",
		"> ~~~mermaid",
		"> sequenceDiagram",
		"> A->>B: request",
		"> ~~~",
		"",
		"```text",
		"not a diagram",
		"```",
	].join("\n");

	assert.equal(await validateMermaidMarkdown(source, "valid.md"), 2);
});

test("rejects invalid Mermaid syntax with its source and diagram number", async () => {
	await assert.rejects(
		validateMermaidMarkdown("```mermaid\nflowchart LR\nA -->\n```", "broken.md"),
		/broken\.md: Mermaid diagram 1 is invalid: Parse error/,
	);
});

test("accepts Markdown with no Mermaid diagrams", async () => {
	assert.equal(await validateMermaidMarkdown("# Notes\n\n```go\npackage main\n```\n", "notes.md"), 0);
});
