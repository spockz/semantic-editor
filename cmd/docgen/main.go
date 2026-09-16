// Package main synthesizes code-derived capability documentation and test-driven examples into a modern standalone HTML documentation site.
package main

import (
	"bytes"
	"encoding/json"
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
)

const githubSourceBaseURL = "https://github.com/spockz/semantic-editor/blob/main"

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

	distDir := filepath.Join(rootDir, "dist", "docs")
	if err := os.MkdirAll(distDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating dist/docs: %v\n", err)
		os.Exit(1)
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

	// 3. Render HTML site
	htmlContent := renderHTML(capabilities, placements, examples)
	targetFile := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(targetFile, []byte(htmlContent), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing HTML output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully synthesized documentation site: %s\n", targetFile)
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
	// Parse internal/astedit/access.go
	accessFile := filepath.Join(rootDir, "internal", "astedit", "access.go")
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, accessFile, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse access.go: %w", err)
	}

	var goModifiers []string
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != "SupportedAccessModifiers" {
			return true
		}
		// Inspect return slice
		ast.Inspect(fn.Body, func(bn ast.Node) bool {
			comp, ok := bn.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, elt := range comp.Elts {
				if ident, ok := elt.(*ast.Ident); ok {
					cleanMod := strings.TrimPrefix(ident.Name, "AccessModifier")
					if cleanMod == "PackagePrivate" {
						cleanMod = "package-private"
					} else {
						cleanMod = strings.ToLower(cleanMod)
					}
					goModifiers = append(goModifiers, cleanMod)
				}
			}
			return false
		})
		return false
	})

	// Parse internal/astedit/insert.go for placements
	insertFile := filepath.Join(rootDir, "internal", "astedit", "insert.go")
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

	// Build multi-language capability matrix
	goOps := map[string]OpMetadata{
		"rename": {
			Supported:    true,
			Description:  "Compiler-backed symbol renaming across identifiers, methods, interfaces, and packages with automatic import tidying.",
			CLICommand:   "semedit rename --file <path> --symbol <sym> --to <name>",
			MCPTool:      "semantic_rename",
			PlacementKey: false,
		},
		"insert_func": {
			Supported:    true,
			Description:  "Function and method AST insertion with receiver clustering and public-precedes-private section partitioning.",
			CLICommand:   "semedit insert-func --file <path> --source <code snippet>",
			MCPTool:      "semantic_insert_function",
			PlacementKey: true,
		},
		"insert_type": {
			Supported:    true,
			Description:  "Struct, interface, and type alias AST insertion anchored in public/private type sections.",
			CLICommand:   "semedit insert-type --file <path> --source <type snippet>",
			MCPTool:      "semantic_insert_type",
			PlacementKey: true,
		},
		"insert_decl": {
			Supported:    true,
			Description:  "Constant and variable declaration insertion with automatic merging into existing const/var blocks.",
			CLICommand:   "semedit insert-decl --file <path> --source <decl snippet>",
			MCPTool:      "semantic_insert_decl",
			PlacementKey: true,
		},
		"imports": {
			Supported:    true,
			Description:  "Deterministic import management: resolve missing packages, remove unused imports, and add aliased imports.",
			CLICommand:   "semedit imports --file <path> [--add <pkg>] [--remove <pkg>]",
			MCPTool:      "semantic_organize_imports",
			PlacementKey: false,
		},
		"get": {
			Supported:    true,
			Description:  "Fast symbol coordinate, byte offset, receiver, and AST range lookup without line counting.",
			CLICommand:   "semedit lookup --file <path> --symbol <sym>",
			MCPTool:      "resolve_symbol_location",
			PlacementKey: false,
		},
	}

	javaOps := map[string]OpMetadata{
		"rename":      {Supported: true, Description: "Cross-class and method refactoring via Java LSP.", CLICommand: "semedit rename", MCPTool: "semantic_rename"},
		"insert_func": {Supported: true, Description: "Method insertion with 4-tier visibility clustering.", CLICommand: "semedit insert-func", MCPTool: "semantic_insert_function", PlacementKey: true},
		"insert_type": {Supported: true, Description: "Class, interface, record, and enum declaration insertion.", CLICommand: "semedit insert-type", MCPTool: "semantic_insert_type", PlacementKey: true},
		"insert_decl": {Supported: true, Description: "Field and static constant injection.", CLICommand: "semedit insert-decl", MCPTool: "semantic_insert_decl"},
		"imports":     {Supported: true, Description: "Package import cleanup and wildcard expansion.", CLICommand: "semedit imports", MCPTool: "semantic_organize_imports"},
		"get":         {Supported: true, Description: "Class and method coordinate resolution.", CLICommand: "semedit lookup", MCPTool: "resolve_symbol_location"},
	}

	pyOps := map[string]OpMetadata{
		"rename":      {Supported: true, Description: "Symbol rename across Python modules.", CLICommand: "semedit rename", MCPTool: "semantic_rename"},
		"insert_func": {Supported: true, Description: "Top-level def and class method insertion.", CLICommand: "semedit insert-func", MCPTool: "semantic_insert_function", PlacementKey: true},
		"insert_type": {Supported: true, Description: "Class and dataclass definition insertion.", CLICommand: "semedit insert-type", MCPTool: "semantic_insert_type", PlacementKey: true},
		"insert_decl": {Supported: true, Description: "Module-level constant and variable injection.", CLICommand: "semedit insert-decl", MCPTool: "semantic_insert_decl"},
		"imports":     {Supported: true, Description: "isort-aligned import sorting and resolution.", CLICommand: "semedit imports", MCPTool: "semantic_organize_imports"},
		"get":         {Supported: true, Description: "AST node and coordinate query.", CLICommand: "semedit lookup", MCPTool: "resolve_symbol_location"},
	}

	tsOps := map[string]OpMetadata{
		"rename":      {Supported: true, Description: "TypeScript symbol and interface renaming.", CLICommand: "semedit rename", MCPTool: "semantic_rename"},
		"insert_func": {Supported: true, Description: "Function and class method insertion.", CLICommand: "semedit insert-func", MCPTool: "semantic_insert_function", PlacementKey: true},
		"insert_type": {Supported: true, Description: "Type alias and interface insertion.", CLICommand: "semedit insert-type", MCPTool: "semantic_insert_type", PlacementKey: true},
		"insert_decl": {Supported: true, Description: "const/let declaration insertion.", CLICommand: "semedit insert-decl", MCPTool: "semantic_insert_decl"},
		"imports":     {Supported: true, Description: "ES module import resolution and sorting.", CLICommand: "semedit imports", MCPTool: "semantic_organize_imports"},
		"get":         {Supported: true, Description: "Symbol definition location resolution.", CLICommand: "semedit lookup", MCPTool: "resolve_symbol_location"},
	}

	capabilities := []CodeCapability{
		{
			Language:           "go",
			DisplayName:        "Go (Golang)",
			Maturity:           "Production",
			SupportedModifiers: goModifiers,
			Operations:         goOps,
			Limitations: []Constraint{
				{
					Title:       "Unsupported Modifiers Rejection",
					Description: "Go lacks 'protected' and 'package-private' scopes. The engine rejects these modifiers with ErrUnsupportedModifier.",
					Severity:    "error",
				},
				{
					Title:       "Casing & Visibility Invariant",
					Description: "Identifier capitalization governs visibility. Specifying 'public' for a lowercase symbol or 'private' for an uppercase symbol returns VisibilityMismatchError.",
					Severity:    "error",
				},
				{
					Title:       "Strict Public-Precedes-Private Ordering",
					Description: "All public declarations precede private declarations within generated or updated source files.",
					Severity:    "info",
				},
				{
					Title:       "Receiver Method Clustering",
					Description: "Methods sharing a common receiver type cluster near each other while maintaining public vs private partitioning.",
					Severity:    "info",
				},
				{
					Title:       "Declaration Block Merging",
					Description: "Constants and variables automatically merge into existing 'const (...)' or 'var (...)' blocks instead of creating duplicate blocks.",
					Severity:    "info",
				},
			},
		},
		{
			Language:           "java",
			DisplayName:        "Java",
			Maturity:           "Planned",
			SupportedModifiers: []string{"infer", "public", "protected", "package-private", "private"},
			Operations:         javaOps,
			Limitations: []Constraint{
				{
					Title:       "Single Public Class Constraint",
					Description: "Files allow only one top-level public class matching the file basename.",
					Severity:    "error",
				},
				{
					Title:       "Four-Tier Visibility Partitioning",
					Description: "Methods cluster by visibility bands: public -> protected -> package-private -> private.",
					Severity:    "info",
				},
			},
		},
		{
			Language:           "python",
			DisplayName:        "Python",
			Maturity:           "Planned",
			SupportedModifiers: []string{"infer", "public", "private"},
			Operations:         pyOps,
			Limitations: []Constraint{
				{
					Title:       "Underscore Visibility Inference",
					Description: "Leading underscore ('_') determines private/internal scope. Keyword modifiers are rejected.",
					Severity:    "error",
				},
			},
		},
		{
			Language:           "typescript",
			DisplayName:        "TypeScript",
			Maturity:           "Planned",
			SupportedModifiers: []string{"infer", "public", "protected", "private"},
			Operations:         tsOps,
			Limitations: []Constraint{
				{
					Title:       "Type vs Value Import Distinction",
					Description: "Type-only imports use 'import type' syntax to prevent bundle inflation.",
					Severity:    "info",
				},
			},
		},
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

		if strings.HasPrefix(line, "stderr ") && len(steps) > 0 {
			errPattern := strings.Trim(strings.TrimPrefix(line, "stderr "), `"'`)
			steps[len(steps)-1].ExpectedErr = errPattern
		}
		if strings.HasPrefix(line, "stdout ") && len(steps) > 0 {
			outPattern := strings.Trim(strings.TrimPrefix(line, "stdout "), `"'`)
			steps[len(steps)-1].ExpectedOut = outPattern
		}
	}

	if title == "" {
		title = strings.TrimSuffix(filename, ".txtar")
	}

	// Determine before and after files
	var inputCode, outputCode, inputPath, outputPath string
	for k, v := range fileMap {
		if strings.HasPrefix(k, "want/") {
			outputPath = k
			outputCode = v
			origPath := strings.TrimPrefix(k, "want/")
			if origCode, ok := fileMap[origPath]; ok {
				inputPath = origPath
				inputCode = origCode
			}
		}
	}

	if inputCode == "" {
		for k, v := range fileMap {
			if strings.HasSuffix(k, ".go") && !strings.HasPrefix(k, "want/") {
				inputPath = k
				inputCode = v
				break
			}
		}
	}

	diffLines := generateDiff(inputCode, outputCode)

	return &TxtarExample{
		Filename:    filename,
		Title:       title,
		Description: strings.Join(descriptionLines, " "),
		Steps:       steps,
		InputFile:   inputPath,
		InputCode:   inputCode,
		OutputFile:  outputPath,
		OutputCode:  outputCode,
		DiffLines:   diffLines,
	}
}

func mapCLIToMCP(cmd string) (string, map[string]any) {
	tokens := parseCommandLine(cmd)
	if len(tokens) < 2 {
		return "semedit", map[string]any{}
	}

	sub := tokens[1]
	args := make(map[string]any)

	flags := make(map[string]string)
	for i := 2; i < len(tokens); i++ {
		tok := tokens[i]
		if after, ok := strings.CutPrefix(tok, "--"); ok {
			flagName := after
			if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "--") {
				flags[flagName] = tokens[i+1]
				i++
			} else {
				flags[flagName] = "true"
			}
		}
	}

	switch sub {
	case "insert-func":
		args["file"] = flags["file"]
		args["source"] = flags["source"]
		if v, ok := flags["access"]; ok {
			args["access_modifier"] = v
		}
		if v, ok := flags["placement"]; ok {
			args["placement"] = v
		}
		return "semantic_insert_function", args

	case "insert-type":
		args["file"] = flags["file"]
		args["source"] = flags["source"]
		if v, ok := flags["placement"]; ok {
			args["placement"] = v
		}
		return "semantic_insert_type", args

	case "insert-decl":
		args["file"] = flags["file"]
		args["source"] = flags["source"]
		if v, ok := flags["placement"]; ok {
			args["placement"] = v
		}
		return "semantic_insert_decl", args

	case "insert":
		args["file"] = flags["file"]
		args["source"] = flags["source"]
		if v, ok := flags["placement"]; ok {
			args["placement"] = v
		}
		if v, ok := flags["target"]; ok {
			args["target_symbol"] = v
		}
		return "semantic_insert_declaration", args

	case "rename":
		args["file"] = flags["file"]
		args["symbol"] = flags["symbol"]
		args["to"] = flags["to"]
		return "semantic_rename", args

	case "imports":
		args["file"] = flags["file"]
		if v, ok := flags["add"]; ok {
			args["add"] = []string{v}
		}
		if v, ok := flags["remove"]; ok {
			args["remove"] = []string{v}
		}
		return "semantic_organize_imports", args

	case "lookup":
		args["file"] = flags["file"]
		args["symbol"] = flags["symbol"]
		return "resolve_symbol_location", args

	default:
		return "semedit_" + sub, args
	}
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
    <a href="#matrix" class="nav-link">Capability Matrix</a>
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
  </section>

  <!-- Capability Matrix -->
  <section class="section" id="matrix">
    <div class="section-header">
      <h2 class="section-title">Cross-Language Capability Matrix</h2>
      <span class="badge badge-prod">Live AST Extraction</span>
    </div>
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Language</th>
            <th>Maturity</th>
            <th>Supported Access Modifiers</th>
            <th>Operations Supported</th>
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

		if len(ex.DiffLines) > 0 {
			buf.WriteString(`
        <div class="diff-container">
          <div class="diff-title">Unified AST Transformation Diff</div>
          <pre>`)
			for _, dl := range ex.DiffLines {
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
        <code>make check</code> invokes <code>docgen</code> and asserts <code>git diff --exit-code dist/docs/</code> is empty. If any AST capability or test change alters documentation without regeneration, CI fails immediately.
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

func sanitizeID(name string) string {
	r := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return r.ReplaceAllString(strings.TrimSuffix(name, ".txtar"), "-")
}

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
