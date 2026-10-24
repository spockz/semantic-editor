// Package operation validates source owners through native extraction preflights
// while leaving helper names supplied by the caller unchanged.
package operation

import (
	"context"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
)

func extractionTargetValidator(key string, raw map[string]any, selectorKey string) nativeTargetValidator {
	switch key {
	case string(backend.OperationExtractFunction):
		return func(ctx context.Context, _ CallContext, project backend.ProjectContext, selector string) error {
			retry := cloneRaw(raw)
			retry[selectorKey] = selector
			request, err := parseExtractFunction(retry)
			if err != nil {
				return err
			}
			return astedit.ValidateFunctionExtractionTarget(ctx, resolveWorkPath(project.RootDir, project.File), request.InFunction, request.Symbol, astedit.ExtractOptions{
				Kind: request.Kind, StartLine: request.StartLine, EndLine: request.EndLine,
			})
		}
	case string(backend.OperationExtractVariable):
		return func(ctx context.Context, _ CallContext, project backend.ProjectContext, selector string) error {
			retry := cloneRaw(raw)
			retry[selectorKey] = selector
			request, err := parseExtractVariable(retry)
			if err != nil {
				return err
			}
			return astedit.ValidateVariableExtractionTarget(ctx, resolveWorkPath(project.RootDir, project.File), request.InFunction, request.Symbol, astedit.ExtractVariableOptions{
				Discriminator: request.Discriminator, ConstructPath: request.ConstructPath, EndConstructPath: request.EndConstructPath, ExpectedRevision: request.ExpectedRevision,
			})
		}
	}
	return nil
}
