// WHY: CLI contracts exercise Java's captured native type-expansion envelope against the current LSP overlay.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func fakeJavaExplicitTypeSymbols(state *fakeLanguageServerState, raw json.RawMessage) []fakeDocumentSymbol {
	document := fakeTextDocumentParams(raw)
	lines := strings.Split(state.documents[document.URI], "\n")
	methodPattern := regexp.MustCompile(`^  (?:public )?(?:static )?(int|String|Runnable) ([A-Za-z]+)\(([^)]*)\) \{`)
	var methods []fakeDocumentSymbol
	for index, line := range lines {
		match := methodPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		end := index
		if !strings.Contains(line, "}") {
			for end = index + 1; end < len(lines); end++ {
				if lines[end] == "  }" {
					break
				}
			}
		}
		if end >= len(lines) {
			failFakeLanguageServer(fmt.Errorf("Java explicit-type fixture method has no closing brace"))
		}
		nameStart := strings.Index(line, match[2]+"(")
		methods = append(methods, fakeDocumentSymbol{Name: match[2], Kind: 6, Detail: match[1] + " " + match[2] + "(" + match[3] + ")", Range: fakeLSPRange{Start: fakeLSPPosition{Line: index, Character: 2}, End: fakeLSPPosition{Line: end, Character: len(lines[end])}}, SelectionRange: fakeRangeAt(index, nameStart, nameStart+len(match[2]))})
	}
	classLine := strings.Count(strings.SplitN(state.documents[document.URI], "class Service {", 2)[0], "\n")
	return []fakeDocumentSymbol{{Name: "Service", Kind: 5, Range: fakeLSPRange{Start: fakeLSPPosition{Line: classLine}, End: fakeLSPPosition{Line: len(lines) - 1}}, SelectionRange: fakeRangeAt(classLine, 6, 13), Children: methods}}
}

func fakeJavaExplicitTypeAction(state *fakeLanguageServerState, raw json.RawMessage) []map[string]any {
	var request struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Range   fakeLSPRange `json:"range"`
		Context struct {
			Only []string `json:"only"`
		} `json:"context"`
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode explicit-type request: %w", err))
	}
	if len(request.Context.Only) != 1 || request.Context.Only[0] != "refactor" {
		return []map[string]any{}
	}
	lines := strings.Split(state.documents[request.TextDocument.URI], "\n")
	if request.Range.Start.Line < 0 || request.Range.Start.Line >= len(lines) {
		failFakeLanguageServer(fmt.Errorf("explicit-type selection line outside overlay"))
	}
	line := lines[request.Range.Start.Line]
	start, end := request.Range.Start.Character, request.Range.End.Character
	if start < 4 || end > len(line) || request.Range.Start.Line != request.Range.End.Line || line[start-4:start] != "var " {
		failFakeLanguageServer(fmt.Errorf("explicit-type selection must cover the exact var identifier"))
	}
	name := line[start:end]
	typeName := "int"
	if name == "items" {
		typeName = ("ArrayList<String>")
	}
	if name == "action" || os.Getenv("SEMEDIT_TEST_JAVA_EXPLICIT_TYPE") == "missing" {
		return []map[string]any{}
	}
	edit := map[string]any{"range": fakeRangeAt(request.Range.Start.Line, start-4, start-1), "newText": typeName}
	workspace := map[string]any{"changes": map[string]any{request.TextDocument.URI: []any{edit}}}
	action := map[string]any{"title": "Change type of '" + name + "' to '" + typeName + "'", "kind": "refactor", "diagnostics": []any{}, "edit": workspace}
	switch os.Getenv("SEMEDIT_TEST_JAVA_EXPLICIT_TYPE") {
	case "disabled":
		action["disabled"] = map[string]any{"reason": "not applicable"}
	case "extra":
		workspace["changes"] = map[string]any{request.TextDocument.URI: []any{edit, edit}}
	case "foreign":
		workspace["changes"] = map[string]any{request.TextDocument.URI + ".foreign": []any{edit}}
	case "command":
		delete(action, "edit")
		action["command"] = map[string]any{"command": "java.apply"}
	case "lazy":
		action["data"] = map[string]any{"resolve": true}
	case "duplicate":
		return []map[string]any{action, action}
	}
	return []map[string]any{action}
}

func captureFakeJavaOpen(state *fakeLanguageServerState, raw json.RawMessage) {
	if os.Getenv("SEMEDIT_TEST_JAVA_SHARED_BATCH") == "" && os.Getenv("SEMEDIT_TEST_JAVA_EXPLICIT_TYPE") == "" {
		return
	}
	opened := fakeTextDocumentParamsWithText(raw)
	state.documents[opened.URI] = opened.Text
}
