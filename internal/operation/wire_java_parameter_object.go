// Package operation registers and executes semantic editing operations.
// WHY: This file coordinates Java parameter-object changes and rewritten invocations.
package operation

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

type javaParameterObjectMethodPlan struct {
	method      backend.InspectedSymbol
	parameters  []javacst.JavaParameter
	fields      []javacst.JavaParameterField
	packageName string
}

type javaLocatedCall struct {
	file      string
	position  backend.Position
	arguments []string
}

func runIntroduceParameterObjectJava(ctx context.Context, cc CallContext, raw map[string]any) (structuralPlanResult, error) {
	file, symbol := stringParam(raw, "file"), stringParam(raw, "symbol")
	if stringParam(raw, "destination_file") != file {
		return structuralPlanResult{}, fmt.Errorf("java destination_file must equal file so the package-private parameter object remains in the selected source")
	}
	project, projectErr := parseReadProject(raw, file)
	if projectErr != nil {
		return structuralPlanResult{}, projectErr
	}
	if project.Language != backend.LanguageJava {
		return structuralPlanResult{}, fmt.Errorf("java introduce_parameter_object requires language=java")
	}
	project = effectiveProject(cc, project)
	project.File = resolveWorkPath(cc.WorkDir, file)
	root := cc.WorkDir
	if root == "" {
		root = project.RootDir
	}
	root, rootErr := filepath.Abs(root)
	if rootErr != nil {
		return structuralPlanResult{}, fmt.Errorf("resolve Java parameter-object workspace: %w", rootErr)
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	parameter, parameterErr := decodeParameterSpec(raw["parameter"])
	if parameterErr != nil {
		return structuralPlanResult{}, parameterErr
	}
	if err := javacst.ValidateJavaIdentifier(parameter.Name); err != nil {
		return structuralPlanResult{}, fmt.Errorf("invalid Java parameter-object parameter name: %w", err)
	}
	if err := javacst.ValidateJavaIdentifier(parameter.Type); err != nil {
		return structuralPlanResult{}, fmt.Errorf("invalid Java parameter-object type name: %w", err)
	}
	fields, fieldsErr := decodeParameterObjectFields(raw["fields"])
	if fieldsErr != nil {
		return structuralPlanResult{}, fieldsErr
	}
	var result structuralPlanResult
	runErr := sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
		var err error
		result, err = runJavaParameterObjectEdit(editCtx, root, file, project, symbol, parameter, fields, raw, service)
		return err
	})
	if runErr != nil {
		return structuralPlanResult{}, runErr
	}
	return result, nil
}

func runJavaParameterObjectEdit(ctx context.Context, root, file string, project backend.ProjectContext, symbol string, parameter ParameterSpec, fields []astedit.ParameterObjectField, raw map[string]any, service *backend.Service) (structuralPlanResult, error) {
	buffer := sourcebuf.FromContext(ctx)
	checkpoint := buffer.Checkpoint()
	before, err := sourcebuf.ReadFile(ctx, project.File)
	if err != nil {
		return structuralPlanResult{}, fmt.Errorf("read Java parameter-object target: %w", err)
	}
	if revisionErr := validateSignatureExpectedRevision(raw, before); revisionErr != nil {
		return structuralPlanResult{}, revisionErr
	}
	methodPlan, err := inspectJavaParameterObjectMethod(ctx, service, project, symbol, before, fields)
	if err != nil {
		return structuralPlanResult{}, err
	}
	callSites, readSet, err := resolveJavaParameterObjectCalls(ctx, service, root, project, symbol, methodPlan.packageName, methodPlan.parameters)
	if err != nil {
		return structuralPlanResult{}, err
	}
	generatedType, err := javacst.InsertJavaParameterObject(before, parameter.Type, methodPlan.fields)
	if err != nil {
		return structuralPlanResult{}, fmt.Errorf("create Java parameter-object type: %w", err)
	}
	if applyErr := buffer.Apply(ctx, []sourcebuf.Change{{Path: project.File, Before: before, After: generatedType}}); applyErr != nil {
		return structuralPlanResult{}, fmt.Errorf("stage Java parameter-object type: %w", applyErr)
	}
	request := javaParameterObjectCodeActionRequest(project, symbol, parameter, methodPlan, callSites, generatedType)
	codeAction, err := service.ApplyCodeAction(ctx, request)
	if err != nil {
		return structuralPlanResult{}, err
	}
	if codeAction == nil || codeAction.File != project.File {
		return structuralPlanResult{}, fmt.Errorf("java parameter-object backend returned an unexpected target file")
	}
	return javaParameterObjectReceipt(root, file, project, symbol, parameter.Type, buffer.ChangesSince(checkpoint), readSet)
}

func javaParameterObjectCodeActionRequest(project backend.ProjectContext, symbol string, parameter ParameterSpec, plan javaParameterObjectMethodPlan, callSites []backend.CodeActionCallSite, generatedType []byte) backend.CodeActionRequest {
	mappedFields := make([]backend.CodeActionParameterField, 0, len(plan.fields))
	for _, field := range plan.fields {
		mappedFields = append(mappedFields, backend.CodeActionParameterField{SourceIndex: field.SourceIndex, ParameterName: field.ParameterName, Name: field.Name, Type: field.Type})
	}
	return backend.CodeActionRequest{
		Operation: backend.OperationIntroduceParameterObject, Project: project, File: project.File,
		Range: plan.method.SelectionRange, Kind: "refactor.change.signature", Symbol: symbol,
		Parameters:            []backend.CodeActionParameter{{Name: parameter.Name, Type: parameter.Type, Argument: "new " + parameter.Type + "()"}},
		ParameterObjectMethod: plan.method.Name, ParameterObjectType: parameter.Type,
		ParameterObjectFields: mappedFields, ParameterObjectCallSites: callSites,
		ExpectedRevision: structuralRevision(generatedType),
	}
}

func javaParameterObjectReceipt(root, file string, project backend.ProjectContext, symbol, typeName string, changes []sourcebuf.Change, readSet map[string]string) (structuralPlanResult, error) {
	readSetItems := make([]FileRevision, 0, len(readSet)+len(changes))
	for path, revision := range readSet {
		readSetItems = append(readSetItems, FileRevision{Path: path, Revision: revision})
	}
	var selectedBefore, selectedAfter []byte
	planned := make([]FileRevision, 0, len(changes))
	for _, change := range changes {
		relative, err := relativeWorkPath(root, change.Path)
		if err != nil {
			return structuralPlanResult{}, err
		}
		readSetItems = append(readSetItems, FileRevision{Path: relative, Revision: structuralRevision(change.Before)})
		planned = append(planned, FileRevision{Path: relative, Revision: structuralRevision(change.After)})
		if change.Path == project.File {
			selectedBefore, selectedAfter = change.Before, change.After
		}
	}
	if selectedBefore == nil || selectedAfter == nil {
		return structuralPlanResult{}, fmt.Errorf("java parameter-object buffer omitted its selected source")
	}
	readSetItems, err := uniqueFileRevisions(readSetItems)
	if err != nil {
		return structuralPlanResult{}, err
	}
	sort.Slice(readSetItems, func(left, right int) bool { return readSetItems[left].Path < readSetItems[right].Path })
	sort.Slice(planned, func(left, right int) bool { return planned[left].Path < planned[right].Path })
	revision := structuralRevision(selectedAfter)
	continuation := SymbolTarget{File: displayWorkPath(root, file), Symbol: typeName, ExpectedRevision: &revision}
	display := displayWorkPath(root, file)
	return structuralPlanResult{
		Edit: FileEditRes{
			File: project.File, Display: display, Symbol: symbol,
			Diff:             astedit.UnifiedDiff(display, string(selectedBefore), string(selectedAfter)),
			Detail:           "Replaced the Java method parameters with a package-private value object and updated every resolved invocation.",
			DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true,
		},
		Draft: EditReceiptDraft{Operation: string(backend.OperationIntroduceParameterObject), Status: EditPersisted, ReadSet: readSetItems, PlannedChanges: planned, Continuations: []SymbolTarget{continuation}},
	}, nil
}

func inspectJavaParameterObjectMethod(ctx context.Context, service *backend.Service, project backend.ProjectContext, symbol string, before []byte, fields []astedit.ParameterObjectField) (javaParameterObjectMethodPlan, error) {
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: symbol})
	if err != nil {
		return javaParameterObjectMethodPlan{}, fmt.Errorf("inspect Java parameter-object method: %w", err)
	}
	if inspection == nil || len(inspection.Matches) != 1 || inspection.Matches[0].Kind != "method" {
		return javaParameterObjectMethodPlan{}, fmt.Errorf("java introduce_parameter_object target %q must resolve to one method", symbol)
	}
	method := inspection.Matches[0]
	start, err := javacst.ByteOffsetFromPosition(before, method.Range.Start)
	if err != nil {
		return javaParameterObjectMethodPlan{}, fmt.Errorf("resolve Java method source range: %w", err)
	}
	end, err := javacst.ByteOffsetFromPosition(before, method.Range.End)
	if err != nil || end <= start || end > len(before) || string(before[start:end]) != method.Source {
		return javaParameterObjectMethodPlan{}, fmt.Errorf("java parameter-object method source changed during inspection")
	}
	methodSource := []byte(method.Source)
	public, err := javacst.JavaMethodExposesOutsidePackage(methodSource)
	if err != nil {
		return javaParameterObjectMethodPlan{}, err
	}
	if public {
		return javaParameterObjectMethodPlan{}, fmt.Errorf("java introduce_parameter_object refuses public or protected methods because the generated type is package-private")
	}
	parameters, err := javacst.ParseJavaMethodParameters(methodSource)
	if err != nil {
		return javaParameterObjectMethodPlan{}, err
	}
	if len(parameters) == 0 || len(parameters) != len(fields) {
		return javaParameterObjectMethodPlan{}, fmt.Errorf("java introduce_parameter_object requires one field for every original method parameter")
	}
	javaFields, err := javaParameterObjectFields(fields, parameters)
	if err != nil {
		return javaParameterObjectMethodPlan{}, err
	}
	packageName, err := javacst.JavaPackageName(before)
	if err != nil {
		return javaParameterObjectMethodPlan{}, err
	}
	return javaParameterObjectMethodPlan{method: method, parameters: parameters, fields: javaFields, packageName: packageName}, nil
}

func javaParameterObjectFields(fields []astedit.ParameterObjectField, parameters []javacst.JavaParameter) ([]javacst.JavaParameterField, error) {
	javaFields := make([]javacst.JavaParameterField, 0, len(fields))
	for index, field := range fields {
		if field.SourceIndex != index || field.Type != parameters[index].Type || field.Tag != "" {
			return nil, fmt.Errorf("java parameter-object field %d must preserve its source parameter type and cannot use a Go struct tag", index)
		}
		javaFields = append(javaFields, javacst.JavaParameterField{SourceIndex: index, ParameterName: parameters[index].Name, Name: field.Name, Type: field.Type})
	}
	return javaFields, nil
}

func resolveJavaParameterObjectCalls(ctx context.Context, service *backend.Service, root string, project backend.ProjectContext, symbol, targetPackage string, parameters []javacst.JavaParameter) ([]backend.CodeActionCallSite, map[string]string, error) {
	references, err := service.FindReferences(ctx, backend.ReferencesRequest{Project: project, Symbol: symbol})
	if err != nil {
		return nil, nil, fmt.Errorf("resolve Java parameter-object callers: %w", err)
	}
	if references == nil || references.Symbol.Kind != "method" {
		return nil, nil, fmt.Errorf("java parameter-object reference search returned no method proof")
	}
	readSet, revisionByFile, err := verifyJavaParameterObjectRevisions(ctx, root, references.Files)
	if err != nil {
		return nil, nil, err
	}
	callsByFile := make(map[string][]javaLocatedCall)
	for _, reference := range references.References {
		call, err := inspectJavaParameterObjectCall(ctx, root, reference, targetPackage, len(parameters), revisionByFile)
		if err != nil {
			return nil, nil, err
		}
		callsByFile[call.file] = append(callsByFile[call.file], call)
	}
	return sortedJavaParameterObjectCallSites(callsByFile, len(references.References)), readSet, nil
}

func verifyJavaParameterObjectRevisions(ctx context.Context, root string, files []backend.FileRevision) (map[string]string, map[string]string, error) {
	readSet := make(map[string]string)
	revisions := make(map[string]string, len(files))
	for _, revision := range files {
		file := javaReferenceAbsolutePath(root, revision.File)
		content, err := sourcebuf.ReadFile(ctx, file)
		if err != nil {
			return nil, nil, fmt.Errorf("read Java reference source %s: %w", file, err)
		}
		if structuralRevision(content) != revision.Revision {
			return nil, nil, fmt.Errorf("java reference source %s changed during parameter-object planning", file)
		}
		relative, err := relativeWorkPath(root, file)
		if err != nil {
			return nil, nil, err
		}
		readSet[relative] = revision.Revision
		revisions[file] = revision.Revision
	}
	return readSet, revisions, nil
}

func inspectJavaParameterObjectCall(ctx context.Context, root string, reference backend.Reference, targetPackage string, parameterCount int, revisions map[string]string) (javaLocatedCall, error) {
	file := javaReferenceAbsolutePath(root, reference.File)
	if _, found := revisions[file]; !found {
		return javaLocatedCall{}, fmt.Errorf("java reference source %s has no recorded revision", reference.File)
	}
	content, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return javaLocatedCall{}, fmt.Errorf("read Java parameter-object caller %s: %w", file, err)
	}
	packageName, err := javacst.JavaPackageName(content)
	if err != nil {
		return javaLocatedCall{}, err
	}
	if packageName != targetPackage {
		return javaLocatedCall{}, fmt.Errorf("java parameter-object caller %s is outside package %q", file, targetPackage)
	}
	arguments, err := javacst.JavaMethodCallArgumentsAt(content, reference.Range)
	if err != nil {
		return javaLocatedCall{}, fmt.Errorf("java reference in %s at [%d:%d-%d:%d] is not a direct method invocation: %w", file, reference.Range.Start.Line, reference.Range.Start.Character, reference.Range.End.Line, reference.Range.End.Character, err)
	}
	if len(arguments) != parameterCount {
		return javaLocatedCall{}, fmt.Errorf("java parameter-object call in %s has %d arguments for %d parameters", file, len(arguments), parameterCount)
	}
	return javaLocatedCall{file: file, position: reference.Range.Start, arguments: arguments}, nil
}

func javaReferenceAbsolutePath(root, file string) string {
	if filepath.IsAbs(file) {
		return filepath.Clean(file)
	}
	return filepath.Join(root, file)
}

func sortedJavaParameterObjectCallSites(callsByFile map[string][]javaLocatedCall, capacity int) []backend.CodeActionCallSite {
	files := make([]string, 0, len(callsByFile))
	for file := range callsByFile {
		files = append(files, file)
	}
	sort.Strings(files)
	calls := make([]backend.CodeActionCallSite, 0, capacity)
	for _, file := range files {
		located := callsByFile[file]
		sort.Slice(located, func(left, right int) bool {
			if located[left].position.Line != located[right].position.Line {
				return located[left].position.Line < located[right].position.Line
			}
			return located[left].position.Character < located[right].position.Character
		})
		for _, call := range located {
			calls = append(calls, backend.CodeActionCallSite{File: file, Arguments: call.arguments})
		}
	}
	return calls
}

func uniqueFileRevisions(revisions []FileRevision) ([]FileRevision, error) {
	byPath := make(map[string]FileRevision, len(revisions))
	for _, revision := range revisions {
		if prior, exists := byPath[revision.Path]; exists && prior.Revision != revision.Revision {
			return nil, fmt.Errorf("java parameter-object read revision changed for %s", revision.Path)
		}
		byPath[revision.Path] = revision
	}
	unique := make([]FileRevision, 0, len(byPath))
	for _, revision := range byPath {
		unique = append(unique, revision)
	}
	return unique, nil
}
