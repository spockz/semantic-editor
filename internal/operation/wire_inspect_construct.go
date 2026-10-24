// Package operation exposes current construct candidates so callers can resolve nested replacement selections without source coordinates.
package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/javacst"
)

const (
	semanticInspectConstructToolName = "semantic_inspect_construct"
	wireExpectedSelectionRevision    = "expected_selection_revision"
	wireCLIExpectedSelectionRevision = "expected-selection-revision"
	wireCandidates                   = "candidates"
	wireEndConstructPath             = "end_construct_path"
)

// InspectConstructReq requests current candidates for a nested construct selector.
type InspectConstructReq struct {
	Project       backend.ProjectContext
	File          string
	InFunction    string
	Kind          backend.ConstructKind
	Discriminator string
}

// InspectConstructRes returns canonical candidate selectors and their opaque revision.
type InspectConstructRes struct {
	File              string                           `json:"file"`
	InFunction        string                           `json:"in_function"`
	Kind              backend.ConstructKind            `json:"kind"`
	Discriminator     string                           `json:"discriminator,omitempty"`
	Candidates        []astedit.ConstructCandidateInfo `json:"candidates"`
	SelectionRevision string                           `json:"expected_selection_revision"`
	ExpectedRevision  string                           `json:"expected_revision"`
}

// GetProjectContext returns the project selected for construct inspection dispatch.
func (r InspectConstructReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext applies the effective project selected by registry dispatch.
func (r *InspectConstructReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var inspectConstructParams = readRequestParams(
	ParameterContract{Name: "file", CLIName: "file", JSONName: "file", Description: "Target: Go source file containing the selected function or method", Type: ParamString, Role: RoleTarget, Required: true, SourceFields: sourceFileContinuations},
	ParameterContract{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Description: "Selector: enclosing Go function or method; the result returns its canonical owner", Type: ParamString, Role: RoleSelector, Required: true, SourceFields: sourceInFunctionContinuations},
	ParameterContract{Name: "kind", CLIName: "kind", JSONName: "kind", Description: "Nested construct kind with path-addressable candidates", Type: ParamString, Required: true, Role: RoleSelector, DynamicEnums: executableInspectConstructKinds},
	ParameterContract{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Description: "Selector: condition, case label, defer call, or exact statement/expression source; required for statement/expression", Type: ParamString, Role: RoleSelector, SourceFields: sourceDiscriminatorContinuations},
)

func parseInspectConstruct(raw map[string]any) (InspectConstructReq, error) {
	var request InspectConstructReq
	if err := CheckParams(raw, inspectConstructParams); err != nil {
		return request, err
	}
	var err error
	if request.File, err = ParseString(raw, "file", "file", true); err != nil {
		return request, err
	}
	if request.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, true); err != nil {
		return request, err
	}
	rawKind, err := ParseString(raw, "kind", "kind", true)
	if err != nil {
		return request, err
	}
	request.Kind = backend.ConstructKind(rawKind)
	if request.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, false); err != nil {
		return request, err
	}
	if request.Kind == backend.ConstructStatementOrExpr && strings.TrimSpace(request.Discriminator) == "" {
		return request, fmt.Errorf("discriminator is required for kind=%s: %w", request.Kind, ErrInvalidParams)
	}
	request.Project, err = parseReadProject(raw, request.File)
	if err != nil {
		return request, err
	}
	return request, nil
}

func inspectConstructRun(ctx context.Context, cc CallContext, request InspectConstructReq) (InspectConstructRes, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	set, err := astedit.InspectConstructCandidates(ctx, resolveWorkPath(cc.WorkDir, request.File), request.InFunction, request.Kind, request.Discriminator)
	if err != nil {
		return InspectConstructRes{}, err
	}
	return InspectConstructRes{
		File: displayWorkPath(cc.WorkDir, request.File), InFunction: set.InFunction, Kind: set.Kind,
		Discriminator: set.Discriminator, Candidates: set.Candidates, SelectionRevision: set.SelectionRevision,
		ExpectedRevision: set.SourceRevision,
	}, nil
}

func inspectConstructRunJava(ctx context.Context, cc CallContext, request InspectConstructReq) (InspectConstructRes, error) {
	project := effectiveProject(cc, request.Project)
	file := resolveWorkPath(cc.WorkDir, request.File)
	project.File = file
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: request.InFunction})
	if err != nil {
		return InspectConstructRes{}, err
	}
	if inspection == nil || len(inspection.Matches) != 1 {
		return InspectConstructRes{}, fmt.Errorf("java construct owner %q must resolve to one method in %s", request.InFunction, displayWorkPath(cc.WorkDir, request.File))
	}
	method := inspection.Matches[0]
	candidates, err := javacst.Candidates([]byte(method.Source), request.Kind, request.Discriminator)
	if err != nil {
		return InspectConstructRes{}, err
	}
	infos := make([]astedit.ConstructCandidateInfo, 0, len(candidates))
	for _, candidate := range candidates {
		infos = append(infos, astedit.ConstructCandidateInfo{ConstructPath: candidate.ConstructPath, EndConstructPath: candidate.ConstructPath, Syntax: candidate.Syntax})
	}
	owner := method.QualifiedName
	revision, err := javacst.SelectionRevision(owner, request.Kind, request.Discriminator, candidates)
	if err != nil {
		return InspectConstructRes{}, err
	}
	return InspectConstructRes{
		File: displayWorkPath(cc.WorkDir, request.File), InFunction: owner, Kind: request.Kind,
		Discriminator: request.Discriminator, Candidates: infos, SelectionRevision: revision,
		ExpectedRevision: method.Revision,
	}, nil
}

func formatInspectConstruct(result InspectConstructRes) (string, error) {
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format construct inspection result: %w", err)
	}
	return string(output), nil
}

func constructCandidateSchema() *ValueSchema {
	text := nonEmptyStringSchema()
	return &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		wireConstructPath: text, wireEndConstructPath: text, "syntax": text,
	}, Required: []string{wireConstructPath, wireEndConstructPath, "syntax"}}
}

func inspectConstructResultSchema() *ValueSchema {
	text := nonEmptyStringSchema()
	return &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"file": text, wireInFunction: text, "kind": text, wireDiscriminator: {Kind: SchemaString},
		wireCandidates: {Kind: SchemaArray, Items: constructCandidateSchema()}, wireExpectedSelectionRevision: text,
		wireExpectedRevision: text,
	}, Required: []string{"file", wireInFunction, "kind", wireCandidates, wireExpectedSelectionRevision, wireExpectedRevision}}
}

func inspectConstructDef() Def[InspectConstructReq, InspectConstructRes] {
	return Def[InspectConstructReq, InspectConstructRes]{
		Key: "inspect_construct", Summary: ("Use this tool instead of guessing nested construct selectors when current choices are unavailable after selection drift or a selector has multiple matches. Lists current candidates with canonical selector context, reusable construct_path and end_construct_path values, an opaque expected_selection_revision for the ordered candidate list, and the full-file expected_revision captured with those candidates."),
		Params: inspectConstructParams, Level: LevelFile, CLIName: "inspect-construct", MCPName: semanticInspectConstructToolName,
		ReadOnly: true, Batchable: true, BufferedBatch: true, Parse: parseInspectConstruct, NormalizeInput: NormalizeConstructKindInput,
		Handlers:   map[backend.LanguageID]func(context.Context, CallContext, InspectConstructReq) (InspectConstructRes, error){backend.LanguageGo: inspectConstructRun, backend.LanguageJava: inspectConstructRunJava},
		Format:     formatInspectConstruct,
		ExampleRaw: map[string]any{"file": wireExampleFile, wireInFunction: "Serve", "kind": "if", wireDiscriminator: "err != nil", "language": "go"},
		Output: &OperationOutputContract{ResultSchema: inspectConstructResultSchema(), Continuations: []OutputContinuation{
			{Path: "result.file", Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: []string{semanticReplaceConstructToolName, semanticExtractFunctionToolName, semanticInsertCaseToolName}},
			{Path: "result.in_function", Schema: nonEmptyStringSchema(), Role: RoleSelector, Consumers: []string{semanticReplaceConstructToolName, semanticExtractFunctionToolName, semanticInsertCaseToolName}},
			{Path: "result.kind", Schema: nonEmptyStringSchema(), Role: RoleSelector, Consumers: []string{semanticReplaceConstructToolName}},
			{Path: "result.discriminator", Schema: ValueSchema{Kind: SchemaString}, Role: RoleSelector, Consumers: []string{semanticReplaceConstructToolName, semanticInsertCaseToolName}, Optional: true},
			{Path: "result." + wireCandidates + "[]." + wireConstructPath, Schema: nonEmptyStringSchema(), Role: RoleSelector, Consumers: []string{semanticReplaceConstructToolName, semanticExtractFunctionToolName, semanticExtractVariableToolName, semanticInsertCaseToolName}},
			{Path: "result." + wireCandidates + "[]." + wireEndConstructPath, Schema: nonEmptyStringSchema(), Role: RoleSelector, Consumers: []string{semanticExtractVariableToolName}},
			{Path: "result." + wireExpectedSelectionRevision, Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: []string{semanticReplaceConstructToolName, semanticInsertCaseToolName}},
			{Path: "result." + wireExpectedRevision, Schema: nonEmptyStringSchema(), Role: RoleTarget, Consumers: []string{semanticExtractFunctionToolName, semanticExtractVariableToolName}},
		}},
		ProjectOutput: func(result InspectConstructRes) (any, error) { return result, nil },
	}
}

func registerInspectConstructOps(registry *Registry) error {
	return Register(registry, inspectConstructDef())
}

func executableInspectConstructKinds(candidate backend.Backend) []string {
	return executableKinds(candidate, backend.OperationInspectConstruct)
}

func supportsInspectableConstructKind(kind backend.ConstructKind) bool {
	return slices.Contains(astedit.GoInspectConstructKinds, kind)
}

func constructSelectionErrorOutput() *OperationErrorOutputContract {
	return &OperationErrorOutputContract{ResultSchema: &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"status":            {Kind: SchemaString, Enum: []any{"ambiguous", "invalid_path", "not_found", "selection_revision_mismatch"}},
		"application_state": {Kind: SchemaString, Enum: []any{"not_applied"}},
		"file":              nonEmptyStringSchema(), wireInFunction: nonEmptyStringSchema(), "kind": nonEmptyStringSchema(), wireDiscriminator: {Kind: SchemaString},
		wireCandidates: {Kind: SchemaArray, Items: constructCandidateSchema()}, wireExpectedSelectionRevision: nonEmptyStringSchema(),
	}, Required: []string{"status", "application_state", "file", wireInFunction, "kind", wireCandidates, wireExpectedSelectionRevision}}}
}

func projectConstructSelectionError(err error) (any, bool, error) {
	var selectionError *astedit.ConstructSelectionError
	if !errors.As(err, &selectionError) {
		return nil, false, nil
	}
	projected := map[string]any{
		"status": selectionError.Status, "application_state": "not_applied", "file": selectionError.File,
		wireInFunction: selectionError.Candidates.InFunction, "kind": string(selectionError.Candidates.Kind),
		wireCandidates: selectionError.Candidates.Candidates, wireExpectedSelectionRevision: selectionError.Candidates.SelectionRevision,
	}
	if selectionError.Candidates.Discriminator != "" {
		projected[wireDiscriminator] = selectionError.Candidates.Discriminator
	}
	return projected, true, nil
}

func annotateConstructSelectionError(err error, file string) {
	if selectionError, ok := errors.AsType[*astedit.ConstructSelectionError](err); ok {
		selectionError.File = file
	}
}
