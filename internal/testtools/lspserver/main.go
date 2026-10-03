// Package main provides a hermetic stdio LSP peer for public CLI contract tests.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
}

func main() {
	if err := serve(os.Stdin, os.Stdout, filepath.Base(os.Args[0])); err != nil {
		os.Exit(2)
	}
}
func serve(input io.Reader, output io.Writer, server string) error {
	reader := bufio.NewReader(input)
	writer := bufio.NewWriter(output)
	state := serverState{}
	for {
		request, err := readMessage(reader)
		if err != nil {
			return err
		}
		consumed, err := handleNotification(writer, server, request, &state)
		if err != nil {
			return err
		}
		if consumed {
			continue
		}
		result, shutdown, err := handleRequest(server, request, state)
		if err != nil {
			return err
		}
		if err := write(writer, message{JSONRPC: "2.0", ID: request.ID, Result: result}); err != nil {
			return err
		}
		if shutdown {
			return nil
		}
	}
}

type serverState struct {
	ambiguous            bool
	included             bool
	conditional          bool
	functionDeclarations bool
}

func readMessage(reader *bufio.Reader) (message, error) {
	headers, err := readHeaders(reader)
	if err != nil {
		return message{}, err
	}
	size, err := contentLength(headers["content-length"])
	if err != nil {
		return message{}, err
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(reader, data); err != nil {
		return message{}, err
	}
	var request message
	if err := json.Unmarshal(data, &request); err != nil {
		return message{}, err
	}
	return request, nil
}

func readHeaders(reader *bufio.Reader) (map[string]string, error) {
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return headers, nil
		}
		key, value, ok := strings.Cut(line, ":")
		if ok {
			headers[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
		}
	}
}

func contentLength(value string) (int, error) {
	size := 0
	if _, err := fmt.Sscanf(value, "%d", &size); err != nil || size < 0 {
		return 0, fmt.Errorf("bad content length")
	}
	return size, nil
}

func handleNotification(writer *bufio.Writer, server string, request message, state *serverState) (bool, error) {
	if request.ID != nil {
		return false, nil
	}
	if request.Method == "textDocument/didOpen" {
		if err := state.observeOpenedDocument(request.Params); err != nil {
			return true, err
		}
		if server == "bash-language-server" {
			if err := publishEmptyDiagnostics(writer, request.Params); err != nil {
				return true, err
			}
		}
	}
	return true, nil
}

func (state *serverState) observeOpenedDocument(params json.RawMessage) error {
	var opened struct {
		TextDocument struct {
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &opened); err != nil {
		return fmt.Errorf("decode didOpen params: %w", err)
	}
	text := opened.TextDocument.Text
	state.ambiguous = strings.Contains(text, "SEMEDIT_TEST_AMBIGUOUS")
	state.included = strings.Contains(text, "include ")
	state.conditional = strings.Contains(text, "ifeq (")
	state.functionDeclarations = strings.Contains(text, "function fun()")
	return nil
}

func publishEmptyDiagnostics(writer *bufio.Writer, params json.RawMessage) error {
	var opened struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &opened); err != nil {
		return fmt.Errorf("decode didOpen diagnostics params: %w", err)
	}
	if opened.TextDocument.URI == "" {
		return fmt.Errorf("didOpen diagnostics missing document URI")
	}
	result := map[string]any{"uri": opened.TextDocument.URI, "version": 1, "diagnostics": []any{}}
	paramsJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode empty diagnostics: %w", err)
	}
	return write(writer, message{JSONRPC: "2.0", Method: "textDocument/publishDiagnostics", Params: paramsJSON})
}

func handleRequest(server string, request message, state serverState) (any, bool, error) {
	switch request.Method {
	case "initialize":
		return map[string]any{"capabilities": map[string]any{"positionEncoding": "utf-16", "documentSymbolProvider": true}}, false, nil
	case "textDocument/documentSymbol":
		symbols, err := documentSymbols(server, request.Params, state)
		return symbols, false, err
	case "shutdown":
		return nil, true, nil
	default:
		return map[string]any{}, false, nil
	}
}

func documentSymbols(server string, params json.RawMessage, state serverState) (any, error) {
	var document struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &document); err != nil {
		return nil, fmt.Errorf("decode documentSymbol params: %w", err)
	}
	if document.TextDocument.URI == "" {
		return nil, fmt.Errorf("documentSymbol params missing document URI")
	}
	if server == "bash-language-server" {
		return bashDocumentSymbols(document.TextDocument.URI, state), nil
	}
	return makeDocumentSymbols(document.TextDocument.URI, state), nil
}

func bashDocumentSymbols(uri string, state serverState) []any {
	symbol := map[string]any{"name": "f", "kind": 12, "location": map[string]any{"uri": uri, "range": rng(1, 0, 6)}}
	if state.ambiguous {
		return []any{symbol, symbol}
	}
	if state.functionDeclarations {
		return []any{
			map[string]any{"name": "fun", "kind": 12, "location": map[string]any{"uri": uri, "range": rng(0, 0, 16)}},
			map[string]any{"name": "loc", "kind": 13, "location": map[string]any{"uri": uri, "range": rng(1, 1, 14)}},
		}
	}
	return []any{symbol}
}

func makeDocumentSymbols(uri string, state serverState) []any {
	symbol := map[string]any{"name": "all", "kind": 12, "range": rng(0, 0, 4), "selectionRange": rng(0, 0, 3)}
	if strings.HasSuffix(uri, "/Read.mk") {
		symbol["range"] = map[string]any{"start": map[string]int{"line": 0, "character": 0}, "end": map[string]int{"line": 2, "character": 10}}
	}
	if state.ambiguous {
		return []any{symbol, symbol}
	}
	if state.included {
		return []any{makeIncludedSymbol()}
	}
	if state.conditional {
		return []any{makeConditionalSymbol()}
	}
	if strings.HasSuffix(uri, "/Variables.mk") {
		return []any{map[string]any{"name": "X", "kind": 13, "range": rng(0, 0, 5), "selectionRange": rng(0, 0, 1)}}
	}
	return []any{symbol}
}

func makeIncludedSymbol() map[string]any {
	return map[string]any{"name": "external", "kind": 12, "range": rng(0, 0, 19), "selectionRange": rng(0, 8, 16)}
}

func makeConditionalSymbol() map[string]any {
	block := map[string]any{"start": map[string]int{"line": 0, "character": 0}, "end": map[string]int{"line": 3, "character": 5}}
	return map[string]any{"name": "ifeq $(MODE),1", "kind": 3, "range": block, "selectionRange": block}
}
func rng(line, start, end int) map[string]any {
	return map[string]any{"start": map[string]int{"line": line, "character": start}, "end": map[string]int{"line": line, "character": end}}
}
func write(w *bufio.Writer, m message) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	return w.Flush()
}
