// Package operation wires deterministic ADR search and index generation into both ingress surfaces.
package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spockz/semantic-editor/internal/adr"
	"github.com/spockz/semantic-editor/internal/backend"
)

type searchADRsReq struct {
	Query     string
	Role      string
	Scope     string
	Area      string
	Languages []string
	Offset    int
	Limit     int
}

type generateADRIndexReq struct{}

var searchADRsParams = []ParameterContract{
	{Name: "query", CLIName: "query", JSONName: "query", Type: ParamString, Description: "Decision context or search terms to match against ADR titles, summaries, descriptions, and rationale"},
	{Name: "role", CLIName: "role", JSONName: "role", Type: ParamString, Description: "Optional decision role label"},
	{Name: "scope", CLIName: "scope", JSONName: "scope", Type: ParamString, Description: "Optional decision scope label"},
	{Name: "area", CLIName: "area", JSONName: "area", Type: ParamString, Description: "Optional decision area label"},
	{Name: "languages", CLIName: "languages", JSONName: "languages", Type: ParamStringSlice, Description: "Optional open-ended language labels; repeat the flag to require multiple labels"},
	{Name: "offset", CLIName: "offset", JSONName: "offset", Type: ParamInteger, Description: "Zero-based result offset", Default: 0},
	{Name: "limit", CLIName: "limit", JSONName: "limit", Type: ParamInteger, Description: "Page size from 1 to 100 (default: 10)", Default: 10},
}

func parseSearchADRs(raw map[string]any) (searchADRsReq, error) {
	var req searchADRsReq
	if err := CheckParams(raw, searchADRsParams); err != nil {
		return req, err
	}
	var err error
	if req.Query, err = ParseString(raw, "query", "", false); err != nil {
		return req, err
	}
	if req.Role, err = ParseString(raw, "role", "", false); err != nil {
		return req, err
	}
	if req.Scope, err = ParseString(raw, "scope", "", false); err != nil {
		return req, err
	}
	if req.Area, err = ParseString(raw, "area", "", false); err != nil {
		return req, err
	}
	if req.Languages, err = ParseStringSlice(raw, "languages", "languages"); err != nil {
		return req, err
	}
	if req.Offset, err = ParseInteger(raw, "offset", "offset", false); err != nil {
		return req, err
	}
	if req.Limit, err = ParseInteger(raw, "limit", "limit", false); err != nil {
		return req, err
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	return req, nil
}

func runSearchADRs(_ context.Context, cc CallContext, req searchADRsReq) (adr.SearchPage, error) {
	records, err := adr.ReadAll(cc.WorkDir)
	if err != nil {
		return adr.SearchPage{}, err
	}
	return adr.Search(records, adr.SearchOptions{
		Query: strings.TrimSpace(req.Query), Role: strings.TrimSpace(req.Role),
		Scope: strings.TrimSpace(req.Scope), Area: strings.TrimSpace(req.Area),
		Languages: req.Languages, Offset: req.Offset, Limit: req.Limit,
	})
}

func projectADRSearch(page adr.SearchPage) (any, error) { return page, nil }

func searchADRsOutputSchema() *ValueSchema {
	items := new(outputObject(map[string]ValueSchema{
		"id": outputString(), "title": outputString(), "status": outputString(),
		"summary": outputString(), "description": outputString(), "role": outputString(),
		"scope": outputString(), "area": outputString(), "languages": {Kind: SchemaArray, Items: new(outputString())},
		"path": outputString(),
	}, "id", "title", "status", "summary", "description", "role", "scope", "area", "languages", "path"))
	return new(outputObject(map[string]ValueSchema{
		"total": {Kind: SchemaInteger}, "offset": {Kind: SchemaInteger}, "limit": {Kind: SchemaInteger},
		"items": {Kind: SchemaArray, Items: items},
	}, "total", "offset", "limit", "items"))
}

func parseGenerateADRIndex(raw map[string]any) (generateADRIndexReq, error) {
	if err := CheckParams(raw, nil); err != nil {
		return generateADRIndexReq{}, err
	}
	return generateADRIndexReq{}, nil
}

type generateADRIndexRes struct {
	Status string `json:"status"`
}

func runGenerateADRIndex(_ context.Context, cc CallContext, _ generateADRIndexReq) (generateADRIndexRes, error) {
	if err := adr.SyncIndex(cc.WorkDir); err != nil {
		return generateADRIndexRes{}, err
	}
	return generateADRIndexRes{Status: "synced"}, nil
}

func generateADRIndexDef() Def[generateADRIndexReq, generateADRIndexRes] {
	return Def[generateADRIndexReq, generateADRIndexRes]{
		Key: "generate_adr_index", Summary: "Use this tool instead of hand-editing the ADR register when validating ADR records and regenerating the marked decision register in docs/adr/README.md.",
		Level: LevelWorkspace, CLIName: "generate-adr-index", MCPName: "semantic_generate_adr_index",
		Parse:    parseGenerateADRIndex,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, generateADRIndexReq) (generateADRIndexRes, error){backend.LanguageAuto: runGenerateADRIndex},
		Format: func(result generateADRIndexRes) (string, error) {
			data, err := json.Marshal(result)
			if err != nil {
				return "", fmt.Errorf("serialize ADR index result: %w", err)
			}
			return string(data), nil
		},
		ProjectOutput: func(result generateADRIndexRes) (any, error) { return result, nil },
		Output:        &OperationOutputContract{ResultSchema: new(outputObject(map[string]ValueSchema{"status": {Kind: SchemaString, Enum: []any{"synced"}}}, "status"))},
		ExampleRaw:    map[string]any{},
	}
}

func searchADRsDef() Def[searchADRsReq, adr.SearchPage] {
	return Def[searchADRsReq, adr.SearchPage]{
		Key: "search_adrs", Summary: "Use this tool instead of scanning ADR files or the index when finding decisions applicable to a proposed decision; results are deterministic, paginated, and include workspace-relative ADR paths.",
		Params: searchADRsParams, Level: LevelWorkspace, CLIName: "search-adrs", MCPName: "semantic_search_adrs",
		Parse:    parseSearchADRs,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, searchADRsReq) (adr.SearchPage, error){backend.LanguageAuto: runSearchADRs},
		ReadOnly: true,
		Format: func(page adr.SearchPage) (string, error) {
			data, err := json.MarshalIndent(page, "", "  ")
			if err != nil {
				return "", fmt.Errorf("serialize ADR search results: %w", err)
			}
			return string(data), nil
		},
		ProjectOutput: projectADRSearch,
		Output:        &OperationOutputContract{ResultSchema: searchADRsOutputSchema()},
		ExampleRaw:    map[string]any{"query": "workspace validation", "limit": 10},
	}
}

func registerADROps(registry *Registry) error {
	if err := Register(registry, searchADRsDef()); err != nil {
		return err
	}
	return Register(registry, generateADRIndexDef())
}
