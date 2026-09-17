package lsp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"semedit/internal/lsp"
)

func TestFramesRoundTripAndRejectMalformedOrOversizedPayload(t *testing.T) {
	reader := lsp.NewFrameReader(strings.NewReader("Content-Length: 2\r\n\r\nhi"), 10)
	payload, err := reader.ReadMessage()
	if err != nil || string(payload) != "hi" {
		t.Fatalf("ReadMessage = %q, %v", payload, err)
	}
	var headers strings.Builder
	for range 2000 {
		headers.WriteString("X: a\r\n")
	}
	headers.WriteString("Content-Length: 0\r\n\r\n")
	if _, err := lsp.NewFrameReader(strings.NewReader(headers.String()), 10).ReadMessage(); !errors.Is(err, lsp.ErrHeaderTooLarge) {
		t.Fatalf("cumulative header error = %v", err)
	}
	for _, input := range []string{"Content-Length nope\r\n\r\nx", "Content-Length: 99\r\n\r\nx", "X-Test: yes\r\n\r\nx", "Content-Length: 2"} {
		_, err := lsp.NewFrameReader(strings.NewReader(input), 10).ReadMessage()
		if err == nil {
			t.Fatalf("ReadMessage(%q) unexpectedly succeeded", input)
		}
		if input[0] == 'C' && strings.Contains(input, "99") && !errors.Is(err, lsp.ErrPayloadTooLarge) {
			t.Fatalf("error = %v, want payload limit", err)
		}
	}
}

func TestProcessClientCapturesStderrWithoutOverwritingCallerWriter(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "printf process-error >&2")
	client, err := lsp.NewProcessClient(cmd)
	if err != nil {
		t.Fatal(err)
	}
	client.Start()
	_ = client.Wait()
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if client.StderrCapture() == nil || string(client.StderrCapture().Bytes()) != "process-error" {
		t.Fatalf("captured stderr = %#v", client.StderrCapture())
	}

	var caller bytes.Buffer
	cmd = exec.CommandContext(context.Background(), "sh", "-c", "printf caller-error >&2")
	cmd.Stderr = &caller
	client, err = lsp.NewProcessClient(cmd)
	if err != nil {
		t.Fatal(err)
	}
	client.Start()
	_ = client.Wait()
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if client.StderrCapture() != nil || caller.String() != "caller-error" {
		t.Fatalf("caller stderr = %q capture=%#v", caller.String(), client.StderrCapture())
	}
}

func TestClientCorrelatesConcurrentRequestsAndNotifications(t *testing.T) {
	clientToPeerR, clientToPeerW := io.Pipe()
	peerToClientR, peerToClientW := io.Pipe()
	client := lsp.NewClient(peerToClientR, clientToPeerW)
	client.Start()
	defer func() { _ = client.Close() }()

	var requestsMu sync.Mutex
	requests := make(map[string]bool)
	go func() {
		frames := lsp.NewFrameReader(clientToPeerR, 1<<20)
		writer := lsp.NewFrameWriter(peerToClientW, 1<<20)
		for range 2 {
			payload, err := frames.ReadMessage()
			if err != nil {
				return
			}
			var request struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
			}
			if json.Unmarshal(payload, &request) != nil {
				return
			}
			requestsMu.Lock()
			requests[request.Method] = true
			requestsMu.Unlock()
			_ = writer.WriteMessage([]byte(`{"jsonrpc":"2.0","id":` + string(request.ID) + `,"result":{"ok":true}}`))
		}
	}()
	var wg sync.WaitGroup
	for _, method := range []string{"one", "two"} {
		wg.Add(1)
		go func(method string) {
			defer wg.Done()
			result, err := client.Request(context.Background(), method, map[string]string{"method": method})
			if err != nil || string(result) != `{"ok":true}` {
				t.Errorf("Request(%s) = %s, %v", method, result, err)
			}
		}(method)
	}
	wg.Wait()
	requestsMu.Lock()
	defer requestsMu.Unlock()
	if len(requests) != 2 {
		t.Fatalf("peer saw requests = %#v", requests)
	}
}

func TestServerInitiatedRequestAndNotificationDoNotBlockReader(t *testing.T) {
	peerToClientR, peerToClientW := io.Pipe()
	clientToPeerR, clientToPeerW := io.Pipe()
	gotNotification := make(chan string, 1)
	client := lsp.NewClient(peerToClientR, clientToPeerW,
		lsp.WithRequestHandler(func(_ context.Context, _ lsp.Request) (json.RawMessage, *lsp.ErrorObject) {
			return json.RawMessage(`{"answer":42}`), nil
		}),
		lsp.WithNotificationHandler(func(notification lsp.Notification) { gotNotification <- notification.Method }),
	)
	client.Start()
	peerReader := lsp.NewFrameReader(clientToPeerR, 1<<20)
	peerWriter := lsp.NewFrameWriter(peerToClientW, 1<<20)
	if err := peerWriter.WriteMessage([]byte(`{"jsonrpc":"2.0","id":"srv","method":"compute","params":{"x":1}}`)); err != nil {
		t.Fatal(err)
	}
	if err := peerWriter.WriteMessage([]byte(`{"jsonrpc":"2.0","method":"changed"}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case method := <-gotNotification:
		if method != "changed" {
			t.Fatalf("notification method = %q", method)
		}
	case <-time.After(time.Second):
		t.Fatal("notification handler blocked")
	}
	response, err := peerReader.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(response), `"id":"srv"`) || !strings.Contains(string(response), `"answer":42`) {
		t.Fatalf("response = %s", response)
	}
	_ = client.Close()
}

func TestRequestCancellationSendsCancelNotificationAndClosureUnblocks(t *testing.T) {
	clientToPeerR, clientToPeerW := io.Pipe()
	peerToClientR, peerToClientW := io.Pipe()
	client := lsp.NewClient(peerToClientR, clientToPeerW)
	client.Start()
	peerReader := lsp.NewFrameReader(clientToPeerR, 1<<20)
	requestDone := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	go func() { _, err := client.Request(ctx, "slow", nil); requestDone <- err }()
	if _, err := peerReader.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	if _, err := peerReader.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	if err := <-requestDone; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("request error = %v", err)
	}
	_ = client.Close()
	if err := client.Wait(); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Wait = %v", err)
	}
	_ = peerToClientW.Close()
	_ = peerToClientR.Close()
}

func TestMalformedJSONAndStderrCapture(t *testing.T) {
	peerToClientR, peerToClientW := io.Pipe()
	client := lsp.NewClient(peerToClientR, io.Discard)
	client.Start()
	writer := lsp.NewFrameWriter(peerToClientW, 1<<20)
	if err := writer.WriteMessage([]byte("not-json")); err != nil {
		t.Fatal(err)
	}
	if err := client.Wait(); !errors.Is(err, lsp.ErrMalformedPayload) {
		t.Fatalf("Wait = %v", err)
	}
	capture := lsp.NewStderrCapture(4)
	if n, err := capture.Write([]byte("abcdef")); n != 6 || err != nil {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if string(capture.Bytes()) != "abcd" || !capture.Truncated() {
		t.Fatalf("capture = %q truncated=%v", capture.Bytes(), capture.Truncated())
	}
}
