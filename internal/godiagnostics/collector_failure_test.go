// WHY: This file verifies all-or-nothing diagnostic collection and recovery after a remote LSP failure.

package godiagnostics_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/lsp"
)

func TestManagedDiagnosticsCollectorFailureDoesNotReturnPartialFindings(t *testing.T) {
	fixture := newCacheInputFixture(t, map[string]string{
		"alpha/value.go": "package alpha\nfunc Value() int { return 1 }\n",
		"beta/value.go":  "package beta\nfunc Value() int { return 2 }\n",
	})
	helperBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	stateDir := t.TempDir()
	marker := filepath.Join(stateDir, "failed-once")
	logPath := filepath.Join(stateDir, "server.log")
	wrapper := filepath.Join(stateDir, "fake-gopls")
	script := "#!/bin/sh\nexec \"$SEMEDIT_DIAGNOSTICS_HELPER_BINARY\" -test.run=^TestManagedDiagnosticsFakeGoplsProcess$ -- \"$@\"\n"
	if writeErr := os.WriteFile(wrapper, []byte(script), 0o700); writeErr != nil {
		t.Fatal(writeErr)
	}
	environment := append(slices.Clone(fixture.environment),
		"SEMEDIT_DIAGNOSTICS_HELPER_MODE=1",
		"SEMEDIT_DIAGNOSTICS_HELPER_BINARY="+helperBinary,
		"SEMEDIT_DIAGNOSTICS_FAIL_MARKER="+marker,
		"SEMEDIT_DIAGNOSTICS_HELPER_LOG="+logPath,
	)
	run := func(ctx context.Context) ([]godiagnostics.Diagnostic, error) {
		return godiagnostics.CheckDetailedWithContext(ctx, fixture.root, wrapper, environment)
	}
	firstCtx, firstCancel := context.WithTimeout(godiagnostics.WithIncrementalDiagnostics(fixture.ctx), 20*time.Second)
	partial, err := run(firstCtx)
	firstCancel()
	if !errors.Is(err, lsp.ErrRemoteResponse) {
		t.Fatalf("first diagnostics error = %v, want remote response error", err)
	}
	if len(partial) != 0 {
		t.Fatalf("failed capture returned partial findings: %+v", partial)
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	events := string(log)
	success := strings.Index(events, "diagnostic-success\n")
	failure := strings.Index(events, "diagnostic-error\n")
	if success < 0 || failure <= success {
		t.Fatalf("server events = %q, want a successful diagnostic response before the remote error", events)
	}
	if got := strings.Count(events, "start\n"); got != 1 {
		t.Fatalf("server starts after failed capture = %d, want 1", got)
	}

	assertManagedDiagnosticsRetryRecovery(fixture.ctx, t, run, fixture.root, wrapper, environment, logPath)
}

func assertManagedDiagnosticsRetryRecovery(baseCtx context.Context, t *testing.T, run func(context.Context) ([]godiagnostics.Diagnostic, error), root, wrapper string, environment []string, logPath string) {
	t.Helper()
	retryCtx, retryCancel := context.WithTimeout(godiagnostics.WithIncrementalDiagnostics(baseCtx), 20*time.Second)
	retained, retryErr := run(retryCtx)
	retryCancel()
	if retryErr != nil {
		t.Fatalf("healthy retry: %v", retryErr)
	}
	freshCtx, freshCancel := context.WithTimeout(context.Background(), 20*time.Second)
	fresh, freshErr := godiagnostics.CheckDetailedWithContext(freshCtx, root, wrapper, environment)
	freshCancel()
	if freshErr != nil {
		t.Fatalf("fresh healthy capture: %v", freshErr)
	}
	if len(retained) == 0 || strings.Join(godiagnostics.Messages(retained), "\n") != strings.Join(godiagnostics.Messages(fresh), "\n") {
		t.Fatalf("healthy retry diagnostics %q differ from fresh capture %q", godiagnostics.Messages(retained), godiagnostics.Messages(fresh))
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(log), "start\n"); got != 3 {
		t.Fatalf("server starts after retry and fresh capture = %d, want failed, retry, and oracle sessions", got)
	}
	events := string(log)
	unchangedCtx, unchangedCancel := context.WithTimeout(godiagnostics.WithIncrementalDiagnostics(baseCtx), 20*time.Second)
	unchanged, unchangedErr := run(unchangedCtx)
	unchangedCancel()
	if unchangedErr != nil {
		t.Fatalf("unchanged capture after retry: %v", unchangedErr)
	}
	if strings.Join(godiagnostics.Messages(unchanged), "\n") != strings.Join(godiagnostics.Messages(fresh), "\n") {
		t.Fatalf("unchanged retry diagnostics %q differ from fresh capture %q", godiagnostics.Messages(unchanged), godiagnostics.Messages(fresh))
	}
	log, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(log) != events {
		t.Fatalf("unchanged capture contacted the server: events before %q, after %q", events, string(log))
	}
}

func TestManagedDiagnosticsFakeGoplsProcess(t *testing.T) {
	if os.Getenv("SEMEDIT_DIAGNOSTICS_HELPER_MODE") != "1" {
		return
	}
	if err := runManagedDiagnosticsFakeGopls(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func runManagedDiagnosticsFakeGopls() error {
	logPath := os.Getenv("SEMEDIT_DIAGNOSTICS_HELPER_LOG")
	markerPath := os.Getenv("SEMEDIT_DIAGNOSTICS_FAIL_MARKER")
	if logPath == "" || markerPath == "" {
		return errors.New("fake gopls helper environment is incomplete")
	}
	logEvent := func(event string) error {
		file, openErr := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if openErr != nil {
			return openErr
		}
		_, writeErr := fmt.Fprintln(file, event)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
	if startErr := logEvent("start"); startErr != nil {
		return startErr
	}
	failOnce, markerErr := createFakeGoplsFailureMarker(markerPath)
	if markerErr != nil {
		return markerErr
	}
	reader := lsp.NewFrameReader(os.Stdin, 1<<20)
	writer := lsp.NewFrameWriter(os.Stdout, 1<<20)
	return serveManagedDiagnosticsFakeGopls(reader, writer, failOnce, logEvent)
}

func createFakeGoplsFailureMarker(path string) (bool, error) {
	marker, markerErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(markerErr, os.ErrExist) {
		return false, nil
	}
	if markerErr != nil {
		return false, markerErr
	}
	if closeErr := marker.Close(); closeErr != nil {
		return false, closeErr
	}
	return true, nil
}

func serveManagedDiagnosticsFakeGopls(reader *lsp.FrameReader, writer *lsp.FrameWriter, failOnce bool, logEvent func(string) error) error {
	diagnostics := 0
	for {
		payload, readErr := reader.ReadMessage()
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
		encoded, exit, responseErr := fakeGoplsResponse(payload, failOnce, &diagnostics, logEvent)
		if responseErr != nil {
			return responseErr
		}
		if exit {
			return nil
		}
		if encoded == nil {
			continue
		}
		if writeErr := writer.WriteMessage(encoded); writeErr != nil {
			return writeErr
		}
	}
}

func fakeGoplsResponse(payload []byte, failOnce bool, diagnostics *int, logEvent func(string) error) ([]byte, bool, error) {
	var request struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if decodeErr := json.Unmarshal(payload, &request); decodeErr != nil {
		return nil, false, decodeErr
	}
	if len(request.ID) == 0 {
		return nil, request.Method == "exit", nil
	}
	response := map[string]any{"jsonrpc": "2.0", "id": request.ID}
	switch request.Method {
	case "initialize":
		response["result"] = map[string]any{
			"capabilities": map[string]any{"diagnosticProvider": map[string]any{}, "positionEncoding": "utf-16"},
		}
	case "textDocument/diagnostic":
		if diagnosticErr := fakeGoplsDiagnosticResult(response, failOnce, diagnostics, logEvent); diagnosticErr != nil {
			return nil, false, diagnosticErr
		}
	default:
		response["result"] = nil
	}
	encoded, encodeErr := json.Marshal(response)
	return encoded, false, encodeErr
}

func fakeGoplsDiagnosticResult(response map[string]any, failOnce bool, diagnostics *int, logEvent func(string) error) error {
	(*diagnostics)++
	if failOnce && *diagnostics == 2 {
		if eventErr := logEvent("diagnostic-error"); eventErr != nil {
			return eventErr
		}
		response["error"] = map[string]any{"code": -32000, "message": "injected diagnostic failure"}
		return nil
	}
	if eventErr := logEvent("diagnostic-success"); eventErr != nil {
		return eventErr
	}
	response["result"] = map[string]any{
		"kind": "full",
		"items": []map[string]any{{
			"range": map[string]any{
				"start": map[string]int{"line": 0, "character": 0},
				"end":   map[string]int{"line": 0, "character": 1},
			},
			"message":  "synthetic finding",
			"severity": 1,
		}},
	}
	return nil
}
