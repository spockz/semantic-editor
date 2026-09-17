// Package main provides deterministic language-server stand-ins for CLI txtar coverage.
package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"

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
	runFakeLanguageServer(fakeDocumentSymbol{"Widget", 23, fakeRange(0, 15), fakeRange(7, 13)})
}

func runFakeJava() {
	if len(os.Args) > 1 && os.Args[1] == "-version" {
		_, _ = os.Stderr.WriteString("openjdk version \"21.0.2\"\n")
		return
	}
	runFakeLanguageServer(fakeDocumentSymbol{"Widget", 5, fakeRange(0, 14), fakeRange(6, 12)})
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

func fakeRange(start, end int) fakeLSPRange {
	return fakeLSPRange{Start: fakeLSPPosition{Character: start}, End: fakeLSPPosition{Character: end}}
}

func runFakeLanguageServer(symbol fakeDocumentSymbol) {
	reader := lsp.NewFrameReader(os.Stdin, 0)
	writer := lsp.NewFrameWriter(os.Stdout, 0)
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
		}
		if json.Unmarshal(payload, &request) != nil || len(request.ID) == 0 {
			continue
		}
		var result any
		switch request.Method {
		case "initialize":
			result = map[string]any{"capabilities": map[string]any{}}
		case "textDocument/documentSymbol":
			result = []fakeDocumentSymbol{symbol}
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
