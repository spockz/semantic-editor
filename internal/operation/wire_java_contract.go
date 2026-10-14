// Package operation registers and executes semantic editing operations.
// WHY: This file routes Java contract edits through JDT LS and shared receipts.
package operation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func runExtractInterfaceJava(ctx context.Context, cc CallContext, req extractInterfaceRequest) (contractOutput, error) {
	if req.DestinationFile != "" && filepath.Clean(req.DestinationFile) != filepath.Clean(req.File) {
		return contractOutput{}, fmt.Errorf("java extract_interface currently writes the interface into file; destination_file must equal file")
	}
	if err := javacst.ValidateJavaIdentifier(req.Symbol); err != nil {
		return contractOutput{}, fmt.Errorf("invalid Java interface name: %w", err)
	}
	if len(req.Members) == 0 {
		return contractOutput{}, fmt.Errorf("java extract_interface requires at least one selected public method")
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	classResult, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.InType})
	if err != nil {
		return contractOutput{}, err
	}
	if classResult == nil || len(classResult.Matches) != 1 || classResult.Matches[0].Kind != "class" {
		return contractOutput{}, fmt.Errorf("java concrete type %q must resolve to one class in file", req.InType)
	}
	class := classResult.Matches[0]
	before, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return contractOutput{}, fmt.Errorf("read Java interface extraction source: %w", err)
	}
	if req.ExpectedRevision != nil && *req.ExpectedRevision != "" && *req.ExpectedRevision != structuralRevision(before) {
		return contractOutput{}, fmt.Errorf("java interface extraction source revision changed")
	}
	baseline, err := service.Verify(ctx, backend.VerifyRequest{Project: project, Path: file, CheckOnly: true})
	if err != nil {
		return contractOutput{}, fmt.Errorf("capture Java interface extraction diagnostics: %w", err)
	}
	selectedMethods, err := resolveJavaInterfaceMethods(ctx, service, project, cc.WorkDir, file, class, req.Members)
	if err != nil {
		return contractOutput{}, err
	}
	interfaceSource, err := buildJavaInterfaceSource(req.Symbol, selectedMethods)
	if err != nil {
		return contractOutput{}, err
	}
	className := class.Name
	if dot := strings.LastIndex(class.QualifiedName, "."); dot >= 0 {
		className = class.QualifiedName[dot+1:]
	}
	withInterface, changed, err := javacst.AddJavaImplementedInterface(before, className, req.Symbol)
	if err != nil {
		return contractOutput{}, fmt.Errorf("add extracted Java interface to class: %w", err)
	}
	if !changed {
		return contractOutput{}, fmt.Errorf("java class %q already implements %q", req.InType, req.Symbol)
	}
	updated, err := javacst.AppendJavaInterface(withInterface, interfaceSource, req.Symbol)
	if err != nil {
		return contractOutput{}, fmt.Errorf("append extracted Java interface: %w", err)
	}
	root := javaContractWorkRoot(cc)
	after, err := stageAndVerifyJavaInterface(ctx, root, service, project, file, updated, baseline.Diagnostics)
	if err != nil {
		return contractOutput{}, err
	}
	return javaExtractInterfaceResult(root, req, file, before, after, service)
}

func stageAndVerifyJavaInterface(ctx context.Context, root string, service *backend.Service, project backend.ProjectContext, file string, updated []byte, baseline []backend.Diagnostic) ([]byte, error) {
	var after []byte
	err := sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
		if err := sourcebuf.WriteFile(editCtx, file, updated); err != nil {
			return fmt.Errorf("stage extracted Java interface: %w", err)
		}
		if _, err := service.Verify(editCtx, backend.VerifyRequest{Project: project, Path: file, FormatSelectedFile: true, OrganizeImports: true}); err != nil {
			return fmt.Errorf("format and verify extracted Java interface: %w", err)
		}
		current, err := sourcebuf.ReadFile(editCtx, file)
		if err != nil {
			return fmt.Errorf("read verified Java interface source: %w", err)
		}
		verification, err := service.Verify(editCtx, backend.VerifyRequest{Project: project, Path: file, CheckOnly: true})
		if err != nil {
			return fmt.Errorf("check extracted Java interface diagnostics: %w", err)
		}
		if javaIntroducedErrors(baseline, verification.Diagnostics) {
			return fmt.Errorf("extracted Java interface introduces JDT LS error diagnostics")
		}
		after = current
		return nil
	})
	return after, err
}

func javaExtractInterfaceResult(root string, req extractInterfaceRequest, file string, before, after []byte, service *backend.Service) (contractOutput, error) {
	relative, err := relativeWorkPath(root, file)
	if err != nil {
		return contractOutput{}, err
	}
	preRevision, postRevision := structuralRevision(before), structuralRevision(after)
	return contractOutput{
		Status: "persisted", File: displayWorkPath(root, req.File), Symbol: req.Symbol,
		Detail:  "Extracted the selected public methods into a package-private Java interface, implemented it on the class, and verified through JDT LS.",
		ReadSet: []contractRevision{{Path: relative, Revision: preRevision}}, Changes: []contractRevision{{Path: relative, Revision: postRevision}},
		ReceiptCandidates: []SymbolTarget{{File: relative, Symbol: req.Symbol, ExpectedRevision: &postRevision}}, javaService: service,
	}, nil
}

func javaContractWorkRoot(cc CallContext) string {
	if cc.WorkDir != "" {
		return cc.WorkDir
	}
	return cc.Project.RootDir
}

func resolveJavaInterfaceMethods(ctx context.Context, service *backend.Service, project backend.ProjectContext, workDir, file string, class backend.InspectedSymbol, targets []contractMemberTarget) ([]backend.InspectedSymbol, error) {
	selected := make([]backend.InspectedSymbol, 0, len(targets))
	seen := make(map[string]bool, len(targets))
	for _, target := range targets {
		if filepath.Clean(resolveWorkPath(workDir, target.File)) != filepath.Clean(file) {
			return nil, fmt.Errorf("java interface methods must be selected from the concrete type file")
		}
		matches := directJavaMethodMatches(class.Children, target.Symbol)
		if len(matches) != 1 {
			return nil, fmt.Errorf("java method selector %q resolves to %d direct methods; use its signature-qualified identity", target.Symbol, len(matches))
		}
		qualifiedName := matches[0].QualifiedName
		if seen[qualifiedName] {
			return nil, fmt.Errorf("java method %q is selected more than once", target.Symbol)
		}
		seen[qualifiedName] = true
		inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: qualifiedName})
		if err != nil {
			return nil, err
		}
		if inspection == nil || len(inspection.Matches) != 1 || inspection.Matches[0].Kind != "method" {
			return nil, fmt.Errorf("java method %q no longer resolves uniquely", qualifiedName)
		}
		selected = append(selected, inspection.Matches[0])
	}
	return selected, nil
}

func directJavaMethodMatches(children []backend.ReadSymbol, selector string) []backend.ReadSymbol {
	var matches []backend.ReadSymbol
	for _, child := range children {
		if child.Kind == "method" && (selector == child.QualifiedName || selector == child.Name) {
			matches = append(matches, child)
		}
	}
	return matches
}

func buildJavaInterfaceSource(name string, methods []backend.InspectedSymbol) ([]byte, error) {
	source := []byte("interface " + name + " {\n")
	for _, method := range methods {
		signature, err := javacst.JavaInterfaceMethodSignature([]byte(method.Source))
		if err != nil {
			return nil, fmt.Errorf("convert Java method %q to interface signature: %w", method.QualifiedName, err)
		}
		source = append(source, []byte("\t"+signature+"\n")...)
	}
	return append(source, '}'), nil
}

func javaIntroducedErrors(before, after []backend.Diagnostic) bool {
	counts := make(map[string]int)
	for _, diagnostic := range before {
		if diagnostic.Severity == 1 {
			counts[javaDiagnosticKey(diagnostic)]++
		}
	}
	for _, diagnostic := range after {
		if diagnostic.Severity != 1 {
			continue
		}
		key := javaDiagnosticKey(diagnostic)
		if counts[key] == 0 {
			return true
		}
		counts[key]--
	}
	return false
}

func javaDiagnosticKey(diagnostic backend.Diagnostic) string {
	var location string
	if diagnostic.Location != nil {
		location = fmt.Sprintf("%s:%d:%d", diagnostic.Location.URI, diagnostic.Location.Range.Start.Line, diagnostic.Location.Range.Start.Character)
	}
	return location + "\x00" + diagnostic.Message
}

func finalizeJavaExtractInterfaceOutput(cc CallContext, result contractOutput) (contractOutput, error) {
	service := result.javaService
	if service == nil {
		service = cc.Service
	}
	if service == nil {
		service = backends.NewDefaultService()
	}
	root := cc.WorkDir
	if root == "" {
		root = cc.Project.RootDir
	}
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	readSet := make([]FileRevision, 0, len(result.ReadSet))
	for _, revision := range result.ReadSet {
		readSet = append(readSet, FileRevision(revision))
	}
	planned := make([]FileRevision, 0, len(result.Changes))
	for _, revision := range result.Changes {
		planned = append(planned, FileRevision(revision))
	}
	receipt, err := finalizeCallReceipt(ctx, cc, root, EditReceiptDraft{
		Operation: result.Operation, Status: EditPersisted, PlannedChanges: planned, ReadSet: readSet, Continuations: result.ReceiptCandidates,
	}, func(resolveCtx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		if candidate.ExpectedRevision == nil {
			return SymbolTarget{}, false, nil
		}
		project := effectiveProject(cc, cc.Project)
		project.File = resolveWorkPath(root, candidate.File)
		inspection, inspectErr := service.Inspect(resolveCtx, backend.InspectRequest{Project: project, Symbol: candidate.Symbol})
		if inspectErr != nil {
			return SymbolTarget{}, false, inspectErr
		}
		if inspection == nil || len(inspection.Matches) != 1 || inspection.Matches[0].Revision != *candidate.ExpectedRevision {
			return SymbolTarget{}, false, nil
		}
		return candidate, true, nil
	})
	if err != nil {
		return contractOutput{}, fmt.Errorf("finalize Java extract-interface receipt: %w", err)
	}
	result.Receipt = receipt
	for _, continuation := range receipt.Continuations {
		if continuation.Symbol == result.Symbol {
			result.Contract = continuation.Symbol
			result.ContractFile = continuation.File
			break
		}
	}
	return result, nil
}

func runImplementContractJava(ctx context.Context, cc CallContext, req implementContractRequest) (contractOutput, error) {
	if req.DestinationFile != "" && filepath.Clean(req.DestinationFile) != filepath.Clean(req.File) {
		return contractOutput{}, fmt.Errorf("java implement_contract writes only the selected type file; destination_file must equal file")
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	plan, err := prepareJavaContractImplementation(ctx, service, project, cc.WorkDir, req)
	if err != nil {
		return contractOutput{}, err
	}
	file, class, contractFile, contractProject, contractSource, before := plan.file, plan.class, plan.contractFile, plan.contractProject, plan.contractSource, plan.before
	actionResult, finalClass, err := generateJavaContractMethods(ctx, cc.WorkDir, service, project, plan, req)
	if err != nil {
		return contractOutput{}, err
	}
	if actionResult == nil {
		return contractOutput{}, fmt.Errorf("java implement_contract completed without a code action result")
	}
	relative, err := filepath.Rel(cc.WorkDir, file)
	if err != nil || !filepath.IsLocal(relative) {
		return contractOutput{}, fmt.Errorf("derive Java contract receipt path for %q: %w", file, err)
	}
	contractRelative, err := filepath.Rel(cc.WorkDir, contractProject.File)
	if err != nil || !filepath.IsLocal(contractRelative) {
		return contractOutput{}, fmt.Errorf("derive Java interface receipt path for %q: %w", contractProject.File, err)
	}
	relative = filepath.ToSlash(relative)
	contractRelative = filepath.ToSlash(contractRelative)
	if len(actionResult.Files) == 0 {
		return contractOutput{}, fmt.Errorf("java implement_contract returned no workspace file edits")
	}
	readSet := make([]contractRevision, 0, len(actionResult.Files)+1)
	changes := make([]contractRevision, 0, len(actionResult.Files))
	for _, fileEdit := range actionResult.Files {
		path, pathErr := relativeWorkPath(cc.WorkDir, fileEdit.File)
		if pathErr != nil {
			return contractOutput{}, fmt.Errorf("derive Java contract workspace path for %q: %w", fileEdit.File, pathErr)
		}
		preimage := fileEdit.Before
		if fileEdit.File == file {
			preimage = before
		}
		readSet = append(readSet, contractRevision{Path: path, Revision: structuralRevision(preimage)})
		if structuralRevision(preimage) != structuralRevision(fileEdit.After) {
			changes = append(changes, contractRevision{Path: path, Revision: structuralRevision(fileEdit.After)})
		}
	}
	if contractRelative != relative {
		readSet = append(readSet, contractRevision{Path: contractRelative, Revision: structuralRevision(contractSource)})
	}
	postRevision := structuralRevision(actionResult.After)
	output := contractOutput{
		Status: "persisted", NoOp: len(changes) == 0, File: displayWorkPath(cc.WorkDir, req.File),
		Symbol: req.Symbol, Contract: req.Contract, ContractFile: displayWorkPath(cc.WorkDir, contractFile),
		Detail:  "Ensured the Java class implements the selected interface and delegated missing method generation to JDT LS.",
		ReadSet: readSet, ReceiptCandidates: []SymbolTarget{{File: relative, Symbol: req.Symbol, ExpectedRevision: &postRevision}}, javaService: service,
	}
	if output.NoOp {
		output.Status = "no_op"
	} else {
		output.Changes = changes
	}
	beforeMethods := javaDirectMethodNames(class.Children)
	for _, method := range finalClass.Children {
		if method.Kind == "method" && !beforeMethods[method.Name] {
			output.ReceiptCandidates = append(output.ReceiptCandidates, SymbolTarget{File: relative, Symbol: method.QualifiedName, ExpectedRevision: &postRevision})
		}
	}
	return output, nil
}

func generateJavaContractMethods(ctx context.Context, workDir string, service *backend.Service, project backend.ProjectContext, plan javaContractImplementationPlan, req implementContractRequest) (*backend.CodeActionResult, backend.InspectedSymbol, error) {
	var actionResult *backend.CodeActionResult
	var finalClass backend.InspectedSymbol
	err := sourcebuf.Run(ctx, workDir, func(editCtx context.Context) error {
		if plan.changed {
			if err := sourcebuf.WriteFile(editCtx, plan.file, plan.updated); err != nil {
				return fmt.Errorf("stage Java implements clause: %w", err)
			}
		}
		working, err := sourcebuf.ReadFile(editCtx, plan.file)
		if err != nil {
			return fmt.Errorf("read staged Java implements clause: %w", err)
		}
		actionResult, err = service.ApplyCodeAction(editCtx, backend.CodeActionRequest{
			Operation: backend.OperationImplementContract, Project: project, File: plan.file,
			Range: plan.class.SelectionRange, Kind: "source.overrideMethods", ExpectedRevision: structuralRevision(working),
		})
		if err != nil {
			return err
		}
		if actionResult == nil || actionResult.File != plan.file {
			return fmt.Errorf("java implement_contract backend returned an unexpected target file")
		}
		inspection, err := service.Inspect(editCtx, backend.InspectRequest{Project: project, Symbol: req.Symbol})
		if err != nil {
			return fmt.Errorf("inspect generated Java contract methods: %w", err)
		}
		if inspection == nil || len(inspection.Matches) != 1 {
			return fmt.Errorf("generated Java contract type %q no longer resolves uniquely", req.Symbol)
		}
		finalClass = inspection.Matches[0]
		return nil
	})
	return actionResult, finalClass, err
}

type javaContractImplementationPlan struct {
	file            string
	class           backend.InspectedSymbol
	contractFile    string
	contractProject backend.ProjectContext
	contractSource  []byte
	before          []byte
	updated         []byte
	changed         bool
}

func prepareJavaContractImplementation(ctx context.Context, service *backend.Service, project backend.ProjectContext, workDir string, req implementContractRequest) (javaContractImplementationPlan, error) {
	file := project.File
	classResult, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.Symbol})
	if err != nil {
		return javaContractImplementationPlan{}, err
	}
	if classResult == nil || len(classResult.Matches) != 1 || classResult.Matches[0].Kind != "class" {
		return javaContractImplementationPlan{}, fmt.Errorf("java concrete type %q must resolve to one class in file", req.Symbol)
	}
	contractFile := req.ContractFile
	if contractFile == "" {
		contractFile = req.File
	}
	contractProject := project
	contractProject.File = resolveWorkPath(workDir, contractFile)
	contractResult, err := service.Inspect(ctx, backend.InspectRequest{Project: contractProject, Symbol: req.Contract})
	if err != nil {
		return javaContractImplementationPlan{}, err
	}
	if contractResult == nil || len(contractResult.Matches) != 1 || contractResult.Matches[0].Kind != "interface" {
		return javaContractImplementationPlan{}, fmt.Errorf("java contract %q must resolve to one interface in contract_file", req.Contract)
	}
	class, contract := classResult.Matches[0], contractResult.Matches[0]
	className := class.Name
	if dot := strings.LastIndex(class.QualifiedName, "."); dot >= 0 {
		className = class.QualifiedName[dot+1:]
	}
	before, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return javaContractImplementationPlan{}, fmt.Errorf("read Java concrete type source: %w", err)
	}
	if req.ExpectedRevision != nil && *req.ExpectedRevision != "" && *req.ExpectedRevision != structuralRevision(before) {
		return javaContractImplementationPlan{}, fmt.Errorf("java implement_contract source revision changed")
	}
	contractSource, err := sourcebuf.ReadFile(ctx, contractProject.File)
	if err != nil {
		return javaContractImplementationPlan{}, fmt.Errorf("read Java interface source: %w", err)
	}
	classPackage, err := javacst.JavaPackageName(before)
	if err != nil {
		return javaContractImplementationPlan{}, fmt.Errorf("read Java concrete type package: %w", err)
	}
	contractPackage, err := javacst.JavaPackageName(contractSource)
	if err != nil {
		return javaContractImplementationPlan{}, fmt.Errorf("read Java interface package: %w", err)
	}
	interfaceName := contract.Name
	if contractPackage != "" && contractPackage != classPackage {
		interfaceName = contractPackage + "." + interfaceName
	}
	updated, changed, err := javacst.AddJavaImplementedInterface(before, className, interfaceName)
	if err != nil {
		return javaContractImplementationPlan{}, fmt.Errorf("add Java implemented interface: %w", err)
	}
	return javaContractImplementationPlan{file: file, class: class, contractFile: contractFile, contractProject: contractProject, contractSource: contractSource, before: before, updated: updated, changed: changed}, nil
}

func javaDirectMethodNames(children []backend.ReadSymbol) map[string]bool {
	methods := make(map[string]bool)
	for _, child := range children {
		if child.Kind == "method" {
			methods[child.Name] = true
		}
	}
	return methods
}

func finalizeJavaContractOutput(cc CallContext, result contractOutput) (contractOutput, error) {
	root := cc.WorkDir
	if root == "" {
		root = cc.Project.RootDir
	}
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	project := effectiveProject(cc, cc.Project)
	service := result.javaService
	if service == nil {
		service = cc.Service
	}
	if service == nil {
		service = backends.NewDefaultService()
	}
	status := EditPersisted
	if result.Status == "no_op" {
		status = EditNoOp
	}
	readSet := make([]FileRevision, 0, len(result.ReadSet))
	for _, revision := range result.ReadSet {
		readSet = append(readSet, FileRevision(revision))
	}
	planned := make([]FileRevision, 0, len(result.Changes))
	for _, revision := range result.Changes {
		planned = append(planned, FileRevision(revision))
	}
	receipt, err := finalizeCallReceipt(ctx, cc, root, EditReceiptDraft{
		Operation: result.Operation, Status: status, PlannedChanges: planned,
		ReadSet: readSet, Continuations: result.ReceiptCandidates,
	}, func(resolveCtx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		candidateProject := project
		candidateProject.File = resolveWorkPath(root, candidate.File)
		inspection, inspectErr := service.Inspect(resolveCtx, backend.InspectRequest{Project: candidateProject, Symbol: candidate.Symbol})
		if inspectErr != nil {
			return SymbolTarget{}, false, inspectErr
		}
		if inspection == nil || len(inspection.Matches) != 1 {
			return SymbolTarget{}, false, nil
		}
		return SymbolTarget{File: candidate.File, Symbol: candidate.Symbol}, true, nil
	})
	if err != nil {
		return contractOutput{}, fmt.Errorf("finalize Java contract receipt: %w", err)
	}
	result.Receipt = receipt
	for _, continuation := range receipt.Continuations {
		if continuation.Symbol == result.Symbol {
			target := continuation
			result.TypeTarget = &target
		} else {
			result.GeneratedMethods = append(result.GeneratedMethods, continuation)
		}
	}
	return result, nil
}
