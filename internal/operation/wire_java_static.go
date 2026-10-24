// Package operation registers the bounded Java make-static refactoring.
// WHY: This operation resolves one Java method before delegating its exact native action to JDT LS.
package operation

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

const (
	javaMakeStaticOperation backend.Operation = "make_static"
)

type makeStaticReq struct {
	Project backend.ProjectContext
	File    string
	Symbol  string
}

func (r makeStaticReq) GetProjectContext() backend.ProjectContext         { return r.Project }
func (r *makeStaticReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var makeStaticParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target: Java source file containing the method", Required: true, SourceFields: []string{sourceLookupFile, sourceRenameFile, "semantic_inspect_symbol.result.continuations[].file", "refactor_make_static.result.continuations[].file"}},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Role: RoleTarget, Description: "Target: exact Java method identity from semantic_lookup or semantic_inspect_symbol", Required: true, SourceFields: []string{sourceLookupSymbol, sourceRenameSymbol, "semantic_inspect_symbol.result.continuations[].symbol", "refactor_make_static.result.continuations[].symbol"}},
}, readProjectParams...)

func parseMakeStatic(raw map[string]any) (makeStaticReq, error) {
	var req makeStaticReq
	if err := CheckParams(raw, makeStaticParams); err != nil {
		return req, err
	}
	req.Project = backend.ProjectContext{Language: backend.LanguageJava}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return req, err
	}
	if req.Project, err = parseReadProject(raw, req.File); err != nil {
		return req, err
	}
	if req.Project.Language == "" {
		req.Project.Language = backend.LanguageJava
	}
	if req.Project.Language != backend.LanguageJava {
		return req, fmt.Errorf("refactor_make_static supports Java only: %w", ErrInvalidParams)
	}
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return req, err
	}
	return req, nil
}

func runMakeStaticJava(ctx context.Context, cc CallContext, req makeStaticReq) (ExtractionResult, error) {
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.Symbol})
	if err != nil {
		return ExtractionResult{}, err
	}
	if inspection == nil || len(inspection.Matches) != 1 {
		return ExtractionResult{}, fmt.Errorf("java method selector %q must resolve exactly once", req.Symbol)
	}
	method := inspection.Matches[0]
	if method.Kind != "method" || method.QualifiedName != req.Symbol || filepath.Clean(resolveWorkPath(cc.WorkDir, method.File)) != filepath.Clean(file) {
		return ExtractionResult{}, fmt.Errorf("java make-static target must be the exact selected method in %q", req.File)
	}
	if method.Name == "" {
		return ExtractionResult{}, fmt.Errorf("java make-static target has no simple method name")
	}
	nameMatches, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: method.Name})
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("check Java make-static method-name uniqueness: %w", err)
	}
	if nameMatches == nil || len(nameMatches.Matches) != 1 || nameMatches.Matches[0].Kind != "method" || nameMatches.Matches[0].QualifiedName != method.QualifiedName || filepath.Clean(resolveWorkPath(cc.WorkDir, nameMatches.Matches[0].File)) != filepath.Clean(file) {
		return ExtractionResult{}, fmt.Errorf("java make-static method name %q must resolve uniquely in the selected file before applying the action", method.Name)
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read Java make-static preimage: %w", err)
	}
	if structuralRevision(source) != method.Revision {
		return ExtractionResult{}, fmt.Errorf("java make-static source changed after method inspection")
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: javaMakeStaticOperation, Project: project, File: file,
		Range: method.SelectionRange, Kind: backend.CodeActionInlineKind, ExpectedRevision: method.Revision,
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || result.File != file || !filepath.IsAbs(result.File) || !bytes.Equal(result.Before, source) || bytes.Equal(result.After, source) {
		return ExtractionResult{}, fmt.Errorf("java make-static backend returned an unexpected or unchanged target edit")
	}
	relative, err := filepath.Rel(cc.WorkDir, file)
	if err != nil || !filepath.IsLocal(relative) {
		return ExtractionResult{}, fmt.Errorf("derive Java make-static receipt path for %q: %w", file, err)
	}
	relative = filepath.ToSlash(relative)
	revision := structuralRevision(result.After)
	postInspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: method.Name})
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("resolve Java make-static method after edit: %w", err)
	}
	if postInspection == nil || len(postInspection.Matches) != 1 {
		return ExtractionResult{}, fmt.Errorf("java make-static post-edit method %q must resolve exactly once", method.Name)
	}
	postMethod := postInspection.Matches[0]
	if postMethod.Kind != "method" || postMethod.Name != method.Name || filepath.Clean(resolveWorkPath(cc.WorkDir, postMethod.File)) != filepath.Clean(file) {
		return ExtractionResult{}, fmt.Errorf("java make-static post-edit identity does not resolve to the selected method")
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	return ExtractionResult{
		Edit: FileEditRes{
			File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
			InFunction:       postMethod.QualifiedName,
			Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
			Detail:           "Made the Java method static through the JDT LS Make Static action.",
			DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true,
		},
		Draft: EditReceiptDraft{Operation: string(javaMakeStaticOperation), Status: EditPersisted, ReadSet: readSet, PlannedChanges: planned,
			Continuations: []SymbolTarget{{File: relative, Symbol: postMethod.QualifiedName, ExpectedRevision: &revision}}},
	}, nil
}

func makeStaticDef() Def[makeStaticReq, ExtractionResult] {
	return Def[makeStaticReq, ExtractionResult]{
		Key:     canonicalMakeStaticKey,
		Summary: "Use this tool instead of manually adding a static modifier to one resolved Java method. It requires the method name to be unique in the selected file, selects the exact enabled JDT LS Make Static action of kind refactor.inline, captures and validates its workspace edit, then formats and verifies affected Java files. Overloads, same-name methods in another class, non-methods, disabled actions, and unavailable targets are refused.",
		Params:  makeStaticParams, Level: LevelSymbol, CLIName: "refactor-make-static", MCPName: canonicalMakeStaticKey,
		Parse:          parseMakeStatic,
		Handlers:       map[backend.LanguageID]func(context.Context, CallContext, makeStaticReq) (ExtractionResult, error){backend.LanguageJava: runMakeStaticJava},
		Format:         formatExtractionResult,
		Output:         extractionReceiptOutput(semanticRenameToolName, semanticReplaceBodyToolName, canonicalMakeStaticKey),
		FinalizeOutput: (finalizeJavaMethodActionOutput[makeStaticReq]),
		ProjectOutput:  projectExtractionResult,
		ExampleRaw:     map[string]any{"file": "src/Service.java", "symbol": "Service.count()"},
		BufferedBatch:  true, Batchable: true,
	}
}
