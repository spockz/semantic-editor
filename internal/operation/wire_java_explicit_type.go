// Package operation registers native Java inferred-type expansion.
// WHY: This distinct intent does not imply arbitrary type migration.
package operation

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

const javaMakeTypeExplicitOperation backend.Operation = "make_type_explicit"
const canonicalMakeTypeExplicitKey = "refactor_make_type_explicit"

type makeTypeExplicitReq struct {
	Project          backend.ProjectContext
	File             string
	Symbol           string
	InFunction       string
	ExpectedRevision string
}

func (r makeTypeExplicitReq) GetProjectContext() backend.ProjectContext         { return r.Project }
func (r *makeTypeExplicitReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var makeTypeExplicitParams = readRequestParams(
	ParameterContract{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Required: true, Description: "Target: Java source file containing the inferred local", SourceFields: []string{sourceInspectFile, sourceRenameFile}},
	ParameterContract{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Role: RoleTarget, Required: true, Description: "Target: unique local var binding name inside the selected method"},
	ParameterContract{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Required: true, Description: "Selector: exact signature-qualified Java method identity", SourceFields: []string{sourceInspectInFunction}},
	ParameterContract{Name: wireExpectedRevision, CLIName: wireCLIExpectedRevision, JSONName: wireExpectedRevision, Type: ParamString, Role: RoleTarget, Required: true, Description: "Guard: full selected-file SHA-256 revision copied from a fresh inspection", SourceFields: []string{sourceInspectExpectedRevision}},
)

func parseMakeTypeExplicit(raw map[string]any) (makeTypeExplicitReq, error) {
	var req makeTypeExplicitReq
	if err := CheckParams(raw, makeTypeExplicitParams); err != nil {
		return req, err
	}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return req, err
	}
	if req.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, true); err != nil {
		return req, err
	}
	if req.ExpectedRevision, err = ParseString(raw, wireExpectedRevision, wireCLIExpectedRevision, true); err != nil {
		return req, err
	}
	if req.Project, err = parseReadProject(raw, req.File); err != nil {
		return req, err
	}
	if req.Project.Language == "" {
		req.Project.Language = backend.LanguageJava
	}
	if req.Project.Language != backend.LanguageJava {
		return req, fmt.Errorf("make-type-explicit supports Java only: %w", ErrInvalidParams)
	}
	if err := javacst.ValidateJavaIdentifier(req.Symbol); err != nil {
		return req, fmt.Errorf("select Java inferred local: %w", err)
	}
	return req, nil
}

func runMakeTypeExplicitJava(ctx context.Context, cc CallContext, req makeTypeExplicitReq) (ExtractionResult, error) {
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.InFunction})
	if err != nil {
		return ExtractionResult{}, err
	}
	if inspection == nil || len(inspection.Matches) != 1 {
		return ExtractionResult{}, fmt.Errorf("java explicit-type owner %q must resolve to one method", req.InFunction)
	}
	method := inspection.Matches[0]
	if method.Kind != "method" || method.Signature == nil || method.QualifiedName != req.InFunction || filepath.Clean(resolveWorkPath(cc.WorkDir, method.File)) != filepath.Clean(file) {
		return ExtractionResult{}, fmt.Errorf("java explicit-type owner must be the exact signature-qualified method in the selected file")
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read Java explicit-type preimage: %w", err)
	}
	if structuralRevision(source) != req.ExpectedRevision || method.Revision != req.ExpectedRevision {
		return ExtractionResult{}, fmt.Errorf("java explicit-type source revision changed")
	}
	local, err := javacst.JavaInferredLocal([]byte(method.Source), req.Symbol)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("select Java inferred local: %w", err)
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{Operation: javaMakeTypeExplicitOperation, Project: project, File: file, Symbol: req.Symbol, EnclosingSymbol: req.InFunction, Range: offsetJavaRange(method.Range, local.Identifier), Kind: "refactor", ExpectedRevision: req.ExpectedRevision})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || result.File != file || len(result.Files) != 1 || result.Files[0].File != file || !bytes.Equal(result.Before, source) || bytes.Equal(result.After, source) {
		return ExtractionResult{}, fmt.Errorf("java explicit-type backend returned an unexpected or unchanged edit")
	}
	relative, err := relativeWorkPath(cc.WorkDir, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("derive Java explicit-type receipt path: %w", err)
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	revision := structuralRevision(result.After)
	return ExtractionResult{Edit: FileEditRes{File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol, InFunction: req.InFunction, Diff: astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)), Detail: "Made the Java local type explicit through JDT LS.", DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true}, Draft: EditReceiptDraft{Operation: string(javaMakeTypeExplicitOperation), Status: EditPersisted, ReadSet: readSet, PlannedChanges: planned, Continuations: []SymbolTarget{{File: relative, Symbol: req.InFunction, ExpectedRevision: &revision}}}}, nil
}

func makeTypeExplicitDef() Def[makeTypeExplicitReq, ExtractionResult] {
	return Def[makeTypeExplicitReq, ExtractionResult]{
		Key: canonicalMakeTypeExplicitKey, Summary: "Use this tool instead of manually changing a Java inferred local type. It replaces one uniquely named method-local var token with the denotable explicit type supplied by JDT LS. Requires an exact signature-qualified owner and fresh file revision. Only an enabled eager action titled Change type of '<local>' to '<type>' of kind refactor with one selected-file token edit is accepted. Fields, parameters, loop/lambda/nested-class scopes, modified declarations, ambiguous locals, non-denotable types, imports, commands, lazy actions, and extra edits are refused.",
		Params: makeTypeExplicitParams, Level: LevelFile, CLIName: "refactor-make-type-explicit", MCPName: canonicalMakeTypeExplicitKey, Parse: parseMakeTypeExplicit,
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, makeTypeExplicitReq) (ExtractionResult, error){backend.LanguageJava: runMakeTypeExplicitJava}, Format: formatExtractionResult,
		Output:         extractionReceiptOutput(semanticReplaceBodyToolName),
		FinalizeOutput: (finalizeJavaMethodActionOutput[makeTypeExplicitReq]), ProjectOutput: projectExtractionResult,
		ExampleRaw: map[string]any{"file": "src/Service.java", "symbol": "total", wireInFunction: "Service.count()", wireExpectedRevision: "<sha256>", "language": "java", wireTrustWorkspace: true}, BufferedBatch: true, Batchable: true,
	}
}
