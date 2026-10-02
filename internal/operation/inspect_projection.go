// This file defines an explicit inspection projection so callers can reuse only captured, typed target evidence.

package operation

import (
	"context"
	"fmt"
	"path/filepath"
	"semedit/internal/backend"
	"semedit/internal/backends"
)

// InspectContinuation carries one captured inspection target into a later guarded operation.
type InspectContinuation struct {
	File             string  `json:"file"`
	Symbol           string  `json:"symbol"`
	ExpectedRevision string  `json:"expected_revision"`
	InFunction       *string `json:"in_function,omitempty"`
	InType           *string `json:"in_type,omitempty"`
	InDeclaration    *string `json:"in_declaration,omitempty"`
}

// InspectProjection keeps legacy inspection text beside explicit typed target evidence.
type InspectProjection struct {
	Inspection    string                `json:"inspection"`
	Continuations []InspectContinuation `json:"continuations"`
}

// InspectProjectionSchema declares the closed MCP projection shape for captured inspection targets.
func InspectProjectionSchema() ValueSchema {
	stringSchema := ValueSchema{Kind: SchemaString}
	continuationSchema := ValueSchema{
		Kind: SchemaObject,
		Properties: map[string]ValueSchema{
			"file":               stringSchema,
			"symbol":             stringSchema,
			wireExpectedRevision: stringSchema,
			wireInFunction:       stringSchema,
			"in_type":            stringSchema,
			wireInDeclaration:    stringSchema,
		},
		Required: []string{"file", "symbol", wireExpectedRevision},
	}
	return ValueSchema{
		Kind: SchemaObject,
		Properties: map[string]ValueSchema{
			"inspection":    stringSchema,
			"continuations": {Kind: SchemaArray, Items: &continuationSchema},
		},
		Required: []string{"inspection", "continuations"},
	}
}

// ProjectInspectResult preserves the legacy inspection text and projects only captured target evidence.
func ProjectInspectResult(result *backend.InspectResult) (InspectProjection, error) {
	if result == nil {
		return InspectProjection{}, fmt.Errorf("project inspection result: nil result")
	}
	inspection, err := formatInspectRead(result)
	if err != nil {
		return InspectProjection{}, fmt.Errorf("format captured inspection: %w", err)
	}
	projection := InspectProjection{Inspection: inspection, Continuations: make([]InspectContinuation, 0, len(result.Matches))}
	for index, match := range result.Matches {
		file := filepath.ToSlash(filepath.Clean(filepath.FromSlash(match.File)))
		if file == "." || !filepath.IsLocal(filepath.FromSlash(file)) {
			return InspectProjection{}, fmt.Errorf("inspection match %d has non-local file %q", index, match.File)
		}
		if match.QualifiedName == "" || match.Revision == "" {
			return InspectProjection{}, fmt.Errorf("inspection match %d lacks qualified identity or captured revision", index)
		}
		continuation := InspectContinuation{
			File:             file,
			Symbol:           match.QualifiedName,
			ExpectedRevision: match.Revision,
		}
		if filepath.Ext(file) == ".go" {
			switch match.Kind {
			case "function", "method":
				continuation.InFunction = &continuation.Symbol
				continuation.InDeclaration = &continuation.Symbol
			case "type", "interface":
				continuation.InType = &continuation.Symbol
				continuation.InDeclaration = &continuation.Symbol
			case "constant", "variable":
				continuation.InDeclaration = &continuation.Symbol
			}
		}
		projection.Continuations = append(projection.Continuations, continuation)
	}
	return projection, nil
}

func validateExpectedRevision(ctx context.Context, cc CallContext, file, symbol, expected string) error {
	if expected == "" {
		return fmt.Errorf("expected_revision is empty")
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := backend.ProjectContext{
		RootDir:        cc.WorkDir,
		File:           file,
		Language:       backend.LanguageGo,
		WorkspaceTrust: cc.Project.WorkspaceTrust,
	}
	result, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: symbol})
	if err != nil {
		return fmt.Errorf("inspect guarded target %q: %w", symbol, err)
	}
	requestedFile := filepath.ToSlash(filepath.Clean(filepath.FromSlash(file)))
	for _, match := range result.Matches {
		reportedFile := filepath.Clean(filepath.FromSlash(match.File))
		if filepath.IsAbs(reportedFile) {
			reportedFile, err = filepath.Rel(cc.WorkDir, reportedFile)
			if err != nil {
				return fmt.Errorf("relativize inspected target file: %w", err)
			}
		}
		if filepath.ToSlash(filepath.Clean(reportedFile)) == requestedFile && match.QualifiedName == symbol && match.Revision == expected {
			return nil
		}
	}
	return fmt.Errorf("expected_revision does not match captured source for %q", symbol)
}
