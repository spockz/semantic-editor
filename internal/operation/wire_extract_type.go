// This file exposes one conservative same-package Go field extraction through typed planning, overlay validation, receipts, and buffered batches.

package operation

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"semedit/internal/astedit"
	"semedit/internal/sourcebuf"

	"golang.org/x/tools/go/packages"
)

const extractTypeToolName = "semantic_extract_type"

type sourceEdit struct {
	start int
	end   int
	text  []byte
}

func extractTypeStructuralDef() Def[map[string]any, structuralPlanResult] {
	member := MemberTargetSchema()
	minimum := 1
	params := []ParameterContract{
		structuralStringParam("file", "Target: existing Go source file containing the private struct and selected fields; copy from a live source continuation.", true, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].file"),
		structuralStringParam("symbol", "Target: caller-chosen fresh private simple name for the extracted struct type.", true, false, RoleTarget),
		structuralStringParam("in_type", "Selector: existing private, non-generic struct declared in file; copy from the matching inspection continuation.", true, true, RoleSelector,
			"semantic_inspect_symbol.result.continuations[].in_type"),
		{Name: "members", JSONName: "members", CLIName: "members", Type: ParamArray, Description: "Selector: non-empty ordered array of exact {file, symbol} identities such as {\"file\":\"cache.go\",\"symbol\":\"cache.entries\"}; each selects one direct private field on in_type in file. New names are supplied separately.", Required: true, Role: RoleSelector,
			Schema: &ValueSchema{Kind: SchemaArray, Items: &member, MinItems: &minimum}},
		structuralStringParam("field", "Replacement: caller-chosen fresh private simple field name for the new composed value on in_type.", true, false, RoleValue),
		structuralStringParam("expected_revision", "Precondition: optional opaque full-file revision from the same live inspection continuation as file and in_type; stale values are refused.", false, true, RoleTarget,
			"semantic_inspect_symbol.result.continuations[].expected_revision"),
	}
	output := structuralOutputContract(false)
	properties := SymbolTargetSchema().Properties
	output.Continuations = append(output.Continuations,
		OutputContinuation{Path: "result.edit_receipt.continuations[].file", Schema: properties["file"], Role: RoleTarget, Consumers: []string{"semantic_inspect_symbol"}},
		OutputContinuation{Path: "result.edit_receipt.continuations[].symbol", Schema: properties["symbol"], Role: RoleTarget, Consumers: []string{"semantic_inspect_symbol"}},
	)
	return structuralPlanDef(structuralPlanSpec{
		Key: "extract_type", CLIName: "extract-type", MCPName: extractTypeToolName,
		Summary: "Use this tool to move selected direct private fields from one private same-package Go struct into a new composed private struct. Select exact fields with typed {file, symbol} members; choose fresh symbol and field names. The source and extracted type stay in the same file. It rewrites only typed direct field selectors in that file. Generated files, keyed or address-taken uses, cross-file or external consumers, selected tags or comments, reflection or unsafe imports, embedded or grouped fields, generic or exported owners, and build variants are refused. Methods stay on the original type, so this operation does not change interface implementations or method sets. The complete overlay is typechecked before one buffered edit; static inputs are required in batches.",
		Params:  params, Output: &output,
		ExampleRaw: map[string]any{"file": "cache.go", "symbol": "cacheState", "in_type": "cache", "members": []any{map[string]any{"file": "cache.go", "symbol": "cache.entries"}}, "field": "state"},
		Plan:       planExtractType,
		Result: func(_ CallContext, raw map[string]any, _ astedit.SourcePlan, _ string) FileEditRes {
			return FileEditRes{Symbol: stringParam(raw, "symbol"), InType: stringParam(raw, "in_type"), Detail: "Extracted selected fields into a composed Go struct type."}
		},
		Continuations:   extractTypeContinuations,
		ResolveWithRoot: resolveExtractTypeTarget,
	})
}

func planExtractType(snapshot *structuralSnapshot, raw map[string]any) (astedit.SourcePlan, error) {
	inType, newType, fieldName := stringParam(raw, "in_type"), stringParam(raw, "symbol"), stringParam(raw, "field")
	if err := validateFreshPrivateIdentifier("symbol", newType); err != nil {
		return astedit.SourcePlan{}, err
	}
	if err := validateFreshPrivateIdentifier("field", fieldName); err != nil {
		return astedit.SourcePlan{}, err
	}
	owner, named, typeName, err := structuralFieldOwner(snapshot, inType)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	if err := rejectGeneratedExtractFile(snapshot, owner); err != nil {
		return astedit.SourcePlan{}, err
	}
	if newType == typeName || fieldName == typeName || fieldName == newType {
		return astedit.SourcePlan{}, fmt.Errorf("new type and composition field names must be distinct from each other and in_type")
	}
	if owner.Types.Scope().Lookup(newType) != nil {
		return astedit.SourcePlan{}, fmt.Errorf("new type name %q already exists in package %s", newType, owner.PkgPath)
	}
	if owner.Types.Scope().Lookup(fieldName) != nil {
		return astedit.SourcePlan{}, fmt.Errorf("composition field name %q collides with a package declaration", fieldName)
	}
	structure := named.Underlying().(*types.Struct)
	if err := rejectExtractTypeAPIExposure(snapshot, owner, named); err != nil {
		return astedit.SourcePlan{}, err
	}
	if err := structuralFieldSourceSet(snapshot, owner, named, typeName, fieldName); err != nil {
		return astedit.SourcePlan{}, err
	}
	if err := rejectStructuralBuildVariants(snapshot, owner); err != nil {
		return astedit.SourcePlan{}, err
	}
	selected, err := parseExtractTypeMembers(raw["members"], snapshot, owner, structure, typeName)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	if err := proveExtractTypeUses(snapshot, owner, selected); err != nil {
		return astedit.SourcePlan{}, err
	}
	source := snapshot.Sources[snapshot.File]
	fset, file, err := parsePlanExtractSource(snapshot.File, source)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	ownerGroup, err := findExtractTypeOwner(file, typeName)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	edits := make([]sourceEdit, 0, len(selected)+len(selected))
	fieldLines := make([]string, 0, len(selected))
	firstField := true
	newline := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		newline = "\r\n"
	}
	for _, field := range selected {
		start, end, line, err := extractTypeFieldLine(source, fset, field)
		if err != nil {
			return astedit.SourcePlan{}, err
		}
		fieldLines = append(fieldLines, line)
		if firstField {
			firstField = false
			indent := source[start:fset.Position(field.decl.Pos()).Offset]
			edits = append(edits, sourceEdit{start: start, end: end, text: []byte(string(indent) + fieldName + " " + newType + newline)})
		} else {
			edits = append(edits, sourceEdit{start: start, end: end})
		}
	}
	for _, pkg := range snapshot.Packages {
		if pkg.ID != owner.ID {
			continue
		}
		seenUse := map[int]bool{}
		for _, syntax := range pkg.Syntax {
			path, err := filepath.Abs(pkg.Fset.Position(syntax.Pos()).Filename)
			if err != nil {
				return astedit.SourcePlan{}, fmt.Errorf("resolve extraction package source: %w", err)
			}
			if path != snapshot.File {
				continue
			}
			ast.Inspect(syntax, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				selection := pkg.TypesInfo.Uses[selector.Sel]
				if isSelectedExtractField(selected, pkg.Fset, selection) {
					offset := pkg.Fset.Position(selector.X.End()).Offset
					if !seenUse[offset] {
						edits = append(edits, sourceEdit{start: offset, end: offset, text: []byte("." + fieldName)})
						seenUse[offset] = true
					}
				}
				return true
			})
		}
	}
	var fields strings.Builder
	for _, line := range fieldLines {
		fields.WriteString("\t")
		fields.WriteString(strings.TrimSpace(line))
		fields.WriteString(newline)
	}
	insertAt := fset.Position(ownerGroup.End()).Offset
	typeText := []byte(newline + newline + "type " + newType + " struct {" + newline + fields.String() + "}")
	edits = append(edits, sourceEdit{start: insertAt, end: insertAt, text: typeText})
	updated, err := applyExtractTypeEdits(source, edits)
	if err != nil {
		return astedit.SourcePlan{}, err
	}
	return astedit.SourcePlan{Source: updated, Selection: astedit.PlanSelection{Kind: "type", Owner: typeName, Target: newType}, RequiredProofs: []astedit.ProofRequirement{astedit.ProofCompleteSourceSet, astedit.ProofTypedMemberSet, astedit.ProofUnkeyedLiteralSet}, Changed: !bytes.Equal(source, updated)}, nil
}

type extractTypeField struct {
	name   string
	object *types.Var
	ident  *ast.Ident
	decl   *ast.Field
}

func parseExtractTypeMembers(raw any, snapshot *structuralSnapshot, owner *packages.Package, structure *types.Struct, typeName string) ([]extractTypeField, error) {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("members must be a non-empty typed array")
	}
	selectedPath, err := filepath.Rel(snapshot.WorkspaceRoot, snapshot.File)
	if err != nil {
		return nil, fmt.Errorf("derive selected member file path for %q: %w", snapshot.File, err)
	}
	if !filepath.IsLocal(selectedPath) {
		return nil, fmt.Errorf("selected member file %q is outside workspace root %q", snapshot.File, snapshot.WorkspaceRoot)
	}
	selected := make([]extractTypeField, 0, len(items))
	seen := map[*types.Var]bool{}
	for index, item := range items {
		target, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("members[%d] must be an object", index)
		}
		file, fileOK := target["file"].(string)
		qualified, symbolOK := target["symbol"].(string)
		if !fileOK || !symbolOK {
			return nil, fmt.Errorf("members[%d] requires string file and symbol fields", index)
		}
		if file != filepath.ToSlash(selectedPath) {
			return nil, fmt.Errorf("members[%d].file must identify selected file %q", index, filepath.ToSlash(selectedPath))
		}
		name, err := structuralMemberName(typeName, qualified)
		if err != nil {
			return nil, fmt.Errorf("members[%d].symbol: %w", index, err)
		}
		field, indexInStruct := findStructField(structure, name)
		if field == nil || indexInStruct < 0 {
			return nil, fmt.Errorf("members[%d].symbol %q does not name a direct field", index, qualified)
		}
		if field.Exported() || field.Type() == nil || seen[field] {
			return nil, fmt.Errorf("members[%d] must select a distinct direct private named field", index)
		}
		ident, err := exactExtractFieldIdent(snapshot, owner, typeName, name)
		if err != nil {
			return nil, err
		}
		decl, err := enclosingExtractField(owner, ident)
		if err != nil {
			return nil, err
		}
		if decl.Tag != nil || decl.Doc != nil || decl.Comment != nil || len(decl.Names) != 1 || decl.Names[0] != ident {
			return nil, fmt.Errorf("field %q must be a single untagged declaration without comments", name)
		}
		seen[field] = true
		selected = append(selected, extractTypeField{name: name, object: field, ident: ident, decl: decl})
	}
	return selected, nil
}

func findStructField(structure *types.Struct, name string) (*types.Var, int) {
	for index := 0; index < structure.NumFields(); index++ {
		field := structure.Field(index)
		if field.Name() == name {
			return field, index
		}
	}
	return nil, -1
}

func exactExtractFieldIdent(snapshot *structuralSnapshot, owner *packages.Package, typeName, name string) (*ast.Ident, error) {
	for _, syntax := range owner.Syntax {
		path, err := filepath.Abs(owner.Fset.Position(syntax.Pos()).Filename)
		if err != nil {
			return nil, fmt.Errorf("resolve selected owner source path: %w", err)
		}
		if path != snapshot.File {
			continue
		}
		for _, decl := range syntax.Decls {
			group, ok := decl.(*ast.GenDecl)
			if !ok || group.Tok != token.TYPE {
				continue
			}
			for _, spec := range group.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typeName {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return nil, fmt.Errorf("in_type %q is not directly declared as a struct", typeName)
				}
				for _, field := range structure.Fields.List {
					for _, ident := range field.Names {
						if ident.Name == name {
							return ident, nil
						}
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("field %s.%s was not found in the selected syntax", typeName, name)
}

func enclosingExtractField(owner *packages.Package, ident *ast.Ident) (*ast.Field, error) {
	for _, syntax := range owner.Syntax {
		var result *ast.Field
		ast.Inspect(syntax, func(node ast.Node) bool {
			field, ok := node.(*ast.Field)
			if !ok {
				return true
			}
			for _, name := range field.Names {
				if name == ident {
					result = field
					return false
				}
			}
			return result == nil
		})
		if result != nil {
			return result, nil
		}
	}
	return nil, fmt.Errorf("selected field declaration is unavailable")
}

func proveExtractTypeUses(snapshot *structuralSnapshot, owner *packages.Package, fields []extractTypeField) error {
	positions := map[string]bool{}
	for _, selected := range fields {
		positions[fmt.Sprintf("%s:%d", snapshot.File, owner.Fset.Position(selected.ident.Pos()).Offset)] = true
	}
	for _, pkg := range snapshot.Packages {
		if pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		selectors := map[*ast.Ident]bool{}
		addressTaken := map[*ast.Ident]bool{}
		for _, syntax := range pkg.Syntax {
			var stack []ast.Node
			ast.Inspect(syntax, func(node ast.Node) bool {
				if node == nil {
					stack = stack[:len(stack)-1]
					return false
				}
				if selector, ok := node.(*ast.SelectorExpr); ok {
					selectors[selector.Sel] = true
					if len(stack) > 0 {
						if unary, ok := stack[len(stack)-1].(*ast.UnaryExpr); ok && unary.Op == token.AND && unary.X == selector {
							addressTaken[selector.Sel] = true
						}
					}
				}
				stack = append(stack, node)
				return true
			})
		}
		for ident, object := range pkg.TypesInfo.Uses {
			field, ok := object.(*types.Var)
			if !ok || !field.IsField() {
				continue
			}
			declaration := pkg.Fset.Position(field.Pos())
			declarationPath, err := filepath.Abs(declaration.Filename)
			if err != nil {
				return fmt.Errorf("resolve selected field declaration path: %w", err)
			}
			if !positions[fmt.Sprintf("%s:%d", declarationPath, declaration.Offset)] {
				continue
			}
			if !selectors[ident] {
				return fmt.Errorf("selected field %q has a non-selector use that cannot be rewritten safely", field.Name())
			}
			if addressTaken[ident] {
				return fmt.Errorf("selected field %q is address-taken, which is outside this layout proof", field.Name())
			}
			usePath, err := filepath.Abs(pkg.Fset.Position(ident.Pos()).Filename)
			if err != nil {
				return fmt.Errorf("resolve selected field use path: %w", err)
			}
			if usePath != snapshot.File {
				return fmt.Errorf("selected field %q is referenced outside the selected file", field.Name())
			}
		}
	}
	return nil
}

func isSelectedExtractField(selected []extractTypeField, fset *token.FileSet, object types.Object) bool {
	field, ok := object.(*types.Var)
	if !ok || !field.IsField() {
		return false
	}
	for _, candidate := range selected {
		if field.Name() == candidate.name && fset.Position(field.Pos()).Offset == fset.Position(candidate.object.Pos()).Offset {
			return true
		}
	}
	return false
}

func rejectStructuralBuildVariants(snapshot *structuralSnapshot, owner *packages.Package) error {
	for _, filename := range owner.IgnoredFiles {
		absolute, err := filepath.Abs(filename)
		if err != nil {
			return fmt.Errorf("resolve ignored Go build variant %q: %w", filename, err)
		}
		if filepath.Dir(absolute) == filepath.Dir(snapshot.File) && filepath.Ext(absolute) == ".go" {
			return fmt.Errorf("package build variants in %q are outside this extraction's proof scope", filepath.Dir(snapshot.File))
		}
	}
	return nil
}

func rejectGeneratedExtractFile(snapshot *structuralSnapshot, owner *packages.Package) error {
	for _, syntax := range owner.Syntax {
		path, err := filepath.Abs(owner.Fset.Position(syntax.Pos()).Filename)
		if err != nil {
			return fmt.Errorf("resolve extraction source path: %w", err)
		}
		if path != snapshot.File {
			continue
		}
		for _, group := range syntax.Comments {
			for _, comment := range group.List {
				if strings.Contains(comment.Text, "Code generated") && strings.Contains(comment.Text, "DO NOT EDIT") {
					return fmt.Errorf("generated Go file %q is outside this extraction's proof scope", snapshot.File)
				}
			}
		}
		return nil
	}
	return fmt.Errorf("selected extraction source %q is absent from its typed package", snapshot.File)
}

func rejectExtractTypeAPIExposure(snapshot *structuralSnapshot, owner *packages.Package, target *types.Named) error {
	for _, pkg := range snapshot.Packages {
		if pkg.PkgPath != owner.PkgPath && !strings.HasPrefix(pkg.PkgPath, owner.PkgPath+" [") {
			continue
		}
		for path := range pkg.Imports {
			if path == "reflect" || path == "unsafe" {
				return fmt.Errorf("package %s imports %s, which is outside this extraction's layout proof", pkg.PkgPath, path)
			}
		}
	}
	for _, name := range owner.Types.Scope().Names() {
		object := owner.Types.Scope().Lookup(name)
		if object != nil && object.Exported() && extractTypeContains(object.Type(), target, map[types.Type]bool{}) {
			return fmt.Errorf("private owner %q escapes through exported package declaration %q", target.Obj().Name(), name)
		}
	}
	for _, name := range owner.Types.Scope().Names() {
		object := owner.Types.Scope().Lookup(name)
		typeName, ok := object.(*types.TypeName)
		if !ok || !typeName.Exported() {
			continue
		}
		named, ok := types.Unalias(typeName.Type()).(*types.Named)
		if !ok {
			continue
		}
		for method := range types.NewMethodSet(named).Methods() {
			if method.Obj().Exported() && extractTypeContains(method.Obj().Type(), target, map[types.Type]bool{}) {
				return fmt.Errorf("private owner %q escapes through exported method %q", target.Obj().Name(), method.Obj().Name())
			}
		}
	}
	for _, syntax := range owner.Syntax {
		for _, declaration := range syntax.Decls {
			if !exportedExtractDeclaration(declaration) {
				continue
			}
			if extractExportedDeclarationUsesOwner(owner, declaration, target) {
				return fmt.Errorf("private owner %q is referenced by an exported declaration", target.Obj().Name())
			}
		}
	}
	return nil
}

func extractExportedDeclarationUsesOwner(owner *packages.Package, declaration ast.Decl, target *types.Named) bool {
	if function, ok := declaration.(*ast.FuncDecl); ok {
		object := owner.TypesInfo.Defs[function.Name]
		if object == nil || object.Type() == nil {
			return true
		}
		if signature, ok := object.Type().(*types.Signature); ok {
			if signatureReturnsInterface(signature) && packageUsesExtractType(owner, target.Obj()) {
				return true
			}
			if extractTypeContains(signature.Params(), target, map[types.Type]bool{}) || extractTypeContains(signature.Results(), target, map[types.Type]bool{}) {
				return true
			}
		}
		return astDeclarationUsesObject(function.Type, owner.TypesInfo, target.Obj()) || astDeclarationUsesObject(function.Body, owner.TypesInfo, target.Obj())
	}
	return astDeclarationUsesObject(declaration, owner.TypesInfo, target.Obj())
}

func signatureReturnsInterface(signature *types.Signature) bool {
	for index := 0; index < signature.Results().Len(); index++ {
		if _, ok := types.Unalias(signature.Results().At(index).Type()).Underlying().(*types.Interface); ok {
			return true
		}
	}
	return false
}

func packageUsesExtractType(owner *packages.Package, target types.Object) bool {
	for _, syntax := range owner.Syntax {
		used := false
		ast.Inspect(syntax, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if ok && owner.TypesInfo.Uses[ident] == target {
				used = true
				return false
			}
			return !used
		})
		if used {
			return true
		}
	}
	return false
}

func astDeclarationUsesObject(node ast.Node, info *types.Info, object types.Object) bool {
	found := false
	ast.Inspect(node, func(candidate ast.Node) bool {
		ident, ok := candidate.(*ast.Ident)
		if ok && info.Uses[ident] == object {
			found = true
			return false
		}
		return !found
	})
	return found
}

func exportedExtractDeclaration(declaration ast.Decl) bool {
	switch node := declaration.(type) {
	case *ast.FuncDecl:
		return node.Name != nil && node.Name.IsExported()
	case *ast.GenDecl:
		for _, raw := range node.Specs {
			switch spec := raw.(type) {
			case *ast.TypeSpec:
				if spec.Name.IsExported() {
					return true
				}
			case *ast.ValueSpec:
				for _, name := range spec.Names {
					if name.IsExported() {
						return true
					}
				}
			}
		}
	}
	return false
}

func extractTypeContains(candidate types.Type, target *types.Named, seen map[types.Type]bool) bool {
	if candidate == nil || seen[candidate] {
		return false
	}
	seen[candidate] = true
	switch value := candidate.(type) {
	case *types.Named:
		if value.Obj() == target.Obj() {
			return true
		}
		for index := 0; index < value.TypeArgs().Len(); index++ {
			if extractTypeContains(value.TypeArgs().At(index), target, seen) {
				return true
			}
		}
		return extractTypeContains(value.Underlying(), target, seen)
	case *types.Alias:
		return extractTypeContains(types.Unalias(value), target, seen)
	case *types.Pointer:
		return extractTypeContains(value.Elem(), target, seen)
	case *types.Slice:
		return extractTypeContains(value.Elem(), target, seen)
	case *types.Array:
		return extractTypeContains(value.Elem(), target, seen)
	case *types.Map:
		return extractTypeContains(value.Key(), target, seen) || extractTypeContains(value.Elem(), target, seen)
	case *types.Chan:
		return extractTypeContains(value.Elem(), target, seen)
	case *types.Signature:
		return extractTypeContains(value.Params(), target, seen) || extractTypeContains(value.Results(), target, seen)
	case *types.Tuple:
		for index := 0; index < value.Len(); index++ {
			if extractTypeContains(value.At(index).Type(), target, seen) {
				return true
			}
		}
	case *types.Struct:
		for index := 0; index < value.NumFields(); index++ {
			if extractTypeContains(value.Field(index).Type(), target, seen) {
				return true
			}
		}
	case *types.Interface:
		value.Complete()
		for index := 0; index < value.NumMethods(); index++ {
			if extractTypeContains(value.Method(index).Type(), target, seen) {
				return true
			}
		}
		for index := 0; index < value.NumEmbeddeds(); index++ {
			if extractTypeContains(value.EmbeddedType(index), target, seen) {
				return true
			}
		}
	case *types.TypeParam:
		return extractTypeContains(value.Constraint(), target, seen)
	case *types.Union:
		for index := 0; index < value.Len(); index++ {
			if extractTypeContains(value.Term(index).Type(), target, seen) {
				return true
			}
		}
	}
	return false
}

func parsePlanExtractSource(path string, source []byte) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, nil, fmt.Errorf("parse selected extraction source %q: %w", path, err)
	}
	return fset, file, nil
}

func findExtractTypeOwner(file *ast.File, name string) (*ast.GenDecl, error) {
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE || len(group.Specs) != 1 {
			continue
		}
		spec, ok := group.Specs[0].(*ast.TypeSpec)
		if ok && spec.Name.Name == name && !spec.Assign.IsValid() && spec.TypeParams == nil {
			structure, ok := spec.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("owner %q must be a directly declared struct", name)
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 0 {
					return nil, fmt.Errorf("owner %q has embedded fields outside this extraction's proof scope", name)
				}
			}
			return group, nil
		}
	}
	return nil, fmt.Errorf("owner %q must be a standalone type declaration", name)
}

func extractTypeFieldLine(source []byte, fset *token.FileSet, field extractTypeField) (int, int, string, error) {
	start := fset.Position(field.decl.Pos()).Offset
	end := fset.Position(field.decl.End()).Offset
	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	lineEnd := bytes.IndexByte(source[end:], '\n')
	if lineEnd < 0 {
		lineEnd = len(source)
	} else {
		lineEnd += end + 1
	}
	if len(bytes.TrimSpace(source[lineStart:start])) != 0 || len(bytes.TrimSpace(source[end:lineEnd])) != 0 {
		return 0, 0, "", fmt.Errorf("field %q must occupy one complete source line", field.name)
	}
	fieldSource := bytes.TrimSpace(source[start:end])
	return lineStart, lineEnd, string(fieldSource), nil
}

func applyExtractTypeEdits(source []byte, edits []sourceEdit) ([]byte, error) {
	sort.Slice(edits, func(left, right int) bool { return edits[left].start > edits[right].start })
	updated := bytes.Clone(source)
	lastStart := len(source) + 1
	for _, edit := range edits {
		if edit.start < 0 || edit.end < edit.start || edit.end > len(source) || edit.end > lastStart {
			return nil, fmt.Errorf("extraction edits overlap or exceed selected source")
		}
		updated = append(updated[:edit.start], append(bytes.Clone(edit.text), updated[edit.end:]...)...)
		lastStart = edit.start
	}
	return updated, nil
}

func validateFreshPrivateIdentifier(name, value string) error {
	if !token.IsIdentifier(value) || token.Lookup(value).IsKeyword() || value == "_" || ast.IsExported(value) {
		return fmt.Errorf("%s must be a fresh private simple Go identifier", name)
	}
	return nil
}

func extractTypeContinuations(cc CallContext, snapshot *structuralSnapshot, raw map[string]any, plan astedit.SourcePlan) ([]SymbolTarget, error) {
	relative, err := filepath.Rel(cc.WorkDir, snapshot.File)
	if err != nil {
		return nil, fmt.Errorf("derive extracted type continuation path %q: %w", snapshot.File, err)
	}
	if !filepath.IsLocal(relative) {
		return nil, fmt.Errorf("extracted type continuation path %q is outside workspace", snapshot.File)
	}
	revision := structuralRevision(plan.Source)
	return []SymbolTarget{{File: filepath.ToSlash(relative), Symbol: stringParam(raw, "symbol"), ExpectedRevision: &revision}}, nil
}

func resolveExtractTypeTarget(ctx context.Context, root string, candidate SymbolTarget) (SymbolTarget, bool, error) {
	path := filepath.Join(root, filepath.FromSlash(candidate.File))
	source, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return SymbolTarget{}, false, fmt.Errorf("read final extracted type target %q: %w", path, err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return SymbolTarget{}, false, fmt.Errorf("parse final extracted type target %q: %w", path, err)
	}
	found := false
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE {
			continue
		}
		for _, rawSpec := range group.Specs {
			spec, ok := rawSpec.(*ast.TypeSpec)
			if ok && spec.Name.Name == candidate.Symbol && !spec.Assign.IsValid() {
				if _, isStruct := spec.Type.(*ast.StructType); !isStruct {
					continue
				}
				found = true
			}
		}
	}
	return candidate, found, nil
}
