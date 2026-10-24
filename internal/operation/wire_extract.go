// Package operation wires bounded Go extraction into the shared CLI and MCP registry.
package operation

import (
	"context"
	"fmt"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
)

// ExtractFunctionReq selects a contiguous statement region and names its helper.
type ExtractFunctionReq struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	InFunction       string
	Kind             astedit.ExtractKind
	StartLine        int
	EndLine          int
	ConstructPath    string
	EndConstructPath string
	ExpectedRevision string
}

// GetProjectContext returns the request project for registry dispatch.
func (r ExtractFunctionReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ExtractFunctionReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

var extractFunctionParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: Go, Java, or Rust source file containing the selected callable", Required: true, SourceFields: append(sourceFileContinuations, semanticInspectConstructToolName+".result.file")},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: fresh simple identifier for the helper declaration to create", Required: true},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Description: "Selector: qualified owner function or method containing the statements", Required: true, SourceFields: append(sourceInFunctionContinuations, semanticInspectConstructToolName+".result.in_function")},
	{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Helper declaration form", Required: true, Enums: []string{"function", "method"}},
	{Name: "start_line", CLIName: "start-line", JSONName: "start_line", Type: ParamInteger, Description: "First selected statement's starting line; use with end_line when no construct_path is supplied"},
	{Name: "end_line", CLIName: "end-line", JSONName: "end_line", Type: ParamInteger, Description: "Last selected statement's ending line; use with start_line when no construct_path is supplied"},
	{Name: wireConstructPath, CLIName: wireCLIConstructPath, JSONName: wireConstructPath, Type: ParamString, Role: RoleSelector, Description: "Selector: zero-based statement path from semantic_inspect_construct; Go paths may descend through supported lexical blocks and require expected_revision", SourceFields: []string{semanticInspectConstructToolName + ".result.candidates[].construct_path"}},
	{Name: "end_construct_path", CLIName: "end-construct-path", JSONName: "end_construct_path", Type: ParamString, Role: RoleSelector, Description: "Selector: inclusive final statement path copied from a candidate in semantic_inspect_construct; both endpoints must be in the same block and it defaults to construct_path"},
	{Name: expectedRevisionJSONName, CLIName: "expected-revision", JSONName: expectedRevisionJSONName, Type: ParamString, Role: RoleTarget, Description: "Guard: full selected-file SHA-256 revision from the same semantic_inspect_construct result as the candidate path, or from semantic_inspect_symbol; required with construct_path", SourceFields: []string{semanticInspectConstructToolName + ".result.expected_revision", sourceInspectExpectedRevision}},
}, readProjectParams...)

func parseExtractFunction(raw map[string]any) (ExtractFunctionReq, error) {
	var req ExtractFunctionReq
	if err := CheckParams(raw, extractFunctionParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Project, err = parseReadProject(raw, req.File); err != nil {
		return req, err
	}
	if req.Project.Language == "" {
		req.Project.Language = backend.LanguageGo
	}
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return req, err
	}
	if req.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, true); err != nil {
		return req, err
	}
	kind, err := ParseEnum(raw, "kind", "kind", []string{"function", "method"}, true, "")
	if err != nil {
		return req, err
	}
	req.Kind = astedit.ExtractKind(kind)
	if req.StartLine, err = ParseInteger(raw, "start_line", "start-line", false); err != nil {
		return req, err
	}
	if req.EndLine, err = ParseInteger(raw, "end_line", "end-line", false); err != nil {
		return req, err
	}
	if req.ConstructPath, err = ParseString(raw, wireConstructPath, wireCLIConstructPath, false); err != nil {
		return req, err
	}
	if req.EndConstructPath, err = ParseString(raw, "end_construct_path", "end-construct-path", false); err != nil {
		return req, err
	}
	if req.ExpectedRevision, err = ParseString(raw, expectedRevisionJSONName, "expected-revision", false); err != nil {
		return req, err
	}
	if value, present := rawLookup(raw, expectedRevisionJSONName, "expected-revision"); present && value != nil && req.ExpectedRevision == "" {
		return req, fmt.Errorf("param %q must not be empty when supplied: %w", expectedRevisionJSONName, ErrInvalidParams)
	}
	if req.Project.Language != backend.LanguageGo && (req.ConstructPath != "" || req.EndConstructPath != "" || req.ExpectedRevision != "") {
		return req, fmt.Errorf("revision-bound construct selection is currently supported only for Go callable extraction: %w", ErrInvalidParams)
	}
	switch {
	case req.ConstructPath != "":
		if req.ExpectedRevision == "" {
			return req, fmt.Errorf("expected_revision is required with construct_path: %w", ErrInvalidParams)
		}
		if req.StartLine != 0 || req.EndLine != 0 {
			return req, fmt.Errorf("construct_path selection cannot include start_line or end_line: %w", ErrInvalidParams)
		}
	case req.EndConstructPath != "":
		return req, fmt.Errorf("end_construct_path requires construct_path: %w", ErrInvalidParams)
	case req.StartLine <= 0 || req.EndLine < req.StartLine:
		return req, fmt.Errorf("start_line and end_line must describe a positive ordered line range: %w", ErrInvalidParams)
	}
	return req, nil
}

func runExtractFunction(ctx context.Context, cc CallContext, req ExtractFunctionReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ExtractFunction(ctx, targetPath, req.InFunction, req.Symbol, astedit.ExtractOptions{
		Kind: req.Kind, StartLine: req.StartLine, EndLine: req.EndLine,
		ConstructPath: req.ConstructPath, EndConstructPath: req.EndConstructPath, ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{
		File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: req.InFunction, Diff: diff, Detail: "Extracted statements into a " + string(req.Kind) + ".",
		Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true,
	}, nil
}

func extractFunctionDef() Def[ExtractFunctionReq, ExtractionResult] {
	return Def[ExtractFunctionReq, ExtractionResult]{
		Key: canonicalExtractCallableKey, Summary: ("Use this tool instead of copying statements to extract a supported contiguous callable region. Go uses typed object flow to derive live-ins and live-outs, and supports direct calls, assignments, a complete scalar if/else, and bounded counted/range loops. Revision-bound paths may select same-block statements in explicit blocks, if/else bodies, counted/range-loop bodies, or direct-call ranges in one expression-switch case/default body, plus one direct simple assignment to an eligible pre-existing owner local or named owner parameter (never the receiver) when that value is read after the whole switch. The helper result is assigned back at the same clause location. Switch declarations, other statement kinds, targets without a post-switch read, cross-case ranges, type-switch/select clauses, unsupported control flow, unsafe aliases, and stale revisions refuse. Java delegates direct statement selections to JDT LS; Rust and Scala accept only their documented bounded direct-call actions.") + automaticVerificationGuidance,
		Params: extractFunctionParams, Level: LevelFile, CLIName: "refactor-extract-callable", MCPName: semanticExtractFunctionToolName,
		Parse:    parseExtractFunction,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ExtractFunctionReq) (ExtractionResult, error){backend.LanguageGo: runExtractFunctionWithReceipt, backend.LanguageJava: runExtractFunctionJavaWithReceipt, backend.LanguageRust: runExtractFunctionRustWithReceipt, backend.LanguageScala: runExtractFunctionScalaWithReceipt},
		Format:   formatExtractionResult,
		Output:   extractionReceiptOutput(semanticRenameToolName, semanticReplaceBodyToolName),
		FinalizeOutput: func(cc CallContext, req ExtractFunctionReq, result ExtractionResult) (ExtractionResult, error) {
			if req.Project.Language == backend.LanguageJava {
				return finalizeJavaExtractionOutput(cc, ExtractVariableReq{Project: req.Project, File: req.File, Symbol: req.Symbol, InFunction: req.InFunction}, result)
			}
			if req.Project.Language == backend.LanguageRust {
				return finalizeRustExtractionOutput(cc, ExtractVariableReq{Project: req.Project, File: req.File, Symbol: req.Symbol}, result)
			}
			if req.Project.Language == backend.LanguageScala {
				return finalizeScalaCallableExtractionOutput(cc, req, result)
			}
			return finalizeExtractionOutput(cc, result)
		},
		ProjectOutput: projectExtractionResult,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "symbol": "writeSummary", wireInFunction: "Server.Serve", "kind": "function", "start_line": 12, "end_line": 12},
		BufferedBatch: true, Batchable: true,
	}
}

func registerExtractOps(registry *Registry) error {
	if err := Register(registry, extractFunctionDef()); err != nil {
		return fmt.Errorf("register extraction operation: %w", err)
	}
	if err := Register(registry, extractVariableDef()); err != nil {
		return fmt.Errorf("register variable extraction operation: %w", err)
	}
	if err := Register(registry, promoteLocalToConstDef()); err != nil {
		return fmt.Errorf("register Rust promote-local-to-const operation: %w", err)
	}
	return nil
}

const expectedRevisionJSONName = "expected_revision"

// ExtractConstantReq selects a constant expression and names its new package constant.
type ExtractConstantReq struct {
	Project           backend.ProjectContext
	File              string
	Symbol            string
	InFunction        string
	Discriminator     string
	OccurrenceIndices []int
	ConstructPath     string
	ExpectedRevision  string
}

// GetProjectContext returns the requested language for dispatch.
func (r ExtractConstantReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext applies the language selected by request dispatch.
func (r *ExtractConstantReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

type extractionSelection struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	InFunction       string
	Discriminator    string
	ConstructPath    string
	ExpectedRevision string
}

func parseExtractionSelection(raw map[string]any, params []ParameterContract) (extractionSelection, error) {
	var selection extractionSelection
	if err := CheckParams(raw, params); err != nil {
		return selection, err
	}
	selection.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if selection.File, err = ParseString(raw, "file", "file", true); err != nil {
		return selection, err
	}
	if selection.Project, err = parseReadProject(raw, selection.File); err != nil {
		return selection, err
	}
	if selection.Project.Language == "" {
		selection.Project.Language = backend.LanguageGo
	}
	if selection.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return selection, err
	}
	if selection.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, true); err != nil {
		return selection, err
	}
	if selection.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, true); err != nil {
		return selection, err
	}
	if selection.ConstructPath, err = ParseString(raw, wireConstructPath, wireCLIConstructPath, false); err != nil {
		return selection, err
	}
	if selection.ExpectedRevision, err = ParseString(raw, expectedRevisionJSONName, "expected-revision", false); err != nil {
		return selection, err
	}
	if err := validateExtractionRevision(raw, selection); err != nil {
		return selection, err
	}
	return selection, nil
}

func validateExtractionRevision(raw map[string]any, selection extractionSelection) error {
	if value, present := rawLookup(raw, expectedRevisionJSONName, "expected-revision"); present && value != nil && selection.ExpectedRevision == "" {
		return fmt.Errorf("param %q must not be empty when supplied: %w", expectedRevisionJSONName, ErrInvalidParams)
	}
	if selection.ConstructPath != "" && selection.ExpectedRevision == "" {
		return fmt.Errorf("param %q is required when construct_path is supplied: %w", expectedRevisionJSONName, ErrInvalidParams)
	}
	return nil
}

func runExtractConstant(ctx context.Context, cc CallContext, req ExtractConstantReq) (FileEditRes, error) {
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ExtractConstant(ctx, targetPath, req.InFunction, req.Symbol, astedit.ExtractConstantOptions{
		Discriminator: req.Discriminator, OccurrenceIndices: req.OccurrenceIndices, ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{
		File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: req.InFunction, Diff: diff, Detail: "Extracted a compile-time constant expression.",
		Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true,
	}, nil
}

// ExtractVariableReq selects one exact expression and names its new local.
type ExtractVariableReq struct {
	Project           backend.ProjectContext
	File              string
	Symbol            string
	Kind              string
	InFunction        string
	Discriminator     string
	OccurrenceIndices []int
	ConstructPath     string
	EndConstructPath  string
	ExpectedRevision  string
}

func extractConstantRequest(req ExtractVariableReq) ExtractConstantReq {
	return ExtractConstantReq{
		Project: req.Project, File: req.File, Symbol: req.Symbol, InFunction: req.InFunction,
		Discriminator: req.Discriminator, OccurrenceIndices: req.OccurrenceIndices, ConstructPath: req.ConstructPath, ExpectedRevision: req.ExpectedRevision,
	}
}

// GetProjectContext returns the request project for registry dispatch.
func (r ExtractVariableReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *ExtractVariableReq) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

var extractVariableParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: source file containing the selected Go or Java method", Required: true, SourceFields: sourceFileContinuations},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: fresh simple local identifier to create; choose a new name not already visible in the owner", Required: true},
	{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Extracted value form; defaults to variable. constant creates a constant where supported; field creates an owner field where supported.", Enums: []string{"variable", "constant", "field"}},
	{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Description: "Selector: qualified owner function or method containing the selected expression", Required: true, SourceFields: sourceInFunctionContinuations},
	{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Description: "Selector: exact source text of the expression to extract; Java and Scala expressions are selected inside the named method", Required: true},
	{Name: wireOccurrenceIndices, CLIName: "occurrence-indices", JSONName: wireOccurrenceIndices, Type: ParamArray, Role: RoleSelector, Description: "Go constant-only selector: sorted, unique, zero-based indices into exact discriminator matches in source order; requires expected_revision and cannot be combined with construct_path", Schema: occurrenceIndicesSchema()},
	{Name: wireConstructPath, CLIName: wireCLIConstructPath, JSONName: wireConstructPath, Type: ParamString, Role: RoleSelector, Description: "Selector: Go call, assignment, or return statement path from semantic_inspect_construct; a supplied path requires its matching expected_revision", SourceFields: []string{semanticInspectConstructToolName + ".result.candidates[].construct_path"}},
	{Name: wireEndConstructPath, CLIName: "end-construct-path", JSONName: wireEndConstructPath, Type: ParamString, Role: RoleSelector, Description: "Go-only selector: inclusive final direct-call or direct simple-assignment path for a contiguous owner-block range; nested assignments are unsupported; requires construct_path and the same expected_revision", SourceFields: []string{semanticInspectConstructToolName + ".result.candidates[]." + wireEndConstructPath}},
	{Name: expectedRevisionJSONName, CLIName: "expected-revision", JSONName: expectedRevisionJSONName, Type: ParamString, Role: RoleTarget, Description: "Guard: selected-file SHA-256 revision copied from semantic_inspect_construct or semantic_inspect_symbol; required with construct_path", SourceFields: []string{semanticInspectConstructToolName + ".result.expected_revision", sourceInspectExpectedRevision}},
}, readProjectParams...)

func parseExtractVariable(raw map[string]any) (ExtractVariableReq, error) {
	selection, err := parseExtractionSelection(raw, extractVariableParams)
	if err != nil {
		return ExtractVariableReq{}, err
	}
	kind, err := ParseEnum(raw, "kind", "kind", []string{"variable", "constant", "field"}, false, "variable")
	if err != nil {
		return ExtractVariableReq{}, err
	}
	if selection.Project.Language == backend.LanguageGo && kind == "constant" && selection.ConstructPath != "" {
		return ExtractVariableReq{}, fmt.Errorf("construct_path is not supported for Go constant extraction; use occurrence_indices for repeated expressions: %w", ErrInvalidParams)
	}
	occurrenceIndices, err := parseConstantOccurrenceIndices(raw)
	if err != nil {
		return ExtractVariableReq{}, err
	}
	if len(occurrenceIndices) > 0 {
		if selection.Project.Language != backend.LanguageGo || kind != "constant" {
			return ExtractVariableReq{}, fmt.Errorf("occurrence_indices is supported only for Go constant extraction: %w", ErrInvalidParams)
		}
		if selection.ExpectedRevision == "" {
			return ExtractVariableReq{}, fmt.Errorf("expected_revision is required with occurrence_indices: %w", ErrInvalidParams)
		}
		if selection.ConstructPath != "" {
			return ExtractVariableReq{}, fmt.Errorf("occurrence_indices cannot be combined with construct_path: %w", ErrInvalidParams)
		}
	}
	endConstructPath, err := ParseString(raw, "end_construct_path", "end-construct-path", false)
	if err != nil {
		return ExtractVariableReq{}, err
	}
	if endConstructPath != "" {
		if selection.Project.Language != backend.LanguageGo || kind != "variable" {
			return ExtractVariableReq{}, fmt.Errorf("end_construct_path is supported only for Go variable extraction: %w", ErrInvalidParams)
		}
		if selection.ConstructPath == "" {
			return ExtractVariableReq{}, fmt.Errorf("end_construct_path requires construct_path: %w", ErrInvalidParams)
		}
	}
	return ExtractVariableReq{Project: selection.Project, File: selection.File, Symbol: selection.Symbol, Kind: kind, InFunction: selection.InFunction, Discriminator: selection.Discriminator, OccurrenceIndices: occurrenceIndices, ConstructPath: selection.ConstructPath, EndConstructPath: endConstructPath, ExpectedRevision: selection.ExpectedRevision}, nil
}

func occurrenceIndicesSchema() *ValueSchema {
	minimum := 1
	return &ValueSchema{Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaInteger}, MinItems: &minimum}
}

func parseConstantOccurrenceIndices(raw map[string]any) ([]int, error) {
	value, present := rawLookup(raw, wireOccurrenceIndices, "occurrence-indices")
	if !present || value == nil {
		return nil, nil
	}
	var values []any
	switch typed := value.(type) {
	case []any:
		values = typed
	case []int:
		values = make([]any, len(typed))
		for index, value := range typed {
			values[index] = value
		}
	default:
		return nil, fmt.Errorf("param %q must be an array of integers: %w", wireOccurrenceIndices, ErrInvalidParams)
	}
	indices := make([]int, len(values))
	for index, value := range values {
		parsed, err := exactNativeInteger(value)
		if err != nil || parsed < 0 {
			return nil, fmt.Errorf("param %q entries must be nonnegative integers: %w", wireOccurrenceIndices, ErrInvalidParams)
		}
		if index > 0 && parsed <= indices[index-1] {
			return nil, fmt.Errorf("param %q entries must be strictly increasing and unique: %w", wireOccurrenceIndices, ErrInvalidParams)
		}
		indices[index] = parsed
	}
	return indices, nil
}

func runExtractVariable(ctx context.Context, cc CallContext, req ExtractVariableReq) (FileEditRes, error) {
	if req.EndConstructPath != "" && (req.Kind != "variable" || req.Project.Language != backend.LanguageGo) {
		return FileEditRes{}, fmt.Errorf("end_construct_path is supported only for Go variable extraction: %w", ErrInvalidParams)
	}
	if req.Kind == "constant" {
		return runExtractConstant(ctx, cc, ExtractConstantReq{Project: req.Project, File: req.File, Symbol: req.Symbol, InFunction: req.InFunction, Discriminator: req.Discriminator, OccurrenceIndices: req.OccurrenceIndices, ConstructPath: req.ConstructPath, ExpectedRevision: req.ExpectedRevision})
	}
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, err := astedit.ExtractVariable(ctx, targetPath, req.InFunction, req.Symbol, astedit.ExtractVariableOptions{
		Discriminator: req.Discriminator, ConstructPath: req.ConstructPath, EndConstructPath: req.EndConstructPath, ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		return FileEditRes{}, err
	}
	delta, diagnosticStatus := finishDelta()
	return FileEditRes{
		File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: req.InFunction, Discriminator: req.Discriminator, Diff: diff,
		Detail: "Extracted the selected expression into a local variable.", Delta: delta,
		DiagnosticStatus: diagnosticStatus, HasDelta: true,
	}, nil
}

func extractVariableDef() Def[ExtractVariableReq, ExtractionResult] {
	return Def[ExtractVariableReq, ExtractionResult]{
		Key: canonicalExtractValueKey, Summary: "Use this tool instead of manually rewriting a call or expression to extract an exact value. kind=variable extracts a local value; kind=constant creates a constant where supported. For repeated Go constants, occurrence_indices selects a revision-bound subset of exact matches; omitted indices do not mean replace-all. For repeated direct Go assignment RHS extraction, supply construct_path, end_construct_path, and their matching expected_revision; only contiguous one-target assignments with stable scalar inputs are supported. kind=field is unsupported by every current backend. Go supports side-effect-free typed expressions from direct call arguments, RHS expressions of simple assignments, single-name short declarations, or single-spec local var declarations with one name and one initializer in direct or nested statement blocks, whole direct or nested return expressions with exactly one explicit result slot (unnamed or one nonblank name), and complete if conditions without an init clause. The original return assigns a named result before deferred functions observe it. Multiple result slots, blank or multiply named result fields, naked returns, and untyped-only expressions refuse. An extracted if condition must be side-effect-free and contain a resolved typed input. Each local stays immediately before its selected statement in the same control-flow block. Grouped var declarations and multi-name or multi-value var specs refuse. When multiple assignment, var-initializer, return, or if-condition expressions match, select one with construct_path and its matching expected_revision. A selected Go call can replace all exact repeated arguments with one local only for stable scalar expressions whose inputs are not address-escaped explicitly or through implicit pointer-receiver method selection, or closure-captured. Java delegates local and constant extraction actions to JDT LS and renames the generated value; Rust accepts only suffixed integer arithmetic as the sole argument of a simple direct call; Scala accepts integer literal arithmetic only in a two-method object with a zero-argument expression-bodied Int owner and a strict Int identity callee. Java, Go, Rust, and Scala refuse kind=field. Java diagnostics are configured; Rust and Scala diagnostics are unavailable. Rust refuses ambiguous textual selections and construct_path; Scala accepts one exact discriminator and refuses construct_path; do not call semantic_verify separately.",
		Params: extractVariableParams, Level: LevelFile, CLIName: "refactor-extract-value", MCPName: semanticExtractVariableToolName,
		Parse: parseExtractVariable,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, ExtractVariableReq) (ExtractionResult, error){
			backend.LanguageGo:    runExtractValueWithReceipt,
			backend.LanguageJava:  runExtractValueJavaWithReceipt,
			backend.LanguageRust:  runExtractVariableRustWithReceipt,
			backend.LanguageScala: runExtractVariableScalaWithReceipt,
		},
		Format: formatExtractionResult,
		Output: extractionReceiptOutput(semanticRenameToolName, semanticReplaceBodyToolName, semanticReplaceDeclToolName),
		FinalizeOutput: func(cc CallContext, req ExtractVariableReq, result ExtractionResult) (ExtractionResult, error) {
			if req.Project.Language == backend.LanguageJava {
				return finalizeJavaExtractionOutput(cc, req, result)
			}
			if req.Project.Language == backend.LanguageRust {
				return finalizeRustExtractionOutput(cc, req, result)
			}
			if req.Project.Language == backend.LanguageScala {
				return finalizeScalaExtractionOutput(cc, req, result)
			}
			return finalizeExtractionOutput(cc, result)
		},
		ProjectOutput: projectExtractionResult,
		ExampleRaw:    map[string]any{"file": wireExampleFile, "symbol": "value", wireInFunction: "PrintValue", wireDiscriminator: "count + 1"},
		BufferedBatch: true, Batchable: true,
	}
}

const wireOccurrenceIndices = "occurrence_indices"
