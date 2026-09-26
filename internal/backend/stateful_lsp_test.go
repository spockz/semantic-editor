// Package backend_test provides stateful LSP test doubles for document tracking tests.
package backend_test

import (
	"context"
	"encoding/json"
	"errors"
)

type statefulTestLspSession struct {
	calls              []string
	open               map[string]string
	openedTexts        []string
	responses          map[string]json.RawMessage
	closeCount         int
	initializeResponse json.RawMessage
}

func newStatefulTestLspSession(initializeResponse json.RawMessage, responses map[string]json.RawMessage) *statefulTestLspSession {
	return &statefulTestLspSession{
		open:               map[string]string{},
		responses:          responses,
		initializeResponse: initializeResponse,
	}
}

func (f *statefulTestLspSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	f.calls = append(f.calls, method)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if method == "initialize" {
		if f.initializeResponse != nil {
			return f.initializeResponse, nil
		}
		return json.RawMessage("{}"), nil
	}
	var request struct {
		TextDocument struct {
			URI string
		}
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encoded, &request); err != nil {
		return nil, err
	}
	source, ok := f.open[request.TextDocument.URI]
	if !ok {
		return nil, errors.New("documentSymbol requested without an open document")
	}
	response, ok := f.responses[source]
	if !ok {
		return nil, errors.New("no symbol response for opened source")
	}
	return response, nil
}

func (f *statefulTestLspSession) Notify(ctx context.Context, method string, params any) error {
	f.calls = append(f.calls, method)
	if err := ctx.Err(); err != nil {
		return err
	}
	if method != "textDocument/didOpen" && method != "textDocument/didClose" {
		return nil
	}
	var notification struct {
		TextDocument struct {
			URI  string
			Text string
		}
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, &notification); err != nil {
		return err
	}
	if method == "textDocument/didOpen" {
		if _, exists := f.open[notification.TextDocument.URI]; exists {
			return errors.New("document opened twice")
		}
		f.open[notification.TextDocument.URI] = notification.TextDocument.Text
		f.openedTexts = append(f.openedTexts, notification.TextDocument.Text)
		return nil
	}
	if _, exists := f.open[notification.TextDocument.URI]; !exists {
		return errors.New("closed document was not open")
	}
	delete(f.open, notification.TextDocument.URI)
	return nil
}

func (f *statefulTestLspSession) Close() error {
	f.closeCount++
	return nil
}
