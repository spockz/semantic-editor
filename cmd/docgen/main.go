// Package main synthesizes code-derived capability documentation and test-driven examples into a Hugo source tree.
package main

import (
	"bytes"
	"cmp"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/operation"
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
	shortcodesDir := filepath.Join(outputDir, "layouts", "shortcodes")
	if err := os.RemoveAll(shortcodesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing stale generated shortcodes: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(shortcodesDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Hugo shortcode directory: %v\n", err)
		os.Exit(1)
	}
	for _, stalePath := range []string{
		filepath.Join(outputDir, "content", "docs", "index.md"),
		filepath.Join(outputDir, "content", "docs", "getting-started"),
		filepath.Join(outputDir, "content", "docs", "reference"),
		filepath.Join(outputDir, "content", "docs", "benchmarks.md"),
		filepath.Join(outputDir, "content", "docs", "benchmarks"),
		filepath.Join(outputDir, "data", "landing.yaml"),
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
	benchmarkDocumentation, err := renderBenchmarkDocumentation(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering benchmark documentation: %v\n", err)
		os.Exit(1)
	}
	benchmarksDir := filepath.Join(outputDir, "content", "docs", "benchmarks")
	if err := os.MkdirAll(benchmarksDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating benchmarks documentation directory: %v\n", err)
		os.Exit(1)
	}
	if err := writeGeneratedFile(filepath.Join(benchmarksDir, "_index.md"), []byte(benchmarkDocumentation.Index)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing benchmarks index: %v\n", err)
		os.Exit(1)
	}
	aggregatesDir := filepath.Join(benchmarksDir, "aggregates")
	if err := os.MkdirAll(aggregatesDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating benchmark aggregate directory: %v\n", err)
		os.Exit(1)
	}
	if err := writeGeneratedFile(filepath.Join(aggregatesDir, "index.md"), []byte(benchmarkDocumentation.Aggregates)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing benchmark aggregates: %v\n", err)
		os.Exit(1)
	}
	runsDir := filepath.Join(benchmarksDir, "runs")
	if err := os.MkdirAll(runsDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating benchmark runs directory: %v\n", err)
		os.Exit(1)
	}
	if err := writeGeneratedFile(filepath.Join(runsDir, "_index.md"), []byte(`---
title: "Benchmark runs"
draft: false
weight: 22
---

Individual benchmark observations are linked from the [empirical benchmark overview](/docs/benchmarks/).
`)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing benchmark runs index: %v\n", err)
		os.Exit(1)
	}
	for _, run := range benchmarkDocumentation.Runs {
		runDir := filepath.Join(runsDir, run.ID)
		if err := os.MkdirAll(runDir, 0o750); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating benchmark run directory: %v\n", err)
			os.Exit(1)
		}
		if err := writeGeneratedFile(filepath.Join(runDir, "index.md"), []byte(renderBenchmarkRunDoc(run))); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing benchmark run %s: %v\n", run.ID, err)
			os.Exit(1)
		}
	}
	if err := writeGeneratedFile(filepath.Join(shortcodesDir, "benchmark-code.html"), []byte(benchmarkCodeShortcode)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing benchmark shortcode: %v\n", err)
		os.Exit(1)
	}
	if err := writeLandingAssets(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing landing assets: %v\n", err)
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
	slices.SortFunc(examples, func(a, b TxtarExample) int {
		return cmp.Compare(a.Filename, b.Filename)
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
	slices.Sort(wantKeys)

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

	for i := range len(cmd) {
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
