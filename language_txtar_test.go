// Package main provides deterministic language-server stand-ins for CLI txtar coverage.
package main

import (
	"encoding/json"
	"errors"
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
	Name           string       `json:"name"`
	Kind           int          `json:"kind"`
	Range          fakeLSPRange `json:"range"`
	SelectionRange fakeLSPRange `json:"selectionRange"`
}

func runFakeRustAnalyzer() {
	symbol := fakeDocumentSymbol{"Widget", 23, fakeRange(0, 15), fakeRange(7, 13)}
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
	runFakeLanguageServer(fakeDocumentSymbol{"Widget", 5, fakeRange(0, 14), fakeRange(6, 12)})
}

func runFakeKotlin() {
	if os.Getenv("SEMEDIT_TEST_KOTLIN_FUNCTION") != "" {
		runFakeLanguageServer(fakeDocumentSymbol{"launch", 12, fakeRangeAt(1, 0, 15), fakeRangeAt(1, 4, 10)})
		return
	}
	runFakeLanguageServer(fakeDocumentSymbol{"Widget", 5, fakeRangeAt(2, 0, 29), fakeRangeAt(2, 6, 12)})
}

func runFakeMetals() {
	runFakeLanguageServer(fakeDocumentSymbol{"Widget", 5, fakeRange(0, 15), fakeRange(7, 13)})
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
	runFakeLanguageServer(fakeDocumentSymbol{"Widget", 5, fakeRange(0, 20), fakeRange(5, 11)})
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
	formattingSeen, codeActionSeen := false, false
	for {
		payload, err := reader.ReadMessage()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			return
		}
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(payload, &request) != nil {
			continue
		}
		if len(request.ID) == 0 {
			if request.Method == "textDocument/didOpen" && (os.Getenv("SEMEDIT_TEST_KOTLIN_VERIFY") != "" || os.Getenv("SEMEDIT_TEST_KOTLIN_WRONG_URI_ONLY") != "") {
				var opened struct {
					TextDocument struct {
						URI string `json:"uri"`
					} `json:"textDocument"`
				}
				_ = json.Unmarshal(request.Params, &opened)
				uri := opened.TextDocument.URI
				if os.Getenv("SEMEDIT_TEST_KOTLIN_WRONG_URI_ONLY") != "" {
					uri = "file:///outside/Widget.kt"
				}
				diagnostics := []any{}
				if os.Getenv("SEMEDIT_TEST_KOTLIN_DIAGNOSTIC") != "" {
					diagnostics = []any{map[string]any{"message": "selected Kotlin warning", "severity": 2, "range": fakeRange(0, 0)}}
				}
				params, _ := json.Marshal(map[string]any{"uri": uri, "diagnostics": diagnostics})
				notification, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/publishDiagnostics", "params": json.RawMessage(params)})
				_ = writer.WriteMessage(notification)
			}
			if request.Method == "textDocument/didChange" {
				var change struct {
					TextDocument struct {
						URI     string `json:"uri"`
						Version int    `json:"version"`
					} `json:"textDocument"`
				}
				_ = json.Unmarshal(request.Params, &change)
				if os.Getenv("SEMEDIT_ASSERT_JAVA_VERIFY") != "" && change.TextDocument.Version > 2 && (!formattingSeen || !codeActionSeen) {
					return
				}
				if change.TextDocument.Version <= 2 {
					continue
				}
				params, _ := json.Marshal(map[string]any{"uri": change.TextDocument.URI, "version": change.TextDocument.Version, "diagnostics": []any{}})
				notification, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/publishDiagnostics", "params": json.RawMessage(params)})
				_ = writer.WriteMessage(notification)
			}
			if request.Method == "workspace/didChangeConfiguration" && os.Getenv("SEMEDIT_ASSERT_JAVA_SETTINGS") != "" && !assertJavaSettings(request.Params, os.Getenv("SEMEDIT_ASSERT_JAVA_SETTINGS")) {
				return
			}
			continue
		}
		var result any
		switch request.Method {
		case "initialize":
			if os.Getenv("SEMEDIT_ASSERT_JAVA_SETTINGS") != "" && !assertJavaSettings(request.Params, os.Getenv("SEMEDIT_ASSERT_JAVA_SETTINGS")) {
				return
			}
			result = map[string]any{"capabilities": map[string]any{}}
		case "workspace/didChangeConfiguration":
			if os.Getenv("SEMEDIT_ASSERT_JAVA_SETTINGS") != "" && !assertJavaSettings(request.Params, os.Getenv("SEMEDIT_ASSERT_JAVA_SETTINGS")) {
				return
			}
		case "textDocument/documentSymbol":
			result = []fakeDocumentSymbol{symbol}
			if os.Getenv("SEMEDIT_TEST_KOTLIN_AMBIGUOUS") != "" {
				result = []fakeDocumentSymbol{symbol, symbol}
			}
		case "textDocument/prepareRename":
			result = symbol.SelectionRange
		case "textDocument/formatting":
			formattingSeen = true
			result = []map[string]any{{"range": fakeLSPRange{Start: fakeLSPPosition{}, End: fakeLSPPosition{Line: 3}}, "newText": "class Widget {}\n"}}
		case "textDocument/codeAction":
			codeActionSeen = true
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			_ = json.Unmarshal(request.Params, &params)
			result = []map[string]any{{"kind": "source.organizeImports", "edit": map[string]any{"changes": map[string]any{params.TextDocument.URI: []map[string]any{{"range": fakeLSPRange{}, "newText": ""}}}}}}
		case "textDocument/rename":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				NewName string `json:"newName"`
			}
			_ = json.Unmarshal(request.Params, &params)
			uri := params.TextDocument.URI
			if parsed, err := url.Parse(uri); err == nil {
				uri = parsed.String()
			}
			edit := map[string]any{"changes": map[string]any{uri: []map[string]any{{"range": symbol.SelectionRange, "newText": params.NewName}}}}
			if os.Getenv("SEMEDIT_RUST_UNSAFE") == "1" || os.Getenv("SEMEDIT_JAVA_UNSAFE") == "1" {
				edit["changes"] = map[string]any{uri: []map[string]any{{"range": symbol.SelectionRange, "newText": params.NewName}}, "file:///foreign.rs": []map[string]any{{"range": symbol.SelectionRange, "newText": params.NewName}}}
			}
			result = edit
		}
		response, err := json.Marshal(struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Result  any             `json:"result"`
		}{JSONRPC: "2.0", ID: request.ID, Result: result})
		if err != nil || writer.WriteMessage(response) != nil {
			return
		}
	}
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
