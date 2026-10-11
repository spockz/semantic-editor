// This file plans one append to a typed keyed literal after syntax and supplied type-object checks pass.

package astedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type (
	// LiteralFamily distinguishes supported keyed composite literal types.
	LiteralFamily string

	// TypedKeyEvidence binds source key text to its analyzed key identity.
	TypedKeyEvidence struct {
		Expression string
		Identity   string
	}

	// CompositeTypeContext supplies typed facts unavailable from syntax alone.
	CompositeTypeContext struct {
		Family          LiteralFamily
		StructFields    []FieldSpec
		MapKeyType      string
		ExistingMapKeys []TypedKeyEvidence
		NewMapKey       TypedKeyEvidence
	}

	// InsertElementIntent selects one literal and supplies one keyed element.
	InsertElementIntent struct {
		Declaration      string
		Discriminator    string
		ConstructPath    OptionalString
		ExpectedRevision OptionalString
		Source           string
		TypeContext      CompositeTypeContext
	}
)

// Literal families define the type facts required for duplicate checks.
const (
	LiteralStruct LiteralFamily = "struct"
	LiteralMap    LiteralFamily = "map"
)

// PlanInsertCompositeElement plans one keyed append using caller-supplied typed context.
func PlanInsertCompositeElement(source []byte, intent InsertElementIntent) (SourcePlan, error) {
	if intent.Declaration == "" || intent.Discriminator == "" || intent.Source == "" {
		return SourcePlan{}, fmt.Errorf("declaration, discriminator, and source are required")
	}
	if err := validateRevisionBoundSelector(source, intent.ConstructPath, intent.ExpectedRevision); err != nil {
		return SourcePlan{}, err
	}
	if strings.ContainsAny(intent.Source, "\r\n") {
		return SourcePlan{}, fmt.Errorf("inserted element must be one single-line element")
	}
	keyed, keySource, err := parseKeyedElement(intent.Source)
	if err != nil {
		return SourcePlan{}, fmt.Errorf("parse inserted element: %w", err)
	}
	fset, file, err := parsePlanSource(source)
	if err != nil {
		return SourcePlan{}, err
	}
	if intent.ConstructPath.Present {
		return SourcePlan{}, fmt.Errorf("construct_path has no compatible live producer for composite literals")
	}
	owner, err := findCompositeOwner(file, intent.Declaration)
	if err != nil {
		return SourcePlan{}, err
	}
	literal, err := selectCompositeTarget(source, fset, file, owner, intent)
	if err != nil {
		return SourcePlan{}, err
	}
	if validationErr := validateTypedElement(source, fset, literal, keyed, keySource, intent.TypeContext); validationErr != nil {
		return SourcePlan{}, validationErr
	}
	updated, err := appendCompositeElement(source, fset, literal, strings.TrimSpace(intent.Source))
	if err != nil {
		return SourcePlan{}, err
	}
	return newSourcePlan(source, updated, PlanSelection{
		Kind:  "literal",
		Owner: intent.Declaration,
		Start: fset.Position(literal.Pos()).Offset,
		End:   fset.Position(literal.End()).Offset,
	}, ProofCompleteSourceSet, ProofTypedLiteral), nil
}

func findCompositeOwner(file *ast.File, name string) (ast.Node, error) {
	var owner ast.Node
	for _, decl := range file.Decls {
		var candidate ast.Node
		switch value := decl.(type) {
		case *ast.FuncDecl:
			if compositeFunctionMatches(value, name) {
				candidate = value
			}
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				matches, err := compositeSpecMatches(spec, name)
				if err != nil {
					return nil, err
				}
				if !matches {
					continue
				}
				if err := setCompositeOwner(&owner, spec, name); err != nil {
					return nil, err
				}
			}
		}
		if candidate != nil {
			if err := setCompositeOwner(&owner, candidate, name); err != nil {
				return nil, err
			}
		}
	}
	if owner == nil {
		return nil, fmt.Errorf("declaration %q not found", name)
	}
	return owner, nil
}

func compositeSpecMatches(spec ast.Spec, name string) (bool, error) {
	switch item := spec.(type) {
	case *ast.TypeSpec:
		return item.Name.Name == name, nil
	case *ast.ValueSpec:
		matches := false
		for _, ident := range item.Names {
			if ident.Name == name {
				if len(item.Names) > 1 {
					return false, fmt.Errorf("declaration %q shares a ValueSpec; composite ownership is unsupported", name)
				}
				matches = true
			}
		}
		return matches, nil
	default:
		return false, nil
	}
}

func setCompositeOwner(owner *ast.Node, candidate ast.Node, name string) error {
	if *owner != nil {
		return fmt.Errorf("declaration selector %q is ambiguous", name)
	}
	*owner = candidate
	return nil
}

func selectCompositeTarget(source []byte, fset *token.FileSet, file *ast.File, owner ast.Node, intent InsertElementIntent) (*ast.CompositeLit, error) {
	literals := make([]*ast.CompositeLit, 0, 1)
	ast.Inspect(owner, func(node ast.Node) bool {
		if literal, ok := node.(*ast.CompositeLit); ok {
			literals = append(literals, literal)
		}
		return true
	})
	matches := make([]*ast.CompositeLit, 0, 1)
	for _, literal := range literals {
		start, end := fset.Position(literal.Pos()).Offset, fset.Position(literal.End()).Offset
		if start >= 0 && end <= len(source) && string(source[start:end]) == intent.Discriminator {
			matches = append(matches, literal)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("literal selector matched %d composites; no compatible construct_path producer exists", len(matches))
	}
	selected := matches[0]
	for _, comment := range file.Comments {
		if comment.Pos() > selected.Pos() && comment.End() < selected.End() {
			return nil, fmt.Errorf("literal comments make append ownership ambiguous")
		}
	}
	return selected, nil
}

func validateTypedElement(source []byte, fset *token.FileSet, literal *ast.CompositeLit, keyed *ast.KeyValueExpr, keySource string, context CompositeTypeContext) error {
	switch context.Family {
	case LiteralStruct:
		if len(context.StructFields) == 0 {
			return fmt.Errorf("typed struct field set is required")
		}
		name, ok := keyed.Key.(*ast.Ident)
		if !ok {
			return fmt.Errorf("struct element key must be a field identifier")
		}
		known := false
		for _, field := range context.StructFields {
			if field.Name == name.Name && field.Type != "" {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("struct field %q is absent from the supplied type context", name.Name)
		}
		for _, existing := range literal.Elts {
			entry, ok := existing.(*ast.KeyValueExpr)
			if !ok {
				return fmt.Errorf("positional struct literals are unsupported")
			}
			existingName, ok := entry.Key.(*ast.Ident)
			if !ok {
				return fmt.Errorf("struct literal contains an unsupported key expression")
			}
			if existingName.Name == name.Name {
				return fmt.Errorf("struct literal already contains field %q", name.Name)
			}
		}
	case LiteralMap:
		return validateTypedMapKey(source, fset, literal, keySource, context)
	default:
		return fmt.Errorf("unsupported typed literal family %q", context.Family)
	}
	return nil
}

func validateTypedMapKey(source []byte, fset *token.FileSet, literal *ast.CompositeLit, keySource string, context CompositeTypeContext) error {
	if context.MapKeyType == "" || context.NewMapKey.Expression == "" || context.NewMapKey.Identity == "" {
		return fmt.Errorf("typed map key context is required")
	}
	if keySource != context.NewMapKey.Expression {
		return fmt.Errorf("new map key evidence does not match the element source")
	}
	if len(context.ExistingMapKeys) != len(literal.Elts) {
		return fmt.Errorf("typed map key evidence does not cover every existing element")
	}
	identities := make(map[string]struct{}, len(context.ExistingMapKeys)+1)
	for index, evidence := range context.ExistingMapKeys {
		if evidence.Identity == "" {
			return fmt.Errorf("existing map key evidence is incomplete")
		}
		entry, ok := literal.Elts[index].(*ast.KeyValueExpr)
		if !ok {
			return fmt.Errorf("map literal contains a positional element")
		}
		start, end := fset.Position(entry.Key.Pos()).Offset, fset.Position(entry.Key.End()).Offset
		if evidence.Expression != string(source[start:end]) {
			return fmt.Errorf("typed map key evidence does not match existing element %d", index)
		}
		if _, duplicate := identities[evidence.Identity]; duplicate {
			return fmt.Errorf("existing map literal contains duplicate typed keys")
		}
		identities[evidence.Identity] = struct{}{}
	}
	if _, duplicate := identities[context.NewMapKey.Identity]; duplicate {
		return fmt.Errorf("map literal already contains the typed key")
	}
	return nil
}

func appendCompositeElement(source []byte, fset *token.FileSet, literal *ast.CompositeLit, element string) ([]byte, error) {
	closeOffset := fset.Position(literal.Rbrace).Offset
	if closeOffset < 0 || closeOffset > len(source) {
		return nil, fmt.Errorf("literal closing brace is outside source")
	}
	if len(literal.Elts) == 0 {
		return spliceSource(source, closeOffset, closeOffset, []byte(element)), nil
	}
	lastEnd := fset.Position(literal.Elts[len(literal.Elts)-1].End()).Offset
	suffix := source[lastEnd:closeOffset]
	if bytes.Contains(suffix, []byte("\n")) {
		closeIndent := string(suffix[bytes.LastIndexByte(suffix, '\n')+1:])
		newline := "\n"
		if bytes.Contains(source, []byte("\r\n")) {
			newline = "\r\n"
		}
		insertion := []byte("," + newline + closeIndent + "\t" + element + "," + newline + closeIndent)
		return spliceSource(source, lastEnd, closeOffset, insertion), nil
	}
	suffix = bytes.TrimLeft(suffix, " \t")
	insertion := append([]byte(", "+element), suffix...)
	return spliceSource(source, lastEnd, closeOffset, insertion), nil
}

func parseKeyedElement(source string) (*ast.KeyValueExpr, string, error) {
	trimmed := strings.TrimSpace(source)
	if strings.HasSuffix(trimmed, ",") {
		return nil, "", fmt.Errorf("element source must not include a trailing comma")
	}
	fset := token.NewFileSet()
	const prefix = "package p\nvar _ = map[any]any{"
	file, err := parser.ParseFile(fset, "element.go", []byte(prefix+source+"}\n"), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, "", err
	}
	if len(file.Comments) != 0 {
		return nil, "", fmt.Errorf("element source must not include comments")
	}
	if len(file.Decls) != 1 {
		return nil, "", fmt.Errorf("element source must not inject declarations")
	}
	gen, ok := file.Decls[0].(*ast.GenDecl)
	if !ok || len(gen.Specs) != 1 {
		return nil, "", fmt.Errorf("element source must contain exactly one keyed element")
	}
	value, ok := gen.Specs[0].(*ast.ValueSpec)
	if !ok || len(value.Values) != 1 {
		return nil, "", fmt.Errorf("element source must contain exactly one keyed element")
	}
	literal, ok := value.Values[0].(*ast.CompositeLit)
	if !ok {
		return nil, "", fmt.Errorf("element source must be one keyed composite element")
	}
	if len(literal.Elts) != 1 {
		return nil, "", fmt.Errorf("expected exactly one literal element")
	}
	keyed, ok := literal.Elts[0].(*ast.KeyValueExpr)
	if !ok {
		return nil, "", fmt.Errorf("literal element must be keyed")
	}
	start := fset.Position(keyed.Key.Pos()).Offset - len(prefix)
	end := fset.Position(keyed.Key.End()).Offset - len(prefix)
	if start < 0 || end < start || end > len(source) {
		return nil, "", fmt.Errorf("keyed element source positions are outside the selected element")
	}
	return keyed, source[start:end], nil
}

// CompositeElementKeySource returns the key expression from one keyed element.
func CompositeElementKeySource(source string) (string, error) {
	_, key, err := parseKeyedElement(source)
	if err != nil {
		return "", fmt.Errorf("parse keyed composite element: %w", err)
	}
	return key, nil
}

func compositeFunctionMatches(function *ast.FuncDecl, selector string) bool {
	if function.Recv == nil {
		return selector == function.Name.Name
	}
	if len(function.Recv.List) != 1 {
		return false
	}
	receiver := function.Recv.List[0].Type
	pointer := false
	if star, ok := receiver.(*ast.StarExpr); ok {
		pointer = true
		receiver = star.X
	}
	var receiverName string
	switch expression := receiver.(type) {
	case *ast.Ident:
		receiverName = expression.Name
	case *ast.IndexExpr:
		if identifier, ok := expression.X.(*ast.Ident); ok {
			receiverName = identifier.Name
		}
	case *ast.IndexListExpr:
		if identifier, ok := expression.X.(*ast.Ident); ok {
			receiverName = identifier.Name
		}
	}
	if receiverName == "" {
		return false
	}
	target := receiverName + "." + function.Name.Name
	return (selector == target || pointer && selector == "(*"+receiverName+")."+function.Name.Name)
}
