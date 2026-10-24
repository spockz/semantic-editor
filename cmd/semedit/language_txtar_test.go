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
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{})
		return
	}
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_VALUE") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{Name: "run", Kind: 12, Range: fakeLSPRange{Start: fakeLSPPosition{Line: 0, Character: 0}, End: fakeLSPPosition{Line: 4, Character: 0}}, SelectionRange: fakeRangeAt(0, 3, 6)})
		return
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{Name: "run", Kind: 12})
		return
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") != "" || os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{Name: "run", Kind: 12, Range: fakeRangeAt(0, 0, 54), SelectionRange: fakeRangeAt(0, 3, 6)})
		return
	}
	if os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{})
		return
	}
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
	if os.Getenv("SEMEDIT_TEST_SCALA_INLINE_VALUE") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{Name: "run", Kind: 6, Range: fakeRangeAt(1, 2, 5), SelectionRange: fakeRangeAt(1, 6, 9)})
		return
	}
	if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_CALLABLE") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{Name: "run", Kind: 6, Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1, Character: 2}, End: fakeLSPPosition{Line: 3, Character: 3}}, SelectionRange: fakeRangeAt(1, 6, 9)})
		return
	}
	if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_VARIABLE") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{Name: "run", Kind: 6, Range: fakeRangeAt(1, 2, 4), SelectionRange: fakeRangeAt(1, 6, 9)})
		return
	}
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
		if request.Method == "workspace/executeCommand" && os.Getenv("SEMEDIT_TEST_SCALA_INLINE_VALUE") != "" {
			if !handleFakeScalaInlineCommand(reader, writer, &state, request) {
				failFakeLanguageServer(errors.New("fake Metals rejected inline-value command"))
			}
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
	symbol          fakeDocumentSymbol
	formattingSeen  bool
	codeActionSeen  bool
	rustURI         string
	rustRenamed     bool
	rustActionCount int
	rustSource      string
	rustSelection   fakeLSPRange
	documents       map[string]string
}

func decodeFakeLSPRequest(payload []byte) (fakeLSPRequest, error) {
	var request fakeLSPRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return fakeLSPRequest{}, err
	}
	return request, nil
}

func handleFakeLanguageNotification(writer *lsp.FrameWriter, state *fakeLanguageServerState, request fakeLSPRequest) bool {
	if request.Method == "initialized" && (os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") != "" || os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" || os.Getenv("SEMEDIT_TEST_RUST_INLINE_VALUE") != "" || os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") != "" || os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST") != "") {
		writeFakeRustServerStatus(writer)
	}
	validateFakeRustClose(request)
	if request.Method == "textDocument/didOpen" {
		captureFakeRustCallableOpen(state, request.Params)
		captureFakeScalaInlineOpen(state, request.Params)
		if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_CALLABLE") != "" {
			opened := fakeTextDocumentParamsWithText(request.Params)
			state.rustURI = opened.URI
			state.documents[opened.URI] = opened.Text
		}
		if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") != "" {
			var opened struct {
				TextDocument struct {
					URI  string `json:"uri"`
					Text string `json:"text"`
				} `json:"textDocument"`
			}
			if err := json.Unmarshal(request.Params, &opened); err != nil || opened.TextDocument.URI == "" {
				failFakeLanguageServer(errors.New("decode Rust batch didOpen"))
			}
			state.rustURI = opened.TextDocument.URI
			state.rustSource = opened.TextDocument.Text
			state.rustRenamed = strings.Contains(opened.TextDocument.Text, "fn execute()")
		}
		if os.Getenv("SEMEDIT_TEST_RUST_INLINE_VALUE") != "" {
			opened := fakeTextDocumentParamsWithText(request.Params)
			state.rustURI = opened.URI
			state.rustSource = opened.Text
		}
		if os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST") != "" {
			opened := fakeTextDocumentParamsWithText(request.Params)
			state.rustURI = opened.URI
			state.rustSource = opened.Text
		}
		captureFakeJavaOpen(state, request.Params)
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

func captureFakeScalaInlineOpen(state *fakeLanguageServerState, raw json.RawMessage) {
	if os.Getenv("SEMEDIT_TEST_SCALA_INLINE_VALUE") == "" {
		return
	}
	opened := fakeTextDocumentParamsWithText(raw)
	state.rustURI = opened.URI
	state.rustSource = opened.Text
	state.documents[opened.URI] = opened.Text
}

func validateFakeRustClose(request fakeLSPRequest) {
	if request.Method != "textDocument/didClose" || os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") == "" && os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") == "" && os.Getenv("SEMEDIT_TEST_RUST_INLINE_VALUE") == "" && os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") == "" && os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST") == "" {
		return
	}
	var closed struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(request.Params, &closed); err != nil || closed.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("invalid Rust didClose payload"))
	}
}

func captureFakeRustCallableOpen(state *fakeLanguageServerState, raw json.RawMessage) {
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") == "" && os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") == "" {
		return
	}
	var opened struct {
		TextDocument struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &opened); err != nil || opened.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Rust callable didOpen"))
	}
	state.rustURI = opened.TextDocument.URI
	state.rustSource = opened.TextDocument.Text
}

func writeFakeRustServerStatus(writer *lsp.FrameWriter) {
	notification, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": "experimental/serverStatus",
		"params": map[string]any{"health": "ok", "quiescent": true},
	})
	if err != nil {
		failFakeLanguageServer(fmt.Errorf("encode fake rust-analyzer status: %w", err))
	}
	if err := writer.WriteMessage(notification); err != nil {
		failFakeLanguageServer(fmt.Errorf("write fake rust-analyzer status: %w", err))
	}
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
		if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" || os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") == "constant" {
			state.rustSource = text
		}
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
		return fakePrepareRename(state, request.Params), true
	case "textDocument/formatting":
		state.formattingSeen = true
		if os.Getenv("SEMEDIT_JAVA_ORGANIZE") != "" {
			return []any{}, true
		}
		return fakeFormattingEdits(), true
	case "textDocument/codeAction":
		return fakeCodeActionRequest(state, request.Params)
	case "codeAction/resolve":
		return fakeResolveCodeActionRequest(state, request.Params)
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
		if os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") != "" {
			return fakeRustInlineCallableRename(state, request.Params)
		}
		if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_CALLABLE") != "" {
			return fakeScalaCallableRename(state, request.Params), true
		}
		if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_VARIABLE") != "" {
			return fakeScalaRename(request.Params), true
		}
		if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") == "constant" {
			return fakeRustConstantRename(state, request.Params)
		}
		if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") != "" || os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" {
			return fakeRustRename(state, request.Params)
		}
		return fakeRenameEdit(state.symbol, request.Params), true
	default:
		return nil, true
	}
}

func fakeCodeActionRequest(state *fakeLanguageServerState, params json.RawMessage) (any, bool) {
	if os.Getenv("SEMEDIT_TEST_JAVA_EXPLICIT_TYPE") != "" {
		return fakeJavaExplicitTypeAction(state, params), true
	}
	state.codeActionSeen = true
	if os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST") != "" {
		return fakeRustPromoteLocalToConstAction(state, params)
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_MAKE_STATIC") != "" {
		var request struct {
			Context struct {
				Only []string `json:"only"`
			} `json:"context"`
		}
		if err := json.Unmarshal(params, &request); err != nil {
			failFakeLanguageServer(fmt.Errorf("decode Java make-static action context: %w", err))
		}
		if len(request.Context.Only) == 1 && request.Context.Only[0] == "source.organizeImports" {
			return []map[string]any{}, true
		}
		return fakeJavaMakeStaticAction(params), true
	}
	if os.Getenv("SEMEDIT_TEST_SCALA_INLINE_VALUE") != "" {
		return fakeScalaInlineAction(state, params), true
	}
	if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_CALLABLE") != "" {
		return fakeScalaExtractCallableAction(state, params), true
	}
	if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_VARIABLE") != "" {
		return fakeScalaExtractAction(state, params), true
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") != "" {
		if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") == "constant" {
			return fakeRustExtractConstantAction(state, params)
		}
		return fakeRustExtractAction(state, params)
	}
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_VALUE") != "" {
		return fakeRustInlineAction(state, params), true
	}
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") != "" {
		return fakeRustInlineCallableAction(state, params)
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" {
		return fakeRustExtractCallableAction(state, params)
	}
	return fakeCodeAction(params), true
}

func fakeResolveCodeActionRequest(state *fakeLanguageServerState, params json.RawMessage) (any, bool) {
	if os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST") != "" {
		return fakeRustResolvePromoteLocalToConstAction(state, params)
	}
	if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_CALLABLE") != "" {
		return fakeScalaResolveExtractCallableAction(state, params)
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" {
		return fakeRustResolveExtractCallableAction(state, params)
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") == "constant" {
		return fakeRustResolveExtractConstantAction(state, params)
	}
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") != "" {
		return fakeRustResolveInlineCallableAction(state, params)
	}
	if mode := os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE"); mode == "lazy" || mode == "disabled-resolved" {
		return fakeRustResolveExtractAction(state, params)
	}
	return nil, true
}

func fakePrepareRename(state *fakeLanguageServerState, raw json.RawMessage) any {
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") != "" {
		return fakeRustInlineCallablePrepareRename(state, raw)
	}
	if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_CALLABLE") != "" {
		return fakeNamedPrepareRename(raw, "Scala callable", "newMethod")
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" {
		return fakeNamedPrepareRename(raw, "Rust callable", "fun_name")
	}
	if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_VARIABLE") != "" {
		return fakeNamedPrepareRename(raw, "Scala extraction", "newValue")
	}
	mode := os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE")
	if mode != "" {
		return fakeRustExtractVariablePrepareRename(state, raw, mode)
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_FUNCTION") != "" {
		return fakeRangeAt(2, 15, 24)
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_VARIABLE") != "" {
		return fakeRangeAt(3, 8, 17)
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_EXTRACT_CONSTANT") != "" {
		return fakeRangeAt(2, 27, 36)
	}
	return state.symbol.SelectionRange
}

func fakeNamedPrepareRename(raw json.RawMessage, label, name string) fakeLSPRange {
	var params struct {
		Position fakeLSPPosition `json:"position"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode %s prepareRename: %w", label, err))
	}
	return fakeRangeAt(params.Position.Line, params.Position.Character, params.Position.Character+len(name))
}

func fakeRustExtractVariablePrepareRename(state *fakeLanguageServerState, raw json.RawMessage, mode string) fakeLSPRange {
	var params struct {
		Position fakeLSPPosition `json:"position"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode Rust extraction prepareRename: %w", err))
	}
	if mode == "constant" {
		start := fakeLSPByteOffset(state.rustSource, params.Position)
		if start < 0 || start >= len(state.rustSource) {
			failFakeLanguageServer(errors.New("Rust constant rename position is outside source"))
		}
		annotation := strings.Index(state.rustSource, "const EXTRACTED: i32")
		if annotation >= 0 && annotation+len("const EXTRACTED: ") == start {
			failFakeLanguageServer(errors.New("Rust primitive annotation must not use prepareRename"))
		}
		end := start
		for end < len(state.rustSource) && fakeIdentifierContinue(state.rustSource[end]) {
			end++
		}
		return fakeLSPRangeAtOffsets(state.rustSource, start, end)
	}
	if mode == "method" {
		return fakeRangeAt(5, 16, 25)
	}
	length := len("extracted")
	if mode == "batch" && params.Position.Line == 0 && params.Position.Character == 3 {
		length = len("run")
	}
	return fakeLSPRange{Start: params.Position, End: fakeLSPPosition{Line: params.Position.Line, Character: params.Position.Character + length}}
}

func fakeRustInlineCallablePrepareRename(state *fakeLanguageServerState, raw json.RawMessage) fakeLSPRange {
	var params struct {
		Position fakeLSPPosition `json:"position"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode Rust inline-call prepareRename: %w", err))
	}
	start := fakeLSPByteOffset(state.rustSource, params.Position)
	if start < 0 || start >= len(state.rustSource) {
		failFakeLanguageServer(errors.New("Rust inline-call rename position is outside source"))
	}
	end := start
	for end < len(state.rustSource) && isFakeIdentifier(state.rustSource[start:end+1]) {
		end++
	}
	return fakeLSPRangeAtOffsets(state.rustSource, start, end)
}

func fakeScalaExtractAction(state *fakeLanguageServerState, raw json.RawMessage) []map[string]any {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Range fakeLSPRange `json:"range"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Scala extraction code action"))
	}
	state.rustURI = params.TextDocument.URI
	edit := map[string]any{"changes": map[string]any{params.TextDocument.URI: []any{
		map[string]any{"range": fakeRangeAt(1, 27, 32), "newText": "newValue"},
		map[string]any{"range": fakeRangeAt(1, 18, 19), "newText": " {\n    val newValue = 1 + 2\n    "},
		map[string]any{"range": fakeRangeAt(1, 33, 33), "newText": "\n  }"},
	}}}
	if os.Getenv("SEMEDIT_TEST_SCALA_EDIT_MODE") == "foreign" {
		edit["changes"].(map[string]any)["file:///workspace/Other.scala"] = []any{}
	}
	return []map[string]any{{
		"title": "Extract `1 + 2` as value", "kind": "refactor.extract",
		"edit": edit,
	}}
}

func fakeScalaInlineAction(state *fakeLanguageServerState, raw json.RawMessage) []map[string]any {
	params := fakeTextDocumentParams(raw)
	if params.URI != state.rustURI || !strings.Contains(state.rustSource, "val value = 42") {
		failFakeLanguageServer(errors.New("Scala inline action target does not match the opened source"))
	}
	position := fakeLSPPosition{Line: 2, Character: 8}
	return []map[string]any{{
		"title": "Inline `value`", "kind": "refactor.inline",
		"command": map[string]any{"title": "Inline value", "command": "inline-value", "arguments": []any{map[string]any{
			"textDocument": map[string]string{"uri": params.URI}, "position": position,
		}}},
	}}
}

func handleFakeScalaInlineCommand(reader *lsp.FrameReader, writer *lsp.FrameWriter, state *fakeLanguageServerState, request fakeLSPRequest) bool {
	var command struct {
		Command   string            `json:"command"`
		Arguments []json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(request.Params, &command); err != nil || command.Command != "inline-value" || len(command.Arguments) != 1 {
		return false
	}
	edit := map[string]any{"changes": map[string]any{state.rustURI: []any{
		map[string]any{"range": fakeLSPRange{Start: fakeLSPPosition{Line: 2, Character: 0}, End: fakeLSPPosition{Line: 3, Character: 0}}, "newText": ""},
		map[string]any{"range": fakeRangeAt(3, 12, 17), "newText": "42"},
		map[string]any{"range": fakeRangeAt(3, 19, 24), "newText": "42"},
	}}}
	callback, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "scala-inline-edit", "method": "workspace/applyEdit", "params": map[string]any{"edit": edit},
	})
	if err != nil || writer.WriteMessage(callback) != nil {
		return false
	}
	response, err := reader.ReadMessage()
	if err != nil {
		return false
	}
	var result struct {
		ID     string `json:"id"`
		Result struct {
			Applied bool `json:"applied"`
		} `json:"result"`
	}
	if json.Unmarshal(response, &result) != nil || result.ID != "scala-inline-edit" || !result.Result.Applied {
		return false
	}
	return true
}

func fakeScalaExtractCallableAction(state *fakeLanguageServerState, raw json.RawMessage) []map[string]any {
	params := fakeTextDocumentParams(raw)
	var request struct {
		Range fakeLSPRange `json:"range"`
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode Scala callable selection: %w", err))
	}
	state.rustURI = params.URI
	data := map[string]any{
		"param": map[string]any{"uri": params.URI}, "range": request.Range,
		"extractPosition": request.Range.Start, "codeActionName": "ExtractMethodCodeAction",
	}
	methodAction := map[string]any{"title": "Extract selection as method in method `run`", "kind": "refactor.extract", "data": data}
	objectData := map[string]any{
		"param": map[string]any{"uri": params.URI}, "range": request.Range,
		"extractPosition": fakeLSPPosition{Line: 0, Character: 2}, "codeActionName": "ExtractMethodCodeAction",
	}
	return []map[string]any{
		methodAction,
		{"title": "Extract selection as method in object `Thing`", "kind": "refactor.extract", "data": objectData},
	}
}

func fakeScalaResolveExtractCallableAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var action struct {
		Title string          `json:"title"`
		Kind  string          `json:"kind"`
		Data  json.RawMessage `json:"data"`
	}
	var data struct {
		Param struct {
			URI string `json:"uri"`
		} `json:"param"`
		Range fakeLSPRange `json:"range"`
	}
	if err := json.Unmarshal(raw, &action); err != nil || action.Title != "Extract selection as method in method `run`" || action.Kind != "refactor.extract" || json.Unmarshal(action.Data, &data) != nil || data.Param.URI == "" {
		failFakeLanguageServer(errors.New("decode Scala extract-method resolve action"))
	}
	source := state.documents[data.Param.URI]
	start, end := fakeLSPByteOffset(source, data.Range.Start), fakeLSPByteOffset(source, data.Range.End)
	if start < 0 || end <= start || end > len(source) {
		failFakeLanguageServer(errors.New("invalid Scala extract-method resolve range"))
	}
	lineStart := strings.LastIndex(source[:start], "\n") + 1
	indent := source[lineStart:start]
	if strings.TrimSpace(indent) != "" {
		failFakeLanguageServer(errors.New("Scala extract-method test selection is not line-indented"))
	}
	statement := source[start:end]
	edit := map[string]any{"changes": map[string]any{data.Param.URI: []any{
		map[string]any{"range": data.Range, "newText": "newMethod()"},
		map[string]any{"range": fakeLSPRange{Start: data.Range.Start, End: data.Range.Start}, "newText": "def newMethod(): Unit =\n" + indent + "  " + statement + "\n\n" + indent},
	}}}
	if os.Getenv("SEMEDIT_TEST_SCALA_CALLABLE_EDIT_MODE") == "foreign" {
		edit["changes"].(map[string]any)["file:///workspace/Other.scala"] = []any{}
	}
	return map[string]any{"title": action.Title, "kind": action.Kind, "data": action.Data, "edit": edit}, true
}

func fakeScalaCallableRename(state *fakeLanguageServerState, raw json.RawMessage) map[string]any {
	var params struct {
		NewName      string `json:"newName"`
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.NewName == "" || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Scala callable rename"))
	}
	source := state.documents[params.TextDocument.URI]
	definition := strings.Index(source, "newMethod")
	if definition < 0 {
		failFakeLanguageServer(errors.New("find Scala callable definition rename span"))
	}
	call := strings.Index(source[definition+len("newMethod"):], "newMethod")
	if call < 0 {
		failFakeLanguageServer(errors.New("find Scala callable rename spans"))
	}
	call += definition + len("newMethod")
	return map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": params.TextDocument.URI, "version": 2},
		"edits": []any{
			map[string]any{"range": fakeLSPRangeAtOffsets(source, definition, definition+len("newMethod")), "newText": params.NewName},
			map[string]any{"range": fakeLSPRangeAtOffsets(source, call, call+len("newMethod")), "newText": params.NewName},
		},
	}}}
}

func fakeTextDocumentParamsWithText(raw json.RawMessage) struct{ URI, Text string } {
	var params struct {
		TextDocument struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Scala callable didOpen"))
	}
	return struct{ URI, Text string }{URI: params.TextDocument.URI, Text: params.TextDocument.Text}
}

func fakeLSPByteOffset(source string, position fakeLSPPosition) int {
	if position.Line < 0 || position.Character < 0 {
		return -1
	}
	lineStart := 0
	for range position.Line {
		next := strings.IndexByte(source[lineStart:], '\n')
		if next < 0 {
			return -1
		}
		lineStart += next + 1
	}
	lineEnd := len(source)
	if next := strings.IndexByte(source[lineStart:], '\n'); next >= 0 {
		lineEnd = lineStart + next
	}
	if lineStart+position.Character > lineEnd {
		return -1
	}
	return lineStart + position.Character
}

func fakeLSPRangeAtOffsets(source string, start, end int) fakeLSPRange {
	position := func(offset int) fakeLSPPosition {
		prefix := source[:offset]
		line := strings.Count(prefix, "\n")
		lineStart := strings.LastIndex(prefix, "\n") + 1
		return fakeLSPPosition{Line: line, Character: offset - lineStart}
	}
	return fakeLSPRange{Start: position(start), End: position(end)}
}

func fakeScalaRename(raw json.RawMessage) map[string]any {
	var params struct {
		NewName      string `json:"newName"`
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.NewName == "" || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Scala extraction rename"))
	}
	version := 2
	if os.Getenv("SEMEDIT_TEST_SCALA_EDIT_MODE") == "stale-rename" {
		version = 1
	}
	return map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": params.TextDocument.URI, "version": version},
		"edits": []any{
			map[string]any{"range": fakeRangeAt(2, 8, 16), "newText": params.NewName},
			map[string]any{"range": fakeRangeAt(3, 12, 20), "newText": params.NewName},
		},
	}}}
}

func fakeRustExtractAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" {
		return fakeRustExtractCallableAction(state, raw)
	}
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode rust extraction code action"))
	}
	state.rustURI = params.TextDocument.URI
	mode := os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE")
	state.rustActionCount++
	if mode == "native-shape" {
		action := map[string]any{
			"title": "Extract into variable", "kind": "refactor.extract",
			"data": map[string]any{"id": "extract_variable:RefactorExtract:0"},
			"edit": map[string]any{"documentChanges": []any{map[string]any{
				"textDocument": map[string]any{"uri": params.TextDocument.URI, "version": 1},
				"edits": []any{
					map[string]any{"range": fakeRangeAt(1, 4, 26), "newText": "let extracted = 1_i32 + 2_i32"},
					map[string]any{"range": fakeRangeAt(1, 27, 27), "newText": "\n    consume(extracted);"},
				},
			}}},
		}
		return []map[string]any{action}, true
	}
	editURI, version, initializer := params.TextDocument.URI, 1, "let extracted = 1_i32 + 2_i32;\n    "
	if mode == "foreign" {
		editURI = "file:///foreign.rs"
	}
	if mode == "stale" {
		version = 7
	}
	if mode == "malformed" {
		initializer = "let extracted = wrap(1_i32 + 2_i32);\n    "
	}
	insertRange, selectedRange := fakeRangeAt(1, 4, 4), fakeRangeAt(1, 12, 25)
	if mode == "method" {
		initializer = "let extracted = 1_i32 + 2_i32;\n        "
		insertRange, selectedRange = fakeRangeAt(4, 8, 8), fakeRangeAt(4, 16, 29)
	}
	if mode == "misplaced" {
		insertRange = fakeRangeAt(0, 0, 0)
	}
	if mode == "batch" && state.rustActionCount == 2 {
		initializer = "let extracted = 4_i32 + 5_i32;\n    "
		insertRange, selectedRange = fakeRangeAt(3, 4, 4), fakeRangeAt(3, 12, 25)
	}
	edit := fakeRustExtractWorkspaceEdit(editURI, version, insertRange, selectedRange, initializer)
	action := map[string]any{
		"title": "Extract into variable", "kind": "refactor.extract",
		"data": map[string]any{"id": "extract_variable:RefactorExtract:0"},
	}
	if mode == "disabled-initial" {
		action["disabled"] = map[string]any{"reason": "selection is not extractable"}
	}
	if mode != "lazy" && mode != "disabled-resolved" {
		action["edit"] = edit
	}
	return []map[string]any{action}, true
}

func fakeRustExtractConstantAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Range fakeLSPRange `json:"range"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Rust extract-constant action request"))
	}
	state.rustURI = params.TextDocument.URI
	state.rustSelection = params.Range
	return []map[string]any{{
		"title": "Extract into constant", "kind": "refactor.extract",
		"data": map[string]any{"id": "extract_constant:RefactorExtract:1:", "version": 1},
	}}, true
}

func fakeRustResolveExtractConstantAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var action map[string]any
	if err := json.Unmarshal(raw, &action); err != nil || action["title"] != "Extract into constant" || action["kind"] != "refactor.extract" {
		failFakeLanguageServer(errors.New("decode Rust extract-constant resolve action"))
	}
	data, ok := action["data"].(map[string]any)
	if !ok || data["id"] != "extract_constant:RefactorExtract:1:" || data["version"] != float64(1) {
		failFakeLanguageServer(errors.New("Rust extract-constant resolve identity changed"))
	}
	delete(action, "data")
	start := fakeLSPByteOffset(state.rustSource, state.rustSelection.Start)
	end := fakeLSPByteOffset(state.rustSource, state.rustSelection.End)
	if start < 0 || end <= start || state.rustSource[start:end] != "1_i32 + 2_i32" {
		failFakeLanguageServer(errors.New("Rust extract-constant fixture selected an unexpected expression"))
	}
	lineStart := strings.LastIndex(state.rustSource[:start], "\n") + 1
	line := state.rustSource[lineStart:]
	indentLength := len(line) - len(strings.TrimLeft(line, " \t"))
	indent := line[:indentLength]
	action["edit"] = map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": state.rustURI, "version": 1},
		"edits": []any{
			map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, lineStart+indentLength, lineStart+indentLength+len("let")), "newText": "const"},
			map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, lineStart+indentLength+len("let "), lineStart+indentLength+len("let total")), "newText": "EXTRACTED:"},
			map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, lineStart+indentLength+len("let total")+1, lineStart+indentLength+len("let total")+1), "newText": "i32 "},
			map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, fakeLSPByteOffset(state.rustSource, fakeLSPPosition{Line: state.rustSelection.End.Line + 1, Character: indentLength}), fakeLSPByteOffset(state.rustSource, fakeLSPPosition{Line: state.rustSelection.End.Line + 1, Character: indentLength})), "newText": "let total = EXTRACTED;\n" + indent},
		},
	}}}
	return action, true
}

func fakeRustPromoteLocalToConstAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Range fakeLSPRange `json:"range"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Rust promote-local-to-const action request"))
	}
	state.rustURI = params.TextDocument.URI
	state.rustSelection = params.Range
	state.rustActionCount++
	return []map[string]any{{
		"title": "Promote local to constant", "kind": "refactor",
		"data": map[string]any{"id": "promote_local_to_const:Refactor:" + strconv.Itoa(state.rustActionCount) + ":"},
	}}, true
}

func fakeRustResolvePromoteLocalToConstAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var action map[string]any
	if err := json.Unmarshal(raw, &action); err != nil || action["title"] != "Promote local to constant" || action["kind"] != "refactor" {
		failFakeLanguageServer(errors.New("decode Rust promote-local-to-const resolve action"))
	}
	data, ok := action["data"].(map[string]any)
	identity := ""
	if ok {
		identity = fmt.Sprint(data["id"])
	}
	if !ok || !strings.HasPrefix(identity, "promote_local_to_const:Refactor:") || !strings.HasSuffix(identity, ":") {
		failFakeLanguageServer(errors.New("Rust promote-local-to-const resolve identity changed"))
	}
	delete(action, "data")
	start := fakeLSPByteOffset(state.rustSource, state.rustSelection.Start)
	end := fakeLSPByteOffset(state.rustSource, state.rustSelection.End)
	if start < 0 || end <= start || end > len(state.rustSource) {
		failFakeLanguageServer(errors.New("Rust promote-local-to-const fixture selection is invalid"))
	}
	name := state.rustSource[start:end]
	declarationLineStart := strings.LastIndex(state.rustSource[:start], "\n") + 1
	declarationLineEnd := strings.IndexByte(state.rustSource[start:], '\n')
	if declarationLineEnd < 0 {
		failFakeLanguageServer(errors.New("Rust promote-local-to-const fixture declaration has no newline"))
	}
	declaration := state.rustSource[declarationLineStart : start+declarationLineEnd]
	letOffset := strings.Index(declaration, "let ")
	if letOffset < 0 || !strings.HasPrefix(declaration[letOffset+4:], name+":") {
		failFakeLanguageServer(errors.New("Rust promote-local-to-const selection is not a local binding"))
	}
	callLineStart := start + declarationLineEnd + 1
	callLineEnd := strings.IndexByte(state.rustSource[callLineStart:], '\n')
	if callLineEnd < 0 {
		callLineEnd = len(state.rustSource) - callLineStart
	}
	callOffsetRelative := strings.Index(state.rustSource[callLineStart:callLineStart+callLineEnd], name)
	callOffset := callLineStart + callOffsetRelative
	if callOffsetRelative < 0 {
		failFakeLanguageServer(errors.New("Rust promote-local-to-const fixture call use is missing"))
	}
	generated := strings.ToUpper(name)
	text := "const"
	mode := os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST")
	if mode == "malformed" || mode == "batch-malformed-second" && state.rustActionCount == 2 {
		text = "static"
	}
	version := any(1)
	if mode == "stale" {
		version = 2
	}
	action["edit"] = map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": state.rustURI, "version": version},
		"edits": []any{
			map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, declarationLineStart+letOffset, declarationLineStart+letOffset+3), "newText": text},
			map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, start, end), "newText": generated},
			map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, callOffset, callOffset+len(name)), "newText": generated},
		},
	}}}
	return action, true
}

func fakeRustExtractCallableAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Range fakeLSPRange `json:"range"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode Rust callable code action request: %w", err))
	}
	state.rustURI = params.TextDocument.URI
	state.rustSelection = params.Range
	state.rustActionCount++
	if target := os.Getenv("SEMEDIT_RUST_CREATE_TARGET_ON_CODEACTION"); target != "" {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			failFakeLanguageServer(fmt.Errorf("create Rust target directory during code action: %w", err))
		}
		if err := os.WriteFile(target, []byte("include!(\"../lib.rs\");\nfn extracted() {}\nfn main() { run(); }\n"), 0o600); err != nil {
			failFakeLanguageServer(fmt.Errorf("create Rust target during code action: %w", err))
		}
	}
	return []map[string]any{{
		"title": "Extract into function", "kind": "refactor.extract",
		"data": map[string]any{"id": "extract_function:RefactorExtract:3:"},
	}}, true
}

func fakeRustResolveExtractCallableAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var action map[string]any
	if err := json.Unmarshal(raw, &action); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode Rust callable resolve action: %w", err))
	}
	mode := os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE")
	if strings.HasPrefix(mode, "multi") {
		start, end := state.rustSelection.Start, state.rustSelection.End
		selected := strings.Split(state.rustSource, "\n")
		if start.Line >= len(selected) || end.Line >= len(selected) || start.Character != 4 || end.Character != len(selected[end.Line]) {
			failFakeLanguageServer(errors.New("Rust multi-call selection is outside the selected direct statements"))
		}
		firstCall := selected[start.Line]
		helperBody := strings.Join(selected[start.Line:end.Line+1], "\n")
		insertLine := strings.Index(state.rustSource, "\n}\n")
		if insertLine < 0 {
			failFakeLanguageServer(errors.New("Rust multi-call owner close was not found"))
		}
		insertLine = strings.Count(state.rustSource[:insertLine+1], "\n") + 1
		ownerEnd := fakeLSPPosition{Line: start.Line + 2, Character: 1}
		preservedSuffix := "\n}"
		if mode == "multi-corrupt-suffix" {
			preservedSuffix += " "
		}
		action["edit"] = map[string]any{"documentChanges": []any{map[string]any{
			"textDocument": map[string]any{"uri": state.rustURI, "version": 1},
			"edits": []any{
				map[string]any{"range": fakeRangeAt(start.Line, 4, 11), "newText": "fun_name"},
				map[string]any{"range": fakeRangeAt(start.Line, 12, len(firstCall)-1), "newText": ")"},
				map[string]any{"range": fakeLSPRange{Start: fakeLSPPosition{Line: start.Line, Character: len(firstCall)}, End: ownerEnd}, "newText": preservedSuffix},
				map[string]any{"range": fakeRangeAt(insertLine, 0, 0), "newText": fmt.Sprintf("\nfn fun_name() {\n%s\n}\n", helperBody)},
			},
		}}}
		return action, true
	}
	callee := fakeRustCallableCallee(state.rustSource)
	calleeEnd := 4 + len(callee)
	argumentStart := calleeEnd + 1
	action["edit"] = map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": state.rustURI, "version": 1},
		"edits": []any{
			map[string]any{"range": fakeRangeAt(1, 4, calleeEnd), "newText": "fun_name"},
			map[string]any{"range": fakeRangeAt(1, argumentStart, argumentStart+len("1_i32 + 2_i32")), "newText": ""},
			map[string]any{"range": fakeRangeAt(3, 0, 0), "newText": fmt.Sprintf("\nfn fun_name() {\n    %s(1_i32 + 2_i32);\n}\n", callee)},
		},
	}}}
	return action, true
}

func fakeRustCallableCallee(source string) string {
	if strings.Contains(source, "fn modify(") {
		return "modify"
	}
	return "consume"
}

func fakeRustFunctionSymbols(source string) []fakeDocumentSymbol {
	lines := strings.Split(source, "\n")
	var symbols []fakeDocumentSymbol
	for startLine, line := range lines {
		declaration := strings.TrimSpace(line)
		if !strings.HasPrefix(declaration, "fn ") {
			continue
		}
		name, _, ok := strings.Cut(strings.TrimPrefix(declaration, "fn "), "(")
		if !ok || name == "" {
			continue
		}
		depth, opened, endLine, endColumn := 0, false, startLine, 0
	functionRange:
		for currentLine := startLine; currentLine < len(lines); currentLine++ {
			for column, char := range lines[currentLine] {
				switch char {
				case '{':
					depth++
					opened = true
				case '}':
					depth--
					if opened && depth == 0 {
						endLine, endColumn = currentLine, column+1
						break functionRange
					}
				}
			}
		}
		if !opened || endColumn == 0 {
			continue
		}
		startColumn := strings.Index(line, "fn ")
		symbols = append(symbols, fakeDocumentSymbol{Name: name, Kind: 12,
			Range:          fakeLSPRange{Start: fakeLSPPosition{Line: startLine, Character: startColumn}, End: fakeLSPPosition{Line: endLine, Character: endColumn}},
			SelectionRange: fakeRangeAt(startLine, startColumn+3, startColumn+3+len(name))})
	}
	return symbols
}

func fakeRustCallableRename(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var params struct {
		NewName string `json:"newName"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.NewName == "" {
		failFakeLanguageServer(errors.New("decode Rust callable rename"))
	}
	useRange := fakeRangeAt(1, 4, 4+len("fun_name"))
	bindingRange := fakeRustGeneratedBindingRange(state.rustSource)
	return fakeRustRenameWorkspaceEdit(state.rustURI, []any{
		map[string]any{"range": useRange, "newText": params.NewName},
		map[string]any{"range": bindingRange, "newText": params.NewName},
	}), true
}

func fakeRustGeneratedBindingRange(source string) fakeLSPRange {
	for line, content := range strings.Split(source, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(content), "fn fun_name()") {
			continue
		}
		start := strings.Index(content, "fun_name")
		return fakeRangeAt(line, start, start+len("fun_name"))
	}
	failFakeLanguageServer(errors.New("generated Rust helper binding not found"))
	return fakeLSPRange{}
}

func fakeRustExtractWorkspaceEdit(uri string, version int, insertionRange, selectionRange any, initializer string) map[string]any {
	return map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": version},
		"edits": []any{
			map[string]any{"range": insertionRange, "newText": initializer},
			map[string]any{"range": selectionRange, "newText": "extracted"},
		},
	}}}
}

func fakeRustInlineAction(state *fakeLanguageServerState, raw json.RawMessage) []map[string]any {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Range fakeLSPRange `json:"range"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Rust inline code action request"))
	}
	lines := strings.Split(state.rustSource, "\n")
	declarationLine := params.Range.Start.Line
	if declarationLine < 0 || declarationLine+1 >= len(lines) {
		failFakeLanguageServer(errors.New("Rust inline selection is outside the opened source"))
	}
	declaration := strings.TrimSpace(lines[declarationLine])
	initializer, ok := strings.CutPrefix(declaration, "let ")
	if !ok {
		failFakeLanguageServer(errors.New("Rust inline selection does not identify a let binding"))
	}
	name, value, ok := strings.Cut(initializer, " = ")
	if !ok || !strings.HasSuffix(value, ";") {
		failFakeLanguageServer(errors.New("Rust inline fixture has an unsupported binding"))
	}
	value = strings.TrimSuffix(value, ";")
	callLine := lines[declarationLine+1]
	call := strings.TrimSpace(callLine)
	open := strings.IndexByte(call, '(')
	close := strings.LastIndex(call, ");")
	if open < 0 || close < open {
		failFakeLanguageServer(errors.New("Rust inline fixture has an unsupported call"))
	}
	argsStart := strings.Index(callLine, "(") + 1
	args := callLine[argsStart : argsStart+close-open-1]
	var edits []any
	edits = append(edits, map[string]any{"range": fakeLSPRange{Start: fakeLSPPosition{Line: declarationLine}, End: fakeLSPPosition{Line: declarationLine + 1}}, "newText": ""})
	argumentStart := argsStart
	for argument := range strings.SplitSeq(args, ",") {
		leading := len(argument) - len(strings.TrimLeft(argument, " \t"))
		trimmed := strings.TrimSpace(argument)
		if trimmed == name {
			start := argumentStart + leading
			edits = append(edits, map[string]any{"range": fakeRangeAt(declarationLine+1, start, start+len(name)), "newText": value})
		}
		argumentStart += len(argument) + 1
	}
	return []map[string]any{{
		"title": "Inline variable", "kind": "refactor.inline",
		"data": map[string]any{"id": "inline_local_variable"},
		"edit": map[string]any{"documentChanges": []any{map[string]any{
			"textDocument": map[string]any{"uri": params.TextDocument.URI, "version": 1},
			"edits":        edits,
		}}},
	}}
}

func fakeRustInlineCallableAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Range fakeLSPRange `json:"range"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Rust inline-call code action request"))
	}
	state.rustURI = params.TextDocument.URI
	state.rustSelection = params.Range
	start, end := fakeLSPByteOffset(state.rustSource, params.Range.Start), fakeLSPByteOffset(state.rustSource, params.Range.End)
	if start < 0 || end <= start || !isFakeIdentifier(state.rustSource[start:end]) {
		failFakeLanguageServer(errors.New("Rust inline-call selection does not identify a function name"))
	}
	return []map[string]any{{
		"title": "Inline `" + state.rustSource[start:end] + "`", "kind": "refactor.inline",
		"data": map[string]any{"id": "inline_call:RefactorInline:0:"},
	}}, true
}

func fakeRustResolveInlineCallableAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var action map[string]any
	if err := json.Unmarshal(raw, &action); err != nil || action["kind"] != "refactor.inline" {
		failFakeLanguageServer(errors.New("decode Rust inline-call resolve action"))
	}
	title, ok := action["title"].(string)
	callee, okName := strings.CutPrefix(title, "Inline `")
	callee, okSuffix := strings.CutSuffix(callee, "`")
	if !ok || !okName || !okSuffix || callee == "" {
		failFakeLanguageServer(errors.New("decode Rust inline-call action title"))
	}
	data, ok := action["data"].(map[string]any)
	if !ok || data["id"] != "inline_call:RefactorInline:0:" {
		failFakeLanguageServer(errors.New("Rust inline-call resolve identity changed"))
	}
	start, end := fakeLSPByteOffset(state.rustSource, state.rustSelection.Start), fakeLSPByteOffset(state.rustSource, state.rustSelection.End)
	callEnd := end + 2
	if start < 0 || callEnd > len(state.rustSource) || state.rustSource[end:callEnd] != "()" {
		failFakeLanguageServer(errors.New("Rust inline-call selection is not a direct zero-argument call"))
	}
	literal := "7_i32"
	calleeDeclaration := "fn " + callee + "() -> i32 { " + literal + " }"
	if !strings.Contains(state.rustSource, calleeDeclaration) {
		failFakeLanguageServer(errors.New("Rust inline-call callee literal was not found"))
	}
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") == "corrupt" {
		literal = "8_i32"
	}
	version := 1
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") == "stale" {
		version = 2
	}
	uri := state.rustURI
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") == "foreign" {
		uri = "file:///foreign.rs"
	}
	return map[string]any{
		"title": action["title"], "kind": action["kind"],
		"edit": map[string]any{"documentChanges": []any{map[string]any{
			"textDocument": map[string]any{"uri": uri, "version": version},
			"edits":        []any{map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, start, callEnd), "newText": literal}},
		}}},
	}, true
}

func isFakeIdentifier(value string) bool {
	if len(value) == 0 || !fakeIdentifierStart(value[0]) {
		return false
	}
	for _, char := range []byte(value[1:]) {
		if !fakeIdentifierStart(char) && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func fakeIdentifierStart(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func fakeIdentifierContinue(value byte) bool {
	return fakeIdentifierStart(value) || value >= '0' && value <= '9'
}

func fakeRustResolveExtractAction(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var action map[string]any
	if err := json.Unmarshal(raw, &action); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode rust lazy code action: %w", err))
	}
	action["edit"] = fakeRustExtractWorkspaceEdit(state.rustURI, 1, fakeRangeAt(1, 4, 4), fakeRangeAt(1, 12, 25), "let extracted = 1_i32 + 2_i32;\n    ")
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") == "disabled-resolved" {
		action["disabled"] = map[string]any{"reason": "selection is no longer extractable"}
	}
	return action, true
}

func fakeRustRename(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" {
		return fakeRustCallableRename(state, raw)
	}
	var params struct {
		NewName string `json:"newName"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.NewName == "" {
		failFakeLanguageServer(errors.New("decode rust extraction rename"))
	}
	mode := os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE")
	if mode == "method" {
		return fakeRustRenameWorkspaceEdit(fakeTextDocumentParams(raw).URI, []any{
			map[string]any{"range": fakeRangeAt(4, 12, 21), "newText": params.NewName},
			map[string]any{"range": fakeRangeAt(5, 16, 25), "newText": params.NewName},
		}), true
	}
	if mode == "batch" {
		first, second := []any{fakeRangeAt(1, 8, 17), fakeRangeAt(2, 12, 21)}, []any{fakeRangeAt(3, 8, 17), fakeRangeAt(4, 12, 21)}
		if state.rustActionCount == 2 {
			first = second
		}
		return fakeRustRenameWorkspaceEdit(state.rustURI, []any{
			map[string]any{"range": first[0], "newText": params.NewName},
			map[string]any{"range": first[1], "newText": params.NewName},
		}), true
	}
	return fakeRustRenameWorkspaceEdit(fakeTextDocumentParams(raw).URI, []any{
		map[string]any{"range": fakeRangeAt(1, 8, 17), "newText": params.NewName},
		map[string]any{"range": fakeRangeAt(2, 12, 21), "newText": params.NewName},
	}), true
}

func fakeRustConstantRename(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var params struct {
		NewName      string `json:"newName"`
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position fakeLSPPosition `json:"position"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.NewName == "" || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Rust constant rename"))
	}
	start := fakeLSPByteOffset(state.rustSource, params.Position)
	if start < 0 || start >= len(state.rustSource) {
		failFakeLanguageServer(errors.New("Rust constant rename position is outside source"))
	}
	end := start
	for end < len(state.rustSource) && fakeIdentifierContinue(state.rustSource[end]) {
		end++
	}
	oldName := state.rustSource[start:end]
	var edits []any
	for offset := 0; offset < len(state.rustSource); {
		index := strings.Index(state.rustSource[offset:], oldName)
		if index < 0 {
			break
		}
		index += offset
		finish := index + len(oldName)
		beforeOK := index == 0 || !fakeIdentifierContinue(state.rustSource[index-1])
		afterOK := finish == len(state.rustSource) || !fakeIdentifierContinue(state.rustSource[finish])
		if beforeOK && afterOK {
			edits = append(edits, map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, index, finish), "newText": params.NewName})
		}
		offset = finish
	}
	return fakeRustRenameWorkspaceEdit(params.TextDocument.URI, edits), true
}

func fakeRustInlineCallableRename(state *fakeLanguageServerState, raw json.RawMessage) (any, bool) {
	var params struct {
		NewName      string `json:"newName"`
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position fakeLSPPosition `json:"position"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.NewName == "" || params.TextDocument.URI == "" {
		failFakeLanguageServer(errors.New("decode Rust inline-call rename"))
	}
	start := fakeLSPByteOffset(state.rustSource, params.Position)
	if start < 0 || start >= len(state.rustSource) {
		failFakeLanguageServer(errors.New("Rust inline-call rename position is outside source"))
	}
	end := start
	for end < len(state.rustSource) && fakeIdentifierContinue(state.rustSource[end]) {
		end++
	}
	oldName := state.rustSource[start:end]
	var edits []any
	for offset := 0; offset < len(state.rustSource); {
		index := strings.Index(state.rustSource[offset:], oldName)
		if index < 0 {
			break
		}
		index += offset
		finish := index + len(oldName)
		beforeOK := index == 0 || !fakeIdentifierContinue(state.rustSource[index-1])
		afterOK := finish == len(state.rustSource) || !fakeIdentifierContinue(state.rustSource[finish])
		if beforeOK && afterOK {
			edits = append(edits, map[string]any{"range": fakeLSPRangeAtOffsets(state.rustSource, index, finish), "newText": params.NewName})
		}
		offset = finish
	}
	return map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": params.TextDocument.URI, "version": 1},
		"edits":        edits,
	}}}, true
}

func fakeRustRenameWorkspaceEdit(uri string, edits []any) map[string]any {
	return map[string]any{"documentChanges": []any{map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "edits": edits}}}
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
	if os.Getenv("SEMEDIT_TEST_SCALA_INLINE_ASSERT_APPLY") != "" {
		var init struct {
			Capabilities struct {
				Workspace struct {
					ApplyEdit bool `json:"applyEdit"`
				} `json:"workspace"`
			} `json:"capabilities"`
		}
		if err := json.Unmarshal(params, &init); err != nil || !init.Capabilities.Workspace.ApplyEdit {
			return nil, false
		}
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") != "" || os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST") != "" {
		var init struct {
			Capabilities struct {
				TextDocument struct {
					DocumentSymbol struct {
						Hierarchical bool `json:"hierarchicalDocumentSymbolSupport"`
					} `json:"documentSymbol"`
				} `json:"textDocument"`
			} `json:"capabilities"`
		}
		if err := json.Unmarshal(params, &init); err != nil || !init.Capabilities.TextDocument.DocumentSymbol.Hierarchical {
			return nil, false
		}
	}
	if !validConfiguredJavaSettings(params) {
		return nil, false
	}
	return map[string]any{"capabilities": map[string]any{}}, true
}

//nolint:gocognit,funlen // This fixture decodes the supported LSP document-symbol wire shapes in one place.
func fakeDocumentSymbols(state *fakeLanguageServerState, raw json.RawMessage) []fakeDocumentSymbol {
	if os.Getenv("SEMEDIT_TEST_JAVA_EXPLICIT_TYPE") != "" {
		return fakeJavaExplicitTypeSymbols(state, raw)
	}
	symbol := state.symbol
	codeActionSeen := state.codeActionSeen
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_VALUE") != "" {
		lines := strings.Split(state.rustSource, "\n")
		endLine := len(lines)
		for i, line := range lines {
			if strings.TrimSpace(line) == "}" {
				endLine = i + 1
				break
			}
		}
		return []fakeDocumentSymbol{{Name: "run", Kind: 12, Range: fakeLSPRange{Start: fakeLSPPosition{Line: 0}, End: fakeLSPPosition{Line: endLine, Character: 0}}, SelectionRange: fakeRangeAt(0, 3, 6)}}
	}
	if os.Getenv("SEMEDIT_TEST_RUST_INLINE_CALLABLE") != "" {
		return fakeRustFunctionSymbols(state.rustSource)
	}
	if os.Getenv("SEMEDIT_TEST_RUST_PROMOTE_LOCAL_TO_CONST") != "" {
		return fakeRustFunctionSymbols(state.rustSource)
	}
	if os.Getenv("SEMEDIT_TEST_SCALA_EXTRACT_CALLABLE") != "" {
		var document struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(raw, &document); err == nil && strings.Contains(state.documents[document.TextDocument.URI], "def logWork(): Unit") {
			return []fakeDocumentSymbol{{Name: "logWork", Kind: 6,
				Range:          fakeLSPRange{Start: fakeLSPPosition{Line: 2}, End: fakeLSPPosition{Line: 3, Character: 20}},
				SelectionRange: fakeRangeAt(2, 8, 14)}}
		}
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_CALLABLE") != "" {
		return fakeRustFunctionSymbols(state.rustSource)
	}
	if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") != "" {
		if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") == "module-owner" {
			return []fakeDocumentSymbol{{Name: "run", Kind: 2,
				Range:          fakeLSPRange{Start: fakeLSPPosition{Line: 0}, End: fakeLSPPosition{Line: 4, Character: 1}},
				SelectionRange: fakeRangeAt(0, 4, 7)}}
		}
		if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") == "method" {
			return []fakeDocumentSymbol{{Name: "run", Kind: 6,
				Range:          fakeLSPRange{Start: fakeLSPPosition{Line: 3, Character: 4}, End: fakeLSPPosition{Line: 5, Character: 5}},
				SelectionRange: fakeRangeAt(3, 7, 10)}}
		}
		name := "run"
		if os.Getenv("SEMEDIT_TEST_RUST_EXTRACT_VARIABLE") == "batch" && state.rustRenamed {
			name = "execute"
		}
		endLine := 2
		if closeOffset := strings.Index(state.rustSource, "\n}"); closeOffset >= 0 {
			endLine = strings.Count(state.rustSource[:closeOffset+2], "\n")
		}
		return []fakeDocumentSymbol{{Name: name, Kind: 12,
			Range:          fakeLSPRange{Start: fakeLSPPosition{Line: 0}, End: fakeLSPPosition{Line: endLine, Character: 1}},
			SelectionRange: fakeRangeAt(0, 3, 6)}}
	}
	if os.Getenv("SEMEDIT_TEST_RUST_AMBIGUOUS") != "" {
		return []fakeDocumentSymbol{
			{Name: "first", Kind: 2, Range: fakeLSPRange{Start: fakeLSPPosition{Line: 0}, End: fakeLSPPosition{Line: 2, Character: 1}}, SelectionRange: fakeRangeAt(0, 4, 9),
				Children: []fakeDocumentSymbol{{Name: "Widget", Kind: 23, Range: fakeRangeAt(1, 4, 21), SelectionRange: fakeRangeAt(1, 15, 21)}}},
			{Name: "second", Kind: 2, Range: fakeLSPRange{Start: fakeLSPPosition{Line: 3}, End: fakeLSPPosition{Line: 5, Character: 1}}, SelectionRange: fakeRangeAt(3, 4, 10),
				Children: []fakeDocumentSymbol{{Name: "Widget", Kind: 23, Range: fakeRangeAt(4, 4, 21), SelectionRange: fakeRangeAt(4, 15, 21)}}},
		}
	}
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
	if os.Getenv("SEMEDIT_TEST_JAVA_MAKE_STATIC_OVERLOAD") != "" {
		return []fakeDocumentSymbol{{Name: "Service", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 4}}, SelectionRange: fakeRangeAt(1, 6, 13),
			Children: []fakeDocumentSymbol{
				{Name: "count", Kind: 6, Detail: "int count()", Range: fakeRangeAt(2, 2, 27), SelectionRange: fakeRangeAt(2, 6, 11)},
				{Name: "count", Kind: 6, Detail: "int count(int n)", Range: fakeRangeAt(3, 2, 32), SelectionRange: fakeRangeAt(3, 6, 11)},
			}}}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_MAKE_STATIC_SECOND_CLASS") != "" {
		makeClass := func(name string, startLine, methodLine, endLine int) fakeDocumentSymbol {
			return fakeDocumentSymbol{Name: name, Kind: 5,
				Range: fakeLSPRange{Start: fakeLSPPosition{Line: startLine}, End: fakeLSPPosition{Line: endLine}}, SelectionRange: fakeRangeAt(startLine, 6, 6+len(name)),
				Children: []fakeDocumentSymbol{{Name: "count", Kind: 6, Detail: "int count()",
					Range: fakeRangeAt(methodLine, 2, 27), SelectionRange: fakeRangeAt(methodLine, 6, 11)}}}
		}
		return []fakeDocumentSymbol{makeClass("Service", 1, 2, 3), makeClass("Other", 4, 5, 6)}
	}
	if os.Getenv("SEMEDIT_TEST_JAVA_MAKE_STATIC") != "" {
		document := fakeTextDocumentParams(raw)
		staticDetail := "int count()"
		if strings.Contains(state.documents[document.URI], "static int count(Service receiver)") {
			staticDetail = "static int count(Service receiver)"
		}
		return []fakeDocumentSymbol{{Name: "Service", Kind: 5,
			Range: fakeLSPRange{Start: fakeLSPPosition{Line: 1}, End: fakeLSPPosition{Line: 4}}, SelectionRange: fakeRangeAt(1, 6, 13),
			Children: []fakeDocumentSymbol{{Name: "count", Kind: 6, Detail: staticDetail,
				Range: fakeRangeAt(2, 2, 27), SelectionRange: fakeRangeAt(2, 6, 11)}}}}
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
			edit := map[string]any{"changes": map[string]any{params.TextDocument.URI: []map[string]any{{"range": fakeRangeAt(3, 11, 25), "newText": "41"}}}}
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
	if os.Getenv("SEMEDIT_TEST_JAVA_CONSTRUCTS") != "" || os.Getenv("SEMEDIT_JAVA_ORGANIZE") != "" {
		return []map[string]any{}
	}
	document := fakeTextDocumentParams(raw)
	edit := map[string]any{"changes": map[string]any{document.URI: []map[string]any{{"range": fakeLSPRange{}, "newText": ""}}}}
	return []map[string]any{{"kind": "source.organizeImports", "edit": edit}}
}

func fakeJavaMakeStaticAction(raw json.RawMessage) []map[string]any {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Context struct {
			Only []string `json:"only"`
		} `json:"context"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		failFakeLanguageServer(fmt.Errorf("decode Java make-static code action: %w", err))
	}
	if params.TextDocument.URI == "" || len(params.Context.Only) != 1 {
		return []map[string]any{}
	}
	switch params.Context.Only[0] {
	case "refactor.inline":
		if os.Getenv("SEMEDIT_TEST_JAVA_MAKE_STATIC_REFUSAL") != "" {
			return []map[string]any{{"title": "Make Static", "kind": "refactor.inline", "disabled": map[string]string{"reason": "not applicable"}}}
		}
		callerURI := strings.TrimSuffix(params.TextDocument.URI, "Service.java") + "Caller.java"
		edit := map[string]any{"changes": map[string]any{
			params.TextDocument.URI: []map[string]any{{
				"range": fakeRangeAt(2, 2, 27), "newText": "static int count(Service receiver) { return 1; }",
			}},
			callerURI: []map[string]any{{
				"range": fakeRangeAt(2, 0, 36), "newText": "  int value = Service.count(new Service());",
			}},
		}}
		return []map[string]any{{"title": "Make Static", "kind": "refactor.inline", "edit": edit}}
	case "source.organizeImports":
		edit := map[string]any{"changes": map[string]any{params.TextDocument.URI: []any{}}}
		return []map[string]any{{"title": "Organize Imports", "kind": "source.organizeImports", "edit": edit}}
	default:
		return []map[string]any{}
	}
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
