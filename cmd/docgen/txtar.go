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
