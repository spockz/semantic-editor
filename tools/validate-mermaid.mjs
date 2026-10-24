// WHY: Validate Mermaid fences in Markdown with Mermaid's parser so documentation checks catch syntax errors without launching a browser.
import { readFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { resolve } from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const require = createRequire(new URL("../.scratch/tools/package.json", import.meta.url));
const { JSDOM } = require("jsdom");
const dom = new JSDOM("");
globalThis.window = dom.window;
globalThis.document = dom.window.document;

const { marked } = await import(require.resolve("marked"));
const mermaid = await import(require.resolve("mermaid"));

export async function validateMermaidMarkdown(source, file = "<input>") {
	const tokens = marked.lexer(source);
	const diagrams = [];
	marked.walkTokens(tokens, (token) => {
		if (token.type !== "code") {
			return;
		}

		const language = String(token.lang ?? "").trim().split(/\s+/, 1)[0].toLowerCase();
		if (language === "mermaid") {
			diagrams.push(token.text);
		}
	});

	for (const [index, diagram] of diagrams.entries()) {
		try {
			await mermaid.default.parse(diagram);
		} catch (error) {
			const message = error instanceof Error ? error.message.split("\n", 1)[0] : String(error);
			throw new Error(`${file}: Mermaid diagram ${index + 1} is invalid: ${message}`, { cause: error });
		}
	}

	return diagrams.length;
}

async function main(files) {
	if (files.length === 0) {
		throw new Error("provide one or more Markdown files to validate");
	}

	let diagramCount = 0;
	for (const file of files) {
		diagramCount += await validateMermaidMarkdown(await readFile(file, "utf8"), file);
	}
	console.log(`Validated ${diagramCount} Mermaid diagram(s) in ${files.length} Markdown file(s).`);
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
	main(process.argv.slice(2)).catch((error) => {
		console.error(error instanceof Error ? error.message : error);
		process.exitCode = 1;
	});
}
