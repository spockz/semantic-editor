// Package astedit extracts safe contiguous statement regions into local helpers.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"
	"github.com/spockz/semantic-editor/internal/telemetry"
)

// ExtractKind selects the declaration form created by ExtractFunction.
type ExtractKind string

const (
	// ExtractKindFunction creates a package-level helper function.
	ExtractKindFunction ExtractKind = "function"
	// ExtractMethod creates a method with the owner receiver.
	ExtractMethod ExtractKind = "method"
)

// ExtractOptions describes a deliberately bounded extraction contract.
type ExtractOptions struct {
	Kind             ExtractKind
	StartLine        int
	EndLine          int
	ConstructPath    string
	EndConstructPath string
	ExpectedRevision string
}

type extractionOutput struct {
	name                string
	typeName            string
	typeValue           types.Type
	newBinding          bool
	existingInCallScope bool
}

type extractionInput struct {
	ctx             context.Context
	cleanPath       string
	ownedBuffer     *sourcebuf.Buffer
	standalone      bool
	content         []byte
	fset            *token.FileSet
	file            *ast.File
	typesInfo       *types.Info
	target          *ast.FuncDecl
	selected        []ast.Stmt
	selectionBlock  *ast.BlockStmt
	selectionSwitch *ast.SwitchStmt
}

type extractionCandidate struct {
	source      []byte
	declaration string
}

// ExtractFunction moves a contiguous region of supported statements from one
// lexical block into a helper at the same execution point, deriving its
// signature from resolved Go bindings.
func ExtractFunction(ctx context.Context, filePath, owner, helper string, opts ExtractOptions) (string, error) {
	input, candidate, err := prepareFunctionExtraction(ctx, filePath, owner, helper, opts)
	if err != nil {
		return "", err
	}
	return publishExtraction(input, candidate, owner)
}

func validateExtractRequest(helper string, opts ExtractOptions) error {
	if opts.Kind != ExtractKindFunction && opts.Kind != ExtractMethod {
		return fmt.Errorf("unsupported extraction kind %q", opts.Kind)
	}
	switch {
	case opts.ConstructPath != "":
		if opts.ExpectedRevision == "" {
			return fmt.Errorf("expected_revision is required with construct_path")
		}
		if opts.StartLine != 0 || opts.EndLine != 0 {
			return fmt.Errorf("construct_path selection cannot include line selectors")
		}
		if err := validateExtractStatementPath(opts.ConstructPath); err != nil {
			return err
		}
		if opts.EndConstructPath != "" {
			if err := validateExtractStatementPath(opts.EndConstructPath); err != nil {
				return err
			}
		}
	case opts.EndConstructPath != "":
		return fmt.Errorf("end_construct_path requires construct_path")
	case opts.StartLine <= 0 || opts.EndLine < opts.StartLine:
		return fmt.Errorf("start_line and end_line must describe a positive ordered line range")
	}
	if !token.IsIdentifier(helper) || token.Lookup(helper).IsKeyword() || helper == "_" {
		return fmt.Errorf("symbol %q is not a valid helper identifier", helper)
	}
	if types.Universe.Lookup(helper) != nil {
		return fmt.Errorf("cannot extract: helper name %q collides with a predeclared Go identifier", helper)
	}
	return nil
}

func loadExtractionInput(ctx context.Context, filePath, owner string, opts ExtractOptions) (extractionInput, error) {
	input := extractionInput{cleanPath: filepath.Clean(filePath)}
	preparedCtx, ownedBuffer, standalone, err := prepareExtractionBuffer(ctx, input.cleanPath)
	if err != nil {
		return extractionInput{}, err
	}
	input.ctx, input.ownedBuffer, input.standalone = preparedCtx, ownedBuffer, standalone
	input.content, err = sourcebuf.ReadFile(input.ctx, input.cleanPath)
	if err != nil {
		return extractionInput{}, fmt.Errorf("read extraction target: %w", err)
	}
	if opts.ExpectedRevision != "" && opts.ExpectedRevision != goReadRevision(input.content) {
		return extractionInput{}, fmt.Errorf("expected_revision does not match the extraction source revision")
	}
	recv, name, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return extractionInput{}, fmt.Errorf("parse in_function selector: %w", err)
	}
	input.fset = token.NewFileSet()
	input.file, err = parser.ParseFile(input.fset, input.cleanPath, input.content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return extractionInput{}, &SyntaxError{File: input.cleanPath, Pos: extractSyntaxPosition(input.fset, err), Cause: err, Err: ErrSyntax}
	}
	loaded, err := loadExtractionPackages(input.ctx, filepath.Dir(input.cleanPath), map[string][]byte{input.cleanPath: input.content}, extractionLoadOptions{
		tests: strings.HasSuffix(filepath.Base(input.cleanPath), "_test.go"), typeInfo: true,
	})
	if err != nil {
		return extractionInput{}, fmt.Errorf("load typed extraction input: %w", err)
	}
	_, input.fset, input.file, input.typesInfo, err = typedExtractionFile(loaded, input.cleanPath)
	if err != nil {
		return extractionInput{}, err
	}
	input.target = findExtractOwner(input.file, name, recv)
	if input.target == nil {
		return extractionInput{}, &symbol.SymbolError{Op: "extract_function", File: input.cleanPath, Symbol: owner, Err: symbol.ErrNotFound}
	}
	if ownerErr := validateExtractOwner(input.target, owner, opts.Kind); ownerErr != nil {
		return extractionInput{}, ownerErr
	}
	input.selected, input.selectionBlock, err = selectExtractStatements(input.fset, input.target.Body, owner, opts)
	if err != nil {
		return extractionInput{}, err
	}
	if err := validateExtractSwitchAssignment(&input, owner); err != nil {
		return extractionInput{}, err
	}
	return input, nil
}

func planExtraction(input extractionInput, owner, helper string, opts ExtractOptions) (extractionCandidate, error) {
	if commentErr := validateExtractionCommentOwnership(input, owner); commentErr != nil {
		return extractionCandidate{}, commentErr
	}
	if helperNameShadowedAtReplacement(input, helper, opts.Kind) {
		return extractionCandidate{}, fmt.Errorf("cannot extract: generated call binding for %q is shadowed at the replacement site", helper)
	}
	if opts.Kind == ExtractMethod {
		if methodErr := validateExtractMethodHelper(input, helper); methodErr != nil {
			return extractionCandidate{}, methodErr
		}
	}
	collides, err := helperNameExists(input.ctx, input.cleanPath, input.file.Name.Name, helper, opts.Kind, input.target)
	if err != nil {
		return extractionCandidate{}, err
	}
	if collides {
		return extractionCandidate{}, fmt.Errorf("cannot extract: package declaration %q already exists", helper)
	}
	flow, err := analyzeExtractionFlow(input, input.selected, opts.Kind)
	if err != nil {
		return extractionCandidate{}, fmt.Errorf("cannot extract %s: %w", owner, err)
	}
	outputs := flow.outputs

	if typeNameErr := validateOutputTypeNames(input.ctx, input.cleanPath, input.file, outputs, owner, input.content); typeNameErr != nil {
		return extractionCandidate{}, typeNameErr
	}
	if outputErr := validateExtractionOutputCallBindings(input, helper, outputs, opts.Kind); outputErr != nil {
		return extractionCandidate{}, outputErr
	}
	params := flow.parameters
	if forwardedErr := validateTypedExtractionAliases(input, input.selected, flow, owner, opts.Kind); forwardedErr != nil {
		return extractionCandidate{}, forwardedErr
	}
	if scopeErr := validateTypedExtractedScope(input, input.selected, flow, owner); scopeErr != nil {
		return extractionCandidate{}, scopeErr
	}
	selected := input.selected
	selectedSource := strings.TrimSpace(string(input.content[input.fset.Position(selected[0].Pos()).Offset:input.fset.Position(selected[len(selected)-1].End()).Offset]))
	if len(flow.locals) > 0 {
		selectedSource = strings.Join(flow.locals, "\n") + "\n" + selectedSource
	}
	if len(flow.unusedLocals) > 0 {
		selectedSource += "\n_ = " + strings.Join(flow.unusedLocals, "\n_ = ")
	}
	callSource := extractionCall(input.target, helper, params.names, outputs, opts.Kind)
	if len(flow.unusedOwnerBindings) > 0 {
		callSource += "\n_ = " + strings.Join(flow.unusedOwnerBindings, "\n_ = ")
	}
	declaration, err := extractionDeclaration(input.fset, input.target, helper, opts.Kind, params.declarations, outputs, selectedSource)
	if err != nil {
		return extractionCandidate{}, err
	}
	source, err := rewriteExtractedStatements(input.content, input.fset, input.target, input.selectionBlock, selected, callSource, declaration, owner, input.file)
	if err != nil {
		return extractionCandidate{}, err
	}
	return extractionCandidate{source: source, declaration: declaration}, nil
}

func validateExtractMethodHelper(input extractionInput, helper string) error {
	if ast.IsExported(helper) {
		return fmt.Errorf("cannot extract method: helper name %q must be unexported to preserve external interface satisfaction", helper)
	}
	receiverType := input.typesInfo.TypeOf(input.target.Recv.List[0].Type)
	if receiverType == nil {
		return fmt.Errorf("cannot extract method: receiver type is unresolved")
	}
	namedType := receiverType
	if pointer, ok := namedType.(*types.Pointer); ok {
		namedType = pointer.Elem()
	}
	named, ok := types.Unalias(namedType).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return fmt.Errorf("cannot extract method: receiver type %s is not a named package type", receiverType)
	}
	for _, methodSetType := range []types.Type{receiverType, types.NewPointer(named)} {
		if object, _, _ := types.LookupFieldOrMethod(methodSetType, true, named.Obj().Pkg(), helper); object != nil {
			return fmt.Errorf("cannot extract: method name %q already exists in the receiver method set or as a promoted field", helper)
		}
	}
	return nil
}

func publishExtraction(input extractionInput, candidate extractionCandidate, owner string) (string, error) {
	formatted, err := prepareFunctionExtractionOutput(input, candidate)
	if err != nil {
		return "", err
	}
	current, err := sourcebuf.ReadFile(input.ctx, input.cleanPath)
	if err != nil {
		return "", fmt.Errorf("recheck extraction target before publication: %w", err)
	}
	if !bytes.Equal(current, input.content) {
		return "", fmt.Errorf("cannot extract %s: source changed during extraction", owner)
	}
	return publishPreparedSource(input.ctx, sourcePublication{
		path: input.cleanPath, diffPath: filepath.Base(input.cleanPath), before: input.content, after: formatted,
		ownedBuffer: input.ownedBuffer, standalone: input.standalone, operation: "extract function",
	})
}

func normalizeExtractionNewlines(source []byte) []byte {
	return append(bytes.TrimRight(source, "\n"), '\n')
}

func findExtractOwner(file *ast.File, name, recv string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == name && extractReceiverName(fn.Recv) == recv {
			return fn
		}
	}
	return nil
}

func validateExtractOwner(target *ast.FuncDecl, owner string, kind ExtractKind) error {
	if target.Body == nil {
		return fmt.Errorf("cannot extract from %s: function has no body", owner)
	}
	if target.Type.TypeParams != nil && len(target.Type.TypeParams.List) > 0 {
		return fmt.Errorf("cannot extract from generic function %s", owner)
	}
	for _, parameter := range target.Type.Params.List {
		if _, variadic := parameter.Type.(*ast.Ellipsis); variadic {
			return fmt.Errorf("cannot extract from variadic function %s", owner)
		}
	}
	if target.Type.Results != nil {
		for _, field := range target.Type.Results.List {
			if len(field.Names) > 0 {
				return fmt.Errorf("cannot extract from %s: named result variables may be affected by the moved statements", owner)
			}
		}
	}
	if target.Recv != nil {
		if len(target.Recv.List) != 1 || len(target.Recv.List[0].Names) != 1 {
			return fmt.Errorf("cannot extract from %s: receiver must have one named variable", owner)
		}
		if extractionReceiverIsInstantiatedGeneric(target.Recv.List[0].Type) {
			return fmt.Errorf("cannot extract from generic receiver method %s", owner)
		}
	}
	if kind == ExtractMethod && target.Recv == nil {
		return fmt.Errorf("cannot extract method from function %s without a receiver", owner)
	}
	return nil
}

func validateExtractionCommentOwnership(input extractionInput, owner string) error {
	selectedStart := input.selected[0].Pos()
	selectedEnd := input.selected[len(input.selected)-1].End()
	for _, group := range input.file.Comments {
		if group.Pos() <= input.target.Body.Lbrace || group.End() >= input.target.Body.Rbrace {
			continue
		}
		if group.Pos() < selectedStart || group.End() > selectedEnd {
			return fmt.Errorf("cannot extract %s: comments outside the selected region require manual comment ownership review", owner)
		}
		if containsDirectiveGroup(group) {
			return fmt.Errorf("cannot extract %s: directive comments cannot be moved with selected statements", owner)
		}
	}
	return nil
}

func extractionReceiverIsInstantiatedGeneric(receiver ast.Expr) bool {
	for {
		switch item := receiver.(type) {
		case *ast.StarExpr:
			receiver = item.X
		case *ast.ParenExpr:
			receiver = item.X
		case *ast.IndexExpr, *ast.IndexListExpr:
			return true
		default:
			return false
		}
	}
}

func selectExtractStatements(fset *token.FileSet, body *ast.BlockStmt, owner string, opts ExtractOptions) ([]ast.Stmt, *ast.BlockStmt, error) {
	selected, block, err := findExtractStatementRange(fset, body, owner, opts)
	if err != nil {
		return nil, nil, err
	}
	if opts.ConstructPath != "" {
		opts.StartLine = fset.Position(selected[0].Pos()).Line
		opts.EndLine = fset.Position(selected[len(selected)-1].End()).Line
	}
	for i, stmt := range selected {
		if err := validateExtractStatement(fset, stmt, i, len(selected), owner, opts); err != nil {
			return nil, nil, err
		}
	}
	return selected, block, nil
}

func findExtractStatementRange(fset *token.FileSet, body *ast.BlockStmt, owner string, opts ExtractOptions) ([]ast.Stmt, *ast.BlockStmt, error) {
	if opts.ConstructPath != "" {
		return findExtractPathRange(body, owner, opts)
	}
	startIdx, endIdx := -1, -1
	for i, stmt := range body.List {
		pos, end := fset.Position(stmt.Pos()), fset.Position(stmt.End())
		if pos.Line == opts.StartLine {
			if startIdx >= 0 {
				return nil, nil, fmt.Errorf("start_line %d matches multiple statements in %s", opts.StartLine, owner)
			}
			startIdx = i
		}
		if end.Line == opts.EndLine {
			if endIdx >= 0 {
				return nil, nil, fmt.Errorf("end_line %d matches multiple statements in %s", opts.EndLine, owner)
			}
			endIdx = i
		}
	}
	if startIdx < 0 || endIdx < 0 || endIdx < startIdx {
		return nil, nil, fmt.Errorf("line range %d-%d does not select a contiguous top-level statement region in %s", opts.StartLine, opts.EndLine, owner)
	}
	return body.List[startIdx : endIdx+1], body, nil
}

type extractStatementLocation struct {
	block      *ast.BlockStmt
	index      int
	switchCase bool
}

func validateExtractStatementPath(path string) error {
	for part := range strings.SplitSeq(path, ".") {
		index, err := strconv.Atoi(part)
		if err != nil || index < 0 || strconv.Itoa(index) != part {
			return fmt.Errorf("construct_path %q must be a canonical numeric statement path", path)
		}
	}
	return nil
}

func findExtractStatementPath(root *ast.BlockStmt, target string, caseBlocks map[*ast.CaseClause]*ast.BlockStmt) (extractStatementLocation, bool) {
	var match extractStatementLocation
	var found bool
	var walkBlock func(*ast.BlockStmt, string, bool)
	var walkStatement func(ast.Stmt, string)
	walkBlock = func(block *ast.BlockStmt, prefix string, switchCase bool) {
		if block == nil || found {
			return
		}
		for index, stmt := range block.List {
			path := childLoopPath(prefix, index)
			if path == target {
				match = extractStatementLocation{block: block, index: index, switchCase: switchCase}
				found = true
				return
			}
			if switchCase {
				continue
			}
			walkStatement(stmt, path)
			if found {
				return
			}
		}
	}
	walkStatement = func(stmt ast.Stmt, path string) {
		switch item := stmt.(type) {
		case *ast.BlockStmt:
			walkBlock(item, childLoopPath(path, 0), false)
		case *ast.IfStmt:
			walkBlock(item.Body, childLoopPath(path, 0), false)
			if item.Else != nil {
				walkStatement(item.Else, childLoopPath(path, 1))
			}
		case *ast.ForStmt:
			walkBlock(item.Body, childLoopPath(path, 0), false)
		case *ast.RangeStmt:
			walkBlock(item.Body, childLoopPath(path, 0), false)
		case *ast.SwitchStmt:
			walkExtractSwitchCases(item, path, caseBlocks, walkBlock)
		case *ast.LabeledStmt:
			walkStatement(item.Stmt, childLoopPath(path, 0))
		}
	}
	walkBlock(root, "", false)
	return match, found
}

func validateExtractStatement(fset *token.FileSet, stmt ast.Stmt, index, count int, owner string, opts ExtractOptions) error {
	switch item := stmt.(type) {
	case *ast.ExprStmt:
		if _, ok := item.X.(*ast.CallExpr); !ok {
			return fmt.Errorf("cannot extract %s: selected statement on line %d is not a call expression", owner, fset.Position(stmt.Pos()).Line)
		}
	case *ast.AssignStmt:
		switch item.Tok {
		case token.DEFINE, token.ASSIGN:
			if len(item.Lhs) == 0 || len(item.Rhs) != len(item.Lhs) && len(item.Rhs) != 1 {
				return fmt.Errorf("cannot extract %s: assignments must have matching targets and values", owner)
			}
		default:
			return fmt.Errorf("cannot extract %s: only short declarations and plain assignments are supported", owner)
		}
	case *ast.IfStmt:
		if index != 0 || count != 1 {
			return fmt.Errorf("cannot extract %s: an if/else must be selected as one complete statement", owner)
		}
	case *ast.ForStmt:
		if index != 0 || count != 1 {
			return fmt.Errorf("cannot extract %s: a counted loop must be selected as one complete statement", owner)
		}
	case *ast.RangeStmt:
		if index != 0 || count != 1 {
			return fmt.Errorf("cannot extract %s: a range loop must be selected as one complete statement", owner)
		}
	default:
		return fmt.Errorf("cannot extract %s: selected statement on line %d is not a call, supported assignment, bounded if/else, or counted loop", owner, fset.Position(stmt.Pos()).Line)
	}
	if index == 0 && fset.Position(stmt.Pos()).Line != opts.StartLine || index == count-1 && fset.Position(stmt.End()).Line != opts.EndLine {
		return fmt.Errorf("line range %d-%d must match statement boundaries exactly", opts.StartLine, opts.EndLine)
	}
	return nil
}

func rejectSelectedRecoverCalls(selected []ast.Stmt, ownerName string) error {
	for _, stmt := range selected {
		var recoveryCall bool
		ast.Inspect(stmt, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok {
				if ident, ok := ast.Unparen(call.Fun).(*ast.Ident); ok && ident.Name == "recover" {
					recoveryCall = true
					return false
				}
			}
			return !recoveryCall
		})
		if recoveryCall {
			return fmt.Errorf("cannot extract %s: selected direct recover() calls change recovery semantics when moved", ownerName)
		}
	}
	return nil
}

func extractionCall(target *ast.FuncDecl, helper string, params []string, outputs []extractionOutput, kind ExtractKind) string {
	args := append([]string(nil), params...)
	callName := helper
	if kind == ExtractMethod {
		receiver := target.Recv.List[0]
		callName = receiver.Names[0].Name + "." + helper
		args = removeString(args, receiver.Names[0].Name)
	}
	call := callName + "(" + strings.Join(args, ", ") + ")"
	if len(outputs) == 0 {
		return call
	}
	names := make([]string, len(outputs))
	declarations := make([]string, 0, len(outputs))
	hasNew, hasExisting := false, false
	for index, output := range outputs {
		names[index] = output.name
		if output.newBinding {
			hasNew = true
		} else {
			hasExisting = true
		}
	}
	operator := " = "
	canReuseExisting := true
	for _, output := range outputs {
		if !output.newBinding && !output.existingInCallScope {
			canReuseExisting = false
		}
	}
	if hasNew && (!hasExisting || canReuseExisting) {
		operator = " := "
	} else if hasNew {
		for _, output := range outputs {
			if output.newBinding {
				declarations = append(declarations, "var "+output.name+" "+output.typeName)
			}
		}
	}
	assignment := strings.Join(names, ", ") + operator + call
	if len(declarations) == 0 {
		return assignment
	}
	return strings.Join(declarations, "\n") + "\n" + assignment
}

func validateExtractionOutputCallBindings(input extractionInput, helper string, outputs []extractionOutput, kind ExtractKind) error {
	hasNew, hasAncestor := false, false
	for _, output := range outputs {
		if output.newBinding {
			hasNew = true
		}
		if !output.newBinding && !output.existingInCallScope {
			hasAncestor = true
		}
	}
	if hasNew && hasAncestor {
		for _, output := range outputs {
			if output.newBinding {
				if err := validatePredeclaredExtractionOutputType(input, output.typeValue); err != nil {
					return fmt.Errorf("cannot extract: %w", err)
				}
			}
		}
	}
	for _, output := range outputs {
		if output.newBinding && output.name == helper {
			return fmt.Errorf("cannot extract: new output %q would shadow the generated helper call", output.name)
		}
		if output.newBinding && kind == ExtractMethod && input.target.Recv != nil && len(input.target.Recv.List[0].Names) == 1 && output.name == input.target.Recv.List[0].Names[0].Name {
			return fmt.Errorf("cannot extract: new output %q would shadow the method receiver at the generated call", output.name)
		}
	}
	return nil
}

func validatePredeclaredExtractionOutputType(input extractionInput, outputType types.Type) error {
	functionScope := input.typesInfo.Scopes[input.target.Type]
	if functionScope == nil {
		return fmt.Errorf("result type binding scope is unavailable")
	}
	scope := functionScope.Innermost(input.selected[0].Pos())
	if scope == nil {
		return fmt.Errorf("result type binding scope is unavailable")
	}
	return validatePredeclaredExtractionType(outputType, input, scope, make(map[types.Type]bool))
}

func validatePredeclaredExtractionType(typ types.Type, input extractionInput, scope *types.Scope, seen map[types.Type]bool) error {
	if seen[typ] {
		return nil
	}
	seen[typ] = true
	position := input.selected[0].Pos()
	switch resolved := typ.(type) {
	case *types.Alias:
		if err := validatePredeclaredExtractionTypeObject(input, scope, position, resolved.Obj()); err != nil {
			return err
		}
	case *types.Basic:
		universeObject, ok := types.Universe.Lookup(resolved.Name()).(*types.TypeName)
		if ok {
			if err := validatePredeclaredExtractionTypeName(scope, position, universeObject); err != nil {
				return err
			}
		}
	case *types.Named:
		if err := validatePredeclaredExtractionTypeObject(input, scope, position, resolved.Obj()); err != nil {
			return err
		}
	}
	for _, child := range predeclaredExtractionTypeChildren(typ) {
		if err := validatePredeclaredExtractionType(child, input, scope, seen); err != nil {
			return err
		}
	}
	return nil
}

func validatePredeclaredExtractionTypeObject(input extractionInput, scope *types.Scope, position token.Pos, expected *types.TypeName) error {
	if expected.Pkg() == nil || expected.Pkg() == input.typesInfo.Defs[input.target.Name].Pkg() {
		return validatePredeclaredExtractionTypeName(scope, position, expected)
	}
	return validatePredeclaredExtractionPackageName(input, scope, position, expected.Pkg())
}

func predeclaredExtractionTypeChildren(typ types.Type) []types.Type {
	switch resolved := typ.(type) {
	case *types.Alias:
		return predeclaredExtractionTypeList(resolved.TypeArgs())
	case *types.Named:
		return predeclaredExtractionTypeList(resolved.TypeArgs())
	case *types.Pointer:
		return []types.Type{resolved.Elem()}
	case *types.Array:
		return []types.Type{resolved.Elem()}
	case *types.Slice:
		return []types.Type{resolved.Elem()}
	case *types.Chan:
		return []types.Type{resolved.Elem()}
	case *types.Map:
		return []types.Type{resolved.Key(), resolved.Elem()}
	case *types.Struct:
		children := make([]types.Type, 0, resolved.NumFields())
		for field := range resolved.Fields() {
			children = append(children, field.Type())
		}
		return children
	case *types.Signature:
		return []types.Type{resolved.Params(), resolved.Results()}
	case *types.Tuple:
		children := make([]types.Type, 0, resolved.Len())
		for variable := range resolved.Variables() {
			children = append(children, variable.Type())
		}
		return children
	case *types.Interface:
		resolved.Complete()
		children := make([]types.Type, 0, resolved.NumMethods()+resolved.NumEmbeddeds())
		for method := range resolved.Methods() {
			children = append(children, method.Type())
		}
		for embedded := range resolved.EmbeddedTypes() {
			children = append(children, embedded)
		}
		return children
	default:
		return nil
	}
}

func predeclaredExtractionTypeList(list *types.TypeList) []types.Type {
	if list == nil {
		return nil
	}
	children := make([]types.Type, 0, list.Len())
	for t := range list.Types() {
		children = append(children, t)
	}
	return children
}

func validatePredeclaredExtractionTypeName(scope *types.Scope, position token.Pos, expected *types.TypeName) error {
	_, actual := scope.LookupParent(expected.Name(), position)
	if actual != expected {
		return fmt.Errorf("result type %q is shadowed at the replacement site", expected.Name())
	}
	return nil
}

func validatePredeclaredExtractionPackageName(input extractionInput, scope *types.Scope, position token.Pos, expected *types.Package) error {
	for _, imported := range input.file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			return fmt.Errorf("decode import path for result type package %q: %w", expected.Name(), err)
		}
		if path != expected.Path() {
			continue
		}
		name := expected.Name()
		if imported.Name != nil {
			if imported.Name.Name == "." {
				return fmt.Errorf("dot import %q prevents proving the generated result type binding", path)
			}
			if imported.Name.Name != "_" {
				name = imported.Name.Name
			}
		}
		var pkgName *types.PkgName
		if imported.Name == nil {
			pkgName, _ = input.typesInfo.Implicits[imported].(*types.PkgName)
		} else {
			pkgName, _ = input.typesInfo.Defs[imported.Name].(*types.PkgName)
		}
		if pkgName == nil {
			return fmt.Errorf("import binding for result type package %q is unavailable", name)
		}
		_, actual := scope.LookupParent(name, position)
		if actual != pkgName {
			return fmt.Errorf("result type package name %q is shadowed at the replacement site", name)
		}
		return nil
	}
	return fmt.Errorf("result type package %q is not imported at the replacement site", expected.Name())
}

func rewriteExtractedStatements(content []byte, fset *token.FileSet, target *ast.FuncDecl, block *ast.BlockStmt, selected []ast.Stmt, callSource, declSource, owner string, file *ast.File) ([]byte, error) {
	if err := validateSelectedExtractionBlock(block, selected); err != nil {
		return nil, fmt.Errorf("cannot extract %s: %w", owner, err)
	}
	start := fset.Position(selected[0].Pos()).Offset
	end := fset.Position(selected[len(selected)-1].End()).Offset
	insert := normalizeInsertionOffset(fset, file, content, fset.Position(target.End()).Offset)
	if insert < end {
		return nil, fmt.Errorf("cannot extract %s: selected region overlaps the declaration boundary", owner)
	}
	updated := make([]byte, 0, len(content)+len(declSource)+len(callSource))
	updated = append(updated, content[:start]...)
	updated = append(updated, callSource...)
	updated = append(updated, content[end:insert]...)
	updated = append(updated, '\n', '\n')
	updated = append(updated, declSource...)
	suffix := content[insert:]
	if len(suffix) > 0 {
		lineEnding := []byte("\n")
		if bytes.HasPrefix(suffix, []byte("\r\n")) {
			lineEnding = []byte("\r\n")
		}
		leadingLineBreaks := 0
		for offset := 0; offset < len(suffix); {
			switch suffix[offset] {
			case '\n':
				leadingLineBreaks++
				offset++
			case '\r':
				if offset+1 >= len(suffix) || suffix[offset+1] != '\n' {
					offset = len(suffix)
					continue
				}
				leadingLineBreaks++
				offset += 2
			default:
				offset = len(suffix)
			}
		}
		for range max(0, 2-leadingLineBreaks) {
			updated = append(updated, lineEnding...)
		}
	}
	updated = append(updated, suffix...)
	return updated, nil
}

func validateSelectedExtractionBlock(block *ast.BlockStmt, selected []ast.Stmt) error {
	if block == nil || len(selected) == 0 {
		return fmt.Errorf("selected statements have no owning block")
	}
	start := -1
	for index, stmt := range block.List {
		if stmt == selected[0] {
			start = index
			break
		}
	}
	if start < 0 || start+len(selected) > len(block.List) {
		return fmt.Errorf("selected statements are not a contiguous range in one block")
	}
	for index, stmt := range selected {
		if block.List[start+index] != stmt {
			return fmt.Errorf("selected statements are not a contiguous range in one block")
		}
	}
	return nil
}

func parameterHasName(fields *ast.FieldList, name string) bool {
	if fields == nil {
		return false
	}
	for _, field := range fields.List {
		for _, ident := range field.Names {
			if ident.Name == name {
				return true
			}
		}
	}
	return false
}

func fieldListHasName(fields *ast.FieldList, name string) bool {
	return parameterHasName(fields, name)
}

func localBindings(body *ast.BlockStmt) map[string]bool {
	locals := make(map[string]bool)
	ast.Inspect(body, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.AssignStmt:
			if item.Tok == token.DEFINE {
				addIdentifierBindings(item.Lhs, locals)
			}
		case *ast.ValueSpec:
			for _, ident := range item.Names {
				locals[ident.Name] = true
			}
		case *ast.RangeStmt:
			if item.Tok == token.DEFINE {
				addIdentifierBindings([]ast.Expr{item.Key, item.Value}, locals)
			}
		}
		return true
	})
	return locals
}

func addIdentifierBindings(expressions []ast.Expr, locals map[string]bool) {
	for _, expr := range expressions {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name != "_" {
			locals[ident.Name] = true
		}
	}
}

type extractionParameterSet struct {
	names        []string
	declarations []string
	positions    map[string]token.Pos
}

func parentSelectorIdent(root ast.Node, target *ast.Ident) bool {
	matched := false
	ast.Inspect(root, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel == target {
			matched = true
			return false
		}
		return !matched
	})
	return matched
}

func extractionDeclaration(fset *token.FileSet, owner *ast.FuncDecl, helper string, kind ExtractKind, params []string, outputs []extractionOutput, body string) (string, error) {
	var prefix string
	if kind == ExtractMethod {
		receiver := owner.Recv.List[0]
		receiverType, err := nodeSource(fset, receiver.Type)
		if err != nil {
			return "", err
		}
		prefix = "func (" + receiver.Names[0].Name + " " + receiverType + ") " + helper
	} else {
		prefix = "func " + helper
	}
	result := ""
	bodySource := body
	if len(outputs) == 1 {
		result = " " + outputs[0].typeName
	} else if len(outputs) > 1 {
		resultTypes := make([]string, len(outputs))
		for index, output := range outputs {
			resultTypes[index] = output.typeName
		}
		result = " (" + strings.Join(resultTypes, ", ") + ")"
	}
	if len(outputs) > 0 {
		resultNames := make([]string, len(outputs))
		for index, output := range outputs {
			resultNames[index] = output.name
		}
		bodySource += "\nreturn " + strings.Join(resultNames, ", ")
	}
	decl := prefix + "(" + strings.Join(params, ", ") + ")" + result + " {\n" + bodySource + "\n}"
	formatted, err := format.Source([]byte(decl))
	if err != nil {
		return "", fmt.Errorf("format extracted declaration: %w", err)
	}
	return strings.TrimSpace(string(formatted)), nil
}

func nodeSource(fset *token.FileSet, node ast.Node) (string, error) {
	var out bytes.Buffer
	if err := format.Node(&out, fset, node); err != nil {
		return "", fmt.Errorf("format extraction syntax: %w", err)
	}
	return out.String(), nil
}

func removeString(values []string, value string) []string {
	for i, item := range values {
		if item == value {
			return append(values[:i], values[i+1:]...)
		}
	}
	return values
}

func helperNameExists(ctx context.Context, filePath, packageName, helper string, kind ExtractKind, owner *ast.FuncDecl) (bool, error) {
	entries, err := os.ReadDir(filepath.Dir(filePath))
	if err != nil {
		return false, fmt.Errorf("read Go package directory for helper collision check: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		candidate := filepath.Join(filepath.Dir(filePath), entry.Name())
		content, err := sourcebuf.ReadFile(ctx, candidate)
		if err != nil {
			return false, fmt.Errorf("read Go package file %q for helper collision check: %w", candidate, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), candidate, content, parser.ParseComments)
		if err != nil {
			return false, fmt.Errorf("parse Go package file %q for helper collision check: %w", candidate, err)
		}
		if parsed.Name.Name != packageName {
			continue
		}
		if packageDeclaresExtractionHelper(parsed, helper, kind, extractReceiverName(owner.Recv)) {
			return true, nil
		}
	}
	return false, nil
}

func packageDeclaresExtractionHelper(file *ast.File, helper string, kind ExtractKind, receiver string) bool {
	if kind == ExtractMethod {
		collision := false
		ast.Inspect(file, func(node ast.Node) bool {
			iface, ok := node.(*ast.InterfaceType)
			if !ok {
				return true
			}
			for _, field := range iface.Methods.List {
				if slices.ContainsFunc(field.Names, func(name *ast.Ident) bool { return name.Name == helper }) {
					collision = true
					return false
				}
			}
			return true
		})
		if collision {
			return true
		}
	}
	for _, decl := range file.Decls {
		if declarationMatchesExtractionHelper(decl, helper, kind, receiver) {
			return true
		}
	}
	return false
}

func declarationMatchesExtractionHelper(decl ast.Decl, helper string, kind ExtractKind, receiver string) bool {
	switch node := decl.(type) {
	case *ast.FuncDecl:
		if node.Name.Name != helper {
			return false
		}
		if kind == ExtractKindFunction {
			return node.Recv == nil
		}
		return node.Recv != nil && extractReceiverName(node.Recv) == receiver
	case *ast.GenDecl:
		if kind == ExtractMethod {
			return false
		}
		return generalDeclarationMatchesHelper(node, helper)
	default:
		return false
	}
}

func generalDeclarationMatchesHelper(decl *ast.GenDecl, helper string) bool {
	for _, spec := range decl.Specs {
		switch item := spec.(type) {
		case *ast.TypeSpec:
			if item.Name.Name == helper {
				return true
			}
		case *ast.ValueSpec:
			if slices.ContainsFunc(item.Names, func(name *ast.Ident) bool { return name.Name == helper }) {
				return true
			}
		}
	}
	return false
}

func validateExtractionCandidate(ctx context.Context, filePath string, source []byte) error {
	defer telemetry.StartPhase(ctx, "go.extract.validate_candidate")()
	overlay := sourcebuf.Overlay(ctx, ".go")
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	cleanPath := filepath.Clean(filePath)
	overlay[cleanPath] = bytes.Clone(source)
	testFile := strings.HasSuffix(filepath.Base(cleanPath), "_test.go")
	loaded, err := loadExtractionPackages(ctx, filepath.Dir(cleanPath), overlay, extractionLoadOptions{tests: true, typeInfo: true})
	if err != nil {
		return fmt.Errorf("validate complete extracted package: %w", err)
	}
	if testFile {
		if _, _, _, _, err := typedExtractionFile(loaded, cleanPath); err != nil {
			return fmt.Errorf("validate complete extracted test package: %w", err)
		}
	}
	return nil
}

func validateOutputTypeNames(ctx context.Context, filePath string, file *ast.File, outputs []extractionOutput, owner string, source []byte) error {
	defer telemetry.StartPhase(ctx, "go.extract.validate_original")()
	overlay := sourcebuf.Overlay(ctx, ".go")
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	overlay[filepath.Clean(filePath)] = bytes.Clone(source)
	testFile := strings.HasSuffix(filepath.Base(filePath), "_test.go")
	loaded, err := loadExtractionPackages(ctx, filepath.Dir(filePath), overlay, extractionLoadOptions{tests: testFile, typeInfo: true})
	if err != nil {
		return fmt.Errorf("resolve extracted result types: %w", err)
	}
	packageForFile, _, _, _, err := typedExtractionFile(loaded, filePath)
	if err != nil {
		return fmt.Errorf("resolve extracted result types: %w", err)
	}
	if packageForFile.Types == nil {
		return fmt.Errorf("resolve extracted result types: package scope is ambiguous")
	}
	for _, output := range outputs {
		universeType := types.Universe.Lookup(output.typeName)
		if universeType == nil {
			continue
		}
		if packageType := packageForFile.Types.Scope().Lookup(output.typeName); packageType != nil {
			return fmt.Errorf("cannot extract %s: generated result type %q is shadowed by a package declaration", owner, output.typeName)
		}
		for _, imported := range file.Imports {
			var alias string
			if imported.Name != nil {
				alias = imported.Name.Name
			} else {
				importPath, unquoteErr := strconv.Unquote(imported.Path.Value)
				if unquoteErr != nil {
					return fmt.Errorf("read import path while checking extracted result type: %w", unquoteErr)
				}
				importedPackage := packageForFile.Imports[importPath]
				if importedPackage == nil || importedPackage.Name == "" {
					return fmt.Errorf("cannot extract %s: imported package %q has no resolved package name", owner, importPath)
				}
				alias = importedPackage.Name
			}
			if alias == "." {
				return fmt.Errorf("cannot extract %s: dot imports prevent proving generated result type bindings", owner)
			}
			if alias != "_" && alias == output.typeName {
				return fmt.Errorf("cannot extract %s: generated result type %q conflicts with an imported package name", owner, output.typeName)
			}
		}
	}
	return nil
}

func helperNameShadowedAtReplacement(input extractionInput, helper string, kind ExtractKind) bool {
	if len(input.selected) == 0 || input.typesInfo == nil || input.target == nil {
		return true
	}
	functionScope := input.typesInfo.Scopes[input.target.Type]
	if functionScope == nil {
		return true
	}
	replacementScope := functionScope.Innermost(input.selected[0].Pos())
	if replacementScope == nil {
		replacementScope = functionScope
	}
	switch kind {
	case ExtractKindFunction:
		declaringScope, object := replacementScope.LookupParent(helper, input.selected[0].Pos())
		for scope := declaringScope; scope != nil; scope = scope.Parent() {
			if scope == functionScope || scope == input.typesInfo.Scopes[input.file] {
				return object != nil
			}
		}
		return false
	case ExtractMethod:
		receiver := input.target.Recv.List[0].Names[0]
		receiverObject := input.typesInfo.Defs[receiver]
		declaringScope, object := replacementScope.LookupParent(receiver.Name, input.selected[0].Pos())
		if receiverObject == nil || object != receiverObject {
			return true
		}
		for scope := declaringScope; scope != nil; scope = scope.Parent() {
			if scope == functionScope {
				return false
			}
		}
		return true
	default:
		return true
	}
}

func findExtractPathRange(body *ast.BlockStmt, owner string, opts ExtractOptions) ([]ast.Stmt, *ast.BlockStmt, error) {
	caseBlocks := make(map[*ast.CaseClause]*ast.BlockStmt)
	start, found := findExtractStatementPath(body, opts.ConstructPath, caseBlocks)
	if !found {
		return nil, nil, fmt.Errorf("construct_path %q does not select a supported statement in %s", opts.ConstructPath, owner)
	}
	end := start
	if opts.EndConstructPath != "" {
		end, found = findExtractStatementPath(body, opts.EndConstructPath, caseBlocks)
		if !found {
			return nil, nil, fmt.Errorf("end_construct_path %q does not select a supported statement in %s", opts.EndConstructPath, owner)
		}
	}
	if start.block != end.block {
		return nil, nil, fmt.Errorf("construct_path range %q-%q must select statements from the same block in %s", opts.ConstructPath, opts.EndConstructPath, owner)
	}
	if end.index < start.index {
		return nil, nil, fmt.Errorf("end_construct_path %q precedes construct_path %q in %s", opts.EndConstructPath, opts.ConstructPath, owner)
	}
	selected := start.block.List[start.index : end.index+1]
	if start.switchCase {
		directCalls := true
		for _, stmt := range selected {
			expression, ok := stmt.(*ast.ExprStmt)
			if !ok {
				directCalls = false
				break
			}
			if _, ok := expression.X.(*ast.CallExpr); !ok {
				directCalls = false
				break
			}
		}
		if !directCalls {
			if len(selected) != 1 {
				return nil, nil, fmt.Errorf("cannot extract %s: switch-case selections support direct call ranges or one existing-local assignment", owner)
			}
			assignment, ok := selected[0].(*ast.AssignStmt)
			if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
				return nil, nil, fmt.Errorf("cannot extract %s: switch-case selections support direct call ranges or one existing-local assignment", owner)
			}
		}
	}
	return selected, start.block, nil
}

func walkExtractSwitchCases(stmt *ast.SwitchStmt, path string, caseBlocks map[*ast.CaseClause]*ast.BlockStmt, walkBlock func(*ast.BlockStmt, string, bool)) {
	for index, statement := range stmt.Body.List {
		clause, ok := statement.(*ast.CaseClause)
		if !ok {
			continue
		}
		block := caseBlocks[clause]
		if block == nil {
			block = &ast.BlockStmt{List: clause.Body}
			caseBlocks[clause] = block
		}
		walkBlock(block, childLoopPath(path, index), true)
	}
}

func validateExtractSwitchAssignment(input *extractionInput, owner string) error {
	if len(input.selected) != 1 {
		return nil
	}
	assignment, ok := input.selected[0].(*ast.AssignStmt)
	if !ok {
		return nil
	}
	ast.Inspect(input.target.Body, func(node ast.Node) bool {
		switchStmt, isSwitch := node.(*ast.SwitchStmt)
		if !isSwitch {
			return true
		}
		for _, item := range switchStmt.Body.List {
			clause, isClause := item.(*ast.CaseClause)
			if isClause && slices.Contains(clause.Body, input.selected[0]) {
				input.selectionSwitch = switchStmt
				return false
			}
		}
		return true
	})
	if input.selectionSwitch == nil {
		return nil
	}
	if assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return fmt.Errorf("cannot extract %s: switch-case assignment must be a single existing-local or parameter assignment", owner)
	}
	target, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok {
		return fmt.Errorf("cannot extract %s: switch-case assignment target must be an existing owner local or named parameter", owner)
	}
	object, ok := input.typesInfo.Uses[target].(*types.Var)
	functionScope := input.typesInfo.Scopes[input.target.Type]
	if !ok || object.IsField() || object.Parent() == nil || functionScope == nil || !functionScope.Contains(object.Pos()) {
		return fmt.Errorf("cannot extract %s: switch-case assignment target must be an existing owner local or named parameter", owner)
	}
	if extractionReceiverObject(*input, object) {
		return fmt.Errorf("cannot extract %s: switch-case assignment does not support the owner receiver", owner)
	}
	parameter := false
	for _, field := range input.target.Type.Params.List {
		for _, name := range field.Names {
			if input.typesInfo.Defs[name] == object {
				parameter = true
				break
			}
		}
		if parameter {
			break
		}
	}
	if !parameter && object.Pos() >= input.selectionSwitch.Pos() {
		return fmt.Errorf("cannot extract %s: switch-case assignment target must be declared before the switch", owner)
	}
	if !extractObjectReadAfter(*input, input.selectionSwitch.End(), object) {
		return fmt.Errorf("cannot extract %s: switch-case assignment target must be read after the switch", owner)
	}
	return nil
}
