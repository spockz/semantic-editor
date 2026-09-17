// Package lsp provides a bounded, protocol-neutral JSON-RPC transport for language servers.
package lsp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

const defaultMaxPayloadSize int64 = 16 << 20
const maxHeaderSize = 8 << 10

var (
	// ErrMalformedHeader indicates an invalid or duplicate Content-Length header.
	ErrMalformedHeader = errors.New("malformed JSON-RPC header")
	// ErrMissingContentLength indicates that no Content-Length header was supplied.
	ErrMissingContentLength = errors.New("missing JSON-RPC Content-Length")
	// ErrPayloadTooLarge indicates that a frame exceeds the configured limit.
	ErrPayloadTooLarge = errors.New("JSON-RPC payload exceeds configured limit")
	// ErrMalformedPayload indicates invalid JSON in a frame payload.
	ErrMalformedPayload = errors.New("malformed JSON-RPC payload")
	// ErrHeaderTooLarge indicates that a frame header exceeds the fixed header limit.
	ErrHeaderTooLarge = errors.New("JSON-RPC header exceeds configured limit")
)

// FrameErrorKind identifies which part of a frame failed validation.
type FrameErrorKind string

const (
	// FrameHeaderError identifies a header validation failure.
	FrameHeaderError FrameErrorKind = "header"
	// FramePayloadError identifies a payload validation or I/O failure.
	FramePayloadError FrameErrorKind = "payload"
)

// FrameError reports a typed framing failure and its configured size limit.
type FrameError struct {
	Kind   FrameErrorKind
	Length int64
	Limit  int64
	Err    error
}

func (e *FrameError) Error() string {
	if e.Err == nil {
		return string(e.Kind) + " framing error"
	}
	return fmt.Sprintf("%s framing error: %v", e.Kind, e.Err)
}

func (e *FrameError) Unwrap() error { return e.Err }

// FrameReader reads bounded Content-Length framed messages.
type FrameReader struct {
	r         *bufio.Reader
	maxLength int64
}

// NewFrameReader constructs a reader with a bounded payload size.
func NewFrameReader(r io.Reader, maxPayloadSize int64) *FrameReader {
	if maxPayloadSize <= 0 {
		maxPayloadSize = defaultMaxPayloadSize
	}
	return &FrameReader{r: bufio.NewReader(r), maxLength: maxPayloadSize}
}

// ReadMessage reads one Content-Length framed payload.
func (r *FrameReader) ReadMessage() ([]byte, error) {
	length := int64(-1)
	headerCount := 0
	headerBytes := 0
	for {
		line, err := r.r.ReadString('\n')
		headerBytes += len(line)
		if len(line) > maxHeaderSize || headerBytes > maxHeaderSize {
			return nil, &FrameError{Kind: FrameHeaderError, Err: ErrHeaderTooLarge}
		}
		if err != nil {
			if len(line) != 0 {
				return nil, &FrameError{Kind: FrameHeaderError, Err: ErrMalformedHeader}
			}
			return nil, err
		}
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			break
		}
		headerCount++
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, &FrameError{Kind: FrameHeaderError, Err: ErrMalformedHeader}
		}
		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			if length >= 0 {
				return nil, &FrameError{Kind: FrameHeaderError, Err: ErrMalformedHeader}
			}
			parsed, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if parseErr != nil || parsed < 0 {
				return nil, &FrameError{Kind: FrameHeaderError, Err: ErrMalformedHeader}
			}
			length = parsed
		}
	}
	if headerCount == 0 || length < 0 {
		return nil, &FrameError{Kind: FrameHeaderError, Err: ErrMissingContentLength}
	}
	if length > r.maxLength {
		return nil, &FrameError{Kind: FramePayloadError, Length: length, Limit: r.maxLength, Err: ErrPayloadTooLarge}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r.r, payload); err != nil {
		return nil, &FrameError{Kind: FramePayloadError, Length: length, Limit: r.maxLength, Err: err}
	}
	return payload, nil
}

// FrameWriter writes serialized bounded Content-Length framed messages.
type FrameWriter struct {
	w         io.Writer
	maxLength int64
	mu        sync.Mutex
}

// NewFrameWriter constructs a serialized writer with a bounded payload size.
func NewFrameWriter(w io.Writer, maxPayloadSize int64) *FrameWriter {
	if maxPayloadSize <= 0 {
		maxPayloadSize = defaultMaxPayloadSize
	}
	return &FrameWriter{w: w, maxLength: maxPayloadSize}
}

// WriteMessage writes one Content-Length framed payload.
func (w *FrameWriter) WriteMessage(payload []byte) error {
	if int64(len(payload)) > w.maxLength {
		return &FrameError{Kind: FramePayloadError, Length: int64(len(payload)), Limit: w.maxLength, Err: ErrPayloadTooLarge}
	}
	frame := bytes.NewBuffer(make([]byte, 0, len(payload)+32))
	fmt.Fprintf(frame, "Content-Length: %d\r\n\r\n", len(payload))
	frame.Write(payload)
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := frame.WriteTo(w.w)
	return err
}
