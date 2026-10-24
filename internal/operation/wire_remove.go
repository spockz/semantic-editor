// This file isolates the exact-node removal request schema and handler from replacement wire contracts.

package operation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/javacst"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

const (
	removeSafeDelete       = "safe_delete"
	removeChecksRun        = "checks_run"
	removeDiagnosticStatus = "diagnostic_status"
	removeDiagnosticDelta  = "diagnostic_delta"
)

// RemoveReq selects a Go declaration or case for exact structural removal.
type RemoveReq struct {
	Project             backend.ProjectContext
	File                string
	Kind                backend.ConstructKind
	Symbol              string
	InFunction          string
	Discriminator       string
	TargetCase          string
	ConstructPath       string
	SafeDelete          bool
	AutoOrganizeImports bool
}

// SafeDeleteRefusalError reports an incomplete or failed preflight as an operation error.
type SafeDeleteRefusalError struct {
	File          string
	Symbol        string
	InFunction    string
	Discriminator string
	TargetCase    string
	ConstructPath string
	ChecksRun     []string
	Reason        string
}

// Error describes why safe deletion refused to mutate the workspace.
func (e *SafeDeleteRefusalError) Error() string {
	return fmt.Sprintf("safe_delete refused: %s (checks run: %s)", e.Reason, strings.Join(e.ChecksRun, ", "))
}

// RemoveRes reports the removal outcome, checks, diagnostics, and continuation selectors.
type RemoveRes struct {
	File             string                   `json:"file,omitempty"`
	Display          string                   `json:"-"`
	Symbol           string                   `json:"symbol,omitempty"`
	InFunction       string                   `json:"in_function,omitempty"`
	Discriminator    string                   `json:"discriminator,omitempty"`
	TargetCase       string                   `json:"target_case,omitempty"`
	ConstructPath    string                   `json:"construct_path,omitempty"`
	SafeDelete       bool                     `json:"safe_delete"`
	Status           string                   `json:"status"`
	ChecksRun        []string                 `json:"checks_run,omitempty"`
	Refusal          string                   `json:"refusal,omitempty"`
	Diff             string                   `json:"diff,omitempty"`
	DiagnosticStatus string                   `json:"diagnostic_status,omitempty"`
	Delta            pipeline.DiagnosticDelta `json:"diagnostic_delta"`
	HasDelta         bool                     `json:"-"`
}

// GetProjectContext returns the request project for registry dispatch.
func (r RemoveReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext replaces the request project during dispatch merge.
func (r *RemoveReq) SetProjectContext(project backend.ProjectContext) { r.Project = project }

var (
	removeFileSources          = sourceFileContinuations
	removeSymbolSources        = sourceSymbolContinuations
	removeDiscriminatorSources = sourceDiscriminatorContinuations
	removeParams               = append([]ParameterContract{
		{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Role: RoleTarget, Description: "Target source file containing the selected construct", Required: true, SourceFields: removeFileSources},
		{Name: "kind", CLIName: "kind", JSONName: "kind", Type: ParamString, Description: "Construct kind supported by the selected language backend", Required: true, DynamicEnums: executableRemoveConstructKinds},
		{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Role: RoleTarget, Description: "Target: qualified top-level declaration to remove; required for function, method, type, constant, or variable, and never an owner selector", SourceFields: removeSymbolSources},
		{Name: wireInFunction, CLIName: wireCLIInFunction, JSONName: wireInFunction, Type: ParamString, Role: RoleSelector, Description: "Selector: owner function or method containing the case target; required only for kind=case", SourceFields: sourceInFunctionContinuations},
		{Name: wireDiscriminator, CLIName: wireDiscriminator, JSONName: wireDiscriminator, Type: ParamString, Role: RoleSelector, Description: "Selector: switch discriminant expression or type-switch assignment to narrow case selection", SourceFields: removeDiscriminatorSources},
		{Name: "target_case", CLIName: "target-case", JSONName: "target_case", Type: ParamString, Description: "Selector: Go case value to remove (for example nil or \"foo\"); use default for a default clause"},
		{Name: wireConstructPath, CLIName: wireCLIConstructPath, JSONName: wireConstructPath, Type: ParamString, Description: "Selector: candidate path from an ambiguity diagnostic (for example 0 or 0.1)"},
		{Name: removeSafeDelete, CLIName: "safe-delete", JSONName: removeSafeDelete, Type: ParamBoolean, Description: "Request fail-closed deletion proof: Go supports unused private standalone constants, eligible free functions, explicitly typed zero-value package variables of any type, initialized basic package variables with compiler-proven constant initializers, method-free defined types, and unreferenced type aliases; Java supports private declarations with complete JDT LS reference coverage. References, unsupported effects, or incomplete analysis refuse the edit.", Default: false},
		{Name: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, JSONName: wireAutoOrganizeImports, Type: ParamBoolean, Description: "Automatically remove imports made unused by this edit (default true)", Default: true},
	}, readProjectParams...)
)

func parseRemove(raw map[string]any) (RemoveReq, error) {
	var req RemoveReq
	if err := CheckParams(raw, removeParams); err != nil {
		return req, err
	}
	if err := parseRemoveFields(raw, &req); err != nil {
		return req, err
	}
	if req.Project.Language == backend.LanguageJava {
		return req, validateJavaRemoveRequest(req)
	}
	return req, validateGoRemoveRequest(req)
}

func parseRemoveFields(raw map[string]any, req *RemoveReq) error {
	req.Project = backend.ProjectContext{Language: backend.LanguageGo}
	var err error
	if req.File, err = ParseString(raw, "file", "file", true); err != nil {
		return err
	}
	if req.Project, err = parseReadProject(raw, req.File); err != nil {
		return err
	} else if req.Project.Language == "" {
		req.Project.Language = backend.LanguageGo
	}
	kind, err := ParseString(raw, "kind", "kind", true)
	if err != nil {
		return err
	}
	req.Kind = backend.ConstructKind(kind)
	if req.Symbol, err = ParseString(raw, "symbol", "symbol", false); err != nil {
		return err
	}
	if req.InFunction, err = ParseString(raw, wireInFunction, wireCLIInFunction, false); err != nil {
		return err
	}
	if req.Discriminator, err = ParseString(raw, wireDiscriminator, wireDiscriminator, false); err != nil {
		return err
	}
	if req.TargetCase, err = ParseString(raw, "target_case", "target-case", false); err != nil {
		return err
	}
	if req.ConstructPath, err = ParseString(raw, wireConstructPath, wireCLIConstructPath, false); err != nil {
		return err
	}
	if req.SafeDelete, err = ParseBool(raw, removeSafeDelete, "safe-delete", false); err != nil {
		return err
	}
	if req.AutoOrganizeImports, err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true); err != nil {
		return err
	}
	return nil
}

func validateGoRemoveRequest(req RemoveReq) error {
	switch req.Kind {
	case backend.ConstructFunction, backend.ConstructMethod, backend.ConstructType, backend.ConstructConst, backend.ConstructVar:
		if req.Symbol == "" {
			return fmt.Errorf("symbol is required for kind=%s", req.Kind)
		}
		if req.InFunction != "" || req.Discriminator != "" || req.TargetCase != "" || req.ConstructPath != "" {
			return fmt.Errorf("nested selectors cannot be combined with declaration kind=%s", req.Kind)
		}
	case backend.ConstructCase:
		if req.InFunction == "" {
			return fmt.Errorf("in_function is required for kind=%s", req.Kind)
		}
		if req.Symbol != "" {
			return fmt.Errorf("symbol is a declaration target and cannot select a case; use in_function and target_case")
		}
	case backend.ConstructStatementOrExpr:
		if req.InFunction == "" || req.Discriminator == "" {
			return fmt.Errorf("in_function and discriminator are required for kind=%s", req.Kind)
		}
		if req.Symbol != "" || req.TargetCase != "" {
			return fmt.Errorf("declaration and case selectors cannot be combined with kind=%s", req.Kind)
		}
	default:
		return fmt.Errorf("unsupported removal kind %q", req.Kind)
	}
	if req.SafeDelete && req.Kind != backend.ConstructFunction && req.Kind != backend.ConstructConst && req.Kind != backend.ConstructVar && req.Kind != backend.ConstructType {
		return fmt.Errorf("safe_delete is supported only for function, const, var, or bounded type removal")
	}
	return nil
}

func validateJavaRemoveRequest(req RemoveReq) error {
	if req.SafeDelete && !isJavaDeclarationRemovalKind(req.Kind) {
		return fmt.Errorf("java safe_delete requires a declaration kind")
	}
	switch req.Kind {
	case backend.ConstructMethod, backend.ConstructConstructor, backend.ConstructField, backend.ConstructType,
		backend.ConstructClass, backend.ConstructInterface, backend.ConstructEnum:
		if strings.TrimSpace(req.Symbol) == "" || req.InFunction != "" || req.Discriminator != "" || req.TargetCase != "" || req.ConstructPath != "" {
			return fmt.Errorf("java declaration removal requires only a unique symbol selector")
		}
	case backend.ConstructLoop, backend.ConstructIf, backend.ConstructElse, backend.ConstructCase,
		javacst.ConstructSwitch, backend.ConstructTryCatch, backend.ConstructTryWithResources,
		backend.ConstructSynchronized, backend.ConstructStatementOrExpr:
		if strings.TrimSpace(req.InFunction) == "" || strings.TrimSpace(req.Discriminator) == "" || req.Symbol != "" || req.TargetCase != "" {
			return fmt.Errorf("java nested removal requires in_function and discriminator")
		}
	default:
		return fmt.Errorf("java remove_construct does not support kind=%s", req.Kind)
	}
	return nil
}

func runRemove(ctx context.Context, cc CallContext, req RemoveReq) (RemoveRes, error) {
	if req.Project.Language == backend.LanguageJava {
		return runRemoveJava(ctx, cc, req)
	}
	targetPath := resolveWorkPath(cc.WorkDir, req.File)
	finishDelta := surroundingDelta(ctx, cc.WorkDir, cc.DeferVerification)
	diff, checks, refusal, err := astedit.Remove(ctx, targetPath, astedit.RemoveOptions{
		Kind: string(req.Kind), Symbol: req.Symbol, InFunction: req.InFunction, Discriminator: req.Discriminator,
		TargetCase: req.TargetCase, ConstructPath: req.ConstructPath, SafeDelete: req.SafeDelete,
	}, effectiveAutoOrganize(cc, req.AutoOrganizeImports))
	if err != nil {
		if snapshotErr, ok := errors.AsType[*astedit.SafeDeleteInputConflictError](err); ok {
			return RemoveRes{}, &SafeDeleteRefusalError{
				File: req.File, Symbol: req.Symbol, InFunction: req.InFunction,
				Discriminator: req.Discriminator, TargetCase: req.TargetCase,
				ConstructPath: req.ConstructPath, ChecksRun: checks, Reason: snapshotErr.Error(),
			}
		}
		return RemoveRes{}, err
	}
	if refusal != "" {
		return RemoveRes{}, &SafeDeleteRefusalError{
			File: req.File, Symbol: req.Symbol, InFunction: req.InFunction,
			Discriminator: req.Discriminator, TargetCase: req.TargetCase,
			ConstructPath: req.ConstructPath, ChecksRun: checks, Reason: refusal,
		}
	}
	delta, diagnosticStatus := finishDelta()
	status := "removed"
	return RemoveRes{
		File: targetPath, Display: displayWorkPath(cc.WorkDir, req.File),
		InFunction: req.InFunction, Discriminator: req.Discriminator, TargetCase: req.TargetCase,
		ConstructPath: req.ConstructPath, SafeDelete: req.SafeDelete, Status: status,
		ChecksRun: checks, Refusal: refusal, Diff: diff, DiagnosticStatus: diagnosticStatus,
		Delta: delta, HasDelta: true,
	}, nil
}

func runRemoveJava(ctx context.Context, cc CallContext, req RemoveReq) (result RemoveRes, retErr error) {
	root, rootErr := filepath.Abs(cc.WorkDir)
	if rootErr != nil {
		return result, fmt.Errorf("resolve Java removal workspace: %w", rootErr)
	}
	file := resolveWorkPath(root, req.File)
	project := effectiveProject(cc, req.Project)
	project.RootDir, project.File = root, file
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	display := displayWorkPath(root, req.File)
	var before, after []byte
	var checks []string
	runErr := sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
		var readErr error
		before, readErr = sourcebuf.ReadFile(editCtx, file)
		if readErr != nil {
			return fmt.Errorf("read Java removal target: %w", readErr)
		}
		updated, removeErr := prepareJavaRemoval(editCtx, service, project, req, display, before, &checks)
		if removeErr != nil {
			return removeErr
		}
		if err := javacst.ValidateSource(updated); err != nil {
			return fmt.Errorf("java removal produces invalid source: %w", err)
		}
		if err := pipeline.WriteAtomicContext(editCtx, file, updated); err != nil {
			return fmt.Errorf("stage Java construct removal: %w", err)
		}
		if _, verifyErr := service.Verify(editCtx, backend.VerifyRequest{Project: project, Path: file, FormatSelectedFile: true, OrganizeImports: effectiveAutoOrganize(cc, req.AutoOrganizeImports)}); verifyErr != nil {
			return fmt.Errorf("verify Java construct removal: %w", verifyErr)
		}
		after, readErr = sourcebuf.ReadFile(editCtx, file)
		if readErr != nil {
			return fmt.Errorf("read verified Java removal result: %w", readErr)
		}
		return nil
	})
	if runErr != nil {
		return result, runErr
	}
	return RemoveRes{
		File: file, Display: display, Symbol: req.Symbol, InFunction: req.InFunction, Discriminator: req.Discriminator,
		TargetCase: req.TargetCase, ConstructPath: req.ConstructPath, SafeDelete: req.SafeDelete, ChecksRun: checks,
		Status: "removed", Diff: astedit.UnifiedDiff(display, string(before), string(after)),
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: true,
	}, nil
}

func prepareJavaRemoval(ctx context.Context, service *backend.Service, project backend.ProjectContext, req RemoveReq, display string, before []byte, checks *[]string) ([]byte, error) {
	if isJavaDeclarationRemovalKind(req.Kind) {
		return removeJavaDeclaration(ctx, service, project, req, display, before, checks)
	}
	return removeJavaNestedConstruct(ctx, service, project, req, display, before)
}

func removeJavaDeclaration(ctx context.Context, service *backend.Service, project backend.ProjectContext, req RemoveReq, display string, before []byte, checks *[]string) ([]byte, error) {
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.Symbol})
	if err != nil {
		return nil, err
	}
	if inspection == nil || len(inspection.Matches) != 1 {
		return nil, fmt.Errorf("java method %q must resolve to one declaration in %s", req.Symbol, display)
	}
	target := inspection.Matches[0]
	targetSource := []byte(target.Source)
	err = javacst.ValidateDeclarationKind(targetSource, req.Kind)
	if err != nil {
		return nil, fmt.Errorf("validate Java removal target: %w", err)
	}
	if req.SafeDelete {
		err = verifyJavaSafeDelete(ctx, service, project, req, targetSource, checks)
		if err != nil {
			return nil, err
		}
	}
	start, err := javacst.ByteOffsetFromPosition(before, target.Range.Start)
	if err != nil {
		return nil, fmt.Errorf("resolve Java method removal offset: %w", err)
	}
	end := start + len(targetSource)
	if end > len(before) || !bytes.Equal(before[start:end], targetSource) {
		return nil, fmt.Errorf("java method source changed during removal")
	}
	updated, err := javacst.RemoveJavaDeclaration(before, start, end, targetSource)
	if err != nil {
		return nil, fmt.Errorf("remove Java declaration: %w", err)
	}
	return updated, nil
}

func verifyJavaSafeDelete(ctx context.Context, service *backend.Service, project backend.ProjectContext, req RemoveReq, targetSource []byte, checks *[]string) error {
	isPrivate, err := javacst.IsPrivateJavaDeclaration(targetSource, req.Kind)
	*checks = append(*checks, "declaration visibility")
	if err != nil || !isPrivate {
		reason := fmt.Sprintf("safe deletion requires a private Java %s", req.Kind)
		if err != nil {
			reason = err.Error()
		}
		return &SafeDeleteRefusalError{File: req.File, Symbol: req.Symbol, ChecksRun: *checks, Reason: reason}
	}
	references, err := service.FindReferences(ctx, backend.ReferencesRequest{Project: project, Symbol: req.Symbol})
	*checks = append(*checks, "JDT LS workspace references")
	if err != nil {
		return &SafeDeleteRefusalError{File: req.File, Symbol: req.Symbol, ChecksRun: *checks, Reason: fmt.Sprintf("JDT LS reference proof failed: %v", err)}
	}
	if references == nil || !references.Scope.Complete {
		return &SafeDeleteRefusalError{File: req.File, Symbol: req.Symbol, ChecksRun: *checks, Reason: "JDT LS reference analysis was incomplete"}
	}
	if len(references.References) > 0 {
		return &SafeDeleteRefusalError{File: req.File, Symbol: req.Symbol, ChecksRun: *checks, Reason: fmt.Sprintf("declaration has %d external reference(s)", len(references.References))}
	}
	return nil
}

func removeJavaNestedConstruct(ctx context.Context, service *backend.Service, project backend.ProjectContext, req RemoveReq, display string, before []byte) ([]byte, error) {
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: req.InFunction})
	if err != nil {
		return nil, err
	}
	if inspection == nil || len(inspection.Matches) != 1 {
		return nil, fmt.Errorf("java method %q must resolve to one declaration in %s", req.InFunction, display)
	}
	method := inspection.Matches[0]
	methodSource := []byte(method.Source)
	start, err := javacst.ByteOffsetFromPosition(before, method.Range.Start)
	if err != nil {
		return nil, fmt.Errorf("resolve Java method source offset: %w", err)
	}
	end := start + len(methodSource)
	if end > len(before) || !bytes.Equal(before[start:end], methodSource) {
		return nil, fmt.Errorf("java method source changed during construct removal")
	}
	candidate, _, err := javacst.Resolve(methodSource, req.Kind, req.Discriminator, req.ConstructPath)
	if err != nil {
		return nil, err
	}
	updatedMethod, err := javacst.RemoveCandidate(methodSource, candidate)
	if err != nil {
		return nil, err
	}
	return spliceJavaSource(before, start, end, updatedMethod), nil
}

func isJavaDeclarationRemovalKind(kind backend.ConstructKind) bool {
	switch kind {
	case backend.ConstructMethod, backend.ConstructConstructor, backend.ConstructField, backend.ConstructType,
		backend.ConstructClass, backend.ConstructInterface, backend.ConstructEnum:
		return true
	default:
		return false
	}
}

func spliceJavaSource(source []byte, start, end int, replacement []byte) []byte {
	updated := make([]byte, 0, len(source)-(end-start)+len(replacement))
	updated = append(updated, source[:start]...)
	updated = append(updated, replacement...)
	updated = append(updated, source[end:]...)
	return updated
}

func removeDef() Def[RemoveReq, RemoveRes] {
	return Def[RemoveReq, RemoveRes]{
		Key:            operationKeyRemoveConstruct,
		Summary:        "Use this tool instead of replace_file_content to remove one supported Go declaration or construct, or one Java method, constructor, field declaration, nested type, or method-local control-flow construct. Java safe_delete checks private declaration visibility and requires complete JDT LS workspace reference coverage with no usages. Syntax is validated and diagnostics are reported; structural removal does not prove behavioral equivalence.",
		Params:         removeParams,
		Level:          LevelFile,
		CLIName:        "remove-construct",
		MCPName:        semanticRemoveConstructToolName,
		Parse:          parseRemove,
		NormalizeInput: NormalizeConstructKindInput,
		Handlers:       map[backend.LanguageID]func(context.Context, CallContext, RemoveReq) (RemoveRes, error){backend.LanguageGo: runRemove, backend.LanguageJava: runRemoveJava},
		Format: func(res RemoveRes) (string, error) {
			if res.Status == "refused" {
				return fmt.Sprintf("Safe deletion refused for %s: %s (checks run: %s).", res.Display, res.Refusal, strings.Join(res.ChecksRun, ", ")), nil
			}
			text := fmt.Sprintf("Removed selected node from %s.", res.Display)
			if res.HasDelta {
				text += " diagnostic_status: " + res.DiagnosticStatus + "."
				if res.DiagnosticStatus == "changed" {
					text = AppendDiagnosticDelta(text, res.Delta)
				}
			}
			if res.SafeDelete {
				text += fmt.Sprintf("\nSafe deletion checks run: %s.", strings.Join(res.ChecksRun, ", "))
			}
			text += "\n" + res.Diff
			return text, nil
		},
		Output: &OperationOutputContract{ResultSchema: &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
			"status":               {Kind: SchemaString, Enum: []any{"removed"}},
			removeSafeDelete:       {Kind: SchemaBoolean},
			removeChecksRun:        {Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}},
			"refusal":              {Kind: SchemaString},
			"diff":                 {Kind: SchemaString},
			removeDiagnosticStatus: {Kind: SchemaString, Enum: []any{"changed", "unchanged", "deferred", diagnosticStatusUnavailable}},
			removeDiagnosticDelta:  diagnosticDeltaSchema(),
		}, Required: []string{"status", removeSafeDelete, removeChecksRun, "diff", removeDiagnosticStatus, removeDiagnosticDelta}}},
		ErrorOutput: &OperationErrorOutputContract{ResultSchema: &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
			"status":         {Kind: SchemaString, Enum: []any{"refused"}},
			removeSafeDelete: {Kind: SchemaBoolean, Enum: []any{true}},
			removeChecksRun:  {Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}},
			"refusal":        {Kind: SchemaString},
		}, Required: []string{"status", removeSafeDelete, removeChecksRun, "refusal"}}},
		ProjectOutput: func(res RemoveRes) (any, error) {
			checks := res.ChecksRun
			if checks == nil {
				checks = []string{}
			}
			return map[string]any{
				"status": res.Status, removeSafeDelete: res.SafeDelete, removeChecksRun: checks,
				"refusal": res.Refusal, "diff": res.Diff,
				removeDiagnosticStatus: res.DiagnosticStatus, removeDiagnosticDelta: diagnosticDeltaOutput(res.Delta),
			}, nil
		},
		ProjectErrorOutput: func(err error) (any, bool, error) {
			var refusal *SafeDeleteRefusalError
			if !errors.As(err, &refusal) {
				return nil, false, nil
			}
			checks := refusal.ChecksRun
			if checks == nil {
				checks = []string{}
			}
			return map[string]any{"status": "refused", removeSafeDelete: true, removeChecksRun: checks, "refusal": refusal.Reason}, true, nil
		},
		ExampleRaw:    map[string]any{"file": wireExampleFile, "kind": "function", "symbol": "obsoleteHelper"},
		BufferedBatch: true, Batchable: true,
	}
}
