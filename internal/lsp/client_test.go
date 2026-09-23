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

func TestRequestCancellationReturnsWhenPeerStopsReading(t *testing.T) {
	clientToPeerR, clientToPeerW := io.Pipe()
	peerToClientR, peerToClientW := io.Pipe()
	client := lsp.NewClient(peerToClientR, clientToPeerW, lsp.WithCloser(clientToPeerW))
	client.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := client.Request(ctx, "slow", nil); done <- err }()
	if _, err := lsp.NewFrameReader(clientToPeerR, 1<<20).ReadMessage(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Request error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Request remained blocked while cancel notification could not be written")
	}
	_ = client.Close()
	_ = peerToClientR.Close()
	_ = client.Wait()
	_ = peerToClientW.Close()
	_ = peerToClientR.Close()
	_ = clientToPeerR.Close()
}

type blockingWriter struct {
	started chan struct{}
	release chan struct{}
}

func (w blockingWriter) Write(p []byte) (int, error) {
	select {
	case w.started <- struct{}{}:
	default:
	}
	<-w.release
	return len(p), nil
}

func TestRequestContextCancelsWhileInitialWriteIsBlocked(t *testing.T) {
	writer := blockingWriter{started: make(chan struct{}, 1), release: make(chan struct{})}
	peerToClientR, peerToClientW := io.Pipe()
	client := lsp.NewClient(peerToClientR, writer)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := client.Request(ctx, "slow", nil); done <- err }()
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("initial write did not start")
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Request error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Request remained blocked in initial write")
	}
	_ = client.Close()
	close(writer.release)
	_ = peerToClientW.Close()
}

type firstWriteGate struct {
	once    sync.Once
	started chan struct{}
	release chan struct{}
	mu      sync.Mutex
	data    bytes.Buffer
}

func (w *firstWriteGate) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.started); <-w.release })
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.data.Write(p)
}

func (w *firstWriteGate) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.data.String()
}

func TestCanceledQueuedRequestIsSkippedBeforeWrite(t *testing.T) {
	peerR, peerW := io.Pipe()
	writer := &firstWriteGate{started: make(chan struct{}), release: make(chan struct{})}
	client := lsp.NewClient(peerR, writer)
	firstDone := make(chan error, 1)
	go func() { _, err := client.Request(context.Background(), "first", nil); firstDone <- err }()
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("first outbound write did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	queuedDone := make(chan error, 1)
	go func() { _, err := client.Request(ctx, "queued", nil); queuedDone <- err }()
	select {
	case err := <-queuedDone:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("queued Request error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("queued Request did not honor its deadline")
	}
	close(writer.release)
	if err := client.Notify(context.Background(), "barrier", nil); err != nil {
		t.Fatal(err)
	}
	if output := writer.String(); strings.Contains(output, "queued") || strings.Contains(output, "$/cancelRequest") || !strings.Contains(output, "barrier") {
		t.Fatalf("outbound frames = %q", output)
	}
	_ = client.Close()
	_ = peerW.Close()
	if err := <-firstDone; !errors.Is(err, lsp.ErrClosed) {
		t.Fatalf("first Request error = %v", err)
	}
}

func TestInitialWriteCancellationQueuesBestEffortCancel(t *testing.T) {
	peerR, peerW := io.Pipe()
	writer := &firstWriteGate{started: make(chan struct{}), release: make(chan struct{})}
	client := lsp.NewClient(peerR, writer)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := client.Request(ctx, "partially_sent", nil); done <- err }()
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("initial outbound write did not start")
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Request error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("initial write cancellation did not return promptly")
	}
	close(writer.release)
	if err := client.Notify(context.Background(), "barrier", nil); err != nil {
		t.Fatal(err)
	}
	if output := writer.String(); !strings.Contains(output, "$/cancelRequest") || !strings.Contains(output, "barrier") {
		t.Fatalf("outbound frames = %q", output)
	}
	_ = client.Close()
	_ = peerW.Close()
}
