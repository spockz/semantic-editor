// Package operation keeps the bounded machine-readable projections for remaining public operations.
package operation

import (
	"fmt"

	"github.com/spockz/semantic-editor/internal/backend"
)

func outputObject(properties map[string]ValueSchema, required ...string) ValueSchema {
	return ValueSchema{Kind: SchemaObject, Properties: properties, Required: required}
}

func outputString() ValueSchema { return ValueSchema{Kind: SchemaString} }

func outputStringList() ValueSchema {
	item := outputString()
	return ValueSchema{Kind: SchemaArray, Items: &item}
}

func boundedStringList(maxItems int) ValueSchema {
	item := outputString()
	return ValueSchema{Kind: SchemaArray, Items: &item, MaxItems: &maxItems}
}

func snapshotOutputSchema() *ValueSchema {
	createdAt := outputString()
	files := boundedStringList(100)
	return new(outputObject(map[string]ValueSchema{
		"status":                   {Kind: SchemaString, Enum: []any{"created"}},
		outputSnapshotIDField:      outputString(),
		"created_at":               createdAt,
		"captured_files":           files,
		"captured_file_count":      {Kind: SchemaInteger},
		"captured_files_truncated": {Kind: SchemaBoolean},
	}, "status", outputSnapshotIDField, "created_at", "captured_files", "captured_file_count", "captured_files_truncated"))
}

func undoOutputSchema() *ValueSchema {
	files := boundedStringList(100)
	diagnostics := boundedStringList(20)
	deltaItems := boundedStringList(20)
	delta := outputObject(map[string]ValueSchema{
		"net_delta": {Kind: SchemaInteger}, introducedField: deltaItems, "resolved": deltaItems, "suggestions": deltaItems,
	}, "net_delta", introducedField, "resolved", "suggestions")
	return new(outputObject(map[string]ValueSchema{
		"status":                     {Kind: SchemaString, Enum: []any{"restored", "no_op"}},
		"restored_files":             files,
		"restored_file_count":        {Kind: SchemaInteger},
		"restored_files_truncated":   {Kind: SchemaBoolean},
		"diagnostics":                diagnostics,
		"diagnostic_count":           {Kind: SchemaInteger},
		"diagnostics_truncated":      {Kind: SchemaBoolean},
		"diagnostic_status":          {Kind: SchemaString, Enum: []any{"available", diagnosticStatusUnavailable}},
		"diagnostic_error":           {Kind: SchemaString, MaxLength: new(2048)},
		"diagnostic_error_truncated": {Kind: SchemaBoolean},
		"diagnostic_delta":           delta,
		"diagnostic_delta_truncated": {Kind: SchemaBoolean},
	}, "status", "restored_files", "restored_file_count", "restored_files_truncated", "diagnostics", "diagnostic_count", "diagnostics_truncated", "diagnostic_status", "diagnostic_error_truncated", "diagnostic_delta_truncated"))
}

func dependencyOutputSchema() *ValueSchema {
	return new(outputObject(map[string]ValueSchema{
		"status":               {Kind: SchemaString, Enum: []any{"added"}},
		"package":              outputString(),
		"go_mod":               outputString(),
		operationKeyDependency: outputString(),
		"poms":                 {Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}},
	}, "status"))
}

func assertionModeOutputSchema() *ValueSchema {
	return new(outputObject(map[string]ValueSchema{
		"status":         {Kind: SchemaString, Enum: []any{"preview", string(EditNoOp), string(EditRefused), string(EditAppliedWorking), string(EditPersisted), string(EditPersistedWithDiagnostics), string(EditPartiallyPublished)}},
		"file":           outputString(),
		"swapped":        {Kind: SchemaInteger},
		editReceiptField: EditReceiptSchema(),
	}, "status", "file", "swapped", editReceiptField))
}

func referencesOutputSchema() *ValueSchema {
	position := outputObject(map[string]ValueSchema{"line": {Kind: SchemaInteger}, "character": {Kind: SchemaInteger}}, "line", "character")
	rangeSchema := outputObject(map[string]ValueSchema{"start": position, "end": position}, "start", "end")
	scope := readScopeOutputSchema()
	symbol := outputObject(map[string]ValueSchema{
		"name": outputString(), outputQualifiedNameField: outputString(), "kind": outputString(), "file": outputString(),
		"range": rangeSchema, "selection_range": rangeSchema,
	}, "name", outputQualifiedNameField, "kind", "file", "range", "selection_range")
	return new(outputObject(map[string]ValueSchema{
		"summary": outputString(), "scope": scope, "symbol": symbol,
		"reference_count": {Kind: SchemaInteger}, "file_count": {Kind: SchemaInteger},
	}, "summary", "scope", "symbol", "reference_count", "file_count"))
}

func outlineOutputSchema() *ValueSchema {
	return new(outputObject(map[string]ValueSchema{
		"summary": outputString(), "scope": readScopeOutputSchema(),
		"file_count": {Kind: SchemaInteger}, "symbol_count": {Kind: SchemaInteger},
	}, "summary", "scope", "file_count", "symbol_count"))
}

func workspaceOutlineOutputSchema() *ValueSchema {
	module := outputObject(map[string]ValueSchema{
		"path": outputString(), "module_path": outputString(), "go_version": outputString(),
		"group_id": outputString(), "artifact_id": outputString(), "version": outputString(), "packaging": outputString(),
		"is_root": {Kind: SchemaBoolean}, "language": outputString(), "synopsis": outputString(), "doc": outputString(),
		"packages_count": {Kind: SchemaInteger},
	}, "path", "is_root", "language", "packages_count")
	pkg := outputObject(map[string]ValueSchema{
		"path": outputString(), "name": outputString(), "synopsis": outputString(), "doc": outputString(),
		"depth": {Kind: SchemaInteger}, "files_count": {Kind: SchemaInteger}, "module": outputString(), "language": outputString(),
	}, "path", "name", "synopsis", "depth", "files_count", "language")
	modules := ValueSchema{Kind: SchemaArray, Items: &module}
	packages := ValueSchema{Kind: SchemaArray, Items: &pkg}
	return new(outputObject(map[string]ValueSchema{
		"scope": readScopeOutputSchema(), "modules": modules, "packages": packages,
	}, "scope", "modules", "packages"))
}

func readScopeOutputSchema() ValueSchema {
	return outputObject(map[string]ValueSchema{
		"kind": outputString(), "path": outputString(), "complete": {Kind: SchemaBoolean}, "limitations": outputStringList(),
	}, "kind", "path", "complete")
}

func truncateStrings(values []string, maxItems int) ([]string, int, bool) {
	total := len(values)
	if values == nil {
		values = []string{}
	}
	if total <= maxItems {
		return values, total, false
	}
	return values[:maxItems], total, true
}

func truncateText(value string, maxRunes int) (string, bool) {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value, false
	}
	return string(runes[:maxRunes]), true
}

func projectSnapshotOutput(result SnapshotRes) (any, error) {
	if result.Result == nil {
		return nil, fmt.Errorf("snapshot result is missing")
	}
	files, fileCount, truncated := truncateStrings(result.Result.CapturedFiles, 100)
	return map[string]any{
		"status": "created", outputSnapshotIDField: result.Result.SnapshotID, "created_at": result.Result.CreatedAt,
		"captured_files": files, "captured_file_count": fileCount, "captured_files_truncated": truncated,
	}, nil
}

func projectUndoOutput(result UndoRes) (any, error) {
	if result.Result == nil {
		return nil, fmt.Errorf("undo result is missing")
	}
	files, fileCount, filesTruncated := truncateStrings(result.Result.RestoredFiles, 100)
	diagnostics, diagnosticCount, diagnosticsTruncated := truncateStrings(result.Result.Diagnostics, 20)
	status := "restored"
	if fileCount == 0 {
		status = "no_op"
	}
	projected := map[string]any{
		"status": status, "restored_files": files, "restored_file_count": fileCount, "restored_files_truncated": filesTruncated,
		"diagnostics": diagnostics, "diagnostic_count": diagnosticCount, "diagnostics_truncated": diagnosticsTruncated,
		"diagnostic_status": result.Result.DiagnosticStatus, "diagnostic_delta_truncated": false, "diagnostic_error_truncated": false,
	}
	if result.Result.DiagnosticError != "" {
		diagnosticError, truncated := truncateText(result.Result.DiagnosticError, 2048)
		projected["diagnostic_error"] = diagnosticError
		projected["diagnostic_error_truncated"] = truncated
	}
	if result.Result.Delta != nil {
		introduced, _, introducedTruncated := truncateStrings(result.Result.Delta.Introduced, 20)
		resolved, _, resolvedTruncated := truncateStrings(result.Result.Delta.Resolved, 20)
		suggestions, _, suggestionsTruncated := truncateStrings(result.Result.Delta.Suggestions, 20)
		projected["diagnostic_delta"] = map[string]any{"net_delta": result.Result.Delta.NetDelta, introducedField: introduced, "resolved": resolved, "suggestions": suggestions}
		projected["diagnostic_delta_truncated"] = introducedTruncated || resolvedTruncated || suggestionsTruncated
	}
	return projected, nil
}

func projectDependencyOutput(result BuildDependencyRes) (any, error) {
	if result.Dependency != "" {
		return map[string]any{"status": "added", operationKeyDependency: result.Dependency, "poms": result.POMs}, nil
	}
	return map[string]any{"status": "added", "package": result.Package, "go_mod": result.GoMod}, nil
}

func projectReferencesOutput(result *backend.ReferencesResult) (any, error) {
	if result == nil {
		return nil, fmt.Errorf("references result is missing")
	}
	symbol := result.Symbol
	return map[string]any{
		"summary":         fmt.Sprintf("References to %s: %d references in %d files.", declarationIdentity(symbol.Name, symbol.QualifiedName), len(result.References), countReferenceFiles(result.References)),
		"scope":           result.Scope,
		"symbol":          map[string]any{"name": symbol.Name, outputQualifiedNameField: symbol.QualifiedName, "kind": symbol.Kind, "file": symbol.File, "range": symbol.Range, "selection_range": symbol.SelectionRange},
		"reference_count": len(result.References),
		"file_count":      countReferenceFiles(result.References),
	}, nil
}

func projectOutlineOutput(result *backend.OutlineResult) (any, error) {
	if result == nil {
		return nil, fmt.Errorf("outline result is missing")
	}
	symbolCount := countReadSymbols(result.Files)
	return map[string]any{
		"summary": fmt.Sprintf("Declaration outline: %d files, %d declarations.", len(result.Files), symbolCount), "scope": result.Scope,
		"file_count": len(result.Files), "symbol_count": symbolCount,
	}, nil
}

func projectWorkspaceOutlineOutput(result WorkspaceOutlineRes) (any, error) {
	if result.Result == nil {
		return nil, fmt.Errorf("workspace outline result is missing")
	}
	modules := result.Result.Modules
	if modules == nil {
		modules = []backend.WorkspaceModule{}
	}
	packages := result.Result.Packages
	if packages == nil {
		packages = []backend.WorkspacePackage{}
	}
	return map[string]any{"scope": result.Result.Scope, "modules": modules, "packages": packages}, nil
}

func projectAssertionModeOutput(result AssertionModeRes) (any, error) {
	status := string(result.Receipt.Status)
	if result.DryRun {
		status = "preview"
	}
	return map[string]any{"status": status, "file": result.File, "swapped": result.Swapped, editReceiptField: result.Receipt}, nil
}
