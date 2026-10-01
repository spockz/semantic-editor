// This file plans one append to a typed keyed literal after syntax and supplied type-object checks pass.

package astedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
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
	if err := validateRevisionBoundSelector(intent.ConstructPath, intent.ExpectedRevision); err != nil {
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
	owner, err := findCompositeOwner(file, intent.Declaration)
	if err != nil {
		return SourcePlan{}, err
	}
	literal, err := selectCompositeTarget(source, fset, file, owner, intent)
	if err != nil {
		return SourcePlan{}, err
	}
	if err := validateTypedElement(source, fset, literal, keyed, keySource, intent.TypeContext); err != nil {
		return SourcePlan{}, err
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
		switch value := decl.(type) {
		case *ast.FuncDecl:
			if value.Name.Name == name {
				if owner != nil {
					return nil, fmt.Errorf("declaration selector %q is ambiguous", name)
				}
				owner = value
			}
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				matches := false
				switch item := spec.(type) {
				case *ast.TypeSpec:
					matches = item.Name.Name == name
				case *ast.ValueSpec:
					for _, ident := range item.Names {
						matches = matches || ident.Name == name
					}
				}
				if !matches {
					continue
				}
				if owner != nil {
					return nil, fmt.Errorf("declaration selector %q is ambiguous", name)
				}
				owner = value
			}
		}
	}
	if owner == nil {
		return nil, fmt.Errorf("declaration %q not found", name)
	}
	return owner, nil
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
	var selected *ast.CompositeLit
	if !intent.ConstructPath.Present {
		if len(matches) != 1 {
			return nil, fmt.Errorf("literal selector matched %d composites; construct_path is required when ambiguous", len(matches))
		}
		selected = matches[0]
	} else {
		index, err := strconv.Atoi(intent.ConstructPath.Value)
		if err != nil || index < 0 || index >= len(literals) {
			return nil, fmt.Errorf("construct_path %q is not a valid zero-based literal index", intent.ConstructPath.Value)
		}
		selected = literals[index]
		start, end := fset.Position(selected.Pos()).Offset, fset.Position(selected.End()).Offset
		if string(source[start:end]) != intent.Discriminator {
			return nil, fmt.Errorf("construct_path %q does not match discriminator", intent.ConstructPath.Value)
		}
	}
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
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "element.go", []byte("package p\nvar _ = map[any]any{"+source+"}\n"), parser.AllErrors)
	if err != nil {
		return nil, "", err
	}
	gen := file.Decls[0].(*ast.GenDecl)
	value := gen.Specs[0].(*ast.ValueSpec)
	literal := value.Values[0].(*ast.CompositeLit)
	if len(literal.Elts) != 1 {
		return nil, "", fmt.Errorf("expected exactly one literal element")
	}
	keyed, ok := literal.Elts[0].(*ast.KeyValueExpr)
	if !ok {
		return nil, "", fmt.Errorf("literal element must be keyed")
	}
	start := fset.Position(keyed.Key.Pos()).Offset - len("package p\nvar _ = map[any]any{")
	end := fset.Position(keyed.Key.End()).Offset - len("package p\nvar _ = map[any]any{")
	return keyed, source[start:end], nil
}
