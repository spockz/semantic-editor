// Package operation registers and executes semantic editing operations.
// WHY: This file routes Java signature changes and records caller edits.
package operation

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func runChangeSignatureJava(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
	return runJavaSignatureChange(ctx, cc, raw, backend.OperationChangeSignature)
}

func runIntroduceParameterJava(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
	return runJavaSignatureChange(ctx, cc, raw, backend.OperationIntroduceParameter)
}

func runJavaSignatureChange(ctx context.Context, cc CallContext, raw map[string]any, operation backend.Operation) (structuralPlanResult, error) {
	file := stringParam(raw, "file")
	symbol := stringParam(raw, "symbol")
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project, err := parseReadProject(raw, file)
	if err != nil {
		return structuralPlanResult{}, err
	}
	if project.Language != backend.LanguageJava {
		return structuralPlanResult{}, fmt.Errorf("java change_signature requires language=java")
	}
	project = effectiveProject(cc, project)
	project.File = resolveWorkPath(cc.WorkDir, file)
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: symbol})
	if err != nil {
		return structuralPlanResult{}, err
	}
	if inspection == nil || len(inspection.Matches) != 1 || inspection.Matches[0].Kind != "method" {
		return structuralPlanResult{}, fmt.Errorf("java change_signature target %q must resolve to one method", symbol)
	}
	method := inspection.Matches[0]
	source, err := sourcebuf.ReadFile(ctx, project.File)
	if err != nil {
		return structuralPlanResult{}, fmt.Errorf("read Java signature preimage: %w", err)
	}
	if revisionErr := validateSignatureExpectedRevision(raw, source); revisionErr != nil {
		return structuralPlanResult{}, revisionErr
	}
	mappings, err := decodeSignatureMappings(raw[signatureParameters])
	if err != nil {
		return structuralPlanResult{}, err
	}
	parameters := javaCodeActionParameters(mappings)
	var expression, expressionPath, parameterName string
	if operation == backend.OperationIntroduceParameter {
		expression = stringParam(raw, wireDiscriminator)
		expressionPath = stringParam(raw, wireConstructPath)
		added, appendErr := appendedSignatureMapping(mappings)
		if appendErr != nil {
			return structuralPlanResult{}, appendErr
		}
		parameterName = added.Name
		if _, _, resolveErr := javacst.ResolveExpression([]byte(method.Source), expression, expressionPath); resolveErr != nil {
			return structuralPlanResult{}, fmt.Errorf("resolve Java expression for new parameter: %w", resolveErr)
		}
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation:        operation,
		Project:          project,
		File:             project.File,
		Range:            method.SelectionRange,
		Kind:             "refactor.change.signature",
		Symbol:           symbol,
		Parameters:       parameters,
		Expression:       expression,
		ExpressionPath:   expressionPath,
		ParameterName:    parameterName,
		ExpectedRevision: structuralRevision(source),
	})
	if err != nil {
		return structuralPlanResult{}, err
	}
	return javaSignatureChangeResult(cc, project, file, symbol, operation, result)
}

func javaCodeActionParameters(mappings []astedit.SignatureParameterMapping) []backend.CodeActionParameter {
	parameters := make([]backend.CodeActionParameter, 0, len(mappings))
	for _, mapping := range mappings {
		parameters = append(parameters, backend.CodeActionParameter{SourceIndex: mapping.SourceIndex, Name: mapping.Name, Type: mapping.Type, Argument: mapping.Argument})
	}
	return parameters
}

func javaSignatureChangeResult(cc CallContext, project backend.ProjectContext, file, symbol string, operation backend.Operation, result *backend.CodeActionResult) (structuralPlanResult, error) {
	if result == nil || result.File != project.File {
		return structuralPlanResult{}, fmt.Errorf("java change_signature backend returned an unexpected target file")
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return structuralPlanResult{}, err
	}
	if len(planned) == 0 {
		return structuralPlanResult{}, fmt.Errorf("java change_signature produced no workspace changes")
	}
	relative, err := filepath.Rel(cc.WorkDir, project.File)
	if err != nil || !filepath.IsLocal(relative) {
		return structuralPlanResult{}, fmt.Errorf("derive Java change_signature target path for %q: %w", project.File, err)
	}
	relative = filepath.ToSlash(relative)
	postRevision := structuralRevision(result.After)
	return structuralPlanResult{
		Edit: FileEditRes{
			File: project.File, Display: displayWorkPath(cc.WorkDir, file), Symbol: symbol,
			Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, file), string(result.Before), string(result.After)),
			Detail:           "Changed the Java method signature and updated JDT LS-resolved references.",
			DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true,
		},
		Draft: EditReceiptDraft{
			Operation: string(operation), Status: EditPersisted,
			ReadSet: readSet, PlannedChanges: planned,
			Continuations: []SymbolTarget{{File: relative, Symbol: symbol, ExpectedRevision: &postRevision}},
		},
	}, nil
}
