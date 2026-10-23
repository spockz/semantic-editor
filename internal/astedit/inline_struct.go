// WHY: Shares typed immediate-use proofs before inline-value moves scalar or struct initializers.

package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/symbol"
)

func planInlineImmediateReturnVariable(ctx context.Context, cleanPath, function, name string, content []byte, ownedBuffer *sourcebuf.Buffer, standalone bool) (inlineVariableEdit, error) {
	overlay := sourcebuf.Overlay(ctx, ".go")
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	overlay[cleanPath] = bytes.Clone(content)
	loaded, err := loadExtractionPackages(ctx, filepath.Dir(cleanPath), overlay, extractionLoadOptions{typeInfo: true})
	if err != nil {
		return inlineVariableEdit{}, fmt.Errorf("resolve inline input: %w", err)
	}
	syntax, err := selectVariableTypeSyntax(loaded, cleanPath)
	if err != nil {
		return inlineVariableEdit{}, err
	}
	receiver, ownerName, err := symbol.ParseIdentifier(function)
	if err != nil {
		return inlineVariableEdit{}, fmt.Errorf("parse inline owner: %w", err)
	}
	owner, err := findInlineFunction(syntax.syntax, receiver, ownerName)
	if err != nil {
		return inlineVariableEdit{}, err
	}
	target, err := findInlineTarget(owner, name)
	if err != nil {
		return inlineVariableEdit{}, err
	}
	use, err := findInlineImmediateValueUse(owner, target, syntax.info)
	if err != nil {
		return inlineVariableEdit{}, fmt.Errorf("cannot inline %q: %w", name, err)
	}
	var operands []inlineComputedOperand
	if use.callArgument || use.assignment || use.varInitializer || use.ifCondition || use.switchTag {
		operands, err = inlineComputedOperands(target.value, owner, syntax.info, syntax.fset)
	} else {
		operands, err = inlineReturnInitializerOperands(target, owner, syntax.info, syntax.fset)
	}
	if err != nil {
		return inlineVariableEdit{}, fmt.Errorf("cannot inline %q: %w", name, err)
	}
	immediateStatement := target.block.List[target.index+1]
	if commentErr := rejectInlineRegionComments(content, syntax.fset, syntax.syntax, target.stmt, immediateStatement); commentErr != nil {
		return inlineVariableEdit{}, fmt.Errorf("cannot inline %q: %w", name, commentErr)
	}
	removedStart, removedEnd, err := standaloneStatementRange(content, syntax.fset, target.stmt, syntax.syntax.Comments)
	if err != nil {
		return inlineVariableEdit{}, fmt.Errorf("cannot inline %q: %w", name, err)
	}
	candidate, err := inlineSource(content, syntax.fset, syntax.syntax, target, []token.Pos{use.reference.Pos()}, name, true)
	if err != nil {
		return inlineVariableEdit{}, err
	}
	if err := validateInlineImmediateCandidate(ctx, cleanPath, function, name, candidate, syntax, target, use, operands, removedEnd-removedStart); err != nil {
		return inlineVariableEdit{}, err
	}
	return inlineVariableEdit{path: cleanPath, name: name, before: content, after: candidate, owned: ownedBuffer, standalone: standalone}, nil
}

func validateInlineImmediateCandidate(ctx context.Context, cleanPath, function, name string, candidate []byte, syntax variableTypeSyntax, target inlineTarget, use inlineImmediateValueUse, operands []inlineComputedOperand, removedBytes int) error {
	candidateContext, err := loadVariableTypeContext(ctx, cleanPath, function, candidate)
	if err != nil {
		return fmt.Errorf("resolve inline candidate: %w", err)
	}
	candidateStatement := candidateContext.owner.Body.List[target.index]
	replacement, err := inlineImmediateValueExpression(candidateStatement, use)
	if err != nil {
		return fmt.Errorf("cannot inline %q: %w", name, err)
	}
	originalType := syntax.info.TypeOf(target.value)
	replacementType := candidateContext.info.TypeOf(replacement)
	if !types.Identical(originalType, replacementType) && extractionTypeIdentity(originalType) != extractionTypeIdentity(replacementType) {
		if !use.ifCondition || !types.Identical(originalType, types.Typ[types.Bool]) || !types.Identical(replacementType, types.Typ[types.UntypedBool]) {
			return fmt.Errorf("cannot inline %q: replacement changes the resolved initializer type from %q to %q", name, extractionTypeIdentity(originalType), extractionTypeIdentity(replacementType))
		}
	}
	if use.callArgument {
		call, ok := inlineImmediateValueCall(candidateStatement)
		if !ok {
			return fmt.Errorf("cannot inline %q: candidate immediate call is unresolved", name)
		}
		functionIdentifier, ok := call.Fun.(*ast.Ident)
		if !ok || !sameInlineCallTarget(use.callTarget, candidateContext.info.Uses[functionIdentifier]) {
			return fmt.Errorf("cannot inline %q: candidate changes the resolved call target", name)
		}
	}
	if use.assignment {
		assignment := candidateStatement.(*ast.AssignStmt)
		targetIdentifier := assignment.Lhs[0].(*ast.Ident)
		var beforeTarget, afterTarget types.Object
		afterOffsetDelta := 0
		if use.shortDeclaration {
			originalAssignment := target.block.List[target.index+1].(*ast.AssignStmt)
			originalIdentifier := originalAssignment.Lhs[0].(*ast.Ident)
			beforeTarget = syntax.info.Defs[originalIdentifier]
			afterTarget = candidateContext.info.Defs[targetIdentifier]
			afterOffsetDelta = -removedBytes
		} else {
			beforeTarget = use.assignmentTarget
			afterTarget = candidateContext.info.Uses[targetIdentifier]
		}
		if !sameInlineVariableTarget(beforeTarget, afterTarget, syntax.fset, candidateContext.fset, afterOffsetDelta) {
			return fmt.Errorf("cannot inline %q: candidate changes the resolved assignment target", name)
		}
	}
	if use.varInitializer {
		spec, ok := inlineSingleVarInitializer(candidateStatement)
		if !ok || !sameInlineVariableTarget(use.declarationTarget, candidateContext.info.Defs[spec.Names[0]], syntax.fset, candidateContext.fset, -removedBytes) {
			return fmt.Errorf("cannot inline %q: candidate changes the var initializer destination binding or type", name)
		}
	}
	if operands != nil {
		if err := validateInlineComputedCandidate(operands, replacement, candidateContext); err != nil {
			return fmt.Errorf("cannot inline %q: %w", name, err)
		}
	}
	return nil
}

type inlineImmediateValueUse struct {
	reference         *ast.Ident
	callArgument      bool
	assignment        bool
	shortDeclaration  bool
	varInitializer    bool
	ifCondition       bool
	switchTag         bool
	callTarget        types.Object
	assignmentTarget  types.Object
	declarationTarget types.Object
}

func sameInlineCallTarget(before, after types.Object) bool {
	switch before := before.(type) {
	case *types.Func:
		after, ok := after.(*types.Func)
		return ok && before.Pkg() != nil && after.Pkg() != nil && before.Pkg().Path() == after.Pkg().Path() && before.Name() == after.Name() && extractionTypeIdentity(before.Type()) == extractionTypeIdentity(after.Type())
	case *types.Builtin:
		after, ok := after.(*types.Builtin)
		return ok && before.Name() == after.Name()
	default:
		return false
	}
}

func sameInlineVariableTarget(before, after types.Object, beforeFset, afterFset *token.FileSet, afterOffsetDelta int) bool {
	beforeVariable, beforeOK := before.(*types.Var)
	afterVariable, afterOK := after.(*types.Var)
	if !beforeOK || !afterOK || beforeVariable.Name() != afterVariable.Name() || extractionTypeIdentity(beforeVariable.Type()) != extractionTypeIdentity(afterVariable.Type()) {
		return false
	}
	beforePackage := beforeVariable.Pkg()
	afterPackage := afterVariable.Pkg()
	if beforePackage == nil || afterPackage == nil || beforePackage.Path() != afterPackage.Path() {
		return false
	}
	beforePackageScope := beforeVariable.Parent() == beforePackage.Scope()
	afterPackageScope := afterVariable.Parent() == afterPackage.Scope()
	if beforePackageScope || afterPackageScope {
		return beforePackageScope && afterPackageScope
	}
	beforePosition := beforeFset.PositionFor(beforeVariable.Pos(), false)
	afterPosition := afterFset.PositionFor(afterVariable.Pos(), false)
	return beforePosition.IsValid() && afterPosition.IsValid() && beforePosition.Filename == afterPosition.Filename && beforePosition.Offset+afterOffsetDelta == afterPosition.Offset
}

func inlineStatementUsesBinding(statement ast.Stmt, info *types.Info, binding types.Object) bool {
	usesBinding := false
	ast.Inspect(statement, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && info.Uses[identifier] == binding {
			usesBinding = true
			return false
		}
		return !usesBinding
	})
	return usesBinding
}

func findInlineImmediateValueUse(owner *ast.FuncDecl, target inlineTarget, info *types.Info) (inlineImmediateValueUse, error) {
	assignment, ok := target.stmt.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || target.block != owner.Body || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return inlineImmediateValueUse{}, fmt.Errorf("initializer requires one inferred := binding in the owner body")
	}
	declaration, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok {
		return inlineImmediateValueUse{}, fmt.Errorf("binding declaration is unresolved")
	}
	binding, ok := info.Defs[declaration].(*types.Var)
	if !ok {
		return inlineImmediateValueUse{}, fmt.Errorf("binding declaration is unresolved")
	}
	if target.index+1 >= len(owner.Body.List) {
		return inlineImmediateValueUse{}, fmt.Errorf("binding requires an immediate direct return, one-argument direct call, one-target assignment, single-value var initializer, or complete if condition")
	}
	statement := owner.Body.List[target.index+1]
	if _, isReturn := statement.(*ast.ReturnStmt); isReturn && (owner.Type.Results == nil || len(owner.Type.Results.List) != 1 || len(owner.Type.Results.List[0].Names) != 0) {
		return inlineImmediateValueUse{}, fmt.Errorf("inline owner must have one unnamed result")
	}
	use, err := inlineImmediateValueUseIdentifier(statement, info, binding)
	if err != nil {
		return inlineImmediateValueUse{}, err
	}
	if !use.callArgument && !use.assignment && !use.varInitializer && !use.ifCondition && !use.switchTag && (owner.Type.Results == nil || len(owner.Type.Results.List) != 1 || len(owner.Type.Results.List[0].Names) != 0) {
		return inlineImmediateValueUse{}, fmt.Errorf("inline owner must have one unnamed result")
	}
	if info.Uses[use.reference] != binding {
		return inlineImmediateValueUse{}, fmt.Errorf("immediate use does not resolve to the selected binding")
	}
	count := 0
	for _, object := range info.Uses {
		if object == binding {
			count++
		}
	}
	if count != 1 {
		return inlineImmediateValueUse{}, fmt.Errorf("binding must have exactly one resolved reference")
	}
	if !types.Identical(binding.Type(), info.TypeOf(target.value)) || !types.Identical(binding.Type(), info.TypeOf(use.reference)) {
		return inlineImmediateValueUse{}, fmt.Errorf("binding and initializer types differ")
	}
	return use, nil
}

func inlineImmediateValueUseIdentifier(statement ast.Stmt, info *types.Info, binding types.Object) (inlineImmediateValueUse, error) {
	switch statement := statement.(type) {
	case *ast.ReturnStmt:
		use, ok := singleDirectReturnUse(statement)
		if !ok || info.Uses[use] != binding {
			return inlineImmediateValueUse{}, fmt.Errorf("binding requires an immediate direct return")
		}
		return inlineImmediateValueUse{reference: use}, nil
	case *ast.ExprStmt:
		if !inlineStatementUsesBinding(statement, info, binding) {
			return inlineImmediateValueUse{}, fmt.Errorf("binding requires an immediate direct return, one-argument direct call, complete if condition, or complete expression switch tag")
		}
		call, ok := statement.X.(*ast.CallExpr)
		if !ok {
			return inlineImmediateValueCallUse(nil, info, binding, "binding requires an immediate direct call with one argument")
		}
		return inlineImmediateValueCallUse(call, info, binding, "binding requires an immediate direct call with one argument")
	case *ast.DeferStmt:
		if !inlineStatementUsesBinding(statement, info, binding) {
			return inlineImmediateValueUse{}, fmt.Errorf("binding requires an immediate direct return, one-argument direct call or defer call, complete if condition, or complete expression switch tag")
		}
		return inlineImmediateValueCallUse(statement.Call, info, binding, "binding requires an immediate direct defer call with one argument")
	case *ast.GoStmt:
		if !inlineStatementUsesBinding(statement, info, binding) {
			return inlineImmediateValueUse{}, fmt.Errorf("binding requires an immediate direct return, one-argument direct call, go call, complete if condition, or complete expression switch tag")
		}
		return inlineImmediateValueCallUse(statement.Call, info, binding, "binding requires one argument to an immediate direct go call")
	case *ast.AssignStmt:
		return inlineImmediateValueUseAssignment(statement, info, binding)
	case *ast.DeclStmt:
		spec, ok := inlineSingleVarInitializer(statement)
		if !ok {
			return inlineImmediateValueUse{}, fmt.Errorf("binding requires one immediate ungrouped var declaration with one name and one initializer")
		}
		use, ok := spec.Values[0].(*ast.Ident)
		if !ok || info.Uses[use] != binding {
			return inlineImmediateValueUse{}, fmt.Errorf("binding must be the var initializer's complete direct expression")
		}
		declaration, ok := info.Defs[spec.Names[0]].(*types.Var)
		if !ok {
			return inlineImmediateValueUse{}, fmt.Errorf("var initializer destination is unresolved")
		}
		return inlineImmediateValueUse{reference: use, varInitializer: true, declarationTarget: declaration}, nil
	case *ast.IfStmt:
		if statement.Init != nil {
			return inlineImmediateValueUse{}, fmt.Errorf("binding cannot inline into an if condition with an init statement")
		}
		use, ok := statement.Cond.(*ast.Ident)
		if !ok || info.Uses[use] != binding {
			return inlineImmediateValueUse{}, fmt.Errorf("binding requires an immediate direct return, one-argument direct call, one-target assignment, single-value var initializer, or complete if condition")
		}
		return inlineImmediateValueUse{reference: use, ifCondition: true}, nil
	case *ast.SwitchStmt:
		return inlineImmediateValueUseSwitchTag(statement, info, binding)
	default:
		return inlineImmediateValueUse{}, fmt.Errorf("binding requires an immediate direct return, one-argument direct call or defer call or go call, one-target assignment, single-value var initializer, complete if condition, or complete expression switch tag")
	}
}

func inlineImmediateValueUseAssignment(statement *ast.AssignStmt, info *types.Info, binding types.Object) (inlineImmediateValueUse, error) {
	if (statement.Tok != token.ASSIGN && statement.Tok != token.DEFINE) || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
		return inlineImmediateValueUse{}, fmt.Errorf("binding requires an immediate simple assignment or short declaration with one target and one value")
	}
	target, ok := statement.Lhs[0].(*ast.Ident)
	if !ok {
		return inlineImmediateValueUse{}, fmt.Errorf("assignment target must be a simple identifier")
	}
	var targetObject types.Object
	if statement.Tok == token.DEFINE {
		targetObject = info.Defs[target]
	} else {
		targetObject = info.Uses[target]
	}
	variable, ok := targetObject.(*types.Var)
	if !ok || variable == binding {
		return inlineImmediateValueUse{}, fmt.Errorf("assignment target must resolve to a distinct variable")
	}
	use, ok := statement.Rhs[0].(*ast.Ident)
	if !ok || info.Uses[use] != binding {
		return inlineImmediateValueUse{}, fmt.Errorf("binding must be the assignment's complete right-hand value")
	}
	return inlineImmediateValueUse{reference: use, assignment: true, shortDeclaration: statement.Tok == token.DEFINE, assignmentTarget: variable}, nil
}

func singleDirectReturnUse(statement *ast.ReturnStmt) (*ast.Ident, bool) {
	if len(statement.Results) != 1 {
		return nil, false
	}
	use, ok := statement.Results[0].(*ast.Ident)
	return use, ok
}

func inlineImmediateValueExpression(statement ast.Stmt, use inlineImmediateValueUse) (ast.Expr, error) {
	if use.ifCondition {
		ifStatement, ok := statement.(*ast.IfStmt)
		if !ok || ifStatement.Init != nil {
			return nil, fmt.Errorf("candidate immediate use is not a complete if condition without an init statement")
		}
		return ifStatement.Cond, nil
	}
	if use.switchTag {
		return inlineImmediateSwitchTagExpression(statement)
	}
	if use.varInitializer {
		spec, ok := inlineSingleVarInitializer(statement)
		if !ok {
			return nil, fmt.Errorf("candidate immediate use is not a single-value var initializer")
		}
		return spec.Values[0], nil
	}
	if use.callArgument {
		call, ok := inlineImmediateValueCall(statement)
		if !ok || call.Ellipsis.IsValid() || len(call.Args) != 1 {
			return nil, fmt.Errorf("candidate immediate use is not a one-argument direct call or defer call")
		}
		if _, directFunction := call.Fun.(*ast.Ident); !directFunction {
			return nil, fmt.Errorf("candidate call target is not a direct function name")
		}
		return call.Args[0], nil
	}
	if use.assignment {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			return nil, fmt.Errorf("candidate immediate use is not a one-target assignment")
		}
		if use.shortDeclaration && assignment.Tok != token.DEFINE || !use.shortDeclaration && assignment.Tok != token.ASSIGN {
			return nil, fmt.Errorf("candidate immediate use changed assignment kind")
		}
		if _, simpleTarget := assignment.Lhs[0].(*ast.Ident); !simpleTarget {
			return nil, fmt.Errorf("candidate assignment target is not a simple identifier")
		}
		return assignment.Rhs[0], nil
	}
	returnStatement, ok := statement.(*ast.ReturnStmt)
	if !ok || len(returnStatement.Results) != 1 {
		return nil, fmt.Errorf("candidate immediate use is not a one-expression return")
	}
	return returnStatement.Results[0], nil
}

func validateInlineStructLiteral(target inlineTarget, info *types.Info) error {
	literal, ok := unparenExpr(target.value).(*ast.CompositeLit)
	if !ok {
		return fmt.Errorf("struct initializer must be a composite literal")
	}
	typeIdentifier, ok := literal.Type.(*ast.Ident)
	if !ok {
		return fmt.Errorf("struct initializer requires a same-package nongeneric named struct")
	}
	typeName, ok := info.Uses[typeIdentifier].(*types.TypeName)
	if !ok {
		return fmt.Errorf("struct initializer type is unresolved")
	}
	named, ok := info.TypeOf(literal).(*types.Named)
	if !ok || named.TypeParams().Len() != 0 || named.TypeArgs().Len() != 0 || named.Obj() != typeName {
		return fmt.Errorf("struct initializer requires a same-package nongeneric named struct")
	}
	assignment := target.stmt.(*ast.AssignStmt)
	binding := info.Defs[assignment.Lhs[0].(*ast.Ident)]
	if typeName.Pkg() == nil || binding == nil || typeName.Pkg() != binding.Pkg() {
		return fmt.Errorf("struct initializer requires a same-package nongeneric named struct")
	}
	structure, ok := named.Underlying().(*types.Struct)
	if !ok {
		return fmt.Errorf("struct initializer requires a named struct")
	}
	fields := make(map[*types.Var]bool, structure.NumFields())
	for field := range structure.Fields() {
		basic, ok := field.Type().(*types.Basic)
		if field.Embedded() || !ok || basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) == 0 || basic.Info()&types.IsUntyped != 0 {
			return fmt.Errorf("every struct field must have a scalar basic type")
		}
		fields[field] = false
	}
	for _, element := range literal.Elts {
		keyed, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return fmt.Errorf("struct initializer must use keyed fields")
		}
		key, ok := keyed.Key.(*ast.Ident)
		if !ok {
			return fmt.Errorf("struct initializer field is unresolved")
		}
		field, ok := info.Uses[key].(*types.Var)
		seen, present := fields[field]
		if !ok || !present || seen {
			return fmt.Errorf("struct initializer field is unresolved or duplicated")
		}
		fields[field] = true
		switch value := keyed.Value.(type) {
		case *ast.BasicLit:
		case *ast.Ident:
			if (value.Name != "true" && value.Name != "false") || info.Uses[value] != types.Universe.Lookup(value.Name) {
				return fmt.Errorf("struct field values must be basic literals or predeclared true/false")
			}
		default:
			return fmt.Errorf("struct field values must be basic literals or predeclared true/false")
		}
		if value, ok := info.Types[keyed.Value]; !ok || value.Value == nil {
			return fmt.Errorf("struct field literal must resolve to a constant")
		}
	}
	return nil
}

func rejectInlineRegionComments(content []byte, fset *token.FileSet, file *ast.File, declaration, returnStatement ast.Stmt) error {
	if _, _, err := standaloneStatementRange(content, fset, declaration, file.Comments); err != nil {
		return err
	}
	if _, _, err := standaloneStatementRange(content, fset, returnStatement, file.Comments); err != nil {
		return err
	}
	for _, group := range file.Comments {
		if group.Pos() <= returnStatement.End() && group.End() >= declaration.Pos() {
			return fmt.Errorf("inline region contains comments or directives")
		}
	}
	return nil
}

func inlineSingleVarInitializer(statement ast.Stmt) (*ast.ValueSpec, bool) {
	declaration, ok := statement.(*ast.DeclStmt)
	if !ok {
		return nil, false
	}
	general, ok := declaration.Decl.(*ast.GenDecl)
	if !ok || general.Tok != token.VAR || general.Lparen.IsValid() || len(general.Specs) != 1 {
		return nil, false
	}
	spec, ok := general.Specs[0].(*ast.ValueSpec)
	if !ok || len(spec.Names) != 1 || len(spec.Values) != 1 {
		return nil, false
	}
	return spec, true
}

func inlineImmediateValueUseSwitchTag(statement *ast.SwitchStmt, info *types.Info, binding types.Object) (inlineImmediateValueUse, error) {
	if statement.Init != nil {
		return inlineImmediateValueUse{}, fmt.Errorf("binding cannot inline into an expression switch with an init statement")
	}
	use, ok := statement.Tag.(*ast.Ident)
	if !ok || info.Uses[use] != binding {
		return inlineImmediateValueUse{}, fmt.Errorf("binding requires an immediate direct return, one-argument direct call, one-target assignment, single-value var initializer, complete if condition, or complete expression switch tag")
	}
	return inlineImmediateValueUse{reference: use, switchTag: true}, nil
}

func inlineImmediateSwitchTagExpression(statement ast.Stmt) (ast.Expr, error) {
	switchStatement, ok := statement.(*ast.SwitchStmt)
	if !ok || switchStatement.Init != nil {
		return nil, fmt.Errorf("candidate immediate use is not a complete expression switch tag without an init statement")
	}
	return switchStatement.Tag, nil
}

func inlineImmediateValueCallUse(call *ast.CallExpr, info *types.Info, binding types.Object, arityRefusal string) (inlineImmediateValueUse, error) {
	if call == nil || call.Ellipsis.IsValid() || len(call.Args) != 1 {
		return inlineImmediateValueUse{}, fmt.Errorf("%s", arityRefusal)
	}
	function, directFunction := call.Fun.(*ast.Ident)
	if !directFunction {
		return inlineImmediateValueUse{}, fmt.Errorf("inline call target must be a direct function name")
	}
	callTarget := info.Uses[function]
	switch callTarget.(type) {
	case *types.Func, *types.Builtin:
	default:
		return inlineImmediateValueUse{}, fmt.Errorf("inline call target must resolve to a declared function or builtin")
	}
	use, ok := call.Args[0].(*ast.Ident)
	if !ok || info.Uses[use] != binding {
		return inlineImmediateValueUse{}, fmt.Errorf("binding must be the call's sole direct argument")
	}
	return inlineImmediateValueUse{reference: use, callArgument: true, callTarget: callTarget}, nil
}

func inlineImmediateValueCall(statement ast.Stmt) (*ast.CallExpr, bool) {
	switch statement := statement.(type) {
	case *ast.ExprStmt:
		call, ok := statement.X.(*ast.CallExpr)
		return call, ok
	case *ast.DeferStmt:
		return statement.Call, true
	case *ast.GoStmt:
		return statement.Call, true
	default:
		return nil, false
	}
}
