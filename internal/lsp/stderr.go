package lsp

import (
	"errors"
	"io"
	"sync"
)

// ErrStderrLimit is retained for callers that want to label a truncated capture.
var ErrStderrLimit = errors.New("stderr capture limit reached")

// StderrCapture stores a bounded copy of a process's stderr stream.
type StderrCapture struct {
	mu        sync.Mutex
	data      []byte
	limit     int
	truncated bool
}

// NewStderrCapture constructs a bounded stderr sink.
func NewStderrCapture(limit int) *StderrCapture {
	if limit < 0 {
		limit = 0
	}
	return &StderrCapture{limit: limit}
}

func (c *StderrCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	remaining := c.limit - len(c.data)
	if remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		c.data = append(c.data, p[:remaining]...)
	}
	if len(p) > remaining {
		c.truncated = true
	}
	return len(p), nil
}

// Bytes returns a copy of the captured stderr prefix.
func (c *StderrCapture) Bytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.data...)
}

// Truncated reports whether bytes were discarded after reaching the limit.
func (c *StderrCapture) Truncated() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.truncated }

// CaptureStderr consumes a separate stderr stream into a bounded result.
func CaptureStderr(r io.Reader, limit int) ([]byte, bool, error) {
	capture := NewStderrCapture(limit)
	_, err := io.Copy(capture, r)
	return capture.Bytes(), capture.Truncated(), err
}
