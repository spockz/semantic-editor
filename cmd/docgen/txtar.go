// txtar.go owns parsing of executable examples and their expected output diffs.
package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"semedit/internal/operation"
	"slices"
	"strings"
)

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
			return nil, fmt.Errorf("read txtar example %s: %w", cleanFilePath, err)
		}

		ex, parseErr := parseTxtarFile(file.Name(), string(content))
		if parseErr != nil {
			return nil, fmt.Errorf("parse txtar example %s: %w", file.Name(), parseErr)
		}
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

func parseTxtarFile(filename string, content string) (*TxtarExample, error) {
	parts := strings.Split(content, "\n-- ")
	fileMap := parseTxtarFiles(parts[1:])
	header, err := parseTxtarHeader(parts[0])
	if err != nil {
		return nil, err
	}
	if header.title == "" {
		header.title = strings.TrimSuffix(filename, ".txtar")
	}
	outputs := buildTxtarOutputs(fileMap, header.comparisons)
	first := firstTxtarExampleFiles(fileMap, outputs)

	return &TxtarExample{
		Filename:    filename,
		Title:       header.title,
		Description: strings.Join(header.descriptionLines, " "),
		Steps:       header.steps,
		InputFile:   first.input,
		InputCode:   first.inputContent,
		OutputFile:  first.output,
		OutputCode:  first.outputContent,
		DiffLines:   first.diff,
		Outputs:     outputs,
	}, nil
}

type parsedTxtarHeader struct {
	title            string
	descriptionLines []string
	steps            []TxtarStep
	comparisons      map[string]string
	currentStep      string
}

type txtarFirstFiles struct {
	input         string
	inputContent  string
	output        string
	outputContent string
	diff          []DiffLine
}

type txtarOutputInput struct {
	path    string
	content string
}

func parseTxtarFiles(parts []string) map[string]string {
	files := make(map[string]string)
	for _, section := range parts {
		separator := strings.Index(section, " --\n")
		if separator < 0 {
			separator = strings.Index(section, " --\r\n")
		}
		if separator < 0 {
			continue
		}
		name := strings.TrimSpace(section[:separator])
		files[name] = strings.TrimSpace(section[separator+4:])
	}
	return files
}

func parseTxtarHeader(source string) (parsedTxtarHeader, error) {
	header := parsedTxtarHeader{comparisons: make(map[string]string)}
	for rawLine := range strings.SplitSeq(source, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if comment, ok := strings.CutPrefix(line, "#"); ok {
			appendTxtarComment(&header, strings.TrimSpace(comment))
			continue
		}
		isCommand, err := appendTxtarCommand(&header, line)
		if err != nil {
			return parsedTxtarHeader{}, err
		}
		if isCommand || appendTxtarComparison(header.comparisons, line) {
			continue
		}
		appendTxtarExpectedOutput(&header, line)
	}
	return header, nil
}

func appendTxtarComment(header *parsedTxtarHeader, comment string) {
	switch {
	case header.title == "":
		header.title = comment
	case isTxtarStepComment(comment):
		header.currentStep = comment
	default:
		header.descriptionLines = append(header.descriptionLines, comment)
	}
}

func isTxtarStepComment(comment string) bool {
	for step := 1; step <= 7; step++ {
		if strings.HasPrefix(comment, fmt.Sprintf("%d.", step)) {
			return true
		}
	}
	return false
}

func appendTxtarCommand(header *parsedTxtarHeader, line string) (bool, error) {
	if !strings.HasPrefix(line, "exec semedit") && !strings.HasPrefix(line, "! exec semedit") {
		return false, nil
	}
	negative := strings.HasPrefix(line, "!")
	command := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "!"), "exec"))
	command = strings.TrimSpace(command)
	toolName, toolArgs := mapCLIToMCP(command)
	argsJSON, err := json.MarshalIndent(toolArgs, "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode MCP arguments for %q: %w", command, err)
	}
	description := header.currentStep
	if description == "" {
		description = fmt.Sprintf("Execute: %s", command)
	}
	header.steps = append(header.steps, TxtarStep{
		Number:      len(header.steps) + 1,
		Description: description,
		Command:     command,
		IsNegated:   negative,
		MCPTool:     toolName,
		MCPArgsJSON: string(argsJSON),
	})
	header.currentStep = ""
	return true, nil
}

func appendTxtarComparison(comparisons map[string]string, line string) bool {
	if !strings.HasPrefix(line, "cmp ") {
		return false
	}
	fields := strings.Fields(line)
	if len(fields) >= 3 {
		comparisons[fields[2]] = fields[1]
	}
	return true
}

func appendTxtarExpectedOutput(header *parsedTxtarHeader, line string) {
	if len(header.steps) == 0 {
		return
	}
	if pattern, ok := strings.CutPrefix(line, "stderr "); ok {
		header.steps[len(header.steps)-1].ExpectedErr = appendExpectedPattern(header.steps[len(header.steps)-1].ExpectedErr, pattern)
	}
	if pattern, ok := strings.CutPrefix(line, "stdout "); ok {
		header.steps[len(header.steps)-1].ExpectedOut = appendExpectedPattern(header.steps[len(header.steps)-1].ExpectedOut, pattern)
	}
}

func appendExpectedPattern(existing, pattern string) string {
	pattern = strings.Trim(pattern, `"'`)
	if existing == "" {
		return pattern
	}
	return existing + "\n" + pattern
}

func buildTxtarOutputs(files, comparisons map[string]string) []TxtarFileOutput {
	wantKeys := make([]string, 0)
	for name := range files {
		if strings.HasPrefix(name, "want/") || strings.HasPrefix(name, "want.") {
			wantKeys = append(wantKeys, name)
		}
	}
	slices.Sort(wantKeys)
	outputs := make([]TxtarFileOutput, 0, len(wantKeys))
	for _, wantKey := range wantKeys {
		input := resolveTxtarOutputInput(wantKey, files, comparisons)
		outputs = append(outputs, TxtarFileOutput{
			Path: input.path, Content: files[wantKey], DiffLines: generateDiff(input.content, files[wantKey]),
		})
	}
	return outputs
}

func resolveTxtarOutputInput(wantKey string, files, comparisons map[string]string) txtarOutputInput {
	if actual, ok := comparisons[wantKey]; ok {
		return txtarOutputInput{path: actual, content: files[actual]}
	}
	if after, ok := strings.CutPrefix(wantKey, "want/"); ok {
		return txtarOutputInput{path: after, content: files[after]}
	}
	if extension := filepath.Ext(wantKey); extension != "" {
		for name, content := range files {
			if strings.HasSuffix(name, extension) && !strings.HasPrefix(name, "want") {
				return txtarOutputInput{path: name, content: content}
			}
		}
	}
	return txtarOutputInput{path: wantKey}
}

func firstTxtarExampleFiles(files map[string]string, outputs []TxtarFileOutput) txtarFirstFiles {
	if len(outputs) > 0 {
		first := outputs[0]
		return txtarFirstFiles{
			input: first.Path, inputContent: files[first.Path], output: first.Path,
			outputContent: first.Content, diff: first.DiffLines,
		}
	}
	for name, content := range files {
		ext := filepath.Ext(name)
		if (ext == ".go" || ext == ".java" || ext == ".rs" || ext == ".scala" || ext == ".kt" || ext == ".hs" || ext == ".sh" || name == "Makefile") && !strings.HasPrefix(name, "want") {
			return txtarFirstFiles{input: name, inputContent: content}
		}
	}
	return txtarFirstFiles{}
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
