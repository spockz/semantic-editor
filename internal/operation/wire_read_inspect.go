// Package operation defines the public request schemas and read-only handlers for Go inspect and outline.
package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/capability"
)

// OutlineReq requests a filtered Go source outline.
type OutlineReq struct {
	Project           backend.ProjectContext
	Path              string
	Kinds             []string
	IncludeUnexported bool
	IncludeTests      bool
}

// GetProjectContext returns the project selected for outline dispatch.
func (r OutlineReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext applies the effective project selected by registry dispatch.
func (r *OutlineReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

// InspectReq requests exact source metadata for one Go symbol.
type InspectReq struct {
	Project backend.ProjectContext
	Symbol  string
}

// GetProjectContext returns the project selected for inspect dispatch.
func (r InspectReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext applies the effective project selected by registry dispatch.
func (r *InspectReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var inspectReadParams = readRequestParams(
	ParameterContract{
		Name: "symbol", CLIName: "symbol", JSONName: "symbol", Description: "Target: qualified Go symbol name to inspect", Type: ParamString, Role: RoleTarget, Required: true,
		SourceFields: inspectReadSourceFields(sourceSymbolContinuations, []string{
			sourceExtractCallableReceiptSymbol, sourceExtractValueReceiptSymbol, sourceInlineValueReceiptSymbol,
			sourceContractExtractSymbol, sourceContractImplementSymbol, sourceConcreteTargetSymbol, sourceGeneratedMethodSymbol,
			"refactor_extract_type.result.edit_receipt.continuations[].symbol", sourceExtractDelegateReceiptSymbol,
			"refactor_make_static.result.continuations[].symbol",
		}, "symbol"),
	},
	ParameterContract{
		Name: "file", CLIName: "file", JSONName: "file", Description: "Target: optional source file that constrains symbol search scope", Type: ParamString, Role: RoleTarget,
		SourceFields: inspectReadSourceFields(sourceFileContinuations, []string{
			sourceContractExtractFile, sourceContractImplementFile, sourceConcreteTargetFile, sourceGeneratedMethodFile,
			"refactor_extract_type.result.edit_receipt.continuations[].file", sourceExtractDelegateReceiptFile,
			"refactor_make_static.result.continuations[].file",
		}, "file"),
	},
)

func inspectReadSourceFields(base, additional []string, signatureField string) []string {
	fields := append([]string(nil), base...)
	fields = append(fields, additional...)
	return append(fields, signatureReceiptSourceFields(signatureField)...)
}

const (
	wireIncludeTests    = "include_tests"
	wireCLIIncludeTests = "include-tests"
)

var outlineReadParams = readRequestParams(
	ParameterContract{Name: "path", CLIName: "path", JSONName: "path", InputAliases: []string{"file"}, Description: "Source file or directory to outline", Type: ParamString, Required: true},
	ParameterContract{Name: "kinds", CLIName: "kind", JSONName: "kinds", Description: "Optional kinds to include: type, interface, function, method, field, constant, variable, package, class, constructor, enum, record, module, property, object, enum_member, trait, target; known kinds absent from a backend produce empty symbols", Type: ParamStringSlice},
	ParameterContract{Name: "include_unexported", CLIName: "include-unexported", JSONName: "include_unexported", Description: "Include unexported Go declarations and non-public Java declarations (default true)", Type: ParamBoolean, Default: true},
	ParameterContract{Name: wireIncludeTests, CLIName: wireCLIIncludeTests, JSONName: wireIncludeTests, Description: "Include test-source files for Go and Java directory scopes (default false)", Type: ParamBoolean, Default: false},
)

func parseInspectRead(raw map[string]any) (InspectReq, error) {
	if err := CheckParams(raw, inspectReadParams); err != nil {
		return InspectReq{}, err
	}
	query, err := ParseString(raw, "symbol", "symbol", true)
	if err != nil {
		return InspectReq{}, err
	}
	file, err := ParseString(raw, "file", "file", false)
	if err != nil {
		return InspectReq{}, err
	}
	project, err := parseReadProject(raw, file)
	if err != nil {
		return InspectReq{}, err
	}
	return InspectReq{Project: project, Symbol: query}, nil
}

func parseOutlineRead(raw map[string]any) (OutlineReq, error) {
	if err := CheckParams(raw, outlineReadParams); err != nil {
		return OutlineReq{}, err
	}
	path, err := ParseString(raw, "path", "path", true)
	if err != nil {
		return OutlineReq{}, err
	}
	kinds, err := ParseStringSlice(raw, "kinds", "kind")
	if err != nil {
		return OutlineReq{}, err
	}
	includeUnexported, err := ParseBool(raw, "include_unexported", "include-unexported", true)
	if err != nil {
		return OutlineReq{}, err
	}
	includeTests, err := ParseBool(raw, wireIncludeTests, wireCLIIncludeTests, false)
	if err != nil {
		return OutlineReq{}, err
	}
	project, err := parseReadProject(raw, path)
	if err != nil {
		return OutlineReq{}, err
	}
	return OutlineReq{Project: project, Path: path, Kinds: kinds, IncludeUnexported: includeUnexported, IncludeTests: includeTests}, nil
}

func inspectReadRun(ctx context.Context, cc CallContext, request InspectReq) (*backend.InspectResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, request.Project)
	return service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: request.Symbol})
}

func outlineReadRun(ctx context.Context, cc CallContext, request OutlineReq) (*backend.OutlineResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, request.Project)
	return service.Outline(ctx, backend.OutlineRequest{
		Project: project, Path: request.Path, Kinds: request.Kinds,
		IncludeUnexported: request.IncludeUnexported, IncludeTests: request.IncludeTests,
	})
}

func formatInspectRead(result *backend.InspectResult) (string, error) {
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format inspect result: %w", err)
	}
	return string(output), nil
}

func formatOutlineRead(result *backend.OutlineResult) (string, error) {
	return renderOutlineNavigation(result), nil
}

func inspectReadDef() Def[InspectReq, *backend.InspectResult] {
	projectionSchema := InspectProjectionSchema()
	return Def[InspectReq, *backend.InspectResult]{
		Key:      capability.OpInspect,
		Summary:  "Use this tool instead of reading source with shell commands when you need a declaration's exact source, metadata, and snapshot revision.",
		Params:   inspectReadParams,
		Level:    LevelSymbol,
		CLIName:  "inspect-symbol",
		MCPName:  semanticInspectSymbolToolName,
		ReadOnly: true,
		Parse:    parseInspectRead,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, InspectReq) (*backend.InspectResult, error){
			backend.LanguageGo:   inspectReadRun,
			backend.LanguageJava: inspectReadRun,
			backend.LanguageMake: inspectReadRun,
		},
		Format:     formatInspectRead,
		ExampleRaw: map[string]any{"symbol": "Server.Start", "language": "go", wireTrustWorkspace: false},
		Output: &OperationOutputContract{
			ResultSchema: &projectionSchema,
			Continuations: []OutputContinuation{
				{Path: "result.continuations[].file", Schema: ValueSchema{Kind: SchemaString}, Role: RoleTarget, Consumers: []string{semanticReplaceBodyToolName, semanticInlineFunctionToolName, semanticMigrateTypeToolName, insertFieldToolName, semanticInsertConstructToolName, extractTypeToolName, extractDelegateToolName, semanticExtractFunctionToolName, signatureChangeConsumer, canonicalMakeStaticKey, canonicalPromoteLocalToConstKey, canonicalMakeTypeExplicitKey}},
				{Path: "result.continuations[].symbol", Schema: ValueSchema{Kind: SchemaString}, Role: RoleTarget, Consumers: []string{semanticReplaceBodyToolName, semanticInlineFunctionToolName, semanticMigrateTypeToolName, extractDelegateToolName, signatureChangeConsumer, canonicalMakeStaticKey, canonicalPromoteLocalToConstKey}},
				{Path: "result.continuations[].expected_revision", Schema: ValueSchema{Kind: SchemaString}, Role: RoleTarget, Consumers: []string{semanticReplaceBodyToolName, semanticInlineFunctionToolName, semanticMigrateTypeToolName, insertFieldToolName, semanticInsertConstructToolName, extractTypeToolName, extractDelegateToolName, semanticExtractFunctionToolName, signatureChangeConsumer, semanticExtractVariableToolName, canonicalPromoteLocalToConstKey, canonicalMakeTypeExplicitKey}},
				{Path: "result.continuations[].in_function", Schema: ValueSchema{Kind: SchemaString}, Role: RoleSelector, Consumers: []string{semanticReplaceConstructToolName, semanticExtractFunctionToolName, semanticExtractVariableToolName, semanticInlineVariableToolName, semanticInsertCaseToolName, semanticInsertConstructToolName, semanticInspectConstructToolName, semanticRemoveConstructToolName, canonicalPromoteLocalToConstKey, canonicalMakeTypeExplicitKey}},
				{Path: "result.continuations[].in_type", Schema: ValueSchema{Kind: SchemaString}, Role: RoleSelector, Consumers: []string{insertFieldToolName, extractTypeToolName, extractDelegateToolName, semanticInsertConstructToolName}},
				{Path: "result.continuations[].in_declaration", Schema: ValueSchema{Kind: SchemaString}, Role: RoleSelector, Consumers: []string{semanticInsertConstructToolName}},
			},
		},
		ProjectOutput: func(result *backend.InspectResult) (any, error) { return ProjectInspectResult(result) },
	}
}

func outlineReadDef() Def[OutlineReq, *backend.OutlineResult] {
	return Def[OutlineReq, *backend.OutlineResult]{
		Key: capability.OpOutline, Summary: ("Use this tool instead of scanning source files with shell commands when you need declarations for a selected source scope; use semantic_workspace_outline first to discover which packages to inspect. Returns one normalized declaration listing in text with workspace-relative file paths, inclusive declaration line spans, and native selectors separate from optional signatures. The listing appears once in text; structured output carries scope and file/declaration counts only. Use a listed file path and native selector with ordinary file-reading tools or semantic_inspect_symbol when that backend supports it. Go and Java support directory scopes."),
		Params: outlineReadParams, Level: LevelSymbol, CLIName: "outline", MCPName: "semantic_outline",
		ReadOnly: true, Parse: parseOutlineRead, PrepareProject: prepareOutlineProject,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, OutlineReq) (*backend.OutlineResult, error){
			backend.LanguageGo:     outlineReadRun,
			backend.LanguageJava:   outlineReadRun,
			backend.LanguageBash:   outlineReadRun,
			backend.LanguageMake:   outlineReadRun,
			backend.LanguageKotlin: outlineReadRun,
			backend.LanguageRust:   outlineReadRun,
			backend.LanguageScala:  outlineReadRun,
		},
		Format:        formatOutlineRead,
		Output:        &OperationOutputContract{ResultSchema: outlineOutputSchema()},
		ProjectOutput: projectOutlineOutput,
		ExampleRaw:    map[string]any{"path": "internal/astedit", "language": "go", "include_unexported": true, "include_tests": false},
	}
}

func prepareOutlineProject(project backend.ProjectContext) (backend.ProjectContext, error) {
	root, err := filepath.Abs(project.RootDir)
	if err != nil {
		return backend.ProjectContext{}, fmt.Errorf("resolve outline root %q: %w", project.RootDir, err)
	}
	target := project.File
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return backend.ProjectContext{}, fmt.Errorf("resolve outline path %q: %w", project.File, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return backend.ProjectContext{}, fmt.Errorf("stat outline path %q: %w", target, err)
	}
	if info.IsDir() {
		project.File = ""
	}
	return project, nil
}

func registerReadInspectOps(registry *Registry) error {
	if err := Register(registry, inspectReadDef()); err != nil {
		return err
	}
	return Register(registry, outlineReadDef())
}
