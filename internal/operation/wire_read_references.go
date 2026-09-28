// Package operation wires typed Go references into the shared CLI and MCP registry.
package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"semedit/internal/backend"
	"semedit/internal/backends"
	"semedit/internal/capability"
)

// ReferencesReq carries target and project selection for typed reference search.
type ReferencesReq struct {
	Project backend.ProjectContext
	Symbol  string
}

// GetProjectContext returns the request project context for dispatcher merging.
func (request ReferencesReq) GetProjectContext() backend.ProjectContext {
	return request.Project
}

// SetProjectContext applies dispatcher project context to the request.
func (request *ReferencesReq) SetProjectContext(project backend.ProjectContext) {
	request.Project = project
}

var referencesParams = readRequestParams(
	ParameterContract{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Description: "Target: qualified symbol identifier (e.g. 'Store.Read' or 'Widget')", Type: ParamString, Required: true, SourceFields: []string{sourceLookupSymbol, sourceRenameSymbol, sourceReplaceBodySymbol}},
	ParameterContract{Name: "file", CLIName: "file", JSONName: "file", Description: "Selector: optional source file containing the target declaration; search still covers the selected Go module", Type: ParamString, SourceFields: []string{sourceLookupFile, sourceRenameFile, sourceReplaceBodyFile, sourceReplaceConstructFile}},
)

func parseReferences(raw map[string]any) (ReferencesReq, error) {
	if err := CheckParams(raw, referencesParams); err != nil {
		return ReferencesReq{}, err
	}
	symbolName, err := ParseString(raw, "symbol", "symbol", true)
	if err != nil {
		return ReferencesReq{}, err
	}
	file, err := ParseString(raw, "file", "file", false)
	if err != nil {
		return ReferencesReq{}, err
	}
	project, err := parseReadProject(raw, file)
	if err != nil {
		return ReferencesReq{}, err
	}
	return ReferencesReq{Project: project, Symbol: symbolName}, nil
}

func referencesRun(ctx context.Context, cc CallContext, request ReferencesReq) (*backend.ReferencesResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	return service.FindReferences(ctx, backend.ReferencesRequest{
		Project: effectiveProject(cc, request.Project),
		Symbol:  request.Symbol,
	})
}

func formatReferences(result *backend.ReferencesResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format references result: %w", err)
	}
	return string(data), nil
}

func referencesDef() Def[ReferencesReq, *backend.ReferencesResult] {
	return Def[ReferencesReq, *backend.ReferencesResult]{
		Key:     capability.OpFindReferences,
		Summary: "Use this tool instead of text search when finding typed references to a Go declaration within its active module build. The optional file selects the target declaration; results span the module, including test variants.",
		Params:  referencesParams, Level: LevelSymbol, CLIName: "find-references", MCPName: "semantic_find_references",
		ReadOnly: true, Batchable: false, Parse: parseReferences,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ReferencesReq) (*backend.ReferencesResult, error){
			backend.LanguageGo: referencesRun,
		},
		Format:     formatReferences,
		ExampleRaw: map[string]any{"symbol": "Store.Read", "file": "api/store.go", "language": "go"},
	}
}
