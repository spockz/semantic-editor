package lsp

import (
	"testing"
	"time"
)

func TestFinishDoesNotBlockOnBufferedPendingResponse(t *testing.T) {
	resultCh := make(chan response, 1)
	resultCh <- response{result: []byte(`{"already":"delivered"}`)}
	client := &Client{done: make(chan struct{}), pending: map[string]chan response{"1": resultCh}}

	done := make(chan struct{})
	go func() {
		client.finish(ErrClosed)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("finish blocked on a buffered response")
	}
}
