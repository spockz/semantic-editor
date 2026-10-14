// Package mcptest validates MCP test responses against advertised output schemas.
package mcptest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/spockz/semantic-editor/internal/mcp"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Server wraps an MCP server and checks every captured structured tool result.
type Server struct {
	*mcp.Server

	validator *responseValidator
	profile   string
	root      string
	options   []mcp.Option
}

// Initialize delegates initialization and refreshes schemas for the resolved workspace root.
func (s *Server) Initialize(ctx context.Context, rawParams json.RawMessage) (map[string]any, error) {
	result, err := s.Server.Initialize(ctx, rawParams)
	if err != nil {
		return result, err
	}
	if root, ok := result["workspaceRoot"].(string); ok {
		s.root = root
		s.validator.discover(s.profile, root, s.options)
	}
	return result, nil
}

// Serve preserves the underlying server's wire behavior while validating emitted tool results.
func (s *Server) Serve(ctx context.Context, input io.Reader) error {
	if s.root != "" {
		s.validator.discover(s.profile, s.root, s.options)
	}
	err := s.Server.Serve(ctx, &requestReader{reader: input, validator: s.validator})
	s.validator.flush()
	return err
}

// NewServer creates an MCP test server that validates structured results against advertised schemas.
func NewServer(t testing.TB, profile, root string, output io.Writer, options ...mcp.Option) *Server {
	t.Helper()
	validator := &responseValidator{
		testName:   t.Name(),
		report:     func(message string) { t.Errorf("%s", message) },
		dst:        output,
		profile:    profile,
		root:       root,
		options:    slices.Clone(options),
		requests:   make(map[string]requestInfo),
		advertised: make(map[string]*jsonschema.Schema),
		full:       make(map[string]*jsonschema.Schema),
	}
	server := &Server{
		validator: validator,
		profile:   profile,
		root:      root,
		options:   slices.Clone(options),
	}
	server.Server = mcp.NewServer(profile, root, validator, options...)
	return server
}

type requestInfo struct {
	method string
	tool   string
}

type responseValidator struct {
	testName       string
	report         func(string)
	dst            io.Writer
	profile        string
	root           string
	options        []mcp.Option
	mu             sync.Mutex
	pending        []byte
	requests       map[string]requestInfo
	advertised     map[string]*jsonschema.Schema
	full           map[string]*jsonschema.Schema
	catalogErr     error
	catalogReady   bool
	catalogRoot    string
	catalogProfile string
}

type requestReader struct {
	reader    io.Reader
	validator *responseValidator
	pending   []byte
}

func (r *requestReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.pending = append(r.pending, p[:n]...)
		for {
			index := bytes.IndexByte(r.pending, '\n')
			if index < 0 {
				break
			}
			line := bytes.Clone(r.pending[:index])
			r.pending = r.pending[index+1:]
			r.validator.recordRequest(line)
		}
	}
	if err == io.EOF && len(r.pending) != 0 {
		r.validator.recordRequest(r.pending)
		r.pending = nil
	}
	return n, err
}

func (r *requestReader) Close() error {
	if closer, ok := r.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (v *responseValidator) discover(profile, root string, options []mcp.Option) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.catalogReady && v.catalogRoot == root && v.catalogProfile == profile {
		return
	}
	advertised, err := probeSchemas(profile, root, options)
	v.catalogErr = nil
	if err != nil {
		v.catalogErr = fmt.Errorf("advertised catalog: %w", err)
	}
	fullOptions := append(slices.Clone(options), mcp.WithToolProfile("full"))
	full, fullErr := probeSchemas("full", root, fullOptions)
	if fullErr != nil && v.catalogErr == nil {
		v.catalogErr = fmt.Errorf("full catalog fallback: %w", fullErr)
	}
	v.advertised = advertised
	v.full = full
	v.catalogRoot = root
	v.catalogProfile = profile
	v.catalogReady = true
}

func probeSchemas(profile, root string, options []mcp.Option) (map[string]*jsonschema.Schema, error) {
	var output bytes.Buffer
	server := mcp.NewServer(profile, root, &output, options...)
	request := "{\"jsonrpc\":\"2.0\",\"id\":\"mcptest-catalog\",\"method\":\"tools/list\"}\n"
	if err := server.Serve(context.Background(), strings.NewReader(request)); err != nil {
		return nil, fmt.Errorf("list tools for profile %q: %w", profile, err)
	}
	var response struct {
		Error  json.RawMessage `json:"error"`
		Result struct {
			Tools []struct {
				Name         string          `json:"name"`
				OutputSchema json.RawMessage `json:"outputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("decode tools/list response: %w", err)
	}
	if len(response.Error) != 0 {
		return nil, fmt.Errorf("tools/list returned JSON-RPC error: %s", response.Error)
	}
	if response.Result.Tools == nil {
		return nil, fmt.Errorf("tools/list response omitted result.tools")
	}
	schemas := make(map[string]*jsonschema.Schema, len(response.Result.Tools))
	for _, tool := range response.Result.Tools {
		if tool.Name == "" || len(tool.OutputSchema) == 0 {
			return nil, fmt.Errorf("tools/list returned a tool without name or outputSchema")
		}
		schema, err := compileSchema(tool.OutputSchema)
		if err != nil {
			return nil, fmt.Errorf("compile %s outputSchema: %w", tool.Name, err)
		}
		schemas[tool.Name] = schema
	}
	return schemas, nil
}

// CompileOutputSchema compiles one advertised MCP output schema for independent validation.
func CompileOutputSchema(raw json.RawMessage) (*jsonschema.Schema, error) {
	return compileSchema(raw)
}

func compileSchema(raw json.RawMessage) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode schema resource: %w", err)
	}
	if err := compiler.AddResource("output-schema.json", document); err != nil {
		return nil, err
	}
	return compiler.Compile("output-schema.json")
}

func (v *responseValidator) recordRequest(line []byte) {
	var request struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if json.Unmarshal(line, &request) != nil || len(request.ID) == 0 {
		return
	}
	key := string(bytes.TrimSpace(request.ID))
	v.mu.Lock()
	v.requests[key] = requestInfo{method: request.Method, tool: request.Params.Name}
	v.mu.Unlock()
}

func (v *responseValidator) Write(p []byte) (int, error) {
	n, err := v.dst.Write(p)
	if err != nil || n == 0 {
		return n, err
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}
	v.mu.Lock()
	v.pending = append(v.pending, p...)
	var lines [][]byte
	for {
		index := bytes.IndexByte(v.pending, '\n')
		if index < 0 {
			break
		}
		lines = append(lines, bytes.Clone(v.pending[:index]))
		v.pending = v.pending[index+1:]
	}
	v.mu.Unlock()
	for _, line := range lines {
		v.recordResponse(line)
	}
	return n, nil
}

func (v *responseValidator) flush() {
	v.mu.Lock()
	line := bytes.Clone(v.pending)
	v.pending = nil
	v.mu.Unlock()
	if len(line) != 0 {
		v.recordResponse(line)
	}
}

func (v *responseValidator) recordResponse(line []byte) {
	var response struct {
		ID     json.RawMessage `json:"id"`
		Error  json.RawMessage `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(line, &response); err != nil {
		v.report(fmt.Sprintf("%s: malformed MCP response: %v", v.testName, err))
		return
	}
	if len(response.ID) == 0 {
		return
	}
	key := string(bytes.TrimSpace(response.ID))
	v.mu.Lock()
	request, ok := v.requests[key]
	delete(v.requests, key)
	v.mu.Unlock()
	if len(response.Error) != 0 {
		return
	}
	if !ok {
		v.report(fmt.Sprintf("%s: MCP response ID %s has no captured request", v.testName, key))
		return
	}
	if request.method == "tools/list" {
		if schemas, err := schemasFromResult(response.Result); err != nil {
			v.report(fmt.Sprintf("%s: decode live tools/list output schemas: %v", v.testName, err))
		} else {
			v.mu.Lock()
			v.advertised = schemas
			v.mu.Unlock()
		}
		return
	}
	if request.method == "initialize" {
		var initialized struct {
			WorkspaceRoot string `json:"workspaceRoot"`
		}
		if json.Unmarshal(response.Result, &initialized) == nil && initialized.WorkspaceRoot != "" {
			v.root = initialized.WorkspaceRoot
			v.discover(v.profile, v.root, v.options)
		}
		return
	}
	if request.method != "tools/call" {
		return
	}
	v.validateToolResult(request.tool, response.Result)
}

func schemasFromResult(raw json.RawMessage) (map[string]*jsonschema.Schema, error) {
	var result struct {
		Tools []struct {
			Name         string          `json:"name"`
			OutputSchema json.RawMessage `json:"outputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.Tools == nil {
		return nil, fmt.Errorf("result omitted tools")
	}
	schemas := make(map[string]*jsonschema.Schema, len(result.Tools))
	for _, tool := range result.Tools {
		if tool.Name == "" || len(tool.OutputSchema) == 0 {
			return nil, fmt.Errorf("tool omitted name or outputSchema")
		}
		schema, err := compileSchema(tool.OutputSchema)
		if err != nil {
			return nil, fmt.Errorf("%s outputSchema: %w", tool.Name, err)
		}
		schemas[tool.Name] = schema
	}
	return schemas, nil
}

func (v *responseValidator) validateToolResult(tool string, raw json.RawMessage) {
	var result struct {
		IsError           bool            `json:"isError"`
		StructuredContent json.RawMessage `json:"structuredContent"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		v.report(fmt.Sprintf("%s: %s tool result is not an object: %v", v.testName, tool, err))
		return
	}
	if len(result.StructuredContent) == 0 || string(result.StructuredContent) == "null" {
		v.report(fmt.Sprintf("%s: %s isError=%t response omitted structuredContent", v.testName, tool, result.IsError))
		return
	}
	v.mu.Lock()
	schema := v.advertised[tool]
	fallback := false
	if schema == nil {
		schema = v.full[tool]
		fallback = schema != nil
	}
	catalogErr := v.catalogErr
	v.mu.Unlock()
	if schema == nil {
		v.report(fmt.Sprintf("%s: %s response has no advertised outputSchema (catalog error: %v)", v.testName, tool, catalogErr))
		return
	}
	var content any
	decoder := json.NewDecoder(bytes.NewReader(result.StructuredContent))
	decoder.UseNumber()
	if err := decoder.Decode(&content); err != nil {
		v.report(fmt.Sprintf("%s: %s structuredContent is invalid JSON: %v", v.testName, tool, err))
		return
	}
	v.validateStatusConsistency(tool, result.IsError, content)
	if err := schema.Validate(content); err != nil {
		catalog := "advertised"
		if fallback {
			catalog = "full-catalog fallback"
		}
		v.report(fmt.Sprintf("%s: %s structuredContent violates %s outputSchema: %s", v.testName, tool, catalog, validationFailureDetail(err)))
	}
}

func validationFailureDetail(err error) string {
	if validationError, ok := errors.AsType[*jsonschema.ValidationError](err); ok {
		if details, marshalErr := json.Marshal(validationError.DetailedOutput()); marshalErr == nil {
			return string(details)
		}
	}
	return err.Error()
}

func (v *responseValidator) validateStatusConsistency(tool string, isError bool, content any) {
	object, ok := content.(map[string]any)
	if !ok {
		return
	}
	if _, hasError := object["error"]; hasError {
		if !isError {
			v.report(fmt.Sprintf("%s: %s structuredContent has an error but isError is false", v.testName, tool))
		}
		return
	}
	native, ok := object["result"].(map[string]any)
	if !ok {
		return
	}
	if tool == "semantic_verify" {
		failure, hasFailure := native["failure"].(string)
		hasNativeFailure := hasFailure && failure != ""
		if hasNativeFailure != isError {
			v.report(fmt.Sprintf("%s: %s native failure presence=%t disagrees with isError=%t", v.testName, tool, hasNativeFailure, isError))
		}
	}
	if tool != "semantic_batch" {
		return
	}
	status, _ := native["status"].(string)
	failureStatus := status == "error" || status == "applied_with_diagnostics"
	if failureStatus != isError {
		v.report(fmt.Sprintf("%s: semantic_batch status %q disagrees with isError=%t", v.testName, status, isError))
	}
	if status == "error" {
		if _, exists := native["diagnostic_delta"]; exists {
			v.report(fmt.Sprintf("%s: semantic_batch error response contains diagnostic_delta", v.testName))
		}
	}
	rows, ok := native["results"].([]any)
	if !ok {
		return
	}
	for index, value := range rows {
		row, ok := value.(map[string]any)
		if !ok {
			continue
		}
		rowStatus, _ := row["status"].(string)
		if rowStatus == "ok" {
			if message, _ := row["error"].(string); message != "" {
				v.report(fmt.Sprintf("%s: semantic_batch result row %d has status ok and an error", v.testName, index))
			}
		}
	}
}
