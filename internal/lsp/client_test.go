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

	"github.com/spockz/semantic-editor/internal/lsp"
	"github.com/spockz/semantic-editor/internal/telemetry"
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

func TestClientRecordsRequestAndNotifyTimingByMethod(t *testing.T) {
	clientToPeerR, clientToPeerW := io.Pipe()
	peerToClientR, peerToClientW := io.Pipe()
	cleanupTimingTestPipes(t, clientToPeerR, clientToPeerW, peerToClientR, peerToClientW)
	client := lsp.NewClient(peerToClientR, clientToPeerW, lsp.WithTelemetryPrefix("gopls"))
	defer func() { _ = client.Close() }()
	peerReader := lsp.NewFrameReader(clientToPeerR, 1<<20)
	peerWriter := lsp.NewFrameWriter(peerToClientW, 1<<20)

	requestMetrics := telemetry.NewMetrics()
	requestCtx := telemetry.WithPhase(telemetry.WithMetrics(context.Background(), requestMetrics), telemetry.PhaseVerificationBefore)
	requestDone := make(chan error, 1)
	go func() {
		result, err := client.Request(requestCtx, "textDocument/diagnostic", nil)
		if err == nil && string(result) != `{"items":[]}` {
			err = errors.New("unexpected diagnostic result: " + string(result))
		}
		requestDone <- err
	}()
	request := readTimingTestMessage(t, peerReader)
	if request.Method != "textDocument/diagnostic" || len(request.ID) == 0 {
		t.Fatalf("request frame = %#v", request)
	}
	time.Sleep(5 * time.Millisecond)
	if err := peerWriter.WriteMessage([]byte(`{"jsonrpc":"2.0","id":` + string(request.ID) + `,"result":{"items":[]}}`)); err != nil {
		t.Fatal(err)
	}
	if err := <-requestDone; err != nil {
		t.Fatal(err)
	}

	notifyMetrics := telemetry.NewMetrics()
	sendTimingTestNotification(telemetry.WithMetrics(context.Background(), notifyMetrics), t, client, "initialized", peerReader)

	remoteDone := make(chan error, 1)
	go func() { _, err := client.Request(requestCtx, "workspace/symbol", nil); remoteDone <- err }()
	remote := readTimingTestMessage(t, peerReader)
	if remote.Method != "workspace/symbol" {
		t.Fatalf("remote-error request = %#v", remote)
	}
	if err := peerWriter.WriteMessage([]byte(`{"jsonrpc":"2.0","id":` + string(remote.ID) + `,"error":{"code":-32000,"message":"unavailable"}}`)); err != nil {
		t.Fatal(err)
	}
	if err := <-remoteDone; !errors.Is(err, lsp.ErrRemoteResponse) {
		t.Fatalf("remote error = %v, want ErrRemoteResponse", err)
	}

	exerciseCanceledTimingRequest(requestCtx, t, client, peerReader)

	assertGoplsTimingMetrics(t, requestMetrics, notifyMetrics)
}

func exerciseCanceledTimingRequest(ctx context.Context, t *testing.T, client *lsp.Client, peerReader *lsp.FrameReader) {
	t.Helper()
	cancelCtx, cancel := context.WithCancel(ctx)
	cancelDone := make(chan error, 1)
	go func() { _, err := client.Request(cancelCtx, "slow", nil); cancelDone <- err }()
	if request := readTimingTestMessage(t, peerReader); request.Method != "slow" {
		t.Fatalf("cancellable request = %#v", request)
	}
	cancel()
	if err := <-cancelDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request = %v, want context.Canceled", err)
	}
	if cancellation := readTimingTestMessage(t, peerReader); cancellation.Method != "$/cancelRequest" {
		t.Fatalf("cancellation frame = %#v", cancellation)
	}
}

func readTimingTestMessage(t *testing.T, reader *lsp.FrameReader) struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
} {
	t.Helper()
	payload, err := reader.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var message struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatal(err)
	}
	return message
}

func sendTimingTestNotification(ctx context.Context, t *testing.T, client *lsp.Client, method string, peerReader *lsp.FrameReader) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- client.Notify(ctx, method, nil) }()
	notification := readTimingTestMessage(t, peerReader)
	if notification.Method != method || len(notification.ID) != 0 {
		t.Fatalf("notification frame = %#v, want notification %q", notification, method)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func assertGoplsTimingMetrics(t *testing.T, requestMetrics, notifyMetrics *telemetry.Metrics) {
	t.Helper()
	requestSnapshot := requestMetrics.Snapshot(0)
	for method, wantCount := range map[string]int{"textDocument/diagnostic": 1, "workspace/symbol": 1, "slow": 1} {
		for _, operation := range []string{"request", "send"} {
			if got := requestSnapshot.Phases["gopls."+operation+"."+method]; got.Count != wantCount {
				t.Errorf("%s timing for %s = %#v, want count %d", operation, method, got, wantCount)
			}
		}
	}
	for _, method := range []string{"textDocument/diagnostic", "workspace/symbol"} {
		if got := requestSnapshot.Phases["gopls.wait."+method]; got.Count != 1 {
			t.Errorf("response wait for %s = %#v, want one", method, got)
		}
	}
	canceledWaits := requestSnapshot.Phases["gopls.wait.slow"].Count
	if canceledWaits < 0 || canceledWaits > 1 {
		t.Errorf("canceled request response waits = %d, want zero or one depending on send completion", canceledWaits)
	}
	if got := requestSnapshot.Phases["gopls.wait.textDocument/diagnostic"]; got.DurationMS < 5 {
		t.Errorf("diagnostic response wait = %dms, want at least 5ms including peer delay", got.DurationMS)
	}
	if _, ok := requestSnapshot.Phases[string(telemetry.PhaseVerificationBefore)]; ok {
		t.Error("gopls request timing was attributed to the scoped verification phase")
	}
	notifySnapshot := notifyMetrics.Snapshot(0)
	if got := notifySnapshot.Phases["gopls.notify.initialized"]; got.Count != 1 {
		t.Errorf("notification timing = %#v, want one notification", got)
	}
	if len(notifySnapshot.Phases) != 1 || len(requestSnapshot.Phases) != 8+canceledWaits {
		t.Errorf("timing context attribution: request=%#v notify=%#v", requestSnapshot.Phases, notifySnapshot.Phases)
	}
}

func cleanupTimingTestPipes(t *testing.T, pipes ...io.Closer) {
	t.Helper()
	t.Cleanup(func() {
		for _, pipe := range pipes {
			_ = pipe.Close()
		}
	})
}

func TestClientTimingIsOptInAndRecorderIsOptional(t *testing.T) {
	clientToPeerR, clientToPeerW := io.Pipe()
	peerToClientR, peerToClientW := io.Pipe()
	cleanupTimingTestPipes(t, clientToPeerR, clientToPeerW, peerToClientR, peerToClientW)
	client := lsp.NewClient(peerToClientR, clientToPeerW)
	defer func() { _ = client.Close() }()
	peerReader := lsp.NewFrameReader(clientToPeerR, 1<<20)
	peerWriter := lsp.NewFrameWriter(peerToClientW, 1<<20)
	metrics := telemetry.NewMetrics()
	done := make(chan error, 1)
	go func() {
		_, err := client.Request(telemetry.WithMetrics(context.Background(), metrics), "plain/request", nil)
		done <- err
	}()
	request := readTimingTestMessage(t, peerReader)
	if err := peerWriter.WriteMessage([]byte(`{"jsonrpc":"2.0","id":` + string(request.ID) + `,"result":null}`)); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	sendTimingTestNotification(telemetry.WithMetrics(context.Background(), metrics), t, client, "plain/notify", peerReader)
	if got := metrics.Snapshot(0).Phases; len(got) != 0 {
		t.Fatalf("timing without an enabled prefix = %#v, want no phases", got)
	}

	clientToPeerR2, clientToPeerW2 := io.Pipe()
	peerToClientR2, peerToClientW2 := io.Pipe()
	cleanupTimingTestPipes(t, clientToPeerR2, clientToPeerW2, peerToClientR2, peerToClientW2)
	client2 := lsp.NewClient(peerToClientR2, clientToPeerW2, lsp.WithTelemetryPrefix("gopls"))
	defer func() { _ = client2.Close() }()
	peerReader2 := lsp.NewFrameReader(clientToPeerR2, 1<<20)
	peerWriter2 := lsp.NewFrameWriter(peerToClientW2, 1<<20)
	done = make(chan error, 1)
	go func() { _, err := client2.Request(context.Background(), "no/recorder", nil); done <- err }()
	request = readTimingTestMessage(t, peerReader2)
	if err := peerWriter2.WriteMessage([]byte(`{"jsonrpc":"2.0","id":` + string(request.ID) + `,"result":null}`)); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	sendTimingTestNotification(context.Background(), t, client2, "no/recorder/notify", peerReader2)
}

func TestProcessClientCapturesStderrWithoutOverwritingCallerWriter(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "printf process-error >&2")
	client, err := lsp.NewProcessClient(cmd)
	if err != nil {
		t.Fatal(err)
	}
	client.Start()
	if waitErr := client.Wait(); waitErr != nil && !errors.Is(waitErr, io.EOF) {
		t.Fatal(waitErr)
	}
	if closeErr := client.Close(); closeErr != nil {
		t.Fatal(closeErr)
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
	if waitErr := client.Wait(); waitErr != nil && !errors.Is(waitErr, io.EOF) {
		t.Fatal(waitErr)
	}
	if closeErr := client.Close(); closeErr != nil {
		t.Fatal(closeErr)
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
