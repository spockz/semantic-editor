// Package main synthesizes code-derived capability documentation and test-driven examples into a Hugo source tree.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"semedit/internal/backend"
	"semedit/internal/operation"
)

const (
	githubRepositoryURL    = "https://github.com/spockz/semantic-editor"
	githubSourceBaseURL    = githubRepositoryURL + "/blob/main"
	githubRawSourceBaseURL = "https://raw.githubusercontent.com/spockz/semantic-editor/main"
	lotusDocsModuleVersion = "v0.3.0"
	bootstrapModuleVersion = "v5.20300.20800"
)

// CodeCapability represents extracted language capability metadata.
type CodeCapability struct {
	Language           string
	DisplayName        string
	Maturity           string
	SupportedModifiers []string
	Operations         map[string]OpMetadata
	Limitations        []Constraint
}

// OpMetadata describes an operation.
type OpMetadata struct {
	Supported    bool
	Description  string
	CLICommand   string
	MCPTool      string
	PlacementKey bool
}

// Constraint describes a language rule.
type Constraint struct {
	Title       string
	Description string
	Severity    string // "error", "warning", "info"
}

// TxtarStep captures a single execution step from a txtar test script.
type TxtarStep struct {
	Number      int
	Description string
	Command     string
	IsNegated   bool
	MCPTool     string
	MCPArgsJSON string
	ExpectedOut string
	ExpectedErr string
}

// TxtarFileOutput captures the expected post-transformation state of a file in a txtar scenario.
type TxtarFileOutput struct {
	Path      string
	Content   string
	DiffLines []DiffLine
}

// TxtarExample captures parsed executable scenario from a .txtar file.
type TxtarExample struct {
	Filename    string
	Title       string
	Description string
	Steps       []TxtarStep
	InputFile   string
	InputCode   string
	OutputFile  string
	OutputCode  string
	DiffLines   []DiffLine
	Outputs     []TxtarFileOutput
}

// DiffLine represents a line in a unified diff.
type DiffLine struct {
	Type    string // "add", "del", "same"
	Content string
}

func main() {
	rootDir, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error locating repo root: %v\n", err)
		os.Exit(1)
	}

	outputDir := filepath.Join(rootDir, ".scratch", "docgen")
	flags := flag.NewFlagSet("docgen", flag.ExitOnError)
	flags.StringVar(&outputDir, "output-dir", outputDir, "directory for the generated Hugo source tree")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "content", "docs"), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating documentation source directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "data"), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Hugo data directory: %v\n", err)
		os.Exit(1)
	}
	for _, stalePath := range []string{
		filepath.Join(outputDir, "content", "docs", "index.md"),
		filepath.Join(outputDir, "content", "docs", "getting-started"),
		filepath.Join(outputDir, "content", "docs", "reference"),
	} {
		if err := os.RemoveAll(stalePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing stale documentation output: %v\n", err)
			os.Exit(1)
		}
	}

	// 1. Extract capabilities from code AST
	capabilities, placements, err := extractCodeCapabilities(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting code capabilities: %v\n", err)
		os.Exit(1)
	}

	// 2. Parse txtar test archives for real-world examples
	examples, err := extractTxtarExamples(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting txtar examples: %v\n", err)
		os.Exit(1)
	}

	// 3. Render Hugo source content
	markdownContent := renderMarkdown(capabilities, placements, examples)
	targetFile := filepath.Join(outputDir, "content", "docs", "reference.md")
	if err := writeGeneratedFile(targetFile, []byte(markdownContent)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Markdown output: %v\n", err)
		os.Exit(1)
	}
	gettingStartedFile := filepath.Join(outputDir, "content", "docs", "getting-started.md")
	if err := writeGeneratedFile(gettingStartedFile, []byte(renderGettingStarted())); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing getting started output: %v\n", err)
		os.Exit(1)
	}
	benchmarksContent, err := renderBenchmarksDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering benchmarks output: %v\n", err)
		os.Exit(1)
	}
	benchmarksFile := filepath.Join(outputDir, "content", "docs", "benchmarks.md")
	if err := writeGeneratedFile(benchmarksFile, []byte(benchmarksContent)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing benchmarks output: %v\n", err)
		os.Exit(1)
	}
	if err := writeHugoConfig(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Hugo configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully synthesized Hugo documentation source: %s\n", targetFile)
	fmt.Printf("Extracted %d languages, %d placement modes, %d txtar workflows.\n", len(capabilities), len(placements), len(examples))
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return os.Getwd()
}

func extractCodeCapabilities(rootDir string) ([]CodeCapability, []string, error) {
	// Parse internal/astedit/insert.go for placement qualifier constants.
	// These are used by the docs template and are not exposed through the backend interface.

	insertFile := filepath.Join(rootDir, "internal", "astedit", "insert.go")
	fset := token.NewFileSet()
	insertNode, err := parser.ParseFile(fset, insertFile, nil, 0)

	var placements []string
	if err == nil {
		ast.Inspect(insertNode, func(n ast.Node) bool {
			gen, ok := n.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				return true
			}
			for _, spec := range gen.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, val := range valSpec.Values {
					if basic, ok := val.(*ast.BasicLit); ok && basic.Kind == token.STRING {
						valStr := strings.Trim(basic.Value, `"`)
						if strings.Contains(valStr, "_") || valStr == "file_start" || valStr == "file_end" {
							placements = append(placements, valStr)
						}
					}
				}
			}
			return true
		})
	}

	if len(placements) == 0 {
		placements = []string{
			"file_start", "file_end",
			"public_start", "public_end",
			"private_start", "private_end",
			"before_symbol", "after_symbol",
		}
	}

	// The operation registry is the documentation source of truth. Language
	// profiles provide the static limitations that accompany its handlers.
	registry := operation.DefaultRegistry()
	langOrder := []backend.LanguageID{
		backend.LanguageGo,
		backend.LanguageRust,
		backend.LanguageJava,
		backend.LanguageScala,
		backend.LanguageHaskell,
	}
	var capabilities []CodeCapability
	for _, lang := range langOrder {
		m, err := registry.Matrix(lang)
		if err != nil {
			return nil, nil, fmt.Errorf("operation capability matrix for %s: %w", lang, err)
		}
		ops := make(map[string]OpMetadata, len(m.Operations))
		for name, op := range m.Operations {
			ops[name] = OpMetadata{
				Supported:    op.Supported,
				Description:  op.Description,
				CLICommand:   op.CLICommand,
				MCPTool:      op.MCPTool,
				PlacementKey: op.PlacementKey,
			}
		}
		constraints := make([]Constraint, len(m.Limitations))
		for i, lim := range m.Limitations {
			constraints[i] = Constraint{
				Title:       lim.Title,
				Description: lim.Description,
				Severity:    lim.Severity,
			}
		}
		capabilities = append(capabilities, CodeCapability{
			Language:           m.Language,
			DisplayName:        m.DisplayName,
			Maturity:           m.Maturity,
			SupportedModifiers: m.SupportedModifiers,
			Operations:         ops,
			Limitations:        constraints,
		})
	}

	return capabilities, placements, nil
}

func extractTxtarExamples(rootDir string) ([]TxtarExample, error) {
	scriptsDir := filepath.Join(rootDir, "testdata", "scripts")
	files, err := os.ReadDir(scriptsDir)
	if err != nil {
		return nil, fmt.Errorf("read scripts dir: %w", err)
	}

	var examples []TxtarExample

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".txtar") {
			continue
		}

		cleanFilePath := filepath.Clean(filepath.Join(scriptsDir, file.Name()))
		// #nosec G304 -- reading verified test archive for example synthesis
		content, err := os.ReadFile(cleanFilePath)
		if err != nil {
			continue
		}

		ex := parseTxtarFile(file.Name(), string(content))
		if ex != nil {
			examples = append(examples, *ex)
		}
	}

	// Sort examples with specialized and declaration first
	sort.Slice(examples, func(i, j int) bool {
		return examples[i].Filename < examples[j].Filename
	})

	return examples, nil
}

func parseTxtarFile(filename string, content string) *TxtarExample {
	parts := strings.Split(content, "\n-- ")
	if len(parts) == 0 {
		return nil
	}

	header := parts[0]
	fileMap := make(map[string]string)

	for i := 1; i < len(parts); i++ {
		sec := parts[i]
		idx := strings.Index(sec, " --\n")
		if idx == -1 {
			idx = strings.Index(sec, " --\r\n")
		}
		if idx == -1 {
			continue
		}
		subPath := strings.TrimSpace(sec[:idx])
		fileContent := sec[idx+4:]
		fileMap[subPath] = strings.TrimSpace(fileContent)
	}

	// Parse header commands
	lines := strings.Split(header, "\n")
	var title string
	var descriptionLines []string
	var steps []TxtarStep
	cmpMap := make(map[string]string)

	currentStepDesc := ""
	stepNum := 1

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		if after, ok := strings.CutPrefix(line, "#"); ok {
			comment := strings.TrimSpace(after)
			switch {
			case title == "":
				title = comment
			case strings.HasPrefix(comment, "1.") || strings.HasPrefix(comment, "2.") ||
				strings.HasPrefix(comment, "3.") || strings.HasPrefix(comment, "4.") ||
				strings.HasPrefix(comment, "5.") || strings.HasPrefix(comment, "6.") ||
				strings.HasPrefix(comment, "7."):
				currentStepDesc = comment
			default:
				descriptionLines = append(descriptionLines, comment)
			}
			continue
		}

		if strings.HasPrefix(line, "exec semedit") || strings.HasPrefix(line, "! exec semedit") {
			isNeg := strings.HasPrefix(line, "!")
			cmdStr := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "!"), "exec"))
			cmdStr = strings.TrimSpace(cmdStr)

			toolName, toolArgs := mapCLIToMCP(cmdStr)
			argsJSON, _ := json.MarshalIndent(toolArgs, "", "  ")

			desc := currentStepDesc
			if desc == "" {
				desc = fmt.Sprintf("Execute: %s", cmdStr)
			}

			steps = append(steps, TxtarStep{
				Number:      stepNum,
				Description: desc,
				Command:     cmdStr,
				IsNegated:   isNeg,
				MCPTool:     toolName,
				MCPArgsJSON: string(argsJSON),
			})
			stepNum++
			currentStepDesc = ""
			continue
		}

		if strings.HasPrefix(line, "cmp ") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				cmpMap[fields[2]] = fields[1]
			}
			continue
		}

		if strings.HasPrefix(line, "stderr ") && len(steps) > 0 {
			errPattern := strings.Trim(strings.TrimPrefix(line, "stderr "), `"'`)
			if steps[len(steps)-1].ExpectedErr == "" {
				steps[len(steps)-1].ExpectedErr = errPattern
			} else {
				steps[len(steps)-1].ExpectedErr += "\n" + errPattern
			}
		}
		if strings.HasPrefix(line, "stdout ") && len(steps) > 0 {
			outPattern := strings.Trim(strings.TrimPrefix(line, "stdout "), `"'`)
			if steps[len(steps)-1].ExpectedOut == "" {
				steps[len(steps)-1].ExpectedOut = outPattern
			} else {
				steps[len(steps)-1].ExpectedOut += "\n" + outPattern
			}
		}
	}

	if title == "" {
		title = strings.TrimSuffix(filename, ".txtar")
	}

	// Identify all expected output (want) files and compute their transformation diffs
	var wantKeys []string
	for k := range fileMap {
		if strings.HasPrefix(k, "want/") || strings.HasPrefix(k, "want.") {
			wantKeys = append(wantKeys, k)
		}
	}
	sort.Strings(wantKeys)

	var outputs []TxtarFileOutput
	for _, wantKey := range wantKeys {
		wantContent := fileMap[wantKey]
		var targetPath string
		var inputContent string

		if actual, ok := cmpMap[wantKey]; ok {
			targetPath = actual
			inputContent = fileMap[actual]
		} else if after, ok := strings.CutPrefix(wantKey, "want/"); ok {
			targetPath = after
			inputContent = fileMap[after]
		} else if ext := filepath.Ext(wantKey); ext != "" {
			for f := range fileMap {
				if strings.HasSuffix(f, ext) && !strings.HasPrefix(f, "want") {
					targetPath = f
					inputContent = fileMap[f]
					break
				}
			}
			if targetPath == "" {
				targetPath = wantKey
			}
		} else {
			targetPath = wantKey
		}

		diff := generateDiff(inputContent, wantContent)
		outputs = append(outputs, TxtarFileOutput{
			Path:      targetPath,
			Content:   wantContent,
			DiffLines: diff,
		})
	}

	var firstInput, firstInputCode, firstOutput, firstOutputCode string
	var firstDiff []DiffLine
	if len(outputs) > 0 {
		firstOutput = outputs[0].Path
		firstOutputCode = outputs[0].Content
		firstDiff = outputs[0].DiffLines
		firstInput = outputs[0].Path
		firstInputCode = fileMap[outputs[0].Path]
	} else {
		for k, v := range fileMap {
			if strings.HasSuffix(k, ".go") && !strings.HasPrefix(k, "want") {
				firstInput = k
				firstInputCode = v
				break
			}
		}
	}

	return &TxtarExample{
		Filename:    filename,
		Title:       title,
		Description: strings.Join(descriptionLines, " "),
		Steps:       steps,
		InputFile:   firstInput,
		InputCode:   firstInputCode,
		OutputFile:  firstOutput,
		OutputCode:  firstOutputCode,
		DiffLines:   firstDiff,
		Outputs:     outputs,
	}
}

func mapCLIToMCP(cmd string) (string, map[string]any) {
	tokens := parseCommandLine(cmd)
	if len(tokens) < 2 {
		return "semedit", map[string]any{}
	}
	entry, ok := operation.DefaultRegistry().LookupCLI(tokens[1])
	if !ok {
		return "semedit_" + tokens[1], map[string]any{}
	}
	flags := make(map[string][]string)
	for i := 2; i < len(tokens); i++ {
		name, isFlag := strings.CutPrefix(tokens[i], "--")
		if !isFlag {
			continue
		}
		value := "true"
		if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "--") {
			value = tokens[i+1]
			i++
		}
		flags[name] = append(flags[name], value)
	}
	args := make(map[string]any)
	for _, param := range entry.Params {
		values := flags[param.CLIName]
		if len(values) == 0 {
			continue
		}
		switch param.Type {
		case operation.ParamBoolean:
			args[param.JSONName] = values[len(values)-1] == "true"
		case operation.ParamStringSlice:
			args[param.JSONName] = values
		default:
			args[param.JSONName] = values[len(values)-1]
		}
	}
	return entry.MCPName, args
}

func parseCommandLine(cmd string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case (c == ' ' || c == '\t') && !inSingle && !inDouble:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func generateDiff(before, after string) []DiffLine {
	if before == "" && after == "" {
		return nil
	}
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")

	var diff []DiffLine
	bSet := make(map[string]bool)
	for _, l := range beforeLines {
		bSet[strings.TrimSpace(l)] = true
	}

	aSet := make(map[string]bool)
	for _, l := range afterLines {
		aSet[strings.TrimSpace(l)] = true
	}

	for _, l := range beforeLines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !aSet[trimmed] {
			diff = append(diff, DiffLine{Type: "del", Content: "- " + l})
		}
	}
	for _, l := range afterLines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !bSet[trimmed] {
			diff = append(diff, DiffLine{Type: "add", Content: "+ " + l})
		} else {
			diff = append(diff, DiffLine{Type: "same", Content: "  " + l})
		}
	}
	return diff
}

func renderMarkdown(caps []CodeCapability, placements []string, examples []TxtarExample) string {
	var buf bytes.Buffer

	buf.WriteString(`---
title: "Automated Capability Documentation"
description: "Deterministic, zero-token refactoring capabilities and executable examples."
icon: "code"
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
		sort.Strings(operations)
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
icon: "rocket_launch"
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

[module]
  [[module.imports]]
    path = "github.com/colinwilson/lotusdocs"
    disable = false
  [[module.imports]]
    path = "github.com/gohugoio/hugo-mod-bootstrap-scss/v5"
    disable = false

[markup]
  [markup.tableOfContents]
    endLevel = 3
    startLevel = 1
  [markup.goldmark]
    [markup.goldmark.renderer]
      unsafe = true

[params]
  google_fonts = [["Inter", "300, 400, 600, 700"], ["Fira Code", "400, 500, 600, 700"]]
  sans_serif_font = "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
  secondary_font = "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
  mono_font = "'Fira Code', SFMono-Regular, Menlo, Monaco, Consolas, monospace"

[params.docs]
  title = "semedit"
  themeColor = "blue"
  darkMode = true
  prism = true
  prismTheme = "lotusdocs"
  repoURL = "https://github.com/spockz/semantic-editor"
  repoBranch = "main"
  breadcrumbs = true
  toc = true
  tocMobile = true
  scrollSpy = true
  backToTop = true
  extLinkNewTab = true

[params.social]
  github = "spockz"

[menu]
  [[menu.primary]]
    name = "Getting Started"
    url = "/docs/getting-started/"
    identifier = "getting-started"
    weight = 1
  [[menu.primary]]
    name = "Documentation"
    url = "/docs/"
    identifier = "docs"
    weight = 10
  [[menu.primary]]
    name = "Benchmarks"
    url = "/docs/benchmarks/"
    identifier = "benchmarks"
    weight = 20
`
	if err := writeGeneratedFile(filepath.Join(outputDir, "hugo.toml"), []byte(config)); err != nil {
		return fmt.Errorf("write hugo.toml: %w", err)
	}
	module := fmt.Sprintf("module semedit-docs\n\ngo 1.23\n\nrequire (\n\tgithub.com/colinwilson/lotusdocs %s\n\tgithub.com/gohugoio/hugo-mod-bootstrap-scss/v5 %s\n)\n", lotusDocsModuleVersion, bootstrapModuleVersion)
	if err := writeGeneratedFile(filepath.Join(outputDir, "go.mod"), []byte(module)); err != nil {
		return fmt.Errorf("write Hugo module go.mod: %w", err)
	}
	landing := `---
title: "semedit"
description: "Intent-driven code editing for AI agents."
icon: "rocket_launch"
draft: false
---

# Intent-driven code editing for AI agents

LLMs plan intent. Host compilers execute zero-token AST refactorings.

[Get started](docs/getting-started/)

[View on GitHub](https://github.com/spockz/semantic-editor)

Open source and MIT licensed.

## Why semedit?

semedit separates semantic intent, decided by the LLM, from mechanical syntax transformation, executed by local host CPUs, compilers, language servers, and AST tools.

### Deterministic edits

Compiler-backed transformations preserve syntactic validity across state transitions and eliminate fragile line-based patching.

### Symbol-based intent

Ask for Server.Start instead of hunting for a byte offset or line number. The symbol resolver finds the exact declaration before the host engine edits it.

### Structured feedback

The execution pipeline formats changes, checks diagnostics, and returns structured results so an agent can continue from compiler evidence.

### One contract for agents

Use the same semantic operations through the CLI or MCP, including rename, declaration insertion, function insertion, type insertion, and import organization.

## How it works

1. The LLM plans a high-level intent.
2. semedit resolves symbols and dispatches to the compiler or language server.
3. The host applies and formats the change deterministically.
4. Diagnostics return to the agent without re-emitting a full file diff.

## Explore the documentation

[Read the capability reference](docs/reference/)

[View empirical benchmarks](docs/benchmarks/)

The reference is generated from compiler capability declarations and executable txtar regression tests, so examples stay aligned with the implementation.
`
	if err := writeGeneratedFile(filepath.Join(outputDir, "content", "_index.md"), []byte(landing)); err != nil {
		return fmt.Errorf("write Hugo landing page: %w", err)
	}
	docsSection := `---
title: "Documentation"
description: "semedit installation and compiler-backed capability reference."
draft: false
weight: 10
---
`
	if err := writeGeneratedFile(filepath.Join(outputDir, "content", "docs", "_index.md"), []byte(docsSection)); err != nil {
		return fmt.Errorf("write Hugo docs section: %w", err)
	}
	landingData := `hero:
  enable: true
  weight: 10
  template: hero
  badge:
    text: "semedit"
    color: primary
    pill: false
    soft: true
  title: "Intent-driven code editing for AI agents"
  subtitle: "LLMs plan intent. Host compilers execute zero-token AST refactorings."
  ctaButton:
    icon: rocket_launch
    btnText: "Get Started"
    url: "/docs/getting-started/"
  cta2Button:
    icon: code
    btnText: "View on GitHub"
    url: "https://github.com/spockz/semantic-editor"
  info: "**Open Source** MIT Licensed."

featureGrid:
  enable: true
  weight: 20
  template: feature grid
  title: "Why semedit?"
  subtitle: "semedit separates semantic intent from mechanical syntax transformation, so agents can ask for a change and let local compiler tooling execute it precisely."
  items:
    - title: "Deterministic edits"
      icon: lock
      description: "Compiler-backed transformations preserve syntactic validity and eliminate fragile line-based patching."
    - title: "Symbol-based intent"
      icon: search
      description: "Ask for Server.Start instead of hunting for byte offsets or line numbers."
    - title: "Structured feedback"
      icon: speed
      description: "Formatting, diagnostics, and compiler evidence return to the agent as structured results."
    - title: "One contract for agents"
      icon: settings
      description: "Use the same semantic operations through the CLI or MCP, including rename and declaration insertion."

imageCompare:
  enable: false
  weight: 30
  template: image compare
`
	if err := writeGeneratedFile(filepath.Join(outputDir, "data", "landing.yaml"), []byte(landingData)); err != nil {
		return fmt.Errorf("write Hugo landing data: %w", err)
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

// renderHTML is retained as a reference during the Hugo migration and is not part of the build path.
//
//nolint:unused
func renderHTML(caps []CodeCapability, placements []string, examples []TxtarExample) string {
	var buf bytes.Buffer

	buf.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>semedit | Automated Capability Documentation & Executable Examples</title>
<style>
:root {
  --bg-primary: #0f141c;
  --bg-secondary: #161f2c;
  --bg-card: #1c283a;
  --bg-card-hover: #223249;
  --border-color: #2b3e58;
  --border-accent: #3b82f6;
  --text-primary: #f1f5f9;
  --text-secondary: #94a3b8;
  --text-muted: #64748b;
  --accent-blue: #38bdf8;
  --accent-green: #34d399;
  --accent-amber: #fbbf24;
  --accent-rose: #fb7185;
  --accent-purple: #c084fc;
  --code-bg: #090d13;
  --font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  --font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
}

* { box-sizing: border-box; margin: 0; padding: 0; }

body {
  background-color: var(--bg-primary);
  color: var(--text-primary);
  font-family: var(--font-sans);
  line-height: 1.6;
  display: flex;
  min-height: 100vh;
}

/* Sidebar */
aside.sidebar {
  width: 300px;
  background-color: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  padding: 1.5rem 1rem;
  position: sticky;
  top: 0;
  height: 100vh;
  overflow-y: auto;
  flex-shrink: 0;
}

.brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 2rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.brand-icon {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #2563eb, #38bdf8);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 1.1rem;
  color: white;
}

.brand-title {
  font-size: 1.15rem;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: var(--text-primary);
}

.brand-badge {
  font-size: 0.7rem;
  background: #1e3a8a;
  color: #93c5fd;
  padding: 2px 6px;
  border-radius: 9999px;
  font-weight: 600;
}

.nav-group {
  margin-bottom: 1.5rem;
}

.nav-heading {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  margin-bottom: 0.5rem;
  padding-left: 0.5rem;
}

.nav-link {
  display: block;
  padding: 0.4rem 0.6rem;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.875rem;
  border-radius: 6px;
  transition: all 0.15s ease;
}

.nav-link:hover {
  background-color: var(--bg-card);
  color: var(--accent-blue);
}

/* Main Content */
main.content {
  flex: 1;
  min-width: 0;
  padding: 2.5rem 3.5rem;
  max-width: 1200px;
  overflow-x: hidden;
}

.hero {
  margin-bottom: 3rem;
  padding-bottom: 2rem;
  border-bottom: 1px solid var(--border-color);
}

.hero-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--accent-blue);
  background: rgba(56, 189, 248, 0.1);
  border: 1px solid rgba(56, 189, 248, 0.25);
  padding: 0.3rem 0.75rem;
  border-radius: 9999px;
  margin-bottom: 1rem;
}

.hero h1 {
  font-size: 2.5rem;
  font-weight: 800;
  letter-spacing: -0.03em;
  margin-bottom: 0.75rem;
  background: linear-gradient(135deg, #f8fafc, #94a3b8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.hero p {
  font-size: 1.15rem;
  color: var(--text-secondary);
  max-width: 850px;
}

.github-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  margin-top: 1.25rem;
  padding: 0.4rem 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 600;
  text-decoration: none;
  transition: border-color 0.15s ease, color 0.15s ease;
}

.github-badge:hover {
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

.github-badge svg {
  fill: currentColor;
  height: 1rem;
  width: 1rem;
}

.section {
  margin-bottom: 4rem;
  scroll-margin-top: 2rem;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-color);
}

.section-title {
  font-size: 1.6rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--text-primary);
}

/* Tables */
.table-container {
  overflow-x: auto;
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  margin-bottom: 2rem;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
  text-align: left;
}

th {
  background-color: var(--bg-card);
  padding: 0.875rem 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-color);
}

td {
  padding: 0.875rem 1.25rem;
  border-bottom: 1px solid var(--border-color);
  color: var(--text-secondary);
  vertical-align: top;
}

tr:last-child td {
  border-bottom: none;
}

tr:hover td {
  background-color: rgba(255, 255, 255, 0.02);
}

/* Badges */
.badge {
  display: inline-block;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
}
.badge-prod { background: rgba(52, 211, 153, 0.15); color: #34d399; border: 1px solid rgba(52, 211, 153, 0.3); }
.badge-plan { background: rgba(148, 163, 184, 0.15); color: #94a3b8; border: 1px solid rgba(148, 163, 184, 0.3); }
.badge-pass { background: rgba(56, 189, 248, 0.15); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.3); }
.badge-mod { background: var(--bg-card); color: var(--accent-blue); font-family: var(--font-mono); font-size: 0.8rem; }

/* Callout Box */
.callout {
  padding: 1.25rem;
  border-radius: 8px;
  margin-bottom: 1.25rem;
  border-left: 4px solid;
  background-color: var(--bg-secondary);
}

.callout-error {
  border-left-color: var(--accent-rose);
  background: rgba(251, 113, 133, 0.05);
}

.callout-info {
  border-left-color: var(--accent-blue);
  background: rgba(56, 189, 248, 0.05);
}

.callout-title {
  font-weight: 700;
  font-size: 0.95rem;
  margin-bottom: 0.4rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.callout-error .callout-title { color: var(--accent-rose); }
.callout-info .callout-title { color: var(--accent-blue); }

.callout-desc {
  font-size: 0.9rem;
  color: var(--text-secondary);
}

/* Example Cards */
.example-card {
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  margin-bottom: 2.5rem;
  overflow: hidden;
}

.card-header {
  background-color: var(--bg-card);
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-title-group h3 {
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 0.2rem;
}

.card-file {
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--accent-blue);
  text-decoration: none;
}

.card-file:hover {
  text-decoration: underline;
}

.card-body {
  padding: 1.5rem;
}

.card-desc {
  font-size: 0.95rem;
  color: var(--text-secondary);
  margin-bottom: 1.5rem;
}

/* Steps list */
.step-box {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1rem;
  margin-bottom: 1rem;
}

.step-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.step-title {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-primary);
}

.dual-invocations {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
  margin-top: 0.75rem;
}

.inv-block {
  background-color: var(--code-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  min-width: 0;
  padding: 0.75rem;
}

.inv-label {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  margin-bottom: 0.4rem;
}

pre {
  font-family: var(--font-mono);
  font-size: 0.82rem;
  max-width: 100%;
  overflow-x: auto;
  line-height: 1.45;
}

code {
  font-family: var(--font-mono);
}

/* Code Diff View */
.diff-container {
  background-color: var(--code-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1rem;
  overflow-x: auto;
  margin-top: 1.5rem;
}

.diff-title {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-muted);
  margin-bottom: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.diff-line {
  font-family: var(--font-mono);
  font-size: 0.82rem;
  padding: 1px 4px;
  white-space: pre;
}

.diff-add {
  background-color: rgba(52, 211, 153, 0.15);
  color: #34d399;
}

.diff-del {
  background-color: rgba(251, 113, 133, 0.15);
  color: #fb7185;
}

.diff-same {
  color: var(--text-secondary);
}

/* Footer */
footer {
  margin-top: 5rem;
  padding-top: 2rem;
  border-top: 1px solid var(--border-color);
  font-size: 0.85rem;
  color: var(--text-muted);
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
</head>
<body>

<aside class="sidebar">
  <div class="brand">
    <div class="brand-icon">SE</div>
    <div>
      <div class="brand-title">semedit</div>
      <span class="brand-badge">DocGen PoC</span>
    </div>
  </div>

  <div class="nav-group">
    <div class="nav-heading">Overview</div>
    <a href="#overview" class="nav-link">Introduction</a>
    <a href="#matrix" class="nav-link">Edit &amp; Refactoring Capabilities</a>
    <a href="#limitations" class="nav-link">Language Rules & Limitations</a>
    <a href="#placements" class="nav-link">Placement Qualifiers</a>
  </div>

  <div class="nav-group">
    <div class="nav-heading">Executable Test Examples</div>`)

	for _, ex := range examples {
		anchorID := sanitizeID(ex.Filename)
		cleanName := strings.TrimSuffix(ex.Filename, ".txtar")
		cleanName = strings.ReplaceAll(cleanName, "_", " ")
		cleanName = toTitleCase(cleanName)
		fmt.Fprintf(&buf, `    <a href="#%s" class="nav-link">%s</a>`+"\n", anchorID, html.EscapeString(cleanName))
	}

	buf.WriteString(`  </div>

  <div class="nav-group">
    <div class="nav-heading">Verification</div>
    <a href="#ci-drift" class="nav-link">CI Drift Guarantees</a>
  </div>
</aside>

<main class="content">

  <section class="hero" id="overview">
    <div class="hero-tag">
      <span>●</span> Code-Derived AST Architecture
    </div>
    <h1>Automated Capability Documentation</h1>
    <p>
      Deterministic, zero-token refactoring capabilities extracted directly from compiler AST implementations and executable test archives (<code>txtar</code>).
    </p>
    <a class="github-badge" href="https://github.com/spockz/semantic-editor">
      <svg aria-hidden="true" viewBox="0 0 16 16"><path d="M8 0a8 8 0 0 0-2.53 15.59c.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82A7.66 7.66 0 0 1 8 4.73c.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.53.73.53 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8 8 0 0 0 8 0Z"/></svg>
      View on GitHub
    </a>
  </section>

  <!-- Edit and refactoring capability matrix -->
  <section class="section" id="matrix">
    <div class="section-header">
      <h2 class="section-title">Cross-Language Edit &amp; Refactoring Capability Matrix</h2>
      <span class="badge badge-prod">Live AST Extraction</span>
    </div>
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Language</th>
            <th>Maturity</th>
            <th>Supported Access Modifiers</th>
            <th>Supported Edit / Refactoring Capabilities</th>
          </tr>
        </thead>
        <tbody>`)

	for _, c := range caps {
		badgeClass := "badge-prod"
		if c.Maturity != "Production" {
			badgeClass = "badge-plan"
		}

		modSpans := make([]string, 0, len(c.SupportedModifiers))
		for _, m := range c.SupportedModifiers {
			modSpans = append(modSpans, fmt.Sprintf(`<span class="badge badge-mod">%s</span>`, m))
		}

		var opNames []string
		for op, meta := range c.Operations {
			if meta.Supported {
				opNames = append(opNames, op)
			}
		}
		sort.Strings(opNames)

		fmt.Fprintf(&buf, `
          <tr>
            <td><strong>%s</strong></td>
            <td><span class="badge %s">%s</span></td>
            <td>%s</td>
            <td><code>%s</code></td>
          </tr>`,
			html.EscapeString(c.DisplayName),
			badgeClass,
			html.EscapeString(c.Maturity),
			strings.Join(modSpans, " "),
			html.EscapeString(strings.Join(opNames, ", ")),
		)
	}

	buf.WriteString(`
        </tbody>
      </table>
    </div>
  </section>

  <!-- Language Limitations & Rules -->
  <section class="section" id="limitations">
    <div class="section-header">
      <h2 class="section-title">Compiler Constraints & Semantic Rules</h2>
    </div>`)

	for _, c := range caps {
		fmt.Fprintf(&buf, `<h3 style="margin: 1.5rem 0 1rem; color: var(--accent-blue);">%s Engine Constraints</h3>`+"\n", html.EscapeString(c.DisplayName))
		for _, rule := range c.Limitations {
			calloutClass := "callout-info"
			icon := "ℹ"
			if rule.Severity == "error" {
				calloutClass = "callout-error"
				icon = "✕"
			}
			fmt.Fprintf(&buf, `
      <div class="callout %s">
        <div class="callout-title"><span>%s</span> %s</div>
        <div class="callout-desc">%s</div>
      </div>`,
				calloutClass,
				icon,
				html.EscapeString(rule.Title),
				html.EscapeString(rule.Description),
			)
		}
	}

	buf.WriteString(`
  </section>

  <!-- Placement Qualifiers -->
  <section class="section" id="placements">
    <div class="section-header">
      <h2 class="section-title">Placement Qualifiers</h2>
      <span class="badge badge-prod">AST Enums</span>
    </div>
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Placement Mode</th>
            <th>Target Identifier Required</th>
            <th>Semantic Behavior</th>
          </tr>
        </thead>
        <tbody>`)

	placementDescriptions := map[string]struct {
		TargetReq bool
		Desc      string
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

	for _, p := range placements {
		info, ok := placementDescriptions[p]
		targetText := "No"
		desc := "Standard placement rule"
		if ok {
			if info.TargetReq {
				targetText = "<strong style='color: var(--accent-amber);'>Yes (--target)</strong>"
			}
			desc = info.Desc
		}
		fmt.Fprintf(&buf, `
          <tr>
            <td><code>%s</code></td>
            <td>%s</td>
            <td>%s</td>
          </tr>`,
			html.EscapeString(p),
			targetText,
			html.EscapeString(desc),
		)
	}

	buf.WriteString(`
        </tbody>
      </table>
    </div>
  </section>

  <!-- Executable Test Examples -->
  <section class="section">
    <div class="section-header">
      <h2 class="section-title">Executable Test Workflows (Mined from txtar)</h2>
      <span class="badge badge-pass">Golden Specifications</span>
    </div>
    <p style="color: var(--text-secondary); margin-bottom: 2rem;">
      Every scenario below is parsed directly from active, compiler-verified regression tests in <code>testdata/scripts/*.txtar</code>.
    </p>`)

	for _, ex := range examples {
		anchorID := sanitizeID(ex.Filename)
		sourceURL := fmt.Sprintf("%s/testdata/scripts/%s", githubSourceBaseURL, ex.Filename)
		fmt.Fprintf(&buf, `
    <div class="example-card" id="%s">
      <div class="card-header">
        <div class="card-title-group">
          <h3>%s</h3>
          <a class="card-file" href="%s">testdata/scripts/%s</a>
        </div>
        <span class="badge badge-pass">Test Passing ✓</span>
      </div>
      <div class="card-body">
        <p class="card-desc">%s</p>`,
			anchorID,
			html.EscapeString(ex.Title),
			html.EscapeString(sourceURL),
			html.EscapeString(ex.Filename),
			html.EscapeString(ex.Description),
		)

		for _, step := range ex.Steps {
			stepBadge := ""
			if step.IsNegated {
				errMsg := "Expected Error"
				if step.ExpectedErr != "" {
					errMsg = "Expected Error: " + step.ExpectedErr
				}
				stepBadge = fmt.Sprintf(`<span class="badge" style="background: rgba(251, 113, 133, 0.2); color: var(--accent-rose);">%s</span>`, html.EscapeString(errMsg))
			} else if step.ExpectedOut != "" {
				stepBadge = fmt.Sprintf(`<span class="badge" style="background: rgba(52, 211, 153, 0.2); color: var(--accent-green);">Assert: %s</span>`, html.EscapeString(step.ExpectedOut))
			}

			fmt.Fprintf(&buf, `
        <div class="step-box">
          <div class="step-meta">
            <span class="step-title">%s</span>
            %s
          </div>
          <div class="dual-invocations">
            <div class="inv-block">
              <div class="inv-label">CLI Invocation</div>
              <pre><code>$ %s</code></pre>
            </div>
            <div class="inv-block">
              <div class="inv-label">Equivalent MCP Tool Call (%s)</div>
              <pre><code>%s</code></pre>
            </div>
          </div>
        </div>`,
				html.EscapeString(step.Description),
				stepBadge,
				html.EscapeString(step.Command),
				html.EscapeString(step.MCPTool),
				html.EscapeString(step.MCPArgsJSON),
			)
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
				title := "Unified AST Transformation Diff"
				if len(outputs) > 1 && out.Path != "" {
					title = fmt.Sprintf("Unified AST Transformation Diff (%s)", out.Path)
				}
				fmt.Fprintf(&buf, `
        <div class="diff-container">
          <div class="diff-title">%s</div>
          <pre>`, html.EscapeString(title))
				for _, dl := range out.DiffLines {
					lineClass := "diff-same"
					switch dl.Type {
					case "add":
						lineClass = "diff-add"
					case "del":
						lineClass = "diff-del"
					}
					fmt.Fprintf(&buf, `<div class="diff-line %s">%s</div>`, lineClass, html.EscapeString(dl.Content))
				}
				buf.WriteString(`</pre>
        </div>`)
			}

			if out.Content != "" {
				title := "Expected Output State"
				if out.Path != "" {
					title = fmt.Sprintf("Expected Output State (%s)", out.Path)
				}
				fmt.Fprintf(&buf, `
        <div class="diff-container">
          <div class="diff-title">%s</div>
          <pre><code>%s</code></pre>
        </div>`, html.EscapeString(title), html.EscapeString(out.Content))
			}
		}

		buf.WriteString(`
      </div>
    </div>`)
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	fmt.Fprintf(&buf, `
  </section>

  <section class="section" id="ci-drift">
    <div class="section-header">
      <h2 class="section-title">Zero-Drift CI Invariant</h2>
    </div>
    <p style="color: var(--text-secondary); margin-bottom: 1rem;">
      Documentation synchronization is enforced during continuous integration:
    </p>
    <div class="callout callout-info">
      <div class="callout-title">Deterministic Compilation Gate</div>
      <div class="callout-desc">
        <code>make docgen</code> regenerates the Hugo site into <code>dist/docs</code> before publication. If any AST capability or test change alters documentation, the generated site changes with it.
      </div>
    </div>
  </section>

  <footer>
    <div>Generated automatically from compiler AST capabilities and regression test archives</div>
    <div class="footer-time">%s</div>
    <div>Zero-Token Semantic Editor Framework</div>
  </footer>

</main>

</body>
</html>`, now)

	return buf.String()
}

//nolint:unused
func sanitizeID(name string) string {
	r := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return r.ReplaceAllString(strings.TrimSuffix(name, ".txtar"), "-")
}

//nolint:unused
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
