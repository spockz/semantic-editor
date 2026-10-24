// Package scala verifies scoped server-request capture over the production LSP client.
// WHY: command success must not hide duplicate or malformed apply-edit callbacks.
package scala

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/lsp"
)

func TestScalaApplyEditCaptureRejectsValidThenMalformedCallback(t *testing.T) {
	clientToPeerR, clientToPeerW := io.Pipe()
	peerToClientR, peerToClientW := io.Pipe()
	t.Cleanup(func() {
		_ = clientToPeerR.Close()
		_ = clientToPeerW.Close()
		_ = peerToClientR.Close()
		_ = peerToClientW.Close()
	})
	session := &scalaProcessSession{}
	session.client = lsp.NewClient(peerToClientR, clientToPeerW, lsp.WithRequestHandler(session.handleServerRequest))
	t.Cleanup(func() { _ = session.Close() })
	peerReader := lsp.NewFrameReader(clientToPeerR, 1<<20)
	peerWriter := lsp.NewFrameWriter(peerToClientW, 1<<20)
	peerDone := make(chan error, 1)
	go func() {
		var command struct {
			ID json.RawMessage `json:"id"`
		}
		payload, err := peerReader.ReadMessage()
		if err != nil {
			peerDone <- err
			return
		}
		if err := json.Unmarshal(payload, &command); err != nil || len(command.ID) == 0 {
			peerDone <- errors.New("malformed command request")
			return
		}
		for _, callback := range []struct {
			id     string
			params string
		}{
			{id: "valid", params: `{"edit":{"changes":{}}}`},
			{id: "malformed", params: `{"edit":null}`},
		} {
			request := `{"jsonrpc":"2.0","id":"` + callback.id + `","method":"workspace/applyEdit","params":` + callback.params + `}`
			if err := peerWriter.WriteMessage([]byte(request)); err != nil {
				peerDone <- err
				return
			}
			if _, err := peerReader.ReadMessage(); err != nil {
				peerDone <- err
				return
			}
		}
		peerDone <- peerWriter.WriteMessage([]byte(`{"jsonrpc":"2.0","id":` + string(command.ID) + `,"result":{}}`))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := session.RequestWithApplyEdit(ctx, "workspace/executeCommand", map[string]any{"command": "inline-value"}, func(raw json.RawMessage) (bool, error) {
		if string(raw) == "null" {
			return false, errors.New("malformed edit")
		}
		return true, nil
	})
	if err == nil || !strings.Contains(err.Error(), "more than one workspace/applyEdit") {
		t.Fatalf("RequestWithApplyEdit error = %v, want duplicate callback failure", err)
	}
	if peerErr := <-peerDone; peerErr != nil {
		t.Fatal(peerErr)
	}
}

func TestScalaApplyEditCapturePreservesCancellationAfterMalformedCallback(t *testing.T) {
	clientToPeerR, clientToPeerW := io.Pipe()
	peerToClientR, peerToClientW := io.Pipe()
	t.Cleanup(func() {
		_ = clientToPeerR.Close()
		_ = clientToPeerW.Close()
		_ = peerToClientR.Close()
		_ = peerToClientW.Close()
	})
	session := &scalaProcessSession{}
	session.client = lsp.NewClient(peerToClientR, clientToPeerW, lsp.WithRequestHandler(session.handleServerRequest))
	t.Cleanup(func() { _ = session.Close() })
	peerReader := lsp.NewFrameReader(clientToPeerR, 1<<20)
	peerWriter := lsp.NewFrameWriter(peerToClientW, 1<<20)
	callbackDone := make(chan error, 1)
	go func() {
		payload, err := peerReader.ReadMessage()
		if err != nil {
			callbackDone <- err
			return
		}
		var command struct {
			ID json.RawMessage `json:"id"`
		}
		if err := json.Unmarshal(payload, &command); err != nil || len(command.ID) == 0 {
			callbackDone <- errors.New("malformed command request")
			return
		}
		if err := peerWriter.WriteMessage([]byte(`{"jsonrpc":"2.0","id":"invalid","method":"workspace/applyEdit","params":{"edit":null}}`)); err != nil {
			callbackDone <- err
			return
		}
		if _, err := peerReader.ReadMessage(); err != nil {
			callbackDone <- err
			return
		}
		callbackDone <- nil
	}()
	ctx, cancel := context.WithCancel(context.Background())
	requestDone := make(chan error, 1)
	go func() {
		_, err := session.RequestWithApplyEdit(ctx, "workspace/executeCommand", map[string]any{"command": "inline-value"}, func(raw json.RawMessage) (bool, error) {
			if string(raw) == "null" {
				return false, errors.New("malformed edit")
			}
			return true, nil
		})
		requestDone <- err
	}()
	if err := <-callbackDone; err != nil {
		t.Fatal(err)
	}
	cancel()
	err := <-requestDone
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RequestWithApplyEdit error = %v, want context cancellation", err)
	}
}
