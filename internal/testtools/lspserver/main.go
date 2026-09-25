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
	ambiguous := false
	included := false
	conditional := false
	functionDeclarations := false
	for {
		headers := map[string]string{}
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			k, v, ok := strings.Cut(line, ":")
			if ok {
				headers[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
			}
		}
		size := 0
		if _, err := fmt.Sscanf(headers["content-length"], "%d", &size); err != nil || size < 0 {
			return fmt.Errorf("bad content length")
		}
		data := make([]byte, size)
		if _, err := io.ReadFull(reader, data); err != nil {
			return err
		}
		var m message
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		if m.ID == nil {
			if m.Method == "textDocument/didOpen" {
				var p struct {
					TextDocument struct {
						Text string `json:"text"`
					} `json:"textDocument"`
				}
				_ = json.Unmarshal(m.Params, &p)
				ambiguous = strings.Contains(p.TextDocument.Text, "SEMEDIT_TEST_AMBIGUOUS")
				included = strings.Contains(p.TextDocument.Text, "include ")
				conditional = strings.Contains(p.TextDocument.Text, "ifeq (")
				functionDeclarations = strings.Contains(p.TextDocument.Text, "function fun()")
			}
			if server == "bash-language-server" && m.Method == "textDocument/didOpen" {
				var p struct {
					TextDocument struct {
						URI string `json:"uri"`
					} `json:"textDocument"`
				}
				_ = json.Unmarshal(m.Params, &p)
				if err := write(writer, message{JSONRPC: "2.0", Method: "textDocument/publishDiagnostics", Params: mustJSON(map[string]any{"uri": p.TextDocument.URI, "version": 1, "diagnostics": []any{}})}); err != nil {
					return err
				}
			}
			continue
		}
		var result any
		switch m.Method {
		case "initialize":
			result = map[string]any{"capabilities": map[string]any{"positionEncoding": "utf-16", "documentSymbolProvider": true}}
		case "textDocument/documentSymbol":
			if server == "bash-language-server" {
				sym := map[string]any{"name": "f", "kind": 12, "location": map[string]any{"uri": "selected", "range": rng(1, 0, 6)}}
				result = []any{sym}
				if functionDeclarations {
					result = []any{
						map[string]any{"name": "fun", "kind": 12, "location": map[string]any{"uri": "selected", "range": rng(0, 0, 16)}},
						map[string]any{"name": "loc", "kind": 13, "location": map[string]any{"uri": "selected", "range": rng(1, 1, 14)}},
					}
				}
				if ambiguous {
					result = []any{sym, sym}
				}
			} else {
				sym := map[string]any{"name": "all", "kind": 12, "range": rng(0, 0, 4), "selectionRange": rng(0, 0, 3)}
				result = []any{sym}
				if conditional {
					wholeBlock := map[string]any{"start": map[string]int{"line": 0, "character": 0}, "end": map[string]int{"line": 3, "character": 5}}
					result = []any{map[string]any{"name": "ifeq $(MODE),1", "kind": 3, "range": wholeBlock, "selectionRange": wholeBlock}}
				}
				if included {
					result = []any{map[string]any{"name": "external", "kind": 12, "range": rng(0, 0, 19), "selectionRange": rng(0, 8, 16)}}
				}
				if ambiguous {
					result = []any{sym, sym}
				}
			}
		case "shutdown":
			result = nil
		default:
			result = map[string]any{}
		}
		if err := write(writer, message{JSONRPC: "2.0", ID: m.ID, Result: result}); err != nil {
			return err
		}
		if m.Method == "shutdown" {
			return nil
		}
	}
}
func rng(line, start, end int) map[string]any {
	return map[string]any{"start": map[string]int{"line": line, "character": start}, "end": map[string]int{"line": line, "character": end}}
}
func mustJSON(v any) json.RawMessage { raw, _ := json.Marshal(v); return raw }
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
