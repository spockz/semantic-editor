// Package astedit derives extraction inputs and outputs from Go object identity.
// WHY: Statement spelling cannot distinguish shadowed bindings or prove live values.
package astedit

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"
	"strconv"
)

type extractionFlow struct {
	parameters          extractionParameterSet
	outputs             []extractionOutput
	locals              []string
	unusedLocals        []string
	unusedOwnerBindings []string
	bindings            []string
	objects             []*types.Var
}

type extractionBinding struct {
	object          *types.Var
	name            string
	readBeforeWrite bool
	readAfterWrite  bool
	written         bool
	declaredInside  bool
}

func analyzeExtractionFlow(input extractionInput, selected []ast.Stmt, kind ExtractKind) (extractionFlow, error) {
	ordered, writtenOrder, err := collectExtractionBindings(input.typesInfo, selected)
	if err != nil {
		return extractionFlow{}, err
	}
	start, end := selected[0].Pos(), selected[len(selected)-1].End()
	flow := extractionFlow{}
	for _, binding := range ordered {
		binding.declaredInside = binding.object.Pos() >= start && binding.object.Pos() <= end
		flow.objects = append(flow.objects, binding.object)
		if !extractionObjectPackageScoped(binding.object) {
			flow.bindings = append(flow.bindings, binding.name)
		}
		if binding.readBeforeWrite && !binding.declaredInside && !extractionObjectPackageScoped(binding.object) && binding.object.Parent() != types.Universe {
			if err := appendExtractionBinding(input, &flow, binding.object, kind); err != nil {
				return extractionFlow{}, err
			}
		}
		if !binding.written {
			continue
		}
		if !binding.declaredInside && !extractionObjectPackageScoped(binding.object) && binding.object.Pos() >= input.target.Body.Pos() && binding.object.Pos() < start && !extractObjectUsedOutside(input, start, end, binding.object) {
			flow.unusedOwnerBindings = append(flow.unusedOwnerBindings, binding.name)
		}
		if !binding.declaredInside && !binding.readBeforeWrite && !extractionObjectPackageScoped(binding.object) && (kind != ExtractMethod || !extractionReceiverObject(input, binding.object)) {
			declaration, err := extractionBindingDeclaration(input, binding.object)
			if err != nil {
				return extractionFlow{}, err
			}
			flow.locals = append(flow.locals, declaration)
			if !binding.readAfterWrite && !extractObjectUsedAfter(input, end, binding.object) {
				flow.unusedLocals = append(flow.unusedLocals, binding.name)
			}
		}
	}
	sortExtractionParameters(&flow.parameters)
	if err := planExtractionOutputs(input, writtenOrder, end, &flow); err != nil {
		return extractionFlow{}, err
	}
	return flow, nil
}

func sortExtractionParameters(parameters *extractionParameterSet) {
	declarations := make(map[string]string, len(parameters.declarations))
	for _, declaration := range parameters.declarations {
		for index := range declaration {
			if declaration[index] == ' ' {
				declarations[declaration[:index]] = declaration
				break
			}
		}
	}
	sort.SliceStable(parameters.names, func(i, j int) bool {
		return parameters.positions[parameters.names[i]] < parameters.positions[parameters.names[j]]
	})
	parameters.declarations = parameters.declarations[:0]
	for _, name := range parameters.names {
		if declaration, ok := declarations[name]; ok {
			parameters.declarations = append(parameters.declarations, declaration)
		}
	}
}

func collectExtractionBindings(info *types.Info, selected []ast.Stmt) ([]*extractionBinding, []*extractionBinding, error) {
	bindings := make(map[*types.Var]*extractionBinding)
	ordered, writtenOrder := make([]*extractionBinding, 0), make([]*extractionBinding, 0)
	for _, stmt := range selected {
		visit := func(object *types.Var, read bool) {
			binding := extractionFlowBinding(bindings, &ordered, object)
			if read {
				if !binding.written {
					binding.readBeforeWrite = true
				} else {
					binding.readAfterWrite = true
				}
				return
			}
			if !binding.written {
				writtenOrder = append(writtenOrder, binding)
			}
			binding.written = true
		}
		if err := visitExtractFlowStatement(info, stmt, visit); err != nil {
			return nil, nil, err
		}
	}
	return ordered, writtenOrder, nil
}

func planExtractionOutputs(input extractionInput, bindings []*extractionBinding, end token.Pos, flow *extractionFlow) error {
	functionScope := input.typesInfo.Scopes[input.target.Type]
	callScope := functionScope.Innermost(input.selected[0].Pos())
	for _, binding := range bindings {
		if extractionObjectPackageScoped(binding.object) || !extractObjectUsedAfter(input, end, binding.object) {
			continue
		}
		if extractionTypeHasLocalName(binding.object.Type()) {
			return fmt.Errorf("cannot extract: selected value %q has a type or binding that cannot be named by a helper", binding.name)
		}
		resultType, err := extractionTypeString(input, binding.object.Type())
		if err != nil {
			return fmt.Errorf("cannot extract result %q: %w", binding.name, err)
		}
		flow.outputs = append(flow.outputs, extractionOutput{
			name:                binding.name,
			typeName:            resultType,
			typeValue:           binding.object.Type(),
			newBinding:          binding.declaredInside,
			existingInCallScope: !binding.declaredInside && binding.object.Parent() == callScope,
		})
		if binding.declaredInside && !extractObjectReadAfter(input, end, binding.object) {
			flow.unusedOwnerBindings = append(flow.unusedOwnerBindings, binding.name)
		}
	}
	return nil
}

func extractionReceiverObject(input extractionInput, object *types.Var) bool {
	return input.target.Recv != nil && len(input.target.Recv.List[0].Names) == 1 && input.typesInfo.Defs[input.target.Recv.List[0].Names[0]] == object
}

func validateTypedExtractionAliases(input extractionInput, selected []ast.Stmt, flow extractionFlow, owner string, kind ExtractKind) error {
	objects := make(map[*types.Var]bool, len(flow.objects)+1)
	for _, object := range flow.objects {
		if !extractionObjectPackageScoped(object) {
			objects[object] = true
		}
	}
	if kind == ExtractMethod {
		receiver := input.target.Recv.List[0].Names[0]
		if object, ok := input.typesInfo.Defs[receiver].(*types.Var); ok {
			objects[object] = true
		}
	}
	if len(objects) == 0 {
		return rejectSelectedRecoverCalls(selected, owner)
	}
	var unsafe string
	ast.Inspect(input.target.Body, func(node ast.Node) bool {
		if unsafe != "" || node == nil {
			return false
		}
		unsafe = extractionAliasHazard(input, node, objects)
		return unsafe == ""
	})
	if unsafe != "" {
		return fmt.Errorf("cannot extract %s: %s may change aliasing or control-flow behavior", owner, unsafe)
	}
	return rejectSelectedRecoverCalls(selected, owner)
}

func extractionAliasHazard(input extractionInput, node ast.Node, objects map[*types.Var]bool) string {
	if node == nil {
		return ""
	}
	switch item := node.(type) {
	case *ast.LabeledStmt:
		return "labels in the owner"
	case *ast.BranchStmt:
		if item.Tok == token.GOTO {
			return "goto control flow in the owner"
		}
	case *ast.DeferStmt, *ast.GoStmt:
		return "defer or go statements in the owner"
	case *ast.UnaryExpr:
		if item.Op == token.AND && extractionExprUsesObject(input.typesInfo, item.X, objects) {
			return "address-taken values in the owner"
		}
	case *ast.SliceExpr:
		if extractionArraySliceAliasesObject(input.typesInfo, item, objects) {
			return "array slices that alias copied variables in the owner"
		}
	case *ast.FuncLit:
		if extractionClosureCaptures(input.typesInfo, item, objects) {
			return "closure-captured values in the owner"
		}
	case *ast.CallExpr:
		if extractionCallCopiesValueReceiver(input, item, objects) {
			return "implicit pointer-receiver calls on copied values"
		}
	case *ast.SelectorExpr:
		if extractionMethodValueCapturesOwner(input, item, objects) {
			return "method values that capture a copied variable"
		}
	}
	return ""
}

func extractionArraySliceAliasesObject(info *types.Info, slice *ast.SliceExpr, objects map[*types.Var]bool) bool {
	if !extractionExprUsesObject(info, slice.X, objects) {
		return false
	}
	typ := types.Unalias(info.TypeOf(slice.X))
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	_, isArray := typ.Underlying().(*types.Array)
	return isArray
}

func extractionMethodValueCapturesOwner(input extractionInput, selector *ast.SelectorExpr, objects map[*types.Var]bool) bool {
	if !extractionUsesImplicitPointerReceiver(input.typesInfo, selector) || !extractionExprUsesObject(input.typesInfo, selector.X, objects) {
		return false
	}
	return !extractionSelectorIsImmediateCall(input.target.Body, selector)
}

func extractionSelectorIsImmediateCall(body *ast.BlockStmt, selector *ast.SelectorExpr) bool {
	immediate := false
	ast.Inspect(body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && call.Fun == selector {
			immediate = true
			return false
		}
		return !immediate
	})
	return immediate
}

func extractionClosureCaptures(info *types.Info, literal *ast.FuncLit, objects map[*types.Var]bool) bool {
	captured := false
	ast.Inspect(literal.Body, func(node ast.Node) bool {
		if captured || node == nil {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if ok {
			object, _ := info.Uses[ident].(*types.Var)
			captured = captured || objects[object]
		}
		return !captured
	})
	return captured
}

func extractionCallCopiesValueReceiver(input extractionInput, call *ast.CallExpr, objects map[*types.Var]bool) bool {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || !extractionExprUsesObject(input.typesInfo, selector.X, objects) {
		return false
	}
	return extractionUsesImplicitPointerReceiver(input.typesInfo, selector)
}

func extractionUsesImplicitPointerReceiver(info *types.Info, selector *ast.SelectorExpr) bool {
	selection := info.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal {
		return false
	}
	signature, ok := selection.Obj().Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	_, pointerReceiver := types.Unalias(signature.Recv().Type()).(*types.Pointer)
	_, pointerValue := types.Unalias(selection.Recv()).(*types.Pointer)
	return pointerReceiver && !pointerValue
}

func extractionExprUsesObject(info *types.Info, expr ast.Expr, objects map[*types.Var]bool) bool {
	used := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if used || node == nil {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if ok {
			object, _ := info.Uses[ident].(*types.Var)
			used = used || objects[object]
		}
		return !used
	})
	return used
}

func validateTypedExtractedScope(input extractionInput, selected []ast.Stmt, flow extractionFlow, owner string) error {
	allowed := make(map[string]bool)
	for _, name := range flow.parameters.names {
		allowed[name] = true
	}
	for _, name := range flow.bindings {
		allowed[name] = true
	}
	for _, output := range flow.outputs {
		allowed[output.name] = true
	}
	for _, stmt := range selected {
		if extractionStatementHasUnsupportedScope(input, stmt, allowed) {
			return fmt.Errorf("cannot extract %s: selected region contains a nested or composite literal, or a binding unavailable to the package helper", owner)
		}
	}
	return nil
}

func extractionStatementHasUnsupportedScope(input extractionInput, stmt ast.Stmt, allowed map[string]bool) bool {
	unsupported := false
	ast.Inspect(stmt, func(node ast.Node) bool {
		if unsupported || node == nil {
			return false
		}
		unsupported = extractionNodeHasUnsupportedScope(input, node, stmt, allowed)
		return !unsupported
	})
	return unsupported
}

func extractionNodeHasUnsupportedScope(input extractionInput, node ast.Node, stmt ast.Stmt, allowed map[string]bool) bool {
	switch item := node.(type) {
	case *ast.FuncLit, *ast.CompositeLit:
		return true
	case *ast.Ident:
		if item.Name == "_" || parentSelectorIdent(stmt, item) {
			return false
		}
		object := input.typesInfo.Uses[item]
		if object == nil {
			object = input.typesInfo.Defs[item]
		}
		return extractionObjectUnavailableInHelper(object, item.Name, allowed)
	}
	return false
}

func extractionObjectUnavailableInHelper(object types.Object, name string, allowed map[string]bool) bool {
	switch item := object.(type) {
	case *types.Var:
		return !extractionObjectPackageScoped(item) && item.Parent() != types.Universe && !allowed[name]
	case *types.Const:
		return item.Parent() != types.Universe && (item.Pkg() == nil || item.Parent() != item.Pkg().Scope())
	case *types.TypeName:
		return item.Parent() != types.Universe && (item.Pkg() == nil || item.Parent() != item.Pkg().Scope())
	}
	return false
}

func visitExtractFlowStatement(info *types.Info, stmt ast.Stmt, visit func(*types.Var, bool)) error {
	switch statement := stmt.(type) {
	case *ast.IfStmt:
		if err := validateExtractionIfElse(info, statement); err != nil {
			return err
		}
		visitExtractionReads(info, statement, visit)
		return nil
	case *ast.ForStmt:
		if err := validateExtractionForLoop(info, statement); err != nil {
			return err
		}
		initialization := statement.Init.(*ast.AssignStmt)
		index := info.Defs[initialization.Lhs[0].(*ast.Ident)].(*types.Var)
		visit(index, false)
		visitExtractionReads(info, statement, visit)
		return nil
	case *ast.RangeStmt:
		if err := validateExtractionRangeLoop(info, statement); err != nil {
			return err
		}
		for _, expression := range []ast.Expr{statement.Key, statement.Value} {
			if identifier, ok := expression.(*ast.Ident); ok && identifier.Name != "_" {
				visit(info.Defs[identifier].(*types.Var), false)
			}
		}
		visitExtractionReads(info, statement, visit)
		return nil
	case *ast.AssignStmt:
		return visitExtractAssignmentFlow(info, statement, visit)
	case *ast.ExprStmt:
		visitExtractionReads(info, statement, visit)
		return nil
	default:
		return fmt.Errorf("cannot extract: selected statements must be calls, simple assignments, bounded if/else statements, or counted loops")
	}
}

func validateExtractionIfElse(info *types.Info, statement *ast.IfStmt) error {
	if statement.Init != nil {
		return fmt.Errorf("if/else extraction does not support an init statement")
	}
	elseBody, ok := statement.Else.(*ast.BlockStmt)
	if !ok {
		return fmt.Errorf("if/else extraction requires a complete else block")
	}
	if !pureTypedExtractionCondition(info, statement.Cond) {
		return fmt.Errorf("if/else extraction condition must be a pure expression over resolved scalar inputs and constants")
	}
	if err := validateExtractionCallBlock(info, statement.Body); err != nil {
		return fmt.Errorf("if branch: %w", err)
	}
	if err := validateExtractionCallBlock(info, elseBody); err != nil {
		return fmt.Errorf("else branch: %w", err)
	}
	var nonScalar string
	ast.Inspect(statement, func(node ast.Node) bool {
		if node == nil || nonScalar != "" {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		variable, ok := info.Uses[ident].(*types.Var)
		if !ok || variable.IsField() || extractionObjectPackageScoped(variable) {
			return true
		}
		if !extractionScalarType(variable.Type()) {
			nonScalar = ident.Name
		}
		return nonScalar == ""
	})
	if nonScalar != "" {
		return fmt.Errorf("if/else extraction cannot copy non-scalar owner input %q into the helper", nonScalar)
	}
	return nil
}

func pureTypedExtractionCondition(info *types.Info, expression ast.Expr) bool {
	if expression == nil || !extractionScalarType(info.TypeOf(expression)) {
		return false
	}
	switch node := expression.(type) {
	case *ast.Ident:
		switch object := info.Uses[node].(type) {
		case *types.Const:
			return extractionScalarType(object.Type()) && (object.Parent() == types.Universe || object.Pkg() != nil && object.Parent() == object.Pkg().Scope())
		case *types.Var:
			return !object.IsField() && !extractionObjectPackageScoped(object) && extractionScalarType(object.Type())
		default:
			return false
		}
	case *ast.BasicLit:
		return extractionScalarType(info.TypeOf(node))
	case *ast.ParenExpr:
		return pureTypedExtractionCondition(info, node.X)
	case *ast.UnaryExpr:
		switch node.Op {
		case token.NOT, token.ADD, token.SUB, token.XOR:
			return pureTypedExtractionCondition(info, node.X)
		default:
			return false
		}
	case *ast.BinaryExpr:
		switch node.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
			token.AND, token.OR, token.XOR, token.AND_NOT, token.SHL, token.SHR,
			token.LAND, token.LOR, token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
			return pureTypedExtractionCondition(info, node.X) && pureTypedExtractionCondition(info, node.Y)
		default:
			return false
		}
	default:
		return false
	}
}

func extractionScalarType(value types.Type) bool {
	if value == nil {
		return false
	}
	basic, ok := types.Unalias(value).Underlying().(*types.Basic)
	return ok && basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) != 0
}

func validateExtractionCallBlock(info *types.Info, block *ast.BlockStmt) error {
	if block == nil || len(block.List) == 0 {
		return fmt.Errorf("branch must contain a direct call statement")
	}
	for _, statement := range block.List {
		expression, ok := statement.(*ast.ExprStmt)
		if !ok {
			return fmt.Errorf("branch statements must be direct calls")
		}
		call, ok := expression.X.(*ast.CallExpr)
		if !ok || !resolvedDirectExtractionCall(info, call) {
			return fmt.Errorf("branch statements must be direct calls")
		}
	}
	return nil
}

func resolvedDirectExtractionCall(info *types.Info, call *ast.CallExpr) bool {
	switch function := call.Fun.(type) {
	case *ast.Ident:
		switch info.Uses[function].(type) {
		case *types.Func, *types.Builtin:
			return true
		default:
			return false
		}
	case *ast.SelectorExpr:
		if selection := info.Selections[function]; selection != nil {
			_, ok := selection.Obj().(*types.Func)
			return ok
		}
		if _, ok := info.Uses[function.Sel].(*types.Func); !ok {
			return false
		}
		qualifier, ok := function.X.(*ast.Ident)
		if !ok {
			return false
		}
		_, ok = info.Uses[qualifier].(*types.PkgName)
		return ok
	default:
		return false
	}
}

func assignmentHasMatchingTuple(info *types.Info, assignment *ast.AssignStmt) bool {
	if len(assignment.Rhs) != 1 || len(assignment.Lhs) < 2 {
		return false
	}
	tuple, ok := info.TypeOf(assignment.Rhs[0]).(*types.Tuple)
	return ok && tuple.Len() == len(assignment.Lhs)
}

func visitExtractionReads(info *types.Info, node ast.Node, visit func(*types.Var, bool)) {
	ast.Inspect(node, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		variable, ok := info.Uses[ident].(*types.Var)
		if ok && variable != nil && !variable.IsField() {
			visit(variable, true)
		}
		return true
	})
}

func extractionFlowBinding(bindings map[*types.Var]*extractionBinding, ordered *[]*extractionBinding, object *types.Var) *extractionBinding {
	if binding := bindings[object]; binding != nil {
		return binding
	}
	binding := &extractionBinding{object: object, name: object.Name()}
	bindings[object] = binding
	*ordered = append(*ordered, binding)
	return binding
}

func appendExtractionBinding(input extractionInput, flow *extractionFlow, object *types.Var, kind ExtractKind) error {
	if flow.parameters.positions == nil {
		flow.parameters.positions = make(map[string]token.Pos)
	}
	flow.parameters.positions[object.Name()] = object.Pos()
	if input.target.Recv != nil && input.target.Recv.List[0].Names[0].Name == object.Name() {
		flow.parameters.names = append(flow.parameters.names, object.Name())
		if kind == ExtractKindFunction {
			text, err := extractionTypeString(input, object.Type())
			if err != nil {
				return err
			}
			flow.parameters.declarations = append(flow.parameters.declarations, object.Name()+" "+text)
		}
		return nil
	}
	typeName, err := extractionTypeString(input, object.Type())
	if err != nil {
		return err
	}
	flow.parameters.names = append(flow.parameters.names, object.Name())
	flow.parameters.declarations = append(flow.parameters.declarations, object.Name()+" "+typeName)
	return nil
}

func extractionBindingDeclaration(input extractionInput, object *types.Var) (string, error) {
	typeName, err := extractionTypeString(input, object.Type())
	if err != nil {
		return "", err
	}
	return "var " + object.Name() + " " + typeName, nil
}

func extractionTypeString(input extractionInput, typ types.Type) (string, error) {
	if extractionTypeHasLocalName(typ) {
		return "", fmt.Errorf("local named types and type parameters cannot appear in a package helper signature")
	}
	aliases := make(map[string]string)
	for _, imported := range input.file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			return "", fmt.Errorf("decode import path: %w", err)
		}
		if imported.Name != nil {
			if imported.Name.Name != "." {
				aliases[path] = imported.Name.Name
			}
		}
	}
	currentPackage := input.typesInfo.Defs[input.target.Name].Pkg()
	text := types.TypeString(typ, func(pkg *types.Package) string {
		if pkg == currentPackage {
			return ""
		}
		if alias, ok := aliases[pkg.Path()]; ok {
			return alias
		}
		return pkg.Name()
	})
	return text, nil
}

func extractionObjectPackageScoped(object *types.Var) bool {
	return object.Pkg() != nil && object.Parent() == object.Pkg().Scope()
}

func extractObjectUsedAfter(input extractionInput, end token.Pos, object *types.Var) bool {
	for _, node := range extractNodesAfterSelection(input, end, object) {
		if extractionNodeReferencesObject(input.typesInfo, node, object) {
			return true
		}
	}
	return false
}

func extractObjectUsedOutside(input extractionInput, start, end token.Pos, object types.Object) bool {
	return extractionNodeReadsObject(input.typesInfo, input.target.Body, object, start, end)
}

func extractObjectReadAfter(input extractionInput, end token.Pos, object *types.Var) bool {
	for _, node := range extractNodesAfterSelection(input, end, object) {
		if extractionNodeReadsObject(input.typesInfo, node, object) {
			return true
		}
	}
	return false
}

type extractionBlockParent struct {
	block     *ast.BlockStmt
	statement ast.Stmt
}

func extractNodesAfterSelection(input extractionInput, end token.Pos, object types.Object) []ast.Node {
	var nodes []ast.Node
	selectionBlock := input.selectionBlock
	if selectionBlock == nil {
		selectionBlock = input.target.Body
	}
	for _, stmt := range selectionBlock.List {
		if stmt.Pos() > end {
			nodes = append(nodes, stmt)
		}
	}

	parents := extractBlockParents(input.target.Body, selectionBlock)
	if input.selectionSwitch != nil {
		parents = append(parents, extractBlockParents(input.target.Body, input.selectionSwitch.Body)...)
	}
	for _, parent := range parents {
		for index, stmt := range parent.block.List {
			if stmt != parent.statement {
				continue
			}
			for _, later := range parent.block.List[index+1:] {
				nodes = append(nodes, later)
			}
			break
		}
		if stmt, ok := parent.statement.(*ast.ForStmt); ok {
			if stmt.Cond != nil {
				nodes = append(nodes, stmt.Cond)
			}
			if stmt.Post != nil {
				nodes = append(nodes, stmt.Post)
			}
			if !extractionObjectDeclaredInBlock(object, stmt.Body) {
				nodes = append(nodes, stmt.Body)
			}
		}
		if stmt, ok := parent.statement.(*ast.RangeStmt); ok {
			if !extractionObjectDeclaredInBlock(object, stmt.Body) {
				nodes = append(nodes, stmt.Body)
			}
		}
	}
	return nodes
}

func extractionObjectDeclaredInBlock(object types.Object, block *ast.BlockStmt) bool {
	return object.Pos() >= block.Pos() && object.Pos() <= block.End()
}

func extractBlockParents(root, target *ast.BlockStmt) []extractionBlockParent {
	var parents []extractionBlockParent
	var find func(*ast.BlockStmt) bool
	var childBlocks func(ast.Stmt) []*ast.BlockStmt
	childBlocks = func(stmt ast.Stmt) []*ast.BlockStmt {
		switch item := stmt.(type) {
		case *ast.BlockStmt:
			return []*ast.BlockStmt{item}
		case *ast.IfStmt:
			blocks := []*ast.BlockStmt{item.Body}
			if block, ok := item.Else.(*ast.BlockStmt); ok {
				blocks = append(blocks, block)
			} else if item.Else != nil {
				blocks = append(blocks, childBlocks(item.Else)...)
			}
			return blocks
		case *ast.ForStmt:
			return []*ast.BlockStmt{item.Body}
		case *ast.RangeStmt:
			return []*ast.BlockStmt{item.Body}
		case *ast.SwitchStmt:
			return []*ast.BlockStmt{item.Body}
		case *ast.LabeledStmt:
			return childBlocks(item.Stmt)
		default:
			return nil
		}
	}
	find = func(block *ast.BlockStmt) bool {
		if block == target {
			return true
		}
		for _, stmt := range block.List {
			for _, nested := range childBlocks(stmt) {
				parents = append(parents, extractionBlockParent{block: block, statement: stmt})
				if find(nested) {
					return true
				}
				parents = parents[:len(parents)-1]
			}
		}
		return false
	}
	find(root)
	return parents
}

func extractionNodeReadsObject(info *types.Info, root ast.Node, object types.Object, excludedRange ...token.Pos) bool {
	if len(excludedRange) != 0 && len(excludedRange) != 2 {
		return true
	}
	excludeSelection := len(excludedRange) == 2 && excludedRange[0].IsValid() && excludedRange[1].IsValid()
	var parents []ast.Node
	read := false
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			parents = parents[:len(parents)-1]
			return !read
		}
		parents = append(parents, node)
		ident, ok := node.(*ast.Ident)
		if !ok || info.Uses[ident] != object {
			return true
		}
		if excludeSelection && ident.Pos() >= excludedRange[0] && ident.End() <= excludedRange[1] {
			return true
		}
		if extractionIdentifierIsPlainWrite(ident, parents) {
			return true
		}
		read = true
		return false
	})
	return read
}

func extractionIdentifierIsPlainWrite(ident *ast.Ident, parents []ast.Node) bool {
	for _, parent := range slices.Backward(parents) {
		switch ancestor := parent.(type) {
		case *ast.AssignStmt:
			if ancestor.Tok == token.ASSIGN || ancestor.Tok == token.DEFINE {
				for _, lhs := range ancestor.Lhs {
					if extractionSimpleAssignmentTarget(lhs, ident) {
						return true
					}
				}
			}
		case *ast.RangeStmt:
			if ancestor.Tok == token.ASSIGN && (extractionSimpleAssignmentTarget(ancestor.Key, ident) || extractionSimpleAssignmentTarget(ancestor.Value, ident)) {
				return true
			}
		}
	}
	return false
}

func extractionSimpleAssignmentTarget(target ast.Expr, ident *ast.Ident) bool {
	switch expression := target.(type) {
	case *ast.Ident:
		return expression == ident
	case *ast.ParenExpr:
		return extractionSimpleAssignmentTarget(expression.X, ident)
	default:
		return false
	}
}

func extractionNodeReferencesObject(info *types.Info, root ast.Node, object types.Object) bool {
	referenced := false
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil || referenced {
			return !referenced
		}
		ident, ok := node.(*ast.Ident)
		if ok && info.Uses[ident] == object {
			referenced = true
			return false
		}
		return true
	})
	return referenced
}

func extractionLocalType(object *types.TypeName) bool {
	return object != nil && object.Pkg() != nil && object.Parent() != object.Pkg().Scope()
}

func extractionTypeHasLocalName(typ types.Type) bool {
	switch item := typ.(type) {
	case *types.Named:
		return extractionLocalType(item.Obj())
	case *types.Alias:
		return extractionTypeHasLocalName(types.Unalias(item))
	case *types.Pointer:
		return extractionTypeHasLocalName(item.Elem())
	case *types.Slice:
		return extractionTypeHasLocalName(item.Elem())
	case *types.Array:
		return extractionTypeHasLocalName(item.Elem())
	case *types.Map:
		return extractionTypeHasLocalName(item.Key()) || extractionTypeHasLocalName(item.Elem())
	case *types.Chan:
		return extractionTypeHasLocalName(item.Elem())
	case *types.Signature:
		if item.Recv() != nil && extractionTypeHasLocalName(item.Recv().Type()) {
			return true
		}
		return extractionTupleHasLocalName(item.Params()) || extractionTupleHasLocalName(item.Results())
	case *types.Tuple:
		return extractionTupleHasLocalName(item)
	case *types.Struct:
		for field := range item.Fields() {
			if extractionTypeHasLocalName(field.Type()) {
				return true
			}
		}
	case *types.Interface:
		for method := range item.Methods() {
			if extractionTypeHasLocalName(method.Type()) {
				return true
			}
		}
	case *types.TypeParam:
		return true
	}
	return false
}

func extractionTupleHasLocalName(tuple *types.Tuple) bool {
	if tuple == nil {
		return false
	}
	for v := range tuple.Variables() {
		if extractionTypeHasLocalName(v.Type()) {
			return true
		}
	}
	return false
}

func visitExtractAssignmentFlow(info *types.Info, assignment *ast.AssignStmt, visit func(*types.Var, bool)) error {
	if assignment.Tok != token.DEFINE && assignment.Tok != token.ASSIGN {
		return fmt.Errorf("cannot extract: only := and = assignments are supported")
	}
	if len(assignment.Lhs) == 0 || len(assignment.Rhs) != len(assignment.Lhs) && !assignmentHasMatchingTuple(info, assignment) {
		return fmt.Errorf("cannot extract: assignments must have matching simple targets and values")
	}
	for _, lhs := range assignment.Lhs {
		if _, ok := lhs.(*ast.Ident); !ok {
			return fmt.Errorf("cannot extract: assignment targets must be simple identifiers")
		}
	}
	for _, rhs := range assignment.Rhs {
		visitExtractionReads(info, rhs, visit)
	}
	for _, lhs := range assignment.Lhs {
		ident := lhs.(*ast.Ident)
		object := info.Defs[ident]
		if object == nil {
			object = info.Uses[ident]
		}
		variable, ok := object.(*types.Var)
		if !ok || variable == nil {
			if ident.Name == "_" {
				continue
			}
			return fmt.Errorf("cannot extract: assignment target %q has no resolved variable identity", ident.Name)
		}
		visit(variable, false)
	}
	return nil
}

func validateExtractionForLoop(info *types.Info, statement *ast.ForStmt) error {
	index, err := extractionForLoopIndex(info, statement)
	if err != nil {
		return err
	}
	if err := validateExtractionLoopBlock(info, statement.Body); err != nil {
		return err
	}
	return validateExtractionLoopInputs(info, statement, "counted-loop", index)
}

func extractionForLoopIndex(info *types.Info, statement *ast.ForStmt) (*types.Var, error) {
	initialization, ok := statement.Init.(*ast.AssignStmt)
	if !ok || initialization.Tok != token.DEFINE || len(initialization.Lhs) != 1 || len(initialization.Rhs) != 1 {
		return nil, fmt.Errorf("counted-loop extraction requires one short-declared integer index")
	}
	indexName, ok := initialization.Lhs[0].(*ast.Ident)
	if !ok || indexName.Name == "_" {
		return nil, fmt.Errorf("counted-loop extraction requires a named integer index")
	}
	index, ok := info.Defs[indexName].(*types.Var)
	if !ok {
		return nil, fmt.Errorf("counted-loop extraction requires a resolved integer index")
	}
	basic, ok := types.Unalias(index.Type()).Underlying().(*types.Basic)
	if !ok || basic.Info()&types.IsInteger == 0 || !pureTypedExtractionCondition(info, initialization.Rhs[0]) {
		return nil, fmt.Errorf("counted-loop extraction requires an integer index and pure scalar initializer")
	}
	condition, ok := statement.Cond.(*ast.BinaryExpr)
	if !ok || !pureTypedExtractionCondition(info, condition) ||
		!extractionNodeReferencesObject(info, condition.X, index) ||
		extractionNodeReferencesObject(info, condition.Y, index) {
		return nil, fmt.Errorf("counted-loop extraction requires a pure comparison with the index and a fixed scalar bound")
	}
	if err := validateExtractionLoopDirection(info, statement, index, condition.Op); err != nil {
		return nil, err
	}
	return index, nil
}

func validateExtractionLoopDirection(info *types.Info, statement *ast.ForStmt, index *types.Var, comparison token.Token) error {
	post, ok := statement.Post.(*ast.IncDecStmt)
	if !ok || post.Tok != token.INC && post.Tok != token.DEC {
		return fmt.Errorf("counted-loop extraction requires a unit index increment or decrement")
	}
	postIndex, ok := post.X.(*ast.Ident)
	if !ok || info.Uses[postIndex] != index {
		return fmt.Errorf("counted-loop post statement must update its declared index")
	}
	switch comparison {
	case token.LSS, token.LEQ:
		if post.Tok != token.INC {
			return fmt.Errorf("counted-loop direction must match its comparison")
		}
	case token.GTR, token.GEQ:
		if post.Tok != token.DEC {
			return fmt.Errorf("counted-loop direction must match its comparison")
		}
	default:
		return fmt.Errorf("counted-loop extraction requires an ordered comparison")
	}
	return nil
}

func validateExtractionLoopBlock(info *types.Info, block *ast.BlockStmt) error {
	for _, bodyStatement := range block.List {
		switch item := bodyStatement.(type) {
		case *ast.ExprStmt:
			call, ok := item.X.(*ast.CallExpr)
			if !ok || !resolvedDirectExtractionCall(info, call) {
				return fmt.Errorf("loop body statements must be direct calls or bounded if/else statements")
			}
		case *ast.IfStmt:
			if err := validateExtractionLoopIf(info, item); err != nil {
				return err
			}
		case *ast.BranchStmt:
			if item.Label != nil || item.Tok != token.BREAK && item.Tok != token.CONTINUE {
				return fmt.Errorf("loop body supports only unlabelled break and continue")
			}
		default:
			return fmt.Errorf("loop body statements must be direct calls, bounded if/else statements, or unlabelled break/continue")
		}
	}
	return nil
}

type extractionChannelRangeBinding struct {
	identifier *ast.Ident
	variable   *types.Var
}

func extractionReceiveChannelRangeBinding(info *types.Info, expression ast.Expr) (extractionChannelRangeBinding, bool) {
	identifier, ok := unparenExpr(expression).(*ast.Ident)
	if !ok {
		return extractionChannelRangeBinding{}, false
	}
	variable, ok := info.Uses[identifier].(*types.Var)
	if !ok || variable.IsField() || extractionObjectPackageScoped(variable) {
		return extractionChannelRangeBinding{}, false
	}
	channel, ok := types.Unalias(variable.Type()).Underlying().(*types.Chan)
	if !ok || channel.Dir() == types.SendOnly {
		return extractionChannelRangeBinding{}, false
	}
	return extractionChannelRangeBinding{identifier: identifier, variable: variable}, true
}

func validateExtractionLoopInputs(info *types.Info, statement ast.Node, loopKind string, locals ...*types.Var) error {
	localObjects := make(map[*types.Var]struct{}, len(locals))
	for _, local := range locals {
		localObjects[local] = struct{}{}
	}
	var channelBinding extractionChannelRangeBinding
	var mapIdentifier *ast.Ident
	var mapVariable *types.Var
	var sliceIdentifier *ast.Ident
	var sliceVariable *types.Var
	if rangeStatement, ok := statement.(*ast.RangeStmt); ok {
		channelBinding, _ = extractionReceiveChannelRangeBinding(info, rangeStatement.X)
		if identifier, variable, err := extractionMapRangeBinding(info, rangeStatement.X); err == nil {
			mapIdentifier, mapVariable = identifier, variable
		}
		if identifier, variable, err := extractionSliceRangeBinding(info, rangeStatement.X); err == nil {
			sliceIdentifier, sliceVariable = identifier, variable
		}
	}
	var nonScalar string
	ast.Inspect(statement, func(node ast.Node) bool {
		if node == nil || nonScalar != "" {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		variable, ok := info.Uses[ident].(*types.Var)
		if !ok || variable.IsField() || extractionObjectPackageScoped(variable) {
			return true
		}
		if _, isLocal := localObjects[variable]; isLocal {
			return true
		}
		isRangeOperand := (ident == channelBinding.identifier && variable == channelBinding.variable) ||
			(ident == mapIdentifier && variable == mapVariable) ||
			(ident == sliceIdentifier && variable == sliceVariable)
		if !extractionScalarType(variable.Type()) && !isRangeOperand {
			nonScalar = ident.Name
		}
		return nonScalar == ""
	})
	if nonScalar != "" {
		return fmt.Errorf("%s extraction cannot copy non-scalar owner input %q into the helper", loopKind, nonScalar)
	}
	return nil
}

func validateExtractionLoopIf(info *types.Info, statement *ast.IfStmt) error {
	if statement.Init != nil || !pureTypedExtractionCondition(info, statement.Cond) {
		return fmt.Errorf("loop body conditions must be pure scalar expressions without init statements")
	}
	if err := validateExtractionLoopBlock(info, statement.Body); err != nil {
		return err
	}
	switch alternative := statement.Else.(type) {
	case nil:
		return nil
	case *ast.BlockStmt:
		return validateExtractionLoopBlock(info, alternative)
	case *ast.IfStmt:
		return validateExtractionLoopIf(info, alternative)
	default:
		return fmt.Errorf("loop body else branch must be a block or if/else statement")
	}
}

func validateExtractionRangeLoop(info *types.Info, statement *ast.RangeStmt) error {
	if statement.Tok != token.DEFINE {
		return fmt.Errorf("range-loop extraction requires short-declared iteration bindings")
	}
	bound := false
	for _, expression := range []ast.Expr{statement.Key, statement.Value} {
		if expression == nil {
			continue
		}
		identifier, ok := expression.(*ast.Ident)
		if !ok {
			return fmt.Errorf("range-loop extraction requires identifier iteration bindings")
		}
		if identifier.Name == "_" {
			continue
		}
		if _, ok := info.Defs[identifier].(*types.Var); !ok {
			return fmt.Errorf("range-loop extraction requires fresh iteration bindings")
		}
		bound = true
	}
	if !bound {
		return fmt.Errorf("range-loop extraction requires at least one named iteration binding")
	}
	if err := validateExtractionRangeOperand(info, statement.X); err != nil {
		return err
	}
	locals := make([]*types.Var, 0, 2)
	for _, expression := range []ast.Expr{statement.Key, statement.Value} {
		identifier, ok := expression.(*ast.Ident)
		if !ok || identifier.Name == "_" {
			continue
		}
		locals = append(locals, info.Defs[identifier].(*types.Var))
	}
	if err := validateExtractionLoopInputs(info, statement, "range-loop", locals...); err != nil {
		return err
	}
	return validateExtractionLoopBlock(info, statement.Body)
}

func validateExtractionRangeOperand(info *types.Info, expression ast.Expr) error {
	operandType := info.TypeOf(expression)
	if operandType == nil {
		return fmt.Errorf("range-loop extraction requires a resolved string, integer, local slice/map identifier, receive-capable channel, or scalar-slice factory operand")
	}
	underlying := types.Unalias(operandType).Underlying()
	switch typed := underlying.(type) {
	case *types.Basic:
		if typed.Info()&(types.IsInteger|types.IsString) == 0 {
			return fmt.Errorf("range-loop extraction requires a supported scalar operand")
		}
		if !pureTypedExtractionCondition(info, expression) {
			return fmt.Errorf("range-loop extraction requires a pure scalar operand")
		}
	case *types.Chan:
		if _, ok := extractionReceiveChannelRangeBinding(info, expression); !ok {
			return fmt.Errorf("range-loop channel operand must be a local receive-capable channel identifier")
		}
	case *types.Map:
		if _, _, err := extractionMapRangeBinding(info, expression); err != nil {
			return err
		}
	case *types.Slice:
		switch unparenExpr(expression).(type) {
		case *ast.Ident, *ast.SliceExpr:
			if _, _, err := extractionSliceRangeBinding(info, expression); err != nil {
				return err
			}
		case *ast.CallExpr:
			if err := validateExtractionSliceFactoryRange(info, expression); err != nil {
				return err
			}
		default:
			return fmt.Errorf("range-loop slice operand must be a local or parameter slice identifier, a two-index slice expression rooted in one, or a direct same-package factory call")
		}
	default:
		if err := validateExtractionSliceFactoryRange(info, expression); err != nil {
			return err
		}
	}
	return nil
}

func validateExtractionSliceFactoryRange(info *types.Info, expression ast.Expr) error {
	call, ok := unparenExpr(expression).(*ast.CallExpr)
	if !ok {
		return fmt.Errorf("range-loop extraction supports only string/integer operands, local slice/map identifiers, local receive-capable channel identifiers, or direct scalar-slice factory calls")
	}
	identifier, ok := call.Fun.(*ast.Ident)
	if !ok {
		return fmt.Errorf("range-loop slice operand must call a direct same-package function")
	}
	function, ok := info.Uses[identifier].(*types.Func)
	if !ok {
		return fmt.Errorf("range-loop slice operand must call a direct same-package function")
	}
	declaredHere := false
	for _, object := range info.Defs {
		if object == function {
			declaredHere = true
			break
		}
	}
	if !declaredHere {
		return fmt.Errorf("range-loop slice factory must be declared in the current package")
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() != nil {
		return fmt.Errorf("range-loop slice factory must be a package function without a receiver")
	}
	if parameters := signature.TypeParams(); parameters != nil && parameters.Len() != 0 {
		return fmt.Errorf("range-loop slice factory cannot be generic")
	}
	if signature.Variadic() || call.Ellipsis.IsValid() || signature.Params().Len() != len(call.Args) {
		return fmt.Errorf("range-loop slice factory must use a non-variadic call with matching arguments")
	}
	results := signature.Results()
	if results.Len() != 1 {
		return fmt.Errorf("range-loop slice factory must return exactly one slice")
	}
	resultType := types.Unalias(results.At(0).Type())
	slice, ok := resultType.Underlying().(*types.Slice)
	if !ok || !extractionScalarType(slice.Elem()) {
		return fmt.Errorf("range-loop slice factory must return a slice of scalar elements")
	}
	for _, argument := range call.Args {
		if !pureTypedExtractionCondition(info, argument) {
			return fmt.Errorf("range-loop slice factory arguments must be pure scalar expressions")
		}
	}
	return nil
}

func extractionMapRangeBinding(info *types.Info, expression ast.Expr) (*ast.Ident, *types.Var, error) {
	identifier, ok := unparenExpr(expression).(*ast.Ident)
	if !ok {
		return nil, nil, fmt.Errorf("range-loop map operand must be a local or parameter map identifier")
	}
	variable, ok := info.Uses[identifier].(*types.Var)
	if !ok || variable.IsField() || extractionObjectPackageScoped(variable) {
		return nil, nil, fmt.Errorf("range-loop map operand must be a local or parameter map identifier")
	}
	mapType, ok := types.Unalias(variable.Type()).Underlying().(*types.Map)
	if !ok {
		return nil, nil, fmt.Errorf("range-loop operand is not a map variable")
	}
	if !extractionScalarType(mapType.Key()) || !extractionScalarType(mapType.Elem()) {
		return nil, nil, fmt.Errorf("range-loop map key and element types must be scalar")
	}
	if extractionTypeHasLocalName(variable.Type()) {
		return nil, nil, fmt.Errorf("range-loop map type must be nameable from a package helper")
	}
	return identifier, variable, nil
}

func extractionSliceRangeBinding(info *types.Info, expression ast.Expr) (*ast.Ident, *types.Var, error) {
	var identifier *ast.Ident
	switch operand := unparenExpr(expression).(type) {
	case *ast.Ident:
		identifier = operand
	case *ast.SliceExpr:
		var ok bool
		identifier, ok = unparenExpr(operand.X).(*ast.Ident)
		if !ok {
			return nil, nil, fmt.Errorf("range-loop slice expression must be rooted in a local or parameter slice identifier")
		}
		for _, bound := range []ast.Expr{operand.Low, operand.High, operand.Max} {
			if bound == nil {
				continue
			}
			if !pureTypedExtractionCondition(info, bound) {
				return nil, nil, fmt.Errorf("range-loop slice bounds must be pure compiler-resolved integer expressions")
			}
			boundType := types.Unalias(info.TypeOf(bound))
			basic, ok := boundType.Underlying().(*types.Basic)
			if !ok || basic.Info()&types.IsInteger == 0 {
				return nil, nil, fmt.Errorf("range-loop slice bounds must be pure compiler-resolved integer expressions")
			}
		}
	default:
		return nil, nil, fmt.Errorf("range-loop slice operand must be a local or parameter slice identifier or a slice expression rooted in one")
	}
	variable, ok := info.Uses[identifier].(*types.Var)
	if !ok || variable.IsField() || extractionObjectPackageScoped(variable) {
		return nil, nil, fmt.Errorf("range-loop slice operand must be a local or parameter slice identifier")
	}
	sliceType, ok := types.Unalias(variable.Type()).Underlying().(*types.Slice)
	if !ok {
		return nil, nil, fmt.Errorf("range-loop operand is not a slice variable")
	}
	if !extractionScalarType(sliceType.Elem()) {
		return nil, nil, fmt.Errorf("range-loop slice element type must be scalar")
	}
	if extractionTypeHasLocalName(variable.Type()) {
		return nil, nil, fmt.Errorf("range-loop slice type must be nameable from a package helper")
	}
	return identifier, variable, nil
}
