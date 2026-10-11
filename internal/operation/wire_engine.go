// Package operation wires host-engine editing operations into the central registry.
package operation

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"semedit/internal/telemetry"
)

const diagnosticStatusUnavailable = "unavailable"

var (
	placementEnum  = []string{"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol"}
	accessEnum     = goModifiers()
	visibilityEnum = []string{"public", "private"}
	groupEnum      = []string{"append", "standalone"}
	caseEnum       = []string{"first", "last", "before_default", "before", "after"}
)

// AppendDiagnosticDelta appends a compiler diagnostic delta to a response base string.
// It mirrors the MCP server helper retired in the ingress migration step.
func AppendDiagnosticDelta(base string, delta pipeline.DiagnosticDelta) string {
	if len(delta.Introduced) == 0 && len(delta.Resolved) == 0 && len(delta.Suggestions) == 0 {
		return base
	}
	var sb strings.Builder
	sb.WriteString(base)
	if len(delta.Introduced) > 0 || len(delta.Resolved) > 0 {
		fmt.Fprintf(&sb, "\nDiagnostics delta (net %d):\n", delta.NetDelta)
		if len(delta.Resolved) > 0 {
			fmt.Fprintf(&sb, "Resolved:\n- %s\n", strings.Join(delta.Resolved, "\n- "))
		}
		if len(delta.Introduced) > 0 {
			fmt.Fprintf(&sb, "Introduced:\n- %s\n", strings.Join(delta.Introduced, "\n- "))
		}
	}
	if len(delta.Suggestions) > 0 {
		fmt.Fprintf(&sb, "\nActionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
	}
	return sb.String()
}

// resolveWorkPath mirrors the MCP server path resolution: relative paths join the workspace root.
func resolveWorkPath(workDir, file string) string {
	if file == "" {
		return workDir
	}
	if workDir != "" && !filepath.IsAbs(file) {
		return filepath.Join(workDir, file)
	}
	return file
}

func displayWorkPath(workDir, file string) string {
	if strings.TrimSpace(file) == "" {
		return ""
	}
	root, err := filepath.Abs(workDir)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(file))
	}
	target := resolveWorkPath(root, file)
	absolute, err := filepath.Abs(target)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(file))
	}
	relative, err := filepath.Rel(root, absolute)
	if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(relative)
	}
	return filepath.ToSlash(filepath.Clean(file))
}

// effectiveAutoOrganize applies the batch deferral flag over the parsed default.
func effectiveAutoOrganize(cc CallContext, parsed bool) bool {
	if cc.DeferImports {
		return false
	}
	return parsed
}

const automaticVerificationGuidance = " This standalone operation automatically verifies diagnostics and has no verification opt-out; do not call semantic_verify separately."

func surroundingDelta(ctx context.Context, workDir string, deferVerification bool) func() (pipeline.DiagnosticDelta, string) {
	if deferVerification {
		return func() (pipeline.DiagnosticDelta, string) { return pipeline.DiagnosticDelta{}, "deferred" }
	}
	before, beforeErr := pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationBefore), workDir)
	return func() (pipeline.DiagnosticDelta, string) {
		after, afterErr := pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationAfter), workDir)
		if beforeErr != nil || afterErr != nil {
			return pipeline.DiagnosticDelta{}, diagnosticStatusUnavailable
		}
		delta := pipeline.ComputeDelta(before, after)
		if len(delta.Introduced)+len(delta.Resolved)+len(delta.Suggestions) == 0 {
			return delta, "unchanged"
		}
		return delta, "changed"
	}
}

// FileEditRes reports a single-file mutation for response rendering and batch post-processing.
type FileEditRes struct {
	File                  string
	DestinationFile       string
	DestinationDisplay    string
	Display               string
	ChangedFiles          []string
	ApplicationState      string
	Symbol                string
	InType                string
	Line                  int
	InFunction            string
	Discriminator         string
	DiagnosticStatus      string
	Diff                  string
	Detail                string
	Delta                 pipeline.DiagnosticDelta
	HasDelta              bool
	ReceiptDraft          EditReceiptDraft
	Receipt               EditReceipt
	Resolve               ContinuationResolver
	SuppressContinuations bool
	WorkingView           bool
	Package               string
	ElementCandidate      *compositeSelectorCandidate
	SelectorContinuation  *compositeSelectorContinuation
}

func formatFileEdit(res FileEditRes) (string, error) {
	text := res.Detail
	if res.Display != "" {
		text += fmt.Sprintf(" file: %s.", res.Display)
	}
	if res.DestinationDisplay != "" {
		text += fmt.Sprintf(" destination_file: %s.", res.DestinationDisplay)
	}
	if res.Symbol != "" {
		text += fmt.Sprintf(" symbol: %s.", res.Symbol)
	}
	if res.InType != "" {
		text += fmt.Sprintf(" in_type: %s.", res.InType)
	}
	if res.Line > 0 {
		text += fmt.Sprintf(" line (navigation only): %d.", res.Line)
	}
	if res.InFunction != "" {
		text += fmt.Sprintf(" in_function: %s.", res.InFunction)
	}
	if res.Discriminator != "" {
		text += fmt.Sprintf(" discriminator: %s.", res.Discriminator)
	}
	if res.ApplicationState != "" {
		text += " application_state: " + res.ApplicationState + "."
	}
	if len(res.ChangedFiles) > 0 {
		text += " changed_files: " + strings.Join(res.ChangedFiles, ", ") + "."
	}
	if res.HasDelta {
		text += " diagnostic_status: " + res.DiagnosticStatus + "."
		if res.DiagnosticStatus == "changed" {
			text = AppendDiagnosticDelta(text, res.Delta)
		}
	}
	if res.Diff != "" {
		text += "\n" + res.Diff
	}
	return text, nil
}

func diagnosticDeltaSchema() ValueSchema {
	textList := ValueSchema{Kind: SchemaArray, Items: &ValueSchema{Kind: SchemaString}}
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"net_delta":   {Kind: SchemaInteger},
		"introduced":  textList,
		"resolved":    textList,
		"suggestions": textList,
	}, Required: []string{"net_delta", "introduced", "resolved"}}
}

func diagnosticDeltaOutput(delta pipeline.DiagnosticDelta) pipeline.DiagnosticDelta {
	if delta.Introduced == nil {
		delta.Introduced = []string{}
	}
	if delta.Resolved == nil {
		delta.Resolved = []string{}
	}
	if delta.Suggestions == nil {
		delta.Suggestions = []string{}
	}
	return delta
}

func receiverNameForDecl(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) != 1 {
		return ""
	}
	return receiverName(decl.Recv.List[0].Type)
}

func receiverName(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.StarExpr:
		return receiverName(node.X)
	case *ast.Ident:
		return node.Name
	case *ast.IndexExpr:
		return receiverName(node.X)
	case *ast.IndexListExpr:
		return receiverName(node.X)
	}
	return ""
}

func qualifiedGoTarget(receiver, name string) string {
	if receiver == "" {
		return name
	}
	return receiver + "." + name
}

// registerEngineOps adds the host-engine file and build operations to the registry.
func registerEngineOps(registry *Registry) error {
	{
		if err := Register(registry, assertionModeDef()); err != nil {
			return err
		}
		if err := Register(registry, insertConstructDef()); err != nil {
			return err
		}
		if err := Register(registry, insertDeclarationDef()); err != nil {
			return err
		}
		if err := Register(registry, insertFunctionDef()); err != nil {
			return err
		}
		if err := Register(registry, insertTypeDef()); err != nil {
			return err
		}
		if err := Register(registry, insertDeclDef()); err != nil {
			return err
		}
		if err := Register(registry, replaceDeclDef()); err != nil {
			return err
		}
		if err := Register(registry, removeDef()); err != nil {
			return err
		}
		if err := Register(registry, organizeImportsDef()); err != nil {
			return err
		}
		if err := Register(registry, addBuildDependencyDef()); err != nil {
			return err
		}
		if err := Register(registry, replaceBodyDef()); err != nil {
			return err
		}
		if err := Register(registry, replaceConstructDef()); err != nil {
			return err
		}
		if err := Register(registry, moveConstructDef()); err != nil {
			return err
		}
		if err := registerExtractOps(registry); err != nil {
			return err
		}
		if err := Register(registry, inlineVariableDef()); err != nil {
			return err
		}
		if err := Register(registry, migrateTypeDef()); err != nil {
			return err
		}
		if err := Register(registry, inlineFunctionDef()); err != nil {
			return err
		}
		if err := Register(registry, scaffoldFileDef()); err != nil {
			return err
		}
		if err := Register(registry, insertCaseDef()); err != nil {
			return err
		}
		if err := Register(registry, changeSignatureDef()); err != nil {
			return fmt.Errorf("register change-signature: %w", err)
		}
		if err := registerSignatureExpansionOps(registry); err != nil {
			return err
		}
		return registerStructuralPlanOps(registry)
	}
}

func goTargetAfterEdit(ctx context.Context, filePath, source string) (string, int, error) {
	selector, selected, err := parseGoTargetSelector(source)
	if err != nil || !selected {
		return "", 0, err
	}
	content, err := sourcebuf.ReadFile(ctx, filePath) // #nosec G304 -- filePath is the resolved target of a validated edit operation.
	if err != nil {
		return "", 0, fmt.Errorf("read edited file %q: %w", filePath, err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", 0, fmt.Errorf("parse edited file %q: %w", filePath, err)
	}
	match := findGoTarget(file, fset, selector)
	if match.Count > 1 {
		return "", 0, nil
	}
	if match.Count == 1 {
		return match.Symbol, match.Line, nil
	}
	return "", 0, fmt.Errorf("inserted declaration target %q is absent from edited file %q", selector.Name, filePath)
}

type goTargetSelector struct {
	Name     string
	Receiver string
}

type goTargetMatch struct {
	Symbol string
	Line   int
	Count  int
}

func parseGoTargetSelector(source string) (goTargetSelector, bool, error) {
	prefix := "package p\n"
	if strings.HasPrefix(strings.TrimSpace(source), "package ") {
		prefix = ""
	}
	snippet, err := parser.ParseFile(token.NewFileSet(), "snippet.go", prefix+source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return goTargetSelector{}, false, fmt.Errorf("parse inserted declaration after write: %w", err)
	}
	if len(snippet.Decls) != 1 {
		return goTargetSelector{}, false, nil
	}
	var selector goTargetSelector
	switch decl := snippet.Decls[0].(type) {
	case *ast.FuncDecl:
		selector.Name = decl.Name.Name
		if decl.Recv != nil && len(decl.Recv.List) == 1 {
			selector.Receiver = receiverName(decl.Recv.List[0].Type)
			if selector.Receiver == "" {
				return goTargetSelector{}, false, nil
			}
		}
	case *ast.GenDecl:
		if len(decl.Specs) != 1 {
			return goTargetSelector{}, false, nil
		}
		switch spec := decl.Specs[0].(type) {
		case *ast.TypeSpec:
			selector.Name = spec.Name.Name
		case *ast.ValueSpec:
			if len(spec.Names) != 1 {
				return goTargetSelector{}, false, nil
			}
			selector.Name = spec.Names[0].Name
		}
	}
	if selector.Name == "" {
		return goTargetSelector{}, false, nil
	}
	return selector, true, nil
}

func findGoTarget(file *ast.File, fset *token.FileSet, selector goTargetSelector) goTargetMatch {
	var result goTargetMatch
	for _, declaration := range file.Decls {
		match := goTargetInDeclaration(declaration, fset, selector)
		if match.Count == 0 {
			continue
		}
		result.Count += match.Count
		if result.Count > 1 {
			return goTargetMatch{Count: result.Count}
		}
		result.Symbol = match.Symbol
		result.Line = match.Line
	}
	return result
}

func goTargetInDeclaration(declaration ast.Decl, fset *token.FileSet, selector goTargetSelector) goTargetMatch {
	switch node := declaration.(type) {
	case *ast.FuncDecl:
		if node.Name.Name != selector.Name || receiverNameForDecl(node) != selector.Receiver {
			return goTargetMatch{}
		}
		return goTargetMatch{Symbol: qualifiedGoTarget(selector.Receiver, selector.Name), Line: fset.Position(node.Pos()).Line, Count: 1}
	case *ast.GenDecl:
		return goTargetInSpecs(node.Specs, fset, selector.Name)
	default:
		return goTargetMatch{}
	}
}

func goTargetInSpecs(specs []ast.Spec, fset *token.FileSet, name string) goTargetMatch {
	var result goTargetMatch
	for _, spec := range specs {
		if !goSpecDeclaresName(spec, name) {
			continue
		}
		result.Count++
		if result.Count == 1 {
			result.Symbol = name
			result.Line = fset.Position(spec.Pos()).Line
		}
	}
	return result
}

func goSpecDeclaresName(spec ast.Spec, name string) bool {
	switch item := spec.(type) {
	case *ast.TypeSpec:
		return item.Name.Name == name
	case *ast.ValueSpec:
		for _, candidate := range item.Names {
			if candidate.Name == name {
				return true
			}
		}
	}
	return false
}

func goExistingTargetAfterEdit(ctx context.Context, filePath, symbol string) (string, int, error) {
	content, err := sourcebuf.ReadFile(ctx, filePath) // #nosec G304 -- filePath is the resolved target of a validated edit operation.
	if err != nil {
		return "", 0, fmt.Errorf("read edited file %q: %w", filePath, err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", 0, fmt.Errorf("parse edited file %q: %w", filePath, err)
	}
	var line int
	count := 0
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, raw := range gen.Specs {
			matched := false
			switch spec := raw.(type) {
			case *ast.TypeSpec:
				matched = spec.Name.Name == symbol
			case *ast.ValueSpec:
				for _, name := range spec.Names {
					matched = matched || name.Name == symbol
				}
			}
			if matched {
				count++
				line = fset.Position(raw.Pos()).Line
			}
		}
	}
	if count != 1 {
		return "", 0, nil
	}
	return symbol, line, nil
}

func runBufferedEdit[T any](ctx context.Context, cc CallContext, root, operationName string, edit func(context.Context, CallContext) (T, error)) (T, error) {
	var result T
	err := sourcebuf.Run(ctx, root, func(editCtx context.Context) error {
		before := sourcebuf.FromContext(editCtx).Checkpoint()
		editCC := cc
		editCC.Ctx = editCtx
		value, err := edit(editCtx, editCC)
		if err != nil {
			return err
		}
		result, err = attachBufferedReceipt(value, sourcebuf.FromContext(editCtx), before, cc, operationName)
		return err
	})
	return result, err
}

func attachBufferedReceipt[T any](value T, buffer *sourcebuf.Buffer, before sourcebuf.Checkpoint, cc CallContext, operationName string) (T, error) {
	fileEdit, ok := any(value).(FileEditRes)
	if !ok {
		return value, nil
	}
	fileEdit.WorkingView = sourcebuf.FromContext(cc.Ctx) != nil
	fileEdit.SuppressContinuations = cc.InBatch || fileEdit.WorkingView
	if fileEdit.ReceiptDraft.Operation == "" {
		draft, err := bufferedReceiptDraft(buffer, before, cc, operationName, fileEdit)
		if err != nil {
			return value, err
		}
		fileEdit.ReceiptDraft = draft
		fileEdit.Resolve = passthroughContinuation
	}
	return any(fileEdit).(T), nil
}

func bufferedReceiptDraft(buffer *sourcebuf.Buffer, before sourcebuf.Checkpoint, cc CallContext, operationName string, result FileEditRes) (EditReceiptDraft, error) {
	workspaceRoot, err := filepath.Abs(cc.WorkDir)
	if err != nil {
		return EditReceiptDraft{}, fmt.Errorf("resolve receipt workspace %q: %w", cc.WorkDir, err)
	}
	draft := EditReceiptDraft{Operation: operationName, Status: EditNoOp}
	for _, change := range buffer.ChangesSince(before) {
		preimage := change.Before
		relative, err := filepath.Rel(workspaceRoot, change.Path)
		if err != nil {
			return EditReceiptDraft{}, fmt.Errorf("derive receipt path for %q: %w", change.Path, err)
		}
		if !filepath.IsLocal(relative) {
			return EditReceiptDraft{}, fmt.Errorf("receipt path %q escapes workspace %q", change.Path, workspaceRoot)
		}
		path := filepath.ToSlash(relative)
		draft.ReadSet = append(draft.ReadSet, FileRevision{Path: path, Revision: structuralRevision(preimage)})
		draft.PlannedChanges = append(draft.PlannedChanges, FileRevision{Path: path, Revision: structuralRevision(change.After)})
	}
	if len(draft.PlannedChanges) > 0 {
		draft.Status = EditPersisted
		if result.DiagnosticStatus == "changed" {
			draft.Status = EditPersistedWithDiagnostics
		}
	}
	if target, ok := bufferedContinuationTarget(result, draft.PlannedChanges); ok {
		draft.Continuations = []SymbolTarget{target}
	}
	return draft, nil
}

func bufferedContinuationTarget(result FileEditRes, changes []FileRevision) (SymbolTarget, bool) {
	file := result.DestinationDisplay
	if file == "" {
		file = result.Display
	}
	if file == "" || result.Symbol == "" {
		return SymbolTarget{}, false
	}
	for _, change := range changes {
		if change.Path == filepath.ToSlash(file) {
			revision := change.Revision
			return SymbolTarget{File: change.Path, Symbol: result.Symbol, ExpectedRevision: &revision}, true
		}
	}
	return SymbolTarget{}, false
}

func passthroughContinuation(_ context.Context, target SymbolTarget) (SymbolTarget, bool, error) {
	return target, true, nil
}

func bufferedFileEditHandler[Req any](operationName string, requestFile func(Req) string, handler func(context.Context, CallContext, Req) (FileEditRes, error)) func(context.Context, CallContext, Req) (FileEditRes, error) {
	return func(ctx context.Context, cc CallContext, request Req) (FileEditRes, error) {
		target := resolveWorkPath(cc.WorkDir, requestFile(request))
		root, err := structuralModuleRoot(target)
		if err != nil {
			return FileEditRes{}, fmt.Errorf("prepare %s source buffer: %w", operationName, err)
		}
		return runBufferedEdit(ctx, cc, root, operationName, func(editCtx context.Context, editCC CallContext) (FileEditRes, error) {
			return handler(editCtx, editCC, request)
		})
	}
}

func fileEditOutputContract(properties map[string]ValueSchema, continuations []OutputContinuation) *OperationOutputContract {
	allProperties := map[string]ValueSchema{
		"result":         {Kind: SchemaString},
		editReceiptField: EditReceiptSchema(),
	}
	maps.Copy(allProperties, properties)
	resultSchema := ValueSchema{Kind: SchemaObject, Properties: allProperties, Required: []string{"result", editReceiptField}}
	return &OperationOutputContract{ResultSchema: &resultSchema, Continuations: continuations}
}

func projectFileEditOutput(result FileEditRes, fields ...string) (any, error) {
	projected := map[string]any{"result": result.Detail, editReceiptField: result.Receipt}
	allowed := make(map[string]bool, len(fields))
	for _, field := range fields {
		allowed[field] = true
	}
	put := func(name string, value any) {
		if allowed[name] && value != nil && value != "" && value != 0 {
			projected[name] = value
		}
	}
	file := result.Display
	if result.DestinationDisplay != "" {
		file = result.DestinationDisplay
	}
	if !result.SuppressContinuations {
		put("file", file)
		put("symbol", result.Symbol)
		put("in_type", result.InType)
		put(wireInFunction, result.InFunction)
		put(wireDiscriminator, result.Discriminator)
	}
	put("line", result.Line)
	put("origin_file", result.Display)
	put("package", result.Package)
	put("diff", result.Diff)
	if result.DiagnosticStatus != "" {
		put("diagnostic_status", result.DiagnosticStatus)
	}
	return projected, nil
}

func finalizeFileEditOutput(cc CallContext, result FileEditRes) (FileEditRes, error) {
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	receipt, err := FinalizeEditReceipt(ctx, cc.WorkDir, result.WorkingView, cc.InBatch || result.SuppressContinuations, result.ReceiptDraft, result.Resolve)
	if err != nil {
		return FileEditRes{}, err
	}
	result.Receipt = receipt
	result.SuppressContinuations = result.SuppressContinuations || cc.InBatch
	return result, nil
}
