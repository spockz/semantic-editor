// Package main provides deterministic language-server stand-ins for CLI txtar coverage.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/spockz/semantic-editor/internal/lsp"
)

type fakeLSPPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type fakeLSPRange struct {
	Start fakeLSPPosition `json:"start"`
	End   fakeLSPPosition `json:"end"`
}

type fakeDocumentSymbol struct {
	Name           string               `json:"name"`
	Kind           int                  `json:"kind"`
	Detail         string               `json:"detail,omitempty"`
	URI            string               `json:"uri,omitempty"`
	Range          fakeLSPRange         `json:"range"`
	SelectionRange fakeLSPRange         `json:"selectionRange"`
	Children       []fakeDocumentSymbol `json:"children,omitempty"`
}

func runFakeRustAnalyzer() {
	symbol := fakeDocumentSymbol{Name: "Widget", Kind: 23, Range: fakeRange(0, 15), SelectionRange: fakeRange(7, 13)}
	if rawLine := os.Getenv("SEMEDIT_RUST_SYMBOL_LINE"); rawLine != "" {
		line, err := strconv.Atoi(rawLine)
		if err != nil {
			return
		}
		start, err := strconv.Atoi(os.Getenv("SEMEDIT_RUST_SYMBOL_START"))
		if err != nil {
			return
		}
		symbol.Range = fakeRangeAt(line, start-1, start+6)
		symbol.SelectionRange = fakeRangeAt(line, start, start+6)
	}
	runFakeLanguageServer(symbol)
}

func runFakeJava() {
	if err := os.Setenv("SEMEDIT_FAKE_JAVA_DIAGNOSTICS", "1"); err != nil {
		failFakeLanguageServer(fmt.Errorf("enable fake Java diagnostics: %w", err))
	}
	if len(os.Args) > 1 && os.Args[1] == "-version" {
		_, _ = os.Stderr.WriteString("openjdk version \"21.0.2\"\n")
		return
	}
	symbol := fakeDocumentSymbol{Name: "Widget", Kind: 5, Range: fakeRange(0, 14), SelectionRange: fakeRange(6, 12)}
	if os.Getenv("SEMEDIT_JAVA_RENAME_FILE_URI") != "" {
		symbol.Range = fakeRangeAt(2, 0, 22)
		symbol.SelectionRange = fakeRangeAt(2, 13, 19)
	}
	runFakeLanguageServer(symbol)
}

func runFakeKotlin() {
	if os.Getenv("SEMEDIT_TEST_KOTLIN_OUTLINE") != "" {
		symbol := fakeDocumentSymbol{
			Name: "Widget", Kind: 5, Detail: "server class detail",
			Range: fakeRangeAt(0, 0, 29), SelectionRange: fakeRangeAt(0, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "run", Kind: 6, Detail: "fun run()", Range: fakeRangeAt(0, 15, 27), SelectionRange: fakeRangeAt(0, 19, 22)}},
		}
		runFakeLanguageServer(symbol)
		return
	}
	if os.Getenv("SEMEDIT_TEST_KOTLIN_FUNCTION") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{"launch", 12, "", "", fakeRangeAt(1, 0, 15), fakeRangeAt(1, 4, 10), nil})
		return
	}
	runFakeLanguageServer(fakeDocumentSymbol{"Widget", 5, "", "", fakeRangeAt(2, 0, 29), fakeRangeAt(2, 6, 12), nil})
}

func runFakeMetals() {
	runFakeLanguageServer(fakeDocumentSymbol{Name: "Widget", Kind: 5, Range: fakeRange(0, 15), SelectionRange: fakeRange(7, 13)})
}

func runFakeGHC() {
	if len(os.Args) > 1 && os.Args[1] == "--numeric-version" {
		_, _ = os.Stdout.WriteString("9.8.2\n")
	}
}

func runFakeHLS() {
	if len(os.Args) > 1 && os.Args[1] == "--probe-tools" {
		_, _ = os.Stdout.WriteString("haskell-language-server version: 2.9.0 (GHC: 9.8.2)\n")
		return
	}
	runFakeLanguageServer(fakeDocumentSymbol{Name: "Widget", Kind: 5, Range: fakeRange(0, 20), SelectionRange: fakeRange(5, 11)})
}

func runFakeMaven() {
	found := false
	for _, arg := range os.Args[1:] {
		if arg == "-o" {
			found = true
		}
	}
	if os.Getenv("SEMEDIT_ASSERT_MAVEN_NETWORK") == "1" && found {
		os.Exit(23)
	}
	if !found {
		if os.Getenv("SEMEDIT_ASSERT_MAVEN_NETWORK") != "1" {
			os.Exit(19)
		}
	}
	_, _ = os.Stdout.WriteString("fake-maven-ok\n")
}

func fakeRange(start, end int) fakeLSPRange {
	return fakeRangeAt(0, start, end)
}

func fakeRangeAt(line, start, end int) fakeLSPRange {
	return fakeLSPRange{Start: fakeLSPPosition{Line: line, Character: start}, End: fakeLSPPosition{Line: line, Character: end}}
}

func runFakeLanguageServer(symbol fakeDocumentSymbol) {
	reader := lsp.NewFrameReader(os.Stdin, 0)
	writer := lsp.NewFrameWriter(os.Stdout, 0)
	state := fakeLanguageServerState{symbol: symbol, documents: make(map[string]string)}
	for {
		payload, err := reader.ReadMessage()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			failFakeLanguageServer(fmt.Errorf("read fake language server frame: %w", err))
		}
		request, err := decodeFakeLSPRequest(payload)
		if err != nil {
			failFakeLanguageServer(err)
		}
		if len(request.ID) == 0 {
			if !handleFakeLanguageNotification(writer, &state, request) {
				failFakeLanguageServer(errors.New("fake language server rejected notification"))
			}
			continue
		}
		result, keepRunning := handleFakeLanguageRequest(&state, request)
		if !keepRunning {
			return
		}
		if request.Method == "textDocument/rename" && os.Getenv("SEMEDIT_FAKE_JAVA_DIAGNOSTICS") != "" {
			document := fakeTextDocumentParams(request.Params)
			version := 2
			writeFakeDiagnostics(writer, document.URI, &version, []any{})
			if crossURI := os.Getenv("SEMEDIT_JAVA_CROSS_RENAME_URI"); crossURI != "" {
				writeFakeDiagnostics(writer, crossURI, &version, []any{})
			}
		}
		if err := writeFakeLSPResponse(writer, request.ID, result); err != nil {
			failFakeLanguageServer(err)
		}
	}
}

func failFakeLanguageServer(err error) {
	if _, writeErr := fmt.Fprintln(os.Stderr, "fake language server:", err); writeErr != nil {
		os.Exit(2)
	}
	os.Exit(2)
}

type fakeLSPRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type fakeLanguageServerState struct {
	symbol         fakeDocumentSymbol
	formattingSeen bool
	codeActionSeen bool
	documents      map[string]string
}

func decodeFakeLSPRequest(payload []byte) (fakeLSPRequest, error) {
	var request fakeLSPRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return fakeLSPRequest{}, err
	}
	return request, nil
}

func handleFakeLanguageNotification(writer *lsp.FrameWriter, state *fakeLanguageServerState, request fakeLSPRequest) bool {
	if request.Method == "textDocument/didOpen" {
		if os.Getenv("SEMEDIT_TEST_JAVA_SHARED_BATCH") != "" {
			var opened struct {
				TextDocument struct {
					URI  string `json:"uri"`
					Text string `json:"text"`
				} `json:"textDocument"`
			}
			if err := json.Unmarshal(request.Params, &opened); err != nil || opened.TextDocument.URI == "" {
				failFakeLanguageServer(errors.New("decode shared-batch didOpen content"))
			}
			state.documents[opened.TextDocument.URI] = opened.TextDocument.Text
		}
		writeConfiguredOpenDiagnostics(writer, request.Params)
		writeConfiguredKotlinDiagnostics(writer, request.Params)
	}
	if request.Method == "textDocument/didChange" && !handleFakeDocumentChange(writer, state, request.Params) {
		return false
	}
	if request.Method == "workspace/didRenameFiles" && os.Getenv("SEMEDIT_JAVA_RENAME_FILE_URI") != "" {
		var params struct {
			Files []struct {
				NewURI string `json:"newUri"`
			} `json:"files"`
		}
		if err := json.Unmarshal(request.Params, &params); err != nil || len(params.Files) != 1 || params.Files[0].NewURI == "" {
			failFakeLanguageServer(errors.New("invalid fake workspace/didRenameFiles notification"))
		}
	}
	if request.Method == "workspace/didChangeConfiguration" && !validConfiguredJavaSettings(request.Params) {
		return false
	}
	return true
}

func writeConfiguredOpenDiagnostics(writer *lsp.FrameWriter, raw json.RawMessage) {
	if os.Getenv("SEMEDIT_FAKE_JAVA_DIAGNOSTICS") == "" && os.Getenv("SEMEDIT_ASSERT_JAVA_SETTINGS") == "" && os.Getenv("SEMEDIT_ASSERT_JAVA_VERIFY") == "" {
		return
	}
	document := fakeTextDocumentParams(raw)
	var params struct {
		TextDocument struct {
			Version int `json:"version"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode fake open document version: %w", err))
	}
	version := params.TextDocument.Version
	writeFakeDiagnostics(writer, document.URI, &version, []any{})
}

func writeConfiguredKotlinDiagnostics(writer *lsp.FrameWriter, raw json.RawMessage) {
	if os.Getenv("SEMEDIT_TEST_KOTLIN_VERIFY") == "" && os.Getenv("SEMEDIT_TEST_KOTLIN_WRONG_URI_ONLY") == "" {
		return
	}
	document := fakeTextDocumentParams(raw)
	uri := document.URI
	if os.Getenv("SEMEDIT_TEST_KOTLIN_WRONG_URI_ONLY") != "" {
		uri = "file:///outside/Widget.kt"
	}
	diagnostics := []any{}
	if os.Getenv("SEMEDIT_TEST_KOTLIN_DIAGNOSTIC") != "" {
		diagnostics = []any{map[string]any{"message": "selected Kotlin warning", "severity": 2, "range": fakeRange(0, 0)}}
	}
	writeFakeDiagnostics(writer, uri, nil, diagnostics)
}

func fakeTextDocumentParams(raw json.RawMessage) struct {
	URI string `json:"uri"`
} {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode fake text document params: %w", err))
	}
	if params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("fake text document params missing URI"))
	}
	return params.TextDocument
}

func writeFakeDiagnostics(writer *lsp.FrameWriter, uri string, version *int, diagnostics []any) {
	paramsValue := map[string]any{"uri": uri, "diagnostics": diagnostics}
	if version != nil {
		paramsValue["version"] = *version
	}
	params, err := json.Marshal(paramsValue)
	if err != nil {
		failFakeLanguageServer(fmt.Errorf("encode fake diagnostics params: %w", err))
	}
	notification, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/publishDiagnostics", "params": json.RawMessage(params)})
	if err != nil {
		failFakeLanguageServer(fmt.Errorf("encode fake diagnostics notification: %w", err))
	}
	if err := writer.WriteMessage(notification); err != nil {
		failFakeLanguageServer(fmt.Errorf("write fake diagnostics notification: %w", err))
	}
}

func handleFakeDocumentChange(writer *lsp.FrameWriter, state *fakeLanguageServerState, raw json.RawMessage) bool {
	var change struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if err := json.Unmarshal(raw, &change); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode fake document change: %w", err))
	}
	if os.Getenv("SEMEDIT_ASSERT_JAVA_VERIFY") != "" && change.TextDocument.Version > 2 && (!state.formattingSeen || !state.codeActionSeen) {
		return false
	}
	if len(change.ContentChanges) > 0 {
		text := change.ContentChanges[len(change.ContentChanges)-1].Text
		state.documents[change.TextDocument.URI] = text
		updateFakeDocumentSymbolFromText(state, text)
	}
	if change.TextDocument.Version <= 2 {
		writeFakeDiagnostics(writer, change.TextDocument.URI, &change.TextDocument.Version, []any{})
		return true
	}
	writeFakeDiagnostics(writer, change.TextDocument.URI, &change.TextDocument.Version, []any{})
	return true
}

func updateFakeDocumentSymbolFromText(state *fakeLanguageServerState, text string) {
	lines := strings.Split(text, "\n")
	position := state.symbol.SelectionRange.Start
	if position.Line < 0 || position.Line >= len(lines) || position.Character < 0 {
		return
	}
	line := []rune(lines[position.Line])
	start, end := position.Character, state.symbol.SelectionRange.End.Character
	if start >= len(line) || end < start || end > len(line) {
		return
	}
	name := string(line[start:end])
	if name == "" || strings.ContainsAny(name, " \t{}();") {
		return
	}
	state.symbol.Name = name
	state.symbol.SelectionRange.End.Character = start + len([]rune(name))
}

func validConfiguredJavaSettings(raw json.RawMessage) bool {
	expected := os.Getenv("SEMEDIT_ASSERT_JAVA_SETTINGS")
	return expected == "" || assertJavaSettings(raw, expected)
}

func handleFakeLanguageRequest(state *fakeLanguageServerState, request fakeLSPRequest) (any, bool) {
	switch request.Method {
	case "initialize":
		return fakeInitializeResult(request.Params)
	case "workspace/didChangeConfiguration":
		return nil, validConfiguredJavaSettings(request.Params)
	case "textDocument/documentSymbol":
		return fakeDocumentSymbols(state, request.Params), true
	case "textDocument/references":
		return fakeJavaReferences(request.Params), true
	case "textDocument/prepareRename":
		if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_FUNCTION") != "" {
			return fakeRangeAt(2, 15, 24), true
		}
		if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_VARIABLE") != "" {
			return fakeRangeAt(3, 8, 17), true
		}
		if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_CONSTANT") != "" {
			return fakeRangeAt(2, 27, 36), true
		}
		return state.symbol.SelectionRange, true
	case "textDocument/formatting":
		state.formattingSeen = true
		if os.Getenv("SEMEDIT_JAVA_ORGANIZE") != "" {
			return []any{}, true
		}
		return fakeFormattingEdits(), true
	case "textDocument/codeAction":
		state.codeActionSeen = true
		return fakeCodeAction(request.Params), true
	case "java/listOverridableMethods":
		if os.Getenv("SEMEDIT_TEST_JAVA_IMPLEMENT_CONTRACT") != "" {
			return map[string]any{"type": "Widget", "methods": []map[string]any{{
				"bindingKey": "Lsample/Runner;.run()V", "name": "run", "parameters": []string{},
				"unimplemented": true, "declaringClass": "sample.Runner", "declaringClassType": "interface",
			}}}, true
		}
		return nil, true
	case "java/addOverridableMethods":
		if os.Getenv("SEMEDIT_TEST_JAVA_IMPLEMENT_CONTRACT") != "" {
			var params struct {
				Context struct {
					TextDocument struct {
						URI string `json:"uri"`
					} `json:"textDocument"`
				} `json:"context"`
			}
			if err := json.Unmarshal(request.Params, &params); err != nil || params.Context.TextDocument.URI == "" {
				failFakeLanguageServer(errors.New("decode fake add-overridable-methods context"))
			}
			return map[string]any{"changes": map[string]any{params.Context.TextDocument.URI: []map[string]any{{
				"range": fakeRangeAt(2, 0, 0), "newText": "  @Override\n  public void run() {}\n",
			}}}}, true
		}
		return nil, true
	case "java/move":
		return fakeJavaMove(request.Params), true
	case "textDocument/rename":
		return fakeRenameEdit(state.symbol, request.Params), true
	default:
		return nil, true
	}
}

func fakeJavaReferences(raw json.RawMessage) []map[string]any {
	var params struct {
		Context struct {
			IncludeDeclaration bool `json:"includeDeclaration"`
		} `json:"context"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.Context.IncludeDeclaration {
		failFakeLanguageServer(errors.New("Java references must exclude the declaration"))
	}
	uri := os.Getenv("SEMEDIT_JAVA_REFERENCE_URI")
	if uri == "" {
		return []map[string]any{}
	}
	line, err := strconv.Atoi(os.Getenv("SEMEDIT_JAVA_REFERENCE_LINE"))
	if err != nil {
		failFakeLanguageServer(fmt.Errorf("parse Java reference line: %w", err))
	}
	start, err := strconv.Atoi(os.Getenv("SEMEDIT_JAVA_REFERENCE_START"))
	if err != nil {
		failFakeLanguageServer(fmt.Errorf("parse Java reference start: %w", err))
	}
	end, err := strconv.Atoi(os.Getenv("SEMEDIT_JAVA_REFERENCE_END"))
	if err != nil {
		failFakeLanguageServer(fmt.Errorf("parse Java reference end: %w", err))
	}
	return []map[string]any{{"uri": uri, "range": fakeRangeAt(line, start, end)}}
}

func fakeInitializeResult(params json.RawMessage) (any, bool) {
	if !validConfiguredJavaSettings(params) {
		return nil, false
	}
	return map[string]any{"capabilities": map[string]any{}}, true
}

//nolint:gocognit,funlen // This fixture decodes the supported LSP document-symbol wire shapes in one place.
func fakeDocumentSymbols(state *fakeLanguageServerState, raw json.RawMessage) []fakeDocumentSymbol {
	symbol := state.symbol
	codeActionSeen := state.codeActionSeen
	if os.Getenv("SEMEDIT_TEST_JAVA_ELEMENT") != "" {
		return []fakeDocumentSymbol{{Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 4, Character: 1}}, SelectionRange: fakeRangeAt(2, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "limits", Kind: 8, Detail: "Map<String, Integer> limits",
				Range: fakeRangeAt(3, 2, 68), SelectionRange: fakeRangeAt(3, 37, 43)}}}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_MOVE_TOP_LEVEL") != "" {
		document := fakeTextDocumentParams(raw)
		parsed, parseErr := url.Parse(document.URI)
		if parseErr != nil {
			return nil
		}
		name := strings.TrimSuffix(filepath.Base(parsed.Path), ".java")
		return []fakeDocumentSymbol{{Name: name, Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 4}}, SelectionRange: fakeRangeAt(2, 13, 19)}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_MOVE_CONSTRUCT") != "" {
		document := fakeTextDocumentParams(raw)
		parsed, parseErr := url.Parse(document.URI)
		if parseErr != nil {
			return nil
		}
		if filepath.Base(parsed.Path) == "Target.java" {
			return []fakeDocumentSymbol{{Name: "Target", Kind: 5,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 3}}, SelectionRange: fakeRangeAt(1, 6, 12)}}
		}
		return []fakeDocumentSymbol{{Name: "Source", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 4}}, SelectionRange: fakeRangeAt(1, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "value", Kind: 6, Detail: "static int value()",
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 3}}, SelectionRange: fakeRangeAt(2, 13, 18)}}}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER") != "" || os.Getenv("SEMEDIT_TEST_JAVA_CHANGE_SIGNATURE") != "" || os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER_OBJECT") != "" {
		document := fakeTextDocumentParams(raw)
		parsed, parseErr := url.Parse(document.URI)
		if parseErr != nil {
			return nil
		}
		if filepath.Base(parsed.Path) == "Caller.java" {
			return []fakeDocumentSymbol{{Name: "Caller", Kind: 5,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 5}}, SelectionRange: fakeRangeAt(1, 6, 12),
				Children: []fakeDocumentSymbol{{Name: "run", Kind: 6, Detail: "void run(Service service)",
					Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 4}}, SelectionRange: fakeRangeAt(2, 7, 10)}}}}
		}
		detail := "void send(int id)"
		switch {
		case os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER_OBJECT") != "":
			detail = "void send(int id, boolean enabled)"
			if codeActionSeen {
				detail = "void send(SendOptions options)"
			}
		case os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER") != "" && codeActionSeen:
			detail = "void send(int id, boolean enabled)"
		}
		return []fakeDocumentSymbol{{Name: "Service", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 6}}, SelectionRange: fakeRangeAt(1, 6, 13),
			Children: []fakeDocumentSymbol{{Name: "send", Kind: 6, Detail: detail,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 5}}, SelectionRange: fakeRangeAt(2, 7, 11)}}}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_INTERFACE") != "" {
		document := fakeTextDocumentParams(raw)
		parsed, parseErr := url.Parse(document.URI)
		if parseErr != nil {
			return nil
		}
		contents, readErr := os.ReadFile(parsed.Path)
		if readErr != nil {
			return nil
		}
		class := fakeDocumentSymbol{Name: "Widget", Kind: 5,
			Range:          fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 3, Character: 1}},
			SelectionRange: fakeRangeAt(1, 13, 19),
			Children: []fakeDocumentSymbol{{Name: "size", Kind: 6, Detail: "public int size(String value)",
				Range: fakeRangeAt(2, 2, 58), SelectionRange: fakeRangeAt(2, 13, 17)}},
		}
		result := []fakeDocumentSymbol{class}
		if strings.Contains(string(contents), "interface Measurable") {
			result = append(result, fakeDocumentSymbol{Name: "Measurable", Kind: 11,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 5}, End: fakeLSPPosition{Line: 7, Character: 1}}, SelectionRange: fakeRangeAt(5, 10, 20)})
		}
		return result
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_INSERT_CASE") != "" {
		return []fakeDocumentSymbol{{Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 8, Character: 1}}, SelectionRange: fakeRangeAt(1, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "run", Kind: 6, Detail: "void run(int status)",
				Range:          fakeLSPRange{Start: fakeLSPPosition{Line: 2, Character: 2}, End: fakeLSPPosition{Line: 7, Character: 3}},
				SelectionRange: fakeRangeAt(2, 7, 10)}}}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_INLINE_FUNCTION") != "" {
		document := fakeTextDocumentParams(raw)
		parsed, parseErr := url.Parse(document.URI)
		if parseErr == nil && filepath.Base(parsed.Path) == "Helper.java" {
			return []fakeDocumentSymbol{{Name: "Helper", Kind: 5,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 4}}, SelectionRange: fakeRangeAt(1, 6, 12),
				Children: []fakeDocumentSymbol{{Name: "value", Kind: 6, Detail: "int value()",
					Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 3}}, SelectionRange: fakeRangeAt(2, 6, 11)}}}}
		}
		return []fakeDocumentSymbol{{Name: "Caller", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 6}}, SelectionRange: fakeRangeAt(1, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "run", Kind: 6, Detail: "int run()",
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 5}}, SelectionRange: fakeRangeAt(2, 6, 9)}}}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_IMPLEMENT_CONTRACT") != "" {
		document := fakeTextDocumentParams(raw)
		parsed, parseErr := url.Parse(document.URI)
		if parseErr == nil && filepath.Base(parsed.Path) == "Runner.java" {
			return []fakeDocumentSymbol{{Name: "Runner", Kind: 11,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 2}}, SelectionRange: fakeRangeAt(1, 10, 16)}}
		}
		children := []fakeDocumentSymbol{}
		ownerEnd := fakeLSPPosition{Line: 2}
		if codeActionSeen {
			ownerEnd = fakeLSPPosition{Line: 4}
			children = append(children, fakeDocumentSymbol{Name: "run", Kind: 6, Detail: "public void run()",
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 4}}, SelectionRange: fakeRangeAt(3, 14, 17)})
		}
		return []fakeDocumentSymbol{{Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: ownerEnd}, SelectionRange: fakeRangeAt(1, 6, 12), Children: children}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_FUNCTION") != "" {
		if codeActionSeen {
			return []fakeDocumentSymbol{{
				Name: "Widget", Kind: 5,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 8, Character: 1}}, SelectionRange: fakeRangeAt(1, 6, 12),
				Children: []fakeDocumentSymbol{
					{Name: "generated", Kind: 6, Detail: "private void generated()", Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2, Character: 2}, End: fakeLSPPosition{Line: 4, Character: 3}}, SelectionRange: fakeRangeAt(2, 15, 24)},
					{Name: "run", Kind: 6, Detail: "void run()", Range: fakeLSPRange{Start: fakeLSPPosition{Line: 5, Character: 2}, End: fakeLSPPosition{Line: 7, Character: 3}}, SelectionRange: fakeRangeAt(5, 7, 10)},
				},
			}}
		}
		return []fakeDocumentSymbol{{
			Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 5, Character: 1}}, SelectionRange: fakeRangeAt(1, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "run", Kind: 6, Detail: "void run()", Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2, Character: 2}, End: fakeLSPPosition{Line: 4, Character: 3}}, SelectionRange: fakeRangeAt(2, 7, 10)}},
		}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_INLINE_VARIABLE") != "" {
		methodEnd := fakeLSPPosition{Line: 5, Character: 3}
		ownerEnd := fakeLSPPosition{Line: 6, Character: 1}
		document := fakeTextDocumentParams(raw)
		parsed, parseErr := url.Parse(document.URI)
		if parseErr == nil {
			contents, readErr := os.ReadFile(parsed.Path)
			if readErr == nil && !strings.Contains(string(contents), "int limit") {
				codeActionSeen = true
			}
		}
		if codeActionSeen {
			methodEnd = fakeLSPPosition{Line: 4, Character: 3}
			ownerEnd = fakeLSPPosition{Line: 5, Character: 1}
		}
		return []fakeDocumentSymbol{{
			Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: ownerEnd}, SelectionRange: fakeRangeAt(1, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "run", Kind: 6, Detail: "void run()",
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2, Character: 2}, End: methodEnd}, SelectionRange: fakeRangeAt(2, 7, 10)}},
		}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_VARIABLE") != "" || os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_CONSTANT") != "" {
		return []fakeDocumentSymbol{{
			Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 5, Character: 1}}, SelectionRange: fakeRangeAt(1, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "run", Kind: 6, Detail: "void run()",
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2, Character: 2}, End: fakeLSPPosition{Line: 4, Character: 3}}, SelectionRange: fakeRangeAt(2, 7, 10)}},
		}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_INSERT_FIELD") != "" || os.Getenv("SEMEDIT_TEST_JAVA_INSERT_MEMBER") != "" {
		if os.Getenv("SEMEDIT_TEST_JAVA_SHARED_BATCH") != "" {
			document := fakeTextDocumentParams(raw)
			parsed, parseErr := url.Parse(document.URI)
			if parseErr != nil {
				return nil
			}
			contents, ok := state.documents[document.URI]
			if !ok {
				fileContents, readErr := os.ReadFile(parsed.Path)
				if readErr != nil {
					return nil
				}
				contents = string(fileContents)
			}
			lines := strings.Split(contents, "\n")
			methodLine, methodEndLine, ownerEndLine := -1, -1, -1
			for index, line := range lines {
				if strings.Contains(line, "void run() {") {
					methodLine = index
				}
				if methodLine >= 0 && index > methodLine && strings.TrimSpace(line) == "}" {
					methodEndLine = index
					break
				}
			}
			for index, line := range slices.Backward(lines) {
				if strings.TrimSpace(line) == "}" {
					ownerEndLine = index
					break
				}
			}
			if methodLine < 0 || methodEndLine < 0 || ownerEndLine < 0 {
				return nil
			}
			return []fakeDocumentSymbol{{Name: "Widget", Kind: 5,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: ownerEndLine, Character: 1}}, SelectionRange: fakeRangeAt(1, 6, 12),
				Children: []fakeDocumentSymbol{{Name: "run", Kind: 6, Detail: "void run()",
					Range: fakeLSPRange{Start: fakeLSPPosition{Line: methodLine, Character: 2}, End: fakeLSPPosition{Line: methodEndLine, Character: 3}}, SelectionRange: fakeRangeAt(methodLine, 7, 10)}}}}
		}
		widget := fakeDocumentSymbol{
			Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 3, Character: 1}}, SelectionRange: fakeRangeAt(1, 6, 12),
		}
		return []fakeDocumentSymbol{widget}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_DOC_COMMENT") != "" {
		return []fakeDocumentSymbol{{
			Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 4, Character: 1}}, SelectionRange: fakeRangeAt(1, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "run", Kind: 6, Detail: "void run()",
				Range: fakeRangeAt(3, 2, 15), SelectionRange: fakeRangeAt(3, 7, 10)}},
		}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_REMOVE_METHOD") != "" {
		method := "  private void run() { work(); }"
		return []fakeDocumentSymbol{{
			Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 6, Character: 1}}, SelectionRange: fakeRangeAt(1, 6, 12),
			Children: []fakeDocumentSymbol{
				{Name: "value", Kind: 8, Detail: "int value", Range: fakeRangeAt(2, 2, 20), SelectionRange: fakeRangeAt(2, 14, 19)},
				{Name: "run", Kind: 6,
					Range: fakeLSPRange{Start: fakeLSPPosition{Line: 4, Character: 2}, End: fakeLSPPosition{Line: 4, Character: len(method)}}, SelectionRange: fakeRangeAt(4, 17, 20)},
			},
		}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_CONSTRUCTS") != "" {
		return []fakeDocumentSymbol{{
			Name: "Widget", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1, Character: 0}, End: fakeLSPPosition{Line: 5, Character: 1}}, SelectionRange: fakeRangeAt(1, 6, 12),
			Children: []fakeDocumentSymbol{{Name: "run", Kind: 6,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: 2, Character: 2}, End: fakeLSPPosition{Line: 4, Character: 3}}, SelectionRange: fakeRangeAt(2, 7, 10)}},
		}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_OVERLOADS") != "" {
		return []fakeDocumentSymbol{{
			Name: "OrderService", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{}, End: fakeLSPPosition{Line: 3, Character: 1}}, SelectionRange: fakeRangeAt(0, 6, 18),
			Children: []fakeDocumentSymbol{
				{Name: "process", Kind: 6, Detail: "void process(Order o)", Range: fakeRangeAt(1, 2, 26), SelectionRange: fakeRangeAt(1, 7, 14)},
				{Name: "process", Kind: 6, Detail: "void process(Order o, boolean retry)", Range: fakeRangeAt(2, 2, 41), SelectionRange: fakeRangeAt(2, 7, 14)},
			},
		}}
	}
	if os.Getenv("SEMEDIT_TEST_KOTLIN_AMBIGUOUS") != "" {
		return []fakeDocumentSymbol{symbol, symbol}
	}
	return []fakeDocumentSymbol{symbol}
}

func fakeJavaMove(raw json.RawMessage) map[string]any {
	if os.Getenv("SEMEDIT_TEST_JAVA_MOVE_TOP_LEVEL") != "" {
		var resource struct {
			SourceURIs  []string `json:"sourceUris"`
			Destination struct {
				URI string `json:"uri"`
			} `json:"destination"`
		}
		if err := json.Unmarshal(raw, &resource); err != nil {
			failFakeLanguageServer(fmt.Errorf("decode Java resource move parameters: %w", err))
		}
		if len(resource.SourceURIs) != 1 {
			failFakeLanguageServer(fmt.Errorf("unexpected Java resource move parameters: %#v", resource))
		}
		sourceURI, err := url.Parse(resource.SourceURIs[0])
		if err != nil {
			failFakeLanguageServer(fmt.Errorf("parse Java resource move source URI: %w", err))
		}
		destination, err := url.Parse(resource.Destination.URI)
		if err != nil {
			failFakeLanguageServer(fmt.Errorf("parse Java resource move destination: %w", err))
		}
		destinationURI := (&url.URL{Scheme: "file", Path: filepath.Join(destination.Path, "Widget.java")}).String()
		callerURI := (&url.URL{Scheme: "file", Path: filepath.Join(filepath.Dir(filepath.Dir(sourceURI.Path)), "client", "Caller.java")}).String()
		edit := map[string]any{"documentChanges": []any{
			map[string]any{"textDocument": map[string]any{"uri": sourceURI.String()}, "edits": []any{map[string]any{"range": fakeRangeAt(0, 8, 14), "newText": "newpkg"}}},
			map[string]any{"textDocument": map[string]any{"uri": callerURI}, "edits": []any{map[string]any{"range": fakeRangeAt(1, 7, 20), "newText": "newpkg.Widget"}}},
			map[string]any{"kind": "rename", "oldUri": sourceURI.String(), "newUri": destinationURI},
		}}
		return map[string]any{"edit": edit}
	}
	var params struct {
		MoveKind string `json:"moveKind"`
		Params   struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		} `json:"params"`
		Destination string `json:"destination"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode Java move parameters: %w", err))
	}
	if params.MoveKind != "moveStaticMember" || params.Destination != "sample.Target" {
		failFakeLanguageServer(fmt.Errorf("unexpected Java move parameters: %#v", params))
	}
	sourceURI, err := url.Parse(params.Params.TextDocument.URI)
	if err != nil {
		failFakeLanguageServer(fmt.Errorf("parse Java move source URI: %w", err))
	}
	destinationURI := (&url.URL{Scheme: "file", Path: filepath.Join(filepath.Dir(sourceURI.Path), "Target.java")}).String()
	edit := map[string]any{"changes": map[string]any{
		params.Params.TextDocument.URI: []any{map[string]any{"range": fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 3}}, "newText": ""}},
		destinationURI:                 []any{map[string]any{"range": fakeRangeAt(2, 0, 0), "newText": "  static int value() { return 1; }\n"}},
	}}
	return map[string]any{"edit": edit}
}

func fakeFormattingEdits() []map[string]any {
	if os.Getenv("SEMEDIT_TEST_JAVA_CONSTRUCTS") != "" || os.Getenv("SEMEDIT_TEST_JAVA_ELEMENT") != "" || os.Getenv("SEMEDIT_TEST_JAVA_INSERT_CASE") != "" || os.Getenv("SEMEDIT_TEST_JAVA_DOC_COMMENT") != "" || os.Getenv("SEMEDIT_TEST_JAVA_CHANGE_SIGNATURE") != "" || os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER") != "" || os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER_OBJECT") != "" {
		return []map[string]any{}
	}
	return []map[string]any{{"range": fakeLSPRange{Start: fakeLSPPosition{}, End: fakeLSPPosition{Line: 3}}, "newText": "class Widget {}\n"}}
}

//nolint:gocognit,funlen // This fixture maps multiple JDT LS action shapes into the fake server response.
func fakeCodeAction(raw json.RawMessage) []map[string]any {
	if os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER") != "" || os.Getenv("SEMEDIT_TEST_JAVA_CHANGE_SIGNATURE") != "" || os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER_OBJECT") != "" {
		var params struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Context struct {
				Only []string `json:"only"`
			} `json:"context"`
		}
		if err := json.Unmarshal(raw, &params); err != nil {
			failFakeLanguageServer(fmt.Errorf("decode Java change-signature code action: %w", err))
		}
		if len(params.Context.Only) != 1 || params.Context.Only[0] != "refactor.change.signature" {
			return []map[string]any{}
		}
		selectedURI, err := url.Parse(params.TextDocument.URI)
		if err != nil {
			failFakeLanguageServer(fmt.Errorf("parse Java change-signature URI: %w", err))
		}
		callerURI := (&url.URL{Scheme: "file", Path: filepath.Join(filepath.Dir(selectedURI.Path), "Caller.java")}).String()
		serviceSignature := "void send(long identifier, boolean enabled)"
		serviceEdits := []map[string]any{{"range": fakeRangeAt(2, 2, 19), "newText": serviceSignature}}
		callerReplacement := "service.send(7, true);"
		switch {
		case os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER_OBJECT") != "":
			serviceSignature = "void send(SendOptions options)"
			serviceEdits[0] = map[string]any{"range": fakeRangeAt(2, 2, 2+len("void send(int id, boolean enabled)")), "newText": serviceSignature}
			callerReplacement = "    service.send(new SendOptions());"
			edit := map[string]any{"changes": map[string]any{
				params.TextDocument.URI: serviceEdits,
				callerURI:               []map[string]any{{"range": fakeRangeAt(3, 0, len("    service.send(loadId(), isEnabled());")), "newText": callerReplacement}},
			}}
			return []map[string]any{{"kind": "refactor.change.signature", "edit": edit}}
		case os.Getenv("SEMEDIT_TEST_JAVA_INTRODUCE_PARAMETER") != "":
			serviceSignature = "void send(int id, boolean enabled)"
			serviceEdits[0]["newText"] = serviceSignature
		default:
			serviceEdits = append(serviceEdits, map[string]any{"range": fakeRangeAt(3, 23, 25), "newText": "identifier"})
		}
		edit := map[string]any{"changes": map[string]any{
			params.TextDocument.URI: serviceEdits,
			callerURI:               []map[string]any{{"range": fakeRangeAt(3, 4, 20), "newText": callerReplacement}},
		}}
		return []map[string]any{{"kind": "refactor.change.signature", "edit": edit}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_INLINE_FUNCTION") != "" {
		var params struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Context struct {
				Only []string `json:"only"`
			} `json:"context"`
		}
		if err := json.Unmarshal(raw, &params); err != nil {
			failFakeLanguageServer(fmt.Errorf("decode Java inline-function code action: %w", err))
		}
		if len(params.Context.Only) == 1 && params.Context.Only[0] == "refactor.inline" {
			edit := map[string]any{"changes": map[string]any{params.TextDocument.URI: []map[string]any{{
				"range": fakeRangeAt(3, 11, 25), "newText": "41",
			}}}}
			return []map[string]any{{"kind": "refactor.inline", "edit": edit}}
		}
		return []map[string]any{}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_FUNCTION") != "" {
		var params struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Context struct {
				Only []string `json:"only"`
			} `json:"context"`
		}
		if err := json.Unmarshal(raw, &params); err != nil {
			failFakeLanguageServer(fmt.Errorf("decode Java extract-function code action: %w", err))
		}
		if len(params.Context.Only) != 1 || params.Context.Only[0] != "refactor.extract.function" {
			return []map[string]any{}
		}
		edit := map[string]any{"changes": map[string]any{params.TextDocument.URI: []map[string]any{
			{"range": fakeLSPRange{Start: fakeLSPPosition{Line: 2, Character: 0}, End: fakeLSPPosition{Line: 2, Character: 0}}, "newText": "  private void generated() {\n    consume(1);\n  }\n"},
			{"range": fakeRangeAt(3, 4, 15), "newText": "generated();"},
		}}}
		return []map[string]any{{"kind": "refactor.extract.function", "edit": edit}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_VARIABLE") != "" || os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_CONSTANT") != "" || os.Getenv("SEMEDIT_TEST_JAVA_INLINE_VARIABLE") != "" {
		var params struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Context struct {
				Only []string `json:"only"`
			} `json:"context"`
		}
		if err := json.Unmarshal(raw, &params); err != nil {
			failFakeLanguageServer(fmt.Errorf("decode Java extraction code action: %w", err))
		}
		if len(params.Context.Only) == 1 && params.Context.Only[0] == "refactor.extract.variable" {
			edit := map[string]any{"changes": map[string]any{params.TextDocument.URI: []map[string]any{
				{"range": fakeRangeAt(3, 4, 4), "newText": "int generated = 1 + 2;\n    "},
				{"range": fakeRangeAt(3, 12, 17), "newText": "generated"},
			}}}
			return []map[string]any{{"kind": "refactor.extract.variable", "edit": edit}}
		}
		if len(params.Context.Only) == 1 && params.Context.Only[0] == "refactor.extract.constant" {
			edit := map[string]any{"changes": map[string]any{params.TextDocument.URI: []map[string]any{
				{"range": fakeRangeAt(1, 14, 14), "newText": "\n  private static final int GENERATED = 1 + 2;"},
				{"range": fakeRangeAt(3, 11, 16), "newText": "GENERATED"},
			}}}
			return []map[string]any{{"kind": "refactor.extract.constant", "edit": edit}}
		}
		if len(params.Context.Only) == 1 && params.Context.Only[0] == "refactor.inline" {
			edit := map[string]any{"changes": map[string]any{params.TextDocument.URI: []map[string]any{
				{"range": fakeLSPRange{Start: fakeLSPPosition{Line: 3, Character: 0}, End: fakeLSPPosition{Line: 4, Character: 0}}, "newText": ""},
				{"range": fakeRangeAt(4, 12, 17), "newText": "10"},
			}}}
			return []map[string]any{{"kind": "refactor.inline", "edit": edit}}
		}
		return []map[string]any{}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_CONSTRUCTS") != "" {
		return []map[string]any{}
	}
	if os.Getenv("SEMEDIT_JAVA_ORGANIZE") != "" {
		return []map[string]any{}
	}
	document := fakeTextDocumentParams(raw)
	edit := map[string]any{"changes": map[string]any{
		document.URI: []map[string]any{{"range": fakeLSPRange{}, "newText": ""}},
	}}
	return []map[string]any{{"kind": "source.organizeImports", "edit": edit}}
}

func fakeRenameEdit(symbol fakeDocumentSymbol, raw json.RawMessage) map[string]any {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		NewName string `json:"newName"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode fake rename params: %w", err))
	}
	if params.TextDocument.URI == "" || params.NewName == "" {
		failFakeLanguageServer(errors.New("fake rename params missing URI or new name"))
	}
	uri := params.TextDocument.URI
	parsed, err := url.Parse(uri)
	if err != nil {
		failFakeLanguageServer(fmt.Errorf("parse fake rename URI: %w", err))
	}
	uri = parsed.String()
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_VARIABLE") != "" {
		return map[string]any{"changes": map[string][]map[string]any{uri: {
			{"range": fakeRangeAt(3, 8, 17), "newText": params.NewName},
			{"range": fakeRangeAt(4, 12, 21), "newText": params.NewName},
		}}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_FUNCTION") != "" {
		return map[string]any{"changes": map[string][]map[string]any{uri: {
			{"range": fakeRangeAt(2, 15, 24), "newText": params.NewName},
			{"range": fakeRangeAt(6, 4, 13), "newText": params.NewName},
		}}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_CONSTANT") != "" {
		return map[string]any{"changes": map[string][]map[string]any{uri: {
			{"range": fakeRangeAt(2, 27, 36), "newText": params.NewName},
			{"range": fakeRangeAt(4, 11, 20), "newText": params.NewName},
		}}}
	}
	changes := map[string][]map[string]any{
		uri: {{"range": symbol.SelectionRange, "newText": params.NewName}},
	}
	if crossURI := os.Getenv("SEMEDIT_JAVA_CROSS_RENAME_URI"); crossURI != "" {
		start, err := strconv.Atoi(os.Getenv("SEMEDIT_JAVA_CROSS_RENAME_START"))
		if err != nil {
			failFakeLanguageServer(fmt.Errorf("parse Java cross-file rename start: %w", err))
		}
		end, err := strconv.Atoi(os.Getenv("SEMEDIT_JAVA_CROSS_RENAME_END"))
		if err != nil {
			failFakeLanguageServer(fmt.Errorf("parse Java cross-file rename end: %w", err))
		}
		changes[crossURI] = []map[string]any{{"range": fakeRangeAt(0, start, end), "newText": params.NewName}}
	}
	if os.Getenv("SEMEDIT_RUST_UNSAFE") == "1" || os.Getenv("SEMEDIT_JAVA_UNSAFE") == "1" {
		changes["file:///foreign.rs"] = []map[string]any{{"range": symbol.SelectionRange, "newText": params.NewName}}
	}
	if newURI := os.Getenv("SEMEDIT_JAVA_RENAME_FILE_URI"); newURI != "" {
		return map[string]any{"documentChanges": []any{
			map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1}, "edits": changes[uri]},
			map[string]any{"kind": "rename", "oldUri": uri, "newUri": newURI},
		}}
	}
	return map[string]any{"changes": changes}
}

func writeFakeLSPResponse(writer *lsp.FrameWriter, id json.RawMessage, result any) error {
	response, err := json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  any             `json:"result"`
	}{JSONRPC: "2.0", ID: id, Result: result})
	if err != nil {
		return err
	}
	return writer.WriteMessage(response)
}

func assertJavaSettings(raw json.RawMessage, expected string) bool {
	var params struct {
		Settings map[string]any `json:"settings"`
	}
	if json.Unmarshal(raw, &params) != nil {
		return false
	}
	readBool := func(name string) (bool, bool) {
		value, ok := params.Settings[name]
		if !ok {
			return false, false
		}
		boolean, ok := value.(bool)
		return boolean, ok
	}
	maven, mavenOK := readBool("java.import.maven.enabled")
	gradle, gradleOK := readBool("java.import.gradle.enabled")
	autobuild, autobuildOK := readBool("java.autobuild.enabled")
	metadata, metadataOK := readBool("java.import.generatesMetadataFilesAtProjectRoot")
	if !mavenOK || !gradleOK || !autobuildOK || !metadataOK {
		return false
	}
	wantMaven := expected == "enabled"
	return maven == wantMaven && !gradle && !autobuild && !metadata
}
