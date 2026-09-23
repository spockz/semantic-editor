// hugo.go contains the Hugo source renderers and embedded landing assets.
package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	githubRepositoryURL    = "https://github.com/spockz/semantic-editor"
	githubSourceBaseURL    = githubRepositoryURL + "/blob/main"
	githubRawSourceBaseURL = "https://raw.githubusercontent.com/spockz/semantic-editor/main"
	hextraModuleVersion    = "v0.12.3"
)

var landingAssetNames = []string{
	"semantic-workflow-banner.png",
	"deterministic-edits.png",
	"symbol-intent.png",
	"structured-feedback.png",
	"agent-contract.png",
}

// landingAssets keeps the marketing imagery with the generator so a documentation build needs no runtime asset fetches.
//
//go:embed assets/landing/*.png
var landingAssets embed.FS

func renderMarkdown(caps []CodeCapability, placements []string, examples []TxtarExample) string {
	var buf bytes.Buffer

	buf.WriteString(`---
title: "Automated Capability Documentation"
description: "Deterministic, zero-token refactoring capabilities and executable examples."
draft: false
toc: true
weight: 10
---

Deterministic, zero-token refactoring capabilities extracted directly from compiler AST implementations and executable test archives (` + "`txtar`" + `).

[View the semedit repository on GitHub](` + githubRepositoryURL + `)

## Cross-Language Edit & Refactoring Capability Matrix

| Language | Maturity | Supported Access Modifiers | Supported Edit / Refactoring Capabilities |
| :--- | :--- | :--- | :--- |` + "\n")
	for _, c := range caps {
		modifiers := make([]string, 0, len(c.SupportedModifiers))
		for _, modifier := range c.SupportedModifiers {
			modifiers = append(modifiers, "`"+modifier+"`")
		}
		var operations []string
		for operation, metadata := range c.Operations {
			if metadata.Supported {
				operations = append(operations, "`"+operation+"`")
			}
		}
		slices.Sort(operations)
		fmt.Fprintf(&buf, "| %s | %s | %s | %s |\n",
			markdownCell(c.DisplayName),
			markdownCell(c.Maturity),
			markdownCell(strings.Join(modifiers, ", ")),
			markdownCell(strings.Join(operations, ", ")),
		)
	}

	buf.WriteString("\n## Language Constraints & Capability Rules\n\n")
	for _, c := range caps {
		fmt.Fprintf(&buf, "### %s Engine Constraints\n\n", markdownCell(c.DisplayName))
		for _, rule := range c.Limitations {
			fmt.Fprintf(&buf, "- **%s**: %s\n", markdownCell(rule.Title), markdownCell(rule.Description))
		}
		buf.WriteString("\n")
	}

	buf.WriteString("## Placement Qualifiers\n\n")
	buf.WriteString("| Placement Mode | Target Identifier Required | Semantic Behavior |\n| :--- | :--- | :--- |\n")
	placementDescriptions := map[string]struct {
		targetRequired bool
		description    string
	}{
		"file_start":    {false, "Prepends declaration immediately after the package/import preamble."},
		"file_end":      {false, "Appends declaration at the conclusion of the file (default fallback)."},
		"public_start":  {false, "Anchors declaration at the start of the public declarations section."},
		"public_end":    {false, "Appends declaration at the boundary concluding public declarations."},
		"private_start": {false, "Anchors declaration at the start of the unexported/private section."},
		"private_end":   {false, "Appends declaration at the boundary concluding private declarations."},
		"before_symbol": {true, "Locates target symbol AST node and injects declaration directly preceding it."},
		"after_symbol":  {true, "Locates target symbol AST node and injects declaration directly succeeding it."},
	}
	for _, placement := range placements {
		info := placementDescriptions[placement]
		required := "No"
		if info.targetRequired {
			required = "Yes"
		}
		fmt.Fprintf(&buf, "| `%s` | %s | %s |\n", placement, required, markdownCell(info.description))
	}

	buf.WriteString("\n## Executable Test Workflows\n\n")
	buf.WriteString("Every scenario below is parsed from active, compiler-verified regression tests in `testdata/scripts/*.txtar`.\n\n")
	for _, ex := range examples {
		fmt.Fprintf(&buf, "### %s\n\n", markdownCell(ex.Title))
		sourcePath := "/testdata/scripts/" + ex.Filename
		fmt.Fprintf(&buf, "Source: [%s](%s) · [Download raw file](%s)\n\n", ex.Filename, githubSourceBaseURL+sourcePath, githubRawSourceBaseURL+sourcePath)
		if ex.Description != "" {
			fmt.Fprintf(&buf, "%s\n\n", markdownCell(ex.Description))
		}
		for _, step := range ex.Steps {
			fmt.Fprintf(&buf, "#### %s\n\n", markdownCell(step.Description))
			buf.WriteString("**CLI invocation**\n\n")
			writeMarkdownCodeBlock(&buf, "console", "$ "+step.Command)
			fmt.Fprintf(&buf, "**Equivalent MCP tool call (`%s`)**\n\n", step.MCPTool)
			writeMarkdownCodeBlock(&buf, "json", step.MCPArgsJSON)
			if step.ExpectedOut != "" {
				for assertLine := range strings.SplitSeq(step.ExpectedOut, "\n") {
					if assertLine != "" {
						fmt.Fprintf(&buf, "> Assert: %s\n\n", markdownCell(assertLine))
					}
				}
			}
			if step.ExpectedErr != "" {
				for errLine := range strings.SplitSeq(step.ExpectedErr, "\n") {
					if errLine != "" {
						fmt.Fprintf(&buf, "> Expected error: %s\n\n", markdownCell(errLine))
					}
				}
			}
		}

		outputs := ex.Outputs
		if len(outputs) == 0 && (ex.OutputCode != "" || len(ex.DiffLines) > 0) {
			outputs = []TxtarFileOutput{{
				Path:      ex.OutputFile,
				Content:   ex.OutputCode,
				DiffLines: ex.DiffLines,
			}}
		}

		for _, out := range outputs {
			if len(out.DiffLines) > 0 {
				if len(outputs) == 1 {
					buf.WriteString("**Unified AST transformation diff**\n\n")
				} else {
					fmt.Fprintf(&buf, "**Unified AST transformation diff (`%s`)**\n\n", out.Path)
				}
				var diff strings.Builder
				for _, line := range out.DiffLines {
					diff.WriteString(line.Content)
					diff.WriteByte('\n')
				}
				writeMarkdownCodeBlock(&buf, "diff", strings.TrimSuffix(diff.String(), "\n"))
			}

			if out.Content != "" {
				if len(outputs) == 1 {
					if out.Path != "" {
						fmt.Fprintf(&buf, "**Expected output state (`%s`)**\n\n", out.Path)
					} else {
						buf.WriteString("**Expected output state**\n\n")
					}
				} else {
					fmt.Fprintf(&buf, "**Expected output state (`%s`)**\n\n", out.Path)
				}
				lang := detectCodeBlockLanguage(out.Path)
				writeMarkdownCodeBlock(&buf, lang, out.Content)
			}
		}
	}

	buf.WriteString("## CI Drift Invariant\n\nDocumentation is regenerated from compiler capabilities and regression test archives during continuous integration before publication.\n")
	return buf.String()
}

func detectCodeBlockLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".scala":
		return "scala"
	case ".hs":
		return "haskell"
	case ".toml":
		return "toml"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".sh":
		return "bash"
	default:
		base := strings.ToLower(filepath.Base(filePath))
		switch base {
		case "go.mod", "go.sum":
			return "go"
		case "cargo.toml":
			return "toml"
		}
		if ext != "" {
			return strings.TrimPrefix(ext, ".")
		}
		return "text"
	}
}

func writeMarkdownCodeBlock(buf *bytes.Buffer, language, content string) {
	fence := "```"
	if strings.Contains(content, fence) {
		fence = "````"
	}
	fmt.Fprintf(buf, "%s%s\n%s\n%s\n\n", fence, language, content, fence)
}

func markdownCell(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "|", "\\|"), "\n", " ")
}

func renderGettingStarted() string {
	return `---
title: "Getting Started"
description: "Install semedit on macOS or Linux and run your first semantic edit."
draft: false
weight: 1
---

semedit turns an editing intent into a compiler-backed change. The CLI and MCP server use the same deterministic engine.

## macOS with Homebrew

Install Go and Hugo Extended with [Homebrew](https://brew.sh/), then install the Go language server and semedit:

~~~console
brew install go hugo
go install golang.org/x/tools/gopls@latest
go install github.com/spockz/semantic-editor@latest
~~~

Ensure the Go binary directory is on your PATH, then verify the installation:

~~~console
semedit --help
~~~

## Linux

Install Go and Hugo Extended using your Linux distribution's package manager. Then install gopls and semedit with Go:

~~~console
sudo apt install golang-go hugo
go install golang.org/x/tools/gopls@latest
go install github.com/spockz/semantic-editor@latest
~~~

Verify the installation:

~~~console
semedit --help
~~~

## Run a semantic edit

From a Go module, resolve a symbol without counting lines:

~~~console
semedit lookup --file api/server.go --symbol Server.Start
~~~

Rename the resolved symbol across the workspace:

~~~console
semedit rename --file api/server.go --symbol Server.Start --to Serve
~~~

## Connect an MCP client

Start the stdio MCP server from the project workspace:

~~~console
semedit mcp
~~~

Configure your MCP client to launch the same command. Keep the executable path absolute when the client does not inherit your shell PATH.

The generated [capability reference](../reference/) contains the available semantic tools and executable examples.
`
}

func writeHugoConfig(outputDir string) error {
	config := `baseURL = "/"
languageCode = "en-us"
title = "semedit"
contentDir = "content"
enableEmoji = true
enableRobotsTXT = true

[module]
  [[module.imports]]
    path = "github.com/imfing/hextra"

[markup]
  [markup.tableOfContents]
    endLevel = 4
    startLevel = 1
  [markup.goldmark]
    [markup.goldmark.renderer]
      unsafe = true

[params]
  description = "Intent-driven code editing for AI agents."

[params.navbar]
  displayTitle = true
  displayLogo = false

[params.theme]
  default = "system"
  displayToggle = true

[params.search]
  enable = true
  type = "flexsearch"

[params.editURL]
  enable = true
  base = "https://github.com/spockz/semantic-editor/edit/main"

[params.page]
  displayPagination = true

[menu]
  [[menu.main]]
    name = "Get started"
    pageRef = "/docs/getting-started"
    weight = 1
  [[menu.main]]
    name = "Reference"
    pageRef = "/docs/reference"
    weight = 2
  [[menu.main]]
    name = "Benchmarks"
    pageRef = "/docs/benchmarks"
    weight = 3
  [[menu.main]]
    name = "Search"
    weight = 4
    [menu.main.params]
      type = "search"
  [[menu.main]]
    name = "GitHub"
    url = "https://github.com/spockz/semantic-editor"
    weight = 5
    [menu.main.params]
      icon = "github"
  [[menu.main]]
    name = "Theme Toggle"
    weight = 6
    [menu.main.params]
      type = "theme-toggle"
      label = true
`
	if err := writeGeneratedFile(filepath.Join(outputDir, "hugo.toml"), []byte(config)); err != nil {
		return fmt.Errorf("write hugo.toml: %w", err)
	}
	module := fmt.Sprintf("module semedit-docs\n\ngo 1.23\n\nrequire github.com/imfing/hextra %s\n", hextraModuleVersion)
	if err := writeGeneratedFile(filepath.Join(outputDir, "go.mod"), []byte(module)); err != nil {
		return fmt.Errorf("write Hugo module go.mod: %w", err)
	}
	landing := `---
title: "semedit"
description: "Intent-driven code editing for AI agents."
draft: false
---

<div class="hx:mt-16 hx:mb-16 hx:text-center">

{{< hextra/hero-badge link="/docs/" >}}
Compiler-backed semantic editing <span aria-hidden="true">→</span>
{{< /hextra/hero-badge >}}

# Intent-driven code editing<br/>for **AI agents**

<p class="hx:mt-6 hx:text-xl hx:text-gray-600 hx:dark:text-gray-400">
LLMs plan the change. Compilers and language servers apply it precisely.
</p>

<figure class="hx:mt-10 hx:mb-10 hx:overflow-hidden hx:rounded-2xl hx:border hx:border-gray-200 hx:shadow-xl hx:dark:border-neutral-800">
  <img src="/images/landing/semantic-workflow-banner.png" alt="An abstract code editor flowing into a precise compiler syntax tree" style="display: block; width: 100%; aspect-ratio: 3 / 1; object-fit: cover;" />
</figure>

<div class="hx:mt-8 hx:flex hx:flex-wrap hx:justify-center hx:gap-3">
{{< hextra/hero-button text="Get started" link="/docs/getting-started/" >}}
{{< hextra/hero-button text="View on GitHub" link="https://github.com/spockz/semantic-editor" style="background-color: transparent; color: inherit; border: 1px solid currentColor;" >}}
</div>

<p class="hx:mt-6 hx:text-sm hx:text-gray-500 hx:dark:text-gray-400">Open source and MIT licensed.</p>
</div>

## Make intent the interface

semedit separates semantic intent from syntax transformation, so agents can ask for the change while local tooling handles the mechanical work.

{{< hextra/feature-grid cols="2" >}}
<a class="hx:block hx:overflow-hidden hx:rounded-xl hx:border hx:border-gray-200 hx:bg-gray-50 hx:transition hover:hx:border-primary-300 hover:hx:shadow-lg hx:dark:border-neutral-800 hx:dark:bg-neutral-900" href="/docs/reference/">
  <img src="/images/landing/deterministic-edits.png" alt="A compiler shield protecting a structured code module" style="display: block; width: 100%; aspect-ratio: 16 / 9; object-fit: cover;" loading="lazy" />
  <span class="hx:block hx:p-5"><strong class="hx:block hx:text-lg">Deterministic edits</strong><span class="hx:mt-2 hx:block hx:text-gray-600 hx:dark:text-gray-400">Use compiler-backed transformations that preserve syntax and eliminate fragile line-based patching.</span></span>
</a>
<a class="hx:block hx:overflow-hidden hx:rounded-xl hx:border hx:border-gray-200 hx:bg-gray-50 hx:transition hover:hx:border-primary-300 hover:hx:shadow-lg hx:dark:border-neutral-800 hx:dark:bg-neutral-900" href="/docs/getting-started/">
  <img src="/images/landing/symbol-intent.png" alt="A target resolved within a connected graph of symbols" style="display: block; width: 100%; aspect-ratio: 16 / 9; object-fit: cover;" loading="lazy" />
  <span class="hx:block hx:p-5"><strong class="hx:block hx:text-lg">Symbol-based intent</strong><span class="hx:mt-2 hx:block hx:text-gray-600 hx:dark:text-gray-400">Ask for <strong>Server.Start</strong> instead of hunting for a byte offset or line number.</span></span>
</a>
<a class="hx:block hx:overflow-hidden hx:rounded-xl hx:border hx:border-gray-200 hx:bg-gray-50 hx:transition hover:hx:border-primary-300 hover:hx:shadow-lg hx:dark:border-neutral-800 hx:dark:bg-neutral-900" href="/docs/reference/">
  <img src="/images/landing/structured-feedback.png" alt="Diagnostics resolving into a clear evidence graph" style="display: block; width: 100%; aspect-ratio: 16 / 9; object-fit: cover;" loading="lazy" />
  <span class="hx:block hx:p-5"><strong class="hx:block hx:text-lg">Structured feedback</strong><span class="hx:mt-2 hx:block hx:text-gray-600 hx:dark:text-gray-400">Receive formatting, diagnostics, and compiler evidence as structured results.</span></span>
</a>
<a class="hx:block hx:overflow-hidden hx:rounded-xl hx:border hx:border-gray-200 hx:bg-gray-50 hx:transition hover:hx:border-primary-300 hover:hx:shadow-lg hx:dark:border-neutral-800 hx:dark:bg-neutral-900" href="/docs/reference/">
  <img src="/images/landing/agent-contract.png" alt="Connected modules sharing one central contract" style="display: block; width: 100%; aspect-ratio: 16 / 9; object-fit: cover;" loading="lazy" />
  <span class="hx:block hx:p-5"><strong class="hx:block hx:text-lg">One contract for agents</strong><span class="hx:mt-2 hx:block hx:text-gray-600 hx:dark:text-gray-400">Use the same semantic operations through the CLI or MCP.</span></span>
</a>
{{< /hextra/feature-grid >}}

## A tighter editing loop

1. An LLM plans a high-level intent.
2. semedit resolves the target symbol and selects the compiler or language server.
3. The host applies and formats the change deterministically.
4. Diagnostics return as evidence for the next decision.

## Explore the documentation

[Read the capability reference](/docs/reference/) to see the available operations and executable examples, or [view empirical benchmarks](/docs/benchmarks/) for measured results.
`
	if err := writeGeneratedFile(filepath.Join(outputDir, "content", "_index.md"), []byte(landing)); err != nil {
		return fmt.Errorf("write Hugo landing page: %w", err)
	}
	docsSection := `---
title: "Documentation"
description: "semedit installation and compiler-backed capability reference."
draft: false
weight: 1
---

Start with the installation guide, then use the reference when you need a specific semantic operation. The benchmark report documents the measured results behind the workflow.

{{< hextra/feature-grid cols="3" >}}
{{< hextra/feature-card title="Get started" icon="terminal" link="/docs/getting-started/" subtitle="Install semedit and run your first compiler-backed edit." >}}
{{< hextra/feature-card title="Reference" icon="shield-check" link="/docs/reference/" subtitle="Browse the generated capability and CLI reference." >}}
{{< hextra/feature-card title="Benchmarks" icon="chart-bar" link="/docs/benchmarks/" subtitle="Review empirical latency, token, and correctness results." >}}
{{< /hextra/feature-grid >}}
`
	if err := writeGeneratedFile(filepath.Join(outputDir, "content", "docs", "_index.md"), []byte(docsSection)); err != nil {
		return fmt.Errorf("write Hugo docs section: %w", err)
	}
	return nil
}

func writeLandingAssets(outputDir string) error {
	assetsDir := filepath.Join(outputDir, "static", "images", "landing")
	if err := os.RemoveAll(assetsDir); err != nil {
		return fmt.Errorf("remove stale landing assets: %w", err)
	}
	if err := os.MkdirAll(assetsDir, 0o750); err != nil {
		return fmt.Errorf("create landing assets directory: %w", err)
	}
	for _, name := range landingAssetNames {
		asset, err := landingAssets.ReadFile(filepath.ToSlash(filepath.Join("assets", "landing", name)))
		if err != nil {
			return fmt.Errorf("read embedded landing asset %q: %w", name, err)
		}
		if err := writeGeneratedFile(filepath.Join(assetsDir, name), asset); err != nil {
			return fmt.Errorf("write landing asset %q: %w", name, err)
		}
	}
	return nil
}

func writeGeneratedFile(targetPath string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".docgen-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set temporary output mode: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary output: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary output: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("rename temporary output: %w", err)
	}
	return nil
}
