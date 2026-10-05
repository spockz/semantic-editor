// This file verifies SourceFields against declarative output contracts, including raw entries that bypass registry registration.

package mcp_test

import (
	"semedit/internal/mcp"
	"semedit/internal/operation"
	"testing"
)

func TestSourceFieldIndexValidatesMetadataEdgesAndRawEntryModes(t *testing.T) {
	integer := operation.ValueSchema{Kind: operation.SchemaInteger}
	stringSchema := operation.ValueSchema{Kind: operation.SchemaString}
	provider := operation.Entry{
		MCPName: "semantic_provider",
		Output: &operation.OperationOutputContract{
			Metadata: []operation.OutputMetadataField{{Field: "file", Schema: integer}},
		},
	}
	cases := []struct {
		name  string
		param operation.ParameterContract
	}{
		{
			name: "metadata cannot rename",
			param: operation.ParameterContract{
				Name: "destination_file", JSONName: "destination_file", Type: operation.ParamString,
				SourceFields: []string{"semantic_provider.file"},
			},
		},
		{
			name: "metadata schema must be assignable",
			param: operation.ParameterContract{
				Name: "file", JSONName: "file", Type: operation.ParamString,
				SourceFields: []string{"semantic_provider.file"},
			},
		},
		{
			name: "metadata is not a live continuation",
			param: operation.ParameterContract{
				Name: "file", JSONName: "file", Type: operation.ParamInteger, Role: operation.RoleTarget,
				SourceFields: []string{"semantic_provider.file"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			consumer := operation.Entry{MCPName: "semantic_consumer", Params: []operation.ParameterContract{tc.param}}
			if err := mcp.ValidateSourceFields([]operation.Entry{provider, consumer}); err == nil {
				t.Fatal("invalid metadata source edge was accepted")
			}
		})
	}

	for _, tc := range []struct {
		name  string
		entry operation.Entry
	}{
		{
			name: "metadata with projector",
			entry: operation.Entry{
				MCPName: "semantic_invalid_mode",
				Output:  provider.Output,
				ProjectOutput: func(any) (any, error) {
					return map[string]any{"result": "ok"}, nil
				},
			},
		},
		{
			name: "typed result without projector",
			entry: operation.Entry{
				MCPName: "semantic_invalid_mode",
				Output: &operation.OperationOutputContract{
					ResultSchema: &stringSchema,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := mcp.ValidateSourceFields([]operation.Entry{tc.entry}); err == nil {
				t.Fatal("raw entry with an invalid output mode was accepted")
			}
		})
	}
}

func TestSourceFieldIndexSkipsEntriesWithoutMCPNames(t *testing.T) {
	fileSchema := operation.ValueSchema{Kind: operation.SchemaString}
	resultSchema := operation.ValueSchema{
		Kind:       operation.SchemaObject,
		Properties: map[string]operation.ValueSchema{"file": fileSchema},
		Required:   []string{"file"},
	}
	provider := operation.Entry{
		MCPName: "semantic_provider",
		Output: &operation.OperationOutputContract{
			ResultSchema: &resultSchema,
			Continuations: []operation.OutputContinuation{{
				Field: "file", Schema: fileSchema, Role: operation.RoleTarget, Consumers: []string{"semantic_consumer"},
			}},
		},
		ProjectOutput: func(any) (any, error) {
			return map[string]any{"file": "selected.go"}, nil
		},
	}
	cliConsumer := operation.Entry{
		Key: "cli-consumer",
		Params: []operation.ParameterContract{{
			Name: "file", JSONName: "file", Type: operation.ParamString, Role: operation.RoleTarget,
			SourceFields: []string{"semantic_provider.file"},
		}},
	}
	entries := []operation.Entry{provider, {Key: "cli-first"}, cliConsumer, {Key: "cli-second"}}
	if err := mcp.ValidateSourceFields(entries); err != nil {
		t.Fatalf("validate CLI-only source consumer: %v", err)
	}
	invalidParams := operation.Entry{Key: "cli-invalid-params", Params: []operation.ParameterContract{{Name: "file", JSONName: "file", Type: operation.ParamObject}}}
	if err := mcp.ValidateSourceFields([]operation.Entry{invalidParams}); err == nil {
		t.Fatal("CLI-only parameter contracts were not validated")
	}
	invalidOutput := operation.Entry{
		Key:    "cli-invalid-output",
		Output: &operation.OperationOutputContract{ResultSchema: &fileSchema},
	}
	if err := mcp.ValidateSourceFields([]operation.Entry{invalidOutput}); err == nil {
		t.Fatal("CLI-only output contracts were not validated")
	}
}
