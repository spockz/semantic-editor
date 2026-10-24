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
	"github.com/spockz/semantic-editor/internal/operation"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Server wraps an MCP server and checks every captured structured tool result.
type Server struct {
	*mcp.Server

	validator *responseValidator
}

// Initialize delegates initialization and refreshes schemas for the resolved workspace root.
func (s *Server) Initialize(ctx context.Context, rawParams json.RawMessage) (map[string]any, error) {
	result, err := s.Server.Initialize(ctx, rawParams)
	if err != nil {
		return result, err
	}
	if root, ok := result["workspaceRoot"].(string); ok {
		s.validator.resetCatalog(root)
		s.validator.captureCatalogs()
	}
	return result, nil
}

// Serve preserves the underlying server's wire behavior while validating emitted tool results.
func (s *Server) Serve(ctx context.Context, input io.Reader) error {
	s.validator.captureCatalogs()
	err := s.Server.Serve(ctx, &requestReader{reader: input, validator: s.validator})
	s.validator.flush()
	return err
}

// NewServer creates an MCP test server that validates structured results against advertised schemas.
func NewServer(t testing.TB, profile, root string, output io.Writer, options ...mcp.Option) *Server {
	t.Helper()
	// Batch behavior tests opt in by default; tests of production defaults construct mcp.NewServer directly.
	testOptions := append([]mcp.Option{mcp.WithBatchingEnabled(true)}, options...)
	validator := &responseValidator{
		testName: t.Name(), report: func(message string) { t.Errorf("%s", message) },
		dst: output, profile: profile, options: slices.Clone(testOptions),
		requests:    make(map[string]requestInfo),
		advertised:  make(map[string]*jsonschema.Schema),
		full:        make(map[string]*jsonschema.Schema),
		catalogRoot: root,
	}
	server := &Server{validator: validator}
	server.Server = mcp.NewServer(profile, root, validator, testOptions...)
	return server
}

type requestInfo struct {
	method string
	tool   string
}

type responseValidator struct {
	testName        string
	report          func(string)
	dst             io.Writer
	profile         string
	options         []mcp.Option
	mu              sync.Mutex
	pending         []byte
	requests        map[string]requestInfo
	advertised      map[string]*jsonschema.Schema
	full            map[string]*jsonschema.Schema
	advertisedErr   error
	advertisedReady bool
	fullErr         error
	fullReady       bool
	catalogRoot     string
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

func (v *responseValidator) resetCatalog(root string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.catalogRoot = root
	v.advertised = make(map[string]*jsonschema.Schema)
	v.full = make(map[string]*jsonschema.Schema)
	v.advertisedErr = nil
	v.advertisedReady = false
	v.fullErr = nil
	v.fullReady = false
}

func (v *responseValidator) setAdvertised(schemas map[string]*jsonschema.Schema) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.advertised = schemas
	v.advertisedErr = nil
	v.advertisedReady = true
	// The full fallback belongs to this session's frozen languages until initialization.
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
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("decode tools/list response: %w", err)
	}
	if len(response.Error) != 0 {
		return nil, fmt.Errorf("tools/list returned JSON-RPC error: %s", response.Error)
	}
	return schemasFromResult(response.Result)
}

// CompileOutputSchema compiles one advertised MCP output schema for independent validation.
func CompileOutputSchema(raw json.RawMessage) (*jsonschema.Schema, error) {
	return compileSchema(raw)
}

func compileSchema(raw json.RawMessage) (*jsonschema.Schema, error) {
	content := string(raw)
	if entry, exists := compiledSchemaCache.Load(content); exists {
		return entry.(schemaCacheEntry).compile()
	}
	entry, _ := compiledSchemaCache.LoadOrStore(content, schemaCacheEntry{
		compile: sync.OnceValues(func() (*jsonschema.Schema, error) {
			return compileSchemaUncached(json.RawMessage(content))
		}),
	})
	return entry.(schemaCacheEntry).compile()
}

type schemaCacheEntry struct {
	compile func() (*jsonschema.Schema, error)
}

// Compiled schemas are shared read-only for the lifetime of the test process.
var compiledSchemaCache sync.Map

func compileSchemaUncached(raw json.RawMessage) (*jsonschema.Schema, error) {
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
			v.setAdvertised(schemas)
		}
		return
	}
	if request.method == "initialize" {
		var initialized struct {
			WorkspaceRoot string `json:"workspaceRoot"`
		}
		if json.Unmarshal(response.Result, &initialized) == nil && initialized.WorkspaceRoot != "" {
			v.resetCatalog(initialized.WorkspaceRoot)
			v.captureCatalogs()
		}
		return
	}
	if request.method == "tools/call" {
		v.validateToolResult(request.tool, response.Result)
	}
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
	catalogErr := v.advertisedErr
	fallback := false
	if schema == nil {
		schema = v.full[tool]
		fallback = schema != nil
		if catalogErr == nil {
			catalogErr = v.fullErr
		}
	}
	if schema == nil {
		if entry, ok := operation.DefaultRegistry().LookupMCP(tool); ok {
			schema = v.advertised[entry.MCPName]
			if schema == nil {
				schema = v.full[entry.MCPName]
				fallback = schema != nil
			}
		}
	}
	v.mu.Unlock()
	if catalogErr != nil {
		v.report(fmt.Sprintf("%s: %s response has no advertised outputSchema (catalog error: %v)", v.testName, tool, catalogErr))
		return
	}
	if schema == nil {
		v.report(fmt.Sprintf("%s: %s response has no advertised outputSchema", v.testName, tool))
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

// captureCatalogs freezes both schema views before calls can change detected languages.
func (v *responseValidator) captureCatalogs() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.catalogRoot == "" || v.advertisedReady && v.fullReady {
		return
	}
	v.advertised, v.advertisedErr = probeSchemas(v.profile, v.catalogRoot, v.options)
	v.advertisedReady = true
	fullOptions := append(slices.Clone(v.options), mcp.WithToolProfile("full"))
	v.full, v.fullErr = probeSchemas(v.profile, v.catalogRoot, fullOptions)
	v.fullReady = true
}
