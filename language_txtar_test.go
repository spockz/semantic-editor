// Package main provides deterministic language-server stand-ins for CLI txtar coverage.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"

	"semedit/internal/lsp"
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
	if len(os.Args) > 1 && os.Args[1] == "-version" {
		_, _ = os.Stderr.WriteString("openjdk version \"21.0.2\"\n")
		return
	}
	runFakeLanguageServer(fakeDocumentSymbol{Name: "Widget", Kind: 5, Range: fakeRange(0, 14), SelectionRange: fakeRange(6, 12)})
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
	state := fakeLanguageServerState{symbol: symbol}
	for {
		payload, err := reader.ReadMessage()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			return
		}
		request, err := decodeFakeLSPRequest(payload)
		if err != nil {
			failFakeLanguageServer(err)
		}
		if len(request.ID) == 0 {
			if !handleFakeLanguageNotification(writer, state, request) {
				failFakeLanguageServer(errors.New("fake language server rejected notification"))
			}
			continue
		}
		result, keepRunning := handleFakeLanguageRequest(&state, request)
		if !keepRunning {
			return
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
}

func decodeFakeLSPRequest(payload []byte) (fakeLSPRequest, error) {
	var request fakeLSPRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return fakeLSPRequest{}, err
	}
	return request, nil
}

func handleFakeLanguageNotification(writer *lsp.FrameWriter, state fakeLanguageServerState, request fakeLSPRequest) bool {
	if request.Method == "textDocument/didOpen" {
		writeConfiguredOpenDiagnostics(writer, request.Params)
		writeConfiguredKotlinDiagnostics(writer, request.Params)
	}
	if request.Method == "textDocument/didChange" && !handleFakeDocumentChange(writer, state, request.Params) {
		return false
	}
	if request.Method == "workspace/didChangeConfiguration" && !validConfiguredJavaSettings(request.Params) {
		return false
	}
	return true
}

func writeConfiguredOpenDiagnostics(writer *lsp.FrameWriter, raw json.RawMessage) {
	if os.Getenv("SEMEDIT_ASSERT_JAVA_SETTINGS") == "" && os.Getenv("SEMEDIT_ASSERT_JAVA_VERIFY") == "" {
		return
	}
	document := fakeTextDocumentParams(raw)
	version := 1
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

func handleFakeDocumentChange(writer *lsp.FrameWriter, state fakeLanguageServerState, raw json.RawMessage) bool {
	var change struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &change); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode fake document change: %w", err))
	}
	if os.Getenv("SEMEDIT_ASSERT_JAVA_VERIFY") != "" && change.TextDocument.Version > 2 && (!state.formattingSeen || !state.codeActionSeen) {
		return false
	}
	if change.TextDocument.Version <= 2 {
		return true
	}
	writeFakeDiagnostics(writer, change.TextDocument.URI, &change.TextDocument.Version, []any{})
	return true
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
		return fakeDocumentSymbols(state.symbol), true
	case "textDocument/prepareRename":
		return state.symbol.SelectionRange, true
	case "textDocument/formatting":
		state.formattingSeen = true
		return fakeFormattingEdits(), true
	case "textDocument/codeAction":
		state.codeActionSeen = true
		return fakeCodeAction(request.Params), true
	case "textDocument/rename":
		return fakeRenameEdit(state.symbol, request.Params), true
	default:
		return nil, true
	}
}

func fakeInitializeResult(params json.RawMessage) (any, bool) {
	if !validConfiguredJavaSettings(params) {
		return nil, false
	}
	return map[string]any{"capabilities": map[string]any{}}, true
}

func fakeDocumentSymbols(symbol fakeDocumentSymbol) []fakeDocumentSymbol {
	if os.Getenv("SEMEDIT_TEST_KOTLIN_AMBIGUOUS") != "" {
		return []fakeDocumentSymbol{symbol, symbol}
	}
	return []fakeDocumentSymbol{symbol}
}

func fakeFormattingEdits() []map[string]any {
	return []map[string]any{{"range": fakeLSPRange{Start: fakeLSPPosition{}, End: fakeLSPPosition{Line: 3}}, "newText": "class Widget {}\n"}}
}

func fakeCodeAction(raw json.RawMessage) []map[string]any {
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
	if parsed, err := url.Parse(uri); err == nil {
		uri = parsed.String()
	}
	changes := map[string][]map[string]any{
		uri: {{"range": symbol.SelectionRange, "newText": params.NewName}},
	}
	if os.Getenv("SEMEDIT_RUST_UNSAFE") == "1" || os.Getenv("SEMEDIT_JAVA_UNSAFE") == "1" {
		changes["file:///foreign.rs"] = []map[string]any{{"range": symbol.SelectionRange, "newText": params.NewName}}
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
