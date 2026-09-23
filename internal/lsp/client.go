package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
)

const jsonRPCVersion = "2.0"
const defaultStderrCaptureLimit = 64 << 10
const outboundQueueCapacity = 32

var (
	// ErrClosed indicates that the client connection has terminated.
	ErrClosed = errors.New("JSON-RPC client is closed")
	// ErrInvalidMessage indicates a JSON-RPC envelope that violates the transport contract.
	ErrInvalidMessage = errors.New("invalid JSON-RPC message")
	// ErrRemoteResponse identifies an error returned by the peer.
	ErrRemoteResponse = errors.New("remote JSON-RPC error")
)

// ErrorObject is the JSON-RPC error payload returned by a peer or request handler.
type ErrorObject struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// RemoteError preserves the response ID and peer error object.
type RemoteError struct {
	ID       json.RawMessage
	RPCError ErrorObject
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.RPCError.Code, e.RPCError.Message)
}
func (e *RemoteError) Unwrap() error { return ErrRemoteResponse }

// Request is a server-initiated JSON-RPC request delivered to a handler.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Notification is a JSON-RPC notification delivered to a handler.
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RequestHandler handles server-initiated requests. A non-nil error object sends an error response.
type RequestHandler func(context.Context, Request) (json.RawMessage, *ErrorObject)

// NotificationHandler handles server-initiated notifications.
type NotificationHandler func(Notification)

// Option configures a Client.
type Option func(*Client)

// WithMaxPayloadSize sets the maximum framed payload size.
func WithMaxPayloadSize(size int64) Option { return func(c *Client) { c.maxPayloadSize = size } }

// WithRequestHandler registers the asynchronous server-request handler.
func WithRequestHandler(handler RequestHandler) Option {
	return func(c *Client) { c.requestHandler = handler }
}

// WithNotificationHandler registers the asynchronous notification handler.
func WithNotificationHandler(handler NotificationHandler) Option {
	return func(c *Client) { c.notificationHandler = handler }
}

// WithCloser supplies the stream/process closer used by Client.Close.
func WithCloser(closer io.Closer) Option { return func(c *Client) { c.closer = closer } }

// Client is a concurrent bidirectional JSON-RPC client over supplied streams.
// Outbound writes use one worker and a bounded queue; a blocked generic writer can
// retain that single worker until it unblocks because io.Writer has no cancel API.
type Client struct {
	reader              io.Reader
	writer              io.Writer
	closer              io.Closer
	maxPayloadSize      int64
	requestHandler      RequestHandler
	notificationHandler NotificationHandler

	frames        *FrameReader
	output        *FrameWriter
	done          chan struct{}
	closeOnce     sync.Once
	startOnce     sync.Once
	started       chan struct{}
	readerErr     chan error
	wg            sync.WaitGroup
	stderrCapture *StderrCapture

	nextID   atomic.Uint64
	mu       sync.Mutex
	pending  map[string]chan response
	outbound chan outboundMessage
}

type response struct {
	result json.RawMessage
	err    error
}

type outboundMessage struct {
	payload   []byte
	ctx       context.Context
	lifecycle *outboundLifecycle
	result    chan error
}

type outboundLifecycle struct {
	mu      sync.Mutex
	started bool
	skipped bool
}

// NewClient constructs a client over reader and writer. Call Start or Request to begin reading.
func NewClient(reader io.Reader, writer io.Writer, options ...Option) *Client {
	c := &Client{reader: reader, writer: writer, maxPayloadSize: defaultMaxPayloadSize, done: make(chan struct{}), started: make(chan struct{}), readerErr: make(chan error, 1), outbound: make(chan outboundMessage, outboundQueueCapacity), pending: make(map[string]chan response)}
	for _, option := range options {
		if option != nil {
			option(c)
		}
	}
	if c.maxPayloadSize <= 0 {
		c.maxPayloadSize = defaultMaxPayloadSize
	}
	if c.closer == nil {
		if closeReader, ok := reader.(io.Closer); ok {
			c.closer = closeReader
		}
	}
	c.frames = NewFrameReader(reader, c.maxPayloadSize)
	c.output = NewFrameWriter(writer, c.maxPayloadSize)
	return c
}

// NewProcessClient starts an already configured command and connects its standard streams.
func NewProcessClient(cmd *exec.Cmd, options ...Option) (*Client, error) {
	var stderrCapture *StderrCapture
	if cmd.Stderr == nil {
		stderrCapture = NewStderrCapture(defaultStderrCaptureLimit)
		cmd.Stderr = stderrCapture
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open LSP stdout: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open LSP stdin: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start LSP process: %w", err)
	}
	c := NewClient(stdout, stdin, options...)
	c.closer = processCloser{stdin: stdin, stdout: stdout, cmd: cmd}
	c.stderrCapture = stderrCapture
	return c, nil
}

type processCloser struct {
	stdin  io.Closer
	stdout io.Closer
	cmd    *exec.Cmd
}

func (p processCloser) Close() error {
	_ = p.stdin.Close()
	_ = p.stdout.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	if err := p.cmd.Wait(); err != nil {
		if _, ok := errors.AsType[*exec.ExitError](err); ok {
			return nil
		}
		return err
	}
	return nil
}

// Start begins the reader loop. It is safe to call more than once.
func (c *Client) Start() {
	c.startOnce.Do(func() {
		close(c.started)
		c.wg.Add(1)
		go c.readLoop()
		go c.writeLoop()
	})
}

func (c *Client) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case message := <-c.outbound:
			if message.lifecycle != nil {
				message.lifecycle.mu.Lock()
				if message.lifecycle.skipped || (message.ctx != nil && message.ctx.Err() != nil) {
					message.lifecycle.skipped = true
					message.lifecycle.mu.Unlock()
					if message.result != nil {
						message.result <- message.ctx.Err()
					}
					continue
				}
				message.lifecycle.started = true
				message.lifecycle.mu.Unlock()
			} else if message.ctx != nil && message.ctx.Err() != nil {
				if message.result != nil {
					message.result <- message.ctx.Err()
				}
				continue
			}
			err := c.output.WriteMessage(message.payload)
			if message.result != nil {
				message.result <- err
			}
		}
	}
}

// Wait waits for the reader loop to terminate and returns its terminal error.
func (c *Client) Wait() error {
	c.Start()
	c.wg.Wait()
	select {
	case err := <-c.readerErr:
		return err
	default:
		return nil
	}
}

// StderrCapture returns the process-owned bounded stderr capture, if installed.
func (c *Client) StderrCapture() *StderrCapture { return c.stderrCapture }

func (c *Client) readLoop() {
	defer c.wg.Done()
	for {
		payload, err := c.frames.ReadMessage()
		if err != nil {
			if !errors.Is(err, ErrClosed) {
				select {
				case c.readerErr <- err:
				default:
				}
			}
			c.finish(err)
			return
		}
		if err := c.dispatch(payload); err != nil {
			select {
			case c.readerErr <- err:
			default:
			}
			c.finish(err)
			return
		}
	}
}

func (c *Client) dispatch(payload []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return &FrameError{Kind: FramePayloadError, Err: fmt.Errorf("%w: %w", ErrMalformedPayload, err)}
	}
	if raw == nil {
		return &FrameError{Kind: FramePayloadError, Err: ErrMalformedPayload}
	}
	var version string
	if err := json.Unmarshal(raw["jsonrpc"], &version); err != nil || version != jsonRPCVersion {
		return &FrameError{Kind: FramePayloadError, Err: ErrInvalidMessage}
	}
	if methodRaw, ok := raw["method"]; ok {
		var method string
		if err := json.Unmarshal(methodRaw, &method); err != nil || method == "" {
			return &FrameError{Kind: FramePayloadError, Err: ErrInvalidMessage}
		}
		if id, hasID := raw["id"]; hasID {
			request := Request{JSONRPC: version, ID: append(json.RawMessage(nil), id...), Method: method, Params: cloneRaw(raw["params"])}
			go c.handleRequest(request)
		} else if c.notificationHandler != nil {
			notification := Notification{JSONRPC: version, Method: method, Params: cloneRaw(raw["params"])}
			go c.notificationHandler(notification)
		}
		return nil
	}
	id, ok := raw["id"]
	if !ok {
		return &FrameError{Kind: FramePayloadError, Err: ErrInvalidMessage}
	}
	c.mu.Lock()
	pending := c.pending[string(id)]
	c.mu.Unlock()
	if pending == nil {
		return nil
	}
	resultRaw, hasResult := raw["result"]
	errorRaw, hasError := raw["error"]
	if hasResult == hasError {
		return &FrameError{Kind: FramePayloadError, Err: ErrInvalidMessage}
	}
	var rpcErr *ErrorObject
	if hasError {
		var decoded ErrorObject
		if err := json.Unmarshal(errorRaw, &decoded); err != nil {
			return &FrameError{Kind: FramePayloadError, Err: ErrInvalidMessage}
		}
		rpcErr = &decoded
	}
	var responseErr error
	if rpcErr != nil {
		responseErr = &RemoteError{ID: cloneRaw(id), RPCError: *rpcErr}
	}
	select {
	case pending <- response{result: cloneRaw(resultRaw), err: responseErr}:
	default:
		return &FrameError{Kind: FramePayloadError, Err: ErrInvalidMessage}
	}
	return nil
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func (c *Client) handleRequest(request Request) {
	var result json.RawMessage
	var rpcErr *ErrorObject
	if c.requestHandler == nil {
		rpcErr = &ErrorObject{Code: -32601, Message: "method not found"}
	} else {
		result, rpcErr = c.requestHandler(c.context(), request)
	}
	message := map[string]any{"jsonrpc": jsonRPCVersion, "id": request.ID}
	switch {
	case rpcErr != nil:
		message["error"] = rpcErr
	case result == nil:
		message["result"] = json.RawMessage("null")
	default:
		message["result"] = result
	}
	_ = c.send(context.Background(), message)
}

func (c *Client) context() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-c.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx
}

func (c *Client) send(ctx context.Context, message any) error {
	_, err := c.sendTracked(ctx, message)
	return err
}

func (c *Client) sendTracked(ctx context.Context, message any) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return false, fmt.Errorf("marshal JSON-RPC message: %w", err)
	}
	c.Start()
	result := make(chan error, 1)
	lifecycle := &outboundLifecycle{}
	select {
	case <-c.done:
		return false, ErrClosed
	case <-ctx.Done():
		return false, ctx.Err()
	case c.outbound <- outboundMessage{payload: payload, ctx: ctx, lifecycle: lifecycle, result: result}:
	}
	select {
	case err := <-result:
		return lifecycle.hasStarted(), err
	case <-c.done:
		return lifecycle.hasStarted(), ErrClosed
	case <-ctx.Done():
		return lifecycle.cancelOrSkip(), ctx.Err()
	}
}

func (l *outboundLifecycle) hasStarted() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.started
}

func (l *outboundLifecycle) cancelOrSkip() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.started {
		l.skipped = true
	}
	return l.started
}

func (c *Client) sendBestEffort(message any) {
	payload, err := json.Marshal(message)
	if err != nil {
		return
	}
	select {
	case <-c.done:
	case c.outbound <- outboundMessage{payload: payload}:
	default:
	}
}

// Request sends a request and waits for its correlated response or context termination.
func (c *Client) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.Start()
	select {
	case <-c.done:
		return nil, ErrClosed
	default:
	}
	id := strconv.FormatUint(c.nextID.Add(1), 10)
	resultCh := make(chan response, 1)
	c.mu.Lock()
	c.pending[id] = resultCh
	c.mu.Unlock()
	message := map[string]any{"jsonrpc": jsonRPCVersion, "id": json.RawMessage(id), "method": method}
	if params != nil {
		message["params"] = params
	}
	started, err := c.sendTracked(ctx, message)
	if err != nil {
		c.removePending(id)
		if started && ctx.Err() != nil {
			c.sendBestEffort(map[string]any{"jsonrpc": jsonRPCVersion, "method": "$/cancelRequest", "params": map[string]any{"id": json.RawMessage(id)}})
		}
		return nil, err
	}
	select {
	case result := <-resultCh:
		c.removePending(id)
		if result.err != nil {
			return nil, result.err
		}
		return result.result, nil
	case <-ctx.Done():
		if c.removePending(id) {
			c.sendBestEffort(map[string]any{"jsonrpc": jsonRPCVersion, "method": "$/cancelRequest", "params": map[string]any{"id": json.RawMessage(id)}})
		}
		return nil, ctx.Err()
	case <-c.done:
		c.removePending(id)
		return nil, ErrClosed
	}
}

func (c *Client) removePending(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.pending[id]; !ok {
		return false
	}
	delete(c.pending, id)
	return true
}

// Notify sends a JSON-RPC notification without waiting for a response.
func (c *Client) Notify(ctx context.Context, method string, params any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	message := map[string]any{"jsonrpc": jsonRPCVersion, "method": method}
	if params != nil {
		message["params"] = params
	}
	return c.send(ctx, message)
}

func (c *Client) finish(_ error) {
	c.closeOnce.Do(func() {
		close(c.done)
		pending := make([]chan response, 0, len(c.pending))
		c.mu.Lock()
		for id, resultCh := range c.pending {
			delete(c.pending, id)
			pending = append(pending, resultCh)
		}
		c.mu.Unlock()
		for _, resultCh := range pending {
			select {
			case resultCh <- response{err: ErrClosed}:
			default:
			}
		}
	})
}

// Close terminates the client and closes its supplied stream or process.
func (c *Client) Close() error {
	c.finish(ErrClosed)
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}
